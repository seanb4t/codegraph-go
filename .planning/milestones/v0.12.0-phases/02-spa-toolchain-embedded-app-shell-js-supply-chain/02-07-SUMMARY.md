---
phase: 02-spa-toolchain-embedded-app-shell-js-supply-chain
plan: 07
subsystem: infra
tags: [pnpm, taskfile, ci, supply-chain, security, strictDepBuilds, pnpm-audit, sveltekit]

# Dependency graph
requires:
  - phase: 02-06
    provides: "web:deps/web:build/web:drift Taskfile targets and the ci.yml test-job JS install/rebuild/drift steps this plan extends"
provides:
  - "task web:deps:strict — positive assertion that BLD-05's strictDepBuilds gate is actually in effect, never a grep of pnpm's warning text"
  - "task web:lockfile — BLD-06's lockfile-SHAPE guard (version, integrity-equality, zero non-registry sources)"
  - "task web:audit — BLD-06's pnpm audit gate, classified on JSON output shape, never on exit code alone"
  - "Two new CI steps folded into the existing test job (D-13, no new job)"
  - "SECURITY.md stating both scanners' real, disjoint scope and the vendored-component gap"
affects: [phase-03-shadcn-svelte-component-work]

# Actuals (#2632)
actuals:
  tokens: 25900
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Scratch-dir `go run` against an already-vendored Go dependency (go.yaml.in/yaml/v3) for structural YAML/JSON reads inside a Taskfile cmds: block — no new .go file ever committed, matching internal/upgrade's own 'never a line scanner' discipline"
    - "Classify a JSON CLI tool's output by OUTPUT SHAPE, not exit code, when two distinct failure modes (registry-unreachable, advisories-present) share the same exit code"

key-files:
  created: []
  modified:
    - Taskfile.yml
    - web/pnpm-workspace.yaml
    - web/pnpm-lock.yaml
    - .github/workflows/ci.yml
    - SECURITY.md
    - web/build/ (rebuilt, marker refreshed)

key-decisions:
  - "pnpm's own `approve-builds --all` does not write the `allowBuilds` key when nothing is pending (confirmed empirically) — so `allowBuilds: {}` was written by hand once to distinguish 'checked, nothing to approve' (committed) from 'never checked' (absent), which the new gate treats as a named failure"
  - "Discovered a real, live GHSA-pxg6-pf52-xh8x advisory (cookie <0.7.0, pulled in transitively by @sveltejs/kit@2.70.3, whose own dependency range is still ^0.6.0) via the empirical pnpm audit --json run this plan's Task 2 required — fixed via a pnpm-workspace.yaml override (cookie@<0.7.0: ^0.7.2) rather than leaving task web:audit legitimately RED on a clean tree"
  - "BLD-05/BLD-06 classifiers use a scratch `go run` (go.yaml.in/yaml/v3 for YAML, stdlib encoding/json for the audit output) instead of a line-oriented shell parser — the same discipline internal/upgrade's own shape tests already use, and no new .go file is committed"

patterns-established:
  - "Non-vacuity sibling assertions (web:lockfile) must be Taskfile deps: of the gate they backstop (web:audit), so the ordering and total independence from the gated command's exit code is structural, not just documented"

requirements-completed: [BLD-05, BLD-06, BLD-03]

