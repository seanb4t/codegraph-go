// Package rustextract walks a Rust file's tree-sitter syntax tree into the
// shared codegraph vocabulary (goextract.FileResult/ExtractedNode/IntraEdge/
// UnresolvedRef, goextract.Kind*/RefKind*), mirroring
// internal/indexer/goextract's shape (D-01) rather than redefining its own
// copy of that vocabulary.
//
// This is a MAINSTREAM-tier extractor (LANG-06, D-04): full structural
// extraction, but only best-effort SAME-FILE/SAME-MODULE cross-reference
// resolution — not the priority-4 full-import-resolution bar. Every
// resolution gap named below is a deliberate, documented tier boundary, not
// a bug.
//
// Node-kind mapping decisions:
//
//   - struct_item -> goextract.KindStruct.
//   - enum_item -> goextract.KindStruct (documented gap: this vocabulary has
//     no distinct "enum" kind; an enum's own variants are never extracted as
//     separate symbols, mirroring the project's existing "reuse an existing
//     Kind rather than mint a new one" discipline).
//   - trait_item -> goextract.KindInterface.
//   - A top-level (root-direct-child) function_item -> goextract.KindFunction.
//     A function_item nested inside an impl_item's own declaration_list body
//     -> goextract.KindMethod (QualifiedName "ImplType.fnname"). Only
//     ROOT-LEVEL declarations are walked (mirrors goextract's/pyextract's own
//     top-level-only scope) — items nested inside a `mod foo { ... }` inline
//     module block are NOT descended into; a documented, accepted gap for
//     this tier.
//   - Rust has no field_declaration-equivalent node this extractor visits at
//     all — a struct's own fields are never walked, matching the ratified "no
//     field node" skip every other extractor in this project already applies.
//   - use_declaration -> goextract.RefKindImports, one ref per leaf path
//     (handling the plain/aliased/grouped-brace/wildcard shapes). This
//     extractor deliberately NEVER populates FileResult.Imports from a `use`
//     statement: resolving `use crate::foo::Bar` into a moduleKey that
//     actually matches the target file's own computed moduleKey
//     (languages_rust.go's crate-name + module-path scheme) requires knowing
//     the enclosing crate's name at Extract time, which this function's
//     shared cross-language signature does not provide (the same
//     architectural constraint tsextract's Config/SetConfig side-channel
//     pattern solves for TS/JS's tsconfig paths — out of scope for this
//     mainstream-tier plan). Consequently cross-file `use`-qualified call/
//     trait-impl resolution is NOT implemented; only same-file (and,
//     incidentally, any other file sharing the exact same computed
//     rustModulePath) resolution via an empty PkgAlias works.
//   - impl_item: the "type" field's struct/enum gets each of the impl
//     block's own function_item children extracted as KindMethod nodes,
//     attached via a same-file contains IntraEdge when the impl'd type is
//     declared in this same file, or a RefKindContains unresolved ref
//     otherwise (mirrors goextract's cross-file receiver-type handling).
//     When the impl_item ALSO carries a "trait" field (`impl Trait for
//     Type`), one RefKindEmbeds unresolved ref (Type -> Trait) is emitted,
//     undistinguished from a "some other embed" shape at parse time (Pattern
//     2) — resolve.go promotes it to "implements" once the target resolves
//     to an interface node, exactly like Java/C#/TS/Python's own extends/
//     implements-shaped refs.
//   - call_expression -> goextract.RefKindCalls. A bare identifier callee is
//     an unqualified call (empty PkgAlias). A field_expression callee
//     (`recv.method()`) checks whether the receiver identifier is likely a
//     same-module type name by PascalCase naming convention (Rust's own
//     near-universal convention distinguishing a type from a local
//     binding/parameter) — an empty PkgAlias routes it through
//     resolveUnqualified; any other identifier or non-identifier receiver
//     routes through the WR-02 synthetic-non-matching-alias pattern so it
//     deterministically ends up "unresolved" rather than risking a false
//     same-module match. A scoped_identifier callee (`Type::method()` /
//     `Self::method()` / `crate::method()`) applies the same PascalCase
//     heuristic to its path segment.
//
// Explicitly out of scope for this mainstream tier (named for the D-11
// capability matrix, not silently missing):
//   - Macro-generated items (derive macros, function-like macros expanding
//     into new struct/fn/trait declarations) are never extracted — this
//     extractor only sees the macro invocation site (macro_invocation), not
//     its expansion, since expansion requires a real macro engine this
//     project does not run.
//   - generic_function callees (`foo::<T>()`, turbofish syntax) and
//     macro_invocation callees (`println!()`) are not resolved into calls
//     edges at all.
//   - Trait default-method inheritance (a type implementing a trait that has
//     a default-bodied method, calling that method without overriding it) is
//     not resolved — no conformance-retry pass exists for this tier
//     (Pitfall 3's two-pass retry is a priority-4-only mechanism).
//   - Items declared inside an inline `mod foo { ... }` block are not
//     extracted (top-level-only scope, see above).
package rustextract
