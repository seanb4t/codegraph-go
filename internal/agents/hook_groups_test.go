package agents

import (
	"encoding/json"
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
	if res.Action != ActionUpdated || !res.ForeignHookMoved {
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
	if again.Action != ActionUnchanged || again.ForeignHookMoved {
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
	if res.Action != ActionUnchanged || res.ForeignHookMoved || readFile(t, path) != before {
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

// TestHookEntryHelpers_WrongTypeHooksValues: a "hooks" value that is not an
// object, or an event value that is not an array, makes the write fail with
// an error that names the key and leaves the file as it was. The read-only
// probe fails the same way and names the file too. The remove side reports
// not-found, because codegraph never wrote into such a value. A null or
// absent key is not wrong.
func TestHookEntryHelpers_WrongTypeHooksValues(t *testing.T) {
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

			if _, err := writeHookEntry(path, "PreToolUse", own, []string{"own-cmd"}); err == nil || !strings.Contains(err.Error(), tc.wantKey) {
				t.Errorf("write error = %v, want one naming %s", err, tc.wantKey)
			}
			if _, err := hasOwnHookBlock(path, "PreToolUse", []string{"own-cmd"}); err == nil || !strings.Contains(err.Error(), tc.wantKey) || !strings.Contains(err.Error(), path) {
				t.Errorf("probe error = %v, want one naming %s and %s", err, path, tc.wantKey)
			}
			if res, err := removeHookEntry(path, "PreToolUse", []string{"own-cmd"}); err != nil || res.Action != ActionNotFound {
				t.Errorf("remove = %+v, %v, want not-found and no error", res, err)
			}
			if got := readFile(t, path); got != tc.content {
				t.Errorf("a helper changed the file to %q", got)
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

// TestClaudeAllowPermission_WrongTypeValues: a "permissions" value that is
// not an object, or an allow value that is not an array, makes the add fail
// with an error that names the key and leaves the file as it was. The remove
// reports not-found.
func TestClaudeAllowPermission_WrongTypeValues(t *testing.T) {
	cases := []struct{ content, wantKey string }{
		{`{"permissions":["x"]}`, `"permissions"`},
		{`{"permissions":{"allow":"x"}}`, `"permissions.allow"`},
	}
	for _, tc := range cases {
		path := filepath.Join(t.TempDir(), "settings.json")
		writeFile(t, path, tc.content)
		if _, err := addClaudeAllowPermission(path); err == nil || !strings.Contains(err.Error(), tc.wantKey) {
			t.Errorf("add over %s: error = %v, want one naming %s", tc.content, err, tc.wantKey)
		}
		if res, err := removeClaudeAllowPermission(path); err != nil || res.Action != ActionNotFound {
			t.Errorf("remove over %s = %+v, %v, want not-found and no error", tc.content, res, err)
		}
		if got := readFile(t, path); got != tc.content {
			t.Errorf("a helper changed the file to %q", got)
		}
	}
}

// TestClaudeInstall_WrongTypeValueSkipsEverySettingsStep: with the allow
// entry and the nudge on, a settings.json whose hooks, hooks.PreToolUse or
// permissions value has the wrong JSON type gets one error that names the
// file, stays byte for byte as it was, and gets no hook script or guard.
func TestClaudeInstall_WrongTypeValueSkipsEverySettingsStep(t *testing.T) {
	for _, content := range []string{`{"hooks":{"PreToolUse":"x"}}`, `{"permissions":["x"]}`, `{"hooks":["x"]}`} {
		t.Run(content, func(t *testing.T) {
			fakeHome(t)
			t.Chdir(t.TempDir())
			settings, err := claudeSettingsPath(LocationLocal)
			if err != nil {
				t.Fatal(err)
			}
			writeFile(t, settings, content)

			res := claudeTarget{}.Install(LocationLocal, InstallOptions{ExecPath: "/usr/local/bin/codegraph", AutoAllow: true, PreToolNudge: PreToolNudgeOn})
			named := 0
			for _, e := range res.Errors {
				if strings.Contains(e.Error(), settings) {
					named++
				}
			}
			if named != 1 {
				t.Errorf("errors naming %s = %d (%v), want exactly one", settings, named, res.Errors)
			}
			if got := readFile(t, settings); got != content {
				t.Errorf("settings.json changed to %q", got)
			}
			assertMissing(t, filepath.Join(".claude", "hooks", "pretooluse-nudge.sh"))
			assertMissing(t, filepath.Join(".claude", "hooks", "session-nudge.sh"))
		})
	}
}

// TestCodexInstall_SplitGroupNoteAndIndexes: an opt-in Codex install over a
// hooks.json whose codegraph group also holds a user hook keeps the other
// groups at their indexes, keeps the user hook in a group of its own with the
// group's matcher and extra field, and says the user hook needs review.
func TestCodexInstall_SplitGroupNoteAndIndexes(t *testing.T) {
	fakeHome(t)
	t.Chdir(t.TempDir())
	codexHome := useCodexHome(t)
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

// TestHookGroups_InstallReinstallUninstallKeepForeignGroups drives one hooks
// array through install, a second install, uninstall and a reinstall, in the
// shapes that matter for Codex's position-keyed trust. Each step asserts the
// exact group list and whether the write reports a foreign hook moved. A
// plain rewrite of a fully owned group is not a move.
func TestHookGroups_InstallReinstallUninstallKeepForeignGroups(t *testing.T) {
	const ownCmd, legacyCmd = "own-cmd", "own-legacy"
	extra := map[string]any{"note": "keep me"}
	a := hookGroup("^Read$", nil, "echo a")
	b := hookGroup("^Glob$", nil, "echo b")
	c := hookGroup("^Grep$", nil, "echo c")
	own := hookGroup("^Bash$", nil, ownCmd)
	legacy := hookGroup("^Bash$", nil, legacyCmd)
	stale := staleOwnGroup("^Bash$", nil, ownCmd)

	cases := []struct {
		name          string
		seed          []any
		afterInstall  []any
		installMoved  bool
		afterRemove   []any
		removeMoved   bool
		afterReinstal []any
	}{
		{
			name:          "mixed group between two foreign groups",
			seed:          []any{a, staleOwnGroup("^Bash$", extra, ownCmd, "echo u"), b},
			afterInstall:  []any{a, hookGroup("^Bash$", extra, "echo u"), b, own},
			installMoved:  true,
			afterRemove:   []any{a, hookGroup("^Bash$", extra, "echo u"), b},
			afterReinstal: []any{a, hookGroup("^Bash$", extra, "echo u"), b, own},
		},
		{
			name:          "mixed group with a hook on each side of codegraph's",
			seed:          []any{a, staleOwnGroup("^Bash$", extra, "echo u", ownCmd, "echo v"), b},
			afterInstall:  []any{a, hookGroup("^Bash$", extra, "echo u", "echo v"), b, own},
			installMoved:  true,
			afterRemove:   []any{a, hookGroup("^Bash$", extra, "echo u", "echo v"), b},
			afterReinstal: []any{a, hookGroup("^Bash$", extra, "echo u", "echo v"), b, own},
		},
		{
			name:          "two owned groups collapse and a later group moves",
			seed:          []any{a, own, b, legacy, c},
			afterInstall:  []any{a, own, b, c},
			installMoved:  true,
			afterRemove:   []any{a, b, c},
			removeMoved:   true,
			afterReinstal: []any{a, b, c, own},
		},
		{
			name:          "two owned groups collapse and the later group keeps its index",
			seed:          []any{a, legacy, b, own},
			afterInstall:  []any{a, own, b},
			afterRemove:   []any{a, b},
			removeMoved:   true,
			afterReinstal: []any{a, b, own},
		},
		{
			name:          "owned group first",
			seed:          []any{stale, a, b},
			afterInstall:  []any{own, a, b},
			afterRemove:   []any{a, b},
			removeMoved:   true,
			afterReinstal: []any{a, b, own},
		},
		{
			name:          "owned group last",
			seed:          []any{a, b, stale},
			afterInstall:  []any{a, b, own},
			afterRemove:   []any{a, b},
			afterReinstal: []any{a, b, own},
		},
	}
	owned := []string{ownCmd, legacyCmd}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "hooks.json")
			writeHooksFile(t, path, "PreToolUse", tc.seed...)
			assertGroups := func(step string, want []any) {
				t.Helper()
				if got := eventGroups(t, path, "PreToolUse"); !reflect.DeepEqual(got, jsonOf(t, want)) {
					t.Fatalf("%s:\n got=%v\nwant=%v", step, got, jsonOf(t, want))
				}
			}

			res, err := writeHookEntry(path, "PreToolUse", []any{own}, owned)
			if err != nil || res.ForeignHookMoved != tc.installMoved {
				t.Fatalf("install = %+v, %v, want ForeignHookMoved %v", res, err, tc.installMoved)
			}
			assertGroups("after install", tc.afterInstall)

			if res, err = writeHookEntry(path, "PreToolUse", []any{own}, owned); err != nil || res.Action != ActionUnchanged || res.ForeignHookMoved {
				t.Fatalf("second install = %+v, %v, want unchanged and no move", res, err)
			}

			res, err = removeHookEntry(path, "PreToolUse", owned)
			if err != nil || res.Action != ActionRemoved || res.ForeignHookMoved != tc.removeMoved {
				t.Fatalf("remove = %+v, %v, want removed with ForeignHookMoved %v", res, err, tc.removeMoved)
			}
			assertGroups("after remove", tc.afterRemove)

			res, err = writeHookEntry(path, "PreToolUse", []any{own}, owned)
			if err != nil || res.ForeignHookMoved {
				t.Fatalf("reinstall = %+v, %v, want no move", res, err)
			}
			assertGroups("after reinstall", tc.afterReinstal)
		})
	}
}

// TestRemoveHookEntry_MixedGroupKeepsUserHookMatcherAndFields: removing
// codegraph's handler from a group that also holds a user handler leaves that
// group in place with the user handler, the matcher and the extra field.
func TestRemoveHookEntry_MixedGroupKeepsUserHookMatcherAndFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hooks.json")
	extra := map[string]any{"note": "x"}
	a := hookGroup("^Read$", nil, "echo a")
	b := hookGroup("^Glob$", nil, "echo b")
	writeHooksFile(t, path, "PreToolUse", a, hookGroup("^Bash$", extra, "own-cmd", "echo user"), b)

	res, err := removeHookEntry(path, "PreToolUse", []string{"own-cmd"})
	if err != nil || res.Action != ActionRemoved || !res.ForeignHookMoved {
		t.Fatalf("remove = %+v, %v, want removed and a foreign hook moved", res, err)
	}
	want := jsonOf(t, []any{a, hookGroup("^Bash$", extra, "echo user"), b})
	if got := eventGroups(t, path, "PreToolUse"); !reflect.DeepEqual(got, want) {
		t.Fatalf("groups after remove:\n got=%v\nwant=%v", got, want)
	}
}

// TestCodexUninstall_MixedGroupKeepsUserHookAndSaysToReviewIt: a global
// uninstall, and an install with the nudge off, strip codegraph's handler
// from a mixed group, keep the user handler with the group's matcher and
// extra field, and add the note that says to review it.
func TestCodexUninstall_MixedGroupKeepsUserHookAndSaysToReviewIt(t *testing.T) {
	steps := map[string]func(c codexTarget) WriteResult{
		"uninstall": func(c codexTarget) WriteResult { return c.Uninstall(LocationGlobal) },
		"install with the nudge off": func(c codexTarget) WriteResult {
			return c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: PreToolNudgeOff})
		},
	}
	for name, step := range steps {
		t.Run(name, func(t *testing.T) {
			fakeHome(t)
			t.Chdir(t.TempDir())
			codexHome := useCodexHome(t)
			hooksPath := filepath.Join(codexHome, "hooks.json")
			ownCmd, err := codexPreToolHookCommand(LocationGlobal)
			if err != nil {
				t.Fatal(err)
			}
			extra := map[string]any{"note": "x"}
			a := hookGroup("^Read$", nil, "echo a")
			b := hookGroup("^Glob$", nil, "echo b")
			writeHooksFile(t, hooksPath, "PreToolUse", a, hookGroup("^Bash$", extra, ownCmd, "echo user"), b)

			res := step(codexTarget{})
			assertNoErrors(t, res)
			want := jsonOf(t, []any{a, hookGroup("^Bash$", extra, "echo user"), b})
			if got := eventGroups(t, hooksPath, "PreToolUse"); !reflect.DeepEqual(got, want) {
				t.Errorf("groups after removal:\n got=%v\nwant=%v", got, want)
			}
			if !noteNaming(res, "review your hooks in /hooks") {
				t.Errorf("notes = %q, want the removal note", res.Notes)
			}
		})
	}
}

