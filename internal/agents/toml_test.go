package agents

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// readTestdata reads a fixture file from internal/agents/testdata/toml,
// failing the test on error.
func readTestdata(t *testing.T, name string) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", "toml", name))
	if err != nil {
		t.Fatalf("readTestdata(%s): %v", name, err)
	}
	return string(data)
}

const tomlUnrelatedTable = `[some_other_table]
key = "value"
nested = ["a", "b"]
`

func codexBody() []string {
	return []string{
		`command = "/usr/local/bin/codegraph"`,
		`args = ["serve", "--mcp"]`,
	}
}

func TestSpliceTOMLTable_AppendsWhenAbsent_PreservesUnrelatedTable(t *testing.T) {
	got := spliceTOMLTable(tomlUnrelatedTable, "mcp_servers.codegraph", codexBody())

	if got == tomlUnrelatedTable {
		t.Fatalf("expected content to change when appending a new table")
	}
	// The unrelated table must appear byte-for-byte unchanged.
	if idx := indexOf(got, tomlUnrelatedTable); idx != 0 {
		t.Fatalf("unrelated table not preserved byte-for-byte at file start:\ngot=%q", got)
	}
	if !containsAll(got, "[mcp_servers.codegraph]", `command = "/usr/local/bin/codegraph"`, `args = ["serve", "--mcp"]`) {
		t.Fatalf("appended table missing expected content: %s", got)
	}
}

func TestSpliceTOMLTable_AppendsIntoEmptyContent(t *testing.T) {
	got := spliceTOMLTable("", "mcp_servers.codegraph", codexBody())
	want := "[mcp_servers.codegraph]\n" + `command = "/usr/local/bin/codegraph"` + "\n" + `args = ["serve", "--mcp"]` + "\n"
	if got != want {
		t.Fatalf("spliceTOMLTable(empty) = %q, want %q", got, want)
	}
}

func TestSpliceTOMLTable_IdenticalContentIsNoOp(t *testing.T) {
	once := spliceTOMLTable(tomlUnrelatedTable, "mcp_servers.codegraph", codexBody())
	twice := spliceTOMLTable(once, "mcp_servers.codegraph", codexBody())
	if once != twice {
		t.Fatalf("splicing identical content should be a byte-identical no-op:\nonce=%q\ntwice=%q", once, twice)
	}
}

func TestSpliceTOMLTable_ReplacesOnlyExistingBlock_PreservesEverythingElse(t *testing.T) {
	pre := "[mcp_servers.codegraph]\n" +
		`command = "/old/codegraph"` + "\n" +
		`args = ["serve", "--mcp"]` + "\n\n" +
		tomlUnrelatedTable

	got := spliceTOMLTable(pre, "mcp_servers.codegraph", codexBody())

	if !containsAll(got, "[some_other_table]", `key = "value"`, `nested = ["a", "b"]`) {
		t.Fatalf("unrelated table lost during replace: %s", got)
	}
	if containsAll(got, "/old/codegraph") {
		t.Fatalf("old command value should have been replaced: %s", got)
	}
	if !containsAll(got, "/usr/local/bin/codegraph") {
		t.Fatalf("new command value missing: %s", got)
	}
}

func TestStripTOMLTable_RemovesOnlyCodegraphBlock_RoundTrip(t *testing.T) {
	spliced := spliceTOMLTable(tomlUnrelatedTable, "mcp_servers.codegraph", codexBody())
	stripped := stripTOMLTable(spliced, "mcp_servers.codegraph")

	if stripped != tomlUnrelatedTable {
		t.Fatalf("stripTOMLTable did not restore pre-splice bytes:\ngot=%q\nwant=%q", stripped, tomlUnrelatedTable)
	}
}

