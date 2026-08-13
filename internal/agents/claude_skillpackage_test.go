package agents

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	claudeassets "github.com/seanb4t/codegraph-go"
)

// TestClaude_Install_WritesSkillPackage_EndToEnd is Task 1's tracer proof
// (07-01-PLAN.md): one global-scope Install call reaches every layer this
// phase adds — the embedded SKILL.md, the executable session-nudge.sh,
// and the SessionStart registration in settings.json — through
// claudeTarget's existing Install/recordFile machinery. It also proves
// the embedded FS never carries Phase 6's verification/ rehearsal
// transcripts (roadmap criterion, RESEARCH Pitfall 2).
func TestClaude_Install_WritesSkillPackage_EndToEnd(t *testing.T) {
	home := fakeHome(t)
	c := claudeTarget{}

	result := c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
	if len(result.Errors) != 0 {
		t.Fatalf("Install returned errors: %v", result.Errors)
	}

	wantSkillMD, err := claudeassets.SkillMarkdown()
	if err != nil {
		t.Fatalf("claudeassets.SkillMarkdown: %v", err)
	}
	skillPath := filepath.Join(home, ".claude", "skills", "codegraph", "SKILL.md")
	gotSkillMD, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("read installed SKILL.md: %v", err)
	}
	if string(gotSkillMD) != string(wantSkillMD) {
		t.Fatalf("installed SKILL.md does not match embedded content:\ngot=%q\nwant=%q", gotSkillMD, wantSkillMD)
	}

	wantScript, err := claudeassets.SessionNudgeScript()
	if err != nil {
		t.Fatalf("claudeassets.SessionNudgeScript: %v", err)
	}
	scriptPath := filepath.Join(home, ".claude", "hooks", "session-nudge.sh")
	scriptInfo, err := os.Stat(scriptPath)
	if err != nil {
		t.Fatalf("stat installed session-nudge.sh: %v", err)
	}
	if scriptInfo.Mode()&0o111 == 0 {
		t.Fatalf("installed session-nudge.sh is not executable: mode %v", scriptInfo.Mode())
	}
	gotScript, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("read installed session-nudge.sh: %v", err)
	}
	if string(gotScript) != string(wantScript) {
		t.Fatalf("installed session-nudge.sh does not match embedded content:\ngot=%q\nwant=%q", gotScript, wantScript)
	}

	settingsPath := filepath.Join(home, ".claude", "settings.json")
	settingsData, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("read installed settings.json: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(settingsData, &decoded); err != nil {
		t.Fatalf("installed settings.json is not valid JSON: %v", err)
	}
	hooks, ok := decoded["hooks"].(map[string]any)
	if !ok {
		t.Fatalf("installed settings.json has no top-level hooks object: %#v", decoded)
	}
	sessionStart, ok := hooks["SessionStart"].([]any)
	if !ok {
		t.Fatalf("installed settings.json has no hooks.SessionStart array: %#v", hooks)
	}
	wantCommand := filepath.Join(home, ".claude", "hooks", "session-nudge.sh")
	gotMatchers := map[string]int{}
	for _, e := range sessionStart {
		entry, ok := e.(map[string]any)
		if !ok {
			t.Fatalf("SessionStart entry is not an object: %#v", e)
		}
		matcher, _ := entry["matcher"].(string)
		gotMatchers[matcher]++
		entries, ok := entry["hooks"].([]any)
		if !ok || len(entries) != 1 {
			t.Fatalf("SessionStart entry %q has unexpected hooks array: %#v", matcher, entry)
		}
		hookEntry, ok := entries[0].(map[string]any)
		if !ok {
			t.Fatalf("SessionStart entry %q hook is not an object: %#v", matcher, entries[0])
		}
		if hookEntry["type"] != "command" {
			t.Fatalf("SessionStart entry %q hook type = %v, want %q", matcher, hookEntry["type"], "command")
		}
		if hookEntry["command"] != wantCommand {
			t.Fatalf("SessionStart entry %q command = %v, want %q", matcher, hookEntry["command"], wantCommand)
		}
	}
	if gotMatchers["startup"] != 1 || gotMatchers["resume"] != 1 || len(gotMatchers) != 2 {
		t.Fatalf("expected exactly one startup and one resume block, got %v", gotMatchers)
	}

	wantFiles := map[string]bool{
		skillPath:    false,
		scriptPath:   false,
		settingsPath: false,
	}
	for _, fr := range result.Files {
		if _, ok := wantFiles[fr.Path]; ok {
			wantFiles[fr.Path] = true
		}
	}
	for path, found := range wantFiles {
		if !found {
			t.Fatalf("result.Files does not include an entry for %s: %+v", path, result.Files)
		}
	}

	var verificationPaths []string
	walkErr := fs.WalkDir(claudeassets.FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if strings.Contains(path, "verification") {
			verificationPaths = append(verificationPaths, path)
		}
		return nil
	})
	if walkErr != nil {
		t.Fatalf("walk claudeassets.FS: %v", walkErr)
	}
	if len(verificationPaths) != 0 {
		t.Fatalf("claudeassets.FS embeds paths under verification/: %v", verificationPaths)
	}
}

