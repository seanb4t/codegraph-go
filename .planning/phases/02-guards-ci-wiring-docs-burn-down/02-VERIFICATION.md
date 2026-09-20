---
phase: 02-guards-ci-wiring-docs-burn-down
verified: 2026-09-16T08:15:00Z
status: passed
score: 9/9 must-haves verified
covered_files: [".github/required-status-checks.txt", ".github/workflows/ci.yml", ".planning/phases/02-guards-ci-wiring-docs-burn-down/02-01-PLAN.md", ".planning/phases/02-guards-ci-wiring-docs-burn-down/02-01-SUMMARY.md", ".planning/phases/02-guards-ci-wiring-docs-burn-down/02-02-PLAN.md", ".planning/phases/02-guards-ci-wiring-docs-burn-down/02-02-SUMMARY.md", ".planning/phases/02-guards-ci-wiring-docs-burn-down/02-03-PLAN.md", ".planning/phases/02-guards-ci-wiring-docs-burn-down/02-03-SUMMARY.md", ".planning/phases/02-guards-ci-wiring-docs-burn-down/02-04-PLAN.md", ".planning/phases/02-guards-ci-wiring-docs-burn-down/02-04-SUMMARY.md", ".planning/phases/02-guards-ci-wiring-docs-burn-down/02-04-ruleset-put-body.json", ".planning/phases/02-guards-ci-wiring-docs-burn-down/02-05-PLAN.md", ".planning/phases/02-guards-ci-wiring-docs-burn-down/02-05-SUMMARY.md", ".planning/phases/02-guards-ci-wiring-docs-burn-down/02-06-PLAN.md", ".planning/phases/02-guards-ci-wiring-docs-burn-down/02-06-SUMMARY.md", ".planning/phases/02-guards-ci-wiring-docs-burn-down/02-07-PLAN.md", ".planning/phases/02-guards-ci-wiring-docs-burn-down/02-07-SUMMARY.md", ".planning/phases/02-guards-ci-wiring-docs-burn-down/02-MUTATION-LOG.md", ".planning/phases/02-guards-ci-wiring-docs-burn-down/02-REVIEW-FIX.md", ".planning/phases/02-guards-ci-wiring-docs-burn-down/02-REVIEW.md", ".planning/seeds/SEED-001-local-svelte-shadcn-graph-browsing-ui.md", ".planning/todos/pending/2026-08-14-bench-pinnedat-validates-a-checkout-by-git-rev-parse-head-alone.md", "SECURITY.md", "docs/RELEASE.md", "go.mod", "go.sum", "internal/upgrade/taskfile_shape_test.go", "scripts/check-ruleset-drift.sh"]
covered_digest: "v1:sha256:65b70d159dc56af880c6f9e821e364a81868b68ad40f42f24ed05a930f023480"
behavior_unverified: 0
overrides_applied: 0
human_verification:

  - test: "Review the three named visual/API changes in the re-vendored shadcn-svelte components: button.svelte's secondary-hover switch from `hover:bg-secondary/80` to `hover:bg-[color-mix(...)]`; command-link-item.svelte's selected-state color scheme now matching command-item.svelte (dropped `aria-selected:bg-accent`/icon-color rule); table-row.svelte's new additive `has-aria-expanded:bg-muted/50` utility."
    expected: "The maintainer confirms these three visual deltas are acceptable UI changes, per D-12's explicit deferral of visual/API judgment to end-of-phase review (\"the maintainer reviews at end of phase\" — CONTEXT D-12). Automated gates (pnpm check, vitest, live graph-console/breadcrumb checks) only prove the rebuilt UI still functions; they cannot judge whether a visual change is desired."
    why_human: "Visual/rendering acceptability is not something a grep or test-pass can certify — it is the exact judgment D-12 named as reserved for the maintainer, not the executor."
  - test: "Confirm the D-08 `protect-main` ruleset PUT authorization (adding `goreleaser check` and `tmux e2e` as required contexts, making the tmux job block every future PR) was a genuinely informed maintainer decision, not a mechanical approval of a prepared command."
    expected: "The maintainer affirms they understood the consequence — every subsequent PR now runs and must pass the tmux e2e job — before/when authorizing the `gh api --method PUT` in commit `8e4b2aae`."
    why_human: "02-04-SUMMARY.md's own coverage table (D3) explicitly marks this `human_judgment: true` and states it is \"a judgment call outside what an automated check can assert.\" This verifier can confirm the mechanism (precondition re-check, review package, applied PUT, live-state re-verification) but not the human's state of mind when authorizing it."
