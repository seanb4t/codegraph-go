---
phase: 10-index-health-the-coverage-denominator
plan: 06
subsystem: testing
tags: [mutation-testing, security-register, validation, phase-close, tdd]

# Dependency graph
requires:
  - phase: 10-index-health-the-coverage-denominator
    provides: "Plans 01-05's full coverage-denominator implementation (schema, graphstore c/ namespace, DiscoverAll, Sync diff, Engine.CoverageSummary/CoverageRows, GetHealth/GetCoverage rpcs, the health-page Coverage section) — this plan closes the phase with proof, not new behavior"
provides:
  - "10-MUTATION-LOG.md: three hand-authored RED demonstrations (D-15 unset-has_coverage-as-zero, D-16 GetIndexCoverage decoy, D-14 present-tense-disk-filter) proving the phase's own guards discriminate, each byte-cleanly reverted"
  - "10-SECURITY.md: the union of Plans 01-05's threat_model rows deduplicated by id (T-10-01..15, T-10-SC — 16 rows), every row closed by a named test or a recorded Verdict, threats_open: 0 at ASVS L1"
  - "10-VALIDATION.md: Per-Task Verification Map filled with real plan/task/test names, Wave 0 checklist ticked for every landed item"
  - "Phase-close gate proof: go vet, Go unit suite, golden suite, proto:drift, vitest suite, svelte-check (0 errors), and web:drift all green at the closing commit"
affects: []

# Actuals (#2632)
actuals:
  tokens: 11341
  tasks: 2
  commits: 2
plan_head_before: 4eba7fb826417d2495547d0880ec6184c9e0ce13

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Guard-discrimination proof: for each guard the phase introduces, apply the exact mutation the guard exists to catch, capture verbatim RED output, revert with git checkout --, and re-confirm GREEN — never trust a guard that has not been watched failing (D-17)."
    - "A structural source-text scan (D-14b) and a package-wide import-boundary archtest are different guard shapes for different mutation classes: the archtest catches a forbidden PACKAGE dependency (internal/indexer root), while the source scan catches a forbidden CALL (os.Stat/os.ReadDir/etc.) smuggled in via an otherwise-unremarkable stdlib import the archtest would never flag."

key-files:
  created:
    - .planning/phases/10-index-health-the-coverage-denominator/10-MUTATION-LOG.md
    - .planning/phases/10-index-health-the-coverage-denominator/10-SECURITY.md
  modified:
    - .planning/phases/10-index-health-the-coverage-denominator/10-VALIDATION.md

key-decisions:
  - "T-10-06 collided as an id across Plan 02 (DoS: oversized file reaching the tree-sitter scanner) and Plan 04 (Elevation of Privilege: absence of a 're-index this file' control) — merged into one register row spanning both threats under a combined category, both mitigations cited, rather than silently dropping one or inventing a new id not present in any plan's threat_model."
  - "T-10-01, T-10-03, T-10-04, T-10-05, T-10-07, T-10-08, T-10-10, and T-10-15 each appeared in two or three plans' threat_model blocks describing the same threat from different layers (write-side vs render-side, Engine-level vs wire-level, from-scratch commit vs Sync commit) — merged into single rows with each plan's mitigation cited and tagged, per the task's own 'deduplicate by id, merge mitigation text' instruction."
  - "The 10-06 PLAN's own local threat_model (T-10-11 'RED transcript repudiation', T-10-16 'prose-only SECURITY.md row') was NOT merged into the T-10-01..15+SC register — it is a distinct, smaller meta-registry about this plan's OWN artifacts' integrity, out of scope for the union of Plans 01-05's threat_model blocks the read_first instructions explicitly named. (10-01's own, unrelated T-10-11 — 'Import of a stream with a forged/unknown record kind' — IS in the register.)"
  - "family (a)'s mutation removed the unset/false has_coverage short-circuit in BOTH CoverageSummary and CoverageRows (not just one) so a single instrument command exercises D-15 at both the summary and the paged-rows read paths, matching the plan's own instrument list citing tests for both functions."
  - "task web:test and task web:drift must be invoked with the -s (silent) flag when their output is grep-counted — task v3.52 echoes the underlying script by default, doubling literal string matches (e.g. 'web:test: PASS' appearing twice: once in the echoed script, once in real output). Discovered while running the phase-close gate; corrected before pasting tails below."

