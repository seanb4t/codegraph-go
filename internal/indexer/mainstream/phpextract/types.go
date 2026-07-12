// Package phpextract walks a PHP file's tree-sitter syntax tree into the
// shared codegraph vocabulary (goextract.FileResult/ExtractedNode/IntraEdge/
// UnresolvedRef, goextract.Kind*/RefKind*), mirroring
// internal/indexer/goextract's shape (D-01) rather than redefining its own
// copy of that vocabulary. Uses the `php/src` grammar accessor
// (cgo.NewPHPParser -> tree_sitter_php.LanguagePHP()), not the sibling
// `php_only/src` grammar.
//
// This is a MAINSTREAM-tier extractor (LANG-06, D-04): full structural
// extraction, but only best-effort SAME-FILE/SAME-NAMESPACE cross-reference
// resolution — not the priority-4 full-import-resolution bar. Every
// resolution gap named below is a deliberate, documented tier boundary, not
// a bug.
//
// Node-kind mapping decisions:
//
//   - class_declaration -> goextract.KindStruct.
//   - interface_declaration -> goextract.KindInterface.
//   - trait_declaration -> goextract.KindStruct (documented gap: this
//     vocabulary has no distinct "trait" kind; a trait's method-providing
//     role is the closest semantic analog to KindStruct here).
//   - A method_declaration inside a class/interface/trait's own
//     declaration_list body -> goextract.KindMethod (QualifiedName
//     "TypeName.methodname"); a top-level function_definition ->
//     goextract.KindFunction. Only TOP-LEVEL declarations (and, per PHP's
//     own unbraced-namespace convention, the direct children of a single
//     braced `namespace Foo { ... }` block) are walked — a declaration
//     nested inside anything else (an anonymous class expression, a
//     closure) is NOT extracted; a documented, accepted gap for this tier.
//   - An interface's method_declaration has no body (abstract by
//     definition) — extracted as a node like any other method, simply with
//     no calls collected from it (no body to walk).
//   - PHP has no property/field-declaration-equivalent node this extractor
//     visits at all — a class's own typed/untyped properties are never
//     walked, matching the ratified "no field node" skip every other
//     extractor in this project already applies.
//   - namespace_use_declaration -> goextract.RefKindImports, one ref per
//     resolved leaf path (handling the plain, aliased, and grouped/braced
//     `use Prefix\{A, B}` forms) — and, unlike Rust's `use`, DOES populate
//     FileResult.Imports (keyed by the alias-or-simple-name), since a PHP
//     `use` statement's imported symbol is unambiguously a class/
//     interface/trait/function name (no crate-name-resolution ambiguity
//     Rust's `use` has).
//   - A class's base_clause (extends) and class_interface_clause
//     (implements), and an interface's own base_clause (extends, possibly
//     multiple) -> goextract.RefKindEmbeds, one ref per named supertype,
//     undistinguished from any other embed shape at parse time (Pattern 2)
//     — resolve.go promotes it to "implements" once the target resolves to
//     an interface node, exactly like Java/C#/TS/Python's own extends/
//     implements-shaped refs. The referenced name's PkgAlias checks
//     FileResult.Imports membership by its SIMPLE (last-segment) name; a
//     fully-qualified inline reference (`extends \App\Base`) is recorded
//     with its simple name only (the fully-qualified text is not itself
//     tracked as an alternate lookup key) — an accepted approximation.
//   - function_call_expression (a bare or namespaced function call),
//     member_call_expression/nullsafe_member_call_expression (`$obj->
//     method()`/`$obj?->method()`), and scoped_call_expression
//     (`Type::method()`) -> goextract.RefKindCalls. `$this->method()`
//     is treated as an implicit same-class call (empty PkgAlias, mirrors
//     Java's `this.method()`); `Type::method()`/`self::`/`static::`/
//     `parent::` are treated as same-module attempts (empty PkgAlias) —
//     PHP class names are conventionally PascalCase and this is the
//     dominant static-call idiom, mirroring pyextract/javaextract's
//     naming-convention heuristic; any other receiver (a local
//     variable/parameter/expression result) routes through the WR-02
//     synthetic-non-matching-alias pattern so it deterministically ends up
//     "unresolved" rather than risking a false same-module match.
//     object_creation_expression (`new Widget()`) is NOT extracted as a
//     calls edge at all — an accepted, documented gap (constructor
//     invocation tracking would need a distinct ref shape this tier does
//     not add).
//
// Explicitly out of scope for this mainstream tier (named for the D-11
// capability matrix, not silently missing):
//   - Dynamic method definition/metaprogramming (`__call`, `__callStatic`,
//     `__get`/`__set` magic methods invoked implicitly, `Closure::bind`) —
//     none of these produce a normal call/method-declaration AST shape this
//     extractor recognizes.
//   - `use TraitName;` inside a class body (PHP's trait-inclusion
//     mechanism, syntactically a `use_declaration` INSIDE a
//     declaration_list, distinct from the file-level
//     namespace_use_declaration this extractor DOES handle) is not
//     extracted as an embeds/contains edge — a trait's methods are not
//     merged onto the including class's own method set.
//   - Cross-NAMESPACE `use`-qualified call/supertype resolution beyond an
//     exact simple-name match against FileResult.Imports: this extractor
//     does track Imports (unlike Rust's `use`), so a same-name match DOES
//     resolve correctly when the target's own computed moduleKey (its
//     declared `namespace` statement, or the composer.json PSR-4 fallback
//     — see languages_php.go) equals the Imports value recorded here.
//   - A file with NO explicit `namespace` declaration falls back to
//     languages_php.go's composer.json PSR-4-derived or bare-relPath
//     moduleKey (D-03) — this is a real, working fallback, not a gap, but
//     is inherently less precise than an explicit `namespace` statement.
package phpextract
