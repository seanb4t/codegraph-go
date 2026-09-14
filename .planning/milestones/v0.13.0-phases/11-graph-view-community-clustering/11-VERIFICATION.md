---
phase: 11-graph-view-community-clustering
verified: 2026-09-13T22:19:27Z
status: passed
score: 12/12 must-haves verified
covered_files: [".planning/phases/11-graph-view-community-clustering/11-01-PLAN.md",".planning/phases/11-graph-view-community-clustering/11-01-SUMMARY.md",".planning/phases/11-graph-view-community-clustering/11-02-PLAN.md",".planning/phases/11-graph-view-community-clustering/11-02-SUMMARY.md",".planning/phases/11-graph-view-community-clustering/11-03-PLAN.md",".planning/phases/11-graph-view-community-clustering/11-03-SUMMARY.md",".planning/phases/11-graph-view-community-clustering/11-04-PLAN.md",".planning/phases/11-graph-view-community-clustering/11-04-SUMMARY.md",".planning/phases/11-graph-view-community-clustering/11-05-PLAN.md",".planning/phases/11-graph-view-community-clustering/11-05-SUMMARY.md","Taskfile.yml","corpora/graph-cluster-observations.json","corpora/graph-cluster-threshold.json","go.mod","internal/query/community.go","internal/query/community_test.go","internal/query/traverse.go","internal/uiproto/uiv1/ui.pb.go","internal/uiproto/uiv1/ui.proto","internal/uiproto/uiv1/uiv1connect/ui.connect.go","internal/uiserver/filegraph_test.go","internal/uiserver/handlers.go","internal/uiserver/readonly_test.go","tools/graphcluster/ancestry_test.go","tools/graphcluster/main.go","tools/graphcluster/main_test.go","web/scripts/check-no-force-layout.mjs","web/src/lib/components/graph/community-palette.ts","web/src/lib/components/graph/file-graph-transform.ts","web/src/lib/components/graph/graph-style.ts","web/src/lib/gen/ui_pb.ts","web/src/routes/graph/+page.svelte","web/tests/file-graph-transform.test.ts","web/tests/graph-communities.test.ts"]
covered_digest: "v1:sha256:e923cff7dce3c624b0b7715b1d5b899a1ce518a4108de6d410a450f7776238fb"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: passed
  previous_verified: "2026-09-13T19:20:00Z"
  reason: "milestone-close re-pin — canonical status read stale (Taskfile.yml edited by Phase 12: new docs:cli / docs:cli:drift targets added elsewhere in the file changed the file's content hash); all must-haves re-executed at HEAD 8c8149de"
  gaps_closed: []
  gaps_remaining: []
  regressions: []
human_verification:
  - test: "Run `codegraph ui` on this repository, open the Graph view, and visually confirm: (a) file nodes show distinct community colours on the SAME layered (ELK) layout as before — no re-layout jump; (b) collapsed/expanded directory compound nodes stay neutral (border only, no fill colour); (c) the toolbar/legend line 'Files fall into N communities …' appears under the cycle summary and its N matches the visible grouping; (d) a cycle-bordered node's red cycle border still reads clearly on top of its community fill colour."
    expected: "Distinct, colour-blind-safe hues group related files; layout position is unchanged from pre-phase; directories render with no community fill; the toolbar count line renders; cycle borders remain visible over community colours."
    status: resolved
    resolution: "Performed 2026-09-13 by the orchestrator against a live `codegraph ui` on this repository (binary built from HEAD 77d4a9c6, server stopped afterwards; accepted by the user). (a) After expanding internal/query, internal/uiserver and internal/indexer via the cytoscape instance: 132 file nodes rendered with 8 distinct graph-community-N classes and 8 distinct palette colours (community-0 rgb(42,120,214), -1 rgb(235,104,52), -2 rgb(27,175,122), -3 rgb(237,161,0), -5 rgb(0,131,0), -6 rgb(74,58,167), -7 rgb(227,73,72), -10 rgb(162,89,217)); 0 nodes where one communityId mapped to two classes; GraphCanvas.svelte byte-identical to the threshold commit so the ELK layout is unchanged. (b) 0 directory compounds carried a community class (109 directories at the default view, neutral). (c) Toolbar line rendered verbatim: \"Files fall into 205 communities, shown as node colours on the same layout.\" (d) Cycle nodes compose both class sets (e.g. `graph-cycle graph-cycle-2 graph-community-7`) and the red cycle border reads over the community fill (screenshot p11-graph-communities-indexer.png in the session scratchpad). Console showed 4 entries (CSP-blocked svelte-logo data URI, 2x text-valign warning, cytoscape notify TypeError at load); a pre-phase binary built from 18ef2434 produced the identical 4 entries, so none are Phase 11 regressions. corpora/breadcrumb-check.json untouched; no codegraph ui process left running."
    why_human: "This is the one visual/runtime confirmation deferred by 11-04-PLAN Task 2's own <human-check> block under workflow.human_verify_mode=end-of-phase — the underlying logic (colour==community mapping, wire count passthrough, layout byte-identity, no-force-layout scan) is machine-verified above, but actually opening the rendered graph view is not something grep/unit tests can observe. After testing, kill the `codegraph ui` process (`pkill -f 'codegraph ui'`) and confirm no `corpora/breadcrumb-check.json` diff was left behind."
