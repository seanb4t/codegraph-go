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

## 05-02 RED-first transcript

Commit: the `test(05-02):` commit, before `release-tag.yml` and `release:tag` exist. Command:
`GOTOOLCHAIN=go1.26.9 go test ./internal/upgrade/ -count=1 -v`. Every failure is an assertion or read failure
on the missing subject; there is no build error. Exactly six tests fail in the whole package; the other
tests, including `TestTagCreationScanFlagsPlantedSites` and `TestGsdTagCreationIsDisabled`, pass.
`TestTagCreationScanFlagsPlantedSites` passes at RED by design: it feeds the scanner literal inputs only.
Its RED evidence is Families 05-02/g, 05-02/h and 05-02/i below, which mutate the scanner (rule 84d1gfpywd).

```
    changie_release_shape_test.go:601: read ../../.github/workflows/release-tag.yml: open ../../.github/workflows/release-tag.yml: no such file or directory
--- FAIL: TestReleaseTagWorkflowShape (0.00s)
    changie_release_shape_test.go:774: ../../Taskfile.yml: no top-level "release:tag" task
--- FAIL: TestReleaseTagTargetShape (0.00s)
    changie_release_shape_test.go:1333: found 0 tag-creating units map[], want exactly 1 (task:release:tag)
    changie_release_shape_test.go:1337: GRD-16: found zero tag-creating commands in task:release:tag, want exactly that one unit; task release:tag is invoked from [], want exactly ["release-tag.yml/tag/Tag the release"]
    changie_release_shape_test.go:1342: release-tag.yml is not among the 17 workflow files on disk
    changie_release_shape_test.go:1349: tag-creation scan: 17 workflow files (17 on disk), 83 run bodies, 1 local actions, 71 Taskfile targets, 241 executed Taskfile scalars, 0 tag-creating lines in 0 unit(s), release:tag invoked from []
--- FAIL: TestTagCreationHasSingleAppTokenSite (0.02s)
    taskfile_shape_test.go:1400: read ../../.github/workflows/release-tag.yml: open ../../.github/workflows/release-tag.yml: no such file or directory
--- FAIL: TestWorkflowRunBodiesInvokeTask (0.01s)
    taskfile_shape_test.go:1613: inScopeWorkflowFiles names "release-tag.yml", which no longer exists on disk
--- FAIL: TestWorkflowFilePopulationMatchesDisk (0.00s)
    taskfile_shape_test.go:1686: read ../../.github/workflows/release-tag.yml: open ../../.github/workflows/release-tag.yml: no such file or directory
--- FAIL: TestInScopeJobsPopulationMatchesDisk (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/upgrade	0.201s
FAIL
```

## 05-02 GREEN

Commits: `179aacb2` (`ci(05-02):` release-tag.yml, release:tag, check:cross desc) and `59fc881d` (`test(05-02):` the
block-scalar-marker fix in TestReleaseTagTargetShape, a one-line test bug RED could not show because release:tag
was absent). `git log --first-parent --no-merges --reverse` over release-tag.yml and
changie_release_shape_test.go since `5a8603f5` orders the 05-02 commits test, ci, test; the first two are
test(05-02) then ci(05-02). Command: the eight named tests, `-count=1 -v`; then `task lint:actions` (exit 0).
The scanner's logged population counts are on the `TestTagCreationHasSingleAppTokenSite` line.

```
    changie_release_shape_test.go:745: ../../.github/workflows/release-tag.yml: 114 fields, lines and uses checked (7 uses: lines, 3 distinct pins, 2 run bodies)
--- PASS: TestReleaseTagWorkflowShape (0.00s)
    changie_release_shape_test.go:885: release:tag: 7 of 7 ordered tokens found; 12 body lines; 1 changie invocation
--- PASS: TestReleaseTagTargetShape (0.00s)
    changie_release_shape_test.go:1350: tag-creation scan: 18 workflow files (18 on disk), 85 run bodies, 1 local actions, 72 Taskfile targets, 244 executed Taskfile scalars, 2 tag-creating lines in 1 unit(s), release:tag invoked from ["release-tag.yml/tag/Tag the release"]
--- PASS: TestTagCreationHasSingleAppTokenSite (0.02s)
--- PASS: TestTagCreationScanFlagsPlantedSites (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/baseline_single_site (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/reject_second_workflow_push_tags (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/reject_second_taskfile_git_tag (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/reject_gh_release_create (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/reject_second_release_tag_invocation (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/reject_release_tag_push_widened (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/ignore_comment_git_tag (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/ignore_desc_git_tag (0.00s)
--- PASS: TestGsdTagCreationIsDisabled (0.00s)
--- PASS: TestWorkflowRunBodiesInvokeTask (0.01s)
--- PASS: TestWorkflowFilePopulationMatchesDisk (0.00s)
--- PASS: TestInScopeJobsPopulationMatchesDisk (0.00s)
PASS
ok  	github.com/seanb4t/codegraph-go/internal/upgrade	0.211s
```

## 05-02 release:tag rehearsal (scratch bare origin)

Evidence, not a committed test (rule eyva2svrcz): it runs this repository's own `release:tag`, whose
skip-if-tag-exists branch is the D-06 loop guard for the milestone merge, against a local bare repository only.
Setup: `git init -q --bare "$T/origin.git"`, `git clone -q --no-tags "$R" "$T/clone"`, `git -C "$T/clone" remote set-url origin "$T/origin.git"`,
with `T` a fresh directory under the session scratchpad and `R` the real checkout. Before every run an origin guard
reads `git remote get-url origin` and `git remote get-url --push origin` in the clone and aborts unless both begin
with `$T/` and name no `github.com`. Every run is `(cd "$T/clone" && GITHUB_ACTIONS=true task release:tag)`, so Task reads
the clone's Taskfile.yml; `GITHUB_ACTIONS` is a prefix assignment on that one command and is never exported (the
calling shell had it unset). Run 4 drops it. The tag is `changie latest` in the clone, the newest seed, `v0.14.1`.

```
### Run 1 (absent tag: tag and push)
origin guard: fetch=/private/tmp/claude-501/-Volumes-Code-github-com-seanb4t-codegraph-go/d5fbaa99-ed39-4562-985b-aad98b49b210/scratchpad/exec-05-02/rehearsal/origin.git push=/private/tmp/claude-501/-Volumes-Code-github-com-seanb4t-codegraph-go/d5fbaa99-ed39-4562-985b-aad98b49b210/scratchpad/exec-05-02/rehearsal/origin.git ok
$ (cd clone && GITHUB_ACTIONS=true task release:tag)
To /private/tmp/claude-501/-Volumes-Code-github-com-seanb4t-codegraph-go/d5fbaa99-ed39-4562-985b-aad98b49b210/scratchpad/exec-05-02/rehearsal/origin.git
 * [new tag]           v0.14.1 -> v0.14.1
release:tag: pushed v0.14.1 at e6850a48813e7c9f8ad6ca372bb92e7a750f08d4
exit=0
origin tags: v0.14.1 
### Run 2 (tag now present on origin: skip)
origin guard: fetch=/private/tmp/claude-501/-Volumes-Code-github-com-seanb4t-codegraph-go/d5fbaa99-ed39-4562-985b-aad98b49b210/scratchpad/exec-05-02/rehearsal/origin.git push=/private/tmp/claude-501/-Volumes-Code-github-com-seanb4t-codegraph-go/d5fbaa99-ed39-4562-985b-aad98b49b210/scratchpad/exec-05-02/rehearsal/origin.git ok
$ (cd clone && GITHUB_ACTIONS=true task release:tag)
release:tag: v0.14.1 already exists on origin — nothing to do
exit=0
### Run 3 (origin points at a missing path: transport error)
origin guard: fetch=/private/tmp/claude-501/-Volumes-Code-github-com-seanb4t-codegraph-go/d5fbaa99-ed39-4562-985b-aad98b49b210/scratchpad/exec-05-02/rehearsal/missing.git push=/private/tmp/claude-501/-Volumes-Code-github-com-seanb4t-codegraph-go/d5fbaa99-ed39-4562-985b-aad98b49b210/scratchpad/exec-05-02/rehearsal/missing.git ok
$ (cd clone && GITHUB_ACTIONS=true task release:tag)
fatal: '/private/tmp/claude-501/-Volumes-Code-github-com-seanb4t-codegraph-go/d5fbaa99-ed39-4562-985b-aad98b49b210/scratchpad/exec-05-02/rehearsal/missing.git' does not appear to be a git repository
fatal: Could not read from remote repository.

Please make sure you have the correct access rights
and the repository exists.
::error::release:tag: git ls-remote exited 128; refusing to guess whether v0.14.1 exists on origin
::error title=Task 'release:tag' failed::exit status 1
task: Failed to run task "release:tag": exit status 1
exit=201
### Run 4 (without GITHUB_ACTIONS: refused by the precondition)
origin guard: fetch=/private/tmp/claude-501/-Volumes-Code-github-com-seanb4t-codegraph-go/d5fbaa99-ed39-4562-985b-aad98b49b210/scratchpad/exec-05-02/rehearsal/origin.git push=/private/tmp/claude-501/-Volumes-Code-github-com-seanb4t-codegraph-go/d5fbaa99-ed39-4562-985b-aad98b49b210/scratchpad/exec-05-02/rehearsal/origin.git ok
$ (cd clone && task release:tag)   # GITHUB_ACTIONS unset
task: release:tag pushes a tag to origin and runs only in release-tag.yml (D-06R: no human git tag). Rehearse it against a scratch clone whose origin is a scratch bare repository, with GITHUB_ACTIONS=true set for that one command.
task: Failed to run task "release:tag": task: precondition not met
exit=201
real-repo tag --list 'v0.15*' before=[] after=[] equal
real-repo refs/tags sha256 before=6d900e5f187b3d9c27c888e29b8e62b624fd5b9d164be219af6730a41f2bc8fd after=6d900e5f187b3d9c27c888e29b8e62b624fd5b9d164be219af6730a41f2bc8fd equal
rc: 0 0 201 201
```

