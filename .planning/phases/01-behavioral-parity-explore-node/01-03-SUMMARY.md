---
phase: 01-behavioral-parity-explore-node
plan: 03
subsystem: api
tags: [tokenizer, regexp, explore, go]

requires:
  - phase: 01-behavioral-parity-explore-node
    provides: (no direct plan dependency — depends_on is empty; reads TS 1.3.1 dist source directly per D-01)
provides:
  - internal/query/tokenize.go — extractSymbolsFromQuery (H1) + extractSearchTerms (H2), the two distinct TS CodeGraph 1.3.1 query tokenizers, ported verbatim
  - stopWords (H2's filter set) and commonWords (H1's filter set) as separate package-level maps
affects: [01-04 (named-symbol seeding, +50 file score), 01-07 (FTS gather channel), explore CLI multi-word query fix]

tech-stack:
  added: []
  patterns:
    - "Two distinct pure-function tokenizers per query string, never conflated: H1 (extractSymbolsFromQuery/commonWords) feeds named-symbol seeding; H2 (extractSearchTerms/stopWords) feeds FTS term gathering"
    - "Ordered-slice + seen-map token accumulation (never map iteration) for deterministic first-seen scan order (D-04)"
    - "WR-05/V5 empty-query guard: TrimSpace(query)==\"\" returns []string{}, never a match-all sentinel"

key-files:
  created:
    - internal/query/tokenize.go
    - internal/query/tokenize_test.go
  modified: []

key-decisions:
  - "Read the live TS 1.3.1 dist source directly (still installed at /opt/homebrew/lib/node_modules/@colbymchenry/codegraph) instead of relying solely on RESEARCH.md's excerpted code, per D-01's 'do NOT guess or hardcode reverse-engineered approximations' mandate — RESEARCH's H2 excerpt omitted the compound-preservation step's actual regex, and the plan's illustrative example incorrectly attributed 'get' to STOP_WORDS (it's actually in H1's commonWords, not H2's stopWords) — both corrected against verified ground truth"
  - "Deferred getStemVariants() FTS-prefix stem expansion (query-utils.js:129-175) — explicitly permitted by the plan's action text ('note it if deferred as a documented D-02 divergence'); extractSearchTerms has no stem-variant parameter, kept as a simple two-arg-free function per CLAUDE.md's simplicity preference rather than adding an always-false boolean parameter"

patterns-established:
  - "Pattern 1: RWR power iteration" - not this plan
  - "Verbatim TS port with file:line citations in doc comments, so future plans (07, 12) can trace behavior back to the TS source"

requirements-completed: [EXPL-01, EXPL-02]

coverage:
  - id: D1
    description: "extractSearchTerms (H2) + stopWords ported verbatim: compound preservation, camelCase/snake_case split, stopword filtering, WR-05 empty-query guard, deterministic order"
    requirement: "EXPL-01"
    verification:
      - kind: unit
        ref: "internal/query/tokenize_test.go#TestExtractSearchTerms"
        status: pass
    human_judgment: false
  - id: D2
    description: "extractSymbolsFromQuery (H1) + commonWords ported verbatim: CamelCase/snake_case/SCREAMING_SNAKE/acronym/dot-notation/plain-lowercase extraction, commonWords filtering, WR-05 empty-query guard, deterministic order"
    requirement: "EXPL-02"
    verification:
      - kind: unit
        ref: "internal/query/tokenize_test.go#TestExtractSymbolsFromQuery"
        status: pass
    human_judgment: false

duration: 20min
completed: 2026-07-15
status: complete
---

# Phase 1 Plan 03: Query Tokenizers (H1 + H2) Summary

**Both of TS CodeGraph 1.3.1's distinct query tokenizers — `extractSymbolsFromQuery` (H1, named-symbol seeding) and `extractSearchTerms` (H2, FTS term gathering) — ported verbatim into `internal/query/tokenize.go`, each with its own separate filter word-list (`commonWords` vs `stopWords`), fully TDD'd and WR-05-safe.**

## Performance

- **Duration:** ~20 min
- **Completed:** 2026-07-15T12:29:25Z
- **Tasks:** 2 (each RED → GREEN)
- **Files modified:** 2 (both new)

## Accomplishments
- `extractSearchTerms` (H2): compound-preserving camelCase/snake_case tokenizer with the exact 90-ish-word `STOP_WORDS` filter set from `search/query-utils.js:102-120,189-242`
- `extractSymbolsFromQuery` (H1): 6-pattern identifier extractor (CamelCase/snake_case/SCREAMING_SNAKE/acronym/dot-notation/plain-lowercase) with the exact `commonWords` filter set from `context/index.js:64-145`
- Both functions verified against the live TS 1.3.1 install (still present at `/opt/homebrew/lib/node_modules/@colbymchenry/codegraph`), not just RESEARCH.md's excerpt — confirms exact regex patterns and word lists byte-for-byte
- WR-05/V5 empty-query guard on both: empty or whitespace-only input returns `[]string{}`, never expandable to "match everything"
- Deterministic first-seen scan order on both (D-04): ordered-slice + seen-map accumulation, no map iteration anywhere in the output path

## Task Commits

Each task's RED and GREEN steps were committed atomically:

1. **Task 1: extractSearchTerms (H2) + STOP_WORDS** — RED `472cedf` (test), GREEN `436dd9e` (feat)
2. **Task 2: extractSymbolsFromQuery (H1) + commonWords** — RED `dd5febc` (test), GREEN `3c250e5` (feat)

