# Spec Delta

## Purpose

How the index represents the source code: which nodes a declaration gets, how node ids stay stable, and how a reference resolves to a declaration. Also how an incremental sync and the reported counts agree with the store.

## ADDED Requirements

### Requirement: A node id depends only on the declaration (D-02)

The indexer SHALL compute a function or method id from the declaration's kind, qualified name, file path and signature key. Only declarations with identical keys SHALL get an ordinal. An edit elsewhere in the file SHALL NOT change the id of a declaration whose key stays the same. Two runs of `codegraph index` over the same tree SHALL produce the same ids.

#### Scenario: An edit above a declaration

- **WHEN** a file declares `function load()`, and a user adds ten lines above it and runs `codegraph sync`
- **THEN** `load` keeps the node id it had before the edit

#### Scenario: An overload is added above another

- **WHEN** a Java class declares `foo(int)`, is indexed, and a user adds `foo(long)` above it and runs `codegraph sync`
- **THEN** `foo(int)` keeps its node id and `foo(long)` gets a different id

#### Scenario: Two full indexes of one tree

- **WHEN** `codegraph index --force` runs twice over the same tree
- **THEN** both runs store the same set of node ids

### Requirement: The signature key is the same in every language

The signature key SHALL be the ordered parameter types as written, with a placeholder for an untyped parameter. It SHALL mark optional, default and variadic parameters. It SHALL NOT include parameter names. The same rule SHALL apply in every language.

#### Scenario: An untyped Python function

- **WHEN** a Python module declares `def f(a, b)`
- **THEN** its signature key is `(_,_)`

#### Scenario: A Go function

- **WHEN** a Go file declares `func f(a, b int)`
- **THEN** its signature key is `(int,int)`

#### Scenario: A Java variadic method

- **WHEN** a Java class declares `foo(int n, String... parts)`
- **THEN** its signature key is `(int,String...)`

### Requirement: Each declaration has its own node

When a file declares two or more symbols with the same kind and qualified name, the indexer SHALL store one node for each. Names and qualified names SHALL NOT change. A C# `partial` type SHALL keep one node for all of its parts, and a reopened Ruby class or module SHALL keep one node.

#### Scenario: A helper declared in several callbacks

- **WHEN** a TypeScript file declares `function nextGeometry()` inside eight `it(...)` callbacks, and the repository is indexed
- **THEN** `codegraph search nextGeometry` lists eight results, each with its own line

#### Scenario: Overloads

- **WHEN** a Java class declares `log(String)` and `log(String, Throwable)`, and the repository is indexed
- **THEN** the index holds two `log` nodes, each with its own signature and location

#### Scenario: Two classes with one name

- **WHEN** a TypeScript file declares `class Model` with method `load` in one callback, and `class Model` with method `save` in another callback
- **THEN** the index holds two `Model` nodes, the first contains only `load`, and the second contains only `save`

#### Scenario: A C# partial class

- **WHEN** two files declare `partial class Order` in the same namespace
- **THEN** the index holds one node for `Order`

### Requirement: Overloads of one name have a set node

When two or more functions or methods share a name under one owner, the indexer SHALL store one `overload_set` node for the name. The owner SHALL contain the set, and the set SHALL contain each overload. A name with one declaration SHALL NOT have a set node.

#### Scenario: A set under a class

- **WHEN** a C# class `Logger` declares three `Write` overloads
- **THEN** the index holds one `overload_set` node named `Write` under `Logger`, which contains the three overloads, and `Logger` contains the set

#### Scenario: A set across the parts of a partial type

- **WHEN** two files declare `partial class Order` and each declares a `Save` overload
- **THEN** the index holds one `overload_set` node for `Save` under `Order`

#### Scenario: One declaration has no set

- **WHEN** a class declares one method `close`
- **THEN** the class contains `close` directly, and no `overload_set` node exists for it

### Requirement: Every declaration records its signature as data

Each function and method node SHALL record its parameters as data: each parameter's name, type as written, and whether it is optional, has a default or is variadic. It SHALL record its return type as written. Each call SHALL record its argument count, whether an argument is spread, each argument's type where the extractor knows it, and for Swift its labels.

#### Scenario: A Java method with a variadic parameter

- **WHEN** a Java class declares `format(String fmt, Object... args)`
- **THEN** `codegraph node format --json` lists two parameters, the second marked variadic, and the return type

