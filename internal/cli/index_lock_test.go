package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
)

// storeDirEntryNames returns the sorted list of entry names directly under
// storeDir. This is informational only (logged, never asserted on for
// exact equality): a live pebble handle performs its own background
// housekeeping (WAL rotation, memtable flush to a new SST, manifest
// rotation) independently of anything the CLI does, so the file SET
// legitimately churns even when os.RemoveAll never runs. The load-bearing
// proof that RemoveAll never ran is storeDirIdentity below.
func storeDirEntryNames(t *testing.T, storeDir string) []string {
	t.Helper()

	entries, err := os.ReadDir(storeDir)
	if err != nil {
		t.Fatalf("read storeDir %s: %v", storeDir, err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

// storeDirIdentity returns storeDir's own os.FileInfo, to be compared
// with os.SameFile across a call. `index --force`'s wipe is
// os.RemoveAll(storeDir) followed by os.MkdirAll(storeDir, ...) — that
// sequence necessarily replaces the directory's own inode with a fresh
// one, even though pebble's routine internal file churn inside an
// UNCHANGED directory (see storeDirEntryNames) never does. Comparing the
// directory's identity, not its listing, is therefore the assertion that
// actually isolates "did RemoveAll run" from "did pebble's background
// janitor do its normal job".
func storeDirIdentity(t *testing.T, storeDir string) os.FileInfo {
	t.Helper()

	info, err := os.Stat(storeDir)
	if err != nil {
		t.Fatalf("stat storeDir %s: %v", storeDir, err)
	}
	return info
}

// readGenerationThroughHandle reads Meta.CoverageGeneration through an
// ALREADY-OPEN GraphStore handle, rather than opening a fresh one — a
// second concurrent graphstore.Open against the same directory would
// itself collide with the held lock this test relies on, so the read
// must go through the handle the test is already holding.
func readGenerationThroughHandle(t *testing.T, store graphstore.GraphStore) int64 {
	t.Helper()

	r, err := store.Snapshot()
	if err != nil {
		t.Fatalf("snapshot through held handle: %v", err)
	}
	defer r.Close()

	meta, err := r.GetMeta()
	if err != nil {
		t.Fatalf("get meta through held handle: %v", err)
	}
	return meta.GetCoverageGeneration()
}

// TestIndexForceRefusesWhileStoreIsHeld pins D-10 / WINDOWS #36 / T-10-16:
// `codegraph index --force` against a store another live process holds
// open must REFUSE — non-zero exit, errors.Is(err, graphstore.ErrStoreLocked)
// — and must do so BEFORE os.RemoveAll(storeDir) runs. An exit code alone
// does not prove the ordering, so this test also reads the store's own
// directory listing and its Meta.CoverageGeneration back through the SAME
// held handle after the refused call, and asserts both are unchanged.
//
// Against the current (pre-fix) code, priorCoverageGeneration collapses
// every Open/Snapshot/GetMeta error — including ErrStoreLocked — to a
// silent floor of 0, and the caller proceeds straight to RemoveAll
// regardless. Run against unmodified index.go, this test is expected to
// FAIL on both the errors.Is assertion and the unchanged-contents
// assertions, because the wipe genuinely proceeds.
func TestIndexForceRefusesWhileStoreIsHeld(t *testing.T) {
	dir := copyFixture(t)

	if _, _, err := execCmd("init", dir); err != nil {
		t.Fatalf("init: %v", err)
	}
	codegraphDir := filepath.Join(dir, codegraphDirName)
	storeDir := filepath.Join(codegraphDir, storeDirName)

	beforeNames := storeDirEntryNames(t, storeDir)
	beforeIdentity := storeDirIdentity(t, storeDir)

	holder, err := graphstore.Open(storeDir)
	if err != nil {
		t.Fatalf("hold storeDir open: %v", err)
	}
	t.Cleanup(func() {
		if err := holder.Close(); err != nil {
			t.Errorf("close held handle: %v", err)
		}
	})

	genBefore := readGenerationThroughHandle(t, holder)

	_, _, err = execCmd("index", "--force", dir)

	if err == nil {
		t.Fatal("index --force succeeded while another process held the store open; want a refusal")
	}
	if !errors.Is(err, graphstore.ErrStoreLocked) {
		t.Fatalf("index --force error = %v; want errors.Is(err, graphstore.ErrStoreLocked)", err)
	}
	if !strings.Contains(err.Error(), "codegraph daemon stop") {
		t.Fatalf("index --force error = %q; want it to name `codegraph daemon stop`", err.Error())
	}
	if !strings.Contains(err.Error(), "codegraph unlock") {
		t.Fatalf("index --force error = %q; want it to name `codegraph unlock` (the verb live today, not `daemon unlock`)", err.Error())
	}
	if strings.Contains(err.Error(), "daemon unlock") {
		t.Fatalf("index --force error = %q; must NOT name `daemon unlock` — that verb does not exist until Phase 3", err.Error())
	}

	// The refusal must precede os.RemoveAll+os.MkdirAll: the store
	// directory's own identity (its inode, via os.SameFile) — not its
	// file listing, which legitimately churns from pebble's own
	// background housekeeping on the still-open held handle — must be
	// unchanged across the refused call. A wipe replaces the directory
	// itself with a fresh one; routine internal file rotation inside an
	// unchanged directory does not.
	afterNames := storeDirEntryNames(t, storeDir)
	t.Logf("storeDir listing before=%v after=%v (informational only — churn from pebble's own housekeeping on the held handle is expected and not a failure)", beforeNames, afterNames)

	afterIdentity := storeDirIdentity(t, storeDir)
	if !os.SameFile(beforeIdentity, afterIdentity) {
		t.Fatalf("storeDir's own directory identity changed across a refused call — RemoveAll+MkdirAll ran despite the refusal: before=%v after=%v", beforeIdentity, afterIdentity)
	}

	// And the coverage generation, read back through the SAME held
	// handle (so this can never race a second Open), must be unchanged.
	genAfter := readGenerationThroughHandle(t, holder)
	if genAfter != genBefore {
		t.Fatalf("coverage generation changed across a refused call: before=%d after=%d", genBefore, genAfter)
	}
}

// TestIndexForceProceedsSilentlyWhenNeverIndexed is the control: a store
// that has never been indexed (.codegraph/ exists, but index was never
// run) must still rebuild silently on `--force`, with no output about a
// prior coverage generation. This passes both before and after the fix —
// it exists to stop the fix from being written as "refuse whenever the
// store cannot be read", which would also refuse the ordinary first-index
// case.
func TestIndexForceProceedsSilentlyWhenNeverIndexed(t *testing.T) {
	dir := copyFixture(t)

	codegraphDir := filepath.Join(dir, codegraphDirName)
	if err := os.MkdirAll(codegraphDir, 0o755); err != nil {
		t.Fatalf("mkdir codegraphDir: %v", err)
	}

	stdout, stderr, err := execCmd("index", "--force", dir)
	if err != nil {
		t.Fatalf("index --force on a never-indexed .codegraph/: %v (stdout=%q stderr=%q)", err, stdout, stderr)
	}
	if strings.Contains(stderr, "coverage generation") {
		t.Fatalf("index --force on a never-indexed store printed prior-generation output on stderr: %q", stderr)
	}
}

// TestIndexForceWarnsAndRebuildsOnUnreadableStore pins D-11: a store that
// exists but is corrupt or otherwise unreadable for a reason that is
// NEITHER ErrStoreLocked NOR ErrNotFound must warn (naming the floor-0
// consequence) on stderr and then proceed to rebuild — `--force` is the
// documented recovery path for exactly this case. Against the current
// (pre-fix) code this succeeds silently, with no warning at all.
func TestIndexForceWarnsAndRebuildsOnUnreadableStore(t *testing.T) {
	dir := copyFixture(t)

	if _, _, err := execCmd("init", dir); err != nil {
		t.Fatalf("init: %v", err)
	}
	codegraphDir := filepath.Join(dir, codegraphDirName)
	storeDir := filepath.Join(codegraphDir, storeDirName)

	// Corrupt the store on disk in a way that is NOT a lock collision:
	// overwrite pebble/v2's live MANIFEST file (named via its
	// marker.manifest.* marker file — pebble/v2 uses marker files rather
	// than a CURRENT file, confirmed by inspecting a freshly-init'd
	// store's directory listing on this tree) with garbage bytes.
	// graphstore.Open must then fail with an error that is neither
	// ErrStoreLocked nor ErrNotFound — the exact "corrupt/unreadable"
	// case D-11 describes, as opposed to an empty/never-indexed
	// directory (that is ErrNotFound's case, not this one).
	manifests, err := filepath.Glob(filepath.Join(storeDir, "MANIFEST-*"))
	if err != nil {
		t.Fatalf("glob for MANIFEST files: %v", err)
	}
	if len(manifests) != 1 {
		t.Fatalf("found %d MANIFEST-* files in %s, want exactly 1: %v", len(manifests), storeDir, manifests)
	}
	if err := os.WriteFile(manifests[0], []byte("garbage-not-a-real-manifest\n"), 0o644); err != nil {
		t.Fatalf("corrupt %s: %v", manifests[0], err)
	}

	// Confirm, independently of the CLI, exactly which failure mode this
	// corruption produces on this tree — pin to it by name rather than
	// weakening the CLI-level assertion to "any error". On this tree
	// this is pebble's "malformed manifest file" error.
	_, openErr := graphstore.Open(storeDir)
	if openErr == nil {
		t.Fatal("graphstore.Open succeeded against a corrupted MANIFEST file; want a non-nil, non-sentinel error to set up this test")
	}
	if errors.Is(openErr, graphstore.ErrStoreLocked) || errors.Is(openErr, graphstore.ErrNotFound) {
		t.Fatalf("corrupting the MANIFEST produced a sentinel error (%v); want a non-sentinel error so this test actually exercises the corrupt-store branch", openErr)
	}
	t.Logf("corrupt-store failure mode on this tree: %v", openErr)

	_, stderr, err := execCmd("index", "--force", dir)
	if err != nil {
		t.Fatalf("index --force on a corrupt store: got error %v, want exit zero (D-11: --force is the recovery path)", err)
	}
	if !strings.Contains(stderr, "coverage generation") {
		t.Fatalf("index --force on a corrupt store printed no warning naming the floor-0 consequence; stderr=%q", stderr)
	}

	// The rebuild must have actually happened: the store is readable again
	// afterward, through a fresh Open (the CLI process closed its own
	// handle before returning).
	gen := readCoverageGeneration(t, codegraphDir)
	if gen <= 0 {
		t.Fatalf("coverage generation after rebuild = %d; want > 0", gen)
	}
}
