---
phase: 08-surface-reconciliation-signed-v1-0-0-release
plan: 07
subsystem: testing
tags: [supply-chain, cgo, go-list, archtest, govulncheck, sbom, reproducibility, charm, lipgloss, bubbletea]

# Dependency graph
requires:
  - phase: 08-surface-reconciliation-signed-v1-0-0-release (plans 01-06)
    provides: internal/cli/present's charm.land/lipgloss/v2, bubbletea/v2, bubbles/v2 dependency (Phase 6/7 TUI work) plus the existing archtest package and its go/packages closure-walk pattern
provides:
  - "internal/cli/present/archtest/charm_cgo_test.go: a durable, fail-closed go test proving the charm.land dependency closure reachable from internal/cli has zero CgoFiles packages"
  - "recorded evidence that govulncheck, SBOM emission, and the reproducible double-build all still pass post-Charm"
affects: [release, ci, supply-chain-audit]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "CGo-closure guard via `go list -deps -json` decoded in-process (not golang.org/x/tools/go/packages) — packages.Package has no CgoFiles field; cmd/go's own list.Package JSON schema is the only toolchain-native source of that signal"
    - "non-vacuous closure assertion: fail if the scoped package set is empty, not just if it contains CGo — prevents a guard from going green because nothing matched"

key-files:
  created:
    - internal/cli/present/archtest/charm_cgo_test.go
  modified: []

key-decisions:
  - "Deviated from the plan's literal 'use golang.org/x/tools/go/packages' instruction: that package's Package struct does not expose CgoFiles (confirmed via `go doc`/`go help list`). Shelled out to `go list -deps -json` instead — the exact RESEARCH-verified command — decoded via encoding/json.Decoder. This is a Rule 3 (blocking issue) auto-fix: the suggested API cannot satisfy the acceptance criteria as literally worded, and go list is toolchain-baseline, not a new dependency."
  - "Verified the guard's fail-closed behavior empirically (not just by code review): temporarily re-scoped charmClosurePathPrefix/closureTarget to internal/indexer/... and github.com/tree-sitter, confirmed the test reports 13 CgoFiles offenders and FAILs, then reverted to the real charm.land/internal/cli scope before committing."
  - "Task 2 (govulncheck/SBOM/double-build re-run) is evidence-only per its own <action> spec — it modifies no source, CI, or release config file. Findings are recorded below rather than as a separate commit."

requirements-completed: [REL-01]

coverage:
  - id: D1
    description: "charm.land dependency closure reachable from internal/cli proven to contain zero CgoFiles packages, via a non-vacuous, fail-closed go test"
    requirement: "REL-01"
    verification:
      - kind: unit
        ref: "internal/cli/present/archtest/charm_cgo_test.go#TestCharmCgoClosure"
        status: pass
    human_judgment: false
  - id: D2
    description: "govulncheck ./... clean on the post-Charm module graph (0 vulnerabilities reachable from code)"
    requirement: "REL-01"
    verification:
      - kind: other
        ref: "govulncheck@v1.6.0 ./... — 0 vulnerabilities in code, 0 in imported packages, 1 informational (GO-2026-5932, golang.org/x/crypto/openpgp, not called) in required-but-unused modules"
        status: pass
    human_judgment: false
  - id: D3
    description: "goreleaser config validates and SBOM (syft) emits without error against a built binary post-Charm"
    requirement: "REL-01"
    verification:
      - kind: other
        ref: "goreleaser check (1 configuration file validated) + syft <binary> -o spdx-json (148 packages emitted, exit 0)"
        status: pass
    human_judgment: false
  - id: D4
    description: "reproducible double-build (linux/amd64, CGO_ENABLED=1) matches ci.yml's reproducibility job byte-for-byte post-Charm"
    requirement: "REL-01"
    verification:
      - kind: other
        ref: "local reproduction of ci.yml's linux/amd64 double-build (matching -trimpath/-buildid=/SOURCE_DATE_EPOCH flags exactly), sha256 identical: e9db986b...48c3c682"
        status: pass
    human_judgment: false

# Metrics
duration: 30min
completed: 2026-07-19
status: complete
---

# Phase 08 Plan 07: Charm Supply-Chain CGo Closure Audit Summary

