package query

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/gitmeta"
)

// runGitW mirrors internal/gitmeta/fixtures_test.go's runGit. Test files
// are not importable across packages, so this is a minimal local copy of
// the same hermetic-flags-plus-skip-on-failure shape — just enough to
// build the one linked-worktree layout this file's tests need. Any git
// failure, including git being absent from PATH, skips the calling test
// (t.Skip, never t.Fatal) — WORK-03's philosophy applies to the fixtures
// that exercise it too (D-15).
func runGitW(t *testing.T, dir string, args ...string) string {
	t.Helper()
	base := []string{
		"-c", "init.defaultBranch=main",
		"-c", "user.name=codegraph-test",
		"-c", "user.email=test@example.invalid",
		"-c", "commit.gpgsign=false",
		"-c", "protocol.file.allow=always",
	}
	full := append(append([]string{}, base...), args...)
	cmd := exec.Command("git", full...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Skipf("git %v failed (git missing or unsupported here): %v: %s", args, err, string(out))
	}
	return string(out)
}

// worktreeMismatchFixture builds a real, indexed main checkout plus a
// linked worktree nested at .claude/worktrees/probe — D-15's motivating
// true-positive layout (a linked worktree placed under a .gitignore'd
// path INSIDE the main checkout, so the .codegraph/ upward walk from
// inside it resolves to the main checkout, not a worktree-local index; a
// worktree in a sibling directory would not exercise the walk-up at all).
// Reuses engine_test.go's copyFixture/indexFixture for the indexing half
// rather than reimplementing it. Returns both absolute paths.
func worktreeMismatchFixture(t *testing.T) (worktreeStart, mainRoot string) {
	t.Helper()

	main := copyFixture(t)
	runGitW(t, main, "init")
	runGitW(t, main, "add", "-A")
	runGitW(t, main, "commit", "-m", "init")

	wt := filepath.Join(main, ".claude", "worktrees", "probe")
	runGitW(t, main, "worktree", "add", "-b", "probe", wt)

	indexFixture(t, main)

	absMain, err := filepath.Abs(main)
	if err != nil {
		t.Fatalf("filepath.Abs(main): %v", err)
	}
	absWt, err := filepath.Abs(wt)
	if err != nil {
		t.Fatalf("filepath.Abs(wt): %v", err)
	}
	return absWt, absMain
}

// TestEngineWorktreeMismatchViaOpenAt is Test 1 (D-14): an Engine opened
// via OpenAt from inside a linked worktree whose index resolves to the
// main checkout reports a non-nil WorktreeMismatch() naming both trees.
func TestEngineWorktreeMismatchViaOpenAt(t *testing.T) {
	wt, main := worktreeMismatchFixture(t)

	eng, closer, err := OpenAt(wt)
	if err != nil {
		t.Fatalf("OpenAt(%s): unexpected error: %v", wt, err)
	}
	t.Cleanup(func() { _ = closer.Close() })

	got := eng.WorktreeMismatch(context.Background())
	if got == nil {
		t.Fatal("WorktreeMismatch() = nil, want a mismatch (worktree queries the main checkout's index)")
	}

	wantIndexRoot, err := filepath.EvalSymlinks(main)
	if err != nil {
		t.Fatalf("EvalSymlinks(main): %v", err)
	}
	wantWorktreeRoot, err := filepath.EvalSymlinks(wt)
	if err != nil {
		t.Fatalf("EvalSymlinks(wt): %v", err)
	}
	if got.IndexRoot != wantIndexRoot {
		t.Errorf("WorktreeMismatch().IndexRoot = %q, want %q", got.IndexRoot, wantIndexRoot)
	}
	if got.WorktreeRoot != wantWorktreeRoot {
		t.Errorf("WorktreeMismatch().WorktreeRoot = %q, want %q", got.WorktreeRoot, wantWorktreeRoot)
	}
}

// TestEngineWorktreeMismatchDegradesSafely is Test 2 (D-14 degradation):
// Engines built via New or NewWithRoot (no start path context) report no
// mismatch and never panic.
func TestEngineWorktreeMismatchDegradesSafely(t *testing.T) {
	t.Run("New", func(t *testing.T) {
		eng := New(nil)
		if got := eng.WorktreeMismatch(context.Background()); got != nil {
			t.Fatalf("WorktreeMismatch() on a New() Engine = %v, want nil", got)
		}
	})
	t.Run("NewWithRoot", func(t *testing.T) {
		eng := NewWithRoot(nil, "/some/repo/root")
		if got := eng.WorktreeMismatch(context.Background()); got != nil {
			t.Fatalf("WorktreeMismatch() on a NewWithRoot() Engine (no startPath) = %v, want nil", got)
		}
	})
}

