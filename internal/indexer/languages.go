package indexer

import (
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/parser"
)

// ProjectDescriptor resolves a repo's per-language module/namespace
// identity (go.mod, pom.xml, *.csproj, package.json+tsconfig.json,
// pyproject.toml, ...), parsed once per repo root (D-03). Go's existing
// go.mod path resolution (languages_go.go) is the first implementation of
// this hook; Wave 2 (discover.go's generalization) adds the remaining
// per-language descriptor parsers behind this same interface.
type ProjectDescriptor interface {
	// ModulePath returns the descriptor's resolved base identity — a Go
	// module path today; a Java/C# root package/namespace, a Python
	// project root, or a TS/JS resolved package name for later languages.
	ModulePath() string
}

// LanguageSpec is the single source of truth for one language's
// parser+extractor+ModuleKey selection (D-01). Every subsequent wave
// (discovery, extract, resolve, per-language extractors, dispatch,
// routing) reads this registry — nothing else can land before it.
type LanguageSpec struct {
	// ID is the language's stable registry key ("go", "java", "csharp",
	// "python", "typescript", ...).
	ID string

	// Extensions are the file extensions (including the leading ".") this
	// language claims during discovery.
	Extensions []string

	// NewParser constructs a fresh parser.Parser for this language, routed
	// through the parser.Parser seam (and, for the CGo backend, the
	// MaxSourceBytes ceiling in newCGoParser/Parse — Security Domain V5).
	NewParser func() (parser.Parser, error)

	// Extract walks one file's already-parsed syntax tree into a
	// goextract.FileResult. moduleKey is this language's cross-file
	// symbol-index key for the file (Go's importPath is the first
	// instance of this concept; other languages compute a structurally
	// different key via ModuleKey below).
	Extract func(p parser.Parser, moduleKey, relPath string, src []byte) (goextract.FileResult, error)

	// ModuleKey computes this language's cross-file symbol-index key for a
	// discovered file, given the repo's resolved ProjectDescriptor (which
	// may be nil if this language's descriptor could not be resolved — a
	// file whose language has no descriptor still gets extracted with
	// path-based identity per D-03, so implementations must tolerate a
	// nil descriptor rather than panicking).
	ModuleKey func(descriptor ProjectDescriptor, relPath string) string

	// Descriptor parses this language's manifest (go.mod, pom.xml,
	// *.csproj, pyproject.toml, package.json+tsconfig.json, ...) once per
	// repo root to resolve module/namespace identity for the whole repo.
	Descriptor func(root string) (ProjectDescriptor, error)
}

// registry and extToLang are package-level, looked up by a stable string
// ID (or extension), never rebuilt per call — the same "registry-keyed-by-
// ID" shape already established by parser/cgo's per-language constructors
// and symbolindex.go's map-based indexing (PATTERNS.md §Registry-keyed-by-ID).
var (
	registry  = map[string]LanguageSpec{}
	extToLang = map[string]string{}
)

// registerLanguage indexes spec by its ID and by each of its Extensions.
// Called from each language's own init() (see languages_go.go).
func registerLanguage(spec LanguageSpec) {
	registry[spec.ID] = spec
	for _, ext := range spec.Extensions {
		extToLang[ext] = spec.ID
	}
}

// lookupLanguageByID returns the registered LanguageSpec for id, if any.
func lookupLanguageByID(id string) (LanguageSpec, bool) {
	spec, ok := registry[id]
	return spec, ok
}

// lookupLanguageByExt returns the registered LanguageSpec whose Extensions
// includes ext (including the leading "."), if any.
func lookupLanguageByExt(ext string) (LanguageSpec, bool) {
	id, ok := extToLang[ext]
	if !ok {
		return LanguageSpec{}, false
	}
	return lookupLanguageByID(id)
}
