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
