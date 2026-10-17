# Phase 5: Release Chain & release-please Retirement - Research

**Researched:** 2026-10-14
**Domain:** GitHub Actions release automation (changie release-PR pattern + App-token tag push), GoReleaser release-notes wiring, test-guard re-pointing, docs/census retirement of release-please
**Confidence:** HIGH on the documented pattern, action pins and guard inventory (all read or run this session); MEDIUM on two behaviors that only Phase 6's live run can confirm (see Open Questions)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

### Governing principle
- **D-01:** **Use the documented pattern; do not design one.** The maintainer, explicitly: "isn't there an action for this and a general approach? Why are we making a space shuttle?" The chain follows changie's own docs (its `gen-release-pr.yml`, "Release Trigger" and GoReleaser integration pages) and `peter-evans/create-pull-request`'s documented token guidance. Where the docs already answer a question, the planner takes the documented answer; custom logic is added only where a locked constraint of this repo forces it, and each such addition must name the constraint.

### Release PR (REL-10)
- **D-02:** The release-PR job runs `changie batch auto` → `changie merge` → `changie latest` (through the existing `task changie` pin, Phase 1 D-04 — CHG-01's one recorded CI install path; no second install via `changie-action`) and opens or updates **one** PR with **`peter-evans/create-pull-request`** (SHA-pinned, version comment) on a fixed branch, regenerated from `main` on every run, title `chore(main): release vX.Y.Z`. With no `*.yaml` fragments in `.changes/unreleased/` it does nothing (`changie batch auto` fails on an empty set — CHG-03 — so the job checks first and no-ops).
- **D-03:** `create-pull-request` is given the **App installation token** (`actions/create-github-app-token`, existing `APP_ID`/`APP_PRIVATE_KEY`). This is the action's documented answer to the second token hazard: a PR opened or updated with `GITHUB_TOKEN` fires no `pull_request` workflows, so `protect-main`'s 9 required checks (incl. `test`, `tmux e2e`, `reproducibility`, `perf regression gate` — `ci.yml` runs on plain `pull_request`) would never report and the PR could never merge.
- **D-04:** The PR body is the rendered `.changes/vX.Y.Z.md`. It is opened **ready for review, never draft** (the App's `author_association` is not OWNER/MEMBER/COLLABORATOR, so `close-draft-prs.yml` would close a draft).
- **D-05:** The hygiene gates recognize the release PR **by path**: in both `pr_template_policy.py`'s exemption set and `require-issue-link.yml`'s `case` list, the two release-please filenames are replaced by a `.changes/` prefix. A PR touching only `.changes/**` plus `CHANGELOG.md` is housekeeping. This is GRD-15's "gains the release PR's own" read literally. `fragment-required` already passes it (no `cmd/**`/`internal/**`).

### Tag (REL-11)
- **D-06:** Trigger is changie's documented **release trigger: a push to `main` that changes `CHANGELOG.md`** — only a release-PR merge (`changie merge`) changes it. The tag job tags `$(changie latest)` at the pushed commit with the App token, and **skips (success, not failure) when that tag already exists**. That skip is what makes the milestone merge — which also changes `CHANGELOG.md` and brings all 15 seed files (`v0.2.0`…`v0.14.1`, already tagged) — a no-op. A failed tag job is re-run like any other job.
  - Supersedes the two earlier answers in this session ("state-based on every push" and "tag the commit that added the file"): with the path trigger, the pushed commit *is* the release merge, so neither extra mechanism is needed.
  - Do **not** add a "tag must point at the commit that added `.changes/vX.Y.Z.md`" assertion: the historical tags point at release-please commits, not at the milestone merge that adds their seed files.
  - **REL-11's parenthetical** ("detected by a new `.changes/vX.Y.Z.md` on `main`") and ROADMAP Phase 5 success criterion 1's matching clause are **amended** to the documented trigger. The planner updates the values in `REQUIREMENTS.md`/`ROADMAP.md`'s existing shapes (no new structure — the planning-artifacts rule, as Phase 4 D-09 did).
- **D-07:** **`pretag-gate` (`task check:cross`, moved verbatim) gates both jobs** — the release-PR job and the tag job each `needs:` it. Its original purpose is "never cut a tag from a `main` state that fails cross-platform resolution", and the tag path is where that applies.

### GoReleaser and rehearsal (REL-12, REL-13)
- **D-08:** `release.yml` changes by exactly one thing: `goreleaser release` receives `--release-notes .changes/${TAG}.md` (changie's documented GoReleaser integration). `.goreleaser.yaml` never sets `changelog.disable: true`, asserted by the existing shape tests.

### Census and docs (REL-14, DOCS-12…14)
- **D-09:** REL-14 is a plain grep census, positive-controlled (first shown to find a planted reference), with exactly the requirement's exclusions — `.planning/`, `CHANGELOG.md` history, this milestone's own notes — where **`.changes/v*.md` seed files count as `CHANGELOG.md` history** (tool-owned, byte-sliced from it; `.changes/v0.2.0.md` mentions release-please and cannot be edited). Everything else is reworded, including test names (`TestReleasePleaseStaysPreMajor` is renamed when re-pointed at `.changie.yaml`'s `Breaking → minor`), code comments, D-06R prose in `tag_authority_test.go`, and the `.changie.yaml` comment at line 20.
- **D-10:** Each of `docs/RELEASE-PROCEDURES.md`'s 45 release-please references gets a recorded verdict in a phase artifact (not inline in the doc).

### Carried forward (not re-asked)
- `release.yml`'s `on: push: tags: v[0-9]*` trigger and `internal/upgrade/verify.go`'s cosign identity are byte-locked; every action SHA-pinned with a version comment; every CI `run:` body is a `task` invocation (`TestWorkflowRunBodiesInvokeTask`; `uses:`-only jobs go in its exception list, as release-please's job did).
- Tests exercise this repository's code only, never how git, GitHub, Actions, changie or `create-pull-request` behave (Phase 4 D-01, rule `eyva2svrcz`). Guards are RED-first against a confirmed-applied, byte-cleanly-reverted mutation (rule `84d1gfpywd`); Go RED evidence per rule `x1cjy9vyhq`.
- **WR-01** (Phase 4 review, open): `CHANGELOG_EXEMPT_MARKER` in `fragment_policy.py` and `EXEMPT_MARKER` in `pr_template_policy.py` backtrack cubically; fix both with GRD-15's `pr_template_policy.py` edit and update the marker-mirror test.
- `TestGsdTagCreationIsDisabled` passes unchanged; GRD-16's new assertion (the App-token tag step is the only `git tag`/`git push --tags` in `.github/workflows/`) goes RED against a planted second tag push.
- Landing: everything rides the milestone PR (#88); no gate-only PR to `main` this phase. The milestone merge is Phase 6's entry.
- Never `[ci skip]` in a commit message (rule `f18zrdsgx5`).

### Claude's Discretion
- Workflow file layout and name (`release.yml` is taken). One `on:` block cannot path-filter per job, so either two files (release-PR on push to `main`; tag on push to `main` with `paths: [CHANGELOG.md]`) or one file whose tag job checks the push's changed paths — pick the simpler; GRD-16's assertion names whichever file holds the tag step.
- `create-pull-request` inputs (branch name, `delete-branch`, commit message, behaviour when there is no diff), and the concurrency group.
- Taskfile target names for the batch/merge/tag bodies.
- REL-13 rehearsal input: an existing `.changes/v*.md` or a scratch batch of the milestone's real fragments — whichever lets the captured output show "non-empty and equals the file" most directly.
- Where the D-10 verdicts and the census transcript live (a phase artifact such as `05-CENSUS.md`).
- How the SAN negative-case fixtures that name `release-please.yml` (`release_workflow_shape_test.go:889`, `taskfile_shape_test.go:2420`) are re-pointed — they assert a wrong SAN is rejected, so any non-`release.yml` workflow name serves.

### Deferred Ideas (OUT OF SCOPE)
- At Phase 6 entry: if release-please has an open release PR on `main` when the milestone merges, it is orphaned (the merge deletes `release-please.yml`) and should be closed by hand. Phase 6's concern, recorded here so it is not lost.

### Reviewed Todos (not folded)
- "Adopt changie for changelog + version and replace release-please…" (score 0.9, `resolves_phase: 5`): the milestone's origin todo. This phase delivers its checklist steps 2, 3 and 5; it closes at milestone end, as Phases 1–4 recorded.
- "tools/bench/runner pinnedAt() validates a checkout by git rev-parse HEAD alone": bench, unrelated (keyword match only).
- "Reply on GH #85 / #82 / #80 with the reshaped contract": server-mode, unrelated.
- "Pin the Go toolchain so Go 1.27 can build codegraph-go": build, unrelated.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| REL-10 | New release workflow: on push to `main` with non-empty `.changes/unreleased/`, `pretag-gate` then `changie batch auto && changie merge`, open/update one `chore(main): release vX.Y.Z` PR; no fragments = no-op | Documented pattern (changie-action `gen-release-pr.yml` + create-pull-request README) in "Standard Stack"/"Code Examples"; `release:batch` target; empty-set behaviour measured (Pitfall 3); newlines defect that makes the PR unmergeable (Pitfall 1) |
| REL-11 | On release-PR merge, App installation token pushes tag `vX.Y.Z`, never `GITHUB_TOKEN` | changie "release trigger" doc (`paths: [CHANGELOG.md]`), checkout-with-App-token, `release:tag` target with skip-if-exists; two-file layout recommendation; amended wording (D-06) |
| REL-12 | `release.yml` trigger/steps/job shape byte-identical except `goreleaser release` receives `--release-notes .changes/${TAG}.md`; no `changelog.disable: true` | The flag belongs in Taskfile `release:goreleaser` (the single definition), not in `release.yml`'s YAML; there is NO existing shape test for `changelog.disable` (Wave 0 gap); `release.yml:188` comment is a census hit |
| REL-13 | `task release:dry-run` renders a non-empty release body equal to a `.changes/vX.Y.Z.md`, captured before the first tag | Snapshot mode skips the changelog pipe (source + experiment); documented `GORELEASER_CURRENT_TAG` non-snapshot rehearsal verified against the real `.goreleaser.yaml` |
| REL-14 | Delete the three release-please files; `changie latest` is the version source; positive-controlled census | Baseline census list (18 tracked files + 3 filenames) and instrument design in "Code Examples"; Phase-5-fragment hazard (Open Question 5) |
| GRD-15 | Re-point every release-please-pinning test (RED first) | Full guard inventory with line numbers; re-point recipe per guard; WR-01 measurements |
| GRD-16 | `TestGsdTagCreationIsDisabled` unchanged; new "only tag push" assertion RED against a planted second tag push | Guard design that works with job bodies living in Taskfile.yml (the literal `git tag` is not in the workflow YAML) |
| DOCS-12 | `CONTRIBUTING.md` §Pull requests teaches fragments/exemption/derived version | Current text located (`CONTRIBUTING.md:135`, `:171`); `TestContributingReferencesRealTaskTargets` constrains `task <target>` mentions |
| DOCS-13 | `docs/RELEASE-PROCEDURES.md` describes the changie chain; 45 references resolved with verdicts | Per-section reference counts, taxonomy of verdicts, rc-escape-hatch hazard (Open Question 4) |
| DOCS-14 | `pr-title.yml` header "hygiene only"; `release:dry-run-signed` comment and `docs/RELEASE.md` pipeline paragraph name the changie workflow | Exact locations; `pr-title.yml:1-18` is a census hit too |
</phase_requirements>

## Summary

The documented pattern is small and the repo already contains most of its parts. changie's own docs prescribe exactly three things: trigger the release on `push` to `main` with `paths: [CHANGELOG.md]`; run `batch auto` → `merge` → `latest` and hand the result to `peter-evans/create-pull-request`; and feed GoReleaser the generated version file with `--release-notes`. `peter-evans/create-pull-request` documents the token hazard the maintainer already identified and names a GitHub App token (via `actions/create-github-app-token`) as the answer that triggers both `push` and `pull_request` workflows. Nothing here needs a custom mechanism beyond five small, constraint-forced additions, all listed in "Custom logic register" below with the locked constraint that forces each.

Three findings change the plan and none is in CONTEXT. **(1) The first real release PR would be unmergeable with the current `.changie.yaml`.** The Phase 1 config overrides only `newlines.afterChangelogHeader`, because the 15 seed files already end in a blank line. A *newly batched* version file has no trailing newline and no blank line after the heading, so `changie merge` glues the next heading onto the last bullet (`...pull/88))## [0.14.1](...`). I reproduced this against the real tree: with the current config, `TestChangieVersionSeedsMatchChangelog` fails on the post-merge tree ("versions only in seeds: [v0.14.1]"), so the release PR's required `test` check would be red. Adding `newlines: {beforeKind: 1, afterKind: 1, endOfVersion: 2}` renders clean markdown and leaves `changie merge --dry-run` byte-identical on the seeds, but it contradicts `TestChangieConfigShape`'s pin that `afterChangelogHeader` is the ONLY newlines key (Phase 1 D-02). **(2) `task release:dry-run` cannot show a release body as written.** GoReleaser's changelog pipe is skipped under `--snapshot`, so a snapshot run silently ignores `--release-notes` (a missing file still exits 0) and GoReleaser never writes the body to disk even in a real run. The documented non-snapshot form (`GORELEASER_CURRENT_TAG="$(changie latest)"` plus `--skip=publish,sign,validate`) does load the file and fails on a missing one; I ran it against the real `.goreleaser.yaml` and it exited 0. **(3) The `--release-notes` flag lives in `Taskfile.yml`, not `release.yml`.** `release.yml`'s only run body is `task release:goreleaser`; the `goreleaser release --clean` line is in that target. So `release.yml` can stay byte-identical on every functional line, with one comment hunk forced by the census (`release.yml:188`).

**Primary recommendation:** Build two small workflow files (`release-pr.yml` on push to `main`; `release-tag.yml` on push to `main` with `paths: [CHANGELOG.md]`), each with its own verbatim `pretag-gate`, their job bodies as `task release:batch` and `task release:tag`; fix `.changie.yaml` newlines first (RED-first, amending the Phase 1 pin deliberately); put `--release-notes` in `release:goreleaser` and rehearse it with a non-snapshot `release:dry-run`; re-point guards RED-first; delete the three release-please files last, after the census instrument has been shown to find a planted reference.

## Project Constraints (from CLAUDE.md)

Treated with the authority of locked decisions. Source: `./.claude/CLAUDE.md` and the user's global `~/.claude/CLAUDE.md` rules.

- **Tech stack:** Go, single static binary; **supply chain:** minimal audited dependencies; every action SHA-pinned with a version comment (also a milestone rule). No new Go dependency is needed or proposed.
- **GSD workflow enforcement:** repo edits go through a GSD command (`/gsd-execute-phase` for this phase's plans).
- **Tool-owned generated files (global rule `generated-files.md`):** do not invent structure in `CHANGELOG.md`, `.changes/v*.md`, `.planning/*` state files, lockfiles. Fill values in existing shapes only. D-06's REQUIREMENTS/ROADMAP amendment is a value edit inside the existing clause. Never hand-edit `.changes/v*.md`.
- **Go toolchain:** `go.mod` declares `go 1.26.9`; the default `go` on this machine is 1.27.2, which breaks `cockroachdb/swiss`. Every local Go command takes the prefix `GOTOOLCHAIN=go1.26.9`. [VERIFIED: `go version` printed `go1.27.2`; `GOTOOLCHAIN=go1.26.9 go version` printed `go1.26.9`]
- **Rules:** `eyva2svrcz` (never test third-party behaviour), `84d1gfpywd` (positive assertion + RED against a confirmed-applied, byte-cleanly-reverted mutation), `x1cjy9vyhq` (Go RED = `test(NN-PP):` commit before implementation + pasted `--- FAIL:` transcript in the SUMMARY; never gate on `gsd-tools check tdd-red-evidence`), `f18zrdsgx5` (never `[ci skip]`/`[skip ci]` in a commit message).
- **Commits:** conventional commits, atomic, end with the attribution lines supplied by the session (`Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>`, `Claude-Session: <url>`). Do not amend or rebase.
- **Shell idiom in Taskfile targets:** Task interprets `cmds:` with its built-in shell, not bash. Follow the existing targets' idioms (`find ... | wc -l`, `case`, `[ ... ]`, `set -euo pipefail`), and prove a new target by running it through `task`, not `bash -c`. [VERIFIED: Taskfile.yml:580-720 `check:changie` uses only those constructs]
- **Docs/prose:** no emojis; user-facing text plain.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Decide whether a release PR is needed (fragment count) | Taskfile target (`release:batch`) | Workflow `if:` on the step output | Job bodies are `task` calls by repo rule (`TestWorkflowRunBodiesInvokeTask`); the workflow only branches on an output |
| Batch/merge/derive version | changie via `task`'s pinned modfile | — | CHG-01's single recorded install path |
| Open/update the one release PR | `peter-evans/create-pull-request` (Actions) | App token | Documented action; fixed branch is regenerated from `main` |
| Make the PR's required checks report | GitHub App installation token | — | `GITHUB_TOKEN` PRs fire no `pull_request` workflows (action README) |
| Gate on cross-platform resolution | `pretag-gate` job (`task check:cross`) | — | D-07; `needs:` in both workflow files |
| Detect "a release PR merged" | GitHub `on.push.paths: [CHANGELOG.md]` filter | `release:tag` skip-if-exists | Documented changie trigger; no custom path detection |
| Create and push the tag | `release:tag` target run in a job whose checkout holds the App token | — | Tag pushed with `GITHUB_TOKEN` would not fire `release.yml` |
| Build, sign, attest, publish | `release.yml` + GoReleaser (unchanged) | — | Byte-locked contract |
| Authoring the Release body | GoReleaser, from `.changes/${TAG}.md` | — | `--release-notes` documented integration |
| Hygiene exemption for the release PR | `scripts/pr_template_policy.py` + `require-issue-link.yml` (mirrored pair) | — | By path prefix `.changes/` (D-05) |
| Prove the above | `internal/upgrade/*_shape_test.go`, `scripts/test_fragment_policy.py`, `task check:*` | actionlint | Tests cover this repo's files only |

## Standard Stack

### Core

| Component | Version / pin | Purpose | Why standard |
|-----------|---------------|---------|--------------|
| changie | v1.26.0 via `go.tool-changie.mod` and `task changie` | `batch auto`, `merge`, `latest`, `next` | Phase 1 D-04 pin; `changie --version` on this machine prints `changie version v1.26.0` [VERIFIED: run this session] |
| `peter-evans/create-pull-request` | `5f6978faf089d4d20b00c7766989d076bb2fc7f1` `# v8.1.1` (published 2026-04-10, ~6 months old) | Open/update the single release PR | Named by changie's own `gen-release-pr.yml` (`@v8`); `v8.1.1` is the newest release, `v8` and `v8.1.1` tags both resolve to this commit [VERIFIED: `gh api repos/peter-evans/create-pull-request/git/ref/tags/v8.1.1` returned `5f6978faf089d4d20b00c7766989d076bb2fc7f1`, type `commit`; releases list newest = v8.1.1] |
| `actions/create-github-app-token` | `bcd2ba49218906704ab6c1aa796996da409d3eb1` `# v3.2.0` (reuse the repo pin) | Mint the App installation token | Already used for release-please and release.yml; v3.3.0 exists but is one day old, so stay on the repo pin [VERIFIED: `gh api repos/actions/create-github-app-token/releases` newest v3.3.0 2026-10-12] |
| `actions/checkout` | `df4cb1c069e1874edd31b4311f1884172cec0e10` `# v6.0.3` | Checkout; the tag job passes `token:` = App token | Repo pin, quoted from `release-please.yml:44` |
| `actions/setup-go` | `924ae3a1cded613372ab5595356fb5720e22ba16` `# v6.5.0` | Go for `task` bootstrap (`go-version-file: go.mod`) | Repo pin, `release-please.yml:47` |
| `./.github/actions/install-task` | local composite | Builds the pinned `task` | Existing, `release-please.yml:52` |
| GoReleaser | v2.17.1 (`go.tool.mod`, `GORELEASER_VERSION`) | `--release-notes` | Unchanged pin |

Verbatim quotes of the reused pins (rule: discrete in-repo values) from `.github/workflows/release-please.yml`:
- line 44: `uses: actions/checkout@df4cb1c069e1874edd31b4311f1884172cec0e10 # v6.0.3`
- line 47: `uses: actions/setup-go@924ae3a1cded613372ab5595356fb5720e22ba16 # v6.5.0`
- line 81: `uses: actions/create-github-app-token@bcd2ba49218906704ab6c1aa796996da409d3eb1 # v3.2.0`

### Supporting

| Component | Purpose | When to use |
|-----------|---------|-------------|
| `miniscruff/changie-action` | Installs changie in CI | **Do not use.** D-02/CHG-01: `task changie` is the one CI install path. The action's `gen-release-pr.yml` is only the *pattern* source |
| `permission-contents` / `permission-pull-requests` inputs on the App-token step | Narrow the minted token below the installation's grant | Optional hardening; inputs exist at the pinned SHA [VERIFIED: `action.yml` at `bcd2ba49` lists `permission-contents`, `permission-pull-requests`]. Not required by the docs; skip unless the maintainer wants it (D-01) |

### Alternatives Considered

| Instead of | Could use | Tradeoff |
|------------|-----------|----------|
| Two workflow files | One file, tag job checks `git diff` for `CHANGELOG.md` | Needs custom path detection (`github.event.before` is the zero SHA on some pushes) where the docs answer with `paths:`. Two files cost one duplicated 4-step `pretag-gate` job |
| Path trigger | State-based "tag if `changie latest` has no tag, on every push" | Rejected by D-06, and with good reason: the rollback runbook (`docs/RELEASE-PROCEDURES.md` §7) deletes tags, and a state-based job would resurrect a deleted tag on the next unrelated push |
| `changie-action`'s `release/<version>` branch | A fixed branch | `release/<version>` makes a new PR each time the derived version changes; D-02 wants one PR on a fixed branch |

**Installation:** none. No new package enters `go.mod` or any tool modfile. [VERIFIED: every component above is a GitHub Action or an existing pin]

## Package Legitimacy Audit

No npm, PyPI or crates packages are added, so the registry seam (`package-legitimacy check --ecosystem npm|pypi|crates`) does not apply. The only new external dependency is one GitHub Action, checked against the GitHub API instead:

| Package | Registry | Age | Adoption | Source Repo | Verdict | Disposition |
|---------|----------|-----|----------|-------------|---------|-------------|
| `peter-evans/create-pull-request` | GitHub Actions | created 2019-07-16; v8.1.1 2026-04-10 | 2,859 stars, MIT, not archived, pushed 2026-10-12 | github.com/peter-evans/create-pull-request | OK (manual check) | Approved, SHA-pinned; also the action named by changie's own `gen-release-pr.yml` |
| `actions/create-github-app-token` | GitHub Actions (first-party) | already pinned in 2 workflows | — | github.com/actions/create-github-app-token | OK | Reuse existing pin |

**Packages removed due to [SLOP] verdict:** none. **Flagged [SUS]:** none.
Note for the planner: `create-pull-request` is a `node24` action that receives the App token (`using: node24` in its `action.yml`), so it is inside the token's trust boundary; the full-SHA pin and the 6-month age are the controls. [VERIFIED: `cpr-action.yml` fetched at tag v8.1.1]

## Architecture Patterns

### System Architecture Diagram

```
 feature PR (carries .changes/unreleased/<Kind>-<ts>.yaml, gated by fragment-required)
        |  squash-merge
        v
 push to main ----------------------------------------------------------------+
        |                                                                      |
        v                                                                      v
 release-pr.yml  (on: push: branches:[main])                       release-tag.yml  (on: push: branches:[main], paths:[CHANGELOG.md])
   pretag-gate -- task check:cross                                   pretag-gate -- task check:cross
        | needs                                                            | needs
        v                                                                  v
   job release-pr                                                     job tag
     1 mint App token (APP_ID / APP_PRIVATE_KEY)                        1 mint App token
     2 checkout main                                                    2 checkout, token = App token (git push credential)
     3 setup-go, install-task                                           3 setup-go, install-task
     4 task release:batch                                               4 task release:tag
          no *.yaml fragments?  -> print, exit 0, no output                 TAG = changie latest
          else: changie batch auto -> merge -> latest                       tag already on origin? -> exit 0 (success)
                writes version=vX.Y.Z to GITHUB_OUTPUT                      else git tag TAG ; git push origin refs/tags/TAG
     5 create-pull-request  (if version output non-empty)                              |
          token = App token ; branch changie/release                                   | tag push authored by the App
          title "chore(main): release vX.Y.Z" ; body-path .changes/vX.Y.Z.md          v
          draft false                                              release.yml  (UNCHANGED trigger: push tags v[0-9]*)
        |                                                             task release:goreleaser
        v                                                               goreleaser release --clean --release-notes .changes/${TAG}.md
 release PR (App-authored) --> ci.yml + hygiene gates RUN (App token)        |
        |  human reviews notes + version, squash-merges                      v
        v                                                           GitHub Release (body = the .changes file) --> post-release-verify (workflow_run on "release")
 push to main touching CHANGELOG.md ---------------------------------> release-tag.yml (right-hand branch above)
 (the same push also triggers release-pr.yml: unreleased/ is now empty, so it no-ops -> no loop)
```

### Recommended Project Structure

```
.github/workflows/
├── release-pr.yml          # NEW  name: release-pr    jobs: pretag-gate, release-pr
├── release-tag.yml         # NEW  name: release-tag   jobs: pretag-gate, tag
├── release.yml             # unchanged except one comment hunk (census, :188)
└── release-please.yml      # DELETED last
Taskfile.yml                # + release:batch, release:tag ; release:goreleaser (+ --release-notes) ; release:dry-run (non-snapshot)
.changie.yaml               # + newlines keys (Pitfall 1)
internal/upgrade/
├── changie_release_shape_test.go   # NEW  workflow + Taskfile shape guards for the new chain (REL-10/11, GRD-16)
├── tag_authority_test.go           # re-pointed + renamed test
└── taskfile_shape_test.go / goreleaser_shape_test.go / release_workflow_shape_test.go   # registry + fixtures + new changelog guard
scripts/pr_template_policy.py, scripts/fragment_policy.py, scripts/test_fragment_policy.py   # exemption swap + WR-01
.planning/phases/05-.../05-CENSUS.md   # D-10 verdicts + census transcripts (phase artifact)
```

Name constraint for both new workflows: `post-release-verify.yml` triggers on `workflow_run: workflows: ["release"]`, an exact match on a workflow's display `name:`. Neither new workflow may be named `release`. [VERIFIED: `.github/workflows/post-release-verify.yml` header lines 35-40 and `TestReleaseWorkflowFileMatchesPattern` asserts release.yml's name is `release`]

### Pattern 1: Documented release trigger (REL-11)

**What:** changie's integration page prescribes this trigger and nothing else for the "release" half.
**Source:** `docs/integrations/release_trigger.md` in miniscruff/changie, fetched raw this session:
```yaml
on:
  push:
    branches: [ main ] # your default branch if different
    paths: [ CHANGELOG.md ] # your changelog file if different
```
GitHub documents that when both `branches` and `paths` are defined the workflow runs only when both match, and that a push diff over 3,000 files may not run a path-filtered workflow (and always runs on more than 1,000 commits). [CITED: docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax] The milestone merge is a large diff; either outcome (workflow runs and skips, or does not run) is harmless because the tag job skips an existing tag.

### Pattern 2: Documented release-PR job (REL-10)

**Source:** `.github/workflows/gen-release-pr.yml` in miniscruff/changie-action (fetched raw): `batch auto`, `merge`, `latest` (its output is the version), then
```yaml
- name: Create Pull Request
  uses: peter-evans/create-pull-request@v8
  with:
    title: Release ${{ steps.latest.outputs.output }}
    branch: release/${{ steps.latest.outputs.output }}
    commit-message: Release ${{ steps.latest.outputs.output }}
```
The repo adapts only: a fixed branch (D-02), `token:` = App token (D-03, create-pull-request README), `body-path` (D-04), title `chore(main): release <version>`, and `task release:batch` in place of the changie-action steps (D-02).

### Pattern 3: App token on the PR action (D-03)

**Source:** create-pull-request `docs/concepts-guidelines.md`, "Authenticating with GitHub App generated tokens" (fetched at v8.1.1): mint with `actions/create-github-app-token` (`app-id`, `private-key`), then `token: ${{ steps.generate-token.outputs.token }}` on the action; "This method will trigger both `on: push` and `on: pull_request` workflows." The README adds that the "Allow GitHub Actions to create and approve pull requests" setting is required for the *default* token. This repo's setting already reads `can_approve_pull_request_reviews: true`, and the App token path does not depend on it. [VERIFIED: `gh api repos/seanb4t/codegraph-go/actions/permissions/workflow`]

Empirical confirmation that App-authored PRs get the full required-check set: the last release PR (#98, author `app/fzy-release-please`) carried 16 check contexts including `test`, `fragment-required`, `pr-title`, `Issue link required` and `Pull request template format`. [VERIFIED: `gh pr view 98 --json statusCheckRollup`]

### Pattern 4: GoReleaser consumes the version file (REL-12)

**Source:** changie `docs/integrations/goreleaser.md` (fetched raw): leave `changelog.disable` false; pass `--release-notes` the file for the latest version; the page's own invocation is `goreleaser --release-notes="changes/$(changie latest)" --skip-validate` and says `--skip-validate` is only needed "because we skip the git tag. If you choose to tag the commit instead you do not need to use this." This repo tags first, so the real release needs no skip. Source check of the pinned GoReleaser: the changelog pipe's `Skip` returns true when `ctx.Snapshot` or `Changelog.Disable` evaluates true; its `Run` loads the file into `ctx.ReleaseNotes` and returns before writing anything to `dist/`; the release body is `Header + ReleaseNotes + Footer` through a template that ends with one newline. [VERIFIED: read `internal/pipe/changelog/changelog.go` and `internal/pipe/release/body.go` in `goreleaser/v2@v2.17.1`]

Divergence from changie's *own* `release.yml`, which strips the heading with `tail -n +3 .changes/$(go run main.go latest).md > rel_notes.md`: this repo passes the file verbatim because SHIP-02 requires the Release body to equal `.changes/v0.15.0.md`. Forced by REQUIREMENTS REL-12/SHIP-02. [VERIFIED: changie `.github/workflows/release.yml` fetched raw]

### Custom logic register (D-01: each addition names its forcing constraint)

| # | Custom logic | Documented answer it departs from | Locked constraint that forces it |
|---|---|---|---|
| C1 | `release:batch` counts `*.yaml` in `.changes/unreleased/` and exits 0 before `batch auto` | changie-action's example has no empty guard | CHG-03: `changie batch auto` exits 1 on an empty set ("no unreleased changes found for automatic bumping") [VERIFIED: ran it] |
| C2 | `release:batch` fails if the rendered `.changes/vX.md` contains a GitHub skip-ci phrase | none | Repo setting `squash_merge_commit_message: PR_BODY` makes the PR body the commit message on `main`; a `[skip ci]` there suppresses the tag workflow's push run [VERIFIED: `gh api repos/seanb4t/codegraph-go` returned `"squash_msg":"PR_BODY"`; skip phrases [CITED: docs.github.com skipping-workflow-runs]] |
| C3 | `release:tag` skips when the tag already exists on `origin` | changie's trigger doc leaves the job to the reader | D-06: the milestone merge changes `CHANGELOG.md` and carries 15 already-tagged seeds |
| C4 | `release:tag` and `release:batch` live in `Taskfile.yml`, not inline `run:` | docs show inline shell | `TestWorkflowRunBodiesInvokeTask` requires every in-scope run body to be exactly `task <target>` |
| C5 | `.changie.yaml` gains `newlines` keys | none (changie defaults) | CHG-02 keeps the 15 seed files verbatim; their blank-line tail differs from changie's default output (Pitfall 1) |
| C6 | `release:goreleaser`/`release:dry-run` assert the notes file exists and is non-empty | GoReleaser only warns on an empty file | REL-13 "body is non-empty" [VERIFIED: scratch run printed `loaded ".changes/v0.2.0.md", but it is empty` and exited 0] |
| C7 | A second workflow file duplicates `pretag-gate` | one workflow | D-07 (both paths gated) + the fact that one `on:` block cannot path-filter per job |

Everything else in the phase (action choice, token handling, trigger, flag) is the documented pattern.

### Anti-Patterns to Avoid

- **Minting the App token for the whole workflow or storing it as a job output.** GitHub drops a masked job output entirely (observed and recorded in `release.yml` lines 169-175, run 31417685002). Mint inside the job that consumes it, as `release.yml` does for the tap token.
- **A tag step with `GITHUB_TOKEN`** (`github.token`, `secrets.GITHUB_TOKEN`) in the checkout: the tag push would not start `release.yml`.
- **`draft: true` / `always-true`** on the PR (D-04): `close-draft-prs.yml` closes App-authored drafts.
- **`changelog.disable: true` or any `release:` header/footer/name_template/draft/disable key** in `.goreleaser.yaml`: the first empties the body, the others change it so it no longer equals the file.
- **Inventing structure in `CHANGELOG.md` or `.changes/v*.md`** (global rule); fix the generator's config instead.

## Don't Hand-Roll

| Problem | Don't build | Use instead | Why |
|---------|-------------|-------------|-----|
| Opening/updating one release PR | `gh pr create`/`gh pr edit` + `git push --force` script | `peter-evans/create-pull-request` | It owns "create if absent, update if present, close if no diff" and the token split |
| Detecting a release merge | `git diff HEAD^ HEAD -- CHANGELOG.md` in a step, or `github.event.head_commit.modified` | `on.push.paths: [CHANGELOG.md]` | changie's documented trigger; zero custom code |
| Version derivation | Parsing commits/PR titles | `changie batch auto` / `changie latest` | The whole point of the milestone |
| Release body | A step that renders notes or edits the Release with `gh release edit` | `goreleaser release --release-notes FILE` | Documented integration; GoReleaser creates draft, uploads, then publishes [VERIFIED: `internal/client/github.go` `CreateRelease` "Always start with a draft release while uploading artifacts"] |
| Census | A bespoke walker in application code | `git grep` / `git ls-files` plus a planted-reference self-test | D-09 "plain grep census" |
| Tag authority guard | Regex over only `.github/workflows/*.yml` | A scan that follows `task <target>` into `Taskfile.yml` (Pitfall 6) | The literal `git tag` is not in the workflow YAML |

**Key insight:** every piece that looks like it needs custom code (empty-set, existing-tag, release detection) is either answered by a documented trigger or reduces to a three-line shell guard whose forcing constraint is already locked.

## Runtime State Inventory

This phase retires a system; the question is what still holds release-please state after every in-repo file is gone.

| Category | Items found | Action required |
|----------|-------------|-----------------|
| Stored data | PR labels `autorelease: pending` / `autorelease: tagged` on historical release PRs (#65…#98); the release-please manifest in git (`.release-please-manifest.json`) | None for history. Manifest is deleted by REL-14 (code edit). If an open release-please PR exists on `main` at the milestone merge, close it by hand (CONTEXT deferred idea, Phase 6) |
| Live service config | GitHub App installation: PR author reads `app/fzy-release-please`, Release author `fzy-release-please[bot]`. The App's *display name* contains "release-please" and cannot be changed by a file edit. Installation permissions include Issues: read/write that only release-please needed | Do **not** rename mid-milestone (Phase 6 checks may match the author login). In-repo docs must not quote the App slug (census). Optional, out-of-repo, later: drop the Issues permission. Ruleset `protect-main` (20157557) has no tag rule, so App tag pushes are unrestricted [VERIFIED: `gh api repos/seanb4t/codegraph-go/rulesets` lists only `protect-main`, target `branch`] |
| OS-registered state | None. No scheduled tasks, launchd or systemd entries mention release-please | None — verified by `git grep` over tracked files (census baseline below) |
| Secrets / env vars | `APP_ID`, `APP_PRIVATE_KEY` unchanged and reused by name [VERIFIED: `gh secret list`]. No secret is named after release-please | None; keep the names (the docs' §9 explains them) |
| Build artifacts | None carry the name. Local `dist/` from dry-runs is gitignored scratch | None |

## Common Pitfalls

### Pitfall 1: The first release PR is unmergeable because newly batched files lack the seed files' tail (HIGH impact, not in CONTEXT)

**What goes wrong:** `changie batch auto` writes `.changes/v0.15.0.md` as heading, `### Breaking`, bullet, and no trailing newline. `changie merge` concatenates version files with no separator, so CHANGELOG.md ends up with `...pull/88))## [0.14.1](...`. The heading is no longer at the start of a line, so `parseChangelogVersionHeadings` misses it and `TestChangieVersionSeedsMatchChangelog` fails, which turns the release PR's required `test` check red.
**Why:** `.changie.yaml` sets only `newlines.afterChangelogHeader: 1` because the seeds end in `\n\n`. [VERIFIED: seeds end `b'...))\n\n'`; current config `.changie.yaml:53-54` reads `newlines:` / `  afterChangelogHeader: 1`]
**Reproduction (this session):** copy the tree, run `changie batch auto && changie merge`, then `go test ./internal/upgrade -run Changie`: `FAIL TestChangieVersionSeedsMatchChangelog ... versions only in seeds: [v0.14.1], versions only in changelog headings: []`.
**Fix (measured):** in `.changie.yaml`
```yaml
newlines:
  afterChangelogHeader: 1
  beforeKind: 1
  afterKind: 1
  endOfVersion: 2
```
With one `Breaking`, one `Features` and one `Fixes` fragment this renders heading, blank, `### Breaking`, blank, bullet, blank, `### Features`, ..., ending `\n\n`, and merged headings each start a line. `changie merge --dry-run` stayed byte-identical to `CHANGELOG.md` on the existing seeds, `task check:changie` passed 13 of 13, and the full `internal/upgrade` suite failed only `TestChangieConfigShape`. [VERIFIED: ran each in scratch trees]
**Consequence for the plan:** `TestChangieConfigShape` (`changie_shape_test.go:283`: `wantNewlinesKeys := []string{"afterChangelogHeader"}`) must change in the same RED-first step, and the `.changie.yaml` header comment about "the ONLY newlines override" must be rewritten (it is also the file's census hit at line 20). This amends Phase 1 D-02 deliberately; surface it to the maintainer in the plan, not silently. Add a `check:changie` leg that batches a probe fragment onto the seeds in the scratch copy and asserts every `## [` heading starts a line and the new file ends with a blank line (this exercises this repo's `.changie.yaml`, not changie's logic).
**Warning signs:** a CHANGELOG diff in a rehearsal where a `## [` heading is not preceded by a blank line.

### Pitfall 2: `release:dry-run` proves nothing about release notes in snapshot mode

**What goes wrong:** `release:dry-run` runs `release --snapshot`; the changelog pipe is skipped for snapshots, so `--release-notes does-not-exist.md` still exits 0 and the body is never loaded.
**Evidence:** scratch repo, same pinned GoReleaser: snapshot + missing file = `release succeeded`; non-snapshot + missing file = `generating changelog ⨯ release failed ... open .changes/nope.md: no such file or directory`; non-snapshot + present file = succeeds, and the body is not written anywhere under `dist/`. An empty file only warns. [VERIFIED: ran all four]
**Fix:** rehearse the way changie's page documents for the pre-tag case: `GORELEASER_CURRENT_TAG="$(changie latest)"`, `--skip=publish,sign,validate` (no `--snapshot`), `--release-notes ".changes/${TAG}.md"`. I ran exactly this against the real `.goreleaser.yaml` in a scratch commit with `GORELEASER_CURRENT_TAG=v0.14.1 ... --release-notes .changes/v0.14.1.md`: four targets built, `generating changelog` ran, archives/SBOM/checksums/homebrew cask rendered, exit 0 in about 14 s with a warm cache. [VERIFIED: ran it; needs `zig`, `syft`, `cosign` on `PATH`]
**What "equals the file" can honestly mean:** GoReleaser's body is `file + "\n"`; it is third-party behaviour and the rules forbid testing it. This repo's own code can prove: the target passes the exact path `.changes/${TAG}.md`, refuses an absent or empty file, and prints a `RELEASE-NOTES-EVIDENCE tag=... file=... bytes=... sha256=...` line the SUMMARY can capture. The Release-body byte comparison is Phase 6's SHIP-02, and should be defined trailing-whitespace-insensitively (Open Question 3).

### Pitfall 3: `changie batch auto` fails on an empty set; every push runs the release-PR job

**What goes wrong:** the release-PR workflow fires on every push to `main`, including the release PR's own merge, when `unreleased/` is empty. Without C1 the job would go red on every non-release push. [VERIFIED: `changie batch auto` on an empty dir exits 1 with `no unreleased changes found for automatic bumping`]
**Fix:** C1, using `find .changes/unreleased -maxdepth 1 -name '*.yaml' | wc -l` (the `.gitkeep` is not counted; `batch` leaves it in place [VERIFIED: `ls -a .changes/unreleased` after a batch]).

### Pitfall 4: A hand-pushed rc tag now fails the release

**What goes wrong:** `docs/RELEASE-PROCEDURES.md` §4 retains a hand-pushed `v0.0.0-rc.N` escape hatch (lines ~177-194). With REL-12, `release:goreleaser` demands `.changes/v0.0.0-rc.N.md`, which does not exist, so GoReleaser fails (non-snapshot + missing file = error). [VERIFIED: Pitfall 2 experiment]
**Fix options for the planner (Open Question 4):** keep the hard failure (fail loud is correct) and rewrite the runbook; changie can batch a prerelease (`changie batch auto -p rc.1` wrote `.changes/v0.15.0-rc.1.md`, and `changie latest` then answered `v0.15.0-rc.1`) [VERIFIED: ran it], but `release:batch` does not pass `-p`, so rc cuts stay out of the automated chain this phase.

### Pitfall 5: A new workflow named `release` or a rename of `release.yml`

`post-release-verify.yml` is keyed on the display name `release`; cosign's SAN pins `release.yml`. Name the new files `release-pr` / `release-tag`. The SAN-negative fixtures should name the real new tag workflow (`release-tag.yml`), which is the sharpest "wrong workflow, right repo, tag ref" case.

### Pitfall 6: The tag-authority guard cannot grep workflow YAML for `git tag`

Because job bodies must be `task <target>`, the literal `git tag` lives in `Taskfile.yml`'s `release:tag`. A guard that scans only `.github/workflows/*.yml` would pass vacuously on the real repo and on a planted second tag push placed in a Taskfile target. The guard must (a) scan workflow `run:` bodies for tag-creating commands, (b) resolve each `task <target>` run body to that target's `cmds` in `Taskfile.yml` and scan those, (c) assert the total count of tag-creating invocations reachable from workflows is exactly 1, (d) assert that one lives in `release-tag.yml`'s `tag` job, whose checkout `with.token` is the output of the preceding `actions/create-github-app-token` step, and that no `github.token`/`secrets.GITHUB_TOKEN` is passed there, and (e) carry a positive assertion that the scan saw at least one workflow, one resolved task target, and the one tag step (rule `84d1gfpywd`). Tag-creating patterns to cover: `git tag`, `git push` with `--tags` or `refs/tags`, `gh api ... git/refs` with a tag ref, `gh release create`. Baseline today: zero hits in `.github/workflows/` and `Taskfile.yml`. [VERIFIED: `rg "git tag|git push|push --tags|--tags|refs/tags|git/refs|gh release" .github Taskfile.yml scripts` showed only `git describe --exact-match --tags`, `gh release download/view`, `refs/tags/` in cosign identity regexes and comments]

### Pitfall 7: WR-01 regexes are worse than the review's table suggests for some inputs and the fix must preserve edge semantics

Measured here on `"<!-- pr-template-exempt: a" + " " * n` (no closer): n=2000 0.005 s, 4000 0.018 s, 8000 0.069 s (about quadratic on this shape; the review measured cubic on a different shape). A body of 65k characters is a multi-second stall per event. [VERIFIED: ran `re` timing]
A drop-in replacement `<!--\s*pr-template-exempt:([^>]*)-->` with the reason taken as `.group(1).strip()` ran in microseconds. The old pattern accepted a reason beginning with `-` when preceded by whitespace (`<!-- pr-template-exempt: --x -->` returned `--x`, the replacement returned none). Decide which is intended, state it in a test, and edit both patterns in lockstep because `test_marker_grammar_mirrors_pr_template_exempt` (`scripts/test_fragment_policy.py:173-181`) compares `pattern` text with `pr_template-exempt` replaced by `changelog-exempt`. RED test: a 60k-character adversarial body must finish under a generous bound (the old pattern needs several seconds there, the fix milliseconds), plus equivalence cases for the grammar kept. `check:fragment-policy` has a floor of 36 defined tests in `Taskfile.yml`; added tests require raising that floor or the gate stops binding the new ones.

### Pitfall 8: Local `MACOS_*` variables defined-but-empty turn notarization on

Running the real `.goreleaser.yaml` in this shell failed at `sign & notarize macOS binaries` with `unable to read p12 bytes: unable to base64 decode key` because `MACOS_SIGN_P12` and its four siblings exist in the environment as empty strings. `env -u` on all five let the same command finish. This affects local rehearsal of `release:dry-run`, not CI. [VERIFIED: `env | rg '^MACOS_'` listed five empty variables; reran with `env -u`] Consider a precondition in `release:dry-run` that names the five variables when any is set-but-empty.

### Pitfall 9: Merge-back and ordering hazards inside the phase

- Delete `release-please.yml`, the config and the manifest only after the tests that read them are re-pointed, or the suite is red between commits. `TestReleasePleaseStaysPreMajor` reads `release-please-config.json`; `inScopeWorkflowFiles` and `inScopeJobs` name `release-please.yml`; `usesOnlyJobExceptions` names its job.
- Until the milestone merges, release-please stays live on `main`; a release cut there needs the quick `261013-esj` reconciliation (`--no-ff` merge plus a verbatim seed `.changes/vX.Y.Z.md`).
- Do not write `[skip ci]`/`[ci skip]` in any commit (rule `f18zrdsgx5`); the same phrases in a fragment body are the C2 hazard.

## Code Examples

Source tags: [CITED] documented pattern; [VERIFIED] run or read this session; shapes are proposals for the planner.

### `.github/workflows/release-pr.yml` (shape)

```yaml
name: release-pr

on:
  push:
    branches: [main]

permissions:
  contents: read

concurrency:
  group: release-pr
  cancel-in-progress: false

jobs:
  pretag-gate:
    name: pre-tag 4-target go list sanity sweep
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@df4cb1c069e1874edd31b4311f1884172cec0e10 # v6.0.3
      - name: Set up Go
        uses: actions/setup-go@924ae3a1cded613372ab5595356fb5720e22ba16 # v6.5.0
        with:
          go-version-file: go.mod
      - name: Install Task
        uses: ./.github/actions/install-task
      - name: Run check:cross sweep
        run: task check:cross

  release-pr:
    name: release PR
    needs: pretag-gate
    runs-on: ubuntu-latest
    steps:
      - name: Mint GitHub App installation token
        id: app-token
        uses: actions/create-github-app-token@bcd2ba49218906704ab6c1aa796996da409d3eb1 # v3.2.0
        with:
          app-id: ${{ secrets.APP_ID }}
          private-key: ${{ secrets.APP_PRIVATE_KEY }}
      - name: Checkout
        uses: actions/checkout@df4cb1c069e1874edd31b4311f1884172cec0e10 # v6.0.3
      - name: Set up Go
        uses: actions/setup-go@924ae3a1cded613372ab5595356fb5720e22ba16 # v6.5.0
        with:
          go-version-file: go.mod
      - name: Install Task
        uses: ./.github/actions/install-task
      - name: Batch fragments and render the release
        id: batch
        run: task release:batch
      - name: Open or update the release PR
        if: steps.batch.outputs.version != ''
        uses: peter-evans/create-pull-request@5f6978faf089d4d20b00c7766989d076bb2fc7f1 # v8.1.1
        with:
          token: ${{ steps.app-token.outputs.token }}
          branch: changie/release
          title: "chore(main): release ${{ steps.batch.outputs.version }}"
          commit-message: "chore(main): release ${{ steps.batch.outputs.version }}"
          body-path: .changes/${{ steps.batch.outputs.version }}.md
          draft: false
```
The job `name:` of `pretag-gate` in the original reads `pre-tag 6-target go list sanity sweep` while `check:cross` runs four pairs; the original comment cites "six"; reword the stale count when moving (census rewrites the comment anyway). The step `Run check:cross sweep` with `run: task check:cross` is the "verbatim" part. [VERIFIED: `release-please.yml:34-55`; `Taskfile.yml` `check:cross` loops four pairs]
`branch:` default would be `create-pull-request/patch`; the fixed name is Claude's discretion. `delete-branch` can stay at its default because the repo auto-deletes merged head branches [VERIFIED: `delete_branch_on_merge: true`]. `release:batch` writes no `version` output when there are no fragments, so the PR step is skipped (the action itself also "exits silently" when there is no diff [CITED: create-pull-request README "Action behaviour"]).

### `.github/workflows/release-tag.yml` (shape)

```yaml
name: release-tag

on:
  push:
    branches: [main]
    paths: [CHANGELOG.md]

permissions:
  contents: read

concurrency:
  group: release-tag
  cancel-in-progress: false

jobs:
  pretag-gate:          # identical to release-pr.yml's
    ...
  tag:
    name: tag
    needs: pretag-gate
    runs-on: ubuntu-latest
    steps:
      - name: Mint GitHub App installation token
        id: app-token
        uses: actions/create-github-app-token@bcd2ba49218906704ab6c1aa796996da409d3eb1 # v3.2.0
        with:
          app-id: ${{ secrets.APP_ID }}
          private-key: ${{ secrets.APP_PRIVATE_KEY }}
      - name: Checkout
        uses: actions/checkout@df4cb1c069e1874edd31b4311f1884172cec0e10 # v6.0.3
        with:
          token: ${{ steps.app-token.outputs.token }}   # the credential `git push` uses
      - name: Set up Go
        ...
      - name: Install Task
        uses: ./.github/actions/install-task
      - name: Tag the release
        run: task release:tag
```
The checkout `token:` input is how the push authenticates; the default `persist-credentials` stays on for this one job (do not set `false`).

### Taskfile targets (shape; idioms follow `check:changie`)

```yaml
  release:batch:
    desc: >-
      Release-PR job body (REL-10): batch the unreleased fragments, merge them
      into CHANGELOG.md, and publish the derived version as a step output.
      With no fragments it prints a notice and exits 0 (changie batch auto
      fails on an empty set, CHG-03). Fails if the rendered notes carry a
      GitHub skip-ci phrase, because the PR body becomes the squash commit
      message on main.
    silent: true
    cmds:
      - |
        set -euo pipefail
        count=$(find .changes/unreleased -maxdepth 1 -type f -name '*.yaml' | wc -l | tr -d ' ')
        if [ "${count}" -eq 0 ]; then
          echo "release:batch: no unreleased fragments — nothing to release"
          exit 0
        fi
        {{.GO_TOOL_CHANGIE}} changie batch auto
        {{.GO_TOOL_CHANGIE}} changie merge
        version=$({{.GO_TOOL_CHANGIE}} changie latest)
        case "${version}" in v[0-9]*.[0-9]*.[0-9]*) ;; *) echo "::error::release:batch: changie latest = '${version}'"; exit 1 ;; esac
        if grep -qiE '\[(skip ci|ci skip|no ci|skip actions|actions skip)\]' ".changes/${version}.md"; then
          echo "::error::release:batch: .changes/${version}.md contains a skip-ci phrase; the PR body becomes the commit message on main"
          exit 1
        fi
        echo "release:batch: rendered ${version} from ${count} fragment(s)"
        if [ -n "${GITHUB_OUTPUT:-}" ]; then echo "version=${version}" >> "${GITHUB_OUTPUT}"; fi

  release:tag:
    desc: >-
      Tag-job body (REL-11). Tags HEAD with `changie latest` and pushes only
      that ref. A tag already on origin is success, not failure (D-06). The
      checkout's credential must be the GitHub App token; a GITHUB_TOKEN push
      would not start release.yml.
    silent: true
    cmds:
      - |
        set -euo pipefail
        tag=$({{.GO_TOOL_CHANGIE}} changie latest)
        case "${tag}" in v[0-9]*.[0-9]*.[0-9]*) ;; *) echo "::error::release:tag: changie latest = '${tag}'"; exit 1 ;; esac
        [ -s ".changes/${tag}.md" ] || { echo "::error::release:tag: .changes/${tag}.md is missing or empty"; exit 1; }
        rc=0
        git ls-remote --exit-code --tags origin "refs/tags/${tag}" >/dev/null 2>&1 || rc=$?
        case "${rc}" in
          0) echo "release:tag: ${tag} already exists on origin — nothing to do"; exit 0 ;;
          2) ;;
          *) echo "::error::release:tag: git ls-remote failed (${rc}); refusing to guess"; exit 1 ;;
        esac
        git tag "${tag}"
        git push origin "refs/tags/${tag}"
```
`git ls-remote --exit-code` returns 2 when nothing matches and other non-zero for transport errors; treating every non-zero as "absent" would tag on a network blip, hence the `case`. Existing tags in the repo are lightweight (`git cat-file -t v0.14.1` printed `commit`), and `post-release-verify` resolves the peeled commit for either kind, so `git tag` without `-a` matches history. [VERIFIED: ran both] The `{{.GO_TOOL_CHANGIE}}` form reuses `GO_TOOL_CHANGIE: GOWORK=off go tool -modfile=go.tool-changie.mod` (`Taskfile.yml:13`, pinned by `TestChangieWrapperTaskRecordsInstallPath`).
Taskfile rules that bind new targets: no `status:` or `platforms:` keys (`TestTaskfileGatesFailLoud`), and any `task <x>` in `CONTRIBUTING.md` must name a real target (`TestContributingReferencesRealTaskTargets`, regex ``` `task\s+([A-Za-z0-9:_-]+)` ```).

### REL-12 edit in `release:goreleaser` (Taskfile.yml, near line 2910)

The target already derives the tag with `TAG="$(git describe --exact-match --tags HEAD)"` inside the `GITHUB_OUTPUT` block (`Taskfile.yml:2915`). Hoist that derivation above the `goreleaser release` call, assert the file, and pass it:
```sh
TAG="$(git describe --exact-match --tags HEAD)"
NOTES=".changes/${TAG}.md"
[ -s "${NOTES}" ] || { echo "::error::${NOTES} is missing or empty"; exit 1; }
"${GORELEASER_BIN}" release --clean --release-notes "${NOTES}"
```
`release.yml` does not change for this (its step stays `run: task release:goreleaser`). `git describe --exact-match` is already a precondition of the target.

### REL-13 rehearsal (`release:dry-run`)

Replace `--snapshot` with the documented pre-tag form and keep the four-binary assertions:
```sh
TAG="$({{.GO_TOOL_CHANGIE}} changie latest)"; NOTES=".changes/${TAG}.md"
[ -s "${NOTES}" ] || { echo "::error::${NOTES} is missing or empty"; exit 1; }
echo "RELEASE-NOTES-EVIDENCE tag=${TAG} file=${NOTES} bytes=$(wc -c < "${NOTES}" | tr -d ' ')"
GORELEASER_CURRENT_TAG="${TAG}" "${GORELEASER_BIN}" release --skip=publish,sign,validate --clean --release-notes "${NOTES}"
```
`artifacts.json` names become `codegraph_<tag>_...` instead of snapshot names; the existing check selects `extra.Format=="binary"` entries and counts four, which a non-snapshot run still produces (I saw `codegraph_v0.14.1_{darwin,linux}_{amd64,arm64}` archived). `release:dry-run-signed` also states "Uses --snapshot" in prose elsewhere; keep it unchanged except for the DOCS-14 comment. No test pins `--snapshot` today. [VERIFIED: `rg snapshot internal/upgrade/*_test.go` returned nothing] The rehearsal can run against an existing file (`v0.14.1`) now; a scratch batch of the real fragment (`v0.15.0`) shows the real next body and is the better evidence for "the body Phase 6 will publish" (the scratch batch output was 16 files in a throwaway tree; do not batch in the real tree).

### GRD-15 recipes (guard inventory, line numbers current at HEAD `1c0ac01e`)

| Guard | Today | Re-point |
|---|---|---|
| `tag_authority_test.go` `TestReleasePleaseStaysPreMajor` (`:90`, const `releasePleaseConfigPath` `:20`) | reads `release-please-config.json` `bump-minor-pre-major` | rename (census), read `.changie.yaml`, find the `kinds` entry with `label: Breaking`, assert `auto: minor`, fail loudly if no `Breaking` kind is found (positive assertion). RED mutation: set it to `major`. Note `TestChangieConfigShape` already pins the same value and the comment `Flip to major at 1.0`; the new test is the named D-06R-style guard, not the only one |
| `tag_authority_test.go` prose (`:9`, `:40`, `:61`, `:70`, `:82`) | "release-please is the SOLE tag authority" | reword to the changie release workflow's App-token step; `TestGsdTagCreationIsDisabled` logic and assertions unchanged (only strings change) |
| `taskfile_shape_test.go:149` | `{Workflow: "release-please.yml", JobID: "pretag-gate"}` | two entries: `release-pr.yml/pretag-gate` and `release-tag.yml/pretag-gate`, plus the `release-pr` and `tag` jobs (they contain `run:` steps, so they belong in `inScopeJobs`, not the uses-only list) |
| `taskfile_shape_test.go:1441-1445` | uses-only exception for job `release-please` | delete the entry; `validateUsesOnlyJobExceptions` fails a stale one |
| `taskfile_shape_test.go:1462` | `inScopeWorkflowFiles = []string{"ci.yml", "release-please.yml", ...}` | swap in `release-pr.yml`, `release-tag.yml` (`TestWorkflowFilePopulationMatchesDisk` fails a file on disk named in neither list) |
| `taskfile_shape_test.go:2420`, `release_workflow_shape_test.go:889` | SAN negative fixtures build a SAN ending `.github/workflows/release-please.yml@refs/tags/v1.2.3` | use `release-tag.yml` |
| `goreleaser_shape_test.go:984-1009` `TestReleaseBlockDoesNotRewriteReleaseBody` | message "release-please owns Release authorship (D-06R)" | same forbidden keys (`name_template`, `header`, `footer`, `draft`, `disable`), restated: GoReleaser now authors the body from `.changes/${TAG}.md`, and a header/footer/name/draft/disable key would make the body differ from the file or unpublish it. Prose-only, so no behavioural RED exists for the restatement; record that explicitly |
| NEW `changelog.disable` guard (no such test exists today) | `rg -i changelog internal/upgrade/goreleaser_shape_test.go` found nothing | decode `.goreleaser.yaml`; assert there is no `changelog` block, or its `disable` is not boolean true and not the string `"true"`; positive assertion that the config parsed and the `release` block was found. RED mutation: add `changelog: {disable: true}` |
| `.goreleaser.yaml:404-410` comment | "release-please creates the tag AND the GitHub Release ... GoReleaser's job is only to add assets" | rewrite: the changie tag arrives first, GoReleaser creates the Release and authors its body from the file |
| `scripts/pr_template_policy.py:54-60` | `# release-please machinery...` plus `".release-please-manifest.json",` and `"release-please-config.json",` in `EXEMPT_EXACT` | remove both names and the comment; add `".changes/"` to `EXEMPT_PREFIXES` (a tuple used with `str.startswith`) |
| `require-issue-link.yml:86-93` | `case` arms `go.tool.mod|...) ;;` and `.release-please-manifest.json|release-please-config.json) ;;` | delete the second arm and the comment; add a `.changes/*) ;;` arm (the first arm's `.planning/*|docs/*|.github/*` is the model) |
| NEW parity check | the two lists are kept "in sync" by a comment only | assert both directions in Python: `.changes/` in `EXEMPT_PREFIXES`; neither release-please filename in `EXEMPT_EXACT`; `require-issue-link.yml` text has a `.changes/*)` arm and no release-please arm. Include a test that a PR touching only `.changes/vX.md`, `.changes/unreleased/x.yaml` and `CHANGELOG.md` passes the policy, with `changed` from `CHANGED_FILES` |
| `changie_shape_test.go:113` comment, `.changie.yaml:20` comment | "release-please-authored" | reword |
| `Taskfile.yml:5253` (`check:cross` desc) | "release-please.yml's pretag-gate job" | name `release-pr.yml` and `release-tag.yml` |
| `CONTRIBUTING.md:90` | same | same |
| NEW GRD-16 guard | none | Pitfall 6 |

Hygiene allowlist semantic to know: a `.changes/` prefix also exempts a contributor PR that touches only `.changes/unreleased/*.yaml` (a fragment-only PR). That is housekeeping by any reading and is consistent with D-05.

### Census instrument (REL-14)

Names and contents, tracked and untracked-not-ignored, with the three exclusions (`.planning/`, `CHANGELOG.md`, `.changes/v*.md`):
```sh
pat='release[-_ ]?please'
git ls-files -co --exclude-standard \
  | grep -vE '^(\.planning/|CHANGELOG\.md$|\.changes/v[0-9][^/]*\.md$)' > "${list}"
grep -iE "${pat}" "${list}"                              # filenames
xargs -d '\n' grep -iEl "${pat}" < "${list}"             # contents (use git grep -iEl where xargs -d is unavailable on macOS)
```
Baseline at HEAD (`git grep -il -E 'release[-_ ]?please'` with those exclusions): `.changie.yaml`, `.github/workflows/{ci,post-release-verify,pr-title,release-please,release,require-issue-link}.yml`, `.goreleaser.yaml`, `CONTRIBUTING.md`, `Taskfile.yml`, `docs/RELEASE-PROCEDURES.md`, `internal/upgrade/{changie_shape,goreleaser_shape,release_workflow_shape,tag_authority,taskfile_shape}_test.go`, `release-please-config.json`, `scripts/pr_template_policy.py` (18 files). The manifest `.release-please-manifest.json` holds no matching text, so only a filename check finds it, hence the two-part instrument. In scope history left alone: `CHANGELOG.md` (2 hits) and `.changes/v0.2.0.md` (2 hits). [VERIFIED: ran it]
Positive control (rule `84d1gfpywd`): the instrument must be run first against a scratch tree with a planted reference and a planted filename and be shown to report both, then against the real tree. Prefer `git grep` for portability on macOS (no `xargs -d`). The instrument can be a `check:` Taskfile target with a `--self-test`-style first phase, mirroring `scripts/check-workflow-output-delimiter.sh --self-test`; the transcripts go in `05-CENSUS.md`.

### WR-01 fix sketch (both files, lockstep)

```python
EXEMPT_MARKER = re.compile(r"<!--\s*pr-template-exempt:([^>]*)-->", re.I)
# main(): reason = m.group(1).strip(); exempt only if non-empty
```
and the same with `changelog-exempt` in `fragment_policy.py`, `find_exemption` unchanged in shape (`match.group(1).strip() or None`). Update `test_marker_grammar_mirrors_pr_template_exempt` if the group name changes (it compares `.pattern` text after a single `replace`, so keep the two patterns identical modulo the marker name). [VERIFIED: current pattern text `scripts/pr_template_policy.py:83` and `scripts/fragment_policy.py:49`]

## State of the Art

| Old approach | Current approach | When changed | Impact |
|---|---|---|---|
| release-please opens the release PR, creates tag AND Release together | changie release PR; App pushes the tag; GoReleaser creates the Release | this phase | `release: published` now fires after assets exist (GoReleaser creates a draft, uploads, then publishes), so `post-release-verify`'s `workflow_run` trigger stays correct but its header rationale about "release-please creates the Release at tag time" is obsolete and must be reworded (census) |
| Version from PR titles | Version from fragments | milestone v0.15.0 | `pr-title.yml` becomes hygiene only (DOCS-14) |
| GoReleaser adds assets to an existing Release | GoReleaser creates the Release (`replace_existing_artifacts: true`, `prerelease: auto`, default name `{{.Tag}}`) | this phase | Release name stays `v0.15.0`-shaped (the prior name was `v0.14.1`) [VERIFIED: `gh release view v0.14.1 --json name`]; Release author changes from the App bot to `github-actions[bot]` |
| `create-github-app-token` `app-id` | `client-id` is the current input; `app-id` is the deprecated alias | v3 | Keep `app-id` (matches the documented secret names `APP_ID`, `APP_PRIVATE_KEY`); do not mix in a secret rename this phase |

**Deprecated/outdated in the repo as of this phase:** `release-please.yml`, `release-please-config.json`, `.release-please-manifest.json`; `docs/RELEASE-PROCEDURES.md` §4 title "now automated by release-please"; the rc hand-tag escape hatch (Pitfall 4).

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `git add` with a directory pathspec (`add-paths: .changes`) stages deletions of the removed fragments, so `add-paths` could be used safely | Code Examples (not used in the recommended shape) | Low: the recommended shape omits `add-paths` (default adds all changes, including deletions) |
| A2 | A push to `main` produced by squash-merging the release PR is a single commit, so `paths: [CHANGELOG.md]` sees `CHANGELOG.md` in the diff | Pattern 1 | The `protect-main` ruleset enforces squash-only plus linear history [VERIFIED], and the last release (#98) merged as one commit; Phase 6 confirms live |
| A3 | GitHub stores the Release body exactly as GoReleaser sends it (file plus one trailing `\n`), so SHIP-02's byte comparison needs trailing-whitespace tolerance | Pitfall 2, Open Question 3 | A strict byte comparison could fail on a trailing newline difference; settled by Phase 6's live Release |
| A4 | The 3,000-file push-diff limit applies to the milestone merge such that the tag workflow may or may not run there | Pattern 1 | None: both outcomes are a no-op because of the skip-if-exists branch |
| A5 | The planner/maintainer accepts amending the Phase 1 D-02 newlines pin (Pitfall 1) | Summary | If refused, the first release PR cannot pass its required `test` check; there is no alternative short of a different tool config |

## Open Questions (RESOLVED)

1. **Amend the Phase 1 `newlines` lock (Pitfall 1).** RESOLVED by CONTEXT D-13 (amend, as recommended).
   - What we know: the current config yields a corrupted CHANGELOG and a red `TestChangieVersionSeedsMatchChangelog` on the first release PR; the fix and its blast radius (one pinned test) are measured.
   - What's unclear: whether the maintainer wants it recorded as a Phase 5 decision (D-11) or a Phase 1 errata.
   - Recommendation: do it in this phase, RED-first, first plan; name it in the plan as amending Phase 1 D-02 under D-01's "forced by a locked constraint" clause (CHG-02 verbatim history).

2. **"Exactly one change" versus the census (REL-12 vs REL-14).** RESOLVED by CONTEXT D-12 (flag in Taskfile `release:goreleaser`).
   - What we know: the real one-line change is in `Taskfile.yml`'s `release:goreleaser`; `release.yml` has one census hit, a comment at line 188 ("pinned to the exact same Action + SHA release-please.yml already carries").
   - What's unclear: whether "byte-identical" tolerates a comment hunk.
   - Recommendation: read REL-12 as "no functional line of `release.yml` changes"; capture `git diff <post-#99 main> -- .github/workflows/release.yml` in the SUMMARY showing exactly one comment hunk, and the Taskfile diff showing the flag. The existing release guards (`TestReleaseWorkflowTriggerIsTagPushOnly`, `TestOIDCWriteScopedToSingleGoreleaserJob`, `TestReleaseJobSetupZigDisablesCache`, `TestAppleSecretsScopedToSingleReleaseJob`) already hold the locked parts; a hash-pin of the file would be a new, fragile guard, so do not add one.

3. **What "equals the file" means for REL-13 and SHIP-02.** RESOLVED by CONTEXT D-14 (REL-13 tests only this repo's wiring; no body-equality relation).
   - What we know: GoReleaser's body is the file plus one newline; the file will end `\n\n` after the newlines fix; the body is never written to disk in a dry run.
   - Recommendation (withdrawn by D-14, 2026-10-14: REL-13 tests only this repository's wiring, so Phase 5 defines no body-equality relation and asserts none): REL-13's captured evidence is the notes-path assertion plus the evidence line, not a GoReleaser-rendered body.

4. **The rc hand-tag escape hatch (Pitfall 4).** RESOLVED by CONTEXT D-15 (removed).
   - Recommendation: remove it from the runbook with a recorded verdict (hand `git tag` contradicts the milestone's "no human git tag" rule, and the new `release:goreleaser` rightly refuses a tag with no `.changes` file), noting that a prerelease can be batched with `changie batch auto -p rc.1` but is not wired into `release:batch`. Maintainer to confirm.

5. **Phase 5's own fragment versus the census.** RESOLVED by CONTEXT D-11 (`.changes/unreleased/` is this milestone's own notes).
   - What we know: the phase-close capability writes a fragment for this phase's user-visible change; natural wording mentions release-please; `.changes/unreleased/*.yaml` is not covered by D-09's `.changes/v*.md` exclusion; it becomes `CHANGELOG` history at release.
   - Recommendation: either word the fragment without the literal ("the previous release automation") or extend the census exclusion to `.changes/unreleased/` and record that. The second keeps the capability unconstrained; ask the maintainer which.

6. **Layout: two files or one.** RESOLVED under CONTEXT Claude's Discretion (two files). Claude's discretion per CONTEXT; this research recommends two (documented `paths:` filter, no custom path detection). The cost is a duplicated `pretag-gate` job and two more registry entries.

## Environment Availability

| Dependency | Required by | Available | Version | Fallback |
|------------|-------------|-----------|---------|----------|
| Go | all tests | yes, but default is 1.27.2 | `GOTOOLCHAIN=go1.26.9` gives 1.26.9 | never use 1.27 (breaks `cockroachdb/swiss`) |
| task | targets, install-task | yes | 3.52.0 (global); CI builds the pin from `go.tool.mod` | — |
| changie | batch/merge/latest | yes | v1.26.0 global, same as the `go.tool-changie.mod` pin | `task changie --` |
| GoReleaser | dry-run | built from `go.tool.mod` (v2.17.1) | built OK in scratch | — |
| zig, syft, cosign | `release:dry-run` | yes | zig present; syft 1.54.1; cosign present | preconditions already name install commands |
| actionlint | workflow lint | yes | v1.7.12 (`task lint:actions` exit 0 at HEAD) | — |
| python3 | policy scripts | yes | 3.13.5 (`python3 -I` for probes) | — |
| gh | verification of settings | yes | 2.102.0 | — |
| jq | dry-run assertions | yes | 1.8.2 | — |
| Secrets `APP_ID`, `APP_PRIVATE_KEY` | both new workflows | present | set 2026-07-29/30 | — |

**Missing dependencies with no fallback:** none. **Local trap:** five defined-but-empty `MACOS_*` variables in this shell break any real GoReleaser run (Pitfall 8).

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go `testing` (`internal/upgrade`, go 1.26.9) + Python `unittest` (`scripts/test_fragment_policy.py`) + Taskfile gate targets (`check:changie`, `check:fragment-policy`) + actionlint |
| Config file | none beyond `Taskfile.yml`, `.golangci.yml`, `.github/workflows/ci.yml` (`test` job runs `task check:changie`) |
| Quick run command | `GOTOOLCHAIN=go1.26.9 go test ./internal/upgrade -count=1 -run '<TestNames>'` (full package: 0.7 s at HEAD) |
| Full suite command | `GOTOOLCHAIN=go1.26.9 go test ./internal/upgrade -count=1 && task check:fragment-policy && task check:changie && task lint:actions` then `task test` at the phase gate |

### Phase Requirements to Test Map

| Req ID | Behavior | Test type | Automated command | File exists? |
|--------|----------|-----------|-------------------|--------------|
| REL-10 | `release-pr.yml`: only `push` to `main`; `release-pr` job `needs: pretag-gate`; PR step SHA-pinned with version comment, `token` = the App-token step output, `draft` not true, title prefix `chore(main): release`, `body-path` `.changes/...`; `release:batch` body orders guard, `batch auto`, `merge`, `latest` and has the skip-ci check; `pretag-gate` run body is `task check:cross` | Go shape | `go test ./internal/upgrade -run 'TestReleasePR'` | Wave 0: new `changie_release_shape_test.go` |
| REL-10 | no fragments: `release:batch` exits 0 and writes no `version` output; with the real fragment it writes `version=v0.15.0` and a well-formed file | `check:` Taskfile gate in a scratch copy (extend `check:changie`, e.g. legs 14-16) | `task check:changie` | Wave 0: new legs |
| REL-10 | rendered release file/CHANGELOG are well-formed (Pitfall 1) | `check:changie` leg + `TestChangieConfigShape` updated | `task check:changie`; `go test -run TestChangieConfigShape` | Wave 0 |
| REL-11 | `release-tag.yml`: `push` to `main` with `paths: [CHANGELOG.md]` exactly; `tag` job `needs: pretag-gate`; App-token mint precedes checkout; checkout `with.token` is the mint output; no `github.token`/`secrets.GITHUB_TOKEN` in that job; run body `task release:tag` | Go shape | `go test -run 'TestReleaseTag'` | Wave 0 |
| REL-11 | `release:tag` tags absent tag and skips existing tag (against a local bare remote), refuses malformed/absent notes, treats `ls-remote` transport errors as failure | Go test that runs the target body (or `task release:tag`) in a temp git repo with a bare `origin`; positive assertion counts pushed refs | `go test -run 'TestReleaseTagTarget'` | Wave 0 |
| REL-12 | `release:goreleaser` passes `--release-notes` with `.changes/${TAG}.md` after asserting the file; `release.yml` run body still the single `task release:goreleaser`; `.goreleaser.yaml` has no `changelog.disable: true` | Go shape (NEW changelog guard) | `go test -run 'TestGoreleaserChangelog|TestReleaseGoreleaserTarget|TestReleaseWorkflow'` | Wave 0 for changelog guard |
| REL-12 | `release.yml` diff vs post-#99 main is one comment hunk | one-time evidence | `git diff 7b12cb92 -- .github/workflows/release.yml` in SUMMARY | n/a |
| REL-13 | `release:dry-run` is non-snapshot, passes the same flag, asserts non-empty file, prints evidence line; captured run for an existing and a scratch-batched file | Go shape + captured `task release:dry-run` transcript | `go test -run 'TestReleaseDryRun'`; `task release:dry-run` (darwin) | Wave 0 |
| REL-14 | census finds planted ref and planted filename in scratch tree; reports zero on the real tree after deletion | Taskfile gate with self-test phase | `task check:release-please-census` (name is the planner's) | Wave 0 |
| GRD-15 | re-pointed `Breaking → minor` test; registry/fixtures; hygiene allowlists swapped; WR-01 | Go + Python | `go test -run 'TestChangieBreaking|TestInScopeJobs|TestWorkflowFilePopulation|TestWorkflowRunBodies'`; `task check:fragment-policy` | tests exist; new ones Wave 0 |
| GRD-16 | `TestGsdTagCreationIsDisabled` unchanged and green; new "only tag push" guard RED against a planted second tag push (workflow run body, and a Taskfile target invoked from a workflow) | Go | `go test -run 'TestGsdTagCreationIsDisabled|TestTagPushIsSingle'` | Wave 0 |
| DOCS-12 | `CONTRIBUTING.md` §Pull requests names fragment, `.changes/unreleased`, `changelog-exempt`, never hand-write `.changes/v*.md`, how the version is derived; no `Release-As`; `task <x>` mentions real | grep-based plan verification + `TestContributingReferencesRealTaskTargets` | `go test -run TestContributingReferencesRealTaskTargets`; `rg` checks in the plan | exists |
| DOCS-13 | RELEASE-PROCEDURES describes fragment → release PR → merge → App-token tag → locked `release.yml`; `pretag-gate` placed in the new workflows; "release is cut by merging the release PR"; 45 verdict rows | census + `05-CENSUS.md` row count equals 45 | census target; `rg -c` over the verdict table | Wave 0 (artifact) |
| DOCS-14 | `pr-title.yml` header says hygiene only; `release:dry-run-signed` desc and `docs/RELEASE.md` pipeline paragraph name `release-pr.yml`/`release-tag.yml` | targeted `rg` verification in plan; census | `rg` commands in plan | n/a |

Every guard above is shown RED against a confirmed-applied, byte-cleanly-reverted mutation, with a positive assertion that it counted a non-zero population (rule `84d1gfpywd`). Suggested mutations: set `Breaking` to `major`; replace the checkout token with `${{ github.token }}`; plant `git push origin --tags` in a workflow run body and a second `git tag` in a Taskfile target a workflow calls; add `changelog: {disable: true}`; remove `--release-notes`; drop `paths:` from the tag workflow; plant `release-please` in a scratch file for the census; re-run the old WR-01 regex against a 60k-character body.

### Sampling Rate

- **Per task commit:** `GOTOOLCHAIN=go1.26.9 go test ./internal/upgrade -count=1 -run '<the plan's tests>'` plus `task lint:actions` when a workflow file changed.
- **Per wave merge:** full `internal/upgrade` package, `task check:fragment-policy`, `task check:changie`, `task lint:actions`.
- **Phase gate:** the full suite plus `task test`; a scratch-tree rehearsal (copy the tree, run `release:batch`, then the `Changie` Go tests and `task check:changie` on it); `task release:dry-run` transcript; census clean. Live proof is Phase 6.

### Wave 0 Gaps

- [ ] `internal/upgrade/changie_release_shape_test.go` — workflow and Taskfile shape for REL-10/11, GRD-16, release:goreleaser/dry-run.
- [ ] A `changelog.disable` guard in `goreleaser_shape_test.go` (REL-12 says "existing shape tests" assert it; none does).
- [ ] `check:changie` legs for the release-file shape, empty-set no-op and `release:batch` output; raise the leg-count constants (`[n/13]` strings and "13 of 13") consistently.
- [ ] `scripts/test_fragment_policy.py` additions (exemption swap, parity, WR-01 timing/grammar) and the `floor=36` bump in `check:fragment-policy`.
- [ ] Census target with self-test, and `05-CENSUS.md`.
- [ ] No new framework install is needed.

## Security Domain

`security_enforcement` is on in `.planning/config.json` (ASVS level 1, block on high). [VERIFIED: file read]

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|------------------|
| V2 Authentication | yes (machine identity) | GitHub App installation token minted per job from `APP_ID`/`APP_PRIVATE_KEY`; never `GITHUB_TOKEN` for tag or PR; mint inside the consuming job (masked outputs are dropped between jobs) |
| V3 Session Management | no | — |
| V4 Access Control | yes | Top-level `permissions: contents: read` on both new workflows; writes happen only through the App token; `protect-main` stays the merge gate; optional `permission-*` narrowing on the token step |
| V5 Input Validation | yes | `release:batch`/`release:tag` validate `changie latest` against `v[0-9]*.[0-9]*.[0-9]*` before writing it to `GITHUB_OUTPUT` or using it in a git ref; fragment text is untrusted-ish content that reaches the PR body and squash commit message (C2) |
| V6 Cryptography | no new use | cosign/attest chain unchanged and byte-locked |
| V14 Configuration / supply chain | yes | Every action full-SHA pinned with version comment; `create-pull-request` receives the token and is a node action, so the pin is the control; no `pull_request_target` in the new workflows; no write permission on `GITHUB_TOKEN` |

### Known Threat Patterns

| Pattern | STRIDE | Standard mitigation |
|---------|--------|---------------------|
| Tag pushed with `GITHUB_TOKEN` silently fails to release (or a second, unreviewed tag path appears) | Tampering / Repudiation | GRD-16 single-tag-push guard resolving through Taskfile; checkout-token assertion |
| `[skip ci]` smuggled through a fragment body into the squash commit message | Tampering | C2 check in `release:batch` |
| Script injection via PR/fragment text into `run:` | Tampering | Step bodies are `task <target>` with no `${{ }}` interpolation inside shell; the version flows through `GITHUB_OUTPUT` into `with:` inputs, not into `run:` |
| Deleted tag resurrected by a state-based job | Tampering | Path-filtered trigger (D-06), skip-if-exists only guards re-runs |
| Over-broad App token for a third-party node action | Elevation | SHA pin; optional `permission-contents`/`permission-pull-requests` narrowing; token minted only in the two jobs that need it |
| Adversarial PR body stalls the hygiene gates (WR-01) | Denial of service | Linear-time marker pattern plus a timing test |

## Sources

### Primary (HIGH confidence)
- changie docs `docs/integrations/release_trigger.md`, `docs/integrations/goreleaser.md` and `.github/workflows/release.yml` (raw GitHub, fetched this session) - trigger, GoReleaser flags, changie's own release job.
- `miniscruff/changie-action` `.github/workflows/gen-release-pr.yml` and `README.md` (raw GitHub) - batch auto, merge, latest, create-pull-request.
- `peter-evans/create-pull-request` v8.1.1 `README.md`, `action.yml`, `docs/concepts-guidelines.md` (raw GitHub at the tag) - inputs, behaviour, token guidance.
- `actions/create-github-app-token` `action.yml` at SHA `bcd2ba49` - inputs including `permission-*`.
- GitHub docs, workflow syntax (`on.push` branches+paths semantics, 3,000-file/1,000-commit limits) and "Skipping workflow runs" (skip phrases, head-commit rule).
- GoReleaser v2.17.1 source in the module cache: `internal/pipe/changelog/changelog.go`, `internal/pipe/release/body.go`, `release.go`, `internal/client/github.go`, `cmd/release.go`.
- Repository files read this session: `release-please.yml`, `release.yml`, `post-release-verify.yml`, `Taskfile.yml` (targets named above), `.changie.yaml`, `.goreleaser.yaml`, `internal/upgrade/*_shape_test.go`, `tag_authority_test.go`, `scripts/*.py`, `require-issue-link.yml`, `pr-title.yml`, `CONTRIBUTING.md`, `docs/RELEASE-PROCEDURES.md`, `docs/RELEASE.md`, REQUIREMENTS/ROADMAP/CONTEXT.
- GitHub API reads: releases and tag refs, repo settings, ruleset 20157557, secrets list (names only), PR #98 check rollup.
- Experiments run in scratch directories: changie batch/merge/latest/next, newlines variants, post-batch guard run, GoReleaser snapshot vs non-snapshot with a present, missing and empty notes file, non-snapshot run on the real config, WR-01 regex timing.

### Secondary (MEDIUM confidence)
- Community reports of a 300-file diff limit for PR path filters (not used for any decision here; the official push limits above were used instead).

### Tertiary (LOW confidence)
- none relied upon.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - every pin read from the repo or the GitHub API; documented pattern fetched from source.
- Architecture: HIGH for the shapes (they follow the docs and the repo's own guards); MEDIUM for live behaviour that only Phase 6 can show (merge-commit push triggering the path filter, Release body trailing newline).
- Pitfalls: HIGH for 1, 2, 3, 6, 7, 8 (reproduced); MEDIUM for 4 and 9 (policy choices).

**Research date:** 2026-10-14
**Valid until:** 2026-11-13 (30 days; the pins are stable, `create-github-app-token` v3.3.0 will be two weeks old by 2026-10-26 and may be adopted then if wanted)
