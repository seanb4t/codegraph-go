package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/seanb4t/codegraph-go/internal/nudge"
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

// codexEventFull builds a Codex PreToolUse event JSON. cmd is marshaled
// verbatim as tool_input.command (a string, a slice, nil, a map — any
// JSON-marshalable value), letting a single builder produce every shape
// D-21's forced-error contract needs. session, agent, and cwd, when
// empty, are OMITTED from the JSON entirely rather than emitted as empty
// strings — the D-21/D-22 "missing" cases (no session, no cwd) need a
// genuinely absent key, not a present-but-empty one.
func codexEventFull(t *testing.T, hookEventName, toolName, session, agent, cwd string, cmd any) string {
	t.Helper()
	m := map[string]any{
		"tool_name":       toolName,
		"tool_input":      map[string]any{"command": cmd},
		"hook_event_name": hookEventName,
	}
	if session != "" {
		m["session_id"] = session
	}
	if agent != "" {
		m["agent_id"] = agent
	}
	if cwd != "" {
		m["cwd"] = cwd
	}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal codex event: %v", err)
	}
	return string(data)
}

// codexEvent is codexEventFull fixed at hook_event_name "PreToolUse".
func codexEvent(t *testing.T, toolName, session, agent, cwd string, cmd any) string {
	return codexEventFull(t, "PreToolUse", toolName, session, agent, cwd, cmd)
}

// TestHookPreToolUseCodex_ForcedErrorContract (D-16/D-21/D-22): every
// decode-ambiguity, tool-mapping, command-shape, cwd, and sentinel-safety
// edge case resolves to silence except the six cases explicitly named
// "_fires".
func TestHookPreToolUseCodex_ForcedErrorContract(t *testing.T) {
	padTo := func(s string, n int) string { return s + strings.Repeat(" ", n-len(s)) }

	baseDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(baseDir, ".codegraph"), 0o755); err != nil {
		t.Fatal(err)
	}
	notIndexedDir := t.TempDir() // no .codegraph
	fileDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(fileDir, ".codegraph"), nil, 0o600); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name       string
		input      string
		envSession string
		setup      func(t *testing.T, tmp string)
		want       string
	}{
		{name: "no_stdin", input: "", want: ""},
		{name: "malformed_json", input: `{"tool_name":`, want: ""},
		{name: "oversized_input", input: padTo(codexEvent(t, "Bash", "s-over", "", baseDir, "rg -n Alpha ."), maxHookStdinBytes+1), want: ""},
		{name: "max_size_input_fires", input: padTo(codexEvent(t, "Bash", "s-max", "", baseDir, "rg -n Alpha ."), maxHookStdinBytes), want: pinnedPreToolUseFire},
		{name: "no_session_id", input: codexEvent(t, "Bash", "", "", baseDir, "rg -n Alpha ."), want: ""},
		{name: "claude_env_session_ignored", input: codexEvent(t, "Bash", "", "", baseDir, "rg -n Alpha ."), envSession: "env-should-be-ignored", want: ""},
		{name: "non_pretooluse_event", input: codexEventFull(t, "PostToolUse", "Bash", "s-post", "", baseDir, "rg -n Alpha ."), want: ""},
		{name: "unmapped_tool", input: codexEvent(t, "apply_patch", "s-apply", "", baseDir, "rg -n Alpha ."), want: ""},
		{name: "lowercase_bash", input: codexEvent(t, "bash", "s-lower", "", baseDir, "rg -n Alpha ."), want: ""},
		{name: "exec_command_tool_fires", input: codexEvent(t, "exec_command", "s-exec", "", baseDir, "rg -n Alpha ."), want: pinnedPreToolUseFire},
		{name: "shell_tool_fires", input: codexEvent(t, "shell", "s-shell", "", baseDir, "rg -n Alpha ."), want: pinnedPreToolUseFire},
		{name: "command_null", input: codexEvent(t, "Bash", "s-null", "", baseDir, nil), want: ""},
		{name: "command_empty_string", input: codexEvent(t, "Bash", "s-empty-str", "", baseDir, ""), want: ""},
		{name: "command_empty_array", input: codexEvent(t, "Bash", "s-empty-arr", "", baseDir, []string{}), want: ""},
		{name: "command_nonstring_array", input: codexEvent(t, "Bash", "s-nonstring", "", baseDir, []any{123, 456}), want: ""},
		{name: "command_object", input: codexEvent(t, "Bash", "s-object", "", baseDir, map[string]any{"foo": "bar"}), want: ""},
		{name: "command_single_element_argv_fires", input: codexEvent(t, "Bash", "s-single-argv", "", baseDir, []string{"rg -n Alpha ."}), want: pinnedPreToolUseFire},
		{name: "command_argv_last_element_fires", input: codexEvent(t, "Bash", "s-last-argv", "", baseDir, []string{"bash", "-lc", "rg -n Alpha ."}), want: pinnedPreToolUseFire},
		{name: "cwd_missing", input: codexEvent(t, "Bash", "s-cwd-missing", "", "", "rg -n Alpha ."), want: ""},
		{name: "cwd_relative", input: codexEvent(t, "Bash", "s-cwd-rel", "", "relative/path", "rg -n Alpha ."), want: ""},
		{name: "cwd_not_indexed", input: codexEvent(t, "Bash", "s-cwd-not-indexed", "", notIndexedDir, "rg -n Alpha ."), want: ""},
		{name: "cwd_codegraph_is_file", input: codexEvent(t, "Bash", "s-cwd-file", "", fileDir, "rg -n Alpha ."), want: ""},
		{
			name:  "unwritable_sentinel_base",
			input: codexEvent(t, "Bash", "s-unwritable", "", baseDir, "rg -n Alpha ."),
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
			input: codexEvent(t, "Bash", "s-symlink", "", baseDir, "rg -n Alpha ."),
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

	if len(cases) != 24 {
		t.Fatalf("test table has %d cases, want 24", len(cases))
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
			stdout, _, err := execCmdWithInput(tc.input, "hook", "pretooluse", "--harness", "codex")
			assertHookContract(t, stdout, err)
			if stdout != tc.want {
				t.Errorf("stdout = %q, want %q", stdout, tc.want)
			}
		})
	}
}

