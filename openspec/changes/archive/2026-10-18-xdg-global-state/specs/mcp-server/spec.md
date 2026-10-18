# mcp-server delta

## MODIFIED Requirements

### Requirement: No index means no tools (MCP-03)

When no index resolves, the server SHALL still start and complete the handshake, SHALL list zero tools, and SHALL run no startup sync and no file watcher. A `.codegraph` directory in the user's home directory SHALL NOT count as an index.

#### Scenario: Listing without an index

- **WHEN** no index resolves, even with `CODEGRAPH_MCP_TOOLS` set
- **THEN** `initialize` succeeds and `tools/list` returns an empty list

#### Scenario: Calling a tool without an index

- **WHEN** a client calls `codegraph_explore` while no index resolves
- **THEN** the server answers with JSON-RPC error `-32602` for an unknown tool

#### Scenario: Started under the home directory while `~/.codegraph` exists

- **WHEN** `codegraph serve --mcp` starts in `<home>` or in `<home>/x/unindexed-repo` and `<home>/.codegraph` exists
- **THEN** `initialize` succeeds with zero tools, no startup sync walks `<home>`, and `<home>/.codegraph/store` is never opened or created
