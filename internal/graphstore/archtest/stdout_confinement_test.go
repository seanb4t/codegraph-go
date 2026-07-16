// Package archtest additionally enforces HYG-02/D-06(b): no serve-reachable
// package may write to os.Stdout — stdout is reserved exclusively for the
// MCP JSON-RPC transport (internal/mcp/server.go's WarnUnknownToolsTo doc
// comment, T-03-07-Leak). internal/cli is DELIBERATELY excluded from this
// guard: it legitimately renders product output via cmd.OutOrStdout()
// (03-PATTERNS.md output-discipline pattern) and is never reachable from
// `serve --mcp`.
//
// The three detection predicates below resolve identifiers through
// go/types (info.Uses + types.Object.Pkg().Path()), NOT string-matching
// the source — the same package-graph precision import_graph_test.go
// already applies at the import level (RESEARCH Anti-Patterns). This
// avoids false positives on comments, string literals, aliased imports,
// and locally shadowed identifiers named "os"/"fmt"/"log". The predicates
// themselves are proven able to fail by
// stdout_detection_selftest_test.go's TestStdoutGuardDetectsViolations.
//
// Residual risk (IN-01): these predicates only match direct,
// statically-resolvable identifier references. A write via
// os.NewFile(1, "").Write(...), syscall.Write(1, ...), or an os.Stdout
// value captured earlier and threaded through an unrelated variable or
// interface would NOT be detected by any of the three predicates. This is
// an inherent limitation of AST/identifier-based static analysis, not a
// fixable defect of this guard — such indirect writes would only be
// caught (if at all) by mcp_stdout_purity_test.go's runtime check of the
// actual `serve --mcp` subprocess's stdout.
package archtest

import (
	"go/ast"
	"go/types"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

// guardedPackages are the six serve-reachable packages HYG-02 locks down
// (D-06b): every package that can execute during a `serve --mcp` session
// (the MCP transport itself, the read-only query engine it delegates to,
// the storage layer it reads through, and the background
// daemon/watch/indexer machinery a live session can trigger via the
// startup reconcile or a debounced sync) must never write to stdout.
//
// internal/cli is DELIBERATELY excluded: it legitimately renders product
// output via cmd.OutOrStdout() (03-PATTERNS.md output-discipline
// pattern) and is never reachable from `serve --mcp` itself.
var guardedPackages = []string{
	"github.com/seanb4t/codegraph-go/internal/mcp",
	"github.com/seanb4t/codegraph-go/internal/graphstore",
	"github.com/seanb4t/codegraph-go/internal/daemon",
	"github.com/seanb4t/codegraph-go/internal/watch",
	"github.com/seanb4t/codegraph-go/internal/indexer",
	"github.com/seanb4t/codegraph-go/internal/query",
}

const modulePathPrefix = "github.com/seanb4t/codegraph-go/"

// excludedInternalPackagePrefixes are module-internal packages that must
// NEVER be pulled into the closure even if some guardedPackage comes to
// (directly or transitively) import them. internal/cli is the one
// documented exclusion (D-06b): it legitimately renders product output via
// cmd.OutOrStdout() and is never reachable from `serve --mcp` itself.
var excludedInternalPackagePrefixes = []string{
	"github.com/seanb4t/codegraph-go/internal/cli",
}

// isModuleInternalPackage reports whether pkgPath belongs to this module
// (so it's a candidate for closure scanning) and is not internal/cli or a
// subpackage of it (the D-06b exclusion). Third-party and stdlib imports
// (fmt, os, log, golang.org/x/tools/..., etc.) always return false here —
// the closure walk only ever descends into this module's own packages.
func isModuleInternalPackage(pkgPath string) bool {
	if !strings.HasPrefix(pkgPath, modulePathPrefix) {
		return false
	}
	for _, excluded := range excludedInternalPackagePrefixes {
		if pkgPath == excluded || strings.HasPrefix(pkgPath, excluded+"/") {
			return false
		}
	}
	return true
}

// closeOverServeReachableImports loads the six guardedPackages (with
// packages.NeedDeps, so dependency packages get full Syntax/TypesInfo, not
// just name/ID metadata) and returns the full transitive, module-internal
// import closure reachable from them: the six roots themselves plus every
// package they import, directly or indirectly, that belongs to this module
// and is not internal/cli.
//
// This closes CR-01: loading only the six roots without NeedDeps, and then
// only ever ranging over the root pkgs slice (never descending into
// pkg.Imports), left every transitively-imported package — internal/schema,
// internal/gitmeta, internal/parser, every internal/indexer/*extract
// package, etc. — completely unscanned despite being genuinely reachable
// during a live `serve --mcp` session.
//
// overlay is passed straight through to packages.Config.Overlay; production
// callers pass nil, stdout_closure_selftest_test.go passes a synthetic
// in-memory violation to prove the closure walk actually reaches into a
// dependency (not just the six named roots).
func closeOverServeReachableImports(t *testing.T, overlay map[string][]byte) map[string]*packages.Package {
	t.Helper()

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedImports | packages.NeedDeps | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
		Overlay: overlay,
		// Tests: false (deliberate divergence — see package doc comment).
	}
	pkgs, err := packages.Load(cfg, guardedPackages...)
	if err != nil {
		t.Fatalf("packages.Load: %v", err)
	}

	// Pitfall-4 sanity check: packages.Load can silently resolve fewer
	// packages than requested (typo, wrong module prefix, a future rename
	// of one of the six guarded packages) without an obvious error. Fail
	// loudly rather than let the guard become vacuously green.
	if len(pkgs) != len(guardedPackages) {
		t.Fatalf("packages.Load resolved %d packages, want %d (guardedPackages) — a guarded package may have been renamed or moved, silently disabling this test's coverage of it", len(pkgs), len(guardedPackages))
	}

	reachable := make(map[string]*packages.Package)
	var walk func(pkg *packages.Package)
	walk = func(pkg *packages.Package) {
		if _, seen := reachable[pkg.PkgPath]; seen {
			return
		}
		reachable[pkg.PkgPath] = pkg
		for path, imp := range pkg.Imports {
			if !isModuleInternalPackage(path) {
				continue
			}
			walk(imp)
		}
	}
	for _, pkg := range pkgs {
		walk(pkg)
	}

	for _, pkg := range reachable {
		if len(pkg.Syntax) == 0 {
			t.Fatalf("%s: packages.Load resolved zero syntax files — this guard cannot check a package it never parsed", pkg.PkgPath)
		}
	}

	return reachable
}

