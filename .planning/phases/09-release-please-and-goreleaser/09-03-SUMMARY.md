---
phase: 09-release-please-and-goreleaser
plan: 03
subsystem: infra
tags: [release-please, github-actions, actionlint, conventional-commits, ci]

# Dependency graph
requires:
  - phase: 09-release-please-and-goreleaser
    provides: "09-01's mustWorkflowStepRunBlock parse-core helper (release_workflow_shape_test.go) — consumed unchanged to extract the PR-title lint step's literal shell"
provides:
  - "internal/upgrade/pr_title_lint_test.go — TestPRTitleLintAcceptsAndRejects, a 17-row table proving the PR-title lint's shipped shell against 4 reject rows, 10 per-type-word accept rows, a scope row, a breaking-marker row, and an injection-safety row"
  - ".github/workflows/pr-title.yml — dedicated pr-title workflow (D-08), pull_request trigger including edited, env-indirected TITLE, no third-party uses:"
  - ".github/workflows/ci.yml's new actionlint job — statically checks all 5 repo workflow files on every PR/push"
  - "Recorded divergence: D-08's 'add to ci.yml' becomes a dedicated pr-title.yml (RESEARCH Pitfall 1)"
affects: [09-04, 09-05, 09-06, 09-07, 09-08]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Table-driven accept/reject shell-execution test (09-02's stubbed-gh idiom, simplified: no PATH stub needed, title supplied purely via env var) — reused for the PR-title lint rather than inventing a third shape"

key-files:
  created:
    - internal/upgrade/pr_title_lint_test.go
    - .github/workflows/pr-title.yml
  modified:
    - .github/workflows/ci.yml

key-decisions:
  - "D-08's 'add to ci.yml' implemented as a dedicated pr-title.yml instead (RESEARCH Pitfall 1) — recorded in the plan's <recorded_divergence> block and restated in this workflow's own header comment, not silently taken"
  - "Re-resolved rhysd/actionlint's latest release live via gh api before writing the pinned version: confirmed v1.7.12, matching the plan's cached value — no delta to record"
  - "actionlint installed via go install at a pinned tag rather than a fourth-party pinned Action — no new supply-chain dependency, matches this repo's pure-Go CI tooling preference"

patterns-established:
  - "Env-var-only adversarial injection test: a title row containing $(...), backticks, and `; rm -rf /` is asserted BOTH to pass the lint AND to leave no marker file on disk — the executable form of the env-indirection guarantee, not an assertion-in-a-comment"

requirements-completed: []

coverage:
  - id: D1
    description: "TestPRTitleLintAcceptsAndRejects executes pr-title.yml's own shipped lint step against 4 reject rows (bare title, unaccepted type word, missing colon-space separator, empty subject) — each asserted non-zero exit plus a ::error:: annotation naming the offending title"
    requirement: "REL-02"
    verification:
      - kind: unit
        ref: "internal/upgrade/pr_title_lint_test.go#TestPRTitleLintAcceptsAndRejects (reject_bare_descriptive_title_no_type_prefix, reject_unaccepted_type_word, reject_missing_colon_space_separator, reject_empty_subject_after_colon)"
        status: pass
    human_judgment: false
  - id: D2
    description: "13 accept rows pass exit 0: one per accepted type word (feat/fix/perf/refactor/docs/chore/ci/test/build/revert), a scoped title, a breaking-change-marker title, and an adversarial shell-metacharacter title proven to produce no side effect"
    requirement: "REL-02"
    verification:
      - kind: unit
        ref: "internal/upgrade/pr_title_lint_test.go#TestPRTitleLintAcceptsAndRejects (all accept_* subtests + accept_adversarial_shell_metacharacters_no_side_effect)"
        status: pass
    human_judgment: false
  - id: D3
    description: "pr-title.yml's pull_request trigger explicitly lists edited alongside opened/synchronize/reopened, closing RESEARCH Pitfall 1 without widening ci.yml's shared trigger"
    requirement: "REL-02"
    verification:
      - kind: other
        ref: "grep -c 'edited' .github/workflows/pr-title.yml == 4; git diff -- .github/workflows/ci.yml release.yml (empty before Task 3)"
        status: pass
    human_judgment: false
  - id: D4
    description: "actionlint job added to ci.yml statically checks all 5 workflow files on every PR/push; diff confined to added lines only"
    requirement: "REL-02"
    verification:
      - kind: unit
        ref: "actionlint .github/workflows/*.yml (exit 0); git diff -U0 -- .github/workflows/ci.yml (added lines only)"
        status: pass
      - kind: other
        ref: "Non-vacuity: actionlint against a scratch copy with a deliberate unknown-top-level-key error, observed exit 1 (see this SUMMARY's Non-Vacuity section)"
        status: pass
    human_judgment: false

duration: 15min
completed: 2026-07-28
status: complete
---

# Phase 9 Plan 3: PR-title conventional-commit gate + actionlint static check Summary

**Shipped release-please's input-quality gate as a dedicated `pr-title.yml` workflow (recorded divergence from D-08's literal "add to ci.yml" wording) plus a new `actionlint` job in `ci.yml` covering all 5 workflow files — both guards proven non-vacuous by executing their shipped shell/binary against a rejecting input, not by reading source.**

