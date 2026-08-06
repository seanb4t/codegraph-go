---
phase: 02-sdk-migration-official-go-sdk-on-the-existing-surface
plan: 03
subsystem: api
tags: [mcp, go-sdk, jsonrpc, error-mapping, testing]

requires:
  - phase: 02-sdk-migration-official-go-sdk-on-the-existing-surface
    provides: "internal/mcp running on modelcontextprotocol/go-sdk v1.7.0 (02-01), including newTestSession/callTool/copyFixture/indexFixture test scaffolding this plan reuses"
provides:
  - "internal/mcp/error_mapping_test.go — four assertions pinning SDK-04's error-to-wire mapping (handler-returned error, schema-rejected missing argument, undeclared argument, engine-level failure), each checked against session-level error, IsError, and content shape"
  - "A demonstrated-RED mutation cycle proving the assertion is a gate, not a description (against a *jsonrpc.Error mutation on codegraph_status's error path)"
affects: [02-05]

actuals:
  tokens: 2059
  tasks: 2
  commits: 1

tech-stack:
  added: []
  patterns:
    - "Explicit three-part assertion (session-level nil error, IsError, single-element *mcp.TextContent type assertion) as the standard shape for any test claiming to pin go-sdk's error-to-wire conversion — checking IsError alone cannot distinguish a protocol error from a tool error (PITFALLS.md Testing Trap B)"

key-files:
  created:
    - internal/mcp/error_mapping_test.go
  modified: []

key-decisions:
  - "Reused newTestSession, copyFixture, and indexFixture from server_test.go/markdown_test.go rather than building a second harness, per the plan's explicit instruction and 02-CONTEXT.md's 'prefer the cheap mechanism' calibration note."
  - "Wrote a dedicated callToolRaw helper (rather than reusing markdown_test.go's callTool) because callTool's t.Fatalf on a non-nil session-level error would hide the distinction this file exists to make explicit — each test needed to inspect that error itself, not have a shared helper fail before the assertion ran."
  - "Chose codegraph_status's confineToRepoRoot rejection (a path outside the server's configured repo root) as the handler-returned-error exemplar and the Task 2 mutation site, since it is the same error path test/wireoracle/MUTATION-PROOF.md's Mutation 3 and server_test.go's TestOpenEnginePathConfinedToRepoRoot already exercise, keeping the mutation's blast radius (and this plan's added surface) minimal."
  - "Chose codegraph_node's 'symbol not found' (Engine.Node's own error) as the engine-level-failure exemplar, since it is a real, already-implemented engine error path with no fixture engineering required."

patterns-established:
  - "A test asserting go-sdk's error-to-wire conversion must check three things in the same test function: CallTool's session-level Go error is nil, result.IsError is true, and result.Content type-asserts to exactly one *mcp.TextContent — never IsError alone."

requirements-completed: [SDK-04]

coverage:
  - id: D1
    description: "A tool handler that returns a plain Go error produces a successful JSON-RPC response with isError:true and error text in content[0].text, never a top-level JSON-RPC error object"
    requirement: "SDK-04"
    verification:
      - kind: unit
        ref: "internal/mcp/error_mapping_test.go#TestHandlerErrorIsToolResultNotProtocolError"
        status: pass
    human_judgment: false
  - id: D2
    description: "tools/call omitting a required argument is rejected by the SDK's own schema validation before the handler body runs, and that rejection is also a tool-visible isError:true result"
    requirement: "SDK-04"
    verification:
      - kind: unit
        ref: "internal/mcp/error_mapping_test.go#TestMissingRequiredArgumentIsToolVisibleError"
        status: pass
    human_judgment: false
  - id: D3
    description: "tools/call carrying an argument the tool's schema does not declare is rejected rather than silently ignored (additionalProperties:false, D-10)"
    requirement: "SDK-04"
    verification:
      - kind: unit
        ref: "internal/mcp/error_mapping_test.go#TestUnknownArgumentIsRejected"
        status: pass
    human_judgment: false
  - id: D4
    description: "The confinement rejection path (confineToRepoRoot) and an engine-level failure (Engine.Node symbol-not-found) both produce the pre-migration tool-visible isError:true shape, not a protocol error"
    requirement: "SDK-04"
    verification:
      - kind: unit
        ref: "internal/mcp/error_mapping_test.go#TestHandlerErrorIsToolResultNotProtocolError"
        status: pass
      - kind: unit
        ref: "internal/mcp/error_mapping_test.go#TestEngineErrorIsToolResult"
        status: pass
    human_judgment: false
  - id: D5
    description: "The error-mapping assertion has been demonstrated RED against a confirmed-applied mutation, so it is a gate rather than a description"
    requirement: "SDK-04"
    verification:
      - kind: unit
        ref: "manual mutation cycle, recorded verbatim below (Task 2) — *jsonrpc.Error mutation on codegraph_status's openEngine error path, TestHandlerErrorIsToolResultNotProtocolError observed FAIL, reverted, observed PASS"
        status: pass
    human_judgment: false

