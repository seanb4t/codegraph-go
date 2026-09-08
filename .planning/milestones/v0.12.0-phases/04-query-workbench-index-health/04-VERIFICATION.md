---
phase: 04-query-workbench-index-health
verified: 2026-08-30T10:49:12Z
status: passed
score: 5/5 must-haves verified
behavior_unverified: 0
overrides_applied: 0
requirements_traceability:
  - id: WRK-01
    plan: "04-04"
    status: satisfied
  - id: WRK-02
    plan: "04-02, 04-06"
    status: satisfied
  - id: WRK-03
    plan: "04-01, 04-04"
    status: satisfied
  - id: WRK-04
    plan: "04-01, 04-04, 04-06, 04-07"
    status: satisfied
  - id: HLT-01
    plan: "04-03, 04-05"
    status: satisfied
  - id: HLT-02
    plan: "04-03, 04-05"
    status: satisfied
  - id: HLT-03
    plan: "04-03, 04-05"
    status: satisfied
notable_process_finding:
  - "04-REVIEW.md's frontmatter still reads status: issues_found and its re-review section (RR-W-01/02/03, dated 2026-08-30T07:12:03Z) was never re-run after the fix commit that closed all three (4f746d57, 2026-08-30T03:18:20-04:00, which post-dates the re-review — timestamps land out of narrative order because the re-review's UTC stamp and the fix commit's local-time stamp are in different timezones with no shared reference; the git DAG confirms the fix commit is a genuine descendant of the reviewed tree). All three fixes were verified directly against source in this session (see Anti-Patterns / Code-Review Cross-Check below), not taken from the stale doc. This is a documentation-currency gap, not a code gap — flagged for awareness, not blocking."
---

# Phase 4: Query Workbench & Index Health Verification Report