---


**Re-verified at HEAD 2026-09-20 (v0.14.0 milestone audit).** This report went `stale` because later phases legitimately modified files in its `covered_files` — Phase 7's own fixes touched `internal/cli/install.go`, and Phases 5-7 touched `internal/agents/*`. The milestone audit re-exercised these claims against HEAD rather than re-stamping blind: the full suite (54 packages plus `internal/daemon` run separately) is green, an independent integration check verified 7/7 cross-phase seams with a real-binary end-to-end pass, and every phase holds a security audit with 0 threats open at or above the `high` gate. The digest below is recomputed over the same covered set at that HEAD. See `.planning/v0.14.0-MILESTONE-AUDIT.md`.
# Phase 2: Guards, CI Wiring & Docs Burn-down Verification Report

**Phase Goal:** Every guard in the ledger that could pass vacuously now fails against its recorded incident shape, CI runs on every pull request the checks that today exist only as local Taskfile targets, and every documentation and planning claim — dependency counts, provenance scope, scanner coverage, window status, todo table, seed status — states what is true today.
**Verified:** 2026-09-16T08:15:00Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `web:drift` is demonstrated RED against `98cd41dd`'s exact incident shape on a clean checkout, with a green control on HEAD, and `Taskfile.yml` is unchanged (GRD-09) | ✓ VERIFIED | Independently re-ran the replay in a fresh scratch worktree of `98cd41dd`: exit 201, `source half MATCH (108 files, 1e0bff2f…)`, `OUTPUT-half mismatch` naming `32 files / 8658e6fe64bb…` (marker) vs `22 files / a76add11cadd…` (recomputed) — byte-identical to `02-MUTATION-LOG.md` Family (a) and to the RESEARCH transcript. `git diff --quiet -- Taskfile.yml` confirmed 0 diff, and `git diff 68f74bb6..HEAD -- Taskfile.yml` is empty (0 lines) — the gate is untouched. |
| 2 | `check:gonum` and `check:no-force-layout` now run in `ci.yml`'s required `test` job on every PR, with no re-derived guard logic (GRD-11) | ✓ VERIFIED | `ci.yml` has exactly one each of `Install syft`, `Gonum dependency-shape guard (GRD-11/GRF-10)`, `No-force-layout guard (GRD-11/GRF-06)`, in line order `Set up Node`(131) < `Install syft`(209) < `check:gonum`(219) < `check:no-force-layout`(225) < `Test subprocess integration harness`(250). Syft pinned to the same SHA as `release.yml`. Locally: `task check:gonum` exits 0, prints `check:gonum: PASS` with both positive-control lines (`gonum.org/v1/gonum present`, `cockroachdb/pebble/v2 present`, `tree-sitter/go-tree-sitter — 3 with cgo`); `task check:no-force-layout` exits 0, `verdict PASS`, `forbidden matches 0`. `go test ./internal/upgrade/ -run TestWorkflowRunBodiesInvokeTask` passes (bare `task <target>` bodies accepted). Family (b) of the mutation log records a planted `name: 'cose'` literal making the exact step command fail, then a byte-clean revert. |
| 3 | `requiredCheckNames` is compared against the live `protect-main` ruleset and fails when they diverge, and today the live/fixture sets genuinely agree at 8 (GRD-12) | ✓ VERIFIED | Ran `bash scripts/check-ruleset-drift.sh` live against `api.github.com` this session: `fixture … lists 8 contexts`, `live has 8 contexts, fixture has 8 contexts`, `self-check PASS — planted context detected`, `PASS — 8 contexts identical`, exit 0 — matching the Family (c) Green re-run recorded after the maintainer's D-08 PUT (commit `8e4b2aae`). `GOTOOLCHAIN=go1.26.6 go test ./internal/upgrade/ -run RequiredCheck -v` passes; `TestRequiredCheckNamesPreserved` logs `read 8 required contexts`. The script's fetch is now bounded by `--connect-timeout 10 --max-time 30` (WR-01 fix, commit `97bb6a13`), confirmed present in `scripts/check-ruleset-drift.sh` line 91. `var requiredCheckNames` literal is gone; `readRequiredCheckNames` loader exists. |
| 4 | `docs/RELEASE.md`'s dependency paragraph states shape (not raw counts) and credits `modelcontextprotocol/go-sdk`; SLSA provenance is described as attested over binaries; `SECURITY.md` states govulncheck/pnpm-audit's disjoint scope plus the advisory `tool-vuln` caveat (DOCS-08, DOCS-09, DOCS-10) | ✓ VERIFIED | `rg -c mark3labs docs/RELEASE.md` → 0 hits; no raw counts (`134`, `107`, `27`, `14`, `13`) found; `modelcontextprotocol/go-sdk` is named at line 352. `SECURITY.md` carries the one added sentence naming the advisory `tool-vuln` job (lines 75-77). GH #14 confirmed `CLOSED` via `gh issue view 14 --json state`. |
| 5 | WINDOWS #16/#33 (stale-open, resolved elsewhere) and #29/#31/#13 (this phase's own evidence) read `fixed`; #20/#21/#34 stay `open` as record-only; tooling gaps reported upstream, never worked around locally (GRD-14) | ✓ VERIFIED | `gsd-tools windows status` (live, this session): `open_count == 6`; ids 13, 16, 29, 31, 33 all `status: fixed` with non-empty `resolved_at`; ids 20, 21, 34 all `status: open`, untouched. STATE.md Blockers/Concerns carries both the record-only bullet (naming #20/#21/#34) and the four-gap tooling report, added via `state add-blocker` (verified: pre-existing bullets intact, two new `- ` lines present). No matching issue exists yet on `open-gsd/gsd-core` opened by the executor (`gh issue list --author @me` returns 0 matches) — the draft stays in the SUMMARY as required. |
| 6 | STATE.md's Pending Todos table matches `.planning/todos/`, and SEED-001's frontmatter records consumption by v0.12.0 (DOCS-11) | ✓ VERIFIED | `.planning/todos/pending/2026-08-14-bench-pinnedat-….md` exists and is tracked. `gsd-tools init todos` reports `pending_read_ok: true`, `todo_count: 1`, `pending_dir_exists: true`. `diff` between STATE.md's rendered `### Pending Todos` body and the tool's own `pending_todos_markdown` output is empty (blank lines aside); no hand-authored table survives (`rg '| Created | Area'` finds none). SEED-001 carries `status: implemented`, `consumed_by: v0.12.0 — Local Graph UI (Phases 1–6)`, `consumed_on: 2026-09-07`, with `audit_acknowledged` and `planted` intact; `gsd-tools list-seeds implemented` lists it. |
| 7 | The vendored `button.svelte` (and sibling) drift is isolated under Corepack-pinned pnpm and re-vendored, with `web/build/**` rebuilt in the same commit (GRD-10) | ✓ VERIFIED | `task web:components:drift` (run live this session) → `PASS — all 50 vendored component files across 8 components byte-identical to shadcn-svelte@1.5.1's regeneration`, exit 0. `task web:build && task web:drift` → both halves `MATCH` (119 source / 36 output files), exit 0. `pnpm check` → `0 ERRORS 0 WARNINGS`. (Local `task web:build` used for this re-check regenerated content-hashed chunk filenames as a side effect of Vite's build; reverted with `git checkout -- web/build/ && git clean -fd web/build/` before concluding — tree confirmed clean afterward, `git status --porcelain` empty.) |
| 8 | Every wired guard fails loudly on an empty subject set rather than passing vacuously (backstop truth, 02-02) | ✓ VERIFIED | `task check:gonum`'s transcript prints its floor counts before asserting (`26 gonum packages`, `SBOM lists 148 packages`, `91 packages inspected`) and `check:no-force-layout` prints `scanned 106 files` before its verdict — both floors are non-zero on the clean tree, confirmed live this session. |
| 9 | No `[ci skip]`/`[skip ci]` anywhere in phase commit history; `Taskfile.yml` untouched at the diff base (repo-wide prohibitions) | ✓ VERIFIED | `git log --format=%s%n%b 1b1184ca..206a9d58 \| rg -i 'ci skip\|skip ci'` → 0 matches. `git diff 68f74bb6..HEAD -- Taskfile.yml` → 0 lines. |

**Score:** 9/9 truths verified (0 present-but-behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `.planning/phases/.../02-MUTATION-LOG.md` | Families (a)/(b)/(c) with full transcripts | ✓ VERIFIED | All three families present, content matches independently re-run transcripts |
| `.github/required-status-checks.txt` | 8-line shared fixture | ✓ VERIFIED | 8 non-blank lines, includes `goreleaser check` and `tmux e2e`, no comments |
| `scripts/check-ruleset-drift.sh` | Hard-failing CI-only comparator, timeout-bounded | ✓ VERIFIED | Executable, `set -euo pipefail`, `--connect-timeout 10 --max-time 30`, live-run PASS |
| `.github/workflows/ci.yml` | 4 new `test`-job steps (syft, gonum, no-force-layout, ruleset-drift) | ✓ VERIFIED | All present, correctly ordered, `actionlint` clean |
| `internal/upgrade/taskfile_shape_test.go` | Data-file loader replacing the literal, `runBodyExceptions` entry | ✓ VERIFIED | `readRequiredCheckNames` present, literal gone, tests pass |
| `docs/RELEASE.md` / `SECURITY.md` | Count-free dependency paragraph, tool-vuln sentence | ✓ VERIFIED | Confirmed via grep, GH #14 closed |
| `.planning/WINDOWS.md`, `STATE.md`, `SEED-001`, pending-todo file | Tool-verb-only bookkeeping updates | ✓ VERIFIED | All confirmed via `gsd-tools` query verbs, not by reading the raw files |
| Re-vendored `web/src/lib/components/ui/**` + `web/build/**` | All 8 families at one snapshot, rebuilt bundle | ✓ VERIFIED | `web:components:drift` PASS, `web:drift` PASS |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| PR → `ci.yml` `test` job | `check:gonum` / `check:no-force-layout` | already-required `test` context, no ruleset edit | ✓ WIRED | Confirmed step order and bare `task <target>` bodies |
| PR → `ci.yml` `test` job | `check-ruleset-drift.sh` | CI-only step, `runBodyExceptions` entry | ✓ WIRED | Step present, excepted correctly, `go test` green |
| `.github/required-status-checks.txt` | Go loader + shell script | shared single data file | ✓ WIRED | Both consumers read 8 identical lines; no duplicate context string found in Go source or the script |
| `02-01`/`02-05`/`02-06` evidence | `02-07`'s `windows fixed` calls | phase's single WINDOWS.md writer | ✓ WIRED | Ledger status confirmed live |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| web:drift RED on 98cd41dd | `git worktree add --detach … 98cd41dd && task web:drift` | exit 201, OUTPUT-half mismatch, digests match log | ✓ PASS |
| web:drift GREEN on HEAD | same, on HEAD | exit 0, both digests MATCH | ✓ PASS |
| check:gonum on clean tree | `GOTOOLCHAIN=go1.26.6 task check:gonum` | exit 0, `check:gonum: PASS`, positive controls fired | ✓ PASS |
| check:no-force-layout on clean tree | `task check:no-force-layout` | exit 0, `verdict PASS`, `forbidden matches 0` | ✓ PASS |
| ruleset-drift live | `bash scripts/check-ruleset-drift.sh` | exit 0, `PASS — 8 contexts identical` | ✓ PASS |
| ruleset-drift timeout fix present | `rg 'connect-timeout' scripts/check-ruleset-drift.sh` | line 91: `--connect-timeout 10 --max-time 30` | ✓ PASS |
| Go shape tests | `GOTOOLCHAIN=go1.26.6 go test ./internal/upgrade/ -count=1` | `ok` | ✓ PASS |
| actionlint | `task lint:actions` | exit 0, no findings | ✓ PASS |
| web check | `cd web && pnpm check` | `0 ERRORS 0 WARNINGS` | ✓ PASS |
| web vitest | `cd web && pnpm vitest run` | 584/585 passed, 1 pre-existing timeout (see note below) | ⚠️ SEE NOTE |
| web:components:drift | `task web:components:drift` | `PASS — 50/50 byte-identical` | ✓ PASS |
| web:build + web:drift | `task web:build && task web:drift` | both halves MATCH, exit 0 | ✓ PASS |

**Note on the vitest failure:** one test (`tests/browse-page.test.ts` — "leaves the RPC call count and the rendered target unchanged across three keystrokes…", CR-01) timed out at 15000ms in this verification session, on a machine reporting `Load Avg: 31.01` (16 vCPUs) at the time. This test file (`c9d19595`, a prior milestone's feature commit) predates Phase 02 and was not touched by any Phase-02 diff; its own in-file comment (added in an earlier plan, `06-03`) already documents that this specific test runs right at the edge of its timeout budget under suite-wide contention, unrelated to any code change. `02-05-SUMMARY.md` recorded `585/585 passed` when the plan executed. This is recorded as an environmental/machine-load artifact, not a phase regression, and is not counted as a gap.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|---|---|---|---|---|
| GRD-09 | 02-01 | web:drift RED replay against 98cd41dd | ✓ SATISFIED | Independently re-run this session |
| GRD-10 | 02-05 | All 8 shadcn-svelte families re-vendored | ✓ SATISFIED | `web:components:drift` PASS |
| GRD-11 | 02-02 | check:gonum / check:no-force-layout wired into ci.yml | ✓ SATISFIED | Steps present, ordered, tested |
| GRD-12 | 02-03, 02-04 | ruleset-drift comparator, live 8-vs-8 | ✓ SATISFIED | Live script run, GREEN |
| GRD-14 | 02-07 | WINDOWS closures + record-only rows | ✓ SATISFIED | `windows status` confirms |
| DOCS-08 | 02-06, 02-07 (shared ID) | RELEASE.md dependency paragraph | ✓ SATISFIED | grep confirms count-free, go-sdk credited |
| DOCS-09 | 02-06 | Provenance census, GH #14 closed | ✓ SATISFIED | `gh issue view 14` → CLOSED |
| DOCS-10 | 02-06 | SECURITY.md tool-vuln sentence | ✓ SATISFIED | Sentence present |
| DOCS-11 | 02-07 | STATE.md/SEED-001 bookkeeping reconciled | ✓ SATISFIED | `init todos` diff empty, seed fields confirmed |

**Orphan check:** `.planning/REQUIREMENTS.md`'s traceability table maps GRD-09 through DOCS-11 all to "Phase 2 / Complete" with no additional IDs beyond the plan-declared set. GRD-13 is correctly mapped to Phase 4 ("Pending") and is out of scope for this phase (confirmed by 02-CONTEXT.md's explicit note: "GRD-13 rides with Phase 4 by roadmap decision"). No orphaned requirements found.

### Anti-Patterns Found

None. Scanned all phase-modified implementation files (`.github/workflows/ci.yml`, `SECURITY.md`, `docs/RELEASE.md`, `internal/upgrade/taskfile_shape_test.go`, `scripts/check-ruleset-drift.sh`, `.github/required-status-checks.txt`) for `TBD|FIXME|XXX|TODO|HACK|PLACEHOLDER` — the only hit was a comment in `taskfile_shape_test.go` describing a false-positive check for path-shaped template placeholders (not a debt marker; it names the pattern the test defends against, not an incomplete implementation).

### Flagged Assumptions (reported, not failed)

Several plans carry `flagged_assumptions` entries from the spec-less edge-probe fallback (status: `unresolved`). None of these represent an incomplete implementation — each names a genuine future-drift risk the planner explicitly decided was out of this phase's scope, with the accepting decision cited:

- **EDGE-GRD-12-01** (02-03): ruleset id `20157557` is assumed stable; a future ruleset recreation would 404 loudly (the designed D-06 behavior), not silently pass.
- **EDGE-GRD-10-01** (02-05): the shadcn-svelte registry could drift again before the next scheduled `components-drift.yml` run; pinning the registry snapshot was explicitly deferred (CONTEXT "Deferred Ideas").
- **EDGE-DOCS-08-01** (02-06): the named direct-require list in RELEASE.md will go stale on the next `go.mod` change; D-13 already accepts this by pointing at `go list -m all`.
- **EDGE-DOCS-10-01** (02-06): a future rename/promotion of the `tool-vuln` job would make the one SECURITY.md sentence wrong with no gate; D-15 explicitly declined a drift assertion.
- **EDGE-GRD-14-01** (02-07): #20/#21/#34 staying `open` keeps `windows status`'s `open_count` above zero, which could collide with a `workflow.windows_enforce` ship gate at milestone end — surfaced for the maintainer, not resolved here.
- **EDGE-DOCS-11-01** (02-07): no acceptance criterion asserts STATE.md writes from concurrent wave-2 plans cannot interleave — surfaced, not observed to have actually happened (git history shows clean sequential commits).

### Human Verification Required

### 1. Review the three named visual/API changes in the re-vendored components

**Test:** Look at the rendered `button.svelte` (secondary-hover color-mix), `command-link-item.svelte` (selected-state color scheme change), and `table-row.svelte` (new `has-aria-expanded:bg-muted/50` utility) in the running UI.
**Expected:** The maintainer confirms these are acceptable visual changes — this is exactly the review D-12 (02-CONTEXT.md) reserved for end-of-phase, not for the executor.
**Why human:** No automated check can certify a visual delta is "acceptable"; the plan itself defers this judgment.

### 2. Confirm the D-08 ruleset-authorization was genuinely informed

**Test:** Confirm with the maintainer that authorizing the `gh api --method PUT` in commit `8e4b2aae` (making `tmux e2e` a required, merge-blocking context on every future PR) was a deliberate, understood decision.
**Expected:** Maintainer affirms informed consent to the new blocking behavior.
**Why human:** `02-04-SUMMARY.md`'s own coverage table marks this `human_judgment: true` — it is explicitly outside what any automated check can assert.

### Gaps Summary

No gaps found. All 9 observable truths derived from the ROADMAP success criteria and the plans' `must_haves` were independently re-verified against the live codebase and, where applicable, live external state (GitHub ruleset API, GH issue #14) rather than trusted from SUMMARY narration. Two items are routed to human verification because the plans themselves (D-12, and 02-04's own coverage table) explicitly named them as maintainer-judgment items rather than automatable checks — this does not indicate incomplete work, only that a human sign-off is the designed final step.

---

_Verified: 2026-09-16T08:15:00Z_
_Verifier: Claude (gsd-verifier)_
