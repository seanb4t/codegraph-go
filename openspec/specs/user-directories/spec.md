# user-directories Specification

## Purpose
Where codegraph keeps files that belong to the user rather than to one repository: per-user state such as the daemon registry, and per-user configuration. Both follow the XDG Base Directory layout on macOS and Linux.

## Requirements

### Requirement: Per-user state lives in the XDG state directory

codegraph SHALL keep per-user state under `$XDG_STATE_HOME/codegraph` when `XDG_STATE_HOME` is set to an absolute path, and under `~/.local/state/codegraph` otherwise. It SHALL NOT create or write anything under `~/.codegraph`.

#### Scenario: The variable is unset

- **WHEN** `XDG_STATE_HOME` is unset or empty
- **THEN** the state directory is `<home>/.local/state/codegraph`

#### Scenario: The variable is set

- **WHEN** `XDG_STATE_HOME` is `/tmp/xdg-state`
- **THEN** the state directory is `/tmp/xdg-state/codegraph`

#### Scenario: The variable is relative

- **WHEN** `XDG_STATE_HOME` is `state`, a relative path
- **THEN** the value is ignored and the state directory is `<home>/.local/state/codegraph`

### Requirement: Per-user config lives in the XDG config directory

codegraph SHALL resolve its per-user configuration directory to `$XDG_CONFIG_HOME/codegraph` when `XDG_CONFIG_HOME` is set to an absolute path, and to `~/.config/codegraph` otherwise. Any future user-level setting SHALL be read from and written to this directory and nowhere else.

#### Scenario: The variable is unset

- **WHEN** `XDG_CONFIG_HOME` is unset or empty
- **THEN** the config directory is `<home>/.config/codegraph`

#### Scenario: The variable is set

- **WHEN** `XDG_CONFIG_HOME` is `/tmp/xdg-config`
- **THEN** the config directory is `/tmp/xdg-config/codegraph`

#### Scenario: The variable is relative

- **WHEN** `XDG_CONFIG_HOME` is `cfg`, a relative path
- **THEN** the value is ignored and the config directory is `<home>/.config/codegraph`

### Requirement: The daemon registry is in the state directory (D-04)

The registry of running daemons, one `<pid>.json` record per daemon, SHALL live at `<state dir>/daemons/`. Registering, deregistering and listing daemons SHALL all use this directory.

#### Scenario: A daemon starts

- **WHEN** `codegraph daemon start` or `codegraph serve --mcp` starts the watcher for a repository
- **THEN** its record is written to `<state dir>/daemons/<pid>.json`

#### Scenario: Daemons are listed

- **WHEN** `codegraph daemon` lists running daemons
- **THEN** it reads only `<state dir>/daemons/`

### Requirement: Records in the old registry location migrate once

When `<state dir>/daemons/` does not exist and `~/.codegraph/daemons/` does, codegraph SHALL move every `<pid>.json` file from the old directory to the new one before it first reads or writes the registry. It SHALL then remove the old `daemons/` directory, and remove `~/.codegraph` only when that directory is empty. It SHALL NOT delete any other file.

#### Scenario: Only the registry is in the old directory

- **WHEN** `~/.codegraph` holds only `daemons/` with two records and the new registry does not exist
- **THEN** both records are in `<state dir>/daemons/` and `~/.codegraph` no longer exists

#### Scenario: Other files share the old directory

- **WHEN** `~/.codegraph` holds `daemons/` and a file codegraph did not write
- **THEN** the records move, `~/.codegraph/daemons/` is removed, and `~/.codegraph` with its other file stays

#### Scenario: The new registry already exists

- **WHEN** `<state dir>/daemons/` exists and `~/.codegraph/daemons/` still holds records
- **THEN** nothing is moved or deleted

#### Scenario: Nothing to migrate

- **WHEN** `~/.codegraph/daemons/` does not exist
- **THEN** the registry is used at its new location with no other filesystem change
