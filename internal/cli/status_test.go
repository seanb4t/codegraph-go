package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/agents"
	"github.com/seanb4t/codegraph-go/internal/realpath/realpathtest"
)

// TestInstallAndUninstallStatus: a result with errors reads "failed" when
// no file changed and "partial" when some did, so the headline never reads
// as success above an error line. Without errors the words are unchanged.
func TestInstallAndUninstallStatus(t *testing.T) {
	failure := []error{errors.New("boom")}
	file := func(action agents.FileAction) []agents.FileResult {
		return []agents.FileResult{{Path: "p", Action: action}}
	}
	cases := []struct {
		name          string
		result        agents.WriteResult
		wantInstall   string
		wantUninstall string
	}{
		{"nothing touched", agents.WriteResult{}, "unchanged", "not-configured"},
		{"only unchanged files", agents.WriteResult{Files: file(agents.ActionUnchanged)}, "unchanged", "not-configured"},
		{"a created file", agents.WriteResult{Files: file(agents.ActionCreated)}, "configured", "not-configured"},
		{"a removed file", agents.WriteResult{Files: file(agents.ActionRemoved)}, "configured", "removed"},
		{"only kept entries", agents.WriteResult{Files: file(agents.ActionKept)}, "kept", "kept"},
		{"a kept entry beside an unchanged one", agents.WriteResult{Files: []agents.FileResult{{Path: "a", Action: agents.ActionKept}, {Path: "b", Action: agents.ActionUnchanged}}}, "kept", "kept"},
		{"a kept entry beside a not-found one", agents.WriteResult{Files: []agents.FileResult{{Path: "a", Action: agents.ActionKept}, {Path: "b", Action: agents.ActionNotFound}}}, "kept", "kept"},
		{"a kept entry beside a removed one", agents.WriteResult{Files: []agents.FileResult{{Path: "a", Action: agents.ActionKept}, {Path: "b", Action: agents.ActionRemoved}}}, "configured", "removed"},
		{"an error and no change", agents.WriteResult{Files: file(agents.ActionUnchanged), Errors: failure}, "failed", "failed"},
		{"an error and nothing touched", agents.WriteResult{Errors: failure}, "failed", "failed"},
		{"an error after a created file", agents.WriteResult{Files: file(agents.ActionCreated), Errors: failure}, "partial", "failed"},
		{"an error after a removed file", agents.WriteResult{Files: file(agents.ActionRemoved), Errors: failure}, "partial", "partial"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := installStatus(tc.result); got != tc.wantInstall {
				t.Errorf("installStatus = %q, want %q", got, tc.wantInstall)
			}
			if got := uninstallStatus(tc.result); got != tc.wantUninstall {
				t.Errorf("uninstallStatus = %q, want %q", got, tc.wantUninstall)
			}
		})
	}
}

// TestBadCodexHome_OnlyAnExplicitCodexTargetFails: a CODEX_HOME that does not
// resolve must not fail install or uninstall for a user who did not name
// Codex. Plain install and uninstall skip Codex with one stderr line and
// succeed, and --target codex fails with the error and a "failed" headline.
func TestBadCodexHome_OnlyAnExplicitCodexTargetFails(t *testing.T) {
	fakeHome(t)
	t.Setenv("CODEX_HOME", "rel")

	for _, args := range [][]string{{"install"}, {"install", "--yes"}, {"uninstall", "--yes"}, {"install", "--target", "all"}} {
		out, errOut, err := execCmd(args...)
		if err != nil {
			t.Fatalf("%v: %v\n%s", args, err, out)
		}
		if got := strings.Count(errOut, "codex: skipped: CODEX_HOME"); got != 1 {
			t.Errorf("%v: stderr has %d skip lines for codex, want 1:\n%s", args, got, errOut)
		}
		if !strings.Contains(out, "Codex CLI: skipped (CODEX_HOME") || strings.Contains(out, "Codex CLI: configured") || strings.Contains(out, "Codex CLI: failed") {
			t.Errorf("%v: stdout has no skipped row for Codex:\n%s", args, out)
		}
		if strings.Contains(out, "note: the shared skill package keeps") {
			t.Errorf("%v: stdout has a skill note although Codex is not a requester:\n%s", args, out)
		}
	}

	for _, command := range []string{"install", "uninstall"} {
		out, _, err := execCmd(command, "--target", "codex")
		if err == nil || !strings.Contains(err.Error(), `CODEX_HOME "rel"`) {
			t.Errorf("%s --target codex error = %v, want one naming CODEX_HOME", command, err)
		}
		if !strings.Contains(out, "Codex CLI: failed") && !strings.Contains(out, "Codex CLI: partial") {
			t.Errorf("%s --target codex stdout = %q, want a failed or partial headline for Codex", command, out)
		}
	}
}

