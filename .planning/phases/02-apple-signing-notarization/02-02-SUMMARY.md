---
phase: 02-apple-signing-notarization
plan: 02
subsystem: infra
tags: [goreleaser, cosign, quill, notarization, yaml, taskfile, go-yaml]

# Dependency graph
requires:
  - phase: 02-apple-signing-notarization
    provides: "plan 02-01's `task verify:gatekeeper` (D-19 oracle) and 02-EVIDENCE.md's SIGN-03 RED baseline against v0.5.1's published, deliberately un-notarized darwin assets"
provides:
  - "`.goreleaser.yaml` `signs:` block (renamed from `binary_signs:`, D-18) with `ids: [raw]` and `artifacts: binary` — cosign now signs release-scoped, after notarize"
  - "`.goreleaser.yaml` `notarize:` block: one `macos:` entry, both darwin build ids explicit, five-term credential-conjunction `enabled:` gate, no entitlements key, 20m per-binary timeout"
  - "internal/upgrade/goreleaser_shape_test.go: TestSignsSidecarMatchesUpgradeContract (renamed+extended), TestNotarizeMacosIdsCoverDarwinBuildIDs, TestNotarizeMacosHasExactlyOneEntry, TestNotarizeMacosOmitsEntitlements, TestNotarizeMacosEnabledIsEnvGated, TestParseGoreleaserSigns_NoSignsBlockIsError, TestParseGoreleaserNotarize_NoNotarizeBlockIsError"
  - "Taskfile.yml: a recorded, in-file notarize-reachability verdict on every goreleaser-invoking target (check:goreleaser, check:darwin-release-build, release:dry-run, release:dry-run-signed, release:goreleaser)"
  - ".planning/phases/02-apple-signing-notarization/deferred-items.md — the darwin-toolchain-canary re-wiring deferral, owned by this phase"
affects: [02-03, 02-04, 02-05, 02-06, 02-07]

# Actuals (#2632)
actuals:
  tokens: 14300
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "resolveGoreleaserFieldTemplateWithFuncs: an injectable template.FuncMap sibling to resolveGoreleaserFieldTemplate, so a GoReleaser template predicate (isEnvSet) can be controlled deterministically from a test without mutating the real process environment or changing the original helper's signature."
    - "Static backstop testing for env-gated templates: resolve a text/template enabled: string under N controlled environments (all-present, all-absent, each single-var-missing) and assert the resolved literal, with an explicit doc-comment boundary stating this proves the template's own logic, not the runtime SDK's evaluation of it."
    - "Two-directional exact-set mutation proof performed on the WORKING FILE with an explicit revert, verified via unchanged `git diff --stat` before/after, when a temporary fixture copy cannot exercise the assertion (the real config path is hardcoded in the test helpers)."

key-files:
  created:
    - .planning/phases/02-apple-signing-notarization/deferred-items.md
  modified:
    - internal/upgrade/goreleaser_shape_test.go
    - .goreleaser.yaml
    - Taskfile.yml
    - .planning/phases/01-cross-compile-spike-goreleaser-release-migration/deferred-items.md

key-decisions:
  - "Applied D-18 (LOCKED, maintainer, 2026-08-09) exactly as specified: renamed binary_signs: to signs:, added ids: [raw] and artifacts: binary, kept cmd/signature/args byte-identical. No re-derivation — the plan explicitly forbade re-litigating this."
  - "Deleted Phase 1's false rationale claiming signs:'s artifacts: binary mode requires archives.formats: binary project-wide (confirmed false at source, D-18/A4) and replaced it with the D-18 citation, the hardcoded-pipe-order finding, and an explicit retraction naming what was wrong and why."
  - "Gated notarize.macos[0].enabled on all FIVE credential variables conjunctively via Go template `and`, not on the certificate alone — per the plan's explicit rejection of a separate enable flag (a sixth input whose accidental presence could enable an incomplete credential set)."
  - "Sized notarize.macos[0].timeout (20m) as a PER-BINARY budget with a comment stating the two-id entry's wall-clock worst case is the SUM of two such budgets (sequential range, not concurrent) — per the plan's retraction of a prior wrong concurrent-notarization claim."
  - "check:darwin-release-build's cosign precondition was removed (D-18 side effect: goreleaser build no longer reaches any cosign call since sign.Pipe is release-scoped), replaced with an explanation that the notarize pipe is STILL reachable there and stopped only by the enabled: credential guard, since --skip=notarize is not valid for the build command."
  - "Re-wiring check:darwin-release-build into darwin-toolchain-canary.yml (a D-18 side benefit named as a candidate, not a mandate) was explicitly deferred, not done — recorded in a new, phase-2-owned deferred-items.md with a one-line backlink from Phase 1's file, per the plan's explicit ownership instruction."

