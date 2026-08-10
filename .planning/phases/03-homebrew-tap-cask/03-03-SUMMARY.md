---
phase: 03-homebrew-tap-cask
plan: 03
subsystem: infra
tags: [github-app, github-actions, oidc, homebrew-tap, credentials]

# Dependency graph
requires:
  - phase: 03-homebrew-tap-cask
    provides: "03-01's homebrew_casks: block referencing HOMEBREW_TAP_TOKEN, and the local-tap-wrapping rehearsal pattern"
provides:
  - "seanb4t/homebrew-tap — public repository, README + LICENSE only, default branch main (matches .goreleaser.yaml's homebrew_casks[0].repository.branch)"
  - "A tap-scoped GitHub App (id 4549710), installed on seanb4t/homebrew-tap alone, distinct from the release-please App"
  - "HOMEBREW_TAP_APP_ID / HOMEBREW_TAP_APP_PRIVATE_KEY repository secrets on seanb4t/codegraph-go"
  - "release.yml's tap-token mint (in-job, REDACTED placement) wired to .goreleaser.yaml's homebrew_casks[0].repository.token via HOMEBREW_TAP_TOKEN"
  - "release:goreleaser preconditions asserting the tap token is present and distinct from GITHUB_TOKEN"
  - "TestHomebrewTapTokenScopedToReleaseJob / TestHomebrewTapAppSecretsDistinctFromReleasePleaseAppSecrets (internal/upgrade)"
affects: [03-04, 03-05]

# Actuals (#2632)
actuals:
  tokens: 3800
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "In-job token mint immediately before the consuming step, when a cross-job output would be masked to empty (GitHub Actions log-masking drops rather than passes a masked job output)"

key-files:
  created: []
  modified:
    - .github/workflows/release.yml
    - Taskfile.yml
    - internal/upgrade/release_workflow_shape_test.go
    - .planning/WINDOWS.md

key-decisions:
  - "Task 1's job-output-survival measurement (blocked in the prior halted session because it needed a real tap-scoped App token) was run for real once Task 2's App existed: a scratch branch (probe/tap-token-hop) carried a two-job probe workflow — Job A mints the tap App's token with no id-token: write and exposes it as a job output; Job B consumes it and asserts non-empty/length/differential-repo-access without ever echoing the value. Real run https://github.com/seanb4t/codegraph-go/actions/runs/31417685002 recorded the job A `Complete job` step logging `##[warning]Skip output 'token' since it may contain secret.` and job B receiving TAP_TOKEN as an EMPTY string (`PROBE-EVIDENCE non_empty=false length=0 tap_collaborators_status=000 main_collaborators_status=000`). Verdict: REDACTED — a masked job output does not survive the cross-job hop; GitHub's masking machinery drops it entirely rather than passing the real value through. The scratch branch and its workflow file were deleted immediately after capture (`git push origin --delete probe/tap-token-hop`, `git branch -D probe/tap-token-hop`) and never merged."
  - "Per the REDACTED verdict, release.yml's Task 3 wiring mints HOMEBREW_TAP_TOKEN INSIDE the release job, immediately before the Release step, using the same actions/create-github-app-token pin release-please.yml already carries (bcd2ba49218906704ab6c1aa796996da409d3eb1 # v3.2.0). This is the accepted-and-recorded tradeoff the plan named for the REDACTED branch: one more Action now executes inside the OIDC-bearing job (T-01-29's surface, widened by one step), chosen because the alternative (a separate id-token-free job passing the token by output) was measured not to work, not because it was preferred. Compensating controls: the SHA pin, and the new step-level-only scoping test."
  - "Two Taskfile.yml preconditions were added to release:goreleaser rather than one, per T-03-09: presence alone would pass in the exact silent-degradation case this plan exists to prevent (unset -> template resolves empty -> cask client silently falls back to the release's own GITHUB_TOKEN, which cannot write the tap). A second precondition asserts HOMEBREW_TAP_TOKEN != GITHUB_TOKEN, closing the case where someone sets the two equal 'to make it work' and the push succeeds against the wrong credential with a green log."

