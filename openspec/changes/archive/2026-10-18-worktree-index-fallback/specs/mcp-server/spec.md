# mcp-server delta

## MODIFIED Requirements

### Requirement: Finding the index (MCP-03, AGNT-03)

The server SHALL take its start directory from `--path` (`-p`) or the current directory, and SHALL resolve the index root from it as index resolution defines. When the index is at or above the start directory, the index root SHALL also be the confinement root. When it comes from the linked-worktree fallback, the confinement root SHALL be the linked worktree's root. If no index resolves, it SHALL run with no index.

#### Scenario: A subdirectory of an indexed repository

- **WHEN** the server starts in a subdirectory of an indexed repository
- **THEN** the repository root is the index root and the tools are offered

#### Scenario: A linked worktree without its own index

- **WHEN** the server starts in a linked worktree that has no index, under a main checkout that has one
- **THEN** the main checkout is the index root, and the worktree stays the default `path` for tool calls

#### Scenario: A linked worktree outside the main checkout

- **WHEN** the server starts in `<other>/wt`, a linked worktree with no index at or above it, whose main checkout `<main>` holds an index
- **THEN** `<main>` is the index root, the tools are offered, and a `codegraph_explore` call without `path` succeeds
- **AND** the startup sync and the file watcher work on `<main>`

#### Scenario: A path outside the linked worktree

- **WHEN** the server runs in that linked worktree and a tool call passes a `path` in `<main>` or anywhere else outside `<other>/wt`
- **THEN** the result has `isError: true` and says the path is outside the server's configured repo root

#### Scenario: No index anywhere above

- **WHEN** no directory at or above the start directory holds an index, and the linked-worktree fallback finds none
- **THEN** the server runs with zero tools

### Requirement: Worktree notice on a borrowed index (WORK-01, WORK-02, WORK-03)

When the start directory is a git worktree that uses another working tree's index, every successful tool result except `codegraph_status` SHALL begin with a one-line notice, and `codegraph_status` SHALL include its own warning instead. Error results SHALL carry no notice.

#### Scenario: A worktree borrowing the main checkout's index

- **WHEN** the server runs in a linked worktree with no index of its own
- **THEN** a successful `codegraph_explore` result begins with the notice
- **AND** the same call from the main checkout has no notice

#### Scenario: A worktree outside the main checkout

- **WHEN** the server runs in a linked worktree outside the main checkout and uses the main checkout's index through the linked-worktree fallback
- **THEN** a successful `codegraph_explore` result begins with the notice, and `codegraph_status` includes the warning

#### Scenario: A failed call

- **WHEN** a tool call fails in such a worktree
- **THEN** the error result carries no notice
