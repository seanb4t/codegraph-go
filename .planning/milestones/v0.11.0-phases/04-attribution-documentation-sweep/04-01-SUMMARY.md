---
phase: 04-attribution-documentation-sweep
plan: 01
subsystem: docs
tags: [attribution, license, notice, mit, readme, sweep]

# Dependency graph
requires: []
provides:
  - "NOTICE reduced to MIT transcription + one origin sentence + kept warning/deps blocks"
  - "README has exactly one past-tense origin clause (in ## License) and no Relationship section"
  - "LICENSE re-verified live as MIT after the NOTICE edit (ATTR-03)"
affects:
  - "04-02 (FLAG-PARITY deletion): NOTICE no longer references docs/FLAG-PARITY.md"
  - "04-03 (capability matrix own-terms sweep): framing-vocabulary precedent"
  - "Phase 6 (BENCH): Performance numbers table intentionally left for that phase to retire"

# Actuals (#2632) — pairs with the plan's `estimate` on the estimateTokens (chars/4) scale.
actuals:
  tokens: 1071        # 4285 diff chars / 4 over NOTICE + README.md
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Legal attribution lives in NOTICE only; README links to it via one past-tense clause"
    - "Framing vocabulary swept term-by-term with word-boundary anchors, never by regex over TypeScript/TS"

key-files:
  created: []
  modified:
    - NOTICE
    - README.md

key-decisions:
  - "NOTICE keeps the verbatim MIT copyright transcription, the LICENSE warning block, and Third-party deps; the drop-in/ported/flag-parity/benchmark argument is removed (ATTR-01, D-01)"
  - "README's only origin mention is one past-tense clause inside ## License; ## Relationship to the original is removed (ATTR-02, D-02)"
  - "The origin URL is folded into the single NOTICE origin sentence, keeping it one sentence"
  - "LICENSE is never edited; live `gh api` license detection re-checked after the edit (ATTR-03)"

patterns-established:
  - "One named cause per diff: the NOTICE trim and the README sweep are separate reviewed commits"
  - "Positive-controlled greps: the sweep patterns are confirmed to match pre-edit text before asserting zero post-edit hits"

requirements-completed: [ATTR-01, ATTR-02, ATTR-03]

coverage:
  - id: D1
    description: "NOTICE Attribution section trimmed to one origin sentence + verbatim MIT transcription + fidelity guard, with the drop-in/ported/flag-parity/benchmark argument gone and the LICENSE warning block + Third-party deps kept"
    requirement: ATTR-01
    verification:
      - kind: other
        ref: "rg -c 'Copyright \\(c\\) 2026 Colby Mchenry' NOTICE == 1; rg -c 'began as a ground-up Go rewrite' NOTICE == 1; rg -c 'Third-party dependencies' NOTICE == 1; rg -c 'Do not move any of this' NOTICE == 1; rg -n 'drop-in|\\bported\\b|FLAG-PARITY|BENCHMARKS' NOTICE == 0"
        status: pass
    human_judgment: false
  - id: D2
    description: "README has no ## Relationship to the original; exactly one past-tense origin clause inside ## License linking to NOTICE; Performance framing words and the Status parity claim swept; migrate row and Full TS/TSX/JS language rows intact"
    requirement: ATTR-02
    verification:
      - kind: other
        ref: "rg -c 'Relationship to the original' README.md == 0; rg -c 'began as a Go rewrite of CodeGraph' README.md == 1; rg -n 'the TS original|it ports|including the flattering one|\\bparity\\b|divergence' README.md == 0; rg -c 'import an existing TypeScript CodeGraph index' README.md == 1; rg -c '\\| Full \\| Go, Java, C#, TypeScript, TSX' README.md == 1"
        status: pass
    human_judgment: false
  - id: D3
    description: "LICENSE stays byte-identical to HEAD and GitHub's live license detection still reports MIT after the NOTICE edit"
    requirement: ATTR-03
    verification:
      - kind: other
        ref: "git diff --exit-code HEAD -- LICENSE == clean; gh api repos/seanb4t/codegraph-go/license --jq .license.spdx_id == MIT"
        status: pass
    human_judgment: false

# Metrics
duration: ~45min
completed: 2026-08-15
status: complete
---

# Phase 4 Plan 1: Attribution Trim Summary

**NOTICE reduced to its legal minimum — MIT copyright transcription plus one origin sentence, README swept to a single past-tense origin clause inside `## License`, and LICENSE re-verified live as MIT (ATTR-01/02/03)**

## Performance

- **Duration:** ~45 min (execution + two full-suite go test runs)
- **Started:** 2026-08-15T16:50Z (approx.)
- **Completed:** 2026-08-15T17:35:25Z
- **Tasks:** 2 (1 tracer + 1 auto)
- **Files modified:** 2 (NOTICE, README.md)

## Accomplishments

