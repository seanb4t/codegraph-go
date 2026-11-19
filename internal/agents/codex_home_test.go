package agents

import (
	"os"
	"path/filepath"
	"reflect"
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
// absolute path makes the config, instructions, hooks.json and guard path
// functions return an error that names it. Install and uninstall report
// that error and write nothing, not under the working directory, not in the
// directory the value would name relative to it, and not under the default
// home. Detect reports the error, and --target auto still selects Codex so
// the error reaches the user.
func TestCodexGlobal_RelativeCodexHomeIsRefused(t *testing.T) {
	home := fakeHome(t)
	cwd := t.TempDir()
	t.Chdir(cwd)
	t.Setenv("CODEX_HOME", "rel")
	untouched := filepath.Join(cwd, "rel", "config.toml")
	writeFile(t, untouched, "keep = true\n")
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

	if res := c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: PreToolNudgeOn}); !errorNaming(res, `"rel"`) {
		t.Errorf("Install errors = %v, want one naming %q", res.Errors, "rel")
	}
	if res := c.Uninstall(LocationGlobal); !errorNaming(res, `"rel"`) {
		t.Errorf("Uninstall errors = %v, want one naming %q", res.Errors, "rel")
	}
	if got := readFile(t, untouched); got != "keep = true\n" {
		t.Errorf("%s changed to %q", untouched, got)
	}
	if names := dirNames(t, cwd); !reflect.DeepEqual(names, []string{"rel"}) {
		t.Errorf("working directory holds %v, want only the rel directory seeded by the test", names)
	}
	if names := dirNames(t, filepath.Join(cwd, "rel")); !reflect.DeepEqual(names, []string{"config.toml"}) {
		t.Errorf("rel holds %v, want only the seeded config.toml", names)
	}
	assertMissing(t, filepath.Join(home, ".codex"))

	if d := c.Detect(LocationGlobal); d.Err == nil || !strings.Contains(d.Err.Error(), `"rel"`) || d.Installed {
		t.Errorf("Detect(global) = %+v, want an error naming %q and not installed", d, "rel")
	}
	targets, err := ResolveTargetFlag("auto", LocationGlobal)
	if err != nil {
		t.Fatal(err)
	}
	selected := false
	for _, target := range targets {
		selected = selected || target.ID() == Codex
	}
	if !selected {
		t.Errorf("--target auto selected %v, want Codex among them so its error is reported", targets)
	}
}

// dirNames returns the sorted entry names of dir.
func dirNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

// TestCodexLocal_RelativeCodexHomeDoesNotFailLocalInstall: a CODEX_HOME that
// is not absolute fails global operations only. A local install, with the
// nudge kept, on or off, succeeds, names $CODEX_HOME/config.toml in its trust
// note, and says that it did not check the global hooks setting. A local
// config.toml that sets the hooks decides first.
func TestCodexLocal_RelativeCodexHomeDoesNotFailLocalInstall(t *testing.T) {
	const bad = `CODEX_HOME "rel" is not an absolute path`
	install := func(t *testing.T, mode PreToolNudgeMode, execPath string) WriteResult {
		t.Helper()
		res := codexTarget{}.Install(LocationLocal, InstallOptions{ExecPath: execPath, PreToolNudge: mode})
		assertNoErrors(t, res)
		return res
	}
	prepare := func(t *testing.T) (hooksPath, guardPath string) {
		fakeHome(t)
		t.Chdir(t.TempDir())
		var err error
		if hooksPath, err = codexHooksJSONPath(LocationLocal); err != nil {
			t.Fatal(err)
		}
		if guardPath, err = codexPreToolGuardPath(LocationLocal); err != nil {
			t.Fatal(err)
		}
		return hooksPath, guardPath
	}

	t.Run("keep with nothing recorded", func(t *testing.T) {
		prepare(t)
		t.Setenv("CODEX_HOME", "rel")
		res := install(t, PreToolNudgeKeep, "/usr/local/bin/codegraph")
		if !noteNaming(res, "$CODEX_HOME/config.toml") {
			t.Errorf("notes = %q, want the trust note to name $CODEX_HOME/config.toml", res.Notes)
		}
	})

	t.Run("on", func(t *testing.T) {
		hooksPath, guardPath := prepare(t)
		t.Setenv("CODEX_HOME", "rel")
		res := install(t, PreToolNudgeOn, "/usr/local/bin/codegraph")
		if !fileExists(hooksPath) || !fileExists(guardPath) {
			t.Errorf("the nudge was not installed: hooks.json %v, guard %v", fileExists(hooksPath), fileExists(guardPath))
		}
		if !noteNaming(res, bad) {
			t.Errorf("notes = %q, want one naming %s", res.Notes, bad)
		}
	})

	t.Run("keep refreshes a recorded nudge", func(t *testing.T) {
		_, guardPath := prepare(t)
		t.Setenv("CODEX_HOME", "")
		install(t, PreToolNudgeOn, "/usr/local/bin/codegraph")
		t.Setenv("CODEX_HOME", "rel")
		res := install(t, PreToolNudgeKeep, "/usr/local/bin/codegraph-v2")
		if got := readFile(t, guardPath); !strings.Contains(got, "codegraph-v2") {
			t.Errorf("Keep did not refresh the guard:\n%s", got)
		}
		if !noteNaming(res, bad) {
			t.Errorf("notes = %q, want one naming %s", res.Notes, bad)
		}
	})

	t.Run("off", func(t *testing.T) {
		hooksPath, guardPath := prepare(t)
		t.Setenv("CODEX_HOME", "")
		install(t, PreToolNudgeOn, "/usr/local/bin/codegraph")
		t.Setenv("CODEX_HOME", "rel")
		install(t, PreToolNudgeOff, "/usr/local/bin/codegraph")
		assertMissing(t, hooksPath)
		assertMissing(t, guardPath)
	})

	t.Run("a local config decides before the global one", func(t *testing.T) {
		hooksPath, _ := prepare(t)
		t.Setenv("CODEX_HOME", "rel")
		writeFile(t, filepath.Join(".codex", "config.toml"), "[features]\nhooks = false\n")
		res := install(t, PreToolNudgeOn, "/usr/local/bin/codegraph")
		assertMissing(t, hooksPath)
		if !noteNaming(res, "explicitly disabled") || noteNaming(res, bad) {
			t.Errorf("notes = %q, want the disabled-hooks note and no mention of CODEX_HOME", res.Notes)
		}
	})
}

