---
phase: 02-guards-ci-wiring-docs-burn-down
plan: 07
subsystem: planning-bookkeeping
tags: [windows-ledger, state-md, seeds, todos, gsd-tools-gaps]

requires:
  - phase: 02-01
    provides: "02-MUTATION-LOG.md Family (a) — GRD-09 RED replay against 98cd41dd, the evidence for closing WINDOWS #29"
  - phase: 02-05
    provides: "task web:components:drift PASS (50 files/8 components byte-identical), the evidence for closing WINDOWS #31"
  - phase: 02-06
    provides: "docs/RELEASE.md dependency-paragraph rewrite dropping stale counts/mark3labs attribution, the evidence for closing WINDOWS #13"
provides:
  - "WINDOWS.md rows #13, #16, #29, #31, #33 closed `fixed` via `gsd-tools windows fixed <id>`, each with recorded verification evidence"
  - "WINDOWS.md rows #20, #21, #34 explicitly stated as record-only (open by decision) in STATE.md Blockers"
  - "A drafted upstream `open-gsd/gsd-core` issue body reporting four `gsd-tools` bookkeeping gaps"
  - "STATE.md's `### Pending Todos` body regenerated from `gsd-tools init todos`'s renderer, replacing a stale hand-authored table"
  - "One genuinely open todo (`bench pinnedAt` HEAD-only check) filed in the add-todo template shape"
  - "SEED-001 frontmatter records consumption by v0.12.0 in SEED-002's field shape"
affects: [milestone-close-audit, gsd-ship-windows-gate]

actuals:
  tokens: 9813
  tasks: 3
  commits: 3
plan_head_before: 80c45c5c176192bdbb21130878f39e3d97b2f88c

tech-stack:
  added: []
  patterns:
    - "Tool-verb-only edits to tool-owned planning files: every WINDOWS.md/STATE.md/seed-frontmatter change went through a `gsd-tools` verb or a value in a shape the tool already renders, never a hand-authored heading/table/status"

key-files:
  created:
    - .planning/todos/pending/2026-08-14-bench-pinnedat-validates-a-checkout-by-git-rev-parse-head-alone.md
  modified:
    - .planning/WINDOWS.md
    - .planning/STATE.md
    - .planning/seeds/SEED-001-local-svelte-shadcn-graph-browsing-ui.md

key-decisions:
  - "Closed #13/#16/#29/#31/#33 via `gsd-tools windows fixed <id>` with all verification evidence recorded here in the SUMMARY (Pitfall 5: the verb accepts no note/reason parameter) rather than attempted as a CLI argument."
  - "#20, #21, #34 stay `open` — not waived, not fixed — because the ledger has no annotate/record-only verb or status; their record-only nature is stated in this SUMMARY and as a `state add-blocker` bullet in STATE.md, per the planning-artifacts rule against inventing a status value."
  - "STATE.md's hand-authored Pending Todos table (17 stale rows, only 1 actually still open) was replaced wholesale with `gsd-tools init todos`'s literal `pending_todos_markdown` render — filling in the tool's own shape, not inventing one (RESEARCH Pitfall 4)."
  - "The bench `pinnedAt()` HEAD-only-checkout gap (RESEARCH open question 2) is filed as a real pending-todo file rather than folded into Blockers prose, since it is unfixed and belongs in the todo system, not the record-only bucket."
  - "The four `gsd-tools` gaps (no annotate/record-only verb, `windows fixed` accepts no note, no CLI verb creates a pending-todo file, no table-rendering path) are reported upstream via a drafted issue body in this SUMMARY plus a STATE.md Blockers bullet — no issue opened on open-gsd/gsd-core by the executor; that is the maintainer's call."
  - "SEED-001's `consumed_by`/`consumed_on` fields were inserted directly after `planted_during`, mirroring SEED-002's exact field order; `audit_acknowledged` and every other pre-existing field left byte-identical."

requirements-completed: [GRD-14, DOCS-11, GRD-09, GRD-10, DOCS-08]

