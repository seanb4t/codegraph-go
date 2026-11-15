## MODIFIED Requirements

### Requirement: A signature selects one overload

`node`, `callers`, `callees` and `impact` SHALL accept `--signature <types>`, which selects one overload by its parameter types. `callers` on one overload SHALL also list the set's ambiguous callers, marked ambiguous, and SHALL NOT list a caller that resolved to another overload.

#### Scenario: Callers of one overload

- **WHEN** `log(String)` and `log(String, Throwable)` exist, one file calls `log("a")` and another calls `log("b", e)`, and a user runs `codegraph callers log --signature "String, Throwable"`
- **THEN** only the call `log("b", e)` is listed

#### Scenario: A signature that matches no overload

- **WHEN** a user runs `codegraph callers log --signature "int"` and no `log(int)` exists
- **THEN** the command reports that no overload of `log` has that signature and lists the signatures that exist

## ADDED Requirements

### Requirement: A line narrows the definitions that node shows (NODE-03)

`node` SHALL accept `--line <n>`, where `n` is a 1-based line number and `0` means no line. When the name has more than one definition, `node` SHALL keep the definitions whose lines contain `n`. When none contains `n`, it SHALL keep the one definition whose first line is nearest to `n`. The line SHALL never leave `node` with no definition. `callers`, `callees` and `impact` select by `--signature` and `--file` instead.

#### Scenario: A line inside one overload

- **WHEN** `Logger.java` declares `log(String msg)` on line 4 and `log(String msg, Throwable t)` on line 6, and a user runs `codegraph node log --line 6`
- **THEN** the output shows only `log` at `Logger.java:6`, with the signature `(String msg, Throwable t)`

#### Scenario: A line inside no definition

- **WHEN** a user runs `codegraph node log --line 7` on the same file
- **THEN** the output shows only `log` at `Logger.java:6`, the definition whose first line is nearest to line 7

#### Scenario: Line zero selects nothing

- **WHEN** a user runs `codegraph node log --line 0` on the same file
- **THEN** the output reports both definitions of `log`, as it does with no `--line`
