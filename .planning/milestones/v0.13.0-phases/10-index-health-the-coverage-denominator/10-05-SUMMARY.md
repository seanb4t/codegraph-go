---
phase: 10-index-health-the-coverage-denominator
plan: 05
subsystem: index-health
tags: [coverage, pagination, connect-rpc, pebble, graphstore, tdd]

# Dependency graph
requires:
  - phase: 10-index-health-the-coverage-denominator
    provides: "Plan 01's CoverageSummary/CoverageRows Engine API and GetHealth/GetCoverage rpcs; Plan 02's committed testdata/coverage fixture plus writeOversizeGoFile/writeDanglingSymlink generation technique; Plan 03's Sync-side exclusion pruning; Plan 04's health-page Coverage UI section"
provides:
  - "Exact, test-pinned discovered/indexed/excluded/extraction_failed counts and by-reason breakdown against the full D-13 fixture, at both the Engine level (store outside the repo) and the real listener level (store pre-created in-repo, matching `codegraph init`'s own ordering) — the two differ by exactly one DIR_DOTPREFIX row (`.codegraph` itself) and both are now asserted, not assumed"
  - "A fixed CoverageRows pagination bug: the resume cursor now walks each segment by skip-until-seen-cursor-path rather than a lexical `path <= cursorPath` comparison, matching the store's real length-prefixed key order"
  - "A closed, test-driven refusal table for malformed GetCoverage page tokens (6 shapes) with a positive control, proven at both the Engine level and over the real wire with the Connect code and no-path-leak assertions"
  - "Reason-filter, page-size clamp (200 default / 1000 max), scrubbed+bounded extraction-failure detail, and the known-vs-unknown-graph contrast, all proven at both layers"
affects: [10-06]

# Actuals (#2632)
actuals:
  tokens: 11600
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Fixture-through-the-real-production-path: copyCoverageFixtureForUI pre-creates storeDir via os.MkdirAll before indexer.Run, mirroring internal/cli/init.go's actual `codegraph init` ordering, so DiscoverAll's walk sees \".codegraph\" already on disk and records it as its own DIR_DOTPREFIX exclusion — deliberately different from the Engine-level and existing uiserverBuildTagFixture helpers, which avoid that phantom for a cleaner unit count. Both orderings are now real, asserted contracts rather than one being an untested assumption."
    - "Skip-until-seen-cursor-path resume walk: a paged store-order iterator that is not guaranteed to agree with lexical string order cannot resume via `path <= cursorPath`; instead each segment replays from its own start and skips forward until it re-observes the exact cursor path once, then resumes emitting."

key-files:
  created: []
  modified:
    - internal/query/coverage.go
    - internal/query/coverage_test.go
    - internal/uiserver/coverage_test.go

key-decisions:
  - "Fixed a genuine CoverageRows pagination bug (Rule 1): the resume cursor compared paths with plain string `<=`, but ExcludedFile/File iteration order is the store's length-prefixed key encoding (keys.go's appendSegment), not lexical path order. Once the cursor moved into the 'x' segment, the 'f' segment's lexical-comparison guard never fired, so every subsequent page re-emitted the same extraction-failed row forever, corrupting concatenation stability and producing an infinite non-terminating page sequence for any fixture with a mix of extraction failures and later-alphabetically-shorter excluded paths. Fixed by tracking a per-segment `skipping` flag that flips off on exact-path match, and skipping the 'f' segment entirely once the cursor sits in 'x'."
  - "The literal row order documented in the plan's must_haves prose (lexical ASCII order) does not match the store's actual key order (length-prefixed: shorter paths sort before longer ones, ties broken by content) — confirmed empirically and consistent with CoverageRows' own pre-existing doc comment (\"each segment walked in the store's own key order\"). Tests assert the REAL order (broken.py, go.mod, vendor, .hidden, huge.go, notes.md, tagged.go[, .codegraph]) rather than the plan's illustrative example, since the acceptance criteria require stability and opacity, not a specific lexical order."
  - "TestCoverageRowsRejectsMalformedTokens' 'empty payload' case from the plan prose (\"base64url of \\\"\\\"\") is mathematically unreachable as a malformed shape: encoding zero bytes always round-trips to the literal empty string, which decodeCoverageToken deliberately special-cases as 'first page' (its own doc comment) — not malformed. Substituted a 'padded (non-raw-url) base64' case (a valid raw-url payload corrupted with trailing '=' padding) so the table still exercises 6 distinct validation branches with a case that is actually malformed."

