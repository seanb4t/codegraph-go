package agents

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/seanb4t/codegraph-go/internal/realpath"
)

// codexTOMLTable is the dotted-key table name spliceTOMLTable/stripTOMLTable
// splice in and out of a Codex config.toml.
const codexTOMLTable = "mcp_servers.codegraph"

// codexTarget implements AgentTarget for Codex CLI (D-09, CODEX-02/03).
// Both scopes: global writes config.toml plus AGENTS.md under codexHomeDir;
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
// (planning-archive-v0.15.0:.planning/milestones/v0.14.0-phases/07-codex-parity/07-LIVE-SESSIONS.md):
// a trusted project's .codex/config.toml loads (both scopes), an untrusted
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
// config, the shared skill package plus D-15's read-only skill roots, and
// the opt-in PreToolUse nudge's codex-json hooks mechanism (07-07, D-18).
func (codexTarget) Capabilities() Capabilities {
	return Capabilities{
		Scopes:       []Location{LocationGlobal, LocationLocal},
		ConfigFormat: ConfigFormatTOML,
		Hooks:        HooksCodexJSON,
		MCPConfig:    codexConfigPath,
		Instructions: codexInstructionsPath,
		SkillDirs:    codexSkillDirs,
	}
}

// codexHomeError reports a CODEX_HOME that is set but cannot name Codex's
// home: it is not an absolute path, or it is not an existing directory.
type codexHomeError struct {
	value, problem string
	cause          error
}

func (e codexHomeError) Error() string {
	msg := fmt.Sprintf("CODEX_HOME %q %s", e.value, e.problem)
	// A missing directory says enough. Any other cause (permission denied, a
	// symlink loop) is part of the message.
	if e.cause != nil && !errors.Is(e.cause, fs.ErrNotExist) {
		msg += ": " + e.cause.Error()
	}
	return msg
}

func (e codexHomeError) Unwrap() error { return e.cause }

// codexHomeDir resolves the directory Codex keeps its global files in:
// $CODEX_HOME when it is set and non-empty, else ~/.codex. Every global
// Codex path (config.toml, AGENTS.md, hooks.json, the PreToolUse guard and
// the read-only skill root) goes through it, so install, uninstall and
// status agree about where Codex's home is. A set CODEX_HOME resolves the way
// Codex resolves it, with every symlink followed (so ".." after a link names
// the link target's parent), and must be an existing absolute directory.
// Anything else is a codexHomeError, never a fallback: install must not
// write to ~/.codex when the user asked for another directory, must not
// create a mistyped one, and a relative guard path registered in hooks.json
// fails from every other working directory. The default ~/.codex is created
// when needed. Only global operations need this. A local install names the
// global config.toml in codexTrustNote and reads its hooks setting, and both
// cope with the error.
func codexHomeDir() (string, error) {
	if v := os.Getenv("CODEX_HOME"); v != "" {
		if !filepath.IsAbs(v) {
			return "", codexHomeError{value: v, problem: "is not an absolute path"}
		}
		real, err := realpath.Resolve(v)
		if err != nil {
			return "", codexHomeError{value: v, problem: "cannot be resolved", cause: err}
		}
		info, err := os.Stat(real)
		if err != nil {
			return "", codexHomeError{value: v, problem: "is not an existing directory", cause: err}
		}
		if !info.IsDir() {
			return "", codexHomeError{value: v, problem: "is not an existing directory"}
		}
		return real, nil
	}
	home, err := userHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codex"), nil
}

// codexConfigPath resolves Codex's MCP config file: local is the
// project-relative <repo>/.codex/config.toml; global is
// config.toml under codexHomeDir.
func codexConfigPath(loc Location) (string, error) {
	if loc == LocationLocal {
		return filepath.Join(".codex", "config.toml"), nil
	}
	dir, err := codexHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.toml"), nil
}

// codexInstructionsPath resolves Codex's instructions file: local is the
// repo-root AGENTS.md (shared with opencode, D-11 lands in 07-06); global
// is AGENTS.md under codexHomeDir.
func codexInstructionsPath(loc Location) (string, error) {
	if loc == LocationLocal {
		return "AGENTS.md", nil
	}
	dir, err := codexHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "AGENTS.md"), nil
}

