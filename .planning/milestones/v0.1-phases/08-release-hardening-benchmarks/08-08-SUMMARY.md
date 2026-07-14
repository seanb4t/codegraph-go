---
phase: 08-release-hardening-benchmarks
plan: 08
subsystem: infra
tags: [github-actions, ci, govulncheck, reproducible-builds, benchmarking, cosign]

# Dependency graph
requires:
  - phase: 08-release-hardening-benchmarks (plan 01)
    provides: ".goreleaser.yaml build config + reproducibility flags (-trimpath, -buildid=, mod_timestamp) that the double-build gate mirrors"
  - phase: 08-release-hardening-benchmarks (plan 07)
    provides: "tools/bench/runner (-mode headtohead / -mode regression) + tools/bench/baseline.json — the harness this plan wires into CI"
provides:
  - ".github/workflows/ci.yml — test + govulncheck + reproducibility double-build gate + perf-regression gate, all on PR/push"
  - ".github/workflows/bench.yml — on-demand/scheduled head-to-head publish, never blocking"
affects: [08-09-docs]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Native reproducibility double-build inline in a CI step (not a separate tool) — mirrors .goreleaser.yaml's exact ldflags/trimpath/mod-time flags so the gate proves what release.yml actually produces is deterministic"
    - "continue-on-error: true on a whole step (not per-command) is the mechanism for D-03's 'blocking on linux/amd64, reported-only on cross-targets' split within a single job"
    - "mapfile + array expansion (not $(...) word-splitting) for go test package-list construction — passes shellcheck/actionlint cleanly"

key-files:
  created:
    - .github/workflows/ci.yml
    - .github/workflows/bench.yml
  modified: []

key-decisions:
  - "govulncheck runs via the pinned golang/govulncheck-action (not a hand-rolled go install step) — the action's own composite steps already checkout+setup-go+install+run, and repo-checkout: false avoids a redundant second checkout"
  - "Reproducibility gate's -X ldflags use a fixed placeholder Version=ci-repro (not a real tag) since ci.yml runs on PR/push, not a release tag — only byte-identity across the two builds in the same job matters, not the version string's real-world accuracy"
  - "Cross-target reproducibility leg covers linux/arm64 via zig cc (not windows or darwin) — it reuses the same ubuntu-latest runner already provisioned for the linux/amd64 blocking leg, avoiding an extra runner just for a report-only check"
  - "bench.yml runs on ubuntu-latest only (not also macos-latest) — matches the RSS-comparability scoping already established by tools/bench (Linux/macOS only, Windows RSS out of scope) while keeping the workflow to a single job"
  - "perf-regression gate passes -ceiling-bytes 4294967296 explicitly (matching the runner's own default) rather than omitting the flag, so the CI-hardware ceiling is self-documenting in the workflow file itself"

patterns-established:
  - "Every Action pin in this repo's workflows is resolved via `gh api repos/<org>/<repo>/git/refs/tags/<tag>` immediately before use, not copied from memory or docs"

requirements-completed: [DIST-03, DIST-04, PERF-01, PERF-02, INDX-06]

