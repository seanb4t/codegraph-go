---
id: SEED-004
status: dormant
planted: 2026-09-25
planted_during: between milestones (post v0.14.0) / gsd-explore GH #85
trigger_when: server mode ships with graph identity (repo, sha) and a per-graph Export/Import path
scope: server-mode, ci, graphstore
---

# SEED-004: CI-distributed indexes — CI uploads a pre-built graph keyed by (repo, sha)

## Why This Matters

PROJECT.md's milestone-2 list names "CI-distributed indexes" beside the central server. Once
the server addresses graphs by `(repo, sha)` (GH #85 exploration, D2), a CI job that already
has the checkout can run the laptop CLI, `Export` the graph, and push it to the server —
skipping the server's own mirror fetch and materialize for that commit. Same identity, same
immutability rules, different producer. This is also the path for repositories the GitHub
App cannot reach (other forges, air-gapped runners).

## When to Surface

- When the multi-graph store lands and `graphstore.Export` per graph exists on the server side.
- When a fovea repository's base build is slower than its CI build of the same commit.
- When a non-GitHub source provider is requested.

## Sketch

- `codegraph-server` gains `ImportGraph(repo, sha, stream)` (client-streaming needs HTTP/2 per
  connect docs — or a PUT with a content-addressed body).
- The laptop CLI gains `codegraph export --for-server` producing the schema-versioned stream.
- Schema version must match or the server rejects; provenance (who built it, from which
  tree hash) stored in the manifest.
