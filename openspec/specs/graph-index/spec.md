# graph-index Specification

## Purpose
How the index represents the source code: which nodes a declaration gets, how node ids stay stable, and how a reference resolves to a declaration. Also how an incremental sync and the reported counts agree with the store.

## Requirements

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

When a file declares two or more symbols with the same kind and qualified name, the indexer SHALL store one node for each. Names and qualified names SHALL NOT change. A C# `partial` type SHALL keep one node for all of its parts, and a reopened Ruby class or module SHALL keep one node. In C and C++, a prototype whose definition is in the same file declares that definition, so it SHALL get no node of its own.

#### Scenario: A C prototype with its definition in the file

- **WHEN** a C file declares `int f(int a);`, defines `int f(int a)` below it, and `g` calls `f(1)`
- **THEN** `f` has one node, and the call in `g` targets it

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
- **THEN** the stored node of `format` lists two parameters, the second marked variadic, and the return type `String`

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

### Requirement: A full index of the same tree gives the same graph (INDX-02)

Run `codegraph index --force` twice over the same tree at the same commit. The two stores SHALL hold the same records, byte for byte: nodes, edges, file records and excluded-file records. The metadata record SHALL also be the same, except its last-sync time. The `--workers` value and the number of CPUs SHALL not change the result. The order in which the indexer reads and resolves files SHALL not change it either.

#### Scenario: Two fresh indexes

- **WHEN** a user runs `codegraph index --force` twice over one unchanged tree, into two empty stores
- **THEN** the two stores hold the same records, and their metadata differs only in the last-sync time

#### Scenario: A different worker count

- **WHEN** a user indexes one tree with `codegraph index --force --workers 1`, and then with `--workers 8`
- **THEN** the two stores hold the same records, and their metadata differs only in the last-sync time

#### Scenario: A different claim order

- **WHEN** one tree is indexed twice with 4 workers, once with the files taken in path order and once in reverse path order
- **THEN** the two stores hold the same records, and their metadata differs only in the last-sync time

#### Scenario: Duplicate edges at one position

- **WHEN** two references in one file resolve to the same source, kind and target at the same line and column
- **THEN** the store holds one edge for them, and two fresh indexes keep the same one

### Requirement: One node merges the copies of an id that several files emit
Two or more files can emit a node with one id, for example the parts of a C# partial type. The store SHALL then hold one node for that id. Its visibility SHALL be the first one in path order that differs from the language default. If no copy differs from the default, its visibility SHALL be the default. If any copy is exported, the node SHALL be exported. Its other fields SHALL come from the first copy in path order, and within one file from the first copy that the file emits.

#### Scenario: A public part and a part with no modifier
- **WHEN** `A.cs` declares `partial class Order` and `B.cs` declares `public partial class Order` in one namespace
- **THEN** the store holds one `Order` node with visibility `public`, and it is exported

#### Scenario: Every part has the default visibility
- **WHEN** `A.cs` and `B.cs` each declare `partial class Order` with no access modifier
- **THEN** the store holds one `Order` node with visibility `internal`

#### Scenario: A route that one file registers twice
- **WHEN** one Express file registers `GET /x` at line 3 and again at line 7
- **THEN** the store holds one route node for `GET /x`, and its start line is 3

#### Scenario: A sync of one part
- **WHEN** the two files of the public part scenario are indexed, `B.cs` drops `public`, and `codegraph sync` runs
- **THEN** the store holds the same records as `codegraph index --force` over the changed tree, and the `Order` node has visibility `internal`

#### Scenario: A sync adds a part to an indexed type
- **WHEN** `A.cs` declares `public partial class Order`, the tree is indexed, `B.cs` is added with `partial class Order` and no modifier, and `codegraph sync` runs
- **THEN** the store holds the same records as `codegraph index --force` over the changed tree, and the `Order` node has visibility `public`

#### Scenario: A sync after one part leaves the type
- **WHEN** `A.cs` declares `partial class Order : IA` and `B.cs` declares `public partial class Order : IB`, the tree is indexed, `B.cs` changes its declaration to `public class Invoice : IB`, and `codegraph sync` runs
- **THEN** the store holds the same records as `codegraph index --force` over the changed tree
- **AND** the `Order` node has visibility `internal`, and no edge from `Order` to `IB` remains

