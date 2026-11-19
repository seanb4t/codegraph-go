package agents

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// countBlockHandlers counts the handlers across the hook blocks.
func countBlockHandlers(blocks []any) int {
	n := 0
	for _, b := range blocks {
		block, _ := b.(map[string]any)
		handlers, _ := block["hooks"].([]any)
		n += len(handlers)
	}
	return n
}

// shellWord runs `printf '%s|' <command>` in sh and returns what it printed.
// Each word the shell reads the command text as comes out followed by a
// "|", so a command that is one word yields exactly one "|", at the end.
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
// stays one shell word when the home directory holds a space and a single
// quote, and a local one when CLAUDE_PROJECT_DIR holds a space.
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

// hookSeed describes the settings.json a hook-ownership test starts from:
// which spellings of codegraph's own handlers it holds, beside a foreign
// PreToolUse block that must stay first.
type hookSeed struct {
	name                    string
	startupBare, resumeBare bool // the SessionStart block holds the bare spelling, else the quoted one
	preToolBare             bool // the first own PreToolUse block holds the bare spelling, else the quoted one
}

// TestClaudeHookCommands_UnquotedEntriesStayOwned: a handler spelled as the
// bare path is owned whether it stands alone or beside the quoted spelling.
// Install leaves only the quoted command and a foreign block at index 0;
// uninstall leaves no codegraph handler and the foreign one.
func TestClaudeHookCommands_UnquotedEntriesStayOwned(t *testing.T) {
	const foreign = "echo foreign"
	seeds := []hookSeed{
		{name: "bare", startupBare: true, resumeBare: true, preToolBare: true},
		{name: "bare and quoted", startupBare: true, resumeBare: false, preToolBare: true},
	}
	for _, loc := range []Location{LocationGlobal, LocationLocal} {
		for _, seed := range seeds {
			t.Run(string(loc)+"/"+seed.name, func(t *testing.T) {
				home := fakeHome(t)
				t.Chdir(t.TempDir())
				prefix := "${CLAUDE_PROJECT_DIR}/.claude/hooks/"
				if loc == LocationGlobal {
					prefix = filepath.Join(home, ".claude", "hooks") + "/"
				}
				bare := map[string]string{"SessionStart": prefix + "session-nudge.sh", "PreToolUse": prefix + "pretooluse-nudge.sh"}
				quotedStart, err := claudeHookCommand(loc)
				if err != nil {
					t.Fatal(err)
				}
				quotedPre, err := claudePreToolHookCommand(loc)
				if err != nil {
					t.Fatal(err)
				}
				settings, err := claudeSettingsPath(loc)
				if err != nil {
					t.Fatal(err)
				}

				writeSeed := func() {
					handler := func(cmd string) []any {
						return []any{map[string]any{"type": "command", "command": cmd}}
					}
					pick := func(isBare bool, bareCmd, quotedCmd string) string {
						if isBare {
							return bareCmd
						}
						return quotedCmd
					}
					data, err := json.Marshal(map[string]any{"hooks": map[string]any{
						"SessionStart": []any{
							map[string]any{"matcher": "startup", "hooks": handler(pick(seed.startupBare, bare["SessionStart"], quotedStart))},
							map[string]any{"matcher": "resume", "hooks": handler(pick(seed.resumeBare, bare["SessionStart"], quotedStart))},
						},
						"PreToolUse": []any{
							map[string]any{"matcher": "Bash", "hooks": handler(foreign)},
							map[string]any{"matcher": "Grep", "hooks": handler(pick(seed.preToolBare, bare["PreToolUse"], quotedPre))},
							map[string]any{"matcher": "Glob", "hooks": handler(quotedPre)},
						},
					}})
					if err != nil {
						t.Fatal(err)
					}
					writeFile(t, settings, string(data))
				}

				writeSeed()
				res := claudeTarget{}.Install(loc, InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: PreToolNudgeOn})
				assertNoErrors(t, res)

				startBlocks, _, err := claudeSessionStartBlocks(loc)
				if err != nil {
					t.Fatal(err)
				}
				start := settingsHandlerCommands(t, settings, "SessionStart")
				if len(start) != countBlockHandlers(startBlocks) {
					t.Errorf("SessionStart has %d handlers after install, want %d", len(start), countBlockHandlers(startBlocks))
				}
				for _, cmd := range start {
					if cmd != quotedStart {
						t.Errorf("SessionStart handler %q after install, want only %q", cmd, quotedStart)
					}
				}
				preBlocks, _, err := claudePreToolUseBlocks(loc)
				if err != nil {
					t.Fatal(err)
				}
				pre := settingsHandlerCommands(t, settings, "PreToolUse")
				if len(pre) != countBlockHandlers(preBlocks)+1 || pre[0] != foreign {
					t.Errorf("PreToolUse handlers after install = %q, want the foreign one first and %d codegraph ones", pre, countBlockHandlers(preBlocks))
				}
				for _, cmd := range pre[1:] {
					if cmd != quotedPre {
						t.Errorf("PreToolUse handler %q after install, want only %q", cmd, quotedPre)
					}
				}

				writeSeed()
				assertNoErrors(t, claudeTarget{}.Uninstall(loc))
				if got := settingsHandlerCommands(t, settings, "SessionStart"); len(got) != 0 {
					t.Errorf("SessionStart handlers after uninstall = %q, want none", got)
				}
				if got := settingsHandlerCommands(t, settings, "PreToolUse"); len(got) != 1 || got[0] != foreign {
					t.Errorf("PreToolUse handlers after uninstall = %q, want only %q", got, foreign)
				}
			})
		}
	}
}

