# Proposal: Give each declaration its own node, resolve references as the language does, and prove the graph invariants with properties

Issue: #221, #198, #188

This change is part 3 of #221. It also closes #198 and #188, because the first invariant test fails on `main` until both are fixed.

## Why

Seven of the last twelve bugs were indexer bugs of one shape: #214, #198, #194, #193, #188, #187, and #197 as a performance relative. After an incremental sync, the graph differed from a fresh index of the same tree. Or a declaration had no node, or two declarations shared one id. The example tests found none of them before a release.

Two of these bugs are still open. #198 is about two declarations in one file with the same kind and qualified name. They get one node id, so the store keeps one and loses the others. In this repository, extraction yields 470 such repeats, for example `nextGeometry` in eight `it(...)` blocks of `web/tests/graph-live-update.test.ts`. In #188, `Meta.NodeCount` counts every extracted node, repeats included, and each sync adds the repeats again. A long-lived index of this repository reported 27,253 nodes for 15,914 stored nodes.

Giving each declaration its own node exposed two older gaps in resolution. Overloads in Java, C#, Kotlin, C++ and Swift were one merged node. The merged node mixed the bodies of all overloads and produced false edges. Guava has 2,375 overloaded names and serilog has 81. A call to an overloaded name must reach the overload the language selects, not the first one in the file. And a call through a variable, such as `logger.log(x)`, never resolved at all. The extractors read the receiver's capitalization to decide whether it is a type or a variable. Most calls in those languages go through a variable.

## What Changes

