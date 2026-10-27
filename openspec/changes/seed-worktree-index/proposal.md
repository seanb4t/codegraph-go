# Proposal: Give a linked worktree an index of its own when the MCP server starts there

Issue: #119

## Why

In a linked worktree without its own `.codegraph/`, the MCP server uses the main checkout's index. So it answers for the main checkout's tree: a symbol added only in the worktree is missing, and one changed there shows its old callers. Every result says so in a notice. The only fix today is to run `codegraph init` by hand in each worktree. Agents now create worktrees for most tasks, so most of their sessions get answers about the wrong tree.

The main checkout's index can be copied cheaply, and the existing incremental sync then brings the copy to the worktree's tree. On this repository (731 files), copying the store took 0.01 s. Syncing a worktree on the main checkout's commit took 0.16 s, against about 2 s for a full index.

## What Changes

- When `codegraph serve --mcp` starts in a linked worktree that borrows the main checkout's index, it gives the worktree an index of its own. It copies the main checkout's store as a pebble checkpoint and syncs the copy with the worktree's files. Then the new index takes the place of `.codegraph/` at the worktree root.
- "Borrows" means what the borrowed-index notice already detects: the index belongs to another working tree of the same repository. This covers a worktree outside the main checkout (the linked-worktree fallback) and a worktree nested inside it. A submodule or an embedded clone is not affected.
- The server makes the worktree index only when the main checkout is indexed. Whether a repository is indexed stays the user's decision.
- When few files differ from the copy, the sync finishes before the server reads its first request, so the first tool call already answers for the worktree. When many files differ, the server does not hold the handshake. It builds the index in the background, from the copy or from scratch, whichever is faster. Until the index is ready, tool calls use the main checkout's index and carry the existing notice. Then they use the new index with no restart, and the file watcher starts on it.
- `--no-worktree-index` or `CODEGRAPH_NO_WORKTREE_INDEX=1` turns this off. The server then borrows the main checkout's index, as it does today.
- If the worktree index cannot be made (for example, the worktree is read-only), stderr says why and the server borrows the main checkout's index, as it does today.
- The new index is invisible to git, like any `.codegraph/`. The main checkout's graph is not changed. `git worktree remove` deletes the worktree index with the worktree.

No CLI output shape or MCP tool result changes. A worktree that has its own index already gets no notice. This change makes that the usual case.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `mcp-server`: a new requirement, "A linked worktree gets an index of its own". "Finding the index" and "Worktree notice on a borrowed index" now describe borrowing as what happens when the worktree index is turned off or not yet ready. "Watcher limits within a session" gains an exception: a worktree index that the server itself builds in the background gets the watcher when it is ready.

`index-resolution` does not change. Its scenario "A worktree with its own index" already makes a worktree's own `.codegraph/` win over the fallback.

## Dependencies

- #187 must be merged first. Incremental sync never adds an edge to a symbol that appears after its caller was synced. A seeded index inherits every edge the main checkout's index is missing, and the sync from the seed to the worktree adds more of the same kind. This change's acceptance test (a seeded index equals a fresh index) fails until #187 is fixed.
- #189 corrects the notice's `codegraph init -i` advice. This change does not change the notice text.

## Out of scope

- One-shot CLI reads (`codegraph explore`, `callers` and the rest) and `codegraph status`. They keep borrowing, with the notice. They are short-lived, and an index built on a read would surprise the user.
- The graph UI (`codegraph ui`). It keeps borrowing.
- `codegraph init` in a worktree. It still builds from scratch: it is the command a user runs for a clean index. The index-writing commands keep their rule of never using the worktree fallback.
- A worktree whose main checkout is not indexed. No index is made. The server offers zero tools, as it does today.
- An index shared between worktrees, or keyed by worktree HEAD inside the main checkout's store.
- Using the cutoff for large diffs anywhere else. A branch switch in the main checkout could also be faster as a full rebuild, but that is a separate change.
- Removing a worktree index when its worktree is pruned without `git worktree remove`.

## Impact

- `internal/graphstore`: a checkpoint function that copies an open store into a new directory with hard links where it can. It is the only new pebble call.
- `internal/indexer`: `Sync` gains an option that makes it stop before it writes anything when more files changed than a limit allows.
- A new seeding step, called from `internal/cli/serve.go` before the startup sync. It uses `gitmeta.DetectIndexMismatch` to decide whether the server borrows, and builds in a temporary directory under the worktree root before it moves the result into place.
- `internal/cli/serve.go`: the new flag and environment variable. The startup sync and the watcher use the worktree index when there is one. In the background case, the watcher starts after the build.
- `internal/mcp`: no logic change. The per-call index resolution and the notice detector's cache (keyed on start directory and index root) already switch to a new worktree index as soon as it exists.
- Changelog fragment of kind Features.