#### Scenario: A sync deletes the only part
- **WHEN** `A.cs` declares `partial class Order : IA` as the only part, the tree is indexed, `A.cs` is deleted, and `codegraph sync` runs
- **THEN** the store holds the same records as `codegraph index --force` over the changed tree, and no `Order` node and no edge from `Order` remains

#### Scenario: A sync deletes the only part of a type that another file names
- **WHEN** `A.cs` declares `partial class Order` as the only part, `C.cs` declares `class Sub : Order`, the tree is indexed, `A.cs` is deleted, and `codegraph sync` runs
- **THEN** the store holds the same records as `codegraph index --force` over the changed tree, and no edge from `Sub` to `Order` remains

#### Scenario: A sync after one part is deleted
- **WHEN** `A.cs` declares `partial class Order : IA` and `B.cs` declares `partial class Order : IB`, the tree is indexed, `B.cs` is deleted, and `codegraph sync` runs
- **THEN** the store holds the same records as `codegraph index --force` over the changed tree, and no edge from `Order` to `IB` remains

### Requirement: A quoted C include links two files

In C and C++, a `#include "path"` SHALL link the including file to the included file. The indexer SHALL look for `path` in the includer's directory, then from the repository root. A match SHALL give one `imports` edge between the two file nodes, and a file SHALL NOT link to itself. An angle-bracket include, a path that starts with `/`, a path that leaves the root and a path with no match SHALL link nothing. A path with an inner `.`, `..` or empty segment SHALL also link nothing.

#### Scenario: A header beside the source file

- **WHEN** `src/main.c` contains `#include "add.h"`, and `src/add.h` exists
- **THEN** the index holds an `imports` edge from the file node of `src/main.c` to the file node of `src/add.h`

#### Scenario: A path from the repository root

- **WHEN** `src/main.c` contains `#include "util/str.h"`, `src/util/str.h` does not exist, and `util/str.h` exists
- **THEN** the index holds an `imports` edge from `src/main.c` to `util/str.h`

#### Scenario: A sibling include directory

- **WHEN** `src/main.c` contains `#include "../include/add.h"` and `#include "../../x.h"`, and `include/add.h` exists
- **THEN** the index holds an `imports` edge from `src/main.c` to `include/add.h`, and none for `../../x.h`

#### Scenario: An absolute path

- **WHEN** `src/main.c` contains `#include "/usr/include/foo.h"`, and `src/usr/include/foo.h` exists
- **THEN** the index holds no `imports` edge from `src/main.c` for that directive

#### Scenario: A system header

- **WHEN** `src/main.c` contains `#include <stdio.h>`, and the repository has a file `stdio.h`
- **THEN** the index holds no `imports` edge from `src/main.c` for that directive

#### Scenario: An inner dot segment

- **WHEN** `src/main.c` holds `#include "util/../add.h"`, and `src/util/str.h` and `src/add.h` are indexed
- **THEN** `src/main.c` has no `imports` edge to `src/add.h`

#### Scenario: A header that includes itself

- **WHEN** `a.h` holds `#include "a.h"` and declares `void f(void);`, and `main.c` includes `a.h` and calls `f()` in `run`
- **THEN** `a.h` has no `imports` edge, and the call's edge targets `f` in `a.h`

### Requirement: A C call resolves through the files its file includes

In C and C++, a bare call that finds no candidate in its own file SHALL look in the files that its file includes. It SHALL then look deeper, one depth at a time. The files of one depth SHALL give one candidate set for the arity filter. The first depth that declares the name SHALL decide. In C, a `struct` tag SHALL NOT count as a declaration of the name for a call. An include cycle SHALL NOT repeat a file, and the result SHALL NOT depend on the order of indexing.

#### Scenario: A function defined in a header

- **WHEN** `util.h` defines `static inline int twice(int x)`, and `main.c` includes `util.h` and calls `twice(2)` in `run`
- **THEN** the call's edge targets `twice` in `util.h`, and `codegraph callers twice` lists `run`

#### Scenario: An include of an include

