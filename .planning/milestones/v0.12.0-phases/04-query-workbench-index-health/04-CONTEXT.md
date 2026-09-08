# Phase 4: Query Workbench & Index Health - Context

**Gathered:** 2026-08-29
**Status:** Ready for planning

<domain>
## Phase Boundary

A developer can run the four graph analyses (`Impact`, `Affected`, `Callers`,
`Callees`) interactively with their own knobs, read the results as real sortable
tables, and tell at a glance whether the index they are reading is worth trusting.

**Requirements:** WRK-01, WRK-02, WRK-03, WRK-04, HLT-01, HLT-02, HLT-03

**Two facts that bound this phase's scope, both verified in tree during discussion:**

1. **All four analyses already exist as RPCs.** `Callers`/`Callees`
   (`symbol`, `limit`), `Impact` (`symbol`, `depth`), `Affected`
   (`repeated files`, `depth`) — `internal/uiproto/uiv1/ui.proto:51-59`. No new
   analysis capability is being built; this phase builds the interactive surface
   over what Phase 1 already shipped.

2. **The engine already computes every health value HLT-01/02/03 needs.**
   `internal/query.StatusResult` (`status.go:47-65`) carries `FilesByLanguage`,
   `Languages`, `WorktreeMismatch`, `IndexHealth`, `PendingChanges`,
   `DbSizeBytes`, `NodesByKind`, `EdgesByKind`. `GetStatusResponse` exposes only
   nine scalars and none of those. **The gap is the wire, not the engine.**

Both routes already exist as Phase 2 placeholder slots
(`web/src/routes/workbench/+page.svelte`, `web/src/routes/health/+page.svelte`).
Per **D-18**, Phase 4 *fills* them and does not restructure navigation.

</domain>

<decisions>
## Implementation Decisions

### Health Data on the Wire (HLT-01, HLT-02, HLT-03)

