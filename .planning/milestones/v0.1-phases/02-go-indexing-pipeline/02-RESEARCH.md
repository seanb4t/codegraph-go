# Phase 2: Go Indexing Pipeline - Research

**Researched:** 2026-07-10
**Domain:** Go static analysis (tree-sitter AST extraction) + two-pass concurrent indexing pipeline + Cobra CLI lifecycle
**Confidence:** HIGH (grammar/API facts verified this session) / MEDIUM (Go-specific extraction & resolution design patterns, which are this project's own engineering, not an off-the-shelf library)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**CLI Lifecycle & Commands (INDX-01, INDX-02)**
- **D-01:** Build the CLI on `spf13/cobra` with root command `codegraph` and subcommands `init`, `index`, `uninit`. `init` = create `.codegraph/` **and** run a full from-scratch index in one step. `index` = deterministic from-scratch rebuild against an already-initialized `.codegraph/`. `uninit` = remove `.codegraph/` cleanly.
- **D-01a:** `init` on an existing `.codegraph/` **errors with guidance** rather than silently clobbering. `uninit` requires confirmation unless `--force`. `index --force` rebuilds without prompting; `--quiet` suppresses progress; `--verbose` emits per-file/per-pass detail; default prints a concise end-of-run summary (files, nodes, edges, duration). A from-scratch `index` MUST be **deterministic** — same input tree ⇒ byte-identical graph.
- **D-01b:** The Pebble store lives in a subdirectory of `.codegraph/` (name is executor's discretion). The store-wide `Meta` record carries schema version + counts + health. No separate on-disk config/JSON file in v1. Whether `init` writes a `.gitignore` hint is executor's discretion.

**Node Identity (RES-01 substrate)**
- **D-02:** Node ids follow **`<kind>:<hash>`** (TS-parity shape), where `<hash>` is a stable content hash over identity-defining fields (kind + qualified_name + file_path minimum; exact tuple executor's discretion, MUST be deterministic/reproducible).
- **D-02a:** Hash MUST be **collision-resistant (SHA-256 family, hex-truncated)** — never MD5. The `<kind>:<hex>` shape is identical to TS; only the algorithm is strengthened.

**Schema Field Parity**
- **D-03:** Additively extend `graph.proto`: `Node` gets `signature`, `docstring`, `visibility`, `is_exported`, `return_type`; `Edge` gets `provenance`, `metadata`. Defer TS fields that don't apply to Go yet. New field numbers stay below `50..59`. `SchemaVersion` stays `1`.
- **D-03a:** Phase 2 writes **only ground-truth AST edges** — every edge has empty/`ast` provenance. `provenance: heuristic` and synthesized dispatch are Phase 5. Do not emit any synthesized edge in Phase 2.

**Two-Pass Indexing Architecture (RES-01)**
- **D-04:** Pass 1 — parallel extract: bounded worker pool (default `runtime.NumCPU()`, tunable), one `Parser` per worker (tree-sitter parsers NOT goroutine-safe), emits per-file (nodes, intra-file edges, unresolved cross-file refs with name+kind+call-site line/col). Read-only, embarrassingly parallel. Pass 2 — sequential resolve: build global symbol index (qualified-name/import-path → node id), resolve refs into calls/imports/type-reference edges, write through `GraphStore`. Resolve pass owns the single coordinated writer.
- **D-04a:** Write batching uses `GraphStore.Writer` (IndexedBatch) in batched windows — never one engine write per symbol. Unresolved references held in memory for v1 (no on-disk `unresolved_refs` table). Batch granularity is executor's discretion so long as no per-symbol commits occur.

**Edge Multiplicity**
- **D-05:** Keep `e/<src>/<kind>/<dst>` unchanged: multiple call sites sharing `(source, kind, target)` collapse to one stored edge, carrying the first/representative call site's line/col. Documented divergence from TS's `idx_edges_identity(source,target,kind,line,col)` — intentional, matches `keys.go`'s stated design.

**Go Extraction Scope (LANG-01)**
- **D-06:** Node kinds: `file`, `function`, `method`, `type` (struct), `interface`, `constant`, `variable` (`field` optional). Edge kinds: `contains` (file→symbol, type→method), `imports` (file→imported package), `calls` (function/method→callee), and type-reference/embedding edges (struct embedding, interface embedding) — the concrete AST-visible form of "type inheritance."
- **D-06a:** Interface→implementation dispatch is **NOT synthesized** in Phase 2 (Phase 5, RES-02/RES-03). Phase-2 "type inheritance" = concrete embedding + type references only.

### Claude's Discretion
- The exact tree-sitter node-type → codegraph node-kind mapping and the tree-walk/query design (design against `testdata/golden/`).
- The precise hash input tuple and truncation length for node ids (must be deterministic + collision-resistant per D-02a).
- Worker-pool sizing knobs, resolve-pass batch-commit granularity, and the in-memory intermediate's data structures.
- `.codegraph/` internal subdirectory naming and whether `init` writes a `.gitignore` hint.
- Whether `field` nodes and doc-comment (`docstring`) extraction are full or minimal in Phase 2.

### Deferred Ideas (OUT OF SCOPE)
- Query commands, `explore`, MCP server / golden-output diffing — **Phase 3**.
- Incremental `sync`, native file watchers, rename/delete pruning, daemon — **Phase 4**. (Content-hashed node ids and the `f/`-namespace range-delete hook are the seams sync binds to; no sync logic built here.)
- Interface→implementation dispatch synthesis, `provenance: heuristic` tagging, framework-aware routing — **Phase 5**.
- Additional languages — **Phase 5**. Extractor should be shaped so a second language is a new extractor behind the same two-pass engine, not a rewrite.
- On-disk `unresolved_refs` staging, `name_segment_vocab` search index — not required for Phase 2.
- 100k-file monorepo scale / peak-RSS bounding — **Phase 8**.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| INDX-01 | `codegraph init` creates `.codegraph/` + builds full graph in one step; `uninit [--force]` removes cleanly | Architecture Patterns §CLI Lifecycle; Cobra command tree in Code Examples |
| INDX-02 | `codegraph index` deterministic from-scratch rebuild with `--force`/`--quiet`/`--verbose` | Common Pitfalls §Determinism; Validation Architecture §determinism test |
| RES-01 | Cross-file resolution: imports, call edges, type inheritance via two-pass parallel-extract → sequential-resolve | Architecture Patterns §Two-Pass Pipeline, §Symbol Index & Resolution |
| LANG-01 | Go: full structural extraction + cross-file resolution (first language, validates pipeline) | Architecture Patterns §Tree-sitter Node Mapping; Code Examples |
</phase_requirements>

## Summary

Phase 2 has almost no new external-library research surface — Phase 1 already selected and battle-tested every load-bearing dependency (Pebble via `GraphStore`, CGo tree-sitter via `parser.Parser`, protobuf schema). The genuine engineering risk in this phase is **project-specific design**, not library selection: (1) the exact tree-sitter-go node-type → codegraph vocabulary mapping, (2) building a correct two-pass Go symbol/import resolver without a full type-checker, and (3) making a concurrent, worker-pool-based indexer produce a **byte-identical** result on every from-scratch run — the last of which has real, non-obvious failure modes (map iteration, edge-dedup "representative" selection, goroutine completion order) that must be designed around explicitly, not bolted on after the fact.

The one new dependency this phase introduces is `spf13/cobra` (verified current: v1.10.2, 44k+ GitHub stars, not archived) for the CLI. Everything else — `golang.org/x/mod` (module-path parsing) and `golang.org/x/sync` (bounded worker pool via `errgroup`) — is **already a pinned indirect dependency in `go.mod`** from Phase 1 and only needs promoting to direct; `golang.org/x/tools/go/packages` is already used by the Phase 1 archtest. This phase should therefore add net-one new module.

**Primary recommendation:** Build the extractor as a single `internal/index` package with three internal seams — `discover` (file enumeration via stdlib `go/build.Context.MatchFile`, no `go list` subprocess), `extract` (per-file tree-sitter walk producing an in-memory intermediate, run inside an `errgroup.Group` with `SetLimit(runtime.NumCPU())`), and `resolve` (single-threaded symbol-index build + `GraphStore.Writer` batched commits) — and treat determinism as a first-class constraint from the first line of code: sort every slice that participates in ordering, and define "representative call site" as a value computed by a deterministic tie-break rule, never by "whichever goroutine wrote it first."

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| CLI argument parsing / lifecycle (`init`/`index`/`uninit`) | CLI (Cobra command layer) | — | User-facing entry point; owns flag parsing and process exit codes, not extraction logic |
| `.codegraph/` directory create/remove | CLI / filesystem | Storage (Pebble open/close) | Directory lifecycle is a thin filesystem operation; the store itself is opened/closed by `graphstore.Open` |
| File discovery + build-tag filtering | Indexer (discover seam) | — | Pure filesystem + stdlib `go/build` logic; no dependency on storage or parser |
| Per-file AST extraction | Indexer (extract seam) | Parser (tree-sitter) | Extract seam owns tree-walk logic; Parser is the narrow syntax-tree-producing dependency it calls into |
| Cross-file symbol resolution | Indexer (resolve seam) | — | Owns the single writer; must run single-threaded per D-04 |
| Node/Edge persistence | Storage (`GraphStore`) | — | Indexer never touches Pebble directly — only through the interface (D-04a boundary, enforced by archtest) |
| Determinism guarantee | Indexer (resolve seam) | Storage (`Export`) | Resolve seam controls processing order; `Export`'s key-sorted iteration is what a determinism test diffs against |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/spf13/cobra` | v1.10.2 (verified current via Go module proxy 2026-07-10; `.claude/CLAUDE.md` cites v1.9.x — bump to current) | CLI framework: `init`/`index`/`uninit` command tree | De facto Go CLI standard (kubectl, docker, gh, hugo); built-in flag/subcommand ergonomics map directly onto D-01's command surface |
| `github.com/tree-sitter/go-tree-sitter` + `tree-sitter-go` | already pinned (Phase 1, `parser.Parser`/`cgo.NewGoParser()`) | AST parsing | Ratified in `PARSER-DECISION.md` — CGo Option A |
| `github.com/cockroachdb/pebble/v2` | already pinned (Phase 1, `GraphStore`) | Storage | Ratified in Phase 1 D-01 |
| `google.golang.org/protobuf` | already pinned (Phase 1, `internal/schema`) | Record encoding | Ratified in Phase 1 D-02 |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `golang.org/x/sync/errgroup` | v0.22.0 (already indirect in `go.mod`; **promote to direct**) | Bounded worker pool for Pass 1 extraction | `errgroup.Group.SetLimit(n)` + `Go(func() error)` is the standard idiomatic bounded-parallelism primitive — avoids hand-rolled `sync.WaitGroup` + semaphore-channel plumbing |
| `golang.org/x/mod/modfile` | v0.38.0 (already indirect in `go.mod`; **promote to direct**) | Parse `go.mod`'s `module` directive to compute the repo's base import path | Needed to map a file's directory to its full Go import path (`<module-path>/<relative-dir>`), which the resolve pass needs for cross-package `imports`/`calls` resolution. Reuse the stdlib-adjacent parser rather than regex-scanning `go.mod` |
| `golang.org/x/tools/go/packages` | v0.48.0 (already direct, used by `archtest`) | NOT recommended for file discovery (see Don't Hand-Roll / Pitfalls) — kept only for its existing archtest use | Shells out to `go list`; heavier and more fragile than `go/build.Context.MatchFile` for this phase's discovery need |
| `go/build` (stdlib) | Go 1.26.5 (pinned toolchain) | Per-file build-tag/GOOS/GOARCH matching during file discovery | `Context.MatchFile(dir, name)` is the exact primitive `go list`/`go/packages` themselves use internally — zero new dependency, no subprocess, works on a repo with compile errors |
| `crypto/sha256` (stdlib) | Go 1.26.5 | Node id hashing (D-02a) | Collision-resistant per D-02a; never `crypto/md5` |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `go/build.Context.MatchFile` for discovery | `golang.org/x/tools/go/packages` (`Load` with `NeedFiles│NeedCompiledGoFiles│Tests:true`) | `go/packages` gives you full package-graph resolution for free (accurate import-path computation per package) but shells out to `go list`, requires a fully resolvable module (fails ungracefully on multi-module workspaces or broken code), and is slower on large repos. Prefer `go/packages` only if resolve-pass import-path computation via manual `go.mod`+directory math (recommended below) proves insufficient in practice |
| `spf13/cobra` | `urfave/cli` | Rejected in `.claude/CLAUDE.md` — Cobra fits CodeGraph's nested command surface; urfave/cli is for flatter CLIs |
| Hand-rolled worker pool (channels + `sync.WaitGroup`) | `errgroup.Group` | `errgroup` gives you first-error propagation and context cancellation for free; a hand-rolled pool would have to reimplement both to be equally safe for a Pass-1 extractor where one file's parse panic/error shouldn't corrupt the run silently |

**Installation:**
```bash
go get github.com/spf13/cobra@v1.10.2
go get -u golang.org/x/sync golang.org/x/mod   # promote existing indirect deps to direct
# do NOT run `go mod tidy` blind — Phase 1 SUMMARY notes this strips deliberately
# pre-pinned-but-unimported deps; promote explicitly as Phase 2 code imports them.
```

**Version verification:** Verified 2026-07-10 via `curl https://proxy.golang.org/<module>/@latest` and `@v/list` (Go's own module proxy, authoritative source of truth for what's actually published) plus a GitHub repo metadata check for `cobra` (44,244 stars, `archived: false`).

## Package Legitimacy Audit

| Package | Registry | Age | Downloads/Popularity | Source Repo | Verdict | Disposition |
|---------|----------|-----|----------------------|--------------|---------|-------------|
| `github.com/spf13/cobra` | Go module proxy | First tag 2013, latest v1.10.2 (2025-12-03) | 44,244 GitHub stars; used by kubectl/docker/gh/hugo | github.com/spf13/cobra | OK | Approved |
| `golang.org/x/sync` | Go module proxy | Official `golang.org/x` extended-stdlib module | N/A — official Go team module | golang.org/x/sync | OK | Approved (already pinned, promote to direct) |
| `golang.org/x/mod` | Go module proxy | Official `golang.org/x` extended-stdlib module | N/A — official Go team module | golang.org/x/mod | OK | Approved (already pinned, promote to direct) |

**Packages removed due to [SLOP] verdict:** none.
**Packages flagged as suspicious [SUS]:** none.

*Note on tooling:* the `gsd-tools query package-legitimacy check` seam supports `npm`/`pypi`/`crates` ecosystems only; it does not have a Go-registry backend. Verification above was performed manually against the **Go module proxy** (`proxy.golang.org`, the authoritative Go registry) and GitHub repo metadata — this is the ecosystem-appropriate substitute per the package-legitimacy protocol's Step 2 ("run the appropriate command for the phase's primary language"), applied here since no Go-native command exists in the shared seam. All three packages are `[VERIFIED: proxy.golang.org]`, not `[ASSUMED]` — they were independently confirmed to exist, be current, and (for cobra) be a real, actively-maintained, widely-adopted project via direct registry/GitHub queries in this session, not training-data recall alone.

## Architecture Patterns

### System Architecture Diagram

```
                    ┌─────────────────────────────────────────┐
                    │  cobra CLI  (codegraph init|index|uninit)│
                    └───────────────────┬───────────────────────┘
                                        │
                    init: mkdir .codegraph/ + run pipeline
                    index: run pipeline against existing .codegraph/
                    uninit: rm .codegraph/ (confirm unless --force)
                                        │
                                        ▼
                    ┌─────────────────────────────────────────┐
                    │            Indexer Pipeline               │
                    │                                           │
                    │  ┌─────────────┐                          │
                    │  │  discover   │  walk repo, filter by     │
                    │  │  (serial)   │  go/build.MatchFile,      │
                    │  │             │  skip vendor/.git/.codegraph│
                    │  └──────┬──────┘  → sorted []string paths  │
                    │         │                                  │
                    │         ▼                                  │
                    │  ┌─────────────────────────────┐           │
                    │  │  Pass 1 — extract (parallel) │           │
                    │  │  errgroup.SetLimit(NumCPU)   │           │
                    │  │  worker N: own Parser        │           │
                    │  │   file bytes → Parse() →     │           │
                    │  │   tree-walk → per-file        │           │
                    │  │   {nodes, intra-file edges,   │           │
                    │  │    unresolved refs}           │           │
                    │  │   results[i] (index-addressed,│           │
                    │  │   not channel-drained)         │           │
                    │  └──────────────┬────────────────┘           │
                    │                 │  (all files done)          │
                    │                 ▼                            │
                    │  ┌─────────────────────────────┐             │
                    │  │  Pass 2 — resolve (serial)   │             │
                    │  │   1. build global symbol      │            │
                    │  │      index (pkg import path   │            │
                    │  │      + qualified name → id)   │            │
                    │  │   2. resolve unresolved refs   │            │
                    │  │      → calls/imports/embeds    │            │
                    │  │   3. dedupe edges (D-05),       │           │
                    │  │      pick deterministic          │          │
                    │  │      representative line/col     │         │
                    │  │   4. GraphStore.NewWriter()      │         │
                    │  │      batched PutNode/PutEdge/    │         │
                    │  │      PutFile/PutMeta → Commit()  │         │
                    │  └──────────────┬───────────────────┘         │
                    └─────────────────┼─────────────────────────────┘
                                      ▼
                    ┌─────────────────────────────────────────┐
                    │   GraphStore (Pebble, Phase 1 substrate)  │
                    │   n/<id>  e/<src>/<kind>/<dst>  f/<path>  │
                    └─────────────────────────────────────────┘
```

A reader can trace: CLI invocation → file discovery → parallel per-file extraction → single-writer resolution/dedup → committed graph, all through the Phase 1 `GraphStore`/`Parser` interfaces without any new package reaching into Pebble or tree-sitter directly.

### Recommended Project Structure
```
internal/
├── indexer/
│   ├── discover.go       # file walk + go/build.MatchFile filtering + module-path lookup
│   ├── discover_test.go
│   ├── extract.go        # Pass 1: worker pool, per-file tree-walk dispatch
│   ├── extract_test.go
│   ├── goextract/        # Go-specific tree-sitter node-kind mapping (extractor logic)
│   │   ├── goextract.go
│   │   └── goextract_test.go
│   ├── resolve.go        # Pass 2: symbol index + reference resolution + edge dedup
│   ├── resolve_test.go
│   ├── nodeid.go         # D-02 hash construction (kind+qname+path → sha256 → <kind>:<hex>)
│   ├── nodeid_test.go
│   └── pipeline.go        # discover → extract → resolve orchestration, Meta stamping
├── cli/
│   ├── root.go            # cobra root command
│   ├── init.go            # `codegraph init`
│   ├── index.go           # `codegraph index`
│   └── uninit.go          # `codegraph uninit`
cmd/
└── codegraph/
    └── main.go
```

### Pattern 1: File Discovery via `go/build.Context.MatchFile` (not `go/packages`)
**What:** Walk the repo with `filepath.WalkDir`, skip `vendor/`, any dot-prefixed directory (`.git`, `.codegraph`, etc.), then for every `*.go` candidate call `build.Default.MatchFile(dir, filename)` to decide whether it belongs to the default build context (GOOS/GOARCH, build tags). This is the exact primitive the `go` toolchain itself uses to decide file inclusion — it does not require a resolvable module graph and does not shell out to `go list`.
**When to use:** Always, for Phase 2 file discovery. `go/packages` remains reserved for the existing archtest use (import-graph enforcement), not file discovery.
**Example:**
```go
// Source: stdlib go/build docs (pkg.go.dev/go/build), verified this session
import "go/build"

func includeFile(dir, name string) (bool, error) {
    ctx := build.Default
    match, err := ctx.MatchFile(dir, name)
    if err != nil {
        return false, err
    }
    return match, nil
}
```
`MatchFile` correctly evaluates both old-style `// +build` and modern `//go:build` constraints and GOOS/GOARCH filename suffixes (`_linux.go`, `_amd64.go`, etc.) — do not hand-roll this parsing (see Don't Hand-Roll).

### Pattern 2: Bounded Worker Pool for Pass 1 (one Parser per worker)
**What:** `errgroup.Group.SetLimit(runtime.NumCPU())`, dispatching one goroutine per discovered file but capping concurrency; each worker owns exactly one `parser.Parser` instance for its lifetime (tree-sitter parsers are not goroutine-safe per Phase 1's `parser.go` contract) — do NOT share one `Parser` across goroutines, and do NOT create a new `Parser` per file (`NewGoParser()` allocates a CGo-backed parser; creating thousands of them for a large repo needlessly churns C allocations).
**When to use:** Pass 1 extraction only. Pass 2 is explicitly single-threaded (D-04).
**Example:**
```go
// Source: golang.org/x/sync/errgroup docs, verified this session
g := new(errgroup.Group)
g.SetLimit(runtime.NumCPU())

results := make([]fileResult, len(files)) // index-addressed, NOT channel-drained
for i, f := range files {
    i, f := i, f
    g.Go(func() error {
        p, err := cgo.NewGoParser()
        if err != nil {
            return err
        }
        defer p.Close()
        r, err := extractFile(p, f) // opens file, Parse(), tree-walk, Tree.Close()
        if err != nil {
            return err
        }
        results[i] = r // safe: each goroutine owns a disjoint index
        return nil
    })
}
if err := g.Wait(); err != nil {
    return err
}
```
Writing to `results[i]` by pre-assigned index — rather than draining a channel in completion order — is what makes Pass 2's input order **independent of goroutine scheduling**, the first line of defense for determinism (see Common Pitfalls).

### Pattern 3: Node ID Construction (D-02/D-02a)
**What:** `id = "<kind>:" + hex(sha256(preimage))[:32]`, where `preimage` is built with **length-prefixed segments** (reusing the `appendSegment` discipline already established in `internal/graphstore/keys.go`, not naive string concatenation) over `(kind, qualified_name, file_path)`. Truncating SHA-256's 64 hex chars to 32 preserves the same *visual* id length as the TS corpus's MD5-shaped ids (`class:1aa9ad9ada394f639ed0f8104462aef5` is 32 hex chars) while satisfying D-02a's "SHA-256 family, never MD5" requirement — 128 bits of a cryptographic hash's output remains collision-resistant for this use case (not a security boundary, just an identity key).
**When to use:** Every `Node` created in Pass 1 or resolved in Pass 2.
**Example:**
```go
// Pattern grounded in keys.go's existing appendSegment (delimiter-injection guard)
// — reused here for the hash preimage, not just the storage key, for the same
// tamper/collision-avoidance reason: naive "kind+qname+path" concatenation could
// let a crafted qualified_name absorb into file_path or vice versa.
func nodeID(kind, qualifiedName, filePath string) string {
    var buf []byte
    buf = appendSegment(buf, kind)
    buf = appendSegment(buf, qualifiedName)
    buf = appendSegment(buf, filePath)
    sum := sha256.Sum256(buf)
    return kind + ":" + hex.EncodeToString(sum[:])[:32]
}
```

### Pattern 4: Qualified Name Construction (Go-specific)
**What:** Per the captured golden fixtures (`ts-schema.dump.sql`), TS CodeGraph's `qualified_name` for Go top-level symbols is simply the **bare declared name** (`"main"`, `"mergeStyle"`, `"EpicKey"`) — NOT package-import-path-prefixed. Node-id uniqueness comes from the `(kind, qualified_name, file_path)` tuple, not from qualified_name alone. Recommend matching this for functions/types/interfaces/constants/variables. For **methods**, recommend `"<ReceiverTypeName>.<MethodName>"` (e.g. `"Runner.Exec"`) — this is the natural Go idiom and is *not* directly confirmed in the captured fixtures (flagged in Assumptions Log).
**When to use:** Building the `qualified_name` field on every extracted `Node`, and as the resolve-pass symbol-index key alongside import path.
**Example:** see Code Examples §Method Receiver Extraction below.

### Pattern 5: Cross-Package Import Resolution
**What:** Build the repo's base module import path once (via `golang.org/x/mod/modfile.Parse` on `go.mod`), then for each file compute `importPath = modulePath + "/" + relDir` (relDir = the file's directory relative to the module root, `""` for the root package). The resolve pass builds a `map[importPath]map[declaredName]nodeID` (the "global symbol index" from D-04) by grouping all Pass-1 results by their computed import path. A `selector_expression` call `pkg.Fn()` resolves by: (a) look up `pkg`'s import alias in the calling file's `import_spec` list to get the real import path, (b) look up that import path + `Fn` in the global symbol index. An unqualified `identifier` call resolves by looking up the *calling file's own* import path + name in the same index (Go allows any file in a package to call any other file's package-level symbol).
**When to use:** Pass 2 resolve, for both `calls` and `imports` edges.
**Example:** see Code Examples §Import Resolution below.

### Anti-Patterns to Avoid
- **Guessing method-call receiver types beyond direct/lexical evidence:** resolving `x.Method()` when `x`'s type requires interface satisfaction, embedded-field promotion across multiple hops, or type inference through a return value is exactly the D-06a-excluded heuristic-dispatch problem — leave it as an unresolved ref (never silently dropped, never guessed) rather than reaching for Phase 5's synthesis work early. Only resolve receiver-method calls when the receiver's declared type is directly visible in the same function's parameter list, receiver clause, or an in-scope `var x T` / `:=` short-decl whose RHS is a literal type construction (`T{}`, `&T{}`, `new(T)`).
- **Creating a new `parser.Parser` per file:** each `NewGoParser()` call allocates CGo state; for a repo with thousands of files this is needless C-allocation churn. Pool one parser per worker goroutine, not per file.
- **Committing per-symbol writes:** violates D-04a explicitly; batch per-file or per-N-files into one `IndexedBatch`, `Commit()` once at the end of Pass 2 (or in bounded windows for very large repos — a Phase 8 concern, not required here).
- **Relying on `go/packages` + `go list` as the default file-discovery path:** shells out to a subprocess, requires full module resolvability, and is measurably slower — reserve it for its existing archtest role only.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Build-tag / GOOS-GOARCH file filtering | A custom `//go:build` / `// +build` constraint parser | `go/build.Context.MatchFile` (stdlib) | This is exactly what the real Go toolchain uses; hand-rolling risks silently mis-indexing platform-specific files (a real correctness bug, not cosmetic) |
| `go.mod` module-path parsing | Regex over `go.mod` text | `golang.org/x/mod/modfile.Parse` (already pinned) | `go.mod` syntax has edge cases (replace directives, multi-line requires, toolchain directives) a regex will eventually mis-parse |
| Bounded concurrent worker pool with error propagation | Hand-rolled channel + `sync.WaitGroup` + manual semaphore | `golang.org/x/sync/errgroup` (`SetLimit`) | `errgroup` gives first-error-wins + context cancellation for free; a hand-rolled version needs to reimplement both correctly under concurrent access |
| Cryptographic content hashing | A custom/weak hash (FNV, CRC32, MD5) for node ids or file content hashes | `crypto/sha256` (stdlib) | D-02a and the `graph.proto` `File.content_hash` doc both mandate SHA-256-family; MD5/CRC are explicitly disallowed for collision resistance reasons |
| AST tree-walking primitives (cursor/traversal) | A hand-rolled recursive descent over `*tree_sitter.Node` | tree-sitter's own `TreeCursor`/`Node.Child(i)` API (already exposed through the Phase 1 `Tree.Inner()` unwrap) | Tree-sitter's cursor API is the correct, tested way to walk a CST; reimplementing traversal risks missing named-vs-anonymous-node distinctions the grammar already encodes |

**Key insight:** almost nothing in this phase should be a "new algorithm" — the discovery, hashing, and concurrency primitives are all either stdlib or already-vetted dependencies. The actual engineering effort belongs entirely in the **Go-specific extraction/resolution mapping** (tree-sitter node types → codegraph vocabulary) and in **disciplined determinism** (sort everything that participates in output order).

## Common Pitfalls

### Pitfall 1: Edge-Dedup "Representative" Call Site Is Order-Dependent By Default
**What goes wrong:** D-05 says duplicate `(source, kind, target)` call sites collapse to one edge, carrying the "first/representative" call site's line/col. If "first" means "whichever goroutine's result got processed first" or "whichever file happened to be visited first by a map iteration," the chosen line/col — and therefore the exported graph's bytes — **varies between runs**, silently breaking D-01a's determinism requirement and Phase 3's golden-diff / Phase 8's reproducibility gate.
**Why it happens:** Go map iteration order is randomized by design; goroutine completion order in a worker pool is inherently nondeterministic; if Pass 2 aggregates call-site candidates into a map keyed by `(src,kind,dst)` and just keeps "whatever we saw last/first while ranging the map," the result is nondeterministic.
**How to avoid:** Define "representative" as a value computed from a **total order**, not processing order: aggregate every candidate call site for a given `(src,kind,dst)` triple into a slice, sort it by `(filePath, line, col)`, and take the first element. Since Pass 1 results are stored index-addressed by a pre-sorted file list (Pattern 2), and each file's own call sites are naturally emitted in source order during the tree-walk, this sort is cheap and the result is reproducible regardless of worker scheduling.
**Warning signs:** A determinism test (see Validation Architecture) that intermittently fails, or fails only under `-race`/higher `GOMAXPROCS`, is the classic symptom.

### Pitfall 2: Pebble's On-Disk Bytes Are Not the Right Determinism Target
**What goes wrong:** Comparing two `.codegraph/` directories' raw `.sst`/`.log` files byte-for-byte across two from-scratch index runs can fail even when the *logical* graph content is identical — Pebble's LSM internals (SSTable segmentation, compaction timing, WAL rotation) are not guaranteed to produce byte-identical files for identical logical writes.
**Why it happens:** LSM-tree storage engines separate "what data is stored" from "how it's physically laid out on disk," and the latter is influenced by write timing/batching that isn't part of this phase's determinism contract.
**How to avoid:** Verify determinism via `GraphStore.Export()` (already implemented in Phase 1, iterates in Pebble's key-sorted order — deterministic for identical KV content) rather than raw file diffing. Strip the one genuinely volatile field, `Meta.last_sync_unix_ms`, before comparing — following the exact precedent `testdata/golden/README.md`'s volatile-field-stripping convention already established for the golden capture harness.
**Warning signs:** A "determinism" test that diffs `.codegraph/` directory trees directly (via `diff -r` or checksum-per-file) rather than calling `Export()`.

### Pitfall 3: CGo Parser/Tree Lifecycle Leaks Under Concurrency
**What goes wrong:** Forgetting to `Close()` a `*parser.Tree` after extraction, or forgetting to `Close()` a worker's `Parser` when its errgroup goroutine returns early on error, leaks C-allocated tree-sitter memory — invisible to the Go garbage collector and invisible to `go test -race`, but real and cumulative across a large repo's file count.
**Why it happens:** The `parser.Tree`/`parser.Parser` contract (Phase 1, `parser.go`) explicitly documents this is the caller's responsibility — CGo memory is not GC-managed.
**How to avoid:** `defer p.Close()` immediately after `NewGoParser()` succeeds inside each worker goroutine (not after the whole errgroup completes); `defer tree.Close()` immediately after a successful `Parse()` call, before doing anything else with the tree, so an extraction error partway through the walk still releases the tree.
**Warning signs:** RSS growing linearly with files-indexed-so-far during a large-repo run; this won't show up in unit tests over small fixtures — needs at least a medium-size real-repo run to notice (ties into the phase's own success-criterion-4 real-world-repo validation).

### Pitfall 4: The 4 MiB `MaxSourceBytes` Ceiling Applies Per-File, Silently
**What goes wrong:** A single generated/vendored/minified `.go` file over 4 MiB causes `Parse()` to return `parser.ErrSourceTooLarge` — if the extractor treats this as a fatal pipeline error rather than a per-file skip-with-warning, one oversized file (a generated protobuf file, a vendored dependency accidentally not excluded, etc.) aborts the entire index run.
**Why it happens:** `parser.MaxSourceBytes` (Phase 1, Security Domain V5 mitigation) is a hard, non-configurable-per-call ceiling by design.
**How to avoid:** Treat `ErrSourceTooLarge` (and any other single-file extraction error) as a per-file, recorded, non-fatal condition — record it on the `File` record's future `errors` field (TS parity: `files.errors` is a JSON array in `ts-schema.sql`; Phase 2's `File` message doesn't yet have this field per D-03's additive list — flagged as an Open Question below) and continue indexing the rest of the repo. `--verbose` should surface these; the default summary should at least count them.
**Warning signs:** Indexing a real-world repo (success criterion 4) that has even one large generated file fails outright instead of producing a graph with one documented gap.

### Pitfall 5: `go/build.Context.MatchFile` Needs `dir` to Be the File's Actual Parent Directory
**What goes wrong:** Calling `MatchFile` with the repo root as `dir` for every file (instead of each file's own containing directory) silently mis-evaluates build tags, because `MatchFile`'s second argument is resolved relative to `dir`.
**Why it happens:** Easy copy-paste mistake when the discovery walk already has an absolute path and it's tempting to pass the same `dir` for every call in a loop.
**How to avoid:** Always pass `filepath.Dir(fullPath)` and `filepath.Base(fullPath)` for the specific file being tested, not a hoisted/cached directory value.

## Code Examples

### Method Receiver Extraction (tree-sitter-go grammar, verified via Context7 this session)
```
// Source: github.com/tree-sitter/tree-sitter-go test corpus (declarations.txt),
// fetched via Context7 this session — official grammar repo, HIGH confidence
package main

func (self Person) Equals(other Person) bool {}
func (v *Value) ObjxMap(optionalDefault ...(Map)) Map {}
```
The `method_declaration` node's `receiver` field is a `parameter_list` containing exactly one `parameter_declaration`; that declaration's `type` field is either a bare `type_identifier` (value receiver) or a `pointer_type` wrapping a `type_identifier` (pointer receiver). Extract the receiver type name by checking for `pointer_type` first and unwrapping one level; use that identifier's text as `<ReceiverTypeName>` for the method's `contains` edge target (type→method) and qualified name.

### Struct Embedding Detection (tree-sitter-go grammar, verified via Context7 this session)
```
// Source: github.com/tree-sitter/tree-sitter-go test corpus (types.txt)
type s2 struct { Person }
```
```
(type_declaration
  (type_spec
    (type_identifier)          ; "s2"
    (struct_type
      (field_declaration_list
        (field_declaration
          (type_identifier))))))  ; "Person" — a field_declaration with a type
                                  ; child but NO name child is an embedded field
```
A `field_declaration` inside a `struct_type` is an **embedded field** iff it has a `type_identifier` (or `qualified_type` for a cross-package embed, e.g. `io.Reader`) child but no separate `name`/`field_identifier` child before it. Emit an embedding edge from the struct's type node to the embedded type node (or leave unresolved if the embedded type is external/stdlib and not in this repo's graph).

### Interface Embedding Detection (tree-sitter-go grammar, verified via Context7 this session)
```
// Source: github.com/tree-sitter/tree-sitter-go test corpus (types.txt)
type i2 interface {
  i1
  io.Reader
  SomeMethod(s string) error
}
```
```
(interface_type
  (type_list
    (qualified_type (package_identifier) (type_identifier)))  ; io.Reader — embedded
  (method_declaration_list
    (method_declaration (method_identifier) (parameter_list ...) (return_type ...))))
```
Interface embedding is a **sibling child of the interface_type's own `type_list`** (distinct from `method_declaration_list`, which holds the interface's own method signatures) — walk both children separately: `type_list` entries become embedding edges, `method_declaration_list` entries describe the interface's method set (Phase 2 need not emit separate nodes for interface method *signatures* — they have no independent body/location worth a node; D-06's node-kind list does not include one).

### Import Resolution
```go
// Executor-designed pattern (not sourced from an external doc) — grounded in
// D-04's "global symbol index (import-path/qualified-name -> node id)"
type symbolIndex struct {
    // keyed by (importPath, declaredName)
    byImportAndName map[string]map[string]string // importPath -> name -> nodeID
}

func (idx *symbolIndex) resolveSelector(callerImportPath string, importsInFile map[string]string, pkgAlias, name string) (nodeID string, ok bool) {
    targetImportPath, isLocal := importsInFile[pkgAlias]
    if !isLocal {
        return "", false // external/stdlib import — no node to resolve to
    }
    names, ok := idx.byImportAndName[targetImportPath]
    if !ok {
        return "", false
    }
    id, ok := names[name]
    return id, ok
}

func (idx *symbolIndex) resolveUnqualified(callerImportPath, name string) (nodeID string, ok bool) {
    names, ok := idx.byImportAndName[callerImportPath]
    if !ok {
        return "", false
    }
    id, ok := names[name]
    return id, ok
}
```

### Bounded Worker Pool (already shown in Pattern 2, repeated here as the canonical reference for the planner)
```go
// Source: golang.org/x/sync/errgroup docs, fetched via WebFetch this session
g := new(errgroup.Group)
g.SetLimit(runtime.NumCPU())
for i, f := range sortedFiles {
    i, f := i, f
    g.Go(func() error { results[i] = extractOne(f); return nil })
}
if err := g.Wait(); err != nil { return err }
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| `smacker/go-tree-sitter` community fork | `tree-sitter/go-tree-sitter` official org module | Aug 2024 (per `.claude/CLAUDE.md`) | Already reflected in Phase 1's dependency choice; nothing new for Phase 2 |
| `go/build.Import` (older, deprecated-in-spirit stdlib API for whole-package loading) | `go/build.Context.MatchFile` for per-file matching, `golang.org/x/tools/go/packages` for full package loading | Ongoing Go-tooling convention since ~2018 | This phase should use `MatchFile` (lightweight, no subprocess) rather than the heavier full-package-load APIs, since it only needs per-file build-tag matching, not type information |

**Deprecated/outdated:** none directly relevant beyond the above — this phase's dependency surface is small and already current per the verification above.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Method qualified_name should be `"<ReceiverTypeName>.<MethodName>"` | Architecture Patterns §Pattern 4 | Low — this only affects the qualified_name field's exact string shape, which feeds the node-id hash's preimage; as long as it's deterministic and unique per (kind, file_path), the graph is still internally correct. Golden fixtures didn't show a method's qualifiedName field directly, so this is a design choice, not a confirmed parity fact |
| A2 | The `File` message needs an `errors` field (JSON-array-like, TS parity) to record per-file extraction failures (oversized file, parse error) without aborting the whole index | Common Pitfalls §Pitfall 4 | Medium — if this field isn't added, per-file failures either (a) silently vanish (bad — hides real problems) or (b) abort the whole run (bad — fragile against one bad file in a large real repo, directly threatening success criterion 4). This is additive to D-03's explicit field list, which the planner should confirm/add |
| A3 | Whether to model `imports` edges via a lightweight synthetic "package" pseudo-node (for intra-module imports) or skip edge emission for imports that don't map to another indexed file | Architecture Patterns §Pattern 5; Open Questions #1 | Medium — affects whether Phase 3's query layer can meaningfully traverse "what does this file import" as a graph edge to a real node, vs. having no target to show. D-06 mentions the edge kind but not a package node kind, so this needs an explicit planner decision |
| A4 | `.codegraph/` should be added to `.gitignore` automatically on `init` is left as executor's discretion per D-01b; recommend doing so (matching typical tool ergonomics) unless the user's repo already ignores it | Architecture Patterns §CLI Lifecycle | Low — purely a UX nicety, not a correctness concern |

## Open Questions

1. **How should `imports` edges target packages that have no node of their own?**
   - What we know: D-06 specifies an `imports` edge kind (file→imported package), but D-06's node-kind list (`file`, `function`, `method`, `type`, `interface`, `constant`, `variable`) has no `package` kind. TS's own schema models each import statement as its own `import:<hash>` node (visible in `ts-schema.dump.sql`), which Phase 2 explicitly does not adopt.
   - What's unclear: whether Phase 2 should (a) introduce a minimal synthetic `package` node kind for intra-module imports only (external/stdlib imports get no edge), (b) target the file node of one "representative" file in the imported package, or (c) skip `imports` edges Phase 2 doesn't have a clean target for and revisit in Phase 5/later.
   - Recommendation: (a) — a minimal `package` pseudo-node (id computed the same way, `package:<hash of import path>`, fields limited to name+qualified_name=import path) for **intra-module packages only**. External/stdlib imports are recorded on the file's extraction intermediate for completeness/debugging but do not produce a graph edge in Phase 2, since there is no node to target and D-03a forbids inventing edges beyond ground truth. Planner should treat this as a decision to ratify or override, not a locked fact.

2. **Should method-call resolution attempt any local variable type tracking, or only the most direct receiver forms?**
   - What we know: D-06a forbids interface-dispatch synthesis (Phase 5 territory). Direct cases (an enclosing method's own receiver calling another method on itself, or a call through a freshly-constructed `T{}`/`&T{}`/`new(T)`) are purely lexical/AST-local and arguably still "ground truth," not heuristic.
   - What's unclear: exactly where the line is between "AST-local, ground truth" and "requires type inference" for the purposes of D-06a's boundary. The captured golden fixtures don't show enough method-call examples to confirm TS's own boundary here.
   - Recommendation: implement the narrowest safe set first (receiver-self calls, calls through freshly-constructed literals in the same statement/nearby scope) and leave everything else as an unresolved ref; expand only if success-criterion-4's real-world-repo validation shows an unacceptable number of falsely-unresolved calls that a slightly wider lexical rule would fix without guessing.

3. **Does `field` node extraction matter for success criterion 4 on a real repo, or can it be skipped entirely in v1?**
   - What we know: D-06 marks `field` nodes as optional/executor's discretion.
   - What's unclear: whether skipping struct field nodes materially changes what Phase 3's queries can show (e.g., "what uses this field") in a way a user would notice on first use.
   - Recommendation: skip `field` nodes in the initial Phase 2 plan (smaller surface, faster to ship and verify); flag as an easy additive follow-up since it's a pure superset addition to the vocabulary, not a breaking change.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | All of Phase 2 | ✓ | go1.26.5 (per `go.mod`) | — |
| C toolchain (CGo) | `internal/parser/cgo` (already required by Phase 1) | ✓ (confirmed in `PARSER-DECISION.md`: Apple clang 21) | — | — |
| `git` | Discovery does not require it (no `.gitignore` respect implemented — see below), but developer workflow does | ✓ (repo is a git repo) | — | — |
| Go module proxy network access | Verifying package versions during this research session | ✓ (proxy.golang.org reachable) | — | — |

**Missing dependencies with no fallback:** none identified for Phase 2 itself.

**Note on `.gitignore` handling:** CONTEXT.md does not mandate honoring `.gitignore` during file discovery, and no such requirement appears in INDX-01/INDX-02/RES-01/LANG-01. Recommend **not** adding a `.gitignore`-parsing dependency in Phase 2 — skip `vendor/` and dot-directories explicitly (a fixed, small denylist), which covers the overwhelmingly common case for a Go repo, and defer full `.gitignore` semantics to a later phase if a real-world repo (success criterion 4) demonstrates a concrete need. This keeps the "Don't Hand-Roll" principle from over-firing into "don't add a dependency we don't yet need."

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go's built-in `testing` package (`go test`) — same as Phase 1 |
| Config file | none — table-driven tests per package, matching Phase 1's established pattern |
| Quick run command | `go test ./internal/indexer/... -run TestXxx -v` (per-package targeted run, <30s) |
| Full suite command | `go test ./... -count=1` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| LANG-01 | Each tree-sitter node type maps to the correct codegraph node kind (function, method, type, interface, constant, variable, embedding) | unit | `go test ./internal/indexer/goextract/... -run TestExtract -v` | ❌ Wave 0 |
| LANG-01 | Method receiver (value and pointer) resolves to the correct qualified name and `type→method` contains edge | unit | `go test ./internal/indexer/goextract/... -run TestMethodReceiver -v` | ❌ Wave 0 |
| RES-01 | Cross-package `pkg.Fn()` call resolves to the correct node id via import-alias → import-path → symbol-index lookup | integration | `go test ./internal/indexer/... -run TestResolveCrossPackageCalls -v` | ❌ Wave 0 |
| RES-01 | Struct/interface embedding produces the type-reference/embedding edge across files in the same repo | integration | `go test ./internal/indexer/... -run TestResolveEmbedding -v` | ❌ Wave 0 |
| INDX-01/INDX-02 | `init`/`index`/`uninit` flag behavior (`--force`, `--quiet`, `--verbose`, error-on-reinit) | integration (CLI) | `go test ./internal/cli/... -run TestInitIndexUninit -v` | ❌ Wave 0 |
| INDX-02 | From-scratch `index` run twice against the same fixture repo produces an identical `GraphStore.Export()` byte stream (after stripping `Meta.last_sync_unix_ms`) | integration (determinism) | `go test ./internal/indexer/... -run TestDeterministicRebuild -v` | ❌ Wave 0 |
| RES-01/LANG-01 (success criterion 4) | Indexing a real multi-package Go fixture repo produces symbols/edges matching an expected-structure fixture | integration (fixture-diff) | `go test ./internal/indexer/... -run TestRealRepoStructure -v` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** targeted `go test ./internal/indexer/... -run <TestName>`
- **Per wave merge:** `go test ./... -count=1`
- **Phase gate:** full suite green before `/gsd-verify-work`, plus a manual real-repo index run against a mid-size Go repo (e.g. the already-available `weft` corpus used for Phase 1's golden capture) confirming success criterion 4 by inspection.

### Wave 0 Gaps
- [ ] `internal/indexer/goextract/goextract_test.go` — covers LANG-01 node/edge kind mapping (table-driven, one case per tree-sitter node type in Architecture Patterns' mapping)
- [ ] `internal/indexer/resolve_test.go` — covers RES-01 cross-file resolution (multi-package fixture, small enough to commit under `testdata/`)
- [ ] `internal/indexer/determinism_test.go` — covers the D-01a byte-identical rebuild guarantee via double-run + `Export()` diff
- [ ] `internal/cli/*_test.go` — covers INDX-01/INDX-02 flag semantics (likely via `cobra`'s own command-execution test helpers or a subprocess harness)
- [ ] A small, committable multi-package Go fixture repo under `testdata/` distinct from the Phase 1 golden corpus (that corpus is read-only ground truth for TS parity diffing, not meant to be mutated/re-indexed repeatedly by Phase 2's own tests) — needed before extraction/resolution tests can run

*(No existing test infrastructure covers Phase 2's behaviors — this is a new package with all-new tests, consistent with the "greenfield" status Phase 1 left the repo in for indexer code specifically.)*

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-------------------|
| V2 Authentication | No | Phase 2 is a local CLI tool with no auth surface |
| V3 Session Management | No | N/A |
| V4 Access Control | No | N/A |
| V5 Input Validation | Yes | `parser.MaxSourceBytes` ceiling (already enforced by Phase 1's `Parser.Parse`) bounds pathological-input DoS surface; the extractor must not construct unbounded-size intermediate structures from a single crafted file (e.g. pathologically deep nesting producing an unbounded-depth recursive tree-walk — mitigate with an explicit walk-depth guard or iterative-not-recursive traversal if depth becomes a concern) |
| V6 Cryptography | Yes | `crypto/sha256` (stdlib) for node-id hashing per D-02a — never a weak/non-cryptographic hash |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|----------------------|
| Crafted node id / file path / qualified name containing key-delimiter bytes forging a Pebble key into another namespace's range | Tampering | Length-prefixed segment encoding (`appendSegment`, already established in `keys.go`) applied consistently to the node-id hash preimage as well (Pattern 3 above) — not just the storage key |
| Pathologically deep/large single source file causing unbounded recursion or memory blowup during tree-walk | Denial of Service | `parser.MaxSourceBytes` (already enforced, Phase 1) bounds input size before parsing; the extractor's own tree-walk should prefer an explicit stack/iterative walk over unbounded Go-stack recursion if profiling on the spike's "deep nesting" corpus (`PARSER-DECISION.md`) shows a concern — tree-sitter's own parse is not naively recursive per the spike's crash-isolation finding, but the *extractor's* walk code is new and should not assume that protection extends to itself |
| A single oversized/unparseable file aborting the entire index run (denial of index availability for the whole repo) | Denial of Service (self-inflicted) | Per-file error containment (Common Pitfalls #4) — one bad file must not prevent indexing the rest of a real-world repo |

## Sources

### Primary (HIGH confidence)
- `tree-sitter/go-tree-sitter` (Context7, fetched this session) — grammar node types: `function_declaration`, `method_declaration` (receiver field shape), `type_declaration`/`struct_type`/`interface_type`, struct/interface embedding shapes, `import_declaration`/`import_spec` variants, `call_expression`/`selector_expression` AST shape, `const_declaration`/`var_declaration`, generic type-parameter shapes
- `proxy.golang.org` (Go module proxy, queried directly this session) — verified current versions: `spf13/cobra@v1.10.2`, `golang.org/x/sync@v0.22.0`, `golang.org/x/mod@v0.38.0`, `golang.org/x/tools@v0.48.0` (all match or update `go.mod`/`.claude/CLAUDE.md`)
- `api.github.com/repos/spf13/cobra` (queried directly this session) — 44,244 stars, not archived
- `pkg.go.dev/golang.org/x/sync/errgroup` (WebFetch, fetched this session) — `Group.SetLimit`, `Go`, `Wait` semantics
- `pkg.go.dev/golang.org/x/tools/go/packages` (WebFetch, fetched this session) — `LoadMode` constants, `Config.Tests` behavior, `CompiledGoFiles` build-tag-filtering behavior, vendor-directory handling
- This repo's own Phase 1 artifacts (`internal/graphstore/store.go`, `keys.go`, `internal/parser/parser.go`, `internal/parser/cgo/parser_cgo.go`, `internal/schema/graph.proto`, `internal/schema/meta.go`, `internal/graphstore/export.go`, `internal/graphstore/archtest/import_graph_test.go`) — read directly this session, ground truth for the interfaces this phase binds to
- `testdata/golden/ts-schema.sql`, `testdata/golden/ts-schema.dump.sql`, `testdata/golden/corpus/weft-go/*.json`, `testdata/golden/README.md` — read directly this session, ground truth for TS parity vocabulary and known TS-side historical bugs (edge-dedup issue #1034)

### Secondary (MEDIUM confidence)
- Go-specific two-pass resolution design (symbol index, import-path computation, method-receiver-based qualified names) — this project's own engineering, informed by Go language semantics (training knowledge) and cross-checked against the grammar facts above and the captured golden fixtures, but not itself sourced from an external document because no such off-the-shelf design exists for this exact storage/schema combination

### Tertiary (LOW confidence)
- none — no unverified WebSearch-only claims were used as load-bearing facts in this document; the one design area with genuine residual ambiguity (import/package-node modeling, method-call resolution boundary) is surfaced explicitly in Open Questions rather than asserted as fact

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — every dependency either already ratified in Phase 1 or verified live against the Go module proxy/GitHub this session
- Architecture (tree-sitter node mapping): HIGH — grammar facts fetched directly from the official grammar repo via Context7 this session
- Architecture (two-pass resolution design): MEDIUM — this is original engineering for this project's specific schema/storage combination, not a documented external pattern; grounded in verified Go language semantics and the project's own Phase 1 interfaces, but the exact import/package-node modeling has open questions flagged above
- Pitfalls (determinism, CGo lifecycle): HIGH — grounded directly in this repo's own Phase 1 contracts (`parser.go`'s documented Close() discipline, `export.go`'s key-sorted iteration) and well-established Go concurrency semantics (map iteration randomization, goroutine scheduling nondeterminism)

**Research date:** 2026-07-10
**Valid until:** 30 days (stable stdlib/well-established-dependency surface; re-verify `cobra` version if planning is delayed past that window)
