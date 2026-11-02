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

Discovery MUST inspect `.sh`, `.bash`, `.zsh`, `.fish`, startup names, and regular extensionless files within repository, directory, and size limits. Registered non-shell extensions MUST keep their classification. Recognized shebangs MUST take precedence. Unsupported interpreters on shell extensions or startup names MUST use `EXCLUSION_REASON_INTERPRETER_MISMATCH`. Extensionless files without a recognized interpreter MUST retain `EXCLUSION_REASON_UNSUPPORTED_EXTENSION`.

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

#### Scenario: An interpreter is outside the recognized set
- **WHEN** a shell extension or startup name specifies an interpreter other than `sh`, `bash`, `zsh`, or `fish`, including `dash`, `ash`, or `ksh`
- **THEN** discovery records an interpreter mismatch
- **AND** direct paths and supported `env` forms use the same exact interpreter set

#### Scenario: An extensionless non-shell script remains unsupported
- **WHEN** an eligible extensionless script names Python, Node, or another unrecognized interpreter
- **THEN** its exclusion remains `EXCLUSION_REASON_UNSUPPORTED_EXTENSION` rather than a shell interpreter mismatch

#### Scenario: A registered non-shell file has a shebang
- **WHEN** a `.py` or `.js` file contains a shell or unsupported shebang
- **THEN** its existing Python or JavaScript extension classification remains unchanged

### Requirement: Bounded startup file discovery

Discovery MUST include documented startup files. Fish exceptions MUST include only root-anchored `.config/fish/config.fish`, `.config/fish/functions/**/*.fish`, and `.config/fish/conf.d/**/*.fish`. Nested equivalents and hidden ancestors MUST remain excluded. Discovery, fsnotify registration and directory events, FSEvents filtering, and the no-daemon status walk MUST use equivalent rules within repository and size boundaries.

#### Scenario: A repository contains Fish configuration
- **WHEN** `.config/fish/functions/prompt.fish` changes
- **THEN** initial discovery and subsequent file watching both include the file
- **AND** unrelated hidden directories remain excluded

#### Scenario: A nested Fish directory resembles the allowed root
- **WHEN** `home/.config/fish/config.fish` or `.hidden/.config/fish/config.fish` exists inside the repository
- **THEN** the Fish exception does not include either path in discovery or watching

#### Scenario: Fish configuration changes without a daemon
- **WHEN** an included Fish file changes after the last synchronization and no daemon sidecar exists
- **THEN** the status staleness walk includes that file and reports the index as stale

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

CodeGraph MUST resolve literal `.` or `source` paths through a shell-specific resolver to existing eligible file nodes, using imports edges. It MUST NOT use Go module-prefix resolution or synthetic Go packages for these paths. Relative candidates MUST carry their static path assumption. Runtime-dependent paths MUST remain unresolved, and source cycles MUST terminate.

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

#### Scenario: Dialects use their supported source commands
- **WHEN** POSIX shell uses `.`, Bash or Zsh uses `.` or `source`, or Fish uses `source` with an eligible literal path
- **THEN** its dialect records the supported source relationship

### Requirement: Source-path dependency updates

CodeGraph MUST persist resolved and unresolved literal source candidates as repository-relative lookup paths. Added, restored, deleted, renamed, or newly ineligible targets MUST re-resolve unchanged importers and transitive dependents. Resolution MUST reapply repository boundaries and terminate on cycles. Synchronization MUST commit path dependencies and resulting graph facts together.

#### Scenario: A missing sourced file is restored
- **WHEN** an unchanged script has an unresolved literal source candidate and that eligible file appears at the recorded path
- **THEN** synchronization resolves its imports and supported function references exactly as a fresh index

### Requirement: Shell parse failures reject file extraction

A shell Tree-sitter error, missing node, or recognized construct outside the selected dialect contract MUST reject extraction for that file. Its file record MUST retain a named indexing error and count as failed. Synchronization MUST remove former extracted facts. Expected facts MUST remain in evaluation denominators. A sourced file MUST retain its independently selected dialect. Other languages MUST retain their existing parse behavior.

#### Scenario: Unsupported Zsh syntax appears beside a function
- **WHEN** the selected parser cannot parse part of the file
- **THEN** indexing reports the parse limitation
- **AND** the file retains an error record but no extracted declarations or outgoing relationships
- **AND** synchronization removes its former facts and invalidates dependent references

#### Scenario: A POSIX-selected file contains a Bash-only construct
- **WHEN** the shared grammar accepts a Bash array or `function` declaration in a `shell` file
- **THEN** the POSIX adapter rejects extraction and names that construct in the file's indexing error
- **AND** sourcing it from Bash does not change its dialect or bypass rejection

### Requirement: Public query and synchronization support

Definitions, callers, callees, impact, and explore MUST handle the new shell nodes and supported relationships. Existing response envelopes MUST remain compatible. Renames, deletion, dialect changes, and edits to sourced files MUST remove stale facts and agree with a fresh index.

#### Scenario: A file changes dialect
- **WHEN** an edit changes a script's recognized shebang from Bash to Fish
- **THEN** synchronization removes the old Bash facts and indexes the new Fish facts

#### Scenario: A sourced function disappears
- **WHEN** an edit deletes a function from a sourced repository file
- **THEN** the updated graph contains no resolved references to the removed function

#### Scenario: An existing index gains shell support
- **WHEN** normal synchronization discovers unchanged shell files absent from an older index
- **THEN** those files gain supported facts without a shell-specific rebuild marker

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
