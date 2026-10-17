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

## Release-PR merge

Task 2 checkpoint reply (verbatim): `PR merged`

```text
$ gh pr view 101 --repo seanb4t/codegraph-go --json state,mergedBy,mergeCommit,mergedAt,title
{"mergeCommit":{"oid":"0246de0e391b7de98052389800f8347fb9657d3b"},"mergedAt":"2026-10-17T17:35:19Z","mergedBy":{"id":"MDQ6VXNlcjQ2Nzg=","is_bot":false,"login":"seanb4t","name":"Sean Brandt"},"state":"MERGED","title":"chore(main): release v0.15.0"}
```

State MERGED, `mergedBy.login` `seanb4t` (the maintainer, not the executor). SHA = 0246de0e391b7de98052389800f8347fb9657d3b.

## SHIP-01 changie latest on main

```text
$ GIT_TERMINAL_PROMPT=0 git fetch origin
   ab49d80e..0246de0e  main       -> origin/main
 * [new tag]           v0.15.0    -> v0.15.0

$ git merge --no-ff -F <msg> origin/main
Merge made by the 'ort' strategy.
 .changes/unreleased/Breaking-20260926-165849.599611000.yaml | 5 -----
 .changes/v0.15.0.md                                         | 6 ++++++
 CHANGELOG.md                                                | 6 ++++++
 3 files changed, 12 insertions(+), 5 deletions(-)

$ git log --oneline -3
bdea3c83 chore(06-01): merge main into the follow-up branch after the release PR merged
0246de0e (tag: v0.15.0, origin/main, origin/HEAD) chore(main): release v0.15.0 (#101)
c4d56838 docs(06-01): record the pre-flight and the App-opened release PR (SHIP-01)

$ git rev-list --parents -n 1 HEAD
bdea3c83547f5ee69df897ae45cd9b9ea653b3b5 c4d56838718e05d047a81c51894e82f504ba504c 0246de0e391b7de98052389800f8347fb9657d3b

$ git diff --quiet 0246de0e391b7de98052389800f8347fb9657d3b HEAD -- .changes CHANGELOG.md && echo DIFF_QUIET_OK
DIFF_QUIET_OK

$ task changie -- latest
v0.15.0
```

The merge commit has two parents; the second is the fetched origin/main (0246de0e). `task changie -- latest` prints v0.15.0, equal to TAG.

## SHIP-02 App-authored tag and release.yml

### release-pr.yml on SHA (zero-fragment no-op)

```text
$ gh run list --repo seanb4t/codegraph-go --workflow release-pr.yml --commit 0246de0e391b7de98052389800f8347fb9657d3b --json databaseId,status,conclusion,event,url
[{"conclusion":"success","databaseId":39054420946,"event":"push","status":"completed","url":"https://github.com/seanb4t/codegraph-go/actions/runs/39054420946"}]

$ gh run view 39054420946 --repo seanb4t/codegraph-go --log | rg 'release:batch|nothing to release'
release PR	Batch fragments into a release	2026-10-17T17:36:24.4178049Z release:batch: no unreleased fragments under .changes/unreleased/ — nothing to release

$ gh pr list --repo seanb4t/codegraph-go --head changie/release --state all --json number,state
[{"number":101,"state":"MERGED"}]
```

Run: https://github.com/seanb4t/codegraph-go/actions/runs/39054420946. Exactly one `changie/release` PR exists and it is MERGED; no second release PR opened. This is not a WR-01 observation: no `changie/release` PR was open for the close step to close.

### release-tag.yml on SHA

