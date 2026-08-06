---
phase: 01-protocol-scoping-the-sdk-independent-wire-oracle
plan: 07
subsystem: testing
tags: [mcp, jsonrpc, stdio, wire-protocol, golden-transcript, mark3labs-mcp-go, mutation-testing, non-vacuity]

# Dependency graph
requires:
  - phase: 01-04
    provides: "test/wireoracle's 17-scenario suite, anchors.go's Anchor infrastructure, TestSpecAnchorsHold/TestToolsListExactSets/TestEveryRegisteredToolHasASuccessfulCallScenario"
  - phase: 01-05
    provides: "The six-era Legacy handshake baseline bringing the suite to exactly 23 scenarios, TestLegacyEraBaselineIsDocumented"
provides:
  - "ExpectedScenarioCount = 23 (test/wireoracle/scenarios.go), the exact-equality non-shrinkage guard TestScenarioCountIsExact enforces"
  - "TestTranscriptSetMatchesScenarioSet — two-way set equality between scenario names and testdata/wireoracle/transcripts/*.golden basenames"
  - "TestEmptyTranscriptNeverMatches plus the statFrozenTranscript/compareBytesLineByLine pure-function split that lets a 'prove this guard fires' demonstration run without propagating into go test's own pass/fail exit code"
  - "test/wireoracle/normalize_test.go: TestNormalizationRulesMatchOnlyTheirOwnField (positive + two negatives per rule), TestRuleTestCoverageScalesWithRules, TestNormalizeIsIdentityWhenNothingMatches, TestNormalizeHelpersFailLoudly"
  - "TestEveryDeclaredFiringRuleActuallyFires — generalized from plan 01's single-scenario ledger check to a real capture across all 23 scenarios"
  - "test/wireoracle/MUTATION-PROOF.md — the one-time D-07 mutation matrix run against the real binary, all four mutations confirmed applied, observed red, and reverted"
  - "test/wireoracle/COVERAGE-BASELINE.md — the dated, human-readable index of the complete frozen 23-scenario pre-migration baseline, and the instruction for Phase 2's first plan to depend on it"
affects: [02-sdk-migration]

# Actuals (#2632)
actuals:
  tokens: 13223
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Pure-logic-core plus *testing.T-failing-wrapper split (statFrozenTranscript/readFrozenTranscript, compareBytesLineByLine/assertBytesEqualLineByLine): lets a test assert that a guard's underlying check FAILS on a deliberately-bad input without a nested t.Run's expected failure propagating into the parent test's (and therefore go test's own) pass/fail exit code — Go's testing package always marks ancestor tests failed when a child t.Run body calls t.Fatalf, regardless of what the caller does with t.Run's returned bool."
    - "A mutation whose trigger request is not exercised by any frozen scenario is driven directly through Capture from a temporary, never-committed _test.go file rather than added to Scenarios() — keeps ExpectedScenarioCount and the transcript-set equality guard undisturbed by a probe that exists only to prove a mutation's blast radius, and the probe's own assertion serves as that mutation's gate."

key-files:
  created:
    - test/wireoracle/normalize_test.go
    - test/wireoracle/MUTATION-PROOF.md
    - test/wireoracle/COVERAGE-BASELINE.md
  modified:
    - test/wireoracle/scenarios.go
    - test/wireoracle/oracle_test.go

key-decisions:
  - "TestNormalizeRuleLedgerIsHonest (plan 01's single-scenario ledger-honesty test) was renamed to TestEveryDeclaredFiringRuleActuallyFires and broadened to accumulate the per-rule ledger across a real capture of all 23 scenarios in Scenarios(), rather than adding a second, near-duplicate test alongside it — the plan's critical_context explicitly named TestEveryDeclaredFiringRuleActuallyFires as this plan's own deliverable, and the existing test already implemented the same property for one scenario, so broadening in place was the non-duplicative choice."
  - "Mutation 3's trigger request (codegraph_explore tools/call with omitted query) is not exercised by any of the 23 frozen scenarios — driven via a temporary, never-committed probe test (test/wireoracle/mutation3_probe_test.go, deleted after evidence capture) per the plan's own offered alternative, rather than adding a temporary scenario to Scenarios(). The measured result — zero of 23 frozen transcripts changed — is recorded in MUTATION-PROOF.md as an open gap / input to Phase 2's SDK-04 audit, not papered over."
  - "Mutation 4's blast radius on TestFrozenTranscriptsMatch (16 of 23 subtests also fail, via the embedded assertSessionLine/assertProtocolVersionAnchor calls that share internalmcp.ProtocolVersion with the isolated TestSpecAnchorsHold anchor) is recorded precisely rather than glossed over: the failure messages are all session-line/anchor mismatches, never a byte-diff message, which is itself the evidence the frozen transcript bytes stayed unchanged — the code-level form of the D-02 asymmetry the plan asked to demonstrate."