// TestClaude_Install_SkillPackageReRunIsUnchanged is Task 2's idempotency
// proof: installing twice at global scope reports ActionUnchanged for all
// three new artifacts on the second run, and their on-disk content is
// identical between runs — raw bytes for SKILL.md/session-nudge.sh (not
// JSON, so raw bytes are the right contract), jsonDeepEqual for
// settings.json (this repo's established JSON byte-invariance contract,
// where key ordering carries no meaning).
func TestClaude_Install_SkillPackageReRunIsUnchanged(t *testing.T) {
	home := fakeHome(t)
	c := claudeTarget{}
	opts := InstallOptions{ExecPath: "/usr/local/bin/codegraph"}

	skillPath := filepath.Join(home, ".claude", "skills", "codegraph", "SKILL.md")
	scriptPath := filepath.Join(home, ".claude", "hooks", "session-nudge.sh")
	settingsPath := filepath.Join(home, ".claude", "settings.json")

	first := c.Install(LocationGlobal, opts)
	if len(first.Errors) != 0 {
		t.Fatalf("first Install returned errors: %v", first.Errors)
	}
	skillBefore := readFile(t, skillPath)
	scriptBefore := readFile(t, scriptPath)
	var settingsBefore map[string]any
	if err := json.Unmarshal([]byte(readFile(t, settingsPath)), &settingsBefore); err != nil {
		t.Fatalf("unmarshal settings.json after first install: %v", err)
	}

	second := c.Install(LocationGlobal, opts)
	if len(second.Errors) != 0 {
		t.Fatalf("second Install returned errors: %v", second.Errors)
	}

	actionFor := func(result WriteResult, path string) (FileAction, bool) {
		for _, fr := range result.Files {
			if fr.Path == path {
				return fr.Action, true
			}
		}
		return "", false
	}
	for _, path := range []string{skillPath, scriptPath, settingsPath} {
		action, found := actionFor(second, path)
		if !found {
			t.Fatalf("second install: no FileResult for %s", path)
		}
		if action != ActionUnchanged {
			t.Fatalf("second install: %s action = %q, want %q", path, action, ActionUnchanged)
		}
	}

	skillAfter := readFile(t, skillPath)
	scriptAfter := readFile(t, scriptPath)
	if skillBefore != skillAfter {
		t.Fatalf("SKILL.md bytes changed on re-run:\nbefore=%q\nafter=%q", skillBefore, skillAfter)
	}
	if scriptBefore != scriptAfter {
		t.Fatalf("session-nudge.sh bytes changed on re-run:\nbefore=%q\nafter=%q", scriptBefore, scriptAfter)
	}

	var settingsAfter map[string]any
	if err := json.Unmarshal([]byte(readFile(t, settingsPath)), &settingsAfter); err != nil {
		t.Fatalf("unmarshal settings.json after second install: %v", err)
	}
	if !jsonDeepEqual(settingsBefore, settingsAfter) {
		t.Fatalf("settings.json not jsonDeepEqual across re-run:\nbefore=%#v\nafter=%#v", settingsBefore, settingsAfter)
	}
}

