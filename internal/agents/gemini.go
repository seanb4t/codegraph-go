package agents

import (
	"os"
	"path/filepath"
)

// geminiTarget implements AgentTarget for Gemini CLI (D-05a). JSON-stdio
// entry at ~/.gemini/settings.json (global) / ./.gemini/settings.json
// (local); marker-fenced instructions at ~/.gemini/GEMINI.md for global
// but at the PROJECT ROOT ./GEMINI.md for local — NOT
// ./.gemini/GEMINI.md (Corrected Per-Agent Parity Table).
type geminiTarget struct{}

func init() {
	registerTarget(geminiTarget{})
}

func (geminiTarget) ID() TargetID                   { return Gemini }
func (geminiTarget) DisplayName() string            { return "Gemini CLI" }
func (geminiTarget) SupportsLocation(Location) bool { return true }

func geminiConfigPath(loc Location) (string, error) {
	if loc == LocationLocal {
		return filepath.Join(".gemini", "settings.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".gemini", "settings.json"), nil
}

// geminiInstructionsPath: global is ~/.gemini/GEMINI.md; local is the
// PROJECT ROOT ./GEMINI.md, never ./.gemini/GEMINI.md (D-05a).
func geminiInstructionsPath(loc Location) (string, error) {
	if loc == LocationLocal {
		return "GEMINI.md", nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".gemini", "GEMINI.md"), nil
}

func (geminiTarget) Detect(loc Location) DetectionResult {
	configPath, err := geminiConfigPath(loc)
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

func (geminiTarget) Install(loc Location, opts InstallOptions) WriteResult {
	var result WriteResult

	if configPath, err := geminiConfigPath(loc); err == nil {
		if fr, err := writeMcpEntry(configPath, func() any {
			return stdioMcpEntry(opts.ExecPath, "serve", "--mcp")
		}); err == nil {
			result.Files = append(result.Files, fr)
		}
	}

	if instrPath, err := geminiInstructionsPath(loc); err == nil {
		if fr, err := upsertInstructionsEntry(instrPath, codegraphSectionStart, codegraphSectionEnd, instructionsBody()); err == nil {
			result.Files = append(result.Files, fr)
		}
	}

	return result
}

func (geminiTarget) Uninstall(loc Location) WriteResult {
	var result WriteResult

	if configPath, err := geminiConfigPath(loc); err == nil {
		if fr, err := removeMcpEntry(configPath); err == nil {
			result.Files = append(result.Files, fr)
		}
	}

	if instrPath, err := geminiInstructionsPath(loc); err == nil {
		if action, err := removeMarkedSection(instrPath, codegraphSectionStart, codegraphSectionEnd); err == nil {
			result.Files = append(result.Files, FileResult{Path: instrPath, Action: action})
		}
	}

	return result
}

func (geminiTarget) DescribePaths(loc Location) []string {
	var paths []string
	if p, err := geminiConfigPath(loc); err == nil {
		paths = append(paths, p)
	}
	if p, err := geminiInstructionsPath(loc); err == nil {
		paths = append(paths, p)
	}
	return paths
}
