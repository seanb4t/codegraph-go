---
phase: 08-release-hardening-benchmarks
plan: 01
subsystem: infra
tags: [goreleaser, cgo, cross-compile, zig, ldflags, reproducible-builds]

# Dependency graph
requires: []
provides:
  - ".goreleaser.yaml at repo root: 6-target CGo build matrix (linux/windows/darwin x amd64/arm64)"
  - "Version ldflags contract wired to internal/version.{Version,Commit,Date}"
  - "Raw-binary archive naming contract matching internal/upgrade.releaseAssetName()"
  - "Reproducibility flags (-trimpath, -buildid=, mod_timestamp) on every build entry"
  - "Proven native darwin/arm64 CGo build via goreleaser build --single-target --snapshot"
affects: [08-04-release-workflow, 08-05-upgrade-e2e-test, 08-08-reproducibility-gate]

# Tech tracking
tech-stack:
  added: [goreleaser v2 config]
  patterns:
    - "YAML anchor (&version_ldflags / *version_ldflags) shares the ldflags list across all 6 build entries without repetition"
    - "Native-runner matrix: linux/amd64 and both darwin targets use the host's native toolchain (no CC/CXX override); only linux/arm64 and both windows targets set CC/CXX to zig cc/c++ cross targets"

key-files:
  created: [.goreleaser.yaml]
  modified: []

key-decisions:
  - "archives.format (deprecated in GoReleaser v2.6+) replaced with archives.formats: [binary] to satisfy `goreleaser check` without deprecation warnings, per Context7-verified GoReleaser docs — same raw-binary behavior, current API"
  - "darwin/amd64 and darwin/arm64 build natively with no CC/CXX override (Finding 2): zig-cross-compiling darwin from Linux risks breaking libresolv-based DNS resolution in the resulting binary"
  - "Optional zig-cross validation (linux/arm64) skipped locally — zig is not installed on this dev host; cross-target validation for linux/windows/darwin-amd64 is explicitly deferred to Plan 08-04's CI release matrix, matching the plan's documented maximum-local-proof scope"

patterns-established:
  - "GoReleaser build ids follow codegraph-<goos>-<goarch> naming, one entry per platform, no goos/goarch fan-out within a single entry"

requirements-completed: [DIST-01, DIST-04]

coverage:
  - id: D1
    description: "GoReleaser v2 config declares all 6 CGo build targets with reproducibility flags, version ldflags, and the raw-binary naming contract; goreleaser check passes"
    requirement: "DIST-01"
    verification:
      - kind: other
        ref: "goreleaser check (exit 0, 1 configuration file(s) validated)"
        status: pass
      - kind: other
        ref: "grep verification: 6x CGO_ENABLED=1, 3x internal/version ldflags symbols, {{ .Tag }} used for Version, -trimpath/-buildid=/mod_timestamp present, no signs:/sbom: block"
        status: pass
    human_judgment: false
  - id: D2
    description: "Native darwin/arm64 CGo build compiles and runs, proving the tree-sitter CGo build path is sound on the host"
    requirement: "DIST-04"
    verification:
      - kind: other
        ref: "goreleaser build --single-target --snapshot --clean (exit 0); file dist/*/codegraph reports Mach-O 64-bit arm64; ./dist/*/codegraph --version prints injected version/commit/date"
        status: pass
    human_judgment: false

# Metrics
duration: 12min
completed: 2026-07-13
status: complete
---

# Phase 08 Plan 01: GoReleaser CGo Build Matrix Summary

**GoReleaser v2 config with a 6-target native/zig-cross CGo build matrix, version ldflags, and a raw-binary asset-naming contract pinned to internal/upgrade.releaseAssetName; native darwin/arm64 build proven locally.**

## Performance

- **Duration:** 12 min
- **Started:** 2026-07-13T17:04:00Z
- **Completed:** 2026-07-13T17:06:11Z
- **Tasks:** 2 completed
- **Files modified:** 1 (`.goreleaser.yaml`, created)

