// Package tsextract walks a TypeScript, TSX, or JavaScript file's
// tree-sitter syntax tree into the shared codegraph vocabulary
// (goextract.FileResult/ExtractedNode/IntraEdge/UnresolvedRef,
// goextract.Kind*/RefKind*) — ONE extractor serving all three grammars
// registered by languages_typescript.go ("typescript" .ts, "tsx" .tsx,
// "javascript" .js/.jsx/.mjs/.cjs), mirroring javaextract's/csharpextract's/
// pyextract's shape (D-01) rather than defining three near-duplicate
// packages. The three grammars share near-identical ES statement/expression
// node kinds (verified directly against each module's own node-types.json,
// not assumed) — a single tree-walk correctly handles all three.
//
// # Node-kind mapping decisions
//
//   - class_declaration / abstract_class_declaration -> goextract.KindStruct
//     (not a new "class" kind) — mirrors every priority-4 sibling's own
//     class->KindStruct decision, keeping struct/class-shaped downstream
//     consumers (Wave 6's implements synthesis) language-agnostic.
//   - interface_declaration -> goextract.KindInterface. TS/TSX only — the
//     JavaScript grammar has no interface_declaration node kind at all, so
//     this simply never fires on a .js/.jsx file.
//   - type_alias_declaration -> goextract.KindTypeAlias. TS/TSX only.
//   - function_declaration / generator_function_declaration ->
//     goextract.KindFunction.
//   - method_definition (a class body's own direct child) ->
//     goextract.KindMethod, QualifiedName "Type.method".
//   - public_field_definition (TS) / field_definition (JS) -> NO node is
//     ever emitted (mirrors goextract's/every sibling's ratified "no field
//     node" skip).
//   - A top-level `export const NAME = ...` whose value is an
//     arrow_function/function_expression/generator_function (the dominant
//     modern TS/JS idiom for exported functions — React components, Express
//     handlers, ...) -> goextract.KindFunction, exactly as if it had been
//     written `export function NAME() {}`. A top-level `export const NAME =
//     <anything else>` -> goextract.KindConstant. Per this plan's own scope
//     ("exported consts as appropriate"), a NON-exported top-level
//     const/let/var declaration emits no node at all — a deliberately
//     bounded scope decision, not an oversight; unlike Go's const/var
//     (extracted regardless of export, since Go models visibility
//     separately from extraction), ES module top-level bindings that are
//     never exported can never be a cross-file reference target, so
//     extracting them would add nodes no resolution pass could ever reach.
//   - class_heritage's `extends`/`implements` targets (TS: separate
//     extends_clause/implements_clause children; JS: a single direct
//     expression child, no `implements` keyword at all in JS) and an
//     interface's extends_type_clause -> a single goextract.RefKindEmbeds
//     unresolved ref per listed supertype (RESEARCH Pattern 2: extends and
//     implements are NOT distinguished at parse time; promoting an "embeds"
//     edge to "implements" once the target's Kind is known is Wave 6's
//     resolve-time job).
//   - import_statement, and an `export { X } from '...'` re-export ->
//     goextract.RefKindImports, one unresolved ref per statement (dependency
//     tracking, mirroring every sibling).
//   - call_expression -> goextract.RefKindCalls.
//
// # Cross-file module resolution (the tsconfig-aware ModuleKey hook)
//
// TS/JS's cross-file identity is "the resolved module specifier" (RESEARCH
// "Don't Hand-Roll") — the LEAST directory-structure-stable of the
// priority-4 languages, and (like Python, unlike Java's package/C#'s
// namespace) a per-FILE identity, not a per-directory-shared one: every
// source file IS its own module. languages_typescript.go's LanguageSpec.
// ModuleKey computes this file's own canonical key as NormalizeModuleKey
// (this package, exported) applied to its relPath — the extension-stripped,
// slash-separated, repo-root-relative path — UNCONDITIONALLY (regardless of
// whether a tsconfig.json/package.json descriptor was found). This is a
// deliberate divergence from every other priority-4 sibling's "nil
// descriptor -> raw relPath" fallback convention: TS/JS's specifier
// resolution below (used to compute a REFERENCED file's key from an import
// statement) is baseUrl-INVARIANT relative-path arithmetic, so this file's
// OWN key must be computed the exact same extension-stripped way in EVERY
// case, descriptor present or not, or a relative import in a repo with no
// tsconfig.json/package.json at all could never match the target file's own
// key.
//
// Extract's shared cross-language signature (Extract(p, moduleKey, relPath,
// src), established in Phase 5's seam and unchanged by four other
// languages already) carries no descriptor parameter — a genuine
// architectural constraint of this plan's file scope (languages.go/
// extract.go are not modified here). tsconfig.json's `paths`/`baseUrl`
// table, needed to resolve a NON-relative (`@app/foo`) import specifier,
// therefore cannot be threaded through Extract's own parameters. This
// package instead exposes a small package-level, mutex-guarded singleton
// (Config/SetConfig/getConfig) that languages_typescript.go's Descriptor
// hook populates once per repo root — the SAME "resolve a repo-root
// manifest once, before any Extract call for that run" ordering Discover
// already guarantees for every language's Descriptor (discover.go: every
// language's Descriptor is resolved BEFORE Extract's worker pool ever
// starts). A single-process indexing run only ever targets one repo root at
// a time, matching this project's existing Discover/Extract/Resolve
// pipeline's own single-repo-per-invocation invariant — this is a bounded,
// documented pragmatic choice (not a general multi-root-safe design), not a
// silent global-state hazard.
//
// Given a specifier and the importing file's own relPath, resolveModuleSpecifier
// resolves it in this order (RESEARCH's own "Don't Hand-Roll" TS/JS
// resolution-order table):
//
//  1. A relative specifier (`./foo`, `../bar/baz`) — pure path arithmetic
//     against the importing file's own directory, needing no descriptor at
//     all. This is the PRIORITY tier per this plan's critical constraints.
//  2. A tsconfig.json `paths`-aliased specifier (`@app/foo` matching a
//     `"@app/*": ["src/app/*"]` entry) — rewritten via the FIRST configured
//     target pattern only (TS's own multi-candidate fallback-list semantics
//     for build-time resolution are out of this bounded extractor's scope),
//     then treated as baseUrl-relative.
//  3. A bare specifier with no `paths` match, when `baseUrl` is configured
//     to something other than "." — best-effort baseUrl-relative resolution
//     (TS's own "non-relative specifier resolves against baseUrl when no
//     path mapping applies" rule).
//  4. Anything else (an ordinary `node_modules` package import, e.g.
//     `import {useState} from 'react'`) is NOT resolved to an intra-repo
//     file — package.json `main`/`exports`-map resolution into a
//     node_modules dependency tree is explicitly out of this bounded
//     extractor's scope (RESEARCH Assumptions Log A1 anticipates this exact
//     edge case: "TS's `exports` map conditional resolution"). A
//     RefKindImports dependency ref is still emitted (for the specifier
//     TEXT, matching every sibling's own external-import behavior), but no
//     Imports[alias] entry is populated — resolve.go's own pre-existing,
//     cross-language isIntraModule check (gated on the Go-specific
//     descriptors["go"].ModulePath(), a pre-existing gap this plan's file
//     scope does not touch — see the plan's own critical_constraints and
//     C#/Java's own prior documentation of the identical gap) means an
//     "imports" edge itself never lands for ANY non-Go language today
//     regardless of what this extractor emits; this only affects the
//     "imports" EDGE, not calls/embeds resolution via Imports, which is
//     this extractor's own, fully independent mechanism.
//
// A tsconfig.json's `extends` chain (one config inheriting another) is NOT
// followed — only the target file's OWN compilerOptions are read. A
// tsconfig.json containing comments or trailing commas (JSONC, extremely
// common in real repos) simply fails encoding/json.Unmarshal; this
// extractor degrades to an EMPTY (no baseUrl, no paths) config rather than
// hand-rolling a JSONC parser, exactly matching T-05-Manifest's
// "malformed config -> path-identity fallback, never a crash" contract.
// Both are explicit, documented, accepted gaps (RESEARCH Assumptions Log
// A1), not silent guesses.
//
// A directory-style import (`import x from './utils'` resolving to
// `utils/index.ts`) is NOT resolved — this extractor's specifier resolution
// is pure path arithmetic with no filesystem access at Extract time to
// determine whether a path segment names a file or a directory. This is a
// deliberate, bounded, documented gap, exactly the shape of gap every
// priority-4 sibling's own resolver documents rather than guesses at.
//
// # Named-import call/heritage resolution (the ES-modules-specific mechanism)
//
// Unlike Go's `pkg.Func()`/Java's `Helper.assist()`/C#'s
// `Namespace.Type.Method()`/Python's `Helper.assist()` — ALL of which
// qualify a cross-file reference via a DIFFERENT identifier than the
// referenced symbol's own name — ES modules' dominant idiom binds an
// imported name DIRECTLY into local scope (`import { Foo } from './foo';
// Foo()`), so the call site's own identifier text is simultaneously "which
// local binding" AND (for an unaliased named import) "the target module's
// own declared symbol name." This extractor tracks, alongside
// FileResult.Imports (local alias -> target moduleKey, populated for every
// import shape: default, namespace, and named), a second, TS/JS-internal-
// only map (extractor.namedImportOrigin: local alias -> the target module's
// OWN declared symbol name) for default and named imports ONLY — a
// namespace import (`import * as NS from './foo'`) needs no such mapping,
// since it is only ever referenced via member access (`NS.Foo()`), which
// resolves through the ordinary Imports-membership check, identical to
// every other priority-4 language's own qualified-call resolution.
//
// A bare identifier call (`Foo()`) or a bare heritage-clause identifier
// (`class Derived extends Foo`) first checks namedImportOrigin: a hit
// emits UnresolvedRef{PkgAlias: localAlias, Name: originModuleOwnName} —
// PkgAlias must be a literal key in FileResult.Imports (resolveSelector's
// own, unchanged, narrowest-safe-set contract), and Name is looked up
// WITHIN that target module's own symbol bucket. A miss falls through to
// an ordinary unqualified (empty PkgAlias) same-module resolution attempt
// — the same fallback every sibling's bare-identifier-call handling uses.
//
// A DEFAULT import (`import Foo from './foo'`) is resolved under the SAME
// mechanism, with origin == the local alias itself: this extractor tracks
// no per-module "the default export"'s own declared name distinct from
// that declaration's own name (`export default class Foo {}` extracts a
// node literally named "Foo", the same as any other class declaration) — a
// default import resolves correctly whenever the local binding text
// coincides with the target's own declared symbol name (the overwhelmingly
// common `export default class Foo {}` + `import Foo from './foo'` idiom).
// A RENAMED default import (`import Renamed from './foo'`) is an explicit,
// documented, accepted gap — this extractor has no per-module "default
// export identity" to consult, and resolving it correctly would require
// either whole-module symbol-table awareness this Pass-1, single-file
// extractor does not have, or a resolve-time convention this plan's file
// scope (tsextract + languages_typescript.go only) does not touch.
//
// A qualified member-access call/heritage reference (`X.method()`, `class
// Derived extends X.Base`) resolves its PkgAlias via: a real import alias
// (any shape) in Imports -> qualified; otherwise an uppercase-leading
// (PascalCase, the near-universal TS/JS class-naming convention, mirroring
// every priority-4 sibling's identical heuristic) identifier -> a
// same-module (empty PkgAlias) attempt; otherwise a synthetic non-matching
// alias, mirroring goextract's own WR-02 fix, so a local-variable/parameter
// receiver (`myWidget.render()`) deterministically falls through to
// "unresolved" rather than risking a same-module false match.
package tsextract

