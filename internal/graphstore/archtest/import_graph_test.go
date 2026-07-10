// Package archtest enforces D-04a: no package outside internal/graphstore
// (and its own subpackages) may import the embedded key-value engine
// directly. Go's internal/ convention alone does NOT stop a sibling
// package under internal/graphstore/... — or any other package in this
// module — from importing pebble/v2 directly; this test, not the
// directory convention, is what actually enforces the boundary
// (RESEARCH Pattern 5).
package archtest

import (
	"testing"

	"golang.org/x/tools/go/packages"
)

const (
	pebbleImportPath      = "github.com/cockroachdb/pebble/v2"
	allowedImporterPrefix = "github.com/seanb4t/codegraph-go/internal/graphstore"
)

// TestNoPackageBypassesGraphStore loads this module's full import graph via
// go/packages (NOT regex/string-matching over source — regex misses
// aliased imports, build-tag-gated files, and test variants) and asserts
// that every package importing pebble/v2 has an import path under
// internal/graphstore's prefix. internal/graphstore itself, and its own
// subpackages (e.g. this archtest package would fail this check if it ever
// imported pebble directly — it does not), are the only legal importers.
func TestNoPackageBypassesGraphStore(t *testing.T) {
	cfg := &packages.Config{
		Mode: packages.NeedImports | packages.NeedName | packages.NeedDeps,
	}
	pkgs, err := packages.Load(cfg, "github.com/seanb4t/codegraph-go/...")
	if err != nil {
		t.Fatalf("packages.Load: %v", err)
	}
	if len(pkgs) == 0 {
		t.Fatal("packages.Load returned no packages — the module import graph did not resolve")
	}

	foundGraphstoreImporter := false
	for _, pkg := range pkgs {
		if _, imports := pkg.Imports[pebbleImportPath]; !imports {
			continue
		}
		if !isAllowedImporter(pkg.PkgPath) {
			t.Errorf("package %s imports %s directly — only %s (and its subpackages) may; route through the GraphStore interface instead", pkg.PkgPath, pebbleImportPath, allowedImporterPrefix)
			continue
		}
		foundGraphstoreImporter = true
	}

	// Sanity check that this test can actually detect a real importer: if
	// internal/graphstore itself no longer imports pebble/v2 (e.g. after a
	// refactor), the check above is vacuously true for the wrong reason —
	// pebble/v2 might not appear in the loaded graph at all, silently
	// disabling this test's ability to ever fail.
	if !foundGraphstoreImporter {
		t.Fatal("no package under internal/graphstore was found importing pebble/v2 — this test cannot verify enforcement; check that pebble_store.go still imports pebble/v2 and that packages.Load resolved it")
	}
}

func isAllowedImporter(pkgPath string) bool {
	if pkgPath == allowedImporterPrefix {
		return true
	}
	return len(pkgPath) > len(allowedImporterPrefix) && pkgPath[:len(allowedImporterPrefix)+1] == allowedImporterPrefix+"/"
}
