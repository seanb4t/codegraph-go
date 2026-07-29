---
phase: 9
slug: release-please-and-goreleaser
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-07-28
---

# Phase 9 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
>
> **SEEDED DRAFT.** Written by plan-phase from `09-RESEARCH.md` §Validation Architecture
> *before* PLAN.md files existed. The Per-Task Verification Map below is keyed to
> research-derived behaviors, not real task IDs. `/gsd-validate-phase 9` MUST reconcile
> this against the finalized plans — do not treat the task IDs here as authoritative.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` (existing `internal/upgrade` convention) + stubbed-`gh` shell tests for CI-logic-only pieces + `actionlint` / `goreleaser check` as static gates |
| **Config file** | none new — extends `internal/upgrade/verify_test.go`'s existing pattern |
| **Quick run command** | `go test ./internal/upgrade/...` |
| **Full suite command** | `go test ./...` + `actionlint .github/workflows/*.yml` + `goreleaser check` |
| **Estimated runtime** | ~30 seconds (quick) / ~4 minutes (full, excluding the isolated `internal/daemon` leg) |

---

## Sampling Rate

- **After every task commit:** `go test ./internal/upgrade/...` + `actionlint` on any touched workflow file
- **After every plan wave:** `go test ./...` + `actionlint .github/workflows/*.yml` + `goreleaser check`
- **Before `/gsd-verify-work`:** all of the above green **PLUS** the disposable scratch-branch live proof (see Manual-Only Verifications) — the real `v1.0.0` must be the *second* end-to-end run of the pipeline, never the first
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

> Task IDs are **placeholders** pending plan finalization. Behaviors and commands are real
> (lifted from `09-RESEARCH.md` §Validation Architecture → Phase Requirements → Test Map).

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 09-TBD | TBD | 0 | REL-02 | — | `release.yml`'s literal `name:` + `on.push.tags` still satisfy `releaseWorkflowRefPattern`; fails immediately if filename or trigger drifts | unit (non-vacuous drift guard) | `go test ./internal/upgrade/... -run TestReleaseWorkflowFileMatchesPattern` | ❌ W0 | ⬜ pending |
| 09-TBD | TBD | 0 | REL-02 | T-09-D04 | D-04 create-vs-upload branch fires `gh release create` when no release exists and `gh release upload --clobber` when one does — proven **both** ways | unit (stubbed `gh` on PATH) | new shell/bats test | ❌ W0 | ⬜ pending |
| 09-TBD | TBD | 0 | REL-02 | — | PR-title conventional-commit regex accepts conformant titles and rejects non-conformant ones | unit (table-driven) | regex tested directly against a valid/invalid title table | ❌ W0 | ⬜ pending |
| 09-TBD | TBD | 1 | REL-02 | V5 (injection) | New/modified workflow YAML is schema-valid, injection-safe (`env:` indirection for `$TAG`/`$REPO`), `permissions:` minimal | static | `actionlint .github/workflows/release-please.yml .github/workflows/release.yml .github/workflows/ci.yml` | ❌ W0 (tool not installed) | ⬜ pending |
| 09-TBD | TBD | 1 | REL-02 | — | `release-please-config.json` + `.release-please-manifest.json` are well-formed and resolve the `v0.1.0` baseline with no bootstrap PR | integration (read-only dry run) | `npx -y release-please@latest release-pr --dry-run --debug --token=$GITHUB_TOKEN --repo-url=seanb4t/codegraph-go --config-file=release-please-config.json --manifest-file=.release-please-manifest.json` | ❌ W0 | ⬜ pending |
| 09-TBD | TBD | 2 | REL-02 | V6 (crypto) | `releaseWorkflowRefPattern` still accepts `release.yml@refs/tags/v…` and rejects a non-release workflow — no regression | unit (existing) | `go test ./internal/upgrade/... -run TestReleaseWorkflowRefPattern` | ✅ exists (`verify_test.go:118-136`) | ⬜ pending |
| 09-TBD | TBD | 2 | REL-02 | — | `.goreleaser.yaml` remains valid and unchanged (D-05 no-regression) | static | `goreleaser check` | ✅ exists | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `TestReleaseWorkflowFileMatchesPattern` (new, in `internal/upgrade/`) — parses `release.yml`'s literal `name:` / `on.push.tags` and asserts against the compiled `releaseWorkflowRefPattern`. **Must be proven non-vacuous**: temporarily rename the workflow or alter the trigger, observe the test fail, restore.
- [ ] Stubbed-`gh` shell test for the D-04 create-vs-upload branch — the single highest-risk edit in the phase. Both branches must be exercised; a test that only covers the upload path proves nothing about the `rc`-tag create path.
- [ ] `actionlint` install step (CI or local `go install`) — no config file needed.
- [ ] PR-title lint regex + its own table-driven test (valid/invalid sample titles).

> **Phase-8 precedent that makes non-vacuity mandatory here:** `CR-01` and `WR-02` were guards
> that were present but never fired, and the recent quick task shipped an inverted `rg -qv`
> guard with the same defect. Every new check in this phase must be demonstrated failing
> against a rejecting input before it counts as coverage.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| GitHub App creation + installation; `APP_ID`/`APP_PRIVATE_KEY` (or `client-id`) secrets exist | REL-02 | Repo has **zero** secrets configured today; App creation is an account-level action no executor can perform | Maintainer creates the App with Contents/PRs/Issues **write** installation permissions (a different scope from the workflow `permissions:` block — RESEARCH Pitfall 2), installs it on `seanb4t/codegraph-go`, stores the secrets |
| release-please's App-token tag push actually fires `release.yml` in this repo's real Actions configuration | REL-02 | Only observable on a live event; repo/org Actions settings cannot be fully confirmed by static API reads | Disposable scratch-branch proof, steps below |
| The cosign OIDC cert SAN issued for an **App-token-triggered** run matches `releaseWorkflowRefPattern` | REL-02 (half B) | The static test proves the workflow *file* didn't drift, not that GitHub's runtime SAN construction behaves identically for an App-triggered run | `cosign verify-blob --bundle <asset>.sigstore.json …` per `docs/RELEASE-PROCEDURES.md` §6 against the disposable release's artifacts |
| An already-shipped `v0.1.0` binary's `codegraph upgrade` succeeds against a release-please-cut release | REL-02 (half B) | Needs a real prior binary and a real new release | Download the `v0.1.0` asset, run `codegraph upgrade` against the disposable prerelease |

### Disposable scratch-branch live proof (run BEFORE the real `v1.0.0`)

1. Short-lived branch off `main` with one trivial `fix:` commit.
2. Run release-please against it via the action's `target-branch` input (or the CLI's `--target-branch`) to produce a real release PR.
3. Merge it — release-please cuts a **real** tag via the App token. **Force it prerelease-shaped** (`-` suffix) so `codegraph upgrade` never resolves it as "latest" for a real user (RESEARCH Pitfall 4).
4. Confirm the tag push fired `release.yml`.
5. `cosign verify-blob` + `slsa-verifier verify-artifact` against the disposable artifacts (`docs/RELEASE-PROCEDURES.md` §6).
6. Delete the disposable tag/release/branch per §7's rollback procedure.
7. Only then cut the real `v1.0.0`.

> This is the direct remedy for the failure that blocked Phase 8's UAT twice: the old REL-02
> could only be satisfied by the real thing. Here the real `v1.0.0` is the *second* end-to-end
> run, and every property except the final publish is provable before it.

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] Every new guard demonstrated failing against a rejecting input (non-vacuity)
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
