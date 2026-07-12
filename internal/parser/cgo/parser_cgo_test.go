package cgo

import (
	"bytes"
	"errors"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/parser"
)

func TestCGoParsesGoSource(t *testing.T) {
	p, err := NewGoParser()
	if err != nil {
		t.Fatalf("NewGoParser: %v", err)
	}
	defer p.Close()

	src := []byte("package main\n\nfunc main() {}\n")
	tree, err := p.Parse(src, nil)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if tree == nil {
		t.Fatal("expected a non-nil tree for valid Go source")
	}
	defer tree.Close()
}

func TestCGoParsesPythonSource(t *testing.T) {
	p, err := NewPythonParser()
	if err != nil {
		t.Fatalf("NewPythonParser: %v", err)
	}
	defer p.Close()

	// A small snippet that exercises Python's external INDENT/DEDENT
	// scanner, per RESEARCH's note that Python carries a real external C
	// scanner (D-05).
	src := []byte("def f():\n    return 1\n")
	tree, err := p.Parse(src, nil)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if tree == nil {
		t.Fatal("expected a non-nil tree for valid Python source")
	}
	defer tree.Close()
}

func TestCGoParseRejectsOversizeInput(t *testing.T) {
	p, err := NewGoParser()
	if err != nil {
		t.Fatalf("NewGoParser: %v", err)
	}
	defer p.Close()

	oversize := bytes.Repeat([]byte("a"), parser.MaxSourceBytes+1)
	tree, err := p.Parse(oversize, nil)
	if tree != nil {
		t.Fatalf("expected nil tree for oversize input, got %v", tree)
	}
	if !errors.Is(err, parser.ErrSourceTooLarge) {
		t.Fatalf("expected ErrSourceTooLarge, got %v", err)
	}
}

func TestCGoCloseIsSafeToCallOnce(t *testing.T) {
	p, err := NewGoParser()
	if err != nil {
		t.Fatalf("NewGoParser: %v", err)
	}
	if err := p.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestCGoTreeCloseIsSafeToCallTwice(t *testing.T) {
	p, err := NewGoParser()
	if err != nil {
		t.Fatalf("NewGoParser: %v", err)
	}
	defer p.Close()

	src := []byte("package main\n\nfunc main() {}\n")
	tree, err := p.Parse(src, nil)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	// Proves WR-05's fix: closing the same *parser.Tree twice must not
	// double-free the underlying native C tree (go-tree-sitter's
	// Tree.Close() unconditionally calls ts_tree_delete with no nil
	// guard on repeat calls).
	if err := tree.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := tree.Close(); err != nil {
		t.Fatalf("second Close: want nil, got %v", err)
	}
}

// TestCGoParsesPriority4Sources proves each of the five priority-4 grammar
// constructors (D-01, LANG-02/03/05) routes through newCGoParser and its
// Parse seam, parsing a trivial valid source snippet without error.
func TestCGoParsesPriority4Sources(t *testing.T) {
	cases := []struct {
		name  string
		newFn func() (*CGoParser, error)
		src   []byte
	}{
		{
			name:  "Java",
			newFn: NewJavaParser,
			src:   []byte("class Main { void f() {} }\n"),
		},
		{
			name:  "CSharp",
			newFn: NewCSharpParser,
			src:   []byte("class Main { void F() {} }\n"),
		},
		{
			name:  "JavaScript",
			newFn: NewJavaScriptParser,
			src:   []byte("function f() { return 1; }\n"),
		},
		{
			name:  "TypeScript",
			newFn: NewTypeScriptParser,
			src:   []byte("function f(): number { return 1; }\n"),
		},
		{
			name:  "TSX",
			newFn: NewTSXParser,
			src:   []byte("const el = <div>hi</div>;\n"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, err := tc.newFn()
			if err != nil {
				t.Fatalf("%s: constructor: %v", tc.name, err)
			}
			if p == nil {
				t.Fatalf("%s: expected a non-nil *CGoParser", tc.name)
			}
			defer p.Close()

			tree, err := p.Parse(tc.src, nil)
			if err != nil {
				t.Fatalf("%s: Parse: %v", tc.name, err)
			}
			if tree == nil {
				t.Fatalf("%s: expected a non-nil tree for valid source", tc.name)
			}
			defer tree.Close()
		})
	}
}

var _ parser.Parser = (*CGoParser)(nil)