coverage:
  - id: D1
    description: "BLD-05: task web:deps:strict asserts strictDepBuilds resolves to true and that allowBuilds is committed (even empty), never grepping pnpm's warning text"
    requirement: BLD-05
    verification:
      - kind: manual_procedural
        ref: "task web:deps:strict (clean-tree run, verify block) + captured RED transcripts: strictDepBuilds flipped false, allowBuilds absent, and a scratch pnpm project where an unapproved esbuild build script makes `pnpm install --frozen-lockfile` exit 1"
        status: pass
    human_judgment: false
  - id: D2
    description: "BLD-06: task web:lockfile (shape: version, integrity equality, zero file:/git/tarball sources) and task web:audit (CLEAN vs SCAN ERROR vs ADVISORIES PRESENT, classified on JSON shape) — CLEAN and SCAN ERROR demonstrated; ADVISORIES PRESENT implemented but unexercised (D-15, recorded limitation)"
    requirement: BLD-06
    verification:
      - kind: manual_procedural
        ref: "task web:lockfile / task web:audit clean-tree runs + five captured RED transcripts: below-floor, changed lockfileVersion, removed integrity field, file: source, and SCAN ERROR against an unreachable registry"
        status: pass
    human_judgment: true
    rationale: "The ADVISORIES-PRESENT branch is real code that no test in this phase exercises (D-15 declined a rotting pinned-advisory fixture) — a human should be aware the three-way classification is only two-thirds demonstrated, per this plan's own <known_limitations>."
  - id: D3
    description: "Both gates folded into the existing test CI job (D-13, no new job); SECURITY.md states both scanners' disjoint scope and the vendored-component gap"
    requirement: BLD-06
    verification:
      - kind: unit
        ref: "internal/upgrade TestWorkflowRunBodiesInvokeTask, TestRequiredCheckNamesPreserved (adapted — see Deviations)"
        status: pass
      - kind: other
        ref: "task lint:actions"
        status: pass
    human_judgment: false
  - id: D4
    description: "Phase's own BLD-03 drift gate left green on the phase's final commit — web/pnpm-workspace.yaml and web/pnpm-lock.yaml edits (inside web:drift's hashed source set) triggered a SOURCE-half mismatch, resolved via task web:build + marker refresh"
    requirement: BLD-03
    verification:
      - kind: other
        ref: "task web:drift (final run, both counts printed, PASS) + internal/uiserver TestEmbeddedFSMatchesOnDiskBuildTree / TestEmbeddedBuildTreeIsNonTrivial"
        status: pass
    human_judgment: false

duration: 22min
completed: 2026-08-25
status: complete
---

# Phase 2 Plan 7: JS Supply-Chain Gates (BLD-05, BLD-06) Summary

**Positive `strictDepBuilds` assertion, a lockfile-shape + `pnpm audit` gate classified on JSON output shape, and a real GHSA-pxg6-pf52-xh8x fix discovered along the way — all folded into the existing CI `test` job.**

## Performance

- **Duration:** ~22 min
- **Completed:** 2026-08-25T00:55:50Z
- **Tasks:** 3
- **Files modified:** 15 (5 hand-authored: Taskfile.yml, ci.yml, SECURITY.md, pnpm-workspace.yaml; 2 machine-updated: pnpm-lock.yaml, web/build/.build-manifest; ~8 regenerated web/build/ output files)

## Accomplishments

- `task web:deps:strict` (BLD-05): reads `web/pnpm-workspace.yaml` with a real YAML decoder (`go.yaml.in/yaml/v3`, already a direct main-module dependency, built into a scratch dir via `go run` — no new `.go` file ever committed) and asserts `strictDepBuilds` is exactly `true` and `allowBuilds` is committed (even if empty), printing both counts unconditionally before any pass/fail branch.
- `task web:lockfile` + `task web:audit` (BLD-06): a lockfile-SHAPE sibling (declared package count against a floor of 50, `lockfileVersion` equality, integrity-bearing-equals-total-resolutions equality, zero `file:`/git/HTTP-tarball sources) that runs to completion as a Taskfile `deps:` of the audit gate and never reads the audit's exit code, plus an audit classifier that reads `pnpm audit --json`'s OUTPUT SHAPE rather than its exit code — empirically confirmed that a clean scan and a registry-unreachable scan both exit 1, so exit code alone cannot tell them apart.
- Both gates folded into the existing `test` CI job (D-13) as two new steps, each a single `task <target>` invocation.
- `SECURITY.md` rewritten to state `govulncheck` and `pnpm audit` cover two disjoint dependency trees, neither a superset of the other, plus a new bullet naming the structural gap: registry-vendored UI component source (e.g. a future `shadcn-svelte add`) is invisible to `pnpm-lock.yaml`-based scanning.
- Discovered and fixed a real, live low-severity advisory (GHSA-pxg6-pf52-xh8x, `cookie` <0.7.0, transitively pulled in by `@sveltejs/kit@2.70.3`) via a `pnpm-workspace.yaml` `overrides` entry, so `task web:audit`'s own `<verify>` (which requires a clean tree) is honestly green rather than masking a real finding.
- `web/build/` rebuilt and `web/build/.build-manifest` refreshed after the Task 1/2 edits to `web/pnpm-workspace.yaml` and `web/pnpm-lock.yaml` moved the SOURCE-half digest; `task web:drift` confirmed PASS as the phase's own final gate.

## Task Commits

