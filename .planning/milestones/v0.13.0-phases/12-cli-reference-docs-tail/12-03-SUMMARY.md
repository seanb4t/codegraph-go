---
phase: 12-cli-reference-docs-tail
plan: 03
subsystem: cli
tags: [cobra, mutation-testing, threat-register, validation, docs-05, docs-06]

# Dependency graph
requires:
  - phase: 12-cli-reference-docs-tail (plan 01)
    provides: "docs/CLI-REFERENCE.md, tools/clidoc, task docs:cli/docs:cli:drift, TestEveryRegisteredFlagIsAccountedFor, the allowlist"
  - phase: 12-cli-reference-docs-tail (plan 02)
    provides: "docs/RELEASE.md brew-trust wording, README.md CLI-reference link"
provides:
  - "12-MUTATION-LOG.md — three RED demonstrations (throwaway hidden flag, deleted reference line, bogus allowlist entry), each applied for real and reverted byte-clean"
  - "12-SECURITY.md — 13 numbered threats + T-12-SC, threats_open: 0"
  - "12-VALIDATION.md — Per-Task Verification Map filled, Wave 0 and Sign-Off ticked, wave_0_complete/nyquist_compliant true"
  - "DOCS-05, DOCS-06, DOCS-07 marked complete in REQUIREMENTS.md (last plan declaring all three)"
affects: []

# Actuals (#2632)
actuals:
  tokens: 11633
  tasks: 2
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "A gate is trusted only after being watched RED against a confirmed-applied, byte-clean-reverted mutation of the project's own artefacts — never a hypothetical 'would fail'"
    - "Pairing two independent instruments (guard + drift gate) against the same mutation to demonstrate which one closes which gap, rather than asserting either alone is sufficient"

key-files:
  created:
    - .planning/phases/12-cli-reference-docs-tail/12-MUTATION-LOG.md
    - .planning/phases/12-cli-reference-docs-tail/12-SECURITY.md
  modified:
    - .planning/phases/12-cli-reference-docs-tail/12-VALIDATION.md
    - .planning/REQUIREMENTS.md

key-decisions:
  - "Family (b) deletes the --no-open Options line, not --editor-url, because --editor-url also appears in ui's own Synopsis prose — deleting it would leave the guard's whole-document flag match green even with the Options line gone, masking the drift gate's exclusivity as the actual content-completeness check. This is recorded in 12-SECURITY.md's Notes as an explicit unclassified residual risk, not silently worked around."
  - "threats_open: 0 is tied to a literal grep-computed count of open high-severity rows, not asserted by hand — three high rows (T-12-01, T-12-02, T-12-08) are closed by name, with T-12-08 explicitly marked closed (verdict) rather than closed, since D-12 rules out a committed gate for its wording."

patterns-established:
  - "MUTATION-LOG house format (11-MUTATION-LOG.md precedent): pre-mutation cleanliness gate, diff-shown mutation, confirmed-applied check, verbatim RED, git checkout -- revert, post-revert gate, verbatim GREEN, per family — now demonstrated for a second phase in a row"

requirements-completed: [DOCS-05, DOCS-06, DOCS-07]

coverage:
  - id: D1
    description: "12-MUTATION-LOG.md proves both new gates (the DOCS-06 accounting guard, the DOCS-05 drift gate) discriminate real mutations of this project's own artefacts, never cobra/doc itself: family (a) a throwaway hidden flag on ui, family (b) a deleted reference line, family (c) a bogus allowlist entry — each applied for real, RED captured verbatim, reverted byte-clean"
    requirement: DOCS-06
    verification:
      - kind: integration
        ref: "12-MUTATION-LOG.md families (a)/(b)/(c) — TestEveryRegisteredFlagIsAccountedFor and task -s docs:cli:drift, all RED transcripts pasted verbatim, all reverts confirmed git diff --quiet"
        status: pass
    human_judgment: false
  - id: D2
    description: "12-SECURITY.md's threat register carries every T-12 threat (13 numbered rows + T-12-SC) with a named test, verify command, or explicit verdict, and threats_open: 0 is honest — computed as the literal count of open high-severity rows (zero), with the three high rows closed by name"
    requirement: DOCS-05
    verification:
      - kind: other
        ref: "12-SECURITY.md Threat Register + Security Audit Trail; open_high == threats_open verified by grep at commit time (0 == 0)"
        status: pass
    human_judgment: false
  - id: D3
    description: "12-VALIDATION.md's Per-Task Verification Map has zero TBD cells, real plan/task/command values, Wave 0 and Sign-Off checklists ticked, and wave_0_complete/nyquist_compliant set true — with the file's heading structure byte-unchanged from what plan-phase seeded (values only, no invented headings)"
    requirement: DOCS-07
    verification:
      - kind: other
        ref: "12-VALIDATION.md heading count (6) matches git show ca015c4b:...12-VALIDATION.md heading count (6); TBD count 0; wave_0_complete/nyquist_compliant true; status: draft left untouched for validate-phase"
        status: pass
    human_judgment: false
  - id: D4
    description: "Phase-close gate green at the final commit: go vet, go build, task test:unit (reaching internal/cli), task test:golden, task lint:go, task proto:drift, task web:drift (both halves MATCH), task docs:cli:drift, and the accounting guard — with git status --porcelain empty"
    requirement: DOCS-05
    verification:
      - kind: integration
        ref: "phase-close gate transcripts pasted below — nine commands, all green"
        status: pass
    human_judgment: false

