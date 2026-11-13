package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/schema"
)

// setupIndexedFixture copies the shared gofixture (copyFixture,
// cli_test.go) into a fresh temp dir and runs `codegraph init` against it
// via execCmd — mirroring TestInitIndexUninit's own setup — so every
// query-command subtest below runs against a real Pebble-backed index
// rather than a mock (03-PATTERNS.md §"Test scaffolding").
func setupIndexedFixture(t *testing.T) string {
	t.Helper()

	dir := copyFixture(t)
	pruneGOOSSuffixedFiles(t, dir)
	if _, _, err := execCmd("init", dir); err != nil {
		t.Fatalf("init fixture: unexpected error: %v", err)
	}
	return dir
}

// TestSearchFullCmd covers VERB-01/VERB-03's `search --full` behavior — the
// fold target of the old TestQueryCmd, now expressed against `search --full`
// instead of the removed `query` verb.
func TestSearchFullCmd(t *testing.T) {
	dir := setupIndexedFixture(t)

	t.Run("--full --json emits the MarshalQueryJSON envelope", func(t *testing.T) {
		out, _, err := execCmd("search", "main", "-p", dir, "--full", "--json")
		if err != nil {
			t.Fatalf("search --full --json: unexpected error: %v", err)
		}
		if !strings.HasPrefix(out, `[{"node":`) {
			t.Fatalf("search --full --json: expected output to start with %q, got %q", `[{"node":`, out)
		}

		var envelopes []struct {
			Node struct {
				Name string `json:"name"`
				Kind string `json:"kind"`
			} `json:"node"`
		}
		if err := json.Unmarshal([]byte(out), &envelopes); err != nil {
			t.Fatalf("search --full --json: invalid JSON: %v\noutput: %s", err, out)
		}
		found := false
		for _, e := range envelopes {
			if e.Node.Name == "main" {
				found = true
			}
		}
		if !found {
			t.Fatalf("search --full --json: expected a %q match, got %+v", "main", envelopes)
		}
	})

	t.Run("default --json stays the Location array", func(t *testing.T) {
		out, _, err := execCmd("search", "main", "-p", dir, "--json")
		if err != nil {
			t.Fatalf("search --json: unexpected error: %v", err)
		}
		if !strings.HasPrefix(out, `[{"name":`) {
			t.Fatalf("search --json: expected output to start with %q, got %q", `[{"name":`, out)
		}
		if strings.Contains(out, `"node"`) {
			t.Fatalf("search --json: expected no %q key, got %q", "node", out)
		}
	})

	t.Run("--full human branch is two lines per hit (D-01)", func(t *testing.T) {
		defaultOut, _, err := execCmd("search", "Alpha", "-p", dir)
		if err != nil {
			t.Fatalf("search Alpha: unexpected error: %v", err)
		}
		fullOut, _, err := execCmd("search", "Alpha", "-p", dir, "--full")
		if err != nil {
			t.Fatalf("search Alpha --full: unexpected error: %v", err)
		}
		want := defaultOut + "    Alpha  () int\n"
		if fullOut != want {
			t.Fatalf("search Alpha --full = %q, want %q", fullOut, want)
		}
	})

	t.Run("empty Signature renders just the qualified name", func(t *testing.T) {
		out, _, err := execCmd("search", "pkga", "-p", dir, "--full")
		if err != nil {
			t.Fatalf("search pkga --full: unexpected error: %v", err)
		}
		if !strings.Contains(out, "    example.com/gofixture/pkga") {
			t.Fatalf("search pkga --full: expected qualified-name-only line, got %q", out)
		}
		for _, line := range strings.Split(out, "\n") {
			if line != strings.TrimRight(line, " \t") {
				t.Fatalf("search pkga --full: line has trailing whitespace: %q", line)
			}
		}
	})

	t.Run("non-indented lines of --full equal default output", func(t *testing.T) {
		for _, term := range []string{"Alpha", "pkga", "helper"} {
			defaultOut, _, err := execCmd("search", term, "-p", dir)
			if err != nil {
				t.Fatalf("search %s: unexpected error: %v", term, err)
			}
			fullOut, _, err := execCmd("search", term, "-p", dir, "--full")
			if err != nil {
				t.Fatalf("search %s --full: unexpected error: %v", term, err)
			}
			var nonIndented []string
			for _, line := range strings.Split(fullOut, "\n") {
				if line == "" {
					continue
				}
				if !strings.HasPrefix(line, " ") {
					nonIndented = append(nonIndented, line)
				}
			}
			got := strings.Join(nonIndented, "\n")
			if got != "" {
				got += "\n"
			}
			if got != defaultOut {
				t.Fatalf("search %s --full non-indented lines = %q, want default output %q", term, got, defaultOut)
			}
		}
	})

	t.Run("zero hits: --json prints [] and human prints nothing", func(t *testing.T) {
		jsonOut, _, err := execCmd("search", "zzz-no-such-symbol", "-p", dir, "--full", "--json")
		if err != nil {
			t.Fatalf("search zzz --full --json: unexpected error: %v", err)
		}
		if strings.TrimSpace(jsonOut) != "[]" {
			t.Fatalf("search zzz --full --json = %q, want %q", jsonOut, "[]")
		}
		humanOut, _, err := execCmd("search", "zzz-no-such-symbol", "-p", dir, "--full")
		if err != nil {
			t.Fatalf("search zzz --full: unexpected error: %v", err)
		}
		if humanOut != "" {
			t.Fatalf("search zzz --full = %q, want empty", humanOut)
		}
		defaultJSONOut, _, err := execCmd("search", "zzz-no-such-symbol", "-p", dir, "--json")
		if err != nil {
			t.Fatalf("search zzz --json: unexpected error: %v", err)
		}
		if strings.TrimSpace(defaultJSONOut) != "[]" {
			t.Fatalf("search zzz --json = %q, want %q", defaultJSONOut, "[]")
		}
	})

	t.Run("--kind rejects an unknown kind", func(t *testing.T) {
		_, _, err := execCmd("search", "main", "-p", dir, "--full", "--kind", "bogus")
		if err == nil {
			t.Fatal("search --full --kind bogus: expected an error, got nil")
		}
	})

	t.Run("not initialized directory errors", func(t *testing.T) {
		fresh := t.TempDir()
		_, _, err := execCmd("search", "main", "-p", fresh, "--full")
		if err == nil {
			t.Fatal("search --full against uninitialized dir: expected an error, got nil")
		}
	})
}

