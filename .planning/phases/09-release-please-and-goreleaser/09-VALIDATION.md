---
phase: 9
slug: release-please-and-goreleaser
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: validated
nyquist_compliant: true
wave_0_complete: true
created: 2026-07-28
validated: 2026-08-01
---

# Phase 9 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
>
> **RECONCILED 2026-08-01 by `/gsd-validate-phase 9`.** This file began as a seeded draft
> written by plan-phase from `09-RESEARCH.md` *before* PLAN.md files existed, with
> `09-TBD` placeholder task IDs and every row `⬜ pending`. It has now been reconciled
> against the executed phase: each row carries its real plan, its real test, and a status
> measured by running it. The one genuine coverage gap found (`goreleaser check` never ran
> in CI) was closed in `05b26ed`. See the Validation Audit at the foot of this file.

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
- **Before `/gsd-verify-work`:** all of the above green
- **Max feedback latency:** 30 seconds

> **Amended 2026-08-01.** This section previously required "the disposable scratch-branch
> live proof … the real `v1.0.0` must be the *second* end-to-end run of the pipeline, never
> the first." Plan 09-07 (that rehearsal) was **skipped by maintainer decision**, and the
> `v1.0.0` target was removed on 2026-07-29 by maintainer directive (D-06R). The condition
> is therefore not merely unmet — it was retired. The accepted consequence, recorded in
> `09-07-SUMMARY.md` rather than buried, is that 09-08 became the pipeline's first live run.
> It succeeded: `v0.2.0` published, 11/11 jobs green. Kept visible rather than deleted so
> the trade stays auditable.

---

## Per-Task Verification Map

> **Reconciled 2026-08-01.** The `09-TBD` placeholders are gone; every row below names the
> plan that delivered it and a status measured by running the command, not inferred.

| # | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | Status |
|---|------|------|-------------|------------|-----------------|-----------|-------------------|--------|
| 1 | 09-01 | 0 | REL-02 | — | `release.yml`'s literal `name:` + `on.push.tags` still satisfy `releaseWorkflowRefPattern`; fails immediately if filename or trigger drifts | unit (non-vacuous drift guard) | `go test ./internal/upgrade/... -run TestReleaseWorkflowFileMatchesPattern` | ✅ green |
| 2 | 09-02 | 2 | REL-02 | T-09-D04 | D-04 create-vs-upload branch fires `gh release create` when no release exists and `gh release upload --clobber` when one does — proven **both** ways | unit (stubbed `gh` on PATH) | `go test ./internal/upgrade/... -run TestPublishReleaseStepBranches` | ✅ green |
| 3 | 09-03 | 2 | REL-02 | — | PR-title conventional-commit regex accepts conformant titles and rejects non-conformant ones | unit (table-driven) | `go test ./internal/upgrade/... -run TestPRTitleLintAcceptsAndRejects` | ✅ green |
| 4 | 09-03 | 2 | REL-02 | V5 (injection) | New/modified workflow YAML is schema-valid, injection-safe (`env:` indirection for `$TAG`/`$REPO`), `permissions:` minimal | static (CI gate) | `actionlint .github/workflows/*.yml` — job `actionlint (workflow static analysis)` | ✅ green |
| 5 | 09-01 | 1 | REL-02 | — | `release-please-config.json` + `.release-please-manifest.json` resolve the baseline with no bootstrap PR | integration (read-only dry run) | — **retired, see note below** | ✅ superseded by production |
| 6 | 09-01 | 2 | REL-02 | V6 (crypto) | `releaseWorkflowRefPattern` still accepts `release.yml@refs/tags/v…` and rejects a non-release workflow — no regression | unit (existing) | `go test ./internal/upgrade/... -run TestReleaseWorkflowRefPattern` | ✅ green |
| 7 | 09-01 | 2 | REL-02 | — | `.goreleaser.yaml` remains valid and unchanged (D-05 no-regression) | static (CI gate) | `goreleaser check` — job `goreleaser check (config validation, DIST-01)` | ✅ green (**gap closed `05b26ed`**) |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

### Row 5 — retired rather than filled

The dry run existed to prove the config resolves the baseline and cuts without a bootstrap PR.
**Release-please has since actually cut `v0.2.0`** from this config — computed `0.1.0 → 0.2.0`,
tag and Release authored by `fzy-release-please[bot]`, no bootstrap PR. Production is
strictly stronger evidence than a dry run of the same code path.

