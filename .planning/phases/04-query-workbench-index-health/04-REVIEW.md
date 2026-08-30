---
phase: 04-query-workbench-index-health
reviewed: 2026-08-30T06:36:36Z
depth: deep
diff_base: ab95f70a
files_reviewed: 71
files_reviewed_list:
  - .github/workflows/components-drift.yml
  - .gitignore
  - Taskfile.yml
  - go.mod
  - go.sum
  - internal/query/files.go
  - internal/query/files_status_test.go
  - internal/uiproto/uiv1/ui.pb.go
  - internal/uiproto/uiv1/ui.proto
  - internal/uiproto/uiv1/uiv1connect/ui.connect.go
  - internal/uiserver/handlers.go
  - internal/uiserver/health_test.go
  - internal/uiserver/readonly_test.go
  - internal/upgrade/taskfile_shape_test.go
  - web/package.json
  - web/pnpm-lock.yaml
  - web/src/lib/browse-url.ts
  - web/src/lib/components/health/CountTable.svelte
  - web/src/lib/components/health/TrustVerdict.svelte
  - web/src/lib/components/health/WorktreeMismatchWarning.svelte
  - web/src/lib/components/ui/table/index.ts
  - web/src/lib/components/ui/table/table-body.svelte
  - web/src/lib/components/ui/table/table-caption.svelte
  - web/src/lib/components/ui/table/table-cell.svelte
  - web/src/lib/components/ui/table/table-footer.svelte
  - web/src/lib/components/ui/table/table-head.svelte
  - web/src/lib/components/ui/table/table-header.svelte
  - web/src/lib/components/ui/table/table-row.svelte
  - web/src/lib/components/ui/table/table.svelte
  - web/src/lib/components/ui/tabs/index.ts
  - web/src/lib/components/ui/tabs/tabs-content.svelte
  - web/src/lib/components/ui/tabs/tabs-list.svelte
  - web/src/lib/components/ui/tabs/tabs-trigger.svelte
  - web/src/lib/components/ui/tabs/tabs.svelte
  - web/src/lib/components/workbench/AnalysisPanel.svelte
  - web/src/lib/components/workbench/DataTable.svelte
  - web/src/lib/components/workbench/FilePicker.svelte
  - web/src/lib/components/workbench/affected-columns.ts
  - web/src/lib/components/workbench/callees-columns.ts
  - web/src/lib/components/workbench/callers-columns.ts
  - web/src/lib/components/workbench/impact-columns.ts
  - web/src/lib/components/workbench/table-features.ts
  - web/src/lib/debounced-rpc.ts
  - web/src/lib/file-search.ts
  - web/src/lib/gen/ui_pb.ts
  - web/src/lib/health-view.ts
  - web/src/lib/search.ts
  - web/src/lib/status.ts
  - web/src/lib/workbench-failure.ts
  - web/src/lib/workbench-url.ts
  - web/src/routes/+layout.svelte
  - web/src/routes/browse/+page.svelte
  - web/src/routes/health/+page.svelte
  - web/src/routes/workbench/+page.svelte
  - web/tests/browse-page.test.ts
  - web/tests/data-table-render-cost.test.ts
  - web/tests/debounced-rpc.test.ts
  - web/tests/degrade-states.test.ts
  - web/tests/file-search.test.ts
  - web/tests/health-page.test.ts
  - web/tests/health-view.test.ts
  - web/tests/setup.ts
  - web/tests/status.test.ts
  - web/tests/support/browse-page-state.svelte.ts
  - web/tests/support/data-table-location-host.svelte
  - web/tests/workbench-affected.test.ts
  - web/tests/workbench-callers-callees.test.ts
  - web/tests/workbench-failure.test.ts
  - web/tests/workbench-impact.test.ts
  - web/tests/workbench-tracer.test.ts
  - web/tests/workbench-url.test.ts
findings:
  critical: 2
  warning: 8
  info: 9
  total: 19
status: issues_found
---

# Phase 4: Code Review Report

