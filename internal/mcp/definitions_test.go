package mcp

import (
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/testtree"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// treeServer writes and indexes files, builds a server with the named
// tools, and returns a function that calls one of them and returns the
// text of the result.
func treeServer(t *testing.T, files map[string]string, tools ...string) func(tool string, args map[string]any) string {
	t.Helper()
	dir := testtree.Write(t, files)
	indexFixture(t, dir)
	enabled := make(map[string]bool, len(tools))
	for _, tool := range tools {
		enabled[tool] = true
	}
	s := BuildServer(true, enabled, dir, dir)
	return func(tool string, args map[string]any) string {
		var b strings.Builder
		for _, c := range callTool(t, s, tool, args).Content {
			if tc, ok := c.(*mcp.TextContent); ok {
				b.WriteString(tc.Text)
			}
		}
		return b.String()
	}
}

// twoRuns is two functions named run. main calls the one in a.py, and
// other calls the one in b.py.
var twoRuns = map[string]string{
	"a.py": "def load():\n    return 1\n\n\ndef run():\n    return load()\n",
	"b.py": "def save():\n    return 2\n\n\ndef run():\n    return save()\n\n\ndef other():\n    return run()\n",
	"m.py": "from a import run\n\n\ndef main():\n    return run()\n",
}

func requireAll(t *testing.T, text string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Errorf("the result lacks %q:\n%s", want, text)
		}
	}
}

// TestCallersToolOnANameWithTwoDefinitions is the scenario "An MCP
// callers call on a name with two definitions".
func TestCallersToolOnANameWithTwoDefinitions(t *testing.T) {
	call := treeServer(t, twoRuns, "callers", "callees", "impact")
	requireAll(t, call("codegraph_callers", map[string]any{"symbol": "run"}), "`main`", "`other`")
}

// TestCallersToolTableHasADefinitionColumn is the scenario "Callers of a
// name with two definitions".
func TestCallersToolTableHasADefinitionColumn(t *testing.T) {
	call := treeServer(t, twoRuns, "callers", "callees", "impact")
	requireAll(t, call("codegraph_callers", map[string]any{"symbol": "run"}),
		"2 definitions named \"run\": a.py:5, b.py:5\n",
		"| Name | Kind | Location | Definition |\n",
		"| `main` | function | `m.py:4` | `a.py:5` |\n",
	)
}

// TestCalleesToolWithAFile is the scenario "An MCP callees call with a
// file".
func TestCalleesToolWithAFile(t *testing.T) {
	call := treeServer(t, twoRuns, "callers", "callees", "impact")
	text := call("codegraph_callees", map[string]any{"symbol": "run", "file": "a.py"})
	requireAll(t, text, "`load`")
	if strings.Contains(text, "`save`") || strings.Contains(text, "definitions named") {
		t.Errorf("callees of run in a.py lists save or the definitions:\n%s", text)
	}
}

// TestCallersToolOfNoCallerListsTheDefinitions is the scenario "No caller
// of a name with two definitions".
func TestCallersToolOfNoCallerListsTheDefinitions(t *testing.T) {
	call := treeServer(t, map[string]string{
		"a.py": "def run():\n    return 1\n",
		"b.py": "def run():\n    return 2\n",
	}, "callers", "callees", "impact")
	requireAll(t, call("codegraph_callers", map[string]any{"symbol": "run"}),
		"no callers found", "2 definitions named \"run\": a.py:1, b.py:1\n")
}

// TestImpactToolRowReachedByTheWalkHasNoDefinition is the scenario "An
// impact row that the walk reached".
func TestImpactToolRowReachedByTheWalkHasNoDefinition(t *testing.T) {
	call := treeServer(t, twoRuns, "callers", "callees", "impact")
	requireAll(t, call("codegraph_impact", map[string]any{"symbol": "run", "depth": 1}),
		"| `run` | function | `a.py:5` | `a.py:5` |\n",
		"| `run` | function | `b.py:5` | `b.py:5` |\n",
		"| `main` | function | `m.py:4` |  |\n",
	)
}

// TestCallersToolTableOfOneDefinitionHasThreeColumns is the scenario
// "Callers of a name with one definition".
func TestCallersToolTableOfOneDefinitionHasThreeColumns(t *testing.T) {
	call := treeServer(t, twoRuns, "callers", "callees", "impact")
	text := call("codegraph_callers", map[string]any{"symbol": "load"})
	requireAll(t, text, "| Name | Kind | Location |\n", "| `run` | function | `a.py:5` |\n")
	if strings.Contains(text, "Definition") || strings.Contains(text, "definitions named") {
		t.Errorf("a name with one definition has a Definition column or a definitions line:\n%s", text)
	}
}

// TestSpanningTableOfAmbiguousRowsHasADefinitionColumn pins that the
// Definition column comes from the result and not from the rows: the
// only caller is ambiguous and names no definition.
func TestSpanningTableOfAmbiguousRowsHasADefinitionColumn(t *testing.T) {
	call := treeServer(t, map[string]string{
		"Sink.java":  "package app;\n\npublic class Sink {\n    public void add(int n) {}\n\n    public void add(String s) {}\n}\n",
		"Other.java": "package app;\n\npublic class Other {\n    public void add() {}\n}\n",
		"C.java":     "package app;\n\npublic class C {\n    public void useC(Sink sink) {\n        sink.add(null);\n    }\n}\n",
	}, "callers", "callees", "impact")
	requireAll(t, call("codegraph_callers", map[string]any{"symbol": "add"}),
		"3 definitions named \"add\": Other.java:4, Sink.java:4, Sink.java:6\n",
		"| Name | Kind | Location | Definition | Overload |\n",
		"| `useC` | method | `C.java:4` |  | ambiguous |\n",
	)
}