```text
$ gh run view 39054420866 --repo seanb4t/codegraph-go --json conclusion,url,jobs --jq '{conclusion,url,jobs:[.jobs[]|{name,conclusion}]}'
{"conclusion":"success","jobs":[{"conclusion":"success","name":"pre-tag 4-target go list sanity sweep"},{"conclusion":"success","name":"tag"}],"url":"https://github.com/seanb4t/codegraph-go/actions/runs/39054420866"}

$ gh run view 39054420866 --repo seanb4t/codegraph-go --log | rg 'release:tag'
tag	Tag the release	2026-10-17T17:36:20.3851786Z release:tag: pushed v0.15.0 at 0246de0e391b7de98052389800f8347fb9657d3b

$ gh api repos/seanb4t/codegraph-go/commits/v0.15.0 --jq .sha
0246de0e391b7de98052389800f8347fb9657d3b
```

Run: https://github.com/seanb4t/codegraph-go/actions/runs/39054420866. The tag resolves to SHA.

### release.yml on the tag push

```text
$ gh run list --repo seanb4t/codegraph-go --workflow release.yml --branch v0.15.0 --event push --json databaseId,status,conclusion,attempt,url
[{"attempt":1,"conclusion":"success","databaseId":39054491964,"status":"completed","url":"https://github.com/seanb4t/codegraph-go/actions/runs/39054491964"}]

$ gh api repos/seanb4t/codegraph-go/actions/runs/39054491964 --jq '{event,head_branch,head_sha,run_attempt,actor:.actor.login,triggering_actor:.triggering_actor.login}'
{"actor":"fzy-release-please[bot]","event":"push","head_branch":"v0.15.0","head_sha":"0246de0e391b7de98052389800f8347fb9657d3b","run_attempt":1,"triggering_actor":"fzy-release-please[bot]"}

$ gh api repos/seanb4t/codegraph-go/actions/runs/39054491964/attempts/1 --jq '{run_attempt,actor:.actor.login,triggering_actor:.triggering_actor.login}'
{"actor":"fzy-release-please[bot]","run_attempt":1,"triggering_actor":"fzy-release-please[bot]"}

$ gh release view v0.15.0 --repo seanb4t/codegraph-go --json tagName,isDraft,isPrerelease,publishedAt,url,author
{"author":{"id":"MDM6Qm90NDE4OTgyODI=","login":"github-actions[bot]"},"isDraft":false,"isPrerelease":false,"publishedAt":"2026-10-17T17:40:18Z","tagName":"v0.15.0","url":"https://github.com/seanb4t/codegraph-go/releases/tag/v0.15.0"}
```

Run: https://github.com/seanb4t/codegraph-go/actions/runs/39054491964 (event push, head_branch v0.15.0, head_sha SHA, conclusion success, one job `build, sign, SBOM, and publish release`, attempt 1). Run actor and attempt 1 `triggering_actor` are both `fzy-release-please[bot]`, so neither `seanb4t` nor `github-actions[bot]` pushed the tag. The Release is published (not draft, not prerelease) at 2026-10-17T17:40:18Z; its record author `github-actions[bot]` is the GoReleaser publish step's GITHUB_TOKEN identity, a different act from authoring the tag.

D-10 re-runs: one in plan 06-01, in Task 1 (ci.yml run 39053062791, flake `TestColorFlagInvalidValue` DB Size, engram w2t54bym10; see `Required checks` above). None in Task 3: release-pr.yml 39054420946, release-tag.yml 39054420866 and release.yml 39054491964 each ran once (attempt 1) and concluded success. The ci.yml run on SHA (39054420975, attempt 1) also concluded success.

## SHIP-02 Release body (D-11)

```text
$ norm() { tr -d '\r' | sed -E 's/[[:space:]]+$//' | awk '{l[NR]=$0} END{s=1; while(NR>=s&&l[s]=="")s++; e=NR; while(e>=s&&l[e]=="")e--; for(i=s;e>=i;i++)print l[i]}'; }
$ gh api repos/seanb4t/codegraph-go/releases/tags/v0.15.0 --jq .body | norm > body.txt
$ git show 0246de0e391b7de98052389800f8347fb9657d3b:.changes/v0.15.0.md | norm > file.txt
$ test -s body.txt && test -s file.txt && cmp body.txt file.txt
CONTENT-EQUAL lines=5
```

