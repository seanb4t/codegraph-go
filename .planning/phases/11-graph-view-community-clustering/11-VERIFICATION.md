---
phase: 11-graph-view-community-clustering
verified: 2026-09-13T19:20:00Z
status: passed
score: 12/12 must-haves verified
covered_files: [".planning/phases/11-graph-view-community-clustering/11-01-PLAN.md",".planning/phases/11-graph-view-community-clustering/11-01-SUMMARY.md",".planning/phases/11-graph-view-community-clustering/11-02-PLAN.md",".planning/phases/11-graph-view-community-clustering/11-02-SUMMARY.md",".planning/phases/11-graph-view-community-clustering/11-03-PLAN.md",".planning/phases/11-graph-view-community-clustering/11-03-SUMMARY.md",".planning/phases/11-graph-view-community-clustering/11-04-PLAN.md",".planning/phases/11-graph-view-community-clustering/11-04-SUMMARY.md",".planning/phases/11-graph-view-community-clustering/11-05-PLAN.md",".planning/phases/11-graph-view-community-clustering/11-05-SUMMARY.md","Taskfile.yml","corpora/graph-cluster-observations.json","corpora/graph-cluster-threshold.json","go.mod","internal/query/community.go","internal/query/community_test.go","internal/query/traverse.go","internal/uiproto/uiv1/ui.pb.go","internal/uiproto/uiv1/ui.proto","internal/uiproto/uiv1/uiv1connect/ui.connect.go","internal/uiserver/filegraph_test.go","internal/uiserver/handlers.go","internal/uiserver/readonly_test.go","tools/graphcluster/ancestry_test.go","tools/graphcluster/main.go","tools/graphcluster/main_test.go","web/scripts/check-no-force-layout.mjs","web/src/lib/components/graph/community-palette.ts","web/src/lib/components/graph/file-graph-transform.ts","web/src/lib/components/graph/graph-style.ts","web/src/lib/gen/ui_pb.ts","web/src/routes/graph/+page.svelte","web/tests/file-graph-transform.test.ts","web/tests/graph-communities.test.ts"]
covered_digest: "v1:sha256:8ae4be1997819ab55ae30486f495560626d8aa621b6498427d13065ea603177c"
behavior_unverified: 0
overrides_applied: 0
human_verification:
  - test: "Run `codegraph ui` on this repository, open the Graph view, and visually confirm: (a) file nodes show distinct community colours on the SAME layered (ELK) layout as before — no re-layout jump; (b) collapsed/expanded directory compound nodes stay neutral (border only, no fill colour); (c) the toolbar/legend line 'Files fall into N communities …' appears under the cycle summary and its N matches the visible grouping; (d) a cycle-bordered node's red cycle border still reads clearly on top of its community fill colour."
    expected: "Distinct, colour-blind-safe hues group related files; layout position is unchanged from pre-phase; directories render with no community fill; the toolbar count line renders; cycle borders remain visible over community colours."
    status: resolved
    resolution: "Performed 2026-09-13 by the orchestrator against a live `codegraph ui` on this repository (binary built from HEAD 77d4a9c6, server stopped afterwards; accepted by the user). (a) After expanding internal/query, internal/uiserver and internal/indexer via the cytoscape instance: 132 file nodes rendered with 8 distinct graph-community-N classes and 8 distinct palette colours (community-0 rgb(42,120,214), -1 rgb(235,104,52), -2 rgb(27,175,122), -3 rgb(237,161,0), -5 rgb(0,131,0), -6 rgb(74,58,167), -7 rgb(227,73,72), -10 rgb(162,89,217)); 0 nodes where one communityId mapped to two classes; GraphCanvas.svelte byte-identical to the threshold commit so the ELK layout is unchanged. (b) 0 directory compounds carried a community class (109 directories at the default view, neutral). (c) Toolbar line rendered verbatim: \"Files fall into 205 communities, shown as node colours on the same layout.\" (d) Cycle nodes compose both class sets (e.g. `graph-cycle graph-cycle-2 graph-community-7`) and the red cycle border reads over the community fill (screenshot p11-graph-communities-indexer.png in the session scratchpad). Console showed 4 entries (CSP-blocked svelte-logo data URI, 2x text-valign warning, cytoscape notify TypeError at load); a pre-phase binary built from 18ef2434 produced the identical 4 entries, so none are Phase 11 regressions. corpora/breadcrumb-check.json untouched; no codegraph ui process left running."
    why_human: "This is the one visual/runtime confirmation deferred by 11-04-PLAN Task 2's own <human-check> block under workflow.human_verify_mode=end-of-phase — the underlying logic (colour==community mapping, wire count passthrough, layout byte-identity, no-force-layout scan) is machine-verified above, but actually opening the rendered graph view is not something grep/unit tests can observe. After testing, kill the `codegraph ui` process (`pkill -f 'codegraph ui'`) and confirm no `corpora/breadcrumb-check.json` diff was left behind."
