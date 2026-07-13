# Phase 8: Release Hardening & Benchmarks - Context

**Gathered:** 2026-07-13
**Status:** Ready for planning
**Mode:** `--auto` (decisions auto-selected to recommended defaults; review before planning)

<domain>
## Phase Boundary

Deliver a trustworthy, fast **v1.0 release** of the Go binary and the benchmarks that prove the project's core value. This is the final phase — it scaffolds the CI/release supply chain and validates the whole project end-to-end. Two halves:

**A. Supply-chain / distribution hardening**
- **DIST-01** — one static binary per platform (macOS/Linux/Windows × amd64+arm64), no bundled runtime, no install-time compilation.
- **DIST-02** — every artifact cosign-signed (keyless) with SLSA build provenance; documented verify commands.
- **DIST-03** — every release publishes an SBOM; `govulncheck` + dependency scanning gate CI.
- **DIST-04** — reproducible builds, verified by a double-build comparison gate in CI.
- **DIST-05** — *(already Complete)* minimal audited dependency tree; CGo (tree-sitter) is the sole documented exception.

**B. Benchmarks + performance regression gates**
- **PERF-01** — published head-to-head benchmarks vs TS CodeGraph (indexing throughput, query latency, peak RSS, cold start) on real repos, comparable methodology, raw per-repo numbers.
- **PERF-02** — CI performance-regression gates against a benchmark corpus that includes a 100k+ file monorepo.
- **INDX-06** — index a 100k+ file monorepo within bounded memory; peak RSS tracked as a first-class CI metric.

**In scope:** GitHub Actions release + CI workflows; `.goreleaser.yaml`; CGo cross-compilation for all 6 targets; cosign keyless signing; SLSA provenance; SBOM; `govulncheck`/dep-scan gate; reproducible-build gate; a benchmark harness (real head-to-head corpus + a deterministic large-monorepo corpus); a perf-regression gate with a committed baseline; verify/benchmark docs.

**Out of scope:** any change to indexer/query/sync/migrate behavior (this phase *validates*, it does not add product features); milestone-2 team-scale features (central server, CI-distributed indexes); package-manager distribution channels beyond raw GitHub-release binaries (Homebrew tap / apt / winget — see Deferred); the Phase-7 TS-migrated-id `sync` reconciliation open question (a sync-engine concern — see Deferred).
</domain>

<decisions>
## Implementation Decisions

> All decisions below were auto-selected (`--auto`) to the recommended option. They are locked for research/planning unless the user revises them. Much of this phase's stack is **already fixed by `CLAUDE.md`/`STACK.md`** (GoReleaser v2, cosign v3 keyless, `slsa-github-generator`, syft SBOM, `govulncheck`) — those are not re-litigated here. The decisions below are the genuine gray areas that the pre-decided stack leaves open.

### CGo cross-compilation — the central technical risk (DIST-01)
- **D-01:** **Native-runner matrix is the RECOMMENDED path (RESEARCH-corrected); `zig cc` from a single Linux runner is the fallback.** `CGO_ENABLED=1`, targeting all 6 platforms (darwin/linux/windows × amd64/arm64).
  - **RESEARCH CORRECTION (08-RESEARCH.md):** the original single-Linux-runner + `zig cc` plan carries a real, multi-source-corroborated risk for **darwin** targets — Apple's SDK / DNS-resolver libraries are not redistributable, so a Linux→darwin CGo cross-link (`zig cc`/`osxcross`) can break `net/http` DNS resolution at link or runtime. GoReleaser's own multi-runner merge tooling (`--split`/`continue --merge`, `prebuilt` builder) is **Pro-only** and this project has no Pro license. ⇒ **Primary plan:** a native GitHub-runner matrix (`ubuntu-latest` builds linux+windows via `zig cc`; `macos-latest` builds darwin natively) with `goreleaser build --single-target` per runner, results merged for the release job. **Fallback:** single-Linux-runner `zig cc` for all 6 if the darwin risk is disproven in practice.
  - PARSER-DECISION.md explicitly deferred this cross-compile validation to Phase 8 and it was **never actually executed in the spike** (zig absent) — validating a real 6-target CGo build (esp. darwin DNS resolution) is the first, highest-risk task of this phase. **Open question for planner (08-RESEARCH.md OQ1):** does the project have/intend a GoReleaser Pro license? If yes, the single-runner merge story simplifies.

