## ADDED Requirements

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
