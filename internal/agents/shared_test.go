package agents

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const (
	testStartMarker = "<!-- CODEGRAPH_START -->"
	testEndMarker   = "<!-- CODEGRAPH_END -->"
)

func testBody(content string) string {
	return testStartMarker + "\n" + content + "\n" + testEndMarker
}

// --- readJSONFile (T-06-01, V5 defensive parse) ---

func TestReadJSONFile_MissingFileReturnsEmptyMap(t *testing.T) {
	dir := t.TempDir()
	got, err := readJSONFile(filepath.Join(dir, "missing.json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("want empty map, got %v", got)
	}
}

func TestReadJSONFile_EmptyFileReturnsEmptyMap(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.json")
	writeFile(t, path, "")

	got, err := readJSONFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("want empty map, got %v", got)
	}
}

func TestReadJSONFile_MalformedReturnsEmptyMapNoPanic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "malformed.json")
	writeFile(t, path, "{not valid json, definitely not")

	got, err := readJSONFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("want empty map on malformed input, got %v", got)
	}
}

// --- replaceOrAppendMarkedSection (marker upsert primitive) ---

func TestMarkerSection_CreatesOnMissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")
	body := testBody("reach for codegraph_explore")

	action, err := replaceOrAppendMarkedSection(path, body, testStartMarker, testEndMarker)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != ActionCreated {
		t.Fatalf("want ActionCreated, got %s", action)
	}
	got := readFile(t, path)
	if got != body+"\n" {
		t.Fatalf("unexpected content:\n%q", got)
	}
}

func TestMarkerSection_UnchangedWhenIdentical(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")
	body := testBody("reach for codegraph_explore")
	writeFile(t, path, body+"\n")

	before := readFile(t, path)
	action, err := replaceOrAppendMarkedSection(path, body, testStartMarker, testEndMarker)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != ActionUnchanged {
		t.Fatalf("want ActionUnchanged, got %s", action)
	}
	after := readFile(t, path)
	if before != after {
		t.Fatalf("bytes changed on a no-op re-run:\nbefore=%q\nafter=%q", before, after)
	}
}

func TestMarkerSection_ReplacesInPlacePreservingOutsideContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")
	oldBody := testBody("old instructions v1")
	original := "# My Project Notes\n\nSome user-written content.\n\n" + oldBody + "\n"
	writeFile(t, path, original)

	newBody := testBody("new instructions v2")
	action, err := replaceOrAppendMarkedSection(path, newBody, testStartMarker, testEndMarker)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != ActionUpdated {
		t.Fatalf("want ActionUpdated, got %s", action)
	}
	got := readFile(t, path)
	want := "# My Project Notes\n\nSome user-written content.\n\n" + newBody + "\n"
	if got != want {
		t.Fatalf("outside-marker content not preserved:\ngot=%q\nwant=%q", got, want)
	}
}

func TestMarkerSection_AppendsWhenNoMarkersPresent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")
	writeFile(t, path, "# My Project Notes\n\nSome user-written content.\n")

	body := testBody("reach for codegraph_explore")
	action, err := replaceOrAppendMarkedSection(path, body, testStartMarker, testEndMarker)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != ActionUpdated {
		t.Fatalf("want ActionUpdated (append), got %s", action)
	}
	got := readFile(t, path)
	want := "# My Project Notes\n\nSome user-written content.\n\n" + body + "\n"
	if got != want {
		t.Fatalf("append did not preserve existing content:\ngot=%q\nwant=%q", got, want)
	}
}

// --- removeMarkedSection ---

func TestMarkerRemove_NoMarkersLeavesBytesUnchanged(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")
	original := "# My Project Notes\n\nNo codegraph section here.\n"
	writeFile(t, path, original)

	action, err := removeMarkedSection(path, testStartMarker, testEndMarker)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != ActionKept {
		t.Fatalf("want ActionKept, got %s", action)
	}
	got := readFile(t, path)
	if got != original {
		t.Fatalf("bytes changed despite no markers present:\ngot=%q\nwant=%q", got, original)
	}
}

func TestMarkerRemove_MissingFileReturnsNotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing.md")

	action, err := removeMarkedSection(path, testStartMarker, testEndMarker)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != ActionNotFound {
		t.Fatalf("want ActionNotFound, got %s", action)
	}
}

// --- Round-trip byte invariance (T-06-02) ---

