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
affects: [03-04, 03-05]

# Actuals (#2632)
actuals:
  tokens: 900
  tasks: 0
  commits: 1

# Tech tracking
tech-stack:
  added: []
  patterns: []

key-files:
  created: []
  modified: []

key-decisions:
  - "Task 1's tap-repository creation was executed and verified (real, external GitHub state), but Task 1's job-output-survival measurement — and therefore Task 2 and Task 3 — could not proceed: the measurement's own acceptance criteria require a real, tap-scoped GitHub App token (an authenticated read must succeed against homebrew-tap and fail against codegraph-go), and that App does not exist until Task 2 creates it. Task 2 requires browser automation the orchestrator explicitly authorized, but no authenticated browser session was reachable in this environment (see Deviations)."

patterns-established: []

requirements-completed: []  # BREW-02 NOT completed — see Deviations/Issues below.

# Metrics
duration: not recorded (setup/precondition investigation dominated; no wall-clock instrumentation captured at task start)
completed: 2026-08-10
status: halted
---

# Phase 3 Plan 3: Homebrew Tap & Credential Summary

**The tap repository `seanb4t/homebrew-tap` is created, minimal, and verified (public, README + LICENSE only, default branch `main`); the GitHub App creation and job-output-survival measurement that depend on browser automation are halted because no authenticated browser session was reachable in this environment.**

## Performance

- **Duration:** not recorded (no start timestamp captured before the worktree/precondition steps; the session was dominated by context loading, tap creation, and the browser-authentication investigation below)
- **Completed:** 2026-08-10T17:03:09Z
- **Tasks:** 0 of 3 fully complete. Task 1 partially complete (tap repository created and verified; job-output-survival measurement blocked). Task 2 blocked before any App-creation step. Task 3 not started (depends on both).
- **Files modified:** 0 in `codegraph-go` (the tap repository is an external resource with no in-repo diff); this SUMMARY.md is the only file this session commits.

## Accomplishments

