package agents

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	claudeassets "github.com/seanb4t/codegraph-go"
)

// codexPreToolFragmentCommand is the literal command the embedded Codex
// hooks fragment uses for its PreToolUse handler: a quoted git-root-relative
// path, as the Codex hooks documentation writes it. codexPreToolUseBlocks
// rewrites this literal into codexPreToolHookCommand(loc) for the location
// being installed.
const codexPreToolFragmentCommand = "\"$(git rev-parse --show-toplevel)/.codex/hooks/codegraph-pretooluse.sh\""

// codexHooksJSONPath resolves Codex's hooks.json file: local is the
// project-relative <repo>/.codex/hooks.json; global is hooks.json under
// codexHomeDir.
func codexHooksJSONPath(loc Location) (string, error) {
	if loc == LocationLocal {
		return filepath.Join(".codex", "hooks.json"), nil
	}
	dir, err := codexHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "hooks.json"), nil
}

// codexPreToolGuardPath resolves where install writes the rendered
// PreToolUse guard: .codex/hooks/codegraph-pretooluse.sh for local,
// hooks/codegraph-pretooluse.sh under codexHomeDir for global. The installed
// filename is deliberately different from either embedded template name
// (codegraph-pretooluse-local.sh / -global.sh), so an install run inside
// this repository never overwrites a template.
func codexPreToolGuardPath(loc Location) (string, error) {
	if loc == LocationLocal {
		return filepath.Join(".codex", "hooks", "codegraph-pretooluse.sh"), nil
	}
	dir, err := codexHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "hooks", "codegraph-pretooluse.sh"), nil
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

// codexLegacyGlobalHookCommand is the single-quoted guard path under
// ~/.codex, the global command spelling used whenever CODEX_HOME is unset.
// It is empty when it equals ownCommand or when the home directory cannot
// be resolved. Nothing writes it. It is matched against $CODEX_HOME/hooks.json
// only, where it identifies codegraph's handler when an older install wrote
// it there, for example when ~/.codex links to CODEX_HOME.
func codexLegacyGlobalHookCommand(ownCommand string) string {
	home, err := userHomeDir()
	if err != nil {
		return ""
	}
	legacy := shellSingleQuote(filepath.Join(home, ".codex", "hooks", "codegraph-pretooluse.sh"))
	if legacy == ownCommand {
		return ""
	}
	return legacy
}

// codexPreToolUseBlocks returns the PreToolUse blocks for loc, derived
// from the embedded Codex hooks fragment, plus the owned commands
// writeHookEntry, removeHookEntry and hasOwnHookBlock use for identity: the
// command the blocks carry and, at global scope when it differs, its
// ~/.codex spelling (codexLegacyGlobalHookCommand). This is a Codex-specific
// copy of the claudePreToolUseBlocks/claudeFragmentEventBlocks pattern,
// deliberately not shared code, since the two fragments and command shapes
// diverge: Codex's fragment carries no CLAUDE_PROJECT_DIR-shaped literal and
// has its own quoted command form.
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
	ownCommands := []string{ownCommand}
	if loc == LocationGlobal {
		if legacy := codexLegacyGlobalHookCommand(ownCommand); legacy != "" {
			ownCommands = append(ownCommands, legacy)
		}
	}
	return blocks, ownCommands, nil
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