func TestRoundTrip_MarkerInsertThenRemoveRestoresPreInsertBytes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")
	preInsert := "# My Project Notes\n\nSome user-written content.\n"
	writeFile(t, path, preInsert)

	body := testBody("reach for codegraph_explore")
	if _, err := replaceOrAppendMarkedSection(path, body, testStartMarker, testEndMarker); err != nil {
		t.Fatalf("insert: unexpected error: %v", err)
	}
	if _, err := removeMarkedSection(path, testStartMarker, testEndMarker); err != nil {
		t.Fatalf("remove: unexpected error: %v", err)
	}

	got := readFile(t, path)
	if got != preInsert {
		t.Fatalf("insert->remove did not restore pre-insert bytes:\ngot=%q\nwant=%q", got, preInsert)
	}
}

func TestRoundTrip_MarkerInsertThenRemoveOnMissingFileDeletesIt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")

	body := testBody("reach for codegraph_explore")
	if _, err := replaceOrAppendMarkedSection(path, body, testStartMarker, testEndMarker); err != nil {
		t.Fatalf("insert: unexpected error: %v", err)
	}
	if _, err := removeMarkedSection(path, testStartMarker, testEndMarker); err != nil {
		t.Fatalf("remove: unexpected error: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("want file removed (it never existed pre-insert), stat err=%v", err)
	}
}

// --- writeMcpEntry (Pattern 2) ---

func TestMcpEntry_PreservesSiblingServerKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")
	writeFile(t, path, `{
  "mcpServers": {
    "other-server": {
      "command": "other-binary",
      "args": ["run"]
    }
  }
}
`)

	entry := map[string]any{
		"type":    "stdio",
		"command": "/usr/local/bin/codegraph",
		"args":    []string{"serve", "--mcp"},
	}
	result, err := writeMcpEntry(path, func() any { return entry })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != ActionCreated {
		t.Fatalf("want ActionCreated, got %s", result.Action)
	}

	got, err := readJSONFile(path)
	if err != nil {
		t.Fatalf("unexpected error re-reading: %v", err)
	}
	mcpServers, _ := got["mcpServers"].(map[string]any)
	if mcpServers == nil {
		t.Fatalf("mcpServers missing after write")
	}
	if _, ok := mcpServers["other-server"]; !ok {
		t.Fatalf("sibling server key was not preserved: %v", mcpServers)
	}
	if _, ok := mcpServers["codegraph"]; !ok {
		t.Fatalf("codegraph entry was not written: %v", mcpServers)
	}
}

func TestMcpEntry_UnchangedOnIdenticalRerun(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")

	entry := map[string]any{
		"type":    "stdio",
		"command": "/usr/local/bin/codegraph",
		"args":    []string{"serve", "--mcp"},
	}
	buildEntry := func() any { return entry }

	if _, err := writeMcpEntry(path, buildEntry); err != nil {
		t.Fatalf("first write: unexpected error: %v", err)
	}
	before := readFile(t, path)

	result, err := writeMcpEntry(path, buildEntry)
	if err != nil {
		t.Fatalf("second write: unexpected error: %v", err)
	}
	if result.Action != ActionUnchanged {
		t.Fatalf("want ActionUnchanged on identical re-run, got %s", result.Action)
	}
	after := readFile(t, path)
	if before != after {
		t.Fatalf("bytes changed on a no-op re-run:\nbefore=%q\nafter=%q", before, after)
	}
}

func TestMcpEntry_UpdatedOnChangedEntry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")

	if _, err := writeMcpEntry(path, func() any {
		return map[string]any{"command": "/old/path/codegraph", "args": []string{"serve", "--mcp"}}
	}); err != nil {
		t.Fatalf("first write: unexpected error: %v", err)
	}

	result, err := writeMcpEntry(path, func() any {
		return map[string]any{"command": "/new/path/codegraph", "args": []string{"serve", "--mcp"}}
	})
	if err != nil {
		t.Fatalf("second write: unexpected error: %v", err)
	}
	if result.Action != ActionUpdated {
		t.Fatalf("want ActionUpdated on a changed entry, got %s", result.Action)
	}
}

// --- Round-trip: writeMcpEntry then removeMcpEntry (D-07 keep-clean) ---

