## ADDED Requirements

### Requirement: Node lists the readers and writers of a data symbol

For a variable, a constant or a field, `node` and `codegraph_node` MUST print the line `**Accessed by ←**` after `**Called by ←**`. The line MUST list each definition that has a `references` edge with an access role to the symbol, once, as `name (path:line, role)`. The role MUST be the stored `read`, `write` or `read_write`. A file node is not a definition, and its edges MUST NOT give an entry. The entries MUST be ordered by file, then name, then line.

#### Scenario: A method reads and writes a field

- **WHEN** `p/Account.java` declares the field `balance` and the method `deposit` at line 7, `deposit` runs `balance += amount` and `return balance`, and a user runs `codegraph node balance`
- **THEN** the output contains the line `**Accessed by ←** deposit (p/Account.java:7, read_write)`

#### Scenario: Readers in two files

- **WHEN** `b.go` declares `func second()` and `a.go` declares `func first()`, both read the package variable `count`, and a user runs `codegraph node count`
- **THEN** `**Accessed by ←**` lists `first (a.go:...)` before `second (b.go:...)`

#### Scenario: Module-level code writes a variable

- **WHEN** `main.go` declares `var x = f()`, no function reads or writes `x`, and a user runs `codegraph node x`
- **THEN** the output contains the line `**Accessed by ←** ` with an empty list

### Requirement: Node lists the initializer value of a data symbol

For a variable, a constant or a field, `node` and `codegraph_node` MUST print the line `**Initialized to →**` after `**Accessed by ←**`. The line MUST list the targets of the symbol's `references` edges with `binding=init`, as `name (path:line)`.

#### Scenario: A variable holds a function

- **WHEN** `main.go` declares `func start()` at line 3 and `var run = start`, and a user runs `codegraph node run`
- **THEN** the output contains the line `**Initialized to →** start (main.go:3)`

### Requirement: The data lines appear only for data symbols

The single-definition view of a data symbol MUST print `**Accessed by ←**` and `**Initialized to →**`, also for an empty list. If a list is empty, a section of the multi-definition view MUST NOT print its line. The output for any kind other than a variable, a constant or a field MUST NOT change.

#### Scenario: Nothing accesses a constant

- **WHEN** a Go file declares `const limit = 3`, no code reads `limit`, and a user runs `codegraph node limit`
- **THEN** the output ends with the lines `**Accessed by ←** ` and `**Initialized to →** `, each with an empty list

#### Scenario: A field and a method share a name

- **WHEN** `p/Account.java` declares the field `size` and the method `size()` at line 11, the method returns the field, and a user runs `codegraph node size`
- **THEN** the field's section contains the line `**Accessed by ←** size (p/Account.java:11, read)`
- **AND** the method's section contains no `**Accessed by ←**` line and no `**Initialized to →**` line

#### Scenario: A function keeps its output

- **WHEN** a user runs `codegraph node deposit` on the `p/Account.java` fixture
- **THEN** the output contains no `**Accessed by ←**` line and no `**Initialized to →**` line

### Requirement: The graph UI carries the data lines

For a data symbol, the graph UI's node detail MUST carry the entries of `**Accessed by ←**` with their roles, and the entries of `**Initialized to →**`. The browse view MUST show each of the two lists that has an entry.

#### Scenario: The graph UI asks for a field

- **WHEN** the graph UI calls `GetNodeDetail` for `balance` on the `p/Account.java` fixture
- **THEN** the response's `accessed_by` holds one entry, whose node is `deposit` and whose access is `read_write`
- **AND** its `initialized_to` is empty
