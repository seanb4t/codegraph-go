package agents

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	claudeassets "github.com/seanb4t/codegraph-go"
)

func TestCursor_ID(t *testing.T) {
	c := cursorTarget{}
	if c.ID() != Cursor {
		t.Fatalf("ID() = %v, want %v", c.ID(), Cursor)
	}
}

func TestCursor_SupportsBothLocations(t *testing.T) {
	c := cursorTarget{}
	if !c.SupportsLocation(LocationGlobal) || !c.SupportsLocation(LocationLocal) {
		t.Fatalf("cursor should support both global and local")
	}
}

func TestCursor_LocalInstall_PathArgIsAbsoluteCwd(t *testing.T) {
	fakeHome(t)
	dir := t.TempDir()
	t.Chdir(dir)

	c := cursorTarget{}
	c.Install(LocationLocal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

	got := readFile(t, filepath.Join(dir, ".cursor", "mcp.json"))
	if !strings.Contains(got, `"--path"`) {
		t.Fatalf("missing --path arg: %s", got)
	}
	resolvedDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}
	if !strings.Contains(got, resolvedDir) {
		t.Fatalf("--path arg is not the absolute cwd %q: %s", resolvedDir, got)
	}
	if strings.Contains(got, "${workspaceFolder}") {
		t.Fatalf("local entry should not carry the workspaceFolder literal: %s", got)
	}
}

func TestCursor_GlobalInstall_PathArgIsWorkspaceFolderLiteral(t *testing.T) {
	home := fakeHome(t)
	c := cursorTarget{}
	c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

	got := readFile(t, filepath.Join(home, ".cursor", "mcp.json"))
	if !strings.Contains(got, `"${workspaceFolder}"`) {
		t.Fatalf("global entry missing literal ${workspaceFolder}: %s", got)
	}
}

func TestCursor_Install_NoInstructionsFileWritten(t *testing.T) {
	home := fakeHome(t)
	c := cursorTarget{}
	c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

	entries, err := os.ReadDir(filepath.Join(home, ".cursor"))
	if err != nil {
		t.Fatalf("read ~/.cursor: %v", err)
	}
	for _, e := range entries {
		if e.Name() != "mcp.json" {
			t.Fatalf("unexpected extra file/dir in ~/.cursor: %s (cursor must not write an instructions file)", e.Name())
		}
	}
}

func TestCursor_Install_SelfHealsLegacyRulesFile(t *testing.T) {
	home := fakeHome(t)
	legacyPath := filepath.Join(home, ".cursor", "rules", "codegraph.mdc")
	writeFile(t, legacyPath, "# legacy instructions\n")

	c := cursorTarget{}
	c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

	if fileExists(legacyPath) {
		t.Fatalf("legacy .cursor/rules/codegraph.mdc was not self-heal-deleted")
	}
}

func TestCursor_Uninstall_RemovesOnlyMcpEntry(t *testing.T) {
	home := fakeHome(t)
	c := cursorTarget{}
	c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
	c.Uninstall(LocationGlobal)

	configPath := filepath.Join(home, ".cursor", "mcp.json")
	if fileExists(configPath) {
		got := readFile(t, configPath)
		if strings.Contains(got, `"codegraph"`) {
			t.Fatalf("codegraph entry not removed: %s", got)
		}
	}
}

func TestCursor_RoundTrip_ByteInvariantWithSibling(t *testing.T) {
	home := fakeHome(t)
	configPath := filepath.Join(home, ".cursor", "mcp.json")
	pre := `{
  "mcpServers": {
    "other-server": { "command": "other-binary" }
  }
}
`
	writeFile(t, configPath, pre)

	c := cursorTarget{}
	c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
	c.Uninstall(LocationGlobal)

	got := readFile(t, configPath)
	var gotObj, wantObj map[string]any
	if err := json.Unmarshal([]byte(got), &gotObj); err != nil {
		t.Fatalf("post-round-trip not valid JSON: %v", err)
	}
	if err := json.Unmarshal([]byte(pre), &wantObj); err != nil {
		t.Fatalf("bad fixture: %v", err)
	}
	if !jsonDeepEqual(gotObj, wantObj) {
		t.Fatalf("round trip not byte-invariant:\ngot=%s\nwant=%s", got, pre)
	}
}

