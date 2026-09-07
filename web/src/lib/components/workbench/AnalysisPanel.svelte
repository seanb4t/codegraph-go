<script lang="ts" module>
	import type { Location } from '$lib/gen/ui_pb';

	// AnalysisResult is exported from the module context so a route call
	// site (or 04-06's Affected panel) can name this type when writing
	// its own `run` function's return type, without needing a generic
	// instantiation of this component to reference it.
	//
	// The result contract is an OBJECT, not a bare array of rows —
	// 04-04-PLAN.md's Task 2 step (c) explains the full rationale for the
	// rejected narrower signature. In short: ImpactResponse carries
	// node_count/edge_count ALONGSIDE affected, and 04-06's
	// AffectedResponse carries an echoed files list alongside
	// affected_tests. D-06 puts both in a header summary above the table
	// rather than in columns, and a row-array-only return type cannot
	// carry those scalars to that summary without a second request or a
	// closure side channel that a superseded dispatch could clobber with
	// a late response. Widening the return type keeps the summary and
	// the rows on the same request-identity-guarded lineage.
	export type AnalysisResult<TSummary> = { rows: Location[]; summary?: TSummary };
</script>

<script lang="ts" generics="TSummary">
	// AnalysisPanel is the ONE per-analysis wrapper (D-06) shared by all
	// four Workbench tabs — Callers already runs through it (04-01's
	// tracer inlined this exact state machine directly in +page.svelte;
	// this plan extracts it here so Impact/Callees/04-06's Affected are
	// each a columns.ts file plus a request function, not a fourth copy
	// of this markup).
	import { getContext, untrack, type Snippet } from 'svelte';
	import type { ColumnDef } from '@tanstack/svelte-table';
	import type { IndexStatus, StatusGate } from '$lib/status';
	import type { LiveStore } from '$lib/live/live-store';
	import { describeWorkbenchFailure } from '$lib/workbench-failure';
	import DataTable from './DataTable.svelte';
	import { locationRowId } from './callers-columns';
	import { features } from './table-features';

	type PanelState<TSummary> =
		| { kind: 'idle' }
		| { kind: 'loading' }
		| { kind: 'loaded'; result: AnalysisResult<TSummary> }
		| { kind: 'failed'; failure: ReturnType<typeof describeWorkbenchFailure> };

	let {
		// requestKey identifies a distinct query — the fields that, when
		// they change, mean "run this analysis again" (mirrors the
		// tracer's own callersKey discipline in the route it was
		// extracted from). AnalysisPanel never inspects requestKey's
		// contents — it only reacts to the string changing — so each tab
		// composes its own key from whichever fields select ITS query
		// (symbol+depth for Impact, symbol+limit for Callers/Callees)
		// without this component needing to know the difference.
		requestKey,
		// run is undefined when the current inputs do not yet describe a
		// runnable query (e.g. no symbol typed) — the panel renders its
		// idle state and issues no request. Defined, it is called with a
		// fresh AbortSignal per dispatch, mirroring search.ts's
		// abort-plus-monotonic-identity guard so a slow response to a
		// superseded query can never overwrite a newer one.
		run,
		columns,
		summary,
		inputs,
		emptyMessage,
		onSelect
	}: {
		requestKey: string;
		run: ((signal: AbortSignal) => Promise<AnalysisResult<TSummary>>) | undefined;
		columns: ColumnDef<typeof features, Location>[];
		summary?: Snippet<[TSummary]>;
		inputs: Snippet;
		emptyMessage: string;
		onSelect?: (row: Location) => void;
	} = $props();

	// Read from context, never created here — this panel never issues a
	// GetStatus call of its own (must_haves: "Moving the depth control or
	// the limit control issues NO additional GetStatus call").
	const statusGate = getContext<StatusGate>('statusGate');
	let indexStatus: IndexStatus = $state({ verdict: 'unknown', commit: 'unknown', commitSha: '' });
	$effect(() => {
		return statusGate.subscribe((s) => {
			indexStatus = s;
		});
	});

	let panelState: PanelState<TSummary> = $state({ kind: 'idle' });

	let requestId = 0;

	// The effect's ONLY tracked dependency is requestKey — reading `run`
	// through `untrack` (mirroring the route-level tracer's
	// `untrack(() => params)`) means a parent re-render that leaves
	// requestKey unchanged can never re-trigger a dispatch. This is the
	// actual mechanism behind WRK-01's "no extra request, no extra
	// GetStatus call" property: replaceState changes page.url, params
	// re-derives, but requestKey (and therefore this effect) only changes
	// when a field that identifies THIS query actually changed.
	//
	// Cancellation and request-identity invalidation are BOTH owned by
	// this effect's own returned cleanup — not by a hand-rolled
	// module-level AbortController variable. Svelte calls the previous
	// run's cleanup before every re-run (requestKey changed) AND on
	// unmount (tab switch away from Workbench), so there is exactly one
	// mechanism, not two: it aborts the in-flight request AND bumps
	// requestId in the same step, which is what makes the abort's own
	// rejection unambiguously stale on every path that aborts — including
	// the `run → undefined` transition (CR-01) and component teardown
	// (WR-02). Without the requestId bump, an aborted request's
	// `Code.Canceled` rejection still carries the id that (until the next
	// dispatch) still equals `requestId`, so the `id !== requestId` guard
	// below passes and a stale abort overwrites a freshly-set `idle`
	// state with a false "Something went wrong" failure.
	$effect(() => {
		requestKey;
		const currentRun = untrack(() => run);

		if (!currentRun) {
			panelState = { kind: 'idle' };
			return;
		}

		panelState = { kind: 'loading' };
		const controller = new AbortController();
		const id = ++requestId;

		currentRun(controller.signal)
			.then((result) => {
				if (id !== requestId) return; // superseded — discard
				panelState = { kind: 'loaded', result };
			})
			.catch((err: unknown) => {
				if (id !== requestId) return; // superseded — discard, not a real failure
				panelState = { kind: 'failed', failure: describeWorkbenchFailure(err, indexStatus) };
			});

		return () => {
			controller.abort();
			requestId += 1; // invalidate this request's own settlement, on every teardown path
		};
	});

	// 06-03 Task 3 (LIV-02): a new generation from the live store
	// re-issues `run` — but ONLY when this panel currently holds a
	// result. A live event must not launch an analysis the developer
	// never asked for (idle/loading/failed all skip it). Coalesced with
	// a PENDING-GENERATION flag, not suppression — see health/+page.svelte's
	// identical comment for why suppression alone would lose the newest
	// state. Reuses the SAME requestId counter as the effect above, so a
	// param-driven dispatch and a live-triggered one can never race.
	const liveStore = getContext<LiveStore | undefined>('liveStore');
	let panelLiveIssuedGeneration: bigint | null = null;
	let panelLivePendingGeneration: bigint | null = null;
	let panelLiveInFlight = false;

	function issueLiveRerun(): void {
		const currentRun = untrack(() => run);
		if (panelState.kind !== 'loaded' || !currentRun) return;
		panelLiveInFlight = true;
		const controller = new AbortController();
		const id = ++requestId;
		currentRun(controller.signal)
			.then((result) => {
				if (id !== requestId) return;
				panelState = { kind: 'loaded', result };
			})
			.catch((err: unknown) => {
				if (id !== requestId) return;
				panelState = { kind: 'failed', failure: describeWorkbenchFailure(err, indexStatus) };
			})
			.finally(() => {
				panelLiveInFlight = false;
				if (panelLivePendingGeneration !== null) {
					panelLivePendingGeneration = null;
					issueLiveRerun();
				}
			});
	}

	$effect(() => {
		if (!liveStore) return;
		let first = true;
		return liveStore.subscribe((live) => {
			if (first) {
				first = false;
				if (live) panelLiveIssuedGeneration = live.event.generation;
				return;
			}
			if (!live) return;
			const generation = live.event.generation;
			if (panelLiveIssuedGeneration !== null && generation <= panelLiveIssuedGeneration) return;
			panelLiveIssuedGeneration = generation;
			if (panelLiveInFlight) {
				if (panelLivePendingGeneration === null || generation > panelLivePendingGeneration) {
					panelLivePendingGeneration = generation;
				}
				return;
			}
			issueLiveRerun();
		});
	});
</script>

<div class="mt-4 flex flex-wrap items-end gap-3">
	{@render inputs()}
</div>

<div class="mt-4">
	{#if panelState.kind === 'failed'}
		<div
			data-testid={`workbench-failure-${panelState.failure.kind}`}
			class="rounded-md border border-destructive/40 bg-destructive/10 p-3 text-sm text-destructive"
		>
			<p class="font-semibold">{panelState.failure.title}</p>
			<p>{panelState.failure.detail}</p>
		</div>
	{:else if panelState.kind === 'loading'}
		<p class="text-sm text-muted-foreground" data-testid="workbench-loading">Loading…</p>
	{:else if panelState.kind === 'loaded'}
		{#if summary && panelState.result.summary !== undefined}
			<div data-testid="workbench-summary" class="mb-3 text-sm text-muted-foreground">
				{@render summary(panelState.result.summary)}
			</div>
		{/if}
		<DataTable
			rows={panelState.result.rows}
			{columns}
			getRowId={locationRowId}
			{emptyMessage}
			{onSelect}
		/>
	{:else}
		<p class="text-sm text-muted-foreground">Enter a symbol to run this analysis.</p>
	{/if}
</div>
