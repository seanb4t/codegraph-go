---
phase: 05-release-chain-release-please-retirement
plan: 05
subsystem: testing
tags: [guard-tests, changie, tag-authority, cosign-san, github-actions, mutation-testing]

requires:
  - phase: 05-release-chain-release-please-retirement
    provides: "05-01: release-pr.yml and the .changie.yaml Breaking kind; 05-02: release-tag.yml and the GRD-16 single-App-token-site scanner; 05-MUTATION-LOG.md convention"
provides:
  - "TestChangieBreakingBumpsMinorPreOne: pins exactly one Breaking kind in .changie.yaml with auto: minor (replaces the guard that read the retired JSON config)"
  - "TestGsdTagCreationIsDisabled: logic unchanged, prose now names the changie workflow App-token step as sole tag authority"
  - "SAN negative fixtures in release_workflow_shape_test.go and taskfile_shape_test.go name release-tag.yml at a tag ref"
  - "cosignIdentitySANCorpus accept verdicts now bind to releaseWorkflowRefPattern, one assertion per row"
  - "ci.yml and post-release-verify.yml comments tell the changie-chain story; no functional line changed"
  - "05-MUTATION-LOG.md: RED against the replacement, the post-re-point simulation, Families 05-05/a-c"
affects: [05-07]

plan_head_before: 521a708409dc21366e8177634a97973b3f9b34b6
plan_head_after: 013322f708c593301fa3f799889547928218abe6

actuals:
  tokens: 7132
  tasks: 2
  commits: 6

tech-stack:
  added: []
  patterns:
    - "a guard that reads a config key matches the entry by label, requires exactly one match, and logs how many entries it scanned"
    - "REL-14's end state is simulated in a detached scratch worktree (the three retired files git rm'd) to show which tests pinned them, then re-run to show the re-pointed ones no longer do"

key-files:
  created: []
  modified:
    - internal/upgrade/tag_authority_test.go
    - internal/upgrade/release_workflow_shape_test.go
    - internal/upgrade/taskfile_shape_test.go
    - .github/workflows/ci.yml
    - .github/workflows/post-release-verify.yml
    - .planning/phases/05-release-chain-release-please-retirement/05-MUTATION-LOG.md

key-decisions:
  - "Test name TestChangieBreakingBumpsMinorPreOne (D-09 requires the rename, the old name matched the census pattern)"
  - "TestGsdTagCreationIsDisabled 'unchanged' means: code tokens outside comments and t.Fatalf lines byte-identical to 5a8603f5; the doc comment and two messages that named the retired tool are reworded under D-09"
  - "SAN negative fixtures name release-tag.yml, the wrong-workflow-in-the-right-repo-at-a-tag-ref case"
  - "The corpus accept field is asserted against the compiled pattern (Rule 2 deviation below) so Family 05-05/c can be RED"

patterns-established:
  - "A corpus field that is only counted is decoration: bind it to the subject, then prove it RED by swapping a row"

requirements-completed: []

coverage:
  - id: D1
    description: "The pre-major guard pins .changie.yaml's Breaking kind to auto: minor, exactly one entry, matched by label; it went RED against the replacement simulation first"
    requirement: GRD-15
    verification:
      - kind: unit
        ref: "internal/upgrade#TestChangieBreakingBumpsMinorPreOne (Families 05-05/a: major flip, duplicated kind, missing kind all RED)"
        status: pass
    human_judgment: false
  - id: D2
    description: "TestGsdTagCreationIsDisabled passes with identical code tokens and still fails when git.create_tag is true"
    requirement: GRD-16
    verification:
      - kind: unit
        ref: "internal/upgrade#TestGsdTagCreationIsDisabled (Family 05-05/b)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Both SAN negative fixtures name release-tag.yml and reject; swapping the corpus row for release.yml is RED"
    requirement: GRD-15
    verification:
      - kind: unit
        ref: "internal/upgrade#TestReleaseWorkflowFileMatchesPattern/reject_renamed_workflow, TestCosignIdentityPolicyBoundaryParityWithCompiledPattern (Family 05-05/c)"
        status: pass
    human_judgment: false
  - id: D4
    description: "ci.yml and post-release-verify.yml comments reworded with no functional line changed; the workflow_run trigger and its do-not-simplify warning kept"
    requirement: GRD-15
    verification:
      - kind: other
        ref: "git diff -U0 5a8603f5 over both files shows only comment lines; task lint:actions exit 0"
        status: pass
    human_judgment: false

