package indexer

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/flock"
	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/graphstore/storetest"
	"github.com/seanb4t/codegraph-go/internal/scratchdir"
)

// The kill tests start this test binary again as a child. The child runs
// TestHelperKilledIndex, which indexes the tree named by the environment
// and sends itself SIGKILL from the hook that killEnv selects.
const (
	killEnv       = "CODEGRAPH_TEST_KILL"
	killRootEnv   = "CODEGRAPH_TEST_KILL_ROOT"
	killStoreEnv  = "CODEGRAPH_TEST_KILL_STORE"
	killMarkerEnv = "CODEGRAPH_TEST_KILL_MARKER"

	killAtCommit   = "commit"
	killAtExchange = "exchange"
)

// TestHelperKilledIndex is the body of the child process of the kill
// tests. Without the environment of a kill test, it does nothing.
func TestHelperKilledIndex(t *testing.T) {
	mode := os.Getenv(killEnv)
	if mode == "" {
		t.Skip("this test is the child process of the kill tests")
	}
	die := func() {
		// The marker tells the parent that the hook ran, so the kill came
		// from the hook and not from a failure before it.
		if err := os.WriteFile(os.Getenv(killMarkerEnv), []byte(mode), 0o644); err != nil {
			panic(err)
		}
		_ = syscall.Kill(os.Getpid(), syscall.SIGKILL)
		select {}
	}
	batchCap = lowCap
	switch mode {
	case killAtCommit:
		commits := 0
		onCommit = func(graphstore.Commit) error {
			if commits++; commits == 2 {
				die()
			}
			return nil
		}
	case killAtExchange:
		onExchange = die
	}
	_, err := run(context.Background(), os.Getenv(killRootEnv), os.Getenv(killStoreEnv), Options{})
	t.Fatalf("the run returned %v; the hook did not kill the process", err)
}

// runKilledIndex runs a full index of root into storeDir in a child process
// that kills itself at mode, and checks that it died by that kill.
func runKilledIndex(t *testing.T, root, storeDir, mode string) {
	t.Helper()
	marker := filepath.Join(t.TempDir(), "killed")
	cmd := exec.Command(os.Args[0], "-test.run=^TestHelperKilledIndex$", "-test.count=1")
	cmd.Env = append(os.Environ(), killEnv+"="+mode, killRootEnv+"="+root, killStoreEnv+"="+storeDir, killMarkerEnv+"="+marker)
	out, err := cmd.CombinedOutput()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("the child process ended with %v, want a kill; output:\n%s", err, out)
	}
	if ws, ok := exitErr.Sys().(syscall.WaitStatus); !ok || !ws.Signaled() || ws.Signal() != syscall.SIGKILL {
		t.Fatalf("the child process ended with %v, want SIGKILL; output:\n%s", err, out)
	}
	// Positive assertion: the hook ran, so the run reached its kill point.
	if got, err := os.ReadFile(marker); err != nil || string(got) != mode {
		t.Fatalf("the kill marker holds %q, %v; want %q, written by the hook", got, err, mode)
	}
}

