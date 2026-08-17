---
phase: 01-corpus-selection-by-measurement
plan: 03
subsystem: mcp,testdata
tags: [go, golden, wire-oracle, re-freeze, sparsity, D-05, FIXT-01]
requires:
  - "01-02 — query.RenderStatusMarkdown emits the Edges by Kind: section (the diff that broke the golden)"
provides:
  - "Re-frozen call-status.golden — call-status wire-oracle transcript carries the Edges by Kind block"
  - "TestStatusMarkdownStaysSparse — D-05 asserted through the real BuildServer call path, not the renderer in isolation"
  - "edgesByKind and filesByLanguage key-presence assertions in golden-parity status subtest"
affects: [01-04, 01-05, 01-06]

actuals:
  tokens: 1790
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "Capture-to-temp-then-move for golden re-freeze: never redirect straight onto the golden path, because a capture failure truncates the committed file before it produces bytes, leaving a zero-byte oracle that statFrozenTranscript then refuses for a second, misleading reason. Write to a temp path, assert non-empty and marker-bearing, then move."
    - "Semantic sparse assertion: every rendered Edges by Kind bullet's count is strictly greater than zero, and the rendered kind set equals EXACTLY the fixture's own positive-count edge-kind set obtained by calling Engine.Status directly — never an incidental row-count bound against len(RankEdges)."

key-files:
  created: []
  modified:
    - testdata/wireoracle/transcripts/call-status.golden
    - internal/mcp/markdown_test.go
    - testdata/golden/golden_parity_test.go

key-decisions:
  - "MCP sparsity is asserted through the real BuildServer call path, not through query.RenderStatusMarkdown in isolation: the handler could theoretically densify before rendering (internal/cli/status.go is the ONLY call site that calls DenseEdgesByKind), and a renderer-level test would not prove it never does. RED-proven: temporarily densifying the handler turned TestStatusMarkdownStaysSparse red (5 zero-valued rows flagged), then reverted byte-clean."
  - "edgesByKind and filesByLanguage are NOT added to the frozen-fixture golden key loop — that loop asserts against the committed third-party fixture, which predates both keys and does not carry them. Same asymmetry dbSizeBytes's comment already documents."
  - "filesByLanguage un-suppression: a comment beside the new assertions records that it was previously suppressed to match an output shape the project no longer owes anyone, and the Compatibility constraint was formally retired 2026-08-13 — so a future reader sees the intent rather than reading the un-suppression as an accident and reverting it."

patterns-established:
  - "Positive edge-kind presence assertions via both the MCP tool path (TestStatusMarkdownStaysSparse) and the golden-parity --json path (TestGoldenParity/status) — the same data key asserted through both surfaces a consumer actually uses."

requirements-completed: [FIXT-01]

