---
phase: 01-corpus-selection-by-measurement
verified: 2026-08-14T00:00:00Z
status: passed
score: 5/5 truths verified
behavior_unverified: 0
overrides_applied: 0
human_validation:
  - truth: "Criterion 4 (FIXT-02): running `task corpora:fetch` twice from clean produces the same tree."
    performed: "2026-08-14 by the orchestrator in a clean scratch CODEGRAPH_CORPUS_DIR (mktemp). Run 1 fetched all 4 locked corpora; root wiped; run 2 re-fetched cold."
    result: "4/4 identical — HEAD and HEAD^{tree} byte-identical across both cold runs (hugo, guava, requests, serilog). Reproducibility confirmed with real network fetches at pinned SHAs. D-12 CODEGRAPH_CORPUS_DIR override also exercised."
  - note: "Verifier-flagged gap (instantiates attribution) fixed during human validation: instantiates=1877 re-attributed to pydantic/pydantic in corpora/selection.json + docs/CORPUS-MEASUREMENT.md; guava's manifest note corrected to its measured 1053 (commits a5ddc57, 0ee8775). Coverage guard re-verified green after both."
behavior_unverified_items:
  - truth: "Criterion 4 (FIXT-02): running `task corpora:fetch` twice from clean produces the same tree."
    test: "From a clean machine/worktree (empty corpus root), run `task corpora:fetch`, delete the corpus root, run it again, and `git rev-parse HEAD^{tree}`-compare the two fetched trees."
    expected: "Both runs produce identical tree contents at the pinned SHAs; the second run's destination passes the four-part integrity check."
    why_human: "The reproducibility invariant is guaranteed structurally (shallow `git fetch --depth 1 origin <sha>` + checkout is deterministic at a pinned SHA, and corpora:assert-one verifies .git present, HEAD==SHA, clean --ignored worktree, and resolvable HEAD^{tree}); `go run ./tools/corpora -mode select` reproduced selection values byte-identically. But the literal network double-fetch over ~150MB of hugo+guava cannot be executed in this verification without fetching from GitHub, and no automated test exercises it."
human_verification:
  - test: "Run `task corpora:fetch` twice from clean to confirm the tree is byte-identical across runs at the pinned SHAs."
    expected: "Both fetches land a tree that passes all four integrity parts (HEAD == pinned SHA, clean --ignored worktree, .git present, HEAD^{tree} resolves); the two runs are identical."
    why_human: "Network-dependent; cannot be exercised in a static/grep verification or by a unit test without fetching the large corpora from GitHub."
  - test: "Confirm no corpus source is committed and corpora land outside the tree."
    expected: "`git ls-files` contains no hugo/guava/requests/serilog checkout; `corpora/` holds only manifest.json, observations.json, selection.json."
    why_human: "Git-derived content-sensitive check; verified here via `git ls-files`/`git status` clean, flagged for human confirmation that the intended D-12 out-of-tree posture holds."
---

# Phase 1: Corpus Selection by Measurement — Verification Report

**Phase Goal:** The project knows, from recorded measurement rather than assumption, exactly which third-party repositories at which commits its golden suite will exercise — and can fetch that tree reproducibly without vendoring it.
**Verified:** 2026-08-14
**Status:** human_needed
**Requirements:** FIXT-01, FIXT-02
**Re-verification:** No — initial verification

## Summary

