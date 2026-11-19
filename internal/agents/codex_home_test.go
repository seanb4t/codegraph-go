package agents

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/realpath/realpathtest"
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
	t.Chdir(t.TempDir())
	codexHome := useCodexHome(t)
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
	codexHome := useCodexHome(t)

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
}

// TestResolveTargets_BadCodexHomeSkipsCodexUnlessAsked: a CODEX_HOME that
// does not resolve must not reach a user who did not ask for Codex.
// --target auto with no agent installed falls back to Claude, and with only
// Claude installed selects Claude. Neither selects Codex, and both list it as
// skipped with the reason. --target all skips it the same way. An explicit
// --target codex still selects it, so its install reports the error.
func TestResolveTargets_BadCodexHomeSkipsCodexUnlessAsked(t *testing.T) {
	home := fakeHome(t)
	t.Chdir(t.TempDir())
	t.Setenv("CODEX_HOME", "rel")
	ids := func(targets []AgentTarget) []TargetID {
		var out []TargetID
		for _, target := range targets {
			out = append(out, target.ID())
		}
		return out
	}
	checkSkipped := func(label string, skipped []SkippedTarget) {
		t.Helper()
		if len(skipped) != 1 || skipped[0].ID != Codex || skipped[0].Err == nil || !strings.Contains(skipped[0].Err.Error(), `"rel"`) {
			t.Errorf("%s: skipped = %v, want only codex with an error naming %q", label, skipped, "rel")
		}
	}

	targets, skipped, err := ResolveTargets("auto", LocationGlobal)
	if err != nil || !reflect.DeepEqual(ids(targets), []TargetID{Claude}) {
		t.Errorf("auto with no agent installed = %v, %v, want [claude]", ids(targets), err)
	}
	checkSkipped("auto with no agent", skipped)

	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	targets, skipped, err = ResolveTargets("auto", LocationGlobal)
	if err != nil || !reflect.DeepEqual(ids(targets), []TargetID{Claude}) {
		t.Errorf("auto with Claude installed = %v, %v, want [claude]", ids(targets), err)
	}
	checkSkipped("auto with Claude", skipped)

	targets, skipped, err = ResolveTargets("all", LocationGlobal)
	if err != nil || slices.Contains(ids(targets), Codex) || len(ids(targets)) != len(AllTargetIDs())-1 {
		t.Errorf("all = %v, %v, want every target but codex", ids(targets), err)
	}
	checkSkipped("all", skipped)

	targets, skipped, err = ResolveTargets("codex", LocationGlobal)
	if err != nil || !reflect.DeepEqual(ids(targets), []TargetID{Codex}) || len(skipped) != 0 {
		t.Errorf("explicit codex = %v, skipped %v, %v, want [codex] and nothing skipped", ids(targets), skipped, err)
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
	const notChecked = "was not checked for a disabled-hooks setting"
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
		const want = `in $CODEX_HOME/config.toml, which does not resolve because CODEX_HOME "rel" is not an absolute path (`
		if !noteNaming(res, want) {
			t.Errorf("notes = %q, want the trust note to say %s", res.Notes, want)
		}
	})

	t.Run("on", func(t *testing.T) {
		hooksPath, guardPath := prepare(t)
		t.Setenv("CODEX_HOME", "rel")
		res := install(t, PreToolNudgeOn, "/usr/local/bin/codegraph")
		if !fileExists(hooksPath) || !fileExists(guardPath) {
			t.Errorf("the nudge was not installed: hooks.json %v, guard %v", fileExists(hooksPath), fileExists(guardPath))
		}
		if !noteNaming(res, notChecked) {
			t.Errorf("notes = %q, want one saying the global setting was not checked", res.Notes)
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
		if !noteNaming(res, notChecked) {
			t.Errorf("notes = %q, want one saying the global setting was not checked", res.Notes)
		}
	})

	t.Run("off", func(t *testing.T) {
		hooksPath, guardPath := prepare(t)
		t.Setenv("CODEX_HOME", "")
		install(t, PreToolNudgeOn, "/usr/local/bin/codegraph")
		if !fileExists(hooksPath) || !fileExists(guardPath) {
			t.Fatalf("precondition: the nudge is not installed: hooks.json %v, guard %v", fileExists(hooksPath), fileExists(guardPath))
		}
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
		if !noteNaming(res, "explicitly disabled") || noteNaming(res, notChecked) {
			t.Errorf("notes = %q, want the disabled-hooks note and none about the global setting", res.Notes)
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
	codexHome := useCodexHome(t)
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
	useCodexHome(t)
	_, owned, err := codexPreToolUseBlocks(LocationLocal)
	if err != nil {
		t.Fatal(err)
	}
	local, err := codexPreToolHookCommand(LocationLocal)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(owned, []string{local}) {
		t.Errorf("local owned commands = %q, want only the local command %q", owned, local)
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
	if !noteNaming(res, "nudge was not refreshed") || !noteNaming(res, hooksPath) {
		t.Errorf("notes = %q, want one saying the nudge was not refreshed and naming %s", res.Notes, hooksPath)
	}
	if got := readFile(t, hooksPath); got != "{not json" {
		t.Errorf("Keep changed hooks.json to %q", got)
	}
}

// TestCodexHomeDir_ResolvesTheWayCodexDoes: a set CODEX_HOME resolves with
// every symlink followed, so ".." after a link names the link target's
// parent, and it must be an existing directory. A missing path, a file and a
// path that only exists as text are errors that name the value, and install
// creates nothing.
func TestCodexHomeDir_ResolvesTheWayCodexDoes(t *testing.T) {
	home := fakeHome(t)
	t.Chdir(t.TempDir())
	base := realpathtest.TempDir(t)
	for _, dir := range []string{"other/sub", "other/codex"} {
		if err := os.MkdirAll(filepath.Join(base, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(filepath.Join(base, "other", "sub"), filepath.Join(base, "link")); err != nil {
		t.Fatal(err)
	}

	// Not filepath.Join: it would clean the ".." as text.
	t.Setenv("CODEX_HOME", base+string(filepath.Separator)+"link"+string(filepath.Separator)+".."+string(filepath.Separator)+"codex")
	got, err := codexHomeDir()
	if want := filepath.Join(base, "other", "codex"); err != nil || got != want {
		t.Fatalf("codexHomeDir() = %q, %v, want %q: the link's parent, not the text-cleaned %s", got, err, want, filepath.Join(base, "codex"))
	}

	if err := os.WriteFile(filepath.Join(base, "file"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	for name, value := range map[string]string{
		"a missing directory": filepath.Join(base, "missing"),
		"a regular file":      filepath.Join(base, "file"),
		"a missing parent":    filepath.Join(base, "missing", "codex"),
	} {
		t.Setenv("CODEX_HOME", value)
		_, err := codexHomeDir()
		if err == nil || !strings.Contains(err.Error(), value) {
			t.Errorf("%s: codexHomeDir() error = %v, want one naming %q", name, err, value)
		}
		if name != "a regular file" && !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s: errors.Is(err, fs.ErrNotExist) = false for %v", name, err)
		}
		res := codexTarget{}.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: PreToolNudgeOn})
		if !errorNaming(res, value) {
			t.Errorf("%s: Install errors = %v, want one naming %q", name, res.Errors, value)
		}
	}
	assertMissing(t, filepath.Join(base, "missing"))
	assertMissing(t, filepath.Join(home, ".codex"))
}

// TestCodexGlobal_UninstallDropsTheSkillRequesterWithABadCodexHome: the
// shared skill directory does not depend on CODEX_HOME, so an uninstall with a
// CODEX_HOME that does not resolve still removes Codex from the shared skill
// manifest, besides reporting the error.
func TestCodexGlobal_UninstallDropsTheSkillRequesterWithABadCodexHome(t *testing.T) {
	fakeHome(t)
	t.Chdir(t.TempDir())
	opts := InstallOptions{ExecPath: "/usr/local/bin/codegraph"}
	opencodeTarget{}.Install(LocationGlobal, opts)
	codexTarget{}.Install(LocationGlobal, opts)
	dir, err := sharedSkillDirPath(LocationGlobal)
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(dir, ".codegraph-manifest.json")
	if m, present, err := readManifest(manifestPath); err != nil || !present || !targetSetEqual(m.Targets, []TargetID{Codex, Opencode}) {
		t.Fatalf("precondition: manifest = %+v, present %v, err %v, want codex and opencode", m, present, err)
	}

	t.Setenv("CODEX_HOME", "rel")
	res := codexTarget{}.Uninstall(LocationGlobal)
	if !errorNaming(res, `"rel"`) {
		t.Errorf("uninstall errors = %v, want one naming %q", res.Errors, "rel")
	}
	m, present, err := readManifest(manifestPath)
	if err != nil || !present || !targetSetEqual(m.Targets, []TargetID{Opencode}) {
		t.Errorf("manifest after uninstall = %+v, present %v, err %v, want only opencode", m, present, err)
	}
}

// TestCodexGlobal_RefusesAGuardPathWithAQuoteOrBackslash: a global guard path
// with a single quote or a backslash is an error that names the path and
// the remedy, and nothing is written, because Codex may run the hook under a
// shell that does not read the quoting as one word.
func TestCodexGlobal_RefusesAGuardPathWithAQuoteOrBackslash(t *testing.T) {
	for _, name := range []string{"it's", `back\slash`} {
		t.Run(name, func(t *testing.T) {
			fakeHome(t)
			t.Chdir(t.TempDir())
			codexHome := filepath.Join(realpathtest.TempDir(t), name)
			if err := os.Mkdir(codexHome, 0o755); err != nil {
				t.Fatal(err)
			}
			t.Setenv("CODEX_HOME", codexHome)

			res := codexTarget{}.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: PreToolNudgeOn})
			if !errorNaming(res, codexHome, "single quote, a backslash", "set CODEX_HOME", "--location local") {
				t.Errorf("errors = %v, want one naming %s and the remedy", res.Errors, codexHome)
			}
			assertMissing(t, filepath.Join(codexHome, "hooks"))
			assertMissing(t, filepath.Join(codexHome, "hooks.json"))
		})
	}
}

// TestCodexLocalTrustNote_DefaultsToTildeConfig: when the global config.toml
// is the default one, the trust note names ~/.codex/config.toml and not the
// absolute path, whether CODEX_HOME is unset or names the default directory.
func TestCodexLocalTrustNote_DefaultsToTildeConfig(t *testing.T) {
	for _, name := range []string{"unset", "the default directory"} {
		t.Run(name, func(t *testing.T) {
			home := realHome(t)
			t.Chdir(t.TempDir())
			if name != "unset" {
				dir := filepath.Join(home, ".codex")
				if err := os.Mkdir(dir, 0o755); err != nil {
					t.Fatal(err)
				}
				t.Setenv("CODEX_HOME", dir)
			}

			res := codexTarget{}.Install(LocationLocal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
			assertNoErrors(t, res)
			for _, n := range res.Notes {
				if strings.Contains(n, `trust_level = "trusted"`) {
					if !strings.Contains(n, "in ~/.codex/config.toml (") || strings.Contains(n, filepath.Join(home, ".codex")) {
						t.Fatalf("trust note = %q, want it to name ~/.codex/config.toml and not the absolute path", n)
					}
					return
				}
			}
			t.Fatalf("no trust note in %v", res.Notes)
		})
	}
}