---

# Phase 11: Graph View — Community Clustering Verification Report

**Phase Goal:** The file/package graph reads as groups rather than a flat mesh — nodes coloured by community, computed fresh and deterministically inside `FileGraph()`, on the layered layout that already shipped. GRF-09's pass threshold is committed in its own commit before any measurement runs, and no clustering is wired into the UI before that measurement resolves.
**Verified:** 2026-09-13T22:19:27Z
**Status:** passed
**Re-verification:** Yes — milestone-close re-verification (previous: passed, 12/12, 2026-09-13T19:20:00Z)

This is a milestone-close re-verification (v0.13.0, Phases 7–12 all complete). The prior report's `covered_files` list included `Taskfile.yml`, which Phase 12 legitimately edited (added the unrelated `docs:cli` / `docs:cli:drift` targets elsewhere in the file), changing the file's content hash and making the canonical `verification.status` read `stale` even though nothing this phase actually depends on — `check:gonum` — was touched. This run re-executed every one of the 12 previously-verified truths against current HEAD (`8c8149de`), confirmed the `check:gonum` task body is untouched by the Phase 12 edit, and re-derived `covered_digest` over the same file list at its current content. `git status --porcelain` was clean before and after (no tracked file other than this report was modified).

## Re-verification Summary

- **Root cause of staleness confirmed:** `git diff 698235a2 HEAD -- Taskfile.yml` shows only new `docs:cli:` / `docs:cli:drift:` task blocks added by Phase 12 (commit `302e5fb1`); the `check:gonum:` task body (lines 1644–1795 in current `Taskfile.yml`) is unmodified — same desc, preconditions, and script content re-run below with identical output to the original verification.
- **No regressions found.** All 12 truths, all required artifacts, all key links, and all behavioral spot-checks re-executed at HEAD produce output identical (same PASS/FAIL verdicts, same counts, same digests) to the original 2026-09-13T19:20:00Z verification.
- **Human verification entry preserved verbatim**, including its `status: resolved` and `resolution:` fields — that live UAT was performed against this repository's actual Graph view and remains valid; nothing in Phase 12 touched the graph rendering path.

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence (re-executed at HEAD `8c8149de`) |
|---|-------|--------|--------------------------------------------|
| 1 | (GRF-09 SC1) `corpora/graph-cluster-threshold.json` is the phase's own, solitary, first commit; its recorded verdict is preserved verbatim; a FAIL would route to the documented 50–59 fallback, never a raised bar | ✓ VERIFIED | `git log --format=%H -- corpora/graph-cluster-threshold.json \| wc -l` = 1 (commit `698235a2`, unchanged). Re-ran `go test -v -run 'TestClusterThresholdCommitIsAncestorOfEveryMeasurement\|TestClusterThresholdDigestMatchesCommittedObservation' ./tools/graphcluster/...`: both PASS, logging "threshold commit 698235a2…; compared 2 measurement commits" and "threshold digest sha256:6adc67…matches; verdict PASS; median 106 ms" — identical to the original verification. `git diff --quiet -- corpora/graph-cluster-threshold.json` clean (untouched since). |
| 2 | (GRF-06 SC2) Opening the graph view shows nodes coloured by community on the existing layered layout; layout is unchanged; a check reports the distinct-community count; no force-directed mode is reachable from any code path | ✓ VERIFIED | `git diff --quiet 698235a2 HEAD -- web/src/lib/components/graph/GraphCanvas.svelte` — exit 0, byte-identical (confirmed directly this run, single `name: 'elk'` layout reference unchanged). Re-ran `task -s check:no-force-layout` live: `{"filesScanned":106,"elkLayoutRefs":2,"elkImportRefs":3,"forbiddenMatches":[],"unresolvedLayoutNames":[{"file":"...GraphCanvas.svelte","line":374,...}],"verdict":"PASS"}` plus the 3 self-tests, all PASS — identical to the original run. Re-ran `pnpm -C web exec vitest run tests/file-graph-transform.test.ts tests/graph-communities.test.ts`: 22/22 tests pass, including the "distinct colours: 12 over 12 distinct ids" assertion. Rendered visual confirmation remains the live UAT resolved and preserved in Human Verification below (unaffected by the Phase 12 Taskfile edit). |
| 3 | (GRF-08 SC3) Running clustering ≥3 times on identical input yields label-canonicalized-equal assignments, reporting the compared-run count; a perturbed iteration order or seed turns it red | ✓ VERIFIED | Re-ran `go test -v -run 'TestAssignCommunitiesDeterministic\|TestAssignCommunitiesSeedPerturbationFlipsTheTieFixture\|TestAssignCommunitiesCanonicalRelabelNeutralisesInsertionOrder\|TestAssignCommunitiesDegenerate\|TestUndirectedPairWeightsSumBothDirections\|TestFileGraphPopulatesCommunityFields\|TestFileGraphCommunitySourceIsFreshComputeOnly' ./internal/query/...`: all PASS. `structured` and `tie` sub-tests log "compared 5 runs" (≥3, unchanged). RED-control logs "seed offset 1 flipped the tie fixture" / "structured fixture invariant under the flipping offset" — same discriminating proof as original. |
| 4 | (GRF-10 SC4) `gonum.org/v1/gonum` passes govulncheck, appears by name in the SBOM, and its import closure is verified cgo-free with a reported package count | ✓ VERIFIED | Re-ran `GOTOOLCHAIN=go1.26.6 task -s check:gonum` live against the current (Phase-12-edited) `Taskfile.yml`: exit 0 — "govulncheck (source mode, main module) clean — 26 gonum packages in the scanned set"; "SBOM lists 148 packages; gonum.org/v1/gonum present 1 time(s) at v0.17.0; positive control github.com/cockroachdb/pebble/v2 present 1 time(s)"; "cgo closure over gonum.org/v1/gonum/graph/community — 91 packages inspected, 0 with cgo (want 0), blas/cgo references 0 (want 0); positive control github.com/tree-sitter/go-tree-sitter — 3 with cgo (want >= 1)"; final "check:gonum: PASS" — every number identical to the original verification, proving the Phase 12 edit (new `docs:cli`/`docs:cli:drift` targets added elsewhere in the file) did not alter this task's behavior. GRF-10's `[ASSUMED]` package-legitimacy note (recorded in `11-SECURITY.md`) carries forward unchanged. |
| 5 | `go.mod` gains exactly one direct-require line for gonum v0.17.0; `go.sum` unchanged | ✓ VERIFIED | Re-derived: `git diff --numstat 698235a2 -- go.mod` = `1 0`; `git diff --quiet 698235a2 -- go.sum` clean; `GOTOOLCHAIN=go1.26.6 go list -m gonum.org/v1/gonum` prints `gonum.org/v1/gonum v0.17.0`. |
| 6 | On a PASS verdict, `Node`'s reserved 50-59 field range stays untouched and no fallback code path was introduced | ✓ VERIFIED | `rg -o 'reserved 50 to 59' internal/schema/graph.proto \| wc -l` = 3 (unchanged); `internal/community` package still does not exist; `rg -o 'GetCommunityId' internal/query/traverse.go internal/query/community.go` = 0; `AssignCommunities(result.Nodes, result.Edges)` call site present exactly once in `traverse.go`. |
| 7 | Wire fields `community_id = 5` / `community_count = 7` exist, are additive, and are field-for-field projected from the engine result over the real listener | ✓ VERIFIED | Re-ran `go test -v -run 'TestFileGraphProjectsEngineResult\|TestUIServiceMethodSetIsExactlyTheReadSet\|TestUIProtoFieldNumbersAreStableAndUnique' ./internal/uiserver/...`: all PASS ("communities on the wire: 2 over 4 nodes"; "inspected 49 messages and 202 fields in the generated uiv1 descriptor"). |
| 8 | No `web/src` file other than the regenerated `web/src/lib/gen/ui_pb.ts` changed before the GRF-09 verdict resolved | ✓ VERIFIED | `git diff --name-only 698235a2 41e118fa -- web/src` still returns only `web/src/lib/gen/ui_pb.ts` (history is immutable — this check is unaffected by later commits). |
| 9 | Build/vet/type-check are clean across the whole repo at HEAD | ✓ VERIFIED | Re-ran `GOTOOLCHAIN=go1.26.6 go build ./...` and `go vet ./...` at HEAD `8c8149de`: both clean. |
| 10 | Web test suite is green, including the new community tests | ✓ VERIFIED | Re-ran the phase's own graph vitest files (`file-graph-transform.test.ts`, `graph-communities.test.ts`): 2 files / 22 tests passed. The Phase 12 regression pass already re-ran the full web suite (584/584) at `af95a438`, cited here for the suite-wide claim per this run's landmine guidance rather than re-running the full 49-file suite a second time in a five-verifier-concurrent window. |
| 11 | The three code-review warnings (WR-01, WR-02, WR-03) raised in iteration 1 were fixed, not merely acknowledged | ✓ VERIFIED | Unchanged since original verification — no commit since `77d4a9c6` touches `community-palette.ts` or `check-no-force-layout.mjs`. Re-ran `task -s check:no-force-layout --self-test`-equivalent (the self-tests run automatically as part of the task) live this session: WR-02 symlink-traversal self-test PASS, WR-03b unresolved-name self-test PASS, injected-'cose' self-test PASS. |
| 12 | The phase's own mutation-testing discipline (RED-controlled discriminators for all four guard families) executed cleanly and left the tree unmodified | ✓ VERIFIED | `11-MUTATION-LOG.md` unchanged since original verification (not touched by Phase 12). `git status --porcelain` at HEAD `8c8149de` clean before and after this re-verification session, consistent with the guard-family mutations having stayed reverted. |

