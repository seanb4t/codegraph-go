package agents

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// settingsHandlerCommands returns every handler command registered under
// event in the settings file at path, in file order.
func settingsHandlerCommands(t *testing.T, path, event string) []string {
	t.Helper()
	blocks, _ := readSettingsHooks(t, path)[event].([]any)
	var commands []string
	for _, b := range blocks {
		block, _ := b.(map[string]any)
		handlers, _ := block["hooks"].([]any)
		for _, h := range handlers {
			if handler, _ := h.(map[string]any); handler != nil {
				if cmd, ok := handler["command"].(string); ok {
					commands = append(commands, cmd)
				}
			}
		}
	}
	return commands
}

// shellWord runs `printf '%s' <command>` in sh and returns what it printed:
// the one word the shell reads the command's text as, or several joined
// words when the text is not quoted.
func shellWord(t *testing.T, command string, env ...string) string {
	t.Helper()
	cmd := exec.Command("sh", "-c", "printf '%s|' "+command)
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("sh -c printf %s: %v", command, err)
	}
	return string(out)
}

// TestClaudeHookCommands_ArePathsAsOneShellWord: a global hook command
// stays one word when the home directory holds a space and a single quote,
// and a local one when CLAUDE_PROJECT_DIR holds a space.
func TestClaudeHookCommands_ArePathsAsOneShellWord(t *testing.T) {
	cases := []struct {
		loc  Location
		prep func(t *testing.T) (root string, env []string)
	}{
		{LocationGlobal, func(t *testing.T) (string, []string) {
			fakeHome(t)
			home := filepath.Join(t.TempDir(), "a b'c")
			t.Setenv("HOME", home)
			t.Chdir(t.TempDir())
			return home, nil
		}},
		{LocationLocal, func(t *testing.T) (string, []string) {
			fakeHome(t)
			t.Chdir(t.TempDir())
			return "/my project", []string{"CLAUDE_PROJECT_DIR=/my project"}
		}},
	}
	for _, tc := range cases {
		t.Run(string(tc.loc), func(t *testing.T) {
			root, env := tc.prep(t)
			res := claudeTarget{}.Install(tc.loc, InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: PreToolNudgeOn})
			assertNoErrors(t, res)

			settings, err := claudeSettingsPath(tc.loc)
			if err != nil {
				t.Fatal(err)
			}
			for event, script := range map[string]string{"SessionStart": "session-nudge.sh", "PreToolUse": "pretooluse-nudge.sh"} {
				commands := settingsHandlerCommands(t, settings, event)
				if len(commands) == 0 {
					t.Fatalf("no %s handler registered in %s", event, settings)
				}
				want := filepath.Join(root, ".claude", "hooks", script) + "|"
				for _, cmd := range commands {
					if got := shellWord(t, cmd, env...); got != want {
						t.Errorf("%s command %q reads as %q, want one word %q", event, cmd, got, want)
					}
				}
			}
		})
	}
}

// TestClaudeHookCommands_UnquotedEntriesStayOwned: a handler registered as
// the bare path is replaced on install and removed on uninstall, and a
// foreign handler beside it is untouched.
func TestClaudeHookCommands_UnquotedEntriesStayOwned(t *testing.T) {
	const foreign = "echo foreign"
	for _, loc := range []Location{LocationGlobal, LocationLocal} {
		t.Run(string(loc), func(t *testing.T) {
			home := fakeHome(t)
			t.Chdir(t.TempDir())
			prefix := "${CLAUDE_PROJECT_DIR}/.claude/hooks/"
			if loc == LocationGlobal {
				prefix = filepath.Join(home, ".claude", "hooks") + "/"
			}
			bare := map[string]string{"SessionStart": prefix + "session-nudge.sh", "PreToolUse": prefix + "pretooluse-nudge.sh"}

			settings, err := claudeSettingsPath(loc)
			if err != nil {
				t.Fatal(err)
			}
			seed := func() {
				handler := func(cmd string) []any {
					return []any{map[string]any{"type": "command", "command": cmd}}
				}
				data, err := json.Marshal(map[string]any{"hooks": map[string]any{
					"SessionStart": []any{
						map[string]any{"matcher": "startup", "hooks": handler(bare["SessionStart"])},
						map[string]any{"matcher": "resume", "hooks": handler(bare["SessionStart"])},
					},
					"PreToolUse": []any{
						map[string]any{"matcher": "Bash", "hooks": handler(foreign)},
						map[string]any{"matcher": "Grep", "hooks": handler(bare["PreToolUse"])},
					},
				}})
				if err != nil {
					t.Fatal(err)
				}
				writeFile(t, settings, string(data))
			}
			countBare := func() (n int) {
				for event, cmd := range bare {
					for _, got := range settingsHandlerCommands(t, settings, event) {
						if got == cmd {
							n++
						}
					}
				}
				return n
			}

			seed()
			res := claudeTarget{}.Install(loc, InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: PreToolNudgeOn})
			assertNoErrors(t, res)
			if n := countBare(); n != 0 {
				t.Errorf("install left %d unquoted handlers beside the quoted ones", n)
			}
			if got := len(settingsHandlerCommands(t, settings, "SessionStart")); got != 2 {
				t.Errorf("SessionStart has %d handlers after install, want 2", got)
			}
			blocks, _, err := claudePreToolUseBlocks(loc)
			if err != nil {
				t.Fatal(err)
			}
			if got := len(settingsHandlerCommands(t, settings, "PreToolUse")); got != len(blocks)+1 {
				t.Errorf("PreToolUse has %d handlers after install, want %d codegraph plus the foreign one", got, len(blocks))
			}

			seed()
			res = claudeTarget{}.Uninstall(loc)
			assertNoErrors(t, res)
			if n := countBare(); n != 0 {
				t.Errorf("uninstall left %d unquoted handlers", n)
			}
			if got := settingsHandlerCommands(t, settings, "PreToolUse"); len(got) != 1 || got[0] != foreign {
				t.Errorf("PreToolUse handlers after uninstall = %q, want only %q", got, foreign)
			}
		})
	}
}
