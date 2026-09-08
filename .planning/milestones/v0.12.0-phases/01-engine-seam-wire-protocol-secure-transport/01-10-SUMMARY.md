---
phase: 01-engine-seam-wire-protocol-secure-transport
plan: 10
subsystem: api
tags: [protobuf, connect-go, uiserver, truncation, utf-8, rpc-05]

# Dependency graph
requires:
  - phase: 01-engine-seam-wire-protocol-secure-transport
    provides: "Plan 01-09's nine-RPC UIService surface (GetNodeDetail, Explore) and its known-number field fixture"
provides:
  - "internal/textutil.TruncateOnRuneBoundary — the one shared rune-boundary cut consumed by internal/mcp and internal/uiserver"
  - "internal/uiserver.truncateSource / countLines — two-tier (line-then-byte) source truncation with exact totals in both units"
  - "uiv1.SourceBlob attached to GetNodeDetailResponse, NodeDefinition and ExploreGroup, wired into every source-producing handler path"
  - "connect.WithSendMaxBytes/WithReadMaxBytes transport backstop on the UIService handler, sized above the multi-def aggregate worst case"
affects: [03-source-viewer, 04-index-health, 05-ui-shell, 06-ui-polish]

# Actuals (#2632)
actuals:
  tokens: 22671
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Shared rune-boundary cut lives in internal/textutil, imported by internal/mcp and internal/uiserver — never duplicated"
    - "Two-tier truncation (line cap primary, byte cap secondary, cut on a rune boundary) with both totals and both returned counts reported so a client never has to recompute or guess which cap fired"
    - "Transport backstop (connect.WithSendMaxBytes/WithReadMaxBytes) sized with headroom above the AGGREGATE worst case (uiMultiDefCap * sourceByteCap), not merely above one blob's cap"
    - "Known-number protobuf field fixture extended via a length-chain constant (uiProtoFieldFixtureLenAtPlan0110 = uiProtoFieldFixtureLenAtPlan0109 + 9) so cross-wave drift breaks the compiler, never just a prose SUMMARY"

key-files:
  created:
    - internal/textutil/truncate.go
    - internal/textutil/truncate_test.go
    - internal/uiserver/truncate.go
    - internal/uiserver/truncate_test.go
    - internal/uiserver/sourceblob_test.go
  modified:
    - internal/mcp/session_line.go
    - internal/uiproto/uiv1/ui.proto
    - internal/uiproto/uiv1/ui.pb.go
    - internal/uiproto/uiv1/uiv1connect/ui.connect.go
    - internal/uiserver/handlers.go
    - internal/uiserver/readonly_test.go
    - internal/uiserver/server.go

key-decisions:
  - "Checkpoint (maintainer-ruled, recorded verbatim below) locked SourceBlob's six fields (content, truncated, total_lines, total_bytes, returned_lines, returned_bytes), the three attachment points at GetNodeDetailResponse=9/NodeDefinition=5/ExploreGroup=4, bytes content with a SCOPED UTF-8 guarantee, the newline-plus-unterminated-final-line counting rule, truncation-never-an-error, and reserved 50-59 on SourceBlob only."
  - "sourceLineCap=4096 and sourceByteCap=262144 (256 KiB) chosen deliberately to avoid colliding, as bare numeric literals, with an unrelated pre-existing '5000' literal in internal/mcp/skill_claims_drift_test.go within the acceptance criterion's required search scope (internal/uiserver/, internal/textutil/, internal/mcp/) — the plan's own worked example used 5000, which would have made the 'exactly 1' numeric-literal gate fail for a reason unrelated to this plan's code."
  - "transportSendMaxBytes=16MiB sized above the multi-definition AGGREGATE worst case (uiMultiDefCap candidates each up to sourceByteCap, ~5MiB), not merely above a single blob's sourceByteCap — the plan's own D-13 rationale is about one blob vs the backstop, but GetNodeDetailResponse's multi-def mode can legitimately carry up to 20 SourceBlobs in one response."
  - "nodeDetailToProto's signature gained an *query.Engine parameter (previously took only query.NodeDetail) so the single-definition mode's new (*Engine).SourceFor wire-layer read has access to the already-open Engine inside the same withEngine call, rather than opening a second one."

