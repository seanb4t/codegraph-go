package capability

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer"
)

// repoRoot locates the repository root from this test file's own package
// directory (internal/indexer/capability), three levels up.
func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("resolved repo root %q does not contain go.mod: %v", root, err)
	}
	return root
}

// TestMatrix_CoversRegisteredLanguages proves the D-11 descriptor covers
// EXACTLY the languages registered in the LanguageSpec registry — no
// missing (a registered language with no matrix entry would silently
// overclaim nothing about it), no extra (a matrix entry for an
// unregistered language would be describing something that doesn't exist).
func TestMatrix_CoversRegisteredLanguages(t *testing.T) {
	registered := indexer.RegisteredLanguageIDs()

	descriptorIDs := make([]string, 0, len(matrix))
	for id := range matrix {
		descriptorIDs = append(descriptorIDs, id)
	}
	sort.Strings(descriptorIDs)

	registeredSet := make(map[string]bool, len(registered))
	for _, id := range registered {
		registeredSet[id] = true
	}
	descriptorSet := make(map[string]bool, len(descriptorIDs))
	for _, id := range descriptorIDs {
		descriptorSet[id] = true
	}

	var missing, extra []string
	for _, id := range registered {
		if !descriptorSet[id] {
			missing = append(missing, id)
		}
	}
	for _, id := range descriptorIDs {
		if !registeredSet[id] {
			extra = append(extra, id)
		}
	}

	if len(missing) > 0 {
		t.Errorf("registered languages missing from the capability matrix: %v", missing)
	}
	if len(extra) > 0 {
		t.Errorf("capability matrix entries for unregistered languages: %v", extra)
	}
}

// TestMatrix_CellsValid proves every capability cell is one of
// full|partial|none, and — D-11's "documented-partial means written down,
// not silently missing" — every non-full cell has a corresponding named
// gap.
func TestMatrix_CellsValid(t *testing.T) {
	for id, entry := range matrix {
		axes := map[string]Coverage{
			"Extraction": entry.Extraction,
			"Resolution": entry.Resolution,
			"Dispatch":   entry.Dispatch,
			"Routing":    entry.Routing,
		}

		hasNonFull := false
		for axis, cov := range axes {
			if !cov.valid() {
				t.Errorf("%s: %s coverage %q is not one of full|partial|none", id, axis, cov)
			}
			if cov != CoverageFull {
				hasNonFull = true
			}
		}

		if hasNonFull && len(entry.Gaps) == 0 {
			t.Errorf("%s: has a non-full axis (%+v) but Gaps is empty — every non-full cell must name its gap", id, axes)
		}
	}
}

// mdTableRowPattern matches one coverage-table row in
// docs/LANGUAGE-CAPABILITY-MATRIX.md: "| `id` | extraction | resolution |
// dispatch | routing |".
var mdTableRowPattern = regexp.MustCompile(`^\|\s*` + "`" + `(\w+)` + "`" + `\s*\|\s*(\w+)\s*\|\s*(\w+)\s*\|\s*(\w+)\s*\|\s*(\w+)\s*\|\s*$`)

// TestMatrix_DocMirrorsDescriptor proves docs/LANGUAGE-CAPABILITY-MATRIX.md
// — the human-readable half of D-11 — carries the EXACT same coverage
// values as the Go descriptor for every language, and that every named gap
// in the descriptor appears verbatim in the doc. If a future edit changes
// one half without the other, this test catches the drift.
func TestMatrix_DocMirrorsDescriptor(t *testing.T) {
	docPath := filepath.Join(repoRoot(t), "docs", "LANGUAGE-CAPABILITY-MATRIX.md")
	raw, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("read %s: %v", docPath, err)
	}
	doc := string(raw)

	found := make(map[string]bool)
	for _, line := range strings.Split(doc, "\n") {
		m := mdTableRowPattern.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		id, extraction, resolution, dispatch, routing := m[1], m[2], m[3], m[4], m[5]

		entry, ok := matrix[id]
		if !ok {
			t.Errorf("doc table row for %q has no matching capability matrix entry", id)
			continue
		}
		found[id] = true

		if Coverage(extraction) != entry.Extraction {
			t.Errorf("%s: doc Extraction=%q, descriptor Extraction=%q", id, extraction, entry.Extraction)
		}
		if Coverage(resolution) != entry.Resolution {
			t.Errorf("%s: doc Resolution=%q, descriptor Resolution=%q", id, resolution, entry.Resolution)
		}
		if Coverage(dispatch) != entry.Dispatch {
			t.Errorf("%s: doc Dispatch=%q, descriptor Dispatch=%q", id, dispatch, entry.Dispatch)
		}
		if Coverage(routing) != entry.Routing {
			t.Errorf("%s: doc Routing=%q, descriptor Routing=%q", id, routing, entry.Routing)
		}
	}

	for id := range matrix {
		if !found[id] {
			t.Errorf("capability matrix entry %q has no corresponding row in the doc table", id)
		}
	}

	// Every named gap must appear verbatim in the doc — proves the doc's
	// per-language gap bullets are not silently missing or reworded.
	for id, entry := range matrix {
		for _, gap := range entry.Gaps {
			if !strings.Contains(doc, gap) {
				t.Errorf("%s: gap not found verbatim in docs/LANGUAGE-CAPABILITY-MATRIX.md: %q", id, gap)
			}
		}
	}
}

