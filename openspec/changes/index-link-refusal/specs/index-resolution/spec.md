## ADDED Requirements

### Requirement: An index that leads outside its directory is refused

A command that reads or builds an index SHALL refuse a directory whose `.codegraph`, or whose `.codegraph/store`, resolves outside that directory. The refusal SHALL be an error, and the command SHALL read, create and write nothing at the resolved target. The upward walk and the linked-worktree fallback SHALL stop at a refused directory, not skip it. A link that resolves inside the directory SHALL work like a real directory.

#### Scenario: A .codegraph link that leads outside

- **WHEN** `<repo>/.codegraph` is a link to `<outside>`, a directory that is not `<repo>` or below it
- **AND** a read command, `codegraph sync` or `codegraph index` runs in `<repo>`
- **THEN** the command exits non-zero with the refusal error, and `<outside>` is unchanged

#### Scenario: A store link that leads outside

- **WHEN** `<repo>/.codegraph` is a directory and `<repo>/.codegraph/store` is a link to `<outside>`
- **THEN** a command that reads or builds the index exits non-zero with the refusal error, and `<outside>` is unchanged

#### Scenario: init through a link that leads outside

- **WHEN** `<repo>/.codegraph` is a link to `<outside>` and `codegraph init` runs in `<repo>`
- **THEN** it exits non-zero with the refusal error, and creates nothing in `<outside>`

#### Scenario: A dangling link is refused, not skipped

- **WHEN** `<parent>` holds `.codegraph/`, and `<parent>/repo/.codegraph` is a link to a missing path outside `<parent>/repo`
- **AND** a read command starts in `<parent>/repo`
- **THEN** the command exits non-zero with the refusal error, and does not use the index of `<parent>`

#### Scenario: A subdirectory link to an ancestor's index

- **WHEN** `<repo>` holds `.codegraph/`, `<repo>/sub/.codegraph` is a link to `../.codegraph`, and a read command starts in `<repo>/sub`
- **THEN** the command exits non-zero with the refusal error

#### Scenario: The main worktree's index leads outside

- **WHEN** the start directory is a linked worktree with no index of its own
- **AND** the `.codegraph` of its main worktree is a link to a directory outside the main worktree
- **THEN** the command exits non-zero with the refusal error

#### Scenario: serve refuses to start

- **WHEN** `<repo>/.codegraph` is a link to `<outside>`, and `codegraph serve --mcp` starts in `<repo>`
- **THEN** it exits non-zero with the refusal error before it serves a request, and `<outside>` is unchanged

#### Scenario: A .codegraph link that stays inside

- **WHEN** `<repo>/.codegraph` is a link to `<repo>/.cg-real`, a directory
- **THEN** `<repo>` is the index root, and commands read and write the index in `<repo>/.cg-real`

#### Scenario: A store link that stays inside

- **WHEN** `<repo>/.codegraph` is a directory and `<repo>/.codegraph/store` is a link to `<repo>/.store-real`
- **THEN** commands read and write the store in `<repo>/.store-real`

### Requirement: The refusal names the link, its target and the next step

The refusal error SHALL name the link as it is written below the directory that holds it, the resolved target, and that directory. The error SHALL tell the user to remove the link or point it inside that directory.

#### Scenario: The text of the refusal

- **WHEN** a command refuses `<repo>/.codegraph`, a link to the missing path `<tmp>/victim`
- **THEN** the error names `<repo>/.codegraph`, the resolved `<tmp>/victim` and the resolved `<repo>`, and contains `remove the link or point it inside that directory`

### Requirement: A running watcher writes only to the index it started with

A watcher SHALL resolve `.codegraph` and its store again before it takes its lock file and before each flush. This applies to the watcher of `codegraph serve --mcp` and of `codegraph daemon start`. If the resolution fails, is refused, or names other paths than at start, the watcher SHALL write nothing to the index, SHALL stop, and SHALL report the reason. **BREAKING**: a move of the index inside the repository also stops the watcher.

#### Scenario: A pull turns .codegraph into a link that leads outside

