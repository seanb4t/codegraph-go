// stdout_closure_selftest_test.go is the CR-01 regression lock: it proves
// the transitive-import closure walk added to
// TestNoStdoutNoiseInServeReachablePackages (stdout_confinement_test.go)
// actually reaches into a guarded package's DEPENDENCIES, not just the six
// named guardedPackages themselves. Before CR-01's fix, an identical
// violation planted in internal/schema (a package none of the six
// guardedPackages name directly, but which every one of them transitively
// imports) left the guard green — this is the exact reproduction the CR-01
// finding documented, turned into a permanent regression test (D-08
// mutation-proof discipline applied to the closure itself, not just the
// detection predicates stdout_detection_selftest_test.go already covers).
package archtest

import (
	"os"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

// schemaMetaGoPath resolves the real, on-disk absolute path of
// internal/schema/meta.go via packages.Load itself (rather than
// constructing the path by hand), so the overlay key below is guaranteed
// to match the exact path packages.Load uses internally when it re-resolves
// the module in TestStdoutGuardCatchesViolationsInTransitiveDependency.
func schemaMetaGoPath(t *testing.T) string {
	t.Helper()
	cfg := &packages.Config{Mode: packages.NeedFiles | packages.NeedCompiledGoFiles}
	pkgs, err := packages.Load(cfg, "github.com/seanb4t/codegraph-go/internal/schema")
	if err != nil {
		t.Fatalf("resolving internal/schema package files: %v", err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("packages.Load resolved %d packages for internal/schema, want 1", len(pkgs))
	}
	for _, f := range pkgs[0].GoFiles {
		if strings.HasSuffix(f, "/meta.go") || f == "meta.go" {
			return f
		}
	}
	t.Fatalf("internal/schema/meta.go not found among resolved files: %v", pkgs[0].GoFiles)
	return ""
}

// TestStdoutGuardCatchesViolationsInTransitiveDependency plants a real
// fmt.Println call inside internal/schema/meta.go via packages.Config.Overlay
// (no file on disk is ever touched — Overlay is purely in-memory content
// substitution for this one packages.Load call) and asserts the closure
// scan flags it. internal/schema is not one of the six guardedPackages, but
// every one of them transitively imports it (mcp, graphstore, daemon,
// watch, indexer, and query all depend on it) — exactly the class of
// violation CR-01 found invisible to the pre-fix guard.
func TestStdoutGuardCatchesViolationsInTransitiveDependency(t *testing.T) {
	metaGoPath := schemaMetaGoPath(t)

	original, err := os.ReadFile(metaGoPath)
	if err != nil {
		t.Fatalf("reading %s: %v", metaGoPath, err)
	}

	const marker = "package schema\n"
	content := string(original)
	idx := strings.Index(content, marker)
	if idx == -1 {
		t.Fatalf("%s: expected to find %q as its package clause — file layout changed, update this fixture", metaGoPath, marker)
	}
	insertAt := idx + len(marker)

	// Insert the import right after the package clause (required import
	// placement) and preserve every original declaration untouched, so
	// every other package that depends on schema.SchemaVersion / NewMeta /
	// IsCurrentSchemaVersion still type-checks correctly during this
	// closure load — only a new, additional function is introduced.
	violated := content[:insertAt] +
		"\nimport \"fmt\"\n" +
		content[insertAt:] +
		"\nfunc stdoutGuardClosureProbe() {\n\tfmt.Println(\"stdout guard closure probe — must be caught by CR-01's transitive-dependency scan\")\n}\n"

	reachable := closeOverServeReachableImports(t, map[string][]byte{metaGoPath: []byte(violated)})

	const schemaPkgPath = "github.com/seanb4t/codegraph-go/internal/schema"
	if _, ok := reachable[schemaPkgPath]; !ok {
		t.Fatalf("%s was not present in the closure — cannot prove the overlaid violation was even scanned", schemaPkgPath)
	}

	violations := scanForStdoutViolations(reachable)
	found := false
	for _, msg := range violations {
		if strings.HasPrefix(msg, schemaPkgPath+": calls a bare fmt.Print") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("planted an fmt.Println violation in internal/schema (a dependency of every guardedPackage, not a guardedPackage itself) but the closure scan did not flag it — got violations: %v", violations)
	}
}
