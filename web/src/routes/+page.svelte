<script lang="ts">
	// D-19: the shell's live GetStatus render — the seam proven in the
	// browser (embedded asset -> generated TS client -> Connect JSON ->
	// Phase 1's handler -> real index data). uiClient is 02-03's single
	// transport site; this page never constructs its own transport.
	import { onMount } from 'svelte';
	import { uiClient } from '$lib/client';
	import type { GetStatusResponse } from '$lib/gen/ui_pb';

	type LoadState =
		| { kind: 'loading' }
		| { kind: 'error'; message: string }
		| { kind: 'loaded'; status: GetStatusResponse };

	let state = $state<LoadState>({ kind: 'loading' });

	onMount(() => {
		uiClient
			.getStatus({})
			.then((status) => {
				state = { kind: 'loaded', status };
			})
			.catch((err: unknown) => {
				// T-02-05-05: a transport-level rejection renders a named
				// error state rather than an indefinite loading spinner.
				// One catch, one error view — this is a shell, not an
				// error-handling framework.
				state = { kind: 'error', message: err instanceof Error ? err.message : String(err) };
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
