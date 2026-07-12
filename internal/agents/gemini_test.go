package agents

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
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

func TestGemini_DescribePaths_ListsConfigAndInstructions(t *testing.T) {
	g := geminiTarget{}
	paths := g.DescribePaths(LocationGlobal)
	if len(paths) != 2 {
		t.Fatalf("want exactly 2 paths (config + instructions), got %v", paths)
	}
}
