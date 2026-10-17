---
phase: 01-changie-baseline
reviewed: 2026-09-25T20:39:54Z
depth: deep
files_reviewed: 23
files_reviewed_list:
  - .changes/header.tpl.md
  - .changes/unreleased/.gitkeep
  - .changes/v0.10.0.md
  - .changes/v0.11.0.md
  - .changes/v0.12.0.md
  - .changes/v0.13.0.md
  - .changes/v0.14.0.md
  - .changes/v0.2.0.md
  - .changes/v0.3.0.md
  - .changes/v0.4.0.md
  - .changes/v0.5.0.md
  - .changes/v0.5.1.md
  - .changes/v0.6.0.md
  - .changes/v0.7.0.md
  - .changes/v0.8.0.md
  - .changes/v0.9.0.md
  - .changie.yaml
  - .github/workflows/ci.yml
  - CONTRIBUTING.md
  - Taskfile.yml
  - go.tool-changie.mod
  - go.tool-changie.sum
  - internal/upgrade/changie_shape_test.go
  - internal/upgrade/taskfile_shape_test.go
findings:
  critical: 0
  warning: 1
  info: 2
  total: 3
status: issues_found
---

# Phase 01: Code Review Report

**Reviewed:** 2026-09-25T20:39:54Z
**Depth:** deep
**Files Reviewed:** 23
**Status:** issues_found

## Summary

Reviewed the changie baseline: the pinned `go.tool-changie.mod`/`.sum`, `.changie.yaml`, the 14 verbatim `.changes/v*.md` seed files plus `header.tpl.md`/`unreleased/.gitkeep` (per phase scope, treated as locked byte-slices — not evaluated for release-please formatting), the `Taskfile.yml` diff (`GO_TOOL_CHANGIE` var, `changie` wrapper task, the 11-leg `check:changie` guard, and the `vuln` task's changie addition), the one new CI step in `ci.yml`, the `CONTRIBUTING.md` prose update, and the two shape-test files (`changie_shape_test.go`, `taskfile_shape_test.go`).

Verification performed, not just read-through:
- Ran `task check:changie` live — all 11 legs pass, scratch-only, source tree checksum-verified unchanged.
- Ran `GOWORK=off go test ./internal/upgrade/...` — all changie- and taskfile-shape guards pass.
- Independently re-measured the `vuln` task's module-count claims (`go list -m -modfile=... all`, excluding each modfile's own main-module line) for both the four-modfile baseline (1257) and the five-modfile total with changie unioned in (1261, +4 net-new) — both numbers in the task's `desc:` are exactly reproducible, not fabricated.
- Confirmed `.changes/*.md` + `header.tpl.md` concatenate to `CHANGELOG.md` byte-for-byte modulo the one `newlines.afterChangelogHeader: 1` blank line the config declares, and that `CHANGELOG.md` itself is untouched by this diff.
- Confirmed `check:changie` is wired immediately after `docs:cli:drift` in `ci.yml`'s `test` job, matching D-08 and the guard test.

No Critical findings. One Warning: `check:changie`'s only use of `sort -V` (GNU/newer-BSD version-sort) has no `command -v`/capability precondition, unlike this same task's existing `go`/`cmp` preconditions, on a project that explicitly supports macOS as a first-class local dev target and whose own Taskfile convention (`crossToolchainTokens`) is to gate anything with an unverified environment dependency. Two Info items on documentation completeness/duplication, not correctness.

## Warnings

### WR-01: `check:changie` leg 1 relies on `sort -V` with no capability guard

**File:** `Taskfile.yml:621`
**Issue:** The baseline-floor check does:
```sh
lowest=$(printf '%s\n' v0.14.0 "${actual}" | sort -V | head -n1)
if [ "${lowest}" != "v0.14.0" ]; then
  echo "::error::check:changie: [1/11] changie latest = ${actual} fell below the v0.14.0 baseline"
  exit 1
fi
```
`-V` (natural/version sort) is a GNU coreutils extension; it is not present in traditional BSD `sort`. It happens to work on the current macOS host (a recent BSD `sort` build that has picked up `-V`), and CI's `test` job runs on Linux (`namespace-profile-linux-amd64-4x8`), so this is invisible in CI today. But it is the only use of `sort -V` anywhere in `Taskfile.yml`, it has no precedent elsewhere to lean on, and — unlike this same task's `command -v go` / `command -v cmp` preconditions (which fail with an actionable `msg:`) — a contributor on an older macOS or minimal container image without a version-sort-capable `sort` gets an opaque `sort: illegal option -- V` under `set -euo pipefail`, not a clear precondition failure. This project's own convention (see `crossToolchainTokens`/preconditions pattern in `internal/upgrade/taskfile_shape_test.go`) is to gate unverified environment capabilities with an actionable message rather than let them fail bare.
**Fix:** Either add a `command -v sort` + version-sort capability precondition (e.g. probe `printf 'v1\nv2\n' | sort -V >/dev/null 2>&1` and fail with an actionable `msg:` if it errors), or replace the version comparison with a portable, dependency-free one (e.g. split `major.minor.patch` on `.` and compare numerically in bash, the same way leg 10/11 already decompose `${actual}` for arithmetic).

## Info

### IN-01: Duplicated before/after checksum snapshot logic

**File:** `Taskfile.yml:604, 806`
**Issue:** The exact same pipeline —
```sh
find .changie.yaml .changes CHANGELOG.md -type f -exec cksum {} + | LC_ALL=C sort
```
— appears twice, once to compute `before` and once to compute `after`, with no shared variable/function. A future edit to the tracked-path set (e.g. adding another file that must stay byte-unchanged) only needs to be made in one place today, but if either copy drifts independently in a future edit, the checksum comparison could silently start comparing different scopes on each side.
**Fix:** Factor the pipeline into a small shell function (`snapshot() { find ... ; }`) called for both `before` and `after`, matching the `fragment_count()` helper pattern already used later in the same task.

### IN-02: `CONTRIBUTING.md`'s tool-modfile sentence is incomplete after this edit

**File:** `CONTRIBUTING.md:124-130`
**Issue:** The updated sentence reads "`task`, `goreleaser`, `actionlint`, and `changie` build on demand from `go.tool.mod`, `go.tool-lint.mod`, and `go.tool-changie.mod`" — this omits `buf`/`protoc-gen-go`/`protoc-gen-connect-go` (`go.tool-proto.mod`) and `golangci-lint` (`go.tool-golangci.mod`), both of which already existed before this phase. The omission predates this phase (the prior sentence was already incomplete, listing only two of what were then four modfiles), but this phase touched this exact sentence to add `changie` and did not correct the pre-existing gap while editing it, so the documentation drift is now wider (missing 2 of 5 modfiles instead of 2 of 4).
**Fix:** While already editing this sentence, list all five tool modfiles (`go.tool.mod`, `go.tool-lint.mod`, `go.tool-proto.mod`, `go.tool-golangci.mod`, `go.tool-changie.mod`) or phrase it generically ("every `go.tool*.mod` file") so it doesn't need updating again on the next tool addition.

---

_Reviewed: 2026-09-25T20:39:54Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
