---
phase: 10
slug: local-build-contribution-and-taskfile-yml-setup
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: validated
nyquist_compliant: true
wave_0_complete: true
created: 2026-08-01
---

# Phase 10 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (Go 1.26.5), standard library `testing` only — no external assertion library |
| **Config file** | none — `go.mod` at repo root; `Taskfile.yml` owns the command bodies (DEV-01) |
| **Quick run command** | `go test ./internal/upgrade/` |
| **Full suite command** | `task test` (serial wrapper: `test:unit` → `test:golden` → `test:integration` → `test:daemon` → `test:race`) |
| **Estimated runtime** | ~40s quick (52 tests) · ~90s full suite |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/upgrade/`
- **After every plan wave:** Run `task test`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 40 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 10-01-01 | 01 | 1 | DEV-01 | T-10-01-01..05 | Tool built from module proxy, not a GitHub-release CDN fetch | integration | `GOWORK=off go tool -modfile=go.tool.mod task check:goreleaser` | ✅ | ✅ green |
| 10-01-02 | 01 | 1 | DEV-01 | T-10-01-01..05 | Lint toolchain isolated in its own modfile; no MVS bleed into main module | integration | `GOWORK=off go tool -modfile=go.tool.mod task lint:actions` | ✅ | ✅ green |
| 10-01-03 | 01 | 1 | DEV-01 | T-10-01-01..05 | Required-check names byte-stable; tool modfiles stay isolated | unit | `go test ./internal/upgrade/ -run 'TestRequiredCheckNamesPreserved\|TestToolModfilesRemainIsolated'` | ✅ | ✅ green |
| 10-02-01 | 02 | 2 | DEV-01 | T-10-02-01..04 | Command bodies ported verbatim — no silent semantic drift | integration | `GOWORK=off go tool -modfile=go.tool.mod task vet test:golden test:integration` | ✅ | ✅ green |
| 10-02-02 | 02 | 2 | DEV-01 | T-10-02-01..04 | No target uses `status:`/`platforms:` (both silently SKIP — D-11/GOLDEN-01); wrapper is serial, never concurrent `deps:` | unit | `go test ./internal/upgrade/ -run 'TestTaskfileGatesFailLoud\|TestTaskfileWrapperIsSerial'` | ✅ | ✅ green |
| 10-02-03 | 02 | 2 | DEV-01 | T-10-02-01..04 | Workflow YAML stays valid after rewire | integration | `GOWORK=off go tool -modfile=go.tool.mod task lint:actions` | ✅ | ✅ green |
| 10-03-01 | 03 | 2 | DEV-01 | T-10-03-01..05 | Maintainer decision gate — darwin runner class | checkpoint:decision | — (blocking human gate) | n/a | 🔶 manual-only |
| 10-03-02 | 03 | 2 | DEV-01 | T-10-03-01..05 | Only movable jobs relocate; SLSA provenance stays put (D-07 structural constraint) | integration | `GOWORK=off go tool -modfile=go.tool.mod task lint:actions` | ✅ | ✅ green |
| 10-03-03 | 03 | 2 | DEV-01 | T-10-03-01..05 | Darwin legs build natively (no zig cross-link → libresolv/DNS regression); SLSA generator stays version-pinned | unit | `go test ./internal/upgrade/ -run 'TestDarwinLegsBuildNatively\|TestProvenanceJobUsesTaggedSLSAGenerator'` | ✅ | ✅ green |
| 10-04-01 | 04 | 3 | DEV-01 | T-10-04-01..05 | Runner identity recorded in the baseline frame descriptor | unit (tdd) | `go test ./internal/bench/... ./tools/bench/...` | ✅ | ✅ green |
| 10-04-02 | 04 | 3 | DEV-01 | T-10-04-01..05 | Candidate baseline measured, not asserted | integration | `GOWORK=off go tool -modfile=go.tool.mod task lint:actions` | ✅ | ✅ green |
| 10-04-03 | 04 | 3 | DEV-01 | T-10-04-01..05 | Maintainer reads the number before it lands (D-05: rebless is sole writer) | checkpoint:decision | — (blocking human gate) | n/a | 🔶 manual-only |
| 10-04-04 | 04 | 3 | DEV-01 | T-10-04-01..05 | Baseline committed with its provenance, copied verbatim from the artifact | unit | `go test ./internal/bench/... ./tools/bench/...` | ✅ | ✅ green |
| 10-05-01 | 05 | 3 | DEV-01 | T-10-05-01..05 | Six-target sweep ported verbatim | integration | `GOWORK=off go tool -modfile=go.tool.mod task check:cross` | ✅ | ✅ green |
| 10-05-02 | 05 | 3 | DEV-01 | T-10-05-01..05 | Pre-tag gate routed through the single definition | integration | `GOWORK=off go tool -modfile=go.tool.mod task lint:actions` | ✅ | ✅ green |
| 10-05-03 | 05 | 3 | DEV-01 | T-10-05-01..05 | `check:cross` and `.goreleaser.yaml` cannot drift apart | unit | `go test ./internal/upgrade/ -run TestCheckCrossMatchesGoreleaserTargets` | ✅ | ✅ green |
| 10-06-01 | 06 | 4 | DEV-01 | T-10-06-01..05 | Cross-runner / cross-scratch_fs comparison REFUSED as a category error; fail-closed when one side is empty | unit (tdd) | `go test ./internal/bench/... ./tools/bench/...` | ✅ | ✅ green (25 subtests) |
| 10-06-02 | 06 | 4 | DEV-01 | T-10-06-01..05 | Perf gate invoked through a task target, runner recorded | integration | `GOWORK=off go tool -modfile=go.tool.mod task lint:actions` | ✅ | ✅ green |
| 10-06-03 | 06 | 4 | DEV-01 | T-10-06-01..05 | Double-build hash-diff reproducibility preserved across the move | integration | `GOWORK=off go tool -modfile=go.tool.mod task check:reproducibility` | ✅ | ✅ green |
| 10-07-01 | 07 | 5 | DEV-01 | T-10-07-01..04 | CONTRIBUTING points at real targets — guarded against drift (gap closed this audit) | unit | `go test ./internal/upgrade/ -run TestContributingReferencesRealTaskTargets` | ✅ | ✅ green |
| 10-07-02 | 07 | 5 | DEV-01 | T-10-07-01..04 | CI invokes `task <target>`; no workflow re-declares a command body (the DEV-01 single-definition property) | unit (tdd) | `go test ./internal/upgrade/ -run TestWorkflowRunBodiesInvokeTask` | ✅ | ✅ green |
| 10-07-03 | 07 | 5 | DEV-01 | T-10-07-01..04 | A clean checkout builds, tests and lints through task alone | integration | `GOWORK=off go tool -modfile=go.tool.mod task build test lint` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky · 🔶 manual-only*

**Sampling continuity:** the only two tasks without an automated command (10-03-01, 10-04-03) are `type="checkpoint:decision" gate="blocking"` maintainer gates and are non-adjacent — no 3 consecutive tasks lack automated verification.

**Non-vacuity:** every guard in `internal/upgrade/` carries a machine-checked companion proving it can FAIL — `_ZeroJobsIsError`, `_AbsentModfileIsError`, `_EmptyFileIsError`, `_MissingWrapperIsError`, `_EmptyBuildsIsError`, `_MissingCheckCrossIsError`, `_UnknownTargetIsError`, plus `TestTaskfileShapeHelpersFailLoudly`, `TestCheckCrossParsersFailLoudly`, `TestWorkflowSourceHelpersFailLoudly`. `TestWorkflowRunBodiesInvokeTask` was additionally proven RED at base commit `82ffd60`.

---

## Wave 0 Requirements

- [x] `internal/upgrade/taskfile_shape_test.go` — shape guards for DEV-01 (pre-existing, extended this audit)
- [x] `internal/upgrade/release_workflow_shape_test.go` — release-pipeline constraint guards for DEV-01
- [x] `internal/bench/regression_test.go` — frame-descriptor gating for DEV-01

*Existing infrastructure covers all phase requirements — no framework install needed.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Darwin runner-class decision (10-03-01) | DEV-01 | `type="checkpoint:decision" gate="blocking"` — a maintainer choice between runner classes has no machine-checkable correct answer | Recorded in 10-03-SUMMARY.md; decision was `namespace-profile-macos-6x14-tahoe`, since machine-proven by `darwin-toolchain-canary.yml` |
| Baseline adoption (10-04-03) | DEV-01 | `type="checkpoint:decision" gate="blocking"` — D-05 requires a human to read the measured number before it is committed; automating adoption would defeat the control | Recorded in 10-04-SUMMARY.md and `tools/bench/BASELINE.md` |
| `release.yml` darwin goreleaser + cosign signing + SLSA attestation | DEV-01 | `release.yml` triggers only on `push: tags: v[0-9]*` — the assembled job cannot run before a real release. The canary proves runner availability and the native toolchain, but runs plain `go build`, not goreleaser | Tracked in `10-UAT.md` item 1; push a real `v[0-9]*` tag and watch the darwin matrix legs end-to-end |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 40s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-08-03

---

## Validation Audit 2026-08-03

| Metric | Count |
|--------|-------|
| Gaps found | 1 |
| Resolved | 1 |
| Escalated | 0 |

**Gap:** DEV-01's clause "*and `CONTRIBUTING.md` points contributors at the targets*" had zero automated verification — `rg -l -i CONTRIBUTING --glob '*_test.go'` returned nothing, while every other DEV-01 clause was guarded. Not hypothetical: 10-REVIEW.md finding WR-04 shows CONTRIBUTING.md's required-checks list had **already** drifted stale (omitting `goreleaser check (DIST-01)`) with nothing going red.

**Resolution:** `TestContributingReferencesRealTaskTargets` added to `internal/upgrade/taskfile_shape_test.go`, with a `_UnknownTargetIsError` companion. It parses every backtick-enclosed `` `task <target>` `` reference from CONTRIBUTING.md, fails if the set is empty, and asserts each resolves to a real Taskfile.yml block. Parser verified to extract exactly the 6 targets currently named — `build`, `check:cross`, `check:reproducibility:arm64`, `lint`, `test`, `vet:daemon-windows` — no more, no fewer.

**Non-vacuity proven by measurement, not assertion:** appended `task definitely-not-a-real-target-xyz` to CONTRIBUTING.md, confirmed the mutation landed before trusting any result, then ran the **main** test (not just the helper): exit 0 on the clean tree → exit 1 on the mutated tree with an actionable DEV-01 message. Reverted with zero residue. Neither CONTRIBUTING.md nor Taskfile.yml was modified — WR-04's stale list was deliberately left in place so the new guard passes for the right reason rather than hiding the documented drift.

Full package after the change: 52 tests pass, `go vet ./internal/upgrade/` exit 0.
