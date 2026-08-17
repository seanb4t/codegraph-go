---
phase: 01-corpus-selection-by-measurement
plan: 06
subsystem: corpus-selection
tags: [corpus, threshold, github-api, manifest, measurement, peek, pebble, tree-sitter]

# Dependency graph
requires:
  - phase: 01-04
    provides: corpora/manifest.json (sole pin authority, D-09), the four-part fetch/assert Taskfile targets
  - phase: 01-05
    provides: -mode measure/select/kinds, ComputeThresholds, SelectLockedSet, the two typed documents, the prose renderer
provides:
  - corpora/observations.json — measured per-kind edge counts and per-language file counts for all eight eligible candidates
  - corpora/selection.json — frozen per-kind thresholds with rationale, locked set, rejected-candidate ledger, empty syntheticKinds
  - corpora/manifest.json — exactly four entries locked (hugo, guava, serilog, requests)
  - docs/CORPUS-MEASUREMENT.md — generated human-readable record with per-kind coverage table naming supplier + measured count
affects: [01-07 (corpus coverage guard), Phase 2 (golden freezing), Phase 3 (non-vacuity proof)]

# Actuals (#2632) — pairs with the plan's estimate to calibrate future estimates.
# Same estimateTokens scale (chars/4), never a harness token count.
actuals:
  tokens: 6690       # chars/4 over the realized plan diff (26760 chars)
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []         # no new dependencies — consumed the existing tools/corpora + internal/corpora machinery
  patterns:
    - "GitHub licence API as the live lock-time licence re-verification source (not the manifest's seeded value)"
    - "Deterministic minimum-cardinality corpus selection with tie-break (total tracked files asc, then repo name asc)"

key-files:
  created:
    - corpora/selection.json   # curated policy: thresholds + rationale, locked set, rejected ledger, empty syntheticKinds
    - docs/CORPUS-MEASUREMENT.md  # generated from observations + selection
  modified:
    - corpora/manifest.json    # locked flags flipped true on exactly the four selected repos
    - corpora/observations.json  # upserted with measurements for all eight candidates

key-decisions:
  - "Locked set is the minimum cardinality (4) that supplies every RankEdges kind at threshold and all five priority languages non-zero, per SelectLockedSet's deterministic tie-break — selected over candidate sets with more members and over heavier same-coverage alternatives."
  - "psf/requests (130 tracked files) locked over tiangolo/fastapi (3137) and pydantic (814) for the Python floor via the total-tracked-files-ascending tie-break."
  - "serilog/serilog (284 tracked files) locked over JamesNK/Newtonsoft.Json (1170) for the C# floor via the same tie-break; both were measured, both supply C#."
  - "hugo supplies the TS/JS non-zero floor through its 25 javascript files, so nest (1684 typescript) is redundant for coverage and not locked."
  - "apache/arrow retained in the manifest unlocked (D-09) and in the rejected ledger with its recorded no-C# disqualification."

patterns-established:
  - "Every candidate tried is named with its score — rejected candidates become evidence, not lost effort (D-17)."
  - "Thresholds must be satisfied by construction: min(max(2, best/2), best) with best over ALL eligible, never the locked subset."
  - "An unlocked candidate stays in the manifest rather than being deleted (D-09)."

requirements-completed: [FIXT-01]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "per-kind coverage threshold frozen by the deterministic rule and independently re-derived"
    requirement: "FIXT-01"
    verification:
      - kind: other
        ref: "re-ran `go run ./tools/corpora -mode select`; byte-equality asserted against committed corpora/selection.json over the 8-candidate eligible universe"
        status: pass
    human_judgment: false
  - id: D2
    description: "all nine RankEdges kinds covered by a named third-party supplier across the locked set, including overrides and type_of with no synthetic path"
    requirement: "FIXT-01"
    verification:
      - kind: other
        ref: "Task 2 verify block 3 recomputed every threshold from observations and asserted a named supplier + measured count >= threshold for each kind"
        status: pass
    human_judgment: false
  - id: D3
    description: "all five priority-4 languages have a non-zero measured file count across the locked set"
    requirement: "FIXT-01"
    verification:
      - kind: other
        ref: "Task 2 verify block 4 — go 889/hugo, java 3229/guava, csharp 216/serilog, python 37/requests, tsjs 27/hugo"
        status: pass
    human_judgment: false
  - id: D4
    description: "locked-set licences MIT or Apache-2.0, re-verified live at lock time"
    requirement: "FIXT-01"
    verification:
      - kind: other
        ref: "`gh api repos/<org>/<repo>/license` for all four locked repos; all Apache-2.0; manifest/selection agreement asserted"
        status: pass
    human_judgment: false

# Metrics
duration: 10min
completed: 2026-08-14
status: complete
---

# Phase [01] Plan [06]: Corpus Spike — Measured Threshold, Locked Set

**The corpus set is locked on recorded per-kind edge-count thresholds computed by a deterministic, independently re-derived rule — every candidate measured, every score recorded, and the two at-risk kinds (overrides, type_of) covered by named third-party repositories' measured counts with no synthetic path.**

## Performance

