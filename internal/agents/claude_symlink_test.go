package agents

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	claudeassets "github.com/seanb4t/codegraph-go"
)

// symlinkedClaudeLayout builds the maintainer's real layout (05-RESEARCH.md
// Pitfall 2, the `npx skills` convention): a fresh fakeHome with
// `<home>/.agents/skills/codegraph` pre-created as a real directory and
// `<home>/.claude/skills/codegraph` a RELATIVE symlink into it
// ("../../.agents/skills/codegraph" — the exact shape observed on the
// maintainer's own machine), so Claude's skill directory and the shared
// directory every other target writes into are ONE physical directory
// (D-17). Returns the fake home root.
func symlinkedClaudeLayout(t *testing.T) string {
	t.Helper()

	home := fakeHome(t)
	if err := os.MkdirAll(filepath.Join(home, ".agents", "skills", "codegraph"), 0o755); err != nil {
		t.Fatalf("mkdir shared skill dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".claude", "skills"), 0o755); err != nil {
		t.Fatalf("mkdir .claude/skills: %v", err)
	}
	target := filepath.Join("..", "..", ".agents", "skills", "codegraph")
	if err := os.Symlink(target, filepath.Join(home, ".claude", "skills", "codegraph")); err != nil {
		t.Fatalf("symlink claude skill dir -> shared skill dir: %v", err)
	}
	return home
}

// TestSymlinkedSkillDir_ClaudeAndSharedAreOnePackage is D-17's tracer test:
// Claude's Install and the shared writer's installSkillPackage, called
// through a symlinked layout, must produce exactly ONE manifest recording
// BOTH requesters — never two independent manifests, and never one
// overwriting the other's requester entry.
func TestSymlinkedSkillDir_ClaudeAndSharedAreOnePackage(t *testing.T) {
	home := symlinkedClaudeLayout(t)
	c := claudeTarget{}

	installResult := c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
	if len(installResult.Errors) != 0 {
		t.Fatalf("claude Install returned errors: %v", installResult.Errors)
	}

	sharedDir, err := sharedSkillDirPath(LocationGlobal)
	if err != nil {
		t.Fatalf("sharedSkillDirPath: %v", err)
	}
	var r WriteResult
	installSkillPackage(&r, sharedDir, LocationGlobal, Cursor, refuseUnmanifested)
	if len(r.Errors) != 0 {
		t.Fatalf("installSkillPackage(cursor) returned errors: %v", r.Errors)
	}

	var manifestPaths []string
	if err := filepath.WalkDir(home, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && d.Name() == skillManifestFileName {
			manifestPaths = append(manifestPaths, path)
		}
		return nil
	}); err != nil {
		t.Fatalf("walk %s: %v", home, err)
	}
	if len(manifestPaths) != 1 {
		t.Fatalf("expected exactly one manifest under home (walked without following symlinks), got %v", manifestPaths)
	}

	m, present, err := readManifest(manifestPaths[0])
	if err != nil || !present {
		t.Fatalf("readManifest(%s): present=%v err=%v", manifestPaths[0], present, err)
	}
	if !targetSetEqual(m.Targets, []TargetID{Claude, Cursor}) {
		t.Fatalf("manifest Targets = %v, want set-equal to [claude cursor]", m.Targets)
	}
	wantKeys := []string{manifestKeySkillMD, manifestKeyScript, manifestKeyHooksFrag}
	if len(m.Files) != len(wantKeys) {
		t.Fatalf("manifest Files = %#v, want exactly the keys %v", m.Files, wantKeys)
	}
	for _, k := range wantKeys {
		if _, ok := m.Files[k]; !ok {
			t.Fatalf("manifest Files missing key %q: %#v", k, m.Files)
		}
	}

	claudeDir, err := claudeSkillDirPath(LocationGlobal)
	if err != nil {
		t.Fatalf("claudeSkillDirPath: %v", err)
	}
	info, err := os.Lstat(claudeDir)
	if err != nil {
		t.Fatalf("Lstat(claudeDir): %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("claude skill dir is no longer a symlink: mode=%v", info.Mode())
	}
}

// TestSymlinkedSkillDir_ClaudeUninstallKeepsOtherRequester is the direct
// regression test for T-05-11: with the shared package requested by both
// claude and cursor, claude's Uninstall must remove only its OWN exclusive
// artifacts (the manifest's script/hooks keys, plus its own session-nudge
// script and SessionStart registration, which live outside the skill
// directory) — the shared SKILL.md and manifest must survive for cursor.
// Only once cursor (the last remaining requester) also uninstalls does the
// shared package actually disappear — and even then, Claude's own symlink
// must never be unlinked (dangling, not removed).
func TestSymlinkedSkillDir_ClaudeUninstallKeepsOtherRequester(t *testing.T) {
	home := symlinkedClaudeLayout(t)
	c := claudeTarget{}

	if r := c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"}); len(r.Errors) != 0 {
		t.Fatalf("claude Install returned errors: %v", r.Errors)
	}
	sharedDir, err := sharedSkillDirPath(LocationGlobal)
	if err != nil {
		t.Fatalf("sharedSkillDirPath: %v", err)
	}
	var installShared WriteResult
	installSkillPackage(&installShared, sharedDir, LocationGlobal, Cursor, refuseUnmanifested)
	if len(installShared.Errors) != 0 {
		t.Fatalf("installSkillPackage(cursor) returned errors: %v", installShared.Errors)
	}

	uninstallResult := c.Uninstall(LocationGlobal)
	if len(uninstallResult.Errors) != 0 {
		t.Fatalf("claude Uninstall returned errors: %v", uninstallResult.Errors)
	}

	wantSkillMD, err := claudeassets.SkillMarkdown()
	if err != nil {
		t.Fatalf("claudeassets.SkillMarkdown: %v", err)
	}
	skillPath := filepath.Join(sharedDir, skillFileName)
	gotSkillMD, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("read shared SKILL.md after claude uninstall: %v", err)
	}
	if string(gotSkillMD) != string(wantSkillMD) {
		t.Fatalf("shared SKILL.md changed after claude uninstall")
	}

	manifestPath := skillManifestPath(sharedDir)
	m, present, err := readManifest(manifestPath)
	if err != nil || !present {
		t.Fatalf("readManifest(%s) after claude uninstall: present=%v err=%v", manifestPath, present, err)
	}
	if !targetSetEqual(m.Targets, []TargetID{Cursor}) {
		t.Fatalf("manifest Targets after claude uninstall = %v, want [cursor]", m.Targets)
	}
	if len(m.Files) != 1 {
		t.Fatalf("manifest Files after claude uninstall = %#v, want exactly the SKILL.md key", m.Files)
	}
	if _, ok := m.Files[manifestKeySkillMD]; !ok {
		t.Fatalf("manifest Files after claude uninstall missing %q: %#v", manifestKeySkillMD, m.Files)
	}

	scriptPath := filepath.Join(home, ".claude", "hooks", "session-nudge.sh")
	if fileExists(scriptPath) {
		t.Fatalf("claude session-nudge.sh still present after claude uninstall")
	}

	settingsPath := filepath.Join(home, ".claude", "settings.json")
	claudeCmd, err := claudeHookCommand(LocationGlobal)
	if err != nil {
		t.Fatalf("claudeHookCommand: %v", err)
	}
	if fileExists(settingsPath) {
		var decoded map[string]any
		if err := json.Unmarshal([]byte(readFile(t, settingsPath)), &decoded); err != nil {
			t.Fatalf("unmarshal settings.json: %v", err)
		}
		if hooks, ok := decoded["hooks"].(map[string]any); ok {
			if sessionStart, ok := hooks["SessionStart"].([]any); ok {
				for _, e := range sessionStart {
					entry, ok := e.(map[string]any)
					if !ok {
						continue
					}
					entries, _ := entry["hooks"].([]any)
					for _, he := range entries {
						hookObj, ok := he.(map[string]any)
						if !ok {
							continue
						}
						if hookObj["command"] == claudeCmd {
							t.Fatalf("settings.json still carries a SessionStart command equal to claudeHookCommand after uninstall: %#v", hookObj)
						}
					}
				}
			}
		}
	}

	claudeDir, err := claudeSkillDirPath(LocationGlobal)
	if err != nil {
		t.Fatalf("claudeSkillDirPath: %v", err)
	}
	info, err := os.Lstat(claudeDir)
	if err != nil {
		t.Fatalf("Lstat(claudeDir) after claude uninstall: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("claude skill dir link removed by claude uninstall: mode=%v", info.Mode())
	}

	// Now cursor — the last remaining requester — uninstalls too.
	var uninstallShared WriteResult
	uninstallSkillPackage(&uninstallShared, sharedDir, Cursor, nil, refuseUnmanifested)
	if len(uninstallShared.Errors) != 0 {
		t.Fatalf("uninstallSkillPackage(cursor) returned errors: %v", uninstallShared.Errors)
	}
	if fileExists(skillPath) {
		t.Fatalf("shared SKILL.md still present after last-requester uninstall")
	}
	if fileExists(manifestPath) {
		t.Fatalf("shared manifest still present after last-requester uninstall")
	}
	if fileExists(sharedDir) {
		t.Fatalf("shared dir still present after last-requester uninstall")
	}
	info2, err := os.Lstat(claudeDir)
	if err != nil {
		t.Fatalf("Lstat(claudeDir) after last-requester uninstall: %v", err)
	}
	if info2.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("claude skill dir link removed after last-requester uninstall: mode=%v", info2.Mode())
	}
}

