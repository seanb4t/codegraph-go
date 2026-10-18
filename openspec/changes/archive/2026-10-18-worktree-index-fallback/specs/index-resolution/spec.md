# index-resolution delta

## MODIFIED Requirements

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

## ADDED Requirements

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