patterns-established:
  - "When a paginated store-backed rpc's resume cursor must survive an iteration order that is NOT guaranteed to equal the natural ordering of the cursor value's own type (e.g. a length-prefixed key encoding vs. a lexical string field), implement resume as 'skip until we re-observe the cursor value once', never as a relational comparison against the cursor value."

requirements-completed: [HLT-04, HLT-05, HLT-06]

coverage:
  - id: D1
    description: "Engine-level coverage contract pinned against the full fixture: exact counts, by-reason breakdown, and the additive invariant"
    requirement: HLT-05
    verification:
      - kind: unit
        ref: "internal/query/coverage_test.go#TestCoverageSummaryMatchesFixture"
        status: pass
    human_judgment: false
  - id: D2
    description: "CoverageRows pagination is stable across page sizes and token replays, and tokens are opaque"
    requirement: HLT-06
    verification:
      - kind: unit
        ref: "internal/query/coverage_test.go#TestCoverageRowsOrderingAndPagingAreStable"
        status: pass
    human_judgment: false
  - id: D3
    description: "Reason filter, page-size clamp, malformed-token refusal table, empty-known contrast, and detail scrubbing/bounding are all pinned at the Engine level"
    requirement: HLT-06
    verification:
      - kind: unit
        ref: "internal/query/coverage_test.go#TestCoverageRowsReasonFilter"
        status: pass
      - kind: unit
        ref: "internal/query/coverage_test.go#TestCoverageRowsRejectsMalformedTokens"
        status: pass
      - kind: unit
        ref: "internal/query/coverage_test.go#TestCoverageRowsPageSizeClamp"
        status: pass
      - kind: unit
        ref: "internal/query/coverage_test.go#TestCoverageRowsEmptyPageOnKnownGraphWithZeroExclusions"
        status: pass
      - kind: unit
        ref: "internal/query/coverage_test.go#TestCoverageRowsDetailIsScrubbedAndBounded"
        status: pass
    human_judgment: false
  - id: D4
    description: "D-14a behavioural guard survives at the Engine level: coverage reasons are read back from the store, never reconstructed by re-walking mutated disk"
    requirement: HLT-05
    verification:
      - kind: unit
        ref: "internal/query/coverage_test.go#TestCoverageRowsSurviveDiskMutationWithoutReindex"
        status: pass
    human_judgment: false
  - id: D5
    description: "Wire-level coverage contract pinned through the real listener, including the .codegraph in-repo DIR_DOTPREFIX effect"
    requirement: HLT-06
    verification:
      - kind: integration
        ref: "internal/uiserver/coverage_test.go#TestGetHealthCoverageMatchesFixture"
        status: pass
    human_judgment: false
  - id: D6
    description: "GetCoverage paging, malformed-token CodeInvalidArgument mapping, page-size clamp, reason filter, and the old-graph known=false contrast all hold over the real Connect wire"
    requirement: HLT-06
    verification:
      - kind: integration
        ref: "internal/uiserver/coverage_test.go#TestGetCoveragePagesRoundTripThroughTheListener"
        status: pass
      - kind: integration
        ref: "internal/uiserver/coverage_test.go#TestGetCoverageMalformedTokenIsInvalidArgument"
        status: pass
      - kind: integration
        ref: "internal/uiserver/coverage_test.go#TestGetCoveragePageSizeIsClampedOnTheWire"
        status: pass
      - kind: integration
        ref: "internal/uiserver/coverage_test.go#TestGetCoverageReasonFilterOnTheWire"
        status: pass
      - kind: integration
        ref: "internal/uiserver/coverage_test.go#TestGetCoverageAndGetHealthOnOldGraphAreKnownFalse"
        status: pass
    human_judgment: false

