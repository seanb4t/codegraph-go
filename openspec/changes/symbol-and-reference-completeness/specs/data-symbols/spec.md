# Spec Delta

## Purpose

Let users find statically named variables and fields and follow their relationships. Preserve useful reference paths without inventing runtime call targets.

## ADDED Requirements

### Requirement: Module and package data declarations

CodeGraph MUST expose statically named package or module bindings in Go, TypeScript, JavaScript, TSX, and Python. Declaration lists MUST preserve each binding. Go `_` and JavaScript or TypeScript destructuring holes MUST NOT become symbols. Python, JavaScript, and TypeScript `_` identifiers within this declaration scope MUST remain symbols. Callable declarations MUST NOT acquire duplicate data nodes.

#### Scenario: Grouped and destructured bindings
- **WHEN** supported source declares multiple package or module bindings in one statement
- **THEN** each statically named binding is discoverable with its owner and source location
- **AND** unrelated declarations retain their identities

#### Scenario: Existing exported arrow function
- **WHEN** an exported TypeScript constant already represents an arrow function
- **THEN** its existing callable identity remains the single declaration identity

### Requirement: Deterministic declaration kinds

Go package `var` and `const` MUST use variable and constant kinds. JavaScript and TypeScript module `var`/`let` MUST use variable, and `const` MUST use constant. Direct module arrow/function expressions MUST use function, with one identity regardless of export. Callable aliases MUST retain their data kind. Python module assignments MUST use variable. Named type members MUST use field, including Java final fields and callable class fields.

#### Scenario: Modifiers do not invent new declaration kinds
- **WHEN** source declares an uppercase Python assignment, a Java static final field, and a TypeScript readonly field
- **THEN** their kinds are variable, field, and field respectively
- **AND** supported modifiers remain metadata

#### Scenario: A module callable is not exported
- **WHEN** a JavaScript module binds a direct function expression to a nonexported constant
- **THEN** that declaration has one function identity
- **AND** a separate alias to that function retains its variable or constant kind

### Requirement: Statically named fields

CodeGraph MUST expose named Go struct fields, TypeScript and JavaScript class fields, Java class and interface fields, and Python class attributes. It MUST expose Python instance attributes assigned through the declared instance receiver. Repeated writes MUST resolve to one logical field per owner and name.

#### Scenario: Multiple fields share a declaration
- **WHEN** a Go struct or Java class declares several named fields together
- **THEN** each field is discoverable under the correct containing type

#### Scenario: Python instance attribute repeats
- **WHEN** two methods assign the same attribute through their declared instance receiver
- **THEN** both assignments reference one field owned by that class
- **AND** a same-named field in another class remains distinct

#### Scenario: A Python instance write shadows a base class attribute
- **WHEN** a subclass method assigns `self.value` and its base declares class attribute `value`
- **THEN** the assignment establishes the subclass field and does not become a write to the base attribute

#### Scenario: A field name is computed dynamically
- **WHEN** a field name depends on runtime computation
- **THEN** CodeGraph does not invent a named field from the expression text

#### Scenario: A field and method share a name
- **WHEN** a Java class declares field `size` and method `size()`
- **THEN** owner-qualified data lookup resolves field reads while calls to `size()` resolve to the method
- **AND** the field does not enter the bare-name call index in fresh or incremental indexing

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

### Requirement: Bounded inherited field resolution

Java and TypeScript field lookup MUST follow uniquely established base-class chains to accessible declaring owners, unless a derived field shadows the name. Python receiver assignments MUST belong to the lexical class. Python reads MUST prefer established instance fields, then uniquely established class chains. Ambiguous ownership, inaccessible private fields, descriptors, and dynamic method resolution MUST remain unresolved.

#### Scenario: An inherited Java or TypeScript field is read
- **WHEN** a derived class reads an accessible field declared only in its established base class
- **THEN** the reference targets the base field
- **AND** adding a same-named derived field changes that reference to the derived owner

