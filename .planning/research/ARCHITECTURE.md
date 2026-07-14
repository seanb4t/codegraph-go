# Architecture Research: v1.0 Integration into codegraph-go

**Domain:** CLI + MCP server for a Go static-binary code-knowledge-graph tool (subsequent-milestone integration research, not greenfield)
**Researched:** 2026-07-14
**Confidence:** HIGH (grounded directly in the read v0.1 source — `internal/query`, `internal/mcp`, `internal/cli`, `internal/daemon`, `internal/watch`, `internal/agents`; MEDIUM on git-hooks-in-worktrees mechanics, which is corroborated by general git documentation, not this project's own prior art)

> Supersedes the pre-implementation ARCHITECTURE.md written before v0.1 was built (2026-07-10, general code-graph-indexer ecosystem survey). This document is scoped to v1.0's integration into the ACTUAL shipped v0.1 codebase, not a greenfield design.

## Standard Architecture

### System Overview (v0.1, unchanged skeleton v1.0 bolts onto)

```
┌──────────────────────────────────────────────────────────────────────┐
│  internal/cli (Cobra)          internal/mcp (stdio, mark3labs/mcp-go) │
│  thin commands, plain fmt.Fprint   handlers, plain text/JSON only     │
└───────────────┬─────────────────────────────┬─────────────────────────┘
                │                             │
                └──────────────┬──────────────┘
                               ▼
                  internal/query.Engine (shared, read-only)
              query/search/callers/callees/impact/affected/
              files/status/node/explore — ONE snapshot per call
                               │
                               ▼
                 internal/graphstore.Reader (Pebble snapshot)

  internal/daemon (single-writer lockfile) ──drives──> indexer.Sync
        ▲                                                   ▲
        │ shares lockfile                                   │
  internal/cli `serve --mcp --watch` (in-process fallback)───┘
        │
  internal/watch (fsnotify + debouncer)

  internal/agents (self-registering AgentTarget registry,
                    marker-fenced instruction injection,
                    surgical MCP-config writes)
```

v1.0 adds four things to this picture, all as **siblings**, not rewrites:

1. A **rendering seam** inside `internal/cli` only (TTY-gated lipgloss/bubbletea presenter wrapping the same plain data `internal/query` already returns).
2. Behavioral changes **inside `internal/query.Engine`** (`explore`/`node` relevance/disambiguation) — no new package, shared by construction because CLI and MCP already both call `Engine`.
3. A new **`internal/gitmeta`** package (worktree detection), consumed by `internal/query` (to populate `WorktreeMismatch`/live staleness) and by `internal/cli` (for the pretty status banner) and threaded into MCP tool results as a compact string prefix.
4. A new **`internal/githooks`** package (git hook install/remove), structurally parallel to `internal/agents` but not part of it — different trust boundary, different registry shape (fixed 3 hooks, not 8 targets).

### Component Responsibilities (v1.0 deltas only)

| Component | Responsibility | New/Modified |
|-----------|----------------|--------------|
| `internal/query.Engine` | Add relevance scoring to `Explore`, multi-def disambiguation + `⚠️ no covering tests` to `Node`, multi-word query arity, richer `Status` fields (DB size; nodes-by-kind/languages already scaffolded) | **Modified** — same package, same public methods, richer internals |
| `internal/gitmeta` (new) | Detect `.codegraph/` "borrowed index" (repo root resolved by `ResolveCodegraphDir` is a different worktree than the one the caller is actually in); compute `worktreeMismatch` string and a live "pending changes" signal | **New**, imported by `internal/query` and `internal/cli` |
| `internal/githooks` (new) | Install/remove `post-commit`/`post-merge`/`post-checkout` hook scripts that shell out to `codegraph sync`; idempotent, marker-fenced like `internal/agents` but a separate package/trust boundary | **New**, imported only by `internal/cli` |
| `internal/cli/present` (new, or a `render*.go` cluster inside `internal/cli`) | TTY-gated lipgloss styling + bubbletea interactive pickers, wrapping `query.StatusResult`/`query.FilesResult`/agent registry data | **New**, imported only by `internal/cli` — **never** by `internal/query` or `internal/mcp` |
| `internal/mcp` | Unchanged shape; benefits automatically from `Engine` improvements; gets a short inline worktree-mismatch notice prefixed onto tool results | **Thin modification** (prefix line only) |
| `internal/cli/serve.go` | `--mcp` implies watch by default; add `--no-watch` | **Modified** |
| `internal/daemon`, `internal/watch` | No structural change — `serve --mcp`'s watcher-by-default just flips the existing `--watch` bool's default and wires it through the same `daemon.New`/lockfile path already there | **Unmodified** (one small new exported read, `daemon.Status`, for the interactive picker) |

## Recommended Project Structure (delta from v0.1)

```
internal/
├── query/                    # UNCHANGED package boundary — richer internals
│   ├── explore.go            # + relevance scoring (rankedNode already carries a `tier`; extend)
│   ├── node.go                # + multi-def disambiguation prompt/list, "no covering tests" check
│   ├── status.go              # + DB size (Pebble disk usage); nodes-by-kind/languages already scaffolded
│   ├── resolve.go             # ResolveCodegraphDir stays upward-walk; gitmeta cross-checks it, doesn't replace it
│   └── gitmeta_bridge.go      # NEW: thin glue — Engine.Status()/Explore() calls gitmeta.Detect(repoRoot) and folds
│                               #      the result into StatusResult.WorktreeMismatch / a new blast/status field
├── gitmeta/                   # NEW package
│   ├── worktree.go            # git-common-dir vs git-dir detection, "borrowed index" comparison
│   └── worktree_test.go
├── githooks/                  # NEW package
│   ├── hooks.go                # Install/Remove/Status for post-commit/post-merge/post-checkout
│   ├── scripts.go              # embedded hook-script templates (marker-fenced, like agents/instructions.go)
│   └── hooks_test.go
├── fsatomic/                   # NEW package (extraction from internal/agents/shared.go)
│   ├── fsatomic.go              # atomicWriteFile, replaceOrAppendMarkedSection, removeMarkedSection
│   └── fsatomic_test.go         # moved verbatim from agents/shared_test.go
├── cli/
│   ├── present/                # NEW subpackage (or file cluster) — the ONLY place lipgloss/bubbletea is imported
│   │   ├── style.go             # lipgloss styles, TTY gate (isatty check on cmd.OutOrStdout())
│   │   ├── status_view.go       # pretty status renderer wrapping query.StatusResult
│   │   ├── files_view.go        # pretty tree renderer wrapping query.FilesResult
│   │   ├── daemon_picker.go     # bubbletea daemon-status/attach picker
│   │   ├── install_picker.go    # bubbletea multi-select, REPLACES promptAgentMultiSelect's bufio prompt when TTY
│   │   └── progress.go          # bubbletea progress model for init/index/sync
│   ├── status.go                # calls present.RenderStatus(result) when TTY, else existing plain fmt.Fprintf
│   ├── files.go                 # same TTY branch for --format tree
│   ├── serve.go                 # --mcp implies watch; --no-watch opt-out
│   ├── githooks.go              # NEW: `codegraph githooks install|remove|status` (dedicated command — see Q4 below)
│   └── archtest/                 # NEW: import-graph guard mirroring graphstore/archtest and migrate/archtest
│       └── no_charm_leak_test.go # asserts bubbletea/lipgloss import path only appears under internal/cli(/present)
└── mcp/
    └── tools.go                  # + one-line worktree-mismatch prefix on tool results; otherwise unchanged
```

### Structure Rationale

- **`internal/query` stays the seam of truth.** It already has `WorktreeMismatch *string` and `PendingChanges` fields sitting inert in `StatusResult` (see `internal/query/status.go`'s mapping table) — v1.0's job for worktree awareness is to make those live, not invent a new shape. Both CLI `status` and MCP `codegraph_status` read the same `StatusResult`, so wiring `gitmeta` in at the `Engine` level means CLI and MCP get worktree awareness in the same commit, with zero risk of the two surfaces drifting.
- **`internal/gitmeta` is new, not folded into `internal/query`,** for the same reason `internal/watch` isn't folded into `internal/daemon`: it's a distinct, independently testable concern (parsing `.git`/`git-common-dir` layout) that `internal/query` calls into, mirroring the existing precedent of `internal/query` depending on `internal/graphstore` and `internal/indexer/goextract` (for `RefKindCalls`) — a read-only engine composing narrow, focused packages is the established pattern here, not a monolith.
- **`internal/githooks` is a new top-level package, not a member of `internal/agents`,** because the trust boundary and registry shape are genuinely different (see Pattern 4 below for the detailed argument) — but it deliberately reuses `internal/agents/shared.go`'s already-proven idioms (marker-fenced replace-or-append, atomic temp-file-then-rename writes). **Recommendation: extract `internal/fsatomic`** from `internal/agents/shared.go` — `atomicWriteFile`, `replaceOrAppendMarkedSection`, `removeMarkedSection` — so both `internal/agents` and `internal/githooks` import it, rather than `internal/githooks` importing `internal/agents` directly (a naming/conceptual smell — a git-plumbing package has no business depending on an agent-config package) or duplicating non-trivial, security-relevant (V12 atomic-write) logic a second time. This breaks the codebase's OTHER established precedent — trivial single-constant duplication across packages (`codegraphDirName` duplicated verbatim in `internal/cli`, `internal/query`, `internal/daemon`, each with a comment explaining why) — but deliberately: that precedent is for one-line constants with nothing to drift; these are ~40-line, tested, security-sensitive functions where a second implementation is a real duplication-drift risk, not cosmetic.
- **`internal/cli/present` (or equivalent) is the ONLY package that imports `bubbletea`/`lipgloss`.** This is the direct analog of the `internal/graphstore/archtest` (pebble confinement) and `internal/migrate/archtest` (`modernc.org/sqlite` confinement) precedents already in this codebase — same pattern, new dependency. Add `internal/cli/archtest` with a `go/packages`-based test (structurally identical to `internal/graphstore/archtest/import_graph_test.go`) asserting `github.com/charmbracelet/{bubbletea,lipgloss}` only appears under `internal/cli/...`. This is the load-bearing guarantee behind "MCP never sees ANSI" — not a design intention alone, but a CI-enforced boundary, matching how this codebase already treats its other supply-chain-sensitive imports.

## Architectural Patterns

### Pattern 1: Plain-data-core + TTY-gated presenter (the rendering seam)

**What:** `internal/query.Engine` methods keep returning exactly what they return today — Go structs (`StatusResult`, `FilesResult`) for JSON-shaped commands, and plain markdown strings (`Explore`, `Node`) for the two flagship agent-facing commands. `internal/mcp` is untouched: it marshals structs to JSON or passes markdown strings through verbatim, exactly as it does now (`tools.go`'s `companionHandler`/`exploreHandler` bodies do not change at all). `internal/cli` adds a presentation layer that takes the SAME `StatusResult`/`FilesResult` value the plain-text branch already prints and, when the output is a real terminal, renders a styled version instead.

**When to use:** Any command whose CLI output currently branches on `--json` (`status`, `files`) is exactly where this seam goes — that branch already proves the command has a clean data/presentation split; add a third branch (pretty) alongside the existing plain and `--json` ones, gated on TTY rather than a flag.

**Trade-offs:** Three render paths per command (`--json` / plain / pretty) instead of two is more surface, but it's additive — the existing two paths are untouched code, so this is zero risk to the golden-template tests that already pin `explore`/`node`/`status` shapes. `explore` and `node` themselves get NO pretty path in v1.0 (per the milestone's own scope: "lipgloss-styled `status`/`files`" only) — their markdown output already IS the human-facing format (it's what the golden corpus pins), so there is nothing to prettify without risking exactly the drift the milestone is trying to avoid.

**Example:**
```go
// internal/cli/status.go — the TTY branch added alongside the existing two
result, err := eng.Status()
if err != nil { return err }

if jsonOut {
    data, _ := query.MarshalStatusJSON(result)
    return writeJSONLine(cmd, data)
}

if present.IsInteractive(cmd.OutOrStdout()) {
    return present.RenderStatus(cmd.OutOrStdout(), result) // lipgloss, TTY only
}

// unchanged plain fallback — exactly today's fmt.Fprintf line
fmt.Fprintf(cmd.OutOrStdout(), "backend=%s files=%d ...\n", ...)
```

```go
// internal/cli/present/style.go
// IsInteractive gates ALL styling — piped/redirected output (a script, a
// CI log, an agent's captured stdout) NEVER receives ANSI, matching the
// same isatty-style check install.go already uses for its own TTY gate
// (installStdinIsInteractive) — just applied to stdout instead of stdin.
func IsInteractive(w io.Writer) bool {
    f, ok := w.(*os.File)
    if !ok {
        return false
    }
    fi, err := f.Stat()
    if err != nil {
        return false
    }
    return fi.Mode()&os.ModeCharDevice != 0
}
```

The one thing to get right: `IsInteractive` must check the SAME writer `cmd.OutOrStdout()` returns (which is `os.Stdout` in production, a `bytes.Buffer` in every test — mirroring `installStdinIsInteractive`'s existing `cmd.InOrStdin() != os.Stdin` short-circuit) — not a bare `os.Stdout` global check, or tests that inject a buffer lose the ability to force the plain branch.

### Pattern 2: Shared-engine algorithm changes, never surface-specific logic

**What:** `explore`'s relevance ranking and `node`'s multi-definition disambiguation are algorithmic changes to `internal/query.Engine.Explore`/`Engine.Node` and their private helpers (`matchNodes`, `lexicalMatchTier` in `internal/query/search.go`; `resolveNodeForDetail` in `internal/query/resolve.go`). They must NOT be implemented as a CLI-side post-filter on `Engine`'s output, and must NOT be duplicated in `internal/mcp`.

**When to use:** Always, for this project — `internal/mcp`'s own package doc comment states the invariant explicitly ("one engine, two front-ends, so MCP output shapes cannot drift into two code paths") and `internal/mcp/tools.go`'s handlers already prove it structurally: `exploreHandler`/`companionHandler` call `eng.Explore(...)`/`eng.Node(...)` and pass the result straight to `mcp.NewToolResultText`, with zero local re-rendering.

**Trade-offs:** None, really — this is not an optional design choice given the existing architecture, it's the only choice that doesn't create a second rendering/ranking path. The one real risk is regression against the golden corpus (`testdata/golden/corpus/weft-go/*.json`): `rankedNode.tier` already carries a lexical match tier (`internal/query/search.go`); extending it with a numeric relevance score (rather than replacing the tier concept) keeps ties broken the same deterministic way the golden tests currently pin, and the new `⚠️ no covering tests` warning is an ADDITIVE line appended to existing blast-radius output (`exploreBlast` already carries `TestFiles []string` — a warning is just "when `TestFiles` is empty for a symbol with `CallerCount > 0`, emit the line"), not a reshape of the existing section order the golden tests assert on.

**Example (conceptual — extend, don't replace):**
```go
// internal/query/explore.go — additive: a new field, not a new code path
type exploreBlast struct {
    Symbol      *schema.Node
    CallerCount int
    TestFiles   []string
    // NEW: derived, not stored — computed in buildBlastEntry
}

func (e *Engine) buildBlastEntry(n *schema.Node, rev map[string][]*schema.Edge) (exploreBlast, error) {
    // ...unchanged existing logic...
    bl := exploreBlast{Symbol: n, CallerCount: len(callers), TestFiles: testFiles}
    return bl, nil
}

// render_markdown.go's RenderExplore appends "⚠️ no covering tests" only
// when bl.CallerCount > 0 && len(bl.TestFiles) == 0 — a pure function of
// data already computed, not a new query.
```

### Pattern 3: Narrow-package worktree detection, cross-checked (not replacing) the upward-walk resolver

**What:** `ResolveCodegraphDir` (`internal/query/resolve.go`) walks upward from `start` looking for the nearest `.codegraph/` — this IS the "borrowed index" bug: in a linked worktree with no `.codegraph/` of its own, the walk finds the MAIN worktree's `.codegraph/` and silently queries it. `internal/gitmeta` doesn't change that walk (v1.0 is explicitly TS-parity scoped: "detect+warn+notice", not "auto-init or share" — see PROJECT.md's Key Decisions). It adds a SEPARATE check, run alongside `ResolveCodegraphDir`, that answers "is `start`'s actual git worktree the same one `.codegraph/`'s resolved directory sits in?"

**When to use:** Called once per `Engine` construction path that has a `repoRoot` (i.e. `OpenAt`), threaded into `StatusResult.WorktreeMismatch` (already a `*string` field, currently always `nil`) and into a new lightweight field/prefix `Explore`/`Node`/every MCP tool result can surface.

**Trade-offs:** Git worktree layout has real edge cases (bare repos, submodules, `.git` as a file vs directory) — scope the v1.0 implementation to the common case (walk for a `.git` file containing `gitdir: <path>` vs a `.git` directory, without shelling out to `git rev-parse`) and fail SOFT (treat "can't determine" as "no mismatch detected", never block a query over it) — this is a UX nicety per the milestone scope, not a correctness gate.

**Example:**
```go
// internal/gitmeta/worktree.go
package gitmeta

// WorktreeInfo answers whether repoRoot's resolved .codegraph/ directory
// belongs to a DIFFERENT git worktree than the one `start` is actually in.
type WorktreeInfo struct {
    Mismatch        bool
    MainWorktree    string // the worktree .codegraph/ actually belongs to
    CurrentWorktree string // the worktree `start` is actually in
}

// Detect never errors out to the caller for anything but a genuine I/O
// failure reading .git — an ambiguous/unsupported git layout resolves to
// Mismatch: false (fail soft), matching Status()'s existing degrade-safely
// precedent for computeStale with no repoRoot.
func Detect(codegraphResolvedDir, callerStartDir string) (WorktreeInfo, error)
```

```go
// internal/query/status.go (or a new gitmeta_bridge.go) — Status wiring
info, _ := gitmeta.Detect(e.repoRoot, e.startDir) // startDir: new Engine field, the ORIGINAL start path OpenAt received, before ResolveCodegraphDir's walk
var mismatch *string
if info.Mismatch {
    msg := fmt.Sprintf("querying %s's index from worktree %s", info.MainWorktree, info.CurrentWorktree)
    mismatch = &msg
}
result.WorktreeMismatch = mismatch
```

Note the ONE structural addition this requires to `Engine`/`OpenAt`: today `NewWithRoot(reader, dir)` only stores the RESOLVED `.codegraph/`-containing directory (`dir` — the walk's answer), not the ORIGINAL `start` the caller passed in. `gitmeta.Detect` needs both (resolved dir vs actual caller cwd) to notice a mismatch. `OpenAt(start string)` must thread `start` itself through to the `Engine` (a new unexported field, e.g. `startDir`) alongside the existing `repoRoot` — a small, additive `Engine` struct change, not a signature-breaking one (`OpenAt`'s existing callers in `internal/cli` and `internal/mcp` don't change).

### Pattern 4: Two independent marker-fenced-write registries (agents vs git hooks), sharing low-level plumbing only

**What:** `internal/agents.AgentTarget` is a per-agent interface implemented by 8 concrete types self-registering via `init()`. Git hooks are NOT the same shape: there are exactly 3 fixed hook names (`post-commit`, `post-merge`, `post-checkout`), one script body (a thin shell wrapper invoking `codegraph sync --quiet` against the worktree the hook fires in), and no "detection" concept analogous to `AgentTarget.Detect` (there's no ambiguity about whether git hooks are "installed" — either the marker-fenced block is in `.git/hooks/<name>` or it isn't).

**When to use:** `internal/githooks` as its own package, exposing `Install(repoRoot string) (WriteResult, error)` / `Remove(repoRoot string) (WriteResult, error)` / `Status(repoRoot string) ([]HookStatus, error)` — deliberately mirroring `agents.WriteResult`'s shape (`Files []FileResult`, `Errors []error`, `Notes []string`) so `internal/cli` can reuse `printAgentResults`-style reporting, but NOT implementing `agents.AgentTarget` or registering into `agents.registry` — these are genuinely different write targets (`.git/hooks/*`, executable shell scripts, vs `~/.claude.json`-style JSON/TOML configs) with a different failure mode worth keeping visually distinct (a broken git hook can silently no-op a sync; a broken agent config just means the agent falls back to grep).

**Trade-offs:** Reusing `agents`'s low-level file-write helpers (`atomicWriteFile`, `replaceOrAppendMarkedSection`, `removeMarkedSection`) means either (a) importing `internal/agents` from `internal/githooks` — cheap but creates a dependency edge from a git-plumbing package to an agent-config package that has nothing to do with git — or (b) extracting those three functions into a new `internal/fsatomic` package both import. **Recommend (b)** — it's a pure refactor of existing, already-tested code (`internal/agents/shared_test.go` covers it), low risk, and removes an awkward cross-domain import.

**Critical git-hooks-specific concern the agents pattern does NOT have to deal with:** hooks must not silently clobber a pre-existing hook installed by another tool (Husky, Lefthook, pre-commit, a repo's own custom hook). `replaceOrAppendMarkedSection`'s existing "no markers present → append after existing content" behavior handles this correctly for the common case (a shell script with other content already in it) as long as the git-hook shell scripts codegraph writes are POSIX-sh (not bash-specific) and end each marker-fenced block in a way that never short-circuits the rest of the hook chain — verify this at implementation time; it's the one place git hooks are meaningfully riskier to auto-write than an agent's JSON config (a malformed JSON write fails loud on next parse; a malformed shell hook fails silently and can block `git commit`/`git checkout` entirely for the user). **Recommendation: git hooks in v1.0 are opt-in only** (per PROJECT.md's "opt-in git sync hooks"), never installed by default `codegraph install`, and the install path should detect+warn (not clobber) an existing non-codegraph hook file that has no marker-fenced insertion point available (e.g., a hook that's a symlink to a shared Lefthook binary, not a plain editable script).

**Worktree note (MEDIUM confidence, general git knowledge not this project's own prior art):** git hooks live under `$(git rev-parse --git-common-dir)/hooks/` — the hooks directory is SHARED across all linked worktrees of a repo (not duplicated per-worktree) unless the repo has `core.hooksPath` overridden. This means a single `codegraph githooks install` run from ANY worktree installs the hook for every worktree of that repo — a genuine, if partial, positive side-effect for the worktree-mismatch problem: a `post-checkout` firing in a linked worktree can run `codegraph sync` scoped to `$(pwd)` (the worktree the hook actually fired in) rather than the main worktree's index — but only if that worktree ALSO has its own `.codegraph/` (auto-init across worktrees is explicitly deferred per PROJECT.md, out of scope for v1.0's "detect+warn" bar).

### Pattern 5: Watcher-on-MCP-by-default, mechanically a default flip plus a deference message, not new plumbing

**What:** `internal/cli/serve.go` already has full `--watch` in-process-fallback plumbing (the `if watchMode && hasIndex { ... daemon.New(...); d.Run(watchCtx) ... }` block) — it is complete and already gracefully defers to a live standalone daemon via the shared lockfile (`daemon.ErrLockLive` is caught and logged, not treated as fatal). v1.0's job here is almost entirely a **default-value flip**: `watchMode` currently defaults `false` via `cmd.Flags().BoolVar(&watchMode, "watch", false, ...)`; it needs to default `true` when `--mcp` is set, with a new `--no-watch` flag to opt out.

**When to use:** This is the correct integration point precisely because it's not a new mechanism — the existing `--watch` code path (debounced fsnotify watcher, shared daemon lockfile, `ErrLockLive` graceful-defer) is already correct and tested; only the flag DEFAULT needs to change.

**Trade-offs:** The one real design decision is flag semantics: Cobra `BoolVar` gives every flag one static default, so "watch defaults true only when `--mcp` is set" needs either (a) a tri-state (`--watch`/`--no-watch`/unset) resolved in `RunE` rather than at flag-registration time, or (b) since `--mcp` is ALWAYS required today (`serve` without it already errors: "`--mcp` is required (stdio is the only supported transport)"), simply defaulting the existing bool to `true` unconditionally is equivalent in practice. **Recommend (b)**: flip the literal default in `BoolVar` from `false` to `true`, rename the flag registration to `--no-watch` with an inverted variable (`noWatch bool`, default `false`), and change the guard from `if watchMode && hasIndex` to `if !noWatch && hasIndex`. This is a two-line diff, not a redesign.

**Example:**
```go
// internal/cli/serve.go — the flag-default flip
var noWatch bool
// ...
cmd.Flags().BoolVar(&noWatch, "no-watch", false, "disable the in-process watcher that runs by default alongside --mcp")
// ...
if !noWatch && hasIndex {
    // ...existing watchMode block body, unchanged...
}
```

`install.go` already writes the byte-identical `serve --mcp` invocation into every agent's MCP config (per PROJECT.md's context: "our `install` already writes the byte-identical `serve --mcp` invocation; only the watch default differs") — so this flip requires ZERO changes to `internal/agents` or any agent config-writer; every already-installed agent gets watch-by-default for free the next time the user upgrades the binary, with no re-`install` needed.

### Pattern 6: Interactive TUI stays CLI-local; engine/MCP see only its inputs and outputs

**What:** The daemon picker (interactive attach/status view over `internal/daemon`) and the `install`/`uninstall` multi-select upgrade (replacing `promptAgentMultiSelect`'s bare `bufio`-scanned numeric-list prompt with a bubbletea list model) both live entirely in `internal/cli/present`. They call INTO existing read-only surfaces — a NEW `daemon.Status(repoRoot) (DaemonStatus, error)` read, `agents.AllTargets()`/`agents.DetectAll()` — and never the reverse.

**When to use:** Any new interactive flow added in v1.0.

**Trade-offs:** `internal/daemon` currently has no "is a daemon running, and what is it doing" read API — `Run` is the only entry point, and liveness/staleness logic (`isStale`, `readLock`) is unexported in `lock.go`. The interactive picker needs a read-only status query (pid, started-at, live/stale) WITHOUT acquiring or releasing the lock. **Add `daemon.Status(codegraphDir string) (DaemonStatus, error)`** — a thin new exported function wrapping the existing unexported `readLock`/`isStale`, returned as a small struct (`Running bool`, `PID int`, `StartedAt time.Time`) — this is the one small `internal/daemon` surface addition v1.0's interactive picker needs; everything else (`Unlock`) already exists and is already a safe read-then-conditionally-write op the picker can call directly.

**Example:**
```go
// internal/daemon/lock.go — new small exported read, no behavior change
type DaemonStatus struct {
    Running   bool
    PID       int
    StartedAt time.Time
}

func Status(codegraphDir string) (DaemonStatus, error) {
    info, ok, err := readLock(codegraphDir)
    if err != nil || !ok {
        return DaemonStatus{}, err
    }
    return DaemonStatus{Running: !isStale(info), PID: info.PID, StartedAt: info.StartedAt}, nil
}
```

```go
// internal/cli/present/daemon_picker.go — bubbletea model, CLI-only
type daemonPickerModel struct {
    status daemon.DaemonStatus // plain data in, from internal/daemon — no bubbletea leak the other direction
}
```

## Data Flow

### Explore/Node request flow (unchanged shape, richer internals)

```
CLI `explore <q>`  ──┐
                      ├──> query.Engine.Explore(q, maxFiles)  [ALL relevance/disambiguation logic here]
MCP codegraph_explore─┘         │
                                 ├──> matchNodes (extended ranking)
                                 ├──> buildBlastEntry (+ no-covering-tests check)
                                 └──> RenderExplore (markdown, unchanged section order)
                                          │
                      ┌───────────────────┴──────────────────┐
                      ▼                                       ▼
        CLI: fmt.Fprint(stdout, out)              MCP: mcp.NewToolResultText(out)
        (present/ NOT involved — explore/node      (unchanged — never touches present/)
         stay plain markdown, no pretty path)
```

### Status request flow (the rendering-seam pattern in full)

```
CLI `status`  ──┐
                 ├──> query.Engine.Status()  [gitmeta.Detect wired in here]
MCP codegraph_status─┘        │
                               ▼
                      query.StatusResult{..., WorktreeMismatch: *string, Stale: bool}
                               │
              ┌────────────────┼─────────────────────┐
              ▼                ▼                      ▼
     CLI --json:         CLI plain (piped/CI):    CLI TTY (present.RenderStatus):
     MarshalStatusJSON   fmt.Fprintf (unchanged)   lipgloss table/box, colored
                                                     Stale/WorktreeMismatch banners
              │
              ▼
     MCP: MarshalStatusJSON + a short worktree-mismatch
     text PREFIX line ("⚠ borrowed index: ...\n\n" + json),
     analogous to staleBanner's existing prefix pattern in
     render_markdown.go — same idiom, new field.
```

### Watcher-on-MCP flow

```
codegraph serve --mcp  (no --no-watch)
        │
        ├──> indexer.Sync (one-shot reconcile, unchanged — already there today)
        │
        ├──> daemon.New(repoPath, opts) + d.Run(watchCtx) in a goroutine
        │         │
        │         ├──> acquire(lockfile)  ──X──> ErrLockLive?  ──> log "deferring to it", MCP still serves reads
        │         └──> OK ──> watch.Open + Debouncer ──> indexer.Sync on every debounced flush
        │
        └──> mcp.BuildServer + server.ServeStdio  (blocks; on ctx cancel, watcher goroutine joins before exit)
```

This flow is IDENTICAL to today's `--watch` opt-in path — only the caller's decision to enter the `if` branch changes (default true vs default false).

## Scaling Considerations

Not the primary axis of this milestone (v1.0 is behavioral parity + human UX on the existing single-repo, single-writer-daemon model — team-scale is explicitly deferred per PROJECT.md's "Deferred to later releases"). Noted only where it interacts with v1.0's specific additions:

| Concern | v1.0 (this milestone) | Later (team scale, out of scope here) |
|---------|------------------------|----------------------------------------|
| Worktree count | `gitmeta.Detect` does a lightweight `.git`-file/dir read per call — fine at N worktrees for a human-scale monorepo; no caching needed | A central server milestone would want worktree metadata cached, not re-read per query |
| Git hooks across worktrees | Shared hooks dir means one install covers all worktrees "for free" (see Pattern 4) — a feature of git's own design, not something this project has to build | N/A |
| Watcher-by-default on every `serve --mcp` | One fsnotify watcher per repo-root process; the existing single-writer lockfile already prevents two watchers double-syncing the same repo — unchanged cost profile from today's opt-in `--watch` | CI-distributed index milestone removes the need for per-agent-session watchers entirely |

## Anti-Patterns

### Anti-Pattern 1: Rendering inside `internal/query` or `internal/mcp`

**What people do:** Add an `if isTTY { ... }` branch, or a lipgloss import, directly inside `Engine.Status()`/`RenderExplore` "since it's convenient" or "just for the CLI's benefit."
**Why it's wrong:** `internal/mcp` calls the exact same `Engine` methods and `render_markdown.go` formatters as the CLI (`internal/mcp`'s own package doc says so explicitly). Any styling that leaks into that shared layer reaches MCP's stdout — which IS the JSON-RPC transport itself (`tools.go`'s `WarnUnknownToolsTo` comment: "Diagnostics never go to stdout — stdout is reserved for the MCP JSON-RPC transport"). ANSI escape codes in an MCP tool result are, at best, garbage the agent has to strip and, at worst, a transport-corrupting bug.
**Do this instead:** `internal/query`/`internal/mcp` return/emit plain data or plain text, unconditionally, forever. All conditional styling lives in `internal/cli/present`, gated by `present.IsInteractive`, called only from `internal/cli`'s own command files.

### Anti-Pattern 2: Duplicating relevance/disambiguation logic per surface

**What people do:** Implement `explore`'s new relevance ranking as a CLI-side re-sort of `Engine.Explore`'s markdown output (e.g., regex-parsing the rendered markdown to reorder file blocks), because "the algorithm only needs to affect what the human sees."
**Why it's wrong:** This is exactly the two-code-path drift `internal/mcp`'s package doc warns against, and it's strictly worse than adding the logic to `Engine` directly — post-processing rendered markdown is fragile (couples to exact string shape) AND leaves MCP's output unimproved, defeating the milestone's own stated goal ("v0.1's golden test proved template shape but never the selection/relevance algorithms" — for BOTH surfaces, not just CLI).
**Do this instead:** Every relevance/disambiguation change is a change to `Engine.Explore`/`Engine.Node`'s Go logic (ranking, tie-breaking, warning computation) before rendering — `RenderExplore`/`RenderNode` stay dumb formatters of already-decided data, exactly as they are today.

### Anti-Pattern 3: Auto-installing git hooks by default

**What people do:** Fold git-hook installation into `codegraph install`'s default flow (alongside the 8 agent targets), reasoning "it's just another sync mechanism, why make the user ask for it twice."
**Why it's wrong:** Unlike agent MCP-config edits (JSON files the tool safely round-trips and can cleanly detect/repair), a malformed or interacting-badly-with-another-tool git hook can silently block `git commit`/`git checkout`/`git merge` for the user — a much higher blast radius for a much smaller convenience win, and PROJECT.md's own milestone scope explicitly calls these "opt-in."
**Do this instead:** A separate, explicit command (`codegraph githooks install`) the user must actively choose, with a clear status/remove path, and detect-but-don't-clobber behavior toward any existing non-codegraph hook content.

### Anti-Pattern 4: A `present`-package dependency cycle back into foundational packages

**What people do:** Let the bubbletea daemon-picker model import `internal/daemon` directly for convenience, then later let something in `internal/daemon` (a future feature) reach back into `internal/cli/present` for a status string, creating a cycle.
**Why it's wrong:** Go forbids import cycles outright (compile error), but the softer version — a "just this once" dependency from a foundational package toward a CLI-presentation package — is a design smell that will resurface as a real compile error the moment someone tries to reuse `internal/daemon` from a second entry point (e.g. a future central-server milestone).
**Do this instead:** `internal/cli/present` depends downward on `internal/daemon`, `internal/query`, `internal/agents`, `internal/gitmeta`, `internal/githooks` — never the reverse. Enforce with the same `internal/cli/archtest` import-graph test that also confines bubbletea/lipgloss (one test, two assertions: "no bubbletea/lipgloss outside `internal/cli`" AND "no package under `internal/{query,mcp,daemon,watch,gitmeta,githooks,agents}` imports `internal/cli`").

## Integration Points

### Internal Boundaries (v1.0 deltas)

| Boundary | Communication | Notes |
|----------|---------------|-------|
| `internal/cli` ↔ `internal/query.Engine` | Direct Go calls, same as today | No change — `status`/`files`/`explore`/`node` command files gain a TTY branch calling into `internal/cli/present`, still calling `Engine` first exactly as before |
| `internal/mcp` ↔ `internal/query.Engine` | Direct Go calls, same as today | `Status`/`Explore`/`Node` results carry richer data (worktree, relevance, warnings) automatically — zero `internal/mcp` code changes needed for the algorithm improvements themselves; only the worktree-prefix formatting (a few lines in `tools.go` or a new `internal/mcp` helper) is new |
| `internal/query.Engine` ↔ `internal/gitmeta` | Direct Go calls | New edge. `internal/gitmeta` has NO dependency back on `internal/query` — it takes plain paths in, returns a plain struct, matching every other package `Engine` composes (`graphstore.Reader`, `goextract.RefKindCalls`) |
| `internal/cli` ↔ `internal/githooks` | Direct Go calls | New edge, CLI-only — mirrors `internal/cli` ↔ `internal/agents` |
| `internal/cli/present` ↔ `internal/daemon` | Direct Go calls (new `daemon.Status` read) | New edge, one direction only |
| `internal/cli` (`serve.go`) ↔ `internal/daemon` | Unchanged — same `daemon.New`/`Run`/`ErrLockLive` calls, just gated by `!noWatch` instead of `watchMode` | No new edge |
| `internal/agents` + `internal/githooks` ↔ `internal/fsatomic` (new) | Direct Go calls, replacing today's package-local `internal/agents/shared.go` helpers | Refactor + new capability — needed so `internal/githooks` can reuse the same atomic-write/marker-fence primitives without importing `internal/agents` |

### External Services

None new for v1.0 — no external service calls are introduced by any of the six integration areas. (`internal/upgrade`'s existing GitHub-releases network path is unaffected and out of this research's scope.)

## Suggested Build Order

Front-loads shared-engine behavioral work (highest risk to the golden-template contract and to MCP/CLI parity) before CLI-only polish, per the milestone's own stated priority ("close behavioral gaps... then... human-facing TUI"). Each phase below is independently shippable/testable; later phases depend on earlier ones only where noted.

**Phase A — `explore`/`node` relevance & disambiguation (internal/query only)**
Highest risk (golden-template regression, the core parity gap), zero new dependencies, zero new packages. Touches `internal/query/explore.go`, `search.go`, `resolve.go`, `render_markdown.go`. Both CLI and MCP improve simultaneously by construction — no MCP-specific work needed. Validate against the existing golden corpus plus new fixtures for ambiguous names / multi-word queries (the milestone's own "behavioral parity test harness" requirement) before moving on.

**Phase B — `status` richer content + worktree awareness (internal/query + new internal/gitmeta)**
Depends on Phase A only for shared test-harness infrastructure, not code. Add `internal/gitmeta`, wire it into `Engine.Status`/`OpenAt` (the `startDir` field addition from Pattern 3), add DB-size to `StatusResult` (nodes-by-kind and languages are already computed in `Status()` today — this is mostly surfacing one new Pebble disk-size call plus the worktree field going live). Both CLI plain-text `status` and MCP `codegraph_status` get worktree-mismatch and the DB-size field in the same commit.

**Phase C — Watcher-on-MCP default flip (internal/cli only)**
Small, mechanical (Pattern 5's two-line diff), but sequenced after A/B so it's validated against the now-final `Engine` behavior it's wrapping, not against code that's still changing under it. No new packages.

**Phase D — Output hygiene (Pebble WAL log silencing, stderr discipline)**
Small, can run in parallel with C — no shared surface. Confirm Pebble's logger can be redirected/silenced (a `pebble.Options{Logger: ...}` at store-open time is the likely mechanism — `internal/graphstore` already controls store-open options centrally, so this is a change confined to `internal/graphstore/pebble_store.go`, not a stderr-suppression hack elsewhere).

**Phase E — Git hooks (new internal/githooks, opt-in surface)**
Depends on the `internal/fsatomic` extraction (a small, low-risk prerequisite refactor of `internal/agents/shared.go` — do this extraction as the first step of Phase E, verified by re-running `internal/agents`'s existing test suite unchanged against the extracted functions). Independent of Phases A-D otherwise. CLI surface: `codegraph githooks install|remove|status` as a new top-level command — the cleaner choice over folding into `install`/`init`, since git hooks are a genuinely separate concern from agent MCP-config (different trust boundary per Anti-Pattern 3), and a dedicated command gives `status`/idempotent `remove` a natural home without overloading `install`'s existing `--target`/`--location` flag semantics, which are agent-registry concepts that don't map onto "3 fixed git hook names."

**Phase F — Rendering seam + TTY-gated pretty `status`/`files` (new internal/cli/present, new dep: lipgloss)**
Depends on B (richer `StatusResult` is what's being prettified) and D (silenced Pebble logs matter more once output is styled — noisy WAL lines under a lipgloss box look worse than under plain text). Add `internal/cli/present` with `IsInteractive` + `RenderStatus`/`RenderFiles` (Pattern 1), plus the `internal/cli/archtest` import-graph guard (write the guard test FIRST, red, then add the lipgloss import — TDD the boundary itself). This phase also adds the `lipgloss` dependency to `go.mod`, so it's the natural place to run `govulncheck`/update the SBOM story for the new dep, per the milestone's own "audits the new Charm deps via govulncheck/SBOM" requirement.

**Phase G — Interactive daemon picker + install multi-select (internal/cli/present, new dep: bubbletea)**
Depends on F (same `present` package, same archtest boundary already in place) and on the small `daemon.Status` read addition (Pattern 6). Replace `promptAgentMultiSelect`'s `bufio` prompt with a bubbletea list model — but only when `installStdinIsInteractive` is true; the existing non-interactive/CI fallback path (`agents.ResolveTargetFlag("auto", loc)`) is untouched, so this is additive to `install.go`, not a rewrite of its control flow. Add a `codegraph daemon status` interactive view (or a picker embedded in an existing command) as the bubbletea entry point.

**Phase H — Behavioral parity test harness formalization + surface reconciliation + `v1.0.0` release cut**
Last, by design — this is the milestone's own closing validation + release phase (per-command flag parity audit, `search` stance decision, signed `v1.0.0` tag), depending on every prior phase being complete and stable.

**Why this order:** A/B (shared engine) come first because they're both the highest-parity-value work AND the riskiest to the golden-template contract — get them stable before building CLI-only polish on top of a still-moving target. C/D are small, low-risk, and unblock nothing else, so they can slot in wherever convenient (shown as C/D for narrative clarity, not a hard dependency). E (git hooks) is fully independent of the rendering work and could run in parallel with F/G if resourced separately — it's ordered after D only to reuse the `fsatomic` extraction's validation cycle, not because of a true dependency. F before G because G's archtest boundary and TTY-gate helper are things G reuses, not reinvents. H last because it's the release gate.

## Sources

- Direct read of this repository's v0.1 source (HIGH confidence — primary source, not inferred): `internal/query/{engine,explore,node,status,resolve,search,render_markdown}.go`, `internal/mcp/{server,tools}.go`, `internal/cli/{serve,daemon,status,root,files,explore,install}.go`, `internal/daemon/{daemon,lock}.go`, `internal/watch/watcher.go`, `internal/agents/{registry,shared}.go`, `internal/graphstore/archtest/import_graph_test.go`, `go.mod`, `.planning/PROJECT.md`
- `charmbracelet/lipgloss` (Context7, resolved — library confirmed current/actively maintained; TTY/color-profile detection is a documented lipgloss concern, corroborating the TTY-gate design rather than needing this project to hand-roll ANSI stripping)
- Git hooks general mechanics (web search, MEDIUM confidence — general git documentation, not project-specific prior art): [Git - Git Hooks](https://git-scm.com/book/en/v2/Customizing-Git-Git-Hooks), [Git - githooks Documentation](https://git-scm.com/docs/githooks) — confirms post-commit/post-merge/post-checkout semantics and the shared-hooks-directory-across-worktrees behavior (`git-common-dir`) this research's Pattern 4 relies on

---
*Architecture research for: codegraph-go v1.0 milestone integration*
*Researched: 2026-07-14*