**Score:** 12/12 truths verified (0 present-but-behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `corpora/graph-cluster-threshold.json` | GRF-09 pass bar, alone, first commit | ✓ VERIFIED | Single commit `698235a2`, unchanged |
| `corpora/graph-cluster-observations.json` | Committed measurement with verdict | ✓ VERIFIED | verdict PASS, median 106ms, digest matches |
| `internal/query/community.go` | Deterministic Louvain engine | ✓ VERIFIED | Re-inspected: `AssignCommunities`, `assignCommunitiesWith`, `undirectedPairWeights`, `canonicalCommunityLabels`, seam `communityOptions`, single package var — unchanged |
| `internal/query/traverse.go` | Fresh-per-call wiring into `FileGraph()` | ✓ VERIFIED | `CommunityID`/`CommunityCount` fields, single `AssignCommunities` call site |
| `tools/graphcluster/main.go` + tests | GRF-09 measurement harness | ✓ VERIFIED | 9/9 tests re-run PASS this session |
| `tools/graphcluster/ancestry_test.go` | Ancestry + digest proof | ✓ VERIFIED | Both tests re-run PASS (not SKIP) at HEAD `8c8149de`, full clone |
| `internal/uiproto/uiv1/ui.proto` + gen | Wire fields | ✓ VERIFIED | `community_id=5`, `community_count=7`, 16 rpcs unchanged |
| `web/src/lib/components/graph/community-palette.ts` | 12-hue palette + guarded index fn | ✓ VERIFIED | Unchanged since original verification |
| `web/src/lib/components/graph/file-graph-transform.ts` | communityId passthrough | ✓ VERIFIED | Re-tested this session (22/22 tests) |
| `web/src/lib/components/graph/graph-style.ts` | 12 static colour rules | ✓ VERIFIED | Unchanged since original verification |
| `web/src/routes/graph/+page.svelte` | Toolbar community count line | ✓ VERIFIED | Unchanged since original verification |
| `web/scripts/check-no-force-layout.mjs` + Taskfile target | Force-layout scan | ✓ VERIFIED | Re-run live this session, PASS, self-tests PASS |
| `Taskfile.yml` `check:gonum` | GRF-10 supply-chain gate | ✓ VERIFIED | Re-run live against the Phase-12-edited file this session — task body untouched, output identical to original run |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| `FileGraph()` rollup | `AssignCommunities` | direct call after cycle block | ✓ WIRED | `traverse.go:347`, re-confirmed |
| `FileGraphResult` | proto `FileGraphResponse`/`FileGraphNode` | `fileGraphToProto`/`fileGraphNodeToProto` | ✓ WIRED | `TestFileGraphProjectsEngineResult` re-run PASS |
| Wire `community_id`/`community_count` | `web/src/lib/gen/ui_pb.ts` → `file-graph-transform.ts` → `graph-style.ts` render | generated client → transform → static CSS-like rules | ✓ WIRED | Re-tested this session (22/22) |
| `corpora/graph-cluster-threshold.json` (first commit) | `tools/graphcluster` reads it | `loadThreshold` | ✓ WIRED | Re-run PASS |
| threshold commit | every measurement commit | git ancestry | ✓ WIRED | Re-run PASS |

### Behavioral Spot-Checks (all re-executed at HEAD `8c8149de`)

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| GRF-08 determinism + RED control | `go test -v -run 'TestAssignCommunities...' ./internal/query/...` | all PASS, "compared 5 runs" x2, seed-flip logged | ✓ PASS |
| GRF-09 harness unit tests | `go test -v ./tools/graphcluster/...` | 9/9 PASS incl. ancestry+digest (not skipped) | ✓ PASS |
| GRF-06 force-layout scan | `task -s check:no-force-layout` | PASS, 106 files scanned, 0 forbidden, 1 advisory | ✓ PASS |
| GRF-10 supply-chain gate | `GOTOOLCHAIN=go1.26.6 task -s check:gonum` | exit 0, PASS, all 4 numbers identical to original run | ✓ PASS |
| Go build/vet | `go build ./... && go vet ./...` | clean | ✓ PASS |
| Web graph tests | `pnpm -C web exec vitest run tests/file-graph-transform.test.ts tests/graph-communities.test.ts` | 22/22 passed | ✓ PASS |
| Full web suite (cited, not re-run) | Phase 12 regression pass at `af95a438` | 584/584 passed, per landmine guidance not re-run mid five-verifier concurrent window | ✓ PASS (cited) |

### Probe Execution

Not applicable — this phase has no `scripts/*/tests/probe-*.sh`; `tools/graphcluster` and `check-no-force-layout.mjs`/`check:gonum` (covered above under Behavioral Spot-Checks) remain this phase's runnable verification instruments and were re-executed directly this session.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| GRF-06 | 11-04 | Colour by community on unchanged layered layout, no force-directed | ✓ SATISFIED | Truth 2, mechanism re-verified; visual confirmation preserved from prior human verification |
| GRF-08 | 11-01 | Deterministic assignment, ≥3-run proof, RED control | ✓ SATISFIED | Truth 3 |
| GRF-09 | 11-01/11-02 | Threshold committed first, measured, verdict preserved, fallback documented | ✓ SATISFIED | Truth 1, 6 |
| GRF-10 | 11-03 | govulncheck clean, SBOM presence, cgo-free closure | ✓ SATISFIED | Truth 4, re-run against Phase-12-edited Taskfile.yml with identical output |

No orphaned requirements — `.planning/REQUIREMENTS.md`'s Phase 11 row lists exactly GRF-06, GRF-08, GRF-09, GRF-10, all four claimed and satisfied.

### Anti-Patterns Found

None. Re-scanned all phase-changed source/test/config files for `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER`/empty-implementation patterns this session — the only hits are pre-existing, non-debt matches already documented in the original verification: intentional `NOTE: inert placeholder` comments in `internal/uiserver/handlers.go` and `internal/uiserver/readonly_test.go` describing a deliberately-locked field (CR-02), a base64-encoded proto file descriptor blob in `ui_pb.ts` that coincidentally contains the substring "placeholder" test-function naming references, and one unrelated pre-existing `XXXXXX` `mktemp` template pattern in `Taskfile.yml`. No new debt markers were introduced by Phase 12's edit to `Taskfile.yml`.

### Ordering Invariants (explicitly re-derived from git this session)

- `git log --format=%H -- corpora/graph-cluster-threshold.json | wc -l` = **1** (commit `698235a2`, unchanged).
- Every commit touching `corpora/graph-cluster-observations.json` and `tools/graphcluster/` descends from `698235a2` (re-confirmed via the re-run `TestClusterThresholdCommitIsAncestorOfEveryMeasurement`).
- No hand-authored `web/src` change (other than `web/src/lib/gen/**`) precedes the observation commit `41e118fa`: `git diff --name-only 698235a2 41e118fa -- web/src` still shows only `web/src/lib/gen/ui_pb.ts` (immutable history, unaffected by later phases).
- `git diff --quiet 698235a2 HEAD -- web/src/lib/components/graph/GraphCanvas.svelte` — exit 0, confirming the ELK layout code path is unchanged all the way to current HEAD (`8c8149de`), not merely at the original verification's HEAD (`77d4a9c6`).

### GRF-10 `[ASSUMED]` Note (reported, not a gap)

Carried forward unchanged: `gonum.org/v1/gonum`'s package-legitimacy verdict is recorded verbatim in `11-SECURITY.md` as `[ASSUMED]` — no npm/pypi/crates legitimacy seam covers a Go module, so no machine verdict was possible. The govulncheck/SBOM/cgo-closure evidence (Truth 4) is machine-verified and was re-confirmed identical against the Phase-12-edited `Taskfile.yml` this session; only the qualitative legitimacy judgment remains an explicitly-flagged assumption.

### Human Verification Required

1. **Visual confirmation of community colouring on the live graph view** — RESOLVED 2026-09-13 (see `resolution` under `human_verification` in the frontmatter, preserved verbatim from the original verification): performed live against `codegraph ui` on this repository and accepted by the user. 132 file nodes across three expanded directories rendered with 8 distinct community colours and 0 class/id inconsistencies, 0 coloured directory compounds, the "Files fall into 205 communities" toolbar line, cycle borders composing over community fills, and a byte-identical ELK layout. This re-verification confirms `GraphCanvas.svelte` remains byte-identical to the threshold commit all the way to current HEAD, so the resolved UAT still applies without needing to be re-performed.

### Gaps Summary

No gaps. This milestone-close re-verification re-executed all 12 previously-verified truths, all required artifacts, all key links, and all behavioral spot-checks against current HEAD (`8c8149de`) and found zero regressions: every command, test, and gate produced output identical to the original 2026-09-13T19:20:00Z verification. The root cause of the canonical `stale` read — Phase 12's Taskfile.yml edit adding unrelated `docs:cli`/`docs:cli:drift` targets — was confirmed to leave the `check:gonum` task body byte-for-byte untouched, and `check:gonum` was re-run live against the current file to prove its behavior is unaffected. The previously-resolved human verification (live UAT of community colouring in the Graph view) is preserved verbatim and remains valid since `GraphCanvas.svelte` is confirmed byte-identical to the threshold commit through current HEAD.

---

_Verified: 2026-09-13T22:19:27Z_
_Verifier: Claude (gsd-verifier)_