---

# Phase 11: Graph View — Community Clustering Verification Report

**Phase Goal:** The file/package graph reads as groups rather than a flat mesh — nodes coloured by community, computed fresh and deterministically inside `FileGraph()`, on the layered layout that already shipped. GRF-09's pass threshold is committed in its own commit before any measurement runs, and no clustering is wired into the UI before that measurement resolves.
**Verified:** 2026-09-13T19:20:00Z
**Status:** passed
**Re-verification:** No — initial verification

All evidence below was reproduced live against HEAD (`77d4a9c6`) in this session — tests re-run, scripts re-executed, git history re-derived — not read off SUMMARY.md transcripts. `git status --porcelain` was clean before and after.

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | (GRF-09 SC1) `corpora/graph-cluster-threshold.json` is the phase's own, solitary, first commit; its recorded verdict is preserved verbatim; a FAIL would route to the documented 50–59 fallback, never a raised bar | ✓ VERIFIED | `git log --format=%H -- corpora/graph-cluster-threshold.json \| wc -l` = 1 (commit `698235a2`, `git show --name-only --format=` for that commit lists exactly that one path). `git merge-base --is-ancestor 698235a2 <c>` = 0 (true) for every commit touching `corpora/graph-cluster-observations.json` and `tools/graphcluster/` (`38b96025`, `41e118fa`, `3eb6978e`, `e9b28e88`). Re-ran `go test ./tools/graphcluster/...`: `TestClusterThresholdCommitIsAncestorOfEveryMeasurement` and `TestClusterThresholdDigestMatchesCommittedObservation` both PASS (not SKIP — this is a full clone), logging "threshold commit 698235a2…; compared 2 measurement commits" and "threshold digest sha256:6adc67…matches; verdict PASS; median 106 ms". Committed observation (`corpora/graph-cluster-observations.json`) verdict is `PASS`, median 106 ms vs max 500 ms (4.7x margin), matching commit `41e118fa`'s message verbatim. Verdict fell PASS, so the fallback was correctly never exercised (see Truth 6 below) — no evidence of a raised bar anywhere in the threshold file (`git diff --quiet -- corpora/graph-cluster-threshold.json` clean; single commit, untouched since). |
| 2 | (GRF-06 SC2) Opening the graph view shows nodes coloured by community on the existing layered layout; layout is unchanged; a check reports the distinct-community count; no force-directed mode is reachable from any code path | ✓ VERIFIED | Mechanism: `GraphCanvas.svelte` is byte-identical to the threshold commit (`git diff --quiet 698235a2 HEAD -- web/src/lib/components/graph/GraphCanvas.svelte`), single `name: 'elk'` layout reference. `file-graph-transform.ts` copies `communityId` from the wire (never invents it) and attaches `graph-community-N` only to file nodes. `graph-style.ts` generates 12 static per-community colour rules. `+page.svelte` renders "Files fall into N communities" from the wire `communityCount` verbatim. Re-ran `node web/scripts/check-no-force-layout.mjs` live: `{"filesScanned":106,"elkLayoutRefs":2,"elkImportRefs":3,"forbiddenMatches":[],"verdict":"PASS"}`; `--self-test` (3 self-tests including the WR-02 symlink-traversal and WR-03b unresolved-name additions) PASS. Re-ran the web suite (`pnpm test`): 584/584 tests pass including `graph-communities.test.ts`'s "distinct colours: 12 over 12 distinct ids" assertion and the route-level community-count-line tests. Rendered visual confirmation ("opening the graph view on a real repo") is the one item this phase's own plan deferred to end-of-phase human verify (11-04-PLAN Task 2's `<human-check>`), never actually executed by any autonomous plan run — routed to Human Verification below. |
| 3 | (GRF-08 SC3) Running clustering ≥3 times on identical input yields label-canonicalized-equal assignments, reporting the compared-run count; a perturbed iteration order or seed turns it red | ✓ VERIFIED | Re-ran `go test -v -run 'TestAssignCommunitiesDeterministic\|TestAssignCommunitiesSeedPerturbationFlipsTheTieFixture\|TestAssignCommunitiesCanonicalRelabelNeutralisesInsertionOrder\|TestAssignCommunitiesDegenerate\|TestUndirectedPairWeightsSumBothDirections\|TestFileGraphPopulatesCommunityFields\|TestFileGraphCommunitySourceIsFreshComputeOnly' ./internal/query/...`: all PASS. `structured` and `tie` sub-tests each log "compared 5 runs" (≥3). The RED-control test logs "seed offset 1 flipped the tie fixture" and "structured fixture invariant under the flipping offset" — i.e. a perturbed seed demonstrably turns the tie sub-test red while the well-separated fixture stays invariant, exactly the discriminating proof GRF-08 requires. `internal/query/community.go` inspected directly: `sort.Strings(paths)` (byte-wise, no case-folding/unicode/norm calls), one package-level `var communityDefaults`, no `sync` import, fixed PCG seed (`rand.NewPCG(opts.seed1, opts.seed2)`), canonical relabeling by smallest member path. |
| 4 | (GRF-10 SC4) `gonum.org/v1/gonum` passes govulncheck, appears by name in the SBOM, and its import closure is verified cgo-free with a reported package count | ✓ VERIFIED | Re-ran `GOTOOLCHAIN=go1.26.6 task -s check:gonum` live (not from SUMMARY): exit 0, output — "govulncheck (source mode, main module) clean — 26 gonum packages in the scanned set"; "SBOM lists 148 packages; gonum.org/v1/gonum present 1 time(s) at v0.17.0; positive control github.com/cockroachdb/pebble/v2 present 1 time(s)"; "cgo closure over gonum.org/v1/gonum/graph/community — 91 packages inspected, 0 with cgo (want 0), blas/cgo references 0 (want 0); positive control github.com/tree-sitter/go-tree-sitter — 3 with cgo (want >= 1)"; final line "check:gonum: PASS". The positive control (tree-sitter genuinely showing 3 cgo packages) proves the scan is not blind. GRF-10's `gonum.org/v1/gonum` package-legitimacy verdict is recorded as `[ASSUMED]` in `11-SECURITY.md` (no npm/pypi/crates legitimacy seam covers a Go module) — reported here as a carried-forward, explicit assumption, not treated as a gap per this task's instructions. |
| 5 | `go.mod` gains exactly one direct-require line for gonum v0.17.0; `go.sum` unchanged | ✓ VERIFIED | `git diff --numstat 698235a2 -- go.mod` = `1 0`; `git diff --quiet 698235a2 -- go.sum` clean; `rg -o '^\tgonum\.org/v1/gonum v0\.17\.0$' go.mod \| wc -l` = 1; `GOTOOLCHAIN=go1.26.6 go list -m gonum.org/v1/gonum` prints `gonum.org/v1/gonum v0.17.0`. |
| 6 | On a PASS verdict, `Node`'s reserved 50-59 field range stays untouched and no fallback code path was introduced | ✓ VERIFIED | `rg -o 'reserved 50 to 59' internal/schema/graph.proto \| wc -l` = 3 (unchanged); `internal/community` package does not exist; `rg -o 'GetCommunityId' internal/query/traverse.go internal/query/community.go` = 0 matches; `AssignCommunities(result.Nodes, result.Edges)` call site present exactly once in `traverse.go` (the fresh-compute path, D-15, still primary). |
| 7 | Wire fields `community_id = 5` / `community_count = 7` exist, are additive, and are field-for-field projected from the engine result over the real listener | ✓ VERIFIED | `internal/uiproto/uiv1/ui.proto`: `int32 community_id = 5;` (FileGraphNode), `int32 community_count = 7;` (FileGraphResponse), both present exactly once; `^  rpc ` count = 16 (unchanged). `internal/uiserver/handlers.go`: `CommunityId: int32(n.CommunityID)`, `CommunityCount: int32(result.CommunityCount)` present. Re-ran `go test -run 'TestFileGraphProjectsEngineResult\|TestUIServiceMethodSetIsExactlyTheReadSet\|TestUIProtoFieldNumbersAreStableAndUnique' ./internal/uiserver/...`: PASS. |
| 8 | No `web/src` file other than the regenerated `web/src/lib/gen/ui_pb.ts` changed before the GRF-09 verdict resolved | ✓ VERIFIED | `git diff --name-only 698235a2 41e118fa -- web/src` returns only `web/src/lib/gen/ui_pb.ts`; the sole intervening `web/src` commit before the verdict (`8da21387`) touched only that generated file. |
| 9 | Build/vet/type-check are clean across the whole repo at HEAD | ✓ VERIFIED | Re-ran `GOTOOLCHAIN=go1.26.6 go build ./...` and `go vet ./...`: both clean. Re-ran `pnpm -C web check` (svelte-check): `COMPLETED 1172 FILES 0 ERRORS 0 WARNINGS 0 FILES_WITH_PROBLEMS`. |
| 10 | Web test suite is green, including the new community tests | ✓ VERIFIED | Re-ran `pnpm -C web test`: `Test Files 49 passed (49)`, `Tests 584 passed (584)`. |
| 11 | The three code-review warnings (WR-01, WR-02, WR-03) raised in iteration 1 were fixed, not merely acknowledged | ✓ VERIFIED | `communityPaletteIndex` in `community-palette.ts` throws on non-integer/`<1` input (WR-01, commit `52955fff`) — read directly, matches. `check-no-force-layout.mjs`'s `walk()` uses `fs.statSync` (follows symlinks) plus a `visitedRealDirs` cycle guard, with a live self-test (`selfTestSymlinkTraversal`) that PASSED when re-run (WR-02, commit `1b61ecee`). The scan's documentation is narrowed to its actual (string-literal / cytoscape-<x>) coverage and reports a non-fatal `unresolvedLayoutNames` advisory (WR-03, commit `00dc535f`) — re-run live, reported `GraphCanvas.svelte:374` as the one advisory (non-blocking) entry, verdict still PASS. |
| 12 | The phase's own mutation-testing discipline (RED-controlled discriminators for all four guard families) executed cleanly and left the tree unmodified | ✓ VERIFIED | `11-MUTATION-LOG.md` documents four families (GRF-08 seed-counter, GRF-09 threshold-widening, GRF-06 planted force-layout literal, GRF-10 cgo-control-neutering), each with a pre-mutation cleanliness gate, the mutation diff, the RED transcript, and a post-revert cleanliness gate. `git status --porcelain` at HEAD is clean, consistent with every mutation having been reverted. |

**Score:** 12/12 truths verified (0 present-but-behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `corpora/graph-cluster-threshold.json` | GRF-09 pass bar, alone, first commit | ✓ VERIFIED | Single commit `698235a2`, all locked values present (max 500, median, 3 runs, minNodes 2, guava pin) |
| `corpora/graph-cluster-observations.json` | Committed measurement with verdict | ✓ VERIFIED | verdict PASS, median 106ms, digest matches, `manifest` storeSource, guava-scale counts (3233/21554/355) |
| `internal/query/community.go` | Deterministic Louvain engine | ✓ VERIFIED | `AssignCommunities`, `assignCommunitiesWith`, `undirectedPairWeights`, `canonicalCommunityLabels`, seam `communityOptions`, single package var |
| `internal/query/traverse.go` | Fresh-per-call wiring into `FileGraph()` | ✓ VERIFIED | `CommunityID`/`CommunityCount` fields, `AssignCommunities(result.Nodes, result.Edges)` call site once |
| `tools/graphcluster/main.go` + tests | GRF-09 measurement harness | ✓ VERIFIED | `run/loadThreshold/thresholdDigest/resolveCorpusStore/measureRuns/medianInt64/judge/writeObservation` present; 7 tests PASS; source scan proves no re-index/write |
| `tools/graphcluster/ancestry_test.go` | Ancestry + digest proof | ✓ VERIFIED | Both tests PASS (not SKIP) at HEAD, full clone |
| `internal/uiproto/uiv1/ui.proto` + gen | Wire fields | ✓ VERIFIED | `community_id=5`, `community_count=7`, 16 rpcs unchanged |
| `web/src/lib/components/graph/community-palette.ts` | 12-hue palette + guarded index fn | ✓ VERIFIED | Throws on invalid id (WR-01 fix present) |
| `web/src/lib/components/graph/file-graph-transform.ts` | communityId passthrough | ✓ VERIFIED | `?? 0` guard, file-nodes-only class attachment |
| `web/src/lib/components/graph/graph-style.ts` | 12 static colour rules | ✓ VERIFIED | `graph-community-{0..11}` rules present |
| `web/src/routes/graph/+page.svelte` | Toolbar community count line | ✓ VERIFIED | Reads `communityCount` verbatim |
| `web/scripts/check-no-force-layout.mjs` + Taskfile target | Force-layout scan | ✓ VERIFIED | Re-run live, PASS, self-test PASS (3 sub-cases) |
| `Taskfile.yml` `check:gonum` | GRF-10 supply-chain gate | ✓ VERIFIED | Re-run live, exit 0, all four required numbers present |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| `FileGraph()` rollup | `AssignCommunities` | direct call after cycle block | ✓ WIRED | `traverse.go:347` |
| `FileGraphResult` | proto `FileGraphResponse`/`FileGraphNode` | `fileGraphToProto`/`fileGraphNodeToProto` | ✓ WIRED | `handlers.go:1037,1051`; `TestFileGraphProjectsEngineResult` PASS |
| Wire `community_id`/`community_count` | `web/src/lib/gen/ui_pb.ts` → `file-graph-transform.ts` → `graph-style.ts` render | generated client → transform → static CSS-like rules | ✓ WIRED | `?? 0` copy, `graph-community-N` class, matching static rule |
| `corpora/graph-cluster-threshold.json` (first commit) | `tools/graphcluster` reads it | `loadThreshold` | ✓ WIRED | Every threshold key exercised by `TestLoadThresholdRefusesMissingBar` |
| threshold commit | every measurement commit | git ancestry | ✓ WIRED | `TestClusterThresholdCommitIsAncestorOfEveryMeasurement` PASS |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| GRF-08 determinism + RED control | `go test -v -run 'TestAssignCommunities...' ./internal/query/...` | all PASS, "compared 5 runs" x2, seed-flip logged | ✓ PASS |
| GRF-09 harness unit tests | `go test -v ./tools/graphcluster/...` | 9/9 PASS incl. ancestry+digest (not skipped) | ✓ PASS |
| GRF-06 force-layout scan | `node web/scripts/check-no-force-layout.mjs [--self-test]` | PASS, 106 files scanned, 0 forbidden | ✓ PASS |
| GRF-10 supply-chain gate | `GOTOOLCHAIN=go1.26.6 task -s check:gonum` | exit 0, PASS, all 4 numbers reported | ✓ PASS |
| Go build/vet | `go build ./... && go vet ./...` | clean | ✓ PASS |
| Svelte type-check | `pnpm -C web check` | 0 errors, 0 warnings | ✓ PASS |
| Web test suite | `pnpm -C web test` | 584/584 passed | ✓ PASS |

### Probe Execution

Not applicable — this phase has no `scripts/*/tests/probe-*.sh` and none are declared in its plans; `tools/graphcluster` and `check-no-force-layout.mjs`/`check:gonum` (covered above under Behavioral Spot-Checks) are this phase's equivalent runnable verification instruments and were executed directly.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| GRF-06 | 11-04 | Colour by community on unchanged layered layout, no force-directed | ✓ SATISFIED | Truth 2, mechanism fully verified; visual confirmation deferred to Human Verification |
| GRF-08 | 11-01 | Deterministic assignment, ≥3-run proof, RED control | ✓ SATISFIED | Truth 3 |
| GRF-09 | 11-01/11-02 | Threshold committed first, measured, verdict preserved, fallback documented | ✓ SATISFIED | Truth 1, 6 |
| GRF-10 | 11-03 | govulncheck clean, SBOM presence, cgo-free closure | ✓ SATISFIED | Truth 4 |

No orphaned requirements — `.planning/REQUIREMENTS.md`'s Phase 11 row lists exactly GRF-06, GRF-08, GRF-09, GRF-10, and all four are claimed across plans 11-01 through 11-04. (REQUIREMENTS.md's checkbox/tracking-table rows for these four remain unchecked/"Pending" at time of this verification — that is expected: those rows are updated by the orchestrator after verification passes, per Phase 10's precedent (HLT-04..06 flipped to `[x]`/"Complete" only in the phase-close commit), not by the phase's own plans.)

### Anti-Patterns Found

None. Scanned all phase-changed source/test/config files for `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER`/empty-implementation patterns — zero hits (the one `XXXXXX` match in `Taskfile.yml` is a pre-existing, unrelated `mktemp` template pattern, not a debt marker).

### Ordering Invariants (explicitly re-derived from git, per this task's instructions)

- `git log --format=%H -- corpora/graph-cluster-threshold.json | wc -l` = **1** (commit `698235a2`, sole file in that commit — confirmed via `git show --name-only --format=`).
- Every commit touching `corpora/graph-cluster-observations.json` and `tools/graphcluster/` (`38b96025`, `41e118fa`, `3eb6978e`, `e9b28e88`) descends from `698235a2` (`git merge-base --is-ancestor` true for all four).
- No hand-authored `web/src` change (other than `web/src/lib/gen/**`) precedes the observation commit `41e118fa`: `git log --oneline 698235a2..41e118fa -- web/src` shows exactly one commit (`8da21387`), and `git diff --name-only 698235a2 41e118fa -- web/src` shows only `web/src/lib/gen/ui_pb.ts`.

### GRF-10 `[ASSUMED]` Note (reported, not a gap)

Per this task's explicit instruction: `gonum.org/v1/gonum`'s package-legitimacy verdict is recorded verbatim in `11-SECURITY.md` as `[ASSUMED]` — no npm/pypi/crates legitimacy seam covers a Go module, so no machine verdict was possible in this autonomous run. The govulncheck/SBOM/cgo-closure evidence (Truth 4) is machine-verified; only the qualitative "is this a legitimate, non-malicious project" judgment is an unresolved, explicitly-flagged assumption, carried forward rather than silently treated as resolved.

### Human Verification Required

1. **Visual confirmation of community colouring on the live graph view** — RESOLVED 2026-09-13 (see `resolution` under `human_verification` in the frontmatter): performed live against `codegraph ui` on this repository and accepted by the user. 132 file nodes across three expanded directories rendered with 8 distinct community colours and 0 class/id inconsistencies, 0 coloured directory compounds, the "Files fall into 205 communities" toolbar line, cycle borders composing over community fills, and a byte-identical ELK layout. The four console entries observed are reproduced identically by a pre-phase build (18ef2434) and are not Phase 11 regressions.

### Gaps Summary

No gaps. All four roadmap success criteria (GRF-06, GRF-08, GRF-09, GRF-10) are backed by re-executed, passing automated evidence: threshold-first git ordering proven directly from history, a real GRF-09 measurement committed with verdict PASS (106ms median vs 500ms max), GRF-08's determinism proven with a working RED control, and GRF-10's supply-chain gate re-run live with all three positive-controlled halves green. The three code-review warnings from iteration 1 were fixed and re-confirmed in code. The phase's own mutation-testing discipline (four RED families) executed and reverted cleanly, leaving `git status` clean. The one visual human-check the phase's own plan deferred to end-of-phase review was performed live and accepted (see Human Verification Required above); nothing remains open.

---

_Verified: 2026-09-13T19:20:00Z_
_Verifier: Claude (gsd-verifier)_
