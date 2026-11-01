# Spec Delta

## Purpose

Let users find statically named variables and fields and follow their relationships. Preserve useful reference paths without inventing runtime call targets.

## ADDED Requirements

### Requirement: Module and package data declarations

CodeGraph MUST expose statically named package or module variables and constants in Go, TypeScript, JavaScript, TSX, and Python. Declaration lists MUST preserve each named binding. Discard bindings MUST NOT become symbols. Existing callable declarations MUST NOT acquire duplicate data nodes.

#### Scenario: Grouped and destructured bindings
- **WHEN** supported source declares multiple package or module bindings in one statement
- **THEN** each statically named binding is discoverable with its owner and source location
- **AND** unrelated declarations retain their identities

#### Scenario: Existing exported arrow function
- **WHEN** an exported TypeScript constant already represents an arrow function
- **THEN** its existing callable identity remains the single declaration identity

### Requirement: Statically named fields

CodeGraph MUST expose named Go struct fields, TypeScript and JavaScript class fields, Java class and interface fields, and Python class attributes. It MUST expose Python instance attributes assigned through the declared instance receiver. Repeated writes MUST resolve to one logical field per owner and name.

#### Scenario: Multiple fields share a declaration
- **WHEN** a Go struct or Java class declares several named fields together
- **THEN** each field is discoverable under the correct containing type

#### Scenario: Python instance attribute repeats
- **WHEN** two methods assign the same attribute through their declared instance receiver
- **THEN** both assignments reference one field owned by that class
- **AND** a same-named field in another class remains distinct

#### Scenario: A field name is computed dynamically
- **WHEN** a field name depends on runtime computation
- **THEN** CodeGraph does not invent a named field from the expression text

### Requirement: Ownership and initialization relationships

Data symbols MUST retain their declaring owner and statically resolvable initializer references. A callable value stored in a variable MUST retain a binding relationship to its target. Initialization MUST NOT create a call edge unless the initializer actually invokes a callable.

#### Scenario: Function value assignment
- **WHEN** a Go package declares `var daemonList = daemon.List`
- **THEN** the graph links the variable to `daemon.List` as a reference
- **AND** the assignment alone does not create a caller of `daemon.List`

#### Scenario: Initializer invokes a function
- **WHEN** a data initializer calls a statically resolved function
- **THEN** the graph records that call separately from the initializer's value references

### Requirement: Scoped data references

CodeGraph MUST link statically resolvable reads and writes to the correct data declaration. Reference metadata MUST distinguish read, write, and combined access. Local shadowing and equal names in other owners MUST NOT create incorrect data references.

#### Scenario: A local binding shadows a module variable
- **WHEN** a function reads a local name that shadows an indexed module variable
- **THEN** the graph does not link that read to the module variable

#### Scenario: One function reads and writes a field
- **WHEN** one source symbol both reads and writes the same field
- **THEN** the graph preserves both access roles even if storage combines the references into one edge

### Requirement: Conservative calls through bindings

CodeGraph MUST resolve calls through a uniquely established static callable binding. It MUST retain the binding path and identify the derived call's evidence. Competing assignments, unknown receivers, or ambiguous targets MUST NOT produce a definitive target. Caller queries MUST continue to represent calls rather than arbitrary references.

#### Scenario: A unique imported function binding is invoked
- **WHEN** a call invokes a variable bound to one imported function without competing writes in the analyzed scope
- **THEN** callers, callees, and impact can follow the call to that function
- **AND** the graph retains the variable's binding relationship

#### Scenario: A mutable binding has competing targets
- **WHEN** the analyzed source assigns different callable values to one variable
- **THEN** CodeGraph retains the known value references
- **AND** it does not claim one definitive call target from the initializer alone

### Requirement: User-visible data symbol access

Definitions and explore MUST expose the new data symbols and their reference relationships. CLI and MCP MUST preserve their existing response envelopes. Kind filters and graph presentation MUST accept the new field kind. A data read MUST NOT become a function caller.

#### Scenario: A user requests a field definition
- **WHEN** a user selects an indexed field through CLI or MCP
- **THEN** the response identifies its owner, kind, language, and source location

#### Scenario: A user asks for callers of a referenced function
- **WHEN** the function is stored in a variable but never invoked
- **THEN** the variable initializer does not appear as a caller

### Requirement: Fresh and updated index agreement

Existing indexes MUST gain the new data facts after the supported rebuild path. Adding, editing, renaming, and deleting data declarations MUST remove stale facts. A fresh index and an incrementally updated index MUST agree on the resulting semantic facts.

#### Scenario: An unchanged source file gains extraction support
- **WHEN** a user opens an index built before this capability
- **THEN** the index rebuilds through the existing migration mechanism
- **AND** unchanged files gain their supported data declarations and references

#### Scenario: A field moves to another owner
- **WHEN** an edit removes a field from one class and adds it to another
- **THEN** queries show the new owner and contain no references to the removed field

### Requirement: Explicit completeness boundary

Acceptance evidence MUST cover each supported language and declaration form with independently expected facts. Documentation MUST distinguish supported static forms from dynamic or deferred forms. Unrestricted locals, reflection, Java record components, enum-specific declarations, and general overload resolution remain outside this change.

#### Scenario: Unsupported forms appear beside supported fields
- **WHEN** a source file contains both supported fields and deferred constructs
- **THEN** the supported fields pass exact acceptance cases
- **AND** the evaluation retains deferred constructs as visible coverage gaps
