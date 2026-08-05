// Package archtest enforces VRFY-02: no externally-defined protocol-version
// constant may be referenced anywhere in this module, including _test.go
// files and testdata/golden — a go.mod dependency bump must never move the
// declared MCP protocol version silently (T-03-02). internal/mcp.ProtocolVersion
// is the one sanctioned source of the value (internal/mcp/protocol_version.go).
//
// This guard forbids the CLASS of reference (an externally-defined Go
// constant whose name looks like a protocol-version identifier), not one
// library's import path — it is deliberately identity-agnostic so it keeps
// firing after Phase 2 replaces mark3labs/mcp-go with a different SDK,
// without allowlist maintenance. The complementary guard that IS
// spelling/identity-based is internal/cli/archtest's forbiddenMCPSDKPrefixes
// import-path list (SDK-02) — that one is what still catches a violation
// this guard's name heuristic misses.
//
// What this guard catches: any external, non-module-owned identifier whose
// name matches the case-insensitive pattern "protocol.?version" AND resolves
// to a *types.Const (a Go constant declaration) — e.g. mark3labs/mcp-go's
// mcp.LATEST_PROTOCOL_VERSION.
//
// What this guard CANNOT catch, by construction (documented rather than
// overstated — the earlier claim of zero-maintenance survival across
// arbitrary future SDKs was wrong): an SDK protocol-version constant spelled
// without "protocol" and "version" adjacent in the name — e.g. LatestVersion,
// CurrentRevision, SpecRev. If Phase 2's replacement SDK names its constant
// one of these, this guard will not flag it; internal/cli/archtest's
// import-path list is the fallback that still would, and Phase 2 must
// either confirm the replacement SDK's constant matches this pattern or add
// its spelling to a complementary check deliberately.
//
// The guard is a go/packages + go/types AST walk, never a text search over
// source — this repository has already shipped an inverted text-search gate
// (T-03-03), and a regex over file bytes would miss aliased imports,
// build-tag-gated files, and (per the ./... skips testdata/ finding below)
// entire directories.
//
// Non-vacuity: TestNoExternalProtocolVersionConstantReferences additionally
// asserts the loaded package set contains
// github.com/seanb4t/codegraph-go/testdata/golden explicitly — `go list
// ./...` (and therefore a naive packages.Load(cfg, "./...")) silently skips
// any directory literally named "testdata" (the Go tool's own documented
// convention), so a scan that only passed "./..." would be structurally
// blind to testdata/golden/golden_parity_test.go, one of the six known
// pre-migration sites (GOLDEN-01's failure class). loadWholeModule passes
// that path as a second, explicit pattern for exactly this reason.
//
// Measurement (indicative only, not a threshold — a bare wall-clock number
// pinned in source goes stale the moment CI or developer hardware differs;
// the plan summary is this guard's dated, authoritative record):
// `go test ./internal/mcp/archtest/... -count=1` measured roughly 1.7s on
// the author's machine, 2026-08-05 (Apple Silicon, darwin/arm64, go1.26.5).
// The plan-01-03 SUMMARY.md records the authoritative dated measurement.
package archtest

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"regexp"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

// modulePathPrefix is this module's own import-path root. isModuleOwnedPath
// uses it boundary-aware (never a raw strings.HasPrefix on its own) so a
// lookalike module such as github.com/seanb4t/codegraph-go-fork is correctly
// treated as external rather than admitted by a naive prefix match.
const modulePathPrefix = "github.com/seanb4t/codegraph-go"

// goldenPackagePath is the exact path the GOLDEN-01 non-vacuity check below
// requires be present in the loaded set — see the package doc comment.
const goldenPackagePath = modulePathPrefix + "/testdata/golden"

// protocolVersionNamePattern is the identifier-name heuristic at the heart
// of this guard: case-insensitive "protocol", optionally one arbitrary
// character (covers both "protocolVersion" and "PROTOCOL_VERSION"-style
// separators), then "version". See the package doc comment for what this
// deliberately does and does not catch.
var protocolVersionNamePattern = regexp.MustCompile(`(?i)protocol.?version`)

