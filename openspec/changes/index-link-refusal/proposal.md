# Proposal: refuse index links that lead outside, at start and while a watcher runs

Issue: #375

This change also resolves #377.

## Why

PR #374 makes every command refuse a `.codegraph`, or a `.codegraph/store`, that resolves outside the directory that holds it. The `index-resolution` spec does not state that refusal (#377), so no scenario traces the tests of #374 to a requirement.

When a command or a watcher starts, the check runs once. A watcher started by `codegraph serve --mcp` or `codegraph daemon start` can run for days. Git replaces an ignored `.codegraph` directory with a committed link without a warning. After such a pull, the next flush follows the link and can write into another project's index (#375). A watcher that defers to a live daemon takes the lock through the link at each retry. The shutdown of the watcher also removes `daemon.lock` through the link. That file can belong to a live daemon of the other project.

## What Changes

- The `index-resolution` spec states the refusal from #374: a `.codegraph` or `.codegraph/store` that resolves outside the directory that holds `.codegraph` is refused, not skipped. The spec covers a dangling link, the linked-worktree fallback, a subdirectory link to an ancestor's index, and the text of the error. A link that resolves inside stays allowed.
- A watcher resolves the index paths again before it takes its lock file and before each flush. If the resolution fails, is refused, or differs from the paths the watcher started with, the watcher writes nothing, stops, and reports why. The `serve --mcp` process keeps serving tool calls. Those calls already refuse a link that leads outside.
- A watcher that shuts down checks two things before it removes its lock file: `.codegraph` still resolves to its start directory, and the lock names its own process. If either check fails, it leaves the lock file in place.
- **BREAKING** (narrow): when its index moves inside the repository, a watcher now stops and asks for a restart. An example is a `.codegraph` directory that becomes a link to `.cg-real`. Before, the watcher kept writing through the new path.

Out of scope:

- `ReplaceStore` through a symlinked live store directory (#364). Only a full index swaps a store, and a watcher never runs a full index.
- File discovery that walks into an index directory inside the repository whose name has no leading dot (#376).
- Closing the time-of-check to time-of-use window between the check and the open of the store. `design.md` records the residual window.
- A watch on `.codegraph` itself that stops the watcher at the moment a link appears, before any flush.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `index-resolution`: adds the refusal of an index that leads outside the directory that holds it. It also adds a second check by a running watcher before each write to the index.

## Impact

- `internal/daemon`: `Daemon.Run` resolves the index paths again before `acquire` and before each flush, and checks the lock owner before the release. When the check fails, `Run` returns the error. `RunWithRetry` returns that error without a retry.
- `internal/cli/serve.go` and `internal/cli/daemon.go`: the existing error paths print the new error, and each file gets one function variable as a test seam. `codegraph daemon start` exits 1.
- `internal/query/confine.go`: no behavior change. The watcher reuses `query.IndexDirs`.
- Specs: `openspec/specs/index-resolution/spec.md`.
- No change to CLI output on success, to MCP tool results, or to the store format.