func TestRoundTrip_McpEntryInsertThenRemoveKeepsConfigClean(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")

	if _, err := writeMcpEntry(path, func() any {
		return map[string]any{"command": "/usr/local/bin/codegraph", "args": []string{"serve", "--mcp"}}
	}); err != nil {
		t.Fatalf("write: unexpected error: %v", err)
	}

	result, err := removeMcpEntry(path)
	if err != nil {
		t.Fatalf("remove: unexpected error: %v", err)
	}
	if result.Action != ActionRemoved {
		t.Fatalf("want ActionRemoved, got %s", result.Action)
	}

	got, err := readJSONFile(path)
	if err != nil {
		t.Fatalf("unexpected error re-reading: %v", err)
	}
	if _, ok := got["mcpServers"]; ok {
		t.Fatalf("want empty mcpServers object removed entirely (keep-clean), got %v", got)
	}
}

func TestRoundTrip_McpEntryInsertThenRemovePreservesSiblingServer(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")
	writeFile(t, path, `{
  "mcpServers": {
    "other-server": { "command": "other-binary" }
  }
}
`)

	if _, err := writeMcpEntry(path, func() any {
		return map[string]any{"command": "/usr/local/bin/codegraph"}
	}); err != nil {
		t.Fatalf("write: unexpected error: %v", err)
	}
	if _, err := removeMcpEntry(path); err != nil {
		t.Fatalf("remove: unexpected error: %v", err)
	}

	got, err := readJSONFile(path)
	if err != nil {
		t.Fatalf("unexpected error re-reading: %v", err)
	}
	mcpServers, _ := got["mcpServers"].(map[string]any)
	if mcpServers == nil {
		t.Fatalf("mcpServers should still exist (sibling server present): %v", got)
	}
	if _, ok := mcpServers["codegraph"]; ok {
		t.Fatalf("codegraph entry should have been removed: %v", mcpServers)
	}
	if _, ok := mcpServers["other-server"]; !ok {
		t.Fatalf("sibling server key was not preserved: %v", mcpServers)
	}
}

// --- upsertInstructionsEntry (thin wrapper) ---

func TestSharedUpsertInstructionsEntry_WritesFencedBlock(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")

	result, err := upsertInstructionsEntry(path, testStartMarker, testEndMarker, "reach for codegraph_explore")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != ActionCreated {
		t.Fatalf("want ActionCreated, got %s", result.Action)
	}
	got := readFile(t, path)
	if !strings.Contains(got, testStartMarker) ||
		!strings.Contains(got, "reach for codegraph_explore") ||
		!strings.Contains(got, testEndMarker) {
		t.Fatalf("written content missing expected fenced block: %q", got)
	}
}

// --- atomicWriteFile permission preservation (WR-05) ---

func TestAtomicWriteFile_PreservesExistingFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not meaningful on Windows")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")
	if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
		t.Fatalf("seed file: %v", err)
	}

	if err := atomicWriteFile(path, `{"a":1}`); err != nil {
		t.Fatalf("atomicWriteFile: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o644 {
		t.Fatalf("atomicWriteFile changed permissions of an existing file: got %o, want %o", got, 0o644)
	}
}

func TestAtomicWriteFile_NewFileGetsConventionalDefaultPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not meaningful on Windows")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "new.json")

	if err := atomicWriteFile(path, `{"a":1}`); err != nil {
		t.Fatalf("atomicWriteFile: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o644 {
		t.Fatalf("new file has unexpected permissions: got %o, want %o", got, 0o644)
	}
}

// --- writeJSONFile format (2-space indent + trailing newline) ---

func TestSharedWriteJSONFile_FormatsWithIndentAndTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.json")

	if err := writeJSONFile(path, map[string]any{"a": 1}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := readFile(t, path)
	want := "{\n  \"a\": 1\n}\n"
	if got != want {
		t.Fatalf("unexpected format:\ngot=%q\nwant=%q", got, want)
	}
}

// --- Shared repo-root AGENTS.md (D-11, 07-06): instructionsRequestedElsewhere
// gates codexTarget.Uninstall and opencodeTarget.Uninstall so a shared
// instructions file's marker block survives while ANOTHER registered target
// still declares that path at that location and reports
// Detect(loc).AlreadyConfigured. ---

// assertAgentsMDKept fails the test unless r's Files contains a FileResult
// for path with Action ActionKept — the D-11 "left alone, another agent
// still uses it" outcome.
func assertAgentsMDKept(t *testing.T, r WriteResult, path string) {
	t.Helper()
	for _, f := range r.Files {
		if f.Path == path && f.Action == ActionKept {
			return
		}
	}
	t.Fatalf("expected %s kept (ActionKept) in result, got %+v", path, r.Files)
}