coverage:
  - id: D1
    description: "ci.yml runs the full test suite (build + go test ./..., minus internal/daemon) on every PR/push, plus internal/daemon's known-flaky soak tests isolated into their own -count=1 step"
    requirement: "DIST-04"
    verification:
      - kind: other
        ref: "Local dry-run: `go test -run '^TestNoSuchTest$' $(go list ./... | grep -v /internal/daemon)` compiled and ran clean across all 36 non-daemon packages this session"
        status: pass
    human_judgment: false
  - id: D2
    description: "govulncheck ./... runs as its own blocking gate, never combined with SBOM generation into one step"
    requirement: "DIST-03"
    verification:
      - kind: other
        ref: "Local run: `govulncheck ./...` — 0 vulnerabilities found, this session"
        status: pass
    human_judgment: false
  - id: D3
    description: "A real linux/amd64 double-build hash-diff gate blocks the job on mismatch; a linux/arm64 cross-target leg is reported via continue-on-error and never silently folded into the same green check"
    requirement: "DIST-04"
    verification: []
    human_judgment: true
    rationale: "The double-build script's shell logic was authored to mirror .goreleaser.yaml's exact flags and was YAML/actionlint-validated, but the CGo cross-toolchain build (zig cc, GOOS=linux target) cannot be executed on this darwin/arm64 development machine without a Linux CGo toolchain — the gate's actual pass/fail behavior can only be proven by a real GitHub Actions run on ubuntu-latest."
  - id: D4
    description: "The perf-regression gate invokes tools/bench/runner -mode regression against the committed tools/bench/baseline.json with the INDX-06 peak-RSS ceiling, offline against the synthetic corpus"
    requirement: "PERF-02"
    verification:
      - kind: e2e
        ref: "Local run: `go run ./tools/bench/runner -mode regression -baseline tools/bench/baseline.json -ceiling-bytes 4294967296` (the exact command in ci.yml) — 'regression gate passed' against the committed baseline, this session"
        status: pass
    human_judgment: false
  - id: D5
    description: "bench.yml triggers only on workflow_dispatch/schedule (never pull_request/push), installs TS codegraph@1.3.1, and runs the headtohead runner, publishing raw numbers as a job summary + artifact"
    requirement: "PERF-01"
    verification: []
    human_judgment: true
    rationale: "The workflow YAML is actionlint-clean and its trigger/step wiring was verified by static inspection (no pull_request/push trigger present; installs the pinned TS reference; invokes runner -mode headtohead), but actually dispatching the workflow requires a live GitHub Actions run — not exercisable from this local session."
  - id: D6
    description: "Every third-party Action in both workflows is pinned to a full commit SHA (no floating @vN majors)"
    requirement: "DIST-04"
    verification:
      - kind: other
        ref: "grep -nE 'uses:.*@v[0-9]+$' .github/workflows/ci.yml .github/workflows/bench.yml — no matches, this session; all SHAs re-resolved via gh api immediately before use"
        status: pass
    human_judgment: false

# Metrics
duration: 25min
completed: 2026-07-13
status: complete
---

# Phase 08 Plan 08: CI Gate Workflows Summary

**`.github/workflows/ci.yml` wires the vuln gate, a real linux/amd64 double-build hash-diff, and the offline perf-regression gate into every PR/push; `.github/workflows/bench.yml` publishes head-to-head numbers on demand without ever blocking a merge.**

## Performance

- **Duration:** 25 min
- **Started:** 2026-07-13 (this session)
- **Completed:** 2026-07-13
- **Tasks:** 2 completed
- **Files modified:** 2 (both new)

