package agents

import (
	"fmt"
	"os"
	"path/filepath"
)

// cursorTarget implements AgentTarget for Cursor (D-05a). Writes
// mcpServers.codegraph at ~/.cursor/mcp.json (global) / ./.cursor/mcp.json
// (local), with a --path arg Cursor's MCP client needs to locate the
// project: local carries the absolute cwd, global carries the literal
// "${workspaceFolder}" string Cursor itself expands. Writes NO
// instructions file — Cursor's legacy .cursor/rules/codegraph.mdc
// (pre-#529) is actively self-heal-deleted on install, never (re)written
// (Pitfall 2).
type cursorTarget struct{}

func init() {
	registerTarget(cursorTarget{})
}

func (cursorTarget) ID() TargetID        { return Cursor }
func (cursorTarget) DisplayName() string { return "Cursor" }

// SupportsLocation is a derivation of the capability table (D-02, D-03).
func (t cursorTarget) SupportsLocation(loc Location) bool {
	return t.Capabilities().Supports(loc)
}

// Capabilities is Cursor's capability table entry (D-01, D-02): both
// scopes, JSON config, no hooks. Declares no instructions — Cursor's
// legacy .cursor/rules/codegraph.mdc (pre-#529) is actively self-heal-
// deleted on install, never (re)written; its instructions target waits on
// the D-11 live probe (05-06/05-07). SkillDirs is the shared package
// (D-06: Cursor relies on the shared .agents/skills/codegraph path — a
// Cursor-specific directory is added only if a live session shows the
// shared path is not read).
func (cursorTarget) Capabilities() Capabilities {
	return Capabilities{
		Scopes:       []Location{LocationGlobal, LocationLocal},
		ConfigFormat: ConfigFormatJSON,
		Hooks:        HooksNone,
		MCPConfig:    cursorConfigPath,
		SkillDirs:    sharedSkillDirs,
	}
}

func cursorConfigPath(loc Location) (string, error) {
	if loc == LocationLocal {
		return filepath.Join(".cursor", "mcp.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cursor", "mcp.json"), nil
}

// cursorLegacyRulesPath is the pre-#529 instructions file a previous
// install may have left behind; Install self-heal-deletes it if present
// (Pitfall 2) — Cursor's own describePaths never lists it as an ongoing
// write target.
func cursorLegacyRulesPath(loc Location) (string, error) {
	if loc == LocationLocal {
		return filepath.Join(".cursor", "rules", "codegraph.mdc"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cursor", "rules", "codegraph.mdc"), nil
}

// Detect is a derivation of the capability table (D-02, D-03).
func (t cursorTarget) Detect(loc Location) DetectionResult {
	caps := t.Capabilities()
	if !caps.Supports(loc) {
		return DetectionResult{}
	}
	configPath, err := caps.MCPConfig(loc)
	if err != nil {
		return DetectionResult{}
	}
	installed := fileExists(configPath)
	if !installed {
		installed = fileExists(filepath.Dir(configPath))
	}
	return DetectionResult{
		Installed:         installed,
		AlreadyConfigured: mcpEntryPresent(configPath),
		ConfigPath:        configPath,
	}
}

func (t cursorTarget) Install(loc Location, opts InstallOptions) WriteResult {
	var result WriteResult

	if legacy, err := cursorLegacyRulesPath(loc); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve cursor legacy rules path: %w", err))
	} else if fileExists(legacy) {
		if err := os.Remove(legacy); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("%s: %w", legacy, err))
		} else {
			result.Files = append(result.Files, FileResult{Path: legacy, Action: ActionRemoved})
		}
	}

	pathArg := "${workspaceFolder}"
	if loc == LocationLocal {
		if cwd, err := os.Getwd(); err == nil {
			if resolved, err := filepath.EvalSymlinks(cwd); err == nil {
				cwd = resolved
			}
			pathArg = cwd
		}
	}

	// CR-01: a config-path resolution error no longer skips the skill step
	// below — every step records its own outcome via recordFile/result.Errors
	// independently (05-04).
	if configPath, err := cursorConfigPath(loc); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve cursor config path: %w", err))
	} else {
		fr, err := writeMcpEntry(configPath, func() any {
			return stdioMcpEntry(opts.ExecPath, "serve", "--mcp", "--path", pathArg)
		})
		recordFile(&result, configPath, fr, err)
	}

	installDeclaredSkill(&result, t, loc)

	return result
}

func (t cursorTarget) Uninstall(loc Location) WriteResult {
	var result WriteResult
	if configPath, err := cursorConfigPath(loc); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve cursor config path: %w", err))
	} else {
		fr, err := removeMcpEntry(configPath)
		recordFile(&result, configPath, fr, err)
	}

	uninstallDeclaredSkill(&result, t, loc)

	return result
}

// DescribePaths is a derivation of the capability table (D-02, D-03).
func (t cursorTarget) DescribePaths(loc Location) []string {
	return describeDeclaredPaths(t, loc)
}
