# Design

## Context

See proposal.md for why. The relevant code today:

- `query.ResolveCodegraphDir` (`internal/query/resolve.go`) walks `filepath.Dir` upward from `filepath.Abs(start)`, skips the home directory, and returns the first directory that holds `.codegraph/`. All read paths use it: `query.OpenAt` (CLI query commands, MCP tool handlers, the UI server), `serveServerPaths` (`internal/cli/serve.go`), the MCP server's index check on every request (`recheckCatalog` in `internal/mcp/server.go`), and the UI's degrade and live-publish paths.
- `serveServerPaths` returns one `repoPath`. `newServeCmd` uses it for three jobs: the store directory for the startup `indexer.Sync`, the root for `daemon.New` (the watcher), and the confinement root it gives `BuildServer`. `confineToRepoRoot` (`internal/mcp/tools.go`) rejects a tool call's `path` unless it is under that root. Today that root is the start directory or one of its ancestors, so the default `path` always passes.
- The `query.Engine` uses only `repoRoot` for its reads (source text, status, the store). `startPath` is used only for worktree detection. So once the resolver returns the main checkout, queries work without other changes.
- `gitmeta.DetectIndexMismatch` already reports a mismatch for a worktree outside the main checkout. `newLinkedWorktreeFixture` in `internal/gitmeta/fixtures_test.go` is exactly that shape and is its main true-positive case. The resolver never gives the detector this input, because it fails first.
- `gitmeta` shells out to `git` with `gitTimeout` (5 s) and turns every failure into an empty result (D-03, WORK-03). `CachingDetector` (`internal/gitmeta/cache.go`) is the package's model for a bounded per-process cache (`maxCacheEntries`).
- One git process costs about 33 ms on the maintainer's macOS machine (measured from Go with `git worktree list --porcelain`). `recheckCatalog` runs on every MCP request, and `openEngine` resolves again on every tool call.
- `init`, `index`, `sync`, `uninit` and `daemon` get their root from `targetRoot` (`internal/cli/init.go`) and never call the resolver.
- The hooks: `session-nudge.sh` and `pretooluse-nudge.sh` test `[ -d "${CLAUDE_PROJECT_DIR:-.}/.codegraph" ]`. The Codex guards test `$root/.codegraph` (local) or `$PWD/.codegraph` (global). In Go, `runHookPreToolUseCodex` checks `event.Cwd/.codegraph` again (D-22). `runHookPreToolUse`, the Claude path, does not check for an index; it relies on the shell gate. The shell scripts are `//go:embed` sources that `codegraph install` renders.

## Goals / Non-Goals

**Goals:**

- A read command resolves the same index from a sibling worktree as from a nested worktree.
- An indexed directory or an ordinary repository without an index starts no git process. A linked worktree starts at most one per process.
- The MCP trust boundary does not widen: a client still reaches only the tree the server started in.

**Non-Goals:**

- No change to the detection gates in `gitmeta.DetectIndexMismatch`, to the notice text, or to the status warning.
- No `.git` file parsing in Go. Git stays the authority on worktree layout.
- No new dependency.

## Decisions

**Find the main worktree with `git worktree list --porcelain`.** The first record of its output is always the main worktree. A record with a `bare` line means the repository has no main checkout. A new `gitmeta.MainWorktree(ctx, dir) string` runs it from `dir` with `gitTimeout` and `cmd.Stdin = nil`, reads the first `worktree <path>` line, and returns `realpath(path)`. It returns "" for a bare repository and for every failure. Alternative considered: `git rev-parse --git-common-dir` and its parent directory. Rejected, because it gives the wrong answer for a main checkout made with `--separate-git-dir`, where the common directory is not `<main>/.git`. Alternative considered: read `.git` and `<gitdir>/commondir` from disk with no process. Rejected, because it copies git's on-disk layout into this repository (relative `gitdir` paths, `worktree.useRelativePaths`, separate git dirs) and breaks the package's rule that git is asked, not imitated.

**Run git only when the nearest `.git` entry is a file.** The upward walk already stats `<dir>/.codegraph` at each level. It now also stats `<dir>/.git` until it finds the first `.git` entry, and records that directory when the entry is a regular file. A linked worktree and a submodule have a `.git` file; a main checkout has a `.git` directory. The fallback runs only if the walk ends with no index and a `.git` file was found. It then runs `MainWorktree` from that directory. So an indexed directory adds no work, an ordinary unindexed repository adds stats only, and a submodule outside any indexed tree costs one git process, whose answer (the submodule itself, which was already checked) gives no index. Only the nearest `.git` entry counts. A normal clone inside a linked worktree is its own repository, and its `.git` directory stops the search for a `.git` file.

**Remember the main worktree for the life of the process.** `MainWorktree`'s answer, keyed by the cleaned directory it ran from, goes into a bounded cache in `gitmeta`, built like `CachingDetector`: the same `maxCacheEntries`, and cleared when full. Only an answer from a git run that exited 0 is cached, and that includes "bare, no main worktree". A timeout or a missing git is not cached, so the next call tries again. Whether the main worktree holds `.codegraph/` is never cached. That is a stat on each call, so an `init` in the main checkout during a session is seen at the next request. Without the cache, an MCP server in a sibling worktree would start a git process at least twice for every tool call (`recheckCatalog`, then `openEngine`). Alternative considered: cache in `serveServerPaths` and pass the root down. Rejected, because `recheckCatalog` and `openEngine` resolve on purpose on every call, and the CLI and the UI would need the same plumbing.

