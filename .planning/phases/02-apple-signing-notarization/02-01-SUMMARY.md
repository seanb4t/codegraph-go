---
phase: 02-apple-signing-notarization
plan: 01
subsystem: infra
tags: [goreleaser, gatekeeper, spctl, taskfile, macos, code-signing, go-yaml]

# Dependency graph
requires:
  - phase: 01-cross-compile-spike-goreleaser-release-migration
    provides: v0.5.1's published, deliberately un-notarized darwin/arm64 and darwin/amd64 release assets — this plan's RED baseline subject
provides:
  - "task verify:gatekeeper" — the single Taskfile target both the SIGN-03 RED baseline (this plan) and the post-release GREEN check (plan 02-06) call, driven by GATEKEEPER_EXPECT
  - "The GATEKEEPER-EVIDENCE schema=1 evidence-line convention (phase-wide, defined here, consumed by later plans' evidence files)"
  - ".planning/phases/02-apple-signing-notarization/02-EVIDENCE.md — the phase's recorded-observation file, referenced rather than duplicated by later plans"
  - "D-19's oracle (spctl -a -vv -t install, exit-status-only verdict) reproduced independently against two vendors (docker, codex) on the maintainer's own machine"
  - "Assumption A2 settled CONFIRMED: the synthetic com.apple.quarantine rig produces the same spctl verdict as a genuine browser download"
affects: [02-02, 02-03, 02-04, 02-05, 02-06, 02-07]

# Actuals (#2632)
actuals:
  tokens: 10900
  tasks: 3
  commits: 4

tech-stack:
  added: []
  patterns:
    - "GATEKEEPER_EXPECT-bound digest asymmetry: a missing GitHub per-asset digest hard-fails on the GREEN path (expect=accepted) but sentinel-and-warns on the RED path (expect=rejected) — sentinel-and-continue on a verification step is explicitly rejected as a default, only permitted where the claim being made does not carry provenance weight"
    - "Verdict derivation from exit status alone, never a substring search of tool output — a second, independent output-text assertion (source_assertion) is captured separately under its own mechanical truth table, so the two failure kinds stay distinguishable"
    - "Non-gating recorded observation (syspolicy_check) — capture and report a tool's verdict without ever letting it gate, when its failure mode is definitionally unreachable given a locked-in scope decision (DIST-06/D-16)"
    - "Real go.yaml.in/yaml/v3 struct decode for Taskfile.yml precondition-shape guards, as an alternative to the file's existing regex-based parseTaskBlocks idiom — proven to decode this repo's real Taskfile.yml cleanly"

key-files:
  created:
    - .planning/phases/02-apple-signing-notarization/02-EVIDENCE.md
  modified:
    - Taskfile.yml
    - internal/upgrade/taskfile_shape_test.go
    - docs/RELEASE-PROCEDURES.md

key-decisions:
  - "Task 1's precondition-guard test decodes Taskfile.yml with a real YAML decoder (gatekeeperTaskfileRoot/gatekeeperTaskYAML structs) rather than the file's established parseTaskBlocks regex-scanner idiom, per the plan's explicit and repeated instruction; verified live that go.yaml.in/yaml/v3 parses this repo's actual Taskfile.yml (32 tasks, all preconditions) without incident before committing to the approach."
  - "Assumption A2 (synthetic vs genuine quarantine xattr) settled CONFIRMED at the Task 2 checkpoint on real hardware: a byte-identical Safari-downloaded copy of codegraph_v0.5.1_darwin_arm64 produced the same rejected/exit-3 spctl verdict as the SYNTHETIC-xattr rig, despite the quarantine flag byte (0081 vs 0083), timestamp, and UUID all differing, and despite the genuine copy carrying an extra com.apple.metadata:kMDItemWhereFroms attribute the synthetic rig does not write. No Taskfile.yml rig changes were made as a result — none were needed."
  - "D-19's oracle (spctl -a -vv -t install, verdict from exit status alone) was independently reproduced against TWO vendors (docker, codex) rather than resting on the prior single measurement recorded in 02-RESEARCH.md."

patterns-established:
  - "Evidence-line schema=1 convention (phase-wide): fixed key order, schema=1 always first, not-found/unknown sentinels for anything unobtainable, no bare string classification of tool output."

requirements-completed: [SIGN-02, SIGN-03]

