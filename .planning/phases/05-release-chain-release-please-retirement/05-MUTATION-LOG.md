# 05-MUTATION-LOG — Release Chain & release-please Retirement

**Phase:** 05-release-chain-release-please-retirement
**Date:** 2026-10-14
**Scope:** RED/GREEN transcripts and mutation families for the release chain, organized by plan:
- 05-01: the release-PR half (tracer RED and GREEN, Families 05-01/a-h);
- 05-02, 05-03, 05-04, 05-05, 05-07: appended by those plans.

Every Go transcript below is a `go test` run. RED evidence is the `test(05-01):` commit that landed
before its implementation plus the pasted `--- FAIL:` output (rule x1cjy9vyhq);
`gsd-tools check tdd-red-evidence` is not used (it parses TAP only and always reports INVALID_RED
for `go test`).

## Pre-mutation cleanliness gate — the convention this log follows

Every mutation runs in a detached scratch worktree whose path contains `p05-scratch-wt`
(`S="$(mktemp -d)/p05-scratch-wt"`, `git worktree add -q --detach "$S" HEAD`); the main checkout is
never mutated. Before applying a mutation, `git -C "$S" diff --quiet -- <file>` is asserted to exit
0, proving no pre-existing edit is overwritten. After applying, `git -C "$S" diff --numstat` plus
the `-U0` hunk are recorded as the applied-confirmation (a mutation that did not land would make a
RED meaningless, rule 84d1gfpywd). The revert is `git -C "$S" checkout -- <file>`, followed by a
"Byte-clean proof" line: `git -C "$S" diff --quiet` exit 0. The scratch worktree is removed with
`git worktree remove --force "$S"` at the end of the run.

## 05-01 RED-first transcript

Commit: the `test(05-01):` commit, before `release-pr.yml`, `release:batch` and the newlines keys
exist. Command: `GOTOOLCHAIN=go1.26.9 go test ./internal/upgrade/ -count=1 -v`. Every failure is an
assertion or read failure on the missing subject; there is no build error.

```
=== RUN   TestReleasePRWorkflowShape
    changie_release_shape_test.go:232: read ../../.github/workflows/release-pr.yml: open ../../.github/workflows/release-pr.yml: no such file or directory
--- FAIL: TestReleasePRWorkflowShape (0.00s)
=== RUN   TestReleaseBatchTargetShape
    changie_release_shape_test.go:439: ../../Taskfile.yml: no top-level "release:batch" task
--- FAIL: TestReleaseBatchTargetShape (0.00s)
=== RUN   TestChangieConfigShape
    changie_shape_test.go:295: ../../.changie.yaml: newlines key set = [afterChangelogHeader], want exactly [afterChangelogHeader afterKind beforeKind endOfVersion] (D-13, Phase 5 05-01, amends Phase 1 D-02: beforeKind, afterKind and endOfVersion make a batched version file end in a blank line, so changie merge never glues a `## [` heading onto the last bullet — RESEARCH Pitfall 1)
--- FAIL: TestChangieConfigShape (0.00s)
=== RUN   TestWorkflowRunBodiesInvokeTask
    taskfile_shape_test.go:1398: read ../../.github/workflows/release-pr.yml: open ../../.github/workflows/release-pr.yml: no such file or directory
--- FAIL: TestWorkflowRunBodiesInvokeTask (0.00s)
=== RUN   TestWorkflowFilePopulationMatchesDisk
    taskfile_shape_test.go:1611: inScopeWorkflowFiles names "release-pr.yml", which no longer exists on disk
--- FAIL: TestWorkflowFilePopulationMatchesDisk (0.00s)
=== RUN   TestInScopeJobsPopulationMatchesDisk
    taskfile_shape_test.go:1684: read ../../.github/workflows/release-pr.yml: open ../../.github/workflows/release-pr.yml: no such file or directory
