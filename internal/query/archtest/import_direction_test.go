// Package archtest enforces threat T-01-18: internal/query is a read-side
// package (the Engine's query surface, consumed by internal/cli,
// internal/mcp, and internal/uiserver) and must not depend on the wire layer
// it is meant to be consumed by, nor on the indexer's pipeline/discovery
// root whose OUTPUT it is meant to read only via internal/graphstore.
//
// Two rules, two scopes:
//
//   - The wire-layer rule (forbiddenWireLayerImports) is checked over EVERY
//     loaded internal/query-rooted package, including test variants
//     (Tests: true) — a wire dependency smuggled into a _test.go file is
//     exactly the case Tests: true exists to catch, and nothing legitimately
//     reaches these packages from internal/query in any variant.
//   - The indexer-root rule (forbiddenProductionOnlyImports) is checked over
//     the PRODUCTION compilation unit only. internal/query/engine_test.go is
//     an in-package (package query) test file that imports the
//     internal/indexer root today, on purpose, to run the real indexer and
//     build test fixtures. Scoping this rule to production code is a
//     deliberate boundary, not a weakening of the forbidden set: the allowed
//     leaves internal/indexer/goextract and internal/indexer/nodeid
//     (allowedIndexerLeaves) remain the only internal/indexer imports
//     permitted anywhere, and the internal/indexer ROOT stays forbidden in
//     every production package under internal/query.
//
// Forward consequence for Phase 10 (HLT-05): the discovery-exclusion helper
// planned there writes reasons from inside the indexer's discovery path;
// internal/query reads them back through internal/graphstore only.
// internal/query may not import that helper — doing so would resolve the
// forbidden internal/indexer root transitively and fail this test.
package archtest