// scanForStdoutViolations walks every file in pkgs and returns one message
// per detected violation (bare os.Stdout reference, bare fmt.Print*, or
// log.SetOutput call) using the same three predicates
// stdout_detection_selftest_test.go proves can actually fire.
func scanForStdoutViolations(pkgs map[string]*packages.Package) []string {
	var violations []string
	for _, pkg := range pkgs {
		for _, f := range pkg.Syntax {
			ast.Inspect(f, func(n ast.Node) bool {
				switch expr := n.(type) {
				case *ast.SelectorExpr:
					if isOSStdoutRef(expr, pkg.TypesInfo) {
						violations = append(violations, pkg.PkgPath+": references os.Stdout — diagnostics must use an explicit stderr writer seam, never stdout directly (HYG-02)")
					}
				case *ast.CallExpr:
					if isBareFmtPrint(expr, pkg.TypesInfo) {
						violations = append(violations, pkg.PkgPath+": calls a bare fmt.Print*/Printf/Println (no explicit writer) — stdout is reserved for the MCP JSON-RPC transport (HYG-02)")
					}
					if isLogSetOutput(expr, pkg.TypesInfo) {
						violations = append(violations, pkg.PkgPath+": calls log.SetOutput — must never redirect stdlib log's default stderr output (HYG-02)")
					}
				}
				return true
			})
		}
	}
	return violations
}