## Families 05-02/a-i — setup

Convention as above (Pre-mutation cleanliness gate). The scratch worktree was created with
`git worktree add -q --detach "$S" HEAD` at the commit that holds `release-tag.yml`, `release:tag` and the four tests,
with `S` a directory named `p05-scratch-wt` under the session scratchpad. Each family runs in `$S` only; the main
checkout is never mutated. Families a-f mutate the subject files and are held by the shape tests and the GRD-16 scan.
Families g-i mutate the scanner and validator inside `changie_release_shape_test.go`, so
`TestTagCreationScanFlagsPlantedSites`, which passes at RED by design, is seen failing on its own assertion: a named
rejection subtest fails because `validateTagCreationScan` returned no error, or an error that does not name the
planted unit. Under g, `baseline_single_site` and both `ignore_*` subtests also go RED, because release:tag then has
no push line the validator can check; that is recorded and does not replace the named subtest's RED.
Under h and i, `baseline_single_site` and `TestTagCreationHasSingleAppTokenSite` stay green.

## Family 05-02/a — tag job checkout uses the default token

Pre-mutation gate: `git -C "$S" diff --quiet -- .github/workflows/release-tag.yml` exit 0

Mutation: replace release-tag.yml's checkout `token:` value with the default token context `${{ github.token }}`.

Applied-confirmation, `git -C "$S" diff --numstat`:

```
1	1	.github/workflows/release-tag.yml
```

Hunk, `git -C "$S" diff -U0 -- .github/workflows/release-tag.yml`:

```diff
diff --git a/.github/workflows/release-tag.yml b/.github/workflows/release-tag.yml
index 3155d586..75ff419a 100644
--- a/.github/workflows/release-tag.yml
+++ b/.github/workflows/release-tag.yml
@@ -79 +79 @@ jobs:
-          token: ${{ steps.app-token.outputs.token }}
+          token: ${{ github.token }}
```

RED guard: `go test ./internal/upgrade/ -run '^(TestReleaseTagWorkflowShape|TestTagCreationHasSingleAppTokenSite)$' -count=1 -v` (from `$S`):

```
    changie_release_shape_test.go:682: job "tag" step 2 (Checkout): with = map[token:${{ github.token }}], want map[token:${{ steps.app-token.outputs.token }}]
    changie_release_shape_test.go:734: non-comment line 50 names the default token (REL-11): "          token: ${{ github.token }}"
    changie_release_shape_test.go:745: ../../.github/workflows/release-tag.yml: 114 fields, lines and uses checked (7 uses: lines, 3 distinct pins, 2 run bodies)
--- FAIL: TestReleaseTagWorkflowShape (0.00s)
    changie_release_shape_test.go:1347: T-05-06: tag job checkout token = "${{ github.token }}", want "${{ steps.app-token.outputs.token }}"
    changie_release_shape_test.go:1350: tag-creation scan: 18 workflow files (18 on disk), 85 run bodies, 1 local actions, 72 Taskfile targets, 244 executed Taskfile scalars, 2 tag-creating lines in 1 unit(s), release:tag invoked from ["release-tag.yml/tag/Tag the release"]
--- FAIL: TestTagCreationHasSingleAppTokenSite (0.03s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/upgrade	0.269s
FAIL
```

Byte-clean proof: reverted with `git -C "$S" checkout -- .github/workflows/release-tag.yml`; `git -C "$S" diff --quiet` exit 0

Re-run GREEN after the revert, same command:

```
    changie_release_shape_test.go:745: ../../.github/workflows/release-tag.yml: 114 fields, lines and uses checked (7 uses: lines, 3 distinct pins, 2 run bodies)
--- PASS: TestReleaseTagWorkflowShape (0.00s)
    changie_release_shape_test.go:1350: tag-creation scan: 18 workflow files (18 on disk), 85 run bodies, 1 local actions, 72 Taskfile targets, 244 executed Taskfile scalars, 2 tag-creating lines in 1 unit(s), release:tag invoked from ["release-tag.yml/tag/Tag the release"]
--- PASS: TestTagCreationHasSingleAppTokenSite (0.02s)
PASS
ok  	github.com/seanb4t/codegraph-go/internal/upgrade	0.195s
```

## Family 05-02/b — planted second tag push in a workflow run body

Pre-mutation gate: `git -C "$S" diff --quiet -- .github/workflows/ci.yml` exit 0

Mutation: append a step to ci.yml's `test` job: `- name: planted` with the run body `git push origin --tags`.

Applied-confirmation, `git -C "$S" diff --numstat`:

```
3	0	.github/workflows/ci.yml
```

Hunk, `git -C "$S" diff -U0 -- .github/workflows/ci.yml`:

```diff
diff --git a/.github/workflows/ci.yml b/.github/workflows/ci.yml
index 85f59d6a..1d48f048 100644
--- a/.github/workflows/ci.yml
+++ b/.github/workflows/ci.yml
@@ -278,0 +279,3 @@ jobs:
+      - name: planted
+        run: git push origin --tags
+
```

RED guard: `go test ./internal/upgrade/ -run '^(TestTagCreationHasSingleAppTokenSite|TestWorkflowRunBodiesInvokeTask)$' -count=1 -v` (from `$S`):

```
    changie_release_shape_test.go:1334: found 2 tag-creating units map[task:release:tag:true workflow:ci.yml/test/planted:true], want exactly 1 (task:release:tag)
    changie_release_shape_test.go:1338: GRD-16: unit "workflow:ci.yml/test/planted" holds a tag-creating command "git push origin --tags"; only task:release:tag may (GRD-16, D-06R)
    changie_release_shape_test.go:1350: tag-creation scan: 18 workflow files (18 on disk), 86 run bodies, 1 local actions, 72 Taskfile targets, 244 executed Taskfile scalars, 3 tag-creating lines in 2 unit(s), release:tag invoked from ["release-tag.yml/tag/Tag the release"]
--- FAIL: TestTagCreationHasSingleAppTokenSite (0.02s)
    taskfile_shape_test.go:1412: ci.yml job "test" step "planted"'s run: body is not a single 'task <target>' call after stripping comments/blanks (D-01): "git push origin --tags"
--- FAIL: TestWorkflowRunBodiesInvokeTask (0.01s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/upgrade	0.197s
FAIL
```

Byte-clean proof: reverted with `git -C "$S" checkout -- .github/workflows/ci.yml`; `git -C "$S" diff --quiet` exit 0

Re-run GREEN after the revert, same command:

```
    changie_release_shape_test.go:1350: tag-creation scan: 18 workflow files (18 on disk), 85 run bodies, 1 local actions, 72 Taskfile targets, 244 executed Taskfile scalars, 2 tag-creating lines in 1 unit(s), release:tag invoked from ["release-tag.yml/tag/Tag the release"]
--- PASS: TestTagCreationHasSingleAppTokenSite (0.02s)
--- PASS: TestWorkflowRunBodiesInvokeTask (0.01s)
PASS
ok  	github.com/seanb4t/codegraph-go/internal/upgrade	0.176s
```

## Family 05-02/c — planted git tag in a Taskfile target a workflow calls

Pre-mutation gate: `git -C "$S" diff --quiet -- Taskfile.yml` exit 0

Mutation: add the line `git tag v9.9.9` to check:cross's cmds block (release-pr.yml and release-tag.yml call check:cross).

Applied-confirmation, `git -C "$S" diff --numstat`:

```
1	0	Taskfile.yml
```

Hunk, `git -C "$S" diff -U0 -- Taskfile.yml`:

```diff
diff --git a/Taskfile.yml b/Taskfile.yml
index ee8c48b9..15ec7edb 100644
--- a/Taskfile.yml
+++ b/Taskfile.yml
@@ -5401,0 +5402 @@ tasks:
+        git tag v9.9.9
```

RED guard: `go test ./internal/upgrade/ -run '^TestTagCreationHasSingleAppTokenSite$' -count=1 -v` (from `$S`):

```
    changie_release_shape_test.go:1334: found 2 tag-creating units map[task:check:cross:true task:release:tag:true], want exactly 1 (task:release:tag)
    changie_release_shape_test.go:1338: GRD-16: unit "task:check:cross" holds a tag-creating command "git tag v9.9.9"; only task:release:tag may (GRD-16, D-06R)
    changie_release_shape_test.go:1350: tag-creation scan: 18 workflow files (18 on disk), 85 run bodies, 1 local actions, 72 Taskfile targets, 244 executed Taskfile scalars, 3 tag-creating lines in 2 unit(s), release:tag invoked from ["release-tag.yml/tag/Tag the release"]
--- FAIL: TestTagCreationHasSingleAppTokenSite (0.02s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/upgrade	0.188s
FAIL
```

Byte-clean proof: reverted with `git -C "$S" checkout -- Taskfile.yml`; `git -C "$S" diff --quiet` exit 0

Re-run GREEN after the revert, same command:

```
    changie_release_shape_test.go:1350: tag-creation scan: 18 workflow files (18 on disk), 85 run bodies, 1 local actions, 72 Taskfile targets, 244 executed Taskfile scalars, 2 tag-creating lines in 1 unit(s), release:tag invoked from ["release-tag.yml/tag/Tag the release"]
--- PASS: TestTagCreationHasSingleAppTokenSite (0.02s)
PASS
ok  	github.com/seanb4t/codegraph-go/internal/upgrade	0.193s
```

## Family 05-02/d — path filter dropped

Pre-mutation gate: `git -C "$S" diff --quiet -- .github/workflows/release-tag.yml` exit 0

Mutation: delete release-tag.yml's `paths: [CHANGELOG.md]` line.

Applied-confirmation, `git -C "$S" diff --numstat`:

```
0	1	.github/workflows/release-tag.yml
```

Hunk, `git -C "$S" diff -U0 -- .github/workflows/release-tag.yml`:

```diff
diff --git a/.github/workflows/release-tag.yml b/.github/workflows/release-tag.yml
index 3155d586..bde2368f 100644
--- a/.github/workflows/release-tag.yml
+++ b/.github/workflows/release-tag.yml
@@ -31 +30,0 @@ on:
-    paths: [CHANGELOG.md]
```

RED guard: `go test ./internal/upgrade/ -run '^TestReleaseTagWorkflowShape$' -count=1 -v` (from `$S`):

```
    changie_release_shape_test.go:621: on.push keys = [branches], want exactly [branches paths]
    changie_release_shape_test.go:745: ../../.github/workflows/release-tag.yml: 113 fields, lines and uses checked (7 uses: lines, 3 distinct pins, 2 run bodies)
--- FAIL: TestReleaseTagWorkflowShape (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/upgrade	0.166s
FAIL
```

Byte-clean proof: reverted with `git -C "$S" checkout -- .github/workflows/release-tag.yml`; `git -C "$S" diff --quiet` exit 0

Re-run GREEN after the revert, same command:

```
    changie_release_shape_test.go:745: ../../.github/workflows/release-tag.yml: 114 fields, lines and uses checked (7 uses: lines, 3 distinct pins, 2 run bodies)
--- PASS: TestReleaseTagWorkflowShape (0.00s)
PASS
ok  	github.com/seanb4t/codegraph-go/internal/upgrade	0.158s
```

## Family 05-02/e — skip-if-exists branch removed

Pre-mutation gate: `git -C "$S" diff --quiet -- Taskfile.yml` exit 0

Mutation: delete release:tag's `0)` case arm.

Applied-confirmation, `git -C "$S" diff --numstat`:

```
0	1	Taskfile.yml
```

Hunk, `git -C "$S" diff -U0 -- Taskfile.yml`:

```diff
diff --git a/Taskfile.yml b/Taskfile.yml
index ee8c48b9..4c041f81 100644
--- a/Taskfile.yml
+++ b/Taskfile.yml
@@ -1059 +1058,0 @@ tasks:
-          0) echo "release:tag: ${tag} already exists on origin — nothing to do"; exit 0 ;;
```

RED guard: `go test ./internal/upgrade/ -run '^TestReleaseTagTargetShape$' -count=1 -v` (from `$S`):

```
    changie_release_shape_test.go:822: release:tag cmds: token "0) echo \"release:tag: ${tag} already exists on origin" (existing-tag arm) not found at or after byte offset 167
    changie_release_shape_test.go:858: release:tag has 1 ls-remote lines, 0 existing-tag arms and 1 transport-error arms, want 1 of each
    changie_release_shape_test.go:885: release:tag: 6 of 7 ordered tokens found; 11 body lines; 1 changie invocation
--- FAIL: TestReleaseTagTargetShape (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/upgrade	0.159s
FAIL
```

Byte-clean proof: reverted with `git -C "$S" checkout -- Taskfile.yml`; `git -C "$S" diff --quiet` exit 0

Re-run GREEN after the revert, same command:

```
    changie_release_shape_test.go:885: release:tag: 7 of 7 ordered tokens found; 12 body lines; 1 changie invocation
--- PASS: TestReleaseTagTargetShape (0.00s)
PASS
ok  	github.com/seanb4t/codegraph-go/internal/upgrade	0.166s
```

## Family 05-02/f — push widened to all tags

Pre-mutation gate: `git -C "$S" diff --quiet -- Taskfile.yml` exit 0

Mutation: change release:tag's push to `git push origin --tags`.

Applied-confirmation, `git -C "$S" diff --numstat`:

```
1	1	Taskfile.yml
```

Hunk, `git -C "$S" diff -U0 -- Taskfile.yml`:

```diff
diff --git a/Taskfile.yml b/Taskfile.yml
index ee8c48b9..0873de49 100644
--- a/Taskfile.yml
+++ b/Taskfile.yml
@@ -1064 +1064 @@ tasks:
-        git push origin "refs/tags/${tag}"
+        git push origin --tags
```

RED guard: `go test ./internal/upgrade/ -run '^(TestReleaseTagTargetShape|TestTagCreationHasSingleAppTokenSite)$' -count=1 -v` (from `$S`):

```
    changie_release_shape_test.go:822: release:tag cmds: token "git push origin \"refs/tags/${tag}\"" (single-ref push) not found at or after byte offset 490
    changie_release_shape_test.go:866: release:tag push widens past the one tag ref (--tags): "git push origin --tags"
    changie_release_shape_test.go:885: release:tag: 6 of 7 ordered tokens found; 12 body lines; 1 changie invocation
--- FAIL: TestReleaseTagTargetShape (0.00s)
    changie_release_shape_test.go:1338: GRD-16: unit "task:release:tag" pushes without naming refs/tags/: "git push origin --tags"; unit "task:release:tag" push uses --tags, which widens past the one tag: "git push origin --tags"
    changie_release_shape_test.go:1350: tag-creation scan: 18 workflow files (18 on disk), 85 run bodies, 1 local actions, 72 Taskfile targets, 244 executed Taskfile scalars, 2 tag-creating lines in 1 unit(s), release:tag invoked from ["release-tag.yml/tag/Tag the release"]
--- FAIL: TestTagCreationHasSingleAppTokenSite (0.02s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/upgrade	0.175s
FAIL
```

Byte-clean proof: reverted with `git -C "$S" checkout -- Taskfile.yml`; `git -C "$S" diff --quiet` exit 0

Re-run GREEN after the revert, same command:

```
    changie_release_shape_test.go:885: release:tag: 7 of 7 ordered tokens found; 12 body lines; 1 changie invocation
--- PASS: TestReleaseTagTargetShape (0.00s)
    changie_release_shape_test.go:1350: tag-creation scan: 18 workflow files (18 on disk), 85 run bodies, 1 local actions, 72 Taskfile targets, 244 executed Taskfile scalars, 2 tag-creating lines in 1 unit(s), release:tag invoked from ["release-tag.yml/tag/Tag the release"]
--- PASS: TestTagCreationHasSingleAppTokenSite (0.02s)
PASS
ok  	github.com/seanb4t/codegraph-go/internal/upgrade	0.181s
```

## Family 05-02/g — scanner's git push pattern disabled

Pre-mutation gate: `git -C "$S" diff --quiet -- internal/upgrade/changie_release_shape_test.go` exit 0

Mutation: edit the `git ... push` pattern so it can no longer match: `push(\s|$)` becomes `pushMUTATED(\s|$)`.

Applied-confirmation, `git -C "$S" diff --numstat`:

```
1	1	internal/upgrade/changie_release_shape_test.go
```

Hunk, `git -C "$S" diff -U0 -- internal/upgrade/changie_release_shape_test.go`:

```diff
diff --git a/internal/upgrade/changie_release_shape_test.go b/internal/upgrade/changie_release_shape_test.go
index b754ccc1..59da1b45 100644
--- a/internal/upgrade/changie_release_shape_test.go
+++ b/internal/upgrade/changie_release_shape_test.go
@@ -905 +905 @@ var (
-	tagCreationGitPushRe         = regexp.MustCompile(`(^|[;&|(\s])git(\s+-[cC]\s+\S+)*\s+push(\s|$)`)
+	tagCreationGitPushRe         = regexp.MustCompile(`(^|[;&|(\s])git(\s+-[cC]\s+\S+)*\s+pushMUTATED(\s|$)`)
```

RED guard: `go test ./internal/upgrade/ -run '^(TestTagCreationScanFlagsPlantedSites|TestTagCreationHasSingleAppTokenSite)$' -count=1 -v` (from `$S`):

```
    changie_release_shape_test.go:1338: GRD-16: unit "task:release:tag" holds no push line to check
    changie_release_shape_test.go:1350: tag-creation scan: 18 workflow files (18 on disk), 85 run bodies, 1 local actions, 72 Taskfile targets, 244 executed Taskfile scalars, 1 tag-creating lines in 1 unit(s), release:tag invoked from ["release-tag.yml/tag/Tag the release"]
--- FAIL: TestTagCreationHasSingleAppTokenSite (0.03s)
    changie_release_shape_test.go:1484: validateTagCreationScan rejected a clean scan: unit "task:release:tag" holds no push line to check
    changie_release_shape_test.go:1504: validateTagCreationScan error "unit \"task:release:tag\" holds no push line to check" does not name the offending unit "workflow:ci.yml/test/planted"
    changie_release_shape_test.go:1484: validateTagCreationScan rejected a clean scan: unit "task:release:tag" holds no push line to check
    changie_release_shape_test.go:1484: validateTagCreationScan rejected a clean scan: unit "task:release:tag" holds no push line to check
--- FAIL: TestTagCreationScanFlagsPlantedSites (0.00s)
    --- FAIL: TestTagCreationScanFlagsPlantedSites/baseline_single_site (0.00s)
    --- FAIL: TestTagCreationScanFlagsPlantedSites/reject_second_workflow_push_tags (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/reject_second_taskfile_git_tag (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/reject_gh_release_create (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/reject_second_release_tag_invocation (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/reject_release_tag_push_widened (0.00s)
    --- FAIL: TestTagCreationScanFlagsPlantedSites/ignore_comment_git_tag (0.00s)
    --- FAIL: TestTagCreationScanFlagsPlantedSites/ignore_desc_git_tag (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/upgrade	0.219s
FAIL
```

Byte-clean proof: reverted with `git -C "$S" checkout -- internal/upgrade/changie_release_shape_test.go`; `git -C "$S" diff --quiet` exit 0

Re-run GREEN after the revert, same command:

```
    changie_release_shape_test.go:1350: tag-creation scan: 18 workflow files (18 on disk), 85 run bodies, 1 local actions, 72 Taskfile targets, 244 executed Taskfile scalars, 2 tag-creating lines in 1 unit(s), release:tag invoked from ["release-tag.yml/tag/Tag the release"]
--- PASS: TestTagCreationHasSingleAppTokenSite (0.02s)
--- PASS: TestTagCreationScanFlagsPlantedSites (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/baseline_single_site (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/reject_second_workflow_push_tags (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/reject_second_taskfile_git_tag (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/reject_gh_release_create (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/reject_second_release_tag_invocation (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/reject_release_tag_push_widened (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/ignore_comment_git_tag (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/ignore_desc_git_tag (0.00s)
PASS
ok  	github.com/seanb4t/codegraph-go/internal/upgrade	0.192s
```

## Family 05-02/h — scanner's git tag and gh release create patterns disabled

Pre-mutation gate: `git -C "$S" diff --quiet -- internal/upgrade/changie_release_shape_test.go` exit 0

Mutation: edit those two patterns the same way and leave the push pattern intact: `tag(\s|$)` becomes `tagMUTATED(\s|$)` and `create\b` becomes `createMUTATED\b`.

Applied-confirmation, `git -C "$S" diff --numstat`:

```
2	2	internal/upgrade/changie_release_shape_test.go
```

Hunk, `git -C "$S" diff -U0 -- internal/upgrade/changie_release_shape_test.go`:

```diff
diff --git a/internal/upgrade/changie_release_shape_test.go b/internal/upgrade/changie_release_shape_test.go
index b754ccc1..c8f23445 100644
--- a/internal/upgrade/changie_release_shape_test.go
+++ b/internal/upgrade/changie_release_shape_test.go
@@ -904 +904 @@ var (
-	tagCreationGitTagRe          = regexp.MustCompile(`(^|[;&|(\s])git(\s+-[cC]\s+\S+)*\s+tag(\s|$)`)
+	tagCreationGitTagRe          = regexp.MustCompile(`(^|[;&|(\s])git(\s+-[cC]\s+\S+)*\s+tagMUTATED(\s|$)`)
@@ -906 +906 @@ var (
-	tagCreationGhReleaseCreateRe = regexp.MustCompile(`\bgh\s+release\s+create\b`)
+	tagCreationGhReleaseCreateRe = regexp.MustCompile(`\bgh\s+release\s+createMUTATED\b`)
```

RED guard: `go test ./internal/upgrade/ -run '^(TestTagCreationScanFlagsPlantedSites|TestTagCreationHasSingleAppTokenSite)$' -count=1 -v` (from `$S`):

```
    changie_release_shape_test.go:1350: tag-creation scan: 18 workflow files (18 on disk), 85 run bodies, 1 local actions, 72 Taskfile targets, 244 executed Taskfile scalars, 1 tag-creating lines in 1 unit(s), release:tag invoked from ["release-tag.yml/tag/Tag the release"]
--- PASS: TestTagCreationHasSingleAppTokenSite (0.02s)
    changie_release_shape_test.go:1501: validateTagCreationScan returned no error, want one naming "task:rogue" (sites: [{task:release:tag git push origin "refs/tags/${tag}"}])
    changie_release_shape_test.go:1501: validateTagCreationScan returned no error, want one naming "workflow:ci.yml/test/planted" (sites: [{task:release:tag git push origin "refs/tags/${tag}"}])
--- FAIL: TestTagCreationScanFlagsPlantedSites (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/baseline_single_site (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/reject_second_workflow_push_tags (0.00s)
    --- FAIL: TestTagCreationScanFlagsPlantedSites/reject_second_taskfile_git_tag (0.00s)
    --- FAIL: TestTagCreationScanFlagsPlantedSites/reject_gh_release_create (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/reject_second_release_tag_invocation (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/reject_release_tag_push_widened (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/ignore_comment_git_tag (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/ignore_desc_git_tag (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/upgrade	0.208s
FAIL
```

Byte-clean proof: reverted with `git -C "$S" checkout -- internal/upgrade/changie_release_shape_test.go`; `git -C "$S" diff --quiet` exit 0

Re-run GREEN after the revert, same command:

```
    changie_release_shape_test.go:1350: tag-creation scan: 18 workflow files (18 on disk), 85 run bodies, 1 local actions, 72 Taskfile targets, 244 executed Taskfile scalars, 2 tag-creating lines in 1 unit(s), release:tag invoked from ["release-tag.yml/tag/Tag the release"]
--- PASS: TestTagCreationHasSingleAppTokenSite (0.02s)
--- PASS: TestTagCreationScanFlagsPlantedSites (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/baseline_single_site (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/reject_second_workflow_push_tags (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/reject_second_taskfile_git_tag (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/reject_gh_release_create (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/reject_second_release_tag_invocation (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/reject_release_tag_push_widened (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/ignore_comment_git_tag (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/ignore_desc_git_tag (0.00s)
PASS
ok  	github.com/seanb4t/codegraph-go/internal/upgrade	0.193s
```

## Family 05-02/i — validator relaxed

Pre-mutation gate: `git -C "$S" diff --quiet -- internal/upgrade/changie_release_shape_test.go` exit 0

Mutation: in validateTagCreationScan accept any non-empty TaskInvocations["release:tag"] instead of exactly the tag job's step, and remove `--tags` from the forbidden push flags (the other three stay).

Applied-confirmation, `git -C "$S" diff --numstat`:

```
2	2	internal/upgrade/changie_release_shape_test.go
```

Hunk, `git -C "$S" diff -U0 -- internal/upgrade/changie_release_shape_test.go`:

```diff
diff --git a/internal/upgrade/changie_release_shape_test.go b/internal/upgrade/changie_release_shape_test.go
index b754ccc1..e3879ba7 100644
--- a/internal/upgrade/changie_release_shape_test.go
+++ b/internal/upgrade/changie_release_shape_test.go
@@ -1180 +1180 @@ func scanTagCreationSites(workflows map[string][]byte, actions map[string][]byte
-var tagCreationForbiddenPushFlags = []string{"--tags", "--follow-tags", "--all", "--mirror"}
+var tagCreationForbiddenPushFlags = []string{"--follow-tags", "--all", "--mirror"}
@@ -1228 +1228 @@ func validateTagCreationScan(scan tagCreationScan) error {
-	if !slices.Equal(invokers, []string{tagCreationSoleInvoker}) {
+	if len(invokers) == 0 {
```

RED guard: `go test ./internal/upgrade/ -run '^(TestTagCreationScanFlagsPlantedSites|TestTagCreationHasSingleAppTokenSite)$' -count=1 -v` (from `$S`):

```
    changie_release_shape_test.go:1350: tag-creation scan: 18 workflow files (18 on disk), 85 run bodies, 1 local actions, 72 Taskfile targets, 244 executed Taskfile scalars, 2 tag-creating lines in 1 unit(s), release:tag invoked from ["release-tag.yml/tag/Tag the release"]
--- PASS: TestTagCreationHasSingleAppTokenSite (0.02s)
    changie_release_shape_test.go:1501: validateTagCreationScan returned no error, want one naming "ci.yml/test/planted" (sites: [{task:release:tag git tag "${tag}"} {task:release:tag git push origin "refs/tags/${tag}"}])
    changie_release_shape_test.go:1501: validateTagCreationScan returned no error, want one naming "task:release:tag" (sites: [{task:release:tag git tag "${tag}"} {task:release:tag git push origin "refs/tags/${tag}" --tags}])
--- FAIL: TestTagCreationScanFlagsPlantedSites (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/baseline_single_site (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/reject_second_workflow_push_tags (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/reject_second_taskfile_git_tag (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/reject_gh_release_create (0.00s)
    --- FAIL: TestTagCreationScanFlagsPlantedSites/reject_second_release_tag_invocation (0.00s)
    --- FAIL: TestTagCreationScanFlagsPlantedSites/reject_release_tag_push_widened (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/ignore_comment_git_tag (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/ignore_desc_git_tag (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/upgrade	0.223s
FAIL
```

Byte-clean proof: reverted with `git -C "$S" checkout -- internal/upgrade/changie_release_shape_test.go`; `git -C "$S" diff --quiet` exit 0

Re-run GREEN after the revert, same command:

```
    changie_release_shape_test.go:1350: tag-creation scan: 18 workflow files (18 on disk), 85 run bodies, 1 local actions, 72 Taskfile targets, 244 executed Taskfile scalars, 2 tag-creating lines in 1 unit(s), release:tag invoked from ["release-tag.yml/tag/Tag the release"]
--- PASS: TestTagCreationHasSingleAppTokenSite (0.02s)
--- PASS: TestTagCreationScanFlagsPlantedSites (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/baseline_single_site (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/reject_second_workflow_push_tags (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/reject_second_taskfile_git_tag (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/reject_gh_release_create (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/reject_second_release_tag_invocation (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/reject_release_tag_push_widened (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/ignore_comment_git_tag (0.00s)
    --- PASS: TestTagCreationScanFlagsPlantedSites/ignore_desc_git_tag (0.00s)
PASS
ok  	github.com/seanb4t/codegraph-go/internal/upgrade	0.198s
```

## Teardown (05-02)

`git worktree remove --force "$S"` and `git worktree prune`, then `git worktree list`:

```
/Volumes/Code/github.com/seanb4t/codegraph-go                        67df2d1d [gsd/v0.15.0-milestone]
/Volumes/Code/herdr-worktrees/codegraph-go/worktree-brave-river-a240 2df78be0 [chore/octopusignore]
```

No `p05-scratch-wt` entry remains.

## 05-03 RED-first transcript

Commit: the `test(05-03):` commit, before the Taskfile.yml, .goreleaser.yaml and release.yml edits. Command:
`GOTOOLCHAIN=go1.26.9 go test ./internal/upgrade/ -count=1 -v`. Every failure is an assertion on the missing
wiring; there is no build error. Exactly two tests fail. `TestGoreleaserChangelogNotDisabled` passes at RED
(no changelog block exists); its RED is Family 05-03/a. `TestReleaseBlockDoesNotRewriteReleaseBody` changed prose
only and passes both before and after; Family 05-03/d shows its unchanged logic still bites. Two long desc
strings in the failure lines are elided as `[desc text elided]`; every other character is verbatim.

```
=== RUN   TestReleaseGoreleaserPassesReleaseNotes
    changie_release_shape_test.go:1573: release:goreleaser cmds: token "NOTES=\".changes/${TAG}.md\"" (notes path from the tag) not found at or after byte offset 1622
    changie_release_shape_test.go:1573: release:goreleaser cmds: token "[ ! -s \"${NOTES}\" ]" (empty-notes check) not found at or after byte offset 1622
    changie_release_shape_test.go:1573: release:goreleaser cmds: token "::error::release:goreleaser: ${NOTES} is missing or empty" (empty-notes error) not found at or after byte offset 1622
    changie_release_shape_test.go:1573: release:goreleaser cmds: token "\"${GORELEASER_BIN}\" release --clean --release-notes \"${NOTES}\"" (release call with notes) not found at or after byte offset 1869
    changie_release_shape_test.go:1582: release:goreleaser release invocation is "\"${GORELEASER_BIN}\" release --clean", want exactly `"${GORELEASER_BIN}" release --clean --release-notes "${NOTES}"`
    changie_release_shape_test.go:1600: release:goreleaser desc does not cite REL-12 and --release-notes: [desc text elided]
    changie_release_shape_test.go:1626: release:goreleaser: 2 of 6 ordered tokens found; 1 release invocation; 1 TAG definition; 2 release.yml run bodies
--- FAIL: TestReleaseGoreleaserPassesReleaseNotes (0.00s)
=== RUN   TestReleaseDryRunRehearsesReleaseNotes
    changie_release_shape_test.go:1648: release:dry-run cmds: token "TAG=\"$({{.GO_TOOL_CHANGIE}} changie latest)\"" (tag from changie latest) not found at or after byte offset 0
    changie_release_shape_test.go:1648: release:dry-run cmds: token "NOTES=\".changes/${TAG}.md\"" (notes path from the tag) not found at or after byte offset 0
    changie_release_shape_test.go:1648: release:dry-run cmds: token "[ ! -s \"${NOTES}\" ]" (empty-notes check) not found at or after byte offset 0
    changie_release_shape_test.go:1648: release:dry-run cmds: token "::error::release:dry-run: ${NOTES} is missing or empty" (empty-notes error) not found at or after byte offset 0
    changie_release_shape_test.go:1648: release:dry-run cmds: token "RELEASE-NOTES-EVIDENCE tag=${TAG} file=${NOTES} bytes=" (evidence line) not found at or after byte offset 0
    changie_release_shape_test.go:1648: release:dry-run cmds: token "sha256=" (evidence hash) not found at or after byte offset 0
    changie_release_shape_test.go:1648: release:dry-run cmds: token "GORELEASER_CURRENT_TAG=\"${TAG}\" \"${GORELEASER_BIN}\" release --skip=publish,sign,validate --clean --release-notes \"${NOTES}\"" (non-snapshot invocation) not found at or after byte offset 0
    changie_release_shape_test.go:1654: release:dry-run runs a --snapshot line, which skips the notes pipe: "echo \"==> goreleaser release --snapshot --skip=publish,sign --clean\""
    changie_release_shape_test.go:1654: release:dry-run runs a --snapshot line, which skips the notes pipe: "\"${GORELEASER_BIN}\" release --snapshot --skip=publish,sign --clean"
    changie_release_shape_test.go:1677: release:dry-run invokes changie 0 times, want exactly 1 (latest)
    changie_release_shape_test.go:1684: release:dry-run desc does not cite REL-13: [desc text elided]
    changie_release_shape_test.go:1686: release:dry-run: 0 of 7 ordered tokens found; 0 changie invocation
--- FAIL: TestReleaseDryRunRehearsesReleaseNotes (0.00s)
=== RUN   TestGoreleaserChangelogNotDisabled
    goreleaser_shape_test.go:1039: ../../.goreleaser.yaml parsed (10 top-level keys, release block with 2 keys); no changelog block
--- PASS: TestGoreleaserChangelogNotDisabled (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/upgrade	0.606s
```

## 05-03 GREEN

Commit: the `ci(05-03):` commit, `b390a9d2`. Command (the seven named tests):
`GOTOOLCHAIN=go1.26.9 go test ./internal/upgrade/ -run '^(TestReleaseGoreleaserPassesReleaseNotes|TestReleaseDryRunRehearsesReleaseNotes|TestGoreleaserChangelogNotDisabled|TestReleaseBlockDoesNotRewriteReleaseBody|TestWorkflowRunBodiesInvokeTask|TestReleaseWorkflowTriggerIsTagPushOnly|TestOIDCWriteScopedToSingleGoreleaserJob)$' -count=1 -v`

```
=== RUN   TestReleaseGoreleaserPassesReleaseNotes
    changie_release_shape_test.go:1626: release:goreleaser: 6 of 6 ordered tokens found; 1 release invocation; 1 TAG definition; 2 release.yml run bodies
--- PASS: TestReleaseGoreleaserPassesReleaseNotes (0.00s)
=== RUN   TestReleaseDryRunRehearsesReleaseNotes
    changie_release_shape_test.go:1686: release:dry-run: 7 of 7 ordered tokens found; 1 changie invocation
--- PASS: TestReleaseDryRunRehearsesReleaseNotes (0.00s)
=== RUN   TestReleaseBlockDoesNotRewriteReleaseBody
--- PASS: TestReleaseBlockDoesNotRewriteReleaseBody (0.00s)
=== RUN   TestGoreleaserChangelogNotDisabled
    goreleaser_shape_test.go:1039: ../../.goreleaser.yaml parsed (10 top-level keys, release block with 2 keys); no changelog block
--- PASS: TestGoreleaserChangelogNotDisabled (0.00s)
=== RUN   TestOIDCWriteScopedToSingleGoreleaserJob
--- PASS: TestOIDCWriteScopedToSingleGoreleaserJob (0.00s)
=== RUN   TestReleaseWorkflowTriggerIsTagPushOnly
--- PASS: TestReleaseWorkflowTriggerIsTagPushOnly (0.00s)
=== RUN   TestWorkflowRunBodiesInvokeTask
--- PASS: TestWorkflowRunBodiesInvokeTask (0.00s)
PASS
ok  	github.com/seanb4t/codegraph-go/internal/upgrade	0.190s
```

The whole package, `GOTOOLCHAIN=go1.26.9 go test ./internal/upgrade/ -count=1`:

```
ok  	github.com/seanb4t/codegraph-go/internal/upgrade	0.473s
```

`task lint:actions` exit 0 (actionlint, no finding). `task check:goreleaser` exit 0 (`1 configuration file(s) validated`).

release.yml hunk count from `7b12cb92`: 1, and the only changed lines are comment lines (the plan's second verify command exits 0).
Commit order on `internal/upgrade/goreleaser_shape_test.go` and `.goreleaser.yaml` since `5a8603f5`: `test(05-03):` then `ci(05-03):` (third verify command exits 0).

## 05-03 REL-12 diff evidence

REL-12 (D-08, D-12): the flag lives in Taskfile.yml's `release:goreleaser`; `release.yml` differs from post-#99 `main` (`7b12cb92`) by one comment hunk.

`git diff 7b12cb92c407ae7e495a62b4ac38d8685e64307c -- .github/workflows/release.yml`:

```diff
diff --git a/.github/workflows/release.yml b/.github/workflows/release.yml
index b511039d..b0499239 100644
--- a/.github/workflows/release.yml
+++ b/.github/workflows/release.yml
@@ -185,7 +185,7 @@ jobs:
       # separate-job placement RESEARCH.md Open Question 2 recommended is
       # not usable here. This step therefore mints inside the release job,
       # immediately before the step that consumes it, pinned to the exact
-      # same Action + SHA release-please.yml already carries. This places
+      # same Action + SHA release-pr.yml and release-tag.yml carry. This places
       # one more Action inside the OIDC boundary (T-01-29's accepted
       # surface, now widened by one step) — chosen because the alternative
       # was measured not to work, not because it was preferred. Compensating
```

The `release:goreleaser` hunks of `git diff 5a8603f55537ebfac0461e652cdbe74359914909 -- Taskfile.yml` (the other hunks of that diff belong to `release:batch`, `release:tag`, `check:cross` and `release:dry-run`):

```diff
diff --git a/Taskfile.yml b/Taskfile.yml
index 8bf4f8ec..d464a8c3 100644
--- a/Taskfile.yml
+++ b/Taskfile.yml
@@ -2789,6 +2952,11 @@ tasks:
       Meant to run in CI on a tag push, NOT casually from a laptop. For the
       local, non-publishing equivalent, use `task release:dry-run`.
 
+      The release notes are `.changes/${TAG}.md`, passed with
+      --release-notes (REL-12, changie's documented GoReleaser integration).
+      A tag with no such file fails here, including a hand-pushed rc tag
+      (D-06R).
+
       RELEASE CLASS, BOTH reachable (plan 02-02's goreleaser-caller
       enumeration): this is the one target where `notarize:` and the
       release-scoped `signs:` (cosign) pipe are BOTH live and BOTH expected
@@ -2899,7 +3067,19 @@ tasks:
         # entry's own goos:/goarch: list, not from the environment. A
         # carried-over GOOS/GOARCH would be at best a no-op and at worst
         # mask a misconfiguration.
-        "${GORELEASER_BIN}" release --clean
+        #
+        # The Release body is the changie version file for the tag being
+        # released (REL-12, D-08), handed to GoReleaser with --release-notes.
+        # GoReleaser only warns on an empty notes file, so an absent or empty
+        # one stops here instead (C6): a Release published with an empty
+        # body cannot be repaired by re-cutting the tag (Phase 1 D-12).
+        TAG="$(git describe --exact-match --tags HEAD)"
+        NOTES=".changes/${TAG}.md"
+        if [ ! -s "${NOTES}" ]; then
+          echo "::error::release:goreleaser: ${NOTES} is missing or empty — GoReleaser would publish a release with an empty body (REL-12, C6)"
+          exit 1
+        fi
+        "${GORELEASER_BIN}" release --clean --release-notes "${NOTES}"
 
         # Emit the checksums-file hash as a $GITHUB_OUTPUT `hashes` value
         # when running under GitHub Actions (GITHUB_OUTPUT is only set
@@ -2912,7 +3092,6 @@ tasks:
         # release.yml step, so the release job's Release step keeps
         # exactly one run: body (T-01-14).
         if [ -n "${GITHUB_OUTPUT:-}" ]; then
-          TAG="$(git describe --exact-match --tags HEAD)"
           CHECKSUMS_FILE="dist/codegraph_${TAG}_checksums.txt"
           if [ ! -f "${CHECKSUMS_FILE}" ]; then
             echo "::error::${CHECKSUMS_FILE} not found — expected GoReleaser's checksum: pipe to have produced it"
```

## Families 05-03/a-e — setup

Scratch worktree: a detached checkout of HEAD (c4adcfd5) at `$S`, whose path contains `p05-scratch-wt`. The convention is the one at the top of this log: pre-mutation gate, applied-confirmation, verbatim RED, byte-clean revert, GREEN re-run.

## Family 05-03/a — changelog disabled

Pre-mutation gate: `git -C "$S" diff --quiet -- .goreleaser.yaml` exit 0

Mutation: append `changelog:` with `  disable: true` to `.goreleaser.yaml`.

Applied-confirmation, `git -C "$S" diff --numstat`:

```
2	0	.goreleaser.yaml
```

Hunk, `git -C "$S" diff -U0 -- .goreleaser.yaml`:

```diff
diff --git a/.goreleaser.yaml b/.goreleaser.yaml
index 350a8ec6..f04adb12 100644
--- a/.goreleaser.yaml
+++ b/.goreleaser.yaml
@@ -642,0 +643,2 @@ homebrew_casks:
+changelog:
+  disable: true
```

RED guard: `GOTOOLCHAIN=go1.26.9 go test ./internal/upgrade/ -run '^TestGoreleaserChangelogNotDisabled$' -count=1 -v` (from `$S`):

```
=== RUN   TestGoreleaserChangelogNotDisabled
    goreleaser_shape_test.go:1048: changelog.disable = true (bool), want absent or the boolean false — a true, "true" or template value empties the Release body (REL-12)
    goreleaser_shape_test.go:1051: ../../.goreleaser.yaml parsed (11 top-level keys, release block with 2 keys); changelog block keys: 1
--- FAIL: TestGoreleaserChangelogNotDisabled (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/upgrade	0.192s
FAIL
```

Byte-clean proof: reverted with `git -C "$S" checkout -- .goreleaser.yaml`; `git -C "$S" diff --quiet` exit 0

Re-run GREEN after the revert, same command:

```
=== RUN   TestGoreleaserChangelogNotDisabled
    goreleaser_shape_test.go:1039: ../../.goreleaser.yaml parsed (10 top-level keys, release block with 2 keys); no changelog block
--- PASS: TestGoreleaserChangelogNotDisabled (0.00s)
PASS
ok  	github.com/seanb4t/codegraph-go/internal/upgrade	0.163s
```

## Family 05-03/b — release notes flag dropped

Pre-mutation gate: `git -C "$S" diff --quiet -- Taskfile.yml` exit 0

Mutation: remove ` --release-notes "${NOTES}"` from `release:goreleaser`'s invocation.

Applied-confirmation, `git -C "$S" diff --numstat`:

```
1	1	Taskfile.yml
```

Hunk, `git -C "$S" diff -U0 -- Taskfile.yml`:

```diff
diff --git a/Taskfile.yml b/Taskfile.yml
index d464a8c3..b6d72bf7 100644
--- a/Taskfile.yml
+++ b/Taskfile.yml
@@ -3082 +3082 @@ tasks:
-        "${GORELEASER_BIN}" release --clean --release-notes "${NOTES}"
+        "${GORELEASER_BIN}" release --clean
```

RED guard: `GOTOOLCHAIN=go1.26.9 go test ./internal/upgrade/ -run '^TestReleaseGoreleaserPassesReleaseNotes$' -count=1 -v` (from `$S`):

```
=== RUN   TestReleaseGoreleaserPassesReleaseNotes
    changie_release_shape_test.go:1573: release:goreleaser cmds: token "\"${GORELEASER_BIN}\" release --clean --release-notes \"${NOTES}\"" (release call with notes) not found at or after byte offset 1765
    changie_release_shape_test.go:1582: release:goreleaser release invocation is "\"${GORELEASER_BIN}\" release --clean", want exactly `"${GORELEASER_BIN}" release --clean --release-notes "${NOTES}"`
    changie_release_shape_test.go:1626: release:goreleaser: 5 of 6 ordered tokens found; 1 release invocation; 1 TAG definition; 2 release.yml run bodies
--- FAIL: TestReleaseGoreleaserPassesReleaseNotes (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/upgrade	0.164s
FAIL
```

Byte-clean proof: reverted with `git -C "$S" checkout -- Taskfile.yml`; `git -C "$S" diff --quiet` exit 0

Re-run GREEN after the revert, same command:

```
=== RUN   TestReleaseGoreleaserPassesReleaseNotes
    changie_release_shape_test.go:1626: release:goreleaser: 6 of 6 ordered tokens found; 1 release invocation; 1 TAG definition; 2 release.yml run bodies
--- PASS: TestReleaseGoreleaserPassesReleaseNotes (0.00s)
PASS
ok  	github.com/seanb4t/codegraph-go/internal/upgrade	0.152s
```

## Family 05-03/c — empty-notes guard removed

Pre-mutation gate: `git -C "$S" diff --quiet -- Taskfile.yml` exit 0

Mutation: delete `release:goreleaser`'s `[ ! -s` check block (the four-line `if` through `fi`).

Applied-confirmation, `git -C "$S" diff --numstat`:

```
0	4	Taskfile.yml
```

Hunk, `git -C "$S" diff -U0 -- Taskfile.yml`:

```diff
diff --git a/Taskfile.yml b/Taskfile.yml
index d464a8c3..188571f7 100644
--- a/Taskfile.yml
+++ b/Taskfile.yml
@@ -3078,4 +3077,0 @@ tasks:
-        if [ ! -s "${NOTES}" ]; then
-          echo "::error::release:goreleaser: ${NOTES} is missing or empty — GoReleaser would publish a release with an empty body (REL-12, C6)"
-          exit 1
-        fi
```

RED guard: `GOTOOLCHAIN=go1.26.9 go test ./internal/upgrade/ -run '^TestReleaseGoreleaserPassesReleaseNotes$' -count=1 -v` (from `$S`):

```
=== RUN   TestReleaseGoreleaserPassesReleaseNotes
    changie_release_shape_test.go:1573: release:goreleaser cmds: token "[ ! -s \"${NOTES}\" ]" (empty-notes check) not found at or after byte offset 1565
    changie_release_shape_test.go:1573: release:goreleaser cmds: token "::error::release:goreleaser: ${NOTES} is missing or empty" (empty-notes error) not found at or after byte offset 1565
    changie_release_shape_test.go:1573: release:goreleaser cmds: token "\"${GORELEASER_BIN}\" release --clean --release-notes \"${NOTES}\"" (release call with notes) not found at or after byte offset 1929
    changie_release_shape_test.go:1626: release:goreleaser: 3 of 6 ordered tokens found; 1 release invocation; 1 TAG definition; 2 release.yml run bodies
--- FAIL: TestReleaseGoreleaserPassesReleaseNotes (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/upgrade	0.169s
FAIL
```

Byte-clean proof: reverted with `git -C "$S" checkout -- Taskfile.yml`; `git -C "$S" diff --quiet` exit 0

Re-run GREEN after the revert, same command:

```
=== RUN   TestReleaseGoreleaserPassesReleaseNotes
    changie_release_shape_test.go:1626: release:goreleaser: 6 of 6 ordered tokens found; 1 release invocation; 1 TAG definition; 2 release.yml run bodies
--- PASS: TestReleaseGoreleaserPassesReleaseNotes (0.00s)
PASS
ok  	github.com/seanb4t/codegraph-go/internal/upgrade	0.173s
```

## Family 05-03/d — header key planted

Pre-mutation gate: `git -C "$S" diff --quiet -- .goreleaser.yaml` exit 0

Mutation: add `  header: "x"` under `.goreleaser.yaml`'s `release:` block, after `prerelease: auto`. The restated guard still bites.

Applied-confirmation, `git -C "$S" diff --numstat`:

```
1	0	.goreleaser.yaml
```

Hunk, `git -C "$S" diff -U0 -- .goreleaser.yaml`:

```diff
diff --git a/.goreleaser.yaml b/.goreleaser.yaml
index 350a8ec6..60c9abe6 100644
--- a/.goreleaser.yaml
+++ b/.goreleaser.yaml
@@ -403,0 +404 @@ release:
+  header: "x"
```

RED guard: `GOTOOLCHAIN=go1.26.9 go test ./internal/upgrade/ -run '^TestReleaseBlockDoesNotRewriteReleaseBody$' -count=1 -v` (from `$S`):

```
=== RUN   TestReleaseBlockDoesNotRewriteReleaseBody
    goreleaser_shape_test.go:1010: release: block declares forbidden key "header" — the Release body is .changes/${TAG}.md, passed to GoReleaser with --release-notes (REL-12); this key would add text to it, rename the Release or leave it unpublished
--- FAIL: TestReleaseBlockDoesNotRewriteReleaseBody (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/upgrade	0.177s
FAIL
```

Byte-clean proof: reverted with `git -C "$S" checkout -- .goreleaser.yaml`; `git -C "$S" diff --quiet` exit 0

Re-run GREEN after the revert, same command:

```
=== RUN   TestReleaseBlockDoesNotRewriteReleaseBody
--- PASS: TestReleaseBlockDoesNotRewriteReleaseBody (0.00s)
PASS
ok  	github.com/seanb4t/codegraph-go/internal/upgrade	0.166s
```

## Family 05-03/e — dry run back to snapshot

Pre-mutation gate: `git -C "$S" diff --quiet -- Taskfile.yml` exit 0

Mutation: add `--snapshot` to `release:dry-run`'s invocation.

Applied-confirmation, `git -C "$S" diff --numstat`:

```
1	1	Taskfile.yml
```

Hunk, `git -C "$S" diff -U0 -- Taskfile.yml`:

```diff
diff --git a/Taskfile.yml b/Taskfile.yml
index d464a8c3..c453b136 100644
--- a/Taskfile.yml
+++ b/Taskfile.yml
@@ -2702 +2702 @@ tasks:
-        GORELEASER_CURRENT_TAG="${TAG}" "${GORELEASER_BIN}" release --skip=publish,sign,validate --clean --release-notes "${NOTES}"
+        GORELEASER_CURRENT_TAG="${TAG}" "${GORELEASER_BIN}" release --snapshot --skip=publish,sign,validate --clean --release-notes "${NOTES}"
```

RED guard: `GOTOOLCHAIN=go1.26.9 go test ./internal/upgrade/ -run '^TestReleaseDryRunRehearsesReleaseNotes$' -count=1 -v` (from `$S`):

```
=== RUN   TestReleaseDryRunRehearsesReleaseNotes
    changie_release_shape_test.go:1648: release:dry-run cmds: token "GORELEASER_CURRENT_TAG=\"${TAG}\" \"${GORELEASER_BIN}\" release --skip=publish,sign,validate --clean --release-notes \"${NOTES}\"" (non-snapshot invocation) not found at or after byte offset 917
    changie_release_shape_test.go:1654: release:dry-run runs a --snapshot line, which skips the notes pipe: "GORELEASER_CURRENT_TAG=\"${TAG}\" \"${GORELEASER_BIN}\" release --snapshot --skip=publish,sign,validate --clean --release-notes \"${NOTES}\""
    changie_release_shape_test.go:1686: release:dry-run: 6 of 7 ordered tokens found; 1 changie invocation
--- FAIL: TestReleaseDryRunRehearsesReleaseNotes (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/upgrade	0.162s
FAIL
```

Byte-clean proof: reverted with `git -C "$S" checkout -- Taskfile.yml`; `git -C "$S" diff --quiet` exit 0

Re-run GREEN after the revert, same command:

```
=== RUN   TestReleaseDryRunRehearsesReleaseNotes
    changie_release_shape_test.go:1686: release:dry-run: 7 of 7 ordered tokens found; 1 changie invocation
--- PASS: TestReleaseDryRunRehearsesReleaseNotes (0.00s)
PASS
ok  	github.com/seanb4t/codegraph-go/internal/upgrade	0.165s
```

## Teardown (05-03 families)

`git worktree remove --force "$S"` and `git worktree prune`, then `git worktree list`:

```
/Volumes/Code/github.com/seanb4t/codegraph-go                        c4adcfd5 [gsd/v0.15.0-milestone]
/Volumes/Code/herdr-worktrees/codegraph-go/worktree-brave-river-a240 2df78be0 [chore/octopusignore]
```

No `p05-scratch-wt` entry remains.

## 05-03 REL-13 rehearsal

Evidence, not a committed test (rule eyva2svrcz). Host: darwin (`go env GOHOSTOS`), with zig, syft, cosign, jq and shasum on PATH. `gh api repos/seanb4t/codegraph-go/git/matching-refs/tags/v0.15 --jq length` (read-only) returned 0 before both runs, so no real v0.15 tag existed. The five `MACOS_*` variables are defined in this host's environment, which would turn notarization on locally (Pitfall 8), so each run is under `env -u` for the five names in .goreleaser.yaml's notarize block; no precondition was added (D-01). Each run's output went to a scratch file; below are the command line and the captured lines only. GoReleaser's own log text is not asserted by anything in this phase (D-14).

Scope (D-14): this rehearsal proves the notes path passed for the tag, the repository's own non-empty guard and a zero exit before any real tag exists; how GoReleaser renders the Release body is GoReleaser's behavior and is not tested in this phase. The sha256 in each evidence line records which file was passed, nothing more.

### Run A, on HEAD (the newest seed)

Command: `env -u MACOS_SIGN_P12 -u MACOS_SIGN_PASSWORD -u MACOS_NOTARY_KEY -u MACOS_NOTARY_KEY_ID -u MACOS_NOTARY_ISSUER_ID GOTOOLCHAIN=go1.26.9 task release:dry-run > "$T" 2>&1; echo "release:dry-run exit=$?" >> "$T"` from the main checkout at `c4adcfd5` (`changie latest` = v0.14.1).

```
RELEASE-NOTES-EVIDENCE tag=v0.14.1 file=.changes/v0.14.1.md bytes=353 sha256=c7703c6606ccff8991260605d9a5ab9a3fc0f9114109baaa300ea5d9f8c5479d
==> goreleaser release --skip=publish,sign,validate --clean --release-notes .changes/v0.14.1.md (GORELEASER_CURRENT_TAG=v0.14.1)
  • release succeeded after 16s
    linux/amd64: dist/codegraph-linux-amd64_linux_amd64_v1/codegraph: ELF 64-bit LSB executable, x86-64, version 1 (SYSV), dynamically linked, interpreter /lib64/ld-linux-x86-64.so.2, for GNU/Linux 2.0.0, with debug_info, not stripped
    linux/arm64: dist/codegraph-linux-arm64_linux_arm64_v8.0/codegraph: ELF 64-bit LSB executable, ARM aarch64, version 1 (SYSV), dynamically linked, interpreter /lib/ld-linux-aarch64.so.1, for GNU/Linux 2.0.0, with debug_info, not stripped
    darwin/amd64: dist/codegraph-darwin-amd64_darwin_amd64_v1/codegraph: Mach-O 64-bit executable x86_64
    darwin/arm64: dist/codegraph-darwin-arm64_darwin_arm64_v8.0/codegraph: Mach-O 64-bit executable arm64
release:dry-run: four correctly-typed binaries produced in one goreleaser release invocation.
release:dry-run exit=0
```

`shasum -a 256 .changes/v0.14.1.md` (the file passed):

```
c7703c6606ccff8991260605d9a5ab9a3fc0f9114109baaa300ea5d9f8c5479d  .changes/v0.14.1.md
```

`git status --porcelain --untracked-files=no` of the main checkout after the run: empty. dist/ is gitignored (`.gitignore:5:/dist/`).

### Run B, after `task release:batch` (the version the first real release PR will carry)

Commands, in a detached scratch worktree whose path contains `p05-scratch-wt`: `(cd "$S" && task release:batch)`, which printed `release:batch: rendered v0.15.0 from 1 fragment(s)` and exited 0, then `(cd "$S" && env -u MACOS_SIGN_P12 -u MACOS_SIGN_PASSWORD -u MACOS_NOTARY_KEY -u MACOS_NOTARY_KEY_ID -u MACOS_NOTARY_ISSUER_ID GOTOOLCHAIN=go1.26.9 task release:dry-run > "$T" 2>&1; echo "release:dry-run exit=$?" >> "$T")`. `changie latest` = v0.15.0.

```
RELEASE-NOTES-EVIDENCE tag=v0.15.0 file=.changes/v0.15.0.md bytes=328 sha256=3dca5a9b89b3aebf73790697c91d4312c347c979bef9072a8c0a763ec7f67456
==> goreleaser release --skip=publish,sign,validate --clean --release-notes .changes/v0.15.0.md (GORELEASER_CURRENT_TAG=v0.15.0)
  • release succeeded after 15s
    linux/amd64: dist/codegraph-linux-amd64_linux_amd64_v1/codegraph: ELF 64-bit LSB executable, x86-64, version 1 (SYSV), dynamically linked, interpreter /lib64/ld-linux-x86-64.so.2, for GNU/Linux 2.0.0, with debug_info, not stripped
    linux/arm64: dist/codegraph-linux-arm64_linux_arm64_v8.0/codegraph: ELF 64-bit LSB executable, ARM aarch64, version 1 (SYSV), dynamically linked, interpreter /lib/ld-linux-aarch64.so.1, for GNU/Linux 2.0.0, with debug_info, not stripped
    darwin/amd64: dist/codegraph-darwin-amd64_darwin_amd64_v1/codegraph: Mach-O 64-bit executable x86_64
    darwin/arm64: dist/codegraph-darwin-arm64_darwin_arm64_v8.0/codegraph: Mach-O 64-bit executable arm64
release:dry-run: four correctly-typed binaries produced in one goreleaser release invocation.
release:dry-run exit=0
```

`shasum -a 256 "$S/.changes/v0.15.0.md"` (the file passed):

```
3dca5a9b89b3aebf73790697c91d4312c347c979bef9072a8c0a763ec7f67456  $S/.changes/v0.15.0.md
```

The worktree's `git status --short` was the same before and after the dry run: the three lines `release:batch` writes (` D .changes/unreleased/Breaking-20260926-165849.599611000.yaml`, ` M CHANGELOG.md`, `?? .changes/v0.15.0.md`), nothing else outside the gitignored dist/. The worktree was then removed with `git worktree remove --force "$S"` and `git worktree prune`.

Deferred CI check (open): linux-cross-canary.yml on pull_request (paths filter includes Taskfile.yml), PR #88, first push after ci(05-03) b390a9d2bda1b7d96aabb7607fc3367974daf11f (comment-only follow-up c4adcfd5ffb43bf13feee5a287cce86bf0bb5d5a); phase verification confirms the run is green before the phase closes. This host's two runs do not cover the CI consumer of release:dry-run; that run does, and this plan does not push to start it.

## 05-04 RED transcript (exemption swap)

Commit: the `test(05-04): expect the release PR's own paths to be housekeeping in both hygiene gates` commit, before
`is_housekeeping`, the `.changes/` prefix and the `.changes/*` shell arm exist. Command, from the repository root:
`python3 -B -m unittest discover -s scripts -p 'test_fragment_policy.py' -v`. `Ran 42 tests`, `FAILED (failures=6)`,
zero `ERROR:` lines. Every failure is an assertion; there is no import or load error.

The four `is_housekeeping` tests fail on their `hasattr(pr_template_policy, "is_housekeeping")` check, because the name
does not exist yet. That proves commit order only (rule x1cjy9vyhq); their own-assertion RED is Family 05-04/e.
`test_exempt_sets_are_the_documented_sets` fails on its own set-equality assertion: `.changes/` is missing from
EXEMPT_PREFIXES. The parity test also fails at RED, on its population floor (3 prefixes parsed, floor 4), because the
shell list has no `.changes/*` arm yet. The plan predicted it would pass at RED; the prefix floor makes it fail, which is
the stronger guard. Its own-assertion RED on the set comparison is Family 05-04/b.

```
FAIL: test_empty_path_list_is_not_housekeeping (test_fragment_policy.HygieneAllowlistTests.test_empty_path_list_is_not_housekeeping)
    self.assertTrue(hasattr(pr_template_policy, "is_housekeeping"))
AssertionError: False is not true

FAIL: test_exempt_sets_are_the_documented_sets (test_fragment_policy.HygieneAllowlistTests.test_exempt_sets_are_the_documented_sets)
AssertionError: Items in the second set but not the first:
'.changes/'

FAIL: test_fragment_only_pr_is_housekeeping (test_fragment_policy.HygieneAllowlistTests.test_fragment_only_pr_is_housekeeping)
    self.assertTrue(hasattr(pr_template_policy, "is_housekeeping"))
AssertionError: False is not true

FAIL: test_hygiene_allowlists_mirror_require_issue_link (test_fragment_policy.HygieneAllowlistTests.test_hygiene_allowlists_mirror_require_issue_link)
    self.assertGreaterEqual(len(prefixes), 4, f"parsed prefixes: {sorted(prefixes)}")
AssertionError: 3 not greater than or equal to 4 : parsed prefixes: ['.github/', '.planning/', 'docs/']

FAIL: test_release_pr_paths_are_housekeeping (test_fragment_policy.HygieneAllowlistTests.test_release_pr_paths_are_housekeeping)
    self.assertTrue(hasattr(pr_template_policy, "is_housekeeping"))
AssertionError: False is not true

FAIL: test_shipped_code_beside_changes_is_not_housekeeping (test_fragment_policy.HygieneAllowlistTests.test_shipped_code_beside_changes_is_not_housekeeping)
    self.assertTrue(hasattr(pr_template_policy, "is_housekeeping"))
AssertionError: False is not true

Ran 42 tests in 0.002s

FAILED (failures=6)
```

## 05-04 GREEN (exemption swap)

Commit: `ci(05-04): exempt the changie release PR by path in both hygiene gates`, after the `test(05-04):` commit above.
`git log --first-parent --no-merges --reverse` over scripts/pr_template_policy.py and scripts/test_fragment_policy.py
since `5a8603f5` orders them test(05-04), ci(05-04). Command, from the repository root:
`python3 -B -m unittest discover -s scripts -p 'test_fragment_policy.py' -v`. The six named tests, from that run:

```
test_empty_path_list_is_not_housekeeping ... ok
test_exempt_sets_are_the_documented_sets ... ok
test_fragment_only_pr_is_housekeeping ... ok
test_hygiene_allowlists_mirror_require_issue_link ... ok
test_release_pr_paths_are_housekeeping ... ok
test_shipped_code_beside_changes_is_not_housekeeping ... ok

Ran 42 tests in 0.001s

OK
```

`task check:fragment-policy` (floor still 36 at this commit) printed
`check:fragment-policy: executed 42 of 42 defined test methods (floor 36)`. `task lint:actions` exited 0.
The three hygiene files hold no census hit, and `.changes/*) ;;` appears once in require-issue-link.yml.

## 05-04 RED transcript (WR-01)

Commit: the `test(05-04): expect linear-time exemption markers with mirrored grammar` commit, before either pattern
changes and before `pr_template_policy.find_exemption` exists. Command, from the repository root:
`python3 -B -m unittest discover -s scripts -p 'test_fragment_policy.py' -v`. `Ran 46 tests`, `FAILED (failures=7)`
(four timing subTests plus the three grammar tests), zero `ERROR:` lines. The run took 13.5 s: the old patterns take
2.9 to 3.8 s per adversarial call, measured below.

Old-pattern measurements (`.search` on the bodies the timing test builds), from the failure messages:

| marker | body | old-pattern time |
|---|---|---|
| changelog-exempt | `<!-- changelog-exempt:` + 2,500 spaces (cubic shape, 2522 chars) | 2.89 s |
| changelog-exempt | `<!-- changelog-exempt: a` + 60,000 spaces (quadratic shape, 60024 chars) | 3.83 s |
| pr-template-exempt | `<!-- pr-template-exempt:` + 2,500 spaces (cubic shape, 2524 chars) | 2.95 s |
| pr-template-exempt | `<!-- pr-template-exempt: a` + 60,000 spaces (quadratic shape, 60026 chars) | 3.82 s |

The timing test fails on its own elapsed-time assertion. The other three tests fail on their
`hasattr(pr_template_policy, "find_exemption")` check, because the name does not exist yet. That proves commit order
only (rule x1cjy9vyhq); their own-assertion RED is Family 05-04/f.

```
FAIL: test_exempt_markers_are_linear_on_adversarial_bodies (test_fragment_policy.MarkerGrammarTests.test_exempt_markers_are_linear_on_adversarial_bodies) (marker='changelog-exempt', shape='cubic shape')
    self.assertLess(
AssertionError: 2.893123999936506 not less than 1.0 : changelog-exempt cubic shape: search took 2.89s on a 2522-char body

FAIL: test_exempt_markers_are_linear_on_adversarial_bodies (test_fragment_policy.MarkerGrammarTests.test_exempt_markers_are_linear_on_adversarial_bodies) (marker='changelog-exempt', shape='quadratic shape')
    self.assertLess(
AssertionError: 3.828044166089967 not less than 1.0 : changelog-exempt quadratic shape: search took 3.83s on a 60024-char body

FAIL: test_exempt_markers_are_linear_on_adversarial_bodies (test_fragment_policy.MarkerGrammarTests.test_exempt_markers_are_linear_on_adversarial_bodies) (marker='pr-template-exempt', shape='cubic shape')
    self.assertLess(
AssertionError: 2.9537973750848323 not less than 1.0 : pr-template-exempt cubic shape: search took 2.95s on a 2524-char body

FAIL: test_exempt_markers_are_linear_on_adversarial_bodies (test_fragment_policy.MarkerGrammarTests.test_exempt_markers_are_linear_on_adversarial_bodies) (marker='pr-template-exempt', shape='quadratic shape')
    self.assertLess(
AssertionError: 3.8166788339149207 not less than 1.0 : pr-template-exempt quadratic shape: search took 3.82s on a 60026-char body

FAIL: test_marker_reason_starting_with_dash_does_not_exempt (test_fragment_policy.MarkerGrammarTests.test_marker_reason_starting_with_dash_does_not_exempt)
    self.assertTrue(hasattr(pr_template_policy, "find_exemption"))
AssertionError: False is not true

FAIL: test_marker_semantics_mirror_pr_template_exempt (test_fragment_policy.MarkerGrammarTests.test_marker_semantics_mirror_pr_template_exempt)
    self.assertTrue(hasattr(pr_template_policy, "find_exemption"))
AssertionError: False is not true

FAIL: test_unterminated_marker_does_not_exempt (test_fragment_policy.MarkerGrammarTests.test_unterminated_marker_does_not_exempt)
    self.assertTrue(hasattr(pr_template_policy, "find_exemption"))
AssertionError: False is not true

Ran 46 tests in 13.494s

FAILED (failures=7)
```

## 05-04 GREEN (WR-01)

Commits: `ci(05-04): make both exemption markers linear and keep their grammar in lockstep`, then
`ci(05-04): raise the fragment-policy floor to the 46 defined tests`, both after the `test(05-04):` commit above.
`python3 -B -m unittest discover -s scripts -p 'test_fragment_policy.py' -v` prints `Ran 46 tests` and `OK`; the existing
GATE-02 tests, including `test_marker_grammar_mirrors_pr_template_exempt`, are unchanged and green. The four WR-01 tests:

```
test_exempt_markers_are_linear_on_adversarial_bodies (test_fragment_policy.MarkerGrammarTests.test_exempt_markers_are_linear_on_adversarial_bodies) ... ok
test_marker_reason_starting_with_dash_does_not_exempt (test_fragment_policy.MarkerGrammarTests.test_marker_reason_starting_with_dash_does_not_exempt) ... ok
test_marker_semantics_mirror_pr_template_exempt (test_fragment_policy.MarkerGrammarTests.test_marker_semantics_mirror_pr_template_exempt) ... ok
test_unterminated_marker_does_not_exempt (test_fragment_policy.MarkerGrammarTests.test_unterminated_marker_does_not_exempt) ... ok
```

`task check:fragment-policy`:

```
Ran 46 tests in 0.002s

OK
check:fragment-policy: executed 46 of 46 defined test methods (floor 46)
```

Old versus new `.search` time on the same four adversarial bodies (new patterns, measured with `time.perf_counter`):

| marker | body | old pattern | new pattern |
|---|---|---|---|
| changelog-exempt | 2,500 spaces after the opener (2522 chars) | 2.89 s | 0.018 ms |
| changelog-exempt | `a` then 60,000 spaces (60024 chars) | 3.83 s | 0.291 ms |
| pr-template-exempt | 2,500 spaces after the opener (2524 chars) | 2.95 s | 0.012 ms |
| pr-template-exempt | `a` then 60,000 spaces (60026 chars) | 3.82 s | 0.293 ms |

The test bound is 1.0 s per call, so the margin over the new patterns is about three orders of magnitude.
`GOTOOLCHAIN=go1.26.9 go test ./internal/upgrade/ -count=1` stays `ok` after the Taskfile.yml change.