// TestClaudeHookCommands_ForeignHandlerInOwnedBlockSurvives: a user handler
// that shares a hooks[] array with codegraph's bare-path handler is kept by
// install and stays after uninstall.
func TestClaudeHookCommands_ForeignHandlerInOwnedBlockSurvives(t *testing.T) {
	const sibling = "echo sibling"
	home := fakeHome(t)
	t.Chdir(t.TempDir())
	settings, err := claudeSettingsPath(LocationGlobal)
	if err != nil {
		t.Fatal(err)
	}
	quoted, err := claudeHookCommand(LocationGlobal)
	if err != nil {
		t.Fatal(err)
	}
	bare := filepath.Join(home, ".claude", "hooks", "session-nudge.sh")
	data, err := json.Marshal(map[string]any{"hooks": map[string]any{"SessionStart": []any{
		map[string]any{"matcher": "startup", "hooks": []any{
			map[string]any{"type": "command", "command": bare},
			map[string]any{"type": "command", "command": sibling},
		}},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, settings, string(data))

	count := func(cmds []string, want string) (n int) {
		for _, c := range cmds {
			if c == want {
				n++
			}
		}
		return n
	}
	for pass := 1; pass <= 2; pass++ {
		assertNoErrors(t, claudeTarget{}.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"}))
		cmds := settingsHandlerCommands(t, settings, "SessionStart")
		if count(cmds, sibling) != 1 || count(cmds, bare) != 0 || count(cmds, quoted) != 2 {
			t.Fatalf("SessionStart handlers after install pass %d = %q, want the sibling once, no bare path, and the quoted command for startup and resume", pass, cmds)
		}
	}

	assertNoErrors(t, claudeTarget{}.Uninstall(LocationGlobal))
	if cmds := settingsHandlerCommands(t, settings, "SessionStart"); len(cmds) != 1 || cmds[0] != sibling {
		t.Fatalf("SessionStart handlers after uninstall = %q, want only %q", cmds, sibling)
	}
}

// TestClaudePreToolNudge_BareHandlerIsEvidenceAndRefreshed: a settings.json
// that holds only the bare-path PreToolUse handlers, with no manifest to
// record the opt-in, still counts as an opt-in, and a plain install
// refreshes the guard and registers the quoted command.
func TestClaudePreToolNudge_BareHandlerIsEvidenceAndRefreshed(t *testing.T) {
	symlinkedClaudeLayout(t)
	sharedDir, err := sharedSkillDirPath(LocationGlobal)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(sharedDir, skillFileName), "# Someone else's skill\n")

	guard, settings, quoted := lifecyclePaths(t, LocationGlobal)
	bare, err := claudePreToolUnquotedHookCommand(LocationGlobal)
	if err != nil {
		t.Fatal(err)
	}
	if bare == quoted || !strings.HasPrefix(quoted, "'") {
		t.Fatalf("precondition: bare %q and quoted %q must differ", bare, quoted)
	}
	data, err := json.Marshal(map[string]any{"hooks": map[string]any{"PreToolUse": []any{
		map[string]any{"matcher": "Grep", "hooks": []any{map[string]any{"type": "command", "command": bare}}},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, settings, string(data))

	if recorded, readable := preToolNudgeEvidenced(LocationGlobal); !recorded || !readable {
		t.Fatalf("preToolNudgeEvidenced = (%v, %v), want (true, true) for a bare-path handler", recorded, readable)
	}

	res := claudeTarget{}.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph-v2", PreToolNudge: PreToolNudgeKeep})
	assertNoErrors(t, res)
	if got := readFile(t, guard); !strings.Contains(got, "codegraph-v2") {
		t.Fatalf("Keep did not rewrite the guard:\n%s", got)
	}
	blocks, _, err := claudePreToolUseBlocks(LocationGlobal)
	if err != nil {
		t.Fatal(err)
	}
	cmds := settingsHandlerCommands(t, settings, "PreToolUse")
	if len(cmds) != countBlockHandlers(blocks) {
		t.Fatalf("PreToolUse handlers = %q, want %d", cmds, countBlockHandlers(blocks))
	}
	for _, cmd := range cmds {
		if cmd != quoted {
			t.Errorf("PreToolUse handler %q after Keep, want only %q", cmd, quoted)
		}
	}
}