patterns-established:
  - "normalizationRuleCases() (normalize_test.go) is the single source of truth for both the per-rule field-anchoring test and its own coverage-scaling guard — TestRuleTestCoverageScalesWithRules derives the exercised rule-name set from the same table TestNormalizationRulesMatchOnlyTheirOwnField iterates, so a rule added to Rules without matching cases fails loudly rather than silently passing an unrelated count check."

requirements-completed: [VRFY-01, VRFY-04]

coverage:
  - id: D1
    description: "Three permanent structural non-vacuity guards (exact scenario count, two-way transcript-set equality, empty-transcript-never-matches) plus per-rule normalization self-defeat cases, each observed red against a confirmed-applied mutation before being trusted"
    requirement: "VRFY-04"
    verification:
      - kind: unit
        ref: "test/wireoracle/oracle_test.go#TestScenarioCountIsExact"
        status: pass
      - kind: unit
        ref: "test/wireoracle/oracle_test.go#TestTranscriptSetMatchesScenarioSet"
        status: pass
      - kind: unit
        ref: "test/wireoracle/oracle_test.go#TestEmptyTranscriptNeverMatches"
        status: pass
      - kind: unit
        ref: "test/wireoracle/normalize_test.go#TestNormalizationRulesMatchOnlyTheirOwnField"
        status: pass
      - kind: unit
        ref: "test/wireoracle/normalize_test.go#TestRuleTestCoverageScalesWithRules"
        status: pass
      - kind: unit
        ref: "test/wireoracle/oracle_test.go#TestEveryDeclaredFiringRuleActuallyFires"
        status: pass
    human_judgment: false
  - id: D2
    description: "One-time D-07 mutation matrix (stray stdout line, dropped tool, changed error shape, changed protocol-version literal) run against the real binary, each confirmed applied, observed red via task test:wireoracle, and reverted to a clean tree"
    requirement: "VRFY-01"
    verification:
      - kind: manual_procedural
        ref: "test/wireoracle/MUTATION-PROOF.md"
        status: pass
    human_judgment: true
    rationale: "The mutation matrix is a one-time, manually-applied-and-reverted procedure captured as a dated document with verbatim terminal output — its accuracy and completeness (were all four mutations genuinely applied and reverted, are the recorded failure messages real) is not itself re-run by any future `go test` invocation, so a human should confirm the document reads as an honest, non-vacuous account rather than trusting a status field."

# Metrics
duration: ~100min
completed: 2026-08-05
status: complete
---

# Phase 01 Plan 07: D-07 Non-Vacuity Proof — Permanent Guards and the Real-Binary Mutation Matrix Summary

**Three permanent structural guards (exact scenario count, two-way transcript-set equality, empty-transcript-never-matches) plus per-rule normalization self-defeat tests, backed by a one-time four-mutation matrix run against the real binary — every guard observed red against a confirmed-applied mutation before being trusted, with mutation 3 honestly recording zero blast radius on the frozen 23-scenario suite as an open gap for Phase 2's SDK-04 audit.**

## Performance

- **Duration:** ~100 min
- **Completed:** 2026-08-05T21:39:12Z
- **Tasks:** 2
- **Files modified:** 2 modified (scenarios.go, oracle_test.go), 3 created (normalize_test.go, MUTATION-PROOF.md, COVERAGE-BASELINE.md)

## Accomplishments

