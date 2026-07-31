---
phase: 09-release-please-and-goreleaser
plan: 07
status: SKIPPED
skipped_at: 2026-07-31
skipped_by: maintainer decision
subsystem: infra
tags: [release-please, cosign, sigstore, skipped, deviation]

# Dependency graph
requires:
  - phase: 09-release-please-and-goreleaser
    provides: "09-06's live release PR #2, authored by the App — the evidence this plan was going to re-derive"
provides:
  - "A recorded decision NOT to run a disposable live release, with the evidence that made it unnecessary"
  - "09-08 proceeds as the pipeline's first live run, deliberately and with the risk stated, rather than by omission"
affects: [09-08]

tech-stack:
  added: []
  patterns: []

key-files:
  created: []
  modified: []
---

# 09-07 — SKIPPED (not executed)

**Task 1's `blocking-human` gate was reached and APPROVED by the maintainer.** Task 2
then halted on a permission deny (`Bash(gh pr merge *)`), and during that pause the
plan's premise was re-examined and the maintainer chose to skip it. Nothing was merged,
tagged, released, dispatched, or pushed. Tasks 2 and 3 never ran.

## Why this plan was skipped

09-07 exists to establish four facts it claims "only a live release event can
establish." Three of the four were already established by other means, and the fourth
is guaranteed by construction.

**Fact 4 — the cosign SAN satisfies `releaseWorkflowRefPattern` — is structurally
guaranteed, not open.** From `internal/upgrade/verify.go`:

```go
releaseWorkflowRefPattern = `^https://github\.com/` + releaseRepoSlug +
    `/\.github/workflows/release\.ya?ml@refs/tags/v[0-9][^\s]*$`
```

The pattern matches repo slug, workflow file path, and tag ref. **It contains no actor
component.** Who pushes the tag — a maintainer's PAT, a GitHub App installation token,
anything else — cannot appear in the SAN and therefore cannot affect whether it matches.
`v0.1.0` and `v0.0.0-rc.3` already verified "OK" through this same pattern (see engram
`8sa948y0g4`). The App-token question is simply orthogonal to it.

**Fact 3 — the publish step takes the upload branch — is covered by plan 09-02's test.**
That test extracts the real shell out of the real `.yml` and runs it against a stubbed
`gh` on PATH, across five cases. It structurally cannot drift from CI.

**Fact 2 — the App's installation scope suffices — is largely demonstrated already.**
PR #2 exists, authored by `app/fzy-release-please`, on a branch the App pushed, carrying
the `autorelease: pending` label the App applied. That is `contents: write` +
`pull_requests: write` + `issues: write`, observed live.

**Fact 1 — an App-authored tag push actually starts `release.yml` — is genuinely open.**
It is documented GitHub behavior (App installation tokens trigger workflows; the default
`GITHUB_TOKEN` does not), and it is the stated reason D-02 chose an App token — written
into `release-please.yml`'s own header comment. But it has not been empirically observed
in this repository.

## What the skip costs, stated plainly

09-08 becomes the pipeline's **first** live run rather than its second. That is the exact
condition 09-07 was written to eliminate, and Phase 8's first live release did catch two
bugs green CI had missed (`8sa948y0g4`: the rc.1 linux-only `go.sum` gap and the rc.2
SLSA `private-repository` opt-in). This risk is accepted knowingly, not overlooked.

The mitigating asymmetry: if fact 1 is wrong, `release.yml` never runs, so **nothing is
signed and nothing is published.** The failure is a GitHub Release sitting empty — loud,
immediate, and forward-fixable by re-pushing the tag or cutting the next patch. It cannot
produce a *wrongly signed* artifact, because producing any artifact at all requires the
workflow to have run.

## What running it would have cost

- **Permanent, undeletable double-signing of the `v0.2.0` tag name in the public Sigstore
  transparency log**, from two different commits. Teardown cannot remove log entries.
- **Tag-name reuse** — defensible only under a chain of conditions (prerelease for its
  entire life, `releases/latest` never resolving to it, repo private) that must each hold
  and each be evidenced.
- A revert commit on `main`, and hand-manipulation of release-please's own state — the
  tool whose entire purpose is to own this lifecycle.

On a **0.x, private** repository where the forward fix is "cut 0.2.1," that price buys
one fact whose failure mode is already harmless.

## Controls added instead (cheaper, and permanent)

Rather than proving the pipeline once with a disposable artifact, the release path was
made structurally safer:

- **Branch ruleset `protect-main`** (id 20157557, active): PR required, squash-only,
  linear history, no force-push, no deletion, and **6 required status checks** — `test`,
  `actionlint`, `govulncheck (DIST-03, blocking)`, `perf regression gate (PERF-02,
  INDX-06)`, `pr-title`, `reproducibility (DIST-04)`. Since release-please only creates
  the tag *after* its PR merges into `main`, this gates the release path at the merge.
- **09-07/09-08 Task 1 hardened to `gate="blocking-human"`** (commit `ba5a548`). Both
  carried `gate="blocking"` while `workflow.auto_advance=true`, so an unattended run
  would have auto-selected 09-08's first option — publish permanently — with no human
  input.

Note the ruleset's bypass is `RepositoryRole` 5 (admin), `mode=always`, so it does not
constrain the maintainer.

## Deviation record

This is a deliberate departure from a planned, plan-checker-passed artifact with its own
STRIDE register. It was proposed by the orchestrator, argued from evidence, and decided
by the maintainer. It is recorded here rather than by deleting the plan, so the reasoning
survives and can be overturned: 09-07-PLAN.md remains on disk, unmodified except for its
Task 1 gate attribute.

**To reverse this decision,** run `/gsd-execute-phase 9` before 09-08 and the plan
executes as written.

## Requirement impact

REL-02 is unaffected. 09-07 was never the plan that satisfies it — 09-08 is. This skip
removes a rehearsal, not a deliverable.
