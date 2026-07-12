// Package swiftextract walks a Swift file's tree-sitter syntax tree into the
// shared codegraph vocabulary (goextract.FileResult/ExtractedNode/IntraEdge/
// UnresolvedRef, goextract.Kind*/RefKind*), mirroring
// internal/indexer/goextract's shape (D-01) rather than redefining its own
// copy of that vocabulary.
//
// This is a MAINSTREAM-tier extractor (LANG-06, D-04), built on the
// `[SUS]`-flagged, community-maintained `alex-pinkus/tree-sitter-swift`
// grammar (T-05-SC — pinned by exact commit after a blocking human-verify
// checkpoint in 05-08, not `@latest`). It is this grammar's FIRST extraction
// exercise in this project: full structural extraction, but only
// best-effort SAME-FILE cross-reference resolution — not the priority-4
// full-import-resolution bar. Every resolution gap named below is a
// deliberate, documented tier boundary, not a bug, and several node-kind
// mapping decisions below were adapted from the plan's original
// expectation after a live parse-tree dump revealed real, verified grammar
// rough edges (this is the "documented-partial acceptable, may be rougher"
// floor 05-CONTEXT.md/05-RESEARCH.md set for this [SUS] grammar).
//
// Node-kind mapping decisions:
//
//   - There is NO dedicated `class_declaration`/`struct_declaration` node
//     pair in this grammar, unlike the plan's original expectation — class,
//     struct, enum, actor, AND extension declarations all share ONE
//     `class_declaration` node kind, distinguished only by an anonymous
//     "declaration_kind" keyword token child ("class"/"struct"/"enum"/
//     "actor"/"extension") — verified via a live parse-tree dump. This
//     extractor scans for that keyword token (swiftDeclKind) rather than
//     relying on a node-kind switch. class/struct/actor/enum ->
//     goextract.KindStruct (this vocabulary has no distinct kind for any of
//     these; enum's own cases are never extracted as separate symbols,
//     mirroring rustextract's identical enum decision). extension
//     declarations are recognized (so they are never accidentally
//     misclassified as a fresh type) but their OWN body is never walked —
//     see "Explicitly out of scope" below.
//   - protocol_declaration -> goextract.KindInterface.
//   - A `function_declaration`'s own "name" field is REUSED (bound
//     "multiple: true" in this grammar) for both the function's own
//     simple_identifier name AND, separately, a return-type-shaped node — a
//     verified real grammar rough edge. ChildByFieldName("name") reliably
//     returns the FIRST bound value (the simple_identifier), which this
//     extractor relies on directly rather than working around.
//     function_declaration nested inside a class/struct/actor's own
//     class_body -> goextract.KindMethod (QualifiedName "Type.fnname"); a
//     ROOT-LEVEL function_declaration -> goextract.KindFunction. A
//     protocol's own protocol_function_declaration (a DIFFERENT node kind,
//     always bodyless) is never extracted — mirrors this project's
//     ratified "no field/bodyless-prototype node" skip.
//   - import_declaration -> goextract.RefKindImports, one ref per import
//     (the module name is the declaration's own "identifier" child's full
//     text — Swift's grammar nests a simple_identifier one level inside
//     that "identifier" node; this extractor reads the outer node's text
//     directly rather than descending further). FileResult.Imports is
//     NEVER populated: a Swift `import Foundation` names a whole external
//     module, not a moduleKey-shaped identifier this tier could match
//     against a same-repo file's own computed ModuleKey (mirrors Rust's
//     identical `use` decision — no local per-symbol import table exists in
//     Swift's own import syntax to key off of in the first place).
//   - inheritance_specifier (a class/struct/actor/protocol_declaration's own
//     direct, unfielded child — ONE per conformance/superclass, verified via
//     a live parse-tree dump with a multi-conformance `class Foo: A, B`) ->
//     goextract.RefKindEmbeds, resolved through its own "inherits_from"
//     field down to a user_type's inner type_identifier (or a bare
//     type_identifier directly). Undistinguished at parse time whether this
//     is a superclass or a protocol conformance (Pattern 2 — resolve.go
//     promotes to "implements" once/if the target resolves to an interface
//     node).
//   - call_expression -> goextract.RefKindCalls. The callee is ALWAYS the
//     call_expression's own first named child (this grammar has no
//     "function" field on call_expression — verified via a live parse-tree
//     dump). A bare `simple_identifier` callee is an unqualified call
//     (empty PkgAlias, covers both a bare function call AND a bare
//     constructor call `Circle()`, indistinguishable at this tier). A
//     `navigation_expression` callee (`recv.method()`) resolves through its
//     own "target"/"suffix" fields (suffix is a wrapping navigation_suffix
//     node with its OWN "suffix" field holding the final simple_identifier
//     member name) — an uppercase-leading identifier target is a
//     same-module attempt (empty PkgAlias, mirroring
//     pyextract/rustextract's PascalCase-type-name heuristic); any other
//     identifier target is forced through the WR-02 synthetic-non-matching-
//     alias pattern (`<local:name>`); any other target shape (including
//     `self`, deliberately NOT special-cased — see below) gets a synthetic
//     `<kind>` alias.
//
// Explicitly out of scope for this mainstream tier (named for the D-11
// capability matrix, not silently missing):
//   - Extension declarations (`extension Foo { ... }`) are recognized at the
//     top level (so they never get misextracted as a fresh, empty "Foo"
//     type) but their OWN member declarations are NEVER walked/extracted —
//     merging an extension's members into its extended type's method set
//     would require a cross-declaration merge this tier does not implement.
//     This is the plan's own explicitly-named gap ("Swift ... extension
//     resolution ... named as gaps").
//   - Protocol-witness resolution (proving a concrete type's method
//     actually satisfies a protocol's requirement) is never attempted — no
//     structural-conformance pass exists for this tier (Pattern 3's Go-only
//     dispatch synthesis does not generalize here).
//   - `self`-qualified calls (`self.method()`) are NOT specially recognized
//     as an implicit same-instance call — `self` is a lowercase
//     simple_identifier like any other local binding, so it is
//     conservatively routed through the WR-02 synthetic-alias path
//     (deterministically "unresolved" rather than a risky guess) — a
//     documented, accepted simplification (mirrored identically by
//     kotlinextract's `this`).
//   - Enum case values, associated values, and property/computed-property
//     declarations are never extracted as their own symbols.
//   - Generic type parameters and constraints are never tracked; a generic
//     type's own name still extracts normally (the "name" field's plain
//     type_identifier/user_type resolution ignores any accompanying
//     type_parameters child).
package swiftextract