**Fail-closed `go list -deps -json`-backed guard test proves the charm.land/lipgloss/bubbletea/bubbles closure (10 packages, 123-package transitive graph) is 100% CGo-free, with govulncheck/SBOM/reproducible-build gates re-confirmed green post-Charm.**

## Performance

- **Duration:** 30 min
- **Started:** 2026-07-19T20:13:59Z (approx, session-relative)
- **Completed:** 2026-07-19T20:13:59Z
- **Tasks:** 2
- **Files modified:** 1 (created)

## Accomplishments
- Added `internal/cli/present/archtest/charm_cgo_test.go`: a durable regression guard that will fail the build the moment a future Charm dependency bump introduces a CGo package into the closure reachable from `internal/cli`.
- Empirically proved the guard is non-vacuous and fail-closed by temporarily re-scoping it at the known-CGo tree-sitter closure (13 offenders reported, test failed as expected), then reverting.
- Re-ran and recorded all pre-existing REL-01 supply-chain gates on the post-Charm module graph: `govulncheck` clean, `go list -deps -json` closure diff reproduced (`cgo_in_closure: []`), `goreleaser check` + `syft` SBOM emission clean, and a local reproduction of ci.yml's linux/amd64 double-build gate hash-matched.
- Made zero changes to `.github/workflows/ci.yml`, `.github/workflows/release.yml`, or `.goreleaser.yaml` — all gates reused as-is per the plan's prohibition.

## Task Commits

Each task was committed atomically:

1. **Task 1: charm.land closure CGo guard test (REL-01, D-07)** - `3c1fa0a` (test)
2. **Task 2: re-run govulncheck / SBOM / double-build gates post-Charm** - no code commit (evidence-only task per its own spec; findings recorded in this SUMMARY)

**Plan metadata:** (recorded in the final docs commit for this plan)

## Files Created/Modified
- `internal/cli/present/archtest/charm_cgo_test.go` - Fail-closed guard: loads `go list -deps -json` over `internal/cli/...`'s transitive closure, scopes to `charm.land/*` import paths, asserts the set is non-empty (a vacuous zero-match pass is itself a failure) and that every matched package has `len(CgoFiles) == 0`.

