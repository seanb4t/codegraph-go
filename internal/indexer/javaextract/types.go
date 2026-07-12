// Package javaextract walks a Java file's tree-sitter syntax tree into the
// shared codegraph vocabulary (goextract.FileResult/ExtractedNode/IntraEdge/
// UnresolvedRef, goextract.Kind*/RefKind*), mirroring
// internal/indexer/goextract's shape (D-01) rather than redefining its own
// copy of that vocabulary.
//
// Node-kind mapping decisions (documented here per the plan's action, not
// enforced by the type system):
//
//   - class_declaration -> goextract.KindStruct. Java has no "struct"
//     keyword; a class is the closest semantic analog to what
//     goextract.KindStruct represents elsewhere (a concrete, field-and-
//     method-bearing type, as opposed to an interface). Reusing KindStruct
//     rather than minting a new "class" kind keeps struct/class-shaped
//     downstream consumers (e.g. RES-02's Wave-C implements synthesis,
//     which pattern-matches "a concrete type vs. an interface") working
//     unchanged across languages.
//   - interface_declaration -> goextract.KindInterface (direct analog).
//   - method_declaration and constructor_declaration -> goextract.KindMethod.
//     A constructor's Name/QualifiedName equal the declaring class's own
//     name (Java constructors share the class's identifier); this is a
//     deliberate, minor QualifiedName collision with the class's own
//     "Type.Type" shape rather than a modeling gap.
//   - field_declaration -> NO node is ever emitted (mirrors goextract's
//     ratified "no field node" skip, 02-RESEARCH.md Open Question 3 — see
//     goextract/types.go's own doc comment). Extending this skip to Java
//     keeps the vocabulary consistent across languages rather than
//     special-casing Java to emit what Go deliberately does not.
//   - import_declaration -> goextract.RefKindImports.
//   - superclass (extends) / super_interfaces / extends_interfaces
//     (implements) -> goextract.RefKindEmbeds, exactly one unresolved ref
//     per listed supertype. Per RESEARCH Pattern 2, this extractor does
//     NOT try to distinguish "class extends class" from "class implements
//     interface" at parse time — both are syntactically an unresolved
//     supertype reference; promoting an "embeds" edge to "implements" once
//     the target's Kind is known (interface vs. not) is Wave 6's (RES-02)
//     resolve-time job, out of this plan's scope.
//   - method_invocation -> goextract.RefKindCalls.
package javaextract