func TestStripTOMLTable_MissingTableIsNoOp(t *testing.T) {
	got := stripTOMLTable(tomlUnrelatedTable, "mcp_servers.codegraph")
	if got != tomlUnrelatedTable {
		t.Fatalf("stripTOMLTable on absent table should be a no-op:\ngot=%q\nwant=%q", got, tomlUnrelatedTable)
	}
}

// D-07: the maintainer's real ~/.codex/config.toml has [mcp_servers.codegraph]
// at 2-space indent with the next column-0 header ([memories]) far below —
// the released-binary data-loss shape (07-CONTEXT.md, blocking finding).
// codex-indented-layout.toml mirrors that shape synthetically; it is never
// derived from the real file.

func TestFindTOMLTableRange_MaintainerIndentedLayout(t *testing.T) {
	content := readTestdata(t, "codex-indented-layout.toml")

	start, end, found := findTOMLTableRange(content, "mcp_servers.codegraph")
	if !found {
		t.Fatalf("findTOMLTableRange: found = false, want true")
	}

	wantHeaderLine := "  [mcp_servers.codegraph]\n"
	if got := content[start:]; !strings.HasPrefix(got, wantHeaderLine) {
		t.Fatalf("content[start:] does not start with the indented header line:\ngot=%q\nwant prefix=%q", got, wantHeaderLine)
	}

	after := content[end:]
	wantAfterPrefix := "\n  [mcp_servers.context7]"
	if !strings.HasPrefix(after, wantAfterPrefix) {
		t.Fatalf("content[end:] = %q, want to begin with %q (the blank line before the next indented sibling header, never [memories])", after, wantAfterPrefix)
	}
}

func TestSpliceTOMLTable_MaintainerIndentedLayout(t *testing.T) {
	input := readTestdata(t, "codex-indented-layout.toml")
	want := readTestdata(t, "codex-indented-layout.installed.toml")

	got := spliceTOMLTable(input, "mcp_servers.codegraph", codexBody())
	if got != want {
		t.Fatalf("spliceTOMLTable maintainer-layout mismatch:\ngot=%q\nwant=%q", got, want)
	}
}

func TestStripTOMLTable_MaintainerIndentedLayout(t *testing.T) {
	installed := readTestdata(t, "codex-indented-layout.installed.toml")
	want := readTestdata(t, "codex-indented-layout.uninstalled.toml")

	got := stripTOMLTable(installed, "mcp_servers.codegraph")
	if got != want {
		t.Fatalf("stripTOMLTable maintainer-layout mismatch:\ngot=%q\nwant=%q", got, want)
	}
}

func TestCodexGlobal_MaintainerIndentedLayoutRoundTrip(t *testing.T) {
	home := fakeHome(t)
	input := readTestdata(t, "codex-indented-layout.toml")
	configPath := filepath.Join(home, ".codex", "config.toml")
	writeFile(t, configPath, input)

	installed := readTestdata(t, "codex-indented-layout.installed.toml")
	installResult := codexTarget{}.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
	if len(installResult.Errors) != 0 {
		t.Fatalf("Install errors: %v", installResult.Errors)
	}
	if got := readFile(t, configPath); got != installed {
		t.Fatalf("post-install config.toml mismatch:\ngot=%q\nwant=%q", got, installed)
	}

	uninstalled := readTestdata(t, "codex-indented-layout.uninstalled.toml")
	uninstallResult := codexTarget{}.Uninstall(LocationGlobal)
	if len(uninstallResult.Errors) != 0 {
		t.Fatalf("Uninstall errors: %v", uninstallResult.Errors)
	}
	if got := readFile(t, configPath); got != uninstalled {
		t.Fatalf("post-uninstall config.toml mismatch:\ngot=%q\nwant=%q", got, uninstalled)
	}
}

// indexOf and containsAll are tiny local helpers to keep these tests
// dependency-free of the strings package's full API surface.
func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}

func containsAll(haystack string, needles ...string) bool {
	for _, n := range needles {
		if indexOf(haystack, n) == -1 {
			return false
		}
	}
	return true
}
