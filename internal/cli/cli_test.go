package cli

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
)

// copyFixture copies internal/indexer/testdata/gofixture into a fresh
// t.TempDir() so commands run against a normal directory with its own
// go.mod, rather than mutating the checked-in testdata tree.
func copyFixture(t *testing.T) string {
	t.Helper()

	src, err := filepath.Abs(filepath.Join("..", "indexer", "testdata", "gofixture"))
	if err != nil {
		t.Fatalf("resolve fixture path: %v", err)
	}
	dst := t.TempDir()

	err = filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copy fixture: %v", err)
	}
	return dst
}

// execCmd runs the real codegraph command tree with the given args and an
// empty stdin, capturing stdout/stderr — the cobra execution-test pattern
// (RESEARCH §Validation Architecture) rather than shelling out to a built
// binary.
func execCmd(args ...string) (stdout, stderr string, err error) {
	return execCmdWithInput("", args...)
}

// execCmdWithInput is execCmd with an explicit stdin, for exercising the
// interactive confirm() prompt in index/uninit without --force.
func execCmdWithInput(input string, args ...string) (stdout, stderr string, err error) {
	root := newRootCmd()
	var outBuf, errBuf bytes.Buffer
	root.SetOut(&outBuf)
	root.SetErr(&errBuf)
	root.SetIn(strings.NewReader(input))
	root.SetArgs(args)
	err = root.Execute()
	return outBuf.String(), errBuf.String(), err
}

// readGraphCounts opens the store under a .codegraph/ directory and reads
// back the committed Meta record's node/edge counts, mirroring
// indexer.readGraphCounts — used to assert a failed re-init or repeated
// rebuild leaves the store's actual contents unchanged/reproducible.
func readGraphCounts(t *testing.T, codegraphDir string) (nodes, edges int) {
	t.Helper()

	store, err := graphstore.Open(filepath.Join(codegraphDir, storeDirName))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	r, err := store.Snapshot()
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	defer r.Close()

	meta, err := r.GetMeta()
	if err != nil {
		t.Fatalf("get meta: %v", err)
	}
	return int(meta.NodeCount), int(meta.EdgeCount)
}

func TestInitIndexUninit(t *testing.T) {
	dir := copyFixture(t)
	codegraphDir := filepath.Join(dir, codegraphDirName)

	t.Run("init creates .codegraph/ and builds the graph in one step", func(t *testing.T) {
		out, _, err := execCmd("init", dir)
		if err != nil {
			t.Fatalf("init: unexpected error: %v", err)
		}
		if _, statErr := os.Stat(codegraphDir); statErr != nil {
			t.Fatalf(".codegraph/ not created: %v", statErr)
		}
		nodes, edges := readGraphCounts(t, codegraphDir)
		if nodes == 0 || edges == 0 {
			t.Fatalf("expected non-zero nodes/edges after init, got nodes=%d edges=%d", nodes, edges)
		}
		if !strings.Contains(out, "files=") {
			t.Fatalf("expected summary output to mention files, got %q", out)
		}
	})

	t.Run("init a second time errors and does not alter the existing store", func(t *testing.T) {
		nodesBefore, edgesBefore := readGraphCounts(t, codegraphDir)

		_, _, err := execCmd("init", dir)
		if !errors.Is(err, ErrAlreadyInitialized) {
			t.Fatalf("expected ErrAlreadyInitialized, got: %v", err)
		}

		nodesAfter, edgesAfter := readGraphCounts(t, codegraphDir)
		if nodesAfter != nodesBefore || edgesAfter != edgesBefore {
			t.Fatalf("store altered by failed re-init: before=(%d,%d) after=(%d,%d)",
				nodesBefore, edgesBefore, nodesAfter, edgesAfter)
		}
	})

	t.Run("index --force is a deterministic from-scratch rebuild", func(t *testing.T) {
		_, _, err := execCmd("index", "--force", dir)
		if err != nil {
			t.Fatalf("index --force (1st): unexpected error: %v", err)
		}
		nodes1, edges1 := readGraphCounts(t, codegraphDir)

		_, _, err = execCmd("index", "--force", dir)
		if err != nil {
			t.Fatalf("index --force (2nd): unexpected error: %v", err)
		}
		nodes2, edges2 := readGraphCounts(t, codegraphDir)

		if nodes1 != nodes2 || edges1 != edges2 {
			t.Fatalf("non-deterministic rebuild: run1=(%d,%d) run2=(%d,%d)", nodes1, edges1, nodes2, edges2)
		}
		if nodes1 == 0 || edges1 == 0 {
			t.Fatalf("expected non-zero nodes/edges, got nodes=%d edges=%d", nodes1, edges1)
		}
	})

	t.Run("index on an uninitialized directory errors", func(t *testing.T) {
		freshDir := t.TempDir()
		_, _, err := execCmd("index", freshDir)
		if !errors.Is(err, ErrNotInitialized) {
			t.Fatalf("expected ErrNotInitialized, got: %v", err)
		}
	})

	t.Run("uninit without --force and without confirmation does not remove", func(t *testing.T) {
		_, _, err := execCmdWithInput("n\n", "uninit", dir)
		if err != nil {
			t.Fatalf("uninit (declined): unexpected error: %v", err)
		}
		if _, statErr := os.Stat(codegraphDir); statErr != nil {
			t.Fatalf(".codegraph/ should still exist after declining confirmation: %v", statErr)
		}
	})

	t.Run("uninit --force removes .codegraph/ cleanly", func(t *testing.T) {
		_, _, err := execCmd("uninit", "--force", dir)
		if err != nil {
			t.Fatalf("uninit --force: unexpected error: %v", err)
		}
		if _, statErr := os.Stat(codegraphDir); !os.IsNotExist(statErr) {
			t.Fatalf(".codegraph/ should be removed, stat err: %v", statErr)
		}
	})

	t.Run("--quiet suppresses the summary; --verbose adds detail beyond default", func(t *testing.T) {
		quietDir := copyFixture(t)
		out, _, err := execCmd("init", "--quiet", quietDir)
		if err != nil {
			t.Fatalf("init --quiet: unexpected error: %v", err)
		}
		if strings.TrimSpace(out) != "" {
			t.Fatalf("expected no output with --quiet, got %q", out)
		}

		defaultDir := copyFixture(t)
		defaultOut, _, err := execCmd("init", defaultDir)
		if err != nil {
			t.Fatalf("init (default): unexpected error: %v", err)
		}

		verboseDir := copyFixture(t)
		verboseOut, _, err := execCmd("init", "--verbose", verboseDir)
		if err != nil {
			t.Fatalf("init --verbose: unexpected error: %v", err)
		}
		if len(verboseOut) <= len(defaultOut) {
			t.Fatalf("expected --verbose output to be longer than default: default=%q verbose=%q", defaultOut, verboseOut)
		}
	})
}
