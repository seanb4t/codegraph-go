package query

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/schema"
)

// newFileSymbolsFixtureEngine builds a small, fully-controlled Engine
// over an in-repo t.TempDir() fixture, following detail_test.go's
// newDetailFixtureEngine convention: a traverseFakeReader node set paired
// with NewWithRoot over a real temp directory. This is required, not
// cosmetic — ValidateRepoRelativePath's confinement re-verifies against
// the filesystem (filepath.EvalSymlinks), so the requested path must
// physically exist on disk even though FileSymbols itself never reads
// its content (T-05-32).
//
// Fixture files and symbols:
//   - three.go: three symbols — Beta (line 1), Alpha (line 5), Zeta
//     (line 5) — proving start-line-then-name ordering (Alpha before
//     Zeta at the tied line 5) — plus its own file-kind record and a
//     synthetic package-kind record sharing its path, both of which
//     FileSymbols must exclude.
//   - unindexed.go: exists on disk but carries no symbol records — the
//     index-miss case, with three.go as its sibling proving the miss is
//     real, not a broken lookup.
func newFileSymbolsFixtureEngine(t *testing.T) *Engine {
	t.Helper()
	dir := t.TempDir()

	nodes := map[string]*schema.Node{
		"three-file": {Id: "three-file", Kind: "file", FilePath: "three.go", Language: "go"},
		"three-pkg":  {Id: "three-pkg", Kind: "package", Name: "p", FilePath: "three.go"},
		"three-beta": {Id: "three-beta", Name: "Beta", Kind: "function", FilePath: "three.go", StartLine: 1, EndLine: 1},
		"three-zeta": {Id: "three-zeta", Name: "Zeta", Kind: "function", FilePath: "three.go", StartLine: 5, EndLine: 5},
		"three-alpha": {
			Id: "three-alpha", Name: "Alpha", Kind: "function", FilePath: "three.go", StartLine: 5, EndLine: 5,
		},
	}

	files := map[string]string{
		"three.go":     "package p\n\nfunc Beta() {}\nfunc Alpha() {}\nfunc Zeta() {}\n",
		"unindexed.go": "package p\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write fixture file %s: %v", name, err)
		}
	}

	return NewWithRoot(&traverseFakeReader{nodes: nodes}, dir)
}

// TestFileSymbolsReturnsAllSymbolsOrderedByLineThenName pins the primary
// behavior: a file carrying three symbol nodes returns all three,
// ordered by start line and then by name for equal lines, with Total
// equal to 3 and Truncated false.
func TestFileSymbolsReturnsAllSymbolsOrderedByLineThenName(t *testing.T) {
	e := newFileSymbolsFixtureEngine(t)

	got, err := e.FileSymbols("three.go")
	if err != nil {
		t.Fatalf("FileSymbols: unexpected error: %v", err)
	}
	if got.Total != 3 {
		t.Fatalf("FileSymbols: Total = %d, want 3", got.Total)
	}
	if got.Truncated {
		t.Fatalf("FileSymbols: Truncated = true, want false")
	}
	if len(got.Symbols) != 3 {
		t.Fatalf("FileSymbols: len(Symbols) = %d, want 3", len(got.Symbols))
	}
	wantOrder := []string{"Beta", "Alpha", "Zeta"}
	for i, want := range wantOrder {
		if got.Symbols[i].Name != want {
			t.Fatalf("FileSymbols: Symbols[%d].Name = %q, want %q (full order: %v)", i, got.Symbols[i].Name, want, wantOrder)
		}
	}
}

