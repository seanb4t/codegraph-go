# Spec Delta

## MODIFIED Requirements

### Requirement: A nudge fires only in an indexed project directory (NUDGE-01, NUDGE-02, NUDGE-04, CODEX-05)

A nudge hook SHALL fire only when its project directory is indexed: `$CLAUDE_PROJECT_DIR` for a Claude Code hook (the current directory when it is unset), and the event's `cwd` for a Codex hook. A directory is indexed when it holds `.codegraph/`, or when it is the root of a linked worktree whose main worktree holds `.codegraph/` and is not the home directory. Every hook SHALL exit 0.

#### Scenario: An indexed project directory

- **WHEN** the project directory holds `.codegraph/`
- **THEN** the SessionStart nudge prints its line, and a matching PreToolUse call gets the nudge as additional context

#### Scenario: A linked worktree of an indexed main checkout

- **WHEN** the project directory is the root of a linked worktree with no `.codegraph/`, and its main worktree holds `.codegraph/`
- **THEN** the SessionStart nudge prints its line, and a matching PreToolUse call gets the nudge from the Claude Code hook and from the Codex hook

#### Scenario: A Codex hook with no git on its PATH

- **WHEN** the Codex hook runs in the root of a linked worktree with no `.codegraph/`, and the session `PATH` it inherits holds no git
- **THEN** it prints nothing and exits 0

#### Scenario: A linked worktree of an unindexed main checkout

- **WHEN** the project directory is the root of a linked worktree, and neither it nor its main worktree holds `.codegraph/`
- **THEN** no hook prints anything, and each exits 0

#### Scenario: Git is unavailable in a linked worktree

- **WHEN** the project directory is the root of a linked worktree, and git cannot run
- **THEN** no hook prints anything, and each exits 0