requirements-completed: [HLT-04, HLT-05, HLT-06]

coverage:
  - id: D1
    description: "10-MUTATION-LOG.md: three RED demonstrations (D-15, D-16, D-14), each with a pre-mutation cleanliness gate, the exact mutation diff, verbatim RED output, a git checkout -- revert, and a GREEN re-run confirmation; family (c) explicitly records that internal/query/archtest stayed green under its mutation and why"
    requirement: HLT-05
    verification:
      - kind: unit
        ref: "internal/query/coverage_test.go#TestCoverageSummaryOnOldGraphIsUnknown (watched RED then GREEN, family a)"
        status: pass
      - kind: unit
        ref: "internal/uiserver/readonly_test.go#TestUIServiceMethodSetIsExactlyTheReadSet (watched RED then GREEN, family b)"
        status: pass
      - kind: unit
        ref: "internal/query/coverage_test.go#TestCoverageSourceNeverWalksDisk (watched RED then GREEN, family c)"
        status: pass
    human_judgment: false
  - id: D2
    description: "10-SECURITY.md: 16-row threat register (T-10-01..15, T-10-SC) covering the union of Plans 01-05's threat_model blocks, every row closed by a test or a Verdict, threats_open: 0"
    requirement: HLT-05
    verification:
      - kind: other
        ref: "gsd verify gate: rg count of distinct T-10-xx ids == 16 AND zero rows lacking a Test.../Verdict: citation (both re-run and passing at commit 0738076)"
        status: pass
    human_judgment: false
  - id: D3
    description: "10-VALIDATION.md: Per-Task Verification Map has zero TBD cells, real plan/task/test names, and Wave 0 checklist ticked for every landed item, without touching the tool-owned frontmatter or headings"
    requirement: HLT-06
    verification:
      - kind: other
        ref: "gsd verify gate: rg count of TBD cells == 0, real-task-id rows >= 8, ticked Wave-0 items >= 8, frontmatter status untouched, 6 headings unchanged"
        status: pass
    human_judgment: false
  - id: D4
    description: "Phase-close gate: go vet, task test:unit, task test:golden, task proto:drift, task web:test, pnpm -C web check (0 errors), task web:drift (both halves MATCH) all green at the closing commit"
    requirement: HLT-04
    verification:
      - kind: other
        ref: "manual gate run, pasted verbatim below (## Phase-close gate section)"
        status: pass
    human_judgment: false

# Metrics
duration: ~50min
completed: 2026-09-13
status: complete
---

# Phase 10 Plan 06: Mutation Log, Security Register, Validation Map & Phase-Close Gate Summary

**Three hand-authored RED demonstrations proving the phase's own guards discriminate (D-15/D-16/D-14), a 16-row security register closing every T-10-xx threat by test or verdict, the validation map filled with what actually landed, and all seven phase-close gates green.**

## Performance

- **Duration:** ~50 min
- **Completed:** 2026-09-13T03:57:30Z
- **Tasks:** 2/2
- **Files modified:** 3 (2 created, 1 modified)

## Accomplishments

- `10-MUTATION-LOG.md` authored in the 07/08/09 house format: a pre-mutation cleanliness gate, three family sections (D-15 unset-`has_coverage`, D-16 `GetIndexCoverage` decoy, D-14 present-tense disk filter), each with the exact mutation diff, verbatim RED output, a `git checkout --` revert, and a GREEN re-confirmation — all byte-clean at close.
- `10-SECURITY.md` authored in the 09 house format: the union of Plans 01-05's `<threat_model>` rows deduplicated by id into 16 rows (T-10-01..15, T-10-SC), every row closed by a named `Test…` function (verified real via `rg` against HEAD) or an explicit `Verdict:`, `threats_open: 0` at ASVS L1, with the write-path trust surface as the register's own boundary sentence.
- `10-VALIDATION.md`'s Per-Task Verification Map filled: every `TBD` cell replaced with the real plan/task/test names and commands from Plans 01-05; the Wave 0 checklist ticked for every landed item; the tool-owned frontmatter and heading structure left untouched.
- All seven phase-close commands (`go vet`, Go unit suite, golden suite, `proto:drift`, `web:test`, `pnpm -C web check`, `web:drift`) confirmed green at the closing commit, with tails pasted below.