### Build → sign → provenance → SBOM wiring (DIST-02, DIST-03)
- **D-02:** **GoReleaser builds; cosign signs EACH BINARY; the SLSA *generic* generator attests; syft SBOMs; govulncheck gates.**
  - GoReleaser v2 builds all 6 targets (per D-01), publishes them as **raw per-platform binaries** (`format: binary`, NOT `.tar.gz` archives), and emits the SBOM via its `sbom:` (syft) block.
  - **RESEARCH CORRECTION (08-RESEARCH.md) — supersedes the original "sign the checksums file" call:** the already-shipped Phase-6 verifier (`internal/upgrade/upgrade.go` `defaultVerify` → `sha256.Sum256(binary)` + `verifyRelease` `WithArtifactDigest`) binds the signature to **each individual binary's own sha256**, and `defaultDownload` fetches `<asset>` + `<asset>.sigstore.json`. ⇒ **cosign v3 keyless must sign each release binary individually and publish a per-binary `<asset>.sigstore.json` bundle** (GitHub Actions OIDC, `id-token: write`, no long-lived key). Signing ONLY `checksums.txt` would make **every real `codegraph upgrade` fail**. (A signed checksums file may still be published for `slsa-verifier`/human use, but the per-binary bundles are mandatory.)
  - **SLSA3 provenance via `generator_generic_slsa3.yml` (the generic generator) over the release artifacts — NOT `builder_go_slsa3.yml`.** Rationale: the Go SLSA builder rebuilds the binary under its own fixed config and does not support the CGo cross-build; running provenance over already-built artifacts avoids that conflict while still producing verifiable SLSA3. **Must pin the generator to a full `@vX.Y.Z` semver tag** — `slsa-verifier` rejects short tags (08-RESEARCH.md).
  - `govulncheck ./...` (call-graph-aware) runs as a **blocking CI gate**; dependency scanning (govulncheck primary; a generic SCA on the SBOM acceptable as secondary) also gates.

### Reproducible builds + double-build gate (DIST-04)
- **D-03:** **Determinism knobs + a scoped double-build gate.** Build with `-trimpath`, a cleared/empty build id (`-ldflags "-buildid="`), `SOURCE_DATE_EPOCH` pinned to the tag's commit date, `-mod=readonly`, and **pinned Go + zig toolchain versions** (zig is itself deterministic — a strong asset here).
  - **Gate scope:** the double-build comparison **blocks on `linux/amd64`** (the canonical, lowest-variance target) and runs **best-effort + reported (non-blocking)** on the other 5 targets. CGo cross-linked artifacts are harder to guarantee bit-identical across the toolchain; being explicit about which target is the hard guarantee is more honest than a green check that hides cross-target drift. Researcher confirms how close the cross-targets get and whether any can be promoted to blocking.

### Benchmark corpus — split by purpose (PERF-01 vs PERF-02/INDX-06)
- **D-04:** **Two distinct corpora for two distinct jobs.**
  - **PERF-01 (published head-to-head):** run against the **installed TS `@colbymchenry/codegraph@1.3.1`** (confirmed present at `/opt/homebrew/bin/codegraph`) on a pinned set of **real OSS repos** (reuse the Phase-1/Phase-5 pinned corpora — `spf13/cobra`, `pallets/flask`, plus a few larger real repos), each pinned by commit SHA, raw per-repo numbers.
  - **PERF-02 (CI regression gate) + INDX-06 (100k+ monorepo):** gate against a **committed, deterministic, network-free synthetic 100k+-file corpus generator** so the CI gate is fully reproducible and never flakes on a remote clone. A **real large public monorepo** (pinned SHA, shallow-clone) MAY additionally be used to *publish* the headline INDX-06 number, but the **CI gate does not depend on network access**.