coverage:
  - id: D1
    description: "WINDOWS #16 and #33 closed via `gsd-tools windows fixed`, each preceded by read-only verification recorded below (GRD-14)"
    requirement: GRD-14
    verification:
      - kind: other
        ref: "rg -n -i 'TS Node process|TS binary|compared fairly' internal/bench/rss.go tools/bench/runner/main.go -> no matches"
        status: pass
      - kind: other
        ref: "gh pr view 71 --json state -> MERGED; gh run view 34859136748's tmux job log -> executed=6 skipped=0 expected=6, no tmux -V mismatch error"
        status: pass
    human_judgment: false
  - id: D2
    description: "WINDOWS #29, #31, #13 closed on the strength of this phase's own prior-plan evidence (GRD-09, GRD-10, DOCS-08)"
    requirement: GRD-09
    verification:
      - kind: other
        ref: "02-MUTATION-LOG.md '## Family (a) — GRD-09' section exists; 02-05-SUMMARY.md records 'task web:components:drift (PASS, all 50 files across 8 components byte-identical)'; docs/RELEASE.md grep for stale counts/mark3labs returns nothing"
        status: pass
    human_judgment: false
  - id: D3
    description: "#20, #21, #34 stay `open` with their record-only status stated in SUMMARY and STATE.md Blockers (no annotate verb exists)"
    requirement: GRD-14
    verification:
      - kind: other
        ref: "gsd-tools windows status -> #20/#21/#34 status=open, unchanged; STATE.md Blockers/Concerns contains the new '#20, #21, #34' bullet"
        status: pass
    human_judgment: false
  - id: D4
    description: "Tooling gaps reported in the repo's established shape (state add-blocker bullet) with a drafted upstream issue body; no issue filed by the executor"
    requirement: GRD-14
    verification:
      - kind: other
        ref: "gh issue list --repo open-gsd/gsd-core --author @me --state open matching gap keywords -> 0"
        status: pass
    human_judgment: false
  - id: D5
    description: "Bench pinnedAt HEAD-only check filed as a pending todo in the add-todo template shape (DOCS-11)"
    requirement: DOCS-11
    verification:
      - kind: other
        ref: "todo file frontmatter/body shape checks (created/title/area/severity/files, ## Problem/## Solution) all pass; git ls-files --error-unmatch succeeds"
        status: pass
    human_judgment: false
  - id: D6
    description: "STATE.md's Pending Todos body regenerated wholesale from `gsd-tools init todos`'s `pending_todos_markdown` (DOCS-11)"
    requirement: DOCS-11
    verification:
      - kind: other
        ref: "diff between STATE.md's rendered section and `pending_todos_markdown` (blank lines stripped) -> no diff; no '| Created | Area' table string remains"
        status: pass
    human_judgment: false
  - id: D7
    description: "SEED-001 records consumption by v0.12.0 in SEED-002's exact field set (DOCS-11)"
    requirement: DOCS-11
    verification:
      - kind: other
        ref: "status/consumed_by/consumed_on/audit_acknowledged/planted field checks all pass; gsd-tools list-seeds implemented lists SEED-001"
        status: pass
    human_judgment: false

duration: ~25min
completed: 2026-09-16
status: complete
---

# Phase 02 Plan 07: Windows Ledger Closure, Tooling-Gap Report, and Bookkeeping Reconciliation Summary

**Closed five stale WINDOWS.md rows through the tool's own verb with recorded evidence, left three genuinely record-only rows untouched and stated as such, regenerated STATE.md's Pending Todos section from the tool's own renderer (replacing a 17-row table that had drifted to 1 real open item), filed the one remaining real todo, recorded SEED-001's consumption, and drafted (without filing) an upstream issue reporting four `gsd-tools` bookkeeping gaps.**

## Performance

- **Duration:** ~25 min
- **Tasks:** 3 completed
- **Files touched:** 4 (1 created, 3 modified)
- **Commits:** 3 (measured via `git rev-list --count 80c45c5c..HEAD`)

## Accomplishments