// isModuleOwnedPath reports whether pkgPath belongs to this module: either
// exactly modulePathPrefix, or prefixed by modulePathPrefix+"/". This is
// boundary-aware specifically to reject a lookalike module path such as
// github.com/seanb4t/codegraph-go-fork, which a raw strings.HasPrefix(pkgPath,
// modulePathPrefix) would incorrectly admit as module-owned.
func isModuleOwnedPath(pkgPath string) bool {
	return pkgPath == modulePathPrefix || strings.HasPrefix(pkgPath, modulePathPrefix+"/")
}

// isExternalProtocolVersionConstant reports whether sel is a reference to
// an externally-defined (non-module-owned) Go constant whose name matches
// protocolVersionNamePattern. It returns true only when ALL of:
//
//   - info.Uses[sel.Sel] resolves to a non-nil object belonging to a
//     non-nil package;
//   - that package's path is not owned by this module (isModuleOwnedPath);
//   - the object's name matches protocolVersionNamePattern; and
//   - the object is a *types.Const — a type-switch arm, not a negative
//     filter. This is deliberately narrower than "external and not a
//     struct field": that broader shape admits functions, types, methods,
//     and package-level vars, which is not what "an SDK-owned
//     protocol-version constant" means and was the source of the
//     false-positive surface flagged in review. req.Params.ProtocolVersion
//     (a *types.Var struct field) must keep compiling at every
//     test/integration site — it is legitimate wire-shape usage, not a
//     constant reference.
//
// A malformed or incomplete input (nil info, sel.Sel not present in
// info.Uses, an object with a nil package) is documented to return false,
// never panic — see TestProtocolVersionGuardHelpersFailLoudly.
func isExternalProtocolVersionConstant(sel *ast.SelectorExpr, info *types.Info) bool {
	if info == nil || sel == nil || sel.Sel == nil {
		return false
	}
	obj, ok := info.Uses[sel.Sel]
	if !ok || obj == nil {
		return false
	}
	pkg := obj.Pkg()
	if pkg == nil {
		return false
	}
	if isModuleOwnedPath(pkg.Path()) {
		return false
	}
	if !protocolVersionNamePattern.MatchString(obj.Name()) {
		return false
	}
	_, isConst := obj.(*types.Const)
	return isConst
}

// violationMessage builds one VRFY-02 violation message naming the rule,
// the offending package/file/line/identifier, and both sanctioned escapes
// — so an eventual false positive is self-explanatory rather than a bare
// t.Errorf with no remediation path.
func violationMessage(pkg *packages.Package, sel *ast.SelectorExpr, obj types.Object) string {
	pos := pkg.Fset.Position(sel.Pos())
	return fmt.Sprintf(
		"VRFY-02: no externally-defined protocol-version constant may be referenced in this module — "+
			"package %s, %s:%d: references %s.%s (package %s). Escape hatches: "+
			"(1) vendor the value into internal/mcp.ProtocolVersion and reference that constant instead, or "+
			"(2) if %s.%s is an unrelated dependency's identifier that merely matches the "+
			"name pattern (?i)protocol.?version, record a reviewed exception in this file's doc comment.",
		pkg.PkgPath, pos.Filename, pos.Line, obj.Pkg().Name(), sel.Sel.Name, obj.Pkg().Path(),
		obj.Pkg().Name(), sel.Sel.Name,
	)
}

// scanForProtocolVersionRefs walks every syntax tree in pkgs and returns one
// violationMessage per *ast.SelectorExpr for which isExternalProtocolVersionConstant
// is true.
func scanForProtocolVersionRefs(pkgs []*packages.Package) []string {
	var violations []string
	for _, pkg := range pkgs {
		for _, f := range pkg.Syntax {
			ast.Inspect(f, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if !isExternalProtocolVersionConstant(sel, pkg.TypesInfo) {
					return true
				}
				obj := pkg.TypesInfo.Uses[sel.Sel]
				violations = append(violations, violationMessage(pkg, sel, obj))
				return true
			})
		}
	}
	return violations
}

