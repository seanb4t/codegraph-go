//go:build tmux

package tmux

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// hashConfigTree returns a hex sha256 digest over every regular file under
// root, implemented in pure Go (filepath.WalkDir + crypto/sha256) rather
// than shelling out to a coreutils checksum binary: that binary does not
// exist on darwin, and this repo ships a darwin binary with contributors
// running this target locally, so a shell-out here would make the test
// pass on Linux and fail on macOS for a reason that has nothing to do with
// the property under test.
//
// The walk covers the WHOLE tree — no per-agent path enumeration — so a
// newly registered agent target's writes are covered automatically with no
// edit here (TTY-05's zero-maintenance requirement). Relative paths are
// collected and sorted before hashing, so the digest is walk-order
// independent rather than accidentally sensitive to filepath.WalkDir's own
// traversal order.
//
// One documented scope limit: codegraph install's interactive picker path
// always uses the default LocationGlobal, which resolves under root when
// root is HOME — the picker never overrides --location. A future test
// exercising LocationLocal must NOT reuse this recipe unmodified, because
// at least one target's local branch (cursorConfigPath) falls back to
// os.Getwd() rather than resolving under root
// [08-RESEARCH.md Pattern 3, internal/agents/cursor.go:84].
func hashConfigTree(t *testing.T, root string) string {
	t.Helper()

	var relPaths []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		relPaths = append(relPaths, rel)
		return nil
	})
	if err != nil {
		t.Fatalf("hashConfigTree: walk %s: %v", root, err)
	}

	sort.Strings(relPaths)

	h := sha256.New()
	for _, rel := range relPaths {
		io.WriteString(h, rel) //nolint:errcheck // hash.Hash.Write never returns an error
		f, openErr := os.Open(filepath.Join(root, rel))
		if openErr != nil {
			t.Fatalf("hashConfigTree: open %s: %v", rel, openErr)
		}
		if _, copyErr := io.Copy(h, f); copyErr != nil {
			f.Close()
			t.Fatalf("hashConfigTree: read %s: %v", rel, copyErr)
		}
		f.Close()
	}

	return hex.EncodeToString(h.Sum(nil))
}
