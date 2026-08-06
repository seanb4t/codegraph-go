---
phase: 04-supply-chain-coverage-daemon-substrate-fixes
plan: 03
subsystem: infra
tags: [goreleaser, taskfile, ci, text-assertion-tests, govulncheck, workflow-yaml]

# Dependency graph
requires:
  - phase: 04-01
    provides: "Taskfile.yml vuln target (advisory), ci.yml tool-vuln job (advisory)"
provides:
  - "TestGoreleaserPinParity: fails the build if go.tool.mod's goreleaser require version and release.yml's GORELEASER_VERSION ever diverge again"
  - "TestGateStancesStated: fails the build if the vuln task desc:, tool-vuln job name:, or transcript-freeze's advisory-guard step name: stop stating ADVISORY, or if tool-vuln and transcript-freeze's stances stop agreeing"
  - "ci.yml's tool-vuln job added to inScopeJobs — bound by the single-definition guard (TestWorkflowRunBodiesInvokeTask)"
  - "release.yml's GORELEASER_VERSION aligned to v2.17.1 (was v2.17.0), matching go.tool.mod, with provenance recorded in the replacement comment"
affects: []

# Actuals (#2632)
actuals:
  tokens: 3856
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns: ["parseX/mustX pure-parser convention extended with parseWorkflowEnvValue (workflow-level env: scoping) and parseTaskDescription (inline + folded block-scalar desc: extraction)"]

key-files:
  created: []
  modified:
    - .github/workflows/release.yml
    - internal/upgrade/taskfile_shape_test.go

key-decisions:
  - "MAINT-03 pin alignment direction confirmed by git history, not assumed: git log --follow -- go.tool.mod returns exactly one commit (82ffd60, 2026-08-01, build(taskfile): add Taskfile.yml + isolated tool modfiles...), which created the file already carrying goreleaser v2.17.1 — never bumped since. git log -S'GORELEASER_VERSION' -- .github/workflows/release.yml returns exactly one commit (ee258d9, 2026-07-13, feat(08-04): native 2-OS runner matrix...). Aligned upward to v2.17.1 because go.tool.mod's pin has live, repeated PR validation (task check:goreleaser, task check:darwin-release-build) while v2.17.0's 'validated locally in Plan 08-01' comment predated both the Taskfile rewiring and go.tool.mod's existence."
  - "VULN-03 stance test re-derived per D-04's mid-phase supersession (04-CONTEXT.md, 04-01-SUMMARY.md): the plan was authored expecting a deliberate BLOCKING-vs-ADVISORY divergence between the tool-modfile scan and transcript-freeze. Implementation surfaced a real, permanently-unfixed vulnerability (GO-2026-5932) in goreleaser's own binary that would have made a blocking gate permanently red, so the tool-modfile scan shipped ADVISORY instead — now matching transcript-freeze rather than differing from it. TestGateStancesStated asserts agreement (both state ADVISORY, and they agree with each other), not divergence, with a comment recording the re-derivation."
  - "Reused the existing releaseWorkflowPath constant from release_workflow_shape_test.go (same package) instead of declaring a duplicate constant as the plan's read_first literally suggested — a duplicate declaration would not compile."
  - "The vuln task's desc: legitimately narrates its own history using both 'ADVISORY' and 'BLOCKING' (e.g. 'this target was designed BLOCKING and was demoted to advisory'), so the stance test checks vulnDesc for ADVISORY via substring only, never via the exactly-one-word parseGateStanceWord parser used for the short job/step display strings — applying the strict single-word parser to the desc prose would have made the guard itself fail against legitimate historical narration."

patterns-established:
  - "parseGateStanceWord: a case-insensitive, exactly-one-of-{advisory,blocking} extractor for short display strings (job name:, step name:) — distinct from a plain substring check, which remains the right tool for free-form prose (desc:) that may legitimately narrate a stance's history using both words."

requirements-completed: [VULN-03, MAINT-03]

