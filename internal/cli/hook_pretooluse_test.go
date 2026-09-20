package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/seanb4t/codegraph-go/internal/nudge"
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

// assertHookContract is the D-16 subcommand-level contract every run must
// hold: Execute() returns nil (so main.go's exit-1 path is unreachable),
// stdout is empty or exactly the pinned object, and no decision key ever
// appears.
func assertHookContract(t *testing.T, stdout string, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("hook pretooluse returned %v, want nil (D-01a: never an error to cobra)", err)
	}
	if stdout != "" && stdout != pinnedPreToolUseFire {
		t.Errorf("stdout = %q, want empty or exactly the pinned fire line", stdout)
	}
	for _, forbidden := range []string{"permissionDecision", `"decision"`, `"continue"`} {
		if strings.Contains(stdout, forbidden) {
			t.Errorf("stdout carries %s; the nudge never decides (NUDGE-03): %q", forbidden, stdout)
		}
	}
}

// hookEvent is a qualifying Grep PreToolUse event for session (omitted
// when empty) and agent (omitted when empty).
func hookEvent(session, agent string) string {
	s := `{"hook_event_name":"PreToolUse","tool_name":"Grep","tool_input":{"pattern":"Alpha"}`
	if session != "" {
		s += `,"session_id":"` + session + `"`
	}
	if agent != "" {
		s += `,"agent_id":"` + agent + `"`
	}
	return s + "}"
}

// sentinelDirUnder is the path DefaultDir resolves to for a TMPDIR of base.
func sentinelDirUnder(base string) string {
	return filepath.Join(base, "codegraph-nudge-"+strconv.Itoa(os.Getuid()))
}

func TestHookPreToolUse_ForcedErrorContract(t *testing.T) {
	padTo := func(s string, n int) string { return s + strings.Repeat(" ", n-len(s)) }

	cases := []struct {
		name       string
		input      string
		envSession string
		setup      func(t *testing.T, tmp string) // may repoint TMPDIR
		want       string
		after      func(t *testing.T, tmp string)
	}{
		{name: "no_stdin", input: "", want: ""},
		{name: "malformed_json", input: `{"tool_name":`, want: ""},
		{name: "oversized_input", input: padTo(hookEvent("s-over", ""), maxHookStdinBytes+1), want: ""},
		{name: "max_size_input_fires", input: padTo(hookEvent("s-max", ""), maxHookStdinBytes), want: pinnedPreToolUseFire},
		{name: "no_session_ids", input: hookEvent("", ""), want: ""},
		{name: "env_session_only", input: hookEvent("", ""), envSession: "env-only", want: pinnedPreToolUseFire},
		{name: "stdin_session_only", input: hookEvent("stdin-only", ""), want: pinnedPreToolUseFire},
		{
			name:  "non_pretooluse_event",
			input: `{"session_id":"s-post","hook_event_name":"PostToolUse","tool_name":"Grep","tool_input":{"pattern":"Alpha"}}`,
			want:  "",
		},
		{
			name:  "unqualified_tool_touches_no_sentinel",
			input: `{"session_id":"s-read","hook_event_name":"PreToolUse","tool_name":"Read","tool_input":{"file_path":"/repo/README.md"}}`,
			want:  "",
			after: func(t *testing.T, tmp string) {
				if _, err := os.Lstat(sentinelDirUnder(tmp)); !os.IsNotExist(err) {
					t.Errorf("sentinel dir after a non-qualifying call: lstat err = %v, want not-exist (classification precedes sentinel I/O)", err)
				}
			},
		},
		{
			name:  "unwritable_sentinel_base",
			input: hookEvent("s-unwritable", ""),
			setup: func(t *testing.T, tmp string) {
				file := filepath.Join(tmp, "regular-file")
				if err := os.WriteFile(file, nil, 0o600); err != nil {
					t.Fatal(err)
				}
				t.Setenv("TMPDIR", file)
			},
			want: "",
		},
		{
			name:  "symlinked_sentinel_dir",
			input: hookEvent("s-symlink", ""),
			setup: func(t *testing.T, tmp string) {
				realDir := filepath.Join(t.TempDir(), "real")
				if err := os.Mkdir(realDir, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(realDir, sentinelDirUnder(tmp)); err != nil {
					t.Fatal(err)
				}
			},
			want: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			isolateHookEnv(t)
			tmp := os.Getenv("TMPDIR")
			if tc.envSession != "" {
				t.Setenv("CLAUDE_CODE_SESSION_ID", tc.envSession)
			}
			if tc.setup != nil {
				tc.setup(t, tmp)
			}
			stdout, _, err := execCmdWithInput(tc.input, "hook", "pretooluse")
			assertHookContract(t, stdout, err)
			if stdout != tc.want {
				t.Errorf("stdout = %q, want %q", stdout, tc.want)
			}
			if tc.after != nil {
				tc.after(t, tmp)
			}
		})
	}
}

