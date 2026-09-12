# Stack Research — v0.13.0 "Guard Hardening & UI Follow-through"

**Domain:** Additions for 7 new capabilities on an existing Go 1.26.6 / CGo-tree-sitter / Pebble / ConnectRPC + Svelte 5 codebase
**Researched:** 2026-09-08
**Confidence:** HIGH (6 of 7 capabilities verified directly against this repo's own source and precedent; 1 — editor URI schemes — is MEDIUM/LOW per-scheme, documented below)

## Headline finding

**Five of the seven capabilities need zero new `go.mod`/`package.json` entries.** This repo already carries, in `go.mod`, every dependency the guard-hardening half of the milestone needs (`golang.org/x/tools/go/packages` for archtests, `go.yaml.in/yaml/v3` for workflow-YAML parsing, `github.com/spf13/cobra` for doc generation), and the tmux harness and IntersectionObserver breadcrumb are both thin wrappers over things the OS/browser already provide. The only two genuine new dependencies are `gonum.org/v1/gonum` (Go, for community detection) and nothing at all on the JS side — the editor-handoff and breadcrumb items are plain browser APIs.

## Recommended Stack

### Core Technologies (new additions only)

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| `gonum.org/v1/gonum` (subpackage `graph/community`) | v0.17.0 (2025-12-29, latest) | Louvain community detection for GRF-06's file/package graph clustering | The only actively-maintained pure-Go graph library with Louvain built in (`community.Modularize`, undirected *and* directed variants — this project's file/package graph is directed by edge-kind). Verified via `pkg.go.dev/gonum.org/v1/gonum/graph/community?tab=imports`: it imports only sibling `gonum.org/v1/gonum/graph/*` and `internal/order` packages — **no BLAS/LAPACK, no cgo, no external dependency** despite gonum's reputation as a numerics-heavy module. High source reputation (Context7), single well-known Go-numerics org, actively released through Dec 2025. |
| tmux (external CLI, NOT a Go module) | whatever `apt-get install tmux` / Homebrew ships | 999.2's real-PTY harness driver | See "The tmux Harness Decision" below — this is deliberately NOT a Go library dependency. |

Nothing else is a new *core* technology. The remaining five capabilities reuse existing dependencies (see below).

### Reused-from-existing-stack (the zero-new-dependency wins)

