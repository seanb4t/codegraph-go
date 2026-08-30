<script lang="ts">
	// Fills the placeholder this route used to be (D-18) with the
	// thinnest production-quality path: one FileGraph rpc on mount,
	// transformed through file-graph-transform.ts, rendered through
	// GraphCanvas.svelte's renderer swap seam. No cycle styling, no
	// in-place expansion, no second call site — those follow only once a
	// downstream latency measurement clears its locked threshold.
	import { onMount } from 'svelte';
	import { uiClient } from '$lib/client';
	import { classifyRpcError, type RpcFailure } from '$lib/rpc-errors';
	import {
		rollupToElements,
		type FileGraphElement
	} from '$lib/components/graph/file-graph-transform';
	import { fileGraphStyle } from '$lib/components/graph/graph-style';
	import GraphCanvas from '$lib/components/graph/GraphCanvas.svelte';

	type GraphState =
		| { kind: 'loading' }
		| { kind: 'failed'; failure: RpcFailure }
		| { kind: 'loaded'; elements: FileGraphElement[]; requestIssuedAt: number };

	let state = $state<GraphState>({ kind: 'loading' });

	onMount(() => {
		// requestIssuedAt is captured HERE, before the rpc call, and
		// handed to GraphCanvas as a plain number prop — the canvas does
		// not guess it and does not reach for a navigation timing API of
		// its own. This is the start of the WIDE timeToInteractiveMs bar:
		// rpc round trip, protobuf decode, and the wire-to-elements
		// transform are all inside it.
		const requestIssuedAt = performance.now();
		uiClient
			.fileGraph({ path: '' })
			.then((response) => {
				state = {
					kind: 'loaded',
					// The collapsed default (05-08, GRF-01's remedy):
					// progressive directory-to-file expansion lands in a
					// later task on this same route; this call already
					// renders the correct first-paint view.
					elements: rollupToElements(response, new Set()),
					requestIssuedAt
				};
			})
			.catch((err: unknown) => {
				state = { kind: 'failed', failure: classifyRpcError(err) };
			});
	});

	// One vocabulary for a failed rpc (D-04): classifyRpcError's four
	// kinds, each with a plain sentence — never a second error taxonomy
	// invented for this route.
	function failureMessage(failure: RpcFailure): string {
		switch (failure.kind) {
			case 'not-found':
				return 'No file graph is available for this repository.';
			case 'invalid-input':
				return `The request was rejected: ${failure.message}`;
			case 'indexing':
				return 'The index is being rebuilt. Please retry shortly.';
			default:
				return `Could not load the graph: ${failure.message}`;
		}
	}
</script>

<h1 class="text-lg font-semibold">Graph</h1>
<p class="mt-1 text-sm text-muted-foreground">
	The files and directories in this repository, grouped by folder, with the dependencies between
	them.
</p>

{#if state.kind === 'loading'}
	<p class="mt-4 text-sm text-muted-foreground" data-testid="graph-loading">
		Loading the file graph…
	</p>
{:else if state.kind === 'failed'}
	<p
		class="mt-4 text-sm text-destructive"
		data-testid={`graph-failure-${state.failure.kind}`}
	>
		{failureMessage(state.failure)}
	</p>
{:else if state.elements.length === 0}
	<p class="mt-4 text-sm text-muted-foreground" data-testid="graph-empty">
		This repository has no files to graph yet.
	</p>
{:else}
	<div class="mt-4 h-[calc(100vh-10rem)] w-full">
		<GraphCanvas
			elements={state.elements}
			style={fileGraphStyle}
			requestIssuedAt={state.requestIssuedAt}
		/>
	</div>
{/if}
