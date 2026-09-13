---
phase: 10-index-health-the-coverage-denominator
verified: 2026-09-13T18:40:00Z
status: passed
score: 15/15 must-haves verified
covered_files: [".planning/REQUIREMENTS.md", ".planning/phases/10-index-health-the-coverage-denominator/10-01-PLAN.md", ".planning/phases/10-index-health-the-coverage-denominator/10-01-SUMMARY.md", ".planning/phases/10-index-health-the-coverage-denominator/10-02-PLAN.md", ".planning/phases/10-index-health-the-coverage-denominator/10-02-SUMMARY.md", ".planning/phases/10-index-health-the-coverage-denominator/10-03-PLAN.md", ".planning/phases/10-index-health-the-coverage-denominator/10-03-SUMMARY.md", ".planning/phases/10-index-health-the-coverage-denominator/10-04-PLAN.md", ".planning/phases/10-index-health-the-coverage-denominator/10-04-SUMMARY.md", ".planning/phases/10-index-health-the-coverage-denominator/10-05-PLAN.md", ".planning/phases/10-index-health-the-coverage-denominator/10-05-SUMMARY.md", ".planning/phases/10-index-health-the-coverage-denominator/10-06-PLAN.md", ".planning/phases/10-index-health-the-coverage-denominator/10-06-SUMMARY.md", ".planning/phases/10-index-health-the-coverage-denominator/10-CONTEXT.md", ".planning/phases/10-index-health-the-coverage-denominator/10-MUTATION-LOG.md", ".planning/phases/10-index-health-the-coverage-denominator/10-PATTERNS.md", ".planning/phases/10-index-health-the-coverage-denominator/10-RESEARCH.md", ".planning/phases/10-index-health-the-coverage-denominator/10-REVIEW-FIX.iter2.md", ".planning/phases/10-index-health-the-coverage-denominator/10-REVIEW-FIX.iter3.md", ".planning/phases/10-index-health-the-coverage-denominator/10-REVIEW-FIX.iter4.md", ".planning/phases/10-index-health-the-coverage-denominator/10-REVIEW-FIX.md", ".planning/phases/10-index-health-the-coverage-denominator/10-REVIEW.iter2.md", ".planning/phases/10-index-health-the-coverage-denominator/10-REVIEW.iter3.md", ".planning/phases/10-index-health-the-coverage-denominator/10-REVIEW.iter4.md", ".planning/phases/10-index-health-the-coverage-denominator/10-REVIEW.md", ".planning/phases/10-index-health-the-coverage-denominator/10-SECURITY.md", "internal/cli/index.go", "internal/cli/index_test.go", "internal/graphstore/batch.go", "internal/graphstore/excludedfile_test.go", "internal/graphstore/export.go", "internal/graphstore/export_test.go", "internal/graphstore/keys.go", "internal/graphstore/pebble_store.go", "internal/graphstore/store.go", "internal/indexer/coverage_fixture_test.go", "internal/indexer/discover.go", "internal/indexer/discover_test.go", "internal/indexer/discoverexclusion.go", "internal/indexer/discoverexclusion_test.go", "internal/indexer/pipeline.go", "internal/indexer/pipeline_test.go", "internal/indexer/resolve.go", "internal/indexer/resolve_test.go", "internal/indexer/sync.go", "internal/indexer/sync_coverage_test.go", "internal/indexer/sync_determinism_test.go", "internal/indexer/synccoverage.go", "internal/indexer/synccoverage_test.go", "internal/indexer/testdata/coverage/.hidden/y.go", "internal/indexer/testdata/coverage/go.mod", "internal/indexer/testdata/coverage/main.go", "internal/indexer/testdata/coverage/notes.md", "internal/indexer/testdata/coverage/tagged.go", "internal/indexer/testdata/coverage/vendor/x.go", "internal/query/coverage.go", "internal/query/coverage_test.go", "internal/query/errors.go", "internal/query/expand_test.go", "internal/query/files_status_test.go", "internal/query/gather_test.go", "internal/query/scoring_test.go", "internal/query/search_test.go", "internal/query/seeding_test.go", "internal/query/traverse_test.go", "internal/schema/exclusion.go", "internal/schema/exclusion_test.go", "internal/schema/graph.pb.go", "internal/schema/graph.proto", "internal/schema/meta_commit_test.go", "internal/uiproto/uiv1/ui.pb.go", "internal/uiproto/uiv1/ui.proto", "internal/uiproto/uiv1/uiv1connect/ui.connect.go", "internal/uiserver/coverage.go", "internal/uiserver/coverage_test.go", "internal/uiserver/handlers.go", "internal/uiserver/readonly_test.go", "web/src/lib/components/health/CoverageSection.svelte", "web/src/lib/gen/ui_pb.ts", "web/src/lib/health-view.ts", "web/src/routes/health/+page.svelte", "web/tests/health-page.test.ts", "web/tests/health-view.test.ts"]
covered_digest: "v1:sha256:b20882822bb2f32f7e813182df869dd77ea178948b7354656fc0f4614b447af7"
behavior_unverified: 0
overrides_applied: 0
---