- **WINDOWS.md**: closed #13, #16, #29, #31, #33 via `gsd-tools windows fixed <id>` (one id per call — the verb accepts no note). `open_count` moved from 11 to 6; #20, #21, #22, #32, #34, #35 remain open, exactly as the plan's must-haves specify.
- **STATE.md Blockers/Concerns**: added two bullets via `gsd-tools state add-blocker` — one naming #20/#21/#34 as open-by-decision record-only rows, one reporting the four `gsd-tools` tooling gaps upstream (drafted issue body below).
- **STATE.md Pending Todos**: replaced the stale hand-authored 4-row table (three already resolved, one genuinely open, plus a "Resolved and filed" table of nine closed items) with the literal one-bullet render `gsd-tools init todos` produces — the tool has no table renderer, only a bullet renderer (RESEARCH Pitfall 4).
- **New pending todo filed**: `.planning/todos/pending/2026-08-14-bench-pinnedat-validates-a-checkout-by-git-rev-parse-head-alone.md`, the one row from the old table that was genuinely still open and unfixed.
- **SEED-001**: `status: dormant` → `status: implemented`, `consumed_by`/`consumed_on` added in SEED-002's field order; `list-seeds implemented` now lists both seeds.

## Task Commits

Each task was committed atomically:

1. **Task 1: Verify, then close, WINDOWS #16, #33, #29, #31 and #13 through the tool verb** — `9f6603a2` (docs)
2. **Task 2: Record-only rows and tooling gaps stated through `state add-blocker`; upstream report drafted** — `3d45ed65` (docs)
3. **Task 3: File the bench pinnedAt todo, regenerate Pending Todos from the tool, record SEED-001 consumption** — `ff500512` (docs)

**Plan metadata:** committed in this same close-out step alongside STATE.md/ROADMAP.md/REQUIREMENTS.md updates.

## Evidence Table (Task 1 — five WINDOWS closures)

| id | claim in the row | verification command | observed result | resolving change |
|----|-------------------|-----------------------|-------------------|-------------------|
| #16 | TS-comparison framing lingers in bench packages, deferred to Phase 6 BENCH-02 | `rg -n -i 'TS Node process\|TS binary\|compared fairly' internal/bench/rss.go tools/bench/runner/main.go` | no matches (exit 1) | v0.11.0 Phase 6 `06-01-SUMMARY.md`, `requirements-completed: [BENCH-02]` — "the two-subject comparison architecture, its `-ts-binary` flag, `resolveTSBinary`/`macOSHomebrewTSBinary`, and `runHeadToHead` are gone" |
| #33 | Task 2's human-check (tmux-e2e version assertion + executed-count line) could not run until a real `ci.yml` run existed on this branch | `gh pr view 71 --json state,mergedAt` → `gh run view 34859136748 --json jobs` → `gh run view --job 104026417872 --log` | PR #71 `MERGED` (2026-09-14T13:50:11Z); job `tmux e2e (real-pty harness, TTY-01..TTY-07)` `conclusion: success`; log shows no `::error::` tmux-version-mismatch line (i.e. `tmux -V` observed `tmux 3.4`, matching `TMUX_EXPECTED_VERSION`) and the line `test:tmux: executed=6 skipped=0 expected=6` | The value was already recorded nowhere new needed — STATE.md line ~285-286 already states "tmux 3.4" from CI run 34658987243; this run reconfirms the same value on the now-merged PR #71, so nothing new was recorded, per the plan's own instruction |
| #29 | `web:drift`'s OUTPUT-half `find`-based enumeration is a gate blind spot; suggested fix declined by the maintainer (D-01/D-02) | `rg -n '^## Family \(a\) — GRD-09' .planning/phases/02-guards-ci-wiring-docs-burn-down/02-MUTATION-LOG.md` | Family (a) section exists with the full RED transcript replaying `98cd41dd`'s exact incident shape and the digest mismatch | 02-01-SUMMARY.md (this phase) |
| #31 | Vendored `button.svelte` drift detected, cause not isolated (local-toolchain hypothesis unconfirmed) | `rg -n -B2 -A2 'byte-identical' .planning/phases/02-guards-ci-wiring-docs-burn-down/02-05-SUMMARY.md` | "task web:components:drift (PASS, all 50 files across 8 components byte-identical) — plan Task 2 `<verify>` gate 1" — registry-side drift confirmed and re-vendored | 02-05-SUMMARY.md (this phase) |
| #13 | `docs/RELEASE.md` states stale dependency counts ("27 direct", "134 total", "remaining 13") and credits `mark3labs/mcp-go` | `rg -n 'mark3labs\|27 direct\|134 total\|remaining 13' docs/RELEASE.md` | no matches (exit 1) | 02-06-SUMMARY.md's § 2 rewrite (this phase), crediting `modelcontextprotocol/go-sdk` and dropping raw counts (D-13) |

