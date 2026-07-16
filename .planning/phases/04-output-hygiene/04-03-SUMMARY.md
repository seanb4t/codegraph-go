---
phase: 04-output-hygiene
plan: 03
subsystem: testing
tags: [integration-test, mcp, json-rpc, stdout, pebble, hygiene]

requires:
  - phase: 04-output-hygiene
    provides: "04-01's quietLogger wiring at graphstore.Open (HYG-01 production behavior this plan's D-09 case observes) and 04-02's structural stdout-confinement archtest (HYG-02's build-time half this plan's D-06a case complements end-to-end)"
provides:
  - "TestServeMCPStdoutIsPureJSONRPC: a raw-stdio harness proving every stdout byte of a real serve --mcp session (startup reconcile + a store-opening tools/call) is a valid JSON-RPC frame, independent of mcp-go's tolerant client parser"
  - "TestSyncStderrNoPebbleNoise: a real sync command driven end-to-end confirming HYG-01's quietLogger actually silences Pebble noise shapes on stderr, without asserting stderr emptiness"
affects: [06-charm-tui]

tech-stack:
  added: []
  patterns:
    - "Raw-stdio JSON-RPC assertion: own cmd.StdoutPipe() directly via bufio.Scanner + json.Unmarshal per line, never layered on mcp-go's client (whose stdio transport silently skips non-frame lines)"
    - "Absence-of-noise-shape assertion (not stderr-emptiness) for CLI-side library-noise regression checks"

key-files:
  created:
    - test/integration/mcp_stdout_purity_test.go
    - test/integration/sync_noise_test.go
  modified: []

key-decisions:
  - "CODEGRAPH_MCP_TOOLS=status env var set on the spawned serve --mcp process so the tools/call to codegraph_status actually registers and opens the store (companion tools are allowlist-gated; codegraph_explore alone is unconditionally visible)"
  - "CODEGRAPH_NO_WATCH=1 set on the spawned process to keep the purity case deterministic — watcher diagnostics already go to stderr only (WATCH-02, off the handshake path), so this only removes timing nondeterminism, never anything the assertion depends on"
  - "No notifications/initialized round trip needed before the tools/call: mcp-go's stdio server marks the session Initialized() synchronously inside handleInitialize (server/stdio.go), before the initialize response is even written back — verified by reading the pinned mcp-go@v0.56.0 source directly"
  - "sync (not status) drives the D-09 noise-absence check, per RESEARCH's Open Question #1 recommendation — exercises strictly more of Pebble's Infof surface (flush + possible compaction) than a bare read-only status call"

patterns-established:
  - "The narrowest possible hand-rolled JSON-RPC reader (cmd.StdoutPipe + bufio.Scanner + json.Unmarshal per line) is the correct tool for the one case where mcp-go's client's designed-in tolerance (silently skipping malformed lines) is disqualifying — every other integration test keeps using newServeClient/mcpclient.Client unmodified"

requirements-completed: [HYG-01, HYG-02]

coverage:
  - id: D1
    description: "Every stdout line from a real serve --mcp session (startup reconcile store-open + a tools/call to codegraph_status, a second store-open) parses as a JSON-RPC frame; the assertion is provably able to fail on a real violation"
    requirement: "HYG-02"
    verification:
      - kind: integration
        ref: "test/integration/mcp_stdout_purity_test.go#TestServeMCPStdoutIsPureJSONRPC"
        status: pass
      - kind: manual_procedural
        ref: "manually injected fmt.Println(\"MUTATION-PROOF-TEST-POLLUTION\") into internal/cli/serve.go's RunE, re-ran TestServeMCPStdoutIsPureJSONRPC with -count=1, confirmed it failed quoting the exact injected bytes (non-JSON-RPC byte on stdout: \"MUTATION-PROOF-TEST-POLLUTION\"), then reverted (git diff confirms zero residual change)"
        status: pass
    human_judgment: false
  - id: D2
    description: "A real sync command driven end-to-end through the spawned binary emits no Pebble-shaped noise ([JOB , WAL , compaction, pickAuto) on stderr, confirming HYG-01's quietLogger effect on the CLI path — without asserting stderr emptiness"
    requirement: "HYG-01"
    verification:
      - kind: integration
        ref: "test/integration/sync_noise_test.go#TestSyncStderrNoPebbleNoise"
        status: pass
    human_judgment: false

