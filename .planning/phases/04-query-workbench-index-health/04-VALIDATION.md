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

### The ONE correct shell shape for a status-AND-floor gate

Cycle-1 cross-AI review found eleven frontend gates written in a shape that **reads**
correct and **executes** wrong. Both shapes are recorded here so the difference is
never re-introduced by pattern-matching on the surrounding prose.

**BROKEN — never write this.** A POSIX compound's status is the LAST command's, so the
floor check's non-zero exit is discarded and only the first command's status survives:

```sh
<cmd>; RC=$?; node -e '…process.exit(1)…'; test "$RC" -eq 0
```

**CORRECT — the floor check and the status check are ONE chained final command**, so a
failure in either is the compound's status:

```sh
<cmd>; RC=$?; node -e '…process.exit(1)…' && test "$RC" -eq 0
```

**Executed proof (run 2026-08-29, recorded so this is evidence and not an argument):**

| Shape | Command | Exit |
|---|---|---|
| broken, floor fails | `bash -c 'true; RC=$?; (exit 1); test "$RC" -eq 0'` | **0** — floor swallowed |
| correct, floor fails | `bash -c 'true; RC=$?; (exit 1) && test "$RC" -eq 0'` | **1** |
| correct, command fails | `bash -c 'false; RC=$?; (exit 0) && test "$RC" -eq 0'` | **1** |
| correct, both pass | `bash -c 'true; RC=$?; (exit 0) && test "$RC" -eq 0'` | **0** |

The Go-side gates in this phase already chain with `&&` and are correct as written
(`04-02` T1/T2, `04-03` T2/T3, `04-07` T2). Every frontend gate in `04-01`, `04-04`,
`04-05`, `04-06` and `04-07` T3 was rewritten to the CORRECT shape in the cycle-1
revision pass. **04-01 Task 2 carries the one-time obligation to re-run the four-row
demonstration above and record it verbatim in `04-01-SUMMARY.md`**, so the shape this
phase's gates depend on is proven discriminating in this environment, not assumed.

---

## Per-Task Verification Map

