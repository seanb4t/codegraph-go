---
phase: 01-behavioral-parity-explore-node
plan: 01
subsystem: testing
tags: [golden-fixtures, mcp, jsonrpc, tree-sitter-go, capture-harness]

# Dependency graph
requires: []
provides:
  - "testdata/golden/corpus/synthetic-parity/ — a purpose-built Go corpus exercising D-03's 4 blind-spot cases (overloaded name, multi-word query, Test*-heavy weakly-connected cluster, structural-beats-lexical)"
  - "testdata/golden/capture.sh capture_behavioral() — reusable extension point for explore-multi/node-multi/explore-mcp/node-mcp fixtures on any corpus"
  - "testdata/golden/mcp-capture.mjs — reusable JSON-RPC stdio MCP client for driving TS's codegraph_explore/codegraph_node tools"
  - "Committed TS 1.3.1 golden fixtures (CLI + MCP) for multi-word explore and overloaded node, on all three corpora"
affects: [01-behavioral-parity-explore-node plan 02+, plan 17 (Go-side fixture regen, F5)]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "MCP-surface golden capture via a minimal hand-rolled JSON-RPC 2.0 stdio client (no SDK dependency) — initialize/notifications-initialized/tools-call handshake"
    - "capture_behavioral() as a second capture_repo-style function, callable immediately after an index is built, without re-indexing"

key-files:
  created:
    - testdata/golden/corpus/synthetic-parity/README.md
    - testdata/golden/corpus/synthetic-parity/src/go.mod
    - testdata/golden/corpus/synthetic-parity/src/accounts/manager.go
    - testdata/golden/corpus/synthetic-parity/src/accounts/validate.go
    - testdata/golden/corpus/synthetic-parity/src/orders/validate.go
    - testdata/golden/corpus/synthetic-parity/src/ledger/ledger.go
    - testdata/golden/corpus/synthetic-parity/src/recovery/recovery.go
    - testdata/golden/corpus/synthetic-parity/src/recovery/recovery_test.go
    - testdata/golden/mcp-capture.mjs
    - "testdata/golden/corpus/{weft-go,colbymchenry-codegraph,synthetic-parity}/{explore,node}-{multi,mcp}.json (12 files)"
  modified:
    - testdata/golden/capture.sh
    - testdata/golden/README.md
    - .gitignore
    - testdata/golden/ts-version.txt
    - "testdata/golden/corpus/colbymchenry-codegraph/{status,query,callers,callees,impact,explore,node}.json (upstream HEAD drift, expected)"

key-decisions:
  - "MCP node capture passes includeCode:true explicitly (TS's MCP tool defaults it false, but the CLI has no way to suppress full bodies) so the CLI and MCP captures test the identical semantic scenario instead of encoding an incidental TS-native default asymmetry unrelated to NODE-04/EXPL-05"
  - "synthetic-parity gets behavioral-only fixtures (no status/query/callers/callees/impact/baseline explore/node) — it exists solely to drive the new multi-def/multi-word/gate/RWR cases, not general tool-surface coverage"
  - "colbymchenry-codegraph's baseline fixtures were allowed to drift to the freshly re-cloned upstream HEAD as an accepted side effect of capture.sh's unconditional reindex-then-capture-all design (verified no test asserts exact node/edge counts before committing)"

patterns-established:
  - "Hand-rolled JSON-RPC 2.0 stdio MCP client pattern (mcp-capture.mjs) for any future TS-MCP-surface golden capture need — no new dependency, ~180 lines"

requirements-completed: [TEST-01]