duration: ~35min
completed: 2026-08-05
status: complete
---

# Phase 2 Plan 3: SDK Migration — official go-sdk on the existing surface Summary

**Four assertions in `internal/mcp/error_mapping_test.go` pin SDK-04's error-to-wire mapping (handler-returned error, schema-rejected missing argument, undeclared argument, engine-level failure) against the real go-sdk backend, demonstrated able to fail via a `*jsonrpc.Error` mutation.**

## Performance

- **Duration:** ~35 min
- **Tasks:** 2
- **Files modified:** 1 (new file; `internal/mcp/tools.go` was mutated and reverted during Task 2, ending byte-identical to its starting state — no net change)

## Accomplishments

- `TestHandlerErrorIsToolResultNotProtocolError` — a `codegraph_status` call whose `path` resolves outside the server's confinement root produces `IsError:true` with `outside` in the text, never a protocol error
- `TestMissingRequiredArgumentIsToolVisibleError` — a `codegraph_explore` call omitting `query` is rejected by go-sdk's `applySchema` before `exploreHandler` runs, also `IsError:true`, with wording that differs from the deleted pre-migration handler text
- `TestUnknownArgumentIsRejected` — a `codegraph_explore` call carrying an undeclared argument is rejected (`additionalProperties:false`, D-10) rather than silently ignored
- `TestEngineErrorIsToolResult` — a `codegraph_node` call for a nonexistent symbol surfaces `Engine.Node`'s `"symbol ... not found"` error as `IsError:true`, never a protocol error
- The assertion was demonstrated RED (Task 2): a `*jsonrpc.Error` mutation on `codegraph_status`'s `openEngine` error path made `TestHandlerErrorIsToolResultNotProtocolError` fail with the exact protocol-error message expected, then reverted cleanly

## Task Commits

1. **Task 1: Pin the four failure shapes SDK-04 names** — `c6015c7` (test)
2. **Task 2: Demonstrate the assertion RED** — no commit (mutation applied, observed, and reverted; `internal/mcp/tools.go` ends byte-identical to its Task-1-committed state, confirmed via `git status --porcelain` and `git diff`)

**Plan metadata:** (this commit)

## Files Created/Modified

