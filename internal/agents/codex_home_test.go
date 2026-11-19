package agents

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// codexGlobalFiles lists the global files a Codex install with the
// PreToolUse nudge on writes under dir.
func codexGlobalFiles(dir string) []string {
	return []string{
		filepath.Join(dir, "config.toml"),
		filepath.Join(dir, "AGENTS.md"),
		filepath.Join(dir, "hooks.json"),
		filepath.Join(dir, "hooks", "codegraph-pretooluse.sh"),
	}
}

// TestCodexGlobal_HonorsCodexHome: with CODEX_HOME set, a global install
// writes every Codex file under it, status reads it, the read-only skill
// root follows it, and uninstall removes what install wrote there.
func TestCodexGlobal_HonorsCodexHome(t *testing.T) {
	home := fakeHome(t)
	codexHome := filepath.Join(t.TempDir(), "elsewhere")
	t.Setenv("CODEX_HOME", codexHome)
	c := codexTarget{}

	res := c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: PreToolNudgeOn})
	assertNoErrors(t, res)

	for _, p := range codexGlobalFiles(codexHome) {
		if !fileExists(p) {
			t.Errorf("install wrote nothing at %s", p)
		}
	}
	assertMissing(t, filepath.Join(home, ".codex"))

	if got := c.Detect(LocationGlobal); !got.AlreadyConfigured || got.ConfigPath != filepath.Join(codexHome, "config.toml") {
		t.Errorf("Detect(global) = %+v, want configured at %s", got, filepath.Join(codexHome, "config.toml"))
	}

	roots, err := c.Capabilities().ReadOnlySkillDirs(LocationGlobal)
	if err != nil {
		t.Fatalf("ReadOnlySkillDirs(global): %v", err)
	}
	if want := filepath.Join(codexHome, "skills", "codegraph"); !containsPath(roots, want) {
		t.Errorf("ReadOnlySkillDirs(global) = %v, want to contain %q", roots, want)
	}

	assertNoErrors(t, c.Uninstall(LocationGlobal))
	for _, p := range codexGlobalFiles(codexHome) {
		assertMissing(t, p)
	}
}

// TestCodexGlobal_EmptyCodexHomeFallsBackToHome: an empty CODEX_HOME is the
// same as an unset one.
func TestCodexGlobal_EmptyCodexHomeFallsBackToHome(t *testing.T) {
	home := fakeHome(t)
	t.Chdir(t.TempDir())
	t.Setenv("CODEX_HOME", "")

	res := codexTarget{}.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: PreToolNudgeOn})
	assertNoErrors(t, res)

	for _, p := range codexGlobalFiles(filepath.Join(home, ".codex")) {
		if !fileExists(p) {
			t.Errorf("install wrote nothing at %s", p)
		}
	}
}

// TestCodexLocalTrustNote_NamesCodexHomeConfig: the trust note points at the
// config.toml Codex actually reads.
func TestCodexLocalTrustNote_NamesCodexHomeConfig(t *testing.T) {
	fakeHome(t)
	t.Chdir(t.TempDir())
	codexHome := filepath.Join(t.TempDir(), "elsewhere")
	t.Setenv("CODEX_HOME", codexHome)

	res := codexTarget{}.Install(LocationLocal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
	assertNoErrors(t, res)

	want := filepath.Join(codexHome, "config.toml")
	for _, n := range res.Notes {
		if strings.Contains(n, `trust_level = "trusted"`) {
			if !strings.Contains(n, want) || strings.Contains(n, "~/.codex/config.toml") {
				t.Fatalf("trust note = %q, want it to name %s", n, want)
			}
			return
		}
	}
	t.Fatalf("no trust note in %v", res.Notes)
}

// TestCodexGlobal_RelativeCodexHomeIsRefused: a CODEX_HOME that is not an
// absolute path fails every global path with an error naming it, and
// install writes nothing under the working directory or the default home.
func TestCodexGlobal_RelativeCodexHomeIsRefused(t *testing.T) {
	home := fakeHome(t)
	cwd := t.TempDir()
	t.Chdir(cwd)
	t.Setenv("CODEX_HOME", "rel")
	c := codexTarget{}

	for name, resolve := range map[string]func() (string, error){
		"config":       func() (string, error) { return codexConfigPath(LocationGlobal) },
		"instructions": func() (string, error) { return codexInstructionsPath(LocationGlobal) },
		"hooks.json":   func() (string, error) { return codexHooksJSONPath(LocationGlobal) },
		"guard":        func() (string, error) { return codexPreToolGuardPath(LocationGlobal) },
	} {
		if _, err := resolve(); err == nil || !strings.Contains(err.Error(), `"rel"`) {
			t.Errorf("%s path error = %v, want one naming %q", name, err, "rel")
		}
	}

	res := c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: PreToolNudgeOn})
	if !errorNaming(res, `"rel"`) {
		t.Errorf("Install errors = %v, want one naming %q", res.Errors, "rel")
	}
	if entries, err := os.ReadDir(cwd); err != nil || len(entries) != 0 {
		t.Errorf("working directory holds %v (err %v), want nothing written there", entries, err)
	}
	assertMissing(t, filepath.Join(home, ".codex"))
}