### Requirement: A call resolves to the declaration the language selects (RES-01)

A call SHALL resolve among the candidates of its name under its owner. Arity SHALL filter first, with ranges for optional, default and variadic parameters and a lower bound under a spread argument. The argument types the extractor knows SHALL filter second. When one candidate remains, the call SHALL get an edge to it. The result SHALL NOT depend on the order in which files are indexed, and SHALL hold after `codegraph index` and after every `codegraph sync`.

#### Scenario: Resolution by arity

- **WHEN** a class declares `log(String)` and `log(String, Throwable)`, and a method calls `log("x", e)`
- **THEN** the call's edge targets `log(String, Throwable)` and `callers` on `log(String)` does not list it

#### Scenario: Resolution by argument type

- **WHEN** a class declares `add(int)` and `add(String)`, and a method with a parameter `String s` calls `add(s)`
- **THEN** the call's edge targets `add(String)`

#### Scenario: A variadic parameter

- **WHEN** a class declares `format(String, Object...)`, and a method calls `format("a", 1, 2, 3)`
- **THEN** the call's edge targets `format(String, Object...)`

### Requirement: A call that static analysis cannot decide targets the set

When several candidates remain after the arity and type filters, the call SHALL get an edge to the `overload_set` node, marked ambiguous. It SHALL NOT get an edge to any single overload. When no candidate remains, the call SHALL stay unresolved.

#### Scenario: An argument whose type is unknown

- **WHEN** a class declares `add(int)` and `add(String)`, and a method calls `add(null)`
- **THEN** the call's edge targets the `add` set, marked ambiguous, and `callers` on either overload lists it as ambiguous

#### Scenario: No overload fits

- **WHEN** a class declares `add(int)` and `add(String)`, and a method calls `add(1, 2)`
- **THEN** the call stays unresolved and no edge targets `add`

### Requirement: A call through a variable resolves through its declared type

When a call's receiver is a parameter, a local, a field or `this`, the indexer SHALL use the receiver's declared type. It SHALL resolve the call against the methods of that type. The indexer SHALL NOT read the receiver's capitalization to decide what it names. A receiver whose type is unknown SHALL stay unresolved.

#### Scenario: A call through a field

- **WHEN** a Java class declares a field `Logger logger` and a method calls `logger.log("x")`, and `Logger` is declared in the same package
- **THEN** the call's edge targets `Logger.log(String)`

#### Scenario: A call through a parameter

- **WHEN** a C# method has a parameter `ILogger log` and calls `log.Write(evt)` where `evt` is a parameter of type `LogEvent`
- **THEN** the call's edge targets `ILogger.Write(LogEvent)`

#### Scenario: A lowercase Go package

- **WHEN** a Go file imports `example.com/m/util` and calls `util.Parse(s)`
- **THEN** the call's edge targets `Parse` in package `util`

#### Scenario: An unknown receiver

- **WHEN** a Java method calls `x.run()` and `x` is the result of a call the indexer cannot resolve
- **THEN** the call stays unresolved and no edge targets any `run`

### Requirement: A local without a declared type takes the type of its initializer

A local with no declared type SHALL take the type of its initializer. The initializer SHALL be a literal, a `new`, a cast or a call with one possible target, or the type SHALL stay unknown.

#### Scenario: A local initialized with new

- **WHEN** a Java method declares `var b = new Builder()` and calls `b.add(s)` with a `String s`, and `Builder` is declared in the same package with `add(String)` and `add(int)`
- **THEN** the call's edge targets `Builder.add(String)`

#### Scenario: A local initialized from a call into another file

- **WHEN** `Factory.make()` in one file returns `Builder`, and a method in another file declares `var b = Factory.make()` and calls `b.add(s)` with a `String s`
- **THEN** the call's edge targets `Builder.add(String)`

#### Scenario: A local initialized from an unresolved call

- **WHEN** a Java method declares `var r = lookup()` where `lookup` does not resolve, and calls `r.run()`
- **THEN** the call stays unresolved

### Requirement: A same-file reference resolves by lexical scope

A same-file call SHALL resolve to the innermost declaration of its name whose scope encloses the call, before a module-level declaration, before the cross-file index. Only declarations that the language exports or declares at module level SHALL be reachable from another file.

#### Scenario: A function declared in a callback

