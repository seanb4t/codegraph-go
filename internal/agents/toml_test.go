package agents

import (
	"errors"
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

// D-07/CR-01 (07-REVIEW.md): a UTF-8 BOM (U+FEFF, bytes EF BB BF) at the
// absolute start of a Codex config.toml must never defeat header
// recognition. This is the common real-world shape a Windows-authored
// project-local .codex/config.toml takes: per 07-LIVE-SESSIONS.md's
// scaffold, such a file commonly holds ONLY the [mcp_servers.*] tables,
// with no preceding model =/approval_policy = prelude, so the BOM sits
// directly in front of codegraph's own header line.
const tomlUTF8BOM = "\xef\xbb\xbf"

func TestFindTOMLTableRange_UTF8BOM(t *testing.T) {
	content := tomlUTF8BOM + "[mcp_servers.codegraph]\n" +
		`command = "/old/codegraph"` + "\n" +
		`args = ["serve", "--mcp"]` + "\n"

	_, _, found := findTOMLTableRange(content, "mcp_servers.codegraph")
	if !found {
		t.Fatalf("findTOMLTableRange: found = false, want true (a leading BOM must not defeat header recognition)")
	}
}

func TestSpliceTOMLTable_UTF8BOM_UpdatesInPlaceNeverDuplicates(t *testing.T) {
	content := tomlUTF8BOM + "[mcp_servers.codegraph]\n" +
		`command = "/old/codegraph"` + "\n" +
		`args = ["serve", "--mcp"]` + "\n"

	got := spliceTOMLTable(content, "mcp_servers.codegraph", codexBody())

	if n := strings.Count(got, "[mcp_servers.codegraph]"); n != 1 {
		t.Fatalf("spliceTOMLTable on a BOM'd file produced %d [mcp_servers.codegraph] headers, want exactly 1 (CR-01 duplicate-table regression):\ngot=%q", n, got)
	}
	if !strings.HasPrefix(got, tomlUTF8BOM) {
		t.Fatalf("spliceTOMLTable must preserve the input's leading BOM byte-for-byte:\ngot=%q", got)
	}
	if strings.Contains(got, "/old/codegraph") {
		t.Fatalf("old command should have been replaced in place, not left alongside a duplicate: %s", got)
	}
}

func TestStripTOMLTable_UTF8BOM_RemovesEntirelyPreservingBOM(t *testing.T) {
	installed := tomlUTF8BOM + "[mcp_servers.codegraph]\n" +
		`command = "/usr/local/bin/codegraph"` + "\n" +
		`args = ["serve", "--mcp"]` + "\n"

	got := stripTOMLTable(installed, "mcp_servers.codegraph")

	if strings.Contains(got, "[mcp_servers.codegraph]") {
		t.Fatalf("stripTOMLTable must remove codegraph's own table from a BOM'd file, got: %q", got)
	}
	if !strings.HasPrefix(got, tomlUTF8BOM) {
		t.Fatalf("stripTOMLTable must preserve the input's leading BOM byte-for-byte, got: %q", got)
	}
}

// TestFindTOMLTableRange_MidFileBOMNotStrippedAsWhitespace pins the other
// half of CR-01's fix: only content's true, absolute leading BOM (offset 0
// of the whole file) is special-cased. A BOM byte sequence that appears on
// any later line must never be treated as leading whitespace to strip
// before recognizing a header — this is pathological, constructed input,
// never a real TOML file, but it proves the fix isn't a blanket per-line
// BOM trim.
func TestFindTOMLTableRange_MidFileBOMNotStrippedAsWhitespace(t *testing.T) {
	content := "[a]\nkey = 1\n" + tomlUTF8BOM + "[mcp_servers.codegraph]\ncommand = \"/x\"\n"

	_, _, found := findTOMLTableRange(content, "mcp_servers.codegraph")
	if found {
		t.Fatalf("findTOMLTableRange found a header on a BOM-prefixed non-first line, want false: a mid-file BOM must not be treated as leading whitespace")
	}
}

// TestSpliceTOMLTable_UTF8BOM_Fixture and TestStripTOMLTable_UTF8BOM_Fixture
// pin the same regression by fixture rather than only inline strings (D-07's
// established convention for the maintainer-indented-layout case above), so
// the round-trip is asserted byte-exact against testdata rather than only
// via substring checks.
func TestSpliceTOMLTable_UTF8BOM_Fixture(t *testing.T) {
	input := readTestdata(t, "codex-bom.toml")
	want := readTestdata(t, "codex-bom.installed.toml")

	got := spliceTOMLTable(input, "mcp_servers.codegraph", codexBody())
	if got != want {
		t.Fatalf("spliceTOMLTable BOM-fixture mismatch:\ngot=%q\nwant=%q", got, want)
	}
}

func TestStripTOMLTable_UTF8BOM_Fixture(t *testing.T) {
	installed := readTestdata(t, "codex-bom.installed.toml")
	want := readTestdata(t, "codex-bom.uninstalled.toml")

	got := stripTOMLTable(installed, "mcp_servers.codegraph")
	if got != want {
		t.Fatalf("stripTOMLTable BOM-fixture mismatch:\ngot=%q\nwant=%q", got, want)
	}
}

// D-07: the remaining cases — a header-like "[" line inside a multi-line
// basic/literal string or an unfinished multi-line array must never end
// codegraph's range.
func TestFindTOMLTableRange_HeaderLikeLinesInsideMultilineStringsAndArrays(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name: "basic_ml_string",
			content: "[mcp_servers.codegraph]\n" +
				"description = \"\"\"\n[not a header]\n\"\"\"\n\n" +
				"[some_other_table]\n" + `key = "value"` + "\n",
		},
		{
			name: "literal_ml_string",
			content: "[mcp_servers.codegraph]\n" +
				"description = '''\n[not a header]\n'''\n\n" +
				"[some_other_table]\n" + `key = "value"` + "\n",
		},
		{
			name: "multiline_array",
			content: "[mcp_servers.codegraph]\n" +
				"rows = [\n[1, 2],\n[3, 4],\n]\n\n" +
				"[some_other_table]\n" + `key = "value"` + "\n",
		},
		{
			name: "nested_array_element",
			content: "[mcp_servers.codegraph]\n" +
				"grid = [\n[\n[1, 2],\n],\n]\n\n" +
				"[some_other_table]\n" + `key = "value"` + "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, end, found := findTOMLTableRange(tt.content, "mcp_servers.codegraph")
			if !found {
				t.Fatalf("findTOMLTableRange: found = false, want true")
			}
			after := strings.TrimLeft(tt.content[end:], "\n")
			if !strings.HasPrefix(after, "[some_other_table]") {
				t.Fatalf("range end landed inside/before the multi-line construct instead of at [some_other_table]: content[end:]=%q", tt.content[end:])
			}
		})
	}
}