patterns-established:
  - "When RESEARCH.md records a cross-job credential-passing question as genuinely open ('neither the action's own README nor the workflow-syntax page states the answer plainly'), measure it on a real, deletable, never-merged scratch-branch workflow run before any production wiring depends on the answer — do not infer from either documentation source alone."

requirements-completed: [BREW-02]

# Metrics
duration: "~50 minutes wall clock, this resumed session (Task 1 measurement through Task 3 completion and WINDOWS cleanup)"
completed: 2026-08-10
status: complete
---

# Phase 3 Plan 3: Homebrew Tap & Credential Summary

**The tap repository, its App-backed write credential, the measured job-output-survival verdict (REDACTED), and `release.yml`'s in-job token mint are all in place and machine-held by new scoping tests — this SUMMARY supersedes the prior halted one, which stopped correctly at a genuine external blocker (no authenticated browser session for GitHub App creation) rather than guessing past it.**

## Performance

- **Duration:** ~50 minutes wall clock for this resumed session (Task 1's remaining measurement, Task 3 in full, WINDOWS cleanup). The prior halted session's duration was not recorded; Task 2 (App creation) was completed and verified by the orchestrator between the two sessions and is not re-timed here.
- **Completed:** 2026-08-10
- **Tasks:** 3 of 3 complete. Task 1 (tap creation + measurement) fully complete across both sessions. Task 2 (App creation, install, secrets) complete — performed and verified by the orchestrator, not this executor; see "Task 2 — Resolution Evidence" below. Task 3 (wiring + tests) complete this session.
- **Files modified this session:** `.github/workflows/release.yml`, `Taskfile.yml`, `internal/upgrade/release_workflow_shape_test.go`, `.planning/WINDOWS.md`.

## Accomplishments

- **Job-output-survival measurement run for real**, on a throwaway scratch branch (`probe/tap-token-hop`, deleted immediately after, never merged): run [31417685002](https://github.com/seanb4t/codegraph-go/actions/runs/31417685002). Job A minted the tap App's token with no `id-token: write` and exposed it as a job output; Job B consumed it without ever echoing the value. Job A's own `Complete job` step logged `##[warning]Skip output 'token' since it may contain secret.`; Job B's env block showed `TAP_TOKEN: ` (empty), and the emitted evidence line reads:
  ```
  PROBE-EVIDENCE non_empty=false length=0 tap_collaborators_status=000 main_collaborators_status=000
  ```
  **Verdict: REDACTED.** GitHub Actions' log-masking machinery drops a job output matching a registered secret mask entirely — it does not pass the real value through to the consuming job.
- **`release.yml` wired per that verdict**: a new "Mint tap-scoped App token" step, using the identical `actions/create-github-app-token@bcd2ba49218906704ab6c1aa796996da409d3eb1 # v3.2.0` pin `release-please.yml` already carries, runs inside the `release` job immediately before the "Release" step. `HOMEBREW_TAP_TOKEN` is fed to that step's own `env:` (never job- or workflow-level), alongside the five existing Apple credential names.
- **Two new `release:goreleaser` preconditions** in `Taskfile.yml`: tap token present, and tap token distinct from `GITHUB_TOKEN` — both demonstrated failing in isolation (see "Precondition Demonstration" below).
- **Two new Go tests** in `internal/upgrade/release_workflow_shape_test.go`: `TestHomebrewTapTokenScopedToReleaseJob` (modelled on `TestAppleSecretsScopedToSingleReleaseJob`, independently re-checking the single `id-token: write` holder rather than delegating) and `TestHomebrewTapAppSecretsDistinctFromReleasePleaseAppSecrets`. Both demonstrated RED against three confirmed-applied mutations, each reverted byte-clean (see "Test RED Demonstrations" below).
- `.planning/WINDOWS.md` entry 2 closed via `gsd-tools windows fixed 2` — `open_count` is now 0 across all three ledger entries.

## Task 2 — Resolution Evidence (completed by the orchestrator, not this executor)

Task 2 (`checkpoint:human-action`) is **complete**. The maintainer created the App; the orchestrator verified it and seeded the secrets. Recorded here verbatim as the checkpoint's resolution evidence, per the resume objective:

**App identity** (`GET /app`, authenticated with a JWT signed by the App's own private key):
```json
{"name":"seanb4t homebrew tap publishing","slug":"seanb4t-homebrew-tap-publishing",
 "id":4549710,"owner":"seanb4t",
 "permissions":{"contents":"write","metadata":"read"},
 "events":[]}
```
`contents: write` is the ONLY granted permission; `metadata: read` is GitHub's mandatory auto-grant. `events: []` confirms the webhook is inactive.

**Installation** (`GET /app/installations`):
```
installation_count=1
id=152719025  account=seanb4t  repository_selection=selected  permissions={"contents":"write","metadata":"read"}
```

**Reachability** with a minted installation token (`GET /installation/repositories`):
```
total_count=1
  seanb4t/homebrew-tap (private=false)
```

**Criterion-5 boundary, proved differentially** — same installation token, same endpoint shape, opposite outcomes:
```
GET /repos/seanb4t/codegraph-go/collaborators     -> HTTP 403
GET /repos/seanb4t/codegraph-go/actions/secrets   -> HTTP 403
GET /repos/seanb4t/codegraph-go/hooks             -> HTTP 403
GET /repos/seanb4t/homebrew-tap/collaborators     -> HTTP 200
```
Methodological note carried forward: the orchestrator's FIRST negative probe was invalid — it read `codegraph-go`'s public README with the App token, got 200, and briefly read that as a scope violation. A control (unauthenticated request -> also HTTP 200) showed the probe was measuring the repo being PUBLIC, not the token's scope. The 403/200 differential above is the corrected, valid proof — this repository's recurring "a check whose result is indistinguishable from its opposite" failure family, caught by adding a control. This session's own probe workflow (above) deliberately used the same collaborators-endpoint differential for the same reason.

**Secrets seeded** as **repository** secrets on `seanb4t/codegraph-go` (verified via `gh secret list --repo seanb4t/codegraph-go`, this session, `repo_secret_count=9`):
```
APP_ID, APP_PRIVATE_KEY                          (pre-existing, release-please)
HOMEBREW_TAP_APP_ID, HOMEBREW_TAP_APP_PRIVATE_KEY (new, this plan)
MACOS_SIGN_P12, MACOS_SIGN_PASSWORD, MACOS_NOTARY_ISSUER_ID, MACOS_NOTARY_KEY_ID, MACOS_NOTARY_KEY
```

**Two Apps are distinct — proved structurally, not by name:** this App's installation list is `[homebrew-tap]` and nothing else, so it is NOT installed on `codegraph-go`. The release-please App behind `APP_ID` must be installed on `codegraph-go` in order to write there. An App installed on a repo the other is not installed on cannot be the same App. `TestHomebrewTapAppSecretsDistinctFromReleasePleaseAppSecrets` (this session) additionally holds the two Apps' SECRET NAMES apart as a machine check, so a future "consolidate the two Apps" edit turns a test red instead of silently failing ROADMAP criterion 5.

The private key came from 1Password (document `t3rawp5moh7pfybhidfj2myr3m`), was used only in a temp file outside the repo, and was shredded. Repo-wide `rg 'BEGIN RSA PRIVATE KEY' --no-ignore --hidden` returns nothing.

## Task Commits

1. **Task 1: Create the tap repository, minimally, and measure whether a minted token survives a job hop** — complete across two sessions. Tap-repository creation (prior session, no in-repo diff, external GitHub state) + this session's job-output-survival measurement (scratch branch, deleted, no lasting in-repo diff — the verdict is recorded in this SUMMARY and in `release.yml`'s mint-step comment).
2. **Task 2: Create and install the tap-scoped GitHub App, and seed its two repository secrets** — complete, performed by the orchestrator between sessions. See "Task 2 — Resolution Evidence" above.
3. **Task 3: Wire the mint into release.yml where the measurement says, and hold the scoping with a test** — complete this session. Commit `f4ca5c9`: `feat(03-03): wire the tap-token mint into release.yml, guarded by scoping tests` — modifies `.github/workflows/release.yml`, `Taskfile.yml`, `internal/upgrade/release_workflow_shape_test.go`.

**Plan metadata:** this commit (SUMMARY.md + WINDOWS.md).

## Files Created/Modified

- `.github/workflows/release.yml` — new "Mint tap-scoped App token" step in the `release` job, immediately before "Release"; `HOMEBREW_TAP_TOKEN` added to the "Release" step's own `env:`.
- `Taskfile.yml` — two new `release:goreleaser` preconditions (tap token present; tap token distinct from `GITHUB_TOKEN`).
- `internal/upgrade/release_workflow_shape_test.go` — `homebrewTapCredentialNames`, `findHomebrewTapCredentialReferences`, `TestHomebrewTapTokenScopedToReleaseJob`, `TestHomebrewTapAppSecretsDistinctFromReleasePleaseAppSecrets`.
- `.planning/WINDOWS.md` — entry 2 marked `fixed` via `gsd-tools windows fixed 2`.
- (External, no in-repo diff, from the prior session) `seanb4t/homebrew-tap` — public, `LICENSE` + `README.md` only, default branch `main`.
- (External, no in-repo diff, from the orchestrator's Task 2) The tap-scoped GitHub App (id 4549710), its installation on `homebrew-tap` alone, and the two repository secrets on `codegraph-go`.

## Precondition Demonstration

`task release:goreleaser`'s full precondition chain cannot isolate the two new checks directly on this machine (an earlier precondition — no tag at `HEAD` — halts first regardless of `HOMEBREW_TAP_TOKEN`). The two new `sh:` checks were therefore run directly, verbatim as written in `Taskfile.yml`, to demonstrate both halt states and the pass state:

```
$ unset HOMEBREW_TAP_TOKEN
$ sh -c '[ -n "${HOMEBREW_TAP_TOKEN:-}" ]'; echo "exit=$?"
exit=1   # HALTS: "HOMEBREW_TAP_TOKEN is not set. ..."

$ export HOMEBREW_TAP_TOKEN="abc123" GITHUB_TOKEN="abc123"
$ sh -c '[ -n "${HOMEBREW_TAP_TOKEN:-}" ]'; echo "exit=$?"
exit=0
$ sh -c '[ "${HOMEBREW_TAP_TOKEN:-}" != "${GITHUB_TOKEN:-}" ]'; echo "exit=$?"
exit=1   # HALTS: "HOMEBREW_TAP_TOKEN is byte-identical to GITHUB_TOKEN. ..."

$ export HOMEBREW_TAP_TOKEN="distinct-value"   # GITHUB_TOKEN still abc123
$ sh -c '[ -n "${HOMEBREW_TAP_TOKEN:-}" ]'; echo "exit=$?"
exit=0
$ sh -c '[ "${HOMEBREW_TAP_TOKEN:-}" != "${GITHUB_TOKEN:-}" ]'; echo "exit=$?"
exit=0   # PASSES
```

## Test RED Demonstrations

Three mutations applied directly to `.github/workflows/release.yml`, each run against `go test ./internal/upgrade/...`, each reverted with `git checkout -- .github/workflows/release.yml` (confirmed byte-clean via `git diff` before re-applying the real Task 3 edits, which that revert also discarded and required re-applying — see Deviations):

1. **Second job referencing `HOMEBREW_TAP_APP_ID`** (a throwaway `scratch-mutation-1` job with `env: FOO: ${{ secrets.HOMEBREW_TAP_APP_ID }}`):
   ```
   release_workflow_shape_test.go:1510: release.yml references a Homebrew tap credential name in job "scratch-mutation-1", want only the release job "release" (scope=step step="leak")
   --- FAIL: TestHomebrewTapTokenScopedToReleaseJob
   ```
2. **Job-level `env:` carrying `HOMEBREW_TAP_TOKEN`** (added `env: HOMEBREW_TAP_TOKEN: mutation-job-level-env` directly under the `release` job):
   ```
   release_workflow_shape_test.go:1513: release.yml references a Homebrew tap credential name at job-level env: (job=release step="") — must be step-level only
   --- FAIL: TestHomebrewTapTokenScopedToReleaseJob
   ```
3. **Second job declaring `id-token: write`** (a throwaway `scratch-mutation-3` job with `permissions: id-token: write`):
   ```
   release_workflow_shape_test.go:643: id-token: write held by unexpected job(s) [scratch-mutation-3] — only the goreleaser job ("release") may hold it, with no allowance (D-11)
   --- FAIL: TestOIDCWriteScopedToSingleGoreleaserJob
   release_workflow_shape_test.go:1282: release.yml declares id-token: write in 2 job(s) [release scratch-mutation-3], want exactly 1
   --- FAIL: TestAppleSecretsScopedToSingleReleaseJob
   release_workflow_shape_test.go:1535: release.yml declares id-token: write in 2 job(s) [release scratch-mutation-3], want exactly 1 (independent re-check, not delegated to TestOIDCWriteScopedToSingleGoreleaserJob)
   --- FAIL: TestHomebrewTapTokenScopedToReleaseJob
   ```
   This third mutation also confirmed the two pre-existing tests (`TestOIDCWriteScopedToSingleGoreleaserJob`, `TestAppleSecretsScopedToSingleReleaseJob`) still correctly detect a second `id-token: write` holder — this change did not weaken them.

## Decisions Made

See `key-decisions` in the frontmatter above (measurement methodology, REDACTED placement, and the two-precondition rationale).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - blocking issue] Probe workflow's job name broke YAML parsing**
- **Found during:** first probe workflow run (`31417642516`), which failed as a "workflow file issue" before any step executed.
- **Issue:** `name: mint tap-scoped App token (no id-token: write)` is unquoted and contains a bare `: ` inside parentheses, which `actionlint`/GitHub's YAML parser reads as a second mapping key rather than plain text (`mapping values are not allowed in this context`).
- **Fix:** quoted the string (`name: "mint tap-scoped App token (no id-token: write)"`). Caught locally via `task lint:actions` before consuming a second real Action-minutes budget guessing further; re-pushed and the run succeeded.
- **Files modified:** `.github/workflows/_probe-tap-token-hop.yml` (scratch branch only, deleted with the branch — no lasting diff).
- **Commit:** none in the surviving history (scratch branch `probe/tap-token-hop`, deleted).

**2. [Rule 1 - bug, self-inflicted] `git checkout -- .github/workflows/release.yml` used to revert a test mutation also discarded this task's own uncommitted Task 3 edits**
- **Found during:** cleanup after the third RED-demonstration mutation, while confirming the file was byte-clean.
- **Issue:** the real Task 3 wiring (the mint step + `HOMEBREW_TAP_TOKEN` env entry) had not yet been committed when the mutation-revert `git checkout -- .github/workflows/release.yml` ran. `git checkout --` restores the file to `HEAD`, which at that point had neither the mutation NOR the legitimate edits — both were discarded together.
- **Fix:** re-applied the two legitimate edits (the mint step, the `HOMEBREW_TAP_TOKEN` env line) via `Edit`, verbatim to what had been present before the revert, then re-ran `task lint:actions` and the full test suite to confirm no silent divergence from the version that had been tested moments earlier.
- **Files modified:** `.github/workflows/release.yml`.
- **Commit:** folded into `f4ca5c9` (the edits were never separately committed before the accidental revert, so there is no lost commit to recover — only the risk of a lost *uncommitted* edit, caught immediately by the next verification step rather than discovered later).
- **Lesson for future RED-demonstration mutations in this repository:** commit legitimate edits BEFORE applying and reverting a test-mutation on the same file, or use a scratch copy for the mutation instead of editing the real file in place.

### Preserved from the Prior Halted Session (not re-litigated, not erased)

The prior `03-03-SUMMARY.md` this document supersedes recorded a genuine external blocker: Task 2's GitHub App creation requires driving GitHub's web UI, and no authenticated `agent-browser` session was reachable in that session's environment. That executor performed six independent checks (session restore, session list, auth-profile list, `agent-browser doctor`, a CDP-port scan against the user's real running Chrome, and an explicit, reasoned refusal to point automation at the user's live daily-driver Chrome profile) before halting and returning a checkpoint rather than guessing past the gate. **That halt was the correct call, not a failure to route around.** Task 2 requires a human-only action (App creation, private-key generation, and installation-repository selection are GitHub UI actions with no CLI/API equivalent available to an unattended agent); the orchestrator's authorization for that session was explicit that an unreachable authenticated browser session is a legitimate stopping point (`<authentication_gates>` class), not something to substitute a workaround for. This session picks up exactly where that halt left off, with Task 2 now completed by the maintainer and verified by the orchestrator — see "Task 2 — Resolution Evidence" above.

---

**Total deviations this session:** 2 auto-fixed (both Rule 1/3, both self-contained and corrected before proceeding); 0 genuine external blockers (the one blocker from the prior session is resolved, not repeated).
**Impact on plan:** none of the above deviations changed the plan's design or required an architectural decision (Rule 4). Both were caught and corrected within the same task before any downstream step depended on the broken state.

## Issues Encountered

None beyond the two auto-fixed deviations above.

## Known Stubs

None. `.goreleaser.yaml`'s `homebrew_casks[0].repository.token` template now resolves to a real, working credential path end-to-end (mint step -> step env -> Taskfile precondition -> GoReleaser template) rather than the placeholder wiring the prior halted session left behind.

## Threat Flags

None beyond what the plan's own `<threat_model>` already covers (T-03-07 through T-03-10). The in-job mint (REDACTED branch) is the surface T-03-10 already named as the accepted-and-recorded tradeoff for this exact scenario; no new, unnamed surface was introduced.

## User Setup Required

None remaining. The App exists, is installed on `homebrew-tap` alone, its two secrets are seeded as repository secrets, and `release.yml`/`Taskfile.yml`/the shape test all consume them correctly per this session's verification.

## Next Phase Readiness

- **Ready for plan 03-04 and 03-05 to build on this plan's credential mechanism.** The tap repository, the App, its installation scope, the two repository secrets, the measured REDACTED placement verdict, `release.yml`'s wiring, the two Taskfile preconditions, and the two new scoping tests are all in place and verified.
- `.planning/WINDOWS.md` now carries `open_count: 0` (entries 1 and 3 were already `fixed`; entry 2 is closed by this session).
- `task lint:actions` and `task test:unit` both exit 0 on the current `HEAD`.

---
*Phase: 03-homebrew-tap-cask*
*Completed: 2026-08-10*

## Self-Check: PASSED

- FOUND: `.github/workflows/release.yml` contains `HOMEBREW_TAP_TOKEN` and the "Mint tap-scoped App token" step (verified via `rg -n "HOMEBREW_TAP_TOKEN|tap-app-token" .github/workflows/release.yml` after the re-apply).
- FOUND: `Taskfile.yml` contains both new `release:goreleaser` preconditions (verified via direct read after edit).
- FOUND: `internal/upgrade/release_workflow_shape_test.go` contains `TestHomebrewTapTokenScopedToReleaseJob` and `TestHomebrewTapAppSecretsDistinctFromReleasePleaseAppSecrets`, both passing (`go test ./internal/upgrade/... -run 'TestHomebrewTap...' -v` — PASS).
- FOUND: commit `f4ca5c9` in `git log --oneline -3` on `gsd/v0.5.0-macos-distribution-homebrew`.
- FOUND: `.planning/WINDOWS.md` `open_count: 0` after `gsd-tools windows fixed 2`.
- FOUND: scratch branch `probe/tap-token-hop` absent from both `git branch` and `git branch -r` after cleanup.
- CONFIRMED: `task lint:actions` exits 0; `task test:unit` exits 0 (both re-run after the Deviation-2 re-apply, not before it).
