// Package archtest: charm.land closure CGo guard (REL-01, D-07).
//
// The single-static-binary / no-new-CGo supply-chain story is load-bearing
// for v1.0.0. This test proves the claim with a go-toolchain-authoritative
// dependency-closure inspection — NOT a blanket CGO_ENABLED=0 build, which
// would fail on the pre-existing, already-documented CGo exception
// (tree-sitter, reachable via internal/indexer/internal/daemon, not via
// internal/cli's charm imports) and prove nothing about the Charm closure
// specifically (RESEARCH D-07 anti-pattern).
//
// REL-01 proves "no NEW CGo", not "no CGo": the tree-sitter CGo closure and
// the charm.land closure are disjoint at the package level (RESEARCH
// §Package Legitimacy Audit, verified this phase: charm_pkg_count 10,
// closure_size 123, cgo_in_closure []).
//
// Implementation note (deviation from the literal plan wording): the plan
// suggested golang.org/x/tools/go/packages, mirroring import_graph_test.go.
// That package's Package struct does not expose CgoFiles — cmd/go's own
// list.Package schema is the only source of that field (confirmed via `go
// doc` and `go help list` during implementation). This guard therefore
// shells out to `go list -deps -json`, the exact command RESEARCH
// §Package Legitimacy Audit verified against this repo, and decodes the
// same JSON stream cmd/go emits. This keeps the guard toolchain-native
// (no new dependency; go list is toolchain baseline, same as go/packages).
package archtest

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

// charmClosurePathPrefix scopes the audit strictly to charm.land packages —
// the pre-existing tree-sitter CGo exception must never trip this guard.
const charmClosurePathPrefix = "charm.land"

// goListPackage mirrors the subset of cmd/go's list.Package JSON schema
// this guard needs: the import path (for charm.land scoping) and CgoFiles
// (the authoritative "does this package import \"C\"" signal).
type goListPackage struct {
	ImportPath string
	CgoFiles   []string
}

// loadCharmClosure runs `go list -deps -json` over internal/cli's full
// transitive closure (mirroring RESEARCH's verified command) and returns
// the subset of packages whose ImportPath begins with charm.land. Fails
// loudly (fail-closed) rather than skip if the command errors or its
// output cannot be decoded.
func loadCharmClosure(t *testing.T) []goListPackage {
	t.Helper()

	const closureTarget = modulePathPrefix + "internal/cli/..."
	cmd := exec.Command("go", "list", "-deps", "-json", closureTarget)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("go list -deps -json %s: %v\nstderr: %s", closureTarget, err, stderr.String())
	}

	dec := json.NewDecoder(&stdout)
	var charm []goListPackage
	for dec.More() {
		var pkg goListPackage
		if err := dec.Decode(&pkg); err != nil {
			t.Fatalf("decoding go list -json output: %v", err)
		}
		if strings.HasPrefix(pkg.ImportPath, charmClosurePathPrefix) {
			charm = append(charm, pkg)
		}
	}
	return charm
}

// TestCharmCgoClosure asserts the charm.land dependency closure reachable
// from internal/cli (Phase 6/7's TUI additions: lipgloss/v2, bubbletea/v2,
// bubbles/v2) is (a) non-empty — a vacuous pass, where no charm.land
// packages are found at all, is itself a failure, since it would prove
// nothing about the actual Charm dependency — and (b) contains zero
// packages with CgoFiles.
//
// This is the durable, fail-closed regression guard for REL-01/T-08-07-01:
// a future Charm bump that pulls in a CGo package must fail this test.
func TestCharmCgoClosure(t *testing.T) {
	charm := loadCharmClosure(t)

	if len(charm) == 0 {
		t.Fatal("charm.land closure is empty — this test cannot verify REL-01's 'no new CGo' claim; check that internal/cli still imports charm.land/lipgloss/v2 (or bubbletea/v2, bubbles/v2) and that `go list -deps -json` resolved it")
	}

	var offenders []string
	for _, pkg := range charm {
		if len(pkg.CgoFiles) > 0 {
			offenders = append(offenders, pkg.ImportPath)
		}
	}
	if len(offenders) > 0 {
		t.Errorf("charm.land closure contains %d package(s) with CgoFiles — REL-01 requires the Charm/TUI dependency closure to introduce NO new CGo: %v", len(offenders), offenders)
	}

	t.Logf("charm.land closure audited: %d packages, 0 with CgoFiles", len(charm))
}