# Metrics
duration: 12min
completed: 2026-09-13
status: complete
---

# Phase 12 Plan 03: CLI Reference Docs Tail — Mutation Log, Security Register, Validation Close-Out Summary

**Three RED demonstrations (a throwaway hidden flag on `ui`, a deleted reference line, a bogus allowlist entry) proved the DOCS-05 drift gate and the DOCS-06 accounting guard discriminate real tampering of this project's own artefacts — never `cobra/doc` itself — each mutation applied for real, captured RED verbatim, and reverted byte-clean; the phase's threat register closes with `threats_open: 0` honestly tied to three named closed-high rows, the validation map records what actually landed, and the full nine-command phase-close gate is green with the tree clean.**

## Performance

- **Duration:** 12 min
- **Started:** 2026-09-13T17:01:50-04:00 (approx, immediately following 12-02)
- **Completed:** 2026-09-13T17:12:06-04:00
- **Tasks:** 2
- **Files modified:** 4 (2 created, 2 modified: `12-MUTATION-LOG.md` and `12-SECURITY.md` created; `12-VALIDATION.md` and `REQUIREMENTS.md` modified)

## Accomplishments

- `12-MUTATION-LOG.md`: three families, each hand-applied to a tracked file, confirmed applied, watched RED with the verbatim transcript, then reverted via `git checkout --` and re-verified GREEN — family (a) additionally proved the drift gate stays GREEN under the same hidden-flag mutation, the exact generator-blind-spot/guard pairing DOCS-06 exists to demonstrate
- `12-SECURITY.md`: 13 numbered threats (T-12-01…T-12-13) plus the shared `T-12-SC` row (14 total), Trust Boundaries, Accepted Risks, and Notes naming all five D-13 rows, two `unclassified` residual-risk assumptions, the resolved research A1/A2 assumptions, the `--version` planner addition, and "no new dependency"; `threats_open: 0` computed as a literal grep-matched count of `open`-status `high`-severity rows (zero — three high rows closed by name: T-12-01, T-12-02, T-12-08)
- `12-VALIDATION.md`: every `TBD` in the Per-Task Verification Map replaced with the real plan/task/command that landed; Wave 0 Requirements (8 items) and Validation Sign-Off (6 items) fully ticked; `wave_0_complete: true` and `nyquist_compliant: true` set; `status: draft` and the file's heading structure left untouched (values only, per the planning-artifacts invariant)
- Phase-close gate run end to end: `go vet`, `go build`, `task test:unit`, `task test:golden`, `task lint:go`, `task proto:drift`, `task web:drift`, `task docs:cli:drift`, and the accounting guard — all green, `git status --porcelain` empty at the final commit
- `DOCS-05`, `DOCS-06`, `DOCS-07` marked complete in `REQUIREMENTS.md` — this is the last of the three plans declaring all three requirement IDs (shared-ID gate #2388)

## Task Commits

Each task was committed atomically:

1. **Task 1: `12-MUTATION-LOG.md` — three families RED and reverted (D-10)** — `d48d1f7b` (docs)
2. **Task 2: `12-SECURITY.md` register + `12-VALIDATION.md` map + phase-close gate (D-13, D-14)** — `21f29846` (docs)

**Plan metadata:** this SUMMARY, committed together with the `REQUIREMENTS.md` completion write (per the objective's scope — STATE.md and ROADMAP.md are explicitly out of scope for this plan; the orchestrator owns those)

## Files Created/Modified

- `.planning/phases/12-cli-reference-docs-tail/12-MUTATION-LOG.md` — three families (a)/(b)/(c), each with a pre-mutation cleanliness gate, a diff of the applied mutation, a confirmed-applied check, the verbatim RED transcript, the revert, a post-revert gate, and the verbatim GREEN transcript, closing with a Closing section and a byte-clean proof
- `.planning/phases/12-cli-reference-docs-tail/12-SECURITY.md` — Trust Boundaries, 14-row Threat Register, Accepted Risks, Notes, Security Audit Trail, Sign-Off
- `.planning/phases/12-cli-reference-docs-tail/12-VALIDATION.md` — Per-Task Verification Map filled, Wave 0 and Sign-Off ticked, frontmatter flags set
- `.planning/REQUIREMENTS.md` — DOCS-05/06/07 checkboxes and traceability rows marked complete

## Family RED Transcripts (from `12-MUTATION-LOG.md`)

### Family (a) — throwaway hidden flag on `ui` (guard RED, drift gate stays GREEN)

```
cli_reference_test.go:236: walked 36 commands (hidden included), inspected 116 flags: 114 accepted via ../../docs/CLI-REFERENCE.md, 1 accepted via testdata/cli-reference-allowlist.txt
    cli_reference_test.go:256: 1 problem(s):
        unaccounted flag: codegraph ui --zz-throwaway (hidden flag — add an allowlist entry with a reason)
--- FAIL: TestEveryRegisteredFlagIsAccountedFor (0.02s)
FAIL
```

```
docs:cli:drift: compared 1 generated file
docs:cli:drift: docs/CLI-REFERENCE.md byte-identical to a fresh regeneration (temporary file only — source tree untouched)
```

### Family (b) — `--no-open` line deleted from the committed reference (both instruments RED)

```
docs:cli:drift: compared 1 generated file
::error::docs:cli:drift: docs/CLI-REFERENCE.md differs from a fresh regeneration by the pinned toolchain — run `task docs:cli` and commit the result
+      --no-open             do not open a browser automatically
task: Failed to run task "docs:cli:drift": exit status 1
```

```
cli_reference_test.go:236: walked 36 commands (hidden included), inspected 115 flags: 113 accepted via ../../docs/CLI-REFERENCE.md, 1 accepted via testdata/cli-reference-allowlist.txt
    cli_reference_test.go:256: 1 problem(s):
        unaccounted flag: codegraph ui --no-open (visible flag on a documented command — missing from docs/CLI-REFERENCE.md; run task docs:cli)
--- FAIL: TestEveryRegisteredFlagIsAccountedFor (0.02s)
FAIL
```

### Family (c) — bogus allowlist entry (guard RED on rot)

```
cli_reference_test.go:236: walked 36 commands (hidden included), inspected 115 flags: 114 accepted via ../../docs/CLI-REFERENCE.md, 1 accepted via testdata/cli-reference-allowlist.txt
    cli_reference_test.go:256: 1 problem(s):
        stale allowlist entry: codegraph ui --zz-bogus (matches no registered command or flag)
--- FAIL: TestEveryRegisteredFlagIsAccountedFor (0.02s)
FAIL
```

All three files confirmed `git diff --quiet` exit 0 after each revert; `git status --porcelain` showed only the new log file before its commit.

## Phase-Close Gate Transcripts

```
$ GOTOOLCHAIN=go1.26.6 go vet ./...
(no output — success)

$ GOTOOLCHAIN=go1.26.6 go build ./...
(no output — success)

$ GOTOOLCHAIN=go1.26.6 task test:unit
... (72 packages) ...
ok  	github.com/seanb4t/codegraph-go/internal/cli	(cached)
... (all ok, no FAIL)

$ GOTOOLCHAIN=go1.26.6 task test:golden
ok  	github.com/seanb4t/codegraph-go/testdata/golden	26.446s

$ GOTOOLCHAIN=go1.26.6 task lint:go
0 issues.

$ GOTOOLCHAIN=go1.26.6 task proto:drift
proto:drift: compared 4 generated files
proto:drift: all 4 generated files byte-identical to the pinned toolchain's regeneration (temporary tree only — source tree untouched)

$ task -s web:drift
web:drift: hashed 116 source files
web:drift: manifested 32 output files
web:drift: source half MATCH (116 files, 9acc57d7a02dfbb51b2f42663dbab8e239683319e1451d31ce8c272229276ff5)
web:drift: output half MATCH (32 files, 5b638277f6c80aa7fca24ddcce1bcf440aad74226f077ed0d7844c960506dfc0)
web:drift: PASS — hashed 116 source files, manifested 32 output files, committed web/build/ matches both digests

$ GOTOOLCHAIN=go1.26.6 task -s docs:cli:drift
docs:cli:drift: compared 1 generated file
docs:cli:drift: docs/CLI-REFERENCE.md byte-identical to a fresh regeneration (temporary file only — source tree untouched)

$ GOTOOLCHAIN=go1.26.6 go test -count=1 -v -run 'TestEveryRegisteredFlagIsAccountedFor' ./internal/cli/
cli_reference_test.go:236: walked 36 commands (hidden included), inspected 115 flags: 114 accepted via ../../docs/CLI-REFERENCE.md, 1 accepted via testdata/cli-reference-allowlist.txt
--- PASS: TestEveryRegisteredFlagIsAccountedFor (0.02s)
ok  	github.com/seanb4t/codegraph-go/internal/cli	0.460s
```

`threats_open: 0` (confirmed in `12-SECURITY.md` frontmatter, matching a literal count of 0 `open`-status `high`-severity rows in the Threat Register). `git status --porcelain` was empty immediately before this SUMMARY's own changes were staged. `.planning/todos/pending/` holds exactly 1 file (the graphstore archtest todo, unrelated to this phase) — the brew-trust todo did not reappear.

**Final commit SHA (this plan's task work):** `21f29846` (Task 2 — `12-SECURITY.md`/`12-VALIDATION.md`); `d48d1f7b` (Task 1 — `12-MUTATION-LOG.md`). The SUMMARY + `REQUIREMENTS.md` commit follows this SUMMARY's creation.

## Decisions Made

- **Family (b) targets `--no-open`, not `--editor-url`** — see key-decisions above. Recorded verbatim in `12-MUTATION-LOG.md`'s Family (b) section and cross-referenced as an explicit `unclassified` residual risk in `12-SECURITY.md`'s Notes, rather than silently working around it without recording why.
- **`threats_open: 0` tied to a literal grep count**, not hand-asserted — the register's Threat Register table format lets `rg -o '\| high \| [a-z]+ \|[^|]*\| open'` count actually-open high rows, which this plan verified equals 0 before writing the frontmatter value.

## Deviations from Plan

None — plan executed exactly as written. All three mutation families went RED exactly as the plan's `<action>` predicted (down to the exact counts-line numbers: 116 flags for family (a), 113-via-reference for family (b), unchanged 115/1-via-allowlist for family (c)); no family required an "observed negative" recording.

## Issues Encountered

None.

## User Setup Required

None — no external service configuration required.

## Requirements Note

`requirements-completed: [DOCS-05, DOCS-06, DOCS-07]` in this SUMMARY's frontmatter. This is the last of the three plans in this phase declaring all three IDs (12-01: DOCS-05/DOCS-06; 12-02: DOCS-05/DOCS-07; 12-03: DOCS-05/DOCS-06/DOCS-07). `gsd_run query requirements.ready-ids` returned `3/3 requirement(s) ready to mark complete` (both 12-01 and 12-02 already have their own `*-SUMMARY.md`), and `requirements.mark-complete DOCS-05 DOCS-06 DOCS-07` applied both the checkbox and traceability-table surfaces in `REQUIREMENTS.md` for all three IDs.

## Scope Note (per this plan's own dispatch instructions)

Per explicit objective-level scoping for this plan's execution, this SUMMARY and its accompanying commit do NOT touch `.planning/STATE.md` or `.planning/ROADMAP.md` — those are the orchestrator's writes. `REQUIREMENTS.md` is committed alongside this SUMMARY per the shared-ID gate's own completion step, which is this plan's own responsibility as the last plan declaring DOCS-05/06/07.

## Next Phase Readiness

- All three plans of Phase 12 (12-01, 12-02, 12-03) are complete with SUMMARY.md files.
- `docs/CLI-REFERENCE.md`, the drift gate, the accounting guard, the brew-trust wording, the README link, and the mutation-log/security/validation close-out artefacts are all in place and green.
- No blockers. Ready for the orchestrator's phase-level roadmap/state sync and `/gsd-verify-work`.

---
*Phase: 12-cli-reference-docs-tail*
*Completed: 2026-09-13*

## Self-Check: PASSED

Both key files confirmed present on disk (`12-MUTATION-LOG.md`, `12-SECURITY.md`); `12-VALIDATION.md` and `.planning/REQUIREMENTS.md` confirmed modified. Both task commits (`d48d1f7b`, `21f29846`) confirmed present in `git log`. `commits: 3` measured via `git rev-list --count c28ed62b..HEAD` against the plan-head ledger (`c28ed62b`, the commit immediately preceding this plan's first task) after this SUMMARY's own commit lands, accounting for the two task commits plus this metadata commit.
