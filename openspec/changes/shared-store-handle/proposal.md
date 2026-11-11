# Proposal: share one store handle in a process, and sync edits made while the watcher starts

Issue: #277

This change also resolves #170.

## Why

In one `serve --mcp` process, a tool call and the live sync each open the graph store. The store allows one open at a time, also inside one process. If the sync holds the store longer than the retry window of the tool call, the tool call returns `isError: true`. The retry window is about 400 ms. An agent then sees a failed `codegraph_explore` for an edit that it made itself.

The integration test `watch_live_sync` shows the defect. On `main` at `1e016455`, 46 of 72 runs failed with 24 copies of the test in parallel. Each failed run holds `codegraph_explore returned a tool error`. In 24 of them the daemon also logged `store lock held: lock held by current process`.

Issue #170 is in the same path. The daemon takes its lock and then opens the watcher. An edit made between those two steps gives no event, and no sync follows the watcher open. The graph misses that edit until a later event arrives. The same test hides the gap: its client writes one file again each 500 ms.

## What Changes

- Two opens of one store in one process share one handle. The second open does not fail on the lock and does not run the retry.
- A tool call that starts during a sync of the same server process answers at once. It reads the graph as it was before the sync or as it is after the sync, never a part of each.
- After the last user closes the shared handle, the process releases the store. A full index and a second process then get the store as they do today.
- A locked store keeps its present result for a holder in another process: a bounded retry, then `isError: true` with the text that the store lock is held.
- After the watcher opens, the daemon runs one catch-up sync. An edit made before the watcher was ready reaches the graph without a second edit.
- The `watch_live_sync` client stops its 500 ms write loop. The test then proves the catch-up sync.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `mcp-server`: the requirement "A briefly locked store does not fail or hang a call" gains the case of a sync in the same server process. A new requirement, "An edit made while the watcher starts reaches the graph", states that such an edit reaches later tool calls. A second new requirement, "A watcher-start sync that cannot get the store leaves the graph marked stale", states the result for a store that another process holds.

## Impact

- `internal/graphstore`: `Open`, `OpenExisting`, `Checkpoint` and `Close` share a handle for each store directory. `OpenStaging` and `ReplaceStore` keep their present behavior.
- `internal/daemon`: `Run` queues one flush after the watcher opens.
- `test/integration`: the live-edit client and the `watch_live_sync` script.
- `internal/graphstore/locktest`: a new test-helper package. Its `HoldStoreLock` holds the Pebble lock of a store as another process does.
- Tests that use a second in-process `Open` as the other holder move to that helper. They are in `internal/graphstore` (`open_lock_test.go`, `checkpoint_test.go`), `internal/uiserver` (nine tests) and `internal/wtindex` (`TestPrepareMainStoreLocked`).
- `internal/daemon` tests that count syncs, `TestSoak` included, take the catch-up sync into account.
- `internal/mcp`: one new test for the `isError` result of a store that another holder keeps.
- No change to CLI output or to MCP result shapes. No new dependency.

## Out of scope

- #197: an incremental sync reads the whole store. This change makes a long sync harmless to tool calls of the same process. It does not make the sync shorter.
- A tool call against a store that another process holds. That case keeps the bounded retry.
- #189, #128, #113, #233, #180 and #156.
- A handle that stays open for the whole session.
- One handle for two spellings of a store path through a symbolic link. Those spellings behave as they do today.