The phase goal is **achieved** in the codebase. All five roadmap success criteria hold; four are verified from working evidence and one (FIXT-02's "run twice from clean") is present and structurally guaranteed but is a network-dependent reproducibility invariant no automated test exercises, so it routes to human verification. One non-blocking documentation-accuracy defect was found in the committed measurement record's threshold rationale prose.

## Goal Achievement

### Observable Truths (Roadmap Success Criteria)

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| SC1 | A committed measurement record names every candidate indexed, reports real per-kind edge + per-language file counts, and names the locked set (FIXT-01) | ✓ VERIFIED | `corpora/observations.json` names all 8 measured candidates (hugo, nest, guava, Newtonsoft.Json, requests, pydantic, serilog, fastapi) with full scanned `edgesByKind` and `filesByLanguage`. `corpora/selection.json` names the locked set (4 entries). `docs/CORPUS-MEASUREMENT.md` (generated from the two) names every candidate, every rejected candidate, the locked set, and both per-kind/per-language tables. All counts are observed outputs, not estimates. |
| SC2 | Across the locked set, each of the 9 `RankEdges` kinds has non-zero measured count from a named repo, and each of the 5 priority-4 languages has non-zero file count (FIXT-01) | ✓ VERIFIED | Locked maxes (computed from observations.json, confirmed by `TestCorpusCoverageClaim` passing): calls 58812/guava, implements 1018/hugo, imports 2160/hugo, extends 2531/guava, overrides 4658/guava, references 13063/guava, instantiates 1053/guava, returns 5591/guava, type_of 11857/guava — all non-zero. Languages: go 889/hugo, java 3229/guava, csharp 216/serilog, python 37/requests, tsjs 27 (hugo 25 js + guava 2 js) — all non-zero. |
| SC3 | `overrides` and `type_of` each covered by a named repo, cited by measured count, not expectation (FIXT-01) | ✓ VERIFIED | guava `overrides`=4658, `type_of`=11857 in observations.json (lines 137, 140). Thresholds are their halves: 2329, 5928. No synthetic path — `selection.json` `syntheticKinds` is empty and `CheckCoverage` step 8 rejects any. These are exactly the two kinds codegraph-go's own Go source produces zero of. |
| SC4 | One Taskfile target fetches the locked corpora at pinned SHAs; twice-from-clean reproduces the same tree; no corpus source committed (FIXT-02) | ⚠️ PRESENT_BEHAVIOR_UNVERIFIED | `task corpora:fetch` (Taskfile.yml:3543) → `corpora:fetch-one` shallow `git init`+`git fetch --depth 1 origin <sha>`+checkout (D-11). Pinned SHAs live only in `corpora/manifest.json` (sole pin authority, D-09). No corpus committed: `git ls-files` clean, no `corpus/{hugo,guava,requests,serilog}` anywhere. Reproducibility guaranteed by construction (deterministic SHA-pin git fetch + four-part content-sensitive integrity check + idempotent fast-path skip); `go run ./tools/corpora -mode select` reproduced selection values byte-identically. The literal network double-fetch is not executed here (no automated test; network-dependent) — see Human Verification. |
| SC5 | CI restores the corpora from cache; a cache miss falls through to a real fetch at the pinned SHAs, never a skip (FIXT-02) | ✓ VERIFIED | `.github/workflows/corpora.yml` restores via `namespacelabs/nscloud-cache-action` (path-based, SHA-bearing destination subdirectory), then runs `task corpora:fetch` **unconditionally** — not gated on any cache-hit output — and `task corpora:assert`. The workflow comments are explicit (lines 56-62, 156-168) that a miss must fall through to a real fetch and a later reader must not reintroduce a condition. Fetch is idempotent; the only skip is a corpus already passing all four integrity parts. |

**Score:** 4/5 truths verified (1 present, behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
| --- | --- | --- | --- |
| `corpora/manifest.json` | Sole pin authority, per-entry repo/SHA/license/language/locked | ✓ VERIFIED | 4 locked (hugo, guava, serilog, requests), 5 unlocked/rejected; every locked entry Apache-2.0. |
| `corpora/observations.json` | Measured per-kind/per-language counts per candidate | ✓ VERIFIED | 8 measured entries with full scanned status; representative real outputs (guava 58812 calls, 4658 overrides, 11857 type_of). |
| `corpora/selection.json` | Frozen thresholds, rationale, locked set, rejected ledger, empty synthetic | ✓ VERIFIED | Thresholds reproduced byte-identically via `-mode select`. |
| `docs/CORPUS-MEASUREMENT.md` | Generated human-readable record | ✓ VERIFIED | Names every candidate + rejected + locked set + both coverage tables. |
| `internal/corpora/coverage.go` | `CheckCoverage` deriving claim, no stored summary | ✓ VERIFIED | 8-step derivation from m/obs/sel; rejects below-threshold, non-locked supplier, missing observation, rejected-candidate supplier, synthetic, empty locked set. |
| `internal/corpora/record.go` | Typed docs, `ComputeThresholds`, `SelectLockedSet`, `PriorityLanguages` | ✓ VERIFIED | Deterministic rule `min(max(2, best/2), best)` over full eligible universe; brute-force min-cardinality selection. |
| `tools/corpora/main.go` | `-mode root/entries/measure/select/kinds` | ✓ VERIFIED | All modes present; select reproduces committed values; measure upserts with no write path to selection.json. |
| `internal/query/status.go` | `edgesByKind` + un-suppressed `filesByLanguage` | ✓ VERIFIED | Full edge scan; `FilesByLanguage` no longer `json:"-"`. |
| `internal/query/render_status.go` | `DenseEdgesByKind` derived from RankEdges | ✓ VERIFIED | Line 123; dense key set = RankEdges members (tested for equality both directions). |
| `internal/cli/status.go` | `--all-kinds` flag, CLI-only (D-04/D-05) | ✓ VERIFIED | Sparse default, dense behind opt-in; MCP stays sparse. |
| `.github/workflows/corpora.yml` | CI fetch/assert/drift; cache miss falls through | ✓ VERIFIED | Unconditional fetch + four-part assert; path-filtered push+PR. |

### Key Link Verification (Wiring)

| From | To | Via | Status | Details |
| --- | --- | --- | --- | --- |
| `corpora:fetch` | `corpora/manifest.json` | `tools/corpora -mode entries` | ✓ WIRED | Fetch reads locked entries only from the sole pin authority (D-09). |
| `corpora:measure` | observations.json | `measureOne` (in-process indexer + `Engine.Status` + `DenseEdgesByKind` + `StripVolatile`) | ✓ WIRED | Same instrument `codegraph status --json --all-kinds` uses; no bespoke parse path (D-06). |
| `corpora:drift` | committed observations.json/`CORPUS-MEASUREMENT.md` | re-measure + `git diff` clean | ✓ WIRED | Path-filtered; only generated files in the diff. |
| `corpora:verify` (cheap half) | three committed docs | `go test -count=1 ./internal/corpora/...` → `CheckCoverage` | ✓ WIRED | Wired into ci.yml on every run (line 107); `-count=1` defeats cache. |
| `CheckCoverage` | derived supplier | locked observations only (step 4) | ✓ WIRED | Non-locked/rejected candidate observation structurally excluded. |
| `corpora:assert-one` | 4-part integrity | .git present + HEAD==sha + clean --ignored + `HEAD^{tree}` resolves | ✓ WIRED | Content-sensitive, deliberately stronger than HEAD-only. |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| --- | --- | --- | --- | --- |
| observations.json `edgesByKind` | per-kind counts | `Engine.Status` full edge iteration | Yes — scanned from index | ✓ FLOWING |
| observations.json `filesByLanguage` | per-language counts | `Engine.Status` file iteration | Yes — scanned from index | ✓ FLOWING |
| selection.json `minEdgesPerKind` | thresholds | `ComputeThresholds` over observations | Yes — reproduced byte-identically | ✓ FLOWING |
| selection.json `lockedSet` | locked identities | `SelectLockedSet` brute-force enumeration | Yes — reproduced byte-identically | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| --- | --- | --- | --- |
| Thresholds locked set reproduce | `go run ./tools/corpora -mode select` | Byte-identical minEdgesPerKind + lockedSet | ✓ PASS |
| Build clean | `go build ./...` | Exit 0 | ✓ PASS |
| Coverage guard passes committed docs | `go test -count=1 ./internal/corpora/...` | ok | ✓ PASS |
| Plan packages pass | `go test -count=1 ./internal/query ./internal/cli ./tools/corpora` | all ok | ✓ PASS |
| Wire-oracle re-freeze (call-status + resources-read-status) | `go test -count=1 ./test/wireoracle/...` | ok | ✓ PASS |
| Double-fetch-from-clean reproducibility | `task corpora:fetch` twice | Not executed (network) | ? SKIP → Human Verification |

### Probe Execution

No `scripts/*/tests/probe-*.sh` probes are declared by any PLAN or SUMMARY in this phase. The phase's executable checks are Go tests and Taskfile targets, all exercised above. SKIPPED (no conventional probes exist).

### Requirements Coverage

| Requirement | Description | Status | Evidence |
| --- | --- | --- | --- |
| FIXT-01 | Recorded measurement names every candidate and the locked set, with the 9 kinds and 5 languages covered by named repos (incl. overrides/type_of), no synthetic | ✓ SATISFIED | observations.json + selection.json + docs/CORPUS-MEASUREMENT.md; 9 kinds and 5 languages all non-zero from named locked repos; overrides/type_of from guava measured counts; syntheticKinds empty; `CheckCoverage` + `select` reproduce. |
| FIXT-02 | One Taskfile target fetches locked corpora at pinned SHAs; twice-from-clean reproduces; no corpus committed; CI cache miss falls through to real fetch | ✓ SATISFIED (SC4 reproducibility behavior pending human) | `corpora:fetch`/`corpora:fetch-one` at pinned SHAs; git clean of corpus source; corpora.yml unconditional fetch after cache restore. |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| --- | --- | --- | --- | --- |
| `corpora/selection.json` (and generated `docs/CORPUS-MEASUREMENT.md` line 10) | `instantiates=1877 (google/guava)` in `thresholdRationale` | Prose attribution error | ⚠️ WARNING | The rationale claims guava supplied the 1877 `instantiates` best value, but guava's measured `instantiates` is 1053 (observations.json line 135); the 1877 is pydantic's (line 149). The threshold VALUE (938) is correctly derived and the doc's own Per-Kind table correctly shows guava measuring 1053. Incorrect only in the rationale prose, which contradicts the same document's table. No success criterion is affected. |

No TODO/FIXME/TBD/XXX debt markers in any modified/created corpus, tool, status, workflow, or docs file (the single "TODO" match in `internal/query/render_status.go:194` is a comment explicitly stating it is NOT a TODO).

### Gap Assessment

**Blocking gaps:** None.

**Warning (non-blocking, documentation accuracy):** `selection.json` `thresholdRationale` and `docs/CORPUS-MEASUREMENT.md` line 10 attribute the measured `instantiates=1877` to google/guava, but that is pydantic's count (guava is 1053). The threshold value 938 and the derived supplier table are correct. Recommend correcting the prose attribution in `selection.json` and regenerating the doc.

**Human verification needed:**
1. **FIXT-02 twice-from-clean reproducibility** — run `task corpora:fetch` twice from a clean corpus root (needs network; ~150MB hugo+guava) and confirm the tree is byte-identical at the pinned SHAs. Structurally guaranteed but not executable here.
2. **D-12 out-of-tree posture** — confirm corpora land outside the git working tree and nothing corpus-source is tracked.

### Gaps Summary

The phase goal is achieved: the corpus set is locked on recorded measurements, the 9-kind vocabulary and 5 priority languages are covered by named third-party repositories including the two at-risk kinds (overrides/type_of) measured on google/guava with no synthetic path, the fetch target is reproducible by construction, and CI falls through to a real fetch on cache miss. One non-blocking prose attribution error in the threshold rationale should be corrected. The single pending item is human confirmation of the network-dependent twice-from-clean reproducibility behavior, which no automated test exercises.

---

_Verified: 2026-08-14_
_Verifier: Claude (gsd-verifier)_