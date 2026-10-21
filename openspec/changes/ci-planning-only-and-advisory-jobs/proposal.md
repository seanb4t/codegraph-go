# Proposal: Skip the heavy CI jobs on planning-only changes and keep the advisory and perf gates honest

Issue: #135, #147, #148

## Why

Three CI annoyances surfaced on the last two pull requests, and all three live in the same two files. A push that changes only `openspec/` still runs the full test suite and every other heavy job in `ci.yml`, which costs about eight runner-minutes per planning commit and gives the known daemon flakes another chance to fail (#135). The perf regression gate reported a 10.2% throughput regression on a test-only change and passed on rerun, so the next false red will be investigated as a regression unless the failure says otherwise (#147). The advisory tool-vuln job failed before scanning anything because one tool-binary download hit a transient proxy error, so a job that promises to "report, never fail" went red (#148).

## What Changes

- `ci.yml` gets a small first job that classifies the push or pull request as planning-only (every changed file is under `openspec/`) or not. The heavy jobs (`test`, `govulncheck`, `tool-vuln`, `reproducibility`, `perf-regression`, `tmux-e2e`) depend on it and skip through a job-level condition when the change is planning-only. GitHub reports a job skipped by a job-level condition as Success, so every required check keeps reporting. Workflow-level `paths-ignore` stays out, for the reason the issue gives: a workflow that never runs never reports, and the PR stays blocked. The cheap jobs (`actionlint`, `transcript-freeze`, `goreleaser-check`) run as today.
- The classifier fails open. When it cannot decide, it reports "not planning-only" and every heavy job runs; the heavy jobs skip only on an explicit planning-only verdict.
- The bench runner's regression failure tells the reader to rerun the job once before investigating, and `tools/bench/BASELINE.md` records the #147 observation next to the #20 data so the variance history is in one place.
- `task vuln` builds each tool binary inside the report loop: a build failure is reported as `BUILD ERROR` for that binary, distinct from `CLEAN`, `DETECTED` and `SCAN ERROR`, and the remaining binaries are still scanned. The task ends with a summary line that counts built, scanned and failed binaries, and it exits 0 whatever the scans found; it exits 1 only when a built binary was never scanned, which is a defect in the loop rather than a finding. `vuln:selftest` stays as the non-vacuity proof and keeps its right to fail the job.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None. This change touches CI configuration, a developer tool's message and a Taskfile target. No product behavior, CLI output or MCP result changes, so the change sets `skip_specs: true`.

## Out of scope

- Characterizing the perf gate's dispatch-to-dispatch spread and retuning the budget or trial count. That needs a dedicated multi-dispatch study on `bench.yml`, as #20 did; this change records the data point and changes the message.
- Other workflows. `corpora.yml` already uses workflow-level path filters for its golden job, and the issue scopes the change to `ci.yml`.
- Treating files outside `openspec/` (`*.md` at the root, `docs/`) as planning-only. The rule is deliberately narrow so a mistaken classification can only run too much, never too little.
- A retry inside the perf gate. A rerun is a human decision until the spread is characterized.

## Impact

- `.github/workflows/ci.yml`: one new job, `needs:` and `if:` on six jobs. Job names do not change, so the `protect-main` ruleset is untouched.
- `scripts/planning_only.py` and its unit test, in the shape of `scripts/fragment_policy.py`.
- `tools/bench/runner/main.go`: the regression failure message. `tools/bench/BASELINE.md`: one paragraph.
- `Taskfile.yml`: the `vuln` target's build and report loop.
- No changelog fragment. `tools/`, `scripts/` and `Taskfile.yml` are not shipped code under `scripts/fragment_policy.py`, and the PR body carries `<!-- changelog-exempt: CI configuration, Taskfile and bench runner only -->`.