// TestCodexGlobal_HandlerPointingIntoDefaultHomeStaysOwned: with CODEX_HOME
// set, a PreToolUse handler in $CODEX_HOME/hooks.json whose command is the
// guard under ~/.codex (the spelling used when CODEX_HOME is unset) is
// codegraph's own. A plain install replaces it, an opt-in install adds no
// second one, and uninstall removes it and leaves a user group untouched.
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
	const foreign = "echo foreign"
	seed := func() {
		writeHooksFile(t, hooksPath, "PreToolUse", hookGroup("^Read$", nil, foreign), hookGroup("^Bash$", nil, legacy))
	}

	for name, mode := range map[string]PreToolNudgeMode{"keep": PreToolNudgeKeep, "on": PreToolNudgeOn} {
		seed()
		res := codexTarget{}.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: mode})
		assertNoErrors(t, res)
		if got := settingsHandlerCommands(t, hooksPath, "PreToolUse"); !reflect.DeepEqual(got, []string{foreign, own}) {
			t.Errorf("%s: PreToolUse handlers = %q, want the foreign one and codegraph's %q", name, got, own)
		}
		if !fileExists(filepath.Join(codexHome, "hooks", "codegraph-pretooluse.sh")) {
			t.Errorf("%s install wrote no guard", name)
		}
	}

	seed()
	assertNoErrors(t, codexTarget{}.Uninstall(LocationGlobal))
	if got := settingsHandlerCommands(t, hooksPath, "PreToolUse"); !reflect.DeepEqual(got, []string{foreign}) {
		t.Errorf("PreToolUse handlers after uninstall = %q, want only %q", got, foreign)
	}
}

// TestCodexLocal_LegacySpellingIsNotOwned: only the global scope owns the
// ~/.codex guard spelling.
func TestCodexLocal_LegacySpellingIsNotOwned(t *testing.T) {
	fakeHome(t)
	t.Setenv("CODEX_HOME", filepath.Join(t.TempDir(), "elsewhere"))
	_, owned, err := codexPreToolUseBlocks(LocationLocal)
	if err != nil {
		t.Fatal(err)
	}
	if len(owned) != 1 {
		t.Errorf("local owned commands = %q, want only the local command", owned)
	}
}

// TestCodexKeep_UnreadableHooksFileSaysSo: a Keep install over a hooks.json
// that cannot be parsed changes nothing and says it did not refresh the nudge.
func TestCodexKeep_UnreadableHooksFileSaysSo(t *testing.T) {
	fakeHome(t)
	t.Chdir(t.TempDir())
	hooksPath, err := codexHooksJSONPath(LocationLocal)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, hooksPath, "{not json")

	res := codexTarget{}.Install(LocationLocal, InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: PreToolNudgeKeep})
	assertNoErrors(t, res)
	if !noteNaming(res, "could not be read") {
		t.Errorf("notes = %q, want one saying hooks.json could not be read", res.Notes)
	}
	if got := readFile(t, hooksPath); got != "{not json" {
		t.Errorf("Keep changed hooks.json to %q", got)
	}
}