duration: 8min
completed: 2026-07-16
status: complete
---

# Phase 4 Plan 3: MCP Stdout Purity & Sync Noise-Absence (End-to-End) Summary

**A raw-stdio harness proves every real `serve --mcp` stdout byte is a JSON-RPC frame (independent of mcp-go's tolerant client), and a real `sync` run confirms HYG-01's quietLogger silences Pebble noise on stderr — closing both requirements' end-to-end half through the existing Phase-3 subprocess harness**

## Performance

- **Duration:** 8 min
- **Started:** 2026-07-16T21:18:59Z
- **Completed:** 2026-07-16T21:25:49Z
- **Tasks:** 2
- **Files modified:** 2 (both created)

## Accomplishments
- `test/integration/mcp_stdout_purity_test.go`: `TestServeMCPStdoutIsPureJSONRPC` spawns the real binary as `serve --mcp` via plain `exec.Command`, owns `cmd.StdinPipe()`/`cmd.StdoutPipe()` directly, hand-frames an `initialize` request (id 1) and a `tools/call` request naming `codegraph_status` (id 2, `CODEGRAPH_MCP_TOOLS=status` allowlisted), and validates EVERY stdout line with `json.Unmarshal` + non-empty `jsonrpc` field via a bounded (30s deadline, goroutine-fed channel) scan loop. Deliberately does NOT use `newServeClient`/`mcpclient.Client` — mcp-go@v0.56.0's stdio transport silently skips any stdout line that fails to parse, which would make a client-layered assertion structurally unable to fail (RESEARCH Pitfall 1).
- `test/integration/sync_noise_test.go`: `TestSyncStderrNoPebbleNoise` drives real `init` then `sync` through `runBinary` and asserts stderr contains none of `[JOB `, `WAL `, `compaction`, `pickAuto` — absence-of-substring on noise shapes only, never emptiness (D-09).
- Manually proved the frame-purity test's fail-capability: injected a stray `fmt.Println` into `internal/cli/serve.go`'s `RunE` (between the startup reconcile and watcher-start), re-ran the test with `-count=1`, confirmed it failed with `non-JSON-RPC byte on stdout: "MUTATION-PROOF-TEST-POLLUTION"`, then reverted (`git diff` confirms zero residual change to `serve.go`).

## Task Commits

Each task was committed atomically:

1. **Task 1: Raw-stdio JSON-RPC frame-purity harness (D-06a)** - `0d9aad3` (test)
2. **Task 2: CLI-side Pebble-noise-absence on stderr (D-09)** - `93488be` (test)

**Plan metadata:** (this commit)

## Files Created/Modified
- `test/integration/mcp_stdout_purity_test.go` - raw-stdio `TestServeMCPStdoutIsPureJSONRPC`, the D-06a end-to-end frame-purity harness
- `test/integration/sync_noise_test.go` - `TestSyncStderrNoPebbleNoise`, the D-09 CLI-noise-absence behavioral check

## Decisions Made
- `CODEGRAPH_MCP_TOOLS=status` set on the spawned process so the `tools/call` actually registers and opens the store (companion tools are allowlist-gated per MCP-02/D-08a; `codegraph_explore` alone needs no allowlist entry)
- `CODEGRAPH_NO_WATCH=1` set for determinism only — watcher diagnostics already go to stderr off the handshake path (WATCH-02), so this cannot affect the purity assertion, only startup timing
- Confirmed (by reading the pinned `mcp-go@v0.56.0` source directly) that no `notifications/initialized` round trip is required before the `tools/call`: the stdio session is marked `Initialized()` synchronously inside `handleInitialize`, before the `initialize` response is even written
- `sync` (not `status`) drives D-09's check per RESEARCH's own Open Question #1 recommendation — strictly more Pebble Infof surface (flush + possible compaction) than a bare read

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- HYG-01 and HYG-02 are both now fully closed end-to-end: structural guards (Plans 01/02) plus this plan's runtime proofs, all riding the existing `test/integration/` harness and existing named CI step (D-10, no new CI machinery)
- `go test ./... && go test ./testdata/golden/... && go test ./test/integration/...` all green
- Phase 4 (Output Hygiene) is complete — ready for `/gsd-verify-work 4` and Phase 5 planning

## Self-Check: PASSED

All claimed files and commit hashes verified present.

---
*Phase: 04-output-hygiene*
*Completed: 2026-07-16*
