# Design

## Context

See proposal.md for why. The relevant code today:

- `newServeCmd` (`internal/cli/serve.go`) resolves the paths with `serveServerPaths(start)`, which returns `(indexRoot, confineRoot, hasIndex)` from `query.ResolveIndex`. Then it runs the startup sync, `indexer.Sync(indexRoot, <indexRoot>/.codegraph/store, ...)`, before the handshake. A lock-held failure there is a stderr warning, and any other failure stops the server. Then `serveWatchStart` starts the watcher goroutine on `indexRoot`, off the handshake path.
- Tool calls do not use the startup paths for reads. `openEngine` (`internal/mcp/tools.go`) confines the call's `path` to the confinement root, then calls `query.OpenAt`, which walks up from that path again on every call. A `.codegraph/` that appears at the worktree root during a session is used from the next call.
- `query.OpenAt` opens `<dir>/.codegraph/store` with `graphstore.Open`, and pebble creates an empty store when the directory has none. So a `.codegraph/` without a finished store answers every query with nothing.
- `gitmeta.DetectIndexMismatch(ctx, start, indexRoot)` returns a `Mismatch{WorktreeRoot, IndexRoot}` exactly when the index belongs to another working tree of the same repository (shared git common dir). It covers a nested worktree, which the upward walk finds, and a sibling worktree, which the linked-worktree fallback finds. It suppresses submodules and embedded clones. The MCP server's `CachingDetector` keys its cache on start path and index root, so a new index root is a new key.
- `graphstore.Open` takes pebble's exclusive directory lock and retries 5 times, 100 ms apart. It wraps a lock still held after that in `ErrStoreLocked`. The graphstore has no checkpoint function yet. pebble v2's `DB.Checkpoint(destDir)` copies the WAL, MANIFEST, OPTIONS and sstables, with hard links where it can (`vfs.LinkOrCopy`).
- `indexer.Sync` classifies files as added, modified or deleted (`internal/indexer/sync.go`, about lines 100 to 146) before it opens a writer. A store without the file index (`needsFileIndexBackfill`) makes it run a full index instead.
- Node ids are `sha256(kind, qualifiedName, repo-relative path)`, so nothing in a store depends on the checkout's absolute path. A checkpoint of the main checkout's store is a valid store for the worktree.
- `codegraph init` writes `.codegraph/.gitignore` with `*`, which makes git ignore the directory and everything in it. `indexer.ShouldSkipDir` skips every dot-prefixed directory in discovery.
- The spike for this change, on this repository (731 files): store checkpoint 0.01 s; a seeded sync of a worktree on the main checkout's commit 0.16 s (every file hashed, none reparsed); a full index about 2 s warm. A worktree 30 commits back (158 files reparsed, 81 dependents) took 11.1 s from the seed, against 7.0 s from scratch.

## Goals / Non-Goals

**Goals:**

- In the common case, the first tool call in a new worktree already answers for the worktree's tree.
- The handshake never waits for work that grows with the size of the diff or the repository, beyond a fixed bound.
- No reader ever opens a worktree index that is not finished.
- Two servers starting in the same worktree at the same time end with one valid index.

**Non-Goals:**

- No change to `DetectIndexMismatch`, the resolver, the notice text or the status warning.
- No change to `indexer.Sync`'s results. Its only new behavior is an optional early stop.
- No config file. codegraph has none, and one switch does not justify one.

## Decisions

**The trigger is the existing borrowed-index detector.** After `serveServerPaths`, the server calls `gitmeta.DetectIndexMismatch(ctx, start, indexRoot)`. A non-nil result means the server borrows, and `Mismatch.WorktreeRoot` is where the worktree index goes. This catches both a sibling and a nested worktree. It also inherits the detector's suppression of submodules and embedded clones and its "no signal" on any git failure. Alternative considered: seed only when `ResolveIndex` took the linked-worktree fallback. Rejected, because a nested worktree (`<main>/.worktrees/wt`) borrows through the upward walk and has the same wrong answers.

**The seed is a pebble checkpoint.** A new `graphstore.Checkpoint(srcDir, destDir string) error` opens the source store with `Open` (so with its lock retries and `ErrStoreLocked`), calls `DB.Checkpoint`, and closes it. It is the only new pebble call, and it stays in graphstore, the one pebble-aware package. Alternative considered: copy the store directory with file operations. Rejected, because the main checkout's watcher can write while the copy runs, and the copy would then be inconsistent. Alternative considered: always build from scratch. Rejected, because in the common case it is about 12 times slower than a seeded sync (2 s against 0.16 s on this repository), and the gap grows with the repository. Alternative considered: one store shared by all worktrees, keyed by worktree HEAD. Rejected in exploration: it needs several writers on one pebble store, and every key would need a worktree dimension.

**The build happens in a temporary directory next to the target.** The seeder creates `<worktree>/.codegraph.seed-<random>/`, writes `.gitignore` with `*` into it first, and builds `store/` inside it. When the store is closed, it renames the directory to `<worktree>/.codegraph`. The temporary directory is on the same filesystem as the target, so the rename is atomic. Its name starts with a dot, so discovery and the watcher skip it. Its own `.gitignore` hides it from git, as `.codegraph/` is hidden. Alternative considered: build in the worktree's git directory. Rejected, because that directory is under the main checkout's `.git`, which can be on another volume, and a rename across volumes fails. Alternative considered: build directly in `<worktree>/.codegraph/`. Rejected, because a tool call would find the directory and open a partly built store, or create an empty one.