// TestClaude_Install_PreservesUnrelatedHooksContent seeds settings.json
// with an unrelated SessionStart block using matcher "startup" — the
// same matcher codegraph uses — plus an unrelated PreToolUse event and an
// unrelated top-level key. A merge implementation keyed on matcher value
// rather than command string would destroy or corrupt the unrelated
// startup block; this is precisely the AGENT-02 failure this phase
// exists to prevent (RESEARCH Pitfall 1).
func TestClaude_Install_PreservesUnrelatedHooksContent(t *testing.T) {
	home := fakeHome(t)
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	writeFile(t, settingsPath, `{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "startup",
        "hooks": [
          {"type": "command", "command": "/some/other/unrelated-script.sh"}
        ]
      }
    ],
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {"type": "command", "command": "/some/pretooluse-guard.sh"}
        ]
      }
    ]
  },
  "someUnrelatedTopLevelKey": true
}
`)

	c := claudeTarget{}
	result := c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
	if len(result.Errors) != 0 {
		t.Fatalf("Install returned errors: %v", result.Errors)
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(readFile(t, settingsPath)), &decoded); err != nil {
		t.Fatalf("unmarshal settings.json: %v", err)
	}

	if v, ok := decoded["someUnrelatedTopLevelKey"]; !ok || v != true {
		t.Fatalf("unrelated top-level key lost: %#v", decoded)
	}

	hooks, ok := decoded["hooks"].(map[string]any)
	if !ok {
		t.Fatalf("hooks object missing: %#v", decoded)
	}
	preToolUse, ok := hooks["PreToolUse"].([]any)
	if !ok || len(preToolUse) != 1 {
		t.Fatalf("unrelated PreToolUse event lost/changed: %#v", hooks["PreToolUse"])
	}

	sessionStart, ok := hooks["SessionStart"].([]any)
	if !ok {
		t.Fatalf("SessionStart missing: %#v", hooks)
	}
	var sawUnrelatedStartup, sawCodegraphStartup, sawCodegraphResume bool
	for _, e := range sessionStart {
		entry, ok := e.(map[string]any)
		if !ok {
			t.Fatalf("SessionStart entry not an object: %#v", e)
		}
		matcher, _ := entry["matcher"].(string)
		entries, _ := entry["hooks"].([]any)
		if len(entries) != 1 {
			t.Fatalf("SessionStart entry %q has unexpected hooks count: %#v", matcher, entry)
		}
		hookObj, _ := entries[0].(map[string]any)
		cmd, _ := hookObj["command"].(string)
		switch {
		case matcher == "startup" && cmd == "/some/other/unrelated-script.sh":
			sawUnrelatedStartup = true
		case matcher == "startup" && strings.HasSuffix(cmd, "session-nudge.sh"):
			sawCodegraphStartup = true
		case matcher == "resume" && strings.HasSuffix(cmd, "session-nudge.sh"):
			sawCodegraphResume = true
		default:
			t.Fatalf("unexpected SessionStart entry: matcher=%q command=%q", matcher, cmd)
		}
	}
	if !sawUnrelatedStartup {
		t.Fatalf("unrelated startup block was removed/corrupted: %#v", sessionStart)
	}
	if !sawCodegraphStartup || !sawCodegraphResume {
		t.Fatalf("codegraph's own startup/resume blocks were not appended: %#v", sessionStart)
	}
	if len(sessionStart) != 3 {
		t.Fatalf("expected 3 SessionStart entries (1 unrelated + 2 codegraph), got %d: %#v", len(sessionStart), sessionStart)
	}
}

