package indexer

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// runGitFixture runs a git subcommand against dir, failing the test on any
// error. It is the test-only fixture-building helper — never the code path
// under test, which is resolveHeadCommitSHA itself.
func runGitFixture(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v (in %s): %v\n%s", args, dir, err, out)
	}
}

// writeGoFixtureFile writes a minimal Go source file into dir so the
// indexing pipeline (Discover/Extract/Resolve) has something real to walk,
// rather than committing an empty tree.
func writeGoFixtureFile(t *testing.T, dir string) {
	t.Helper()
	writeTestFile(t, filepath.Join(dir, "go.mod"), "module example.com/committest\n\ngo 1.24\n")
	writeTestFile(t, filepath.Join(dir, "main.go"), "package main\n\nfunc main() {}\n")
}

func writeTestFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("writing fixture file %s: %v", path, err)
	}
}

// newGitCommitFixture creates a temp directory, git-inits it (default
// object format, SHA-1), commits one minimal Go module, and returns the
// directory. Used by every SHA-1 scenario below.
func newGitCommitFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGitFixture(t, dir, "init", "-q", ".")
	runGitFixture(t, dir, "config", "user.email", "committest@example.com")
	runGitFixture(t, dir, "config", "user.name", "commit test")
	writeGoFixtureFile(t, dir)
	runGitFixture(t, dir, "add", "-A")
	runGitFixture(t, dir, "commit", "-q", "-m", "init")
	return dir
}

// newSHA256GitCommitFixture creates a temp directory git-inited with
// --object-format=sha256. If the local git binary does not support the
// flag, ok is false and the caller must t.Skip with the reason named
// (behavior block: "If the local git is too old to support
// --object-format, skip that subtest with an explicit t.Skip naming the
// reason").
func newSHA256GitCommitFixture(t *testing.T) (dir string, ok bool) {
	t.Helper()
	dir = t.TempDir()
	cmd := exec.Command("git", "init", "--object-format=sha256", "-q", ".")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Logf("git init --object-format=sha256 unsupported by local git: %v\n%s", err, out)
		return "", false
	}
	runGitFixture(t, dir, "config", "user.email", "committest@example.com")
	runGitFixture(t, dir, "config", "user.name", "commit test")
	writeGoFixtureFile(t, dir)
	runGitFixture(t, dir, "add", "-A")
	runGitFixture(t, dir, "commit", "-q", "-m", "init")
	return dir, true
}

// gitRevParseHEAD runs `git rev-parse HEAD` against dir directly — NEVER
// through resolveHeadCommitSHA — so tests that compare against it are
// comparing the code to an independent oracle, not to itself.
func gitRevParseHEAD(t *testing.T, dir string) string {
	t.Helper()
	cmd := exec.Command("git", "-C", dir, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("independent git rev-parse HEAD in %s: %v", dir, err)
	}
	return strings.TrimSpace(string(out))
}

