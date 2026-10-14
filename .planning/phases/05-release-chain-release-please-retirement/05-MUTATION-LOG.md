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

## Families 05-01/a-h — setup

All eight mutations run in one detached scratch worktree of HEAD (`c4c19ade`) whose path contains
`p05-scratch-wt`, created with `git worktree add -q --detach "$S" HEAD`; the main checkout is never
mutated. Each family follows the cleanliness-gate convention above: pre-mutation gate, applied-confirmation
(numstat and `-U0` hunk), the guard run RED, the revert, the byte-clean proof, then the guard GREEN. A
family whose mutation did not change the file aborts with `MUTATION DID NOT LAND`; none did.

## Family 05-01/a — newlines key dropped

Pre-mutation gate: `git -C "$S" diff --quiet -- .changie.yaml` exit 0

Mutation: delete the `endOfVersion: 2` line from .changie.yaml.

Applied-confirmation, `git -C "$S" diff --numstat`:

```
0	1	.changie.yaml
```

Hunk, `git -C "$S" diff -U0 -- .changie.yaml`:

```diff
diff --git a/.changie.yaml b/.changie.yaml
index 99db6edf..b8197fe1 100644
--- a/.changie.yaml
+++ b/.changie.yaml
@@ -64 +63,0 @@ newlines:
-  endOfVersion: 2
```

RED guard: `go test ./internal/upgrade/ -run '^TestChangieConfigShape$' -count=1 -v` (from `$S`):

```
    changie_shape_test.go:295: ../../.changie.yaml: newlines key set = [afterChangelogHeader afterKind beforeKind], want exactly [afterChangelogHeader afterKind beforeKind endOfVersion] (D-13, Phase 5 05-01, amends Phase 1 D-02: beforeKind, afterKind and endOfVersion make a batched version file end in a blank line, so changie merge never glues a `## [` heading onto the last bullet — RESEARCH Pitfall 1)