// TestSymlinkedSkillDir_DanglingLinkReinstall proves a dangling
// `~/.claude/skills/codegraph` link (its target removed by the last
// requester leaving) is healed by the next claude Install: the resolved
// target directory is recreated, SKILL.md and a fresh single-requester
// manifest are written into it, and the link itself still resolves there.
func TestSymlinkedSkillDir_DanglingLinkReinstall(t *testing.T) {
	symlinkedClaudeLayout(t)
	c := claudeTarget{}

	if r := c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"}); len(r.Errors) != 0 {
		t.Fatalf("claude Install returned errors: %v", r.Errors)
	}
	sharedDir, err := sharedSkillDirPath(LocationGlobal)
	if err != nil {
		t.Fatalf("sharedSkillDirPath: %v", err)
	}
	var installShared WriteResult
	installSkillPackage(&installShared, sharedDir, LocationGlobal, Cursor, refuseUnmanifested)
	if len(installShared.Errors) != 0 {
		t.Fatalf("installSkillPackage(cursor) returned errors: %v", installShared.Errors)
	}
	if r := c.Uninstall(LocationGlobal); len(r.Errors) != 0 {
		t.Fatalf("claude Uninstall returned errors: %v", r.Errors)
	}
	var uninstallShared WriteResult
	uninstallSkillPackage(&uninstallShared, sharedDir, Cursor, nil, refuseUnmanifested)
	if len(uninstallShared.Errors) != 0 {
		t.Fatalf("uninstallSkillPackage(cursor) returned errors: %v", uninstallShared.Errors)
	}

	claudeDir, err := claudeSkillDirPath(LocationGlobal)
	if err != nil {
		t.Fatalf("claudeSkillDirPath: %v", err)
	}
	if fileExists(claudeDir) {
		t.Fatalf("precondition not met: claude skill dir should be a dangling link before reinstall")
	}
	if info, lerr := os.Lstat(claudeDir); lerr != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("precondition not met: claude skill dir should still be a symlink: info=%v err=%v", info, lerr)
	}

	reinstall := c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
	if len(reinstall.Errors) != 0 {
		t.Fatalf("claude Install through dangling link returned errors: %v", reinstall.Errors)
	}

	wantSkillMD, err := claudeassets.SkillMarkdown()
	if err != nil {
		t.Fatalf("claudeassets.SkillMarkdown: %v", err)
	}
	gotSkillMD, err := os.ReadFile(filepath.Join(sharedDir, skillFileName))
	if err != nil {
		t.Fatalf("read SKILL.md through the resolved real dir: %v", err)
	}
	if string(gotSkillMD) != string(wantSkillMD) {
		t.Fatalf("SKILL.md content after reinstall does not match embedded content")
	}

	manifestPath := skillManifestPath(sharedDir)
	m, present, err := readManifest(manifestPath)
	if err != nil || !present {
		t.Fatalf("readManifest(%s) after reinstall: present=%v err=%v", manifestPath, present, err)
	}
	if !targetSetEqual(m.Targets, []TargetID{Claude}) {
		t.Fatalf("manifest Targets after reinstall = %v, want [claude]", m.Targets)
	}

	resolved, err := filepath.EvalSymlinks(claudeDir)
	if err != nil {
		t.Fatalf("EvalSymlinks(claudeDir) after reinstall: %v", err)
	}
	wantResolved, err := filepath.EvalSymlinks(sharedDir)
	if err != nil {
		t.Fatalf("EvalSymlinks(sharedDir) after reinstall: %v", err)
	}
	if resolved != wantResolved {
		t.Fatalf("claude skill dir link resolves to %q, want %q", resolved, wantResolved)
	}
}

