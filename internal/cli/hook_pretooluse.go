package cli

import (
	"encoding/json"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/nudge"
)

// maxHookStdinBytes caps how much of a hook event the subcommand reads; a
// larger event is treated as undecodable and ignored (T-06-06).
const maxHookStdinBytes = 1 << 20

// hookNow is the cooldown gate's clock; a seam so tests pin the instant
// instead of waiting (D-17).
var hookNow = time.Now

// hookQualifies is the classifier the adapter consults; a seam so a test
// can force a panic inside the subcommand and prove it is recovered.
var hookQualifies = nudge.Qualifies

// claudePreToolUseInput is the subset of Claude Code's PreToolUse event
// the adapter reads.
type claudePreToolUseInput struct {
	SessionID     string `json:"session_id"`
	AgentID       string `json:"agent_id"`
	HookEventName string `json:"hook_event_name"`
	ToolName      string `json:"tool_name"`
	ToolInput     struct {
		Command  string `json:"command"`
		FilePath string `json:"file_path"`
		Pattern  string `json:"pattern"`
	} `json:"tool_input"`
}

// codexPreToolUseInput is the subset of Codex CLI's PreToolUse event the
// Codex envelope reads (D-21): session_id, agent_id (subagents only — the
// live evidence shows no agent_id on the main thread, 07-LIVE-SESSIONS.md
// B5), hook_event_name, tool_name, cwd, and tool_input.command as a raw
// JSON value since Codex sends it as either a string or an argv array
// (codexShellCommand resolves which).
type codexPreToolUseInput struct {
	SessionID     string `json:"session_id"`
	AgentID       string `json:"agent_id"`
	HookEventName string `json:"hook_event_name"`
	ToolName      string `json:"tool_name"`
	Cwd           string `json:"cwd"`
	ToolInput     struct {
		Command json.RawMessage `json:"command"`
	} `json:"tool_input"`
}

// codexShellToolNames are the tool_name values the Codex envelope maps
// onto the shell rule (D-21): "Bash" is the name the live evidence
// confirmed; "exec_command" and "shell" are defensive aliases for other
// Codex-family tool surfaces the docs mention but this repo has not
// observed live.
var codexShellToolNames = map[string]bool{"Bash": true, "exec_command": true, "shell": true}

// codexShellCommand extracts the shell command string from raw
// (tool_input.command), accepting either a JSON string or a JSON array
// (D-21): Codex may send argv-form for some tool surfaces, and the LAST
// element is used when it is itself a non-empty string. Any doubt — a nil/
// empty raw value, null, an empty string, a non-string/non-array shape, or
// an empty array — resolves to ok=false (silent), matching D-16's "any
// doubt, silent" posture.
func codexShellCommand(raw json.RawMessage) (string, bool) {
	if len(raw) == 0 {
		return "", false
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		if s == "" {
			return "", false
		}
		return s, true
	}
	var arr []json.RawMessage
	if err := json.Unmarshal(raw, &arr); err == nil {
		if len(arr) == 0 {
			return "", false
		}
		var last string
		if err := json.Unmarshal(arr[len(arr)-1], &last); err != nil || last == "" {
			return "", false
		}
		return last, true
	}
	return "", false
}

// claudeHookSpecificOutput carries only the event name and the added
// context — never a permission decision (NUDGE-03).
type claudeHookSpecificOutput struct {
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext"`
}

// claudeHookOutput is the one JSON object printed when the nudge fires.
type claudeHookOutput struct {
	HookSpecificOutput claudeHookSpecificOutput `json:"hookSpecificOutput"`
}

// newHookCmd builds the hidden `codegraph hook` parent: agent hook entry
// points invoked by installed hook scripts, never typed by a human (Phase 6
// D-01a). Hidden and absent from commandGroups, like man.
func newHookCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "hook",
		Short:  "Agent hook entry points (invoked by installed hooks, not by hand)",
		Hidden: true,
	}
	cmd.AddCommand(newHookPreToolUseCmd())
	return cmd
}

// newHookPreToolUseCmd builds `codegraph hook pretooluse`, the Claude Code
// and Codex CLI PreToolUse nudge reached through each harness's installed
// guard script (D-01, D-21). It never returns an error, so
// cmd/codegraph/main.go's print-and-exit-1 path is unreachable from it,
// and its first statement recovers any panic in the whole body, the stdin
// read included (D-01a): the contract is additionalContext-only and exit 0
// on every path (NUDGE-03), for both harness envelopes. An unrecognized
// --harness value is silent — the same "any doubt, silent" posture every
// other decode ambiguity in this file already takes.
func newHookPreToolUseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "pretooluse",
		Short:  "Claude Code and Codex CLI PreToolUse nudge (reads the hook event on stdin)",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			defer func() { _ = recover() }()
			harness, _ := cmd.Flags().GetString("harness")
			switch harness {
			case "codex":
				runHookPreToolUseCodex(cmd.InOrStdin(), cmd.OutOrStdout())
			case "claude":
				runHookPreToolUse(cmd.InOrStdin(), cmd.OutOrStdout(), os.Getenv)
			}
			return nil
		},
	}
	cmd.Flags().String("harness", "claude", "harness envelope to use (claude or codex)")
	return cmd
}

