---
phase: 08-surface-reconciliation-signed-v1-0-0-release
fixed_at: 2026-07-19T21:45:00Z
review_path: .planning/phases/08-surface-reconciliation-signed-v1-0-0-release/08-REVIEW.md
iteration: 1
findings_in_scope: 9
fixed: 8
skipped: 1
status: partial
---

# Phase 08: Code Review Fix Report

**Fixed at:** 2026-07-19
**Source review:** .planning/phases/08-surface-reconciliation-signed-v1-0-0-release/08-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 9 (1 critical, 7 warning, 1 info — full `all` scope)
- Fixed: 8
- Skipped: 1 (IN-01, low priority, rationale below)

All fixes were verified per-fix (`go build ./...` + the relevant test package) and, before this report was written, against the full suite (`go test ./... -count=1`) and the golden corpus harness (`go test ./testdata/golden/... -count=1`). Both passed except two known-flaky, timing-sensitive `internal/daemon` tests (`TestRunWatchdogCancelsRunOnSimulatedReparent`, `TestDaemonRunWaitsForInFlightFlushBeforeReleasingLock`, and intermittently `TestDaemonFlushLockRequeueGivesUpPerEpisode`) that are unrelated to any file touched by this fix pass (`internal/daemon` has no import dependency on `internal/query` or `internal/cli/affected.go`) — confirmed flaky by re-running `internal/daemon` alone twice, once passing clean.

## Fixed Issues

### CR-01: `affected --stdin` has no cap on the number of ingested lines — unbounded-memory DoS

**Files modified:** `internal/query/validate.go`, `internal/cli/affected.go`
**Commit:** 9019fe4
**Applied fix:** Added `query.MaxAffectedFiles` (10000) and `query.ValidateAffectedFiles`, mirroring `validateMaxFiles`/`validateLimit`'s "reject outright" posture. `collectAffectedFiles` now checks the cap as each stdin line is added and returns an error (rather than silently truncating) once exceeded; `RunE` was updated to propagate the new `(files, err)` return. Committed together with WR-03/WR-06 since all three live in the same `collectAffectedFiles`/`--quiet` code region in `affected.go` and would have produced overlapping diff hunks if split into three commits.

### WR-01: `dirPrefixMatches` has no path-separator boundary check

**Files modified:** `internal/query/files.go`, `internal/query/files_status_test.go`
**Commit:** ea2b889
**Applied fix:** `dirPrefixMatches` now requires an exact segment match or a following `/` (checked against both the un-prefixed and `"./"`-prefixed forms of `dir`), instead of a bare `strings.HasPrefix`. Kept the "prefix match, not a glob" contract (D-03) intact — verified via the existing `"dir is not treated as a glob"` subtest, unchanged. Extended the `dirPrefixMatches` table test with 4 new sibling-collision regression cases (`pkgab/bar.go` vs `pkga`/`pkga/`, the real `pkga/foo.go` match, and an exact-path boundary case).

### WR-02: `AffectedResult.Files` has no nil→empty-slice normalization

**Files modified:** `internal/query/traverse.go`, `internal/query/traverse_test.go`
**Commit:** d887e38
**Applied fix:** `Engine.Affected` now normalizes a nil `files` argument to `[]string{}` before building the result, mirroring the `tests := []Location{}` convention the function (and `Callees`/`Callers`) already use. Added a new `"Affected nil files argument"` subtest to `TestZeroMatchJSONShapesAreEmptyArraysNotNull` asserting `"files"` never marshals as JSON `null` — the one array field that regression suite previously never exercised.

### WR-03: `affected --quiet` writes raw, unescaped file paths

**Files modified:** `internal/cli/affected.go`
**Commit:** 9019fe4 (bundled with CR-01/WR-06, see above)
**Applied fix:** The `--quiet` output loop now skips (rather than emits) any `FilePath` containing an embedded `\n`/`\r` via `strings.ContainsAny`, preventing an adversarially-named indexed file from injecting an extra line into the one-path-per-line machine-readable stream.

### WR-04: `Affected`'s reverse BFS omits the RES-02 interface-dispatch composition

**Files modified:** `internal/query/traverse.go`, `internal/query/traverse_test.go`
**Commit:** 43c25ae
**Applied fix:** `Affected` now builds `BuildImplementsIndex`/`buildContainsIndex` alongside its existing reverse-adjacency map and composes `dispatchSiblingIDs` at every frontier hop — exactly mirroring `Impact`'s existing composition. Frontier changed from `[]string` (node IDs) to `[]*schema.Node` to carry the node object `dispatchSiblingIDs` needs. Added `TestAffected_DispatchTraversal` (proves the positive case: a test reachable only through a sibling implementer's same-named method is now surfaced) and `TestAffected_DispatchTraversal_NoImplementsEdgesUnaffected` (proves the composition is a strict no-op with no implements edges, the common case). Golden corpus and full test suite pass unchanged.