patterns-established:
  - "Reachability-class taxonomy for goreleaser-invoking Taskfile targets (CHECK / BUILD / RELEASE), each carrying an in-file comment naming which of GoReleaser's pipes reach it and what stops them — reusable for any future pipe addition to .goreleaser.yaml."

requirements-completed: [SIGN-01, SIGN-04]

coverage:
  - id: D1
    description: "signs: block renamed from binary_signs: (D-18) with ids: [raw], artifacts: binary, byte-identical signature/cmd/args; retracted-and-replaced load-bearing comment citing D-18; TestSignsSidecarMatchesUpgradeContract asserts the property (resolve-and-distinctness) for all 4 platforms, not a literal string."
    requirement: SIGN-04
    verification:
      - kind: unit
        ref: "internal/upgrade/goreleaser_shape_test.go#TestSignsSidecarMatchesUpgradeContract"
        status: pass
      - kind: unit
        ref: "internal/upgrade/goreleaser_shape_test.go#TestParseGoreleaserSigns_NoSignsBlockIsError"
        status: pass
    human_judgment: false
  - id: D2
    description: "notarize: block added — notarize.macos[0].ids names exactly the two darwin build ids (exact-set, both directions mutation-proved); enabled: is a five-term credential conjunction (7-environment static backstop); sign: declares no entitlements key; exactly one macos: entry enforced by the parser (mutation-proved via a synthetic 2-entry fixture)."
    requirement: SIGN-01
    verification:
      - kind: unit
        ref: "internal/upgrade/goreleaser_shape_test.go#TestNotarizeMacosIdsCoverDarwinBuildIDs"
        status: pass
      - kind: unit
        ref: "internal/upgrade/goreleaser_shape_test.go#TestNotarizeMacosEnabledIsEnvGated"
        status: pass
      - kind: unit
        ref: "internal/upgrade/goreleaser_shape_test.go#TestNotarizeMacosHasExactlyOneEntry"
        status: pass
      - kind: unit
        ref: "internal/upgrade/goreleaser_shape_test.go#TestNotarizeMacosOmitsEntitlements"
        status: pass
      - kind: unit
        ref: "internal/upgrade/goreleaser_shape_test.go#TestParseGoreleaserNotarize_NoNotarizeBlockIsError"
        status: pass
    human_judgment: false
  - id: D3
    description: "Every goreleaser-invoking Taskfile target (check:goreleaser, check:darwin-release-build, release:dry-run, release:dry-run-signed, release:goreleaser) carries an in-file, reachability-class comment stating whether notarize: reaches it and what stops it; check:darwin-release-build's stale binary_signs:/Sigstore-session comment and cosign precondition were corrected."
    verification:
      - kind: other
        ref: "rg -c 'notarize' Taskfile.yml (19); rg -c 'command -v cosign' Taskfile.yml (4, one lower than HEAD's 5); rg -n 'Sigstore session' Taskfile.yml (0 matches)"
        status: pass
      - kind: unit
        ref: "internal/upgrade/goreleaser_shape_test.go#TestWorkflowRunBodiesInvokeTask"
        status: pass
    human_judgment: false
  - id: D4
    description: "deferred-items.md created for phase 02, recording the deferred darwin-toolchain-canary re-wiring with its revisit condition; Phase 1's deferred-items.md carries a one-line backlink and gains no Phase-2 content."
    verification:
      - kind: other
        ref: ".planning/phases/02-apple-signing-notarization/deferred-items.md (created); .planning/phases/01-cross-compile-spike-goreleaser-release-migration/deferred-items.md (backlink only)"
        status: pass
    human_judgment: false

duration: ~55min
completed: 2026-08-09
status: complete
---

# Phase 2 Plan 02: `signs:`/`notarize:` Pipeline Reorder Summary

**Moved cosign off the build-scoped `binary_signs:` pipe onto the release-scoped `signs:` pipe (D-18) so it signs post-notarization bytes by construction, turned on GoReleaser's native `notarize.macos` block with a five-credential conjunctive gate and no entitlements, and recorded a reachability verdict for every `goreleaser`-invoking Taskfile target.**

## Performance

- **Duration:** ~55 min of active agent work
- **Tasks:** 3 (all `type="auto"`, no checkpoints)
- **Files modified:** 5 (1 created, 4 modified)

## Accomplishments

