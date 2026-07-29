---
created: 2026-07-14T17:43:30.073Z
title: Document release procedures (maintainer runbook)
area: docs
resolves_phase: 8
status: resolved
resolved_at: 2026-07-28T00:00:00.000Z
resolved_by_phase: 9
files:
  - docs/RELEASE.md
  - .github/workflows/release.yml
  - internal/upgrade/verify.go
---

## Resolution (2026-07-28, Phase 9)

Phase 8 shipped `docs/RELEASE-PROCEDURES.md`'s original 8 sections; Phase 9
(`09-04-PLAN.md`) rewrote the three sections this phase's release-please
automation invalidated and added two new ones, closing this todo in full:

- **§3 Branch/tag model** — rewritten for the fast-forward merge model.
- **§4 Cutting a release** — rewritten for the release-PR-merge trigger,
  replacing the maintainer-manual `git tag` as the normal path.
- **§7 Rollback / cleanup** — broadened to cover all three artifacts a
  release-please cut leaves behind.
- **§9 GitHub App prerequisite (D-02)** — new section covering App
  creation, installation permissions, secrets, and the pre-flight checks
  this todo's original "non-obvious, easy-to-forget steps" problem
  statement was asking for.
- **§10 Recorded divergences (Phase 9)** — new section recording every
  divergence this phase accepted.

§1, §2, §5, §6, §8 (all originally requested by this todo) survive from
Phase 8 substantively unchanged, with additive automation notes on §1 and
§6.

## Problem

`docs/RELEASE.md` documents how a *user* verifies a downloaded artifact, but there is no **maintainer runbook** for how to actually *cut* a release. The first live releases in this session surfaced several non-obvious, easy-to-forget steps and gotchas that should be written down before the next release (or a second maintainer) is attempted:

- **Pre-tag cross-platform check:** run `GOOS=<os> GOARCH=<arch> go list -mod=readonly ./...` for all 6 targets before tagging. `rc.1` failed because a **linux-only** `go.sum` hash (`prometheus/procfs`) was missing — invisible on a darwin-only dev machine and to green local CI.
- **Tag conventions:** `release.yml` triggers on `v[0-9]*`. Release candidates are `v0.0.0-rc.N` (the `-` makes them GitHub prereleases); the stable tag is `vX.Y.Z` (no suffix → full release, and becomes what `codegraph upgrade` resolves as "latest"). Internal milestone markers use a **non-matching** name (`milestone-vX.Y`) so they never fire a release.
- **Private-repo SLSA:** while the repo is private, the SLSA generic generator needs `private-repository: true` (already set) — and note that keyless cosign already publishes the repo name to the public Sigstore/Rekor transparency log via the cert SAN.
- **LOCKED contract (do not break):** `release.yml`'s filename, its `v[0-9]*` tag trigger, and the cosign keyless identity are pinned by `internal/upgrade/verify.go`'s `releaseWorkflowRefPattern`/`releaseOIDCIssuer`/`releaseRepoSlug`. Renaming the workflow or changing the trigger silently breaks `codegraph upgrade` for every user — must change `verify.go` in lockstep. Asset names must match `releaseAssetName()`.
- **Post-release verification:** download a binary, run `--version`, and `cosign verify-blob --bundle <asset>.sigstore.json --certificate-oidc-issuer https://token.actions.githubusercontent.com --certificate-identity-regexp '<the SAN pattern>' <binary>` (plus `slsa-verifier verify-artifact` against `multiple.intoto.jsonl`).
- **Signing flakiness:** local 1Password commit signing was intermittently failing during pipeline commits (`failed to fill/write commit object`) — document the `-c commit.gpgsign=false` fallback per the repo rule for automated commits only.

## Solution

TBD — add a `docs/RELEASE-PROCEDURES.md` (or a "Cutting a release" section in `docs/RELEASE.md` / a `CONTRIBUTING.md`) as a step-by-step maintainer runbook covering: pre-tag checks → tag conventions (rc vs stable vs marker) → what the tag push triggers → the verify.go LOCKED contract → post-release artifact verification commands → rollback/cleanup (deleting a failed rc tag+release; nothing is published until the build gate passes, so a failed build produces no Sigstore entries). Pairs naturally with backlog item 999.1 (local build/Taskfile/CONTRIBUTING).
