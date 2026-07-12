// Package kotlinextract walks a Kotlin file's tree-sitter syntax tree into
// the shared codegraph vocabulary (goextract.FileResult/ExtractedNode/
// IntraEdge/UnresolvedRef, goextract.Kind*/RefKind*), mirroring
// internal/indexer/goextract's shape (D-01) rather than redefining its own
// copy of that vocabulary.
//
// This is a MAINSTREAM-tier extractor (LANG-06, D-04), built on the
// `[SUS]`-flagged, community-maintained `tree-sitter-grammars/tree-sitter-
// kotlin@v1.1.0` grammar (T-05-SC — pinned after a blocking human-verify
// checkpoint in 05-08, replacing an originally-approved but unbuildable
// source, not `@latest`). It is this grammar's FIRST extraction exercise in
// this project: full structural extraction, but only best-effort SAME-FILE
// cross-reference resolution — not the priority-4 full-import-resolution
// bar. Every resolution gap named below is a deliberate, documented tier
// boundary, not a bug. This grammar turned out CLEANER than Swift's — no
// double-bound field or other verified rough edge was found during this
// package's own live parse-tree exploration.
//
// Node-kind mapping decisions:
//
//   - Kotlin's `class_declaration` node kind covers class/interface/
//     enum/object declarations alike, distinguished only by an ANONYMOUS
//     "class"/"interface"/"enum"/"object" keyword token child (this
//     grammar's node-types.json declares NO field for it at all — verified
//     via a live parse-tree dump). This extractor scans for that keyword
//     token (kotlinDeclKind) rather than a node-kind switch. "class" and
//     "enum" -> goextract.KindStruct (this vocabulary has no distinct
//     "enum" kind; an enum's own entries are never extracted as separate
//     symbols, mirroring rustextract's identical enum decision).
//     "interface" -> goextract.KindInterface. "object" (Kotlin's singleton
//     declaration) is never extracted — a documented gap (no Kind in this
//     vocabulary fits a language-level singleton without inventing one,
//     and this tier does not).
//   - function_declaration nested inside a class_declaration's own
//     class_body -> goextract.KindMethod (QualifiedName "Type.fnname"),
//     INCLUDING an interface's own bodyless method requirement (its
//     function_declaration simply has no function_body child — Kotlin does
//     not use a separate node kind for an interface requirement the way
//     Swift's protocol_function_declaration does, so this extractor emits
//     the method node regardless of whether a body is present, mirroring
//     phpextract's identical "an interface method has no body" handling).
//     A ROOT-LEVEL function_declaration -> goextract.KindFunction.
//   - `package` (package_header, root-level only) -> a parse-time
//     ImportPath OVERRIDE of the discovery-time path-based placeholder
//     (mirrors phpextract's/csharpextract's identical parse-time-namespace-
//     override pattern — Kotlin's package, like C#/PHP's namespace, is
//     declared IN the source, not derived purely from directory layout).
//   - `import` -> goextract.RefKindImports, one ref per import (the fully
//     dotted qualified_identifier text). FileResult.Imports IS populated
//     (alias = the qualified name's last dotted segment, mirroring PHP's
//     `use` decision — a Kotlin import target is unambiguously a
//     class/function/property's own simple name, not an ambiguous crate-
//     relative path the way Rust's `use` is).
//   - delegation_specifier (a class_declaration's own superclass/interface
//     conformance list, one per delegation_specifier -- either a plain
//     `user_type` for interface conformance or a `constructor_invocation`
//     wrapping one for an invoked superclass constructor, both verified via
//     live parse-tree dumps) -> goextract.RefKindEmbeds (Pattern 2 —
//     undistinguished at parse time; resolve.go promotes to "implements" if
//     the target resolves to an interface node).
//   - call_expression -> goextract.RefKindCalls. This grammar's
//     call_expression has NO named fields at all (verified via node-types.
//     json) — the callee is always the call_expression's own first named
//     child. A bare `identifier` callee is an unqualified call (empty
//     PkgAlias, covers both a bare function call AND a bare constructor
//     call `Circle(1.0)`, indistinguishable at this tier). A
//     `navigation_expression` callee (`recv.method()`, itself field-less —
//     receiver is its first named child, member name its second) resolves
//     the same PascalCase-vs-lowercase heuristic every other mainstream
//     extractor in this project uses; a non-identifier receiver (including
//     a CHAINED call, `c.area().toInt()`, whose own receiver is itself a
//     call_expression) gets a synthetic `<kind>` alias rather than being
//     silently dropped.
//
// Explicitly out of scope for this mainstream tier (named for the D-11
// capability matrix, not silently missing):
//   - Kotlin extension functions (`fun String.shout() { ... }`, a
//     function_declaration with a receiver TYPE prefix rather than a
//     class_body container) are never extracted as methods of their
//     receiver type — this tier only recognizes class-BODY-nested methods.
//     This is the plan's own explicitly-named gap ("Kotlin extension-
//     function ... resolution ... named as gaps").
//   - `object` singleton declarations and companion objects
//     (`companion object { ... }`, itself declared via a DIFFERENT node
//     kind this extractor never walks) are never extracted — the plan's own
//     explicitly-named "companion-object resolution" gap.
//   - `this`-qualified calls (`this.method()`) are NOT specially recognized
//     as an implicit same-instance call — `this` is a lowercase identifier
//     like any other local binding, so it is conservatively routed through
//     the WR-02 synthetic-alias path (mirrored identically by
//     swiftextract's `self`).
//   - Data class auto-generated members (`copy()`, `componentN()`,
//     synthesized `equals`/`hashCode`/`toString` for a `data class`) are
//     never extracted — this extractor only sees the class's own
//     source-level function_declaration children.
//   - Property declarations (`val`/`var`) are never extracted as their own
//     symbols, matching this project's ratified "no field node" skip.
package kotlinextract
