package agents

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestClaude_ID(t *testing.T) {
	c := claudeTarget{}
	if c.ID() != Claude {
		t.Fatalf("ID() = %v, want %v", c.ID(), Claude)
	}
}

func TestClaude_SupportsBothLocations(t *testing.T) {
	c := claudeTarget{}
	if !c.SupportsLocation(LocationGlobal) {
		t.Fatalf("claude should support global")
	}
	if !c.SupportsLocation(LocationLocal) {
		t.Fatalf("claude should support local")
	}
}

func TestClaude_GlobalInstall_WritesConfigAndInstructions(t *testing.T) {
	home := fakeHome(t)
	c := claudeTarget{}

	result := c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
	if len(result.Files) == 0 {
		t.Fatalf("expected files touched, got none")
	}

	configPath := filepath.Join(home, ".claude.json")
	got := readFile(t, configPath)
	if !strings.Contains(got, `"codegraph"`) ||
		!strings.Contains(got, `"type": "stdio"`) ||
		!strings.Contains(got, "/usr/local/bin/codegraph") {
		t.Fatalf("unexpected ~/.claude.json content: %s", got)
	}

	instrPath := filepath.Join(home, ".claude", "CLAUDE.md")
	instr := readFile(t, instrPath)
	if !strings.Contains(instr, codegraphSectionStart) || !strings.Contains(instr, "codegraph_explore") {
		t.Fatalf("unexpected CLAUDE.md content: %s", instr)
	}
}

func TestClaude_GlobalInstall_InstructionsBodyMatchesCanonicalBlock(t *testing.T) {
	home := fakeHome(t)
	c := claudeTarget{}
	c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

	got := readFile(t, filepath.Join(home, ".claude", "CLAUDE.md"))
	want := codegraphInstructionsBlock + "\n"
	if got != want {
		t.Fatalf("CLAUDE.md body does not match canonical instructions block:\ngot=%q\nwant=%q", got, want)
	}
}

