// Package goldenspec is the single authoritative capture spec shared by
// testdata/golden/gocapture (the generator, package main, invoked with
// `go run ./testdata/golden/gocapture`) and the byte-identity oracle in
// testdata/golden/byte_identity_test.go (package golden, invoked with
// `go test ./testdata/golden/`). It must NOT live under testdata/ — the go
// tool ignores directories named "testdata" when expanding `...`, and an
// import path resolving into one is not a shape worth depending on. Both
// consumers import this package by its normal module path instead.
//
// Before this package existed, the locked capture arguments and the MCP
// call shape were declared TWICE — once in gocapture/main.go and once in
// testdata/golden/behavioral_test.go — and a third copy was about to be
// written for the oracle. A changed argument value in one copy could
// silently diverge from the others with nothing to catch it. Extracting
// these values here (moved, not retyped — the values are unchanged from
// the freeze) means there is exactly one authoritative table and one
// authoritative MCP call shape, and TestLockedCorpusArgsAreTheFrozenValues
// pins the table against a literal fixture so a silent edit fails a test
// instead of silently asking the oracle a different question than the
// generator asked (01-CONTEXT.md D-04, phase 01, plan 01-02).
package goldenspec

// PerCorpusArgs holds the symbol/query parameters used to capture one
// locked corpus's golden fixtures. Moved from
// testdata/golden/gocapture/main.go's unexported perCorpusArgs
// (2026-08-22); field values are unchanged from the freeze.
type PerCorpusArgs struct {
	BaselineSymbol     string
	BaselineSymbolFile string
	BaselineQuery      string
	MultiSymbol        string
	MultiQuery         string
}

// LockedCorpusArgs defines the committed symbol/query values per locked
// corpus. These produce the expected golden set per corpus: {explore,
// node, explore-multi, node-multi, explore-mcp, node-mcp} — 6 goldens.
//
// Moved verbatim from testdata/golden/gocapture/main.go's unexported
// lockedCorpusArgs (originally declared at main.go:73-102, moved
// 2026-08-22). These values are frozen alongside the goldens they
// produced in testdata/golden/corpus/ — changing one here without
// regenerating every golden it feeds is a defect, which is exactly what
// TestLockedCorpusArgsAreTheFrozenValues in spec_test.go exists to catch.
var LockedCorpusArgs = map[string]PerCorpusArgs{
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

// LanguageToLockedSlug is the EXPLICIT committed language->locked-slug map
// (H3). hugo supplies the tsjs leg from its JS files even though its
// manifest language is "go". Shared by the gocapture resolver, the
// hermetic test resolver (lockedCorpusDir), and the completeness guard.
//
// Moved verbatim from testdata/golden/gocapture/main.go's unexported
// languageToLockedSlug (2026-08-22) and from the second copy independently
// declared in testdata/golden/behavioral_test.go.
var LanguageToLockedSlug = map[string]string{
	"go":     "hugo",
	"tsjs":   "hugo",
	"java":   "guava",
	"csharp": "serilog",
	"python": "requests",
}

// SlugToRepo maps each locked-slug to its manifest repo slug for lookup.
//
// Moved verbatim from testdata/golden/gocapture/main.go's unexported
// slugToRepo (2026-08-22) and from the second copy independently declared
// in testdata/golden/behavioral_test.go.
var SlugToRepo = map[string]string{
	"hugo":     "gohugoio/hugo",
	"guava":    "google/guava",
	"serilog":  "serilog/serilog",
	"requests": "psf/requests",
}

// GoldenCapture mirrors the wrap_text envelope shape
// ({"command": ..., "output": ...}), so go-*.json fixtures are
// structurally identical to their siblings and can be loaded with a
// loadGoldenOutputIn-style helper.
//
// Moved verbatim, fields and JSON tags unchanged, from
// testdata/golden/gocapture/main.go's unexported goldenCapture
// (2026-08-22) and from the second copy independently declared in
// testdata/golden/behavioral_test.go.
type GoldenCapture struct {
	Command string `json:"command"`
	Output  string `json:"output"`
}
