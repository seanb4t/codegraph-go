---
phase: 08-release-hardening-benchmarks
verified: 2026-07-13T00:00:00Z
status: human_needed
verdict: passed-with-pending
score: 8/8 requirements built and locally proven; PERF-01 numbers ratified + published (2026-07-13); 1 item (DIST-02 live signed release) requires a real tagged release to fully close
overrides_applied: 0
known_gaps:
  - id: DIST-02-live-release
    statement: "Live 6-target signed release (real OIDC cert, darwin DNS resolution under CI, cosign SAN match) has not been exercised — no v* tag has ever been pushed."
    disposition: "publish-pending — code/config verified correct by static + local proof; requires cutting a real tag to close"
  - id: PERF-01-raw-numbers
    statement: "docs/BENCHMARKS.md now publishes the median-of-3 head-to-head numbers vs installed TS 1.3.1 (Go wins every metric); 3 raw runs committed."
    disposition: "RESOLVED + RATIFIED 2026-07-13 — real median-of-3 numbers published and committed. Follow-up (non-blocking): re-run on standardized CI hardware via bench.yml for the canonical absolute figures; ratios are hardware-durable."
---

# Phase 8: Release Hardening & Benchmarks Verification Report

**Phase Goal:** A trustworthy, fast v1.0 release — signed, attested, reproducible static binaries with a minimal audited dependency tree, plus published head-to-head benchmarks and CI regression gates that validate the whole project.
**Verified:** 2026-07-13
**Status:** human_needed (verdict: passed-with-pending) — automated + local proof all pass; 2 manual items require a real tagged release to close (DIST-02 live signing, PERF-01 published numbers)
**Re-verification:** No — initial verification

## Method

This is not a rerun-the-summaries check. Every claim below was independently re-derived from the actual files in this repo, plus these commands run live in this session:

- `goreleaser check` (exit 0)
- `actionlint .github/workflows/release.yml .github/workflows/ci.yml .github/workflows/bench.yml` (exit 0, no findings)
- `go build ./...` (exit 0)
- `go test ./internal/upgrade/... -run TestReleaseAssetNameMatchesGoReleaser -v` (6/6 sub-tests pass)
- `go test ./internal/upgrade/... -run TestVerifyReleaseE2E -v` (skips cleanly, exact honest reason — no real artifact yet)
- `govulncheck ./...` (0 reachable vulnerabilities)
- `go test ./internal/bench/... -v` (all pass, incl. tolerance-band + absolute-ceiling cases)
- `go run ./tools/bench/runner -mode regression -baseline tools/bench/baseline.json -ceiling-bytes 4294967296` (gate passed against a fresh live measurement, ~12.7k files/s, ~853MB peak RSS — same order of magnitude as the committed baseline)
- `go test ./tools/bench/gencorpus/... -run 'TestDeterministic|TestFileCountExceeds100k' -v` (both pass; 120k-file materialization confirmed)
- full non-daemon suite (`go list ./... | grep -v /internal/daemon$` then `go test`) — all packages pass
- `go test ./internal/daemon/... -count=1` — passes in isolation (per the documented flake-isolation policy)
- `rg 'import "C"' --type go .` — zero hits outside `internal/parser/cgo` (no new CGo introduced)
- manual byte-count of `go.mod`'s require blocks: 134 total / 27 direct / 107 indirect / 14 of the 27 direct are `tree-sitter-*` — matches `docs/RELEASE.md`'s stated numbers exactly
- `rg 'uses:.*@v[0-9]' .github/workflows/*.yml` cross-checked against SHA-pinned lines — only the SLSA generic generator uses a bare semver tag, which is its own documented, `slsa-verifier`-mandated requirement (not a floating-tag oversight)

## Roadmap Success Criteria (the contract)

