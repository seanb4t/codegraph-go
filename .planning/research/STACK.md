# Technology Stack

**Project:** CodeGraph Go — Go port of TypeScript CodeGraph
**Researched:** 2026-07-10
**Confidence:** MEDIUM (HIGH on release-engineering tooling; MEDIUM on storage/MCP SDKs; MEDIUM-LOW on the parser strategy, which needs an empirical Phase-1 spike before it can be called HIGH)

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

```bash
# Core
go get github.com/tree-sitter/go-tree-sitter@v0.25
go get github.com/tree-sitter/tree-sitter-go/bindings/go
go get github.com/cockroachdb/pebble@latest
go get github.com/mark3labs/mcp-go@latest
go get github.com/spf13/cobra@latest
go get github.com/fsnotify/fsnotify@latest
go get modernc.org/sqlite@latest   # migration reader only

# Dev / CI tooling (not go.mod deps — install as CLI tools in CI)
go install golang.org/x/vuln/cmd/govulncheck@latest
go install github.com/sigstore/cosign/v3/cmd/cosign@latest
# goreleaser + slsa-github-generator installed as GitHub Actions, not go-installed
```

## The Parser Decision (the central open question)

This is the one recommendation in this document that should be treated as **provisional pending a Phase-1 spike**, not settled fact. Three real options exist; here is what the evidence shows for each.

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

**If the Phase-1 parser spike shows WASM/wazero overhead is acceptable (say, <20% of total indexing wall-clock on real repos) and the build pipeline cost is tractable:**
- Migrate the parser layer to WASM grammars run via `wazero`, gaining `CGO_ENABLED=0` end-to-end and crash isolation.
- Keep the parser behind a narrow internal interface (`Parser.Parse([]byte, *Tree) (*Tree, error)`-shaped) from day one specifically so this swap is a backend change, not an architecture change.

