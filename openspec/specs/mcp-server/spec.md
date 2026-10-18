# mcp-server Specification

## Purpose
The MCP server that `codegraph serve --mcp` runs over stdio. It gives agent clients read-only access to a repository's pre-built code graph through a small set of tools and reference resources.

## Requirements

### Requirement: Stdio transport behind an explicit flag (MCP-01)

`codegraph serve` SHALL run the MCP server only when `--mcp` is given, speaking newline-delimited JSON-RPC over stdin and stdout. It SHALL open no network listener, and it SHALL refuse to start without `--mcp`.

#### Scenario: The flag is missing

- **WHEN** `codegraph serve` runs without `--mcp`
- **THEN** it exits with a non-zero status and an error that names `--mcp`

#### Scenario: A stdio session

- **WHEN** `codegraph serve --mcp` runs
- **THEN** it reads JSON-RPC requests on stdin and writes its responses on stdout

### Requirement: Stdout carries only protocol messages (HYG-02, TUI-01)

Every line the server writes to stdout SHALL be a JSON-RPC message. Warnings, banners, the session line and any library logging SHALL go to stderr. Tool result text SHALL never contain terminal escape sequences.

#### Scenario: A session that opens the store

- **WHEN** a session runs a tool call that opens the graph store after the startup sync
- **THEN** every line on stdout parses as a JSON-RPC message

#### Scenario: Styled stderr

- **WHEN** stderr is a terminal and colour output is allowed
- **THEN** only stderr messages are styled, and stdout stays unstyled

### Requirement: Owed responses survive the end of stdin (v0.3.0 SDK-01, v0.12.0 FIX-01)

When stdin reaches end of file, the server SHALL still write the response to every request it has already accepted, waiting at most 5 seconds for them, and SHALL then exit.

#### Scenario: The client closes stdin right after its last request

- **WHEN** a client writes a request and closes stdin immediately
- **THEN** the response to that request arrives on stdout before the process exits

#### Scenario: A notification is not a response

- **WHEN** the server writes a notification between accepting a request and answering it
- **THEN** it keeps waiting for the real response before it exits

#### Scenario: A response that never comes

- **WHEN** an accepted request is never answered
- **THEN** the process exits no later than 5 seconds after stdin closed

### Requirement: One stderr session line per initialize (VRFY-03)

After each successful `initialize`, the server SHALL write exactly one stderr line of the form `codegraph: mcp-session requested=<v> negotiated=<v> client=<name>/<version> tools=<n>`. It SHALL sanitize the client-supplied fields so that they cannot break the line. The line cannot be turned off.

#### Scenario: A normal handshake

- **WHEN** a client named `probe` at version `1` initializes offering `2025-11-25`, with an index present and no tool filter
- **THEN** stderr carries `codegraph: mcp-session requested=2025-11-25 negotiated=2025-11-25 client=probe/1 tools=8`

#### Scenario: Hostile or missing client fields

- **WHEN** a client field is empty, or contains spaces, control characters or invalid UTF-8
- **THEN** an empty field is rendered as `<unknown>`, spaces and control characters become `_`, invalid bytes become U+FFFD, and each field is cut to at most 256 bytes
- **AND** the output stays a single line

#### Scenario: The requested version is omitted

- **WHEN** the `initialize` request omits `protocolVersion`
- **THEN** the session line says `requested=<unknown>`

#### Scenario: Sessionless traffic

- **WHEN** a client sends only sessionless 2026-07-28 requests and never `initialize`
- **THEN** no session line is written

### Requirement: Classic initialize negotiation (SPEC-06, VRFY-02)

On `initialize`, the server SHALL answer with the offered protocol revision when it is one of `2025-11-25`, `2025-06-18`, `2025-03-26` or `2024-11-05`. For any other offered revision, or none, it SHALL answer with `2025-11-25` and no error.

#### Scenario: A supported older revision

- **WHEN** a client offers `2024-11-05`
- **THEN** the result's `protocolVersion` is `2024-11-05`
- **AND** a later `tools/call` of `codegraph_explore` succeeds

#### Scenario: An unsupported revision through initialize

- **WHEN** a client offers `2026-07-28` through `initialize`
- **THEN** the result's `protocolVersion` is `2025-11-25` and there is no error

#### Scenario: No revision offered

- **WHEN** the `initialize` request omits `protocolVersion`
- **THEN** the result's `protocolVersion` is `2025-11-25`

### Requirement: Sessionless 2026-07-28 requests and server discovery (SPEC-01, SPEC-02, SPEC-07)

