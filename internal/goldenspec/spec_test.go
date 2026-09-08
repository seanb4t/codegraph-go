package goldenspec

import (
	"reflect"
	"testing"
)

// wantLockedCorpusArgs is a literal fixture of LockedCorpusArgs's expected
// value, independent of the package's own declaration, so a silent edit
// to LockedCorpusArgs fails this test instead of silently asking the
// byte-identity oracle a different question than testdata/golden/gocapture
// asked.
//
// Provenance: transcribed from testdata/golden/gocapture/main.go's
// unexported lockedCorpusArgs, originally declared at main.go lines
// 73-102, on 2026-08-22 (the same commit that moved the map into this
// package as LockedCorpusArgs). Re-verify against the golden capture
// command that produced testdata/golden/corpus/ before editing this
// fixture — a stale fixture here would make this guard assert the wrong
// thing rather than fail loudly.
var wantLockedCorpusArgs = map[string]PerCorpusArgs{
	"hugo": {
		BaselineSymbol:     "Page",
		BaselineSymbolFile: "",
		BaselineQuery:      "page content",
		MultiSymbol:        "Site",
		MultiQuery:         "page content template",
	},
	"guava": {
		BaselineSymbol:     "Preconditions",
		BaselineSymbolFile: "",
		BaselineQuery:      "check precondition",
		MultiSymbol:        "ImmutableList",
		MultiQuery:         "immutable collection",
	},
	"serilog": {
		BaselineSymbol:     "LoggerConfiguration",
		BaselineSymbolFile: "",
		BaselineQuery:      "configure logger",
		MultiSymbol:        "LogEvent",
		MultiQuery:         "log configuration",
	},
	"requests": {
		BaselineSymbol:     "Session",
		BaselineSymbolFile: "",
		BaselineQuery:      "http session",
		MultiSymbol:        "Request",
		MultiQuery:         "http request session",
	},
}

// TestLockedCorpusArgsAreTheFrozenValues asserts LockedCorpusArgs equals
// the literal fixture above by full value equality, and separately
// asserts its length is exactly 4. This is what makes a silent argument
// change in LockedCorpusArgs fail a test rather than silently changing
// what both the generator and the byte-identity oracle ask of a live
// Engine.
func TestLockedCorpusArgsAreTheFrozenValues(t *testing.T) {
	if len(LockedCorpusArgs) != 4 {
		t.Fatalf("len(LockedCorpusArgs) = %d, want 4", len(LockedCorpusArgs))
	}
	if !reflect.DeepEqual(LockedCorpusArgs, wantLockedCorpusArgs) {
		t.Fatalf("LockedCorpusArgs diverged from the frozen fixture:\ngot:  %+v\nwant: %+v", LockedCorpusArgs, wantLockedCorpusArgs)
	}
}
