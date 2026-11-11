# Spec Delta

## Purpose

How a full index runs. It covers the limit on the work in flight and on each write batch. It also covers how a new store replaces the old one and how a run keeps other writers out. Last, it covers how a run ends on an error.

## ADDED Requirements

### Requirement: Work in flight is bounded by the worker count
A full index reads each file in three passes: extract, structure and resolve. Route detection reads files between the first two. In each of these steps, a file holds working state from the moment a worker claims it until the step is done with it. A full index SHALL let at most W files hold working state at any time. For a `--workers` value above zero, W SHALL be that value. Otherwise W SHALL be `runtime.GOMAXPROCS(0)`.

#### Scenario: Where working state ends in each pass
- **WHEN** a full index with 2 workers runs over a tree of 3 Go files and one Express file
- **AND** it reports that a file stopped holding working state in a step
- **THEN** for the extract pass, a read of the spill file at the offset of that file returns its record
- **AND** for route detection, the route result of the run holds the route nodes of that file
- **AND** for the structure pass, the structural edge list of the run holds the edges of that file
- **AND** for the resolve pass, after the run ends, the store holds the records of that file

#### Scenario: Two workers over many files
- **WHEN** `codegraph index --workers 2` runs over a tree of 500 Go source files
- **THEN** no more than 2 files hold working state at the same time in any pass
- **AND** 2 files hold working state at the same time at least once

#### Scenario: Default worker count follows the CPU limit
- **WHEN** `codegraph index` runs without `--workers`, or with `--workers 0` or `--workers -1`, in a process whose `GOMAXPROCS` is 3
- **THEN** no more than 3 files hold working state at the same time
- **AND** 3 files hold working state at the same time at least once

#### Scenario: One worker
- **WHEN** `codegraph index --workers 1` runs in a process whose `GOMAXPROCS` is 3
- **THEN** no more than 1 file holds working state at any time

#### Scenario: More workers than files
- **WHEN** a full index runs with a worker count of 2^40 over a tree of 3 Go source files
- **THEN** the run succeeds, and no more than 3 files hold working state at the same time

### Requirement: Write batches are bounded
A full index SHALL write the graph in batches under a fixed cap. Before it adds a record to a batch that is already above the cap, the writer SHALL commit that batch. So a batch SHALL exceed the cap by at most its last record. Each batch SHALL hold at least one record. A run SHALL commit at least one batch, the one that holds the metadata record.

#### Scenario: A large graph spans several batches
- **WHEN** a full index of a tree of 500 Go source files runs with a batch cap of 64 KiB
- **THEN** it commits more than one batch
- **AND** each batch without its last record is at or below 64 KiB, and no batch is empty

#### Scenario: A small graph still commits
- **WHEN** a full index runs over a tree with one Go source file
- **THEN** it commits at least one batch, and the store holds the node of that file

#### Scenario: An empty tree
- **WHEN** a full index runs over a tree with no source files
- **THEN** it commits exactly one batch, which holds the metadata record

### Requirement: An import writes in bounded batches
`graphstore.Import` SHALL write the records of an export stream in batches under the same cap as a full index. When the stream holds a metadata record, `Import` SHALL commit it in its last batch. `Import` SHALL commit no empty batch.

#### Scenario: A large export
- **WHEN** `graphstore.Import` reads an export of 5,000 nodes into an empty store with a batch cap of 64 KiB
- **THEN** it commits more than one batch, and each batch without its last record is at or below 64 KiB
- **AND** a new export of that store is byte-identical to the input

#### Scenario: An empty export
- **WHEN** `graphstore.Import` reads an export stream with no records into an empty store
- **THEN** it returns no error, commits no batch, and the store holds no record

### Requirement: A failed full index leaves the previous store unchanged
A full index SHALL build the new store apart from the live store and SHALL replace the live store in one atomic step. A run that fails, is cancelled or stops halfway SHALL leave the previous store readable, with the same records it held before the run.

#### Scenario: A commit error halfway through
- **WHEN** a full index of an indexed tree of 500 Go source files runs with a batch cap of 64 KiB
- **AND** its third batch commit returns an error
- **THEN** the run returns that error
- **AND** the store holds the same records as before the run, its metadata record included

#### Scenario: The exchange cannot take the live store
- **WHEN** a full index of an indexed tree runs
- **AND** another handle holds the store open from the start of the resolve pass until the run returns
- **THEN** the run returns the store-locked error
- **AND** the store holds the same records as before the run, and the `.codegraph` directory holds no scratch directory of that run

