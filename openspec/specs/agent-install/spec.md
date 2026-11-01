# agent-install Specification

## Purpose
What codegraph records for a coding agent when `codegraph install` or the refresh after `codegraph upgrade` configures it, starting with which binary path may be written into the agent's configuration.

## Requirements

### Requirement: Only a durable binary path is recorded (D-04)

An install or refresh SHALL record the absolute path of the binary it runs from, or its stable link, in every agent configuration and guard it writes. It SHALL refuse an ephemeral path: a base name ending in `.test`, a path element that is `go-build` plus digits (a Go work directory), or a path element of 64 hex digits plus `-d` (a Go build-cache executable directory). A refusal SHALL write nothing for that agent and SHALL name the path.

#### Scenario: A real binary

- **WHEN** `codegraph install` runs from `/usr/local/bin/codegraph`
- **THEN** every MCP entry and guard it writes names `/usr/local/bin/codegraph`

#### Scenario: A Go test binary

- **WHEN** `codegraph install` runs from `<tmp>/go-build123/b001/cli.test`
- **THEN** it exits non-zero with one error that names the path, and no agent configuration changes

#### Scenario: A `go run` binary

- **WHEN** `codegraph install` runs from `<tmp>/go-build123/exe/codegraph`
- **THEN** it exits non-zero with one error that names the path, and no agent configuration changes

#### Scenario: A cached `go run` binary

- **WHEN** `codegraph install` runs from `<cache>/69/<64 hex digits>-d/codegraph`
- **THEN** it exits non-zero with one error that names the path, and no agent configuration changes

#### Scenario: The refresh after an upgrade

- **WHEN** the refresh after `codegraph upgrade` is given an ephemeral path for a configured location
- **THEN** that location is not rewritten and the refresh reports the refusal as a warning

### Requirement: The ephemeral-path check has an explicit override

When the environment variable `CODEGRAPH_ALLOW_EPHEMERAL_EXEC_PATH` is `1`, an install or refresh SHALL record an ephemeral path as if it were durable.

#### Scenario: Override set

- **WHEN** `CODEGRAPH_ALLOW_EPHEMERAL_EXEC_PATH=1` and `codegraph install` runs from `<tmp>/go-build123/b001/cli.test`
- **THEN** the MCP entries name that path and the command exits zero

### Requirement: A versioned package-manager path is recorded through its stable link

If a package manager's stable link resolves to the same file as the binary, an install or refresh SHALL record the link in place of the versioned path. The versioned paths are `<dir>/installs/<tool>/<version>/<rest>` (mise and asdf), whose stable link is `<dir>/installs/<tool>/latest/<rest>`, and `<prefix>/Caskroom/<token>/<version>/<name>` (a Homebrew cask), whose stable link is `<prefix>/bin/<name>`. A `<version>` element of `latest` is not a versioned path.

#### Scenario: mise `latest` names the running version

- **WHEN** `codegraph install` runs from `<data>/installs/codegraph/0.16.3/codegraph` and `<data>/installs/codegraph/latest` is a link to `./0.16.3`
- **THEN** every MCP entry and guard it writes names `<data>/installs/codegraph/latest/codegraph`, and it prints no warning

#### Scenario: mise `latest` names another version

- **WHEN** `codegraph install` runs from `<data>/installs/codegraph/0.15.0/codegraph` and `<data>/installs/codegraph/latest` is a link to `./0.16.3`
- **THEN** every MCP entry and guard it writes names `<data>/installs/codegraph/0.15.0/codegraph`

#### Scenario: A Homebrew cask with its `bin` link

- **WHEN** `codegraph install` runs from `<prefix>/Caskroom/codegraph/0.16.3/codegraph` and `<prefix>/bin/codegraph` is a link to that file
- **THEN** every MCP entry and guard it writes names `<prefix>/bin/codegraph`

#### Scenario: Already the stable link

- **WHEN** `codegraph install` runs from `<data>/installs/codegraph/latest/codegraph`
- **THEN** every MCP entry and guard it writes names that path, and it prints no warning

#### Scenario: The refresh after an upgrade

- **WHEN** the refresh after `codegraph upgrade` is given `<data>/installs/codegraph/0.16.3/codegraph` and `latest` is a link to `./0.16.3`
- **THEN** every location it rewrites names `<data>/installs/codegraph/latest/codegraph`

### Requirement: A versioned path without a stable link is recorded with a warning

When `codegraph install` records a versioned package-manager path because no stable link resolves to the same file, it SHALL print one warning line to stderr. The warning SHALL name the path and SHALL say that a package-manager upgrade or prune can break it. The install SHALL still write every configuration, stdout SHALL not change, and the exit status SHALL not change.

#### Scenario: asdf, which keeps no `latest` link

- **WHEN** `codegraph install --target opencode` runs from `~/.asdf/installs/codegraph/0.16.3/bin/codegraph` and `~/.asdf/installs/codegraph/latest` does not exist
- **THEN** the opencode entry names that path, stderr holds exactly one warning that names it, and the command exits zero

#### Scenario: Several targets in one run

- **WHEN** `codegraph install --target all` runs from a versioned path with no stable link
- **THEN** stderr holds exactly one warning, not one per agent

#### Scenario: A path that is not versioned

- **WHEN** `codegraph install` runs from `/usr/local/bin/codegraph`
- **THEN** stderr holds no warning
