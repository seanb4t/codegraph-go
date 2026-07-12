package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeHome points HOME (and every home-derived env var codegraph's agent
// targets consult — XDG_CONFIG_HOME for opencode, HERMES_HOME for Hermes)
// at fresh subdirectories of one isolated t.TempDir(), so every
// global-scope install/uninstall test in this file runs against a fake
// home rather than the real developer machine — mirrors
// internal/agents/testhelpers_test.go's fakeHome one package up. Also
// t.Chdir()s into a fresh project directory so local-scope tests
// (relative paths like ./.mcp.json, ./GEMINI.md) never touch this repo's
// own working tree. Returns the fake home root.
func fakeHome(t *testing.T) string {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("HERMES_HOME", filepath.Join(home, ".hermes"))

	project := t.TempDir()
	t.Chdir(project)

	return home
}

// readJSONMap reads path and decodes it as a generic JSON object, failing
// the test on any I/O or decode error — used to assert on written agent
// config shape without hand-rolling per-test unmarshal boilerplate.
func readJSONMap(t *testing.T, path string) map[string]any {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("readJSONMap(%s): %v", path, err)
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("readJSONMap(%s): unmarshal: %v", path, err)
	}
	return out
}

// readFileString reads path as a string, failing the test on error.
func readFileString(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("readFileString(%s): %v", path, err)
	}
	return string(data)
}

// TestInstall_TargetAll_WritesAndReportsPerAgent asserts `install
// --target all --location global` installs every registered target
// (all 8 support global) and reports one status line per agent plus at
// least one per-file "created:" action line.
func TestInstall_TargetAll_WritesAndReportsPerAgent(t *testing.T) {
	home := fakeHome(t)

	out, _, err := execCmd("install", "--target", "all", "--location", "global")
	if err != nil {
		t.Fatalf("install --target all: %v", err)
	}
	for _, name := range []string{"Claude Code:", "Cursor:", "Codex CLI:", "opencode:", "Gemini CLI:", "Antigravity:", "Hermes Agent:", "Kiro:"} {
		if !strings.Contains(out, name) {
			t.Errorf("expected output to mention %s, got:\n%s", name, out)
		}
	}
	if !strings.Contains(out, "created:") {
		t.Errorf("expected at least one 'created:' file action, got:\n%s", out)
	}

	claudeConfig := filepath.Join(home, ".claude.json")
	if _, statErr := os.Stat(claudeConfig); statErr != nil {
		t.Fatalf("expected %s to be written: %v", claudeConfig, statErr)
	}
}

// TestInstall_TargetCSV_SelectsExactlyThose asserts `install --target
// claude,cursor` configures exactly those two agents, not the full
// roster.
func TestInstall_TargetCSV_SelectsExactlyThose(t *testing.T) {
	fakeHome(t)

	out, _, err := execCmd("install", "--target", "claude,cursor", "--location", "global")
	if err != nil {
		t.Fatalf("install --target claude,cursor: %v", err)
	}
	if !strings.Contains(out, "Claude Code:") || !strings.Contains(out, "Cursor:") {
		t.Fatalf("expected Claude Code and Cursor in output, got:\n%s", out)
	}
	if strings.Contains(out, "Codex CLI:") || strings.Contains(out, "Gemini CLI:") {
		t.Fatalf("expected only the two selected agents in output, got:\n%s", out)
	}
}

// TestInstall_TargetCSV_UnknownID_ErrorsNoWrite asserts an unknown
// --target csv id surfaces a clear error and writes nothing — not even
// the valid ids earlier in the list (T-06-04-01, no partial write).
func TestInstall_TargetCSV_UnknownID_ErrorsNoWrite(t *testing.T) {
	home := fakeHome(t)

	_, _, err := execCmd("install", "--target", "claude,bogus", "--location", "global")
	if err == nil {
		t.Fatal("expected an error for an unknown --target id, got nil")
	}

	claudeConfig := filepath.Join(home, ".claude.json")
	if _, statErr := os.Stat(claudeConfig); !os.IsNotExist(statErr) {
		t.Fatalf("expected no partial write on unknown-id error, but %s exists", claudeConfig)
	}
}

// TestInstall_TargetNone_InstallsNothing asserts `install --target none`
// writes no files and says so.
func TestInstall_TargetNone_InstallsNothing(t *testing.T) {
	home := fakeHome(t)

	out, _, err := execCmd("install", "--target", "none")
	if err != nil {
		t.Fatalf("install --target none: %v", err)
	}
	if !strings.Contains(out, "no agents selected") {
		t.Fatalf("expected a 'no agents selected' message, got:\n%s", out)
	}

	claudeConfig := filepath.Join(home, ".claude.json")
	if _, statErr := os.Stat(claudeConfig); !os.IsNotExist(statErr) {
		t.Fatalf("expected no files written for --target none, but %s exists", claudeConfig)
	}
}

