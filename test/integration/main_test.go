// Package integration is TEST-04's subprocess integration harness (D-17):
// a normal Go package (never testdata/ — GOLDEN-01 cost Phase 2 a Critical
// when a suite silently didn't run) so `go test ./...` reaches it, PLUS an
// explicit named CI step (.github/workflows/ci.yml) so a future refactor of
// the filtered `go list ./...` line can never silently drop it either.
//
// Every test in this package drives the REAL, spawned codegraph binary —
// its real argv, its real process cwd, real stdio JSON-RPC — the exact
// cwd/argv->path-derivation->handler wiring seam in-process
// BuildServer->CallTool tests structurally bypass (D-19). That seam is
// precisely where Phase 2's CR-01 regression lived, dead in production for
// a whole phase while every in-process test stayed green.
package integration

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/gitmeta"
)

// binPath is the absolute path to the codegraph binary every test in this
// package spawns — either freshly built from source by TestMain, or an
// externally supplied binary named by the testBinEnvVar environment
// variable (see resolveTestBinPath's doc comment for the resolution
// contract). Never silently the wrong one: TestMain aborts by name rather
// than falling back to a build when the override is set but invalid.
var binPath string

// testBinEnvVar is the environment variable name that lets this harness
// run against an externally supplied binary instead of building one from
// source: CODEGRAPH_TEST_BIN. Defined once so the literal appears exactly
// twice in this package — this doc comment and the assignment below.
const testBinEnvVar = "CODEGRAPH_TEST_BIN"

// resolveTestBinPath resolves the raw value of the testBinEnvVar
// environment variable into a usable binary path. It is a pure function —
// no os.Getenv, no os.Exit, no writes — specifically so it is a table
// test's ideal subject; TestMain is the only caller that touches the
// environment or the process exit code.
//
// Contract: for a non-empty raw value there are exactly two outcomes —
// (path, true, nil): the override is valid, use it; or ("", false, err):
// the override is invalid, abort by name. There is no third outcome: no
// input returns useEnv=true together with a non-nil error, and no
// non-empty input returns useEnv=false with a nil error. That absence is
// the property that forbids a silent fallback to a local `go build` on a
// bad override — a fallback here would let a job claiming to test the
// notarized release binary quietly test a locally rebuilt one instead.
//
// The checks below are STAT-LEVEL only: they confirm raw exists, is a
// regular file, and carries at least one UNIX execute-permission bit
// (owner, group, or other) — never that it is a valid,
// architecture-compatible executable. This is deliberate, not an
// oversight: this repository's harnesses target macOS and Linux only, so
// UNIX mode semantics are the correct and only check to make here. A mode
// bit proves neither architecture compatibility nor that the file is a
// valid Mach-O/ELF; the real "does this binary actually run on this
// machine" question is answered by the test suite itself failing, which
// under ROADMAP criterion 4 is exactly the signal wanted — a
// hardened-runtime library-validation failure should surface as a test
// failure, not be pre-empted by a probe-execute here (which would be a
// redundant surface and a foot-gun in a path that also handles untrusted
// downloads).
func resolveTestBinPath(raw string) (path string, useEnv bool, err error) {
	if raw == "" {
		return "", false, nil
	}

	info, statErr := os.Stat(raw)
	if statErr != nil {
		return "", false, fmt.Errorf("%s=%q: %w", testBinEnvVar, raw, statErr)
	}
	if info.IsDir() {
		return "", false, fmt.Errorf("%s=%q: not a regular file (is a directory)", testBinEnvVar, raw)
	}
	if !info.Mode().IsRegular() {
		return "", false, fmt.Errorf("%s=%q: not a regular file", testBinEnvVar, raw)
	}
	if info.Mode().Perm()&0o111 == 0 {
		return "", false, fmt.Errorf("%s=%q: not executable (no execute permission bit set)", testBinEnvVar, raw)
	}

	abs, absErr := filepath.Abs(raw)
	if absErr != nil {
		return "", false, fmt.Errorf("%s=%q: resolve absolute path: %w", testBinEnvVar, raw, absErr)
	}
	return abs, true, nil
}

// TestMain resolves the testBinEnvVar override first (see
// resolveTestBinPath). When unset, it builds the real release binary
// hermetically, once, into a package-level temp dir before any test in
// this package runs — this harness's entire value proposition is testing
// the REAL production binary (D-18/D-19), so a build failure is a hard
// stop. When set and valid, it runs every test against that externally
// supplied binary instead — no temp dir is created, no build runs, and
// nothing this harness did not create is cleaned up. When set and
// invalid, TestMain aborts before creating a temp dir, before building,
// and before running a single test: it prints a message naming the
// environment variable and the offending path to stderr and exits
// non-zero.
//
// This deliberately inverts the skip-when-absent policy
// internal/upgrade/verify_release_e2e_test.go's e2eArtifactPaths
// established for a missing artifact: that test tolerates absence because
// a skipped signature check is honest, whereas this harness silently
// rebuilding from source on a bad override would report a pass for bytes
// nobody shipped — the opposite of honest.
func TestMain(m *testing.M) {
	resolved, useEnv, err := resolveTestBinPath(os.Getenv(testBinEnvVar))
	if err != nil {
		fmt.Fprintln(os.Stderr, "integration: TestMain:", err)
		os.Exit(1)
	}
	if useEnv {
		binPath = resolved
		os.Exit(m.Run())
	}

	tmpDir, err := os.MkdirTemp("", "codegraph-integration-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "integration: TestMain: MkdirTemp: %v\n", err)
		os.Exit(1)
	}

	binPath = filepath.Join(tmpDir, "codegraph")
	buildCmd := exec.Command("go", "build", "-o", binPath, "github.com/seanb4t/codegraph-go/cmd/codegraph")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "integration: TestMain: go build github.com/seanb4t/codegraph-go/cmd/codegraph failed: %v\n%s\n", err, out)
		_ = os.RemoveAll(tmpDir)
		os.Exit(1)
	}

	code := m.Run()
	_ = os.RemoveAll(tmpDir)
	os.Exit(code)
}

