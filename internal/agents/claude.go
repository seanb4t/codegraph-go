package agents

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// claudeAllowToken is the permission entry Claude's settings.json needs so
// AutoAllow-installed users aren't prompted per-tool for every
// mcp__codegraph__* call (D-05).
const claudeAllowToken = "mcp__codegraph__*"

// claudeTarget implements AgentTarget for Claude Code (D-02, D-05, D-07,
// D-08). Global scope writes ~/.claude.json; local scope writes
// ./.mcp.json — NEVER ./.claude.json, which Claude Code silently never
// reads (Pitfall 3, TS issue #207). Both scopes upsert a marker-fenced
// CLAUDE.md instructions block and, when InstallOptions.AutoAllow is set,
// append "mcp__codegraph__*" to settings.json's permissions.allow list.
type claudeTarget struct{}

func init() {
	registerTarget(claudeTarget{})
}

func (claudeTarget) ID() TargetID                   { return Claude }
func (claudeTarget) DisplayName() string            { return "Claude Code" }
func (claudeTarget) SupportsLocation(Location) bool { return true }

// fileExists reports whether path exists (any file type), swallowing stat
// errors that indicate genuine absence — every per-agent Detect
// implementation uses this to check for an agent's own config/dir (D-03).
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// stdioMcpEntry builds the {type:"stdio", command, args} entry shape
// shared by Claude, Cursor, Gemini, and Kiro (Pattern 2). Antigravity does
// NOT use this — it has its own no-type entry builder (Pitfall 6).
func stdioMcpEntry(execPath string, args ...string) map[string]any {
	return map[string]any{
		"type":    "stdio",
		"command": execPath,
		"args":    args,
	}
}

// mcpEntryPresent reports whether path's JSON already has an
// mcpServers.codegraph entry, used by Detect's AlreadyConfigured field.
func mcpEntryPresent(path string) bool {
	existing, err := readJSONFile(path)
	if err != nil {
		return false
	}
	mcpServers, _ := existing["mcpServers"].(map[string]any)
	if mcpServers == nil {
		return false
	}
	_, ok := mcpServers["codegraph"]
	return ok
}

// instructionsBody returns codegraphInstructionsBlock's content with the
// surrounding markers and their adjoining newlines stripped, for callers
// that pass content to upsertInstructionsEntry (which re-adds the
// startMarker + "\n" + content + "\n" + endMarker wrapping itself) — the
// two compose back to codegraphInstructionsBlock byte-for-byte (D-01a).
func instructionsBody() string {
	s := strings.TrimPrefix(codegraphInstructionsBlock, codegraphSectionStart+"\n")
	s = strings.TrimSuffix(s, "\n"+codegraphSectionEnd)
	return s
}