- **D-01:** **A new health RPC carries the rich health data.
  `GetStatusResponse` is NOT extended.**

  > **⚠ CORRECTED after research (2026-08-29) — the rpc is named `GetHealth`,
  > NOT `GetIndexHealth`. Read this before writing any proto or test.**
  >
  > Discussion named this rpc `GetIndexHealth` throughout. That name **fails an
  > existing test** and must not be used. `internal/uiserver/readonly_test.go:89-93`
  > defines `mutatingVerbs`, a literal fixture of write-verb substrings that
  > includes **`"Index"`**, and applies it with a bare `strings.Contains` over
  > every `UIServiceHandler` method name. `"GetIndexHealth"` contains `"Index"`,
  > so `TestUIServiceDeclaresNoMutatingMethod` would fail.
  >
  > **The guard is correct and must NOT be weakened.** It is deliberately built
  > as a complementary pair — a negative verb-substring check alongside
  > `TestUIServiceMethodSetIsExactlyTheReadSet`'s positive set equality — and it
  > asserts a positive count of inspected method names *before* applying the verb
  > check, citing rule `84d1gfpywd` in its own comment. Adding an allowlist
  > exception would punch a hole in a guard designed not to go vacuous. **Rename
  > the rpc; do not touch `mutatingVerbs`.**
  >
  > **Chosen name: `GetHealth`** — matches the existing `/health` route, keeps the
  > domain concept ("index health") intact in requirement and UI language, and is
  > the minimal change that clears the collision. Verified clean against the full
  > `mutatingVerbs` list, positive-controlled by confirming all ten existing
  > method names also pass (so the check discriminates rather than accepting
  > everything). `GetGraphHealth` and `GetDiagnostics` are also clean if a later
  > reviewer prefers one; `GetIndexStats` is NOT (same `"Index"` collision).
  >
  > **Two literals must be updated in the same change** (`readonly_test.go`), and
  > research confirmed a third does NOT need updating:
  > - `wantUIServiceMethods` (lines 31-42) — add `"GetHealth": {}`.
  > - the `10` literal at line 61 (`if len(got) != 10`) — becomes `11`.
  > - **`proto:drift`'s `nfiles -lt 4` floor (`Taskfile.yml:339-348`) needs NO
  >   change** — it counts generated *files*, and adding an rpc to the existing
  >   `ui.proto` does not change the file count. This corrects an assumption in
  >   the research brief itself.

  Phase 3's `createStatusGate` (`web/src/lib/status.ts`) fetches `GetStatus` on
  **every navigation** — that is its documented contract ("fetch on load and
  navigation only", asserted by a no-timer-API criterion). Adding per-language
  maps, node/edge-by-kind maps and worktree detection to that message would put
  the health page's entire cost on every click in the application.

  The split follows the standard health-endpoint discipline: a cheap,
  frequently-polled signal and a separate diagnostic endpoint. `GetStatus` keeps
  its nine scalars unchanged; `GetHealth` is the 11th rpc on `UIService`,
  additive and read-only per **D-02a**.
  — **Reversibility:** cheap — collapsing the two messages later is a mechanical
  merge; splitting them after the fact would require re-auditing every caller.

- **D-02:** **The worktree-mismatch check runs ONLY inside `GetHealth`.**

  `gitmeta.DetectIndexMismatch` spawns **up to four git subprocesses** — its own
  doc comment says so (`internal/gitmeta/detect.go:22`). A worktree mismatch is a
  configuration-level fact that cannot change mid-session without the user moving
  the repository, so per-navigation freshness buys nothing and costs four
  process spawns per click. Compute it when the health view is opened, nowhere else.

- **D-03:** **HLT-01's word "coverage" means per-language file counts.**

  Read "coverage, per-language file counts, and node and edge counts" as one
  enumeration, satisfied by `StatusResult.FilesByLanguage` and
  `StatusResult.Languages`.

  **Explicitly NOT chosen, so the planner does not silently pick either:**
  - `internal/indexer/capability/matrix.go` defines a per-language, per-axis
    `Coverage` type (`full|partial|none`). It is a real and arguably richer trust
    signal, and it is what "coverage" already means elsewhere in this codebase —
    but surfacing it is a second data source and a larger UI than the requirement
    asks for. Out of scope for Phase 4.
  - An indexed-vs-on-disk ratio does not exist anywhere today and would require a
    filesystem walk — new engine work inside a UI phase. Rejected.

- **D-04:** **HLT-02's trust verdict reuses Phase 3's `classifyStatus` unchanged.**

  `web/src/lib/status.ts` already produces a five-member `StatusVerdict`
  (ok/stale/no-index/indexing/unknown) plus an orthogonal `CommitKnowledge`, and
  `StatusBanner.svelte` renders it application-wide. The health page renders that
  same verdict prominently, above the raw numbers.

  **No second verdict function is defined.** A health-specific verdict weighing
  `IndexHealth.ReindexRecommended`, `PendingChanges` and worktree mismatch was
  considered and rejected: two verdict functions over one index can disagree, and
  the banner and the health page contradicting each other about staleness is
  exactly the repudiation failure HLT-02 exists to prevent.

### Result Tables (WRK-04)

- **D-05:** **TanStack Table is the table engine — `@tanstack/svelte-table@9.2.4`
  plus shadcn-svelte's `table` component.**

  **Maintainer ruling, verbatim:** *"do the right and idiomatic thing. add deps
  where they are appropriate, prefer them over novel or half baked solutions."*
  Dependency count is a tiebreaker, never the decision-maker.

  This is the sanctioned path: shadcn-svelte deliberately ships **no** sorting
  abstraction — its `data-table` documentation is a *guide* that says the
  implementation "requires the addition of the Table component and the
  installation of the `@tanstack/svelte-table` dependency". Hand-rolling sorting
  would diverge from the component system 03-06 already committed to.

  **Version verified during discussion:** `npm view @tanstack/svelte-table
  dist-tags` → `latest: 9.2.4` (v9 is out of beta; `beta` tag still points at
  `9.0.0-beta.80`), `time.modified` 2026-08-28. This is a stable release, not a
  pre-release gamble — the planner does not need to re-litigate that.

- **D-06:** **One shared `DataTable` shell + one `columns.ts` per analysis.**

  All four analyses return the byte-identical `Location` message — `name`,
  `kind`, `file_path`, `start_line` (`ui.proto:100-105`). The maintainer chose
  "four per-analysis tables"; under TanStack that means four declarative
  `ColumnDef[]` arrays and four thin wrappers over a single shared shell, which
  is the structure shadcn's own data-table guide prescribes. **Sorting lives in
  the engine and is written once** — "four tables" costs four column definitions,
  not four sort implementations.

  `Impact`'s scalar `node_count`/`edge_count` and `Affected`'s echoed `files`
  render as a header summary above their table, not as columns.

- **D-07:** **Sorting is client-side over the returned rows, via TanStack's
  `createSortedRowModel()`.**

  No sort parameter is added to any RPC. Depth and limit continue to pass
  straight through to the server's existing bounds — `MaxLimit = 1000` **refuses**
  (`validateLimit`), `MaxDepth = 50` **clamps** (`clampDepth`) — with no
  client-side duplicate, per the ROADMAP's own Notes for this phase.

- **D-08:** **Render cost is measured, and virtualized with
  `@tanstack/svelte-virtual` if it crosses a threshold.**

  A workbench table can return up to `MaxLimit` = 1000 rows. This is the folded
  render-cost todo's home. Measurement comes first; if the threshold trips, the
  fix is the TanStack virtualization companion, **not** hand-rolled windowing and
  **not** a second client-side row cap over results the server already bounded.

  This decision is the load-bearing reason D-05 chose TanStack over a hand-rolled
  sort: hand-rolling *sorting* is ~30 lines, hand-rolling *virtualization* is not,
  and discovering the threshold later would mean rewriting onto TanStack anyway.

- **D-09:** **CORRECTION — D-22 is a recorded gap, not a gate, and it does not
  mandate a checkpoint for Phase 4.**

  This correction was made mid-discussion after a maintainer challenge, and is
  recorded so the planner does not inherit the wrong constraint.

  `03-CONTEXT.md:357` states the gap: `shadcn-svelte add` writes `.svelte`
  **source files** into the repo that never enter `pnpm-lock.yaml`, so
  "**BLD-06's audit gate structurally cannot see them**". Its stated mitigation is
  "**scope discipline, not new machinery**". The `gate="blocking-human"`
  checkpoint was **03-06's own plan-level choice** for implementing that
  principle — not a standing rule.

  **The risk asymmetry, stated plainly because it is easy to get backwards:**

  | Artifact | In `pnpm-lock.yaml`? | Seen by `web:audit` / `web:deps:strict` / `web:lockfile`? |
  |---|---|---|
  | `@tanstack/svelte-table`, `@tanstack/svelte-virtual` | **yes** | **yes — fully covered** |
  | vendored `table` `.svelte` source | **no** | **no — invisible to the gate** |

  Adding the npm dependencies is therefore the **low-ceremony, well-covered**
  path. The vendored component source is the unscanned surface. `03-RESEARCH.md:148`
  confirms the same asymmetry for the CLI itself (`pnpm dlx`, never in the lockfile).

  How Phase 4 reviews the ~6 vendored `table` files is the planner's call — but
  see **D-16**, which closes the gap structurally and supersedes per-phase human reads.

### Workbench Routing & URL Model (NAV-01, WRK-01..03)

- **D-10:** **The Workbench gets a sibling `web/src/lib/workbench-url.ts`,
  importing browse-url's already-exported primitives.**

  **D-13 scopes `browse-url.ts` as the one URL grammar *for the Browse view*** —
  a sibling grammar for a different view is not a violation of it. `browse-url.ts`
  already exports `isShapeInteger` / `parseShapeInteger` / the `INTEGER_SHAPE`
  discipline precisely so that every writer validates against the same reader
  ("rather than maintaining a second, looser check that can silently disagree
  with the parser"). `workbench-url.ts` imports those rather than redefining them.

  Browse's frozen, round-trip-tested grammar is not modified.

- **D-11:** **Multiple files are encoded as repeated `file=` keys, read with
  `getAll()`.**

  `?file=a.go&file=b.go` — what `URLSearchParams` natively models and what
  `AffectedRequest.files` (already `repeated`) expects. No delimiter is invented.

  **This is a real difference from `browse-url.ts`, and the planner must not
  copy Browse's reader here:** browse-url reads known keys with
  `params.get('file')`, which silently drops all but the first occurrence. The
  workbench grammar reads its multi-valued key with `getAll()`. Comma-joining was
  rejected — it breaks on any path containing a comma and needs a bespoke escape
  rule where the platform already has one.

- **D-12:** **The Workbench's full input state lives in the URL** — analysis mode,
  inputs, depth, and limit.

  A shared Workbench URL reconstructs the exact query its sender ran. Encoding
  only the analysis and primary input would let a shared link silently reproduce a
  *different* result whenever the sender had moved the depth control — the same
  T-03-28 link-fidelity failure Phase 3 mitigated with its one-serializer /
  one-parser round-trip assertion.

- **D-13:** **The four analyses are tabs, one per analysis; the selected tab is
  the `mode` URL parameter.**

  Each tab carries its own input controls and result table, mapping directly onto
  D-06's four `columns.ts`. shadcn-svelte's `tabs` is a Bits UI primitive from the
  same family already vendored in 03-06.

### Files, Selection & Supply Chain

- **D-14:** **The `Files` glob bug is fixed by adopting
  `github.com/bmatcuk/doublestar/v4` in `internal/query/files.go`.**

  **This is not a UI-local bug.** `Engine.Files` has three callers — CLI
  (`internal/cli/files.go:46`), MCP (`internal/mcp/tools.go:512`), and UI
  (`internal/uiserver/handlers.go:433`). `FilesOptions.Pattern` uses
  `filepath.Match`, whose `*` never crosses a `/`, so live file search returns
  **zero** matches for any nested path — confirmed live in Phase 3's UAT, not
  assumed.

  **Verified during discussion:** `doublestar.Match` is documented as "a
  **drop-in replacement** for `path.Match()`" (which is what this code wants —
  forward-slashed paths), adding `**` and `{alts}`. Pure Go, zero dependencies.
  Because **no existing caller uses `**`**, every pattern in use today behaves
  identically: no CLI golden churn, and no MCP frozen-transcript re-baselining —
  which **T-03-14 forbids**. The fix lands for all three callers at once.

  Rejected: a new orthogonal `Substring` option (two overlapping narrowing
  mechanisms on one API, inventing a matcher the ecosystem already standardises);
  and a client-side-only pattern hack (leaves the bug live for CLI and MCP).

  > **Planner must verify, do not assume:** that `**/*term*` matches a
  > root-level file (i.e. that `**` matches zero path segments in doublestar v4).
  > The UI's live search builds patterns from a bare term, and root-level files
  > currently *work* — the fix must not regress them. Prove it with a test
  > carrying both a nested and a root-level case.

- **D-15:** **Multi-file selection is search-and-add with removable chips.**

  A search input over the `Files` RPC (working at depth once D-14 lands); each
  pick becomes a removable chip; the chip set serializes to D-11's repeated
  `file=` params. Reuses 03-06's vendored `Command` primitive and `search.ts`'s
  debounce/cancellation controller pattern — no new interaction model.

  A file tree with checkboxes was rejected for this phase: `FileTreeNode` is
  already on the wire, but tree UI plus checkbox-cascade semantics is real scope,
  and **Phase 5 is the phase chartered for structural views**.

- **D-16:** **The shadcn-svelte registry-pinning todo is folded in and closed
  with a `task web:components:drift` target — local and release path, NOT PR CI.**

  Naming matches the existing `web:build`/`web:drift` and `proto:gen`/`proto:drift`
  pairing convention. The target re-runs the pinned-version CLI for each added
  component and byte-diffs against what is committed under
  `web/src/lib/components/ui/`, failing on any difference.

  **Deliberately not in PR CI**: the todo itself flags that re-running `add`
  against a live network registry is a materially different risk from the
  self-contained `web:drift`/`proto:drift` guards, which diff a locally
  regenerated artifact. Wiring it into required checks would make an external
  service a merge dependency — and per rule `f18zrdsgx5` a wedged required check
  on `protect-main` is painful to recover from.

  This closes the gap D-22 recorded, replacing per-phase human reads with real
  drift coverage. It was declined as scope growth by Phase 2 and again by Phase 3;
  **the maintainer folded it in here deliberately** ("prefer them over novel or
  half baked solutions").

### Folded Todos

Three pending todos were folded into this phase during discussion:

- `.planning/todos/pending/2026-08-29-files-rpc-pattern-glob-cannot-cross-directory-boundaries-for-live-file-search.md`
  → closed by **D-14**.
- `.planning/todos/pending/2026-08-28-client-side-render-cost-measurement-for-browse-views.md`
  → addressed by **D-08** (measurement, then virtualize if over threshold).
- `.planning/todos/pending/2026-08-28-shadcn-svelte-registry-version-pinning-with-source-match.md`
  → closed by **D-16**.

Each must be moved to `.planning/todos/completed/` with a resolution record as
part of this phase, following 03-03's precedent (`git mv`, verifiable resolution).

### Claude's Discretion

- **WRK-04 criterion 3's error taxonomy** — how `classifyRpcError` (D-04, Phase 3)
  and the status gate compose into a distinguishable "not found / index stale /
  server error". The pieces exist; the composition is the planner's.
- **The health page's concrete layout** — what "above the raw numbers" means
  visually for HLT-02, and how the HLT-03 worktree warning is made "impossible to miss".
- **Whether `GetHealth` defines its own message or reuses `GetStatusResponse`
  fragments.**
- **The render-cost threshold value** in D-08, and how it is measured.
- Empty, loading and error states for each of the four analyses.
- Exact chip-affordance design in D-15.

</decisions>

<specifics>
## Specific Ideas

- **Maintainer's stated engineering philosophy for this phase, applied to D-05,
  D-14 and D-16:** *"do the right and idiomatic thing. add deps where they are
  appropriate, prefer them over novel or half baked solutions."* When a decision
  in this phase is between an established ecosystem library and a bespoke local
  mechanism, the library wins unless there is a correctness reason it cannot.

- **A maintainer challenge corrected a live error mid-discussion** (recorded in
  D-09): the orchestrator had asserted D-22 created "two execution-time gates"
  for adding dependencies. That was wrong in both directions — D-22 is not a gate,
  and npm dependencies are the *covered* path while vendored source is the
  unscanned one. Downstream agents should treat D-09's table as the authority on
  which artifacts the audit gate can actually see.

</specifics>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope and requirements
- `.planning/ROADMAP.md` §"Phase 4: Query Workbench & Index Health" — goal, the
  five success criteria, and the Notes paragraph on depth/limit passing straight
  through to existing server-side bounds.
- `.planning/REQUIREMENTS.md` — WRK-01..04 (lines 57-60), HLT-01..03 (lines 72-74).

### Prior-phase decisions this phase builds on
- `.planning/phases/03-browse-inspect-navigation/03-CONTEXT.md` — D-13 (one URL
  grammar for Browse), D-04 (one Connect-error classification), D-18 (route
  placeholders are filled, not restructured), D-22 (the vendored-component
  supply-chain gap; see D-09 above for the correction to how it binds).
- `.planning/phases/02-spa-toolchain-embedded-app-shell-js-supply-chain/02-CONTEXT.md`
  — D-01/D-02/D-03 (SvelteKit + adapter-static, `web/`, committed `web/build/`).
- `.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-CONTEXT.md`
  — D-02a additive-only proto discipline; D-04's corrected exit-status-AND-count
  verification rule, which every `<verify>` gate in this phase must follow.

### Wire and engine surfaces
- `internal/uiproto/uiv1/ui.proto` — the 10 existing rpcs (lines 51-59, 67),
  `Location` (100-105), `GetStatusResponse` (124-…), `CallersRequest` (258),
  `ImpactRequest` (285), `AffectedRequest` (307).
- `internal/query/status.go:47-65` — `StatusResult`, the source of every health
  value D-01/D-03 exposes; `IndexHealth` (79-86), `PendingChanges` (70-74).
- `internal/gitmeta/detect.go` — `Mismatch` (10-13) and `DetectIndexMismatch`,
  whose four-subprocess cost drives D-02.
- `internal/query/validate.go:22,26` — `MaxDepth = 50`, `MaxLimit = 1000`, the
  server-side bounds D-07 refuses to duplicate client-side.
- `internal/query/files.go:14-30,136-177` — `FilesOptions.Pattern` and the
  `filepath.Match` call sites D-14 replaces.

### Frontend surfaces being extended
- `web/src/lib/browse-url.ts` — `BROWSE_PARAM_KEYS`, `isShapeInteger`,
  `parseShapeInteger`, and the deliberate ordered-multimap treatment of unknown
  params. D-10 imports from here; D-11 deliberately diverges on `getAll()`.
- `web/src/lib/status.ts` — `classifyStatus`, `StatusVerdict`, `createStatusGate`;
  reused unchanged by D-04, and the reason D-01 keeps `GetStatus` cheap.
- `web/src/lib/rpc-errors.ts` — `classifyRpcError`, Phase 3's D-04 deliverable,
  explicitly designated for reuse by Phases 4/5/6.
- `web/src/lib/search.ts` — the debounce + AbortController + request-identity
  controller pattern D-15 reuses.
- `web/src/routes/workbench/+page.svelte`, `web/src/routes/health/+page.svelte` —
  the placeholder slots this phase fills.

### External library documentation
- shadcn-svelte `data-table` guide — the source of D-05/D-06's structure
  (shared `DataTable` shell + per-table `columns.ts`); states the TanStack
  dependency requirement explicitly.
- `github.com/bmatcuk/doublestar/v4` (pkg.go.dev) — `Match` as a drop-in
  replacement for `path.Match`; `UPGRADING.md` for the v3→v4 `**` corner cases.

### Standing repository rules
- Rule `84d1gfpywd` — every guard carries a positive assertion that it did its
  work; negative-only guards pass vacuously.
- Rule `f18zrdsgx5` — never `[ci skip]`; `protect-main` requires 6 status checks.
  Cited by D-16's decision to keep the drift target out of required CI.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **`StatusResult`'s health fields** (`internal/query/status.go:47-65`) — every
  value HLT-01/02/03 needs is already computed. `GetHealth` is a mapping
  layer, not new engine work.
- **All four analysis RPCs** — `Callers`, `Callees`, `Impact`, `Affected` already
  exist and are already bounded server-side. Phase 4 adds no analysis capability.
- **`classifyStatus` / `StatusBanner`** — the app-wide verdict, reused verbatim by D-04.
- **`classifyRpcError`** (`web/src/lib/rpc-errors.ts`) — Phase 3 built this
  explicitly for Phases 4/5/6.
- **`search.ts`'s controller pattern** — debounce, AbortController, monotonic
  request-identity guard; D-15's file picker reuses the shape rather than
  reinventing cancellation.
- **Vendored `Command`, `dialog`, `button`, `input`, `input-group`, `textarea`**
  (`web/src/lib/components/ui/`) — the picker and tabs build on this family.

### Established Patterns
- **"ONE X" discipline** — one URL grammar per view, one error classifier, one
  highlighter, one gather path. D-06's shared `DataTable` shell and D-10's
  primitive-importing sibling grammar both honour it.
- **Additive-only proto changes** (D-02a) — D-01's new rpc adds; it does not
  reshape `GetStatusResponse`.
- **Drift guards for anything generated or fetched** — `web:drift` for
  `web/build/`, `proto:drift` for the generated TS client. D-16 completes the set
  with `web:components:drift`.
- **Bounds live server-side, never duplicated client-side** — the ROADMAP's own
  Notes for this phase; enforced by D-07.

### Integration Points
- `internal/uiserver/handlers.go` — where `GetHealth` mounts, alongside the
  existing `Files` handler at line 433 that D-14's fix flows through.
- `internal/uiproto/uiv1/ui.proto` + `task proto:gen` — the new rpc regenerates
  both the Go and the committed TypeScript client; `proto:drift` must stay green.
- `web/src/routes/workbench/+page.svelte` and `health/+page.svelte` — the two
  placeholder slots.
- `Taskfile.yml` — `web:components:drift` joins the existing `web:*` target family.
- `go.mod` — `doublestar/v4` is the phase's one new Go dependency; note the
  `vuln` target's 8-binary scan and `TestToolModfilesPopulationMatchesDisk`
  operate on **tool** modfiles, so a main-module dependency does not touch them.

</code_context>

<deferred>
## Deferred Ideas

- **Capability-matrix coverage in the health view** — `internal/indexer/capability/matrix.go`'s
  per-language `full|partial|none` axes are arguably the richer trust signal, but
  surfacing them is a second data source and a larger UI than HLT-01 asks for.
  Revisit if the per-language file counts prove insufficient in use.
- **Indexed-vs-on-disk coverage ratio** — the most literal reading of "coverage",
  but nothing computes it today and it needs a filesystem walk. A future
  engine-side capability, not a UI phase's work.
- **File tree with checkboxes for multi-file selection** — rejected for D-15 in
  favour of search-and-add. Phase 5 is chartered for structural views and is the
  natural home if a tree is still wanted.
- **Server-side sorting** — adding a sort parameter to the four analysis RPCs
  would order the full result set before the limit applies, which client-side
  sorting cannot. Deferred as a wire change this phase's ROADMAP notes push away from.
- **Wiring `web:components:drift` into required PR CI** — deliberately out of
  scope per D-16; revisit only alongside a decision about network access in
  required checks.

</deferred>

---

*Phase: 04-query-workbench-index-health*
*Context gathered: 2026-08-29*