## Task Commits

1. **Task 1: 10-MUTATION-LOG.md — families (a) D-15, (b) D-16, (c) D-14, each RED then byte-clean reverted** - `7e515769` (test)
2. **Task 2: 10-SECURITY.md, 10-VALIDATION.md map filled, and the phase-close gate** - `0738076` (docs)

**Plan metadata:** (this commit, following)

## Files Created/Modified

- `.planning/phases/10-index-health-the-coverage-denominator/10-MUTATION-LOG.md` - Three RED-then-reverted mutation demonstrations
- `.planning/phases/10-index-health-the-coverage-denominator/10-SECURITY.md` - 16-row threat register, closed at ASVS L1
- `.planning/phases/10-index-health-the-coverage-denominator/10-VALIDATION.md` - Per-Task Verification Map and Wave 0 checklist filled

## Decisions Made

- **T-10-06 id collision, merged not dropped.** Plan 02 and Plan 04 independently assigned `T-10-06` to two unrelated threats (a DoS via an oversized file reaching the tree-sitter scanner, and an Elevation-of-Privilege absence-of-control on the coverage view). Rather than silently keep only one or invent an unlisted id, the register merges both under one row with a combined category and both mitigations cited — preserving the 16-id count the phase's plans actually declared.
- **Eight threat ids spanned multiple plans describing the same threat from different layers** (T-10-01 write-vs-render, T-10-03 server-vs-client page-size clamp, T-10-04 Engine-vs-wire token validation, T-10-05 Engine-vs-wire detail scrubbing, T-10-07 archtest-vs-behavioural-vs-Sync-prune reason integrity, T-10-08 schema-vs-UI enum skew, T-10-10 from-scratch-vs-Sync torn-write, T-10-15 UI-vs-wire unknown-state) — each merged into one row citing every contributing plan's mitigation, per the task's own deduplicate-and-merge instruction.
- **10-06's own local threat_model (T-10-11 RED-transcript repudiation, T-10-16 prose-only-row repudiation) was excluded from the T-10-01..15+SC register.** These are a distinct, smaller meta-registry about this plan's own artifact quality (satisfied by construction: the MUTATION-LOG's verbatim-paste convention and this SECURITY.md's own verify gate), not part of the union of Plans 01-05's threat_model blocks the read_first instructions scoped the register to. Plan 01's own, unrelated `T-10-11` ("Import of a stream with a forged/unknown record kind") IS in the register — the id collision is between 10-06's local threat_model and 10-01's, not within the register itself.
- **Family (a)'s D-15 mutation touched both `CoverageSummary` and `CoverageRows`**, not just one function, so a single mutation exercises the unset-`has_coverage` short-circuit at both the summary and paged-rows read paths in one RED/GREEN cycle, matching the plan's own instrument list (which cites tests for both).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking issue] `task web:test`/`task web:drift` output double-counted a grep target under default (non-silent) invocation**
- **Found during:** Task 2, phase-close gate
- **Issue:** `task web:test > file 2>&1` (without `-s`) causes Task v3.52 to echo the underlying shell script into the captured output before running it, so a literal string like `web:test: PASS` appears twice (once in the echoed script source, once in the real result line) — a positive-count grep gate (`rg -o '...' | wc -l`) would see 2 instead of the expected 1.
- **Fix:** Re-ran both `task web:test` and `task web:drift` with the `-s` (silent) flag, as the repo's own `repo_landmines` guidance for this exact plan already named ("`task` (v3.52) echoes scripts — use `task -s` for any grep-counted output"). Confirmed `web:test: PASS` and `web:drift: ... half MATCH` counts matched the plan's expected values after the fix.
- **Files modified:** none (gate-invocation correction only, no source change)
- **Verification:** `rg -o 'web:test: PASS' /tmp/10-06-webtest.txt | wc -l` == 1; `rg -o '\bhalf MATCH\b' /tmp/10-06-drift.txt | wc -l` == 2 (source half + output half)
- **Committed in:** N/A (no source file affected; documented here per the fix-attempt-limit / deviation-tracking convention)