// TestSymlinkedSkillDir_ForeignContentKeptForeign proves T-05-12: when the
// real directory a symlinked Claude path resolves to already holds
// non-codegraph content with no manifest, Install must never overwrite it
// (the foreign-content policy, D-14, applies through the symlink exactly as
// it would to a plain directory) — while every one of Claude's OTHER
// artifacts (MCP config, CLAUDE.md, script, SessionStart hooks) is still
// written normally. Uninstall must leave the foreign content untouched too.
func TestSymlinkedSkillDir_ForeignContentKeptForeign(t *testing.T) {
	home := symlinkedClaudeLayout(t)
	sharedDir, err := sharedSkillDirPath(LocationGlobal)
	if err != nil {
		t.Fatalf("sharedSkillDirPath: %v", err)
	}
	foreignContent := "# Someone else's skill\n\nThis was never written by codegraph.\n"
	writeFile(t, filepath.Join(sharedDir, skillFileName), foreignContent)

	claudeDir, err := claudeSkillDirPath(LocationGlobal)
	if err != nil {
		t.Fatalf("claudeSkillDirPath: %v", err)
	}

	c := claudeTarget{}
	result := c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
	if len(result.Errors) != 0 {
		t.Fatalf("claude Install returned errors: %v", result.Errors)
	}

	foundForeign := false
	for _, fr := range result.Files {
		if fr.Path == claudeDir && fr.Action == ActionKeptForeign {
			foundForeign = true
		}
	}
	if !foundForeign {
		t.Fatalf("result.Files does not include a %q entry for %s: %+v", ActionKeptForeign, claudeDir, result.Files)
	}

	gotContent := readFile(t, filepath.Join(sharedDir, skillFileName))
	if gotContent != foreignContent {
		t.Fatalf("foreign SKILL.md content changed:\ngot=%q\nwant=%q", gotContent, foreignContent)
	}

	var manifestPaths []string
	for _, root := range []string{filepath.Join(home, ".agents"), filepath.Join(home, ".claude", "skills")} {
		if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && d.Name() == skillManifestFileName {
				manifestPaths = append(manifestPaths, path)
			}
			return nil
		}); err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}
	if len(manifestPaths) != 0 {
		t.Fatalf("a manifest exists despite foreign content: %v", manifestPaths)
	}

	configPath, err := claudeConfigPath(LocationGlobal)
	if err != nil {
		t.Fatalf("claudeConfigPath: %v", err)
	}
	if !fileExists(configPath) {
		t.Fatalf("claude MCP config was not written: %s", configPath)
	}
	instrPath, err := claudeInstructionsPath(LocationGlobal)
	if err != nil {
		t.Fatalf("claudeInstructionsPath: %v", err)
	}
	if !fileExists(instrPath) {
		t.Fatalf("claude CLAUDE.md was not written: %s", instrPath)
	}
	scriptPath, err := claudeHooksScriptPath(LocationGlobal)
	if err != nil {
		t.Fatalf("claudeHooksScriptPath: %v", err)
	}
	if !fileExists(scriptPath) {
		t.Fatalf("claude session-nudge.sh was not written: %s", scriptPath)
	}
	settingsPath, err := claudeSettingsPath(LocationGlobal)
	if err != nil {
		t.Fatalf("claudeSettingsPath: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(readFile(t, settingsPath)), &decoded); err != nil {
		t.Fatalf("unmarshal settings.json: %v", err)
	}
	hooks, ok := decoded["hooks"].(map[string]any)
	if !ok {
		t.Fatalf("SessionStart hooks were not written despite foreign skill content: %#v", decoded)
	}
	if _, ok := hooks["SessionStart"].([]any); !ok {
		t.Fatalf("SessionStart hooks were not written despite foreign skill content: %#v", hooks)
	}

	uninstallResult := c.Uninstall(LocationGlobal)
	if len(uninstallResult.Errors) != 0 {
		t.Fatalf("claude Uninstall returned errors: %v", uninstallResult.Errors)
	}
	afterContent := readFile(t, filepath.Join(sharedDir, skillFileName))
	if afterContent != foreignContent {
		t.Fatalf("foreign SKILL.md content changed after uninstall:\ngot=%q\nwant=%q", afterContent, foreignContent)
	}
}