| Capability | Existing dependency reused | Confidence | Verified how |
|---|---|---|---|
| #5 Import-direction archtest (T-01-18) | `golang.org/x/tools/go/packages` (already a direct `go.mod` require, v0.48.0) | HIGH | `internal/graphstore/archtest/import_graph_test.go` is the **exact precedent already in this repo** — it loads the whole module's import graph via `packages.Load` with `Mode: packages.NeedImports\|NeedName\|NeedDeps, Tests: true` and asserts no package outside an allowed prefix imports a forbidden package, including a positive-control "sanity check that this test can actually detect a real importer." T-01-18's mitigation (`go list -deps ./internal/query` must not contain `connectrpc.com/connect` or `internal/uiproto`) is a second instance of this identical pattern, just against `internal/query` instead of `internal/graphstore`. |
| #6 Cobra doc generation | `github.com/spf13/cobra` (already a direct require, v1.10.2) | HIGH | `spf13/cobra/doc` is a **subpackage of the module already vendored** — `github.com/spf13/cobra/doc`, not a separate `go.mod` entry. `doc.GenMarkdownTreeCustom` walks the full command tree Cobra already builds (`newRootCmd()`, 23 top-level commands in `internal/cli/root.go`) and can drive either the actual doc generation or, per the milestone's ask ("hand-authored + structural test"), a **drift guard**: generate into a temp dir, diff flag names/shorthands extracted from the generated output against `docs/CLI-REFERENCE.md`'s own prose. |
| #7 Workflow YAML `if:` guard | `go.yaml.in/yaml/v3` (already a direct require, v3.0.4) | HIGH | Already the in-tree YAML library for this exact class of test. `internal/upgrade/bench_workflow_shape_test.go` already declares `type benchJob struct { ... If string \`yaml:"if"\` ... }` and unmarshals real `.github/workflows/*.yml` off disk with `yaml.Unmarshal`. `internal/upgrade/proto_task_test.go`'s `workflowFileYAML`/`workflowJobYAML` types plus `TestProtoDriftTaskIsInvokedByCI` are the closest existing pattern for "parse the real on-disk workflow, walk jobs by ID, assert a shape." The new guard for `post-release-verify.yml`'s event-aware conclusion `if:` is the same shape with a new typed struct field, not a new tool. |
| #4 "where am I" breadcrumb (BRW-10) | Native `IntersectionObserver` Web API (zero package) | HIGH | Svelte 5's `$effect` rune is the idiomatic place to attach/detach an observer (`$effect(() => { const io = new IntersectionObserver(cb, opts); io.observe(el); return () => io.disconnect(); })`), exactly the pattern `svelte-motion`'s `useInView` and community actions like `svelte-5-inview` wrap — but those wrappers add ~1-2KB and a `package.json` line for what is a ~20-line `use:` action. `SourcePane.svelte` (the long source view, already in `web/src/lib/components/`) is the mount point: observe line/symbol boundary markers, keep the top-most intersecting entry as the breadcrumb's current symbol. |
| #3 editor handoff (BRW-11) | Native `<a href>` / `window.location.assign()` (zero package) | MEDIUM (varies by editor — see below) | Opening a custom URI scheme from a loopback web page is a plain anchor-tag or `location.assign(uri)` call; no JS library exists or is needed for this, and adding one (e.g. a "deep link" npm package) would be supply-chain weight for a one-line `window.open`/`<a>` navigation the browser already handles, including its own external-protocol confirmation prompt (that prompt IS the safety mechanism — do not try to suppress or route around it). |
| #1 tmux real-PTY harness | `os/exec` (stdlib) driving the `tmux` binary directly | HIGH | See dedicated section below. |

## The tmux Harness Decision (999.2)

**Recommendation: `os/exec` wrapping the `tmux` CLI directly. Do NOT add a Go tmux client library.**

