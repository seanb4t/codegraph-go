# Phase 4: Query Workbench & Index Health - Research

**Researched:** 2026-08-29
**Domain:** SvelteKit/Svelte 5 frontend over an existing Go/ConnectRPC read-only API (table UI, health dashboard, one new RPC, one Go dependency swap)
**Confidence:** HIGH — every claim below is either read from this repository this session (file:line cited) or confirmed by running a real command/program and quoting its verbatim output. Two claims that appeared settled in `04-CONTEXT.md`/the research brief were found to be **wrong** during verification; both are corrected explicitly below (see Common Pitfalls #1 and #2).

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

D-01 through D-16, copied verbatim from `04-CONTEXT.md`'s `<decisions>` section — the planner MUST honor these; this research does not re-litigate any of them (only D-01's literal RPC *name* is challenged, in Common Pitfalls #1, on verified technical grounds — the RPC's *behavior* per D-01 is unchanged):

- **D-01:** A new `GetIndexHealth` RPC carries the rich health data. `GetStatusResponse` is NOT extended. `GetStatus` keeps its nine scalars unchanged; the new rpc is the 11th rpc on `UIService`, additive and read-only per D-02a. — Reversibility: cheap.
- **D-02:** The worktree-mismatch check runs ONLY inside the new health RPC — never per-navigation on `GetStatus`.
- **D-03:** HLT-01's word "coverage" means per-language file counts (`StatusResult.FilesByLanguage`/`Languages`). Explicitly NOT the capability-matrix `Coverage` type, and NOT an indexed-vs-on-disk ratio — both out of scope.
- **D-04:** HLT-02's trust verdict reuses Phase 3's `classifyStatus` unchanged. No second verdict function is defined.
- **D-05:** TanStack Table is the table engine — `@tanstack/svelte-table@9.2.4` plus shadcn-svelte's `table` component. Maintainer ruling: "do the right and idiomatic thing. add deps where they are appropriate, prefer them over novel or half baked solutions."
- **D-06:** One shared `DataTable` shell + one `columns.ts` per analysis. Sorting lives in the engine and is written once.
- **D-07:** Sorting is client-side over the returned rows, via TanStack's `createSortedRowModel()`. No sort parameter is added to any RPC. Depth/limit pass straight through to the server's existing bounds with no client-side duplicate.
- **D-08:** Render cost is measured, and virtualized with `@tanstack/svelte-virtual` if it crosses a threshold. Measurement comes first.
- **D-09:** CORRECTION — D-22 (03-CONTEXT.md) is a recorded gap, not a gate, and does not mandate a checkpoint for Phase 4. Adding npm dependencies (`@tanstack/svelte-table`, `@tanstack/svelte-virtual`) is the well-covered path (in `pnpm-lock.yaml`); vendored `.svelte` component source is the unscanned surface (see D-16).
- **D-10:** The Workbench gets a sibling `web/src/lib/workbench-url.ts`, importing browse-url's already-exported primitives (`isShapeInteger`/`parseShapeInteger`). Browse's frozen grammar is not modified.
- **D-11:** Multiple files are encoded as repeated `file=` keys, read with `getAll()` — NOT comma-joined, NOT `params.get('file')` (which drops all but the first occurrence).
- **D-12:** The Workbench's full input state lives in the URL — analysis mode, inputs, depth, and limit.
- **D-13:** The four analyses are tabs, one per analysis; the selected tab is the `mode` URL parameter. shadcn-svelte's `tabs` (a Bits UI primitive from the same family already vendored in 03-06).
- **D-14:** The `Files` glob bug is fixed by adopting `github.com/bmatcuk/doublestar/v4` in `internal/query/files.go`. Not UI-local — fixes all three callers (CLI, MCP, UI) at once. **Planner must verify, do not assume:** that `**/*term*` matches a root-level file (`**` matches zero path segments in doublestar v4) — VERIFIED this session, see Code Examples.
- **D-15:** Multi-file selection is search-and-add with removable chips, reusing 03-06's vendored `Command` primitive and `search.ts`'s debounce/cancellation controller pattern. A file tree with checkboxes was rejected for this phase (deferred to Phase 5).
- **D-16:** The shadcn-svelte registry-pinning todo is folded in and closed with a `task web:components:drift` target — local and release path, NOT PR CI. Re-runs the pinned-version CLI for each added component and byte-diffs against what is committed.

Three pending todos folded into this phase: the `files-rpc-pattern-glob` todo (closed by D-14), the `client-side-render-cost-measurement` todo (addressed by D-08), and the `shadcn-svelte-registry-version-pinning` todo (closed by D-16). Each must be moved to `.planning/todos/completed/` with a resolution record as part of this phase.

### Claude's Discretion

- WRK-04 criterion 3's error taxonomy — how `classifyRpcError` (Phase 3) and the status gate compose into a distinguishable "not found / index stale / server error". The pieces exist; the composition is the planner's. (This research documents both pieces' exact current shapes in the Architectural Responsibility Map and cites `rpc-errors.ts`/`status.ts` directly — see those sections.)
- The health page's concrete layout — what "above the raw numbers" means visually for HLT-02, and how the HLT-03 worktree warning is made "impossible to miss".
- Whether `GetIndexHealth` defines its own message or reuses `GetStatusResponse` fragments. (This research recommends a new dedicated message — see Pattern 1 and Pitfall #5.)
- The render-cost threshold value in D-08, and how it is measured.
- Empty, loading and error states for each of the four analyses. (This research's Pattern 3 shows shadcn-svelte's own `{:else}`-inside-`{#each}` idiom for the empty case.)
- Exact chip-affordance design in D-15.

### Deferred Ideas (OUT OF SCOPE)

- Capability-matrix coverage in the health view (`internal/indexer/capability/matrix.go`'s per-language `full|partial|none` axes) — a second data source, larger UI than HLT-01 asks for.
- Indexed-vs-on-disk coverage ratio — nothing computes it today; needs a filesystem walk; a future engine-side capability.
- File tree with checkboxes for multi-file selection — rejected for D-15; Phase 5 is chartered for structural views.
- Server-side sorting — would need to run before the limit applies, which client-side sorting cannot; deferred as a wire change this phase's ROADMAP notes push away from.
- Wiring `web:components:drift` into required PR CI — deliberately out of scope per D-16.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|--------------|----------------------|
| WRK-01 | User can run `Impact` on a symbol and adjust traversal depth interactively | `ImpactRequest`/`ImpactResponse` already exist (`ui.proto:283-296`); depth passes straight through to `clampDepth` (`validate.go:71-79`). Research: Pattern 2/3 (TanStack `createTable` + shadcn `Table.*`), `workbench-url.ts` sibling grammar (D-10) |
| WRK-02 | User can select multiple files and see what they affect | `AffectedRequest.files` already `repeated` (`ui.proto:305-311`); D-11's `getAll()` reading, D-15's search-and-add picker reusing `search.ts`'s controller pattern (Don't Hand-Roll table) |
| WRK-03 | User can run `Callers`/`Callees` with an adjustable result limit | `CallersRequest`/`CalleesRequest` already exist (`ui.proto:258-278`); limit passes straight through to `validateLimit` (`validate.go:107-115`). Handler pattern (Pattern 1) directly reusable |
| WRK-04 | Workbench results render as structured, sortable tables; a failed query names its failure kind | TanStack Table v9 real API (Pattern 2), shadcn-svelte `table` registry contents (Code Examples), `rpc-errors.ts`/`status.ts` exact shapes (Architectural Responsibility Map row) for the error-taxonomy composition |
| HLT-01 | Freshness, coverage, per-language file counts, node/edge counts all readable from one view | `StatusResult` already computes all of it (`status.go:47-65`); new RPC is a mapping layer only (Pattern 1) |
| HLT-02 | Staleness renders as a trust verdict above the raw numbers | `classifyStatus`/`StatusVerdict` reused unchanged (`status.ts:30,49-62`) |
| HLT-03 | Worktree mismatch renders as a loud, first-class warning | `gitmeta.Mismatch`/`DetectIndexMismatch` (`detect.go:10-38`); representation recommendation in Pitfall #5 |
</phase_requirements>

## Summary

This phase is almost entirely composition, not invention. All four analyses (`Impact`, `Affected`, `Callers`, `Callees`) already exist as bounded RPCs (`internal/uiproto/uiv1/ui.proto:51-58`, confirmed read this session), and every value the health view needs is already computed by `query.StatusResult` (`internal/query/status.go:47-65`). The phase's real work is: (1) one new additive RPC (`GetIndexHealth`) mapping existing engine data onto the wire, (2) a TanStack Table v9 + shadcn-svelte `table`/`tabs` UI layer over the four analyses, (3) a `doublestar/v4` swap fixing the root-cause glob bug in `internal/query/files.go`, and (4) a `web:components:drift` Taskfile target closing the vendored-component supply-chain gap.

Two verification results change what the planner should do versus what `04-CONTEXT.md` assumed. First, **the RPC name `GetIndexHealth` — used throughout `04-CONTEXT.md` — will fail an existing test** (`internal/uiserver/readonly_test.go:89-113`'s `TestUIServiceDeclaresNoMutatingMethod`) because that test does a bare `strings.Contains(name, "Index")` and `"Index"` is one of its 19 listed mutating-verb substrings; `"GetIndexHealth"` contains `"Index"` literally. This is a real, confirmed collision, not a hypothetical — the planner must either rename the RPC or add a documented, narrow exception to that test. Second, the CONTEXT brief's framing of `proto:drift`'s file-count floor as something "the planner must update" is itself incorrect: the floor counts **generated files** (currently 4: `graph.pb.go`, `ui.pb.go`, `ui.connect.go`, `ui_pb.ts`), not RPC methods, and adding one RPC to the existing `ui.proto` does not add a file — `nfiles` stays 4, unaffected. The test that genuinely DOES need a literal update is a different one: `internal/uiserver/readonly_test.go:31-42,61-66`'s `wantUIServiceMethods` map and its hardcoded `10`.

`doublestar.Match` was verified live (not assumed) to fix the root-level regression risk the brief specifically worried about: a bare-term pattern like `**/*claude*` matches a root-level file `claudeassets.go` (`**` matches zero path segments) exactly as it matches the nested case. TanStack Table v9's actual Svelte adapter is a full rune-based rewrite (`createTable`, `table.atoms.<slice>.get()`, `$effect.pre`-driven option sync) that supersedes the store-based (`writable<TableOptions>`, `createSvelteTable`) API still shown on the public docs site as of this session — the public docs page is stale relative to the package that just shipped (both confirmed from the real `main`-branch source, matching the installed `9.2.4` npm metadata exactly).

**Primary recommendation:** build `GetIndexHealth` as a straight field-projection RPC following the existing `statusToProto`/`locationToProto` mapper convention (`internal/uiserver/handlers.go:198-227`), rename it to avoid the `mutatingVerbs` collision (e.g. `GetHealth` or `GetGraphHealth`), use `@tanstack/svelte-table@9.2.4`'s real rune-based `createTable`/`FlexRender` API (not the store-based API shown in generic Context7 docs), and wire `web:components:drift` on `proto:drift`'s regenerate-and-byte-compare shape, not `web:drift`'s hash-comparison shape (components have no build marker to hash against).

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Four analysis RPCs (Impact/Affected/Callers/Callees) | API / Backend | — | Already exist, already bounded server-side (`internal/query/validate.go:22,26`) — Phase 4 adds no analysis capability, only a client |
| Depth/limit bounds enforcement | API / Backend | — | `validateDepth`/`clampDepth`/`clampAffectedDepth`/`validateLimit` already enforce this; D-07 explicitly forbids a client-side duplicate |
| Result table rendering + client-side sort | Browser / Client | — | D-05/D-07 — TanStack Table v9, sorted over the already-returned, already-bounded row set; no sort parameter added to any RPC |
| Row virtualization (conditional) | Browser / Client | — | D-08 — only if render-cost measurement crosses a threshold; `@tanstack/svelte-virtual` |
| Workbench URL state (mode/inputs/depth/limit) | Browser / Client | — | D-10/D-11/D-12 — `workbench-url.ts`, a sibling grammar to `browse-url.ts` |
| Multi-file search-and-add picker | Browser / Client | API / Backend (Files RPC) | D-15 — client debounce/cancellation reusing `search.ts`'s pattern, backed by the existing `Files` RPC once D-14's glob fix lands |
| Health data aggregation (freshness, per-language counts, node/edge counts) | API / Backend | — | Already computed by `query.StatusResult` (`internal/query/status.go:47-65`); the new RPC is a mapping layer, not new engine work |
| Worktree mismatch detection | API / Backend | — | `gitmeta.DetectIndexMismatch` — 4 git subprocesses, D-02 confines this to the new health RPC only, never per-navigation |
| Trust verdict (staleness banner) | Browser / Client | — | D-04 — reuses `classifyStatus`/`StatusVerdict` unchanged (`web/src/lib/status.ts:49-62`), no second verdict function |
| Error taxonomy (not-found/stale/server-error) | Browser / Client | API / Backend (error codes) | Composition of the server's existing `connect.Code` classification and the client's existing `classifyRpcError` (`web/src/lib/rpc-errors.ts:28-65`) — Claude's discretion per CONTEXT.md |
| Recursive glob pattern matching (`Files` RPC) | API / Backend | — | `internal/query/files.go` — shared by CLI, MCP, and UI; D-14 fixes it once for all three callers, not UI-local |
| Vendored shadcn-svelte component drift detection | Build / CI tooling | — | D-16 — `task web:components:drift`, local + release path, not required PR CI |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `@tanstack/svelte-table` | `9.2.4` `[VERIFIED: npm registry, official GitHub source]` | Headless table engine — sorting, column defs, row model | D-05 locked; verified pure rune-based Svelte 5 adapter (see Common Pitfalls #3) |
| `github.com/bmatcuk/doublestar/v4` | `v4.10.0` `[VERIFIED: pkg.go.dev, executed test program]` | Recursive glob matching (`**`) replacing `filepath.Match` in `internal/query/files.go` | D-14 locked; drop-in for `path.Match` per official docs, confirmed pure Go, zero transitive deps, `CGO_ENABLED=0` builds clean |
| `shadcn-svelte` CLI | `1.5.1` `[VERIFIED: npm registry]` | Vendors `table` (9 files) and `tabs` (5 files) component source into `web/src/lib/components/ui/` | D-05/D-13 locked; same CLI/version already used by 03-06, registry style `vega` per `web/components.json:11` |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `@tanstack/svelte-virtual` | `3.13.36` `[VERIFIED: npm registry]` | Row virtualization for the workbench table | Only if D-08's render-cost measurement crosses its threshold — **do not install preemptively**; confirmed to exist and support Svelte 5 (`peerDependencies: svelte: "^3.48.0 \|\| ^4.0.0 \|\| ^5.0.0"`) so it is available the moment it is needed |

### Alternatives Considered

None — D-05/D-14/D-16 already close off the alternatives (hand-rolled sort, `filepath.Match`-based `Substring` option, client-side-only glob hack); see `04-CONTEXT.md` for the maintainer's rejections. No new alternatives surfaced during this research.

**Installation:**
```bash
# Go dependency (main module, not a tool modfile)
go get github.com/bmatcuk/doublestar/v4@v4.10.0

# Frontend — from web/
pnpm add @tanstack/svelte-table@9.2.4
pnpm dlx shadcn-svelte@latest add table tabs
# @tanstack/svelte-virtual: install ONLY if D-08's threshold trips
```

**Version verification (commands actually run this session):**
```
$ npm view @tanstack/svelte-table version        -> 9.2.4
$ npm view @tanstack/svelte-table dist-tags      -> { alpha: '9.0.0-alpha.54', beta: '9.0.0-beta.80', latest: '9.2.4' }
$ npm view @tanstack/svelte-table peerDependencies -> { svelte: '^5.0.0' }
$ npm view @tanstack/svelte-virtual version       -> 3.13.36
$ npm view shadcn-svelte version                  -> 1.5.1
$ go get github.com/bmatcuk/doublestar/v4@latest  -> added v4.10.0
$ go mod graph (scratch module)                   -> doublestar-test github.com/bmatcuk/doublestar/v4@v4.10.0
                                                      go@1.27.0 toolchain@go1.27.0   (ZERO transitive deps beyond the Go toolchain itself)
$ CGO_ENABLED=0 go build ./...                     -> builds clean
```
`doublestar/v4`'s own `go.mod` (`github.com/bmatcuk/doublestar/v4@v4.10.0/go.mod`, fetched this session) declares `go 1.16` — compatible with this repo's `go 1.26.5` (`go.mod:3`, read this session).

## Package Legitimacy Audit

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|--------------|---------|-------------|
| `@tanstack/svelte-table` | npm | created 2022-05-06 `[VERIFIED: npm view time.created]` | 58,918/wk | `github.com/TanStack/table.git` | `[SUS]` (`gsd-tools package-legitimacy check`, reason: `too-new`) | **Approved** — the `too-new` signal reads the LATEST version's publish timestamp (`2026-08-28`, one day before this session), not package age. This is the identical false-positive shape 03-01-SUMMARY.md already recorded and approved for `@testing-library/jest-dom`: a recent release on a 4-year-old, high-download, org-maintained project is not a new or hijacked package. |
| `@tanstack/svelte-virtual` | npm | created 2022-07-19 `[VERIFIED: npm view time.created]` | 78,281/wk | `github.com/TanStack/virtual.git` | `[SUS]` (`too-new`) | **Approved** — same false-positive shape; only install if D-08's threshold is actually crossed. |
| `shadcn-svelte` | npm | created 2023-05-26 `[VERIFIED: npm view time.created]` | 108,654/wk | `github.com/huntabyte/shadcn-svelte.git` | `[SUS]` (`too-new`) | **Approved** — already in active use since 03-06 (`03-CONTEXT.md`); same tool, same version family (1.5.0 -> 1.5.1). |
| `github.com/bmatcuk/doublestar/v4` | Go / pkg.go.dev | published Jan 2026 (v4.10.0), module itself since 2020 | n/a (Go has no download-count registry) | `github.com/bmatcuk/doublestar` | Not covered by `gsd-tools package-legitimacy check` (Go ecosystem unsupported — confirmed: the tool's usage string only lists `npm\|pypi\|crates`) | **Approved by manual verification**: `go mod graph` shows zero transitive dependencies beyond the Go toolchain; `CGO_ENABLED=0 go build` succeeds; official docs (pkg.go.dev, fetched this session) explicitly describe `Match` as "a drop-in replacement for `path.Match()`". |

**Packages removed due to `[SLOP]` verdict:** none.
**Packages flagged as suspicious `[SUS]`:** `@tanstack/svelte-table`, `@tanstack/svelte-virtual`, `shadcn-svelte` — all three approved above under the same "recent-release-on-established-project" reasoning 03-01 already used; the planner should still add a `checkpoint:human-verify` task before install per the standard `[SUS]` protocol, even though this research found the flag to be a false positive, so a human confirms the same reasoning before the install actually runs.

*Package name provenance: `@tanstack/svelte-table`, `@tanstack/svelte-virtual`, and `doublestar/v4` were all named in `04-CONTEXT.md` (itself the product of `/gsd-discuss-phase`, which ran its own verification per that document's text) — this session independently re-verified all three against the live registry/source rather than trusting the prior verification, and every finding above is `[VERIFIED]` against a tool run in this session.*

## Architecture Patterns

### System Architecture Diagram

```
 Browser (SvelteKit SPA, adapter-static)
 ┌────────────────────────────────────────────────────────────────┐
 │  /workbench route                     /health route             │
 │  ┌──────────────────────────────┐    ┌───────────────────────┐  │
 │  │ Tabs: Impact│Affected│        │    │ Trust verdict banner  │  │
 │  │  Callers│Callees (D-13)       │    │  (classifyStatus,     │  │
 │  │                                │    │   reused unchanged)   │  │
 │  │  depth/limit/file-picker       │    │  ── above ──          │  │
 │  │  inputs, each writes           │    │ Freshness / per-lang  │  │
 │  │  workbench-url.ts state ──┐    │    │ file counts / node+   │  │
 │  │                            │    │    │ edge counts           │  │
 │  │  DataTable (shared shell,  │    │    │ Worktree-mismatch     │  │
 │  │  D-06) + per-analysis      │    │    │  loud warning (HLT-03)│  │
 │  │  columns.ts, TanStack      │    │    └───────────┬───────────┘  │
 │  │  createTable + client-side │    │                │              │
 │  │  sort (D-07)                │    │                │              │
 │  └──────────────┬─────────────┘    │                │              │
 └─────────────────┼──────────────────┴────────────────┼──────────────┘
                    │ ConnectRPC (Impact/Affected/           │ ConnectRPC
                    │ Callers/Callees/Files)                 │ (GetIndexHealth,
                    ▼                                        ▼  new 11th RPC)
        ┌───────────────────────────────────────────────────────────────┐
        │ internal/uiserver (connect-go handlers, withEngine/openEngine) │
        │  ── each RPC opens-snapshots-closes the store (SRV-04) ──      │
        │  Callers/Callees/Impact/Affected: unchanged, already bounded   │
        │  by validateDepth/clampDepth/clampAffectedDepth/validateLimit  │
        │  (internal/query/validate.go:22,26)                            │
        │  Files: internal/query/files.go — Pattern matching swaps       │
        │  filepath.Match -> doublestar.Match (D-14), same 3 callers     │
        │  (CLI, MCP, UI) fixed at once                                  │
        │  GetIndexHealth (new): maps query.StatusResult fields already  │
        │  computed (status.go:47-65) onto a new proto message; runs     │
        │  gitmeta.DetectIndexMismatch ONLY here (D-02), not on          │
        │  GetStatus's per-navigation path                               │
        └───────────────────────────────┬────────────────────────────────┘
                                         │ query.Engine (read-only)
                                         ▼
                              graphstore.Reader (Pebble, opened per RPC)
```

### Recommended Project Structure

```
web/src/
├── lib/
│   ├── workbench-url.ts          # NEW — sibling URL grammar (D-10), imports
│   │                              #   isShapeInteger/parseShapeInteger from browse-url.ts
│   ├── components/
│   │   ├── ui/
│   │   │   ├── table/            # NEW — vendored via `shadcn-svelte add table` (9 files)
│   │   │   └── tabs/             # NEW — vendored via `shadcn-svelte add tabs` (5 files)
│   │   └── workbench/
│   │       ├── DataTable.svelte  # NEW — shared shell (D-06), one instance, 4 callers
│   │       ├── impact-columns.ts     # NEW — ColumnDef<Location> for Impact
│   │       ├── affected-columns.ts   # NEW — ColumnDef<Location> for Affected
│   │       ├── callers-columns.ts    # NEW — ColumnDef<Location> for Callers
│   │       ├── callees-columns.ts    # NEW — ColumnDef<Location> for Callees
│   │       └── FilePicker.svelte     # NEW — D-15 search-and-add, reuses search.ts pattern
│   └── gen/ui_pb.ts               # REGENERATED by `task proto:gen` after ui.proto changes
├── routes/
│   ├── workbench/+page.svelte     # FILLED, not restructured (D-18)
│   └── health/+page.svelte        # FILLED, not restructured (D-18)
internal/
├── uiproto/uiv1/ui.proto          # +1 rpc (renamed from GetIndexHealth — see Pitfall #1),
│                                   #   +1 response message, map<string,int64> fields
├── uiserver/handlers.go           # +1 handler, +1 mapper function (follows statusToProto)
├── uiserver/readonly_test.go      # wantUIServiceMethods map +1 entry, literal 10 -> 11
└── query/files.go                 # filepath.Match -> doublestar.Match (2 call sites: sanity-check + real match)
```

### Pattern 1: The established handler-mapper convention (follow exactly, do not invent a new one)

**What:** Every existing RPC handler (`Callers`, `Callees`, `Files`, `GetStatus`) follows the same three-part shape: (1) a named `xToProto` mapper function translating one engine struct field-for-field onto its proto message, (2) the handler itself wrapped in `withEngine(ctx, s.repoPath, func(eng *query.Engine) error { ... })`, (3) errors returned unwrapped from inside the closure — `withEngine`/`mapEngineError` do the Connect-code translation.

**When to use:** For the new health RPC's handler, exactly this shape.

**Example (verbatim, read this session):**
```go
// Source: internal/uiserver/handlers.go:218-227 (statusToProto) and :461-478 (Callers handler)
func statusToProto(result query.StatusResult, commitSHA string) *uiv1.GetStatusResponse {
	return &uiv1.GetStatusResponse{
		Initialized: result.Initialized,
		Version:     result.Version,
		NodeCount:   result.NodeCount,
		EdgeCount:   result.EdgeCount,
		FileCount:   result.FileCount,
		Stale:       result.Stale,
		CommitSha:   commitSHA,
		StoreExists: true,
	}
}

func (s *uiService) Callers(ctx context.Context, req *connect.Request[uiv1.CallersRequest]) (*connect.Response[uiv1.CallersResponse], error) {
	var resp *uiv1.CallersResponse
	err := withEngine(ctx, s.repoPath, func(eng *query.Engine) error {
		result, err := eng.Callers(req.Msg.GetSymbol(), int(req.Msg.GetLimit()))
		if err != nil {
			return err
		}
		resp = &uiv1.CallersResponse{
			Symbol:  result.Symbol,
			Callers: locationsToProto(result.Callers),
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}
```
The new health RPC's handler should call `eng.Status(ctx)` (already what `GetStatus` calls) inside `withEngine`, then call `gitmeta.DetectIndexMismatch` (D-02, only here), then build the response via a new `healthToProto`-style mapper — matching this shape, not `GetStatus`'s special-cased `openEngine`-direct pattern (that pattern exists ONLY because `GetStatus` must answer even when the store can't open; the health RPC has no such degrade-and-still-answer requirement per `04-CONTEXT.md`).

### Pattern 2: TanStack Table v9's actual Svelte 5 API (verified from live `main`-branch source, NOT the public docs page — see Pitfall #3)

**What:** `createTable()` builds a rune-backed table instance; `table.atoms.<slice>.get()` and `table.getRowModel()`/`table.getHeaderGroups()` read reactively inside a template, `$derived`, or `$effect`; `<FlexRender header={header} />` / `<FlexRender cell={cell} />` render header/cell content.

**When to use:** Every one of the four analysis tables (D-06).

**Example — sorting (Source: `github.com/TanStack/table` `main` branch, `examples/svelte/sorting/src/{tableHelper.svelte.ts,App.svelte}`, fetched verbatim this session, matches installed `9.2.4`):**
```ts
// tableHelper.svelte.ts
import {
  createSortedRowModel,
  rowSortingFeature,
  sortFn_alphanumeric,
  sortFn_text,
  tableFeatures,
} from '@tanstack/svelte-table'

export const features = tableFeatures({
  rowSortingFeature,
  sortedRowModel: createSortedRowModel(),
  sortFns: {
    alphanumeric: sortFn_alphanumeric,
    text: sortFn_text,
  },
})
```
```svelte
<!-- App.svelte -->
<script lang="ts">
  import type { ColumnDef } from '@tanstack/svelte-table'
  import { FlexRender, createTable, renderComponent } from '@tanstack/svelte-table'
  import { features } from './tableHelper.svelte'

  let data = $state(makeData(1_000))

  const table = createTable({
    features,
    get data() { return data },
    columns,
    debugTable: true,
  })
</script>

<table>
  <thead>
    {#each table.getHeaderGroups() as headerGroup (headerGroup.id)}
      <tr>
        {#each headerGroup.headers as header (header.id)}
          <th colSpan={header.colSpan}>
            {#if !header.isPlaceholder}
              <FlexRender header={header} />
            {/if}
          </th>
        {/each}
      </tr>
    {/each}
  </thead>
  <tbody>
    {#each table.getRowModel().rows as row (row.id)}
      <tr>
        {#each row.getAllCells() as cell (cell.id)}
          <td><FlexRender cell={cell} /></td>
        {/each}
      </tr>
    {/each}
  </tbody>
</table>
```
Header click-to-sort: `header.column.getToggleSortingHandler()` as the `onclick` handler; `header.column.getIsSorted()` returns `'asc' | 'desc' | false` for the indicator.

**Column definitions for a `Location` row (`name`, `kind`, `file_path`, `start_line` — string/string/string/int32, `ui.proto:100-105`):** TanStack Table auto-detects sort comparator by JS value type when no explicit `sortFn` is set (confirmed via the official React v9 migration example's comment: "this column will sort in ascending order by default since it is a string column" / "...descending order by default since it is a number column"). For the three string columns an explicit `sortFn: 'alphanumeric'` is the safer choice for file paths (mixed casing/segments); `start_line` needs no explicit `sortFn` — numeric auto-detection is correct.

### Pattern 3: shadcn-svelte `table` + TanStack v9 composition (official example, matches this project's exact vendoring convention)

**What:** shadcn-svelte's vendored `Table.Root`/`Table.Header`/`Table.Row`/`Table.Head`/`Table.Body`/`Table.Cell` wrap plain HTML table elements (`table/table.svelte` etc.) with `FlexRender` doing the actual cell/header content projection.

**Example (Source: `github.com/TanStack/table` `main` branch, `examples/svelte/lib-shadcn/src/App.svelte`, fetched verbatim this session):**
```svelte
<script lang="ts">
  import { FlexRender, createTable, /* ... */ } from '@tanstack/svelte-table'
  import * as Table from '@/lib/components/ui/table'
  // ...
  const table = createTable({ features, columns, get data() { return data } })
</script>

<Table.Root>
  <Table.Header>
    {#each table.getHeaderGroups() as headerGroup (headerGroup.id)}
      <Table.Row>
        {#each headerGroup.headers as header (header.id)}
          <Table.Head colspan={header.colSpan}>
            {#if !header.isPlaceholder}
              {#if header.column.getCanSort()}
                <Button variant="ghost" size="sm" onclick={header.column.getToggleSortingHandler()}>
                  <FlexRender {header} />
                </Button>
              {:else}
                <FlexRender {header} />
              {/if}
            {/if}
          </Table.Head>
        {/each}
      </Table.Row>
    {/each}
  </Table.Header>
  <Table.Body>
    {#each table.getRowModel().rows as row (row.id)}
      <Table.Row>
        {#each row.getAllCells() as cell (cell.id)}
          <Table.Cell><FlexRender {cell} /></Table.Cell>
        {/each}
      </Table.Row>
    {:else}
      <Table.Row><Table.Cell colspan={columns.length} class="h-24 text-center">No results.</Table.Cell></Table.Row>
    {/each}
  </Table.Body>
</Table.Root>
```
This `{:else}` empty-state branch inside `{#each}` maps directly onto D-06's requirement that empty/loading/error states be designed per analysis (Claude's discretion item).

### Anti-Patterns to Avoid

- **Using the store-based `createSvelteTable(writable(...))` API shown on `tanstack.com/table/latest/docs/framework/svelte/`:** that page (fetched via Context7 this session) describes the pre-v9 API and is stale relative to the installed `9.2.4` package — see Pitfall #3. Using it will not compile against the real `9.2.4` exports.
- **Adding a client-side sort parameter to any RPC:** D-07 forbids this explicitly; server-side sorting was evaluated and deferred (`04-CONTEXT.md` Deferred Ideas) because it would need to run before the limit applies, which is out of scope this phase.
- **A second client-side row cap on top of `MaxLimit`:** D-08 explicitly names this as rejected — virtualize, don't re-cap what the server already bounded.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|--------------|-----|
| Table sorting, column definitions | A hand-rolled `Array.prototype.sort` + manual header click state | `@tanstack/svelte-table`'s `rowSortingFeature`/`createSortedRowModel` | D-05 locked; maintainer directive ("do the right and idiomatic thing"); shadcn-svelte's own `data-table` guide states the TanStack dependency requirement explicitly |
| Row virtualization at scale (up to `MaxLimit`=1000 rows) | Manual windowing / IntersectionObserver-based lazy render | `@tanstack/svelte-virtual` (conditional on D-08's threshold) | Confirmed available and Svelte-5-compatible now, so no gap exists between "threshold trips" and "library is ready" |
| Recursive glob matching (`**` crossing `/`) | A bespoke recursive-descent path matcher, or the rejected `Substring` option | `doublestar.Match` | D-14 locked; verified drop-in for `path.Match`, verified `**` behavior live |
| Debounce + request-cancellation for the file-picker's search-as-you-type | A second bespoke `AbortController`/timer implementation | `search.ts`'s existing controller pattern (`liveRequestId`, `AbortController`, ~150ms debounce — `web/src/lib/search.ts:16-140`, read this session) | D-15 locked; one implementation of this pattern already exists in this exact codebase |
| Depth/limit range validation in the UI | A second `if (depth > 50)` check mirroring the server | Nothing — pass straight through; server already refuses/clamps | D-07/ROADMAP Notes — the server's `validateDepth`/`clampDepth`/`clampAffectedDepth`/`validateLimit` (`internal/query/validate.go:71-142`, read this session) are the only copy of this rule |
| Health-page trust verdict | A second `classifyHealthStatus`-style function reading `IndexHealth.ReindexRecommended`/`PendingChanges`/worktree mismatch | `classifyStatus` (`web/src/lib/status.ts:49-62`), unchanged | D-04 locked; two verdict functions over one index can disagree — exactly the failure HLT-02 exists to prevent |

**Key insight:** every "don't hand-roll" item in this phase is already a locked decision in `04-CONTEXT.md` (D-05, D-08, D-14, D-15, D-07, D-04) — this research's job was to confirm each chosen library actually does what the decision assumes it does, not to re-litigate the choice.

## Common Pitfalls

### Pitfall 1 (CRITICAL — verified, not hypothetical): The RPC name `GetIndexHealth` collides with an existing negative-verb guard

**What goes wrong:** `04-CONTEXT.md` names the new RPC `GetIndexHealth` throughout (D-01, D-02, D-03, canonical refs, code context — at least 8 occurrences). `internal/uiserver/readonly_test.go:89-119`'s `TestUIServiceDeclaresNoMutatingMethod` iterates every method on `uiv1connect.UIServiceHandler` and fails if `strings.Contains(name, verb)` for any `verb` in a fixed list (`internal/uiserver/readonly_test.go:89-93`, read verbatim this session):
```go
var mutatingVerbs = []string{
	"Create", "Update", "Delete", "Remove", "Set", "Put", "Post",
	"Write", "Add", "Insert", "Mutate", "Patch", "Modify", "Sync",
	"Reindex", "Index", "Clear", "Reset", "Save",
}
```
`"Index"` is literally in this list (added, presumably, to catch a hypothetical `Reindex`/`IndexFiles` mutating method). Confirmed live this session:
```
$ python3 -c "print('Index' in 'GetIndexHealth')"
True
```
Any RPC name containing the substring `Index` — `GetIndexHealth`, `GetIndexStatus`, `IndexHealthCheck`, etc. — will fail this test on first compile-and-run.

**Why it happens:** the guard is a blunt substring check (documented in its own comment as the deliberate complementary negative-style check to the positive `wantUIServiceMethods` set-equality test), not a word-boundary or verb-position check.

**How to avoid:** rename the RPC to avoid the substring `Index` entirely — `GetHealth` or `GetGraphHealth` both satisfy HLT-01/02/03's "index health" concept without embedding the trigger string, and the RPC's proto-level identity does not need to match D-01's prose spelling; a name substitution here changes no locked *behavior*. **Do not** try to work around this by editing `mutatingVerbs` to special-case `GetIndexHealth` — that weakens a guard whose entire purpose is catching an accidentally-mutating verb, for a naming preference. If the planner nonetheless wants to keep the literal name `GetIndexHealth`, the correct fix is a documented, narrowly-scoped allowlist entry in the test (e.g. an explicit `except` set checked before the substring loop) — not silently renaming the pattern or removing `"Index"` from the list.

**Warning signs:** `go test ./internal/uiserver/...` failing with `method "GetIndexHealth" contains mutating verb "Index"` the moment the new RPC is added to `ui.proto` and codegen runs.

### Pitfall 2 (corrects the research brief itself): `proto:drift`'s floor is per generated FILE, not per RPC — it does NOT need updating; a DIFFERENT test does

**What goes wrong (if believed uncritically):** the research brief's priority list describes a "PLANNER TRAP: the floor is a hardcoded `nfiles -lt N` check that must be updated when the generated file count changes," implying adding the 11th RPC requires a `Taskfile.yml` edit. This is not correct for this specific change.

**What's actually true, read verbatim this session (`Taskfile.yml:339-348`):**
```bash
files=$(git ls-files -- 'internal/schema/*.pb.go' 'internal/uiproto/uiv1/*.pb.go' 'internal/uiproto/uiv1/uiv1connect/*.connect.go' 'web/src/lib/gen/*.ts')
nfiles=0
if [ -n "${files}" ]; then
  nfiles=$(printf '%s\n' "${files}" | wc -l | tr -d ' ')
fi
echo "proto:drift: compared ${nfiles} generated files"
if [ "${nfiles}" -lt 4 ]; then
  echo "::error::proto:drift: enumerated only ${nfiles} committed generated files ..."
  exit 1
fi
```
This counts **files matching four glob patterns**, currently 4 files total: `internal/schema/graph.pb.go` (1), `internal/uiproto/uiv1/ui.pb.go` (1), `internal/uiproto/uiv1/uiv1connect/ui.connect.go` (1), `web/src/lib/gen/ui_pb.ts` (1). Adding one RPC and one response message to the ALREADY-EXISTING `ui.proto` file adds new content to these SAME four files (bigger `ui.pb.go`, bigger `ui.connect.go`, bigger `ui_pb.ts`) — it does not create a fifth file. `nfiles` stays 4, still `>= 4`, the floor comparison is unaffected. **No `Taskfile.yml` change is needed for `proto:drift`.**

**The test that DOES need a literal update** is a completely different one, in a different file: `internal/uiserver/readonly_test.go:31-42` (`wantUIServiceMethods`, a `map[string]struct{}` of the 10 current RPC names) and line 61 (`if len(got) != 10`). Both must gain the new RPC's name and become `11`, following the exact precedent already set when `GetPermalink` was added as the 10th RPC in plan 03-05 (comment at `readonly_test.go:26-30`, read this session, documents that prior update).

**How to avoid:** the planner should scope a task to `internal/uiserver/readonly_test.go`'s `wantUIServiceMethods` map and its `10`/`len(got) != 10` literal — not to `Taskfile.yml`'s `proto:drift` target.

### Pitfall 3: The public TanStack Table docs page is stale relative to the just-shipped v9 Svelte adapter

**What goes wrong:** fetching `tanstack.com/table/latest/docs/installation.md` (via Context7 this session) returns: *"The `@tanstack/svelte-table` package supports Svelte 3 and Svelte 4. For Svelte 5, a built-in adapter is not yet available, but you can still use TanStack Table by installing the `@tanstack/table-core` package and integrating a custom community-provided adapter."* This directly contradicts what the actually-published, actually-installed package requires.

**Why it happens:** `@tanstack/svelte-table@9.2.4` was published `2026-08-28` (confirmed via `npm view @tanstack/svelte-table time.modified`-equivalent registry metadata, and independently via `04-CONTEXT.md`'s own D-05 note, "`time.modified` 2026-08-28") — one day before this research session. The docs site has not caught up. The real, current package (verified from `github.com/TanStack/table`'s `main` branch `packages/svelte-table/package.json`, fetched this session) declares `peerDependencies: { svelte: "^5.0.0" }` — it dropped Svelte 3/4 support entirely and ships a full runes-based rewrite (`createTable`, `$effect.pre`, `svelteReactivity()`), the opposite of what the docs page says.

**How to avoid:** do not trust `tanstack.com/table/latest/docs/framework/svelte/*` pages for this specific version. Use the real source (`github.com/TanStack/table` `main` branch, `packages/svelte-table/src/`) and the real examples (`examples/svelte/sorting/`, `examples/svelte/lib-shadcn/`) as the source of truth — both fetched and quoted verbatim in this document's Code Examples / Pattern sections.

**Warning signs:** any code following the docs-page pattern (`import { writable } from 'svelte/store'`, `createSvelteTable(options)` taking a `Writable<TableOptions>`) will fail to type-check or resolve imports against the installed `9.2.4` package, whose actual exports are `createTable`, `createTableHook`, `createTableState`, `FlexRender`, `renderComponent`, `renderSnippet` (confirmed via `packages/svelte-table/src/index.ts`, fetched verbatim this session) — there is no `createSvelteTable` export at all in `9.2.4`.

### Pitfall 4: shadcn-svelte's `table` and `tabs` additions are NOT symmetric with respect to `pnpm-lock.yaml` coverage

**What goes wrong:** treating both components as equally "safe" or equally "risky" under D-09's audit-gate table.

**What's actually true (confirmed via the live registry JSON, fetched this session):** `table` (9 files: `table/index.ts`, `table-body.svelte`, `table-caption.svelte`, `table-cell.svelte`, `table-footer.svelte`, `table-head.svelte`, `table-header.svelte`, `table-row.svelte`, `table.svelte`) declares **no** `devDependencies`/`dependencies`/`registryDependencies` at all — it's plain HTML-element wrappers, adds zero new npm packages. `tabs` (5 files) declares `devDependencies: ["bits-ui@^2.16.3", "@internationalized/date@^3.12.0", "tailwind-variants@^3.3.0"]` — all three ranges are already satisfied by the versions 03-06 installed (`bits-ui@^2.19.0`, `@internationalized/date@^3.12.3`, `tailwind-variants@^3.3.1`, `web/package.json:26,25,32`, read this session), so `add tabs` should write no NEW `package.json` entry either, but the CLI does consult the registry over the network for both — D-09's own table applies identically to both: the `.svelte` source is the unscanned surface for both components, regardless of whether either happens to add an npm dependency this time.

**How to avoid:** no code change implied; this is a documentation nuance for whoever writes the Package Legitimacy Audit entries and the `web:components:drift` task description — don't claim `tabs` "requires no new dependency" as a blanket safety argument; its declared devDependencies are satisfied THIS TIME, not structurally guaranteed to remain so.

### Pitfall 5: There is no existing "optional nested message" precedent in `ui.proto` — pick a representation deliberately

**What goes wrong:** assuming `optional WorktreeMismatch mismatch = N;` is needed to make the field nil-checkable, mirroring the `optional int32 line = 3;` pattern used for `GetNodeDetailRequest`/`GetPermalinkRequest` (`ui.proto:338,558-559`, read this session).

**Why it's a mismatch:** the `optional` keyword in proto3 is needed for **scalar** field presence-tracking (an `int32` has no natural "unset" value distinct from `0`). Message-typed fields are ALREADY presence-tracked in proto3 by construction — Go generates `*uiv1.WorktreeMismatch`, naturally `nil`-able, with no `optional` keyword required. The codebase's one precedent for representing an optional *value* — `PermalinkAvailability`'s three-state enum (`ui.proto:561-590`) plus flattened string fields, never a nested optional message — is solving a different problem (distinguishing "checked, absent" from "could not check" for a value that has no natural analog to `gitmeta.Mismatch`'s pointer-or-nil shape).

**How to avoid:** define a plain `WorktreeMismatch` message mirroring `gitmeta.Mismatch{WorktreeRoot, IndexRoot}` (`internal/gitmeta/detect.go:10-13`, read this session — fields are `WorktreeRoot string`, `IndexRoot string`) and use it as an ordinary (non-`optional`) message field on the health response; `nil` on the Go side already means "no mismatch" with zero extra ceremony. This is the more idiomatic and lower-effort choice, not merely an acceptable one.

## Code Examples

### doublestar root-level and nested match — verified live this session

```
$ cat main.go
package main

import (
	"fmt"
	"github.com/bmatcuk/doublestar/v4"
)

func main() {
	tests := []struct{ pattern, name string }{
		{"**/*claude*", "claudeassets.go"},
		{"**/*detail*", "internal/query/detail.go"},
		{"**/*claude*", "internal/query/claude.go"},
		{"**/*term*", "term.go"},
		{"**/*term*", "internal/foo/term.go"},
	}
	for _, t := range tests {
		matched, err := doublestar.Match(t.pattern, t.name)
		fmt.Printf("doublestar.Match(%q, %q) = %v, err=%v\n", t.pattern, t.name, matched, err)
	}
}

$ go run main.go
doublestar.Match("**/*claude*", "claudeassets.go") = true, err=<nil>
doublestar.Match("**/*detail*", "internal/query/detail.go") = true, err=<nil>
doublestar.Match("**/*claude*", "internal/query/claude.go") = true, err=<nil>
doublestar.Match("**/*term*", "term.go") = true, err=<nil>
doublestar.Match("**/*term*", "internal/foo/term.go") = true, err=<nil>
```
`**` matches zero path segments (root-level case preserved) AND crosses `/` (nested case, the bug being fixed) — both confirmed in one program, module `github.com/bmatcuk/doublestar/v4 v4.10.0`.

The two call sites in `internal/query/files.go` that need `filepath.Match` -> `doublestar.Match` (both read this session):
```go
// Source: internal/query/files.go:147-149 (sanity-check call) and :167-172 (real match)
if opts.Pattern != "" {
    if _, err := filepath.Match(opts.Pattern, "sanity-check"); err != nil {
        return FilesResult{}, fmt.Errorf("query: invalid pattern %q: %w", opts.Pattern, err)
    }
}
// ...
if opts.Pattern != "" {
    matched, err := filepath.Match(opts.Pattern, p)   // p is already filepath.ToSlash(f.Path) — line 157
    if err != nil {
        return FilesResult{}, fmt.Errorf("query: invalid pattern %q: %w", opts.Pattern, err)
    }
    if !matched {
        continue
    }
}
```
Because `p` is already forward-slashed (`filepath.ToSlash`, `files.go:157`) before matching, `doublestar.Match` (not `doublestar.PathMatch`) is the correct drop-in — `Match` "always uses '/' as the path separator" per its own doc comment (pkg.go.dev, fetched this session), which is exactly what this call site already produces.

### Proto `map<string, int64>` — existing precedent in this repo

```protobuf
// Source: internal/schema/graph.proto:94 (read this session) — establishes the
// idiomatic map syntax already used in this codebase, for Node.metadata:
map<string, string> metadata = 7;
```
The new health response's `FilesByLanguage`/`NodesByKind`/`EdgesByKind` (all `map[string]int64` on the Go side, `StatusResult` struct fields, `status.go:53-55`) should follow this exact syntax with `int64` values: `map<string, int64> files_by_language = N;` etc. Go codegen produces `map[string]int64` automatically — no special handling needed beyond what `graph.pb.go`'s existing `Metadata map[string]string` field already demonstrates works cleanly through this repo's `buf`/`protoc-gen-go` pipeline.

### `web:drift`'s floor-and-digest shape — the sibling convention `web:components:drift` should follow structurally (though NOT its hash-comparison strategy — see Pitfall discussion below)

```bash
# Source: Taskfile.yml:1120-1181 (web:drift), read verbatim this session
echo "web:drift: hashed ${SRC_N} source files"
echo "web:drift: manifested ${OUT_N} output files"
SRC_FLOOR=8
if [ "${SRC_N}" -lt "${SRC_FLOOR}" ]; then
  echo "::error::web:drift: hashed only ${SRC_N} source files, want at least ${SRC_FLOOR} ..." >&2
  exit 1
fi
# ... (byte-for-byte digest comparison against a committed manifest file)
```
`web:components:drift` (D-16) has no build-marker file to compare against (unlike `web:drift`'s `.build-manifest`), so its shape should follow `proto:drift`'s regenerate-and-byte-compare pattern instead: re-run `pnpm dlx shadcn-svelte@1.5.1 add table tabs -y -o` (pinned version, not `@latest`, so a registry-side change doesn't silently pass) into a scratch directory, byte-compare against `web/src/lib/components/ui/{table,tabs}/`, and report a count-before-comparing per rule `84d1gfpywd` — e.g. "compared 14 vendored component files" (9 + 5), with a floor of 14 (or whatever the actual committed count is once both components are added), mirroring `proto:drift`'s `echo "proto:drift: compared ${nfiles} generated files"` (`Taskfile.yml:346`) done BEFORE any comparison.

```bash
# Source: Taskfile.yml:339-348 (proto:drift's regenerate-and-byte-compare shape — read verbatim
# this session; this is the pattern to mirror for web:components:drift, not web:drift's hashing)
echo "proto:drift: compared ${nfiles} generated files"
if [ "${nfiles}" -lt 4 ]; then
  echo "::error::proto:drift: enumerated only ${nfiles} committed generated files ..."
  exit 1
fi
# ... regenerate into ${scratch}/gen, then:
if ! cmp -s "${f}" "${fresh}"; then
  echo "::error::proto:drift: ${f} differs from the pinned toolchain's regeneration ..."
  drift_found=1
fi
```

### shadcn-svelte `table`/`tabs` registry contents (fetched from the live `vega`-style registry this session)

```
$ curl -s https://shadcn-svelte.com/registry/styles/vega/table.json | jq '.files[].target'
"table/index.ts" "table/table-body.svelte" "table/table-caption.svelte" "table/table-cell.svelte"
"table/table-footer.svelte" "table/table-head.svelte" "table/table-header.svelte"
"table/table-row.svelte" "table/table.svelte"
# devDependencies: none — pure HTML-element wrappers

$ curl -s https://shadcn-svelte.com/registry/styles/vega/tabs.json | jq '.devDependencies, [.files[].target]'
["bits-ui@^2.16.3", "@internationalized/date@^3.12.0", "tailwind-variants@^3.3.0"]
["tabs/index.ts", "tabs/tabs-content.svelte", "tabs/tabs-list.svelte", "tabs/tabs-trigger.svelte", "tabs/tabs.svelte"]
```
CLI invocation, matching 03-06's established convention (`03-RESEARCH.md:126`, read this session): `pnpm dlx shadcn-svelte@latest add command popover` — for this phase: `pnpm dlx shadcn-svelte@latest add table tabs` (run from `web/`, per `web/components.json`'s already-configured aliases).

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|-------------------|----------------|--------|
| `@tanstack/svelte-table` v8 store-based API (`createSvelteTable(writable(options))`) | v9 rune-based API (`createTable(options)`, `table.atoms.<slice>.get()`) | `9.0.0` (beta) through `9.2.4` (stable, published 2026-08-28) | A full rewrite, not an incremental bump — v8-era code (including the public docs site's own current Svelte examples, as of this session) will not run against `9.2.4`; see Pitfall #3 |
| TanStack Table's per-framework feature bundling (v8: `getSortedRowModel()` bundled by default) | `tableFeatures({ rowSortingFeature, sortedRowModel: createSortedRowModel(), sortFns })` — explicit feature registration required | v9 across all frameworks (confirmed via the official React migration guide, fetched this session, describing the identical pattern for `@tanstack/react-table`) | Every feature (sorting, pagination, filtering) must be explicitly opted into via `tableFeatures()`; nothing is bundled by default anymore |

**Deprecated/outdated:** the `tanstack.com/table/latest/docs/framework/svelte/` pages, as fetched via Context7 in this session, describe the pre-v9 Svelte adapter and should not be used as an implementation reference for this phase — use the `main`-branch source and `examples/svelte/` directory instead (both cited throughout this document).

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|-----------------|
| A1 | The `web:components:drift` floor count should be "14" (9 table files + 5 tabs files) | Code Examples, `web:drift`/`proto:drift` shape section | LOW — this is a planner-computed value at task-writing time, not a locked research fact; the actual floor must be derived from the files genuinely committed after `add table tabs` runs, not hardcoded from this document's count, which could shift if the registry changes between now and implementation |
| A2 | Renaming the RPC away from `GetIndexHealth` (Pitfall #1) does not violate any locked CONTEXT.md decision, since D-01 through D-16 describe the RPC's *behavior* (new, additive, split from GetStatus) rather than committing to the literal string `GetIndexHealth` as a wire-contract requirement | Common Pitfalls #1 | MEDIUM — if the maintainer intended `GetIndexHealth` as a hard-locked literal name (not just descriptive prose), this assumption is wrong and the planner should instead add the documented `mutatingVerbs` exception route described as the fallback in Pitfall #1, or raise this as a discuss-phase follow-up before implementation |

## Open Questions

1. **Exact final RPC name for the new health endpoint.**
   - What we know: `GetIndexHealth` fails `TestUIServiceDeclaresNoMutatingMethod` (verified, Pitfall #1); `GetHealth`/`GetGraphHealth` do not contain any of the 19 listed mutating-verb substrings (checked each against the full list: `Create, Update, Delete, Remove, Set, Put, Post, Write, Add, Insert, Mutate, Patch, Modify, Sync, Reindex, Index, Clear, Reset, Save` — none match `GetHealth` or `GetGraphHealth`).
   - What's unclear: whether the maintainer would prefer the documented-exception route (keep `GetIndexHealth`, add a narrow allowlist to the test) over a rename.
   - Recommendation: the planner should pick one and record it as a plan-level decision; either is mechanically sound, this is a naming/test-scope call, not a technical blocker.

2. **`web:components:drift`'s exact byte-compare granularity.**
   - What we know: `proto:drift`'s shape (regenerate into scratch, byte-compare against committed files, floor-assert the count) is the right structural model (D-16's own text: "re-runs the pinned-version CLI for each added component and byte-diffs against what is committed").
   - What's unclear: whether the CLI's `add` command is idempotent/deterministic enough to regenerate byte-identical output for components already present in the target directory (untested this session — the CLI was only invoked with `--help`, not a live `add` against a scratch tree) — `-o`/`--overwrite` and `-y` flags exist and were confirmed via `--help` output, but their exact interaction with a target directory that already has different content was not exercised live.
   - Recommendation: the plan should include a task-level verification step that runs `pnpm dlx shadcn-svelte@1.5.1 add table tabs -y -o` into a genuinely empty scratch directory once, confirms the output byte-matches what gets committed, before wiring the drift task's pass/fail logic around that assumption.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|--------------|-----------|---------|----------|
| Go toolchain | `doublestar/v4` dependency add, `proto:gen`/`proto:drift` regeneration | ✓ | `go 1.26.5` (`go.mod:3`) | — |
| `git` | `proto:drift`'s `git ls-files` enumeration, `web:drift`'s source enumeration | ✓ | — | — |
| `pnpm` | Frontend dependency install, `shadcn-svelte` CLI via `pnpm dlx` | ✓ (per `web/package.json:6`, `packageManager: pnpm@11.23.0`) | 11.23.0 | — |
| `buf` (pinned via `go.tool-proto.mod`) | `proto:gen`/`proto:drift` regeneration for the new RPC | ✓ (existing toolchain, unchanged this phase) | pinned, per Taskfile precondition | — |
| Network access to `shadcn-svelte.com` registry | `pnpm dlx shadcn-svelte add table tabs` | ✓ (confirmed reachable this session — registry JSON fetched successfully) | — | D-16 deliberately keeps `web:components:drift` OUT of required PR CI for exactly this reason (network dependency in a merge gate) |

**Missing dependencies with no fallback:** none.
**Missing dependencies with fallback:** none — every dependency this phase needs was confirmed reachable and correctly versioned this session.

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Frontend framework | Vitest `4.1.11` (`web/package.json:29`), `@testing-library/svelte@5.4.2` |
| Frontend config | `web/vite.config.ts` (no separate `vitest.config.ts` — confirmed via `head -1` showing the `/// <reference types="vitest/config" />` triple-slash directive inline) |
| Frontend test location | `web/tests/*.test.ts` (confirmed via `find` this session — NOT `web/src/**/*.test.ts`; existing convention: `browse-url.test.ts`, `status.test.ts`, `rpc-errors.test.ts`, `search.test.ts`, etc., 16 files total) |
| Frontend quick run | `cd web && pnpm test` (= `vitest run`, `web/package.json:14`) |
| Backend framework | Go `testing` package, no third-party test framework |
| Backend config | none (stdlib) |
| Backend quick run | `go test ./internal/uiserver/... ./internal/query/... ./internal/gitmeta/...` |
| Backend full suite | `task test:unit` (`Taskfile.yml:117-133` — `go test` over every package except `internal/daemon`) |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|---------------------|---------------|
| WRK-01 | Impact depth control updates blast radius without navigation | component/unit | `pnpm test -- workbench-impact` (new file) | ❌ Wave 0 |
| WRK-02 | Multi-file selection via Affected | component/unit | `pnpm test -- workbench-affected` (new file) | ❌ Wave 0 |
| WRK-03 | Callers/Callees with adjustable limit | component/unit | `pnpm test -- workbench-callers-callees` (new file) | ❌ Wave 0 |
| WRK-04 | Sortable tables, 3-way error taxonomy | component/unit | `pnpm test -- data-table`, extends existing `rpc-errors.test.ts` | ❌ Wave 0 (table), ✓ extends existing (errors) |
| HLT-01/02/03 | Health RPC + verdict rendering + worktree warning | Go unit (`internal/uiserver`) + frontend component | `go test ./internal/uiserver/... -run Health`, `pnpm test -- health-page` | ❌ Wave 0 (both) |
| — | `readonly_test.go` set-equality/mutating-verb guards stay green with the new RPC | Go unit | `go test ./internal/uiserver/... -run TestUIServiceMethodSetIsExactlyTheReadSet` and `-run TestUIServiceDeclaresNoMutatingMethod` | ✓ exists, needs literal update (Pitfall #1, #2) |
| — | `doublestar` root-level + nested glob regression | Go unit | `go test ./internal/query/... -run TestFiles` (extend existing file, add root-level + nested cases per D-14's own instruction) | ✓ existing test file to extend (`internal/query` has `Files`-related tests already, per D-14's "prove it with a test carrying both a nested and a root-level case") |

### Sampling Rate

- **Per task commit:** `pnpm test` (frontend changes), `go test ./internal/uiserver/... ./internal/query/...` (backend changes)
- **Per wave merge:** `task test:unit` + `cd web && pnpm test`
- **Phase gate:** `task proto:drift`, `task web:drift`, new `task web:components:drift`, full `task test:unit`, and `cd web && pnpm test` all green before `/gsd-verify-work`

### Wave 0 Gaps

- [ ] `web/tests/workbench-*.test.ts` — new component tests for the four analysis tabs (WRK-01..04)
- [ ] `web/tests/health-page.test.ts` — new component test for the health view (HLT-01..03)
- [ ] `web/tests/workbench-url.test.ts` — new URL-grammar round-trip test, mirroring `browse-url.test.ts`'s existing shape
- [ ] `internal/uiserver/handlers_test.go` (or a new `health_test.go`) — Go unit coverage for the new health handler, mirroring existing `Callers`/`Callees` handler test shape
- [ ] Extend `internal/query`'s existing `Files`-pattern test to add the root-level + nested `doublestar` regression cases D-14 itself requires

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|-----------------|---------|---------------------|
| V5 Input Validation | yes | Already-existing `validateDepth`/`clampDepth`/`clampAffectedDepth`/`validateLimit` (`internal/query/validate.go:71-142`) — this phase adds no new validation surface, only new callers of the existing one; D-07 explicitly forbids a client-side duplicate |
| V4 Access Control | no | No new auth/access-control surface — SRV-03's read-only-by-construction posture (unauthenticated loopback-only server) is unchanged; the new RPC is read-only like all ten existing ones |
| V2/V3 Authentication/Session | no | No new auth surface this phase |
| V6 Cryptography | no | No cryptographic operation introduced |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|-------------------------|
| Unbounded traversal/allocation via crafted depth/limit values on the new workbench UI's inputs | Denial of Service | Already mitigated server-side; D-07 forbids re-implementing the check client-side where it could drift or be bypassed by a direct RPC call bypassing the UI |
| A malformed or adversarial glob pattern passed to `Files` via the new file-picker | Denial of Service (ReDoS-adjacent) | `doublestar.Match`'s pattern compilation is bounded (no backtracking regex engine underneath — it is a glob matcher, not a general regex engine); existing sanity-check call (`files.go:147-149`) already rejects malformed patterns before the real scan runs, unchanged by the D-14 swap |
| The new health RPC spawning git subprocesses (`gitmeta.DetectIndexMismatch`, up to 4 per call) being hit repeatedly by a client to exhaust process/fd budget | Denial of Service | D-02 already scopes this to the health RPC alone (never per-navigation); no additional rate-limiting is in scope for this phase per SRV-03's existing "single local user" threat model |

## Sources

### Primary (HIGH confidence — tool-verified this session)

- `go run` against a scratch module — `doublestar.Match` root-level and nested behavior, verbatim output quoted above
- `go mod graph`, `CGO_ENABLED=0 go build` — `doublestar/v4` zero-transitive-deps, pure-Go confirmation
- `npm view` (multiple packages) — versions, dist-tags, peerDependencies, `time.created`
- `gsd-tools query package-legitimacy check --ecosystem npm` — verdicts for `@tanstack/svelte-table`, `@tanstack/svelte-virtual`, `shadcn-svelte`
- `pkg.go.dev/github.com/bmatcuk/doublestar/v4` (fetched) — official `Match` doc comment, "drop-in replacement for `path.Match()`"
- `github.com/TanStack/table` `main` branch, fetched verbatim via `curl`/raw.githubusercontent.com — `packages/svelte-table/src/{index.ts,createTable.svelte.ts,createTableState.svelte.ts,FlexRender.svelte}`, `packages/svelte-table/package.json` (peerDependencies), `examples/svelte/{sorting,lib-shadcn}/src/*` — the real, current v9 Svelte API surface
- `shadcn-svelte.com/registry/styles/vega/{table,tabs}.json` (fetched) — exact file lists, `devDependencies`
- This repository, read this session: `internal/uiproto/uiv1/ui.proto` (full), `internal/query/status.go`, `internal/query/validate.go`, `internal/query/files.go`, `internal/gitmeta/detect.go`, `internal/uiserver/handlers.go`, `internal/uiserver/readonly_test.go`, `internal/schema/graph.proto`, `Taskfile.yml` (`proto:gen`/`proto:drift`/`web:drift`/`test:unit`), `web/package.json`, `web/components.json`, `web/src/lib/{rpc-errors.ts,status.ts,browse-url.ts,search.ts}`, `web/src/routes/{workbench,health}/+page.svelte`, `go.mod`

### Secondary (MEDIUM confidence)

- `tanstack.com/table/latest/docs/*` (via Context7) — general v9 React migration-guide concepts (feature registration pattern, `tableFeatures()`) cross-checked against the real Svelte source and found consistent for the framework-agnostic parts; the Svelte-specific installation/API pages were found STALE (Pitfall #3) and are explicitly NOT relied on for Svelte-specific claims

### Tertiary (LOW confidence)

None — every claim in this document is either tool-verified this session or explicitly flagged `[ASSUMED]` in the Assumptions Log above.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — every version/package confirmed via live registry query or executed program this session
- Architecture: HIGH — every pattern quoted verbatim from either this repository (file:line) or the real upstream source (fetched, not from training memory)
- Pitfalls: HIGH — all five are either confirmed by executing code (Pitfall #1's `python3` substring check, Pitfall #1's implication traced through the actual test file) or by reading the actual source/registry artifacts (Pitfalls #2-5)

**Research date:** 2026-08-29
**Valid until:** 7 days for the TanStack Table version-specific claims (Pitfall #3 shows the ecosystem is moving fast right around this exact date — re-verify `@tanstack/svelte-table`'s latest version before implementation if more than a few days elapse); 30 days for the Go-side claims (`doublestar/v4`, proto/Taskfile mechanics) which are far more stable.
