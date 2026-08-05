// protocol_version_selftest_test.go is VRFY-02's non-vacuity proof (T-03-03):
// it plants a real, syntactically-used reference to the SDK's protocol-version
// constant via packages.Config.Overlay (no file on disk is ever touched) and
// asserts scanForProtocolVersionRefs actually flags it. Modelled directly on
// internal/graphstore/archtest/stdout_closure_selftest_test.go's overlay
// technique: resolve the target file's real on-disk path via packages.Load
// itself, insert the violation immediately after the package clause, and
// preserve every original declaration untouched so the rest of the module
// still type-checks during the overlaid load.
package archtest

import (
	"os"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

// protocolVersionGoPath resolves the real, on-disk absolute path of
// internal/mcp/protocol_version.go via packages.Load itself (rather than
// constructing the path by hand), so the overlay key below is guaranteed to
// match the exact path packages.Load uses internally when it re-resolves
// the module in TestProtocolVersionGuardCatchesOverlaidViolation.
func protocolVersionGoPath(t *testing.T) string {
	t.Helper()

	cfg := &packages.Config{Mode: packages.NeedFiles | packages.NeedCompiledGoFiles}
	pkgs, err := packages.Load(cfg, modulePathPrefix+"/internal/mcp")
	if err != nil {
		t.Fatalf("resolving internal/mcp package files: %v", err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("packages.Load resolved %d packages for internal/mcp, want 1", len(pkgs))
	}
	for _, f := range pkgs[0].GoFiles {
		if strings.HasSuffix(f, "/protocol_version.go") {
			return f
		}
	}
	t.Fatalf("internal/mcp/protocol_version.go not found among resolved files: %v", pkgs[0].GoFiles)
	return ""
}

// TestProtocolVersionGuardCatchesOverlaidViolation plants a package-level
// var in internal/mcp/protocol_version.go (via Overlay only — the real file
// on disk is never touched) that imports mark3labs/mcp-go/mcp and reads its
// LATEST_PROTOCOL_VERSION constant, then asserts scanForProtocolVersionRefs
// flags it. The overlay inserts a syntactically USED reference (a
// package-level var initializer), not a bare import — an unused import
// would fail the load for an unrelated reason (Go rejects unused imports at
// type-check time) and this self-test would "pass" because compilation
// broke, proving nothing about the guard itself.
func TestProtocolVersionGuardCatchesOverlaidViolation(t *testing.T) {
	path := protocolVersionGoPath(t)

	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}

	const marker = "package mcp\n"
	content := string(original)
	idx := strings.Index(content, marker)
	if idx == -1 {
		t.Fatalf("%s: expected to find %q as its package clause — file layout changed, update this fixture", path, marker)
	}
	insertAt := idx + len(marker)

	// Insert the import right after the package clause and preserve every
	// original declaration untouched, so ProtocolVersion itself and every
	// site that references it still type-check correctly during this
	// overlaid load.
	violated := content[:insertAt] +
		"\nimport \"github.com/mark3labs/mcp-go/mcp\"\n" +
		content[insertAt:] +
		"\nvar protocolVersionGuardSelfTestProbe = mcp.LATEST_PROTOCOL_VERSION\n"

	pkgs := loadWholeModule(t, map[string][]byte{path: []byte(violated)})

	const targetPkgPath = modulePathPrefix + "/internal/mcp"
	found := false
	for _, pkg := range pkgs {
		if stripTestVariant(pkg.PkgPath) != targetPkgPath {
			continue
		}
		found = true
		if len(pkg.TypeErrors) > 0 {
			t.Fatalf("overlaid %s has %d type error(s) — the overlay broke compilation for an unrelated "+
				"reason, proving nothing about the guard: %v", targetPkgPath, len(pkg.TypeErrors), pkg.TypeErrors)
		}
	}
	if !found {
		t.Fatalf("overlaid package %s was not present in the loaded set — cannot prove the injected "+
			"violation was even scanned", targetPkgPath)
	}

	violations := scanForProtocolVersionRefs(pkgs)
	matched := false
	for _, msg := range violations {
		if strings.Contains(msg, "VRFY-02") && strings.Contains(msg, "LATEST_PROTOCOL_VERSION") && strings.Contains(msg, targetPkgPath) {
			matched = true
			break
		}
	}
	if !matched {
		t.Fatalf("planted a reference to mcp.LATEST_PROTOCOL_VERSION in %s (an in-memory overlay only — "+
			"the real file on disk is untouched) but scanForProtocolVersionRefs did not flag it — got "+
			"violations: %v", targetPkgPath, violations)
	}
}
