---
phase: 08-release-hardening-benchmarks
plan: 09
subsystem: infra
tags: [docs, cosign, slsa, sbom, benchmarking, dependency-audit]

# Dependency graph
requires:
  - phase: 08-release-hardening-benchmarks (plan 04)
    provides: ".github/workflows/release.yml — the real signing/SLSA/SBOM pipeline RELEASE.md's verify commands describe"
  - phase: 08-release-hardening-benchmarks (plan 07)
    provides: "tools/bench/runner + tools/bench/baseline.json — the real benchmark numbers/commands BENCHMARKS.md documents"
provides:
  - "docs/RELEASE.md — cosign/slsa-verifier/SBOM verify commands matching internal/upgrade/verify.go's real identity, the DIST-05 audited-dependency narrative (134 total requires: 27 direct/107 indirect, 14 of 27 direct are tree-sitter grammar modules), and the linux/amd64-blocking reproducibility posture"
  - "docs/BENCHMARKS.md — methodology (median-of-5, OS-level external peak RSS with rationale), pinned real-repo table, honestly-TBD head-to-head numbers table + regenerate command, the one real committed baseline.json number, and the regression gate + explicit re-bless path"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Doc-only plan: no code changes, two new files under docs/ cross-referencing already-shipped Plan 08-04/08-07/08-08 artifacts as the single source of truth"

key-files:
  created:
    - docs/RELEASE.md
    - docs/BENCHMARKS.md
  modified: []

key-decisions:
  - "Reported the DIST-05 dependency count precisely rather than repeating CONTEXT.md's imprecise '134 direct requires' phrasing verbatim: go.mod's require blocks total 134 entries, of which 27 are direct and 107 are indirect/transitive; 14 of the 27 direct requires are tree-sitter grammar modules. This is more honest than restating the plan's own inexact framing while still landing on the same overall narrative (grammar modules dominate the direct dependency list)."
  - "BENCHMARKS.md's raw head-to-head numbers table is marked TBD with the exact regenerate command, not populated with the manual run mentioned in 08-07-SUMMARY.md — that run's output was never captured to a committed file (only its pass/fail outcome was recorded), so no real per-repo numbers exist to cite verbatim. Fabricating plausible-looking numbers would violate the plan's own prohibition; the one real committed number (tools/bench/baseline.json, the synthetic-corpus regression baseline) is included verbatim instead."
  - "RELEASE.md documents only the linux/arm64 leg as having an automated (non-blocking) double-build check today (per ci.yml as shipped in Plan 08-08) — windows/darwin best-effort reproducibility is stated as a posture, not an implemented CI check, since ci.yml does not currently double-build those targets."

patterns-established: []

requirements-completed: [DIST-05, DIST-02, DIST-03, PERF-01]