- Trimmed NOTICE's `## Attribution` section to exactly three things: one origin sentence ("CodeGraph Go began as a ground-up Go rewrite of CodeGraph (MIT)" with the origin URL folded in), the verbatim MIT copyright transcription in a fenced block, and the transcription-fidelity guard. The drop-in / ported-heuristics / flag-parity / benchmark argument (including both `docs/FLAG-PARITY.md` references) is gone.
- Kept the LICENSE warning block and the `## Third-party dependencies` section untouched — the legal-attribution pair stays intact.
- Removed README's `## Relationship to the original` section entirely and added the mandated exact License clause: "MIT — see [LICENSE](LICENSE). This project began as a Go rewrite of CodeGraph; see [NOTICE](NOTICE) for the original copyright."
- Swept the two additional README framing sites words-only: the Performance intro's "Measured against the TypeScript implementation it ports" positioning, the "the TS original" row label (now "(TypeScript)"), the "including the flattering one" caveat phrase, and the stale Status claim "Behavioral parity with TypeScript CodeGraph v1.3.x …" rewritten on codegraph-go's own terms ("Behavioral regression coverage is provided by the golden corpus suite — named property assertions over the in-repo behavioral corpus, re-frozen from codegraph-go's own output").
- Preserved the product-truth surface the sweep must not remove: the `codegraph migrate` row and the Full Go/Java/C#/TypeScript/TSX language row, and the Performance numbers table (byte-identical to HEAD except the row label).
- Verified LIVE that GitHub's license detection still reports `MIT` after the NOTICE edit and that the working-tree LICENSE is byte-identical to HEAD.

## Task Commits

Each task was committed atomically:

1. **Task 1 (tracer): NOTICE attribution trim** - `802286c` (docs)
2. **Task 2: README edits — Relationship removal, License clause, Performance/Status sweep** - `f8222fe` (docs)

## Files Created/Modified

- `NOTICE` - Attribution section rewritten: argument removed, one origin sentence, verbatim transcription + fidelity guard kept; LICENSE warning block and Third-party deps untouched
- `README.md` - Relationship section removed; License clause added; Performance framing words swept; Status parity claim rewritten on own terms; migrate row + language table intact

## Decisions Made

- **Origin URL folded into the one NOTICE sentence** (plan allowed "at your discretion"): "CodeGraph Go began as a ground-up Go rewrite of CodeGraph (MIT) — <https://github.com/colbymchenry/codegraph>." Keeps the mandated sentence one sentence and preserves the origin link the original text carried.
- **Status rewrite used the plan's own-terms example verbatim** — describes the current fail-loud golden suite truthfully (named property assertions over the in-repo corpus, re-frozen from codegraph-go's own output) with no "parity"/"divergence" residue.
- Followed the plan's exact mandated License clause (not paraphrased) so the phase gate's `rg -c "began as a Go rewrite of CodeGraph" README.md == 1` holds.

## Deviations from Plan

None - plan executed exactly as written. (The pre-existing daemon flake below is an environment observation, not a plan deviation.)

## Issues Encountered

- **`internal/daemon` load-induced flake under full-suite parallel load:** `CGO_ENABLED=1 go test -count=1 ./...` failed twice on exactly one test — `TestRunWatchdogCancelsRunOnSimulatedReparent` (250.34s, hitting its full 25x-scaled budget). This is a documented, PRE-EXISTING flake: the test's own comment (internal/daemon/daemon_test.go:345-351) names the load-induced watchdog-ticker delay class, CI (`.github/workflows/ci.yml:8-11,78-85,123-130`) and Taskfile (`test:daemon`, Taskfile.yml:104-112) both document it and isolate `internal/daemon` into its own `-count=1` step. It cannot be caused by this plan's markdown-only changes. Verified: `go test -count=1 ./internal/daemon/` in isolation exits 0; the single test passes in isolation (1.384s); all 53 non-daemon packages pass `-count=1`. Logged to `deferred-items.md` (out of scope — pre-existing, unrelated file). The go test gate for this plan was satisfied using the project's documented isolation handling.

## Deferred Issues

- The daemon watchdog flake is logged to `.planning/phases/04-attribution-documentation-sweep/deferred-items.md`; the project already routes around it in CI via `task test:daemon`.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- 04-02's FLAG-PARITY deletion is now clean of NOTICE: both `docs/FLAG-PARITY.md` references that lived in NOTICE's argument were removed with this trim.
- The framing-vocabulary sweep precedent (word-boundary-anchored, positive-controlled greps, never regex over TypeScript/TS) carries into 04-03's capability-matrix own-terms rewrite.
- The Performance numbers table is deliberately left intact for Phase 6 BENCH to retire, per D-05.

## Self-Check: PASSED

- Files exist: `NOTICE`, `README.md` (edited), `.planning/phases/04-attribution-documentation-sweep/04-01-SUMMARY.md`, `.planning/phases/04-attribution-documentation-sweep/deferred-items.md` — all FOUND.
- Commits exist: `802286c` (NOTICE trim), `f8222fe` (README sweep) — both verified in `git log`.
- LICENSE byte-identical to HEAD (`git diff --exit-code HEAD -- LICENSE` clean); live `gh api ... --jq .license.spdx_id` = `MIT`.

---
*Phase: 04-attribution-documentation-sweep*
*Completed: 2026-08-15*