func TestFindTOMLTableRange_OwnSubtablesInRange(t *testing.T) {
	content := "[mcp_servers.codegraph]\n" +
		`command = "/old/codegraph"` + "\n\n" +
		"[mcp_servers.codegraph.env]\n" +
		`FOO = "bar"` + "\n\n" +
		"[mcp_servers.codegraph_other]\n" +
		`key = "value"` + "\n"

	_, end, found := findTOMLTableRange(content, "mcp_servers.codegraph")
	if !found {
		t.Fatalf("findTOMLTableRange: found = false, want true")
	}
	if got := strings.TrimLeft(content[end:], "\n"); !strings.HasPrefix(got, "[mcp_servers.codegraph_other]") {
		t.Fatalf("range end = %q, want to reach [mcp_servers.codegraph_other] (own subtable must stay in range, prefix-without-dot table must end it)", got)
	}

	spliced := spliceTOMLTable(content, "mcp_servers.codegraph", codexBody())
	if strings.Contains(spliced, "mcp_servers.codegraph.env") {
		t.Fatalf("splice must drop the own subtable along with the old body: %s", spliced)
	}
	if !strings.Contains(spliced, "[mcp_servers.codegraph_other]") {
		t.Fatalf("splice must preserve the following unrelated table: %s", spliced)
	}

	stripped := stripTOMLTable(content, "mcp_servers.codegraph")
	if strings.Contains(stripped, "mcp_servers.codegraph.env") {
		t.Fatalf("strip must remove the own subtable: %s", stripped)
	}
	if !strings.Contains(stripped, "[mcp_servers.codegraph_other]") {
		t.Fatalf("strip must preserve the following unrelated table: %s", stripped)
	}
}