// TestUninstallSkippedRowNote_NamesTheTargetAndNeedsARecordedRequester: the
// note under a skipped row says that the shared skill package keeps the
// target as a requester, names the target and the command that fixes it, and
// appears only when the manifest records the target.
func TestUninstallSkippedRowNote_NamesTheTargetAndNeedsARecordedRequester(t *testing.T) {
	const note = "note: the shared skill package keeps codex as a requester until its detection is fixed and `codegraph uninstall --target codex` runs"

	fakeHome(t)
	t.Setenv("CODEX_HOME", "")
	if _, _, err := execCmd("install", "--target", "codex"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CODEX_HOME", "rel")
	out, _, err := execCmd("uninstall", "--yes")
	if err != nil {
		t.Fatalf("uninstall: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Codex CLI: skipped (CODEX_HOME") || !strings.Contains(out, note) {
		t.Errorf("stdout lacks the skipped row and its note:\n%s", out)
	}

	fakeHome(t)
	t.Setenv("CODEX_HOME", "rel")
	out, _, err = execCmd("uninstall", "--yes")
	if err != nil {
		t.Fatalf("uninstall: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Codex CLI: skipped (CODEX_HOME") || strings.Contains(out, "note: the shared skill package") {
		t.Errorf("stdout should have the skipped row and no skill note:\n%s", out)
	}
}

// TestUninstallClaudeLocal_SettingsOutsideProjectReadsKept: an uninstall whose
// only outcome is a settings.json kept outside the project reads "kept", not
// "not-configured", although it also reports files that were not found. The
// kept file and its note appear once, not once per step.
func TestUninstallClaudeLocal_SettingsOutsideProjectReadsKept(t *testing.T) {
	fakeHome(t)
	target := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(target, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	realpathtest.Symlink(t, target, filepath.Join(".claude", "settings.json"))

	out, _, err := execCmd("uninstall", "--target", "claude", "--location", "local")
	if err != nil {
		t.Fatalf("uninstall: %v\n%s", err, out)
	}
	if !strings.HasPrefix(out, "Claude Code: kept\n") {
		t.Errorf("headline is not kept:\n%s", out)
	}
	if n := strings.Count(out, "  kept: .claude/settings.json\n"); n != 1 {
		t.Errorf("the kept row appears %d times, want 1:\n%s", n, out)
	}
	if n := strings.Count(out, "note: .claude/settings.json is left as it is"); n != 1 {
		t.Errorf("the kept note appears %d times, want 1:\n%s", n, out)
	}
}

// TestUninstall_UserOwnedInstructionFilesReadNotConfigured: an agent that was
// never installed reads not-configured even when the user keeps their own
// CLAUDE.md, AGENTS.md or GEMINI.md, at global and at local scope, and an
// uninstall that follows a real one reads not-configured too.
func TestUninstall_UserOwnedInstructionFilesReadNotConfigured(t *testing.T) {
	scopes := []struct {
		location string
		files    func(home string) []string
	}{
		{"global", func(home string) []string {
			return []string{
				filepath.Join(home, ".claude", "CLAUDE.md"), filepath.Join(home, ".codex", "AGENTS.md"),
				filepath.Join(home, ".gemini", "GEMINI.md"), filepath.Join(home, ".config", "opencode", "AGENTS.md"),
			}
		}},
		{"local", func(string) []string {
			return []string{filepath.Join(".claude", "CLAUDE.md"), "AGENTS.md", "GEMINI.md"}
		}},
	}
	for _, scope := range scopes {
		t.Run(scope.location, func(t *testing.T) {
			home := fakeHome(t)
			for _, path := range scope.files(home) {
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte("# my own notes\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			out, _, err := execCmd("uninstall", "--target", "claude,codex,gemini,opencode", "--location", scope.location, "--yes")
			if err != nil {
				t.Fatalf("uninstall: %v\n%s", err, out)
			}
			for _, name := range []string{"Claude Code", "Codex CLI", "Gemini CLI", "opencode"} {
				if !strings.Contains(out, name+": not-configured\n") {
					t.Errorf("%s does not read not-configured:\n%s", name, out)
				}
			}
			if strings.Contains(out, "  kept: ") {
				t.Errorf("the report has a kept row for files that hold nothing of codegraph's:\n%s", out)
			}

			if _, _, err := execCmd("install", "--target", "claude", "--location", scope.location); err != nil {
				t.Fatal(err)
			}
			if out, _, err := execCmd("uninstall", "--target", "claude", "--location", scope.location, "--yes"); err != nil || !strings.Contains(out, "Claude Code: removed\n") {
				t.Fatalf("the uninstall after an install did not read removed: %v\n%s", err, out)
			}
			out, _, err = execCmd("uninstall", "--target", "claude", "--location", scope.location, "--yes")
			if err != nil || !strings.Contains(out, "Claude Code: not-configured\n") {
				t.Errorf("the second uninstall does not read not-configured: %v\n%s", err, out)
			}
		})
	}
}