duration: 4min
completed: 2026-10-14
status: complete
---

# Phase 5 Plan 05: Pre-major guard re-point and census rewording Summary

**The release-please-pinning tests are re-pointed at the changie chain: the pre-1.0 guard now reads .changie.yaml's Breaking kind, the wrong-workflow SAN fixtures name release-tag.yml, and a decorative SAN-corpus field became a real assertion, each proven RED against the replacement or a mutation.**

## Performance

- **Duration:** 4 min
- **Started:** 2026-10-14T22:13:10Z
- **Completed:** 2026-10-14T22:17:00Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments

- RED against the replacement shown first: with the three retired files removed in a scratch worktree, exactly four tests fail (transcript below).
- TestChangieBreakingBumpsMinorPreOne replaces TestReleasePleaseStaysPreMajor and the JSON config path constant. It fails on a flip to major, on a duplicated Breaking kind and on a missing one.
- TestGsdTagCreationIsDisabled keeps its code tokens; only prose naming the retired tool changed (D-09).
- SAN negative fixtures re-pointed at release-tag.yml; the SAN corpus now asserts each row's accept verdict against the compiled pattern.
- Comment-only rewording of ci.yml (actionlint comment), post-release-verify.yml (trigger rationale) and TestInScopeJobsPopulationMatchesDisk's doc.

## RED against the replacement

Scratch worktree of HEAD `521a7084`, `git rm` of `release-please.yml`, `release-please-config.json`, `.release-please-manifest.json`; `go test ./internal/upgrade/ -count=1 -v` exit 1. Full text is in 05-MUTATION-LOG.md under `## 05-05 RED against the replacement`.

```
=== RUN   TestReleasePleaseStaysPreMajor
    tag_authority_test.go:93: read ../../release-please-config.json: open ../../release-please-config.json: no such file or directory
--- FAIL: TestReleasePleaseStaysPreMajor (0.00s)
=== RUN   TestWorkflowRunBodiesInvokeTask
    taskfile_shape_test.go:1400: read ../../.github/workflows/release-please.yml: open ../../.github/workflows/release-please.yml: no such file or directory
--- FAIL: TestWorkflowRunBodiesInvokeTask (0.00s)
=== RUN   TestWorkflowFilePopulationMatchesDisk
    taskfile_shape_test.go:1613: inScopeWorkflowFiles names "release-please.yml", which no longer exists on disk
--- FAIL: TestWorkflowFilePopulationMatchesDisk (0.00s)
=== RUN   TestInScopeJobsPopulationMatchesDisk
    taskfile_shape_test.go:1669: usesOnlyJobExceptions: release-please.yml/release-please: read ../../.github/workflows/release-please.yml: open ../../.github/workflows/release-please.yml: no such file or directory
--- FAIL: TestInScopeJobsPopulationMatchesDisk (0.00s)
```

The first is re-pointed here; the other three are registry tests whose entries 05-07 removes with the files.

## Post-re-point simulation

Same simulation at `252d4096` (the re-point commit), `-run '^(TestChangieBreakingBumpsMinorPreOne|TestGsdTagCreationIsDisabled)$'`:

```
--- PASS: TestGsdTagCreationIsDisabled (0.00s)
    tag_authority_test.go:129: scanned 5 kinds; Breaking auto = minor
--- PASS: TestChangieBreakingBumpsMinorPreOne (0.00s)
PASS
```

## TestGsdTagCreationIsDisabled code-token equality

Plan verify command: the function body from `5a8603f5` and from HEAD, with full-line comments and `t.Fatalf(` lines removed, compared with `test "$old" = "$new"` (non-empty). Result: equal. The only changed lines in the function are its doc comment and the two t.Fatalf messages naming the retired tool.

## Mutation families (all in 05-MUTATION-LOG.md)

- `## Family 05-05/a`: Breaking `auto: major` is RED in TestChangieBreakingBumpsMinorPreOne and TestChangieConfigShape; a further section `Family 05-05/a, count edges` covers a duplicated and a missing Breaking kind.
- `## Family 05-05/b`: `"create_tag": true` is RED in TestGsdTagCreationIsDisabled.
- `## Family 05-05/c`: the corpus row swapped to release.yml is RED in TestCosignIdentityPolicyBoundaryParityWithCompiledPattern.

## Task Commits