// TestInstall_NoTargetNonTTY_ResolvesAutoWithoutBlocking asserts a plain
// `install` with no --target and no TTY (execCmd always wires stdin to a
// strings.Reader, never os.Stdin) resolves straight to auto without
// reading stdin — an empty/short reader would hang forever on a real
// interactive read if the no-TTY fallback were broken (D-03, T-06-04-02).
func TestInstall_NoTargetNonTTY_ResolvesAutoWithoutBlocking(t *testing.T) {
	home := fakeHome(t)

	out, _, err := execCmd("install")
	if err != nil {
		t.Fatalf("install (no --target): %v", err)
	}
	// Zero agents are ever "detected" in a fresh fake home, so auto falls
	// back to just Claude (registry.ResolveTargetFlag's least-surprise
	// fallback).
	if !strings.Contains(out, "Claude Code:") {
		t.Fatalf("expected auto fallback to configure Claude Code, got:\n%s", out)
	}

	claudeConfig := filepath.Join(home, ".claude.json")
	if _, statErr := os.Stat(claudeConfig); statErr != nil {
		t.Fatalf("expected %s to be written: %v", claudeConfig, statErr)
	}
}

// TestInstall_ExecPathAppearsInWrittenConfig asserts the running test
// binary's own os.Executable() path lands in the written MCP config's
// command field (D-04) — not a bare "codegraph" PATH guess.
func TestInstall_ExecPathAppearsInWrittenConfig(t *testing.T) {
	home := fakeHome(t)

	if _, _, err := execCmd("install", "--target", "claude", "--location", "global"); err != nil {
		t.Fatalf("install --target claude: %v", err)
	}

	wantExec, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable(): %v", err)
	}

	cfg := readJSONMap(t, filepath.Join(home, ".claude.json"))
	mcpServers, _ := cfg["mcpServers"].(map[string]any)
	entry, _ := mcpServers["codegraph"].(map[string]any)
	if entry["command"] != wantExec {
		t.Fatalf("codegraph.command = %v, want %v", entry["command"], wantExec)
	}
}

// TestInstall_Idempotent_RerunReportsUnchanged asserts re-running install
// twice reports only "unchanged" the second time — a byte-level no-op
// (D-07).
func TestInstall_Idempotent_RerunReportsUnchanged(t *testing.T) {
	fakeHome(t)

	if _, _, err := execCmd("install", "--target", "claude", "--location", "global"); err != nil {
		t.Fatalf("install (1st run): %v", err)
	}

	out, _, err := execCmd("install", "--target", "claude", "--location", "global")
	if err != nil {
		t.Fatalf("install (2nd run): %v", err)
	}
	if strings.Contains(out, "created:") || strings.Contains(out, "updated:") {
		t.Fatalf("expected re-run to report only unchanged files, got:\n%s", out)
	}
	if !strings.Contains(out, "Claude Code: unchanged") {
		t.Fatalf("expected top-level 'unchanged' status, got:\n%s", out)
	}
}

// TestInstall_AutoAllow_TogglesPermission asserts --auto-allow appends
// mcp__codegraph__* to Claude Code's settings.json permissions.allow list
// (D-05); the flag defaults to false and is a no-op for every other
// target (only asserted for Claude here, the one target that has the
// concept).
func TestInstall_AutoAllow_TogglesPermission(t *testing.T) {
	home := fakeHome(t)

	if _, _, err := execCmd("install", "--target", "claude", "--location", "global", "--auto-allow"); err != nil {
		t.Fatalf("install --auto-allow: %v", err)
	}

	settings := readJSONMap(t, filepath.Join(home, ".claude", "settings.json"))
	permissions, _ := settings["permissions"].(map[string]any)
	allow, _ := permissions["allow"].([]any)
	found := false
	for _, v := range allow {
		if v == "mcp__codegraph__*" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected permissions.allow to contain mcp__codegraph__*, got %v", allow)
	}
}

// TestInstall_InvalidLocation_Errors asserts an unrecognized --location
// value is a clear error, not a silent misconfiguration.
func TestInstall_InvalidLocation_Errors(t *testing.T) {
	fakeHome(t)

	if _, _, err := execCmd("install", "--location", "bogus"); err == nil {
		t.Fatal("expected an error for an invalid --location value")
	}
}

