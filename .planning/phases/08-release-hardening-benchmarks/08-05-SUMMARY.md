---
phase: 08-release-hardening-benchmarks
plan: 05
subsystem: infra
tags: [sigstore, cosign, goreleaser, upgrade, e2e-test]

requires:
  - phase: 08-01
    provides: ".goreleaser.yaml archives.name_template (raw-binary, per-target naming contract)"
  - phase: 08-04
    provides: "release.yml assembly job that per-binary cosign-signs the 6 shipped release targets"
provides:
  - "TestReleaseAssetNameMatchesGoReleaser pinning releaseAssetName<->name_template agreement (D-14) for all 6 os/arch pairs"
  - "TestVerifyReleaseE2E: real-signed-artifact e2e test against verifyRelease's PRODUCTION identity, skip-if-absent"
affects: [08-08-reproducibility-gate]

tech-stack:
  added: []
  patterns:
    - "e2e test skip-if-absent seam: committed testdata fixture (primary) or CODEGRAPH_E2E_BINARY/CODEGRAPH_E2E_BUNDLE env vars (live variant), else t.Skip with a clear reason — never a spurious failure"

key-files:
  created:
    - internal/upgrade/verify_release_e2e_test.go
  modified:
    - internal/upgrade/upgrade.go

key-decisions:
  - "releaseAssetName already agreed with .goreleaser.yaml's name_template byte-for-byte (codegraph_<tag>_<goos>_<goarch>[.exe]) for all 6 targets — no logic change needed; only its doc comment was updated to record the agreement is now test-proven, not just asserted in prose"
  - "TestVerifyReleaseE2E prioritizes a committed testdata/ fixture pair over CODEGRAPH_E2E_BINARY/CODEGRAPH_E2E_BUNDLE env vars, falling through to a clean t.Skip when neither is present — no real signed release exists yet (DIST-02's first real tag-triggered release is still pending)"

patterns-established:
  - "Pattern: e2e tests gated on real external artifacts skip cleanly via t.Skip with an actionable message, rather than being short-mode-gated or failing — keeps `go test ./...` always green pre-release while still proving the real path once an artifact exists"

requirements-completed: [DIST-02]

coverage:
  - id: D1
    description: "releaseAssetName's output is pinned equal to GoReleaser's archives.name_template for all 6 shipped (os,arch) release targets (D-14 closed)"
    requirement: "DIST-02"
    verification:
      - kind: unit
        ref: "internal/upgrade/verify_release_e2e_test.go#TestReleaseAssetNameMatchesGoReleaser"
        status: pass
    human_judgment: false
  - id: D2
    description: "A real signed release binary + .sigstore.json bundle passes verifyRelease under the PRODUCTION identity (releaseWorkflowRefPattern/releaseOIDCIssuer), and a wrong-identity input is rejected — offline this run (no artifact exists yet), so the test cleanly skips"
    requirement: "DIST-02"
    verification:
      - kind: e2e
        ref: "internal/upgrade/verify_release_e2e_test.go#TestVerifyReleaseE2E"
        status: unknown
    human_judgment: true
    rationale: "No real signed release artifact exists yet — DIST-02's first real tag-triggered release.yml run hasn't shipped. The test is proven to compile, run, and skip cleanly (never fail spuriously) in this session, but the actual accept/reject assertions against a genuine signed binary can only be exercised once a fixture (testdata/e2e-release-binary[.sigstore.json]) or CODEGRAPH_E2E_BINARY/CODEGRAPH_E2E_BUNDLE-pointed real artifact exists — see 08-VALIDATION.md's 'First real signed-release verify' manual-only row."

duration: 3min
completed: 2026-07-13
status: complete
---

# Phase 08 Plan 05: Upgrade E2E Verification Loop-Closer Summary

**Pinned the releaseAssetName<->GoReleaser name_template asset-naming contract (D-14) with a test, and added the real-signed-artifact e2e test against `verifyRelease`'s production identity that Finding 1 required — currently skip-clean pending a real DIST-02 release.**

## Performance