// indexedTree returns a tree of 500 Go files and the path of a store of it
// inside the tree, as `codegraph init` lays it out, with one full index
// done.
func indexedTree(t *testing.T) (root, storeDir string) {
	t.Helper()
	root = goTree(t, 500, false)
	storeDir = filepath.Join(root, ".codegraph", "store")
	if _, err := Run(root, storeDir, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	return root, storeDir
}

// freshExport returns the export of a new full index of root, into a store
// outside the tree, with the coverage generation cleared.
func freshExport(t *testing.T, root string) []byte {
	t.Helper()
	storeDir := filepath.Join(t.TempDir(), "store")
	if _, err := Run(root, storeDir, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	return storetest.ExportNormalized(t, storeDir, true)
}

// failAtCommit makes the nth commit of a run fail, under the low cap. It
// returns the number of commits that ran.
func failAtCommit(t *testing.T, n int) *int {
	t.Helper()
	setHook(t, &batchCap, lowCap)
	commits := new(int)
	setHook(t, &onCommit, func(graphstore.Commit) error {
		if *commits++; *commits == n {
			return errInjectedCommitFailure
		}
		return nil
	})
	return commits
}

func TestFailedIndexKeepsStore(t *testing.T) {
	root, storeDir := indexedTree(t)
	before := storetest.ExportNormalized(t, storeDir, false)
	writeFile(t, filepath.Join(root, "pkg", "added.go"), "package pkg\n\nfunc Added() {}\n")
	commits := failAtCommit(t, 3)

	_, err := run(context.Background(), root, storeDir, Options{})
	if !errors.Is(err, errInjectedCommitFailure) {
		t.Fatalf("run error = %v, want the error of the third commit", err)
	}
	// Positive assertion: the run committed batches before it stopped.
	if *commits != 3 {
		t.Fatalf("%d commits ran, want 3", *commits)
	}
	if !bytes.Equal(storetest.ExportNormalized(t, storeDir, false), before) {
		t.Errorf("the store changed in a full index that failed")
	}
}

func TestKilledIndexKeepsStore(t *testing.T) {
	root, storeDir := indexedTree(t)
	before := storetest.ExportNormalized(t, storeDir, false)
	writeFile(t, filepath.Join(root, "pkg", "added.go"), "package pkg\n\nfunc Added() {}\n")

	runKilledIndex(t, root, storeDir, killAtCommit)
	if !bytes.Equal(storetest.ExportNormalized(t, storeDir, false), before) {
		t.Fatalf("the store changed in a full index that was killed")
	}
	if left := scratchDirs(t, storeDir); len(left) != 1 {
		t.Fatalf("the killed run left the scratch directories %v, want its one directory", left)
	}

	// A second full index runs to its end and cleans up after the first.
	if _, err := Run(root, storeDir, Options{}); err != nil {
		t.Fatalf("Run after the killed run: %v", err)
	}
	if left := scratchDirs(t, storeDir); len(left) != 0 {
		t.Errorf("the scratch directories %v are still there after the next full index", left)
	}
	if !bytes.Equal(storetest.ExportNormalized(t, storeDir, true), freshExport(t, root)) {
		t.Errorf("the store after the second full index differs from a fresh index")
	}
}

func TestKilledAfterExchange(t *testing.T) {
	root, storeDir := indexedTree(t)
	writeFile(t, filepath.Join(root, "pkg", "added.go"), "package pkg\n\nfunc Added() {}\n")

	runKilledIndex(t, root, storeDir, killAtExchange)
	if !bytes.Equal(storetest.ExportNormalized(t, storeDir, true), freshExport(t, root)) {
		t.Errorf("the store does not hold the new graph after a kill that came after the exchange")
	}
}

func TestFirstIndexFailureLeavesNoGraph(t *testing.T) {
	root := goTree(t, 500, false)
	// The layout that `codegraph init` leaves before its first index: the
	// .codegraph directory with an empty store directory.
	storeDir := filepath.Join(root, ".codegraph", "store")
	if err := os.MkdirAll(storeDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	commits := failAtCommit(t, 3)

	_, err := run(context.Background(), root, storeDir, Options{})
	if !errors.Is(err, errInjectedCommitFailure) {
		t.Fatalf("run error = %v, want the error of the third commit", err)
	}
	if *commits != 3 {
		t.Fatalf("%d commits ran, want 3", *commits)
	}
	// An export holds every node, edge, file record and metadata record.
	if export := storetest.ExportNormalized(t, storeDir, false); len(export) != 0 {
		t.Errorf("the store holds %d bytes of records after a first index that failed, want none", len(export))
	}
}

func TestFirstIndexWithConcurrentOpen(t *testing.T) {
	root := goTree(t, 500, false)
	storeDir := filepath.Join(root, ".codegraph", "store")
	// The fresh index runs first, with its store outside the tree. The
	// tree has no .codegraph directory yet, so the excluded-file records
	// of both runs are those of a tree without one.
	fresh := freshExport(t, root)
	opened := false
	setHook(t, &onPass, func(step string, _ *runState) {
		if step != stepExtract {
			return
		}
		// An open creates the live store while the run builds its own.
		s, err := graphstore.Open(storeDir)
		if err != nil {
			t.Errorf("graphstore.Open at the start of pass 1: %v", err)
			return
		}
		opened = true
		if err := s.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})

	if _, err := run(context.Background(), root, storeDir, Options{}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !opened {
		t.Fatal("the hook did not open the store")
	}
	if !bytes.Equal(storetest.ExportNormalized(t, storeDir, true), fresh) {
		t.Errorf("the store differs from a fresh index, the excluded-file records included")
	}
}

func TestSuccessfulIndexDropsOldRecords(t *testing.T) {
	root, storeDir := indexedTree(t)
	if err := os.Remove(filepath.Join(root, "pkg", "f0007.go")); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, err := Run(root, storeDir, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !bytes.Equal(storetest.ExportNormalized(t, storeDir, true), freshExport(t, root)) {
		t.Errorf("the store differs from a fresh index of the changed tree: old records stayed")
	}
}

func TestIndexLeavesNoScratch(t *testing.T) {
	root, storeDir := indexedTree(t)
	t.Run("after a commit error", func(t *testing.T) {
		commits := failAtCommit(t, 2)
		if _, err := run(context.Background(), root, storeDir, Options{}); !errors.Is(err, errInjectedCommitFailure) {
			t.Fatalf("run error = %v, want the injected commit error", err)
		}
		if *commits != 2 {
			t.Fatalf("%d commits ran, want 2", *commits)
		}
		if left := scratchDirs(t, storeDir); len(left) != 0 {
			t.Errorf("the run left the scratch directories %v", left)
		}
	})
	t.Run("after a success", func(t *testing.T) {
		if _, err := Run(root, storeDir, Options{}); err != nil {
			t.Fatalf("Run: %v", err)
		}
		if left := scratchDirs(t, storeDir); len(left) != 0 {
			t.Errorf("the run left the scratch directories %v", left)
		}
		for _, name := range []string{spillName, scratchdir.LockName} {
			if _, err := os.Lstat(filepath.Join(storeDir, name)); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("the store directory holds %s after the exchange: %v", name, err)
			}
		}
	})
}

func TestExchangeFailureKeepsStore(t *testing.T) {
	root, storeDir := indexedTree(t)
	before := storetest.ExportNormalized(t, storeDir, false)
	var held graphstore.GraphStore
	setHook(t, &onPass, func(step string, _ *runState) {
		if step != stepResolve {
			return
		}
		s, err := graphstore.Open(storeDir)
		if err != nil {
			t.Errorf("graphstore.Open at the start of pass 3: %v", err)
			return
		}
		held = s
	})

	_, err := run(context.Background(), root, storeDir, Options{})
	if held == nil {
		t.Fatal("the hook did not open the store")
	}
	if !errors.Is(err, graphstore.ErrStoreLocked) {
		held.Close()
		t.Fatalf("run with the live store held open = %v, want ErrStoreLocked", err)
	}
	if left := scratchDirs(t, storeDir); len(left) != 0 {
		t.Errorf("the run left the scratch directories %v", left)
	}
	if err := held.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	onPass = nil
	if !bytes.Equal(storetest.ExportNormalized(t, storeDir, false), before) {
		t.Errorf("the store changed in a full index whose exchange failed")
	}

	// Positive assertion: the failed run left no lock in this process.
	lock, err := graphstore.LockWriter(storeDir)
	if err != nil {
		t.Fatalf("LockWriter after the failed run: %v", err)
	}
	lock.Close()
	if _, err := Run(root, storeDir, Options{}); err != nil {
		t.Fatalf("a second full index: %v", err)
	}
}

func TestIndexSweepsScratch(t *testing.T) {
	prefix := "store" + scratchSuffix
	t.Run("a pending directory whose lock no process holds", func(t *testing.T) {
		root, storeDir := indexedTree(t)
		leftover := filepath.Join(filepath.Dir(storeDir), scratchdir.PendingPrefix(prefix)+"crashed")
		if err := os.MkdirAll(filepath.Join(leftover, stagingName), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		writeFile(t, filepath.Join(leftover, scratchdir.LockName), "")

		if _, err := Run(root, storeDir, Options{}); err != nil {
			t.Fatalf("Run: %v", err)
		}
		if _, err := os.Lstat(leftover); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("Lstat(%s) = %v, want the leftover gone after the run", leftover, err)
		}
	})
	t.Run("a directory whose lock the test holds", func(t *testing.T) {
		root, storeDir := indexedTree(t)
		live := filepath.Join(filepath.Dir(storeDir), prefix+"building")
		if err := os.MkdirAll(filepath.Join(live, stagingName), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		lockPath := filepath.Join(live, scratchdir.LockName)
		lock, err := flock.Open(lockPath, flock.Exclusive)
		if err != nil {
			t.Fatalf("flock.Open: %v", err)
		}
		defer lock.Close()

		if _, err := Run(root, storeDir, Options{}); err != nil {
			t.Fatalf("Run: %v", err)
		}
		for _, path := range []string{filepath.Join(live, stagingName), lockPath} {
			if _, err := os.Lstat(path); err != nil {
				t.Errorf("Lstat(%s) = %v, want the run to leave a directory that another build holds", path, err)
			}
		}
	})
}