// TestClaude_Install_HooksBoundaryStates covers the three boundary shapes
// writeHookEntry must satisfy: an absent settings.json, one holding
// exactly codegraph's own current blocks (built by installing once and
// re-reading what was written, so the fixture cannot drift from the
// embedded fragment), and one holding many unrelated entries.
func TestClaude_Install_HooksBoundaryStates(t *testing.T) {
	t.Run("absent settings.json yields only codegraph's blocks", func(t *testing.T) {
		home := fakeHome(t)
		c := claudeTarget{}
		result := c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
		if len(result.Errors) != 0 {
			t.Fatalf("Install returned errors: %v", result.Errors)
		}
		settingsPath := filepath.Join(home, ".claude", "settings.json")
		var decoded map[string]any
		if err := json.Unmarshal([]byte(readFile(t, settingsPath)), &decoded); err != nil {
			t.Fatalf("unmarshal settings.json: %v", err)
		}
		hooks, ok := decoded["hooks"].(map[string]any)
		if !ok {
			t.Fatalf("hooks missing: %#v", decoded)
		}
		sessionStart, ok := hooks["SessionStart"].([]any)
		if !ok || len(sessionStart) != 2 {
			t.Fatalf("expected exactly 2 SessionStart entries (codegraph's own), got %#v", hooks["SessionStart"])
		}
		if len(hooks) != 1 {
			t.Fatalf("expected only the SessionStart event key, got %#v", hooks)
		}
	})

	t.Run("settings.json holding exactly codegraph's own blocks yields ActionUnchanged and zero writes", func(t *testing.T) {
		home := fakeHome(t)
		c := claudeTarget{}
		opts := InstallOptions{ExecPath: "/usr/local/bin/codegraph"}
		c.Install(LocationGlobal, opts) // seed: exactly what codegraph itself wrote

		settingsPath := filepath.Join(home, ".claude", "settings.json")
		before := readFile(t, settingsPath)

		second := c.Install(LocationGlobal, opts)
		if len(second.Errors) != 0 {
			t.Fatalf("second Install returned errors: %v", second.Errors)
		}
		var action FileAction
		found := false
		for _, fr := range second.Files {
			if fr.Path == settingsPath {
				action = fr.Action
				found = true
			}
		}
		if !found {
			t.Fatalf("no FileResult for %s", settingsPath)
		}
		if action != ActionUnchanged {
			t.Fatalf("action = %q, want %q", action, ActionUnchanged)
		}
		after := readFile(t, settingsPath)
		if before != after {
			t.Fatalf("settings.json bytes changed despite ActionUnchanged:\nbefore=%q\nafter=%q", before, after)
		}
	})

	t.Run("settings.json holding many unrelated entries preserves all of them", func(t *testing.T) {
		home := fakeHome(t)
		settingsPath := filepath.Join(home, ".claude", "settings.json")
		writeFile(t, settingsPath, `{
  "hooks": {
    "SessionStart": [
      {"matcher": "startup", "hooks": [{"type": "command", "command": "/unrelated/a.sh"}]},
      {"matcher": "clear", "hooks": [{"type": "command", "command": "/unrelated/b.sh"}]}
    ],
    "PostToolUse": [
      {"matcher": "Edit", "hooks": [{"type": "command", "command": "/unrelated/c.sh"}]}
    ]
  },
  "permissions": {"allow": ["Bash(git *)"]},
  "anotherUnrelatedKey": "value"
}
`)
		c := claudeTarget{}
		result := c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
		if len(result.Errors) != 0 {
			t.Fatalf("Install returned errors: %v", result.Errors)
		}
		var decoded map[string]any
		if err := json.Unmarshal([]byte(readFile(t, settingsPath)), &decoded); err != nil {
			t.Fatalf("unmarshal settings.json: %v", err)
		}
		if decoded["anotherUnrelatedKey"] != "value" {
			t.Fatalf("unrelated top-level key lost: %#v", decoded)
		}
		permissions, ok := decoded["permissions"].(map[string]any)
		if !ok {
			t.Fatalf("permissions lost: %#v", decoded)
		}
		allow, ok := permissions["allow"].([]any)
		if !ok || len(allow) != 1 || allow[0] != "Bash(git *)" {
			t.Fatalf("permissions.allow changed: %#v", permissions)
		}
		hooks, ok := decoded["hooks"].(map[string]any)
		if !ok {
			t.Fatalf("hooks lost: %#v", decoded)
		}
		postToolUse, ok := hooks["PostToolUse"].([]any)
		if !ok || len(postToolUse) != 1 {
			t.Fatalf("PostToolUse changed: %#v", hooks["PostToolUse"])
		}
		sessionStart, ok := hooks["SessionStart"].([]any)
		if !ok || len(sessionStart) != 4 {
			t.Fatalf("expected 4 SessionStart entries (2 unrelated + 2 codegraph), got %#v", hooks["SessionStart"])
		}
	})
}

