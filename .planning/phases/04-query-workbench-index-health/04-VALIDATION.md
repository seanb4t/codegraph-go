---
phase: 4
slug: query-workbench-index-health
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-29
---

# Phase 4 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Seeded by plan-phase §5.5 from `04-RESEARCH.md` §Validation Architecture, before
> plans exist. The Per-Task Verification Map is filled once PLAN.md task IDs are
> assigned; `/gsd-validate-phase` sets `status: validated`.

---

## Test Infrastructure

This phase spans **two** test stacks. Both must be green.

| Property | Frontend | Backend |
|----------|----------|---------|
| **Framework** | Vitest `4.1.11` + `@testing-library/svelte@5.4.2` (`web/package.json:29`) | Go `testing` (stdlib, no third-party framework) |
| **Config file** | `web/vite.config.ts` — the `test:` block lives inline via a `/// <reference types="vitest/config" />` directive; there is deliberately **no** separate `vitest.config.ts` (03-01 pattern: keeps test config inside the BLD-03 digest) | none (stdlib) |
| **Test location** | `web/tests/*.test.ts` — **not** `web/src/**/*.test.ts`; 16 existing files follow this convention | `internal/**/[name]_test.go`, package-local |
| **Quick run** | `cd web && pnpm test` (= `vitest run`, `web/package.json:14`) | `go test ./internal/uiserver/... ./internal/query/... ./internal/gitmeta/...` |
| **Full suite** | `task web:test` (asserts a positive executed-test count before judging pass/fail — 03-01's D3) | `task test:unit` (`Taskfile.yml:117-133`) |
| **Estimated runtime** | ~5–15s | ~30–60s |

> **Toolchain note:** `GOTOOLCHAIN=go1.26.5` is required locally — go1.27 breaks the
> `cockroachdb/swiss` build (memory `fmss86zf82`). CI is unaffected (`go-version-file` pins 1.26).

---

## Sampling Rate

- **After every task commit:** `cd web && pnpm test` (frontend changes) · `go test ./internal/uiserver/... ./internal/query/...` (backend changes)
- **After every plan wave:** `task test:unit` **and** `cd web && pnpm test`
- **Before `/gsd-verify-work`:** `task proto:drift`, `task web:drift`, the new `task web:components:drift`, full `task test:unit`, and `cd web && pnpm test` all green
- **Max feedback latency:** ~60s (backend full suite dominates)

> **Scoring discipline (01-CONTEXT.md D-04, as corrected):** a verify gate must honor
> **both** the command's exit status **and** a named `--- PASS` / executed-test count
> floor. Exit status alone is vacuous — `go test -run PATTERN` exits 0 when the pattern
> matches nothing. Capture status before any pipe (`set -o pipefail` / `${PIPESTATUS[0]}`).

---

## Per-Task Verification Map

*Seeded before planning — task IDs do not exist yet. The planner MUST populate this
table as it assigns task IDs, and every task must land in it.*

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 04-01-T1 | 04-01 | 1 | WRK-04 | T-04-SC | `[SUS]` npm/CLI legitimacy decided by a human before install; never auto-approvable | checkpoint | *(blocking-human; answer recorded in SUMMARY)* | n/a | ⬜ pending |
| 04-01-T2 | 04-01 | 1 | WRK-03, WRK-04 | T-04-02, T-04-03 | limit passes through to the server bound, no client-side duplicate; URL round-trip fidelity | component (tracer) | `cd web && pnpm exec vitest run --reporter=json --outputFile=/tmp/wb-tracer.json tests/workbench-tracer.test.ts` + count floor 5 | ❌ new | ⬜ pending |
| 04-01-T3 | 04-01 | 1 | WRK-04 | T-04-01 | vendored source carries no raw-HTML sink; failure kinds pairwise distinct | unit + source review | `cd web && pnpm exec vitest run … tests/workbench-url.test.ts tests/workbench-failure.test.ts` + count floor 12 | ❌ new | ⬜ pending |
| 04-02-T1 | 04-02 | 1 | WRK-02 | T-04-06 | glob regression asserted in BOTH directions before the fix | Go unit (RED) | `GOTOOLCHAIN=go1.26.5 go test ./internal/query/... -run TestFilesPatternRecursiveGlob -v` + subtest count ≥5 + non-zero exit | ✓ extends `files_status_test.go` | ⬜ pending |
| 04-02-T2 | 04-02 | 1 | WRK-02 | T-04-SC-GO, T-04-05 | pre-scan sanity check retained; no golden or frozen transcript changes | Go unit (GREEN) | `GOTOOLCHAIN=go1.26.5 go test ./internal/query/... ./internal/cli/... ./internal/mcp/...` + `ok` count ≥3 | ✓ | ⬜ pending |
| 04-02-T3 | 04-02 | 1 | WRK-02 | — | stale comment asserting a fixed bug is retired | file assertion | `test -f .planning/todos/completed/…files-rpc-pattern-glob…md` + resolution-record grep | n/a | ⬜ pending |
| 04-03-T1 | 04-03 | 1 | HLT-01, HLT-02, HLT-03 | T-04-09 | one-way proto field numbering frozen by a human; host-path exposure acknowledged | checkpoint | *(blocking-human; decision recorded in SUMMARY)* | n/a | ⬜ pending |
| 04-03-T2 | 04-03 | 1 | HLT-01 | T-04-08 | read-only method set stays exactly the read set; `mutatingVerbs` untouched | Go unit | `GOTOOLCHAIN=go1.26.5 go test ./internal/uiserver/... -run 'TestUIServiceMethodSetIsExactlyTheReadSet\|TestUIServiceDeclaresNoMutatingMethod' -v` + PASS count ≥2 | ✓ needs literal update | ⬜ pending |
| 04-03-T3 | 04-03 | 1 | HLT-01, HLT-03 | T-04-10, T-04-11 | one engine open per call; commit SHA validated at the read site; no degrade-and-answer | Go unit | `GOTOOLCHAIN=go1.26.5 go test ./internal/uiserver/... -run TestGetHealth -v` + PASS count ≥6 | ❌ new `health_test.go` | ⬜ pending |
| 04-04-T1 | 04-04 | 2 | WRK-04 | T-04-SC-TABS | vendored tabs source reviewed with a positive control; lockfile consistent | source review | `test $(ls web/src/lib/components/ui/tabs \| wc -l) -eq 5 && pnpm install --frozen-lockfile && pnpm check` | n/a | ⬜ pending |
| 04-04-T2 | 04-04 | 2 | WRK-01, WRK-04 | T-04-13, T-04-14, T-04-15 | depth passes through unbounded; no navigation on control change; per-dispatch abort | component | `cd web && pnpm exec vitest run … tests/workbench-impact.test.ts` + count floor 8 | ❌ new | ⬜ pending |
| 04-04-T3 | 04-04 | 2 | WRK-03, WRK-04 | T-04-13, T-04-16 | four failure kinds render four provably distinct strings (Set size 4) | component | `cd web && pnpm exec vitest run … tests/workbench-callers-callees.test.ts` + count floor 7 | ❌ new | ⬜ pending |
| 04-05-T1 | 04-05 | 2 | HLT-01, HLT-02 | T-04-17 | no second verdict function; blank-roots mismatch is FALSE | unit | `cd web && pnpm exec vitest run … tests/health-view.test.ts` + count floor 8 | ❌ new | ⬜ pending |
| 04-05-T2 | 04-05 | 2 | HLT-01, HLT-02, HLT-03 | T-04-18, T-04-20 | one gate, one GetHealth call, no timer API, no project/index path rendered | component | `cd web && pnpm check` + health-view suite still green | ❌ new | ⬜ pending |
| 04-05-T3 | 04-05 | 2 | HLT-01, HLT-02, HLT-03 | T-04-17, T-04-21 | verdict precedes numbers (DOM order); warning present AND absent both asserted | component | `cd web && pnpm exec vitest run … tests/health-page.test.ts` + count floor 8 | ❌ new | ⬜ pending |
| 04-06-T1 | 04-06 | 3 | WRK-02 | T-04-22, T-04-23 | min-length + debounce + abort + out-of-order discard | unit | `cd web && pnpm exec vitest run … tests/file-search.test.ts` + count floor 7 | ❌ new | ⬜ pending |
| 04-06-T2 | 04-06 | 3 | WRK-02 | T-04-24 | repeated `file=` both directions; comma-containing path round-trips | component | `cd web && pnpm exec vitest run … tests/workbench-affected.test.ts` + count floor 6 | ❌ new | ⬜ pending |
| 04-06-T3 | 04-06 | 3 | WRK-02, WRK-04 | T-04-25 | four-tab completeness asserted against a module-derived mode set | component | `cd web && pnpm exec vitest run … tests/workbench-affected.test.ts` + count floor 12 | ❌ new | ⬜ pending |
| 04-07-T1 | 04-07 | 4 | WRK-04 | T-04-27 | regeneration determinism PROVEN before the guard depends on it | live probe | *(recorded finding; four questions answered in SUMMARY)* | n/a | ⬜ pending |
| 04-07-T2 | 04-07 | 4 | WRK-04 | T-04-27, T-04-28, T-04-29 | disk-derived subject set; count reported before comparing; RED-proven | Taskfile gate | `task web:components:drift` + printed `compared N vendored component files` + exit 0 | ❌ new target | ⬜ pending |
| 04-07-T3 | 04-07 | 4 | WRK-04 | T-04-30, T-04-31 | 1000-row measurement with a row-count positive control; committed bundle current | component + gate | `cd web && pnpm exec vitest run … tests/data-table-render-cost.test.ts` + count floor 2; `task web:drift` exit 0 | ❌ new | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

### Requirement → Test Map (from research, pre-assigned)

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|--------------|
| WRK-01 | `Impact` depth control updates blast radius without navigation | component | `cd web && pnpm test -- workbench-impact` | ❌ Wave 0 |
| WRK-02 | Multi-file selection drives `Affected` | component | `cd web && pnpm test -- workbench-affected` | ❌ Wave 0 |
| WRK-03 | `Callers`/`Callees` with adjustable limit | component | `cd web && pnpm test -- workbench-callers-callees` | ❌ Wave 0 |
| WRK-04 | Sortable tables + three-way error taxonomy | component | `cd web && pnpm test -- data-table`; extends existing `rpc-errors.test.ts` | ❌ Wave 0 (table) · ✓ extends existing (errors) |
| HLT-01/02/03 | Health RPC, verdict rendering, worktree warning | Go unit + component | `go test ./internal/uiserver/... -run Health`; `cd web && pnpm test -- health-page` | ❌ Wave 0 (both) |
| — (regression) | `readonly_test.go` guards stay green with the 11th rpc | Go unit | `go test ./internal/uiserver/... -run TestUIServiceMethodSetIsExactlyTheReadSet` and `-run TestUIServiceDeclaresNoMutatingMethod` | ✓ exists — **needs literal update**, see below |
| — (regression) | `doublestar` root-level **and** nested glob | Go unit | `go test ./internal/query/... -run TestFiles` | ✓ existing file to extend |

### Load-bearing literal updates (not optional)

`internal/uiserver/readonly_test.go` must change in the **same** commit as the new rpc:

- `wantUIServiceMethods` (lines 31-42) — add the new method name.
- the `10` literal (line 61, `if len(got) != 10`) — becomes `11`.
- **`mutatingVerbs` (lines 89-93) must NOT be touched.** It contains `"Index"`, which is
  why D-01's rpc is named `GetHealth` rather than `GetIndexHealth`. Weakening this list to
  admit a new name would punch a hole in a guard explicitly built not to go vacuous.
- **`proto:drift`'s `nfiles -lt 4` floor (`Taskfile.yml:339-348`) needs NO change** — it
  counts generated *files*, and adding an rpc to the existing `ui.proto` does not change
  the file count. Verified during research; do not "fix" it.

---

## Wave 0 Requirements

- [ ] `web/tests/workbench-url.test.ts` — URL-grammar round-trip, mirroring `browse-url.test.ts`'s existing shape. **Must include a repeated-`file=` case** (D-11's `getAll()` divergence from browse-url's `params.get()`), which is the single easiest thing to get silently wrong in this phase.
- [ ] `web/tests/workbench-*.test.ts` — component tests for the four analysis tabs (WRK-01..04).
- [ ] `web/tests/health-page.test.ts` — component test for the health view (HLT-01..03).
- [ ] `internal/uiserver/health_test.go` (or extend `handlers_test.go`) — Go coverage for the new health handler, mirroring the existing `Callers`/`Callees` handler test shape.
- [ ] Extend `internal/query`'s existing `Files`-pattern test with the root-level **and** nested `doublestar` cases D-14 requires. **Both directions are mandatory** — a nested-only test would pass while root-level search silently regressed, which is the exact failure mode D-14 exists to prevent.

---

## Manual-Only Verifications

Deliberately short. Phase 3 established that "requires a human" is a **testable claim, not
a category** — two of its three `why_human` justifications turned out to be false, and
`getComputedStyle` / DOM-order assertions closed them (memory `c5prbx1mj4`). Anything below
that can be machine-checked **must** be, and the entries here are scoped to the residue.

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Sort order *reads* correctly to a developer scanning a 1000-row table | WRK-04 | Perceptual judgment about legibility at scale, not correctness. The sort itself is fully unit-testable and must be unit-tested. | Run a `Callers` query at `limit=1000`, sort each column, confirm the table stays readable and responsive. |
| Blast-radius change "without leaving the page" *feels* immediate | WRK-01 | Latency perception. The no-navigation property is assertable (URL unchanged, no `goto`); the felt responsiveness is not. | Move the depth control across several values; confirm results update in place with no full-page transition. |

**Explicitly NOT manual — assert these in code:**

- **HLT-02's "above the raw numbers"** is a DOM-order property. Assert the verdict element
  precedes the numeric block via `compareDocumentPosition` or index-in-parent — do not
  file this as a visual judgment.
- **HLT-03's "impossible to miss"** worktree warning: assert presence, distinct role/styling
  hook, and that it renders *before* the numbers. Use the Phase 3 technique — `getComputedStyle`
  distinguishes a loaded theme from an unstyled fallback.
- **WRK-04's three-way error distinction** (not found / index stale / server error): assert
  three *pairwise-different* rendered strings, not merely that each renders something. A
  test asserting each state "shows a message" passes vacuously if all three collapse to one.

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or a Wave 0 dependency
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags (`vitest` not `vitest --watch`)
- [ ] Feedback latency < 60s
- [ ] Every verify gate honors exit status **and** a positive count floor
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