## Accomplishments
- Authored `.goreleaser.yaml` with 6 build entries (linux/windows/darwin × amd64/arm64), each with `CGO_ENABLED=1`, `-trimpath`, `mod_timestamp`, and a shared ldflags anchor injecting `internal/version.{Version,Commit,Date}` via `{{ .Tag }}`/`{{ .FullCommit }}`/`{{ .CommitDate }}`
- Wired zig cross-toolchain (`CC=zig cc -target ...` / `CXX=zig c++ -target ...`) for linux/arm64 and both windows targets only; linux/amd64 and both darwin targets rely on the native runner toolchain (Finding 2's native-matrix strategy)
- Set `archives.formats: [binary]` (raw binary, no tar.gz/zip) with `name_template` producing `codegraph_<tag>_<os>_<arch>[.exe]` — verified to match `internal/upgrade.releaseAssetName()`'s exact output shape
- Added `checksum` block (sha256, `codegraph_<tag>_checksums.txt`) and top-of-file contract comments; deliberately omitted `signs:`/`sbom:` (owned by release.yml's assembly job per Plan 08-04)
- Ran `goreleaser build --single-target --snapshot --clean` on the native darwin/arm64 dev host: produced a Mach-O 64-bit arm64 `codegraph` binary that runs `--version` and prints the ldflags-injected identity (`codegraph version v0.0.0 (commit ..., built ...)`)

## Task Commits

Each task was committed atomically:

1. **Task 1: Author .goreleaser.yaml with the 6-target CGo matrix, reproducibility flags, ldflags, and the raw-binary naming contract** - `ad2c0e3` (feat)
2. **Task 2: Prove the CGo build config compiles and runs on the native host (darwin/arm64)** - no commit (proof-only task; produced no file changes — `goreleaser build --single-target --snapshot --clean` output lives in the gitignored `dist/` directory)

**Plan metadata:** (this commit)

## Files Created/Modified
- `.goreleaser.yaml` - GoReleaser v2 build config: 6-target CGo matrix, reproducibility flags, version ldflags, raw-binary naming contract

## Decisions Made
- Used `archives.formats: [binary]` instead of the plan-literal `archives.format: binary` — GoReleaser v2.17.0 (the locally installed version) deprecates the singular `format` key in favor of the plural `formats` list as of v2.6. Verified against Context7-fetched GoReleaser docs before making the substitution; behavior is identical (raw binary, no archive), only the schema key changed. This is a Rule 1 auto-fix (deprecated-property warning would otherwise surface on every `goreleaser check`/`release` run).
- Optional zig-installable secondary local validation (linux/arm64) was skipped — zig is not present on this dev host, and the plan explicitly marks this step optional and non-blocking, deferring full cross-target proof to Plan 08-04's CI matrix.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Replaced deprecated `archives.format: binary` with `archives.formats: [binary]`**
- **Found during:** Task 1 (first `goreleaser check` run)
- **Issue:** `goreleaser check` failed with exit 2 (`configuration is valid, but uses deprecated properties` — `archives.format` deprecated since GoReleaser v2.6 in favor of `archives.formats`)
- **Fix:** Changed `format: binary` to `formats: [binary]` under the single `archives:` entry
- **Files modified:** `.goreleaser.yaml`
- **Verification:** `goreleaser check` now exits 0 with "1 configuration file(s) validated"
- **Committed in:** `ad2c0e3` (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 bug fix — deprecated schema key)
**Impact on plan:** Necessary correctness fix for the installed GoReleaser version; no behavioral change to the raw-binary contract. No scope creep.

## Issues Encountered
- `zig` is not installed on this dev host, so the plan's optional linux/arm64 secondary local zig-cross validation could not be attempted. This was explicitly optional per the plan and does not block the plan's done criteria — cross-target validation (linux/arm64, both windows, darwin/amd64) is delegated to Plan 08-04's CI release matrix.
- The native darwin/arm64 build emitted a benign upstream warning (`'TOKEN_COUNT' macro redefined`) from the `tree-sitter-swift` grammar's C scanner during compilation. This is a pre-existing warning in a third-party grammar, out of scope for this plan (build still succeeded, binary still ran correctly) — logged here per the deviation-rules scope boundary, not fixed.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- `.goreleaser.yaml` is ready for Plan 08-04 (release workflow) to invoke per-runner via `goreleaser build --single-target` for all 6 targets across the linux/macos runner matrix
- The naming contract is proven consistent with `internal/upgrade.releaseAssetName()`, unblocking Plan 08-05's upgrade e2e test
- Reproducibility flags are in place for Plan 08-08's double-build determinism gate
- No blockers; cross-target (non-darwin/arm64) build proof remains CI-only until Plan 08-04 executes

---
*Phase: 08-release-hardening-benchmarks*
*Completed: 2026-07-13*

## Self-Check: PASSED
- FOUND: .goreleaser.yaml
- FOUND: ad2c0e3 (Task 1 commit)
