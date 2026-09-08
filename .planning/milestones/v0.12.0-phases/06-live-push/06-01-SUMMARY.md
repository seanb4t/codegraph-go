---
phase: 06-live-push
plan: 01
subsystem: api
tags: [connect-rpc, protobuf, streaming, codegen, ui-server]

requires:
  - phase: 05-graph-view
    provides: a stable 13-rpc UIService surface and its two descriptor/method-set guards to extend
provides:
  - "WatchGraph: the 14th UIService rpc and the service's first Connect server-streaming method"
  - "WatchGraphRequest (since_generation) and WatchGraphEvent (generation, initialized, stale, store_exists, indexing_in_progress, commit_sha), frozen field numbers per D-02a"
  - "Regenerated Go (ui.pb.go, uiv1connect/ui.connect.go) and TypeScript (web/src/lib/gen/ui_pb.ts) wire surfaces, byte-identical to the pinned toolchain"
  - "A CodeUnimplemented placeholder (*uiService).WatchGraph satisfying the interface, structurally gated to be replaced by 06-04"
affects: [06-02, 06-03, 06-04, 06-05, 06-06, 06-07]

actuals:
  tokens: 15709
  tasks: 3
  commits: 2

tech-stack:
  added: []
  patterns:
    - "Streaming rpc placeholder pattern: a *uiService method satisfying uiv1connect.UIServiceHandler by returning connect.NewError(connect.CodeUnimplemented, ...) with a doc comment naming the plan that replaces it — pinned by a grep assertion rather than a test that would be written and deleted in the same phase"
    - "Compile-time TypeScript method-kind assertion: derive ReturnType<typeof client.method>, assign it to a function typed to expect AsyncIterable<T>, and let pnpm run check (svelte-check) be the enforcer — proves streaming vs unary through the type system rather than string-counting the generated file"

key-files:
  created:
    - internal/uiserver/rpcname_test.go
    - internal/uiserver/livehandler.go
    - web/tests/gen-watchgraph-type.test.ts
  modified:
    - internal/uiproto/uiv1/ui.proto
    - internal/uiproto/uiv1/ui.pb.go
    - internal/uiproto/uiv1/uiv1connect/ui.connect.go
    - web/src/lib/gen/ui_pb.ts
    - internal/uiserver/readonly_test.go

key-decisions:
  - "WatchGraph chosen as the 14th rpc name, verified clean against the live 19-member mutatingVerbs fixture with GetIndexHealth as the standing positive control; WatchIndex/IndexEvents/StreamIndex/LiveUpdates all rejected"
  - "since_generation alone is the resume handle on WatchGraphRequest — widening was offered at the freeze checkpoint and declined (maintainer decision 2026-09-07): nothing in the five phase success criteria needs more, and D-01's event source is a single store-wide signal with nothing per-client to filter on"
  - "D-07's six-field event set frozen as proposed (maintainer decision 2026-09-07): mirrors GetStatusResponse's field NAMES so classifyStatus consumes the event directly; node_count/edge_count were offered and declined because nothing shipped renders them and an unpopulated field invites a future reader to assume it is meaningful"
  - "Server-streaming, not bidirectional (maintainer decision 2026-09-07): the client has nothing to say after subscribing"
  - "The 06-04 placeholder body returns connect.CodeUnimplemented, named explicitly, so it can never be mistaken for a real internal error if it ever escaped this wave; no unit test was written for it (06-04 deletes the body one wave later) — pinned instead by a grep assertion in this plan's own acceptance criteria"

patterns-established:
  - "Streaming-method placeholder: land the interface-satisfying no-op body in the same wave as regeneration, gated structurally by the next wave's end-to-end test rather than by a unit test that would be written and deleted in the same phase"

requirements-completed: [RPC-04]

coverage:
  - id: D1
    description: "WatchGraph rpc name proven clean against the live mutatingVerbs fixture, with a positive control proving the checker discriminates"
    verification:
      - kind: unit
        ref: "internal/uiserver/rpcname_test.go#TestRPCNameIsCleanAgainstMutatingVerbs"
        status: pass
    human_judgment: false
  - id: D2
    description: "Frozen proto block (WatchGraph rpc + WatchGraphRequest + WatchGraphEvent, seven field numbers) approved by the maintainer before regeneration"
    verification: []
    human_judgment: true
    rationale: "Field-number freeze is a one-way door (D-02a) requiring an explicit human approval, not a property any test can assert — approval text is recorded verbatim below."
  - id: D3
    description: "uiv1connect.UIServiceHandler declares exactly 14 methods and wantUIServiceMethods agrees in both directions"
    verification:
      - kind: unit
        ref: "internal/uiserver/readonly_test.go#TestUIServiceMethodSetIsExactlyTheReadSet"
        status: pass
      - kind: unit
        ref: "internal/uiserver/readonly_test.go#TestUIServiceDeclaresNoMutatingMethod"
        status: pass
    human_judgment: false
  - id: D4
    description: "Seven new field numbers pinned and covered in both directions by the generated-descriptor guard"
    verification:
      - kind: unit
        ref: "internal/uiserver/readonly_test.go#TestUIProtoFieldNumbersAreStableAndUnique"
        status: pass
    human_judgment: false
  - id: D5
    description: "Both generated language surfaces regenerated and byte-identical to a fresh regeneration through the pinned toolchain"
    verification:
      - kind: other
        ref: "task proto:drift"
        status: pass
    human_judgment: false
  - id: D6
    description: "Generated TypeScript client's watchGraph is server-streaming (AsyncIterable<WatchGraphEvent>), proven by the type system"
    verification:
      - kind: unit
        ref: "web/tests/gen-watchgraph-type.test.ts"
        status: pass
      - kind: other
        ref: "pnpm -C web run check"
        status: pass
    human_judgment: false