// TestCursor_DescribePaths_ListsMcpConfigAndSharedSkill supersedes
// TestCursor_DescribePaths_ListsOnlyMcpConfig: Cursor now declares
// SkillDirs: sharedSkillDirs (D-06), so DescribePaths grows from 1 path to
// 3 — the MCP config, the shared SKILL.md, and its sidecar manifest.
func TestCursor_DescribePaths_ListsMcpConfigAndSharedSkill(t *testing.T) {
	c := cursorTarget{}
	paths := c.DescribePaths(LocationGlobal)
	if len(paths) != 3 {
		t.Fatalf("want exactly 3 paths (mcp config + shared SKILL.md + manifest), got %v", paths)
	}
	var sawMCP, sawSkillMD, sawManifest bool
	for _, p := range paths {
		switch {
		case strings.HasSuffix(p, filepath.Join(".cursor", "mcp.json")):
			sawMCP = true
		case strings.HasSuffix(p, filepath.Join(".agents", "skills", "codegraph", "SKILL.md")):
			sawSkillMD = true
		case strings.HasSuffix(p, filepath.Join(".agents", "skills", "codegraph", ".codegraph-manifest.json")):
			sawManifest = true
		}
	}
	if !sawMCP || !sawSkillMD || !sawManifest {
		t.Fatalf("DescribePaths missing an expected entry (mcp=%v skillmd=%v manifest=%v): %v", sawMCP, sawSkillMD, sawManifest, paths)
	}
}

