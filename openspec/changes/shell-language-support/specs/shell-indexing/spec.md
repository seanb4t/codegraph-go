# Spec Delta

## Purpose

Expose useful static relationships in repository shell sources. Keep dialect selection and runtime-dependent limits visible so users can judge the graph.

## ADDED Requirements

### Requirement: Separate dialect identities

CodeGraph MUST represent POSIX shell, Bash, Zsh, and Fish as distinct languages. A dialect MUST pass its declared extraction cases before registration. Shared parser infrastructure MUST NOT imply shared language coverage.

#### Scenario: A Fish script contains Fish syntax
- **WHEN** a file is selected as Fish
- **THEN** CodeGraph uses the Fish contract and reports its language as `fish`
- **AND** it does not reinterpret the file as Bash

### Requirement: Deterministic dialect discovery

Discovery MUST recognize `.sh`, `.bash`, `.zsh`, and `.fish` files, supported shell shebangs, and documented startup filenames. A recognized shebang MUST take precedence over an extension or startup filename. An explicit unsupported interpreter MUST prevent a misleading shell fallback. Unidentified extensionless files MUST remain unsupported.

#### Scenario: Bash uses the generic shell extension
- **WHEN** `build.sh` starts with `#!/usr/bin/env bash`
- **THEN** discovery selects Bash

#### Scenario: A shell filename names another interpreter
- **WHEN** `build.sh` starts with `#!/usr/bin/env python3`
- **THEN** discovery does not index it as POSIX shell or Bash
- **AND** the exclusion identifies the interpreter mismatch

#### Scenario: An extensionless script selects Zsh
- **WHEN** an extensionless file has a supported Zsh shebang
- **THEN** discovery selects Zsh without executing the interpreter

### Requirement: Bounded startup file discovery

Discovery MUST include documented startup files within the repository, including `.config/fish/config.fish` and Fish function and configuration directories. Directory exceptions MUST remain path-specific. Discovery and file watching MUST use equivalent inclusion rules and preserve repository boundaries and existing size limits.

#### Scenario: A repository contains Fish configuration
- **WHEN** `.config/fish/functions/prompt.fish` changes
- **THEN** initial discovery and subsequent file watching both include the file
- **AND** unrelated hidden directories remain excluded

#### Scenario: A source link escapes the repository
- **WHEN** a shell file or sourced path points outside the repository through a symbolic link
- **THEN** CodeGraph does not traverse that path as part of shell indexing

### Requirement: Functions and variable bindings

Each dialect MUST expose named functions and statically named variable bindings within its declared scope rules. Repeated assignments to one binding MUST NOT create duplicate declarations. Command arguments, comments, quoted literals, and literal heredoc bodies MUST NOT create false declarations.

#### Scenario: A shell variable changes value
- **WHEN** a script assigns the same global variable twice
- **THEN** the graph exposes one binding and preserves write access without duplicate declarations

#### Scenario: A Fish function declares a local binding
- **WHEN** `set -l` introduces a local name inside a Fish function
- **THEN** that binding remains distinct from the same global name

#### Scenario: Declaration-like text is literal
- **WHEN** a quoted literal or literal heredoc contains function syntax
- **THEN** CodeGraph does not emit a function for that text

### Requirement: Static references and direct calls

CodeGraph MUST record statically resolved variable reads and direct calls to known shell functions. It MUST respect dialect-specific scope and shadowing. External commands, dynamic command names, and runtime-dependent bindings MUST NOT become invented repository functions.

#### Scenario: A literal function command appears in a pipeline
- **WHEN** a pipeline invokes an unambiguous function declared in the analyzed source
- **THEN** the graph records the call from its containing source symbol

#### Scenario: A variable supplies the command name
- **WHEN** a command invokes the contents of a variable
- **THEN** CodeGraph does not claim a definite function target from the variable's spelling

#### Scenario: Dynamic scope prevents a unique binding
- **WHEN** a variable read depends on the calling function's runtime scope
- **THEN** CodeGraph does not resolve it to an unrelated same-named global

### Requirement: Literal source relationships

CodeGraph MUST record literal `.` or `source` relationships to eligible repository files. Relative-path candidates MUST carry the documented static path assumption. Runtime-dependent paths MUST remain unresolved. The graph MUST distinguish source relationships from calls and must terminate on source cycles.

#### Scenario: A relative source path has one repository candidate
- **WHEN** a script sources `./lib.sh` and that file exists beside the script
- **THEN** the graph records the candidate with its script-directory assumption
- **AND** the graph does not claim that the runtime working directory is known

#### Scenario: A source path requires environment expansion
- **WHEN** a script sources a path that depends on an environment variable
- **THEN** CodeGraph retains the unresolved relationship without executing the script or reading the runtime environment

#### Scenario: Two files source each other
- **WHEN** each file contains a literal source path to the other
- **THEN** indexing terminates and retains the source relationships without duplicate declarations

### Requirement: Partial parsing is visible

A shell parse failure MUST appear through existing indexing error reporting. CodeGraph MUST NOT report a file with rejected syntax as fully indexed. Recoverable declarations MUST follow the existing partial-result policy, and unresolved syntax MUST remain visible in evaluation results.

#### Scenario: Unsupported Zsh syntax appears beside a function
- **WHEN** the selected parser cannot parse part of the file
- **THEN** indexing reports the parse limitation
- **AND** a nonempty graph does not count as full acceptance for that file

### Requirement: Public query and synchronization support

Definitions, callers, callees, impact, and explore MUST handle the new shell nodes and supported relationships. Existing response envelopes MUST remain compatible. Renames, deletion, dialect changes, and edits to sourced files MUST remove stale facts and agree with a fresh index.

#### Scenario: A file changes dialect
- **WHEN** an edit changes a script's recognized shebang from Bash to Fish
- **THEN** synchronization removes the old Bash facts and indexes the new Fish facts

#### Scenario: A sourced function disappears
- **WHEN** an edit deletes a function from a sourced repository file
- **THEN** the updated graph contains no resolved references to the removed function

### Requirement: Runtime-free indexing

Shell indexing MUST NOT execute source files, startup files, shell commands, command substitutions, or parser installation scripts. Released binaries MUST NOT require a shell interpreter for parsing. Parser dependencies MUST support the existing macOS and Linux release targets.

#### Scenario: Source text contains a side effect
- **WHEN** indexing encounters a command substitution that writes a file
- **THEN** indexing extracts only source facts and the side effect never occurs

### Requirement: Dialect-specific acceptance evidence

Each dialect MUST have independent expected declarations, relationships, public query answers, and negative cases. Documentation MUST distinguish supported constructs, partial resolution, and unsupported syntax. A passing Bash suite MUST NOT establish Zsh or Fish support.

#### Scenario: One dialect passes and another fails
- **WHEN** Bash acceptance passes but a required Fish case fails
- **THEN** the report preserves separate results and prevents a complete Fish support claim
