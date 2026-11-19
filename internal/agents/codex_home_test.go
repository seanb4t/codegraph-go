package agents

import (
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