**One resolver with two answers.** A new `query.ResolveIndex(start) (Resolution, error)` returns `Resolution{IndexRoot, ConfineRoot}`. For an index the walk finds, both are the found directory. For the fallback, `IndexRoot` is the main worktree and `ConfineRoot` is the linked worktree's root (the directory that held the `.git` file). `ResolveCodegraphDir` keeps its signature and returns `IndexRoot`, so `OpenAt`, `recheckCatalog` and the UI need no edit. The home-directory rule applies to the main worktree too: it is compared with `filepath.Clean(os.UserHomeDir())`, as the walk does.

**The confinement root is the linked worktree, not the main checkout.** `serveServerPaths` returns `(indexRoot, confineRoot, hasIndex)`. `newServeCmd` gives `indexRoot` to `indexer.Sync` and `daemon.New`, and `confineRoot` to `BuildServer`'s `repoPath`. `BuildServer`'s promise holds: the confinement root is the start directory or an ancestor of it, so the default `path` passes. A tool call's `path` inside the worktree goes through `OpenAt`, which takes the same fallback to the same index. Alternative considered: confine to the main checkout. Rejected, because the server's own default `path` would then be outside the root, and every call would fail. Alternative considered: allow both trees. Rejected, because it would let a client reach a second tree. The main checkout's files reach the client only as index results, and they do so already.

**The hooks get one Go check and cheap shell gates.** `query.ProjectIndexed(dir) bool` is true when `dir` holds `.codegraph/`, or when `dir/.git` is a file and the cached `MainWorktree(dir)` holds `.codegraph/` and is not the home directory. `runHookPreToolUseCodex` uses it in place of its `os.Stat`. `runHookPreToolUse` gains the same check, with `CLAUDE_PROJECT_DIR` from its `getenv` (or the working directory when that is unset), as its shell gate uses. The shell gates widen to `[ -d "$d/.codegraph" ] || [ -f "$d/.git" ] || exit 0`, and the Go check makes the final decision. `session-nudge.sh` cannot ask Go, because it never runs the binary (NUDGE-01). In the `.git`-file case it runs `git -C "$d" worktree list --porcelain`, reads the first `worktree` line, and prints the nudge when that record is not bare, the path is not `$HOME`, and `<path>/.codegraph` is a directory. Alternative considered: let `session-nudge.sh` call a new `codegraph hook session`. Rejected, because the script's contract (NUDGE-01, and the comment at its top) is that it never runs the binary, and the git call is needed only in a linked worktree.

**Tests use real git fixtures.** The tests check our resolver against real git worktrees. They do not test git's behavior. `internal/gitmeta` already builds such fixtures (`initRepo`, `newLinkedWorktreeFixture`, `newSubmoduleFixture`). Tests in `internal/query` build the same shapes. To prove the cache, a test sets `PATH` to an empty directory after the first call, and the second call must still return the main worktree. To prove that git does not run, a test puts a stub `git` first on `PATH`. The stub writes a marker file and then runs the real git. The test then checks that the marker is present for a linked worktree (the positive control) and absent for an ordinary repository.

## Risks / Trade-offs

- [The main checkout moves during a long MCP session] → The cached path is stale, the stat fails, and the server reports no index until it restarts. Git itself needs `git worktree repair` after such a move, so this is rare. The cache holds only paths, never whether an index exists.
- [A sibling worktree's results come from the main checkout's tree] → This is already the nested-worktree behavior, and the notice says so on every result. A per-worktree index is #119.
- [A newline in the main worktree's path] → The first-line parse returns a wrong path, the stat fails, and the result is no index. That is the safe failure, so `-z` output (git 2.36 and later) is not needed.
- [Two implementations of the main-worktree lookup, Go and shell] → Both read the same first porcelain record. The shell copy is about five lines and has its own stub-based tests in `internal/agents`, next to the existing hook tests.
- [The SessionStart hook in a linked worktree now runs git] → One local git process, once per session start, and only when `.git` is a file. NUDGE-02's "no overhead" still holds for a repository with a `.git` directory, and the `agent-nudges` spec now states the exact limit.
- [The Codex guard's header (D-22) says Codex runs hooks with an empty `PATH`] → If that is so, `codegraph hook pretooluse --harness codex` finds no git, `MainWorktree` returns "", and the Codex nudge stays silent in a linked worktree. This is the safe failure. The claim was not verified for this change, and the `agent-nudges` spec states the limit. #140 covers giving the Codex hook a git it can find.
- [Users keep the old hook copies until they re-run `codegraph install`] → The resolver, CLI and MCP fix works without a reinstall. Only the nudges need it. The changelog fragment says so.

## Migration Plan

1. Ship in one release. Read commands and the MCP server change with the binary.
2. Users re-run `codegraph install` to get the new hook scripts. Old copies keep working as before; they only stay silent in worktrees.

Rollback: revert the release. No data format or on-disk state changes, so no step is needed.
