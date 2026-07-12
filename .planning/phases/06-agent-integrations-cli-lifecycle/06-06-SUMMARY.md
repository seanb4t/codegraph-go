---
phase: 06-agent-integrations-cli-lifecycle
plan: 06
subsystem: cli
tags: [sigstore-go, cosign-keyless, self-update, atomic-swap, github-releases, cli-lifecycle]

# Dependency graph
requires:
  - phase: 06-agent-integrations-cli-lifecycle
    provides: "internal/version.Info() (06-05) for --check's current-vs-latest comparison"
provides:
  - "internal/upgrade package: resolveLatestVersion/resolveLatestVersionViaAPI (GitHub Releases redirect trick + API fallback), verifyRelease (in-process sigstore-go keyless verification), atomicSwap (temp-in-same-dir + os.Rename POSIX / rename-aside Windows), Run orchestrator (resolve -> check?/download -> verify -> swap, fail-closed)"
  - "codegraph upgrade [version] [--check] command registered in root.go"
  - "go.mod direct dependency on github.com/sigstore/sigstore-go v1.2.2"
affects: [08-release-engineering-distribution (DIST-02 finalizes releaseRepoSlug/releaseWorkflowRefPattern/releaseAssetName placeholders)]

# Tech tracking
tech-stack:
  added: [github.com/sigstore/sigstore-go v1.2.2]
  patterns:
    - "Injectable orchestration seams: upgrade.Options carries unexported resolveLatest/download/verify/swap func fields, nil defaulting to the real implementation — lets upgrade_test.go drive the whole resolve/download/verify/swap sequence with fakes, fully offline"
    - "Package-level var indirection for CLI-layer testability: internal/cli/upgrade.go's upgradeRunFunc var mirrors the same seam pattern one level up so the thin command's flag/arg wiring is testable without touching upgrade.Run's real network path"
    - "Hermetic sigstore-go fixtures: testdata/{valid-bundle,trusted-root}.json copied verbatim from sigstore-go's own embedded pkg/testing/data (real signed sigstore-js release provenance + the public-good TUF trust root) rather than a live TUF fetch or a hand-forged bundle"

key-files:
  created:
    - internal/upgrade/release.go
    - internal/upgrade/verify.go
    - internal/upgrade/swap.go
    - internal/upgrade/upgrade.go
    - internal/upgrade/release_test.go
    - internal/upgrade/verify_test.go
    - internal/upgrade/swap_test.go
    - internal/upgrade/upgrade_test.go
    - internal/upgrade/testdata/valid-bundle.json
    - internal/upgrade/testdata/trusted-root.json
    - internal/cli/upgrade.go
    - internal/cli/upgrade_test.go
  modified:
    - internal/cli/root.go
    - go.mod
    - go.sum

key-decisions:
  - "Tampered-artifact reject test modeled as a digest MISMATCH against a real, validly-signed bundle rather than a hand-corrupted bundle file — a byte-corrupted bundle would just fail JSON/protobuf parsing (never exercising the crypto policy check); the realistic MITM threat is an attacker swapping the downloaded BYTES while being unable to forge a matching Fulcio-signed digest, which is exactly what WithArtifactDigest catches."
  - "Used sigstore-go's own embedded hermetic test fixtures (pkg/testing/data's sigstore-js release bundle + public-good trust root, copied verbatim into internal/upgrade/testdata) for fully offline accept/reject tests, resolving Open Question 1 via its recommended approach (a) — proves the wiring/policy-construction/error-handling path end to end against a real signed bundle with zero network dependency, including the TrustedRoot load (root.NewTrustedRootFromJSON from a static file, not FetchTrustedRoot)."
  - "verify.go's release identity constants (releaseRepoSlug, releaseWorkflowRefPattern) are set to seanb4t/codegraph-go as Phase-8-finalized placeholders with explicit doc comments — they compile and are wired into upgrade.Run's production defaultVerify path today but are not exercised by any test (verify_test.go always drives verifyRelease directly with the fixture identity)."
  - "defaultDownload is a real (if untested) GitHub Releases asset-fetch implementation — not a hard-coded 'not available' stub — matching D-14's intent that the upgrade CLIENT be fully implemented this phase; releaseAssetName's exact naming convention is documented as a Phase-8/goreleaser-finalized placeholder."
  - "Added a package-level upgradeRunFunc var in internal/cli/upgrade.go (mirrors internal/upgrade.Options' own seam pattern) so the CLI-layer test asserts flag/arg/os.Executable wiring without ever invoking upgrade.Run's real network path."
  - "go.mod/go.sum resynced via GOFLAGS=-mod=mod go build ./... (not go mod tidy) after manually promoting sigstore-go to the direct require block — pulled in its ~20-module transitive subtree and bumped several PRE-EXISTING indirect deps' versions (prometheus/client_golang, klauspost/compress, spf13/cast, spf13/pflag, etc.) to satisfy the shared module graph; no existing direct require was removed (git diff go.mod has zero deleted lines)."

