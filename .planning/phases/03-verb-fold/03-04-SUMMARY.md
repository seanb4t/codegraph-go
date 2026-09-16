---
phase: 03-verb-fold
plan: 04
subsystem: cli
tags: [roadmap-backlog, census, mutation-log, conventional-commits, tdd]

# Dependency graph
requires:
  - phase: 03-verb-fold/03-03
    provides: "Family (b)'s RED demonstration and the generated-surface/MCP proofs; the clean, byte-identical-to-baseline tree Family (c)'s after-census re-runs against"
provides:
  - "ROADMAP.md ## Backlog row `### Phase 999.5: remove the query/unlock rename stubs` — the v0.15.0 stub-removal record (D-09), written exclusively through the gsd-tools phase verb"
  - "03-MUTATION-LOG.md's `## Family (c) — VERB-05` — the after-half of the before/after census pair, proving zero old-verb references survive outside the stub declaration"
  - "the phase commit-history audit (D-15/VERB-08): one `feat(cli)!:` commit with its BREAKING CHANGE footer, everything else docs:, no CI-skip marker"
  - "the phase-closing green gate: build, internal/cli tests, daemon lock tests, docs:cli:drift, wireoracle"
affects: []

# Actuals (#2632)
actuals:
  tokens: 2420
  tasks: 2
  commits: 2
  plan_head_before: 45ecf206

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "ROADMAP backlog rows are written exclusively through gsd-tools phase add --id; the only hand edit is a value fill inside the block the verb wrote (planning-artifacts.md, D-09)."
    - "A before/after census pair is only trustworthy when both runs use the byte-identical rg instrument and the byte-identical planted positive control — reused verbatim from Family (a) rather than re-derived."

key-files:
  created:
    - .planning/phases/999.5-remove-the-query-unlock-rename-stubs/.gitkeep
  modified:
    - .planning/ROADMAP.md
    - .planning/phases/03-verb-fold/03-MUTATION-LOG.md

key-decisions:
  - "The 999.5 Goal value cites the feat commit's full 40-char SHA (5d69ee2ea3c6276ed73c946b24be93612fae1698) rather than the short form used elsewhere in this phase's docs, satisfying the verify gate's [0-9a-f]{7,40} check unambiguously."
  - "Family (c)'s final-gate daemon leg scoped to the plan's own named tests (TestUnlock|TestAcquire|TestIsStale) rather than the full internal/daemon suite, per the plan's own <verify> command — the known TestDaemonSharedWriter load flake (recorded in Baseline) was not encountered because it was never invoked in this narrower run."

requirements-completed: [VERB-05, VERB-08]

coverage:
  - id: D1
    description: "999.5 backlog row for the v0.15.0 stub removal, written by gsd-tools phase add --id, Goal value filled, ROADMAP parser view unchanged"
    requirement: "VERB-08"
    verification:
      - kind: other
        ref: "plan Task 1 verify gate 1 (heading position, Goal content, no version token)"
        status: pass
      - kind: other
        ref: "plan Task 1 verify gate 2 (commit-shape: additions-only diff, milestone phase filter unchanged, roadmap validate clean)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Family (c) post-fold VERB-05 census: same instrument and control as Family (a), hits confined to the stub declaration (renamed.go + 2 allowlist lines), phase commit-history audit, and the final gate all green"
    requirement: "VERB-05"
    verification:
      - kind: other
        ref: "plan Task 2 verify gate 1 (planted control found on :1:/:3:, removed byte-clean)"
        status: pass
      - kind: other
        ref: "plan Task 2 verify gate 2 (after-census: allowlist=2, renamed.go>=1, no other path)"
        status: pass
      - kind: other
        ref: "plan Task 2 verify gate 3 (commit audit: one feat(cli)!:, footer present, rest test/docs/chore, no CI-skip)"
        status: pass
      - kind: other
        ref: "plan Task 2 verify gate 4 (final gate: build, internal/cli tests, daemon lock tests, docs:cli:drift, wireoracle, clean tree)"
        status: pass
      - kind: other
        ref: "plan Task 2 verify gate 5 (Family (c) recorded, three families present, control gone from disk and index)"
        status: pass
    human_judgment: false

duration: ~25min
completed: 2026-09-16
status: complete
---

# Phase 3 Plan 04: Verb Fold — ROADMAP backlog row and post-fold VERB-05 census Summary

**Closed the phase: `999.5` backlog row (v0.15.0 stub removal) written through the `gsd-tools phase add` verb, and Family (c)'s after-fold census proving the only surviving `codegraph query`/`codegraph unlock` references live inside the stubs' own declaration (`renamed.go` + 2 allowlist lines) — with the phase's full commit-history audit and green gate recorded alongside.**

## Performance

