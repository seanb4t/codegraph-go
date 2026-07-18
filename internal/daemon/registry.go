package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/seanb4t/codegraph-go/internal/fsatomic"
)

// Record is the JSON payload of one ~/.codegraph/daemons/<pid>.json entry
// (D-04): the minimum a cross-project daemon picker (07-07) or `daemon stop
// --all` (07-04) needs to identify and act on a running daemon.
type Record struct {
	PID       int       `json:"pid"`
	StartedAt time.Time `json:"startedAt"`
	RepoRoot  string    `json:"repoRoot"`
}

// registryDir returns ~/.codegraph/daemons/, the global directory of
// per-daemon record files (D-04 — deliberately not the per-repo
// .codegraph/daemon.lock). It is a package-level func var (mirroring the
// onSync*/onWatchOpen test-seam convention already used in daemon.go) so
// registry_test.go can point it at t.TempDir() instead of the real home
// directory.
var registryDir = defaultRegistryDir

func defaultRegistryDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codegraph", "daemons"), nil
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
