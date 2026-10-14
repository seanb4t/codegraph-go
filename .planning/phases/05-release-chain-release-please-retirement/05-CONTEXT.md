# Phase 5: Release Chain & release-please Retirement - Context

**Gathered:** 2026-10-13
**Status:** Ready for planning

<domain>
## Phase Boundary

release-please is replaced end to end by the **documented changie release pattern**: a workflow batches unreleased fragments into one reviewable `chore(main): release vX.Y.Z` PR, and when that PR merges the GitHub App pushes tag `vX.Y.Z` into the unchanged, locked `release.yml`, which passes `--release-notes .changes/${TAG}.md` to GoReleaser. Every guard that pinned release-please is re-pointed RED-first, the contributor and release docs describe the chain, and release-please's three files are deleted as the phase's last change (REL-10…14, GRD-15, GRD-16, DOCS-12…14).

In scope:
- The new release workflow (release-PR job + tag job, both behind `pretag-gate`).
- REL-12's single `release.yml` edit, REL-13's dry-run rehearsal.
- GRD-15/GRD-16 guard re-pointing, including the hygiene allowlist swap in `scripts/pr_template_policy.py` **and** `.github/workflows/require-issue-link.yml` (kept in sync, as today), and the carried-forward WR-01 regex fix.
- DOCS-12…14 prose, the REL-14 census, and deleting `release-please.yml`, `release-please-config.json`, `.release-please-manifest.json`.

Out of scope:
- Any live run of the chain — that is Phase 6, entered by the milestone PR's merge (GitHub runs the workflow file from the pushed commit, so that merge runs the new workflow, not release-please).
- `release.yml`'s trigger, cosign identity, and build/sign/attest steps (byte-locked).
- Porting to other repos (SEED-005 / PORT-01, v2).

</domain>

<decisions>
## Implementation Decisions

### Governing principle
- **D-01:** **Use the documented pattern; do not design one.** The maintainer, explicitly: "isn't there an action for this and a general approach? Why are we making a space shuttle?" The chain follows changie's own docs (its `gen-release-pr.yml`, "Release Trigger" and GoReleaser integration pages) and `peter-evans/create-pull-request`'s documented token guidance. Where the docs already answer a question, the planner takes the documented answer; custom logic is added only where a locked constraint of this repo forces it, and each such addition must name the constraint.

