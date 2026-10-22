# agent-install Specification

## Purpose
What codegraph records for a coding agent when `codegraph install` or the refresh after `codegraph upgrade` configures it, starting with which binary path may be written into the agent's configuration.

## Requirements

### Requirement: Only a durable binary path is recorded (D-04)

An install or refresh SHALL record the absolute path of the binary it runs from in every agent configuration and guard it writes. It SHALL refuse an ephemeral path: a base name ending in `.test`, a path element that is `go-build` plus digits (a Go work directory), or a path element of 64 hex digits plus `-d` (a Go build-cache executable directory). A refusal SHALL write nothing for that agent and SHALL name the path.

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
