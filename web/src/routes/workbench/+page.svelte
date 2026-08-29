<script lang="ts">
	// D-18: fills the Phase 4 placeholder this route mounted. Reads view
	// state EXCLUSIVELY from page.url.searchParams via workbench-url.ts's
	// parseWorkbenchParams — the full input state lives in the URL (D-12)
	// — and every control write goes through serializeWorkbenchParams +
	// replaceState, never a direct state assignment. Only the `callers`
	// mode is wired this task (04-01 Task 2); the other three modes render
	// an explicit "not yet wired" placeholder that 04-04/04-06 replace.
	import { page } from '$app/state';
	import { replaceState } from '$app/navigation';
	import { getContext, untrack } from 'svelte';
	import { uiClient } from '$lib/client';
	import type { IndexStatus, StatusGate } from '$lib/status';
	import {
		parseWorkbenchParams,
		serializeWorkbenchParams,
		isShapeInteger,
		type WorkbenchParams
	} from '$lib/workbench-url';
	import { describeWorkbenchFailure } from '$lib/workbench-failure';
	import DataTable from '$lib/components/workbench/DataTable.svelte';
	import { callersColumns, locationRowId } from '$lib/components/workbench/callers-columns';
	import type { Location } from '$lib/gen/ui_pb';

	let params = $derived(parseWorkbenchParams(page.url.searchParams));

	// D-03/D-05: this view's own copy of the layout's shared index-health
	// status, read by SUBSCRIBING to the SAME gate the layout created —
	// never a second gate, which would be a second GetStatus call.
	const statusGate = getContext<StatusGate>('statusGate');
	let indexStatus = $state<IndexStatus>({ verdict: 'unknown', commit: 'unknown' });
	$effect(() => {
		return statusGate.subscribe((s) => {
			indexStatus = s;
		});
	});

	type CallersState =
		| { kind: 'idle' }
		| { kind: 'loading' }
		| { kind: 'loaded'; rows: Location[] }
		| { kind: 'failed'; failure: ReturnType<typeof describeWorkbenchFailure> };

	let callersState = $state<CallersState>({ kind: 'idle' });

	// The load effect depends ONLY on the fields identifying a distinct
	// Callers query — mode/symbol/limit — mirroring the Browse route's own
	// targetKey discipline (browse/+page.svelte) so an unrelated URL write
	// never tears down an in-flight or already-loaded result.
	let callersKey = $derived(
		JSON.stringify([params.mode ?? null, params.symbol ?? null, params.limit ?? null])
	);

	$effect(() => {
		callersKey;
		const current = untrack(() => params);

		if (current.mode !== 'callers' || !current.symbol) {
			callersState = { kind: 'idle' };
			return;
		}

		callersState = { kind: 'loading' };
		const symbol = current.symbol;
		// D-07: limit passes straight through unchanged — the server's own
		// validateLimit already bounds it for every caller. No client-side
		// range check is applied here or anywhere in web/.
		const limit = current.limit ?? 0;

		uiClient
			.callers({ symbol, limit })
			.then((response) => {
				callersState = { kind: 'loaded', rows: response.callers };
			})
			.catch((err: unknown) => {
				callersState = {
					kind: 'failed',
					failure: describeWorkbenchFailure(err, indexStatus)
				};
			});
	});

	// Every Workbench control write goes through this one function (D-12):
	// merge the changed field(s) into the CURRENT parsed params, serialize
	// the whole result, and replace the URL in place — the address bar
	// stays correct and shareable at every instant, with no history entry
	// per keystroke.
	function writeParams(next: Partial<WorkbenchParams>): void {
		const merged: WorkbenchParams = { ...params, ...next };
		const url = new URL(page.url.href);
		url.search = serializeWorkbenchParams(merged).toString();
		replaceState(url, {});
	}

	function handleSymbolInput(e: Event): void {
		const value = (e.currentTarget as HTMLInputElement).value;
		writeParams({ symbol: value || undefined });
	}

	// limitInvalid (mirrors NeighborsPanel.svelte's depthInvalid, IN-08):
	// a non-conforming value is simply not written — the input keeps
	// showing what the developer typed, but the URL never claims a limit
	// this app cannot itself parse back on the very next read.
	let limitInvalid = $state(false);

	function handleLimitInput(e: Event): void {
		const raw = (e.currentTarget as HTMLInputElement).value;
		if (raw === '') {
			limitInvalid = false;
			writeParams({ limit: undefined });
			return;
		}
		if (!isShapeInteger(raw)) {
			limitInvalid = true;
			return;
		}
		limitInvalid = false;
		writeParams({ limit: Number(raw) });
	}
</script>

<h1 class="text-lg font-semibold">Workbench</h1>
<p class="mt-1 text-sm text-muted-foreground">
	Phase 4: run the four graph analyses interactively with their own knobs.
</p>

{#if params.mode === 'callers'}
	<div class="mt-4 flex flex-wrap items-end gap-3">
		<label class="text-sm">
			Symbol
			<input
				type="text"
				value={params.symbol ?? ''}
				oninput={handleSymbolInput}
				data-testid="workbench-symbol-input"
				class="block rounded-md border border-input bg-background px-2 py-1 text-sm"
			/>
		</label>
		<label class="text-sm">
			Limit
			<input
				type="number"
				value={params.limit ?? ''}
				oninput={handleLimitInput}
				data-testid="workbench-limit-input"
				aria-invalid={limitInvalid}
				class="block rounded-md border border-input bg-background px-2 py-1 text-sm"
			/>
		</label>
	</div>

	<div class="mt-4">
		{#if callersState.kind === 'failed'}
			<div
				data-testid={`workbench-failure-${callersState.failure.kind}`}
				class="rounded-md border border-destructive/40 bg-destructive/10 p-3 text-sm text-destructive"
			>
				<p class="font-semibold">{callersState.failure.title}</p>
				<p>{callersState.failure.detail}</p>
			</div>
		{:else if callersState.kind === 'loading'}
			<p class="text-sm text-muted-foreground" data-testid="workbench-loading">Loading…</p>
		{:else if callersState.kind === 'loaded'}
			<DataTable
				rows={callersState.rows}
				columns={callersColumns}
				getRowId={locationRowId}
				emptyMessage="No callers found."
			/>
		{:else}
			<p class="text-sm text-muted-foreground">Enter a symbol to run Callers.</p>
		{/if}
	</div>
{:else}
	<p class="mt-4 text-sm text-muted-foreground" data-testid="workbench-mode-placeholder">
		This analysis is not yet wired.
	</p>
{/if}