- **WHEN** a watcher runs in `<repo>`, `<repo>/.codegraph` is then replaced by a link to `<outside>`, and a source file in `<repo>` changes
- **THEN** the watcher stops with the refusal error, and `<outside>` is unchanged

#### Scenario: A pull turns the store into a link that leads outside

- **WHEN** a watcher runs in `<repo>`, `<repo>/.codegraph/store` is then replaced by a link to `<outside>`, and a source file in `<repo>` changes
- **THEN** the watcher stops with the refusal error, and `<outside>` is unchanged

#### Scenario: The index moves inside the repository

- **WHEN** a watcher runs in `<repo>` with a real `<repo>/.codegraph` directory
- **AND** `<repo>/.codegraph` is then replaced by a link to `<repo>/.cg-real`, and a source file in `<repo>` changes
- **THEN** the watcher stops with an error that names the path at start and the new path
- **AND** the error tells the user to restart the watcher, and nothing under `<repo>/.cg-real` changes

#### Scenario: The store moves inside the repository

- **WHEN** a watcher runs in `<repo>` with a real `<repo>/.codegraph/store` directory
- **AND** `<repo>/.codegraph/store` is then replaced by a link to `<repo>/.store-real`, and a source file in `<repo>` changes
- **THEN** the watcher stops with an error that names both store paths, and nothing under `<repo>/.store-real` changes

#### Scenario: The index can no longer be resolved

- **WHEN** a watcher runs in `<repo>`, `<repo>/.codegraph` is then replaced by a link to itself, and a source file in `<repo>` changes
- **THEN** the watcher stops with the resolve error, and does not sync

#### Scenario: A deferred watcher retries after a pull

- **WHEN** the watcher of `codegraph serve --mcp` in `<repo>` waits because another live daemon holds the lock
- **AND** `<repo>/.codegraph` is then replaced by a link to `<outside>`, which holds a `daemon.lock` file
- **THEN** at its next retry the watcher stops with the refusal error, and `<outside>/daemon.lock` is unchanged

#### Scenario: The index is unchanged

- **WHEN** a watcher runs in `<repo>`, and a source file changes to add the function `added`
- **AND** `.codegraph` and its store resolve to the paths they had at start
- **THEN** after the flush, `codegraph search added` in `<repo>` finds the function

#### Scenario: daemon start reports the stop

- **WHEN** `codegraph daemon start` runs in `<repo>`, `<repo>/.codegraph` is then replaced by a link to `<outside>`, and a source file in `<repo>` changes
- **THEN** the command prints the refusal error on stderr and exits 1

#### Scenario: serve keeps serving after its watcher stops

- **WHEN** `codegraph serve --mcp` runs in `<repo>`, `<repo>/.codegraph` is then replaced by a link to `<outside>`, and a source file in `<repo>` changes
- **THEN** stderr has the line `codegraph serve: watcher: ` followed by the refusal error
- **AND** the server process keeps running

### Requirement: A stopping watcher removes only its own lock file

When a watcher stops for any reason, it SHALL remove its lock file only if `.codegraph` still resolves to the directory it had at start and the lock file names the watcher's own process. Otherwise, the watcher SHALL leave the lock file in place.

#### Scenario: Shutdown after .codegraph became a link to another project

- **WHEN** a watcher runs in `<repo>`, and `<repo>/.codegraph` is then replaced by a link to `<other>/.codegraph`, which holds a `daemon.lock` file
- **AND** the watcher receives SIGTERM
- **THEN** `<other>/.codegraph/daemon.lock` still exists with its content unchanged

#### Scenario: Shutdown after only the store changed

- **WHEN** a watcher runs in `<repo>`, and `<repo>/.codegraph/store` is then replaced by a link to `<outside>`
- **AND** the watcher stops
- **THEN** the watcher removes its own `<repo>/.codegraph/daemon.lock`

#### Scenario: Shutdown with the index unchanged

- **WHEN** a watcher in `<repo>` receives SIGTERM, and `.codegraph` and its store resolve to the paths they had at start
- **THEN** the watcher removes its `<repo>/.codegraph/daemon.lock`
