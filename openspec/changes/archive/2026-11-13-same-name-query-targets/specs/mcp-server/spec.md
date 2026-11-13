# Spec Delta

## ADDED Requirements

### Requirement: A table over several definitions names the definition of each row

When `codegraph_callers`, `codegraph_callees` or `codegraph_impact` spans definitions, as the `graph-queries` capability defines it, the result SHALL list the definitions before the table. If the query finds no row, the result SHALL still list them. The table SHALL have a Definition column with the name of each row's definition. The cell SHALL be empty for a row that names no definition. Any other result SHALL have no such list and no such column.

#### Scenario: Callers of a name with two definitions

- **WHEN** `a.py:1` and `b.py:1` each declare `def run()`, `main` calls the one in `a.py`, and an agent calls `codegraph_callers` with `symbol: "run"`
- **THEN** the text lists `a.py:1` and `b.py:1` as the 2 definitions of `run`
- **AND** the table has the columns Name, Kind, Location and Definition, and the row for `main` holds `a.py:1` under Definition

#### Scenario: No caller of a name with two definitions

- **WHEN** `a.py:1` and `b.py:1` each declare `def run()`, no function calls either, and an agent calls `codegraph_callers` with `symbol: "run"`
- **THEN** the text says that no callers were found
- **AND** the text lists `a.py:1` and `b.py:1` as the 2 definitions of `run`

#### Scenario: An impact row that the walk reached

- **WHEN** `a.go:1` and `b.go:1` in two packages each declare `func Reset()`, `main` calls both, and an agent calls `codegraph_impact` with `symbol: "Reset"`
- **THEN** the two rows for `Reset` hold `a.go:1` and `b.go:1` under Definition
- **AND** the row for `main` has an empty Definition cell

#### Scenario: Callers of a name with one definition

- **WHEN** one function `Alpha` exists, `Beta` calls it, and an agent calls `codegraph_callers` with `symbol: "Alpha"`
- **THEN** the table has exactly the columns Name, Kind and Location

#### Scenario: The input schema of the three tools

- **WHEN** a client reads the input schemas of `codegraph_callers`, `codegraph_callees` and `codegraph_impact`
- **THEN** each one declares an optional string property `file`