**Phase Goal:** "A developer can run the four graph analyses interactively with their own knobs, read the results as real tables, and tell at a glance whether the index they are reading is worth trusting."
**Verified:** 2026-08-30T10:49:12Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (ROADMAP Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Impact runs interactively with a depth control that changes the blast radius without leaving the page (WRK-01) | ✓ VERIFIED | `web/tests/workbench-impact.test.ts:172-204` drives the real route: asserts rows change (`Node-depth-2`→`Node-depth-4`), the `<h1>Workbench</h1>` DOM node reference is IDENTICAL before/after (`headingAfter === headingBefore`, proving no remount), and `goto` (mocked, spied) is `not.toHaveBeenCalled()`. Independently re-run this session: `pnpm vitest run` → 268/268 pass, includes this file. Depth input (`web/src/routes/workbench/+page.svelte:234-239`) carries no `max` attribute and no client-side clamp (`rg "clamp\|Math.min\|Math.max\|max="` → 0 hits in +page.svelte/workbench-url.ts). |
| 2 | Multi-file selection shows what files affect (WRK-02); Callers/Callees have an adjustable, unbounded result limit (WRK-03) | ✓ VERIFIED | WRK-02: `web/src/lib/components/workbench/FilePicker.svelte` + `web/tests/workbench-affected.test.ts:133-228` — chips add/dedupe/remove, URL round-trips via repeated `file=` (`getAll('file')` at `workbench-url.ts:77`, never `.get`), a comma-containing path round-trips intact (proves no delimiter-encoding, D-11). Nested-file search depends on 04-02's recursive-glob fix; re-ran `TestFilesPatternRecursiveGlob` this session (`go test -run TestFilesPatternRecursiveGlob -v ./internal/query/...`) — all 7 subtests PASS (nested, root_level, non_recursive_unchanged, brace_alternation, malformed_refused, refusal_precedes_scan, escaped_metacharacter_is_literal). WRK-03: `web/tests/workbench-callers-callees.test.ts:91-125` proves `limit=5`→`limit=50` reaches the stubbed RPC unchanged and re-runs without navigation; no clamp found in the limit input (`workbench-limit-input`, `+page.svelte:249-254`, no `max` attribute). |
| 3 | Workbench results render as structured, sortable tables; failures are distinguishable by kind — not-found / index-stale / server-error, etc. (WRK-04) | ✓ VERIFIED | Sorting: `web/src/lib/components/workbench/DataTable.svelte` composes shadcn `Table.*` + `@tanstack/svelte-table`'s `createTable`/`FlexRender` (no hand-rolled sort). Failure taxonomy: `web/src/lib/workbench-failure.ts` returns 4 distinct `kind`s (`not-found`, `invalid-input`, `index-stale`, `server-error`) over 5 distinct rendered titles (the `no-index` sub-branch of `index-stale` has its own title, verified distinct from `indexing`'s title by `workbench-failure.test.ts`'s WR-01 test). Pairwise-distinctness is asserted programmatically: `workbench-failure.test.ts` — `new Set(titles).size === 4` (4-of-4 sample) plus a direct not-equal check for the 5th title pair. End-to-end: `workbench-callers-callees.test.ts:260-296` — `WRK-04 criterion 3` test drives 4 real rejections through the mounted route and asserts `new Set(renderedTestIds).size === 4` AND `new Set(renderedTitles).size === 4` — collapsing to one message would fail this test. |
| 4 | Index freshness, per-language file counts, and node/edge counts are readable from one health view (HLT-01) | ✓ VERIFIED | `web/src/routes/health/+page.svelte:96-118` renders freshness (`health-freshness`, commit SHA, schema version, reindex-recommended) plus three `CountTable` instances (`filesByLanguage`, `nodesByKind`, `edgesByKind`) sourced from one `GetHealth` call. `GetHealthResponse` (11-field-plus wire, `ui.proto:701-760`) carries exactly these; "coverage" is deliberately scoped to per-language file counts per D-03 (capability matrix explicitly deferred) — matches shipped code, not a gap. `readonly_test.go`'s `wantUIServiceMethods` confirms `GetHealth` is the 11th read-only rpc; re-ran `TestUIServiceMethodSetIsExactlyTheReadSet`/`TestUIServiceDeclaresNoMutatingMethod`/`TestGetHealth*` this session (`GIT_CONFIG_GLOBAL=/dev/null go test ./internal/uiserver/...` — see note below) — all PASS. |
| 5 | Staleness reads as a trust verdict above the raw numbers (HLT-02); worktree mismatch is a loud, unmissable warning (HLT-03) | ✓ VERIFIED | HLT-02: `web/tests/health-page.test.ts:110-115` asserts `verdict.compareDocumentPosition(freshness) & Node.DOCUMENT_POSITION_FOLLOWING` — a genuine DOM-order property, not text-order inference. `TrustVerdict` renders unconditionally above the `{#if pageState.kind === 'failed'}...{:else}` numeric block in `+page.svelte:69-118`. HLT-03: `health-page.test.ts:119-136` asserts `role="alert"` + both roots named + `warning.compareDocumentPosition(verdict) & DOCUMENT_POSITION_FOLLOWING` (warning precedes verdict); `:138-143` asserts ABSENCE on a clean tree (`queryByTestId(...)` returns null) — both directions tested, so a permanently-on warning would fail. Read `WorktreeMismatchWarning.svelte` directly: `border-2 border-red-600 bg-red-100 text-red-900 font-semibold`, `role="alert"` — visually and semantically distinct from `TrustVerdict`'s `role="status"` (5 occurrences, no red). `classifyStatus` remains 5-verdict/2-presence-flag (`status.ts:27-38`), one function, `commitSha` added additively. |

**Score:** 5/5 truths verified (0 present-but-behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `web/src/lib/workbench-url.ts` | Workbench URL grammar (mode/symbol/file[]/depth/limit) | ✓ VERIFIED | `getAll('file')` at line 77; round-trip tests pass |
| `web/src/lib/workbench-failure.ts` | 4-kind failure taxonomy composed over classifyRpcError | ✓ VERIFIED | Composes, does not redefine, `classifyRpcError` |
| `web/src/lib/components/workbench/DataTable.svelte` | Generic sortable table shell | ✓ VERIFIED | `generics="TRow extends RowData"`, reused by CountTable (04-05) and all 4 `*-columns.ts` |
| `web/src/lib/components/workbench/AnalysisPanel.svelte` | Shared per-analysis state machine | ✓ VERIFIED | idle/loading/loaded/failed states, `AnalysisResult<TSummary>` contract |
| `internal/query/files.go` | Recursive-glob Files matcher | ✓ VERIFIED | `doublestar.Match` at both call sites (lines 154, 177) |
| `internal/uiproto/uiv1/ui.proto` | `GetHealth` rpc + response messages | ✓ VERIFIED | 11th read-only method, frozen 16-field response |
| `web/src/lib/health-view.ts` | toCountRows, describeFreshness, hasWorktreeMismatch | ✓ VERIFIED | All three consumed by `/health` route |
| `web/src/lib/components/health/{TrustVerdict,WorktreeMismatchWarning,CountTable}.svelte` | Health view components | ✓ VERIFIED | All wired into `/health` in correct DOM order |
| `web/src/lib/debounced-rpc.ts` | Single debounce/abort/identity mechanism | ✓ VERIFIED | Imported by both `search.ts` and `file-search.ts` |
| `web/src/lib/components/workbench/FilePicker.svelte` | Multi-file chip picker | ✓ VERIFIED | Search-and-add, removable chips, props-in/callback-out |
| `Taskfile.yml` (`web:components:drift`, `web:render-cost`) | Drift guard + opt-in render-cost measurement | ✓ VERIFIED | Both independently re-run this session, PASS |
| `.github/workflows/components-drift.yml` | Schedule/dispatch-only drift workflow | ✓ VERIFIED | No `pull_request` trigger present |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `workbench-url.ts` | `browse-url.ts` | imports `parseShapeInteger`/`isShapeInteger` | ✓ WIRED | No second integer-shape grammar |
| `DataTable.svelte` | `@tanstack/svelte-table` + shadcn `Table.*` | `createTable`, `FlexRender` | ✓ WIRED | Generic, not typed to `Location` |
| `workbench-failure.ts` | `rpc-errors.ts` | `classifyRpcError` | ✓ WIRED | One classifier in the tree (grep-confirmed) |
| `/workbench` route | `workbench-url.ts` | `parseWorkbenchParams(page.url.searchParams)` | ✓ WIRED | Sole reader of the query string |
| `status.ts` `ROUTE_LOCAL_PARAMS` | `navigationIdentity` | route-scoped exclusion table | ✓ WIRED | `T-04-32` GetStatus-quiescence test passes |
| `/health` route | shared `statusGate` (context) | `getContext<StatusGate>('statusGate')` | ✓ WIRED | No second `GetStatus` call issued |
| `/health` route | `uiClient.getHealth` | one call per view-open, AbortController-scoped | ✓ WIRED | `+page.svelte:41-56` |
| `CountTable.svelte` | `DataTable.svelte` | generic instantiation at `TRow = CountRow` | ✓ WIRED | No second table implementation |
| `file-search.ts` / `search.ts` | `debounced-rpc.ts` | `createDebouncedRpc` (both) | ✓ WIRED | One mechanism, configured twice |
| `+page.svelte` (Affected tab) | `AnalysisPanel.svelte` | fourth call site of shared panel | ✓ WIRED | Uses `AnalysisResult{rows,summary}` contract |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|---------------------|--------|
| Callers/Callees/Impact/Affected tables | `Location[]` rows | Real Connect RPC (`uiClient.callers/callees/impact/affected`) | Yes — server-round-trip, no static fallback | ✓ FLOWING |
| `/health` count tables | `filesByLanguage`/`nodesByKind`/`edgesByKind` | `uiClient.getHealth()` → `internal/query.StatusResult` (via `healthToProto`) | Yes | ✓ FLOWING |
| `/health` trust verdict | `IndexStatus` | Shared `statusGate` (per-navigation `GetStatus`) | Yes | ✓ FLOWING |
| `/health` worktree warning | `worktreeMismatch` | `GetHealthResponse.worktree_mismatch` ← `Engine.WorktreeMismatch` (once-per-Engine latch) | Yes | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Web unit test suite | `pnpm vitest run` (cwd `web/`) | `Test Files 28 passed (28)` / `Tests 268 passed (268)` | ✓ PASS |
| TypeScript type-check | `pnpm check` (cwd `web/`) | `1020 FILES 0 ERRORS 0 WARNINGS 0 FILES_WITH_PROBLEMS` | ✓ PASS |
| Recursive-glob fix (WRK-02 dependency) | `go test -run TestFilesPatternRecursiveGlob -v ./internal/query/...` | 7/7 subtests PASS | ✓ PASS |
| GetHealth rpc surface + guards | `GIT_CONFIG_GLOBAL=/dev/null GOTOOLCHAIN=go1.26.5 go test -run "TestUIServiceMethodSetIsExactlyTheReadSet\|TestUIServiceDeclaresNoMutatingMethod\|TestGetHealth" ./internal/uiserver/...` | `ok` (all subtests pass) | ✓ PASS |
| Web asset drift | `task web:drift` | `PASS — hashed 103 source files, manifested 31 output files, committed web/build/ matches both digests` | ✓ PASS |
| Proto drift | `task proto:drift` | `all 4 generated files byte-identical` | ✓ PASS |
| Component vendoring drift | `task web:components:drift` | `PASS — all 50 vendored component files across 8 components byte-identical to shadcn-svelte@1.5.1's regeneration` | ✓ PASS |

**Note on the Go test run:** the first two attempts (`go test ./internal/uiserver/...` under this session's ambient global git config) failed with `1Password: agent returned an error` / `failed to write commit object` — a **local-environment artifact**: the sandbox cannot reach the 1Password SSH-signing agent the global `~/.gitconfig` (`commit.gpgsign=true`, `gpg.format=ssh`) requires for `git commit` calls the health tests make against scratch repos. Re-ran with `GIT_CONFIG_GLOBAL=/dev/null` (bypasses the global config file for that process only, no file mutated) — all tests pass in 1.6s. Confirmed not a code defect: `git config --global --get commit.gpgsign` → `true`, `gpg.format` → `ssh`.

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|-------------|-----------------|--------------|--------|----------|
| WRK-01 | 04-04 | Interactive Impact depth control | ✓ SATISFIED | `workbench-impact.test.ts:172-204`, no-remount + no-goto proof |
| WRK-02 | 04-02, 04-06 | Multi-file selection → Affected | ✓ SATISFIED | `FilePicker.svelte`, `workbench-affected.test.ts`, recursive-glob fix in `files.go` |
| WRK-03 | 04-01, 04-04 | Adjustable Callers/Callees limit | ✓ SATISFIED | `workbench-callers-callees.test.ts:91-125`, unbounded reach to RPC |
| WRK-04 | 04-01, 04-04, 04-06, 04-07 | Sortable tables + distinct failure kinds | ✓ SATISFIED | `DataTable.svelte` (tanstack sort), `workbench-failure.ts` (4 kinds), `workbench-callers-callees.test.ts:260-296` (4-way distinctness at the route) |
| HLT-01 | 04-03, 04-05 | Freshness, file/node/edge counts in one view | ✓ SATISFIED | `GetHealth` rpc, `/health` route renders all named values |
| HLT-02 | 04-03, 04-05 | Verdict above numbers | ✓ SATISFIED | `compareDocumentPosition` assertion, `health-page.test.ts:110-115` |
| HLT-03 | 04-03, 04-05 | Loud worktree-mismatch warning | ✓ SATISFIED | `role="alert"`, red styling, both-directions absence/presence test |

No orphaned requirements: `.planning/REQUIREMENTS.md` maps exactly WRK-01..04 and HLT-01..03 to Phase 4; all 7 IDs appear in at least one plan's `requirements:` frontmatter field, and no plan claims an ID outside this set.

### Anti-Patterns Found

None blocking. Scanned all key modified files under this phase (`workbench-url.ts`, `workbench-failure.ts`, `DataTable.svelte`, `AnalysisPanel.svelte`, `health-view.ts`, `debounced-rpc.ts`, `file-search.ts`, both `+page.svelte` routes, `files.go`, `handlers.go`, `ui.proto`, `Taskfile.yml`) for `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER`/"not yet implemented"/"coming soon": zero real hits (one false-positive `XXXXXX` mktemp template in `Taskfile.yml`, one comment in `health/+page.svelte` referring to a placeholder this phase *filled*, not a live one).

### Code-Review Cross-Check (REVIEW.md staleness)

`04-REVIEW.md`'s frontmatter (`status: issues_found`) and its re-review section (dated `2026-08-30T07:12:03Z`) record **3 new WARNINGs** (RR-W-01 aria-rowcount off-by-one, RR-W-02 contradictory "No results." rendering, RR-W-03 vacuous upper-bound-only test guard) that were **not yet fixed at the time that document was written**. A subsequent commit, `4f746d57` ("fix(04): close re-review warnings RR-W-01/02/03"), fixed all three — but `04-REVIEW.md` was never updated to reflect it (no re-re-review commit exists). Verified directly against current source rather than trusting either document:

- RR-W-01: `DataTable.svelte:128` — `aria-rowcount={table.getRowModel().rows.length + 1}` (header now counted). FIXED.
- RR-W-02: `FilePicker.svelte:113-115` — `Command.Empty` now gated `{#if !searchState.failure}`. FIXED.
- RR-W-03: `data-table-virtualization.test.ts:80-81` — `expect(domRows.length).toBeGreaterThan(0)` paired with the pre-existing upper bound, closing the vacuous-pass. FIXED (not the exact literal-list assertion the review suggested, but discriminates correctly).

This is recorded as a **process/documentation-currency finding**, not a code gap — the phase's actual delivered code has zero open findings from the deep review + re-review cycle.

### Human Verification Required

None. Every must-have in this phase was resolvable by direct code inspection, an independently re-run automated test, or an independently re-run Taskfile/drift gate. The two properties that most commonly hide behind "looks fine to the eye" — DOM ordering (HLT-02) and visual loudness (HLT-03) — are both asserted programmatically (`compareDocumentPosition`, `role` attribute + Tailwind class inspection) rather than deferred to a human judgment call.

### Gaps Summary

No gaps. All 5 ROADMAP success criteria are independently verified against the codebase (not SUMMARY.md claims), all 7 requirement IDs are traced to satisfying evidence with no orphans, the 35-threat security register closes at `threats_open: 0`, and the phase's own deep code-review cycle (2 critical, 8 warning, 9 info → re-review 8/10 correct + 3 new warnings) closes to zero open findings once the final fix commit (`4f746d57`) is checked against source rather than against the review document, which was not updated after that commit landed.

---

_Verified: 2026-08-30T10:49:12Z_
_Verifier: Claude (gsd-verifier)_
