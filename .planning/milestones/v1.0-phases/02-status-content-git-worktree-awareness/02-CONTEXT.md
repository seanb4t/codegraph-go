# Phase 2: status Content & Git/Worktree Awareness - Context

**Gathered:** 2026-07-15
**Status:** Ready for planning

<domain>
## Phase Boundary

Two independent-but-co-shipped capabilities, both landing in the shared
`internal/query` read seam so the CLI command and the MCP tool improve in the
same commit:

- **`status` content (STAT-01/02/03):** replace our terse one-line
  `backend=… files=… nodes=…` with TS 1.3.1's full status *content* — Pebble
  on-disk DB size, a nodes-by-kind breakdown, a files-by-language breakdown,
  and the live pending-changes / reindex-recommended signal — instead of the
  v0.1 Phase-3 inert placeholders and the golden-corpus DB-size strip.
- **Git/worktree awareness (WORK-01/02/03):** detect that the resolved
  `.codegraph/` index belongs to a *different* git working tree than the caller
  (the silent "worktree queries the main branch's graph" correctness bug),
  print a verbose warning from `status`, and prefix every other read tool
  (CLI + MCP) with a compact one-line notice via a shared `withWorktreeNotice`
  equivalent. Detection is best-effort and never blocks a query.
- **TEST-02 fixtures:** worktree detection has passing fixtures for
  linked-worktree, submodule, nested-clone, monorepo-subdir,
  `.claude/worktrees/`, and symlinked layouts.
- **MCP output shape (SURF-06 — pulled in from Phase 8, user decision
  2026-07-15):** the 5 JSON-shaped MCP read tools
  (`callers`/`callees`/`impact`/`search`/`files`) switch from raw
  `json.Marshal` output to markdown, matching the 3 tools that already do
  (`explore`/`node`/`status`) and matching TS (which returns markdown from
  *every* MCP tool). CLI `--json` is unaffected. See D-16.

**Requirements:** STAT-01, STAT-02, STAT-03, WORK-01, WORK-02, WORK-03,
TEST-02, SURF-06 (8 total).