Post-closure ledger state (`gsd-tools windows status`): `open_count: 6`; #13/#16/#29/#31/#33 all `status: fixed` with a non-empty `resolved_at`; #20/#21/#34 unchanged at `status: open`.

## Record-only rows (#20, #21, #34) — why they stay `open`

- **#20** (`web/vite.config.ts`): no `svelte.config.js` exists in this SvelteKit toolchain version (`kit ^2.63.0`'s `sv 0.17.0` scaffold) — adapter config lives in `vite.config.ts`'s `sveltekit()` plugin options by design. A Phase-2 deviation-by-design, not a defect.
- **#21** (`web/package.json`): TypeScript pinned to `6.0.3` (the scaffold's own verified-compatible default) rather than the plan's literal `5.9.3` — also a Phase-2 deviation-by-design.
- **#34** (`test/tmux/frame_stability_test.go`): the TTY-06 family (d) mutation-log finding — the D-06-specified `v.AltScreen=false` mutation does not fail `TestInstallPickerFrameStableWhileIdle`, because the assertion measures post-settle idle stability, which converges past the AltScreen-driven settling transient the mutation targets. Confirmed and reproduced twice, documented honestly rather than forced (08-MUTATION-LOG.md family (d)).

None of these are pending work — they are documented, deliberate deviations or an honestly-reported mutation-log finding. `gsd-tools windows` has no "record-only"/"annotate" status, so per the planning-artifacts rule (never invent a shape in a tool-owned file) they stay `open` with their nature stated here and in STATE.md's Blockers/Concerns, rather than being waived (which would misrepresent them as deferred defects) or fixed (nothing was fixed).

## Drafted Upstream Issue (open-gsd/gsd-core) — NOT filed

Per the plan's explicit instruction, the executor does not run `gh issue create`. The following is the ready-to-file body; filing is the maintainer's call.

> **Title:** `windows`/STATE.md bookkeeping: no record-only ledger status, no note on `fixed`, no todo-file verb, and STATE.md's todo renderer only emits bullets
>
> **Body:**
>
> While reconciling `.planning/WINDOWS.md` and `.planning/STATE.md` against ground truth (codegraph-go, phase `02-guards-ci-wiring-docs-burn-down`, plan 07), we hit four gaps in `gsd-tools`' bookkeeping surface. All four forced a "state the gap in prose, cite it here" workaround rather than a clean tool-native fix, per the project's own planning-artifacts rule (never invent structure in a tool-owned file).
>
> 1. **No "record-only"/"annotate" ledger status or verb.** `gsd-tools windows` offers `status | append | waive | fixed` (`gsd-core/bin/lib/broken-windows.cjs`). Some ledger rows are neither "pending work" nor "a defect someone chose not to fix" (waive's semantics) nor "actually fixed" — they are intentional design deviations or an honestly-reported mutation-log finding that will never change state. Today these rows have no home except staying `open` forever with an out-of-band explanation, which erodes the ledger's `open_count` signal over time.
> 2. **`windows fixed <id>` accepts no note/reason.** `markFixed(ledger, id, opts)` (`broken-windows.cjs:286-293`) takes only `{ now }`; `cmdWindowsMarkFixed` (`:1068-1088`) parses exactly one positional, zero flags. Only `markWaived` accepts a `reason` string. Verification evidence for a `fixed` row can currently only live in a plan's SUMMARY.md, disconnected from the ledger entry itself.
> 3. **No CLI verb creates a pending-todo file.** The `/gsd-add-todo` workflow is agent-authored (writes the file by hand per its own `create_file` step); there is no `gsd-tools todo add` (or similar) command an agent or script can invoke non-interactively to file one.
> 4. **STATE.md's Pending Todos renderer only emits bullets, never a table.** `renderPendingTodosMarkdown` (`gsd-core/bin/lib/init.cjs:2024`, consumed via `gsd-tools init todos`'s `pending_todos_markdown` field) always produces "one bullet per todo, each capped at `PENDING_TODO_BULLET_MAX_CHARS`." Any project whose STATE.md predates this convention (or was hand-authored as a table) has no tool-native path back to a richer, multi-column rendering — the fix is always "replace with bullets," which may lose information (severity, resolution notes) that a table could have carried.
>
> **The ask:** an `annotate`/`record-only` status on `windows` (or a `--note` flag accepted by `fixed`), a `gsd-tools todo add` verb, and a STATE.md todo renderer that can express what workflows actually write (or an explicit statement that bullets are the permanent, intentional shape). Happy to provide the exact call sites above if useful for triage.