// stripTestVariant normalizes the additional PkgPath forms Tests: true
// introduces back to the underlying import path (copied from the house
// pattern in internal/graphstore/archtest/import_graph_test.go), so
// path-equality checks below apply uniformly regardless of which variant
// loaded a given package:
//   - "domain/path [domain/path.test]"      (package compiled for test)
//   - "domain/path_test [domain/path.test]" (external test package)
//   - "domain/path.test"                    (synthesized test main)
func stripTestVariant(pkgPath string) string {
	if i := strings.IndexByte(pkgPath, ' '); i >= 0 {
		pkgPath = pkgPath[:i]
	}
	pkgPath = strings.TrimSuffix(pkgPath, "_test")
	pkgPath = strings.TrimSuffix(pkgPath, ".test")
	return pkgPath
}

// loadWholeModule loads every package in this module (Tests: true, so a
// reference hidden inside a _test.go file is not invisible) plus, as an
// explicit second pattern, testdata/golden — the one known VRFY-02 site the
// Go tool's own `./...` expansion would otherwise silently skip, since it
// treats any directory literally named "testdata" as excluded from
// wildcard matches (GOLDEN-01's failure class; see the package doc
// comment). overlay is passed straight through to packages.Config.Overlay;
// production callers pass nil, protocol_version_selftest_test.go passes a
// synthetic in-memory violation.
//
// NeedDeps is deliberately NOT requested: this module's own packages
// resolve external identifiers (e.g. mcp.LATEST_PROTOCOL_VERSION) through
// their own NeedTypes/NeedTypesInfo without needing the SDK's package
// exposed as a separate entry in the returned slice — measured, not
// assumed, per RESEARCH Pitfall 5. Add it only if a real case demonstrates
// the predicate cannot resolve an identifier without it.
func loadWholeModule(t *testing.T, overlay map[string][]byte) []*packages.Package {
	t.Helper()

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedImports | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
		Tests:   true,
		Overlay: overlay,
	}
	pkgs, err := packages.Load(cfg, modulePathPrefix+"/...", goldenPackagePath)
	if err != nil {
		t.Fatalf("packages.Load: %v", err)
	}
	if len(pkgs) == 0 {
		t.Fatal("packages.Load returned no packages — the module import graph did not resolve")
	}

	foundGolden := false
	for _, pkg := range pkgs {
		if stripTestVariant(pkg.PkgPath) == goldenPackagePath {
			foundGolden = true
			break
		}
	}
	if !foundGolden {
		t.Fatalf("packages.Load did not resolve %s — go's \"./...\" (and modulePathPrefix+\"/...\") expansion "+
			"skips any directory named testdata, so this guard must load it explicitly as a second pattern; "+
			"without it, testdata/golden/golden_parity_test.go's protocol-version reference would be invisible "+
			"to this scan (GOLDEN-01's failure class)", goldenPackagePath)
	}

	return pkgs
}

// TestNoExternalProtocolVersionConstantReferences is VRFY-02's production
// guard: after the six known pre-migration sites are moved onto
// internal/mcp.ProtocolVersion, the whole-module scan (including
// testdata/golden) must return zero violations.
func TestNoExternalProtocolVersionConstantReferences(t *testing.T) {
	pkgs := loadWholeModule(t, nil)

	for _, msg := range scanForProtocolVersionRefs(pkgs) {
		t.Errorf("%s", msg)
	}
}

