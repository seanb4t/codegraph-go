// mcp_sdk_selftest_test.go is SDK-02's non-vacuity proof (T-03-03): it
// plants a real, syntactically-used direct import of the forbidden MCP SDK
// package into internal/cli/serve.go via packages.Config.Overlay (no file
// on disk is ever touched) and asserts the same predicate
// TestInternalCLIImportsNoMCPSDK uses reports a violation. Modelled on
// internal/graphstore/archtest/stdout_closure_selftest_test.go's overlay
// technique: resolve the target file's real on-disk path via packages.Load
// itself, insert the violation immediately after the package clause, and
// preserve every original declaration untouched so the rest of the file
// still type-checks during the overlaid load.
//
// Phase 2 (02-04, SDK-03) re-points the planted import from
// mark3labs/mcp-go/server to go-sdk's own
// github.com/modelcontextprotocol/go-sdk/mcp, once mark3labs leaves go.mod
// entirely — the mark3labs overlay would otherwise stop resolving for an
// unrelated reason (the import itself no longer type-checks anywhere in the
// module), proving nothing about the guard. This closes a real gap
// 02-PATTERNS.md flagged: before this change, the self-test only ever
// proved forbiddenMCPSDKPrefixes could catch the SDK being REMOVED
// (mark3labs, forward-declared but never the one actually in the tree by
// the time this self-test's sibling ran); it never proved the list catches
// the SDK actually being ADOPTED.
package archtest

import (
	"os"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

// serveGoPath resolves the real, on-disk absolute path of
// internal/cli/serve.go via packages.Load itself (rather than constructing
// the path by hand), so the overlay key below is guaranteed to match the
// exact path packages.Load uses internally when it re-resolves the package
// in TestInternalCLIImportsNoMCPSDK_PlantedImportIsError.
func serveGoPath(t *testing.T) string {
	t.Helper()

	cfg := &packages.Config{Mode: packages.NeedFiles | packages.NeedCompiledGoFiles}
	pkgs, err := packages.Load(cfg, internalCLIPkgPath)
	if err != nil {
		t.Fatalf("resolving %s package files: %v", internalCLIPkgPath, err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("packages.Load resolved %d packages for %s, want 1", len(pkgs), internalCLIPkgPath)
	}
	for _, f := range pkgs[0].GoFiles {
		if strings.HasSuffix(f, "/serve.go") {
			return f
		}
	}
	t.Fatalf("internal/cli/serve.go not found among resolved files: %v", pkgs[0].GoFiles)
	return ""
}

// TestInternalCLIImportsNoMCPSDK_PlantedImportIsError plants BOTH a direct
// import of github.com/modelcontextprotocol/go-sdk/mcp AND a package-level
// declaration that references it (var mcpSDKConfinementSelfTestProbe =
// sdkmcp.NewServer) into internal/cli/serve.go, in-memory only, and asserts
// the direct-import check flags it. A bare unused import fails the load
// for an unrelated reason — Go rejects unused imports at type-check time —
// and a self-test that "passes" because compilation broke would prove
// nothing about this guard; the referencing declaration (a genuine,
// exported, non-invoked function value from the package actually in the
// tree) is what makes this a genuine planted-defect proof.
func TestInternalCLIImportsNoMCPSDK_PlantedImportIsError(t *testing.T) {
	path := serveGoPath(t)

	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}

	const marker = "package cli\n"
	content := string(original)
	idx := strings.Index(content, marker)
	if idx == -1 {
		t.Fatalf("%s: expected to find %q as its package clause — file layout changed, update this fixture", path, marker)
	}
	insertAt := idx + len(marker)

	violated := content[:insertAt] +
		"\nimport sdkmcp \"github.com/modelcontextprotocol/go-sdk/mcp\"\n" +
		content[insertAt:] +
		"\nvar mcpSDKConfinementSelfTestProbe = sdkmcp.NewServer\n"

	cfg := &packages.Config{
		Mode:    packages.NeedName | packages.NeedImports | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
		Tests:   false,
		Overlay: map[string][]byte{path: []byte(violated)},
	}
	pkgs, err := packages.Load(cfg, internalCLIPkgPath)
	if err != nil {
		t.Fatalf("packages.Load: %v", err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("packages.Load resolved %d packages for %s, want 1", len(pkgs), internalCLIPkgPath)
	}

	pkg := pkgs[0]
	if len(pkg.TypeErrors) > 0 {
		t.Fatalf("overlaid %s has %d type error(s) — the overlay broke compilation for an unrelated reason, "+
			"proving nothing about the guard: %v", internalCLIPkgPath, len(pkg.TypeErrors), pkg.TypeErrors)
	}

	found := false
	for imp := range pkg.Imports {
		if _, forbidden := hasForbiddenPrefix(imp); forbidden {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("planted a direct import of github.com/modelcontextprotocol/go-sdk/mcp into %s (an "+
			"in-memory overlay only — the real file on disk is untouched) but it was not detected as a "+
			"forbidden import; resolved imports: %v", path, pkg.Imports)
	}
}
