---
phase: 01-foundation-storage-schema-parser-strategy
plan: 04
subsystem: testing
tags: [golden-fixtures, sqlite, parity-testing, migration, codegraph-cli, jq]

# Dependency graph
requires:
  - phase: 01-foundation-storage-schema-parser-strategy (01-01)
    provides: go.mod baseline, project scaffolding
provides:
  - "testdata/golden/ fixtures: TS CodeGraph v1.3.1 SQLite schema DDL + representative dump, golden JSON tool-output corpus, version stamp, and a smoke test"
  - "Reproducible capture.sh harness for re-capturing fixtures against the live TS CLI"
affects: [phase-03-mcp-parity, phase-04-sync, phase-07-migration-reader]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Determinism-stripping via jq walk() for volatile JSON fields (score, *_at/*At, dbSizeBytes) and sed for volatile SQL dump timestamps"
    - "Golden fixture smoke tests using stdlib only (testing/os/encoding-json/path-filepath), no test framework dependency"

key-files:
  created:
    - testdata/golden/capture.sh
    - testdata/golden/ts-version.txt
    - testdata/golden/ts-schema.sql
    - testdata/golden/ts-schema.dump.sql
    - testdata/golden/README.md
    - testdata/golden/golden_test.go
    - testdata/golden/corpus/weft-go/*.json
    - testdata/golden/corpus/colbymchenry-codegraph/*.json
  modified: []

key-decisions:
  - "Corpus repos: seanb4t/weft (public, compact, 84-file mostly-Go repo already available locally) for the Go corpus, and colbymchenry/codegraph (cloned to a throwaway temp dir at capture time, not committed as source) for the multi-language TS corpus, per D-06a"
  - "explore/node have no --json flag on the TS CLI; their raw markdown text output is wrapped in a {command, output} JSON envelope so every corpus fixture is uniformly JSON"
  - "Extended the volatile-field strip list beyond RESEARCH's explicit Pitfall-1 fields (score, *_at/*At) to also cover dbSizeBytes (WAL/fragmentation-dependent) and projectPath/indexPath (machine-local absolute paths, normalized to <CORPUS_PATH> rather than deleted) after empirically observing these also break byte-for-byte reproducibility"
  - "SQL dump captures a representative sample (LIMIT 5 rows per table via sqlite3 '.mode insert') rather than a full data dump, to keep the fixture small and git-friendly while still exercising every table's column shapes"
  - "Schema DDL captured once from the Go corpus (weft) after a forced reindex, since DDL doesn't vary by corpus and Go is Phase 2's first parser-target language"

requirements-completed: [MCP-04]

coverage:
  - id: D1
    description: "TS .codegraph/ SQLite schema DDL (ts-schema.sql, ts-schema.dump.sql) captured from a live, force-reindexed v1.3.1 index, byte-for-byte reproducible"
    requirement: "MCP-04"
    verification:
      - kind: other
        ref: "testdata/golden/capture.sh (run twice back-to-back; diff -rq showed zero differences across ts-schema.sql, ts-schema.dump.sql, and all corpus/*.json fixtures)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Golden JSON snapshots of codegraph_explore + companion tools (query/callers/callees/impact/node/status) on a pinned two-repo corpus, determinism-stripped"
    requirement: "MCP-04"
    verification:
      - kind: unit
        ref: "testdata/golden/golden_test.go#TestGoldenFixturesExist"
        status: pass
    human_judgment: false
  - id: D3
    description: "TestGoldenFixturesExist smoke test guards against a future re-capture forgetting to strip volatile fields"
    verification:
      - kind: unit
        ref: "testdata/golden/golden_test.go#TestGoldenFixturesExist/corpus_JSON_fixtures_are_stripped_of_volatile_fields (manually verified it fails when a score field is reintroduced, then confirmed clean pass after restore)"
        status: pass
    human_judgment: false

# Metrics
duration: ~25min
completed: 2026-07-10
status: complete
---

# Phase 01 Plan 04: TS CodeGraph v1.3.1 Golden Fixtures Summary

**Captured the TS CodeGraph v1.3.1 `.codegraph/` SQLite schema DDL and a golden JSON corpus (status/query/callers/callees/impact/explore/node) from a two-repo pinned corpus (weft, colbymchenry/codegraph), with all non-deterministic fields stripped and verified byte-for-byte reproducible across two capture runs.**

## Performance

- **Duration:** ~25 min
- **Completed:** 2026-07-10
- **Tasks:** 2/2
- **Files modified:** 19 created (1 script, 1 README, 2 SQL, 1 version stamp, 1 Go test, 14 JSON fixtures)

## Accomplishments
- `capture.sh`: a reproducible bash harness (`set -euo pipefail`) that force-reindexes both corpus repos, captures TS schema DDL + a representative sample SQL dump, and captures 7 golden JSON tool outputs per repo — all piped through a jq-based determinism-stripping filter
- Empirically verified byte-for-byte reproducibility: ran the full capture twice back-to-back (each with a fresh `codegraph index --force`) and `diff -rq` showed zero differences across every fixture file, JSON and SQL
- `TestGoldenFixturesExist` smoke test (stdlib-only) asserts fixture existence/non-emptiness and recursively scans every corpus JSON fixture's keys for reintroduced volatile fields — manually confirmed it fails loudly when a `score` field is injected, then confirmed clean pass after restoring
- README documents exactly which fields are volatile and why (score, `*_at`/`*At`, `dbSizeBytes`, `projectPath`/`indexPath`), the corpus provenance (pinned commits for both repos), and the historical edge-dedup bug #1034 with its implication for Phase 2's edge-identity key design

## Task Commits

Each task was committed atomically:

1. **Task 1: Author the capture harness and capture TS schema DDL + golden corpus** - `12f26e1` (feat)
2. **Task 2: Add the golden-fixtures smoke test** - `c3cf524` (test)

_No plan-metadata commit issued separately; STATE/ROADMAP/REQUIREMENTS updates are bundled into the final docs commit below._

## Files Created/Modified
- `testdata/golden/capture.sh` - reproducible capture harness (force-reindex, capture, determinism-strip)
- `testdata/golden/README.md` - documents volatile-field stripping, corpus provenance, and the #1034 edge-dedup note
- `testdata/golden/ts-version.txt` - `codegraph_version=1.3.1` + UTC capture date
- `testdata/golden/ts-schema.sql` - full `.schema` DDL from the weft (Go) corpus index
- `testdata/golden/ts-schema.dump.sql` - representative 5-row sample per table (`nodes`, `edges`, `files`, `schema_versions`), timestamp-normalized
- `testdata/golden/corpus/weft-go/{status,query,callers,callees,impact,explore,node}.json` - golden tool outputs for the Go corpus
- `testdata/golden/corpus/colbymchenry-codegraph/{status,query,callers,callees,impact,explore,node}.json` - golden tool outputs for the multi-language TS corpus
- `testdata/golden/golden_test.go` - `TestGoldenFixturesExist` smoke test

## Decisions Made
- Used `seanb4t/weft` (verified public via `gh repo view`) as the Go corpus repo instead of a purpose-built minimal fixture, since it was already available, compact (84 files), and matched D-06a's criteria exactly
- Cloned `colbymchenry/codegraph` into a throwaway `mktemp -d` directory rather than committing its source tree — only the captured JSON tool outputs are committed, keeping the corpus small per the plan's "keep it small enough to live in git" constraint
- Picked stable, existing symbols per repo (`mergeStyle` in weft, `searchNodes` in colbymchenry/codegraph) for the callers/callees/impact/node captures rather than synthetic ones, so the fixtures reflect real call-graph shapes
- Extended the strip list beyond RESEARCH's explicitly named `score`/`*_at`/`*At` fields to also strip `dbSizeBytes` and normalize `projectPath`/`indexPath` — discovered during implementation that these also vary/leak machine-local state, which would have broken the "reproduces byte-for-byte" requirement this plan exists to satisfy (Rule 2: auto-added missing correctness/reproducibility handling)
- Fixed an initial regex bug in the SQL-dump timestamp stripper: `modified_at`/`indexed_at` columns store epoch-ms as REAL with a fractional component (e.g. `1783108606938.7051`), not plain INTEGER as the schema DDL declares — the strip regex was extended to consume the optional decimal suffix

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing critical functionality] Stripped additional non-deterministic fields beyond RESEARCH's Pitfall 1 list**
- **Found during:** Task 1 (building capture.sh, verifying reproducibility)
- **Issue:** RESEARCH.md's Pitfall 1 named `score` and `*_at`/`*At` timestamps as volatile, but `status.json` also carries `dbSizeBytes` (WAL/fragmentation-dependent, not stable across identical reindexes) and machine-local absolute paths (`projectPath`, `indexPath`) that would leak this capture machine's filesystem layout into a committed fixture
- **Fix:** Extended the jq strip filter to delete `dbSizeBytes` and normalize `projectPath`/`indexPath` to the literal `<CORPUS_PATH>`; documented both in README's volatile-fields table
- **Files modified:** testdata/golden/capture.sh, testdata/golden/README.md
- **Verification:** Two full back-to-back capture runs (each with a fresh `index --force`) diffed byte-for-byte identical across all fixtures
- **Committed in:** `12f26e1` (Task 1 commit)

**2. [Rule 1 - Bug] Fixed SQL-dump timestamp regex to consume fractional epoch-ms**
- **Found during:** Task 1 (first capture run inspection)
- **Issue:** The initial `sed -E 's/[0-9]{13}/<EPOCH_MS>/g'` regex only stripped the integer portion of `modified_at`/`indexed_at` values that SQLite stored as REAL (e.g. `1783108606938.7051`), leaving a `.7051`-style fractional remainder that would differ on every recapture
- **Fix:** Extended the regex to `[0-9]{13}(\.[0-9]+)?` so the optional fractional suffix is consumed too
- **Files modified:** testdata/golden/capture.sh
- **Verification:** Re-ran capture; `grep -o '[0-9]{13}' ts-schema.dump.sql` returned nothing, and a second capture run diffed identical
- **Committed in:** `12f26e1` (Task 1 commit)

---

**Total deviations:** 2 auto-fixed (1 missing critical/reproducibility, 1 bug)
**Impact on plan:** Both fixes were necessary to satisfy the plan's own core requirement — byte-for-byte reproducible fixtures. No scope creep; no architectural changes.

## Issues Encountered
- The `codegraph explore`/`codegraph node` CLI subcommands have no `--json` flag (only `query`, `status`, `callers`, `callees`, `impact` do) — resolved by wrapping their markdown text output in a `{command, output}` JSON envelope, documented in README, so every corpus fixture stays uniformly JSON as the plan's acceptance criteria require.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- `testdata/golden/` is the frozen parity oracle Phase 3 (MCP-04), Phase 4, and Phase 7 will diff against
- Phase 2's edge-identity key design has an explicit, documented open question to resolve (line/col inclusion for multi-call-site edges), informed by the captured `idx_edges_identity` unique index and the #1034 history
- If the live TS CLI becomes unavailable before a future re-capture is needed, the already-committed fixtures remain valid and should not be hand-edited

---
*Phase: 01-foundation-storage-schema-parser-strategy*
*Completed: 2026-07-10*

## Self-Check: PASSED

All claimed files exist (capture.sh, README.md, ts-version.txt, ts-schema.sql, ts-schema.dump.sql, golden_test.go, and both corpus status.json fixtures) and both commit hashes (`12f26e1`, `c3cf524`) are present in git log.
