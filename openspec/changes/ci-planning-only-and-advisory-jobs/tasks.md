# Tasks

## 1. Planning-only classifier

- [ ] 1.1 Add `scripts/planning_only.py` with a pure `classify(paths) -> bool` and a `main()` that lists PR files through paginated `gh api pulls/{n}/files` or push files through `compare/{before}...{after}`, prints `planning_only=<true|false>` to `$GITHUB_OUTPUT`, logs the file count and verdict, and prints `false` with a stderr reason on any API error, an all-zero `before`, an empty list or a compare listing at the 300-file cap. Verify with `scripts/test_planning_only.py` covering an all-`openspec/` list, a mixed list, an empty list, a path that only contains the word `openspec`, and each fail-open path.
- [ ] 1.2 Add the `changes` job to `.github/workflows/ci.yml` running the script with `GITHUB_REPOSITORY`, `GITHUB_EVENT_NAME`, `PR_NUMBER`, `BEFORE_SHA`, `AFTER_SHA` and `GH_TOKEN` passed through `env:`; verify `task lint` (actionlint) passes and `scripts/check-workflow-output-delimiter.sh` still passes.
- [ ] 1.3 Add `needs: [changes]` and `if: ${{ !cancelled() && needs.changes.outputs.planning_only != 'true' }}` to `test`, `govulncheck`, `tool-vuln`, `reproducibility`, `perf-regression` and `tmux-e2e`, leaving every job name unchanged; verify with `rg -n '^    name:' .github/workflows/ci.yml` before and after that the name list is identical.
- [ ] 1.4 Verify on the PR itself: push a commit that touches only `openspec/` and confirm in the run that `changes` prints `planning_only=true`, the six heavy jobs show as skipped and the PR's required checks are all green; then push a code commit and confirm all jobs ran.

## 2. Advisory vuln job

- [ ] 2.1 Rework the `vuln` target in `Taskfile.yml` so each binary is built inside the report loop under `set +e`, a failed build prints `::warning::<name>: BUILD ERROR` plus the build output and continues, the scan runs only for built binaries, and the task ends with a `vuln: built N/9, scanned N, build errors N, detected N, scan errors N` summary and exit 0. Verify by running `GOMODCACHE=$(mktemp -d) GOPROXY=off task vuln` and confirming nine BUILD ERROR lines, the summary and exit status 0; then run plain `task vuln` and confirm the summary shows nine built.
- [ ] 2.2 Update the target's `desc:` and the `ci.yml` step name and comment so they say a build failure is reported, never fatal; verify `task vuln:selftest` still exits 0.

## 3. Perf gate message and record

- [ ] 3.1 Extend the regression failure in `tools/bench/runner/main.go` with the rerun sentence from design D4; verify with a new `TestRegressionFailureSaysRerunFirst` in `tools/bench/runner/main_test.go` that the wrapped error carries the sentence and the original `bench:` message.
- [ ] 3.2 Add the #147 observation (run 39501339136, 10.2% on a test-only change, green on rerun) to `tools/bench/BASELINE.md` beside the #20 figures, stating that #20 measured within-dispatch spread and #147 between dispatches; verify `task lint` passes.

## 4. Changelog

- [ ] 4.1 No changelog fragment: no file under `cmd/` or `internal/` changes. Put `<!-- changelog-exempt: CI configuration, Taskfile and bench runner only -->` in the PR body and verify the `fragment-required` check passes.
