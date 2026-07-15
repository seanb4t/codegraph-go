package query

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/schema"
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

		// "widget" (not "e") — the TS-parity tokenizers (H1/H2, plan 03)
		// apply the same length-floor filtering real TS does, so a
		// single-character query now tokenizes to nothing (matching real
		// TS behavior, not the old lexical substring matcher's degenerate
		// "everything with an e in it" match). "widget" instead matches
		// pkga's Widget struct AND structurally reaches pkgb.go (Run
		// constructs a Widget{} and calls its Describe method), so the
		// maxFiles cap still has 2+ candidate files to cap down to 1.
		got, err := engine.Explore("widget", 1)
		if err != nil {
			t.Fatalf("Explore: unexpected error: %v", err)
		}
		if count := strings.Count(got, "```go\n"); count != 1 {
			t.Fatalf("Explore(\"widget\", maxFiles=1): got %d fenced source blocks, want 1 (max-files must cap distinct files):\n%s", count, got)
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

// mkMultiDefNode builds a minimal schema.Node for RenderNodeMultiDef unit
// tests — only the fields the multi-def renderer touches (Name/Kind/
// FilePath/StartLine/Signature/Id) need to be populated.
func mkMultiDefNode(id, file string, startLine int32) *schema.Node {
	return &schema.Node{Id: id, Name: "X", Kind: "function", FilePath: file, StartLine: startLine, Signature: "() error"}
}

// TestRenderNodeMultiDef pins NODE-02's exact multi-def markdown shape
// (RESEARCH §8, Pitfall 4): the two-line header, the HARD_CAP=16/
// BODY_BUDGET=12000 full-body budget (always rendering at least the
// first), and the LIST_CAP=20 overflow list. Exercises RenderNodeMultiDef
// directly with a fake fetch closure — no Engine/reader needed, since the
// renderer's I/O is injected (RESEARCH Pitfall — keep the renderer pure).
func TestRenderNodeMultiDef(t *testing.T) {
	t.Run("two defs renders the two-line header and both full bodies with no overflow", func(t *testing.T) {
		matches := []*schema.Node{mkMultiDefNode("a", "pkg/a.go", 10), mkMultiDefNode("b", "pkg/b.go", 20)}
		fetch := func(n *schema.Node) ([]byte, []*schema.Node, []*schema.Node, error) {
			return []byte("func X() error {\n\treturn nil\n}\n"), nil, nil, nil
		}

		got, err := RenderNodeMultiDef("X", matches, fetch)
		if err != nil {
			t.Fatalf("RenderNodeMultiDef: unexpected error: %v", err)
		}

		wantHead := "**2 definitions named \"X\"**\nReturning 2 in full — pick the one you need (no Read required).\n\n"
		if !strings.HasPrefix(got, wantHead) {
			t.Fatalf("RenderNodeMultiDef head mismatch:\ngot:  %q\nwant prefix: %q", got, wantHead)
		}
		if !strings.Contains(got, "\n\n---\n\n") {
			t.Fatalf("RenderNodeMultiDef: expected the two bodies separated by \"\\n\\n---\\n\\n\":\n%s", got)
		}
		if strings.Contains(got, "**Other definitions**") {
			t.Fatalf("RenderNodeMultiDef: unexpected overflow list for 2 defs within HARD_CAP:\n%s", got)
		}
		if !strings.HasSuffix(got, "```\n") {
			t.Fatalf("RenderNodeMultiDef: got %q, want it to end with the closing code fence (no Trail — calls/calledBy are both empty)", got)
		}
	})

	t.Run("more than HARD_CAP defs renders 16 full bodies and lists the rest", func(t *testing.T) {
		var matches []*schema.Node
		for i := 0; i < 20; i++ {
			matches = append(matches, mkMultiDefNode(fmt.Sprintf("n%02d", i), fmt.Sprintf("pkg/f%02d.go", i), int32(i+1)))
		}
		fetch := func(n *schema.Node) ([]byte, []*schema.Node, []*schema.Node, error) {
			return []byte("func X() {}\n"), nil, nil, nil
		}

		got, err := RenderNodeMultiDef("X", matches, fetch)
		if err != nil {
			t.Fatalf("RenderNodeMultiDef: unexpected error: %v", err)
		}

		wantSecondLine := "Returning 16 in full; 4 more listed below — pick the one you need (no Read required)."
		if !strings.Contains(got, wantSecondLine) {
			t.Fatalf("RenderNodeMultiDef: got %q, want it to contain %q", got, wantSecondLine)
		}
		if count := strings.Count(got, "\n\n---\n\n"); count != 15 {
			t.Fatalf("RenderNodeMultiDef: got %d section separators, want 15 (16 rendered bodies)", count)
		}
		if !strings.Contains(got, "**Other definitions**") {
			t.Fatalf("RenderNodeMultiDef: expected an overflow list for 20 defs > HARD_CAP=16:\n%s", got)
		}
		if n := strings.Count(got, "- `X` (function) —"); n != 4 {
			t.Fatalf("RenderNodeMultiDef: got %d listed overflow bullets, want 4:\n%s", n, got)
		}
		if strings.Contains(got, "more\n") && strings.Contains(got, "- …") {
			t.Fatalf("RenderNodeMultiDef: unexpected \"+K more\" line for only 4 overflow defs (LIST_CAP=20):\n%s", got)
		}
		if !strings.Contains(got, "> Need one of these in full? Call codegraph_node again with `file` (e.g. `\"f16.go\"`) or `line` — do NOT Read it.\n") {
			t.Fatalf("RenderNodeMultiDef: missing or mismatched closing hint line:\n%s", got)
		}
	})

	t.Run("bodies exceeding BODY_BUDGET stop early but always render the first", func(t *testing.T) {
		big := strings.Repeat("x", 13000)
		var matches []*schema.Node
		for i := 0; i < 3; i++ {
			matches = append(matches, mkMultiDefNode(fmt.Sprintf("n%d", i), fmt.Sprintf("pkg/f%d.go", i), int32(i+1)))
		}
		fetch := func(n *schema.Node) ([]byte, []*schema.Node, []*schema.Node, error) {
			return []byte(big + "\n"), nil, nil, nil
		}

		got, err := RenderNodeMultiDef("X", matches, fetch)
		if err != nil {
			t.Fatalf("RenderNodeMultiDef: unexpected error: %v", err)
		}
		if !strings.Contains(got, "Returning 1 in full; 2 more listed below") {
			t.Fatalf("RenderNodeMultiDef: expected exactly 1 rendered body when the first alone exceeds BODY_BUDGET=12000:\n%s", got[:min(200, len(got))])
		}
	})

	t.Run("a def with both empty Calls and CalledBy omits the Trail line entirely", func(t *testing.T) {
		matches := []*schema.Node{mkMultiDefNode("a", "pkg/a.go", 10), mkMultiDefNode("b", "pkg/b.go", 20)}
		fetch := func(n *schema.Node) ([]byte, []*schema.Node, []*schema.Node, error) {
			return []byte("func X() error {\n\treturn nil\n}\n"), nil, nil, nil
		}

		got, err := RenderNodeMultiDef("X", matches, fetch)
		if err != nil {
			t.Fatalf("RenderNodeMultiDef: unexpected error: %v", err)
		}
		if strings.Contains(got, "Trail") || strings.Contains(got, "Calls") || strings.Contains(got, "Called by") {
			t.Fatalf("RenderNodeMultiDef: Trail/Calls/CalledBy lines must be omitted when both are empty (matches live TS golden captures):\n%s", got)
		}
	})
}

// TestNoCoveringTestsWarning pins EXPL-04's exact trigger + exact string
// (RESEARCH §5, verbatim): a root with >=1 direct caller and NO covering
// test file ends with "; ⚠️ no covering tests found"; a root with covering
// test files keeps the existing "; tests: ..." clause (no warning); a
// root with ZERO callers gets NEITHER clause (mirrors TS's early-continue
// before the callers/tests block is ever reached).
func TestNoCoveringTestsWarning(t *testing.T) {
	root := &schema.Node{Name: "recoverAccount", Kind: "function", FilePath: "recovery/recovery.go", StartLine: 8}

	t.Run("callers but no covering test file appends the exact warning", func(t *testing.T) {
		got := renderBlastBullet(exploreBlast{Symbol: root, CallerCount: 1})
		want := "- `recoverAccount` (recovery/recovery.go:8) — 1 caller in `recovery/recovery.go`; ⚠️ no covering tests found"
		if got != want {
			t.Fatalf("renderBlastBullet: got %q, want %q", got, want)
		}
	})

	t.Run("callers with a covering test file keeps the existing tests: clause, no warning", func(t *testing.T) {
		got := renderBlastBullet(exploreBlast{Symbol: root, CallerCount: 1, TestFiles: []string{"recovery/recovery_test.go"}})
		want := "- `recoverAccount` (recovery/recovery.go:8) — 1 caller in `recovery/recovery.go`; tests: `recovery/recovery_test.go`"
		if got != want {
			t.Fatalf("renderBlastBullet: got %q, want %q", got, want)
		}
		if strings.Contains(got, "no covering tests") {
			t.Fatalf("renderBlastBullet: unexpected warning alongside a tests: clause:\n%s", got)
		}
	})

	t.Run("zero callers gets neither clause", func(t *testing.T) {
		got := renderBlastBullet(exploreBlast{Symbol: root, CallerCount: 0})
		want := "- `recoverAccount` (recovery/recovery.go:8) — 0 callers in `recovery/recovery.go`"
		if got != want {
			t.Fatalf("renderBlastBullet: got %q, want %q (no tests: clause, no warning)", got, want)
		}
	})
}

// TestSkeletonization pins H20's polymorphic-sibling skeletonization
// (RESEARCH §C.2/H20): an off-spine file whose class implements an
// interface with >= MIN_SIBLINGS=3 total implementers renders as a
// signature-only skeleton; below the threshold, or when the file IS
// on-spine (a central file), it renders full source as usual.
func TestSkeletonization(t *testing.T) {
	widget := &schema.Node{Id: "widget", Name: "Widget", Kind: "struct", FilePath: "pkg/widget.go", Signature: " struct{}"}
	gadget := &schema.Node{Id: "gadget", Name: "Gadget", Kind: "struct", FilePath: "pkg/gadget.go", Signature: " struct{}"}

	t.Run(">=3 implementers on an off-spine file triggers skeletonization", func(t *testing.T) {
		implementsIdx := map[string][]*schema.Edge{
			"iface": {
				{Source: "widget", Target: "iface", Kind: "implements"},
				{Source: "typeB", Target: "iface", Kind: "implements"},
				{Source: "typeC", Target: "iface", Kind: "implements"},
			},
		}
		groups := []exploreFileGroup{{Path: "pkg/widget.go", Symbols: []*schema.Node{widget}}}

		skeleton := computeSkeletonFiles(groups, nil, implementsIdx)
		if !skeleton["pkg/widget.go"] {
			t.Fatal("computeSkeletonFiles: expected pkg/widget.go to be flagged for skeletonization at 3 implementers")
		}

		got := RenderExplore("widget", 1, 1, groups, nil, map[string][]byte{"pkg/widget.go": []byte("package pkg\n\ntype Widget struct{}\n")}, false, skeleton)
		if !strings.Contains(got, "struct Widget struct{}\n```") {
			t.Fatalf("RenderExplore: expected a signature-only skeleton block, got:\n%s", got)
		}
		if strings.Contains(got, "package pkg") {
			t.Fatalf("RenderExplore: skeletonized file must not render full source (no 'package pkg' line):\n%s", got)
		}
	})

	t.Run("below MIN_SIBLINGS=3 does not skeletonize", func(t *testing.T) {
		implementsIdx := map[string][]*schema.Edge{
			"iface": {
				{Source: "gadget", Target: "iface", Kind: "implements"},
				{Source: "typeB", Target: "iface", Kind: "implements"},
			},
		}
		groups := []exploreFileGroup{{Path: "pkg/gadget.go", Symbols: []*schema.Node{gadget}}}

		skeleton := computeSkeletonFiles(groups, nil, implementsIdx)
		if skeleton["pkg/gadget.go"] {
			t.Fatal("computeSkeletonFiles: 2 implementers must not trigger skeletonization (MIN_SIBLINGS=3)")
		}
	})

	t.Run("an on-spine (central) file never skeletonizes, even at >=3 implementers", func(t *testing.T) {
		implementsIdx := map[string][]*schema.Edge{
			"iface": {
				{Source: "widget", Target: "iface", Kind: "implements"},
				{Source: "typeB", Target: "iface", Kind: "implements"},
				{Source: "typeC", Target: "iface", Kind: "implements"},
			},
		}
		groups := []exploreFileGroup{{Path: "pkg/widget.go", Symbols: []*schema.Node{widget}}}
		centralFiles := map[string]bool{"pkg/widget.go": true}

		skeleton := computeSkeletonFiles(groups, centralFiles, implementsIdx)
		if skeleton["pkg/widget.go"] {
			t.Fatal("computeSkeletonFiles: a central (on-spine) file must never skeletonize")
		}
	})
}

// TestRenderNodeSingleDefUnchanged is NODE-04's regression pin: a symbol
// with exactly one definition must still render via the original
// single-def RenderNode path, byte-for-byte identical to a direct
// RenderNode call — the multi-def machinery must never touch it.
func TestRenderNodeSingleDefUnchanged(t *testing.T) {
	engine, _ := nodeExploreFixture(t)

	alpha, err := engine.resolveSymbolNode("Alpha")
	if err != nil {
		t.Fatalf("resolveSymbolNode(Alpha): unexpected error: %v", err)
	}
	calls, err := engine.fetchCalls(alpha)
	if err != nil {
		t.Fatalf("fetchCalls(Alpha): unexpected error: %v", err)
	}
	rev, err := BuildReverseAdjacency(engine.reader)
	if err != nil {
		t.Fatalf("BuildReverseAdjacency: unexpected error: %v", err)
	}
	calledBy, err := engine.fetchCalledBy(alpha, rev)
	if err != nil {
		t.Fatalf("fetchCalledBy(Alpha): unexpected error: %v", err)
	}
	want := RenderNode(alpha, calls, calledBy)

	got, err := engine.Node("Alpha", "")
	if err != nil {
		t.Fatalf("Node(Alpha): unexpected error: %v", err)
	}
	if got != want {
		t.Fatalf("Node(Alpha) diverges from a direct RenderNode call (NODE-04 regression):\ngot:  %q\nwant: %q", got, want)
	}
}

// TestNodeMultiDefWiring proves Engine.Node dispatches to the multi-def
// render path end-to-end (not just the isolated enumeration/render unit
// tests) when a symbol has more than one definition — the CLI and MCP
// call the same Engine.Node method (D-08b), so this is the shared
// integration point both surfaces rely on (EXPL-05/NODE-04's "shared
// engine" structural constraint applied to NODE-01/02).
func TestNodeMultiDefWiring(t *testing.T) {
	dir := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		full := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	write("a/dup.go", "package a\n\nfunc Dup() int {\n\treturn 1\n}\n")
	write("b/dup.go", "package b\n\nfunc Dup() int {\n\treturn 2\n}\n")

	nodes := map[string]*schema.Node{
		"a-dup": {Id: "a-dup", Name: "Dup", Kind: "function", FilePath: "a/dup.go", StartLine: 3, Signature: "() int"},
		"b-dup": {Id: "b-dup", Name: "Dup", Kind: "function", FilePath: "b/dup.go", StartLine: 3, Signature: "() int"},
	}
	e := NewWithRoot(&traverseFakeReader{nodes: nodes}, dir)

	got, err := e.Node("Dup", "")
	if err != nil {
		t.Fatalf("Node(Dup): unexpected error: %v", err)
	}

	wantHead := "**2 definitions named \"Dup\"**\nReturning 2 in full — pick the one you need (no Read required).\n\n"
	if !strings.HasPrefix(got, wantHead) {
		t.Fatalf("Node(Dup) head mismatch:\ngot:  %q\nwant prefix: %q", got, wantHead)
	}
	if !strings.Contains(got, "**Dup** (function)") {
		t.Fatalf("Node(Dup): missing per-def header in:\n%s", got)
	}
}