1. **Task 1: BLD-05 — assert `strictDepBuilds` is in effect, and prove the gate can fire** - `b530fd2a` (feat)
2. **Task 2: BLD-06 — lockfile-shape guard plus a `pnpm audit` gate** - `a4db0ce1` (feat)
3. **Task 3: Fold both gates into the `test` job, update SECURITY.md, leave `web:drift` green** - `0f70d5b5` (feat)

**Plan metadata:** (this commit)

## Files Created/Modified

- `Taskfile.yml` - new `web:deps:strict`, `web:lockfile`, `web:audit` targets
- `web/pnpm-workspace.yaml` - `allowBuilds: {}` written by hand (pnpm's own tool doesn't write an empty map); `overrides: { cookie@<0.7.0: ^0.7.2 }` fixing GHSA-pxg6-pf52-xh8x
- `web/pnpm-lock.yaml` - regenerated by the override (`pnpm install`, then reconfirmed frozen)
- `.github/workflows/ci.yml` - two new steps in the existing `test` job
- `SECURITY.md` - two-scanner scope statement + vendored-component gap bullet
- `web/build/`, `web/build/.build-manifest` - rebuilt to re-satisfy `web:drift` after the source-set edits above

## Decisions Made

- **`allowBuilds: {}` written by hand, not by `pnpm approve-builds`.** Empirically confirmed `pnpm approve-builds --all` reports "There are no packages awaiting approval" for this dependency tree and, importantly, writes *nothing* to `pnpm-workspace.yaml` in that case — the key stays totally absent. Since the new gate treats total absence as a named failure distinct from a committed empty map (per ROADMAP's "approval state belongs in committed config" instruction), the empty map had to be added directly. Documented inline in the file's own comment.
- **Fixed a live vulnerability discovered mid-plan rather than deferring it.** Task 2's required empirical `pnpm audit --json` run surfaced a real, current, low-severity advisory (not a hypothetical) — `@sveltejs/kit@2.70.3`'s own dependency range still pins `cookie: ^0.6.0` upstream, so a plain `pnpm update` could not reach the patched line. Forced via `pnpm-workspace.yaml`'s `overrides` map. Verified `pnpm build` / `svelte-check` unaffected (cookie 0.6→0.7 is a parsing-strictness fix, not an API break). This is a Rule 1 auto-fix: the plan's own Task 2 `<verify>` requires a clean audit on this tree, and leaving the advisory in place would make the new gate legitimately RED at commit time.
- **Classifiers use Go, not shell line-scanning or Node.** `go.yaml.in/yaml/v3` (already a direct dependency of the main module, and the exact library `internal/upgrade`'s own shape tests use) for the workspace/lockfile YAML reads; stdlib `encoding/json` for the audit output. Built into a `mktemp -d` scratch directory via `go run` at task time — no new `.go` file is ever committed, matching this plan's `<artifacts_this_phase_produces>` constraint ("New files: none").

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed a live GHSA-pxg6-pf52-xh8x advisory discovered during Task 2's empirical audit run**
- **Found during:** Task 2 (empirically confirming `pnpm audit --json`'s output shape against the real tree, per the plan's own instruction)
- **Issue:** The real, unmodified dependency tree carried an active low-severity advisory (`cookie` <0.7.0 accepting out-of-bounds characters in cookie name/path/domain, GHSA-pxg6-pf52-xh8x / advisory 1103907) via `@sveltejs/kit@2.70.3`'s transitive `cookie` dependency. `@sveltejs/kit@latest`'s own published range is still `cookie: ^0.6.0`, so a normal `pnpm update` could not fix it.
- **Fix:** Added `overrides: { cookie@<0.7.0: ^0.7.2 }` to `web/pnpm-workspace.yaml`, ran `pnpm install` to regenerate the lockfile, reconfirmed `pnpm install --frozen-lockfile` is stable, and reconfirmed `pnpm build` / `svelte-check` are unaffected.
- **Files modified:** `web/pnpm-workspace.yaml`, `web/pnpm-lock.yaml`
- **Verification:** `pnpm audit --json` reports zero advisories after the override; captured in the empirical-shape transcript this plan's Task 2 already required.
- **Committed in:** `a4db0ce1` (Task 2 commit)

**2. [Rule 1 - Bug] Adapted Task 3's verify command to two real tests instead of a third, non-existent one**
- **Found during:** Task 3, running the plan's own `<verify>` block
- **Issue:** The plan's `<verify>` names three tests via `-run 'TestWorkflowRunBodiesInvokeTask|TestWorkflowRunStepsInvokeTaskTargets|TestRequiredCheckNamesPreserved'`. `TestWorkflowRunStepsInvokeTaskTargets` does not exist as a function in `internal/upgrade/taskfile_shape_test.go` — it survives only as a stale comment reference; the property it once named (failing on a `runBodyExceptions` entry matching no real step) is now folded into `TestWorkflowRunBodiesInvokeTask` itself.
- **Fix:** Ran verification against the two tests that actually exist and cover the required properties (`TestWorkflowRunBodiesInvokeTask`, `TestRequiredCheckNamesPreserved`), both PASS with the required exactly-one-PASS-line, zero-FAIL-line evidence.
- **Files modified:** none (verification-only; the plan's `<verify>` text itself was not edited)
- **Verification:** `go test ./internal/upgrade/... -run 'TestWorkflowRunBodiesInvokeTask|TestRequiredCheckNamesPreserved$' -v` — 2/2 PASS, 0 FAIL.
- **Committed in:** n/a (deviation is in the verification procedure, not committed code)

---

**Total deviations:** 2 auto-fixed (1 Rule 1 vulnerability fix, 1 Rule 1 verify-command adaptation)
**Impact on plan:** The vulnerability fix is a necessary correctness/security fix the plan's own gate design surfaced live; without it, `task web:audit` would have been honestly RED on a clean tree. The verify-command adaptation changes only which shell command was run to prove the same structural properties — no scope creep, no weakening of the assertion.

## Known Stubs

None. All three new Taskfile targets (`web:deps:strict`, `web:lockfile`, `web:audit`) are fully wired against real data (the committed `web/pnpm-workspace.yaml` and `web/pnpm-lock.yaml`), not mock or placeholder input.

## Known Limitations (carried from PLAN.md `<known_limitations>`)

`task web:audit`'s three-way classification (CLEAN / SCAN ERROR / ADVISORIES PRESENT) has its ADVISORIES-PRESENT branch implemented and named but **not exercised** in this phase — D-15 explicitly declined a `vuln:selftest`-shaped pinned-advisory fixture because a pinned JS advisory rots as packages are yanked or patched, and no low-maintenance pin was found. What this plan actually proves: the sibling assertion (`web:lockfile`) proves the scan had real input, and the classifier never lets a failed scan read as clean (CLEAN and SCAN ERROR are both demonstrated with real, captured transcripts). It does not prove the detector can fire on a real advisory. `T-02-07-07` in `02-07-PLAN.md`'s threat register carries this as an accepted risk with a revisit condition.

## Issues Encountered

None beyond the two deviations documented above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- ROADMAP criteria 4 and 5's JS halves are closed: CI fails loudly on an unapproved lifecycle script, and `pnpm audit` runs as a named gate distinguishing "scanned and clean" from "the scan itself failed," backed by a lockfile-shape sibling assertion structurally incapable of reading the audit's exit code.
- Phase 2's own BLD-03 drift gate is green on the phase's final commit.
- This is the last plan in Phase 2 — ready for phase-level verification.
- Phase 3 inherits the recorded, accepted gap: registry-vendored `shadcn-svelte` component source will be invisible to `pnpm audit` from its first `shadcn-svelte add` onward (`SECURITY.md`, `02-CONTEXT.md`).

## Self-Check: PASSED

- `Taskfile.yml` contains `web:deps:strict`, `web:lockfile`, `web:audit` — FOUND (`task --list` confirms registration; `task <target>` runs succeed).
- `web/pnpm-workspace.yaml` contains `allowBuilds: {}` and `overrides: { cookie@<0.7.0: ^0.7.2 }` — FOUND.
- `.github/workflows/ci.yml` contains the two new `test`-job steps — FOUND (`task lint:actions` clean, `TestWorkflowRunBodiesInvokeTask` PASS).
- `SECURITY.md` states both scanners' scope and the vendored-component gap — FOUND.
- Commits `b530fd2a`, `a4db0ce1`, `0f70d5b5` — FOUND (`git log --oneline -5`).
- `task web:drift` PASS on the final tree, `git status --porcelain web/` empty — FOUND.
- `go test ./...` green (50 `ok` packages, 0 `FAIL` lines) — FOUND.

---
*Phase: 02-spa-toolchain-embedded-app-shell-js-supply-chain*
*Completed: 2026-08-25*