- **Duration:** 10 min
- **Started:** 2026-08-14T19:00:38Z
- **Completed:** 2026-08-14T19:10:37Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Measured all eight eligible candidates individually via `-repos` (`task corpora:fetch-one` → `corpora:assert-one` → `-mode measure` per repo), recorded each in `corpora/observations.json`.
- Settled `overrides` and `type_of` against real indexing output: guava provides overrides=4658, type_of=11857; Newtonsoft.Json 70/720; nest 86/21; both kinds are covered by multiple named third-party repositories.
- Computed per-kind thresholds by `corpora.ComputeThresholds` over ALL eligible candidates (`min(max(2, best/2), best)`) and proved the committed set is the global optimum by re-running `-mode select` and requiring byte equality.
- Locked the minimum-cardinality 4-repo set (hugo, guava, serilog, requests), flipped locked flags in the manifest, authored `corpora/selection.json`, and regenerated `docs/CORPUS-MEASUREMENT.md`.
- Re-verified every locked licence live against the GitHub licence API (all Apache-2.0) and every pinned SHA live at lock time via the four-part integrity check (`task corpora:fetch` / `corpora:assert` → 4/4 OK).
- Confirmed `task corpora:measure` regeneration is byte-identical across consecutive runs against the locked set.

## Task Commits

Each task was committed atomically:

1. **Task 1: Measure at-risk kinds overrides/type_of first, one candidate at a time** - `bf2b843` (feat)
2. **Task 2: Complete the language sweep, compute the threshold deterministically, lock the set** - `97bcb05` (feat)

**Plan metadata:** (no docs-only SUMMARY commit — plan files were already committed in this plan's HEAD)

## Files Created/Modified
- `corpora/selection.json` - Created. Curated policy: per-kind thresholds, threshold rationale naming each best value, locked set (4 repos at SHA), rejected-candidate ledger (arrow + 4 measured-not-locked), empty `syntheticKinds`.
- `docs/CORPUS-MEASUREMENT.md` - Created. Generated prose: threshold rationale, locked set, per-kind coverage table naming supplier + measured count for all nine kinds, per-language file counts with PASS, all-measured-candidates section, rejected ledger.
- `corpora/manifest.json` - Modified. Locked flags flipped `true` on gohugoio/hugo, google/guava, serilog/serilog, psf/requests; all others stay unlocked per D-09.
- `corpora/observations.json` - Modified. Upserted with measurements for all eight eligible candidates (never reconstructed).

## Decisions Made
- **Locked set = minimum cardinality (4).** SelectLockedSet enumerated subsets in increasing cardinality and returned the first satisfying all nine thresholds and all five priority languages, tie-broken by total tracked files ascending then repo name.
- **psf/requests over fastapi/pydantic for Python.** All three were measured; requests has the smallest tracked-file footprint (130 vs 3137/814) and still supplies a non-zero Python count (37 files), winning the total-tracked-files-ascending tie-break.
- **serilog over Newtonsoft.Json for C#.** Both supply C# and both were measured; serilog's far smaller footprint (284 vs 1170 tracked files) wins the same tie-break.
- **hugo supplies TS/JS**, so nest (1684 typescript files) is redundant for coverage and correctly left unlocked.
- **Every candidate trial recorded.** apache/arrow stays in the manifest unlocked (D-09) and appears in the rejected ledger with its recorded no-C# disqualification; the four measured-not-locked candidates each carry their measured score in the ledger.

## Deviations from Plan

None - plan executed exactly as written. No Rule 1-4 auto-fixes and no halts were needed: both at-risk kinds measured non-zero from named third-party repositories on the first pass, so the bounded search never exhausted and no `best == 0` / unsatisfiable-threshold condition fired.

## Issues Encountered
- None. All live GitHub API licence lookups, real shallow git fetches, individual measurements, the deterministic re-derivation, and the locked-set fetch/assert prescribed by the plan succeeded on the first attempt.

## User Setup Required
None - no external service configuration required. `gh auth status` was already authenticated, satisfying the Task 1 precondition for network access to github.com.

## Next Phase Readiness
- **Plan 01-07 (corpus coverage guard)** consumes this plan's outputs directly: `corpora/selection.json` (thresholds + locked set + empty syntheticKinds) and `corpora/manifest.json` (locked flags) are both committed and in agreement. 01-07's guard requires exact equality between the selection's locked set and the manifest's locked flags, the recomputed-threshold check, and a failure on any non-empty `syntheticKinds` — all of which this plan's artifacts already satisfy.
- **Phase 2** can freeze every golden against the now-frozen thresholds and locked repository-at-SHA identities; `task corpora:fetch` / `corpora:assert` are live and non-empty (4/4 OK), and the four locked corpora are present at their pinned SHAs.
- **Phase 3** non-vacuity proof has a named locked set with measured per-kind suppliers to test against.

## Self-Check

- FOUND: corpora/selection.json
- FOUND: corpora/observations.json
- FOUND: docs/CORPUS-MEASUREMENT.md
- FOUND: corpora/manifest.json (4 locked entries, all Apache-2.0)
- FOUND: commit bf2b843 (Task 1)
- FOUND: commit 97bcb05 (Task 2)

## Self-Check: PASSED

---
*Phase: 01-corpus-selection-by-measurement*
*Completed: 2026-08-14*