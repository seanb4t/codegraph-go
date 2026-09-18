package agents

import (
	"fmt"
	"os"
	"path/filepath"
)

// antigravityTarget implements AgentTarget for Antigravity IDE (D-06).
// Global-only (SupportsLocation(LocationLocal) reports false). Writes its
// own {command, args} entry with NO "type" field — Antigravity rejects
// entries carrying type:"stdio" (Pitfall 6) — deliberately never routed
// through the shared stdioMcpEntry builder. Writes no instructions file of
// its own; it shares ~/.gemini/GEMINI.md, written only by the Gemini
// target.
type antigravityTarget struct{}

func init() {
	registerTarget(antigravityTarget{})
}

func (antigravityTarget) ID() TargetID        { return Antigravity }
func (antigravityTarget) DisplayName() string { return "Antigravity" }

// SupportsLocation is a derivation of the capability table (D-02, D-03).
func (t antigravityTarget) SupportsLocation(loc Location) bool {
	return t.Capabilities().Supports(loc)
}

// Capabilities is Antigravity's capability table entry (D-01, D-02):
// global-only, JSON config, no hooks, antigravitySkillDirs for its skill
// directory (AGENT-07). Its instructions reach Antigravity only through
// Gemini's ~/.gemini/GEMINI.md (D-06(c)) — it declares none of its own.
func (antigravityTarget) Capabilities() Capabilities {
	return Capabilities{
		Scopes:       []Location{LocationGlobal},
		ConfigFormat: ConfigFormatJSON,
		Hooks:        HooksNone,
		MCPConfig:    globalOnlyPath(antigravityConfigPath),
		SkillDirs:    antigravitySkillDirs,
	}
}

// antigravitySkillDirs resolves Antigravity's skill directories, GLOBAL
// ONLY (AGENT-07, D-06 corrections (b)(c); [CITED: antigravity.google/
// docs/skills.md, fetched 2026-09-18]): index 0 — the `agy` CLI's
// documented global path, `~/.gemini/antigravity-cli/skills/codegraph/` —
// is the one this target WRITES; it is the surface 05-06's live `agy`
// session actually exercises. Index 1, the 2.0/IDE path
// `~/.gemini/config/skills/codegraph/`, is `[ASSUMED]`, documented only,
// and NEVER written here — this milestone's herdr-driven live-session
// method drives a CLI, not a GUI IDE, so there is no live surface to
// verify it (D-06(b)). Antigravity's own docs list `.agents/skills/` only
// at WORKSPACE scope (never global), which this global-only target never
// writes, and its instructions arrive solely through Gemini's own
// `~/.gemini/GEMINI.md` write (D-06(c)) — no new AGENTS.md here. At
// LocationLocal (unsupported — Antigravity is global-only) this returns
// nil, nil: Capabilities.Supports/WrittenSkillDir/ReadOnlySkillDirs never
// call it for local in normal use, but a caller error surfaces as "no skill
// dirs" rather than a resolved local path.
func antigravitySkillDirs(loc Location) ([]string, error) {
	if loc != LocationGlobal {
		return nil, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return []string{
		filepath.Join(home, ".gemini", "antigravity-cli", "skills", "codegraph"),
		filepath.Join(home, ".gemini", "config", "skills", "codegraph"),
	}, nil
}

// antigravityUnifiedPath is the post-migration config location a current
// Antigravity release reads/writes.
func antigravityUnifiedPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".gemini", "config", "mcp_config.json"), nil
}

// antigravityLegacyPath is the pre-migration config location an older
// Antigravity release used.
func antigravityLegacyPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".gemini", "antigravity", "mcp_config.json"), nil
}

// antigravityMigratedMarker's presence (or the unified file already
// existing) signals that this machine's Antigravity install has completed
// its unified-config migration.
func antigravityMigratedMarker() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".gemini", "config", ".migrated"), nil
}

// antigravityConfigPath resolves the config path Detect/Uninstall/the
// capability table target: the unified path once migration has happened
// (marker present or the unified file already exists) OR on a machine with
// NO Antigravity config at all (the table must print the path Install
// actually writes on a fresh machine — D-02); the legacy path ONLY for an
// unmigrated machine whose legacy file exists.
func antigravityConfigPath() (string, error) {
	unified, err := antigravityUnifiedPath()
	if err != nil {
		return "", err
	}
	legacy, err := antigravityLegacyPath()
	if err != nil {
		return "", err
	}
	marker, err := antigravityMigratedMarker()
	if err != nil {
		return "", err
	}
	if fileExists(marker) || fileExists(unified) || !fileExists(legacy) {
		return unified, nil
	}
	return legacy, nil
}

