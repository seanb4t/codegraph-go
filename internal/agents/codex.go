package agents

import (
	"fmt"
	"os"
	"path/filepath"
)

// codexTOMLTable is the dotted-key table name spliceTOMLTable/stripTOMLTable
// splice in and out of a Codex config.toml.
const codexTOMLTable = "mcp_servers.codegraph"

// codexTarget implements AgentTarget for Codex CLI (D-09, CODEX-02/03).
// Both scopes: global writes ~/.codex/config.toml plus ~/.codex/AGENTS.md;
// local writes <repo>/.codex/config.toml, which Codex loads only for a
// trusted project (codexTrustNote), plus the repo-root AGENTS.md shared
// with opencode. Both scopes write the shared .agents/skills/codegraph
// package (D-14) and, per the live D-15 verdict, also declare
// .codex/skills and $CODEX_HOME/skills as read-only SkillDirs entries —
// codegraph never writes a second copy there, since Codex does not merge
// same-name skills. Config edits go through the hand-rolled single-table
// splice in toml.go (avoids taking a general TOML dependency). Hooks stay
// HooksNone this plan — 07-07 adds codex-json.
//
// Verified live 2026-09-19 against codex-cli 0.155.0
// (.planning/phases/07-codex-parity/07-LIVE-SESSIONS.md, CODEX-01): a
// trusted project's .codex/config.toml loads (both scopes), an untrusted
// project's does not, project .agents/skills and AGENTS.md are read
// regardless of trust, and the real HOME's own files are never touched by
// this scratch verification.
type codexTarget struct{}

func init() {
	registerTarget(codexTarget{})
}

func (codexTarget) ID() TargetID        { return Codex }
func (codexTarget) DisplayName() string { return "Codex CLI" }

// SupportsLocation is a derivation of the capability table (D-02, D-03).
func (t codexTarget) SupportsLocation(loc Location) bool {
	return t.Capabilities().Supports(loc)
}

// Capabilities is Codex's capability table entry (D-09): both scopes, TOML
// config, the shared skill package plus D-15's read-only skill roots, no
// hooks yet (07-07 adds codex-json).
func (codexTarget) Capabilities() Capabilities {
	return Capabilities{
		Scopes:       []Location{LocationGlobal, LocationLocal},
		ConfigFormat: ConfigFormatTOML,
		Hooks:        HooksNone,
		MCPConfig:    codexConfigPath,
		Instructions: codexInstructionsPath,
		SkillDirs:    codexSkillDirs,
	}
}