--- FAIL: TestChangieConfigShape (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/upgrade	0.209s
FAIL
```

RED guard: `task check:changie` (from `$S`):

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
::error::check:changie: [14/14] (ii) CHANGELOG.md holds a '## [' heading that is not at the start of its line: 14:- check:changie collision probe two ([#3](https://github.com/seanb4t/codegraph-go/pull/3))## [0.14.1](https://github.com/seanb4t/codegraph-go/compare/v0.14.0...v0.14.1) (2026-10-13)
task: Failed to run task "check:changie": exit status 1
```

Byte-clean proof: reverted with `git -C "$S" checkout -- .changie.yaml`; `git -C "$S" diff --quiet` exit 0

Re-run GREEN after the revert, `go test ./internal/upgrade/ -run '^TestChangieConfigShape$' -count=1 -v`:

```
--- PASS: TestChangieConfigShape (0.00s)
ok  	github.com/seanb4t/codegraph-go/internal/upgrade	0.170s
```

Re-run GREEN after the revert, `task check:changie` (last line):

```
check:changie: 14 of 14 checks passed against a scratch copy (source tree byte-unchanged)
```

## Family 05-01/b — empty-set no-op removed

Pre-mutation gate: `git -C "$S" diff --quiet -- Taskfile.yml` exit 0

Mutation: delete the `exit 0` line of release:batch's zero-count branch.

Applied-confirmation, `git -C "$S" diff --numstat`:

```
0	1	Taskfile.yml
```

Hunk, `git -C "$S" diff -U0 -- Taskfile.yml`:

```diff
diff --git a/Taskfile.yml b/Taskfile.yml
index 419fddaa..4a59a853 100644
--- a/Taskfile.yml
+++ b/Taskfile.yml
@@ -1022 +1021,0 @@ tasks:
-          exit 0
```

RED guard: `go test ./internal/upgrade/ -run '^TestReleaseBatchTargetShape$' -count=1 -v` (from `$S`):

```
    changie_release_shape_test.go:471: release:batch cmds: token "exit 0" (zero-count exit 0) not found at or after byte offset 112
    changie_release_shape_test.go:528: release:batch: 6 of 7 ordered tokens found; 3 changie invocations; 17 workflow files scanned
--- FAIL: TestReleaseBatchTargetShape (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/upgrade	0.159s
FAIL
```

Byte-clean proof: reverted with `git -C "$S" checkout -- Taskfile.yml`; `git -C "$S" diff --quiet` exit 0

Re-run GREEN after the revert, `go test ./internal/upgrade/ -run '^TestReleaseBatchTargetShape$' -count=1 -v`:

```
--- PASS: TestReleaseBatchTargetShape (0.00s)
ok  	github.com/seanb4t/codegraph-go/internal/upgrade	0.155s
```

## Family 05-01/c — skip-ci scan removed

Pre-mutation gate: `git -C "$S" diff --quiet -- Taskfile.yml` exit 0

Mutation: delete release:batch's skip-ci scan `if` block (the grep line through its `fi`).

Applied-confirmation, `git -C "$S" diff --numstat`:

```
0	4	Taskfile.yml
```

Hunk, `git -C "$S" diff -U0 -- Taskfile.yml`:

```diff
diff --git a/Taskfile.yml b/Taskfile.yml
index 419fddaa..19ea022b 100644
--- a/Taskfile.yml
+++ b/Taskfile.yml
@@ -1027,4 +1026,0 @@ tasks:
-        if grep -qiE '\[(skip ci|ci skip|no ci|skip actions|actions skip)\]' ".changes/${version}.md"; then
-          echo "::error::release:batch: .changes/${version}.md contains a GitHub skip-ci phrase; the release PR body becomes the squash commit message on main and would suppress release-tag.yml"
-          exit 1
-        fi
```

RED guard: `go test ./internal/upgrade/ -run '^TestReleaseBatchTargetShape$' -count=1 -v` (from `$S`):

```
    changie_release_shape_test.go:471: release:batch cmds: token "\\[(skip ci|ci skip|no ci|skip actions|actions skip)\\]" (skip-ci phrase scan) not found at or after byte offset 450
    changie_release_shape_test.go:528: release:batch: 6 of 7 ordered tokens found; 3 changie invocations; 17 workflow files scanned
--- FAIL: TestReleaseBatchTargetShape (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/upgrade	0.210s
FAIL
```

Byte-clean proof: reverted with `git -C "$S" checkout -- Taskfile.yml`; `git -C "$S" diff --quiet` exit 0

Re-run GREEN after the revert, `go test ./internal/upgrade/ -run '^TestReleaseBatchTargetShape$' -count=1 -v`:

```
--- PASS: TestReleaseBatchTargetShape (0.00s)
ok  	github.com/seanb4t/codegraph-go/internal/upgrade	0.149s
```

## Family 05-01/d — PR opened with the default token

Pre-mutation gate: `git -C "$S" diff --quiet -- .github/workflows/release-pr.yml` exit 0

Mutation: replace the create-pull-request `token:` value with the default-token secret expression.

Applied-confirmation, `git -C "$S" diff --numstat`:

```
1	1	.github/workflows/release-pr.yml
```

Hunk, `git -C "$S" diff -U0 -- .github/workflows/release-pr.yml`:

```diff
diff --git a/.github/workflows/release-pr.yml b/.github/workflows/release-pr.yml
index ce0eee02..d398c33f 100644
--- a/.github/workflows/release-pr.yml
+++ b/.github/workflows/release-pr.yml
@@ -93 +93 @@ jobs:
-          token: ${{ steps.app-token.outputs.token }}
+          token: ${{ secrets.GITHUB_TOKEN }}
```

RED guard: `go test ./internal/upgrade/ -run '^TestReleasePRWorkflowShape$' -count=1 -v` (from `$S`):

```
    changie_release_shape_test.go:312: job "release-pr" step 6 (Open or update the release PR): with = map[body-path:.changes/${{ steps.batch.outputs.version }}.md branch:changie/release commit-message:chore(main): release ${{ steps.batch.outputs.version }} draft:false title:chore(main): release ${{ steps.batch.outputs.version }} token:${{ secrets.GITHUB_TOKEN }}], want map[body-path:.changes/${{ steps.batch.outputs.version }}.md branch:changie/release commit-message:chore(main): release ${{ steps.batch.outputs.version }} draft:false title:chore(main): release ${{ steps.batch.outputs.version }} token:${{ steps.app-token.outputs.token }}]
    changie_release_shape_test.go:399: non-comment line 65 names the default token (D-03): "          token: ${{ secrets.GITHUB_TOKEN }}"
    changie_release_shape_test.go:407: ../../.github/workflows/release-pr.yml: 151 fields, lines and uses checked (8 uses: lines, 4 distinct pins, 2 run bodies)
--- FAIL: TestReleasePRWorkflowShape (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/upgrade	0.162s
FAIL
```

Byte-clean proof: reverted with `git -C "$S" checkout -- .github/workflows/release-pr.yml`; `git -C "$S" diff --quiet` exit 0

Re-run GREEN after the revert, `go test ./internal/upgrade/ -run '^TestReleasePRWorkflowShape$' -count=1 -v`:

```
--- PASS: TestReleasePRWorkflowShape (0.00s)
ok  	github.com/seanb4t/codegraph-go/internal/upgrade	0.164s
```

## Family 05-01/e — release PR opened as draft

Pre-mutation gate: `git -C "$S" diff --quiet -- .github/workflows/release-pr.yml` exit 0

Mutation: change `draft: false` to `draft: true`.

Applied-confirmation, `git -C "$S" diff --numstat`:

```
1	1	.github/workflows/release-pr.yml
```

Hunk, `git -C "$S" diff -U0 -- .github/workflows/release-pr.yml`:

```diff
diff --git a/.github/workflows/release-pr.yml b/.github/workflows/release-pr.yml
index ce0eee02..e50b13b8 100644
--- a/.github/workflows/release-pr.yml
+++ b/.github/workflows/release-pr.yml
@@ -98 +98 @@ jobs:
-          draft: false
+          draft: true
```

RED guard: `go test ./internal/upgrade/ -run '^TestReleasePRWorkflowShape$' -count=1 -v` (from `$S`):

```
    changie_release_shape_test.go:312: job "release-pr" step 6 (Open or update the release PR): with = map[body-path:.changes/${{ steps.batch.outputs.version }}.md branch:changie/release commit-message:chore(main): release ${{ steps.batch.outputs.version }} draft:true title:chore(main): release ${{ steps.batch.outputs.version }} token:${{ steps.app-token.outputs.token }}], want map[body-path:.changes/${{ steps.batch.outputs.version }}.md branch:changie/release commit-message:chore(main): release ${{ steps.batch.outputs.version }} draft:false title:chore(main): release ${{ steps.batch.outputs.version }} token:${{ steps.app-token.outputs.token }}]
    changie_release_shape_test.go:407: ../../.github/workflows/release-pr.yml: 151 fields, lines and uses checked (8 uses: lines, 4 distinct pins, 2 run bodies)
--- FAIL: TestReleasePRWorkflowShape (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/upgrade	0.160s
FAIL
```

Byte-clean proof: reverted with `git -C "$S" checkout -- .github/workflows/release-pr.yml`; `git -C "$S" diff --quiet` exit 0

Re-run GREEN after the revert, `go test ./internal/upgrade/ -run '^TestReleasePRWorkflowShape$' -count=1 -v`:

```
--- PASS: TestReleasePRWorkflowShape (0.00s)
ok  	github.com/seanb4t/codegraph-go/internal/upgrade	0.151s
```

## Family 05-01/f — registry entry deleted

Pre-mutation gate: `git -C "$S" diff --quiet -- internal/upgrade/taskfile_shape_test.go` exit 0

Mutation: delete inScopeJobs' `{Workflow: "release-pr.yml", JobID: "release-pr"}` line.

Applied-confirmation, `git -C "$S" diff --numstat`:

```
0	1	internal/upgrade/taskfile_shape_test.go
```

Hunk, `git -C "$S" diff -U0 -- internal/upgrade/taskfile_shape_test.go`:

```diff
diff --git a/internal/upgrade/taskfile_shape_test.go b/internal/upgrade/taskfile_shape_test.go
index 03ecd106..d3c9963d 100644
--- a/internal/upgrade/taskfile_shape_test.go
+++ b/internal/upgrade/taskfile_shape_test.go
@@ -155 +154,0 @@ var inScopeJobs = []inScopeJob{
-	{Workflow: "release-pr.yml", JobID: "release-pr"},
```

RED guard: `go test ./internal/upgrade/ -run '^TestInScopeJobsPopulationMatchesDisk$' -count=1 -v` (from `$S`):

```
    taskfile_shape_test.go:1718: job(s) on disk in [ci.yml release-please.yml corpora.yml components-drift.yml fragment-required.yml release-pr.yml] not present in inScopeJobs and not in usesOnlyJobExceptions: [release-pr.yml/release-pr] — a new CI job was added without registering it here
--- FAIL: TestInScopeJobsPopulationMatchesDisk (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/upgrade	0.181s
FAIL
```

Byte-clean proof: reverted with `git -C "$S" checkout -- internal/upgrade/taskfile_shape_test.go`; `git -C "$S" diff --quiet` exit 0

Re-run GREEN after the revert, `go test ./internal/upgrade/ -run '^TestInScopeJobsPopulationMatchesDisk$' -count=1 -v`:

```
--- PASS: TestInScopeJobsPopulationMatchesDisk (0.00s)
ok  	github.com/seanb4t/codegraph-go/internal/upgrade	0.166s
```

## Family 05-01/g — kinds reordered

Pre-mutation gate: `git -C "$S" diff --quiet -- .changie.yaml` exit 0

Mutation: move the Breaking kind's two lines (`- label: Breaking`, `auto: minor`) to directly after the Fixes kind's two lines.

Applied-confirmation, `git -C "$S" diff --numstat`:

```
2	2	.changie.yaml
```

Hunk, `git -C "$S" diff -U0 -- .changie.yaml`:

```diff
diff --git a/.changie.yaml b/.changie.yaml
index 99db6edf..c454215e 100644
--- a/.changie.yaml
+++ b/.changie.yaml
@@ -45,2 +44,0 @@ kinds:
-  - label: Breaking
-    auto: minor        # pre-1.0: breaking bumps minor. Flip to major at 1.0.
@@ -50,0 +49,2 @@ kinds:
+  - label: Breaking
+    auto: minor        # pre-1.0: breaking bumps minor. Flip to major at 1.0.
```

RED guard: `task check:changie` (from `$S`):

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
::error::check:changie: [14/14] (iii) .changes/v0.15.0.md does not render '### Breaking' (line '10') before '### Fixes' (line '3')
task: Failed to run task "check:changie": exit status 1
```

RED guard: `go test ./internal/upgrade/ -run '^TestChangieConfigShape$' -count=1 -v` (from `$S`):

```
    changie_shape_test.go:246: ../../.changie.yaml: kinds[0].label = "Features", want "Breaking" (position-order matters — it fixes the kind-section order of every future batch)
--- FAIL: TestChangieConfigShape (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/upgrade	0.157s
FAIL
```

Byte-clean proof: reverted with `git -C "$S" checkout -- .changie.yaml`; `git -C "$S" diff --quiet` exit 0

Re-run GREEN after the revert, `task check:changie` (last line):

```
check:changie: 14 of 14 checks passed against a scratch copy (source tree byte-unchanged)
```

Re-run GREEN after the revert, `go test ./internal/upgrade/ -run '^TestChangieConfigShape$' -count=1 -v`:

```
--- PASS: TestChangieConfigShape (0.00s)
ok  	github.com/seanb4t/codegraph-go/internal/upgrade	0.150s
```

## Family 05-01/h — CHANGELOG.md history rewritten at merge

Pre-mutation gate: `git -C "$S" diff --quiet -- .changie.yaml` exit 0

Mutation: append one line at the end of .changie.yaml: `replacements: [{path: CHANGELOG.md, find: '0\.14\.1', replace: '{{.VersionNoPrefix}}'}]`.

Applied-confirmation, `git -C "$S" diff --numstat`:

```
1	0	.changie.yaml
```

Hunk, `git -C "$S" diff -U0 -- .changie.yaml`:

```diff
diff --git a/.changie.yaml b/.changie.yaml
index 99db6edf..f0bd82e9 100644
--- a/.changie.yaml
+++ b/.changie.yaml
@@ -64,0 +65 @@ newlines:
+replacements: [{path: CHANGELOG.md, find: '0\.14\.1', replace: '{{.VersionNoPrefix}}'}]
```

RED guard: `task check:changie` (from `$S`):

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
::error::check:changie: [14/14] (iv) merged CHANGELOG.md is not header + blank line + .changes/v0.15.0.md + the old history byte for byte
--- /var/folders/_b/3hyf5qvs62q0wh2vyh856z580000gn/T/tmp.xZs64OQjnu/leg14.expected	2026-10-14 17:15:34
+++ /var/folders/_b/3hyf5qvs62q0wh2vyh856z580000gn/T/tmp.xZs64OQjnu/work/CHANGELOG.md	2026-10-14 17:15:34
@@ -13,7 +13,7 @@
 - check:changie collision probe one ([#2](https://github.com/seanb4t/codegraph-go/pull/2))
 - check:changie collision probe two ([#3](https://github.com/seanb4t/codegraph-go/pull/3))
 
-## [0.14.1](https://github.com/seanb4t/codegraph-go/compare/v0.14.0...v0.14.1) (2026-10-13)
+## [0.15.0](https://github.com/seanb4t/codegraph-go/compare/v0.14.0...v0.15.0) (2026-10-13)
 
 
 ### Bug Fixes
task: Failed to run task "check:changie": exit status 1
```

RED guard: `go test ./internal/upgrade/ -run '^TestChangieConfigShape$' -count=1 -v` (from `$S`):

```
    changie_shape_test.go:181: ../../.changie.yaml: root mapping key set = [changeFormat changelogPath changesDir custom fragmentFileFormat headerPath kindFormat kinds newlines replacements unreleasedDir versionExt versionFormat], want exactly [changeFormat changelogPath changesDir custom fragmentFileFormat headerPath kindFormat kinds newlines unreleasedDir versionExt versionFormat]
--- FAIL: TestChangieConfigShape (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/upgrade	0.151s
FAIL
```

Byte-clean proof: reverted with `git -C "$S" checkout -- .changie.yaml`; `git -C "$S" diff --quiet` exit 0

Re-run GREEN after the revert, `task check:changie` (last line):

```
check:changie: 14 of 14 checks passed against a scratch copy (source tree byte-unchanged)
```

Re-run GREEN after the revert, `go test ./internal/upgrade/ -run '^TestChangieConfigShape$' -count=1 -v`:

```
--- PASS: TestChangieConfigShape (0.00s)
ok  	github.com/seanb4t/codegraph-go/internal/upgrade	0.154s
```

## Teardown

`git worktree remove --force "$S"` and `git worktree prune`, then `git worktree list`:

```
/Volumes/Code/github.com/seanb4t/codegraph-go                        c4c19ade [gsd/v0.15.0-milestone]
/Volumes/Code/herdr-worktrees/codegraph-go/worktree-brave-river-a240 2df78be0 [chore/octopusignore]
```

No `p05-scratch-wt` entry remains.