#### Scenario: A Python descriptor controls an attribute
- **WHEN** an attribute read depends on descriptor behavior rather than an established data field
- **THEN** CodeGraph leaves the data target unresolved

### Requirement: Conservative calls through bindings

CodeGraph MUST resolve calls through a unique static callable binding within a named analysis context. Evidence MUST retain the binding path and context. Competing assignments within that context, unknown receivers, or ambiguous targets MUST prevent a definitive target. Caller queries MUST represent calls rather than arbitrary references. Equal call edges MUST merge direct and derived evidence deterministically.

#### Scenario: A unique imported function binding is invoked
- **WHEN** a call invokes a variable bound to one imported function without competing writes in the analyzed scope
- **THEN** callers, callees, and impact can follow the call to that function
- **AND** the graph retains the variable's binding relationship

#### Scenario: A mutable binding has competing targets
- **WHEN** the analyzed source assigns different callable values to one variable
- **THEN** CodeGraph retains the known value references
- **AND** it does not claim one definitive call target from the initializer alone

#### Scenario: A mutable function-kind binding changes value
- **WHEN** a JavaScript or TypeScript module reassigns a function-expression binding declared with `let` or `var`
- **THEN** calls through that binding pass the same competing-write analysis regardless of its function kind
- **AND** no direct name lookup bypasses that analysis

#### Scenario: A data binding is invoked
- **WHEN** a caller invokes a variable that uniquely binds a function
- **THEN** its access to the variable is a read reference and its call targets the function
- **AND** no calls edge targets the variable node

#### Scenario: Direct and alias calls share endpoints
- **WHEN** one function calls a target directly and through a supported alias
- **THEN** one call edge retains direct evidence and the alias path with its context
- **AND** reversed extraction order produces the same evidence

### Requirement: Go production and test binding contexts

Go production binding analysis MUST exclude `_test.go` writes without excluding test files from indexing. Test analysis MUST include production and test writes. Test callers MUST use test context. Non-test callers MUST carry evidence only for contexts that establish a unique target. Evidence MUST distinguish `go-production` from `go-test` and MUST NOT imply equivalent runtime behavior.

#### Scenario: Tests replace the daemon function binding
- **WHEN** `internal/cli/daemon.go` calls `daemonList` and `internal/cli/daemon_test.go` assigns a competing stub
- **THEN** production evidence connects the non-test caller to `daemon.List`
- **AND** conflicting test-context call evidence is absent while test declarations and known value references remain indexed

#### Scenario: Production code replaces the binding
- **WHEN** non-test source assigns a competing target to `daemonList`
- **THEN** the initializer alone cannot establish the production call target

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

### Requirement: Dependency-aware reference updates

Synchronization MUST persist binding-write contributions and lookup dependencies, including unresolved bindings and inherited owner/name chains. Changes to write sets, owners, or shadowing declarations MUST re-resolve unchanged dependent files. Added and removed conflicts MUST update call evidence for each analysis context. The batch MUST commit dependency metadata and resulting facts together.

#### Scenario: Another file adds then removes a competing write
- **WHEN** a production file adds a competing assignment to a binding used by an unchanged caller
- **THEN** synchronization removes the caller's unsupported derived call evidence
- **AND** removing that assignment restores the same evidence as a fresh index

#### Scenario: A subclass gains a shadowing field
- **WHEN** a class edit adds a field that shadows its base field
- **THEN** unchanged files that used the inherited lookup re-resolve to the subclass field

### Requirement: Explicit completeness boundary

Acceptance evidence MUST cover each supported language and declaration form with independently expected facts. Documentation MUST distinguish supported static forms from dynamic or deferred forms. Unrestricted locals, reflection, Java record components, enum-specific declarations, and general overload resolution remain outside this change.

#### Scenario: Unsupported forms appear beside supported fields
- **WHEN** a source file contains both supported fields and deferred constructs
- **THEN** the supported fields pass exact acceptance cases
- **AND** the evaluation retains deferred constructs as visible coverage gaps
