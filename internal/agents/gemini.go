package agents

import (
	"fmt"
	"os"
	"path/filepath"
)

// geminiTarget implements AgentTarget for Gemini CLI (D-05a). JSON-stdio
// entry at ~/.gemini/settings.json (global) / ./.gemini/settings.json
// (local); marker-fenced instructions at ~/.gemini/GEMINI.md for global
// but at the PROJECT ROOT ./GEMINI.md for local — NOT
// ./.gemini/GEMINI.md (see the per-agent install-coverage table).
type geminiTarget struct{}

func init() {
	registerTarget(geminiTarget{})
}

func (geminiTarget) ID() TargetID        { return Gemini }
func (geminiTarget) DisplayName() string { return "Gemini CLI" }

// SupportsLocation is a derivation of the capability table (D-02, D-03).
func (t geminiTarget) SupportsLocation(loc Location) bool {
	return t.Capabilities().Supports(loc)
}

// Capabilities is Gemini's capability table entry (D-01, D-02): both
// scopes, JSON config, no hooks, and geminiSkillDirs for its skill
// directory (AGENT-10, D-06 correction (a)).
func (geminiTarget) Capabilities() Capabilities {
	return Capabilities{
		Scopes:       []Location{LocationGlobal, LocationLocal},
		ConfigFormat: ConfigFormatJSON,
		Hooks:        HooksNone,
		MCPConfig:    geminiConfigPath,
		Instructions: geminiInstructionsPath,
		SkillDirs:    geminiSkillDirs,
	}
}

// geminiSkillDirs resolves Gemini CLI's skill directories (AGENT-10, D-06
// correction (a); [CITED: raw.githubusercontent.com/google-gemini/
// gemini-cli/main/docs/cli/skills.md, fetched 2026-09-18]): index 0 — the
// harness-specific `.gemini/skills/codegraph/` directory AGENT-10 names —
// is the one this target WRITES via installDeclaredSkill/
// uninstallDeclaredSkill; index 1, the shared `.agents/skills/codegraph/`
// alias, is a DOCUMENTED READ PATH ONLY, never written here. Per the cited
// docs, Gemini CLI reads both at the same tier and the `.agents/skills/`
// alias actually outranks `.gemini/skills/` on a name collision within a
// tier — so a coexisting shared package (written by Cursor/opencode, same
// embedded content) is harmless: Gemini's own same-tier precedence makes
// the duplicate a silent no-op, never a conflict.
func geminiSkillDirs(loc Location) ([]string, error) {
	var harnessDir string
	if loc == LocationLocal {
		harnessDir = filepath.Join(".gemini", "skills", "codegraph")
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		harnessDir = filepath.Join(home, ".gemini", "skills", "codegraph")
	}
	shared, err := sharedSkillDirPath(loc)
	if err != nil {
		return nil, err
	}
	return []string{harnessDir, shared}, nil
}

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

// Detect is a derivation of the capability table (D-02, D-03).
func (t geminiTarget) Detect(loc Location) DetectionResult {
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

func (t geminiTarget) Install(loc Location, opts InstallOptions) WriteResult {
	var result WriteResult

	if configPath, err := geminiConfigPath(loc); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve gemini config path: %w", err))
	} else {
		fr, err := writeMcpEntry(configPath, func() any {
			return stdioMcpEntry(opts.ExecPath, "serve", "--mcp")
		})
		recordFile(&result, configPath, fr, err)
	}

	if instrPath, err := geminiInstructionsPath(loc); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve gemini instructions path: %w", err))
	} else {
		fr, err := upsertInstructionsEntry(instrPath, codegraphSectionStart, codegraphSectionEnd, instructionsBody())
		recordFile(&result, instrPath, fr, err)
	}

	installDeclaredSkill(&result, t, loc)

	return result
}

func (t geminiTarget) Uninstall(loc Location) WriteResult {
	var result WriteResult

	if configPath, err := geminiConfigPath(loc); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve gemini config path: %w", err))
	} else {
		fr, err := removeMcpEntry(configPath)
		recordFile(&result, configPath, fr, err)
	}

	if instrPath, err := geminiInstructionsPath(loc); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve gemini instructions path: %w", err))
	} else {
		action, err := removeMarkedSection(instrPath, codegraphSectionStart, codegraphSectionEnd)
		recordAction(&result, instrPath, action, err)
	}

	uninstallDeclaredSkill(&result, t, loc)

	return result
}

// DescribePaths is a derivation of the capability table (D-02, D-03).
func (t geminiTarget) DescribePaths(loc Location) []string {
	return describeDeclaredPaths(t, loc)
}