// TestHookPreToolUseCodex_CooldownPerAgent (D-05, D-06): the 60s cooldown
// applies per (session, agent) key — a subagent's own key fires
// independently of the main thread's, and the window elapsing lets the
// same key fire again.
func TestHookPreToolUseCodex_CooldownPerAgent(t *testing.T) {
	isolateHookEnv(t)
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".codegraph"), 0o755); err != nil {
		t.Fatal(err)
	}
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
		event := codexEvent(t, "Bash", "s-codex-agent", s.agent, dir, "rg -n Alpha .")
		stdout, _, err := execCmdWithInput(event, "hook", "pretooluse", "--harness", "codex")
		assertHookContract(t, stdout, err)
		if stdout != s.want {
			t.Errorf("%s: stdout = %q, want %q", s.why, stdout, s.want)
		}
	}
}

// codexCorpusRow mirrors internal/nudge's own corpusRow shape (that type
// is unexported to the nudge package and its _test.go file, so this is a
// separate decode target reading the SAME testdata files — D-15/D-21).
type codexCorpusRow struct {
	Tool  string `json:"tool"`
	Input string `json:"input"`
	Want  bool   `json:"want"`
	Note  string `json:"note"`
}

// loadCodexCorpus reads internal/nudge/testdata/<name>.json relative to
// this package directory — the SAME fixture file the Claude path's own
// corpus tests read, never a copied set of rows (D-00).
func loadCodexCorpus(t *testing.T, name string) []codexCorpusRow {
	t.Helper()
	path := filepath.Join("..", "nudge", "testdata", name+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var rows []codexCorpusRow
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return rows
}

// TestHookPreToolUseCodex_ShellCorpora drives every tool=="shell" row of
// both D-15 corpora through the Codex envelope, as a plain string and as
// an argv array (["bash", "-lc", <input>]), asserting the fire outcome
// matches the row's own `want` in both forms.
func TestHookPreToolUseCodex_ShellCorpora(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".codegraph"), 0o755); err != nil {
		t.Fatal(err)
	}

	for _, corpusName := range []string{"true-positives", "false-positives"} {
		t.Run(corpusName, func(t *testing.T) {
			rows := loadCodexCorpus(t, corpusName)
			driven, fired := 0, 0
			for i, row := range rows {
				if row.Tool != "shell" {
					continue
				}
				for _, form := range []string{"string", "argv"} {
					driven++
					isolateHookEnv(t)
					var cmd any = row.Input
					if form == "argv" {
						cmd = []string{"bash", "-lc", row.Input}
					}
					session := fmt.Sprintf("s-corpus-%s-%d-%s", corpusName, i, form)
					event := codexEvent(t, "Bash", session, "", dir, cmd)
					stdout, _, err := execCmdWithInput(event, "hook", "pretooluse", "--harness", "codex")
					assertHookContract(t, stdout, err)
					gotFire := stdout == pinnedPreToolUseFire
					if gotFire != row.Want {
						t.Errorf("row %d (%s form) input=%q: fired=%v, want=%v (%s)", i, form, row.Input, gotFire, row.Want, row.Note)
					}
					if gotFire {
						fired++
					}
				}
			}
			t.Logf("%s: drove %d shell rows (string+argv forms), %d fired", corpusName, driven, fired)
			if driven < 14 {
				t.Fatalf("drove only %d rows for %s, want at least 14", driven, corpusName)
			}
		})
	}
}

// TestHookPreToolUseCodex_PanicIsRecovered mirrors
// TestHookPreToolUse_PanicIsRecovered for the Codex envelope: a forced
// panic inside the shared classifier is recovered before anything is
// written.
func TestHookPreToolUseCodex_PanicIsRecovered(t *testing.T) {
	isolateHookEnv(t)
	orig := hookQualifies
	t.Cleanup(func() { hookQualifies = orig })
	hookQualifies = func(nudge.Tool, string) bool { panic("forced panic in the classifier") }

	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".codegraph"), 0o755); err != nil {
		t.Fatal(err)
	}
	event := codexEvent(t, "Bash", "s-codex-panic", "", dir, "rg -n Alpha .")
	stdout, _, err := execCmdWithInput(event, "hook", "pretooluse", "--harness", "codex")
	assertHookContract(t, stdout, err)
	if stdout != "" {
		t.Errorf("stdout = %q, want empty (the panic is recovered before anything is written)", stdout)
	}
}

// TestHookPreToolUse_UnknownHarnessIsSilent: an unrecognized --harness
// value is silent — neither envelope runs.
func TestHookPreToolUse_UnknownHarnessIsSilent(t *testing.T) {
	isolateHookEnv(t)
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".codegraph"), 0o755); err != nil {
		t.Fatal(err)
	}
	event := codexEvent(t, "Bash", "s-unknown-harness", "", dir, "rg -n Alpha .")
	stdout, stderr, err := execCmdWithInput(event, "hook", "pretooluse", "--harness", "gemini")
	if err != nil {
		t.Fatalf("unknown harness returned %v, want nil", err)
	}
	if stdout != "" || stderr != "" {
		t.Fatalf("stdout = %q, stderr = %q, want both empty (unknown --harness is silent)", stdout, stderr)
	}
}
