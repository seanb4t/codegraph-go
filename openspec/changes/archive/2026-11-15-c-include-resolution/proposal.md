# C include resolution

Issue: #259

## Why

In C and C++, a call to a function in another file never resolves. Each file is its own module, and `#include` never links files, so a bare call looks only in its own file. A prototype in a header and its definition in a `.c` file are two nodes that never meet. In the repro from #259, `codegraph callers add` prints 0 callers and "2 definitions named "add": add.c:2, add.h:1", although `main.c` includes `add.h` and calls `add(1, 2)`. Most C and C++ code calls across files this way, so `callers`, `callees` and `impact` miss most of a C project's call graph.

## What Changes

- A quoted `#include "path"` links the including file to the included file. The path resolves relative to the directory of the including file, then relative to the repository root. The index stores an `imports` edge between the two file nodes.
- A bare call that finds no candidate in its own file looks in the files that its file includes, transitively, nearest first.
- A call can reach a prototype with external linkage. If exactly one definition in another indexed file has the same name and signature, the call resolves to that definition. With no such definition, the call targets the prototype. With several, the call also targets the prototype, because the definitions are alternatives (for example, one per platform).
- The extractor records which function nodes are prototypes and which functions are `static`. A `static` function has `module` visibility, as a TypeScript symbol that its file does not export, and is not exported. A `static` definition never matches a prototype from another file.
- The prototype rule from #250 does not change: a prototype whose definition is in the same file gets no node. Alternatives under `#if` keep their rule.
- A header, an including file or a definition file can change, appear or go away. After each such step, `codegraph sync` gives the same graph as a fresh index.
- The first sync of an index built before this change rebuilds it once, because its function nodes do not record prototypes.
- `docs/LANGUAGE-CAPABILITY-MATRIX.md` and its descriptor in `internal/indexer/capability/matrix.go` drop the gap "#include never populates Imports" for C and C++, and record the new rules and their limits.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `graph-index`: adds requirements for C and C++ include links and for calls through included files. It also adds requirements for a prototype that resolves to its definition in another file, for sync equivalence, and for the one-time rebuild.

## Impact

- Code: the C and C++ extractor (`internal/indexer/mainstream/cextract`), the include edge mark (`internal/indexer/goextract`), the resolver (`internal/indexer/resolve`) and the symbol view (`internal/indexer/symbols`). A new package `internal/bfs` holds the shared level walk, which `symbols` and `internal/query` both use. Also sync (`internal/indexer/sync.go`, `storeview.go`) and the schema (`internal/schema/graph.proto`), which gets the prototype marker, a parameter match key and a `Meta` flag.
- Output: C and C++ graphs gain `imports` edges between files and `calls` edges across files. A `static` function now reports `module` visibility and is not exported. A C call no longer targets a `struct` tag of the same name. The prototype marker and the match key are stored, but no output shows them. A same-file prototype that differs from its definition only in a comment or in the names inside a function-pointer parameter now gets no node, by the rule of #250. These are additive changes to a compatibility surface.
- Corpora: no pinned corpus is C or C++, so `corpora/observations.json` is not expected to change. `task corpora:measure` confirms it.
- Store: a new `Meta` flag, `has_c_prototypes`, makes the first sync of an existing index rebuild it once. `schema_version` does not change.

## Out of scope

- An angle-bracket include (`#include <...>`), and include directories from build files such as `CMakeLists.txt` or `compile_commands.json`. These includes link nothing.
- A call without an include that declares the name (an implicit declaration).
- Macros that define or rename functions, and calls through function pointers. A top-level qualifier of a parameter stays in the match key, so `int f(int);` does not match `int f(const int x)`, although C treats them as one function.
- A C++ header with the `.h` extension. The index reads every `.h` file as C, so in such a header a call never targets a `struct` of the same name.
- C++ namespace-qualified names across files. Namespace bodies stay flattened, so a prototype can match a definition with the same name in another namespace. The capability matrix keeps this gap.
- Types, fields and variables across headers: a struct or a global declared in a header. Only calls change.
