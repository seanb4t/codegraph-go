# Spec Delta

## MODIFIED Requirements

### Requirement: A full index of the same tree gives the same graph (INDX-02)

Run `codegraph index --force` twice over the same tree at the same commit. The two stores SHALL hold the same records, byte for byte: nodes, edges, file records and excluded-file records. The metadata record SHALL also be the same, except its last-sync time. The `--workers` value and the number of CPUs SHALL not change the result. The order in which the indexer reads and resolves files SHALL not change it either.

#### Scenario: Two fresh indexes

- **WHEN** a user runs `codegraph index --force` twice over one unchanged tree, into two empty stores
- **THEN** the two stores hold the same records, and their metadata differs only in the last-sync time

#### Scenario: A different worker count

- **WHEN** a user indexes one tree with `codegraph index --force --workers 1`, and then with `--workers 8`
- **THEN** the two stores hold the same records, and their metadata differs only in the last-sync time

#### Scenario: A different claim order

- **WHEN** one tree is indexed twice with 4 workers, once with the files taken in path order and once in reverse path order
- **THEN** the two stores hold the same records, and their metadata differs only in the last-sync time

#### Scenario: Duplicate edges at one position

- **WHEN** two references in one file resolve to the same source, kind and target at the same line and column
- **THEN** the store holds one edge for them, and two fresh indexes keep the same one

## ADDED Requirements

### Requirement: One node merges the copies of an id that several files emit
Two or more files can emit a node with one id, for example the parts of a C# partial type. The store SHALL then hold one node for that id. Its visibility SHALL be the first one in path order that differs from the language default. If no copy differs from the default, its visibility SHALL be the default. If any copy is exported, the node SHALL be exported. Its other fields SHALL come from the first copy in path order, and within one file from the first copy that the file emits.

#### Scenario: A public part and a part with no modifier
- **WHEN** `A.cs` declares `partial class Order` and `B.cs` declares `public partial class Order` in one namespace
- **THEN** the store holds one `Order` node with visibility `public`, and it is exported

#### Scenario: Every part has the default visibility
- **WHEN** `A.cs` and `B.cs` each declare `partial class Order` with no access modifier
- **THEN** the store holds one `Order` node with visibility `internal`

#### Scenario: A route that one file registers twice
- **WHEN** one Express file registers `GET /x` at line 3 and again at line 7
- **THEN** the store holds one route node for `GET /x`, and its start line is 3

#### Scenario: A sync of one part
- **WHEN** the two files of the public part scenario are indexed, `B.cs` drops `public`, and `codegraph sync` runs
- **THEN** the store holds the same records as `codegraph index --force` over the changed tree, and the `Order` node has visibility `internal`

#### Scenario: A sync adds a part to an indexed type
- **WHEN** `A.cs` declares `public partial class Order`, the tree is indexed, `B.cs` is added with `partial class Order` and no modifier, and `codegraph sync` runs
- **THEN** the store holds the same records as `codegraph index --force` over the changed tree, and the `Order` node has visibility `public`

#### Scenario: A sync deletes the only part
- **WHEN** `A.cs` declares `partial class Order : IA` as the only part, the tree is indexed, `A.cs` is deleted, and `codegraph sync` runs
- **THEN** the store holds the same records as `codegraph index --force` over the changed tree, and no `Order` node and no edge from `Order` remains

#### Scenario: A sync after one part is deleted
- **WHEN** `A.cs` declares `partial class Order : IA` and `B.cs` declares `partial class Order : IB`, the tree is indexed, `B.cs` is deleted, and `codegraph sync` runs
- **THEN** the store holds the same records as `codegraph index --force` over the changed tree, and no edge from `Order` to `IB` remains
