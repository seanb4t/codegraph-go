---
phase: 03-watcher-on-mcp-default
plan: 05
subsystem: testing
tags: [integration-test, mcp-go, stdio, watch-policy, ci]

# Dependency graph
requires:
  - phase: 03-watcher-on-mcp-default
    provides: "test/integration/'s harness substrate (03-04): binPath, buildWorktreeFixture, newServeClient, runBinary; internal/cli/serve.go's default-on watcher (03-03) and internal/watch/policy.go's verbatim WATCH-03 reason strings (03-01)"
provides:
  - "TestDefaultWatchHandshakePrompt — WATCH-01/WATCH-02 proven on the real spawned binary: bare `serve --mcp` (no watch flag in argv) completes Initialize+ListTools within a bounded context and advertises codegraph_explore"
  - "TestNoWatchEnvDisablesViaStderr — WATCH-03 proven on the real spawned binary: CODEGRAPH_NO_WATCH=1 in the child env still completes Initialize (never-block) and the child's real stderr (via mcp-go client.GetStderr) carries the verbatim D-12/D-13 disabled message"
  - "newServeClientWithEnv — a package-local sibling of 03-04's newServeClient that additionally passes env into the spawned subprocess (unused by TestDefaultWatchHandshakePrompt, available for future env-varying cases)"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "client.GetStderr(c) + a background goroutine reading into a mutex-guarded bytes.Buffer, polled with a bounded grace window — the pattern for observing a subprocess's off-handshake-path stderr output without racing the pipe buffer or hanging on an unbounded read"

key-files:
  created: [test/integration/watch_default_test.go]
  modified: []

key-decisions:
  - "Both tests reuse 03-04's buildWorktreeFixture (git-backed, t.Skip on git absence) purely for its indexed main checkout, discarding the worktree half — simpler than inventing a second git-free fixture helper, and it satisfies the plan's 'skip cleanly on git absence' acceptance criterion for free"
  - "newServeClientWithEnv is a NEW package-local helper (not a signature change to worktree_notice_test.go's newServeClient) since this plan's files_modified is scoped to watch_default_test.go alone; it duplicates newServeClient's ~15 lines with one added env parameter"
  - "TestDefaultWatchHandshakePrompt reuses newServeClient's own Initialize call as the first half of the Initialize+ListTools bounded-context assertion, rather than re-implementing Initialize inline — Initialize and ListTools both run under the same 30s ctx"
  - "Stderr is read continuously from a goroutine started BEFORE Initialize (not after), since serve.go's disabled message is printed by an off-handshake-path goroutine (D-06) that can race Initialize's return; the assertion polls a 10s grace window after Initialize rather than requiring the message to already be present"

patterns-established:
  - "Bounded-grace-window polling loop (100ms tick, 10s deadline) for asserting eventual content in a subprocess's continuously-drained stderr buffer, distinct from a bounded context.WithTimeout used for RPC calls"

requirements-completed: [TEST-04, WATCH-01, WATCH-02]

coverage:
  - id: D1
    description: "TestDefaultWatchHandshakePrompt: bare serve --mcp (no watch flag in argv) completes Initialize+ListTools within a bounded context and advertises codegraph_explore, proving the default-on watcher does not delay the handshake"
    requirement: "WATCH-01"
    verification:
      - kind: integration
        ref: "test/integration/watch_default_test.go#TestDefaultWatchHandshakePrompt"
        status: pass
    human_judgment: false
  - id: D2
    description: "Same test also proves WATCH-02 (the handshake-path budget guarantee) end-to-end over real stdio, complementing 03-03's structural unit test"
    requirement: "WATCH-02"
    verification:
      - kind: integration
        ref: "test/integration/watch_default_test.go#TestDefaultWatchHandshakePrompt"
        status: pass
    human_judgment: false
  - id: D3
    description: "TestNoWatchEnvDisablesViaStderr: CODEGRAPH_NO_WATCH=1 in the child env still completes Initialize (never-block) and the child's real stderr carries the verbatim 'File watcher disabled — CODEGRAPH_NO_WATCH=1 is set' message plus the codegraph sync refresh guidance"
    requirement: "TEST-04"
    verification:
      - kind: integration
        ref: "test/integration/watch_default_test.go#TestNoWatchEnvDisablesViaStderr"
        status: pass
    human_judgment: false
  - id: D4
    description: "Full-suite verification stays green with the two new cases added: go test ./..., go test ./testdata/golden/..., go test ./test/integration/... (now 3 tests total)"
    verification:
      - kind: other
        ref: "go test ./... && go test ./testdata/golden/... && go test ./test/integration/... -count=1 -v (all pass, run 2026-07-16)"
        status: pass
    human_judgment: false

