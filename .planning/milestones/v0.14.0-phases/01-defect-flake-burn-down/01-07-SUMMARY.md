---
phase: 01-defect-flake-burn-down
plan: 07
subsystem: ci
tags: [github-actions, pull_request_target, heredoc-injection, openssl, shell-harness, security]

# Dependency graph
requires:
  - phase: 01-01
    provides: "Taskfile.yml's check:no-force-layout -> check:graph-console region — this plan inserts check:workflow-output-delimiter between them, so 01-01's Taskfile.yml edit had to land first to avoid two plans editing the same lines"
provides:
  - "scripts/check-workflow-output-delimiter.sh — repo-local, self-tested harness that extracts and exercises both pull_request_target workflows' 'Collect changed files' run: bodies against a fork-authored attack payload"
  - "task check:workflow-output-delimiter Taskfile target (--self-test then the real run)"
  - "Per-run 128-bit $GITHUB_OUTPUT heredoc delimiter in require-issue-link.yml and pr-template-format.yml, closing GH #15 (FIX-10)"
affects: []

# Actuals (#2632)
actuals:
  tokens: 4378
  tasks: 2
  commits: 2
  plan_head_before: ab7363e617fef24d9f2d548ec249285b44e703bc

tech-stack:
  added: []
  patterns:
    - "Repo-local shell harness for testing pull_request_target workflow shell logic outside an actual Actions run: extract the run: body straight from the YAML with awk (no YAML parser dependency), execute it against a stub CLI placed first on PATH, assert with diff against an expected-output file. Follows scripts/inject-cosign-key.sh's script contract shape (shebang + purpose paragraph + Usage: block, set -euo pipefail, ::error::-prefixed diagnostics, mktemp -d + trap cleanup) and check:no-force-layout's --self-test-then-real Taskfile ordering."
    - "PRFILES_$(openssl rand -hex 16) as a per-run $GITHUB_OUTPUT heredoc delimiter is the repeatable pattern for any future pull_request_target workflow that writes fork-controlled content through a multi-line output — a fixed literal delimiter is guessable because the workflow file is public, regardless of length."

key-files:
  created:
    - scripts/check-workflow-output-delimiter.sh
  modified:
    - .github/workflows/require-issue-link.yml
    - .github/workflows/pr-template-format.yml
    - Taskfile.yml

key-decisions:
  - "Followed the plan's explicit correction to 01-RESEARCH.md/01-PATTERNS.md's shared-script-extraction recommendation: generated the delimiter INLINE in both workflows instead of extracting scripts/write-multiline-output.sh. require-issue-link.yml deliberately performs no checkout under pull_request_target (recorded as a security property in its own header comment); a shared script would not be reachable from that job without adding one, trading a narrow injection defect for a broader one."
  - "Built the $GITHUB_OUTPUT comparator with `diff` against a written expected-payload file rather than bash array iteration, after the first draft hit a documented bash < 4.4 'unbound variable' bug on `${arr[@]}` expansion of an empty array under `set -euo pipefail` — reproduced on this machine's default /bin/bash 3.2 (macOS), which is also `env bash`'s resolution given this session's PATH. Recorded as a portability note in the script's own header so a future editor does not reintroduce array iteration and silently break on macOS dev machines even though CI's ubuntu-latest bash is newer."
  - "The --self-test positive control embeds its own independent, locally-constructed copy of the OLD fixed-delimiter form (a heredoc literal inside the script, not extracted from the workflow files). This keeps the regression test for 'can this harness still detect the defect' working permanently — including after the fix lands and the vulnerable form no longer exists anywhere in the repo to extract from."

requirements-completed: [FIX-10]

