# 06-EVIDENCE — First Release Through the Chain

- **Phase:** 06-first-release-through-the-chain (plan 06-01)
- **Date:** 2026-10-17
- **Branch:** `gsd/v0.15.0-changie-release-management`
- **Scope:** live observations of the chain with read-only commands; the D-12 evidence summary and sign-off close the file.

Convention: every transcript is pasted into a fenced `text` block, with the command line prefixed `$ `. Release-note text contains `## ` lines, so it stays inside a fence or it would become a heading. Names used throughout: REPO is `seanb4t/codegraph-go`; M is #88's squash commit; N is the release PR number; TAG is the derived version; SHA is the release PR's squash commit (the tag commit, known after the Task 2 merge).

## Pre-flight

```text
$ gh pr view 88 --repo seanb4t/codegraph-go --json state,mergedAt,mergeCommit
{"mergeCommit":{"oid":"ab49d80ef3dc933e9a73096268ba12db8c0098b3"},"mergedAt":"2026-10-17T17:15:44Z","state":"MERGED"}

M = ab49d80ef3dc933e9a73096268ba12db8c0098b3

$ GIT_TERMINAL_PROMPT=0 git fetch origin
From https://github.com/seanb4t/codegraph-go
 * [new branch]        changie/release -> origin/changie/release

$ git branch --show-current
gsd/v0.15.0-changie-release-management
$ git rev-parse HEAD origin/main
ab49d80ef3dc933e9a73096268ba12db8c0098b3
ab49d80ef3dc933e9a73096268ba12db8c0098b3
$ git merge-base --is-ancestor ab49d80ef3dc933e9a73096268ba12db8c0098b3 HEAD && echo ANCESTOR_OK
ANCESTOR_OK
$ git diff --name-only origin/main...HEAD -- ':(exclude).planning/'
(no output: the branch carries no non-.planning change)

$ git show origin/main:.planning/REQUIREMENTS.md | rg '\*\*SHIP-02\*\*'
- [ ] **SHIP-02**: The resulting tag is authored by the GitHub App, `release.yml` ran on that tag push, and the GitHub Release body is content-equal to `.changes/v0.15.0.md`: identical after both sides are normalized the same way (CRLF→LF, trailing whitespace stripped on each line, leading and trailing blank lines stripped; amended by Phase 6 D-11, 2026-10-17)
$ git cat-file -e origin/main:.planning/phases/06-first-release-through-the-chain/06-01-PLAN.md && echo PLAN_ON_MAIN
PLAN_ON_MAIN

$ task changie -- latest
v0.14.1
$ task changie -- next auto
v0.15.0
$ git ls-tree --name-only ab49d80ef3dc933e9a73096268ba12db8c0098b3 -- .changes/unreleased/ | rg -o '\.yaml$' | wc -l
       1
$ git ls-tree --name-only ab49d80ef3dc933e9a73096268ba12db8c0098b3 -- .changes/unreleased/
.changes/unreleased/.gitkeep
.changes/unreleased/Breaking-20260926-165849.599611000.yaml
```

The fragment count is 1, the one `Breaking` fragment (D-06).

next auto: v0.15.0

## SHIP-01 release PR

### release-pr.yml on M

```text
$ gh run list --repo seanb4t/codegraph-go --workflow release-pr.yml --commit ab49d80ef3dc933e9a73096268ba12db8c0098b3 --json databaseId,status,conclusion,event,url
[{"conclusion":"success","databaseId":39052978056,"event":"push","status":"completed","url":"https://github.com/seanb4t/codegraph-go/actions/runs/39052978056"}]

$ gh run view 39052978056 --repo seanb4t/codegraph-go --log | rg 'release:batch'
release PR	Batch fragments into a release	2026-10-17T17:16:45.7199278Z task release:batch
release PR	Batch fragments into a release	2026-10-17T17:16:52.2158715Z release:batch: rendered v0.15.0 from 1 fragment(s)
```

Run: https://github.com/seanb4t/codegraph-go/actions/runs/39052978056 (event push, conclusion success).

### The release PR

```text
$ gh pr list --repo seanb4t/codegraph-go --head changie/release --state all --json number,title,author,isDraft,url
[{"author":{"is_bot":true,"login":"app/fzy-release-please"},"isDraft":false,"number":101,"title":"chore(main): release v0.15.0","url":"https://github.com/seanb4t/codegraph-go/pull/101"}]

$ gh pr view 101 --repo seanb4t/codegraph-go --json number,url,title,isDraft,author,headRefName,mergeable,mergeStateStatus
{"author":{"is_bot":true,"login":"app/fzy-release-please"},"headRefName":"changie/release","isDraft":false,"mergeStateStatus":"CLEAN","mergeable":"MERGEABLE","number":101,"title":"chore(main): release v0.15.0","url":"https://github.com/seanb4t/codegraph-go/pull/101"}

$ gh pr diff 101 --repo seanb4t/codegraph-go --name-only
.changes/unreleased/Breaking-20260926-165849.599611000.yaml
.changes/v0.15.0.md
CHANGELOG.md
```