// TestConfiguredSkillLocations_RequiresClaudeInTargets is the direct
// regression test for T-05-13: a manifest at Claude's own path is no
// longer sufficient proof Claude was ever installed there — since D-17,
// that path can belong to a shared directory another agent alone
// requested. ConfiguredSkillLocations(Claude) must require Claude among
// the manifest's requesters, while a legacy (no targets key) or corrupt
// manifest — both of which read as owned by Claude via manifestRequesters'
// D-07 rule — still count (WR-04 preserved).
func TestConfiguredSkillLocations_RequiresClaudeInTargets(t *testing.T) {
	t.Run("symlinked shared dir installed by another target only", func(t *testing.T) {
		symlinkedClaudeLayout(t)
		sharedDir, err := sharedSkillDirPath(LocationGlobal)
		if err != nil {
			t.Fatalf("sharedSkillDirPath: %v", err)
		}
		var r WriteResult
		installSkillPackage(&r, sharedDir, LocationGlobal, Cursor, refuseUnmanifested)
		if len(r.Errors) != 0 {
			t.Fatalf("installSkillPackage(cursor) returned errors: %v", r.Errors)
		}

		locs := ConfiguredSkillLocations(Claude)
		for _, l := range locs {
			if l == LocationGlobal {
				t.Fatalf("ConfiguredSkillLocations(Claude) contains global when only cursor requested the shared package: %v", locs)
			}
		}

		c := claudeTarget{}
		if ir := c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"}); len(ir.Errors) != 0 {
			t.Fatalf("claude Install returned errors: %v", ir.Errors)
		}
		locsAfter := ConfiguredSkillLocations(Claude)
		found := false
		for _, l := range locsAfter {
			if l == LocationGlobal {
				found = true
			}
		}
		if !found {
			t.Fatalf("ConfiguredSkillLocations(Claude) does not contain global after claude Install: %v", locsAfter)
		}
	})

	t.Run("manifest at claude's path naming only cursor is excluded", func(t *testing.T) {
		home := fakeHome(t)
		manifestPath := filepath.Join(home, ".claude", "skills", "codegraph", ".codegraph-manifest.json")
		if _, err := writeManifest(manifestPath, skillManifest{
			SchemaVersion:    manifestSchemaVersion,
			CodegraphVersion: "v0.14.0",
			Location:         string(LocationGlobal),
			Files:            map[string]string{manifestKeySkillMD: "sha256:aaaa"},
			Targets:          []TargetID{Cursor},
		}); err != nil {
			t.Fatalf("seed manifest: %v", err)
		}

		locs := ConfiguredSkillLocations(Claude)
		for _, l := range locs {
			if l == LocationGlobal {
				t.Fatalf("ConfiguredSkillLocations(Claude) includes global for a manifest naming only cursor: %v", locs)
			}
		}
	})

	t.Run("legacy manifest with no targets key is included", func(t *testing.T) {
		home := fakeHome(t)
		manifestPath := filepath.Join(home, ".claude", "skills", "codegraph", ".codegraph-manifest.json")
		if _, err := writeManifest(manifestPath, skillManifest{
			SchemaVersion:    1,
			CodegraphVersion: "v0.10.0",
			Location:         string(LocationGlobal),
			Files:            map[string]string{manifestKeySkillMD: "sha256:aaaa"},
		}); err != nil {
			t.Fatalf("seed legacy manifest: %v", err)
		}

		locs := ConfiguredSkillLocations(Claude)
		found := false
		for _, l := range locs {
			if l == LocationGlobal {
				found = true
			}
		}
		if !found {
			t.Fatalf("ConfiguredSkillLocations(Claude) excludes global for a legacy (no targets key) manifest: %v", locs)
		}
	})

	t.Run("corrupt manifest is included", func(t *testing.T) {
		home := fakeHome(t)
		manifestPath := filepath.Join(home, ".claude", "skills", "codegraph", ".codegraph-manifest.json")
		if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
			t.Fatalf("mkdir skill dir: %v", err)
		}
		if err := os.WriteFile(manifestPath, []byte("{not valid json"), 0o644); err != nil {
			t.Fatalf("seed corrupt manifest: %v", err)
		}

		locs := ConfiguredSkillLocations(Claude)
		found := false
		for _, l := range locs {
			if l == LocationGlobal {
				found = true
			}
		}
		if !found {
			t.Fatalf("ConfiguredSkillLocations(Claude) excludes global for a corrupt manifest: %v", locs)
		}
	})
}