// TestClaude_Install_HookCommandIsLocationAware pins Pitfall 4: the
// command written for LocationGlobal is the absolute resolved script
// path (never a bare "~"), the command written for LocationLocal is the
// literal ${CLAUDE_PROJECT_DIR}-anchored path Phase 6 dogfooded, and the
// two strings are not equal.
func TestClaude_Install_HookCommandIsLocationAware(t *testing.T) {
	home := fakeHome(t)

	globalCmd, err := claudeHookCommand(LocationGlobal)
	if err != nil {
		t.Fatalf("claudeHookCommand(global): %v", err)
	}
	wantGlobal := filepath.Join(home, ".claude", "hooks", "session-nudge.sh")
	if globalCmd != wantGlobal {
		t.Fatalf("global command = %q, want %q", globalCmd, wantGlobal)
	}
	if strings.Contains(globalCmd, "~") || strings.Contains(globalCmd, "${CLAUDE_PROJECT_DIR}") {
		t.Fatalf("global command must be a resolved absolute path, got %q", globalCmd)
	}

	localCmd, err := claudeHookCommand(LocationLocal)
	if err != nil {
		t.Fatalf("claudeHookCommand(local): %v", err)
	}
	wantLocal := "${CLAUDE_PROJECT_DIR}/.claude/hooks/session-nudge.sh"
	if localCmd != wantLocal {
		t.Fatalf("local command = %q, want %q", localCmd, wantLocal)
	}

	if globalCmd == localCmd {
		t.Fatalf("global and local commands must differ, both = %q", globalCmd)
	}
}

// TestClaudeAssets_EmbedsNoVerificationTranscripts proves the embed-scope
// test is not vacuous: it asserts the walk found exactly 3 entries (not
// merely "none under verification/", which a directory pattern excluding
// only some files could satisfy by accident) and separately asserts the
// real on-disk verification/ directory is non-empty, so this test would
// go red if a directory pattern were ever substituted for the three
// explicit file patterns in claudeassets.go.
func TestClaudeAssets_EmbedsNoVerificationTranscripts(t *testing.T) {
	var walked []string
	err := fs.WalkDir(claudeassets.FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			walked = append(walked, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk claudeassets.FS: %v", err)
	}
	if len(walked) != 3 {
		t.Fatalf("expected exactly 3 embedded files, got %d: %v", len(walked), walked)
	}
	for _, p := range walked {
		if strings.Contains(p, "verification") {
			t.Fatalf("claudeassets.FS embeds a path under verification/: %s", p)
		}
	}

	// Guard-the-guard: prove the walk assertion above is not vacuously
	// green by confirming the real on-disk verification/ directory it
	// must NOT have picked up is actually non-empty.
	verificationDir := filepath.Join("..", "..", ".claude", "skills", "codegraph", "verification")
	entries, err := os.ReadDir(verificationDir)
	if err != nil {
		t.Fatalf("read real verification dir %s: %v", verificationDir, err)
	}
	if len(entries) == 0 {
		t.Fatalf("real verification dir %s is empty — this test would pass vacuously even if a directory pattern were substituted for explicit file patterns", verificationDir)
	}
}
