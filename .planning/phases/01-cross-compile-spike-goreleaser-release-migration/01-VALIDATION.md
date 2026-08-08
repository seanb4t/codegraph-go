---
phase: 1
slug: cross-compile-spike-goreleaser-release-migration
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-08
---

# Phase 1 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go standard `testing` — release-pipeline correctness is enforced by **workflow-shape tests** that parse the on-disk YAML of `.github/workflows/*.yml`, `.goreleaser.yaml`, and `Taskfile.yml` and assert structural invariants |
| **Config file** | none — standard `go test`, no framework config |
| **Quick run command** | `go test ./internal/upgrade/... -run <TestName>` |
| **Full suite command** | `go test ./internal/upgrade/...` (shape tests); `task test:unit` (broader unit suite) |
| **Estimated runtime** | ~0.6s test time / ~2.2s wall including compile (measured 2026-08-08) |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/upgrade/... -run <TestName>` for the shape test that task adds or rewrites
- **After every plan wave:** Run `go test ./internal/upgrade/...`, plus `task check:goreleaser` (DIST-01) when the wave touched `.goreleaser.yaml`
- **Before `/gsd-verify-work`:** Full `go test ./...` green, plus a real dispatch of the D-03 canary workflow (requires live Namespace macOS + Linux runners — cannot be simulated locally)
- **Max feedback latency:** 3 seconds (local shape tests); canary/post-release legs are CI-latency bound and sampled per wave, not per task

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| {N}-01-01 | 01 | 1 | REQ-{XX} | T-{N}-01 / — | {expected secure behavior or "N/A"} | unit | `{command}` | ✅ / ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

*Populated by `/gsd-plan-phase` task breakdown and completed by `/gsd-validate-phase`.*

---

## Wave 0 Requirements

- [ ] New canary workflow (D-03) exercising zig-cross-from-macOS for both linux legs and **executing** the resulting binaries on real Linux (amd64 profile + new arm64 profile) — the REL-05 spike itself
- [ ] New Taskfile target(s) for the D-06 `goreleaser release --snapshot --skip=publish,sign` dry run
- [ ] New Go shape tests in `internal/upgrade/` for: D-11's single-job `id-token: write` scope, D-14's `.sigstore.json` template contract, REL-07's hand-rolled-`sha256sum`-absence assertion
- [ ] Rewrite of `TestDarwinLegsBuildNatively` per D-13's new invariant
- [ ] Rewrite of `TestProvenanceJobUsesTaggedSLSAGenerator` (or its replacement) for the `actions/attest-build-provenance` job shape, per D-10
- [ ] New post-release automated self-upgrade job/workflow (D-08)
- [ ] New Namespace linux-arm64 runner profile — **infrastructure, not code**; dashboard-provisioned, unknown lead time, blocks the REL-05 arm64 leg

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| FAIL-bar variation list enumerated before the first spike run | REL-05 | Planning deliverable (D-04), not executable code — the list defines what counts as spike failure and must exist *before* evidence is gathered to avoid post-hoc goalpost-moving | Record the enumerated variations in the plan/PR body prior to the first canary dispatch; a third party must be able to re-read them against the recorded run |
| Architecture decision recorded on re-inspectable evidence (zig-cross success, or GoReleaser Pro adoption with the three named gate repairs) | REL-05 | Human judgment on a one-way-door decision; the *inputs* are automated (canary run output) but the recorded decision is not | Attach the canary run URL + fixture-indexing output to the decision record; if either leg fails, record GoReleaser Pro adoption with `check:goreleaser`/DIST-01, `TestGoreleaserPinParity`, and `tool-vuln`/VULN-01-02-03 entered as scope |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 3s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
