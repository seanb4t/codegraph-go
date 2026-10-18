# Proposal: Find the main checkout's index from a linked worktree anywhere on disk

Issue: #139

## Why

A linked git worktree uses the main checkout's index only when the worktree sits inside the main checkout. The index search (`query.ResolveCodegraphDir`) walks only parent directories. A worktree at `../sibling`, or in a tool's own worktree directory, never passes through the main checkout. So the CLI says `query: not initialized`, the MCP server offers zero tools, and the hook nudges stay silent, although an index exists. Agents that create worktrees outside the repository are now common, so a worktree session often loses codegraph completely.

## What Changes

- Index resolution gains a second step. When the upward walk finds no index, and the nearest `.git` entry above the start directory is a file (a linked worktree), codegraph asks git for the repository's main worktree. If the main worktree holds `.codegraph/`, it is the index root. The home-directory rule still applies to it.
- The walk stays first. A worktree with its own `.codegraph/`, and any indexed ancestor, still win. A nested worktree resolves exactly as it does today.
- Git runs only on that fallback path, never for an indexed directory or for an ordinary repository without an index. The main-worktree answer is remembered for the life of the process, so the MCP server does not start a git process on every request.
- Any git failure (git missing, a timeout, a bare repository with no main worktree) means "no index", as the walk's own failure does today.
- The MCP server separates two roots that are the same today. The index root is where `.codegraph/` lives; the startup sync and the watcher use it. The confinement root limits the `path` argument of a tool call. When the index comes from the fallback, the confinement root is the linked worktree's root, so the server's own default `path` is accepted and a client still cannot point a tool call at another directory.
- The borrowed-index notice and the `codegraph status` warning now also appear in a sibling worktree. The detection code already handles this case; it never received a sibling worktree before.
- The hook nudges treat a linked worktree root as indexed when its main worktree holds `.codegraph/`. The shell gates pass when `.codegraph/` exists or `.git` is a file. `codegraph hook pretooluse` then makes the full check for both harnesses. `session-nudge.sh` still never runs the codegraph binary; in a linked worktree it asks git for the main worktree.

No CLI output shape or MCP tool result changes. The notice and the warning are existing shapes that now reach one more case.

## Capabilities

### New Capabilities

- `agent-nudges`: when the SessionStart and PreToolUse nudges fire, by whether the project directory is indexed, including a linked worktree whose main checkout is indexed. This first specifies the existing NUDGE-01 to NUDGE-04 and CODEX-05 index gate.

### Modified Capabilities

- `index-resolution`: the nearest-ancestor requirement ends in a worktree fallback instead of "not initialized", and a new requirement defines that fallback.
- `mcp-server`: "Finding the index" separates the index root from the confinement root and gains a sibling-worktree scenario. "Worktree notice on a borrowed index" gains a sibling-worktree scenario.

## Out of scope

- A per-worktree index, automatic `init` in a worktree, or an index keyed by worktree HEAD. That is #119. Results in a sibling worktree still reflect the main checkout's tree, with the notice saying so, as in a nested worktree today.
- Commands that write an index. `init`, `index`, `sync`, `uninit` and `daemon` keep using their target directory and never take the fallback, so a git hook that runs `codegraph sync` in a worktree cannot write the worktree's files into the main checkout's index.
- Any other linked worktree's index. Only the main worktree is checked, not the other worktrees of the repository.
- Worktrees found through `GIT_DIR` or `GIT_WORK_TREE` with no `.git` entry on disk.
- A hook copy that a new worktree does not carry. A rendered hook script that is untracked in the main checkout is not present in a new worktree; installing hooks per worktree is a separate problem.
- The Codex nudge in a linked worktree when git is not on the hook's `PATH`. The Codex guard's own header says Codex runs hooks with an empty `PATH`; #140 covers it.
- A start directory below a project root for the nudges. The SessionStart nudge still checks only `$CLAUDE_PROJECT_DIR`, not its parents.

## Impact

- `internal/gitmeta`: a new main-worktree lookup (`git worktree list --porcelain`) with the package's timeout and degrade contract, and a process-lifetime memo.
- `internal/query/resolve.go`: the walk records the nearest `.git` entry, then falls back. A new resolver call returns the index root and the confinement root; `ResolveCodegraphDir` keeps its signature. Tests in `internal/query`.
- `internal/cli/serve.go`: `serveServerPaths` returns both roots; `BuildServer` gets the confinement root, the sync and the watcher get the index root. `internal/cli/serve_test.go` pins it.
- `internal/mcp`: no logic change. A test shows a tool call from a sibling worktree succeeds and carries the notice.
- `internal/cli/hook_pretooluse.go`: both harness paths check the index through one helper.
- `.claude/hooks/session-nudge.sh`, `.claude/hooks/pretooluse-nudge.sh`, `.codex/hooks/codegraph-pretooluse-local.sh`, `.codex/hooks/codegraph-pretooluse-global.sh`: embedded product code. Users get the new copies when they re-run `codegraph install`.
- The UI (`internal/uiserver`) and the query commands change through the resolver only.
- Changelog fragment of kind Fixes.
