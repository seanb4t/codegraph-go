package agents

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/realpath/realpathtest"
)

// seedClaudeLocalHooks writes the Claude hook script and guard in the
// project and links .claude/settings.json to a file outside it that holds both
// registrations. It returns the outside settings file, the script and the
// guard.
func seedClaudeLocalHooks(t *testing.T) (outsideSettings, script, guard string) {
	t.Helper()
	_, outside := projectAndOutside(t)
	outsideSettings = filepath.Join(outside, "settings.json")
	sessionCmd, err := claudeHookCommand(LocationLocal)
	if err != nil {
		t.Fatal(err)
	}
	preToolCmd, err := claudePreToolHookCommand(LocationLocal)
	if err != nil {
		t.Fatal(err)
	}
	data := `{"hooks":{"SessionStart":[{"matcher":"startup","hooks":[{"type":"command","command":` + jsonString(t, sessionCmd) + `}]}],` +
		`"PreToolUse":[{"matcher":"Grep","hooks":[{"type":"command","command":` + jsonString(t, preToolCmd) + `}]}]}}`
	writeFile(t, outsideSettings, data)
	realpathtest.Symlink(t, outsideSettings, filepath.Join(".claude", "settings.json"))
	script = filepath.Join(".claude", "hooks", "session-nudge.sh")
	guard = filepath.Join(".claude", "hooks", "pretooluse-nudge.sh")
	writeFile(t, script, "#!/bin/sh\n")
	writeFile(t, guard, "#!/bin/sh\n")
	return outsideSettings, script, guard
}

// jsonString quotes s as a JSON string.
func jsonString(t *testing.T, s string) string {
	t.Helper()
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// TestClaudeUninstall_KeepsScriptAndGuardWhenSettingsIsOutsideProject: an
// uninstall keeps a local settings.json whose real location is outside the
// project, so the registrations in it remain, and it keeps the hook script
// and the guard that those registrations point at, with a note for each.
func TestClaudeUninstall_KeepsScriptAndGuardWhenSettingsIsOutsideProject(t *testing.T) {
	outsideSettings, script, guard := seedClaudeLocalHooks(t)
	before := readFile(t, outsideSettings)

	res := claudeTarget{}.Uninstall(LocationLocal)
	if !fileExists(script) || !fileExists(guard) {
		t.Errorf("the script (%v) or the guard (%v) was removed although their registrations remain", fileExists(script), fileExists(guard))
	}
	if readFile(t, outsideSettings) != before {
		t.Errorf("the outside settings file changed")
	}
	for _, kept := range []string{script, guard} {
		if !noteNaming(res, kept+" is left in place because its registration in") {
			t.Errorf("notes = %q, want one for %s", res.Notes, kept)
		}
	}
}

// TestClaudeUninstall_KeepsScriptWhenSessionStartRemovalFails: the SessionStart
// registration comes out before the script, and the script stays when the
// registration could not be removed.
func TestClaudeUninstall_KeepsScriptWhenSessionStartRemovalFails(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("a read-only directory does not stop root")
	}
	fakeHome(t)
	t.Chdir(t.TempDir())
	settings, err := claudeSettingsPath(LocationLocal)
	if err != nil {
		t.Fatal(err)
	}
	cmd, err := claudeHookCommand(LocationLocal)
	if err != nil {
		t.Fatal(err)
	}
	writeHooksFile(t, settings, "SessionStart", hookGroup("startup", nil, cmd))
	script := filepath.Join(".claude", "hooks", "session-nudge.sh")
	writeFile(t, script, "#!/bin/sh\n")
	dir := filepath.Dir(settings)
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	res := claudeTarget{}.Uninstall(LocationLocal)
	if !errorNaming(res, settings) {
		t.Errorf("errors = %v, want one naming %s", res.Errors, settings)
	}
	if !fileExists(script) {
		t.Errorf("the script was removed although the registration could not be")
	}
}

// TestCodexUninstall_KeepsGuardWhenHooksIsOutsideProject: an uninstall keeps a
// local hooks.json whose real location is outside the project and the guard
// its registration points at, with a note.
func TestCodexUninstall_KeepsGuardWhenHooksIsOutsideProject(t *testing.T) {
	_, outside := projectAndOutside(t)
	ownCmd, err := codexPreToolHookCommand(LocationLocal)
	if err != nil {
		t.Fatal(err)
	}
	outsideHooks := filepath.Join(outside, "hooks.json")
	writeHooksFile(t, outsideHooks, "PreToolUse", hookGroup("^Bash$", nil, ownCmd))
	before := readFile(t, outsideHooks)
	realpathtest.Symlink(t, outsideHooks, filepath.Join(".codex", "hooks.json"))
	guard := filepath.Join(".codex", "hooks", "codegraph-pretooluse.sh")
	writeFile(t, guard, "#!/bin/sh\n")

	res := codexTarget{}.Uninstall(LocationLocal)
	if !fileExists(guard) {
		t.Errorf("the guard was removed although its registration remains")
	}
	if readFile(t, outsideHooks) != before {
		t.Errorf("the outside hooks file changed")
	}
	if !noteNaming(res, guard+" is left in place because its registration in") {
		t.Errorf("notes = %q, want one for the guard", res.Notes)
	}
}