- **WHEN** `main.c` includes `api.h`, `api.h` includes `add.h`, `add.h` declares `int add(int a, int b);`, and `main.c` calls `add(1, 2)`
- **THEN** the call's edge targets `add` in `add.h`, the same target that a direct `#include "add.h"` in `main.c` gives

#### Scenario: A struct tag in a nearer header

- **WHEN** `main.c` holds `struct add { int x; };` and includes `a.h`, `a.h` holds only `struct add { int y; };` and includes `b.h`, `b.h` declares `int add(int a, int b);`, and `main.c` calls `add(1, 2)` in `run`
- **THEN** the call's edge targets `add` in `b.h`

#### Scenario: Include guards that form a cycle

- **WHEN** `a.h` includes `b.h`, `b.h` includes `a.h`, `main.c` includes `a.h`, and `b.h` declares `void f(void);` that `main.c` calls
- **THEN** indexing finishes, and the call's edge targets `f` in `b.h`

#### Scenario: The nearest header wins

- **WHEN** `main.c` includes `a.h` and `b.h`, `b.h` includes `c.h`, `a.h` declares `int f(int x);`, `c.h` declares `int f(int x);`, and `main.c` calls `f(1)`
- **THEN** the call's edge targets `f` in `a.h`

#### Scenario: Two headers at one depth

- **WHEN** `main.c` includes `a.h` and `b.h`, `a.h` declares `int f(int x);`, `b.h` declares `int f(int x, int y);`, and `main.c` calls `f(1, 2)`
- **THEN** the call's edge targets `f` in `b.h`

#### Scenario: A header at depth 1 and depth 2

- **WHEN** `main.c` includes `a.h` and `b.h`, `a.h` includes `b.h`, `a.h` declares `int f(int x);`, `b.h` declares `int f(int x, int y);`, and `main.c` calls `f(1, 2)`
- **THEN** the call's edge targets `f` in `b.h`

#### Scenario: The nearest header has the wrong arity

- **WHEN** `main.c` includes `a.h`, `a.h` declares `int f(int x);` and includes `b.h`, `b.h` declares `int f(int x, int y);`, and `main.c` calls `f(1, 2)`
- **THEN** the call's edge targets `f` in `a.h`, as a lone candidate in one file does today

#### Scenario: Every candidate at the first depth has the wrong arity

- **WHEN** `main.c` includes `a.h` and `b.h`, `a.h` and `b.h` each declare `int f(int x);`, `a.h` includes `c.h`, `c.h` declares `int f(int x, int y);`, and `main.c` calls `f(1, 2)`
- **THEN** the index holds no `calls` edge from the function in `main.c` that makes the call

#### Scenario: A name the file does not include

- **WHEN** `other.c` defines `int add(int a, int b)`, and `main.c` calls `add(1, 2)` but includes no file that declares `add`
- **THEN** the index holds no `calls` edge from the function in `main.c` that makes the call

### Requirement: A C static function is local to its file

In C and C++, a free function with internal linkage SHALL have `module` visibility and SHALL NOT be exported. A free function has internal linkage when its definition or an earlier declaration of it in its file says `static`. In C, any earlier declaration of the name counts. Any other free function, and any C++ member function, SHALL keep its visibility.

#### Scenario: A static helper

- **WHEN** `add.c` defines `static int clamp(int x)` and `int add(int a, int b)`
- **THEN** `clamp` has `module` visibility and is not exported, and `add` has `public` visibility and is exported

#### Scenario: A static forward declaration

- **WHEN** `util.c` declares `static int h(int);` and below it defines `int h(int x)` with no storage class
- **THEN** `h` has one node, with `module` visibility, and it is not exported

#### Scenario: A static declaration outside the definition's #if

- **WHEN** `util.c` declares `static int h(int);` at top level, and below it defines `int h(int x)` inside `#ifdef X`
- **THEN** the definition of `h` has `module` visibility, and it is not exported

#### Scenario: A C++ static member function

- **WHEN** `s.cpp` declares `class S { public: static int f() { return 0; } };`
- **THEN** `S.f` keeps `public` visibility and is exported

### Requirement: A C prototype resolves to its definition in another file

In C and C++, the extractor SHALL record whether a function node is a prototype. A call can reach a prototype that is not `static`. If exactly one non-`static` definition in another indexed C or C++ file has the same name and signature key, the call SHALL target that definition. With no such definition, or with several, the call SHALL target the prototype.

