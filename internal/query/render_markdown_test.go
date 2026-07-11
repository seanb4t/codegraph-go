package query

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// goldenExploreOutput/goldenNodeOutput load the golden fixture's "output"
// string at test time (not a hardcoded copy) so drift in either direction
// (a paraphrase, a reordered section, a changed disclaimer) fails this
// test rather than silently passing (D-05a/D-05b, RESEARCH Pitfall 3).
type goldenCapture struct {
	Command string `json:"command"`
	Output  string `json:"output"`
}

func loadGoldenOutput(t *testing.T, name string) string {
	t.Helper()

	path := filepath.Join("..", "..", "testdata", "golden", "corpus", "weft-go", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden fixture %s: %v", path, err)
	}
	var capture goldenCapture
	if err := json.Unmarshal(data, &capture); err != nil {
		t.Fatalf("unmarshal golden fixture %s: %v", path, err)
	}
	if capture.Output == "" {
		t.Fatalf("golden fixture %s has an empty output field", path)
	}
	return capture.Output
}

// extractDisclaimer pulls the "> ..." blockquote paragraph out of an
// explore markdown output (golden or rendered) — the text between "> " and
// the following blank line. Used to prove Engine.Explore's disclaimer is
// byte-identical to the golden's, not paraphrased (D-05a, T-03-06-Drift).
func extractDisclaimer(t *testing.T, output string) string {
	t.Helper()

	marker := "> "
	start := strings.Index(output, marker)
	if start == -1 {
		t.Fatalf("output has no %q blockquote disclaimer:\n%s", marker, output)
	}
	start += len(marker)
	rest := output[start:]
	end := strings.Index(rest, "\n\n")
	if end == -1 {
		t.Fatalf("output's disclaimer blockquote has no terminating blank line:\n%s", output)
	}
	return rest[:end]
}

