// Package capability is the D-11 language capability descriptor: the
// machine-readable half of the "language capability matrix" (the other
// half is the human-readable docs/LANGUAGE-CAPABILITY-MATRIX.md, which this
// package's own consistency test — matrix_test.go — keeps mirrored to this
// file exactly, coverage value for coverage value, gap for gap).
//
// D-11 (05-CONTEXT.md): "Ship a language capability matrix — both a
// committed human-readable doc and a machine-readable capability
// descriptor per language (extraction / resolution / dispatch / routing:
// full | partial | none). Priority-4 = full across the board; mainstream-6
// = extraction + best-effort resolution, every gap named. 'Documented-
// partial' (LANG-06) means the gap is written down in the matrix, not
// silently missing."
//
// Every entry below is populated from the ACTUAL coverage each language's
// own plan SUMMARY reported (05-04..05-12), not from aspiration — a "full"
// value is never claimed unless that plan's own SUMMARY (and, for
// priority-4, a corresponding golden-parity test in testdata/golden)
// backs it up. Gaps is deliberately not restricted to non-full axes only:
// an otherwise-"full" axis may still carry a documented heuristic boundary
// (e.g. Java's PascalCase/camelCase call-qualifier disambiguation) — D-11's
// "documented-partial means written down" discipline applies to every
// honestly-discovered limitation, not only ones severe enough to drop a
// coverage value.
package capability

// Coverage is the full|partial|none coverage level for one capability axis.
type Coverage string

const (
	// CoverageFull means this axis is validated end-to-end for this
	// language: priority-4 languages via a corresponding golden-parity
	// test (testdata/golden), mainstream-tier languages via their own
	// extraction/resolution test suite reaching the full bar for that
	// specific axis (e.g. a mainstream language's Dispatch axis is "full"
	// only when it both emits an interface-shaped node AND resolves the
	// declared-conformance reference within its own documented resolution
	// scope).
	CoverageFull Coverage = "full"
	// CoveragePartial means this axis works within an explicitly bounded,
	// named scope (see Gaps) — extraction/resolution/dispatch/routing is
	// real and tested, but does not cover every case a full-fidelity
	// implementation would.
	CoveragePartial Coverage = "partial"
	// CoverageNone means this axis does not apply or is not implemented
	// for this language — always paired with a named gap explaining why
	// (e.g. "no framework route detector exists for this language" or
	// "this language has no declared-interface construct").
	CoverageNone Coverage = "none"
)

// valid reports whether c is one of the three sanctioned Coverage values.
func (c Coverage) valid() bool {
	switch c {
	case CoverageFull, CoveragePartial, CoverageNone:
		return true
	default:
		return false
	}
}

// CapabilityEntry describes one language's D-11 coverage across the four
// capability axes tracked since Phase 5:
//
//   - Extraction: does this language's extractor turn source into the
//     shared codegraph vocabulary (nodes + intra-file edges)?
//   - Resolution: does cross-file symbol resolution connect a reference
//     (call/embeds) to its target declaration?
//   - Dispatch: does interface->implementation dispatch synthesis (RES-02,
//     05-09) apply to this language, and does it work within its declared
//     scope? A language with no declared-interface construct is "none",
//     not "full" — dispatch never fires, there is nothing to be full of.
//   - Routing: does a framework-aware route detector (LANG-07, 05-12)
//     exist and fire for this language?
//
// Gaps names every documented limitation this language's own plan SUMMARY
// recorded — see the package doc comment for why this is not restricted to
// non-full axes only.
type CapabilityEntry struct {
	Extraction Coverage
	Resolution Coverage
	Dispatch   Coverage
	Routing    Coverage
	Gaps       []string
}

