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
// PreToolUse nudge reached through the installed guard script (D-01). It
// never returns an error, so cmd/codegraph/main.go's print-and-exit-1 path
// is unreachable from it, and it recovers any panic (D-01a): the contract
// is additionalContext-only and exit 0 on every path (NUDGE-03).
func newHookPreToolUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "pretooluse",
		Short:  "Claude Code PreToolUse nudge (reads the hook event on stdin)",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			defer func() { _ = recover() }()
			runHookPreToolUse(cmd.InOrStdin(), cmd.OutOrStdout(), os.Getenv)
			return nil
		},
	}
}

// runHookPreToolUse is the Claude envelope adapter around the
// harness-neutral nudge core: it decodes the event, maps the tool onto a
// nudge.Tool, and prints the pinned context object when the call
// qualifies and a session key exists. Every failure is silent.
func runHookPreToolUse(in io.Reader, out io.Writer, getenv func(string) string) {
	data, err := io.ReadAll(io.LimitReader(in, maxHookStdinBytes+1))
	if err != nil || len(data) == 0 || len(data) > maxHookStdinBytes {
		return
	}
	var event claudePreToolUseInput
	if err := json.Unmarshal(data, &event); err != nil {
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
	if !nudge.Qualifies(tool, input) {
		return
	}

	// D-07: the env id wins, stdin's is the fallback; never fire unkeyed.
	session := getenv("CLAUDE_CODE_SESSION_ID")
	if session == "" {
		session = event.SessionID
	}
	if session == "" {
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
