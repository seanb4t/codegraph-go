# mcp-server delta

## ADDED Requirements

### Requirement: A linked worktree gets an index of its own

When the server starts in a git worktree that would use the index of another working tree of the same repository, it SHALL make an index for that worktree at the worktree's root. The index SHALL hold the graph of the worktree's own files. The server SHALL make it from the other working tree's index where it can, SHALL NOT change that index's graph, and SHALL then serve the new index.

#### Scenario: A worktree outside the main checkout

- **WHEN** the server starts in `<other>/wt`, a linked worktree with no index, whose main checkout `<main>` holds an index, and `wt` adds a function `OnlyHere` that `<main>` does not have
- **THEN** `<other>/wt/.codegraph/` exists, a `codegraph_explore` call for `OnlyHere` finds it, and the result carries no worktree notice
- **AND** the graph in `<main>/.codegraph/` is unchanged

#### Scenario: A worktree inside the main checkout

- **WHEN** the server starts in `<main>/.worktrees/wt`, a linked worktree with no index of its own, and `<main>` holds an index
- **THEN** `<main>/.worktrees/wt/.codegraph/` exists and is the index root

#### Scenario: The worktree index matches a fresh index

- **WHEN** the server has made the worktree index, and `codegraph index --force` then runs in a copy of the same worktree
- **THEN** both indexes hold the same nodes and the same edges

#### Scenario: Git does not see the worktree index

- **WHEN** the server has made the worktree index
- **THEN** `git status` in the worktree lists no new path

#### Scenario: The main checkout has no index

- **WHEN** the server starts in a linked worktree, and neither the worktree nor its main checkout holds an index
- **THEN** no `.codegraph/` is created, and the server runs with zero tools

#### Scenario: A submodule

- **WHEN** the server starts in a submodule with no index of its own, inside a super-repository that holds an index
- **THEN** no `.codegraph/` is created in the submodule, and the super-repository's index is the index root

#### Scenario: The worktree already has an index

- **WHEN** the server starts in a linked worktree that holds `.codegraph/`
- **THEN** the server uses that index and makes no other

### Requirement: A worktree index with few changes is ready before the first request

When at most a fixed number of the worktree's files differ from the index the server copies, the server SHALL finish the worktree index before it reads its first request.

#### Scenario: A worktree on the main checkout's commit

- **WHEN** the server starts in a linked worktree with no index, at the commit its indexed main checkout is on, with no uncommitted changes
- **THEN** the first `codegraph_explore` call reads the worktree index and carries no worktree notice

### Requirement: A worktree index with many changes is built in the background

When more files differ than that number, or the other working tree's index cannot be copied, the server SHALL answer `initialize` without waiting for the worktree index. Until it is ready, tool calls SHALL use the other working tree's index, with the worktree notice. After it is ready, tool calls SHALL use it with no restart or reconnect.

#### Scenario: Many changed files

- **WHEN** the server starts in a linked worktree with no index, and more files differ from the main checkout's index than the server's limit
- **THEN** `initialize` answers before the worktree index is ready
- **AND** a `codegraph_explore` call made before it is ready begins with the worktree notice
- **AND** a `codegraph_explore` call made after it is ready reads the worktree index and carries no notice

#### Scenario: The main checkout's store is held by another process

- **WHEN** the server starts in a linked worktree with no index, and another codegraph process holds the main checkout's store for longer than the server waits
- **THEN** the server builds the worktree index from the worktree's files alone, in the background

#### Scenario: Stderr reports the background build

- **WHEN** the server builds the worktree index in the background
- **THEN** stderr gets one line when the build starts and one when the index is ready

### Requirement: Turning the worktree index off

`--no-worktree-index` or `CODEGRAPH_NO_WORKTREE_INDEX=1` SHALL stop the server from making a worktree index. The server SHALL then use the other working tree's index, with the worktree notice.

#### Scenario: The flag

- **WHEN** `codegraph serve --mcp --no-worktree-index` starts in a linked worktree with no index, whose main checkout holds one
- **THEN** no `.codegraph/` is created in the worktree, and a `codegraph_explore` result begins with the worktree notice

#### Scenario: The environment variable

- **WHEN** the server starts there with `CODEGRAPH_NO_WORKTREE_INDEX=1`
- **THEN** no `.codegraph/` is created in the worktree

#### Scenario: Other values

- **WHEN** `CODEGRAPH_NO_WORKTREE_INDEX` is set to anything other than `1`
- **THEN** it has no effect

### Requirement: Worktree index failures

If the worktree index cannot be made, the server SHALL say why on stderr and SHALL keep serving from the other working tree's index, with the worktree notice. A tool call SHALL NOT read a worktree index that is not finished.

#### Scenario: A read-only worktree