// TestInstall_WriteFailure_ReportsErrorAndNonZeroExit is the CR-01
// regression test: a hard I/O failure while writing an agent's config
// (simulated here by seeding ~/.claude.json as a directory, so the write
// helper's read-before-write step fails with a genuine, non-"not exist"
// error) must surface as a non-zero exit and an "error:" line in the
// per-agent report — never look identical to a silent no-op/"unchanged".
func TestInstall_WriteFailure_ReportsErrorAndNonZeroExit(t *testing.T) {
	home := fakeHome(t)
	claudeConfig := filepath.Join(home, ".claude.json")
	if err := os.Mkdir(claudeConfig, 0o755); err != nil {
		t.Fatalf("seed directory-in-place-of-file: %v", err)
	}

	out, _, err := execCmd("install", "--target", "claude", "--location", "global")
	if err == nil {
		t.Fatalf("expected install to return a non-nil error when a target's config write fails; output:\n%s", out)
	}
	if !strings.Contains(out, "error:") {
		t.Fatalf("expected install output to include an 'error:' line, got:\n%s", out)
	}
}

// TestUninstall_ReportsRemovedAndNotConfigured asserts uninstall reports
// "removed" for an agent install actually configured and
// "not-configured" for one that was never touched (D-08).
func TestUninstall_ReportsRemovedAndNotConfigured(t *testing.T) {
	fakeHome(t)

	if _, _, err := execCmd("install", "--target", "claude", "--location", "global"); err != nil {
		t.Fatalf("install --target claude: %v", err)
	}

	out, _, err := execCmd("uninstall", "--target", "claude,cursor", "--location", "global")
	if err != nil {
		t.Fatalf("uninstall --target claude,cursor: %v", err)
	}
	if !strings.Contains(out, "Claude Code: removed") {
		t.Fatalf("expected 'Claude Code: removed', got:\n%s", out)
	}
	if !strings.Contains(out, "Cursor: not-configured") {
		t.Fatalf("expected 'Cursor: not-configured', got:\n%s", out)
	}
}

// TestUninstall_ReportsUnsupportedForWrongLocation asserts uninstall
// reports "unsupported" (never an error) for a target/location
// combination the agent doesn't support (Codex is global-only).
func TestUninstall_ReportsUnsupportedForWrongLocation(t *testing.T) {
	fakeHome(t)

	out, _, err := execCmd("uninstall", "--target", "codex", "--location", "local")
	if err != nil {
		t.Fatalf("uninstall --target codex --location local: %v", err)
	}
	if !strings.Contains(out, "Codex CLI: unsupported") {
		t.Fatalf("expected 'Codex CLI: unsupported', got:\n%s", out)
	}
}

// TestUninstall_NeverInstalledAgent_NoError asserts uninstalling an agent
// that was never configured is a clean not-configured status, never an
// error (D-08).
func TestUninstall_NeverInstalledAgent_NoError(t *testing.T) {
	fakeHome(t)

	out, _, err := execCmd("uninstall", "--target", "hermes", "--location", "global")
	if err != nil {
		t.Fatalf("uninstall --target hermes (never installed): unexpected error: %v", err)
	}
	if !strings.Contains(out, "Hermes Agent: not-configured") {
		t.Fatalf("expected 'Hermes Agent: not-configured', got:\n%s", out)
	}
}

// TestUninstall_NoTargetDefaultsToAllWithoutPrompting asserts a plain
// `uninstall` with no --target and no TTY operates on every registered
// target without prompting (D-08's parity fallback — a destructive
// reversal command defaults to "all", not an interactive picker).
func TestUninstall_NoTargetDefaultsToAllWithoutPrompting(t *testing.T) {
	fakeHome(t)

	out, _, err := execCmd("uninstall")
	if err != nil {
		t.Fatalf("uninstall (no --target): %v", err)
	}
	for _, name := range []string{"Claude Code:", "Cursor:", "Kiro:"} {
		if !strings.Contains(out, name) {
			t.Fatalf("expected default uninstall to cover %s, got:\n%s", name, out)
		}
	}
}