The normalized file side (identical to the normalized Release body):

```text
## [v0.15.0](https://github.com/seanb4t/codegraph-go/releases/tag/v0.15.0) — 2026-10-17

### Breaking

- Removed the hidden `query` and `unlock` commands, so running either now fails as an unknown command; run `search --full <term>` or `daemon unlock [path]` instead. ([#88](https://github.com/seanb4t/codegraph-go/pull/88))
```

Both sides are non-empty and `cmp`-equal under the D-11 normalization (CR removed, trailing whitespace stripped per line, leading and trailing blank lines stripped).

## SHIP-03 post-release-verify

```text
$ task changie -- latest
v0.15.0
$ gh api repos/seanb4t/codegraph-go/commits/v0.15.0 --jq .sha
0246de0e391b7de98052389800f8347fb9657d3b
$ gh api repos/seanb4t/codegraph-go/releases/latest --jq .tag_name
v0.15.0
$ git diff --stat v0.14.0 0246de0e391b7de98052389800f8347fb9657d3b -- internal/upgrade/verify.go
(no output: the cosign identity file is unchanged between v0.14.0 and the tag commit)

$ gh run list --repo seanb4t/codegraph-go --workflow post-release-verify.yml --event workflow_run --limit 20 --json databaseId,status,conclusion,headSha,createdAt
(the only entry whose headSha is 0246de0e391b7de98052389800f8347fb9657d3b)
{"conclusion":"success","createdAt":"2026-10-17T17:40:27Z","databaseId":39054786827,"headSha":"0246de0e391b7de98052389800f8347fb9657d3b","status":"completed"}

$ gh run view 39054786827 --repo seanb4t/codegraph-go --json conclusion,url,jobs --jq '.conclusion, .url, (.jobs[]|"\(.name) => \(.conclusion)")'
success
https://github.com/seanb4t/codegraph-go/actions/runs/39054786827
resolve and validate the tag under verification => success
Gatekeeper verdict (darwin/arm64) => success
Gatekeeper verdict (darwin/amd64) => success
self-upgrade proof (darwin/arm64) => success
self-upgrade proof (linux/amd64) => success
verify supply-chain claims against the published release => success
notarized suite proof (darwin/arm64) => success
```

Run: https://github.com/seanb4t/codegraph-go/actions/runs/39054786827, conclusion `success`, 7 jobs, all `success`, selected by `headSha` equal to the tag commit. GitHub's latest release is `v0.15.0`, equal to TAG (A4 holds). `internal/upgrade/verify.go` is byte-identical between v0.14.0 and the tag commit, so the cosign identity is unchanged.

## SHIP-03 upgrade darwin/arm64

The published v0.14.0 raw asset, run with `HOME` set to an empty scratch dir that is also the cwd. The scratch dir is under `/private/tmp/claude-501/` and was removed afterwards. The Homebrew `codegraph` was not run.

```text
$ shasum -a 256 "$HOME/.claude/skills/codegraph/.codegraph-manifest.json"   # before, real HOME
eb80e383fa7a01ac4292ce0e2a740093a2a69a52d4fe971ba36fd021bbcd7b45  /Users/sean/.claude/skills/codegraph/.codegraph-manifest.json
$ W=$(mktemp -d)  # empty scratch dir, also HOME and cwd
W=/private/tmp/claude-501/p06/w.cADiEu
$ gh release download v0.14.0 -p codegraph_v0.14.0_darwin_arm64 --dir $W
$ HOME=$W ./codegraph --version
codegraph version v0.14.0 (commit 80377d2805415988cc3bc07e111132f8dad697e5, built 2026-09-20T13:06:18Z)
$ HOME=$W ./codegraph upgrade --check
codegraph v0.15.0 is available (current: v0.14.0)
$ HOME=$W ./codegraph upgrade
upgraded to v0.15.0
$ HOME=$W ./codegraph --version
codegraph version v0.15.0 (commit 0246de0e391b7de98052389800f8347fb9657d3b, built 2026-10-17T17:35:19Z)
$ shasum -a 256 "$HOME/.claude/skills/codegraph/.codegraph-manifest.json"   # after, real HOME
eb80e383fa7a01ac4292ce0e2a740093a2a69a52d4fe971ba36fd021bbcd7b45  /Users/sean/.claude/skills/codegraph/.codegraph-manifest.json
$ rm -rf $W
```