// TestStatusWorktreeMismatchLive is Test 3 (WORK-01): Status() surfaces a
// live, non-nil WorktreeMismatch for a mismatching Engine, and nil for a
// normal in-tree Engine.
func TestStatusWorktreeMismatchLive(t *testing.T) {
	wt, main := worktreeMismatchFixture(t)

	// mismatchEngine and cleanEngine both resolve to the SAME store dir
	// (main's .codegraph/store) — Pebble holds an exclusive lock per store
	// directory, so mismatchEngine must be closed before cleanEngine opens
	// (mirrors internal/mcp's openEngine: fresh Engine per call, never two
	// held open concurrently on one store).
	mismatchEngine, closer1, err := OpenAt(wt)
	if err != nil {
		t.Fatalf("OpenAt(%s): unexpected error: %v", wt, err)
	}

	got, err := mismatchEngine.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: unexpected error: %v", err)
	}
	if err := closer1.Close(); err != nil {
		t.Fatalf("closer1.Close(): unexpected error: %v", err)
	}
	if got.WorktreeMismatch == nil {
		t.Fatal("Status().WorktreeMismatch = nil, want a live mismatch")
	}

	cleanEngine, closer2, err := OpenAt(main)
	if err != nil {
		t.Fatalf("OpenAt(%s): unexpected error: %v", main, err)
	}
	t.Cleanup(func() { _ = closer2.Close() })

	gotClean, err := cleanEngine.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: unexpected error: %v", err)
	}
	if gotClean.WorktreeMismatch != nil {
		t.Fatalf("Status().WorktreeMismatch = %v, want nil for an in-tree Engine", gotClean.WorktreeMismatch)
	}
}

// TestStatusWorktreeMismatchJSONShape is Test 4: MarshalStatusJSON of a
// mismatching status decodes to an OBJECT at key "worktreeMismatch" with
// exactly the sub-keys worktreeRoot/indexRoot; of a clean status, to the
// literal JSON null token.
func TestStatusWorktreeMismatchJSONShape(t *testing.T) {
	wt, main := worktreeMismatchFixture(t)

	// mismatchEngine and cleanEngine both resolve to the SAME store dir
	// (main's .codegraph/store) — Pebble holds an exclusive lock per store
	// directory, so mismatchEngine must be closed before cleanEngine opens.
	mismatchEngine, closer1, err := OpenAt(wt)
	if err != nil {
		t.Fatalf("OpenAt(%s): unexpected error: %v", wt, err)
	}

	got, err := mismatchEngine.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: unexpected error: %v", err)
	}
	if err := closer1.Close(); err != nil {
		t.Fatalf("closer1.Close(): unexpected error: %v", err)
	}

	raw, err := MarshalStatusJSON(got)
	if err != nil {
		t.Fatalf("MarshalStatusJSON: unexpected error: %v", err)
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		t.Fatalf("unmarshal status JSON: %v", err)
	}
	sub, ok := top["worktreeMismatch"]
	if !ok {
		t.Fatal(`status JSON is missing the "worktreeMismatch" key`)
	}

	var obj map[string]json.RawMessage
	if err := json.Unmarshal(sub, &obj); err != nil {
		t.Fatalf("worktreeMismatch did not decode as a JSON object: %v (raw=%s)", err, sub)
	}
	if len(obj) != 2 {
		t.Fatalf("worktreeMismatch object has %d keys, want exactly 2 (worktreeRoot, indexRoot): %s", len(obj), sub)
	}
	if _, ok := obj["worktreeRoot"]; !ok {
		t.Errorf(`worktreeMismatch object missing "worktreeRoot": %s`, sub)
	}
	if _, ok := obj["indexRoot"]; !ok {
		t.Errorf(`worktreeMismatch object missing "indexRoot": %s`, sub)
	}

	cleanEngine, closer2, err := OpenAt(main)
	if err != nil {
		t.Fatalf("OpenAt(%s): unexpected error: %v", main, err)
	}
	t.Cleanup(func() { _ = closer2.Close() })

	gotClean, err := cleanEngine.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: unexpected error: %v", err)
	}
	rawClean, err := MarshalStatusJSON(gotClean)
	if err != nil {
		t.Fatalf("MarshalStatusJSON: unexpected error: %v", err)
	}
	var topClean map[string]json.RawMessage
	if err := json.Unmarshal(rawClean, &topClean); err != nil {
		t.Fatalf("unmarshal clean status JSON: %v", err)
	}
	subClean, ok := topClean["worktreeMismatch"]
	if !ok {
		t.Fatal(`clean status JSON is missing the "worktreeMismatch" key`)
	}
	if string(subClean) != "null" {
		t.Fatalf("clean status worktreeMismatch = %s, want the literal JSON null token", subClean)
	}
}