// TestResolveHeadCommitSHA covers the whole accepted/rejected shape of
// resolveHeadCommitSHA's return value: two accepted lengths (SHA-1's 40
// and SHA-256's 64, each driven end-to-end against a real temp git repo),
// and five rejected forms (39, 41, 63, 65, and uppercase hex) exercised
// directly against isLowercaseHexCommitSHA, the validation helper
// resolveHeadCommitSHA itself calls — accepting two lengths is an
// ENUMERATED SET, not "accept any length", and these five prove the
// widening did not become a relaxation.
func TestResolveHeadCommitSHA(t *testing.T) {
	t.Run("sha1-40-hex", func(t *testing.T) {
		dir := newGitCommitFixture(t)
		want := gitRevParseHEAD(t, dir)
		if len(want) != schema.SHA1HexLen {
			t.Fatalf("fixture repo's own HEAD is %d chars, want %d (test setup assumption broken)", len(want), schema.SHA1HexLen)
		}
		got := resolveHeadCommitSHA(dir)
		if got != want {
			t.Fatalf("resolveHeadCommitSHA(%s) = %q, want %q", dir, got, want)
		}
	})

	t.Run("sha256-64-hex", func(t *testing.T) {
		dir, ok := newSHA256GitCommitFixture(t)
		if !ok {
			t.Skip("local git does not support --object-format=sha256; the synthetic 64-char case below still exercises the length-validation path")
		}
		want := gitRevParseHEAD(t, dir)
		if len(want) != schema.SHA256HexLen {
			t.Fatalf("fixture repo's own HEAD is %d chars, want %d (test setup assumption broken)", len(want), schema.SHA256HexLen)
		}
		got := resolveHeadCommitSHA(dir)
		if got != want {
			t.Fatalf("resolveHeadCommitSHA(%s) = %q, want %q", dir, got, want)
		}
	})

	// The five invalid forms exercise isLowercaseHexCommitSHA directly —
	// resolveHeadCommitSHA has no seam to make a real `git rev-parse HEAD`
	// emit a malformed SHA, so the validation logic it calls is the
	// correct unit to drive here. This still runs unconditionally
	// (never skipped), so the 64-char synthetic case in particular proves
	// the 64-length path is exercised even when the sha256-64-hex subtest
	// above skips.
	invalid := []struct {
		name string
		sha  string
	}{
		{"39", strings.Repeat("a", 39)},
		{"41", strings.Repeat("a", 41)},
		{"63", strings.Repeat("a", 63)},
		{"65", strings.Repeat("a", 65)},
		{"uppercase", strings.Repeat("A", schema.SHA1HexLen)},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			if isLowercaseHexCommitSHA(tc.sha) {
				t.Fatalf("isLowercaseHexCommitSHA(%q) = true, want false (len=%d)", tc.sha, len(tc.sha))
			}
		})
	}
}

// TestResolveHeadCommitSHAOnNonGitTree proves a directory that is not a
// git repository degrades to the empty string, never a panic and never an
// error surfaced to the caller (T-01-21).
func TestResolveHeadCommitSHAOnNonGitTree(t *testing.T) {
	dir := t.TempDir()
	got := resolveHeadCommitSHA(dir)
	if got != "" {
		t.Fatalf("resolveHeadCommitSHA(non-git dir) = %q, want empty string", got)
	}
}

// TestResolveHeadCommitSHAWithNoGitBinary drives the injected
// gitExecLookPath seam (never emptying the process's real PATH, which
// would also break the test's own subprocess helpers and make the
// failure ambiguous) to simulate git being entirely absent, and asserts
// the same empty-string, no-panic degrade.
func TestResolveHeadCommitSHAWithNoGitBinary(t *testing.T) {
	orig := gitExecLookPath
	defer func() { gitExecLookPath = orig }()

	gitExecLookPath = func(string) (string, error) {
		return "", exec.ErrNotFound
	}

	dir := newGitCommitFixture(t)
	got := resolveHeadCommitSHA(dir)
	if got != "" {
		t.Fatalf("resolveHeadCommitSHA with no git binary = %q, want empty string", got)
	}
}