**A file lock guards each build, and losing a race is normal.** The seeder holds an exclusive `flock` on `<tmp>/.lock` for the whole build (through `golang.org/x/sys/unix`, already a dependency, behind a unix build tag like `locked_unix.go`). The kernel releases the lock when the process dies. Before it starts, a seeder removes every `.codegraph.seed-*` directory whose lock it can take, which clears the remains of a crashed server and never touches a build in progress. Just before the rename, the seeder checks for `<worktree>/.codegraph`. If one exists, or the rename fails because one appeared, the seeder deletes its own directory and uses the existing index. One small gap remains: a rename onto an empty directory succeeds on Linux and macOS. An empty `.codegraph/` exists only inside `codegraph init`'s single `MkdirAll` call, so this is accepted.

**Few changes finish in the foreground. Everything else finishes in the background.** This differs from the exploration's decision to do all of the work before the handshake. A full build cannot hold the handshake: on a large repository it can take longer than a client's MCP startup timeout. The client would then restart the server, which would start the build again and might never finish. So the work splits at a bound:

1. Foreground, before the startup sync: checkpoint the main checkout's store into the temporary directory. Skip to the background when the source store is locked after `Open`'s retries, has no file index, has a different schema version, or is on another device than the worktree (a cross-device checkpoint copies every file).
2. Foreground: run `indexer.Sync` on the seed with the new `Options.MaxChanged` set to the foreground limit. `Sync` returns `ErrTooManyChanges{Changed, Total}` right after it classifies the files, before it opens a writer, when added plus modified plus deleted files exceed the limit. Within the limit, the sync finishes, the rename happens, and `serveServerPaths(start)` runs again and returns the worktree as both roots.
3. Background: the watcher goroutine builds the index first. It runs `Sync` on the seed when the changed files are at most a fraction of the total, and `indexer.Run` into a fresh store otherwise or when there is no seed. Then it renames the directory into place and calls `daemon.New` on the worktree. Until the rename, tool calls borrow the main checkout's index with the notice. The startup sync of the main checkout runs as it does today, so "Startup sync before serving" does not change.

Provisional values: a foreground limit of 50 changed files, and a seeded background sync up to 10% changed files. The spike gives about 50 ms per reparsed file with dependents, so 50 files hold the handshake for about 2.5 s. A benchmark task calibrates both values. Alternative considered: everything in the foreground, as decided in exploration. Rejected for the reason above. Alternative considered: everything in the background. Rejected, because the common case (0.16 s) would race the agent's first tool call, and that call would get the main checkout's answers.

**In the background case, the watcher starts on the worktree after the build.** `serveWatchStart` gets an optional build step that runs before `daemon.New`, inside the goroutine it already starts. So the handshake is not delayed, and the watcher watches the index that the session will serve. The main checkout is not watched by this session during the build: its own sessions keep it fresh, and results from it carry the notice.

**The opt-out is a flag and an environment variable.** `--no-worktree-index` and `CODEGRAPH_NO_WORKTREE_INDEX=1` follow `--no-watch` and `CODEGRAPH_NO_WATCH=1`: only the value `1` counts, and the server reads it once at startup. An agent's MCP configuration can set either one.

**`codegraph init` does not seed.** It is the command a user runs for a known-clean index, and the index-writing commands keep their rule of never touching the main checkout's index because of a worktree. A seeded `init` would be faster, but a fresh build is the way out of any drift in a seed.

**Acceptance is equality with a fresh index.** The main test builds a main checkout and a worktree whose branch adds, changes and removes symbols. It includes the #187 shape: an unchanged file that calls a symbol the worktree adds. The test seeds the worktree index, runs `indexer.Run` on a copy of the worktree, and compares the two graphs' nodes and edges. This is the property every other test relies on, so it comes first.

## Risks / Trade-offs

- [A seed inherits drift from the main checkout's index] → #187 is a prerequisite, and the equality test fails while it is open. Node-count drift in the meta record (#188) is copied too. `codegraph status` counts nodes by a scan, so only `sync`'s summary shows it, and the server runs `sync` quietly.
- [Each worktree index costs disk] → The checkpoint shares sstables through hard links, but compaction in either store replaces them over time, so a long-lived worktree index costs as much as a `codegraph init` there. `git worktree remove` deletes it with the worktree.
- [A user ran `codegraph uninit` in a worktree, and the next server makes the index again] → `--no-worktree-index` or `CODEGRAPH_NO_WORKTREE_INDEX=1` stops it.
- [The checkpoint briefly holds the main checkout's store lock] → It links about a dozen files, so it holds the lock for milliseconds. A sync in the main checkout that collides retries through `Open`'s backoff, as it already does with tool calls.
- [The provisional limits are wrong for other repositories] → They are constants, calibrated by a benchmark on this repository. The spec says only "a fixed number", so new values need no spec change. In the spike, a file `Sync` reparsed cost between 5 and 17 times a file in a full index, depending on which full-index run is compared. If that cost drops, the benchmark moves the limits up.
- [A tool call during a background build answers for the main checkout's tree] → The notice says so, as it does today, and stderr reports when the worktree index is ready.

## Migration Plan

Ship in one release. There is no data migration: a worktree index is an ordinary index that any codegraph version reads. Worktrees that already hold `.codegraph/` are not touched. To go back to borrowing, set `CODEGRAPH_NO_WORKTREE_INDEX=1` in the agent's MCP configuration, or downgrade. A worktree index that a server already made stays, and is used, until `codegraph uninit` removes it.

## Open Questions

- The two limits' final values. The benchmark task sets them, and neither changes the specs or the tasks.
