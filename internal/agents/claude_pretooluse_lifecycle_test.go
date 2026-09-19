package agents

import (
	"encoding/json"
	"os"
	"sort"
	"strings"
	"testing"
)

// The sticky opt-in lifecycle of the Claude Code PreToolUse nudge (v0.14.0
// Phase 6, D-10/D-11, NUDGE-06). Every test isolates HOME with fakeHome and
// the project with t.Chdir(t.TempDir()); ExecPaths are absolute literals.

const (
	lifecycleExecA = "/opt/a/codegraph"
	lifecycleExecB = "/opt/b/codegraph"
)

// lifecycleInstall runs a Claude Install and fails the test on any error.
func lifecycleInstall(t *testing.T, loc Location, execPath string, mode PreToolNudgeMode) WriteResult {
	t.Helper()
	res := claudeTarget{}.Install(loc, InstallOptions{ExecPath: execPath, PreToolNudge: mode})
	if len(res.Errors) != 0 {
		t.Fatalf("Install(%s, mode=%d) errors: %v", loc, mode, res.Errors)
	}
	return res
}

// lifecycleManifest reads Claude's manifest at loc, failing unless it is
// present and decodable.
func lifecycleManifest(t *testing.T, loc Location) skillManifest {
	t.Helper()
	path, err := claudeManifestPath(loc)
	if err != nil {
		t.Fatalf("claudeManifestPath(%s): %v", loc, err)
	}
	m, present, err := readManifest(path)
	if err != nil || !present {
		t.Fatalf("readManifest(%s): present=%v err=%v", path, present, err)
	}
	return m
}

// lifecyclePaths resolves the guard, settings and own PreToolUse command at loc.
func lifecyclePaths(t *testing.T, loc Location) (guard, settings, ownCommand string) {
	t.Helper()
	var err error
	if guard, err = claudePreToolGuardPath(loc); err != nil {
		t.Fatalf("claudePreToolGuardPath(%s): %v", loc, err)
	}
	if settings, err = claudeSettingsPath(loc); err != nil {
		t.Fatalf("claudeSettingsPath(%s): %v", loc, err)
	}
	if ownCommand, err = claudePreToolHookCommand(loc); err != nil {
		t.Fatalf("claudePreToolHookCommand(%s): %v", loc, err)
	}
	return guard, settings, ownCommand
}

// preToolUseBlocksAt returns hooks.PreToolUse from settingsPath, or nil when
// the file or the key is absent.
func preToolUseBlocksAt(t *testing.T, settingsPath string) []any {
	t.Helper()
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		return nil
	}
	blocks, _ := readSettingsHooks(t, settingsPath)["PreToolUse"].([]any)
	return blocks
}

// countOwnPreToolHandlers counts PreToolUse handlers whose command is
// exactly ownCommand (the exact-identity ownership rule, 242ec0a).
func countOwnPreToolHandlers(t *testing.T, settingsPath, ownCommand string) int {
	t.Helper()
	n := 0
	for _, b := range preToolUseBlocksAt(t, settingsPath) {
		block, _ := b.(map[string]any)
		handlers, _ := block["hooks"].([]any)
		for _, h := range handlers {
			if handler, _ := h.(map[string]any); handler != nil && handler["command"] == ownCommand {
				n++
			}
		}
	}
	return n
}

// fileAction returns the action recorded for path in res.Files, or "" when
// res.Files has no entry for it.
func fileAction(res WriteResult, path string) FileAction {
	for _, fr := range res.Files {
		if fr.Path == path {
			return fr.Action
		}
	}
	return ""
}

func TestPreToolNudge_OnRecordsManifestKeys(t *testing.T) {
	fakeHome(t)
	t.Chdir(t.TempDir())

	lifecycleInstall(t, LocationGlobal, lifecycleExecA, PreToolNudgeOn)
	guard, _, _ := lifecyclePaths(t, LocationGlobal)
	m := lifecycleManifest(t, LocationGlobal)

	guardBytes, err := os.ReadFile(guard)
	if err != nil {
		t.Fatalf("read guard: %v", err)
	}
	if got, want := m.Files[manifestKeyPreToolGuard], hashContent(guardBytes); got != want {
		t.Fatalf("manifest %q = %q, want the hash of the rendered guard on disk %q", manifestKeyPreToolGuard, got, want)
	}
	blocks, _, err := claudePreToolUseBlocks(LocationGlobal)
	if err != nil {
		t.Fatalf("claudePreToolUseBlocks: %v", err)
	}
	wantFrag, err := hashOwnedHookBlocks(blocks)
	if err != nil {
		t.Fatalf("hashOwnedHookBlocks: %v", err)
	}
	if got := m.Files[manifestKeyPreToolFrag]; got != wantFrag {
		t.Fatalf("manifest %q = %q, want %q", manifestKeyPreToolFrag, got, wantFrag)
	}
}