// TestRenderFullLine pins D-01's exact whitespace for the --full human
// branch's second line: no truncation, escaping, or normalisation of the
// QualifiedName/Signature bytes.
func TestRenderFullLine(t *testing.T) {
	tests := []struct {
		name string
		node *schema.Node
		want string
	}{
		{
			name: "signature present: qualified name and signature separated by two spaces",
			node: &schema.Node{QualifiedName: "a.B", Signature: "(x int) error"},
			want: "    a.B  (x int) error",
		},
		{
			name: "empty signature: qualified name only",
			node: &schema.Node{QualifiedName: "example.com/x", Signature: ""},
			want: "    example.com/x",
		},
		{
			name: "signature bytes verbatim: no truncation, escaping, or normalisation",
			node: &schema.Node{QualifiedName: "q", Signature: "(s string) (résumé, error)"},
			want: "    q  (s string) (résumé, error)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := renderFullLine(tt.node); got != tt.want {
				t.Fatalf("renderFullLine(%+v) = %q, want %q", tt.node, got, tt.want)
			}
		})
	}
}

// TestSearchFlagShortForms covers VERB-02/VERB-04: `search` gains `query`'s
// short flags (-k/-l/-j alongside the existing -p), on both the default and
// --full branches, byte-identical to their long forms.
func TestSearchFlagShortForms(t *testing.T) {
	dir := setupIndexedFixture(t)

	type flagPair struct {
		name  string
		long  []string
		short []string
	}
	pairs := []flagPair{
		{"kind", []string{"--kind", "function"}, []string{"-k", "function"}},
		{"limit", []string{"--limit", "1"}, []string{"-l", "1"}},
		{"json", []string{"--json"}, []string{"-j"}},
		{"path", []string{"--path", dir}, []string{"-p", dir}},
	}

	modes := []struct {
		name string
		flag []string
	}{
		{"default", nil},
		{"full", []string{"--full"}},
	}

	for _, mode := range modes {
		t.Run(mode.name, func(t *testing.T) {
			for _, p := range pairs {
				t.Run(p.name, func(t *testing.T) {
					longArgs := append([]string{"search", "main"}, mode.flag...)
					longArgs = append(longArgs, p.long...)
					shortArgs := append([]string{"search", "main"}, mode.flag...)
					shortArgs = append(shortArgs, p.short...)
					if p.name != "path" {
						longArgs = append(longArgs, "-p", dir)
						shortArgs = append(shortArgs, "-p", dir)
					}

					longOut, _, longErr := execCmd(longArgs...)
					if longErr != nil {
						t.Fatalf("long form %v: unexpected error: %v", longArgs, longErr)
					}
					shortOut, _, shortErr := execCmd(shortArgs...)
					if shortErr != nil {
						t.Fatalf("short form %v: unexpected error: %v", shortArgs, shortErr)
					}
					if longOut == "" {
						t.Fatalf("long form %v: expected non-empty output", longArgs)
					}
					if longOut != shortOut {
						t.Fatalf("long form %v = %q, short form %v = %q: expected byte-identical output", longArgs, longOut, shortArgs, shortOut)
					}
					if p.name == "json" {
						wantPrefix := `[{"name":`
						if mode.name == "full" {
							wantPrefix = `[{"node":`
						}
						if !strings.HasPrefix(longOut, wantPrefix) {
							t.Fatalf("mode=%s json output = %q, want prefix %q", mode.name, longOut, wantPrefix)
						}
					}
				})
			}
		})
	}
}

