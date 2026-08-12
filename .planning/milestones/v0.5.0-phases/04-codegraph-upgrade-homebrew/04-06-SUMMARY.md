---
phase: 04-codegraph-upgrade-homebrew
plan: 06
subsystem: acceptance-evidence
tags: [homebrew, acceptance, evidence, upgrade]
dependency-graph:
  requires:
    - detectBrewManaged (plan 04-01)
    - brewPointerMessage (plan 04-01)
    - TestDetectBrewManaged_RealInstall (plan 04-01)
    - seanb4t/tap published cask (Phase 3)
  provides:
    - 04-EVIDENCE.md
    - UPGR-ACCEPTANCE-EVIDENCE line
  affects: []
tech-stack:
  added: []
  patterns:
    - "Single trapped shell script (arm EXIT trap before the first mutating byte) as the only mechanism that can make 'every mutation is covered by restoration' literally true across brew tap/trust/install and a Homebrew-owned payload substitution"
    - "RUN_ID binding across three fixed-path artifacts (baseline/harness-log/receipt) so a stale artifact from a prior run cannot stand in for the current one"
    - "Bidirectional baseline restoration (TAP_PREEXISTING/TAP_TRUSTED_BEFORE recorded before mutating, TAP_ACTION/TRUST_ACTION restoring to exactly that recorded state, never to an unconditionally-clean state)"
key-files:
  created:
    - .planning/phases/04-codegraph-upgrade-homebrew/04-EVIDENCE.md
  modified: []
decisions:
  - "Ran the plan's Task 1 trapped script for real against the maintainer's live machine and the real seanb4t/tap — no rehearsal tap, no dry run — per the plan's own explicit prohibition on substituting a local tap"
  - "Leg 3 (a released binary that itself carries this phase's code, installed fully naturally) was named as an unexecuted, accepted gap with its closing condition rather than approximated or implied — v0.8.0 predates this phase's code in any tagged release"
metrics:
  duration: "~20 minutes"
  completed: 2026-08-11
status: complete
actuals:
  tokens: 8002
  tasks: 2
  commits: 1
---

# Phase 4 Plan 06: Real-tap acceptance evidence Summary

Ran the plan's single trapped script against the maintainer's real Homebrew and the real,
published `seanb4t/tap`: tapped, trusted (already-trusted, pre-existing grant left alone),
installed `codegraph 0.8.0`, ran `TestDetectBrewManaged_RealInstall` against the genuine
Caskroom tree (PASS), substituted the Caskroom payload with a locally built binary, observed
`upgrade` / `upgrade --check` / `upgrade --force` / `upgrade --help` through the installed
`bin/codegraph` symlink, and let the EXIT trap restore the machine — verified by an equal
before/after payload sha256 pair, `RESTORE_VERDICT=ok`, `RESTORE_INVOCATIONS=1`,
`TAP_ACTION=untapped` (the tap was not there before this run) and `TRUST_ACTION=left-trusted`
(the trust grant WAS there before this run, from a prior maintainer session, and was left
alone). `04-EVIDENCE.md` records three legs of distinct evidentiary status, maps all four
amended ROADMAP Phase-4 success criteria, and dispositions the three carried spec-less-probe
assumptions from `04-01-PLAN.md`.

## What Was Built