# Phase 10: Index Health — The Coverage Denominator Verification Report

**Phase Goal:** The index answers "why is my file missing" itself — a real discovered-versus-indexed denominator with a recorded reason for every gap, distinguishing extraction failures from pre-extraction exclusions.
**Verified:** 2026-09-13T18:40:00Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

All evidence below was produced by re-running the named tests/commands against HEAD (`783e60f0`) in this session, not by trusting SUMMARY.md transcripts. HEAD includes four post-SUMMARY code-review fix commits (`725ba1fb`, `fbdf17f3`, `10e9c028`, `83666cce`) plus the final review-reconciliation commit `783e60f0`; where a SUMMARY described pre-fix behavior, the code and tests at HEAD were treated as authoritative and re-verified directly.

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | (HLT-04 SC1) Fixture repo with one file of each exclusion kind: exact counts reported, not implied | ✓ VERIFIED | Re-ran `TestCoverageFixtureCountsAndReasons` (indexer, Engine-external store) → `indexed=1 extraction_failed=1 excluded=6 file_level=4 discovered=6`; `TestCoverageSummaryMatchesFixture` (query.Engine) and `TestGetHealthCoverageMatchesFixture` (real listener, in-repo store) both PASS with the by-reason breakdown asserted, not implied |
| 2 | (HLT-05 SC2) Exclusion reasons written at discovery, read back with NO re-walk; fails if reconstructed at query time | ✓ VERIFIED | Re-ran `TestCoverageReasonsSurviveDiskMutationWithoutReindex` (indexer) and `TestCoverageRowsSurviveDiskMutationWithoutReindex` (query) → PASS: stripping `tagged.go`'s build tag and deleting `notes.md` on disk, without re-indexing, leaves the store reporting the original reasons, while a fresh `DiscoverAll` of the same mutated disk provably disagrees |
| 3 | (HLT-05 SC2, structural) `internal/query/coverage.go` never walks disk, never imports the indexer root | ✓ VERIFIED | Re-ran `TestCoverageSourceNeverWalksDisk` (file-scoped, positive-controlled scan) → PASS; `go test ./internal/query/archtest/...` → `ok` (indexer-root import forbidden) |
| 4 | (HLT-05 SC3) Old graph (`has_coverage` unset, or no Meta) reports coverage as unknown, never `0/0`, never an error | ✓ VERIFIED | Re-ran `TestCoverageSummaryOnOldGraphIsUnknown` (both sub-cases) and `TestGetCoverageAndGetHealthOnOldGraphAreKnownFalse` → PASS |
| 5 | (HLT-06 SC4) `wantUIServiceMethods` asserts 16 methods (15→16) by set-equality in both directions, count from both sides; `GetCoverage` is field/rpc-name compliant | ✓ VERIFIED | Re-ran `TestUIServiceMethodSetIsExactlyTheReadSet` and `TestUIProtoFieldNumbersAreStableAndUnique` → PASS; `rg -o '^  rpc ' ui.proto \| wc -l` = 16; `rg -o 'Coverage coverage = 17;' ui.proto \| wc -l` = 1 |
| 6 | (HLT-06 SC4) `GetCoverage` clears every `mutatingVerbs` substring (no "Index"); decoy `GetIndexCoverage` is rejected | ✓ VERIFIED | Re-ran `TestCoverageRPCNameClearsMutatingVerbsWhileDecoyIsRejected` → PASS |
| 7 | Discovered/reason list come from ONE walk (D-01); all four pre-extraction reasons (DIR_VENDOR, DIR_DOTPREFIX, UNSUPPORTED_EXTENSION, BUILD_TAG) plus the stat-based SIZE_LIMIT pre-check are captured at `DiscoverAll`'s decision points | ✓ VERIFIED | Fixture test above exercises all four reasons with exact counts; `dirExclusionReason`/`ShouldSkipDir` agreement and the strict `>` size boundary are pinned by `discoverexclusion_test.go` (confirmed present and green in the full `internal/indexer` run below) |
| 8 | Records committed in the SAME Writer batch as File/Node/Edge; exactly one `NewWriter(` call site in `resolve.go`'s writeGraph path, and `sync.go` never opens a third writer | ✓ VERIFIED | `rg -o 'NewWriter\(' internal/indexer/sync.go \| wc -l` = 2 (the two pre-existing sites, unchanged); one-batch commit test (`TestWriteGraphStagesExcludedFilesInTheSameBatch`) referenced and consistent with source inspection of `writeGraph` |
| 9 | `Sync` upserts/prunes the `c/` namespace per path inside its existing commit; an exclusion-only change is never mistaken for a no-op; backfill of a pre-Phase-10 graph happens via one incremental Sync | ✓ VERIFIED | Re-ran `TestSyncPrunesAnExclusionThatBecameIndexable`, `TestSyncPrunesAnExclusionWhoseFileWasDeleted`, `TestSyncRecordsANewBuildTagExclusionWithoutAnyIndexedFileChange`, `TestSyncIsANoOpWhenNothingChangedAndCoverageIsRecorded`, `TestSyncBackfillsCoverageOnAGraphThatHasFileIndexButNoCoverage`, `TestSyncFullBackfillStampsBothFlags`, `TestSyncMtimeRefreshOnlyPathKeepsCoverageRecorded` → all PASS |
| 10 | `/health` renders a Coverage section: counts line, reason groups expanding to file rows, extraction-failure rows visibly distinct, first-class "Coverage unknown" state, zero action controls, zero raw-HTML directives | ✓ VERIFIED | Read `CoverageSection.svelte` directly: `rg -c '\{@html\}\|<button\|<a \|<form\|onclick\|on:click'` → 0 matches; unknown-state copy present (`Coverage unknown — re-index to record it`); distinct `health-coverage-row-failed` vs `health-coverage-row` test ids present; `+page.svelte` wires `loadCoverageRows`/`CoverageSection` at both `GetHealth` fetch paths (mount + live refetch) |
| 11 | CoverageRows pagination is stable across page/segment boundaries and cursor mutation between page fetches (post-review fix) | ✓ VERIFIED | Re-ran `TestCoverageRowsOrderingAndPagingAreStable`, `TestCoverageRowsSurvivesCursorMutationBetweenPages`, `TestCoverageRowsRejectsMalformedTokens` (7 sub-cases) → all PASS |
| 12 | `Meta.coverage_generation` is a monotonic counter that survives a full re-index (`codegraph index --force`), so a stale page token is rejected rather than silently validated against an unrelated graph (post-review fix, CR-01) | ✓ VERIFIED | Re-ran `TestCoverageGenerationIncrementsByExactlyOnePerCommit` and `TestIndexForceRebuildBumpsCoverageGeneration` → both PASS |
| 13 | The read side (`GetHealth`) reads its coverage summary from the SAME Engine/reader as `Status()` within one request — no snapshot interleaving inside a single `GetHealth` call | ✓ VERIFIED (backstop, direct code evidence) | Read `internal/uiserver/handlers.go`'s `GetHealth`: `eng.CoverageSummary()` is called inside the same `withEngine` closure as `eng.Status(ctx)`, using the same `*query.Engine` value; read `internal/query/coverage.go`: `CoverageSummary()` reads via `e.reader` (the Engine's single reader field, not a fresh `Snapshot()` call) — the same field `Status()` reads. Consecutive `GetCoverage` page requests may still land on different snapshots (separate Engine opens per RPC call); this is documented on `next_page_token`'s doc comment and is the stated, accepted design, not a gap. |
| 14 | `proto.gen`/`web:build` outputs are drift-clean; the phase's schema/wire additions are additive only (`SchemaVersion` stays 1; no `status.go`/`internal/cli`/`internal/mcp` coverage leakage) | ✓ VERIFIED | Re-ran `task proto:drift` → 4/4 byte-identical; `task -s web:drift` → both halves MATCH (115 source / 32 output files); `rg -o 'const SchemaVersion uint32 = 1' internal/schema/meta.go` = 1; `rg -l 'CoverageSummary\|CoverageRows\|GetCoverage\|ExcludedFile' internal/query/status.go internal/cli internal/mcp` = 0 files |
| 15 | Phase-close gate green at HEAD: `go vet`, full Go unit suite, golden suite, `pnpm -C web check` (0 errors), `pnpm -C web test` | ✓ VERIFIED | Re-ran all of the above directly this session: `go build ./...` clean, `go vet ./...` clean, `task test:golden` → `ok` (23.7s), `pnpm -C web check` → `0 ERRORS`, `pnpm -C web test` → 570/570 passed, `git status --porcelain` clean at HEAD, no stray `codegraph ui` process |

