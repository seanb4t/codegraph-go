---
id: SEED-001
status: dormant
planted: 2026-07-14
planted_during: v0.1 (Initial Release) milestone
trigger_when: when CLI-surface parity with TS CodeGraph is reached (the path to 1.0)
scope: large
audit_acknowledged:
  milestone: v0.11.0
  at: 2026-08-17
  status: dormant
---

# SEED-001: local Svelte + shadcn-svelte UI for browsing/viewing/querying the graph

## Why This Matters

Today the graph is reachable only through the CLI and the stdio MCP server (agent-facing). A **local, human-facing web UI** — running against a local `.codegraph/` (e.g. served by a `codegraph serve` / `codegraph ui` mode) — would let a developer *browse* symbols and files, *view* a node's source + neighbors, and *run* the same queries (`query`/`callers`/`callees`/`impact`/`affected`/`explore`) interactively, with graph visualization. It's the natural human counterpart to the agent-facing MCP surface and a differentiator the TS impl's value could be matched/exceeded on. Deliberately deferred: chasing a UI before the core CLI surface is at parity would be building on a moving base.

## When to Surface

**Trigger:** once CLI-surface parity with TS CodeGraph is reached (the remaining bar for a 1.0 — see ROADMAP "Next" / PROJECT.md Key Decision "Full parity"). Building the UI on a still-shifting command/output surface would mean rework; wait until the query/CLI contract is stable.

This seed will surface during `/gsd-new-milestone` when the milestone scope matches (UI / visualization / developer-experience).

## Scope Estimate

**Large** — a full milestone. A Svelte + shadcn-svelte front-end, a local HTTP surface exposing the read-only query engine (reuse `internal/query.Engine`, not a reimplementation), graph-visualization rendering, and packaging (ideally embedded into the single static binary so it stays "one binary, no runtime" — investigate `embed` of the built SPA). Local-only, no hosted platform (that's permanently out of scope).

## Breadcrumbs

- `internal/query` — the read-only `Engine` (query/search/callers/callees/impact/affected/files/status/node/explore) the UI should sit on top of; do not duplicate query logic.
- `internal/cli` + `cmd/codegraph` — where a new `serve`-style HTTP/UI subcommand would register (Phase-6 cobra pattern); `serve --mcp` is the existing agent transport.
- `internal/mcp` — the explore/node markdown templates already shape "verbatim source + call paths + blast radius"; the UI can render the same underlying data.
- `.planning/PROJECT.md` (Out of Scope) — "graph visualization UI — planned future milestones; the schema anticipates them (annotations, export) but v0.1 does not implement them." This seed is that future milestone.
- **`shadcn-svelte` project skill** is available in this environment — use it for component scaffolding/registry when the UI is built.
- Storage already supports the substrate: `Meta` reserves annotation ranges (embedding vectors, community assignments) and there's a bulk graph export/import stream — useful for a viz that wants the whole graph.

## Notes

Captured via one-shot seed. Keep it **local-first** (runs against the on-disk `.codegraph/`, no cloud) and ideally **embedded in the static binary** to preserve the "single verifiable binary, no bundled runtime" property. Pairs with the eventual embeddings/community-detection future work (also reserved in the schema).