The server SHALL answer `server/discover` without a prior `initialize`, returning its capabilities, its instructions and `supportedVersions` equal to `["2026-07-28","2025-11-25","2025-06-18","2025-03-26","2024-11-05"]`. It SHALL validate the `_meta` of each 2026-07-28 request.

#### Scenario: Discovery, then a sessionless tool call

- **WHEN** a client sends `server/discover` with valid 2026-07-28 `_meta` and no `initialize`
- **THEN** it receives the capabilities, `supportedVersions` and `instructions`
- **AND** a sessionless `tools/call` of `codegraph_explore` then succeeds

#### Scenario: Client capabilities are missing

- **WHEN** a 2026-07-28 request's `_meta` lacks `io.modelcontextprotocol/clientCapabilities`
- **THEN** the server answers with JSON-RPC error `-32602`

#### Scenario: An unknown later revision

- **WHEN** a 2026-07-28 request's `_meta` names a revision newer than `2026-07-28`, such as `2099-01-01`
- **THEN** the server answers with JSON-RPC error `-32022`, whose `data` lists the supported and the requested revisions

### Requirement: Result metadata on 2026-07-28 requests (SPEC-03, SPEC-08)

Results to 2026-07-28 requests SHALL carry `resultType: "complete"` and the server's identity under `_meta["io.modelcontextprotocol/serverInfo"]`. Results inside a classic `initialize` session SHALL carry neither.

#### Scenario: A sessionless tool call

- **WHEN** a 2026-07-28 `tools/call` succeeds
- **THEN** the result has `resultType: "complete"` and `_meta["io.modelcontextprotocol/serverInfo"]` with the name `codegraph`

#### Scenario: A classic session

- **WHEN** a client that used `initialize` calls a tool
- **THEN** the result has neither `resultType` nor the server identity in `_meta`

### Requirement: Server identity

The server SHALL report the name `codegraph` and a version string in its server information.

#### Scenario: Initialize

- **WHEN** a client initializes
- **THEN** `serverInfo.name` is `codegraph` and `serverInfo.version` is a non-empty string

### Requirement: Fixed capabilities, independent of index state (MCP-03, RSRC-03, SPEC-09)

`initialize` and `server/discover` SHALL advertise exactly two capabilities, whether or not an index exists: `tools` with `listChanged: true`, and `resources` with no sub-features. They SHALL NOT advertise prompts, logging or completions.

#### Scenario: No index

- **WHEN** a client initializes in a directory with no index
- **THEN** the capabilities still include `tools` with `listChanged: true`

#### Scenario: Nothing else is advertised

- **WHEN** a client reads the capabilities
- **THEN** they contain no `prompts`, `logging` or `completions` entry, and `resources` has no `subscribe` or `listChanged`

### Requirement: Initialize comes first in a classic session

A request without 2026-07-28 `_meta` that arrives before `initialize` SHALL be rejected with a JSON-RPC error and SHALL NOT be executed.

#### Scenario: A tool call before initialize

- **WHEN** a `tools/call` arrives with no earlier `initialize` and no 2026-07-28 `_meta`
- **THEN** the server answers with a JSON-RPC error
- **AND** no session line is written

### Requirement: Server instructions (SPEC-07, WIRE-01, AGENT-14)

The server SHALL send one fixed instructions string, the same on `initialize` and `server/discover`, containing no host path, index location or environment value. It SHALL tell agents to try `codegraph_explore` first, say when tools appear and how to narrow them, and point to the reference resources and the installed skill.

#### Scenario: The string never varies

- **WHEN** any client initializes or calls `server/discover`, in any directory
- **THEN** `instructions` is the same string byte for byte

#### Scenario: What the string promises

- **WHEN** an agent reads the instructions
- **THEN** it learns that all eight tools register by default once an index exists, with no client restart
- **AND** that an empty tool list means there is no index and `codegraph init` creates one
- **AND** that `CODEGRAPH_MCP_TOOLS` narrows the default set and `resources/list` holds per-tool reference documents

#### Scenario: Shape

- **WHEN** the instructions are measured
- **THEN** they are a single paragraph with no newline character

### Requirement: Default tool set (MCP-01, VERB-07)

When an index resolves and `CODEGRAPH_MCP_TOOLS` is unset, the server SHALL offer exactly eight tools: `codegraph_explore`, `codegraph_node`, `codegraph_search`, `codegraph_callers`, `codegraph_callees`, `codegraph_impact`, `codegraph_files` and `codegraph_status`. This replaced MCP-01's original single default tool on 2026-08-09.

#### Scenario: Default listing

- **WHEN** a client calls `tools/list` with an index present and no tool filter
- **THEN** the answer lists exactly those eight tools

