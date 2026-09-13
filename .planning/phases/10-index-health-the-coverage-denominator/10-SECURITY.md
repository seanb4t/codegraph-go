---
phase: "10"
slug: "index-health-the-coverage-denominator"
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: "2026-09-12"
---

# Phase 10 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

**Register origin:** `register_authored_at_plan_time: true` — all five phase plans (10-01
through 10-05) carry a `<threat_model>` block. The register below is the union of every
plan's rows, deduplicated by id; where two plans reused the same id, the mitigation cell
merges both plans' text (noted inline by `(10-0N)` tags). Verification depth is ASVS L1
(grep/execution-level mitigation presence), which the workflow's short-circuit rule declares
sufficient for `threats_open: 0` at L1.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|----------------|
| repository filesystem → `DiscoverAll` → `c/` graphstore namespace | walker-produced path/detail strings from occasionally adversarial third-party trees are persisted verbatim, in the same one-batch commit as `File`/`Node`/`Edge` records | file/directory path strings, byte sizes, GOOS/GOARCH pairs |
| `c/` namespace → `internal/query` → `GetHealth`/`GetCoverage` → browser | persisted strings and counts cross to the loopback SPA; `page_size`/`page_token`/`reason` cross back in from the client on `GetCoverage` | persisted path/detail strings, counts, an opaque page cursor |
| ARCH-01 export stream → `Import` | externally-produced framed records now include kind 5 (`ExcludedFile`) | serialized `ExcludedFile` protobuf records |
| `CoverageSection.svelte` → user | the health page's Coverage section renders the above strings/counts; the remedy for any gap is text only | rendered path/detail/reason text; zero server-mutating controls |