**Hard constraints for this phase:**
- **Plain text only, content-not-color.** Phase 6 ("Rendering Seam & Pretty
  status/files") owns lipgloss/TTY colorization (TUI-02). Phase 2 delivers the
  *sections, fields, ordering, and wording*; Phase 6 paints them. Do not
  introduce `charm.land/*` imports here.
- **Shared engine.** Worktree awareness is computed in `internal/query` (one
  code path) so CLI + MCP both get it structurally, not via duplicated
  plumbing — per the ROADMAP note.
- **Best-effort, never blocking (WORK-03).** Any git failure, missing git,
  timeout, or non-repo path ⇒ "no mismatch", query proceeds unchanged.

**Not in this phase:** watcher-on-MCP default (Phase 3), Pebble WAL/log stderr
noise — HYG-01 (Phase 4), git sync hooks — HOOK-* (Phase 5), any
colorization/TTY-gating/lipgloss of `status` — TUI-02 (Phase 6), `impact`
depth / short-flag aliases / `affected` flags — SURF-01..05 (Phase 8).
**Note:** SURF-06 (MCP markdown output) IS in this phase — pulled forward from
Phase 8 by explicit user decision on 2026-07-15 while resolving 02-RESEARCH.md
Open Question #3. SURF-01..05 remain Phase 8.

**Explicitly out of scope (REQUIREMENTS.md "Out of Scope" table):** the exact
`pendingChanges` added/modified/removed **count** at `status`-time — it would
require re-running Sync's diff on every `status` call. STAT-03's live
`stale` / `reindexRecommended` signal is the v1.0 bar. See D-06.

</domain>

<decisions>
## Implementation Decisions

### ★ TS ground truth is AVAILABLE — memory's "dist vanished" was a relocation

- **D-01:** **The TS 1.3.1 white-box source is intact and MUST be read for
  verbatim constants/strings** — correcting memory `9zt8afrs8k`'s "the live TS
  1.3.1 dist VANISHED mid-execution (only .d.ts left)" gotcha. The top-level
  `…/@colbymchenry/codegraph/dist/` does indeed hold only `.d.ts` stubs, but
  the **real implementation was relocated into the platform sub-package**:
  `…/@colbymchenry/codegraph/node_modules/@colbymchenry/codegraph-darwin-arm64/lib/dist/`
  (195 `.js` files). `codegraph --version` → `1.3.1` and the binary is live on
  PATH. Phase 2 therefore does **white-box constant extraction like Phase 1's
  D-01**, not frozen-golden archaeology. Every string/constant below was read
  from that tree this session — planner/researcher re-verify against it, do not
  paraphrase. **Capture anything still needed before a future npm prune.**

### Worktree detection algorithm (WORK-01/WORK-03)

- **D-02:** **Port TS `sync/worktree.js`'s 4-gate cascade verbatim** into the new
  `internal/gitmeta` package. ⚠️ **The requirement text's shorthand
  ("`--show-toplevel` vs `--git-common-dir`") materially understates the real
  algorithm, and a naive reading inverts one gate.** The actual TS logic in
  `detectWorktreeIndexMismatch(startPath, indexRoot)`:
  1. `worktreeRoot := gitWorktreeRoot(startPath)`; if null ⇒ **no mismatch**
     (not a repo / no git).
  2. `resolvedIndexRoot := realpath(indexRoot)`; if `worktreeRoot ==
     resolvedIndexRoot` ⇒ **no mismatch** (index is our own tree).
  3. **`gitWorktreeRoot(resolvedIndexRoot) != resolvedIndexRoot` ⇒ no
     mismatch.** The index root must *itself* be a working-tree root. This is
     the gate that kills monorepo-subdir + non-git + plain-ancestor false
     positives.
  4. `worktreeCommon := gitCommonDir(worktreeRoot)`;
     `indexCommon := gitCommonDir(resolvedIndexRoot)`; **if both non-null AND
     they DIFFER ⇒ no mismatch.** Note the polarity: a *genuine* borrowed
     worktree **shares** a common dir; a submodule / embedded clone is a
     **different repository** and is therefore *suppressed*. (TS refs #1031,
     #1033.)
  5. Otherwise ⇒ mismatch `{worktreeRoot, indexRoot}`.
- **D-03:** **Git invocation contract mirrors TS exactly:** `os/exec` only
  (stdlib — no go-git, per `p82eny7gf5`), `git rev-parse --show-toplevel` and
  `git rev-parse --git-common-dir`, `cwd` = the probed dir, **5s timeout**
  (TS's `timeout: 5000`, added because an unbounded hang trips the daemon's
  60s liveness watchdog — TS #1139; use `exec.CommandContext`), **stderr
  discarded**, trimmed stdout, and **any error/empty ⇒ null**. `--git-common-dir`
  is **relative to cwd unless already absolute** — resolve it against the probed
  dir before realpath. **`EvalSymlinks` both sides** (TS `realpathSync`), and on
  `EvalSymlinks` error fall back to the plain absolute path (TS's `catch { return
  path.resolve(p) }`) rather than failing.
- **D-04:** `internal/gitmeta` exposes the primitives (`WorktreeRoot`,
  `CommonDir`, `DetectIndexMismatch`) and stays **free of query/render
  concerns** so Phase 5's git sync hooks (HOOK-*) can reuse it. It is a
  **new package** per `p82eny7gf5`'s locked seam.

### Status content (STAT-01/02/03)

- **D-05:** ★ **`filesByLanguage` is a genuinely NEW computation — the
  requirement's parenthetical is wrong.** STAT-02 says the data is "already
  computed in `StatusResult` — surface it". That is **true for
  `NodesByKind map[string]int64`** (real counts, already scanned) but **false
  for languages**: `StatusResult.Languages` is a bare **`[]string`** with no
  counts. TS's own JSON does the same — it *derives* the flat list **from** the
  count map and throws the counts away:
  `languages: Object.entries(stats.filesByLanguage).filter(([,c]) => c > 0).map(([lang]) => lang)`.
  The counts only ever appear in *rendered* output. **Decision:** add
  `FilesByLanguage map[string]int64` to `StatusResult`, computed from the file
  scan; keep `Languages []string` **derived from it** (`count > 0`, existing
  order/shape) so the golden JSON shape stays parity-stable; render the counts
  in the human output. Planner: this is an extra scan/aggregation, not a
  render-only change — size the task accordingly.
- **D-06:** **STAT-03 is largely already satisfied — verify, don't rebuild.**
  v0.1's D-04a already made the top-level `stale` field **live**
  (`computeStale`: `.codegraph/.sync-pending` sidecar OR newest source mtime >
  `Meta.last_sync_unix_ms`), and `index.reindexRecommended` is already **derived**
  (`!schema.IsCurrentSchemaVersion(meta)`), not a placeholder. Both are already
  printed by our terse CLI line. STAT-03's remaining work is therefore
  **surfacing them in the new sectioned layout + rendering TS's
  pending/up-to-date/reindex advisory lines**, plus a
  reachability check that they're live on **both** CLI and MCP. `PendingChanges`
  **stays the inert all-zero placeholder** (explicit REQUIREMENTS out-of-scope
  row); render the advisory from `stale`, not from a count.
  ⚠️ This is exactly the CR-02 "implemented + unit-tested ≠ delivered" trap from
  memory `9zt8afrs8k` — trace `stale` end-to-end through both surfaces.
- **D-07:** **`dbSizeBytes` = recursive byte sum of the Pebble store dir**
  (`.codegraph/store/` — SSTables + WAL + MANIFEST), via a `filepath.WalkDir`
  sum, best-effort (skip unreadable entries rather than erroring the whole
  `status`). Pebble has no single-file analog to SQLite's page count; the
  directory sum is the honest Go-truthful reading of TS's
  `stats.dbSizeBytes`. **Key name `dbSizeBytes` (raw bytes) in `--json`**
  matching TS; **human output renders `(bytes / 1024 / 1024).toFixed(2)` MB**
  matching TS's `DB Size:   X.XX MB`.
- **D-08:** **Reverse the golden strip for the Go side ONLY, and assert
  presence-and-plausibility rather than byte-equality.** `testdata/golden/README.md`
  strips `dbSizeBytes` from `status.json` as volatile ("SQLite WAL/page-
  fragmentation dependent; not guaranteed byte-stable across reindexes even of
  identical source") and `golden_test.go` encodes that as an invariant. That
  rationale is **even stronger for Pebble** (LSM compaction makes the on-disk
  byte total genuinely nondeterministic across identical reindexes). So:
  **keep the TS golden's strip** (the frozen TS oracle cannot supply a stable
  byte value), and assert Go's `dbSizeBytes` as a **documented Phase-1 D-02
  allowed divergence**: key present, integer, `> 0`, and MB-rendering
  well-formed (`^\d+\.\d{2} MB$`). Do **not** assert cross-run byte stability.
  Planner MUST check whether `golden_test.go`'s "no volatile field" invariant
  needs a narrowly-scoped exemption for this key and update
  `testdata/golden/README.md`'s table to record the reversal + its rationale.

### Status layout: adopt TS's sections now, colorize in Phase 6

- **D-09:** **Adopt TS's sectioned plain-text layout now** (content + section
  headers + ordering + wording), leaving *only* ANSI color for Phase 6. Our
  current single-line `backend=… files=… stale=…` is not extendable to the
  required content, and adding fields to it would guarantee a second rewrite at
  Phase 6. Target structure, from `bin/codegraph.js`:
  ```
  CodeGraph Status

  Project: <path>
  [verbose worktree warning, if any]
  [index-state / pendingRefs advisories, if any]

  Index Statistics:
    Files:     <n>
    Nodes:     <n>
    Edges:     <n>
    DB Size:   <X.XX> MB
    Backend:   pebble
  Nodes by Kind:
    <kind padEnd(15)> <count>
  Files by Language:
    <lang padEnd(15)> <count>
  [Pending Changes: … | Index is up to date]
  [reindex advisory, if reindexRecommended]
  ```
  **Both breakdowns: filter `count > 0`, sort by count DESC, `padEnd(15)` on the
  key.** `Journal:` is **dropped** (no Pebble analog — consistent with the
  existing `journalMode` drop in `StatusResult`'s decision table). `Backend:` is
  the Go-truthful `pebble`, not TS's `node:sqlite … (full WAL)`.
- **D-10:** **`formatNumber` is a parity trap — hand-roll comma grouping, do not
  add a dep.** TS uses `n.toLocaleString()`, which is **locale-dependent** (`1,223`
  in en-US; `1.223` / `1 223` elsewhere). Go has no stdlib equivalent.
  **Decision:** implement a small fixed **en-US-style comma grouper** in
  `internal/query` (or the presenter) — deterministic, locale-independent,
  matching TS's default-locale CI behavior. Do **not** pull in
  `golang.org/x/text/message` for this (new dep, and it re-introduces locale
  variance). Record as a documented D-02 divergence: we pin en-US grouping where
  TS follows the host locale.

### Notice delivery + verbatim strings (WORK-02)

- **D-11:** **Two distinct renderings, ported verbatim** from `sync/worktree.js`:
  - **Verbose** (`status` only) — `worktreeMismatchWarning`:
    ```
    This CodeGraph index belongs to a different git working tree.
      Running in: <worktreeRoot>
      Index from: <indexRoot>
    Results reflect that tree's code (often a different branch), not this worktree — symbols changed only here are missing. Run "codegraph init -i" in this worktree for a worktree-local index.
    ```
  - **Compact** (every *other* read tool) — `worktreeMismatchNotice`:
    ```
    ⚠ CodeGraph results below come from a different git worktree (<indexRoot>), not where you're working (<worktreeRoot>) — they may reflect another branch, and symbols changed only here are missing. Run "codegraph init -i" here for a worktree-local index.
    ```
  ★ **The glyph is U+26A0 `⚠` (bytes `e2 9a a0`) with NO U+FE0F variation
  selector** — verified by hexdump this session. It is **not** the `⚠️` used by
  Phase 1's "no covering tests" warning. Getting this wrong is a silent
  byte-parity failure. It matches our existing `staleBannerText`'s `⚠`.
- **D-12:** **Mirror the `staleBanner` precedent — ONE uniform text-prefix across
  all 7 non-status read tools.** TS's
  `withWorktreeNotice` prefixes `${notice}\n\n${first.text}` onto the first text
  content block, **no-ops on `isError` results**, and **excludes
  `codegraph_status`** (which embeds its own verbose form). Our
  `staleBanner(stale)` already establishes exactly this prepend-to-rendered-string
  pattern in `render_markdown.go`. **Decision:** implement
  `worktreeNotice(mismatch)` alongside it and apply the literal TS text-prefix
  to all 7 non-status read tools
  (`explore`/`node`/`search`/`callers`/`callees`/`impact`/`files`), with `status`
  taking the verbose form. **MCP `status` wraps the verbose warning as a
  blockquote** (`> ⚠ ` + warning with `\n` → `\n> `), matching TS's
  `mcp/tools.js`; the CLI prints it via its warn-style line.
  ⚠️ **TS's CLI `warn()` writes to `console.log` = STDOUT, not stderr** — match
  that for CLI parity, while keeping MCP's JSON-RPC stdout clean (diagnostics
  ride *inside* the tool result payload, never raw stdout — HYG-02's rule holds).
  **★ CORRECTED 2026-07-15 (post-research). A proposed `_worktreeNotice`
  JSON-field hybrid is WITHDRAWN — it rested on a false premise.** The idea was
  to protect `json.Unmarshal` on the 5 JSON-shaped tools by giving them a field
  instead of a text prefix. 02-RESEARCH.md Pitfall 1 + a direct source read
  killed it: those handlers do `mcp.NewToolResultText(string(json.Marshal(...)))`
  — they put JSON *inside a text block*, and **MCP text content is consumed by a
  language model, not a parser** (nothing in this repo, nor Claude Code,
  unmarshals it). The "contract" being protected had no consumer. D-16 moves
  those 5 tools to markdown regardless, so the question is moot: one shape, one
  mechanism. **Do not reintroduce a per-tool notice mechanism.**
- **D-13:** **Detection is computed once and cached (negative results included) —
  but the cache MUST NOT live only on `Engine`.** TS caches per `${startPath}\u0000${indexRoot}` and holds
  "that first verdict until restart" (#926), because detection costs **2 `git`
  subprocesses** and MCP is long-lived — re-probing on every tool call would be
  a real latency regression.
  **★ CORRECTED 2026-07-15 (post-research; 02-RESEARCH.md Corrections #1 +
  Open Question #2).** This decision originally read "resolve lazily-once
  (`sync.Once`) on the `Engine` … MCP gets the win." **That is FALSE:**
  `internal/mcp/tools.go`'s `openEngine` builds a **fresh `Engine` on every
  single tool call** by design, so an Engine-scoped cache yields **zero**
  cross-call benefit on the exact surface the cache exists for. **Corrected
  decision:** put the cache in `internal/gitmeta` itself (e.g. a
  `CachingDetector` — mutex-guarded `map[string]*Mismatch` + a
  `Detect(startPath, indexRoot) *Mismatch` method); `internal/mcp` constructs
  **one per server** and closes over it in every handler; the CLI constructs one
  per invocation (free — it's one-shot) so both surfaces share the identical
  type. Cache **negative results too** (nil == "checked, no mismatch", which must
  stay distinguishable from "not yet checked").

### Plumbing: the Engine must learn `startPath` (WORK-01)

- **D-14:** ★ **`OpenAt` currently DISCARDS the caller's start path — that is the
  load-bearing plumbing change of this phase.** `Engine{reader, repoRoot}` holds
  `repoRoot` = the **resolved index root** (the output of
  `ResolveCodegraphDir`'s upward walk), and `OpenAt(start)` throws `start` away
  after resolving. Worktree detection needs **both** sides
  (`detect(startPath, indexRoot)`) — TS deliberately captures
  `startPath = path.resolve(pathArg || process.cwd())` **before** the walk-up, and
  our `StatusResult` decision table even records that "Engine carries no path
  context in its read-only Reader-only design". **Decision:** retain the caller's
  start path on the `Engine` (e.g. `startPath`) in `OpenAt`, since `OpenAt` is
  "the single read seam CLI commands and MCP tool handlers both" use — that is
  precisely what delivers CLI + MCP worktree awareness **in one commit** per the
  ROADMAP note. Engines built via `New`/`NewWithRoot` (tests, no start path)
  must degrade to "no mismatch", never panic.

### Test fixtures (TEST-02)

- **D-15:** **Fixtures drive real `git` via `os/exec` in `t.TempDir()`** — no
  faked `.git` directories. The whole point of WORK-03 is that the gates behave
  correctly against git's *actual* `--show-toplevel` / `--git-common-dir`
  semantics (especially the submodule vs linked-worktree common-dir distinction,
  which a hand-built `.git` would not reproduce). Build all six layouts:
  linked-worktree (`git worktree add`), submodule (`git submodule add`),
  nested-clone (embedded clone, no gitlink), monorepo-subdir (plain subdir of
  one repo), `.claude/worktrees/<name>/` (the Sean-specific GSD layout that
  motivated this — a linked worktree *inside* the main checkout, which is the
  true-positive case), and symlinked paths (symlinked start dir and/or index
  root — the `EvalSymlinks` both-sides case). **`t.Skip` when `git` is absent**;
  set deterministic `GIT_*` env (`-c init.defaultBranch`, author/committer) so
  fixtures are hermetic and CI-stable.
  **Expected verdicts:** linked-worktree ⇒ **mismatch**; `.claude/worktrees/` ⇒
  **mismatch**; submodule ⇒ **no** mismatch (gate 4); nested-clone ⇒ **no**
  mismatch (gate 4); monorepo-subdir ⇒ **no** mismatch (gate 3); non-git ⇒
  **no** mismatch (gate 1); symlinked ⇒ **no** mismatch when it's really the
  same tree (gate 2 after `EvalSymlinks`).

### MCP output shape (SURF-06 — pulled in from Phase 8 by user decision)

- **D-16:** ★ **The 5 JSON-shaped MCP read tools switch to markdown**
  (`callers`/`callees`/`impact`/`search`/`files`), joining the 3 that already do
  (`explore`/`node`/`status`). **The CLI `--json` flag is UNTOUCHED and keeps
  emitting JSON.** Rationale, in priority order:
  1. **The consumer is a language model, not a parser.** Every one of those
     handlers currently does `mcp.NewToolResultText(string(json.Marshal(...)))`
     — JSON stuffed into a *text* block. Nothing unmarshals it. JSON is paying
     parser tax with no parser.
  2. **Token cost, measured on this repo's own index (not estimated):**
     `files` → **28,506 bytes JSON vs ~16,835 markdown = -41%**. ~14KB of that
     is the keys `path`/`language`/`nodeCount`/`edgeCount` repeated once per
     record across **308 records**. JSON is the worst shape for uniform record
     lists precisely because it re-states every key on every row.
  3. **It CLOSES a TS divergence rather than opening one.** TS returns markdown
     from *every* MCP tool; our JSON tools are the anomaly. This moves toward
     the drop-in-parity bar.
  **★ D-16 PREMISE CORRECTED 2026-07-15 (post-research).** This decision's first
  draft said the 5 tools would join "the 3 that already do
  (`explore`/`node`/`status`)". **That is factually WRONG: MCP `codegraph_status`
  emits JSON today** (`internal/mcp/tools.go:339-343` → `MarshalStatusJSON`).
  Only **2** MCP tools are markdown (`explore`, `node`); **6** are JSON. The
  error came from reading TS's markdown MCP-status renderer and assuming our Go
  side matched. Consequence: see **D-17** — MCP status needs its own markdown
  renderer, which D-12's blockquote requirement already implicitly demanded.
  **★ SURF-06 IS ADDITIVE, NOT A SWAP (research finding — highest-risk detail).**
  **Every one of the 5 `Marshal*JSON` helpers is SHARED with the CLI `--json`
  path** (e.g. `MarshalCallersJSON` ← MCP `tools.go:248` **and** CLI
  `callers.go:41`; `search` has no helper and inline-`json.Marshal`s on *both*
  surfaces). Therefore: **NEVER modify a `Marshal*JSON` body** — it is
  simultaneously the CLI contract and the golden oracle. Add **sibling `Render*`
  functions** and change **only the six `tools.go` call sites**.
  **★ TEST BLIND SPOT (research finding — treat as CR-02 recurrence risk).**
  **Zero existing tests assert the MCP success payload of ANY of these 5 tools**
  (MCP coverage today = `explore` markdown + `status` error-path only). SURF-06
  could be skipped wholesale and `go test ./...` would stay green — the exact
  "implemented/marked-complete but undelivered" trap from memory `9zt8afrs8k`.
  **Required TDD red test:** assert the MCP text **is not valid JSON** *AND*
  contains a markdown marker — either assertion alone is defeatable. **Do NOT
  extend `TestExploreCLIMatchesMCP` to these 5** — SURF-06 makes CLI≠MCP
  *intentionally* true for them.
  4. **It dissolves the D-12 notice problem** — with all 7 non-status read tools
     on one shape, a single text-prefix mechanism works everywhere, and no
     handler gets touched twice (Phase 2 is already rewiring all 7 result paths
     for WORK-02; doing the shape change in the same pass is strictly cheaper
     than a second visit in Phase 8).
  **Rejected alternatives (do not revisit without new evidence):** **TOON** —
  measured **13,740 bytes (-52%)**, a further ~11pt win over markdown, but it is
  a 2025-era format with thin model exposure; trading comprehension reliability
  for 11% on a tool whose entire value is agent legibility is a bad trade, and it
  adds an encoder + novel syntax. **YAML** — whitespace-fragile, barely cheaper
  for tabular data, no comprehension win over markdown. **TOML** — a config
  format; arrays-of-records are its weakness. **Bare path list** (-61%) — lossy,
  drops the counts.
  **Shape guidance:** these payloads are uniform record lists, so a **markdown
  table** (header row once, then rows) captures the win; keep it plain-text (no
  ANSI — Phase 6 owns color). Exact table vs. list-per-record formatting is
  Claude's Discretion; optimize for model legibility and stable ordering.
  **Parity note:** the golden/parity harness asserts the **CLI** `--json` shape —
  confirm which existing tests assert MCP text content and update them
  deliberately; do not let a JSON→markdown change silently pass a test that was
  only ever checking the CLI path.

- **D-17:** ★ **MCP `status` gets its OWN markdown renderer — TS ships TWO
  structurally different status renderings, and we need both.** Surfaced by
  research after D-16's premise error. This is **latent scope that D-12 already
  demanded**, not a new capability: D-12 requires MCP status to wrap the verbose
  warning in a **blockquote** (`> ⚠ …`), which is meaningless prepended to a JSON
  blob — so MCP status cannot stay `MarshalStatusJSON`. Verified in the TS
  source, the two renderings are genuinely different shapes:
  - **CLI** (`bin/codegraph.js` ~900-985) — padded columns:
    `Index Statistics:` / `  Files:     1,234` / `  DB Size:   1.23 MB`, with
    `padEnd(15)` breakdowns. This is **D-09's** target.
  - **MCP** (`mcp/tools.js` ~3890-3945) — bolded-key bullet lists:
    `**CodeGraph Status**`, `**Files indexed:** N`, `**Database size:** X.XX MB`,
    `**Nodes by Kind:**` + `- kind: count`, `**Languages:**` + `- lang: count`.
  **Decision:** implement both, sourced verbatim from their respective TS call
  sites. They share the same `StatusResult` data (STAT-01/02/03) but NOT the same
  renderer. Neither is covered by SURF-06's 5-tool scope (`status` is a 6th JSON
  tool) nor by D-09 (CLI-only) — plan it as its own task. Both stay plain-text
  (no ANSI — Phase 6 owns TUI-02). **`MarshalStatusJSON` itself is untouched** —
  the CLI `--json` path still uses it (same additive rule as D-16).

### Claude's Discretion
- File layout within `internal/query` for the status sections + the notice
  helper (extend `status.go`/`render_markdown.go` vs a new `render_status.go`),
  the internal shape of the `FilesByLanguage` aggregation, the exact
  `internal/gitmeta` function signatures, and the fixture-builder helper
  structure — planner/executor choose, so long as the plain-text-only,
  shared-engine, and never-blocking constraints hold.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope & requirements
- `.planning/ROADMAP.md` § "Phase 2: status Content & Git/Worktree Awareness"
  (lines 101–116) — goal, the 5 success criteria, and the `internal/gitmeta`
  ("stdlib `os/exec` only — two `git rev-parse` calls, no pure-Go git lib") +
  "validate the edge-case fixtures before any pretty rendering" notes.
- `.planning/REQUIREMENTS.md` § STAT-01..03, WORK-01..03, TEST-02 — the 7
  requirements this phase closes. **Also read the § "Out of Scope" table row for
  `pendingChanges` exact counts** — it is the guardrail behind D-06.
- `.planning/PROJECT.md` § Key Decisions — the "Worktree awareness scoped at
  parity for v1.0" row (detect + warn + notice only; auto-init/share deferred)
  and the "agent/MCP output stays plain" rule.
- `.planning/phases/01-behavioral-parity-explore-node/01-CONTEXT.md` § D-02 —
  the **normalized/structural parity oracle + documented allowed-divergence
  list**, which D-08 (dbSizeBytes) and D-10 (formatNumber locale) both invoke.

### TS 1.3.1 reference implementation (white-box source of truth — see D-01)
- `/opt/homebrew/lib/node_modules/@colbymchenry/codegraph/node_modules/@colbymchenry/codegraph-darwin-arm64/lib/dist/sync/worktree.js`
  — **ABSOLUTE external path.** The complete WORK-01/02/03 ground truth: the
  4-gate `detectWorktreeIndexMismatch` cascade, the 5s-timeout `git rev-parse`
  contract, `realpath` handling, and the verbatim `worktreeMismatchWarning` /
  `worktreeMismatchNotice` strings. **Note the top-level `…/codegraph/dist/` has
  only `.d.ts` stubs — the real `.js` is under this platform sub-package.**
- `…/codegraph-darwin-arm64/lib/dist/bin/codegraph.js` lines ~795–985 — the TS
  CLI `status` command: `detectWorktreeIndexMismatch(startPath, projectPath)`
  called **before** open, the `--json` key set (incl. `dbSizeBytes`, and
  `languages` derived from `filesByLanguage`), and the human "Index Statistics:
  / Nodes by Kind: / Files by Language: / Pending Changes:" layout with
  `padEnd(15)` + `formatNumber`. `formatNumber` (line ~247) = `toLocaleString()`
  (D-10); `warn` (line ~319) = `console.log` → **stdout** (D-12).
- `…/codegraph-darwin-arm64/lib/dist/mcp/tools.js` lines ~1040–1080 —
  `worktreeMismatchFor` (the `${startPath}\u0000${indexRoot}` cache, negative
  results cached, #926) and `withWorktreeNotice` (prefix `notice\n\n`, skip
  `isError`, exclude `codegraph_status`). Lines ~3880–3945 — the MCP `status`
  blockquote form (`> ⚠ …`) and its `**Nodes by Kind:**` / `**Languages:**`
  `count > 0` rendering.
- `…/codegraph-darwin-arm64/lib/dist/ui/glyphs.js` — the `UNICODE_GLYPHS` /
  `ASCII_GLYPHS` table (`warn: '⚠'`, `dash: '—'` vs `'-'`) and the
  `supportsUnicode()` gate. Context for D-11's glyph; the ASCII-fallback
  behavior itself is Phase-6 rendering territory.

### Current implementation (the extension points)
- `internal/query/status.go` — `StatusResult` + its **authoritative per-key
  TS→Go/Pebble decision table** (the doc comment at lines ~17–41: read this
  first, it explains every existing remap/drop/placeholder). Holds the inert
  `PendingChanges`, the `WorktreeMismatch *string` placeholder, live `Stale`,
  `IndexHealth.ReindexRecommended`, `NodesByKind`, `Languages []string`, and
  `computeStale` / the `.sync-pending` sidecar.
- `internal/query/engine.go` — `Engine{reader, repoRoot}`, `New`, `NewWithRoot`,
  and `OpenAt(start)` — **the D-14 plumbing change** (`start` is currently
  discarded after `ResolveCodegraphDir`).
- `internal/query/resolve.go` § `ResolveCodegraphDir` (lines 31–48) — the
  upward-walk that *causes* the borrowed-index bug; its returned dir is the
  `indexRoot` side of detection.
- `internal/query/render_markdown.go` § `staleBannerText` / `staleBanner`
  (lines ~18–31, used at ~362) — **the precedent D-12 mirrors** for the notice
  prefix; already uses the same U+26A0 `⚠`.
- `internal/cli/status.go` — the terse `backend=%s files=%d …` line D-09
  replaces, plus the `--json` / `MarshalStatusJSON` path.
- `internal/mcp/tools.go` — the 8 read tools (`codegraph_explore`, `_node`,
  `_search`, `_callers`, `_callees`, `_impact`, `_files`, `_status`); 7 take the
  compact notice, `_status` takes the verbose form (D-12).
- `internal/query/files_status_test.go` (lines ~360–372) — the existing tests
  **asserting the all-zero `PendingChanges` / nil `WorktreeMismatch`
  placeholders**. WORK-01 makes `WorktreeMismatch` live ⇒ these assertions must
  be updated, not deleted; `PendingChanges` stays zero (D-06).

### Golden / fixture harness (extend, don't rebuild)
- `testdata/golden/README.md` § "Volatile fields (Pitfall 1)" (lines ~102–125) —
  the table that strips `dbSizeBytes` / `lastIndexed` / `*_at` and the
  empirically-verified reproducibility claim. **D-08 amends this table.**
- `testdata/golden/golden_test.go` § `TestGoldenFixturesExist` — encodes the
  no-volatile-field invariant that D-08 may need to narrowly exempt.
- `testdata/golden/golden_parity_test.go` (lines ~610–660) — the status parity
  assertions, incl. the `worktreeMismatch`/`pendingChanges` "Phase-4 sync
  placeholder" checks and the expected-key list (which must gain `dbSizeBytes`).
- `testdata/golden/corpus/weft-go/status.json`,
  `testdata/golden/corpus/colbymchenry-codegraph/status.json` — the frozen TS
  1.3.1 status oracles (note: `dbSizeBytes`/`lastIndexed` already stripped;
  `languages` is a bare list; `projectPath`/`indexPath` → `<CORPUS_PATH>`).

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `staleBanner(stale)` / `staleBannerText` (`internal/query/render_markdown.go`):
  the exact prepend-a-warning-to-rendered-output pattern WORK-02's notice needs
  — and it already uses the correct U+26A0 `⚠` glyph.
- `computeStale` + the `.codegraph/.sync-pending` sidecar (`status.go`, v0.1
  D-04a): STAT-03's live signal **already exists** — surface it, don't rebuild.
- `StatusResult.NodesByKind map[string]int64`: real counts, already computed via
  the `IterateNodes` scan — STAT-02's nodes half is genuinely render-only.
- `ResolveCodegraphDir(start)` (`resolve.go`): returns the index root — the
  `indexRoot` argument to detection.
- `OpenAt` as "the single read seam CLI commands and MCP tool handlers both"
  use: the one place to plumb `startPath` so both surfaces gain awareness at once.
- `MarshalStatusJSON` + `StatusResult`'s decision-table doc comment: the
  established mechanism and documentation convention for every TS→Go key remap.

### Established Patterns
- **Shared `Engine`, one commit** — CLI + MCP both call `Engine.Status()`; parity
  is structural, mirroring how Phase 1 shipped explore/node.
- **Documented per-key remap table** — `StatusResult`'s doc comment is the
  project's convention for recording every TS→Go divergence *at the code*, not
  just in planning docs. New keys (`dbSizeBytes`, `filesByLanguage`) and the
  now-live `worktreeMismatch` must be added to that table.
- **Best-effort degradation over hard failure** — WR-04's dangling-edge
  tolerance; worktree detection follows the same philosophy (WORK-03).
- **Plain-string renderers** — `RenderExplore`/`RenderNode` return ANSI-free
  text; the new status sections must too (Phase 6 archtest will enforce it).
- **Golden-corpus diff harness** — `testdata/golden/` is the parity oracle to
  extend, with volatile fields stripped/normalized rather than asserted.

### Integration Points
- `OpenAt` → retain `startPath` on `Engine` (D-14) → `Engine.Status()` computes
  the live `WorktreeMismatch`; the 7 non-status read renderers gain the notice
  prefix.
- `internal/gitmeta` (new) ← consumed by `internal/query`; later reused by
  Phase 5's `internal/githooks` (`isGitRepo`-style probes).
- `internal/cli/status.go` → replaces the terse line with the sectioned layout;
  `--json` gains `dbSizeBytes` + keeps `languages` shape.
- `internal/mcp/tools.go` → `codegraph_status` renders the blockquote verbose
  warning; the other 7 tools get the compact prefix.
- `testdata/golden/golden_parity_test.go` + `files_status_test.go` → placeholder
  assertions flip from "must be inert" to "must be live".

</code_context>

<specifics>
## Specific Ideas

- **Verbatim strings, hexdump-verified.** The notice glyph is U+26A0 `⚠`
  (`e2 9a a0`) — **no** U+FE0F variation selector, unlike Phase 1's `⚠️`. Copy
  both the warning and the notice character-for-character from `worktree.js`;
  do not paraphrase or "improve" the wording, including the `"codegraph init -i"`
  advice (quoted, with the `-i` flag).
- **The `.claude/worktrees/<name>/` layout is the motivating true positive** —
  TS's own module doc calls out that tools place worktrees "under `.gitignore`d
  paths like `.claude/worktrees/<name>/`", and memory `76t84ynav5` flags this as
  the case that "bites Sean's GSD worktree-per-phase" flow. It must be a
  first-class fixture, not an afterthought.
- **Watch the gate-4 polarity.** Reviewers/planners reading only the requirement
  text will expect "common dirs differ ⇒ mismatch". The truth is the inverse:
  differing common dirs ⇒ **suppress** (it's a submodule/embedded clone, a
  different repo the parent index already covers). Encode this in a test with an
  explanatory comment so it survives future "simplification".
- **Memory correction worth carrying forward:** the TS reference is alive at the
  platform-sub-package path (D-01). Phase 1's post-mortem recorded it as gone.

</specifics>

<deferred>
## Deferred Ideas

- **Colorized / TTY-gated `status`** (lipgloss sections, `chalk`-equivalent
  green/cyan/yellow, the ASCII-vs-Unicode glyph fallback from `ui/glyphs.js`) —
  **Phase 6** (TUI-02, "Rendering Seam & Pretty status/files"). Phase 2 lays down
  the layout that Phase 6 paints.
- **Exact `pendingChanges` added/modified/removed counts** — explicitly **out of
  scope for v1.0** per the REQUIREMENTS.md Out-of-Scope table (needs Sync's diff
  re-run per `status` call). Revisit only if the live `stale` signal proves
  insufficient in practice.
- **`Journal:` / journalMode line** — no Pebble analog; permanently dropped
  (consistent with the existing `StatusResult` drop).
- **Auto-init / index-sharing for worktrees** ("make worktree support better") —
  explicitly deferred past v1.0 per PROJECT.md's Key Decisions row; v1.0 is
  detect + warn + notice only.
- **Reusing `internal/gitmeta` for git sync hooks** — Phase 5 (HOOK-01/02/03);
  design the package so it's reusable, but do not build hook logic here.
- **`getPendingFiles()` per-file freshness section + `isWatcherDegraded()`
  "Auto-sync disabled" section** in TS's MCP status — both depend on a live
  watcher, which is **Phase 3** (WATCH-01). Not Phase 2 content.

### Reviewed Todos (not folded)
- **"Document release procedures (maintainer runbook)"**
  (`2026-07-14-document-release-cut-procedures-runbook.md`, matcher score 0.40) —
  **deferred to Phase 8 (REL).** The 0.40 score came from generic keyword
  overlap ("phase", "internal"), not topical relevance; it is a
  release-engineering docs task with no bearing on status content or worktree
  detection. The `--auto` ≥0.4 auto-fold default is overridden by the scope
  guardrail, consistent with the identical call made in Phase 1's CONTEXT and
  with `p82eny7gf5`'s "release-runbook-resolves-phase-8".

</deferred>

---

*Phase: 2-status-content-git-worktree-awareness*
*Context gathered: 2026-07-15*
</content>
</invoke>
