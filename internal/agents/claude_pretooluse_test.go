package agents

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// preToolFragmentCommandLiteral is the local-scope registered command,
// hand-typed as an independent oracle for claudePreToolFragmentCommand.
const preToolFragmentCommandLiteral = "${CLAUDE_PROJECT_DIR}/.claude/hooks/pretooluse-nudge.sh"

// readSettingsHooks decodes settingsPath and returns its "hooks" object.
func readSettingsHooks(t *testing.T, settingsPath string) map[string]any {
	t.Helper()
	var settings map[string]any
	if err := json.Unmarshal([]byte(readFile(t, settingsPath)), &settings); err != nil {
		t.Fatalf("decode %s: %v", settingsPath, err)
	}
	hooks, _ := settings["hooks"].(map[string]any)
	return hooks
}

// assertPreToolUseBlocks checks hooks.PreToolUse is exactly the four D-02
// blocks, every handler registering wantCommand with timeout 5 and no
// statusMessage (D-12).
func assertPreToolUseBlocks(t *testing.T, hooks map[string]any, wantCommand string) {
	t.Helper()
	blocks, ok := hooks["PreToolUse"].([]any)
	if !ok {
		t.Fatalf("hooks.PreToolUse missing or not an array: %#v", hooks["PreToolUse"])
	}
	wantMatchers := []string{"Bash", "Grep", "Glob", "Read"}
	if len(blocks) != len(wantMatchers) {
		t.Fatalf("hooks.PreToolUse has %d blocks, want %d: %#v", len(blocks), len(wantMatchers), blocks)
	}
	for i, b := range blocks {
		block, ok := b.(map[string]any)
		if !ok {
			t.Fatalf("block %d is not an object: %#v", i, b)
		}
		if block["matcher"] != wantMatchers[i] {
			t.Fatalf("block %d matcher = %#v, want %q", i, block["matcher"], wantMatchers[i])
		}
		handlers, _ := block["hooks"].([]any)
		wantIfs := []string{""}
		if wantMatchers[i] == "Bash" {
			wantIfs = []string{"Bash(grep *)", "Bash(rg *)", "Bash(find *)"}
		}
		if len(handlers) != len(wantIfs) {
			t.Fatalf("block %s has %d handlers, want %d: %#v", wantMatchers[i], len(handlers), len(wantIfs), handlers)
		}
		for j, h := range handlers {
			handler, ok := h.(map[string]any)
			if !ok {
				t.Fatalf("block %s handler %d is not an object: %#v", wantMatchers[i], j, h)
			}
			if handler["type"] != "command" {
				t.Errorf("block %s handler %d type = %#v, want command", wantMatchers[i], j, handler["type"])
			}
			if handler["command"] != wantCommand {
				t.Errorf("block %s handler %d command = %#v, want %q", wantMatchers[i], j, handler["command"], wantCommand)
			}
			if handler["timeout"] != float64(5) {
				t.Errorf("block %s handler %d timeout = %#v, want 5", wantMatchers[i], j, handler["timeout"])
			}
			if _, has := handler["statusMessage"]; has {
				t.Errorf("block %s handler %d carries statusMessage (D-12)", wantMatchers[i], j)
			}
			gotIf, hasIf := handler["if"]
			if wantIfs[j] == "" {
				if hasIf {
					t.Errorf("block %s handler %d has if = %#v, want none", wantMatchers[i], j, gotIf)
				}
			} else if gotIf != wantIfs[j] {
				t.Errorf("block %s handler %d if = %#v, want %q", wantMatchers[i], j, gotIf, wantIfs[j])
			}
		}
	}
}

