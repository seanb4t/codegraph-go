// stdout_transport_allowlist_selftest_test.go proves
// stdoutTransportWriterAllowlist (stdout_confinement_test.go) is scoped as
// narrowly as its own doc comment claims: it suppresses HYG-02's os.Stdout
// violation ONLY inside internal/mcp's real ServeStdio function, never
// anywhere else in the same file or package — a violation planted in a
// DIFFERENT function is still caught (D-08 mutation-proof discipline
// applied to the allowlist itself, mirroring
// stdout_closure_selftest_test.go's technique for the closure walk and
// stdout_detection_selftest_test.go's technique for the raw predicates).
package archtest

import (
	"os"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

// internalMCPServerGoPath resolves the real, on-disk absolute path of
// internal/mcp/server.go via packages.Load itself, mirroring
// stdout_closure_selftest_test.go's schemaMetaGoPath helper, so the
// overlay key below is guaranteed to match what packages.Load uses
// internally when it re-resolves the module for this test's own
// closeOverServeReachableImports call.
func internalMCPServerGoPath(t *testing.T) string {
	t.Helper()
	cfg := &packages.Config{Mode: packages.NeedFiles | packages.NeedCompiledGoFiles}
	pkgs, err := packages.Load(cfg, "github.com/seanb4t/codegraph-go/internal/mcp")
	if err != nil {
		t.Fatalf("resolving internal/mcp package files: %v", err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("packages.Load resolved %d packages for internal/mcp, want 1", len(pkgs))
	}
	for _, f := range pkgs[0].GoFiles {
		if strings.HasSuffix(f, "/server.go") || f == "server.go" {
			return f
		}
	}
	t.Fatalf("internal/mcp/server.go not found among resolved files: %v", pkgs[0].GoFiles)
	return ""
}

// TestStdoutTransportAllowlistDoesNotOverSuppress plants a SECOND,
// synthetic os.Stdout reference inside internal/mcp/server.go via
// packages.Config.Overlay (no file on disk is ever touched), in a
// function named something other than "ServeStdio", and asserts the
// closure scan STILL flags it — proving
// stdoutTransportWriterAllowlist["internal/mcp"] = "ServeStdio" is scoped
// to that one function, not the whole file or package. Without this
// proof, a blanket per-package or per-file suppression could silently
// hide a genuine future stdout leak added anywhere else in internal/mcp.
func TestStdoutTransportAllowlistDoesNotOverSuppress(t *testing.T) {
	serverGoPath := internalMCPServerGoPath(t)

	original, err := os.ReadFile(serverGoPath)
	if err != nil {
		t.Fatalf("reading %s: %v", serverGoPath, err)
	}

	// Append a new function — not "ServeStdio" — that references
	// os.Stdout directly. internal/mcp/server.go already imports "os", so
	// no import line needs to be added.
	violated := string(original) +
		"\nfunc stdoutGuardAllowlistProbe() { _ = os.Stdout }\n"

	reachable := closeOverServeReachableImports(t, map[string][]byte{serverGoPath: []byte(violated)})

	const mcpPkgPath = "github.com/seanb4t/codegraph-go/internal/mcp"
	if _, ok := reachable[mcpPkgPath]; !ok {
		t.Fatalf("%s was not present in the closure — cannot prove the overlaid violation was even scanned", mcpPkgPath)
	}

	violations := scanForStdoutViolations(reachable)
	found := false
	for _, msg := range violations {
		if strings.HasPrefix(msg, mcpPkgPath+": references os.Stdout") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("planted an os.Stdout reference in internal/mcp's stdoutGuardAllowlistProbe (not the allowlisted ServeStdio) but the closure scan did not flag it — stdoutTransportWriterAllowlist is over-suppressing; got violations: %v", violations)
	}
}
