package daemon

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// daemonPackageSource parses every non-test .go file in the current
// directory — which `go test` sets to the package directory — with
// go/parser. This is the same "parse repository source and assert a
// structural property" discipline internal/upgrade/taskfile_shape_test.go
// and internal/mcp/tools_schema_drift_test.go's parseQueryConstants
// already use: a passing `go test -race` run proves only that no race
// happened in THAT run, never that the design makes one impossible.
func daemonPackageSource(t *testing.T) (*token.FileSet, map[string]*ast.File) {
	t.Helper()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("os.ReadDir(.): %v", err)
	}

	fset := token.NewFileSet()
	files := make(map[string]*ast.File)
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parser.ParseFile(%s): %v", name, err)
		}
		files[name] = f
	}
	return fset, files
}

// TestWatchdogSeamShape is the D-13 structural guard: the watchdog's
// parent-pid seam must be a per-instance Daemon field threaded through
// startWatchdog's parameters, never a package-level mutable binding. A
// passing `go test -race` run proves only that no race happened in one
// run; this guard proves the design makes the race structurally
// impossible by asserting the shape of the source itself, so a future
// reintroduction of the global fails loudly here rather than waiting to
// be caught by luck or code review.
func TestWatchdogSeamShape(t *testing.T) {
	fset, files := daemonPackageSource(t)

	t.Run("no_package_level_parent_pid_seam", func(t *testing.T) {
		for name, f := range files {
			for _, decl := range f.Decls {
				gd, ok := decl.(*ast.GenDecl)
				if !ok || gd.Tok != token.VAR {
					continue
				}
				for _, spec := range gd.Specs {
					vs, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for _, n := range vs.Names {
						if n.Name == "getppid" {
							pos := fset.Position(n.Pos())
							t.Fatalf("D-13: found package-level `var getppid` at internal/daemon/%s:%d — the parent-pid reader must be a per-instance Daemon field (unexported, no exported setter, mirroring the onSync/onSyncStart/syncFn/onWatchOpen convention) so the FIX-08 race is structurally impossible, not merely serialized by test join discipline", name, pos.Line)
						}
					}
				}
			}
		}
	})

	t.Run("daemon_carries_parent_pid_reader_field", func(t *testing.T) {
		f, ok := files["daemon.go"]
		if !ok {
			t.Fatal("internal/daemon/daemon.go was not parsed — cannot inspect the Daemon struct")
		}
		ts := findTypeSpec(f, "Daemon")
		if ts == nil {
			t.Fatal("no `type Daemon struct` found in internal/daemon/daemon.go")
		}
		st, ok := ts.Type.(*ast.StructType)
		if !ok {
			t.Fatal("Daemon is declared but is not a struct type")
		}
		var fieldNames []string
		for _, field := range st.Fields.List {
			for _, fn := range field.Names {
				fieldNames = append(fieldNames, fn.Name)
			}
			if isNiladicIntFunc(field.Type) {
				return // found it — subtest passes
			}
		}
		t.Fatalf("D-13: Daemon has no field of type `func() int` (the injected per-instance parent-pid reader) — existing fields: %v", fieldNames)
	})

	t.Run("start_watchdog_takes_injected_seam_and_ticks", func(t *testing.T) {
		f, ok := files["watchdog.go"]
		if !ok {
			t.Fatal("internal/daemon/watchdog.go was not parsed — cannot inspect startWatchdog")
		}
		fn := findFuncDecl(f, "startWatchdog")
		if fn == nil {
			t.Fatal("no `func startWatchdog` found in internal/daemon/watchdog.go")
		}
		var haveFunc, haveChan bool
		var paramTypes []string
		if fn.Type.Params != nil {
			for _, field := range fn.Type.Params.List {
				paramTypes = append(paramTypes, exprString(field.Type))
				if isNiladicIntFunc(field.Type) {
					haveFunc = true
				}
				if isRecvTimeChan(field.Type) {
					haveChan = true
				}
			}
		}
		if !haveFunc || !haveChan {
			t.Fatalf("D-13/D-14: startWatchdog's parameter types = %v, want a `func() int` parameter (injected parent-pid reader) AND a `<-chan time.Time` parameter (injected tick source) among them", paramTypes)
		}
	})

	t.Run("guard_parsed_a_nonempty_file_set", func(t *testing.T) {
		if len(files) == 0 {
			t.Fatal("parsed zero non-test .go files in the current directory — this shape guard would pass vacuously; check the working directory `go test` sets for this package")
		}
		if _, ok := files["watchdog.go"]; !ok {
			t.Error("internal/daemon/watchdog.go was not among the parsed files")
		}
		if _, ok := files["daemon.go"]; !ok {
			t.Error("internal/daemon/daemon.go was not among the parsed files")
		}
	})
}

// findTypeSpec returns the *ast.TypeSpec for the top-level type declaration
// named name in f, or nil if none exists.
func findTypeSpec(f *ast.File, name string) *ast.TypeSpec {
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		for _, spec := range gd.Specs {
			if ts, ok := spec.(*ast.TypeSpec); ok && ts.Name.Name == name {
				return ts
			}
		}
	}
	return nil
}

// findFuncDecl returns the top-level (non-method) function declaration
// named name in f, or nil if none exists.
func findFuncDecl(f *ast.File, name string) *ast.FuncDecl {
	for _, decl := range f.Decls {
		if fd, ok := decl.(*ast.FuncDecl); ok && fd.Recv == nil && fd.Name.Name == name {
			return fd
		}
	}
	return nil
}

// isNiladicIntFunc reports whether expr is exactly `func() int` — no
// parameters, a single unnamed int result.
func isNiladicIntFunc(expr ast.Expr) bool {
	ft, ok := expr.(*ast.FuncType)
	if !ok {
		return false
	}
	if ft.Params != nil && len(ft.Params.List) > 0 {
		return false
	}
	if ft.Results == nil || len(ft.Results.List) != 1 {
		return false
	}
	result := ft.Results.List[0]
	if len(result.Names) != 0 {
		return false
	}
	ident, ok := result.Type.(*ast.Ident)
	return ok && ident.Name == "int"
}

// isRecvTimeChan reports whether expr is exactly `<-chan time.Time`.
func isRecvTimeChan(expr ast.Expr) bool {
	ct, ok := expr.(*ast.ChanType)
	if !ok || ct.Dir != ast.RECV {
		return false
	}
	sel, ok := ct.Value.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Time" {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	return ok && ident.Name == "time"
}

// exprString renders a small subset of ast.Expr shapes for diagnostic
// messages — it does not need to be exhaustive, only readable in a test
// failure.
func exprString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.FuncType:
		return "func(...)"
	case *ast.ChanType:
		dir := "chan "
		if e.Dir == ast.RECV {
			dir = "<-chan "
		}
		return dir + exprString(e.Value)
	case *ast.SelectorExpr:
		return exprString(e.X) + "." + e.Sel.Name
	case *ast.StarExpr:
		return "*" + exprString(e.X)
	default:
		return "?"
	}
}