// codexConfigPath resolves Codex's MCP config file: local is the
// project-relative <repo>/.codex/config.toml; global is
// ~/.codex/config.toml.
func codexConfigPath(loc Location) (string, error) {
	if loc == LocationLocal {
		return filepath.Join(".codex", "config.toml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codex", "config.toml"), nil
}

// codexInstructionsPath resolves Codex's instructions file: local is the
// repo-root AGENTS.md (shared with opencode, D-11 lands in 07-06); global
// is ~/.codex/AGENTS.md, unchanged.
func codexInstructionsPath(loc Location) (string, error) {
	if loc == LocationLocal {
		return "AGENTS.md", nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codex", "AGENTS.md"), nil
}

// codexSkillDirs resolves Codex's skill directories (D-14, D-15): index 0
// is always the shared package this target writes; a second, read-only
// entry is appended for the Codex-specific root the live evidence showed
// Codex actually reads at that scope — ".codex/skills/codegraph" locally
// (D-15 ".codex/skills read: yes"), "$CODEX_HOME/skills/codegraph"
// (~/.codex/skills/codegraph) globally (D-15 "CODEX_HOME/skills read:
// yes") — 07-LIVE-SESSIONS.md CODEX-01 verdicts, both "yes". codegraph
// never writes to either read-only root (D-14): same-name skills are not
// merged by Codex, so a second physical copy would list twice.
func codexSkillDirs(loc Location) ([]string, error) {
	dirs, err := sharedSkillDirs(loc)
	if err != nil {
		return nil, err
	}
	if loc == LocationLocal {
		return append(dirs, filepath.Join(".codex", "skills", "codegraph")), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return append(dirs, filepath.Join(home, ".codex", "skills", "codegraph")), nil
}

// codexTrustNote returns the D-10 advisory Note every local install adds:
// Codex loads a project's MCP server only once the project is trusted, via
// its own TUI trust prompt or the "trust_level" key it writes into the
// user's ~/.codex/config.toml — codegraph never writes that entry itself
// (D-10; 07-LIVE-SESSIONS.md confirmed a -c override does NOT grant trust,
// so this note never suggests one). Per the live D-16/A2 verdicts (both
// "no": neither the codegraph skill nor the AGENTS.md block is
// trust-gated), the note also says those two are read regardless of
// trust, so a user does not conclude the whole install needs trusting.
// Returns "" at global scope, where no trust concept applies.
func codexTrustNote(loc Location) (string, error) {
	if loc != LocationLocal {
		return "", nil
	}
	root, err := filepath.Abs(".")
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(
		"Codex loads this project's MCP server (%s) only once the project is trusted — accept Codex's trust prompt, or add `trust_level = \"trusted\"` under `[projects.%q]` in ~/.codex/config.toml (codegraph never writes this entry itself). The codegraph skill and the AGENTS.md block are read regardless of trust.",
		root, root,
	), nil
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

// Detect is a derivation of the capability table (D-02, D-03): at local
// scope, an existing .codex/ directory (with or without a config.toml yet)
// counts as "installed", the same detection shape --target auto and the
// agent picker's pre-check both rely on.
func (t codexTarget) Detect(loc Location) DetectionResult {
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
	_, _, already := findTOMLTableRange(readFileOrEmpty(configPath), codexTOMLTable)
	return DetectionResult{
		Installed:         installed,
		AlreadyConfigured: already,
		ConfigPath:        configPath,
	}
}

// Install resolves every path through Capabilities() (D-09): the config
// step refuses (never duplicates) a conflicting existing definition via
// tomlTableConflict before splicing; the instructions step is unchanged in
// shape; installDeclaredSkill writes the shared skill package (D-14); and
// a local install appends codexTrustNote when non-empty (D-10). Each step
// records its own outcome independently — a failed step never
// short-circuits a later one.
func (t codexTarget) Install(loc Location, opts InstallOptions) WriteResult {
	var result WriteResult
	caps := t.Capabilities()

	if configPath, err := caps.MCPConfig(loc); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve codex config path: %w", err))
	} else {
		existed := fileExists(configPath)
		existing := readFileOrEmpty(configPath)
		if cerr := tomlTableConflict(existing, codexTOMLTable); cerr != nil {
			result.Errors = append(result.Errors, fmt.Errorf("%s: %w", configPath, cerr))
		} else {
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
	}

	if instrPath, err := caps.InstructionsPath(loc); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve codex instructions path: %w", err))
	} else {
		fr, err := upsertInstructionsEntry(instrPath, codegraphSectionStart, codegraphSectionEnd, instructionsBody())
		recordFile(&result, instrPath, fr, err)

		// D-12: an AGENTS.override.md beside the instructions file Codex
		// reads shadows AGENTS.md for Codex — the block is still written
		// above (a later install of the override's content could still
		// pull it in), but the user should know Codex will not see it
		// until then. codegraph never writes AGENTS.override.md itself.
		overridePath := filepath.Join(filepath.Dir(instrPath), "AGENTS.override.md")
		if fileExists(overridePath) {
			result.Notes = append(result.Notes, fmt.Sprintf(
				"%s shadows %s for Codex — Codex will not see the codegraph block there until the override includes it (codegraph never writes AGENTS.override.md itself)",
				overridePath, instrPath,
			))
		}
	}

	installDeclaredSkill(&result, t, loc)

	if note, err := codexTrustNote(loc); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve codex trust note: %w", err))
	} else if note != "" {
		result.Notes = append(result.Notes, note)
	}

	return result
}

// Uninstall mirrors Install's path resolution: a conflicting existing
// definition is refused (never partially stripped); a strip that empties
// config.toml entirely removes the file rather than leaving an empty one
// (the removeMarkedSection/removeHookEntry keep-clean precedent); the
// instructions step is gated by instructionsRequestedElsewhere (D-11): the
// repo-root AGENTS.md is shared with opencode at local scope, so codex's
// own block is only removed when no other registered target still
// declares that same file and reports itself configured there —
// uninstallDeclaredSkill mirrors the shared skill package's install step.
func (t codexTarget) Uninstall(loc Location) WriteResult {
	var result WriteResult
	caps := t.Capabilities()

	if configPath, err := caps.MCPConfig(loc); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve codex config path: %w", err))
	} else if !fileExists(configPath) {
		result.Files = append(result.Files, FileResult{Path: configPath, Action: ActionNotFound})
	} else {
		existing := readFileOrEmpty(configPath)
		if cerr := tomlTableConflict(existing, codexTOMLTable); cerr != nil {
			result.Errors = append(result.Errors, fmt.Errorf("%s: %w", configPath, cerr))
		} else {
			updated := stripTOMLTable(existing, codexTOMLTable)
			switch {
			case updated == existing:
				result.Files = append(result.Files, FileResult{Path: configPath, Action: ActionNotFound})
			case updated == "":
				if err := os.Remove(configPath); err != nil {
					result.Errors = append(result.Errors, fmt.Errorf("%s: %w", configPath, err))
				} else {
					result.Files = append(result.Files, FileResult{Path: configPath, Action: ActionRemoved})
				}
			default:
				if err := atomicWriteFile(configPath, updated); err != nil {
					result.Errors = append(result.Errors, fmt.Errorf("%s: %w", configPath, err))
				} else {
					result.Files = append(result.Files, FileResult{Path: configPath, Action: ActionRemoved})
				}
			}
		}
	}

	if instrPath, err := caps.InstructionsPath(loc); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve codex instructions path: %w", err))
	} else if others := instructionsRequestedElsewhere(instrPath, loc, t.ID()); len(others) > 0 {
		result.Files = append(result.Files, FileResult{Path: instrPath, Action: ActionKept})
		result.Notes = append(result.Notes, instructionsKeptNote(instrPath, others))
	} else {
		action, err := removeMarkedSection(instrPath, codegraphSectionStart, codegraphSectionEnd)
		recordAction(&result, instrPath, action, err)
	}

	uninstallDeclaredSkill(&result, t, loc)

	return result
}

// DescribePaths is a derivation of the capability table (D-02, D-03).
func (t codexTarget) DescribePaths(loc Location) []string {
	return describeDeclaredPaths(t, loc)
}