**The write path's new trust surface is a walker-produced string persisted then rendered — the same escaping discipline as `File.path`, enforced at the render boundary, not at write time.**

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-10-01 | Tampering (XSS) | walker-produced `path`/`detail` strings persisted (10-02) and rendered on `/health` (10-04) | high | mitigate | Verdict: write side unsanitized-at-write, same disposition as `File.path` today (10-02); render side is Svelte text interpolation only, source gate asserts zero raw-HTML directives in `CoverageSection.svelte`; vitest `renders a markup-bearing path as literal text — no img element is created (T-10-01)` in `web/tests/health-page.test.ts` (same discipline as T-09-06) (10-04) | closed |
| T-10-02 | Tampering | `excludedFileKey` / `c/` graphstore namespace | medium | mitigate | `appendSegment` length-prefixed encoding (never concatenation); `TestExcludedFileKeyStartsWithPrefixAndIsolatesNeighbours` (foreign-prefix loop + `/`, NUL, 0xFF path round trip); `DeleteAllExcludedFiles` bounded by `rangeUpperBound([]byte{prefixExcludedFile})` and proven not to touch `f/n/m` in `TestDeleteAllExcludedFilesClearsOnlyTheNamespace` (10-01) | closed |
| T-10-03 | Denial of Service | `GetCoverage`/`CoverageRows` page size, unbounded client paging loop | medium | mitigate | Server clamp `CoverageMaxPageSize = 1000` in `Engine.CoverageRows` (10-01), exercised at both layers by `TestCoverageRowsPageSizeClamp` and `TestGetCoveragePageSizeIsClampedOnTheWire` (10-05); client-side `COVERAGE_MAX_PAGES = 100` throws `page limit exceeded` inside `fetchAllCoverageRows`, tested (10-04) — a full-stack closure from server clamp to client bound | closed |
| T-10-04 | Tampering / DoS | `GetCoverageRequest.page_token` | medium | mitigate | Opaque base64url cursor decoded by `decodeCoverageToken` (kind byte, UTF-8, ≤ 4096 bytes), used ONLY as an in-memory comparison — never a Pebble bound, never a filesystem path (10-01); closed refusal table (≥ 6 malformed shapes, incl. a padded-base64 case) with a positive control, wire-mapped to `CodeInvalidArgument`: `TestCoverageRowsRejectsMalformedTokens`, `TestGetCoverageMalformedTokenIsInvalidArgument` (10-05) | closed |
| T-10-05 | Information Disclosure | extraction-failure `detail` carrying the absolute host path from `File.errors` | medium | mitigate | `coverageExtractionDetail` replaces `e.repoRoot` with `.` and cuts the result at 256 bytes on a rune boundary (10-01); `TestCoverageRowsDetailIsScrubbedAndBounded`, `TestGetCoverageReasonFilterOnTheWire` confirm no absolute-root leak on the wire (10-05) | closed |
| T-10-06 | Denial of Service / Elevation of Privilege | (a) a multi-megabyte file reaching the tree-sitter C scanner; (b) a "re-index this file" control on the coverage view | medium | mitigate | (a) D-04's stat-based `exceedsSizeLimit` pre-check short-circuits BEFORE any read; `parser.ErrSourceTooLarge` remains the backstop, `TestExtractPool_OversizedFileContained` unchanged and green (10-02). (b) By construction — no button/anchor/form/handler exists in `CoverageSection.svelte` (source gate + `queryAllByRole('button'|'link')` == 0 in tests); the server's method set has no mutating rpc (`TestUIServiceMethodSetIsExactlyTheReadSet`) (10-04) | closed |
| T-10-07 | Repudiation | reasons re-derived at read time (a query-time walk cannot tell a build-tag exclusion from an absent file); a stale `c/` record outliving the condition that produced it | high | mitigate | `TestCoverageSourceNeverWalksDisk` (file-scoped, positive-controlled D-14b source scan) + `internal/query/archtest` (indexer root forbidden) (10-01); `TestCoverageReasonsSurviveDiskMutationWithoutReindex` / `TestCoverageRowsSurviveDiskMutationWithoutReindex` (D-14a behavioural guard, 10-02/10-05); per-path prune on every Sync (`TestSyncPrunesAnExclusionThatBecameIndexable`, `TestSyncPrunesAnExclusionWhoseFileWasDeleted`) plus the `coverageDirty` gate so an exclusion-only change is never skipped (10-03); watched fail live against the real implementation in `10-MUTATION-LOG.md` family (c) | closed |
| T-10-08 | Tampering (schema/wire skew) | a UI enum value diverging from the schema enum; an unrecognised `ExclusionReason` number in a row | low | mitigate | Closed enums on both surfaces; `TestExclusionReasonEnumsAgree` pins the (name, number) sets both directions, `UNSPECIFIED = 0` the explicit unknown (10-01); `reasonKeyOf` falls back to `UNSPECIFIED`, `reasonLabel` renders "Unknown reason (…)", `groupCoverageRows` keeps the row — tested (10-04) | closed |
| T-10-09 | Spoofing | DNS rebinding against the new `GetCoverage` mux entry | low | accept | Verdict: inherited — `GetCoverage` registers on the SAME Connect handler `originHostGuard` already wraps; no new mux entry (03-SECURITY.md T-03-13 disposition, 09-SECURITY.md T-09-08 precedent) (10-01) | closed (accepted) |
| T-10-10 | Tampering (torn write) | exclusion records committing separately from `File`/`Node`/`Edge` mutations of the same commit | medium | mitigate | One `NewWriter(`/`Commit()` in `writeGraph` (source gate) + `TestWriteGraphStagesExcludedFilesInTheSameBatch` (`commitCalls == 1`) for a from-scratch index (10-01); `stageExclusionDiff` takes the caller's Writer, `NewWriter(` count in `sync.go` pinned at 2 (the two pre-existing sites), every Sync test reads back after ONE commit (10-03) | closed |
| T-10-11 | Tampering | `Import` of a stream with a forged/unknown record kind | low | mitigate | Closed `exportKind*` switch — an unknown kind returns the existing error (`TestImportRejectsUnknownRecordKind`); `maxImportRecordBytes` unchanged (10-01) | closed |
| T-10-12 | Tampering | a dangling symlink or unreadable file turning a whole `Discover` into an error (availability of the index) | low | accept | Verdict: pre-existing behaviour for `.go` symlinks (Go's `MatchFile` opens the file); the probe test documents the boundary; non-Go read failures are already per-file (`extract.go`). The dangling-symlink extraction-failure technique itself was VERIFIED by a live probe on first run (darwin/arm64) — no chmod-unreadable fallback was needed (10-02-SUMMARY.md key-decisions) (10-02) | closed (accepted) |
| T-10-13 | Denial of Service | a repo with many excluded files paying a full rewrite of the namespace on every Sync | low | mitigate | `proto.Equal`-based diff upserts only absent/changed records; `TestDiffExclusionsUpsertsAbsentAndChangedOnly` asserts the unchanged record is not re-written, and the sync no-op test asserts no write on a clean tree (10-03) | closed |
| T-10-14 | Information Disclosure / Tampering | a pre-Phase-10 graph claiming coverage "known" with an empty namespace | medium | mitigate | `has_coverage` is stamped only in the SAME commit that stages the exclusion diff; `TestSyncBackfillsCoverageOnAGraphThatHasFileIndexButNoCoverage` and `TestSyncFullBackfillStampsBothFlags` fabricate the pre-state (`fabricatePreCoverageStore`, a literal `HasCoverage = false` assignment) and assert records exist before the flag reads true (10-03) | closed |
| T-10-15 | Repudiation | an unknown coverage state rendered as an empty table (reads as "no gaps"); an old graph read as 0/0 over the wire | medium | mitigate | First-class `health-coverage-unknown` copy with no counts and no table, tested for both `coverage` absent and `known === false` (10-04); `TestGetCoverageAndGetHealthOnOldGraphAreKnownFalse` — both rpcs answer `known == false` with no counts, never `0/0`, never `CodeInternal` (10-05); watched fail live in `10-MUTATION-LOG.md` family (a) | closed |
| T-10-SC | Tampering | npm/pip/cargo installs | low | accept | Verdict: not applicable — this phase adds no dependency and runs no package-manager install task (10-RESEARCH.md Package Legitimacy Audit: N/A) | closed (accepted) |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above `high` count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|--------------|------|
| R-10-01 | T-10-09 | `originHostGuard` already wraps the whole mux; `GetCoverage` adds no new mux entry, so no new DNS-rebinding surface (inherited from 03-SECURITY.md T-03-13 / 09-SECURITY.md T-09-08) | plan 10-01 | 2026-09-12 |
| R-10-02 | T-10-12 | Pre-existing behaviour for `.go` symlinks; the extraction-failure technique this phase's fixture relies on was verified by a live probe, not merely assumed | plan 10-02 | 2026-09-12 |
| R-10-03 | T-10-SC | No package is installed by this phase (10-RESEARCH.md Package Legitimacy Audit: N/A) | plans 10-01..10-05 | 2026-09-12 |
| R-10-04 | SRV-04 (paging consistency, `CoverageRows` doc comment) | `CoverageSummary`/`CoverageRows` read from whatever snapshot the caller's Engine holds; two consecutive `CoverageRows` page calls against a live server may be answered from different snapshots (each request opens its own Engine, SRV-04), so paging is a best-effort walk of a changing graph, not a single consistent cursor across calls — accepted, documented behaviour, not a defect | plan 10-01 | 2026-09-12 |

---

## Notes

**1. Open Question 1 (10-RESEARCH.md) — export/import round trip: implemented.**
`exportKindExcludedFile = 5` is the fifth export record kind (`internal/graphstore/export.go`),
with a lossless round-trip test exercised in Plan 01's Task 2. No `.planning/WINDOWS.md` entry
was needed — the round trip is proven, not deferred.

**2. Open Question 3 (10-RESEARCH.md) — live-browser verification: not adopted, decided
vitest-level.** Plan 10-04's key-decisions record the rationale: HLT-04's text carries no
live-browser mandate, the Go real-listener tests already prove the wire (Plan 01), and
`CoverageSection.svelte` is a pure projection of `GetHealthResponse.coverage` /
`GetCoverageResponse` with no new browser-only behavior (viewport, focus, animation) that only
a live browser could exercise. No `web/scripts/coverage-check.mjs` was built; verification
stays at `web/tests/health-page.test.ts` / `health-view.test.ts` (vitest).

**3. Discretion decisions recorded as verdicts.** The key-prefix byte for the new namespace
(`'c'`, `prefixExcludedFile` in `internal/graphstore/keys.go` — one of the free single-byte
prefixes named in 10-CONTEXT.md D-05) and the `excluded_by_reason` map's wire keys (the enum's
full generated names, e.g. `"EXCLUSION_REASON_BUILD_TAG"`, not numbers — D-09 discretion,
resolved through `ExclusionReasonSchema.values` on the TS side, never a hand-written switch)
were both Claude's own discretion per 10-CONTEXT.md, exercised and recorded in Plan 01's
key-decisions.