### Requirement: Tool input schemas and listing order (SDK-05, NODE-03, SURF-01)

Each tool SHALL publish a fixed description and an input schema that rejects undeclared properties. `tools/list` SHALL return the tools in the same order every time.

#### Scenario: Stable order

- **WHEN** a client calls `tools/list` twice
- **THEN** both answers list the tools in the same order: callees, callers, explore, files, impact, node, search, status

#### Scenario: Required and defaulted arguments

- **WHEN** a client reads the input schemas
- **THEN** `codegraph_explore` and `codegraph_search` require `query`, and `codegraph_callers`, `codegraph_callees` and `codegraph_impact` require `symbol`
- **AND** `codegraph_explore`'s `max_files` defaults to 5, and `codegraph_impact`'s `depth` defaults to 2 with a maximum of 50
- **AND** `codegraph_node` takes optional `symbol`, `file` and `line`, and every tool takes an optional `path`

### Requirement: Tools are read-only and closed-world

Every tool SHALL be annotated with `readOnlyHint: true`, `openWorldHint: false` and `idempotentHint: false`, and SHALL NOT carry `destructiveHint`.

#### Scenario: Annotations in the listing

- **WHEN** a host reads the tool list
- **THEN** every codegraph tool is marked read-only and closed-world, and none is marked destructive

### Requirement: Narrowing the tool set (MCP-02)

`CODEGRAPH_MCP_TOOLS` SHALL narrow the seven companion tools by their short names `node`, `search`, `callers`, `callees`, `impact`, `files` and `status`, matched case-sensitively. `codegraph_explore` SHALL always be offered when an index resolves. Unknown names SHALL be ignored with a stderr warning and SHALL never stop startup. The original allowlist form of MCP-02 was replaced by this narrowing form on 2026-08-09.

#### Scenario: Unset

- **WHEN** `CODEGRAPH_MCP_TOOLS` is unset
- **THEN** all eight tools are offered

#### Scenario: A narrowed list

- **WHEN** `CODEGRAPH_MCP_TOOLS` is `node,status`
- **THEN** exactly `codegraph_explore`, `codegraph_node` and `codegraph_status` are offered

#### Scenario: The empty string

- **WHEN** `CODEGRAPH_MCP_TOOLS` is set to the empty string
- **THEN** only `codegraph_explore` is offered

#### Scenario: Unknown names

- **WHEN** an entry is not a companion short name, such as `bogus`, `codegraph_node` or `explore`
- **THEN** stderr gets one warning per unknown name and one line saying how many of the seven companions will register
- **AND** the server starts normally

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

### Requirement: The tool set follows the index during a session (SPEC-05)

Before answering `initialize`, `tools/list`, `tools/call` or `server/discover`, the server SHALL check again whether an index resolves for its start directory, and SHALL add or remove tools to match, honoring `CODEGRAPH_MCP_TOOLS`, with no restart or reconnect.

#### Scenario: An index appears mid-session

- **WHEN** a session starts with no index and `codegraph init` runs during it
- **THEN** the next `tools/list` on the same connection lists the full, filtered tool set

#### Scenario: The index disappears mid-session

- **WHEN** the `.codegraph/` directory is removed during a session
- **THEN** the next `tools/list` returns zero tools

#### Scenario: Nothing changes

- **WHEN** the index state does not change between two `tools/list` calls
- **THEN** both return the same tools with no duplicates

### Requirement: Tool-list change notifications (SPEC-09)

When the tool set changes, the server SHALL send one content-free `notifications/tools/list_changed` to each 2026-07-28 client that subscribed through `subscriptions/listen` with `toolsListChanged: true`.

#### Scenario: Subscribing

- **WHEN** a 2026-07-28 client opens `subscriptions/listen` asking for `toolsListChanged`
- **THEN** it receives `notifications/subscriptions/acknowledged` echoing exactly `{"toolsListChanged": true}`

#### Scenario: The tool set changes

- **WHEN** an index then appears
- **THEN** exactly one `notifications/tools/list_changed` arrives on the same stream

### Requirement: Catalog and resource answers are not cacheable (SPEC-04)

Answers to `tools/list`, `server/discover`, `resources/list` and `resources/read` SHALL carry `ttlMs: 0` and `cacheScope: "private"`.

#### Scenario: Any of the four methods

- **WHEN** the server answers `tools/list`, `server/discover`, `resources/list` or `resources/read`
- **THEN** the result has `ttlMs` equal to 0 and `cacheScope` equal to `private`

### Requirement: Tool calls are read-only and confined to the index root

