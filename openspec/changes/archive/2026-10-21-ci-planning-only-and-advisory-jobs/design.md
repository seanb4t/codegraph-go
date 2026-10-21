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

`scripts/planning_only.py` lists the changed files and prints `planning_only=true` only when the list is non-empty and every path is under `openspec/`. For a pull request it calls `fragment_policy.fetch_changed_files`, which pages through `pulls/{n}/files` and reports an error when the fetched count differs from the PR's `changed_files`, so the verdict covers the whole PR rather than the last push and a truncated listing is undecided. Every `FileChange` contributes its path and, for a rename, its previous path, so a file moved out of `internal/` into `openspec/` is a code change. For a push it reads `compare/{before}...{after}` and takes `filename` and `previous_filename` from each entry; a listing that reaches the API's 300-file cap is treated as undecided. Any API error, an all-zero `before` (new branch or force push), an empty list or an undecided listing prints `planning_only=false` with the reason on stderr. The script logs the file count and the verdict so the job log shows it did its work.

Alternative: `dorny/paths-filter`. It would add a pinned third-party action and a second Renovate surface for a 40-line rule the repository already knows how to write and unit-test. Rejected.

Alternative: `git diff` in the job. Needs a deeper checkout and separate handling of the PR merge commit; the API gives the file list directly. Rejected.

### D2. Job-level `if` on the heavy jobs, with `!cancelled()`

The six heavy jobs get `needs: [changes]` and `if: ${{ !cancelled() && needs.changes.outputs.planning_only != 'true' }}`. The condition is written against `'true'` so an empty output (classifier failed, was cancelled or never ran) runs the job. `!cancelled()` rather than `always()`, so a cancelled workflow does not start six heavy jobs. The cheap jobs keep no dependency and gain nothing from one.

### D3. The vuln loop moves to `scripts/vuln_report.py`

The build-and-scan loop leaves the Taskfile for a script in the shape of `scripts/fragment_policy.py`: a fixed roster of the nine tools (name, modfile, package), a `Runner` seam for the two commands, and a pure `classify(build_rc, scan_rc) -> Outcome` that maps a failed build to `BUILD ERROR`, scan exit 0 to `CLEAN`, exit 3 to `DETECTED` and any other exit to `SCAN ERROR`. For each tool it builds with `GOWORK=off go build -modfile=<modfile> -o <scratch>/<name> <package>`, scans a built binary with the scanner command the Taskfile passes (`{{.GO_TOOL}} govulncheck -mode=binary`), prints the same `::warning::` lines as today with the command output, and ends with `vuln: built N/9, scanned N, build errors N, detected N, scan errors N`. It exits 0 whatever the scans found and exits 1 only when `built + build errors` is not the roster size or `scanned` is not `built`: a tool never attempted or a built binary never scanned is a defect in the loop, not a vulnerability finding. `scripts/test_vuln_report.py` drives the real loop with canned `Runner` results, so the detection branch and the summary are proven by the repository's own test rather than by reading a log: exit 3 reported as `DETECTED`, a failed build followed by a successful scan of the next tool, exit 2 as `SCAN ERROR`, every tool attempted, and the exact summary counts. The `vuln` target becomes `python3 scripts/vuln_report.py` with the scanner command as an argument, its `desc:` is cut to what the target does and where the roster lives, and `vuln:selftest` stays unchanged as the proof that the real scanner detects. The `ci.yml` step name says that a build failure is a reported outcome too.

Alternative: keep the loop in bash and inject a fake scanner through a Taskfile variable. Testable only by running `task` from a test, which no script test here does. Rejected.

### D4. The rerun hint lives in the runner, not `internal/bench`

The runner wraps the gate's error in `tools/bench/runner/main.go` with one sentence: a no-op change has tripped this gate once on a dispatch-to-dispatch spread larger than the whole budget (#147); rerun the job once before treating the result as a regression, because a real regression reproduces. `internal/bench.CheckRegression` keeps its exact message, so nothing under `internal/` changes and no changelog question arises.

### D5. The script suites run from a job the classifier cannot skip

A new `check:ci-scripts` task runs `scripts/test_planning_only.py` and `scripts/test_vuln_report.py` in the shape of `check:fragment-policy`: stdlib `unittest` over literal inputs, passing only when the executed count equals the number of `def test_` methods in each file, that count meets a floor, and the result line is `OK`, so an undiscovered or skipped test cannot pass. It is a step in the `actionlint` job, which keeps no `needs: [changes]`, so a change to the classifier can never skip its own tests. `check:fragment-policy` stays where it is in the `test` job.

## Risks / Trade-offs

- [The classifier misreads a code change as planning-only] → The rule is a prefix test on every path, with no globs and no exclusions, and the unit test covers mixed lists, an empty list and paths that merely contain `openspec`. A wrong `true` would need the API to return a wrong file list: a short PR listing fails the count check, and a rename counts both of its paths.
- [Every run waits for the classifier before the heavy jobs start] → One small `ubuntu-latest` job with one API call, well under a minute, against eight minutes saved on planning-only runs.
- [A required job that is always skipped would satisfy the ruleset forever] → Only a planning-only change skips, and the condition reads the classifier's explicit `true`. A code change that reaches `main` still runs everything.
- [The vuln summary line hides a regression in the loop itself] → `test_vuln_report.py` fails when the loop misclassifies an exit code or skips a tool, the script exits 1 when the attempted or scanned counts fall short of the roster, and `vuln:selftest` is unchanged and still fails the job when the real scanner stops detecting.
- [A planning-only misclassification hides the classifier's own test failure] → The suites run from `actionlint`, which every run executes.
- [A reader ignores the rerun hint and investigates anyway] → The hint names the issue and the measured spread so the decision is informed, and `BASELINE.md` carries the data.