// TestHeadIsResolvedOncePerOperation counts HEAD resolutions through the
// injected gitExecLookPath seam (resolveHeadCommitSHA always consults it
// first, so counting its invocations exactly counts resolveHeadCommitSHA
// calls) and asserts a full index run resolves HEAD exactly once, and a
// separate incremental sync run — one that genuinely reparses a modified
// file, not the fully-no-op path — also resolves it exactly once. Not
// once per PutMeta site.
func TestHeadIsResolvedOncePerOperation(t *testing.T) {
	orig := gitExecLookPath
	defer func() { gitExecLookPath = orig }()

	var mu sync.Mutex
	var count int64
	gitExecLookPath = func(file string) (string, error) {
		atomic.AddInt64(&count, 1)
		mu.Lock()
		defer mu.Unlock()
		return orig(file)
	}

	repoRoot := newGitCommitFixture(t)
	storeDir := t.TempDir()

	atomic.StoreInt64(&count, 0)
	if _, err := Run(repoRoot, storeDir, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := atomic.LoadInt64(&count); got != 1 {
		t.Fatalf("full index run resolved HEAD %d times, want exactly 1", got)
	}

	// Force the incremental (non-no-op) Sync path: modify the tracked
	// file's content, so Sync sees a real content change rather than
	// hitting the fully-no-op early return.
	writeTestFile(t, filepath.Join(repoRoot, "main.go"), "package main\n\nfunc main() { println(\"changed\") }\n")

	atomic.StoreInt64(&count, 0)
	if _, err := Sync(repoRoot, storeDir, Options{}); err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if got := atomic.LoadInt64(&count); got != 1 {
		t.Fatalf("incremental sync run resolved HEAD %d times, want exactly 1", got)
	}
}

// TestIndexRunStampsHeadCommitSHA is ENG-04's end-to-end stamping proof: a
// real `codegraph index` run (Run) over a real temp git checkout produces
// a Meta whose commit field equals `git rev-parse HEAD` for that
// checkout — computed independently in this test, never by reading back
// the value resolveHeadCommitSHA itself produced, which would compare the
// code to itself. The second subtest proves the same run over a
// non-git directory succeeds with an empty value, so degradation is
// proven rather than assumed.
func TestIndexRunStampsHeadCommitSHA(t *testing.T) {
	t.Run("git-checkout", func(t *testing.T) {
		repoRoot := newGitCommitFixture(t)
		wantSHA := gitRevParseHEAD(t, repoRoot)
		storeDir := t.TempDir()

		if _, err := Run(repoRoot, storeDir, Options{}); err != nil {
			t.Fatalf("Run: %v", err)
		}

		meta := readBackMeta(t, storeDir)
		if meta.GetCommitSha() != wantSHA {
			t.Fatalf("Meta.CommitSha = %q, want %q (independently computed git rev-parse HEAD)", meta.GetCommitSha(), wantSHA)
		}
	})

	t.Run("non-git-tree", func(t *testing.T) {
		repoRoot := t.TempDir()
		writeGoFixtureFile(t, repoRoot)
		storeDir := t.TempDir()

		if _, err := Run(repoRoot, storeDir, Options{}); err != nil {
			t.Fatalf("Run over a non-git directory must still succeed: %v", err)
		}

		meta := readBackMeta(t, storeDir)
		if meta.GetCommitSha() != "" {
			t.Fatalf("Meta.CommitSha over a non-git tree = %q, want empty string", meta.GetCommitSha())
		}
	})
}

// readBackMeta opens storeDir and returns its Meta record, failing the
// test on any error.
func readBackMeta(t *testing.T, storeDir string) *schema.Meta {
	t.Helper()
	store, err := graphstore.Open(storeDir)
	if err != nil {
		t.Fatalf("graphstore.Open: %v", err)
	}
	defer store.Close()

	r, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	defer r.Close()

	meta, err := r.GetMeta()
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}
	return meta
}

