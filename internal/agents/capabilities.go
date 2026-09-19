// Package agents (this file): the per-target capability table (D-01,
// D-02) — the single source every target's SupportsLocation, DescribePaths,
// and Detect's path inputs derive from (D-03), and the data
// --print-config-style renders (D-04). The table lives ON the targets:
// AgentTarget.Capabilities() returns one struct literal per target file,
// referencing that target's existing path functions by name — never a
// second hand-written copy of any path the table holds.
package agents

import (
	"errors"
	"fmt"
	"path/filepath"
)

// HookMechanism identifies how a target registers lifecycle hooks — the
// "hooks=" field --print-config-style renders (D-02). No target declares
// HooksCodexJSON this phase (Phase 7 owns Codex's hooks literal); it exists
// now so HookFiles has a defined, loud failure mode rather than silently
// returning nothing the day a target's literal switches to it.
type HookMechanism string

const (
	HooksNone       HookMechanism = "none"
	HooksClaudeJSON HookMechanism = "claude-json"
	HooksCodexJSON  HookMechanism = "codex-json"
)

// ConfigFormat identifies the on-disk syntax of a target's MCP config file —
// the "format=" field --print-config-style renders (D-02).
type ConfigFormat string

const (
	ConfigFormatJSON  ConfigFormat = "json"
	ConfigFormatJSONC ConfigFormat = "jsonc"
	ConfigFormatTOML  ConfigFormat = "toml"
	ConfigFormatYAML  ConfigFormat = "yaml"
)

// skillFileName and skillManifestFileName are the two fixed filenames
// describeDeclaredPaths joins onto a declared, written skill directory —
// the same SKILL.md / .codegraph-manifest.json names claudeSkillFilePath
// and claudeManifestPath already produce, now derived once for every
// target's skill directory rather than restated per target.
const (
	skillFileName         = "SKILL.md"
	skillManifestFileName = ".codegraph-manifest.json"
)

// errHookFilesUndeclared is returned by (Capabilities).HookFiles when Hooks
// is HooksCodexJSON — no target declares that mechanism this phase, so
// there is no known file set to name. A future literal that switches to
// HooksCodexJSON without also declaring its files must fail loudly here
// (surfacing as a D-03 guard failure) rather than silently describing no
// hook files at all.
var errHookFilesUndeclared = errors.New("agents: hook files not declared for HooksCodexJSON")

// PathFunc resolves one target path at a given Location — the shape every
// existing per-target path function (claudeConfigPath, cursorConfigPath,
// codexConfigPath via globalOnlyPath, etc.) already has.
type PathFunc func(Location) (string, error)

// PathsFunc resolves an ORDERED list of paths at a given Location — used
// only for SkillDirs, where index 0 is the directory a target writes and
// any later entries are documented read-paths only, never written by this
// target (D-02).
type PathsFunc func(Location) ([]string, error)

// Capabilities is one target's capability table entry (D-02): which
// locations it supports, its config file syntax, its hook mechanism, and
// per-location resolvers for its MCP config, instructions file, and skill
// directories. A nil Instructions or SkillDirs means "this target declares
// none" — not an error.
type Capabilities struct {
	// Scopes lists every Location this target supports — the same set
	// SupportsLocation now derives from.
	Scopes []Location
	// ConfigFormat is this target's MCP config file syntax.
	ConfigFormat ConfigFormat
	// Hooks is this target's lifecycle-hook mechanism, HooksNone if it
	// registers none.
	Hooks HookMechanism
	// MCPConfig resolves this target's MCP config file path at a
	// supported Location. Required for every Location in Scopes.
	MCPConfig PathFunc
	// Instructions resolves this target's marker-fenced instructions file
	// path at a supported Location. Nil (or "" at a given Location) means
	// this target writes no instructions file there.
	Instructions PathFunc
	// SkillDirs resolves this target's ordered skill directories at a
	// supported Location: index 0 is the directory this target writes;
	// any later entries are documented read-paths only. Nil or empty
	// means this target declares no skill directory.
	SkillDirs PathsFunc
}

// Supports reports whether loc is one of c's declared Scopes.
func (c Capabilities) Supports(loc Location) bool {
	for _, s := range c.Scopes {
		if s == loc {
			return true
		}
	}
	return false
}