- Every function and method records its signature as data: each parameter's name, type as written, and whether it is optional, has a default or is variadic, plus the return type as written. Every call records its argument count and whether an argument is spread. It records each argument's type where the syntax or the enclosing body shows it, and for Swift its labels.
- A function or method id is a hash of its kind, qualified name, file and signature key. The key is the ordered parameter types as written, with a placeholder for an untyped parameter. It marks optional, default and variadic parameters. One rule applies to every language. Two declarations with identical keys are redefinitions or alternatives, and only they get an ordinal. Adding or removing an overload changes no other id.
- Each overload is its own node. Two or more overloads of one name under one owner get an `overload_set` node that contains them. A name with one declaration has no set node.
- A call resolves to the overload the language selects: first by arity, with ranges for optional, default and variadic parameters, then by the argument types the extractor knows. A call that static analysis cannot decide gets an edge to the set, marked ambiguous. It never gets an edge to a guessed overload.
- A call through a variable resolves through the variable's declared type. The variable is a parameter, a typed local, a field of the enclosing type, or `this`. A local without a type takes the type of a literal, a `new`, a cast or a call with one possible target. The call case is filled in the cross-file pass, because a file is extracted on its own. The receiver's capitalization is never a signal. A receiver whose type is unknown stays unresolved, as today.
- A same-file reference resolves by lexical scope: the innermost enclosing declaration of the name wins. Only exported or module-level declarations enter the cross-file index.
- Repeats that are not overloads follow their language's rule. In Python, Ruby and JavaScript the last module-level definition wins. Copies under `#if`, `#ifdef` or a `function_exists` guard are alternatives, so a call to them targets their set as ambiguous. A reopened Ruby class or module is one node. A Swift function's qualified name includes its argument labels. A C# `partial` type keeps one shared node, as `docs/LANGUAGE-CAPABILITY-MATRIX.md` documents.
- The methods of a repeated class attach to their own class, not to the last class of that name.
- A query on a bare name that is a set reports across its overloads, each result labeled with its signature. `--line` and a new `--signature` select one overload. `callers` on one overload also lists the set's ambiguous callers, marked as such.
- `Meta.NodeCount` equals the number of nodes in the store after `codegraph index` and after every `codegraph sync`.
- A new `Meta` flag marks a store that has unique declaration ids. A store without the flag gets one full rebuild on its next sync, the same way `has_grouped_vars` worked for #214. The rebuild starts from an empty graph, so it also removes stale records and corrects a drifted node count.
- New property and fuzz tests:
  - A model-based `rapid` property: `Sync` over a generated sequence of edits gives the same graph as a fresh index of the final tree. The generator writes Go, TypeScript and Java files, with overloads and typed calls in Java.
  - A tree-sitter count of declarations, compared with the extracted nodes, over the in-repository fixtures and the four pinned corpora. No two nodes share an id, and each edge endpoint is a node that exists.
  - A stateful `rapid` property for the watcher: after any sequence of directory creates, renames and deletes, the watcher covers every directory (the #208 class).
  - Two bugs that the properties found are fixed here. A sync now reparses the files whose candidate set changes, and the Linux watcher keeps a moved directory watched.
  - Native Go fuzz targets for the key segments: decoding an encoded segment gives it back, and two different segment lists never encode to the same bytes. The seed corpus runs as a unit test on every push. A weekly scheduled workflow runs the fuzzer for a fixed time.

Out of scope:

- A type checker. Resolution uses declared types, literals and one level of return-type propagation. Generic inference, the element type of a collection, flow analysis across reassignments and inherited fields stay unknown, and an unknown type matches any parameter.
- C++ templates and implicit conversions, and Swift's resolution by return type.
- References by path across languages, for example `subprocess.run(["./deploy.sh"])`. The reference record and the candidate sets are language-neutral so that a later change can add them. A follow-up issue tracks it.
- New qualified names for anonymous scopes, for example a name taken from the `it(...)` description string.
- Mutation testing. #222 trials a tool after this change merges.
- A BDD or Gherkin runner. A test name that names its scenario gives the traceability.
- The CI and test-suite rework in #221 parts 1, 2 and 4. The change `rework-ci-test-suite` holds that work.

## Capabilities

### New Capabilities

- `graph-index`: how the index represents the source. It covers declaration identity, overload sets and how a reference resolves. It also covers the equality of an incremental sync and a fresh index, and the counts that the store reports.
- `graph-queries`: how a query names a symbol that has several overloads, and how it reports across them.

### Modified Capabilities

None.

## Impact

- `internal/indexer/nodeid`: a per-file id allocator next to `NodeID`, with the signature key in the hash.
- Every extractor: structured signatures at each declaration, argument data at each call, a per-function type environment for receivers and arguments, and scope tracking. The extractors are `goextract`, `tsextract`, `pyextract`, `javaextract` and `csharpextract` (not for partial types). The `mainstream` extractors for C, Kotlin, PHP, Ruby, Rust and Swift change the same way. The casing heuristics in `tsextract.memberAccessAlias`, `csharpextract.isLikelyTypeName` and `goextract` (WR-02) go away.
- `internal/indexer/resolve.go` (`writeGraph`, the calls branch), `internal/indexer/sync.go` (`nodesAdded`, the two `Meta` writers, `needsFileIndexBackfill`, the re-resolution step) and `internal/indexer/symbolindex.go` (candidate sets).
- `internal/query`: a bare name that is a set, and `--signature` on `node`, `callers`, `callees` and `impact`, in the CLI and the MCP tools. The output shapes gain a signature label on results. This is a change to the CLI and MCP compatibility surface.
- `internal/graphstore`: a writer method that deletes the whole graph in one batch.
- `internal/schema/graph.proto`: a `Param` message, structured parameters and a return type on `Node`, and argument data on `Edge`. Also one new node kind, `overload_set`, and one new `Meta` field. Every field is additive, and `SchemaVersion` stays 1.
- Every function and method id changes once, because the signature key enters the hash. An existing index rebuilds once, on its first sync after the upgrade. The golden outputs of the pinned corpora change where they list overloads.
- A new test-only dependency, `pgregory.net/rapid`, through the minimal-dependency audit.
- A new scheduled workflow for the fuzz run.
