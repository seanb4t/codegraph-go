# index-resolution Specification

## Purpose
How a codegraph command finds the index it works on from the directory it starts in, and which directories can never be an index root.

## Requirements

### Requirement: The nearest ancestor with an index is the root (D-01a)

A command SHALL resolve its index root by walking from its start directory upward, parent by parent, and SHALL use the first directory on that path that holds a `.codegraph` directory, the start directory included. When the walk reaches the filesystem root without a match, the command SHALL try the linked-worktree fallback, and SHALL report that no index is initialized only when the fallback finds none either.

#### Scenario: The start directory is indexed

- **WHEN** the start directory holds `.codegraph/`
- **THEN** the start directory is the index root

#### Scenario: An ancestor is indexed

- **WHEN** the start directory is `<repo>/a/b` and only `<repo>` holds `.codegraph/`
- **THEN** `<repo>` is the index root

#### Scenario: Nothing is indexed

- **WHEN** no directory from the start directory up to the filesystem root holds `.codegraph/`, and the start directory is not in a linked worktree whose main worktree holds `.codegraph/`
- **THEN** the command reports that no index is initialized

### Requirement: The home directory is never an index root

The upward walk SHALL skip the user's home directory, even when it holds a `.codegraph` directory, and SHALL continue to the parent. Directories below and above the home directory are unaffected.

#### Scenario: Started under the home directory with no index

- **WHEN** `<home>/.codegraph` exists and the start directory is `<home>/x/unindexed-repo`
- **THEN** the command reports that no index is initialized

#### Scenario: Started in the home directory

- **WHEN** `<home>/.codegraph` exists and the start directory is `<home>`
- **THEN** the command reports that no index is initialized

#### Scenario: An indexed project under the home directory

- **WHEN** `<home>/.codegraph` exists and the start directory is `<home>/x/repo/pkg`, where `<home>/x/repo` holds `.codegraph/`
- **THEN** `<home>/x/repo` is the index root

#### Scenario: The home directory cannot be determined

- **WHEN** the user's home directory cannot be resolved
- **THEN** the walk behaves as if no directory were skipped

### Requirement: Index commands refuse the home directory as a target

`codegraph init`, `codegraph index` and `codegraph uninit` SHALL refuse to run when their target directory is the user's home directory, SHALL exit non-zero, and SHALL say that the home directory cannot be a codegraph project. They SHALL create, change or remove nothing in that case.

#### Scenario: init in the home directory

- **WHEN** `codegraph init` runs with the home directory as its target
- **THEN** it exits non-zero with an error that names the home directory, and `<home>/.codegraph` is not created

#### Scenario: uninit in the home directory

- **WHEN** `codegraph uninit --force` runs with the home directory as its target while `<home>/.codegraph` exists
- **THEN** it exits non-zero and `<home>/.codegraph` is unchanged

#### Scenario: A project under the home directory

- **WHEN** `codegraph init` runs with `<home>/x/repo` as its target
- **THEN** it proceeds as usual

### Requirement: A linked worktree falls back to its main worktree's index

When the upward walk finds no index and the nearest `.git` entry at or above the start directory is a file, a read command SHALL ask git for the repository's main worktree. If the main worktree holds `.codegraph/` and is not the user's home directory, it SHALL be the index root. Any other outcome, including a git failure or a repository with no main worktree, SHALL mean no index is initialized.

#### Scenario: A worktree outside the main checkout

- **WHEN** `<main>` holds `.codegraph/` and the start directory is `<other>/wt/pkg`, where `<other>/wt` is a linked worktree of the repository whose main worktree is `<main>`, and no directory above the start directory holds `.codegraph/`
- **THEN** `<main>` is the index root

#### Scenario: A worktree with its own index

- **WHEN** a linked worktree `<other>/wt` holds `.codegraph/` and its main worktree `<main>` holds one too
- **THEN** `<other>/wt` is the index root

#### Scenario: A worktree inside the main checkout

- **WHEN** the start directory is `<main>/.worktrees/wt`, a linked worktree, and `<main>` holds `.codegraph/`
- **THEN** `<main>` is the index root, found by the upward walk

#### Scenario: An indexed ancestor comes first

- **WHEN** the start directory is a linked worktree `<dir>/wt`, `<dir>` holds `.codegraph/`, and the main worktree `<main>` holds one too
- **THEN** `<dir>` is the index root

