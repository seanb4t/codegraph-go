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

var _ parser.Parser = (*CGoParser)(nil)
