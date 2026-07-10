<!-- GSD:project-start source:PROJECT.md -->

## Project

**CodeGraph Go**

A ground-up Go rewrite of [CodeGraph](https://github.com/colbymchenry/codegraph) — the pre-indexed code knowledge graph for coding agents (Claude Code, Cursor, Codex, Gemini, etc.). One static Go binary replaces the TypeScript version's bundled-Node distribution, delivering the same agent-facing experience (CLI, MCP server, auto-sync) with better performance, a verifiable supply chain, and an architecture designed to grow into team-scale usage.

**Core Value:** An agent user can uninstall TypeScript CodeGraph, install the Go binary, migrate their indexes, and everything works the same or better — faster, from a single verifiably-built binary.

### Constraints

- **Tech stack**: Go (latest stable), single static binary per platform — the performance and supply-chain story both depend on it
- **Supply chain**: Minimal, audited dependencies; prefer pure Go (CGo only with explicit justification pending parser research); signed + attested + reproducible + SBOM'd releases
- **Compatibility**: Behavioral parity with TS CodeGraph v1.3.x agent-facing surface (MCP tools, CLI semantics); one-way migration from its SQLite format
- **Architecture**: v1 storage and process design must accommodate milestone-2 team features (central server, CI-distributed indexes, concurrent access) without a rewrite
- **Licensing**: Original is MIT — port with attribution

<!-- GSD:project-end -->

<!-- GSD:stack-start source:research/STACK.md -->

## Technology Stack

## Recommended Stack

### Core Technologies

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| Go | latest stable (1.24/1.25 line) | Language/runtime | Static binaries, cross-compilation, project constraint |
| `github.com/tree-sitter/go-tree-sitter` | v0.25.x + per-language grammar modules (`tree-sitter/tree-sitter-<lang>`) | Source parsing into concrete syntax trees | Official, org-maintained since Aug 2024 (successor to the community `smacker/go-tree-sitter` fork); only path with mature, broad, actively-updated grammar coverage for 12+ languages today. **This is a justified CGo exception** — see "The Parser Decision" below. |
| `modernc.org/sqlite` | latest (v1.3x line) | Pure-Go SQLite driver — used ONLY for the migration reader that ingests existing TS-CodeGraph `.codegraph/` SQLite indexes | CGo-free, cross-compiles trivially. Do not use for the new graph store itself (see Storage below) — this is purely the read path for the one-way migration tool. |
| `github.com/mark3labs/mcp-go` | latest (v0.3x+ line) | MCP server (stdio transport, tool registration) | Broadest real-world adoption among Go MCP servers today, simple ergonomic API (`server.NewMCPServer`, `mcp.NewTool`, `server.ServeStdio`), lets v1 ship now. Re-evaluate the official SDK at each subsequent milestone (see Alternatives). |
| `github.com/spf13/cobra` | v1.9.x | CLI framework | De facto standard for Go CLIs (kubectl, docker, gh, hugo); built-in shell-completion and man-page generation map directly onto the parity command surface (`init`, `install`, `uninstall`, `uninit`, `upgrade`, `explore`, migrate). |
| `github.com/fsnotify/fsnotify` | v1.9.x | Cross-platform low-level file event notifications | Only realistic cross-platform (inotify/kqueue/ReadDirectoryChangesW) primitive in the Go ecosystem; auto-sync must be built on top of it, not instead of it. |

### Storage — the new graph format

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| `github.com/cockroachdb/pebble` | latest | Embedded LSM key-value store backing the new graph format | Pure Go, no CGo, built specifically because CGo-wrapped RocksDB was untenable for CockroachDB's hot path — same motivation this project has. Handles concurrent access far better than bbolt's single-writer model (critical for "optimized for concurrent access and monorepo scale"), has range deletes (useful for pruning stale file/symbol subgraphs on re-index) and snapshots (useful for consistent-read MCP queries while a background re-index is writing). Actively maintained, production-proven at large scale. |

### Supporting Libraries

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/tree-sitter/go-tree-sitter` grammar modules | per-language, pinned | Language grammars | One `go.mod` require per supported language (`tree-sitter-go`, `tree-sitter-java`, `tree-sitter-c-sharp`, `tree-sitter-python`, `tree-sitter-typescript`, etc.) — only pull in what's needed per the Go → Java/C# → Python → TS/JS priority order. |
| `github.com/goreleaser/goreleaser` | v2.x | Release build/packaging automation | Cross-platform builds, checksums, SBOM (Syft) and cosign signing wired in as native config blocks; the standard tool for exactly this supply-chain bundle. |
| `github.com/sigstore/cosign` (v3 CLI) | latest | Keyless artifact signing | Sign the GoReleaser checksums file via GitHub Actions OIDC (`id-token: write`) — no long-lived private key to manage or leak. |
| `slsa-framework/slsa-github-generator` (`builder_go_slsa3.yml`) | pinned `@vX.Y.Z` tag | SLSA Build Level 3 provenance | Reusable GH Actions workflow purpose-built for Go binaries; produces verifiable provenance consumable by `slsa-verifier`. |
| `anchore/syft` (via GoReleaser's `sbom:` block) | latest | SBOM generation | Default choice — one line of GoReleaser config (`sbom: {artifacts: all}`), covers Go and any non-Go release assets uniformly. |
| `CycloneDX/cyclonedx-gomod` (optional, `app` mode) | latest | Higher-fidelity Go-specific SBOM | Use instead of / alongside Syft if you need SBOMs that respect Go build constraints per platform (only modules actually compiled into each binary) — more precise than a generic Syft scan for a multi-platform Go release. |
| `golang.org/x/vuln/cmd/govulncheck` | latest | Vulnerability scanning gate | `govulncheck ./...` in CI; call-graph-aware (flags only reachable vulns, not the whole dependency tree) — lower noise than naive SCA tools, output formats include SARIF for GitHub code-scanning integration. |
| `github.com/spf13/viper` | latest | Layered config (flags/env/file) | Pairs with Cobra for CLI config; only pull in if config complexity grows beyond flags + a single config file. |
| `github.com/tetratelabs/wazero` | latest | Pure-Go WASM runtime | Only needed if/when the Phase-1 parser spike (see below) decides to move tree-sitter grammars to a WASM sandbox instead of CGo. Not a v1 dependency under the primary recommendation. |

## Installation

# Core

# Dev / CI tooling (not go.mod deps — install as CLI tools in CI)

# goreleaser + slsa-github-generator installed as GitHub Actions, not go-installed

## The Parser Decision (the central open question)

### Option A — CGo tree-sitter bindings (`tree-sitter/go-tree-sitter`) — RECOMMENDED for v1

- **Maturity:** Official, org-maintained, actively released through 2026. Grammar coverage for all 12+ target languages exists today as separate, independently-versioned Go modules.
- **Performance:** Baseline — every other option is measured against this. Full-parse and incremental-reparse performance is the best available.
- **Cost:** Breaks `CGO_ENABLED=0`. Requires a C toolchain in CI for every target platform (solvable — `zig cc` or `musl-cross` cross-compilation is a well-trodden path with GoReleaser, and this project's `mattn/go-sqlite3`-equivalent problem doesn't even arise here since SQLite itself won't be CGo — see Storage). The end-user artifact is still a single static binary per platform; CGo only complicates the *build* environment, not the *distribution* format.
- **Risk:** A memory-safety bug in a grammar's hand-written C scanner (some language grammars have external C/C++ scanners, e.g. for heredocs or significant whitespace) can, in the worst case, crash the whole host process — including the MCP server — since there's no process/memory isolation between the C code and the Go host. For a tool that parses arbitrary, occasionally adversarial, third-party monorepo code, this is a real reliability tail-risk, not just a purity concern.
- **Verdict:** This is the **justified CGo exception** referenced in the project's constraints. Given today's tooling landscape, it's the only path with both full 12-language coverage and top-tier performance. Cross-compilation risk is manageable with standard tooling (zig cc + GoReleaser); the crash-isolation risk is real but has historically been rare in practice for the well-exercised grammars this project needs first (Go, Java, C#, Python, TS/JS).

### Option B — tree-sitter grammars compiled to WASM, run via `wazero` — WORTH A PHASE-1 SPIKE, not v1-default

- **Precedent that the pattern works at scale:** `ncruces/go-sqlite3` ships the *entire* SQLite C library compiled to WASM and run through wazero as a production, CGo-free `database/sql` driver — proof the approach scales to a C codebase considerably larger and more complex than tree-sitter's runtime. This is a genuinely strong signal that a WASM/wazero tree-sitter is technically viable, not just theoretically possible.
- **The official maintainers explicitly declined this path** (`tree-sitter/go-tree-sitter#16`): they are not removing the C dependency, and if they ever added a WASM backend they'd target `wasmtime`, not `wazero`. That means there is **no official, ready-made WASM distribution of tree-sitter + grammars to consume** — building one is this project's own engineering work: compiling the tree-sitter C runtime and each grammar's C sources to a standalone WASM module (via `zig cc` or `wasi-sdk`), plus a hand-written C-ABI shim, since wazero can only pass scalar values across the host/guest boundary and tree-sitter's C API passes structs (`TSNode`, `TSPoint`) by value.
- **Performance cost:** informal/community benchmarks of this exact pattern (WASM tree-sitter via wazero vs. CGo tree-sitter) put the WASM path at roughly **2x slower** than CGo for full parsing, dominated by host↔guest call-boundary overhead per AST node — mitigable but not eliminated by batching node exports. Treat this figure as directional, not authoritative (single low-confidence community source); a Phase-1 benchmark on this project's actual target languages and corpora is required to get a trustworthy number.
- **Benefit beyond purity:** true crash/fault isolation — a WASM guest trap becomes a recoverable Go error instead of a process-killing SIGSEGV. This directly addresses Option A's reliability tail-risk when parsing adversarial or malformed third-party code.
- **Verdict:** Real, credible, and aligned with the project's stated pure-Go preference — but it requires building and maintaining a non-trivial grammar-to-WASM compilation pipeline for every one of the 12+ target languages, with no existing mature open-source library to lean on (the closest prior art, `malivvan/tree-sitter`, is explicitly pre-release/experimental). **Recommendation: spike this in Phase 1 against 2-3 real languages (start with Go, since it's the first language target) and benchmark indexing throughput on a real monorepo before committing.** If the ~2x parse-time cost is invisible against the rest of the indexing pipeline (symbol resolution, storage writes, embedding — as one community report claimed) and the pipeline engineering cost is acceptable, migrate the parser layer behind an interface swap; if not, Option A stands.