## Decisions Made
- **`go list -deps -json` instead of `golang.org/x/tools/go/packages`:** the plan's literal wording said to mirror `import_graph_test.go`'s `go/packages` pattern, but `packages.Package` has no `CgoFiles` field (verified via `go doc golang.org/x/tools/go/packages.Package` and `go help list`). `cmd/go`'s own `list.Package` JSON schema is the only toolchain-native source of `CgoFiles`. The guard shells to `go list -deps -json github.com/seanb4t/codegraph-go/internal/cli/...` and decodes the streamed JSON with `encoding/json.Decoder` — same verified command as RESEARCH §Package Legitimacy Audit, no new dependency introduced (Rule 3: blocking issue, the suggested API cannot satisfy the task).
- **Fail-closed proof was executed, not just asserted:** per the task's acceptance criteria, the plan explicitly required demonstrating the guard actually fails on a known-CGo scope. This was done by temporarily editing the test's constants to target `internal/indexer/...` / `github.com/tree-sitter`, observing 13 CgoFiles offenders and a FAIL, then reverting file contents to the real charm.land/internal/cli scope before running `go vet`/`go test` a final time and committing.
- **No CI/release/goreleaser file changes:** all three supply-chain gates (govulncheck, SBOM, reproducibility) were re-run locally against the current post-Charm graph exactly as they run in CI, confirming they remain green without needing any pipeline edits.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Switched CGo-detection API from go/packages to go list -deps -json**
- **Found during:** Task 1 implementation
- **Issue:** `golang.org/x/tools/go/packages`'s `Package` struct (the API the plan instructed to reuse from `import_graph_test.go`) does not expose a `CgoFiles` field — confirmed by inspecting its full field list via `go doc`. Using it as literally specified would have made it impossible to implement the required CgoFiles assertion.
- **Fix:** Implemented the closure load via `os/exec` running `go list -deps -json <target>` (the RESEARCH-verified command) and decoded the streamed JSON with a minimal local struct (`ImportPath`, `CgoFiles`). This is still 100% toolchain-native (no new dependency) and produces the identical data RESEARCH already verified (`charm_pkg_count: 10`, `cgo_in_closure: []`).
- **Files modified:** `internal/cli/present/archtest/charm_cgo_test.go`
- **Verification:** `go vet ./internal/cli/present/archtest/...` and `go test ./internal/cli/present/archtest/... -run CharmCgo -count=1 -v` both pass; fail-closed behavior separately confirmed by temporary re-scoping (see Decisions Made above).
- **Committed in:** `3c1fa0a` (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking — API substitution required to satisfy the task's own acceptance criteria)
**Impact on plan:** No scope creep. The substitution preserves every acceptance criterion (non-vacuous, fail-closed, zero-CgoFiles assertion, no new dependency) using an equally toolchain-native mechanism.

## Issues Encountered
- Local cross-compilation of the linux/amd64 reproducibility double-build (needed to reproduce ci.yml's gate on this darwin/arm64 dev machine) initially failed with a zig cache `AccessDenied` error because zig's default local cache directory resolved inside the (effectively read-only) Go module cache. Resolved by pointing `ZIG_GLOBAL_CACHE_DIR`/`ZIG_LOCAL_CACHE_DIR` at a writable scratch directory; both builds then succeeded and hash-matched (`e9db986b...48c3c682`). This is a local-machine reproduction detail only — ci.yml's own `reproducibility` job runs natively on an `ubuntu-latest` linux/amd64 runner and does not need zig for this leg (zig is only installed there for the separate linux/arm64 cross-target leg per the workflow's own comment), so no CI change was needed or made.
- `syft` was not preinstalled locally; installed via `brew install syft` (v1.48.0) to reproduce release.yml's SBOM step. This is a local verification tool only — release.yml already installs syft itself via `anchore/sbom-action/download-syft`, unchanged.

## Recorded Evidence (Task 2)

| Gate | Command | Result |
|------|---------|--------|
| CGo closure guard | `go test ./internal/cli/present/archtest/... -run CharmCgo -count=1` | PASS — 10 charm.land packages audited, 0 with CgoFiles |
| Closure diff (RESEARCH command) | `go list -deps -json ./internal/cli/... \| jq -s '...'` | `{"charm_pkg_count":10,"charm_direct_cgo":[],"closure_size":123,"cgo_in_closure":[]}` — reproduces RESEARCH exactly |
| Vulnerability scan | `govulncheck@v1.6.0 ./...` | 0 vulnerabilities affecting code; 1 informational (GO-2026-5932, `golang.org/x/crypto/openpgp`, unmaintained/unsafe, code doesn't call it) in required-but-unreached modules — pre-existing, not new, not blocking |
| goreleaser config | `goreleaser check` | "1 configuration file(s) validated" |
| SBOM emission | `syft <built-binary> -o spdx-json=...` | exit 0, 148 packages in emitted SPDX document |
| Reproducible double-build | Local reproduction of ci.yml's `reproducibility` job (linux/amd64, `CGO_ENABLED=1`, `-trimpath -buildid= -ldflags` with pinned `SOURCE_DATE_EPOCH`) | Two independent builds, identical SHA-256 (`e9db986bd90127a468202bf0a224608b58ff59150cbdf2c9c87c9d8048c3c682`) |

No changes were made to `.github/workflows/ci.yml`, `.github/workflows/release.yml`, or `.goreleaser.yaml` — every gate above reused the existing pipeline configuration verbatim. The `CGO_ENABLED=0` anti-pattern (RESEARCH D-07) was not used anywhere in this audit.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- REL-01 is fully satisfied: the "no new CGo in Charm" claim is now self-verifying via a committed regression test, not a one-time manual check.
- All pre-existing supply-chain gates (govulncheck, SBOM, reproducibility) confirmed green on the current graph; no blockers for proceeding to remaining Phase 08 plans (08-08, 08-09) or the eventual signed v1.0.0 release.

---
*Phase: 08-surface-reconciliation-signed-v1-0-0-release*
*Completed: 2026-07-19*

## Self-Check: PASSED
- FOUND: internal/cli/present/archtest/charm_cgo_test.go
- FOUND: .planning/phases/08-surface-reconciliation-signed-v1-0-0-release/08-07-SUMMARY.md
- FOUND: commit 3c1fa0a
- FOUND: commit d32b4b2
