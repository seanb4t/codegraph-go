package agents

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	claudeassets "github.com/seanb4t/codegraph-go"
)

func TestKiro_ID(t *testing.T) {
	k := kiroTarget{}
	if k.ID() != Kiro {
		t.Fatalf("ID() = %v, want %v", k.ID(), Kiro)
	}
}

func TestKiro_SupportsBothLocations(t *testing.T) {
	k := kiroTarget{}
	if !k.SupportsLocation(LocationGlobal) || !k.SupportsLocation(LocationLocal) {
		t.Fatalf("kiro should support both global and local")
	}
}

func TestKiro_Install_NoInstructionsFileWritten(t *testing.T) {
	home := fakeHome(t)
	k := kiroTarget{}
	k.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

	if fileExists(filepath.Join(home, ".kiro", "steering", "codegraph.md")) {
		t.Fatalf("kiro install must not (re)write the legacy steering file")
	}
}

func TestKiro_Install_SelfHealsLegacySteeringFile(t *testing.T) {
	home := fakeHome(t)
	legacyPath := filepath.Join(home, ".kiro", "steering", "codegraph.md")
	writeFile(t, legacyPath, "# legacy steering doc\n")

	k := kiroTarget{}
	k.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

	if fileExists(legacyPath) {
		t.Fatalf("legacy steering file was not self-heal-deleted")
	}
}