func TestPreToolNudge_KeepRefreshesWhenRecorded(t *testing.T) {
	fakeHome(t)
	t.Chdir(t.TempDir())
	guard, settings, own := lifecyclePaths(t, LocationGlobal)

	lifecycleInstall(t, LocationGlobal, lifecycleExecA, PreToolNudgeOn)
	if !strings.Contains(readFile(t, guard), "\ncodegraph_bin='"+lifecycleExecA+"'\n") {
		t.Fatalf("precondition: the On install did not render %s into the guard", lifecycleExecA)
	}

	res := lifecycleInstall(t, LocationGlobal, lifecycleExecB, PreToolNudgeKeep)
	content := readFile(t, guard)
	if !strings.Contains(content, "\ncodegraph_bin='"+lifecycleExecB+"'\n") {
		t.Fatalf("Keep with a recorded opt-in did not re-render the guard for the moved binary (D-01b):\n%s", content)
	}
	if got := fileAction(res, guard); got != ActionUpdated {
		t.Fatalf("guard FileResult = %q, want %q", got, ActionUpdated)
	}
	if n := countOwnPreToolHandlers(t, settings, own); n != 6 {
		t.Fatalf("own PreToolUse handlers after Keep = %d, want 6", n)
	}
	if got, want := lifecycleManifest(t, LocationGlobal).Files[manifestKeyPreToolGuard], hashContent([]byte(content)); got != want {
		t.Fatalf("manifest guard hash = %q, want the refreshed guard's %q", got, want)
	}
}

func TestPreToolNudge_KeepNoopWhenNotRecorded(t *testing.T) {
	fakeHome(t)
	t.Chdir(t.TempDir())
	guard, settings, _ := lifecyclePaths(t, LocationGlobal)

	res := lifecycleInstall(t, LocationGlobal, lifecycleExecA, PreToolNudgeKeep)
	if _, err := os.Lstat(guard); !os.IsNotExist(err) {
		t.Fatalf("a fresh Keep install wrote %s (Lstat err %v)", guard, err)
	}
	if fileAction(res, guard) != "" {
		t.Fatalf("a fresh Keep install reported the guard: %#v", res.Files)
	}
	if _, ok := readSettingsHooks(t, settings)["PreToolUse"]; ok {
		t.Fatalf("a fresh Keep install wrote hooks.PreToolUse")
	}
	m := lifecycleManifest(t, LocationGlobal)
	for _, k := range []string{manifestKeyPreToolGuard, manifestKeyPreToolFrag} {
		if _, ok := m.Files[k]; ok {
			t.Fatalf("a fresh Keep install recorded %q: %#v", k, m.Files)
		}
	}

	// Positive control: an On install into the same scope records both keys,
	// so the absence checks above can see presence.
	lifecycleInstall(t, LocationGlobal, lifecycleExecA, PreToolNudgeOn)
	m = lifecycleManifest(t, LocationGlobal)
	for _, k := range []string{manifestKeyPreToolGuard, manifestKeyPreToolFrag} {
		if _, ok := m.Files[k]; !ok {
			t.Fatalf("positive control: an On install did not record %q: %#v", k, m.Files)
		}
	}
}

func TestPreToolNudge_KeepWithUnreadableManifestTouchesNothing(t *testing.T) {
	fakeHome(t)
	t.Chdir(t.TempDir())
	guard, settings, own := lifecyclePaths(t, LocationGlobal)

	lifecycleInstall(t, LocationGlobal, lifecycleExecA, PreToolNudgeOn)
	before := readFile(t, guard)
	if !strings.Contains(before, "\ncodegraph_bin='"+lifecycleExecA+"'\n") {
		t.Fatalf("precondition: the On install did not render %s into the guard", lifecycleExecA)
	}
	manifestPath, err := claudeManifestPath(LocationGlobal)
	if err != nil {
		t.Fatalf("claudeManifestPath: %v", err)
	}
	writeFile(t, manifestPath, "{not json")
	if recorded, readable := preToolNudgeRecorded(LocationGlobal); recorded || readable {
		t.Fatalf("preToolNudgeRecorded on a corrupt manifest = (%v, %v), want (false, false)", recorded, readable)
	}

	lifecycleInstall(t, LocationGlobal, lifecycleExecB, PreToolNudgeKeep)
	if after := readFile(t, guard); after != before {
		t.Fatalf("Keep with an unreadable manifest refreshed the guard:\n%s", after)
	}
	if n := countOwnPreToolHandlers(t, settings, own); n != 6 {
		t.Fatalf("Keep with an unreadable manifest changed the own PreToolUse handlers: %d, want 6", n)
	}
}

