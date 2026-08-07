---
phase: 03-2026-07-28-spec-compliance
plan: 05
subsystem: mcp
tags: [mcp, go-sdk, wire-oracle, sep-2575, instructions, legacy-compat, reviewed-diff]

# Dependency graph
requires:
  - phase: 03-2026-07-28-spec-compliance
    provides: "03-01's server/discover cacheScope fix and modern-discover-explore tracer; 03-02's advisory D-03 guard; 03-03's SPEC-02 _meta anchors; 03-04's live tool catalog and ExpectedScenarioCount at 27"
provides:
  - "SPEC-07 closed: ServerOptions.Instructions ships a compile-time-constant, interpolation-free usage-guidance string on every initialize AND every server/discover result"
  - "SPEC-06 closed: legacy-2024-11-05 (the oldest Legacy era) now proves a completed session AND a successful tool call, paired with handshake-explore's equivalent proof at 2025-11-25 (the newest era)"
  - "The phase's one reviewed-diff pass (D-06): 24 of 27 transcripts re-frozen, every changed line attributed to one of three named additive causes, zero unattributed lines, three transcripts confirmed byte-identical"
  - "Wire oracle re-proved non-vacuous against the Phase-3-complete binary via Mutation 1, with a byte-clean revert"
  - "COVERAGE-BASELINE.md rewritten to describe the 27-scenario corpus and its phase-by-phase growth"
affects: [phase-04-maintenance]

# Actuals (#2632)
actuals:
  tokens: 18036
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "A wire-contract string (SPEC-07's instructions) is a package-level compile-time const with a doc comment stating the interpolation prohibition explicitly, plus an acceptance-time grep and a corpus-wide byte-identity check (sort -u | wc -l == 1) as the two independent controls against T-03-19"
    - "D-06's reviewed-diff mechanism run a third time in this phase: fresh capture into a scratch directory (never over frozen files), full diff read, causes named with counts, checkpoint approval BEFORE any golden-file write"

key-files:
  created: []
  modified:
    - internal/mcp/server.go
    - test/wireoracle/scenarios.go
    - test/wireoracle/COVERAGE-BASELINE.md
    - testdata/wireoracle/transcripts/call-callees.golden
    - testdata/wireoracle/transcripts/call-callers.golden
    - testdata/wireoracle/transcripts/call-files.golden
    - testdata/wireoracle/transcripts/call-impact.golden
    - testdata/wireoracle/transcripts/call-node.golden
    - testdata/wireoracle/transcripts/call-search.golden
    - testdata/wireoracle/transcripts/call-status.golden
    - testdata/wireoracle/transcripts/error-confinement-reject.golden
    - testdata/wireoracle/transcripts/error-malformed-args.golden
    - testdata/wireoracle/transcripts/error-unknown-method.golden
    - testdata/wireoracle/transcripts/error-unknown-tool.golden
    - testdata/wireoracle/transcripts/handshake-explore.golden
    - testdata/wireoracle/transcripts/index-appears-mid-session.golden
    - testdata/wireoracle/transcripts/legacy-2024-11-05.golden
    - testdata/wireoracle/transcripts/legacy-2025-03-26.golden
    - testdata/wireoracle/transcripts/legacy-2025-06-18.golden
    - testdata/wireoracle/transcripts/legacy-2025-11-25.golden
    - testdata/wireoracle/transcripts/legacy-omitted-version.golden
    - testdata/wireoracle/transcripts/legacy-unsupported-2026-07-28.golden
    - testdata/wireoracle/transcripts/modern-discover-explore.golden
    - testdata/wireoracle/transcripts/toolslist-allowlist.golden
    - testdata/wireoracle/transcripts/toolslist-default.golden
    - testdata/wireoracle/transcripts/toolslist-no-index.golden
    - testdata/wireoracle/transcripts/toolslist-repeat.golden

