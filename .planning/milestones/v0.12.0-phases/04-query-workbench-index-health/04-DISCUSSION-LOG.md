# Phase 4: Query Workbench & Index Health - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-29
**Phase:** 04-query-workbench-index-health
**Areas discussed:** Health data on the wire, Sortable table implementation, Workbench routing & URL model, Multi-file selection for WRK-02

---

## Todo Folding (cross-reference step)

7 pending todos were matched against this phase; 2 scored 0.9, 1 scored 0.4.

| Option | Description | Selected |
|--------|-------------|----------|
| Files glob bug (query, 0.9) | `filepath.Match`'s `*` never crosses `/`; live file search returns zero matches for nested files | |
| Render-cost measurement (ui, 0.9) | No measurement for large result sets; no virtualization anywhere | |
| Both | Glob bug blocks the picker; render cost gets worse with 1000-row tables | ✓ |
| Neither — keep Phase 4 tight | Scope strictly to WRK-01..04 and HLT-01..03 | |

**User's choice:** Both.
**Notes:** A third todo (shadcn-svelte registry-version pinning, supply-chain) was folded in later, mid-discussion, after the D-22 challenge — see the Supply Chain section below.

---

## Health data on the wire

📊 Research presented: health-endpoint design consensus (split cheap frequently-polled signal from expensive diagnostics; memoize with short TTL where deep checks must stay hot).

### Q1 — Where should the rich health data live on the wire?

| Option | Description | Selected |
|--------|-------------|----------|
| New `GetIndexHealth` RPC | Keep `GetStatus` as 9 cheap scalars; one additive read-only rpc for the health view | ✓ |
| Extend `GetStatusResponse` additively | One rpc, one round trip; every navigation serializes per-language maps | |
| Extend `GetStatus` + server-side memoization | One message, cache the expensive parts behind a TTL | |

**User's choice:** New `GetIndexHealth` RPC (recommended).
**Notes:** Driven by the finding that Phase 3's `createStatusGate` fetches `GetStatus` on every navigation.

### Q2 — When should HLT-03's worktree check run?

| Option | Description | Selected |
|--------|-------------|----------|
| Only in the health RPC | Config-level fact; per-navigation freshness buys nothing | ✓ |
| Once at server start, cached | Cheapest; misses a mid-session worktree change | |
| On every `GetStatus`, uncached | Always current; up to 4 git subprocess spawns per click | |

**User's choice:** Only in the health RPC (recommended).

### Q3 — What does HLT-01's "coverage" mean?

| Option | Description | Selected |
|--------|-------------|----------|
| Per-language file counts only | `FilesByLanguage` + `Languages`, already on the engine | ✓ |
| Capability matrix (full/partial/none) | `internal/indexer/capability/matrix.go` — what "coverage" already means in this codebase | |
| Indexed-vs-on-disk ratio | Most literal reading; needs a filesystem walk that does not exist | |
| Both file counts and capability matrix | Most complete; largest health surface | |

**User's choice:** Per-language file counts only (recommended).
**Notes:** The capability matrix was surfaced during option-grounding, not assumed away — it is recorded as a deferred idea rather than silently dropped.

### Q4 — Where does HLT-02's trust verdict come from?

| Option | Description | Selected |
|--------|-------------|----------|
| Reuse Phase 3's `classifyStatus` | One verdict function app-wide; banner and health page cannot disagree | ✓ |
| Health-specific verdict, richer inputs | Weighs `ReindexRecommended`, `PendingChanges`, worktree mismatch | |
| Reuse `classifyStatus`, extend it additively | One function, wider inputs; touches shipped Phase 3 code | |

**User's choice:** Reuse `classifyStatus` (recommended).

---

## Sortable table implementation

📊 Research presented (Context7, `/huntabyte/shadcn-svelte`): `table` is presentational wrappers with no TanStack dependency; `data-table` is a *guide* that requires `@tanstack/svelte-table` (TanStack v9); v9 tree-shakes unregistered features via `tableFeatures({...})`.

### Q1 — How should WRK-04's sortable tables be built?

| Option | Description | Selected |
|--------|-------------|----------|
| Vendor `table` only, hand-roll sort | ~6 files, zero new npm packages, ~30 lines of sort | |
| Vendor `table` + `@tanstack/svelte-table` | Full sorting/filtering/pagination/selection; new runtime dep tree | ✓ |
| Hand-roll everything, no vendoring | Plain `<table>` + Tailwind; diverges from the component system | |

**User's choice (free text):** *"what is idiomatic here, and correct? Feels like 2. adding deps is not a decision maker, doing the right thing is"*