- `ExpectedScenarioCount = 23` declared as a literal constant beside `Scenarios()` in `scenarios.go`; `TestScenarioCountIsExact` compares with exact equality.
- `TestTranscriptSetMatchesScenarioSet` reads `testdata/wireoracle/transcripts/` with `os.ReadDir` and asserts two-way set equality against scenario names — a renamed transcript names both the missing and the orphaned entry.
- `TestEmptyTranscriptNeverMatches` proves three empty-transcript edges never compare equal, backed by a new `statFrozenTranscript`/`compareBytesLineByLine` pure-function split so the "prove this guard fires" demonstration doesn't propagate a nested `t.Run` failure into `go test`'s own exit code.
- `test/wireoracle/normalize_test.go` (new): one positive + two negative cases per rule in `Rules` (`TestNormalizationRulesMatchOnlyTheirOwnField`), with `TestRuleTestCoverageScalesWithRules` tying the exercised rule-name set to `len(Rules)`; plus `TestNormalizeIsIdentityWhenNothingMatches` and `TestNormalizeHelpersFailLoudly`.
- `TestEveryDeclaredFiringRuleActuallyFires`: generalized plan 01's single-scenario `TestNormalizeRuleLedgerIsHonest` to accumulate the per-rule ledger across a real capture of all 23 scenarios.
- All four structural guards demonstrated red against a confirmed-applied mutation (dropped scenario, renamed transcript, emptied transcript, flipped `ExpectFires`), then reverted — verbatim failures recorded below.
- `test/wireoracle/MUTATION-PROOF.md`: the one-time D-07 mutation matrix run against the real, unmodified binary — a stray stdout line, a dropped tool registration, `exploreHandler`'s missing-query error-shape change, and a changed `ProtocolVersion` literal — each confirmed applied, observed red independently via `task test:wireoracle`, and reverted to a clean tree.
- `test/wireoracle/COVERAGE-BASELINE.md`: dated, records the exact 23-scenario set grouped by coverage category, and the instruction for Phase 2's first plan to depend on this file and its four enforcing tests.
- Full-suite wall-clock measured at ~16.9s (`go test`) / ~17.7s (`task` CLI overhead included), well within the 3-minute budget CONTEXT left open.

## Task 1 Guard Red-Proofs (verbatim failure messages)

**1. `TestScenarioCountIsExact`, `call-status` scenario temporarily commented out:**
```
oracle_test.go:164: len(Scenarios()) = 22, want exactly 23 (ExpectedScenarioCount) — either a scenario silently disappeared or one was added without updating the constant beside Scenarios()
```

**2. `TestTranscriptSetMatchesScenarioSet`, `call-status.golden` temporarily renamed to `call-status-RENAMED.golden`:**
```
oracle_test.go:215: scenario/transcript set mismatch: missing transcripts (scenario has no .golden file) = [call-status]; orphaned transcripts (.golden file has no scenario) = [call-status-RENAMED]
```

**3. `readFrozenTranscript`'s zero-byte guard (`TestFrozenTranscriptsMatch/call-status`), `call-status.golden` temporarily truncated to zero bytes:**
```
oracle_test.go:124: frozen transcript at ../../testdata/wireoracle/transcripts/call-status.golden is zero bytes — an empty frozen transcript can never be a valid comparison target (D-07)
```
Confirmed this also fails the full suite: `go test ./test/wireoracle/... -count=1` → `FAIL ... TestFrozenTranscriptsMatch/call-status`, not a plain byte-diff.

**4. `TestEveryDeclaredFiringRuleActuallyFires`, `repoDir` rule's `ExpectFires` flipped `false` → `true`:**
```
oracle_test.go:482: rule "repoDir": ExpectFires=true but the accumulated ledger recorded 0 hits across all 23 scenarios — rule stopped firing
```

All four breaks were confirmed applied (file inspected/diffed) before trusting the red, then reverted; `git diff --stat` showed zero lines changed after each revert.

## Task 2 Mutation Matrix (summary — full verbatim output and analysis in MUTATION-PROOF.md)