coverage:
  - id: D1
    description: "synthetic-parity corpus exists, indexes cleanly with the current Go binary, and exposes an overloaded name (2+ nodes) plus a structural-beats-lexical edge-count asymmetry"
    requirement: "TEST-01"
    verification:
      - kind: other
        ref: "go run ./cmd/codegraph index --force -q testdata/golden/corpus/synthetic-parity/src && go run ./cmd/codegraph query Validate --json (2 defs) && go run ./cmd/codegraph callers/callees ReconcileLedger (4 edges) vs AccountBalanceHelper (0 edges)"
        status: pass
    human_judgment: false
  - id: D2
    description: "capture.sh is extended with explore-multi/node-multi/explore-mcp/node-mcp invocations for all three corpora without altering the existing --max-files 1 / -f baseline calls"
    requirement: "TEST-01"
    verification:
      - kind: other
        ref: "bash -n testdata/golden/capture.sh; grep count of explore-multi|node-multi|mcp-capture|explore-mcp|node-mcp >= 3"
        status: pass
    human_judgment: false
  - id: D3
    description: "TS 1.3.1 golden fixtures (CLI + MCP) captured and committed for the new behavioral cases on all three corpora"
    requirement: "TEST-01"
    verification:
      - kind: other
        ref: "go test ./testdata/golden/... (existing golden_test.go/golden_parity_test.go, unaffected); jq -e '.output' on all 12 new fixture files; jq -e '.output | contains(\"definitions named\")' on synthetic-parity/node-multi.json"
        status: pass
    human_judgment: false

# Metrics
duration: 55min
completed: 2026-07-15
status: complete
---

# Phase 1 Plan 1: Behavioral Fixture Harness + TS 1.3.1 Golden Capture Summary

**A purpose-built synthetic-parity Go corpus plus TS 1.3.1 CLI+MCP golden fixtures for multi-word explore and overloaded node, landed before any ranking algorithm code — D-06's fixtures-first guard against re-earning v0.1's blind spot.**

## Performance

- **Duration:** ~55 min
- **Started:** 2026-07-15T11:15:00Z (approx.)
- **Completed:** 2026-07-15T12:12:31Z
- **Tasks:** 3
- **Files modified:** 30 (8 new corpus files, 1 new mcp-capture.mjs, 12 new golden fixture JSONs, capture.sh + README.md + .gitignore + ts-version.txt modified, 7 colbymchenry-codegraph baseline fixtures re-captured)

## Accomplishments
- Built `testdata/golden/corpus/synthetic-parity/` — a small, self-contained Go source tree deliberately exercising all four D-03 blind-spot cases (overloaded `Validate`, multi-word `UserAccountManager`, Test*-heavy weakly-connected `TestAccountRecovery`/`recoverAccount`, structural-beats-lexical `AccountBalanceHelper`/`ReconcileLedger`) — verified mechanically via `query --json`/`callers`/`callees` against the current Go binary, no algorithm required
- Extended `capture.sh` with `capture_behavioral()`, adding `explore-multi`/`node-multi` CLI invocations (no `--max-files 1`/`-f`) for all three corpora, leaving the existing template-parity baseline invocations untouched
- Wrote `testdata/golden/mcp-capture.mjs`, a ~180-line hand-rolled JSON-RPC 2.0 stdio client that drives the live TS `codegraph serve --mcp` server (`CODEGRAPH_MCP_TOOLS=explore,node`) to capture `explore-mcp`/`node-mcp` fixtures — no existing in-repo precedent, no new dependency, version-gated on `codegraph --version` == `1.3.1`
- Ran the full extended capture against the live TS 1.3.1 install, producing and committing 12 new golden fixture files (4 fixtures × 3 corpora), plus updated `README.md` provenance documenting the D-03 case map, per-corpus query/symbol choices, and the deferred Go-side fixture regen (plan 17, F5)

## Task Commits

Each task was committed atomically:

1. **Task 1: Build the synthetic-parity corpus source tree** - `e8e3687` (test)
2. **Task 2: Extend capture.sh with behavioral + MCP-surface invocations** - `650d4ec` (feat)
3. **Task 3: Capture TS 1.3.1 goldens + update README provenance** - `d2aef04` (test)

_Note: Task 3's commit also swept in colbymchenry-codegraph's baseline fixture drift, an unavoidable side effect of capture.sh's unconditional reindex-then-capture-all design for that unpinned corpus — documented in the commit message and README._

