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
package archtest

import (
	"go/ast"
	"go/types"
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

// TestNoStdoutNoiseInServeReachablePackages loads the six guardedPackages'
// type-checked syntax via go/packages and fails if any production file
// references os.Stdout, calls a bare fmt.Print*, or calls log.SetOutput
// (D-06b). Expected GREEN today — D-07's zero-violation baseline; if this
// test ever surfaces a real violation, fixing it (route the offending
// write to an explicit stderr writer) is in scope, not something to
// suppress or exclude.
//
// Mode divergence from the two existing archtest precedents
// (import_graph_test.go, modernc_confinement_test.go): those use
// Tests: true because an import-graph bypass could be hidden inside some
// OTHER package's _test.go file, which the import graph must still catch.
// This guard is scoped to six NAMED packages' own reachability from the
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
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedImports | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
		// Tests: false (deliberate divergence — see doc comment above).
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
	for _, pkg := range pkgs {
		if len(pkg.Syntax) == 0 {
			t.Fatalf("%s: packages.Load resolved zero syntax files — this guard cannot check a package it never parsed", pkg.PkgPath)
		}
	}

	for _, pkg := range pkgs {
		for _, f := range pkg.Syntax {
			ast.Inspect(f, func(n ast.Node) bool {
				switch expr := n.(type) {
				case *ast.SelectorExpr:
					if isOSStdoutRef(expr, pkg.TypesInfo) {
						t.Errorf("%s: references os.Stdout — diagnostics must use an explicit stderr writer seam, never stdout directly (HYG-02)", pkg.PkgPath)
					}
				case *ast.CallExpr:
					if isBareFmtPrint(expr, pkg.TypesInfo) {
						t.Errorf("%s: calls a bare fmt.Print*/Printf/Println (no explicit writer) — stdout is reserved for the MCP JSON-RPC transport (HYG-02)", pkg.PkgPath)
					}
					if isLogSetOutput(expr, pkg.TypesInfo) {
						t.Errorf("%s: calls log.SetOutput — must never redirect stdlib log's default stderr output (HYG-02)", pkg.PkgPath)
					}
				}
				return true
			})
		}
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