import (
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

const (
	queryPrefix = "github.com/seanb4t/codegraph-go/internal/query"

	graphstoreImportPath = "github.com/seanb4t/codegraph-go/internal/graphstore"
	goextractImportPath  = "github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	nodeidImportPath     = "github.com/seanb4t/codegraph-go/internal/indexer/nodeid"
	parserImportPath     = "github.com/seanb4t/codegraph-go/internal/parser"
)

// forbiddenWireLayerImports is the D-02 union: the roadmap's wire-layer list
// plus the T-01-18 todo's list. Checked over every loaded internal/query
// -rooted package's RESOLVED TRANSITIVE dependency set (not merely direct
// imports), including test variants — nothing legitimately reaches these
// from internal/query in any variant, and a wire import smuggled into a
// _test.go file is precisely the case Tests: true exists to catch.
var forbiddenWireLayerImports = []string{
	"github.com/seanb4t/codegraph-go/internal/uiserver",
	"github.com/seanb4t/codegraph-go/internal/mcp",
	"github.com/seanb4t/codegraph-go/internal/uiproto",
	"connectrpc.com/connect",
}

// forbiddenProductionOnlyImports is D-01's single entry: the internal/indexer
// ROOT package (the pipeline/discovery path). Matching against a package's
// resolved dependency set is EXACT PATH, never prefix — allowedIndexerLeaves
// share the "internal/indexer" string prefix but are never matched by this
// slice, because the set built by transitiveDeps stores whole import paths
// and membership is checked by exact key lookup.
var forbiddenProductionOnlyImports = []string{
	"github.com/seanb4t/codegraph-go/internal/indexer",
}

// allowedIndexerLeaves names the two internal/indexer subpackages
// internal/query legitimately resolves today: goextract for edge-kind
// constants, nodeid for node-ID hashing. Named here so the allow-list is a
// readable, referenced fact rather than an implicit consequence of the
// exact-path matching in forbiddenProductionOnlyImports.
var allowedIndexerLeaves = []string{
	goextractImportPath,
	nodeidImportPath,
}

// TestQueryImportsNoWireLayerOrIndexerRoot loads the module's import graph
// rooted at internal/query via go/packages (NOT regex/string-matching over
// source — regex misses aliased imports, build-tag-gated files, and test
// variants) and asserts internal/query's RESOLVED TRANSITIVE dependency set
// never contains a forbidden wire-layer path (checked over every loaded
// variant) or the internal/indexer root (checked over the production
// compilation unit only; see the package doc for why).
func TestQueryImportsNoWireLayerOrIndexerRoot(t *testing.T) {
	cfg := &packages.Config{
		Mode: packages.NeedImports | packages.NeedName | packages.NeedDeps,
		// Tests: true is required so an import statement that appears only
		// inside a _test.go file is not invisible to this check. Without it,
		// go/packages loads only each package's non-test compilation unit.
		Tests: true,
	}
	pkgs, err := packages.Load(cfg, queryPrefix+"/...")
	if err != nil {
		t.Fatalf("packages.Load: %v", err)
	}
	if len(pkgs) == 0 {
		t.Fatal("packages.Load returned no packages — the module import graph did not resolve")
	}
	// packages.Load reports a per-package failure (an unresolvable import, a
	// build error reachable from internal/query) in pkg.Errors, NOT in its
	// top-level error return. A broken subtree is then absent from, or has
	// an incomplete Imports map in, the returned graph, so every forbidden-
	// import assertion below looks for members that were never added and
	// passes vacuously. PrintErrors both surfaces each error on stderr and
	// returns the count; a non-zero count means this test cannot verify
	// anything about the affected package(s) and must refuse (CR-01).
	if n := packages.PrintErrors(pkgs); n > 0 {
		t.Fatalf("packages.Load reported %d package error(s) — the import graph did not fully resolve, so this test cannot verify anything about the affected package(s); see the errors above", n)
	}

	foundProductionQuery := false
	var productionQueryPkg *packages.Package

	for _, pkg := range pkgs {
		normalized := stripTestVariant(pkg.PkgPath)
		if normalized != queryPrefix && !strings.HasPrefix(normalized, queryPrefix+"/") {
			continue
		}
		// isTestVariant must be computed from pkg.ID, not pkg.PkgPath: this
		// x/tools version gives the "package X compiled with its own
		// internal _test.go files" variant the SAME PkgPath as the true
		// production package (both report PkgPath == "internal/query"),
		// and only pkg.ID carries the distinguishing " [X.test]" suffix
		// that marks it as a test-augmented compilation unit. Empirically
		// confirmed against this module: the internal/query "for test"
		// variant's PkgPath equals the production PkgPath exactly, while
		// its ID is "internal/query [internal/query.test]".
		isTestVariant := pkg.ID != stripTestVariant(pkg.ID)

		deps := transitiveDeps(pkg)

		for _, forbidden := range forbiddenWireLayerImports {
			if deps[forbidden] {
				t.Errorf("package %s resolves forbidden wire-layer dependency %s in its transitive dependency set (the dependency may be indirect) — internal/query must not depend on the wire layer it is consumed by", pkg.PkgPath, forbidden)
			}
		}

		if isTestVariant {
			// The indexer-root rule is production-scoped: see the package
			// doc for why internal/query/engine_test.go is excluded here.
			continue
		}

		for _, forbidden := range forbiddenProductionOnlyImports {
			if deps[forbidden] {
				t.Errorf("production package %s resolves the forbidden internal/indexer root %s in its transitive dependency set (the dependency may be indirect) — only internal/indexer/goextract and internal/indexer/nodeid are allowed leaves; internal/query/engine_test.go is the one legitimate in-package importer of the root, and this rule is scoped to exclude only that test file, not to permit the root from production code", pkg.PkgPath, forbidden)
			}
		}

		if normalized == queryPrefix {
			foundProductionQuery = true
			productionQueryPkg = pkg
		}
	}

	if !foundProductionQuery {
		t.Fatal("the load pattern resolved no production internal/query package — this test verified nothing; check queryPrefix and the load pattern")
	}

	productionDeps := transitiveDeps(productionQueryPkg)

	if !productionDeps[graphstoreImportPath] {
		t.Fatalf("production internal/query's resolved transitive dependency set does not contain %s — this test can no longer verify enforcement; check that internal/query still depends on internal/graphstore", graphstoreImportPath)
	}
	// The allow-list is a LIVE assertion, not documentation: every allowed
	// indexer leaf must actually be in the production dependency set. If a
	// leaf ever drops out, the exact-path root check above is vacuous for
	// that leaf (D-01/D-03) and this is the assertion that says so.
	for _, leaf := range allowedIndexerLeaves {
		if !productionDeps[leaf] {
			t.Fatalf("production internal/query's resolved transitive dependency set does not contain the allowed indexer leaf %s — this test can no longer verify enforcement; check that internal/query still resolves it", leaf)
		}
	}

	if !productionDeps[parserImportPath] {
		t.Fatalf("%s is not in production internal/query's resolved transitive dependency set — the walk may no longer be resolving beyond one hop; choose another transitive-only witness rather than deleting this assertion", parserImportPath)
	}
	if _, direct := productionQueryPkg.Imports[parserImportPath]; direct {
		t.Fatalf("%s is now a DIRECT import of production internal/query — it no longer proves the transitive walk resolves beyond one hop; choose another transitive-only witness rather than deleting this assertion", parserImportPath)
	}

	t.Logf("loaded %d packages; production internal/query resolved %d transitive dependencies", len(pkgs), len(productionDeps))
}

// transitiveDeps walks pkg.Imports recursively — NOT a single-hop check of
// pkg.Imports — and returns the set of import paths reached, normalized via
// stripTestVariant. This walk is what D-02 requires: a violation smuggled
// through an intermediary package must be caught, not just a direct import.
// A visited set keyed by PkgPath guards against a cyclic or diamond import
// graph so the walk terminates.
func transitiveDeps(pkg *packages.Package) map[string]bool {
	deps := make(map[string]bool)
	visited := make(map[string]bool)

	var walk func(p *packages.Package)
	walk = func(p *packages.Package) {
		if visited[p.PkgPath] {
			return
		}
		visited[p.PkgPath] = true
		for path, dep := range p.Imports {
			deps[stripTestVariant(path)] = true
			walk(dep)
		}
	}
	walk(pkg)
	return deps
}

// stripTestVariant normalizes the additional PkgPath forms Tests: true
// introduces back to the underlying import path, so prefix/equality checks
// apply uniformly regardless of which variant loaded the import:
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