// matrix is the package-level D-11 capability descriptor, keyed by the
// same language ID the LanguageSpec registry (internal/indexer/languages.go)
// uses. Populated from 05-04 (Java), 05-05 (C#), 05-06 (Python), 05-07
// (TypeScript/TSX/JavaScript), 05-09 (dispatch synthesis), 05-10
// (Rust/Ruby/PHP), 05-11 (C/C++/Swift/Kotlin), and 05-12 (framework
// routing)'s own per-language coverage assessments.
var matrix = map[string]CapabilityEntry{
	// --- Go (reference implementation, Phases 1-3) ---
	"go": {
		Extraction: CoverageFull,
		Resolution: CoverageFull,
		Dispatch:   CoverageFull,
		Routing:    CoverageFull,
	},

	// --- Priority-4 (LANG-02..05): full extraction+resolution+dispatch,
	// validated on real repos or (where no live TS CodeGraph v1.3.x CLI
	// was available) via RESEARCH's documented source-as-specification +
	// self-consistency fallback. ---
	"java": {
		Extraction: CoverageFull,
		Resolution: CoverageFull,
		Dispatch:   CoverageFull,
		Routing:    CoverageFull,
		Gaps: []string{
			"Same-package qualified call qualifiers are disambiguated from local-variable receivers via a PascalCase/camelCase naming-convention heuristic (no local-variable type table is tracked).",
			"No live TS CodeGraph v1.3.x CLI or curated Java corpus was available in this environment; TestCorpusBehavior_Java validates via the documented source-as-specification plus self-consistency fallback rather than a byte/shape diff against captured TS output.",
		},
	},
	"csharp": {
		Extraction: CoverageFull,
		Resolution: CoverageFull,
		Dispatch:   CoverageFull,
		Routing:    CoverageFull,
		Gaps: []string{
			"A bare using-imported cross-namespace call or inheritance reference — C#'s dominant idiom — is an accepted, documented gap: only fully-qualified cross-namespace references and same-namespace references resolve, since no global symbol table is built at parse time.",
			"partial class/struct/record/interface fragments share one node keyed by (qualifiedName, namespace) with a deterministic sentinel FilePath/StartLine, rather than a resolve.go-coordinated first-fragment-by-path tie-break (Pitfall 5, scheme-b variant).",
			"No live TS CodeGraph v1.3.x CLI or curated C# corpus was available in this environment; TestCorpusBehavior_CSharp validates via the same source-as-specification plus self-consistency fallback as Java.",
		},
	},
	"python": {
		Extraction: CoverageFull,
		Resolution: CoverageFull,
		Dispatch:   CoverageNone,
		Routing:    CoverageFull,
		Gaps: []string{
			"pyextract emits no KindInterface nodes — Python has no declared-interface construct, so base-class RefKindEmbeds refs never promote to an implements edge; interface->implementation dispatch traversal does not apply to Python.",
			"A plain unaliased `import foo.bar` populates no Imports entry (Python's own binding semantics bind only the top-level name); only an aliased plain import or a from-import populates Imports.",
			"Wildcard from-imports (`from x import *`) are not resolved.",
			"No live TS CodeGraph v1.3.x CLI or curated Python golden corpus is committed to this repo; TestCorpusBehavior_Python self-skips by default (it was smoke-tested this session against a real 168-file corpus, see 05-06-SUMMARY.md).",
		},
	},
	"typescript": {
		Extraction: CoverageFull,
		Resolution: CoverageFull,
		Dispatch:   CoverageFull,
		Routing:    CoverageFull,
		Gaps: []string{
			"A renamed default import (`import Renamed from './foo'`) is an accepted gap — a default import only resolves when the local binding text coincides with the target's own declared symbol name.",
			"Directory-style imports (`./utils` resolving to `utils/index.ts`) and node_modules/package.json main/exports-map resolution are not implemented — only relative-specifier and tsconfig paths/baseUrl-aliased specifiers resolve.",
			"No live TS CodeGraph v1.3.x CLI or curated TS/JS golden corpus is committed to this repo; TestCorpusBehavior_TSJS self-skips by default (it was smoke-tested this session against a real 13,464-file corpus, see 05-07-SUMMARY.md).",
		},
	},
	"tsx": {
		Extraction: CoverageFull,
		Resolution: CoverageFull,
		Dispatch:   CoverageFull,
		Routing:    CoverageFull,
		Gaps: []string{
			"A renamed default import (`import Renamed from './foo'`) is an accepted gap — a default import only resolves when the local binding text coincides with the target's own declared symbol name.",
			"Directory-style imports (`./utils` resolving to `utils/index.ts`) and node_modules/package.json main/exports-map resolution are not implemented — only relative-specifier and tsconfig paths/baseUrl-aliased specifiers resolve.",
			"No live TS CodeGraph v1.3.x CLI or curated TS/JS golden corpus is committed to this repo; TestCorpusBehavior_TSJS self-skips by default (it was smoke-tested this session against a real 13,464-file corpus, see 05-07-SUMMARY.md).",
		},
	},
	"javascript": {
		Extraction: CoverageFull,
		Resolution: CoverageFull,
		Dispatch:   CoverageNone,
		Routing:    CoverageFull,
		Gaps: []string{
			"JavaScript's own grammar has no interface_declaration node and no implements clause — class heritage is extends-only, so RefKindEmbeds refs never resolve to an interface target and never promote to an implements edge for pure-JavaScript files.",
			"A renamed default import (`import Renamed from './foo'`) is an accepted gap — a default import only resolves when the local binding text coincides with the target's own declared symbol name.",
			"Directory-style imports and node_modules/package.json main/exports-map resolution are not implemented — only relative-specifier and tsconfig paths/baseUrl-aliased specifiers resolve.",
			"No live TS CodeGraph v1.3.x CLI or curated TS/JS golden corpus is committed to this repo; TestCorpusBehavior_TSJS self-skips by default (it was smoke-tested this session against a real 13,464-file corpus, see 05-07-SUMMARY.md).",
		},
	},

	// --- Mainstream-6 (LANG-06): full extraction, best-effort resolution,
	// every gap named (D-04/D-11's "documented-partial" tier). ---
	"rust": {
		Extraction: CoverageFull,
		Resolution: CoveragePartial,
		Dispatch:   CoveragePartial,
		Routing:    CoverageNone,
		Gaps: []string{
			"`use` declarations are extracted as dependency refs but deliberately never populate Imports — the enclosing crate name needed to compute a matching moduleKey is not available at Extract time, so cross-crate/cross-module use-qualified call and trait resolution does not work; only same-file and same-computed-module-path resolution succeeds.",
			"Nested `mod { }` blocks are not descended into; macro-generated items are invisible; generic_function/macro_invocation callees are unresolved; trait default-method inheritance is not resolved (no conformance-retry pass for Rust).",
			"No framework route detector exists for Rust — LANG-07 covers only the five priority frameworks (Gin/Spring/ASP.NET/Django-Flask-FastAPI/Express-NestJS).",
		},
	},
	"ruby": {
		Extraction: CoverageFull,
		Resolution: CoveragePartial,
		Dispatch:   CoverageNone,
		Routing:    CoverageNone,
		Gaps: []string{
			"Ruby's moduleKey is per-file (like Python's), so there is no same-module (directory-shared) resolution tier; require binds no local alias, and require_relative populates Imports but only helps when the required file's basename happens to textually match the referenced constant.",
			"A bare, no-parens, no-argument method call (`helper`) is grammatically ambiguous with a local-variable reference in tree-sitter-ruby's own grammar — a verified real grammar limitation, not an oversight (`helper()`, with parens, extracts correctly).",
			"`include`/`extend`/`prepend` mixins are not extracted as edges; dynamic method definition (`define_method`, `method_missing`) is invisible; Ruby has no declared-interface construct in this extractor, so dispatch synthesis never fires.",
			"No framework route detector exists for Ruby (e.g. Rails) — LANG-07 covers only the five priority frameworks.",
		},
	},
	"php": {
		Extraction: CoverageFull,
		Resolution: CoveragePartial,
		Dispatch:   CoveragePartial,
		Routing:    CoverageNone,
		Gaps: []string{
			"Cross-module resolution works only when a use-imported simple name's target moduleKey matches exactly; a fully-qualified inline reference (`extends \\App\\Base`) is recorded by simple name only, not as an alternate lookup key.",
			"Trait inclusion (`use TraitName;` inside a class body) is not extracted as an edge; `object_creation_expression` (`new Widget()`) is not tracked as a calls edge; magic methods (`__call`, `__get`) are invisible.",
			"A namespace-less file falls back to a less-precise composer.json PSR-4/path-derived moduleKey.",
			"No framework route detector exists for PHP (e.g. Laravel/Symfony) — LANG-07 covers only the five priority frameworks.",
		},
	},
	"c": {
		Extraction: CoverageFull,
		Resolution: CoveragePartial,
		Dispatch:   CoverageNone,
		Routing:    CoverageNone,
		Gaps: []string{
			"#include never populates Imports — there is no cross-file call resolution for C; only same-file (unqualified) calls resolve.",
			"Anonymous structs, preprocessor-macro-generated symbols, and multi-declarator declaration lines (only the first declarator is considered) are not extracted.",
			"C has no interface/dispatch construct — not applicable to a procedural language; no implements synthesis is attempted.",
			"No framework route detector exists for C — LANG-07 covers only the five priority frameworks.",
			"`.h` files default to the C grammar (a documented default disposition for the C/C++ header-extension ambiguity) — a C++-only header using C++-specific syntax will misparse under the C grammar unless renamed .hpp/.hh.",
		},
	},
	"cpp": {
		Extraction: CoverageFull,
		Resolution: CoveragePartial,
		Dispatch:   CoverageNone,
		Routing:    CoverageNone,
		Gaps: []string{
			"#include never populates Imports; an out-of-line method whose qualifying type lives in a different file resolves via the same cross-file RefKindContains fallback rustextract's impl blocks use, but namespace-qualified name disambiguation does not work since namespace bodies are transparently flattened.",
			"Template declarations/instantiations, pure-virtual/bodyless method prototypes, operator overloads, and qualified-identifier base classes (`class A : ns::Base`) are not extracted/resolved.",
			"cextract never emits KindInterface for C++ (no interface/pure-virtual distinction) — base_class_clause embeds edges never promote to an implements edge.",
			"No framework route detector exists for C++ — LANG-07 covers only the five priority frameworks.",
			"`.h` files default to the C grammar (a documented default disposition for the C/C++ header-extension ambiguity); C++ claims only unambiguous extensions (.cpp/.cc/.cxx/.hpp/.hh).",
		},
	},
	"swift": {
		Extraction: CoverageFull,
		Resolution: CoveragePartial,
		Dispatch:   CoveragePartial,
		Routing:    CoverageNone,
		Gaps: []string{
			"No module-scoped symbol index exists at this tier — the SPM Sources/<Target>/... ModuleKey is a discovery-time placeholder only; import never populates Imports, so only same-file resolution succeeds.",
			"Extension member extraction/merge is not implemented (an extension's own member declarations are recognized but never walked); protocol-witness resolution, enum cases/associated values, property declarations, and generic constraints are not extracted.",
			"self-qualified calls are not specially recognized — routed through the generic local-alias path like any other lowercase receiver.",
			"swift is a `[SUS]`-tier community grammar (alex-pinkus/tree-sitter-swift), pinned by exact commit only after a blocking human-verify supply-chain checkpoint (05-08); its function_declaration binds its own \"name\" field twice (once to the real identifier, once to the return-type node) — a verified grammar rough edge this extractor relies on tree-sitter's first-bound-value behavior to work around.",
			"No framework route detector exists for Swift (e.g. Vapor) — LANG-07 covers only the five priority frameworks.",
		},
	},
	"kotlin": {
		Extraction: CoverageFull,
		Resolution: CoveragePartial,
		Dispatch:   CoveragePartial,
		Routing:    CoverageNone,
		Gaps: []string{
			"Cross-module resolution works only when an import-qualified simple name's target package computes a matching key; there is no fully-qualified-inline-reference alternate lookup.",
			"Extension functions, companion objects, and `object` singleton declarations are not extracted; data-class auto-generated members and property declarations are not extracted.",
			"this-qualified calls are not specially recognized — routed through the generic local-alias path like any other lowercase receiver.",
			"kotlin is a `[SUS]`-tier community grammar (tree-sitter-grammars/tree-sitter-kotlin@v1.1.0), pinned by exact semver only after a blocking human-verify supply-chain checkpoint (05-08), replacing an originally-approved fwcd fork that failed to build.",
			"No framework route detector exists for Kotlin (e.g. Ktor, or Spring Boot written in Kotlin — the Spring detector's opt-in Signature is gated on the \"java\" LanguageSpec only) — LANG-07 covers only the five priority frameworks.",
		},
	},
}

// Lookup returns the CapabilityEntry registered for id, if any.
func Lookup(id string) (CapabilityEntry, bool) {
	e, ok := matrix[id]
	return e, ok
}

// All returns a defensive copy of the full capability descriptor, keyed by
// language ID — callers may range over it freely without risk of mutating
// the package-level matrix.
func All() map[string]CapabilityEntry {
	out := make(map[string]CapabilityEntry, len(matrix))
	for id, e := range matrix {
		gaps := make([]string, len(e.Gaps))
		copy(gaps, e.Gaps)
		e.Gaps = gaps
		out[id] = e
	}
	return out
}