- **Duration:** 3 min
- **Started:** 2026-07-13T18:14:26Z
- **Completed:** 2026-07-13T18:16:20Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- `TestReleaseAssetNameMatchesGoReleaser` proves, for all 6 shipped (os,arch) release targets, that `releaseAssetName()`'s output is byte-identical to `.goreleaser.yaml`'s `archives.name_template` output — closing D-14. No mismatch was found; `releaseAssetName`'s doc comment was updated to record the agreement is now test-proven.
- `TestVerifyReleaseE2E` exercises the full sign->download->verify chain: sha256-over-the-binary (Finding 1's exact call path, never a checksums file), a real cosign v3 `.sigstore.json` bundle, and `verifyRelease` under the PRODUCTION `releaseWorkflowRefPattern`/`releaseOIDCIssuer` constants (never the offline fixture SAN from `verify_test.go`). It sources the real artifact via a committed `testdata/` fixture pair (primary) or `CODEGRAPH_E2E_BINARY`/`CODEGRAPH_E2E_BUNDLE` env vars (live variant), and includes a negative sub-case asserting a wrong SAN regex is rejected.
- No real signed release exists yet, so `TestVerifyReleaseE2E` currently `t.Skip()`s with an actionable message — proven to never fail spuriously — and will exercise its assertions for real the moment a fixture or live artifact is supplied.

## Task Commits

Each task was committed atomically:

1. **Task 1: Assert releaseAssetName agrees with the GoReleaser name_template for all 6 os/arch** - `8d64231` (test)
2. **Task 2: End-to-end verifyRelease against a real signed artifact (skip-if-absent), asserting the production identity** - `d3f4296` (test)

**Plan metadata:** (this commit)

_Note: Both tasks landed as single `test(08-05):` commits — no `feat`/`fix` commit was needed since `releaseAssetName` already agreed with the template and `verifyRelease`/`verify.go` were already fully implemented in Phase 6._

## Files Created/Modified
- `internal/upgrade/verify_release_e2e_test.go` - New file: `TestReleaseAssetNameMatchesGoReleaser` (D-14 asset-name pin) + `TestVerifyReleaseE2E` (real-artifact e2e, skip-if-absent)
- `internal/upgrade/upgrade.go` - `releaseAssetName`'s doc comment updated to record the D-14 agreement is now test-proven, not just asserted

## Decisions Made
- No mismatch existed between `releaseAssetName` and GoReleaser's `name_template`, so Task 1 required only a doc-comment update to `upgrade.go`, not a logic fix.
- `TestVerifyReleaseE2E` fetches the LIVE Sigstore trusted root (`fetchTrustedRoot()`) rather than the offline embedded fixture trust root used by `verify_test.go`, since a real captured bundle is signed against the real public-good Sigstore instance, not the sigstore-js test fixture's certs. This is the one path in the package that touches the network, and it is scoped to only run when a real artifact is actually supplied — never on a bare `go test ./...`.

## Deviations from Plan

None - plan executed exactly as written. Both tasks completed with only `test(08-05):` commits, matching the plan's own conditional wording ("Commit `test(08-05):` ... and a `fix(08-05):` commit if upgrade.go changed" — no fix was needed).

## TDD Gate Compliance

This plan's own tasks describe conditional (not mandatory) implementation changes: Task 1 says "if a mismatch is found... finalize `releaseAssetName`... Run to GREEN", and Task 2 is inherently a skip-if-absent test with no corresponding production code to write (the verifier it exercises, `verifyRelease`, already shipped in Phase 6). Both tests passed (Task 1: fully green; Task 2: cleanly skipped) on first run, with no mismatch and no missing implementation to fix — so no `feat`/`fix` commit was warranted. This is expected and matches the plan's design intent (a loop-closing, proof-only plan), not a TDD gate violation.

## Issues Encountered
None.

## User Setup Required

None - no external service configuration required. Once a real DIST-02 release ships (a real `v*` tag push through `release.yml`), capture its binary + `.sigstore.json` bundle for the current host's os/arch under `internal/upgrade/testdata/e2e-release-binary` and `internal/upgrade/testdata/e2e-release-binary.sigstore.json` (or point `CODEGRAPH_E2E_BINARY`/`CODEGRAPH_E2E_BUNDLE` at downloaded copies) to exercise `TestVerifyReleaseE2E`'s real assertions — see `08-VALIDATION.md`'s "First real signed-release verify" manual-only row.

## Next Phase Readiness
- D-14's asset-naming contract is proven, not just asserted — `codegraph upgrade` cannot 404 on a name mismatch for any of the 6 shipped targets.
- The e2e verify loop-closer test exists and is wired correctly; it will start proving real signed-artifact acceptance/rejection automatically the moment a real release or captured fixture is available, with zero further code changes required.
- No blockers for Plan 08-08 (reproducibility gate) or later phases.

---
*Phase: 08-release-hardening-benchmarks*
*Completed: 2026-07-13*

## Self-Check: PASSED
- FOUND: internal/upgrade/verify_release_e2e_test.go
- FOUND: internal/upgrade/upgrade.go
- FOUND commit 8d64231
- FOUND commit d3f4296