#### Scenario: The repro from #259

- **WHEN** `add.h` declares `int add(int a, int b);`, `add.c` includes `add.h` and defines `int add(int a, int b)`, and `main.c` includes `add.h` and calls `add(1, 2)` in `run`
- **THEN** the call's edge targets the definition in `add.c`, and `codegraph callers add` lists `run` as a caller of `add` in `add.c`

#### Scenario: A library prototype

- **WHEN** `vendor.h` declares `int vendor_init(void);`, no indexed file defines `vendor_init`, and `main.c` includes `vendor.h` and calls `vendor_init()`
- **THEN** the call's edge targets the prototype in `vendor.h`

#### Scenario: One definition for each platform

- **WHEN** `io.h` declares `int open_port(int n);`, both `io_posix.c` and `io_win.c` define `int open_port(int n)`, and `main.c` includes `io.h` and calls `open_port(1)`
- **THEN** the call's edge targets the prototype in `io.h`, and no edge from that call targets either definition

#### Scenario: A static definition of the same name

- **WHEN** `add.h` declares `int add(int a, int b);`, `add.c` defines only `static int add(int a, int b)`, and `main.c` includes `add.h` and calls `add(1, 2)`
- **THEN** the call's edge targets the prototype in `add.h`

#### Scenario: A prototype under an include guard

- **WHEN** `add.h` wraps `int add(int a, int b);` in `#ifndef ADD_H`, `#define ADD_H` and `#endif`, `add.c` defines `int add(int a, int b)` outside any `#if`, and `main.c` includes `add.h` and calls `add(1, 2)`
- **THEN** the call's edge targets the definition in `add.c`

#### Scenario: Pointer spacing and unnamed parameters

- **WHEN** `str.h` declares `int count(const char *);`, `str.c` defines `int count(const char* s)`, and `main.c` includes `str.h` and calls `count("x")`
- **THEN** the call's edge targets the definition in `str.c`

#### Scenario: Two definitions under #if in one file

- **WHEN** `io.h` declares `int open_port(int n);`, `io.c` defines `int open_port(int n)` under `#ifdef A` and again under `#else`, and `main.c` includes `io.h` and calls `open_port(1)`
- **THEN** the call's edge targets the prototype in `io.h`, because the count is of definitions and there are two

#### Scenario: A C++ default argument

- **WHEN** `f.hpp` declares `int f(int a, int b = 0);`, `f.cpp` defines `int f(int a, int b)`, and `main.cpp` includes `f.hpp` and calls `f(1)`
- **THEN** the call's edge targets the definition in `f.cpp`

#### Scenario: A static prototype

- **WHEN** `add.h` declares `static int add(int a, int b);`, `add.c` defines `int add(int a, int b)`, and `main.c` includes `add.h` and calls `add(1, 2)`
- **THEN** the call's edge targets the prototype in `add.h`

#### Scenario: A definition in the prototype's own file

- **WHEN** `add.h` declares `int add(int a, int b);` outside any `#if` and defines it inside `#ifdef X`, `add.c` also defines `int add(int a, int b)`, and `main.c` includes `add.h` and calls `add(1, 2)`
- **THEN** the call's edge targets the `overload_set` node named `add`, whose members are the prototype and the definition in `add.h`, as a pair under `#if` gets in one file today

#### Scenario: A C++ method of the same name

- **WHEN** `f.hpp` declares `int f(int x);`, `s.cpp` defines `int S::f(int x)` for `class S`, and `main.cpp` includes `f.hpp` and calls `f(1)`
- **THEN** the call's edge targets the prototype in `f.hpp`

#### Scenario: Another signature

- **WHEN** `add.h` declares `int add(int a, int b);`, `add.c` defines only `int add(long a, long b)`, and `main.c` includes `add.h` and calls `add(1, 2)`
- **THEN** the call's edge targets the prototype in `add.h`

### Requirement: A C prototype matches its definition by one rule

A prototype whose definition is in its own file SHALL get no node. The same-file match and the cross-file match SHALL compare the parameter types only. They SHALL ignore parameter names, the names inside a function-pointer parameter, comments, spacing and default arguments. A top-level qualifier of a parameter SHALL count. Alternatives under `#if` SHALL keep their rule.

