# Spec Delta

## Purpose

How the index represents the source code: which nodes a declaration gets, how node ids stay stable, and how an incremental sync and the reported counts agree with the store.

## ADDED Requirements

### Requirement: A node id depends only on the declaration (D-02)

The indexer SHALL compute a node id from the declaration's kind, qualified name and file path, plus an ordinal for a repeated declaration. An edit SHALL NOT change the id of a declaration whose kind, qualified name, file path and ordinal stay the same. Two runs of `codegraph index` over the same tree SHALL produce the same ids.

#### Scenario: An edit above a declaration

- **WHEN** a file declares `function load()`, and a user adds ten lines above it and runs `codegraph sync`
- **THEN** `load` keeps the node id it had before the edit

#### Scenario: Two full indexes of one tree

- **WHEN** `codegraph index --force` runs twice over the same tree
- **THEN** both runs store the same set of node ids

### Requirement: Each declaration has its own node

When a file declares two or more symbols with the same kind and qualified name, the indexer SHALL store one node for each. The first in source order SHALL keep the id that a single declaration gets. Each later one SHALL get an id that also depends on its ordinal. Names and qualified names SHALL NOT change. A C# `partial` type SHALL keep one node for all of its parts.

#### Scenario: A helper declared in several callbacks

- **WHEN** a TypeScript file declares `function nextGeometry()` inside eight `it(...)` callbacks, and the repository is indexed
- **THEN** `codegraph search nextGeometry` lists eight results, each with its own line

#### Scenario: The first copy keeps its id

- **WHEN** a file declares `function e()` once, is indexed, and a second `function e()` is then added below the first
- **THEN** after `codegraph sync`, the first `e` keeps its node id and the second `e` has a different id

#### Scenario: Two classes with one name

- **WHEN** a TypeScript file declares `class Model` with method `load` in one callback, and `class Model` with method `save` in another callback
- **THEN** the index holds two `Model` nodes, the first contains only `load`, and the second contains only `save`

#### Scenario: A C# partial class

- **WHEN** two files declare `partial class Order` in the same namespace
- **THEN** the index holds one node for `Order`

### Requirement: A reference to a repeated name resolves to the first copy (RES-01)

A file can declare one name more than once with the same kind and qualified name. A reference that resolves to that name SHALL target the first copy in source order. This SHALL hold after `codegraph index` and after every `codegraph sync`.

#### Scenario: Another file calls a repeated function

- **WHEN** `a.ts` declares `function e()` three times and exports the first, and `b.ts` imports `e` and calls it
- **THEN** `codegraph callers` on the first `e` lists the call from `b.ts`, and the other two copies have no caller from `b.ts`

### Requirement: An incremental sync gives the same graph as a fresh index (INDX-03, INDX-04)

A user can create, edit, rename and delete files in any sequence and run `codegraph sync` after each step. After the last sync, the store SHALL hold the same nodes and edges as `codegraph index --force` over the final tree, except route nodes and their edges. Route detection runs only in a full index, a limitation that `docs/LANGUAGE-CAPABILITY-MATRIX.md` documents.

#### Scenario: A sequence of edits

- **WHEN** a user adds a file, renames a function in it, deletes a second file, and runs `codegraph sync` after each step
- **THEN** the store holds the same nodes and edges as a fresh index of the final tree

#### Scenario: A repeated declaration is removed

- **WHEN** a file declares `function e()` twice, is indexed, and the first copy is then deleted and `codegraph sync` runs
- **THEN** the store holds the same nodes and edges as a fresh index of the final file

### Requirement: Each edge connects two stored nodes (INDX-04)

Each edge in the store SHALL have a source and a target that are nodes in the store, after `codegraph index` and after every `codegraph sync`.

#### Scenario: A file that other files call is deleted

- **WHEN** file `b.go` calls a function in `a.go`, and `a.go` is deleted and `codegraph sync` runs
- **THEN** no stored edge names a node from `a.go`

### Requirement: The reported node count equals the stored nodes

The `nodes=` count that `codegraph index` and `codegraph sync` print SHALL equal the number of nodes in the store, which `codegraph status` reports as `Nodes:`.

#### Scenario: A file with repeated declarations is synced many times

- **WHEN** a file that declares `function e()` three times is edited and synced ten times
- **THEN** the `nodes=` count of the last sync equals the `Nodes:` count of `codegraph status`

### Requirement: An index from before unique declaration ids is rebuilt once

A store written by a version without unique declaration ids SHALL be rebuilt from scratch by its next `codegraph sync`. Later syncs SHALL be incremental. The rebuild SHALL also correct the stored node count.

#### Scenario: The first sync after an upgrade

- **WHEN** a store written before this change has a drifted node count, and the new version runs `codegraph sync`
- **THEN** the sync rebuilds the whole graph, and its `nodes=` count equals the `Nodes:` count of `codegraph status`

#### Scenario: Records that a fresh index does not have

- **WHEN** a store written before this change still holds the node of a deleted file, and the new version runs `codegraph sync`
- **THEN** the store holds the same nodes and edges as a fresh index of the tree

#### Scenario: The syncs after the rebuild

- **WHEN** the new version runs `codegraph sync` a second and a third time, each with one file changed
- **THEN** each of those syncs reparses only that file
