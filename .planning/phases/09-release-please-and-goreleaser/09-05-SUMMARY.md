---
phase: 09-release-please-and-goreleaser
plan: 05
subsystem: infra
tags: [github-app, secrets, ci, release-please, checkpoint]

# Dependency graph
requires:
  - phase: 09-release-please-and-goreleaser
    provides: "09-01's release-please spine + App-token step in release-please.yml, and 09-04's docs/RELEASE-PROCEDURES.md §9 runbook this checkpoint follows"
provides:
  - "An installed GitHub App (`fzy-release-please`, App ID 3982691) with the three required repository permissions plus mandatory metadata:read"
  - "Two repository secrets, APP_ID and APP_PRIVATE_KEY, confirmed present by name (never by value)"
  - "Confirmed repository Actions configuration: Actions enabled, workflow permissions read + PR-create/approve enabled"
  - "docs/RELEASE-PROCEDURES.md §9 corrected: documents App reuse as an acceptable shape, and adds the previously-undocumented 'create and approve pull requests' pre-flight check with the reasoning for why it is not load-bearing for D-02"
affects: [09-06, 09-07, 09-08]

# Tech tracking
tech-stack:
  added: []
  patterns: []

key-files:
  created: []
  modified:
    - docs/RELEASE-PROCEDURES.md

key-decisions:
  - "The maintainer reused a pre-existing GitHub App (`fzy-release-please`, App ID 3982691, Client ID `Iv23liWI8ZRUnsSPAcOr`) rather than creating a new single-purpose App the plan suggested naming `codegraph-go-release-please`. Permissions were verified still exactly the required three (Contents/Pull requests/Issues, all write) plus the mandatory implicit metadata:read, so the App is correctly scoped even though it is shared. Accepted as a documented deviation with its real consequence recorded against threat T-09-05-02: a key leak's blast radius now spans every repository the App is installed on, not just this one. `repository_selection` (single-repo vs. all-repos) could not be read via the unauthenticated `/apps/<slug>` endpoint (HTTP 403 without App auth) and is recorded as unverified rather than asserted either way."
  - "The 'Allow GitHub Actions to create and approve pull requests' repository setting was independently re-verified live (via `gh api repos/seanb4t/codegraph-go/actions/permissions/workflow`) at SUMMARY-authoring time and returned `can_approve_pull_request_reviews: true`. An earlier check during setup had shown it disabled; the maintainer enabled it before this SUMMARY was finalized. Recorded as met, with the live command output as evidence. `default_workflow_permissions` remains `read`, so `GITHUB_TOKEN` itself is unaffected — only the PR-creation/approval capability for that default token was widened. Also recorded: D-02's App-token design does not depend on this setting (the App authors the PR as a distinct actor from `GITHUB_TOKEN`), so its value was never load-bearing for this pipeline — useful context for a future maintainer weighing whether to turn it back off."
  - "docs/RELEASE-PROCEDURES.md §9 was corrected in two places where it disagreed with what actually happened: (1) it now documents that reusing an existing shared App is an acceptable installation shape, alongside the blast-radius tradeoff and the repository_selection verification limitation; (2) it now documents the 'create and approve pull requests' setting and its pre-flight command, which plan 09-05 required checking but 09-04's runbook draft never captured."

patterns-established: []

requirements-completed: []