- There is no actively-maintained, widely-adopted Go equivalent of Python's `libtmux` (which itself just wraps the `tmux` CLI via subprocess calls — it is not a native tmux protocol client either). The one real-world Go precedent found (`steipete/tmuxwatch`) is "a thin wrapper over the tmux binary for snapshot capture, capture-pane, send-keys, kill-session" — i.e., it does exactly what a hand-rolled `exec.Command("tmux", "send-keys", ...)` helper does, with no abstraction worth a `go.mod` entry.
- tmux's own control surface (`send-keys`, `capture-pane -p`, `capture-pane -e` for escape sequences, `list-panes`) is already a stable, scriptable CLI contract; wrapping it directly keeps the harness auditable in the same `os/exec`-based style the project already uses for `git`, `brew`, and `task` interop elsewhere.
- **Build-tag gating** (the milestone's own requirement — "skip cleanly without tmux") is a `//go:build tmux_e2e` tag plus a runtime `exec.LookPath("tmux")` early-skip inside the test, mirroring how `internal/upgrade`'s shape tests already skip cleanly when their target file is absent, rather than failing.
- **CI availability is not guaranteed by default** — GitHub's `ubuntu-latest`/`ubuntu-24.04` hosted runner images are not confirmed via this research to ship tmux preinstalled (the `actions/runner-images` README lists were not conclusively checked). Do not assume it is present; add an explicit `apt-get install -y tmux` step (or `brew install tmux` if the job runs on `macos-latest`) ahead of the harness job, which is cheap and idempotent and removes the dependency on an unverified assumption. This is a CI workflow change, not a `go.mod` change.
- **What NOT to add:** no `github.com/gdamore/tcell` or similar terminal-emulation library — the point of this harness is to prove behavior in a *real* PTY via a *real* `tmux`, not to re-simulate one in-process (that would just be the existing headless test suite with extra steps).

## What NOT to Add (per capability)

| Avoid | Why | Use Instead |
|-------|-----|--------------|
| `github.com/OpenPeeDeeP/depguard` or `fe3dback/go-arch-lint` or `arch-go/arch-go` as a **`go.mod` import** | All three are standalone binaries/golangci-lint plugins meant to be invoked externally, not imported into application code — pulling one into `go.mod` would add supply-chain weight for zero runtime benefit, and the milestone's own T-01-18 mitigation is already specified as `go list -deps`-shaped, which this repo has already built the identical pattern for (`import_graph_test.go`). `depguard` IS worth enabling as a **golangci-lint linter entry** (bundled inside the already-external `golangci-lint` CI binary — see `.golangci.yml`'s `# enabled-linters: N` convention) as a *belt-and-suspenders* layer alongside the archtest, but the archtest is the primary, demonstrated-RED guard the milestone rule (`84d1gfpywd`) requires. |
| A Go tmux client library (`gotmux`, similar) | Thin, low-adoption, solves nothing `os/exec` + the stable tmux CLI doesn't already solve. | `os/exec.Command("tmux", ...)` |
| `svelte-intersection-observer`, `svelte-5-inview`, `svelte-motion`'s `useInView`, or any npm scroll-spy package | Each wraps ~15-20 lines of native `IntersectionObserver` + Svelte 5 `$effect` in a `package.json` dependency; `pnpm audit`'s JS vulnerability gate and the drift-guard convention this project already runs (`web:components:drift`) both get one more surface to watch for a trivial win. | A hand-written `use:inView` Svelte action colocated with `SourcePane.svelte`. |
| Any "deep link" / "protocol handler" JS npm package for BRW-11 | Opening `vscode://…` etc. from a webpage is a native anchor/`location.assign` call; no abstraction is needed and none of the popular candidates add meaningful safety (the browser's own external-protocol confirmation IS the safety boundary). | `<a href={editorUri}>` or `window.location.assign(editorUri)`, built from a configurable scheme template. |
| A generic C/C++ graph library via cgo (e.g. igraph bindings) for GRF-06 | Breaks the CGo-tree-sitter-only exception this project already treats as a justified special case — a second cgo dependency for a feature this small is not defensible. | `gonum.org/v1/gonum/graph/community` (pure Go, verified above). |
| `github.com/mattn/go-sqlite3`-style CGo YAML/TOML parsers, or a second YAML library (`gopkg.in/yaml.v3` directly, `sigs.k8s.io/yaml`) for #7 | `go.yaml.in/yaml/v3` is already in `go.mod` and already used for this exact class of test — a second YAML library would be a straight duplicate. | `go.yaml.in/yaml/v3` |

## Editor URI Schemes — what's actually accepted in 2026 (BRW-11 detail)

Verified with MEDIUM-to-LOW confidence per scheme (community/undocumented sources; VS Code's own is the most solid):

| Editor | Scheme | Format | Confidence |
|---|---|---|---|
| VS Code | `vscode://file/{absolute-path}:{line}:{col}` | Official, stable, documented via VS Code's own URI handler. | HIGH |
| Cursor (VS Code fork) | `cursor://file/{absolute-path}:{line}:{col}` | Same shape as VS Code (Cursor forks VS Code's URI handler registration); confirmed via community usage, not first-party docs. | MEDIUM |
| VSCodium | `vscodium://file/{absolute-path}:{line}:{col}` | Same VS Code fork lineage. | MEDIUM |
| JetBrains IDEs (IntelliJ IDEA, PyCharm, WebStorm, GoLand, etc.) | `jetbrains://<ide>/navigate/reference?project={name}&path={path}:{line}:{col}` (also seen as `idea://open?file={path}&line={line}` for older/direct-IDE registration) | Routed through JetBrains Toolbox; **JetBrains' own community forum and a YouTrack ticket (`TBX-3965`) confirm this remains undocumented officially** — works in practice, no stability guarantee. | LOW-MEDIUM |
| Zed | `zed://` scheme exists (`cli::RegisterZedScheme` action registers it) but **file+line open-via-URL is an open feature request as of the most recent research** (`zed-industries/zed#8482`, unresolved). Zed's CLI (`zed {path}:{line}`) works locally but is not the same as a URI scheme a webpage can navigate to. | LOW — do not promise Zed support; degrade gracefully. |

**Implication for the config surface:** BRW-11's "configurable URI scheme" should be a **template string** (e.g. `{scheme}://file/{path}:{line}:{col}` with the JetBrains shape as a documented alternate template, not a hardcoded per-editor enum) — the schemes are similar-shaped but not identical, and Zed's is not reliably available at all. Default to the VS Code/Cursor/VSCodium shape (the one with real documentation) and let users override the template for anything else, including JetBrains' `navigate/reference` query-param shape.

## Integration Cost

| Capability | `go.mod` / `package.json` diff | Build/CI diff | Notes |
|---|---|---|---|
| #1 tmux harness | None | New CI job (or job step) installing `tmux`; new build tag `tmux_e2e` | Isolated package under e.g. `internal/e2e/tmux/`, mirrors `internal/upgrade`'s skip-cleanly convention |
| #2 GRF-06 clustering | `+1` line: `gonum.org/v1/gonum v0.17.0` (direct) | None beyond normal `go build`/`govulncheck` | Server-side only (`internal/uiserver` or `internal/query`), no wire-schema change needed beyond the annotation space the schema already reserves |
| #3 editor handoff | None | None | Pure `web/src` change; config surface for the URI template lives wherever other UI config already lives |
| #4 breadcrumb | None | None | Pure `web/src` change inside `SourcePane.svelte` |
| #5 archtest | None | None (test-only) | New `_test.go` file, same package shape as `import_graph_test.go` |
| #6 CLI reference + drift guard | None | None | New `docs/CLI-REFERENCE.md` + new `_test.go` importing `github.com/spf13/cobra/doc` (already-vendored subpackage) |
| #7 workflow `if:` guard | None | None (test-only) | Extends existing `internal/upgrade` shape-test file(s) |

**Total new `go.mod` requires: 1** (`gonum.org/v1/gonum`). **Total new `package.json` entries: 0.**

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|--------------------------|
| `gonum.org/v1/gonum/graph/community` (Louvain) | A hand-rolled label-propagation implementation (simpler algorithm, ~100 lines, zero dependency) | If gonum's module-level supply-chain surface (it's a large multi-purpose numerics module, even though only a lean subpackage is imported) is judged unacceptable at review time, label propagation is a legitimate zero-dependency fallback — noisier/less stable communities than Louvain, but "good enough" for a directory-structure-substitute clustering view. Keep this as the documented fallback if `gonum` is rejected in requirements review. |
| `os/exec` + `tmux` CLI | A PTY-only harness using `github.com/creack/pty` (already a common Go PTY library) without tmux at all | If the goal were only "run the binary under a real PTY" without multiplexer semantics (panes, alt-screen restore across a detach/attach cycle), `creack/pty` is lighter. The milestone explicitly asks for tmux-specific behavior (send-keys, capture-pane, alt-screen enter/restore, DECRQM), so tmux itself is the right tool, not a substitute. |
| `spf13/cobra/doc` for the CLI reference drift guard | A fully hand-maintained `docs/CLI-REFERENCE.md` with no generation step at all, just the drift *test* comparing flag introspection (via `cmd.Flags()`) against markdown text | This is actually closer to what the milestone asks for ("self-authored CLI reference doc ... with a drift guard") — `cobra/doc` generates prose in Cobra's own voice, which may not match "self-authored." Recommend using `cobra/doc`'s generation *only* to produce the comparison ground-truth inside the test (walk `cmd.Flags()` directly is equivalent and arguably simpler than diffing generated markdown), and hand-author the prose in `docs/CLI-REFERENCE.md` separately. Either way, zero new `go.mod` entries — `cmd.Flags()` introspection needs no `doc` subpackage import at all, just `cobra.Command` (already used everywhere in `internal/cli`). |

## Version Compatibility

| Package | Compatible With | Notes |
|---|---|---|
| `gonum.org/v1/gonum@v0.17.0` | Go 1.21+ (gonum tracks recent Go; this project is on 1.26.6) | No known compatibility constraint against this project's other dependencies — `graph/community`'s import set (verified above) shares no packages with any existing `go.mod` entry, so there is no version-skew risk to reconcile. |
| `go.yaml.in/yaml/v3@v3.0.4` (existing) | Already pinned; no bump needed for #7 | Confirmed the existing pin already supports the typed-struct-with-`yaml:"if"`-tag pattern the guard needs (`bench_workflow_shape_test.go` already does this). |
| `github.com/spf13/cobra@v1.10.2` (existing) | `cobra/doc` subpackage ships in lockstep with `cobra` itself — no separate version to track | No bump needed; the `doc` subpackage has been part of the `spf13/cobra` module for years. |

## Sources

- `internal/graphstore/archtest/import_graph_test.go` (this repo, direct read) — the existing archtest pattern for #5, HIGH confidence (it's the actual code)
- `internal/upgrade/bench_workflow_shape_test.go`, `internal/upgrade/proto_task_test.go`, `internal/upgrade/release_workflow_shape_test.go` (this repo, direct read) — existing YAML-parsing and workflow-shape-test conventions for #7, HIGH confidence
- `internal/cli/root.go` (this repo, direct read) — Cobra command tree shape for #6, HIGH confidence
- `go.mod` (this repo, direct read) — confirms `golang.org/x/tools v0.48.0`, `go.yaml.in/yaml/v3 v3.0.4`, `github.com/spf13/cobra v1.10.2` are already direct requires, HIGH confidence
- Context7 `/gonum/gonum` (Louvain algorithm docs, `graph/community` API surface) — MEDIUM-HIGH confidence, official repo docs via Context7
- `pkg.go.dev/gonum.org/v1/gonum/graph/community` (web, import graph verification) — HIGH confidence, first-party Go module index
- `proxy.golang.org/gonum.org/v1/gonum/@latest` (web, version verification) — HIGH confidence, authoritative Go module proxy, confirms v0.17.0 as latest (2025-12-29)
- `pkg.go.dev/github.com/spf13/cobra/doc`, `github.com/spf13/cobra` GitHub `doc/md_docs.go` (web) — MEDIUM confidence, official repo + pkg.go.dev
- `steipete/tmuxwatch`, `tmux-python/libtmux` docs (web) — LOW-MEDIUM confidence, community precedent for the "thin CLI wrapper, no client library" tmux automation pattern
- `zed-industries/zed#8482` GitHub issue (web) — MEDIUM confidence, first-party issue tracker, confirms Zed file-open-via-URL is NOT yet shipped
- VS Code URI handler community docs, JetBrains community forum + YouTrack `TBX-3965` (web) — MEDIUM (VS Code) / LOW (JetBrains, explicitly undocumented) confidence
- `actions/runner-images` (web, inconclusive) — could not confirm tmux preinstalled on GitHub-hosted Ubuntu runners; treated as absent-by-default in the recommendation
- Community Svelte 5 IntersectionObserver patterns (`svelte-motion`, `svelte-5-inview`, community blog posts) (web) — MEDIUM confidence, confirms `$effect`-based native `IntersectionObserver` is the idiomatic Svelte 5 pattern

---
*Stack research for: v0.13.0 Guard Hardening & UI Follow-through*
*Researched: 2026-09-08*
