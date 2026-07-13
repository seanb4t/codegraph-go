# Phase 8: Release Hardening & Benchmarks - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-13
**Phase:** 8-Release Hardening & Benchmarks
**Mode:** `--auto` (all gray areas auto-selected to the recommended option; no interactive prompts)
**Areas discussed:** CGo cross-compile toolchain, supply-chain tool wiring, reproducible-build gate, benchmark corpus split, perf metric methodology + gate policy, workflow/file layout

---

## CGo Cross-Compilation Toolchain (DIST-01)

| Option | Description | Selected |
|--------|-------------|----------|
| `zig cc` via GoReleaser, single Linux runner | Cross-compile all 6 targets `CGO_ENABLED=1` with `zig cc -target`; single builder; matches CLAUDE.md/STACK.md/PARSER-DECISION.md | ✓ |
| Native-runner matrix | ubuntu/macos/windows GitHub runners each build their own platform natively | (documented fallback) |
| musl-cross toolchain | Traditional cross-compile toolchain per target | |

**Auto-selected:** `zig cc` via GoReleaser (recommended default).
**Notes:** Cross-compile was **never executed** in the Phase-1 spike (zig/wasi-sdk absent) — PARSER-DECISION.md deferred validation to Phase 8. Native-runner matrix is the documented fallback if macOS SDK / Windows CGo linking via zig proves untenable. This is the phase's highest-risk task.

---

## Build → Sign → Provenance → SBOM Wiring (DIST-02, DIST-03)

| Option | Description | Selected |
|--------|-------------|----------|
| GoReleaser build + cosign keyless + SLSA **generic** generator + syft + govulncheck | Provenance runs over GoReleaser's artifacts/checksums via `generator_generic_slsa3.yml` | ✓ |
| SLSA `builder_go_slsa3.yml` (Go builder) | Go-specific SLSA builder rebuilds the binary itself | (rejected — can't do zig-cc CGo cross-build) |

**Auto-selected:** GoReleaser + cosign keyless + SLSA generic generator + syft SBOM + govulncheck (recommended default).
**Notes:** The Go SLSA builder rebuilds under a fixed config incompatible with `zig cc` CGo cross-compilation; running provenance over already-built artifacts avoids that conflict while still yielding verifiable SLSA3.

---

## Reproducible Builds + Double-Build Gate (DIST-04)

| Option | Description | Selected |
|--------|-------------|----------|
| Determinism knobs + linux/amd64-blocking gate | `-trimpath`/cleared-buildid/`SOURCE_DATE_EPOCH`/pinned zig+Go; blocking double-build on linux/amd64, others best-effort reported | ✓ |
| Blocking double-build on all 6 targets | Require bit-identical rebuilds on every target | (deferred — revisit once cross-target numbers known) |

**Auto-selected:** determinism knobs + linux/amd64-blocking gate, others reported (recommended default).
**Notes:** zig is deterministic (reproducibility asset). CGo cross-linked artifacts are harder to guarantee bit-identical; being explicit about the hard-guarantee target beats a green check that hides drift.

---

## Benchmark Corpus Split (PERF-01 vs PERF-02/INDX-06)

| Option | Description | Selected |
|--------|-------------|----------|
| Split: real pinned repos for head-to-head; committed synthetic 100k+ for CI gate | PERF-01 vs installed TS 1.3.1 on real repos; PERF-02 gate on a network-free deterministic generator | ✓ |
| Single real 100k+ public monorepo for everything | Clone one big real repo at pinned SHA for both head-to-head and the gate | (real repo used to publish INDX-06 number only, not as CI-gate dependency) |
| Existing corpus only | Reuse only the small in-repo corpora | |

**Auto-selected:** split corpora (recommended default).
**Notes:** TS `@colbymchenry/codegraph@1.3.1` confirmed installed at `/opt/homebrew/bin/codegraph`. CI gate must not depend on network; a committed synthetic 100k+ generator keeps it reproducible.

---

## Perf Metric Methodology + Regression-Gate Policy (PERF-01/02, INDX-06)

| Option | Description | Selected |
|--------|-------------|----------|
| OS-level peak RSS + median-of-5 + committed baseline + tolerance-band fail | External `ru_maxrss`/`/usr/bin/time`; throughput/query-latency/cold-start; committed baseline JSON, fail beyond band, documented re-bless | ✓ |
| benchmark-action historical tracking | External action stores history and compares | |
| Warn-only gate | Report regressions, never fail CI | |

**Auto-selected:** OS-level peak RSS + median-of-5 + committed baseline + tolerance-band fail (recommended default).
**Notes:** OS-level RSS is the only fair comparison against the TS Node process. Starting bands: throughput −10%, peak RSS +15% (tune in planning).

---

## Repository Layout + Workflow Structure

| Option | Description | Selected |
|--------|-------------|----------|
| `release.yml` (LOCKED) + `ci.yml` + `bench.yml`, `.goreleaser.yaml`, `internal/bench`+`tools/bench`, docs | Separate release/CI/bench workflows; harness reuses `tools/spike` pattern | ✓ |
| Single mega-workflow | All jobs in one workflow file | (rejected — release identity must be isolated in `release.yml`) |

**Auto-selected:** separate workflows + standard layout (recommended default).
**Notes:** `.github/workflows/release.yml` name + `v[0-9]*` tag trigger + cosign SAN are **LOCKED** by the Phase-6 verifier (`internal/upgrade/verify.go`); deviation silently breaks `codegraph upgrade`.

---

## Claude's Discretion

- Exact tolerance-band percentages, median-N value, synthetic-corpus file/language mix, the extra real repos in the head-to-head set, GoReleaser template details, doc phrasing.
- Whether the real 100k+ monorepo headline number is published from a manual/scheduled job vs a one-time captured artifact.

## Deferred Ideas

- Teach `sync`/`index` to reconcile a TS-migrated graph's ids without a full churn pass (Phase-7 D-01 open question) — sync-engine concern.
- Package-manager distribution channels (Homebrew/apt/winget) beyond raw GitHub-release binaries.
- Milestone-2 team-scale features (central server, CI-distributed indexes, concurrent multi-writer).
- Promoting the reproducibility double-build gate to blocking on all 6 targets.