coverage:
  - id: D4-re-freeze
    description: "call-status.golden re-frozen from live output through the existing test/wireoracle/cmd/wireoracle capture entrypoint, carrying the new Edges by Kind: block — exactly one transcript changed, attributable to the named cause (01-02's new section)"
    requirement: FIXT-01
    verification:
      - kind: e2e
        ref: "go test -count=1 ./test/wireoracle/... passes; TestFrozenTranscriptsMatch/call-status passes"
        status: pass
      - kind: source
        ref: "git diff --name-only ed0e5df -- testdata/wireoracle/transcripts/ | wc -l == 1"
        status: pass
      - kind: source
        ref: "git diff --quiet ed0e5df -- testdata/wireoracle/transcripts/resources-read-status.golden (unchanged)"
        status: pass
      - kind: source
        ref: "git diff --name-only ed0e5df -- test/wireoracle/ is empty (no new tooling)"
        status: pass
      - kind: source
        ref: "test -s testdata/wireoracle/transcripts/call-status.golden (non-empty)"
        status: pass
      - kind: source
        ref: "grep -c 'Edges by Kind' testdata/wireoracle/transcripts/call-status.golden == 1"
        status: pass
    human_judgment: false

  - id: D4-mcp-sparsity
    description: "D-05: codegraph_status with empty args (the argument-less MCP surface) returns markdown whose Edges by Kind block is sparse — every row's count is strictly greater than zero, and the rendered kind set equals exactly the fixture's own positive-count set from Engine.Status"
    requirement: FIXT-01
    verification:
      - kind: e2e
        ref: "go test -count=1 ./internal/mcp/... -run TestStatusMarkdownStaysSparse passes (real BuildServer call path)"
        status: pass
      - kind: source
        ref: "RED-proven: temporarily densifying the handler turns the test red (5 zero rows flagged), then reverted byte-clean"
        status: pass
    human_judgment: false

  - id: D4-json-key-presence
    description: "edgesByKind and filesByLanguage are both present in our own status --json output via golden-parity status subtest, with edgesByKind values all positive integers and filesByLanguage key set matching Languages's members"
    requirement: FIXT-01
    verification:
      - kind: e2e
        ref: "go test -count=1 ./testdata/golden/... -run TestGoldenParity/status passes (real weft corpus)"
        status: pass
    human_judgment: false

metrics:
  duration: ~6h (includes session API limit interruption)
  completed: 2026-08-14
  status: complete
---

# Phase 1 Plan 3: Wire-Oracle Re-Freeze & Surface Assertions Summary

**Re-frozen `call-status.golden` through the existing capture entrypoint to include the new `Edges by Kind:` section, added a semantic MCP sparsity assertion driven through the real `BuildServer` tool path (RED-proven against a densified handler), and landed `edgesByKind`/`filesByLanguage` key-presence assertions in the golden-parity `status` subtest against the real weft corpus.**

## Performance

- **Duration:** ~6 hours (includes interruption for session API limit)
- **Tasks:** 2
- **Commits:** 2
- **Files modified:** 3

## Accomplishments

- **Task 1 — Re-freeze:** Captured the `call-status` scenario through `test/wireoracle/cmd/wireoracle` (the existing, purpose-built entrypoint) against a freshly built binary. Captured to a temp file first, verified non-empty and marker-bearing, confirmed the diff was confined to exactly the `Edges by Kind` bullet block insertion, then moved into place. Exactly one transcript changed (`call-status.golden`); `resources-read-status.golden` untouched. No new tooling introduced. Commit message names the single cause and carries no CI skip directive.

- **Task 2 — MCP sparsity assertion + golden-parity key presence:**
  - `TestStatusMarkdownStaysSparse` drives `codegraph_status` through a real `BuildServer` with empty args, parses the `Edges by Kind:` bullet block, and asserts every count is strictly greater than zero (sparse). The rendered kind set is compared against the fixture's own positive-count set from `Engine.Status`, proving in both directions that no absent kind was synthesized and no present kind was dropped. RED-proven: temporarily densifying the handler produced 5 zero-row failures, then reverted byte-clean.
  - `parseEdgesByKindMarkdown` and `positiveEdgeCounts` helpers extracted for reusability.
  - Golden-parity `status` subtest asserts both `edgesByKind` and `filesByLanguage` are present in our own `--json` output, with `edgesByKind` values all positive integers and `filesByLanguage`'s key set matching `Languages`'s members. A comment records the un-suppression rationale so future readers do not read it as an accidental revert.
  - `internal/mcp/tools.go` confirmed untouched by `git diff --name-only`.

## Task Commits

| # | Task | Commit | Message |
|---|------|--------|---------|
| 1 | Re-freeze call-status.golden | `fc026fc` | `test(01-03): re-freeze call-status.golden for edges-by-kind section` |
| 2 | MCP sparsity assertion + golden-parity key presence | `a3ceb2d` | `test(01-03): add MCP sparsity assertion and golden-parity key presence assertions` |

## Files Created/Modified