// newFakeConst, newFakeFunc, newFakeVar, newFakeField, and newFakeTypeName
// build hand-constructed go/types objects for TestIsExternalProtocolVersionConstantMatrix.
// isExternalProtocolVersionConstant only inspects info.Uses[sel.Sel] — never
// sel.X's own structure — so a directly-constructed types.Object exercises
// exactly the same code path a real packages.Load-resolved reference would,
// without needing a real external module (including, for the lookalike-path
// case, a package whose path is textually similar to but not owned by this
// module — something a real go.mod-based fixture could only provide via an
// actual second module).
func newFakePackage(path string) *types.Package {
	name := path
	if i := strings.LastIndexByte(path, '/'); i >= 0 {
		name = path[i+1:]
	}
	return types.NewPackage(path, name)
}

func newFakeConst(pkgPath, name string) types.Object {
	pkg := newFakePackage(pkgPath)
	return types.NewConst(token.NoPos, pkg, name, types.Typ[types.String], nil)
}

func newFakeFunc(pkgPath, name string) types.Object {
	pkg := newFakePackage(pkgPath)
	sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	return types.NewFunc(token.NoPos, pkg, name, sig)
}

func newFakeVar(pkgPath, name string) types.Object {
	pkg := newFakePackage(pkgPath)
	return types.NewVar(token.NoPos, pkg, name, types.Typ[types.String])
}

func newFakeField(pkgPath, name string) types.Object {
	pkg := newFakePackage(pkgPath)
	return types.NewField(token.NoPos, pkg, name, types.Typ[types.String], false)
}

func newFakeTypeName(pkgPath, name string) types.Object {
	pkg := newFakePackage(pkgPath)
	return types.NewTypeName(token.NoPos, pkg, name, nil)
}

// selUsing builds a minimal *ast.SelectorExpr plus a *types.Info whose
// Uses map resolves that expression's Sel identifier to obj — the smallest
// input isExternalProtocolVersionConstant needs, isolating the predicate
// from packages.Load/go list entirely for this table.
func selUsing(obj types.Object) (*ast.SelectorExpr, *types.Info) {
	ident := &ast.Ident{Name: obj.Name()}
	sel := &ast.SelectorExpr{Sel: ident}
	info := &types.Info{Uses: map[*ast.Ident]types.Object{ident: obj}}
	return sel, info
}

// TestIsExternalProtocolVersionConstantMatrix is the false-positive half of
// the predicate — the reason it is restricted to *types.Const rather than a
// negative "external and not a struct field" filter. Every row constructs a
// types.Object of a specific kind/package and asserts isExternalProtocolVersionConstant's
// verdict.
func TestIsExternalProtocolVersionConstantMatrix(t *testing.T) {
	const modernSDKPath = "github.com/mark3labs/mcp-go/mcp"
	const lookalikeModulePath = modulePathPrefix + "-fork"
	const thisModulePath = modulePathPrefix + "/internal/mcp"
	const unrelatedSDKPath = "example.com/sdk"

	cases := []struct {
		name string
		obj  types.Object
		want bool
	}{
		{
			name: "external const mcp.LATEST_PROTOCOL_VERSION",
			obj:  newFakeConst(modernSDKPath, "LATEST_PROTOCOL_VERSION"),
			want: true,
		},
		{
			name: "external func call somepkg.ProtocolVersion()",
			obj:  newFakeFunc(unrelatedSDKPath, "ProtocolVersion"),
			want: false,
		},
		{
			name: "external struct field read/write req.Params.ProtocolVersion",
			obj:  newFakeField(modernSDKPath, "ProtocolVersion"),
			want: false,
		},
		{
			name: "external type name somepkg.ProtocolVersion used in a declaration",
			obj:  newFakeTypeName(unrelatedSDKPath, "ProtocolVersion"),
			want: false,
		},
		{
			// Allowed with the reason recorded here: package-level vars are
			// out of scope until a real case forces them in (see the
			// predicate's doc comment) — this row is that documentation,
			// enforced.
			name: "external package-level var somepkg.ProtocolVersion",
			obj:  newFakeVar(unrelatedSDKPath, "ProtocolVersion"),
			want: false,
		},
		{
			name: "repo-owned const mcp.ProtocolVersion (this module)",
			obj:  newFakeConst(thisModulePath, "ProtocolVersion"),
			want: false,
		},
		{
			// The boundary-aware ownership check must not admit this: the
			// lookalike path shares modulePathPrefix as a raw string
			// prefix but is a different module and must be treated as
			// external.
			name: "lookalike module path const (github.com/seanb4t/codegraph-go-fork)",
			obj:  newFakeConst(lookalikeModulePath, "ProtocolVersion"),
			want: true,
		},
		{
			// Known heuristic gap, recorded rather than silently omitted:
			// the (?i)protocol.?version pattern cannot catch a
			// differently-spelled SDK constant. This is exactly the
			// spelling internal/cli/archtest's import-path list exists to
			// still catch.
			name: "external const spelled LatestVersion (no \"protocol\" in the name — known heuristic gap)",
			obj:  newFakeConst(unrelatedSDKPath, "LatestVersion"),
			want: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sel, info := selUsing(c.obj)
			got := isExternalProtocolVersionConstant(sel, info)
			if got != c.want {
				t.Fatalf("isExternalProtocolVersionConstant(%s.%s) = %v, want %v", c.obj.Pkg().Path(), c.obj.Name(), got, c.want)
			}
		})
	}
}