func TestSearchCmd(t *testing.T) {
	dir := setupIndexedFixture(t)

	out, _, err := execCmd("search", "Alpha", "-p", dir, "--json")
	if err != nil {
		t.Fatalf("search: unexpected error: %v", err)
	}
	var locs []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(out), &locs); err != nil {
		t.Fatalf("search --json: invalid JSON: %v\noutput: %s", err, out)
	}
	if len(locs) == 0 {
		t.Fatalf("search --json: expected at least one match for %q, got none", "Alpha")
	}
}

func TestCallersCalleesCmd(t *testing.T) {
	dir := setupIndexedFixture(t)

	t.Run("callees(Alpha) includes helper", func(t *testing.T) {
		out, _, err := execCmd("callees", "Alpha", "-p", dir, "--json")
		if err != nil {
			t.Fatalf("callees: unexpected error: %v", err)
		}
		var result struct {
			Symbol  string `json:"symbol"`
			Callees []struct {
				Name string `json:"name"`
			} `json:"callees"`
		}
		if err := json.Unmarshal([]byte(out), &result); err != nil {
			t.Fatalf("callees --json: invalid JSON: %v\noutput: %s", err, out)
		}
		found := false
		for _, c := range result.Callees {
			if c.Name == "helper" {
				found = true
			}
		}
		if !found {
			t.Fatalf("callees(Alpha): expected %q among callees, got %+v", "helper", result.Callees)
		}
	})

	t.Run("callers(helper) includes Alpha", func(t *testing.T) {
		out, _, err := execCmd("callers", "helper", "-p", dir, "--json")
		if err != nil {
			t.Fatalf("callers: unexpected error: %v", err)
		}
		var result struct {
			Symbol  string `json:"symbol"`
			Callers []struct {
				Name string `json:"name"`
			} `json:"callers"`
		}
		if err := json.Unmarshal([]byte(out), &result); err != nil {
			t.Fatalf("callers --json: invalid JSON: %v\noutput: %s", err, out)
		}
		found := false
		for _, c := range result.Callers {
			if c.Name == "Alpha" {
				found = true
			}
		}
		if !found {
			t.Fatalf("callers(helper): expected %q among callers, got %+v", "Alpha", result.Callers)
		}
	})
}