coverage:
  - id: D1
    description: "verify:gatekeeper Taskfile target: 12 named preconditions (TAG/REPO/GOOS/GOARCH/GH_TOKEN/GATEKEEPER_EXPECT presence+value-validation/gh/jq/xattr/spctl/darwin-host), D-19 oracle (spctl -a -vv -t install, verdict from exit status alone, never -t exec, never a substring search), GATEKEEPER_EXPECT-bound digest asymmetry, SYNTHETIC quarantine xattr with hard-failing readback confirmation, non-gating syspolicy_check recording, and the GATEKEEPER-EVIDENCE schema=1 output on both pass and fail paths."
    requirement: SIGN-03
    verification:
      - kind: unit
        ref: "internal/upgrade/taskfile_shape_test.go#TestVerifyGatekeeperDeclaresNamedPreconditions"
        status: pass
      - kind: unit
        ref: "internal/upgrade/taskfile_shape_test.go#TestVerifyGatekeeperDeclaresNamedPreconditions_MissingTargetIsError"
        status: pass
      - kind: manual_procedural
        ref: "Task 2 checkpoint — task verify:gatekeeper run live on the maintainer's Mac against v0.5.1 darwin/arm64 and darwin/amd64 (RED), a deliberate GATEKEEPER_EXPECT mismatch (gate-can-fail proof), and an invalid GATEKEEPER_EXPECT value (input-validation proof)"
        status: pass
    human_judgment: true
    rationale: "spctl/syspolicy_check/xattr behavior can only be observed on real macOS hardware (D-08 — no CI rehearsal in this plan); the checkpoint's live results are the actual evidence, not a proxy for it."
  - id: D2
    description: "SIGN-03 RED baseline recorded in 02-EVIDENCE.md: dated RED observations for both darwin arches (spctl exit 3, digest_match=true), the pre-xattr NON-EVIDENCE control, the D-19 positive controls (docker+codex, exit 0) and negative control, assumption A1 marked CLOSED, assumption A2 marked CONFIRMED with the browser-download comparison, the four insufficient-check enumeration, and a docs/RELEASE-PROCEDURES.md §7.1 pointer."
    requirement: SIGN-03
    verification:
      - kind: manual_procedural
        ref: "02-EVIDENCE.md, cross-checked against the Task 2 checkpoint transcript verbatim"
        status: pass
    human_judgment: true
    rationale: "The RED baseline's substance is a recorded observation against real Apple tooling on real hardware, not a property a CI assertion can independently re-derive."

duration: ~35min active work (spans a human checkpoint pause between Task 1 and Task 3, not counted)
completed: 2026-08-09
status: complete
---

# Phase 2 Plan 01: Gatekeeper Tracer & SIGN-03 RED Baseline Summary

**Built `task verify:gatekeeper` — a reusable, D-19-compliant Gatekeeper oracle (`spctl -a -vv -t install`, exit-status-only verdict) — proved it fires RED against the real, un-notarized, published `v0.5.1` darwin assets on real hardware, and settled the phase's one open rig-fidelity question (assumption A2: CONFIRMED, synthetic quarantine matches a genuine browser download's verdict).**

## Performance

- **Duration:** ~35 min of active agent work (Task 1 implementation + Task 3 evidence recording); a human checkpoint pause for live Mac verification separates them and is not counted in this figure.
- **Started:** 2026-08-09 (first commit 12:01 local)
- **Completed:** 2026-08-09 (last commit 12:30 local)
- **Tasks:** 3 (all completed — Task 2 was a `checkpoint:human-verify` and was approved with full live-hardware evidence)
- **Files modified:** 4 (1 created, 3 modified)

## Accomplishments

