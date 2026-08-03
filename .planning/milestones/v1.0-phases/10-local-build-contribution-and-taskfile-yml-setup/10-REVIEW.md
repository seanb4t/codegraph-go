---
phase: 10-local-build-contribution-and-taskfile-yml-setup
reviewed: 2026-08-02T00:00:00Z
depth: deep
files_reviewed: 22
files_reviewed_list:
  - .github/actionlint.yaml
  - .github/actions/install-task/action.yml
  - .github/workflows/bench.yml
  - .github/workflows/ci.yml
  - .github/workflows/darwin-toolchain-canary.yml
  - .github/workflows/release-please.yml
  - .github/workflows/release.yml
  - CONTRIBUTING.md
  - Taskfile.yml
  - docs/RELEASE-PROCEDURES.md
  - go.tool-lint.mod
  - go.tool-lint.sum
  - go.tool.mod
  - go.tool.sum
  - internal/bench/metrics.go
  - internal/bench/regression.go
  - internal/bench/regression_test.go
  - internal/upgrade/release_workflow_shape_test.go
  - internal/upgrade/taskfile_shape_test.go
  - tools/bench/BASELINE.md
  - tools/bench/baseline.json
  - tools/bench/cpudiag/main.go
  - tools/bench/runner/main.go
  - tools/bench/runner/main_test.go
findings:
  critical: 0
  warning: 7
  info: 1
  total: 8
status: issues_found
---

# Phase 10: Code Review Report

**Reviewed:** 2026-08-02T00:00:00Z
**Depth:** deep
**Files Reviewed:** 23
**Status:** issues_found

## Summary

This phase is unusually well-guarded: `internal/bench/regression.go`'s frame-descriptor
gating (GOOS/GOARCH, runner, scratch_fs) is exhaustively tested in
`regression_test.go` including every empty-vs-populated boundary the phase context called
out, and `internal/upgrade/{release_workflow_shape_test.go,taskfile_shape_test.go}`
mechanically enforce the single-definition (`task <target>`) property, the
`status:`/`platforms:` ban, the darwin-native-build requirement, and the required-check
name set against a live-verified GitHub ruleset fixture. I could not find a bug in that
core gating logic or a workflow target that silently no-ops.

What I did find is a cluster of **documentation-vs-reality drift** in files that are
themselves part of this phase's deliverable (`CONTRIBUTING.md`,
`docs/RELEASE-PROCEDURES.md`), one internal self-contradiction inside
`docs/RELEASE-PROCEDURES.md` itself, a coverage gap in `darwin-toolchain-canary.yml`'s
path-scoped trigger that undermines the workflow's own stated purpose, an asymmetric input
validation gap in `CheckRegression` (currently masked by upstream guarantees, but present in
a function whose whole design contract is "never misleads, never panics"), and a genuine,
unaddressed gap in vulnerability-scanning coverage for the two isolated tool modfiles this
phase introduced. None of these break a currently-green CI run today, which is exactly why
they are worth surfacing before they calcify.

## Warnings

### WR-01: `docs/RELEASE-PROCEDURES.md` still names the pre-migration runner classes

**File:** `docs/RELEASE-PROCEDURES.md:111-114`
**Issue:** Step 6's `build` bullet reads:

> compiles all 6 `(GOOS,GOARCH)` targets (native darwin matrix via `macos-latest`/Xcode
> clang; zig cross-compilation for `linux/arm64` and both `windows` targets from
> `ubuntu-latest`)

`release.yml`'s actual `build` matrix (lines 66-112, verified against the file directly)
puts linux/windows legs on `namespace-profile-linux-amd64-4x8` and the darwin legs on
`namespace-profile-macos-6x14-tahoe` — neither `ubuntu-latest` nor `macos-latest` appear
anywhere in the current matrix. The doc's own header even records the 10-03 maintainer
checkpoint that made this move. A maintainer following this runbook during an actual
release would be told the wrong runner classes for the pipeline they are about to operate.
**Fix:** Update the bullet to name `namespace-profile-linux-amd64-4x8` and
`namespace-profile-macos-6x14-tahoe`, matching the `build` matrix's `runner:` values.

### WR-02: `docs/RELEASE-PROCEDURES.md` contradicts its own correction about the SLSA provenance subject