func TestImpactCmd(t *testing.T) {
	dir := setupIndexedFixture(t)

	out, _, err := execCmd("impact", "helper", "-p", dir, "--depth", "2", "--json")
	if err != nil {
		t.Fatalf("impact: unexpected error: %v", err)
	}
	var result struct {
		Symbol    string `json:"symbol"`
		Depth     int    `json:"depth"`
		NodeCount int    `json:"nodeCount"`
		EdgeCount int    `json:"edgeCount"`
		Affected  []struct {
			Name string `json:"name"`
		} `json:"affected"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("impact --json: invalid JSON: %v\noutput: %s", err, out)
	}
	if result.Symbol != "helper" {
		t.Fatalf("impact: symbol = %q, want %q", result.Symbol, "helper")
	}
	if result.Depth != 2 {
		t.Fatalf("impact: depth = %d, want 2", result.Depth)
	}
	if result.NodeCount != len(result.Affected) {
		t.Fatalf("impact: nodeCount = %d, want len(affected) = %d", result.NodeCount, len(result.Affected))
	}
	found := false
	for _, n := range result.Affected {
		if n.Name == "Alpha" {
			found = true
		}
	}
	if !found {
		t.Fatalf("impact(helper, depth=2): expected %q among affected, got %+v", "Alpha", result.Affected)
	}
}

func TestAffectedCmd(t *testing.T) {
	dir := setupIndexedFixture(t)

	out, _, err := execCmd("affected", "pkga/pkga.go", "-p", dir, "--json")
	if err != nil {
		t.Fatalf("affected: unexpected error: %v", err)
	}
	var result struct {
		Files         []string `json:"files"`
		AffectedTests []struct {
			Name string `json:"name"`
		} `json:"affectedTests"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("affected --json: invalid JSON: %v\noutput: %s", err, out)
	}
	if len(result.Files) != 1 || result.Files[0] != "pkga/pkga.go" {
		t.Fatalf("affected: files = %+v, want [pkga/pkga.go]", result.Files)
	}
	// gofixture has no _test.go files, so no test symbols are reachable —
	// asserting the shape (files echoed back, an empty/absent
	// affectedTests) rather than fabricating a golden fixture (D-07a).
}

func TestFilesCmd(t *testing.T) {
	dir := setupIndexedFixture(t)

	t.Run("flat format lists indexed files", func(t *testing.T) {
		out, _, err := execCmd("files", "-p", dir, "--json")
		if err != nil {
			t.Fatalf("files: unexpected error: %v", err)
		}
		var result struct {
			Format string `json:"format"`
			Files  []struct {
				Path string `json:"path"`
			} `json:"files"`
		}
		if err := json.Unmarshal([]byte(out), &result); err != nil {
			t.Fatalf("files --json: invalid JSON: %v\noutput: %s", err, out)
		}
		if result.Format != "flat" {
			t.Fatalf("files: format = %q, want %q", result.Format, "flat")
		}
		if len(result.Files) == 0 {
			t.Fatal("files: expected at least one indexed file, got none")
		}
	})

	t.Run("--format tree returns a nested projection", func(t *testing.T) {
		out, _, err := execCmd("files", "-p", dir, "--format", "tree", "--json")
		if err != nil {
			t.Fatalf("files --format tree: unexpected error: %v", err)
		}
		var result struct {
			Format string `json:"format"`
			Tree   []struct {
				Name string `json:"name"`
			} `json:"tree"`
		}
		if err := json.Unmarshal([]byte(out), &result); err != nil {
			t.Fatalf("files --format tree --json: invalid JSON: %v\noutput: %s", err, out)
		}
		if result.Format != "tree" {
			t.Fatalf("files --format tree: format = %q, want %q", result.Format, "tree")
		}
	})
}