coverage:
  - id: D1
    description: "docs/RELEASE.md gives copy-pasteable cosign verify-blob, slsa-verifier verify-artifact, and SBOM inspection commands matching internal/upgrade/verify.go's exact issuer + SAN identity, plus the DIST-05 audited-dependency narrative and the linux/amd64-blocking/other-targets-reported reproducibility posture"
    requirement: "DIST-02"
    verification:
      - kind: other
        ref: "test -f docs/RELEASE.md && grep -qi cosign docs/RELEASE.md && grep -qi slsa docs/RELEASE.md && grep -qi tree-sitter docs/RELEASE.md && grep -qi reproduc docs/RELEASE.md — all pass, this session"
        status: pass
    human_judgment: false
  - id: D2
    description: "docs/RELEASE.md's DIST-05 narrative states CGo/tree-sitter as the sole documented dependency exception and explains the grammar-module-dominated composition of the direct dependency list, rather than implying a small flat tree"
    requirement: "DIST-05"
    verification:
      - kind: other
        ref: "Manual count this session: awk/grep over go.mod confirms 134 total require entries (27 direct, 107 indirect), 14 of 27 direct are tree-sitter-prefixed modules — matches docs/RELEASE.md §2's stated breakdown exactly"
        status: pass
    human_judgment: false
  - id: D3
    description: "docs/BENCHMARKS.md documents median-of-5 + OS-level external peak RSS methodology (with the in-process-vs-external rationale), the pinned real-repo manifest with commit SHAs, a per-repo Go-vs-TS numbers table (honestly TBD, not fabricated, with the exact regenerate command), the regression gate's tolerance bands/absolute ceiling, and the explicit human re-bless command"
    requirement: "PERF-01"
    verification:
      - kind: other
        ref: "test -f docs/BENCHMARKS.md && grep -qi median docs/BENCHMARKS.md && grep -qi rss docs/BENCHMARKS.md && grep -Eqi 'rebless|re-bless' docs/BENCHMARKS.md && grep -qi '1.3.1' docs/BENCHMARKS.md — all pass, this session"
        status: pass
    human_judgment: false
  - id: D4
    description: "Neither document invents a verify command the pipeline doesn't produce artifacts for, nor fabricates a benchmark number that wasn't actually measured"
    requirement: "DIST-03"
    verification: []
    human_judgment: true
    rationale: "Cross-checking RELEASE.md's verify commands against the real, already-shipped release.yml/verify.go, and confirming BENCHMARKS.md's TBD framing is genuinely honest (not merely well-worded), is a documentation-accuracy judgment call best made by a human reviewer reading both the docs and the referenced source files side by side — no automated check can prove a doc's narrative claims are honest, only that certain keywords are present."

# Metrics
duration: 6min
completed: 2026-07-13
status: complete
---

# Phase 08 Plan 09: Release & Benchmark Documentation Summary

**`docs/RELEASE.md` gives users the exact cosign/slsa-verifier/SBOM commands matching the shipped `internal/upgrade/verify.go` identity plus an honest DIST-05 dependency-tree narrative (134 total go.mod requires: 27 direct/107 indirect, 14 of 27 direct are tree-sitter grammar modules); `docs/BENCHMARKS.md` documents the real median-of-5/external-RSS methodology and regression gate, with the head-to-head numbers table honestly marked TBD since no live run has been captured to a committed file yet.**

## Performance

- **Duration:** 6 min
- **Started:** 2026-07-13T21:34:05Z
- **Completed:** 2026-07-13T21:40:00Z
- **Tasks:** 2 completed
- **Files modified:** 2 (both new)

## Accomplishments
- `docs/RELEASE.md`: exact `cosign verify-blob` command asserting the same issuer (`https://token.actions.githubusercontent.com`) and SAN regex (`release.yml @ refs/tags/v*`) `internal/upgrade/verify.go` enforces in-process; `slsa-verifier verify-artifact` command against the checksums-file provenance; SBOM inspection via `jq` over the per-binary `.spdx.json`; a precise (not hand-wavy) dependency-tree breakdown — 134 total `go.mod` require entries, 27 direct, 107 indirect, 14 of the 27 direct requires are tree-sitter grammar modules — with CGo/tree-sitter named as the sole documented exception; reproducibility posture stating linux/amd64 as the sole blocking double-build target, with a local-reproduce recipe a user can run themselves
- `docs/BENCHMARKS.md`: methodology section explains median-of-5 and why peak RSS must be measured externally (OS-level `getrusage`/`ru_maxrss` via `bench.PeakRSSBytes`) rather than in-process, since there's no in-process equivalent for the TS/Node subject at all; documents the three pinned real repos (weft-go, colbymchenry-codegraph, cockroachdb-pebble) with full commit SHAs from `tools/bench/realcorpus/manifest.go`; the raw per-repo Go-vs-TS table is honestly marked `TBD` (no live run's output was ever captured to a file) with the exact `go run ./tools/bench/runner -mode headtohead` command to regenerate it; the one genuinely real, committed number — `tools/bench/baseline.json`'s 120,000-file synthetic-corpus regression baseline — is reproduced verbatim; regression-gate section documents the 10%/15% tolerance bands, the independent absolute peak-RSS ceiling, and the explicit human-only `-rebless` command
- Both documents cross-reference only artifacts the already-shipped `.github/workflows/release.yml` (Plan 08-04) and `tools/bench/runner` (Plan 08-07)/`ci.yml`+`bench.yml` (Plan 08-08) actually produce — no invented verify command or fabricated benchmark number

