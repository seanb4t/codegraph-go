---
phase: 6
slug: live-push
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
status: validated
nyquist_compliant: true
wave_0_complete: true
created: 2026-09-01
validated: 2026-09-07
---

# Phase 6 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Seeded from `06-RESEARCH.md` §Validation Architecture before plans exist. The Per-Task
> Verification Map is filled once PLAN.md task IDs are assigned; `/gsd-validate-phase` sets
> `status: validated`.

---

## Test Infrastructure

Three stacks. All must be green.

| Property | Go | Frontend (unit) | Real browser |
|----------|----|-----------------|--------------|
| **Framework** | stdlib `testing` + `go.uber.org/goleak` | Vitest `4.1.11` + `@testing-library/svelte` | `@playwright/test@1.62.1` |
| **Config** | none — flags only | inline `test:` block in `web/vite.config.ts` | none — standalone `.mjs` scripts |
| **Location** | `internal/**/[name]_test.go` | `web/tests/*.test.ts` (never beside source) | `web/scripts/*-check.mjs` |
| **Quick run** | `GOTOOLCHAIN=go1.26.5 go test ./internal/uiserver/...` | `cd web && pnpm test -- <pattern>` | `node web/scripts/<script>.mjs` |
| **Full suite** | `GOTOOLCHAIN=go1.26.5 task test:unit` | `task web:test` | all check scripts |

> **`GOTOOLCHAIN=go1.26.5` is required on every local Go command** — go1.27 breaks the
> `cockroachdb/swiss` build. CI is unaffected.
>
> **`task test:unit` deliberately excludes `internal/daemon`** (`Taskfile.yml:131`) and
> isolates it separately, because its watchdog test is timing-sensitive under parallel load.
> A bare `go test ./...` will fail there; that is known, not a regression.
>
> **jsdom has no layout engine** — `offsetWidth`/`offsetHeight` are always `0`. Cytoscape
> tests use `headless: true` and assert over the graph model, never pixels.
>
> **Real gestures need trusted input** — cytoscape's drag path uses `setPointerCapture`,
> which `dispatchEvent` cannot satisfy. Playwright's `page.mouse.*` drives it correctly.
> Follow `web/scripts/graph-collapse-affordance-check.mjs`'s established pattern: real input,
> a committed diagnostic JSON record written on **every** run including failure.

---

## Sampling Rate

- **After every task commit:** `GOTOOLCHAIN=go1.26.5 go test ./internal/uiserver/...` for Go
  changes · `cd web && pnpm test -- <touched-pattern>` for frontend changes.
- **After every plan wave:** `GOTOOLCHAIN=go1.26.5 task test:unit` + `task web:test`.
- **Before `/gsd-verify-work`:** full suites green, **plus** the two real-process gates below,
  which no unit test can substitute for.
- **Max feedback latency:** ~60 seconds.

---

## Per-Task Verification Map

*Filled once PLAN.md task IDs are assigned. The requirement→test mapping below is fixed and
must be preserved when the IDs land.*