Automating it would also cost more than it returns: `release-please release-pr --dry-run`
requires a `GITHUB_TOKEN` and a live API call on every CI run, and can fail for reasons
unrelated to the config it is meant to validate. Recorded as **superseded**, not as an
unmet gap — if the config is ever restructured, the next real release re-tests it.

### Row 7 — the gap that was real, and its non-vacuity proof

Nothing in CI validated `.goreleaser.yaml`. It is consumed at release time by
`goreleaser build --single-target`, so a config error would surface only during an actual
release — the most expensive possible moment. Closed by a `goreleaser-check` job in
`ci.yml` (pinned `goreleaser/v2@v2.17.1`, matching the `actionlint` job's install shape).

Proven non-vacuous by measurement, not assertion — this repo has a documented history
(Phase-8 `CR-01`/`WR-02`, and an inverted `rg -qv` guard) of gates that were present but
could never fire:

| Config fed to `goreleaser check` | Real exit code |
|---|---:|
| the real `.goreleaser.yaml` | **0** |
| same config + one unknown top-level key (**valid YAML**) | **1** — `field totally_bogus_key not found in type config.Project` |

The rejecting input is valid YAML, so this demonstrates the gate validates GoReleaser's
**schema**, not merely that the file parses. `git diff .goreleaser.yaml` is clean — the
mutation was applied to a copy and never to the tracked config.

---

## Wave 0 Requirements

- [x] `TestReleaseWorkflowFileMatchesPattern` (`internal/upgrade/release_workflow_shape_test.go:300`) — **non-vacuity is built into the test itself** rather than demonstrated once and lost: subtests `reject_renamed_workflow` and `reject_branch_ref` are permanent rejecting inputs. `TestReleaseWorkflowTriggerIsTagPushOnly` (line 347) covers the trigger half.
- [x] Stubbed-`gh` test for the D-04 create-vs-upload branch — `TestPublishReleaseStepBranches` (`release_publish_step_test.go:168`), **5 cases**: `release_exists_uploads_with_clobber`, `release_absent_prerelease_tag_creates_with_prerelease_and_notes`, `release_absent_stable_tag_creates_without_prerelease`, `zero_assets_fails_loud_invokes_neither_branch`, `release_absent_create_fails_no_upload_fallback`. Both branches exercised, as required.
- [x] `actionlint` — CI job `actionlint (workflow static analysis)`, pinned `@v1.7.12`, and a **required status check** on `main`.
- [x] PR-title lint regex + table-driven test — `TestPRTitleLintAcceptsAndRejects` (`pr_title_lint_test.go:81`), 17 subtests spanning every accepted type, scopes, breaking-change marker, four rejecting cases, and an adversarial shell-metacharacter case.

> **The Phase-8 precedent below was honoured.** Every guard in this phase carries its
> rejecting input as a permanent subtest, so non-vacuity cannot decay: a future change that
> makes a guard unfireable breaks a named test rather than passing silently. The one gate
> added during validation (`goreleaser check`) was proven non-vacuous by measurement — see
> "Row 7" above.

> **Phase-8 precedent that makes non-vacuity mandatory here:** `CR-01` and `WR-02` were guards
> that were present but never fired, and the recent quick task shipped an inverted `rg -qv`
> guard with the same defect. Every new check in this phase must be demonstrated failing
> against a rejecting input before it counts as coverage.

---

## Manual-Only Verifications

**All four resolved 2026-08-01. Zero manual-only items remain outstanding.**

| Behavior | Requirement | Status | Evidence |
|----------|-------------|--------|----------|
| GitHub App creation + installation; `APP_ID`/`APP_PRIVATE_KEY` secrets exist | REL-02 | ✅ satisfied | `gh secret list` → `APP_ID` (2026-07-29), `APP_PRIVATE_KEY` (2026-07-30). Both live. The first `APP_PRIVATE_KEY` held the **wrong App's key** and failed with a 401 JWT-decode error — diagnosed and replaced by the maintainer (`09-06-SUMMARY.md`) |
| release-please's App-token tag push actually fires `release.yml` | REL-02 | ✅ satisfied | Run `30675077940`: `event=push`, `ref=v0.2.0`, `conclusion=success`, 11/11 jobs green. This was the single genuinely-open fact 09-07 was written to prove; it held on the real cut |
| The cosign OIDC cert SAN for an **App-token-triggered** run matches `releaseWorkflowRefPattern` | REL-02 (half B) | ✅ satisfied | `cosign verify-blob` per `docs/RELEASE.md` §6(a) run verbatim → `Verified OK` on darwin/arm64 **and** linux/amd64. `TestVerifyReleaseE2E` ran (not skipped) against the production issuer and SAN pattern, both subtests passing |
| An already-shipped `v0.1.0` binary's `codegraph upgrade` succeeds against a release-please-cut release | REL-02 (half B) | ✅ satisfied | Two platforms, from genuinely shipped v0.1.0 binaries downloaded fresh: darwin/arm64 `773223fd…` → `a64c1549…`, linux/amd64 `6f05e630…` → `3cba14af…` (x86_64 container). Each installed sha equals both the SLSA-attested subject and the checksums entry |

### The disposable scratch-branch proof — OBSOLETE, retired 2026-08-01

The seven-step rehearsal that stood here required cutting a disposable prerelease and tearing
it down *before* "the real `v1.0.0`". It is retired for two independent reasons, either of
which alone would suffice:

1. **Plan 09-07 was skipped by maintainer decision.** Three of its four target facts were
   already established by other means; the fourth (an App-authored tag push fires
   `release.yml`) was genuinely open but fails *harmlessly* — no workflow run means nothing
   signed and nothing published. Running it would have permanently double-signed the
   `v0.2.0` tag name in the public Sigstore transparency log, which cannot be undone.
   Full reasoning in `09-07-SUMMARY.md`.
2. **There is no `v1.0.0` to rehearse for.** The version target was removed on 2026-07-29 by
   maintainer directive (D-06R); versions follow Conventional Commits, and the first
   automated cut was `0.2.0`.

Controls were added in its place rather than the risk simply being absorbed: branch ruleset
`protect-main` (squash-only, linear history, 6 required status checks) gating the merge that
creates the tag, and both irreversible checkpoints hardened from `gate="blocking"` to
`gate="blocking-human"` so an unattended run could not auto-select "publish permanently".

> **What the original note got right, kept for the record:** it observed that the old REL-02
> "could only be satisfied by the real thing", which had blocked Phase 8's UAT twice. That
> diagnosis was correct, and it was fixed — but by *recategorizing REL-02 from an event into
> a property*, not by adding a rehearsal. The rehearsal was a second answer to a question
> already resolved elsewhere.
>
> The accepted consequence — 09-08 became the pipeline's first live run — is recorded as an
> outcome, not a vindication. Phase 8's first live release *did* catch two bugs green CI had
> missed. This one did not, but that was not knowable in advance.

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies — 6 of 7 rows carry a runnable command; row 5 is recorded as superseded by production evidence, with reasoning
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references — all four Wave 0 items delivered
- [x] Every new guard demonstrated failing against a rejecting input (non-vacuity) — rows 1–3 carry permanent rejecting subtests; row 7's gate proven by measurement (exit 1 on valid-YAML/invalid-schema, exit 0 on the real config)
- [x] No watch-mode flags — Go `testing` and static CLI gates only; `-count=1` used throughout
- [x] Feedback latency < 30s — the five mapped Go tests complete in **0.551s** together
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** validated 2026-08-01 via `/gsd-validate-phase 9`.

---

## Validation Audit 2026-08-01

| Metric | Count |
|--------|-------|
| Rows audited | 7 |
| Already covered | 5 |
| Gaps found | 2 |
| Resolved | 1 (row 7 — `goreleaser check` CI gate, `05b26ed`) |
| Retired as superseded | 1 (row 5 — release-please dry run) |
| Escalated | 0 |
| Manual-only outstanding | **0** (all four resolved) |

**What the audit changed.** The file arrived as a plan-time draft: `status: draft`,
`nyquist_compliant: false`, seven `09-TBD` placeholder task IDs, every row `⬜ pending`, and
a Manual-Only section built around a rehearsal that never ran and a version that was never
cut. Statuses were established by **running** each command, not by reading the summaries.

**The one real gap.** `.goreleaser.yaml` had no automated validation anywhere — a config
error would have surfaced only during an actual release. `goreleaser check` now runs in CI.

**Caveat on scope.** This audit verifies that Phase 09's mapped behaviors have automated
coverage that currently passes. It does not re-verify the phase goal (that is
`09-VERIFICATION.md`, `status: passed`, 4/4) and it does not extend to the two known open
defects outside this phase's scope — issue #16 (`CheckRegression` never compares
`Metrics.Repo`) and issue #17 (a daemon test failing 3/3 under full-suite load while CI's
isolation strategy hides it). Neither is a Phase 09 deliverable; both are tracked.
