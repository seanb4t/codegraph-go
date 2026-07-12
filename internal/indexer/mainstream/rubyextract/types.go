// Package rubyextract walks a Ruby file's tree-sitter syntax tree into the
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
//   - class -> goextract.KindStruct.
//   - module -> goextract.KindStruct (documented gap: this vocabulary has no
//     distinct "module"/namespace kind; Ruby's module — used both as a
//     namespace and as a mixin — is the closest semantic analog to what
//     KindStruct represents elsewhere).
//   - A method/singleton_method declared directly in a class/module's own
//     body -> goextract.KindMethod (QualifiedName "ScopeName.methodname"); a
//     TOP-LEVEL (not nested in any class/module) method/singleton_method ->
//     goextract.KindFunction. Ruby's own top-level "main object" methods are
//     genuinely callable as bare functions, matching this convention.
//   - Ruby has no field_declaration-equivalent node this extractor visits at
//     all — instance-variable assignments (`@size = 1`) are simply never
//     walked, matching the ratified "no field node" skip every other
//     extractor in this project already applies.
//   - require/require_relative are ordinary method calls in Ruby's own
//     grammar (no dedicated import-statement node kind exists) — this
//     extractor recognizes the `require("...")`/`require_relative("...")`
//     call SHAPE specifically (a receiverless call whose method name is
//     exactly "require"/"require_relative" and whose sole argument is a
//     static string literal) and emits goextract.RefKindImports instead of
//     goextract.RefKindCalls for it, so it is never double-counted as both
//     an import and a call. A DYNAMIC require path (string interpolation,
//     a variable, `Kernel.require`) is not recognized at all — an accepted,
//     documented gap. `require_relative` additionally resolves its target
//     to a directory-relative moduleKey and records it in FileResult.Imports
//     (see languages_ruby.go's unconditional path-based ModuleKey); plain
//     `require` (a $LOAD_PATH-relative gem/stdlib name, not a file path) is
//     recorded as a dependency ref only, never resolved.
//   - A class's declared superclass (`class Derived < Base`) ->
//     goextract.RefKindEmbeds, undistinguished from any other embed shape at
//     parse time (Pattern 2) — Ruby modules included via `include`/`extend`
//     (a runtime method call, not a static declaration) are NOT extracted as
//     embeds at all, a documented, accepted gap (mixin resolution would
//     require simulating Ruby's method-lookup ancestry chain, out of scope
//     for this tier).
//   - call -> goextract.RefKindCalls (except the require/require_relative
//     shape above). No receiver, or receiver == self -> an implicit
//     same-class call (empty PkgAlias). A `constant`-receiver call
//     (`Widget.build`) is treated as a same-module attempt (empty PkgAlias)
//     since Ruby constants are PascalCase by the language's own convention —
//     this extractor tracks no import-alias table for constants (require
//     binds no name the way Python's `import`/JS's named-import does), so
//     unlike pyextract/tsextract there is no "genuine import alias" branch
//     to check first. Any other receiver (a local variable/parameter/method
//     call result) routes through the WR-02 synthetic-non-matching-alias
//     pattern so it deterministically ends up "unresolved" rather than
//     risking a false same-module match.
//
// Explicitly out of scope for this mainstream tier (named for the D-11
// capability matrix, not silently missing):
//   - A bare, no-parens, no-argument method invocation (`helper` alone, as
//     opposed to `helper()`) is grammatically AMBIGUOUS with a local-variable
//     reference in tree-sitter-ruby's own grammar — it parses as a plain
//     `identifier` node, not a `call` node, unless it takes an explicit
//     argument list. Disambiguating it would require tracking local-variable
//     assignment scope (which name is a variable vs. a method at each point
//     in the method body), out of scope for this tier. This is a REAL,
//     verified grammar-level limitation, not an oversight — `helper()` (with
//     parens) is extracted correctly.
//   - Dynamic method definition/metaprogramming (`define_method`,
//     `method_missing`, `class_eval`, `attr_accessor`-generated accessors) —
//     none of these produce a `method`/`singleton_method` AST node, so no
//     symbol is ever extracted for them.
//   - `include`/`extend`/`prepend` mixin resolution (see above).
//   - A class opened and reopened across multiple files/statements (Ruby's
//     "monkey-patching" idiom) is extracted once per occurrence, each
//     producing its own node under the SAME node id (Rust/Go's own
//     same-name/same-file collision behavior applies identically here — no
//     special reopened-class merging is attempted).
//   - Cross-file `require`/`require_relative`-based call and superclass
//     resolution beyond the directory-relative path match described above:
//     since `require` binds no local alias, a reference like `Widget.build`
//     can only resolve within the SAME file/module — a genuinely cross-file
//     `Widget` defined via `require_relative` elsewhere is NOT resolved
//     through any alias table (there isn't one).
package rubyextract