# Metrics
duration: 5min
completed: 2026-07-16
status: complete
---

# Phase 3 Plan 5: D-21 WATCH Subprocess Coverage Summary

**Two new subprocess integration tests riding 03-04's harness — default-on `serve --mcp` completes the handshake promptly (WATCH-01/02), and `CODEGRAPH_NO_WATCH=1` disables the watcher observably via the real child's verbatim stderr message (WATCH-03) — closing the loop that the 03-03 default flip and 03-01 verbatim message actually reach production over real argv/stdio.**

## Performance

- **Duration:** 5 min
- **Started:** 2026-07-16T14:36:10Z
- **Completed:** 2026-07-16T14:40:49Z
- **Tasks:** 2
- **Files modified:** 1

## Accomplishments
- `TestDefaultWatchHandshakePrompt` — spawns `serve --mcp` with NO watch flag in argv (the default-on path), completes `Initialize` + `ListTools` within a single 30s bounded context, and asserts `codegraph_explore` is advertised — proving the default-on watcher never delays the handshake or first-tool availability, on the real production binary
- `TestNoWatchEnvDisablesViaStderr` — spawns `serve --mcp` with `CODEGRAPH_NO_WATCH=1` in the child's environment, confirms `Initialize` still completes (best-effort/never-block), and asserts the real child's stderr (captured via `mcpclient.GetStderr`) contains the verbatim `File watcher disabled — CODEGRAPH_NO_WATCH=1 is set` message plus the `codegraph sync` refresh guidance
- Both cases ride 03-04's substrate unchanged (`binPath`, `buildWorktreeFixture`, `newServeClient`) — zero new dependencies, no second `TestMain`
- Full verification suite (`go test ./...`, `./testdata/golden/...`, `./test/integration/...`) green with all three integration tests (including 03-04's CR-01 anchor) passing together

## Task Commits

Both tasks landed in one commit — they add to the same new file created in a single pass, so there is no independently-buildable intermediate state between them (same rationale 03-04 used to combine its Task 1/2):

1. **Task 1+2: watch_default_test.go — WATCH-01/02 handshake-prompt case + WATCH-03 NO_WATCH stderr case** - `afe96d2` (test)

**Plan metadata:** (this commit)

## Files Created/Modified
- `test/integration/watch_default_test.go` - `TestDefaultWatchHandshakePrompt`, `newServeClientWithEnv`, `TestNoWatchEnvDisablesViaStderr`

## Decisions Made
- Both tests reuse `buildWorktreeFixture`'s indexed main checkout (ignoring its worktree half) rather than building a second, git-free fixture helper — reuses 03-04's proven git-absence skip behavior for free.
- `newServeClientWithEnv` is a new package-local sibling of `newServeClient`, not a signature change to `worktree_notice_test.go` — this plan's `files_modified` is scoped to `watch_default_test.go` alone.
- Stderr is drained by a goroutine started before `Initialize` (not after), since the disabled message comes from serve.go's off-handshake-path goroutine and can race `Initialize`'s return; the assertion polls a bounded 10s grace window instead of requiring strict pre-Initialize ordering.

## Deviations from Plan

None — plan executed exactly as written. The only intentional adjustment (documented above, not a Rule 1-4 deviation) is combining both tasks into a single commit, matching the plan's own `files_modified` scoping to one new file.

## Issues Encountered

None. `go build ./...`, `go vet ./test/integration/...`, and `gofmt -l` all clean. Both new tests pass individually and alongside the existing CR-01 anchor test.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- TEST-04's WATCH coverage is complete: default-on handshake promptness (WATCH-01/02) and the NO_WATCH off-switch (WATCH-03) are both proven end-to-end on the real spawned binary, riding the 03-04 substrate with zero new dependencies.
- Phase 3's remaining requirement, WATCH-04 (concurrent-session convergence via defer-and-retry), is out of this plan's scope (03-02 territory per STATE.md's decision log — "WATCH-04 marked complete by 03-02").
- No blockers.

---
*Phase: 03-watcher-on-mcp-default*
*Completed: 2026-07-16*

## Self-Check: PASSED

All created files (`test/integration/watch_default_test.go`, `.planning/phases/03-watcher-on-mcp-default/03-05-SUMMARY.md`) and both commit hashes (`afe96d2`, `6a89c83`) verified present in the working tree and git history.
