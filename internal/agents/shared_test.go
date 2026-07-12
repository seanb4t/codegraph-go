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
