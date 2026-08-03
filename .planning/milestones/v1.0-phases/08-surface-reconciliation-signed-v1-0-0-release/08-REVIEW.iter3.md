---
phase: 08-surface-reconciliation-signed-v1-0-0-release
reviewed: 2026-07-19T21:41:39Z
depth: deep
files_reviewed: 8
files_reviewed_list:
  - internal/query/validate.go
  - internal/query/files.go
  - internal/query/traverse.go
  - internal/query/files_status_test.go
  - internal/query/traverse_test.go
  - internal/cli/affected.go
  - internal/cli/affected_test.go
  - internal/upgrade/upgrade.go
findings:
  critical: 1
  warning: 2
  info: 0
  total: 3
status: issues_found
---

# Phase 8: Code Review Report (iteration 2 — re-review)

**Reviewed:** 2026-07-19T21:41:39Z
**Depth:** deep
**Files Reviewed:** 8
**Status:** issues_found

## Narrative Findings (AI reviewer)

### Summary

This is iteration 2 of the `--auto` fix loop, re-verifying the 6 fix commits
(`9019fe4`, `ea2b889`, `d887e38`, `43c25ae`, `8a6ac3b`, `d3f077c`) applied
against iteration-1's findings. Verification method: read every changed
file in full, re-read each fix's own diff (`git show`), cross-checked
against `internal/query/traverse_test.go`/`internal/cli/affected_test.go`,
ran `go build`, `go vet`, `staticcheck`, the package test suites
(`internal/query`, `internal/cli`, `internal/upgrade`), and the golden
parity suite (`testdata/golden/...`) — all green, consistent with the
re-review context's stated baseline.

**Verified correct, no regression:**
- **WR-01** (`dirPrefixMatches`, `ea2b889`): now requires a path-separator
  boundary (exact match or `dir+"/"` prefix) on both the plain and
  `./`-prefixed forms; still a pure prefix check (no glob semantics
  introduced, D-03 preserved). `files_status_test.go`'s new `pkgab/bar.go`
  vs. `pkga` cases directly pin the fixed boundary.
- **WR-02** (`AffectedResult.Files` nil→`[]string{}`, `d887e38`): mirrors
  the existing `Callees`/`Callers`/`Affected.AffectedTests` empty-slice
  convention exactly; `TestZeroMatchJSONShapesAreEmptyArraysNotNull`'s new
  "Affected nil files argument" subtest proves it. No double-normalization.