coverage:
  - id: D1
    description: "release.yml's GORELEASER_VERSION aligned to v2.17.1, matching go.tool.mod's goreleaser require line, with the replacement comment recording the provenance (both pin sites' git history, the alignment direction, and the enforcing test) and no other line in the file touched"
    requirement: MAINT-03
    verification:
      - kind: other
        ref: "git diff .github/workflows/release.yml (pre-Task-2) touches only the env: block's comment and value; task lint:actions and task check:goreleaser both pass"
        status: pass
      - kind: unit
        ref: "go test ./internal/upgrade/ -run TestGoreleaserPinParity"
        status: pass
    human_judgment: false
  - id: D2
    description: "TestGoreleaserPinParity fails the build on any future divergence between go.tool.mod's goreleaser pin and release.yml's GORELEASER_VERSION, naming both file paths and both versions in the failure message"
    requirement: MAINT-03
    verification:
      - kind: unit
        ref: "go test ./internal/upgrade/ -run TestGoreleaserPinParity (mutation: release.yml pin set to v2.17.0 against go.tool.mod's v2.17.1 — observed red with the exact expected message, then cleanly reverted; see Mutation Demonstrations below)"
        status: pass
    human_judgment: false
  - id: D3
    description: "TestGateStancesStated asserts the vuln task's desc:, the tool-vuln job's name:, and transcript-freeze's advisory-guard step name: all state ADVISORY, and that tool-vuln and transcript-freeze's stances agree with each other — re-derived from the plan's original divergence-pair design per D-04's supersession"
    requirement: VULN-03
    verification:
      - kind: unit
        ref: "go test ./internal/upgrade/ -run TestGateStancesStated (three mutations: vuln desc: stance wording removed, tool-vuln job renamed, transcript-freeze step renamed — each observed red with the expected message naming the offending site, then cleanly reverted; see Mutation Demonstrations below)"
        status: pass
    human_judgment: false
  - id: D4
    description: "ci.yml's tool-vuln job added to inScopeJobs with no new runBodyExceptions entry; TestWorkflowRunBodiesInvokeTask passes against it and fails when an inline go build step is added"
    requirement: VULN-03
    verification:
      - kind: unit
        ref: "go test ./internal/upgrade/ -run TestWorkflowRunBodiesInvokeTask (mutation: added a 'go build ./...' step to tool-vuln — observed red naming the offending step and verb, then cleanly reverted)"
        status: pass
    human_judgment: false

duration: 40min
completed: 2026-08-06
status: complete
---

# Phase 4 Plan 3: GoReleaser Pin Parity and Gate-Stance Text-Assertion Guards Summary

**Aligned release.yml's GoReleaser pin upward to v2.17.1 (matching go.tool.mod, per confirmed git-history provenance) and added two text-assertion tests — TestGoreleaserPinParity and TestGateStancesStated (re-derived from a divergence-pair to an agreement-pair per D-04's mid-phase supersession) — each demonstrated red against a confirmed-applied mutation before being trusted.**

## Performance

- **Duration:** ~40 min
- **Started:** 2026-08-06T19:53:42Z (Task 1 commit)
- **Completed:** 2026-08-06T20:00:18Z (Task 2 commit) — reading/verification/mutation-demonstration work preceded and followed both commits
- **Tasks:** 2
- **Files modified:** 2 (`.github/workflows/release.yml`, `internal/upgrade/taskfile_shape_test.go`)

## Accomplishments

- Confirmed both git-history findings the plan's provenance claim rested on, exactly as recorded: `go.tool.mod` created already at goreleaser v2.17.1 in `82ffd60` (2026-08-01), never bumped since; `release.yml`'s v2.17.0 set once in `ee258d9` (2026-07-13) — 19 days apart, neither a deliberate move away from the other.
- Aligned `release.yml`'s `GORELEASER_VERSION` upward to `v2.17.1`, replacing the stale "validated locally in Plan 08-01" comment with one naming both pin sites, the enforcing test, and the one-line reason for the alignment direction. `git diff` touches only the `env:` block.
- Added `parseWorkflowEnvValue` (workflow-level-only `env:` key extraction, rejecting job/step-level `env:` blocks at deeper indentation) and `parseTaskDescription` (inline and YAML folded/literal block-scalar `desc:` extraction) to `internal/upgrade/taskfile_shape_test.go`, following the file's established `parseX`/`mustX` non-nil-error-on-miss convention.
- Added `TestGoreleaserPinParity`: reads both pin sites off disk, normalizes the leading `v`, and fails naming both file paths and both versions on any divergence.
- Added `TestGateStancesStated`, re-derived from the plan's original design (see Deviations below) to assert agreement rather than divergence between the tool-modfile scan's stance and transcript-freeze's stance, since D-04 was superseded mid-phase and both are now deliberately ADVISORY.
- Added `ci.yml`'s `tool-vuln` job to `inScopeJobs`; `TestWorkflowRunBodiesInvokeTask` passes against it with zero new `runBodyExceptions` entries, confirming both of 04-01's `task vuln` / `task vuln:selftest` steps already satisfy the single-definition property.
- Demonstrated all five required mutations red, each with the exact expected failure message, each cleanly reverted (`git diff` empty afterward) — full log below.