- `testdata/wireoracle/transcripts/call-status.golden` — re-frozen to carry the `**Edges by Kind:**` bullet block
- `internal/mcp/markdown_test.go` — `TestStatusMarkdownStaysSparse` (+ `parseEdgesByKindMarkdown`, `positiveEdgeCounts` helpers)
- `testdata/golden/golden_parity_test.go` — `edgesByKind` and `filesByLanguage` presence assertions in the `status` subtest

## Decisions Made

- **Sparsity asserted through the real call path, not the renderer:** `TestStatusMarkdownStaysSparse` drives `codegraph_status` through a real `BuildServer` + `callTool`, because the handler could theoretically densify before rendering and a renderer-level test would not catch it. The RED proof confirmed this concern was valid — the handler is genuinely the only enforcement point for D-05.
- **Semantic assertion, not incidental:** The sparse invariant is defined as "every rendered row's count > 0 and the rendered kind set equals the fixture's own positive-count set" — never as "the row count is fewer than `len(RankEdges)`", which is an incidental property of today's small fixture and would break if the fixture grew to cover all 9 kinds.
- **Golden-parity keys in our own output, not the frozen fixture:** `edgesByKind` and `filesByLanguage` are asserted against our own `--json` decoded map, never added to the frozen-fixture key loop (which predates both keys).
- **Un-suppression rationale recorded in place:** The comment beside the `filesByLanguage` assertions names the retired Compatibility constraint and the date (2026-08-13) so the un-suppression is read as intentional, not accidental.

## Deviations from Plan

None — plan executed exactly as written.

## Issues Encountered

- **Pre-existing daemon flake in full suite:** `TestDaemonFlushLockRequeueGivesUpPerEpisode` (~250s, documented pre-existing flake per STATE.md "Daemon extreme-load tail") failed under full-suite parallel load but passed in isolation — confirmed unrelated to this plan's changes. Only affected packages (`./internal/mcp/...`, `./testdata/golden/...`, `./test/wireoracle/...`) all passed cleanly.
- **Session API limit interruption:** A mid-execution session limit pause extended plan wall-clock to ~6h; actual working time was substantially less.
- **macOS sandbox:** The `mv` for the re-frozen golden produced an ownership warning (`Operation not permitted`) but the file was successfully written and verified non-empty.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- `call-status.golden` re-frozen: `TestFrozenTranscriptsMatch/call-status` is GREEN again
- D-05's MCP sparsity asserted and RED-proven through the real tool path
- `edgesByKind` and `filesByLanguage` key presence asserted against real weft corpus
- Ready downstream plans to build on these assertions (01-04, 01-05, 01-06)
- No blockers

## Self-Check: PASSED

- FOUND: `testdata/wireoracle/transcripts/call-status.golden` (re-frozen, 1142 bytes, contains "Edges by Kind")
- FOUND: `internal/mcp/markdown_test.go` (modified, contains `TestStatusMarkdownStaysSparse`)
- FOUND: `testdata/golden/golden_parity_test.go` (modified, contains `edgesByKind` and `filesByLanguage` assertions)
- FOUND: commit `fc026fc` (Task 1)
- FOUND: commit `a3ceb2d` (Task 2)
- PASSED: `go test -count=1 ./internal/mcp/...`
- PASSED: `go test -count=1 ./testdata/golden/...` (status subtest ran against real weft corpus)
- PASSED: `go test -count=1 ./test/wireoracle/...` (call-status subtest now green)
- CONFIRMED: Exactly one transcript changed (`call-status.golden`)
- CONFIRMED: `resources-read-status.golden` untouched
- CONFIRMED: No new files under `test/wireoracle/`
- CONFIRMED: `internal/mcp/tools.go` untouched
- CONFIRMED: No CI skip directive in any commit message

---
*Phase: 01-corpus-selection-by-measurement*
*Plan: 03 — Wire-Oracle Re-Freeze & Surface Assertions*
*Completed: 2026-08-14*