func TestClaude_LocalInstall_WritesMcpJSONNotClaudeJSON(t *testing.T) {
	fakeHome(t)
	dir := t.TempDir()
	t.Chdir(dir)
	c := claudeTarget{}

	c.Install(LocationLocal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

	mcpPath := filepath.Join(dir, ".mcp.json")
	if !fileExists(mcpPath) {
		t.Fatalf("./.mcp.json was not written")
	}
	claudeJSONPath := filepath.Join(dir, ".claude.json")
	if fileExists(claudeJSONPath) {
		t.Fatalf("./.claude.json should not have been created (Pitfall 3)")
	}
	got := readFile(t, mcpPath)
	if !strings.Contains(got, `"codegraph"`) {
		t.Fatalf("codegraph entry missing from .mcp.json: %s", got)
	}

	instrPath := filepath.Join(dir, ".claude", "CLAUDE.md")
	if !fileExists(instrPath) {
		t.Fatalf("./.claude/CLAUDE.md was not written for local scope")
	}
}

func TestClaude_GlobalInstall_AutoAllowAddsPermissionOnceIdempotent(t *testing.T) {
	home := fakeHome(t)
	c := claudeTarget{}
	opts := InstallOptions{ExecPath: "/usr/local/bin/codegraph", AutoAllow: true}

	c.Install(LocationGlobal, opts)
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	first := readFile(t, settingsPath)
	if strings.Count(first, "mcp__codegraph__*") != 1 {
		t.Fatalf("want exactly one allow token after first install, got: %s", first)
	}

	c.Install(LocationGlobal, opts)
	second := readFile(t, settingsPath)
	if strings.Count(second, "mcp__codegraph__*") != 1 {
		t.Fatalf("re-run duplicated the allow token: %s", second)
	}
}

func TestClaude_Install_NoAutoAllow_NoSettingsWrite(t *testing.T) {
	home := fakeHome(t)
	c := claudeTarget{}
	c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

	settingsPath := filepath.Join(home, ".claude", "settings.json")
	if fileExists(settingsPath) {
		t.Fatalf("settings.json should not be written without AutoAllow")
	}
}

func TestClaude_LocalInstall_MigratesLegacyClaudeJSON(t *testing.T) {
	fakeHome(t)
	dir := t.TempDir()
	t.Chdir(dir)

	legacyPath := filepath.Join(dir, ".claude.json")
	writeFile(t, legacyPath, `{
  "mcpServers": {
    "codegraph": { "type": "stdio", "command": "/old/codegraph", "args": ["serve", "--mcp"] },
    "other-server": { "command": "other-binary" }
  }
}
`)

	c := claudeTarget{}
	c.Install(LocationLocal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

	mcpJSON := readFile(t, filepath.Join(dir, ".mcp.json"))
	if !strings.Contains(mcpJSON, "/usr/local/bin/codegraph") {
		t.Fatalf(".mcp.json missing migrated/installed entry: %s", mcpJSON)
	}

	legacyAfter := readFile(t, legacyPath)
	if strings.Contains(legacyAfter, `"codegraph"`) {
		t.Fatalf("legacy ./.claude.json still has a codegraph entry: %s", legacyAfter)
	}
	if !strings.Contains(legacyAfter, "other-server") {
		t.Fatalf("legacy ./.claude.json lost an unrelated sibling key: %s", legacyAfter)
	}
}

func TestClaude_Uninstall_StripsLegacyClaudeJSONToo(t *testing.T) {
	fakeHome(t)
	dir := t.TempDir()
	t.Chdir(dir)

	legacyPath := filepath.Join(dir, ".claude.json")
	writeFile(t, legacyPath, `{
  "mcpServers": {
    "codegraph": { "type": "stdio", "command": "/old/codegraph", "args": ["serve", "--mcp"] }
  }
}
`)

	c := claudeTarget{}
	c.Install(LocationLocal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
	c.Uninstall(LocationLocal)

	if fileExists(filepath.Join(dir, ".mcp.json")) {
		mcpAfter := readFile(t, filepath.Join(dir, ".mcp.json"))
		if strings.Contains(mcpAfter, `"codegraph"`) {
			t.Fatalf("codegraph entry not removed from .mcp.json: %s", mcpAfter)
		}
	}
	if fileExists(legacyPath) {
		legacyAfter := readFile(t, legacyPath)
		if strings.Contains(legacyAfter, `"codegraph"`) {
			t.Fatalf("codegraph entry not stripped from legacy ./.claude.json: %s", legacyAfter)
		}
	}
}

func TestClaude_GlobalRoundTrip_ByteInvariantWithSibling(t *testing.T) {
	home := fakeHome(t)
	configPath := filepath.Join(home, ".claude.json")
	pre := `{
  "mcpServers": {
    "other-server": {
      "command": "other-binary",
      "args": ["run"]
    }
  },
  "someUnrelatedKey": true
}
`
	writeFile(t, configPath, pre)

	c := claudeTarget{}
	c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph", AutoAllow: true})
	c.Uninstall(LocationGlobal)

	got := readFile(t, configPath)
	var gotObj, wantObj map[string]any
	if err := json.Unmarshal([]byte(got), &gotObj); err != nil {
		t.Fatalf("post-round-trip config not valid JSON: %v", err)
	}
	if err := json.Unmarshal([]byte(pre), &wantObj); err != nil {
		t.Fatalf("bad fixture: %v", err)
	}
	if !jsonDeepEqual(gotObj, wantObj) {
		t.Fatalf("round trip not byte-invariant:\ngot=%s\nwant=%s", got, pre)
	}

	settingsPath := filepath.Join(home, ".claude", "settings.json")
	if fileExists(settingsPath) {
		settingsContent := readFile(t, settingsPath)
		if strings.Contains(settingsContent, "mcp__codegraph__*") {
			t.Fatalf("allow token not removed on uninstall: %s", settingsContent)
		}
	}

	instrPath := filepath.Join(home, ".claude", "CLAUDE.md")
	if fileExists(instrPath) {
		t.Fatalf("CLAUDE.md should have been removed entirely on uninstall (never existed pre-install)")
	}
}

func TestClaude_Uninstall_PreservesUnrelatedAllowEntries(t *testing.T) {
	home := fakeHome(t)
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	writeFile(t, settingsPath, `{
  "permissions": {
    "allow": ["Bash(git *)", "mcp__codegraph__*"]
  }
}
`)
	c := claudeTarget{}
	c.Uninstall(LocationGlobal)

	got := readFile(t, settingsPath)
	if strings.Contains(got, "mcp__codegraph__*") {
		t.Fatalf("allow token not removed: %s", got)
	}
	if !strings.Contains(got, "Bash(git *)") {
		t.Fatalf("unrelated allow entry lost: %s", got)
	}
}

func TestClaude_Detect_GlobalNotInstalled(t *testing.T) {
	fakeHome(t)
	c := claudeTarget{}
	got := c.Detect(LocationGlobal)
	if got.Installed || got.AlreadyConfigured {
		t.Fatalf("expected not installed/not configured on fresh fake home, got %+v", got)
	}
}

func TestClaude_Detect_AfterInstallReportsConfigured(t *testing.T) {
	fakeHome(t)
	c := claudeTarget{}
	c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
	got := c.Detect(LocationGlobal)
	if !got.AlreadyConfigured {
		t.Fatalf("expected AlreadyConfigured after install, got %+v", got)
	}
}

func TestClaude_DescribePaths_ListsConfigInstructionsAndSettings(t *testing.T) {
	c := claudeTarget{}
	paths := c.DescribePaths(LocationGlobal)
	if len(paths) < 2 {
		t.Fatalf("expected at least config + instructions paths, got %v", paths)
	}
}

func TestClaude_Install_ReRunIsByteIdempotent(t *testing.T) {
	home := fakeHome(t)
	c := claudeTarget{}
	opts := InstallOptions{ExecPath: "/usr/local/bin/codegraph", AutoAllow: true}

	c.Install(LocationGlobal, opts)
	configBefore := readFile(t, filepath.Join(home, ".claude.json"))
	instrBefore := readFile(t, filepath.Join(home, ".claude", "CLAUDE.md"))
	settingsBefore := readFile(t, filepath.Join(home, ".claude", "settings.json"))

	c.Install(LocationGlobal, opts)
	configAfter := readFile(t, filepath.Join(home, ".claude.json"))
	instrAfter := readFile(t, filepath.Join(home, ".claude", "CLAUDE.md"))
	settingsAfter := readFile(t, filepath.Join(home, ".claude", "settings.json"))

	if configBefore != configAfter {
		t.Fatalf("~/.claude.json changed on idempotent re-run:\nbefore=%q\nafter=%q", configBefore, configAfter)
	}
	if instrBefore != instrAfter {
		t.Fatalf("CLAUDE.md changed on idempotent re-run:\nbefore=%q\nafter=%q", instrBefore, instrAfter)
	}
	if settingsBefore != settingsAfter {
		t.Fatalf("settings.json changed on idempotent re-run:\nbefore=%q\nafter=%q", settingsBefore, settingsAfter)
	}
}