func TestPreToolNudge_OffRemovesAndForgets(t *testing.T) {
	fakeHome(t)
	t.Chdir(t.TempDir())
	guard, settings, own := lifecyclePaths(t, LocationGlobal)

	lifecycleInstall(t, LocationGlobal, lifecycleExecA, PreToolNudgeOn)
	if n := countOwnPreToolHandlers(t, settings, own); n != 6 {
		t.Fatalf("precondition: own PreToolUse handlers after On = %d, want 6", n)
	}

	res := lifecycleInstall(t, LocationGlobal, lifecycleExecA, PreToolNudgeOff)
	if _, err := os.Lstat(guard); !os.IsNotExist(err) {
		t.Fatalf("Off left the guard %s (Lstat err %v)", guard, err)
	}
	if got := fileAction(res, guard); got != ActionRemoved {
		t.Fatalf("Off guard FileResult = %q, want %q", got, ActionRemoved)
	}
	if n := countOwnPreToolHandlers(t, settings, own); n != 0 {
		t.Fatalf("Off left %d own PreToolUse handlers", n)
	}
	m := lifecycleManifest(t, LocationGlobal)
	for _, k := range []string{manifestKeyPreToolGuard, manifestKeyPreToolFrag} {
		if _, ok := m.Files[k]; ok {
			t.Fatalf("Off kept the manifest record %q: %#v", k, m.Files)
		}
	}

	lifecycleInstall(t, LocationGlobal, lifecycleExecB, PreToolNudgeKeep)
	if _, err := os.Lstat(guard); !os.IsNotExist(err) {
		t.Fatalf("Keep after Off re-added the guard (Lstat err %v)", err)
	}
	if n := countOwnPreToolHandlers(t, settings, own); n != 0 {
		t.Fatalf("Keep after Off re-added %d own PreToolUse handlers", n)
	}
}

// actionList renders res.Files as sorted "action path" lines.
func actionList(res WriteResult) []string {
	out := make([]string, 0, len(res.Files))
	for _, fr := range res.Files {
		out = append(out, string(fr.Action)+" "+fr.Path)
	}
	sort.Strings(out)
	return out
}

func TestPreToolNudge_OffWhenNeverOptedAddsNoFiles(t *testing.T) {
	run := func(mode PreToolNudgeMode) []string {
		fakeHome(t)
		t.Chdir(t.TempDir())
		return actionList(lifecycleInstall(t, LocationLocal, lifecycleExecA, mode))
	}
	keep := run(PreToolNudgeKeep)
	off := run(PreToolNudgeOff)
	if strings.Join(keep, "\n") != strings.Join(off, "\n") {
		t.Fatalf("an Off install on a never-opted location reported different files than Keep:\nkeep:\n%s\noff:\n%s",
			strings.Join(keep, "\n"), strings.Join(off, "\n"))
	}
	guard, _, _ := lifecyclePaths(t, LocationLocal)
	for _, line := range off {
		if strings.HasSuffix(line, " "+guard) {
			t.Fatalf("an Off install on a never-opted location reported the guard: %q", line)
		}
	}
	if len(off) == 0 {
		t.Fatalf("the Off install reported no files at all — the comparison above would be vacuous")
	}
}