Each tool call SHALL read a fresh snapshot of the index and SHALL never write the graph or start a reindex. The optional `path` argument SHALL default to the server's start directory, and a `path` outside the resolved index root SHALL be rejected.

#### Scenario: No path given

- **WHEN** a tool call omits `path`
- **THEN** it reads the index for the server's start directory

#### Scenario: A path inside the index root

- **WHEN** `path` points inside the index root but outside the start directory
- **THEN** the call succeeds

#### Scenario: A path outside the index root

- **WHEN** `path` resolves outside the index root, even to another indexed project
- **THEN** the result has `isError: true` and says the path is outside the server's configured repo root

### Requirement: Tool results are markdown shared with the CLI (SURF-06, EXPL-05, NODE-04)

Each successful tool result SHALL be a single text item of markdown produced by the same query engine as the CLI, with no JSON payload and no structured content. The results of `codegraph_explore` and `codegraph_node` SHALL equal the output of `codegraph explore` and `codegraph node` for the same query, byte for byte.

#### Scenario: Table-shaped results

- **WHEN** `codegraph_callers`, `codegraph_callees`, `codegraph_impact` or `codegraph_search` succeeds
- **THEN** the text is a markdown table with the columns Name, Kind and Location

#### Scenario: Files and status

- **WHEN** `codegraph_files` succeeds
- **THEN** the text is a markdown table with the columns Path, Language, Nodes and Edges
- **AND** a successful `codegraph_status` returns a report headed `**CodeGraph Status**`

#### Scenario: CLI and MCP agree

- **WHEN** the same query runs through `codegraph explore` and through `codegraph_explore`
- **THEN** the two outputs are identical, and the same holds for `codegraph node` and `codegraph_node`, including the `line` argument

### Requirement: Worktree notice on a borrowed index (WORK-01, WORK-02, WORK-03)

When the start directory is a git worktree that uses another working tree's index, every successful tool result except `codegraph_status` SHALL begin with a one-line notice, and `codegraph_status` SHALL include its own warning instead. Error results SHALL carry no notice.

#### Scenario: A worktree borrowing the main checkout's index

- **WHEN** the server runs in a linked worktree with no index of its own
- **THEN** a successful `codegraph_explore` result begins with the notice
- **AND** the same call from the main checkout has no notice

#### Scenario: A failed call

- **WHEN** a tool call fails in such a worktree
- **THEN** the error result carries no notice

### Requirement: Tool failures are tool results and unroutable requests are protocol errors (SDK-04)

A failure inside a known tool SHALL come back as a JSON-RPC result with `isError: true` and exactly one text item. A request that the server cannot route SHALL get a JSON-RPC error object.

#### Scenario: Bad arguments to a known tool

- **WHEN** a call omits a required argument or passes an undeclared one
- **THEN** the result has `isError: true`, and a missing argument is named in the text

#### Scenario: An engine error

- **WHEN** `codegraph_node` names a symbol that does not exist
- **THEN** the result has `isError: true` and its text says the symbol was not found

#### Scenario: Unroutable requests

- **WHEN** the tool name is unknown or empty, or `arguments` is not an object
- **THEN** the server answers with JSON-RPC error `-32602`
- **AND** an unknown method gets JSON-RPC error `-32601`

### Requirement: A briefly locked store does not fail or hang a call

When a tool call finds the graph store locked by a running sync, it SHALL retry for a bounded time. If the store is still locked, it SHALL return a result with `isError: true` instead of waiting indefinitely.

#### Scenario: A short lock

- **WHEN** a sync holds the store for less than the retry window as a tool call starts
- **THEN** the call succeeds

#### Scenario: A long lock

- **WHEN** another process keeps the store locked past the retry window
- **THEN** the call returns a result with `isError: true` that says the store lock is held

### Requirement: Reference resources are always listed (RSRC-01, RSRC-03, GUARD-02)

The server SHALL list ten `text/markdown` resources whether or not an index exists: `codegraph://tools/<name>` for each of the eight tools by short name, `codegraph://tools-filter` and `codegraph://index-state`.

#### Scenario: A never-indexed repository

- **WHEN** a client calls `resources/list` where no index exists
- **THEN** all ten resources are listed while `tools/list` is empty

#### Scenario: Listing entries

- **WHEN** a client reads the resource list
- **THEN** each entry's `name` is its file name, such as `explore.md`, its `mimeType` is `text/markdown`, and the entries are ordered by URI

### Requirement: Reading a resource (RSRC-02)

`resources/read` on a listed URI SHALL return the resource's full markdown text with `mimeType: text/markdown`. Any other URI SHALL get JSON-RPC error `-32602`, with the requested URI in the error data.

