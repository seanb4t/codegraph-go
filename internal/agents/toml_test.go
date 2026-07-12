package agents

import "testing"

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