// TestSharedAgentsMD_UninstallOrders (D-11) is the full order x pre-state
// table: for each of codex_then_opencode, opencode_then_codex, and
// target_all (every registered target's Uninstall(local), AllTargets
// order), and for each of a pre-existing foreign AGENTS.md and no
// pre-existing file, install codex and opencode at local scope, uninstall
// in the named order, and assert AGENTS.md is restored to its exact
// pre-install state once the last sharer is gone — a two-step order's
// FIRST uninstall must report the file kept while the block is still
// present.
func TestSharedAgentsMD_UninstallOrders(t *testing.T) {
	orders := []string{"codex_then_opencode", "opencode_then_codex", "target_all"}
	pres := []string{"preexisting", "absent"}
	executed := 0
	for _, order := range orders {
		order := order
		for _, pre := range pres {
			pre := pre
			t.Run(order+"/"+pre, func(t *testing.T) {
				executed++
				fakeHome(t)
				dir := t.TempDir()
				t.Chdir(dir)

				agentsPath := filepath.Join(dir, "AGENTS.md")
				var preBytes string
				if pre == "preexisting" {
					preBytes = "# Team Notes\n\nParagraph one, written before codegraph ever ran.\n\n" +
						"Paragraph two, also pre-existing.\n"
					writeFile(t, agentsPath, preBytes)
				}

				codex := codexTarget{}
				opencode := opencodeTarget{}
				opts := InstallOptions{ExecPath: "/usr/local/bin/codegraph"}

				if r := codex.Install(LocationLocal, opts); len(r.Errors) != 0 {
					t.Fatalf("codex install: %v", r.Errors)
				}
				if r := opencode.Install(LocationLocal, opts); len(r.Errors) != 0 {
					t.Fatalf("opencode install: %v", r.Errors)
				}
				if got := readFile(t, agentsPath); !strings.Contains(got, codegraphSectionStart) {
					t.Fatalf("expected codegraph block present after both installs: %q", got)
				}

				switch order {
				case "codex_then_opencode":
					r1 := codex.Uninstall(LocationLocal)
					assertAgentsMDKept(t, r1, agentsPath)
					if got := readFile(t, agentsPath); !strings.Contains(got, codegraphSectionStart) {
						t.Fatalf("block removed after the FIRST uninstall (codex), want kept: %q", got)
					}
					opencode.Uninstall(LocationLocal)
				case "opencode_then_codex":
					r1 := opencode.Uninstall(LocationLocal)
					assertAgentsMDKept(t, r1, agentsPath)
					if got := readFile(t, agentsPath); !strings.Contains(got, codegraphSectionStart) {
						t.Fatalf("block removed after the FIRST uninstall (opencode), want kept: %q", got)
					}
					codex.Uninstall(LocationLocal)
				case "target_all":
					for _, target := range AllTargets() {
						target.Uninstall(LocationLocal)
					}
				}

				if pre == "preexisting" {
					got := readFile(t, agentsPath)
					if got != preBytes {
						t.Fatalf("AGENTS.md not byte-identical to pre-install bytes after the last uninstall:\ngot=%q\nwant=%q", got, preBytes)
					}
				} else if fileExists(agentsPath) {
					t.Fatalf("AGENTS.md should not exist after the last uninstall (never existed pre-install), got=%q", readFile(t, agentsPath))
				}
			})
		}
	}
	if executed != 6 {
		t.Fatalf("executed %d order/pre leaves, want 6", executed)
	}
}

