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
status: clean
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

---

## Re-review (post-fix)

**Re-reviewed:** 2026-08-30T07:12:03Z
**Depth:** deep
**Fix-pass range:** `d8f4289f..HEAD` (10 fix commits + 2 bundle rebuilds + 1 dispositions doc)
**Verdict:** 10/10 findings have a fix present. **8 are correct. 1 (WR-05) is present but
introduces a new WAI-ARIA MUST violation. 1 (WR-07) is present but leaves the exact
misleading string it was raised against still on screen.** Three new WARNINGs, four new
INFOs. `status:` stays `issues_found`.

**New findings:** 0 Critical, 3 Warning, 4 Info.

### Gates run this session (commands + verbatim results)

| Gate | Command | Result |
|---|---|---|
| web unit tests | `pnpm vitest run` (cwd `web/`) | `Test Files 28 passed (28)` / `Tests 268 passed (268)` |
| type check | `pnpm check` | `1020 FILES 0 ERRORS 0 WARNINGS 0 FILES_WITH_PROBLEMS` |
| bundle drift | `task web:drift` | `PASS — hashed 103 source files, manifested 31 output files` |
| proto drift | `task proto:drift` | `all 4 generated files byte-identical to the pinned toolchain's regeneration` |
| component drift | `task web:components:drift` | `PASS — all 50 vendored component files across 8 components byte-identical` |
| Go | `GOTOOLCHAIN=go1.26.5 go test -count=1 -run TestGetHealth ./internal/uiserver/` | 6/6 PASS |

