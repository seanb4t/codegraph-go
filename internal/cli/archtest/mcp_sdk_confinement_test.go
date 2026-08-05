// Package archtest enforces SDK-02: internal/cli's own DIRECT production
// imports must never include an MCP SDK package. internal/cli bootstraps
// the stdio server entirely through the internal/mcp.Server seam
// (mcp.NewStdioServer) — serve.go itself names no SDK package, even though
// internal/cli legitimately REACHES the SDK transitively, through
// internal/mcp. The check below is scoped to direct imports only, which is
// exactly SDK-02's criterion; it is a sibling of
// internal/graphstore/archtest and internal/cli/present/archtest, but
// simpler than either — SDK-02 asks "does serve.go itself name an SDK
// import", not "is the SDK reachable" (it always is, via internal/mcp).
//
// This is a go/packages import-graph scan, never a text search over
// serve.go — a text search misses aliased imports, build-tag-gated files,
// and test variants (the established house style).
package archtest

import (
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

const modulePathPrefix = "github.com/seanb4t/codegraph-go"

// internalCLIPkgPath is the one package this guard scans (Tests: false —
// SDK-02 is about the production import list, not what a _test.go file in
// this package happens to import).
const internalCLIPkgPath = modulePathPrefix + "/internal/cli"

// forbiddenMCPSDKPrefixes are the import-path prefixes internal/cli must
// never directly import. github.com/modelcontextprotocol/go-sdk is
// deliberately forward-declared: Phase 2 replaces
// github.com/mark3labs/mcp-go with it, and a guard naming only today's
// dependency would go silently vacuous at exactly the moment — a Phase 2
// SDK swap — it matters most. Any future MCP SDK must be added here;
// assertMCPSDKImporterExists below is what makes a stale list fail loudly
// instead of passing.
var forbiddenMCPSDKPrefixes = []string{
	"github.com/mark3labs/mcp-go",
	"github.com/modelcontextprotocol/go-sdk",
}

// hasForbiddenPrefix reports whether importPath is exactly one of
// forbiddenMCPSDKPrefixes, or a subpackage of one (prefix+"/").
func hasForbiddenPrefix(importPath string) (string, bool) {
	for _, prefix := range forbiddenMCPSDKPrefixes {
		if importPath == prefix || strings.HasPrefix(importPath, prefix+"/") {
			return prefix, true
		}
	}
	return "", false
}

// assertMCPSDKImporterExists is the self-defeat sanity guard (mirroring
// internal/cli/present/archtest's assertCharmImporterExists): without it,
// if the SDK ever left the tree entirely, or forbiddenMCPSDKPrefixes went
// stale relative to whatever replaced it, TestInternalCLIImportsNoMCPSDK's
// direct-import check would pass vacuously — nothing in the module imports
// any forbidden prefix at all, so of course internal/cli doesn't either.
// This whole-module scan (Tests: true, so a bypass hidden in a _test.go
// file is not invisible) fails loudly instead.
func assertMCPSDKImporterExists(t *testing.T) {
	t.Helper()

	cfg := &packages.Config{
		Mode:  packages.NeedImports | packages.NeedName,
		Tests: true,
	}
	pkgs, err := packages.Load(cfg, modulePathPrefix+"/...")
	if err != nil {
		t.Fatalf("packages.Load: %v", err)
	}
	if len(pkgs) == 0 {
		t.Fatal("packages.Load returned no packages — the module import graph did not resolve")
	}

	for _, pkg := range pkgs {
		for imp := range pkg.Imports {
			if _, forbidden := hasForbiddenPrefix(imp); forbidden {
				return
			}
		}
	}
	t.Fatalf("no package in the module was found directly importing any of %v — this test cannot verify "+
		"enforcement; check that internal/mcp still imports the MCP SDK and that packages.Load resolved it",
		forbiddenMCPSDKPrefixes)
}

// TestInternalCLIImportsNoMCPSDK loads internal/cli's own production files
// (Tests: false, exactly one package expected) and fails, naming the
// package path and the offending import, if any DIRECT import carries a
// forbidden prefix. It also runs the self-defeat guard above, so this test
// cannot pass vacuously — either because the SDK left the tree entirely or
// forbiddenMCPSDKPrefixes went stale.
func TestInternalCLIImportsNoMCPSDK(t *testing.T) {
	cfg := &packages.Config{
		Mode:  packages.NeedName | packages.NeedImports,
		Tests: false,
	}
	pkgs, err := packages.Load(cfg, internalCLIPkgPath)
	if err != nil {
		t.Fatalf("packages.Load: %v", err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("packages.Load resolved %d packages for %s, want 1", len(pkgs), internalCLIPkgPath)
	}

	pkg := pkgs[0]
	for imp := range pkg.Imports {
		prefix, forbidden := hasForbiddenPrefix(imp)
		if !forbidden {
			continue
		}
		t.Errorf("package %s directly imports %s (forbidden prefix %q) — internal/cli must bootstrap "+
			"entirely through the internal/mcp.Server seam (mcp.NewStdioServer), never an MCP SDK package "+
			"itself (SDK-02)", pkg.PkgPath, imp, prefix)
	}

	assertMCPSDKImporterExists(t)
}
