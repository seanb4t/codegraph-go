---
phase: 5
slug: file-package-graph-view
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-30
---

# Phase 5 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Seeded by plan-phase §5.5 from `05-RESEARCH.md` §Validation Architecture, before
> plans exist. The Per-Task Verification Map is filled once PLAN.md task IDs are
> assigned; `/gsd-validate-phase` sets `status: validated`.

---

## Test Infrastructure

This phase spans **two** test stacks. Both must be green.

| Property | Frontend | Backend |
|----------|----------|---------|
| **Framework** | Vitest `4.1.11` + `@testing-library/svelte@5.4.2` (`web/package.json`) | Go `testing` (stdlib, no third-party framework) |
| **Config file** | `web/vite.config.ts` — the `test:` block is inline via `/// <reference types="vitest/config" />`; there is deliberately **no** separate `vitest.config.ts`. Global jsdom stubs live in `web/tests/setup.ts` | none (stdlib) |
| **Test location** | `web/tests/*.test.ts` — **not** `web/src/**/*.test.ts` | `internal/**/[name]_test.go`, package-local |
| **Quick run** | `cd web && pnpm test -- file-graph` | `GOTOOLCHAIN=go1.26.5 go test ./internal/query/... -run FileGraph` |
| **Full suite** | `task web:test` (asserts a positive executed-test count before judging pass/fail) | `task test:unit` |
| **Estimated runtime** | ~5–15s | ~30–60s |

> **Toolchain note:** `GOTOOLCHAIN=go1.26.5` is required on every local Go command —
> go1.27 breaks the `cockroachdb/swiss` build. CI is unaffected (`go-version-file` pins 1.26).

> **Renderer-testing note (Pitfall 4):** jsdom has no layout engine, so `offsetWidth` /
> `offsetHeight` are always `0`. This already bit `@tanstack/virtual-core` in Phase 4 and
> will bite Cytoscape here. Keep canvas-dependent assertions **out** of jsdom: instantiate
> Cytoscape with `headless: true` and assert over the graph model, not over pixels.

---

## Sampling Rate

- **After every task commit:** `cd web && pnpm test -- <touched-file-pattern>` (frontend
  changes) · `GOTOOLCHAIN=go1.26.5 go test ./internal/query/... -run <TestName>` (backend
  changes) — scoped and fast.
- **After every plan wave:** `task web:test` (full frontend) · `task test:unit` (full backend).
- **Before `/gsd-verify-work`:** full suite green, **plus** a mandatory live browser UAT
  against the built binary. This is not optional garnish — see Manual-Only Verifications.
- **Max feedback latency:** ~60 seconds.

---

## Per-Task Verification Map

