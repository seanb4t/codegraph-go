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
)

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