| # | Mutation | Gate that went red | Blast radius |
|---|---|---|---|
| 1 | Stray non-JSON stdout line in `BuildServer` | `TestFrozenTranscriptsMatch` (all 23) + `TestSpecAnchorsHold`'s framing invariant (all 23); corroborated independently by `internal/graphstore/archtest`'s static `TestNoStdoutNoiseInServeReachablePackages` | All 23 scenarios |
| 2 | Dropped `codegraph_status` tool registration | `TestFrozenTranscriptsMatch` (`toolslist-allowlist`, `toolslist-repeat`, `call-status`, 6 session-line-tools-count mismatches), `TestToolsListExactSets`, `TestEveryRegisteredToolHasASuccessfulCallScenario` | 9 scenarios + 2 structural assertions |
| 3 | `exploreHandler`'s missing-query branch: `return mcp.NewToolResultError(...), nil` → `return nil, err` | A temporary, never-committed probe test (`mutation3_probe_test.go`), driven via `Capture` outside `Scenarios()` | **Zero of 23 frozen transcripts** — recorded as an honest open gap / SDK-04 input, not papered over |
| 4 | `internal/mcp.ProtocolVersion`: `"2025-11-25"` → `"9999-99-99"` | `TestSpecAnchorsHold`'s protocolVersion anchor (isolated, non-circular); `TestFrozenTranscriptsMatch`'s embedded session-line/anchor checks for 16 non-era scenarios (never a byte-diff — proving the wire bytes stayed unchanged, the D-02 asymmetry) | Anchor-only; frozen transcript bytes provably unchanged |

Every mutation was confirmed applied via `git diff` before its failure was trusted, and reverted to a byte-identical tree (`git diff --stat` showed zero lines changed) before the next mutation began.

## Task Commits

1. **Task 1: Permanent in-suite guards that attack the harness** — `f0bbbf8` (test)
2. **Task 2: One-time mutation matrix against the real binary, demonstrated and reverted** — `1659c71` (docs)

_Note: no plan-metadata commit is included in this list — per the objective, this executor does not update STATE.md/ROADMAP.md; the orchestrator owns those writes and the metadata commit that follows._

## Files Created/Modified

- `test/wireoracle/scenarios.go` — `ExpectedScenarioCount = 23` constant declared beside `Scenarios()`
- `test/wireoracle/oracle_test.go` — `readFrozenTranscript`/`statFrozenTranscript`, `compareBytesLineByLine`/`assertBytesEqualLineByLine` pure-logic split, `TestScenarioCountIsExact`, `TestTranscriptSetMatchesScenarioSet`, `TestEmptyTranscriptNeverMatches`, `TestEveryDeclaredFiringRuleActuallyFires` (renamed and broadened from `TestNormalizeRuleLedgerIsHonest`)
- `test/wireoracle/normalize_test.go` (new) — `normalizationRuleCases()`, `TestNormalizationRulesMatchOnlyTheirOwnField`, `TestRuleTestCoverageScalesWithRules`, `TestNormalizeIsIdentityWhenNothingMatches`, `TestNormalizeHelpersFailLoudly`
- `test/wireoracle/MUTATION-PROOF.md` (new) — the four-mutation matrix, dated 2026-08-05, with verbatim failure output and revert confirmations
- `test/wireoracle/COVERAGE-BASELINE.md` (new) — dated, complete 23-scenario index grouped by coverage category

## Decisions Made

