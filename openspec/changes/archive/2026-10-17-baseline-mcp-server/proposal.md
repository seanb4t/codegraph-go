# Proposal: Baseline spec for the MCP server

Issue: #122

## Why

codegraph-go now plans behavior changes with OpenSpec, but no spec yet describes what already ships. The MCP server (`codegraph serve --mcp`) is the surface agents depend on most. Open work against it (#74, #113, #121) needs a written baseline to propose deltas against.

## What Changes

- Add the `mcp-server` capability spec. It describes the current, shipped behavior of `codegraph serve --mcp` as ADDED requirements only. No behavior changes.
- Each requirement names its origin requirement ID from the planning archive (tag `planning-archive-v0.15.0`), so ID references in code comments resolve to a living spec.
- `tasks.md` maps each requirement to the test, golden or wire-oracle transcript that pins it today. A requirement that nothing pins is marked unpinned, not dropped.
- Small tidy-up in the same PR: two comments in `internal/upgrade` tests still tell readers not to gate on `gsd-tools check tdd-red-evidence`, a GSD tool this repository no longer uses. They are removed.

## Out of scope

- Fixing known defects. They are excluded, not specified as correct behavior:
  - #74: `serve --mcp` takes the global `~/.codegraph` directory as a project.
  - #113: the stdin reader buffers an unterminated line without bound.
  - Not yet tracked: one malformed JSON line ends the session and drops responses already owed, because the SDK treats it as a read error and sends no `-32700` reply.
  - Not yet tracked: two SDK-default methods, `prompts/list` and `resources/templates/list`, answer with `cacheScope: "public"`.
- Specifying behavior that looks unintended: a relative tool `path` resolves against the process directory, and a nested index inside the index root is readable.
- New behavior, such as elicitation (#121) or a network transport.
- Other capabilities. `graph-queries`, `indexing`, `agent-integrations` and the rest get their own baseline changes.
- New tests for unpinned behavior. They are recorded here and can follow as their own changes.

## Capabilities

### New Capabilities

- `mcp-server`: the MCP server that `codegraph serve --mcp` runs over stdio. It covers protocol negotiation, the tool set and how it is narrowed, resources and server instructions, how the server finds its index and syncs it at startup, and the errors a client sees.

### Modified Capabilities

None.

## Impact

- Archiving the change creates `openspec/specs/mcp-server/spec.md`.
- No change to code behavior, APIs, output shapes or dependencies.
- Two comment-only edits in `internal/upgrade/changie_shape_test.go` and `internal/upgrade/changie_release_shape_test.go`.
- From now on, changes to `internal/mcp` or `serve --mcp` propose deltas against this spec.
