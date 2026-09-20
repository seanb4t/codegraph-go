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

// pruneGOOSSuffixedFiles deletes every fixture file whose name carries a
// GOOS or GOARCH build-constraint suffix, making the copied corpus index
// to the same file set on every runner.
//
// gofixture deliberately ships `skip_linux.go` so the indexer's own
// discovery tests can prove `go/build.MatchFile` filtering is honored
// (internal/indexer/discover_test.go). That same file makes the corpus
// platform-dependent for everyone else: it is excluded on darwin and
// indexed on linux, so a byte-exact golden frozen on one runner cannot
// match the other. `internal/cli/index_test.go` and
// `internal/query/files_status_test.go` already work around this by
// deriving their expectations instead of hardcoding counts; a golden is
// hardcoded bytes by construction (D-16), so the corpus itself has to be
// portable. Dropping the file is a no-op on darwin — the indexer already
// skipped it — so the frozen goldens stay byte-identical there and start
// matching on linux.
func pruneGOOSSuffixedFiles(t *testing.T, dir string) {
	t.Helper()

	// The GOOS/GOARCH values a filename suffix can name; `go help build`
	// treats `name_$GOOS.go` and `name_$GOARCH.go` as implicit constraints.
	constrained := map[string]bool{
		"aix": true, "android": true, "darwin": true, "dragonfly": true,
		"freebsd": true, "hurd": true, "illumos": true, "ios": true,
		"js": true, "linux": true, "nacl": true, "netbsd": true,
		"openbsd": true, "plan9": true, "solaris": true, "wasip1": true,
		"windows": true, "zos": true,
		"386": true, "amd64": true, "arm": true, "arm64": true,
		"loong64": true, "mips": true, "mips64": true, "mips64le": true,
		"mipsle": true, "ppc64": true, "ppc64le": true, "riscv64": true,
		"s390x": true, "wasm": true,
	}

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		base := strings.TrimSuffix(filepath.Base(path), ".go")
		base = strings.TrimSuffix(base, "_test")
		if i := strings.LastIndex(base, "_"); i >= 0 && constrained[base[i+1:]] {
			return os.Remove(path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("prune GOOS-suffixed fixture files: %v", err)
	}
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

// TestGoldenFixtureIsPlatformPortable is the positive assertion for
// pruneGOOSSuffixedFiles: the corpus the byte-exact goldens are produced
// from must carry no GOOS/GOARCH-suffixed file on ANY runner, or the same
// golden cannot match on darwin and linux both. The raw fixture must
// still ship one (the indexer's discovery tests need it), so this also
// pins that the prune had something to do rather than passing vacuously.
func TestGoldenFixtureIsPlatformPortable(t *testing.T) {
	raw := copyFixture(t)
	if _, err := os.Stat(filepath.Join(raw, "skip_linux.go")); err != nil {
		t.Fatalf("raw gofixture no longer ships skip_linux.go, so this guard is vacuous: %v", err)
	}

	pruned := copyFixture(t)
	pruneGOOSSuffixedFiles(t, pruned)
	if _, err := os.Stat(filepath.Join(pruned, "skip_linux.go")); !os.IsNotExist(err) {
		t.Errorf("skip_linux.go survived the prune (stat err = %v); goldens frozen on darwin cannot match linux", err)
	}

	var leftovers []string
	err := filepath.WalkDir(pruned, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, "_linux.go") {
			leftovers = append(leftovers, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk pruned fixture: %v", err)
	}
	if len(leftovers) > 0 {
		t.Errorf("platform-suffixed files left in the golden corpus: %v", leftovers)
	}
}
