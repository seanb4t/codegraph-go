package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/seanb4t/codegraph-go/internal/fsatomic"
	"github.com/seanb4t/codegraph-go/internal/userdirs"
)

// Record is the JSON payload of one <state dir>/daemons/<pid>.json entry
// (D-04): the minimum a cross-project daemon picker (07-07) or `daemon stop
// --all` (07-04) needs to identify and act on a running daemon.
type Record struct {
	PID       int       `json:"pid"`
	StartedAt time.Time `json:"startedAt"`
	RepoRoot  string    `json:"repoRoot"`
}

// registryDirName is the registry's subdirectory inside the per-user state
// directory, and inside the legacy ~/.codegraph directory.
const registryDirName = "daemons"

// legacyStateDirName is the per-user directory older releases wrote the
// registry under. It has the same name as the per-repository index marker,
// which is why the registry moved out of it.
const legacyStateDirName = ".codegraph"

// registryDir returns the global directory of per-daemon record files
// (D-04 — deliberately not the per-repo .codegraph/daemon.lock):
// <userdirs.StateDir()>/daemons. It is a package-level func var (mirroring
// the onSync*/onWatchOpen test-seam convention already used in daemon.go)
// so registry_test.go can point it at t.TempDir() instead of the real
// state directory.
var registryDir = defaultRegistryDir

// defaultRegistryDir resolves the registry directory and, on the way, moves
// any records an older release left in ~/.codegraph/daemons into it. The
// migration is a no-op once the new directory exists, so the steady-state
// cost is one stat per call.
func defaultRegistryDir() (string, error) {
	stateDir, err := userdirs.StateDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(stateDir, registryDirName)
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if err := migrateLegacyRegistry(filepath.Join(home, legacyStateDirName, registryDirName), dir); err != nil {
		return "", err
	}
	return dir, nil
}

// migrateLegacyRegistry moves every <pid>.json record from legacy into dir,
// once: it does nothing when dir already exists or legacy does not. Each
// record is read, written atomically to dir, then removed, so a state
// directory on another filesystem works. Afterwards legacy and its parent
// are removed with os.Remove, which fails on a non-empty directory; those
// failures are ignored, so a parent that still holds files codegraph did
// not write (another tool's, or an index built there by mistake) stays as it is.
func migrateLegacyRegistry(legacy, dir string) error {
	if _, err := os.Stat(dir); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	entries, err := os.ReadDir(legacy)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		src := filepath.Join(legacy, e.Name())
		data, err := os.ReadFile(src)
		if err != nil {
			if os.IsNotExist(err) {
				continue // deregistered while we were migrating
			}
			return err
		}
		if err := fsatomic.WriteFile(filepath.Join(dir, e.Name()), string(data)); err != nil {
			return err
		}
		if err := os.Remove(src); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	_ = os.Remove(legacy)
	_ = os.Remove(filepath.Dir(legacy))
	return nil
}

// recordPath returns the record file path for pid within the registry dir.
func recordPath(dir string, pid int) string {
	return filepath.Join(dir, fmt.Sprintf("%d.json", pid))
}

// Register atomically writes this daemon's record to the global registry
// (D-04/D-06), keyed by pid — never merged with any other daemon's record,
// including one for the same RepoRoot. fsatomic.WriteFile creates the
// registry dir if needed and guarantees a mid-write reader never observes a
// partial file.
func Register(rec Record) error {
	dir, err := registryDir()
	if err != nil {
		return err
	}
	data, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	return fsatomic.WriteFile(recordPath(dir, rec.PID), string(data))
}

// Deregister removes pid's record from the global registry (D-06), mirroring
// lock.go's release(): a missing file is a nil (success) no-op, not an
// error.
func Deregister(pid int) error {
	dir, err := registryDir()
	if err != nil {
		return err
	}
	if err := os.Remove(recordPath(dir, pid)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// List reads every record in the global registry and self-heals on this
// SAME call (D-05) — no background reaper: each record's pid is checked
// against lock.go's isStale/isProcessLive (same package, unexported, no
// second liveness implementation), and any record found stale is removed
// from disk and excluded, mirroring acquire()'s existing
// detect-and-clear-on-every-independent-call discipline (Phase 4 D-16),
// generalized from one lockfile to many. An absent or empty registry dir is
// not an error — it returns (nil, nil). A file that vanishes between
// ReadDir and ReadFile (raced by a concurrent Deregister/prune) or a
// malformed record is skipped, not fatal.
func List() ([]Record, error) {
	dir, err := registryDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var live []Record
	for _, e := range entries {
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue // vanished between ReadDir and ReadFile — another scan raced us
		}
		var rec Record
		if err := json.Unmarshal(data, &rec); err != nil {
			continue // malformed record — treat as unreadable, not live
		}
		if isStale(lockInfo{PID: rec.PID, StartedAt: rec.StartedAt}) {
			_ = os.Remove(path) // self-heal; best-effort, matches release()'s style
			continue
		}
		live = append(live, rec)
	}
	return live, nil
}
