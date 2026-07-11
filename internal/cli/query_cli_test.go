package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

// setupIndexedFixture copies the shared gofixture (copyFixture,
// cli_test.go) into a fresh temp dir and runs `codegraph init` against it
// via execCmd — mirroring TestInitIndexUninit's own setup — so every
// query-command subtest below runs against a real Pebble-backed index
// rather than a mock (03-PATTERNS.md §"Test scaffolding").
func setupIndexedFixture(t *testing.T) string {
	t.Helper()

	dir := copyFixture(t)
	if _, _, err := execCmd("init", dir); err != nil {
		t.Fatalf("init fixture: unexpected error: %v", err)
	}
	return dir
}

func TestQueryCmd(t *testing.T) {
	dir := setupIndexedFixture(t)

	t.Run("--json emits the golden query.json envelope shape", func(t *testing.T) {
		out, _, err := execCmd("query", "main", "-p", dir, "--json")
		if err != nil {
			t.Fatalf("query: unexpected error: %v", err)
		}

		var envelopes []struct {
			Node struct {
				Name string `json:"name"`
				Kind string `json:"kind"`
			} `json:"node"`
		}
		if err := json.Unmarshal([]byte(out), &envelopes); err != nil {
			t.Fatalf("query --json: invalid JSON: %v\noutput: %s", err, out)
		}
		found := false
		for _, e := range envelopes {
			if e.Node.Name == "main" {
				found = true
			}
		}
		if !found {
			t.Fatalf("query --json: expected a %q match, got %+v", "main", envelopes)
		}
	})

	t.Run("default output is human-readable, not JSON", func(t *testing.T) {
		out, _, err := execCmd("query", "main", "-p", dir)
		if err != nil {
			t.Fatalf("query: unexpected error: %v", err)
		}
		if !strings.Contains(out, "main") {
			t.Fatalf("query: expected output to mention %q, got %q", "main", out)
		}
		if strings.HasPrefix(strings.TrimSpace(out), "[") {
			t.Fatalf("query (no --json): expected human-readable output, got JSON: %q", out)
		}
	})

	t.Run("--kind rejects an unknown kind", func(t *testing.T) {
		_, _, err := execCmd("query", "main", "-p", dir, "--kind", "bogus")
		if err == nil {
			t.Fatal("query --kind bogus: expected an error, got nil")
		}
	})

	t.Run("not initialized directory errors", func(t *testing.T) {
		fresh := t.TempDir()
		_, _, err := execCmd("query", "main", "-p", fresh)
		if err == nil {
			t.Fatal("query against uninitialized dir: expected an error, got nil")
		}
	})
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
