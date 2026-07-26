---
phase: 08-surface-reconciliation-signed-v1-0-0-release
fixed_at: 2026-07-19T21:52:00Z
review_path: .planning/phases/08-surface-reconciliation-signed-v1-0-0-release/08-REVIEW.md
iteration: 2
findings_in_scope: 3
fixed: 3
skipped: 0
status: complete
---

# Phase 08: Code Review Fix Report (iteration 2)

**Fixed at:** 2026-07-19
**Source review:** `.planning/phases/08-surface-reconciliation-signed-v1-0-0-release/08-REVIEW.md` (iteration-2 re-review)
**Iteration:** 2
**Iteration-1 report archived at:** `08-REVIEW-FIX.iter2.md`

**Summary:**
- Findings in scope: 3 (1 critical, 2 warning)
- Fixed: 3
- Skipped: 0

All three findings were **incomplete or no-op fixes from iteration 1** that the
`--auto` re-review loop caught — the same failure mode recorded in this
project's Phase-6 history ("CR-01 sanitizer fix was INCOMPLETE", "fix composed
into new critical"). None were new defects introduced by iteration 1; each was
an iteration-1 fix that looked correct in review but did not actually take
effect.

## Fixed Issues

### CR-01: WR-06's `scanner.Buffer` call did not enforce `affectedStdinMaxLineBytes` — the real ceiling was silently 64 KiB, not 4096 bytes

**Files modified:** `internal/cli/affected.go`, `internal/cli/affected_test.go`
**Commit:** `333ab82`
**Applied fix:** `bufio.Scanner.Buffer(buf, max)`'s effective per-token ceiling is
`max(max, cap(buf))`, not `max` — so iteration 1's
`scanner.Buffer(make([]byte, 0, 64*1024), affectedStdinMaxLineBytes)` left the
real limit at the 65536-byte initial capacity and the intended 4096-byte cap
never fired. The initial capacity is now allocated at exactly
`affectedStdinMaxLineBytes`, so the two arguments agree and the cap is genuinely
enforced. The `bufio.ErrTooLong` surfacing added in iteration 1 (WR-06) was
already correct and is unchanged — it simply had no reachable trigger until now.
Added `TestAffectedStdinLineTooLong` with both boundary cases: a line exceeding
the cap by one byte errors, and a line just under the cap succeeds.

### WR-01: `ValidateAffectedFiles` was dead code — CR-01's cap was duplicated inline instead of calling it, with no test coverage

**Files modified:** `internal/cli/affected.go`, `internal/cli/affected_test.go`
**Commit:** `331a5a6`
**Applied fix:** `collectAffectedFiles` reimplemented the bound inline as
`len(files)+1 > query.MaxAffectedFiles`, leaving the exported
`query.ValidateAffectedFiles` — added by iteration 1 specifically to own this
bound — unreferenced by any caller. The inline check is replaced with a call to
`query.ValidateAffectedFiles(len(files) + 1)`, wrapping its error with the
`affected:` command prefix. The bound now has exactly one definition site,
matching the `validateMaxFiles`/`validateLimit` convention the rest of
`internal/query/validate.go` follows. Added `TestAffectedStdinTooManyFiles`
pinning rejection past `MaxAffectedFiles` — the coverage the finding called out
as absent.

### WR-02: `Callees` was left out of the WR-05 determinism fix

**Files modified:** `internal/query/traverse.go`, `internal/query/traverse_test.go`
**Commit:** `4feb6ff`
**Applied fix:** Iteration 1's WR-05 fix (`d3f077c`) applied `sortLocations` to
`Impact`, `Callers`, and `Affected` but skipped `Callees`, leaving one of the
four traversals with its ordering determined by `IterateEdges`'s raw Pebble
key-range scan order. `Callees` now calls `sortLocations(locs)` **before** the
`limit`/`MaxLimit` truncation, so the cap selects a deterministic prefix rather
than an arbitrary one — matching the ordering and the sort-then-truncate
sequencing its three siblings already use. Added
`TestCalleesSortedDeterministically`.

## Verification

Verified directly on the integration branch after fast-forwarding (not taken on
the fixer's report alone — the iteration-2 fixer was interrupted before it could
self-report, so every claim below was re-run first-hand):

- `go build ./...` — clean (only a pre-existing C macro-redefinition **warning**
  from the vendored `alex-pinkus/tree-sitter-swift` scanner; not an error and
  not from this phase).
- `go test ./internal/cli/... ./internal/query/... -count=1` — all green.
- Full suite excluding `internal/daemon` — all green, including the golden
  parity corpus (`testdata/golden/...`).
- Each fix's diff was read directly (`git diff d3f077c..HEAD`) rather than
  inferred from the commit subjects.

**Known-flaky, not a regression:** `internal/daemon` showed two failures under
full-suite parallel load — `TestRunWatchdogCancelsRunOnSimulatedReparent` and
`TestSoak` — and both pass when the package is run in isolation (`ok 3.432s`).
`git diff 7e10906..HEAD -- internal/daemon/` is **empty**: no Phase-8 commit,
in either fix iteration, touched that package. This widens the previously
recorded flaky set in `internal/daemon` from two timing tests to four; all four
are load-sensitive and unrelated to this phase's changes.

---

_Fixed: 2026-07-19_
_Fixer: Claude (gsd-code-fixer, iteration 2 — interrupted before self-reporting; report reconstructed and independently verified)_
_Iteration: 2_
