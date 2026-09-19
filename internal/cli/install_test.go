package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/agents"
)

// installSGRSequence is a local copy of present/ansistrip_test.go's SGR
// stripper — a _test.go helper cannot be imported across packages, so
// every package with its own styled-output test carries this one-line
// regexp rather than exporting it out of present.
var installSGRSequence = regexp.MustCompile("\x1b\\[[0-9;]*m")

// stripInstallSGR removes every SGR escape sequence from s.
func stripInstallSGR(s string) string {
	return installSGRSequence.ReplaceAllString(s, "")
}

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

// TestInstall_StyledOutputStripsToPlain asserts printAgentResults' styled
// branch (--color=always) strips back to byte-identical plain output —
// both for a normal two-target install (TestInstall_TargetCSV_
// SelectsExactlyThose's shape) and for the write-failure/non-zero-exit
// shape (TestInstall_WriteFailure_ReportsErrorAndNonZeroExit's fixture) —
// without weakening errors.Join's non-nil-error contract (CR-01). Each
// comparison runs plain and styled against their OWN fresh fakeHome (two
// independent t.TempDir()s), so the two absolute home paths embedded in
// the output are normalized away before comparing; everything else must
// match byte-for-byte once ANSI is stripped.
func TestInstall_StyledOutputStripsToPlain(t *testing.T) {
	plainHome := fakeHome(t)
	plain, _, err := execCmd("install", "--target", "claude,cursor", "--location", "global")
	if err != nil {
		t.Fatalf("install --target claude,cursor (plain): %v", err)
	}
	plainNorm := strings.ReplaceAll(plain, plainHome, "<HOME>")

	styledHome := fakeHome(t)
	styled, _, err := execCmd("install", "--target", "claude,cursor", "--location", "global", "--color=always")
	if err != nil {
		t.Fatalf("install --target claude,cursor (styled): %v", err)
	}
	if !strings.Contains(styled, "\x1b[") {
		t.Fatalf("expected styled output to contain an ESC byte, got:\n%q", styled)
	}
	styledNorm := strings.ReplaceAll(stripInstallSGR(styled), styledHome, "<HOME>")
	if styledNorm != plainNorm {
		t.Fatalf("stripped+normalized styled output does not equal plain:\nplain:  %q\nstyled: %q", plainNorm, styledNorm)
	}

	// Write-failure shape: a non-nil error, in both plain and styled, with
	// a stripped "  error:" line equal once normalized.
	plainFailHome := fakeHome(t)
	plainFailConfig := filepath.Join(plainFailHome, ".claude.json")
	if err := os.Mkdir(plainFailConfig, 0o755); err != nil {
		t.Fatalf("seed directory-in-place-of-file (plain): %v", err)
	}
	plainFail, _, plainFailErr := execCmd("install", "--target", "claude", "--location", "global")
	if plainFailErr == nil {
		t.Fatalf("expected plain write-failure run to return a non-nil error; output:\n%s", plainFail)
	}
	plainFailNorm := strings.ReplaceAll(plainFail, plainFailHome, "<HOME>")

	styledFailHome := fakeHome(t)
	styledFailConfig := filepath.Join(styledFailHome, ".claude.json")
	if err := os.Mkdir(styledFailConfig, 0o755); err != nil {
		t.Fatalf("seed directory-in-place-of-file (styled): %v", err)
	}
	styledFail, _, styledFailErr := execCmd("install", "--target", "claude", "--location", "global", "--color=always")
	if styledFailErr == nil {
		t.Fatalf("expected styled write-failure run to return a non-nil error; output:\n%s", styledFail)
	}
	if !strings.Contains(styledFail, "\x1b[") {
		t.Fatalf("expected styled write-failure output to contain an ESC byte, got:\n%q", styledFail)
	}
	styledFailNorm := strings.ReplaceAll(stripInstallSGR(styledFail), styledFailHome, "<HOME>")
	if styledFailNorm != plainFailNorm {
		t.Fatalf("stripped+normalized styled write-failure output does not equal plain:\nplain:  %q\nstyled: %q", plainFailNorm, styledFailNorm)
	}
	if !strings.Contains(styledFailNorm, "error:") {
		t.Fatalf("expected normalized styled write-failure output to include an 'error:' line, got:\n%s", styledFailNorm)
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
// combination the agent doesn't support (Hermes is global-only; Codex
// gained local scope in D-09 so it is no longer this test's example).
func TestUninstall_ReportsUnsupportedForWrongLocation(t *testing.T) {
	fakeHome(t)

	out, _, err := execCmd("uninstall", "--target", "hermes", "--location", "local")
	if err != nil {
		t.Fatalf("uninstall --target hermes --location local: %v", err)
	}
	if !strings.Contains(out, "Hermes Agent: unsupported") {
		t.Fatalf("expected 'Hermes Agent: unsupported', got:\n%s", out)
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
// target without prompting (D-08's destructive-reversal default — a
// destructive reversal command defaults to "all", not an interactive
// picker).
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

// withStubbedPicker forces the interactiveAllowed/runAgentPicker
// package-level seams for the duration of one test — interactiveAllowed
// always reports allowed (as if on a real TTY), and picker stands in for
// tui.RunAgentPicker without ever constructing a real tea.Program. Both
// seams are restored on cleanup.
func withStubbedPicker(t *testing.T, picker func(*cobra.Command, agents.Location) ([]agents.AgentTarget, error)) {
	t.Helper()
	origAllowed, origPicker := interactiveAllowed, runAgentPicker
	t.Cleanup(func() {
		interactiveAllowed = origAllowed
		runAgentPicker = origPicker
	})
	interactiveAllowed = func(*cobra.Command) bool { return true }
	runAgentPicker = picker
}

// TestInstall_Yes_ShortCircuitsBeforeInteractiveBranch is the Pitfall-6
// regression: -y must short-circuit to auto BEFORE the TTY branch, not
// merely skip rendering the picker. interactiveAllowed is forced to report
// true (as if on a real TTY) and runAgentPicker is forced to fail the test
// if ever invoked — if -y didn't check first in the switch, this would
// exercise the interactive branch instead of resolving straight to auto.
func TestInstall_Yes_ShortCircuitsBeforeInteractiveBranch(t *testing.T) {
	home := fakeHome(t)
	withStubbedPicker(t, func(*cobra.Command, agents.Location) ([]agents.AgentTarget, error) {
		t.Fatal("runAgentPicker must never be called when -y is set")
		return nil, nil
	})

	out, _, err := execCmd("install", "-y", "--location", "global")
	if err != nil {
		t.Fatalf("install -y: %v", err)
	}
	if !strings.Contains(out, "Claude Code:") {
		t.Fatalf("expected -y to resolve auto (fallback to Claude in a fresh fake home), got:\n%s", out)
	}

	claudeConfig := filepath.Join(home, ".claude.json")
	if _, statErr := os.Stat(claudeConfig); statErr != nil {
		t.Fatalf("expected %s to be written: %v", claudeConfig, statErr)
	}
}

// TestInstall_InteractiveAllowed_CallsRunAgentPicker asserts that, with no
// -y and no --target, a forced-allowed TTY takes the picker branch and
// uses ITS resolved targets (not auto's) — proving the switch's wiring
// order: yes (false here) -> --target (unset) -> interactiveAllowed (true)
// -> runAgentPicker.
func TestInstall_InteractiveAllowed_CallsRunAgentPicker(t *testing.T) {
	home := fakeHome(t)
	var called bool
	withStubbedPicker(t, func(cmd *cobra.Command, loc agents.Location) ([]agents.AgentTarget, error) {
		called = true
		return agents.ResolveTargetFlag("claude", loc)
	})

	out, _, err := execCmd("install", "--location", "global")
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if !called {
		t.Fatal("expected runAgentPicker to be called when interactiveAllowed is true and neither -y nor --target is set")
	}
	if !strings.Contains(out, "Claude Code:") {
		t.Fatalf("expected the picker's resolved targets to be used, got:\n%s", out)
	}

	claudeConfig := filepath.Join(home, ".claude.json")
	if _, statErr := os.Stat(claudeConfig); statErr != nil {
		t.Fatalf("expected %s to be written: %v", claudeConfig, statErr)
	}
}

// TestUninstall_Yes_ShortCircuitsBeforeInteractiveBranch mirrors install's
// Pitfall-6 regression for uninstall: -y must resolve straight to the
// non-interactive default (every registered target) even with
// interactiveAllowed forced true, and must never invoke runAgentPicker.
func TestUninstall_Yes_ShortCircuitsBeforeInteractiveBranch(t *testing.T) {
	fakeHome(t)

	if _, _, err := execCmd("install", "--target", "claude,cursor", "--location", "global"); err != nil {
		t.Fatalf("install --target claude,cursor: %v", err)
	}

	withStubbedPicker(t, func(*cobra.Command, agents.Location) ([]agents.AgentTarget, error) {
		t.Fatal("runAgentPicker must never be called when -y is set")
		return nil, nil
	})

	out, _, err := execCmd("uninstall", "-y", "--location", "global")
	if err != nil {
		t.Fatalf("uninstall -y: %v", err)
	}
	if !strings.Contains(out, "Claude Code: removed") || !strings.Contains(out, "Cursor: removed") {
		t.Fatalf("expected -y to resolve to the full roster (all), got:\n%s", out)
	}
}

// TestUninstall_InteractiveAllowed_CallsRunAgentPicker mirrors install's
// wiring-order test for uninstall's switch.
func TestUninstall_InteractiveAllowed_CallsRunAgentPicker(t *testing.T) {
	fakeHome(t)

	if _, _, err := execCmd("install", "--target", "claude", "--location", "global"); err != nil {
		t.Fatalf("install --target claude: %v", err)
	}

	var called bool
	withStubbedPicker(t, func(cmd *cobra.Command, loc agents.Location) ([]agents.AgentTarget, error) {
		called = true
		return agents.ResolveTargetFlag("claude", loc)
	})

	out, _, err := execCmd("uninstall", "--location", "global")
	if err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	if !called {
		t.Fatal("expected runAgentPicker to be called when interactiveAllowed is true and neither -y nor --target is set")
	}
	if !strings.Contains(out, "Claude Code: removed") {
		t.Fatalf("expected the picker's resolved targets to be used, got:\n%s", out)
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

// TestInstallStatus_KeptForeignIsNotAChange (D-14): a foreign skill
// directory codegraph left untouched (agents.ActionKeptForeign) must not
// flip install's per-agent headline to "configured" — it is not a change
// codegraph made. A genuine change (agents.ActionCreated) alongside a
// kept-foreign entry still reports "configured".
func TestInstallStatus_KeptForeignIsNotAChange(t *testing.T) {
	cases := []struct {
		name   string
		result agents.WriteResult
		want   string
	}{
		{
			name: "unchanged plus kept foreign",
			result: agents.WriteResult{Files: []agents.FileResult{
				{Path: "a", Action: agents.ActionUnchanged},
				{Path: "b", Action: agents.ActionKeptForeign},
			}},
			want: "unchanged",
		},
		{
			name: "created plus kept foreign",
			result: agents.WriteResult{Files: []agents.FileResult{
				{Path: "a", Action: agents.ActionCreated},
				{Path: "b", Action: agents.ActionKeptForeign},
			}},
			want: "configured",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := installStatus(tc.result); got != tc.want {
				t.Fatalf("installStatus(%+v) = %q, want %q", tc.result, got, tc.want)
			}
		})
	}
}

// preToolNudgeNote is D-09's (07-08-widened) stderr note, printed once when
// --pretool-nudge is given (either value) and neither Claude Code nor Codex
// CLI is among the resolved targets.
const preToolNudgeNote = "note: --pretool-nudge only configures Claude Code and Codex CLI, neither of which is among the selected agents; nothing was changed for them"

// localPreToolGuard is where a local opt-in writes the rendered guard, and
// localPreToolCommand the command every local PreToolUse handler registers.
const (
	localPreToolGuard   = ".claude/hooks/pretooluse-nudge.sh"
	localPreToolCommand = "${CLAUDE_PROJECT_DIR}/.claude/hooks/pretooluse-nudge.sh"
)

// ownPreToolHandlerCount counts the PreToolUse handlers in settingsPath
// whose command is exactly command; an absent file or key counts 0.
func ownPreToolHandlerCount(t *testing.T, settingsPath, command string) int {
	t.Helper()
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		return 0
	}
	hooks, _ := readJSONMap(t, settingsPath)["hooks"].(map[string]any)
	blocks, _ := hooks["PreToolUse"].([]any)
	n := 0
	for _, b := range blocks {
		block, _ := b.(map[string]any)
		handlers, _ := block["hooks"].([]any)
		for _, h := range handlers {
			if handler, _ := h.(map[string]any); handler != nil && handler["command"] == command {
				n++
			}
		}
	}
	return n
}

// hasPreToolUseKey reports whether settingsPath carries hooks.PreToolUse.
func hasPreToolUseKey(t *testing.T, settingsPath string) bool {
	t.Helper()
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		return false
	}
	hooks, _ := readJSONMap(t, settingsPath)["hooks"].(map[string]any)
	_, ok := hooks["PreToolUse"]
	return ok
}

// TestInstall_PreToolNudge_OptInRegisters: --pretool-nudge writes the guard
// and a PreToolUse registration (D-09).
func TestInstall_PreToolNudge_OptInRegisters(t *testing.T) {
	fakeHome(t)

	if _, _, err := execCmd("install", "--target", "claude", "--location", "local", "--pretool-nudge"); err != nil {
		t.Fatalf("install --pretool-nudge: %v", err)
	}
	if _, err := os.Stat(localPreToolGuard); err != nil {
		t.Fatalf("guard %s not written: %v", localPreToolGuard, err)
	}
	if !hasPreToolUseKey(t, filepath.Join(".claude", "settings.json")) {
		t.Fatalf("settings.json has no hooks.PreToolUse after an opt-in install")
	}
}

// TestInstall_PreToolNudge_StickyAcrossPlainInstall: a plain install (flag
// not given) keeps and refreshes a recorded opt-in (D-10 Keep).
func TestInstall_PreToolNudge_StickyAcrossPlainInstall(t *testing.T) {
	fakeHome(t)
	settings := filepath.Join(".claude", "settings.json")

	if _, _, err := execCmd("install", "--target", "claude", "--location", "local", "--pretool-nudge"); err != nil {
		t.Fatalf("install --pretool-nudge: %v", err)
	}
	out, _, err := execCmd("install", "--target", "claude", "--location", "local")
	if err != nil {
		t.Fatalf("plain install: %v", err)
	}
	if _, err := os.Stat(localPreToolGuard); err != nil {
		t.Fatalf("a plain install removed the opted-in guard: %v", err)
	}
	if n := ownPreToolHandlerCount(t, settings, localPreToolCommand); n != 6 {
		t.Fatalf("own PreToolUse handlers after a plain install = %d, want 6", n)
	}
	if !strings.Contains(out, "unchanged: "+localPreToolGuard) {
		t.Fatalf("plain install did not report the guard unchanged (Keep refresh); stdout:\n%s", out)
	}
}

// TestInstall_PreToolNudge_ExplicitFalseRemoves: --pretool-nudge=false is
// the opt-out (D-10 Off), and a later plain install does not bring it back.
func TestInstall_PreToolNudge_ExplicitFalseRemoves(t *testing.T) {
	fakeHome(t)
	settings := filepath.Join(".claude", "settings.json")

	if _, _, err := execCmd("install", "--target", "claude", "--location", "local", "--pretool-nudge"); err != nil {
		t.Fatalf("install --pretool-nudge: %v", err)
	}
	if n := ownPreToolHandlerCount(t, settings, localPreToolCommand); n == 0 {
		t.Fatalf("precondition: the opt-in registered no own PreToolUse handler")
	}
	if _, _, err := execCmd("install", "--target", "claude", "--location", "local", "--pretool-nudge=false"); err != nil {
		t.Fatalf("install --pretool-nudge=false: %v", err)
	}
	assertOff := func(when string) {
		t.Helper()
		if _, err := os.Stat(localPreToolGuard); !os.IsNotExist(err) {
			t.Fatalf("%s: guard %s still present (stat err %v)", when, localPreToolGuard, err)
		}
		if n := ownPreToolHandlerCount(t, settings, localPreToolCommand); n != 0 {
			t.Fatalf("%s: %d own PreToolUse handlers remain, want 0", when, n)
		}
	}
	assertOff("after --pretool-nudge=false")

	if _, _, err := execCmd("install", "--target", "claude", "--location", "local"); err != nil {
		t.Fatalf("plain install: %v", err)
	}
	assertOff("after a later plain install")
}

// localCodexPreToolGuard is where a local Codex opt-in writes the rendered
// guard (mirrors localPreToolGuard for Claude).
const localCodexPreToolGuard = ".codex/hooks/codegraph-pretooluse.sh"

// TestInstall_PreToolNudge_NoteWhenNeitherClaudeNorCodexSelected (renamed
// from …WhenClaudeNotSelected, 07-08/D-09 widened): given either value
// while neither Claude nor Codex is a resolved target, install says so once
// on stderr and still succeeds.
func TestInstall_PreToolNudge_NoteWhenNeitherClaudeNorCodexSelected(t *testing.T) {
	for _, tc := range []struct{ name, flag string }{
		{"given_true", "--pretool-nudge"},
		{"given_false", "--pretool-nudge=false"},
	} {
		flag := tc.flag
		t.Run(tc.name, func(t *testing.T) {
			fakeHome(t)

			stdout, stderr, err := execCmd("install", "--target", "cursor", "--location", "local", flag)
			if err != nil {
				t.Fatalf("install --target cursor %s: %v", flag, err)
			}
			if got := strings.Count(stderr, preToolNudgeNote+"\n"); got != 1 {
				t.Fatalf("stderr carries the note %d times, want 1; stderr:\n%s", got, stderr)
			}
			if strings.Contains(stdout, "--pretool-nudge") {
				t.Fatalf("the note leaked onto stdout:\n%s", stdout)
			}
			if _, err := os.Stat(localPreToolGuard); !os.IsNotExist(err) {
				t.Fatalf("a Cursor-only install wrote %s (stat err %v)", localPreToolGuard, err)
			}
			if _, err := os.Stat(localCodexPreToolGuard); !os.IsNotExist(err) {
				t.Fatalf("a Cursor-only install wrote %s (stat err %v)", localCodexPreToolGuard, err)
			}
		})
	}
}

// TestInstall_PreToolNudge_NoNoteWhenCodexSelected: given true while Codex
// is a resolved target, install prints no note and writes Codex's guard and
// hooks.json (D-09 widened — the note's absence when Claude is selected is
// already covered by TestInstall_PreToolNudge_NoNoteWhenClaudeSelected).
func TestInstall_PreToolNudge_NoNoteWhenCodexSelected(t *testing.T) {
	fakeHome(t)

	_, stderr, err := execCmd("install", "--target", "codex,cursor", "--location", "local", "--pretool-nudge")
	if err != nil {
		t.Fatalf("install --target codex,cursor --pretool-nudge: %v", err)
	}
	if strings.Contains(stderr, "note: --pretool-nudge") {
		t.Fatalf("note printed although Codex was selected; stderr:\n%s", stderr)
	}
	if _, err := os.Stat(localCodexPreToolGuard); err != nil {
		t.Fatalf("positive control: Codex was selected but the guard was not written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(".codex", "hooks.json")); err != nil {
		t.Fatalf("positive control: Codex was selected but hooks.json was not written: %v", err)
	}
}

// TestInstall_PreToolNudge_CodexOptInRegisters mirrors
// TestInstall_PreToolNudge_OptInRegisters/StickyAcrossPlainInstall/
// ExplicitFalseRemoves for Codex: a CLI-level opt-in registers the guard
// and hooks.json group, a plain install keeps it, and
// --pretool-nudge=false removes it.
func TestInstall_PreToolNudge_CodexOptInRegisters(t *testing.T) {
	fakeHome(t)
	hooksPath := filepath.Join(".codex", "hooks.json")

	if _, _, err := execCmd("install", "--target", "codex", "--location", "local", "--pretool-nudge"); err != nil {
		t.Fatalf("install --target codex --pretool-nudge: %v", err)
	}
	if _, err := os.Stat(localCodexPreToolGuard); err != nil {
		t.Fatalf("guard %s not written: %v", localCodexPreToolGuard, err)
	}
	if _, err := os.Stat(hooksPath); err != nil {
		t.Fatalf("hooks.json %s not written: %v", hooksPath, err)
	}

	if _, _, err := execCmd("install", "--target", "codex", "--location", "local"); err != nil {
		t.Fatalf("plain install --target codex: %v", err)
	}
	if _, err := os.Stat(localCodexPreToolGuard); err != nil {
		t.Fatalf("a plain install removed the opted-in Codex guard: %v", err)
	}

	if _, _, err := execCmd("install", "--target", "codex", "--location", "local", "--pretool-nudge=false"); err != nil {
		t.Fatalf("install --target codex --pretool-nudge=false: %v", err)
	}
	if _, err := os.Stat(localCodexPreToolGuard); !os.IsNotExist(err) {
		t.Fatalf("guard %s still present after --pretool-nudge=false (stat err %v)", localCodexPreToolGuard, err)
	}
}

// TestInstallHelpNeverAdvisesTrustBypass (D-19, T-07-24): install --help,
// uninstall --help, and the generated CLI reference never advise bypassing
// Codex's hook trust review, and install --help names both Codex and
// /hooks. The forbidden token is built by concatenation so this test's own
// source never matches it.
func TestInstallHelpNeverAdvisesTrustBypass(t *testing.T) {
	forbidden := "--dangerously-" + "bypass-hook-trust"

	installHelp, _, err := execCmd("install", "--help")
	if err != nil {
		t.Fatalf("install --help: %v", err)
	}
	if strings.Contains(installHelp, forbidden) {
		t.Fatalf("install --help advises trust bypass:\n%s", installHelp)
	}
	if !strings.Contains(installHelp, "Codex") {
		t.Fatalf("install --help does not mention Codex:\n%s", installHelp)
	}
	if !strings.Contains(installHelp, "/hooks") {
		t.Fatalf("install --help does not mention /hooks:\n%s", installHelp)
	}

	uninstallHelp, _, err := execCmd("uninstall", "--help")
	if err != nil {
		t.Fatalf("uninstall --help: %v", err)
	}
	if strings.Contains(uninstallHelp, forbidden) {
		t.Fatalf("uninstall --help advises trust bypass:\n%s", uninstallHelp)
	}

	refBytes, err := os.ReadFile(cliReferenceDocPath)
	if err != nil {
		t.Fatalf("read %s: %v", cliReferenceDocPath, err)
	}
	if strings.Contains(string(refBytes), forbidden) {
		t.Fatalf("%s advises trust bypass", cliReferenceDocPath)
	}
}

// TestInstall_PreToolNudge_NoNoteWhenClaudeSelected: no note when Claude is
// selected, and none when the flag was not given at all.
func TestInstall_PreToolNudge_NoNoteWhenClaudeSelected(t *testing.T) {
	fakeHome(t)

	_, stderr, err := execCmd("install", "--target", "claude,cursor", "--location", "local", "--pretool-nudge")
	if err != nil {
		t.Fatalf("install --target claude,cursor --pretool-nudge: %v", err)
	}
	if strings.Contains(stderr, "note: --pretool-nudge") {
		t.Fatalf("note printed although Claude was selected; stderr:\n%s", stderr)
	}
	if _, err := os.Stat(localPreToolGuard); err != nil {
		t.Fatalf("positive control: Claude was selected but the guard was not written: %v", err)
	}

	_, stderr, err = execCmd("install", "--target", "cursor", "--location", "local")
	if err != nil {
		t.Fatalf("plain install --target cursor: %v", err)
	}
	if strings.Contains(stderr, "note: --pretool-nudge") {
		t.Fatalf("note printed although --pretool-nudge was not given; stderr:\n%s", stderr)
	}
}

// TestInstall_YesWithExplicitTarget_HonoursTarget is the D-13 regression:
// an explicit --target must win over -y/--yes, not be discarded in favour
// of the non-interactive "auto" default. Before the fix, install's switch
// checked `case yes:` before `case cmd.Flags().Changed("target"):`, so
// `install --target codex --yes` silently configured whatever "auto"
// resolved to (Claude, in a fresh fake home) instead of Codex.
// runAgentPicker is stubbed to fail the test if ever invoked — -y must
// short-circuit before the interactive branch too (Pitfall 6), and an
// explicit --target must not reopen that question.
func TestInstall_YesWithExplicitTarget_HonoursTarget(t *testing.T) {
	home := fakeHome(t)
	withStubbedPicker(t, func(*cobra.Command, agents.Location) ([]agents.AgentTarget, error) {
		t.Fatal("runAgentPicker must never be called when --target is explicit")
		return nil, nil
	})

	out, _, err := execCmd("install", "--target", "codex", "-y", "--location", "global")
	if err != nil {
		t.Fatalf("install --target codex -y: %v", err)
	}
	if !strings.Contains(out, "Codex CLI:") {
		t.Fatalf("expected explicit --target codex to configure Codex, got:\n%s", out)
	}
	if strings.Contains(out, "Claude Code:") {
		t.Fatalf("expected --yes NOT to widen an explicit --target codex to Claude, got:\n%s", out)
	}

	if _, statErr := os.Stat(filepath.Join(home, ".codex", "config.toml")); statErr != nil {
		t.Fatalf("expected %s/.codex/config.toml to be written: %v", home, statErr)
	}
	if _, statErr := os.Stat(filepath.Join(home, ".claude.json")); !os.IsNotExist(statErr) {
		t.Fatalf("expected %s/.claude.json NOT to be written (--target codex must not touch Claude), stat err: %v", home, statErr)
	}
}

// TestUninstall_YesWithExplicitTarget_HonoursTarget mirrors the install
// regression for uninstall: an explicit --target must win over
// -y/--yes, which otherwise resolves to "all" and would remove every
// installed agent's configuration rather than just the one named.
func TestUninstall_YesWithExplicitTarget_HonoursTarget(t *testing.T) {
	home := fakeHome(t)

	if _, _, err := execCmd("install", "--target", "claude,codex", "--location", "global"); err != nil {
		t.Fatalf("install --target claude,codex: %v", err)
	}

	withStubbedPicker(t, func(*cobra.Command, agents.Location) ([]agents.AgentTarget, error) {
		t.Fatal("runAgentPicker must never be called when --target is explicit")
		return nil, nil
	})

	out, _, err := execCmd("uninstall", "--target", "codex", "--yes", "--location", "global")
	if err != nil {
		t.Fatalf("uninstall --target codex --yes: %v", err)
	}
	if !strings.Contains(out, "Codex CLI:") {
		t.Fatalf("expected explicit --target codex to be reported, got:\n%s", out)
	}
	if strings.Contains(out, "Claude Code:") {
		t.Fatalf("expected --yes NOT to widen an explicit --target codex to Claude, got:\n%s", out)
	}

	claudeConfig := readJSONMap(t, filepath.Join(home, ".claude.json"))
	mcpServers, _ := claudeConfig["mcpServers"].(map[string]any)
	if _, ok := mcpServers["codegraph"]; !ok {
		t.Fatalf("expected Claude's mcpServers.codegraph entry to survive an explicit --target codex uninstall, got: %v", mcpServers)
	}

	// Codex's config.toml held only the codegraph table (nothing else was
	// ever written to it), so stripping that table empties the file
	// entirely — the D-07/D-08 keep-clean precedent removes it rather than
	// leaving an empty file behind (D-09: this now applies at every scope,
	// not just local).
	codexConfigPath := filepath.Join(home, ".codex", "config.toml")
	if _, statErr := os.Stat(codexConfigPath); !os.IsNotExist(statErr) {
		t.Fatalf("expected %s to be removed entirely (emptied by stripping the sole codegraph table), stat err: %v", codexConfigPath, statErr)
	}
}
