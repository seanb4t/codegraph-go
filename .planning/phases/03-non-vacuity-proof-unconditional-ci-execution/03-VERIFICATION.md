---
phase: 03-non-vacuity-proof-unconditional-ci-execution
verified: 2026-08-15T14:55:25Z
status: human_needed
score: 4/4 must-haves verified
behavior_unverified: 0
overrides_applied: 0
human_verification:
  - test: "Read 03-MUTATION-LOG.md's five family entries ((a) behavioral properties, (b) golden byte-identity, (c) CLI==MCP trio, (d) hermetic resolution, (e) coverage guard) and confirm each pasted RED failure is a genuine observation of the family going red under its defined mutation — mutation applied -> observed failure -> byte-clean revert."
    expected: "Each family row shows a real failing `go test` output whose message names the mutated assertion (e.g. the defs-count boundary, the 25/26 golden count, the 'CLI and MCP output diverge' divergence, the lockedCorpusDir stat failure, the CheckCoverage below-threshold report) followed by a revert command and a green re-run proof."
    why_human: "Automation cannot re-derive a past observation without re-applying the mutations, which would mutate the working tree (prohibited). The record is complete and internally consistent with the code (line numbers, message strings and threshold values match exactly), but the genuineness of each pasted failure is a human judgment. The plan's own coverage metadata declares human_judgment: true for FIXT-07 with this same rationale."
---

# Phase 3: Non-Vacuity Proof & Unconditional CI Execution — Verification Report

**Phase Goal:** The re-baselined golden suite is trusted because it has been watched fail (FIXT-07), and CI cannot silently stop running it (FIXT-03).
**Verified:** 2026-08-15T14:55:25Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| #   | Truth (roadmap success criterion) | Status | Evidence |
| --- | --------------------------------- | ------ | -------- |
| 1   | FIXT-07: Each assertion family demonstrated RED against a confirmed-applied, byte-cleanly-reverted mutation, recorded per family | ✓ VERIFIED (record complete + consistent; see Human Verification) | `03-MUTATION-LOG.md` records all five families with mutation applied, pasted failing output, revert command, and byte-clean proof; suite green afterward; no residue in `git diff` |
| 2   | FIXT-03: A CI run reports the number of golden scenarios executed and asserts that count against an expected value | ✓ VERIFIED | `TestGoldenScenarioCountIsExact` logs `goldens: 26/26, CASES cases: 4/4, total: 30/30` — exact equality, zero-guards on both legs, executed-keyed counters in `TestReFrozenGoldensValid` (`verified`) and `TestCorpusBehaviorSynthetic` (`executedCases`) |
| 3   | FIXT-03: A corpus fetch/cache failure fails the CI job loudly; no golden test can self-skip in CI | ✓ VERIFIED | `corpora.yml` `golden` job runs `task corpora:fetch` → `task corpora:assert` → `task test:golden` unconditionally (not gated on cache-hit); zero `t.Skip`/`SkipNow` in `testdata/golden/*.go`; `lockedCorpusDir` fails loud via `t.Fatalf`; `-count=1` defeats the go test cache |
| 4   | FIXT-03: Removing a locked corpus turns the job red — demonstrated by doing it | ✓ VERIFIED | Family (d) rehearsal renamed the hugo tree and observed `TestPriorityLanguagesResolveToLockedCorpus` red (fail-NOT-skip); `task corpora:assert` fails on a missing corpus (four-part integrity check) |