// InstructionsPath resolves c's instructions file path at loc, returning
// "" (no error) when c declares no instructions resolver.
func (c Capabilities) InstructionsPath(loc Location) (string, error) {
	if c.Instructions == nil {
		return "", nil
	}
	return c.Instructions(loc)
}

// WrittenSkillDir resolves c's FIRST declared skill directory at loc — the
// one this target actually writes — returning "" (no error) when c
// declares no skill directories.
func (c Capabilities) WrittenSkillDir(loc Location) (string, error) {
	if c.SkillDirs == nil {
		return "", nil
	}
	dirs, err := c.SkillDirs(loc)
	if err != nil {
		return "", err
	}
	if len(dirs) == 0 {
		return "", nil
	}
	return dirs[0], nil
}

// ReadOnlySkillDirs resolves every skill directory in c's SkillDirs AFTER
// the first (written) one — documented read-paths only, never written by
// this target. Returns nil (no error) when c declares zero or one skill
// directory.
func (c Capabilities) ReadOnlySkillDirs(loc Location) ([]string, error) {
	if c.SkillDirs == nil {
		return nil, nil
	}
	dirs, err := c.SkillDirs(loc)
	if err != nil {
		return nil, err
	}
	if len(dirs) <= 1 {
		return nil, nil
	}
	return dirs[1:], nil
}

// HookFiles resolves the file(s) c's declared Hooks mechanism touches at
// loc: HooksClaudeJSON names the two files Claude's SessionStart
// registration writes (claudeSettingsPath, claudeHooksScriptPath);
// HooksNone names none; HooksCodexJSON is undeclared this phase and errors
// loudly via errHookFilesUndeclared rather than silently naming nothing.
func (c Capabilities) HookFiles(loc Location) ([]string, error) {
	switch c.Hooks {
	case HooksClaudeJSON:
		settingsPath, err := claudeSettingsPath(loc)
		if err != nil {
			return nil, err
		}
		scriptPath, err := claudeHooksScriptPath(loc)
		if err != nil {
			return nil, err
		}
		return []string{settingsPath, scriptPath}, nil
	case HooksCodexJSON:
		return nil, errHookFilesUndeclared
	default:
		return nil, nil
	}
}

// globalOnlyPath adapts a no-argument, global-only path function (e.g.
// codexConfigPath, hermesConfigPath, antigravityConfigPath) into a PathFunc
// — the shape Capabilities.MCPConfig/Instructions requires. Calling the
// returned PathFunc with anything other than LocationGlobal is a caller
// error (every global-only target's Scopes excludes LocationLocal, so
// Capabilities.Supports already guards this in normal use) and returns a
// named error rather than silently resolving a path for an unsupported
// scope.
func globalOnlyPath(fn func() (string, error)) PathFunc {
	return func(loc Location) (string, error) {
		if loc != LocationGlobal {
			return "", fmt.Errorf("agents: global-only path requested for location %q", loc)
		}
		return fn()
	}
}

// declaredSkillFallback resolves the requester set an unreadable or
// pre-`targets` manifest is read as, for requester's written skill
// directory dir at loc (CR-01, 05-REVIEW.md). "Assume Claude" (D-07) is
// justified only where Claude could have written that manifest: when dir
// is the same physical directory as Claude's own skill directory — D-17's
// `npx skills` layout, where `~/.claude/skills/codegraph` is a symlink onto
// the shared `.agents/skills/codegraph`. The comparison is sameSkillDir
// against claudeSkillDirPath, the same D-17-aware check
// installSkillPackageWithFallback uses for its advisory note. Comparing
// against the shared path would always match for Cursor and opencode,
// whose declared directory is the shared path. In every other case (a
// machine without Claude's layout, or a harness-exclusive directory for
// Gemini, Kiro or Antigravity) the fallback is [requester], so a corrupted
// manifest self-heals to its real owner instead of naming a phantom Claude
// that no uninstall would ever remove (D-08).
func declaredSkillFallback(dir string, loc Location, requester TargetID) ([]TargetID, error) {
	claudeDir, err := claudeSkillDirPath(loc)
	if err != nil {
		return nil, err
	}
	same, err := sameSkillDir(dir, claudeDir)
	if err != nil {
		return nil, err
	}
	if same {
		return []TargetID{Claude}, nil
	}
	return []TargetID{requester}, nil
}