- RED confirmed first: `TestSignsSidecarMatchesUpgradeContract` (renamed from `TestBinarySignsSidecarMatchesUpgradeContract`) and four new notarize tests all failed against the unmodified `.goreleaser.yaml`, non-vacuity companions passed unconditionally — captured verbatim below.
- GREEN: `.goreleaser.yaml`'s `binary_signs:` key renamed to `signs:` with `ids: [raw]` added, `cmd`/`signature`/`args` byte-identical; a `notarize:` top-level block added with one `macos:` entry (both darwin build ids explicit, five-term conjunctive `enabled:` gate, no `entitlements:` key, `wait: true`, a 20m per-binary timeout). All Task 1 tests pass; `task check:goreleaser` exits 0; `task test:unit` passes.
- Phase 1's false rationale for rejecting `signs:` (claiming a project-wide `archives.formats: binary` precondition that does not exist at source) deleted and replaced with the D-18 citation, the hardcoded-pipe-order finding, and an explicit retraction.
- Both directions of the exact-set `ids:` assertions mutation-proved on the real config file, then cleanly reverted (`git diff --stat` unchanged before/after both mutations): adding `zip` to `signs[0].ids` turns `TestSignsSidecarMatchesUpgradeContract` red; adding a linux build id to `notarize.macos[0].ids` turns `TestNotarizeMacosIdsCoverDarwinBuildIDs` red.
- Every `goreleaser`-invoking Taskfile target now carries an in-file reachability-class comment (CHECK/BUILD/RELEASE) naming whether `notarize:` reaches it and what stops it. `check:darwin-release-build`'s stale "requires an interactive Sigstore session" comment and cosign precondition were corrected — D-18 means `goreleaser build` no longer reaches any cosign call, but it still reaches `notarize:` (build-scoped, no `--skip=notarize` escape hatch for the build command), stopped only by the credential guard.
- `.planning/phases/02-apple-signing-notarization/deferred-items.md` created: the darwin-toolchain-canary re-wiring (a D-18 side benefit, named a candidate not a mandate) is deferred, not done, with its revisit condition recorded. Phase 1's deferred-items.md gained a one-line backlink instead of Phase-2 content.

## Task Commits

Each task was committed atomically:

1. **Task 1: RED — retarget and extend the config-shape tests to the post-D-18 shape** - `c19a880` (test)
2. **Task 2: GREEN — apply the D-18 ruling and turn on `notarize:`** - `25a6529` (feat)
3. **Task 3: Enumerate every `goreleaser` caller and state what stops the notarize pipe reaching it** - `7f040fb` (docs)

**Plan metadata:** commit pending (this SUMMARY.md, worktree mode — STATE.md/ROADMAP.md excluded, orchestrator-owned)

## RED output (Task 1, against the unmodified `.goreleaser.yaml`)

```
=== RUN   TestSignsSidecarMatchesUpgradeContract
    goreleaser_shape_test.go:644: mustGoreleaserSigns: parseGoreleaserSigns: no signs: entries found
--- FAIL: TestSignsSidecarMatchesUpgradeContract (0.00s)
=== RUN   TestParseGoreleaserSigns_NoSignsBlockIsError
--- PASS: TestParseGoreleaserSigns_NoSignsBlockIsError (0.00s)
=== RUN   TestParseGoreleaserNotarize_NoNotarizeBlockIsError
--- PASS: TestParseGoreleaserNotarize_NoNotarizeBlockIsError (0.00s)
=== RUN   TestNotarizeMacosIdsCoverDarwinBuildIDs
    goreleaser_shape_test.go:748: mustGoreleaserNotarizeMacos: parseGoreleaserNotarizeMacos: no notarize.macos: entries found
--- FAIL: TestNotarizeMacosIdsCoverDarwinBuildIDs (0.00s)
=== RUN   TestNotarizeMacosHasExactlyOneEntry
    goreleaser_shape_test.go:773: mustGoreleaserNotarizeMacos: parseGoreleaserNotarizeMacos: no notarize.macos: entries found
--- FAIL: TestNotarizeMacosHasExactlyOneEntry (0.00s)
=== RUN   TestNotarizeMacosOmitsEntitlements
    goreleaser_shape_test.go:817: mustGoreleaserNotarizeMacos: parseGoreleaserNotarizeMacos: no notarize.macos: entries found
--- FAIL: TestNotarizeMacosOmitsEntitlements (0.00s)
=== RUN   TestNotarizeMacosEnabledIsEnvGated
    goreleaser_shape_test.go:853: mustGoreleaserNotarizeMacos: parseGoreleaserNotarizeMacos: no notarize.macos: entries found
--- FAIL: TestNotarizeMacosEnabledIsEnvGated (0.00s)
FAIL
```