func TestKiro_LocalInstall_SelfHealsLegacySteeringFile(t *testing.T) {
	fakeHome(t)
	dir := t.TempDir()
	t.Chdir(dir)

	legacyPath := filepath.Join(dir, ".kiro", "steering", "codegraph.md")
	writeFile(t, legacyPath, "# legacy steering doc\n")

	k := kiroTarget{}
	k.Install(LocationLocal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

	if fileExists(legacyPath) {
		t.Fatalf("legacy local steering file was not self-heal-deleted")
	}
	if !fileExists(filepath.Join(dir, ".kiro", "settings", "mcp.json")) {
		t.Fatalf("./.kiro/settings/mcp.json was not written")
	}
}

func TestKiro_Install_NoteCarriesDisabledByDefaultHint(t *testing.T) {
	fakeHome(t)
	k := kiroTarget{}
	result := k.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

	found := false
	for _, n := range result.Notes {
		if strings.Contains(n, "disabled by default") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a disabled-by-default note in WriteResult.Notes, got %v", result.Notes)
	}
}

func TestKiro_EntryHasTypeStdio(t *testing.T) {
	home := fakeHome(t)
	k := kiroTarget{}
	k.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

	configPath := filepath.Join(home, ".kiro", "settings", "mcp.json")
	got := readFile(t, configPath)
	var decoded map[string]any
	if err := json.Unmarshal([]byte(got), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	mcpServers, _ := decoded["mcpServers"].(map[string]any)
	entry, _ := mcpServers["codegraph"].(map[string]any)
	if entry["type"] != "stdio" {
		t.Fatalf("kiro entry must carry type:stdio, got %v", entry)
	}
}

func TestKiro_RoundTrip_ByteInvariant(t *testing.T) {
	home := fakeHome(t)
	configPath := filepath.Join(home, ".kiro", "settings", "mcp.json")
	pre := `{
  "mcpServers": {
    "other-server": { "command": "other-binary" }
  }
}
`
	writeFile(t, configPath, pre)

	k := kiroTarget{}
	k.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
	k.Uninstall(LocationGlobal)

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

// TestKiro_DescribePaths_ListsMcpConfigAndSkill supersedes
// TestKiro_DescribePaths_ListsOnlyMcpConfig: Kiro now declares
// SkillDirs: kiroSkillDirs (AGENT-11, D-06 correction (d)), so
// DescribePaths grows from 1 path to 3 — the MCP config, the Kiro
// SKILL.md, and its sidecar manifest.
func TestKiro_DescribePaths_ListsMcpConfigAndSkill(t *testing.T) {
	k := kiroTarget{}
	paths := k.DescribePaths(LocationGlobal)
	if len(paths) != 3 {
		t.Fatalf("want exactly 3 paths (mcp config + Kiro SKILL.md + manifest), got %v", paths)
	}
	var sawMCP, sawSkillMD, sawManifest bool
	for _, p := range paths {
		switch {
		case strings.HasSuffix(p, filepath.Join(".kiro", "settings", "mcp.json")):
			sawMCP = true
		case strings.HasSuffix(p, filepath.Join(".kiro", "skills", "codegraph", "SKILL.md")):
			sawSkillMD = true
		case strings.HasSuffix(p, filepath.Join(".kiro", "skills", "codegraph", ".codegraph-manifest.json")):
			sawManifest = true
		}
	}
	if !sawMCP || !sawSkillMD || !sawManifest {
		t.Fatalf("DescribePaths missing an expected entry (mcp=%v skillmd=%v manifest=%v): %v", sawMCP, sawSkillMD, sawManifest, paths)
	}
}

// TestKiro_Install_WritesHarnessSkillDir (AGENT-11, D-06 correction (d),
// kiro.dev/docs/steering, fetched 2026-09-18): Kiro installs the codegraph
// skill at its own harness-specific directory (.kiro/skills/codegraph) —
// the only entry in kiroSkillDirs — reversed completely by Uninstall.
func TestKiro_Install_WritesHarnessSkillDir(t *testing.T) {
	for _, loc := range []Location{LocationGlobal, LocationLocal} {
		t.Run(string(loc), func(t *testing.T) {
			fakeHome(t)
			if loc == LocationLocal {
				dir := t.TempDir()
				t.Chdir(dir)
			}
			k := kiroTarget{}
			k.Install(loc, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

			harnessDir, err := k.Capabilities().WrittenSkillDir(loc)
			if err != nil {
				t.Fatalf("WrittenSkillDir: %v", err)
			}
			skillPath := filepath.Join(harnessDir, "SKILL.md")
			want, err := claudeassets.SkillMarkdown()
			if err != nil {
				t.Fatalf("claudeassets.SkillMarkdown: %v", err)
			}
			if got := readFile(t, skillPath); got != string(want) {
				t.Fatalf("harness SKILL.md at %s does not match the embed", skillPath)
			}
			m, present, err := readManifest(skillManifestPath(harnessDir))
			if err != nil || !present {
				t.Fatalf("expected a manifest at %s (present=%v err=%v)", harnessDir, present, err)
			}
			if len(m.Targets) != 1 || !containsTarget(m.Targets, Kiro) {
				t.Fatalf("manifest targets = %v, want exactly [kiro]", m.Targets)
			}

			k.Uninstall(loc)
			if fileExists(skillPath) {
				t.Fatalf("harness SKILL.md not removed after uninstall")
			}
			if fileExists(skillManifestPath(harnessDir)) {
				t.Fatalf("harness manifest not removed after uninstall")
			}
			if fileExists(harnessDir) {
				t.Fatalf("harness skill dir not swept after uninstall")
			}
		})
	}
}

// TestKiro_Install_WritesNoAgentsMd (D-06(d)): Kiro's own instructions
// handling is unchanged by this plan — it writes NO AGENTS.md at either
// scope (its own reads of ./AGENTS.md / ~/.kiro/steering/AGENTS.md,
// written by other targets, are advisory-only per kiro.dev/docs/steering)
// and it writes no shared .agents/skills/codegraph alias.
func TestKiro_Install_WritesNoAgentsMd(t *testing.T) {
	for _, loc := range []Location{LocationGlobal, LocationLocal} {
		t.Run(string(loc), func(t *testing.T) {
			home := fakeHome(t)
			if loc == LocationLocal {
				dir := t.TempDir()
				t.Chdir(dir)
			}
			k := kiroTarget{}
			k.Install(loc, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

			if fileExists("AGENTS.md") {
				t.Fatalf("kiro install must not write a project-root AGENTS.md")
			}
			if fileExists(filepath.Join(home, ".kiro", "steering", "AGENTS.md")) {
				t.Fatalf("kiro install must not write a global steering AGENTS.md")
			}
			sharedDir, err := sharedSkillDirPath(loc)
			if err != nil {
				t.Fatalf("sharedSkillDirPath: %v", err)
			}
			if fileExists(sharedDir) {
				t.Fatalf("kiro must not write the shared .agents/skills/codegraph alias, found %s", sharedDir)
			}
		})
	}
}
