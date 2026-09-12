---
created: 2026-09-07T00:00:00.000Z
title: internal/query's dependency-direction invariant (T-01-18) has no persisted regression test
area: architecture
resolves_phase: 7
severity: medium
status: resolved
resolved_at: 2026-09-08T00:00:00.000Z
resolved_by_phase: 7
files:

  - internal/query/
  - internal/graphstore/archtest/import_graph_test.go
  - internal/query/archtest/import_direction_test.go

threat_ref: T-01-18
audit_acknowledged:
  milestone: v0.12.0
  at: 2026-09-08
---

## Problem

Phase 1's threat T-01-18 protects the direction of the Engine seam: `internal/query`
must not depend on the wire layer. The UI is meant to be a *third consumer* of the
Engine (alongside `internal/cli` and `internal/mcp`), so a dependency edge pointing
the other way — `internal/query` importing `connectrpc.com/connect` or
`internal/uiproto` — would invert the architecture the whole milestone rests on.

The plan's stated mitigation was:

```
go list -deps ./internal/query   # must contain no connectrpc.com/connect,
                                 # no internal/uiproto
```

That check was written into `01-04-PLAN.md` and `01-05-PLAN.md`'s `<verification>`
blocks as a **one-shot acceptance check**. It was run once, at plan time, and never
committed as a test. Nothing re-runs it.

Found during the retroactive Phase 1 security audit (2026-09-07, `01-SECURITY.md`
Audit Note 1), confirmed absent by `rg` across the whole tree with a positive
control: `internal/graphstore/archtest/import_graph_test.go` *does* exist and does
exactly this kind of import-graph assertion for another package — so the search
technique works and the absence here is genuine, not a search miss.

## Current state

The invariant is **empirically true right now**. The audit re-ran the check live at
HEAD: `go list -deps ./internal/query` contains zero `connectrpc.com/connect` and
zero `internal/uiproto` entries, with `google.golang.org/protobuf` confirmed present
as a positive control (it arrives legitimately via `internal/schema`, proving the
extraction is live rather than returning an empty set).

So this is not an open vulnerability. It is an **unguarded invariant**: the property
holds, and nothing would notice if a future change broke it.

## Why it matters

This is the exact failure shape rule `84d1gfpywd` names. A mitigation that exists
only as prose in a completed plan is indistinguishable, six months later, from a
mitigation that was never implemented — and the cost of rediscovering it is another
full audit. The milestone's cross-phase integration check independently confirmed
"no duplicated traversal logic in `internal/uiserver`", but that is the *other*
direction; nothing tests this one.

## Suggested fix

Add an archtest beside the existing one, following
`internal/graphstore/archtest/import_graph_test.go`'s established shape rather than
inventing a new mechanism:

- assert `go list -deps ./internal/query` (or `golang.org/x/tools/go/packages`)
  contains no `connectrpc.com/connect` and no `internal/uiproto` entry
- carry a **positive control** in the same test — assert a package that IS expected
  in the dependency set is found (e.g. `google.golang.org/protobuf` via
  `internal/schema`), so an extractor that silently returns nothing fails loudly
  rather than passing vacuously
- prove it goes RED before landing it

Note the "count uses, never the declaration" discipline: assert over the resolved
dependency set, not over import lines in source files, which a build-tag-guarded or
transitively-reached import would evade.

## Resolution (2026-09-08, Phase 7, 07-02-PLAN.md)

Landed as `internal/query/archtest/import_direction_test.go`,
`TestQueryImportsNoWireLayerOrIndexerRoot`. Forbidden wire-layer set expanded to
the D-02 union of the roadmap list and this todo's list (`internal/uiserver`,
`internal/mcp`, `internal/uiproto`, `connectrpc.com/connect`), checked over the
resolved transitive dependency set (`packages.NeedDeps`, not direct imports) of
every loaded `internal/query`-rooted package including test variants. A second,
production-scoped rule additionally forbids the `internal/indexer` root while
allowing the two leaves `internal/query` already resolves
(`internal/indexer/goextract`, `internal/indexer/nodeid`) — scoped to production
because `internal/query/engine_test.go` legitimately imports the root to build
fixtures.

Both forbidden sets were proven RED against a real import (`07-MUTATION-LOG.md`
family (b)) before being called a guard, then reverted byte-clean. Positive
controls assert `internal/graphstore` and `internal/indexer/goextract` present,
and `internal/parser` present-but-not-direct, proving the walk resolves beyond
one hop rather than passing vacuously on a degenerate load.