### WR-05: Traversal results are never independently sorted before being returned

**Files modified:** `internal/query/traverse.go`, `internal/query/traverse_test.go`
**Commit:** d3f077c
**Applied fix:** Added `sortLocations` (sorts by `FilePath`, then `Name`, then `StartLine`, via `sort.SliceStable`) and applied it to `Impact.Affected`, `Callers.Callers`, and `Affected.AffectedTests` before returning — for `Callers`, sorting happens *before* the `limit`/`MaxLimit` cap so the capped subset is itself deterministic, not just its order. Added `TestImpactCallersAffectedDeterministicAcrossRepeatedCalls`, which asserts byte-identical JSON across 5 repeated calls for all three result types. Golden-corpus parity tests compare results as a **set** (`assertSubset`/`toLocSet`, D-05b), so they were unaffected by the ordering change.

**Requires human verification:** this fix changed an established ordering behavior that had a passing unit test locking it in — `TestImpact`'s `depth=2` subtest previously asserted `Affected[0].Name == "helper"` (an incidental BFS-visitation-order artifact, not a documented API contract). That assertion was updated to expect the new sorted order (`Alpha, helper, Run`). No downstream code was found indexing into `Affected[0]`/`Callers[0]`/`AffectedTests[0]` by position outside this one test, but please confirm no external consumer (CLI rendering, a future MCP client) relies on the old "self/BFS-order-first" ordering before this phase is signed off — `internal/cli/impact.go`'s renderer iterates the full slice and does not special-case index 0.

### WR-06: `bufio.Scanner`'s default 64KB per-line limit silently truncates the entire remaining stdin stream

**Files modified:** `internal/cli/affected.go`
**Commit:** 9019fe4 (bundled with CR-01/WR-03, see above)
**Applied fix:** `scanner.Buffer` now sets an explicit ceiling of 4096 bytes (`affectedStdinMaxLineBytes`, the common `PATH_MAX`) — legitimately long paths are no longer at risk of the default 64KB limit's coarser behavior, and a genuine `bufio.ErrTooLong` (which indicates malformed input, not ordinary EOF) is now surfaced as an explicit error via `scanner.Err()` instead of being silently swallowed along with every line after the oversized one.

### WR-07: `isTestSymbol`'s naming heuristic can misclassify production code as a test symbol

**Files modified:** `internal/query/traverse.go`, `internal/query/traverse_test.go`
**Commit:** 8a6ac3b
**Applied fix:** Dropped the `Name`-based `Test*`/`Benchmark*` fallback entirely, per the review's own reasoning that the `_test.go`-suffix check alone already covers Go's test-naming convention (a `TestXxx` function not in a `_test.go` file cannot be run by `go test` in the first place). Added `TestIsTestSymbol` directly pinning both the positive cases (real test/benchmark functions, and any function in a `_test.go` file) and the negative cases (production functions merely named like a test, outside a `_test.go` file).

## Skipped Issues

### IN-01: `downloadReleaseAsset` has no `context.Context` for early cancellation

**File:** `internal/upgrade/upgrade.go:221-224`
**Reason:** Threading a `context.Context` cleanly through this call requires changing the public `Run(currentVersion, targetPath string, opts Options) error` signature (and/or adding a `Context` field to `Options`), the unexported `downloadFunc`/`defaultDownload` injectable-seam type, `downloadReleaseAsset` itself, and the `internal/cli/upgrade.go` call site — plus updating roughly a dozen `Run(...)` call sites in `upgrade_test.go`. This is a public-API-shape change for a low-priority ergonomic improvement (the existing 5-minute `downloadHTTPClient.Timeout` already bounds every call; only early Ctrl+C cancellation is missing). Given the fix guidance's explicit "low priority... else skip-with-rationale" instruction and this phase's "correctness and NOT breaking green tests matter more than fixing every nit" priority for the v1.0 release, this was left unfixed rather than risking a signature-ripple change this late in the release phase. Deferred to a follow-up phase.

**Original issue:** `downloadHTTPClient.Get(url)` is called with a `//nolint:gosec,noctx` suppression; a user hitting Ctrl+C mid-download has no way to short-circuit before the 5-minute timeout ceiling.

---

_Fixed: 2026-07-19_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