coverage:
  - id: D1
    description: "A repo-local harness extracts and exercises the actual shell block both pull_request_target workflows ship, against a payload containing a path named after the old fixed heredoc delimiter, and fails (RED) against the pre-fix workflows, naming both files in its failure output"
    requirement: FIX-10
    verification:
      - kind: other
        ref: "bash scripts/check-workflow-output-delimiter.sh (RED transcript captured pre-fix — both require-issue-link.yml and pr-template-format.yml FAIL)"
        status: pass
    human_judgment: false
  - id: D2
    description: "The harness's --self-test positive control proves it can still detect a local copy of the old fixed-delimiter form corrupting the payload, and that two invocations of the delimiter-generating expression differ"
    requirement: FIX-10
    verification:
      - kind: other
        ref: "bash scripts/check-workflow-output-delimiter.sh --self-test"
        status: pass
    human_judgment: false
  - id: D3
    description: "Both require-issue-link.yml and pr-template-format.yml derive a per-run 128-bit delimiter (openssl rand -hex 16) for their $GITHUB_OUTPUT heredoc; the harness (including --self-test) passes against both fixed files (GREEN)"
    requirement: FIX-10
    verification:
      - kind: other
        ref: "bash scripts/check-workflow-output-delimiter.sh --self-test && bash scripts/check-workflow-output-delimiter.sh (GREEN transcript, both PASS)"
        status: pass
    human_judgment: false
  - id: D4
    description: "No checkout added to require-issue-link.yml, neither permissions: block changed, task check:workflow-output-delimiter exists and is not referenced by any workflow file, and the workflow-shape guards in internal/upgrade stay green"
    requirement: FIX-10
    verification:
      - kind: unit
        ref: "go test ./internal/upgrade/... -count=1 (ok, 0.666s); rg checks for actions/checkout presence/absence, permissions: line-set, and absence of check-workflow-output-delimiter references under .github/workflows/"
        status: pass
    human_judgment: false

duration: ~20min
completed: 2026-09-15
status: complete
---

# Phase 1 Plan 7: Per-Run $GITHUB_OUTPUT Delimiter in Both pull_request_target Workflows Summary

**Both `require-issue-link.yml` and `pr-template-format.yml` now derive `PRFILES_$(openssl rand -hex 16)` per run instead of a fixed `PRFILES_EOF` heredoc delimiter, pinned by a new self-tested shell harness that extracts and exercises the actual shipped shell blocks — closing GH #15/FIX-10 without adding a checkout or touching either workflow's `permissions:` block.**

## Performance

- **Duration:** ~20 min
- **Tasks:** 2
- **Files modified:** 4 (1 new script, 2 workflow files, 1 Taskfile edit)

## Accomplishments