*Seeded before planning — task IDs do not exist yet. The planner MUST populate this
table as it assigns task IDs, and every task must land in it.*

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 04-01-T1 | 04-01 | 1 | WRK-04 | T-04-SC | `[SUS]` npm/CLI legitimacy decided by a human before install; never auto-approvable | checkpoint | *(blocking-human; answer recorded in SUMMARY)* | n/a | ⬜ pending |
| 04-01-T2 | 04-01 | 1 | WRK-03, WRK-04 | T-04-02, T-04-03, T-04-32 | limit passes through to the server bound, no client-side duplicate; URL round-trip fidelity; a Workbench control change mints NO new navigation identity | component (tracer) | `cd web && pnpm exec vitest run --reporter=json --outputFile=/tmp/wb-tracer.json tests/workbench-tracer.test.ts; RC=$?; node -e '…floor 6…' && test "$RC" -eq 0` | ❌ new | ⬜ pending |
| 04-01-T3 | 04-01 | 1 | WRK-04 | T-04-01, T-04-32 | vendored source carries no raw-HTML sink; failure kinds pairwise distinct; negative depth/limit pass through; route-local identity asserted for BOTH /workbench and /browse | unit + source review | `cd web && pnpm exec vitest run … tests/workbench-url.test.ts tests/workbench-failure.test.ts tests/status.test.ts` + count floor 16 (`&&`-chained) | ❌ new | ⬜ pending |
| 04-02-T1 | 04-02 | 1 | WRK-02 | T-04-05, T-04-06 | glob regression asserted in BOTH directions before the fix; refusal proven to precede `IterateFiles` by an instrumented reader; the escape convention 04-06 relies on is matcher-verified | Go unit (RED) | `GOTOOLCHAIN=go1.26.5 go test ./internal/query/... -run TestFilesPatternRecursiveGlob -v` + subtest count ≥7 + non-zero exit | ✓ extends `files_status_test.go` | ⬜ pending |
| 04-02-T2 | 04-02 | 1 | WRK-02 | T-04-SC-GO, T-04-05 | pre-scan sanity check retained; no golden or frozen transcript changes | Go unit (GREEN) | `GOTOOLCHAIN=go1.26.5 go test ./internal/query/... ./internal/cli/... ./internal/mcp/...` + `ok` count ≥3 | ✓ | ⬜ pending |
| 04-02-T3 | 04-02 | 1 | WRK-02 | — | stale comment asserting a fixed bug is retired | file assertion | `test -f .planning/todos/completed/…files-rpc-pattern-glob…md` + resolution-record grep | n/a | ⬜ pending |
| 04-03-T1 | 04-03 | 1 | HLT-01, HLT-02, HLT-03 | T-04-09 | one-way proto field numbering frozen by a human; host-path exposure acknowledged | checkpoint | *(blocking-human; decision recorded in SUMMARY)* | n/a | ⬜ pending |
| 04-03-T2 | 04-03 | 1 | HLT-01 | T-04-08 | read-only method set stays exactly the read set; `mutatingVerbs` untouched | Go unit | `GOTOOLCHAIN=go1.26.5 go test ./internal/uiserver/... -run 'TestUIServiceMethodSetIsExactlyTheReadSet\|TestUIServiceDeclaresNoMutatingMethod' -v` + PASS count ≥2 | ✓ needs literal update | ⬜ pending |
| 04-03-T3 | 04-03 | 1 | HLT-01, HLT-03 | T-04-10, T-04-11 | one engine open per call; commit SHA validated at the read site; no degrade-and-answer | Go unit | `GOTOOLCHAIN=go1.26.5 go test ./internal/uiserver/... -run TestGetHealth -v` + PASS count ≥6 | ❌ new `health_test.go` | ⬜ pending |
| 04-04-T1 | 04-04 | 2 | WRK-04 | T-04-SC-TABS | vendored tabs source reviewed with a positive control; lockfile consistent | source review | `test $(ls web/src/lib/components/ui/tabs \| wc -l) -eq 5 && pnpm install --frozen-lockfile && pnpm check` | n/a | ⬜ pending |
| 04-04-T2 | 04-04 | 2 | WRK-01, WRK-04 | T-04-13, T-04-14, T-04-15, T-04-32 | depth passes through unbounded; no navigation on control change; per-dispatch abort; **`GetStatus` call count unchanged across a depth edit**; `{rows,summary}` result contract carries `nodeCount`/`edgeCount` | component | `cd web && pnpm exec vitest run … tests/workbench-impact.test.ts` + count floor 9 (`&&`-chained) | ❌ new | ⬜ pending |
| 04-04-T3 | 04-04 | 2 | WRK-03, WRK-04 | T-04-13, T-04-16, T-04-32 | four failure kinds render four provably distinct strings (Set size 4); `GetStatus` call count unchanged across a limit edit | component | `cd web && pnpm exec vitest run … tests/workbench-callers-callees.test.ts` + count floor 8 (`&&`-chained) | ❌ new | ⬜ pending |
| 04-05-T1 | 04-05 | 2 | HLT-01, HLT-02 | T-04-17, T-04-22 | no second verdict function; blank-roots mismatch is FALSE; two-snapshot disagreement has a stated display rule AND is computable — `IndexStatus` is widened additively with `commitSha` while `StatusVerdict` keeps five members and `CommitKnowledge` keeps two | unit | `cd web && pnpm exec vitest run … tests/health-view.test.ts` + `… tests/status.test.ts`, SEPARATE floors 8 and 18, both statuses and both floors `&&`-chained as ONE final command | ❌ new (`health-view.test.ts`) + ✓ extends `status.test.ts` | ⬜ pending |
| 04-05-T2 | 04-05 | 2 | HLT-01, HLT-02, HLT-03 | T-04-18, T-04-20 | one gate, one GetHealth call, no timer API, no project/index path rendered; `CountTable` binds the GENERIC `DataTable` over `CountRow` | component | `cd web && pnpm check` + health-view suite still green (`&&`-chained) | ❌ new | ⬜ pending |
| 04-05-T3 | 04-05 | 2 | HLT-01, HLT-02, HLT-03 | T-04-17, T-04-21, T-04-22 | verdict precedes numbers (DOM order); warning present AND absent both asserted; stale-snapshot notice asserted in BOTH directions (the ABSENT direction is what catches a comparison reading the presence flag instead of the SHA) | component | `cd web && pnpm exec vitest run … tests/health-page.test.ts` + count floor 9 (`&&`-chained) | ❌ new | ⬜ pending |
| 04-06-T1 | 04-06 | 3 | WRK-02 | T-04-22, T-04-23 | min-length + debounce + **cleared-timer** (pre-dispatch) + **abort of a genuinely overlapping dispatch** + out-of-order discard; **glob metacharacters escaped**; `search.ts` refactored onto the shared controller with its Phase-3 suite byte-unchanged | unit | `cd web && pnpm exec vitest run … tests/file-search.test.ts tests/debounced-rpc.test.ts tests/search.test.ts` + count floor 8 for file-search (`&&`-chained) | ❌ new | ⬜ pending |
| 04-06-T2 | 04-06 | 3 | WRK-02 | T-04-24 | repeated `file=` both directions; comma-containing path round-trips | component | `cd web && pnpm exec vitest run … tests/workbench-affected.test.ts` + count floor 6 (`&&`-chained) | ❌ new | ⬜ pending |
| 04-06-T3 | 04-06 | 3 | WRK-02, WRK-04 | T-04-25, T-04-32 | four-tab completeness asserted against a module-derived mode set; echoed `files` arrive through the `{rows,summary}` contract; chip edits mint no new navigation identity | component | `cd web && pnpm exec vitest run … tests/workbench-affected.test.ts` + count floor 13 (`&&`-chained) | ❌ new | ⬜ pending |
| 04-07-T1 | 04-07 | 4 | WRK-04 | T-04-27 | regeneration determinism PROVEN before the guard depends on it, over EVERY vendored family (disk-derived), not just this phase's two | live probe | *(recorded finding; five questions answered in SUMMARY)* | n/a | ⬜ pending |
| 04-07-T2 | 04-07 | 4 | WRK-04 | T-04-27, T-04-28, T-04-29 | disk-derived subject set (pathspec WITHOUT a trailing slash — the trailing-slash form returns zero); non-zero population floor asserted before comparing; RED-proven; wired into a non-PR schedule/dispatch workflow so the mitigation actually runs | Taskfile gate + workflow | `task web:components:drift` + printed `compared N vendored component files` + exit 0; `GOTOOLCHAIN=go1.26.5 go test ./internal/upgrade/... -run 'TestWorkflowFilePopulationMatchesDisk\|TestInScopeJobsPopulationMatchesDisk\|TestWorkflowRunBodiesInvokeTask' -v` | ❌ new target + new workflow | ⬜ pending |
| 04-07-T3 | 04-07 | 4 | WRK-04 | T-04-30, T-04-31 | 1000-row measurement with a row-count positive control; **wall-clock thresholds are opt-in via `task web:render-cost`, never asserted inside the PR-required `task web:test`**; committed bundle current | component + gate | `cd web && pnpm exec vitest run … tests/data-table-render-cost.test.ts` + count floor 3 (`&&`-chained); `task web:render-cost` (opt-in); `task web:drift` exit 0 | ❌ new | ⬜ pending |

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