### Option C — Native pure-Go tree-sitter reimplementation — NOT RECOMMENDED for v1, monitor only

- Community projects reimplementing the tree-sitter parse-table runtime in pure Go exist and claim substantial performance wins over the CGo binding, particularly on incremental reparse (the dominant editor/agent workload).
- **Do not adopt for v1.** These projects are new, have thin community/production track records, and — critically — solving 12+ language grammar-table generation and external-scanner compatibility from scratch is exactly the "huge effort" the project's own constraints document already flagged as the downside of this path. The risk of picking an immature, single-maintainer dependency for the load-bearing parser of the whole product is disproportionate to the payoff versus Option A or a validated Option B.
- Revisit only if, after a real v1 ships, Option A's CGo cross-compilation or crash-isolation costs prove painful in practice AND a specific pure-Go runtime has matured (broader grammar coverage, multiple maintainers, production adopters beyond its own author).

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|--------------------------|
| `mark3labs/mcp-go` | `modelcontextprotocol/go-sdk` (official) | The official SDK is maintained in collaboration with Google and is spec-complete for the current MCP revision, shipping 2026-07-28 protocol support in v1.7.0 pre-releases as of mid-2026 — it is the long-term correct target. It's newer and has less real-world Go-server mileage than `mark3labs/mcp-go` today. Re-evaluate at the next milestone boundary; if the official SDK's stdio transport and tool-registration ergonomics have matured and its adoption has grown, migrate. Don't block v1 parity on this decision either way — the two APIs are structurally similar enough that a later swap is a bounded refactor, not a rewrite. |
| `cockroachdb/pebble` | `dgraph-io/badger` | Badger's WiscKey (key/value-separated) design wins when values are large (1-100KB+). If profiling shows CodeGraph's stored payloads (source snippets, docstrings) are large enough to dominate write-amplification, Badger becomes competitive. For the graph's dominant workload — small structured records (symbols, edges, file metadata) — Pebble's design and lower read-latency are the better fit, and Pebble's `IndexedBatch` (read-your-writes within a transaction) maps cleanly onto graph-mutation semantics. |
| `cockroachdb/pebble` | `etcd-io/bbolt` | bbolt is simpler and has the longest production track record of any pure-Go embedded store, but its single-writer transaction model is a structural bottleneck for "optimized for concurrent access" — reject for the primary graph store. Still worth considering for small, low-write-volume auxiliary metadata (e.g. per-project config) where simplicity outweighs concurrency needs. |
| `modernc.org/sqlite` (migration reader only) | `mattn/go-sqlite3` | `mattn/go-sqlite3` is faster (it's the CGo baseline the pure-Go driver is benchmarked against) but requires CGO_ENABLED=1 for a component (the migration tool) that only needs to *read* existing TS-CodeGraph SQLite files once per user, not serve production query load — the CGo cross-compilation and static-binary cost isn't worth paying for this narrow, one-shot use case. |
| `spf13/cobra` | `urfave/cli` | urfave/cli has a smaller API surface, faster cold start, and smaller binary footprint in microbenchmarks, and is a reasonable choice for flat, few-command CLIs. CodeGraph's command surface (init/install/uninstall/uninit/upgrade/explore + agent-installer subcommands + migration) is exactly the kind of nested, completion-needing hierarchy Cobra is built for — reject urfave/cli for this project. |
| `anchore/syft` | `CycloneDX/cyclonedx-gomod` | Use cyclonedx-gomod's `app` mode instead of (or in addition to) Syft when you need SBOMs that are precise about which modules actually compiled into *each platform's* binary (it evaluates Go build constraints; Syft's scan is more generic). Reasonable to run both and publish whichever the target ecosystem (SPDX vs CycloneDX consumers) expects. |

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|--------------|
| `smacker/go-tree-sitter` | Community fork bundling all grammars in one repo, coupling grammar updates to the wrapper's release cycle; the tree-sitter org itself points users toward the official bindings as the maintained path forward. | `tree-sitter/go-tree-sitter` + per-language grammar modules |
| `mattn/go-sqlite3` for the new graph store | Requires CGO_ENABLED=1 for the primary storage layer, defeating the single-static-binary / no-CGo goal for the component that matters most (it's not just the migration reader — it's every write on every index run). Also, SQLite's single-writer-lock model works against "optimized for concurrent access." | `cockroachdb/pebble` (new format) + `modernc.org/sqlite` (migration reader only, isolated to a one-shot code path) |
| Hand-rolled recursive directory walking with polling for file watching | fsnotify's kqueue/inotify backends exist specifically to avoid polling's CPU cost and latency at monorepo scale; polling doesn't get better with more files, it gets linearly worse. | `fsnotify.NewWatcher` (or `NewBufferedWatcher` for high-volume trees), with an explicit recursive-add-on-Create loop since fsnotify doesn't recurse automatically |
| Long-lived, manually-managed signing keys for release artifacts | Key custody/leak risk, no attestation of *how* the key was used; doesn't satisfy a "verifiable supply chain" story. | `cosign` keyless signing via GitHub Actions OIDC (`id-token: write`) |
| Skipping `govulncheck` in favor of a generic dependency-list scanner (e.g. `go list -m all` + manual CVE lookups) | Naive dependency scanners flag every CVE in every transitive dependency regardless of reachability, producing enormous false-positive noise that teams learn to ignore. | `govulncheck ./...` — call-graph-aware, only flags vulnerabilities in code paths actually reachable from your program |

## Stack Patterns by Variant

- Migrate the parser layer to WASM grammars run via `wazero`, gaining `CGO_ENABLED=0` end-to-end and crash isolation.
- Keep the parser behind a narrow internal interface (`Parser.Parse([]byte, *Tree) (*Tree, error)`-shaped) from day one specifically so this swap is a backend change, not an architecture change.
- Consider building only the highest-priority languages (Go, Java, C#) via CGo initially and gate lower-priority languages behind the same interface, deferring the wazero decision until it's forced rather than optional.
- Batch incremental updates per file-change debounce window into a single Pebble `Batch`/`IndexedBatch` commit rather than per-symbol writes — this is a usage pattern fix, not a storage-engine swap; only reconsider Badger if batching doesn't resolve it.

## Version Compatibility

| Package A | Compatible With | Notes |
|-----------|------------------|-------|
| `tree-sitter/go-tree-sitter@v0.25.x` | `tree-sitter/tree-sitter-<lang>` grammar modules pinned to matching `v0.2x` lines | Grammar modules version independently; pin each explicitly and re-verify compatibility on bump — the official repo's own compatibility table (in `go.mod`/README) is the source of truth per release, don't assume lockstep versioning across grammars. |
| `cockroachdb/pebble` | Go 1.23+ | Track Pebble's stated minimum Go version at upgrade time; it moves with CockroachDB's own toolchain requirements. |
| `mark3labs/mcp-go` | MCP protocol rev in use by target agent clients (Claude Code, Cursor, etc.) | Verify which protocol revision `mark3labs/mcp-go`'s current release implements against the revision your target agents expect — the MCP spec itself is still moving (2025-11-25 → 2026-07-28 during this research window). |
| `modernc.org/sqlite` | Existing TS-CodeGraph `.codegraph/` SQLite schema (v1.3.x) | Purely a read compatibility concern for the migration tool — confirm the on-disk SQLite file format/pragmas TS CodeGraph writes are readable by this driver's supported SQLite version before finalizing the migration tool design. |

## Sources

- `tree-sitter/go-tree-sitter` (Context7, MEDIUM confidence) — API surface, incremental parsing, CGo wrapping confirmed
- `tree-sitter/go-tree-sitter#16` GitHub issue (web, cross-checked/verified — MEDIUM confidence) — official maintainers' stance on WASM/wazero, directly authoritative on this point
- `ncruces/go-sqlite3` (Context7, MEDIUM confidence) — proof pattern that a large C codebase (SQLite) runs successfully as WASM via wazero in production Go tooling
- `pkg.go.dev/modernc.org/sqlite/benchmark` (Context7, MEDIUM confidence, official benchmark data) — pure-Go vs CGo SQLite performance deltas
- `etcd-io/bbolt`, `cockroachdb/pebble` (Context7, MEDIUM confidence) — official docs, concurrency models, transaction semantics
- `dgraph-io/badger` design docs + community benchmark (web, LOW confidence, cross-referenced against official README claims) — WiscKey tradeoffs
- `modelcontextprotocol/go-sdk` GitHub releases (web, MEDIUM confidence — cross-checked against multiple release notes) — official SDK maturity and protocol version support timeline
- `mark3labs/mcp-go` (Context7, MEDIUM confidence) — stdio server API and adoption
- `fsnotify/fsnotify` (Context7, MEDIUM confidence, official README/docs) — recursive-watch limitations, platform-specific fd/watch limits
- `spf13/cobra` (Context7, MEDIUM confidence) + community comparison sources (web, LOW confidence) — CLI framework comparison
- `goreleaser/goreleaser` (Context7, MEDIUM confidence, official docs) — reproducible build config, SBOM/signing integration
- `sigstore/docs`, `goreleaser/example-supply-chain` (web, LOW confidence but consistent with official sigstore docs) — cosign keyless signing pattern
- `slsa-framework/slsa-github-generator` (web, MEDIUM confidence — official repo README) — SLSA3 Go builder workflow
- `CycloneDX/cyclonedx-gomod` README (web, LOW confidence) — SBOM mode comparison vs Syft
- `pkg.go.dev/golang.org/x/vuln` (Context7, MEDIUM confidence, official docs) — govulncheck usage and CI integration

<!-- GSD:stack-end -->

<!-- GSD:conventions-start source:CONVENTIONS.md -->

## Conventions

Conventions not yet established. Will populate as patterns emerge during development.
<!-- GSD:conventions-end -->

<!-- GSD:architecture-start source:ARCHITECTURE.md -->

## Architecture

Architecture not yet mapped. Follow existing patterns found in the codebase.
<!-- GSD:architecture-end -->

<!-- GSD:skills-start source:skills/ -->

## Project Skills

No project skills found. Add skills to any of: `.claude/skills/`, `.agents/skills/`, `.cursor/skills/`, `.github/skills/`, or `.codex/skills/` with a `SKILL.md` index file.
<!-- GSD:skills-end -->

<!-- GSD:workflow-start source:GSD defaults -->

## GSD Workflow Enforcement

Before using Edit, Write, or other file-changing tools, start work through a GSD command so planning artifacts and execution context stay in sync.

Use these entry points:

- `/gsd-quick` for small fixes, doc updates, and ad-hoc tasks
- `/gsd-debug` for investigation and bug fixing
- `/gsd-execute-phase` for planned phase work

Do not make direct repo edits outside a GSD workflow unless the user explicitly asks to bypass it.
<!-- GSD:workflow-end -->

<!-- GSD:profile-start -->

## Developer Profile

> Profile not yet configured. Run `/gsd-profile-user` to generate your developer profile.
> This section is managed by `generate-claude-profile` -- do not edit manually.
<!-- GSD:profile-end -->