- `seanb4t/homebrew-tap` created via `gh repo create --public`, seeded with exactly `LICENSE` (copied verbatim from this repository's own MIT license) and `README.md` (links to `seanb4t/codegraph-go` for install instructions, carries no `brew tap`/`brew install` command line itself — per D-15, so there is exactly one place the install command can drift).
- Verified via `gh repo view seanb4t/homebrew-tap --json name,visibility,defaultBranchRef,isPrivate`: `visibility: PUBLIC`, `defaultBranchRef.name: main`.
- Verified via `gh api repos/seanb4t/homebrew-tap/contents`: exactly two entries, `LICENSE` and `README.md`. No `Casks/` directory exists (GoReleaser will create it on first release, per D-15).
- Confirmed `.goreleaser.yaml`'s existing `homebrew_casks[0].repository.branch: main` (set in plan 03-01) already matches the tap's real default branch — no fix needed.

## Task Commits

1. **Task 1: Create the tap repository, minimally, and measure whether a minted token survives a job hop** — **partially complete, not committed as done.** The tap-repository creation and verification is real, external GitHub state (no in-repo file diff to commit — see Files Created/Modified). The job-output-survival measurement half of this task (the scratch-branch probe workflow) could not run: measuring it correctly requires a real, tap-scoped GitHub App token so the probe's own acceptance criterion ("a single authenticated read against the tap repository ... succeeds while the same read against `seanb4t/codegraph-go` does not") is meaningful — and that App does not exist until Task 2 creates it. See Deviations.
2. **Task 2: Create and install the tap-scoped GitHub App, and seed its two repository secrets** — **not started.** Blocked before any App-creation step: no authenticated browser session was reachable. See Deviations.
3. **Task 3: Wire the mint into release.yml where the measurement says, and hold the scoping with a test** — **not started.** Depends on Task 1's SURVIVES/REDACTED verdict (Task 1 incomplete) and Task 2's two repository secrets (Task 2 not started).

**Plan metadata:** this commit (SUMMARY.md + WINDOWS.md)

## Files Created/Modified

None in `codegraph-go`. The tap repository (`seanb4t/homebrew-tap`) is an external GitHub resource created via `gh repo create` and seeded via a throwaway local git clone in the scratchpad directory (never inside this worktree) — it has no representation in this repository's git history.

## Decisions Made

- **Sequencing note (not a plan deviation, a task-ordering observation):** Task 1's own action text requires minting a token "scoped by owner and to the tap repository only" for the job-output-survival measurement. No App is scoped to the tap repository until Task 2 creates and installs one. The plan's Task 1/Task 2/Task 3 numbering therefore encodes a real dependency in the reverse of its textual order for this sub-step: Task 1's measurement cannot produce a trustworthy SURVIVES/REDACTED verdict without Task 2's App already existing. This was recognized before attempting the probe (rather than fabricating a probe against the wrong App, which would have produced a wrong or meaningless verdict) — see Deviations for why Task 2 itself could not be completed to unblock this.

## Deviations from Plan

### Not Auto-fixed — Genuine External Blocker (documented, not worked around)

**[Not a Rule 1-3 case — no code fix exists] Task 2's GitHub App creation could not proceed: no authenticated browser session was reachable**

- **Found during:** Task 2, before any App-creation step in the GitHub UI.
- **Context:** The orchestrator's authorization for this plan explicitly permits driving GitHub's web UI via the `agent-browser` skill to create the GitHub App myself, "The maintainer's browser session is expected to be authenticated to github.com as `seanb4t`." Requirement 5 of that authorization is explicit: "HALT rather than guess... If the browser is not authenticated... STOP, leave a clear record of what state the App is in (created? installed? key generated?), and return a checkpoint."
- **What was checked, in order, before halting:**
  1. `agent-browser --session <worktree-scoped> --restore open https://github.com/settings/apps` — landed on `https://github.com/login?return_to=...`, not the Apps settings page. Not authenticated.
  2. `agent-browser session list` — showed only the session(s) this investigation itself created (a `default` session that predated this task, and the worktree-scoped one). Neither carried GitHub auth.
  3. Closed both and retried with the plain `default` session and `--restore` — same result: redirected to `github.com/login`.
  4. `agent-browser auth list` — "No auth profiles saved". No vault-stored GitHub credential exists for agent-browser to use.
  5. `agent-browser doctor` — clean environment (Chrome for Testing present, no active daemons, "2 saved state file(s)" — inspected and confirmed these were only the two just-created session config files from steps 1-3, not a pre-existing authenticated profile).
  6. `ps aux | grep -i chrome` — the user's real, already-running Google Chrome (multiple long-lived renderer processes, oldest since "Sun09AM") has **no `--remote-debugging-port`** flag, so `agent-browser --auto-connect`/`--cdp` cannot attach to it. Confirmed via `lsof -iTCP -sTCP:LISTEN` that no CDP port (922x or otherwise associated with Chrome) is listening.
  7. Considered, and explicitly declined, pointing `agent-browser --profile` at the user's real Chrome profile directory (`~/Library/Application Support/Google/Chrome`, found via `doctor`'s "2 Chrome profile(s)" line): the user's real Chrome is currently running against that same profile directory, so a second automation-driven Chrome instance opening it concurrently risks a profile-lock conflict or, if it somehow succeeded, would hand automated control to every other logged-in site in the user's actual daily-driver browser — a far larger blast radius than "authenticate to github.com as seanb4t," and not something the authorization's "expected to be authenticated" framing consented to. This is exactly the kind of guess Requirement 5 forbids trading for progress.
