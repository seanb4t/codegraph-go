package cli

import (
	"errors"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/agents"
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
		if strings.Contains(out, "Codex CLI:") {
			t.Errorf("%v: stdout reports Codex although it was skipped:\n%s", args, out)
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