// TestFileSymbolsExcludesFileNode pins that the file's own file-kind node
// is NOT among the returned symbols, while the three real symbols still
// are — asserting both sides so the exclusion cannot pass by returning
// nothing at all.
func TestFileSymbolsExcludesFileNode(t *testing.T) {
	e := newFileSymbolsFixtureEngine(t)

	got, err := e.FileSymbols("three.go")
	if err != nil {
		t.Fatalf("FileSymbols: unexpected error: %v", err)
	}
	found := map[string]bool{}
	for _, s := range got.Symbols {
		if s.Kind == "file" {
			t.Fatalf("FileSymbols: returned the file's own file-kind record %q, want it excluded", s.Id)
		}
		found[s.Name] = true
	}
	for _, want := range []string{"Beta", "Alpha", "Zeta"} {
		if !found[want] {
			t.Fatalf("FileSymbols: real symbol %q missing from result", want)
		}
	}
}

// TestFileSymbolsExcludesPackagePseudoNode pins that a synthetic
// package-kind node is never returned, for the same reason it is
// excluded from Engine.FileGraph's rollup (D-08).
func TestFileSymbolsExcludesPackagePseudoNode(t *testing.T) {
	e := newFileSymbolsFixtureEngine(t)

	got, err := e.FileSymbols("three.go")
	if err != nil {
		t.Fatalf("FileSymbols: unexpected error: %v", err)
	}
	for _, s := range got.Symbols {
		if s.Kind == "package" {
			t.Fatalf("FileSymbols: returned a synthetic package-kind record %q, want it excluded", s.Id)
		}
	}
	if got.Total != 3 {
		t.Fatalf("FileSymbols: Total = %d, want 3 (package pseudo-node excluded from the total too)", got.Total)
	}
}

// TestFileSymbolsRejectsPathEscapingRepoRootBeforeScanning pins T-05-28:
// a path that escapes the repository root returns an error, and the
// error is produced BEFORE any scan runs. The fixture node's FilePath is
// deliberately set to the SAME escaping string being requested, so a
// scan-then-validate implementation would find and return it — proving
// the ordering, not merely the rejection.
func TestFileSymbolsRejectsPathEscapingRepoRootBeforeScanning(t *testing.T) {
	dir := t.TempDir()
	const escaping = "../outside.go"
	nodes := map[string]*schema.Node{
		"outside-sym": {Id: "outside-sym", Name: "Outside", Kind: "function", FilePath: escaping, StartLine: 1},
	}
	e := NewWithRoot(&traverseFakeReader{nodes: nodes}, dir)

	got, err := e.FileSymbols(escaping)
	if err == nil {
		t.Fatalf("FileSymbols(%q): got result %+v, want an error (path escapes the repo root)", escaping, got)
	}
	if len(got.Symbols) != 0 {
		t.Fatalf("FileSymbols(%q): returned %d symbols alongside the error, want none — the scan must not have run", escaping, len(got.Symbols))
	}
}

// TestFileSymbolsRejectsAbsolutePathBeforeScanning pins T-05-28's second
// case: an absolute path returns an error, produced BEFORE any scan
// runs, using the same scan-would-otherwise-match fixture technique.
func TestFileSymbolsRejectsAbsolutePathBeforeScanning(t *testing.T) {
	dir := t.TempDir()
	const abs = "/etc/passwd"
	nodes := map[string]*schema.Node{
		"abs-sym": {Id: "abs-sym", Name: "AbsSym", Kind: "function", FilePath: abs, StartLine: 1},
	}
	e := NewWithRoot(&traverseFakeReader{nodes: nodes}, dir)

	got, err := e.FileSymbols(abs)
	if err == nil {
		t.Fatalf("FileSymbols(%q): got result %+v, want an error (absolute path is not allowed)", abs, got)
	}
	if len(got.Symbols) != 0 {
		t.Fatalf("FileSymbols(%q): returned %d symbols alongside the error, want none — the scan must not have run", abs, len(got.Symbols))
	}
}

