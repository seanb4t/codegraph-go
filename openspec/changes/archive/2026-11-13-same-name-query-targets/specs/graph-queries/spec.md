# Spec Delta

## ADDED Requirements

### Requirement: A bare name with several definitions reports across them

A definition of a name is a node with that name that is not an `overload_set` node. When a bare name has more than one definition and no `--signature` is given, `callers`, `callees` and `impact` SHALL report across every definition. The result SHALL NOT depend on the ids of the nodes. This rule includes the members of an overload set and a definition outside the set.

#### Scenario: Two functions of one name in two files

- **WHEN** `a.py` declares `def run()` whose body calls `load()`, `b.py` declares `def run()` whose body calls `save()`, and a user runs `codegraph callees run`
- **THEN** both `load` and `save` are listed

#### Scenario: A typing stub and the real function

- **GIVEN** `lib.py` declares `to_list` under `@overload` and then declares the real `to_list`
- **AND** the function `build` in `app.py` calls `to_list(x)`
- **WHEN** a user runs `codegraph callers to_list`
- **THEN** `build` is listed

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

If a query starts from definitions that are not all members of one overload set, it spans definitions. The output of such a query SHALL list the definitions. If the query finds no result, the output SHALL still list them. The `--json` output SHALL carry the list in a `definitions` field. A query over one overload set alone SHALL NOT list definitions.

#### Scenario: The definitions are listed

- **WHEN** `a.py:1` and `b.py:1` each declare `def run()`, `main` calls the one in `a.py`, and a user runs `codegraph callers run`
- **THEN** the output states that `run` has 2 definitions and lists `a.py:1` and `b.py:1`

#### Scenario: A query that spans definitions and finds no result

- **WHEN** `a.py:1` and `b.py:1` each declare `def run()`, no function calls either, and a user runs `codegraph callers run`
- **THEN** the output states that `run` has 0 callers
- **AND** the output lists `a.py:1` and `b.py:1` as the 2 definitions of `run`

#### Scenario: JSON output over several definitions

- **WHEN** `a.py:1` and `b.py:1` each declare `def run()`, and a user runs `codegraph callers run --json`
- **THEN** the JSON object has a `definitions` list with both definitions

#### Scenario: One overload set alone

- **WHEN** `Logger` declares `log(String)` and `log(String, Throwable)`, no other node is named `log`, and a user runs `codegraph callers log --json`
- **THEN** each entry of `callers` has its `signature` field
- **AND** the JSON object has no `definitions` field, and no entry has a `definition` field

### Requirement: A definition is named by its file and line

Where the output names a definition, the name SHALL be the file and the line of the definition, as `<file>:<line>`. A package has no file, and a type whose parts are in several files has none. Such a definition SHALL be named by its kind and its qualified name.

#### Scenario: A function and a package of one name

- **WHEN** a Go module has the package `example.com/app/cache`, `main.go:3` declares `func cache()`, and a user runs `codegraph callers cache`
- **THEN** the output lists the 2 definitions `main.go:3` and `package example.com/app/cache`

### Requirement: A row of a query that spans definitions names its definition

When a query spans definitions, each result of `callers` and `callees` SHALL name the definition it belongs to. A result that belongs to two definitions SHALL appear once for each. A caller marked ambiguous calls an overload set and not one definition, and SHALL name no definition. A callee marked ambiguous is itself an overload set, and SHALL name the definition that makes the call. The `--json` output SHALL carry the name in a `definition` field of the entry.

#### Scenario: A caller names the definition it calls

- **WHEN** `a.py:1` and `b.py:1` each declare `def run()`, `main` calls the one in `a.py`, and a user runs `codegraph callers run --json`
- **THEN** the entry for `main` has a `definition` field that holds `a.py:1`

#### Scenario: A signature that matches in two files

- **GIVEN** `Logger.java:4` and `Audit.java:7` each declare `log(String)`
- **AND** `Main.start` calls the one in `Logger.java`, and `Job.run` calls the one in `Audit.java`
- **WHEN** a user runs `codegraph callers log --signature "String" --json`
- **THEN** the entry for `Main.start` has the `definition` `Logger.java:4`, and the entry for `Job.run` has the `definition` `Audit.java:7`

#### Scenario: An ambiguous callee in a query that spans definitions

- **GIVEN** `Job.java` and `Main.java` each declare a method `run`, at lines 3 and 8
- **AND** the `run` in `Job.java` calls `log(null)`, and `Logger` declares `log(String)` and `log(Throwable)`
- **WHEN** a user runs `codegraph callees run --json`
- **THEN** the entry for `log` is marked ambiguous and its `definition` field is `Job.java:3`