// nodeExploreFixture copies+indexes the shared gofixture (engine_test.go's
// copyFixture/indexFixture, reused at runtime per Wave-3 isolation) and
// opens an Engine on it via OpenAt, which threads the fixture's repo root
// through to the Engine so Node (file mode) and Explore can read source
// fresh from disk (D-05a).
func nodeExploreFixture(t *testing.T) (*Engine, string) {
	t.Helper()

	dir := copyFixture(t)
	indexFixture(t, dir)

	engine, closer, err := OpenAt(dir)
	if err != nil {
		t.Fatalf("OpenAt: unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = closer.Close() })
	return engine, dir
}

// TestNode pins Engine.Node's two modes: symbol detail (golden node.json
// section order: name/kind, Location, Signature, Trail, Calls →, Called by
// ←) and file mode (line-numbered, tab-indented verbatim read).
func TestNode(t *testing.T) {
	t.Run("symbol mode renders the fixed node.json section order", func(t *testing.T) {
		engine, _ := nodeExploreFixture(t)

		// Independent oracle: resolve the same symbols traverse.go already
		// resolves, so this test does not just re-implement Node's own
		// logic as its own check.
		alpha, err := engine.resolveSymbolNode("Alpha")
		if err != nil {
			t.Fatalf("resolveSymbolNode(Alpha): unexpected error: %v", err)
		}
		helper, err := engine.resolveSymbolNode("helper")
		if err != nil {
			t.Fatalf("resolveSymbolNode(helper): unexpected error: %v", err)
		}
		run, err := engine.resolveSymbolNode("Run")
		if err != nil {
			t.Fatalf("resolveSymbolNode(Run): unexpected error: %v", err)
		}

		got, err := engine.Node("Alpha", "")
		if err != nil {
			t.Fatalf("Node: unexpected error: %v", err)
		}

		wantHead := fmt.Sprintf(
			"**Alpha** (function)\n\n**Location:** %s:%d\n**Signature:** `%s`\n**Trail — codegraph_node any of these to follow it (no Read needed)**\n",
			alpha.FilePath, alpha.StartLine, alpha.Signature,
		)
		if !strings.HasPrefix(got, wantHead) {
			t.Fatalf("Node(Alpha) head mismatch:\ngot:\n%s\nwant prefix:\n%s", got, wantHead)
		}

		wantCalls := fmt.Sprintf("**Calls →** %s (%s:%d)\n", helper.Name, helper.FilePath, helper.StartLine)
		if !strings.Contains(got, wantCalls) {
			t.Fatalf("Node(Alpha) missing Calls → line %q in:\n%s", wantCalls, got)
		}

		wantCalledBy := fmt.Sprintf("**Called by ←** %s (%s:%d)\n", run.Name, run.FilePath, run.StartLine)
		if !strings.Contains(got, wantCalledBy) {
			t.Fatalf("Node(Alpha) missing Called by ← line %q in:\n%s", wantCalledBy, got)
		}
	})

	t.Run("file mode renders a line-numbered, tab-indented verbatim read", func(t *testing.T) {
		engine, dir := nodeExploreFixture(t)

		want, err := os.ReadFile(filepath.Join(dir, "pkga", "pkga.go"))
		if err != nil {
			t.Fatalf("read fixture file directly: %v", err)
		}
		wantLines := strings.Split(string(want), "\n")
		if wantLines[len(wantLines)-1] == "" {
			wantLines = wantLines[:len(wantLines)-1]
		}

		got, err := engine.Node("", "pkga/pkga.go")
		if err != nil {
			t.Fatalf("Node (file mode): unexpected error: %v", err)
		}

		if !strings.HasPrefix(got, "```go\n1\t"+wantLines[0]+"\n") {
			t.Fatalf("Node (file mode): got %q, want a fenced block starting with line 1 %q", got, wantLines[0])
		}
		lastLine := fmt.Sprintf("%d\t%s\n```\n", len(wantLines), wantLines[len(wantLines)-1])
		if !strings.HasSuffix(got, lastLine) {
			t.Fatalf("Node (file mode): got %q, want it to end with %q", got, lastLine)
		}
		if strings.Count(got, "\n") != len(wantLines)+2 {
			t.Fatalf("Node (file mode): got %d newlines, want %d (fence-open + %d source lines + fence-close)", strings.Count(got, "\n"), len(wantLines)+2, len(wantLines))
		}
	})

	t.Run("path escaping the repo root is rejected", func(t *testing.T) {
		engine, _ := nodeExploreFixture(t)

		if _, err := engine.Node("", "../../../../etc/passwd"); err == nil {
			t.Fatal("Node (file mode): expected an error for a path escaping the repo root, got nil")
		}
		if _, err := engine.Node("", "/etc/passwd"); err == nil {
			t.Fatal("Node (file mode): expected an error for an absolute path, got nil")
		}
	})
}

// TestExplore pins the golden explore.json markdown contract: the
// exploration header, the "Found N symbol(s) across M file(s)." line, the
// blast-radius bullets, the verbatim-source disclaimer (byte-equal to the
// golden's), and per-file line-numbered fenced source blocks.
func TestExplore(t *testing.T) {
	t.Run("renders the fixed explore.json section order for a single-file match", func(t *testing.T) {
		engine, dir := nodeExploreFixture(t)

		alpha, err := engine.resolveSymbolNode("Alpha")
		if err != nil {
			t.Fatalf("resolveSymbolNode(Alpha): unexpected error: %v", err)
		}

		got, err := engine.Explore("Alpha", 1)
		if err != nil {
			t.Fatalf("Explore: unexpected error: %v", err)
		}

		wantHead := "**Exploration: Alpha**\n\nFound 1 symbol across 1 file.\n\n**Blast radius — what depends on these (update/verify before editing)**\n\n"
		if !strings.HasPrefix(got, wantHead) {
			t.Fatalf("Explore(Alpha) head mismatch:\ngot:\n%s\nwant prefix:\n%s", got, wantHead)
		}

		wantBullet := fmt.Sprintf("- `Alpha` (%s:%d) — 1 caller in `%s`", alpha.FilePath, alpha.StartLine, alpha.FilePath)
		if !strings.Contains(got, wantBullet) {
			t.Fatalf("Explore(Alpha) missing blast-radius bullet %q in:\n%s", wantBullet, got)
		}
		// Alpha's only caller (pkgb.Run) is not a test symbol, so the
		// bullet must not claim any test files.
		if strings.Contains(got, "; tests:") {
			t.Fatalf("Explore(Alpha) bullet unexpectedly includes a tests: clause (Run is not a test caller):\n%s", got)
		}

		if !strings.Contains(got, "\n\n**Source Code**\n\n") {
			t.Fatalf("Explore(Alpha) missing **Source Code** section:\n%s", got)
		}

		gotDisclaimer := extractDisclaimer(t, got)
		wantDisclaimer := extractDisclaimer(t, loadGoldenOutput(t, "explore.json"))
		if gotDisclaimer != wantDisclaimer {
			t.Fatalf("Explore disclaimer diverges from the golden explore.json disclaimer (D-05a — must be verbatim, not paraphrased):\ngot:  %q\nwant: %q", gotDisclaimer, wantDisclaimer)
		}

		wantFileHeader := fmt.Sprintf("**`%s`** — Alpha(function)\n\n```go\n1\t", alpha.FilePath)
		if !strings.Contains(got, wantFileHeader) {
			t.Fatalf("Explore(Alpha) missing per-file source header %q in:\n%s", wantFileHeader, got)
		}

		// Source must be read fresh from disk, byte-for-byte identical to
		// what's on disk right now (D-05a).
		raw, err := os.ReadFile(filepath.Join(dir, alpha.FilePath))
		if err != nil {
			t.Fatalf("read fixture file directly: %v", err)
		}
		rawLines := strings.Split(string(raw), "\n")
		if rawLines[len(rawLines)-1] == "" {
			rawLines = rawLines[:len(rawLines)-1]
		}
		lastSourceLine := fmt.Sprintf("%d\t%s\n```\n\n", len(rawLines), rawLines[len(rawLines)-1])
		if !strings.HasSuffix(got, lastSourceLine) {
			t.Fatalf("Explore(Alpha) got %q, want it to end with %q", got, lastSourceLine)
		}
	})

	t.Run("max-files caps the number of rendered file blocks", func(t *testing.T) {
		engine, _ := nodeExploreFixture(t)

		got, err := engine.Explore("e", 1)
		if err != nil {
			t.Fatalf("Explore: unexpected error: %v", err)
		}
		if count := strings.Count(got, "```go\n"); count != 1 {
			t.Fatalf("Explore(\"e\", maxFiles=1): got %d fenced source blocks, want 1 (max-files must cap distinct files):\n%s", count, got)
		}
	})

	t.Run("no staleness banner when the graph is current", func(t *testing.T) {
		engine, _ := nodeExploreFixture(t)

		got, err := engine.Explore("Alpha", 1)
		if err != nil {
			t.Fatalf("Explore: unexpected error: %v", err)
		}
		if !strings.HasPrefix(got, "**Exploration: Alpha**") {
			t.Fatalf("Explore(Alpha): got %q, want no staleness banner prefix (graph is current)", got)
		}
	})

	t.Run("staleness banner prepended when a sync is pending", func(t *testing.T) {
		engine, dir := nodeExploreFixture(t)

		sidecar := filepath.Join(dir, codegraphDirName, staleSidecarName)
		if err := os.WriteFile(sidecar, nil, 0o644); err != nil {
			t.Fatalf("write sidecar: %v", err)
		}

		got, err := engine.Explore("Alpha", 1)
		if err != nil {
			t.Fatalf("Explore: unexpected error: %v", err)
		}
		if !strings.HasPrefix(got, staleBannerText) {
			t.Fatalf("Explore(Alpha) with pending sync: got %q, want it to start with the staleness banner %q", got, staleBannerText)
		}
		if !strings.Contains(got, "**Exploration: Alpha**") {
			t.Fatalf("Explore(Alpha) with pending sync: banner must not replace the exploration header:\n%s", got)
		}
	})
}