**4. D-12 untouched surfaces confirmed.** `codegraph status --json` and the MCP surface are
out of scope this phase (10-CONTEXT.md D-12). Plan 01's own gate proved no coverage symbol
leaked into `internal/query/status.go`, `internal/cli/`, or `internal/mcp/`:
`statusResultKnownFieldNames` in `internal/query/meta_test.go` stayed untouched and green, and
`task test:golden` reported the frozen `status.json` goldens unmoved
(`ok github.com/seanb4t/codegraph-go/testdata/golden`, 10-01-SUMMARY.md).

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-09-12 | 16 | 16 | 0 | plan 10-06 (ASVS L1 inline; auditor short-circuited per `threats_open:0` + `register_authored_at_plan_time:true` + `asvs_level:1`) |

**Audit note — what was checked by execution vs. by reading.** The three `high`-severity rows
(T-10-01, T-10-07) — and every row citing a test this plan's own `10-MUTATION-LOG.md` exercises
(T-10-15 family (a), T-10-07 family (c)) — were checked by **execution** during this plan:
`TestCoverageSummaryOnOldGraphIsUnknown`, `TestCoverageRowsUnknownGraphAndSingleRow`,
`TestGetCoverageAndGetHealthOnOldGraphAreKnownFalse`, `TestCoverageSourceNeverWalksDisk`,
`TestCoverageRowsSurviveDiskMutationWithoutReindex`, `TestUIServiceMethodSetIsExactlyTheReadSet`,
and `TestCoverageRPCNameClearsMutatingVerbsWhileDecoyIsRejected` were all re-run at HEAD and
observed both RED (under the family (a)/(b)/(c) mutations) and GREEN (after revert). Every
other `Test…`-cited name in this register was independently resolved to a real `func Test…(`
definition via `rg` against `internal/` at HEAD — not trusted from any plan's prose. Every
`.test.ts`-cited vitest file was confirmed present under `web/tests/`. The `low`/`medium` rows
disposed `accept` (T-10-09, T-10-12, T-10-SC) were checked by **reading** the cited
source/design property and the inherited prior-phase disposition, consistent with
08-SECURITY.md's and 09-SECURITY.md's own audit-note precedent for accepted risks that inherit
an existing, already-verified guard.

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-09-12

**Outstanding, not security-blocking:**
- Open Question 3's optional live-Chromium coverage gate remains unbuilt by deliberate decision
  (Notes item 2 above) — vitest-level coverage is the current, sufficient state per Plan 04's
  own rationale.
