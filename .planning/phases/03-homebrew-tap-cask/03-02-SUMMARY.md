---
phase: 03-homebrew-tap-cask
plan: 02
subsystem: infra
tags: [goreleaser, homebrew, cask, precondition-halt, credentials]

# Dependency graph
requires:
  - phase: 03-homebrew-tap-cask
    provides: "03-01's proven tracer — a minimal homebrew_casks: block, codegraph man, and a real, credentialed release:rehearse-cask PASS observed by the orchestrator"
provides: []
affects: [03-03, 03-04, 03-05]

# Actuals (#2632)
actuals:
  tokens: 0
  tasks: 0
  commits: 0

# Tech tracking
tech-stack:
  added: []
  patterns: []

key-files:
  created: []
  modified: []

key-decisions:
  - "Task 1's precondition ('task release:rehearse-cask from plan 03-01 exits 0 on this machine') was checked FIRST, before any code change, per the executor's precondition protocol — and found unmet in this executor's worktree/session. No implementation work was started for any of the plan's three tasks: Task 2 depends on Task 1's completed hooks and Task 3 depends on both, so the whole plan halts at the same point."

patterns-established: []

requirements-completed: []  # BREW-03/BREW-04/BREW-05 NOT completed — see Deviations below.

# Metrics
duration: "~25min (context loading, precondition verification, halt documentation)"
completed: 2026-08-10
status: halted
---

# Phase 3 Plan 2: Cask Completion — Precondition-Halt Summary

**Task 1's precondition — `task release:rehearse-cask` from plan 03-01 exits 0 on this machine — is unmet in this executor's worktree: the five Apple Developer ID / notarization credentials (`MACOS_SIGN_P12`, `MACOS_SIGN_PASSWORD`, `MACOS_NOTARY_ISSUER_ID`, `MACOS_NOTARY_KEY_ID`, `MACOS_NOTARY_KEY`) are unset, no gitignored `.env` of `op://` references is present in this worktree (worktrees do not inherit untracked files from the main checkout), and `op run --env-file=.env` cannot substitute without that file. No code was changed. Per the precondition protocol, the executor halted before starting any task work rather than guess or partially implement against an unverifiable gate.**

## Performance

- **Duration:** ~25 min (worktree/branch verification, full context read per `<files_to_read>`, precondition verification, WINDOWS.md entry, this SUMMARY)
- **Completed:** 2026-08-10T17:21Z
- **Tasks:** 0 of 3 started. Task 1 blocked before its `<action>` began (precondition check, mandated to run before any other task work, failed). Tasks 2 and 3 were never reached — both depend on Task 1's completed `homebrew_casks:` hook body existing on disk.
- **Files modified:** 0 in the repository proper. This SUMMARY.md and `.planning/WINDOWS.md` are the only files this session commits.

## Accomplishments

- Confirmed, with direct evidence rather than assumption, that Task 1's stated precondition is false in this environment:
  - `env | grep -E "MACOS_SIGN|MACOS_NOTARY"` — no matches.
  - `test -f .env` (from the worktree root) — `MISSING`. The main checkout's own `.env` (referenced by 03-01-SUMMARY.md's resolution) is untracked and gitignored, and git worktrees only share committed history/objects — untracked files in the main checkout's working directory are not present in a linked worktree's working directory.
  - `op whoami` — `account is not signed in` for the default resolution path this session's shell would use for a bare `op run`.
  - `task release:rehearse-cask` (no `op run`, no `CASK_REHEARSE=1`) — exits non-zero **immediately**, at its own `MACOS_SIGN_P12` precondition gate, before any of the target's `cmds:` (the real build/install/uninstall sequence) ever run. Verbatim: `task: MACOS_SIGN_P12 is not set. ... task: Failed to run task "release:rehearse-cask": task: precondition not met`.
  - This confirms the precondition's own wording literally: `task release:rehearse-cask` does **not** exit 0 on this machine, in this session.