The manifest sha256 is the same before and after. No output line named a path under the real `~/.claude`. The upgraded binary reports the tag and the tag commit.

## SHIP-03 upgrade linux/amd64

Run on an ephemeral Namespace devbox (D-13 fallback; OrbStack amd64 emulation fails with `exec format error`, RESEARCH section 7).

Devbox creation, exact flags (checked against `devbox create --help` and the devboxes skill):

```text
$ devbox create --ephemeral --platform linux/amd64 --no_checkout --name p06-upgrade-amd64 --purpose 'v0.14.0 upgrade proof (Phase 6 SHIP-03)' --image builtin:default --size s
```

- Image `builtin:default` (Ubuntu 24.04.3 LTS, glibc), size `s` (4 vCPU / 8 GB). No `--setup_github`, so no GitHub credential reached the devbox (T-06-09).
- The first `devbox create` attempt failed with `Not logged in to Namespace` (auth gate). `devbox auth check-login` printed nothing and exited 0 in that state, so it is not a reliable login test. `devbox login` then completed the browser flow and reported `You are now logged into workspace "Personal"`; the create was repeated and succeeded. `devbox create` also wrote a `p06-upgrade-amd64.devbox.namespace` entry into `~/.ssh/config`; `rg p06-upgrade ~/.ssh/config` finds nothing after teardown.
- `ca-certificates` install: not needed. curl fetched the asset over HTTPS on the stock image (A2 holds).

```text
$ devbox exec p06-upgrade-amd64 -- sh -c 'set -e; cat /etc/os-release | head -2; S=$(mktemp -d); cd "$S"; export HOME="$S"; uname -m; curl -fsSLo codegraph https://github.com/seanb4t/codegraph-go/releases/download/v0.14.0/codegraph_v0.14.0_linux_amd64; chmod +x codegraph; ./codegraph --version; ./codegraph upgrade --check; ./codegraph upgrade; ./codegraph --version'
PRETTY_NAME="Ubuntu 24.04.3 LTS"
NAME="Ubuntu"
x86_64
codegraph version v0.14.0 (commit 80377d2805415988cc3bc07e111132f8dad697e5, built 2026-09-20T13:06:18Z)
codegraph v0.15.0 is available (current: v0.14.0)
upgraded to v0.15.0
codegraph version v0.15.0 (commit 0246de0e391b7de98052389800f8347fb9657d3b, built 2026-10-17T17:35:19Z)
```

Teardown:

```text
$ devbox expire p06-upgrade-amd64 --force
done.
$ devbox list
No devbox available yet. Try running `devbox create`.
```

`devbox list` no longer shows `p06-upgrade-amd64`.

## SHIP-04 SEED-005 and capability tag

Capability record, confirmed live rather than copied from RESEARCH (D-14):

```text
$ jq '.entries.changie|{version,source}' .gsd-capabilities.json
{
  "version": "0.1.2",
  "source": "https://github.com/seanb4t/gsd-capability-changie.git#v0.1.2"
}
$ git -C .gsd/capabilities/changie rev-parse HEAD
2631063f05fffc7152d70a9c98eb0b7479a0029a
$ gh api repos/seanb4t/gsd-capability-changie/commits/v0.1.2 --jq .sha
2631063f05fffc7152d70a9c98eb0b7479a0029a
$ rg -n -F 'gsd-capability-changie.git#' CONTRIBUTING.md
233:gsd-tools capability install https://github.com/seanb4t/gsd-capability-changie.git#v0.1.2 --scope project
239:gsd-tools capability install https://github.com/seanb4t/gsd-capability-changie.git#v0.1.2 --scope global
```