// TestOpenAtAbsolutizesStartPath is Test 5 (Pitfall 3): OpenAt called with
// a RELATIVE start path stores an absolute startPath, so downstream
// equality comparisons against EvalSymlinks-resolved paths are
// byte-comparable.
func TestOpenAtAbsolutizesStartPath(t *testing.T) {
	wt, _ := worktreeMismatchFixture(t)

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}
	rel, err := filepath.Rel(cwd, wt)
	if err != nil {
		t.Fatalf("filepath.Rel(%s, %s): %v", cwd, wt, err)
	}

	eng, closer, err := OpenAt(rel)
	if err != nil {
		t.Fatalf("OpenAt(%s): unexpected error: %v", rel, err)
	}
	t.Cleanup(func() { _ = closer.Close() })

	if !filepath.IsAbs(eng.startPath) {
		t.Fatalf("Engine.startPath = %q (from relative OpenAt(%q)), want an absolute path (Pitfall 3)", eng.startPath, rel)
	}
	wantAbs, err := filepath.Abs(wt)
	if err != nil {
		t.Fatalf("filepath.Abs(wt): %v", err)
	}
	if eng.startPath != wantAbs {
		t.Fatalf("Engine.startPath = %q, want %q", eng.startPath, wantAbs)
	}
}

// TestWorktreeMismatchCachedPerEngine is Test 6 (once-per-Engine): two
// WorktreeMismatch() calls on one Engine return the same pointer,
// evidencing detection runs at most once.
func TestWorktreeMismatchCachedPerEngine(t *testing.T) {
	wt, _ := worktreeMismatchFixture(t)

	eng, closer, err := OpenAt(wt)
	if err != nil {
		t.Fatalf("OpenAt(%s): unexpected error: %v", wt, err)
	}
	t.Cleanup(func() { _ = closer.Close() })

	m1 := eng.WorktreeMismatch(context.Background())
	m2 := eng.WorktreeMismatch(context.Background())
	if m1 == nil {
		t.Fatal("WorktreeMismatch() = nil, want a mismatch to exercise the cache")
	}
	if m1 != m2 {
		t.Fatalf("WorktreeMismatch() returned different pointers across two calls on the same Engine (m1=%p, m2=%p) — detection must run at most once per Engine", m1, m2)
	}
}

// TestWorktreeMismatchSharedDetector is Test 7 (D-13 injection): an Engine
// given a shared CachingDetector via UseDetector serves its verdict from
// that detector, so two Engines sharing one detector detect once in
// total — proven here by pointer identity of the returned *Mismatch
// across two independently-constructed Engines (a private, per-Engine
// detection would produce an equal-value but distinct-pointer *Mismatch).
// eng1 is closed before eng2 opens — Pebble holds an exclusive lock per
// store directory, so two Engines on the SAME store cannot be open
// simultaneously; this also mirrors internal/mcp's openEngine, which
// opens and closes a fresh Engine per call rather than holding two open
// concurrently.
func TestWorktreeMismatchSharedDetector(t *testing.T) {
	wt, _ := worktreeMismatchFixture(t)

	shared := gitmeta.NewCachingDetector()

	eng1, closer1, err := OpenAt(wt)
	if err != nil {
		t.Fatalf("OpenAt(%s): unexpected error: %v", wt, err)
	}
	eng1.UseDetector(shared)
	m1 := eng1.WorktreeMismatch(context.Background())
	if err := closer1.Close(); err != nil {
		t.Fatalf("closer1.Close(): unexpected error: %v", err)
	}

	eng2, closer2, err := OpenAt(wt)
	if err != nil {
		t.Fatalf("OpenAt(%s): unexpected error: %v", wt, err)
	}
	t.Cleanup(func() { _ = closer2.Close() })
	eng2.UseDetector(shared)
	m2 := eng2.WorktreeMismatch(context.Background())

	if m1 == nil || m2 == nil {
		t.Fatalf("WorktreeMismatch() = (%v, %v), want both non-nil to exercise the shared cache", m1, m2)
	}
	if m1 != m2 {
		t.Fatalf("two Engines sharing one CachingDetector returned different *Mismatch pointers (m1=%p, m2=%p) — the shared detector's cache is not being used", m1, m2)
	}
}