### Release PR (REL-10)
- **D-02:** The release-PR job runs `changie batch auto` → `changie merge` → `changie latest` (through the existing `task changie` pin, Phase 1 D-04 — CHG-01's one recorded CI install path; no second install via `changie-action`) and opens or updates **one** PR with **`peter-evans/create-pull-request`** (SHA-pinned, version comment) on a fixed branch, regenerated from `main` on every run, title `chore(main): release vX.Y.Z`. With no `*.yaml` fragments in `.changes/unreleased/` it does nothing (`changie batch auto` fails on an empty set — CHG-03 — so the job checks first and no-ops).
- **D-03:** `create-pull-request` is given the **App installation token** (`actions/create-github-app-token`, existing `APP_ID`/`APP_PRIVATE_KEY`). This is the action's documented answer to the second token hazard: a PR opened or updated with `GITHUB_TOKEN` fires no `pull_request` workflows, so `protect-main`'s 9 required checks (incl. `test`, `tmux e2e`, `reproducibility`, `perf regression gate` — `ci.yml` runs on plain `pull_request`) would never report and the PR could never merge.
- **D-04:** The PR body is the rendered `.changes/vX.Y.Z.md`. It is opened **ready for review, never draft** (the App's `author_association` is not OWNER/MEMBER/COLLABORATOR, so `close-draft-prs.yml` would close a draft).
- **D-05:** The hygiene gates recognize the release PR **by path**: in both `pr_template_policy.py`'s exemption set and `require-issue-link.yml`'s `case` list, the two release-please filenames are replaced by a `.changes/` prefix. A PR touching only `.changes/**` plus `CHANGELOG.md` is housekeeping. This is GRD-15's "gains the release PR's own" read literally. `fragment-required` already passes it (no `cmd/**`/`internal/**`).
- **D-13:** (Maintainer, at plan time 2026-10-14.) **Amend Phase 1 D-02's `newlines` lock.** `.changie.yaml` gains `newlines: {beforeKind: 1, afterKind: 1, endOfVersion: 2}`. With only `afterChangelogHeader: 1`, a freshly batched `.changes/vX.Y.Z.md` has no trailing newline, so `changie merge` glues the next `## [x.y.z]` heading onto the last bullet. `TestChangieVersionSeedsMatchChangelog` then fails on the post-batch tree, the release PR's required `test` check goes red, and the PR cannot merge (RESEARCH Pitfall 1, reproduced). With the amendment the seeds stay byte-identical (`changie merge --dry-run`) and `check:changie` passes 13/13. `TestChangieConfigShape`'s "only newlines key" pin changes on purpose, RED-first. This uses changie's own config keys, not custom logic (D-01); a Taskfile post-batch newline patch was considered and rejected.

### Tag (REL-11)
- **D-06:** Trigger is changie's documented **release trigger: a push to `main` that changes `CHANGELOG.md`** — only a release-PR merge (`changie merge`) changes it. The tag job tags `$(changie latest)` at the pushed commit with the App token, and **skips (success, not failure) when that tag already exists**. That skip is what makes the milestone merge — which also changes `CHANGELOG.md` and brings all 15 seed files (`v0.2.0`…`v0.14.1`, already tagged) — a no-op. A failed tag job is re-run like any other job.
  - Supersedes the two earlier answers in this session ("state-based on every push" and "tag the commit that added the file"): with the path trigger, the pushed commit *is* the release merge, so neither extra mechanism is needed.
  - Do **not** add a "tag must point at the commit that added `.changes/vX.Y.Z.md`" assertion: the historical tags point at release-please commits, not at the milestone merge that adds their seed files.
  - **REL-11's parenthetical** ("detected by a new `.changes/vX.Y.Z.md` on `main`") and ROADMAP Phase 5 success criterion 1's matching clause are **amended** to the documented trigger. The planner updates the values in `REQUIREMENTS.md`/`ROADMAP.md`'s existing shapes (no new structure — the planning-artifacts rule, as Phase 4 D-09 did).
- **D-07:** **`pretag-gate` (`task check:cross`, moved verbatim) gates both jobs** — the release-PR job and the tag job each `needs:` it. Its original purpose is "never cut a tag from a `main` state that fails cross-platform resolution", and the tag path is where that applies.

### GoReleaser and rehearsal (REL-12, REL-13)
- **D-08:** `release.yml` changes by exactly one thing: `goreleaser release` receives `--release-notes .changes/${TAG}.md` (changie's documented GoReleaser integration). `.goreleaser.yaml` never sets `changelog.disable: true`, asserted by the existing shape tests. (Placement refined by D-12.)
- **D-12:** (Maintainer, at plan time 2026-10-14.) **The `--release-notes .changes/${TAG}.md` flag lives in `Taskfile.yml`'s `release:goreleaser`**, the target `release.yml`'s only run body invokes. The carried-forward rule that every CI `run:` body is a `task` invocation (`TestWorkflowRunBodiesInvokeTask`) cannot coexist with a literal flag in `release.yml`, and the rule wins. `release.yml`'s functional lines stay byte-identical to post-#99 `main` (`7b12cb92`). Its only diff is the census comment hunk near line 188 (REL-14). REL-12 and ROADMAP Phase 5 success criterion 2 are amended to this placement in their existing shapes, as D-06 amended REL-11/SC1. No hash-pin guard is added on `release.yml`; the existing release guards hold its locked parts.

### Census and docs (REL-14, DOCS-12…14)
- **D-09:** REL-14 is a plain grep census, positive-controlled (first shown to find a planted reference), with exactly the requirement's exclusions — `.planning/`, `CHANGELOG.md` history, this milestone's own notes — where **`.changes/v*.md` seed files count as `CHANGELOG.md` history** (tool-owned, byte-sliced from it; `.changes/v0.2.0.md` mentions release-please and cannot be edited). Everything else is reworded, including test names (`TestReleasePleaseStaysPreMajor` is renamed when re-pointed at `.changie.yaml`'s `Breaking → minor`), code comments, D-06R prose in `tag_authority_test.go`, and the `.changie.yaml` comment at line 20.
- **D-10:** Each of `docs/RELEASE-PROCEDURES.md`'s 45 release-please references gets a recorded verdict in a phase artifact (not inline in the doc).
- **D-11:** (Maintainer, at plan time 2026-10-14.) **`.changes/unreleased/` is "this milestone's own notes"** for D-09 and REL-14, so the census excludes it. Between releases it holds exactly the fragments that `changie batch` turns into v0.15.0's notes, which then become `.changes/v0.15.0.md` (already counted as history). Phase 5's own closing fragment may therefore name release-please. No other `.changes/` path is excluded: `.changes/header.tpl.md` stays in scope. The positive control proves that boundary in both directions.

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

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements and roadmap
- `.planning/REQUIREMENTS.md` — REL-10…14, GRD-15, GRD-16, DOCS-12…14 (REL-11 parenthetical amended by D-06)
- `.planning/ROADMAP.md` §"Phase 5: Release Chain & release-please Retirement" — success criteria, the two token hazards, the no-loop note, the census scope note
- `.planning/notes/changie-release-management.md` — the milestone's design note (D1–D4, the chain diagram, retirements list)
- `.planning/seeds/SEED-005-changie-template-for-other-repos.md` — why the chain must stay portable (other repos copy it)

### Documented pattern (D-01)
- changie docs, `docs/integrations/release_trigger.md` — `on: push: branches: [main], paths: [CHANGELOG.md]`
- changie docs, `docs/integrations/goreleaser.md` — `--release-notes` with `changie latest`
- `miniscruff/changie-action` `.github/workflows/gen-release-pr.yml` — batch auto → merge → latest → `peter-evans/create-pull-request`
- `peter-evans/create-pull-request` README — its "triggering further workflow runs" section (GitHub App token via `actions/create-github-app-token`)

### Files being replaced or edited
- `.github/workflows/release-please.yml` — `pretag-gate` job to move verbatim; App-token step shape (`actions/create-github-app-token@bcd2ba49…` v3.2.0) to reuse
- `.github/workflows/release.yml` — the one REL-12 edit; `task release:goreleaser` is its single `run:` body
- `release-please-config.json`, `.release-please-manifest.json` — deleted
- `.goreleaser.yaml:404-406` — release-please rationale comment (census)
- `scripts/pr_template_policy.py:54-60` — release-please exemptions → `.changes/` (D-05); `EXEMPT_MARKER` (WR-01)
- `scripts/fragment_policy.py` — `CHANGELOG_EXEMPT_MARKER` (WR-01)
- `.github/workflows/require-issue-link.yml:87-93` — the mirrored `case` list (D-05)
- `.github/workflows/pr-title.yml:1-18` — header becomes "hygiene only" (DOCS-14)
- `.github/workflows/ci.yml:551`, `.github/workflows/post-release-verify.yml:21-23`, `.github/workflows/release.yml:188` — release-please mentions (census)
- `Taskfile.yml:5253` and the `release:dry-run` / `release:dry-run-signed` targets (`:2467`, `:2601`) — REL-13, DOCS-14
- `CONTRIBUTING.md` §Pull requests — DOCS-12; `docs/RELEASE-PROCEDURES.md` (45 refs) — DOCS-13; `docs/RELEASE.md` pipeline paragraph — DOCS-14

### Guards touched
- `internal/upgrade/tag_authority_test.go` — `TestGsdTagCreationIsDisabled` (unchanged), `TestReleasePleaseStaysPreMajor` (re-pointed and renamed), D-06R prose; GRD-16's new assertion
- `internal/upgrade/taskfile_shape_test.go` — job registry `:149` (`pretag-gate`), exception entry `:1442-1444`, `inScopeWorkflowFiles` `:1462`, SAN fixture `:2420`
- `internal/upgrade/goreleaser_shape_test.go:987-1008` — forbidden-`release:`-keys rationale
- `internal/upgrade/release_workflow_shape_test.go:889` — SAN fixture
- `internal/upgrade/changie_shape_test.go:113` — comment (census)

### Project rules
- engram rules `84d1gfpywd` (RED-first, positive assertion), `eyva2svrcz` (never test third-party behaviour), `x1cjy9vyhq` (Go RED evidence), `f18zrdsgx5` (never `[ci skip]`)
- `.planning/phases/04-fragment-required-gate/04-CONTEXT.md` — D-01 (what is tested), D-04 (fragment = ADDED file), D-10 (`pr_template_policy.py` edited only here)
- `.planning/phases/04-fragment-required-gate/04-REVIEW.md` — WR-01

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `release-please.yml`'s `pretag-gate` job (checkout → setup-go `go-version-file: go.mod` → `./.github/actions/install-task` → `task check:cross`) moves verbatim.
- Its App-token step (`actions/create-github-app-token`, `app-id`/`private-key` from `APP_ID`/`APP_PRIVATE_KEY`) is the shape both new jobs reuse.
- `task changie` (isolated `go.tool-changie.mod`, Phase 1) runs changie in CI without a new install path.

### Established Patterns
- Every workflow file is in exactly one of `inScopeWorkflowFiles` / `workflowFileExceptions`; a new file in neither fails `TestWorkflowFilePopulationMatchesDisk`. A `uses:`-only job needs an exception entry with a reason (release-please's job at `taskfile_shape_test.go:1442` is the precedent).
- The two hygiene allowlists (`pr_template_policy.py` and `require-issue-link.yml`) are a mirrored pair; edit both together.

### Integration Points
- `protect-main` (ruleset 20157557, read 2026-10-13): squash-only, linear history, 0 required approvals, 9 required contexts incl. `fragment-required`, `strict: false`. Repo: `squash_merge_commit_message: PR_BODY`, so the release PR's body (D-04) becomes the release commit's message on `main` — a fragment body containing `[skip ci]` would suppress the tag push's workflows; worth a sanitizing check or a note.
- `require_extra_approval_for_unattributed_changes: true` affects Copilot PRs only and has no effect at 0 approvals — not a hazard for the App-authored PR.
- `auto-close-unsolicited-prs.yml` fires only on `feat:`/`perf:` titles — `chore(main):` is safe.
- Until the milestone merges, release-please stays live on `main`: any releasable PR merged to `main` before then cuts a release-please release and needs the quick 261013-esj reconciliation (merge back plus a verbatim `.changes/vX.Y.Z.md` seed).

</code_context>

<specifics>
## Specific Ideas

- "isn't there an action for this and a general approach? Why are we making a space shuttle?" — the maintainer's standing test for this phase (D-01). Prefer the documented pattern and fewer moving parts; question any mechanism the docs don't call for.

</specifics>

<deferred>
## Deferred Ideas

- At Phase 6 entry: if release-please has an open release PR on `main` when the milestone merges, it is orphaned (the merge deletes `release-please.yml`) and should be closed by hand. Phase 6's concern, recorded here so it is not lost.

### Reviewed Todos (not folded)
- "Adopt changie for changelog + version and replace release-please…" (score 0.9, `resolves_phase: 5`): the milestone's origin todo. This phase delivers its checklist steps 2, 3 and 5; it closes at milestone end, as Phases 1–4 recorded.
- "tools/bench/runner pinnedAt() validates a checkout by git rev-parse HEAD alone": bench, unrelated (keyword match only).
- "Reply on GH #85 / #82 / #80 with the reshaped contract": server-mode, unrelated.
- "Pin the Go toolchain so Go 1.27 can build codegraph-go": build, unrelated.

</deferred>

---

*Phase: 05-release-chain-release-please-retirement*
*Context gathered: 2026-10-13*
