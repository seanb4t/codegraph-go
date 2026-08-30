---
phase: 04-query-workbench-index-health
plan: 03
subsystem: uiserver
tags: [connectrpc, protobuf, tdd, worktree-detection, health-endpoint]

requires:
  - phase: 01-ui-service-foundation (plan 09)
    provides: "the nine-method read-only UIService surface, wantUIServiceMethods/mutatingVerbs's complementary-guard shape, uiProtoFieldNumbers' known-field-number fixture and its extension convention"
  - phase: 03-browse-inspect-navigation (plan 05)
    provides: "GetPermalink's own human-checkpoint wire-shape freeze precedent (D-06), the pattern this plan's Task 1 checkpoint mirrors"
provides:
  - "internal/uiproto/uiv1/ui.proto — GetHealth, the eleventh read-only UIService rpc, plus GetHealthRequest/GetHealthResponse/WorktreeMismatch/PendingChanges/IndexHealth messages"
  - "internal/uiserver/handlers.go — (*uiService).GetHealth, healthToProto, worktreeMismatchToProto"
  - "internal/uiserver/health_test.go — GetHealth's Go test coverage, including the uiserver-local worktree-mismatch fixture"
affects: [04-05]

actuals:
  tokens: 23221
  tasks: 3
  commits: 2

tech-stack:
  added: []
  patterns:
    - "A second, richer per-rpc health projection kept SEPARATE from the cheap per-navigation status rpc (D-01): GetStatusResponse stays a nine/ten-scalar probe; GetHealthResponse carries the full StatusResult (maps, index-health, worktree-mismatch) behind its own rpc, so no per-navigation caller pays for data it doesn't render."
    - "A test-only fixture that CANNOT be imported across packages (query's unexported worktreeMismatchFixture) is REPLICATED locally with a comment naming what it mirrors and why, never exported from its origin package to satisfy one cross-package caller."
    - "A live filesystem byte-sum (DbSizeBytes) compared between two independent scans of the SAME live Pebble store must assert a sane bound (> 0), never byte-exact equality — two scans can observe a few bytes of WAL/compaction drift between them, mirroring internal/query/files_status_test.go:518's existing convention for the identical field."

key-files:
  created:
    - internal/uiserver/health_test.go
  modified:
    - internal/uiproto/uiv1/ui.proto
    - internal/uiproto/uiv1/ui.pb.go
    - internal/uiproto/uiv1/uiv1connect/ui.connect.go
    - web/src/lib/gen/ui_pb.ts
    - internal/uiserver/handlers.go
    - internal/uiserver/readonly_test.go

key-decisions:
  - "Task 1 checkpoint (human, gate=blocking-human, reversibility=one-way): maintainer verbatim answer \"Approve as proposed\" against the exact 16-field GetHealthResponse plus WorktreeMismatch(2)/PendingChanges(3)/IndexHealth(6)/GetHealthRequest(1) shape presented at the checkpoint. All five sub-decisions answered explicitly: (1) worktree_mismatch's host-absolute paths are the one scoped exception to the project_path/index_path privacy stance, extended to the browser surface; (2) commit_sha is deliberately duplicated onto GetHealthResponse, validated through the identical schema.IndexedCommitSHA/schema.IsCommitSHA gate GetStatus uses, never re-derived; (3) message IndexHealth / field index_health stand; (4) the rpc is GetHealth; (5) PendingChanges/IndexHealth keep their unprefixed names, disambiguated in handlers.go via package qualifiers (query.PendingChanges vs uiv1.PendingChanges), the same discipline query.Location/uiv1.Location already establish in that file. The Engine.Status pre-existing per-navigation git-subprocess-cost finding was acknowledged as recorded-and-flagged, not fixed — see the finding below."
  - "Codegen generated exactly the frozen shape with zero amendment: task proto:gen and task proto:drift both ran clean on the first attempt against the approved shape, and proto:drift reported 'compared 4 generated files' both before and after Task 3's changes — the floor was checked, not assumed."
  - "GetHealth follows the ORDINARY withEngine handler shape (Callers/Callees/Files' convention), not GetStatus's openEngine-direct degrade-and-answer shape. eng.Status(ctx) is called exactly once inside the withEngine closure — it already populates WorktreeMismatch via Engine.WorktreeMismatch's once-per-Engine mismatchOnce latch (status.go:359) — so no second explicit detection call exists in the handler."