**Reviewed:** 2026-08-30T06:36:36Z
**Depth:** deep (cross-file: import graph, call chains across the Go engine → wire → Svelte boundary)
**Diff base:** `ab95f70a..HEAD` (as instructed; the workflow's own scope derivation was NOT used)
**Files Reviewed:** 71 (33 `web/build/` artifacts excluded as generated build output)
**Status:** issues_found

## Summary

The locked-decision surface is clean. Every one of the four decisions the review brief
flagged as most likely to be violated is provably honoured, and I verified each with a
command rather than by reading prose:

| Decision | Check run | Result |
|---|---|---|
| D-04 (exactly one `classifyStatus`) | `rg -ln 'function classifyStatus' web/src/` | 1 file: `web/src/lib/status.ts` |
| D-06 (one `DataTable`, four `columns.ts`) | `git ls-files \| rg -c 'DataTable\.svelte'` → `1`; `git ls-files 'web/src/lib/components/workbench/*-columns.ts' \| wc -l` → `4` | compliant |
| D-07 (no client-side `MaxDepth`/`MaxLimit`) | `rg -n 'MaxLimit\|MaxDepth\|1000\|\bmax=\|Math\.min\|Math\.max'` over `workbench-url.ts`, `components/workbench/`, `routes/workbench/+page.svelte`, `file-search.ts` | only two prose comments; zero executable bounds |
| D-11 (`getAll` in workbench, `get` in browse) | `rg -c "getAll\('file'\)" workbench-url.ts` → `1`; `rg -c "get\('file'\)" browse-url.ts` → `1` | the intended asymmetry, not a defect |

`escapeGlobLiteral` also holds up under adversarial testing. I built a standalone Go
module against the real `github.com/bmatcuk/doublestar/v4 v4.10.0` matcher and ran the
JS escape set (`/[\\*?[\]{}]/g`) against every printable ASCII character as a single-char
term plus 16 adversarial multi-char terms (`**`, `[!a]`, `[[:alpha:]]`, `{a,b}`, `a\`,
`*/../*`, …). Every escaped term matched its literal path, none errored the server's
pre-scan sanity check, and none of `*`, `?`, `[a-z]`, `{a,b}` produced a wildcard false
positive. The escape set is complete for doublestar's dialect. The `GetHealth` handler,
`healthToProto` nil-handling for `*gitmeta.Mismatch`, the `withEngine`-not-degrade shape,
and the `IndexedCommitSHA`/`IsCommitSHA` gate are all correct and match `GetStatus`'s
own validated derivation. The vendored `table`/`tabs` source scanned clean (0 `{@html}`,
0 `innerHTML`-family, 0 network/fs/eval sinks, positive control `rg -o 'class'` → 50).

The defects are elsewhere, and two of them are user-facing correctness failures that the
plan-level convergence cycles could not have caught because they only exist in the code.

The most serious is a **reproduced, not theorised** bug: clearing the Workbench Symbol
field while an analysis request is in flight renders a red *"Something went wrong / This
operation was aborted"* banner instead of returning to the idle state. I wrote a
throwaway vitest reproduction against the real `+page.svelte`, ran it, captured the
rendered DOM, and deleted the file (`git status --porcelain` confirmed clean afterward).

The second is a trust-signal integrity failure: `/health` — the page whose entire purpose
per HLT-02 is to answer *"should I trust this index"* — renders a hardcoded, always-zero
"Pending changes" tally as if it were a live measurement, and no test in the repository
exercises that branch.

## Narrative Findings (AI reviewer)

## Critical Issues

### CR-01: Clearing an analysis input while a request is in flight renders a false "Something went wrong" error

**File:** `web/src/lib/components/workbench/AnalysisPanel.svelte:100-125`
**Also reachable via:** `web/src/routes/workbench/+page.svelte:76-79` (`handleSymbolInput`), `:287`, `:326`, `:338` (the `params.symbol ? make…Run(…) : undefined` ternaries)

**Issue:** The dispatch effect aborts the previous request *before* deciding whether the
new inputs are runnable, and the not-runnable path returns **without bumping the request
identity**:

```js
$effect(() => {
	requestKey;
	const currentRun = untrack(() => run);

	if (abortController) abortController.abort();      // line 104 — aborts request id N

	if (!currentRun) {
		panelState = { kind: 'idle' };
		return;                                        // line 108 — returns BEFORE ++requestId
	}

	panelState = { kind: 'loading' };
	const controller = new AbortController();
	abortController = controller;
	const id = ++requestId;                            // line 114 — never reached on the idle path
	…
		.catch((err: unknown) => {
			if (id !== requestId) return; // superseded — discard, not a real failure
			panelState = { kind: 'failed', failure: describeWorkbenchFailure(err, indexStatus) };
		});                                            // lines 121-124
```

When the developer clears the Symbol field, `run` becomes `undefined`, the in-flight
request is aborted, and `panelState` is correctly set to `idle` — but because `requestId`
was not incremented, the abort's own rejection passes the `id !== requestId` guard at
line 122 and immediately overwrites `idle` with a failure state. `connect-web` surfaces
an aborted request as `ConnectError` with `Code.Canceled`, which
`web/src/lib/rpc-errors.ts:56-57` falls through to `kind: 'unknown'`, which
`web/src/lib/workbench-failure.ts:60-66` maps to `kind: 'server-error'`, title
*"Something went wrong"*.

The `.then` path has the same hole, though it is harder to reach.

**Reproduction (executed this session, then deleted):** a throwaway
`web/tests/zzz-review-repro.test.ts` mounting the real `+page.svelte` at
`?mode=callers&symbol=Foo&limit=5` with a stub whose promise rejects with
`new ConnectError('This operation was aborted', Code.Canceled)` on abort, then firing
`input` with `value: ''` on `workbench-symbol-input`. Verbatim rendered output:

```
<div
  class="rounded-md border border-destructive/40 bg-destructive/10 p-3 text-sm text-destructive"
  data-testid="workbench-failure-server-error"
>
  <p class="font-semibold">Something went wrong</p>
  <p>This operation was aborted</p>
</div>
```

Expected: `Enter a symbol to run this analysis.` — `git status --porcelain` is clean; the
repro file was removed.

**Why no existing test catches it:** `rg -n "symbol.*''|clear|idle|Enter a symbol" web/tests/workbench-*.test.ts`
returns no matches. The `run → undefined` transition is entirely uncovered.

**Fix:** invalidate the request identity on every abort, not only on the dispatch path.

```js
$effect(() => {
	requestKey;
	const currentRun = untrack(() => run);

	if (abortController) {
		abortController.abort();
		abortController = undefined;
		requestId += 1; // invalidate the aborted request's own settlement
	}

	if (!currentRun) {
		panelState = { kind: 'idle' };
		return;
	}
	…
```

Alternatively, discard `Code.Canceled` explicitly in the `.catch`. The identity bump is
preferable — it matches what `web/src/lib/debounced-rpc.ts:89` already does for exactly
this case (`requestId += 1; // invalidate any in-flight response too`), and closes both
the `.then` and `.catch` holes at once.

---

### CR-02: `/health` presents a hardcoded, always-zero "Pending changes" tally as a live trust signal

**Files:**
- `internal/query/status.go:345-368` — the `StatusResult` composite literal
- `internal/query/files_status_test.go:471-476` — the test that locks it as inert
- `internal/uiserver/handlers.go:900-904` — `healthToProto` maps it onto the wire
- `internal/uiproto/uiv1/ui.proto` `GetHealthResponse.pending_changes = 12`
- `web/src/routes/health/+page.svelte:92-97` — renders it

**Issue:** `StatusResult.PendingChanges` is **never assigned anywhere in `internal/query`**.
`rg -n 'PendingChanges' internal/query/*.go` returns only the struct field declaration
(`status.go:61`), the type declaration (`status.go:70`), and tests — the `StatusResult{…}`
literal returned by `Status()` (`status.go:345-368`) has no `PendingChanges:` key at all,
so it is always the zero value. The engine's own doc comment says so
(`status.go:36`): *"the added/modified/removed COUNT breakdown remains an inert
placeholder"*, and `files_status_test.go:471` pins it — the subtest is literally named
`"PendingChanges stays an inert placeholder; WorktreeMismatch is live and genuinely nil here"`.

Phase 4 nonetheless put it on the wire and rendered it verbatim:

```svelte
{#if pageState.response.pendingChanges}
	<p data-testid="health-pending-changes">
		Pending changes: {pageState.response.pendingChanges.added} added, {pageState.response
			.pendingChanges.modified} modified, {pageState.response.pendingChanges.removed} removed
	</p>
{/if}
```

`healthToProto` always constructs `&uiv1.PendingChanges{}`, so the `{#if}` is always
truthy and the page **always** reads *"Pending changes: 0 added, 0 modified, 0 removed"*.
A developer with 50 uncommitted, unindexed files reads a confident, specific,
authoritative-looking zero on the one page whose stated job (HLT-02, 04-CONTEXT.md:9-11)
is *"tell at a glance whether the index they are reading is worth trusting."* This is
worse than omitting the number: it actively repudiates the `stale` flag rendered two
lines above it, which *is* live (`status.go:331`, `computeStale`).

**Coverage gap that hid it:** `rg -n 'pendingChanges|health-pending-changes' web/tests/`
returns exactly two hits, both `pendingChanges: undefined` (`health-page.test.ts:46`,
`health-view.test.ts:24`). The rendering branch is never entered by any test. The Go-side
assertion (`internal/uiserver/health_test.go:166-174`) compares the response against
`eng.Status()`'s own `PendingChanges` — a tautology that passes for `{0,0,0}` and would
pass for any other value equally.

**Fix (smallest correct change):** stop rendering a fabricated fact. Either drop the
block from `health/+page.svelte`, or gate it on a real signal and label it honestly:

```svelte
{#if pageState.response.stale}
	<p data-testid="health-pending-changes">
		The working tree has changed since this index was built. Run <code>codegraph index</code>.
	</p>
{/if}
```

and add a `// NOTE: inert placeholder — internal/query/status.go never populates this;
see files_status_test.go:471` comment on `ui.proto`'s `pending_changes` field so the next
consumer does not repeat the mistake. If the number is wanted for real, that is engine
work (`Sync`'s diff at `Status()` time), explicitly out of scope per `status.go:36`.

## Warnings

### WR-01: The Workbench tells the developer the index "is being rebuilt" when no index exists

**File:** `web/src/lib/workbench-failure.ts:40-48`

**Issue:** The `not-found` + `verdict === 'no-index'` branch returns
`title: 'Index is being rebuilt'`. Nothing is being rebuilt — there is no index at all.
The `detail` is correct (*"No index exists yet for this repository…"*), but the bolded
title is what a developer reads first, and it tells them to **wait** when the correct
action is `codegraph init`.

This directly contradicts the app's own copy for the identical state.
`web/src/lib/components/health/TrustVerdict.svelte:34-43` renders, for the same
`verdict === 'no-index'`: *"No index was found for this repository — there is nothing to
trust yet. Run `codegraph init` to create one."* Two surfaces in the same app, driven by
the same `classifyStatus` verdict, making contradictory statements — the exact repudiation
failure D-04 (04-CONTEXT.md:114-125) exists to prevent.

**Fix:** give the branch its own title and keep the taxonomy `kind` for testid grouping.

```js
case 'not-found':
	if (status.verdict === 'no-index') {
		return {
			kind: 'index-stale',
			title: 'No index for this repository',
			detail: 'No index exists yet, so the symbol could not be looked up. Run `codegraph init`.'
		};
	}
```

Note `workbench-failure.test.ts`'s "four titles pairwise distinct via Set size 4"
assertion still holds — this branch's title is currently a *duplicate* of the `indexing`
branch's, so distinctness is preserved only because both map to the same `kind`.

---

### WR-02: `AnalysisPanel`'s dispatch effect has no teardown — in-flight analyses are never cancelled on unmount

**File:** `web/src/lib/components/workbench/AnalysisPanel.svelte:100-125`

**Issue:** The effect never returns a cleanup function. The only `return` (line 108) is
the bare early return on the idle path, which Svelte treats as "no teardown". Consequently
switching Workbench tabs, or navigating away from `/workbench`, destroys the component
while its `Callers`/`Callees`/`Impact`/`Affected` RPC is still running. The server keeps
walking the graph to completion for a result nobody will read — and `Impact` at depth is
the most expensive call this UI can make.

Contrast `web/src/routes/health/+page.svelte:58` (`return () => controller.abort();`) and
`web/src/lib/components/workbench/FilePicker.svelte:51-53`
(`return () => controller.dispose();`), which both do this correctly.

**Fix:**

```js
	currentRun(controller.signal).then(…).catch(…);

	return () => {
		controller.abort();
		requestId += 1;
	};
```

The `requestId += 1` in the teardown also closes CR-01's `.catch` hole for the unmount
case.

---

### WR-03: The Symbol field fires one full graph-analysis RPC per keystroke, with no debounce

**File:** `web/src/routes/workbench/+page.svelte:76-79` and `:201-207`

**Issue:** `handleSymbolInput` is bound to `oninput`, and writes `symbol` into the URL on
every keystroke. That changes `impactKey`/`callersKey`/`calleesKey`
(`+page.svelte:186-194`), which re-triggers `AnalysisPanel`'s dispatch effect, which
issues a fresh `Callers`/`Callees`/`Impact` request. Typing `HandleRequest` issues 13
requests, each aborted by the next. Every one of them re-enters `withEngine`/`openEngine`
server-side and contends for the Pebble store lock — the exact amplification the `q`
exclusion in `status.ts`'s `VIEW_LOCAL_PARAMS` and the new `ROUTE_LOCAL_PARAMS` table
were added to prevent for `GetStatus`, reintroduced one layer up for the far more
expensive analysis RPCs.

The codebase already owns the correct mechanism and applies it to the two *cheaper*
search surfaces: `web/src/lib/debounced-rpc.ts` (`createDebouncedRpc`) with
`SEARCH_DEBOUNCE_MS = 150` (`web/src/lib/search.ts:47`), used by `search.ts` and
`file-search.ts`. The Workbench does not use it.

This also means CR-01 fires on a routine backspace-to-empty, not just an exotic sequence.

**Fix:** debounce the URL write for the free-text field only (leave `depth`/`limit`
immediate, since those are committed values):

```js
let symbolTimer: ReturnType<typeof setTimeout> | undefined;
function handleSymbolInput(e: Event): void {
	const value = (e.currentTarget as HTMLInputElement).value;
	clearTimeout(symbolTimer);
	symbolTimer = setTimeout(() => writeParams({ symbol: value || undefined }), SEARCH_DEBOUNCE_MS);
}
```

or move the write to `onchange`/Enter. Either keeps D-12's "full input state lives in the
URL" intact.

---

### WR-04: `DataTable` indexes the row model with an unguarded virtual index

**File:** `web/src/lib/components/workbench/DataTable.svelte:176-187`

**Issue:**

```svelte
{#each virtualRows as virtualRow (virtualRow.key)}
	{@const row = table.getRowModel().rows[virtualRow.index]}
	<Table.Row data-testid={`table-row-${row.id}`} …>
		{#each row.getAllCells() as cell (cell.id)}
```

`virtualRows` comes from `$rowVirtualizer.getVirtualItems()`, whose `count` is only
refreshed by the `$effect` at lines 114-124 — and in Svelte 5 user effects run *after*
the DOM render effects in the same flush. So on the first render after `rows` shrinks in
place, the template computes virtual items from the **stale** count while indexing into
the **new**, shorter row model. If the scroll offset is past the new end, `row` is
`undefined` and `row.id` throws a `TypeError` mid-render, blanking the component. There
is no bounds check and no `{#if row}` guard.

**Reachability today:** not reachable, and I checked rather than assumed. Every current
caller either remounts the table (`AnalysisPanel.svelte:141-155` transitions through
`{kind:'loading'}`, destroying `DataTable`, before rendering the new rows) or never
changes `rows` at all (`CountTable.svelte:46`, fed by a single `GetHealth`). So this is
**latent**, not live. It is still a Warning because `DataTable` is the deliberately
generic, single shared shell (D-06) that 04-01-SUMMARY.md explicitly hands to Phase 5,
and the first caller that swaps rows in place without an intervening loading state gets a
crash rather than a stale frame.

**Fix:**

```svelte
{#each virtualRows as virtualRow (virtualRow.key)}
	{@const row = table.getRowModel().rows[virtualRow.index]}
	{#if row}
		<Table.Row …>…</Table.Row>
	{/if}
{/each}
```

---

### WR-05: Virtualization sets `aria-rowcount` over the full model but no `aria-rowindex` on any row

**File:** `web/src/lib/components/workbench/DataTable.svelte:128` (`aria-rowcount`), rows at `:178-186`

**Issue:** `rg -n 'aria-rowindex|aria-rowcount' web/src/` returns exactly one code hit:
`aria-rowcount` at `DataTable.svelte:128`. `aria-rowindex` appears nowhere.

WAI-ARIA requires that when the rows present in the DOM are a subset of the total (which
is precisely what 04-07's virtualization made true), each rendered row carry
`aria-rowindex` so assistive technology can place it within the declared count. Without
it, a screen reader is told "1000 rows" and handed ~27, with no way to know which. The
two spacer `<tr aria-hidden="true">` elements (lines 172, 189) correctly remove themselves
from the accessibility tree, which makes the mismatch total rather than partial.

Before virtualization, `aria-rowcount` was redundant-but-consistent; after it, it is
actively wrong. This is a regression introduced by the 04-07 Task 4 change.

**Fix:** carry the model index onto each rendered row (and the header row):

```svelte
<Table.Row aria-rowindex={virtualRow.index + 2} …>
```

(`+2` because `aria-rowindex` is 1-based and row 1 is the header row, which should get
`aria-rowindex={1}` on `Table.Header`'s `<tr>`.)

---

### WR-06: `web:components:drift` never enables Corepack, so its own scheduled workflow can fail for an environment reason

**Files:** `Taskfile.yml:1275-1279` (preconditions), `Taskfile.yml` `web:components:drift` cmds (the `pnpm install` / `pnpm dlx` lines), `.github/workflows/components-drift.yml:85-91`

**Issue:** The target's precondition accepts *either* tool:

```yaml
- sh: command -v corepack || command -v pnpm
  msg: "neither corepack nor pnpm is on PATH — web:components:drift regenerates via `pnpm install` and `pnpm dlx shadcn-svelte` inside a scratch tree."
```

but its command body then invokes **bare `pnpm`** twice, and never runs `corepack enable`.
`corepack` merely being on `PATH` does not put a `pnpm` shim on `PATH`; `corepack enable`
is what does that. The only place in this repository that runs it is `web:deps`
(`Taskfile.yml:419-421`):

```sh
if command -v corepack >/dev/null 2>&1; then
  corepack enable >/dev/null 2>&1 || true
```

and `web:components:drift` neither declares `deps: [web:deps]` (unlike its sibling
`web:render-cost`, which does) nor duplicates that resolution. The workflow's only run
step is `run: task web:components:drift` after `actions/setup-node` — no `web:deps`, no
`corepack enable`.

So the precondition can pass on a host where the recipe cannot run. Because this is a
`schedule`-only, non-required job (correctly so, per D-16), a failure here is nobody's
merge blocker and can sit red indefinitely — which is exactly how a supply-chain guard
stops guarding. Note the guard's *logic* is sound: it prints `nfiles`/`ncomponents`
before comparing anything, carries structural floors (8 / 2), uses `set -euo pipefail`,
and accumulates `drift_found` so every differing file is reported. Rule `84d1gfpywd` is
satisfied. It is the bootstrap that is incomplete.

I have **not** verified whether `ubuntu-latest` happens to ship a global `pnpm` that
would mask this — treat that as unverified. The structural inconsistency between the
precondition and the body is verified and stands on its own.

**Fix:** add `deps: [web:deps]` to `web:components:drift` (it needs a hydrated
`node_modules` anyway, and `web:deps` already asserts the lockfile was not rewritten), or
add the same `corepack enable` resolution block to the top of its own recipe.

---

### WR-07: `FilePicker` never renders `searchState.failure` — a failed file search looks identical to "no matches"

**Files:** `web/src/lib/components/workbench/FilePicker.svelte:72-96`, `web/src/lib/file-search.ts:51-55, 102-104`

**Issue:** `FileSearchState` declares `failure: RpcFailure | undefined`
(`file-search.ts:53`) and `createFileSearchController`'s `onFailure` populates it
(`file-search.ts:102-104`). `rg -n 'failure' web/src/lib/components/workbench/FilePicker.svelte`
returns **nothing** (positive control: `rg -c 'searchState'` on the same file → `5`). The
field is written and never read.

The user-visible consequence: if the `Files` RPC fails — server unavailable, index being
rebuilt, a malformed pattern that slipped past `escapeGlobLiteral` — the results array
stays empty and `Command.Empty` renders the flat, reassuring string **"No results."**
The developer concludes their file does not exist and stops looking. This is the same
class of failure WRK-04 criterion 3 spends four distinct failure kinds preventing on the
analysis side; the file picker gets none of it.

For fairness: the pre-existing `SearchPanel.svelte` has the same gap for `liveFailure`
(`rg -n 'liveFailure' web/src/lib/components/browse/SearchPanel.svelte` → no match), so
this is a copied pattern rather than a novel one. It is still a new dead field in new
code.

**Fix:** render it, reusing the taxonomy the Workbench already has:

```svelte
{#if searchState.failure}
	<p role="status" data-testid="file-picker-failure" class="px-2 py-1 text-sm text-destructive">
		{describeWorkbenchFailure(searchState.failure, indexStatus).title}
	</p>
{/if}
```

or, minimally, surface `searchState.failure.message` so "the search broke" is
distinguishable from "nothing matched".

---

### WR-08: `debounced-rpc.ts`'s `dispose()` aborts without invalidating the request identity

**File:** `web/src/lib/debounced-rpc.ts:100-109` (compare `:81-92`)

**Issue:** The below-minimum path does the right thing — abort *and* `requestId += 1`
(line 89, commented `// invalidate any in-flight response too`). `dispose()` does only
half of it:

```js
function dispose(): void {
	if (debounceTimer !== undefined) { clearTimeout(debounceTimer); debounceTimer = undefined; }
	if (abort) { abort.abort(); abort = undefined; }
	// requestId is NOT bumped
}
```

So a caller that disposes while a request is in flight gets that request's abort
rejection delivered to `onFailure` (the `id !== requestId` guard at line 70 passes,
because nothing moved `requestId`), which writes into a store belonging to a component
that is already being torn down. `FilePicker.svelte:51-53` disposes on unmount, so this
fires on every tab switch away from Affected mid-search.

The visible impact today is nil only because of WR-07 — nobody renders the field it
poisons. That is two bugs cancelling, not a correct design. This is also the identical
root cause as CR-01, in the one module that was extracted specifically to be *the* place
this mechanism lives.

This behaviour is faithfully inherited from `search.ts`'s pre-extraction shape, so it is
not a regression — but the extraction was the moment to fix it, and `debounced-rpc.ts`'s
own header comment (lines 10-16) claims it owns "a monotonically increasing request id
whose stale responses are discarded in BOTH the resolve and the reject paths", which is
not true across `dispose()`.

**Fix:**

```js
function dispose(): void {
	if (debounceTimer !== undefined) { clearTimeout(debounceTimer); debounceTimer = undefined; }
	if (abort) { abort.abort(); abort = undefined; }
	requestId += 1; // invalidate any in-flight response, mirroring the below-minimum path
}
```

## Info

### IN-01: The four `*-columns.ts` files are byte-identical apart from their exported name

**Files:** `web/src/lib/components/workbench/{callers,callees,impact,affected}-columns.ts`

Verified: `diff <(sed -n '/^export const/,$p' affected-columns.ts) <(sed -n '/^export const/,$p' impact-columns.ts)`
reports exactly one differing line (the constant name), and the same for `callees`. The
`callers` file additionally exports `locationRowId`.

D-06 sanctions "four `ColumnDef[]` arrays", so this is not a decision violation. The
practical risk is drift: adding a fifth `Location` field means four edits, and three-of-four
is a silent inconsistency across analyses. Consider a single
`export const locationColumns` in `callers-columns.ts` that the other three re-export
under their own names — four files preserved, one definition.

### IN-02: `onSelect` is dead in production, and the row click handler is mouse-only

**Files:** `web/src/lib/components/workbench/DataTable.svelte:50, 56, 180-181`; `AnalysisPanel.svelte:65, 73, 154`

`rg -n 'onSelect' web/src/ web/tests/` shows the prop threaded through `AnalysisPanel`
into `DataTable`, but no `AnalysisPanel` instantiation in
`web/src/routes/workbench/+page.svelte` (lines 285, 310, 324, 336) ever supplies it. The
only consumer is `web/tests/support/data-table-location-host.svelte:40`. So none of the
four result tables is click-through today.

If it is wired later, note `onclick` on `Table.Row` with `cursor-pointer` and no
`tabindex`/`role="button"`/`onkeydown` is not keyboard-reachable.

### IN-03: `DataTable`'s doc comment claims a "selection" feature that is not registered

**Files:** `web/src/lib/components/workbench/DataTable.svelte:28-31`, `table-features.ts:24-31`

The comment states the row model "still carries every row … sorting, **selection** and
`aria-rowcount` all operate over the FULL set." `table-features.ts` registers only
`rowSortingFeature` + `createSortedRowModel()` + two `sortFns`. There is no
`rowSelectionFeature`, so there is no selection at all. Drop the word, or the next reader
will assume selection survives virtualization when there is nothing to survive.

### IN-04: `IndexHealth.pending_refs` is also an inert hardcoded zero on the wire

**Files:** `internal/query/status.go:366` (`PendingRefs: 0,`), `internal/uiproto/uiv1/ui.proto` `IndexHealth.pending_refs = 6`

Same shape as CR-02, but currently harmless because nothing renders it. Worth a
`// inert placeholder` note on the proto field so a future health-page iteration does not
surface it the way `pending_changes` was surfaced.

### IN-05: Virtualization silently narrows every DOM-based DataTable assertion

**File:** `web/tests/setup.ts:31-46`

The stub is correctly scoped — it keys on `this.dataset.testid === 'data-table-scroll'`
and returns jsdom's native `0` for every other element, so it cannot mask a layout bug
elsewhere, and `configurable: true` leaves it overridable per test. Good.

The consequence worth recording: with `STUBBED_HEIGHT = 600`, `ROW_HEIGHT_PX = 37` and
`OVERSCAN = 10`, roughly 27 rows land in the DOM. Every `getAllByRole('row')` /
`getByText` assertion across `workbench-*.test.ts` and `health-page.test.ts` now covers
only that window. All current fixtures are far smaller than 27 rows, so nothing is
currently untested — but a future fixture that crosses the window will lose coverage
without any test turning red. `data-table-render-cost.test.ts:109-124` is the only test
that reasons about the model-vs-DOM distinction.

### IN-06: The glob sanity check probes against a fixed dummy name

**File:** `internal/query/files.go:153-157`

```go
if _, err := doublestar.Match(opts.Pattern, "sanity-check"); err != nil {
```

`Match` reports `ErrBadPattern` only for malformed constructs it actually walks into
while matching against *this specific name*, so a malformed pattern whose bad part sits
behind a segment that fails to match `"sanity-check"` early is not caught here. It is
caught on the first indexed entry that reaches it (lines 177-180), so no incorrect result
escapes — but the "refusal precedes scan" property that
`files_status_test.go:830-839` asserts holds only for patterns whose defect is reachable
from a slash-less 13-character name. This shape predates Phase 4 (the matcher swap kept
it verbatim); noting it so the property is not over-trusted.

### IN-07: `web:components:drift` cannot see a file the CLI emits that was never committed

**File:** `Taskfile.yml` `web:components:drift`, the `while IFS= read -r f … done <<< "${files}"` loop

The loop iterates `git ls-files` output and checks committed → regenerated. A file the
pinned CLI now produces that is absent from the repository (a component gaining a new
sub-file in a future registry revision) produces no `fresh`-missing error and no `cmp`
mismatch — it is simply never visited. Adding a reverse pass (`find "${scratch}/web/src/lib/components/ui" -type f`
compared against the committed set) would close it. Low priority: the target is pinned to
`@1.5.1`, so the registry cannot introduce new files without a version bump.

### IN-08 (verified clean, no defect): `escapeGlobLiteral` is complete for doublestar's dialect

**File:** `web/src/lib/file-search.ts:68-72`

Recorded as a positive result because the brief asked for it specifically. I built a
standalone Go module requiring `github.com/bmatcuk/doublestar/v4 v4.10.0` and ported the
JS regex `/[\\*?[\]{}]/g` to `regexp.MustCompile(`[\\*?\[\]{}]`)`, then asserted, for every
printable ASCII codepoint `0x20..0x7e` as a single-character term, that
`**/*<escaped>*` (a) does not error the `doublestar.Match(pat, "sanity-check")` gate and
(b) matches `pkg/a<term>b.go`. All 95 passed. A second table asserted no false positives
for `*`, `?`, `[a-z]`, `{a,b}`, and a third exercised 16 adversarial multi-char terms
(`**`, `\`, `\\`, `[]`, `{}`, `[!a]`, `[^a]`, `a\`, `**/**`, `[[:alpha:]]`, `*/../*`, …)
for both no-error and literal-match. All passed. Backtracking cost for a 30×`*a` term was
583ns escaped. Nothing bypasses the escape: `file-search.ts:83` is the single pattern
construction site and it wraps `escapeGlobLiteral(term)` unconditionally.

### IN-09: `<input type="number">` makes the shape-invalid states largely unreachable and silently drops the parameter

**File:** `web/src/routes/workbench/+page.svelte:97-125`, `:216-222`, `:232-238`, `:249-256`

For `type="number"`, a browser reports `value === ''` for any input it cannot parse as a
number (`"1e"`, `"-"`, `"1.2.3"`). That hits the `raw === ''` branch, which calls
`writeParams({ depth: undefined })` and **removes the parameter from the URL** rather than
setting `depthInvalid`. So `aria-invalid` is effectively only reachable for shaped-but-
non-integer values like `"1.5"`, and a mid-typing state can silently discard a previously
committed depth. Either use `type="text"` with `inputmode="numeric"` (so `isShapeInteger`
sees what was typed and `depthInvalid` does its job), or distinguish "cleared" from
"unparseable" before writing.

---

## Verified Clean (recorded so the next reviewer does not re-derive it)

- **Locked decisions D-04 / D-06 / D-07 / D-11** — all compliant, commands and outputs in
  the Summary table above.
- **`GetHealth` handler** (`internal/uiserver/handlers.go:955-981`) — uses the ordinary
  `withEngine` shape, not `GetStatus`'s degrade-and-answer path; calls `eng.Status(ctx)`
  exactly once; `schema.IndexedCommitSHA` returns `("", false)` for an absent SHA
  (`internal/schema/meta.go:41-50`), so the `ok && !IsCommitSHA` guard is complete and
  byte-identical to `GetStatus`'s at `handlers.go:337-338`.
- **`worktreeMismatchToProto`** (`handlers.go:920-929`) — nil in, nil out; the common
  clean-tree path is one obvious branch.
- **Vendored supply chain** (`web/src/lib/components/ui/{table,tabs}`, 14 files) —
  `rg -o '\{@html'` → 0; `rg -o 'innerHTML|outerHTML|insertAdjacentHTML|createContextualFragment|dangerously'` → 0;
  `rg -no 'fetch\(|XMLHttpRequest|eval\(|new Function|require\(|import\(|node:fs|readFile|writeFile|localStorage|document\.cookie'` → 0;
  positive control `rg -o 'class'` → 50.
- **Generated protobuf** — `GetHealth` present in all three artifacts
  (`ui.pb.go` 57 hits, `ui.connect.go` 20, `ui_pb.ts` 19); all four new message schemas
  present in the TS client; `readonly_test.go`'s `11`-method literal and the +28
  field-number fixture both updated consistently.
- **`web/build/` freshness** — `git log --oneline -3 -- web/src/` and
  `-- web/build/` both name `3b7de15d` as the most recent commit, so the committed bundle
  was rebuilt in the same commit as the last source change.
- **`web/tests/setup.ts` stub scoping** — cannot leak to unrelated components (IN-05
  records the one real consequence).
- **`go vet ./internal/query/... ./internal/uiserver/...`** — clean.
  **`go test ./internal/uiserver/...`** — `ok`.
- **`git status --porcelain`** — clean at the end of this review; the one throwaway
  reproduction file written for CR-01 was removed.

---

_Reviewed: 2026-08-30T06:36:36Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
_Diff base: ab95f70a_
