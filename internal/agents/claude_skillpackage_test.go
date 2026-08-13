package agents

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	claudeassets "github.com/seanb4t/codegraph-go"
	"github.com/seanb4t/codegraph-go/internal/version"
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

// TestClaude_SkillPackageRoundTripIsByteInvariant is Plan 02 Task 1's
// central proof: an install→uninstall round trip against a settings.json
// that already carries an unrelated SessionStart block sharing
// codegraph's own "startup" matcher, an unrelated PreToolUse event, and
// an unrelated top-level key restores the file to exactly its pre-install
// bytes (jsonDeepEqual), and both the SKILL.md and the session-nudge.sh
// codegraph wrote are gone.
func TestClaude_SkillPackageRoundTripIsByteInvariant(t *testing.T) {
	home := fakeHome(t)
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	pre := `{
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
`
	writeFile(t, settingsPath, pre)

	c := claudeTarget{}
	installResult := c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
	if len(installResult.Errors) != 0 {
		t.Fatalf("Install returned errors: %v", installResult.Errors)
	}
	uninstallResult := c.Uninstall(LocationGlobal)
	if len(uninstallResult.Errors) != 0 {
		t.Fatalf("Uninstall returned errors: %v", uninstallResult.Errors)
	}

	var gotObj, wantObj map[string]any
	got := readFile(t, settingsPath)
	if err := json.Unmarshal([]byte(got), &gotObj); err != nil {
		t.Fatalf("post-round-trip settings.json not valid JSON: %v", err)
	}
	if err := json.Unmarshal([]byte(pre), &wantObj); err != nil {
		t.Fatalf("bad fixture: %v", err)
	}
	if !jsonDeepEqual(gotObj, wantObj) {
		t.Fatalf("round trip not byte-invariant:\ngot=%s\nwant=%s", got, pre)
	}

	skillPath := filepath.Join(home, ".claude", "skills", "codegraph", "SKILL.md")
	if fileExists(skillPath) {
		t.Fatalf("SKILL.md still present after uninstall")
	}
	scriptPath := filepath.Join(home, ".claude", "hooks", "session-nudge.sh")
	if fileExists(scriptPath) {
		t.Fatalf("session-nudge.sh still present after uninstall")
	}
}