func TestNodeCmd(t *testing.T) {
	dir := setupIndexedFixture(t)

	t.Run("symbol detail emits markdown, not JSON", func(t *testing.T) {
		out, _, err := execCmd("node", "Alpha", "-p", dir)
		if err != nil {
			t.Fatalf("node: unexpected error: %v", err)
		}
		if !strings.Contains(out, "**Alpha** (function)") {
			t.Fatalf("node Alpha: expected markdown header, got %q", out)
		}
		if !strings.Contains(out, "**Calls →**") || !strings.Contains(out, "**Called by ←**") {
			t.Fatalf("node Alpha: expected Calls/Called-by trail, got %q", out)
		}
		if strings.HasPrefix(strings.TrimSpace(out), "{") {
			t.Fatalf("node Alpha: expected markdown, got JSON-shaped output: %q", out)
		}
	})

	t.Run("file mode (-f, no symbol) emits a line-numbered read", func(t *testing.T) {
		out, _, err := execCmd("node", "-p", dir, "-f", "pkga/pkga.go")
		if err != nil {
			t.Fatalf("node -f: unexpected error: %v", err)
		}
		if !strings.Contains(out, "```go") {
			t.Fatalf("node -f: expected a fenced code block, got %q", out)
		}
		if !strings.Contains(out, "1\t") {
			t.Fatalf("node -f: expected line-numbered output, got %q", out)
		}
	})

	t.Run("no symbol and no file errors", func(t *testing.T) {
		_, _, err := execCmd("node", "-p", dir)
		if err == nil {
			t.Fatal("node with no symbol and no -f: expected an error, got nil")
		}
	})
}

func TestExploreCmd(t *testing.T) {
	dir := setupIndexedFixture(t)

	out, _, err := execCmd("explore", "Alpha", "-p", dir)
	if err != nil {
		t.Fatalf("explore: unexpected error: %v", err)
	}
	if !strings.Contains(out, "**Exploration: Alpha**") {
		t.Fatalf("explore: expected exploration header, got %q", out)
	}
	if !strings.Contains(out, "**Blast radius") {
		t.Fatalf("explore: expected blast-radius section, got %q", out)
	}
	if !strings.Contains(out, "```go") {
		t.Fatalf("explore: expected fenced verbatim source, got %q", out)
	}
	if strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Fatalf("explore: expected markdown, got JSON-shaped output: %q", out)
	}
}

func TestServeCmdRequiresMCPFlag(t *testing.T) {
	dir := setupIndexedFixture(t)

	_, _, err := execCmd("serve", "-p", dir)
	if err == nil {
		t.Fatal("serve without --mcp: expected an error, got nil")
	}
}