**Notes — this exchange changed the decision and the framing:**
The orchestrator had presented dependency count as a decision driver. The user rejected that premise. On re-examination the orchestrator agreed and answered directly: TanStack is the idiomatic path because shadcn-svelte deliberately ships no sorting abstraction and its own docs punt to TanStack.

One correctness objection was checked rather than assumed: `npm view @tanstack/svelte-table dist-tags` → `latest: 9.2.4`, `beta: 9.0.0-beta.80`, `time.modified` 2026-08-28. v9 is **stable**, so the "pre-release gamble" objection did not hold.

The orchestrator raised one push-back on Q2's answer (see below) and asked a plain-text follow-up. The user replied: *"do the right and idiomatic thing. add deps where they are appropriate, prefer them over novel or half baked solutions."* That ruling resolved both open questions and is recorded verbatim in CONTEXT.md `<specifics>`.

### Q2 — One table component or four?

| Option | Description | Selected |
|--------|-------------|----------|
| One generic `ResultTable` | All four analyses return the identical `Location` shape | |
| Four per-analysis tables | Maximum freedom to diverge columns later | ✓ |
| Generic core + per-analysis wrappers | Middle ground | |

**User's choice:** Four per-analysis tables.
**Notes:** The orchestrator pushed back once — the row shape is byte-identical and the codebase has a strong "ONE X" grain (D-13, D-04, D-19) — and asked whether "four tables" meant four `ColumnDef` arrays over a shared shell (the idiomatic shadcn structure) or four fully independent components. The user's "do the right and idiomatic thing" ruling resolved this to the former, which is what CONTEXT.md D-06 records.

### Q3 — Where does sorting happen?

| Option | Description | Selected |
|--------|-------------|----------|
| Client-side over returned rows | No proto change; server bounds unduplicated | ✓ |
| Client-side plus truncation notice | Same, plus an explicit "showing first N" notice | |
| Add a sort parameter to the RPCs | Server-side sort over the full set before limit | |

**User's choice:** Client-side over the returned rows (recommended).

### Q4 — What about render cost at up to 1000 rows?

| Option | Description | Selected |
|--------|-------------|----------|
| Measure only, no virtualization | Honors the todo as written; decide later with data | |
| Measure and virtualize if over a threshold | Solves it now if real | ✓ |
| Skip measurement; cap rendered rows | Cheapest; adds a second client-side bound | |

**User's choice:** Measure and virtualize if over a threshold.
**Notes:** This answer is what made TanStack decisive — hand-rolled virtualization is materially harder than hand-rolled sorting, so a hand-rolled sort would have been rewritten onto TanStack the moment the threshold tripped.

---

## Supply Chain (raised by maintainer challenge, mid-discussion)

**Maintainer challenge, verbatim:** *"WTF is D-22 and why is there a new gate just to add a vendored dep?"*

**Outcome — the orchestrator was wrong twice and corrected both:**

1. **D-22 is not a gate.** `03-CONTEXT.md:357` records a *gap* — vendored `.svelte` source never enters `pnpm-lock.yaml`, so BLD-06's audit gate structurally cannot see it — with the mitigation "scope discipline, **not new machinery**". The `gate="blocking-human"` checkpoint was 03-06's own plan-level choice, not a standing rule, and does not bind Phase 4.

2. **The risk was stated backwards.** npm dependencies **do** enter the lockfile and **are** covered by `web:audit` / `web:deps:strict` / `web:lockfile`. Vendored `.svelte` source is the *unscanned* surface. So adding TanStack is the low-ceremony, well-covered path — the opposite of what the orchestrator's phrasing implied.

Both corrections are recorded in CONTEXT.md D-09 with the artifact-vs-gate-coverage table.

### Q — How far should Phase 4 take the folded registry-pinning todo?

| Option | Description | Selected |
|--------|-------------|----------|
| `web:components:drift`, local-only | Byte-diff against the pinned CLI; local + release path, not PR CI | ✓ |
| `web:components:drift`, wired into CI | Strongest guarantee; makes an external registry a merge dependency | |
| Pin the version only, no drift target | Cheap; proves nothing about the committed bytes | |

**User's choice:** `web:components:drift`, local-only (recommended).
**Notes:** Folded in after the user answered "go ahead" to the orchestrator's offer to close the gap structurally rather than repeat a per-phase human read. Declined as scope growth by Phase 2 and again by Phase 3; deliberately reversed here.

---

## Workbench routing & URL model

No web research: routing was pre-answered by D-18 (the `/workbench` and `/health` routes already exist as Phase 2 placeholder slots), and the repo's own `browse-url.ts` conventions are authoritative for the rest.

### Q1 — Own URL grammar module or extend `browse-url.ts`?