### Perf metric methodology + regression-gate policy (PERF-01, PERF-02, INDX-06)
- **D-05:** **Fair, external, median-of-N measurement with a committed baseline.**
  - **Metrics:** indexing throughput (files/s and bytes/s on a from-scratch `index --force`), query latency (median over a fixed query set), **peak RSS**, and cold start (`--version`/`status` wall time).
  - **Peak RSS measured at the OS level** (`getrusage` `ru_maxrss` / `/usr/bin/time -v` on the child process) rather than in-process — this is the only apples-to-apples way to compare the Go binary against the TS **Node** process, and it makes peak RSS a first-class externally-observable CI metric (INDX-06).
  - **median-of-5** runs per metric to damp noise.
  - **Regression gate:** a **committed baseline metrics JSON** + **tolerance-band failure** (starting points: throughput regression > 10%, peak RSS growth > 15% → fail; tune during planning), with a **documented, explicit re-bless path** to update the baseline intentionally.

### Repository layout + workflow structure
- **D-06:** **Locked release workflow + separate CI/bench workflows.**
  - `.github/workflows/release.yml` — **name and trigger are LOCKED** (see Locked/Carried Forward): must be exactly `release.yml`, trigger on tags matching `v[0-9]*`, and its cosign keyless signature must carry the SAN the Phase-6 verifier requires.
  - `.github/workflows/ci.yml` — test suite + `govulncheck` + reproducibility double-build gate + perf-regression gate (on PR/push).
  - `.github/workflows/bench.yml` — head-to-head publish, run on-demand/scheduled (not blocking).
  - `.goreleaser.yaml` at repo root.
  - Benchmark harness: a Go package under `internal/bench` (measurement) + `tools/bench` (runner, synthetic-corpus generator, pinned real-corpus manifest) — reuse the pinned-fixture pattern from `tools/spike/testdata/`.
  - Docs: `docs/RELEASE.md` (verify signature + provenance + SBOM commands) and `docs/BENCHMARKS.md` (methodology + raw numbers).

### Locked / Carried Forward (not gray areas)
- **Release identity is dictated by the already-shipped Phase-6 upgrade verifier** (`internal/upgrade/verify.go`). The release workflow MUST produce a cosign keyless signature whose certificate satisfies:
  - **Issuer:** `https://token.actions.githubusercontent.com` (`releaseOIDCIssuer`).
  - **SAN pattern:** `^https://github\.com/seanb4t/codegraph-go/\.github/workflows/release\.ya?ml@refs/tags/v[0-9][^\s]*$` (`releaseWorkflowRefPattern`).
  - ⇒ The workflow file MUST be named `release.yml`/`release.yaml`, live at `.github/workflows/`, and fire on **tag** refs `v[0-9]*`. **Renaming the workflow, changing the trigger, or signing from a different workflow silently breaks `codegraph upgrade` for every user.** If the workflow filename must change, `releaseWorkflowRefPattern` in `verify.go` must change in lockstep.
  - **Release repo slug:** `seanb4t/codegraph-go` (`releaseRepoSlug`) — the GitHub Releases the binary resolves/downloads from.
- **Release-asset contract is dictated by `internal/upgrade/upgrade.go` (also Phase-6-shipped):**
  - `defaultDownload` fetches a **raw binary** `<asset>` plus a per-binary bundle `<asset>.sigstore.json` from `releases/download/<tag>/`. ⇒ the release MUST publish raw binaries + a `.sigstore.json` sibling per binary (see D-02), not tar.gz archives.
  - `defaultVerify` hashes the **downloaded binary itself** (`sha256.Sum256(binary)`) — the signature must be over the binary, not the archive or checksums.
  - `releaseAssetName(version)` currently returns the placeholder `codegraph_<tag>_<goos>_<goarch>[.exe]` and is marked "Phase-8-finalized (D-14)". ⇒ the planner must **either** make GoReleaser's `name_template` produce exactly that string **or** update `releaseAssetName` (+ its tests) to match GoReleaser's real output. The two MUST agree or `codegraph upgrade` 404s on the asset.
- **Version stamping:** the release build MUST inject, via `-ldflags -X`, exactly these symbols (consumed by `codegraph version`):
  - `github.com/seanb4t/codegraph-go/internal/version.Version=<semver>`
  - `github.com/seanb4t/codegraph-go/internal/version.Commit=<sha>`
  - `github.com/seanb4t/codegraph-go/internal/version.Date=<RFC3339>`
