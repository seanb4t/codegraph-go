package cli

import (
	"os"
	"path/filepath"
	"testing"
)

// codexBashEvent is a qualifying Codex PreToolUse Bash event for session
// and cwd — both required: Codex's cwd re-check happens in the Go core
// itself (D-22), not just the guard.
func codexBashEvent(session, cwd string) string {
	return `{"session_id":"` + session + `","hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{"command":"rg -n Alpha ."},"cwd":"` + cwd + `"}`
}

// TestHookPreToolUseCodex_BashFiresPinnedContext (L6, D-21): a qualifying
// Bash search event in an indexed cwd, run through the Codex envelope,
// prints exactly the pinned fire line and exits 0.
func TestHookPreToolUseCodex_BashFiresPinnedContext(t *testing.T) {
	isolateHookEnv(t)
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".codegraph"), 0o755); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := execCmdWithInput(codexBashEvent("s-codex-1", dir), "hook", "pretooluse", "--harness", "codex")
	if err != nil {
		t.Fatalf("hook pretooluse --harness codex returned %v, want nil (D-01a: never an error to cobra)", err)
	}
	if stdout != pinnedPreToolUseFire {
		t.Fatalf("stdout = %q, want the pinned fire line %q", stdout, pinnedPreToolUseFire)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

// TestHookPreToolUseCodex_NotIndexedCwdIsSilent (L6, D-22): the same event
// with a cwd lacking .codegraph produces no output — the Go core re-checks
// the indexed state, not just the guard.
func TestHookPreToolUseCodex_NotIndexedCwdIsSilent(t *testing.T) {
	isolateHookEnv(t)
	dir := t.TempDir() // no .codegraph directory

	stdout, stderr, err := execCmdWithInput(codexBashEvent("s-codex-2", dir), "hook", "pretooluse", "--harness", "codex")
	if err != nil {
		t.Fatalf("hook pretooluse --harness codex returned %v, want nil", err)
	}
	if stdout != "" || stderr != "" {
		t.Fatalf("stdout = %q, stderr = %q, want both empty (un-indexed cwd)", stdout, stderr)
	}
}
