# Pitfalls Research

**Domain:** Local-first code knowledge graph / code intelligence tool for AI coding agents (Go port of TypeScript CodeGraph)
**Researched:** 2026-07-10
**Confidence:** HIGH (SQLite concurrency, fsnotify/FSEvents limits, CGo cross-compilation, SLSA/reproducible-build mechanics — all sourced from official docs, maintainer threads, and multiple independent reproductions) / MEDIUM (wazero-vs-CGo tree-sitter benchmarks, incremental-graph correctness patterns, monorepo memory profiles — sourced from a handful of comparable projects' issue trackers and PRs, not yet reproduced independently)

## Critical Pitfalls

### Pitfall 1: CGo tree-sitter bindings silently break the "single static binary, all platforms" promise

**What goes wrong:**
Every mainstream Go tree-sitter binding (`smacker/go-tree-sitter`, `tree-sitter/go-tree-sitter`) is CGo-based. CGo is **disabled by default when cross-compiling** (opposite of native builds), so a `GOOS=windows GOARCH=amd64 go build` from a macOS/Linux CI runner silently produces a broken or non-CGo build unless you explicitly wire up a C cross-toolchain per target OS/arch. Teams that get this working with a cross C toolchain (mingw-w64, zig cc) then hit a second wall: the resulting binaries can crash at runtime due to linker flag mismatches between host and cross builds (confirmed in a still-open golang/go linker issue on `-target` vs `--target` clang flag handling), or ship without required runtime DLLs on Windows.

**Why it happens:**
The project's "single static binary per platform, no bundled runtime" constraint directly conflicts with CGo's dependency on a per-target C toolchain and dynamic libc/libstdc++ linkage on some platforms. This tension is invisible during local development (native builds work fine) and only surfaces in CI when cross-compiling for the first non-native target — often late in a release cycle.

**How to avoid:**
Decide the parser strategy (this is PROJECT.md's flagged open question) before writing any parser-integration code, using this decision rule: if CGo tree-sitter is chosen, budget for a full C cross-toolchain matrix in CI (Docker images per target, or zig cc) and validate boot-time behavior on real Windows/Linux/macOS hardware (or in CI-hosted VMs), not just "it links." If pure-Go static builds are the hard requirement (which the project's constraints imply), the two viable non-CGo paths are: (a) tree-sitter grammars compiled to WASM and run via wazero, or (b) a pure-Go tree-sitter reimplementation (e.g. the `gotreesitter`-class projects now emerging). Benchmark data from a comparable Go project (dvcdsys/code-index) found wazero-run real tree-sitter ~2x slower than native CGo but ~5x faster than a pure-Go reimplementation, with zero parse errors vs. some parse failures in the pure-Go path — making wazero the practical default unless the perf gap is proven to matter for this project's target repo sizes.

**Warning signs:** `CGO_ENABLED=0` needed anywhere in the build for other reasons (e.g. `netgo`/`osusergo` tags, minimal container images) conflicts with a CGo parser dependency; first Windows cross-build from a non-Windows CI runner fails or produces a binary that crashes at startup; release binary size/dependency graph includes libc variants per platform.

**Phase to address:** Architecture/parser-strategy phase, before any language extractor work begins — this is a foundational, expensive-to-reverse decision, not a per-language detail.

---

### Pitfall 2: SQLite "WAL mode" is mistaken for "concurrent writes," causing silent data loss or `database is locked` storms under multi-agent-session load

**What goes wrong:**
WAL mode enables concurrent *readers* alongside a single *writer* — it does not allow concurrent writers. Teams that enable `journal_mode=WAL` and assume the concurrency problem is solved then hit `SQLITE_BUSY`/"database is locked" errors under real multi-session load (multiple agent sessions or an MCP server + a background watcher writing simultaneously), or — worse — a transaction silently fails and the caller doesn't notice because Go's `database/sql` connection pool opens multiple physical connections that each need per-connection PRAGMA configuration and get out of sync. A documented real-world instance of this exact failure mode: TS CodeGraph itself shipped a bug (issue #773 on the source repo) where the daemon's incremental watch-sync intermittently hit `FOREIGN KEY constraint failed` because a single-file re-sync inserted edges referencing a node id deleted earlier in the same transaction; the edge insert silently failed, the daemon logged it but reported "healthy," and call-graph accuracy degraded invisibly over days until a full reindex.

**Why it happens:**
`database/sql`'s connection pool is not a single connection — `SetMaxOpenConns` defaults to unlimited, so each new pooled connection needs its own PRAGMA configuration (journal_mode, busy_timeout, foreign_keys are per-connection state, not per-database). Deferred (default) `BEGIN` transactions don't acquire the write lock until the first write statement, so a read-then-write transaction can get an immediate `SQLITE_BUSY` even with `busy_timeout` set, because SQLite determines waiting is pointless once another connection has already modified the DB.

**How to avoid:**
Split the connection pool into two `sql.DB` handles: a writer pool with `SetMaxOpenConns(1)` and `_txlock=immediate` (issues `BEGIN IMMEDIATE`, acquiring the write lock at transaction start instead of on first write), and a reader pool with a bounded number of connections (e.g. 4–16) with no txlock override. Set `busy_timeout >= 5000ms` via DSN pragma parameters (not a one-time `db.Exec`, which only configures whichever connection happens to be checked out) so it applies to every connection the pool opens. For incremental edge writes specifically, insert edges in a way that's immune to intra-transaction node-id ordering (batch-lookup which endpoint ids actually exist in the DB before inserting, rather than delete-then-insert-by-row-id) — this is exactly the fix TS CodeGraph shipped for issue #773. Fail loud on any FK violation or busy timeout rather than logging-and-continuing; surface index staleness in `status`/`sync` output instead of reporting "healthy" on a degraded graph.

**Warning signs:** "database is locked" errors under any concurrent load test with 2+ simultaneous writers (multiple MCP client sessions, or watcher + explicit CLI reindex); FK constraint failures anywhere in logs; `callers`/`callees` query results that silently shrink over the life of a long-running daemon without a corresponding full reindex.

**Phase to address:** Storage-engine/schema-design phase (concurrency model must be decided alongside the new SQLite schema, since PROJECT.md explicitly calls out "new storage format designed for concurrent access" as a requirement) and again at watcher/auto-sync implementation (where the actual write pattern under concurrent load is exercised).

---

### Pitfall 3: Naive incremental re-indexing silently corrupts the graph — stale nodes after renames, orphaned edges after deletes, dropped inbound edges

**What goes wrong:**
This is the single most common and highest-impact bug class across every comparable project surveyed (FalkorDB/code-graph, Understand-Anything, graphify, TS CodeGraph itself). Three distinct, independently-observed failure modes:
1. **Rename orphans:** if the changed-file detection is git-diff-based with default rename detection, a rename shows up as only the new path; the old path's nodes and edges are never pruned and persist forever as ghosts pointing at a file that no longer exists.
2. **Inbound-edge pruning bug:** a naive "delete edges where source OR target references a removed node" rule deletes edges *into* a changed file from files that were *not* re-analyzed in this pass — because those unchanged source files never get re-run, the inbound edges are gone forever, not just temporarily stale. (Documented in Understand-Anything issue #366: 194 inbound edges lost in one 64-file incremental run from this exact bug.)
3. **Symbol-removed-from-surviving-file leak:** when a file still exists but a symbol was deleted from it, incremental logic that only evicts nodes for files that "no longer exist on disk" never notices the symbol is gone — the stale node and its inbound edges survive indefinitely until a full clean rebuild. (Documented in graphify issue #1116.)

**Why it happens:**
Incremental update logic is almost always written and tested against the "happy path" (a file's content changed, re-parse it) and under-tested against the boundary cases (rename, delete, partial-content-removal-in-a-surviving-file) because those require multi-step fixtures to catch. The evidence-based invalidation model used by more mature projects (CoreGraph-class tools) — tagging every node/edge with the source file that "evidenced" it, and pruning strictly by evidence-file rather than by naive source/target matching — is the correct pattern but is not the obvious first implementation.

**How to avoid:**
Design the incremental-update algorithm around **evidence-based invalidation** from day one: every node and edge carries the file path whose parse produced it. On a file-change event: (a) for the changed file, delete all nodes/edges it produced (by evidence, not by re-checking target existence); (b) re-parse and re-insert; (c) re-run cross-file resolution only for edges whose *unresolved* endpoint could now point at something in the changed file — do not re-run resolution project-wide on every change (perf), but do not skip re-resolving inbound edges either (correctness). For renames, detect them explicitly (compare old-path/new-path pairs from `git diff --name-status -M`, or content-hash matching for non-git workflows) and treat as delete(old)+add(new), not as an unrelated pair of independent changes. For deletes-of-symbols-within-surviving-files, mark every extracted-this-run node ("origin" tag) and evict any previously-known node for that file whose id is absent from the fresh extraction, regardless of whether the file itself still exists. Build a differential test suite explicitly for these three cases (rename, delete-a-symbol-keep-the-file, delete-a-whole-file) before considering incremental sync "done" — treat this as a first-class Nyquist-style verification gate, not a nice-to-have.

**Warning signs:** node/edge counts that only grow over a long-running watcher session and never shrink even when code is deleted; `callers`/`callees` results that differ between a freshly-synced graph and a freshly-full-reindexed graph on the same commit; git-mv or IDE-rename operations followed by stale search results referencing the old path.

**Phase to address:** Incremental-indexer/watcher phase — this needs its own dedicated verification pass with rename/delete/edit fixtures, separate from "basic parsing works" and separate from "full index works." Do not let it ride along as an afterthought of the full-indexer phase.

---

### Pitfall 4: File watcher reliability gaps across macOS/Linux/Windows are treated as "someone else's library's problem" and go unhandled

**What goes wrong:**
fsnotify (the standard Go cross-platform watcher) has real, documented, per-platform limits that differ enough to cause production incidents if unhandled: Linux inotify has a `fs.inotify.max_user_watches` ceiling (distro-dependent default, commonly 8192–65536) that a large monorepo can exceed, producing "no space left on device" — a genuinely confusing error message for a disk-space-unrelated problem; macOS FSEvents/kqueue-based watching opens a file descriptor per watched path (not per directory) under some backends, hitting "too many open files" faster than Linux; Windows `ReadDirectoryChangesW` has a fixed default 64KB event buffer that silently drops events (`ErrEventOverflow`) under a burst of changes (e.g. `git checkout` of a large branch, `npm install`, a build tool touching thousands of files), and — critically — the Windows backend does **not** remove/re-target a watch when the watched directory itself is renamed, unlike every other backend. Additionally: fsnotify only watches *directories* non-recursively by default (subdirectories require explicit `Add` calls), and watching individual files rather than directories is explicitly discouraged because editors write via atomic temp-file-then-rename and the watch on the original path is silently lost after the first edit.

**Why it happens:**
The library's cross-platform abstraction hides real semantic differences (recursive vs. non-recursive, file-based vs. inode-based identity, buffer sizing) behind one API, so it's easy to write code that "works on my Mac" and fails invisibly on a teammate's Linux/Windows box or on a large enough repo.

**How to avoid:**
Watch directories, not individual files, and re-establish subdirectory watches on every `Create` event for a directory (fsnotify does not recurse automatically). Detect and handle `ErrEventOverflow` explicitly rather than assuming the events channel is complete — treat overflow as "trigger a full reconciliation scan of the affected subtree," not as an error to log and ignore. On Linux, document (and where possible programmatically check via `/proc/sys/fs/inotify/max_user_watches`) the sysctl requirement for large monorepos, and fail with an actionable error message rather than the raw "no space left on device" when the watch limit is hit. On Windows, explicitly re-add the watch after any rename event on a watched directory, since the platform won't do it automatically. Build a periodic reconciliation pass (hash-diff against the filesystem, independent of the watcher's event stream) as a correctness backstop — every comparable project surveyed that ships a `--full`/reconciliation fallback treats it as required infrastructure, not optional; watcher event streams should be treated as a performance optimization over polling, not a correctness guarantee.

**Warning signs:** graph staleness reports correlate with large file-count operations (git checkout, dependency install, IDE bulk-rename) rather than single-file edits; issues that reproduce only on Windows or only on Linux CI runners; watcher silently stops producing events after a directory rename on Windows.

**Phase to address:** Auto-sync/watcher phase, with explicit platform-matrix testing (not just "runs on the dev's machine") before calling the feature done.

---

### Pitfall 5: Underestimating cross-file resolution complexity per language — attempting all 12+ languages "at parity" in one pass stalls the whole port

**What goes wrong:**
Cross-file resolution is not a single algorithm applied uniformly across languages — every comparable multi-language code-graph tool surveyed (CoreGraph, sqry, SDL-MCP, the TS CodeGraph source itself) implements a *distinct* resolver per language family: TypeScript/JavaScript needs import-alias mapping, barrel re-export following, and namespace-import resolution; Java needs package-based namespacing and wildcard-import resolution; C#/C++ need header-pair inference and `using`/`#include` chain traversal; Rust needs module-tree construction from file paths and trait-method resolution via impl-block ownership; and several ecosystems (Swift↔Objective-C, React Native JS↔native bridge, Spring/DI interface-to-implementation binding) require framework-specific "dynamic dispatch synthesis" passes that go beyond static import-following entirely. Teams that treat "cross-file resolution" as one shared component to build once and parametrize per language consistently discover, language by language, that the shared abstraction doesn't fit and requires per-language rework — this is why TS CodeGraph itself ships a whole taxonomy of per-language bridge/synthesizer passes (React Native, Fabric/Codegen, Expo Modules, MyBatis XML-to-DAO, DI container binding) rather than one generic resolver.

**Why it happens:**
The surface area looks similar (name → definition lookup) but the actual resolution rules are semantically different per language and per framework convention, and framework-level dynamic dispatch (DI containers, native bridges) requires bespoke pattern recognition that has nothing to do with the base language's import syntax.

**How to avoid:**
This validates PROJECT.md's chosen sequencing (Go → Java/C# → Python → TS/JS → remainder) — but make the phase boundaries explicit about *what "parity" means per language*: ship a language's static (same-file + explicit-import) resolution first, and treat framework-level dynamic-dispatch synthesis (DI, native bridges, ORMs) as a separate, later-sequenced capability per language rather than a blocking requirement for that language's "done" milestone. Build a generic fallback resolver (project-wide name index with confidence-based disambiguation: 0 candidates → drop as external, 1 candidate → high-confidence edge, 2–5 candidates → emit all as ambiguous-but-useful, >5 → drop as too-generic-to-be-useful) as the baseline for every language, then layer semantic per-language resolvers (import-aware, scope-aware) on top only where it's proven to matter — this pattern appears independently in both TS CodeGraph-adjacent projects and lets a language ship with an 80%-useful resolver before its full semantic resolver is built. Track resolution completeness with an explicit "why unresolved" taxonomy (external package / dynamic dispatch not yet supported / genuinely missing) per language rather than a single "% resolved" number, so gaps are visible and prioritizable instead of hidden inside an aggregate metric.

**Warning signs:** a language's resolver PR keeps growing in scope because "just one more framework case" keeps surfacing; parity claims based on "the language parses" rather than "cross-file calls resolve correctly on a real-world sample repo in that language"; no visibility into what fraction of edges are unresolved and why.

**Phase to address:** Every language-support phase should split into (a) structural extraction + same-file resolution, (b) explicit-import cross-file resolution, (c) framework/dynamic-dispatch synthesis — with (c) explicitly allowed to lag behind and ship as a follow-up within the same language rather than gating the language's initial release.

---

### Pitfall 6: Migration tool against the TS SQLite schema is treated as a one-shot script instead of a defensive, resumable, validated conversion

**What goes wrong:**
SQLite's `ALTER TABLE` support is deliberately minimal (rename table, rename column, add column, drop column only in newer versions) — any structural change beyond that (changing a column type, adding a NOT NULL column with no default to a populated table, restructuring relationships) requires the full create-new-table/copy-data/drop-old/rename dance. A migration tool that does this in one large locked transaction on a real user's multi-GB `.codegraph/` SQLite file will hold a write lock for seconds-to-minutes, is entirely non-resumable if interrupted (disk full, process killed, laptop sleep mid-migration), and — because the new storage format is intentionally *not* schema-compatible with the old one (PROJECT.md explicitly calls for "a new storage format... with a migration tool converting existing indexes") — any partial or off-by-one-version-assumption failure risks silently producing a corrupt or incomplete Go-format index that "looks" migrated but is missing nodes/edges, since there's no equivalent of a foreign schema validator to catch it.

**Why it happens:**
Migration is usually built and tested against a small, clean, single-run fixture DB, not against a real user's `.codegraph/` directory that has been through years of incremental syncs, partial failures, and possibly multiple TS CodeGraph versions with schema drift of its own.

**How to avoid:**
Build the migration as an explicit multi-phase, resumable process (record progress per file/table processed, not just a single pass/fail), and validate against multiple *real* `.codegraph/` databases pulled from actual daily-use TS CodeGraph installs (Sean's own indexes are the obvious first fixtures, given PROJECT.md's context) rather than only synthetic small fixtures — real databases surface version-drift and partial-corruption edge cases synthetic ones don't. Never mutate the source TS SQLite file in place; migration should always read-only from the old file and write a fresh Go-format store, so a failed or interrupted migration leaves the original untouched and the migration is trivially re-runnable. Add a post-migration validation pass that checks structural invariants (node count sanity, no dangling edges, file-count parity with a fresh scan) and reports discrepancies rather than assuming success from "the script exited 0." Version-stamp the migration tool against the specific TS CodeGraph schema version(s) it supports and fail loudly (not silently skip rows) on an unrecognized schema shape, since TS CodeGraph's own schema has evolved across its 1.3.x line.

**Warning signs:** migration "succeeds" but node/edge counts in the new store don't match what a fresh full index of the same repo produces; migration tested only against freshly-created small test repos, never against an aged multi-month real index; no resumability if the migration process is killed partway.

**Phase to address:** Dedicated migration-tool phase, sequenced after the new storage format is finalized and stable — do not build the migration tool against a moving-target schema, and do not treat it as a footnote of the storage-format phase.

---

### Pitfall 7: Monorepo-scale indexing blows up memory because materialization patterns that are fine at demo scale (a few thousand files) are not fine at 10k+ files / millions of nodes

**What goes wrong:**
Every comparable project's issue tracker surveyed shows the same recurring shape: code that works fine on demo-sized repos allocates unbounded in-memory structures that scale with total graph size rather than with the size of the change being processed, and blows up RSS on real monorepos. Concrete documented instances: (a) a `getAllNodes()`/`getNodesByKind('method')`-style "load broad set into an array, then filter" pattern that materializes millions of rows before filtering — TS CodeGraph itself hit this exact bug on a 10M-node validation repo (PR #900: reference resolution allocating large same-name candidate arrays for common names like `Output`/`Result`/`string` drove the process to its heap limit, and a native-bridge method-map builder that eagerly materialized 2.6M method nodes drove RSS to ~4GB on its own); (b) tree-sitter parser-table/arena churn where a parser's internal LR tables get rebuilt on every `NewParser()` call instead of being cached/pooled, observed to drive one comparable Go indexer's RSS from an expected ~1.8GB to 9–100GB on a large external repo; (c) entity-ID duplication where long fully-qualified string IDs (e.g. `app/services/user.ts::function::createUser`) are cloned into every edge/index structure that references them, becoming the dominant memory cost at multi-million-entity scale.

**Why it happens:**
These patterns are invisible at the repo sizes used during day-to-day development and only manifest on genuinely large monorepos, which are rarely part of a fast local dev loop — so they ship, pass CI (small fixture repos), and then OOM on the first large real-world repo a user tries.

**How to avoid:**
From the outset, push filtering into SQL/storage-layer queries rather than loading broad result sets into Go memory and filtering in-process (this is directly analogous to the fix TS CodeGraph shipped in PR #900: filtered lookups by name+kind+file-prefix pushed into the query layer instead of `SELECT *` + in-memory filter). Cache/pool expensive per-parse resources (tree-sitter parser instances, grammar tables) rather than reconstructing them per file or per call. Intern repeated strings (entity IDs, file paths, symbol names) using an interning scheme (a `HashMap<string,int32>` handle table, or Go equivalent) once graph size crosses a threshold where duplication becomes the dominant memory cost — this is a well-known win pattern across multiple comparable projects and directly supports the "monorepo scale" requirement in PROJECT.md. Build a large-repo benchmark into the test/CI matrix early (a real 100k+-file open-source monorepo, not just fixtures) specifically to catch memory-scaling regressions before they reach users, and track peak RSS as a tracked metric alongside indexing throughput and query latency (which PROJECT.md already commits to publishing as benchmarks).

**Warning signs:** indexing throughput that degrades non-linearly as repo size grows (rather than roughly linearly); CI/dev-loop test repos are all small-to-medium, with no large-repo test in the regular suite; memory profiling only happens reactively after a user reports an OOM, not proactively as a gate.

**Phase to address:** Should be an explicit non-functional requirement validated at the end of the core-indexer phase and re-validated whenever a new language extractor or cross-file resolver is added (since resolvers are exactly where the candidate-set-materialization pattern above tends to reappear) — not deferred to a later "scale/perf" phase, since PROJECT.md commits to monorepo-scale storage design from v1.

---

### Pitfall 8: Goroutine leaks in the long-running watcher/daemon process go undetected until memory/goroutine-count creeps up over days

**What goes wrong:**
A long-running watcher process is exactly the shape of Go program most prone to slow-burn goroutine leaks: a `context.Context` that's created but never canceled on the happy path (only handled on error), a `time.Ticker` started without a paired `defer t.Stop()`, or a channel send/receive with no corresponding receiver/sender after a shutdown path runs. These leaks are invisible in short-lived test runs and integration tests (the process exits before the leak accumulates) and only manifest as slowly climbing RSS/goroutine count over hours-to-days of real daemon uptime — exactly the operating profile of an "auto-sync keeps the graph current" watcher that PROJECT.md requires. fsnotify's own maintainers have documented and worked around a related deadlock class in their own event-draining loop (fsnotify/fsnotify#502), underscoring that this failure mode is real even in well-maintained watcher code, not just application-level bugs.

**Why it happens:**
Context-cancellation and ticker-cleanup discipline is easy to get right in the "obvious" code path and easy to miss in error/shutdown paths that are exercised rarely in tests; unit tests that don't assert "goroutine count returned to baseline after Close()" will pass indefinitely while leaking.

**How to avoid:**
Every long-running goroutine the watcher/daemon spawns must take a `context.Context` and be provably torn down on `Close()`/shutdown — pair every `context.WithCancel` with a deferred `cancel()`, every `time.NewTicker` with a deferred `Stop()`, and every fan-out goroutine with a `sync.WaitGroup` that `Close()` waits on before returning (mirroring the pattern used by Go's own gopls filewatcher: a `run()`/`process()` goroutine pair with an explicit `stop` channel and `wg.Wait()` in `Close()`). Add goroutine-leak detection to the test suite (e.g. `goleak`-style "no goroutines running after test/Close() that don't match an expected baseline") specifically for watcher and daemon lifecycle tests, not just unit tests of pure functions. Expose a debug/metrics endpoint (or at minimum a `pprof` hook, gated behind an explicit flag/env var as several comparable projects do) so goroutine-count and RSS drift can be diagnosed in the field without needing to reproduce locally.

**Warning signs:** daemon/watcher RSS or goroutine count (visible via `runtime.NumGoroutine()` or pprof) that only grows over a multi-day uptime and never plateaus; issue reports describing "codegraph got slow/used a lot of memory after running for a while" with no obvious single trigger.

**Phase to address:** Auto-sync/watcher phase, with an explicit soak-test (run the watcher against a repo with simulated ongoing changes for an extended period, assert stable goroutine/memory baseline) as part of that phase's verification, not deferred to a post-launch bug report.

---

### Pitfall 9: MCP tool output shape and CLI semantics drift from the TS original in ways that silently break existing agent configs

**What goes wrong:**
The project's core value proposition is "an agent user can uninstall TS CodeGraph, install the Go binary... and everything works the same or better" — this is a much stricter bar than typical API compatibility, because MCP clients (Claude Code, Cursor, etc.) parse tool *output* shape, not just tool *input* schema, and any drift (a renamed field, a restructured JSON response, a changed error-message format, different exit codes from CLI commands) breaks agents that have learned to parse the old shape, often silently: the agent doesn't crash, it just gets worse results and no one notices immediately. Documented failure patterns from other MCP server rewrites/ports: schema drift between a tool's declared `inputSchema` and what the handler actually validates (a parameter rename that the SDK doesn't enforce at the framework boundary, so malformed calls pass through silently and the handler either crashes or returns garbage); tool descriptions/response shapes changing without a corresponding "breaking change" signal, which is exactly why dedicated MCP-schema-diffing tools (mcp-drift, mcp-schema-evolution) now exist as a category.

**Why it happens:**
Ports naturally reimplement rather than literally copy every response-formatting detail, and it's tempting to "improve" a response shape while porting it — but for a drop-in-replacement goal, any such improvement is a breaking change unless it's additive-only (new optional fields) and the original shape is preserved.

**How to avoid:**
Before writing the Go MCP server, capture a **golden-output corpus**: run the actual TS CodeGraph MCP server and CLI against a fixed set of real repos and record exact tool-call outputs, CLI stdout/stderr/exit-codes, and error message formats for every tool/command in the parity surface (`codegraph_explore` and companions, `init`/`install`/`uninstall`/`uninit`/`upgrade`/`explore`). Treat that corpus as the acceptance test for the Go port — a byte-for-byte or structurally-equivalent match (excluding genuinely nondeterministic fields like timestamps) is the parity bar, not "looks similar." Validate JSON-RPC/stdio hygiene explicitly (never write non-protocol bytes to stdout in the MCP server; all diagnostics to stderr) since this is a common, easy-to-miss mistake in any new MCP server implementation, TS or Go. Where the new implementation deliberately changes behavior (e.g. a new storage format enabling something the old one couldn't), make it additive (new optional fields/tools) rather than altering existing response shapes, and gate any genuinely breaking change behind an explicit version negotiation rather than silent drift.

**Warning signs:** golden-corpus diffing isn't part of the test suite (parity is being asserted by eyeballing, not by automated comparison); tool response fields get renamed or restructured "because it's cleaner in Go" during the port; no test coverage of CLI exit codes and stderr/stdout separation.

**Phase to address:** Should be set up as test infrastructure *before* the MCP-server and CLI phases begin (capture the golden corpus from the TS original early, while it's still easy to run both side by side), then used as the acceptance gate for those phases.

---

### Pitfall 10: Supply-chain "theater" — signing, SBOM, and reproducible-build claims that don't actually verify anything

**What goes wrong:**
It's easy to add cosign signing, an SLSA workflow, and an SBOM generator to a release pipeline and have every step individually "work" (commands exit 0, artifacts are produced) while the overall chain doesn't actually deliver the security property being claimed. Concrete gaps observed in real Go supply-chain setups: (a) using `goreleaser` alone for multi-target builds only achieves SLSA **Build L2** (not L3) because the build environment isn't isolated from the calling workflow — L3 requires the provenance to be generated by a separate reusable workflow with its own OIDC identity that the calling workflow cannot influence, which most goreleaser-only setups don't wire up; (b) reproducible-build claims broken by non-obvious non-determinism sources: embedded absolute filesystem paths (fixed by `-trimpath`), embedded build timestamps via `time.Now()` in `init()` or a stamped `BuildTime` ldflag (fixed by deriving the timestamp from the commit rather than build-wall-clock), CGo-linked dependencies pulling in a non-deterministic system C toolchain (a second reason to avoid CGo for parser bindings, beyond cross-compilation), and unpinned Go toolchain versions across CI runs; (c) an SBOM that lists declared dependencies but was never cross-checked against what's actually compiled into the binary (dead-code-eliminated deps still appearing in a naive SBOM), which weakens its value as an attack-surface description even though it's technically "generated."

**Why it happens:**
Each individual tool (cosign, SLSA generator, SBOM generator) is easy to bolt on and demo successfully in isolation; the properties that actually matter (build isolation for L3, byte-for-byte reproducibility, SBOM-matches-actual-binary-contents) require integration discipline across the whole pipeline that isn't enforced by any single tool succeeding.

**How to avoid:**
Decide explicitly which SLSA build level is the actual target (L2 via goreleaser is meaningfully cheaper and may be adequate; L3 requires the `slsa-framework/slsa-github-generator` reusable-workflow pattern, run as an isolated second job that attests to already-built binaries/images rather than building them itself) and verify the choice was actually achieved — don't assume goreleaser + a signing step equals L3. Set `-trimpath`, disable or carefully pin CGO for release builds, derive any embedded build-time value from the commit timestamp (`git log -1 --format=%ct`) rather than wall-clock `time.Now()`, and pin the Go toolchain version via the `go.mod` toolchain directive so CI and any independent verifier use byte-identical compilers. Validate reproducibility empirically, not just by following a recipe: build the same commit twice (ideally on two independent runners/environments) and diff the SHA-256 of the output binaries as a CI gate, catching regressions the moment they're introduced rather than discovering non-reproducibility later when someone tries to independently verify a release. Run `govulncheck` (or equivalent) as a distinct step from SBOM generation — the SBOM proves what's in the binary, vuln scanning is what actually catches known-bad dependencies; don't conflate the two or treat SBOM presence as vulnerability coverage.

**Warning signs:** SLSA provenance badge/claim exists but no one has run `slsa-verifier` against a real release to confirm it actually verifies; two builds of the same tag produce binaries with different hashes; SBOM generation and dependency-vuln-scanning are the same CI step (a sign they're being conflated); no CI job that specifically re-builds and hash-diffs to prove reproducibility.

**Phase to address:** Release-engineering phase, with an explicit verification step (not just "the workflow ran") — reproducibility and SLSA-level claims should be proven with a CI gate (double-build hash diff, `slsa-verifier` run against a real published release) before they're advertised as project features, since PROJECT.md commits to this as a differentiator, not an afterthought.

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|-----------------|------------------|
| Ship CGo tree-sitter for the first native-only milestone build (e.g. Go-language support only, on the dev's own platform) | Fastest possible parse throughput, unblocks early feature work | Cross-compilation and the "single static binary, all platforms" story break the moment a second platform target is needed; expensive to retrofit if language-extractor code assumes CGo-only tree-sitter APIs | Only as a throwaway spike to validate parser output quality before committing to a strategy — never as the shipped v1 parser path, given the project's explicit static-binary constraint |
| Full-rebuild-on-every-change instead of evidence-based incremental invalidation | Trivially correct, no rename/delete edge cases to get wrong | Unusable auto-sync UX on any repo large enough to matter (defeats the "no manual re-runs" requirement); may mask the very correctness bugs that need fixing before incremental sync ships | Acceptable as the CLI-triggered `--force`/full-reindex fallback path (every comparable project keeps one) — never acceptable as the *only* sync mechanism given PROJECT.md's auto-sync requirement |
| Skip evidence-tagging on nodes/edges and prune by naive source/target file match | Simpler initial data model | Directly reproduces the rename-orphan and inbound-edge-pruning bugs documented in Pitfall 3 across multiple comparable projects | Never — this is cheap to build correctly from the start and expensive to retrofit into an existing schema/index |
| Defer per-language framework/dynamic-dispatch synthesis (DI, native bridges, ORMs) past a language's initial ship | Unblocks shipping a language's static resolution sooner | Some real-world call graphs will look incomplete/wrong (interface calls, cross-language bridges) until synthesis passes land | Acceptable and recommended per Pitfall 5 — as long as it's tracked and visible (resolution-outcome taxonomy), not silently absent |
| Single-connection SQLite writer with no write queue prioritization | Simple to reason about, avoids most lock contention bugs immediately | Under heavy concurrent multi-agent-session load, writes serialize and queue depth could grow; acceptable until proven otherwise by load testing | Acceptable for v1 given SQLite's actual single-writer nature — revisit only if benchmarks show writer-queue latency is a real problem at target scale |

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|-------------------|
| MCP stdio transport | Writing debug/log output to stdout, corrupting the JSON-RPC message stream and producing a cryptic "MCP server disconnected" error with no clue pointing at the real cause | All diagnostic/log output must go to stderr; never `fmt.Println`/`log.Print`-to-stdout anywhere in the MCP server binary's code path |
| `database/sql` connection pooling with SQLite | Assuming one `PRAGMA` exec at startup configures every future connection; assuming `SetMaxOpenConns` default (unlimited) is safe for a writer pool | Configure PRAGMAs via DSN parameters so every new pooled connection gets them; explicitly set `SetMaxOpenConns(1)` on the writer pool |
| fsnotify on Windows | Assuming a watch survives a directory rename, as it does on every other backend | Explicitly detect rename events on watched directories and re-add the watch on Windows; don't assume backend parity |
| Cross-language MCP client compatibility | Assuming an MCP SDK enforces the declared `inputSchema` at the framework boundary | Validate/parse arguments explicitly inside the handler (don't trust the client sent a schema-conformant payload) and return a structured tool-error, not a thrown exception, on mismatch |

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|-----------------|
| In-memory materialize-then-filter on node/edge queries (`getAllNodes()`-style) | Indexing throughput degrades non-linearly as repo size grows; RSS spikes during reference resolution specifically | Push name/kind/file-prefix filters into SQL; use streaming iterators for scan-and-filter passes | Documented to hit hard OOM around 10M-node graphs (TS CodeGraph's own validation repo); likely visible well before that on constrained dev machines |
| Re-constructing tree-sitter parser instances/grammar tables per file or per call instead of pooling | RSS balloons far beyond the size of the actual parsed content; process instability under sustained indexing | Pool/reuse parser instances; cache grammar tables once per language, not per parse call | Observed to drive RSS from an expected ~1.8GB to 9–100GB on a large external repo in a comparable Go indexer |
| Un-interned duplicate string IDs (fully-qualified entity names) cloned across every edge/index structure | Peak memory scales faster than node/edge count would suggest, dominated by string duplication rather than actual graph content | Intern entity IDs/paths/names behind integer handles once graph size crosses a meaningful threshold | Documented as the dominant memory cost at ~2.7M-entity / ~1.7M-edge scale in a comparable project |
| Watcher event-stream treated as a complete, gap-free source of truth | Graph silently diverges from disk state after event-buffer overflow (large git operations, dependency installs) with no visible error | Treat overflow (`ErrEventOverflow`) as a trigger for full-subtree reconciliation, and run periodic hash-diff reconciliation independent of the event stream as a backstop | Breaks specifically under high-event-volume operations (bulk file operations), not steady-state single-file edits |

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Treating cosign signing + SLSA workflow presence as proof of a specific SLSA build level | Users/downstream consumers trust a security claim (e.g. "L3") that wasn't actually achieved, because goreleaser-only pipelines top out at L2 without the isolated reusable-workflow step | Explicitly verify the achieved level with `slsa-verifier` against a real published release before advertising it |
| SBOM generated from declared `go.mod` dependencies rather than what's actually linked into the binary | SBOM overstates or misrepresents the real attack surface (dead-code-eliminated deps still listed), undermining its value for downstream vulnerability triage | Generate SBOM from the actual binary (Syft-against-artifact style) where feasible, not purely from the module graph |
| Conflating SBOM generation with vulnerability scanning as "supply chain done" | Known-vulnerable transitive dependencies ship undetected because no one is actually running `govulncheck`/equivalent as a distinct gate | Run vulnerability scanning as its own CI gate, separate from and in addition to SBOM generation |
| MCP tool input trusted without validation because "the client declared the schema" | A hallucinated or malformed tool-call argument (e.g. wrong type, unexpected array) reaches business logic unchecked, potentially causing crashes or, in a worse case, being used to construct unsafe filesystem/shell operations if the CLI-wrapping pattern is used anywhere internally | Validate at the handler boundary explicitly; never pass client-supplied strings into shell/`exec.Command` with `shell: true`-equivalent semantics — always pass args as an array |

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-------------------|
| Full reindex on a large existing DB does row-by-row `DELETE` before rebuilding, making the CLI appear hung for minutes with no progress indicator | User assumes the tool crashed or is broken; loses trust in the port before parity is even evaluated | Drop and recreate the DB file (under the same lock) instead of row-by-row delete for full rebuilds, and show a progress indicator immediately, not only after the slow pre-step completes — TS CodeGraph itself hit and fixed this exact issue (PR #900) |
| Auto-sync reports "healthy"/"up to date" even when incremental updates have been silently failing (e.g. FK violations dropped edges) | Users trust stale/wrong query results for an extended period with no signal anything is wrong | Surface degraded-index state explicitly in `status`/`sync` output whenever any incremental update step fails, rather than only logging it |
| Migration tool gives no indication of progress or resumability on a large existing index | User can't tell if a multi-minute migration on a large `.codegraph/` directory is progressing or stuck, and has no safe way to retry after an interruption | Make migration explicitly resumable with visible per-file/per-table progress, and never mutate the source file so retry is always safe |

## "Looks Done But Isn't" Checklist

- [ ] **Incremental sync:** Often missing rename-detection and inbound-edge preservation — verify with a fixture test that does git-mv on a file with cross-file callers and asserts both the old-path node is gone and the inbound edges to the new path survive.
- [ ] **Cross-file resolution "parity" for a language:** Often means "parses without error," not "resolves calls correctly" — verify against a real-world sample repo in that language with known call-graph shape, not just a syntax-coverage fixture.
- [ ] **MCP tool parity:** Often verified by manual eyeballing of one or two example calls — verify with an automated golden-output diff against the TS original across the full tool surface and a representative repo set.
- [ ] **Static binary / cross-platform build:** Often verified only on the developer's native platform — verify CI actually cross-builds and boots (not just links) on all three target OSes, including a check that no CGo dependency slipped in.
- [ ] **SQLite concurrency:** Often "tested" only with a single writer in dev — verify under an explicit concurrent-writer load test (simulated multiple agent sessions + watcher) that reproduces `SITE_BUSY`/FK-failure conditions before shipping.
- [ ] **Reproducible builds / SLSA claims:** Often "implemented" by following a recipe once — verify with an actual double-build hash-diff CI gate and a `slsa-verifier` run against a real release artifact.
- [ ] **Watcher reliability:** Often tested only with single-file edits — verify against bulk operations (git checkout of a large diff, `go mod tidy`-scale file churn) that can overflow platform event buffers.
- [ ] **Migration tool:** Often tested only against small synthetic fixtures — verify against multiple real, aged `.codegraph/` directories with a post-migration structural-invariant check.

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|----------------|-----------------|
| Stale/orphaned nodes from incremental-sync bugs (Pitfall 3) | LOW | Ship a `--force`/full-reindex path (as every comparable project does) as the immediate user-facing recovery; fix the incremental algorithm separately, informed by the specific failure fixture |
| CGo/cross-compilation lock-in discovered late (Pitfall 1) | HIGH | Requires re-architecting the parser integration layer around an abstraction that can swap CGo/WASM/pure-Go backends; cost scales with how much per-language extractor code directly assumed CGo-specific APIs, so isolating the tree-sitter binding behind an internal interface from day one drastically lowers this cost if a switch is later needed |
| SQLite lock contention under real multi-session load discovered post-launch (Pitfall 2) | MEDIUM | Split reader/writer pools and add `_txlock=immediate` — a schema-compatible change that doesn't require a data migration, but does require a release and user upgrade |
| Migration tool corrupts or incompletely converts a user's index (Pitfall 6) | LOW (if source untouched) / HIGH (if source was mutated) | If the migration tool is correctly read-only against the source, recovery is "delete the bad output, re-run the (fixed) migration tool" — this is exactly why never mutating the source file is a hard requirement, not a nice-to-have |
| Non-reproducible or under-leveled supply-chain claims discovered by an external auditor (Pitfall 10) | MEDIUM | Re-run the release pipeline with the isolated-workflow / `-trimpath` / pinned-toolchain fixes, re-publish provenance for the affected release, and add the double-build hash-diff CI gate so it can't regress silently again |

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|-------------------|----------------|
| CGo tree-sitter breaks static-binary/cross-compile story | Parser-strategy/architecture phase (before any language extractor work) | CI cross-builds and boot-tests on all three target OSes; confirm zero CGo dependency if pure-Go/WASM path chosen |
| SQLite WAL misunderstood as full concurrency | Storage-engine/schema-design phase; re-verified at watcher phase | Concurrent-writer load test reproduces and confirms absence of `SQLITE_BUSY`/FK failures under simulated multi-agent-session load |
| Incremental sync corrupts graph on rename/delete | Incremental-indexer/watcher phase (dedicated verification, not folded into full-indexer phase) | Fixture suite explicitly covering rename, symbol-delete-in-surviving-file, and file-delete, diffed against a full-reindex baseline |
| Watcher unreliable across platforms | Auto-sync/watcher phase | Platform-matrix test including bulk-file-operation event-overflow scenarios on Linux/macOS/Windows |
| Cross-file resolution complexity per language stalls parity | Every language-support phase, split into static/import/framework-synthesis sub-stages | Resolution-outcome taxonomy tracked per language; "parity" defined as call-graph correctness on a real sample repo, not just parse coverage |
| Migration tool corrupts/loses data | Dedicated migration-tool phase, after storage format is finalized | Post-migration structural-invariant check against multiple real aged `.codegraph/` fixtures; resumability verified by killing the process mid-run |
| Monorepo-scale memory blowup | End of core-indexer phase; re-verified whenever a new resolver/extractor is added | Large real-world repo (100k+ files) in the benchmark/CI matrix; peak RSS tracked as a first-class metric alongside throughput and query latency |
| Goroutine leaks in long-running watcher/daemon | Auto-sync/watcher phase | Soak test (extended-duration run with simulated ongoing changes) asserting stable goroutine count/RSS baseline; leak-detection assertions in lifecycle tests |
| MCP/CLI behavioral drift from TS original | Test-infrastructure setup before MCP-server/CLI phases begin | Golden-output corpus captured from the real TS CodeGraph, diffed automatically against the Go port's output on every relevant tool/command |
| Supply-chain signing/SBOM/reproducibility theater | Release-engineering phase | Double-build hash-diff CI gate; `slsa-verifier` run against a real published release confirming the claimed build level; `govulncheck` as a distinct gate from SBOM generation |

## Sources

- fsnotify/fsnotify official docs and source (`fsnotify.go`, `backend_inotify.go`) — platform-specific watch limits, rename-on-Windows gap, buffer overflow behavior
- fsnotify/fsevents (macOS FSEvents wrapper) — 4096-path watch ceiling, per-file descriptor cost, "unstable API" caveats
- `smacker/go-tree-sitter` GitHub issue #120 and `golang/go` issue #73406 — CGo cross-compilation breakage and cross-linked-binary crash reports
- dvcdsys/code-index PR #78 and PR #81 — real-world OOM root-caused to tree-sitter parser-table churn; wazero-vs-CGo-vs-pure-Go tree-sitter benchmark data
- colbymchenry/codegraph (source project) issue #773 and PR #900 — documented incremental-sync FK-drop bug and its fix; documented large-repo OOM root causes and fixes, directly on the project being ported
- FalkorDB/code-graph issue #665, Egonex-AI/Understand-Anything issue #366, safishamsi/graphify issue #1116 — independently-observed incremental-indexing correctness bugs (rename orphans, inbound-edge pruning, surviving-file symbol-delete leaks)
- simplecore-inc/coregraph docs (`change-tracking.md`) — evidence-based invalidation model as the correct pattern
- mattn/go-sqlite3 issue #274/#1022, modernc.org/sqlite usage guide, tenthousandmeters.com SQLite concurrent-writes analysis — WAL-mode semantics, `BEGIN IMMEDIATE` vs deferred, reader/writer pool splitting
- Reproducible Builds in Go practical guides (safeguard.sh) and `slsa-framework/slsa-github-generator` README — `-trimpath`, toolchain pinning, SLSA L2-vs-L3 distinction, goreleaser integration gaps
- Stack Harbor goroutine-leak runbook, Go gopls `filewatcher.go` source, kubernetes/kubernetes PR #137213 — goroutine-leak patterns and the context-cancel/ticker-stop/WaitGroup-teardown fix pattern
- MCP production-gotchas writeups (albinogeek.com, mcp-drift, mcp-schema-evolution) and mark3labs/mcp-go PR #834 — stdio stdout-corruption trap, schema-validation gaps, tool-schema-drift detection tooling as a now-emerging category
- zzet.org (gortex) cross-language resolution writeup, simplecore-inc/coregraph, sqry.dev docs, GlitterKill/sdl-mcp docs — per-language and per-framework resolver complexity as the norm across comparable multi-language code-graph tools

---
*Pitfalls research for: local-first code knowledge graph / code intelligence tool for AI coding agents (Go port)*
*Researched: 2026-07-10*