// TestSharedAgentsMD_KeptWhileOtherConfigured (D-11) pins the single-step
// case in isolation: after installing both codex and opencode at local
// scope, uninstalling codex alone reports AGENTS.md kept with a Note
// naming opencode, the file is byte-identical to its pre-uninstall bytes,
// and codex's OWN config table is still removed (the gate only affects the
// shared instructions file, never codex's other steps).
func TestSharedAgentsMD_KeptWhileOtherConfigured(t *testing.T) {
	fakeHome(t)
	dir := t.TempDir()
	t.Chdir(dir)

	codex := codexTarget{}
	opencode := opencodeTarget{}
	opts := InstallOptions{ExecPath: "/usr/local/bin/codegraph"}

	if r := codex.Install(LocationLocal, opts); len(r.Errors) != 0 {
		t.Fatalf("codex install: %v", r.Errors)
	}
	if r := opencode.Install(LocationLocal, opts); len(r.Errors) != 0 {
		t.Fatalf("opencode install: %v", r.Errors)
	}

	agentsPath := filepath.Join(dir, "AGENTS.md")
	preUninstall := readFile(t, agentsPath)

	result := codex.Uninstall(LocationLocal)
	if len(result.Errors) != 0 {
		t.Fatalf("codex uninstall: %v", result.Errors)
	}
	assertAgentsMDKept(t, result, agentsPath)

	foundNote := false
	for _, n := range result.Notes {
		if strings.Contains(n, "opencode") {
			foundNote = true
		}
	}
	if !foundNote {
		t.Fatalf("expected a Note naming opencode, got %v", result.Notes)
	}

	if got := readFile(t, agentsPath); got != preUninstall {
		t.Fatalf("AGENTS.md not byte-identical to its pre-uninstall bytes:\ngot=%q\nwant=%q", got, preUninstall)
	}

	configPath := filepath.Join(dir, ".codex", "config.toml")
	if _, _, found := findTOMLTableRange(readFileOrEmpty(configPath), codexTOMLTable); found {
		t.Fatalf("codex's own config table should still have been removed, got: %s", readFileOrEmpty(configPath))
	}
}

// TestSharedAgentsMD_GlobalPathsNotShared (D-11) confirms the gate is a
// no-op at global scope, where codex and opencode declare DIFFERENT
// instructions paths (~/.codex/AGENTS.md vs <cfgdir>/opencode/AGENTS.md):
// uninstalling codex globally removes its own block even while opencode is
// configured globally too.
func TestSharedAgentsMD_GlobalPathsNotShared(t *testing.T) {
	home := fakeHome(t)

	codex := codexTarget{}
	opencode := opencodeTarget{}
	opts := InstallOptions{ExecPath: "/usr/local/bin/codegraph"}

	if r := codex.Install(LocationGlobal, opts); len(r.Errors) != 0 {
		t.Fatalf("codex install: %v", r.Errors)
	}
	if r := opencode.Install(LocationGlobal, opts); len(r.Errors) != 0 {
		t.Fatalf("opencode install: %v", r.Errors)
	}

	codexInstrPath := filepath.Join(home, ".codex", "AGENTS.md")
	result := codex.Uninstall(LocationGlobal)
	if len(result.Errors) != 0 {
		t.Fatalf("codex uninstall: %v", result.Errors)
	}
	if fileExists(codexInstrPath) {
		t.Fatalf("codex's own global AGENTS.md should have been removed entirely (never pre-existed), got: %s", readFile(t, codexInstrPath))
	}
}

// --- blockOwnsAnyCommand (WR-02, 06-REVIEW.md: shared ownership-identity
// predicate for writeHookEntry, removeHookEntry, and hasOwnHookBlock) ---

func TestBlockOwnsAnyCommand(t *testing.T) {
	own := []string{"codegraph hook pretooluse"}

	t.Run("own command in multi-handler block", func(t *testing.T) {
		block := map[string]any{
			"matcher": "startup",
			"hooks": []any{
				map[string]any{"type": "command", "command": "some-other-command"},
				map[string]any{"type": "command", "command": "codegraph hook pretooluse"},
			},
		}
		if !blockOwnsAnyCommand(block, own) {
			t.Fatal("expected a block containing an own command among multiple hooks to be owned")
		}
	})

	t.Run("foreign-only block is not owned", func(t *testing.T) {
		block := map[string]any{
			"matcher": "startup",
			"hooks": []any{
				map[string]any{"type": "command", "command": "some-other-command"},
			},
		}
		if blockOwnsAnyCommand(block, own) {
			t.Fatal("expected a block with no own command to be reported as not owned")
		}
	})

	t.Run("matcher and if differences are ignored", func(t *testing.T) {
		block := map[string]any{
			"matcher": "totally-different-matcher",
			"if":      "some-condition-codegraph-never-writes",
			"hooks": []any{
				map[string]any{"type": "command", "command": "codegraph hook pretooluse"},
			},
		}
		if !blockOwnsAnyCommand(block, own) {
			t.Fatal("expected ownership to be determined by command identity alone, ignoring matcher/if")
		}
	})
}
