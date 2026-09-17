// Package archtest enforces TUI-01: none of the six serve-reachable
// packages (the MCP transport itself, the read-only query engine it
// delegates to, the storage layer it reads through, and the background
// daemon/watch/indexer machinery a live session can trigger) may ever
// import any package under either charm vanity root — charm.land/... or
// github.com/charmbracelet/... — the agent/MCP surface must stay ANSI-free
// forever (D-01, D-11; widened by 04-02/D-15/GRD-13).
//
// Forbidden paths are matched by PREFIX on the two charm vanity roots, not
// by an exact-match literal list: some charm-family packages
// (colorprofile, x/ansi) are hosted at github.com/charmbracelet/... rather
// than charm.land/..., so an exact list or a charm.land/-only prefix would
// silently miss them (04-RESEARCH.md Pitfall 4). charm.land/lipgloss (no
// /v2 suffix) resolving to a real, different, WRONG module (the old v0/v1
// API line, also re-hosted at the same vanity domain, RESEARCH Finding 1)
// is unaffected by this widening — the prefix still only matches actual
// charm.land/... or github.com/charmbracelet/... import paths, and the
// self-defeat probe below stays pinned to the exact /v2-suffixed path.
//
// Build order (D-12): this file lands FIRST, before internal/cli/present
// exists or any Charm dependency enters go.mod. Until a real charm
// importer exists, the self-defeat guard below fails closed — proving
// this test can actually detect a violation rather than being vacuously
// green because nothing imports charm at all yet.
package archtest

import (
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

// guardedPackages are the six serve-reachable packages TUI-01/D-11 locks
// down: every package that can execute during a `serve --mcp` session
// (the MCP transport itself, the read-only query engine it delegates to,
// the storage layer it reads through, and the background
// daemon/watch/indexer machinery a live session can trigger via the
// startup reconcile or a debounced sync) must never import charm.
//
// Copied verbatim from internal/graphstore/archtest/stdout_confinement_test.go
// per D-11/06-PATTERNS.md — the same six-package set defends the whole
// agent path against ANSI/charm reachability, not only query/mcp.
var guardedPackages = []string{
	"github.com/seanb4t/codegraph-go/internal/mcp",
	"github.com/seanb4t/codegraph-go/internal/graphstore",
	"github.com/seanb4t/codegraph-go/internal/daemon",
	"github.com/seanb4t/codegraph-go/internal/watch",
	"github.com/seanb4t/codegraph-go/internal/indexer",
	"github.com/seanb4t/codegraph-go/internal/query",
}

// forbiddenImportPathPrefixes are the two charm vanity import-path roots
// (D-15/GRD-13, widened from the original three-literal exact-match list).
// Both roots are forbidden by PREFIX, not by an exact-match list, because
// colorprofile and x/ansi live under github.com/charmbracelet/... while
// lipgloss/bubbletea/bubbles live under charm.land/... — a prefix on
// charm.land/ alone would miss the former (04-RESEARCH.md Pitfall 4).
var forbiddenImportPathPrefixes = []string{
	"charm.land/",
	"github.com/charmbracelet/",
}

// charmImporterProbePath is the specific forbidden path the self-defeat
// guard (D-12) looks for to prove a real charm importer exists somewhere
// in the module. lipgloss is the one Phase 6 actually adds (D-13 — no
// bubbletea/bubbles in this phase).
const charmImporterProbePath = "charm.land/lipgloss/v2"

const modulePathPrefix = "github.com/seanb4t/codegraph-go/"

// excludedInternalPackagePrefixes are module-internal packages that must
// NEVER be pulled into the closure even if some guardedPackage comes to
// (directly or transitively) import them. internal/cli is excluded here
// for the same reason internal/graphstore/archtest's D-06b guard excludes
// it: it is never reachable from `serve --mcp` itself, and it is also the
// package tree (internal/cli/present) that legitimately owns the sole
// charm import this phase adds — it must not be treated as part of the
// serve-reachable closure.
var excludedInternalPackagePrefixes = []string{
	"github.com/seanb4t/codegraph-go/internal/cli",
}

// isModuleInternalPackage reports whether pkgPath belongs to this module
// (so it's a candidate for closure scanning) and is not internal/cli or a
// subpackage of it. Third-party and stdlib imports always return false —
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
// packages.NeedDeps, so dependency packages get full Imports metadata,
// not just name/ID) and returns the full transitive, module-internal
// import closure reachable from them: the six roots themselves plus every
// package they import, directly or indirectly, that belongs to this
// module and is not internal/cli.
func closeOverServeReachableImports(t *testing.T) map[string]*packages.Package {
	t.Helper()

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedImports | packages.NeedDeps,
		// Tests: false (deliberate — mirrors stdout_confinement_test.go):
		// this guard is scoped to the six NAMED roots' real reachability
		// from the `serve --mcp` binary itself, not to what a _test.go
		// file happens to import during `go test`.
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

	return reachable
}

// assertCharmImporterExists is the D-12 self-defeat sanity guard: without
// it, if internal/cli/present ever loses its only charm import (e.g. a
// future refactor deletes the pretty renderer entirely), the forbidden-path
// check above becomes vacuously true for the wrong reason — nothing in
// the module imports charm at all, so of course the guarded set doesn't
// either. This whole-module scan (Tests: true, so a bypass hidden in a
// _test.go file is not invisible) fails loudly instead, exactly mirroring
// internal/graphstore/archtest/import_graph_test.go's
// foundGraphstoreImporter check (renamed foundCharmImporter here). This is
// what makes the test RED today (no charm dep yet) and honest tomorrow.
func assertCharmImporterExists(t *testing.T) {
	t.Helper()

	cfg := &packages.Config{
		Mode:  packages.NeedImports | packages.NeedName | packages.NeedDeps,
		Tests: true,
	}
	pkgs, err := packages.Load(cfg, modulePathPrefix+"...")
	if err != nil {
		t.Fatalf("packages.Load: %v", err)
	}
	if len(pkgs) == 0 {
		t.Fatal("packages.Load returned no packages — the module import graph did not resolve")
	}

	foundCharmImporter := false
	for _, pkg := range pkgs {
		if _, imports := pkg.Imports[charmImporterProbePath]; imports {
			foundCharmImporter = true
			break
		}
	}
	if !foundCharmImporter {
		t.Fatalf("no package in the module was found importing %s — this test cannot verify enforcement; check that internal/cli/present still imports %s and that packages.Load resolved it", charmImporterProbePath, charmImporterProbePath)
	}
}

// TestNoCharmInServeReachablePackages loads the full serve-reachable
// import closure of the six guardedPackages via go/packages (D-10: never
// regex/source scanning — that misses aliased imports, build-tag-gated
// files, and test variants) and fails if any package in that closure
// imports a path under either forbidden charm vanity root (D-15/GRD-13),
// reporting EVERY hit rather than stopping at the first. It also runs the
// D-12 self-defeat guard, so this test cannot go green for the wrong
// reason (nothing imports charm at all).
func TestNoCharmInServeReachablePackages(t *testing.T) {
	reachable := closeOverServeReachableImports(t)

	for _, pkg := range reachable {
		for path := range pkg.Imports {
			for _, prefix := range forbiddenImportPathPrefixes {
				if strings.HasPrefix(path, prefix) {
					t.Errorf("package %s imports %s — charm styling must never reach the serve-reachable closure (TUI-01); charm-family usage must be confined to internal/cli/present", pkg.PkgPath, path)
				}
			}
		}
	}

	assertCharmImporterExists(t)
}