func TestHookPreToolUse_EnvSessionTakesPrecedence(t *testing.T) {
	isolateHookEnv(t)
	run := func(want, why string) {
		t.Helper()
		stdout, _, err := execCmdWithInput(hookEvent("B", ""), "hook", "pretooluse")
		assertHookContract(t, stdout, err)
		if stdout != want {
			t.Errorf("%s: stdout = %q, want %q", why, stdout, want)
		}
	}

	t.Setenv("CLAUDE_CODE_SESSION_ID", "A")
	run(pinnedPreToolUseFire, "env A + stdin B, first call")
	run("", "env A + stdin B again (key A is cooling down)")
	t.Setenv("CLAUDE_CODE_SESSION_ID", "")
	run(pinnedPreToolUseFire, "stdin B alone (key B was never used, D-07)")
}

func TestHookPreToolUse_CooldownPerAgent(t *testing.T) {
	isolateHookEnv(t)
	t0 := time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)
	orig := hookNow
	t.Cleanup(func() { hookNow = orig })

	steps := []struct {
		at    time.Duration
		agent string
		want  string
		why   string
	}{
		{0, "", pinnedPreToolUseFire, "main, first call"},
		{0, "", "", "main again (cooldown)"},
		{0, "sub-1", pinnedPreToolUseFire, "subagent sub-1, first call (own key, D-06)"},
		{0, "sub-1", "", "sub-1 again (cooldown)"},
		{59 * time.Second, "", "", "main at T+59s (inside the window)"},
		{60 * time.Second, "", pinnedPreToolUseFire, "main at T+60s (window elapsed, D-05)"},
	}
	for _, s := range steps {
		now := t0.Add(s.at)
		hookNow = func() time.Time { return now }
		stdout, _, err := execCmdWithInput(hookEvent("s-agent", s.agent), "hook", "pretooluse")
		assertHookContract(t, stdout, err)
		if stdout != s.want {
			t.Errorf("%s: stdout = %q, want %q", s.why, stdout, s.want)
		}
	}
}

func TestHookPreToolUse_ParallelRuns(t *testing.T) {
	isolateHookEnv(t)
	getenv := func(string) string { return "" }

	const n = 16
	outs := make([]string, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var buf bytes.Buffer
			runHookPreToolUse(strings.NewReader(hookEvent("s-parallel", "")), &buf, getenv)
			outs[i] = buf.String()
		}()
	}
	wg.Wait()

	fired := 0
	for i, out := range outs {
		assertHookContract(t, out, nil)
		if out == pinnedPreToolUseFire {
			fired++
		} else if out != "" {
			t.Errorf("run %d: stdout = %q, want empty or pinned", i, out)
		}
	}
	if fired < 1 {
		t.Errorf("16 parallel runs fired %d times, want at least 1", fired)
	}
}

func TestHookPreToolUse_PanicIsRecovered(t *testing.T) {
	isolateHookEnv(t)
	orig := hookQualifies
	t.Cleanup(func() { hookQualifies = orig })
	hookQualifies = func(nudge.Tool, string) bool { panic("forced panic in the classifier") }

	stdout, _, err := execCmdWithInput(hookEvent("s-panic", ""), "hook", "pretooluse")
	assertHookContract(t, stdout, err)
	if stdout != "" {
		t.Errorf("stdout = %q, want empty (the panic is recovered before anything is written)", stdout)
	}
}

func TestHookPreToolUse_NoAncestorPreRun(t *testing.T) {
	root := newRootCmd()
	pre, _, err := root.Find([]string{"hook", "pretooluse"})
	if err != nil || pre == nil || pre.Name() != "pretooluse" {
		t.Fatalf("Find(hook pretooluse) = %v, %v", pre, err)
	}
	visited := 0
	for c := pre; c != nil; c = c.Parent() {
		visited++
		if c.PersistentPreRun != nil || c.PersistentPreRunE != nil || c.PreRun != nil || c.PreRunE != nil {
			t.Errorf("%q sets a PreRun hook; one that errors would reach main.go's exit-1 path before the recover (RESEARCH Pitfall 1)", c.CommandPath())
		}
	}
	if visited < 3 {
		t.Errorf("walked %d commands, want at least 3 (pretooluse, hook, root)", visited)
	}
}