#### Scenario: A listed resource

- **WHEN** a client reads `codegraph://tools/explore`
- **THEN** it receives non-empty markdown text

#### Scenario: An unknown resource

- **WHEN** a client reads `codegraph://tools/does-not-exist`
- **THEN** the server answers with JSON-RPC error `-32602`

### Requirement: Resource text matches the server (GUARD-01, GUARD-02)

Every count, default, limit and environment-variable name stated in resource text SHALL match the tools the server actually offers, and resource text SHALL contain no host-specific facts.

#### Scenario: A stated default

- **WHEN** a resource states a tool's default or maximum, such as the depth of `codegraph_impact`
- **THEN** the value equals the one the tool's input schema uses

#### Scenario: Host facts

- **WHEN** any resource is read on any machine
- **THEN** its text contains no absolute host path or other machine-specific value

### Requirement: Finding the index (MCP-03, AGNT-03)

The server SHALL take its start directory from `--path` (`-p`) or the current directory. It SHALL use the nearest directory at or above it that holds the repository's `.codegraph/` index as both the index root and the confinement root. If none exists up to the filesystem root, it SHALL run with no index.

#### Scenario: A subdirectory of an indexed repository

- **WHEN** the server starts in a subdirectory of an indexed repository
- **THEN** the repository root is the index root and the tools are offered

#### Scenario: A linked worktree without its own index

- **WHEN** the server starts in a linked worktree that has no index, under a main checkout that has one
- **THEN** the main checkout is the index root, and the worktree stays the default `path` for tool calls

#### Scenario: No index anywhere above

- **WHEN** no directory at or above the start directory holds an index
- **THEN** the server runs with zero tools

### Requirement: Startup sync before serving (SYNC-03)

When an index exists at startup, the server SHALL bring it up to date with an incremental sync before it reads its first request, so that the first tool call sees changes made while no watcher ran.

#### Scenario: Offline changes

- **WHEN** files changed while no watcher was running
- **THEN** the first `codegraph_explore` after connecting finds the changed symbols

#### Scenario: Nothing changed

- **WHEN** no file changed since the last sync
- **THEN** the startup sync reparses no files

### Requirement: Startup sync failures

If another codegraph process holds the store at startup, the server SHALL skip the startup sync with a stderr warning and keep serving. Any other startup sync failure SHALL stop the server before the handshake.

#### Scenario: The store is locked

- **WHEN** another codegraph process holds the store as the server starts
- **THEN** stderr gets a warning that the startup sync was skipped, and the server serves requests

#### Scenario: The store cannot be read

- **WHEN** the store is corrupt or unreadable at startup
- **THEN** the server exits with a non-zero status before the handshake

### Requirement: File watcher on by default (WATCH-01, WATCH-02, WATCH-03, SYNC-01)

When an index exists at startup, the server SHALL start a file watcher in the background without delaying the handshake, so that edits reach later tool calls after the debounce window. `--no-watch` or `CODEGRAPH_NO_WATCH=1` SHALL disable it. `--watch` or `CODEGRAPH_FORCE_WATCH=1` SHALL force it on where it is off by default, on WSL2 Windows drives.

#### Scenario: Default start

- **WHEN** `codegraph serve --mcp` starts with an index and no watch flag
- **THEN** `initialize` and `tools/list` answer without waiting for the watcher

#### Scenario: Disabled

- **WHEN** the watcher is disabled by `--no-watch` or `CODEGRAPH_NO_WATCH=1`
- **THEN** the server still runs, and stderr gets one line saying the watcher is disabled and that `codegraph sync` refreshes the graph

#### Scenario: A WSL2 Windows drive

- **WHEN** the project is on a WSL2 `/mnt/<letter>` drive and the watcher is not forced
- **THEN** the watcher is off, and stderr gives the WSL2 reason

#### Scenario: Environment values

- **WHEN** `CODEGRAPH_NO_WATCH` or `CODEGRAPH_FORCE_WATCH` is set to anything other than `1`
- **THEN** it has no effect

### Requirement: Watcher limits within a session

The watcher decision SHALL be made once, at startup: an index created during a session gets tools but no watcher until the next session. Watcher failures SHALL be reported on stderr and SHALL NOT stop the server.

#### Scenario: An index created mid-session

- **WHEN** `codegraph init` runs during a session that started without an index
- **THEN** the tools appear, but no watcher starts in that session

#### Scenario: The watcher fails

- **WHEN** the file watcher cannot start or its event stream fails
- **THEN** stderr reports the watcher error, and tool calls keep working
