<script lang="ts">
	// Fills the placeholder this route used to be (D-18) with the
	// thinnest production-quality path: one FileGraph rpc on mount,
	// rolled up through file-graph-transform.ts, rendered through
	// GraphCanvas.svelte's renderer swap seam. The collapsed directory
	// view is the default first paint; selecting a directory reveals its
	// files in place, with zero further requests — the whole file graph
	// already arrived on mount. Selecting a FILE node does nothing here:
	// file-to-symbol expansion is a later route feature, not this one's.
	import { onMount } from 'svelte';
	import { uiClient } from '$lib/client';
	import { classifyRpcError, type RpcFailure } from '$lib/rpc-errors';
	import {
		rollupToElements,
		plannedNodeCount,
		EXPANSION_NODE_CEILING,
		type FileGraphElement
	} from '$lib/components/graph/file-graph-transform';
	import { fileGraphStyle } from '$lib/components/graph/graph-style';
	import GraphCanvas from '$lib/components/graph/GraphCanvas.svelte';
	import type { FileGraphResponse } from '$lib/gen/ui_pb';

	type GraphState =
		| { kind: 'loading' }
		| { kind: 'failed'; failure: RpcFailure }
		| { kind: 'loaded'; response: FileGraphResponse; requestIssuedAt: number };

	let graphState = $state<GraphState>({ kind: 'loading' });
	// The typed set of expanded directory paths — the WHOLE of expansion
	// state (T-05-47). Never a rendered class list, never a parsed node
	// id: rollupToElements(response, expandedDirs) recomputes the
	// complete element array from this set and the already-fetched
	// response on every change.
	let expandedDirs = $state<Set<string>>(new Set());
	let refusalMessage = $state<string | undefined>(undefined);

	let elements = $derived(
		graphState.kind === 'loaded' ? rollupToElements(graphState.response, expandedDirs) : ([] as FileGraphElement[])
	);
	let nodeCount = $derived(elements.filter((el) => !('source' in el.data)).length);
	let edgeCount = $derived(elements.filter((el) => 'source' in el.data).length);

	// cycleGroups regroups the CURRENT elements array (already respecting
	// expandedDirs) by the typed numeric cycleId already on element data —
	// a file node's own cycleId, or a collapsed directory's cycleIds
	// union (collapsedDirElement's own union of its files, a server-side
	// value — see D-06).
	// Every file's cycle id is represented by SOME element in the current
	// view (itself when its directory is expanded, its collapsed
	// directory's union otherwise), so this always yields exactly
	// cycleCount distinct groups regardless of expansion state. Sorted by
	// cycle id ascending for a stable visiting order across activations.
	// Reads ONLY data.cycleId / data.cycleIds — never a class list, never
	// a computed adjacency: this is a grouping of typed data the server
	// already computed, not a second derivation of it.
	let cycleGroups = $derived.by(() => {
		const groups = new Map<number, string[]>();
		function addTo(cycleId: number, id: string) {
			const existing = groups.get(cycleId);
			if (existing) {
				existing.push(id);
			} else {
				groups.set(cycleId, [id]);
			}
		}
		for (const el of elements) {
			const data = el.data;
			if ('source' in data) continue;
			if (data.cycleId !== undefined && data.cycleId !== 0) {
				addTo(data.cycleId, data.id);
			}
			if (data.cycleIds !== undefined) {
				for (const cycleId of data.cycleIds) {
					addTo(cycleId, data.id);
				}
			}
		}
		return [...groups.entries()].sort((a, b) => a[0] - b[0]).map(([, ids]) => ids);
	});

	// The whole of cycle-focus state (T-05-47's discipline, applied
	// here): cycleFocusIds is the id set currently handed to GraphCanvas
	// as its focusNodeIds prop; cycleFocusNextIndex is which group
	// activateCycleFocus visits next (wraps via modulo);
	// cycleFocusViewing is the 1-based "cycle N of M" the control's own
	// label states, so repeated activation is legible rather than a jump
	// to somewhere unexplained.
	let cycleFocusIds = $state<string[]>([]);
	let cycleFocusNextIndex = $state(0);
	let cycleFocusViewing = $state<number | undefined>(undefined);

	function activateCycleFocus() {
		const groups = cycleGroups;
		if (groups.length === 0) return;
		const idx = cycleFocusNextIndex % groups.length;
		cycleFocusIds = groups[idx];
		cycleFocusViewing = idx + 1;
		cycleFocusNextIndex = idx + 1;
	}

	onMount(() => {
		// requestIssuedAt is captured HERE, before the rpc call, and
		// handed to GraphCanvas as a plain number prop — the canvas does
		// not guess it and does not reach for a navigation timing API of
		// its own. This is the start of the WIDE timeToInteractiveMs bar:
		// rpc round trip, protobuf decode, and the wire-to-elements
		// transform are all inside it. This is the ONLY FileGraph call
		// this route ever issues — an expansion or a collapse never
		// requests again, because the whole file graph already arrived
		// here.
		const requestIssuedAt = performance.now();
		uiClient
			.fileGraph({ path: '' })
			.then((response) => {
				graphState = { kind: 'loaded', response, requestIssuedAt };
			})
			.catch((err: unknown) => {
				graphState = { kind: 'failed', failure: classifyRpcError(err) };
			});
	});

	// handleNodeSelected is GraphCanvas's onNodeSelected callback. A file
	// tap is ignored outright — file-to-symbol expansion belongs to a
	// later route feature, not this one. A directory tap either collapses an already
	// expanded directory (always allowed — collapsing can only shrink the
	// rendered count) or expands a collapsed one, consulting
	// plannedNodeCount against EXPANSION_NODE_CEILING FIRST and leaving
	// the graph untouched with a readable refusal when the ceiling would
	// be exceeded.
	function handleNodeSelected(id: string, isDirectory: boolean) {
		if (!isDirectory || graphState.kind !== 'loaded') return;

		if (expandedDirs.has(id)) {
			const next = new Set(expandedDirs);
			next.delete(id);
			expandedDirs = next;
			refusalMessage = undefined;
			return;
		}

		const next = new Set(expandedDirs);
		next.add(id);
		const planned = plannedNodeCount(graphState.response, next);
		if (planned > EXPANSION_NODE_CEILING) {
			refusalMessage = `Expanding this directory would render ${planned} nodes, past the limit of ${EXPANSION_NODE_CEILING}. Collapse another directory first.`;
			return;
		}
		refusalMessage = undefined;
		expandedDirs = next;
	}

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

