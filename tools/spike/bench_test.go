// Package main (test file): the D-05 head-to-head benchmark harness. Both
// backends are exercised through the exact same parser.Parser interface
// (D-05b) over the pinned corpus in testdata/ (see testdata/ATTRIBUTION.md),
// producing the real, measured numbers PARSER-DECISION.md folds in — this
// file intentionally does NOT hand-tune either backend beyond what the
// production interface exposes.
//
// Design note on per-iteration parser recreation: the wazero arm's package
// doc records that individual parsed trees cannot be freed (no ts_tree_delete
// export), so repeated Parse calls against one long-lived WParser
// accumulate guest memory. To keep both backends' benchmarks safe to run at
// whatever -benchtime the caller picks (and to keep the comparison fair —
// same lifecycle shape for both arms), every benchmark iteration here
// creates a fresh parser instance inside the b.StopTimer() window and closes
// it before the next iteration. Only the Parse call(s) under measurement run
// inside the timed window.
package main

import (
	"fmt"
	"testing"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/seanb4t/codegraph-go/internal/parser"
)

func benchFullParse(b *testing.B, backend, lang string) {
	var files []corpusFile
	switch lang {
	case "go":
		files = mustLoadGoCorpus()
	case "python":
		files = mustLoadPythonCorpus()
	default:
		b.Fatalf("unknown lang %q", lang)
	}
	if len(files) == 0 {
		b.Fatalf("empty corpus for %s/%s — check testdata/ population", backend, lang)
	}

	b.SetBytes(totalBytes(files))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		p, err := newParser(backend, lang)
		if err != nil {
			b.Fatalf("newParser(%s, %s): %v", backend, lang, err)
		}
		b.StartTimer()

		for _, f := range files {
			tree, err := p.Parse(f.Source, nil)
			if err != nil {
				b.Fatalf("Parse(%s): %v", f.Name, err)
			}
			closeTreeIfClosable(tree)
		}

		b.StopTimer()
		if err := p.Close(); err != nil {
			b.Fatalf("Close: %v", err)
		}
		b.StartTimer()
	}
}

func BenchmarkCGoFullParse_Go(b *testing.B)        { benchFullParse(b, "cgo", "go") }
func BenchmarkCGoFullParse_Python(b *testing.B)    { benchFullParse(b, "cgo", "python") }
func BenchmarkWazeroFullParse_Go(b *testing.B)     { benchFullParse(b, "wazero", "go") }
func BenchmarkWazeroFullParse_Python(b *testing.B) { benchFullParse(b, "wazero", "python") }

// closeTreeIfClosable frees a CGo-backed tree_sitter.Tree's C allocations.
// The wazero arm has no equivalent per-tree free (package doc: no
// ts_tree_delete export) — its memory is bounded instead by recreating the
// parser instance every benchmark iteration (see file doc above).
func closeTreeIfClosable(t *parser.Tree) {
	if native, ok := t.Inner().(*tree_sitter.Tree); ok {
		native.Close()
	}
}

// largestFile picks the biggest corpus file as the incremental-reparse
// baseline — a bigger tree makes the "reuse unaffected subtrees" benefit (or
// its absence, for the wazero arm's weaker hint-only path) more visible.
func largestFile(files []corpusFile) corpusFile {
	best := files[0]
	for _, f := range files[1:] {
		if len(f.Source) > len(best.Source) {
			best = f
		}
	}
	return best
}

// appendedSource returns base plus one trailing newline byte, and the
// tree-sitter Point/byte offset of the end of the ORIGINAL base content —
// the coordinates a real single-keystroke append-at-EOF edit needs.
func appendedSource(base []byte) (newSource []byte, endByte uint, endPoint tree_sitter.Point) {
	newSource = make([]byte, len(base)+1)
	copy(newSource, base)
	newSource[len(base)] = '\n'

	var row, col uint
	for _, ch := range base {
		if ch == '\n' {
			row++
			col = 0
		} else {
			col++
		}
	}
	return newSource, uint(len(base)), tree_sitter.Point{Row: row, Column: col}
}