- **WHEN** the server starts in a linked worktree with no index, and the worktree root is not writable
- **THEN** stderr says that the worktree index could not be made and why, and tool calls succeed with the worktree notice

#### Scenario: Another process makes the index first

- **WHEN** a `.codegraph/` appears at the worktree root while the server builds its worktree index
- **THEN** the server uses that `.codegraph/`, leaves it unchanged, and discards its own build

#### Scenario: A build in progress

- **WHEN** a tool call runs while the worktree index is being built
- **THEN** the call reads the other working tree's index, not a part of the new one

#### Scenario: A build the server did not finish

- **WHEN** a server stops while it builds a worktree index, and a new server then starts in the same worktree
- **THEN** the new server removes what the first one left and makes the worktree index

## MODIFIED Requirements

### Requirement: Finding the index (MCP-03, AGNT-03)

The server SHALL take its start directory from `--path` (`-p`) or the current directory, and SHALL resolve the index root from it as index resolution defines, after it makes any worktree index. When the index is at or above the start directory, the index root SHALL also be the confinement root. When it comes from the linked-worktree fallback, the confinement root SHALL be the linked worktree's root. If no index resolves, it SHALL run with no index.

#### Scenario: A subdirectory of an indexed repository

- **WHEN** the server starts in a subdirectory of an indexed repository
- **THEN** the repository root is the index root and the tools are offered

#### Scenario: A linked worktree without its own index

- **WHEN** the server starts in a linked worktree that has no index, under a main checkout that has one, with the worktree index turned off
- **THEN** the main checkout is the index root, and the worktree stays the default `path` for tool calls

#### Scenario: A linked worktree outside the main checkout

- **WHEN** the server starts in `<other>/wt`, a linked worktree with no index at or above it, whose main checkout `<main>` holds an index, with the worktree index turned off
- **THEN** `<main>` is the index root, the tools are offered, and a `codegraph_explore` call without `path` succeeds
- **AND** the startup sync and the file watcher work on `<main>`

#### Scenario: A linked worktree that gets its own index

- **WHEN** the server starts in `<other>/wt`, a linked worktree with no index at or above it, whose main checkout `<main>` holds an index, and the worktree index is ready before the first request
- **THEN** `<other>/wt` is the index root and the confinement root
- **AND** the startup sync and the file watcher work on `<other>/wt`

#### Scenario: A path outside the linked worktree

- **WHEN** the server runs in that linked worktree and a tool call passes a `path` in `<main>` or anywhere else outside `<other>/wt`
- **THEN** the result has `isError: true` and says the path is outside the server's configured repo root

#### Scenario: No index anywhere above

- **WHEN** no directory at or above the start directory holds an index, and the linked-worktree fallback finds none
- **THEN** the server runs with zero tools

### Requirement: Worktree notice on a borrowed index (WORK-01, WORK-02, WORK-03)

When the start directory is a git worktree that uses another working tree's index, every successful tool result except `codegraph_status` SHALL begin with a one-line notice, and `codegraph_status` SHALL include its own warning instead. Error results SHALL carry no notice.

#### Scenario: A worktree borrowing the main checkout's index

- **WHEN** the server runs in a linked worktree with no index of its own, with the worktree index turned off
- **THEN** a successful `codegraph_explore` result begins with the notice
- **AND** the same call from the main checkout has no notice

#### Scenario: A worktree outside the main checkout

- **WHEN** the server runs in a linked worktree outside the main checkout and uses the main checkout's index through the linked-worktree fallback, with the worktree index turned off
- **THEN** a successful `codegraph_explore` result begins with the notice, and `codegraph_status` includes the warning

#### Scenario: A worktree index not yet ready

- **WHEN** the server runs in a linked worktree whose own index is still being built in the background
- **THEN** a successful `codegraph_explore` result begins with the notice

#### Scenario: A failed call

- **WHEN** a tool call fails in such a worktree
- **THEN** the error result carries no notice

### Requirement: Watcher limits within a session

The watcher decision SHALL be made once, at startup: an index created during a session by another command gets tools but no watcher until the next session. A worktree index that the server itself builds in the background SHALL get the watcher when it is ready. Watcher failures SHALL be reported on stderr and SHALL NOT stop the server.

#### Scenario: An index created mid-session

- **WHEN** `codegraph init` runs during a session that started without an index
- **THEN** the tools appear, but no watcher starts in that session

#### Scenario: A worktree index built in the background

- **WHEN** the server finishes a worktree index in the background, and the watcher is not disabled
- **THEN** a file the user then edits in the worktree reaches a later tool call after the debounce window

#### Scenario: The watcher fails

- **WHEN** the file watcher cannot start or its event stream fails
- **THEN** stderr reports the watcher error, and tool calls keep working
