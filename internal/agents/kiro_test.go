package agents

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
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

func TestKiro_DescribePaths_ListsOnlyMcpConfig(t *testing.T) {
	k := kiroTarget{}
	paths := k.DescribePaths(LocationGlobal)
	if len(paths) != 1 {
		t.Fatalf("want exactly 1 path (the MCP config), got %v", paths)
	}
}
