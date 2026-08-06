// Package main is VULN-02's permanent, deliberately-vulnerable proof
// program — it exists only so `task vuln:selftest` has a real, reachable
// vulnerability to detect, proving the tool-modfile scan (see Taskfile.yml's
// `vuln` target) can actually fire rather than being a documented gate that
// never does.
//
// It targets advisory GO-2026-5932 (golang.org/x/crypto/openpgp) because
// that advisory is permanently unfixed upstream ("Fixed in: N/A" — the
// package is deprecated in favor of github.com/ProtonMail/go-crypto with no
// planned patch), which makes it a stable target: this program never needs
// re-pinning to stay red, unlike a fixable CVE that would eventually age
// out from under a selftest built against it.
//
// It lives under testdata/ specifically so the go tool excludes it from
// "./..." expansion (GOLDEN-01: the go tool ignores any directory named
// "testdata" when expanding "./..." — see Taskfile.yml's test:golden target
// for the same property used deliberately elsewhere in this repo). That
// exclusion is load-bearing here, not incidental: if this package ever
// entered the main module's package graph, it would turn ci.yml's blocking
// "govulncheck (DIST-03, blocking)" source-mode gate permanently red and
// could reach a release binary — importing a known-vulnerable package on
// purpose is safe only because it is structurally unreachable from the
// main module's build.
//
// It is never built by `go build ./...`, `go vet ./...`, or any other
// main-module target — only by `task vuln:selftest`, which builds it
// explicitly via `GOWORK=off go build -modfile=go.tool.mod ./testdata/vulnredpoc`
// (the same isolated-tool-modfile mechanism Taskfile.yml's `vuln` target
// uses to build task/goreleaser/govulncheck/actionlint), so it never adds a
// requirement to the root go.mod either.
package main

import (
	"fmt"
	"strings"

	"golang.org/x/crypto/openpgp" //nolint:staticcheck // GO-2026-5932: deliberate, see package doc comment above.
)

func main() {
	// The call must be reachable from main, not just imported — a bare
	// import resolves to a module-level-only match (govulncheck exits 0),
	// which proves nothing. Calling a vulnerable symbol against an empty
	// reader produces a real, cheap, side-effect-free symbol-level hit.
	_, err := openpgp.ReadArmoredKeyRing(strings.NewReader(""))
	fmt.Println("openpgp.ReadArmoredKeyRing returned:", err)
}