## Task Commits

Each task was committed atomically:

1. **Task 1: docs/RELEASE.md — verify commands + DIST-05 audited-deps narrative + reproducibility posture** - `285050c` (docs)
2. **Task 2: docs/BENCHMARKS.md — methodology + raw head-to-head numbers + regression gate + re-bless** - `8ed855d` (docs)

**Plan metadata:** (this commit)

## Files Created/Modified
- `docs/RELEASE.md` - verify-signature/provenance/SBOM commands, DIST-05 dependency-tree narrative, reproducibility posture, local-reproduce recipe
- `docs/BENCHMARKS.md` - methodology, pinned real-repo table, honestly-TBD head-to-head table + regenerate command, real committed baseline number, regression gate + re-bless path

## Decisions Made
- Reported the DIST-05 dependency count precisely (134 total go.mod require entries: 27 direct, 107 indirect; 14 of 27 direct are tree-sitter grammar modules) rather than restating CONTEXT.md's less-precise "134 direct requires" phrasing verbatim — same overall narrative (grammar modules dominate the direct dependency list), more accurate arithmetic.
- Left the head-to-head numbers table as `TBD` with the exact regenerate command instead of citing the manual run mentioned in `08-07-SUMMARY.md`, whose output was never captured to a committed file — only its pass/fail outcome was recorded. Fabricating plausible numbers to fill the table would violate the plan's own prohibition against invented benchmark data.
- Documented only the `linux/arm64` leg as having an automated (non-blocking) double-build check today, matching what `ci.yml` (Plan 08-08) actually implements — `windows`/`darwin` best-effort reproducibility is stated as a posture/goal, not an already-implemented CI check, since no double-build step for those targets exists yet.

## Deviations from Plan

None — plan executed exactly as written. Both tasks' acceptance criteria (grep-based content checks) pass as specified.

## Issues Encountered

- No committed artifact exists with real per-repo head-to-head benchmark numbers (Plan 08-07's manual verification run during that session was never persisted to a file, per its own SUMMARY). This is not a defect in this plan — the plan itself anticipated this possibility ("if a run has not yet produced them" / "use clearly-marked placeholders") — but it means `docs/BENCHMARKS.md`'s raw-numbers table is not yet fully populated with real head-to-head data. Populating it requires either triggering `bench.yml` via `workflow_dispatch` on a real GitHub Actions run, or running `go run ./tools/bench/runner -mode headtohead` locally with the TS reference binary installed and transcribing the output — both documented in `docs/BENCHMARKS.md` itself.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

This is the final plan of the final phase (08-release-hardening-benchmarks) and the final phase of milestone v1.3. Remaining before the milestone can be considered fully proven in production (already flagged by prior plans in this phase, not new to this one):
- A real tagged `v*` release has never been pushed — `docs/RELEASE.md`'s verify commands are correct against the pipeline as built, but have not been exercised against a live release artifact (DIST-02/DIST-03 real-CI validation remains pending, consistent with `08-04-SUMMARY.md` and `STATE.md`).
- `bench.yml` has never had a live `workflow_dispatch`/scheduled run — `docs/BENCHMARKS.md`'s head-to-head table stays `TBD` until one completes and its output is transcribed back into this file.
- No blockers to closing out this plan or the phase; the deferred items above are pre-existing, already-documented, real-CI-validation gaps carried forward from Plans 08-04/08-07/08-08, not new gaps introduced here.

---
*Phase: 08-release-hardening-benchmarks*
*Completed: 2026-07-13*

## Self-Check: PASSED
- FOUND: docs/RELEASE.md
- FOUND: docs/BENCHMARKS.md
- FOUND: 285050c (Task 1 commit)
- FOUND: 8ed855d (Task 2 commit)