- `internal/mcp/error_mapping_test.go` — new file, package `mcp`. Four test functions plus a `callToolRaw` helper (a session-level-error-preserving variant of `markdown_test.go`'s `callTool`, needed because that helper's `t.Fatalf` on a non-nil session error would short-circuit before this file's own assertion could run).

## Decisions Made

See `key-decisions` in frontmatter. In summary: reused all existing test scaffolding (`newTestSession`, `copyFixture`, `indexFixture`) per the plan's explicit instruction; added one small helper (`callToolRaw`) because the existing `callTool` helper's fail-fast behavior on a protocol error would hide exactly the distinction this file exists to assert; picked the confinement-rejection and engine-not-found paths as the handler-error and engine-error exemplars because both were already real, already-tested error paths requiring no new fixture engineering.

## Deviations from Plan

None — plan executed exactly as written. `internal/mcp/tools.go` and `internal/mcp/server.go` were not modified by the committed state of this plan (Task 2's mutation was applied, observed, and fully reverted — see the Mutation Cycle section below for the complete record, in `MUTATION-PROOF.md`'s format per the plan's instruction not to write a second document).

## Missing-Required-Argument Text (verbatim, for plan 02-05's diff review)

Captured directly from a passing run of `TestMissingRequiredArgumentIsToolVisibleError`:

```
validating "arguments": validating root: required: missing properties: ["query"]
```

This is go-sdk's `applySchema` validator's own wording, confirmed to differ from the pre-migration handler-authored text `required argument "query" not found` (asserted directly in the test — see the `preMigrationText` check in `TestMissingRequiredArgumentIsToolVisibleError`).

## Mutation Cycle (Task 2, MUTATION-PROOF.md format)

**Handler and site named exactly:** `companionHandler`'s `"status"` branch (`internal/mcp/tools.go`), the `openEngine` error path that `TestHandlerErrorIsToolResultNotProtocolError` exercises via a `codegraph_status` call with a `path` outside the server's configured repo root — the same error path `test/wireoracle/MUTATION-PROOF.md`'s Mutation 3 and `server_test.go`'s `TestOpenEnginePathConfinedToRepoRoot` already cover.

**Edit:**

```diff
--- a/internal/mcp/tools.go
+++ b/internal/mcp/tools.go
@@ -6,6 +6,7 @@ import (
 	"path/filepath"
 	"strings"

+	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
 	"github.com/modelcontextprotocol/go-sdk/mcp"

 	"github.com/seanb4t/codegraph-go/internal/gitmeta"
@@ -459,7 +460,7 @@ func companionHandler(s *mcp.Server, name, repoPath, defaultPath string, detecto
 		mcp.AddTool(s, tool, func(ctx context.Context, req *mcp.CallToolRequest, args StatusArgs) (*mcp.CallToolResult, any, error) {
 			eng, close, err := openEngine(args.Path, defaultPath, repoPath, detector)
 			if err != nil {
-				return nil, nil, err
+				return nil, nil, &jsonrpc.Error{Code: jsonrpc.CodeInternalError, Message: err.Error()}
 			}
 			defer close()
```

`jsonrpc.Error` is go-sdk's own type (`jsonrpc.Error = jsonrpc2.WireError`, `github.com/modelcontextprotocol/go-sdk/jsonrpc`); `toolForErr`'s wrapper (`02-RESEARCH.md` Q5) short-circuits on this concrete type and returns it as a top-level protocol error instead of packing it into `CallToolResult.Content` — exactly the mark3labs-shaped behavior SDK-04 exists to detect.

**Confirmed applied:**

```
$ git diff internal/mcp/tools.go
[the diff above, present]
$ go build ./...
$ echo $?
0
```

**Verbatim failure**, running `go test ./internal/mcp/... -count=1 -run TestHandlerErrorIsToolResultNotProtocolError -v`:

```
=== RUN   TestHandlerErrorIsToolResultNotProtocolError
    error_mapping_test.go:70: CallTool codegraph_status returned a session-level (protocol) error: calling "tools/call": mcp: path "/var/folders/_b/3hyf5qvs62q0wh2vyh856z580000gn/T/TestHandlerErrorIsToolResultNotProtocolError2785550900/002" is outside the server's configured repo root — want a successful JSON-RPC result with isError:true
--- FAIL: TestHandlerErrorIsToolResultNotProtocolError (0.13s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/mcp	0.473s
```

**Revert confirmation:**

```
$ git status --porcelain internal/mcp/tools.go
(empty)
$ git diff internal/mcp/tools.go
(empty)
$ go build ./...
$ echo $?
0
$ go test ./internal/mcp/... -count=1
ok  	github.com/seanb4t/codegraph-go/internal/mcp	3.350s
ok  	github.com/seanb4t/codegraph-go/internal/mcp/archtest	1.763s
```

The tree was confirmed clean (`git status --porcelain internal/mcp/tools.go` empty) before this document was written.

## Issues Encountered

None.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- `internal/mcp/tools.go`, `internal/mcp/server.go`, `test/wireoracle/`, and `testdata/wireoracle/` are all unmodified (confirmed via `git diff --stat`), matching this plan's explicit prohibitions.
- SDK-04 is closed: all four `must_haves.truths` bullets from the plan frontmatter are pinned by a passing, demonstrated-able-to-fail test, and the observed missing-required-argument wording is recorded verbatim above for plan 02-05's diff review.
- No new dependency was added to `go.mod` — the Task 2 mutation's `jsonrpc` import was reverted along with the rest of the mutation; `github.com/modelcontextprotocol/go-sdk/jsonrpc` is already reachable transitively via the SDK's own `mcp` package, but this plan does not add a new direct import to any committed file.

---
*Phase: 02-sdk-migration-official-go-sdk-on-the-existing-surface*
*Completed: 2026-08-05*

## Self-Check: PASSED

- `internal/mcp/error_mapping_test.go` confirmed present on disk.
- This SUMMARY confirmed present on disk.
- Commit `c6015c7` confirmed present in `git log`.