// runHookPreToolUse is the Claude envelope adapter around the
// harness-neutral nudge core: it decodes the event, maps the tool onto a
// nudge.Tool, and prints the pinned context object when the call
// qualifies, a (session, agent) key exists, and that key is outside its
// cooldown. It never returns an error to cobra and never writes to stderr
// (D-01a); the session id is the env id with stdin's as the fallback and
// agent_id comes from stdin only (D-07); every failure, including any
// sentinel error, is silent (D-08). Classification runs before any
// sentinel I/O, so a non-qualifying call touches no file.
func runHookPreToolUse(in io.Reader, out io.Writer, getenv func(string) string) {
	data, err := io.ReadAll(io.LimitReader(in, maxHookStdinBytes+1))
	if err != nil || len(data) == 0 || len(data) > maxHookStdinBytes {
		return
	}
	var event claudePreToolUseInput
	if err := json.Unmarshal(data, &event); err != nil {
		return
	}
	if event.HookEventName != "" && event.HookEventName != "PreToolUse" {
		return
	}

	var tool nudge.Tool
	var input string
	switch event.ToolName {
	case "Bash":
		tool, input = nudge.ToolShell, event.ToolInput.Command
	case "Grep":
		tool, input = nudge.ToolGrep, event.ToolInput.Pattern
	case "Glob":
		tool, input = nudge.ToolGlob, event.ToolInput.Pattern
	case "Read":
		tool, input = nudge.ToolRead, event.ToolInput.FilePath
	default:
		return
	}
	if !hookQualifies(tool, input) {
		return
	}

	// D-07: the env id wins, stdin's is the fallback; never fire unkeyed.
	session := getenv("CLAUDE_CODE_SESSION_ID")
	if session == "" {
		session = event.SessionID
	}
	key, ok := nudge.SessionKey(session, event.AgentID)
	if !ok {
		return
	}
	if !(nudge.Gate{Dir: nudge.DefaultDir(), Now: hookNow}).Due(key) {
		return
	}

	line, err := json.Marshal(claudeHookOutput{HookSpecificOutput: claudeHookSpecificOutput{
		HookEventName:     "PreToolUse",
		AdditionalContext: nudge.Text,
	}})
	if err != nil {
		return
	}
	_, _ = out.Write(append(line, '\n'))
}

// runHookPreToolUseCodex is the Codex envelope adapter around the same
// harness-neutral nudge core runHookPreToolUse uses (D-21): it decodes the
// event, maps tool_name onto the shell rule (Codex has no Grep/Glob/Read
// tools, so only the shell corpora ever fire), keys the cooldown on
// stdin's session_id plus agent_id (main when absent, per D-06/D-07's
// SessionKey contract — no env-var fallback exists for Codex, unlike
// Claude's CLAUDE_CODE_SESSION_ID), and prints the same pinned output
// object. It never returns an error and never writes to stderr, matching
// runHookPreToolUse's contract exactly (D-16).
func runHookPreToolUseCodex(in io.Reader, out io.Writer) {
	data, err := io.ReadAll(io.LimitReader(in, maxHookStdinBytes+1))
	if err != nil || len(data) == 0 || len(data) > maxHookStdinBytes {
		return
	}
	var event codexPreToolUseInput
	if err := json.Unmarshal(data, &event); err != nil {
		return
	}
	if event.HookEventName != "" && event.HookEventName != "PreToolUse" {
		return
	}
	if !codexShellToolNames[event.ToolName] {
		return
	}
	command, ok := codexShellCommand(event.ToolInput.Command)
	if !ok {
		return
	}
	if !hookQualifies(nudge.ToolShell, command) {
		return
	}

	key, ok := nudge.SessionKey(event.SessionID, event.AgentID)
	if !ok {
		return
	}
	if !(nudge.Gate{Dir: nudge.DefaultDir(), Now: hookNow}).Due(key) {
		return
	}

	line, err := json.Marshal(claudeHookOutput{HookSpecificOutput: claudeHookSpecificOutput{
		HookEventName:     "PreToolUse",
		AdditionalContext: nudge.Text,
	}})
	if err != nil {
		return
	}
	_, _ = out.Write(append(line, '\n'))
}
