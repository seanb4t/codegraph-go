package agents

import (
	"fmt"
	"os"
	"path/filepath"
)

// kiroDisabledByDefaultNote is surfaced in WriteResult.Notes on every
// Install call — Kiro IDE ships MCP support disabled by default even with
// a valid config file present (Corrected Per-Agent Parity Table).
const kiroDisabledByDefaultNote = "Kiro IDE ships MCP support disabled by default — enable it in Settings (Kiro CLI users can skip this step)."

// kiroTarget implements AgentTarget for Kiro (D-06). JSON-stdio entry at
// ~/.kiro/settings/mcp.json (global) / ./.kiro/settings/mcp.json (local).
// Writes NO instructions file; a legacy ~/.kiro/steering/codegraph.md (or
// ./.kiro/steering/codegraph.md) left by a prior install is self-heal-
// deleted on install (Pitfall 2).
type kiroTarget struct{}

func init() {
	registerTarget(kiroTarget{})
}

func (kiroTarget) ID() TargetID                   { return Kiro }
func (kiroTarget) DisplayName() string            { return "Kiro" }
func (kiroTarget) SupportsLocation(Location) bool { return true }

func kiroConfigPath(loc Location) (string, error) {
	if loc == LocationLocal {
		return filepath.Join(".kiro", "settings", "mcp.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".kiro", "settings", "mcp.json"), nil
}

// kiroLegacySteeringPath is the legacy instructions file a prior install
// may have left behind; Install self-heal-deletes it if present (Pitfall
// 2) — Kiro's own DescribePaths never lists it as an ongoing write target.
func kiroLegacySteeringPath(loc Location) (string, error) {
	if loc == LocationLocal {
		return filepath.Join(".kiro", "steering", "codegraph.md"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".kiro", "steering", "codegraph.md"), nil
}

func (kiroTarget) Detect(loc Location) DetectionResult {
	configPath, err := kiroConfigPath(loc)
	if err != nil {
		return DetectionResult{}
	}
	installed := fileExists(configPath)
	if !installed {
		installed = fileExists(filepath.Dir(filepath.Dir(configPath)))
	}
	return DetectionResult{
		Installed:         installed,
		AlreadyConfigured: mcpEntryPresent(configPath),
		ConfigPath:        configPath,
	}
}

func (kiroTarget) Install(loc Location, opts InstallOptions) WriteResult {
	var result WriteResult

	if legacy, err := kiroLegacySteeringPath(loc); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve kiro legacy steering path: %w", err))
	} else if fileExists(legacy) {
		if err := os.Remove(legacy); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("%s: %w", legacy, err))
		} else {
			result.Files = append(result.Files, FileResult{Path: legacy, Action: ActionRemoved})
		}
	}

	if configPath, err := kiroConfigPath(loc); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve kiro config path: %w", err))
	} else {
		fr, err := writeMcpEntry(configPath, func() any {
			return stdioMcpEntry(opts.ExecPath, "serve", "--mcp")
		})
		recordFile(&result, configPath, fr, err)
	}

	result.Notes = append(result.Notes, kiroDisabledByDefaultNote)
	return result
}

func (kiroTarget) Uninstall(loc Location) WriteResult {
	var result WriteResult
	configPath, err := kiroConfigPath(loc)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve kiro config path: %w", err))
		return result
	}
	fr, err := removeMcpEntry(configPath)
	recordFile(&result, configPath, fr, err)
	return result
}

func (kiroTarget) DescribePaths(loc Location) []string {
	if p, err := kiroConfigPath(loc); err == nil {
		return []string{p}
	}
	return nil
}