`go test` under the ambient toolchain (`go version go1.27.0 darwin/arm64`) fails to build
`github.com/cockroachdb/swiss` (`undefined: fastrand64`, `undefined: getRuntimeHasher`).
That is a pre-existing toolchain-vs-dependency issue unrelated to this fix pass —
`GOTOOLCHAIN=go1.26.5` (go.mod's `go 1.26.5`) builds and passes.

### Locked decisions — verified not violated

`git diff d8f4289f..HEAD --stat -- internal/uiserver/readonly_test.go go.mod go.sum
web/package.json web/pnpm-lock.yaml .github/` produces **no output** — `mutatingVerbs` is
byte-unchanged, and no npm or Go dependency was added. Drift floors are unchanged:
`rg -n "nfiles.*-lt|ncomponents.*-lt" Taskfile.yml` → `347: -lt 4` (proto), `1323: -lt 8`
and `1327: -lt 2` (components) — **not** raised to today's 50/8. `GetHealthResponse`'s
field numbers are untouched (`pending_changes = 12` still present; the diff on `ui.proto`
is comment-only). No test was deleted: the same diff filtered for removed `func Test` /
`it(` / `describe(` / `t.Run(` lines returns nothing — `health_test.go`'s tautological
assertion was **restated**, not removed.

### Per-finding verdicts

#### CR-01 — CONFIRMED FIXED

`AnalysisPanel.svelte:137-140` returns `() => { controller.abort(); requestId += 1; }`.
Svelte runs the previous run's cleanup before the re-run, so on `run → undefined` the
sequence is `abort()` → `requestId = N+1` → body sets `panelState = {kind:'idle'}` and
returns no cleanup; the abort's `Code.Canceled` rejection then fails the `id !== requestId`
guard at `:133` and is discarded. Locked by
`workbench-callers-callees.test.ts:298-325`, which drives the real route.

#### WR-02 — CONFIRMED FIXED, and NOT over-cancelling

Verified empirically, not by reading. A throwaway probe (`web/tests/zzz-rereview-probe2.test.ts`,
written, run, deleted — `git status --porcelain` clean afterwards) mounted the real
`+page.svelte` with a callers stub that records its `AbortSignal`:

```
PROBE aborted before tab switch = false
PROBE aborted after tab switch  = true
PROBE failure banner present    = false
PROBE aborted after unmount     = true
PROBE callers dispatches after unrelated write = 1
PROBE first signal aborted (after unrelated write) = false
```

The last two lines are the over-cancellation check the brief asked for: writing an
unrelated URL param (`depth`, which `callersKey` does not include) neither re-dispatches
nor aborts. The `untrack(() => run)` + primitive-keyed `$derived` combination still holds
after the change.

#### CR-02 — CONFIRMED FIXED, and the replacement assertion is genuinely falsifiable

I did not take the claim. I mutated `handlers.go:906` from
`Added: int32(result.PendingChanges.Added)` to `Added: int32(result.FileCount)` and ran
the test. Verbatim:

```
--- FAIL: TestGetHealthProjectsStatusResult (0.16s)
    health_test.go:190: pending_changes = added:4, want the documented all-zero placeholder {0,0,0}
FAIL
```

`handlers.go` was restored (`git status --porcelain` clean). The old assertion could not
have failed here; the new one does. The `pc == nil` fatal at `health_test.go:186` also
still covers a dropped mapping. `status.go:34` is verified to be the `pendingChanges`
mapping-table row the comment cites, and `files_status_test.go:471` is verified to be the
named subtest.

The hand-written NOTE comments added to the **generated** `internal/uiproto/uiv1/ui.pb.go`
and `web/src/lib/gen/ui_pb.ts` are not a hand-edit defect: `task proto:drift` (whose file
set includes `web/src/lib/gen/*.ts`, `Taskfile.yml:341`) reports all 4 files byte-identical
to regeneration from `ui.proto`.

`/health` no longer renders the tally: `rg -n "pendingChanges|pending-changes" web/src/`
returns only the generated type at `ui_pb.ts:1560`.

#### WR-01 — CONFIRMED FIXED

`workbench-failure.ts:54` now reads `title: 'No index for this repository'`. See RR-I-02
for the stale doc comment this left behind.

#### WR-03 — CONFIRMED FIXED

`+page.svelte:88-96`: the timer, the `clearTimeout` on each keystroke, and — importantly —
the `$effect` cleanup that clears a pending timer on destroy, so a `replaceState` can never
fire after the route component is gone. Locked by
`workbench-callers-callees.test.ts:327-349` (13 keystrokes → one call, asserted as
`expect(calls).toEqual([{ symbol: 'HandleRequest', limit: 5 }])`).

#### WR-04 — CONFIRMED FIXED, and it does not mask data

The brief asked whether `{#if row}` silently skips real rows. It does not. Probe
(`web/tests/zzz-rereview-probe.test.ts`, written, run, deleted): mount 60 rows, scroll to
`60*37-600`, then `rerender` in place with 5 rows:

```
PROBE after shrink to 5, dom data rows = 5 [
  'table-row-pkg/file0.go:1:symbol-0', … 'table-row-pkg/file4.go:5:symbol-4'
]
```

All five survive. The guard covers exactly the one transient frame in which the template
reads virtual items computed from the pre-`setOptions` count — the `$effect` at
`DataTable.svelte:114-124` then corrects it. See RR-W-03 for the regression test that
would not notice if this stopped being true.

#### WR-05 — PRESENT BUT INCORRECT → see RR-W-01

#### WR-06 — CONFIRMED FIXED

`Taskfile.yml:1286-1299`. `task web:components:drift` executed end-to-end this session and
printed `PASS — all 50 vendored component files across 8 components byte-identical to
shadcn-svelte@1.5.1's regeneration`. Floors unchanged. See RR-I-03 for what the copied
block dropped.

#### WR-07 — PRESENT, REASONING CORRECT, OUTCOME PARTIAL → see RR-W-02

The fixer's rejection of the review's own suggested `describeWorkbenchFailure(searchState.failure, …)`
is **right**, and I verified the mechanism rather than the prose: `searchState.failure` is
an `RpcFailure` plain object (`file-search.ts:103` → `classifyRpcError(err)`), and
`rpc-errors.ts:30-35` would take the `!(err instanceof ConnectError)` arm, hit
`err instanceof Error ? err.message : String(err)`, and produce the literal string
`"[object Object]"` with `kind: 'unknown'` for every input. Deviating from the suggestion
was correct.

#### WR-08 — CONFIRMED FIXED

`debounced-rpc.ts:119`. No dispose-then-late-resolve path survives: `dispose()` aborts and
bumps in the same synchronous block, and the rejection can only be delivered on a later
microtask. The double-bump interaction the brief flagged is benign — `setQuery` below
minimum sets `abort = undefined` first, so a subsequent `dispose()` only bumps.
`debounced-rpc.test.ts:244-276` locks it.

---

## Narrative Findings (AI reviewer) — re-review

### RR-W-01 (WARNING): WR-05's `aria-rowindex` now exceeds `aria-rowcount` — a WAI-ARIA MUST violation the fix itself created

**File:** `web/src/lib/components/workbench/DataTable.svelte:128` (`aria-rowcount`), `:131` (header `aria-rowindex={1}`), `:190` (`aria-rowindex={virtualRow.index + 2}`)

**Issue:** The fix assigns the header row `aria-rowindex={1}` and each data row
`virtualRow.index + 2`, but left `aria-rowcount` at
`{table.getRowModel().rows.length}` — the **data** row count, which does not include the
header row the fix just numbered as row 1. So for N data rows the maximum
`aria-rowindex` is `N + 1` while `aria-rowcount` is `N`.

WAI-ARIA 1.2, `aria-rowindex`, verbatim from `https://www.w3.org/TR/wai-aria-1.2/`
(fetched this session):

> Authors **MUST** set the value for `aria-rowindex` to an integer greater than or equal to
> 1, greater than the `aria-rowindex` value of any previous rows, and **less than or equal
> to the number of rows in the full table**.

**Reproduced, not theorised.** Probe (`web/tests/zzz-rereview-probe.test.ts`, written, run,
deleted): 50 rows, scrolled to `50*37-600` so the window includes the last row:

```
PROBE aria-rowcount    = 50
PROBE max aria-rowindex = 51
```

A screen reader on the last row announces "row 51 of 50". Before the fix `aria-rowcount`
was merely unaccompanied; now it is contradicted by a sibling attribute on the same table.
`data-table-virtualization.test.ts:31-41` asserts only header=1 and first-data-row=2, so it
locks the off-by-one in rather than catching it.

**Fix:**

```svelte
<Table.Root aria-rowcount={table.getRowModel().rows.length + 1}>
```

and extend `data-table-virtualization.test.ts` with the invariant that actually matters:

```ts
const idx = within(table).getAllByRole('row').map((r) => Number(r.getAttribute('aria-rowindex')));
expect(Math.max(...idx)).toBeLessThanOrEqual(Number(table.getAttribute('aria-rowcount')));
```

---

### RR-W-02 (WARNING): WR-07 surfaces the failure but leaves the misleading "No results." rendering directly beneath it

**File:** `web/src/lib/components/workbench/FilePicker.svelte:99-108`

**Issue:** WR-07's stated harm was the string *"No results."*: *"the developer concludes
their file does not exist and stops looking"* (04-REVIEW.md, WR-07). The fix added a
failure paragraph at `:99-105` but did not gate `Command.Empty`'s literal at `:108`, which
still renders whenever `searchState.results.length === 0` — which is exactly the failed
case.

**Reproduced.** Probe (`web/tests/zzz-rereview-probe3.test.ts`, written, run, deleted),
mounting the real `FilePicker` with `files: () => Promise.reject(new Error('server unavailable'))`:

```
PROBE failure text          = "server unavailable"
PROBE "No results." present = true
PROBE full text             = "server unavailable No results."
```

The user is now told two contradictory things at once: the search broke, **and** there are
no results. The existing regression test
(`workbench-affected.test.ts:178-190`) asserts only `toHaveTextContent('server unavailable')`
on the new element — it passes with the contradiction still on screen. This is the
"satisfies its own literal check while the user-facing property stays broken" shape.

Confirmed non-issue in the same probe: the message is correctly cleared on the
below-minimum path (`PROBE after fail = true`, `PROBE after backspace-to-1char = false`).

**Fix:** make the two mutually exclusive.

```svelte
<Command.List>
	{#if !searchState.failure}
		<Command.Empty>No results.</Command.Empty>
	{/if}
	…
```

and add `expect(root.textContent).not.toContain('No results.')` to
`workbench-affected.test.ts`'s WR-07 case.

---

### RR-W-03 (WARNING): WR-04's regression guard cannot fail — its only row assertion passes at zero rows

**File:** `web/tests/data-table-virtualization.test.ts:74-75`

**Issue:**

```ts
const domRows = within(result.getByRole('table')).getAllByRole('row').slice(1);
expect(domRows.length).toBeLessThanOrEqual(5);
```

The failure mode WR-04's `{#if row}` guard introduces is *silently skipping rows*, and
`toBeLessThanOrEqual(5)` is satisfied by **0**. If a future change made the guard swallow
every row — the precise regression this file exists to detect — this test stays green.
`expect(result.getByRole('table')).toBeInTheDocument()` only proves the component did not
throw.

I measured the real value this session (probe output quoted under WR-04 above): it is
exactly 5, all five present and correct. So the strict assertion is available today at no
cost.

**Fix:**

```ts
expect(domRows.map((r) => r.getAttribute('data-testid'))).toEqual([
	'table-row-pkg/file0.go:1:symbol-0',
	'table-row-pkg/file1.go:2:symbol-1',
	'table-row-pkg/file2.go:3:symbol-2',
	'table-row-pkg/file3.go:4:symbol-3',
	'table-row-pkg/file4.go:5:symbol-4'
]);
```

---

## Info — re-review

### RR-I-01: The header row's `aria-rowindex={1}` is hardcoded inside the header-group loop

**File:** `web/src/lib/components/workbench/DataTable.svelte:130-131`

```svelte
{#each table.getHeaderGroups() as headerGroup (headerGroup.id)}
	<Table.Row aria-rowindex={1}>
```

With a grouped column definition TanStack emits more than one header group, and every
header row would then claim `aria-rowindex="1"` — violating the same spec clause quoted in
RR-W-01 ("greater than the `aria-rowindex` value of any previous rows"), and colliding with
the data rows' `+2` offset. Unreachable today: all four `*-columns.ts` files define flat
`ColumnDef` arrays with no `columns:` nesting. Worth deriving it from the loop index while
the code is being touched, since `DataTable` is the deliberately generic shell D-06 hands
to Phase 5.

### RR-I-02: `workbench-failure.ts`'s "four titles" doc comment is now stale, and the test it points at samples 4 of 5 branches

**File:** `web/src/lib/workbench-failure.ts:26-29`

> `// The four titles below are PAIRWISE DISTINCT … and is asserted by web/tests/workbench-failure.test.ts.`

WR-01 made the `not-found` + `no-index` branch a fifth distinct title, so there are now
five. The test the comment cites (`workbench-failure.test.ts:78-86`) builds its Set from
four inputs and never exercises the `indexing` branch, so its `Set(titles).size === 4`
assertion still passes — but it no longer asserts what the comment claims. The new WR-01
test (`:99-121`) covers the one newly-risky pair (`indexing` vs `no-index`) directly, so
nothing is unguarded; only the comment and the older test's framing are out of date.

### RR-I-03: WR-06's copied Corepack block drops `web:deps`'s pnpm-version guard and announces success unconditionally

**File:** `Taskfile.yml:1286-1299` (compare `web:deps` at `:419-432`)

Two divergences from the block it says it mirrors:

1. `web:deps`'s `elif` arm rejects a pnpm older than major 10 with a named error
   (`Taskfile.yml:423-428`); `web:components:drift`'s `elif` just echoes and proceeds, so
   a pnpm-8 host reaches `pnpm install --frozen-lockfile` against a lockfile it cannot
   honour and fails with a confusing message instead of the diagnostic one that already
   exists a thousand lines up the same file.
2. `corepack enable >/dev/null 2>&1 || true` swallows failure, yet the very next line
   echoes `"corepack resolved the pnpm version from web/package.json's packageManager
   field"` unconditionally. On a host where `corepack enable` cannot write its shims, the
   log claims success and the recipe then dies at `pnpm: command not found`. (`web:deps`
   has the identical shape, so this is a copied trait, not a novel one.)

Neither is a merge blocker — this target is schedule-only per D-16 — and it did run green
here.

### RR-I-04: `GetHealthResponse.stale` is populated on the wire and read by nothing

**Files:** `internal/uiserver/handlers.go:917` (`Stale: result.Stale`), `web/src/lib/gen/ui_pb.ts:1584` (field 15), `web/src/routes/health/+page.svelte`

`rg -n "\bstale\b" web/src/` shows `GetStatusResponse.stale` **is** rendered
(`routes/+page.svelte:86`, `<dd>{status.stale ? 'yes' : 'no'}</dd>`) and `IndexStatus`'s
`'stale'` verdict drives `TrustVerdict.svelte:25-31` — but nothing reads
`GetHealthResponse.stale`. CR-02's fix took the "remove the render" branch rather than the
review's suggested "gate on the real `stale` signal" branch, which is defensible, but the
result is that `/health` now carries **no** pending-work signal at all and `stale` joins
`pending_changes` as a wire field with no consumer.

Correcting the original review's own record while I am here: CR-02 asserted the fabricated
tally *"actively repudiates the `stale` flag rendered two lines above it."* That was wrong
— `stale` was never rendered on `/health`; the line two above it is
`Re-index recommended: {freshness.reindexRecommended ? 'yes' : 'no'}`. The finding's
substance stands; that one sentence did not.

---

## Evidence hygiene

Three throwaway probe files were written under `web/tests/`
(`zzz-rereview-probe.test.ts`, `zzz-rereview-probe2.test.ts`, `zzz-rereview-probe3.test.ts`)
and one source mutation was applied to `internal/uiserver/handlers.go`. All four were
reverted; `git status --porcelain` returns empty. Every quoted probe output above is
verbatim stdout from a run in this session.

Not verified, stated as such: I did not exercise the `web:components:drift` corepack branch
on a corepack-only host (RR-I-03 is a read of the recipe against its sibling, plus one
successful local run on a host that has both).

---

_Re-reviewed: 2026-08-30T07:12:03Z_
_Reviewer: Claude (gsd-code-reviewer), independent post-fix verification pass_
_Depth: deep_
_Fix-pass base: d8f4289f_

---

## Re-review findings — resolution (2026-08-30)

RR-W-01, RR-W-02 and RR-W-03 were all fixed in commit `4f746d57`, after the re-review
section above was written. This note closes the currency gap the Phase 4 verifier
identified: the section above records them as they stood at re-review time, not as they
stand now.

- **RR-W-01** — `aria-rowcount` is now `table.getRowModel().rows.length + 1`
  (`DataTable.svelte:128`), so it can never be smaller than the largest `aria-rowindex`
  (header at 1, data rows at `index + 2`). The render-cost assertion that read
  `aria-rowcount` as a model-row proxy subtracts the header rather than being relaxed;
  verified it still fails against a truncated (500) and a blank (1) table.
- **RR-W-02** — `<Command.Empty>No results.</Command.Empty>` is now gated on
  `{#if !searchState.failure}` (`FilePicker.svelte:113`), so the literal DOM text
  "server unavailable No results." can no longer occur.
- **RR-W-03** — the virtualization guard gained `expect(domRows.length).toBeGreaterThan(0)`
  alongside its existing upper bound (`data-table-virtualization.test.ts`), so a blanked
  component (0 rows) now fails where it previously passed.

Verified after the fix: `task web:test` 268/268, `pnpm check` 1020 files / 0 errors,
`web:drift` / `web:components:drift` / `proto:drift` / `web:render-cost` all PASS, bundle
rebuilt. Frontmatter `status` moved `issues_found` → `clean` accordingly.