// runGitI mirrors internal/cli/notice_test.go's runGitC (itself the third
// package-local copy of internal/query/engine_worktree_test.go's runGitW
// and internal/mcp/markdown_test.go's runGitM) — a deliberate FOURTH
// package-local copy of the same hermetic-flags-plus-skip-on-failure
// shape, since Go test helpers are not importable across packages. Any
// git failure, including git being absent from PATH, skips the calling
// test (t.Skip, never t.Fatal) — WORK-03's best-effort philosophy applies
// to the fixtures that exercise it too.
func runGitI(t *testing.T, dir string, args ...string) string {
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

// noticeGlyph sources the D-11 notice glyph from gitmeta.Mismatch.Notice()
// itself, rather than a pasted literal — one source of truth for the byte
// sequence (U+26A0, no U+FE0F variation selector), mirroring
// internal/cli/notice_test.go's helper of the same name so this package
// cannot silently drift from Phase 2's constant.
func noticeGlyph(t *testing.T) string {
	t.Helper()
	m := &gitmeta.Mismatch{WorktreeRoot: "/a", IndexRoot: "/b"}
	notice := m.Notice()
	r := []rune(notice)
	if len(r) == 0 {
		t.Fatal("gitmeta.Mismatch.Notice() returned an empty string")
	}
	return string(r[0])
}

// containsBareNoticeGlyph reports whether s contains glyph (U+26A0) NOT
// immediately followed by a U+FE0F variation selector. Phase 1's EXPL-04
// "⚠️ no covering tests found" warning uses the emoji-presentation variant
// (U+26A0 + U+FE0F), whose UTF-8 bytes have the bare glyph's bytes as a
// PREFIX — a naive strings.Contains(s, glyph) would false-positive against
// that pre-existing, unrelated warning (Pitfall 5). This distinguishes the
// two.
func containsBareNoticeGlyph(s, glyph string) bool {
	const variationSelector = "️"
	idx := 0
	for {
		i := strings.Index(s[idx:], glyph)
		if i < 0 {
			return false
		}
		pos := idx + i
		rest := s[pos+len(glyph):]
		if !strings.HasPrefix(rest, variationSelector) {
			return true
		}
		idx = pos + len(glyph)
	}
}

// copyFixture copies internal/indexer/testdata/gofixture into a fresh
// t.TempDir() so subprocess commands run against a normal directory with
// its own go.mod, rather than mutating the checked-in testdata tree —
// mirroring internal/cli/cli_test.go's helper of the same name (this
// package sits two directories deeper, hence the different relative
// path).
func copyFixture(t *testing.T) string {
	t.Helper()

	src, err := filepath.Abs(filepath.Join("..", "..", "internal", "indexer", "testdata", "gofixture"))
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

// runBinary drives the REAL spawned binPath binary via exec.Command with
// cmd.Dir set to dir — D-19's seam: the real cwd/argv the process actually
// receives, not an in-process command-tree call. env, when non-nil, is
// appended to the subprocess's inherited environment (os.Environ()) rather
// than replacing it, so PATH/HOME/etc. survive.
func runBinary(t *testing.T, dir string, env []string, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	cmd := exec.Command(binPath, args...)
	cmd.Dir = dir
	if env != nil {
		cmd.Env = append(os.Environ(), env...)
	}
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}

// buildWorktreeFixture builds a real, indexed main checkout plus a linked
// worktree nested at .claude/worktrees/probe — D-15/D-20's motivating
// true-positive layout, mirroring internal/cli/notice_test.go's
// statusWorktreeMismatchFixture and internal/mcp/markdown_test.go's
// mcpWorktreeMismatchFixture (a third variant, package-local per the
// established convention). Unlike both of those, the main checkout is
// indexed via the SUBPROCESS binary (runBinary), not an in-process
// execCmd/CLI-tree call — the D-19 seam distinction this whole harness
// exists to exercise. Returns absolute worktree-start and main-root paths.
func buildWorktreeFixture(t *testing.T) (worktreeStart, mainRoot string) {
	t.Helper()

	main := copyFixture(t)
	runGitI(t, main, "init")
	runGitI(t, main, "add", "-A")
	runGitI(t, main, "commit", "-m", "init")

	wt := filepath.Join(main, ".claude", "worktrees", "probe")
	runGitI(t, main, "worktree", "add", "-b", "probe", wt)

	// codegraph init <path> (internal/cli/init.go: targetRoot resolves
	// args[0] via filepath.Abs) — driven through the real binary, cwd=main.
	if _, stderr, err := runBinary(t, main, nil, "init", main); err != nil {
		t.Fatalf("init fixture via subprocess binary: %v: %s", err, stderr)
	}

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
