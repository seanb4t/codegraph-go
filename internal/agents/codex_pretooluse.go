package agents

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	claudeassets "github.com/seanb4t/codegraph-go"
)

// codexPreToolFragmentCommand is the literal command the embedded Codex
// hooks fragment uses for its PreToolUse handler — the exact D-20 local
// form: a quoted git-root-relative path, matching the Codex docs' own
// example verbatim (07-LIVE-SESSIONS.md, learn.chatgpt.com/docs/hooks).
// codexPreToolUseBlocks rewrites this literal into
// codexPreToolHookCommand(loc) for the location actually being installed.
const codexPreToolFragmentCommand = "\"$(git rev-parse --show-toplevel)/.codex/hooks/codegraph-pretooluse.sh\""

// codexHooksJSONPath resolves Codex's hooks.json file: local is the
// project-relative <repo>/.codex/hooks.json; global is
// ~/.codex/hooks.json.
func codexHooksJSONPath(loc Location) (string, error) {
	if loc == LocationLocal {
		return filepath.Join(".codex", "hooks.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codex", "hooks.json"), nil
}

// codexPreToolGuardPath resolves where install writes the rendered
// PreToolUse guard: .codex/hooks/codegraph-pretooluse.sh for local,
// <home>/.codex/hooks/codegraph-pretooluse.sh for global. The installed
// filename is deliberately different from either embedded template name
// (codegraph-pretooluse-local.sh / -global.sh), so an install run inside
// this repository never overwrites a template.
func codexPreToolGuardPath(loc Location) (string, error) {
	if loc == LocationLocal {
		return filepath.Join(".codex", "hooks", "codegraph-pretooluse.sh"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codex", "hooks", "codegraph-pretooluse.sh"), nil
}

// codexPreToolHookCommand returns the command string registered for the
// PreToolUse handler at loc: the fragment's quoted git-root-relative
// literal for local, a single-quoted absolute guard path for global
// (D-20) — never the binary path (D-19).
func codexPreToolHookCommand(loc Location) (string, error) {
	if loc == LocationLocal {
		return codexPreToolFragmentCommand, nil
	}
	guardPath, err := codexPreToolGuardPath(LocationGlobal)
	if err != nil {
		return "", err
	}
	return shellSingleQuote(guardPath), nil
}

// codexPreToolUseBlocks returns the PreToolUse blocks for loc, derived
// from the embedded Codex hooks fragment, plus the single owned command
// writeHookEntry uses for identity. This is a Codex-specific copy of the
// claudePreToolUseBlocks/claudeFragmentEventBlocks pattern — deliberately
// not shared code, since the two fragments and command shapes diverge
// (D-21): Codex's fragment carries no CLAUDE_PROJECT_DIR-shaped literal
// and its own quoted command form (D-20).
func codexPreToolUseBlocks(loc Location) ([]any, []string, error) {
	ownCommand, err := codexPreToolHookCommand(loc)
	if err != nil {
		return nil, nil, err
	}
	data, err := claudeassets.CodexHooksFragment()
	if err != nil {
		return nil, nil, err
	}
	var decoded struct {
		Hooks map[string][]any `json:"hooks"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil, nil, fmt.Errorf("decode embedded Codex hooks fragment: %w", err)
	}
	source, ok := decoded.Hooks["PreToolUse"]
	if !ok {
		return nil, nil, errors.New("embedded Codex hooks fragment has no hooks.PreToolUse")
	}

	blocks := make([]any, 0, len(source))
	for _, b := range source {
		obj, ok := b.(map[string]any)
		if !ok {
			blocks = append(blocks, b)
			continue
		}
		rewritten := make(map[string]any, len(obj))
		for k, v := range obj {
			rewritten[k] = v
		}
		if entries, ok := rewritten["hooks"].([]any); ok {
			newEntries := make([]any, 0, len(entries))
			for _, e := range entries {
				eo, ok := e.(map[string]any)
				if !ok {
					newEntries = append(newEntries, e)
					continue
				}
				newEO := make(map[string]any, len(eo))
				for k, v := range eo {
					newEO[k] = v
				}
				if cmd, ok := newEO["command"].(string); ok && cmd == codexPreToolFragmentCommand {
					newEO["command"] = ownCommand
				}
				newEntries = append(newEntries, newEO)
			}
			rewritten["hooks"] = newEntries
		}
		blocks = append(blocks, rewritten)
	}
	return blocks, []string{ownCommand}, nil
}

// renderCodexPreToolGuard renders the embedded guard template for loc
// (local or global — D-22 gives them different indexed-repo checks) and
// execPath, the absolute path of the binary running install (D-19). An
// empty or relative path is rejected — a rendered guard never falls back
// to PATH — and so is a template that does not carry the shared
// preToolGuardExecPathToken exactly once.
func renderCodexPreToolGuard(loc Location, execPath string) (string, error) {
	if execPath == "" {
		return "", errors.New("render Codex PreToolUse guard: empty binary path")
	}
	if !filepath.IsAbs(execPath) {
		return "", fmt.Errorf("render Codex PreToolUse guard: binary path %q is not absolute", execPath)
	}
	var data []byte
	var err error
	if loc == LocationLocal {
		data, err = claudeassets.CodexPreToolUseGuardLocalTemplate()
	} else {
		data, err = claudeassets.CodexPreToolUseGuardGlobalTemplate()
	}
	if err != nil {
		return "", fmt.Errorf("render Codex PreToolUse guard: %w", err)
	}
	tmpl := string(data)
	if n := strings.Count(tmpl, preToolGuardExecPathToken); n != 1 {
		return "", fmt.Errorf("render Codex PreToolUse guard: template carries the binary-path token %d times, want 1", n)
	}
	return strings.Replace(tmpl, preToolGuardExecPathToken, shellSingleQuote(execPath), 1), nil
}

// installCodexPreToolNudge renders and writes the guard, then registers it
// in hooks.json — the tracer's On-only path (07-08 gives Keep/Off their
// meaning; D-18). The guard is written first; a guard-write failure skips
// the registration step entirely, so no registration is ever left
// pointing at a guard this call did not successfully write (mirrors
// claude.go's Install havePreTool discipline).
func installCodexPreToolNudge(result *WriteResult, loc Location, execPath string) {
	guardPath, err := codexPreToolGuardPath(loc)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve codex PreToolUse guard path: %w", err))
		return
	}
	rendered, err := renderCodexPreToolGuard(loc, execPath)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("%s: %w", guardPath, err))
		return
	}
	fr, werr := writeEmbeddedFile(guardPath, rendered, true)
	recordFile(result, guardPath, fr, werr)
	if werr != nil {
		return
	}

	hooksPath, err := codexHooksJSONPath(loc)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve codex hooks.json path: %w", err))
		return
	}
	blocks, ownCommands, err := codexPreToolUseBlocks(loc)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("%s: %w", hooksPath, err))
		return
	}
	fr, werr = writeHookEntry(hooksPath, "PreToolUse", blocks, ownCommands)
	recordFile(result, hooksPath, fr, werr)
}

// uninstallCodexPreToolNudge always attempts removal of both the guard and
// the hooks.json registration, reporting not-found when the user never
// opted in — matching the "uninstall always attempts removal" discipline
// already established for every other codegraph-owned artifact in this
// package (D-09/D-11).
func uninstallCodexPreToolNudge(result *WriteResult, loc Location) {
	if guardPath, err := codexPreToolGuardPath(loc); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve codex PreToolUse guard path: %w", err))
	} else {
		fr, rerr := removeEmbeddedFile(guardPath)
		recordFile(result, guardPath, fr, rerr)
	}

	hooksPath, err := codexHooksJSONPath(loc)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve codex hooks.json path: %w", err))
		return
	}
	_, ownCommands, err := codexPreToolUseBlocks(loc)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("%s: %w", hooksPath, err))
		return
	}
	fr, werr := removeHookEntry(hooksPath, "PreToolUse", ownCommands)
	recordFile(result, hooksPath, fr, werr)
}