key-decisions:
  - "The instructions string ships identically to BOTH the initialize result and the server/discover result via the single ServerOptions.Instructions field, deliberately: zero of the eight roster agent clients speak 2026-07-28 today, so a discover-only string would reach no real user."
  - "SPEC-06 covered by one added request (legacy-2024-11-05's trailing tools/call), not a new scenario. handshake-explore already proves the mechanism at 2025-11-25 (newest Legacy era); go-sdk's callTool/toolForErr never consult protocol version once initialize succeeds, so one success at each end of the Legacy range is judged sufficient for the three eras between — a LOW-confidence judgment call from 03-RESEARCH.md, recorded as such, not overstated as proof."
  - "Count correction recorded in the re-freeze commit, not treated as a semantic finding: the checkpoint text predicted 22 transcripts for cause 1; the true figure is 23 because 03-04 added index-appears-mid-session (a normal initialize-bearing scenario) after that text was drafted. Same mechanism, corrected arithmetic."
  - "Both deferred items (legacy-unsupported-2026-07-28's rename, the annotations hint correction) remain declined and recorded as non-goals in the plan — reconsidered and explicitly not taken, so the omissions read as decisions."
  - "No new test symbol for SPEC-06 — TestEveryRegisteredToolHasASuccessfulCallScenario and TestLegacyEraBaselineIsDocumented already cover it structurally; adding a bespoke test would duplicate assertions that already fire."

patterns-established: []

requirements-completed: [SPEC-06, SPEC-07]

coverage:
  - id: D1
    description: "ServerOptions.Instructions set from a newline-free, interpolation-free, ~511-byte compile-time constant covering all four required content points (what codegraph indexes and that codegraph_explore is the first tool to try; path defaults to server cwd; empty tool list means run codegraph init, not broken; tools appear without client restart)"
    requirement: "SPEC-07"
    verification:
      - kind: integration
        ref: "test/wireoracle TestFrozenTranscriptsMatch/modern-discover-explore (instructions in the discover result)"
        status: pass
      - kind: unit
        ref: "go test ./internal/mcp/... -count=1"
        status: pass
    human_judgment: false
  - id: D2
    description: "The identical instructions string also appears on every classic initialize result (23 transcripts), reaching every one of the eight roster agent clients that speak Legacy protocol revisions today"
    requirement: "SPEC-07"
    verification:
      - kind: integration
        ref: "test/wireoracle TestFrozenTranscriptsMatch (24/27 transcripts changed, 23 carrying the instructions field on their initialize line)"
        status: pass
    human_judgment: false
  - id: D3
    description: "T-03-19 mitigated: one unique instructions value across every transcript that carries it (byte-identity check), zero host-path leakage across the frozen corpus"
    requirement: "SPEC-07"
    verification:
      - kind: other
        ref: "rg -o '\"instructions\":\"[^\"]*\"' testdata/wireoracle/transcripts/*.golden | sort -u | wc -l -> 1; rg -c '/Users/|/home/|/tmp/|/var/folders' testdata/wireoracle/transcripts/*.golden | rg -v ':0$' -> empty"
        status: pass
    human_judgment: false
  - id: D4
    description: "legacy-2024-11-05 (the oldest Legacy era) proves a completed session AND a successful codegraph_explore tools/call, not merely negotiation"
    requirement: "SPEC-06"
    verification:
      - kind: integration
        ref: "test/wireoracle TestFrozenTranscriptsMatch/legacy-2024-11-05; TestEveryRegisteredToolHasASuccessfulCallScenario; TestLegacyEraBaselineIsDocumented"
        status: pass
    human_judgment: false
  - id: D5
    description: "ExpectedScenarioCount unchanged at 27 — no scenario added or removed by this plan"
    verification:
      - kind: unit
        ref: "test/wireoracle TestScenarioCountIsExact"
        status: pass
    human_judgment: false
  - id: D6
    description: "The phase's one reviewed-diff pass: full 27-scenario diff produced from a fresh capture into a scratch directory (never over frozen files), every changed line attributed to one of three named causes with per-cause transcript counts, zero unattributed lines, checkpoint approval obtained before any golden-file write"
    verification: []
    human_judgment: true
    rationale: "This is the class of judgment call D-06's human-diff-read checkpoint exists for — a comparator cannot distinguish cosmetic-additive movement from a silent regression. The maintainer reviewed the full attribution (grouped by cause, with counts, must-not-have-changed properties, and anchor confirmation) at the Task 2 checkpoint and approved it explicitly; recorded here as already adjudicated."
  - id: D7
    description: "Re-freeze commit's changed-file set equals exactly the approved cause list's 24 transcripts, no others"
    verification:
      - kind: other
        ref: "git diff --name-status 7c5d074~1 7c5d074 -- testdata/wireoracle/transcripts/ (exactly 24 files)"
        status: pass
    human_judgment: false
  - id: D8
    description: "Wire oracle demonstrated RED against a confirmed-applied mutation (Phase 1's Mutation 1) on the Phase-3-complete binary — both TestFrozenTranscriptsMatch and TestSpecAnchorsHold's framing invariant, 27/27 each — then green after a byte-clean revert"
    verification:
      - kind: integration
        ref: "go test ./test/wireoracle/... -count=1 (mutation applied: 27/27 TestFrozenTranscriptsMatch fail + 27/27 TestSpecAnchorsHold framing fail; reverted: ok, git status --porcelain internal/mcp/server.go empty)"
        status: pass
    human_judgment: false
  - id: D9
    description: "Both hand-authored spec anchors (-32601, -32602) held; 03-01's assertDiscoverCacheControl held; 03-03's -32022 anchor held — none moved as a side effect of this plan's changes"
    verification:
      - kind: integration
        ref: "go test ./test/wireoracle/... -run TestSpecAnchorsHold -v (all pass, captured fresh against the Task-1 binary independent of frozen bytes)"
        status: pass
    human_judgment: false
  - id: D10
    description: "COVERAGE-BASELINE.md describes 27 scenarios (header and Total both say 27); remaining references to 23 are historically accurate (the Phase 1 mark3labs-era baseline this corpus grew from)"
    verification:
      - kind: other
        ref: "rg -n 'Scenario count|## Total' test/wireoracle/COVERAGE-BASELINE.md"
        status: pass
    human_judgment: false
  - id: D11
    description: "go test ./... -count=1 passes except the documented pre-existing internal/daemon flake (issue #17 / MAINT-02, Phase 4), confirmed to pass in isolation and confirmed untouched by this plan's diff"
    verification:
      - kind: other
        ref: "go test ./internal/daemon/... -count=1 -> ok; git diff --stat 21e9fcb HEAD -- internal/daemon/ -> empty"
        status: pass
    human_judgment: false