// TestNoStdoutNoiseInServeReachablePackages loads the full serve-reachable
// import closure of the six guardedPackages via go/packages (CR-01: not
// just their own files — every module-internal package they transitively
// import, internal/cli excluded) and fails if any production file in that
// closure references os.Stdout, calls a bare fmt.Print*, or calls
// log.SetOutput (D-06b). Expected GREEN today — D-07's zero-violation
// baseline; if this test ever surfaces a real violation, fixing it (route
// the offending write to an explicit stderr writer) is in scope, not
// something to suppress or exclude.
//
// Mode divergence from the two existing archtest precedents
// (import_graph_test.go, modernc_confinement_test.go): those use
// Tests: true because an import-graph bypass could be hidden inside some
// OTHER package's _test.go file, which the import graph must still catch.
// This guard is scoped to the six NAMED roots' own reachability from the
// real `serve --mcp` binary — a _test.go file printing to stdout during
// `go test` never touches that binary's stdout, so Tests: false is the
// correct scope here (RESEARCH Open Question #2). This also avoids false
// positives on this package's own test helpers: logger_test.go's
// mutation-proof wiring test calls log.SetOutput intentionally (to
// redirect stdlib log's default output into a capture buffer for the
// duration of a test), and indexer's _test.go files use fmt.Println — both
// are legitimate test-only constructs Tests: false correctly excludes from
// production-path scanning.
func TestNoStdoutNoiseInServeReachablePackages(t *testing.T) {
	reachable := closeOverServeReachableImports(t, nil)

	// CR-01 regression guard: the closure must have actually grown beyond
	// the six named roots, and must include at least one package known
	// (per the CR-01 review's own reproduction) to be transitively
	// imported by every guarded package — otherwise the walk above
	// silently stopped growing the reachable set, and this guard would be
	// vacuously blind to dependency violations again, for the exact same
	// reason CR-01 flagged.
	if len(reachable) <= len(guardedPackages) {
		t.Fatalf("closure over guardedPackages resolved only %d packages (the six roots) — the transitive-import walk did not grow the reachable set; this guard would be vacuously blind to violations in dependencies (CR-01)", len(reachable))
	}
	const mustBeReachable = "github.com/seanb4t/codegraph-go/internal/schema"
	if _, ok := reachable[mustBeReachable]; !ok {
		t.Fatalf("%s was not found in the closure — it is transitively imported by every guardedPackage; its absence means the closure walk is broken (CR-01)", mustBeReachable)
	}

	for _, msg := range scanForStdoutViolations(reachable) {
		t.Errorf("%s", msg)
	}
}

// isOSStdoutRef reports whether sel is a qualified-identifier reference
// to the os package's Stdout var (os.Stdout, or via an aliased import —
// e.g. myos.Stdout — since resolution goes through the imported
// package's path, not the identifier's local name). It does NOT flag
// os.Stderr, a string/comment containing the text "os.Stdout", or a
// locally shadowed identifier named os (e.g. a struct field named Stdout
// on a shadowing local variable): that field-selector case resolves
// sel.X through info.Uses to a non-package object, so the *types.PkgName
// type assertion below fails and the reference is correctly ignored.
func isOSStdoutRef(sel *ast.SelectorExpr, info *types.Info) bool {
	if sel.Sel.Name != "Stdout" {
		return false
	}
	xIdent, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	pkgName, ok := info.Uses[xIdent].(*types.PkgName)
	if !ok || pkgName.Imported().Path() != "os" {
		return false
	}
	obj, ok := info.Uses[sel.Sel]
	if !ok || obj.Pkg() == nil || obj.Pkg().Path() != "os" {
		return false
	}
	return true
}

// isBareFmtPrint reports whether call invokes fmt.Print, fmt.Printf, or
// fmt.Println — the three fmt functions that write directly to stdout
// with no explicit io.Writer argument. It does NOT flag fmt.Fprint /
// fmt.Fprintf / fmt.Fprintln (an explicit writer is supplied — this is
// how the codebase's diagnostics seams already route to stderr) nor
// fmt.Sprintf (returns a string, writes nothing).
func isBareFmtPrint(call *ast.CallExpr, info *types.Info) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	switch sel.Sel.Name {
	case "Print", "Printf", "Println":
	default:
		return false
	}
	xIdent, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	pkgName, ok := info.Uses[xIdent].(*types.PkgName)
	if !ok || pkgName.Imported().Path() != "fmt" {
		return false
	}
	obj, ok := info.Uses[sel.Sel]
	if !ok || obj.Pkg() == nil || obj.Pkg().Path() != "fmt" {
		return false
	}
	return true
}

// isLogSetOutput reports whether call invokes log.SetOutput — the
// stdlib log package's default output is already os.Stderr
// ($GOROOT/src/log/log.go's `var std = New(os.Stderr, ...)`), so a call
// redirecting it is either pointless or a regression risk (D-06 note).
// It does NOT flag log.Printf / log.Println (compliant — they use std's
// default stderr output unless something already called SetOutput).
func isLogSetOutput(call *ast.CallExpr, info *types.Info) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	if sel.Sel.Name != "SetOutput" {
		return false
	}
	xIdent, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	pkgName, ok := info.Uses[xIdent].(*types.PkgName)
	if !ok || pkgName.Imported().Path() != "log" {
		return false
	}
	obj, ok := info.Uses[sel.Sel]
	if !ok || obj.Pkg() == nil || obj.Pkg().Path() != "log" {
		return false
	}
	return true
}