*Filled once PLAN.md task IDs are assigned. Requirement→test mapping is already fixed
below and must be preserved when the task IDs land.*

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 05-01 T1 | 05-01 | 1 | GRF-01 | T-05-01, T-05-06 | The pass condition is committed alone, before any measurement artifact exists | artifact + git ordering | `test -f corpora/graph-render-threshold.json && test ! -e corpora/graph-render-observations.json` | ❌ W0 | ⬜ pending |
| 05-01 T2 | 05-01 | 1 | ENG-03 | T-05-02, T-05-05 | Rollup aggregates only over the indexed record set; no filesystem read; package pseudo-nodes excluded and counted | unit (Go) | `GOTOOLCHAIN=go1.26.5 go test ./internal/query/... -run TestFileGraph -race` | ❌ W0 | ⬜ pending |
| 05-01 T3 | 05-01 | 1 | GRF-04 | T-05-04 | Cycle detection over the aggregated file graph, server-side and iterative | unit (Go) | `GOTOOLCHAIN=go1.26.5 go test ./internal/query/... -run TestFileGraphCycles` | ❌ W0 | ⬜ pending |
| 05-02 T1 | 05-02 | 2 | ENG-03 | T-05-11 | Wire shape frozen by a human before codegen; field numbers additive-only | checkpoint (recorded) | recorded in 05-02-SUMMARY.md | ❌ W0 | ⬜ pending |
| 05-02 T2 | 05-02 | 2 | ENG-03 | T-05-07 | `FileGraph` clean against all 19 mutating verbs; method count 11→12; verb fixture byte-unchanged | unit (Go) | `GOTOOLCHAIN=go1.26.5 go test ./internal/uiserver/... -run TestUIService` | ✅ extend `readonly_test.go` | ⬜ pending |
| 05-02 T3 | 05-02 | 2 | ENG-03 | T-05-08, T-05-09, T-05-12 | Handler never degrades; request path unread; response size MEASURED under the transport ceiling | unit (Go) + measurement | `GOTOOLCHAIN=go1.26.5 go test ./internal/uiserver/... -run TestFileGraph` | ❌ W0 | ⬜ pending |
| 05-03 T1 | 05-03 | 3 | GRF-05 | T-05-SC | Package legitimacy approved by a human before any install runs | checkpoint (recorded) | recorded in 05-03-SUMMARY.md | ❌ W0 | ⬜ pending |
| 05-03 T2 | 05-03 | 3 | GRF-02 | T-05-14 | Compound parent chains built from file paths; DOM-free and renderer-free | unit (vitest, DOM-free) | `cd web && pnpm test -- file-graph-transform` | ❌ W0 | ⬜ pending |
| 05-03 T3 | 05-03 | 3 | GRF-02, GRF-05 | T-05-13, T-05-15, T-05-17 | Exactly one importer of the renderer; no worker; no force-directed layout; no planning vocabulary | unit (vitest, headless) + manual | `cd web && pnpm test -- graph-tracer` | ❌ W0 | ⬜ pending |
| 05-04 T1 | 05-04 | 4 | GRF-01 | T-05-19 | Comparator fails closed on a missing, non-numeric or unreadable metric | unit (vitest) | `cd web && pnpm test -- graph-verdict` | ❌ W0 | ⬜ pending |
| 05-04 T2 | 05-04 | 4 | GRF-01 | T-05-18, T-05-20, T-05-22 | Measured latency recorded against a threshold whose commit provably predates it | spike (recorded measurement) | `node -e` verdict-completeness gate in 05-04 T2 | ❌ W0 | ⬜ pending |
| 05-04 T3 | 05-04 | 4 | GRF-01 | T-05-18 | A human reads the verdict; the threshold is never edited in response | checkpoint (recorded) | recorded in 05-04-SUMMARY.md and STATE.md | ❌ W0 | ⬜ pending |
| 05-05 T1 | 05-05 | 5 | GRF-04 | T-05-25 | Cycle classes come from wire fields only; no client-side connectivity derivation | unit (vitest, Cytoscape headless) | `cd web && pnpm test -- graph-cycles` | ❌ W0 | ⬜ pending |
| 05-05 T2 | 05-05 | 5 | GRF-04 | T-05-24 | Cycle count stated without interaction; focus passed as data, not a renderer call | unit (vitest) | `cd web && pnpm test -- graph-cycles` | ❌ W0 | ⬜ pending |
| 05-05 T3 | 05-05 | 5 | GRF-02 | T-05-23, T-05-26, T-05-27 | Per-kind rows only for kinds the sparse wire map carries; shared table reused | unit (vitest) + manual | `cd web && pnpm test -- graph-edge-detail` | ❌ W0 | ⬜ pending |
| 05-06 T1 | 05-06 | 6 | GRF-03 | T-05-29 | Thirteenth rpc's name and shape frozen by a human before codegen | checkpoint (recorded) | recorded in 05-06-SUMMARY.md | ❌ W0 | ⬜ pending |
| 05-06 T2 | 05-06 | 6 | GRF-03 | T-05-28, T-05-30, T-05-32 | Repo-root confinement runs before any read; capped and counted; index records not disk | unit (Go) | `GOTOOLCHAIN=go1.26.5 go test ./internal/query/... -run TestFileSymbols -race` | ❌ W0 | ⬜ pending |
| 05-06 T3 | 05-06 | 6 | GRF-03 | T-05-29, T-05-31 | Method count 12→13; verb fixture byte-unchanged; both rejection cases assert invalid-argument | unit (Go) | `GOTOOLCHAIN=go1.26.5 go test ./internal/uiserver/... -run 'TestUIService\|TestFileSymbols'` | ❌ W0 | ⬜ pending |
| 05-07 T1 | 05-07 | 7 | GRF-03 | T-05-36 | Symbol element ids cannot collide with a file or directory path | unit (vitest, DOM-free) | `cd web && pnpm test -- graph-expand` | ❌ W0 | ⬜ pending |
| 05-07 T2 | 05-07 | 7 | GRF-03 | T-05-33, T-05-34, T-05-35 | One request per file; no duplicate children; truncation stated in both directions | unit (vitest, Cytoscape headless) | `cd web && pnpm test -- graph-expand` | ❌ W0 | ⬜ pending |
| 05-07 T3 | 05-07 | 7 | GRF-02, GRF-03, GRF-05 | T-05-37, T-05-38 | Shipped bundle matches its source; whole view confirmed in a real browser on two repositories | full suite + manual | `task web:drift && cd web && pnpm test` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/query/filegraph_test.go` — ENG-03: aggregation correctness, `contains`
      exclusion (D-03), self-edge exclusion, and the D-08 `"package"`-pseudo-node exclusion.
      **The package-node case MUST be tested against this repository's own index** — `google/guava`
      has zero such nodes, so the measurement corpus structurally cannot catch it.
- [ ] `internal/query/filegraph_cycles_test.go` — GRF-04 server-side detection. Must include a
      negative control: a 1-node non-cycle is NOT flagged.
- [ ] `web/tests/file-graph-transform.test.ts` — GRF-02 compound-parent construction, DOM-free.
- [ ] `web/tests/graph-canvas.test.ts` — GRF-04 cycle-class assignment via headless Cytoscape,
      DOM-free.
- [ ] Possible `web/tests/setup.ts` extension — a scoped `offsetWidth`/`offsetHeight` stub keyed
      to a `file-graph-canvas` testid, mirroring the existing `data-table-scroll` stub, **only**
      if a non-headless rendering test proves unavoidable. Pitfall 4's recommendation is to
      avoid needing it at all.
- [ ] No test-framework install needed — vitest and Go stdlib are both already configured.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Click-to-expand actually expands a file into its symbols in place | GRF-03 | jsdom cannot exercise Cytoscape's real `tap` gesture recognition or canvas hit-testing | Build the binary, run `codegraph ui --no-open`, drive `/graph` with agent-browser: click a file node, assert child symbol nodes appear within the same view |
| The graph reads as hierarchical rather than as a hairball; pan/zoom stays smooth | GRF-01, GRF-02 | Layout quality and frame time are visual/temporal properties with no jsdom equivalent | Live browser session against the largest indexed corpus; record time-to-first-paint and pan/zoom frame time against the pre-locked threshold |
| `/graph` no longer renders planning vocabulary | GRF-05 (shipped copy) | **`rg -i "phase 5" web/src/routes/graph/` is a cheap pre-check, NOT a substitute.** No guard exists anywhere in this codebase against leaked planning vocabulary in rendered copy — Phase 4 shipped the identical defect on `/workbench` past five machine-verification layers | View the rendered page in a live browser and read the actual subtitle text |

> **This table is load-bearing.** The phase's own `<specifics>` mandate a live browser UAT
> because Phase 4 demonstrated that machine verification alone is categorically insufficient
> for a rendered-copy defect of exactly this class. Do not treat these rows as optional.

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] Every zero-count assertion paired with a positive control (rule `84d1gfpywd`)
- [ ] Every upper bound paired with a non-zero lower bound (Phase 4 corollary — `<= N` alone is satisfied by 0)
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