| # | Success Criterion | Status | Evidence |
|---|---|---|---|
| 1 | Single static binary per platform (6 targets), no bundled runtime, no install-time compilation | ✓ VERIFIED (build config); ⚠️ UNPROVEN live | `.goreleaser.yaml` declares all 6 `CGO_ENABLED=1` builds; `goreleaser check` passes; native `darwin/arm64` build proven locally in Plan 08-01 (Mach-O binary, `--version` prints injected ldflags). The full 6-target native-runner CI matrix (`release.yml`'s `build` job) has never actually run — no `v*` tag has been pushed. |
| 2 | Signed + attested + SBOM'd + govulncheck-gated + reproducible + minimal audited deps | ✓ VERIFIED (code), ⚠️ publish-pending (live signing) | Per-binary `cosign sign-blob --bundle`, per-binary `syft` SBOM, SLSA3 generic-generator provenance, and `govulncheck` blocking gate are all wired and match `internal/upgrade/verify.go`'s exact identity constants byte-for-byte (issuer + SAN regex cross-checked line-by-line, see below). `govulncheck ./...` run locally: 0 reachable vulnerabilities. Reproducibility double-build gate blocks on `linux/amd64`, reports (non-blocking) on `linux/arm64`, matching the documented D-03 scope. DIST-05 dependency narrative (134/27/107/14) verified against `go.mod` exactly. **The one real signed release has not been cut**, so the actual cosign OIDC cert, SLSA attestation, and live 6-target matrix remain unexercised in production. |
| 3 | Published head-to-head benchmarks (throughput/latency/RSS/cold start) vs TS CodeGraph, raw per-repo numbers | ⚠️ PARTIAL / publish-pending | `tools/bench/runner -mode headtohead` and the pinned 3-repo `realcorpus` manifest exist and are documented as manually exercised successfully (08-07-SUMMARY.md). `bench.yml` is wired to run and publish it weekly/on-demand. **`docs/BENCHMARKS.md`'s raw numbers table is explicitly `TBD`** — no committed real per-repo numbers exist yet, only the exact regenerate command. The roadmap criterion's literal requirement ("...with raw per-repo numbers") is not yet met; the mechanism that produces them is. |
| 4 | 100k+ file monorepo indexing within bounded memory; peak RSS as first-class CI metric; regression gate incl. that monorepo | ✓ VERIFIED | `gencorpus.ProductionFileCount = 120000` (exceeds 100k+); `TestFileCountExceeds100k` passes; `tools/bench/baseline.json` is real committed runner output (non-round numbers: 12816.38 files/s, 842350592 bytes RSS — not fabricated); `internal/bench.CheckRegression` enforces both a 15% relative RSS tolerance and an independent absolute ceiling; `ci.yml`'s `perf-regression` job runs this entirely offline. Re-ran the regression command live this session — passed cleanly against a fresh measurement. |

## Per-Requirement Verdicts

| Requirement | Verdict | Evidence |
|---|---|---|
| **DIST-01** | Met (build config); live-matrix unproven | 6-target `.goreleaser.yaml`, `goreleaser check` clean, raw-binary format (not tar.gz), native darwin/local proof done. Real CI 6-target run pending first tag. |
| **DIST-02** | Complete-pending-first-release | Signing/identity wiring is code-verified correct (SAN pattern, issuer, per-binary `.sigstore.json`, hash-the-binary-not-the-archive contract all match `verify.go` exactly). `TestReleaseAssetNameMatchesGoReleaser` passes for all 6 targets. `TestVerifyReleaseE2E` compiles/runs/skips cleanly (proven not to spuriously fail) but the actual accept/reject assertions against a genuine signed artifact are **not yet exercised** — no real release exists. Recommend: cut a real `v0.1.0`-style pre-release tag as the very next action before calling this fully closed. |
| **DIST-03** | Met | Per-binary syft SBOM in `release.yml`; `govulncheck` runs as its own blocking job in `ci.yml`, deliberately separate from SBOM generation; ran locally, 0 reachable vulnerabilities. |
| **DIST-04** | Met (as scoped) | Double-build gate blocks on `linux/amd64` (verified logic matches `.goreleaser.yaml`'s exact ldflags/trimpath/mod-time flags), reports non-blocking on `linux/arm64`; determinism knobs (`-trimpath`, `-buildid=`, `mod_timestamp`, pinned GoReleaser/zig versions) present. Honestly scoped — windows/darwin cross-target reproducibility is stated as posture, not an implemented CI check, in `docs/RELEASE.md`. |
| **DIST-05** | Met | Already-complete narrative re-verified: `go.mod` has 134 total requires (27 direct/107 indirect), 14 of 27 direct are `tree-sitter-*` grammar modules — exact match to `docs/RELEASE.md` §2. No new CGo found outside `internal/parser/cgo` (`import "C"` grep). |
| **PERF-01** | Publish-pending | Harness (`tools/bench/runner -mode headtohead`, `realcorpus` 3-repo pinned manifest with commit SHAs + query probes) exists and is documented as manually proven functional. `docs/BENCHMARKS.md` numbers table is honestly `TBD`, not fabricated — but the roadmap's literal "published ... raw per-repo numbers" is not yet satisfied. |
| **PERF-02** | Met | `internal/bench.CheckRegression` (tolerance-band + absolute-ceiling) fully unit-tested; `tools/bench/baseline.json` is real committed output; live re-run of the gate this session passed against a fresh measurement. |
| **INDX-06** | Met | `ProductionFileCount=120000` exceeds 100k+; `TestFileCountExceeds100k` passes; absolute peak-RSS ceiling enforced independently of relative tolerance; `ci.yml`'s `perf-regression` job runs this entirely offline against the synthetic corpus. |

## Anti-Pattern / Debt-Marker Scan

- `rg 'TBD|FIXME|XXX' .github/workflows/*.yml .goreleaser.yaml internal/bench/*.go tools/bench/**/*.go docs/RELEASE.md docs/BENCHMARKS.md` → only hits in `docs/BENCHMARKS.md`'s raw-numbers table (7 `TBD` cells + 1 prose reference). These are not unreferenced debt markers left by oversight — they are the explicit, deliberate, documented PERF-01 publish-pending gap already called out above and in the plan's own key-decisions (08-09-SUMMARY.md: "Fabricating plausible-looking numbers would violate the plan's own prohibition"). Treated as the known PERF-01 gap, not a separate blocker.
- No `console.log`-only stubs, no `return null`/`return []` hollow implementations found in the CI/release/bench code touched by this phase.
- No floating (unpinned) third-party Action tags; the one bare-semver reference (`slsa-framework/slsa-github-generator@v2.1.0`) is a documented, `slsa-verifier`-mandated exception, not an oversight.
- No `${{ }}` GitHub Actions context values interpolated directly inside any `run:` script body — all such values pass through `env:` first (checked across all three workflow files).
- No `CGO_ENABLED=0` anywhere in the release/CI config (would silently break the parser).

## Requirements Coverage vs REQUIREMENTS.md

REQUIREMENTS.md marks all 8 requirements (DIST-01..05, PERF-01/02, INDX-06) as **Complete**. This verification finds that marking accurate for 6 of 8 (DIST-01 build-config, DIST-03, DIST-04, DIST-05, PERF-02, INDX-06), and **optimistic** for DIST-02 and PERF-01 — both are fully built and locally proven correct, but their roadmap-level completion criteria (a real signed release existing; real numbers being published) genuinely have not happened yet. This is the same honest gap the phase's own SUMMARY.md files (08-04, 08-05, 08-09) already flag — this verification independently confirms those flags are accurate and not overstated in either direction.

## Human Verification / Action Items Before Calling v1.0 Fully Shipped

1. **Cut a real pre-release tag** (e.g. `v0.1.0-rc1`) and let `release.yml` run to completion. Confirm: all 6 binaries build (especially watch the two `windows` cross-targets and both native `darwin` legs), cosign signs each individually, SLSA provenance attaches, and `cosign verify-blob` / `slsa-verifier verify-artifact` against the real artifacts succeed using the exact commands in `docs/RELEASE.md`. This closes DIST-01's live-matrix proof and DIST-02 fully.
2. **Run `bench.yml` (workflow_dispatch) or the local `-mode headtohead` command** and commit the real output into `docs/BENCHMARKS.md`'s numbers table, replacing the `TBD` cells. This closes PERF-01.
3. Once (1) produces a real `.sigstore.json`, either commit it + the binary as `internal/upgrade/testdata/e2e-release-binary[.sigstore.json]` or point `CODEGRAPH_E2E_BINARY`/`CODEGRAPH_E2E_BUNDLE` at it and re-run `TestVerifyReleaseE2E` to get a real PASS (not a skip) on the production identity path.

## Gaps Summary

No blocking defects were found in any code, config, or wiring this phase produced — every artifact examined is substantive, correctly wired, and (where testable without a real tagged release) passes. The two open items are not implementation gaps; they are **real-world validation steps that structurally cannot happen inside a development session** — they require pushing an actual git tag and letting GitHub Actions' OIDC/cosign infrastructure run for real. Both are explicitly and honestly documented in the shipped docs (`docs/RELEASE.md`, `docs/BENCHMARKS.md`) rather than hidden or fabricated, which is itself evidence the phase's honesty discipline held under adversarial re-check.

---

_Verified: 2026-07-13_
_Verifier: Claude (gsd-verifier)_