patterns-established:
  - "Verify-before-swap fail-closed orchestration: internal/upgrade.Run never calls swap unless verify returned nil, proven at both the unit level (verify.go's own accept/reject tests) and the orchestrator level (upgrade_test.go's tampered-download-never-swaps test asserting the swap seam is literally never invoked)"

requirements-completed: [CLI-02]

coverage:
  - id: D1
    description: "codegraph upgrade --check resolves the latest GitHub release version and reports availability WITHOUT downloading a binary"
    requirement: "CLI-02"
    verification:
      - kind: unit
        ref: "internal/upgrade/upgrade_test.go#TestUpgradeRun_CheckReportsAvailabilityWithoutDownloading"
        status: pass
      - kind: unit
        ref: "internal/cli/upgrade_test.go#TestUpgradeCommand_DelegatesWithCheckAndVersion"
        status: pass
    human_judgment: false
  - id: D2
    description: "codegraph upgrade downloads the target-platform binary, verifies its sigstore-go signature/provenance in-process, and only then atomically replaces the running binary"
    requirement: "CLI-02"
    verification:
      - kind: unit
        ref: "internal/upgrade/verify_test.go#TestVerifyRelease_AcceptsValidBundle"
        status: pass
      - kind: unit
        ref: "internal/upgrade/upgrade_test.go#TestUpgradeRun_ValidPathVerifiesBeforeSwap"
        status: pass
    human_judgment: false
  - id: D3
    description: "A tampered or unverifiable artifact is REJECTED before any swap — the original binary is left untouched (the security-critical mitigation)"
    requirement: "CLI-02"
    verification:
      - kind: unit
        ref: "internal/upgrade/verify_test.go#TestVerifyRelease_RejectsTamperedArtifact"
        status: pass
      - kind: unit
        ref: "internal/upgrade/verify_test.go#TestVerifyRelease_RejectsWrongIdentity"
        status: pass
      - kind: unit
        ref: "internal/upgrade/upgrade_test.go#TestUpgradeRun_TamperedDownloadNeverSwaps"
        status: pass
      - kind: other
        ref: "rg -n 'exec.Command|os/exec' internal/upgrade/verify.go (no matches)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Atomic self-replace uses a temp file in the target's own directory + os.Rename (POSIX)/rename-aside (Windows), leaving no partial binary on interruption, and refuses a non-writable target before downloading"
    requirement: "CLI-02"
    verification:
      - kind: unit
        ref: "internal/upgrade/swap_test.go#TestSwap_ReplacesTargetAtomically"
        status: pass
      - kind: unit
        ref: "internal/upgrade/swap_test.go#TestSwap_NotWritableTargetDirLeavesOriginalIntact"
        status: pass
      - kind: unit
        ref: "internal/upgrade/upgrade_test.go#TestUpgradeRun_RefusesNonWritableTargetBeforeDownloading"
        status: pass
    human_judgment: false

duration: 7min
completed: 2026-07-12
status: complete
---

# Phase 6 Plan 6: Signature-Verified Self-Update (upgrade) Summary

**`internal/upgrade` package + `codegraph upgrade [version] [--check]`: GitHub Releases redirect-trick resolution, in-process sigstore-go keyless verification (no cosign CLI), and a fail-closed verify-before-atomic-swap orchestrator, proven offline against sigstore-go's own real signed release fixture.**

## Performance

- **Duration:** 7 min
- **Started:** 2026-07-12T15:34:19-04:00
- **Completed:** 2026-07-12T15:41:39-04:00
- **Tasks:** 3
- **Files modified:** 15 (12 created, 3 modified)

## Accomplishments
- `resolveLatestVersion`/`resolveLatestVersionViaAPI` (`release.go`): the unauthenticated GitHub Releases redirect trick (no rate limit) with a rate-limited-API fallback, both fully offline-testable via an injected `httpDoer` seam
- `atomicSwap` (`swap.go`): temp-file-in-same-directory + `os.Rename` (POSIX) / rename-self-aside dance (Windows), with a `checkWritable` precondition shared by the swap itself and the orchestrator's pre-download fail-fast check (D-13)
- `verifyRelease` (`verify.go`): in-process `sigstore-go` keyless verification — Fulcio certificate identity + `sha512`/`sha256`-agnostic artifact digest policy — with zero `os/exec` usage, proven against sigstore-go's own real signed `sigstore-js` release bundle and the public-good trust root (both embedded as offline JSON fixtures)
- `Run` orchestrator (`upgrade.go`): `resolve -> --check?/download -> verify -> swap`, structurally fail-closed — a verification error is fatal and the swap seam is provably never invoked on that path (asserted at both the unit and orchestrator level)
- `codegraph upgrade [version] [--check]` registered in `root.go`, a thin command resolving `os.Executable()` + `version.Info().Version` and delegating everything else to `upgrade.Run` via an injectable `upgradeRunFunc` var
- `github.com/sigstore/sigstore-go v1.2.2` pinned into `go.mod`'s direct require block; `go.sum` resynced for its transitive subtree with zero existing direct requires removed

