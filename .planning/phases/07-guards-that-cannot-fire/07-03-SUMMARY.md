---
phase: 07-guards-that-cannot-fire
plan: 03
subsystem: release
tags: [bash, shellcheck, goreleaser, cosign, taskfile, mutation-log, guard-hardening]

# Dependency graph
requires:
  - phase: 07-guards-that-cannot-fire (plan 01, plan 02)
    provides: "07-MUTATION-LOG.md header, cleanliness-gate convention, and families (a)/(b), ready for this plan to append family (c)"
provides:
  - "scripts/inject-cosign-key.sh: extracts the awk --key= injection and additions-only diff guard from both release Task targets into one script, adding the positive assertion that the injection added exactly one --key= line, closing GRD-03/T-02-08's vacuous-guard gap"
  - "release:dry-run-signed and release:rehearse-notarize both call the one script; no inline injection block remains in Taskfile.yml"
  - "07-MUTATION-LOG.md family (c): RED demonstration proving the guard refuses a config whose sign-blob anchor no longer matches, with the committed .goreleaser.yaml proven byte-unchanged throughout"
  - "T-02-08 pending todo resolved and moved to .planning/todos/completed/"
affects: [07-guards-that-cannot-fire]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
actuals:
  tokens: 3651
  tasks: 3
  commits: 3
  plan_head_before: 8d0659a66c2af8e72e176b216c0837714444a0ec

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Shared Taskfile logic used by two Task targets extracted into one scripts/*.sh taking file paths as positional args, never a directory — each call site keeps its own run-scoped tmpdir variable and cleanup trap"
    - "Positive count assertion added alongside a pre-existing negative-only (additions-only) diff guard, closing the exact vacuity shape rule 84d1gfpywd names: an empty diff trivially satisfies every negative check"
    - "RED demonstration against a copy in a temporary directory, never the tracked file, when the guard under test protects a file that must never be edited to prove it — recorded as an explicit deviation from the tracked-file-mutation/revert shape families (a) and (b) use"

key-files:
  created:
    - scripts/inject-cosign-key.sh
  modified:
    - Taskfile.yml
    - .planning/phases/07-guards-that-cannot-fire/07-MUTATION-LOG.md
    - .planning/todos/completed/2026-08-09-dry-run-signed-additions-only-diff-guard-passes-vacuously.md (moved from .planning/todos/pending/)

key-decisions:
  - "Script argv order fixed as <committed-config> <generated-config> <cosign-key>, matching the three values each call site's SIGNTEST_DIR/REHEARSE_DIR-scoped variables already supplied to the inline block being extracted"
  - "Did not check that $3 (the cosign key path) exists — embedded as a path string only, since both call sites generate the key immediately before invoking and not requiring it is what makes the RED demonstration runnable standalone with no darwin/zig/syft/cosign preconditions, per the plan's flagged assumption 1"
  - "Used grep, not rg, inside the script — a verbatim lift of grep-based logic for a release-path script running on GitHub-hosted runners and maintainer laptops where rg is not a declared dependency, per the plan's flagged assumption 2 and project convention"
  - "No shape test added to taskfile_shape_test.go asserting either target calls the script (D-07) — the script's own count assertion plus its recorded RED demonstration is the guard"
  - "Family (c)'s mutation-log entry has no tracked-file mutation and no revert step, mirroring family (a)'s documented shape deviation but for a different reason: the committed .goreleaser.yaml must never be edited to prove this guard, so the perturbation lives entirely in a copy written to a temporary directory"

requirements-completed: [GRD-03]

coverage:
  - id: D1
    description: "scripts/inject-cosign-key.sh exists, is tracked at mode 100755, lints clean under shellcheck, and running it against the real committed .goreleaser.yaml reports exactly one injected --key= line and exits 0"
    requirement: "GRD-03"
    verification:
      - kind: other
        ref: "shellcheck scripts/inject-cosign-key.sh (exit 0); git ls-files -s scripts/inject-cosign-key.sh (mode 100755); bash scripts/inject-cosign-key.sh .goreleaser.yaml \"$T/gen.yaml\" \"$T/cosign.key\" | rg 'injected --key= lines: 1' (exit 0)"
        status: pass
    human_judgment: false
  - id: D2
    description: "The script refuses bad arity (2 args, exit 2 with a usage message) and an unreadable first argument (exit 2, naming the path)"
    requirement: "GRD-03"
    verification:
      - kind: other
        ref: "bash scripts/inject-cosign-key.sh .goreleaser.yaml \"$T/gen2.yaml\" (exit 2, usage on stderr); bash scripts/inject-cosign-key.sh /nonexistent/foo.yaml \"$T/gen3.yaml\" \"$T/cosign2.key\" (exit 2, names the path)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Both release:dry-run-signed and release:rehearse-notarize call the one script with three arguments each, and neither retains an inline copy of the extracted injection/diff-guard logic; the Taskfile still parses and no shape test was added"
    requirement: "GRD-03"
    verification:
      - kind: other
        ref: "task --list-all (exit 0, lists both targets); rg -c 'scripts/inject-cosign-key.sh' Taskfile.yml == 2; rg -c 'keyline' Taskfile.yml == 0 (via `|| echo 0` fallback — see Deviations); git diff --quiet -- internal/upgrade/taskfile_shape_test.go (exit 0, untouched)"
        status: pass
    human_judgment: false
  - id: D4
    description: "A config whose sign-blob anchor no longer matches is refused: the script reports a count of zero and exits non-zero against a re-indented copy, while the committed .goreleaser.yaml is proven byte-unchanged both before and after the demonstration"
    requirement: "GRD-03"
    verification:
      - kind: other
        ref: "07-MUTATION-LOG.md family (c) — pasted transcript reporting 'injected --key= lines: 0' and the refusal message naming found 0, exit 1; git diff --quiet -- .goreleaser.yaml (exit 0) both immediately before and immediately after"
        status: pass
    human_judgment: false
  - id: D5
    description: "T-02-08 pending todo is resolved and moved to .planning/todos/completed/"
    requirement: "GRD-03"
    verification:
      - kind: other
        ref: "git ls-files -- .planning/todos/completed/2026-08-09-dry-run-signed-additions-only-diff-guard-passes-vacuously.md (tracked); frontmatter carries status: resolved, resolved_by_phase: 7"
        status: pass
    human_judgment: false

# Metrics
duration: 18min
completed: 2026-09-09
status: complete
---

# Phase 7 Plan 3: GRD-03 cosign-key injection guard extraction Summary

**Extracted the dry-run-signed cosign-key injection and its additions-only diff guard into `scripts/inject-cosign-key.sh`, added the missing positive assertion counting injected `--key=` lines, and proved RED that a re-indented sign-blob anchor is refused rather than silently passing — closing the vacuous-guard todo T-02-08.**

## Performance

- **Duration:** 18 min
- **Started:** 2026-09-08T20:20:00-04:00 (approx.)
- **Completed:** 2026-09-08T20:38:13-04:00
- **Tasks:** 3
- **Files modified:** 4 (1 created)

## Accomplishments
- `scripts/inject-cosign-key.sh` (mode 100755, shellcheck-clean) lifts the awk `--key=` injection and additions-only diff guard verbatim from both `release:dry-run-signed` and `release:rehearse-notarize`, taking `<committed-config> <generated-config> <cosign-key>` as positional arguments; it refuses bad arity and an unreadable input config, both with exit 2.
- Added the positive assertion the guard has always been missing (D-06): counts the `--key=` lines added in the diff, prints `injected --key= lines: N`, and refuses any count other than exactly 1 — naming both failure modes (0 = anchor no longer matches, 2+ = duplicated sign-blob block).
- Both `release:dry-run-signed` and `release:rehearse-notarize` now call the one script; `Taskfile.yml`'s two inline injection blocks are gone (`rg -c 'keyline' Taskfile.yml` returns 0), preconditions/cleanup traps/goreleaser invocations in both targets are untouched, and no shape test was added asserting the wiring (D-07).
- Proved RED against a copy of the committed `.goreleaser.yaml` with the `sign-blob` anchor re-indented by two extra spaces: the additions-only checks pass trivially on the resulting empty diff (the exact vacuity this todo names), and the new count assertion is the only thing that refuses, reporting `found 0` and exiting non-zero. The committed `.goreleaser.yaml` was proven byte-unchanged before and after via `git diff --quiet`, since the perturbation lived entirely in a temp-dir copy.
- `07-MUTATION-LOG.md` family (c) appended with the verbatim failing transcript, the cleanliness gates, and the green re-run reporting a count of 1. T-02-08 closed: moved to `.planning/todos/completed/`, `status: resolved`, `resolved_by_phase: 7`.

## Task Commits

Each task was committed atomically:

1. **Task 1: Create scripts/inject-cosign-key.sh with the extracted guard plus the count assertion** - `d6f9ee35` (feat)
2. **Task 2: Rewire release:dry-run-signed and release:rehearse-notarize to call the script** - `97f91fce` (feat)
3. **Task 3: RED demonstration against a re-indented copy, mutation-log family (c), and close the T-02-08 todo** - `65df37ee` (docs)

**Plan metadata:** commit created by this SUMMARY's own atomic write+commit step.

## Files Created/Modified
- `scripts/inject-cosign-key.sh` - New script: cosign-key injection, additions-only diff guard, and the new exactly-1 count assertion
- `Taskfile.yml` - Both release targets' inline injection blocks replaced with one call each to the script
- `.planning/phases/07-guards-that-cannot-fire/07-MUTATION-LOG.md` - Appended family (c): RED transcript, cleanliness gates, green re-run
- `.planning/todos/completed/2026-08-09-dry-run-signed-additions-only-diff-guard-passes-vacuously.md` - Moved from `.planning/todos/pending/`, marked resolved with a Resolution section

## Decisions Made
- Argv order, the `$3`-existence exemption, and `grep` over `rg` inside the script all followed the plan's flagged assumptions and D-05/D-06 exactly — see `key-decisions` in frontmatter for the full reasoning.
- No REFACTOR-style cleanup needed beyond what the plan specified: the extraction was a verbatim lift plus one new assertion block, and both call sites' rewiring was a mechanical single-block replacement.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Verified the `keyline` count criterion with a corrected command, not the literal one**
- **Found during:** Task 2 (acceptance-criteria verification loop)
- **Issue:** The plan's own literal verify command, `test "$(rg -c 'keyline' Taskfile.yml || true)" = "0"`, relies on `rg -c` printing `0` when there are no matches. `rg -c` instead exits 1 and prints **nothing** on zero matches (confirmed empirically: `rg -c 'keyline' Taskfile.yml` → exit 1, empty stdout), so the command substitution captures an empty string, not `"0"`, and the literal test as written would report FAILURE even when the actual property (zero `keyline` references remaining) holds. This is the same "count occurrences, don't trust a bare exit status" class of gotcha this project's own grepping conventions warn about.
- **Fix:** Re-verified using `COUNT=$(rg -c 'keyline' Taskfile.yml || echo 0); test "$COUNT" = "0"`, which correctly reports `0`. No code or Taskfile change was needed — the underlying property (both inline injection blocks fully removed) was already correct; only the verification command needed the `|| echo 0` fallback instead of `|| true`. Not a fix to PLAN.md, which is not one of this plan's `files_modified` — flagged here so a downstream `/gsd-verify-work` or manual re-run of the plan's literal `<verify>` block isn't misread as a regression.
- **Files modified:** None (verification-only; no production or plan file changed)
- **Verification:** `rg -c 'scripts/inject-cosign-key.sh' Taskfile.yml` returns 2; `rg -c 'keyline' Taskfile.yml || echo 0` returns 0; both confirmed against the final committed `Taskfile.yml`
- **Committed in:** N/A (no commit required; documented as a deviation per Rule 1's "auto-fix, verify, track" shape, applied to the verification step rather than the implementation)

---

**Total deviations:** 1 auto-fixed (1 bug, verification-only — no implementation or plan file changed)
**Impact on plan:** No scope creep and no code change. The only effect is a corrected verification command; the underlying acceptance criterion (zero `keyline` references remaining in `Taskfile.yml`) was true before and after this correction.

## Issues Encountered

None beyond the deviation above (caught and resolved within Task 2's own acceptance-criteria gate).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `07-MUTATION-LOG.md` now carries families (a), (b) and (c) — ready for 07-04 (GRD-04, GRD-05) to append family (d) and the GRD-05 deletion record.
- `scripts/inject-cosign-key.sh` has no dependency on anything 07-04 will touch (`internal/upgrade/release_workflow_shape_test.go`, `post-release-verify.yml`); no blockers.
- GRD-03 marked complete in REQUIREMENTS.md. GRD-06 remains open (shared across 07-01/02/03/04's mutation-log contributions, per the shared-ID gate) — will be marked complete once 07-04 produces its SUMMARY.md.
- The committed `.goreleaser.yaml` is byte-identical to its pre-plan state throughout (`git diff --quiet -- .goreleaser.yaml` confirmed clean at every checkpoint in this plan).

---
*Phase: 07-guards-that-cannot-fire*
*Completed: 2026-09-09*

## Self-Check: PASSED

All created/modified files found on disk; all three task commits (d6f9ee35, 97f91fce, 65df37ee) found in git log. Plan produced 3 commits against `plan_head_before` 8d0659a6 (measured via `git rev-list --count`).