---

**Total deviations:** 1 auto-fixed (1 Rule 3 — blocking gate-invocation issue, no source change)
**Impact on plan:** No scope creep; the fix was a correction to how a phase-close gate command was invoked, not to any production or test code. All seven phase-close gates verified green with the corrected invocation.

## Issues Encountered

None beyond the gate-invocation deviation above.

## User Setup Required

None - no external service configuration required.

## Phase-close gate

All seven commands re-run at the closing commit `0738076`, tails pasted verbatim:

**1. `GOTOOLCHAIN=go1.26.6 go vet ./...`**
```
(no output — exit 0)
```

**2. `GOTOOLCHAIN=go1.26.6 task test:unit`**
```
ok  	github.com/seanb4t/codegraph-go/tools/bench/runner	(cached)
ok  	github.com/seanb4t/codegraph-go/tools/corpora	(cached)
ok  	github.com/seanb4t/codegraph-go/tools/mcpaudit	(cached)
ok  	github.com/seanb4t/codegraph-go/tools/transcriptfreeze	(cached)
ok  	github.com/seanb4t/codegraph-go/web	(cached)
```
(zero `FAIL` lines across the full output; exit 0)

**3. `GOTOOLCHAIN=go1.26.6 task test:golden`**
```
task: [test:golden] go test -count=1 ./testdata/golden/...
ok  	github.com/seanb4t/codegraph-go/testdata/golden	23.337s
?   	github.com/seanb4t/codegraph-go/testdata/golden/gocapture	[no test files]
```

**4. `GOTOOLCHAIN=go1.26.6 task proto:drift`**
```
proto:drift: compared 4 generated files
proto:drift: all 4 generated files byte-identical to the pinned toolchain's regeneration (temporary tree only — source tree untouched)
```

**5. `task -s web:test`**
```
web:test: observed numTotalTests=566 numPassedTests=566 (vitest exit 0)
web:test: PASS — 566 of 566 tests passed
```

**6. `pnpm -C web check`**
```
$ svelte-kit sync && svelte-check --tsconfig ./tsconfig.json
1789271799987 START "/Volumes/Code/github.com/seanb4t/codegraph-go/web"
1789271799990 COMPLETED 1170 FILES 0 ERRORS 0 WARNINGS 0 FILES_WITH_PROBLEMS
```

**7. `task -s web:drift`**
```
web:drift: hashed 115 source files
web:drift: manifested 32 output files
web:drift: source half MATCH (115 files, d04e36b7fab5d2f70ce08271f1e96fd7d4505ae83ae3dc523ca3fe5a75486fa6)
web:drift: output half MATCH (32 files, 447d1065b253aff3143e3e56f58c96bbe08bd41f0b36d806e800ff9525a3bb5d)
web:drift: PASS — hashed 115 source files, manifested 32 output files, committed web/build/ matches both digests
```

**Security state at close:** `threats_open: 0` (10-SECURITY.md frontmatter, ASVS L1, 16/16 threats closed).

## Next Phase Readiness

Phase 10 (Index Health — The Coverage Denominator) is complete: all six plans landed (01-06), all three requirements (HLT-04, HLT-05, HLT-06) closed, the security register shows `threats_open: 0`, the validation map reflects real landed tests, and the full phase-close gate suite is green. No blockers for the next phase.

---
*Phase: 10-index-health-the-coverage-denominator*
*Completed: 2026-09-13*

## Self-Check: PASSED

- FOUND: `.planning/phases/10-index-health-the-coverage-denominator/10-MUTATION-LOG.md`
- FOUND: `.planning/phases/10-index-health-the-coverage-denominator/10-SECURITY.md`
- FOUND: `.planning/phases/10-index-health-the-coverage-denominator/10-VALIDATION.md`
- FOUND: commit `7e515769` (Task 1)
- FOUND: commit `0738076` (Task 2)