requirements-completed: [HLT-01, HLT-02, HLT-03]

coverage:
  - id: D0
    description: "GetHealthResponse's field set and numbering, and its five design sub-decisions, were frozen by explicit maintainer approval before any codegen ran (D-02a one-way door)"
    verification:
      - kind: human
        ref: "Task 1 checkpoint (gate=blocking-human): maintainer reply \"Approve as proposed\" with all five sub-decisions answered individually, recorded verbatim in this SUMMARY's key-decisions"
        status: pass
    human_judgment: true
    rationale: "Proto field numbers are additive-only and unreclaimable once shipped — a wrong shape can only be `reserved` and worked around forever, so freezing it is inherently a human, not an automatable, decision."
  - id: D1
    description: "UIService exposes exactly eleven read-only methods, and the eleventh is GetHealth"
    requirement: HLT-01
    verification:
      - kind: unit
        ref: "internal/uiserver/readonly_test.go#TestUIServiceMethodSetIsExactlyTheReadSet — PASS, inspected 11 methods, exact set match"
        status: pass
      - kind: unit
        ref: "internal/uiserver/readonly_test.go#TestUIServiceDeclaresNoMutatingMethod — PASS, inspected 11 UIServiceHandler method names against 19 mutating verbs, zero matches"
        status: pass
    human_judgment: false
  - id: D2
    description: "A GetHealth call against an initialized repository returns per-language file counts, per-kind node counts, per-kind edge counts, the schema version, the db size, the pending-change tallies and the index-health block — every value HLT-01 names, compared entry-for-entry against an independently-computed eng.Status(ctx) call for the same fixture"
    requirement: HLT-02
    verification:
      - kind: unit
        ref: "internal/uiserver/health_test.go#TestGetHealthProjectsStatusResult — PASS (stable across 3 repeated runs)"
        status: pass
    human_judgment: false
  - id: D3
    description: "A GetHealth call from a worktree whose index belongs to a different working tree returns a populated worktree_mismatch naming both roots; from an in-tree repository it returns nil — both directions asserted, not only the positive case"
    requirement: HLT-03
    verification:
      - kind: unit
        ref: "internal/uiserver/health_test.go#TestGetHealthWorktreeMismatchPopulated — PASS"
        status: pass
      - kind: unit
        ref: "internal/uiserver/health_test.go#TestGetHealthWorktreeMismatchNilInTree — PASS"
        status: pass
    human_judgment: false
  - id: D4
    description: "GetStatusResponse's field set is byte-identical to what it was before this plan"
    verification:
      - kind: other
        ref: "git diff internal/uiproto/uiv1/ui.proto shows ZERO removed lines across both commits (only additions inside service UIService and only new messages appended); statusToProto is unchanged"
        status: pass
    human_judgment: false
  - id: D5
    description: "Regenerating both protobuf surfaces still produces no diff and the drift guard still reports the number of files it compared"
    verification:
      - kind: other
        ref: "task proto:drift -> 'proto:drift: compared 4 generated files' / 'all 4 generated files byte-identical to the pinned toolchain's regeneration', run clean both immediately after Task 2's codegen and again after Task 3's handlers.go/readonly_test.go changes"
        status: pass
    human_judgment: false
  - id: D6
    description: "A malformed or attacker-planted commit_sha in a Meta record degrades to empty rather than leaving the server (T-04-11)"
    verification:
      - kind: unit
        ref: "internal/uiserver/health_test.go#TestGetHealthCommitShaValidated — PASS"
        status: pass
    human_judgment: false
  - id: D7
    description: "GetHealth opens the engine through the single withEngine/openEngine seam exactly once per call (SRV-04)"
    verification:
      - kind: unit
        ref: "internal/uiserver/health_test.go#TestGetHealthOpensEngineExactlyOnce — PASS"
        status: pass
    human_judgment: false
  - id: D8
    description: "GetHealth does NOT copy GetStatus's degrade-and-answer shape: a store locked past graphstore.Open's retry budget returns an ERROR (CodeUnavailable), never a successful degraded response"
    verification:
      - kind: unit
        ref: "internal/uiserver/health_test.go#TestGetHealthDoesNotDegrade — PASS"
        status: pass
      - kind: other
        ref: "rg -n 'func \\(s \\*uiService\\) GetHealth' -A 30 internal/uiserver/handlers.go | rg -c 'degradedStatus' -> 0, positive-controlled by rg -c 'degradedStatus' internal/uiserver/handlers.go -> 4 (GetStatus's own use)"
        status: pass
    human_judgment: false
  - id: D9
    description: "The full unit suite is green with the new rpc wired in, including the pre-existing known-field-number fixture extended for the five new messages' 28 fields"
    verification:
      - kind: unit
        ref: "GOTOOLCHAIN=go1.26.5 task test:unit — all packages ok, including internal/uiserver"
        status: pass
      - kind: unit
        ref: "internal/uiserver/readonly_test.go#TestUIProtoFieldNumbersAreStableAndUnique — PASS, inspected 34 messages and 142 fields"
        status: pass
      - kind: other
        ref: "cd web && pnpm check — 991 files, 0 errors, 0 warnings"
        status: pass