- **Stack (from `CLAUDE.md`/`STACK.md`, do not re-decide):** GoReleaser v2, cosign v3 keyless, `slsa-framework/slsa-github-generator` (pinned tag), syft SBOM, `govulncheck`. Optional higher-fidelity `cyclonedx-gomod` may supplement syft.
- **CGo is the sole documented exception (DIST-05 Complete).** `CGO_ENABLED=1` is required (parser fails to build otherwise). No *new* CGo may be introduced.

### Claude's Discretion
- Exact tolerance-band percentages, median-N value, synthetic-corpus file/language mix, the specific extra real repos in the head-to-head set, GoReleaser template details, and doc phrasing are left to planning/execution — provided the success criteria and the locked constraints above hold.
- Whether the real 100k+ monorepo headline number is published from a manual/scheduled job vs a one-time captured artifact.
</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope & requirements
- `.planning/ROADMAP.md` — Phase 8 "Release Hardening & Benchmarks" goal + 4 success criteria; "scaffolds early, validates last".
- `.planning/REQUIREMENTS.md` — **DIST-01..05**, **PERF-01/02**, **INDX-06** (DIST-05 already Complete; the rest Pending).
- `.planning/PROJECT.md` — core value ("everything works the same or better — faster, from a single verifiably-built binary"); supply-chain + minimal-dependency constraints.

### Release-identity contract (LOCKED — the release workflow MUST satisfy this)
- `internal/upgrade/verify.go` §42–44 — `releaseOIDCIssuer`, `releaseRepoSlug`, `releaseWorkflowRefPattern`: the cosign identity the shipped binary already enforces. **The release workflow's name/trigger/signing identity are dictated here.**
- `internal/upgrade/upgrade.go` — `releaseWorkflowRefPattern` usage + the verify-before-swap upgrade flow that consumes released artifacts.
- `internal/upgrade/release.go` — how the binary resolves the latest release tag from `seanb4t/codegraph-go` (GitHub Releases redirect trick + API fallback); the release must publish tags in the `v...` shape this expects.
- `internal/version/version.go` §2–6 — the three `-ldflags -X` symbols (`Version`/`Commit`/`Date`) the release build must inject.

### Parser / CGo constraint (drives DIST-01 + DIST-04 difficulty)
- `PARSER-DECISION.md` — Option A (CGo tree-sitter) selected; **cross-compile validation explicitly deferred to Phase 8** (zig/wasi-sdk absent in the spike env); documents the `zig cc -target` approach and that `CGO_ENABLED=0` fails for the parser.
- `.planning/research/PITFALLS.md` §~20 — CGo cross-compile warning signs (Windows cross-build from non-Windows CI, per-platform libc variants, `CGO_ENABLED=0` conflicts).
- `.planning/research/STACK.md` — the pinned stack rationale (GoReleaser/cosign/SLSA/syft/govulncheck) + the `zig cc` cross-compile note.
- `go.mod` — CGo grammar modules (`tree-sitter/go-tree-sitter` + 14 grammars); the dependency tree the DIST-05 audit + SBOM describe (134 direct requires, bulk = grammar modules).

### Benchmark inputs
- `tools/spike/testdata/` + `tools/spike/testdata/ATTRIBUTION.md` — the pinned, network-free real-repo fixture pattern to reuse for the head-to-head corpus.
- `testdata/golden/corpus/colbymchenry-codegraph`, `testdata/golden/corpus/weft-go` — existing pinned corpus repos (candidate head-to-head inputs).
- Installed TS reference: `@colbymchenry/codegraph@1.3.1` at `/opt/homebrew/bin/codegraph` — the head-to-head comparison target (PERF-01).

### Project stack doctrine
- `.claude/CLAUDE.md` — the recommended stack table + "The Parser Decision" (CGo justified exception) + supply-chain tool choices; the authoritative source for what is pre-decided vs open.
</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **`internal/upgrade`** (Phase 6): the *consumer* of everything this phase produces — cosign keyless verify-before-swap, release-tag resolution, ldflags version identity. Its constants are the contract the release workflow must satisfy; its `verify_test.go` fixtures show the identity shape.
- **`internal/version/version.go`**: the ldflags injection points; `codegraph version --json` output the release build must populate.
- **`tools/spike/`**: an existing pinned-fixture, interface-driven benchmark harness (CGo vs wazero) — the closest prior art for the PERF harness structure, corpus pinning, and reproducible-numbers discipline.
- **Phase-6 CLI wiring** (`internal/cli`, `cmd/codegraph/main.go`): if any bench/verify subcommand is added, follow this registration pattern (though most of this phase is CI config, not new CLI surface).