- **State left behind:** No GitHub App was created. No private key was generated or downloaded. Nothing was installed on any repository. No PEM, App ID, or App secret exists anywhere in this worktree, in shell history, or in any log — none of the App-creation steps that would produce such material were reached. All `agent-browser` sessions opened during this investigation were closed (`agent-browser close --all`) before this task ended; `agent-browser session list` confirms zero active sessions.
- **Why this was NOT auto-fixed:** there is no code-level or configuration-level remedy. Authenticating a browser to a specific human's GitHub account is either a credential the agent does not and should not possess, or requires the human's own interactive login — the exact class of gate `<authentication_gates>` names, and requirement 5's own text anticipates this outcome as a legitimate, expected stopping point rather than a failure to route around.
- **What unblocks it:** either (a) the maintainer authenticates an `agent-browser` session reachable by this worktree — e.g. launching their real Chrome with `--remote-debugging-port=9222` and re-running with `agent-browser --cdp 9222 ...` or `--auto-connect`, or logging into a fresh headed `agent-browser` window and saving that session — after which this plan's Task 2 (and the Task 1 measurement and Task 3 it unblocks) can be re-attempted under the same authorization; or (b) the maintainer creates the App manually, following Task 2's own `<instructions>` in `03-03-PLAN.md`, and records the two repository secrets and the App-installation repository-access list per Task 2's `<verification>`, after which a re-dispatched executor can resume at Task 1's measurement and Task 3.

---

**Total deviations:** 0 auto-fixed; 1 genuine external blocker (documented, not worked around, not silently absorbed).
**Impact on plan:** Task 1's tap-repository creation is real, correct, and independently verified — safe for later plans to build on. Task 1's job-output-survival measurement, Task 2's App creation, and Task 3's wiring/tests are all blocked on the same root cause (no reachable authenticated browser session) and remain undone.

## Issues Encountered

- See Deviations above — the sole issue this session encountered was the unreachable authenticated browser session, investigated exhaustively (six independent checks) before halting rather than guessing.

## Known Stubs

None. No code was written this session (the tap repository is real, minimal, external state per D-15 — not a stub standing in for missing functionality).

## Threat Flags

None beyond what the plan's own `<threat_model>` already covers. No code was written this session, so no new surface exists to flag.

## User Setup Required

**External credentials require manual configuration to complete this plan — see Deviations above for the two unblock paths.** Either:
- Authenticate an `agent-browser` session reachable by this worktree and re-dispatch this plan under the same orchestrator authorization, or
- Manually perform Task 2's GitHub App creation, installation, and secret-seeding steps (`03-03-PLAN.md` Task 2's `<instructions>`), then re-dispatch to resume at Task 1's job-output-survival measurement.

## Next Phase Readiness

- **Not ready for plan 03-04 or 03-05 to build on this plan's credential mechanism.** Only the tap repository itself exists; the GitHub App, its installation, its two repository secrets, the job-output-survival verdict, and `release.yml`'s tap-token mint are all still outstanding.
- **What IS proven and safe to build on:** `seanb4t/homebrew-tap` exists, is public, contains exactly `LICENSE` and `README.md`, has default branch `main` matching `.goreleaser.yaml`'s existing `repository.branch: main`, and carries no install-command text of its own (D-15).
- **Blocker for full closure:** the browser-authentication gap above. This plan should be re-dispatched (not silently marked complete) once either unblock path is taken.
- **`.planning/WINDOWS.md`** now carries one open `unrun-verify` entry for this blocker — it must be resolved (not silently dropped) before this milestone ships.

---
*Phase: 03-homebrew-tap-cask*
*Completed: 2026-08-10*

## Self-Check: PASSED

- FOUND: `seanb4t/homebrew-tap` (verified via `gh repo view seanb4t/homebrew-tap --json name,visibility,defaultBranchRef,isPrivate`)
- FOUND: exactly `LICENSE` and `README.md` in the tap repo (verified via `gh api repos/seanb4t/homebrew-tap/contents`)
- N/A: no commit hash to verify in `codegraph-go` for Task 1 (external resource, no in-repo diff) — this SUMMARY.md's own commit hash is verified after commit, below.