- **Duration:** ~25 min
- **Started:** 2026-09-16T18:39:21Z (STATE.md's last recorded timestamp, plan dispatched immediately after)
- **Completed:** 2026-09-16
- **Tasks:** 2
- **Files modified:** 3 (1 created, 2 modified)

## Accomplishments

- `### Phase 999.5: remove the query/unlock rename stubs` is now under `## Backlog`, written exclusively by `gsd_run phase add "remove the query/unlock rename stubs" --id 999.5` (predicted and confirmed via `phase next-decimal 999` → `999.5`) — the diff to `.planning/ROADMAP.md` is additions-only (one `+###` heading, zero removed lines), the roadmap parser's `getMilestonePhaseFilter` still returns the same active phase set (`01-defect-flake-burn-down`, `02-guards-ci-wiring-docs-burn-down`, `03-verb-fold`) with `999.5-…` excluded, and `roadmap validate` is clean.
- The block's `**Goal:**` value (the only hand edit, inside the tool-written shape) names `v0.15.0`, the exact files to delete (`internal/cli/renamed.go`, `renamed_test.go`), the `root.go` registrations to drop, the two allowlist lines to remove, `task docs:cli`, the re-run of this phase's census expecting zero hits, VERB-03/VERB-04, and the feat commit's full SHA `5d69ee2ea3c6276ed73c946b24be93612fae1698`.
- Family (c) re-ran the identical `rg -nU -w --hidden 'codegraph\s+(query|unlock)' …` instrument and the identical three-line planted control Family (a) used — found on `:1:` and `:3:` before the real run was trusted, removed byte-clean. The after-census reports `after: 4 lines across 2 files (renamed.go=2, allowlist=2)`, with every hit inside `internal/cli/renamed.go`'s doc comment and the two D-12 allowlist lines — nothing in `docs/`, README, `.claude/`, `.github/`, `internal/mcp/`, `internal/daemon/`, `internal/cli/index.go`, `Taskfile.yml`, or any test file.
- The commit-history audit over the `feat(cli)!:` commit's parent (`5d69ee2e^`) to HEAD found 7 commits: exactly one `feat(cli)!:` subject carrying the `BREAKING CHANGE:` footer, and six `docs(` commits — every subject regex-conformant against `pr-title.yml`, no `ci skip`/`skip ci` marker anywhere.
- The final gate is green under `GOTOOLCHAIN=go1.26.6`: `go build ./...`, 5 `internal/cli/...` packages (`internal/cli`, `archtest`, `present`, `present/archtest`, `tui`), the three named daemon lock tests, `task docs:cli:drift` (`compared 1 generated file`, byte-identical), and `task test:wireoracle` — tree clean at the end.

## Task Commits

1. **Task 1: Backlog row `999.5`** — `ef3cf0c4` `docs(03-04): add backlog row 999.5 for the v0.15.0 removal of the query/unlock stubs (D-09)`
2. **Task 2: VERB-05 census AFTER the fold + commit audit + final gate** — `8adea168` `docs(03-04): record the post-fold VERB-05 census and the phase commit audit (Family c)`

**Plan metadata:** this SUMMARY is committed separately per the sequential-execution instructions (STATE.md/ROADMAP.md updates are orchestrator-owned for this run).

## Files Created/Modified

- `.planning/ROADMAP.md` — `## Backlog` row `### Phase 999.5: remove the query/unlock rename stubs`, Goal value filled (D-09)
- `.planning/phases/999.5-remove-the-query-unlock-rename-stubs/.gitkeep` (new, created by the `phase add` verb)
- `.planning/phases/03-verb-fold/03-MUTATION-LOG.md` — `## Family (c) — VERB-05: positive-controlled census AFTER the fold` appended

## Decisions Made

- Used the feat commit's full 40-character SHA in the Goal value rather than the short form, so the verify gate's hex-pattern check is satisfied without ambiguity and future readers of the backlog row don't need to re-resolve an abbreviation.
- Family (c)'s final gate ran exactly the plan's own scoped daemon-test invocation (`-run 'TestUnlock|TestAcquire|TestIsStale'`), not the full `internal/daemon` suite — the known `TestDaemonSharedWriter` load flake recorded in the phase Baseline was correctly out of scope for this narrower run and did not need to be reported.

## Deviations from Plan

None - plan executed exactly as written. Both tasks' automated verify gates passed on the first attempt; no auto-fixes, no architectural questions, no blockers.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The phase is closed: VERB-05's before/after pair is complete (Family (a) 14 lines/6 files → Family (c) 4 lines/2 files, all inside the stub declaration), VERB-08's commit-shape and ROADMAP-record halves are both satisfied, and the full gate (build, cli/daemon tests, docs:cli:drift, wireoracle) is green at HEAD.
- `999.5` sits in `## Backlog` as a fully-scoped removal ready for `/gsd-review-backlog` to promote when v0.15.0 planning begins; it requires no further action from this milestone.
- No blockers. Tree clean (`git status --porcelain -- internal docs cmd testdata` empty at the final gate).

---
*Phase: 03-verb-fold*
*Completed: 2026-09-16*

## Self-Check: PASSED