#### Scenario: An array parameter with and without a name

- **WHEN** `arr.h` declares `int sum(int[10]);`, `arr.c` defines `int sum(int a[10])`, and `main.c` includes `arr.h` and calls `sum(v)` in `run`
- **THEN** the call's edge targets `sum` in `arr.c`

#### Scenario: A function-pointer parameter with and without names

- **WHEN** `cb.h` declares `void on(void (*)(int));`, `cb.c` defines `void on(void (*cb)(int x)) {}`, and `main.c` includes `cb.h` and calls `on(f)` in `run`
- **THEN** the call's edge targets `on` in `cb.c`

#### Scenario: Pointer spacing in one file

- **WHEN** `str.c` declares `int count(const char *);` and below it defines `int count(const char* s)`
- **THEN** `count` has one node in `str.c`

#### Scenario: A default argument in one file

- **WHEN** `s.cpp` declares `int f(int a, int b = 0);` and below it defines `int f(int a, int b)`
- **THEN** `f` has one node in `s.cpp`

### Requirement: A sync keeps C include resolution equal to a fresh index

A user can edit, add, rename or delete a C or C++ file, and run `codegraph sync` after each step. The file can be a header, an including file or a definition file. After each sync, the store SHALL hold the same nodes and edges as `codegraph index --force` over the same tree. In its scenarios, the repro from #259 is three files. `add.h` declares `int add(int a, int b);`. `add.c` includes `add.h` and defines `int add(int a, int b)`. `main.c` includes `add.h` and calls `add(1, 2)` in `run`.

#### Scenario: A definition appears after the call

- **WHEN** the repro from #259 is indexed without `add.c`, and then a user adds `add.c` with the definition and runs `codegraph sync`
- **THEN** the store holds the same nodes and edges as a fresh index, so the call targets the definition in `add.c`

#### Scenario: A header loses its prototype

- **WHEN** the repro from #259 is indexed, and then a user deletes the prototype from `add.h` and runs `codegraph sync`
- **THEN** the store holds the same nodes and edges as a fresh index, so the index holds no `calls` edge for the call in `main.c`

#### Scenario: An include is removed

- **WHEN** the repro from #259 is indexed, and then a user removes `#include "add.h"` from `main.c` and runs `codegraph sync`
- **THEN** the store holds the same nodes and edges as a fresh index, with no `imports` edge from `main.c` to `add.h`

#### Scenario: A header appears after its include

- **WHEN** `src/main.c` contains `#include "add.h"` and calls `add(1, 2)`, the repository is indexed without `src/add.h`, and then a user adds `src/add.h` with `int add(int a, int b);` and runs `codegraph sync`
- **THEN** the store holds the same nodes and edges as a fresh index, so the call targets the prototype in `src/add.h`

#### Scenario: An include cycle in a sync

- **WHEN** `a.h` includes `b.h`, `b.h` includes `a.h` and declares `void f(void);`, `main.c` includes `a.h` and calls `f()`, the repository is indexed, and then a user adds `void g(void);` to `b.h` and runs `codegraph sync`
- **THEN** the sync ends, and the store holds the same nodes and edges as a fresh index of the same tree

#### Scenario: A header is deleted

- **WHEN** the repro from #259 is indexed, and then a user deletes `add.h` and runs `codegraph sync`
- **THEN** the store holds the same nodes and edges as a fresh index, with no `imports` edge to `add.h` and no call edge from `run`

#### Scenario: A header moves to the repository root

- **WHEN** `src/main.c` includes `"add.h"`, `src/add.h` declares `int add(int a, int b);`, the repository is indexed, and then a user moves `src/add.h` to `add.h` and runs `codegraph sync`
- **THEN** the store holds the same nodes and edges as a fresh index, so the include of `src/main.c` links to `add.h`

#### Scenario: An indirect header is deleted

- **WHEN** `main.c` includes `api.h`, `api.h` includes `add.h`, `add.h` declares `int add(int a, int b);`, `add.c` defines it, `main.c` calls `add(1, 2)`, the repository is indexed, and then a user deletes `add.h` and runs `codegraph sync`
- **THEN** the store holds the same nodes and edges as a fresh index, so the index holds no `calls` edge for the call in `main.c`

