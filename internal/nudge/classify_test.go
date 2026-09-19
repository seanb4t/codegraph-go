package nudge

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// corpusRow is one D-15 corpus entry. The corpora are harness-neutral: Tool
// names a nudge.Tool family, never a harness's own tool name, so CODEX-05
// can drive the same files through its adapter.
type corpusRow struct {
	Tool  string `json:"tool"`
	Input string `json:"input"`
	Want  bool   `json:"want"`
	Note  string `json:"note"`
}

// corpus names a D-15 fixture and the class its rows belong to: classWant is
// the Qualifies result the ideal classifier would give every row. A row whose
// want differs from classWant is an accepted miss or accepted false positive
// and must say why in its note.
type corpus struct {
	name      string
	classWant bool
}

var corpora = []corpus{
	{name: "true-positives", classWant: true},
	{name: "false-positives", classWant: false},
}

func loadCorpus(t *testing.T, name string) []corpusRow {
	t.Helper()
	path := filepath.Join("testdata", name+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var rows []corpusRow
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return rows
}

// TestQualifiesCorpora drives every D-15 row through Qualifies as its own
// subtest.
func TestQualifiesCorpora(t *testing.T) {
	for _, c := range corpora {
		for i, row := range loadCorpus(t, c.name) {
			t.Run(fmt.Sprintf("%s/%d", c.name, i), func(t *testing.T) {
				if got := Qualifies(Tool(row.Tool), row.Input); got != row.Want {
					t.Errorf("Qualifies(%q, %q) = %v, want %v (note: %q)", row.Tool, row.Input, got, row.Want, row.Note)
				}
			})
		}
	}
}

// TestCorporaShape is the corpora's positive floor: enough rows, every tool
// family covered, disagreeing rows explained, and the measured fire rates
// logged (NUDGE-05's unit-level half; the live fire rate is plan 06-06).
func TestCorporaShape(t *testing.T) {
	const minRows, minInClass = 14, 12
	validTools := map[string]bool{
		string(ToolShell): true, string(ToolGrep): true, string(ToolGlob): true, string(ToolRead): true,
		"webfetch": true, // the unknown-tool row
	}
	required := map[string][]Tool{
		"true-positives":  {ToolShell, ToolGrep, ToolGlob, ToolRead},
		"false-positives": {ToolShell, ToolRead},
	}

	fired := map[string]int{}
	total := map[string]int{}
	for _, c := range corpora {
		rows := loadCorpus(t, c.name)
		total[c.name] = len(rows)
		if len(rows) < minRows {
			t.Errorf("%s: %d rows, want at least %d", c.name, len(rows), minRows)
		}
		seen := map[string]bool{}
		inClass := 0
		for i, row := range rows {
			if !validTools[row.Tool] {
				t.Errorf("%s/%d: tool %q is not a harness-neutral tool family", c.name, i, row.Tool)
			}
			seen[row.Tool] = true
			if row.Want == c.classWant {
				inClass++
			} else if row.Note == "" {
				t.Errorf("%s/%d: want=%v disagrees with the corpus class but carries no note naming why (%q)", c.name, i, row.Want, row.Input)
			}
			if Qualifies(Tool(row.Tool), row.Input) {
				fired[c.name]++
			}
		}
		for _, tool := range required[c.name] {
			if !seen[string(tool)] {
				t.Errorf("%s: no row covers tool %q", c.name, tool)
			}
		}
		if inClass < minInClass {
			t.Errorf("%s: %d rows want %v, want at least %d", c.name, inClass, c.classWant, minInClass)
		}
	}
	t.Logf("true-positive corpus: %d/%d fire; false-positive corpus: %d/%d fire (accepted, noted)",
		fired["true-positives"], total["true-positives"], fired["false-positives"], total["false-positives"])
}

// TestQualifiesShellFirstWord pins D-02: only the first word after leading
// NAME=value assignments counts, and any parse doubt is silence.
func TestQualifiesShellFirstWord(t *testing.T) {
	cases := []struct {
		input string
		want  bool
	}{
		{"grep x", true},
		{"  rg x", true},
		{"find . -name x", true},
		{"egrep x", true},
		{"fgrep x", true},
		{"LC_ALL=C grep x", true},
		{"A=1 B=2 rg x", true},
		{"grep x | head", true},
		{"", false},
		{"   ", false},
		{"A=1", false},
		{"git log | grep x", false},
		{"cd d && grep x", false},
		{"sudo find /", false},
		{`"grep" x`, false},
		{"$(which grep) x", false},
		{`FOO="a b" grep x`, false},
		{"X=$(pwd) grep x", false},
		{"GREP x", false},
		{"grep;ls", false},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%q", tc.input), func(t *testing.T) {
			if got := Qualifies(ToolShell, tc.input); got != tc.want {
				t.Errorf("Qualifies(shell, %q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

// TestQualifiesReadNonCodeExtensions pins D-03's Read exclusion list.
func TestQualifiesReadNonCodeExtensions(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"/repo/a.md", false},
		{"/repo/A.MD", false},
		{"/repo/a.json", false},
		{"/repo/a.yaml", false},
		{"/repo/a.yml", false},
		{"/repo/a.toml", false},
		{"/repo/a.txt", false},
		{"/repo/a.lock", false},
		{"/repo/a.go", true},
		{"/repo/a.tsx", true},
		{"/repo/a.py", true},
		{"/repo/Makefile", true},
		{"/repo/a.mdx", true},
		{"/repo/a.json5", true},
		{"", false},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%q", tc.path), func(t *testing.T) {
			if got := Qualifies(ToolRead, tc.path); got != tc.want {
				t.Errorf("Qualifies(read, %q) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}
