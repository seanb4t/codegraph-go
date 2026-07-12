// Package routes implements LANG-07's per-framework route-detector
// registry (D-08): a small, opt-in-per-detected-dependency (D-09) set of
// AST-based detectors — Gin (Go), Spring (Java), ASP.NET (C#),
// Django/Flask/FastAPI (Python), Express/NestJS (TypeScript/JavaScript) —
// each scanning one already-parsed file's syntax tree for that framework's
// route-declaration shape and resolving the route's handler to an
// already-known function/method node id.
//
// This package deliberately owns NO indexer-package types (avoiding an
// import cycle: internal/indexer wires this registry in, not the other
// way around) and does NO manifest reading, NO parsing, and NO node/edge
// construction of its own — those are the caller's job (routes_detect.go
// in internal/indexer), which already has repoRoot, DiscoveredFile, and
// goextract.FileResult in scope. A Detector's Signature is handed the
// caller's already-read manifest text (D-09: "do not re-read the manifest
// per detector" — the caller reads each language's manifest exactly once,
// regardless of how many detectors share that language), and its Walk is
// handed an already-parsed root node plus a HandlerResolver the caller
// builds from that file's own (and, as a fallback, its whole language's)
// already-extracted function/method nodes.
package routes

import tree_sitter "github.com/tree-sitter/go-tree-sitter"

// HandlerResolver resolves a detected route's handler reference to an
// already-known symbol node id (D-08: "resolve the handler argument to its
// symbol node via the symbol index"). A route whose handler cannot be
// resolved is never emitted with a dangling edge (D-06a) — Walk
// implementations MUST treat a false ok as "skip this match", never
// synthesize a placeholder id.
type HandlerResolver interface {
	// ResolveByName resolves a bare identifier (e.g. a Gin/Express handler
	// argument, or a Django path()/re_path() view reference) to a
	// function/method node id — same-file first, falling back to a
	// cross-file, same-language lookup (the caller's global index) so a
	// handler declared in a different file (Django's views.py, an Express
	// route file split from its handler module) still resolves. Ambiguous
	// same-name matches are resolved deterministically by the caller
	// (lowest node id wins), never last-write-wins.
	ResolveByName(name string) (id string, ok bool)

	// ResolveByLine resolves the function/method node whose OWN declaration
	// starts at line (1-based) — used by every annotation/decorator-based
	// framework (Spring, ASP.NET, Flask/FastAPI, NestJS), where the route's
	// "handler" IS the annotated/decorated declaration itself, not a
	// separate call argument. line must be the ENCLOSING
	// method/function-declaration node's own StartPosition().Row+1 (the
	// same position each per-language extractor already records as that
	// node's StartLine), not the annotation/decorator's own line — those
	// differ for Java/C# (attribute is part of the declaration's span) vs.
	// Python/TS (decorator is a separate preceding sibling).
	ResolveByLine(line int32) (id string, ok bool)
}

// Route is one detected framework route, resolved against a
// HandlerResolver by the Detector's own Walk implementation. The caller
// (routes_detect.go) turns a non-skipped Route into a schema.Node (Kind
// goextract.KindRoute) and a schema.Edge (Kind goextract.RefKindCalls,
// Provenance "heuristic") — Detector implementations never construct
// schema types directly, keeping this package free of a schema/goextract
// dependency.
type Route struct {
	// HTTPMethod is the route's HTTP verb ("GET", "POST", ... or "ANY" for
	// a framework construct with no single verb, e.g. Django's path()).
	HTTPMethod string
	// Path is the route's declared path/pattern, exactly as written in
	// source (no normalization — "/users/:id", "/users/{id}",
	// "users/<int:id>" are all left in their own framework's syntax).
	Path string
	// HandlerID is the resolved handler's node id (via HandlerResolver).
	// Empty means "could not resolve" — the caller MUST skip emitting a
	// route for a Route with an empty HandlerID (D-06a: never a dangling
	// edge).
	HandlerID string
	// Line and Col are the route's own declaration site (the call
	// expression's or annotation/decorator's start position) — carried
	// onto the synthesized schema.Edge's Line/Col (RES-03).
	Line, Col int32
}

// Detector is a per-framework, opt-in AST-based route detector (D-08).
type Detector struct {
	// ID names the framework, e.g. "gin-route", "spring-route" — the value
	// the caller stamps onto the synthesized edge's
	// Metadata["synthesizedBy"] (RES-03).
	ID string
	// Language is the LanguageSpec.ID this detector's Walk expects the
	// already-parsed root node to have come from ("go", "java", "csharp",
	// "python", "typescript", "tsx", "javascript").
	Language string
	// Signature reports whether manifestText — the caller's already-read,
	// raw manifest content for this Language — indicates this framework is
	// a dependency (D-09's opt-in gate). Never re-reads a manifest file
	// itself.
	Signature func(manifestText string) bool
	// Walk scans one file's already-parsed root node for this framework's
	// route-declaration shape, resolving each match's handler via
	// resolve and returning one Route per successfully resolved match
	// (never a Route with an empty HandlerID — filter those out before
	// returning, matching this package's own HandlerResolver contract).
	Walk func(root *tree_sitter.Node, src []byte, resolve HandlerResolver) []Route
}

// registry is package-level, populated once via each detector file's own
// init() call to Register — the same "registry-keyed-by-ID, never rebuilt
// per call" shape languages.go and symbolindex.go already established
// (05-PATTERNS.md §Registry-keyed-by-ID). Detectors are appended in
// init()-registration order (deterministic: Go runs a package's init()
// functions in the file names' lexical order), so Registered()'s output
// order is itself deterministic run-to-run without an explicit sort.
var registry []Detector

// Register appends d to the package-level detector registry. Called from
// each per-framework file's own init() (gin.go, spring.go, aspnet.go,
// django.go, express.go).
func Register(d Detector) {
	registry = append(registry, d)
}

// Registered returns a defensive copy of every registered Detector, in
// registration order.
func Registered() []Detector {
	out := make([]Detector, len(registry))
	copy(out, registry)
	return out
}
