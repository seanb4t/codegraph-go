// Package csharpextract walks a C# file's tree-sitter syntax tree into the
// shared codegraph vocabulary (goextract.FileResult/ExtractedNode/IntraEdge/
// UnresolvedRef, goextract.Kind*/RefKind*), mirroring
// internal/indexer/javaextract's shape (D-01) rather than redefining its
// own copy of that vocabulary.
//
// Node-kind mapping decisions (documented here per the plan's action, not
// enforced by the type system):
//
//   - class_declaration / struct_declaration / record_declaration ->
//     goextract.KindStruct. C# genuinely has a struct keyword (unlike
//     Java), but reusing ONE kind for all three concrete-type-declaration
//     shapes keeps struct/class-shaped downstream consumers (Wave 6's
//     implements synthesis) language-agnostic, exactly mirroring Java's own
//     class_declaration -> KindStruct decision.
//   - interface_declaration -> goextract.KindInterface (direct analog).
//   - method_declaration and constructor_declaration -> goextract.KindMethod.
//     A constructor's Name/QualifiedName equal the declaring type's own name
//     (C# constructors share the type's identifier, exactly like Java) — a
//     deliberate, minor QualifiedName collision with the type's own
//     "Type.Type" shape, not a modeling gap.
//   - property_declaration and field_declaration -> NO node is ever
//     emitted (mirrors goextract's ratified "no field node" skip,
//     02-RESEARCH.md Open Question 3, and javaextract's own field skip).
//     Extending this skip to C# properties keeps the vocabulary consistent
//     across languages rather than special-casing C# to emit what Go/Java
//     deliberately do not.
//   - using_directive -> goextract.RefKindImports, one unresolved ref per
//     directive. A plain `using Foo.Bar;` (the overwhelmingly common case)
//     names a whole NAMESPACE, not a single class the way Java's `import`
//     does — so, unlike javaextract, this extractor does NOT populate
//     result.Imports from a plain using directive (there is no single
//     simple name to key it by). The ALIAS form (`using X = Foo.Bar;`) DOES
//     name one specific target, so it populates result.Imports["X"] =
//     "Foo.Bar" exactly like a Go/Java import alias.
//   - base_list entries (`: IFoo, BaseClass`) -> goextract.RefKindEmbeds,
//     exactly one unresolved ref per listed supertype. Per RESEARCH
//     Pattern 2, C#'s base_list is a single comma-separated list with NO
//     syntactic marker distinguishing "class extends class" from "class
//     implements interface" — this extractor does not try to distinguish
//     them at parse time; promoting an "embeds" edge to "implements" once
//     the target's Kind is known is Wave 6's (RES-02) resolve-time job.
//   - invocation_expression -> goextract.RefKindCalls.
//
// # Cross-namespace call/embeds qualifier resolution (documented heuristic)
//
// C#'s `using` directive imports an entire NAMESPACE, not a single class —
// there is no Java-style "import statement names both the simple class name
// AND its declaring package" mapping this extractor can lean on. Without a
// full cross-file symbol table at parse time (out of scope — Pass 1 sees
// only one file), this extractor resolves a qualified reference's PkgAlias
// via two, and only two, deterministic mechanisms:
//
//  1. A FULLY-QUALIFIED reference (`Other.Namespace.Base`, `Other.Namespace.
//     Helper.Assist()`) already spells out its own declaring namespace in
//     its own AST shape (a qualified_name's "qualifier" field) — that text
//     is used directly as PkgAlias, self-mapped into result.Imports so
//     resolveSelector's exact-match lookup succeeds without needing a prior
//     `using` declaration at all.
//  2. An unqualified, PascalCase-leading simple name (`Helper.Assist()`,
//     `: BaseClass`) is attempted as a SAME-NAMESPACE reference (empty
//     PkgAlias, resolveUnqualified against the caller's own namespace) —
//     mirroring javaextract's same-package heuristic. A camelCase-leading
//     name is routed through a synthetic non-matching alias instead
//     (mirrors goextract's WR-02 fix), since it is very likely a local
//     variable/field/parameter, not a type.
//
// A bare (non-fully-qualified) reference to a type that is genuinely
// declared in a DIFFERENT namespace, reached only via a namespace-level
// `using` (the idiomatic, overwhelmingly common C# pattern), is an
// EXPLICIT, DOCUMENTED, ACCEPTED gap: this extractor cannot know which of a
// file's (possibly many) `using` namespaces actually declares a given bare
// simple name without either a full symbol table at parse time or a
// resolve-time multi-candidate retry — both outside this plan's file scope
// (csharpextract + languages_csharp.go only; resolve.go/symbolindex.go are
// not modified by this plan). RESEARCH's own Assumptions Log (A1) already
// anticipates real per-language resolution algorithms may have edge cases
// not fully captured pre-D-12; this is exactly such a case, surfaced here
// rather than silently guessed at.
//
// # Partial-class node identity (Pitfall 5 — the plan's central open question)
//
// This extractor implements RESEARCH's scheme (b): a `partial class`/
// `partial struct`/`partial record`/`partial interface`'s node is keyed by
// (qualifiedName, namespace) ONLY — no filePath — via
// nodeid.NodeID(kind, name, importPath), so every fragment (regardless of
// which file it is declared in) computes the exact same node id. Each
// fragment's own methods are still `contains`-edged from that ONE shared id
// (no data loss — every fragment's methods survive).
//
// RESEARCH's literal suggestion for the tie-break on the shared node's own
// FilePath/StartLine/EndLine was "the FIRST fragment encountered,
// deterministic tie-break by file path sort order" — but genuinely
// resolving "which fragment sorts first" requires CROSS-FILE coordination
// (comparing this file's path against every OTHER fragment's path), which
// only resolve.go's writeGraph (which ranges every file's results in a
// single pass) could do — and this plan's file scope is deliberately
// limited to csharpextract + languages_csharp.go, not resolve.go. Instead,
// this extractor uses a DETERMINISTIC SENTINEL for a partial type's shared
// node: FilePath="", StartLine=0, EndLine=0, StartCol=0, EndCol=0. Every
// fragment computes this SAME sentinel independently, so there is no
// dependency on write order at all (the "collision" Pitfall 5 warns about
// is a collision of IDENTICAL content, not a race between two DIFFERENT
// candidate values) — this mirrors resolve.go's own pre-existing
// kindPackage pseudo-node pattern, which already carries no FilePath.
// Consumers wanting a specific fragment's location can navigate via that
// fragment's own `contains`-edged methods, each of which keeps its own
// real, per-fragment FilePath/StartLine.
//
// This is a deliberate, documented refinement of RESEARCH's suggested tie-
// break — not scheme (a) (RESEARCH's fallback of leaving each fragment as
// a separate, FILE-scoped node with no single "the class" node) — chosen
// because it fully achieves scheme (b)'s core goal (ONE shared node id, no
// method data loss, deterministic across runs) without requiring changes
// outside this plan's file scope.
package csharpextract
