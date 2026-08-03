# Phase 2: status Content & Git/Worktree Awareness - Research

**Researched:** 2026-07-15
**Extended:** 2026-07-15 — scope change: SURF-06 (MCP markdown output) pulled in from Phase 8; D-12/D-13 corrected upstream using this research's findings
**Domain:** Go stdlib `os/exec` git introspection, Pebble on-disk store, MCP/CLI shared-engine rendering
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**D-01 (★ TS ground truth available):** The TS 1.3.1 white-box source is
intact and MUST be read for verbatim constants/strings. The top-level
`…/@colbymchenry/codegraph/dist/` holds only `.d.ts` stubs, but the real
implementation was relocated into
`…/@colbymchenry/codegraph/node_modules/@colbymchenry/codegraph-darwin-arm64/lib/dist/`
(195 `.js` files). Phase 2 does white-box constant extraction, not frozen-
golden archaeology.

**D-02 (Worktree detection algorithm):** Port TS `sync/worktree.js`'s 4-gate
cascade verbatim into `internal/gitmeta`. Gate 1: no `worktreeRoot` (not a
repo) ⇒ no mismatch. Gate 2: `worktreeRoot == resolvedIndexRoot` ⇒ no
mismatch. Gate 3: `gitWorktreeRoot(resolvedIndexRoot) != resolvedIndexRoot`
⇒ no mismatch (index root must itself be a working-tree root). Gate 4:
`worktreeCommon && indexCommon && worktreeCommon != indexCommon` ⇒ no
mismatch (submodule/embedded clone suppression — note the polarity:
differing common dirs SUPPRESS, they do not trigger). Otherwise ⇒ mismatch
`{worktreeRoot, indexRoot}`.

**D-03 (Git invocation contract):** `os/exec` only (stdlib, no go-git);
`git rev-parse --show-toplevel` and `git rev-parse --git-common-dir`; `cwd`
= the probed dir; 5s timeout via `exec.CommandContext`; stderr discarded;
trimmed stdout; any error/empty ⇒ null. `--git-common-dir` is relative to
cwd unless already absolute — resolve against the probed dir before
realpath. `EvalSymlinks` both sides, falling back to the plain absolute path
on `EvalSymlinks` error.

**D-04 (`internal/gitmeta` package):** Exposes `WorktreeRoot`, `CommonDir`,
`DetectIndexMismatch`, free of query/render concerns so Phase 5's git sync
hooks can reuse it. New package.

**D-05 (`filesByLanguage` is a NEW computation):** `StatusResult.Languages`
is currently a bare `[]string` with no counts — true parity requires adding
`FilesByLanguage map[string]int64`, computed from the file scan, with
`Languages []string` re-derived from it (count > 0). This is an extra
scan/aggregation, size the task accordingly.

**D-06 (STAT-03 largely already satisfied):** `stale` is already live
(`.codegraph/.sync-pending` sidecar OR mtime > `last_sync_unix_ms`), and
`index.reindexRecommended` is already derived
(`!schema.IsCurrentSchemaVersion(meta)`). Remaining work: surface them in
the new sectioned layout + render TS's pending/up-to-date/reindex advisory
lines, plus verify reachability on BOTH CLI and MCP. `PendingChanges` stays
the inert all-zero placeholder (explicit REQUIREMENTS out-of-scope row);
render the advisory from `stale`, not from a count.

**D-07 (`dbSizeBytes`):** Recursive byte sum of the Pebble store dir
(`.codegraph/store/` — SSTables + WAL + MANIFEST) via `filepath.WalkDir`,
best-effort (skip unreadable entries). Key name `dbSizeBytes` (raw bytes) in
`--json`; human output renders `(bytes/1024/1024).toFixed(2)` MB.

**D-08 (Golden strip reversal, Go side only):** Keep the TS golden's strip
(dbSizeBytes stays stripped in the frozen TS oracle fixtures). Assert Go's
`dbSizeBytes` as a documented allowed divergence: key present, integer,
`> 0`, MB-rendering well-formed (`^\d+\.\d{2} MB$`). Do not assert cross-run
byte stability. Check whether `golden_test.go`'s "no volatile field"
invariant needs a narrowly-scoped exemption; update
`testdata/golden/README.md`'s table to record the reversal + rationale.

