# index-resolution delta

## Purpose

How a codegraph command finds the index it works on from the directory it starts in, and which directories can never be an index root.

## ADDED Requirements

### Requirement: The nearest ancestor with an index is the root (D-01a)

A command SHALL resolve its index root by walking from its start directory upward, parent by parent, and SHALL use the first directory on that path that holds a `.codegraph` directory, the start directory included. When the walk reaches the filesystem root without a match, the command SHALL report that no index is initialized.

#### Scenario: The start directory is indexed

- **WHEN** the start directory holds `.codegraph/`
- **THEN** the start directory is the index root

#### Scenario: An ancestor is indexed

- **WHEN** the start directory is `<repo>/a/b` and only `<repo>` holds `.codegraph/`
- **THEN** `<repo>` is the index root

#### Scenario: Nothing is indexed

- **WHEN** no directory from the start directory up to the filesystem root holds `.codegraph/`
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
