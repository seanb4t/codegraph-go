package agents

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	claudeassets "github.com/seanb4t/codegraph-go"
)

// claudePreToolFragmentCommand is the literal project-relative command the
// embedded hooks fragment uses for every PreToolUse handler — the value
// claudePreToolUseBlocks rewrites into claudePreToolHookCommand(loc). It
// names the guard script, never the binary, so the registered identity
// survives a change of the binary's location (v0.14.0 Phase 6 D-01b).
const claudePreToolFragmentCommand = "${CLAUDE_PROJECT_DIR}/.claude/hooks/pretooluse-nudge.sh"

// preToolGuardExecPathToken is the single-quoted placeholder the guard
// template carries exactly once, on its binary-path assignment line;
// renderPreToolGuard replaces it with the POSIX-quoted absolute ExecPath.
const preToolGuardExecPathToken = "'@codegraph-exec-path@'"

// claudePreToolGuardPath resolves where install writes the rendered
// PreToolUse guard: .claude/hooks/pretooluse-nudge.sh for local,
// <home>/.claude/hooks/pretooluse-nudge.sh for global, the same shape as
// claudeHooksScriptPath.
func claudePreToolGuardPath(loc Location) (string, error) {
	if loc == LocationLocal {
		return filepath.Join(".claude", "hooks", "pretooluse-nudge.sh"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "hooks", "pretooluse-nudge.sh"), nil
}

// claudePreToolHookCommand returns the command string registered for every
// PreToolUse handler at loc, mirroring claudeHookCommand (D-12): the
// fragment's project-relative literal for local, the absolute guard path
// for global — never the binary path (D-01b).
func claudePreToolHookCommand(loc Location) (string, error) {
	if loc == LocationLocal {
		return claudePreToolFragmentCommand, nil
	}
	return claudePreToolGuardPath(LocationGlobal)
}

// claudePreToolUseBlocks returns the PreToolUse blocks for loc, derived
// from the embedded fragment, plus the single owned command writeHookEntry
// uses for identity.
func claudePreToolUseBlocks(loc Location) ([]any, []string, error) {
	ownCommand, err := claudePreToolHookCommand(loc)
	if err != nil {
		return nil, nil, err
	}
	blocks, err := claudeFragmentEventBlocks("PreToolUse", claudePreToolFragmentCommand, ownCommand)
	if err != nil {
		return nil, nil, err
	}
	return blocks, []string{ownCommand}, nil
}

// shellSingleQuote returns s as one POSIX single-quoted shell word: each
// embedded single quote is emitted as close-quote, backslash-quote,
// reopen-quote, so no byte of s is ever interpreted by the shell (T-06-01).
func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// renderPreToolGuard renders the embedded guard template for execPath,
// the absolute path of the binary running install (D-01b). An empty or
// relative path is rejected — a rendered guard never falls back to PATH —
// and so is a template that does not carry the token exactly once.
func renderPreToolGuard(execPath string) (string, error) {
	if execPath == "" {
		return "", errors.New("render PreToolUse guard: empty binary path")
	}
	if !filepath.IsAbs(execPath) {
		return "", fmt.Errorf("render PreToolUse guard: binary path %q is not absolute", execPath)
	}
	data, err := claudeassets.PreToolUseGuardTemplate()
	if err != nil {
		return "", fmt.Errorf("render PreToolUse guard: %w", err)
	}
	tmpl := string(data)
	if n := strings.Count(tmpl, preToolGuardExecPathToken); n != 1 {
		return "", fmt.Errorf("render PreToolUse guard: template carries the binary-path token %d times, want 1", n)
	}
	return strings.Replace(tmpl, preToolGuardExecPathToken, shellSingleQuote(execPath), 1), nil
}