| Option | Description | Selected |
|--------|-------------|----------|
| Sibling `workbench-url.ts` | Imports browse-url's exported primitives; leaves Browse's frozen grammar untouched | ✓ |
| Extend `browse-url.ts` | One module for both views | |
| Shared core + two thin grammars | Cleanest long-term; refactors shipped Phase 3 code | |

**User's choice:** Sibling `workbench-url.ts` (recommended).

### Q2 — How are multiple files encoded?

| Option | Description | Selected |
|--------|-------------|----------|
| Repeated `file=` keys, multi-valued | What `URLSearchParams` models natively | ✓ |
| Comma-joined `files=a,b` | Breaks on commas in paths; needs a bespoke escape rule | |
| Ride the unknown-pairs multimap | Zero grammar change; no schema for the load-bearing params | |

**User's choice:** Repeated `file=` keys (recommended).
**Notes:** Requires `getAll()` in the workbench grammar — a deliberate divergence from browse-url's `params.get()`, flagged in CONTEXT.md D-11 so the planner does not copy the wrong reader.

### Q3 — How much Workbench input state belongs in the URL?

| Option | Description | Selected |
|--------|-------------|----------|
| Full state — mode, inputs, depth, limit | A link reconstructs the exact query the sender ran | ✓ |
| Analysis + primary input only | Shorter URLs; a shared link can reproduce a different result | |
| Nothing — ephemeral, deep-link out only | Simplest; reads the NAV-01 dependency as one-directional | |

**User's choice:** Full state (recommended).

### Q4 — How does the user pick among the four analyses?

| Option | Description | Selected |
|--------|-------------|----------|
| Tabs, one per analysis | Maps onto the four-`columns.ts` decision; `mode` URL param | ✓ |
| Single form with an analysis selector | Least chrome; form must reshape on selection | |
| All four stacked on one page | Comparable side by side; long page, four concurrent RPCs | |

**User's choice:** Tabs (recommended).

---

## Multi-file selection for WRK-02

Grounding performed before the questions: `Engine.Files` has **three** callers — CLI (`internal/cli/files.go:46`), MCP (`internal/mcp/tools.go:512`), UI (`internal/uiserver/handlers.go:433`) — so the glob bug is not UI-local. `doublestar` compatibility was verified via its own docs rather than assumed.

### Q1 — How should the Files glob bug be fixed?

| Option | Description | Selected |
|--------|-------------|----------|
| Adopt `bmatcuk/doublestar/v4` | Drop-in for `path.Match`; pure Go, zero deps; no golden or transcript churn | ✓ |
| Add a new orthogonal `Substring` option | Strictly additive, no dep; two overlapping narrowing mechanisms | |
| Fix client-side pattern construction only | No Go change; leaves the bug live for CLI and MCP | |

**User's choice:** Adopt `bmatcuk/doublestar/v4` (recommended).
**Notes:** Chosen partly because T-03-14 forbids re-baselining frozen MCP transcripts, and no existing caller uses `**`, so behavior for every current pattern is unchanged.

### Q2 — How does a developer select multiple files?

| Option | Description | Selected |
|--------|-------------|----------|
| Search-and-add with chips | Reuses 03-06's Command primitive and `search.ts`'s controller pattern | ✓ |
| File tree with checkboxes | `FileTreeNode` already on the wire; new tree + cascade semantics | |
| Paste-a-list textarea | Fast with paths in hand; no discovery affordance | |

**User's choice:** Search-and-add with chips (recommended).

---

## Claude's Discretion

- WRK-04 criterion 3's error taxonomy — composing `classifyRpcError` and the status gate into "not found / index stale / server error".
- The health page's concrete layout, including what "above the raw numbers" means for HLT-02 and how HLT-03's warning is made unmissable.
- Whether `GetIndexHealth` defines its own message or reuses `GetStatusResponse` fragments.
- The render-cost threshold value in D-08 and how it is measured.
- Empty, loading and error states per analysis.
- Chip-affordance design in D-15.

## Deferred Ideas

- Capability-matrix coverage in the health view (`internal/indexer/capability/matrix.go`).
- Indexed-vs-on-disk coverage ratio (needs a filesystem walk; no engine support today).
- File tree with checkboxes for multi-file selection — Phase 5 is chartered for structural views.
- Server-side sorting via a sort parameter on the four analysis RPCs.
- Wiring `web:components:drift` into required PR CI.

## Final Gate

Offered four further gray areas (error taxonomy, HLT-02 layout contract, `GetIndexHealth` message shape, render-cost threshold value). User selected **"I'm ready for context"** — all four are recorded under Claude's Discretion in CONTEXT.md.