func TestStatusCmd(t *testing.T) {
	dir := setupIndexedFixture(t)

	out, _, err := execCmd("status", "-p", dir, "--json")
	if err != nil {
		t.Fatalf("status: unexpected error: %v", err)
	}
	var result struct {
		Initialized bool   `json:"initialized"`
		Backend     string `json:"backend"`
		NodeCount   int64  `json:"nodeCount"`
		EdgeCount   int64  `json:"edgeCount"`
		Index       struct {
			State string `json:"state"`
		} `json:"index"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("status --json: invalid JSON: %v\noutput: %s", err, out)
	}
	if !result.Initialized {
		t.Fatal("status: expected initialized=true")
	}
	if result.Backend != "pebble" {
		t.Fatalf("status: backend = %q, want %q", result.Backend, "pebble")
	}
	if result.NodeCount == 0 || result.EdgeCount == 0 {
		t.Fatalf("status: expected non-zero counts, got nodeCount=%d edgeCount=%d", result.NodeCount, result.EdgeCount)
	}
	if result.Index.State != "complete" {
		t.Fatalf("status: index.state = %q, want %q", result.Index.State, "complete")
	}
}

// pyTree writes and indexes a Python tree through the real init command.
func pyTree(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, src := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, err := execCmd("init", dir); err != nil {
		t.Fatalf("init: %v", err)
	}
	return dir
}

// twoRunsTree is two functions named run. main calls the one in a.py.
func twoRunsTree(t *testing.T) string {
	return pyTree(t, map[string]string{
		"a.py": "def load():\n    return 1\n\n\ndef run():\n    return load()\n",
		"b.py": "def save():\n    return 2\n\n\ndef run():\n    return save()\n",
		"m.py": "from a import run\n\n\ndef main():\n    return run()\n",
	})
}

// TestCallersTextListsTheDefinitions covers the scenarios "The
// definitions are listed" and "One definition in text", and the cap of
// the definitions line on both of its sides.
func TestCallersTextListsTheDefinitions(t *testing.T) {
	dir := twoRunsTree(t)
	out, _, err := execCmd("callers", "run", "-p", dir)
	if err != nil {
		t.Fatalf("callers run: %v", err)
	}
	want := "run has 1 caller(s):\n2 definitions named \"run\": a.py:5, b.py:5\n  main (function) m.py:4 (definition a.py:5)\n"
	if out != want {
		t.Errorf("callers run =\n%q\nwant\n%q", out, want)
	}

	one, _, err := execCmd("callers", "load", "-p", dir)
	if err != nil {
		t.Fatalf("callers load: %v", err)
	}
	if want := "load has 1 caller(s):\n  run (function) a.py:5\n"; one != want {
		t.Errorf("callers load =\n%q\nwant\n%q", one, want)
	}
	if strings.Contains(one, "definitions named") {
		t.Errorf("a name with one definition lists definitions:\n%s", one)
	}

	for _, n := range []int{20, 21} {
		files := map[string]string{}
		for i := 0; i < n; i++ {
			files[fmt.Sprintf("f%02d.py", i)] = "def many():\n    return 1\n"
		}
		out, _, err := execCmd("callers", "many", "-p", pyTree(t, files))
		if err != nil {
			t.Fatalf("callers many (%d): %v", n, err)
		}
		line := strings.SplitN(out, "\n", 3)[1]
		if got := strings.Count(line, ".py:1"); got != 20 {
			t.Errorf("%d definitions: the line holds %d labels, want 20:\n%s", n, got, line)
		}
		if tail := fmt.Sprintf(", +%d more", n-20); (n > 20) != strings.HasSuffix(line, tail) || (n == 20 && strings.Contains(line, "more")) {
			t.Errorf("%d definitions: wrong tail of the line:\n%s", n, line)
		}
	}
}

// TestCallersTextOfNoResultListsTheDefinitions is the scenario "A query
// that spans definitions and finds no result".
func TestCallersTextOfNoResultListsTheDefinitions(t *testing.T) {
	dir := pyTree(t, map[string]string{
		"a.py": "def run():\n    return 1\n",
		"b.py": "def run():\n    return 2\n",
	})
	out, _, err := execCmd("callers", "run", "-p", dir)
	if err != nil {
		t.Fatalf("callers run: %v", err)
	}
	if want := "run has 0 caller(s):\n2 definitions named \"run\": a.py:1, b.py:1\n"; out != want {
		t.Errorf("callers run =\n%q\nwant\n%q", out, want)
	}
}

// TestCalleesFileFlag runs --file and -f through the real command tree:
// a match selects one definition, and no match exits non-zero and lists
// the files.
func TestCalleesFileFlag(t *testing.T) {
	dir := twoRunsTree(t)
	for _, flag := range []string{"--file", "-f"} {
		out, _, err := execCmd("callees", "run", flag, "a.py", "-p", dir)
		if err != nil {
			t.Fatalf("callees run %s a.py: %v", flag, err)
		}
		if want := "run calls 1 callee(s):\n  load (function) a.py:1\n"; out != want {
			t.Errorf("callees run %s a.py =\n%q\nwant\n%q", flag, out, want)
		}
	}
	for _, cmd := range []string{"callers", "callees", "impact"} {
		_, _, err := execCmd(cmd, "run", "--file", "c.py", "-p", dir)
		if err == nil || !strings.Contains(err.Error(), `no definition of "run" is in a file that matches "c.py"; the files are a.py, b.py`) {
			t.Errorf("%s run --file c.py: error = %v, want the no-match error with both files", cmd, err)
		}
	}
}