duration: 191min
completed: 2026-09-07
status: complete
---

# Phase 6 Plan 1: WatchGraph Wire-Surface Freeze Summary

**Froze and regenerated the 14th UIService rpc — `WatchGraph`, the service's first Connect server-streaming method — with seven new field numbers pinned in both directions and a structurally-gated `CodeUnimplemented` placeholder.**

## Performance

- **Duration:** 191 min (includes the blocking-human proto-freeze checkpoint pause)
- **Started:** 2026-09-07T14:50:42Z
- **Completed:** 2026-09-07T19:02:10Z (approx, self-check time)
- **Tasks:** 3 (2 `auto` + 1 `checkpoint:human-verify` gate="blocking-human")
- **Files modified:** 8 (3 created, 5 modified)

## Accomplishments

- `WatchGraph(WatchGraphRequest) returns (stream WatchGraphEvent)` added additively to `service UIService` in `internal/uiproto/uiv1/ui.proto`, verified clean against the live 19-member `mutatingVerbs` fixture (`internal/uiserver/rpcname_test.go`) before being written.
- Seven field numbers frozen and pinned in both directions by `TestUIProtoFieldNumbersAreStableAndUnique`: `WatchGraphRequest.since_generation = 1`; `WatchGraphEvent.generation = 1, initialized = 2, stale = 3, store_exists = 4, indexing_in_progress = 5, commit_sha = 6`.
- Both generated language surfaces regenerated via `task proto:gen` and confirmed byte-identical to the pinned toolchain via `task proto:drift` (4 files compared, clean).
- `internal/uiserver/readonly_test.go`'s two descriptor/method-set fixtures extended to 14 methods and +7 field entries, with two three-waves-stale prose sites corrected in the same task.
- A structurally-gated `CodeUnimplemented` placeholder lands `(*uiService).WatchGraph` so the package compiles; `06-04`'s end-to-end test cannot pass while this body survives.
- A compile-time TypeScript assertion (`web/tests/gen-watchgraph-type.test.ts`) proves `watchGraph` is server-streaming, enforced by `pnpm run check`.

## Task Commits

Each task was committed atomically:

1. **Task 1: Name the rpc, prove the name clean, author the frozen proto block** — `477cfa0d` (test)
2. **Task 2: Proto freeze review — checkpoint** — no commit (human-verify gate; approval recorded below)
3. **Task 3: Regenerate both language surfaces and extend the two descriptor fixtures** — `c5833c5c` (feat)

**Plan metadata:** commit follows this SUMMARY.

## Files Created/Modified

- `internal/uiserver/rpcname_test.go` — new: proves `WatchGraph` collides with no `mutatingVerbs` member, with `GetIndexHealth` as the positive control
- `internal/uiproto/uiv1/ui.proto` — `WatchGraph` rpc + `WatchGraphRequest`/`WatchGraphEvent` messages appended to `service UIService`
- `internal/uiproto/uiv1/ui.pb.go` — regenerated (Go messages)
- `internal/uiproto/uiv1/uiv1connect/ui.connect.go` — regenerated (14-method handler/client interfaces, `UIServiceWatchGraphProcedure`)
- `web/src/lib/gen/ui_pb.ts` — regenerated (TypeScript schemas + `watchGraph` client method)
- `internal/uiserver/livehandler.go` — new: `(*uiService).WatchGraph` placeholder returning `connect.CodeUnimplemented`
- `internal/uiserver/readonly_test.go` — `wantUIServiceMethods` (13→14), `uiProtoFieldNumbers` (+7, `uiProtoFieldFixtureLenAtPlan0601`), two stale-prose corrections
- `web/tests/gen-watchgraph-type.test.ts` — new: compile-time proof `watchGraph` returns `AsyncIterable<WatchGraphEvent>`

## Proto Freeze Checkpoint — Recorded Verbatim

**Approval:** "approved — MAINTAINER DECISION, 2026-09-07. Freeze the seven field numbers as proposed and proceed to Task 3."

**The four questions and answers, as given:**