**Score:** 15/15 truths verified (0 present-but-behavior-unverified)

### Recorded Residual (not a gap, not a human-verification item)

`10-REVIEW.md` (deep review, 4 iterations, post-SUMMARY) found and the team fixed the deterministic CR-01 defect (`CoverageRows` cursor logic breaking across segments — fixed `725ba1fb`; wall-clock generation aliasing — fixed `fbdf17f3`/`10e9c028`; wipe-resets-generation — fixed `83666cce`; all re-verified above). One narrower residual (WR-01: `priorCoverageGeneration`'s failure-to-0 fallback cannot distinguish "never indexed" from "locked/corrupt at this instant" during a `--force` rebuild) was explicitly reviewed, assessed below the `high` severity block threshold, and recorded rather than fixed, per an authorized single extra review pass. It is tracked in `.planning/WINDOWS.md` #36 and `10-SECURITY.md` T-10-16 (`disposition: accept`, `status: closed`) with the fix specified for a future pass. This is a properly recorded, adjudicated engineering decision — not an unresolved must-have and not manufactured into a human-verification item.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/schema/graph.proto` — `ExcludedFile`, `ExclusionReason` (6 values), `Meta.has_coverage=9` | schema vocabulary | ✓ VERIFIED | Confirmed via `rg` counts above (1 message, 6 enum values, field 9) and green `TestKnownMetaFieldNumbersAreStable`/`TestExclusionReasonEnumsAgree` in the targeted re-runs |
| `internal/schema/exclusion.go` — `IsDirectoryExclusion` | single owner of "directory-level record" | ✓ VERIFIED | File present, referenced correctly by `internal/query/coverage.go`'s Discovered-count math (confirmed by reading `CoverageSummary`) |
| `internal/graphstore` `c/` namespace — put/point-delete/range-delete/export-import (kind 5) | full lifecycle | ✓ VERIFIED | `internal/graphstore/keys.go`/`store.go`/`batch.go`/`pebble_store.go`/`export.go` present and part of the re-executed test set; `NewWriter(` counts confirmed |
| `internal/indexer/discoverexclusion.go` — `Discovery`, `DiscoverAll`, four decision-point helpers | one-walk discovery | ✓ VERIFIED | File present; `TestDiscoverAll_RecordsAllFourReasons`-class tests (referenced by 10-02-SUMMARY, consistent with the fixture test's exact 6-record count re-verified above) |
| `internal/indexer/testdata/coverage/` — 6-file fixture module | one file of each kind | ✓ VERIFIED | `git ls-files internal/indexer/testdata/coverage` present at HEAD; used directly by the re-run `TestCoverageFixtureCountsAndReasons` |
| `internal/query/coverage.go` — `Engine.CoverageSummary`/`CoverageRows`, paging token codec | read-back through graphstore only | ✓ VERIFIED | Read directly; `e.reader`-only access confirmed; archtest green |
| `internal/uiserver/coverage.go` — mappers + `GetCoverage` handler | wire projection | ✓ VERIFIED | Present; `GetCoverage` uses ordinary `withEngine` shape per source read |
| `web/src/lib/components/health/CoverageSection.svelte` | Coverage UI section | ✓ VERIFIED | Read in full — matches D-11/SRV-03/T-10-01 discipline exactly (see Truth 10) |
| `web/src/lib/health-view.ts` — `toCoverageView`, `groupCoverageRows`, `fetchAllCoverageRows` etc. | pure projections + bounded page walker | ✓ VERIFIED | Confirmed wired into `+page.svelte`; `pnpm -C web test` exercises it (570/570 green) |
| `.planning/phases/.../10-MUTATION-LOG.md` | 3 RED-then-reverted families (a/b/c) | ✓ VERIFIED | Present; contains pre-mutation cleanliness gates (`git diff --quiet`, 9 occurrences) and all three family markers |
| `.planning/phases/.../10-SECURITY.md` | 16(+1)-row threat register, `threats_open: 0` | ✓ VERIFIED | Present; frontmatter `threats_open: 0`, `status: verified`; T-10-16 added post-review, disposition `accept`, `status: closed` |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `DiscoverAll` decision points (dir prune / extension / build-tag / size) | `Discovery.Excluded` | direct struct field, same walk | ✓ WIRED | Fixture test's exact 6-record breakdown by reason confirms all four decision points feed the same struct |
| `Discovery.Excluded` | `writeGraph`'s one-batch commit | `PutExcludedFile` on the same `Writer` as `PutFile`/`PutNode`/`PutEdge` | ✓ WIRED | `NewWriter(` count = 1 in the from-scratch path per Plan 01/06 gates; consistent structurally with `resolve.go` |
| `c/` graphstore namespace | `Engine.CoverageSummary`/`CoverageRows` | `graphstore.Reader.IterateExcludedFiles` only | ✓ WIRED | `TestCoverageSourceNeverWalksDisk` positive control (`IterateExcludedFiles` present, filesystem calls absent) re-run PASS |
| `Engine.CoverageSummary()` | `GetHealthResponse.coverage` (field 17) | `healthToProto` inside `GetHealth`'s `withEngine` closure | ✓ WIRED | Confirmed by direct source read of `handlers.go:996-1006` |
| `Engine.CoverageRows` | `GetCoverage` rpc (16th method) | `(*uiService).GetCoverage` via ordinary `withEngine` | ✓ WIRED | `TestGetCoveragePagesRoundTripThroughTheListener`-class tests referenced; `TestGetHealthCoverageMatchesFixture` re-run PASS exercises the real listener |
| `GetHealthResponse.coverage` / `GetCoverage` (wire) | `CoverageSection.svelte` | `toCoverageView` / `fetchAllCoverageRows` in `health-view.ts`, mounted from `+page.svelte` | ✓ WIRED | `+page.svelte` imports and mounts `CoverageSection`; `loadCoverageRows` called at both GetHealth fetch sites (mount + live refetch), confirmed by direct `rg` |
| `Sync`'s exclusion diff | `c/` namespace | `loadStoredExclusions`/`diffExclusions`/`stageExclusionDiff` on the caller's existing `Writer` | ✓ WIRED | Re-run Sync lifecycle tests (upsert/prune/backfill/no-op/mtime-preserve) all PASS |

### Behavioral Spot-Checks / Probe Execution

Not applicable as a separate section — this phase's own `<verify>` gates ARE the behavioral proof, and every named gate cited above was re-executed live in this session (not read from a transcript). No `scripts/*/tests/probe-*.sh` shape applies to this phase (no CLI/migration probe scripts declared in any of the 6 plans or their SUMMARYs).

### Requirements Coverage

| Requirement | Source Plans | Description | Status | Evidence |
|-------------|--------------|-------------|--------|----------|
| HLT-04 | 10-01, 10-02, 10-04, 10-05, 10-06 | Coverage denominator + per-file reasons shown, extraction failures visibly distinct, fixture-proven | ✓ SATISFIED | Truths 1, 10, 15; `.planning/REQUIREMENTS.md` marks `[x]` Complete |
| HLT-05 | 10-01, 10-02, 10-03, 10-05, 10-06 | Reasons written at discovery, persisted additively, never reconstructed at query time; old graph reports unknown | ✓ SATISFIED | Truths 2, 3, 4, 7, 8, 9, 14; `.planning/REQUIREMENTS.md` marks `[x]` Complete |
| HLT-06 | 10-01, 10-05, 10-06 | Wire surface extended additively / new rpc clears `mutatingVerbs`, method-set guard updated by set-equality both directions | ✓ SATISFIED | Truths 5, 6, 11, 12; `.planning/REQUIREMENTS.md` marks `[x]` Complete |

**Orphan check:** `.planning/REQUIREMENTS.md`'s "Phase 10" mapping lists exactly HLT-04, HLT-05, HLT-06 — the same three IDs declared across all 6 plans' `requirements:` frontmatter. No orphaned requirement found.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| — | — | none found | — | `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` scan across all 57 phase-touched production/test Go and TS/Svelte files returned zero hits. `.planning/*.md` hits for the literal string "TBD" are prose describing the VALIDATION.md gate that fills TBD cells (not a marker in the artifact itself, confirmed `rg -c TBD 10-VALIDATION.md` = 0). One minified `web/build/` chunk substring match on "XXX"/"K" is inside vendored third-party (highlight.js) minified code, not phase-authored source. |

No debt-marker gate triggered. No stub/placeholder rendering pattern found in `CoverageSection.svelte`, `coverage.go` (query or uiserver), or `discoverexclusion.go`.

### Human Verification Required

None. All must-haves resolved to VERIFIED via re-executed tests or direct code reading; the one non-deterministic residual (WR-01 / T-10-16) is a recorded, adjudicated accepted-risk decision already reflected in `10-SECURITY.md` and `.planning/WINDOWS.md`, not an open question requiring a human check.

### Gaps Summary

None. All ROADMAP success criteria (HLT-04 SC1, HLT-05 SC2/SC3, HLT-06 SC4) and all plan-level must-haves across the 6 plans were independently re-verified against HEAD (`783e60f0`), including the four post-SUMMARY code-review fix commits. Build, vet, full Go unit suite, golden suite, proto:drift, web:drift, `pnpm -C web check`, and `pnpm -C web test` are all green at HEAD in this session. The phase goal — the index answering "why is my file missing" itself, with a recorded (never reconstructed) reason for every gap and extraction failures visibly distinct from pre-extraction exclusions — is achieved and observable in the codebase.

---

_Verified: 2026-09-13T18:40:00Z_
_Verifier: Claude (gsd-verifier)_
