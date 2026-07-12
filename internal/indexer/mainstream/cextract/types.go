// Package cextract walks a C or C++ file's tree-sitter syntax tree into the
// shared codegraph vocabulary (goextract.FileResult/ExtractedNode/IntraEdge/
// UnresolvedRef, goextract.Kind*/RefKind*), mirroring
// internal/indexer/goextract's shape (D-01) rather than redefining its own
// copy of that vocabulary.
//
// This is ONE extractor shared by TWO grammars (C and C++, LANG-06, D-11):
// languages_c.go and languages_cpp.go both route through Extract, which
// determines which language it is parsing from relPath's own extension
// (languageForExt) — the parser.Parser it is handed is already the
// language-specific one (cgo.NewCParser/NewCppParser), so a C-grammar tree
// simply never contains a C++-only node kind (class_specifier,
// namespace_definition, base_class_clause, qualified_identifier) and vice
// versa; one shared switch-based walker is safe for both.
//
// MAINSTREAM-tier (D-04): full structural extraction, but only best-effort
// SAME-FILE/SAME-MODULE cross-reference resolution — not the priority-4
// full-import-resolution bar. Every resolution gap named below is a
// deliberate, documented tier boundary, not a bug.
//
// Node-kind mapping decisions:
//
//   - struct_specifier (C and C++) -> goextract.KindStruct, keyed by its
//     "name" field (type_identifier only — an anonymous struct with no name
//     field, or one with a template_type/qualified_identifier name, is
//     skipped: a documented gap, see "template instantiations" below).
//   - class_specifier (C++) -> goextract.KindStruct (this vocabulary has no
//     distinct "class" kind, mirroring the project's existing "reuse an
//     existing Kind rather than mint a new one" discipline). Only a plain
//     type_identifier name is handled; a class_specifier whose own "name"
//     field is a template_type (`template <typename T> class Widget`) or a
//     qualified_identifier is skipped — a documented gap (template
//     instantiations are never tracked by this tier, matching Rust's own
//     documented macro/generic boundary).
//   - namespace_definition (C++) is NOT itself emitted as a node (no Kind in
//     this vocabulary fits a namespace) — its own declaration_list body is
//     instead walked TRANSPARENTLY, exactly as if its declarations were
//     direct top-level siblings (recursively, so nested namespaces are fully
//     flattened too). This means namespace-qualified names are never
//     disambiguated by namespace at this tier — a documented, accepted gap
//     (mirrors phpextract's identical one-level braced-namespace expansion,
//     just applied recursively here since C++ namespaces nest freely).
//   - preproc_include -> goextract.RefKindImports (best-effort): the
//     `"..."`  or `<...>` path text is recorded verbatim (quotes/angle
//     brackets stripped) as the ref's Name. FileResult.Imports is NEVER
//     populated from a #include — an include path is a filesystem path, not
//     a moduleKey-shaped identifier this tier could match against another
//     file's own computed ModuleKey (mirrors Rust's identical `use`
//     decision).
//   - type_definition (typedef) -> goextract.KindTypeAlias, keyed by the
//     declarator's resolved simple name (walking through any
//     pointer/array/reference wrapping — `typedef struct Foo *FooPtr;`
//     resolves to "FooPtr").
//   - A root-level (or namespace-flattened) function_definition (has a
//     body) -> goextract.KindFunction, keyed by its declarator's resolved
//     identifier name. A root-level `declaration` node whose declarator
//     resolves to a function_declarator (a C/C++ function PROTOTYPE with no
//     body — common in headers) is ALSO extracted as goextract.KindFunction
//     (no calls collected, since there is no body) — this is deliberate:
//     header-declared prototypes are real, useful cross-file symbols even
//     though this tier's Extract only ever sees one file at a time. A
//     top-level function_definition whose declarator resolves to a C++
//     qualified_identifier (`RetType Type::method() { ... }`, the dominant
//     C++ idiom for a class's own method BODIES defined outside the class)
//     is extracted as goextract.KindMethod instead — attached via a
//     same-file contains IntraEdge when the qualifying type is declared in
//     this same file, or a RefKindContains unresolved ref otherwise
//     (mirrors rustextract's identical cross-file impl-block handling).
//   - Inside a class_specifier's own field_declaration_list body, ONLY a
//     direct function_definition child (an INLINE method — one with a body
//     written directly in the class) -> goextract.KindMethod. A
//     field_declaration child (a member variable, OR a bodyless method
//     prototype — including every pure-virtual `= 0` declaration) is never
//     extracted at all — this tier's ratified "no field node" skip extends
//     to bodyless/pure-virtual method declarations too, a documented gap
//     (mirrors PHP's own "an interface method has no body" being extracted
//     anyway being the ONE exception in this project; C++'s pure-virtual
//     methods are NOT that exception, since field_declaration is a
//     genuinely different, unwalked node kind here, not a bodyless
//     method_declaration).
//   - A method/function declarator resolving to a C++ destructor_name
//     (`~Widget`) is extracted using that node's own full text (e.g.
//     "~Widget") as its Name. A declarator resolving to an operator_name
//     (`operator+`) is skipped entirely — a documented gap (operator
//     overloads are never extracted as symbols by this tier).
//   - base_class_clause (C++, direct unfielded child of class_specifier) ->
//     goextract.RefKindEmbeds, one ref per PLAIN type_identifier base
//     (Pattern 2 — extends/implements undistinguished at parse time;
//     resolve.go promotes it to "implements" if the target later resolves
//     to an interface node — though this tier never emits KindInterface for
//     C/C++, so this promotion is inert here, kept only for vocabulary
//     consistency with every other extractor). A qualified_identifier base
//     (`class A : public ns::Base`) is skipped — a documented gap.
//   - call_expression -> goextract.RefKindCalls. A bare `identifier` callee
//     is an unqualified call (empty PkgAlias). A `field_expression` callee
//     (`recv.method()` / `recv->method()`, C's "argument"/"field" field
//     names) checks its "argument" receiver: `this` is treated as an
//     implicit same-class call (empty PkgAlias); an uppercase-leading
//     identifier receiver is treated as a same-module attempt (empty
//     PkgAlias, mirroring pyextract/rustextract's PascalCase-type-name
//     heuristic); any other identifier receiver is forced through the
//     WR-02 synthetic-non-matching-alias pattern (`<local:name>`) so it
//     deterministically ends up "unresolved" rather than risking a false
//     same-module match; any other receiver shape gets a synthetic
//     `<kind>` alias. A `qualified_identifier` callee (`Type::method()` /
//     `Namespace::func()`) is recorded with an empty PkgAlias — a
//     same-module attempt (this tier cannot distinguish a namespace
//     qualifier from a class qualifier at this node shape, matching C++'s
//     own genuine grammar-level ambiguity here — both produce
//     `scope: (namespace_identifier)`).
//
// The C vs. C++ `.h` extension ambiguity: languages_c.go claims ".h" (the
// documented default disposition — a C++ project's own .h headers are
// parsed with the C grammar, a named, accepted D-11 matrix gap);
// languages_cpp.go claims only ".hpp"/".hh" (unambiguously C++).
//
// Explicitly out of scope for this mainstream tier (named for the D-11
// capability matrix, not silently missing):
//   - Preprocessor-macro-generated symbols (function-like/object-like macro
//     expansions producing new declarations) are never extracted — this
//     extractor only sees the macro invocation/definition site
//     (preproc_call/preproc_def/preproc_function_def), not any expansion,
//     since expansion requires a real preprocessor this project does not
//     run.
//   - Template instantiations and template declarations themselves
//     (`template <typename T> class Widget { ... }`,
//     `template <typename T> T max(T a, T b) { ... }`) are never extracted
//     — a class/function whose own name field resolves to a template_type
//     or whose declaration is wrapped in a template_declaration node is
//     silently skipped by this tier's plain-type_identifier-only name
//     resolution.
//   - Pure-virtual and other bodyless method declarations inside a class
//     body (field_declaration, not function_definition) are never
//     extracted, matching this project's ratified "no field node" skip.
//   - Operator overloads (`operator+`, `operator==`, ...) are never
//     extracted as symbols.
//   - Namespace-qualified names are never disambiguated by namespace —
//     namespace bodies are transparently flattened into the enclosing
//     file's own top-level scope.
//   - A multi-declarator top-level `declaration` (`int x, foo();`) only
//     ever considers the FIRST bound declarator (tree-sitter's own
//     ChildByFieldName behavior for a "multiple: true" field) — a rare,
//     accepted approximation.
package cextract
