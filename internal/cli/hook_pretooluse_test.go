package cli

import (
	"strings"
	"testing"
)

// pinnedPreToolUseFire is the exact stdout `codegraph hook pretooluse`
// prints when it fires (NUDGE-03, D-14): one JSON object carrying only
// hookSpecificOutput.additionalContext, then a newline. Hand-typed here — an
// independent oracle, deliberately not built from nudge.Text — so a drift
// in either the envelope or the sentence turns the tests below RED.
const pinnedPreToolUseFire = `{"hookSpecificOutput":{"hookEventName":"PreToolUse","additionalContext":"This repo has a codegraph index: codegraph_explore (CLI: ` + "`codegraph explore`" + `) returns the matching symbols' source and call paths for where-is-X and how-does-Y questions."}}` + "\n"

// isolateHookEnv clears the session id the executor itself inherits from
// Claude Code and points TMPDIR at a fresh directory, so no test reads a
// real session's id or (once 06-03 adds it) another run's cooldown
// sentinel.
func isolateHookEnv(t *testing.T) {
	t.Helper()
	t.Setenv("CLAUDE_CODE_SESSION_ID", "")
	t.Setenv("TMPDIR", t.TempDir())
}

func TestHookPreToolUse_GrepFiresPinnedContext(t *testing.T) {
	isolateHookEnv(t)

	cases := []struct {
		name  string
		input string
	}{
		{"grep", `{"session_id":"s-1","hook_event_name":"PreToolUse","tool_name":"Grep","tool_input":{"pattern":"Alpha"}}`},
		{"glob", `{"session_id":"s-2","hook_event_name":"PreToolUse","tool_name":"Glob","tool_input":{"pattern":"**/*.go"}}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, err := execCmdWithInput(tc.input, "hook", "pretooluse")
			if err != nil {
				t.Fatalf("hook pretooluse returned %v, want nil (D-01a: never an error to cobra)", err)
			}
			if stdout != pinnedPreToolUseFire {
				t.Fatalf("stdout = %q, want the pinned fire line %q", stdout, pinnedPreToolUseFire)
			}
			if stderr != "" {
				t.Fatalf("stderr = %q, want empty", stderr)
			}
		})
	}
}

func TestHookPreToolUse_NoSessionIsSilent(t *testing.T) {
	isolateHookEnv(t)
	const noSession = `{"hook_event_name":"PreToolUse","tool_name":"Grep","tool_input":{"pattern":"Alpha"}}`

	stdout, stderr, err := execCmdWithInput(noSession, "hook", "pretooluse")
	if err != nil {
		t.Fatalf("no session: err = %v, want nil", err)
	}
	if stdout != "" || stderr != "" {
		t.Fatalf("no session: stdout = %q, stderr = %q, want both empty (D-07: never fire unkeyed)", stdout, stderr)
	}

	// Positive control: the same event with the env session id fires, so
	// the silence above is the missing key and not a dead command (D-07).
	t.Setenv("CLAUDE_CODE_SESSION_ID", "env-1")
	stdout, _, err = execCmdWithInput(noSession, "hook", "pretooluse")
	if err != nil {
		t.Fatalf("env session: err = %v, want nil", err)
	}
	if stdout != pinnedPreToolUseFire {
		t.Fatalf("env session: stdout = %q, want the pinned fire line", stdout)
	}
}

func TestHookCmd_HiddenTwoLevel(t *testing.T) {
	root := newRootCmd()

	hook, _, err := root.Find([]string{"hook"})
	if err != nil || hook == nil || hook.Name() != "hook" {
		t.Fatalf("Find(hook) = %v, %v; want the hook command", hook, err)
	}
	pre, _, err := root.Find([]string{"hook", "pretooluse"})
	if err != nil || pre == nil || pre.Name() != "pretooluse" {
		t.Fatalf("Find(hook pretooluse) = %v, %v; want the pretooluse command", pre, err)
	}
	if !hook.Hidden {
		t.Errorf("hook.Hidden = false, want true (D-01a)")
	}
	if !pre.Hidden {
		t.Errorf("hook pretooluse.Hidden = false, want true (D-01a)")
	}
	for _, name := range []string{"hook", "pretooluse"} {
		if _, ok := commandGroups[name]; ok {
			t.Errorf("commandGroups has %q; hidden commands stay groupless", name)
		}
	}

	stdout, _, err := execCmd("--help")
	if err != nil {
		t.Fatalf("codegraph --help: %v", err)
	}
	if !strings.Contains(stdout, "install") {
		t.Fatalf("codegraph --help output does not list install; the scan below would be vacuous:\n%s", stdout)
	}
	for _, line := range strings.Split(stdout, "\n") {
		if f := strings.Fields(line); len(f) > 0 && f[0] == "hook" {
			t.Errorf("codegraph --help lists hook: %q", line)
		}
	}
}
