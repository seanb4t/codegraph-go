# Spec Delta

## MODIFIED Requirements

### Requirement: A briefly locked store does not fail or hang a call

A tool call SHALL NOT fail or wait because the sync of its own server process has the graph store open. That call SHALL read the graph of one completed sync, never a part of a sync. Two moments keep the bounded retry of a store that another process holds: while a sync puts a rebuilt store of an earlier release in place, and while the server process closes its last handle of the store. When a tool call finds the graph store locked by another process, it SHALL retry for a bounded time. If the store is still locked after the last attempt, it SHALL return a result with `isError: true` instead of waiting indefinitely.

#### Scenario: A short lock

- **WHEN** another process holds the store as a tool call starts and releases it before the last attempt of the retry
- **THEN** the call returns a result without `isError`

#### Scenario: A long lock

- **WHEN** another process keeps the store locked through every attempt of the retry
- **THEN** the call returns a result with `isError: true` that says the store lock is held

#### Scenario: A sync of the same server longer than the retry window

- **GIVEN** the watcher of a server runs a sync that lasts longer than the retry window
- **WHEN** a client of that server calls `codegraph_explore` with the query `Alpha` during the sync
- **THEN** the call returns a result without `isError` while the sync still holds the store open

#### Scenario: A call during one sync

- **GIVEN** the watcher of a server runs one sync that adds `gen_a.go`, which defines `GenA`, and `gen_b.go`, which defines `GenB`
- **WHEN** a client of that server calls `codegraph_search` with the query `Gen` during the sync
- **THEN** the result lists both `GenA` and `GenB`, or it lists neither

#### Scenario: Tool calls back to back during a live edit

- **GIVEN** a client copies 26 Go files into the watched directory `pkga`, and one of them, `zeta.go`, defines `func Zeta`
- **WHEN** the client calls `codegraph_explore` with the query `Zeta` again and again
- **THEN** no call that starts while a sync has the store open returns `isError: true`, and a payload contains `func Zeta` before the 90 s limit of the client

## ADDED Requirements

### Requirement: An edit made while the watcher starts reaches the graph

The server SHALL sync an edit that lands after its startup sync and before its file watcher is ready. That edit SHALL reach later tool calls with no further edit and no manual sync. A watcher that takes over from another session SHALL do the same for an edit made during the takeover. An edit that changes neither the size nor the modification time of a file is outside this rule, as it is for every sync.

#### Scenario: An edit right after the handshake

- **GIVEN** a client writes `pkga/zeta.go`, which defines `func Zeta`, as soon as `initialize` answers
- **WHEN** the client writes nothing more and calls `codegraph_explore` with the query `Zeta` again and again
- **THEN** a payload contains `func Zeta`

#### Scenario: An edit during a takeover

- **GIVEN** the session that runs the watcher ends
- **WHEN** the file `convB.go`, which defines `func ConvB`, appears before the watcher of a second session is ready, and nothing more changes
- **THEN** a `codegraph_explore` call of the second session with the query `ConvB` returns a payload that contains `func ConvB`

#### Scenario: Nothing changed while the watcher started

- **WHEN** no file changed between the startup sync and the moment the watcher is ready
- **THEN** the sync that follows the watcher start reports zero reparsed files

### Requirement: A watcher-start sync that cannot get the store leaves the graph marked stale

If another process holds the store through every retry of the sync that follows the watcher start, the server SHALL keep the graph marked stale until the next edit starts a new sync.

#### Scenario: Another process holds the store while the watcher starts

- **GIVEN** another process held the store through every retry of the sync that follows the watcher start, and then released it
- **WHEN** a client calls `codegraph_status` before any further edit
- **THEN** the result reports the graph as stale