func TestClaude_Install_PreToolNudgeOn_WritesGuardAndBlocks(t *testing.T) {
	t.Run("local", func(t *testing.T) {
		fakeHome(t)
		t.Chdir(t.TempDir())

		res := claudeTarget{}.Install(LocationLocal, InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: PreToolNudgeOn})
		if len(res.Errors) != 0 {
			t.Fatalf("Install errors: %v", res.Errors)
		}

		guard := filepath.Join(".claude", "hooks", "pretooluse-nudge.sh")
		info, err := os.Stat(guard)
		if err != nil {
			t.Fatalf("guard not written: %v", err)
		}
		if info.Mode()&0o111 == 0 {
			t.Fatalf("guard mode %v is not executable", info.Mode())
		}
		content := readFile(t, guard)
		if !strings.HasPrefix(content, "#!/bin/sh\n") {
			t.Fatalf("guard does not start with #!/bin/sh:\n%s", content)
		}
		if !strings.Contains(content, "\ncodegraph_bin='/usr/local/bin/codegraph'\n") {
			t.Fatalf("guard lacks the rendered ExecPath line:\n%s", content)
		}
		if strings.Contains(content, "@codegraph-exec-path@") {
			t.Fatalf("guard still carries the unrendered token:\n%s", content)
		}

		hooks := readSettingsHooks(t, filepath.Join(".claude", "settings.json"))
		assertPreToolUseBlocks(t, hooks, preToolFragmentCommandLiteral)
		if _, ok := hooks["SessionStart"]; !ok {
			t.Fatalf("hooks.SessionStart missing after an opt-in install")
		}
	})

	t.Run("global", func(t *testing.T) {
		home := fakeHome(t)
		t.Chdir(t.TempDir())

		res := claudeTarget{}.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: PreToolNudgeOn})
		if len(res.Errors) != 0 {
			t.Fatalf("Install errors: %v", res.Errors)
		}
		guard := filepath.Join(home, ".claude", "hooks", "pretooluse-nudge.sh")
		if _, err := os.Stat(guard); err != nil {
			t.Fatalf("global guard not written: %v", err)
		}
		hooks := readSettingsHooks(t, filepath.Join(home, ".claude", "settings.json"))
		assertPreToolUseBlocks(t, hooks, guard)
		if _, ok := hooks["SessionStart"]; !ok {
			t.Fatalf("hooks.SessionStart missing after an opt-in install")
		}
	})
}

func TestClaude_Install_DefaultWritesNoPreToolUse(t *testing.T) {
	for _, loc := range []Location{LocationLocal, LocationGlobal} {
		t.Run(string(loc), func(t *testing.T) {
			fakeHome(t)
			t.Chdir(t.TempDir())

			guard, err := claudePreToolGuardPath(loc)
			if err != nil {
				t.Fatalf("claudePreToolGuardPath(%s): %v", loc, err)
			}
			settingsPath, err := claudeSettingsPath(loc)
			if err != nil {
				t.Fatalf("claudeSettingsPath(%s): %v", loc, err)
			}

			res := claudeTarget{}.Install(loc, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
			if len(res.Errors) != 0 {
				t.Fatalf("Install errors: %v", res.Errors)
			}
			if _, err := os.Lstat(guard); !os.IsNotExist(err) {
				t.Fatalf("default install wrote %s (Lstat err %v); the nudge is opt-in (D-09)", guard, err)
			}
			hooks := readSettingsHooks(t, settingsPath)
			if _, ok := hooks["SessionStart"]; !ok {
				t.Fatalf("hooks.SessionStart missing — the absence check below would be vacuous")
			}
			if _, ok := hooks["PreToolUse"]; ok {
				t.Fatalf("default install wrote hooks.PreToolUse; the nudge is opt-in (D-09)")
			}

			// Positive control: an opt-in install into the same scope makes
			// both the guard and the key appear, so the absence checks
			// above can see presence.
			res = claudeTarget{}.Install(loc, InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: PreToolNudgeOn})
			if len(res.Errors) != 0 {
				t.Fatalf("opt-in Install errors: %v", res.Errors)
			}
			if _, err := os.Lstat(guard); err != nil {
				t.Fatalf("positive control: opt-in install did not write %s: %v", guard, err)
			}
			if _, ok := readSettingsHooks(t, settingsPath)["PreToolUse"]; !ok {
				t.Fatalf("positive control: opt-in install did not write hooks.PreToolUse")
			}
		})
	}
}
