---
phase: 10-index-health-the-coverage-denominator
reviewed: 2026-09-13T00:00:00Z
depth: deep
files_reviewed: 16
files_reviewed_list:
  - internal/cli/index.go
  - internal/cli/index_test.go
  - internal/cli/init.go
  - internal/indexer/pipeline.go
  - internal/indexer/pipeline_test.go
  - internal/indexer/resolve.go
  - internal/indexer/resolve_test.go
  - internal/indexer/sync.go
  - internal/indexer/sync_coverage_test.go
  - internal/query/coverage.go
  - internal/query/coverage_test.go
  - internal/query/errors.go
  - internal/uiserver/handlers.go
  - internal/uiserver/coverage_test.go
  - internal/graphstore/pebble_store.go
  - internal/graphstore/store.go
findings:
  critical: 0
  warning: 1
  info: 3
  total: 4
status: issues_found
---

# Phase 10: Code Review Report

**Reviewed:** 2026-09-13T00:00:00Z
**Depth:** deep
**Files Reviewed:** 16
**Status:** issues_found

## Summary

Targeted deep re-review confirming the CR-01 closure from `83666cce` (`priorCoverageGeneration` in `internal/cli/index.go`, threaded as `indexer.Options.CoverageGenerationFloor` into `pipeline.go` → `Resolve` → `writeGraph`, which now stamps `max(priorGeneration, floor) + 1`). **CR-01 is genuinely closed for the case the finding described** — traced end to end, verified against the six specific checks the iteration context asked for:

**(a) Pre-wipe read closes before `RemoveAll`; no lock-file leak; but a real, if narrow, residual exists.** `priorCoverageGeneration` (`internal/cli/index.go:30-48`) opens read-only via `graphstore.Open` → `Snapshot()` → `GetMeta()`, with `defer store.Close()` and `defer r.Close()` executing in LIFO order (reader closed, then store) strictly before the function returns — `newIndexCmd` calls this synchronously and only then calls `os.RemoveAll(storeDir)` (`index.go:101` then `:107`), so the two never race and no lock file is left behind. Confirmed by a clean `TestIndexForceRebuildBumpsCoverageGeneration` run with no lock errors. **However**, every failure mode of that read — including `graphstore.ErrStoreLocked` (a live holder: a concurrent `codegraph ui`/MCP-server per-request open, an in-flight `Sync` flush, or another `index`/`sync` invocation) surviving the full `openLockRetryAttempts × openLockRetryBackoff` (~400ms) retry budget in `graphstore.Open` — is tolerated to `0`, indistinguishable from "store was never indexed." See WR-01 below: this is a real, if narrow, reintroduction of CR-01's own aliasing shape, and I am flagging it explicitly per the review brief rather than silently accepting the "tolerate to 0" comment's framing at face value.

**(b) `codegraph init` does not rebuild an existing store — confirmed no fourth wipe path.** `internal/cli/init.go:47-52` `os.Stat`s `codegraphDir` and returns `ErrAlreadyInitialized` immediately if it exists, before `storeDir` is ever touched — it never reaches `os.MkdirAll`/`indexer.Run` on that path. `init.go` has zero diff against `8d28c634` (confirmed via `git diff --stat`): it was not touched by the CR-01 fix and did not need to be, since it can never wipe an existing store. No floor-threading gap exists here because there is no wipe to float across.

**(c) No other `RemoveAll`+rebuild path exists.** `rg -n 'RemoveAll' internal/` finds exactly one production wipe-then-rebuild call site: `internal/cli/index.go:107`. `internal/cli/uninit.go:53` also calls `RemoveAll`, but only to delete `.codegraph/` wholesale as a terminal operation (no `indexer.Run` follows it, no store is rebuilt in the same invocation). Every other `RemoveAll` hit is test-fixture cleanup (`internal/mcp/server_test.go`, `internal/githooks/githooks_test.go`, `internal/daemon/daemon_test.go`, `internal/agents/manifest_test.go`). `internal/indexer/sync.go`'s `Sync` entry point never wipes `storeDir` — it always operates incrementally against an existing store. Closed.

**(d) `max(prior, floor)+1` is applied in `writeGraph` only; `sync.go`'s two commit sites are unaffected and correctly so.** `sync.go:220` and `:465` both do `newMeta.CoverageGeneration = meta.GetCoverageGeneration() + 1`, where `meta` is read once via `r0.GetMeta()` at the top of `Sync` (never through a wiped store — `Sync` has no wipe path per (c)) and never mutated in place before either write site reads it. This is unchanged by the CR-01 fix (`sync.go` is not in the fix's modified-files list and has zero incremental diff attributable to `83666cce`) and correctly stays a plain read-modify-write, since `Sync` never sees an empty store the way `codegraph index`'s wipe does. Closed.

