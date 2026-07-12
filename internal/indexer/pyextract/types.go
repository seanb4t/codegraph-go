// Package pyextract walks a Python file's tree-sitter syntax tree into the
// shared codegraph vocabulary (goextract.FileResult/ExtractedNode/IntraEdge/
// UnresolvedRef, goextract.Kind*/RefKind*), mirroring
// internal/indexer/goextract's shape (D-01) rather than redefining its own
// copy of that vocabulary — the same discipline javaextract/csharpextract
// already follow.
//
// Node-kind mapping decisions (documented here per the plan's action, not
// enforced by the type system):
//
//   - class_definition -> goextract.KindStruct. Python has no "struct"
//     keyword; a class is the closest semantic analog to what
//     goextract.KindStruct represents elsewhere (a concrete, method-bearing
//     type). Reusing KindStruct rather than minting a new "class" kind keeps
//     struct/class-shaped downstream consumers (e.g. RES-02's Wave-C
//     implements synthesis) working unchanged across languages.
//   - function_definition nested directly in a class_definition's own body
//     -> goextract.KindMethod (QualifiedName "ClassName.methodname"); a
//     module-level function_definition -> goextract.KindFunction. A
//     decorated_definition is transparently unwrapped to its "definition"
//     field before this classification — the decorator(s) themselves are
//     not extracted as edges (out of scope for this plan, same boundary
//     LANG-07's Wave-D framework-route detection will later mine the same
//     AST for).
//   - Python has no field_declaration-equivalent node this extractor visits
//     at all — module-level and class-body assignments (`self.x = 1`,
//     `CONST = 1`) are simply never walked, matching the ratified "no field
//     node" skip goextract/javaextract/csharpextract all already apply.
//   - import_statement / import_from_statement -> goextract.RefKindImports.
//     A plain `import foo.bar` (no `as` alias) binds ONLY the top-level
//     package name ("foo") in real Python semantics — not the full dotted
//     path "foo.bar" — and this extractor tracks no multi-level attribute-
//     chain resolution beyond a single `identifier.attribute` hop, so (
//     mirroring csharpextract's documented "a plain `using` directive does
//     NOT populate Imports" gap) a plain import only records the
//     RefKindImports dependency ref, never an Imports map entry. An
//     aliased import (`import foo.bar as baz`) DOES populate
//     Imports["baz"] = "foo.bar", since baz genuinely IS the full dotted
//     module. `from foo.bar import Baz [as alias]` populates
//     Imports[alias-or-"Baz"] = "foo.bar", exactly mirroring
//     javaextract's per-class import handling. A relative import
//     (`from . import x`, `from ..pkg import y`) is resolved against the
//     current file's own enclosing-package dotted path (computed from its
//     ModuleKey) before being recorded — see resolveFromModule.
//   - A class's base-class list (the argument_list in class_definition's
//     "superclasses" field) -> goextract.RefKindEmbeds, one unresolved ref
//     per positional base-class argument (a simple identifier or a single
//     `module.Attr`-shaped attribute chain); keyword arguments
//     (`metaclass=...`) and starred/double-starred base-list entries are
//     skipped — they are not concrete supertype references this extractor
//     can resolve. Per RESEARCH Pattern 2 (also followed by
//     javaextract/csharpextract), this does NOT try to distinguish a
//     "real" base class from a Protocol/ABC/mixin at parse time; promoting
//     an "embeds" edge based on the target's actual Kind is Wave 6's
//     (RES-02) resolve-time job, out of this plan's scope.
//   - call -> goextract.RefKindCalls. `self.method()`/`cls.method()` are
//     treated as an implicit same-class call (empty PkgAlias, mirrors
//     Java's `this.method()`); an uppercase-leading (PascalCase) bare
//     identifier receiver that is not a real import is treated as a
//     likely same-module class reference (empty PkgAlias, the same naming-
//     convention heuristic javaextract/csharpextract already use, since
//     this extractor tracks no local-variable type table); any other
//     identifier or non-identifier receiver routes through the WR-02
//     synthetic-non-matching-alias pattern so it deterministically ends up
//     "unresolved" rather than risking a false same-module match.
//
// Python's resolution fidelity is inherently more heuristic than Go/Java/
// C#'s: Python has no static types on most references, PEP 420 implicit
// namespace packages and non-conventional sys.path layouts are real,
// documented risks (05-RESEARCH.md Assumptions Log A1) this extractor does
// NOT attempt to fully solve — where a reference cannot be resolved
// deterministically, it is left unresolved rather than guessed at, and any
// resulting under-resolution is expected to surface via the D-12
// golden-parity diff (testdata/golden/parity_python_test.go) rather than
// being silently masked.
package pyextract