- Wrote `scripts/check-workflow-output-delimiter.sh`: extracts the live `run:` body of each workflow's "Collect changed files" step directly from the YAML at run time (awk, no YAML parser dependency), executes it against a stub `gh` emitting a five-line fork-authored payload — including a path named exactly `PRFILES_EOF` (the old delimiter), one with a space, one with a single quote — and parses the resulting `$GITHUB_OUTPUT` the way GitHub Actions does, asserting an exact round trip.
- Watched the harness fail (RED) against the current, unmodified workflows: both `require-issue-link.yml` and `pr-template-format.yml` corrupt the payload (the value truncates to 0 lines instead of 5), named individually in the failure output. Transcript captured before any workflow file was touched (rule `84d1gfpywd`); confirmed via `git show --format= --name-only HEAD` that Task 1's commit touched no `.github/workflows/` file.
- Added `--self-test`: a positive control that runs an independently-embedded copy of the OLD fixed-delimiter form and asserts it still corrupts the same payload, plus a concurrency-edge check that two invocations of `PRFILES_$(openssl rand -hex 16)` in immediate succession differ.
- Added `task check:workflow-output-delimiter` (Taskfile target, modelled on `check:no-force-layout`'s precondition/cmds shape) running `--self-test` first, then the real check. Confirmed via `task check:workflow-output-delimiter` that the CLI wiring itself works, and that no file under `.github/workflows/` references the script.
- Fixed both workflows: each now computes `DELIM="PRFILES_$(openssl rand -hex 16)"` once at the top of its `Collect changed files` step and uses it as both heredoc opener and terminator, with an identical explanatory comment in each file pointing at the sibling workflow and at `task check:workflow-output-delimiter`. `require-issue-link.yml` still performs no checkout; `pr-template-format.yml` still checks out the base branch; neither `permissions:` block changed (`contents: read`, `issues: write`, `pull-requests: read`, no `contents: write`, confirmed in both files). The harness (including `--self-test`) now passes (GREEN) against both fixed files.
- `go test ./internal/upgrade/... -count=1` stays green — `workflowFileExceptions`' reason strings for both files ("gh CLI/echo/policy checks over changed paths, not a Taskfile-target duplicate" for `require-issue-link.yml`; "invoke a repo policy script ... and git plumbing, not a Taskfile-target duplicate" for `pr-template-format.yml`) still accurately describe both steps after adding the `openssl rand -hex 16` line — neither step became a Taskfile-target duplicate, so no reason string needed updating.

## Task Commits

1. **Task 1 (RED): a harness that corrupts the shipped block with a fork-named path** - `d6920f98` (test)
2. **Task 2 (GREEN): per-run delimiter inline in both workflows** - `fb697292` (fix)

**Plan metadata:** _pending — added in the final docs commit_

## Files Created/Modified

- `scripts/check-workflow-output-delimiter.sh` - new harness: extracts + exercises both workflows' `Collect changed files` blocks against a fork-authored payload; `--self-test` positive control
- `Taskfile.yml` - new `check:workflow-output-delimiter` target (`--self-test` then the real run), inserted between `check:no-force-layout` and `check:graph-console`
- `.github/workflows/require-issue-link.yml` - `Collect changed files` step now derives a per-run `DELIM` via `openssl rand -hex 16` instead of the fixed `PRFILES_EOF` literal
- `.github/workflows/pr-template-format.yml` - same fix, same shape, own output key (`files`)

## Decisions Made

See `key-decisions` in frontmatter: inline delimiter generation (not the researched shared-script extraction) per the plan's own correction; `diff`-based comparison instead of bash array iteration after a bash 3.2 portability bug; the `--self-test` positive control's old-form copy is embedded independently of the workflow files so it keeps working after the fix lands.

## Deviations from Plan

None — plan executed exactly as written. Both tasks' `<verify>` gates and `<acceptance_criteria>` were run and passed as specified.

## Issues Encountered

- **First draft of the comparator hit a bash 3.2 array-expansion bug.** The initial implementation stored the parsed `$GITHUB_OUTPUT` value in a bash array (`actual=()`) and iterated it with `"${actual[@]}"` for comparison and error reporting. Under `set -euo pipefail`, expanding `${actual[@]}` on a genuinely empty array raises "unbound variable" in bash versions before 4.4 — and this machine's default `/bin/bash` (also what `env bash` resolves to, given this session's `PATH`) is 3.2.57, the stock macOS shell. The bug was caught during the plan's own RED-demonstration run (piped through `tee`, which surfaced it deterministically) before any commit — rewrote the comparator to write the expected payload to a scratch file once and `diff` it against the parsed value file, which needs no array expansion at all. Documented as a portability note in the script's own header comment so a future editor does not reintroduce array iteration; the harness has since been re-verified RED (pre-fix) and GREEN (post-fix) with the diff-based implementation, which is what actually shipped.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- FIX-10 is closed: neither `pull_request_target` workflow's multi-line `$GITHUB_OUTPUT` write is terminable by a fork-controlled file path, and the fix is pinned by a repeatable, self-tested, repo-local gate.
- `task check:workflow-output-delimiter` exists but is deliberately not wired into any workflow — CI wiring is Phase 2's scope per the plan's own file boundary.
- No blockers for the remaining Phase 1 plans.

## Self-Check: PASSED

- FOUND: `scripts/check-workflow-output-delimiter.sh` (executable)
- FOUND: `task check:workflow-output-delimiter` target in `Taskfile.yml`
- FOUND: commit `d6920f98` (Task 1)
- FOUND: commit `fb697292` (Task 2)
- Re-ran plan-level `<verification>`: harness RED confirmed pre-fix (both workflows named in failure output, self-test PASS); harness GREEN confirmed post-fix (`--self-test` and real run both PASS for both files); `rg -c 'openssl rand -hex 16'` = 1 for each workflow; no `<<PRFILES_EOF` literal remains in either workflow file; `require-issue-link.yml` has no `actions/checkout`, `pr-template-format.yml` still does; both `permissions:` blocks unchanged (`contents: read`, `issues: write`, `pull-requests: read`, no `contents: write`); `go test ./internal/upgrade/... -count=1` → `ok` (0.666s); commit messages contain no `[ci skip]`/`[skip ci]` marker; `git status --short` clean after both commits.
- `git rev-list --count ab7363e..HEAD` = 2, matching `actuals.commits: 2`.

---
*Phase: 01-defect-flake-burn-down*
*Completed: 2026-09-15*