duration: ~40min (spanning a human checkpoint pause between Task 1 and Task 3)
completed: 2026-08-06
status: complete
---

# Phase 3 Plan 5: Instructions String, Legacy Tool-Call Coverage, and the Phase's Reviewed Re-freeze Summary

**SPEC-07's usage-guidance string (compile-time constant, no interpolation, reaching both `initialize` and `server/discover` via the same SDK field) and SPEC-06's oldest-era tool-call proof landed together, then moved through the phase's one reviewed-diff pass — 24 of 27 transcripts re-frozen, every line attributed to one of three named additive causes, and the oracle re-proved able to fail against the Phase-3 binary.**

## Performance

- **Duration:** ~40 min total span (commit-to-commit), including a human checkpoint pause between Task 1's mechanical attribution and Task 3's re-freeze
- **Started:** 2026-08-06T12:10:26-04:00 (first task commit)
- **Completed:** 2026-08-06T12:27:50-04:00 (final task commit)
- **Tasks:** 3 (Task 1 auto, Task 2 checkpoint:human-verify blocking, Task 3 auto)
- **Files modified:** 27 (2 code/scenario files, 1 doc file, 24 re-frozen transcripts)

## Accomplishments

- Added a package-level `instructions` constant to `internal/mcp/server.go` — a single-paragraph, newline-free, ~511-byte compile-time string literal with no interpolation of any kind (no `Sprintf`, no concatenation, no `repoPath`/`startPath`/`os.Getenv` reference) — covering all four content requirements: what codegraph indexes and that `codegraph_explore` is the first tool to try for a "where is X"/"how does Y work" question; that every tool's `path` argument defaults to the server's own working directory; that an empty tool list means "run `codegraph init`," not "the server is broken"; and that tools appear without a client restart (true because of 03-04's live catalog).
- Wired it into `BuildServer`'s existing `&mcp.ServerOptions{...}` literal via `Instructions: instructions`, reaching both the `initialize` result and the `server/discover` result through the identical SDK field — deliberate, since zero of the eight roster agent clients speak `2026-07-28` today and a discover-only string would reach no real user.
- Appended a third request (`toolCallRequest(3, "codegraph_explore", ...)`) to `legacy-2024-11-05` in `test/wireoracle/scenarios.go`, proving the OLDEST Legacy era completes a session AND successfully calls a tool — paired with `handshake-explore`'s existing proof at `2025-11-25`, the NEWEST era. Left `ExpectedScenarioCount` at 27 (no scenario added or removed) and left every other era scenario's name, `EraOfferedVersion`, `EraNegotiatedVersion`, and `ExpectTools` untouched.
- Ran the phase's one reviewed-diff pass (D-06): built a fresh binary, captured all 27 scenarios into a scratch directory via the oracle's own capture CLI, diffed against the frozen corpus, and read every line. Attributed all 173 changed lines across 24 files to three named additive causes (23 transcripts gaining `instructions` on `initialize`, 1 gaining it on `server/discover`, 1 gaining the SPEC-06 `tools/call`), confirmed three transcripts byte-identical rather than skipped, confirmed all four spec anchors held, and confirmed five must-not-have-changed properties (protocolVersion, `toolslist-no-index`'s capabilities object, tool-array membership, `cacheScope`, error codes) with observed values. Presented at the Task 2 checkpoint and approved by the maintainer.
- Re-froze exactly the 24 approved transcripts from a fresh capture (never hand-written), leaving the other 3 untouched, and committed with the full cause enumeration — including the count correction (predicted 22, true 23, traced to 03-04's `index-appears-mid-session` landing after the checkpoint text was drafted) — as D-06's entire mechanism (no ledger file).
- Re-proved the oracle's non-vacuity on the Phase-3-complete binary: re-applied Phase 1's Mutation 1 (a stray non-JSON stdout line), confirmed the mutation actually landed via `git diff` before trusting the RED, rebuilt, observed both `TestFrozenTranscriptsMatch` and `TestSpecAnchorsHold`'s framing invariant go RED across all 27 scenarios, reverted, confirmed a byte-clean `git status --porcelain`/`git diff` on `internal/mcp/server.go`, rebuilt, confirmed green.
- Rewrote `test/wireoracle/COVERAGE-BASELINE.md` to describe the current 27-scenario corpus (adding category sections for 03-01/03-03/03-04's four scenarios and documenting `legacy-2024-11-05`'s new request), with a History section recording how the corpus grew phase-by-phase so a later reader does not need to reconstruct it from `git log`.

## Task Commits

Each task was committed atomically:

1. **Task 1: The instructions string, and the oldest era's tool call** - `caaeea5` (feat)
2. **Task 2: The diff review — this phase's actual acceptance mechanism (D-06)** - checkpoint, no commit (read-only diff review against a scratch capture; full attribution presented at the checkpoint and approved by the maintainer)
3. **Task 3: Re-freeze through the reviewed pass, and re-prove the oracle can fail** - `7c5d074` (test) + `07edb16` (docs, COVERAGE-BASELINE.md)

_No plan-metadata commit yet — this SUMMARY plus STATE.md/ROADMAP.md/REQUIREMENTS.md updates land in the final `docs(03-05)` commit below._

## Files Created/Modified

- `internal/mcp/server.go` - added the `instructions` package-level constant and wired it into `ServerOptions.Instructions`
- `test/wireoracle/scenarios.go` - appended `legacy-2024-11-05`'s trailing `tools/call` request and its doc comment recording the SPEC-06 judgment call
- `test/wireoracle/COVERAGE-BASELINE.md` - rewritten for the 27-scenario corpus with a phase-by-phase History section
- 24× `testdata/wireoracle/transcripts/*.golden` - re-frozen from a fresh capture via the oracle's own capture CLI, no hand edits (see key-files above for the full list)

## Decisions Made

See `key-decisions` in frontmatter for the verbatim record of: the dual initialize/discover instructions mechanism and why; SPEC-06's one-added-request approach and its LOW-confidence sufficiency judgment; the count-correction note (22 predicted vs. 23 observed, traced to 03-04); the two deferred items remaining declined; and why no new test symbol was added for SPEC-06.

## Deviations from Plan

None requiring a stop or architectural discussion. One correction worth recording:

**Count correction, not a Rule 1-4 deviation:** The plan's Task 2 checkpoint text predicted "22 of the pre-existing transcripts" for cause 1. The full attribution (Task 2) found 23. Traced the discrepancy: only `edge-call-before-initialize` and the 2 `modern-meta-*` scenarios lack `initialize` today (not "the three `modern-*` scenarios" as the plan's prose read — `modern-discover-explore` also lacks `initialize` but is separately covered under cause 2). The predicted count also predates `index-appears-mid-session` (03-04), which does carry a normal `initialize` and was the one transcript accounting for the difference. Same mechanism, corrected arithmetic — escalated at the checkpoint per T-03-23's "anything else" instruction rather than silently absorbed, and the maintainer's approval message explicitly directed recording it as a note, not a cause.

## Issues Encountered

- The `internal/daemon` full-suite flake (`TestSoak`, timing-sensitive) reproduced once during this plan's final `go test ./... -count=1` run. Confirmed pre-existing (issue #17 / MAINT-02, Phase 4), confirmed to pass in isolation (`go test ./internal/daemon/... -count=1` -> ok), and confirmed untouched by this plan's diff (`git diff --stat 21e9fcb HEAD -- internal/daemon/` -> empty). Not investigated or fixed, per the standing scope-boundary rule and this plan's explicit instruction.
- `TRANSCRIPT_FREEZE_BASE=HEAD~2 task check:transcript-freeze` was not run as part of this plan's verification — it is not named in Task 3's acceptance criteria, and the plan's own critical-facts note states the D-03 guard's advisory report (expected to fire on this commit's transcript diff) is neither a failure to investigate nor something to silence. `go test ./tools/transcriptfreeze/... -count=1` (part of the full-suite run) confirms the guard itself is unmodified and still passing its own test suite.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- SPEC-06 and SPEC-07 are closed. SPEC-01/02/03/04/05/08 were closed by 03-01/03-03/03-04 (03-02 built no requirement-closing surface, per its own SUMMARY). All eight of Phase 3's requirements are now complete.
- `ExpectedScenarioCount` is 27, unchanged by this plan; `Scenarios()`/`testdata/wireoracle/transcripts/` agree.
- The wire-oracle corpus's one-reviewed-diff-pass mechanism (D-06) has now run three times across this phase (03-01, 03-03/03-04 individually, and this plan's consolidated pass) plus once in Phase 2 (02-05) — four demonstrated instances of the same mechanism, each catching or confirming no unattributed movement.
- `COVERAGE-BASELINE.md` now documents an explicit "Instruction for whoever next extends this corpus" section, generalizing the four-step protocol (bump the count, capture fresh, one reviewed-diff pass, update the doc) for Phase 4 or any later phase that touches this corpus.
- No blockers. `internal/mcp/archtest`'s VRFY-02 guard was not triggered — no protocol-version literal or SDK constant reference was introduced by this plan.

---
*Phase: 03-2026-07-28-spec-compliance*
*Completed: 2026-08-06*

## Self-Check: PASSED

All claimed files verified present: `internal/mcp/server.go`, `test/wireoracle/scenarios.go`, `test/wireoracle/COVERAGE-BASELINE.md`, `testdata/wireoracle/transcripts/legacy-2024-11-05.golden`, `testdata/wireoracle/transcripts/modern-discover-explore.golden`, this SUMMARY. All three commit hashes (`caaeea5`, `7c5d074`, `07edb16`) confirmed present in `git log --oneline --all`.