1. **Is `WatchGraph` the name to live with?** Yes. Verified clean against all 19 `mutatingVerbs` with a live positive control proving `GetIndexHealth` still collides on `Index`. `WatchIndex`, `IndexEvents`, `StreamIndex` (all contain `Index`) and `LiveUpdates` (contains `Update`) are rejected by that same fixture.
2. **Is `since_generation` the right resume handle, or should the request carry more?** `since_generation` alone. Widening the request was offered and declined: nothing in the five phase success criteria needs more, and D-01 makes the event source a single store-wide signal, so a per-client filter would have nothing to filter on.
3. **Is the six-field event set complete?** Yes — D-07, chosen because it mirrors `GetStatusResponse`'s field *names* exactly so `classifyStatus` consumes it directly with no second representation of the same state. Adding `node_count`/`edge_count` now was offered and declined — nothing shipped renders them, and a field that exists but is never populated invites a future reader to assume it is meaningful. Adding fields later is additive-only, so nothing is foreclosed.
4. **Is `stream WatchGraphEvent` (server-streaming, not bidi) the intended shape?** Yes — the client has nothing to say after subscribing.

**Frozen permanently:** `WatchGraphRequest.since_generation = 1`; `WatchGraphEvent.generation = 1, initialized = 2, stale = 3, store_exists = 4, indexing_in_progress = 5, commit_sha = 6`. Method count 13 → 14 once regenerated.

**Orchestrator's independent verification before approving** (checked, not taken on report):
- All three generated files genuinely untouched at approval time: `WatchGraph` appeared 0 times in `ui.pb.go`, `ui_pb.ts` and `ui.connect.go`, `git status --porcelain` clean on each, 7 hits in `ui.proto`.
- `mutatingVerbs` untouched across the whole phase: `git diff 08c9f206..HEAD -- internal/uiserver/readonly_test.go` was 0 lines, `"Reindex"` still present.
- `TestRPCNameIsCleanAgainstMutatingVerbs` re-run live: PASS, logging `mutatingVerbs has 19 members and still includes Reindex`.

**Captured `TestUIProtoFieldNumbersAreStableAndUnique` PASS line, evidence the descriptor guard ran in the same task that extended its fixture:**

```
--- PASS: TestUIProtoFieldNumbersAreStableAndUnique (0.00s)
    readonly_test.go:551: inspected 42 messages and 169 fields in the generated uiv1 descriptor
```

## Decisions Made

See `key-decisions` in frontmatter — all four are the maintainer's proto-freeze answers, recorded verbatim above, plus the placeholder-body design (`CodeUnimplemented`, no unit test, pinned by a grep assertion because `06-04` deletes the body one wave later).

## Deviations from Plan

None — plan executed exactly as written. The rpc name, request shape, and event field set all matched the plan's proposed text; the maintainer approved as proposed with no amendment.

## Issues Encountered

None.

## Verification Re-run (plan-level `<verification>` block, end-to-end)

- `GOTOOLCHAIN=go1.26.5 go build ./...` — succeeds, no output.
- `GOTOOLCHAIN=go1.26.5 task test:unit` — green, all packages `ok` (including `internal/uiserver` at 36.008s); `internal/daemon`'s watchdog test correctly excluded per the target's own deliberate design.
- `task proto:drift` — green: `proto:drift: compared 4 generated files` / `all 4 generated files byte-identical to the pinned toolchain's regeneration`.
- `pnpm -C web run check` — exits 0: `COMPLETED 1153 FILES 0 ERRORS 0 WARNINGS 0 FILES_WITH_PROBLEMS`, with `web/tests/gen-watchgraph-type.test.ts` present and included via the SvelteKit-generated tsconfig's `../tests/**/*.ts`.
- The blocking-human proto-freeze checkpoint was answered by explicit maintainer approval (recorded verbatim above), never auto-advanced — `gate="blocking-human"` was honored regardless of `workflow.auto_advance`.

## Known Stubs

- `(*uiService).WatchGraph` in `internal/uiserver/livehandler.go` returns `connect.CodeUnimplemented` unconditionally. This is the plan's own explicit design (Task 3 action, `<done>` criterion: "the proto block is authored but not yet regenerated" → now regenerated with an interface-satisfying no-op): no subscriber-register/Send-loop/deregister lifecycle is built in this plan. **Resolved by 06-04**, whose own end-to-end test (a real Connect client receiving a real, watcher-triggered event over real HTTP/1.1) cannot pass while this body survives — that test is the structural gate preventing the placeholder from shipping past this phase.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

`06-02` through `06-07` can now generate against a stable 14-rpc surface: `WatchGraphRequest`/`WatchGraphEvent` types exist in both Go and TypeScript, the Connect procedure and streaming client/handler plumbing exist, and the placeholder handler compiles cleanly. No blockers. `06-04` must replace `livehandler.go`'s placeholder body before its own end-to-end tracer test can pass.

---
*Phase: 06-live-push*
*Completed: 2026-09-07*