// TestSharedSkillWriter_NotesSameDirAsClaude is D-17's "both writers
// compare" other direction: the shared writer itself (installSkillPackage
// for a requester other than Claude) must also detect and surface a
// symlinked shared directory, so the user sees that one package serves
// both agents even when they only ever look at, say, Cursor's install
// output.
func TestSharedSkillWriter_NotesSameDirAsClaude(t *testing.T) {
	t.Run("symlinked layout notes the shared directory", func(t *testing.T) {
		symlinkedClaudeLayout(t)
		sharedDir, err := sharedSkillDirPath(LocationGlobal)
		if err != nil {
			t.Fatalf("sharedSkillDirPath: %v", err)
		}
		claudeDir, err := claudeSkillDirPath(LocationGlobal)
		if err != nil {
			t.Fatalf("claudeSkillDirPath: %v", err)
		}

		var r WriteResult
		installSkillPackage(&r, sharedDir, LocationGlobal, Cursor, refuseUnmanifested)
		if len(r.Errors) != 0 {
			t.Fatalf("installSkillPackage(cursor) returned errors: %v", r.Errors)
		}
		if len(r.Notes) != 1 {
			t.Fatalf("Notes = %v, want exactly one entry", r.Notes)
		}
		if !strings.Contains(r.Notes[0], sharedDir) || !strings.Contains(r.Notes[0], claudeDir) {
			t.Fatalf("Notes[0] = %q, want it to mention both %q and %q", r.Notes[0], sharedDir, claudeDir)
		}
	})

	t.Run("plain (non-symlinked) layout has no notes", func(t *testing.T) {
		fakeHome(t)
		sharedDir, err := sharedSkillDirPath(LocationGlobal)
		if err != nil {
			t.Fatalf("sharedSkillDirPath: %v", err)
		}
		var r WriteResult
		installSkillPackage(&r, sharedDir, LocationGlobal, Cursor, refuseUnmanifested)
		if len(r.Errors) != 0 {
			t.Fatalf("installSkillPackage(cursor) returned errors: %v", r.Errors)
		}
		if len(r.Notes) != 0 {
			t.Fatalf("Notes = %v, want none", r.Notes)
		}
	})

	t.Run("requester claude never notes itself", func(t *testing.T) {
		symlinkedClaudeLayout(t)
		sharedDir, err := sharedSkillDirPath(LocationGlobal)
		if err != nil {
			t.Fatalf("sharedSkillDirPath: %v", err)
		}
		var r WriteResult
		installSkillPackage(&r, sharedDir, LocationGlobal, Claude, refuseUnmanifested)
		if len(r.Errors) != 0 {
			t.Fatalf("installSkillPackage(claude) returned errors: %v", r.Errors)
		}
		if len(r.Notes) != 0 {
			t.Fatalf("Notes = %v, want none when requester is claude", r.Notes)
		}
	})
}