duration: 55min
completed: 2026-09-12
status: complete
commits: 2
plan_head_before: b9e10377
---

# Phase 10 Plan 05: Full-Fixture Engine + Listener Coverage Contract Summary

**Pinned the exact discovered/indexed/excluded/extraction_failed coverage numbers, stable two-segment paging, and a closed malformed-token refusal set at both the Engine and wire layers — and fixed a real CoverageRows pagination bug the tests exposed (lexical cursor comparison against a non-lexical store key order).**

## Performance

- **Duration:** 55 min
- **Started:** 2026-09-12T00:00:00Z (approx.)
- **Completed:** 2026-09-12
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments

- `TestCoverageSummaryMatchesFixture` (Engine level) proves the full D-13 fixture — Plan 02's six committed files plus generated `huge.go`/`broken.py` — reports exactly `discovered=6 indexed=1 excluded=6 extraction_failed=1` with the complete by-reason breakdown, store held outside the repo copy.
- `TestGetHealthCoverageMatchesFixture` (wire level) proves the identical fixture, indexed the way `codegraph init` really orders it (storeDir pre-created before discovery), reports `discovered=6 indexed=1 excluded=7 extraction_failed=1` — the extra row is `.codegraph` itself as a second DIR_DOTPREFIX exclusion, asserted as a real row, not just a count.
- Fixed a genuine pagination bug in `CoverageRows`: the resume cursor's `path <= cursorPath` lexical comparison silently broke once paging crossed segments, because the store's real iteration order is a length-prefixed key encoding, not lexical path order — this caused infinite duplicate rows on any multi-segment paged fixture. Replaced with a skip-until-seen-cursor-path walk.
- A closed six-shape malformed-page-token refusal table (not-base64, corrupted-padding, unknown kind byte, oversized path, invalid UTF-8, cross-segment-with-filter) is proven at both layers with a positive control, plus `CodeInvalidArgument` and no-path-leak assertions on the wire.
- Reason filter, page-size clamp (200/1000), scrubbed+bounded extraction detail, and the known-vs-unknown-graph contrast are all proven at both layers.

## Task Commits

1. **Task 1: Engine-level contract** — `152018cd` (test) — fixture summary, ordering/paging stability, reason filter, token refusals, clamp, empty-known, detail scrubbing; includes the `CoverageRows` cursor fix
2. **Task 2: Listener-level contract** — `dfdca783` (test) — fixture numbers over `GetHealth`, `GetCoverage` paging round trip, `CodeInvalidArgument` mapping, clamp, filter, old graph — through the real server

**Plan metadata:** (this commit)

## Files Created/Modified

- `internal/query/coverage.go` — fixed `CoverageRows`' resume-cursor logic (skip-until-seen-cursor-path per segment instead of a lexical `<=` comparison); no other behavior changed
- `internal/query/coverage_test.go` — 8 new tests plus fixture helpers (`copyCoverageFixtureForQuery`, `writeOversizeGoFileForQuery`, `writeDanglingSymlinkForQuery`, `indexCoverageFixtureExternal`)
- `internal/uiserver/coverage_test.go` — 6 new tests plus fixture helpers (`copyCoverageFixtureForUI`, `writeOversizeGoFileForUI`, `fmtPaddedNameForUI`)

## Decisions Made