// TestInstallUninstallRoundTrip_PreservesSiblingEntry asserts install
// then uninstall via the real commands preserves an unrelated sibling
// mcpServers entry untouched (D-07, D-08) — the command-level analog of
// 06-02/06-03's per-target round-trip tests.
func TestInstallUninstallRoundTrip_PreservesSiblingEntry(t *testing.T) {
	home := fakeHome(t)

	claudeConfig := filepath.Join(home, ".claude.json")
	seed := map[string]any{
		"mcpServers": map[string]any{
			"other-server": map[string]any{"command": "other-binary"},
		},
	}
	seedData, err := json.MarshalIndent(seed, "", "  ")
	if err != nil {
		t.Fatalf("marshal seed: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(claudeConfig), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(claudeConfig, seedData, 0o644); err != nil {
		t.Fatalf("seed write: %v", err)
	}

	if _, _, err := execCmd("install", "--target", "claude", "--location", "global"); err != nil {
		t.Fatalf("install --target claude: %v", err)
	}

	afterInstall := readJSONMap(t, claudeConfig)
	mcpServers, _ := afterInstall["mcpServers"].(map[string]any)
	if _, ok := mcpServers["other-server"]; !ok {
		t.Fatalf("expected other-server entry to survive install, got: %v", mcpServers)
	}
	if _, ok := mcpServers["codegraph"]; !ok {
		t.Fatalf("expected codegraph entry after install, got: %v", mcpServers)
	}

	if _, _, err := execCmd("uninstall", "--target", "claude", "--location", "global"); err != nil {
		t.Fatalf("uninstall --target claude: %v", err)
	}

	afterUninstall := readJSONMap(t, claudeConfig)
	mcpServersAfter, _ := afterUninstall["mcpServers"].(map[string]any)
	if _, ok := mcpServersAfter["codegraph"]; ok {
		t.Fatalf("expected codegraph entry removed after uninstall, got: %v", mcpServersAfter)
	}
	other, ok := mcpServersAfter["other-server"].(map[string]any)
	if !ok || other["command"] != "other-binary" {
		t.Fatalf("expected other-server entry preserved untouched after uninstall, got: %v", mcpServersAfter)
	}
}

// TestInstallUninstallRoundTrip_TempHome_RestoresPreInstallState is the
// automated substitute for this plan's live-agent-verify checkpoint (a
// real agent handshake can't be exercised in this environment): it drives
// a full install→uninstall round trip against a throwaway $HOME and
// asserts the written config/instructions files' shape and that
// uninstall restores pre-install state modulo the CodeGraph section
// (D-01). The residual live-agent handshake itself remains a manual
// follow-up — see SUMMARY.md.
func TestInstallUninstallRoundTrip_TempHome_RestoresPreInstallState(t *testing.T) {
	home := fakeHome(t)

	claudeConfig := filepath.Join(home, ".claude.json")
	claudeInstructions := filepath.Join(home, ".claude", "CLAUDE.md")

	if _, statErr := os.Stat(claudeConfig); !os.IsNotExist(statErr) {
		t.Fatalf("precondition: %s should not exist yet", claudeConfig)
	}

	if _, _, err := execCmd("install", "--target", "auto", "--location", "global"); err != nil {
		t.Fatalf("install --target auto: %v", err)
	}
	if _, statErr := os.Stat(claudeConfig); statErr != nil {
		t.Fatalf("expected %s to exist after install: %v", claudeConfig, statErr)
	}
	if _, statErr := os.Stat(claudeInstructions); statErr != nil {
		t.Fatalf("expected %s to exist after install: %v", claudeInstructions, statErr)
	}
	instructions := readFileString(t, claudeInstructions)
	if !strings.Contains(instructions, "CODEGRAPH_START") {
		t.Fatalf("expected instructions file to carry the marker block, got:\n%s", instructions)
	}
	if !strings.Contains(instructions, "codegraph_explore") {
		t.Fatalf("expected instructions file to reference codegraph_explore, got:\n%s", instructions)
	}

	if _, _, err := execCmd("uninstall", "--target", "auto", "--location", "global"); err != nil {
		t.Fatalf("uninstall --target auto: %v", err)
	}

	// The marker-fenced instructions file had nothing but the codegraph
	// block, so removing it restores the file to its pre-install
	// (nonexistent) state entirely (shared.go's removeMarkedSection).
	if _, statErr := os.Stat(claudeInstructions); !os.IsNotExist(statErr) {
		t.Fatalf("expected %s to be removed entirely after uninstall, stat err: %v", claudeInstructions, statErr)
	}

	// The MCP config file itself remains (D-01: "pre-install bytes modulo
	// the CodeGraph section") but must carry no codegraph/mcpServers
	// trace, since it was codegraph-only.
	final := readJSONMap(t, claudeConfig)
	if _, ok := final["mcpServers"]; ok {
		t.Fatalf("expected mcpServers removed entirely (was codegraph-only), got: %v", final)
	}
}