#### Scenario: An ambiguous caller in a query that spans definitions

- **GIVEN** `Calc` declares `add(int)` and `add(String)`, and `Run.go` calls `add(null)`
- **AND** another file declares an unrelated function `add()`
- **WHEN** a user runs `codegraph callers add --json`
- **THEN** the entry for `Run.go` is marked ambiguous and has no `definition` field

### Requirement: Impact names each start definition

When an `impact` query spans definitions, each definition SHALL be a result that names itself. A symbol that the walk reaches SHALL appear once and name no definition.

#### Scenario: Impact reaches one symbol from two definitions

- **WHEN** `a.go` and `b.go` in two packages each declare `func Reset()`, `main` calls both, and a user runs `codegraph impact Reset --json`
- **THEN** `affected` holds both `Reset` definitions, each with its `definition` field
- **AND** `affected` holds `main` once, with no `definition` field

### Requirement: The order of the results does not depend on node ids

The results of `callers`, `callees` and `impact` SHALL be ordered by file, then name, then line. Results that are equal in all three SHALL be ordered by the name of their definition, then by their signature, then by their kind. The list of definitions SHALL be ordered by file, then line, then kind and qualified name.

#### Scenario: One caller of two definitions

- **WHEN** `a.go:1` and `b.go:1` in two packages each declare `func Reset()`, `main` calls both, and a user runs `codegraph callers Reset`
- **THEN** `main` is listed twice, first with the definition `a.go:1` and then with `b.go:1`

#### Scenario: Tied results of one definition

- **WHEN** one function `run` exists, a method `x` and a function `x` at the same line of `c.go` call it, and a user runs `codegraph callers run`
- **THEN** the function `x` is listed before the method `x`

### Requirement: A name with one definition keeps its output

When a bare name has exactly one definition, `callers`, `callees` and `impact` SHALL give the output they gave before this requirement. This holds byte for byte, for the text output and for `--json`, with one exception: results that are equal in file, name and line follow the order requirement above. The `--json` output SHALL NOT contain the `definitions` field or a `definition` field.

#### Scenario: One definition in JSON

- **WHEN** one function `Alpha` exists, `Beta` calls it, and a user runs `codegraph callers Alpha --json`
- **THEN** the JSON object has exactly the keys `symbol` and `callers`
- **AND** the entry for `Beta` has exactly the keys `name`, `kind`, `filePath` and `startLine`

#### Scenario: One definition in text

- **WHEN** one function `Alpha` exists, `Beta` calls it, and a user runs `codegraph callers Alpha`
- **THEN** the first line is `Alpha has 1 caller(s):`
- **AND** the output has no line that lists definitions

### Requirement: A file selects the definitions in it

`callers`, `callees` and `impact` SHALL accept `--file <path>`. With it, the query SHALL start only from the definitions whose file path contains the value. The comparison SHALL ignore letter case and SHALL treat `\` as `/`, as `node --file` does. `--file` and `--signature` together SHALL select the definitions that match both. When no definition matches, the command SHALL fail and list each file that holds a definition once.

#### Scenario: Callees of the definition in one file

- **WHEN** `a.py` declares `def run()` whose body calls `load()`, `b.py` declares `def run()` whose body calls `save()`, and a user runs `codegraph callees run --file a.py`
- **THEN** `load` is listed and `save` is not
- **AND** the output has no line that lists definitions

#### Scenario: A file named by the end of its path

- **WHEN** `src/app/models.py` and `tests/conftest.py` each declare `prepare_url`, and a user runs `codegraph callees prepare_url --file models.py`
- **THEN** only the callees of the definition in `src/app/models.py` are listed

#### Scenario: A file with no definition of the name

- **WHEN** `a.py` declares `def run()` twice, `b.py` declares `def run()`, and a user runs `codegraph callers run --file c.py`
- **THEN** the command exits non-zero and says that no definition of `run` is in a file that matches `c.py`
- **AND** the message lists `a.py` once and `b.py` once

#### Scenario: A file and a signature

- **GIVEN** `Logger.java` declares `log(String)` and `log(String, Throwable)`, and `Audit.java` declares `log(String)`
- **AND** `Main.start` calls `log("a")` of `Logger`, `Main.stop` calls `log("b", e)` of `Logger`, and `Job.run` calls `log("c")` of `Audit`
- **WHEN** a user runs `codegraph callers log --file Logger.java --signature "String"`
- **THEN** `Main.start` is listed, and `Main.stop` and `Job.run` are not

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
