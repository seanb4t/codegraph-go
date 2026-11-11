# Proposal

Issue: #271

## Why

The peak memory of a full index grows with the number of files times the memory that each file needs. Pass 1 keeps the extraction result of every file in one slice. Resolution then runs on one thread over all of them, and one Pebble batch holds the whole graph. Because of this, each byte that a feature adds per file costs 120,000 bytes on the regression corpus. #250 raised peak RSS (resident set size, the memory a process holds) by about 40% for an identical graph (#265). #268 removed that overhead, but the next feature can add it again. #272 split the indexer into the discover, extract, symbols and resolve layers, so the pipeline can now change without a rewrite of the resolver.

## What Changes

- A full index runs as a pipeline of parallel passes with barriers between them. A file holds working state from the moment a worker claims it until the writer stages its records. At most W files do so at any time. W defaults to `runtime.GOMAXPROCS(0)`, which follows container CPU limits, and `--workers` sets it.
- Pass 1 extracts each file and writes the extraction result to a scratch spill file. The later passes read each result back by its offset.
- The later passes resolve types, then build the type hierarchy at a barrier, then resolve calls. The edges of each file are collapsed and staged as soon as that file is done. The edges whose owner node has no file, such as the base edges of a C# partial type, are collapsed together at the end.
- The writer is one goroutine. It puts the records into Pebble batches and commits a batch as soon as its size passes a fixed cap.
- A full index builds the new store in a locked scratch directory next to the live store. It swaps the new store with the live store in one atomic exchange. A run that fails or stops halfway leaves the previous graph readable and unchanged. `codegraph index` no longer deletes the store before it starts.
- A full index holds a new writer lock for its whole run, and so does `codegraph sync`. A sync during a full index fails with the store-locked error, and a query still reads the previous graph.
- The first error cancels the whole run. No goroutine and no scratch file outlives the run.
- When several files emit one node id, the store holds one node that merges their copies. Today the store keeps one copy, which an unstable sort chooses. This changes the graph of a C# partial type whose parts differ in their visibility. A sync that reparses one part of such a type also reparses the other parts, and removes the edges of a part that is gone.
- `graphstore.Import` writes through the same bounded batch writer, under the same cap.
- One default and one help text serve `--workers` in `codegraph init`, `index`, `sync` and `daemon`. The default changes from `runtime.NumCPU()` to `runtime.GOMAXPROCS(0)`. These are the same on a machine without a CPU limit.
- `Options.CoverageGenerationFloor` goes away. The run reads the prior coverage generation from the live store itself, because the CLI no longer deletes the store first.

## Capabilities

### New Capabilities

- `index-build`: How a full index runs. It covers the bound on the work in flight and on each write batch. It also covers how a new store replaces the old one, and how a run ends on an error.

### Modified Capabilities

- `graph-index`: INDX-02 gets a scenario for the order in which workers take files. A new requirement states how one node merges the copies that several files emit for one id. A sync merges them in the same way.

## Impact

- `internal/indexer`: `pipeline.go`, `extract.go`, `resolve.go`, `stage.go`, `sync.go` and `routes_detect.go` change. A new spill codec, a shared claim loop, one per-file extraction body and a new writer are added. Route detection runs on the claim loop. `writeGraph`, `Resolve`, `resolveRefs` and the `resolveFunc` seam go away, and the unexported `run` takes a context.
- `internal/indexer/resolve`: `Link` splits into a part that runs once at the hierarchy barrier and a part that runs for each file. `Link` itself becomes a loop over the per-file part, plus the units that a sync does not reparse. `CollapseEdges` runs for each file and once more for the edges with no owner file.
- `internal/indexer/symbols`: `Builder.AddResults` takes a declarations type, built by one constructor, and the builder merges nodes that share an id. `AddInterfaceMethods` goes away. `extract.File` sets the interface methods of each node, `extract.LanguageSpec` gets a default visibility, and `csharpextract` names its default as a constant.
- `internal/graphstore`: an open that does not create a store, and an open of a staging store with the WAL off. Also the swap lock in `open`, the writer lock, and a lock retry helper that takes a classifier for each kind of lock. Also the replacement of a store, a bounded batch writer that commits on the batch length, and a batched `Import`. `NewBoundedWriter` replaces `NewWriterSized` on the `GraphStore` interface, `BatchSize` and `Writer.DeleteGraph` go away, and the export kind tags become exported. `Checkpoint` changes only by the swap lock.
- `internal/recordio`, new: the deterministic marshal and the export framing, with an exported frame reader. The tests of `internal/indexer` and of `phpextract` use the reader in place of their own copies.
- `internal/fsatomic`: the atomic exchange of two directories, split by platform, and a rename of a directory. Each one runs `fsync` on the parent of each path. `scratchdir` uses the rename.
- `internal/flock`, new: the `flock` helper of `wtindex`, with a shared or exclusive mode, for `scratchdir` and `graphstore`.
- `internal/scratchdir`, new: the locked scratch directory that `wtindex` uses today, moved out so that `internal/indexer` can use it for the scratch directory of a store. `internal/wtindex` uses it from there and stops deleting the store before its full rebuild.
- `internal/cli`: `index.go` no longer deletes the store, reads the prior coverage generation or checks the lock itself. A new test holds the writer lock and checks the remedy text. `index.go`, `init.go`, `sync.go` and `daemon.go` register `--workers` through one helper with one help text.
- `internal/gencorpus`, new: the corpus generator of `tools/bench/gencorpus`, moved so that a test can import it. The command and `tools/bench/runner` import it, and the runner keeps its subprocess build. `tools/bench/BASELINE.md` names the new path. The comment of `tools/bench/cpudiag` names the new worker default.
- `internal/daemon`: the package comment names the boundary between `daemon.lock` and the writer lock.
- Disk: during a full index, the scratch directory needs about as much disk space again as the live store.
- Dependencies: none new. `golang.org/x/sync`, `golang.org/x/sys`, `go.uber.org/goleak` and `pgregory.net/rapid` are already in `go.mod`. `testing/synctest` is in the standard library.

## Out of scope

- The single atomic commit of an incremental `codegraph sync` (D-01b). The changed files bound its size. The Sync backfill path calls the same run as `codegraph index`, so it gets the new pipeline.
- The memory of the run-wide symbol index itself. It holds every node, as it does today, and it still grows with the repository.
- Bulk loads of sorted SSTables through Pebble's ingest, and concurrent batch commits.
- Tracking benchmark results over time (#269).
- Native Windows. The atomic exchange uses Linux `renameat2` and macOS `renamex_np`.
