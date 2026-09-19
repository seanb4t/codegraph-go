// Package nudge is the harness-neutral core of the PreToolUse nudge
// (Phase 6, D-01a): it decides whether a tool call qualifies for the
// codegraph pointer and holds the pinned pointer text. It knows nothing
// about any harness's stdin/stdout envelope and imports nothing from this
// module, so the Codex nudge (CODEX-05) reuses it behind its own adapter.
package nudge

// Tool is a harness-neutral tool family. Each harness adapter maps its own
// tool names and input fields onto one of these.
type Tool string

const (
	ToolShell Tool = "shell" // Claude Bash; input = tool_input.command
	ToolGrep  Tool = "grep"  // Claude Grep; input = tool_input.pattern (ignored)
	ToolGlob  Tool = "glob"  // Claude Glob; input = tool_input.pattern (ignored)
	ToolRead  Tool = "read"  // Claude Read; input = tool_input.file_path
)

// Qualifies reports whether a call to tool with input should receive the
// nudge. Grep and Glob always qualify (D-03). Shell (first-word
// grep/egrep/fgrep/rg/find, D-02) and Read (unless an obvious non-code
// file, D-03) are filled in by plan 06-02; until then they never qualify,
// so the tracer can only stay silent on them, never over-fire.
func Qualifies(tool Tool, input string) bool {
	switch tool {
	case ToolGrep, ToolGlob:
		return true
	default:
		return false
	}
}