// TestClaudeUninstall_MixedGroupKeepsUserHook: an uninstall strips codegraph's
// PreToolUse handler from a mixed group and keeps the user handler with the
// group's matcher and extra field.
func TestClaudeUninstall_MixedGroupKeepsUserHook(t *testing.T) {
	fakeHome(t)
	t.Chdir(t.TempDir())
	settings, err := claudeSettingsPath(LocationLocal)
	if err != nil {
		t.Fatal(err)
	}
	ownCmd, err := claudePreToolHookCommand(LocationLocal)
	if err != nil {
		t.Fatal(err)
	}
	extra := map[string]any{"note": "x"}
	a := hookGroup("^Read$", nil, "echo a")
	writeHooksFile(t, settings, "PreToolUse", a, hookGroup("^Bash$", extra, ownCmd, "echo user"))

	assertNoErrors(t, claudeTarget{}.Uninstall(LocationLocal))
	want := jsonOf(t, []any{a, hookGroup("^Bash$", extra, "echo user")})
	if got := eventGroups(t, settings, "PreToolUse"); !reflect.DeepEqual(got, want) {
		t.Fatalf("groups after uninstall:\n got=%v\nwant=%v", got, want)
	}
}

// TestCodexInstall_WrongTypeHooksValueWritesNoGuard: a hooks.json value of
// the wrong type fails the install with no guard script left behind and the
// file unchanged. An uninstall that cannot read hooks.json keeps the guard,
// so no registration points at a missing script.
func TestCodexInstall_WrongTypeHooksValueWritesNoGuard(t *testing.T) {
	fakeHome(t)
	t.Chdir(t.TempDir())
	hooksPath, err := codexHooksJSONPath(LocationLocal)
	if err != nil {
		t.Fatal(err)
	}
	guard, err := codexPreToolGuardPath(LocationLocal)
	if err != nil {
		t.Fatal(err)
	}
	const wrong = `{"hooks":{"PreToolUse":"x"}}`
	writeFile(t, hooksPath, wrong)

	res := codexTarget{}.Install(LocationLocal, InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: PreToolNudgeOn})
	if !errorNaming(res, hooksPath, `"hooks.PreToolUse"`) {
		t.Errorf("errors = %v, want one naming %s and the key", res.Errors, hooksPath)
	}
	assertMissing(t, guard)
	if got := readFile(t, hooksPath); got != wrong {
		t.Errorf("hooks.json changed to %q", got)
	}

	writeFile(t, hooksPath, "{not json")
	writeFile(t, guard, "#!/bin/sh\n")
	res = codexTarget{}.Uninstall(LocationLocal)
	if !errorNaming(res, hooksPath) {
		t.Errorf("uninstall errors = %v, want one naming %s", res.Errors, hooksPath)
	}
	if !fileExists(guard) {
		t.Errorf("uninstall removed the guard although it could not remove the registration")
	}
}

// TestClaudeUninstall_KeepsGuardWhenRegistrationStays: an uninstall that
// cannot parse settings.json keeps the PreToolUse guard that the
// registration points at.
func TestClaudeUninstall_KeepsGuardWhenRegistrationStays(t *testing.T) {
	fakeHome(t)
	t.Chdir(t.TempDir())
	settings, err := claudeSettingsPath(LocationLocal)
	if err != nil {
		t.Fatal(err)
	}
	guard, err := claudePreToolGuardPath(LocationLocal)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, settings, "{not json")
	writeFile(t, guard, "#!/bin/sh\n")

	res := claudeTarget{}.Uninstall(LocationLocal)
	if !errorNaming(res, settings) {
		t.Errorf("uninstall errors = %v, want one naming %s", res.Errors, settings)
	}
	if !fileExists(guard) {
		t.Errorf("uninstall removed the guard although it could not remove the registration")
	}
}
