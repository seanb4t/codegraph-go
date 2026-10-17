# Phase 6: First Release Through the Chain - Research

**Researched:** 2026-10-17
**Domain:** Running an already-built release chain (changie -> GitHub App -> GoReleaser -> cosign) once, live, and recording the evidence
**Confidence:** HIGH on the chain's static shape and on the upgrade path; MEDIUM on live GitHub behavior that has never run in this repo (App-opened release PR, App-pushed tag, GoReleaser creating the Release with `--release-notes`)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** **Just make it work. No space shuttle.** The maintainer said this during this discussion, repeating Phase 5 D-01. Phase 6 runs the chain as built and records what happened. The planner adds no new workflow, guard, test, harness or verification infrastructure. Each step uses the plainest tool that answers it: `gh`, `curl`, `docker`, `cmp`. Plans stay few and short.
- **D-02:** **Plan first; CONTEXT and PLAN ride #88.** `/gsd-plan-phase 6` runs on `gsd/v0.15.0-milestone` before the merge, so this phase's plan is reviewed inside the milestone PR.
- **D-03:** **The #88 merge is a precondition, outside `/gsd-execute-phase 6`.** Pre-merge work runs on the milestone branch, in this order: (1) the four `/gsd-quick` fix clusters (D-07); (2) the pre-merge audit (D-05), plus a fix for anything it finds; (3) the milestone issue and `Closes #N` (D-08); (4) `gh pr ready 88`, never `/gsd-ship`'s create step (STATE); (5) the maintainer squash-merges #88. Execution then starts on a **fresh follow-up branch cut from `main`**. Executor commits never land on the squash-merged, dead milestone branch. Skip execute-phase's `handle_branching`, as the STATE carry-over already says.
- **D-04:** **One planning-only follow-up PR carries everything after the merge.** That is Phase 6's SUMMARY, VERIFICATION and UAT, the final milestone audit, and the milestone close/archive. `fragment-required` passes a `.planning/`-only diff with no marker, and `release-pr.yml` no-ops because no fragments remain after the release.
- **D-05:** **The milestone audit runs twice.** A pre-merge pass audits Phases 1-5, with Phase 6 showing as pending, so cross-phase gaps get fixed inside #88 before it starts the chain. A final pass runs after Phase 6 verifies, on the follow-up branch, covering all 29 requirements.
- **D-06:** **v0.15.0's notes are the one existing `Breaking` fragment** (the `query`/`unlock` removal). No fragment is added. The capability's no-user-visible-change verdicts for Phases 4 and 5 stand.
- **D-07:** **These findings are fixed inside #88 before it merges, as four `/gsd-quick` tasks:** (1) WR-01: in the zero-fragment path, `release-pr` closes any open `changie/release` PR, through a Taskfile target so `TestWorkflowRunBodiesInvokeTask` holds; (2) Runbook WR-02/03/04 in `docs/RELEASE-PROCEDURES.md` (§7/§9 drop the "disposable live proof" framing; §12 replace the tap-failure remedy that re-runs the full `goreleaser release`; §4 step 5 correct the trigger claim: the milestone merge also changes `CHANGELOG.md` and fires `release-tag.yml`, which skips with success); (3) Hygiene WR-02 + WR-03: `require-issue-link.yml` and `pr_template_policy.is_housekeeping` list files through the paginated REST call, cross-check the count against `changed_files`, fail closed on mismatch, and escape the exemption reason; (4) IN-05: the tag job checks out with `persist-credentials: false`, and `release:tag` gets the App token only at push time, through `env:`; this intentionally changes `validateTagJobCredential` and the `release-tag.yml` comment; re-point them RED-first (rule `84d1gfpywd`, Go RED evidence per `x1cjy9vyhq`). Each fix is recorded as `fixed` with its commit in `05-REVIEW-DISPOSITION.md`, including the seven 2026-10-14 findings the ID reuse dropped (`jf9tgc0bb5`), restored through the tool's disposition step, never by hand.
- **D-08:** **Open one milestone issue and put `Closes #N` in #88's body,** following the #76 precedent. Creating the issue is outward-facing, so the maintainer authorizes it at that step. #88's body becomes the squash commit message on `main` (`squash_merge_commit_message: PR_BODY`), so it must not contain `[skip ci]` (rule `f18zrdsgx5`).
- **D-09:** **v0.15.0 is a real, stable release from the moment the tag is pushed. Recovery is patch-forward only.** Any defect ships as the next patch through a `Fixes` fragment (if shipped code changes) and the chain. Nothing is torn down, deleted or re-tagged. The runbook's §7 teardown applies to no published `vX.Y.Z`. Reversibility: one-way.
- **D-10:** **Re-runs: infrastructure failures once; everything else is fixed.** A failed chain job may be re-run once if its log shows an infrastructure cause, with the cause recorded (runner loss, network, a GitHub 5xx, an Apple notary timeout). A failure that would fail the same way again is never re-run: config, code, a guard, a wrong key. It is fixed forward through a PR.
- **D-11:** **SHIP-02's body check is content-equal, not byte-for-byte.** Normalize both sides the same way: CRLF->LF, strip trailing whitespace on each line, and strip leading and trailing blank lines. After that the Release body (`gh api repos/seanb4t/codegraph-go/releases/tags/vX.Y.Z --jq .body`) must equal `.changes/vX.Y.Z.md`. Any other difference is a defect, handled under D-09. The planner amends SHIP-02 in `REQUIREMENTS.md` and ROADMAP Phase 6 success criterion 2 to this wording, in their existing shapes.
- **D-12:** **The agent verifies; the maintainer signs off once.** The maintainer reviews the rendered notes and version and merges the release PR (SC1). The agent gathers and checks everything else, then presents one evidence summary for a single blocking sign-off (SC2 and SC3's human checkpoint). App authorship is shown by who pushed the tag, because `release:tag` makes a lightweight tag with no tagger: the `release-tag` run that logged `pushed vX.Y.Z at <sha>`, plus the `release.yml` tag-push run whose `triggering_actor` is the App's `[bot]` login, not `seanb4t` and not `github-actions[bot]`. The other checks are `changie latest` on `main`, a green `post-release-verify`, and the upgrade runs (D-13).
- **D-13:** **The upgrade starts from v0.14.0, as SHIP-03 says.** Both runs are on this Mac, use the published v0.14.0 tarball, and never touch the Homebrew install. `upgrade.Run` refuses a Homebrew-managed target before any network call (`internal/upgrade/brew.go`). darwin/arm64: extract into a scratch dir, run `./codegraph upgrade`, then confirm `./codegraph --version` reports the new release. linux/amd64: run the same steps in `docker run --platform linux/amd64` (OrbStack, Rosetta). If emulation gets in the way, fall back to a Namespace `devbox`. Do not build anything new for it. Verification runs inside the binary (`sigstore-go` against the unchanged cosign identity) and needs only network access.
- **D-14:** Annotate SEED-005 with the verified trigger (this release's version and Release URL). Fill values only in shapes the seed file already has, using the tool's verb if one exists (generated-files rule). Record the capability tag this release ran with. `CONTRIBUTING.md` names `gsd-capability-changie.git#v0.1.2`, and the executor confirms the installed one rather than assuming.

### Claude's Discretion

- The follow-up branch name, and whether it is cut right after the #88 merge or after the release-PR merge. Either way it is before the first executor commit.
- Grouping of the four quick tasks, and the order of quick tasks vs. the pre-merge audit, within D-03's sequence.
- The milestone issue's title and body.
- The evidence summary's format, and where the transcripts live (a phase artifact such as `06-VERIFICATION.md`).

### Deferred Ideas (OUT OF SCOPE)

- Upgrading through Homebrew (`brew upgrade` after the tap update) is not required by SHIP-03 and is not done in this phase.
- Reviewed todos not folded: "Pin the Go toolchain so Go 1.27 can build codegraph-go"; "tools/bench/runner pinnedAt() validates a checkout by git rev-parse HEAD alone"; "Reply on GH #85 / #82 / #80 with the reshaped contract".
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SHIP-01 | Release PR opened by the workflow from this milestone's fragments, reviewed, merged; `changie latest` on `main` then answers `v0.15.0` | Derived version confirmed v0.15.0 by dry-run; trigger, title, branch, and required-check behavior on an App-opened PR traced (sections 1, 5) |
| SHIP-02 | Tag authored by the App, `release.yml` ran on it, Release body equals `.changes/v0.15.0.md` (content-equal per D-11) | App identity, `gh` evidence commands, and the normalized comparison command (sections 3, 4) |
| SHIP-03 | `post-release-verify` green; `codegraph upgrade` from v0.14.0 verifies against the unchanged cosign identity on darwin/arm64 and linux/amd64 | Upgrade code is byte-identical v0.14.0..HEAD; darwin path rehearsed live; linux emulation is broken on this Mac (sections 2, 6, 7) |
| SHIP-04 | SEED-005 annotated with the verified trigger; capability tag recorded | Seed shape and the capability-version record located (section 8) |
</phase_requirements>

## Summary

The chain is fully built and, apart from the live GitHub-side behavior, statically checks out. With the one `Breaking` fragment and the `.changie.yaml` kind config, `changie batch auto` derives **v0.15.0** (dry-run run against a scratch copy; `changie latest` answers `v0.14.1` today). The upgrade path is unchanged between the published v0.14.0 and `HEAD`: `internal/upgrade/{upgrade,release,verify,brew,swap}.go` and `internal/cli/upgrade.go` have an empty `git diff --stat v0.14.0 HEAD`, the asset names and `.sigstore.json` sidecar naming are unchanged in `.goreleaser.yaml`, and the cosign identity pins the workflow file and tag ref (`release.yml@refs/tags/v...`), not the pusher. I ran the v0.14.0 darwin/arm64 binary against the live latest release (v0.14.1) in a scratch dir and it verified and swapped successfully, so the mechanics work today.

Four things can bite the live run and are worth planning around, none of which needs new infrastructure: (1) **`codegraph upgrade` rewrites the maintainer's real Claude skills** after a successful swap unless `HOME` is redirected (their `~/.claude/skills/codegraph/.codegraph-manifest.json` exists); (2) **OrbStack's amd64 emulation is broken right now** (`exec format error` on every `--platform linux/amd64` image), so D-13's linux leg needs the `devbox` fallback; (3) **`triggering_actor` on a workflow run flips to whoever re-runs it** (v0.14.1's `release` run shows `actor=fzy-release-please[bot]`, `triggering_actor=seanb4t`, `run_attempt=3`), so D-12's evidence must read `actor.login` and `run_attempt == 1`; (4) **the milestone merge does not change `CHANGELOG.md`** (`git diff origin/main..HEAD -- CHANGELOG.md` is empty), so `release-tag.yml` will not fire at all on the #88 merge, contradicting the premise of D-07 item 2.

**Primary recommendation:** Run the chain exactly as built; observe it with the read-only `gh` commands in Code Examples; run both upgrade proofs with `HOME` pointed at the scratch dir and only after `release.yml` has completed; do the linux leg on a `devbox`.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Version derivation, release notes rendering | changie (pinned v1.26.0 via `task changie`) | `.changie.yaml` kinds | changie owns `batch auto`, `merge`, `latest`; no repo code derives a version |
| Release PR open/update | `release-pr.yml` + App token + `peter-evans/create-pull-request` | `task release:batch` | App token is what makes required checks fire |
| Tag push | `release-tag.yml` -> `task release:tag` (App token) | `pretag-gate` | Only tag-creating command in the repo |
| Build, sign, notarize, publish, Release body | `release.yml` (locked) -> `task release:goreleaser` | GoReleaser `--release-notes` | Cosign SAN is pinned to this workflow file |
| Post-publish verification | `post-release-verify.yml` (`workflow_run`) | `task verify:*` | Re-proves supply chain against the published assets |
| Upgrade proof from v0.14.0 | The published v0.14.0 binary (`internal/upgrade`) | `gh`/`curl`/`devbox` for fetching | Verification runs in-process in the binary |

## Standard Stack

No packages are installed in this phase.

| Tool | Version | Used for |
|------|---------|----------|
| `gh` | local | read-only run/PR/release queries, `gh release download` |
| `task changie -- ...` | changie v1.26.0 (pinned `go.tool-changie.mod`; the `mise` shim is the same v1.26.0) | `latest`, `next auto`, `batch auto --dry-run` [VERIFIED: ran `GOWORK=off go list -m -modfile=go.tool-changie.mod github.com/miniscruff/changie` -> `v1.26.0`] |
| `curl`, `cmp`/`diff`, `file`, `jq` | system | asset fetch, body comparison |
| `docker` (OrbStack 2.2.3) | local | only if amd64 emulation is repaired |
| `devbox` (Namespace) | local CLI | linux/amd64 native fallback (D-13) |

### Package Legitimacy Audit

No external packages are installed or recommended in this phase. Not applicable.

## Findings by Unknown

### 1. Derived version: v0.15.0 (confirmed)

- `.changie.yaml:45-46`: `- label: Breaking` / `auto: minor        # pre-1.0: breaking bumps minor. Flip to major at 1.0.` [VERIFIED: .changie.yaml:45-46]
- Latest version file: `.changes/v0.14.1.md` (also `v0.14.0.md` ... `v0.2.0.md`); `.changes/unreleased/` holds exactly `Breaking-20260926-165849.599611000.yaml` (`kind: Breaking`, `custom: PR: "88"`).
- Today: `task changie -- latest` -> `v0.14.1`; `task changie -- next auto` -> `v0.15.0`. [VERIFIED: ran both this session]
- Dry-run in a scratch copy of `.changes/`, `.changie.yaml`, `CHANGELOG.md` (`changie batch auto --dry-run`, exit 0, fragment still present afterward, repo `git status` clean): rendered `## [v0.15.0](https://github.com/seanb4t/codegraph-go/releases/tag/v0.15.0) — 2026-10-17`, `### Breaking`, the `query`/`unlock` bullet ending `([#88](https://github.com/seanb4t/codegraph-go/pull/88))`. [VERIFIED: ran this session]
- The date in the version heading is the date `release:batch` runs (release-PR creation or latest update), not the tag date. Not a defect; do not compare it to the tag date.
- Not v1.0.0: Breaking is `minor` here. If the maintainer adds any other kind before merge the derived version can only stay `v0.15.0` (Breaking/Features -> minor, the max of the set).

### 2. v0.14.0 -> new release upgrade compatibility: no mismatch found

| Item | v0.14.0 binary expects | Current chain publishes | Status |
|------|-----------------------|-------------------------|--------|
| Target selection | `GET https://github.com/seanb4t/codegraph-go/releases/latest`, read the `Location` redirect; fall back to unauthenticated `api.github.com/.../releases/latest`; positional `[version]` pins a tag [VERIFIED: `git show v0.14.0:internal/upgrade/release.go` lines 61, 86] | n/a | OK. No token env var is read anywhere in `internal/upgrade` (rg for `GITHUB_TOKEN\|GH_TOKEN\|Authorization` found nothing), so the unauthenticated API fallback (60/hr) is the only rate-limit exposure and the redirect path avoids it |
| Asset name | `codegraph_%s_%s_%s` (tag, GOOS, GOARCH): the **raw binary**, not an archive [VERIFIED: `git show v0.14.0:internal/upgrade/upgrade.go` line 229] | `.goreleaser.yaml` archive `raw` with `formats: [binary]` and name `{{ .ProjectName }}_{{ .Tag }}_{{ .Os }}_{{ .Arch }}` (lines 140-152) | OK. v0.14.0 published exactly this, e.g. `codegraph_v0.14.0_darwin_arm64`; there is **no tarball**, there is a raw binary plus a `.zip` |
| Signature sidecar | `<asset>.sigstore.json` [VERIFIED: upgrade.go line 214 `assetName+".sigstore.json"`] | `signature: "{{ .ProjectName }}_{{ .Tag }}_{{ .Os }}_{{ .Arch }}.sigstore.json"` (`.goreleaser.yaml:328`) | OK |
| Checksums file | not used by `upgrade` | `codegraph_{{ .Tag }}_checksums.txt` | n/a |
| Cosign identity | issuer `https://token.actions.githubusercontent.com`; SAN regex `^https://github\.com/seanb4t/codegraph-go/\.github/workflows/release\.ya?ml@refs/tags/v[0-9][^\s]*$` [VERIFIED: `git show v0.14.0:internal/upgrade/verify.go:42-44`] | `release.yml` unchanged in name/trigger (`on: push: tags: ["v[0-9]*"]`) | OK |

- **The identity does not depend on who pushed the tag.** The SAN comes from the job's workflow ref, not the actor. Direct evidence: v0.14.1's `release` run was started by the App (`actor=fzy-release-please[bot]`) and re-run by `seanb4t` (attempt 3), and the v0.14.0 binary verified its artifact today (rehearsal below).
- `git diff --stat v0.14.0 HEAD` is empty for `internal/upgrade/{upgrade,release,verify,brew,swap}.go` and `internal/cli/upgrade.go`. [VERIFIED: ran this session]
- Diff of `release.yml` v0.14.0..HEAD is two comment-and-`use-cache: false` hunks only; `.goreleaser.yaml` differs only in the `release:` block comment. [VERIFIED: `git diff v0.14.0 HEAD`]
- **Other refusals besides Homebrew:** (a) `checkWritable`: target dir must allow `os.CreateTemp` or it refuses before download (`swap.go:18-28`); (b) same version without `--force` is a no-op ("already on"); (c) verification failure is fatal and never swaps. `--check` never mutates. There is no dev-build refusal.
- **Target is whatever GitHub calls "latest".** GoReleaser creates the Release first and uploads assets after, so there is a window where `releases/latest` points at v0.15.0 with partial assets. Run the upgrade proofs only after `release.yml` has finished (post-release-verify only starts then).

### 3. The release body

- `release:goreleaser` runs `"${GORELEASER_BIN}" release --clean --release-notes "${NOTES}"` with `NOTES=".changes/${TAG}.md"` (`Taskfile.yml:3078-3083`) and refuses a missing or empty file (`[ ! -s "${NOTES}" ]`). [VERIFIED: Taskfile.yml:3078-3083]
- `.goreleaser.yaml` `release:` has only `replace_existing_artifacts: true` (line 399) and `prerelease: auto` (line 403); no `header`, `footer`, `name_template`, `draft` or `disable`, and no `changelog:` key (`rg` for `changelog` finds only a comment). So nothing adds text to the notes. [VERIFIED: .goreleaser.yaml:398-403, rg]
- This is the first time GoReleaser *creates* the Release in this repo (release-please created it before). Behavior with `--release-notes` is third-party behavior and is not to be unit-tested (rule `eyva2svrcz`); the live run is its first observation.
- Positive control for the comparison command below: applied to the already-published v0.14.1, body vs `.changes/v0.14.1.md` normalizes to equal (6 lines each, `diff` empty). [VERIFIED: ran this session]

### 4. App authorship evidence

- Both `release-pr.yml` and `release-tag.yml` mint the token with `actions/create-github-app-token` from secrets `APP_ID` and `APP_PRIVATE_KEY` (repo secrets exist, created 2026-07-29/30). The App is **`fzy-release-please`** (display name "fzy Release-Please", id 3982691, owner seanb4t). [VERIFIED: `gh api apps/fzy-release-please`; `gh secret list`] That `APP_ID` is that App's id is [ASSUMED] (secret values are unreadable), but it is corroborated: every release-please PR and the v0.14.1 tag run (before the chain existed) were authored by `fzy-release-please[bot]`, and the runbook says the same App is reused.
- Bot login: **`fzy-release-please[bot]`** (user id 291367322, type Bot). In `gh pr list` it prints as `app/fzy-release-please`.
- `actor` and `triggering_actor` both exist on the workflow-run REST object. [VERIFIED: `gh api repos/seanb4t/codegraph-go/actions/runs/38414831563` returned `"actor":"fzy-release-please[bot]"`, `"triggering_actor":"seanb4t"`, `"run_attempt":3`]
- **Trap:** `triggering_actor` is the person who last re-ran the run; `actor` is the original pusher. Read `actor.login` and require `run_attempt == 1`; if D-10 permits a re-run of `release.yml`, `triggering_actor` will show the maintainer and that is expected, not an authorship failure.
- The `pushed` log line comes from `Taskfile.yml:1065`: `echo "release:tag: pushed ${tag} at $(git rev-parse HEAD)"`. [VERIFIED: Taskfile.yml:1065]

### 5. The release PR path (SHIP-01)

- **After #88 squash-merges:** `release-pr.yml` fires on `push: branches: [main]` (`pretag-gate` -> `release-pr` job). `release:batch` batches, merges, and writes `version` to `GITHUB_OUTPUT`; `peter-evans/create-pull-request` opens branch **`changie/release`**, title **`chore(main): release v0.15.0`**, body = `.changes/v0.15.0.md`, `draft: false` (`release-pr.yml:94-97`). The PR contains `.changes/v0.15.0.md`, the deleted fragment, and `CHANGELOG.md`.
- **Required checks on the App-opened PR** (`protect-main` ruleset 20157557: squash only, 0 approvals, `required_review_thread_resolution: true`, 9 contexts: `test`, `actionlint (workflow static analysis)`, `perf regression gate (PERF-02, INDX-06)`, `pr-title`, `reproducibility (double-build hash-diff, DIST-04)`, `govulncheck (DIST-03, blocking)`, `goreleaser check (config validation, DIST-01)`, `tmux e2e (real-pty harness, TTY-01..TTY-07)`, `fragment-required`). [VERIFIED: `gh api repos/.../rulesets/20157557`]
  - `ci.yml` triggers on `pull_request` with no path filter, and none of the 8 ci jobs above has an `if:` (only `transcript-freeze` does, and it is not required), so they all report. `pr-title` regex `^(feat|fix|perf|refactor|docs|chore|ci|test|build|revert)(\([a-z0-9_-]+\))?!?: .+` accepts `chore(main): release v0.15.0`.
  - `fragment-required`: the release PR changes only `.changes/**` and `CHANGELOG.md`; `fragment_policy.is_shipped_code` is true only under `cmd/` or `internal/`, so the verdict is `NO_SHIPPED_CODE` (pass). `Issue link required` is not a required context and path-exempts `.changes/*` and `CHANGELOG.md` anyway. `auto-close-unsolicited-prs` only acts on `feat`/`perf` titles. `close-draft-prs` only acts on drafts.
  - The `test` job runs `task check:changie` (scratch copy; `.changes/v0.15.0.md` joining the `v*.md` set does not trip its `floor 14` count). Phase 5 rehearsed this with its 14 legs; the live run is the first real PR.
- **What `release-tag.yml` does when #88 merges:** it triggers only `paths: [CHANGELOG.md]` (`release-tag.yml:31`). **`git diff origin/main..HEAD -- CHANGELOG.md` is empty now** (main already carries the v0.14.1 section from release-please #98), so the #88 merge push does not touch `CHANGELOG.md` and `release-tag.yml` will not run at all. If it ever did, `release:tag` skips with success because `v0.14.1` exists on origin. D-07 item 2's "the milestone merge also changes `CHANGELOG.md` and fires `release-tag.yml`" is stale (it was measured against a stale local `main`, `d8c6ec38`, vs `origin/main` `7b12cb92`). The runbook edit should say "if a merge to `main` changes `CHANGELOG.md`, `release-tag` runs and skips with success when the tag exists", and the executor should recheck the diff right before the merge. Expected observation after the #88 merge: `release-tag.yml` has **no** run for that SHA (or a skipped-with-success one).
- **After the release PR is squash-merged** (squash title `PR_TITLE`, message `PR_BODY`; branch auto-deleted, `delete_branch_on_merge: true`): the push changes `CHANGELOG.md`, so `release-tag.yml` runs `pretag-gate` then `tag`: `changie latest` -> `v0.15.0`, `git ls-remote --exit-code` returns 2 (absent), `git tag v0.15.0` at the squash commit, `git push origin refs/tags/v0.15.0` with the App token. That tag push fires `release.yml` (tag filter `v[0-9]*`). When `release.yml` completes, `workflow_run` fires `post-release-verify.yml`. `release-pr.yml` also runs on the same push and no-ops (no fragments; WR-01's fix closes any open `changie/release` PR in that path).
- **IN-05 live-risk (D-07 item 4):** after the fix, the tag job's push auth moves to `env:` at push time. That is the one piece of the chain no rehearsal can fully prove, and a failure there happens after the release-PR merge. A re-run of the failed `release-tag` run executes the workflow file **at the original SHA**, so a fix merged afterwards is not picked up, and no other trigger re-fires the job (no `workflow_dispatch`; touching `CHANGELOG.md` by hand is forbidden by `.changie.yaml`). Plan accordingly: keep the IN-05 change minimal and rehearse it before merging #88 with the existing scratch-bare-repo rehearsal named in `release:tag`'s precondition message (`GITHUB_ACTIONS=true` against a scratch clone). Raise any failure to the maintainer rather than improvising.

### 6. post-release-verify (SHIP-03)

- Trigger: `workflow_run` on workflow named `release`, `types: [completed]`, plus `workflow_dispatch` with a `tag` input. Jobs (7 in a normal run): `resolve-tag`, `verify-supply-chain`, `self-upgrade` x2 (darwin/arm64, linux/amd64), `gatekeeper` x2 (arm64, amd64), `notarized-suite`. It verifies asset set, cosign signature, attestation, Gatekeeper verdict, notarized suite, and a self-upgrade from the **prior stable release (v0.14.1)** to the new tag, in CI.
- **Skipped is not green.** A `workflow_run` of a failed `release` attempt produces a run with conclusion `skipped` (v0.14.1 has two skipped runs before the successful one). The workflow run lists `headBranch: main`, not the tag. Select the run by `headSha == <tag commit>` and require every job `success`.
- Note this is CI's self-upgrade from v0.14.1; the SHIP-03 proof from v0.14.0 is the local D-13 run. Same code path (see 2).

### 7. Upgrade proof mechanics (D-13)

**Rehearsed today** (v0.14.0 -> v0.14.1, scratch only): `gh release download v0.14.0 -p codegraph_v0.14.0_darwin_arm64`, `chmod +x`, `HOME=<scratch>`; `--version` printed v0.14.0, `upgrade --check` printed `codegraph v0.14.1 is available (current: v0.14.0)`, `upgrade` printed `upgraded to v0.14.1`, `--version` then printed v0.14.1 with the v0.14.1 commit. [VERIFIED: ran this session]

- **Must set `HOME` to the scratch dir (and run from it).** After a successful swap, `cli/upgrade.go` calls `refreshInstalledSkills(target, ...)`, which reads `~/.claude/skills/codegraph/.codegraph-manifest.json` and re-installs the Claude skill package with `ExecPath` = the binary just upgraded. The maintainer's real manifest exists (`/Users/sean/.claude/skills/codegraph/.codegraph-manifest.json`, 529 bytes), so an un-isolated run would rewrite their real skills to point at a scratch binary. `os.UserHomeDir()` honors `$HOME`; the local-scope path is cwd-relative, so run with cwd = the scratch dir. [VERIFIED: internal/agents/claude.go `claudeSkillDirPath`; internal/cli/upgrade.go refresh call]. The TUF trust-root cache is written under `$HOME/.sigstore`, so a writable `HOME` is required in the container too.
- The raw darwin asset is a plain download (no quarantine attribute), so no Gatekeeper prompt; no extraction step exists because the asset is the binary. The `.zip` is not used.
- **linux/amd64: the v0.14.0 asset is a dynamically linked glibc binary**: `ELF 64-bit LSB executable, x86-64, version 1 (SYSV), dynamically linked, interpreter /lib64/ld-linux-x86-64.so.2` (zig `-target x86_64-linux-gnu`). [VERIFIED: `file` on the downloaded asset] Alpine/musl will not run it. It needs a glibc image plus `ca-certificates` (`debian:*-slim` ships without it).
- **OrbStack amd64 emulation is not working on this Mac right now.** `orbctl config show` reports `rosetta: true`, but `docker run --rm --platform linux/amd64 hello-world` and `alpine uname -m` both fail with `exec format error`. [VERIFIED: ran this session] Fixing it means changing OrbStack's VM (a restart, or `tonistiigi/binfmt`), which the maintainer should authorize. D-13's stated fallback is the Namespace `devbox` (native linux/amd64, glibc), so plan the linux leg on a `devbox` and treat the docker form below as the alternative if emulation is repaired.
- Rate limits: none for the normal path (redirect + `github.com/.../releases/download`); no token is honored or needed.

### 8. SHIP-04

- `.planning/seeds/SEED-005-changie-template-for-other-repos.md` frontmatter keys: `id`, `status: dormant`, `planted`, `planted_during`, `trigger_when`, `scope`; body sections `# SEED-005: ...`, `## Why This Matters`, `## When to Surface`. [VERIFIED: read in full] It has no `## Notes`, `## Breadcrumbs`, `consumed_by` or `consumed_on`. SEED-001/002 use `status: implemented`, `consumed_by`, `consumed_on` for a *consumed* seed; SEED-005 is not consumed (PORT-01, the port, is v2).
- There is **no gsd-tools verb that writes a post-plant annotation**: `list-seeds` is read-only (it surfaces only frontmatter fields and the title), and `plant-seed --enrich` rewrites only trigger/scope/why. [VERIFIED: `gsd-tools list-seeds` output; `.claude/gsd-core/workflows/plant-seed.md` enrich step] Recommended minimal edit: add a short prose paragraph inside the existing `## When to Surface` section stating the trigger fired, with the version, the Release URL (`https://github.com/seanb4t/codegraph-go/releases/tag/v0.15.0` if that is the derived version), the date, and the capability tag. Leave `status: dormant` and `trigger_when` untouched (the criterion text should stay readable). See Open Question 1 for the alternative.
- Installed capability, confirmable from the repo (the file is gitignored but local): `jq '.entries.changie|{version,source}' .gsd-capabilities.json` -> `{"version":"0.1.2","source":"https://github.com/seanb4t/gsd-capability-changie.git#v0.1.2"}`; `.gsd/capabilities/changie/capability.json` -> `version: 0.1.2`; the capability checkout's own `git rev-parse HEAD` is `2631063f05fffc7152d70a9c98eb0b7479a0029a`, equal to `gh api repos/seanb4t/gsd-capability-changie/commits/v0.1.2 --jq .sha`. [VERIFIED: ran this session] The ledger's `integrity` field is empty, so the version string plus the commit is the record. CONTRIBUTING.md names the same tag (`#v0.1.2`, lines 233/239).

### 9. The follow-up branch (D-03/D-04)

- `fragment-required` passes a `.planning/`-only diff with no marker: `is_shipped_code` ignores anything outside `cmd/` and `internal/`, giving verdict `NO_SHIPPED_CODE`. `Issue link required` and `Pull request template format` path-exempt `.planning/` (`EXEMPT_PREFIXES` includes `.planning/`, `.changes/`). [VERIFIED: scripts/fragment_policy.py, scripts/pr_template_policy.py:38-43, require-issue-link.yml]
- `release-pr.yml` no-ops with zero fragments: `release:batch` counts `*.yaml` under `.changes/unreleased/`, prints "nothing to release", exits 0 without writing `version`, so `steps.batch.outputs.version != ''` is false and `create-pull-request` is skipped (`Taskfile.yml:1019-1022`).
- **Caveat:** that holds only if no fragment is added on the way. If Phase 6's own close (the capability's `verify:post` step) writes a fragment into `.changes/unreleased/`, the follow-up PR's merge would open a *new* release PR. D-06 holds that Phase 6 ships no user-visible change; confirm `ls .changes/unreleased/` shows only `.gitkeep` on the follow-up branch before it merges.
- Cut the follow-up branch from `origin/main` after `git fetch`; local `main` is stale (`d8c6ec38` vs `origin/main` `7b12cb92`).

## Architecture Patterns

### Chain data flow

```
#88 squash-merge (main push, PR_BODY as message)
   |
   +--> release-pr.yml: pretag-gate -> release:batch (changie batch auto + merge)
   |        -> create-pull-request [App token] -> PR "chore(main): release v0.15.0" (branch changie/release)
   |              -> pull_request checks (9 required) run because token is the App's
   |
   +--> release-tag.yml: NOT triggered (CHANGELOG.md unchanged by #88)

maintainer reviews notes+version, squash-merges release PR
   |
   +--> release-pr.yml (no fragments -> no-op)
   +--> release-tag.yml: pretag-gate -> tag job: changie latest = v0.15.0
            -> ls-remote absent -> git tag -> git push [App token]
                  |
                  +--> release.yml (tag push; actor = fzy-release-please[bot])
                           task release:goreleaser: build x4, notarize, cosign sign-blob,
                           syft SBOM, Release created with --release-notes .changes/v0.15.0.md,
                           attest provenance
                              |
                              +--> post-release-verify.yml (workflow_run completed)
                                       7 jobs against the published assets

agent: read-only gh checks + local upgrade runs (HOME isolated) -> one evidence summary -> one maintainer sign-off
follow-up planning-only PR from origin/main carries SUMMARY/VERIFICATION/UAT/audit/milestone close
```

### Anti-Patterns to Avoid

- **Running `./codegraph upgrade` with the real `HOME`:** rewrites the maintainer's installed Claude skills (see 7).
- **Reading `triggering_actor` after a re-run:** it becomes the re-runner. Use `actor` and `run_attempt`.
- **Treating a `skipped` post-release-verify run as green.**
- **Running the upgrade proofs before `release.yml` finishes:** `releases/latest` can point at a Release with partial assets.
- **Testing or guarding GitHub/GoReleaser/changie behavior** (rule `eyva2svrcz`): observe, do not assert in code.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Checking authorship | a new guard or workflow | `gh api .../actions/runs/<id>` fields `actor`, `triggering_actor`, `run_attempt` plus the `release-tag` log line | D-01; the fields exist today |
| Body equality | a test or script file | the `norm` shell function + `cmp` below | D-11 normalization is three transforms |
| Upgrade proof | a harness | the published v0.14.0 asset + `./codegraph upgrade` | the binary verifies itself |
| Recovery from a bad publish | tag deletion, re-tag, hand-pushed tag | next patch through a `Fixes` fragment (D-09) | one-way |

## Common Pitfalls

### Pitfall 1: Un-isolated `HOME` during the upgrade proof
**What goes wrong:** the post-swap refresh rewrites `~/.claude/skills/codegraph` and its hooks for the maintainer, pointing at a scratch binary path. **How to avoid:** `HOME=$SCRATCH` and cwd = `$SCRATCH` on every `./codegraph upgrade`. **Warning sign:** the output lists `updated:`/`created:` paths under `/Users/sean/.claude`.

### Pitfall 2: `triggering_actor` after a re-run
See 4. v0.14.1 is the live example (attempt 3).

### Pitfall 3: amd64 containers fail with `exec format error`
See 7. Do not interpret it as a codegraph defect; go to `devbox`.

### Pitfall 4: A failed `release-tag` run cannot be repaired by fixing forward
See 5. Rehearse IN-05 before the #88 merge; do not re-run a deterministic failure (D-10).

### Pitfall 5: Stale local `main`
The local `main` (`d8c6ec38`) is behind `origin/main` (`7b12cb92`); every "does X change on merge" check must use `origin/main` after `git fetch`.

### Pitfall 6: Infra-flaky release job
v0.14.1's `release` run needed attempt 3 (a restored `.zig-cache` dirtied the workspace; fixed by `use-cache: false`, already in `release.yml`). Apple notarization (`timeout: 20m`) and Namespace runner loss are the plausible remaining infra causes; D-10 allows one re-run for those, with the cause recorded.

## Code Examples

All read-only except the local upgrade runs, which write only into `$SCRATCH`.

```bash
REPO=seanb4t/codegraph-go; TAG=v0.15.0   # replace TAG with `task changie -- latest` on the follow-up branch
SHA=$(gh api repos/$REPO/commits/$TAG --jq .sha)

# SC1: the release PR and the derived version
gh pr list --repo $REPO --head changie/release --state all --json number,title,author,state,url
gh pr view <n> --repo $REPO --json title,body,author,statusCheckRollup
task changie -- latest            # on a checkout of origin/main (or the follow-up branch): expect v0.15.0

# SC2(a): the tag job and its pushed line
gh run list --repo $REPO --workflow release-tag.yml --limit 5 --json databaseId,conclusion,headSha,event
gh run view <release-tag-run-id> --repo $REPO --log | rg "release:tag: pushed $TAG at"

# SC2(b): the release.yml tag-push run
gh run list --repo $REPO --workflow release.yml --branch $TAG --json databaseId,attempt,conclusion,headSha
gh api repos/$REPO/actions/runs/<release-run-id> \
  --jq '{event,head_branch,head_sha,run_attempt,actor:.actor.login,triggering_actor:.triggering_actor.login}'
# expect event=push, head_branch=$TAG, head_sha=$SHA, run_attempt=1,
# actor = triggering_actor = fzy-release-please[bot]

# SC2(c): body content-equal (D-11), with a non-emptiness assertion
norm() { tr -d '\r' | sed -E 's/[[:space:]]+$//' | awk '{l[NR]=$0} END{s=1; while(s<=NR&&l[s]=="")s++; e=NR; while(e>=s&&l[e]=="")e--; for(i=s;i<=e;i++)print l[i]}'; }
gh api repos/$REPO/releases/tags/$TAG --jq .body | norm > "$SCRATCH/body.txt"
git show "$TAG:.changes/$TAG.md" | norm > "$SCRATCH/file.txt"
[ -s "$SCRATCH/body.txt" ] && [ -s "$SCRATCH/file.txt" ] && cmp "$SCRATCH/body.txt" "$SCRATCH/file.txt" \
  && echo "CONTENT-EQUAL lines=$(wc -l < "$SCRATCH/file.txt")"

# Asset set unchanged vs v0.14.0 (expect 17 names, identical after the version is masked)
diff <(gh release view v0.14.0 --repo $REPO --json assets --jq '.assets[].name' | sed 's/v0.14.0/vX/' | sort) \
     <(gh release view $TAG    --repo $REPO --json assets --jq '.assets[].name' | sed "s/$TAG/vX/"  | sort)

# SC3: post-release-verify for THIS tag (skipped runs belong to failed release attempts)
gh run list --repo $REPO --workflow post-release-verify.yml --limit 10 --json databaseId,conclusion,headSha,createdAt
gh run view <id> --repo $REPO --json jobs --jq '.jobs[]|"\(.name) => \(.conclusion)"'   # 7 jobs, all success
```

Upgrade proofs (after `release.yml` has completed):

```bash
# darwin/arm64 (this Mac). Each proof gets its own empty dir; HOME is the same dir.
SCRATCH=$(mktemp -d); cd "$SCRATCH"
gh release download v0.14.0 --repo $REPO -p codegraph_v0.14.0_darwin_arm64 --dir "$SCRATCH"
cp codegraph_v0.14.0_darwin_arm64 codegraph && chmod +x codegraph
HOME="$SCRATCH" ./codegraph --version            # v0.14.0
HOME="$SCRATCH" ./codegraph upgrade --check      # "... v0.15.0 is available (current: v0.14.0)"
HOME="$SCRATCH" ./codegraph upgrade              # "upgraded to v0.15.0"
HOME="$SCRATCH" ./codegraph --version            # v0.15.0 and the tag's commit

# linux/amd64, native on a Namespace devbox (glibc, CA bundle assumed present):
#   curl -fsSLo codegraph https://github.com/seanb4t/codegraph-go/releases/download/v0.14.0/codegraph_v0.14.0_linux_amd64
#   chmod +x codegraph; export HOME=$(mktemp -d); ./codegraph upgrade; ./codegraph --version
# linux/amd64 under docker, only if emulation is repaired (glibc image + CA certs, writable HOME):
#   docker run --rm --platform linux/amd64 -v "$SCRATCH:/w" -w /w -e HOME=/tmp/h debian:trixie-slim \
#     sh -c 'apt-get update -qq && apt-get install -y -qq ca-certificates && ./codegraph upgrade && ./codegraph --version'

# SHIP-04 capability record
jq '.entries.changie|{version,source}' .gsd-capabilities.json
```

## State of the Art

| Old | Current | Impact |
|-----|---------|--------|
| release-please created the tag and the Release body | changie fragments -> release PR -> App-pushed tag -> GoReleaser creates the Release with `--release-notes` | The "GoReleaser creates the Release" step has never run live in this repo |
| `post-release-verify` started on `release: published` | `workflow_run` on `release` completing | Fires after all assets exist |

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `APP_ID` is the id of `fzy-release-please` (the App that authored every release-please PR) | 4 | The tag would show a different `[bot]` login; the evidence still names whichever App it is, but the expected string in the summary would need changing |
| A2 | A Namespace `devbox` image has curl, glibc and CA certificates and outbound HTTPS | 7 | The linux leg needs an `apt-get install ca-certificates` step or a different image |
| A3 | GoReleaser v2.17.1 puts `--release-notes` content in the Release body verbatim (no header/footer/changelog text added, since none is configured) | 3 | Body differs from the file; handled under D-09 as a defect found by the live run |
| A4 | GoReleaser marks the new Release "latest" (its default), so unpinned `upgrade` resolves v0.15.0 | 2, 7 | `upgrade` would stay on v0.14.1; the proof fails visibly and `upgrade v0.15.0` pins it |

## Open Questions

1. **Where to put the SEED-005 annotation.**
   - Known: no tool verb exists; SEED-005 has no `## Notes`; SEED-001..003 do (and the plant-seed template emits `## Notes`).
   - Recommendation: prose inside the existing `## When to Surface`. Adding `## Notes` is template-sanctioned but is a heading this file lacks; the maintainer decides if they prefer it.
2. **D-07 item 2's premise is stale** (see 5): the `#88` merge does not change `CHANGELOG.md`. The runbook wording should be conditional. The planner should tell the quick-task author, not rewrite the decision.
3. **linux/amd64 leg:** devbox (no repair needed) vs the maintainer authorizing an OrbStack restart. Recommend devbox per D-13's own fallback.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `gh` (authenticated) | all evidence commands | yes | local | none needed |
| `task`, `changie` (via `go tool`) | `changie latest`, `next auto` | yes | changie v1.26.0 | none needed |
| `jq`, `file`, `curl`, `cmp`, `awk` | comparison and checks | yes | system | none needed |
| Docker linux/amd64 containers | linux upgrade proof via docker | **no (`exec format error`)** | OrbStack 2.2.3, `rosetta: true` in config | `devbox` |
| `devbox` CLI | linux upgrade proof | yes (installed) | not probed (not created) | none needed |
| Network to github.com, Sigstore TUF | `upgrade` | yes (rehearsal passed) | n/a | none |

**Missing with fallback:** amd64 emulation under OrbStack -> `devbox` (D-13).

Side effects of this research on the machine: a rehearsal binary and downloads live under the session scratchpad; the docker images `hello-world` and `debian:trixie-slim` I pulled for the emulation probe were removed; `alpine:latest` (amd64 variant) remains pulled.

## Validation Architecture

Verification in this phase is observation of live runs with read-only commands, not new tests (D-01). No test files are added; no Wave 0 gaps.

### Test Framework
| Property | Value |
|----------|-------|
| Framework | none new; existing Go tests, Python `scripts/test_*.py`, `task check:*` |
| Config file | `Taskfile.yml` |
| Quick run command | `task check:changie` (14 legs) / `task check:fragment-policy` where a pre-merge quick task touches those files |
| Full suite command | `task test` (also `task lint:actions`, `go test ./internal/upgrade/...`) |

These still apply only to the pre-merge quick tasks (D-07), which run outside `/gsd-execute-phase 6`.

### Phase Requirements -> Observation Map
| Req ID | Behavior | Type | Observation command | Positive assertion |
|--------|----------|------|---------------------|--------------------|
| SHIP-01 | Release PR opened by the workflow, merged; `changie latest` = derived version | live, human checkpoint | `gh pr list --head changie/release --state all ...`; `task changie -- latest` on `origin/main` | PR author is `fzy-release-please[bot]`, title is `chore(main): release <v>`, `latest` equals the PR's version |
| SHIP-02 | App-authored tag; `release.yml` ran; body content-equal | live, human checkpoint | `gh run view <id> --log \| rg "release:tag: pushed"`; `gh api .../actions/runs/<id>` fields; the `norm`+`cmp` command | `pushed <tag> at <sha>` with `<sha>` = tag commit; `actor` = bot, `run_attempt` = 1; non-empty files and `cmp` exit 0 |
| SHIP-03 | `post-release-verify` green; upgrade from v0.14.0 verifies on both platforms | live, human checkpoint | `gh run view <id> --json jobs`; the two upgrade transcripts | 7 jobs `success` on the run whose `headSha` = tag commit; `--version` after upgrade prints the new tag |
| SHIP-04 | Seed annotated; capability tag recorded | file edit | `jq` on `.gsd-capabilities.json`; `git diff` of the seed | the annotation names version, Release URL, `v0.1.2` + commit |

### Sampling Rate
- **Per task:** the observation command for that step; transcript pasted into `06-VERIFICATION.md`.
- **Phase gate:** the single evidence summary signed off once (D-12).

### Wave 0 Gaps
None. Existing infrastructure covers the pre-merge tasks; the live run needs only `gh`/`curl`.

## Security Domain

`security_enforcement` is enabled. This phase adds no code.

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2/V3 Authentication, Session | no | n/a |
| V4 Access Control | yes | `protect-main` ruleset, App with Contents + Pull requests write only (runbook section 9) |
| V5 Input Validation | no new inputs | existing `release:batch` skip-ci guard on rendered notes |
| V6 Cryptography | yes | verification stays in-process (`sigstore-go`), identity pinned to the workflow ref; nothing hand-rolled |

| Threat | STRIDE | Mitigation here |
|--------|--------|-----------------|
| Running a downloaded binary with the real `HOME` | Tampering | `HOME` isolation (Pitfall 1); scratch dirs, one empty dir per download |
| Unverified artifact executed | Spoofing | the v0.14.0 asset comes from the published release; its `.sigstore.json` can be checked with `cosign verify-blob` before running, as `verify:self-upgrade` does |
| Tag by a human or `GITHUB_TOKEN` | Spoofing/Repudiation | `actor` = App `[bot]` check (D-12) |

## Sources

### Primary (HIGH confidence)
- Repository files read this session: `.changie.yaml`, `.goreleaser.yaml`, `Taskfile.yml` (`release:batch`, `release:tag`, `release:goreleaser`, `verify:self-upgrade`), `.github/workflows/{release-pr,release-tag,release,post-release-verify,ci,fragment-required,require-issue-link,pr-title,close-draft-prs,auto-close-unsolicited-prs}.yml`, `scripts/fragment_policy.py`, `scripts/pr_template_policy.py`, `internal/upgrade/*`, `internal/cli/upgrade.go`, `internal/agents/{claude,manifest}.go`, `docs/RELEASE-PROCEDURES.md`, SEED-001..005, `.gsd-capabilities.json`
- `git show v0.14.0:internal/upgrade/{upgrade,release,verify,brew}.go` and `git diff --stat v0.14.0 HEAD` on the same paths
- Read-only GitHub API queries: ruleset 20157557, repo settings, App `fzy-release-please`, run 38414831563, post-release-verify runs, v0.14.0/v0.14.1 releases, PR #88 state and threads, capability tag commit
- Local runs: `changie batch auto --dry-run` (scratch), `task changie -- latest|next auto`, darwin upgrade rehearsal v0.14.0 -> v0.14.1 (scratch, isolated `HOME`), `file` on the linux asset, OrbStack amd64 probe

### Secondary / Tertiary
None used; no web research was needed.

## Metadata

**Confidence breakdown:**
- Derived version, upgrade compatibility, comparison command: HIGH (executed or diffed)
- Release PR / tag / post-release-verify live behavior: MEDIUM (traced statically; first live run)
- linux/amd64 leg: MEDIUM (emulation failure verified; devbox contents assumed)

**Research date:** 2026-10-17
**Valid until:** until #88 merges (the repo state and `origin/main` change the facts in 5 and 9)