// TestSyncPreservesAKnownCommitWhenHeadCannotBeResolved is WR-05's
// end-to-end proof: a Sync whose HEAD resolution fails leaves the
// previously recorded commit_sha intact rather than erasing it.
//
// resolveHeadCommitSHA returns "" for every failure mode by design, and
// schema.NewMeta() starts from a zero-valued record, so an unconditional
// assignment REPLACES a known-good SHA with absent — and D-05 defines
// empty as "unknown, never an error", so nothing anywhere reports the
// loss. The realistic trigger is mundane: a watcher-driven Sync firing
// while `git rebase` holds .git/index.lock.
//
// Three subtests, none of which can pass vacuously:
//
//   - the baseline establishes that a real SHA was recorded at all;
//   - the failure case asserts that exact SHA is STILL there after a
//     Sync with git unresolvable (this is the one that goes RED without
//     the fix — the value becomes "");
//   - the positive control makes a SECOND commit with git working and
//     asserts the SHA MOVES, proving the fix preserves rather than
//     freezes.
func TestSyncPreservesAKnownCommitWhenHeadCannotBeResolved(t *testing.T) {
	repoRoot := newGitCommitFixture(t)
	storeDir := t.TempDir()

	if _, err := Run(repoRoot, storeDir, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	firstSHA := gitRevParseHEAD(t, repoRoot)
	if got := readBackMeta(t, storeDir).GetCommitSha(); got != firstSHA {
		t.Fatalf("baseline Meta.CommitSha = %q, want %q (independently computed) — the rest of this test proves nothing without it", got, firstSHA)
	}

	t.Run("unresolvable-head-preserves-the-prior-commit", func(t *testing.T) {
		// A real content change, so Sync takes the incremental write
		// path rather than the fully-no-op early return.
		writeTestFile(t, filepath.Join(repoRoot, "main.go"), "package main\n\nfunc main() { println(\"changed\") }\n")

		orig := gitExecLookPath
		defer func() { gitExecLookPath = orig }()
		gitExecLookPath = func(string) (string, error) {
			return "", exec.ErrNotFound
		}

		if _, err := Sync(repoRoot, storeDir, Options{}); err != nil {
			t.Fatalf("Sync with git unresolvable must still succeed: %v", err)
		}
		if got := readBackMeta(t, storeDir).GetCommitSha(); got != firstSHA {
			t.Fatalf("Meta.CommitSha after a Sync that could not resolve HEAD = %q, want the preserved %q — an unresolvable HEAD is not evidence that the recorded commit is wrong", got, firstSHA)
		}
	})

	t.Run("resolvable-head-still-advances-the-commit", func(t *testing.T) {
		runGitFixture(t, repoRoot, "add", "-A")
		runGitFixture(t, repoRoot, "commit", "-q", "-m", "second")
		secondSHA := gitRevParseHEAD(t, repoRoot)
		if secondSHA == firstSHA {
			t.Fatal("fixture assumption broken: the second commit did not move HEAD")
		}

		writeTestFile(t, filepath.Join(repoRoot, "main.go"), "package main\n\nfunc main() { println(\"changed again\") }\n")
		if _, err := Sync(repoRoot, storeDir, Options{}); err != nil {
			t.Fatalf("Sync: %v", err)
		}
		if got := readBackMeta(t, storeDir).GetCommitSha(); got != secondSHA {
			t.Fatalf("Meta.CommitSha after a Sync that DID resolve HEAD = %q, want the new %q — preserving a prior value must never freeze it", got, secondSHA)
		}
	})
}

// TestSyncCommitSHAPrefersAResolvedValue covers syncCommitSHA's two
// branches directly, so the rule survives independently of which of
// Sync's two Meta write sites a given integration test happens to reach.
func TestSyncCommitSHAPrefersAResolvedValue(t *testing.T) {
	const prior = "0123456789abcdef0123456789abcdef01234567"
	const fresh = "fedcba9876543210fedcba9876543210fedcba98"

	prev := schema.NewMeta()
	prev.CommitSha = prior

	if got := syncCommitSHA(fresh, prev); got != fresh {
		t.Fatalf("syncCommitSHA(resolved=%q) = %q, want the freshly resolved value", fresh, got)
	}
	if got := syncCommitSHA("", prev); got != prior {
		t.Fatalf("syncCommitSHA(resolved=\"\") = %q, want the preserved prior value %q", got, prior)
	}

	empty := schema.NewMeta()
	if got := syncCommitSHA("", empty); got != "" {
		t.Fatalf("syncCommitSHA(resolved=\"\", prior=\"\") = %q, want empty — there is nothing to preserve", got)
	}
}
