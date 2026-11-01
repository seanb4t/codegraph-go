# Proposal: Give each declaration its own node, and prove the graph invariants with properties

Issue: #221, #198, #188

This change is part 3 of #221. It also closes #198 and #188, because the first invariant test fails on `main` until both are fixed.

## Why

Seven of the last twelve bugs were indexer bugs of one shape: #214, #198, #194, #193, #188, #187, and #197 as a performance relative. After an incremental sync, the graph differed from a fresh index of the same tree. Or a declaration had no node, or two declarations shared one id. The example tests found none of them before a release.

Two of these bugs are still open. #198 is about two declarations in one file with the same kind and qualified name. They get one node id, so the store keeps one and loses the others. In this repository, extraction yields 470 such repeats, for example `nextGeometry` in eight `it(...)` blocks of `web/tests/graph-live-update.test.ts`. In #188, `Meta.NodeCount` counts every extracted node, repeats included, and each sync adds the repeats again. A long-lived index of this repository reported 27,253 nodes for 15,914 stored nodes.

## What Changes

- When a file declares two or more symbols with the same kind and qualified name, each one gets its own node. The first one in source order keeps the id it has today. The second and later ones get an ordinal in the id preimage. Qualified names, names and the CLI and MCP output shapes do not change. A query now lists every copy, each with its own line.
- C# `partial` types keep one shared node, as `docs/LANGUAGE-CAPABILITY-MATRIX.md` documents.
- `Meta.NodeCount` equals the number of nodes in the store after `codegraph index` and after every `codegraph sync`.
- A new `Meta` flag marks a store that has unique declaration ids. A store without the flag gets one full rebuild on its next sync, the same way `has_grouped_vars` worked for #214. That rebuild also corrects a drifted node count.
- New property and fuzz tests:
  - A model-based `rapid` property: `Sync` over a generated sequence of edits gives the same graph as a fresh index of the final tree.
  - A walk over the frozen corpora in `testdata/golden/corpus/`: each declaration has exactly one node, no two nodes share an id, and each edge endpoint is a node that exists.
  - A stateful `rapid` property for the watcher: after any sequence of directory creates, renames and deletes, the watch set equals the directory set (the #208 class).
  - A native Go fuzz target for `keyenc`: decoding an encoded key gives the key back, and the byte order of encoded keys equals the sort order of the keys. The seed corpus runs as a unit test on every push. A weekly scheduled workflow runs the fuzzer for a fixed time.

Out of scope:

- Scoped reference resolution. A reference names a qualified name, not a copy. So a call to `nextGeometry` from the third `it(...)` block still resolves to the first copy.
- New qualified names for anonymous scopes, for example a name taken from the `it(...)` description string.
- Mutation testing. #222 trials a tool after this change merges.
- A BDD or Gherkin runner. A test name that names its scenario gives the traceability.
- The CI and test-suite rework in #221 parts 1, 2 and 4. The change `rework-ci-test-suite` holds that work.

## Capabilities

### New Capabilities

- `graph-index`: how the index represents the source. It covers declaration identity, the equality of an incremental sync and a fresh index, and the counts that the store reports.

### Modified Capabilities

None.

## Impact

- `internal/indexer/nodeid`: a per-file id allocator next to `NodeID`.
- Each extractor that calls `nodeid.NodeID` for a declaration: `goextract`, `tsextract`, `pyextract`, `javaextract`, `csharpextract` (not for partial types), and the `mainstream` extractors for C, Kotlin, PHP, Ruby, Rust and Swift.
- `internal/indexer/resolve.go` (`writeGraph`) and `internal/indexer/sync.go` (`nodesAdded`, `needsFileIndexBackfill`).
- `internal/schema/graph.proto`: one new `Meta` field. This is a change to the graph schema, which #221 lists as a sensitive area. The change is additive, and `SchemaVersion` stays 1.
- Node ids change for the second and later copies of a repeated declaration. An existing index rebuilds once, on its first sync after the upgrade.
- A new test-only dependency, `pgregory.net/rapid`, through the minimal-dependency audit.
- A new scheduled workflow for the fuzz run.