- `verify:gatekeeper` Taskfile target: downloads a PUBLISHED darwin release asset (never `dist/`, never a local rebuild), cross-checks GitHub's recorded per-asset digest, applies and confirms a SYNTHETIC `com.apple.quarantine` xattr, and derives the verdict from `spctl -a -vv -t install`'s exit status alone — the D-19 LOCKED oracle, never `-t exec`.
- `syspolicy_check distribution` is recorded as a non-gating observation only, per D-19 — its Fatal/exit-70 verdict on an unstapled binary is definitionally unreachable while stapling stays out of scope (D-16/DIST-06), so it is never allowed to fail the target.
- A `SOURCE-ASSERTION-MISMATCH` truth table makes the exit-status verdict and the output-text source assertion two independent, mechanically-checked properties.
- `TestVerifyGatekeeperDeclaresNamedPreconditions` (+ a `MissingTargetIsError` non-vacuity companion) pins the 12-precondition contract via a real YAML decode of `Taskfile.yml`.
- Live-hardware checkpoint (Task 2): RED baseline reproduced on both darwin arches (exit 3), the gate proved it can fail (deliberate mismatch) and halt on bad input (before any network round-trip), and D-19's positive control was independently reproduced against two vendors (docker, codex).
- `.planning/phases/02-apple-signing-notarization/02-EVIDENCE.md` created: the phase's single recorded-observation file, with the RED baseline, D-19's positive/negative controls, assumption A1 marked CLOSED, assumption A2 marked CONFIRMED, and the four insufficient-check enumeration.
- `docs/RELEASE-PROCEDURES.md` §7.1's `v0.5.1` preserved-baseline row now links to `02-EVIDENCE.md`.

## Task Commits

Each task was committed atomically:

1. **Task 1: End-to-end Gatekeeper verdict on one published darwin asset — `verify:gatekeeper`** - `44e156a` (feat)
2. **Task 2: Run the gate on a real Mac and watch it fail** - checkpoint, no direct commit (human-executed on real hardware; results folded into Task 3's evidence file)
3. **Task 3: Record the SIGN-03 RED baseline and settle the synthetic-quarantine question** - `4693d78` (docs), `4c9ec9f` (docs, addendum: the four insufficient checks named explicitly)

**Plan metadata:** commit pending (this SUMMARY.md + REQUIREMENTS.md, worktree mode — STATE.md/ROADMAP.md excluded, orchestrator-owned)

## Files Created/Modified

- `Taskfile.yml` - new `verify:gatekeeper` target (~260 lines): 12 named preconditions, the D-19 oracle, digest cross-check with GATEKEEPER_EXPECT-bound asymmetry, SYNTHETIC quarantine rig, GATEKEEPER-EVIDENCE schema=1 output
- `internal/upgrade/taskfile_shape_test.go` - `TestVerifyGatekeeperDeclaresNamedPreconditions` + `TestVerifyGatekeeperDeclaresNamedPreconditions_MissingTargetIsError`, decoding `Taskfile.yml` with a real YAML decoder
- `.planning/phases/02-apple-signing-notarization/02-EVIDENCE.md` - new: the phase's evidence-line schema restatement, SIGN-03 RED baseline, D-19 controls, A1/A2 resolutions, checkpoint observations
- `docs/RELEASE-PROCEDURES.md` - one-line link from §7.1's `v0.5.1` row to `02-EVIDENCE.md`

## Decisions Made

- **Real YAML decode over the file's existing regex idiom for this one guard.** `internal/upgrade/taskfile_shape_test.go`'s established idiom for `Taskfile.yml` specifically is a regex-based block scanner (`parseTaskBlocks`); real `yaml.Unmarshal` decoding was, before this plan, only used for `.goreleaser.yaml` and GitHub Actions workflow YAML in that file. The plan's acceptance criteria explicitly and repeatedly required a real YAML decoder for this specific test ("never a line scanner"). Verified live, before committing to the approach, that `go.yaml.in/yaml/v3` decodes this repo's actual `Taskfile.yml` (32 real tasks) cleanly. Implemented as new, narrowly-scoped structs (`gatekeeperPrecondition`/`gatekeeperTaskYAML`/`gatekeeperTaskfileRoot`) rather than retrofitting the file's other parsers.
- **A2 CONFIRMED — no rig changes.** Per the plan's own branching instruction ("If assumption A2 was REFUTED... stop and change the rig"), since A2 was CONFIRMED, `Taskfile.yml`'s `verify:gatekeeper` target was left exactly as committed in Task 1; its existing comments already correctly describe the xattr as "a simulation until Task 2's checkpoint confirms it" without asserting a specific prior outcome, so nothing needed correcting.
- **Both D-19 positive controls captured, not just one.** The plan's action text asked for "a notarized Developer ID CLI" (singular); the checkpoint measured two independent vendors (docker, codex) and both are recorded, since it is strictly stronger evidence for the oracle's discriminating power at no extra cost.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Own cautionary comment about avoiding Go-template braces contained literal `{{ }}`**
- **Found during:** Task 1, first live `task --list` validation of the new target
- **Issue:** A comment in `verify:gatekeeper` warning that "a stray `{{ }}` here would be eaten" itself contained `{{ }}`, which `task`'s own Go `text/template` pass ate before the shell ever saw the cmds string — producing `template: :94: missing value for command` and breaking the entire target, including every OTHER task in the file's `--list-all` output.
- **Fix:** Reworded the comment to describe the hazard without literal double-brace syntax ("a stray double-brace template marker").
- **Files modified:** `Taskfile.yml`
- **Verification:** `task --list` and `task --list-all` both parse cleanly afterward; re-ran the invalid-`GATEKEEPER_EXPECT`-value precondition test to confirm the target itself still halts correctly.
- **Committed in:** `44e156a` (part of Task 1's commit — caught and fixed before that commit, not a separate follow-up)

**2. [Rule 1 - Bug] Two prose mentions of the literal string `GATEKEEPER-EVIDENCE` in `02-EVIDENCE.md` were not paired with `schema=1` on the same line**
- **Found during:** Task 3, self-check against the acceptance criterion "`rg -c 'schema=1'` equals the `GATEKEEPER-EVIDENCE` count"
- **Issue:** The general evidence-line-schema description and the OBS-2 discussion mentioned the prefix name on lines that didn't also carry `schema=1`, which — while numerically coincidental to a passing count on the first draft — was not a reliable pairing and could regress silently on a future edit.
- **Fix:** Reworded both mentions so every line containing `GATEKEEPER-EVIDENCE` also contains `schema=1` (and vice versa, apart from the one fully generic schema-convention bullet, which was reworded to avoid the literal `schema=1` string entirely rather than name a prefix it doesn't describe).
- **Files modified:** `.planning/phases/02-apple-signing-notarization/02-EVIDENCE.md`
- **Verification:** `rg -c 'GATEKEEPER-EVIDENCE'` and `rg -c 'schema=1'` both return 5, and manual line-by-line inspection confirms every matching line for one string also matches the other.
- **Committed in:** `4693d78`

---

**Total deviations:** 2 auto-fixed (both Rule 1 — bugs found and fixed before their respective task commits landed)
**Impact on plan:** Both fixes were necessary for the target/evidence file to be correct at all; no scope creep.

## Issues Encountered

- **Pre-existing, unrelated test flake** (`test/wireoracle`'s `TestFrozenTranscriptsMatch/error-unknown-method`): failed once during a full `task test:unit` run, passed cleanly on isolated re-run and on a second full-suite re-run. Touches no file this plan modified; matches the class of async-ordering wire-oracle flakes already tracked in `STATE.md`'s Pending Todos. Not fixed (out of scope, Rule scope boundary), not newly logged (already a known class).
- **Digest-absent code path unexercised against this baseline** (OBS-1, recorded in `02-EVIDENCE.md`): `gh v2.97.0` exposes a `digest` field on every `v0.5.1` asset, so `verify:gatekeeper`'s two digest-missing branches (RED-path sentinel-and-continue, GREEN-path hard failure) never fired during the checkpoint. Recorded as a known-untested path, not silently counted as covered; whether/how to force-exercise it is a deferred follow-up decision, out of scope for this plan.

## Known Stubs

None.

## Next Phase Readiness

- `verify:gatekeeper` is the settled, reusable target plan 02-06's post-release GREEN check will call unchanged (only `GATEKEEPER_EXPECT=accepted` and the real `TAG` differ).
- The evidence-line schema (`schema=1` first field, fixed key order, `not-found`/`unknown` sentinels) is now established phase-wide and ready for later plans' evidence files to follow.
- Assumption A2's CONFIRMED verdict de-risks the remaining phase: no rig rework is needed before plan 02-06's GREEN check trusts the same synthetic-quarantine mechanism.
- OBS-1 (digest-absent path untested) is a candidate for a later plan or a follow-up decision — not blocking, but should not be assumed covered.
- No blockers for plan 02-02 (Developer ID codesigning + notarization implementation).

---
*Phase: 02-apple-signing-notarization*
*Completed: 2026-08-09*