// installDeclaredSkill resolves t's declared, written skill directory (the
// D-01 derivation of Capabilities().SkillDirs via WrittenSkillDir) and, if
// one is declared for loc, installs the shared skill package there through
// installSkillPackageWithFallback (AGENT-08, CR-01): install and uninstall
// derive the skill step from the ONE table. Every target except Claude —
// which has its own symlink-aware policy via claudeSkillPolicy (D-17) —
// calls this instead of hand-rolling its own skill-directory write. A
// resolution error is recorded via result.Errors (CR-01); "" (no error)
// means t declares no skill directory at loc and this is a silent no-op.
func installDeclaredSkill(result *WriteResult, t AgentTarget, loc Location) {
	dir, err := t.Capabilities().WrittenSkillDir(loc)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve %s skill dir path: %w", t.ID(), err))
		return
	}
	if dir == "" {
		return
	}
	fallback, err := declaredSkillFallback(dir, loc, t.ID())
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("%s: %w", dir, err))
		return
	}
	installSkillPackageWithFallback(result, dir, loc, t.ID(), refuseUnmanifested, fallback)
}

// uninstallDeclaredSkill mirrors installDeclaredSkill for Uninstall — every
// skill-writing target other than Claude has no exclusive manifest keys of
// its own (its only contribution to the shared manifest's Files map is the
// SKILL.md hash every requester shares), so exclusiveKeys is always nil
// here.
func uninstallDeclaredSkill(result *WriteResult, t AgentTarget, loc Location) {
	dir, err := t.Capabilities().WrittenSkillDir(loc)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve %s skill dir path: %w", t.ID(), err))
		return
	}
	if dir == "" {
		return
	}
	fallback, err := declaredSkillFallback(dir, loc, t.ID())
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("%s: %w", dir, err))
		return
	}
	uninstallSkillPackageWithFallback(result, dir, t.ID(), nil, refuseUnmanifested, fallback)
}

// describeDeclaredPaths is the shared DescribePaths body every target's
// DescribePaths(loc Location) []string now delegates to as
// `return describeDeclaredPaths(t, loc)` (D-02, D-03): nil for an
// unsupported loc; otherwise, in order, MCPConfig, InstructionsPath (if
// non-empty), HookFiles, then the written skill directory's SKILL.md and
// manifest paths (if a skill directory is declared) — duplicates removed,
// each resolution error skipping that entry exactly like every target's
// former hand-written DescribePaths did.
func describeDeclaredPaths(t AgentTarget, loc Location) []string {
	caps := t.Capabilities()
	if !caps.Supports(loc) {
		return nil
	}

	var paths []string
	seen := make(map[string]bool)
	add := func(p string) {
		if p == "" || seen[p] {
			return
		}
		seen[p] = true
		paths = append(paths, p)
	}

	if caps.MCPConfig != nil {
		if p, err := caps.MCPConfig(loc); err == nil {
			add(p)
		}
	}
	if instr, err := caps.InstructionsPath(loc); err == nil {
		add(instr)
	}
	if hookFiles, err := caps.HookFiles(loc); err == nil {
		for _, p := range hookFiles {
			add(p)
		}
	} else if errors.Is(err, errHookFilesUndeclared) {
		// WR-01 (code review 05-REVIEW.md): errHookFilesUndeclared's and
		// HookFiles's doc comments both promise a target literal that sets
		// Hooks: HooksCodexJSON without a HookFiles case "must fail loudly
		// here ... rather than silently describing no hook files at all."
		// DescribePaths has no error return to surface this through, so a
		// programmer error — a hooks mechanism this package cannot
		// describe — panics instead of silently shipping an
		// incomplete-but-successful path list. Every other HookFiles error
		// (e.g. a path resolution failure for HooksClaudeJSON) is still
		// silently skipped here, consistent with MCPConfig/InstructionsPath
		// above.
		panic(fmt.Sprintf("agents: %s declares Hooks=%q with no HookFiles case: %v", t.ID(), caps.Hooks, err))
	}
	if skillDir, err := caps.WrittenSkillDir(loc); err == nil && skillDir != "" {
		add(filepath.Join(skillDir, skillFileName))
		add(filepath.Join(skillDir, skillManifestFileName))
	}

	return paths
}
