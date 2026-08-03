---
phase: 01-behavioral-parity-explore-node
plan: 07
subsystem: api
tags: [explore, hybrid-search, rwr, scoring, go]

requires:
  - phase: 01-behavioral-parity-explore-node
    provides: "internal/query/tokenize.go (01-03) — extractSymbolsFromQuery (H1) feeds channels 1/2, extractSearchTerms (H2) feeds channel 3"
provides:
  - "internal/query/gather.go — gatherChannel1 (H3), gatherChannel2 (H4), gatherChannel3 (H5), gatherMerge (H6), and the shared isTestFile path predicate"
  - "gatherCandidate/gatherChannelKind scaffold (node + accumulated score + which channel(s) hit) that plan 10's rerankers extend"
affects: ["01-10 (extends gather.go with H7-H21 rerankers)", "01-11 (subgraph expansion consumes the merged candidate set)", "01-14 (relevance gate reuses isTestFile)"]

tech-stack:
  added: []
  patterns:
    - "Three independently-scored gather channels (exact-name+co-location, titlecase definition-prefix, FTS multi-term) merged max-score-wins by node id, deterministic score-desc/Id-asc tie-break (D-04) throughout"
    - "gatherCandidate.Channels set (union across channels) so downstream heuristics can see provenance, not just the winning score"
    - "Shared path-based isTestFile predicate defined once (gather.go) instead of duplicated across plans 10/14"

key-files:
  created:
    - internal/query/gather.go
    - internal/query/gather_test.go
  modified: []

key-decisions:
  - "Base per-match scores for channel 1 (exact-name) and channel 3 (FTS) are a documented default (10.0, matched across both channels) — RESEARCH §C.2 pins only the boost DELTAS (+20*(distinctSymbolsInFile-1) for H3, +5*(termHits-1) for H5), not the flat single-match base; the TS install's dist JS is no longer present on this machine (only .d.ts declarations remain — verified: dist/ has zero .js files), so the base could not be re-derived from source and is called out explicitly in gather.go's doc comments"
  - "Channel 2's definitionKinds filter maps TS's class/interface/struct/trait/protocol/enum/type_alias set onto exactly three Go Kind values (KindStruct, KindInterface, KindTypeAlias) — every priority-4 extractor in this repo already collapses class/struct/trait/protocol/enum into KindStruct (documented in javaextract/pyextract/tsextract's own types.go comments), so there is no separate class/enum Kind anywhere to have missed; a documented D-02 consolidation, not a dropped kind"
  - "Channel 2 does not apply TS's getStemVariants() stem-variant expansion — inherited from 01-03's explicit D-02 deferral (extractSymbolsFromQuery/extractSearchTerms ship with no stem-variant hook); this plan's action text assumed H2 already had stem variants, which 01-03-SUMMARY.md's own Next-Phase-Readiness note flags as still unported. Documented here as an inherited divergence, not re-litigated"
  - "channel3ImportKind (\"import\") exclusion is ported verbatim even though no priority-4 extractor in this repo currently emits an import-kind NODE (imports are edges, RefKindImports) — a no-op today, kept for behavioral parity and future-proofing, and directly tested via a synthetic import-kind node"

patterns-established:
  - "gatherMerge's channel-union-plus-max-score-wins shape is the pattern plan 10's H7-H21 rerankers apply their own score mutations on top of, without needing to re-run the three channels"

requirements-completed: [EXPL-02]

