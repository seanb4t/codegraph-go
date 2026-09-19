// Package nudge is the harness-neutral core of the PreToolUse nudge
// (Phase 6, D-01a): it decides whether a tool call qualifies for the
// codegraph pointer and holds the pinned pointer text. It knows nothing
// about any harness's stdin/stdout envelope and imports nothing from this
// module, so the Codex nudge (CODEX-05) reuses it behind its own adapter.
package nudge

import (
	"path/filepath"
	"regexp"
	"strings"
)

// Tool is a harness-neutral tool family. Each harness adapter maps its own
// tool names and input fields onto one of these.
type Tool string

const (
	ToolShell Tool = "shell" // Claude Bash; input = tool_input.command
	ToolGrep  Tool = "grep"  // Claude Grep; input = tool_input.pattern (ignored)
	ToolGlob  Tool = "glob"  // Claude Glob; input = tool_input.pattern (ignored)
	ToolRead  Tool = "read"  // Claude Read; input = tool_input.file_path
)

// searchCommands are the shell first words that qualify (D-02).
var searchCommands = map[string]bool{
	"grep": true, "egrep": true, "fgrep": true, "rg": true, "find": true,
}

// nonCodeExtensions are the lower-cased Read extensions that never qualify
// (D-03).
var nonCodeExtensions = map[string]bool{
	".md": true, ".json": true, ".yaml": true, ".yml": true,
	".toml": true, ".txt": true, ".lock": true,
}

// assignmentRe matches a leading NAME=value shell assignment word.
var assignmentRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)

// Qualifies reports whether a call to tool with input should receive the
// nudge. It is the authoritative layer of a two-layer check and is shared,
// with the D-15 corpora in testdata/, by the Codex nudge (CODEX-05).
//
// Shell (D-02): the harness pre-filter (Claude's `if` rules such as
// `Bash(grep *)`) also matches pipe tails like `git log | grep x`, so only
// the command's first word counts here: after skipping leading NAME=value
// assignments it must be exactly grep, egrep, fgrep, rg or find. Any parse
// doubt stays silent — an assignment carrying a quote, backtick, `$` or
// backslash, an empty command, or one made only of assignments.
//
// Grep and Glob always qualify; Read qualifies unless the path is empty or
// an obvious non-code file by extension (D-03). Any other tool never does.
func Qualifies(tool Tool, input string) bool {
	switch tool {
	case ToolGrep, ToolGlob:
		return true
	case ToolShell:
		return shellQualifies(input)
	case ToolRead:
		if input == "" {
			return false
		}
		return !nonCodeExtensions[strings.ToLower(filepath.Ext(input))]
	default:
		return false
	}
}

func shellQualifies(command string) bool {
	fields := strings.Fields(command)
	i := 0
	for ; i < len(fields) && assignmentRe.MatchString(fields[i]); i++ {
		if strings.ContainsAny(fields[i], "'\"`$\\") {
			return false
		}
	}
	if i == len(fields) {
		return false
	}
	return searchCommands[fields[i]]
}