func TestFindTOMLTableRange_TrailingCommentsStayWithNextTable(t *testing.T) {
	t.Run("before_next_header", func(t *testing.T) {
		content := "[mcp_servers.codegraph]\n" +
			`command = "/old/codegraph"` + "\n\n" +
			"# a user comment\n" +
			"[some_other_table]\n" + `key = "value"` + "\n"

		_, end, found := findTOMLTableRange(content, "mcp_servers.codegraph")
		if !found {
			t.Fatalf("findTOMLTableRange: found = false, want true")
		}
		if after := content[end:]; !strings.Contains(after, "# a user comment") {
			t.Fatalf("range end swallowed the preceding comment: %q", after)
		}
		if stripped := stripTOMLTable(content, "mcp_servers.codegraph"); !strings.Contains(stripped, "# a user comment") {
			t.Fatalf("strip must keep the comment: %s", stripped)
		}
	})

	t.Run("before_eof", func(t *testing.T) {
		content := "[mcp_servers.codegraph]\n" +
			`command = "/old/codegraph"` + "\n\n" +
			"# trailing comment\n"

		_, end, found := findTOMLTableRange(content, "mcp_servers.codegraph")
		if !found {
			t.Fatalf("findTOMLTableRange: found = false, want true")
		}
		if got := content[end:]; !strings.Contains(got, "# trailing comment") {
			t.Fatalf("range end swallowed the trailing EOF comment: %q", got)
		}
	})
}

func TestFindTOMLTableRange_HeaderWithTrailingComment(t *testing.T) {
	content := "[mcp_servers.codegraph] # managed\n" + `command = "/old/codegraph"` + "\n"

	start, _, found := findTOMLTableRange(content, "mcp_servers.codegraph")
	if !found {
		t.Fatalf("findTOMLTableRange: found = false, want true for a header with a trailing comment")
	}
	if start != 0 {
		t.Fatalf("start = %d, want 0", start)
	}
}

// assertNoLoneLF fails the test if s contains an LF byte not immediately
// preceded by a CR, i.e. content that mixes bare "\n" into an otherwise
// CRLF file.
func assertNoLoneLF(t *testing.T, s string) {
	t.Helper()
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' && (i == 0 || s[i-1] != '\r') {
			t.Fatalf("found LF not preceded by CR at byte %d:\n%q", i, s)
		}
	}
}

func TestSpliceTOMLTable_CRLFPreserved(t *testing.T) {
	t.Run("append", func(t *testing.T) {
		crlf := strings.ReplaceAll(tomlUnrelatedTable, "\n", "\r\n")
		got := spliceTOMLTable(crlf, "mcp_servers.codegraph", codexBody())
		assertNoLoneLF(t, got)
		if !strings.Contains(got, "[mcp_servers.codegraph]\r\n") {
			t.Fatalf("appended header is not CRLF-terminated: %q", got)
		}
	})

	t.Run("replace", func(t *testing.T) {
		pre := "[mcp_servers.codegraph]\n" +
			`command = "/old/codegraph"` + "\n" +
			`args = ["serve", "--mcp"]` + "\n\n" +
			tomlUnrelatedTable
		crlf := strings.ReplaceAll(pre, "\n", "\r\n")
		got := spliceTOMLTable(crlf, "mcp_servers.codegraph", codexBody())
		assertNoLoneLF(t, got)
	})
}