All three name tag `v0.1.2`, and the installed checkout commit equals the tag's commit in the capability repo. The tag is `v0.1.2`, commit `2631063f05fffc7152d70a9c98eb0b7479a0029a`.

Release facts:

```text
$ gh release view v0.15.0 --repo seanb4t/codegraph-go --json url,publishedAt
{"publishedAt":"2026-10-17T17:40:18Z","url":"https://github.com/seanb4t/codegraph-go/releases/tag/v0.15.0"}
```

Release date: 2026-10-17. Every `trigger_when` condition is recorded above before the annotation: an App-authored tag and the release.yml run (`## SHIP-02 App-authored tag and release.yml`), a content-equal body (`## SHIP-02 Release body (D-11)`), and a green post-release-verify (`## SHIP-03 post-release-verify`).

SEED-005 annotation (one prose paragraph at the end of `## When to Surface`; no heading, no frontmatter key; `status: dormant` and `trigger_when` unchanged):

```text
$ git diff origin/main --numstat -- .planning/seeds/SEED-005-changie-template-for-other-repos.md
8	0	.planning/seeds/SEED-005-changie-template-for-other-repos.md
$ git diff origin/main -- .planning/seeds/SEED-005-changie-template-for-other-repos.md
@@ -26,3 +26,11 @@ with small edits.
   and `plugin.json` (use changie `replacements` — they run on `merge`); router-hosts relies on
   `squash_merge_commit_message=PR_BODY` carrying a `gate_status:` trailer — unaffected, but
   verify the fragment gate's exemption marker does not collide.
+
+Trigger fired 2026-10-17: codegraph-go released v0.15.0 through the changie chain
+(https://github.com/seanb4t/codegraph-go/releases/tag/v0.15.0). The GitHub App pushed the tag,
+release.yml ran on it, the Release body is content-equal to `.changes/v0.15.0.md` (Phase 6
+D-11), post-release-verify is green, and an installed v0.14.0 upgraded to it on darwin/arm64 and
+linux/amd64. The phase-close fragment question above is answered by gsd-capability-changie
+v0.1.2 (commit 2631063f05fffc7152d70a9c98eb0b7479a0029a), the capability this release ran with.
+Evidence: `.planning/phases/06-first-release-through-the-chain/06-EVIDENCE.md`.
```

Upstream gap: no gsd-tools verb writes a post-plant annotation on a seed (`list-seeds` is read-only; `plant-seed --enrich` rewrites only trigger, scope and why). The paragraph is a value inside an existing section; no structure was invented.

D-06 check (this branch adds nothing under `.changes/`):

```text
$ git diff --name-only origin/main...HEAD -- .changes/
(no output)
$ ls -A .changes/unreleased/
.gitkeep
```

## Evidence summary

