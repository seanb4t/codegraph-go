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
| TBD | TBD | TBD | ENG-03 | TBD | Rollup aggregates only over the indexed record set; no filesystem read | unit (Go) | `go test ./internal/query/... -run TestFileGraph` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | ENG-03 (D-08) | TBD | `"package"`-kind pseudo-nodes excluded — no phantom `""` file node | unit (Go) | `go test ./internal/query/... -run TestFileGraph` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | GRF-04 | TBD | Cycle detection over the aggregated file graph, server-side | unit (Go) | `go test ./internal/query/... -run TestFileGraphCycles` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | RPC surface (12th rpc) | TBD | `FileGraph` clean against all 19 `mutatingVerbs`; method count 11→12 | unit (Go) | `go test ./internal/uiserver/... -run TestUIService` | ✅ extend `readonly_test.go` | ⬜ pending |
| TBD | TBD | TBD | GRF-02 | TBD | Compound `parent` chains built from `FilePath`; never emits force-directed layout config | unit (vitest, DOM-free) | `pnpm test -- file-graph-transform` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | GRF-04 (client) | TBD | Cycle membership maps to a distinguishing class on the correct nodes | unit (vitest, Cytoscape `headless: true`) | `pnpm test -- graph-canvas` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | GRF-01 | TBD | Measured latency recorded against a threshold locked **before** dispatch | spike (recorded measurement) | see the GRF-01 spike plan | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | GRF-03, GRF-05 | TBD | In-place expansion works; renderer stays behind the swappable seam | manual + unit | see Manual-Only Verifications | ⬜ pending |

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