// TestCodexLocalTrustNote_DefaultsToTildeConfig: without CODEX_HOME the
// trust note names ~/.codex/config.toml, not an expanded home path.
func TestCodexLocalTrustNote_DefaultsToTildeConfig(t *testing.T) {
	fakeHome(t)
	t.Chdir(t.TempDir())
	t.Setenv("CODEX_HOME", "")

	res := codexTarget{}.Install(LocationLocal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
	assertNoErrors(t, res)
	for _, n := range res.Notes {
		if strings.Contains(n, `trust_level = "trusted"`) {
			if !strings.Contains(n, "in ~/.codex/config.toml (") {
				t.Fatalf("trust note = %q, want it to name ~/.codex/config.toml", n)
			}
			return
		}
	}
	t.Fatalf("no trust note in %v", res.Notes)
}

// TestCodexGlobal_HandlerPointingIntoDefaultHomeStaysOwned: with CODEX_HOME
// set, a PreToolUse handler whose command is the guard under ~/.codex (the
// spelling used when CODEX_HOME is unset) is still codegraph's own. A plain
// install refreshes it, an opt-in install adds no second one, and uninstall
// removes it.
func TestCodexGlobal_HandlerPointingIntoDefaultHomeStaysOwned(t *testing.T) {
	home := fakeHome(t)
	t.Chdir(t.TempDir())
	codexHome := filepath.Join(t.TempDir(), "elsewhere")
	t.Setenv("CODEX_HOME", codexHome)
	hooksPath := filepath.Join(codexHome, "hooks.json")
	own, err := codexPreToolHookCommand(LocationGlobal)
	if err != nil {
		t.Fatal(err)
	}
	legacy := shellSingleQuote(filepath.Join(home, ".codex", "hooks", "codegraph-pretooluse.sh"))
	if legacy == own {
		t.Fatalf("precondition: legacy and own spellings are both %q", own)
	}
	seed := func() {
		data, err := json.Marshal(map[string]any{"hooks": map[string]any{"PreToolUse": []any{
			map[string]any{"matcher": "^Bash$", "hooks": []any{map[string]any{"type": "command", "command": legacy, "timeout": 5}}},
		}}})
		if err != nil {
			t.Fatal(err)
		}
		writeFile(t, hooksPath, string(data))
	}

	for name, mode := range map[string]PreToolNudgeMode{"keep": PreToolNudgeKeep, "on": PreToolNudgeOn} {
		seed()
		res := codexTarget{}.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: mode})
		assertNoErrors(t, res)
		if got := settingsHandlerCommands(t, hooksPath, "PreToolUse"); len(got) != 1 || got[0] != own {
			t.Errorf("%s: PreToolUse handlers = %q, want only %q", name, got, own)
		}
		if !fileExists(filepath.Join(codexHome, "hooks", "codegraph-pretooluse.sh")) {
			t.Errorf("%s install wrote no guard", name)
		}
	}

	seed()
	assertNoErrors(t, codexTarget{}.Uninstall(LocationGlobal))
	assertMissing(t, hooksPath)
}
