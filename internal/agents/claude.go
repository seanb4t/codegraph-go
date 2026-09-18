package agents

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	claudeassets "github.com/seanb4t/codegraph-go"
)

// claudeAllowToken is the permission entry Claude's settings.json needs so
// AutoAllow-installed users aren't prompted per-tool for every
// mcp__codegraph__* call (D-05).
const claudeAllowToken = "mcp__codegraph__*"

// claudeTarget implements AgentTarget for Claude Code (D-02, D-05, D-07,
// D-08). Global scope writes ~/.claude.json; local scope writes
// ./.mcp.json — NEVER ./.claude.json, which Claude Code silently never
// reads (Pitfall 3, TS issue #207). Both scopes upsert a marker-fenced
// CLAUDE.md instructions block and, when InstallOptions.AutoAllow is set,
// append "mcp__codegraph__*" to settings.json's permissions.allow list.
type claudeTarget struct{}

func init() {
	registerTarget(claudeTarget{})
}

func (claudeTarget) ID() TargetID        { return Claude }
func (claudeTarget) DisplayName() string { return "Claude Code" }

// SupportsLocation is a derivation of the capability table (D-02, D-03).
func (t claudeTarget) SupportsLocation(loc Location) bool {
	return t.Capabilities().Supports(loc)
}

// Capabilities is Claude's capability table entry (D-01, D-02): both
// scopes, JSON config, a claude-json hooks mechanism (its files are
// hardcoded in Capabilities.HookFiles), and the one target whose
// SkillDirs is populated — the shared writer's skill package.
func (claudeTarget) Capabilities() Capabilities {
	return Capabilities{
		Scopes:       []Location{LocationGlobal, LocationLocal},
		ConfigFormat: ConfigFormatJSON,
		Hooks:        HooksClaudeJSON,
		MCPConfig:    claudeConfigPath,
		Instructions: claudeInstructionsPath,
		SkillDirs: func(loc Location) ([]string, error) {
			dir, err := claudeSkillDirPath(loc)
			if err != nil {
				return nil, err
			}
			return []string{dir}, nil
		},
	}
}

// fileExists reports whether path exists (any file type), swallowing stat
// errors that indicate genuine absence — every per-agent Detect
// implementation uses this to check for an agent's own config/dir (D-03).
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// stdioMcpEntry builds the {type:"stdio", command, args} entry shape
// shared by Claude, Cursor, Gemini, and Kiro (Pattern 2). Antigravity does
// NOT use this — it has its own no-type entry builder (Pitfall 6).
func stdioMcpEntry(execPath string, args ...string) map[string]any {
	return map[string]any{
		"type":    "stdio",
		"command": execPath,
		"args":    args,
	}
}

// mcpEntryPresent reports whether path's JSON already has an
// mcpServers.codegraph entry, used by Detect's AlreadyConfigured field.
func mcpEntryPresent(path string) bool {
	existing, err := readJSONFile(path)
	if err != nil {
		return false
	}
	mcpServers, _ := existing["mcpServers"].(map[string]any)
	if mcpServers == nil {
		return false
	}
	_, ok := mcpServers["codegraph"]
	return ok
}

// instructionsBody returns codegraphInstructionsBlock's content with the
// surrounding markers and their adjoining newlines stripped, for callers
// that pass content to upsertInstructionsEntry (which re-adds the
// startMarker + "\n" + content + "\n" + endMarker wrapping itself) — the
// two compose back to codegraphInstructionsBlock byte-for-byte (D-01a).
func instructionsBody() string {
	s := strings.TrimPrefix(codegraphInstructionsBlock, codegraphSectionStart+"\n")
	s = strings.TrimSuffix(s, "\n"+codegraphSectionEnd)
	return s
}