See `key-decisions` in frontmatter for full rationale. Summary:
- `TestEveryDeclaredFiringRuleActuallyFires` supersedes plan 01's `TestNormalizeRuleLedgerIsHonest` in place (renamed + broadened to all 23 scenarios) rather than duplicating it.
- Mutation 3's trigger request is driven via a temporary, never-committed probe test rather than a temporary `Scenarios()` entry, per the plan's own offered alternative.
- Mutation 3's zero blast radius on the frozen suite, and mutation 4's dual failure mode inside `TestFrozenTranscriptsMatch`, are both recorded precisely in `MUTATION-PROOF.md` rather than simplified or omitted.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `TestEmptyTranscriptNeverMatches`'s first implementation made `go test ./test/wireoracle/...` exit non-zero**
- **Found during:** Task 1, first verification run of the new guards
- **Issue:** The initial implementation used nested `t.Run("comparison", func(t *testing.T) { assertBytesEqualLineByLine(...) })` and checked the returned `ok` bool to confirm the comparison failed as expected. This is structurally broken: Go's testing package always propagates a child `t.Run`'s failure to mark every ancestor test (and therefore the whole package) failed, regardless of what the caller does with the returned bool — so a subtest deliberately constructed to fail (proving the guard fires) made the overall suite report `FAIL`, violating the acceptance criterion that `go test ./test/wireoracle/... -count=1` exits 0.
- **Fix:** Split `assertBytesEqualLineByLine` into a pure `compareBytesLineByLine(got, want []byte) error` core plus a thin `*testing.T`-failing wrapper, and did the same for `readFrozenTranscript` → `statFrozenTranscript(goldenPath string) ([]byte, error)`. `TestEmptyTranscriptNeverMatches` now calls the pure functions directly and asserts on the returned `error`, never going through a nested `t.Run` whose body itself fails.
- **Files modified:** `test/wireoracle/oracle_test.go`
- **Verification:** `go test ./test/wireoracle/... -count=1 -v` exits 0, with `TestEmptyTranscriptNeverMatches`'s three subtests all `PASS` (each logging the expected internal failure message via `t.Logf`, not propagating it).
- **Committed in:** `f0bbbf8` (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (Rule 1 — a structural bug in the guard's own test-design, caught on first run, before any commit)
**Impact on plan:** Necessary for the plan's own acceptance criterion (`go test ./test/wireoracle/... -count=1` exits 0) to hold at all. No scope creep — the fix only restructured how two existing helper functions are called; the guards' semantics and failure messages are unchanged.

## Issues Encountered

**Pre-existing, unrelated flake: the VRFY-03 stderr session line is occasionally absent under repeated back-to-back test-binary runs.** Observed intermittently (roughly 1 in 3-4 runs) as `stderr must contain exactly one "codegraph: mcp-session" line, found 0: []` for an arbitrary scenario, both on the clean pre-mutation tree and after several mutation reverts — always resolved by re-running once, and always with `git diff --stat` confirming the source was byte-identical to the last known-clean state at the time. This is the same class of concurrency edge plan 04's SUMMARY documented for mark3labs v0.56.0's stdio transport (tools/call dispatched asynchronously). It is out of scope for this plan (Scope Boundary — not caused by this task's changes, not part of any acceptance criterion here, and the plan's own `<verify>` blocks use `-count=1` single runs, which were always confirmed green before proceeding). Not fixed; noted here for visibility.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The wire oracle now carries all three D-07 structural non-vacuity guards, the per-rule normalization self-defeat suite, and a dated, confirmed-applied mutation matrix — the phase's non-vacuity story (D-07) is complete.
- `test/wireoracle/COVERAGE-BASELINE.md` and the four tests it names (`TestScenarioCountIsExact`, `TestTranscriptSetMatchesScenarioSet`, `TestEveryRegisteredToolHasASuccessfulCallScenario`, `TestLegacyEraBaselineIsDocumented`) are ready for Phase 2's first plan to declare an explicit dependency on, per the instruction recorded in both `COVERAGE-BASELINE.md` and the phase plan.
- **Open finding for Phase 2's SDK-04 audit:** mutation 3 (`exploreHandler`'s missing-query error shape) has zero blast radius on the frozen 23-scenario suite — no frozen scenario exercises any handler's own required-argument-validation failure path. This is not fixed here (extending the frozen set is out of scope; `ExpectedScenarioCount` and the transcript-set guard are locked at 23) — Phase 2's SDK-04 audit should treat this as a known, measured gap.
- No blockers. `go.mod`'s MCP dependency line is unchanged (per this plan's threat register, T-07-SC).
- This SUMMARY does not update `.planning/STATE.md` or `.planning/ROADMAP.md` — per the objective, that is the orchestrator's responsibility after this plan's completion is reported.

---
*Phase: 01-protocol-scoping-the-sdk-independent-wire-oracle*
*Completed: 2026-08-05*

## Self-Check: PASSED

All files created/modified this plan verified present on disk (`test/wireoracle/normalize_test.go`,
`test/wireoracle/MUTATION-PROOF.md`, `test/wireoracle/COVERAGE-BASELINE.md`,
`test/wireoracle/scenarios.go`, `test/wireoracle/oracle_test.go`); commits `f0bbbf8` and `1659c71`
verified present in `git log --oneline --all`.