{#if graphState.kind === 'loading'}
	<p class="mt-4 text-sm text-muted-foreground" data-testid="graph-loading">
		Loading the file graph…
	</p>
{:else if graphState.kind === 'failed'}
	<p class="mt-4 text-sm text-destructive" data-testid={`graph-failure-${graphState.failure.kind}`}>
		{failureMessage(graphState.failure)}
	</p>
{:else if elements.length === 0}
	<p class="mt-4 text-sm text-muted-foreground" data-testid="graph-empty">
		This repository has no files to graph yet.
	</p>
{:else}
	<p class="mt-1 text-xs text-muted-foreground" data-testid="graph-summary">
		{nodeCount} nodes, {edgeCount} edges. Select a directory to reveal the files inside it; select
		it again to collapse it back.
	</p>
	<p class="mt-1 text-xs text-muted-foreground" data-testid="graph-cycle-summary">
		{#if graphState.response.cycleCount > 0}
			This repository has {graphState.response.cycleCount} dependency cycle{graphState.response
				.cycleCount === 1
				? ''
				: 's'}.
		{:else}
			This repository has no dependency cycles.
		{/if}
	</p>
	{#if graphState.response.cycleCount > 0}
		<button
			type="button"
			class="mt-1 text-xs text-primary underline"
			data-testid="graph-cycle-focus"
			onclick={activateCycleFocus}
		>
			{#if cycleFocusViewing !== undefined}
				Focus next cycle (cycle {cycleFocusViewing} of {graphState.response.cycleCount})
			{:else}
				Focus a cycle ({graphState.response.cycleCount} total)
			{/if}
		</button>
	{/if}
	{#if refusalMessage}
		<p class="mt-1 text-xs text-destructive" data-testid="graph-refusal">{refusalMessage}</p>
	{/if}
	<div class="mt-4 h-[calc(100vh-10rem)] w-full">
		<GraphCanvas
			{elements}
			style={fileGraphStyle}
			requestIssuedAt={graphState.requestIssuedAt}
			focusNodeIds={cycleFocusIds}
			onNodeSelected={handleNodeSelected}
		/>
	</div>
{/if}
