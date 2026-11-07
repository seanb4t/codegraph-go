## ADDED Requirements

### Requirement: A full index of the same tree gives the same graph (INDX-02)

Run `codegraph index --force` twice over the same tree at the same commit. The two stores SHALL hold the same records, byte for byte: nodes, edges, file records and excluded-file records. The metadata record SHALL also be the same, except its last-sync time. The `--workers` value and the number of CPUs SHALL not change the result. The order in which the indexer reads and resolves files SHALL not change it either.

#### Scenario: Two fresh indexes

- **WHEN** a user runs `codegraph index --force` twice over one unchanged tree, into two empty stores
- **THEN** the two stores hold the same records, and their metadata differs only in the last-sync time

#### Scenario: A different worker count

- **WHEN** a user indexes one tree with `codegraph index --force --workers 1`, and then with `--workers 8`
- **THEN** the two stores hold the same records, and their metadata differs only in the last-sync time

#### Scenario: Duplicate edges at one position

- **WHEN** two references in one file resolve to the same source, kind and target at the same line and column
- **THEN** the store holds one edge for them, and two fresh indexes keep the same one