**File:** `docs/RELEASE-PROCEDURES.md:122-124` (contradicted by the file's own §6, lines
232-238)
**Issue:** Step 6's `provenance` bullet reads: "runs SLSA3 build provenance
(`slsa-framework/slsa-github-generator`'s generic generator) **over the checksums file**,
producing an `.intoto.jsonl` attestation." This is precisely the wording `release.yml`'s own
header comment (lines 208-219) calls out as wrong and says "propagated into ... `docs/RELEASE-PROCEDURES.md` as an
instruction to verify the checksums file, which fails against every valid release." This
file's own §6 (lines 232-238) documents the correction — provenance is attested over the
**six binaries**, with the checksums file only as a transport encoding for the subject list
— but the earlier §4 walkthrough was never updated to match, so the same wrong claim still
appears twice in this one file: once corrected, once not.
**Fix:** Change the step-6 bullet to: "runs SLSA3 build provenance ... over the six release
binaries (the checksums file only carries the subject list, per §6's correction below)."

### WR-03: `docs/RELEASE-PROCEDURES.md` and the repo's own required-checks fixture disagree about whether `main` has a ruleset

**File:** `docs/RELEASE-PROCEDURES.md:408-411`
**Issue:** "**Branch protection note:** the repo has no rulesets today, so this does not
currently apply." But `internal/upgrade/taskfile_shape_test.go`'s `requiredCheckNames`
fixture (lines 36-51) is sourced from "`gh api repos/seanb4t/codegraph-go/rulesets/20157557`,
re-verified live 2026-08-01" — i.e., a ruleset with ID 20157557 already exists and already
enforces required status checks — and `CONTRIBUTING.md` independently states "`main` is
protected; merges are **squash-only** and linear history is enforced," which are exactly the
kind of rules a ruleset configures. Either the "no rulesets today" claim is stale, or the doc
means something narrower ("no ruleset that blocks direct pushes specifically") that it never
states. Left as written, a maintainer reading this section would wrongly conclude the App
doesn't need bypass-actor consideration, when a live ruleset already governs this repo.
**Fix:** Reconcile the claim against `gh api repos/seanb4t/codegraph-go/rulesets/20157557`
(and any other active rulesets) and state precisely which specific protections (if any)
would block release-please's direct push, rather than asserting no ruleset exists at all.

### WR-04: `CONTRIBUTING.md` undercounts the required status checks by one

**File:** `CONTRIBUTING.md:46-48`
**Issue:** "All six required checks pass: `test`, `actionlint`,
`govulncheck (DIST-03, blocking)`, `perf regression gate (PERF-02, INDX-06)`, `pr-title`,
`reproducibility (double-build hash-diff, DIST-04)`." This list omits
`goreleaser check (config validation, DIST-01)` — the seventh job registered in `ci.yml`
(lines 323-345) and one of the entries `internal/upgrade/taskfile_shape_test.go`'s own
`requiredCheckNames` fixture asserts is a live, GitHub-ruleset-enforced required context
(confirmed by `.planning/phases/10-local-build-contribution-and-taskfile-yml-setup/10-01-PLAN.md:181`,
which lists all six ruleset contexts by name including `goreleaser check (config
validation, DIST-01)`). A contributor reading only `CONTRIBUTING.md` would not know this
check exists or gates their merge.
**Fix:** Add `goreleaser check (config validation, DIST-01)` to the list and correct "six"
to "seven."

### WR-05: `darwin-toolchain-canary.yml`'s path filter omits the two files its own steps depend on

**File:** `.github/workflows/darwin-toolchain-canary.yml:26-35`
**Issue:** The workflow's header comment states its `pull_request` trigger exists to
"re-prove the toolchain automatically whenever something that can affect the darwin build
changes." Its only job step besides checkout/setup-go is `uses: ./.github/actions/install-task`
followed by `task check:darwin-toolchain`. The `paths:` filter lists
`.github/workflows/release.yml`, `.github/workflows/darwin-toolchain-canary.yml`,
`.goreleaser.yaml`, `Taskfile.yml`, `go.mod`, and `go.sum` — but omits
`.github/actions/install-task/action.yml` (the composite action this workflow directly
`uses:`) and `go.tool.mod`/`go.tool.sum` (the modfile that action builds `task` from). A
change to either — e.g., a `task` version bump in `go.tool.mod`, or a change to how
`install-task` builds it — would not re-trigger this canary on a PR, defeating the stated
purpose of the path-scoped trigger for exactly the class of change most likely to break the
toolchain bootstrap this canary exists to protect.
**Fix:** Add `.github/actions/install-task/action.yml`, `go.tool.mod`, and `go.tool.sum` to
the `paths:` list.

### WR-06: `CheckRegression` validates `baseline`'s numeric fields but not `current`'s

**File:** `internal/bench/regression.go:105-133`
**Issue:** Lines 105-110 reject a degenerate `baseline.FilesPerSec <= 0` or
`baseline.PeakRSSBytes <= 0` with a clear error. `current`'s equivalent fields are never
validated. If `current.PeakRSSBytes` is ever `0` or negative (e.g., a caller-constructed
`Metrics`, or a future measurement-path bug), the relative RSS check at line 120
(`rssDelta := float64(current.PeakRSSBytes-baseline.PeakRSSBytes) / float64(baseline.PeakRSSBytes)`)
produces a negative delta that trivially clears `DefaultRSSTolerance`, and the absolute
ceiling check at line 128 (`current.PeakRSSBytes > ceilingBytes`) is likewise false for a
non-positive value — so a broken RSS measurement would **silently pass** both the relative
and the absolute (INDX-06) RSS gates instead of failing loud, exactly the failure mode this
package's doc comment (lines 29-35: "CheckRegression ... never panics ... returns a plain
error instead of dividing by zero") claims to guard against, just on the other side of the
comparison. In the current call chain this is masked because
`internal/bench.PeakRSSBytes` (`rss.go`) always returns an error rather than `0` on failure,
so it's not reachable today via `tools/bench/runner` — but `CheckRegression` is a public
function whose only defense is its own body, and a degenerate `current` is exactly as
plausible a caller mistake as a degenerate `baseline`.
**Fix:** Add symmetric guards: `if current.FilesPerSec <= 0 { return fmt.Errorf(...) }` and
`if current.PeakRSSBytes <= 0 { return fmt.Errorf(...) }`, mirroring the existing baseline
checks.

### WR-07: The tool modfiles this phase introduced are not covered by any vulnerability scan

**File:** `go.tool.mod`, `go.tool-lint.mod`, `.github/workflows/ci.yml:156-171`
**Issue:** `go.tool.mod` pulls in ~400 modules (task, goreleaser, and their transitive
dependency trees — AWS/GCP/Azure SDKs, Kubernetes client libraries, cosign, sigstore, etc.);
`go.tool-lint.mod` pulls in actionlint's tree. Both are built from source and **executed as
trusted CI tooling** — `goreleaser` signs and publishes releases, `task` drives every CI job
body. `ci.yml`'s blocking `govulncheck (DIST-03, blocking)` job (lines 156-171) runs
`golang/govulncheck-action` against the root `go.mod` only; the `Taskfile.yml` `vuln` target
(the only thing that ever points govulncheck at `go.tool.mod`) is explicitly documented as
"a LOCAL convenience target only" (`go.tool.mod:10-15`) and is never invoked by any CI job.
`go.tool-lint.mod` has no vulnerability-scanning path at all, local or CI. This phase's own
research (`.planning/phases/10-.../10-RESEARCH.md:1012`) reasoned about this tradeoff only
from the angle of "a compromised transitive dependency ... cannot reach the release binary's
own SBOM/govulncheck surface" — a real and valid point about the *shipped binary*, but it
does not address the separate risk that a known-vulnerable dependency inside `go-task` or
`goreleaser` itself is executed, unscanned, in every CI run and in the credentialed release
pipeline (cosign signing, `gh release` publishing).
**Fix:** At minimum, run `task vuln` (or `govulncheck -modfile=go.tool.mod`) as a
non-blocking, reported CI step so a vulnerability in the executed tooling is visible; ideally
promote it to a blocking gate now that `go.tool.mod`/`go.tool-lint.mod` are stable, isolated
modules with no impact on the release binary's own SBOM.

## Info

### IN-01: `test:unit`'s daemon-exclusion filter only matches the exact package path

**File:** `Taskfile.yml:55-57`
**Issue:** `grep -v '/internal/daemon$'` excludes exactly the package whose import path ends
in `/internal/daemon`; it would not exclude a hypothetical future `internal/daemon/foo`
subpackage. `test:daemon` (line 84) is likewise non-recursive
(`go test ./internal/daemon/ -count=1`, no `...`). Today `internal/daemon` has no
subdirectories, so this is harmless, but if one is ever added, its tests would land in
neither the isolated `-count=1` daemon step nor be excluded from `test:unit`'s
full-parallel-load run — silently reopening the exact flakiness this isolation exists to
prevent, with no error or warning at the point the subpackage is added.
**Fix:** Either note this constraint in the `test:daemon`/`test:unit` `desc:` fields, or
change both to operate on `./internal/daemon/...` consistently so a future subpackage is
automatically covered by both mechanisms (excluded from one, isolated in the other).

---

_Reviewed: 2026-08-02T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