func TestStripTOMLTable_CRLFRoundTrip(t *testing.T) {
	crlf := strings.ReplaceAll(tomlUnrelatedTable, "\n", "\r\n")
	spliced := spliceTOMLTable(crlf, "mcp_servers.codegraph", codexBody())
	stripped := stripTOMLTable(spliced, "mcp_servers.codegraph")
	if stripped != crlf {
		t.Fatalf("stripTOMLTable(splice(x)) != x for CRLF input:\ngot=%q\nwant=%q", stripped, crlf)
	}
}

// tomlConflictFixtures is shared by TestTOMLTableConflict and the two
// ConflictLeavesContentUnchanged tests, so the same inputs prove both
// "reports the conflict" and "leaves content untouched."
func tomlConflictFixtures() []struct {
	name         string
	content      string
	wantConflict bool
} {
	return []struct {
		name         string
		content      string
		wantConflict bool
	}{
		{
			name:         "inline_table_under_parent",
			content:      "[mcp_servers]\n" + `codegraph = { command = "/x" }` + "\n",
			wantConflict: true,
		},
		{
			name:         "dotted_key_at_root",
			content:      `mcp_servers.codegraph.command = "/x"` + "\n",
			wantConflict: true,
		},
		{
			name:         "dotted_key_under_parent",
			content:      "[mcp_servers]\n" + `codegraph.command = "/x"` + "\n",
			wantConflict: true,
		},
		{
			name:         "quoted_header",
			content:      `["mcp_servers"."codegraph"]` + "\n" + `command = "/x"` + "\n",
			wantConflict: true,
		},
		{
			name:         "spaced_header",
			content:      "[mcp_servers . codegraph]\n" + `command = "/x"` + "\n",
			wantConflict: true,
		},
		{
			name:         "array_of_tables",
			content:      "[[mcp_servers.codegraph]]\n" + `command = "/x"` + "\n",
			wantConflict: true,
		},
		{
			name: "duplicate_header",
			content: "[mcp_servers.codegraph]\n" + `command = "/a"` + "\n\n" +
				"[mcp_servers.codegraph]\n" + `command = "/b"` + "\n",
			wantConflict: true,
		},
		{
			name: "detached_own_subtable",
			content: "[mcp_servers.codegraph.env]\n" + `FOO = "bar"` + "\n\n" +
				"[some_other_table]\n" + `key = "x"` + "\n\n" +
				"[mcp_servers.codegraph]\n" + `command = "/x"` + "\n",
			wantConflict: true,
		},
		{
			name: "clean_own_table",
			content: "[mcp_servers.codegraph]\n" + `command = "/x"` + "\n\n" +
				"[mcp_servers.codegraph.env]\n" + `FOO = "bar"` + "\n",
			wantConflict: false,
		},
		{
			name:         "prefix_lookalike_table",
			content:      "[mcp_servers.codegraph_other]\n" + `key = "value"` + "\n",
			wantConflict: false,
		},
		{
			name:         "unrelated_key_named_codegraph",
			content:      "[other]\n" + `codegraph_path = "x"` + "\n",
			wantConflict: false,
		},
		{
			name:         "quoted_project_trust_table",
			content:      `[projects."/tmp/a.b/repo"]` + "\n" + `trust_level = "trusted"` + "\n",
			wantConflict: false,
		},
	}
}

func TestTOMLTableConflict(t *testing.T) {
	for _, tt := range tomlConflictFixtures() {
		t.Run(tt.name, func(t *testing.T) {
			err := tomlTableConflict(tt.content, "mcp_servers.codegraph")
			if tt.wantConflict {
				if err == nil {
					t.Fatalf("tomlTableConflict(%q) = nil, want a conflict error", tt.content)
				}
				if !errors.Is(err, errTOMLTableConflict) {
					t.Fatalf("tomlTableConflict error does not wrap errTOMLTableConflict: %v", err)
				}
			} else if err != nil {
				t.Fatalf("tomlTableConflict(%q) = %v, want nil", tt.content, err)
			}
		})
	}
}