func TestPreToolNudge_UninstallAlwaysAttempts(t *testing.T) {
	t.Run("never-opted", func(t *testing.T) {
		fakeHome(t)
		t.Chdir(t.TempDir())
		guard, _, _ := lifecyclePaths(t, LocationGlobal)
		lifecycleInstall(t, LocationGlobal, lifecycleExecA, PreToolNudgeKeep)
		res := claudeTarget{}.Uninstall(LocationGlobal)
		if len(res.Errors) != 0 {
			t.Fatalf("Uninstall errors: %v", res.Errors)
		}
		if got := fileAction(res, guard); got != ActionNotFound {
			t.Fatalf("Uninstall guard FileResult = %q, want %q (D-11: always attempted)", got, ActionNotFound)
		}
	})
	t.Run("opted", func(t *testing.T) {
		fakeHome(t)
		t.Chdir(t.TempDir())
		guard, settings, own := lifecyclePaths(t, LocationGlobal)
		lifecycleInstall(t, LocationGlobal, lifecycleExecA, PreToolNudgeOn)
		res := claudeTarget{}.Uninstall(LocationGlobal)
		if len(res.Errors) != 0 {
			t.Fatalf("Uninstall errors: %v", res.Errors)
		}
		if got := fileAction(res, guard); got != ActionRemoved {
			t.Fatalf("Uninstall guard FileResult = %q, want %q", got, ActionRemoved)
		}
		if _, err := os.Lstat(guard); !os.IsNotExist(err) {
			t.Fatalf("Uninstall left the guard (Lstat err %v)", err)
		}
		if n := countOwnPreToolHandlers(t, settings, own); n != 0 {
			t.Fatalf("Uninstall left %d own PreToolUse handlers", n)
		}
	})
}

func TestPreToolNudge_ReinstallIsIdempotent(t *testing.T) {
	fakeHome(t)
	t.Chdir(t.TempDir())
	guard, settings, _ := lifecyclePaths(t, LocationGlobal)
	manifestPath, err := claudeManifestPath(LocationGlobal)
	if err != nil {
		t.Fatalf("claudeManifestPath: %v", err)
	}
	snapshot := func() [3]string {
		return [3]string{readFile(t, settings), readFile(t, guard), readFile(t, manifestPath)}
	}

	lifecycleInstall(t, LocationGlobal, lifecycleExecA, PreToolNudgeOn)
	first := snapshot()
	second := lifecycleInstall(t, LocationGlobal, lifecycleExecA, PreToolNudgeOn)
	if len(second.Files) == 0 {
		t.Fatalf("second install reported no files")
	}
	for _, fr := range second.Files {
		if fr.Action != ActionUnchanged {
			t.Errorf("second On install: %s = %q, want %q", fr.Path, fr.Action, ActionUnchanged)
		}
	}
	if fileAction(second, guard) != ActionUnchanged {
		t.Errorf("second On install did not report the guard as unchanged: %#v", second.Files)
	}
	if got := snapshot(); got != first {
		t.Fatalf("second On install changed bytes (settings, guard, manifest)")
	}

	u1 := claudeTarget{}.Uninstall(LocationGlobal)
	if len(u1.Errors) != 0 {
		t.Fatalf("first Uninstall errors: %v", u1.Errors)
	}
	if got := fileAction(u1, guard); got != ActionRemoved {
		t.Fatalf("first Uninstall guard = %q, want %q", got, ActionRemoved)
	}
	u2 := claudeTarget{}.Uninstall(LocationGlobal)
	if len(u2.Errors) != 0 {
		t.Fatalf("second Uninstall errors: %v", u2.Errors)
	}
	if got := fileAction(u2, guard); got != ActionNotFound {
		t.Fatalf("second Uninstall guard = %q, want %q", got, ActionNotFound)
	}
}