// benchIncrementalReparseCGo exercises the RESEARCH "Code Examples" pattern
// (InputEdit + ChangedRanges) — a true incremental reparse that reuses
// unaffected subtrees from the baseline tree.
func benchIncrementalReparseCGo(b *testing.B, lang string) {
	var files []corpusFile
	switch lang {
	case "go":
		files = mustLoadGoCorpus()
	case "python":
		files = mustLoadPythonCorpus()
	default:
		b.Fatalf("unknown lang %q", lang)
	}
	base := largestFile(files).Source
	newSource, endByte, endPoint := appendedSource(base)

	b.SetBytes(int64(len(newSource)))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		p, err := newParser("cgo", lang)
		if err != nil {
			b.Fatalf("newParser: %v", err)
		}
		baseTree, err := p.Parse(base, nil)
		if err != nil {
			b.Fatalf("baseline Parse: %v", err)
		}
		native, ok := baseTree.Inner().(*tree_sitter.Tree)
		if !ok {
			b.Fatal("expected *tree_sitter.Tree from cgo backend")
		}
		edit := &tree_sitter.InputEdit{
			StartByte:      endByte,
			OldEndByte:     endByte,
			NewEndByte:     endByte + 1,
			StartPosition:  endPoint,
			OldEndPosition: endPoint,
			NewEndPosition: tree_sitter.Point{Row: endPoint.Row + 1, Column: 0},
		}
		native.Edit(edit)
		b.StartTimer()

		newTree, err := p.Parse(newSource, parser.NewTree(native))
		if err != nil {
			b.Fatalf("incremental Parse: %v", err)
		}

		b.StopTimer()
		newNative, ok := newTree.Inner().(*tree_sitter.Tree)
		if !ok {
			b.Fatal("expected *tree_sitter.Tree from cgo backend")
		}
		// Exercise the metric RESEARCH's Code Examples names explicitly —
		// the set of subtrees tree-sitter judges as affected by the edit.
		_ = native.ChangedRanges(newNative)
		native.Close()
		newNative.Close()
		if err := p.Close(); err != nil {
			b.Fatalf("Close: %v", err)
		}
		b.StartTimer()
	}
}

func BenchmarkCGoIncrementalReparse_Go(b *testing.B)     { benchIncrementalReparseCGo(b, "go") }
func BenchmarkCGoIncrementalReparse_Python(b *testing.B) { benchIncrementalReparseCGo(b, "python") }

// benchIncrementalReparseWazero exercises the wazero arm's WEAKER
// hint-only reparse path: the embedded WASM binary's ABI does not export
// ts_tree_edit (package doc), so there is no way to annotate the edit region
// before reparsing — the old tree pointer is passed as a bare hint to
// ts_parser_parse_string, which is not the same operation as the CGo arm's
// InputEdit-annotated incremental reparse above. This benchmark exists to
// measure exactly that gap, not to claim parity with it.
func benchIncrementalReparseWazero(b *testing.B, lang string) {
	var files []corpusFile
	switch lang {
	case "go":
		files = mustLoadGoCorpus()
	case "python":
		files = mustLoadPythonCorpus()
	default:
		b.Fatalf("unknown lang %q", lang)
	}
	base := largestFile(files).Source
	newSource, _, _ := appendedSource(base)

	b.SetBytes(int64(len(newSource)))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		p, err := newParser("wazero", lang)
		if err != nil {
			b.Fatalf("newParser: %v", err)
		}
		baseTree, err := p.Parse(base, nil)
		if err != nil {
			b.Fatalf("baseline Parse: %v", err)
		}
		b.StartTimer()

		if _, err := p.Parse(newSource, baseTree); err != nil {
			b.Fatalf("hint-only reparse: %v", err)
		}

		b.StopTimer()
		if err := p.Close(); err != nil {
			b.Fatalf("Close: %v", err)
		}
		b.StartTimer()
	}
}

func BenchmarkWazeroIncrementalReparse_Go(b *testing.B) { benchIncrementalReparseWazero(b, "go") }
func BenchmarkWazeroIncrementalReparse_Python(b *testing.B) {
	benchIncrementalReparseWazero(b, "python")
}

// TestCorpusSanity is not a benchmark: it fails fast (before any -bench run)
// if the embedded corpus is empty or a language's file set didn't survive
// go:embed, since a silently-empty corpus would make every benchmark above
// report a meaningless 0-byte throughput rather than an error.
func TestCorpusSanity(t *testing.T) {
	goFiles := mustLoadGoCorpus()
	pyFiles := mustLoadPythonCorpus()
	if len(goFiles) == 0 {
		t.Fatal("go corpus is empty — see testdata/ATTRIBUTION.md")
	}
	if len(pyFiles) == 0 {
		t.Fatal("python corpus is empty — see testdata/ATTRIBUTION.md")
	}
	t.Logf("go corpus: %d files, %s", len(goFiles), fmt.Sprintf("%d bytes", totalBytes(goFiles)))
	t.Logf("python corpus: %d files, %s", len(pyFiles), fmt.Sprintf("%d bytes", totalBytes(pyFiles)))
}