duration: 70min
completed: 2026-08-30
status: complete
---

# Phase 4 Plan 3: GetHealth — the eleventh read-only rpc Summary

**A new additive, read-only `GetHealth` rpc puts every value `internal/query.StatusResult` already computes — per-language/per-kind maps, pending-change tallies, the index-health block, and worktree-mismatch detection in both directions — on the wire, without touching `GetStatusResponse`'s existing nine scalars.**

## What Was Built

`GetHealth` is `UIService`'s eleventh method, frozen at a maintainer-approved wire-shape checkpoint (Task 1) before any codegen ran, then implemented (Tasks 2-3) as an ordinary `withEngine`-wrapped handler with a named `healthToProto` mapper — following `Callers`/`Callees`/`Files`'s convention, explicitly NOT `GetStatus`'s single-exception degrade-and-answer shape. `GetStatusResponse` stays byte-identical: the split keeps the client's per-navigation status gate cheap (D-01) while giving the new health view a single rpc for every value it needs (HLT-01/02/03).

Five new proto messages carry the shape: `GetHealthRequest` (an unread forward-compat `path` field, mirroring `GetStatusRequest`'s own), `WorktreeMismatch` (the wire projection of `gitmeta.Mismatch`, the one scoped exception to this service's path-privacy stance), `PendingChanges` and `IndexHealth` (direct projections of their `internal/query` counterparts, kept unprefixed per the checkpoint's sub-decision 5), and `GetHealthResponse` itself (16 fields, numbered 1-16 exactly as approved).

## Deviations from Plan

None of the plan's specified behavior or scope changed. Four implementation-time issues were found and fixed during execution (Rules 1 and 3), all documented below for transparency, plus two verification-command false positives that were investigated and confirmed benign rather than fixed.

### Issues Encountered (auto-fixed)

**1. [Rule 3 — blocking build issue] `uiv1connect.UIServiceHandler` requiring `GetHealth` broke the whole `internal/uiserver` package's compilation the moment Task 2's codegen ran, before Task 3's handler existed.**
- **Found during:** Task 2, step (e) — re-running the readonly guards after codegen.
- **Root cause:** `*uiService` has no forward-compatibility embed; Go requires it to implement every method of `uiv1connect.UIServiceHandler` for the package to build at all. Task 2's own file list (per the plan) did not include `handlers.go`, but the interface it regenerates unconditionally does.
- **Fix:** added a minimal `GetHealth` placeholder to `handlers.go` in Task 2's commit, returning `connect.CodeUnimplemented` rather than a fabricated response — deliberately chosen so Task 3's `health_test.go` RED phase would observe real, honest failures against it (confirmed: all 6 new tests failed with `unimplemented` errors before Task 3's real implementation existed). Task 3 replaced (not extended) the placeholder with the real handler.
- **Files modified:** `internal/uiserver/handlers.go` (both commits).
- **Verification:** both readonly guards GREEN after the placeholder; `TestGetHealth*` RED (6 FAIL, 0 PASS) against the placeholder before Task 3; GREEN (6 PASS) after.

**2. [Rule 3 — blocking test-suite failure] `internal/uiserver/readonly_test.go`'s pre-existing `TestUIProtoFieldNumbersAreStableAndUnique` known-field-number fixture required extension for the 28 fields the five new messages declare.**
- **Found during:** Task 3, running `task test:unit` for the plan's overall `<verification>` — not named in either task's file list or read_first.
- **Symptom:** `descriptor declares field GetHealthResponse.pending_changes, but no fixture entry covers it` — this guard asserts the fixture covers EVERY field of EVERY message in the generated descriptor, and the plan's read_first for Task 2 did not surface this second load-bearing fixture in the same file as `wantUIServiceMethods`/`mutatingVerbs`.
- **Fix:** added `uiProtoFieldFixtureLenAtPlan0403 = uiProtoFieldFixtureLenAtPlan0305 + 28` (following the file's own established chained-extension convention from 01-10/01-11/03-05) and appended all 28 new `(message, field, number)` entries; updated the length assertion to the new constant.
- **Files modified:** `internal/uiserver/readonly_test.go`.
- **Verification:** `TestUIProtoFieldNumbersAreStableAndUnique` PASS, inspected 34 messages and 142 fields; `task test:unit` green end to end afterward.

**3. [Rule 1 — bug in this plan's own new test] `TestGetHealthProjectsStatusResult` initially asserted `db_size_bytes` byte-exact against an independently-opened Engine's own `Status()` scan, which is inherently racy against a live Pebble store.**
- **Found during:** Task 3, first GREEN run — observed a real, transient mismatch (`8409` vs `8654`), not a hypothetical.
- **Root cause:** `DbSizeBytes` is a live recursive byte-sum over `.codegraph/store/`'s SSTables/WAL/MANIFEST (D-07); two separate scans of the same live store (this RPC's internal scan, and this test's own independent verification `Engine`) can observe a few bytes of drift between them.
- **Fix:** changed the assertion to `> 0`, mirroring `internal/query/files_status_test.go:518`'s existing convention for the identical field.
- **Files modified:** `internal/uiserver/health_test.go`.
- **Verification:** `TestGetHealthProjectsStatusResult` PASS across 3 repeated runs (`-count=3`).

### Verification-criterion false positives investigated and confirmed benign (not fixed)

**4. `rg -o 'GetIndexHealth|GetIndexStats' internal/ web/src/lib/gen | wc -l` reports 1, not the plan's expected 0.**
- **Cause:** the one hit is `internal/uiproto/uiv1/ui.pb.go:2877`'s auto-generated Go getter `func (x *GetHealthResponse) GetIndexHealth() *IndexHealth` — protoc-gen-go's standard `Get<FieldName>` accessor for the maintainer-approved `index_health` field (checkpoint sub-decision 3), not the rejected rpc name.
- **Confirmed the actual invariant holds:** `uiv1connect.UIServiceHandler`'s method set (`internal/uiproto/uiv1/uiv1connect/ui.connect.go:280-...`) contains only `GetHealth`, never `GetIndexHealth`/`GetIndexStats` — the `mutatingVerbs` guard iterates that interface's method names exclusively, and `TestUIServiceDeclaresNoMutatingMethod` passes with all 11 inspected. This is an unavoidable, foreseeable byproduct of approving field name `index_health` at Task 1, not a naming-landmine violation.

**5. `rg -c 'TestGetHealthWorktreeMismatchPopulated|TestGetHealthWorktreeMismatchNilInTree' internal/uiserver/health_test.go` reports 4, not the plan's expected 2.**
- **Cause:** each test name appears twice per test (once in its doc comment, once in its `func` declaration) — `rg -c` counts matching LINES, and both tests are thoroughly documented.
- **Confirmed the actual invariant holds:** `rg -c '^func TestGetHealthWorktreeMismatchPopulated\(t \*testing\.T\)|^func TestGetHealthWorktreeMismatchNilInTree\(t \*testing\.T\)' internal/uiserver/health_test.go` reports exactly 2 — both distinct test functions exist.

**Impact on plan:** none of the five affected the plan's specified wire shape, handler behavior, or scope.

## Finding Recorded, Not Fixed (per Task 3 step (e))

`Engine.Status` already calls `Engine.WorktreeMismatch` internally (`internal/query/status.go:359`), and the pre-existing `GetStatus` handler already calls `eng.Status(ctx)` (`internal/uiserver/handlers.go:316`, unchanged by this plan) — so the up-to-four git subprocesses D-02 confines to `GetHealth`'s response SHAPE are, in practice, **already spawned on every navigation today**, and have been since Phase 1. D-02 is honored exactly as written: the mismatch field is not added to `GetStatusResponse`. But the underlying git-subprocess cost predates this phase, is shared by the CLI's `status --json` path, and eliminating it would change `Engine.Status`'s contract for all three consumers (browser UI, CLI, MCP) — out of scope for this plan. **Flagged here for the maintainer as a follow-up, not fixed.**

## TDD Gate Compliance

This plan's two `type="auto" tdd="true"` tasks each specify RED-then-GREEN as an in-task sequence with the RED output recorded in this SUMMARY (per each task's own `<action>` steps), rather than the generic `tdd_execution` guidance's plan-wide `test(...)` commit followed by a separate `feat(...)` commit. No standalone `test(...)`-typed commit exists in this plan's git history; both commits are `feat(04-03): ...`. RED output was captured via direct `go test` runs (not committed) before each GREEN implementation:
- Task 2 RED: `TestUIServiceMethodSetIsExactlyTheReadSet` FAIL (observed 10, want 11), `TestUIServiceDeclaresNoMutatingMethod` still PASS (inspected 10) — recorded above `git log` at commit `bfbbe9aa`.
- Task 3 RED: all 6 new `TestGetHealth*` tests FAIL against the Task 2 placeholder (0 PASS, 6 FAIL, `go test` exit 1) — recorded before commit `a212d769`.

Both RED observations are genuine (build succeeded, assertions failed honestly), and both were followed by a GREEN commit. This is a task-level, not commit-type-level, satisfaction of the RED/GREEN discipline — flagged per the generic protocol's own instruction to note when the plan-wide `test(...)`/`feat(...)` commit pair is absent.

## Self-Check

- `internal/uiproto/uiv1/ui.proto` — FOUND
- `internal/uiproto/uiv1/ui.pb.go` — FOUND
- `internal/uiproto/uiv1/uiv1connect/ui.connect.go` — FOUND
- `web/src/lib/gen/ui_pb.ts` — FOUND
- `internal/uiserver/handlers.go` — FOUND
- `internal/uiserver/readonly_test.go` — FOUND
- `internal/uiserver/health_test.go` — FOUND
- Commit `bfbbe9aa` — FOUND in `git log`
- Commit `a212d769` — FOUND in `git log`

## Self-Check: PASSED

All 7 claimed files verified present on disk; both claimed commits (`bfbbe9aa`, `a212d769`) verified present in `git log`. No missing items.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- `GetHealth` is live on the wire and ready for 04-05's health view to consume directly — every value HLT-01/02/03 name is present, field-for-field mirrored from `internal/query.StatusResult`.
- `GetStatusResponse` is untouched; the client's existing per-navigation status gate needs no changes from this plan.
- The `Engine.Status` per-navigation git-subprocess-cost finding (above) is a candidate for a future maintainer-directed follow-up phase — not blocking for 04-05.
- No blockers for 04-05; sibling plans 04-01 and 04-02 remain untouched by this plan.

---
*Phase: 04-query-workbench-index-health*
*Completed: 2026-08-30*