## Performance

- **Duration:** ~15 min
- **Tasks:** 3
- **Files modified:** 3 (2 new, 1 modified)

## Accomplishments
- `internal/upgrade/pr_title_lint_test.go`: `TestPRTitleLintAcceptsAndRejects`, 17 subtests (4 reject, 13 accept including one adversarial injection-safety row) executing the lint step's own extracted shell against each title, never reading the regex as text.
- `.github/workflows/pr-title.yml`: new dedicated workflow (D-08), `pull_request: types: [opened, edited, synchronize, reopened]`, single hand-written `grep -E` step, title bound only via `env: TITLE:`, zero `uses:` actions.
- `.github/workflows/ci.yml`: new `actionlint` job (one added job, zero lines touched in any existing job/trigger/permissions block) running `go install github.com/rhysd/actionlint/cmd/actionlint@v1.7.12` then linting all 5 workflow files.
- Both new guards proven non-vacuous: the PR-title lint observed rejecting a real bad title with a diagnosable annotation and accepting a real good one; actionlint observed failing on a deliberately-broken scratch copy of a workflow file.
- `release.yml`, `internal/upgrade/verify.go`, and `.goreleaser.yaml` confirmed byte-identical throughout (`git diff` empty at every checkpoint).

## Task Commits

1. **Task 1: RED — table-driven test executing the PR-title lint's own shell** - `4dfd1df` (test)
2. **Task 2: GREEN — .github/workflows/pr-title.yml (D-08)** - `4c2cc37` (feat)
3. **Task 3: actionlint static gate job in ci.yml** - `04cc0ec` (feat)

**Plan metadata:** (this commit, docs)

## Files Created/Modified
- `internal/upgrade/pr_title_lint_test.go` - `TestPRTitleLintAcceptsAndRejects` + `runPRTitleLintStep` helper (Task 1)
- `.github/workflows/pr-title.yml` - PR-title conventional-commit lint workflow (Task 2)
- `.github/workflows/ci.yml` - new `actionlint` job appended (Task 3)