## Files Created/Modified

- `.planning/todos/pending/2026-08-14-bench-pinnedat-validates-a-checkout-by-git-rev-parse-head-alone.md` — new pending todo, add-todo template shape
- `.planning/WINDOWS.md` — five rows (#13, #16, #29, #31, #33) marked `fixed` via the tool verb
- `.planning/STATE.md` — two new Blockers/Concerns bullets (record-only rows; tooling gaps); Pending Todos body replaced wholesale
- `.planning/seeds/SEED-001-local-svelte-shadcn-graph-browsing-ui.md` — `status`/`consumed_by`/`consumed_on` values added

## Removed Pending-Todos-Table Rows — Disposition (carried forward per RESEARCH Pitfall 4)

Copied here before the table's removal from STATE.md, so the closure notes remain discoverable:

**Pending rows (4), now superseded:**

| Created | Area | Severity | Title | Actual disposition |
|---------|------|----------|-------|---------------------|
| 2026-08-07 | mcp | major | Wire oracle `toolslist-repeat` response ordering flake | Already resolved — `.planning/todos/completed/2026-08-07-wire-oracle-toolslist-repeat-response-ordering-flake.md` exists; root cause fixed via R2 (Phase 03 decision log) |
| 2026-08-10 | ci | — | Add golangci-lint with gofmt and idiomatic Go linters | Already resolved — completed file exists; golangci-lint added in an isolated tool modfile (Phase 3 decisions) |
| 2026-08-14 | bench | — | `pinnedAt()` validates a checkout by `git rev-parse HEAD` alone | **Genuinely still open** — filed as a real pending-todo file by this plan (see above) |
| — | mcp | major | CR-01 `pendingWriter` counter corruption | Already resolved — shipped as v0.12.0 Phase 1's `FIX-01` |

**"Resolved and filed" rows (9), already in `.planning/todos/completed/` and now simply no longer duplicated in STATE.md's stale table:**

| Resolved | Area | Title |
|----------|------|-------|
| 2026-07-28 | docs | Document release procedures — closed by 09-04's `docs/RELEASE-PROCEDURES.md` rewrite |
| 2026-07-31 | perf | Bisect the indexer throughput regression — REFUTED |
| 2026-07-31 | perf | Rebless perf baseline on ubuntu-latest — DONE |
| 2026-08-13 | agents | Author a codegraph usage skill for agents — closed by v0.10.0 Phases 6–8 |
| 2026-09-08 | release | `dry-run-signed` additions-only diff guard passes vacuously — closed by Phase 7 GRD-03 |
| 2026-09-08 | ci | post-release-verify conclusion guard has no regression assertion — closed by Phase 7 GRD-04 |
| 2026-09-08 | ci | Tap App secret distinctness test is tautological — closed by Phase 7 GRD-05 (test deleted) |
| 2026-09-13 | docs | `brew trust` broader-tap-grant wording — closed by Phase 12 DOCS-07 |
| 2026-09-13 | testing | `internal/graphstore/archtest` ignored per-package load errors — closed by quick task 260913-pkp |

## Decisions Made

See `key-decisions` in frontmatter.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - over-broad verify assertion] Task 2's second automated `<verify>` command asserts zero removed lines across the entire STATE.md diff, but `gsd-tools state add-blocker` unconditionally bumps the `last_updated` and `state_head` frontmatter fields on every invocation**
- **Found during:** Task 2
- **Issue:** The literal command `test "$(rg -c '^-[^-]' /tmp/02-07-state.diff || echo 0)" = "0"` fails because the commit's diff always contains 2 removed/2 added frontmatter lines (`last_updated`, `state_head`) — confirmed as universal, expected tool behavior by inspecting every prior 02-0X plan-completion commit in this phase (e.g. `80c45c5c`, `40dec267`, `67c9f493`), all of which show the identical frontmatter-only diff shape whenever STATE.md is touched (`state.cjs:2842` unconditionally sets `last_updated` on every write).
- **Fix:** Re-ran the same check scoped to the diff hunk covering the Blockers/Concerns section body only (excluding the frontmatter hunk): `sed -n '/^@@ -381/,$p'` isolates the body hunk, which shows exactly 0 removed lines and exactly 2 added `- ` bullet lines — matching the gate's actual intent (no hand-edited content removed, exactly two verb-added bullets). The literal unscoped command's failure is reported here rather than silently reinterpreted; both the literal and the scoped results are shown in this deviation for the record.
- **Files modified:** None (verification-only; no code or content change was needed — the STATE.md content itself is correct and matches every other acceptance criterion for the task)
- **Verification:** `git show --format= HEAD -- .planning/STATE.md | sed -n '/^@@ -381/,$p'` → 0 removed lines, 2 added `- ` lines (see Task 2 verify transcript above)
- **Commit:** `3d45ed65` (the content itself; no separate fix commit needed)