| Task ID | Plan | Wave | Requirement | Threat Ref | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------|-------------------|-------------|--------|
| 06-01 T1 | 06-01 | 1 | RPC-04 | T-06-01 | unit (Go) — rpc name clean against the live fixture, with a positive control | `GOTOOLCHAIN=go1.26.5 go test ./internal/uiserver/ -run '^TestRPCName' -v -count=1` | ✅ | ✅ green |
| 06-01 T3 | 06-01 | 1 | RPC-04 | T-06-02, T-06-03 | fixture (Go) — descriptor field-number stability, both directions | `GOTOOLCHAIN=go1.26.5 go test ./internal/uiserver/ -run '^TestUIProtoFieldNumbersAreStableAndUnique' -v -count=1` · `task proto:drift` | ✅ | ✅ green |
| 06-02 T1 | 06-02 | 2 | LIV-01 | T-06-05, T-06-11, T-06-12 | unit (Go) — event fires only on a real metadata change; open/close balanced | `GOTOOLCHAIN=go1.26.5 go test ./internal/uiserver/ -run '^TestLiveChange' -v -count=1` | ✅ | ✅ green |
| 06-02 T2 | 06-02 | 2 | LIV-03 | T-06-07, T-06-08, T-06-09, T-06-10 | unit (Go, `-race` + goleak) — bounded coalescing, cleanup, counter separation | `GOTOOLCHAIN=go1.26.5 go test ./internal/uiserver/ -run '^TestLiveRegistry' -v -count=1 -race` | ✅ | ✅ green |
| 06-02 T3 | 06-02 | 2 | LIV-01 | T-06-06 | integration (Go) — real store write through fsnotify + debounce | `GOTOOLCHAIN=go1.26.5 go test ./internal/uiserver/ -run '^TestLiveWatcher' -v -count=1 -race` | ✅ | ✅ green |
| 06-03 T1 | 06-03 | 2 | LIV-03 | T-06-13, T-06-16 | unit (vitest) — incremental consumption, jittered backoff, generation resume | `cd web && pnpm exec vitest run tests/live-client.test.ts` | ✅ | ✅ green |
| 06-03 T2 | 06-03 | 2 | LIV-02 | T-06-15 | unit (vitest) — one classifier for both inputs; chrome updates with no round trip | `cd web && pnpm exec vitest run tests/live-store.test.ts` | ✅ | ✅ green |
| 06-03 T3 | 06-03 | 2 | LIV-02 | T-06-14 | component (vitest) — open view re-fetches exactly once per new generation | `cd web && pnpm exec vitest run` | ✅ | ✅ green |
| 06-04 T1 | 06-04 | 3 | RPC-04 | T-06-19, T-06-21, T-06-22 | integration (Go) — stream outlives a deliberately tiny write deadline; a unary path does not | `GOTOOLCHAIN=go1.26.5 go test ./internal/uiserver/ -run '^TestStreamDeadline' -v -count=1 -race` | ✅ | ✅ green |
| 06-04 T2 | 06-04 | 3 | LIV-03 | T-06-20, T-06-23, T-06-24 | unit (Go, `-race` + goleak) — subscribe / send / deregister lifecycle | `GOTOOLCHAIN=go1.26.5 go test ./internal/uiserver/ -run '^TestWatchGraphHandler' -v -count=1 -race` | ✅ | ✅ green |
| 06-04 T3 | 06-04 | 3 | RPC-04 (criterion 2) | T-06-21 | **tracer** — real client, real store write, per-message TIMING not arrival | `GOTOOLCHAIN=go1.26.5 go test ./internal/uiserver/ -run '^TestWatchGraphStream' -v -count=1 -race` | ✅ | ✅ green |
| 06-05 T1 | 06-05 | 4 | LIV-04 | T-06-27, T-06-31 | unit (vitest, headless graph model) — fast path runs no layout; survivors written back | `cd web && pnpm exec vitest run tests/graph-live-update.test.ts` | ✅ | ✅ green |
| 06-05 T2 | 06-05 | 4 | LIV-02 | T-06-28, T-06-29 | component (vitest) — graph re-fetches through its own rpcs, coalesced | `cd web && pnpm exec vitest run` | ✅ | ✅ green |
| 06-05 T3 | 06-05 | 4 | LIV-04 | T-06-30 | Playwright e2e at guava scale — survivor displacement measured and committed | `node web/scripts/graph-live-update-check.mjs` | ✅ | ✅ green |
| 06-06 T1 | 06-06 | 5 | LIV-03 | T-06-45 | harness gate (node) — the fixed-port proxy rewrites BOTH Host and Origin to the upstream authority and streams rather than buffers, proven against a recording upstream | `node -e '<proxy gate>' --input-type=module` (the full command is in the plan) | ✅ | ✅ green |
| 06-06 T2 | 06-06 | 5 | LIV-03 (criteria 2, 3) | T-06-33, T-06-34 | Playwright, 3+ real tabs — per-message timing paired with a per-tab received-generation superset, fan-out isolation with the blocked tab proven blocked, jittered reconnect behind the Task 1 proxy | `node web/scripts/live-push-multitab-check.mjs` | ✅ | ✅ green |
| 06-07 T1 | 06-07 | 6 | LIV-01 | T-06-SC | build-exclusion gate (Go) — the `//go:build ignore` probe compiles and runs but never enters the shipped build, and uses the generated client rather than curl | `GOTOOLCHAIN=go1.26.5 go build ./... && go vet ./... && go run scripts/live-push-probe.go --help` | ✅ | ✅ green |
| 06-07 T2 | 06-07 | 6 | LIV-01 (criterion 5) | T-06-32, T-06-37, T-06-46 | **real multi-process**, never a stub — see Real-Process Gates below; flush duration bounded against a measured no-UI baseline | `bash scripts/live-push-concurrency-check.sh` | ✅ | ✅ green |
| 06-07 T3 | 06-07 | 6 | LIV-03 (verdict) | T-06-35, T-06-36 | **recorded finding** with a live positive control, not a test — see Recorded Verdicts below; plus the post-commit bundle staging assertion | `test "$(rg -c 'type pendingWriter struct' internal/mcp/server.go)" -eq 1` (the control) · `test -z "$(git status --porcelain -- web/build)"` after the bundle commit | ❌ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