> **Corpus finding (cycle-1 review, verified in-tree 2026-08-29 — do NOT assume the
> shared fixture is sufficient).** `internal/indexer/testdata/gofixture` contains exactly
> `go.mod`, `main.go`, `skip_linux.go`, `pkga/embed.go`, `pkga/pkga.go`, `pkgb/pkgb.go`.
> That is **one** directory level deep and **one** indexed extension (`.go`; `go.mod` is
> not a registered extension, `internal/indexer/discover.go:74-76`). 04-02's RED
> expectations require a **two**-directory-deep path and a **second** extension, so the
> test MUST write those files into the `copyFixture(t)` temp tree **before**
> `indexFixture(t, dir)` runs. `typescript` is a registered language
> (`internal/indexer/languages_typescript.go`), so a `.ts` file is genuinely indexed.

- [ ] `web/tests/debounced-rpc.test.ts` — the extracted debounce/abort/request-identity
      controller 04-06 configures twice (D-15 satisfied structurally, not by sharing two
      constants). `web/tests/search.test.ts` must stay green **byte-unchanged** across
      that refactor; that is the regression proof.

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

**Explicitly NOT a pass/fail assertion inside `task web:test` (cycle-1 review, HIGH):**

- **D-08's render-cost thresholds.** `task web:test` is a PR-triggered REQUIRED CI job
  (`.github/workflows/ci.yml:149`), and a wall-clock jsdom threshold in a required check
  is a flake generator — the same nondeterministic-gate failure mode this phase rejects
  everywhere else. `web/tests/data-table-render-cost.test.ts` therefore asserts only
  DETERMINISTIC structure (1000 rows rendered; a sort toggle genuinely reorders) and
  *prints* the medians. The threshold comparison is opt-in behind
  `RENDER_COST_ASSERT=1` / `task web:render-cost`, which is deliberately in no workflow.
  D-08 is measurement-first; that is exactly what this split preserves.

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