coverage:
  - id: D1
    description: "Both required repository secrets (APP_ID, APP_PRIVATE_KEY) exist, confirmed by name and timestamp only"
    verification:
      - kind: other
        ref: "gh secret list --repo seanb4t/codegraph-go"
        status: pass
    human_judgment: false
  - id: D2
    description: "The GitHub App's declared installation permissions are exactly Contents/Pull requests/Issues at read-and-write, with metadata:read present only as GitHub's mandatory implicit grant (not an over-scope)"
    verification:
      - kind: other
        ref: "gh api /apps/fzy-release-please --jq '{slug,id,owner,permissions,events}'"
        status: pass
    human_judgment: false
  - id: D3
    description: "Repository Actions configuration confirmed: Actions enabled; workflow default permissions read; PR create-and-approve enabled (re-verified live, superseding an earlier disabled reading)"
    verification:
      - kind: other
        ref: "gh api repos/seanb4t/codegraph-go/actions/permissions; gh api repos/seanb4t/codegraph-go/actions/permissions/workflow"
        status: pass
    human_judgment: false
  - id: D4
    description: "No private key material left in the working tree; no secret value appears anywhere in this plan's artifacts"
    verification:
      - kind: other
        ref: "git status --porcelain | rg '\\.(pem|key)$' (no matches); repo-wide fd -H -e pem -e key (no matches)"
        status: pass
    human_judgment: true
    rationale: "Executor visibility is scoped to this repository's working tree; whether the maintainer also deleted the downloaded .pem from their local Downloads folder (outside the repo) is attested by the maintainer's report, not independently observable by the executor. Flagging for human confirmation rather than asserting full deletion as machine-verified."
  - id: D5
    description: "Accepted deviation recorded: reused pre-existing shared App vs. plan's suggested purpose-built App, including the T-09-05-02 blast-radius consequence and the unverified repository_selection scope"
    verification:
      - kind: other
        ref: "manual review of this SUMMARY's key-decisions and Deviations section against threat_model T-09-05-02"
        status: pass
    human_judgment: false

duration: 15min
completed: 2026-07-29
status: complete
---

# Phase 9 Plan 5: GitHub App creation + secrets checkpoint Summary

**GitHub App `fzy-release-please` (ID 3982691) installed with exactly Contents/Pull requests/Issues at read-write plus mandatory metadata:read; both `APP_ID`/`APP_PRIVATE_KEY` secrets confirmed present by name; repository Actions PR-create/approve setting confirmed enabled on live re-check.**

## Performance

- **Duration:** ~15 min
- **Tasks:** 1 (checkpoint:human-action)
- **Files modified:** 1 (docs/RELEASE-PROCEDURES.md, corrections only — no code)

## Accomplishments

- The GitHub App required by D-02 (an App installation token whose tag push actually triggers `release.yml`, unlike the default `GITHUB_TOKEN`) is installed and its two secrets are stored under the exact names `release-please.yml` expects.
- Permission-scope must_have **passes**, with reasoning recorded rather than merely asserted: the App's declared permissions are `{"contents":"write","issues":"write","metadata":"read","pull_requests":"write"}`. `metadata: read` is GitHub's mandatory, non-removable grant on every App installation — it is not a discretionary over-grant and does not violate the "no broader scope" requirement. `"events": []` confirms no webhook subscription was created, satisfying the plan's "uncheck Webhook → Active" instruction (release-please is driven by workflow runs, not webhooks, so a webhook here would be pure unused attack surface).
- Repository Actions configuration confirmed twofold: Actions are enabled (`{"enabled":true,"allowed_actions":"all","sha_pinning_required":false}`), and the "Allow GitHub Actions to create and approve pull requests" setting is enabled (`can_approve_pull_request_reviews: true`, `default_workflow_permissions: "read"`), re-verified live at SUMMARY-authoring time.
- No branch protection exists on `main` (`404 Branch not protected`), so the fast-forward merge D-09 relies on in 09-06 is ungated and no App bypass-actor configuration is required.
- `docs/RELEASE-PROCEDURES.md` §9 corrected in two places where it disagreed with reality (see Deviations below).

## Task Commits

