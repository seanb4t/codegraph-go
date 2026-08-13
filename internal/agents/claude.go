package agents

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	claudeassets "github.com/seanb4t/codegraph-go"
	"github.com/seanb4t/codegraph-go/internal/version"
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

func (claudeTarget) ID() TargetID                   { return Claude }
func (claudeTarget) DisplayName() string            { return "Claude Code" }
func (claudeTarget) SupportsLocation(Location) bool { return true }

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

func (claudeTarget) Detect(loc Location) DetectionResult {
	configPath, err := claudeConfigPath(loc)
	if err != nil {
		return DetectionResult{}
	}
	installed := fileExists(configPath)
	if !installed && loc == LocationGlobal {
		if home, herr := os.UserHomeDir(); herr == nil {
			installed = fileExists(filepath.Join(home, ".claude"))
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

	// Phase 7: install the binary's own embedded Claude Code skill
	// package (SKILL.md, executable session-nudge.sh, SessionStart
	// registration) — follows --location with no special-casing (D-01),
	// funnelled through recordFile like every step above (CR-01).
	//
	// The three content values are captured here (skillMDContent,
	// scriptContent, sessionStartBlocks) so Plan 03's manifest step below
	// can hash exactly what this Install call intended to write, rather
	// than re-reading the files back from disk — re-reading would make
	// the manifest record what survived the write instead of what
	// codegraph wrote, which would make D-05's drift check permanently
	// self-satisfying.
	var (
		skillMDContent     []byte
		haveSkillMDContent bool
		scriptContent      []byte
		haveScriptContent  bool
		sessionStartBlocks []any
		haveSessionStart   bool
	)

	if skillFilePath, err := claudeSkillFilePath(loc); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve claude skill file path: %w", err))
	} else {
		content, rerr := claudeassets.SkillMarkdown()
		if rerr != nil {
			result.Errors = append(result.Errors, fmt.Errorf("%s: %w", skillFilePath, rerr))
		} else {
			skillMDContent = content
			haveSkillMDContent = true
			fr, werr := writeEmbeddedFile(skillFilePath, string(content), false)
			recordFile(&result, skillFilePath, fr, werr)
		}
	}

	if scriptPath, err := claudeHooksScriptPath(loc); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve claude hooks script path: %w", err))
	} else {
		content, rerr := claudeassets.SessionNudgeScript()
		if rerr != nil {
			result.Errors = append(result.Errors, fmt.Errorf("%s: %w", scriptPath, rerr))
		} else {
			scriptContent = content
			haveScriptContent = true
			fr, werr := writeEmbeddedFile(scriptPath, string(content), true)
			recordFile(&result, scriptPath, fr, werr)
		}
	}

	if settingsPath, err := claudeSettingsPath(loc); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve claude settings path: %w", err))
	} else {
		blocks, ownCommands, berr := claudeSessionStartBlocks(loc)
		if berr != nil {
			result.Errors = append(result.Errors, fmt.Errorf("%s: %w", settingsPath, berr))
		} else {
			sessionStartBlocks = blocks
			haveSessionStart = true
			fr, werr := writeHookEntry(settingsPath, "SessionStart", blocks, ownCommands)
			recordFile(&result, settingsPath, fr, werr)
		}
	}

	// Plan 03 (D-03/D-04): write the sidecar manifest describing exactly
	// what the three steps above intended to write. Only proceeds if all
	// three artifacts' content resolved — a manifest recording a hash for
	// content that was never actually available would be worse than no
	// manifest at all.
	if manifestPath, err := claudeManifestPath(loc); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve claude manifest path: %w", err))
	} else if haveSkillMDContent && haveScriptContent && haveSessionStart {
		hooksHash, herr := hashOwnedHookBlocks(sessionStartBlocks)
		if herr != nil {
			result.Errors = append(result.Errors, fmt.Errorf("%s: %w", manifestPath, herr))
		} else {
			m := skillManifest{
				SchemaVersion:    manifestSchemaVersion,
				CodegraphVersion: version.Info().Version,
				Location:         string(loc),
				Files: map[string]string{
					manifestKeySkillMD:   hashContent(skillMDContent),
					manifestKeyScript:    hashContent(scriptContent),
					manifestKeyHooksFrag: hooksHash,
				},
			}
			fr, werr := writeManifest(manifestPath, m)
			recordFile(&result, manifestPath, fr, werr)
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

	// Plan 03: remove the sidecar manifest before the SKILL.md removal
	// below, so the removeSkillDirIfEmpty sweep that follows SKILL.md's
	// removal sees an already manifest-free directory. A manifest that
	// does not exist reports ActionNotFound and is not an error (D-08).
	if manifestPath, err := claudeManifestPath(loc); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve claude manifest path: %w", err))
	} else {
		fr, rerr := removeEmbeddedFile(manifestPath)
		recordFile(&result, manifestPath, fr, rerr)
	}

	// Plan 02: remove exactly the three artifacts Phase 7's Install wrote
	// — the skill file, the executable script, and codegraph's own
	// SessionStart blocks — each funnelled through recordFile like every
	// step above (CR-01). Never a recursive directory delete: the skill
	// directory is only removed once emptying codegraph's own file
	// leaves nothing else behind (this plan's must_haves.prohibitions).
	if skillFilePath, err := claudeSkillFilePath(loc); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve claude skill file path: %w", err))
	} else {
		fr, rerr := removeEmbeddedFile(skillFilePath)
		recordFile(&result, skillFilePath, fr, rerr)
		if rerr == nil {
			if skillDir, derr := claudeSkillDirPath(loc); derr == nil {
				if cerr := removeSkillDirIfEmpty(skillDir); cerr != nil {
					result.Errors = append(result.Errors, fmt.Errorf("%s: %w", skillDir, cerr))
				}
			}
		}
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

func (claudeTarget) DescribePaths(loc Location) []string {
	var paths []string
	if p, err := claudeConfigPath(loc); err == nil {
		paths = append(paths, p)
	}
	if p, err := claudeInstructionsPath(loc); err == nil {
		paths = append(paths, p)
	}
	if p, err := claudeSettingsPath(loc); err == nil {
		paths = append(paths, p)
	}
	if p, err := claudeSkillFilePath(loc); err == nil {
		paths = append(paths, p)
	}
	if p, err := claudeHooksScriptPath(loc); err == nil {
		paths = append(paths, p)
	}
	if p, err := claudeManifestPath(loc); err == nil {
		paths = append(paths, p)
	}
	return paths
}