coverage:
  - id: D1
    description: "Channel 1 (H3): exact-name lookup + co-location boost, verbatim +20*(distinctSymbolsInFile-1) constant, trimmed to searchLimit*2"
    requirement: "EXPL-02"
    verification:
      - kind: unit
        ref: "internal/query/gather_test.go#TestGatherChannel1_CoLocationBoost"
        status: pass
      - kind: unit
        ref: "internal/query/gather_test.go#TestGatherChannel1_TrimsToSearchLimitTimesTwo"
        status: pass
    human_judgment: false
  - id: D2
    description: "Channel 2 (H4): titlecase definition-prefix search restricted to definition kinds, verbatim 15+brevityBonus formula"
    requirement: "EXPL-02"
    verification:
      - kind: unit
        ref: "internal/query/gather_test.go#TestGatherChannel2_TitlecasePrefixBrevity"
        status: pass
      - kind: unit
        ref: "internal/query/gather_test.go#TestGatherChannel2_OnlyDefinitionKindsParticipate"
        status: pass
    human_judgment: false
  - id: D3
    description: "Shared isTestFile path predicate (RESEARCH §5) — filename patterns, test directories, non-production directories"
    requirement: "EXPL-02"
    verification:
      - kind: unit
        ref: "internal/query/gather_test.go#TestIsTestFile"
        status: pass
    human_judgment: false
  - id: D4
    description: "Channel 3 (H5): FTS multi-term search, verbatim +5*(termHits-1) constant, import-kind exclusion unless an explicit kind filter is given"
    requirement: "EXPL-02"
    verification:
      - kind: unit
        ref: "internal/query/gather_test.go#TestGatherChannel3_MultiTermBoost"
        status: pass
      - kind: unit
        ref: "internal/query/gather_test.go#TestGatherChannel3_ExcludesImportKindWithoutFilter"
        status: pass
    human_judgment: false
  - id: D5
    description: "H6 merge: dedup by node id across all three channels, max-score-wins (not summed), deterministic order"
    requirement: "EXPL-02"
    verification:
      - kind: unit
        ref: "internal/query/gather_test.go#TestGatherMerge_MaxScoreWinsNotSummed"
        status: pass
      - kind: unit
        ref: "internal/query/gather_test.go#TestGatherMerge_UnionAcrossDistinctNodes"
        status: pass
    human_judgment: false

duration: 15min
completed: 2026-07-15
status: complete
---

# Phase 1 Plan 07: Hybrid Gather Channels (H3-H6) Summary

**Three independently-scored TS-parity gather channels (exact-name+co-location, titlecase definition-prefix, FTS multi-term) merged max-score-wins into `internal/query/gather.go`, plus the shared path-based `isTestFile` predicate plans 10/14 both reuse.**

## Performance

- **Duration:** ~15 min
- **Completed:** 2026-07-15T13:38:27Z
- **Tasks:** 2 (each RED → GREEN)
- **Files modified:** 2 (both new)