N = 101. TAG = v0.15.0, equal to the recorded `task changie -- next auto` output as an exact string. PR URL: https://github.com/seanb4t/codegraph-go/pull/101. The PR body is the rendered `.changes/v0.15.0.md`:

```text
$ gh pr view 101 --repo seanb4t/codegraph-go --json body --jq .body
## [v0.15.0](https://github.com/seanb4t/codegraph-go/releases/tag/v0.15.0) — 2026-10-17

### Breaking

- Removed the hidden `query` and `unlock` commands, so running either now fails as an unknown command; run `search --full <term>` or `daemon unlock [path]` instead. ([#88](https://github.com/seanb4t/codegraph-go/pull/88))
```

### Required checks (05-UAT test 3)

First pass, ci.yml run https://github.com/seanb4t/codegraph-go/actions/runs/39053062791, attempt 1: eight required checks passed and `test` failed. The failure log was read through `gh api repos/seanb4t/codegraph-go/actions/jobs/117455402007/logs`, because `gh run view --log-failed` refuses while the run is in progress:

```text
2026-10-17T17:18:59.9944070Z --- FAIL: TestColorFlagInvalidValue (0.02s)
2026-10-17T17:18:59.9963070Z     colorflag_test.go:205: status --color=never output differs from the bare (no-flag) plain output:
2026-10-17T17:18:59.9978171Z           DB Size:   0.02 MB      (--- bare ---)
2026-10-17T17:18:59.9997774Z           DB Size:   0.01 MB      (--- never ---)
2026-10-17T17:19:00.0020534Z FAIL	github.com/seanb4t/codegraph-go/internal/cli	3.195s
2026-10-17T17:19:42.5965809Z ##[error]Process completed with exit code 201.
```

That is the only `--- FAIL` in the job (count 1), and the only differing line in the two outputs is `DB Size:`. This is the known nondeterministic flake recorded in engram w2t54bym10 (pebble flush timing changes the rounded size), not a code or config defect. The orchestrator pre-authorized the one re-run for exactly this signature.

D-10 re-runs: one. Run 39053062791, `gh run rerun 39053062791 --repo seanb4t/codegraph-go --failed`. Cause: nondeterministic `DB Size:` byte difference in `internal/cli` `TestColorFlagInvalidValue` (flake, engram w2t54bym10). Attempt 2 concluded `success`. No other job was re-run.

```text
$ gh run view 39053062791 --repo seanb4t/codegraph-go --json status,conclusion,attempt
{"attempt":2,"conclusion":"success","status":"completed"}

$ gh api repos/seanb4t/codegraph-go/rulesets/20157557 --jq '[.rules[]|select(.type=="required_status_checks")|.parameters.required_status_checks[]]|length'
9

$ gh pr checks 101 --repo seanb4t/codegraph-go --required --json name,bucket
[{"bucket":"pass","name":"test"},{"bucket":"pass","name":"reproducibility (double-build hash-diff, DIST-04)"},{"bucket":"pass","name":"actionlint (workflow static analysis)"},{"bucket":"pass","name":"fragment-required"},{"bucket":"pass","name":"govulncheck (DIST-03, blocking)"},{"bucket":"pass","name":"perf regression gate (PERF-02, INDX-06)"},{"bucket":"pass","name":"tmux e2e (real-pty harness, TTY-01..TTY-07)"},{"bucket":"pass","name":"pr-title"},{"bucket":"pass","name":"goreleaser check (config validation, DIST-01)"}]
```

Nine of nine required checks report `pass`, equal to the protect-main count of 9.

### The #88 merge did not fire release-tag.yml (05-UAT test 4, first half)

```text
$ gh run list --repo seanb4t/codegraph-go --workflow release-tag.yml --commit ab49d80ef3dc933e9a73096268ba12db8c0098b3 --json databaseId,conclusion
[]
```

Outcome: no release-tag.yml run on M, as RESEARCH section 5 predicted (the #88 merge did not change `CHANGELOG.md`). The positive control for the empty result is the release-pr.yml run found by the same `--commit` query on M above (run 39052978056).