## Task Commits

1. **Task 1: MAINT-03 — align the pin upward, provenance recorded in the comment** - `c3f82f0` (build)
2. **Task 2: The two guards — pin parity and stated stance, each demonstrated red first** - `c220cdb` (test)

## Files Created/Modified

- `.github/workflows/release.yml` — `GORELEASER_VERSION` changed `v2.17.0` → `v2.17.1`; replacement comment records provenance, the enforcing test name, and the alignment-direction reason. No other line changed.
- `internal/upgrade/taskfile_shape_test.go` — added `parseWorkflowEnvValue`/`mustWorkflowEnvValue`, `parseTaskDescription`/`mustTaskDescription`, `hasStanceWord`/`parseGateStanceWord`, `TestGoreleaserPinParity`, `TestGateStancesStated`, two new fail-loudly cases in `TestTaskfileShapeHelpersFailLoudly` (`parseWorkflowEnvValue`, `parseTaskDescription` empty-input), `TestParseWorkflowEnvValue_NoWorkflowLevelEnvBlockIsError` (the jobs-but-no-env-block edge case), and one `inScopeJobs` entry (`ci.yml`/`tool-vuln`).

## Decisions Made

**MAINT-03 — pin alignment direction, confirmed not assumed:**

- `git log --oneline --follow -- go.tool.mod` → exactly one commit: `82ffd60` (2026-08-01) — `build(taskfile): add Taskfile.yml + isolated tool modfiles, route CI through task (#19)`. The file was created already carrying `github.com/goreleaser/goreleaser/v2 v2.17.1 // indirect`; it has never been bumped since.
- `git log --oneline -S'GORELEASER_VERSION' -- .github/workflows/release.yml` → exactly one commit: `ee258d9` (2026-07-13) — `feat(08-04): native 2-OS runner matrix builds all 6 raw release binaries`. `GORELEASER_VERSION: "v2.17.0"` has never changed since.
- 19 days apart, both single-appearance commits — neither pin was ever deliberately moved away from the other. Aligned upward to `v2.17.1` because `go.tool.mod`'s pin carries live, repeated PR validation (`task check:goreleaser`, `task check:darwin-release-build`) while `release.yml`'s "validated locally in Plan 08-01" provenance predated both the Taskfile rewiring and `go.tool.mod`'s existence. `v2.17.1`'s changelog (GORISCV64/GOPPC64 target-name parsing, build-target sorting) is inert for this repo's linux/darwin/windows amd64/arm64 matrix.

**VULN-03 — stance test re-derived from divergence to agreement (D-04 supersession):**

- The plan's Task 2 `<action>` was authored under the pre-checkpoint assumption that the tool-modfile scan would ship BLOCKING, deliberately differing from transcript-freeze's ADVISORY stance — and directed the test to assert "the two stances are different from each other."
- 04-01's Task 1 checkpoint superseded D-04 mid-phase: implementation surfaced a real, permanently-unfixed, symbol-reachable vulnerability (`GO-2026-5932`) in `goreleaser`'s own binary that would have made a blocking gate permanently red from its first CI run. The tool-modfile scan shipped ADVISORY instead. 04-CONTEXT.md's replacement text states explicitly: "This now matches the D-03 transcript guard's stance rather than deliberately differing from it. The earlier note about two coexisting stances no longer applies."
- Per this plan's own `critical_execution_facts` instruction ("Any acceptance criterion or assertion in your plan that expects the word 'BLOCKING' ... must be re-derived to expect 'ADVISORY' ... do the same rather than silently reinterpreting"), `TestGateStancesStated` was re-derived: it asserts all three sites (vuln desc:, tool-vuln job name:, transcript-freeze advisory-guard step name:) state ADVISORY, and that tool-vuln's and transcript-freeze's stance words are EQUAL to each other (not different) — so a future silent flip of either site back to BLOCKING, re-introducing an unstated mismatch, still fails loudly. A comment directly above the test records this re-derivation and cites both source documents.
- This is not a Rule 4 architectural stop: it falls squarely within the pre-authorized re-derivation the orchestrator's `critical_execution_facts` explicitly anticipated and directed, extended logically from "the word changes" to "the comparison direction changes" (since the underlying fact — the two gates' stances no longer diverge — is the same fact driving both).

