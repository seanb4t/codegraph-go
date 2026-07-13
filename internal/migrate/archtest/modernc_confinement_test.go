// Package archtest enforces the read-only-migration-reader confinement
// boundary from 07-CONTEXT.md / PROJECT.md ("migration reader only,
// isolated to a one-shot code path"): modernc.org/sqlite must never be
// importable from internal/graphstore, internal/indexer, internal/query,
// internal/cli, or any other runtime hot path — only internal/migrate (and
// its own subpackages, e.g. migratetest) may import it. This mirrors
// internal/graphstore/archtest's TestNoPackageBypassesGraphStore, which
// enforces the analogous D-04a pebble-confinement boundary the same way:
// via go/packages import-graph inspection, not directory convention alone.
package archtest

import (
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

const (
	modernSQLiteImportPath = "modernc.org/sqlite"
	allowedImporterPrefix  = "github.com/seanb4t/codegraph-go/internal/migrate"
)

// TestModerncSQLiteConfinedToMigrate loads this module's full import graph
// via go/packages (NOT regex/string-matching over source — regex misses
// aliased imports, build-tag-gated files, and test variants) and asserts
// that every package importing modernc.org/sqlite has an import path under
// internal/migrate's prefix. internal/migrate itself, and its own
// subpackages (e.g. internal/migrate/migratetest, the one legitimate
// importer as of this plan), are the only legal importers.
func TestModerncSQLiteConfinedToMigrate(t *testing.T) {
	cfg := &packages.Config{
		Mode: packages.NeedImports | packages.NeedName | packages.NeedDeps,
		// Tests: true is required so an import statement that appears only
		// inside a _test.go file (including a hypothetical bypass package
		// outside internal/migrate that imports modernc.org/sqlite solely
		// from its test file) is not invisible to this check. Without it,
		// go/packages loads only each package's non-test compilation unit.
		Tests: true,
	}
	pkgs, err := packages.Load(cfg, "github.com/seanb4t/codegraph-go/...")
	if err != nil {
		t.Fatalf("packages.Load: %v", err)
	}
	if len(pkgs) == 0 {
		t.Fatal("packages.Load returned no packages — the module import graph did not resolve")
	}

	foundMigrateImporter := false
	for _, pkg := range pkgs {
		if _, imports := pkg.Imports[modernSQLiteImportPath]; !imports {
			continue
		}
		if !isAllowedImporter(pkg.PkgPath) {
			t.Errorf("package %s imports %s directly — only %s (and its subpackages) may; the migration reader must stay isolated to a one-shot code path", pkg.PkgPath, modernSQLiteImportPath, allowedImporterPrefix)
			continue
		}
		foundMigrateImporter = true
	}

	// Sanity check that this test can actually detect a real importer: if
	// internal/migrate itself no longer imports modernc.org/sqlite (e.g.
	// after a refactor), the check above is vacuously true for the wrong
	// reason — modernc.org/sqlite might not appear in the loaded graph at
	// all, silently disabling this test's ability to ever fail.
	if !foundMigrateImporter {
		t.Fatal("no package under internal/migrate was found importing modernc.org/sqlite — this test cannot verify enforcement; check that migratetest/fixture.go still imports modernc.org/sqlite and that packages.Load resolved it")
	}
}

func isAllowedImporter(pkgPath string) bool {
	base := stripTestVariant(pkgPath)
	if base == allowedImporterPrefix {
		return true
	}
	return len(base) > len(allowedImporterPrefix) && base[:len(allowedImporterPrefix)+1] == allowedImporterPrefix+"/"
}

// stripTestVariant normalizes the additional PkgPath forms Tests: true
// introduces back to the underlying import path, so the allowed-prefix
// check applies uniformly regardless of which variant loaded the import:
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