**Score:** 4/4 truths verified (0 present, behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `.github/workflows/corpora.yml` | New unconditional `golden` job (fetch → assert → test:golden); widened transitive-closure path filter | ✓ VERIFIED | Job at lines 215–273; fetch/assert/test steps unconditional with explicit anti-gating comments; path filter names `internal/graphstore/**`, `internal/parser/**`, `internal/schema/**` (+ real closure) in both `pull_request` and `push` blocks; `internal/store/**` correctly absent |
| `.github/workflows/ci.yml` | Corpus-dependent `test:golden` step removed | ✓ VERIFIED | Step removed; replaced with a pointer comment; `rg "test:golden" .github/workflows/ci.yml` → exit 1 (empty) |
| `Taskfile.yml` | `test:golden` target carries `-count=1` | ✓ VERIFIED | `cmds: go test -count=1 ./testdata/golden/...` |
| `testdata/golden/golden_test.go` | `TestGoldenScenarioCountIsExact` + exact `verified` guard | ✓ VERIFIED | `ExpectedGoldenScenarioCount = 30` beside `expectedGoCaptures`; derives `goldenTotal` (26) + `caseTotal` (4) with exact equality; `TestReFrozenGoldensValid` guard upgraded to `verified != expectedTotal` |
| `testdata/golden/behavioral_test.go` | `executedCases` accumulator keyed to per-case closure | ✓ VERIFIED | Incremented inside the case loop; exact `!= len(cases)` post-loop assertion |
| `internal/upgrade/taskfile_shape_test.go` | `{corpora.yml, golden}` in `inScopeJobs` | ✓ VERIFIED | Entry added; `TestWorkflowRunBodiesInvokeTask` green |
| `.planning/phases/03-non-vacuity-proof-unconditional-ci-execution/03-MUTATION-LOG.md` | Per-family RED record (FIXT-07) | ✓ VERIFIED | Five family rows, each with mutation + pasted failure + revert + byte-clean proof; D-01 call and corpus-precondition headers recorded |

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | --- | --- | ------ | ------- |
| `TestGoldenScenarioCountIsExact` (test) | `test:golden -count=1` (task) | Taskfile `cmds` | ✓ WIRED | Verified `-count=1` in the target body; golden suite ran green via the task locally |
| `test:golden` (task) | unconditional `golden` job (workflow) | `corpora.yml` step `run: task test:golden` | ✓ WIRED | Job runs fetch → assert → test; fetch/assert unconditional, not cache-gated |
| `golden` job (workflow) | `inScopeJobs` (guard) | `taskfile_shape_test.go` entry | ✓ WIRED | `{Workflow: "corpora.yml", JobID: "golden"}` bound; guard test passes |
| 26-golden enumeration | authoritative `expectedGoCaptures` slice | `TestReFrozenGoldensValid` / `TestGoldenScenarioCountIsExact` | ✓ WIRED | Slice literal, not `filepath.Glob`; deterministic order |
| 4 CASES cases | real committed `corpus/behavioral/CASES.json` | `loadBehavioralCases` | ✓ WIRED | Reads `../../corpus/behavioral/CASES.json`; Fatals on empty list; 4 cases confirmed (a–d) |
| Family (d) rehearsal | `lockedCorpusDir` fail-loud path | `t.Fatalf` at behavioral_test.go:103 | ✓ WIRED | `os.Stat` + `t.Fatalf`, never `t.Skip` — fail-NOT-skip confirmed in code and in log |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| `TestGoldenScenarioCountIsExact` | `goldenTotal`, `caseTotal`, `total` | `expectedGoCaptures` slice + `len(loadBehavioralCases())` from committed CASES.json | Yes — 26 + 4 = 30, exact | ✓ FLOWING |
| `TestReFrozenGoldensValid` | `verified` counter | per-subtest `os.ReadFile` on real golden paths | Yes — 26/26 files read | ✓ FLOWING |
| `TestCorpusBehaviorSynthetic` | `executedCases` | in-loop counter over `loadBehavioralCases()` | Yes — 4/4 cases ran | ✓ FLOWING |
| `golden` job | corpora | `task corpora:fetch` → `task corpora:assert` (4/4 verified) → `task test:golden` | Yes — real pinned corpora at `~/.cache/codegraph/corpora` | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Clean build | `go build ./...` | exit 0 | ✓ PASS |
| Full golden suite green (after rehearsal) | `go test -count=1 ./testdata/golden/...` | `ok` 23.8s / 26.0s | ✓ PASS |
| Count self-assertion proves 30/30 exact | `go test -count=1 ./testdata/golden/... -run TestGoldenScenarioCountIsExact -v` | `golden suite scenario count: goldens: 26/26, CASES cases: 4/4, total: 30/30` | ✓ PASS |
| Golden byte-identity guard | `go test -count=1 ./testdata/golden/... -run TestReFrozenGoldensValid -v` | 26/26 subtests PASS | ✓ PASS |
| Behavioral properties (4 CASES cases) | `go test -count=1 ./testdata/golden/... -run TestCorpusBehaviorSynthetic -v` | 4/4 subtests PASS | ✓ PASS |
| inScopeJobs guard | `go test -count=1 ./internal/upgrade/... -run TestWorkflowRunBodiesInvokeTask` | `ok` | ✓ PASS |
| Coverage guard (family e) | `go test -count=1 ./internal/corpora/... -run TestCorpusCoverageClaim` | `ok` | ✓ PASS |
| Corpus fetch | `task corpora:fetch` | `fetched or confirmed 4/4 locked corpora` | ✓ PASS |
| Corpus assert | `task corpora:assert` | `verified 4/4 locked corpora` | ✓ PASS |
| No `test:golden` in ci.yml | `rg "test:golden" .github/workflows/ci.yml` | exit 1 (no matches) | ✓ PASS |
| No self-skip in golden tests | `rg "t.Skip|t.Skipf|SkipNow" testdata/golden/*.go` | no matches (all hits are comments asserting fail-loud posture) | ✓ PASS |
| `internal/store` absent | `ls -d internal/store` | No such file or directory | ✓ PASS (path filter correctly omits it) |

### Probe Execution

No probes declared for this phase; the phase is verified through the behavioral spot-checks above (test-layer, task-layer, and workflow-layer) plus the mutation-rehearsal record. Not applicable.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ----------- | ----------- | ------ | -------- |
| FIXT-03 | 03-01 | No golden test self-skips in CI; suite runs on every CI run; fetch/cache failure fails loudly; job carries a positive executed-scenario-count assertion | ✓ SATISFIED | `TestGoldenScenarioCountIsExact` (30/30 exact, executed-keyed counters); unconditional `golden` job in corpora.yml; `-count=1`; ci.yml step removed; no `t.Skip`; `internal/store` absent from filter |
| FIXT-07 | 03-02 | Re-baselined suite demonstrated RED against confirmed-applied, byte-cleanly-reverted mutation per family | ✓ SATISFIED (recorded; human read of pasted output recommended) | `03-MUTATION-LOG.md` — five families, each with mutation + pasted RED + revert + byte-clean proof; suite green afterward; no residue in `git diff` against pre-phase base |

No orphaned requirements: both `FIXT-03` and `FIXT-07` are claimed by phase plans and covered above.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| `testdata/golden/golden_test.go` | 131 | `continue` labeled "skip here" | ℹ️ Info | Duplicate-report avoidance within a fixture-parse check that already fails via `t.Errorf` in the parse test above — not a suite self-skip; no FIXT-03 impact |
| 03-01-SUMMARY.md / 03-02-SUMMARY.md | key-decisions/commits | Commit hashes referenced in SUMMARYs do not match the actual git log (`70dd07c`/`d9f343d`/`c7f9424`/`a13d7c3`/`ac8d7fb` vs actual `288e4ff`/`49a4c96`/`44b9247`/`8936546`/`e125ad6`) | ⚠️ Warning | Documentation accuracy only — the code state matches the SUMMARY descriptions exactly; the stale hashes are non-load-bearing evidence references |

No `TBD`/`FIXME`/`XXX` markers, no placeholder implementations, no hardcoded-empty data paths in the phase's changed source.

### Human Verification Required

#### 1. Read the FIXT-07 mutation-rehearsal record (03-MUTATION-LOG.md)

**Test:** Read `.planning/phases/03-non-vacuity-proof-unconditional-ci-execution/03-MUTATION-LOG.md`'s five family entries and confirm each pasted RED failure is a genuine observation of the family going red under its defined mutation — mutation applied → observed failure → byte-clean revert.

**Expected:** Each family row shows a real failing `go test` output whose message names the mutated assertion:
- (a) `Node("Validate"): got 2 defs, want 2` (defs-count boundary `!= 2` → `!= 1`)
- (b) `25/26 goldens verified` naming the missing `corpus/hugo/go-explore.json`
- (c) `CLI and MCP output diverge (EXPL-05)` on the behavioral row only
- (d) `lockedCorpusDir("go"): ... not found ... run 'task corpora:fetch'` (fail-NOT-skip)
- (e) `kind calls derived count 58812 below threshold 999999`

followed by a revert command and a green re-run proof for each.

**Why human:** Automation cannot re-derive a past observation without re-applying the mutations, which would mutate the working tree (prohibited during verification). The record is complete and internally consistent with the code (line numbers, message strings, and threshold values match the current tree exactly), but the genuineness of each pasted failure is a human judgment. The plan's own coverage metadata declares `human_judgment: true` for FIXT-07 with this same rationale.

### Gaps Summary

No gaps found. All four roadmap success criteria are verified against the codebase: the scenario-count self-assertion proves 30/30 exact execution, the golden job in corpora.yml is unconditional and fail-loud, ci.yml no longer runs the corpus-dependent suite, and the mutation log records all five assertion families going RED with byte-clean reverts and a green post-rehearsal suite.

The single outstanding item is the FIXT-07 observation-authenticity check, which the phase's own plan routes to a human reader of the mutation log. No code change, no wiring gap, and no blocker blocks proceeding to Phase 4.

---

_Verified: 2026-08-15T14:55:25Z_
_Verifier: Claude (gsd-verifier)_
