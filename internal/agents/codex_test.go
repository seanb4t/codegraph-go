package agents

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestCodex_ID(t *testing.T) {
	c := codexTarget{}
	if c.ID() != Codex {
		t.Fatalf("ID() = %v, want %v", c.ID(), Codex)
	}
}

// TestCodex_SupportsLocation_GlobalAndLocal (D-09) replaces
// TestCodex_SupportsLocation_GlobalOnly: the scope flip means Codex now
// supports both scopes through the capability table.
func TestCodex_SupportsLocation_GlobalAndLocal(t *testing.T) {
	c := codexTarget{}
	if !c.SupportsLocation(LocationGlobal) {
		t.Fatalf("codex should support global")
	}
	if !c.SupportsLocation(LocationLocal) {
		t.Fatalf("codex should support local (D-09 scope flip)")
	}
}

// TestCodex_Install_Local_WritesConfigInstructionsAndSkill (D-09, D-14)
// replaces TestCodex_Install_Local_IsUnsupportedNoWrite: a local install
// writes .codex/config.toml (the codegraph table), AGENTS.md (the marker
// block), and the shared skill package — nothing under the fake HOME.
func TestCodex_Install_Local_WritesConfigInstructionsAndSkill(t *testing.T) {
	home := fakeHome(t)
	dir := t.TempDir()
	t.Chdir(dir)
	c := codexTarget{}

	result := c.Install(LocationLocal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
	if len(result.Errors) != 0 {
		t.Fatalf("Install(local) returned errors: %v", result.Errors)
	}

	configPath := filepath.Join(dir, ".codex", "config.toml")
	got := readFile(t, configPath)
	if !strings.Contains(got, "[mcp_servers.codegraph]") ||
		!strings.Contains(got, `command = "/usr/local/bin/codegraph"`) ||
		!strings.Contains(got, `args = ["serve", "--mcp"]`) {
		t.Fatalf("unexpected local config.toml content: %s", got)
	}

	instrPath := filepath.Join(dir, "AGENTS.md")
	instr := readFile(t, instrPath)
	if !strings.Contains(instr, codegraphSectionStart) {
		t.Fatalf("unexpected local AGENTS.md content: %s", instr)
	}

	skillPath := filepath.Join(dir, ".agents", "skills", "codegraph", "SKILL.md")
	if !fileExists(skillPath) {
		t.Fatalf("expected shared skill SKILL.md at %s", skillPath)
	}
	manifestPath := filepath.Join(dir, ".agents", "skills", "codegraph", ".codegraph-manifest.json")
	if !fileExists(manifestPath) {
		t.Fatalf("expected shared skill manifest at %s", manifestPath)
	}

	if fileExists(filepath.Join(home, ".codex")) {
		t.Fatalf("local install must not write anything under the fake HOME (%s)", home)
	}
	if fileExists(filepath.Join(home, ".agents")) {
		t.Fatalf("local install must not write anything under the fake HOME (%s)", home)
	}
}

// TestCodex_Install_Local_TrustNote (D-10) asserts a local install carries
// exactly one Note naming trust_level = "trusted" and the absolute project
// root, a global install carries none, and neither config.toml nor the
// fake HOME's global config.toml ever gains a [projects header — codegraph
// never writes the trust entry itself.
func TestCodex_Install_Local_TrustNote(t *testing.T) {
	home := fakeHome(t)
	dir := t.TempDir()
	t.Chdir(dir)
	c := codexTarget{}

	localResult := c.Install(LocationLocal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
	trustNotes := 0
	for _, n := range localResult.Notes {
		if strings.Contains(n, `trust_level = "trusted"`) {
			trustNotes++
			if !strings.Contains(n, dir) {
				t.Fatalf("trust note does not name the absolute project root %q: %q", dir, n)
			}
		}
	}
	if trustNotes != 1 {
		t.Fatalf("expected exactly one trust note, got %d: %v", trustNotes, localResult.Notes)
	}

	globalResult := c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
	for _, n := range globalResult.Notes {
		if strings.Contains(n, "trust_level") {
			t.Fatalf("global install must carry no trust note, got: %q", n)
		}
	}

	localConfig := readFile(t, filepath.Join(dir, ".codex", "config.toml"))
	if strings.Contains(localConfig, "[projects") {
		t.Fatalf("codegraph must never write a [projects header, got local config:\n%s", localConfig)
	}
	globalConfig := readFile(t, filepath.Join(home, ".codex", "config.toml"))
	if strings.Contains(globalConfig, "[projects") {
		t.Fatalf("codegraph must never write a [projects header, got global config:\n%s", globalConfig)
	}
}

// TestCodex_Install_RefusesConflictingCodegraphTable (D-07) asserts an
// existing inline codegraph definition under [mcp_servers] is refused with
// a named error, the config.toml is left byte-identical, and the AGENTS.md
// step still runs. Uninstall carries the same refusal.
func TestCodex_Install_RefusesConflictingCodegraphTable(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	c := codexTarget{}

	conflicting := "[mcp_servers]\ncodegraph = { command = \"/other/bin\" }\n"
	configPath := filepath.Join(dir, ".codex", "config.toml")
	relConfigPath := filepath.Join(".codex", "config.toml")
	writeFile(t, configPath, conflicting)

	result := c.Install(LocationLocal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
	if len(result.Errors) == 0 {
		t.Fatalf("expected a conflict error, got none: %+v", result)
	}
	foundNamed := false
	for _, err := range result.Errors {
		if strings.Contains(err.Error(), relConfigPath) {
			foundNamed = true
		}
	}
	if !foundNamed {
		t.Fatalf("expected an error naming %s, got: %v", relConfigPath, result.Errors)
	}
	got := readFile(t, configPath)
	if got != conflicting {
		t.Fatalf("config.toml must stay byte-identical on conflict:\ngot=%q\nwant=%q", got, conflicting)
	}
	if !fileExists(filepath.Join(dir, "AGENTS.md")) {
		t.Fatalf("AGENTS.md step must still run despite the config conflict")
	}

	uninstallResult := c.Uninstall(LocationLocal)
	uninstallErrored := false
	for _, err := range uninstallResult.Errors {
		if strings.Contains(err.Error(), relConfigPath) {
			uninstallErrored = true
		}
	}
	if !uninstallErrored {
		t.Fatalf("expected uninstall to also refuse the conflict, got: %v", uninstallResult.Errors)
	}
	got2 := readFile(t, configPath)
	if got2 != conflicting {
		t.Fatalf("config.toml must stay byte-identical after uninstall's refusal:\ngot=%q\nwant=%q", got2, conflicting)
	}
}

// TestCodex_Uninstall_EmptiedConfigIsRemoved (D-07/D-08 keep-clean
// precedent) asserts that a local install into an empty directory,
// followed by uninstall, removes config.toml entirely (rather than
// leaving an empty file) with a FileResult of removed.
func TestCodex_Uninstall_EmptiedConfigIsRemoved(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	c := codexTarget{}

	c.Install(LocationLocal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
	configPath := filepath.Join(dir, ".codex", "config.toml")
	relConfigPath := filepath.Join(".codex", "config.toml")
	if !fileExists(configPath) {
		t.Fatalf("precondition: config.toml should exist after install")
	}

	result := c.Uninstall(LocationLocal)
	if fileExists(configPath) {
		t.Fatalf("config.toml should have been removed entirely after uninstall")
	}
	found := false
	for _, fr := range result.Files {
		if fr.Path == relConfigPath {
			found = true
			if fr.Action != ActionRemoved {
				t.Fatalf("expected FileResult action %q for %s, got %q", ActionRemoved, relConfigPath, fr.Action)
			}
		}
	}
	if !found {
		t.Fatalf("expected a FileResult for %s, got %v", relConfigPath, result.Files)
	}
}

// TestCodex_DescribePaths_Local (D-09) replaces TestCodex_DescribePaths_LocalEmpty:
// local scope now declares config.toml, AGENTS.md and the shared skill
// package's SKILL.md and manifest paths.
func TestCodex_DescribePaths_Local(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	c := codexTarget{}
	paths := c.DescribePaths(LocationLocal)

	want := []string{
		filepath.Join(".codex", "config.toml"),
		"AGENTS.md",
		filepath.Join(".agents", "skills", "codegraph", "SKILL.md"),
		filepath.Join(".agents", "skills", "codegraph", ".codegraph-manifest.json"),
	}
	for _, w := range want {
		found := false
		for _, p := range paths {
			if p == w {
				found = true
			}
		}
		if !found {
			t.Fatalf("DescribePaths(local) missing %q, got %v", w, paths)
		}
	}
	for _, p := range paths {
		if strings.Contains(p, filepath.Join(".codex", "skills")) {
			t.Fatalf("DescribePaths(local) must never list a .codex/skills path (D-14 read-only), got %v", paths)
		}
	}
}

func TestCodex_GlobalInstall_WritesTOMLTableAndInstructions(t *testing.T) {
	home := fakeHome(t)
	c := codexTarget{}

	result := c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
	if len(result.Files) == 0 {
		t.Fatalf("expected files touched, got none")
	}

	configPath := filepath.Join(home, ".codex", "config.toml")
	got := readFile(t, configPath)
	if !strings.Contains(got, "[mcp_servers.codegraph]") ||
		!strings.Contains(got, `command = "/usr/local/bin/codegraph"`) ||
		!strings.Contains(got, `args = ["serve", "--mcp"]`) {
		t.Fatalf("unexpected config.toml content: %s", got)
	}

	instrPath := filepath.Join(home, ".codex", "AGENTS.md")
	instr := readFile(t, instrPath)
	if !strings.Contains(instr, codegraphSectionStart) || !strings.Contains(instr, "codegraph_explore") {
		t.Fatalf("unexpected AGENTS.md content: %s", instr)
	}
}

func TestCodex_GlobalInstall_PreservesUnrelatedTOMLTable(t *testing.T) {
	home := fakeHome(t)
	configPath := filepath.Join(home, ".codex", "config.toml")
	pre := "[some_other_table]\n" + `key = "value"` + "\n"
	writeFile(t, configPath, pre)

	c := codexTarget{}
	c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

	got := readFile(t, configPath)
	if !strings.Contains(got, "[some_other_table]") || !strings.Contains(got, `key = "value"`) {
		t.Fatalf("unrelated table not preserved: %s", got)
	}
	if !strings.Contains(got, "[mcp_servers.codegraph]") {
		t.Fatalf("codegraph table missing: %s", got)
	}
}

func TestCodex_GlobalRoundTrip_ByteInvariant(t *testing.T) {
	home := fakeHome(t)
	configPath := filepath.Join(home, ".codex", "config.toml")
	pre := "[some_other_table]\n" + `key = "value"` + "\n"
	writeFile(t, configPath, pre)

	c := codexTarget{}
	c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
	c.Uninstall(LocationGlobal)

	got := readFile(t, configPath)
	if got != pre {
		t.Fatalf("round trip not byte-invariant:\ngot=%q\nwant=%q", got, pre)
	}

	instrPath := filepath.Join(home, ".codex", "AGENTS.md")
	if fileExists(instrPath) {
		t.Fatalf("AGENTS.md should have been removed entirely on uninstall (never existed pre-install)")
	}
}

func TestCodex_Install_ReRunIsByteIdempotent(t *testing.T) {
	home := fakeHome(t)
	c := codexTarget{}
	opts := InstallOptions{ExecPath: "/usr/local/bin/codegraph"}

	c.Install(LocationGlobal, opts)
	configBefore := readFile(t, filepath.Join(home, ".codex", "config.toml"))
	instrBefore := readFile(t, filepath.Join(home, ".codex", "AGENTS.md"))

	c.Install(LocationGlobal, opts)
	configAfter := readFile(t, filepath.Join(home, ".codex", "config.toml"))
	instrAfter := readFile(t, filepath.Join(home, ".codex", "AGENTS.md"))

	if configBefore != configAfter {
		t.Fatalf("config.toml changed on idempotent re-run:\nbefore=%q\nafter=%q", configBefore, configAfter)
	}
	if instrBefore != instrAfter {
		t.Fatalf("AGENTS.md changed on idempotent re-run:\nbefore=%q\nafter=%q", instrBefore, instrAfter)
	}
}

// TestCodex_Install_Local_IsIdempotent (CODEX-03) asserts a second local
// install reports every file unchanged and changes no byte.
func TestCodex_Install_Local_IsIdempotent(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	c := codexTarget{}
	opts := InstallOptions{ExecPath: "/usr/local/bin/codegraph"}

	c.Install(LocationLocal, opts)
	configBefore := readFile(t, filepath.Join(dir, ".codex", "config.toml"))
	instrBefore := readFile(t, filepath.Join(dir, "AGENTS.md"))
	skillBefore := readFile(t, filepath.Join(dir, ".agents", "skills", "codegraph", "SKILL.md"))

	result := c.Install(LocationLocal, opts)
	for _, fr := range result.Files {
		if fr.Action != ActionUnchanged {
			t.Errorf("second install: expected unchanged for %s, got %q", fr.Path, fr.Action)
		}
	}

	configAfter := readFile(t, filepath.Join(dir, ".codex", "config.toml"))
	instrAfter := readFile(t, filepath.Join(dir, "AGENTS.md"))
	skillAfter := readFile(t, filepath.Join(dir, ".agents", "skills", "codegraph", "SKILL.md"))
	if configBefore != configAfter {
		t.Fatalf("config.toml changed on idempotent re-run:\nbefore=%q\nafter=%q", configBefore, configAfter)
	}
	if instrBefore != instrAfter {
		t.Fatalf("AGENTS.md changed on idempotent re-run:\nbefore=%q\nafter=%q", instrBefore, instrAfter)
	}
	if skillBefore != skillAfter {
		t.Fatalf("SKILL.md changed on idempotent re-run:\nbefore=%q\nafter=%q", skillBefore, skillAfter)
	}
}

func TestCodex_Uninstall_MissingConfigIsNotFoundNoError(t *testing.T) {
	fakeHome(t)
	c := codexTarget{}
	result := c.Uninstall(LocationGlobal)
	// Must not panic/error; files list should reflect not-found/kept status.
	for _, fr := range result.Files {
		if fr.Action == ActionUpdated || fr.Action == ActionRemoved {
			t.Fatalf("expected no removed/updated actions against a never-installed config, got %+v", fr)
		}
	}
}

func TestCodex_Detect_AfterInstallReportsConfigured(t *testing.T) {
	fakeHome(t)
	c := codexTarget{}
	c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
	got := c.Detect(LocationGlobal)
	if !got.AlreadyConfigured {
		t.Fatalf("expected AlreadyConfigured after install, got %+v", got)
	}
}

// TestCodex_Detect_Local_AfterInstallReportsConfigured (D-09) is
// TestCodex_Detect_AfterInstallReportsConfigured's local-scope
// counterpart, now that Codex supports local.
func TestCodex_Detect_Local_AfterInstallReportsConfigured(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	c := codexTarget{}
	c.Install(LocationLocal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
	got := c.Detect(LocationLocal)
	if !got.AlreadyConfigured {
		t.Fatalf("expected AlreadyConfigured after local install, got %+v", got)
	}
}

func TestCodex_DescribePaths_Global(t *testing.T) {
	c := codexTarget{}
	paths := c.DescribePaths(LocationGlobal)
	if len(paths) < 2 {
		t.Fatalf("expected at least config + instructions paths, got %v", paths)
	}
}

// Task 2 (07-05) note: the three tests below (TestCodex_SharedSkillPackage_
// LastRequester, TestCodex_Install_Local_IsIdempotent,
// TestCodex_ReadOnlySkillDirsFollowLiveVerdict) were written as Task 2's own
// planned RED/GREEN cycle, but Task 1's Capabilities()/installDeclaredSkill/
// uninstallDeclaredSkill/codexSkillDirs implementation already satisfies
// every one of their assertions — they passed the moment Task 1's GREEN
// commit landed, with no further code change required. Recorded honestly
// (05-03-SUMMARY.md precedent, TestSymlinkedSkillDir_ClaudeAndSharedAreOnePackage)
// rather than reshaping any of the three to force an artificial RED they do
// not have; all three are kept as permanent regression coverage for D-14's
// shared-skill-requester contract and D-15's read-only skill roots.

// TestCodex_SharedSkillPackage_LastRequester (D-14) asserts Codex is one
// more requester of the shared skill package: installing opencode then
// codex yields one manifest with targets {codex, opencode}; uninstalling
// codex keeps SKILL.md+manifest with targets {opencode}; uninstalling
// opencode removes both. A sibling foreign skill directory is untouched
// throughout.
func TestCodex_SharedSkillPackage_LastRequester(t *testing.T) {
	for _, loc := range []Location{LocationLocal, LocationGlobal} {
		loc := loc
		t.Run(string(loc), func(t *testing.T) {
			fakeHome(t)
			if loc == LocationLocal {
				t.Chdir(t.TempDir())
			}

			opencode := opencodeTarget{}
			codex := codexTarget{}
			opts := InstallOptions{ExecPath: "/usr/local/bin/codegraph"}

			dir, err := sharedSkillDirPath(loc)
			if err != nil {
				t.Fatalf("sharedSkillDirPath: %v", err)
			}
			sibling := filepath.Join(filepath.Dir(dir), "other", "SKILL.md")
			writeFile(t, sibling, "unrelated content\n")
			siblingBefore := readFile(t, sibling)

			opencode.Install(loc, opts)
			codex.Install(loc, opts)

			manifestPath := filepath.Join(dir, ".codegraph-manifest.json")
			m, present, err := readManifest(manifestPath)
			if err != nil || !present {
				t.Fatalf("readManifest after opencode+codex install: present=%v err=%v", present, err)
			}
			if !targetSetEqual(m.Targets, []TargetID{Codex, Opencode}) {
				t.Fatalf("targets = %v, want exactly {codex, opencode}", m.Targets)
			}
			if readFile(t, sibling) != siblingBefore {
				t.Fatalf("sibling skill directory was modified")
			}

			codex.Uninstall(loc)
			if !fileExists(filepath.Join(dir, "SKILL.md")) {
				t.Fatalf("SKILL.md should remain after uninstalling one of two requesters")
			}
			m2, present2, err2 := readManifest(manifestPath)
			if err2 != nil || !present2 {
				t.Fatalf("readManifest after codex uninstall: present=%v err=%v", present2, err2)
			}
			if !targetSetEqual(m2.Targets, []TargetID{Opencode}) {
				t.Fatalf("targets after codex uninstall = %v, want exactly {opencode}", m2.Targets)
			}
			if readFile(t, sibling) != siblingBefore {
				t.Fatalf("sibling skill directory was modified")
			}

			opencode.Uninstall(loc)
			if fileExists(filepath.Join(dir, "SKILL.md")) {
				t.Fatalf("SKILL.md should be gone after the last requester uninstalls")
			}
			if fileExists(manifestPath) {
				t.Fatalf("manifest should be gone after the last requester uninstalls")
			}
			if readFile(t, sibling) != siblingBefore {
				t.Fatalf("sibling skill directory was modified")
			}
		})
	}
}

// TestCodex_ReadOnlySkillDirsFollowLiveVerdict (D-15) pins
// Capabilities().ReadOnlySkillDirs(loc) to exactly the branch
// 07-LIVE-SESSIONS.md's CODEX-01 verdicts selected:
//
//	D-15 .codex/skills read: yes (B7: r0 = trusted/.codex/skills lists cgprobe-dotcodex; also listed untrusted, B8)
//	D-15 CODEX_HOME/skills read: yes (B7: r1 = home/.codex/skills lists cgprobe-codexhome)
//
// Both are "yes", so both read-only roots are declared. DescribePaths must
// never list either — they are read-only, never written (D-14).
func TestCodex_ReadOnlySkillDirsFollowLiveVerdict(t *testing.T) {
	home := fakeHome(t)
	dir := t.TempDir()
	t.Chdir(dir)
	c := codexTarget{}
	caps := c.Capabilities()

	localRO, err := caps.ReadOnlySkillDirs(LocationLocal)
	if err != nil {
		t.Fatalf("ReadOnlySkillDirs(local): %v", err)
	}
	wantLocal := filepath.Join(".codex", "skills", "codegraph")
	if !containsPath(localRO, wantLocal) {
		t.Fatalf("ReadOnlySkillDirs(local) = %v, want to contain %q (D-15 .codex/skills read: yes)", localRO, wantLocal)
	}

	globalRO, err := caps.ReadOnlySkillDirs(LocationGlobal)
	if err != nil {
		t.Fatalf("ReadOnlySkillDirs(global): %v", err)
	}
	wantGlobal := filepath.Join(home, ".codex", "skills", "codegraph")
	if !containsPath(globalRO, wantGlobal) {
		t.Fatalf("ReadOnlySkillDirs(global) = %v, want to contain %q (D-15 CODEX_HOME/skills read: yes)", globalRO, wantGlobal)
	}

	for _, loc := range []Location{LocationLocal, LocationGlobal} {
		for _, p := range c.DescribePaths(loc) {
			if strings.Contains(p, filepath.Join(".codex", "skills")) {
				t.Fatalf("DescribePaths(%s) must never list a .codex/skills read-only path, got %q", loc, p)
			}
		}
	}
}