// checkCodexGuardPath refuses a global guard path that a hook command cannot
// quote for every shell. The command single-quotes the path for a POSIX
// shell, but Codex may run it under another login shell. In fish, for one, a
// backslash escapes the quote, so a path with a single quote or a backslash
// can end the quoting and run the rest as commands. A control character or
// an invalid UTF-8 byte is refused too. The check is deliberately
// conservative, and it does not apply to local paths.
func checkCodexGuardPath(path string) error {
	bad := strings.ContainsAny(path, `'\`) || !utf8.ValidString(path) || strings.ContainsFunc(path, unicode.IsControl)
	if bad {
		return fmt.Errorf("%s: a global Codex hook path with a single quote, a backslash, a control character or invalid UTF-8 is refused, because the shell that runs the hook may not read it as one word; set CODEX_HOME to a path without those characters, or install with --location local", path)
	}
	return nil
}

// codexHooksPreflight reads the hooks.json at loc strictly and checks that
// hooks.PreToolUse can take codegraph's registration. A file whose real
// location is outside the project (at local scope), a malformed file and a
// value of the wrong JSON type are each one error naming the file. Only a
// failure to compute the path is left to the registration step, which
// reports it.
func codexHooksPreflight(loc Location) error {
	path, err := codexHooksJSONPath(loc)
	if err != nil {
		return nil
	}
	real, err := preflightUserFile(path, loc)
	if err != nil {
		return err
	}
	existing, _, err := readJSONFileStrict(real)
	if err == nil {
		_, _, err = hookEventArray(existing, "PreToolUse")
	}
	if err != nil {
		if !strings.Contains(err.Error(), path) {
			err = fmt.Errorf("%s: %w", path, err)
		}
		return err
	}
	return nil
}

// installCodexPreToolNudge renders and writes the guard, then registers it
// in hooks.json. The guard is written first, and a guard-write failure
// skips the registration, so no registration points at a guard this call did
// not write (the same discipline as the Claude install's havePreTool).
func installCodexPreToolNudge(result *WriteResult, loc Location, execPath string) {
	guardPath, err := codexPreToolGuardPath(loc)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve codex PreToolUse guard path: %w", err))
		return
	}
	if loc == LocationGlobal {
		if err := checkCodexGuardPath(guardPath); err != nil {
			result.Errors = append(result.Errors, err)
			return
		}
	}
	rendered, err := renderCodexPreToolGuard(loc, execPath)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("%s: %w", guardPath, err))
		return
	}
	// hooks.json is read before the guard is written, so a value of the wrong
	// type cannot leave a guard that nothing registers.
	if err := codexHooksPreflight(loc); err != nil {
		result.Errors = append(result.Errors, err)
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
	_ = recordUserFile(result, hooksPath, loc, opInstall, func(real string) (FileResult, error) {
		return writeHookEntry(real, "PreToolUse", blocks, ownCommands)
	})
}

// codexHooksExplicitlyDisabled reports whether the config.toml governing
// loc explicitly disables Codex hooks (D-18): for local, the project
// `.codex/config.toml` is read first; if it sets `features.hooks` (any of
// tomlBoolSetting's three recognized forms) that value governs; else if it
// sets the deprecated `features.codex_hooks` that governs instead; else the
// same two checks fall back to the global config.toml under codexHomeDir.
// For global, only the global file is consulted. source names the governing
// file and key, for codexHooksDisabledNote. At local scope, a CODEX_HOME or a
// home directory that does not resolve leaves the global setting unknown: the
// local decision goes on, and note names the bad value. A read error other than
// not-exist is returned unwrapped — the caller records it and writes
// nothing, per the "cannot tell, so don't guess" posture the rest of this
// package's config reads already follow.
func codexHooksExplicitlyDisabled(loc Location) (disabled bool, source, note string, err error) {
	if loc == LocationLocal {
		localPath, err := codexConfigPath(LocationLocal)
		if err != nil {
			return false, "", "", err
		}
		if disabled, source, found, err := codexConfigHooksSetting(localPath); err != nil {
			return false, "", "", err
		} else if found {
			return disabled, source, "", nil
		}
	}

	globalPath, err := codexConfigPath(LocationGlobal)
	if err != nil {
		var homeErr codexHomeError
		var relativeHome relativeHomeError
		if loc == LocationLocal && errors.As(err, &relativeHome) {
			return false, "", "the global Codex config.toml was not checked for a disabled-hooks setting: " + relativeHome.short(), nil
		}
		if loc == LocationLocal && errors.As(err, &homeErr) {
			return false, "", fmt.Sprintf("the global Codex config.toml was not checked for a disabled-hooks setting: %v", err), nil
		}
		return false, "", "", err
	}
	disabled, source, found, err := codexConfigHooksSetting(globalPath)
	if err != nil {
		return false, "", "", err
	}
	if !found {
		return false, "", "", nil
	}
	return disabled, source, "", nil
}

// codexConfigHooksSetting reads path (a Codex config.toml) and reports
// whether it sets the `[features]` table's `hooks` key, falling back to the
// deprecated `codex_hooks` alias when `hooks` is absent — both read through
// tomlBoolSetting, so any of its three recognized forms applies. found is
// false when path does not exist or sets neither key; disabled is the
// logical negation of whatever boolean value was found (hooks=false means
// disabled=true). A missing file is reported found=false, err=nil — the
// same "absent is not an error" posture readFileOrEmpty's callers already
// use throughout this package.
func codexConfigHooksSetting(path string) (disabled bool, source string, found bool, err error) {
	data, rerr := os.ReadFile(path)
	if rerr != nil {
		if os.IsNotExist(rerr) {
			return false, "", false, nil
		}
		return false, "", false, rerr
	}
	content := string(data)
	if v, set := tomlBoolSetting(content, "features", "hooks"); set {
		return !v, fmt.Sprintf("%s ([features] hooks)", path), true, nil
	}
	if v, set := tomlBoolSetting(content, "features", "codex_hooks"); set {
		return !v, fmt.Sprintf("%s ([features] codex_hooks)", path), true, nil
	}
	return false, "", false, nil
}

// codexHookTrustNote returns the Note every opt-in Codex PreToolUse install
// adds: Codex skips a new or changed hook until it is trusted in /hooks, so
// the user must open that review themselves. The note never advises
// `--dangerously-bypass-hook-trust`. When moved is true, the install also
// split a group or shifted a group of the user's, so the note adds that
// those hooks need review too.
func codexHookTrustNote(moved bool) string {
	note := "Codex skips a new or changed hook until you trust it — open /hooks in Codex and trust the codegraph PreToolUse hook"
	if moved {
		note += ", and review your other hooks too: this install moved or regrouped one of them"
	}
	return note
}

// codexHookRemovalNote is the Note an uninstall adds when removing
// codegraph's hook moved or regrouped a hook of the user's.
const codexHookRemovalNote = "Removing the codegraph hook moved or regrouped one of your other hooks, and Codex skips a changed hook until you trust it: review your hooks in /hooks"

// codexForeignHookMoved reports whether files (a WriteResult.Files slice)
// records a hook write that moved or regrouped a foreign hook.
func codexForeignHookMoved(files []FileResult) bool {
	for _, f := range files {
		if f.ForeignHookMoved {
			return true
		}
	}
	return false
}

// codexHooksDisabledNote returns the Note printed when an opt-in
// install is skipped because source (a codexConfigHooksSetting result)
// explicitly disables Codex hooks.
func codexHooksDisabledNote(source string) string {
	return fmt.Sprintf("Codex hooks are explicitly disabled by %s — the codegraph PreToolUse nudge was not installed", source)
}

// codexHooksFileWasWritten reports whether files (a WriteResult.Files
// slice, typically the tail appended by one installCodexPreToolNudge call)
// records hooksPath as created or updated — the Keep-refresh trigger for
// codexHookTrustNote: Keep only re-announces trust when the registered
// definition itself actually changed, never on a byte-identical refresh.
func codexHooksFileWasWritten(files []FileResult, hooksPath string) bool {
	for _, f := range files {
		if f.Path == hooksPath && (f.Action == ActionCreated || f.Action == ActionUpdated) {
			return true
		}
	}
	return false
}

// uninstallCodexPreToolNudge removes the hooks.json registration and then the
// guard, and reports not-found for each when the user never opted in. The
// guard stays, with a note, when the registration may remain: its removal
// failed, or hooks.json was kept because it lies outside the project. op is
// the command that asked: an install that turns the nudge off refuses such a
// local hooks.json, and an uninstall keeps it with a note. When removing the
// registration moved or regrouped a hook of the user's, the result carries a
// note that says to review it.
func uninstallCodexPreToolNudge(result *WriteResult, loc Location, op userFileOp) {
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
	beforeFiles := len(result.Files)
	rerr := recordUserFile(result, hooksPath, loc, op, func(real string) (FileResult, error) {
		return removeHookEntry(real, "PreToolUse", ownCommands)
	})
	if rerr == nil && codexForeignHookMoved(result.Files[beforeFiles:]) {
		result.Notes = append(result.Notes, codexHookRemovalNote)
	}

	if guardPath, err := codexPreToolGuardPath(loc); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve codex PreToolUse guard path: %w", err))
	} else if registrationRemains(result, hooksPath, rerr) {
		keepFileNote(result, guardPath, hooksPath)
	} else {
		fr, rerr := removeEmbeddedFile(guardPath)
		recordFile(result, guardPath, fr, rerr)
	}
}