## Task Commits

Each task was committed atomically (RED -> GREEN):

1. **Task 1: release resolution + atomic swap**
   - `cee06e1` test(06-06): add failing tests for release resolution + atomic swap
   - `ee9707f` feat(06-06): add release resolution + atomic swap (internal/upgrade)
2. **Task 2: sigstore-go verification — the security gate**
   - `248da15` test(06-06): add failing tests for sigstore-go verification
   - `c6a5161` feat(06-06): add sigstore-go verification (internal/upgrade/verify.go)
3. **Task 3: upgrade orchestrator + CLI command**
   - `0d378f8` test(06-06): add failing tests for upgrade orchestrator + CLI command
   - `665a22b` feat(06-06): add upgrade orchestrator + codegraph upgrade command (CLI-02)

**Plan metadata:** (this commit)

## Files Created/Modified
- `internal/upgrade/release.go` - `resolveLatestVersion`/`resolveLatestVersionViaAPI`, `httpDoer` seam, redirect-capturing HTTP client
- `internal/upgrade/verify.go` - `verifyRelease` (sigstore-go policy check), `loadBundle`, `fetchTrustedRoot`, Phase-8-finalized identity constants
- `internal/upgrade/swap.go` - `atomicSwap`, `checkWritable`, POSIX/Windows branch
- `internal/upgrade/upgrade.go` - `Options`/`Run` orchestrator, `defaultResolveLatest`/`defaultDownload`/`defaultVerify`
- `internal/upgrade/{release,verify,swap,upgrade}_test.go` - RED-then-GREEN test suites for each file above
- `internal/upgrade/testdata/{valid-bundle,trusted-root}.json` - hermetic fixtures copied from sigstore-go's own `pkg/testing/data`
- `internal/cli/upgrade.go` - `newUpgradeCmd()`, `upgradeRunFunc` seam
- `internal/cli/upgrade_test.go` - flag/arg delegation + error-propagation tests
- `internal/cli/root.go` - registered `newUpgradeCmd()`
- `go.mod`/`go.sum` - `sigstore-go` direct require + resynced transitive subtree

## Decisions Made
See `key-decisions` in frontmatter — six decisions, all Rule 1-3 class (test-fixture strategy, identity-constant placeholders, real-vs-stub download implementation, CLI testability seam, go.mod resync mechanics). None are architectural changes (Rule 4); all necessary to make the plan's own acceptance criteria satisfiable while staying honest about what's Phase-8-finalized.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] go.sum desync across unrelated pre-existing packages after pinning sigstore-go**
- **Found during:** Task 3 (`go test ./internal/cli/... -run TestUpgrade` failed with "missing go.sum entry" errors for `spf13/cast`, `prometheus/client_golang`, `klauspost/compress`, `spf13/pflag` — packages unrelated to sigstore-go, already used by `mcp-go`/`pebble`/`cobra`)
- **Issue:** Pinning `sigstore-go`'s direct require caused Go's module graph resolution (MVS) to bump several shared transitive dependencies' minimum versions; under `-mod=readonly` (the project default), `go.sum` lacked entries for the newly-selected versions, breaking `internal/cli` builds even though `internal/cli` doesn't import sigstore-go directly
- **Fix:** Ran `GOFLAGS=-mod=mod go build ./...` once to let Go resolve and record every affected `go.sum` entry, then verified `go build ./...` succeeds again under the default `-mod=readonly` and confirmed via `git diff go.mod` that zero existing direct requires were deleted (only additions/version bumps to already-indirect entries)
- **Files modified:** `go.mod`, `go.sum`
- **Verification:** `go build ./...` and `go vet ./...` clean under default readonly mode; `git diff go.mod | grep '^-'` empty (no deletions)
- **Committed in:** `665a22b` (Task 3 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Necessary consequence of adding a real dependency with a large transitive subtree (RESEARCH's own documented tradeoff); no scope creep, no direct dependency lost.

## Issues Encountered
None beyond the go.sum resync above.

## User Setup Required
None - no external service configuration required. `codegraph upgrade --check` was smoke-tested against the real (release-less) `seanb4t/codegraph-go` GitHub repo and produced a clear, non-crashing error ("could not resolve the latest version from GitHub (API returned 404 Not Found)") rather than downloading or swapping anything — the honest "no-op/err gracefully offline" behavior the plan's success criteria allow for a project with no releases yet.

## Next Phase Readiness
- `internal/upgrade` is fully implemented and unit-tested against fixtures per D-14; Phase 8 (DIST-02) only needs to update `releaseRepoSlug`/`releaseWorkflowRefPattern`/`releaseAssetName` to their final production values once signed releases actually ship — no structural changes required.
- The verify-before-swap fail-closed invariant is proven at three independent levels (bundle-policy unit tests, orchestrator call-order tests, and the no-os/exec grep), giving Phase 8's real-release cutover a solid regression net.

---
*Phase: 06-agent-integrations-cli-lifecycle*
*Completed: 2026-07-12*
