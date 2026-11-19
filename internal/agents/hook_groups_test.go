package agents

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// hookGroup builds one {matcher, hooks[]} group whose handlers carry the
// given commands. A command that starts with "own" gets timeout 5, the
// way codegraph's own handler is written.
func hookGroup(matcher string, extra map[string]any, commands ...string) map[string]any {
	handlers := make([]any, 0, len(commands))
	for _, cmd := range commands {
		h := map[string]any{"type": "command", "command": cmd}
		if strings.HasPrefix(cmd, "own") {
			h["timeout"] = float64(5)
		}
		handlers = append(handlers, h)
	}
	g := map[string]any{"matcher": matcher, "hooks": handlers}
	for k, v := range extra {
		g[k] = v
	}
	return g
}

// staleOwnGroup is a group holding codegraph's handler with a timeout that
// differs from the one hookGroup writes, so the group needs rewriting.
func staleOwnGroup(matcher string, extra map[string]any, commands ...string) map[string]any {
	g := hookGroup(matcher, extra, commands...)
	for _, h := range g["hooks"].([]any) {
		if hm := h.(map[string]any); strings.HasPrefix(hm["command"].(string), "own") {
			hm["timeout"] = float64(10)
		}
	}
	return g
}

// jsonOf marshals v and decodes it back, so values built in Go compare
// equal to values read from a file.
func jsonOf(t *testing.T, v any) any {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var out any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// writeHooksFile writes {"hooks": {event: groups}} to path.
func writeHooksFile(t *testing.T, path, event string, groups ...any) {
	t.Helper()
	data, err := json.Marshal(map[string]any{"hooks": map[string]any{event: groups}})
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, path, string(data))
}

// eventGroups returns hooks.<event> of the file at path as decoded JSON.
func eventGroups(t *testing.T, path, event string) []any {
	t.Helper()
	groups, _ := readSettingsHooks(t, path)[event].([]any)
	return groups
}