**If cross-compilation friction from CGo tree-sitter proves painful in CI (e.g. flaky `zig cc` toolchain issues across GOOS/GOARCH matrix):**
- Consider building only the highest-priority languages (Go, Java, C#) via CGo initially and gate lower-priority languages behind the same interface, deferring the wazero decision until it's forced rather than optional.

**If monorepo-scale write concurrency benchmarks show Pebble's LSM write-amplification is a problem for the specific symbol/edge update pattern (many small, frequent single-record updates from file-watcher-triggered incremental re-indexing):**
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

**Caveat on the parser-strategy web sources:** several search results returned during this research (e.g. blog posts and repos describing specific wazero/tree-sitter proof-of-concept benchmarks) could not be independently corroborated beyond a single low-confidence source each, and at least a few carried signs of low-quality/synthetic content. Only the directly-verifiable, canonical sources (the official `tree-sitter/go-tree-sitter#16` issue, and `ncruces/go-sqlite3`'s own documented approach) were treated as load-bearing for the Option B recommendation above; the ~2x performance-overhead figure is explicitly flagged as directional pending the Phase-1 spike, not as a verified number to design around.

---
*Stack research for: local-first code knowledge graph / MCP server tool, Go*
*Researched: 2026-07-10*

---

# Addendum: v1.0 Milestone — Human TUI & Parity Stack

**Project:** CodeGraph Go — v1.0 Drop-in Parity & Human UX
**Researched:** 2026-07-14
**Confidence:** MEDIUM-HIGH (versions/import-paths verified directly against tagged `go.mod` files on GitHub, not just search snippets; audit posture is directional, not a ground-truth `govulncheck` run)

**Scope:** Only the NEW stack surface for v1.0 — the Charm TUI, TTY-gating, and git/worktree detection. Everything in the v0.1 sections above (Pebble, tree-sitter, mcp-go, Cobra, fsnotify, sigstore-go, hujson, modernc.org/sqlite) is unchanged and already in `go.mod`.

## Recommended Additions

### Core Technologies

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| `charm.land/bubbletea/v2` | v2.0.8 (stable, released 2026-07-03) | Event-loop framework for the **interactive** screens only — `daemon` picker, `install`/`uninstall` multi-select, `init`/`index`/`sync` progress | The only mature, pure-Go, actively maintained TUI event-loop for Go (Elm-architecture Model-Update-View). **Do NOT use it for one-shot output** (`status`, `files`) — that's lipgloss's job (see below). Each interactive Cobra `RunE` constructs its own `tea.NewProgram(model).Run()` and returns control afterward; bubbletea does not wrap the whole CLI. |
| `charm.land/lipgloss/v2` | v2.0.5 | Declarative styling (colors, borders, layout) for **static, one-shot prints** — `status`, `files`, and any plain agent-facing text that gets a human-facing styled variant | CSS-like styling API; v2's `lipgloss.Println`/`Printf` auto-downsample colors per-stream via `colorprofile`, fixing v1's stdin/stdout TTY-detection bugs (see TTY-gating below). This is the mechanism for "styled when human, plain when piped." |
| `charm.land/bubbles/v2` | v2.1.1 | Pre-built interactive components: `list` (daemon picker, multi-select), `spinner`/`progress` (`init`/`index`/`sync` progress), `textinput` (if any interactive prompts are needed) | Maintained in lockstep with bubbletea/lipgloss v2 by the same org; reimplementing list/spinner/progress from scratch is wasted effort with no upside. |

**Critical: v2 import paths changed to Charm's vanity domain.** All three moved from `github.com/charmbracelet/{bubbletea,lipgloss,bubbles}` to `charm.land/{bubbletea,lipgloss,bubbles}/v2` as part of the v2 release. Verified directly against each repo's tagged `go.mod` (not just docs/blog claims, which lag — pkg.go.dev's own cache still showed a stale `v2.0.0-beta.6` as "latest" for bubbletea at research time; GitHub tags/go.mod are the source of truth here). Use the `charm.land/...` path in new code — the old `github.com/charmbracelet/...` v2 path will not resolve for current tags.

### TTY Detection — Use `colorprofile`, gate interactivity with `golang.org/x/term`

Two distinct decisions need two distinct mechanisms:

1. **"Should this one-shot print use color/styling?"** → `github.com/charmbracelet/colorprofile` (`colorprofile.Detect(os.Stdout, os.Environ())`), which lipgloss v2 uses internally via `lipgloss.Println`/`Printf`. It checks isatty on the *specific stream being written to* (fixing v1's stdout/stdin conflation bug — matters here because MCP writes JSON-RPC to stdout while a human might be watching stderr, or vice versa), plus `NO_COLOR`/`CLICOLOR`/`TERM=dumb`/CI-env conventions. This is already pulled in transitively by lipgloss v2 — do not hand-roll a second isatty check for styling decisions; let lipgloss own it.
2. **"Should this command even attempt to launch an interactive bubbletea program?"** → `golang.org/x/term.IsTerminal(int(os.Stdout.Fd()))` (and stdin, since bubbletea reads keystrokes). **Already an indirect dependency in `go.mod` today** (`golang.org/x/term v0.45.0`, pulled in by sigstore-go) — promoting it to a direct, explicit use costs nothing new in the dependency graph. Gate every interactive command (`daemon` picker, `install`/`uninstall` multi-select) on this check up front: non-TTY → fall back to a flag-driven/scripted path (or a plain-text error telling the user to pass explicit flags) instead of calling `tea.NewProgram(...).Run()`, which would otherwise hang or misbehave when stdin/stdout are pipes. This is exactly the mechanism that keeps `serve --mcp` (stdio JSON-RPC) and golden-parity test output clean — MCP and CI never see ANSI or an interactive prompt.

`github.com/mattn/go-isatty` is also already an indirect dep (v0.0.20, likely via a transitive path) but should NOT be used as a third, competing isatty check — pick `golang.org/x/term` for the boolean gate (stdlib-adjacent, already used this way in the Go ecosystem) and let `colorprofile` own the styling decision. Don't mix three isatty implementations for what is conceptually one policy.

### Git / Worktree Detection — stdlib `os/exec`, no new dependency

The TS reference (`dist/sync/worktree.d.ts`, inspected directly) confirms the exact shape to port: `gitWorktreeRoot(dir)` shells out to `git rev-parse --show-toplevel` (per-worktree root — main checkout and each linked worktree report distinct paths) and `gitCommonDir(dir)` to `git rev-parse --git-common-dir` (the shared `.git` all worktrees of one repo point at — same value across worktrees of one repo, different for a submodule/nested repo). Detection is explicitly best-effort: git unavailable or not-a-repo → report "no mismatch," never fail the command.

**Use `os/exec.Command("git", "rev-parse", ...)` directly — no library needed.** This is two simple subprocess calls with string output, not general git object access. A pure-Go git implementation (`go-git/go-git`) would be strictly worse here: it pulls dozens of transitive dependencies (crypto/ssh, gcfg, sha256-simd, billy filesystem abstraction, etc.) to replace two `exec.Command` calls, works against a real installed `git` the user already trusts and has configured (credential helpers, worktree metadata, hooks), and TS parity is explicitly "shell out to the git binary, degrade gracefully if missing" — reimplementing that in a pure-Go git library is scope creep, not parity.

### Git Hooks (post-commit/merge/checkout)

Also stdlib-only: writing an executable shell script (or a shim invoking `codegraph sync`) into `$(git rev-parse --git-dir)/hooks/post-commit` etc. is `os.WriteFile` + `os.Chmod(..., 0o755)`, using the already-necessary `--git-common-dir`/`--git-dir` resolution above. No hook-management library exists or is warranted for three static shim files.

## Installation

```bash
go get charm.land/bubbletea/v2@v2.0.8
go get charm.land/lipgloss/v2@v2.0.5
go get charm.land/bubbles/v2@v2.1.1
# golang.org/x/term: already an indirect dep; go.mod will promote it to direct
# automatically once code imports it directly — no separate `go get` required,
# but running `go mod tidy` after adding the import is sufficient.
```

No new dependency is needed for git/worktree detection or git hooks (`os/exec`, `os.WriteFile`, `os.Chmod` — stdlib).

## Supply-Chain Impact (quantified against the "minimal audited deps, no new CGo" constraint)

**No new CGo.** The entire Charm ecosystem (bubbletea/lipgloss/bubbles + their terminal-I/O helpers `charmbracelet/x/term`, `charmbracelet/x/termios`, `charmbracelet/x/windows`, `muesli/cancelreader`) is pure Go — terminal raw-mode/ioctl access is done via `golang.org/x/sys` syscalls, not cgo. tree-sitter remains the sole documented CGo exception, unchanged by this addendum.

**New transitive modules pulled in (deduplicated across all three Charm libraries):** approximately 20 new `go.sum` entries — `charm.land/{bubbletea,lipgloss,bubbles}/v2` (3 direct) plus indirect: `charmbracelet/colorprofile`, `charmbracelet/ultraviolet`, `charmbracelet/x/{ansi,term,termios,windows,exp/golden}`, `lucasb-eyer/go-colorful`, `muesli/cancelreader`, `rivo/uniseg`, `mattn/go-runewidth`, `clipperhouse/{displaywidth,uax29/v2,stringish}`, `xo/terminfo`, `aymanbagabas/go-udiff`, `MakeNowJust/heredoc`, `atotto/clipboard`, `charmbracelet/harmonica`, `sahilm/fuzzy`, `kylelemons/godebug`. A handful overlap with what's already in `go.mod` (`golang.org/x/sync`, `golang.org/x/sys`, `dustin/go-humanize` are all already indirect deps from Pebble/sigstore-go, so those specific version constraints just get unified, not net-new). This is a moderate, bounded addition for what a TUI framework needs — no bloated kitchen-sink transitive tree (no HTTP clients, no crypto beyond what's already pulled by sigstore-go, no database drivers).

**Audit posture:** All Charm-org modules (`charmbracelet/*`, and the vanity `charm.land/*` re-exports of the same repos) are actively maintained by a company (Charm) whose tools (`gh`'s `glamour` rendering, `soft-serve`, Kubernetes-adjacent tooling) have wide production adoption — treat as MEDIUM-HIGH confidence, same tier as the v0.1 stack's `mcp-go`/`cobra`. The smaller peripheral libs (`sahilm/fuzzy`, `MakeNowJust/heredoc`, `kylelemons/godebug`, `clipperhouse/*`) are lower-profile single-purpose utilities with thinner track records — LOW-MEDIUM confidence individually, but low-risk given their narrow scope (fuzzy string matching, heredoc string parsing, diff formatting) and the fact they run only in the interactive human-TUI code path, never in the MCP/agent-facing hot path. Run `govulncheck ./...` (already gating CI per the v0.1 release-hardening phase) after adding these — no known blocking CVEs identified during this research, but this addendum does not substitute for that gate.

**One runtime-only caveat, not a build/supply-chain one:** `atotto/clipboard` (pulled in by `bubbles/v2` for components that support copy/paste) shells out to `pbcopy`/`pbpaste` on macOS and `xclip`/`xsel`/`wl-copy` on Linux at runtime if a component's clipboard feature is actually exercised. This does not affect the static-binary build or CGo status — it's an optional runtime `exec.Command` call, exactly analogous to the git shell-outs above — but note it if any interactive component ends up wiring clipboard support, since it can silently no-op on a minimal Linux container without those binaries.

## What NOT to Use (v1.0 addendum)

| Avoid | Why | Use Instead |
|-------|-----|--------------|
| `bubbletea` for `status`/`files`/any one-shot printed output | It's an event-loop framework for *interactive* programs (keystroke handling, a render loop, alt-screen management) — using it to print a static table is architectural overkill and, worse, would force those commands through TTY/raw-mode negotiation they don't need, risking exactly the "hangs when piped" failure mode the parity requirement explicitly rules out. | `lipgloss` styling + a plain `fmt.Println`/`lipgloss.Println` call, gated by `colorprofile`/`x/term` as above |
| `github.com/charmbracelet/bubbletea` (old, non-vanity v2 import path) | Doesn't resolve for current v2 tags — the module was renamed to `charm.land/bubbletea/v2` as part of the v2 release; only the v1 line (now unmaintained-for-new-features) still lives under the old path. | `charm.land/bubbletea/v2` |
| `go-git/go-git` for worktree/common-dir detection | Pure-Go git implementation pulling dozens of transitive deps (SSH, crypto, custom filesystem abstraction) to replace two `git rev-parse` subprocess calls — the TS reference itself just shells out; matching that is parity, reimplementing git is not. | `os/exec.Command("git", "rev-parse", ...)` |
| A second/third isatty library (`mattn/go-isatty` used directly, alongside `colorprofile` and `x/term`) for TTY-gating decisions | Three different isatty mechanisms making three independently-reasoned decisions is a bug magnet — e.g. one code path thinking it's a TTY while another doesn't, producing inconsistent styled/plain output within the same command. | Pick `golang.org/x/term.IsTerminal` for the "launch interactive program?" boolean; let `colorprofile` (already load-bearing inside lipgloss v2) own styling decisions. `mattn/go-isatty` stays an untouched transitive dep, not a direct import. |

## Version Compatibility

| Package A | Compatible With | Notes |
|-----------|------------------|-------|
| `charm.land/bubbletea/v2@v2.0.8` | `charm.land/lipgloss/v2@v2.0.5`, `charm.land/bubbles/v2@v2.1.1` | All three are released and versioned independently by the same Charm org but developed in lockstep for v2; `bubbles/v2`'s own `go.mod` requires `bubbletea/v2` and `lipgloss/v2` directly, so `go mod tidy` will resolve a mutually consistent set — don't hand-pin mismatched majors (v1 lipgloss with v2 bubbletea, etc.), the v1→v2 APIs are not source-compatible. |
| `charm.land/bubbletea/v2` | Go 1.25.0+ (per its own `go.mod`) | Below the project's `go 1.26.5` floor — no conflict. |
| `charm.land/lipgloss/v2` / `charm.land/bubbles/v2` | Go 1.24.2+ | Also below the project floor — no conflict. |
| `golang.org/x/term` (TTY gate) | Already pinned `v0.45.0` in `go.mod` (indirect, via sigstore-go) | No version bump forced by adding a direct import — verify `go mod tidy` doesn't need to raise it once Charm's own `golang.org/x/sys` requirement is reconciled (Charm modules currently request `x/sys` in the `v0.41.0`–`v0.46.0` range depending on which was fetched; Go's MVS will pick the highest, still compatible). |

## Sources

- `charmbracelet/bubbletea` GitHub tags API + raw `go.mod` at tag `v2.0.8` (web, direct GitHub content — HIGH confidence, ground truth over docs/blog claims) — confirmed latest stable version, vanity import path `charm.land/bubbletea/v2`, pure-Go dependency list
- `charmbracelet/lipgloss` raw `go.mod` at tag `v2.0.0` + GitHub tags API (web, HIGH confidence) — confirmed v2.0.5 latest, vanity import path, dependency list
- `charmbracelet/bubbles` raw `go.mod` at tag `v2.0.0` + GitHub tags API (web, HIGH confidence) — confirmed v2.1.1 latest, vanity import path, dependency list
- `charmbracelet/colorprofile` raw `go.mod` (web, HIGH confidence) — confirmed pure-Go, dependency footprint
- Context7 `/websites/pkg_go_dev_github_com_charmbracelet_bubbletea` and `/charmbracelet/{bubbles,lipgloss}` (Context7, MEDIUM confidence — pkg.go.dev's own version cache was stale relative to GitHub tags at research time, so version numbers were cross-verified against raw `go.mod` rather than trusted from Context7/pkg.go.dev alone)
- Lip Gloss v2 discussion #506 "What's New" + GitHub issue #439 (web, MEDIUM confidence, official repo discussion) — confirmed `colorprofile`-based per-stream TTY detection replacing v1's stdin/stdout conflation bug, `lipgloss.Println`/`Printf` auto-downsampling behavior
- `/opt/homebrew/lib/node_modules/@colbymchenry/codegraph/dist/sync/worktree.d.ts` (installed TS CodeGraph v1.3.1 reference, direct inspection — HIGH confidence, this IS the parity target) — confirmed `git rev-parse --show-toplevel`/`--git-common-dir` shell-out approach and best-effort-degrade semantics to port
- Existing project `go.mod` (direct inspection) — confirmed `golang.org/x/term v0.45.0` and `github.com/mattn/go-isatty v0.0.20` are already present as indirect dependencies, zero net-new cost for the TTY-gate boolean check
- `slsa-framework`/`govulncheck` posture: extrapolated from the v0.1 STACK.md's already-established CI gates (Sources section above); no new CVE database query performed for the Charm module set — treat the "no known blocking CVEs" statement as LOW-MEDIUM confidence pending an actual `govulncheck ./...` run once these deps land in `go.mod`

---
*Stack research for: human-facing TUI + git/worktree parity, Go*
*Researched: 2026-07-14*
