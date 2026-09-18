package agents

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	claudeassets "github.com/seanb4t/codegraph-go"
)

func TestAntigravity_ID(t *testing.T) {
	a := antigravityTarget{}
	if a.ID() != Antigravity {
		t.Fatalf("ID() = %v, want %v", a.ID(), Antigravity)
	}
}

func TestAntigravity_GlobalOnly(t *testing.T) {
	a := antigravityTarget{}
	if a.SupportsLocation(LocationLocal) {
		t.Fatalf("antigravity must be global-only (SupportsLocation(local) must be false)")
	}
	if !a.SupportsLocation(LocationGlobal) {
		t.Fatalf("antigravity must support global")
	}
}

func TestAntigravity_Install_EntryHasNoTypeField(t *testing.T) {
	home := fakeHome(t)
	a := antigravityTarget{}
	a.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

	configPath := filepath.Join(home, ".gemini", "config", "mcp_config.json")
	got := readFile(t, configPath)

	var decoded map[string]any
	if err := json.Unmarshal([]byte(got), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	mcpServers, _ := decoded["mcpServers"].(map[string]any)
	entry, _ := mcpServers["codegraph"].(map[string]any)
	if entry == nil {
		t.Fatalf("codegraph entry missing: %s", got)
	}
	if _, hasType := entry["type"]; hasType {
		t.Fatalf("antigravity entry must not carry a type field: %v", entry)
	}
	if entry["command"] != "/usr/local/bin/codegraph" {
		t.Fatalf("unexpected command: %v", entry["command"])
	}
}

func TestAntigravity_Install_NoOpForLocal(t *testing.T) {
	fakeHome(t)
	dir := t.TempDir()
	t.Chdir(dir)

	a := antigravityTarget{}
	result := a.Install(LocationLocal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
	if len(result.Files) != 0 {
		t.Fatalf("expected no files touched for unsupported local scope, got %v", result.Files)
	}
}

func TestAntigravity_Install_SweepsStaleLegacyEntry(t *testing.T) {
	home := fakeHome(t)
	legacyPath := filepath.Join(home, ".gemini", "antigravity", "mcp_config.json")
	writeFile(t, legacyPath, `{
  "mcpServers": {
    "codegraph": { "command": "/old/codegraph", "args": ["serve", "--mcp"] }
  }
}
`)

	a := antigravityTarget{}
	a.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

	unifiedPath := filepath.Join(home, ".gemini", "config", "mcp_config.json")
	got := readFile(t, unifiedPath)
	if !strings.Contains(got, "/usr/local/bin/codegraph") {
		t.Fatalf("unified config missing the installed entry: %s", got)
	}

	legacyAfter, err := readJSONFile(legacyPath)
	if err != nil {
		t.Fatalf("re-read legacy: %v", err)
	}
	if _, ok := legacyAfter["mcpServers"]; ok {
		t.Fatalf("stale legacy entry not swept: %v", legacyAfter)
	}
}

func TestAntigravity_RoundTrip_ByteInvariantWithSibling(t *testing.T) {
	home := fakeHome(t)
	configPath := filepath.Join(home, ".gemini", "config", "mcp_config.json")
	pre := `{
  "mcpServers": {
    "other-server": { "command": "other-binary" }
  }
}
`
	writeFile(t, configPath, pre)

	a := antigravityTarget{}
	a.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
	a.Uninstall(LocationGlobal)

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

func TestAntigravity_NoInstructionsFileWritten(t *testing.T) {
	home := fakeHome(t)
	a := antigravityTarget{}
	a.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

	if fileExists(filepath.Join(home, ".gemini", "GEMINI.md")) {
		t.Fatalf("antigravity must not write GEMINI.md itself (only the Gemini target does)")
	}
}

// TestAntigravity_Install_UnifiedWriteFailure_PreservesLegacyEntryNoMarker
// is the CR-02 regression test: if the write of the migrated entry into
// the unified config fails, the legacy entry must NOT be removed and the
// ".migrated" marker must NOT be written — otherwise a partial-write
// failure would permanently record "migration done" while the codegraph
// entry exists in neither file (silent data loss with no rollback). The
// failure is simulated by chmod'ing the unified config's parent directory
// read-only AFTER it exists (so the pre-migration fileExists(unified)
// check still sees "no unified file yet" and the sweep proceeds), which
// makes atomicWriteFile's os.CreateTemp fail with a genuine permission
// error for both the migration write and the final writeMcpEntry call.
func TestAntigravity_Install_UnifiedWriteFailure_PreservesLegacyEntryNoMarker(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission-bit write-failure simulation is POSIX-specific")
	}
	if os.Geteuid() == 0 {
		t.Skip("running as root bypasses permission bits")
	}

	home := fakeHome(t)
	legacyPath := filepath.Join(home, ".gemini", "antigravity", "mcp_config.json")
	writeFile(t, legacyPath, `{
  "mcpServers": {
    "codegraph": { "command": "/old/codegraph", "args": ["serve", "--mcp"] }
  }
}
`)

	unifiedDir := filepath.Join(home, ".gemini", "config")
	if err := os.MkdirAll(unifiedDir, 0o755); err != nil {
		t.Fatalf("seed unified dir: %v", err)
	}
	if err := os.Chmod(unifiedDir, 0o500); err != nil {
		t.Fatalf("chmod unified dir read-only: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(unifiedDir, 0o755) })

	a := antigravityTarget{}
	result := a.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

	if len(result.Errors) == 0 {
		t.Fatalf("expected Install to report an error on unified write failure, got none: %+v", result)
	}

	legacyAfter, err := readJSONFile(legacyPath)
	if err != nil {
		t.Fatalf("re-read legacy: %v", err)
	}
	mcpServers, _ := legacyAfter["mcpServers"].(map[string]any)
	if _, ok := mcpServers["codegraph"]; !ok {
		t.Fatalf("legacy entry was removed despite the unified write failing (CR-02 data loss): %v", legacyAfter)
	}

	markerPath := filepath.Join(unifiedDir, ".migrated")
	if fileExists(markerPath) {
		t.Fatalf("migrated marker was written despite the unified write failing (CR-02): %s exists", markerPath)
	}
}

func TestAntigravity_DescribePaths_GlobalOnly(t *testing.T) {
	a := antigravityTarget{}
	if paths := a.DescribePaths(LocationLocal); len(paths) != 0 {
		t.Fatalf("DescribePaths(local) should be empty for a global-only target, got %v", paths)
	}
}

// TestAntigravity_Install_WritesCliSkillDir (AGENT-07, D-06 corrections
// (b)(c); [CITED: antigravity.google/docs/skills.md, fetched 2026-09-18]):
// Antigravity installs the codegraph skill at the `agy` CLI's documented
// global skill directory — index 0 of antigravitySkillDirs, the surface
// 05-06's live session exercises — never the [ASSUMED] 2.0/IDE path, the
// shared .agents/skills alias (workspace-scope only for Antigravity, which
// this global-only target never writes), or a new AGENTS.md (D-06(c):
// Antigravity's instructions arrive only via Gemini's own
// ~/.gemini/GEMINI.md write).
func TestAntigravity_Install_WritesCliSkillDir(t *testing.T) {
	home := fakeHome(t)
	a := antigravityTarget{}
	a.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

	cliDir := filepath.Join(home, ".gemini", "antigravity-cli", "skills", "codegraph")
	skillPath := filepath.Join(cliDir, "SKILL.md")
	want, err := claudeassets.SkillMarkdown()
	if err != nil {
		t.Fatalf("claudeassets.SkillMarkdown: %v", err)
	}
	if got := readFile(t, skillPath); got != string(want) {
		t.Fatalf("CLI skill SKILL.md at %s does not match the embed", skillPath)
	}
	m, present, err := readManifest(skillManifestPath(cliDir))
	if err != nil || !present {
		t.Fatalf("expected a manifest at %s (present=%v err=%v)", cliDir, present, err)
	}
	if len(m.Targets) != 1 || !containsTarget(m.Targets, Antigravity) {
		t.Fatalf("manifest targets = %v, want exactly [antigravity]", m.Targets)
	}

	ideDir := filepath.Join(home, ".gemini", "config", "skills", "codegraph")
	if fileExists(ideDir) {
		t.Fatalf("antigravity must not write the [ASSUMED] 2.0/IDE skill path, found %s", ideDir)
	}
	if fileExists(filepath.Join(home, ".agents", "skills")) {
		t.Fatalf("antigravity must not write under the shared .agents/skills alias")
	}
	if fileExists(filepath.Join(home, ".gemini", "AGENTS.md")) {
		t.Fatalf("antigravity must not write a new AGENTS.md")
	}

	a.Uninstall(LocationGlobal)
	if fileExists(skillPath) {
		t.Fatalf("CLI SKILL.md not removed after uninstall")
	}
	if fileExists(skillManifestPath(cliDir)) {
		t.Fatalf("CLI manifest not removed after uninstall")
	}
	if fileExists(cliDir) {
		t.Fatalf("CLI skill dir not swept after uninstall")
	}
}

// TestAntigravity_ReadOnlySkillDirDocumented (D-06(b)): the 2.0/IDE skill
// path is documented via Capabilities().ReadOnlySkillDirs but never
// written by Install — an [ASSUMED] read path with no live surface this
// milestone's herdr-driven method can verify (it drives the CLI, not a
// GUI IDE).
func TestAntigravity_ReadOnlySkillDirDocumented(t *testing.T) {
	home := fakeHome(t)
	a := antigravityTarget{}
	got, err := a.Capabilities().ReadOnlySkillDirs(LocationGlobal)
	if err != nil {
		t.Fatalf("ReadOnlySkillDirs: %v", err)
	}
	want := filepath.Join(home, ".gemini", "config", "skills", "codegraph")
	if len(got) != 1 || got[0] != want {
		t.Fatalf("ReadOnlySkillDirs(global) = %v, want [%s]", got, want)
	}
}

// TestAntigravity_Local_WritesNothing (D-06): Antigravity is global-only;
// Install(local) and Uninstall(local) must be complete no-ops, creating
// nothing on disk.
func TestAntigravity_Local_WritesNothing(t *testing.T) {
	fakeHome(t)
	dir := t.TempDir()
	t.Chdir(dir)

	a := antigravityTarget{}
	installResult := a.Install(LocationLocal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
	if len(installResult.Files) != 0 || len(installResult.Errors) != 0 {
		t.Fatalf("Install(local) = %+v, want empty", installResult)
	}
	uninstallResult := a.Uninstall(LocationLocal)
	if len(uninstallResult.Files) != 0 || len(uninstallResult.Errors) != 0 {
		t.Fatalf("Uninstall(local) = %+v, want empty", uninstallResult)
	}
}
