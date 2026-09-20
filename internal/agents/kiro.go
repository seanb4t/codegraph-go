package agents

import (
	"fmt"
	"os"
	"path/filepath"
)

// kiroDisabledByDefaultNote is surfaced in WriteResult.Notes on every
// Install call — Kiro IDE ships MCP support disabled by default even with
// a valid config file present (see the per-agent install-coverage table).
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

func (kiroTarget) ID() TargetID        { return Kiro }
func (kiroTarget) DisplayName() string { return "Kiro" }

// SupportsLocation is a derivation of the capability table (D-02, D-03).
func (t kiroTarget) SupportsLocation(loc Location) bool {
	return t.Capabilities().Supports(loc)
}

// Capabilities is Kiro's capability table entry (D-01, D-02): both
// scopes, JSON config, no hooks. "AGENTS.md retained as a steering
// source" is Kiro reading files it already reads — no instructions write
// (D-06(d)). kiroSkillDirs supplies its skill directory (AGENT-11).
func (kiroTarget) Capabilities() Capabilities {
	return Capabilities{
		Scopes:       []Location{LocationGlobal, LocationLocal},
		ConfigFormat: ConfigFormatJSON,
		Hooks:        HooksNone,
		MCPConfig:    kiroConfigPath,
		SkillDirs:    kiroSkillDirs,
	}
}

// kiroSkillDirs resolves Kiro's skill directory (AGENT-11, D-06 correction
// (d); [CITED: kiro.dev/docs/steering, fetched 2026-09-18]): harness-
// specific only — `.kiro/skills/codegraph/` (local) /
// `~/.kiro/skills/codegraph/` (global) — no shared `.agents/skills/`
// alias is documented for Kiro's skill discovery, so this is the sole,
// written entry. Per the cited doc, Kiro separately reads a literal
// `AGENTS.md` at `./AGENTS.md` (local) and `~/.kiro/steering/AGENTS.md`
// (global) on its own — a codegraph instructions block another target
// (opencode, Codex, Cursor once probed) writes to either path may reach
// Kiro a second time. That is an advisory only (D-06(d)): Kiro's own
// instructions handling is unchanged by this plan; no new AGENTS.md write
// is added here.
func kiroSkillDirs(loc Location) ([]string, error) {
	if loc == LocationLocal {
		return []string{filepath.Join(".kiro", "skills", "codegraph")}, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return []string{filepath.Join(home, ".kiro", "skills", "codegraph")}, nil
}

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

// Detect is a derivation of the capability table (D-02, D-03).
func (t kiroTarget) Detect(loc Location) DetectionResult {
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
		installed = fileExists(filepath.Dir(filepath.Dir(configPath)))
	}
	return DetectionResult{
		Installed:         installed,
		AlreadyConfigured: mcpEntryPresent(configPath),
		ConfigPath:        configPath,
	}
}

func (t kiroTarget) Install(loc Location, opts InstallOptions) WriteResult {
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

	installDeclaredSkill(&result, t, loc)

	result.Notes = append(result.Notes, kiroDisabledByDefaultNote)
	return result
}

// CR-01: Uninstall no longer returns early on a config-path resolution
// error — every step records its own outcome independently (05-04
// precedent), so the skill step below still runs even if the config path
// step failed.
func (t kiroTarget) Uninstall(loc Location) WriteResult {
	var result WriteResult

	if configPath, err := kiroConfigPath(loc); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve kiro config path: %w", err))
	} else {
		fr, err := removeMcpEntry(configPath)
		recordFile(&result, configPath, fr, err)
	}

	uninstallDeclaredSkill(&result, t, loc)

	return result
}

// DescribePaths is a derivation of the capability table (D-02, D-03).
func (t kiroTarget) DescribePaths(loc Location) []string {
	return describeDeclaredPaths(t, loc)
}