func TestSpliceTOMLTable_ConflictLeavesContentUnchanged(t *testing.T) {
	for _, tt := range tomlConflictFixtures() {
		if !tt.wantConflict {
			continue
		}
		t.Run(tt.name, func(t *testing.T) {
			got := spliceTOMLTable(tt.content, "mcp_servers.codegraph", codexBody())
			if got != tt.content {
				t.Fatalf("spliceTOMLTable on a conflicting input must leave content unchanged:\ngot=%q\nwant=%q", got, tt.content)
			}
		})
	}
}

func TestStripTOMLTable_ConflictLeavesContentUnchanged(t *testing.T) {
	for _, tt := range tomlConflictFixtures() {
		if !tt.wantConflict {
			continue
		}
		t.Run(tt.name, func(t *testing.T) {
			got := stripTOMLTable(tt.content, "mcp_servers.codegraph")
			if got != tt.content {
				t.Fatalf("stripTOMLTable on a conflicting input must leave content unchanged:\ngot=%q\nwant=%q", got, tt.content)
			}
		})
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

// TestTOMLBoolSetting (D-18) pins tomlBoolSetting's three recognized forms
// — a plain in-table key, a root dotted key, and a single-line inline
// table — plus the shapes it must correctly report as unset rather than
// guess at.
func TestTOMLBoolSetting(t *testing.T) {
	cases := []struct {
		name      string
		content   string
		table     string
		key       string
		wantValue bool
		wantSet   bool
	}{
		{
			name:      "table_false",
			content:   "[features]\nhooks = false\n",
			table:     "features",
			key:       "hooks",
			wantValue: false,
			wantSet:   true,
		},
		{
			name:      "table_true",
			content:   "[features]\nhooks = true\n",
			table:     "features",
			key:       "hooks",
			wantValue: true,
			wantSet:   true,
		},
		{
			name:      "table_indented_header",
			content:   "  [features]\n  hooks = false\n",
			table:     "features",
			key:       "hooks",
			wantValue: false,
			wantSet:   true,
		},
		{
			name:      "trailing_comment",
			content:   "[features]\nhooks = false  # disabled on purpose\n",
			table:     "features",
			key:       "hooks",
			wantValue: false,
			wantSet:   true,
		},
		{
			name:      "dotted_root",
			content:   "features.hooks = false\n",
			table:     "features",
			key:       "hooks",
			wantValue: false,
			wantSet:   true,
		},
		{
			name:      "inline_table",
			content:   "features = { hooks = false, other = true }\n",
			table:     "features",
			key:       "hooks",
			wantValue: false,
			wantSet:   true,
		},
		{
			name:    "not_bool_is_unset",
			content: "[features]\nhooks = \"false\"\n",
			table:   "features",
			key:     "hooks",
			wantSet: false,
		},
		{
			name:    "absent_is_unset",
			content: "[other]\nkey = \"value\"\n",
			table:   "features",
			key:     "hooks",
			wantSet: false,
		},
		{
			name:    "key_in_other_table_is_unset",
			content: "[other]\nhooks = false\n",
			table:   "features",
			key:     "hooks",
			wantSet: false,
		},
		{
			name: "inside_multiline_string_is_unset",
			content: "[features]\n" +
				"notes = \"\"\"\n" +
				"hooks = false\n" +
				"\"\"\"\n",
			table:   "features",
			key:     "hooks",
			wantSet: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotValue, gotSet := tomlBoolSetting(tc.content, tc.table, tc.key)
			if gotSet != tc.wantSet {
				t.Fatalf("tomlBoolSetting(%q, %q, %q) set = %v, want %v", tc.content, tc.table, tc.key, gotSet, tc.wantSet)
			}
			if gotSet && gotValue != tc.wantValue {
				t.Fatalf("tomlBoolSetting(%q, %q, %q) value = %v, want %v", tc.content, tc.table, tc.key, gotValue, tc.wantValue)
			}
		})
	}
}