// TestClaude_Uninstall_RemovesOnlyOwnBlockUnderSharedMatcher: with
// codegraph's own block and an unrelated block both registered under
// matcher "startup", uninstall removes exactly codegraph's block and
// leaves the unrelated one — ownership is identified by command string,
// never by matcher value (T-07-04).
func TestClaude_Uninstall_RemovesOnlyOwnBlockUnderSharedMatcher(t *testing.T) {
	home := fakeHome(t)
	c := claudeTarget{}
	c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

	settingsPath := filepath.Join(home, ".claude", "settings.json")
	var decoded map[string]any
	if err := json.Unmarshal([]byte(readFile(t, settingsPath)), &decoded); err != nil {
		t.Fatalf("unmarshal settings.json: %v", err)
	}
	hooks := decoded["hooks"].(map[string]any)
	sessionStart := hooks["SessionStart"].([]any)
	sessionStart = append(sessionStart, map[string]any{
		"matcher": "startup",
		"hooks": []any{
			map[string]any{"type": "command", "command": "/some/other/unrelated-script.sh"},
		},
	})
	hooks["SessionStart"] = sessionStart
	decoded["hooks"] = hooks
	out, err := json.MarshalIndent(decoded, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	writeFile(t, settingsPath, string(out)+"\n")

	result := c.Uninstall(LocationGlobal)
	if len(result.Errors) != 0 {
		t.Fatalf("Uninstall returned errors: %v", result.Errors)
	}

	var after map[string]any
	if err := json.Unmarshal([]byte(readFile(t, settingsPath)), &after); err != nil {
		t.Fatalf("unmarshal post-uninstall settings.json: %v", err)
	}
	hooksAfter, ok := after["hooks"].(map[string]any)
	if !ok {
		t.Fatalf("hooks missing entirely after uninstall: %#v", after)
	}
	sessionStartAfter, ok := hooksAfter["SessionStart"].([]any)
	if !ok || len(sessionStartAfter) != 1 {
		t.Fatalf("expected exactly 1 surviving SessionStart entry, got %#v", hooksAfter["SessionStart"])
	}
	entry, ok := sessionStartAfter[0].(map[string]any)
	if !ok || entry["matcher"] != "startup" {
		t.Fatalf("surviving entry has wrong matcher: %#v", sessionStartAfter[0])
	}
	entries, ok := entry["hooks"].([]any)
	if !ok || len(entries) != 1 {
		t.Fatalf("surviving entry has unexpected hooks count: %#v", entry)
	}
	hookObj, ok := entries[0].(map[string]any)
	if !ok || hookObj["command"] != "/some/other/unrelated-script.sh" {
		t.Fatalf("wrong surviving command: %#v", entries[0])
	}
}

// TestClaude_Uninstall_KeepsEmptyParentsClean proves the keep-clean
// cascade: removing codegraph's own SessionStart blocks deletes the
// SessionStart key when it empties, then deletes the hooks key when
// that empties too, and deletes settings.json entirely when that leaves
// the whole object empty — but an unrelated event under hooks keeps the
// hooks key (and the file) alive.
func TestClaude_Uninstall_KeepsEmptyParentsClean(t *testing.T) {
	t.Run("SessionStart, hooks, and the file itself are all removed when codegraph's blocks are the only content", func(t *testing.T) {
		home := fakeHome(t)
		c := claudeTarget{}
		c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

		settingsPath := filepath.Join(home, ".claude", "settings.json")
		result := c.Uninstall(LocationGlobal)
		if len(result.Errors) != 0 {
			t.Fatalf("Uninstall returned errors: %v", result.Errors)
		}

		if fileExists(settingsPath) {
			t.Fatalf("settings.json should have been removed entirely: %s", readFile(t, settingsPath))
		}
	})

	t.Run("hooks key survives when an unrelated event remains, SessionStart key is removed", func(t *testing.T) {
		home := fakeHome(t)
		c := claudeTarget{}
		c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

		settingsPath := filepath.Join(home, ".claude", "settings.json")
		var decoded map[string]any
		if err := json.Unmarshal([]byte(readFile(t, settingsPath)), &decoded); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		hooks := decoded["hooks"].(map[string]any)
		hooks["PreToolUse"] = []any{
			map[string]any{
				"matcher": "Bash",
				"hooks":   []any{map[string]any{"type": "command", "command": "/unrelated-guard.sh"}},
			},
		}
		decoded["hooks"] = hooks
		out, err := json.MarshalIndent(decoded, "", "  ")
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		writeFile(t, settingsPath, string(out)+"\n")

		result := c.Uninstall(LocationGlobal)
		if len(result.Errors) != 0 {
			t.Fatalf("Uninstall returned errors: %v", result.Errors)
		}

		var after map[string]any
		if err := json.Unmarshal([]byte(readFile(t, settingsPath)), &after); err != nil {
			t.Fatalf("unmarshal post-uninstall: %v", err)
		}
		hooksAfter, ok := after["hooks"].(map[string]any)
		if !ok {
			t.Fatalf("hooks key removed even though PreToolUse should have kept it alive: %#v", after)
		}
		if _, ok := hooksAfter["SessionStart"]; ok {
			t.Fatalf("SessionStart key should have been removed: %#v", hooksAfter)
		}
		preToolUse, ok := hooksAfter["PreToolUse"].([]any)
		if !ok || len(preToolUse) != 1 {
			t.Fatalf("unrelated PreToolUse event lost: %#v", hooksAfter["PreToolUse"])
		}
	})
}

// TestClaude_Uninstall_PreservesUserFileInSkillDir proves uninstall never
// recursively deletes the skill directory: a user-authored file placed
// next to the installed SKILL.md survives uninstall and keeps the
// directory alive; once that user file is gone, a fresh install+uninstall
// removes the now-empty directory (must_haves.prohibitions — no recursive
// delete anywhere in this phase).
func TestClaude_Uninstall_PreservesUserFileInSkillDir(t *testing.T) {
	home := fakeHome(t)
	c := claudeTarget{}
	c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

	skillDir := filepath.Join(home, ".claude", "skills", "codegraph")
	userFile := filepath.Join(skillDir, "user-notes.md")
	writeFile(t, userFile, "these are my own notes\n")

	result := c.Uninstall(LocationGlobal)
	if len(result.Errors) != 0 {
		t.Fatalf("Uninstall returned errors: %v", result.Errors)
	}

	if !fileExists(userFile) {
		t.Fatalf("user-authored file was removed by uninstall")
	}
	if !fileExists(skillDir) {
		t.Fatalf("skill directory was removed while a user file still lives in it")
	}

	if err := os.Remove(userFile); err != nil {
		t.Fatalf("remove user file: %v", err)
	}
	c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
	result2 := c.Uninstall(LocationGlobal)
	if len(result2.Errors) != 0 {
		t.Fatalf("second Uninstall returned errors: %v", result2.Errors)
	}
	if fileExists(skillDir) {
		t.Fatalf("skill directory should have been removed once empty")
	}
}

// TestClaude_Uninstall_NeverInstalledIsNotFound: uninstall against a
// clean fake home reports ActionNotFound for every new artifact, no
// Errors, and creates no files — matching the pre-existing D-08
// invariant (types.go) for the three artifacts this phase adds.
func TestClaude_Uninstall_NeverInstalledIsNotFound(t *testing.T) {
	home := fakeHome(t)
	c := claudeTarget{}

	result := c.Uninstall(LocationGlobal)
	if len(result.Errors) != 0 {
		t.Fatalf("Uninstall on a never-installed target returned errors: %v", result.Errors)
	}

	skillPath := filepath.Join(home, ".claude", "skills", "codegraph", "SKILL.md")
	scriptPath := filepath.Join(home, ".claude", "hooks", "session-nudge.sh")
	settingsPath := filepath.Join(home, ".claude", "settings.json")

	wantNotFound := map[string]bool{
		skillPath:    false,
		scriptPath:   false,
		settingsPath: false,
	}
	for _, fr := range result.Files {
		if _, ok := wantNotFound[fr.Path]; ok {
			if fr.Action != ActionNotFound {
				t.Fatalf("%s: action = %q, want %q", fr.Path, fr.Action, ActionNotFound)
			}
			wantNotFound[fr.Path] = true
		}
	}
	for path, found := range wantNotFound {
		if !found {
			t.Fatalf("no FileResult reported for %s", path)
		}
	}

	if fileExists(skillPath) || fileExists(scriptPath) || fileExists(settingsPath) {
		t.Fatalf("uninstall against never-installed target created files")
	}
}

// TestClaude_DescribePaths_NoDuplicates: DescribePaths(LocationGlobal)
// lists every file codegraph touches (MCP config, instructions,
// settings.json, SKILL.md, hooks script, manifest) with no duplicate
// entries — settingsPath is already listed for the AutoAllow permission
// step and must not be double-counted for the hooks step. The exact
// six-path count is pinned by TestClaude_DescribePaths_IncludesManifest
// (Plan 03), which supersedes this test's original five-path assertion
// now that the manifest step (D-03) adds a sixth path.
func TestClaude_DescribePaths_NoDuplicates(t *testing.T) {
	c := claudeTarget{}
	paths := c.DescribePaths(LocationGlobal)
	seen := map[string]bool{}
	for _, p := range paths {
		if seen[p] {
			t.Fatalf("duplicate path in DescribePaths: %s (%v)", p, paths)
		}
		seen[p] = true
	}
}

// TestClaude_Install_WritesManifest is Plan 03 Task 2's central proof:
// after a global install, <home>/.claude/skills/codegraph/.codegraph-
// manifest.json exists, records version.Info().Version, records location
// as "global", and its three file hashes each equal the hash of the
// corresponding artifact as actually written to disk.
func TestClaude_Install_WritesManifest(t *testing.T) {
	home := fakeHome(t)
	c := claudeTarget{}

	result := c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
	if len(result.Errors) != 0 {
		t.Fatalf("Install returned errors: %v", result.Errors)
	}

	manifestPath := filepath.Join(home, ".claude", "skills", "codegraph", ".codegraph-manifest.json")
	m, present, err := readManifest(manifestPath)
	if err != nil {
		t.Fatalf("readManifest: %v", err)
	}
	if !present {
		t.Fatalf("manifest not present after install")
	}
	if m.CodegraphVersion != version.Info().Version {
		t.Fatalf("CodegraphVersion = %q, want %q", m.CodegraphVersion, version.Info().Version)
	}
	if m.Location != string(LocationGlobal) {
		t.Fatalf("Location = %q, want %q", m.Location, LocationGlobal)
	}

	skillPath := filepath.Join(home, ".claude", "skills", "codegraph", "SKILL.md")
	scriptPath := filepath.Join(home, ".claude", "hooks", "session-nudge.sh")
	settingsPath := filepath.Join(home, ".claude", "settings.json")

	wantSkillHash := hashContent([]byte(readFile(t, skillPath)))
	wantScriptHash := hashContent([]byte(readFile(t, scriptPath)))
	if m.Files[manifestKeySkillMD] != wantSkillHash {
		t.Fatalf("SKILL.md hash = %q, want %q", m.Files[manifestKeySkillMD], wantSkillHash)
	}
	if m.Files[manifestKeyScript] != wantScriptHash {
		t.Fatalf("script hash = %q, want %q", m.Files[manifestKeyScript], wantScriptHash)
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(readFile(t, settingsPath)), &decoded); err != nil {
		t.Fatalf("unmarshal settings.json: %v", err)
	}
	hooks, _ := decoded["hooks"].(map[string]any)
	sessionStart, _ := hooks["SessionStart"].([]any)
	wantHooksHash, err := hashOwnedHookBlocks(sessionStart)
	if err != nil {
		t.Fatalf("hashOwnedHookBlocks: %v", err)
	}
	if m.Files[manifestKeyHooksFrag] != wantHooksHash {
		t.Fatalf("hooks fragment hash = %q, want %q", m.Files[manifestKeyHooksFrag], wantHooksHash)
	}
}

// TestClaude_Install_ManifestIsIdempotent: a second install leaves the
// manifest's raw bytes identical — including installed_at.
func TestClaude_Install_ManifestIsIdempotent(t *testing.T) {
	home := fakeHome(t)
	c := claudeTarget{}
	opts := InstallOptions{ExecPath: "/usr/local/bin/codegraph"}

	first := c.Install(LocationGlobal, opts)
	if len(first.Errors) != 0 {
		t.Fatalf("first Install returned errors: %v", first.Errors)
	}
	manifestPath := filepath.Join(home, ".claude", "skills", "codegraph", ".codegraph-manifest.json")
	before := readFile(t, manifestPath)

	second := c.Install(LocationGlobal, opts)
	if len(second.Errors) != 0 {
		t.Fatalf("second Install returned errors: %v", second.Errors)
	}
	after := readFile(t, manifestPath)
	if before != after {
		t.Fatalf("manifest bytes changed on re-run:\nbefore=%q\nafter=%q", before, after)
	}

	var action FileAction
	found := false
	for _, fr := range second.Files {
		if fr.Path == manifestPath {
			action = fr.Action
			found = true
		}
	}
	if !found {
		t.Fatalf("no FileResult for manifest path on second install")
	}
	if action != ActionUnchanged {
		t.Fatalf("second install manifest action = %q, want %q", action, ActionUnchanged)
	}
}

// TestClaude_Uninstall_RemovesManifest: the manifest is removed, and the
// skill directory is then removed because removing SKILL.md and the
// manifest leaves it empty.
func TestClaude_Uninstall_RemovesManifest(t *testing.T) {
	home := fakeHome(t)
	c := claudeTarget{}
	c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

	manifestPath := filepath.Join(home, ".claude", "skills", "codegraph", ".codegraph-manifest.json")
	skillDir := filepath.Join(home, ".claude", "skills", "codegraph")
	if !fileExists(manifestPath) {
		t.Fatalf("manifest not written by install")
	}

	result := c.Uninstall(LocationGlobal)
	if len(result.Errors) != 0 {
		t.Fatalf("Uninstall returned errors: %v", result.Errors)
	}

	if _, err := os.Stat(manifestPath); !os.IsNotExist(err) {
		t.Fatalf("manifest still present after uninstall (err=%v)", err)
	}
	if _, err := os.Stat(skillDir); !os.IsNotExist(err) {
		t.Fatalf("skill dir still present after uninstall (err=%v)", err)
	}
}

// TestClaude_DescribePaths_IncludesManifest: the manifest path appears in
// DescribePaths(loc) for both locations, with no duplicate entries.
func TestClaude_DescribePaths_IncludesManifest(t *testing.T) {
	c := claudeTarget{}
	for _, loc := range []Location{LocationGlobal, LocationLocal} {
		paths := c.DescribePaths(loc)
		manifestPath, err := claudeManifestPath(loc)
		if err != nil {
			t.Fatalf("claudeManifestPath(%s): %v", loc, err)
		}
		found := 0
		for _, p := range paths {
			if p == manifestPath {
				found++
			}
		}
		if found != 1 {
			t.Fatalf("DescribePaths(%s) contains manifest path %d times, want 1: %v", loc, found, paths)
		}
		seen := map[string]bool{}
		for _, p := range paths {
			if seen[p] {
				t.Fatalf("DescribePaths(%s) has duplicate entry %s: %v", loc, p, paths)
			}
			seen[p] = true
		}
		if len(paths) != 6 {
			t.Fatalf("DescribePaths(%s) expected 6 paths, got %d: %v", loc, len(paths), paths)
		}
	}
}

// TestClaude_Install_SilentlyOverwritesHandEditedSkill: install, hand-edit
// SKILL.md, install again. The file's bytes equal the embedded SKILL.md
// again. result.Notes is empty and result.Errors is empty — no prompt, no
// warning surfaced. The manifest's SKILL.md hash still matches the
// embedded content (D-05).
func TestClaude_Install_SilentlyOverwritesHandEditedSkill(t *testing.T) {
	home := fakeHome(t)
	c := claudeTarget{}
	opts := InstallOptions{ExecPath: "/usr/local/bin/codegraph"}

	first := c.Install(LocationGlobal, opts)
	if len(first.Errors) != 0 {
		t.Fatalf("first Install returned errors: %v", first.Errors)
	}

	skillPath := filepath.Join(home, ".claude", "skills", "codegraph", "SKILL.md")
	writeFile(t, skillPath, "hand-edited content that is definitely not the embedded SKILL.md\n")

	second := c.Install(LocationGlobal, opts)
	if len(second.Errors) != 0 {
		t.Fatalf("second Install returned errors: %v", second.Errors)
	}
	if len(second.Notes) != 0 {
		t.Fatalf("second Install surfaced a warning: %v", second.Notes)
	}

	wantSkillMD, err := claudeassets.SkillMarkdown()
	if err != nil {
		t.Fatalf("claudeassets.SkillMarkdown: %v", err)
	}
	gotSkillMD := readFile(t, skillPath)
	if gotSkillMD != string(wantSkillMD) {
		t.Fatalf("hand-edited SKILL.md was not silently restored:\ngot=%q\nwant=%q", gotSkillMD, wantSkillMD)
	}

	manifestPath := filepath.Join(home, ".claude", "skills", "codegraph", ".codegraph-manifest.json")
	m, present, err := readManifest(manifestPath)
	if err != nil {
		t.Fatalf("readManifest: %v", err)
	}
	if !present {
		t.Fatalf("manifest missing after second install")
	}
	wantHash := hashContent(wantSkillMD)
	if m.Files[manifestKeySkillMD] != wantHash {
		t.Fatalf("manifest SKILL.md hash = %q, want %q", m.Files[manifestKeySkillMD], wantHash)
	}
}

// TestClaude_Install_HandEditedHookBlockDuplicatesRatherThanOverwritesUnrelated:
// install, hand-edit codegraph's own "startup" block's command in
// settings.json, install again. codegraph no longer recognizes the edited
// block as its own (command-string identity is the ONLY ownership signal —
// RESEARCH Pitfall 1) and appends a fresh, correct startup+resume pair
// alongside it rather than overwriting it: SessionStart ends with 3
// entries — the hand-edited one untouched, plus codegraph's own two. Any
// unrelated block under a DIFFERENT event key is also still untouched.
// This is the accepted, deliberate tradeoff: a matcher-and-shape recovery
// heuristic was tried and reverted after security review found it let
// codegraph silently claim — and overwrite — an unrelated hook that merely
// shared a matcher name (see TestClaude_Install_NeverClaimsOwnershipOfUnrelatedHookUnderSameMatcher).
// Duplication is untidy; silent data loss on content codegraph never wrote
// is not an acceptable trade to avoid it.
func TestClaude_Install_HandEditedHookBlockDuplicatesRatherThanOverwritesUnrelated(t *testing.T) {
	home := fakeHome(t)
	c := claudeTarget{}
	opts := InstallOptions{ExecPath: "/usr/local/bin/codegraph"}
	settingsPath := filepath.Join(home, ".claude", "settings.json")

	first := c.Install(LocationGlobal, opts)
	if len(first.Errors) != 0 {
		t.Fatalf("first Install returned errors: %v", first.Errors)
	}

	wantCommand, err := claudeHookCommand(LocationGlobal)
	if err != nil {
		t.Fatalf("claudeHookCommand: %v", err)
	}

	// Hand-edit codegraph's own "startup" block's command, and add a
	// genuinely unrelated PreToolUse block that must survive untouched.
	var decoded map[string]any
	if err := json.Unmarshal([]byte(readFile(t, settingsPath)), &decoded); err != nil {
		t.Fatalf("unmarshal settings.json: %v", err)
	}
	hooks := decoded["hooks"].(map[string]any)
	sessionStart := hooks["SessionStart"].([]any)
	editedCount := 0
	for _, e := range sessionStart {
		entry := e.(map[string]any)
		if entry["matcher"] != "startup" {
			continue
		}
		entries := entry["hooks"].([]any)
		hookObj := entries[0].(map[string]any)
		hookObj["command"] = "/hand-edited/garbage-path.sh"
		editedCount++
	}
	if editedCount != 1 {
		t.Fatalf("expected to edit exactly 1 startup block, edited %d", editedCount)
	}
	hooks["PreToolUse"] = []any{
		map[string]any{
			"matcher": "Bash",
			"hooks":   []any{map[string]any{"type": "command", "command": "/unrelated/guard.sh"}},
		},
	}
	decoded["hooks"] = hooks
	out, err := json.MarshalIndent(decoded, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	writeFile(t, settingsPath, string(out)+"\n")

	second := c.Install(LocationGlobal, opts)
	if len(second.Errors) != 0 {
		t.Fatalf("second Install returned errors: %v", second.Errors)
	}
	if len(second.Notes) != 0 {
		t.Fatalf("second Install surfaced a warning: %v", second.Notes)
	}

	var after map[string]any
	if err := json.Unmarshal([]byte(readFile(t, settingsPath)), &after); err != nil {
		t.Fatalf("unmarshal post-install settings.json: %v", err)
	}
	hooksAfter := after["hooks"].(map[string]any)

	preToolUseAfter, ok := hooksAfter["PreToolUse"].([]any)
	if !ok || len(preToolUseAfter) != 1 {
		t.Fatalf("unrelated PreToolUse block lost/changed: %#v", hooksAfter["PreToolUse"])
	}

	sessionStartAfter, ok := hooksAfter["SessionStart"].([]any)
	if !ok || len(sessionStartAfter) != 3 {
		t.Fatalf("expected exactly 3 SessionStart entries (hand-edited startup preserved + fresh startup+resume appended), got %d: %#v", len(sessionStartAfter), hooksAfter["SessionStart"])
	}
	var sawGarbage, sawStartup, sawResume bool
	for _, e := range sessionStartAfter {
		entry := e.(map[string]any)
		matcher, _ := entry["matcher"].(string)
		entries, _ := entry["hooks"].([]any)
		if len(entries) != 1 {
			t.Fatalf("SessionStart entry %q has unexpected hooks count: %#v", matcher, entry)
		}
		hookObj, _ := entries[0].(map[string]any)
		cmd, _ := hookObj["command"].(string)
		switch {
		case matcher == "startup" && cmd == "/hand-edited/garbage-path.sh":
			sawGarbage = true
		case matcher == "startup" && cmd == wantCommand:
			sawStartup = true
		case matcher == "resume" && cmd == wantCommand:
			sawResume = true
		default:
			t.Fatalf("unexpected SessionStart entry: matcher=%q command=%q", matcher, cmd)
		}
	}
	if !sawGarbage {
		t.Fatalf("hand-edited startup block was removed, not preserved as unowned")
	}
	if !sawStartup || !sawResume {
		t.Fatalf("codegraph's own startup+resume blocks were not appended: sawStartup=%v sawResume=%v", sawStartup, sawResume)
	}
}

// TestClaude_Install_NeverClaimsOwnershipOfUnrelatedHookUnderSameMatcher is
// the direct regression test for the vulnerability the reverted recovery
// path introduced: an unrelated, single-command hook a user placed under
// the SAME matcher name codegraph uses ("startup"), at a location where a
// codegraph manifest already exists (i.e. codegraph "previously
// configured" this location — the exact precondition the reverted
// recovery path gated on), must survive install byte-for-byte untouched.
// codegraph's own startup+resume blocks are added alongside it, never
// merged into or replacing it — ownership is by exact command-string
// match only, never by matcher name or block shape (RESEARCH Pitfall 1).
func TestClaude_Install_NeverClaimsOwnershipOfUnrelatedHookUnderSameMatcher(t *testing.T) {
	home := fakeHome(t)
	c := claudeTarget{}
	opts := InstallOptions{ExecPath: "/usr/local/bin/codegraph"}
	settingsPath := filepath.Join(home, ".claude", "settings.json")

	// First install establishes a manifest at this location — the
	// precondition the reverted recovery path required before it would
	// treat a same-matcher block as owned by shape alone. Then uninstall
	// codegraph's own hooks (but the manifest itself is a separate
	// artifact from the SessionStart registration — the scenario under
	// test is "manifest present, but the matcher slot now holds someone
	// else's unrelated hook", which arises whenever a user hand-edits
	// settings.json without touching the manifest file).
	first := c.Install(LocationGlobal, opts)
	if len(first.Errors) != 0 {
		t.Fatalf("first Install returned errors: %v", first.Errors)
	}
	manifestPath := filepath.Join(home, ".claude", "skills", "codegraph", ".codegraph-manifest.json")
	if _, present, err := readManifest(manifestPath); err != nil || !present {
		t.Fatalf("manifest not present after first install (present=%v err=%v) — precondition for this test not met", present, err)
	}

	// Replace codegraph's SessionStart registration entirely with a single
	// unrelated hook under the "startup" matcher — the manifest is left
	// exactly as the first install wrote it.
	unrelatedCommand := "/opt/some-other-tool/on-startup.sh"
	settingsContent := `{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "startup",
        "hooks": [
          {"type": "command", "command": "` + unrelatedCommand + `"}
        ]
      }
    ]
  }
}
`
	writeFile(t, settingsPath, settingsContent)

	second := c.Install(LocationGlobal, opts)
	if len(second.Errors) != 0 {
		t.Fatalf("second Install returned errors: %v", second.Errors)
	}

	var after map[string]any
	if err := json.Unmarshal([]byte(readFile(t, settingsPath)), &after); err != nil {
		t.Fatalf("unmarshal post-install settings.json: %v", err)
	}
	hooksAfter := after["hooks"].(map[string]any)
	sessionStartAfter, ok := hooksAfter["SessionStart"].([]any)
	if !ok {
		t.Fatalf("SessionStart missing after install: %#v", hooksAfter)
	}

	wantCommand, err := claudeHookCommand(LocationGlobal)
	if err != nil {
		t.Fatalf("claudeHookCommand: %v", err)
	}

	var sawUnrelatedUntouched, sawStartup, sawResume bool
	unrelatedStartupCount := 0
	for _, e := range sessionStartAfter {
		entry := e.(map[string]any)
		matcher, _ := entry["matcher"].(string)
		entries, _ := entry["hooks"].([]any)
		if len(entries) != 1 {
			t.Fatalf("SessionStart entry %q has unexpected hooks count: %#v", matcher, entry)
		}
		hookObj, _ := entries[0].(map[string]any)
		cmd, _ := hookObj["command"].(string)
		switch {
		case matcher == "startup" && cmd == unrelatedCommand:
			sawUnrelatedUntouched = true
			unrelatedStartupCount++
		case matcher == "startup" && cmd == wantCommand:
			sawStartup = true
		case matcher == "resume" && cmd == wantCommand:
			sawResume = true
		default:
			t.Fatalf("unexpected SessionStart entry: matcher=%q command=%q", matcher, cmd)
		}
	}
	if !sawUnrelatedUntouched {
		t.Fatalf("unrelated hook under the \"startup\" matcher was overwritten — ownership was claimed by matcher/shape instead of command identity: %#v", sessionStartAfter)
	}
	if unrelatedStartupCount != 1 {
		t.Fatalf("unrelated \"startup\" hook count = %d, want exactly 1 (must not be duplicated or merged)", unrelatedStartupCount)
	}
	if !sawStartup || !sawResume {
		t.Fatalf("codegraph's own startup+resume blocks were not added alongside the unrelated hook: sawStartup=%v sawResume=%v, entries=%#v", sawStartup, sawResume, sessionStartAfter)
	}
}

// TestClaude_Install_MissingManifestIsTreatedAsDrifted: install, delete
// the manifest, hand-edit SKILL.md, install again. SKILL.md is restored to
// the embedded content and the manifest is rewritten — a missing manifest
// never causes install to leave a file alone.
func TestClaude_Install_MissingManifestIsTreatedAsDrifted(t *testing.T) {
	home := fakeHome(t)
	c := claudeTarget{}
	opts := InstallOptions{ExecPath: "/usr/local/bin/codegraph"}

	first := c.Install(LocationGlobal, opts)
	if len(first.Errors) != 0 {
		t.Fatalf("first Install returned errors: %v", first.Errors)
	}

	manifestPath := filepath.Join(home, ".claude", "skills", "codegraph", ".codegraph-manifest.json")
	if err := os.Remove(manifestPath); err != nil {
		t.Fatalf("remove manifest: %v", err)
	}

	skillPath := filepath.Join(home, ".claude", "skills", "codegraph", "SKILL.md")
	writeFile(t, skillPath, "hand-edited after the manifest was deleted\n")

	second := c.Install(LocationGlobal, opts)
	if len(second.Errors) != 0 {
		t.Fatalf("second Install returned errors: %v", second.Errors)
	}

	wantSkillMD, err := claudeassets.SkillMarkdown()
	if err != nil {
		t.Fatalf("claudeassets.SkillMarkdown: %v", err)
	}
	gotSkillMD := readFile(t, skillPath)
	if gotSkillMD != string(wantSkillMD) {
		t.Fatalf("SKILL.md not restored after missing-manifest install:\ngot=%q\nwant=%q", gotSkillMD, wantSkillMD)
	}

	_, present, err := readManifest(manifestPath)
	if err != nil {
		t.Fatalf("readManifest after re-install: %v", err)
	}
	if !present {
		t.Fatalf("manifest was not rewritten after being deleted")
	}
}

// TestClaude_Install_DriftIsDetectableFromTheManifest: after a hand edit
// and *before* the next install, readManifest's stored SKILL.md hash
// differs from hashContent of the file's current bytes — proving the
// manifest actually carries a usable drift signal rather than a
// decorative field.
func TestClaude_Install_DriftIsDetectableFromTheManifest(t *testing.T) {
	home := fakeHome(t)
	c := claudeTarget{}
	opts := InstallOptions{ExecPath: "/usr/local/bin/codegraph"}

	if result := c.Install(LocationGlobal, opts); len(result.Errors) != 0 {
		t.Fatalf("Install returned errors: %v", result.Errors)
	}

	skillPath := filepath.Join(home, ".claude", "skills", "codegraph", "SKILL.md")
	writeFile(t, skillPath, "drifted content\n")

	manifestPath := filepath.Join(home, ".claude", "skills", "codegraph", ".codegraph-manifest.json")
	m, present, err := readManifest(manifestPath)
	if err != nil {
		t.Fatalf("readManifest: %v", err)
	}
	if !present {
		t.Fatalf("manifest missing")
	}

	currentHash := hashContent([]byte(readFile(t, skillPath)))
	if m.Files[manifestKeySkillMD] == currentHash {
		t.Fatalf("manifest's stored hash matched the drifted content's hash — drift signal is not usable")
	}
}

// TestClaude_Install_ManifestNotWrittenOnPartialWriteFailure is the
// regression test for code review CR-01: SKILL.md and session-nudge.sh
// write successfully, but the SessionStart hooks step fails because
// settings.json is pre-seeded with malformed JSON (readJSONFileStrict
// refuses to touch it). The manifest must NOT be written — recording a
// hash for the hooks fragment that was never actually written to disk
// would assert success that never happened. A prior version of this code
// gated the manifest write on content *resolution* rather than write
// *success*, so it wrote a manifest anyway; this proves that regression
// cannot recur.
func TestClaude_Install_ManifestNotWrittenOnPartialWriteFailure(t *testing.T) {
	home := fakeHome(t)
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	writeFile(t, settingsPath, malformedSettingsJSON)

	c := claudeTarget{}
	result := c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

	if len(result.Errors) == 0 {
		t.Fatalf("expected result.Errors to be non-empty (malformed settings.json), got none")
	}

	skillPath := filepath.Join(home, ".claude", "skills", "codegraph", "SKILL.md")
	if !fileExists(skillPath) {
		t.Fatalf("SKILL.md should have been written despite the hooks-step failure — precondition for this test not met")
	}

	manifestPath := filepath.Join(home, ".claude", "skills", "codegraph", ".codegraph-manifest.json")
	if fileExists(manifestPath) {
		t.Fatalf("manifest was written despite a partial write failure — it now falsely asserts the hooks fragment was written when it was not")
	}
}

// TestClaude_Install_RestoresLostExecutableBitWithoutContentChange is the
// regression test for code review CR-02: session-nudge.sh's content
// already matches the embedded version, but its executable bit has been
// stripped (simulating a chmod, backup/restore, or AV quarantine round-
// trip). A prior version of writeEmbeddedFile's idempotency short-circuit
// compared only file bytes, so it returned ActionUnchanged and never
// called chmod — the SessionStart hook would then silently fail to run,
// forever, since content never differs from the embedded version again.
func TestClaude_Install_RestoresLostExecutableBitWithoutContentChange(t *testing.T) {
	home := fakeHome(t)
	c := claudeTarget{}
	opts := InstallOptions{ExecPath: "/usr/local/bin/codegraph"}

	first := c.Install(LocationGlobal, opts)
	if len(first.Errors) != 0 {
		t.Fatalf("first Install returned errors: %v", first.Errors)
	}

	scriptPath := filepath.Join(home, ".claude", "hooks", "session-nudge.sh")
	if err := os.Chmod(scriptPath, 0o644); err != nil {
		t.Fatalf("chmod scriptPath to 0o644: %v", err)
	}

	second := c.Install(LocationGlobal, opts)
	if len(second.Errors) != 0 {
		t.Fatalf("second Install returned errors: %v", second.Errors)
	}

	info, err := os.Stat(scriptPath)
	if err != nil {
		t.Fatalf("stat scriptPath: %v", err)
	}
	if info.Mode()&0o111 == 0 {
		t.Fatalf("session-nudge.sh executable bit was not restored: mode = %v", info.Mode())
	}
}
