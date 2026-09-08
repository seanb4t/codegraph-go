<script lang="ts">
	// D-19: the shell's live GetStatus render — the seam proven in the
	// browser (embedded asset -> generated TS client -> Connect JSON ->
	// Phase 1's handler -> real index data). uiClient is 02-03's single
	// transport site; this page never constructs its own transport.
	//
	// 06-08/LIV-02: this route also APPLIES live events. "Live" in the
	// paragraph above is Phase 2's sense — real data rather than a
	// placeholder — and did NOT mean live-updating. Until 06-08 this, the
	// default landing view, was the one index-data surface that never
	// re-fetched: it rendered `Stale: no` over counts a generation old.
	import { onMount, getContext } from 'svelte';
	import { uiClient } from '$lib/client';
	import type { LiveStore } from '$lib/live/live-store';
	import type { GetStatusResponse } from '$lib/gen/ui_pb';

	type LoadState =
		| { kind: 'loading' }
		| { kind: 'error'; message: string }
		| { kind: 'loaded'; status: GetStatusResponse };

	let state = $state<LoadState>({ kind: 'loading' });

	const liveStore = getContext<LiveStore | undefined>('liveStore');

	// CR-01 (06-REVIEW.md), applied at this site: the mount fetch and every
	// live-triggered fetch share ONE monotonic ordering token, so a slower
	// mount response can never overwrite a newer live-triggered one. Without
	// it, a live refetch that resolves first is silently reverted by the
	// mount fetch's now-stale response, with nothing left to correct it.
	let requestId = 0;
	let liveIssuedGeneration: bigint | null = null;
	let livePendingGeneration: bigint | null = null;
	let liveInFlight = false;

	// One fetch path for both triggers. `generation` is null for the mount
	// fetch and the event's generation for a live-triggered one; only the
	// latter participates in the pending-generation coalescer.
	function issueStatusFetch(generation: bigint | null): void {
		if (generation !== null) {
			liveIssuedGeneration = generation;
			liveInFlight = true;
		}
		const id = ++requestId;
		uiClient
			.getStatus({})
			.then((status) => {
				if (id !== requestId) return;
				state = { kind: 'loaded', status };
			})
			.catch((err: unknown) => {
				// T-02-05-05: a transport-level rejection renders a named
				// error state rather than an indefinite loading spinner.
				// One catch, one error view — this is a shell, not an
				// error-handling framework.
				if (id !== requestId) return;
				state = { kind: 'error', message: err instanceof Error ? err.message : String(err) };
			})
			.finally(() => {
				if (generation === null) return;
				liveInFlight = false;
				if (livePendingGeneration !== null) {
					const next = livePendingGeneration;
					livePendingGeneration = null;
					issueStatusFetch(next);
				}
			});
	}

	onMount(() => {
		issueStatusFetch(null);
	});

	// LIV-02: a new generation re-issues this SAME GetStatus call — no rpc's
	// wire shape is duplicated into the live event (D-05). The event is a
	// TRIGGER; GetStatus stays the only source of every number rendered
	// below. Coalesced with a pending-generation flag rather than
	// suppression: an event arriving while a fetch is in flight is recorded
	// (newest generation wins) and issues exactly one follow-up once that
	// fetch settles — suppression alone would permanently lose it.
	$effect(() => {
		if (!liveStore) return;
		let first = true;
		return liveStore.subscribe((live) => {
			if (first) {
				// The store's synchronous initial delivery is a BASELINE, not
				// a trigger — otherwise mounting this view while a live event
				// is already current would issue an extra, redundant fetch.
				first = false;
				if (live) liveIssuedGeneration = live.event.generation;
				return;
			}
			if (!live) return;
			const generation = live.event.generation;
			if (liveIssuedGeneration !== null && generation <= liveIssuedGeneration) return;
			if (liveInFlight) {
				if (livePendingGeneration === null || generation > livePendingGeneration) {
					livePendingGeneration = generation;
				}
				return;
			}
			issueStatusFetch(generation);
		});
	});

	// D-19/T-02-05-07: the re-index-in-progress state is selected
	// DIRECTLY from indexingInProgress (field 9, ui.proto:172-177) —
	// NEVER inferred from storeExists && !initialized. Phase 1 added
	// field 9 for exactly this; re-deriving it here would reimplement a
	// server-side classification on strictly worse data and silently
	// diverge the moment that classification changes server-side.
	function classify(status: GetStatusResponse): 'healthy' | 'reindexing' | 'not-initialized' {
		if (status.initialized) return 'healthy';
		if (status.indexingInProgress) return 'reindexing';
		return 'not-initialized';
	}
</script>

<h1 class="text-lg font-semibold">Status</h1>

{#if state.kind === 'loading'}
	<p class="mt-4 text-sm text-muted-foreground">Loading status…</p>
{:else if state.kind === 'error'}
	<p class="mt-4 text-sm text-destructive">Could not reach the server: {state.message}</p>
{:else}
	{@const status = state.status}
	{@const kind = classify(status)}

	<p class="mt-4 text-sm font-medium">
		{#if kind === 'healthy'}
			Index is healthy.
		{:else if kind === 'reindexing'}
			A re-index is in progress. Please retry shortly — results will be stale until it
			completes.
		{:else}
			No index found for this repository.
		{/if}
	</p>

	<!-- All nine GetStatusResponse fields (ui.proto:116-177), each with a
	     human label rather than its wire name, per D-19's honesty
	     requirement. -->
	<dl class="mt-4 grid max-w-md grid-cols-[max-content_1fr] gap-x-6 gap-y-2 text-sm">
		<dt class="text-muted-foreground">Initialized</dt>
		<dd>{status.initialized ? 'yes' : 'no'}</dd>

		<dt class="text-muted-foreground">Schema version</dt>
		<dd>{status.version || 'unknown'}</dd>

		<dt class="text-muted-foreground">Nodes</dt>
		<dd>{status.nodeCount}</dd>

		<dt class="text-muted-foreground">Edges</dt>
		<dd>{status.edgeCount}</dd>

		<dt class="text-muted-foreground">Files</dt>
		<dd>{status.fileCount}</dd>

		<dt class="text-muted-foreground">Stale</dt>
		<dd>{status.stale ? 'yes' : 'no'}</dd>

		<dt class="text-muted-foreground">Indexed commit</dt>
		<dd>
			{#if status.commitSha}
				{status.commitSha}
			{:else}
				<!-- ENG-04/Phase 1 D-16: an empty commit_sha is a documented,
				     expected value — the graph predates the commit-aware
				     schema field, or was built outside a git checkout —
				     never an error. -->
				unknown (graph predates the commit-aware schema field, or was built outside a git
				checkout)
			{/if}
		</dd>

		<dt class="text-muted-foreground">Index directory found</dt>
		<dd>{status.storeExists ? 'yes' : 'no'}</dd>

		<dt class="text-muted-foreground">Re-index in progress</dt>
		<dd>{status.indexingInProgress ? 'yes' : 'no'}</dd>
	</dl>
{/if}
