package agents

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tailscale/hujson"
)

const opencodeFixtureWithComments = `{
  // top-level line comment
  "model": "anthropic/claude-sonnet-4-5", // trailing comment
  /* block comment describing the server block */
  "server": {
    "port": 4096
  }
}
`

func opencodeStandardize(t *testing.T, jsonc string) map[string]any {
	t.Helper()
	std, err := hujson.Standardize([]byte(jsonc))
	if err != nil {
		t.Fatalf("hujson.Standardize: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(std, &out); err != nil {
		t.Fatalf("json.Unmarshal standardized opencode config: %v\n%s", err, std)
	}
	return out
}

func TestOpencode_ID(t *testing.T) {
	o := opencodeTarget{}
	if o.ID() != Opencode {
		t.Fatalf("ID() = %v, want %v", o.ID(), Opencode)
	}
}

func TestOpencode_SupportsBothLocations(t *testing.T) {
	o := opencodeTarget{}
	if !o.SupportsLocation(LocationGlobal) || !o.SupportsLocation(LocationLocal) {
		t.Fatalf("opencode should support both global and local")
	}
}

func TestOpencode_ConfigDirResolution_XDGWinsOnEveryOS(t *testing.T) {
	home := fakeHome(t)
	xdg := filepath.Join(home, "custom-xdg")
	t.Setenv("XDG_CONFIG_HOME", xdg)

	got, err := resolveOpencodeConfigDir()
	if err != nil {
		t.Fatalf("resolveOpencodeConfigDir: %v", err)
	}
	if got != xdg {
		t.Fatalf("resolveOpencodeConfigDir() = %q, want XDG_CONFIG_HOME %q (Pitfall 4: no Windows special-case)", got, xdg)
	}
}

func TestOpencode_ConfigDirResolution_FallsBackToDotConfigWhenXDGUnset(t *testing.T) {
	home := fakeHome(t)
	t.Setenv("XDG_CONFIG_HOME", "")

	got, err := resolveOpencodeConfigDir()
	if err != nil {
		t.Fatalf("resolveOpencodeConfigDir: %v", err)
	}
	want := filepath.Join(home, ".config")
	if got != want {
		t.Fatalf("resolveOpencodeConfigDir() = %q, want %q", got, want)
	}
}

func TestOpencode_GlobalInstall_PreservesComments_ThenIdempotentReRun(t *testing.T) {
	home := fakeHome(t)
	configPath := filepath.Join(home, ".config", "opencode", "opencode.jsonc")
	writeFile(t, configPath, opencodeFixtureWithComments)

	o := opencodeTarget{}
	o.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

	afterFirst := readFile(t, configPath)
	if !strings.Contains(afterFirst, "top-level line comment") ||
		!strings.Contains(afterFirst, "trailing comment") ||
		!strings.Contains(afterFirst, "block comment describing the server block") {
		t.Fatalf("comments not preserved after install:\n%s", afterFirst)
	}

	o.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
	afterSecond := readFile(t, configPath)
	if afterSecond != afterFirst {
		t.Fatalf("re-run not idempotent:\nfirst=%q\nsecond=%q", afterFirst, afterSecond)
	}
	if !strings.Contains(afterSecond, "top-level line comment") ||
		!strings.Contains(afterSecond, "trailing comment") {
		t.Fatalf("comments lost on idempotent re-run:\n%s", afterSecond)
	}
}

func TestOpencode_Install_WritesCombinedCommandArrayAndSchema(t *testing.T) {
	home := fakeHome(t)
	o := opencodeTarget{}
	o.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

	configPath := filepath.Join(home, ".config", "opencode", "opencode.jsonc")
	decoded := opencodeStandardize(t, readFile(t, configPath))

	mcp, _ := decoded["mcp"].(map[string]any)
	if mcp == nil {
		t.Fatalf("mcp key missing: %v", decoded)
	}
	entry, _ := mcp["codegraph"].(map[string]any)
	if entry == nil {
		t.Fatalf("mcp.codegraph missing: %v", mcp)
	}
	cmd, _ := entry["command"].([]any)
	if len(cmd) != 3 {
		t.Fatalf("expected a combined [binary,...args] command array of length 3, got %v", entry["command"])
	}
	if cmd[0] != "/usr/local/bin/codegraph" {
		t.Fatalf("command[0] = %v, want ExecPath", cmd[0])
	}
	if cmd[1] != "serve" || cmd[2] != "--mcp" {
		t.Fatalf("unexpected command tail: %v", cmd)
	}
	if entry["enabled"] != true {
		t.Fatalf("enabled not true: %v", entry)
	}
	if entry["type"] != "local" {
		t.Fatalf("type not local: %v", entry)
	}
	if decoded["$schema"] == nil {
		t.Fatalf("$schema not written when absent")
	}
}

func TestOpencode_LocalInstall_WritesProjectRootFiles(t *testing.T) {
	fakeHome(t)
	dir := t.TempDir()
	t.Chdir(dir)
	o := opencodeTarget{}

	o.Install(LocationLocal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

	configPath := filepath.Join(dir, "opencode.jsonc")
	if !fileExists(configPath) {
		t.Fatalf("./opencode.jsonc was not written")
	}
	instrPath := filepath.Join(dir, "AGENTS.md")
	if !fileExists(instrPath) {
		t.Fatalf("./AGENTS.md was not written")
	}
	instr := readFile(t, instrPath)
	if !strings.Contains(instr, codegraphSectionStart) {
		t.Fatalf("AGENTS.md missing marker block: %s", instr)
	}
}

func TestOpencode_PrefersExistingJSONOverJSONC(t *testing.T) {
	home := fakeHome(t)
	dir := filepath.Join(home, ".config", "opencode")
	jsonPath := filepath.Join(dir, "opencode.json")
	writeFile(t, jsonPath, `{"model": "anthropic/claude-sonnet-4-5"}`)

	o := opencodeTarget{}
	o.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

	if !fileExists(jsonPath) {
		t.Fatalf("existing opencode.json should have been edited in place")
	}
	if fileExists(filepath.Join(dir, "opencode.jsonc")) {
		t.Fatalf("a fresh opencode.jsonc should not be created when opencode.json already exists")
	}
	got := readFile(t, jsonPath)
	if !strings.Contains(got, `"codegraph"`) {
		t.Fatalf("codegraph entry missing from opencode.json: %s", got)
	}
}

func TestOpencode_Uninstall_RemovesEntryPreservesComments(t *testing.T) {
	home := fakeHome(t)
	configPath := filepath.Join(home, ".config", "opencode", "opencode.jsonc")
	writeFile(t, configPath, opencodeFixtureWithComments)

	o := opencodeTarget{}
	o.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
	o.Uninstall(LocationGlobal)

	got := readFile(t, configPath)
	if strings.Contains(got, `"codegraph"`) {
		t.Fatalf("codegraph entry not removed: %s", got)
	}
	if !strings.Contains(got, "top-level line comment") || !strings.Contains(got, "trailing comment") {
		t.Fatalf("comments lost on uninstall: %s", got)
	}

	instrPath := filepath.Join(home, ".config", "opencode", "AGENTS.md")
	if fileExists(instrPath) {
		t.Fatalf("AGENTS.md should have been removed entirely on uninstall (never existed pre-install)")
	}
}

func TestOpencode_AppDataSweep_RemovesStaleEntryWhenDifferentFromXDG(t *testing.T) {
	home := fakeHome(t)
	xdg := filepath.Join(home, ".config")
	t.Setenv("XDG_CONFIG_HOME", xdg)
	appData := filepath.Join(home, "AppData", "Roaming")
	t.Setenv("APPDATA", appData)

	stalePath := filepath.Join(appData, "opencode", "opencode.jsonc")
	writeFile(t, stalePath, `{
  "mcp": { "codegraph": { "type": "local", "command": ["/old/codegraph", "serve", "--mcp"], "enabled": true } }
}
`)

	o := opencodeTarget{}
	o.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

	got := readFile(t, stalePath)
	if strings.Contains(got, `"codegraph"`) {
		t.Fatalf("stale %%APPDATA%% codegraph entry not swept: %s", got)
	}
}

func TestOpencode_AppDataSweep_SkippedWhenAppDataMatchesXDG(t *testing.T) {
	home := fakeHome(t)
	xdg := filepath.Join(home, "same-dir")
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("APPDATA", xdg)

	o := opencodeTarget{}
	o.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

	// If the sweep didn't skip (APPDATA == resolved XDG dir), it would sweep
	// the exact file Install just wrote, stripping the entry right back out.
	configPath := filepath.Join(xdg, "opencode", "opencode.jsonc")
	got := readFile(t, configPath)
	if !strings.Contains(got, `"codegraph"`) {
		t.Fatalf("sweep incorrectly ran against the just-installed config when APPDATA == resolved XDG dir: %s", got)
	}
}

func TestOpencode_Detect_AfterInstallReportsConfigured(t *testing.T) {
	fakeHome(t)
	o := opencodeTarget{}
	o.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
	got := o.Detect(LocationGlobal)
	if !got.AlreadyConfigured {
		t.Fatalf("expected AlreadyConfigured after install, got %+v", got)
	}
}

func TestOpencode_DescribePaths(t *testing.T) {
	o := opencodeTarget{}
	paths := o.DescribePaths(LocationGlobal)
	if len(paths) < 2 {
		t.Fatalf("expected at least config + instructions paths, got %v", paths)
	}
}
