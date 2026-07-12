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

func (cursorTarget) ID() TargetID                   { return Cursor }
func (cursorTarget) DisplayName() string            { return "Cursor" }
func (cursorTarget) SupportsLocation(Location) bool { return true }

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

func (cursorTarget) Detect(loc Location) DetectionResult {
	configPath, err := cursorConfigPath(loc)
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

func (cursorTarget) Install(loc Location, opts InstallOptions) WriteResult {
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

	configPath, err := cursorConfigPath(loc)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve cursor config path: %w", err))
		return result
	}
	fr, err := writeMcpEntry(configPath, func() any {
		return stdioMcpEntry(opts.ExecPath, "serve", "--mcp", "--path", pathArg)
	})
	recordFile(&result, configPath, fr, err)
	return result
}

func (cursorTarget) Uninstall(loc Location) WriteResult {
	var result WriteResult
	configPath, err := cursorConfigPath(loc)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve cursor config path: %w", err))
		return result
	}
	fr, err := removeMcpEntry(configPath)
	recordFile(&result, configPath, fr, err)
	return result
}

func (cursorTarget) DescribePaths(loc Location) []string {
	if p, err := cursorConfigPath(loc); err == nil {
		return []string{p}
	}
	return nil
}