> ⚠ **Corrected 2026-09-01 (plan review cycle 1).** The 06-01 T3 row previously named
> `TestKnownUIProtoFieldNumbersAreStable` and marked it **File Exists ✅**. That test does not
> exist in `internal/uiserver`: `rg -n 'func TestKnownUIProtoFieldNumbersAreStable'
> internal/uiserver/readonly_test.go` returns ZERO matches, and the name belongs to
> `internal/schema/meta_commit_test.go:120`. The real guard is
> **`TestUIProtoFieldNumbersAreStableAndUnique`** at `internal/uiserver/readonly_test.go:482`.
> Because `go test -run` exits 0 when its pattern matches nothing, the wrong name meant the
> sole enforcer of the seven pinned field numbers would never have run in the task that adds
> them. Every citation in `06-01-PLAN.md` was corrected in the same pass, and that plan's
> Task 3 verify now asserts the guard's `--- PASS` line BY NAME, separately from its aggregate
> count, so the same skip cannot recur silently.

> **Waves changed in the same pass.** `06-05` gained an undeclared dependency on `06-04`
> (its Task 3 waits for live traffic that only exists once `06-04` replaces `livehandler.go`'s
> placeholder), so `06-05` moved to wave 4 and `06-06` to wave 5. The wave column below
> reflects the corrected assignment.
>
> **Re-slice (maintainer decision, same date).** `06-06` carried BOTH real-world gates at an
> estimated 100k tokens. It was split into `06-06` (wave 5 — the browser gates: the fixed-port
> proxy and the three-tab check) and `06-07` (wave 6 — criterion 5's real-process gate, the
> recorded verdict, and the phase's single `web/build` rebuild). `06-06` sat at the END of the
> DAG, so no other plan's `depends_on` changed. The phase now has SEVEN plans.

---

## Wave 0 Requirements

- [x] `internal/uiserver/livepublish_test.go` — LIV-01, plus the goleak-guarded soak shape
      criterion 5 needs.
- [x] `internal/uiserver/livehandler_test.go` — RPC-04 at the handler level, with
      **message-by-message** `httptest` assertions (see the criterion-2 warning below).
- [x] `web/tests/live-client.test.ts` — browser-side reconnect/backoff and generation
      tracking; vitest can mock `fetch`'s streaming body.
- [x] `web/scripts/live-push-multitab-check.mjs` — **new**, 3+ real tabs, following
      `graph-collapse-affordance-check.mjs`'s precedent.
- [x] `web/scripts/graph-live-update-check.mjs` — **new**, measures guava-scale survivor
      displacement across a live update. This is D-06's mandated measurement.
- [x] No framework install needed — vitest, Playwright and goleak are all already present.

---

## ⚠ Two criteria that ordinary tests CANNOT satisfy

These come from the ROADMAP's own Notes and are the reason this phase carries a research
flag. A plan that satisfies only the table above has not satisfied the phase.

### Criterion 2 — assert per-message TIMING, not arrival

> *"Criterion 2's message-by-message assertion measures per-message delivery latency, not
> eventual arrival — streaming that is silently buffered still passes an 'it all arrived'
> test."*

**A test asserting that N messages were received is vacuous here.** Silent buffering
delivers all N at the end and passes. Assert the **interval between** messages, or assert
that message *k* is observable before message *k+1* is sent.

### Criterion 5 — real processes, never a stub

> *"With `codegraph daemon` and `serve --mcp` running against the same store, a live-push
> session survives repeated real re-index flushes without starving a sync or holding the
> store open — verified against the real processes, not a stub."*

The ROADMAP calls this **non-negotiable** and explains why: *"the property it must not
violate is only observable under genuine concurrent multi-process use, and running
`codegraph ui` alone is the dev workflow that hides it."*

The concrete hazard: `internal/graphstore/store.go:14-21` documents *"many lock-free readers
via Snapshot, plus one [writer]"*. A Pebble snapshot **held open pins SSTables and blocks
compaction**. So the gate must show that repeated real re-index flushes proceed while a
live-push session is open — with actual `codegraph daemon` and `serve --mcp` processes
running against the same store.

---

## Manual-Only Verifications

| Behavior | Requirement | Why not automatable here | Instructions |
|----------|-------------|--------------------------|--------------|
| *(none identified yet)* | — | — | Before recording anything here, **try to verify it**. In Phase 3 two of three `why_human` claims were false on arrival, and in Phase 5 a "needs a human on a real trackpad" item turned out to be a `setPointerCapture` limitation in the tooling — disproved by driving Playwright directly. A `why_human` claim is a **testable claim, not a category**. |

---

## Recorded Verdicts (criterion 3)

LIV-03 requires the stream's lifecycle bookkeeping be *"checked against Phase 1's
`pendingWriter` root cause — server-initiated writes counted separately from
client-initiated pending state — with the verdict recorded rather than assumed."*

Research already established the finding to confirm and record: **no `pendingWriter`
analogue exists in `internal/uiserver`** — a full read of every non-test file found zero
counters or in-flight trackers. Record that verdict explicitly in the SUMMARY with the
evidence; "we checked and it's fine" does not satisfy the criterion.

---

## Validation Audit 2026-09-07

| Metric | Count |
|--------|-------|
| Map rows | 19 |
| Gaps found (MISSING) | **0** |
| Rows green | 19 |
| Resolved this audit | 0 (none needed) |

Re-run fresh 2026-09-07: **44 `--- PASS`, 0 `--- FAIL`** on the Go side, with `-race` on every
concurrency-bearing suite — `TestLiveChange` (9), `TestLiveRegistry` (11, race), `TestLiveWatcher`
(6, race), `TestStreamDeadline` (4, race), `TestWatchGraphHandler` (11, race),
`TestWatchGraphStream` (1, race), plus the rpc-name and field-number guards (1 each).
`go build ./... && go vet ./...` exit 0. **Positive control:** a nonexistent `-run` pattern
returned PASS=0 while exiting 0.

All four vitest suites and all four harness programs exist on disk
(`live-client`, `live-store`, `graph-live-update`, `live-route-refetch`;
`graph-live-update-check.mjs`, `live-push-multitab-check.mjs`,
`live-push-concurrency-check.sh`, `live-push-probe.go`), and `pnpm test` is **467/467**.

**The three browser/multi-process gates are committed records, and all three read
`success: true`:**

| Corpus | What it proves |
|---|---|
| `corpora/graph-live-update-check.json` | criterion 4 — 349 leaf survivors at **0px** max and mean displacement, real guava checkout |
| `corpora/live-push-multitab-check.json` | criteria 2 & 3 — 3 real tabs, `blockedTabReceiptsDuringBlock: 0`, `reconnectAttemptsPerTab [3,3,3]` |
| `corpora/live-push-concurrency-check.json` | criterion 5 — real `daemon` + `serve --mcp` + `ui`, 5/5 flushes completed, 0 starved |

Criterion 5's gate was **independently re-executed during this milestone's verification pass**
and passed against a *freshly measured* baseline (296ms) that differs from the committed
record's (261ms) — which is itself the proof that the bound is measured rather than hardcoded.

**06-07 T3's verdict control re-checked, not assumed:** `type pendingWriter struct` appears
exactly **1** time in `internal/mcp/server.go` (the discriminating positive control), and
`git status --porcelain -- web/build` is empty after the bundle commit.

**Scope note:** `06-08-PLAN.md` was added after this map was seeded, closing the criterion-1 gap
the live browser UAT found post-verification (the root Status route never applied live events).
Its verification is the root-route describe block in `web/tests/live-route-refetch.test.ts` —
three cases, all demonstrated RED before the fix and independently re-demonstrated RED twice by
the verifier — plus a live browser re-confirmation (616→617 files, applied <50ms, no reload).
LIV-01 and LIV-02 are covered by rows in this map either way, so no requirement is unverified.

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 60s
- [x] Every zero-count assertion paired with a positive control (rule `84d1gfpywd`)
- [x] Every upper bound paired with a non-zero lower bound
- [x] **Criterion 2's timing assertion present** — not merely an arrival count — and PAIRED with a receipt floor at both levels (Go: triggered-receipt count equals trigger count, floor 3; browser: every tab's received-generation set is a superset of `triggeredGenerations`), because an inter-arrival minimum computed over one receipt is vacuously satisfied
- [x] **Criterion 5 verified against real `daemon` + `serve --mcp` processes** — not a stub — with `serve --mcp` held open over persistent stdio pipes for the whole run and the stream held by a real envelope-decoding Connect client, not `curl`
- [x] Flush duration bounded against a measured no-UI baseline, so a slowed-but-completing sync cannot pass as unstarved
- [x] The `pendingWriter`-analogue verdict recorded in the SUMMARY, with a control that discriminates on `type pendingWriter struct` rather than on the bare word
- [x] Every named test cited in this file and in any PLAN.md verified to EXIST (`rg -n 'func <Name>'` with a positive control) — the cycle-1 review found a cited test that did not
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** validated 2026-09-07