**(e) The new CLI test genuinely exercises the wipe, not a mock.** `TestIndexForceRebuildBumpsCoverageGeneration` (`internal/cli/index_test.go`) calls `execCmd("init", dir)`, mints a real page token via `query.OpenAt` + `CoverageRows(PageSize:1)` against the post-init store (`staleToken := page1.NextPageToken`), then calls `execCmd("index", "--force", dir)` — the real CLI command tree, driving the actual `RemoveAll`+`MkdirAll`+`indexer.Run` sequence, not a simulated one. It asserts both `gen2 > gen1` (traced: reverting the fix, `writeGraph`'s own post-wipe Snapshot read returns `ErrNotFound` → `priorGeneration=0` both times → `gen1==gen2==1`, so the assertion is load-bearing) and that replaying `staleToken` — minted strictly before the wipe — against the rebuilt store returns `errors.Is(err, query.ErrAborted)`. Both `go test ./internal/cli/...` and the full targeted suite (`./internal/cli/... ./internal/indexer/... ./internal/query/... ./internal/uiserver/... ./internal/graphstore/...`) pass. Closed.

**(f) No proto/web drift.** `git diff faaab420..HEAD -- internal/uiproto internal/schema/graph.proto web/src` is empty (re-verified this pass) and `task proto:drift` passes clean (4/4 generated files byte-identical). `go build ./...` and `go vet ./...` are clean.

**Net verdict:** CR-01 as originally described (deterministic reset-to-1 on every `codegraph index` run, regardless of concurrency) is fixed. One narrower, previously-unstated residual survives the fix by construction (WR-01) and should be weighed by the team rather than silently accepted: a concurrent long-lived store holder at the exact moment of `codegraph index`'s pre-wipe read degrades the fix back to pre-CR-01 behavior for that one rebuild. IN-01/IN-02/IN-03 are unchanged from `10-REVIEW.iter4.md`, verified still present, and carried forward as Info.

## Warnings

### WR-01: `priorCoverageGeneration`'s blanket failure-to-0 fallback silently re-admits CR-01's exact aliasing bug when the store is genuinely locked, not merely absent

**File:** `internal/cli/index.go:30-48`
**Issue:** `priorCoverageGeneration` collapses three semantically distinct outcomes into the same return value, `0`:

1. The store has never been indexed (`graphstore.Open` succeeds against an empty/new directory, `GetMeta` returns `graphstore.ErrNotFound`) — `0` is exactly correct here.
2. The store directory doesn't exist at all — same as above, `0` is correct.
3. **The store exists, has real coverage history, and is transiently unreadable** — `graphstore.Open` fails with `graphstore.ErrStoreLocked` after exhausting its own bounded retry (`openLockRetryAttempts=5` × `openLockRetryBackoff=100ms`, ~400ms total, `internal/graphstore/pebble_store.go:67-81`), because another process or goroutine genuinely holds the Pebble directory lock at that moment — a live `codegraph ui`/MCP-server per-request `openEngine` call (`internal/uiserver/handlers.go:56`), an in-flight `indexer.Sync` debounced flush, or a second concurrent `index`/`sync` invocation. `GetMeta` returning any error other than `ErrNotFound` (e.g., a corrupted/partially-written Meta record) falls into the same bucket.

Case 3 is exactly CR-01's bug shape, reintroduced: the floor silently becomes `0`, `writeGraph`'s own post-wipe read of the now-empty target store also reads `0` (`resolve.go:763-775`), so the rebuild stamps `CoverageGeneration = 1` — identical to what a genuinely-fresh store would stamp, and identical to what the *previous* rebuild also stamped if it hit the same race or even just ran through the ordinary case-1/2 path. A client holding a pre-wipe page token whose embedded generation happens to equal the new store's `1` (the common case for any store that has not yet had many coverage-bearing commits) will have that token silently validated (`coverage.go:211`, `cursorGeneration != generation` is `false`) against a store built from a completely different graph.

This is a materially narrower window than the original CR-01 (it requires an active lock holder to survive the full ~400ms retry budget at the exact moment `codegraph index` runs its pre-wipe read — an ordinary debounced-flush collision is already absorbed by that retry), so I am not classifying it as a Critical regression of the fixed behavior. But it is a real, reachable failure mode in this project's own stated direction ("optimized for concurrent access," a daemon/watcher that is default-on, an MCP/UI server process that opens the store per request) and it is currently silent: nothing distinguishes "confirmed no prior store" from "store exists but is locked/corrupt" in the log output, the CLI's own diagnostics, or a test. There is no test anywhere in the tree that exercises `priorCoverageGeneration`'s lock-held or corrupted-Meta branches (`rg -n priorCoverageGeneration internal/` finds only the three internal/cli/index.go references — no `_test.go` hit).

**Fix:** Distinguish "confirmed absent" (`errors.Is(err, graphstore.ErrNotFound)` from `GetMeta`, or the directory genuinely does not exist) from "present but unreadable" (`errors.Is(err, graphstore.ErrStoreLocked)`, or any other `Open`/`Snapshot`/`GetMeta` error). For the latter, either:
- surface a hard error from `newIndexCmd` before ever calling `RemoveAll` (forcing the operator to retry once the holder releases, which is consistent with this codebase's "never silently block/degrade past a lock" philosophy already documented in `pebble_store.go`'s own comments), or
- at minimum, emit a diagnostic/warning line when the fallback fires for a reason other than "genuinely never indexed," so an operator investigating a coverage-page-token bug report has a signal to look at.
Either way, add a regression test that holds the store open (mirroring `internal/graphstore/open_lock_test.go`'s `TestOpenSecondOpenInProcessReturnsErrStoreLocked` pattern) across a `codegraph index --force` run and asserts the chosen behavior (hard failure, or a floor that is provably still safe) rather than leaving this path silently untested.

## Info

### IN-01: `coverageExtractionDetail`'s absolute-path scrub is a plain substring replace, not anchored to a path boundary

**File:** `internal/query/coverage.go:337-343`
**Issue:** Unchanged from prior reviews (verified still present at the cited lines). `strings.ReplaceAll(detail, repoRoot, ".")` replaces every literal occurrence of `repoRoot`, not just a leading-path occurrence, and depends on `e.repoRoot`'s capitalization/symlink-resolution matching whatever was embedded in the underlying parser error verbatim (e.g. a `/tmp` vs `/private/tmp` class of mismatch on macOS). Carried forward as low-severity: pre-existing pattern, not introduced or touched by this fix.
**Fix:** No action required unless a broader repoRoot-redaction audit is already planned.

### IN-02: `unsupportedExtensionDetail`/exclusion `detail` strings are unbounded on the write path

**File:** `internal/indexer/discoverexclusion.go:78-100`
**Issue:** Unchanged from prior reviews (not part of this fix's diff, contents re-verified identical). No explicit byte bound is applied at write time the way `coverageDetailMaxBytes` bounds the read-time `Detail` for extraction failures. Low-severity: filesystem path-component limits already bound this in practice.
**Fix:** No action required now; noted for completeness since coverage rows are exposed over the wire to a browser.

### IN-03: `fetchAllCoverageRows`'s second (retry) attempt discards any rows it collected before aborting a second time

**File:** `web/src/lib/health-view.ts:335-365`
**Issue:** Unchanged from prior reviews (not part of this fix's diff, contents re-verified identical — confirmed `web/src` has zero diff against `faaab420`). When the retried walk itself throws `Code.Aborted` partway through, the final `.catch` returns a hardcoded `{ known: true, rows: [], incomplete: true }` rather than whatever partial `rows` the second attempt had already accumulated. Not a correctness bug (`incomplete: true` already tells the UI not to trust the list), just a minor, easily-avoidable loss of otherwise-good partial data.
**Fix:** Optional: thread the partially-collected rows out of the second `attempt()` call. Not required before shipping.

## Notes for threat-model reconciliation (`10-SECURITY.md`)

`10-SECURITY.md` predates both the CR-01 and WR-01 findings across this review's iterations. For the reconciliation pass, I recommend:

- **Add** a threat row for WR-01 above: "A live store holder (UI/MCP server per-request open, in-flight Sync flush, or a second concurrent index/sync) surviving `codegraph index`'s pre-wipe generation-floor read past its ~400ms retry budget silently resets `CoverageGeneration` to 1, reintroducing page-token aliasing across a rebuild." Suggested severity: Low/Medium (narrow race window, data-consistency impact only — a client may transiently see rows spliced from two unrelated index generations with no error signal — no confidentiality/integrity-of-storage impact, no code execution, no auth bypass).
- **No change needed** for the original CR-01 threat row (if one exists): the deterministic, always-fires shape it described is confirmed fixed by `83666cce` per items (a)-(f) above.
- I found no unmitigated Critical/High-severity threat in this file set at ASVS L1 scope: no injection, no hardcoded secrets, no unsafe deserialization, no path traversal in the reviewed files (the coverage detail-scrubbing paths in IN-01/IN-02 are informational-quality issues, not exploitable disclosure — `coverageDetailMaxBytes` and the repoRoot substitution already prevent the primary host-path leak scenario `T-10-05` documents).

---

_Reviewed: 2026-09-13T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
