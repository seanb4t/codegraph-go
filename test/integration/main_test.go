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

// binPath is the absolute path to the real codegraph binary TestMain
// builds once per `go test` run (D-18) — every test in this package reads
// it; the harness never relies on a pre-built artifact or a PATH lookup.
var binPath string

// TestMain builds the real release binary hermetically, once, into a
// package-level temp dir before any test in this package runs. This
// harness's entire value proposition is testing the REAL production
// binary (D-18/D-19), so a build failure is a hard stop — it prints the
// build output and os.Exit(1) rather than letting every test silently
// skip or fail with a confusing "file not found".
func TestMain(m *testing.M) {
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
