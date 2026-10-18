# agent-nudges delta

## Purpose

When the SessionStart and PreToolUse hooks that codegraph installs for coding agents nudge the agent toward codegraph, and the cost they may add in a project that has no index.

## ADDED Requirements

### Requirement: A nudge fires only in an indexed project directory (NUDGE-01, NUDGE-02, NUDGE-04, CODEX-05)

A nudge hook SHALL fire only when its project directory is indexed: `$CLAUDE_PROJECT_DIR` for a Claude Code hook (the current directory when it is unset), and the event's `cwd` for a Codex hook. A directory is indexed when it holds `.codegraph/`, or when it is the root of a linked worktree whose main worktree holds `.codegraph/` and is not the home directory. Every hook SHALL exit 0.

#### Scenario: An indexed project directory

- **WHEN** the project directory holds `.codegraph/`
- **THEN** the SessionStart nudge prints its line, and a matching PreToolUse call gets the nudge as additional context

#### Scenario: A linked worktree of an indexed main checkout

- **WHEN** the project directory is the root of a linked worktree with no `.codegraph/`, and its main worktree holds `.codegraph/`
- **THEN** the SessionStart nudge prints its line, and a matching PreToolUse call gets the nudge from the Claude Code hook
- **AND** the Codex hook gives the nudge when git is on its `PATH`

#### Scenario: A Codex hook with no git on its PATH

- **WHEN** the Codex hook runs in the root of a linked worktree with no `.codegraph/`, and no git is on its `PATH`
- **THEN** it prints nothing and exits 0

#### Scenario: A linked worktree of an unindexed main checkout

- **WHEN** the project directory is the root of a linked worktree, and neither it nor its main worktree holds `.codegraph/`
- **THEN** no hook prints anything, and each exits 0

#### Scenario: Git is unavailable in a linked worktree

- **WHEN** the project directory is the root of a linked worktree, and git cannot run
- **THEN** no hook prints anything, and each exits 0

### Requirement: The index check adds no cost outside a linked worktree (NUDGE-02)

When the project directory holds no `.codegraph/` and its `.git` entry is not a file, every nudge hook SHALL decide not to fire without running git or the codegraph binary. In a linked worktree, the SessionStart hook SHALL run at most one git command and SHALL NOT run the codegraph binary.

#### Scenario: An ordinary repository without an index

- **WHEN** the project directory is an unindexed repository whose `.git` is a directory
- **THEN** no hook runs git or the codegraph binary

#### Scenario: SessionStart in a linked worktree

- **WHEN** the SessionStart hook runs in the root of a linked worktree with no `.codegraph/`
- **THEN** it runs one git command and does not run the codegraph binary