---

**Total deviations:** 1 auto-fixed (Rule 1 — verify-gate scope, not a code defect)
**Impact on plan:** No functional impact. The plan's actual intent (two clean verb-added bullets, no content removed) is fully met; only an over-broad automated grep in the plan's own `<verify>` block needed a scoped re-interpretation, which is documented above rather than silently worked around.

## Issues Encountered

None beyond the deviation above.

## TDD Gate Compliance

Not applicable. This plan's `type` is `execute`, every task is planning bookkeeping performed through tool verbs (`gsd-tools windows fixed`, `gsd-tools state add-blocker`, `gsd-tools init todos`) or a value edit in an already-tool-rendered shape (SEED-001 frontmatter) — none of the three tasks carry `tdd="true"`, none contain a `<behavior>` block, and none modify non-test application source files. No RED-commit gate applies.

## Known Stubs

None.

## Self-Check: PASSED

- `[ -f .planning/todos/pending/2026-08-14-bench-pinnedat-validates-a-checkout-by-git-rev-parse-head-alone.md]` → FOUND
- `git log --oneline --all | grep -q 9f6603a2` → FOUND
- `git log --oneline --all | grep -q 3d45ed65` → FOUND
- `git log --oneline --all | grep -q ff500512` → FOUND
- All three tasks' `<acceptance_criteria>` re-verified per the Task Commits / Evidence Table sections above — all PASS (one documented deviation on an over-broad verify-gate scope, no functional gap)
- Plan-level `<verification>`: `windows status` → 13/16/29/31/33 fixed with `resolved_at`; 20/21/34 open; `open_count` 6 — PASS. Blockers: two verb-added bullets, zero content lines removed — PASS. Pending Todos body == `pending_todos_markdown` — PASS. SEED-001 carries SEED-002's consumption fields — PASS.

Ready for phase verification (`/gsd-verify-work 02`) — this was the final plan in Phase 02.