// antigravityEntry builds the entry shape WITHOUT a "type" field
// (Pitfall 6) — deliberately not routed through stdioMcpEntry.
func antigravityEntry(execPath string) map[string]any {
	return map[string]any{
		"command": execPath,
		"args":    []string{"serve", "--mcp"},
	}
}

// readMcpEntry returns mcpServers.codegraph from path's JSON, if present.
func readMcpEntry(path string) (any, bool) {
	existing, err := readJSONFile(path)
	if err != nil {
		return nil, false
	}
	mcpServers, _ := existing["mcpServers"].(map[string]any)
	if mcpServers == nil {
		return nil, false
	}
	entry, ok := mcpServers["codegraph"]
	return entry, ok
}

// Detect is a derivation of the capability table (D-02, D-03): the
// installed-evidence fallback is filepath.Dir(filepath.Dir(configPath)) —
// ~/.gemini for both the unified and the legacy config path.
func (t antigravityTarget) Detect(loc Location) DetectionResult {
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

func (t antigravityTarget) Install(loc Location, opts InstallOptions) WriteResult {
	var result WriteResult
	if loc != LocationGlobal {
		return result
	}

	unified, err := antigravityUnifiedPath()
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve antigravity unified config path: %w", err))
		return result
	}
	legacy, err := antigravityLegacyPath()
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve antigravity legacy config path: %w", err))
		return result
	}
	marker, err := antigravityMigratedMarker()
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve antigravity migrated marker path: %w", err))
		return result
	}

	// migrationOK tracks whether it's safe to (a) drop the legacy entry
	// and (b) write the ".migrated" marker. CR-02: neither may happen
	// unless the unified file is confirmed to actually hold the
	// codegraph entry — otherwise a partial-write failure here would
	// permanently record "migration done" while the entry exists in
	// NEITHER file, silently reverting the user's antigravity MCP config
	// to unconfigured with no rollback.
	migrationOK := true

	if !fileExists(marker) && !fileExists(unified) && fileExists(legacy) {
		// Pre-migration legacy config, no unified file yet — sweep the
		// stale legacy codegraph entry into the unified path before this
		// install writes the current entry.
		if legacyEntry, ok := readMcpEntry(legacy); ok {
			unifiedExisting, _ := readJSONFile(unified)
			unifiedServers, _ := unifiedExisting["mcpServers"].(map[string]any)
			if unifiedServers == nil {
				unifiedServers = map[string]any{}
			}
			unifiedServers["codegraph"] = legacyEntry
			unifiedExisting["mcpServers"] = unifiedServers
			if err := writeJSONFile(unified, unifiedExisting); err != nil {
				migrationOK = false
				result.Errors = append(result.Errors, fmt.Errorf("migrate legacy entry into %s: %w", unified, err))
			} else {
				result.Files = append(result.Files, FileResult{Path: unified, Action: ActionUpdated})
			}
		}
		// Only remove the legacy entry once the unified write above is
		// confirmed successful (CR-02) — never lose the entry to a
		// partial-write failure with no file left holding it.
		if migrationOK {
			fr, err := removeMcpEntry(legacy)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("%s: %w", legacy, err))
			} else if fr.Action == ActionRemoved {
				result.Files = append(result.Files, fr)
			}
		}
	}

	fr, err := writeMcpEntry(unified, func() any {
		return antigravityEntry(opts.ExecPath)
	})
	if err != nil {
		migrationOK = false
		result.Errors = append(result.Errors, fmt.Errorf("%s: %w", unified, err))
	} else {
		result.Files = append(result.Files, fr)
	}

	// The ".migrated" marker is only written once the unified file is
	// confirmed to hold the current entry (CR-02) — writing it
	// unconditionally would permanently record "migration done" even on
	// a write failure above.
	if migrationOK && !fileExists(marker) {
		if err := os.MkdirAll(filepath.Dir(marker), 0o755); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("%s: %w", marker, err))
		} else if err := os.WriteFile(marker, []byte("migrated\n"), 0o644); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("%s: %w", marker, err))
		} else {
			result.Files = append(result.Files, FileResult{Path: marker, Action: ActionCreated})
		}
	}

	installDeclaredSkill(&result, t, loc)

	return result
}

func (t antigravityTarget) Uninstall(loc Location) WriteResult {
	var result WriteResult
	if loc != LocationGlobal {
		return result
	}
	configPath, err := antigravityConfigPath()
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve antigravity config path: %w", err))
	} else {
		fr, err := removeMcpEntry(configPath)
		recordFile(&result, configPath, fr, err)
	}

	uninstallDeclaredSkill(&result, t, loc)

	return result
}

// DescribePaths is a derivation of the capability table (D-02, D-03).
func (t antigravityTarget) DescribePaths(loc Location) []string {
	return describeDeclaredPaths(t, loc)
}
