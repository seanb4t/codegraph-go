---
phase: "11"
slug: "graph-view-community-clustering"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: "2026-09-13"
---

# Phase 11 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Seeded from `11-RESEARCH.md` § Validation Architecture; every command and file path
> below was verified against `Taskfile.yml`, `web/package.json`, and the tree during research.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` stdlib (`task test:unit`, excludes `internal/daemon`; `task test:golden` for the frozen goldens); `vitest` via `pnpm -C web test` wrapped by `task web:test` (numTotalTests floor); committed Node `.mjs` scripts under `web/scripts/` for source-scan gates; Go in-process harness under `tools/` for the GRF-09 measurement (never Playwright — D-06) |
| **Config file** | `go.mod` (pins `go 1.26.6`; gains the `gonum.org/v1/gonum v0.17.0` direct require this phase); `web/vite.config.ts` (vitest); `Taskfile.yml` (new `check:gonum` target, D-14) |
| **Quick run command** | `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/query/... ./internal/uiserver/...` · `pnpm -C web test -- file-graph` |
| **Full suite command** | `GOTOOLCHAIN=go1.26.6 task test:unit` + `GOTOOLCHAIN=go1.26.6 task test:golden` + `task web:test` + `task proto:drift` + `task -s web:drift` + `task check:gonum` |
| **Estimated runtime** | ~45 s targeted Go · ~15 s vitest · ~2–3 min full Go suite · GRF-09 harness: guava store open + 3 cold clustering runs, seconds not minutes |

**Toolchain prefix (repo landmine).** Local Go is 1.27.1; `go.mod` pins 1.26.6 and `cockroachdb/swiss` breaks on newer toolchains. Prefix every local `go build` / `go test` / `go vet` / `go list` / `govulncheck` with `GOTOOLCHAIN=go1.26.6`. Research confirmed an unprefixed `go list -deps` fails with "updates to go.mod needed" until the direct require lands.

**Type-check gate (Phase 9 lesson, 09-06).** Any task touching `web/src` must re-run `pnpm -C web check` and gate on RC=0 plus `rg -o -i '\b0 errors\b' | wc -l` = 1 — piped `svelte-check` prints its machine format in uppercase. Re-run it after every fix pass, not only the test suites.

**Proto regeneration.** `task proto:gen` regenerates BOTH protos; `task proto:drift` proves the committed generated code matches. `FileGraphNode.community_id = 5` / `FileGraphResponse.community_count = 7` ship with their regenerated output and the `readonly_test.go` fixture update in the SAME commit (the 09-01 / 10-01 recipe). No new rpc: `wantUIServiceMethods` stays at 16.

**Threshold-first ordering (GRF-09, D-05).** `corpora/graph-cluster-threshold.json` is committed ALONE, before any harness or measurement commit, and is never rewritten by the verdict mechanism. The ancestry proof is a persisted check (D-08), not the manual `git merge-base` one-liner GRF-01 used.

---

## Sampling Rate

- **After every task commit:** the quick Go command scoped to the touched packages, plus `pnpm -C web test -- file-graph` and `pnpm -C web check` for any web-touching task
- **After every plan wave:** `GOTOOLCHAIN=go1.26.6 task test:unit` + `task test:golden` + `task web:test` + `task proto:drift`
- **Before `/gsd-verify-work`:** full suite green, `GOTOOLCHAIN=go1.26.6 go vet ./...`, `task -s web:drift` MATCH, `task check:gonum` green with its three positive controls, and the GRF-09 harness run recorded (PASS or FAIL with the D-07 fallback executed) with its observation file committed
- **Max feedback latency:** ~90 seconds (the GRF-09 harness is a phase gate, not a per-task sample)

---

## Per-Task Verification Map

Task IDs are assigned by the planner; this map binds requirements to their automated commands so
the planner can attach them. Test names are illustrative until the planner fixes them.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| TBD | TBD | 1 | GRF-09 | T-11 threshold tampering | `corpora/graph-cluster-threshold.json` exists in exactly one commit that is an ancestor of every commit touching the observation/verdict files; the check reports the threshold commit hash and the count of measurement commits compared | unit (Go, git-backed) | `GOTOOLCHAIN=go1.26.6 go test -count=1 -run 'TestClusterThresholdCommitIsAncestorOfEveryMeasurement' ./tools/...` | ❌ Wave 0 | ⬜ pending |
| TBD | TBD | 1 | GRF-10 | T-11 supply chain | `gonum.org/v1/gonum` direct require; `govulncheck` clean; SBOM names it (positive control: `github.com/cockroachdb/pebble` also present); cgo scan over the `graph/community` closure reports the package count and zero non-zero `CgoFiles` rows (positive control: a known-cgo package such as `github.com/tree-sitter/go-tree-sitter` reports > 0) | script (Taskfile) | `GOTOOLCHAIN=go1.26.6 task check:gonum` | ❌ Wave 0 | ⬜ pending |
| TBD | TBD | 1 | GRF-08 | T-11 non-determinism | ≥3 runs of the clustering on identical sorted input yield canonical-relabel-equal assignments; the test logs the run count it compared; a perturbed node order or seed (test-only seam) turns it RED | unit (Go) | `GOTOOLCHAIN=go1.26.6 go test -count=1 -run 'TestAssignCommunitiesDeterministic' -v ./internal/query/...` | ❌ Wave 0 (mirrors `TestFileGraphCyclesDeterministicIds`) | ⬜ pending |
| TBD | TBD | 1 | GRF-08 | — | Degenerate inputs: zero edges → each node its own community (ids 1..N); single node → community 1; empty graph → `community_count = 0`; ids are dense 1-based, 0 never emitted once computed | unit (Go) | `GOTOOLCHAIN=go1.26.6 go test -count=1 -run 'TestAssignCommunitiesDegenerate' ./internal/query/...` | ❌ Wave 0 | ⬜ pending |
| TBD | TBD | 1 | GRF-09 | — | Harness reads the threshold file (never its own defaults), opens the cached guava store, times `AssignCommunities` in isolation as median-of-3 cold runs, writes an observation with `verdict` PASS/FAIL and never rewrites the threshold; the run's exit status mirrors the verdict | harness (Go in-process) | `GOTOOLCHAIN=go1.26.6 go run ./tools/corpora -mode cluster-measure` (or sibling tool per planner) | ❌ Wave 0 | ⬜ pending |
| TBD | TBD | 2 | GRF-06 | — | `FileGraph()` populates `CommunityID` (1-based) on every node and `CommunityCount` on the result, fresh per call (D-15); on the wire as `community_id = 5` / `community_count = 7`; field-number fixture and 16-method set unchanged | unit + integration (Go) | `GOTOOLCHAIN=go1.26.6 go test -count=1 -run 'TestFileGraphPopulatesCommunityFields\|TestUIServiceMethodSetIsExactlyTheReadSet\|TestKnownUIProtoFieldNumbersAreStable' ./internal/query/... ./internal/uiserver/...` && `task proto:drift` | ❌ Wave 0 (extends `readonly_test.go`) | ⬜ pending |
| TBD | TBD | 2 | GRF-06 | — | Transform assigns a palette class/colour keyed by `communityId` on nodes; two nodes with equal `communityId` get equal colour, differing ids differ (within the 12-hue cycle); directory compounds stay neutral (D-11); layout config unchanged | unit (vitest) | `pnpm -C web test -- file-graph` | ❌ Wave 0 (extends `web/tests/file-graph-transform.test.ts`) | ⬜ pending |
| TBD | TBD | 2 | GRF-06 | — | Positive-controlled source scan over `web/src`: ≥1 `elk` layout reference found, zero matches for `cose\|fcose\|cola\|euler\|spread\|force` layout names; reports both counts | script (Node) | `node web/scripts/check-no-force-layout.mjs` | ❌ Wave 0 | ⬜ pending |
| TBD | TBD | 2 | GRF-06 | — | Graph view toolbar shows "N communities" and the count equals the distinct communities rendered | unit (vitest) | `pnpm -C web test -- graph-page` | ❌ Wave 0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `corpora/graph-cluster-threshold.json` — committed ALONE first (D-05); `maxClusteringMs: 500`, `corpus` pin (`google/guava@94f39958…`), `runs: 3`, `statistic: median`, pre-written `onFailure` naming the D-07 persistence fallback
- [ ] `internal/query/community.go` + `internal/query/community_test.go` — `AssignCommunities` over the `FileGraphResult` rollup, sorted-path node ids, fixed `rand.NewPCG` seed, canonical relabel; determinism + RED-control seam + degenerate cases
- [ ] `tools/…` GRF-09 harness (extend `tools/corpora -mode measure` or a sibling) + its git-backed ancestry test
- [ ] `Taskfile.yml` `check:gonum` target — govulncheck (pinned via `go.tool.mod`, precedent `Taskfile.yml:1514`), `syft` SBOM grep with positive control, `go list -deps` cgo scan with positive control
- [ ] `internal/uiproto/uiv1/ui.proto` — `FileGraphNode.community_id = 5`, `FileGraphResponse.community_count = 7`; regenerated output + `readonly_test.go` fixture in the same commit
- [ ] `web/src/lib/components/graph/file-graph-transform.ts` / `graph-style.ts` — `communityId` on element data, 12-hue palette class generator alongside the `cycleDiscriminatorClass` precedent; `web/tests/file-graph-transform.test.ts` extended
- [ ] `web/scripts/check-no-force-layout.mjs` — positive-controlled layout-name scan
- [ ] `11-MUTATION-LOG.md` — families: seed perturbation / node-order perturbation (GRF-08), threshold widened after measurement (GRF-09 ancestry), a force-layout name injected into `web/src` (GRF-06 scan), cgo package injected into the scan's positive control (GRF-10) — each RED, byte-cleanly reverted
- [ ] `11-SECURITY.md` — threat register per D-16
- [ ] Framework install: `gonum.org/v1/gonum v0.17.0` promoted to a direct require (already in `go.sum` transitively); `govulncheck` and `syft` present locally (research verified)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Community colours are distinguishable on the live layered layout and do not clash with the fixed file-node blue / cycle-border red (research assumption A4) | GRF-06 | Cosmetic judgement over a real render | `codegraph ui` on this repo → Graph view → confirm coloured groups on the unchanged ELK layout and the "N communities" line; kill the process afterwards |

*All other phase behaviors have automated verification.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 90s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
