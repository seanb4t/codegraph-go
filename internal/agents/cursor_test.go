package agents

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

func TestCursor_DescribePaths_ListsOnlyMcpConfig(t *testing.T) {
	c := cursorTarget{}
	paths := c.DescribePaths(LocationGlobal)
	if len(paths) != 1 {
		t.Fatalf("want exactly 1 path (the MCP config), got %v", paths)
	}
	if !strings.HasSuffix(paths[0], filepath.Join(".cursor", "mcp.json")) {
		t.Fatalf("unexpected DescribePaths entry: %v", paths)
	}
}