// codexSkillDirs resolves Codex's skill directories: index 0 is always the
// shared package this target writes. A second, read-only entry names the
// Codex-specific root that Codex reads at that scope: ".codex/skills/codegraph"
// locally, and skills/codegraph under codexHomeDir globally. codegraph never
// writes to either read-only root, because Codex does not merge same-name
// skills, so a second physical copy would list twice.
func codexSkillDirs(loc Location) ([]string, error) {
	dirs, err := sharedSkillDirs(loc)
	if err != nil {
		return nil, err
	}
	if loc == LocationLocal {
		return append(dirs, filepath.Join(".codex", "skills", "codegraph")), nil
	}
	dir, err := codexHomeDir()
	if err != nil {
		// The shared skill directory does not depend on CODEX_HOME, so a bad
		// value only drops the read-only CODEX_HOME entry.
		var homeErr codexHomeError
		if errors.As(err, &homeErr) {
			return dirs, nil
		}
		return nil, err
	}
	return append(dirs, filepath.Join(dir, "skills", "codegraph")), nil
}

// codexTrustNote returns the advisory Note every local install adds: Codex
// loads a project's MCP server only once the project is trusted, via its own
// TUI trust prompt or the "trust_level" key it writes into the user's global
// config.toml (codexConfigPath, shown as ~/.codex/config.toml when that is
// the default location, and as $CODEX_HOME/config.toml when CODEX_HOME does
// not resolve). codegraph never writes that entry itself, and the note never
// suggests a -c override, which does not grant trust. Neither the codegraph
// skill nor the AGENTS.md block is trust-gated, so the note also says those
// two are read regardless of trust, so a user does not conclude the whole
// install needs trusting. Returns "" at global scope, where no trust concept
// applies.
func codexTrustNote(loc Location) (string, error) {
	if loc != LocationLocal {
		return "", nil
	}
	root, err := filepath.Abs(".")
	if err != nil {
		return "", err
	}
	var configRef string
	configPath, err := codexConfigPath(LocationGlobal)
	var homeErr codexHomeError
	switch {
	case err == nil:
		configRef = configPath
		if home, herr := userHomeDir(); herr == nil {
			// CODEX_HOME resolves symlinks, so the default is compared both as
			// written and resolved.
			def := filepath.Join(home, ".codex", "config.toml")
			if real, rerr := realpath.Resolve(def); configPath == def || (rerr == nil && configPath == real) {
				configRef = "~/.codex/config.toml"
			}
		}
	case errors.As(err, &homeErr):
		configRef = fmt.Sprintf("$CODEX_HOME/config.toml, which does not resolve because %v", err)
	default:
		return "", err
	}
	return fmt.Sprintf(
		"Codex loads this project's MCP server (%s) only once the project is trusted — accept Codex's trust prompt, or add `trust_level = \"trusted\"` under `[projects.%q]` in %s (codegraph never writes this entry itself). The codegraph skill and the AGENTS.md block are read regardless of trust.",
		root, root, configRef,
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

// readFileOrEmpty is for a read that decides nothing on disk (Detect). A
// read-modify-write uses readUserFile.
func readFileOrEmpty(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// readUserFile is the read half of a read-modify-write of a text file the
// user owns. A missing file gives "": nothing is lost when it is written.
// Any other read error is returned, so the caller writes nothing; treating
// an unreadable file as empty would replace its content.
func readUserFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("could not read existing file: %w", err)
	}
	return string(data), nil
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
		return DetectionResult{Err: err}
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
	if err := opts.Validate(); err != nil {
		result.Errors = append(result.Errors, err)
		return result
	}
	caps := t.Capabilities()

	if configPath, err := caps.MCPConfig(loc); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve codex config path: %w", err))
	} else {
		_ = recordUserAction(&result, configPath, loc, opInstall, func(real string) (FileAction, error) {
			existed := fileExists(real)
			existing, rerr := readUserFile(real)
			if rerr != nil {
				return "", rerr
			}
			if cerr := tomlTableConflict(existing, codexTOMLTable); cerr != nil {
				return "", cerr
			}
			updated := spliceTOMLTable(existing, codexTOMLTable, codexTableBody(opts.ExecPath))
			if updated == existing {
				return ActionUnchanged, nil
			}
			if err := atomicWriteFile(real, updated); err != nil {
				return "", err
			}
			if !existed {
				return ActionCreated, nil
			}
			return ActionUpdated, nil
		})
	}

	if instrPath, err := caps.InstructionsPath(loc); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve codex instructions path: %w", err))
	} else {
		installInstructions(&result, instrPath, loc)

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

	// 07-08: the sticky Keep/On/Off lifecycle (D-18, D-19, D-23).
	switch opts.PreToolNudge {
	case PreToolNudgeOn:
		disabled, source, globalNote, err := codexHooksExplicitlyDisabled(loc)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("resolve codex hooks-disabled setting: %w", err))
		} else if disabled {
			result.Notes = append(result.Notes, codexHooksDisabledNote(source))
		} else {
			if globalNote != "" {
				result.Notes = append(result.Notes, globalNote)
			}
			beforeErrs, beforeFiles := len(result.Errors), len(result.Files)
			installCodexPreToolNudge(&result, loc, opts.ExecPath)
			if len(result.Errors) == beforeErrs {
				result.Notes = append(result.Notes, codexHookTrustNote(codexForeignHookMoved(result.Files[beforeFiles:])))
			}
		}
	case PreToolNudgeKeep:
		if hooksPath, err := codexHooksJSONPath(loc); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("resolve codex hooks.json path: %w", err))
		} else if _, ownCommands, berr := codexPreToolUseBlocks(loc); berr != nil {
			result.Errors = append(result.Errors, fmt.Errorf("%s: %w", hooksPath, berr))
		} else if recorded, herr := hasOwnHookBlock(hooksPath, "PreToolUse", ownCommands); herr != nil {
			// An unreadable or malformed hooks.json: Keep touches nothing,
			// "cannot tell, so don't guess" (matches Claude's
			// preToolNudgeEvidenced posture for the same ambiguity), and
			// says so.
			result.Notes = append(result.Notes, fmt.Sprintf("the codegraph PreToolUse nudge was not refreshed: %v", herr))
		} else if recorded {
			disabled, source, globalNote, derr := codexHooksExplicitlyDisabled(loc)
			if derr != nil {
				result.Errors = append(result.Errors, fmt.Errorf("resolve codex hooks-disabled setting: %w", derr))
			} else if disabled {
				result.Notes = append(result.Notes, codexHooksDisabledNote(source))
			} else {
				if globalNote != "" {
					result.Notes = append(result.Notes, globalNote)
				}
				beforeFiles := len(result.Files)
				installCodexPreToolNudge(&result, loc, opts.ExecPath)
				if codexHooksFileWasWritten(result.Files[beforeFiles:], hooksPath) {
					result.Notes = append(result.Notes, codexHookTrustNote(codexForeignHookMoved(result.Files[beforeFiles:])))
				}
			}
		}
		// !recorded: Keep touches nothing, as for a user who never opted in.
	case PreToolNudgeOff:
		uninstallCodexPreToolNudge(&result, loc, opInstall)
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
	} else {
		// The existence test runs on the real location, after the resolve,
		// so a symlink loop is an error and not "no file".
		_ = recordUserAction(&result, configPath, loc, opUninstall, func(real string) (FileAction, error) {
			if !fileExists(real) {
				return ActionNotFound, nil
			}
			existing, rerr := readUserFile(real)
			if rerr != nil {
				return "", rerr
			}
			if cerr := tomlTableConflict(existing, codexTOMLTable); cerr != nil {
				return "", cerr
			}
			updated := stripTOMLTable(existing, codexTOMLTable)
			switch updated {
			case existing:
				return ActionNotFound, nil
			case "":
				if err := os.Remove(real); err != nil {
					return "", err
				}
			default:
				if err := atomicWriteFile(real, updated); err != nil {
					return "", err
				}
			}
			return ActionRemoved, nil
		})
	}

	if instrPath, err := caps.InstructionsPath(loc); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve codex instructions path: %w", err))
	} else {
		removeSharedInstructions(&result, instrPath, loc, t.ID())
	}

	uninstallDeclaredSkill(&result, t, loc)

	// D-09/D-11 discipline: uninstall always attempts removal, whether or
	// not the opt-in was ever recorded — reports not-found when it wasn't.
	uninstallCodexPreToolNudge(&result, loc, opUninstall)

	return result
}

// DescribePaths is a derivation of the capability table (D-02, D-03).
func (t codexTarget) DescribePaths(loc Location) []string {
	return describeDeclaredPaths(t, loc)
}