#### Scenario: A header appears at the root

- **WHEN** `src/main.c` contains `#include "util/str.h"`, neither `src/util/str.h` nor `util/str.h` exists, the repository is indexed, and then a user adds `util/str.h` and runs `codegraph sync`
- **THEN** the store holds the same nodes and edges as a fresh index, so the include of `src/main.c` links to `util/str.h`

#### Scenario: A header's own include moves

- **WHEN** `main.c` includes `src/api.h`, `src/api.h` contains `#include "add.h"`, only `add.h` at the root exists and declares `int add(int a, int b);`, `main.c` calls `add(1, 2)`, the repository is indexed, and then a user adds `src/add.h` that declares `int add(int a);` and runs `codegraph sync`
- **THEN** the store holds the same nodes and edges as a fresh index, so the call in `main.c` targets `add` in `src/add.h`

#### Scenario: A nearer header appears

- **WHEN** `src/main.c` contains `#include "util/str.h"`, only `util/str.h` exists and is linked, and then a user adds `src/util/str.h` and runs `codegraph sync`
- **THEN** the store holds the same nodes and edges as a fresh index, so the include of `src/main.c` links to `src/util/str.h`

#### Scenario: A definition loses its body

- **WHEN** `p.h` declares `int add(int a, int b);`, `d.c` defines it, `main.c` includes `p.h` and calls `add(1, 2)`, the repository is indexed, and then a user removes the body of `add` in `d.c`, which leaves `int add(int a, int b);` at the same place, and runs `codegraph sync`
- **THEN** the store holds the same nodes and edges as a fresh index, so the call targets the prototype in `p.h`

#### Scenario: A prototype gains its body

- **WHEN** `p.h` declares `int add(int a, int b);`, `d.c` holds only `int add(int a, int b);`, `main.c` includes `p.h` and calls `add(1, 2)`, the repository is indexed, and then a user adds the body of `add` in `d.c` and runs `codegraph sync`
- **THEN** the store holds the same nodes and edges as a fresh index, so the call targets the definition in `d.c`

#### Scenario: A header with no symbols

- **WHEN** `main.c` includes `api.h`, `api.h` holds only `#include "add.h"`, `add.h` declares `int add(int a, int b);`, `main.c` calls `add(1, 2)`, the repository is indexed, and then a user adds the line `int y = 0;` to the body of `main` in `main.c` and runs `codegraph sync`
- **THEN** the store holds the same nodes and edges as a fresh index

#### Scenario: A definition loses static

- **WHEN** `add.h` declares `int add(int a, int b);`, `add.c` defines `static int add(int a, int b)`, `main.c` includes `add.h` and calls `add(1, 2)`, the repository is indexed, and then a user removes `static` from `add.c` and runs `codegraph sync`
- **THEN** the store holds the same nodes and edges as a fresh index, so the call targets the definition in `add.c`

#### Scenario: A second definition appears

- **WHEN** the repro from #259 is indexed, and then a user adds `add2.c` with another `int add(int a, int b)` and runs `codegraph sync`
- **THEN** the store holds the same nodes and edges as a fresh index, so the call targets the prototype in `add.h`

### Requirement: An index from before C prototypes are recorded is rebuilt once

A store written by a version that does not record C and C++ prototypes SHALL be rebuilt from scratch by its next `codegraph sync`. Later syncs SHALL be incremental.

#### Scenario: The first sync after an upgrade

- **WHEN** a store written before this change indexes the repro from #259, and the new version runs `codegraph sync`
- **THEN** the sync's `FilesReparsed` count, which `codegraph sync` prints after `reparsed=`, is the number of indexed files, and the store holds the same nodes and edges as a fresh index

#### Scenario: The syncs after the rebuild

- **WHEN** after that rebuild, a user changes the body of `void log_msg(void)` in `log.c`, which no other file calls, from `{}` to `{ int x = 1; }` and runs `codegraph sync`, then changes it to `{ int x = 2; }` and runs `codegraph sync` a third time
- **THEN** the `FilesReparsed` count of the second and the third sync, which `codegraph sync` prints after `reparsed=`, is 1
