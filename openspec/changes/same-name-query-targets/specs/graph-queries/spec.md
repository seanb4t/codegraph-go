# Spec Delta

## ADDED Requirements

### Requirement: A bare name with several definitions reports across them

A definition of a name is a node with that name that is not an `overload_set` node. When a bare name has more than one definition and no `--signature` is given, `callers`, `callees` and `impact` SHALL report across every definition. The result SHALL NOT depend on the ids of the nodes. This rule includes the members of an overload set and a definition outside the set.

#### Scenario: Two functions of one name in two files

- **WHEN** `a.py` declares `def run()` whose body calls `load()`, `b.py` declares `def run()` whose body calls `save()`, and a user runs `codegraph callees run`
- **THEN** both `load` and `save` are listed

#### Scenario: A typing stub and the real function

- **WHEN** a Python module declares `to_list` under `@overload` and then declares the real `to_list`, another module calls `to_list(x)`, and a user runs `codegraph callers to_list`
- **THEN** the calling function is listed

#### Scenario: A method and a test helper of one name

- **GIVEN** a class declares the method `prepare_url`, which calls `requote_uri`
- **AND** a test file declares a function `prepare_url` with no call in its body
- **WHEN** a user runs `codegraph callees prepare_url`
- **THEN** `requote_uri` is listed

#### Scenario: Impact over two definitions

- **WHEN** `a.go` declares `func Reset()` called by `A`, `b.go` in another package declares `func Reset()` called by `B`, and a user runs `codegraph impact Reset`
- **THEN** both `A` and `B` are listed as affected

#### Scenario: An overload set and a definition outside it

- **WHEN** `Logger` declares `log(String)` and `log(String, Throwable)`, another file declares an unrelated function `log()` called by `main`, and a user runs `codegraph callers log`
- **THEN** the callers of both overloads are listed
- **AND** `main` is listed

### Requirement: A result over several definitions lists them

A query spans definitions when it reports across definitions that are not all members of one overload set. The output of such a query SHALL list the definitions, each with its file and line. The `--json` output SHALL carry the list in a `definitions` field. A query over one overload set alone SHALL NOT list definitions.

#### Scenario: The definitions are listed

- **WHEN** `a.py:1` and `b.py:1` each declare `def run()`, `main` calls the one in `a.py`, and a user runs `codegraph callers run`
- **THEN** the output states that `run` has 2 definitions and lists `a.py:1` and `b.py:1`

#### Scenario: JSON output over several definitions

- **WHEN** `a.py:1` and `b.py:1` each declare `def run()`, and a user runs `codegraph callers run --json`
- **THEN** the JSON object has a `definitions` list with both definitions

#### Scenario: One overload set alone

- **WHEN** `Logger` declares `log(String)` and `log(String, Throwable)`, no other node is named `log`, and a user runs `codegraph callers log --json`
- **THEN** each entry of `callers` has its `signature` field
- **AND** the JSON object has no `definitions` field, and no entry has a `definition` field

### Requirement: A row of a query that spans definitions names its definition

When a query spans definitions, each result of `callers` and `callees` SHALL name the definition it belongs to by file and line. A result that belongs to two definitions SHALL appear once for each. In `impact`, each definition SHALL be a result that names itself, and a symbol that the walk reaches SHALL appear once and name no definition. The `--json` output SHALL carry the name in a `definition` field of the entry.

#### Scenario: A caller names the definition it calls

- **WHEN** `a.py:1` and `b.py:1` each declare `def run()`, `main` calls the one in `a.py`, and a user runs `codegraph callers run --json`
- **THEN** the entry for `main` has a `definition` field that holds `a.py:1`

#### Scenario: One caller of two definitions

- **WHEN** `a.go` and `b.go` in two packages each declare `func Reset()`, `main` calls both, and a user runs `codegraph callers Reset`
- **THEN** `main` is listed twice, once with each definition

#### Scenario: Impact reaches one symbol from two definitions

- **WHEN** `a.go` and `b.go` in two packages each declare `func Reset()`, `main` calls both, and a user runs `codegraph impact Reset --json`
- **THEN** `affected` holds both `Reset` definitions, each with its `definition` field
- **AND** `affected` holds `main` once, with no `definition` field

### Requirement: A name with one definition keeps its output

When a bare name has exactly one definition, `callers`, `callees` and `impact` SHALL give the output they gave before this requirement. This holds byte for byte, for the text output and for `--json`. The `--json` output SHALL NOT contain the `definitions` field or a `definition` field.

#### Scenario: One definition in JSON

- **WHEN** one function `Alpha` exists, `Beta` calls it, and a user runs `codegraph callers Alpha --json`
- **THEN** the JSON object has exactly the keys `symbol` and `callers`
- **AND** the entry for `Beta` has exactly the keys `name`, `kind`, `filePath` and `startLine`

#### Scenario: One definition in text

- **WHEN** one function `Alpha` exists, `Beta` calls it, and a user runs `codegraph callers Alpha`
- **THEN** the first line is `Alpha has 1 caller(s):`
- **AND** the output has no line that lists definitions

### Requirement: A file selects the definitions in it

`callers`, `callees` and `impact` SHALL accept `--file <path>`, a path from the repository root. With it, the query SHALL start only from the definitions of the name in that file. `--file` and `--signature` together SHALL select the definitions that match both. When the file holds no definition of the name, the command SHALL fail and list the files that hold one.

#### Scenario: Callees of the definition in one file

- **WHEN** `a.py` declares `def run()` whose body calls `load()`, `b.py` declares `def run()` whose body calls `save()`, and a user runs `codegraph callees run --file a.py`
- **THEN** `load` is listed and `save` is not
- **AND** the output has no line that lists definitions

#### Scenario: A file with no definition of the name

- **WHEN** `a.py` and `b.py` each declare `def run()`, and a user runs `codegraph callers run --file c.py`
- **THEN** the command exits non-zero, says that `c.py` has no definition of `run`, and lists `a.py` and `b.py`

#### Scenario: A file and a signature

- **WHEN** `Logger.java` declares `log(String)` and `log(String, Throwable)`, `Audit.java` declares `log(String)`, and a user runs `codegraph callers log --file Logger.java --signature "String"`
- **THEN** only the callers of `log(String)` in `Logger.java` are listed

### Requirement: The MCP tools select by file

The MCP tools `codegraph_callers`, `codegraph_callees` and `codegraph_impact` SHALL accept an optional `file` input with the meaning of `--file`. Without it, they SHALL report across every definition of the name, as the CLI commands do.

#### Scenario: An MCP callers call on a name with two definitions

- **WHEN** `a.py` and `b.py` each declare `def run()`, each has one caller, and an agent calls `codegraph_callers` with `symbol: "run"` and no `file`
- **THEN** the result lists both callers

#### Scenario: An MCP callees call with a file

- **GIVEN** `a.py` declares `def run()` whose body calls `load()`
- **AND** `b.py` declares `def run()` whose body calls `save()`
- **WHEN** an agent calls `codegraph_callees` with `symbol: "run"` and `file: "a.py"`
- **THEN** the result lists `load` and does not list `save`
