---
phase: 01-foundation-storage-schema-parser-strategy
plan: 05
subsystem: database
tags: [pebble, keyspace, key-encoding, security, go]

# Dependency graph
requires:
  - phase: 01-foundation-storage-schema-parser-strategy (plan 01)
    provides: internal/graphstore package skeleton (doc.go), pebble/v2 pinned in go.mod
provides:
  - internal/graphstore/keys.go — typed key encoders for meta/n/e/f namespaces + reserved a/ (ARCH-01)
  - Delimiter-safe segment encoding (varint length-prefix) neutralizing key-injection (V5/T-01-02)
  - edgeSrcPrefix + rangeUpperBound generic range-scan/range-delete boundary helpers
  - fileSubgraphPrefix backing Plan 01-06's DeleteFileSubgraph range-delete
affects: [01-06 (pebbleStore implementation consumes these encoders exclusively — no raw []byte keys elsewhere)]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Length-prefixed (varint) segment encoding instead of literal-separator concatenation for all Pebble key construction — the single mitigation for key-injection across the whole keyspace"
    - "Generic rangeUpperBound(prefix) byte-successor helper reused by both edge range-scans and file range-deletes, rather than one-off boundary math per namespace"

key-files:
  created:
    - internal/graphstore/keys.go
    - internal/graphstore/keyenc_test.go
  modified: []

key-decisions:
  - "fileSubgraphPrefix scopes only the file's own f/ record in v1 (node/edge records are not yet keyed by owning file); extending one range-delete call to also prune a file's node/edge records is deferred to Plan 01-06's storage-layout design, not part of this key-encoder foundation"
  - "rangeUpperBound is a standalone, namespace-agnostic helper (not baked into each *Prefix function) so both edgeSrcPrefix range-scans and fileSubgraphPrefix range-deletes share one boundary implementation"
  - "Committed Task 1 (encoders) before Task 2 (tests) per the plan's explicit task order — this plan's frontmatter is type: execute, not type: tdd, so the plan-level RED-before-GREEN gate does not apply; Task 2's tests passed on first run because Task 1's length-prefix design is correct by construction, not because tests were skipped"

patterns-established:
  - "Pattern: no code outside keys.go may construct a raw []byte Pebble key — Plan 01-06's pebbleStore must call these builders exclusively"
  - "Pattern: edgeKey documents the v1 line/col-not-in-key decision inline so Phase 2 extractor design has a pointer back to Pitfall 2 / Open Question 3"

requirements-completed: [INDX-05]

coverage:
  - id: D1
    description: "Typed key encoders for meta/n/e/f namespaces plus a reserved a/ annotation namespace, with delimiter-safe (length-prefixed) segment encoding preventing raw '/'-concatenation of untrusted path/id"
    requirement: "INDX-05"
    verification:
      - kind: unit
        ref: "go build ./internal/graphstore/ && go vet ./internal/graphstore/"
        status: pass
    human_judgment: false
  - id: D2
    description: "Key-injection resistance proven: adversarial ids/paths containing '/', 0x00, 0xFF, unicode, and empty strings never collide or cross namespace boundaries"
    requirement: "INDX-05"
    verification:
      - kind: unit
        ref: "internal/graphstore/keyenc_test.go#TestKeyEncodingRejectsDelimiterInjection"
        status: pass
    human_judgment: false
  - id: D3
    description: "Edge keys sort by source so callers/callees/impact are a single contiguous range scan, including against naive-prefix-adjacent sources (e.g. 'alpha' vs 'a')"
    requirement: "INDX-05"
    verification:
      - kind: unit
        ref: "internal/graphstore/keyenc_test.go#TestEdgeKeyRangeScanOrdering"
        status: pass
    human_judgment: false
  - id: D4
    description: "File-subgraph range-delete window covers exactly one file's own record and excludes lexicographically adjacent files (foo vs foobar, bar.go vs foobar.go)"
    requirement: "INDX-05"
    verification:
      - kind: unit
        ref: "internal/graphstore/keyenc_test.go#TestFileSubgraphRangeDeleteBoundary"
        status: pass
    human_judgment: false

# Metrics
duration: 5min
completed: 2026-07-10
status: complete
---

# Phase 01 Plan 05: Typed Keyspace & Key-Injection Guard Summary