- **`.planning/phases/04-codegraph-upgrade-homebrew/04-EVIDENCE.md`** (new): the sole artifact
  this plan produces. Structured around three legs —
  - **Leg 1** (genuine layout, no mutation, executed): real `brew tap seanb4t/tap` +
    `brew trust --tap seanb4t/tap` (already-trusted) + `brew install codegraph`, the real
    Caskroom listing and `INSTALL_RECEIPT.json` placement recorded verbatim, and
    `TestDetectBrewManaged_RealInstall`'s PASS read out of the run's own fixed-path harness log
    (re-running it after the task is impossible by construction — the cask is uninstalled by
    then), with every logged field cross-checked against the on-disk listings.
  - **Leg 2** (genuine layout, substituted payload, executed, named as a substitution in its
    own heading): the Caskroom payload overwritten with a binary built from this worktree,
    preserved-then-restored bytes proven equal by sha256, and all four `upgrade` observations
    captured through the installed symlink — bare `upgrade` exits 1 with the pointer message,
    `--check` exits 0 with the same pointer and no manufactured version number, `--force`'s
    output is byte-identical to bare `upgrade`'s, `--help` names both exit behaviours.
  - **Leg 3** (fully natural path, NOT executed, NOT claimed): named with its closing
    condition — the next release cut after this phase's plans merge republishes the tap cask
    from a tag carrying this phase's code, at which point a natural `brew install`/`brew
    upgrade` exercises this exact path with nothing substituted.
  - A criterion-mapping section citing all four amended Phase-4 success criteria to the leg
    (or the unit-level proof from plan 04-01) that evidences each.
  - A `## Carried assumption dispositions` table closing UPGR-01 and UPGR-03 outright and
    partially closing UPGR-02 (the genuine-tree half closed by Leg 1; the relative-symlink
    chain / bind-mount / case-insensitive-filesystem edges explicitly carried forward
    unclosed, since none of those three shapes occurred in this real install).
  - One `UPGR-ACCEPTANCE-EVIDENCE` machine-parseable line with brew version, cask version,
    install directory, all three exit codes, both payload hashes, and the restoration verdict.

## Verification

- Task 1's full `<automated>` gate (harness-log PASS pattern, `TRAP_ARMED=1`,
  `RESTORE_VERDICT=ok`, `RESTORE_INVOCATIONS=1`, per-key payload-hash cardinality legs plus
  distinctness, the `node -e` RUN_ID/TAP_ACTION/TRUST_ACTION binding check, and the two
  machine re-probes for tap and trust agreement) — ran verbatim against the real receipt,
  baseline and harness-log files produced by the run: `GATE_PASS`.
- Task 2's full `<automated>` gate (evidence-line presence, `Leg 1`/`Leg 2`/`Leg 3` tokens,
  `UPGR-0N` requirement-ID presence and disposition-table-row-shape checks, `Criterion 1`
  through `Criterion 4` presence) — ran verbatim against the written `04-EVIDENCE.md`:
  `GATE2_PASS`.
- `task test` (unit, golden, integration, wireoracle, daemon, race) — exit 0, all packages
  green; this plan's script never touches the repository, only Homebrew-owned paths and a
  scratch directory outside it.
- `git status --porcelain` — clean before the commit except the new `04-EVIDENCE.md`.
- `git diff --exit-code go.mod go.sum` — exit 0, no dependency added (T-04-SC).
- `git diff --diff-filter=D --name-only HEAD~1 HEAD` — empty; no unintended deletions in the
  commit.

## Deviations from Plan

None — plan executed exactly as written, including the six-step (nine-numbered-step) trapped
script structure, the bidirectional tap/trust restoration, the idempotent EXIT-trap-with-guard,
and the `TRAP_ARMED=1` marker positioned exactly between trap registration and `brew tap`.

One thing worth recording plainly rather than as a deviation: this machine's real state at run
time was the "trap laid" combination the orchestrator's pre-run baseline described —
`TAP_PREEXISTING=no` (the tap was not tapped before this run) paired with
`TAP_TRUSTED_BEFORE=yes` (the trust grant already existed from an earlier maintainer session,
independent of tap presence). The trap correctly produced the asymmetric restoration this
combination requires: `TAP_ACTION=untapped`, `TRUST_ACTION=left-trusted` — untapping what this
run added while leaving alone a trust grant this run did not make.

## Known Stubs

None.

## Threat Flags

None — this plan introduces no new network endpoint, auth path, file-access pattern, or
schema change at a trust boundary beyond what `<threat_model>` in `04-06-PLAN.md` already
registers (T-04-19 through T-04-22, T-04-SC), and no code was changed by this plan.

## Self-Check: PASSED

- `.planning/phases/04-codegraph-upgrade-homebrew/04-EVIDENCE.md` — FOUND
- Commit `14d2853` — FOUND in `git log --oneline --all`
- `git status --porcelain` after commit — clean (only this plan's own SUMMARY.md pending, to
  be committed by the final metadata commit)