// TestFileSymbolsUnknownPathReturnsEmptyNotError pins that a path the
// index does not carry returns an empty list, a zero total, and no
// error — while a sibling path in the same fixture still returns its
// symbols, so the empty result is a real miss and not a broken lookup.
func TestFileSymbolsUnknownPathReturnsEmptyNotError(t *testing.T) {
	e := newFileSymbolsFixtureEngine(t)

	got, err := e.FileSymbols("unindexed.go")
	if err != nil {
		t.Fatalf("FileSymbols(unindexed.go): unexpected error: %v", err)
	}
	if got.Total != 0 {
		t.Fatalf("FileSymbols(unindexed.go): Total = %d, want 0", got.Total)
	}
	if len(got.Symbols) != 0 {
		t.Fatalf("FileSymbols(unindexed.go): len(Symbols) = %d, want 0", len(got.Symbols))
	}
	if got.Truncated {
		t.Fatalf("FileSymbols(unindexed.go): Truncated = true, want false")
	}

	sibling, err := e.FileSymbols("three.go")
	if err != nil {
		t.Fatalf("FileSymbols(three.go): unexpected error: %v", err)
	}
	if sibling.Total != 3 {
		t.Fatalf("FileSymbols(three.go): Total = %d, want 3 (sibling path must still return its symbols)", sibling.Total)
	}
}

// TestFileSymbolsCapsAtMaxFileSymbols pins T-05-30: a file carrying more
// symbols than MaxFileSymbols returns exactly the cap's worth, a total
// naming the real larger number, and Truncated true. The returned length
// is asserted to equal the cap EXACTLY and the total to be STRICTLY
// greater — a less-than-or-equal assertion alone is satisfied by an
// empty list (rule 84d1gfpywd).
func TestFileSymbolsCapsAtMaxFileSymbols(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "big.go"), []byte("package p\n"), 0o644); err != nil {
		t.Fatalf("write fixture file big.go: %v", err)
	}

	const overCap = MaxFileSymbols + 500
	nodes := make(map[string]*schema.Node, overCap)
	for i := 0; i < overCap; i++ {
		id := fmt.Sprintf("big-%04d", i)
		nodes[id] = &schema.Node{
			Id:        id,
			Name:      fmt.Sprintf("Sym%04d", i),
			Kind:      "function",
			FilePath:  "big.go",
			StartLine: int32(i + 1),
			EndLine:   int32(i + 1),
		}
	}
	e := NewWithRoot(&traverseFakeReader{nodes: nodes}, dir)

	got, err := e.FileSymbols("big.go")
	if err != nil {
		t.Fatalf("FileSymbols(big.go): unexpected error: %v", err)
	}
	if len(got.Symbols) != MaxFileSymbols {
		t.Fatalf("FileSymbols(big.go): len(Symbols) = %d, want exactly MaxFileSymbols (%d)", len(got.Symbols), MaxFileSymbols)
	}
	if got.Total <= int64(len(got.Symbols)) {
		t.Fatalf("FileSymbols(big.go): Total = %d, want strictly greater than the returned length %d", got.Total, len(got.Symbols))
	}
	if !got.Truncated {
		t.Fatalf("FileSymbols(big.go): Truncated = false, want true")
	}
}

// TestFileSymbolsDeterministicAcrossRepeatedCalls pins that two
// consecutive calls for the same path return the same list in the same
// order.
func TestFileSymbolsDeterministicAcrossRepeatedCalls(t *testing.T) {
	e := newFileSymbolsFixtureEngine(t)

	first, err := e.FileSymbols("three.go")
	if err != nil {
		t.Fatalf("FileSymbols (call 1): unexpected error: %v", err)
	}
	second, err := e.FileSymbols("three.go")
	if err != nil {
		t.Fatalf("FileSymbols (call 2): unexpected error: %v", err)
	}
	if len(first.Symbols) != len(second.Symbols) {
		t.Fatalf("FileSymbols: call 1 returned %d symbols, call 2 returned %d — want identical", len(first.Symbols), len(second.Symbols))
	}
	for i := range first.Symbols {
		if first.Symbols[i].Id != second.Symbols[i].Id {
			t.Fatalf("FileSymbols: order differs at index %d: call 1 = %q, call 2 = %q", i, first.Symbols[i].Id, second.Symbols[i].Id)
		}
	}
}