**Typed Pebble keyspace (meta/n/e/f + reserved a/) with varint length-prefixed segment encoding that proves key-injection resistant across 11 adversarial inputs**

## Performance

- **Duration:** ~5 min
- **Started:** 2026-07-10T16:55Z
- **Completed:** 2026-07-10T16:56Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- `internal/graphstore/keys.go`: single-byte prefix constants for `m`/`n`/`e`/`f` plus a reserved `a` (post-v1 embeddings/community assignments, ARCH-01), with `nodeKey`, `edgeKey`, `edgeSrcPrefix`, `fileKey`, `fileSubgraphPrefix`, `metaKey`, and a generic `rangeUpperBound` byte-successor helper
- Delimiter-safe segment encoding: every variable segment is length-prefixed (varint) rather than concatenated with a literal `/`, so a crafted id/path containing `/`, `0x00`, or `0xFF` cannot forge a key boundary (Security V5 / T-01-02, Tampering)
- `edgeKey` preserves `src` as the primary segment so `edgeSrcPrefix(src)` isolates one source's outgoing edges as a contiguous byte range — proven correct even against naive-prefix-adjacent sources like `"a"` vs `"alpha"`
- `edgeKey`'s doc comment records the v1 decision that line/col are NOT part of the key (RESEARCH Pitfall 2 / Open Question 3), pointing at Phase 2 extractor design for revisiting call-site multiplicity
- Three test functions (`TestKeyEncodingRejectsDelimiterInjection`, `TestEdgeKeyRangeScanOrdering`, `TestFileSubgraphRangeDeleteBoundary`) covering injection resistance, range-scan ordering, and range-delete boundary correctness — all passing

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement typed key encoders with delimiter-safe segment encoding** - `19c4595` (feat)
2. **Task 2: Test key-injection resistance and range-scan/range-delete boundaries** - `e0997c5` (test)

**Plan metadata:** (this commit)

_Note: Task order follows the plan exactly (implementation, then its test suite) — this is a `type: execute` plan, not `type: tdd`, so the strict test-before-implementation gate does not apply here._

## Files Created/Modified
- `internal/graphstore/keys.go` - prefix constants, `appendSegment` (varint length-prefix), `nodeKey`, `edgeKey`, `edgeSrcPrefix`, `fileKey`, `fileSubgraphPrefix`, `metaKey`, `rangeUpperBound`
- `internal/graphstore/keyenc_test.go` - `TestKeyEncodingRejectsDelimiterInjection`, `TestEdgeKeyRangeScanOrdering`, `TestFileSubgraphRangeDeleteBoundary`

## Decisions Made
- `fileSubgraphPrefix` scopes only the file's own `f/` record in v1, since node/edge records aren't yet keyed by owning file — documented in-code as a Plan 01-06 storage-layout decision, not expanded here to avoid over-scoping this key-encoder foundation plan
- `rangeUpperBound` implemented once as a namespace-agnostic byte-successor helper (increment last non-`0xFF` byte, trim trailing `0xFF`s) and reused by both edge range-scans and file range-deletes, rather than duplicating boundary math per namespace

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None. Task 2's tests passed on the first `go test` run because Task 1's length-prefix encoding is correct by construction (not because verification was skipped) — confirmed by explicitly testing naive-prefix-adjacent inputs (`"foo"` vs `"foobar"`, `"a"` vs `"alpha"`, `"bar.go"` vs `"foobar.go"`) that would have collided under literal `/`-concatenation.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- `internal/graphstore/keys.go` is ready for Plan 01-06's `pebbleStore` to consume exclusively for all key construction (no raw `[]byte` keys elsewhere)
- The reserved `a/` annotation prefix is defined and documented, satisfying the ARCH-01 physical slot for post-v1 embeddings/community assignments without requiring a future migration
- No blockers for Plan 01-06

---
*Phase: 01-foundation-storage-schema-parser-strategy*
*Completed: 2026-07-10*

## Self-Check: PASSED

- FOUND: internal/graphstore/keys.go
- FOUND: internal/graphstore/keyenc_test.go
- FOUND: commit 19c4595 (feat: encoders)
- FOUND: commit e0997c5 (test: injection/boundary tests)