// claudeConfigPath returns the MCP config file for loc: ~/.claude.json for
// global, ./.mcp.json for local (Pitfall 3 — never ./.claude.json).
func claudeConfigPath(loc Location) (string, error) {
	if loc == LocationLocal {
		return ".mcp.json", nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude.json"), nil
}

// claudeLegacyLocalConfigPath is the pre-#207 incorrect local-scope file a
// previous install may have written to. Install migrates any codegraph
// entry found here into claudeConfigPath(local); Uninstall strips it from
// both locations (Pitfall 3).
func claudeLegacyLocalConfigPath() string {
	return ".claude.json"
}

func claudeInstructionsPath(loc Location) (string, error) {
	if loc == LocationLocal {
		return filepath.Join(".claude", "CLAUDE.md"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "CLAUDE.md"), nil
}

func claudeSettingsPath(loc Location) (string, error) {
	if loc == LocationLocal {
		return filepath.Join(".claude", "settings.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "settings.json"), nil
}

// addClaudeAllowPermission appends claudeAllowToken to permissions.allow in
// path's JSON if absent, idempotently (D-05).
func addClaudeAllowPermission(path string) (FileResult, error) {
	existing, err := readJSONFile(path)
	if err != nil {
		return FileResult{}, err
	}
	existedBefore := fileExists(path)
	permissions, _ := existing["permissions"].(map[string]any)
	if permissions == nil {
		permissions = map[string]any{}
	}
	allow, _ := permissions["allow"].([]any)
	for _, v := range allow {
		if s, ok := v.(string); ok && s == claudeAllowToken {
			return FileResult{Path: path, Action: ActionUnchanged}, nil
		}
	}
	allow = append(allow, claudeAllowToken)
	permissions["allow"] = allow
	existing["permissions"] = permissions
	if err := writeJSONFile(path, existing); err != nil {
		return FileResult{}, err
	}
	action := ActionCreated
	if existedBefore {
		action = ActionUpdated
	}
	return FileResult{Path: path, Action: action}, nil
}

// removeClaudeAllowPermission removes claudeAllowToken from
// permissions.allow in path's JSON if present, leaving every other allow
// entry and unrelated key untouched (D-05, T-06-02-01).
func removeClaudeAllowPermission(path string) (FileResult, error) {
	existing, err := readJSONFile(path)
	if err != nil {
		return FileResult{}, err
	}
	permissions, ok := existing["permissions"].(map[string]any)
	if !ok {
		return FileResult{Path: path, Action: ActionNotFound}, nil
	}
	allow, ok := permissions["allow"].([]any)
	if !ok {
		return FileResult{Path: path, Action: ActionNotFound}, nil
	}
	found := false
	newAllow := make([]any, 0, len(allow))
	for _, v := range allow {
		if s, ok := v.(string); ok && s == claudeAllowToken {
			found = true
			continue
		}
		newAllow = append(newAllow, v)
	}
	if !found {
		return FileResult{Path: path, Action: ActionNotFound}, nil
	}
	if len(newAllow) == 0 {
		delete(permissions, "allow")
	} else {
		permissions["allow"] = newAllow
	}
	if len(permissions) == 0 {
		delete(existing, "permissions")
	} else {
		existing["permissions"] = permissions
	}
	if err := writeJSONFile(path, existing); err != nil {
		return FileResult{}, err
	}
	return FileResult{Path: path, Action: ActionRemoved}, nil
}

func (claudeTarget) Detect(loc Location) DetectionResult {
	configPath, err := claudeConfigPath(loc)
	if err != nil {
		return DetectionResult{}
	}
	installed := fileExists(configPath)
	if !installed && loc == LocationGlobal {
		if home, herr := os.UserHomeDir(); herr == nil {
			installed = fileExists(filepath.Join(home, ".claude"))
		}
	}
	return DetectionResult{
		Installed:         installed,
		AlreadyConfigured: mcpEntryPresent(configPath),
		ConfigPath:        configPath,
	}
}

func (claudeTarget) Install(loc Location, opts InstallOptions) WriteResult {
	var result WriteResult

	// Pitfall 3: migrate a legacy ./.claude.json local entry into
	// ./.mcp.json before writing the correct entry.
	if loc == LocationLocal {
		legacyPath := claudeLegacyLocalConfigPath()
		if fr, err := removeMcpEntry(legacyPath); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("%s: %w", legacyPath, err))
		} else if fr.Action == ActionRemoved {
			result.Files = append(result.Files, fr)
		}
	}

	if configPath, err := claudeConfigPath(loc); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve claude config path: %w", err))
	} else {
		fr, err := writeMcpEntry(configPath, func() any {
			return stdioMcpEntry(opts.ExecPath, "serve", "--mcp")
		})
		recordFile(&result, configPath, fr, err)
	}

	if instrPath, err := claudeInstructionsPath(loc); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve claude instructions path: %w", err))
	} else {
		fr, err := upsertInstructionsEntry(instrPath, codegraphSectionStart, codegraphSectionEnd, instructionsBody())
		recordFile(&result, instrPath, fr, err)
	}

	if opts.AutoAllow {
		if settingsPath, err := claudeSettingsPath(loc); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("resolve claude settings path: %w", err))
		} else {
			fr, err := addClaudeAllowPermission(settingsPath)
			recordFile(&result, settingsPath, fr, err)
		}
	}

	return result
}

func (claudeTarget) Uninstall(loc Location) WriteResult {
	var result WriteResult

	if configPath, err := claudeConfigPath(loc); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve claude config path: %w", err))
	} else {
		fr, err := removeMcpEntry(configPath)
		recordFile(&result, configPath, fr, err)
	}

	if loc == LocationLocal {
		legacyPath := claudeLegacyLocalConfigPath()
		fr, err := removeMcpEntry(legacyPath)
		recordFile(&result, legacyPath, fr, err)
	}

	if instrPath, err := claudeInstructionsPath(loc); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve claude instructions path: %w", err))
	} else {
		action, err := removeMarkedSection(instrPath, codegraphSectionStart, codegraphSectionEnd)
		recordAction(&result, instrPath, action, err)
	}

	if settingsPath, err := claudeSettingsPath(loc); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve claude settings path: %w", err))
	} else {
		fr, err := removeClaudeAllowPermission(settingsPath)
		recordFile(&result, settingsPath, fr, err)
	}

	return result
}

func (claudeTarget) DescribePaths(loc Location) []string {
	var paths []string
	if p, err := claudeConfigPath(loc); err == nil {
		paths = append(paths, p)
	}
	if p, err := claudeInstructionsPath(loc); err == nil {
		paths = append(paths, p)
	}
	if p, err := claudeSettingsPath(loc); err == nil {
		paths = append(paths, p)
	}
	return paths
}