// TestWriteHookEntry_SplitKeepsForeignGroupIndexes: when codegraph's handler
// shares a group with a user handler and needs rewriting, the user handler
// stays in a group of its own at the original index with the original
// group's other fields, no other foreign group changes index, and
// codegraph's group goes last.
func TestWriteHookEntry_SplitKeepsForeignGroupIndexes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hooks.json")
	extra := map[string]any{"note": "keep me"}
	groupA := hookGroup("^Read$", nil, "echo a")
	groupB := hookGroup("^Glob$", nil, "echo b")
	own := hookGroup("^Bash$", nil, "own-cmd")
	writeHooksFile(t, path, "PreToolUse", groupA, staleOwnGroup("^Bash$", extra, "own-cmd", "echo user"), groupB)

	res, err := writeHookEntry(path, "PreToolUse", []any{own}, []string{"own-cmd"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Action != ActionUpdated || !res.HookGroupSplit {
		t.Fatalf("result = %+v, want an update that reports the split", res)
	}
	want := jsonOf(t, []any{groupA, hookGroup("^Bash$", extra, "echo user"), groupB, own})
	if got := eventGroups(t, path, "PreToolUse"); !reflect.DeepEqual(got, want) {
		t.Fatalf("groups after the split:\n got=%v\nwant=%v", got, want)
	}

	again, err := writeHookEntry(path, "PreToolUse", []any{own}, []string{"own-cmd"})
	if err != nil {
		t.Fatal(err)
	}
	if again.Action != ActionUnchanged || again.HookGroupSplit {
		t.Fatalf("second write = %+v, want unchanged and no split", again)
	}
}

// TestWriteHookEntry_CurrentMixedGroupIsLeftAlone: a group that mixes
// codegraph's handler with a user handler stays as it is while codegraph's
// part already matches what install writes.
func TestWriteHookEntry_CurrentMixedGroupIsLeftAlone(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hooks.json")
	own := hookGroup("^Bash$", nil, "own-cmd")
	writeHooksFile(t, path, "PreToolUse", hookGroup("^Bash$", nil, "own-cmd", "echo user"))
	before := readFile(t, path)

	res, err := writeHookEntry(path, "PreToolUse", []any{own}, []string{"own-cmd"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Action != ActionUnchanged || res.HookGroupSplit || readFile(t, path) != before {
		t.Fatalf("result = %+v, file changed = %v, want a no-op", res, readFile(t, path) != before)
	}
}

// TestWriteHookEntry_MixedGroupBesideFullyOwnedGroup: codegraph's group goes
// where the fully owned group was, so a foreign group after it keeps its index.
func TestWriteHookEntry_MixedGroupBesideFullyOwnedGroup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hooks.json")
	groupA := hookGroup("^Read$", nil, "echo a")
	groupB := hookGroup("^Glob$", nil, "echo b")
	own := hookGroup("^Bash$", nil, "own-cmd")
	writeHooksFile(t, path, "PreToolUse", groupA, staleOwnGroup("^Bash$", nil, "own-cmd", "echo user"), staleOwnGroup("^Bash$", nil, "own-cmd"), groupB)

	if _, err := writeHookEntry(path, "PreToolUse", []any{own}, []string{"own-cmd"}); err != nil {
		t.Fatal(err)
	}
	want := jsonOf(t, []any{groupA, hookGroup("^Bash$", nil, "echo user"), own, groupB})
	if got := eventGroups(t, path, "PreToolUse"); !reflect.DeepEqual(got, want) {
		t.Fatalf("groups:\n got=%v\nwant=%v", got, want)
	}
}

// TestWriteHookEntry_WrongTypeHooksValuesAreErrors: a "hooks" value that is
// not an object, or an event value that is not an array, is an error naming
// the file and the key, and the file stays as it was. A null or absent key is
// not an error.
func TestWriteHookEntry_WrongTypeHooksValuesAreErrors(t *testing.T) {
	own := []any{hookGroup("^Bash$", nil, "own-cmd")}
	cases := []struct {
		name, content, wantKey string
	}{
		{"hooks is an array", `{"hooks":["x"]}`, `"hooks"`},
		{"hooks is a string", `{"hooks":"x"}`, `"hooks"`},
		{"event is an object", `{"hooks":{"PreToolUse":{"a":1}}}`, `"hooks.PreToolUse"`},
		{"event is a string", `{"hooks":{"PreToolUse":"x"}}`, `"hooks.PreToolUse"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "settings.json")
			writeFile(t, path, tc.content)
			for name, run := range map[string]func() error{
				"write":  func() error { _, err := writeHookEntry(path, "PreToolUse", own, []string{"own-cmd"}); return err },
				"remove": func() error { _, err := removeHookEntry(path, "PreToolUse", []string{"own-cmd"}); return err },
				"probe":  func() error { _, err := hasOwnHookBlock(path, "PreToolUse", []string{"own-cmd"}); return err },
			} {
				err := run()
				if err == nil || !strings.Contains(err.Error(), path) || !strings.Contains(err.Error(), tc.wantKey) {
					t.Errorf("%s error = %v, want one naming %s and %s", name, err, path, tc.wantKey)
				}
				if got := readFile(t, path); got != tc.content {
					t.Errorf("%s changed the file to %q", name, got)
				}
			}
		})
	}

	for _, content := range []string{`{"hooks":null}`, `{"hooks":{"PreToolUse":null}}`, `{}`} {
		path := filepath.Join(t.TempDir(), "settings.json")
		writeFile(t, path, content)
		if _, err := writeHookEntry(path, "PreToolUse", own, []string{"own-cmd"}); err != nil {
			t.Errorf("write over %s: %v", content, err)
		}
	}
}

// TestAddClaudeAllowPermission_WrongTypeValuesAreErrors: a "permissions"
// value that is not an object, or an allow value that is not an array, is an
// error naming the file and the key, and the file stays as it was.
func TestAddClaudeAllowPermission_WrongTypeValuesAreErrors(t *testing.T) {
	cases := []struct{ content, wantKey string }{
		{`{"permissions":["x"]}`, `"permissions"`},
		{`{"permissions":{"allow":"x"}}`, `"permissions.allow"`},
	}
	for _, tc := range cases {
		path := filepath.Join(t.TempDir(), "settings.json")
		writeFile(t, path, tc.content)
		_, err := addClaudeAllowPermission(path)
		if err == nil || !strings.Contains(err.Error(), path) || !strings.Contains(err.Error(), tc.wantKey) {
			t.Errorf("addClaudeAllowPermission(%s) error = %v, want one naming %s", tc.content, err, tc.wantKey)
		}
		if got := readFile(t, path); got != tc.content {
			t.Errorf("addClaudeAllowPermission changed the file to %q", got)
		}
	}
}

// TestClaudeInstall_WrongTypeHooksValueIsReportedAndKept: an install over a
// settings.json whose "hooks" is not an object reports an error naming the
// file and leaves the file as it was.
func TestClaudeInstall_WrongTypeHooksValueIsReportedAndKept(t *testing.T) {
	fakeHome(t)
	t.Chdir(t.TempDir())
	settings, err := claudeSettingsPath(LocationLocal)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, settings, `{"hooks":["x"]}`)

	res := claudeTarget{}.Install(LocationLocal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
	if !errorNaming(res, settings, `"hooks"`) {
		t.Fatalf("Install errors = %v, want one naming %s and \"hooks\"", res.Errors, settings)
	}
	if got := readFile(t, settings); got != `{"hooks":["x"]}` {
		t.Fatalf("Install changed settings.json to %q", got)
	}
	if _, err := os.Stat(settings); err != nil {
		t.Fatal(err)
	}
}

// TestCodexInstall_SplitGroupNoteAndIndexes: an opt-in Codex install over a
// hooks.json whose codegraph group also holds a user hook keeps the other
// groups at their indexes, keeps the user hook in a group of its own with the
// group's matcher and extra field, and says the user hook needs review.
func TestCodexInstall_SplitGroupNoteAndIndexes(t *testing.T) {
	fakeHome(t)
	t.Chdir(t.TempDir())
	codexHome := filepath.Join(t.TempDir(), "codex")
	t.Setenv("CODEX_HOME", codexHome)
	hooksPath := filepath.Join(codexHome, "hooks.json")
	ownCmd, err := codexPreToolHookCommand(LocationGlobal)
	if err != nil {
		t.Fatal(err)
	}
	extra := map[string]any{"note": "keep me"}
	groupA := hookGroup("^Read$", nil, "echo a")
	groupB := hookGroup("^Glob$", nil, "echo b")
	writeHooksFile(t, hooksPath, "PreToolUse", groupA, staleOwnGroup("^Bash$", extra, ownCmd, "echo user"), groupB)

	res := codexTarget{}.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: PreToolNudgeOn})
	assertNoErrors(t, res)

	got := eventGroups(t, hooksPath, "PreToolUse")
	if len(got) != 4 || !reflect.DeepEqual(got[0], jsonOf(t, groupA)) || !reflect.DeepEqual(got[2], jsonOf(t, groupB)) {
		t.Fatalf("groups after install = %v, want the two foreign groups at indexes 0 and 2 and four groups in all", got)
	}
	if want := jsonOf(t, hookGroup("^Bash$", extra, "echo user")); !reflect.DeepEqual(got[1], want) {
		t.Errorf("the user hook's group = %v, want %v", got[1], want)
	}
	if cmds := settingsHandlerCommands(t, hooksPath, "PreToolUse"); cmds[len(cmds)-1] != ownCmd {
		t.Errorf("handler commands = %q, want codegraph's last", cmds)
	}
	if !noteNaming(res, "review your other hook") {
		t.Errorf("notes = %q, want the trust note to mention the split user hook", res.Notes)
	}
}
