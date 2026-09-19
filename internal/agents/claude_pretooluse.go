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

// preToolNudgeEvidenced widens preToolNudgeRecorded's manifest-only check
// (code review CR-01, 06-REVIEW.md): a Claude skill directory that is a
// symlinked shared directory holding pre-existing foreign, unmanifested
// content (D-14/D-17) makes Install skip its manifest-record step entirely
// (claude.go's `haveSkillMDContent` gate), even though the PreToolUse
// guard and its settings.json registration were written successfully. The
// manifest alone then silently under-reports a genuine, live opt-in,
// breaking D-10's "sticky until explicitly turned off" promise and
// upgrade's guard-refresh path.
//
// recorded is true when EITHER preToolNudgeRecorded's manifest check is
// true OR settings.json at loc already carries one of codegraph's own
// PreToolUse command blocks by EXACT identity — the same ownership test
// writeHookEntry/removeHookEntry themselves use (242ec0a), read here
// through hasOwnHookBlock's read-only probe rather than any write path.
// This only ever widens "recorded" using evidence codegraph itself would
// have written under an earlier explicit opt-in; it can never make Keep
// ADD the hook somewhere no such evidence exists.
//
// The settings.json widening applies ONLY when the manifest is genuinely
// ABSENT (preToolNudgeRecorded's readable-but-not-recorded case — exactly
// what a foreign/unmanifested skill directory produces, since no manifest
// is ever written there). A manifest that EXISTS but cannot be read or
// decoded is a distinct, deliberately more conservative failure mode:
// TestPreToolNudge_KeepWithUnreadableManifestTouchesNothing (06-04) pins
// "cannot tell, so touch nothing" for that case regardless of
// settings.json's content, and this function preserves that verdict
// unchanged — corruption of a manifest that WAS written is a different,
// untrusted signal than one that was never written at all, so it is never
// overridden by evidence found elsewhere.
func preToolNudgeEvidenced(loc Location) (recorded, readable bool) {
	manifestRecorded, manifestReadable := preToolNudgeRecorded(loc)
	if manifestRecorded {
		return true, true
	}
	if !manifestReadable {
		return false, false
	}

	settingsPath, err := claudeSettingsPath(loc)
	if err != nil {
		return false, true
	}
	_, ownCommands, err := claudePreToolUseBlocks(loc)
	if err != nil {
		return false, true
	}
	settingsEvidenced, err := hasOwnHookBlock(settingsPath, "PreToolUse", ownCommands)
	if err != nil {
		// settings.json itself is unreadable/corrupt: report not-recorded
		// rather than claiming evidence this read could not confirm. The
		// manifest side already established readable=true (genuinely
		// absent), so that verdict stands.
		return false, true
	}
	return settingsEvidenced, true
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