| Criterion | Requirement | Observed | Evidence section | Verdict |
|---|---|---|---|---|
| ROADMAP Phase 6 SC1 | SHIP-01 | The App (`app/fzy-release-please`) opened release PR #101 `chore(main): release v0.15.0` from the one `Breaking` fragment; 9 of 9 required checks passed; the maintainer squash-merged it; `task changie -- latest` prints `v0.15.0`, equal to the pre-merge `next auto` | `## Pre-flight`, `## SHIP-01 release PR`, `## Release-PR merge`, `## SHIP-01 changie latest on main` | PASS |
| ROADMAP Phase 6 SC2 | SHIP-02 | Tag `v0.15.0` at `0246de0e391b7de98052389800f8347fb9657d3b`; release.yml run 39054491964 on the tag push has actor and attempt-1 triggering actor `fzy-release-please[bot]`; the Release body is content-equal to `.changes/v0.15.0.md` under the D-11 normalization (`CONTENT-EQUAL lines=5`) | `## SHIP-02 App-authored tag and release.yml`, `## SHIP-02 Release body (D-11)` | PASS |
| ROADMAP Phase 6 SC3 | SHIP-03 | post-release-verify run 39054786827: `success`, 7 of 7 jobs `success`, headSha is the tag commit; GitHub latest is `v0.15.0`; `internal/upgrade/verify.go` unchanged since v0.14.0; v0.14.0 upgraded to `v0.15.0` on darwin/arm64 (isolated HOME, manifest sha256 unchanged) and on linux/amd64 (`x86_64` Namespace devbox, torn down) | `## SHIP-03 post-release-verify`, `## SHIP-03 upgrade darwin/arm64`, `## SHIP-03 upgrade linux/amd64` | PASS |
| ROADMAP Phase 6 SC4 | SHIP-04 | SEED-005 gained one prose paragraph in `## When to Surface` (8 lines added, 0 deleted); capability `gsd-capability-changie` `v0.1.2` at commit `2631063f05fffc7152d70a9c98eb0b7479a0029a` is recorded, equal to the tag commit and to CONTRIBUTING.md | `## SHIP-04 SEED-005 and capability tag` | PASS |

Discharged 05-UAT Deferred Follow-Ups (05-UAT.md is tool-owned and was not edited):

- `05-UAT test 3` (the App opens the release PR and its required checks report): see `## SHIP-01 release PR`. PR #101 was opened by the App and 9 of 9 required checks report `pass`.
- `05-UAT test 4` (the path filter fires, the App-token tag push starts release.yml, and the milestone merge is a no-op): see `## SHIP-01 release PR` (the #88 merge fired no release-tag.yml run) and `## SHIP-02 App-authored tag and release.yml` (the release squash merge fired release-tag.yml, the App pushed the tag, release.yml ran on it).

D-10 re-runs: one. ci.yml run 39053062791 on the release PR (plan 06-01 Task 1), `gh run rerun --failed`, cause: the known nondeterministic `DB Size:` difference in `internal/cli` `TestColorFlagInvalidValue` (engram w2t54bym10); attempt 2 succeeded. None in plan 06-02: post-release-verify run 39054786827 was not re-run, and neither upgrade leg was repeated to reach a pass. The one failed `devbox create` (not logged in) was an auth gate, not a CI step.

Assumption outcomes (RESEARCH Assumptions Log):

- A1 (`APP_ID` is `fzy-release-please`): held. The release PR author is `app/fzy-release-please` and the tag-push run actor is `fzy-release-please[bot]`.
- A2 (devbox image has curl, glibc, CA certificates, outbound HTTPS): held. `builtin:default` (Ubuntu 24.04.3 LTS) fetched the asset with no `ca-certificates` install.
- A3 (GoReleaser puts `--release-notes` content in the Release body verbatim): held. The body is content-equal (`CONTENT-EQUAL lines=5`).
- A4 (GoReleaser marks the new Release as latest): held. `gh api repos/seanb4t/codegraph-go/releases/latest --jq .tag_name` prints `v0.15.0`, and the unpinned `codegraph upgrade --check` offered it.

Carry-forward (D-06): before the follow-up PR merges, `git diff --name-only origin/main...HEAD -- .changes/unreleased/` must still print nothing. The capability's verify:post at this phase's close must not write a fragment, or that merge would open a new release PR. Observed now: it prints nothing, and `.changes/unreleased/` holds only `.gitkeep`.

## Sign-off

The maintainer reviewed the `## Evidence summary` table (SC1 to SC4, all PASS) and the SEED-005 diff, as presented by the orchestrator in this session, and replied on 2026-10-17 (D-12, the blocking-human checkpoint in 06-02 Task 3):

> and approved

Signed off by: the maintainer (seanb4t), 2026-10-17. The reply reports no issues. The sign-off covers SC2 and SC3's human checkpoint and the evidence summary as a whole.
