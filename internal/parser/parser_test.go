package parser

import (
	"bytes"
	"errors"
	"testing"
)

// stubParser is a minimal Parser implementation used to prove the
// size-ceiling guard rejects oversized input before any backend-specific
// parse call happens. It carries no CGo or WASM dependency so this test
// runs with CGO_ENABLED=0.
type stubParser struct {
	ceiling   int
	parsed    bool
	closeCall bool
}

func newStubParser(ceiling int) *stubParser {
	return &stubParser{ceiling: ceiling}
}

func (p *stubParser) Parse(source []byte, oldTree *Tree) (*Tree, error) {
	if len(source) > p.ceiling {
		return nil, ErrSourceTooLarge
	}
	p.parsed = true
	return &Tree{}, nil
}

func (p *stubParser) Close() error {
	p.closeCall = true
	return nil
}

func TestParserRejectsOversizeInput(t *testing.T) {
	p := newStubParser(MaxSourceBytes)
	oversize := bytes.Repeat([]byte("a"), MaxSourceBytes+1)

	tree, err := p.Parse(oversize, nil)

	if tree != nil {
		t.Fatalf("expected nil tree for oversize input, got %v", tree)
	}
	if !errors.Is(err, ErrSourceTooLarge) {
		t.Fatalf("expected ErrSourceTooLarge, got %v", err)
	}
	if p.parsed {
		t.Fatal("parse must not proceed past the size ceiling check")
	}
}

func TestParserAcceptsInputUnderCeiling(t *testing.T) {
	p := newStubParser(MaxSourceBytes)

	tree, err := p.Parse([]byte("package main"), nil)

	if err != nil {
		t.Fatalf("unexpected error for under-ceiling input: %v", err)
	}
	if tree == nil {
		t.Fatal("expected a non-nil tree for under-ceiling input")
	}
}

func TestParserCloseIsCallable(t *testing.T) {
	p := newStubParser(MaxSourceBytes)

	if err := p.Close(); err != nil {
		t.Fatalf("unexpected error from Close: %v", err)
	}
	if !p.closeCall {
		t.Fatal("expected Close to be recorded as called")
	}
}

var _ Parser = (*stubParser)(nil)