1. **Task 1: Create + install the GitHub App and store APP_ID / APP_PRIVATE_KEY** — maintainer-performed, no executor commit (per plan's explicit prohibition on any executor creating/installing the App or writing secrets). Verification commands below were run read-only by the executor after the maintainer reported completion.

**Plan metadata:** committed alongside this SUMMARY (docs commit).

## Verification Evidence (recorded verbatim, names/metadata only — no secret values)

**1. `gh secret list --repo seanb4t/codegraph-go`:**
```
APP_ID	2026-07-29T17:54:12Z
APP_PRIVATE_KEY	2026-07-29T17:59:49Z
```
Both required secrets present.

**2. `gh api /apps/fzy-release-please --jq '{slug,id,owner,permissions,events}'`:**
```json
{"events":[],"id":3982691,"owner":"seanb4t","permissions":{"contents":"write","issues":"write","metadata":"read","pull_requests":"write"},"slug":"fzy-release-please"}
```
Exactly the three required permissions at write, plus mandatory metadata:read; no webhook events subscribed.

**3. `gh api repos/seanb4t/codegraph-go/actions/permissions`:**
```json
{"enabled":true,"allowed_actions":"all","sha_pinning_required":false}
```

**4. `gh api repos/seanb4t/codegraph-go/actions/permissions/workflow`** — re-verified live at SUMMARY-authoring time, superseding an earlier reading taken during setup that showed this disabled:
```json
{"default_workflow_permissions":"read","can_approve_pull_request_reviews":true}
```
The plan's acceptance criterion "the create-and-approve-pull-requests setting is recorded as confirmed enabled" is **met**, evidenced by this live re-check. `default_workflow_permissions` staying `"read"` means `GITHUB_TOKEN` itself remains minimally privileged; only the PR-creation/approval capability was widened. Worth noting for a future maintainer: D-02's App-token design does not actually depend on this setting — release-please's PR is authored by the App as a distinct actor from `GITHUB_TOKEN` — so this setting was never load-bearing for the pipeline, even during the window it was off.

**5. Key-material check:**
```
git status --porcelain | rg '\.(pem|key)$'   -> no matches
fd -H -e pem -e key (repo-wide)               -> no matches
```
No private-key file left anywhere in the working tree.

**6. `gh api repos/seanb4t/codegraph-go/branches/main/protection`:**
```
HTTP 404 "Branch not protected"
```
Confirms no branch-protection/bypass-actor configuration is needed for this App to push its version-bump commit.

## Files Created/Modified

- `docs/RELEASE-PROCEDURES.md` — §9 corrected: (1) documents that reusing an existing shared App is an acceptable installation shape (with the blast-radius tradeoff and the `repository_selection`-unreadable-via-unauthenticated-endpoint limitation both noted), and (2) adds the previously-missing "create and approve pull requests" pre-flight command and the reasoning for why its value is not load-bearing for D-02. No other section touched; zero code/workflow diff.

## Decisions Made

- **Accepted deviation — pre-existing shared App, not purpose-built.** The plan suggested naming a new App `codegraph-go-release-please`; the maintainer instead reused an existing App, `fzy-release-please` (App ID `3982691`, owner `seanb4t`), presumably already installed on other repositories. Its permission set is nonetheless correct and minimal — verified above — and reusing one release-automation App across repos is normal practice. Real consequence recorded against **T-09-05-02** (private-key disclosure, severity high): that threat's mitigation was scoped assuming a single-repo App; with a shared App, the blast radius of a key leak now spans every repository the App is installed on, not just this one. `repository_selection` (single-repo vs. all-repos install scope) could not be read via the unauthenticated `/apps/fzy-release-please` endpoint (which returned no such field; an authenticated `/installation`-shaped call would be needed and was correctly not attempted, since that requires an App-signed JWT this checkpoint has no reason to mint) — this scope is recorded as **unverified**, not asserted either way.
- **Client ID recorded for future migration only.** `Iv23liWI8ZRUnsSPAcOr` is the App's Client ID. It is NOT used by the current workflow (`release-please.yml` uses the `app-id` input per D-02, wired to the numeric App ID `3982691`). Recorded here only because the plan's step 8 and the runbook's deprecated-input note both flag that a future migration to the `client-id` input requires re-seeding `APP_PRIVATE_KEY`'s sibling secret with this different value — not just renaming the workflow input.
- **PR-create/approve setting: met on live re-check, superseding an initial disabled reading.** During setup the setting was first observed disabled (`can_approve_pull_request_reviews: false`); before this SUMMARY was finalized the maintainer enabled it, and an independent, executor-run `gh api` call confirmed `true` at SUMMARY-authoring time. The acceptance criterion is recorded as met on that basis, not assumed. The `release-please.yml` sequencing fact still stands and is worth preserving: the workflow triggers only on `push: branches: [main]` (no `workflow_dispatch`), so there was no way to smoke-test the App token before a real merge — 09-06's release-PR open remains the first live exercise of the App token end-to-end, regardless of this setting's now-confirmed state.
- **Runbook corrected, not silently left stale.** Per the plan's "if the runbook and reality disagree, fix the runbook" instruction, `docs/RELEASE-PROCEDURES.md` §9 was updated in the two places above rather than leaving it to describe a purpose-built single-repo App and omit the PR-approval setting entirely.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical documentation] §9 never documented the "create and approve pull requests" pre-flight check**
- **Found during:** Task 1 verification
- **Issue:** Plan step 6 and RESEARCH's Open Question 1 both require checking this Actions setting before relying on the App-token pipeline, but 09-04's §9 runbook draft never captured it — only `actions/permissions` (Actions-enabled) was documented, not `actions/permissions/workflow` (PR-create/approve).
- **Fix:** Added the missing pre-flight command and a paragraph explaining the setting, its current confirmed-enabled state, and why D-02's App-token design does not actually depend on it.
- **Files modified:** docs/RELEASE-PROCEDURES.md
- **Verification:** Manual read-through; `git diff --stat` confirms this is the only file touched.

**2. [Rule 2 - Missing Critical documentation] §9 did not anticipate App reuse or the repository_selection verification gap**
- **Found during:** Task 1 verification
- **Issue:** The plan and 09-04's draft both assumed a newly-created, purpose-built App. The maintainer reused an existing one, which changes the correct blast-radius reasoning for T-09-05-02 and surfaces a verification gap (repository_selection unreadable without App auth) the runbook gave no guidance on.
- **Fix:** Added a paragraph to §9 step 3 documenting App reuse as an acceptable shape (contingent on the permission set staying exactly the required three), the blast-radius tradeoff to record when it happens, and the repository_selection verification limitation.
- **Files modified:** docs/RELEASE-PROCEDURES.md
- **Verification:** Manual read-through; matches the reasoning recorded in this SUMMARY's Decisions Made section.

---

**Total deviations:** 2 auto-fixed (both Rule 2 — missing critical documentation), 0 architectural.
**Impact on plan:** Both fixes bring the durable runbook in line with what this checkpoint actually required and actually found; no code, workflow, or secret-handling behavior was touched.

## Issues Encountered

None blocking. One informational note: the PR-create/approve setting was observed disabled during initial setup and enabled before this SUMMARY was finalized — resolved by an independent live re-check rather than assumed from either report.

## User Setup Required

None further for this plan — the one hard human prerequisite (GitHub App creation, installation, and secret storage) is now complete. No executor action remains for App provisioning.

## Next Phase Readiness

- Both `APP_ID`/`APP_PRIVATE_KEY` secrets are in place; `release-please.yml`'s App-token step has what it needs to mint an installation token.
- The App's permission scope is verified correct (exactly Contents/Pull requests/Issues at write, plus mandatory metadata:read).
- Repository Actions configuration is confirmed compatible: Actions enabled, and PR create-and-approve enabled (independently re-verified, not merely assumed).
- No branch protection exists on `main`, so no bypass-actor configuration is required for 09-06's fast-forward merge.
- `release-please.yml` has never fired against a real push (`push: branches: [main]` only, no `workflow_dispatch`) — 09-06/09-07's disposable live proof is the first end-to-end exercise of the full App-token wiring, exactly as §9's "disposable live proof" paragraph anticipates.
- REL-02 remains `[ ]` In Progress — this plan closes only the human-prerequisite checkpoint, not the requirement itself, which 09-06 through 09-08 continue to own.
- No blockers for 09-06.

## Self-Check: PASSED

`docs/RELEASE-PROCEDURES.md` confirmed modified on disk with both corrections present (App-reuse paragraph in step 3, PR-create/approve pre-flight command + note after the pre-flight commands block). `git diff --stat` confirms exactly one file changed, zero workflow/source diff. No secret value appears in this SUMMARY or in the diff.

---
*Phase: 09-release-please-and-goreleaser*
*Completed: 2026-07-29*
</content>
</invoke>