- **WR-04** (dispatch-sibling composition in `Affected`'s BFS, `43c25ae`):
  mirrors `Impact`'s existing frontier/`dispatchSiblingIDs` composition
  exactly (test-leaf pruning is unaffected — a leaf test dependent is
  never added to `next`, so `dispatchSiblingIDs` is never invoked on it);
  `TestAffected_DispatchTraversal` /
  `TestAffected_DispatchTraversal_NoImplementsEdgesUnaffected` cover both
  the positive and no-op cases. Golden corpus (`TestGoldenBehavioralRealCorpora`)
  still passes.
- **WR-07** (`isTestSymbol` tightened to `_test.go`-suffix only, `8a6ac3b`):
  confirmed the only caller sites are `traverse.go`'s `Affected` and
  `explore.go`'s caller-classification (`explore.go:84`) — both intend
  "is this node declared in a test file," so dropping the
  `Test*`/`Benchmark*` name fallback is applied consistently, not just in
  one call site. `TestIsTestSymbol`'s new cases cover the previously
  misclassified production-function names.
- **WR-05** (`sortLocations`, `d3f077c`) for **Callers** and **Impact**:
  `(FilePath, Name, StartLine)` via `sort.SliceStable`, applied before
  `limit`/`MaxLimit` capping (so the capped subset is itself deterministic).
  This is a defensible, documented ordering (mirrors `sortFileTree`'s
  existing "reproducible JSON output" convention, D-06) — not an arbitrary
  break. `TestImpact`'s updated `Affected[0]=="Alpha"` (not `"helper"`)
  assertion correctly reflects the new order (`"Alpha" < "helper"`
  lexicographically), and the new
  `TestImpactCallersAffectedDeterministicAcrossRepeatedCalls` proves
  byte-identical JSON across 5 repeated calls. Golden parity's
  `impact`/`callers` subtests compare via set/count, not exact array order,
  so this doesn't regress them (confirmed by running the suite).

**New issues found in this iteration** (see below): one fix (WR-06) does
not actually do what its own doc comment and commit message claim, and two
of the six commits leave the codebase in a state that's internally
inconsistent or has dead code / a coverage gap. None of these was caught
by the green test suite, `go vet`, or `staticcheck` — all require reading
the fix's actual runtime semantics, not just its stated intent.

### Critical Issues

#### CR-01: WR-06's `scanner.Buffer` call does not enforce `affectedStdinMaxLineBytes` — the real per-line ceiling is silently 64 KiB, not 4096 bytes

**File:** `internal/cli/affected.go:216`

**Issue:** The fix's intent (per its own doc comment on `affectedStdinMaxLineBytes`, lines 159-166, and the `WR-06` doc comment on `collectAffectedFiles`, lines 186-191) is: a `--stdin` line longer than 4096 bytes (`affectedStdinMaxLineBytes`) is malformed input and must surface `bufio.ErrTooLong` as an explicit error. The actual call is:

```go
scanner.Buffer(make([]byte, 0, 64*1024), affectedStdinMaxLineBytes)
```

Per `bufio.Scanner.Buffer`'s documented contract: *"The maximum token size must be less than the larger of max and cap(buf)."* Here `cap(buf) == 65536` and `max == 4096` — the **larger** of the two is `65536`, so the actual enforced ceiling is 64 KiB, not the intended 4096 bytes. A line between 4097 and 65536 bytes is silently accepted with no error, exactly the "silently truncating/dropping" failure mode WR-06 was supposed to close. I confirmed this behavior directly:

```go
// reproduction: a 10000-byte line, with max=4096 passed to scanner.Buffer
scanner.Buffer(make([]byte, 0, 64*1024), 4096)
// scanner.Scan() succeeds, scanner.Err() == nil — no ErrTooLong surfaced
```
Output: `scanned line length: 10000` / `no error — 10000-byte line was accepted despite max=4096`.

This is not merely a documentation mismatch — it is the actual security/robustness property WR-06 was meant to add (bounding a single `--stdin` line to a sane, PATH_MAX-scale ceiling and surfacing an explicit error on anything past it), and it doesn't hold. Also note the *pre-fix* code had no `scanner.Buffer` call at all, meaning the default `bufio.MaxScanTokenSize` (64 KiB) applied — so despite the added code and the extensive doc comments, the actual per-line ceiling is completely unchanged by this commit. No test in the suite exercises a line between 4097 and 65536 bytes, which is why this passed CI.

**Fix:** Make the initial buffer's capacity no larger than the intended ceiling, so `max` is actually the binding constraint:

```go
scanner.Buffer(make([]byte, 0, affectedStdinMaxLineBytes), affectedStdinMaxLineBytes)
```

(or pass `nil`/a zero-cap buffer as the first argument — `cap(nil) == 0 < affectedStdinMaxLineBytes`, so `max` still wins). Add a regression test asserting a stdin line of, say, `affectedStdinMaxLineBytes+1` bytes produces the documented `"affected: --stdin line exceeds maximum %d bytes"` error, and a line of exactly `affectedStdinMaxLineBytes` bytes succeeds.

### Warnings

#### WR-01: `ValidateAffectedFiles` is dead code — CR-01's cap enforcement is duplicated inline instead of calling it, and has no test coverage

**File:** `internal/query/validate.go:153` / `internal/cli/affected.go:200-202`

**Issue:** Commit `9019fe4` (CR-01) added an exported `query.ValidateAffectedFiles(n int) error`, whose doc comment states: *"Exported so internal/cli/affected.go's collectAffectedFiles can enforce the cap as it reads stdin, before Engine.Affected is ever called."* It is never called anywhere — `collectAffectedFiles`'s `add` closure reimplements the identical check inline instead:

```go
if len(files)+1 > query.MaxAffectedFiles {
    return fmt.Errorf("affected: input exceeds maximum %d files", query.MaxAffectedFiles)
}
```

Confirmed via `grep -rn "ValidateAffectedFiles"` — the only occurrences are the function's own definition and doc comment in `validate.go`; no call site exists in the codebase, and no test calls it either. This is currently harmless in behavior (both implementations enforce the same `n > MaxAffectedFiles` boundary correctly, verified: allows exactly 10000 unique files, rejects the 10001st), but it's a genuine defect in the *fix itself*: an exported, documented, "this is why it exists" API that its own commit message describes as the enforcement path is quietly unused, leaving two independent sources of truth for the same validation rule that can silently diverge on a future edit to either one. There is also no test anywhere (`internal/cli` or `internal/query`) that exercises the actual >10000-file rejection path end-to-end — the CR-01 fix's headline behavior is untested.

**Fix:** Either (a) have `collectAffectedFiles`'s `add` closure call `query.ValidateAffectedFiles(len(files) + 1)` instead of re-deriving the same comparison inline, so there is one source of truth, or (b) if the CLI-local inline check is intentionally kept lightweight, delete the unused exported `ValidateAffectedFiles` (and its doc comment's false claim) rather than leaving a misleading, uncalled API. Either way, add a test that feeds `query.MaxAffectedFiles+1` distinct lines via `--stdin` and asserts the `"input exceeds maximum %d files"` error is returned.

#### WR-02: `Callees` was left out of the WR-05 determinism fix — it has a different, unproven ordering contract than its three sibling functions

**File:** `internal/query/traverse.go` (`Callees`, ~lines 290-334; `sortLocations`, line 242)

**Issue:** Commit `d3f077c` (WR-05) added `sortLocations` (canonical `(FilePath, Name, StartLine)` order) and applied it to `Callers` (line 398), `Impact` (line 489), and `Affected` (line 634) — each with a matching new regression test in `TestImpactCallersAffectedDeterministicAcrossRepeatedCalls`. `Callees` has the exact same structural shape (`locs := []Location{}`, append-only, then `limit`/`MaxLimit` capping) and the same rationale the commit gives for the other three applies verbatim — its order currently comes from `IterateEdges(node.Id)`'s raw Pebble key-range scan order (confirmed via `internal/graphstore/pebble_store.go:285`), not from any ordering this package asserts or tests. In production this happens to be stable across repeated calls against the same store snapshot (Pebble's sorted-key iteration), but it is a *different* order than the `(FilePath, Name, StartLine)` contract the other three now guarantee — an inconsistency in the very "reproducible, package-owned ordering" convention this fix was meant to establish uniformly, and `Callees` has no analogous determinism test (`TestImpactCallersAffectedDeterministicAcrossRepeatedCalls` does not cover it).

**Fix:** Add `sortLocations(locs)` to `Callees` immediately before its `limit`/`MaxLimit` capping (mirroring `Callers`'s placement), and add a `Callees` subtest to `TestImpactCallersAffectedDeterministicAcrossRepeatedCalls` (or a sibling test) asserting byte-identical JSON across repeated calls, matching the other three.

---

_Reviewed: 2026-07-19T21:41:39Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
