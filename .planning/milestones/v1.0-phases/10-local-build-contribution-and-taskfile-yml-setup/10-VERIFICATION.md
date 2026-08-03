---
phase: 10-local-build-contribution-and-taskfile-yml-setup
verified: 2026-08-02T21:30:00Z
status: passed
score: 3/3 roadmap success criteria verified (all DEV-01 sub-must-haves also verified)
behavior_unverified: 0
overrides_applied: 0
human_verification:

  - test: "Push a real v[0-9]* tag and watch release.yml's `build` job end-to-end for the two darwin matrix legs (goos=darwin, goarch=arm64/amd64), specifically the goreleaser invocation, cosign signing, and SLSA attestation steps that run on top of the plain `go build` the canary already proves."
    expected: "The `build` job's goreleaser step produces both signed darwin binaries with release ldflags, the `assemble`/`provenance` steps attest them correctly, and (ideally) a `codegraph upgrade` smoke test succeeds against the resulting macOS binary on a real Mac."
    why_human: "`darwin-toolchain-canary.yml` (added by this phase, 10-03) already machine-proves the two riskiest components of D-08 on the SAME `namespace-profile-macos-6x14-tahoe` label release.yml uses: (1) Namespace serves the label rather than queuing it indefinitely, and (2) the host is a genuine native Apple Silicon toolchain, not a substituted cross-build. Verified directly this session: `gh run list --workflow=darwin-toolchain-canary.yml` shows three successful runs, including one at the CURRENT HEAD (a61ccf1, 2026-08-02T19:17:11Z); that run's log shows `Apple clang version 21.0.0`, `Target: arm64-apple-darwin25.6.0` (native arm64, not zig), and both `CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build ./cmd/codegraph` and `GOARCH=amd64 go build ./cmd/codegraph` completing successfully. What remains genuinely unexercised is narrower than 'the darwin leg': the canary runs a plain `go build -o /dev/null`, not goreleaser — so release.yml's actual darwin build path (goreleaser with release ldflags, cosign signing, SLSA attestation) has never run on this runner class, no `codegraph upgrade` smoke test has been performed against a resulting binary, and `release.yml`'s `on: push: tags:` trigger means the assembled job as a whole still first runs during a real release. This residual gap still warrants a human check at the next tag push, even though runner-availability and native-toolchain risk are already discharged."
---

# Phase 10: Local Build Tooling & CONTRIBUTING Verification Report