#### Scenario: The process is killed before the exchange
- **WHEN** a full index of an indexed tree of 500 Go source files runs with a batch cap of 64 KiB
- **AND** its process gets `SIGKILL` after its second batch commit
- **THEN** the next command that opens the store reads the records that the store held before the run

#### Scenario: The process is killed after the exchange
- **WHEN** the process of a full index gets `SIGKILL` right after the exchange and before it removes the old store
- **THEN** the next command that opens the store reads the new graph

#### Scenario: A first index fails
- **WHEN** a tree of 500 Go source files has a `.codegraph` directory and no indexed graph
- **AND** a full index of that tree runs with a batch cap of 64 KiB, and its third batch commit returns an error
- **THEN** the run returns that error, and the store holds no node, edge, file record or metadata record

#### Scenario: A reader opens the store during a first index
- **WHEN** a full index runs over a tree with no store, and another handle opens and closes the store during the extract pass
- **THEN** the run succeeds, and the store holds the new graph

#### Scenario: A full index succeeds
- **WHEN** a tree is indexed, one of its files is deleted, and a full index runs again
- **THEN** the store holds the same records as a fresh index of the changed tree, and no record of the deleted file remains

### Requirement: A full index is the only writer of its store
While a full index runs, no other writer SHALL commit to its store. A sync that starts during the run SHALL fail with the store-locked error and SHALL leave the store unchanged. A read of the store during the run SHALL see the previous graph. The run SHALL write a coverage generation one more than the generation that the store held at the start of the run.

#### Scenario: A sync during a full index
- **WHEN** a sync of the same tree starts while a full index of its store is in the extract pass
- **THEN** the sync returns the store-locked error
- **AND** after the full index ends, the store holds its graph, with a coverage generation one more than before

#### Scenario: A read during a full index
- **WHEN** a handle opens and reads the store while a full index of an indexed tree is in its resolve pass
- **THEN** it reads the same records as before the full index started

### Requirement: A full index leaves no scratch data
After a full index ends, with success or with an error, no scratch directory or spill file of that run SHALL remain. A run creates its scratch directory under a pending name, and renames it after it holds the lock of that directory. The next full index SHALL remove the scratch directories that a killed run left. A full index SHALL NOT remove a scratch directory whose lock another process holds.

#### Scenario: After a commit error
- **WHEN** a full index of a tree of 500 Go source files runs with a batch cap of 64 KiB
- **AND** its third batch commit returns an error
- **THEN** the `.codegraph` directory holds no scratch directory of that run

#### Scenario: After a success
- **WHEN** a full index succeeds
- **THEN** the `.codegraph` directory holds no scratch directory, and the store directory holds no spill file

#### Scenario: After a killed run
- **WHEN** a full index of a tree of 500 Go source files runs with a batch cap of 64 KiB
- **AND** its process gets `SIGKILL` after its second batch commit, and the next full index runs to its end
- **THEN** the scratch directory of the killed run does not exist, and the store holds the records of a fresh index

#### Scenario: A run killed while it created its scratch directory
- **WHEN** the `.codegraph` directory holds a pending scratch directory whose lock file no process holds, and a full index runs to its end
- **THEN** that pending directory does not exist

#### Scenario: A scratch directory that another process holds
- **WHEN** another process holds the lock file of a scratch directory in the `.codegraph` directory, and a full index runs to its end
- **THEN** that scratch directory and its lock file still exist

### Requirement: The first error stops the whole run
When any worker or the writer of a full index fails, the run SHALL stop every pass and SHALL return that first error. After the run returns, no goroutine of the run SHALL remain.

#### Scenario: A writer error
- **WHEN** a full index of a tree of 500 Go source files runs with a batch cap of 64 KiB
- **AND** its second batch commit returns an error while workers still resolve files
- **THEN** the run returns that commit error
- **AND** after it returns, no goroutine of the run remains

#### Scenario: A worker error
- **WHEN** a full index runs, and one worker returns an internal error for one file
- **AND** the step of that worker is the extract pass, route detection, the structure pass or the resolve pass
- **THEN** the run returns that error
- **AND** after it returns, no goroutine of the run remains, and the `.codegraph` directory holds no scratch directory of that run

#### Scenario: A cancel while workers wait
- **WHEN** the writer stops reading, every worker waits for a token, and the context of the run is cancelled
- **THEN** the run returns the cancel error
- **AND** after it returns, no goroutine of the run remains
