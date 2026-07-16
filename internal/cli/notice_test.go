package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/gitmeta"
)

// runGitC mirrors internal/query/engine_worktree_test.go's runGitW and
// internal/mcp/markdown_test.go's runGitM — a third, package-local copy of
// the same hermetic-flags-plus-skip-on-failure shape (D-15), since Go test
// helpers are not importable across packages. Any git failure, including
// git being absent from PATH, skips the calling test (t.Skip, never
// t.Fatal) — WORK-03's best-effort philosophy applies to the fixtures that
// exercise it too.
func runGitC(t *testing.T, dir string, args ...string) string {
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

// statusWorktreeMismatchFixture builds a real, indexed main checkout plus a
// linked worktree nested at .claude/worktrees/probe — D-15's motivating
// true-positive layout, mirroring internal/query/engine_worktree_test.go's
// worktreeMismatchFixture and internal/mcp/markdown_test.go's
// mcpWorktreeMismatchFixture (a third, package-local copy since Go test
// helpers are not importable across packages). The main checkout is indexed
// via the real `codegraph init` command (execCmd), not a direct
// internal/indexer call, exercising the same CLI init path every other
// cli_test.go fixture does. Returns both absolute paths.
func statusWorktreeMismatchFixture(t *testing.T) (worktreeStart, mainRoot string) {
	t.Helper()

	main := copyFixture(t)
	runGitC(t, main, "init")
	runGitC(t, main, "add", "-A")
	runGitC(t, main, "commit", "-m", "init")

	wt := filepath.Join(main, ".claude", "worktrees", "probe")
	runGitC(t, main, "worktree", "add", "-b", "probe", wt)

	if _, _, err := execCmd("init", main); err != nil {
		t.Fatalf("init fixture: unexpected error: %v", err)
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

// noticeGlyph sources the D-11 notice glyph from gitmeta.Mismatch.Notice()
// itself, rather than a pasted literal — one source of truth for the byte
// sequence (U+26A0, no U+FE0F variation selector), so this test file cannot
// silently drift from plan 02-01's constant.
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
// that pre-existing, unrelated warning. This distinguishes the two.
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

type noticeCase struct {
	name string
	args []string
}

// noticeCommandCases is the shared table of the 9 non-status read commands
// this plan wires the compact notice into (WORK-02), each with a symbol/
// query argument the gofixture corpus resolves (Alpha calls helper — see
// query_cli_test.go's TestCallersCalleesCmd). Reused across Test 7
// (mismatch presence), Test 8 (clean-tree absence), and Test 9 (--json
// suppression).
//
// WR-04: "query" and "affected" were originally omitted from this table
// with no documented reason — an undetected blind spot, since both are
// registered read commands (root.go) that call query.OpenAt and render
// human output exactly like the other 7, yet carried no notice. Both are
// now wired in internal/cli/query.go and internal/cli/affected.go and
// covered here.
func noticeCommandCases(path string) []noticeCase {
	return []noticeCase{
		{"explore", []string{"explore", "Alpha", "-p", path}},
		{"node", []string{"node", "Alpha", "-p", path}},
		{"query", []string{"query", "Alpha", "-p", path}},
		{"search", []string{"search", "Alpha", "-p", path}},
		{"callers", []string{"callers", "helper", "-p", path}},
		{"callees", []string{"callees", "Alpha", "-p", path}},
		{"impact", []string{"impact", "helper", "-p", path}},
		{"files", []string{"files", "-p", path}},
		{"affected", []string{"affected", "pkga/pkga.go", "-p", path}},
	}
}

// TestNoticeOnWorktreeMismatch is Test 7 (WORK-02): from a real
// .claude/worktrees/ fixture whose index resolves to the main checkout,
// each of the 7 non-status read commands' output STARTS WITH the compact
// notice. The "files" row passes --path as a RELATIVE path — the Pitfall-3
// absolutization guard: resolveStartPath returns the raw flag value
// unresolved, and plan 02-04 made OpenAt absolutize it; a regression there
// would surface here as a silent false negative (missing notice), not a
// crash.
func TestNoticeOnWorktreeMismatch(t *testing.T) {
	wt, _ := statusWorktreeMismatchFixture(t)
	glyph := noticeGlyph(t)

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}
	relWt, err := filepath.Rel(cwd, wt)
	if err != nil {
		t.Fatalf("filepath.Rel(%s, %s): %v", cwd, wt, err)
	}

	cases := noticeCommandCases(wt)
	for i, tc := range cases {
		if tc.name == "files" {
			cases[i].args = []string{"files", "-p", relWt}
		}
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, _, err := execCmd(tc.args...)
			if err != nil {
				t.Fatalf("%s: unexpected error: %v", tc.name, err)
			}
			if !strings.HasPrefix(out, glyph) {
				t.Fatalf("%s: expected output to start with the compact notice glyph %q, got: %q", tc.name, glyph, out)
			}
		})
	}
}

// TestNoticeAbsentOnCleanTree is Test 8 (D-12): against a clean,
// non-borrowed fixture, none of the 8 commands' output (the 7 non-status
// commands plus status itself) contains the notice glyph.
func TestNoticeAbsentOnCleanTree(t *testing.T) {
	dir := setupIndexedFixture(t)
	glyph := noticeGlyph(t)

	cases := noticeCommandCases(dir)
	cases = append(cases, noticeCase{"status", []string{"status", "-p", dir}})

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, _, err := execCmd(tc.args...)
			if err != nil {
				t.Fatalf("%s: unexpected error: %v", tc.name, err)
			}
			if containsBareNoticeGlyph(out, glyph) {
				t.Fatalf("%s: expected NO notice glyph on a clean tree, got: %q", tc.name, out)
			}
		})
	}
}

// TestNoticeSuppressedInJSON is Test 9 (D-12): --json output on every
// command that has the flag never carries the notice, even against a real
// worktree-mismatch fixture where the human branch WOULD show it — proving
// the print is gated strictly inside each command's human-output branch,
// after the `if jsonOut { …; return }` early return (T-02-31).
func TestNoticeSuppressedInJSON(t *testing.T) {
	wt, _ := statusWorktreeMismatchFixture(t)
	glyph := noticeGlyph(t)

	cases := []noticeCase{
		{"query", []string{"query", "Alpha", "-p", wt, "--json"}},
		{"search", []string{"search", "Alpha", "-p", wt, "--json"}},
		{"callers", []string{"callers", "helper", "-p", wt, "--json"}},
		{"callees", []string{"callees", "Alpha", "-p", wt, "--json"}},
		{"impact", []string{"impact", "helper", "-p", wt, "--json"}},
		{"files", []string{"files", "-p", wt, "--json"}},
		{"affected", []string{"affected", "pkga/pkga.go", "-p", wt, "--json"}},
		{"status", []string{"status", "-p", wt, "--json"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, _, err := execCmd(tc.args...)
			if err != nil {
				t.Fatalf("%s --json: unexpected error: %v", tc.name, err)
			}
			if containsBareNoticeGlyph(out, glyph) {
				t.Fatalf("%s --json: expected NO notice glyph in JSON output, got: %q", tc.name, out)
			}
		})
	}
}
