# Design

## Context

`ci.yml` runs nine jobs on every pull request and every push to `main`. `protect-main` pins its required checks by job name. GitHub reports a check when the job runs or when a job-level condition skips it (the skipped job reports Success); a workflow-level path filter leaves the check pending forever (GitHub docs, "Handling skipped but required checks"). `transcript-freeze` already skips through a job-level `if`, so the pattern is in use here.

The repository's CI scripts (`scripts/fragment_policy.py`, `scripts/pr_template_policy.py`) read untrusted PR data only through the environment, list PR files through paginated `gh api` calls, and fail closed. The classifier follows the same shape but fails open, because the safe failure for a "skip work" gate is to do the work.

`task vuln` builds nine tool binaries under `set -euo pipefail` before its report loop, so the first failed `go build` aborts the task. The report loop already classifies each scan as CLEAN, DETECTED or SCAN ERROR and never fails.

`tools/bench/runner -mode regression` medians three sessions and compares against `tools/bench/baseline.json`. #20 measured a 0.35% session-to-session spread on `ubuntu-latest` plus disk; #147 observed a 10.2% delta between two dispatches on a no-op change. The two numbers do not conflict: #20 measured sessions inside one dispatch, #147 compared two dispatches.

## Goals / Non-Goals

**Goals:**
- A planning-only pull request or push reports every required check green without running the heavy jobs.
- A classifier failure can only cause extra work, never a skipped gate.
- The advisory vuln job stays green on a transient build failure and still says which binary went unscanned.
- The perf gate's first false red is treated as a rerun, not an investigation.

**Non-Goals:**
- Changing which checks are required, or any job's name.
- Reducing the heavy jobs' cost on code changes.
- Making the perf gate retry itself.

## Decisions

### D1. A first-party classifier script, not a path-filter action

`scripts/planning_only.py` lists the changed files and prints `planning_only=true` only when the list is non-empty and every path is under `openspec/`. For a pull request it pages through `pulls/{n}/files`, the same listing `fragment_policy.py` uses, so the verdict covers the whole PR rather than the last push. For a push it reads `compare/{before}...{after}`; a listing that reaches the API's 300-file cap is treated as undecided. Any API error, an all-zero `before` (new branch or force push), an empty list or an undecided listing prints `planning_only=false` with the reason on stderr. The script logs the file count and the verdict so the job log shows it did its work.

Alternative: `dorny/paths-filter`. It would add a pinned third-party action and a second Renovate surface for a 40-line rule the repository already knows how to write and unit-test. Rejected.

Alternative: `git diff` in the job. Needs a deeper checkout and separate handling of the PR merge commit; the API gives the file list directly. Rejected.

### D2. Job-level `if` on the heavy jobs, with `!cancelled()`

The six heavy jobs get `needs: [changes]` and `if: ${{ !cancelled() && needs.changes.outputs.planning_only != 'true' }}`. The condition is written against `'true'` so an empty output (classifier failed, was cancelled or never ran) runs the job. `!cancelled()` rather than `always()`, so a cancelled workflow does not start six heavy jobs. The cheap jobs keep no dependency and gain nothing from one.

### D3. The vuln target builds inside the loop

Each binary is built right before its scan with `set +e` around the build, the way the scan already is. A failed build prints `::warning::<name>: BUILD ERROR` with the `go build` output and moves on. The task ends with `vuln: built N/9, scanned N, build errors N, detected N, scan errors N` and exits 0. The `ci.yml` step name and the target's `desc:` say that a build failure is a reported outcome too.

### D4. The rerun hint lives in the runner, not `internal/bench`

The runner wraps the gate's error in `tools/bench/runner/main.go` with one sentence: a no-op change has tripped this gate once on a dispatch-to-dispatch spread larger than the whole budget (#147); rerun the job once before treating the result as a regression, because a real regression reproduces. `internal/bench.CheckRegression` keeps its exact message, so nothing under `internal/` changes and no changelog question arises.

## Risks / Trade-offs

- [The classifier misreads a code change as planning-only] → The rule is a prefix test on every path, with no globs and no exclusions, and the unit test covers mixed lists, an empty list and paths that merely contain `openspec`. A wrong `true` would need the API to return a wrong file list.
- [Every run waits for the classifier before the heavy jobs start] → One small `ubuntu-latest` job with one API call, well under a minute, against eight minutes saved on planning-only runs.
- [A required job that is always skipped would satisfy the ruleset forever] → Only a planning-only change skips, and the condition reads the classifier's explicit `true`. A code change that reaches `main` still runs everything.
- [The vuln summary line hides a regression in the loop itself] → `vuln:selftest` is unchanged and still fails the job when detection stops firing.
- [A reader ignores the rerun hint and investigates anyway] → The hint names the issue and the measured spread so the decision is informed, and `BASELINE.md` carries the data.
