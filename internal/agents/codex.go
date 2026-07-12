package agents

import (
	"fmt"
	"os"
	"path/filepath"
)

// codexTOMLTable is the dotted-key table name spliceTOMLTable/stripTOMLTable
// splice in and out of ~/.codex/config.toml.
const codexTOMLTable = "mcp_servers.codegraph"

// codexTarget implements AgentTarget for Codex CLI (D-05a, D-07). Global
// scope only — Codex has no per-project config concept TS integrates with,
// so SupportsLocation(LocationLocal) is false and Install/Uninstall at
// local are no-ops. Edits ~/.codex/config.toml via the hand-rolled
// single-table splice in toml.go (mirrors TS's own toml.ts — no general
// TOML dependency) and upserts a marker-fenced ~/.codex/AGENTS.md block,
// one of only 4 of 8 targets that gets an instructions file.
type codexTarget struct{}

func init() {
	registerTarget(codexTarget{})
}

func (codexTarget) ID() TargetID                       { return Codex }
func (codexTarget) DisplayName() string                { return "Codex CLI" }
func (codexTarget) SupportsLocation(loc Location) bool { return loc == LocationGlobal }

func codexConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codex", "config.toml"), nil
}

func codexInstructionsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codex", "AGENTS.md"), nil
}

// codexTableBody renders the [mcp_servers.codegraph] table body lines for
// execPath — command = "<execPath>", args = ["serve", "--mcp"].
func codexTableBody(execPath string) []string {
	return []string{
		"command = " + tomlString(execPath),
		"args = " + tomlStringArray([]string{"serve", "--mcp"}),
	}
}

func readFileOrEmpty(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

func (codexTarget) Detect(loc Location) DetectionResult {
	if loc != LocationGlobal {
		return DetectionResult{}
	}
	configPath, err := codexConfigPath()
	if err != nil {
		return DetectionResult{}
	}
	installed := fileExists(configPath)
	if !installed {
		if home, herr := os.UserHomeDir(); herr == nil {
			installed = fileExists(filepath.Join(home, ".codex"))
		}
	}
	_, _, already := findTOMLTableRange(readFileOrEmpty(configPath), codexTOMLTable)
	return DetectionResult{
		Installed:         installed,
		AlreadyConfigured: already,
		ConfigPath:        configPath,
	}
}

func (codexTarget) Install(loc Location, opts InstallOptions) WriteResult {
	var result WriteResult
	if loc != LocationGlobal {
		return result
	}

	if configPath, err := codexConfigPath(); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve codex config path: %w", err))
	} else {
		existed := fileExists(configPath)
		existing := readFileOrEmpty(configPath)
		updated := spliceTOMLTable(existing, codexTOMLTable, codexTableBody(opts.ExecPath))
		if updated == existing {
			result.Files = append(result.Files, FileResult{Path: configPath, Action: ActionUnchanged})
		} else if err := atomicWriteFile(configPath, updated); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("%s: %w", configPath, err))
		} else {
			action := ActionUpdated
			if !existed {
				action = ActionCreated
			}
			result.Files = append(result.Files, FileResult{Path: configPath, Action: action})
		}
	}

	if instrPath, err := codexInstructionsPath(); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve codex instructions path: %w", err))
	} else {
		fr, err := upsertInstructionsEntry(instrPath, codegraphSectionStart, codegraphSectionEnd, instructionsBody())
		recordFile(&result, instrPath, fr, err)
	}

	return result
}

func (codexTarget) Uninstall(loc Location) WriteResult {
	var result WriteResult
	if loc != LocationGlobal {
		return result
	}

	if configPath, err := codexConfigPath(); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve codex config path: %w", err))
	} else if !fileExists(configPath) {
		result.Files = append(result.Files, FileResult{Path: configPath, Action: ActionNotFound})
	} else {
		existing := readFileOrEmpty(configPath)
		updated := stripTOMLTable(existing, codexTOMLTable)
		if updated == existing {
			result.Files = append(result.Files, FileResult{Path: configPath, Action: ActionNotFound})
		} else if err := atomicWriteFile(configPath, updated); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("%s: %w", configPath, err))
		} else {
			result.Files = append(result.Files, FileResult{Path: configPath, Action: ActionRemoved})
		}
	}

	if instrPath, err := codexInstructionsPath(); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve codex instructions path: %w", err))
	} else {
		action, err := removeMarkedSection(instrPath, codegraphSectionStart, codegraphSectionEnd)
		recordAction(&result, instrPath, action, err)
	}

	return result
}

func (codexTarget) DescribePaths(loc Location) []string {
	if loc != LocationGlobal {
		return nil
	}
	var paths []string
	if p, err := codexConfigPath(); err == nil {
		paths = append(paths, p)
	}
	if p, err := codexInstructionsPath(); err == nil {
		paths = append(paths, p)
	}
	return paths
}