See `key-decisions` in frontmatter: the CoverageRows cursor fix, the empirically-verified (not prose-assumed) row order, and the malformed-token table substitution for the mathematically-unreachable "empty payload via empty-string encoding" case.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed CoverageRows pagination cursor comparing paths lexically against a non-lexical store key order**
- **Found during:** Task 1, writing `TestCoverageRowsOrderingAndPagingAreStable`
- **Issue:** `CoverageRows`' resume logic used `path <= cursorPath` (a plain string comparison) to decide "already returned this row." The underlying `IterateExcludedFiles`/`IterateFiles` order is the store's length-prefixed key encoding (`keys.go`'s `appendSegment`: shorter paths sort before longer ones regardless of lexical content), which is NOT the same ordering as plain string comparison whenever paths differ in byte length. Once a page's cursor advanced into the 'x' (excluded) segment, the 'f' (extraction-failed) segment's guard (`cursorSeg == 'f' && ...`) never applied, so the extraction-failed row(s) were re-emitted on every subsequent page — confirmed by an isolated reproduction (`go test` probe) that showed `[broken.py go.mod vendor broken.py]` for a 3-row-page walk that should never repeat a row. This would also silently corrupt any page whose 'x'-segment cursor happened to sort lexically ahead of not-yet-returned rows still pending in store-iteration order (also reproduced).
- **Fix:** Replaced the lexical cursor comparison with a per-segment `skipping` flag that starts `true` only when the cursor names that segment, flips to `false` the instant the iterator reproduces the exact cursor path (a fresh iterator over unchanged data reproduces an identical sequence), and skips the 'f' segment in its entirety once the cursor is in the later 'x' segment.
- **Files modified:** `internal/query/coverage.go`
- **Verification:** `TestCoverageRowsOrderingAndPagingAreStable` (concatenation equality via `reflect.DeepEqual`, token-replay identity, opaque-token check) plus the full `go test ./internal/query/...` suite green
- **Committed in:** `152018cd` (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Necessary for pagination correctness — without it, `GetCoverage` paging was silently broken for any fixture mixing extraction failures with excluded files. No scope creep; the fix is confined to `CoverageRows`' cursor logic, exactly as the plan anticipated ("any fix... is small and local").

## Issues Encountered

- The plan's `must_haves.truths` prose describes the single-page row order as lexical ASCII order (`.hidden, go.mod, huge.go, notes.md, tagged.go, vendor`). Empirical verification (and `CoverageRows`' own pre-existing doc comment, "each segment walked in the store's own key order") shows the actual, stable order is the store's length-prefixed key order (`go.mod, vendor, .hidden, huge.go, notes.md, tagged.go`). Tests assert the real, verified order — the acceptance criteria require stability and token opacity, not a specific lexical ordering, so this is a documentation imprecision in the plan rather than a defect to chase.
- The plan's second `<verify>` block for Task 1 greps for the literal string `CoverageMaxPageSize = 1000` (single space) in `coverage.go`; gofmt's column-alignment inserts extra spaces before `=` in that const block (confirmed `gofmt -l` reports the file as already correctly formatted), so the literal regex only matches 2 of the 3 intended constants. This is pre-existing since Plan 01 and unrelated to this plan's changes — confirmed the three constants (`CoverageDefaultPageSize = 200`, `CoverageMaxPageSize = 1000`, `coverageDetailMaxBytes = 256`) are all present and correctly valued via direct inspection and the passing test suite; not reformatted away from gofmt canonical style to satisfy a literal grep.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `internal/query/coverage.go` and both `coverage_test.go` files are green, gofmt-clean, `go vet`-clean, and `internal/query/archtest` remains green (zero forbidden filesystem calls in `coverage.go`, no indexer-root import in production code).
- Plan 06 (mutation log, security register, validation map, phase-close gate) can now cite `TestCoverageRowsSurviveDiskMutationWithoutReindex` (Engine-level D-14a twin) and this plan's fixture/paging/refusal tests directly by name for its Per-Task Verification Map and Threat Register test links.
- `HLT-04/05/06` requirements remain `Pending` in `REQUIREMENTS.md` per the shared-ID gate (#2388): Plan 06 also declares all three IDs and has not yet produced a SUMMARY, so completion is correctly deferred until it finishes.

---
*Phase: 10-index-health-the-coverage-denominator*
*Completed: 2026-09-12*

## Self-Check: PASSED