--- FAIL: TestInScopeJobsPopulationMatchesDisk (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/upgrade	0.508s
```

Exactly these six tests fail; every other test in the package passes.

Command: `task check:changie` (exit 201, stops at leg 14). Legs 1-13 print ok; leg 14 reports (i) and
(ii) and holds (iii) and (iv), as measured at planning time:

```
check:changie: found 15 seeded version files under .changes/ (floor 14)
check:changie: [1/14] changie latest = v0.14.1 (newest CHANGELOG.md heading v0.14.1; baseline v0.14.0) — ok
check:changie: [2/14] scratch unreleased/ holds 0 fragments before next auto
check:changie: [2/14] changie next auto (empty unreleased/) refused: no unreleased changes found for automatic bumping — ok
check:changie: [3/14] changie merge --dry-run byte-identical to CHANGELOG.md (dry run wrote nothing) — ok
check:changie: [4/14] changie new -k Fixes writes exactly one fragment with kind/body set — ok
check:changie: [5/14] changie new with no PR accepted: fragment written without a PR field — ok
check:changie: [6/14] changie new -k Undeclared refused: invalid kind: Undeclared — ok
check:changie: [7/14] changie new -m PR=0 refused: input below minimum: 0 < 1 — ok
check:changie: [8/14] changie new -m PR=abc refused: invalid number — ok
check:changie: [9/14] two back-to-back new -k Fixes calls raised the fragment count to 4 (sub-second fragmentFileFormat holds) — ok
check:changie: [10/14] with only Fixes fragments present, next auto = v0.14.2 (patch successor of v0.14.1) — ok
check:changie: [11/14] a Breaking fragment derives the minor successor v0.15.0 (pre-1.0: Breaking -> minor) — ok
check:changie: [12/14] changie batch auto --dry-run renders the no-PR fragment without a link and a PR fragment with its link (dry run wrote nothing) — ok
check:changie: [13/14] task changie -- passed a body carrying command substitution, backticks and ; byte-literal, and nothing executed — ok
::error::check:changie: [14/14] (i) .changes/v0.15.0.md does not end in two newlines (a blank line), so the next merged heading would be glued onto its last bullet
::error::check:changie: [14/14] (ii) CHANGELOG.md holds a '## [' heading that is not at the start of its line: 10:- check:changie collision probe two ([#3](https://github.com/seanb4t/codegraph-go/pull/3))## [0.14.1](https://github.com/seanb4t/codegraph-go/compare/v0.14.0...v0.14.1) (2026-10-13)
task: Failed to run task "check:changie": exit status 1
```

## 05-01 Tracer RED (end to end)

At the `test(05-01):` commit (`219303c7`), in a detached scratch worktree of HEAD (path contains
`p05-scratch-wt`), with today's `.changie.yaml` (only `afterChangelogHeader: 1`) and the real Phase 3
Breaking fragment: `task changie -- batch auto` (exit 0), `task changie -- merge` (exit 0). The merged
CHANGELOG.md glues the next heading onto the last bullet. Command:
`rg -n '## \[' CHANGELOG.md | rg -v ':## \['` prints the glued line (line 5, wrapped here for width):

```
5:- Removed the hidden `query` and `unlock` commands, so running either now fails as an unknown command; run `search --full <term>` or `daemon unlock [path]` instead. ([#88](https://github.com/seanb4t/codegraph-go/pull/88))## [0.14.1](https://github.com/seanb4t/codegraph-go/compare/v0.14.0...v0.14.1) (2026-10-13)
```

The head of the batched CHANGELOG.md shows the same config also omits the blank lines after the
version heading and around the kind heading:

```
# Changelog

## [v0.15.0](https://github.com/seanb4t/codegraph-go/releases/tag/v0.15.0) — 2026-10-14
### Breaking
- Removed the hidden `query` and `unlock` commands, ...
```

Then, in the same worktree, `GOTOOLCHAIN=go1.26.9 go test ./internal/upgrade/ -run '^TestChangieVersionSeedsMatchChangelog$' -count=1 -v`:

```
=== RUN   TestChangieVersionSeedsMatchChangelog
    changie_shape_test.go:366: ../../.changes: 16 seed files; ../../CHANGELOG.md: 15 version headings
    changie_shape_test.go:399: ../../.changes vs ../../CHANGELOG.md: set mismatch — versions only in seeds: [v0.14.1], versions only in changelog headings: []
--- FAIL: TestChangieVersionSeedsMatchChangelog (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/upgrade	0.194s
```

This is the release PR's required `test` job going red: the glued `## [0.14.1]` heading is lost from the
heading set, so a release PR built with today's config could not merge (RESEARCH Pitfall 1).
Teardown: `git worktree remove --force "$S"` and `git worktree prune`; `git worktree list` shows no
`p05-scratch-wt` entry.

## 05-01 Tracer GREEN (end to end)

At the `ci(05-01):` commit (`106e6cd6`) plus the `test(05-01):` pin-order fix (`c4a2c122`). The
real Phase 3 Breaking fragment is batched only inside a disposable detached worktree of HEAD
(path contains `p05-scratch-wt`); the real tree's CHANGELOG.md and `.changes/` are untouched.

Step 1, `task changie -- next auto` before batching prints `v0.15.0`. Step 2, in the worktree,
`GITHUB_OUTPUT=<file> task release:batch` exits 0 and prints:

```
release:batch: rendered v0.15.0 from 1 fragment(s)
```

The GITHUB_OUTPUT file holds exactly `version=v0.15.0` (equal to the `next auto` value), and
`.changes/v0.15.0.md` exists and is non-empty. Its head:

```

## [v0.15.0](https://github.com/seanb4t/codegraph-go/releases/tag/v0.15.0) — 2026-10-14

### Breaking

- Removed the hidden `query` and `unlock` commands, so running either now fails as an unknown command; run `search --full <term>` or `daemon unlock [path]` instead. ([#88](https://github.com/seanb4t/codegraph-go/pull/88))
```

`git status --short` in the worktree: ` D .changes/unreleased/Breaking-20260926-165849.599611000.yaml`,
` M CHANGELOG.md`, `?? .changes/v0.15.0.md`. Step 3, in the same worktree,
`GOTOOLCHAIN=go1.26.9 go test ./internal/upgrade/ -run '^TestChangie' -count=1 -v`:

```
--- PASS: TestChangieConfigShape (0.00s)
--- PASS: TestChangieBaselineLayout (0.00s)
--- PASS: TestChangieVersionSeedsMatchChangelog (0.00s)
--- PASS: TestChangieToolPinnedInIsolatedModfile (0.00s)
--- PASS: TestChangieWrapperTaskRecordsInstallPath (0.00s)
--- PASS: TestChangieShapeParsersFailLoudly (0.00s)
--- PASS: TestChangieCheckWiredIntoCI (0.00s)
--- PASS: TestChangieBinaryInToolVulnScan (0.00s)
ok  	github.com/seanb4t/codegraph-go/internal/upgrade	0.188s
```

Step 4, `task check:changie` in the batched worktree (its scratch copy now batches from v0.15.0, so leg
14 renders v0.16.0):

```
check:changie: [14/14] changie batch auto then merge renders v0.16.0 with a trailing blank line, every heading at line start, kinds in .changie.yaml order, and history byte-identical — ok
check:changie: 14 of 14 checks passed against a scratch copy (source tree byte-unchanged)
```

So the release PR's content passes the required `test` job's changie guards. The same six named
tests in the main checkout (command from the plan's first verify block):

```
--- PASS: TestReleasePRWorkflowShape (0.00s)
--- PASS: TestReleaseBatchTargetShape (0.00s)
--- PASS: TestChangieConfigShape (0.00s)
--- PASS: TestWorkflowRunBodiesInvokeTask (0.00s)
--- PASS: TestWorkflowFilePopulationMatchesDisk (0.00s)
--- PASS: TestInScopeJobsPopulationMatchesDisk (0.00s)
```

`task check:changie` in the main checkout prints `check:changie: 14 of 14 checks passed against a
scratch copy (source tree byte-unchanged)`, and `task lint:actions` exits 0.

### Empty set (REL-10 edge, empty and adjacency)

In a fresh detached worktree of HEAD, `rm -f .changes/unreleased/*.yaml` leaves only `.gitkeep`
(`ls -A .changes/unreleased` prints `.gitkeep`; `.gitkeep` never counts, the count matches only
`*.yaml`). `GITHUB_OUTPUT=<file> task release:batch` exits 0 and prints:

```
release:batch: no unreleased fragments under .changes/unreleased/ — nothing to release
```

The GITHUB_OUTPUT file is 0 bytes, and `git status --porcelain -- CHANGELOG.md ':(glob).changes/v*.md'`
prints nothing. The step from zero fragments to one is the step from this no-op to the batch above.
Teardown: worktree removed and pruned; `git worktree list` shows no `p05-scratch-wt` entry.