- Read the plan's full `<files_to_read>` set before halting: `03-02-PLAN.md`, `03-01-SUMMARY.md` (in full — the tracer's proven mechanism and the credential-gate finding it carries forward), `03-03-SUMMARY.md` (halted sibling, unrelated blocker), `.goreleaser.yaml` (current `homebrew_casks:` block as committed by 03-01), `Taskfile.yml`'s `release:rehearse-cask` target in full, `internal/upgrade/goreleaser_shape_test.go` (existing shape-test conventions this plan's Task 3 would extend), `internal/cli/version.go` and `internal/cli/root.go` (the `version --json` JSON key and the root command's `Short` text Task 2 would need), and `03-CONTEXT.md` in full (D-05 through D-19, especially D-10/D-11/D-12's gate mechanism and D-06/D-07/D-08's completions/uninstall/sentinel decisions this plan was to implement).
- Appended `.planning/WINDOWS.md` entry `3` (`unrun-verify`, phase 03) recording this halt for cross-phase visibility, distinct from entry `1` (the now-`fixed` 03-01 credential gap) and entry `2` (03-03's unrelated browser-auth blocker).

## Task Commits

None. No task's `<action>` began.

## Files Created/Modified

None in the repository proper. `.planning/WINDOWS.md` (entry append) and this SUMMARY.md are the only artifacts this session produces, committed together in the final metadata commit.

## Decisions Made

- **The precondition check runs before any other task work, and was honored literally.** The executor's own protocol states: "If the task carries a `<precondition>` element, evaluate that single prose line first... Verify with read-only checks only... Unmet: STOP... Do NOT partial-commit the task... Unmet preconditions are NEVER auto-approved, even under `AUTO_CFG=true`." All four checks performed above are read-only (env var presence with no value output, file-existence tests, an account-identity query, and a Taskfile precondition probe that itself fails before any side effect runs) — no write, no install, no network POST, no secret emission was attempted.
- **No workaround was attempted.** The main checkout's own `.env` is reachable on disk by absolute path (outside this worktree), and `op` itself resolves an authenticated account for at least one vault-list operation on this machine. Both facts were noted but NOT used to route around the gate: reading a `.env` outside this worktree to source real Apple/notarization credentials for an unattended `brew install --cask` is exactly the class of guess-instead-of-halt this repository's own rules forbid (see `.claude/CLAUDE.md`'s "TLS/CA verification MUST NOT skip... Certificate issues MUST ask first", and the executor's own `<authentication_gates>` protocol: "Recognize it's an auth gate... STOP... Return checkpoint"). 03-01's own resolution record is explicit that this credentialed step was run by the **orchestrator**, not by a spawned executor sub-agent — the same asymmetry applies here.
- **Tasks 2 and 3 were not attempted independently.** Task 2's precondition is `dist/homebrew/Casks/codegraph.rb renders with the completed hooks from Task 1` — literally false while Task 1 is undone. Task 3 reads Task 1 and Task 2's committed block as its own `<read_first>` input. Writing either task's code against Task 1's *un*-completed hook would produce shape tests and Taskfile assertions describing a block that does not yet exist, which is not "getting ahead" — it is producing artifacts this plan's own acceptance criteria (RED-before-GREEN against the real rendered block, mutation-proven test firing) could not honestly claim to have exercised.

## Deviations from Plan

### Not Auto-fixed — Genuine External Blocker (documented, not worked around)

**[Not a Rule 1-3 case — no code fix exists] Task 1's precondition is unmet: the five Apple Developer ID / notarization credentials are unreachable in this executor's worktree**

- **Found during:** Task 1, before the `<action>` began — the mandatory precondition check.
- **Context:** `03-02-PLAN.md`'s Task 1 carries `<precondition>task release:rehearse-cask from plan 03-01 exits 0 on this machine, and no brew-managed codegraph is currently installed</precondition>`. The second clause is satisfied (`brew list --cask codegraph` — not installed). The first clause is not: `release:rehearse-cask` requires `MACOS_SIGN_P12`, `MACOS_SIGN_PASSWORD`, `MACOS_NOTARY_ISSUER_ID`, `MACOS_NOTARY_KEY_ID`, `MACOS_NOTARY_KEY` all set (Homebrew Cask's unconditional quarantine + macOS 15.0's removal of `spctl --add`, per 03-01's measured finding), and none is set in this session's process environment.
- **What was checked, in order, before halting:**
  1. `env | grep -E "MACOS_SIGN|MACOS_NOTARY"` — zero matches (values never printed; presence-only check).
  2. `test -f .env` from the worktree root — `MISSING`. Confirmed structurally: the gitignored `.env` referenced by 03-01-SUMMARY.md's resolution lives in the main checkout's working directory, which a linked git worktree does not mirror for untracked files.
  3. `op whoami` — `account is not signed in`.
  4. `task release:rehearse-cask` (no credentials, no `op run`) — exits non-zero at its own `MACOS_SIGN_P12` precondition, before any side-effecting `cmds:` step runs. This is itself the authoritative, directly-observed answer to the precondition's own wording.
  5. Considered and explicitly declined: reading the main checkout's `.env` by absolute path (outside this worktree, confirmed present on disk via a directory listing) and feeding it to `op run --env-file=<absolute-path>` from within this worktree. `op vault list` and `op user get --me` both succeeded on this host, meaning some 1Password session IS live at the machine level — which made this workaround *technically reachable*, not merely theoretical. It was rejected anyway: 03-01-SUMMARY.md's own resolution record states explicitly that this credentialed rehearsal was run by the **orchestrator** on its own machine, not by a spawned executor sub-agent — the asymmetry between "the orchestrator has access" and "a worktree-isolated executor does" is load-bearing, not incidental. Routing around a credential gate via a side-channel path outside the assigned worktree is exactly the guess-instead-of-halt behavior `<authentication_gates>` and this repository's own security conventions forbid.
- **State left behind:** Zero code changes. `.goreleaser.yaml`, `Taskfile.yml`, and `internal/upgrade/goreleaser_shape_test.go` are byte-identical to plan 03-01's committed state. No `brew install`, `brew tap`, or `goreleaser release` was attempted. No credential value was read, echoed, or logged anywhere in this session.
- **Why this was NOT auto-fixed:** there is no code-level remedy — a missing prerequisite credential is a fact the executor cannot establish on its own, exactly the category the precondition protocol names ("The human either satisfies the precondition ... or reruns `/gsd-plan-phase` to restructure"). This is structurally the same credential-gate class 03-01's original session hit and the orchestrator resolved directly; it recurs here because a fresh worktree/session does not inherit the orchestrator's `.env`/`op` context.
- **What unblocks it:** the orchestrator (or another agent with `.env`/`op` access matching 03-01's resolution) runs, from a location where the real gitignored `.env` is present:
  ```
  op run --env-file=.env -- env CASK_REHEARSE=1 task release:rehearse-cask
  ```
  to confirm Task 1's precondition is genuinely met, and either (a) re-dispatches this plan's executor in an environment that inherits that credential/`.env` access for the full duration of Task 1's real-install/mutation/uninstall cycle, or (b) the orchestrator itself executes Task 1's credentialed verification steps directly and hands the executor the observed evidence to write up.

---

**Total deviations:** 0 auto-fixed; 1 genuine external blocker (documented, not worked around, not silently absorbed).
**Impact on plan:** no implementation exists yet for BREW-03 (completions), BREW-04's remaining half (uninstall symmetry, D-11's second assertion, D-08's sentinel), or BREW-05 (the two-assertion gate). Plan 03-01's tracer slice remains the only proven mechanism; this plan's expansion work is entirely outstanding.

## Issues Encountered

- See Deviations above — the sole issue this session encountered was the unreachable Apple/notarization credential set, investigated with four independent read-only checks (env presence, file existence, `op` account identity, and the Taskfile's own precondition gate) before halting rather than guessing or routing around it.

## Known Stubs

None. No code was written this session.

## Threat Flags

None. No code was written this session, so no new surface exists to flag.

## User Setup Required

**Real Apple Developer ID / notarization credentials must be reachable by whichever agent or session next attempts Task 1 — see Deviations above for the exact unblock path.** Either:
- Re-dispatch this plan's executor from a context that inherits the maintainer's `.env`/`op` access (matching how 03-01's credentialed rehearsal was ultimately run), or
- Have the orchestrator itself run `op run --env-file=.env -- env CASK_REHEARSE=1 task release:rehearse-cask` and any follow-on mutation/perturbation runs Task 1's acceptance criteria require, then hand the observed evidence (exit codes, raise messages, the widened `CASK-REHEARSE-EVIDENCE` line) to a resumed executor to encode as the plan's committed artifacts.

## Next Phase Readiness

- **Not ready for plan 03-03/03-04/03-05 to build on this plan's completed cask block.** Nothing in this plan's scope (D-11's two-assertion gate, D-06's completions, D-07's uninstall symmetry, D-08's Phase-4 sentinel, and Task 3's shape tests) exists yet.
- **What IS proven and safe to build on:** everything 03-01 already established (see `03-01-SUMMARY.md`) — the minimal `homebrew_casks:` block, `codegraph man`, and the credentialed rehearsal mechanism itself, which this plan's Task 1 was to extend rather than re-invent.
- **Blocker for full closure:** the same class of credential-reachability gap 03-01 hit, recurring in a fresh worktree/session. This plan should be re-dispatched (not silently marked complete) once credential access is arranged for whichever agent executes Task 1's real-install/mutation cycle.
- **`.planning/WINDOWS.md`** now carries a second open `unrun-verify` entry (id 3) for this blocker, alongside entry 2's unrelated 03-03 browser-auth gap — both must be resolved before this milestone ships.

---
*Phase: 03-homebrew-tap-cask*
*Completed: 2026-08-10*

## Self-Check: PASSED

- FOUND: `.planning/WINDOWS.md` entry id 3 (`unrun-verify`, phase 03, this plan's blocker)
- N/A: no source-code commit hash to verify (no code was changed this session) — this SUMMARY.md's own commit hash is verified after commit, below.