// goldenTestFuncsByLanguage maps a priority-4 language ID to the
// behavioral-suite test function name testdata/golden/ must define for a
// "full" Resolution or Dispatch entry to be trustworthy (D-12: a "full"
// entry in the matrix must have a corresponding green behavioral-suite
// test — 05-RESEARCH.md §Validation Architecture).
var goldenTestFuncsByLanguage = map[string]string{
	"go":         "TestCorpusBehavior_Go",
	"java":       "TestCorpusBehavior_Java",
	"csharp":     "TestCorpusBehavior_CSharp",
	"python":     "TestCorpusBehavior_Python",
	"typescript": "TestCorpusBehavior_TSJS",
	"tsx":        "TestCorpusBehavior_TSJS",
	"javascript": "TestCorpusBehavior_TSJS",
}

// goldenTestFuncNames scans every *_test.go file directly under
// testdata/golden/ (via go/parser, never executing anything) and returns
// the set of top-level test function names declared there.
func goldenTestFuncNames(t *testing.T) map[string]bool {
	t.Helper()
	dir := filepath.Join(repoRoot(t), "testdata", "golden")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}

	names := make(map[string]bool)
	fset := token.NewFileSet()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil {
				continue
			}
			names[fn.Name.Name] = true
		}
	}
	return names
}

// TestMatrix_FullPriority4EntriesHaveGoldenTest is the D-11/D-12 phase gate
// (05-VALIDATION.md §Sampling Rate: "a 'full' entry in the matrix must have
// a corresponding green behavioral golden test"): every priority-4 language
// whose Resolution or Dispatch is "full" must have its mapped
// testdata/golden test function actually declared — this test would fail
// if a future edit marked a language "full" without ever wiring its
// golden harness, or renamed/removed that harness without updating
// the matrix.
func TestMatrix_FullPriority4EntriesHaveGoldenTest(t *testing.T) {
	declared := goldenTestFuncNames(t)

	for id, entry := range matrix {
		wantFuncName, isPriority4 := goldenTestFuncsByLanguage[id]
		if !isPriority4 {
			continue
		}
		if entry.Resolution != CoverageFull && entry.Dispatch != CoverageFull {
			continue
		}
		if !declared[wantFuncName] {
			t.Errorf("%s: Resolution/Dispatch is full but %s is not declared under testdata/golden/", id, wantFuncName)
		}
	}
}

// TestMatrix_PartialEntriesNameGaps is the second half of the D-11/D-12
// phase gate: every "partial" cell (not just "any non-full cell", scoped
// specifically to partial per the plan's own acceptance criteria) must
// carry at least one named gap — a "partial" coverage claim with zero
// documented boundary would be exactly the silent-overclaim D-11 forbids.
func TestMatrix_PartialEntriesNameGaps(t *testing.T) {
	for id, entry := range matrix {
		axes := map[string]Coverage{
			"Extraction": entry.Extraction,
			"Resolution": entry.Resolution,
			"Dispatch":   entry.Dispatch,
			"Routing":    entry.Routing,
		}
		for axis, cov := range axes {
			if cov == CoveragePartial && len(entry.Gaps) == 0 {
				t.Errorf("%s: %s is partial but Gaps is empty", id, axis)
			}
		}
	}
}

// TestLookup proves the Lookup accessor round-trips the package-level
// matrix correctly, including the not-found case.
func TestLookup(t *testing.T) {
	entry, ok := Lookup("go")
	if !ok {
		t.Fatal("expected Lookup(\"go\") to return ok=true")
	}
	if entry.Extraction != CoverageFull {
		t.Errorf("expected go Extraction=full, got %q", entry.Extraction)
	}

	if _, ok := Lookup("cobol"); ok {
		t.Error("expected Lookup(\"cobol\") to return ok=false")
	}
}

// TestAll proves All() returns a defensive copy — mutating the returned
// map/slices must never affect the package-level matrix.
func TestAll(t *testing.T) {
	all := All()
	if len(all) != len(matrix) {
		t.Fatalf("expected All() to return %d entries, got %d", len(matrix), len(all))
	}

	rust := all["rust"]
	if len(rust.Gaps) == 0 {
		t.Fatal("expected rust to have gaps for this mutation test to be meaningful")
	}
	rust.Gaps[0] = "MUTATED"
	all["rust"] = rust

	original, ok := Lookup("rust")
	if !ok {
		t.Fatal("expected Lookup(\"rust\") to return ok=true")
	}
	if original.Gaps[0] == "MUTATED" {
		t.Fatal("mutating All()'s returned copy leaked into the package-level matrix")
	}
}