**D-09 (Status layout — adopt TS's sections now):** Adopt TS's sectioned
plain-text layout now (content + section headers + ordering + wording),
leaving only ANSI color for Phase 6. Target structure: `CodeGraph Status` /
`Project:` / [verbose worktree warning] / [advisories] / `Index
Statistics:` (Files/Nodes/Edges/DB Size/Backend) / `Nodes by Kind:` /
`Files by Language:` / [Pending Changes | up to date] / [reindex advisory].
Both breakdowns: filter `count > 0`, sort by count DESC, `padEnd(15)` on the
key. `Journal:` dropped (no Pebble analog). `Backend:` is `pebble`, not
TS's `node:sqlite`.

**D-10 (`formatNumber` — hand-roll, no dep):** TS's `toLocaleString()` is
locale-dependent; implement a fixed en-US-style comma grouper in
`internal/query`. Do not pull in `golang.org/x/text/message`. Documented
divergence: pin en-US grouping where TS follows host locale.

**D-11 (Notice strings, ported verbatim):** Verbose (`status` only) —
`worktreeMismatchWarning`: "This CodeGraph index belongs to a different git
working tree.\n  Running in: <worktreeRoot>\n  Index from:
<indexRoot>\nResults reflect that tree's code (often a different branch),
not this worktree — symbols changed only here are missing. Run "codegraph
init -i" in this worktree for a worktree-local index." Compact (every other
read tool) — `worktreeMismatchNotice`: "⚠ CodeGraph results below come from
a different git worktree (<indexRoot>), not where you're working
(<worktreeRoot>) — they may reflect another branch, and symbols changed
only here are missing. Run "codegraph init -i" here for a worktree-local
index." The glyph is U+26A0 `⚠` (bytes `e2 9a a0`), NO U+FE0F variation
selector — distinct from Phase 1's `⚠️`.

**D-12 (Notice delivery — ONE uniform text-prefix across all 7 non-status
read tools) — ★ CORRECTED 2026-07-15 post-research:** TS's
`withWorktreeNotice` prefixes `${notice}\n\n${first.text}` onto the first
text content block, no-ops on `isError` results, and excludes
`codegraph_status` (which embeds its own verbose form). Mirror
`staleBanner`'s prepend-to-rendered-string pattern via a new
`worktreeNotice(mismatch)`, applying the **literal TS text-prefix** to all 7
non-status read tools
(`explore`/`node`/`search`/`callers`/`callees`/`impact`/`files`), with
`status` taking the verbose form. MCP `status` wraps the verbose warning as
a blockquote (`> ⚠ ` + warning with `\n` → `\n> `). TS's CLI `warn()` writes
to `console.log` = STDOUT, not stderr — match that for CLI parity, while
keeping MCP's JSON-RPC stdout clean. **The proposed `_worktreeNotice`
JSON-field hybrid is WITHDRAWN** — this research's Pitfall 1 established the
premise was false (MCP text content is model-facing, never unmarshaled by
any consumer in this repo or by Claude Code), and D-16 moves those 5 tools
to markdown regardless, making the question moot. Do not reintroduce a
per-tool notice mechanism.

**D-13 (Detection cached once, negative results included — cache MUST NOT
live only on `Engine`) — ★ CORRECTED 2026-07-15 post-research:** TS caches
per `${startPath}\u0000${indexRoot}` and holds the first verdict until
restart (#926), because detection costs git subprocesses and MCP is
long-lived. The original "resolve lazily-once (`sync.Once`) on the `Engine`
… MCP gets the win" is **FALSE** — `openEngine` builds a fresh `Engine` per
tool call, so an Engine-scoped cache yields zero cross-call benefit on the
exact surface the cache exists for (this research's Corrections #1).
**Corrected decision:** put the cache in `internal/gitmeta` itself (a
`CachingDetector` — mutex-guarded `map[string]*Mismatch` + a
`Detect(startPath, indexRoot) *Mismatch` method); `internal/mcp` constructs
one per server and closes over it in every handler; the CLI constructs one
per invocation. Cache negative results too (nil == "checked, no mismatch",
distinguishable from "not yet checked").

**D-16 (SURF-06 — the 5 JSON-shaped MCP read tools switch to markdown):**
`callers`/`callees`/`impact`/`search`/`files` move from raw `json.Marshal`
output to markdown. **The CLI `--json` flag is UNTOUCHED and keeps emitting
JSON.** Rationale: (1) the consumer is a language model, not a parser —
nothing unmarshals MCP text content; (2) measured token cost on this repo's
index: `files` = 28,506 B JSON vs ~16,835 B markdown (**-41%**), ~14 KB of
which is 4 keys repeated across 308 records; (3) it CLOSES a TS divergence
(TS returns markdown from every MCP tool); (4) it dissolves the D-12 notice
problem — one shape, one mechanism, no handler touched twice. **Rejected (do
not revisit without new evidence):** TOON (13,740 B, -52%, but thin model
exposure + encoder cost), YAML (whitespace-fragile, no comprehension win),
TOML (config format, arrays-of-records are its weakness), bare path list
(-61% but lossy). **Shape guidance:** markdown table (header once, then
rows) for these uniform record lists; plain-text only (no ANSI — Phase 6
owns color); exact table vs list-per-record is Claude's Discretion,
optimizing for model legibility and stable ordering. **Parity note:** the
golden/parity harness asserts the CLI `--json` shape — confirm which
existing tests assert MCP text content and update them deliberately; do not
let a JSON→markdown change silently pass a test that only checked the CLI
path.

**D-14 (★ Engine must learn `startPath`):** `OpenAt` currently DISCARDS the
caller's start path. `Engine{reader, repoRoot}` holds `repoRoot` = the
resolved index root, and `OpenAt(start)` throws `start` away after
resolving. Worktree detection needs both sides
(`detect(startPath, indexRoot)`). Retain the caller's start path on the
`Engine` (e.g. `startPath`) in `OpenAt`. Engines built via `New`/
`NewWithRoot` (tests, no start path) must degrade to "no mismatch," never
panic.

**D-15 (Test fixtures drive real `git`):** No faked `.git` directories.
Build all six layouts via `os/exec`: linked-worktree (`git worktree add`),
submodule (`git submodule add`), nested-clone (embedded clone, no gitlink),
monorepo-subdir (plain subdir), `.claude/worktrees/<name>/` (the
Sean-specific GSD layout — the true-positive case), symlinked paths
(`EvalSymlinks` both-sides case). `t.Skip` when git is absent; set
deterministic `GIT_*` env for hermetic, CI-stable fixtures. Expected
verdicts: linked-worktree ⇒ mismatch; `.claude/worktrees/` ⇒ mismatch;
submodule ⇒ no mismatch (gate 4); nested-clone ⇒ no mismatch (gate 4);
monorepo-subdir ⇒ no mismatch (gate 3); non-git ⇒ no mismatch (gate 1);
symlinked ⇒ no mismatch when it's really the same tree (gate 2).

### Claude's Discretion

File layout within `internal/query` for the status sections + the notice
helper (extend `status.go`/`render_markdown.go` vs a new
`render_status.go`), the internal shape of the `FilesByLanguage`
aggregation, the exact `internal/gitmeta` function signatures, and the
fixture-builder helper structure — planner/executor choose, so long as the
plain-text-only, shared-engine, and never-blocking constraints hold.

### Deferred Ideas (OUT OF SCOPE)

- Colorized/TTY-gated `status` (lipgloss sections, ASCII-vs-Unicode glyph
  fallback) — Phase 6 (TUI-02). Phase 2 lays down the layout Phase 6 paints.
- Exact `pendingChanges` added/modified/removed counts — explicitly out of
  scope for v1.0 (needs Sync's diff re-run per `status` call).
- `Journal:`/journalMode line — no Pebble analog, permanently dropped.
- Auto-init/index-sharing for worktrees ("make worktree support better") —
  deferred past v1.0 (WORK-FUT-01); v1.0 is detect + warn + notice only.
- Reusing `internal/gitmeta` for git sync hooks — Phase 5 (HOOK-01/02/03);
  design for reuse, don't build hook logic here.
- `getPendingFiles()` per-file freshness + `isWatcherDegraded()` sections in
  TS's MCP status — depend on a live watcher (Phase 3, WATCH-01).
- "Document release procedures" — deferred to Phase 8 (REL); not topically
  relevant to this phase.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| STAT-01 | `status` reports DB size (Pebble on-disk bytes), reversing the Phase-3 golden-corpus strip | Pattern 2 (WalkDir dbSizeBytes helper) + Pitfall 2 (golden exemption location) + Open Question #1 (Metrics() alternative) |
| STAT-02 | `status` reports nodes-by-kind and files-by-language breakdowns | Corrections #4 (FilesByLanguage is one field-read in the existing scan, not a second pass) |
| STAT-03 | `status` reports a live pending-changes / reindex-recommended state | Confirmed already live+reachable both surfaces (D-06 verification) — this phase's work is surfacing/rendering, not building |
| WORK-01 | Worktree/index mismatch detection via `git rev-parse` comparison | Pattern 1 (git subprocess primitive) + verbatim D-02/D-03 cascade confirmation |
| WORK-02 | Verbose `status` warning + compact notice on 6 other read tools (CLI+MCP) | Corrections #3 + Pitfall 1 + Open Questions #3/#4 (the JSON-vs-markdown MCP tension and the no-TS-precedent CLI design) |
| WORK-03 | Best-effort, never-blocking detection | Pattern 1 (bounded timeout, error-as-no-signal) + Security Domain (DoS mitigation) |
| TEST-02 | Fixtures for linked-worktree, submodule, nested-clone, monorepo-subdir, `.claude/worktrees/`, symlinked | Don't Hand-Roll (real git via os/exec) + Validation Architecture Wave 0 Gaps (no existing repo-building test helper) |
| SURF-06 | The 5 JSON-shaped MCP read tools return markdown; CLI `--json` unchanged | § SURF-06 (shared-helper risk, renderer placement, markdown shapes) + Corrections #5 (status is NOT already markdown — D-16's premise is wrong) + Validation Architecture (zero existing MCP coverage for these 5 tools) |
</phase_requirements>

## Summary

This phase has two independently-shippable halves that land through the same
`internal/query` seam: (1) replace `status`'s terse one-liner with TS 1.3.1's
full sectioned output (DB size, nodes-by-kind, files-by-language, live
staleness/reindex advisories), and (2) detect when the resolved
`.codegraph/` index belongs to a *different* git working tree than the
caller and warn about it on every read surface.

The TS 1.3.1 reference source is intact and was read directly this session
(confirming CONTEXT.md D-01's correction). `sync/worktree.js`'s 4-gate
cascade, its `git rev-parse` subprocess contract, and both warning strings
were read byte-for-byte and **match CONTEXT.md's D-02/D-03/D-11
transcriptions exactly — no discrepancy found**, including the subtle gate-4
suppression polarity. This research instead surfaces five things CONTEXT.md
could not have known without inspecting the current Go codebase directly:
(a) Pebble exposes a real `Metrics().DiskSpaceUsage()` API that is a
plumbing-heavier but more idiomatic alternative to D-07's `filepath.WalkDir`
sum; (b) the specific location `golden_parity_test.go` line ~651 that needs
a narrow `dbSizeBytes` exemption for D-08 (the shared `volatileKeys` map in
`golden_test.go` itself must NOT change — it governs the frozen TS goldens,
which correctly stay stripped); (c) `FilesByLanguage` can be computed inside
the **existing** `IterateFiles()` loop already in `Status()` — it needs one
more field read (`fileIt.File().Language`), not a second store scan; (d) the
TS CLI **never** wires worktree detection into any command but `status` —
the Go CLI's compact notice on the other 6 commands is a **new
extension beyond TS parity**, not a port, with no TS source to copy layout
from; and (e) — the most load-bearing finding — **D-13's "cache on the
Engine via `sync.Once`" plan provides zero cross-call benefit for MCP**,
because `internal/mcp/tools.go`'s `openEngine` constructs a brand-new
`*query.Engine` on every single tool call by design (D-02 Pitfall 2); the
cache must live at `BuildServer`/server-construction scope instead.

**Primary recommendation:** Port `sync/worktree.js` verbatim into a new
`internal/gitmeta` package exactly as CONTEXT.md specifies (no changes
needed there); implement `dbSizeBytes` via `filepath.WalkDir` over
`.codegraph/store/` (zero new plumbing, D-04a-compliant by construction);
add `startPath` to `Engine` and computed-once mismatch state, but put the
**cross-call cache** in a `gitmeta.CachingDetector` constructed once per MCP
server (D-13, corrected); and treat SURF-06 as **strictly additive** — new
`Render*` functions in `internal/query`, with the shared `Marshal*JSON`
helpers left untouched because the CLI `--json` path depends on them.

**Scope-change addendum (2026-07-15).** SURF-06 was added as an 8th
requirement after this research's Open Question #3 was escalated to the user.
Three findings from investigating it materially affect planning: (1) **all
five `Marshal*JSON` helpers are shared with the CLI `--json` path** — SURF-06
is not a swap, it is an addition (§ SURF-06); (2) **no existing test asserts
the MCP success payload of any of the 5 tools** — SURF-06 could be skipped
entirely and CI stays green, so the MCP assertion must be written first as
the TDD red test (§ Validation Architecture); and (3) **D-16's premise that
`status` "already" emits markdown on MCP is factually wrong** — it emits JSON
today, and D-12's blockquote requirement means it needs its own markdown
renderer in TS's distinct `handleStatus` shape (Corrections #5).

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Git worktree/common-dir detection | Backend (`internal/gitmeta`, new pkg) | — | Pure stdlib `os/exec` subprocess calls against the local git binary; no I/O boundary crosses a client/server split in this single-binary CLI+MCP product |
| Worktree-mismatch caching | Backend (`internal/mcp` server construction for MCP; none needed for one-shot CLI) | Backend (`internal/query.Engine`, intra-call only) | MCP is the only long-lived process in this codebase; the cache must outlive a single tool call, which only server-construction scope provides |
| Status content aggregation (DB size, nodes-by-kind, files-by-language, staleness) | Backend (`internal/query.Engine.Status`) | Storage (`internal/graphstore`, if the Pebble-Metrics route is chosen) | `internal/query` is the single read seam both CLI and MCP call (D-08b); `internal/graphstore` is the only package allowed to touch pebble/v2 directly (D-04a) |
| Status/notice rendering (plain text, sections, comma-grouping) | Backend (`internal/query` render helpers) | CLI / MCP (thin presentation glue) | Matches the existing `RenderExplore`/`RenderNode` precedent — one render function per output surface, called identically by both CLI and MCP |
| CLI `--json` output (callers/callees/impact/search/files/status) | Backend (`internal/query.Marshal*JSON`) | CLI (`internal/cli/*.go`, thin `writeJSONLine`) | **Unchanged by this phase.** These helpers are the golden-parity JSON-shape oracle AND the CLI `--json` contract — SURF-06 must not touch them (see § SURF-06's shared-helper finding) |
| MCP markdown output (callers/callees/impact/search/files) — SURF-06 | Backend (`internal/query`, new `Render*` funcs next to `RenderExplore`/`RenderNode`) | MCP (`internal/mcp/tools.go`, swaps its `Marshal*JSON` call for the `Render*` call) | The shared-engine rule (D-08b) puts every rendering in `internal/query` so CLI and MCP can never drift into two shapes; `internal/mcp` stays a thin delegation layer that re-renders nothing, exactly as its existing doc comments promise |

> **★ Superseded row (kept for traceability):** an earlier version of this
> map recorded "these 5 tools' MCP surface emits raw JSON text — the
> notice-injection point must respect that existing JSON contract." SURF-06
> (D-16) **dissolves that constraint entirely**: the JSON contract on the MCP
> surface is being deleted, not worked around. The two rows above replace it.
> See § SURF-06 and Corrections #3's follow-up note.

## Standard Stack

### Core

No new dependencies for this phase. Everything is stdlib:

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `os/exec` | stdlib (go 1.26.5) | `git rev-parse --show-toplevel` / `--git-common-dir` subprocess calls | CONTEXT.md D-03/D-04 lock this — "stdlib `os/exec` only — no pure-Go git lib" — matches the project's minimal-dependency constraint and TS's own `child_process.execFileSync` mechanism exactly |
| `path/filepath` | stdlib | `EvalSymlinks`, `Abs`, `WalkDir` for realpath resolution and the DB-size directory sum | Already the convention throughout `internal/query` (`resolve.go`, `status.go`) |
| `sync` | stdlib | `sync.Once`/mutex-guarded map for mismatch caching | No new dependency; matches the existing `pebbleStore`'s `sync.RWMutex` convention in `internal/graphstore/pebble_store.go` |

### Supporting

None — no supporting libraries needed. `golang.org/x/text/message` is explicitly **rejected** for D-10's comma-grouping (see Don't Hand-Roll).

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `filepath.WalkDir` sum for `dbSizeBytes` (D-07, locked) | `pebbleStore.db.Metrics().DiskSpaceUsage()` (verified to exist, `pebble/v2 v2.1.6`) | Cheaper (no per-file `stat` I/O) and closer to Pebble's own internal notion of live+obsolete+zombie bytes, but requires extending the `graphstore.GraphStore`/`Reader` interface and giving `Engine` a handle to the store (not just its snapshot `Reader`) — plumbing D-07 did not anticipate. See Open Questions #1. |
| Engine-scoped `sync.Once` cache (D-13, locked) | Server-construction-scoped cache (a small map/struct closed over by `internal/mcp/server.go`'s `BuildServer`) | D-13's stated rationale ("MCP gets the win") only holds if the cache outlives one tool call. Go's `Engine` is deliberately rebuilt fresh per MCP call (D-02 Pitfall 2), so an `Engine`-scoped `sync.Once` never fires twice — see Open Questions #2 and the Corrections section below. |

**Installation:** none — no `go get` required for this phase.

**Version verification:** N/A — no external packages added. `go.mod` already pins `github.com/cockroachdb/pebble/v2 v2.1.6` (verified via `go doc github.com/cockroachdb/pebble/v2 DB.Metrics` — resolves cleanly against the module cache) and go `1.26.5` (verified via `go version`).

## Package Legitimacy Audit

**Not applicable this phase.** No new external packages are introduced — `internal/gitmeta` is stdlib-only (`os/exec`, `path/filepath`, `os`) per CONTEXT.md D-03/D-04, and the status-content work touches only already-vendored `internal/graphstore`/`internal/schema` types. No `go.mod` changes are expected as an outcome of this phase's plans.

## Corrections to CONTEXT.md (evidence-based)

CONTEXT.md's D-02/D-03/D-11 (the worktree detection algorithm and both
warning/notice strings) were re-verified against the live TS 1.3.1 source
this session **byte-for-byte and found correct with no discrepancy** —
including the gate-4 suppression polarity CONTEXT.md specifically flagged as
counter-intuitive, and the glyph verification (see below). Nothing in those
decisions needs correcting.

The following are not corrections to *locked* decisions so much as
load-bearing implementation facts CONTEXT.md's decisions did not (and could
not, without a codebase read) account for:

1. **D-13's "Engine-scoped `sync.Once`" cache does not deliver the "MCP gets
   the win" benefit it claims.** `internal/mcp/tools.go`'s `openEngine`
   (lines 47-64) opens a **fresh** `query.OpenAt` snapshot on every single
   tool call — its own doc comment says so explicitly: *"opens a FRESH
   query.OpenAt snapshot for this call (D-02/D-08b, RESEARCH Pitfall 2 —
   never a snapshot cached at server construction)."* There is no
   engine/path cache anywhere in `internal/mcp` today (confirmed via a repo
   grep for `cache|Cache|sync.Once` in that package — zero hits besides a
   doc-comment mention). A `sync.Once` field on `*query.Engine` is therefore
   scoped to exactly one tool call and is hit at most once regardless — it
   dedupes nothing across calls, unlike TS's `this.worktreeMismatchCache`
   Map, which lives on the long-lived MCP server object and survives across
   every tool call in the session (that's precisely what issue #926, cited
   in D-13, is about). **Recommendation:** keep the `Engine`-level
   once-per-call resolution (it is still useful — it means `Status()` and
   the render path never call `git` twice within one request), but add a
   **second**, server-scoped cache in `internal/mcp/server.go`'s
   `BuildServer` (a small `map[string]*gitmeta.Mismatch` + mutex, or a tiny
   `gitmeta.CachingDetector` type, keyed on `startPath\x00indexRoot` exactly
   like TS), constructed once per process and closed over by every
   registered tool handler. The CLI needs no such cache (D-13 already notes
   this correctly — it is one-shot).
2. **D-13's "2 git subprocesses" undercounts the worst case.**
   `detectWorktreeIndexMismatch` calls `gitWorktreeRoot` up to twice
   (`startPath`, then `resolvedIndexRoot`) and `gitCommonDir` up to twice
   (`worktreeRoot`, then `resolvedIndexRoot`) — **up to 4** subprocess
   spawns in the full-cascade (genuine-mismatch) case, not always 2. Early
   returns (non-repo, same-tree, non-worktree-index-root) short-circuit at 1
   or 2 calls. This doesn't change any decision, just the cost estimate
   informing the caching Recommendation above.
3. **Go's MCP surface for `callers`/`callees`/`impact`/`search`/`files` is
   raw JSON text, not markdown, unlike TS.** TS's `withWorktreeNotice`
   mechanism works uniformly because *every* TS MCP tool result is
   markdown-formatted text (confirmed: `mcp/tools.js`'s `handleStatus`
   builds a `lines.join('\n')` markdown string, and the same is true for its
   sibling handlers). Go's `internal/mcp/tools.go` `companionHandler` for
   `node`/`search`/`callers`/`callees`/`impact`/`files` marshals a
   structured Go value with `json.Marshal`/`Marshal*JSON` and returns
   `mcp.NewToolResultText(string(data))` — i.e. the tool's entire text
   content **is** parseable JSON today for 5 of the 7 companion tools
   (`node` is the exception — it already returns markdown via
   `Engine.Node`). Literally prepending `notice + "\n\n"` (TS's exact
   mechanism) to these 5 tools' output would make their MCP payload
   sometimes-JSON, sometimes-text-then-JSON — a real (if rare, mismatch-only)
   behavior change to their machine-readability contract. This is not
   something CONTEXT.md's D-12 could have caught without reading
   `tools.go`, since CONTEXT.md's canonical refs point only at TS's
   `mcp/tools.js`. See Open Questions #3 for the concrete design choice this
   forces.
   > **★ RESOLVED 2026-07-15 — escalated to the user, who folded the fix into
   > this phase as SURF-06 (D-16).** The JSON-parseability concern was
   > correctly diagnosed but the *conclusion* inverted on inspection: the
   > "contract" had **no consumer** (MCP text content is model-facing; nothing
   > in this repo nor Claude Code unmarshals it), so rather than protect the
   > JSON shape, the decision is to **delete it** — all 5 tools move to
   > markdown. The `_worktreeNotice` field hybrid is withdrawn; D-12 is now a
   > single uniform text-prefix across all 7 non-status read tools. This
   > finding's value was surfacing the fork, not the shape it recommended.
5. **★ NEW — D-16's premise is factually wrong about `status`: MCP
   `codegraph_status` does NOT currently emit markdown.** D-16 states the 5
   tools switch to markdown *"matching the 3 tools that already do
   (`explore`/`node`/`status`)"*. Direct source read of
   `internal/mcp/tools.go` lines 327-344 shows `codegraph_status`'s handler
   ends with `data, err := query.MarshalStatusJSON(result)` →
   `mcp.NewToolResultText(string(data))` — **raw JSON, identical in shape to
   the 5 tools D-16 is changing.** Only **2** Go MCP tools emit markdown
   today (`explore` via `Engine.Explore`, `node` via `Engine.Node`); **6**
   emit JSON (`search`/`callers`/`callees`/`impact`/`files`/`status`).
   **Why this matters to the planner (scope, not pedantry):** D-12 already
   requires MCP `status` to render a **blockquote verbose warning** (`> ⚠ ` +
   the warning with `\n` → `\n> `) "matching TS's `mcp/tools.js`" — a
   markdown construct that is meaningless prepended to a JSON blob. So MCP
   `status` needs a markdown renderer too, and it is **not** covered by
   SURF-06's 5-tool scope, **nor** by D-09's CLI layout — because **TS has
   two structurally different status renderings**, both verified this session:
   - CLI (`bin/codegraph.js` ~900-935): `Index Statistics:` / `  Files:
     <n>` / `Nodes by Kind:` / `  <kind padEnd(15)> <count>` — D-09's target.
   - MCP (`mcp/tools.js` ~3894-3938): `**CodeGraph Status**` / `**Files
     indexed:** <n>` / `**Total nodes:** <n>` / `**Nodes by Kind:**` /
     `- <kind>: <count>` / `**Languages:**` / `- <lang>: <count>` — a
     bolded-key + bullet-list markdown form, NOT the CLI's padded columns.
   **Recommendation:** treat "MCP `status` markdown renderer (TS
   `handleStatus` form)" as an explicit task under STAT-01/02/03 + D-12,
   sized separately from D-09's CLI renderer. Do not assume one renderer
   serves both surfaces — TS deliberately ships two, and D-12's blockquote
   requirement only makes sense against the markdown one. Note this does not
   contradict the shared-engine rule: **both** renderers live in
   `internal/query` (like `RenderExplore`), they are just two functions
   (`RenderStatus` for the CLI form, `RenderStatusMarkdown` for the MCP form),
   each called by exactly one surface — the same way TS does it.
4. **`FilesByLanguage` is one field-read inside the scan `Status()` already
   runs, not a second scan.** `schema.File` (via `internal/schema/graph.pb.go`
   line 330) already carries a `Language string` field, and
   `internal/query/status.go`'s `Status()` already iterates every file via
   `fileIt.Next()` to compute `fileCount` — it simply never calls
   `fileIt.File()` inside that loop today. Adding
   `fileLang := fileIt.File().Language; filesByLang[fileLang]++` to the
   *existing* loop is the entire scan-side change; CONTEXT.md D-05's "size
   the task accordingly, this is an extra scan" framing is correct in
   *implementation-weight* terms (it's still genuinely new aggregation
   logic + a new struct field + wiring through `MarshalStatusJSON` and both
   renderers) but not in *I/O* terms — no second `IterateFiles()` call is
   needed. This also means `Languages []string` can be **re-derived from
   the new `FilesByLanguage` map** (as D-05 specifies) essentially for free
   in the same pass, rather than kept as today's separate node-derived
   `languageSet`.

## Architecture Patterns

### System Architecture Diagram

```
CLI command (cobra RunE)              MCP tool handler (server.ToolHandlerFunc)
   resolveStartPath(--path|cwd)           resolvePath(req, defaultPath)
        │  (may be relative)                    │
        │                                confineToRepoRoot(path, repoPath)
        │                                        │  (always absolute)
        └───────────────┬────────────────────────┘
                         ▼
              query.OpenAt(start)                 ← D-14: start is now RETAINED
                         │                            (filepath.Abs'd) as Engine.startPath,
                         ▼                            not discarded after the walk-up
         ResolveCodegraphDir(start) → repoRoot (index root, walked UP from start)
                         │
                         ▼
     graphstore.Open(repoRoot/.codegraph/store) → Snapshot() → Reader
                         │
                         ▼
         Engine{reader, repoRoot, startPath}      ← new field this phase
                         │
        ┌────────────────┼─────────────────────────────┐
        ▼                ▼                               ▼
  Engine.Status()   Engine.Explore()/.Node()      Engine.Callers()/.Callees()/
  (dbSizeBytes,      (markdown string,             .Impact()/.Search()/.Files()
   nodesByKind,       already shared render        (structured *Result values;
   filesByLanguage,   path — CLI+MCP identical)     CLI renders text, MCP marshals
   stale, index.*)                                  JSON — TWO separate renderers)
        │                                               │
        ▼                                               ▼
  gitmeta.DetectIndexMismatch(startPath, repoRoot)  (same detection, called once
  — best-effort, cached per D-13 (Engine-scoped      per Engine either via Status()
  once-per-call; MCP ALSO needs a server-scoped       or a shared Engine.WorktreeMismatch()
  cache — see Corrections #1)                         accessor)
        │
        ▼
  status: verbose worktreeMismatchWarning (own section)
  other 6 tools: compact worktreeMismatchNotice
    - CLI: printed as its own warn-style line, human mode only (skip on --json,
      matching TS status.go's own --json branch never calling warn())
    - MCP explore/node: text-prefixed onto the markdown string (safe — always text)
    - MCP callers/callees/impact/search/files: OPEN QUESTION — text-prefix onto
      JSON (matches TS mechanism, breaks naive JSON.parse) vs. add a JSON field
      (preserves JSON contract, diverges from TS's literal mechanism)
```

### Recommended Project Structure

```
internal/gitmeta/            # NEW package (D-04, stdlib os/exec only)
├── worktree.go               # gitWorktreeRoot, gitCommonDir, realpath (D-03)
├── detect.go                 # DetectIndexMismatch (the 4-gate cascade, D-02)
├── notice.go                 # Mismatch struct + Warning()/Notice() string builders (D-11)
└── *_test.go                 # unit tests for the gates against real git repos

internal/gitmeta/fixtures/    # or a _test.go helper file — TEST-02's six layouts
└── (t.TempDir()-built linked-worktree / submodule / nested-clone / monorepo-subdir /
     .claude-worktrees / symlinked fixtures, built via os/exec git commands, D-15)

internal/query/
├── status.go                  # extended: FilesByLanguage, DbSizeBytes fields + computation
├── engine.go                  # extended: startPath field, WorktreeMismatch() cached accessor
├── render_markdown.go         # extended: RenderStatus (or new render_status.go — discretion)
│                               #   + worktreeNotice() alongside the existing staleBanner()
└── resolve.go                 # unchanged — still the indexRoot side of detection

internal/mcp/
├── server.go                  # extended: server-scoped mismatch cache, closed over by handlers
└── tools.go                   # extended: notice injection per the Open Questions #3 decision

internal/cli/
├── status.go                  # replaced: sectioned layout instead of the terse one-liner
└── (node/explore/search/callers/callees/impact/files).go  # each gains a notice print call
```

### Pattern 1: Best-effort git subprocess with bounded timeout (D-03)

**What:** Every git introspection call is wrapped in a 5s
`exec.CommandContext` timeout, discards stderr, and treats *any* error
(including timeout, missing git, non-repo) as "no signal" rather than
propagating an error.

**When to use:** `internal/gitmeta`'s `gitWorktreeRoot`/`gitCommonDir` —
never let git absence/slowness block a read query (WORK-03).

**Example (ported from `sync/worktree.js`, confirmed verbatim this session):**
```go
// Source: TS sync/worktree.js gitWorktreeRoot/gitCommonDir, ported per D-03
func gitWorktreeRoot(ctx context.Context, dir string) string {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	out, err := cmd.Output() // stderr discarded — mirrors stdio: ['ignore','pipe','ignore']
	if err != nil {
		return ""
	}
	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return ""
	}
	return realpath(trimmed)
}

// realpath mirrors TS's realpathSync-with-fallback: on EvalSymlinks error,
// fall back to the plain absolute path rather than failing (D-03).
func realpath(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved
	}
	return abs
}
```

### Pattern 2: dbSizeBytes via best-effort directory walk (D-07, mirrors existing `newestSourceMtime`)

**What:** Sum every regular file's size under `.codegraph/store/`, skipping
unreadable entries rather than failing the whole `status` call — the same
best-effort shape `status.go`'s existing `newestSourceMtime` already uses.

**Example:**
```go
// Source: this plan, mirroring status.go's existing newestSourceMtime pattern
func dbSizeBytes(storeDir string) (int64, error) {
	var total int64
	err := filepath.WalkDir(storeDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // best-effort: skip unreadable entries, don't abort the walk
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		total += info.Size()
		return nil
	})
	return total, err
}
```

**Alternative (not chosen without further plumbing — see Open Questions #1):**
`pebbleStore.db.Metrics().DiskSpaceUsage()` — verified to exist in
`pebble/v2 v2.1.6` (`go doc github.com/cockroachdb/pebble/v2 Metrics` /
`DB.Metrics`) and sums `WAL.PhysicalSize + WAL.ObsoletePhysicalSize +
Table.Local.{Live,Obsolete,Zombie}Size + BlobFiles.Local.{...}Size +
options/manifest file sizes + in-progress compaction bytes` — a superset
that avoids filesystem `stat` I/O per file, but requires extending
`graphstore.GraphStore`/`Reader` and giving `Engine` a store handle (not
just its snapshot `Reader`), which today it does not have.

### Pattern 3: MB rendering (D-07)

```go
// Source: TS bin/codegraph.js line 904, `(stats.dbSizeBytes/1024/1024).toFixed(2)`
fmt.Sprintf("%.2f MB", float64(dbSizeBytes)/1024/1024)
```

### Pattern 4: Comma-grouped number formatting (D-10, hand-rolled, no `x/text`)

```go
// Source: this plan — deterministic en-US-style grouping, TS's
// n.toLocaleString() default-locale (en-US in CI) behavior, per D-10.
func formatNumber(n int64) string {
	s := strconv.FormatInt(n, 10)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	parts = append([]string{s}, parts...)
	out := strings.Join(parts, ",")
	if neg {
		out = "-" + out
	}
	return out
}
```

### Anti-Patterns to Avoid

- **Caching the worktree mismatch only on `*query.Engine`:** looks correct
  (mirrors D-13's literal wording) but delivers zero cross-call benefit for
  MCP because `Engine` is rebuilt fresh per tool call — see Corrections #1.
  Always pair it with a server-construction-scoped cache for MCP.
- **Prepending the compact notice inside `internal/query`'s `Marshal*JSON`
  functions:** these functions are the golden-parity JSON-shape oracle for
  `callers.json`/`impact.json`/etc — mutating their output shape to embed
  free text would break every existing byte-shape assertion in
  `golden_parity_test.go`. Any notice injection for the 5 JSON-shaped tools
  belongs at the MCP result-wrapping layer (or as an explicit new optional
  field), never inside the `Marshal*JSON` functions themselves.
- **Re-walking `.codegraph/` (the whole directory, including
  `.sync-pending`) for `dbSizeBytes`:** D-07 is explicit that only
  `store/` counts — walking the parent `.codegraph/` would double-count
  nothing incorrectly today, but would silently start including any future
  sidecar files (e.g. a config file) as "database size," which is not
  TS-truthful (TS's `dbPath` is the SQLite file alone).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Git worktree/common-dir resolution | A custom `.git`/`.git` gitlink-file parser | `git rev-parse --show-toplevel` / `--git-common-dir` subprocess (D-02/D-03, locked) | Git's own resolution correctly handles gitlink files, `.git` as a file (worktrees/submodules), and repo-format edge cases a hand-rolled parser would get wrong — this is exactly why TS shells out instead of reading `.git` directly |
| Locale-aware number formatting | `golang.org/x/text/message` | A ~15-line hand-rolled comma-grouper (D-10, locked) | The x/text route reintroduces the exact locale variance D-10 exists to eliminate (Go's default `message.Printer` is locale-neutral by default and would need explicit `language.AmericanEnglish` configuration to match TS's CI behavior anyway — simpler to hand-roll a fixed grouper) |
| Test git repos (linked worktrees, submodules) | Hand-built fake `.git` directory structures | Real `git init`/`git worktree add`/`git submodule add` via `os/exec` in `t.TempDir()` (D-15, locked) | The whole point of TEST-02 is validating against git's *actual* `--show-toplevel`/`--git-common-dir` semantics — especially the submodule-vs-linked-worktree common-dir distinction a hand-built `.git` cannot reproduce faithfully |
| DB size introspection | A custom Pebble SSTable/WAL byte-layout parser | `filepath.WalkDir` over `.codegraph/store/` (chosen) or `pebble.DB.Metrics().DiskSpaceUsage()` (alternative, see Open Questions #1) | Both are supported, tested primitives; a hand-rolled SSTable format reader would need to track Pebble's internal file-naming/format across upgrades |

**Key insight:** every piece of new logic this phase needs (git introspection,
number formatting, DB sizing) has either an official upstream API
(`git rev-parse`, `pebble.Metrics()`) or was already deliberately
hand-rolled by the TS reference for good reason (`toLocaleString()` avoidance)
— there is no case in this phase where reaching for a new third-party
dependency is the right call.

## Common Pitfalls

### Pitfall 1: Prepending the notice breaks the existing JSON-only contract for 5 of 7 MCP tools — ★ RESOLVED by SURF-06, retained as the reasoning trail

> **★ STATUS: RESOLVED 2026-07-15. Do not action this pitfall as written.**
> It correctly surfaced the fork, and was escalated to the user, who folded
> the fix into this phase as **SURF-06 / D-16**: the 5 tools move to
> markdown, so the JSON contract this pitfall protects **ceases to exist**.
> The recommended mitigation below (choose between text-prefix and a JSON
> field) is **superseded** — the answer is "neither." Retained because the
> analysis is what produced SURF-06, and because a future reader hitting a
> JSON-shaped MCP tool should understand why it isn't one anymore. See
> § SURF-06.

**What goes wrong:** Literally porting TS's `withWorktreeNotice`
(`${notice}\n\n${first.text}`) onto Go's `callers`/`callees`/`impact`/
`search`/`files` MCP handlers turns their currently-always-valid-JSON text
content into "notice line, blank line, then JSON" whenever a mismatch is
detected — silently breaking any downstream code that does a raw
`JSON.parse()`/`json.Unmarshal()` on the MCP tool's text content.

**Why it happens:** CONTEXT.md's D-12 was written against TS's source,
where *every* MCP tool already returns markdown-formatted text (never raw
JSON) — the prefix-a-string mechanism was always safe there. Go's
architecture diverged from that (these 5 tools emit `json.Marshal` output
directly) in a prior phase, for reasons unrelated to this one.

**How to avoid:** Decide explicitly (see Open Questions #3) whether to
(a) literally match TS's mechanism and document the JSON-validity caveat,
or (b) add a new optional JSON field (e.g. `"_worktreeNotice"`) to the
existing marshaled shape instead of text-prefixing. Do not silently pick
one without recording the choice — it's the kind of decision a future
`--json`-consuming caller will be surprised by either way.

### Pitfall 2: dbSizeBytes tripping the shared `findVolatileKeys` self-check

**What goes wrong:** `testdata/golden/golden_parity_test.go` line ~651 calls
`findVolatileKeys(decoded, "our status.json")` against OUR OWN marshaled
`StatusResult` JSON, using the *same* `volatileKeys` map (`golden_test.go`)
that flags `dbSizeBytes` as forbidden. The moment `StatusResult` gains a
`DbSizeBytes` field (D-07), this specific assertion starts failing —
**not** `TestGoldenFixturesExist` (which only globs `corpus/*/*.json`, the
frozen TS oracle files, and must keep failing if a *TS* golden ever
re-includes `dbSizeBytes` — do not touch that shared map).

**Why it happens:** the same `isVolatileKey`/`volatileKeys` function is
reused for two different purposes: (1) asserting the frozen TS capture
fixtures stay stripped (must never change), and (2) asserting OUR OWN
output has no non-deterministic fields (was true before D-07, is
intentionally no longer true for exactly this one key after D-07).

**How to avoid:** at `golden_parity_test.go` line ~651, delete
`"dbSizeBytes"` from the decoded map (or call a small
`findVolatileKeysExcept(decoded, "our status.json", "dbSizeBytes")`
variant) before the volatility check, and add a **separate** plausibility
assertion immediately after per D-08: key present, `> 0`, integer type, and
`testdata/golden/README.md`'s volatile-fields table gets a new row
documenting the exemption's rationale (Pebble LSM compaction makes
byte-for-byte reproducibility strictly less achievable than SQLite's, so
the divergence is *stronger*, not weaker, than the existing TS-side
stripping rationale).

### Pitfall 3: `resolveStartPath`'s CLI `--path` value may be relative

**What goes wrong:** `internal/cli/query.go`'s `resolveStartPath` returns
the raw `--path`/`-p` flag value **unresolved** when supplied (only
`os.Getwd()`'s absolute-by-construction result is used as the fallback).
If `OpenAt` stores this verbatim as `Engine.startPath` and later hands it to
`gitmeta.DetectIndexMismatch` without first calling `filepath.Abs`, a
relative `--path ../foo` would produce a `startPath` that is not
byte-comparable against `EvalSymlinks`-resolved paths downstream, breaking
gate 2's `worktreeRoot == resolvedIndexRoot` equality check in some edge
cases (though `exec.Cmd.Dir` itself tolerates relative dirs fine, so
detection would still generally work — it's the equality comparisons that
are the risk).

**How to avoid:** `OpenAt` should call `filepath.Abs(start)` before storing
`Engine.startPath` — mirroring TS's own `path.resolve(pathArg ||
process.cwd())` at the CLI entry point, and consistent with
`ResolveCodegraphDir`'s existing `filepath.Abs` call on the same input.

**Warning signs:** a TEST-02 fixture that passes `--path` as a relative
string during a `chdir`-shifted test process could silently produce a false
negative if this isn't handled.

### Pitfall 4: TS's `Journal:`/`indexState` warning branches have no Go equivalent — don't try to port them

**What goes wrong:** TS's status command prints extra `warn()` lines for
`indexState === 'indexing' | 'partial' | 'failed'` and for
`pendingRefs > 0` (`bin/codegraph.js` lines 886-897). Go's `IndexHealth.State`
is only ever `"complete"` or `"not_indexed"` (no partial/indexing/failed
concept exists in the Go indexer), and `PendingRefs` is hard-pinned to `0`
(D-06). Attempting to port these branches produces dead code that can never
fire.

**How to avoid:** skip these TS branches entirely when porting the status
command's human-output layout (D-09's target structure already omits them)
— they are TS-only states with no Go analog, not omissions to fix.

## SURF-06: MCP Markdown Output (added 2026-07-15 — scope change, D-16)

### ★ The highest-risk finding: every `Marshal*JSON` helper is SHARED with the CLI `--json` path

**This is not a simple swap.** All five helpers the MCP handlers call are the
*same functions* the CLI `--json` flag calls. Verified exhaustively by
grepping every call site of each helper:

| Helper | Defined at | MCP caller (changes) | **CLI `--json` caller (MUST NOT change)** | Also called by |
|--------|-----------|---------------------|-------------------------------------------|----------------|
| `query.MarshalCallersJSON` | `internal/query/traverse.go:546` | `internal/mcp/tools.go:248` | **`internal/cli/callers.go:41`** | `internal/query/traverse_test.go:632` |
| `query.MarshalCalleesJSON` | `internal/query/traverse.go:545` | `internal/mcp/tools.go:272` | **`internal/cli/callees.go:42`** | `internal/query/traverse_test.go:616` |
| `query.MarshalImpactJSON` | `internal/query/traverse.go:547` | `internal/mcp/tools.go:296` | **`internal/cli/impact.go:43`** | — |
| `query.MarshalFilesJSON` | `internal/query/files.go:219` | `internal/mcp/tools.go:321` | **`internal/cli/files.go:51`** | `internal/query/files_status_test.go:287` |
| *(none — `search` has no helper)* | — | `internal/mcp/tools.go:224` (`json.Marshal(locs)` inline) | **`internal/cli/search.go:45`** (`json.Marshal(locs)` inline) | — |
| `query.MarshalStatusJSON` | `internal/query/status.go:252` | `internal/mcp/tools.go:339` ← **also JSON today, see Corrections #5** | **`internal/cli/status.go:41`** | `files_status_test.go:375`, `golden_parity_test.go:635` |

**The implication the planner must internalize:** *mutating any
`Marshal*JSON` helper to emit markdown would silently break the CLI `--json`
contract and the golden parity oracle simultaneously.* SURF-06 is therefore
**strictly additive**:

- **Do NOT touch** any `Marshal*JSON` function body. They stay JSON forever —
  they are the CLI contract *and* `golden_parity_test.go`'s shape oracle.
- **DO add** new sibling `Render*` functions in `internal/query` (markdown,
  plain-text, ANSI-free — next to `RenderExplore`/`RenderNode`).
- **DO change** only the six `internal/mcp/tools.go` call sites (lines
  ~224, 248, 272, 296, 321, and — per Corrections #5 — 339) to call
  `Render*` instead of `Marshal*JSON`.

After the change, each helper family has exactly one caller per surface:
`Marshal*JSON` ← CLI only; `Render*` ← MCP only. That asymmetry is
intentional and worth a doc comment on each, since a future reader will
otherwise "helpfully" reunify them.

### Where the markdown renderers live: `internal/query`, not `internal/mcp`

**Recommendation: `internal/query`.** Three converging reasons:

1. **The shared-engine rule (D-08b) is explicit and already enforced by
   precedent.** `internal/mcp/tools.go`'s own doc comments state the
   invariant twice: *"returns the markdown result as-is (no re-rendering in
   internal/mcp, D-08b)"* (line ~79) and *"marshal via the matching
   internal/query Marshal\*JSON formatter (D-08b — no JSON shaping
   re-implemented here)"* (line ~169). Putting renderers in `internal/mcp`
   would violate a rule the file documents about itself.
2. **`RenderExplore`/`RenderNode` already establish the location.** They live
   in `internal/query/render_markdown.go` and are called by both surfaces.
   The new `Render*` functions are the same category of thing.
3. **Testability.** `internal/query` renderers are unit-testable without
   standing up an MCP server; `internal/mcp` ones would need the in-process
   client harness for every assertion.

**Placement within `internal/query` is Claude's Discretion** (CONTEXT.md
grants this). Suggested: a new `render_results.go` for the 5 SURF-06
renderers, keeping `render_markdown.go` focused on explore/node — but
extending `render_markdown.go` is equally defensible.

**Naming caution:** avoid `RenderFiles`/`RenderStatus` colliding with
Phase 6's TUI-02 work (which will add *colorized* renderers for `status` and
`files`). Phase 6 owns the ANSI/lipgloss layer and will need its own names;
picking `Render*Markdown` or `Render*Text` now may save a rename later. Flag
only — not a blocker.

### Recommended markdown shape

**Key structural finding: 4 of the 5 payloads are the SAME record type.**
`Location{Name, Kind, FilePath, StartLine}` (`internal/query/search.go:17`)
backs `callers`, `callees`, `impact`, and `search`. Only `files` differs
(`FileEntry{Path, Language, NodeCount, EdgeCount}`). So **one shared
`renderLocationTable([]Location) string` helper serves 4 tools**, and `files`
gets its own. That is 2 table renderers + 4 thin headers, not 5 bespoke
implementations.

**Shape: markdown table.** Header row once, then one row per record — this is
exactly where the -41% win comes from (JSON re-states all 4 keys on every one
of 308 records; a table states them once). Rejected alternatives are recorded
in D-16 and are not revisited here.

```markdown
**Callers of `Alpha`** — 3 callers

| Name | Kind | Location |
|---|---|---|
| `helper` | function | `internal/a/helper.go:12` |
| `Run` | method | `internal/b/run.go:40` |
```

```markdown
**Impact of `helper`** — depth 2, 5 nodes, 7 edges

| Name | Kind | Location |
|---|---|---|
| `helper` | function | `internal/a/helper.go:12` |
```

```markdown
**Files** — 308 indexed

| Path | Language | Nodes | Edges |
|---|---|---|---|
| `internal/a/helper.go` | go | 12 | 30 |
```

Notes:
- **Collapse `FilePath`+`StartLine` into one `path:line` cell.** It matches
  the existing `formatNodeRef` convention (`render_markdown.go:107`,
  `"%s (%s:%d)"`) that explore/node already use, and saves a column.
- **Backtick identifiers and paths** — consistent with `RenderExplore`'s
  per-file headers and `RenderNodeMultiDef`'s overflow list.
- **Header line carries the scalars** (`symbol`, `depth`, `nodeCount`,
  `edgeCount`) that have no place in a per-row table — mirroring
  `ImpactResult`'s non-list fields and TS's own `**Files indexed:** N`
  bolded-key style.
- **Empty result sets:** render the header plus an explicit "No callers
  found." line rather than a headerless empty table — a bare table header
  with zero rows reads as a rendering bug to a model.

**`files --format tree` needs a SECOND shape — a table cannot represent it.**
`FilesResult` is a union: `Format:"flat"` populates `Files []FileEntry`
(tabular), but `Format:"tree"` populates `Tree []*FileTreeNode` (nested,
`internal/query/files.go:57-71`). The MCP `files` tool exposes `format`
(`tools.go:152`) and defaults to `""` → flat, so the tree path is reachable
but non-default. Render the tree as an indented plain-text list, mirroring
the CLI's existing `printFileTree` (`internal/cli/files.go:60`) — and
consider whether that helper should move to `internal/query` for reuse rather
than being reimplemented (Claude's Discretion; noting that TS's MCP `files`
defaults to `'tree'` while ours defaults to flat is a pre-existing,
out-of-scope divergence — do NOT "fix" it here, that is SURF-territory for
Phase 8).

### Ordering determinism

**No new sorting work is needed — every result slice is already
deterministically ordered before it reaches a renderer.** The renderer must
simply **preserve slice order** and never re-sort. Verified per tool:

| Tool | Ordering source | Deterministic? |
|------|----------------|----------------|
| `files` | `sort.Slice(entries, ...Path < ...Path)` — `files.go:149` | ✓ path-ascending |
| `search` | `sort.SliceStable(results, ...)` by lexical tier — `search.go:98` | ✓ tier + stable input order |
| `callers` | `BuildReverseAdjacency` map iteration, edges sorted `edges[i].Source < edges[j].Source` — `traverse.go:82` | ✓ (via the sorted adjacency) |
| `callees` | `IterateEdges(node.Id)` — a single contiguous range scan ordered `(src, kind, dst)` by the store contract (`graphstore/store.go:58-61`) | ✓ storage-key order |
| `impact` | BFS over the sorted reverse adjacency, `visited` guard — `traverse.go:424-449` | ✓ BFS order over deterministic input |

**Do not add a `sort` call inside a renderer.** Doing so would silently
diverge MCP's row order from the CLI `--json` array order for the same query
— reintroducing exactly the CLI/MCP drift the shared-engine rule exists to
prevent.

### Conflicts with this document's earlier findings

| Earlier finding | Status after SURF-06 |
|---|---|
| **Pitfall 1** ("prepending the notice breaks the JSON contract for 5 of 7 MCP tools") | **Dissolved, not contradicted.** The pitfall correctly identified the fork; D-16 removes the JSON shape entirely, so there is no contract left to break. Pitfall 1 is retained below as the *reasoning trail* that produced SURF-06, annotated as resolved. |
| **Open Question #3** (text-prefix vs. JSON field) | **Answered by the user:** neither — the JSON goes away. Annotated in place. |
| **Architectural Responsibility Map** row for "`--json` / MCP JSON-shaped output" | **Split into two rows** (CLI-JSON vs MCP-markdown) — the original conflated two surfaces that SURF-06 now separates. Superseded row noted inline. |
| **Corrections #3** ("Go's MCP surface for these 5 is raw JSON, unlike TS") | **Still accurate as a statement of the pre-phase state**, and it is the *evidence* for SURF-06. After this phase it becomes historical. |
| **Anti-Patterns** ("never prepend the notice inside `Marshal*JSON`") | **Strengthened.** Now doubly true: those helpers are the CLI contract, and SURF-06 gives MCP its own render path, so there is zero reason for MCP concerns to reach them. |
| **D-12/D-13 as originally written in CONTEXT.md** | Both corrected upstream using this research's findings; the `<user_constraints>` block above reflects the corrected text. |

No finding is invalidated. SURF-06 resolves a fork this research surfaced
rather than overturning any verified fact.

## Code Examples

### Verified TS worktree detection cascade (verbatim source read this session)

```js
// Source: sync/worktree.js, detectWorktreeIndexMismatch — confirmed
// byte-for-byte against CONTEXT.md D-02's transcription, no discrepancy.
function detectWorktreeIndexMismatch(startPath, indexRoot) {
    const worktreeRoot = gitWorktreeRoot(startPath);
    if (!worktreeRoot) return null;                                    // gate 1
    const resolvedIndexRoot = realpath(indexRoot);
    if (worktreeRoot === resolvedIndexRoot) return null;                // gate 2
    if (gitWorktreeRoot(resolvedIndexRoot) !== resolvedIndexRoot) return null; // gate 3
    const worktreeCommon = gitCommonDir(worktreeRoot);
    const indexCommon = gitCommonDir(resolvedIndexRoot);
    if (worktreeCommon && indexCommon && worktreeCommon !== indexCommon) return null; // gate 4
    return { worktreeRoot, indexRoot: resolvedIndexRoot };
}
```

### Verified glyph bytes (hexdump, this session)

```
$ python3 -c "..." # scanned sync/worktree.js for \xe2\x9a\xa0
b'e(m) {\n    return (`\xe2\x9a\xa0 CodeGraph result'
```
Exactly one occurrence, immediately followed by `" CodeGraph result…"` — no
`\xef\xb8\x8f` (U+FE0F variation selector) anywhere in the file. Confirms
D-11's glyph claim independently.

### TS CLI's own `--json` mode never calls `warn()` — the precedent for gating the CLI notice line

```js
// Source: bin/codegraph.js lines 838-879 (status command's JSON branch)
if (options.json) {
    console.log(JSON.stringify({ /* ...worktreeMismatch as a structured field... */ }));
    return; // returns BEFORE the human-output branch's warn(worktreeMismatchWarning(...)) call
}
```
This directly supports gating the Go CLI's new compact-notice line to the
human-output branch only, skipping it when `--json` is set — every Go CLI
command already has this exact `if jsonOut { ...; return }` / human-branch
split (`internal/cli/{status,callers,callees,impact,search,files}.go` all
follow this shape; `node`/`explore` have no `--json` mode at all).

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| `status` one-line `backend=… files=… nodes=…` summary (v0.1) | TS-parity sectioned layout (`Index Statistics:`/`Nodes by Kind:`/`Files by Language:`) | This phase (D-09) | `--json` shape gains `dbSizeBytes`/`filesByLanguage`; human output structurally changes, breaking anything scraping the old one-liner (none identified in-repo) |
| `WorktreeMismatch *string` always-nil placeholder | Live `*gitmeta.Mismatch`-derived value | This phase (WORK-01) | `files_status_test.go`'s existing "Phase-4-only keys render present-but-inert" subtest must be updated, not deleted (per CONTEXT.md's canonical_refs) |

**Deprecated/outdated:** the `PendingChanges{Added,Modified,Removed}`
all-zero placeholder remains — explicitly out of scope for v1.0
(REQUIREMENTS.md's Out of Scope table); do not attempt to make it live in
this phase.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `pebble.DB.Metrics().DiskSpaceUsage()` (verified to exist via `go doc`) behaves identically across concurrent snapshot reads the way `filepath.WalkDir` does — i.e. it's safe to call from a read-only context without racing an in-flight writer | Standard Stack / Pattern 2 alternative | Low — this is presented only as an alternative not chosen; if picked later, verify against Pebble's own concurrency docs before adopting |
| A2 | No `internal/mcp` test currently exercises multi-call caching behavior (confirmed via grep — no cache exists to test) | Corrections #1 | Low — grep-verified absence, not an assumption about behavior |
| A3 | D-16's token measurements (`files` = 28,506 B JSON → ~16,835 B markdown, -41%; TOON 13,740 B, -52%) are accurate — **[CITED: coordinator measurement, not independently re-run in this session]** | § SURF-06 / D-16 | Low — the *decision* (markdown over JSON) rests on three other independent rationales (no parser consumer, closes a TS divergence, dissolves the D-12 notice problem), any one of which carries it. A wrong percentage would not change the choice; the ~41% figure is directionally corroborated by the structural fact (verified) that JSON re-states 4 keys × 308 records |
| A4 | The markdown table shape recommended in § SURF-06 is more legible to a model than the JSON it replaces — asserted from the record-list structure, **not** measured against a model in this session | § SURF-06 (Recommended markdown shape) | Low-Medium — if wrong, the fix is a shape change inside the new `Render*` funcs (a bounded, single-file edit), not an architectural rework; D-16 explicitly leaves exact table-vs-list to Claude's Discretion |

**If this table is empty:** N/A — two low-risk assumptions logged above,
both about an alternative path this research does not recommend adopting
without further validation.

## Open Questions

1. **dbSizeBytes: WalkDir (locked, D-07) or extend `graphstore` for
   `Metrics().DiskSpaceUsage()`?**
   - What we know: both are viable; WalkDir needs zero interface changes
     and stays fully within D-04a's pebble-confinement boundary trivially
     (it never imports pebble); `Metrics()` is cheaper and more
     "Pebble-idiomatic" but requires extending `GraphStore`/`Reader` and
     giving `Engine` access to the store, not just its snapshot.
   - What's unclear: whether the plan should invest in that extra plumbing
     this phase or defer it.
   - Recommendation: **keep D-07's WalkDir approach** for this phase (it is
     the locked decision, and the plumbing cost of the alternative is real);
     document the Pebble `Metrics()` API's existence in a code comment as a
     future optimization path, not a blocker.

2. **Where does the worktree-mismatch cache actually live for MCP?**
   - What we know: an `Engine`-scoped `sync.Once` (D-13's literal text)
     provides zero cross-call benefit because `Engine` is rebuilt per MCP
     call (Corrections #1).
   - What's unclear: exact shape of the server-scoped cache (a field on a
     new small struct returned by `BuildServer`, vs. a package-level
     `sync.Map`, vs. threading a `*gitmeta.CachingDetector` through every
     handler closure) — this is squarely "Claude's Discretion" territory
     CONTEXT.md already grants for `internal/gitmeta`'s function signatures,
     just needs the planner to actually *use* that discretion here rather
     than defaulting to the (non-functional for MCP) Engine-only cache D-13's
     prose literally describes.
   - Recommendation: add a small cache type to `internal/gitmeta` itself
     (e.g. `type CachingDetector struct{ mu sync.Mutex; cache map[string]*Mismatch }`
     with a `Detect(startPath, indexRoot string) *Mismatch` method), have
     `internal/mcp.BuildServer` construct one and close over it in every
     handler; have the CLI construct one per invocation too (free, since
     it's one-shot) so `Engine`/CLI code paths share the exact same type.

3. **~~Do the 5 JSON-shaped MCP tools get a literal text-prefix notice
   (byte-parity with TS) or a new JSON field (preserves JSON-parseability)?~~
   — ★ ANSWERED 2026-07-15: NEITHER. Escalated to the user → became SURF-06
   (D-16). The 5 tools move to markdown; the JSON shape is deleted, so the
   question is moot and the `_worktreeNotice` field hybrid is withdrawn. D-12
   is now one uniform text-prefix across all 7 non-status read tools. The
   original analysis is preserved below for traceability; see § SURF-06 for
   the live guidance.**
   - What we know: `node`/`explore`/`status` are markdown-or-JSON-with-a-
     known-shape already; `callers`/`callees`/`impact`/`search`/`files` are
     currently *always* valid JSON on their MCP surface.
   - What's unclear: whether an agent client depends on that always-JSON
     property strongly enough that breaking it (even only in the rare
     mismatch case) is unacceptable, vs. WORK-02's literal wording ("every
     other read tool... prefixes a compact single-line notice") intending
     exactly the TS mechanism regardless of shape.
   - Recommendation: implement the literal TS-matching text-prefix
     mechanism (satisfies WORK-02's wording and D-12's explicit "mirror
     `withWorktreeNotice`" instruction most directly), but **document the
     JSON-validity caveat explicitly** in `internal/mcp/tools.go`'s doc
     comments and in the phase's SUMMARY, so it's a recorded, deliberate
     choice rather than a silently-introduced behavior change. This
     question should be surfaced to the user during `/gsd-plan-phase` if
     the planner has any doubt — it's a genuine product-behavior fork, not
     a pure implementation detail.

4. **Exact CLI presentation for the compact notice on the 6 non-status
   commands (no TS precedent exists to copy).**
   - What we know: TS's CLI never wires `worktreeMismatchNotice` into any
     command besides `status` (verified via grep — zero other call sites);
     this is entirely new Go-side design work, gated to human-output mode
     only (Pitfall 4/TS's own `--json` precedent).
   - What's unclear: exact line placement (before vs. after the command's
     main output), and whether it goes to stdout (matching TS's `warn()` =
     `console.log`) or stderr.
   - Recommendation: stdout, printed once immediately after `OpenAt`
     succeeds and before the command's main output — mirrors `status`'s own
     placement (`Project:` line, then the warning, per D-09's target
     structure) and keeps MCP's JSON-RPC stdout framing concern (HYG-02,
     future phase) irrelevant to the CLI surface. This is "Claude's
     Discretion" per CONTEXT.md; recorded here as a recommendation, not a
     requirement.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `git` CLI | `internal/gitmeta`'s `os/exec` calls (WORK-01/02/03) + TEST-02 fixtures | ✓ | 2.55.0 (verified via `git --version`) | Best-effort design already handles absence (WORK-03) — any git failure/missing binary degrades to "no mismatch," and TEST-02 fixtures `t.Skip` (not `t.Fatal`) when git is unavailable, per the project's existing `resolveColbymchenryCorpus` convention |
| Go toolchain | Everything | ✓ | go1.26.5 darwin/arm64 | — |
| `github.com/cockroachdb/pebble/v2` | Status DB-size computation | ✓ | v2.1.6 (in `go.mod`, resolves via `go doc`) | — |

**Missing dependencies with no fallback:** none.

**Missing dependencies with fallback:** none — `git` itself has WORK-03's
best-effort fallback built into the locked design.

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` (`go test`) — no third-party test framework in this repo |
| Config file | none — no `go.mod` test-tool config; standard `go test` |
| Quick run command | `go test ./internal/gitmeta/... ./internal/query/... -run "Worktree\|Status\|DiskUsage" -v` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| STAT-01 | `status` reports Pebble on-disk DB size | unit | `go test ./internal/query/... -run TestStatus -v` | ❌ Wave 0 — extend `internal/query/files_status_test.go` or add `status_test.go` |
| STAT-02 | `status` reports nodes-by-kind + files-by-language | unit | `go test ./internal/query/... -run TestStatus -v` | ❌ Wave 0 — same file |
| STAT-03 | `status` reports live stale/reindexRecommended (already partially covered) | unit | `go test ./internal/query/... -run TestStatusStaleness -v` | ✅ `internal/query/status_staleness_test.go` exists — extend for the sectioned-output reachability check |
| WORK-01 | Worktree/index mismatch detected via the 4-gate cascade | unit | `go test ./internal/gitmeta/... -v` | ❌ Wave 0 — package doesn't exist yet |
| WORK-02 | Verbose warning (`status`) + compact notice (6 other tools, CLI+MCP) | integration | `go test ./internal/mcp/... -run TestNotice -v` and `go test ./internal/cli/... -run TestNotice -v` | ❌ Wave 0 |
| WORK-03 | Best-effort, never-blocking, no false positives | unit | `go test ./internal/gitmeta/... -run TestNoFalsePositive -v` | ❌ Wave 0 |
| TEST-02 | Six fixture layouts (linked-worktree, submodule, nested-clone, monorepo-subdir, `.claude/worktrees/`, symlinked) | integration | `go test ./internal/gitmeta/... -run TestFixture -v` | ❌ Wave 0 — no repo-building test helper exists anywhere in this codebase yet (closest precedent: `testdata/golden/golden_parity_test.go`'s `resolveColbymchenryCorpus`, which only *clones*, never *builds*, a repo) |
| SURF-06 | The 5 MCP tools (`callers`/`callees`/`impact`/`search`/`files`) return markdown, not JSON | unit | `go test ./internal/query/... -run TestRender -v` | ❌ Wave 0 — new `Render*` funcs + their tests |
| SURF-06 | Each of the 5 MCP tools' **success payload** is markdown end-to-end (not just the renderer in isolation) | integration | `go test ./internal/mcp/... -run TestMarkdownOutput -v` | ❌ Wave 0 — **zero existing coverage, see the blind spot below** |
| SURF-06 | CLI `--json` still emits valid JSON for all 5 commands (the regression guard) | integration | `go test ./internal/cli/... -run "TestSearchCmd\|TestCallersCalleesCmd\|TestImpactCmd\|TestFilesCmd" -v` | ✅ `internal/cli/query_cli_test.go` — these already `json.Unmarshal` the CLI output and MUST keep passing untouched |

### ★ SURF-06 test-surface analysis (CONTEXT.md D-16 explicitly asks for this)

**Which tests assert MCP text content:**

| File | Test | Tool exercised | Asserts | Breaks under SURF-06? |
|---|---|---|---|---|
| `internal/mcp/server_test.go:176` | `TestExploreHandlerDelegatesToEngine` | `codegraph_explore` | text contains `"**Exploration:"` | **No** — explore is already markdown, untouched |
| `internal/mcp/server_test.go:224` | `TestOpenEnginePathConfinedToRepoRoot` | `codegraph_status` | **error path only** (`IsError`, text contains `"outside"`) | **No** — asserts a refusal message, never the success payload |
| `internal/mcp/reconnect_test.go:24` | `TestReconnectReconcile` | `codegraph_explore` | text content | **No** — explore, already markdown |
| `testdata/golden/golden_parity_test.go:1352` | `TestExploreCLIMatchesMCP` | `codegraph_explore` | CLI output == MCP output | **No** — explore |
| `testdata/golden/golden_parity_test.go:1395` | `TestNodeCLIMatchesMCP` | `codegraph_node` | CLI output == MCP output | **No** — node, already markdown |
| `testdata/golden/golden_parity_test.go:1447` | `TestNodeLineHintCLIMatchesMCP` | `codegraph_node` | CLI output == MCP output | **No** — node |

**Which tests assert the CLI `--json` shape (must keep passing, unchanged):**

| File | Test | Asserts |
|---|---|---|
| `internal/cli/query_cli_test.go:82` | `TestSearchCmd` | `json.Unmarshal` of `search --json` |
| `internal/cli/query_cli_test.go:100` | `TestCallersCalleesCmd` | `json.Unmarshal` of `callers`/`callees --json` |
| `internal/cli/query_cli_test.go:154` | `TestImpactCmd` | `json.Unmarshal` of `impact --json` |
| `internal/cli/query_cli_test.go:217` | `TestFilesCmd` | `json.Unmarshal` of `files --json` |
| `internal/query/traverse_test.go:616,632` | (subtests) | `MarshalCalleesJSON`/`MarshalCallersJSON` output directly |
| `internal/query/files_status_test.go:287,375` | (subtests) | `MarshalFilesJSON`/`MarshalStatusJSON` output directly |
| `testdata/golden/golden_parity_test.go:635` | `TestGoldenParity/status` | `MarshalStatusJSON` shape vs. the frozen TS oracle |

**★ THE BLIND SPOT (this is what D-16's parity note is warning about, and it
is worse than expected):**

> **Not one existing test asserts the MCP success-payload text of ANY of the
> 5 tools SURF-06 changes.** MCP coverage is `explore` (markdown) + `status`
> (error path only). `callers`/`callees`/`impact`/`search`/`files` have
> **zero** MCP-surface assertions of any kind.

Consequences the planner must design around:

1. **SURF-06 could be implemented wrong — or not implemented at all — and
   `go test ./...` stays fully green.** There is no failing test to drive the
   change, and no test to catch a future regression back to JSON. This is a
   TDD-mode phase (`tdd_mode: true` in `.planning/config.json`), so the new
   `internal/mcp` markdown assertions should be written **first**, as the
   red test that SURF-06 turns green.
2. **The risk is inverted from the intuition.** One might fear "we'll break
   the CLI `--json` contract by mutating a shared helper." In fact the CLI
   path is *well* guarded (7 tests across 3 files would catch it instantly) —
   so the shared-helper hazard is loud, not silent. The genuinely silent
   failure mode is the MCP side going unverified, in either direction.
3. **Recommended new test (Wave 0):** an `internal/mcp` table-driven test
   that `CallTool`s each of the 5 tools against the existing `copyFixture` +
   `indexFixture` harness (already in `server_test.go`) and asserts the text
   is markdown **and is NOT valid JSON** — i.e. assert
   `json.Unmarshal([]byte(text.Text), &any{})` **fails** *and* the text
   contains the expected markdown marker (e.g. a `| Name | Kind |` header
   row). The negative JSON assertion is what makes a silent regression to
   `json.Marshal` impossible; the positive marker assertion is what makes an
   empty/garbage render impossible. Both are needed — either alone is
   defeatable.
4. **Do not extend `TestExploreCLIMatchesMCP` to the 5 tools.** That harness
   asserts *CLI output == MCP output*, which is exactly what SURF-06 makes
   **intentionally false** for these 5 (CLI JSON ≠ MCP markdown). The
   CLI==MCP byte-identity invariant holds only for `explore`/`node`, whose
   CLI surface has no `--json` mode at all (`internal/cli/{explore,node}.go`
   — "Emits markdown text only — no --json, per D-01b"). Codifying that
   distinction in a comment would prevent a future contributor from
   "restoring parity" by breaking SURF-06.

### Sampling Rate

- **Per task commit:** `go test ./internal/gitmeta/... ./internal/query/... ./internal/mcp/... ./internal/cli/...`
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full suite green, plus `go test ./testdata/golden/...` (golden parity, including the D-08 `dbSizeBytes` exemption) before `/gsd-verify-work`

### Wave 0 Gaps

- [ ] `internal/gitmeta/` package scaffolding — does not exist yet (worktree.go, detect.go, notice.go + tests)
- [ ] `internal/gitmeta/*_fixtures_test.go` (or equivalent) — the six-layout git-repo-building test helper (TEST-02); no existing precedent in this codebase builds a real git repo from scratch with `os/exec`, only clones one
- [ ] `internal/query/status_test.go` (or extend `files_status_test.go`) — `DbSizeBytes`/`FilesByLanguage` assertions
- [ ] `internal/mcp/server_test.go` extension — server-scoped mismatch cache behavior (Open Questions #2)
- [ ] `testdata/golden/golden_parity_test.go` line ~651 — the `dbSizeBytes` volatility exemption (Pitfall 2) plus a plausibility assertion
- [ ] `testdata/golden/README.md` — new row in the volatile-fields table recording the D-08 exemption + rationale
- [ ] **SURF-06:** `internal/query/render_results.go` (or extend `render_markdown.go`) — `renderLocationTable` + the 5 `Render*` funcs (4 share the `Location` shape; `files` needs both a flat table and a tree form)
- [ ] **SURF-06:** `internal/query/render_results_test.go` — renderer unit tests (empty sets, single row, multi-row, tree format, order preservation)
- [ ] **SURF-06:** `internal/mcp/server_test.go` extension — the 5-tool markdown assertion described in the blind-spot section above (**write first — TDD red test**; nothing exists today)
- [ ] **SURF-06 / Corrections #5:** `internal/query` — an MCP-form status markdown renderer (TS `handleStatus`'s bolded-key + bullet-list shape), distinct from D-09's CLI padded-column form; D-12's blockquote warning requires it

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | No auth surface in this phase |
| V3 Session Management | no | N/A |
| V4 Access Control | no | Not touched — `confineToRepoRoot`'s existing MCP path-confinement boundary (CR-02, prior phase) is unchanged by this phase and continues to apply before `OpenAt` is ever called |
| V5 Input Validation | yes | `startPath`/`indexRoot` values passed to `exec.CommandContext` as the subprocess's `Dir`, never as shell-interpolated arguments — `exec.Command("git", "rev-parse", ...)` with a fixed argv and `cmd.Dir` set separately is not vulnerable to shell injection (no shell is invoked; `os/exec` execs the binary directly) |
| V6 Cryptography | no | N/A |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Argument/command injection via a malicious `--path` or MCP `path` arg reaching `git rev-parse` | Tampering | Not exploitable here: `exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")` with `cmd.Dir = dir` never shells out through `/bin/sh -c`, so `dir` cannot break out of being a plain working-directory argument regardless of its contents; git itself will simply fail (→ "no mismatch") on a bogus path |
| Unbounded subprocess hang (DoS) if git hangs (e.g. a corrupted repo, a network-mounted `.git`, an interactive credential prompt) | Denial of Service | The 5s `exec.CommandContext` timeout (D-03, ported from TS #1139's own documented fix for exactly this) — already locked; `--stdio ignore` for stdin prevents git from blocking on an interactive prompt in the first place |
| Information disclosure: the verbose warning embeds absolute filesystem paths (`worktreeRoot`, `indexRoot`) in tool output | Information Disclosure | Accepted/intentional — this is the whole point of the warning (telling the agent/human exactly which two trees are involved); no different in risk profile from `status`'s pre-existing `projectPath`/`indexPath` fields, which the codebase already renders as empty strings in the MCP-exposed JSON (per `status.go`'s existing decision table) specifically to avoid leaking host paths through that surface — **verify the new worktree paths get the same treatment or a documented exception**, since `worktreeMismatchWarning`/`worktreeMismatchNotice` interpolate raw absolute paths that `StatusResult.ProjectPath`/`IndexPath` deliberately do NOT |

**Flag for planner:** the existing `StatusResult.ProjectPath`/`IndexPath`
fields are deliberately rendered as empty strings in JSON output
specifically to avoid leaking host-local absolute paths through the MCP
surface (`status.go`'s decision table: *"Engine carries no path context in
its read-only Reader-only design... trivially satisfies T-03-05-Leak by
having nothing host-specific to leak"*). The new worktree warning/notice
strings **do** interpolate absolute host paths
(`m.worktreeRoot`/`m.indexRoot`) by design — this is a deliberate,
necessary divergence from that prior privacy stance (the warning is
useless without the paths), but the plan should explicitly note this is an
intentional, scoped exception to the codebase's existing "no host paths in
MCP output" convention, not an oversight.

## Sources

### Primary (HIGH confidence)

- `/opt/homebrew/lib/node_modules/@colbymchenry/codegraph/node_modules/@colbymchenry/codegraph-darwin-arm64/lib/dist/sync/worktree.js` — read in full this session; confirmed `codegraph --version` → `1.3.1` live on PATH
- `…/lib/dist/bin/codegraph.js` (lines 240-340, 793-969) — `formatNumber`, `warn`, the full `status` command body
- `…/lib/dist/mcp/tools.js` (lines 1020-1120, 1220-1290, 3860-3960) — `worktreeMismatchFor`, `withWorktreeNotice`, `handleStatus`
- This codebase's own source, read directly: `internal/query/{status,engine,resolve,render_markdown,traverse,files,search}.go`, `internal/cli/{status,callers,callees,impact,files,search,node,explore,query}.go`, `internal/mcp/{tools,server}.go`, `internal/graphstore/{store,pebble_store}.go`, `internal/schema/graph.pb.go`, `internal/indexer/{languages,discover}.go`
- **SURF-06 test-surface analysis (2026-07-15):** `internal/mcp/{server_test,reconnect_test}.go`, `internal/cli/query_cli_test.go`, `internal/query/{traverse_test,files_status_test}.go`, `testdata/golden/golden_parity_test.go` — all read directly; plus an exhaustive grep of every `Marshal*JSON` call site across the module, which is what established the CLI/MCP helper sharing and the zero-MCP-coverage blind spot
- `go doc github.com/cockroachdb/pebble/v2 DB.Metrics / Metrics / DB.EstimateDiskUsage` and `$GOMODCACHE/github.com/cockroachdb/pebble/v2@v2.1.6/metrics.go`'s `DiskSpaceUsage()` source — read directly against the pinned `v2.1.6` module in this repo's module cache
- `testdata/golden/{golden_test.go,golden_parity_test.go,README.md}` — read directly for the D-08 volatile-key exemption location and mechanism

### Secondary (MEDIUM confidence)

- `.planning/phases/02-status-content-git-worktree-awareness/02-CONTEXT.md` — the locked decisions this research verifies against and builds on
- `.planning/REQUIREMENTS.md` §STAT/WORK/TEST — requirement text cross-checked against both TS source and current Go source

### Tertiary (LOW confidence)

- None — every claim in this document traces to a directly-read source file or a directly-run tool command (`go doc`, `git --version`, a `python3` hexdump script) this session.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependencies; stdlib-only, verified against the pinned `go.mod`/`go doc` output
- Architecture: HIGH — every claim about current Go code structure (Engine, OpenAt, MCP handler shapes, GraphStore boundary) is a direct source read this session, not inference
- Pitfalls: HIGH — the JSON-vs-text-prefix and Engine-cache findings are derived from direct code inspection (grep for cache usage, reading `openEngine`'s doc comment), not speculation
- SURF-06 (added 2026-07-15): HIGH — the shared-helper table, the test-surface tables, and the "status is JSON not markdown" correction are each an exhaustive grep or a direct file read with cited line numbers; the only [CITED]-not-[VERIFIED] item is D-16's byte measurements (28,506 B → ~16,835 B), taken from the coordinator's own measurement and not independently re-run here

**Research date:** 2026-07-15 (extended same day for the SURF-06 scope change)
**Valid until:** 30 days (stable domain — stdlib `os/exec`/`filepath`, a pinned Pebble version, and a frozen TS 1.3.1 reference source; re-verify only if `go.mod`'s `pebble/v2` version bumps or the TS reference package is pruned per D-01's "capture anything still needed before a future npm prune" note)
