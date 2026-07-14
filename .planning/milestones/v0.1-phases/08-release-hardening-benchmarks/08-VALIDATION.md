---
phase: 8
slug: release-hardening-benchmarks
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-07-13
---

# Phase 8 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution. Seeded from 08-RESEARCH.md § Validation Architecture.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go built-in `testing` (`go test`) — no third-party test framework in this repo |
| **Config file** | none — plain `go test`; `internal/daemon` has a documented pre-existing flake (`TestSoak`, flush-lock) under full-suite parallel load |
| **Quick run command** | `go test ./internal/bench/... -run TestSmoke` (new this phase) |
| **Full suite command** | `go test ./...` (isolate `internal/daemon` with `-count=1` if it flakes) + `go test ./internal/bench/... -bench=. -benchmem` for the perf gate |
| **Estimated runtime** | ~60–120s for `./...`; bench/regression run is separate and longer |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/bench/... -run TestSmoke` (fast sanity of RSS-capture/normalization logic — the single most likely silent-corruption bug this phase, Pitfall 4)
- **After every plan wave:** Run `go test ./...` + `govulncheck ./...`
- **Before `/gsd-verify-work`:** Full suite green + a real double-build hash-diff run + at least one actual `verifyRelease` pass against a real signed release artifact
- **Max feedback latency:** ~120 seconds (per-commit quick run is single-digit seconds)

---

## Per-Task Verification Map

> Task IDs are assigned by the planner; this seeds the requirement→test mapping. Every entry is Wave-0-new (no `.github/workflows/`, `internal/bench/`, or `tools/bench/` exists yet).

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| (TBD) | (TBD) | — | DIST-01 | — | Single static binary per platform, no bundled runtime | smoke (CI) | per-matrix `file <bin>` + `./codegraph --version` | ❌ W0 | ⬜ pending |
| (TBD) | (TBD) | — | DIST-02 | T-tamper / T-oidc-scope | Each binary cosign-signed; `verifyRelease` accepts real artifact, rejects wrong identity | integration (e2e) | `go test ./internal/upgrade/ -run VerifyReleaseE2E` | ❌ W0 | ⬜ pending |
| (TBD) | (TBD) | — | DIST-03 | — | SBOM published; govulncheck gates | smoke (CI) | `govulncheck ./...` (blocking) + assert `*.spdx.json` release asset | ❌ W0 | ⬜ pending |
| (TBD) | (TBD) | — | DIST-04 | — | Reproducible build, double-build match | integration (CI script) | double-build hash-diff, blocking on linux/amd64 | ❌ W0 | ⬜ pending |
| (TBD) | (TBD) | — | DIST-05 | — | Dependency tree minimal + audited, CGo sole exception | smoke | dep-audit assertion / SBOM narrative (DIST-05 already Complete) | ✅ | ✅ green |
| (TBD) | (TBD) | — | PERF-01 | — | Head-to-head vs TS 1.3.1, median-of-N, raw numbers | manual-only (published artifact) | `go run ./tools/bench/runner -mode headtohead` → `docs/BENCHMARKS.md` | ❌ W0 | ⬜ pending |
| (TBD) | (TBD) | — | PERF-02 | — | CI regression gate vs baseline incl. 100k+ corpus | integration (CI gate) | `go run ./tools/bench/runner -mode regression` vs `baseline.json` | ❌ W0 | ⬜ pending |
| (TBD) | (TBD) | — | INDX-06 | — | Index 100k+ monorepo in bounded memory; peak RSS tracked | integration (CI gate) | same regression run asserts peak-RSS ceiling (not just relative) | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/bench/rss_test.go` — unit tests for the Linux-KB-vs-Darwin-bytes `ru_maxrss` normalization (Pattern 1 / Pitfall 4) — cheap, pure-function, guards the top silent-corruption risk
- [ ] `internal/upgrade/verify_release_e2e_test.go` — real-signed-artifact-against-`verifyRelease` test (the test CONTEXT.md `<specifics>` explicitly calls for)
- [ ] `tools/bench/gencorpus/` — synthetic corpus generator + a determinism test (same seed → same file count/hash)
- [ ] `tools/bench/baseline.json` — initial committed baseline, established by a first real run before the gate blocks
- [ ] `.github/workflows/{release,ci,bench}.yml` — none exist yet; this phase creates all three from scratch

*Nothing existing covers these — the entire CI/release/bench surface is new.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Published head-to-head benchmark numbers | PERF-01 | It is a published artifact, not a pass/fail gate; requires the installed TS `codegraph@1.3.1` and real corpora | `go run ./tools/bench/runner -mode headtohead`, review `docs/BENCHMARKS.md` raw per-repo numbers for plausibility |
| First real signed-release verify | DIST-02 | Requires a real tag-triggered release to exist (OIDC signing only happens in the real `release.yml` run) | Cut a `v*` pre-release tag, then run `codegraph upgrade --check`/a `verifyRelease` dry-run against the produced `.sigstore.json` |
| Reproducibility of cross-targets (non-linux/amd64) | DIST-04 | CGo cross-linked artifacts are best-effort, not the blocking guarantee | Compare double-build hashes on darwin/windows targets; record drift in `docs/RELEASE.md`, do not block |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 120s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