**Minor implementation decisions:**

- Reused the existing `releaseWorkflowPath` constant (declared in `release_workflow_shape_test.go`, same `upgrade` package) rather than declaring a second one in `taskfile_shape_test.go` as the plan's `read_first` literally suggested — Go does not allow duplicate constant declarations in one package, so this is a Rule 1 auto-fix, not a design choice.
- `parseGateStanceWord` (exactly-one-of-{advisory,blocking}, case-insensitive whole word) is used only for short display strings (job `name:`, step `name:`) where an unambiguous single-word stance statement is the expected shape. The `vuln` task's `desc:` is free-form prose that legitimately narrates its own history using both words ("this target was designed BLOCKING and was demoted to advisory") — applying the strict single-word parser there would make the guard fail against the real, correct file. The stance test therefore checks the `desc:` via a plain case-insensitive substring (`hasStanceWord`) instead, matching the plan's own instruction to "match on stance wording case-insensitively and on the word itself, not on a whole formatted string."

## Mutation Demonstrations (all five, each observed red then cleanly reverted)

| # | Mutation | Command | Observed failure message | Reverted, `git diff` empty? |
|---|---|---|---|---|
| 1 | `release.yml`'s `GORELEASER_VERSION` changed `v2.17.1` → `v2.17.0` | `go test ./internal/upgrade/... -run TestGoreleaserPinParity -v` | `GoReleaser pin mismatch (MAINT-03): ../../go.tool.mod requires goreleaser@v2.17.1 but ../../.github/workflows/release.yml sets GORELEASER_VERSION=v2.17.0 — the two pin sites must name the same version` | Yes |
| 2 | `vuln` task's `desc:` — both "ADVISORY" and "advisory" occurrences replaced with non-stance text | `go test ./internal/upgrade/... -run TestGateStancesStated -v` | `Taskfile.yml vuln task's desc: does not state the ADVISORY stance (VULN-03): "MUTATION-TEST-REMOVED-STANCE (VULN-01/VULN-03; ...)"` (full desc text quoted) | Yes |
| 3 | `tool-vuln` job's `name:` — `, advisory)` suffix removed | `go test ./internal/upgrade/... -run TestGateStancesStated -v` | `ci.yml tool-vuln job's name: parseGateStanceWord: text states neither "advisory" nor "blocking": "tool-vuln (VULN-01/02/03)"` | Yes |
| 4 | transcript-freeze's advisory-guard step — `advisory ` removed from `name:` | `go test ./internal/upgrade/... -run TestGateStancesStated -v` | `ci.yml transcript-freeze job has no step whose name: states the ADVISORY stance (D-03)` | Yes |
| 5 | Added a `run: go build ./...` step to `tool-vuln` job | `go test ./internal/upgrade/... -run TestWorkflowRunBodiesInvokeTask -v` | `ci.yml job "tool-vuln" step "MUTATION-TEST inline invocation" invokes "go build" directly instead of a task target — single-definition property violated (D-01)` | Yes |

After each revert, `git diff` for the mutated file returned zero lines (confirmed individually via `git diff <file> \| wc -l` → `0`).

## Deviations from Plan

### 1. [Rule 4-adjacent, pre-authorized by the orchestrator's critical_execution_facts — not silently reinterpreted] Stance test re-derived from a divergence assertion to an agreement assertion

