package agents

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	claudeassets "github.com/seanb4t/codegraph-go"
)

func TestGemini_ID(t *testing.T) {
	g := geminiTarget{}
	if g.ID() != Gemini {
		t.Fatalf("ID() = %v, want %v", g.ID(), Gemini)
	}
}

func TestGemini_SupportsBothLocations(t *testing.T) {
	g := geminiTarget{}
	if !g.SupportsLocation(LocationGlobal) || !g.SupportsLocation(LocationLocal) {
		t.Fatalf("gemini should support both global and local")
	}
}

func TestGemini_LocalInstall_InstructionsAtProjectRoot(t *testing.T) {
	fakeHome(t)
	dir := t.TempDir()
	t.Chdir(dir)

	g := geminiTarget{}
	g.Install(LocationLocal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

	rootPath := filepath.Join(dir, "GEMINI.md")
	if !fileExists(rootPath) {
		t.Fatalf("GEMINI.md was not written at project root")
	}
	wrongPath := filepath.Join(dir, ".gemini", "GEMINI.md")
	if fileExists(wrongPath) {
		t.Fatalf("GEMINI.md must not be written under ./.gemini/, only at project root")
	}

	configPath := filepath.Join(dir, ".gemini", "settings.json")
	got := readFile(t, configPath)
	if !strings.Contains(got, `"codegraph"`) {
		t.Fatalf("codegraph entry missing from local settings.json: %s", got)
	}
}

func TestGemini_GlobalInstall_InstructionsUnderGeminiDir(t *testing.T) {
	home := fakeHome(t)
	g := geminiTarget{}
	g.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

	instrPath := filepath.Join(home, ".gemini", "GEMINI.md")
	got := readFile(t, instrPath)
	if !strings.Contains(got, codegraphSectionStart) || !strings.Contains(got, "codegraph_explore") {
		t.Fatalf("unexpected GEMINI.md content: %s", got)
	}
}

func TestGemini_EntryHasTypeStdio(t *testing.T) {
	home := fakeHome(t)
	g := geminiTarget{}
	g.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

	configPath := filepath.Join(home, ".gemini", "settings.json")
	got := readFile(t, configPath)
	var decoded map[string]any
	if err := json.Unmarshal([]byte(got), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	mcpServers, _ := decoded["mcpServers"].(map[string]any)
	entry, _ := mcpServers["codegraph"].(map[string]any)
	if entry["type"] != "stdio" {
		t.Fatalf("gemini entry must carry type:stdio, got %v", entry)
	}
}

func TestGemini_RoundTrip_ByteInvariant(t *testing.T) {
	home := fakeHome(t)
	configPath := filepath.Join(home, ".gemini", "settings.json")
	pre := `{
  "mcpServers": {
    "other-server": { "command": "other-binary" }
  },
  "theme": "dark"
}
`
	writeFile(t, configPath, pre)

	g := geminiTarget{}
	g.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
	g.Uninstall(LocationGlobal)

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

	instrPath := filepath.Join(home, ".gemini", "GEMINI.md")
	if fileExists(instrPath) {
		t.Fatalf("GEMINI.md should have been removed entirely on uninstall (never existed pre-install)")
	}
}

// TestGemini_DescribePaths supersedes TestGemini_DescribePaths_ListsConfigAndInstructions:
// Gemini now declares SkillDirs: geminiSkillDirs (AGENT-10, D-06 correction
// (a)), so DescribePaths grows from 2 paths to 4 — config, instructions,
// the harness skill SKILL.md, and its sidecar manifest.
func TestGemini_DescribePaths(t *testing.T) {
	g := geminiTarget{}
	paths := g.DescribePaths(LocationGlobal)
	if len(paths) != 4 {
		t.Fatalf("want exactly 4 paths (config + instructions + harness SKILL.md + manifest), got %v", paths)
	}
	var sawConfig, sawInstr, sawSkillMD, sawManifest bool
	for _, p := range paths {
		switch {
		case strings.HasSuffix(p, filepath.Join(".gemini", "settings.json")):
			sawConfig = true
		case strings.HasSuffix(p, "GEMINI.md"):
			sawInstr = true
		case strings.HasSuffix(p, filepath.Join(".gemini", "skills", "codegraph", "SKILL.md")):
			sawSkillMD = true
		case strings.HasSuffix(p, filepath.Join(".gemini", "skills", "codegraph", ".codegraph-manifest.json")):
			sawManifest = true
		}
	}
	if !sawConfig || !sawInstr || !sawSkillMD || !sawManifest {
		t.Fatalf("DescribePaths missing an expected entry (config=%v instr=%v skillmd=%v manifest=%v): %v", sawConfig, sawInstr, sawSkillMD, sawManifest, paths)
	}
}

// TestGemini_Install_WritesHarnessSkillDir (AGENT-10, D-06 correction (a),
// google-gemini/gemini-cli docs/cli/skills.md, fetched 2026-09-18): Gemini
// CLI installs the codegraph skill at its OWN harness-specific directory
// (.gemini/skills/codegraph) — index 0 of geminiSkillDirs — never the
// shared .agents/skills/codegraph alias, which stays a documented read
// path only. GEMINI.md instructions are still written alongside it.
func TestGemini_Install_WritesHarnessSkillDir(t *testing.T) {
	for _, loc := range []Location{LocationGlobal, LocationLocal} {
		t.Run(string(loc), func(t *testing.T) {
			home := fakeHome(t)
			if loc == LocationLocal {
				dir := t.TempDir()
				t.Chdir(dir)
			}
			g := geminiTarget{}
			g.Install(loc, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

			harnessDir, err := g.Capabilities().WrittenSkillDir(loc)
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
			if len(m.Targets) != 1 || !containsTarget(m.Targets, Gemini) {
				t.Fatalf("manifest targets = %v, want exactly [gemini]", m.Targets)
			}

			sharedDir, err := sharedSkillDirPath(loc)
			if err != nil {
				t.Fatalf("sharedSkillDirPath: %v", err)
			}
			if fileExists(sharedDir) {
				t.Fatalf("gemini must not write the shared .agents/skills/codegraph alias, found %s", sharedDir)
			}

			var instrPath string
			if loc == LocationLocal {
				instrPath = "GEMINI.md"
			} else {
				instrPath = filepath.Join(home, ".gemini", "GEMINI.md")
			}
			if !fileExists(instrPath) {
				t.Fatalf("GEMINI.md instructions were not written: %s", instrPath)
			}

			g.Uninstall(loc)
			if fileExists(skillPath) {
				t.Fatalf("harness SKILL.md not removed after uninstall")
			}
			if fileExists(skillManifestPath(harnessDir)) {
				t.Fatalf("harness manifest not removed after uninstall")
			}
			if fileExists(harnessDir) {
				t.Fatalf("harness skill dir not swept after uninstall")
			}
			if strings.Contains(readFileOrEmpty(instrPath), codegraphSectionStart) {
				t.Fatalf("GEMINI.md still has codegraph's marker block after uninstall")
			}
		})
	}
}

// TestGemini_CorruptedManifestAtHarnessExclusiveDir_DoesNotFalselyAttributeClaude
// is CR-01's regression test (code review 05-REVIEW.md): Gemini's
// harness-exclusive `.gemini/skills/codegraph` directory (D-06) is one
// Claude never wrote to under any schema, so an unreadable/corrupt
// manifest there must never be read as owned by Claude — that fallback is
// justified only for Claude's own directory and the shared
// `.agents/skills/codegraph` directory reached through a D-17 symlink,
// neither of which applies here. Before the fix, manifestRequesters
// unconditionally falls back to [claude] on a read error, so the
// self-healed manifest wrongly lists claude as a co-owner, and Uninstall
// of gemini (the only real requester) leaves the package behind forever
// (D-08 violation) since nothing else ever uninstalls "claude" from this
// directory.
func TestGemini_CorruptedManifestAtHarnessExclusiveDir_DoesNotFalselyAttributeClaude(t *testing.T) {
	fakeHome(t)
	dir := t.TempDir()
	t.Chdir(dir)

	g := geminiTarget{}
	g.Install(LocationLocal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

	skillDir := filepath.Join(dir, ".gemini", "skills", "codegraph")
	manifestPath := skillManifestPath(skillDir)
	if !fileExists(manifestPath) {
		t.Fatalf("manifest not written by first install: %s", manifestPath)
	}
	if err := os.WriteFile(manifestPath, []byte("{not valid json"), 0o644); err != nil {
		t.Fatalf("corrupt manifest: %v", err)
	}

	// Re-run install over the corrupted manifest: self-healing must not
	// invent Claude as a co-owner of a directory Claude never wrote to.
	g.Install(LocationLocal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

	m, present, err := readManifest(manifestPath)
	if err != nil || !present {
		t.Fatalf("manifest not present after self-heal reinstall: present=%v err=%v", present, err)
	}
	if containsTarget(m.Targets, Claude) {
		t.Fatalf("CONFIRMED BUG: Claude falsely attributed as requester of %s: targets after self-heal reinstall = %v", skillDir, m.Targets)
	}
	if !targetSetEqual(m.Targets, []TargetID{Gemini}) {
		t.Fatalf("Targets after self-heal reinstall = %v, want [gemini]", m.Targets)
	}

	// Gemini is the only real requester — uninstalling it must fully
	// remove the package (D-08: deleted only when targets becomes empty).
	g.Uninstall(LocationLocal)

	if fileExists(skillDir) {
		t.Fatalf("CONFIRMED BUG: skill dir survived uninstall of its only real requester (gemini) due to phantom claude attribution")
	}
}