## Accomplishments
- `gatherChannel1` (H3): exact-name lookup over `extractSymbolsFromQuery`'s output, with the verbatim `+20*(distinctSymbolsInFile-1)` co-location boost for files that define more than one distinct query symbol, trimmed to `searchLimit*2`
- `gatherChannel2` (H4): titlecase definition-prefix search restricted to `definitionKinds` (struct/interface/type_alias — this codebase's collapsed mapping of TS's class/interface/struct/trait/protocol/enum/type_alias set), with the verbatim `15 + brevityBonus` formula (`brevityBonus = max(0, 10-(nameLen-prefixLen)/3)`)
- `gatherChannel3` (H5): FTS-style multi-term text search over Name/QualifiedName driven by `extractSearchTerms`'s output, with the verbatim `+5*(termHits-1)` boost and import-kind exclusion unless an explicit kind filter is supplied
- `gatherMerge` (H6): dedup by node id across any number of channel result sets, keeping the MAX score (not the sum) and the UNION of every channel that hit a node
- `isTestFile`: RESEARCH §5's path-based test-file predicate — filename patterns (`test_*`/`*_test.*`/`*.test.*`/`*-spec.*`/`*Test.ext`/`*Tests.ext`/`*TestCase.ext`/`*Spec.ext`), test directories (`tests?`/`__tests__`/`specs?`/`testlib`/`testing`/CamelCase `*Test*` Gradle-Kotlin dirs), and non-production directories (integration/sample(s)/example(s)/fixture(s)/benchmark(s)/demo(s)) — defined once here, distinct from `traverse.go`'s symbol-name `isTestSymbol`
- All four functions plus the shared `gatherCandidate`/`gatherChannelKind`/`sortGatherCandidates` scaffold are deterministic (score-descending then Id-ascending, D-04's codebase-wide tie-break)

## Task Commits

Each task's RED and GREEN steps were committed atomically:

1. **Task 1: Channels 1-2 + shared isTestFile** — RED `70fa25a` (test), GREEN `c2685ee` (feat)
2. **Task 2: Channel 3 + H6 merge** — RED `f565de5` (test), GREEN `9715bc6` (feat)

_TDD gate sequence verified: test(...) commits precede their feat(...) commits in git log for both tasks._

## Files Created/Modified
- `internal/query/gather.go` - `gatherCandidate`, `gatherChannelKind`, `definitionKinds`, `sortGatherCandidates`, `gatherChannel1`, `titlecase`, `gatherChannel2`, `gatherChannel3`, `gatherMerge`, `isTestFile` + their supporting regexps/constant blocks
- `internal/query/gather_test.go` - `gatherFakeReader` test double (mirrors `search_test.go`'s `searchFakeReader`) + table/scenario tests for every channel, the merge, and `isTestFile`

## Decisions Made
- Chose a documented default base score (10.0) for channel 1's exact-name match and channel 3's single-term-hit match, since RESEARCH only pins the boost deltas, not the flat base — see key-decisions above for full rationale (the TS dist JS is no longer present on this machine to re-derive it from source; only its `.d.ts` type declarations remain).
- Mapped TS's 7-kind Channel-2 definition-kind set onto this codebase's 3 collapsed Kind values (struct/interface/type_alias) rather than inventing new Kind constants, since every priority-4 extractor already documents that collapse.
- Did not add stem-variant expansion to channel 2, consistent with 01-03's explicit deferral of `getStemVariants()`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug/Plan-source mismatch] Channel 2 does not use stem variants, correcting the plan's assumption that H2 already has them**
- **Found during:** Task 1 (implementing `gatherChannel2`)
- **Issue:** The plan's action text says to expand "query terms with the stem variants used by H2," but 01-03-SUMMARY.md (this plan's own dependency) explicitly and deliberately deferred `getStemVariants()` — neither `extractSymbolsFromQuery` nor `extractSearchTerms` has a stem-variant hook, so there is nothing to reuse.
- **Fix:** Implemented channel 2 against the literal titlecased symbol tokens only, matching what 01-03 actually shipped, and documented the inherited divergence in `gatherChannel2`'s doc comment for whichever future plan ports `getStemVariants`.
- **Files modified:** internal/query/gather.go
- **Verification:** `go test ./internal/query/ -run TestGatherChannel2 -count=1 -v` — all subtests pass
- **Committed in:** c2685ee (Task 1 GREEN)

---

**Total deviations:** 1 auto-fixed (Rule 1 — the plan's action text assumed a dependency capability that plan 03 had already, deliberately, not shipped; resolved in favor of the actual shipped tokenizer contract)
**Impact on plan:** No scope creep — the channel still ports every constant RESEARCH cites verbatim; only the stem-variant input source is scoped to what actually exists today.

## Issues Encountered
The TS 1.3.1 install's dist JS source (readable by earlier plans per D-01, e.g. 01-03) is no longer present on this machine — only `.d.ts` type-declaration files remain under `/opt/homebrew/lib/node_modules/@colbymchenry/codegraph/dist/` (verified: zero `.js` files anywhere under `dist/`). This plan's constants come entirely from 01-RESEARCH.md's frozen citations (§C.2's table), which is sufficient for every constant this plan needed to port verbatim; the two base-score values not cited by RESEARCH (channel 1/3 per-match base) are documented defaults rather than re-derived from source, per the key-decisions above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- `internal/query/gather.go`'s `gatherChannel1`/`gatherChannel2`/`gatherChannel3`/`gatherMerge` are ready for plan 10 to extend with H7-H21 (test-file dampening, core-directory boost, multi-term co-occurrence re-rank, type-hierarchy expansion, etc.) and to wire into `Explore()` as the replacement for `matchNodes`'s naive lexical input construction (RESEARCH Pitfall 1) — none of that wiring happened in this plan by design (see this plan's key_context note).
- `isTestFile` is ready for plan 10's H7 test-file dampening and plan 14's relevance gate to import directly — both are unexported package-level functions within `internal/query`, no further plumbing needed.
- The channel-1/channel-3 base-score values (10.0 each) are an implementation detail, not a parity-critical constant — if a later plan captures the true TS base score from a fresh install, only the two constant declarations in gather.go need to change; no call site depends on the specific value.

---
*Phase: 01-behavioral-parity-explore-node*
*Completed: 2026-07-15*

## Self-Check: PASSED

All created files and commit hashes verified present.