- **Found during:** Task 2 planning (before writing `TestGateStancesStated`)
- **Issue:** The plan's `<action>` and `<behavior>` blocks, written before D-04's mid-phase supersession, instructed the stance test to assert "the two stances are different from each other" (a deliberate blocking-vs-advisory pair). Since 04-01's checkpoint, both the tool-modfile scan and transcript-freeze are ADVISORY — asserting "different" would make the guard permanently and correctly-failingly red against the real, correct file, which is not the intent.
- **Fix:** Re-derived per the orchestrator's explicit instruction to re-derive (not silently reinterpret) any blocking-assuming assertion. `TestGateStancesStated` now asserts all three sites state ADVISORY and that the tool-vuln/transcript-freeze pair agree with each other — the "asserted as a pair, not two independent checks" property the plan wanted is preserved; only the expected relationship (equal, not different) changed, matching 04-CONTEXT.md's own supersession text.
- **Files modified:** `internal/upgrade/taskfile_shape_test.go`
- **Verification:** `go test ./internal/upgrade/... -run TestGateStancesStated` passes against the real files; three of the five mutation demonstrations (2, 3, 4 above) directly exercise this test and each fails with a message naming the offending site.
- **Committed in:** `c220cdb`

### 2. [Rule 1 — auto-fix, not a design choice] Reused existing `releaseWorkflowPath` constant instead of declaring a duplicate

- **Found during:** Task 2, before writing the pin-parity test
- **Issue:** The plan's `read_first`/`action` text says to add a `release.yml` path constant to `taskfile_shape_test.go`. `release_workflow_shape_test.go` (same `internal/upgrade` package) already declares `const releaseWorkflowPath = "../../.github/workflows/release.yml"`. Declaring a second constant with the same name would not compile (duplicate declaration in one package).
- **Fix:** Reused the existing constant directly; no new constant added.
- **Files modified:** `internal/upgrade/taskfile_shape_test.go`
- **Verification:** `go build ./internal/upgrade/...` and `go vet ./internal/upgrade/...` both clean; `go test ./internal/upgrade/...` passes.
- **Committed in:** `c220cdb`

---

**Total deviations:** 2 (1 pre-authorized re-derivation per the orchestrator's explicit instruction, 1 Rule 1 auto-fix to avoid a compile error)
**Impact on plan:** Both deviations are documented, not silently applied. The stance test's re-derivation preserves the plan's stated intent (a pair-asserted, self-defeat-proof stance guard) under the corrected fact (agreement, not divergence). The constant reuse has zero behavioral impact.

## Issues Encountered

None beyond the two deviations documented above.

## User Setup Required

None - no external service configuration required.

## Operator Action Item (carried forward from the threat model)

T-04-12 (from the plan's threat register): the new `tool-vuln` job is **not** a required branch-protection status check. Making it one is a GitHub ruleset (20157557) change outside this repository's scope — this plan's in-repo mitigations are `TestGateStancesStated` (stance legibility) and the `inScopeJobs` binding (single-definition property). A maintainer should decide separately whether `tool-vuln` (or `vuln:selftest` specifically, which is fail-capable even though the parent job is advisory) should join `requiredCheckNames`.

## Next Phase Readiness

- Phase 4 is now complete: 04-01 (VULN-01/02/03 scanning mechanism), 04-02 (MAINT-01/02 daemon join discipline), 04-03 (VULN-03 stance legibility + MAINT-03 pin parity) all landed.
- `GO-2026-5932` in `goreleaser` remains a real, accepted, unmitigated exposure in this project's release tooling (recorded in 04-01-SUMMARY.md) — the advisory `task vuln` report is the only thing surfacing it; this plan did not change that.
- The residual darwin release-path check (release.yml's goreleaser/cosign/SLSA steps not yet run on the macOS runner class outside the permanent canary) is unaffected by this plan — MAINT-03 touched only the pin value and comment.

## Self-Check: PASSED

- `.github/workflows/release.yml` — FOUND, `GORELEASER_VERSION: "v2.17.1"` present
- `internal/upgrade/taskfile_shape_test.go` — FOUND, `TestGoreleaserPinParity` and `TestGateStancesStated` present
- Commit `c3f82f0` — FOUND in `git log --oneline`
- Commit `c220cdb` — FOUND in `git log --oneline`
- `go test ./internal/upgrade/...` — PASS (full package, including all pre-existing tests)
- `task lint:actions`, `task check:goreleaser`, `task test:unit` — all PASS

---
*Phase: 04-supply-chain-coverage-daemon-substrate-fixes*
*Completed: 2026-08-06*