All four named tests (`TestSignsSidecarMatchesUpgradeContract`, `TestNotarizeMacosIdsCoverDarwinBuildIDs`, `TestNotarizeMacosEnabledIsEnvGated`, `TestNotarizeMacosOmitsEntitlements`) failed for the right reason — no `signs:`/`notarize:` block existed yet. `TestNotarizeMacosHasExactlyOneEntry` also failed for the same reason (not separately required by the acceptance criteria's named list, but consistent). Both non-vacuity companions passed unconditionally, as designed.

## Mutation observations (Task 2, acceptance criteria)

Both mutations were applied to the WORKING `.goreleaser.yaml` (the test helpers hardcode the real config path, so a temp-fixture copy cannot exercise these assertions), then reverted; `git diff --stat .goreleaser.yaml` was identical (150 insertions / 47 deletions) before and after each mutation, confirming a clean revert.

1. **`signs[0].ids` too much:** temporarily changed `ids: [raw]` to `ids: [raw, zip]`. `TestSignsSidecarMatchesUpgradeContract` failed: `signs[0].ids = [raw zip], want exactly the set [raw]`. Reverted; test passes again.
2. **`notarize.macos[0].ids` too much:** temporarily added `codegraph-linux-amd64` to `notarize.macos[0].ids`. `TestNotarizeMacosIdsCoverDarwinBuildIDs` failed: `notarize.macos[0].ids = [codegraph-darwin-amd64 codegraph-darwin-arm64 codegraph-linux-amd64], want exactly the set [codegraph-darwin-amd64 codegraph-darwin-arm64]`. Reverted; test passes again.

Both mutations prove the exact-set assertions fail on matching TOO MUCH, not only on matching too little (the silent-skip direction was already covered by the RED baseline in Task 1, where the blocks were absent entirely).

`TestNotarizeMacosHasExactlyOneEntry`'s own mutation proof (a synthetic in-memory 2-entry YAML fixture, not a working-file mutation) is exercised inside the test itself on every run — see the test body for the fixture and the assertion that `parseGoreleaserNotarizeMacos` rejects it.

## Files Created/Modified

- `internal/upgrade/goreleaser_shape_test.go` - renamed `goreleaserBinarySign`→`goreleaserSign` (+`IDs` field), `goreleaserBinarySignsConfig`→`goreleaserSignsConfig`, `parseGoreleaserBinarySigns`→`parseGoreleaserSigns`, `mustGoreleaserBinarySigns`→`mustGoreleaserSigns`; `TestBinarySignsSidecarMatchesUpgradeContract`→`TestSignsSidecarMatchesUpgradeContract` (+`ids` exact-set assertion, rewritten doc comment); added `goreleaserNotarizeMacos`/`goreleaserNotarizeConfig`/`parseGoreleaserNotarizeMacos`/`mustGoreleaserNotarizeMacos`, `resolveGoreleaserFieldTemplateWithFuncs`, `isEnvSetFuncMap`, `notarizeCredentialVars`, `wantDarwinBuildIDs`, and five new/companion tests
- `.goreleaser.yaml` - `binary_signs:` renamed to `signs:` (+`ids: [raw]`); new `notarize:` block with one `macos:` entry; header contract paragraphs (a)/(c) updated to name the live pipe set and D-18; false Phase-1 rationale deleted and replaced
- `Taskfile.yml` - reachability-class comments added to `check:goreleaser`, `check:darwin-release-build`, `release:dry-run`, `release:dry-run-signed`, `release:goreleaser`; `check:darwin-release-build`'s stale desc/precondition corrected (cosign precondition removed, `Sigstore session` phrase removed); three stale `binary_signs:` references fixed
- `.planning/phases/02-apple-signing-notarization/deferred-items.md` - new: the darwin-toolchain-canary re-wiring deferral
- `.planning/phases/01-cross-compile-spike-goreleaser-release-migration/deferred-items.md` - one-line backlink to the new Phase 2 file

## Decisions Made

See `key-decisions` in frontmatter. All decisions were dictated by the plan's `<critical_locked_decisions>` (D-18) or by the plan's own explicit instructions (five-term conjunction, per-binary timeout sizing, deferral ownership) — no independent architectural choices were made.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Invalid Go syntax in a chained `if` init statement**
- **Found during:** Task 1, `go vet` after first draft of `TestSignsSidecarMatchesUpgradeContract`'s `ids` assertion
- **Issue:** `if want := sortedJoin(...); got := sortedJoin(...); got != want {` is not valid Go — a single `if` statement can carry only one init statement before its condition, not two chained by semicolons.
- **Fix:** Split into two separate statements (`wantSignIDs := ...` then `if got := ...; got != wantSignIDs {`).
- **Files modified:** `internal/upgrade/goreleaser_shape_test.go`
- **Verification:** `go vet ./internal/upgrade/` clean afterward.
- **Committed in:** `c19a880` (part of Task 1's commit — caught and fixed before that commit, not a separate follow-up)

**2. [Rule 1 - Bug] Doc comment still named the retired test function `TestBinarySignsSidecarMatchesUpgradeContract`**
- **Found during:** Task 1, self-check against the acceptance criterion `rg -c 'TestBinarySignsSidecarMatchesUpgradeContract' internal/upgrade/` returning 0
- **Issue:** The renamed test's own doc comment parenthetically referenced its old name, which the acceptance criterion requires to be completely gone, not merely relocated to a comment.
- **Fix:** Reworded the doc comment to describe the retirement without repeating the literal old identifier.
- **Files modified:** `internal/upgrade/goreleaser_shape_test.go`
- **Verification:** `rg -c 'TestBinarySignsSidecarMatchesUpgradeContract' internal/upgrade/` returns 0 (no output, `rg` exit 1).
- **Committed in:** `c19a880` (part of Task 1's commit)

**3. [Rule 1 - Bug] Rewritten `check:darwin-release-build` description still contained the literal phrase "interactive Sigstore session"**
- **Found during:** Task 3, self-check against the acceptance criterion `rg -n 'Sigstore session' Taskfile.yml` returning no line inside that target
- **Issue:** The corrected description explained the target NO LONGER needs an interactive Sigstore session — but the acceptance criterion requires the phrase absent entirely, not merely negated.
- **Fix:** Reworded to "no longer needs a browser-driven keyless-signing flow or an OIDC token" — same meaning, without the literal phrase.
- **Files modified:** `Taskfile.yml`
- **Verification:** `rg -n 'Sigstore session' Taskfile.yml` returns no matches; `task --list-all` still parses cleanly.
- **Committed in:** `7f040fb` (part of Task 3's commit)

---

**Total deviations:** 3 auto-fixed (all Rule 1 — bugs found and fixed before their respective task commits landed)
**Impact on plan:** All three fixes were necessary for the code to compile / for the acceptance criteria to actually hold; no scope creep.

## Issues Encountered

- **Two pre-existing, load-sensitive test flakes** (`test/integration`'s `TestLiveEditAutoSyncReachesExplore` and `test/wireoracle`'s `TestFrozenTranscriptsMatch/toolslist-default`): both failed once during a full `task test:unit` run while sibling worktree agents (plans 02-03, 02-05) were running concurrently, both passed cleanly on isolated re-run, and a subsequent full-suite re-run was entirely clean. Neither package is in this plan's `files_modified`. Not fixed (out of scope, matches this repo's documented rotating load-sensitive flake class in `test/wireoracle`/`internal/daemon`), not newly logged (already a known class per `STATE.md`'s Pending Todos).

## Known Stubs

None.

## Threat Flags

None — this plan operates entirely within the `<threat_model>` T-02-05/T-02-06/T-02-07/T-02-08 dispositions already declared in `02-02-PLAN.md`; no new security-relevant surface was introduced outside that register.

## Next Phase Readiness

- `.goreleaser.yaml`'s `signs:`/`notarize:` shape is now pinned by seven tests (five new/retargeted plus the two non-vacuity companions), all green, with `task check:goreleaser` and `task test:unit` both passing.
- The pipeline-ordering half of SIGN-04 is true by construction (cosign now runs after notarize, per GoReleaser's hardcoded pipe order) — plan 02-04's rehearsal is what MEASURES this empirically (D-05/D-07's one-time recorded mutation is still owed, and this plan's Task 2 mutation proofs were scoped to the static config-shape assertions only, not a live GoReleaser run's sha256 diff).
- Every `goreleaser`-invoking Taskfile target's notarize-reachability is now documented in-file; the darwin-toolchain-canary re-wiring stays explicitly deferred (see `deferred-items.md`) pending plan 02-07's real notarized release observing the `enabled:` guard fire in both directions in actual CI.
- No blockers for plan 02-03 or the rest of the wave.

## Self-Check: PASSED

- FOUND: `.planning/phases/02-apple-signing-notarization/02-02-SUMMARY.md`
- FOUND: commit `c19a880` (Task 1)
- FOUND: commit `25a6529` (Task 2)
- FOUND: commit `7f040fb` (Task 3)
- FOUND: commit `1812c6c` (plan metadata — SUMMARY.md + REQUIREMENTS.md)

---
*Phase: 02-apple-signing-notarization*
*Completed: 2026-08-09*