#### Scenario: The main worktree is not indexed

- **WHEN** the start directory is a linked worktree and its main worktree holds no `.codegraph/`
- **THEN** the command reports that no index is initialized

#### Scenario: A worktree of a bare repository

- **WHEN** the start directory is a linked worktree of a bare repository, so the repository has no main worktree
- **THEN** the command reports that no index is initialized

#### Scenario: The main worktree is the home directory

- **WHEN** the start directory is a linked worktree whose main worktree is `<home>`, and `<home>/.codegraph` exists
- **THEN** the command reports that no index is initialized

#### Scenario: Git is unavailable

- **WHEN** the start directory is a linked worktree whose main worktree holds `.codegraph/`, and git cannot run or does not answer within its timeout
- **THEN** the command reports that no index is initialized, and does not fail in any other way

#### Scenario: A submodule outside any indexed directory

- **WHEN** the start directory is in a submodule, and no directory at or above the submodule holds `.codegraph/`
- **THEN** the command reports that no index is initialized

### Requirement: Only a worktree the repository records borrows its index

The linked-worktree fallback SHALL apply only when the repository records the worktree both ways: the worktree's git directory is in the repository's `worktrees` directory, and that git directory names the worktree's `.git` file back. A `.git` file without that record SHALL NOT make another directory's index the index root.

#### Scenario: A .git file naming another repository's git directory

- **WHEN** the start directory holds a hand-written `.git` file that names `<main>/.git`, `<main>` holds `.codegraph/`, and no directory at or above the start directory holds `.codegraph/`
- **THEN** the command reports that no index is initialized

#### Scenario: A .git file naming another worktree's git directory

- **WHEN** the start directory holds a hand-written `.git` file that names the git directory of a different linked worktree of `<main>`, and `<main>` holds `.codegraph/`
- **THEN** the command reports that no index is initialized

### Requirement: Index-writing commands never use the worktree fallback

`codegraph init`, `codegraph index`, `codegraph sync`, `codegraph uninit` and `codegraph daemon` SHALL act on their target directory only. They SHALL NOT use the worktree fallback, and SHALL NOT read or write the main worktree's index because of it.

#### Scenario: sync in a sibling worktree

- **WHEN** `codegraph sync` runs in a linked worktree that holds no `.codegraph/`, while its main worktree holds one
- **THEN** the main worktree's index is unchanged

#### Scenario: init in a sibling worktree

- **WHEN** `codegraph init` runs in a linked worktree that holds no `.codegraph/`, while its main worktree holds one
- **THEN** the worktree gets its own `.codegraph/`, and later read commands there use it

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
- **AND** the error tells the user to restart the watcher, and no index or store file under `<repo>/.cg-real` changes

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

When a watcher stops for any reason, it SHALL check two things before it removes its lock file: `.codegraph` still resolves inside the repository, and the lock file names the watcher's own process. If either check fails, the watcher SHALL leave the lock file in place.

#### Scenario: Shutdown after .codegraph became a link to another project

- **WHEN** a watcher runs in `<repo>`, and `<repo>/.codegraph` is then replaced by a link to `<other>/.codegraph`, which holds a `daemon.lock` file
- **AND** the watcher receives SIGTERM
- **THEN** `<other>/.codegraph/daemon.lock` still exists with its content unchanged

#### Scenario: Shutdown after only the store changed

- **WHEN** a watcher runs in `<repo>`, and `<repo>/.codegraph/store` is then replaced by a link to `<outside>`
- **AND** the watcher receives SIGTERM
- **THEN** the watcher removes its own `<repo>/.codegraph/daemon.lock`

#### Scenario: Shutdown after the index moved inside the repository

- **WHEN** a watcher runs in `<repo>`, and `<repo>/.codegraph` is then renamed to `<repo>/.cg-real` and replaced by a link to it
- **AND** the watcher receives SIGTERM
- **THEN** the watcher removes its own `<repo>/.cg-real/daemon.lock`

#### Scenario: Shutdown with the index unchanged

- **WHEN** a watcher in `<repo>` receives SIGTERM, and `.codegraph` and its store resolve to the paths they had at start
- **THEN** the watcher removes its `<repo>/.codegraph/daemon.lock`