## Accomplishments
- `ci.yml`: 4 jobs — `test` (build + isolated-daemon test split), `govulncheck` (blocking, separate job from any SBOM step), `reproducibility` (linux/amd64 blocking double-build hash-diff mirroring `.goreleaser.yaml`'s own determinism flags; linux/arm64 cross-target leg via zig, reported-only via `continue-on-error`), `perf-regression` (offline `tools/bench/runner -mode regression` against the committed `tools/bench/baseline.json`)
- `bench.yml`: single `headtohead` job, `workflow_dispatch` + weekly `schedule` triggers only (no `pull_request`/`push`), installs Node + TS `codegraph@1.3.1`, builds the Go binary, runs `runner -mode headtohead`, publishes raw JSON to the job summary and as an artifact
- Every third-party Action (checkout, setup-go, setup-node, setup-zig, govulncheck-action, upload-artifact) pinned to a full commit SHA resolved via `gh api` at execution time (not copied from stale docs)
- Both files verified with `actionlint` (0 issues) and YAML-parse validated
- Sanity-verified locally against the real toolchain in this session: `govulncheck ./...` (0 vulnerabilities), the exact `runner -mode regression` invocation from `ci.yml` (passes against the committed baseline), and the daemon-excluded test-package list (compiles clean across all 36 non-daemon packages)

## Task Commits

Each task was committed atomically:

1. **Task 1: ci.yml — test + govulncheck + reproducibility double-build gate + perf-regression gate** - `fcf6794` (feat)
2. **Task 2: bench.yml — on-demand/scheduled head-to-head publish** - `3c44a83` (feat)

**Plan metadata:** (this commit)

## Files Created/Modified
- `.github/workflows/ci.yml` - test/govulncheck/reproducibility/perf-regression gates on every PR/push
- `.github/workflows/bench.yml` - on-demand/scheduled head-to-head publish (non-blocking)

## Decisions Made
- Used the pinned `golang/govulncheck-action` (with `repo-checkout: false`) instead of a hand-rolled `go install govulncheck` step — the action already wraps checkout+setup-go+install+run and is itself pinned to a full SHA, matching the "don't hand-roll" guidance from 08-RESEARCH.md.
- The reproducibility gate's `-X` ldflags use a fixed `Version=ci-repro` placeholder (not a real semver tag) — this gate runs on PR/push, not a release tag, so only byte-identity between the two same-job builds matters, not a real-world-accurate version string.
- Reused the same `ubuntu-latest` runner for both the linux/amd64 blocking leg and the linux/arm64 report-only leg (via `mlugg/setup-zig`, same SHA pin as `release.yml`), avoiding a second runner just for a non-blocking check.
- `bench.yml` runs on `ubuntu-latest` only, consistent with `tools/bench`'s own Linux/macOS-only RSS-comparability scoping (Windows RSS explicitly out of scope per 08-RESEARCH.md Pattern 1).
- Passed `-ceiling-bytes 4294967296` explicitly in the perf-regression step (matching the runner's own default of 4 GiB) so the CI-hardware ceiling is visible directly in the workflow file rather than relying on an implicit default.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Unquoted command substitution triggered an actionlint/shellcheck failure**
- **Found during:** Task 1 (`ci.yml` test job), first `actionlint` run
- **Issue:** `go test $(go list ./... | grep -v '/internal/daemon$')` triggered shellcheck SC2046 (word-splitting risk on unquoted command substitution), causing `actionlint` to exit non-zero — violating the plan's own acceptance criterion that the workflow be actionlint-clean.
- **Fix:** Replaced with `mapfile -t pkgs < <(...)` + `go test "${pkgs[@]}"`, which shellcheck/actionlint accept cleanly.
- **Files modified:** `.github/workflows/ci.yml`
- **Verification:** `actionlint .github/workflows/ci.yml .github/workflows/bench.yml` exits 0 after the fix; local `bash -c` dry-run confirms the package-list construction still selects all 36 non-daemon packages correctly.
- **Committed in:** `fcf6794` (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 bug fix surfaced by the plan's own actionlint acceptance check)
**Impact on plan:** Necessary for the plan's own actionlint-clean acceptance criterion. No scope creep.

## Issues Encountered
- The local development environment lacks Python's `pyyaml` module (only stock Python 3, no pip-installed packages), so the plan's literal `python3 -c "import yaml; ..."` verification command could not run as written. Substituted an equivalent `ruby -ryaml -e "YAML.load_file(...)"` check (Ruby's stdlib YAML parser, always present on this macOS host) plus `actionlint`'s own YAML validation, both of which parse and validate the workflow files successfully. This does not affect what ships — GitHub Actions itself is the authoritative YAML parser for these files in production, and `actionlint` already subsumes a stricter structural check than a bare `yaml.safe_load`.
- The reproducibility gate's cross-toolchain CGo build (zig cc targeting `linux/amd64`/`linux/arm64`) cannot be executed on this darwin/arm64 development machine — no Linux CGo toolchain is available locally. The shell logic was authored to exactly mirror `.goreleaser.yaml`'s reproducibility flags and is YAML/actionlint-validated, but its actual pass/fail behavior (D3 in coverage) can only be proven by a real `ubuntu-latest` GitHub Actions run.

## User Setup Required

None - no external service configuration required. All Action SHA pins were resolved via `gh api` (already-authenticated `gh` CLI); no new secrets or permissions beyond the workflows' own declared `permissions:` blocks are needed.

## Next Phase Readiness
- `ci.yml` and `bench.yml` are ready to run on the next real PR/push and `workflow_dispatch`/schedule trigger respectively — no further wiring needed for DIST-03/DIST-04/PERF-01/PERF-02/INDX-06 to be enforced.
- **Deferred to real CI:** the reproducibility gate's actual cross-compile behavior (both the blocking linux/amd64 leg and the reported linux/arm64 leg) and `bench.yml`'s live `workflow_dispatch` run have not yet been exercised on real GitHub Actions hardware — recommend triggering both once this plan's commits are pushed, before treating DIST-04/PERF-01 as fully proven in production (consistent with STATE.md's existing note that real CI validation of the 6-target release build remains pending).
- Plan 08-09 (docs) can now reference these two workflow files directly when writing `docs/RELEASE.md`'s verify-commands section and `docs/BENCHMARKS.md`'s methodology section.
- No blockers.

---
*Phase: 08-release-hardening-benchmarks*
*Completed: 2026-07-13*

## Self-Check: PASSED
- FOUND: .github/workflows/ci.yml
- FOUND: .github/workflows/bench.yml
- FOUND: .planning/phases/08-release-hardening-benchmarks/08-08-SUMMARY.md
- FOUND: fcf6794 (Task 1 commit)
- FOUND: 3c44a83 (Task 2 commit)