// TestCursor_Install_WritesSharedSkillPackage (AGENT-04, D-06): Cursor
// installs the codegraph skill through the shared .agents/skills/codegraph
// package (installDeclaredSkill -> installSkillPackage), never a
// Cursor-specific directory — and Uninstall reverses it completely.
func TestCursor_Install_WritesSharedSkillPackage(t *testing.T) {
	for _, loc := range []Location{LocationGlobal, LocationLocal} {
		t.Run(string(loc), func(t *testing.T) {
			fakeHome(t)
			if loc == LocationLocal {
				dir := t.TempDir()
				t.Chdir(dir)
			}
			c := cursorTarget{}
			c.Install(loc, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

			dir, err := sharedSkillDirPath(loc)
			if err != nil {
				t.Fatalf("sharedSkillDirPath: %v", err)
			}
			skillPath := filepath.Join(dir, "SKILL.md")
			want, err := claudeassets.SkillMarkdown()
			if err != nil {
				t.Fatalf("claudeassets.SkillMarkdown: %v", err)
			}
			if got := readFile(t, skillPath); got != string(want) {
				t.Fatalf("shared SKILL.md at %s does not match the embed", skillPath)
			}
			m, present, err := readManifest(skillManifestPath(dir))
			if err != nil || !present {
				t.Fatalf("expected a manifest at %s (present=%v err=%v)", dir, present, err)
			}
			if len(m.Targets) != 1 || !containsTarget(m.Targets, Cursor) {
				t.Fatalf("manifest targets = %v, want exactly [cursor]", m.Targets)
			}

			c.Uninstall(loc)
			if fileExists(skillPath) {
				t.Fatalf("shared SKILL.md not removed after uninstall")
			}
			if fileExists(skillManifestPath(dir)) {
				t.Fatalf("shared manifest not removed after uninstall")
			}
			if fileExists(dir) {
				t.Fatalf("shared skill dir not swept after uninstall")
			}
		})
	}
}

// TestCursor_CorruptedManifestAtSharedDir_NoClaudePresent_DoesNotFalselyAttributeClaude
// is CR-01's iteration-2 regression test (05-REVIEW.md): unlike
// TestGemini_CorruptedManifestAtHarnessExclusiveDir_DoesNotFalselyAttributeClaude,
// which exercises a harness-exclusive directory, this drives the REAL
// production shared-directory path -- cursorTarget{}.Install()/Uninstall()
// through installDeclaredSkill/uninstallDeclaredSkill and
// declaredSkillFallback -- with NO Claude directory present anywhere and NO
// D-17 symlink in play. Cursor's declared skill directory IS
// sharedSkillDirPath by definition (cursor.go), so comparing dir against
// the shared path itself (the iteration-1 defect) is tautologically true
// on every machine, with or without Claude -- self-healing a corrupted
// manifest into [claude cursor] and permanently orphaning the shared
// package on Uninstall, since nothing ever uninstalls a "claude" that was
// never really there. Comparing dir against Claude's OWN declared
// directory instead correctly yields [cursor] here, because the two
// directories are not the same physical directory on this machine.
func TestCursor_CorruptedManifestAtSharedDir_NoClaudePresent_DoesNotFalselyAttributeClaude(t *testing.T) {
	fakeHome(t)

	c := cursorTarget{}
	c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

	sharedDir, err := sharedSkillDirPath(LocationGlobal)
	if err != nil {
		t.Fatalf("sharedSkillDirPath: %v", err)
	}
	manifestPath := skillManifestPath(sharedDir)
	if !fileExists(manifestPath) {
		t.Fatalf("manifest not written by first install: %s", manifestPath)
	}
	if err := os.WriteFile(manifestPath, []byte("{not valid json"), 0o644); err != nil {
		t.Fatalf("corrupt manifest: %v", err)
	}

	// Re-run install over the corrupted manifest: self-healing must not
	// invent Claude as a co-owner of a directory Claude never wrote to and
	// is not symlinked onto.
	c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

	m, present, err := readManifest(manifestPath)
	if err != nil || !present {
		t.Fatalf("manifest not present after self-heal reinstall: present=%v err=%v", present, err)
	}
	if containsTarget(m.Targets, Claude) {
		t.Fatalf("CONFIRMED BUG: Claude falsely attributed as requester of %s with no claude install/symlink ever present: targets=%v", sharedDir, m.Targets)
	}
	if !targetSetEqual(m.Targets, []TargetID{Cursor}) {
		t.Fatalf("Targets after self-heal reinstall (no claude ever involved) = %v, want [cursor]", m.Targets)
	}

	// Cursor is the only real requester -- uninstalling it must fully remove
	// the shared package (D-08: deleted only when targets becomes empty).
	c.Uninstall(LocationGlobal)

	if fileExists(sharedDir) {
		t.Fatalf("CONFIRMED BUG: shared skill dir survived uninstall of its only real requester (cursor) due to phantom claude attribution")
	}
}

// TestCursor_CorruptedManifestAtSharedDir_ClaudeSymlinked_StillAttributesClaude
// is the positive control for the fix above: when Claude's own skill
// directory really IS a symlink onto the shared directory (D-17's `npx
// skills` layout, symlinkedClaudeLayout), a legacy manifest (no targets
// key -- the pre-05-02 schema, which manifestRequesters' D-07 rule already
// reads as owned by Claude) at the shared directory must still resolve
// "assume Claude" to [Claude] here -- this is the genuine case
// declaredSkillFallback exists to preserve. Uninstalling Cursor (a
// non-owner of a package it never requested through this path) must leave
// Claude's package fully intact.
func TestCursor_CorruptedManifestAtSharedDir_ClaudeSymlinked_StillAttributesClaude(t *testing.T) {
	symlinkedClaudeLayout(t)

	sharedDir, err := sharedSkillDirPath(LocationGlobal)
	if err != nil {
		t.Fatalf("sharedSkillDirPath: %v", err)
	}
	manifestPath := skillManifestPath(sharedDir)
	if _, err := writeManifest(manifestPath, skillManifest{
		SchemaVersion:    1,
		CodegraphVersion: "v0.10.0",
		Location:         string(LocationGlobal),
		Files:            map[string]string{manifestKeySkillMD: "sha256:aaaa"},
		// No Targets key: the legacy pre-05-02 schema.
	}); err != nil {
		t.Fatalf("seed legacy manifest: %v", err)
	}

	c := cursorTarget{}
	c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

	m, present, err := readManifest(manifestPath)
	if err != nil || !present {
		t.Fatalf("manifest not present after install over legacy manifest: present=%v err=%v", present, err)
	}
	if !containsTarget(m.Targets, Claude) {
		t.Fatalf("Claude's own skill dir is symlinked onto %s, but the legacy-manifest fallback did not attribute Claude: targets=%v", sharedDir, m.Targets)
	}
	if !targetSetEqual(m.Targets, []TargetID{Claude, Cursor}) {
		t.Fatalf("Targets after install over legacy manifest = %v, want [claude cursor]", m.Targets)
	}

	// Cursor uninstalls; Claude's package (SKILL.md + manifest, still
	// naming Claude) must survive since Claude never relinquished it.
	c.Uninstall(LocationGlobal)

	if !fileExists(sharedDir) {
		t.Fatalf("shared skill dir removed after cursor uninstall, but claude still owns the package")
	}
	afterM, present, err := readManifest(manifestPath)
	if err != nil || !present {
		t.Fatalf("manifest missing after cursor uninstall, but claude still owns the package: present=%v err=%v", present, err)
	}
	if !targetSetEqual(afterM.Targets, []TargetID{Claude}) {
		t.Fatalf("Targets after cursor uninstall = %v, want [claude]", afterM.Targets)
	}
}