// TestProtocolVersionGuardHelpersFailLoudly proves the two pure predicate
// helpers this file introduces return a documented false — never panic —
// on malformed or incomplete input, per T-03-03's repudiation concern:
// a guard whose helpers can panic on a legitimate edge case (an identifier
// resolution gap, a nil TypesInfo from an ill-typed package) would itself
// be a source of false negatives if the panic were ever recovered upstream.
func TestProtocolVersionGuardHelpersFailLoudly(t *testing.T) {
	t.Run("isExternalProtocolVersionConstant with nil info", func(t *testing.T) {
		sel := &ast.SelectorExpr{Sel: &ast.Ident{Name: "ProtocolVersion"}}
		if got := isExternalProtocolVersionConstant(sel, nil); got {
			t.Fatalf("isExternalProtocolVersionConstant(sel, nil) = true, want false")
		}
	})

	t.Run("isExternalProtocolVersionConstant with sel.Sel absent from info.Uses", func(t *testing.T) {
		sel := &ast.SelectorExpr{Sel: &ast.Ident{Name: "ProtocolVersion"}}
		info := &types.Info{Uses: map[*ast.Ident]types.Object{}}
		if got := isExternalProtocolVersionConstant(sel, info); got {
			t.Fatalf("isExternalProtocolVersionConstant with an unresolved identifier = true, want false")
		}
	})

	t.Run("isExternalProtocolVersionConstant with nil sel", func(t *testing.T) {
		if got := isExternalProtocolVersionConstant(nil, &types.Info{}); got {
			t.Fatalf("isExternalProtocolVersionConstant(nil, ...) = true, want false")
		}
	})

	t.Run("isModuleOwnedPath with empty string", func(t *testing.T) {
		if isModuleOwnedPath("") {
			t.Fatalf("isModuleOwnedPath(\"\") = true, want false")
		}
	})

	t.Run("isModuleOwnedPath with exact module path (no trailing slash)", func(t *testing.T) {
		if !isModuleOwnedPath(modulePathPrefix) {
			t.Fatalf("isModuleOwnedPath(%q) = false, want true", modulePathPrefix)
		}
	})

	t.Run("isModuleOwnedPath rejects a lookalike prefix", func(t *testing.T) {
		if isModuleOwnedPath(modulePathPrefix + "-fork") {
			t.Fatalf("isModuleOwnedPath(%q) = true, want false (lookalike module path must be treated as external)", modulePathPrefix+"-fork")
		}
	})
}