_TDD gate sequence verified: test(...) commits precede their feat(...) commits in git log for both tasks._

## Files Created/Modified
- `internal/query/tokenize.go` - `stopWords`, `commonWords`, `extractSearchTerms`, `extractSymbolsFromQuery`, and their supporting compiled regexps
- `internal/query/tokenize_test.go` - `TestExtractSearchTerms`, `TestExtractSymbolsFromQuery` (table of subtests each)

## Decisions Made
- Consulted the live TS 1.3.1 dist source directly rather than relying solely on RESEARCH.md's excerpt, per D-01. This surfaced two corrections to the plan's illustrative examples (see Deviations below) — both resolved in favor of the verified ground-truth source, consistent with D-01's explicit instruction.
- Deferred `getStemVariants()` FTS-prefix stem expansion entirely (not even a stub parameter) — the plan's action text explicitly permits this deferral, and adding an always-`false` boolean parameter with no live callers would be dead-code-adjacent complexity the project's style guidance discourages. A follow-on plan (07, FTS gather channel) can add the hook when it's actually wired to something.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Corrected the plan's illustrative STOP_WORDS/commonWords example against verified TS source**
- **Found during:** Task 1 (writing the RED test for `extractSearchTerms`)
- **Issue:** The plan's `<behavior>` block asserted `"getUserName by id"` → `"get"` is dropped because "get is in STOP_WORDS". Reading the live TS dist source (`search/query-utils.js:102-120`) shows `"get"` is NOT in `STOP_WORDS` (H2's filter) — it's in `commonWords` (H1's filter, `context/index.js:118-143`), a genuinely separate list. The plan conflated the two lists in its example, which is exactly the anti-pattern RESEARCH itself warns against.
- **Fix:** Wrote the RED/GREEN test to assert the verified behavior: `"get"` IS retained by `extractSearchTerms` (H2), while `commonWords` (H1) does filter it — exercised separately in `TestExtractSymbolsFromQuery`'s use of `"the and or"`. Ported both word lists exactly as they appear in the live source.
- **Files modified:** internal/query/tokenize.go, internal/query/tokenize_test.go
- **Verification:** `go test ./internal/query/ -run 'TestExtractSearchTerms|TestExtractSymbolsFromQuery' -count=1` — all subtests pass
- **Committed in:** 472cedf (RED), 436dd9e (GREEN)

**2. [Rule 1 - Bug] Implemented compound-identifier preservation in H2, which RESEARCH.md's excerpt omitted**
- **Found during:** Task 1
- **Issue:** RESEARCH.md §2's code excerpt for `extractSearchTerms` listed compound-preservation as numbered comments ("1. Preserve compound identifiers...", "2. Preserve snake_case compounds...") but did not show the actual regex/logic for those two steps — only the subsequent split-and-filter logic was shown verbatim. Porting only the shown code would silently drop TS's compound-preservation behavior (e.g., `"getUserName"` also emitting `"getusername"` as one token, not just its split parts).
- **Fix:** Read the live TS dist source directly and ported the omitted `compoundPattern`/`snakePattern` regex-based preservation pass verbatim (`search/query-utils.js:196-211`), confirmed via the RED test's `"HTTPServer"` and `"getUserName by id"` cases (which now assert `"httpserver"`/`"getusername"` are present, not just their split words).
- **Files modified:** internal/query/tokenize.go, internal/query/tokenize_test.go
- **Verification:** `go test ./internal/query/ -run TestExtractSearchTerms -count=1 -v` — subtest "multi-word query splits and preserves compounds" and "acronym/camel boundary split" both pass
- **Committed in:** 472cedf (RED), 436dd9e (GREEN)

---

**Total deviations:** 2 auto-fixed (both Rule 1 — corrections against the verified TS 1.3.1 dist source, made possible because the install is still present on this machine per D-01)
**Impact on plan:** Both corrections increase fidelity to the actual TS algorithm (the phase's core goal); no scope creep, no architectural change. Neither required user sign-off — both are within Rule 1's "code doesn't work as intended" scope once the ground-truth source was consulted.

## Issues Encountered
None beyond the two deviations above — both tokenizers matched their manually-traced expected output on first `go test` run (Go's `regexp` (RE2) leftmost-first semantics matched the hand-traced JS backtracking-regex behavior exactly for every pattern used).

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- `internal/query/tokenize.go` is ready for plan 07 (FTS gather channel, consumes `extractSearchTerms`) and plan 12 (named-symbol seeding, consumes `extractSymbolsFromQuery` for the +50 file score) to import directly — both are unexported package-level functions within `internal/query`, no further wiring needed to call them from other files in the same package.
- The CLI's `cobra.ExactArgs(1)` bug (multi-word `explore` queries need quoting today) is NOT fixed by this plan — that's `internal/cli/explore.go`'s variadic-args fix, tracked separately in RESEARCH §10 and belongs to a later plan in this phase's wave sequence.
- `getStemVariants()` remains unported — flagged for whichever plan wires `extractSearchTerms` into the actual FTS gather (plan 07) to decide whether stem expansion is needed for behavioral parity at that point.

---
*Phase: 01-behavioral-parity-explore-node*
*Completed: 2026-07-15*

## Self-Check: PASSED

All created files and commit hashes verified present.