// TestPreToolNudge_HandEditedOwnEntryDuplicates: ownership is the exact
// command string, never the matcher (242ec0a). Each own PreToolUse block
// holds exactly one handler, so re-pointing ONE handler (the Bash(rg *) one)
// makes that block no longer codegraph's: the next opt-in install keeps it
// byte-identical and re-adds the full owned set beside it — duplicated, not
// overwritten via unedited siblings (NUDGE-06).
func TestPreToolNudge_HandEditedOwnEntryDuplicates(t *testing.T) {
	fakeHome(t)
	t.Chdir(t.TempDir())
	_, settings, own := lifecyclePaths(t, LocationGlobal)

	lifecycleInstall(t, LocationGlobal, lifecycleExecA, PreToolNudgeOn)

	var decoded map[string]any
	if err := json.Unmarshal([]byte(readFile(t, settings)), &decoded); err != nil {
		t.Fatalf("decode settings: %v", err)
	}
	hooks := decoded["hooks"].(map[string]any)
	blocks := hooks["PreToolUse"].([]any)
	if len(blocks) != 6 {
		t.Fatalf("after the first On install PreToolUse has %d blocks, want 6 owned: %#v", len(blocks), blocks)
	}
	edited := -1
	for i, b := range blocks {
		block := b.(map[string]any)
		handlers := block["hooks"].([]any)
		if len(handlers) != 1 {
			t.Fatalf("own block %d has %d handlers, want 1: %#v", i, len(handlers), block)
		}
		handler := handlers[0].(map[string]any)
		if block["matcher"] == "Bash" && handler["if"] == "Bash(rg *)" && handler["command"] == own {
			handler["command"] = own + " --edited"
			edited = i
			break
		}
	}
	if edited < 0 {
		t.Fatalf("no codegraph Bash(rg *) handler to hand-edit: %#v", blocks)
	}
	editedBlock, err := normalizeJSON(blocks[edited])
	if err != nil {
		t.Fatalf("normalize edited block: %v", err)
	}
	out, err := json.MarshalIndent(decoded, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	writeFile(t, settings, string(out)+"\n")

	lifecycleInstall(t, LocationGlobal, lifecycleExecA, PreToolNudgeOn)

	after := preToolUseBlocksAt(t, settings)
	if len(after) != 7 {
		t.Fatalf("PreToolUse has %d blocks, want 7 (the hand-edited one + 6 fresh owned): %#v", len(after), after)
	}
	wantOwn, _, err := claudePreToolUseBlocks(LocationGlobal)
	if err != nil {
		t.Fatalf("claudePreToolUseBlocks: %v", err)
	}
	normalizedOwn, err := normalizeJSON(wantOwn)
	if err != nil {
		t.Fatalf("normalize own blocks: %v", err)
	}
	sawEdited := 0
	for _, b := range after {
		if jsonDeepEqual(b, editedBlock) {
			sawEdited++
		}
	}
	if sawEdited != 1 {
		t.Fatalf("the hand-edited block appears %d times byte-identical, want 1: %#v", sawEdited, after)
	}
	for i, w := range normalizedOwn.([]any) {
		found := false
		for _, b := range after {
			if jsonDeepEqual(b, w) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("own block %d %#v missing after reinstall: %#v", i, w, after)
		}
	}
	if n := countOwnPreToolHandlers(t, settings, own); n != 6 {
		t.Fatalf("own PreToolUse handlers = %d, want a fresh owned set of 6", n)
	}
}

func TestPreToolNudge_ClaudeUninstallDropsKeysFromSharedManifest(t *testing.T) {
	symlinkedClaudeLayout(t)
	t.Chdir(t.TempDir())

	lifecycleInstall(t, LocationGlobal, lifecycleExecA, PreToolNudgeOn)
	sharedDir, err := sharedSkillDirPath(LocationGlobal)
	if err != nil {
		t.Fatalf("sharedSkillDirPath: %v", err)
	}
	var shared WriteResult
	installSkillPackage(&shared, sharedDir, LocationGlobal, Cursor, refuseUnmanifested)
	if len(shared.Errors) != 0 {
		t.Fatalf("installSkillPackage(cursor) errors: %v", shared.Errors)
	}
	manifestPath := skillManifestPath(sharedDir)
	m, present, err := readManifest(manifestPath)
	if err != nil || !present {
		t.Fatalf("readManifest before uninstall: present=%v err=%v", present, err)
	}
	for _, k := range []string{manifestKeyPreToolGuard, manifestKeyPreToolFrag} {
		if _, ok := m.Files[k]; !ok {
			t.Fatalf("precondition: the shared manifest does not record %q: %#v", k, m.Files)
		}
	}

	res := claudeTarget{}.Uninstall(LocationGlobal)
	if len(res.Errors) != 0 {
		t.Fatalf("claude Uninstall errors: %v", res.Errors)
	}
	m, present, err = readManifest(manifestPath)
	if err != nil || !present {
		t.Fatalf("readManifest after uninstall: present=%v err=%v", present, err)
	}
	if !targetSetEqual(m.Targets, []TargetID{Cursor}) {
		t.Fatalf("manifest Targets = %v, want [cursor]", m.Targets)
	}
	if len(m.Files) != 1 {
		t.Fatalf("manifest Files = %#v, want exactly the SKILL.md key", m.Files)
	}
	if _, ok := m.Files[manifestKeySkillMD]; !ok {
		t.Fatalf("manifest Files missing %q: %#v", manifestKeySkillMD, m.Files)
	}
}