**Phase Goal:** Contributor-facing local dev tooling — the repo has no `Makefile`/`Taskfile`/`scripts/`. Add a `Taskfile.yml` (go-task) wrapping the common local workflows plus a `CONTRIBUTING.md` documenting the CGo toolchain prerequisites, so a new contributor can build/test/lint from a clean checkout without reverse-engineering the CI workflows.
**Verified:** 2026-08-02T21:30:00Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (ROADMAP Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `Taskfile.yml` wraps all 9 named command families (build w/ release ldflags, test w/ daemon-flake isolation, `-race`, `go vet`, `govulncheck`, `actionlint`, `goreleaser check`, bench runner modes, cross-`GOOS` pre-tag sweep) | ✓ VERIFIED | Every family has a real, non-stub target body (`build:release`, `test:daemon`, `test:race`, `vet`, `vuln`, `lint:actions`, `check:goreleaser`, `bench:regression`, `check:cross`). Ran each locally this session: `task build` (exit 0), `task build:release` (exit 0, produced a version-stamped 62MB binary), `task vet` (exit 0), `task vuln` (exit 0, govulncheck ran against `go.tool.mod`-built binary, 0 code vulnerabilities), `task lint:actions` (exit 0, actionlint built cold from `go.tool-lint.mod`), `task check:goreleaser` (exit 0), `task check:cross` (exit 0, 6-target sweep). `task test:daemon`/`task test:race` already confirmed green by the orchestrator's pre-verification regression gate. Only the two *write* bench modes (`headtohead`, `-rebless`) stay outside a task target — this is a **documented D-01 exception**, confirmed in `bench.yml`'s own in-file comment (lines 159-170) and `10-CONTEXT.md`: keeping `-rebless` unreachable from tab-completion is the deliberate mitigation for the historical wrong-platform-baseline incident. `bench:regression` (the read-only gate invocation) IS a real task target (D-14). |
| 2 | `CONTRIBUTING.md` documents the CGo toolchain prerequisites (zig for cross-builds, mingw-w64 for windows vet) | ✓ VERIFIED (pre-satisfied, per D-00) | `CONTRIBUTING.md` lines 57-96 name both `zig` and `mingw-w64` explicitly, plus a pointer to `PARSER-DECISION.md`. This was already true before Phase 10 began (OSS-readiness work, D-00) — Phase 10 added only a pointer paragraph (lines 75-96) directing contributors at the task targets, and did not need to (and did not) rewrite the pre-existing toolchain prose. Recording as pre-satisfied per the phase's own context, not manufacturing credit. |
| 3 | A clean checkout can build, test, and lint via task targets alone | ✓ VERIFIED | Build leg: `task build` and `task build:release` both exit 0 locally. Test leg: orchestrator's pre-verification regression gate — `task test` exit 0, all five legs green (unit/golden/integration/daemon 6.8s/race 15.6s+4.3s+8.4s), 50 packages, zero FAIL/panic/DATA RACE. Lint leg: `task vet` (exit 0) and `task lint:actions` (exit 0, cold-built actionlint from the isolated tool modfile) both run this session — together these are exactly what `task lint`'s wrapper invokes. **Additionally**, a real, live GitHub Actions run on this branch's current HEAD (`a61ccf1`, checked via `gh run list`/`gh run view`) shows all six `ci.yml` gating jobs green: `test`, `govulncheck (DIST-03, blocking)`, `actionlint (workflow static analysis)`, `goreleaser check (config validation, DIST-01)`, `perf regression gate (PERF-02, INDX-06)`, `reproducibility (double-build hash-diff, DIST-04)` — all invoking `task <target>`, all scheduled and completed on the Namespace runner classes this phase moved them to. This resolves the "not yet observed on a real pushed CI event" caveat both 10-01-SUMMARY.md and 10-02-SUMMARY.md explicitly carried forward as an open concern. |

**Score:** 3/3 roadmap success criteria verified (0 present-but-behavior-unverified)

### DEV-01 Sub-Must-Haves (merged from the 7 plans' frontmatter, deduplicated against the roadmap criteria above)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 4 | CI workflow `run:` bodies invoke `task <target>` rather than re-declaring command bodies (single-definition property, D-01/D-02) | ✓ VERIFIED | `TestWorkflowRunBodiesInvokeTask` (`internal/upgrade/taskfile_shape_test.go`) parses the real, on-disk YAML for `ci.yml` (jobs: test, actionlint, goreleaser-check, reproducibility, perf-regression) and `release-please.yml` (job: pretag-gate) and asserts every step's `run:` body is exactly `task <target>`, with exactly two named, reasoned exceptions (an apt-get install step, a `$GITHUB_OUTPUT`-writing step). Ran it directly this session: PASS. Manually re-inspected `ci.yml` and `release-please.yml` line-by-line — confirms the same. |
| 5 | No task target uses `status:`/`platforms:` (silent-skip fields); cross-toolchain targets gate via `preconditions:` with actionable `msg:` (D-11) | ✓ VERIFIED | `TestTaskfileGatesFailLoud` — PASS. Manual read of `Taskfile.yml` confirms `vet:daemon-windows` and `check:darwin-toolchain` both use `preconditions:` with an install-instruction `msg:`; no target anywhere uses `status:` or `platforms:`. |
| 6 | Two isolated tool modfiles (`go.tool.mod`, `go.tool-lint.mod`) keep build tools out of root `go.mod` (D-03) | ✓ VERIFIED | `TestToolModfilesRemainIsolated` — PASS. Both files exist, are distinct, root `go.mod` declares no `tool` directive and requires none of `task`/`goreleaser`/`actionlint` directly. |
| 7 | `check:cross`'s 6-target sweep is set-equal to `.goreleaser.yaml`'s build matrix (D-15) | ✓ VERIFIED | `TestCheckCrossMatchesGoreleaserTargets` — PASS. `release-please.yml`'s `pretag-gate` job now calls `task check:cross`, confirmed by direct read (no inline sweep remains). |
| 8 | The runner/scratch_fs perf-gate blind spot is closed (D-09) — a regression check refuses a runner or scratch_fs mismatch as a category error, before any numeric comparison | ✓ VERIFIED | `go test ./internal/bench/... -run TestCheckRegression -v` — all 24 subtests PASS, including every empty-vs-populated and mismatch boundary named in the plan (runner mismatch, scratch_fs mismatch, both refused even when GOOS/GOARCH match). Matches the orchestrator's note that a reviewer adversarially probed this gate and could not break it. |
| 9 | Requirement ID minted; ROADMAP and REQUIREMENTS.md both traceable (D-16) | ✓ VERIFIED | `ROADMAP.md` line 432: `**Requirements**: DEV-01` (not `TBD`). `REQUIREMENTS.md` line 95: full DEV-01 entry marked `[x]`; line 185: `\| DEV-01 \| Phase 10 \| Complete \|`; line 198: dated note recording the minting rationale and the D-00 pre-satisfaction of criterion 2. |
| 10 | Darwin release leg (D-08) stays a real native macOS build — never substituted with a zig cross-link | ✓ VERIFIED (runner-availability + native-toolchain); ⚠️ residual gap (goreleaser/signing/upgrade-smoke) | `TestDarwinLegsBuildNatively` (`internal/upgrade/release_workflow_shape_test.go`) — PASS on the YAML text. Beyond the shape test, `darwin-toolchain-canary.yml` (added by this phase, 10-03) machine-proves the two riskiest live-behavior components on the exact same `namespace-profile-macos-6x14-tahoe` label: `gh run list --workflow=darwin-toolchain-canary.yml` shows 3 successful runs, the most recent at the current HEAD (`a61ccf1`); that run's log shows `Apple clang version 21.0.0`, `Target: arm64-apple-darwin25.6.0` (native, not a zig cross-link), and both `CGO_ENABLED=1 GOOS=darwin GOARCH=arm64/amd64 go build ./cmd/codegraph` completing successfully. So Namespace serving the profile (rather than queuing indefinitely) and the toolchain being genuinely native are both machine-observed, not merely asserted. What remains unproven is narrower: the canary builds with plain `go build`, not goreleaser — so goreleaser's own darwin build path, cosign signing, and SLSA attestation on this runner class, plus a `codegraph upgrade` smoke test, are still unexercised until a real release tag is pushed. See Human Verification below. |

**Score:** 10/10 sub-must-haves present-and-wired-verified; #10's runner-availability and native-toolchain components are now live-verified via the darwin-toolchain-canary; its narrower residual (goreleaser/signing/upgrade-smoke on that runner) is what routes to human verification, pending a real tag push.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `Taskfile.yml` | 9 command families + 2 contributor wrappers | ✓ VERIFIED | 515 lines, all targets substantive, no stubs; every named family executed successfully this session or in live CI. |
| `CONTRIBUTING.md` | Toolchain prereqs + task-target pointer | ✓ VERIFIED | Pre-existing prereqs (D-00) + new pointer paragraph (lines 75-96), confirmed by direct read. |
| `go.tool.mod` / `go.tool.sum` | Isolated task+goreleaser tool module | ✓ VERIFIED | Exists, isolated per `TestToolModfilesRemainIsolated`; `task --version` and `goreleaser check` both ran successfully from it this session. |
| `go.tool-lint.mod` / `go.tool-lint.sum` | Isolated actionlint tool module | ✓ VERIFIED | Exists, isolated; `actionlint` ran successfully from it this session (cold build). |
| `.github/actions/install-task/action.yml` | Composite action bootstrapping `task` from the module proxy | ✓ VERIFIED | Present, used by every CI job that calls `task`; confirmed live-green in `gh run view` for the current HEAD's `ci.yml` run. |
| `.github/workflows/ci.yml` | Rewired to `task <target>` calls, Namespace runners | ✓ VERIFIED | Confirmed by direct read and `TestWorkflowRunBodiesInvokeTask`; live-green in a real GitHub Actions run at `a61ccf1`. |
| `.github/workflows/release-please.yml` | `pretag-gate` calls `task check:cross` | ✓ VERIFIED | Confirmed by direct read; stays on `ubuntu-latest` per D-15's explicit scope note. |
| `.github/workflows/release.yml` | Namespace runners (linux/windows legs, darwin native, SLSA job unmoved) | ✓ VERIFIED (structure + runner-availability/native-toolchain via canary) / ⚠️ residual gap (goreleaser/signing/upgrade-smoke unexercised) | Shape-tested (`TestDarwinLegsBuildNatively`, `TestProvenanceJobUsesTaggedSLSAGenerator`, both PASS) and manually confirmed by direct read; no `run:` body changed (only runner classes/cache steps, matching the plan's own prohibition). `darwin-toolchain-canary.yml` machine-proves Namespace serves `namespace-profile-macos-6x14-tahoe` as a genuine native host (3 real runs, most recent at current HEAD). The narrower residual — goreleaser's actual darwin build/sign/attest path — is the one open human-verification item (see above). |
| `.github/workflows/bench.yml` | Namespace for headtohead; `ubuntu-latest` for rebless (documented D-06 deviation) | ✓ VERIFIED | Confirmed by direct read of the file's own extensive in-file rationale comments; `gh run list --workflow=bench.yml` shows 5 recent real runs on this branch, all green. |
| `internal/upgrade/taskfile_shape_test.go` | Single-definition + gate-fail-loud guards | ✓ VERIFIED | 1178 lines; ran directly, all 12 named tests PASS. |
| `internal/upgrade/release_workflow_shape_test.go` | D-07/D-08 guards | ✓ VERIFIED | Ran directly, all named tests PASS. |
| `internal/bench/regression.go` | Runner/scratch_fs category-error checks | ✓ VERIFIED | `TestCheckRegression`'s 24 subtests PASS, including every runner/scratch_fs boundary. |
| `tools/bench/baseline.json` | Non-stale, `runner`-carrying baseline | ✓ VERIFIED | `runner: "ubuntu-latest"`, `scratch_fs: "disk"` both populated; written by the documented `-rebless` path per 10-04-SUMMARY.md's provenance trail. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `ci.yml` job `test` | `Taskfile.yml` targets | `task build`/`vet`/`test:unit`/`test:golden`/`test:integration`/`test:daemon`/`test:race`/`vet:windows`/`vet:daemon-windows` | ✓ WIRED | Confirmed by direct read + live green run. |
| `ci.yml` job `actionlint` | `task lint:actions` | `{{.GO_TOOL_LINT}} actionlint` | ✓ WIRED | Confirmed by direct read + live green run + local execution. |
| `ci.yml` job `goreleaser-check` | `task check:goreleaser` | `{{.GO_TOOL}} goreleaser check` | ✓ WIRED | Confirmed by direct read + live green run + local execution. |
| `ci.yml` job `perf-regression` | `task bench:regression` | `go run ./tools/bench/runner -mode regression ...` | ✓ WIRED | Confirmed by direct read + live green run. |
| `ci.yml` job `reproducibility` | `task check:reproducibility` / `task check:reproducibility:arm64` | double-build hash-diff | ✓ WIRED | Confirmed by direct read + live green run. |
| `release-please.yml` job `pretag-gate` | `task check:cross` | 6-target `go list -mod=readonly` sweep | ✓ WIRED | Confirmed by direct read; runs on every push to `main` (not tag-gated), so it has been exercised on `main` pushes even though `release.yml` itself has not. |
| `release.yml` `build` matrix | `namespace-profile-linux-amd64-4x8` (linux/windows) / `namespace-profile-macos-6x14-tahoe` (darwin) | `runner:` matrix column | ✓ WIRED (structure) / ✓ VERIFIED live (runner-availability + native toolchain, via `darwin-toolchain-canary.yml` on the same label) / ⚠️ UNPROVEN (goreleaser build/sign/attest path specifically) | Linux/windows legs share the same Namespace label already proven green elsewhere in this phase's CI; the darwin label's scheduling and native-toolchain behavior are proven by the canary (3 real runs, most recent at current HEAD), but goreleaser itself has not yet run on it. |
| `release.yml` `provenance` job | `slsa-framework/slsa-github-generator/.github/workflows/generator_generic_slsa3.yml@v2.1.0` | reusable workflow (D-07, cannot override `runs-on`) | ✓ WIRED | Confirmed unchanged by direct read + `TestProvenanceJobUsesTaggedSLSAGenerator`. |
| `.github/actions/install-task` | `go.tool.mod` | `GOWORK=off go build -modfile=go.tool.mod ... github.com/go-task/task/v3/cmd/task` | ✓ WIRED | Confirmed by direct read; no `arduino/setup-task` or `taskfile.dev` install script anywhere under `.github/`. |
| `.github/workflows/darwin-toolchain-canary.yml` | `namespace-profile-macos-6x14-tahoe` | `runs-on:`, `task check:darwin-toolchain` | ✓ VERIFIED live | 3 real successful runs (`gh run list`), most recent at current HEAD `a61ccf1`; log confirms native Apple clang (arm64-apple-darwin) and both darwin arches CGo-building successfully. |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|---|---|---|---|---|
| DEV-01 | 10-01..10-07 (all 7) | Single local entry point for every CI-enforced command, exactly one definition, CONTRIBUTING pointer | ✓ SATISFIED | ROADMAP.md and REQUIREMENTS.md both trace it; all constituent sub-must-haves verified above, with the darwin release build's goreleaser/signing/upgrade-smoke path routed to human verification pending the next real tag push (not a failure — the runner-availability and native-toolchain risk that would have made this a failure is already live-verified via the canary). |

No orphaned requirements found — `REQUIREMENTS.md`'s "Phase 10" grep matches exactly DEV-01, which is claimed by all 7 plans' frontmatter.

### Anti-Patterns Found

No debt markers (`TBD`/`FIXME`/`XXX`) found in any phase-modified file (`Taskfile.yml`, `CONTRIBUTING.md`, both tool modfiles, the composite action, the four rewired workflow files, both shape-test files, `internal/bench/regression.go`, `internal/bench/metrics.go`, `tools/bench/runner/main.go`, `tools/bench/baseline.json`, `tools/bench/BASELINE.md`, `docs/RELEASE-PROCEDURES.md`).

Independent code review (`10-REVIEW.md`, 22 files, deep) found 0 critical issues and 7 warnings — none of which block the DEV-01 goal, all of which are legitimate follow-up items:

| File | Issue | Severity | Impact on this phase's goal |
|------|-------|----------|------------------------------|
| `docs/RELEASE-PROCEDURES.md:111-114` | Still names pre-migration runner classes (`macos-latest`/`ubuntu-latest`) | Warning | Doc drift only — `release.yml` itself is correct; does not affect DEV-01. |
| `docs/RELEASE-PROCEDURES.md:122-124` | Repeats a stale SLSA-provenance-subject claim its own §6 already corrected | Warning | Doc drift, pre-existing content this phase did not touch. |
| `docs/RELEASE-PROCEDURES.md:408-411` | Disagrees with the repo's own live ruleset fixture about whether `main` has branch protection | Warning | Doc drift, pre-existing content. |
| `CONTRIBUTING.md:46-48` | Lists 6 required checks, omits the 7th (`goreleaser check`) | Warning | Minor undercounting in the phase's own deliverable — does not affect whether the checks exist or run; worth a follow-up fix. |
| `darwin-toolchain-canary.yml:26-35` | `paths:` filter omits `install-task/action.yml` and `go.tool.mod`/`go.tool.sum` | Warning | Weakens (does not break) the canary's own trigger scope for future changes to the tool bootstrap; does not affect this verification's use of the canary's already-observed, already-green runs as evidence. |
| `internal/bench/regression.go:105-133` | `CheckRegression` validates `baseline`'s numeric fields but not `current`'s | Warning | Latent, currently-unreachable defect in a defense-in-depth check; does not affect any of this phase's roadmap criteria. |
| `go.tool.mod`/`go.tool-lint.mod` | No vulnerability scan covers the ~400 modules these tool modfiles pull in | Warning | Real gap, but outside the literal wording of criterion 1 (which asks that `govulncheck` be wrapped for wrapping's sake, not that the tool modfiles themselves be scanned). Worth a follow-up. |

None of these warrant a gap in this verification — they are pre-existing or adjacent documentation/coverage issues, not failures of the phase's own must-haves.

### Human Verification Required

1. **Darwin release build's goreleaser/signing/upgrade-smoke path (D-08 residual)**
   **Test:** Push a real `v[0-9]*` tag and watch `release.yml`'s `build` job end-to-end for the two darwin matrix legs, specifically the goreleaser invocation, cosign signing, and SLSA attestation steps that run on top of the plain `go build` `darwin-toolchain-canary.yml` already proves.
   **Expected:** The `build` job's goreleaser step produces both signed darwin binaries with release ldflags, `assemble`/`provenance` attest them correctly, and (ideally) a `codegraph upgrade` smoke test succeeds against the resulting macOS binary on a real Mac.
   **Why human:** `darwin-toolchain-canary.yml` (added by this phase, 10-03) already machine-proves the two riskiest components of D-08 on the same `namespace-profile-macos-6x14-tahoe` label: Namespace serves the label rather than queuing it indefinitely, and the host is a genuine native Apple Silicon toolchain, not a substituted cross-build — confirmed this session via 3 real successful canary runs (`gh run list --workflow=darwin-toolchain-canary.yml`), the most recent at the current HEAD (`a61ccf1`), whose log shows `Apple clang version 21.0.0`, `Target: arm64-apple-darwin25.6.0`, and both darwin arches CGo-building successfully. What remains genuinely unexercised is narrower: the canary runs plain `go build -o /dev/null`, not goreleaser, so release.yml's actual darwin build path (goreleaser with release ldflags, cosign signing, SLSA attestation) has never run on this runner class; no `codegraph upgrade` smoke test has been performed; and `release.yml`'s `on: push: tags:` trigger means the assembled job as a whole first runs during a real release. This residual still warrants a human check at the next tag push, even with runner-availability and native-toolchain risk already discharged.

### Gaps Summary

No gaps found against any of the phase's must-haves. All 3 ROADMAP success criteria and all 10 merged sub-must-haves from the 7 plans are fully verified — including, for the darwin release leg, live evidence (via `darwin-toolchain-canary.yml`'s 3 real runs on the same runner label) that Namespace serves the profile natively, not merely a shape-test assertion. The one item still routed to human verification is narrower than "the darwin leg": goreleaser's own build/sign/attest path and a `codegraph upgrade` smoke test on a real release binary, which genuinely cannot run before a real `v*` tag push. This is a documented, already-tracked residual risk (`STATE.md`, `10-03-SUMMARY.md` "Known Unknowns"), not a surprise discovered during this pass.

---

_Verified: 2026-08-02T21:30:00Z_
_Verifier: Claude (gsd-verifier)_