## Files Created/Modified
- `testdata/golden/corpus/synthetic-parity/src/{accounts,orders,ledger,recovery}/*.go` - The D-03 case-map source tree (Validate ×2, UserAccountManager, TestAccountRecovery/recoverAccount, AccountBalanceHelper/ReconcileLedger)
- `testdata/golden/corpus/synthetic-parity/README.md` - Per-file D-03 case map + mechanical verification recipe
- `testdata/golden/mcp-capture.mjs` - JSON-RPC stdio MCP capture client (explore+node tools, includeCode:true parity fix, version gate)
- `testdata/golden/capture.sh` - `capture_behavioral()` + per-corpus wiring for `weft-go`/`colbymchenry-codegraph`/`synthetic-parity`
- `testdata/golden/README.md` - New corpus table row, behavioral-fixtures section, per-corpus query/symbol table, deferred-regen note
- `.gitignore` - Ignore synthetic-parity's local `.codegraph/` index data
- `testdata/golden/corpus/*/{explore,node}-{multi,mcp}.json` (12 files) - The captured TS 1.3.1 goldens
- `testdata/golden/corpus/colbymchenry-codegraph/{status,query,callers,callees,impact,explore,node}.json` - Re-captured against the (unpinned) upstream HEAD as an expected side effect

## Decisions Made
- MCP `codegraph_node` capture passes `includeCode: true` explicitly — TS's MCP tool schema defaults it `false` but the TS CLI has no flag to suppress full bodies, so without this fix the CLI and MCP fixtures would encode an incidental TS-native default asymmetry unrelated to what NODE-04/EXPL-05 actually need to port. Discovered by diffing the first capture run's `node-multi.json` vs `node-mcp.json` output.
- `synthetic-parity` intentionally gets ONLY the four new behavioral fixtures, not the full `status`/`query`/`callers`/`callees`/`impact`/baseline `explore`/`node` set `capture_repo` produces for the two real-world corpora — it exists purely to drive the new cases.
- Chose real symbols/queries in `weft-go` (`Run`, 10 defs / `"epic worktree"`, 4 files) and `colbymchenry-codegraph` (`resolve`, 27 defs / `"generated file detection"`, 4 files) by querying each repo's live TS SQLite index directly, rather than guessing.

## Deviations from Plan

None — plan executed as written. The `includeCode:true` addition and the colbymchenry-codegraph baseline drift are both within Task 2/3's own scope (getting the MCP capture and the live-oracle capture right), not unplanned work outside the plan's boundary.

## Issues Encountered
- First MCP capture attempt failed with "isn't indexed with codegraph" because the synthetic-parity corpus's `.codegraph/` still held the Go binary's Pebble-format index from Task 1's verification step — resolved by re-running `codegraph index --force` (TS) before the MCP probe; harmless since `.codegraph/` is fully gitignored and disposable.
- `index --force` alone fails ("not initialized") on a never-initialized path — the correct bootstrap is `codegraph init` (creates + indexes) followed by `index --force` for re-runs; `capture.sh`'s synthetic-parity section does both (`init ... || true` then `index --force`) for idempotency across repeated runs.

## User Setup Required

None - no external service configuration required. (The plan's `user_setup` block asked only to confirm TS 1.3.1 was on PATH before capture — confirmed via `codegraph --version` → `1.3.1` at the start of execution.)

## Next Phase Readiness
- The behavioral oracle (synthetic-parity corpus + TS 1.3.1 CLI+MCP goldens for multi-word explore and overloaded node) is committed and ready for plan 02+'s D-09 edge-kind expansion and EXPL-02's RWR port to diff against.
- Go-side EXPECTED fixtures are explicitly NOT regenerated here — deferred to plan 17 (F5) per the phase's ordering constraint (F1→F3→F4→F5), after the edge-kind expansion and RWR pipeline land.
- No blockers. `go test ./testdata/golden/...` and `go build ./...` both green after this plan's changes.

---
*Phase: 01-behavioral-parity-explore-node*
*Completed: 2026-07-15*

## Self-Check: PASSED

- FOUND: testdata/golden/corpus/synthetic-parity/README.md
- FOUND: testdata/golden/mcp-capture.mjs
- FOUND: testdata/golden/corpus/synthetic-parity/node-multi.json
- FOUND commit: e8e3687
- FOUND commit: 650d4ec
- FOUND commit: d2aef04