## Decisions Made
- **Recorded divergence honored (see plan's `<recorded_divergence>` block, restated in `pr-title.yml`'s own header comment):** D-08 says "add a lightweight PR-title conventional-commit check to `ci.yml`." What shipped is the same hand-written check in a dedicated `.github/workflows/pr-title.yml` with its own narrowly-scoped `pull_request: types: [opened, edited, synchronize, reopened]` trigger. Reason: `ci.yml`'s shared `pull_request:` trigger uses GitHub's default event types, which exclude `edited` — a contributor fixing only a bad title would get no re-run there. Widening `ci.yml`'s trigger would re-run the heavy `test`/`govulncheck`/`reproducibility`/`perf-regression` jobs on every title *or description* edit. What's preserved: D-08's substance in full (blocking gate, hand-written `grep -E`, `env:` indirection). What diverges: only the file the job lives in — recorded here and in `pr-title.yml`'s header comment, per this repo's documented-divergence-over-silent-drift convention (`docs/RELEASE-PROCEDURES.md` §5 style). `docs/RELEASE-PROCEDURES.md` §10 itself is plan 09-04's deliverable, not this plan's.
- Chose the reject-row/accept-row table shape reusing 09-02's stubbed-shell-execution idiom (`mustWorkflowStepRunBlock` + `exec.Command("bash", ...)`), simplified since no `PATH` stub was needed — the title is supplied purely via one environment variable, not routed through a fake CLI. No third testing idiom invented.
- Re-resolved `rhysd/actionlint`'s latest release live via `gh api repos/rhysd/actionlint/releases/latest --jq .tag_name` before writing the pinned version into `ci.yml`: confirmed `v1.7.12`, matching the plan's cached value exactly — no delta to record.
- Did not run `requirements mark-complete REL-02` — REL-02 is a spanning requirement across plans 09-01..09-08 (per 09-01's SUMMARY, which explicitly reverted a premature completion mark). `.planning/REQUIREMENTS.md` REL-02 stays `[ ]` In Progress; it should only flip to Complete on the plan that actually cuts the live tag and proves the signed-artifact SAN (09-07 or 09-08 per the phase's coverage table).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Header comment's own prose tripped the zero-`uses:` acceptance criterion**
- **Found during:** Task 2, running the acceptance-criteria check `grep -c 'uses:' .github/workflows/pr-title.yml` (expected 0)
- **Issue:** The workflow's own explanatory header comment contained the literal substring `` `uses:` `` inside a sentence ("This workflow adds no third-party `uses:` action at all") — `grep -c` counted that comment line as a match, returning 1 instead of the required 0, even though the workflow adds no actual `uses:` step.
- **Fix:** Reworded the sentence to "This workflow adds no third-party Action reference at all" — same meaning, no longer contains the literal `uses:` token.
- **Files modified:** `.github/workflows/pr-title.yml`
- **Verification:** Re-ran `grep -c 'uses:' .github/workflows/pr-title.yml` → `0` (grep's own exit 1 for zero matches, as expected). `actionlint .github/workflows/pr-title.yml` still exits 0; `go test ./internal/upgrade/ -run TestPRTitleLintAcceptsAndRejects` still all-PASS.
- **Committed in:** `4c2cc37` (Task 2 commit — the fix landed before the commit, not as a follow-up)

---

**Total deviations:** 1 auto-fixed (a self-inflicted acceptance-criterion trip from the workflow's own prose, not a functional defect)
**Impact on plan:** No scope creep; the fix was required for the plan's own stated acceptance criterion to hold.

## Non-Vacuity: Observed Break-Observe-Restore / Direct-Execution Output (mandatory, recorded verbatim)

### Guard 1 — PR-title lint (Task 2 acceptance criteria)

Reject path — non-conformant title, invoked directly against the shipped shell:
```
$ TITLE='update some stuff' bash <extracted-script>
::error::PR title is not Conventional-Commits-shaped: update some stuff
exit: 1
```

Accept path — conformant title:
```
$ TITLE='feat: add new indexer optimization' bash <extracted-script>
exit: 0
```
(nothing printed to stdout or stderr)

Injection-safety — adversarial title containing `$(...)`, backticks, and `; rm -rf /`, re-confirmed manually outside the Go test:
```
$ TITLE='feat: totally normal subject $(touch /tmp/.../INJECTED) `touch /tmp/.../INJECTED` ; rm -rf / #' bash <extracted-script>
exit: 0
no side effect: /tmp/.../INJECTED absent
```
The title crossed into the step as data (via the `TITLE` env var) and was never evaluated as shell text — the marker file was never created despite the embedded command substitution and backtick syntax.

### Guard 2 — `actionlint` job (Task 3 acceptance criteria)

Introduced a deliberate schema error (an unknown top-level key) into a scratch copy of `ci.yml` (outside the repo tree, `mktemp -d`-based, never staged):
```
$ actionlint /tmp/.../scratch-ci.yml
/tmp/.../scratch-ci.yml:306:1: unexpected key "bogus_unknown_top_level_key" for "workflow" section.
expected one of "concurrency", "defaults", "env", "jobs", "name", "on", "permissions", "run-name" [syntax-check]
    |
306 | bogus_unknown_top_level_key: true
    | ^~~~~~~~~~~~~~~~~~~~~~~~~~~~
observed exit: 1
```
Scratch copy deleted immediately after; `git status --porcelain .github/workflows/` showed only the plan's own intended files (empty at the time of the check, before this task's real edit was staged).

Post-edit companion checks, all green:
```
$ grep -c 'actionlint@v1\.' .github/workflows/ci.yml
1
$ actionlint .github/workflows/*.yml
(exit 0, no output)
$ go test ./internal/upgrade/ -count=1
ok  	github.com/seanb4t/codegraph-go/internal/upgrade	0.597s
$ git diff -- .github/workflows/release.yml internal/upgrade/verify.go .goreleaser.yaml
(empty)
```

## Issues Encountered
None beyond the self-inflicted `uses:` prose collision documented above.

## User Setup Required
None for this plan.

## Next Phase Readiness
- release-please's input-quality gate (PR-title conventional-commit lint) and the workflow-wide static-analysis gate (actionlint) are both live and proven non-vacuous.
- `release.yml`, `internal/upgrade/verify.go`, and `.goreleaser.yaml` remain byte-unchanged — the LOCKED contract (D-01) is untouched.
- `docs/RELEASE-PROCEDURES.md` §10 (documenting this plan's recorded divergence in the runbook itself) is plan 09-04's deliverable, not landed here — flagging for 09-04 to pick up.
- No blockers for 09-04 onward. 09-05's GitHub App provisioning remains the phase's one hard external dependency before a real App-token-authored tag push can exercise the release-please spine in production.

## Self-Check: PASSED

All 3 files confirmed present/modified on disk (`internal/upgrade/pr_title_lint_test.go`, `.github/workflows/pr-title.yml`, `.github/workflows/ci.yml`); all 3 commit hashes (`4dfd1df`, `4c2cc37`, `04cc0ec`) confirmed present in `git log --oneline --all`.

## Self-Check: PASSED (re-verified)

All 3 created/modified files confirmed present on disk; all 3 task commit hashes confirmed present in `git log --oneline --all`.

---
*Phase: 09-release-please-and-goreleaser*
*Completed: 2026-07-28*