import "sync"

// Config carries the tsconfig.json compilerOptions fields this extractor's
// module-specifier resolution consults — BaseURL and Paths — resolved once
// per repo root by languages_typescript.go's Descriptor hook via SetConfig
// (see this file's own package doc, "Cross-file module resolution", for the
// full architectural rationale for this package-level singleton).
type Config struct {
	// BaseURL is tsconfig.json's compilerOptions.baseUrl, "" when absent or
	// unconfigured (in which case only relative-specifier and paths-aliased
	// resolution apply — see resolveModuleSpecifier).
	BaseURL string
	// Paths is tsconfig.json's compilerOptions.paths: an alias PATTERN (at
	// most one "*" wildcard, TS's own documented convention) -> an ordered
	// list of target patterns. Only the FIRST target pattern per alias is
	// consulted (TS's own multi-candidate build-time fallback-list
	// semantics are out of this bounded extractor's scope).
	Paths map[string][]string
}

var (
	configMu     sync.RWMutex
	globalConfig Config
)

// SetConfig installs cfg as the module-specifier-resolution config every
// subsequent Extract call (in this process) consults, until the next
// SetConfig call. Called once per repo root by languages_typescript.go's
// Descriptor hook, itself called once per Discover run and always BEFORE
// Discover's caller ever invokes Extract for that same run (discover.go's
// own documented per-language Descriptor-then-ModuleKey-then-later-Extract
// ordering) — see this file's package doc for the full single-repo-per-
// process rationale.
func SetConfig(cfg Config) {
	configMu.Lock()
	defer configMu.Unlock()
	globalConfig = cfg
}

// getConfig returns the currently installed Config (the zero value —
// no baseUrl, no paths — if SetConfig has never been called).
func getConfig() Config {
	configMu.RLock()
	defer configMu.RUnlock()
	return globalConfig
}