// claudeConfigPath returns the MCP config file for loc: ~/.claude.json for
// global, ./.mcp.json for local (Pitfall 3 — never ./.claude.json).
func claudeConfigPath(loc Location) (string, error) {
	if loc == LocationLocal {
		return ".mcp.json", nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude.json"), nil
}

// claudeLegacyLocalConfigPath is the pre-#207 incorrect local-scope file a
// previous install may have written to. Install migrates any codegraph
// entry found here into claudeConfigPath(local); Uninstall strips it from
// both locations (Pitfall 3).
func claudeLegacyLocalConfigPath() string {
	return ".claude.json"
}

func claudeInstructionsPath(loc Location) (string, error) {
	if loc == LocationLocal {
		return filepath.Join(".claude", "CLAUDE.md"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "CLAUDE.md"), nil
}

func claudeSettingsPath(loc Location) (string, error) {
	if loc == LocationLocal {
		return filepath.Join(".claude", "settings.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "settings.json"), nil
}

// claudeFragmentCommand is the literal project-relative command string
// Phase 6's embedded hooks fragment (.claude/hooks/hooks.json) uses for
// every SessionStart entry — the value claudeSessionStartBlocks rewrites
// into claudeHookCommand(loc) for the location actually being installed.
const claudeFragmentCommand = "${CLAUDE_PROJECT_DIR}/.claude/hooks/session-nudge.sh"

// claudeSkillDirPath resolves to the directory Phase 7 installs the
// Claude Code skill into: .claude/skills/codegraph for local,
// <home>/.claude/skills/codegraph for global — the same global/local
// branch shape as claudeConfigPath.
func claudeSkillDirPath(loc Location) (string, error) {
	if loc == LocationLocal {
		return filepath.Join(".claude", "skills", "codegraph"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "skills", "codegraph"), nil
}

// claudeSkillFilePath is claudeSkillDirPath(loc) joined with SKILL.md.
func claudeSkillFilePath(loc Location) (string, error) {
	dir, err := claudeSkillDirPath(loc)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "SKILL.md"), nil
}

// claudeManifestPath is claudeSkillDirPath(loc) joined with the sidecar
// manifest filename (D-03). The dot prefix keeps the file out of any
// future recursive skill-content scan and signals "codegraph-internal, not
// skill content" the same way .codegraph/ itself is dot-prefixed at the
// project root.
func claudeManifestPath(loc Location) (string, error) {
	dir, err := claudeSkillDirPath(loc)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ".codegraph-manifest.json"), nil
}

// claudeHooksScriptPath resolves to where Phase 7 installs the
// SessionStart nudge script: .claude/hooks/session-nudge.sh for local,
// <home>/.claude/hooks/session-nudge.sh for global.
func claudeHooksScriptPath(loc Location) (string, error) {
	if loc == LocationLocal {
		return filepath.Join(".claude", "hooks", "session-nudge.sh"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "hooks", "session-nudge.sh"), nil
}

// claudeSkillPolicy resolves Install/Uninstall's foreign-content policy for
// loc's skill directory (D-17, RESEARCH Pitfall 2): the maintainer's own
// `~/.claude/skills/codegraph -> ../../.agents/skills/codegraph` symlink
// (the `npx skills` convention) makes Claude's skill directory and the
// shared directory every other target writes into the SAME physical
// directory. sameSkillDir (filepath.EvalSymlinks-based, dangling links
// followed) is used rather than a literal path comparison because a
// symlink is exactly the case a literal string comparison cannot see
// through, and a dangling link (its target not yet created) must still
// resolve identically to its eventual target so a fresh install through
// the link lands on the shared package rather than a Claude-only one. When
// the two paths coincide, refuseUnmanifested applies — D-14's
// foreign-content rule governs the shared directory, and Claude must never
// adopt content it does not uniquely own there. When they are genuinely
// distinct directories, adoptUnmanifested preserves Claude's v0.10.0
// non-shared behaviour byte-for-byte (D-05). A comparison error (e.g. a
// symlink cycle) is returned to the caller, which records it and falls
// back to the conservative refuseUnmanifested rather than ever adopting
// content whose relationship to the shared dir could not be established.
func claudeSkillPolicy(loc Location) (unmanifestedPolicy, error) {
	claudeDir, err := claudeSkillDirPath(loc)
	if err != nil {
		return refuseUnmanifested, err
	}
	sharedDir, err := sharedSkillDirPath(loc)
	if err != nil {
		return refuseUnmanifested, err
	}
	same, err := sameSkillDir(claudeDir, sharedDir)
	if err != nil {
		return refuseUnmanifested, err
	}
	if same {
		return refuseUnmanifested, nil
	}
	return adoptUnmanifested, nil
}

// claudeHookCommand returns the command string Phase 7 writes into
// hooks.SessionStart[].hooks[].command for loc. Local scope reuses Phase
// 6's dogfooded, project-relative fragment verbatim so
// TestHookRegistrationMatchesFragmentAndScript stays green. Global scope
// uses the fully-resolved absolute path to the script this same install
// writes (claudeHooksScriptPath(LocationGlobal)) rather than a literal
// "~" — RESEARCH Assumption A3 flags tilde expansion in a shell-form hook
// command as unverified against a live session, and resolving it here
// costs nothing and removes the assumption entirely. Copying Phase 6's
// project-relative command verbatim into a global install would name a
// path that exists in no project but this one (RESEARCH Pitfall 4).
func claudeHookCommand(loc Location) (string, error) {
	if loc == LocationLocal {
		return claudeFragmentCommand, nil
	}
	return claudeHooksScriptPath(LocationGlobal)
}

// claudeSessionStartBlocks decodes the embedded hooks fragment
// (claudeassets.HooksFragment) and rewrites every command field whose
// value equals the fragment's own literal project-relative command
// (claudeFragmentCommand) into claudeHookCommand(loc). Deriving the
// blocks from the embedded fragment rather than re-authoring them in Go
// keeps Phase 6's .claude/ the canonical source (Phase 6 D-04) — no
// matcher literal is hand-typed here. Returns the rewritten blocks and
// the single-element list of owned command strings writeHookEntry uses
// for identity.
func claudeSessionStartBlocks(loc Location) ([]any, []string, error) {
	data, err := claudeassets.HooksFragment()
	if err != nil {
		return nil, nil, err
	}
	var decoded struct {
		Hooks struct {
			SessionStart []any `json:"SessionStart"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil, nil, fmt.Errorf("decode embedded hooks fragment: %w", err)
	}

	ownCommand, err := claudeHookCommand(loc)
	if err != nil {
		return nil, nil, err
	}

	blocks := make([]any, 0, len(decoded.Hooks.SessionStart))
	for _, b := range decoded.Hooks.SessionStart {
		obj, ok := b.(map[string]any)
		if !ok {
			blocks = append(blocks, b)
			continue
		}
		rewritten := make(map[string]any, len(obj))
		for k, v := range obj {
			rewritten[k] = v
		}
		if entries, ok := rewritten["hooks"].([]any); ok {
			newEntries := make([]any, 0, len(entries))
			for _, e := range entries {
				eo, ok := e.(map[string]any)
				if !ok {
					newEntries = append(newEntries, e)
					continue
				}
				newEO := make(map[string]any, len(eo))
				for k, v := range eo {
					newEO[k] = v
				}
				if cmd, ok := newEO["command"].(string); ok && cmd == claudeFragmentCommand {
					newEO["command"] = ownCommand
				}
				newEntries = append(newEntries, newEO)
			}
			rewritten["hooks"] = newEntries
		}
		blocks = append(blocks, rewritten)
	}

	return blocks, []string{ownCommand}, nil
}

// addClaudeAllowPermission appends claudeAllowToken to permissions.allow in
// path's JSON if absent, idempotently (D-05). Reads through
// readJSONFileStrict, not readJSONFile: this function writes back to
// claudeSettingsPath(loc), the same file Plan 01's hooks step
// (writeHookEntry) merges into, so it must share that step's fail-loud
// read posture (Plan 02 Task 3). Leaving this on the permissive fallback
// would mean one Install call has two contradictory postures toward one
// file — the hooks step refusing to touch an unparseable settings.json
// while this step overwrites it with only codegraph's own content.
func addClaudeAllowPermission(path string) (FileResult, error) {
	existing, existedBefore, err := readJSONFileStrict(path)
	if err != nil {
		return FileResult{}, err
	}
	permissions, _ := existing["permissions"].(map[string]any)
	if permissions == nil {
		permissions = map[string]any{}
	}
	allow, _ := permissions["allow"].([]any)
	for _, v := range allow {
		if s, ok := v.(string); ok && s == claudeAllowToken {
			return FileResult{Path: path, Action: ActionUnchanged}, nil
		}
	}
	allow = append(allow, claudeAllowToken)
	permissions["allow"] = allow
	existing["permissions"] = permissions
	if err := writeJSONFile(path, existing); err != nil {
		return FileResult{}, err
	}
	action := ActionCreated
	if existedBefore {
		action = ActionUpdated
	}
	return FileResult{Path: path, Action: action}, nil
}

// removeClaudeAllowPermission removes claudeAllowToken from
// permissions.allow in path's JSON if present, leaving every other allow
// entry and unrelated key untouched (D-05, T-06-02-01). Reads through
// readJSONFileStrict for the same reason addClaudeAllowPermission does
// (Plan 02 Task 3) — the hooks removal step on this same file
// (removeHookEntry) already uses the strict reader, and a malformed or
// unreadable settings.json must make every step touching it refuse to
// write, not just some of them.
func removeClaudeAllowPermission(path string) (FileResult, error) {
	existing, _, err := readJSONFileStrict(path)
	if err != nil {
		return FileResult{}, err
	}
	permissions, ok := existing["permissions"].(map[string]any)
	if !ok {
		return FileResult{Path: path, Action: ActionNotFound}, nil
	}
	allow, ok := permissions["allow"].([]any)
	if !ok {
		return FileResult{Path: path, Action: ActionNotFound}, nil
	}
	found := false
	newAllow := make([]any, 0, len(allow))
	for _, v := range allow {
		if s, ok := v.(string); ok && s == claudeAllowToken {
			found = true
			continue
		}
		newAllow = append(newAllow, v)
	}
	if !found {
		return FileResult{Path: path, Action: ActionNotFound}, nil
	}
	if len(newAllow) == 0 {
		delete(permissions, "allow")
	} else {
		permissions["allow"] = newAllow
	}
	if len(permissions) == 0 {
		delete(existing, "permissions")
	} else {
		existing["permissions"] = permissions
	}
	if err := writeJSONFile(path, existing); err != nil {
		return FileResult{}, err
	}
	return FileResult{Path: path, Action: ActionRemoved}, nil
}

// Detect is a derivation of the capability table (D-02, D-03): the
// installed-evidence fallback is the ancestor of a table path
// (filepath.Dir of the global instructions path, i.e. ~/.claude) rather
// than a restated literal.
func (t claudeTarget) Detect(loc Location) DetectionResult {
	caps := t.Capabilities()
	if !caps.Supports(loc) {
		return DetectionResult{}
	}
	configPath, err := caps.MCPConfig(loc)
	if err != nil {
		return DetectionResult{}
	}
	installed := fileExists(configPath)
	if !installed && loc == LocationGlobal {
		if instrPath, ierr := caps.InstructionsPath(loc); ierr == nil {
			installed = fileExists(filepath.Dir(instrPath))
		}
	}
	return DetectionResult{
		Installed:         installed,
		AlreadyConfigured: mcpEntryPresent(configPath),
		ConfigPath:        configPath,
	}
}

func (claudeTarget) Install(loc Location, opts InstallOptions) WriteResult {
	var result WriteResult

	// Pitfall 3: migrate a legacy ./.claude.json local entry into
	// ./.mcp.json before writing the correct entry.
	if loc == LocationLocal {
		legacyPath := claudeLegacyLocalConfigPath()
		if fr, err := removeMcpEntry(legacyPath); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("%s: %w", legacyPath, err))
		} else if fr.Action == ActionRemoved {
			result.Files = append(result.Files, fr)
		}
	}

	if configPath, err := claudeConfigPath(loc); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve claude config path: %w", err))
	} else {
		fr, err := writeMcpEntry(configPath, func() any {
			return stdioMcpEntry(opts.ExecPath, "serve", "--mcp")
		})
		recordFile(&result, configPath, fr, err)
	}

	if instrPath, err := claudeInstructionsPath(loc); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve claude instructions path: %w", err))
	} else {
		fr, err := upsertInstructionsEntry(instrPath, codegraphSectionStart, codegraphSectionEnd, instructionsBody())
		recordFile(&result, instrPath, fr, err)
	}

	if opts.AutoAllow {
		if settingsPath, err := claudeSettingsPath(loc); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("resolve claude settings path: %w", err))
		} else {
			fr, err := addClaudeAllowPermission(settingsPath)
			recordFile(&result, settingsPath, fr, err)
		}
	}

	// Phase 7 (D-17 as of 05-03): install the binary's own embedded Claude
	// Code skill package (SKILL.md, executable session-nudge.sh,
	// SessionStart registration) — follows --location with no
	// special-casing (D-01), funnelled through recordFile like every step
	// above (CR-01).
	//
	// The content values captured here (skillMDContent, scriptContent,
	// sessionStartBlocks) let the recordSkillManifest step below hash
	// exactly what this Install call intended to write, rather than
	// re-reading the files back from disk — re-reading would make the
	// manifest record what survived the write instead of what codegraph
	// wrote, which would make D-05's drift check permanently
	// self-satisfying. Each have* flag is set only after ITS OWN write
	// succeeds, never merely on content resolution — a manifest recording
	// a hash for an artifact whose disk write just failed would assert
	// success that never happened (code review CR-01). Errors still
	// surface via recordFile/result.Errors either way.
	var (
		skillMDContent     []byte
		haveSkillMDContent bool
		scriptContent      []byte
		haveScriptContent  bool
		sessionStartBlocks []any
		haveSessionStart   bool
	)

	// D-17: claudeSkillPolicy resolves whether Claude's own skill
	// directory and the shared directory every other target writes into
	// are the SAME physical directory (a symlink — 05-RESEARCH.md
	// Pitfall 2) BEFORE any write. When they coincide the shared writer's
	// foreign-content policy (D-14) governs; when they are distinct,
	// Claude keeps its v0.10.0 non-shared adopt behaviour (D-05).
	var claudeSkillDir string
	skillPolicy := refuseUnmanifested
	if dir, err := claudeSkillDirPath(loc); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve claude skill dir path: %w", err))
	} else {
		claudeSkillDir = dir
		if p, perr := claudeSkillPolicy(loc); perr != nil {
			result.Errors = append(result.Errors, fmt.Errorf("%s: %w", dir, perr))
		} else {
			skillPolicy = p
		}
		content, ok := writeSkillFile(&result, claudeSkillDir, skillPolicy)
		if ok {
			skillMDContent = content
			haveSkillMDContent = true
		}
	}

	if scriptPath, err := claudeHooksScriptPath(loc); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve claude hooks script path: %w", err))
	} else {
		content, rerr := claudeassets.SessionNudgeScript()
		if rerr != nil {
			result.Errors = append(result.Errors, fmt.Errorf("%s: %w", scriptPath, rerr))
		} else {
			fr, werr := writeEmbeddedFile(scriptPath, string(content), true)
			recordFile(&result, scriptPath, fr, werr)
			if werr == nil {
				scriptContent = content
				haveScriptContent = true
			}
		}
	}

	if settingsPath, err := claudeSettingsPath(loc); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve claude settings path: %w", err))
	} else {
		blocks, ownCommands, berr := claudeSessionStartBlocks(loc)
		if berr != nil {
			result.Errors = append(result.Errors, fmt.Errorf("%s: %w", settingsPath, berr))
		} else {
			fr, werr := writeHookEntry(settingsPath, "SessionStart", blocks, ownCommands)
			recordFile(&result, settingsPath, fr, werr)
			if werr == nil {
				sessionStartBlocks = blocks
				haveSessionStart = true
			}
		}
	}

	// D-17 (Plan 03, superseding Plan 03's original hand-built manifest):
	// record Claude's ownership through the manifest-owned writer (05-02)
	// rather than a hand-built skillManifest — this is what makes a
	// symlinked shared directory's manifest end up with targets containing
	// BOTH claude and whichever other agent installed there, instead of
	// Claude's own write silently clobbering theirs (D-05/D-07/D-08 all
	// apply via recordSkillManifest). Only proceeds if all three
	// artifacts' content resolved — recording a hash for content that was
	// never actually written would be worse than no manifest at all
	// (CR-01, unchanged posture). When writeSkillFile above kept a foreign
	// directory untouched, haveSkillMDContent is false and this step is
	// skipped entirely — every other Claude artifact above still wrote.
	if claudeSkillDir != "" && haveSkillMDContent && haveScriptContent && haveSessionStart {
		hooksHash, herr := hashOwnedHookBlocks(sessionStartBlocks)
		if herr != nil {
			result.Errors = append(result.Errors, fmt.Errorf("%s: %w", claudeSkillDir, herr))
		} else {
			recordSkillManifest(&result, claudeSkillDir, loc, Claude, map[string]string{
				manifestKeySkillMD:   hashContent(skillMDContent),
				manifestKeyScript:    hashContent(scriptContent),
				manifestKeyHooksFrag: hooksHash,
			})
		}
	}

	return result
}

func (claudeTarget) Uninstall(loc Location) WriteResult {
	var result WriteResult

	if configPath, err := claudeConfigPath(loc); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve claude config path: %w", err))
	} else {
		fr, err := removeMcpEntry(configPath)
		recordFile(&result, configPath, fr, err)
	}

	if loc == LocationLocal {
		legacyPath := claudeLegacyLocalConfigPath()
		fr, err := removeMcpEntry(legacyPath)
		recordFile(&result, legacyPath, fr, err)
	}

	if instrPath, err := claudeInstructionsPath(loc); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve claude instructions path: %w", err))
	} else {
		action, err := removeMarkedSection(instrPath, codegraphSectionStart, codegraphSectionEnd)
		recordAction(&result, instrPath, action, err)
	}

	if settingsPath, err := claudeSettingsPath(loc); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve claude settings path: %w", err))
	} else {
		fr, err := removeClaudeAllowPermission(settingsPath)
		recordFile(&result, settingsPath, fr, err)
	}

	// D-17 (Plan 03, superseding Plan 02/03's hand-rolled manifest+SKILL.md
	// removal): remove Claude's ownership through the manifest-owned
	// writer (05-02), using the same sameSkillDir-derived policy Install
	// uses. In a symlinked layout this is D-08's last-requester rule: only
	// when Claude is the LAST requester does the shared package actually
	// get removed (manifest, then SKILL.md, then the directory-empty
	// sweep — never a recursive delete, this plan's
	// must_haves.prohibitions); otherwise Claude's own two exclusive
	// manifest keys (script, hooks fragment) are dropped and the package
	// is left intact for whichever other target still requests it
	// (T-05-11). Claude's session-nudge script and SessionStart
	// registration live outside the skill directory and are removed by
	// the two steps below exactly as before, regardless of requester
	// count.
	if claudeDir, err := claudeSkillDirPath(loc); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve claude skill dir path: %w", err))
	} else {
		policy, perr := claudeSkillPolicy(loc)
		if perr != nil {
			result.Errors = append(result.Errors, fmt.Errorf("%s: %w", claudeDir, perr))
			policy = refuseUnmanifested
		}
		uninstallSkillPackage(&result, claudeDir, Claude, []string{manifestKeyScript, manifestKeyHooksFrag}, policy)
	}

	if scriptPath, err := claudeHooksScriptPath(loc); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve claude hooks script path: %w", err))
	} else {
		fr, rerr := removeEmbeddedFile(scriptPath)
		recordFile(&result, scriptPath, fr, rerr)
	}

	if settingsPath, err := claudeSettingsPath(loc); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve claude settings path: %w", err))
	} else {
		_, ownCommands, berr := claudeSessionStartBlocks(loc)
		if berr != nil {
			result.Errors = append(result.Errors, fmt.Errorf("%s: %w", settingsPath, berr))
		} else {
			fr, werr := removeHookEntry(settingsPath, "SessionStart", ownCommands)
			recordFile(&result, settingsPath, fr, werr)
		}
	}

	return result
}

// DescribePaths is a derivation of the capability table (D-02, D-03).
func (t claudeTarget) DescribePaths(loc Location) []string {
	return describeDeclaredPaths(t, loc)
}
