---
phase: 07-guards-that-cannot-fire
reviewed: 2026-09-08T00:00:00Z
depth: deep
files_reviewed: 6
files_reviewed_list:
  - internal/bench/regression.go
  - internal/bench/regression_test.go
  - internal/query/archtest/import_direction_test.go
  - internal/upgrade/release_workflow_shape_test.go
  - scripts/inject-cosign-key.sh
  - Taskfile.yml
findings:
  critical: 1
  warning: 1
  info: 1
  total: 3
status: issues_found
---

# Phase 07: Code Review Report

**Reviewed:** 2026-09-08
**Depth:** deep
**Files Reviewed:** 6
**Status:** issues_found

## Summary

This phase's own `07-MUTATION-LOG.md` is unusually rigorous: each of the four rewritten guard families (GRD-01 through GRD-04) already has a documented RED/GREEN demonstration with pasted failing output and a proven byte-clean revert. I reproduced all four independently (running the actual test suites and the shell script by hand against synthetic and real fixtures) and every one of them holds up — `CheckRegression`'s degenerate-current checks, the `internal/query` archtest's wire-layer/indexer-root/runner/scratch_fs checks, `inject-cosign-key.sh`'s additions-only + exactly-one-`--key=` assertion, and `TestPostReleaseJobsDeclareConclusionGuard`'s content (not merely presence) check all fail correctly against the specific defect each was built to catch, and the two rewired `Taskfile.yml` targets now call the extracted script with byte-identical semantics to what they inlined before.

The one place this phase's own standard — "a guard MUST carry a positive assertion that it did its work" — was not fully carried through is the new `internal/query` archtest itself (GRD-02). I proved by direct reproduction (not speculation) that `TestQueryImportsNoWireLayerOrIndexerRoot` passes silently when a package in `internal/query`'s dependency tree has an import that cannot even resolve (a `go/packages` load error), because the test checks `packages.Load`'s top-level `error` return but never inspects each package's `.Errors` field or calls `packages.PrintErrors`. This is exactly the class of vacuous-pass this phase exists to close, on the one guard in scope that wasn't specifically mutation-tested for it.

## Critical Issues

### CR-01: `internal/query` archtest passes vacuously when a dependency fails to load

**File:** `internal/query/archtest/import_direction_test.go:88-176`

**Issue:** `TestQueryImportsNoWireLayerOrIndexerRoot` calls `packages.Load` and checks only the top-level `error` return (line 97-99) and `len(pkgs) == 0` (line 100-102). `go/packages` does not surface a per-package load failure (e.g., an import of a package that does not exist, a build-tag-excluded file, a genuine compile error reachable from `internal/query`) through that top-level error — it records it in the affected `packages.Package.Errors` field instead, while `Load` itself still returns `err == nil` and a non-empty `pkgs` slice. Because the broken package is either absent from, or has an incomplete `Imports` map in, the returned graph, `transitiveDeps` silently under-reports that package's dependency set, and the forbidden-import assertions look for members that were never added — the check "passes" while actually verifying nothing about the broken subtree.

Reproduced directly (not simulated): added a single blank import of a nonexistent package to `internal/query/traverse.go` (`_ "github.com/seanb4t/codegraph-go/internal/does-not-exist-xyz123"`), which makes `internal/query` fail to build. `GOTOOLCHAIN=go1.26.6 go test -count=1 -v ./internal/query/archtest/...` still reports:
```
--- PASS: TestQueryImportsNoWireLayerOrIndexerRoot (0.07s)
ok  	github.com/seanb4t/codegraph-go/internal/query/archtest	0.177s
```
A standalone `go/packages` probe against the same broken tree confirms the mechanism: `packages.Load` returns `err == nil`, while `packages.PrintErrors(pkgs)` reports exactly 1 error (`no required module provides package ...`). The change was reverted immediately after capture; `git diff --quiet -- internal/query/traverse.go` is clean.

This is not a hypothetical edge case for this package: `internal/query` is exactly the read-side boundary this test exists to police, and a partially-resolvable import graph (a typo'd import, a since-renamed internal package, a build-tag-gated file on the CI platform) is precisely the kind of drift that should turn this test red rather than let it quietly stop checking anything for the affected package.

**Fix:**
```go
pkgs, err := packages.Load(cfg, queryPrefix+"/...")
if err != nil {
	t.Fatalf("packages.Load: %v", err)
}
if len(pkgs) == 0 {
	t.Fatal("packages.Load returned no packages — the module import graph did not resolve")
}
if n := packages.PrintErrors(pkgs); n > 0 {
	t.Fatalf("packages.Load reported %d package error(s) — the import graph did not fully resolve, so this test cannot verify anything about the affected package(s); see errors above", n)
}
```
`packages.PrintErrors` both prints each package's errors to stderr (useful for CI diagnosis) and returns the count, so a single non-zero check both surfaces and fails on any partial-load condition before the (now trustworthy) transitive-dependency walk runs.

## Warnings

### WR-01: `inject-cosign-key.sh` embeds `COSIGN_KEY` into generated YAML unescaped

**File:** `scripts/inject-cosign-key.sh:54-57`

**Issue:** The injected line is built as `"      - \"--key=${COSIGN_KEY}\""` and passed to `awk -v keyline=...`. `awk -v` safely handles arbitrary bytes as an awk *value*, but the value itself is a YAML double-quoted scalar — if `$3` (the cosign key path) ever contained a literal `"` or backslash, the emitted line would no longer be valid YAML (or would silently truncate the flag value at the embedded quote), and the "additions-only" diff guard would not catch this because the line is still syntactically an *addition*, just a malformed one. Today both call sites in `Taskfile.yml` pass a path under a `mktemp -d` directory they control, so this is not currently reachable, but the script's own usage comment says the key path argument is deliberately "not checked for existence" so this script can run standalone — nothing else in the script's contract guarantees the value is quote-safe either.

**Fix:** Quote-escape the value before interpolating it into the YAML scalar, e.g.:
```bash
ESCAPED_KEY=$(printf '%s' "${COSIGN_KEY}" | sed 's/[\\"]/\\&/g')
awk -v keyline="      - \"--key=${ESCAPED_KEY}\"" '...'
```
or, more simply, document explicitly in the usage comment that `$3` must not contain `"` or `\`, matching the guarantee both call sites already happen to provide.

## Info

### IN-01: `containsFold` reimplements case-insensitive substring search instead of using `strings`

**File:** `internal/bench/regression_test.go:563-589`

**Issue:** `containsFold` hand-rolls a rune-by-rune case-insensitive substring scan with an ASCII-only case fold (`a >= 'A' && a <= 'Z'`), justified by a comment claiming it avoids "import[ing] strings twice with different casing assumptions." That justification doesn't hold up — a single `strings.Contains(strings.ToLower(s), strings.ToLower(substr))` (one import of `"strings"`, used twice) achieves the same result with less custom code to maintain, and `strings.ToLower` correctly handles non-ASCII case folding that the hand-rolled version does not (immaterial for today's ASCII-only error strings, but a needless divergence from the standard library for no real benefit).

**Fix:**
```go
import "strings"

func containsFold(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
```

---

_Reviewed: 2026-09-08_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
