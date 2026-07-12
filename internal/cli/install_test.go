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