- **WHEN** `a.ts` exports a module-level `function e()`, a callback in `a.ts` declares another `function e()` and calls `e()`, and `b.ts` imports `e` and calls it
- **THEN** the call in the callback targets the nested `e`, and the call in `b.ts` targets the exported `e`. The nested `e` has no caller from `b.ts`

#### Scenario: A nested declaration is not visible across files

- **WHEN** a TypeScript file declares `function helper()` only inside a callback, and `b.ts` calls `helper()` without an import
- **THEN** the call in `b.ts` stays unresolved

### Requirement: Repeats that are not overloads follow their language

In Python, Ruby and JavaScript, a call to a name that a module defines twice at module level SHALL resolve to the last definition. Declarations under different conditional-compilation blocks SHALL be alternatives, and a call to them SHALL target their set as ambiguous. A Swift function's qualified name SHALL include its argument labels.

#### Scenario: A Python redefinition

- **WHEN** a Python module defines `def load()` twice at module level, and a function calls `load()`
- **THEN** the call's edge targets the second `load`

#### Scenario: Alternatives under `#if`

- **WHEN** a C# class declares `Process(string)` under `#if FEATURE_SPAN` and another `Process(string)` under `#else`, and a method calls `Process(s)`
- **THEN** the call's edge targets the `Process` set, marked ambiguous

#### Scenario: Swift labels

- **WHEN** a Swift file declares `func move(to point: Point)` and `func move(by delta: Vector)`
- **THEN** the index holds `move(to:)` and `move(by:)` as two functions with different qualified names

### Requirement: An incremental sync gives the same graph as a fresh index (INDX-03, INDX-04)

A user can create, edit, rename and delete files in any sequence and run `codegraph sync` after each step. After the last sync, the store SHALL hold the same nodes and edges as `codegraph index --force` over the final tree, except route nodes and their edges. Route detection runs only in a full index, a limitation that `docs/LANGUAGE-CAPABILITY-MATRIX.md` documents.

#### Scenario: A sequence of edits

- **WHEN** a user adds a file, renames a function in it, deletes a second file, and runs `codegraph sync` after each step
- **THEN** the store holds the same nodes and edges as a fresh index of the final tree

#### Scenario: A new overload changes a resolution

- **WHEN** a class declares `add(Object)` and another file calls `add(s)` with a `String s`, and then a user adds `add(String)` and runs `codegraph sync`
- **THEN** the store holds the same nodes and edges as a fresh index, so the call targets `add(String)`

#### Scenario: A repeated declaration is removed

- **WHEN** a file declares `function e()` twice, is indexed, and the first copy is then deleted and `codegraph sync` runs
- **THEN** the store holds the same nodes and edges as a fresh index of the final file

### Requirement: Each edge connects two stored nodes (INDX-04)

Each edge in the store SHALL have a source and a target that are nodes in the store, after `codegraph index` and after every `codegraph sync`.

#### Scenario: A file that other files call is deleted

- **WHEN** file `b.go` calls a function in `a.go`, and `a.go` is deleted and `codegraph sync` runs
- **THEN** no stored edge names a node from `a.go`

### Requirement: The reported node count equals the stored nodes

The `nodes=` count that `codegraph index` and `codegraph sync` print SHALL equal the number of nodes in the store, which `codegraph status` reports as `Nodes:`.

#### Scenario: A file with repeated declarations is synced many times

- **WHEN** a file that declares `function e()` three times is edited and synced ten times
- **THEN** the `nodes=` count of the last sync equals the `Nodes:` count of `codegraph status`

### Requirement: An index from before unique declaration ids is rebuilt once

A store written by a version without unique declaration ids SHALL be rebuilt from scratch by its next `codegraph sync`. Later syncs SHALL be incremental. The rebuild SHALL also correct the stored node count.

#### Scenario: The first sync after an upgrade

- **WHEN** a store written before this change has a drifted node count, and the new version runs `codegraph sync`
- **THEN** the sync rebuilds the whole graph, and its `nodes=` count equals the `Nodes:` count of `codegraph status`

#### Scenario: Records that a fresh index does not have

- **WHEN** a store written before this change still holds the node of a deleted file, and the new version runs `codegraph sync`
- **THEN** the store holds the same nodes and edges as a fresh index of the tree

#### Scenario: The syncs after the rebuild

- **WHEN** the new version runs `codegraph sync` a second and a third time, each with one file changed
- **THEN** each of those syncs reparses only that file