1. Task 1: `252d4096` test(05-05): pin pre-1.0 Breaking to minor in .changie.yaml, and name the changie workflow as tag authority; `b1b1768c` docs(05-05): record the RED-against-replacement and pre-major re-point evidence
2. Task 2: `5e357cde` test(05-05): re-point the wrong-workflow SAN fixtures at release-tag.yml; `1c4ef019` ci(05-05): reword the retired workflow out of the actionlint and post-release-verify comments; `91a75711` test(05-05): make the SAN corpus accept verdicts bind to the compiled pattern; `013322f7` docs(05-05): record mutation family 05-05/c

**Plan metadata:** the `docs(05-05): complete ...` commit that follows this file.

## TDD Gate Compliance

The RED evidence for this plan is the scratch-worktree transcript above plus mutations, recorded in 05-MUTATION-LOG.md before the re-point (rule x1cjy9vyhq: Go RED is the pasted `--- FAIL:` output; `gsd-tools check tdd-red-evidence` is TAP-only and was skipped under that rule). The plan is `type: execute`; there is no separate `feat` commit because the change is test-only. The `test(05-05):` commits carry the implementation.

## Decisions Made

See `key-decisions` above. All are two-way doors recorded by the plan.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing critical functionality] cosignIdentitySANCorpus.accept was never asserted against the pattern**
- **Found during:** Task 2, running Family 05-05/c
- **Issue:** Changing the corpus row "different workflow filename, same repo, tag ref" to release.yml left `go test ./internal/upgrade/ -count=1` at `ok`. The `accept` field was only counted by check 5 (at least 2 accepted, 4 rejected) and the parity check compares extracted literals to the compiled pattern, so no test read a row's verdict against `releaseWorkflowRefPattern`. The negative fixture the plan re-points was therefore decoration, and Family 05-05/c could not be RED (rule 84d1gfpywd).
- **Fix:** One loop in TestCosignIdentityPolicyBoundaryParityWithCompiledPattern: for each corpus row, `compiled.MatchString(c.san)` must equal `c.accept`. All existing rows already satisfied it. The probe result (GREEN before) is recorded in Family 05-05/c.
- **Files modified:** internal/upgrade/taskfile_shape_test.go
- **Commit:** `91a75711`

**Total deviations:** 1 auto-fixed (Rule 2). **Impact:** one extra `test(05-05):` commit; no change to any decision; strengthens the guard the plan relied on.

### Plan notes

- Scratch worktrees lived under the session scratchpad (`.../exec-05-05/*/p05-scratch-wt`) rather than `mktemp -d`, per the orchestrator note. All were removed and pruned; `git worktree list` shows no `p05-scratch-wt`.
- `.git/gsd-plan-head-before-05-05` held a non-ancestor commit and was reset to the starting HEAD `521a7084` before work began.
- A full `go test ./...` was not run (known internal/daemon flush-lock/TestSoak flake under load); `go test ./internal/upgrade/ -count=1`, `go vet ./internal/upgrade/`, `go build ./...` and `task lint:actions` all pass.

## Issues Encountered

None.

## Verification

- `go test ./internal/upgrade/ -count=1`: ok. `task lint:actions`: exit 0.
- Census: no hit in release_workflow_shape_test.go, tag_authority_test.go, ci.yml or post-release-verify.yml. taskfile_shape_test.go holds exactly 5 hit lines (the inScopeJobs entry, the three lines of the usesOnlyJobExceptions entry, inScopeWorkflowFiles) for 05-07 to remove with the files.
- ci.yml and post-release-verify.yml differ from `5a8603f5` in comment lines only; post-release-verify.yml's `workflows: ["release"]` and `types: [completed]` are untouched.
- `.planning/config.json` still has `git.create_tag == false`; no mutation leaked into the main tree.
- Retired files, their registry entries and `release-please.yml` are untouched (05-07).

## Requirements

`requirements-completed: []`. GRD-15 is partial: the registry entries and the three retired files go in 05-07. GRD-16 is partial here: it completes with 05-02's TestTagCreationHasSingleAppTokenSite plus this plan's unchanged-test evidence (the orchestrator reconciles at phase close).

## Next Phase Readiness

05-07 can delete the three retired files and the five registry hit lines; the simulation shows the only tests then failing are the three registry tests it also edits.

## Known Stubs

None.

## Threat Flags

None.

## Self-Check: PASSED