// TestSymlinkedSkillDir_PreToolNudgeEvidencedBySettingsWhenForeign is CR-01's
// regression test (06-REVIEW.md): combines symlinkedClaudeLayout with
// pre-existing foreign, unmanifested content in the shared directory (the
// exact precondition TestSymlinkedSkillDir_ForeignContentKeptForeign
// already proves keeps the manifest step from EVER running there) and
// PreToolNudge: PreToolNudgeOn. Before the fix, the guard and its
// settings.json registration were written successfully but never recorded
// anywhere, so a subsequent Keep install silently forgot the opt-in
// forever (D-10's "sticky until explicitly turned off" broken). After the
// fix, preToolNudgeEvidenced treats settings.json's own registration as
// evidence, so the opt-in is refreshed rather than orphaned.
func TestSymlinkedSkillDir_PreToolNudgeEvidencedBySettingsWhenForeign(t *testing.T) {
	symlinkedClaudeLayout(t)
	sharedDir, err := sharedSkillDirPath(LocationGlobal)
	if err != nil {
		t.Fatalf("sharedSkillDirPath: %v", err)
	}
	writeFile(t, filepath.Join(sharedDir, skillFileName), "# Someone else's skill\n\nThis was never written by codegraph.\n")

	c := claudeTarget{}
	onResult := c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: PreToolNudgeOn})
	if len(onResult.Errors) != 0 {
		t.Fatalf("On install returned errors: %v", onResult.Errors)
	}

	guardPath, err := claudePreToolGuardPath(LocationGlobal)
	if err != nil {
		t.Fatalf("claudePreToolGuardPath: %v", err)
	}
	if !fileExists(guardPath) {
		t.Fatalf("On install did not write the guard at %s despite the foreign skill dir", guardPath)
	}

	manifestPath, err := claudeManifestPath(LocationGlobal)
	if err != nil {
		t.Fatalf("claudeManifestPath: %v", err)
	}
	if fileExists(manifestPath) {
		t.Fatalf("precondition broken: a manifest exists at %s despite foreign skill content", manifestPath)
	}

	// The defect CR-01 found: the manifest-only check reports the opt-in as
	// unrecorded here, even though it was genuinely, successfully opted in.
	if recorded, _ := preToolNudgeRecorded(LocationGlobal); recorded {
		t.Fatalf("precondition broken: preToolNudgeRecorded already true with no manifest on disk — this test no longer isolates CR-01")
	}

	// The fix: preToolNudgeEvidenced widens that to true via settings.json's
	// own registration.
	if recorded, readable := preToolNudgeEvidenced(LocationGlobal); !recorded || !readable {
		t.Fatalf("preToolNudgeEvidenced = (%v, %v), want (true, true): settings.json already carries codegraph's own PreToolUse registration (CR-01)", recorded, readable)
	}

	// A moved binary: a subsequent plain (Keep) install must refresh the
	// guard's baked-in ExecPath, never silently leave an orphaned old one
	// forever (the CR-01 consequence chain's step 4, upgrade's refresh).
	keepResult := c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph-v2", PreToolNudge: PreToolNudgeKeep})
	if len(keepResult.Errors) != 0 {
		t.Fatalf("Keep install returned errors: %v", keepResult.Errors)
	}
	guardAfterKeep := readFile(t, guardPath)
	if !strings.Contains(guardAfterKeep, "codegraph-v2") {
		t.Fatalf("Keep did not refresh the orphaned guard for the moved binary; guard still reads:\n%s", guardAfterKeep)
	}

	// The foreign SKILL.md content itself must still be untouched — CR-01's
	// fix is scoped to the PreToolUse record, never a license to touch
	// D-14's protected foreign content.
	gotContent := readFile(t, filepath.Join(sharedDir, skillFileName))
	if gotContent != "# Someone else's skill\n\nThis was never written by codegraph.\n" {
		t.Fatalf("foreign SKILL.md content changed:\ngot=%q", gotContent)
	}
}