// TestInstall_SettingsOutsideProjectIsOneErrorAndWritesNoScriptOrGuard: a
// local settings.json (Claude) or hooks.json (Codex) whose real location is
// outside the project is one error that names the file and gives the way
// forward, and the install writes no script, guard or registration.
func TestInstall_SettingsOutsideProjectIsOneErrorAndWritesNoScriptOrGuard(t *testing.T) {
	t.Run("claude", func(t *testing.T) {
		_, outside := projectAndOutside(t)
		outsideSettings := filepath.Join(outside, "settings.json")
		writeFile(t, outsideSettings, "{}\n")
		link := filepath.Join(".claude", "settings.json")
		realpathtest.Symlink(t, outsideSettings, link)

		res := claudeTarget{}.Install(LocationLocal, InstallOptions{ExecPath: "/usr/local/bin/codegraph", AutoAllow: true, PreToolNudge: PreToolNudgeOn})
		if n := countErrorsNaming(res, link); n != 1 || !errorNaming(res, link, "outside the project", "--location global") {
			t.Errorf("errors = %v, want exactly one naming %s with the way forward", res.Errors, link)
		}
		if readFile(t, outsideSettings) != "{}\n" {
			t.Errorf("the outside settings file changed")
		}
		assertMissing(t, filepath.Join(".claude", "hooks", "session-nudge.sh"))
		assertMissing(t, filepath.Join(".claude", "hooks", "pretooluse-nudge.sh"))
	})
	t.Run("codex", func(t *testing.T) {
		_, outside := projectAndOutside(t)
		outsideHooks := filepath.Join(outside, "hooks.json")
		writeFile(t, outsideHooks, "{}\n")
		link := filepath.Join(".codex", "hooks.json")
		realpathtest.Symlink(t, outsideHooks, link)

		res := codexTarget{}.Install(LocationLocal, InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: PreToolNudgeOn})
		if n := countErrorsNaming(res, link); n != 1 || !errorNaming(res, link, "outside the project", "--location global") {
			t.Errorf("errors = %v, want exactly one naming %s with the way forward", res.Errors, link)
		}
		if readFile(t, outsideHooks) != "{}\n" {
			t.Errorf("the outside hooks file changed")
		}
		assertMissing(t, filepath.Join(".codex", "hooks", "codegraph-pretooluse.sh"))
	})
}

// countErrorsNaming counts the errors of result that mention part.
func countErrorsNaming(result WriteResult, part string) int {
	n := 0
	for _, err := range result.Errors {
		if strings.Contains(err.Error(), part) {
			n++
		}
	}
	return n
}

// TestClaudeInstall_NudgeOffIsGatedOnThePreflight: an install that turns the
// nudge off over a settings.json that fails the preflight reports the error
// and leaves the guard in place, because the registration may still be there.
func TestClaudeInstall_NudgeOffIsGatedOnThePreflight(t *testing.T) {
	fakeHome(t)
	t.Chdir(t.TempDir())
	settings, err := claudeSettingsPath(LocationLocal)
	if err != nil {
		t.Fatal(err)
	}
	guard, err := claudePreToolGuardPath(LocationLocal)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, settings, "{not json")
	writeFile(t, guard, "#!/bin/sh\n")

	res := claudeTarget{}.Install(LocationLocal, InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: PreToolNudgeOff})
	if !errorNaming(res, settings) {
		t.Errorf("errors = %v, want one naming %s", res.Errors, settings)
	}
	if !fileExists(guard) {
		t.Errorf("the guard was removed although settings.json could not be read")
	}
	if got := readFile(t, settings); got != "{not json" {
		t.Errorf("settings.json changed to %q", got)
	}
}

// TestCheckCodexGuardPath: a global guard path with a single quote, a
// backslash, a control character or an invalid UTF-8 byte is refused with the
// path and the remedy, and an ordinary path with a space is not.
func TestCheckCodexGuardPath(t *testing.T) {
	for name, path := range map[string]string{
		"single quote":  "/home/it's/.codex/hooks/g.sh",
		"backslash":     `/home/a\b/.codex/hooks/g.sh`,
		"control":       "/home/a\x01b/.codex/hooks/g.sh",
		"newline":       "/home/a\nb/.codex/hooks/g.sh",
		"invalid UTF-8": "/home/a\xffb/.codex/hooks/g.sh",
	} {
		err := checkCodexGuardPath(path)
		if err == nil || !strings.Contains(err.Error(), path) || !strings.Contains(err.Error(), "set CODEX_HOME") || !strings.Contains(err.Error(), "--location local") {
			t.Errorf("%s: error = %v, want one naming the path and the remedy", name, err)
		}
	}
	if err := checkCodexGuardPath("/home/a b/.codex/hooks/g.sh"); err != nil {
		t.Errorf("a path with a space: %v", err)
	}
}