### Established Patterns
- **Reproducibility-first fixtures** (Phase 1 spike): corpora pinned by commit + committed as static testdata, no network at test time — extend this to the perf corpora (esp. the synthetic 100k+ generator for the CI gate).
- **Fail-loud / verify-before-act ethos** (Phases 4/6/7 memory: deep review repeatedly caught bugs the green suite missed on I/O/crypto/network code): the release + verify path is exactly that profile. **Recommend a deep `/gsd-code-review` after execution**, especially on the signing/verify/reproducibility wiring, and treat a green CI as necessary-not-sufficient.
- **Additive, no-behavior-change discipline:** this phase must not perturb indexer/query/sync/migrate output — the reproducibility + determinism gates actually *protect* that invariant.

### Integration Points
- **CI ↔ upgrade contract:** the release workflow's cosign identity must exactly match `internal/upgrade/verify.go`. This is the single highest-consequence integration point in the phase.
- **CGo toolchain ↔ build matrix:** `zig cc` (or native-runner fallback) is wired into GoReleaser's build config; this is where DIST-01 succeeds or fails.
- **SLSA generic generator ↔ GoReleaser artifacts:** provenance runs over GoReleaser's checksums, not a re-build.
- **Benchmark harness ↔ installed TS 1.3.1:** the head-to-head runner shells out to `/opt/homebrew/bin/codegraph` for the TS side and the freshly-built Go binary for ours.
- **Flake note (carry forward):** `internal/daemon` `TestSoak` + flush-lock tests are known pre-existing flakes under full-suite parallel load. If `go test ./...` fails there in CI, re-run `go test ./internal/daemon/ -count=1` isolated before treating it as a regression — the new CI gate should account for this (e.g. `-count=1` isolation or a documented retry policy), not paper over it.
</code_context>

<specifics>
## Specific Ideas

- The **inversion** is the defining feature of this phase: the signature *verifier* (Phase 6) shipped before the *signer* (Phase 8). Treat `internal/upgrade/verify.go` as an executable spec for the release workflow, and add an end-to-end test that a real signed release artifact passes `verifyRelease` against the production identity (Phase 6 left a `verifyRelease`-against-fixture seam for exactly this).
- Peak RSS must be measured **outside** the process to fairly compare against the TS Node runtime — an in-process Go `runtime.MemStats` reading would be an unfair, non-comparable number.
- The "minimal audited dependency tree" story (DIST-05) has a subtlety worth stating in `docs/RELEASE.md`: the 134 direct requires are dominated by the 14 tree-sitter grammar modules; the SBOM + audit narrative should make that composition explicit rather than implying a small flat tree.
- Reproducibility is *easier* here than a generic CGo project because `zig` is deterministic — lean into pinning zig + Go versions as the reproducibility foundation.
</specifics>

<deferred>
## Deferred Ideas

- **Teach `sync`/`index` to reconcile a TS-migrated graph's ids without a full churn pass** (Phase-7 D-01 open question). A sync-engine concern, not release/benchmark scope — record it, do not fold it into Phase 8.
- **Package-manager distribution channels** (Homebrew tap, apt/deb, winget, `go install` convenience) beyond raw GitHub-release static binaries. DIST-01 only requires downloadable static binaries; distribution channels are a post-v1 nice-to-have.
- **Milestone-2 team-scale features** (central server, CI-distributed indexes, concurrent multi-writer access) — explicitly a later milestone; the v1 storage/process design already accommodates them, but building them is out of scope.
- **Promoting the reproducibility double-build gate to blocking on all 6 targets** (vs the linux/amd64-blocking + others-reported D-03 default) — revisit once the cross-target determinism numbers are known.

### Reviewed Todos (not folded)
None — no pending todos matched Phase 8 (`todo.match-phase 8` → 0 matches).
</deferred>

---

*Phase: 8-Release Hardening & Benchmarks*
*Context gathered: 2026-07-13*