requirements-completed: [RPC-05]

coverage:
  - id: D1
    description: "Shared rune-boundary truncation helper (internal/textutil.TruncateOnRuneBoundary), consumed by both internal/mcp and internal/uiserver, replacing the duplicated unexported copy"
    requirement: "RPC-05"
    verification:
      - kind: unit
        ref: "internal/textutil/truncate_test.go#TestTruncateOnRuneBoundary"
        status: pass
      - kind: unit
        ref: "internal/mcp/... (go test -race ./internal/mcp/...)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Two-tier source truncation (line cap primary, byte cap secondary) with exact totals in both units, never signalled as an error"
    requirement: "RPC-05"
    verification:
      - kind: unit
        ref: "internal/uiserver/truncate_test.go#TestTruncateSourceUnderBothCaps,TestTruncateSourceExceedsLineCap,TestTruncateSourceExceedsByteCap,TestTruncateSourceLineCapBoundary,TestTruncateSourceNeverSplitsARune,TestTruncateSourceEmptyInput,TestTruncateSourceTotalsAreExact,TestCountLinesSemantics"
        status: pass
    human_judgment: false
  - id: D3
    description: "SourceBlob wired end-to-end at all three attachment points (GetNodeDetail single-def/file mode, multi-def candidates, Explore groups) through a real Connect client"
    requirement: "RPC-05"
    verification:
      - kind: integration
        ref: "internal/uiserver/sourceblob_test.go#TestUIServiceSourceBlobTruncatesAnOversizedDefinitionSource,TestUIServiceSourceBlobOnEveryAttachmentPoint,TestUIServiceSourceBlobGroupSourceMatchesItsGroupPath,TestUIServiceSourceBlobAtTheCapClearsTheTransportBackstop,TestUIServiceSourceBlobFileModeUsesTheSameTruncationPath"
        status: pass
    human_judgment: false
  - id: D4
    description: "Transport backstop (WithSendMaxBytes/WithReadMaxBytes) mounted on the UIService handler, strictly above the application byte cap"
    requirement: "RPC-05"
    verification:
      - kind: unit
        ref: "internal/uiserver/truncate_test.go#TestTransportBackstopSitsAboveApplicationCap"
        status: pass
      - kind: integration
        ref: "internal/uiserver/sourceblob_test.go#TestUIServiceSourceBlobAtTheCapClearsTheTransportBackstop"
        status: pass
    human_judgment: false
  - id: D5
    description: "01-09's known-number field fixture extended by exactly 9 entries, every prior entry still resolving unchanged"
    requirement: "RPC-01"
    verification:
      - kind: unit
        ref: "internal/uiserver/readonly_test.go#TestUIProtoFieldNumbersAreStableAndUnique"
        status: pass
    human_judgment: false

duration: unrecorded
completed: 2026-08-23
status: complete
---

# Phase 1 Plan 10: Bounded Source Blobs Summary

**Two-tier (line-then-byte, rune-safe) source truncation shared via a new internal/textutil package, wired into every GetNodeDetail/Explore source path behind a transport backstop sized above the multi-definition aggregate worst case.**

## Performance

- **Duration:** Not recorded (start timestamp was not captured at agent launch)
- **Completed:** 2026-08-23T17:09:07Z
- **Tasks:** 2 (plus one pre-ruled checkpoint)
- **Files modified:** 12

## Checkpoint Ruling (recorded verbatim, pre-approved by the maintainer)

Task 1's `type="checkpoint:decision" gate="blocking"` was **APPROVED AS SPECIFIED** by the maintainer before this execution began (all five sub-decisions), per the orchestrator's explicit instruction not to pause:

1. `SourceBlob` carries six fields numbered 1-6: content, truncated flag, true total in lines, true total in bytes, returned count in lines, returned count in bytes.
2. Three attachment points at the next free number in each message: `GetNodeDetailResponse` = 9, `NodeDefinition` = 5, `ExploreGroup` = 4.
3. Content is `bytes`, with the UTF-8 guarantee SCOPED — valid input stays valid because the cut never splits a rune — never a universal claim.
4. Line counting: newlines, plus one when non-empty and not newline-terminated (`"" → 0`, `"\n" → 1`, `"a\nb\n" → 2`, `"a\nb" → 2`), one named helper referenced by both the total and the returned count.
5. Truncation is never an error.

**Plus the maintainer's one addition beyond the plan text:** `reserved 50 to 59;` added to `SourceBlob` only (mirroring `Node`/`Edge`/`Meta`'s identical clauses), documented in `SourceBlob`'s doc comment as reserved for future payload metadata (compression, encoding hints, syntax-highlight spans).

All numbers spent by this ruling (SourceBlob's 1-6, the three attachment numbers, and the reserved band) are one-way under D-02a and are now shipped.

## Accomplishments

- `internal/textutil.TruncateOnRuneBoundary` is the single rune-safe cut in the tree, moved unchanged from `internal/mcp/session_line.go`'s unexported `truncateOnRuneBoundary` and now consumed by both `internal/mcp` and `internal/uiserver` (T-01-37).
- `internal/uiserver/truncate.go` implements `countLines` (the locked newline-plus-unterminated-final-line rule, one implementation for both totals and returned counts) and `truncateSource` (line cap primary at `sourceLineCap`=4096, byte cap secondary at `sourceByteCap`=262144 bytes, cut on a rune boundary), plus the transport constants `transportSendMaxBytes`=16 MiB and `transportReadMaxBytes`=1 MiB.
- `ui.proto` gained `SourceBlob` (six fields, `reserved 50 to 59`) attached to `GetNodeDetailResponse.source`=9, `NodeDefinition.source`=5, `ExploreGroup.source`=4 — the exact numbers plan 01-09 recorded as intent. Regenerated via `task proto:gen`; `task proto:drift` confirms zero drift.
- `internal/uiserver/readonly_test.go`'s known-number fixture extended by exactly 9 entries via `uiProtoFieldFixtureLenAtPlan0110 = uiProtoFieldFixtureLenAtPlan0109 + 9`; every prior entry from 01-09 still resolves unchanged.
- `handlers.go` wires `truncateSource` into all four source-producing paths (file mode, single-def mode via a new `(*Engine).SourceFor` wire-layer read, multi-def per-candidate, Explore per-group by path key) through one shared `sourceBlobToProto` mapper.
- `server.go`'s `Listen` mounts `connect.WithSendMaxBytes`/`WithReadMaxBytes` on the UIService handler (D-13) — connect-go's own default is unlimited on both sides.

## Task Commits

1. **Task 1: One shared rune-boundary helper, then two-tier truncation with exact totals** - `2119572` (feat)
2. **Task 2: Attach SourceBlob to the three response points and mount the transport backstop** - `c58de94` (feat)

**Plan metadata:** (this commit, docs: complete plan)

## Files Created/Modified

- `internal/textutil/truncate.go` - `TruncateOnRuneBoundary`, moved from internal/mcp
- `internal/textutil/truncate_test.go` - `TestTruncateOnRuneBoundary`
- `internal/mcp/session_line.go` - now calls `textutil.TruncateOnRuneBoundary`; the old unexported copy is gone
- `internal/uiserver/truncate.go` - `countLines`, `firstNLines`, `truncatedSource`, `truncateSource`, and the four named caps
- `internal/uiserver/truncate_test.go` - the nine Task-1 test functions/subtests
- `internal/uiproto/uiv1/ui.proto` - `SourceBlob` message plus the three attachment fields
- `internal/uiproto/uiv1/ui.pb.go`, `internal/uiproto/uiv1/uiv1connect/ui.connect.go` - regenerated
- `internal/uiserver/handlers.go` - `sourceBlobToProto`, truncation wired into every source path, `nodeDetailToProto` now takes `*query.Engine`
- `internal/uiserver/readonly_test.go` - fixture extended by 9 entries
- `internal/uiserver/server.go` - transport backstop options on `NewUIServiceHandler`
- `internal/uiserver/sourceblob_test.go` - the five Task-2 end-to-end test functions

## Decisions Made

- **Numeric cap values chosen to avoid an unrelated literal collision.** The plan's own worked examples used `sourceLineCap=5000`, but the acceptance criterion's numeric-literal gate (`rg` over `internal/uiserver/ internal/textutil/ internal/mcp/`, requiring the chosen value's literal to appear EXACTLY once) would have failed against that value because `internal/mcp/skill_claims_drift_test.go:447` already contains an unrelated `5000` inside that same search scope. Picked `4096` (line cap) and `262144` = 256 KiB (byte cap) instead — both verified collision-free in the scoped search before committing. Documented in `truncate.go`'s doc comments with the corpora-observation caveat (no per-file line-count distribution exists in `corpora/observations.json`, only aggregate node/edge counts, so the values are sized against common editor "large file" conventions rather than a measured percentile).
- **`transportSendMaxBytes` sized above the aggregate, not the single-blob, worst case.** GetNodeDetail's multi-definition mode can return up to `uiMultiDefCap` (20) gathered candidates in one response, each carrying its own `SourceBlob` up to `sourceByteCap` — an aggregate worst case around 5 MiB before overhead. 16 MiB leaves comfortable headroom above that, not merely above `sourceByteCap` itself.
- **`nodeDetailToProto` gained an `*query.Engine` parameter.** The single-definition mode's source (`SourceBlob` on `GetNodeDetailResponse`) requires a NEW read (`(*Engine).SourceFor`) that `DefinitionDetail` for that mode never populates (only the multi-def path's `DefinitionDetail.Source` is populated, by `buildMultiDefDetail`'s existing per-candidate closure). Threading the already-open `Engine` through avoids opening a second one inside the same `withEngine` call.
- **Task 2's `<verify>` proto-idempotency clause (`task proto:gen && git status --porcelain internal/uiproto/` empty) was run AFTER the task's single commit, not before.** That clause only produces empty output relative to a committed baseline; run pre-commit against 01-09's committed tree it would show this plan's own new (uncommitted) content as "drift." Ran it as a POST-commit confirmation instead (both `task proto:gen` idempotency and `task proto:drift`'s scratch-dir comparison passed cleanly against the committed tree), which is the substantively equivalent and non-vacuous form of the same check.

## Deviations from Plan

### Auto-fixed / Adjusted Issues

**1. [Not a Rule 1-4 deviation — gate-derivation headroom] `TestTruncateSourceTotalsAreExact` carries two `t.Run` subtests instead of the plan's literal zero**
- **Found during:** Task 1, writing `truncate_test.go`
- **Issue:** The plan's floor derivation for Task 1 explicitly enumerated `TestTruncateSourceTotalsAreExact` as contributing exactly 1 PASS line (no subtests). Implementing it, an untruncated-input case and a truncated-input case were both worth asserting distinctly, so two `t.Run` subtests (`untruncated`, `truncated`) were added.
- **Effect:** Task 1's total PASS-line count is 19 rather than the plan's derived 17, both comfortably above the stated floor of 12. The `<verify>` gate (status 0 AND count >= 12) still passes; no floor derivation is invalidated, since the derivation is a MINIMUM, not an exact-match requirement.
- **Files modified:** `internal/uiserver/truncate_test.go`
- **Committed in:** `2119572` (Task 1 commit)

**2. [Numeric value substitution, documented above under Decisions Made] `sourceLineCap`/`sourceByteCap` use different numbers than the plan's worked examples**
- **Found during:** Task 1, running the plan's own numeric-literal acceptance-criteria check
- **Issue:** The plan's worked examples suggested values like `5000`; verifying the plan's OWN acceptance criterion (exactly-one-occurrence numeric-literal gate) against that value failed due to an unrelated pre-existing literal in `internal/mcp/skill_claims_drift_test.go`, inside the criterion's own mandated search scope.
- **Fix:** Chose `4096` and `262144` instead, verified collision-free by running the exact criterion command before finalizing.
- **Files modified:** `internal/uiserver/truncate.go`
- **Committed in:** `2119572` (Task 1 commit)

---

**Total deviations:** 2 (both documentation/value adjustments within the plan's own stated tolerances; no scope creep, no architectural changes).
**Impact on plan:** None on behavior or requirements coverage — both are calibration details the plan explicitly left to execution-time discretion (numeric cap values) or a floor-vs-exact-count distinction the plan's own gate conventions already treat as a minimum.

## Issues Encountered

- The plan's Task 2 `<verify>` block's first clause (`task proto:gen && test -d internal/uiproto && test -z "$(git status --porcelain internal/uiproto/)"`) is only meaningful when run against a COMMITTED baseline (it is structurally identical to `task proto:drift`'s in-place, non-scratch-dir form). Run before Task 2's commit it would show this plan's own legitimate new content as "drift". Resolved by running the sourceblob/inherited test legs pre-commit (both passed), committing Task 2 atomically, then confirming the proto-idempotency clause AND `task proto:drift` post-commit (both passed cleanly). See Decisions Made above.
- `internal/query/explore.go` around line 236 does not contain a comment citing "T-01-25" as the plan's `read_first` for Task 2 stated (searched the whole repo scope; no `T-01-25` reference exists anywhere in `internal/`). This appears to be a stale/incorrect cross-reference in the plan text — proceeded using `internal/query/detail.go`'s own `ExploreResult.Sources` doc comment (which DOES document the per-group-path keying this plan's `ExploreGroup.source` field relies on) as the equivalent in-repo precedent. No functional impact; noted here per the plan's own instruction to document discrepancies between plan prose and actual source.

## User Setup Required

None — no external service configuration required.

## Known Stubs

None.

## Next Phase Readiness

- RPC-05 is fully implemented and verified end-to-end: every source blob leaving the process (file mode, single-def mode, multi-def candidates, Explore groups) is bounded in two units, cut on a rune boundary, and explicitly marked truncated with true totals — never as an error.
- `internal/textutil` is now available as a shared home for future security-relevant string/byte helpers; `internal/uiserver`'s `sourceLineCap`/`sourceByteCap`/`transportSendMaxBytes`/`transportReadMaxBytes` are all named, single-source-of-truth constants Phase 3's source viewer can reference directly (e.g. to render "showing first N of M lines/bytes" using the six `SourceBlob` fields).
- Plan 01-11 is the next and final extender of `internal/uiserver/readonly_test.go`'s known-number fixture (`GetStatusResponse.store_exists = 8`, `.indexing_in_progress = 9`) and owns SRV-04's degrade path — no blockers from this plan.
- Full plan-level `<verification>` block confirmed: `go build ./...` clean; `go test -race -count=1 ./internal/uiserver/... ./internal/textutil/... ./internal/mcp/...` passes; `task proto:drift` reports 3/3 unchanged; `task test:unit` green across all packages; `task test:golden` reports `attempted=26 completed=26 matched=26`; `go vet` clean on all three touched package trees.

## Self-Check: PASSED

All created files verified present on disk (`internal/textutil/truncate.go`, `internal/textutil/truncate_test.go`, `internal/uiserver/truncate.go`, `internal/uiserver/truncate_test.go`, `internal/uiserver/sourceblob_test.go`, this SUMMARY.md); both task commit hashes (`2119572`, `c58de94`) verified present in `git log --oneline --all`.

---
*Phase: 01-engine-seam-wire-protocol-secure-transport*
*Completed: 2026-08-23*
