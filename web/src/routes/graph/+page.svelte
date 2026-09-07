<script lang="ts">
	// Fills the placeholder this route used to be (D-18) with the
	// thinnest production-quality path: one FileGraph rpc on mount,
	// rolled up through file-graph-transform.ts, rendered through
	// GraphCanvas.svelte's renderer swap seam. The collapsed directory
	// view is the default first paint; selecting a directory reveals its
	// files in place, with zero further requests — the whole file graph
	// already arrived on mount. Selecting a FILE node fetches and expands
	// its symbols in place — the second, INCREMENTAL expansion path this
	// route drives, layered on top of the directory rollup rather than
	// recomputed by it.
	import { onMount, getContext } from 'svelte';
	import { uiClient } from '$lib/client';
	import { classifyRpcError, type RpcFailure } from '$lib/rpc-errors';
	import {
		rollupToElements,
		plannedNodeCount,
		EXPANSION_NODE_CEILING,
		symbolElementsForFile,
		symbolElementIdsForFile,
		dirOf,
		type FileGraphElement
	} from '$lib/components/graph/file-graph-transform';
	import { fileGraphStyle } from '$lib/components/graph/graph-style';
	import GraphCanvas from '$lib/components/graph/GraphCanvas.svelte';
	import DataTable from '$lib/components/workbench/DataTable.svelte';
	import { edgeKindColumns, edgeKindRowId, type EdgeKindRow } from '$lib/components/graph/edge-kind-columns';
	import type { LiveStore } from '$lib/live/live-store';
	import type { FileGraphResponse, FileSymbolsResponse } from '$lib/gen/ui_pb';
	import type { FileGraphEdgeData } from '$lib/components/graph/file-graph-transform';

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

	// File-to-symbol expansion state — deliberately SEPARATE from
	// expandedDirs/elements above rather than folded into
	// the same rollup: a file's symbols are fetched incrementally, per
	// file, and applied to GraphCanvas through its addedElements/
	// removedElementIds props (an ADD/REMOVE seam), never through a
	// recomputed `elements` array. fileSymbolsCache is the WHOLE of "have
	// we already fetched this file's symbols" — once a path is cached,
	// every subsequent expand/collapse/re-expand of that file is served
	// from it with zero further requests, the once-per-file contract this
	// plan's own three-click test proves. expandedFiles is the WHOLE of
	// "which files are currently showing their symbols" — reassigned
	// (never mutated in place), the same discipline expandedDirs already
	// follows.
	let fileSymbolsCache = $state<Map<string, FileSymbolsResponse>>(new Map());
	let expandedFiles = $state<Set<string>>(new Set());
	let fileExpansionFailures = $state<Map<string, RpcFailure>>(new Map());
	// addedElements/removedElementIds are the plain-data batches handed to
	// GraphCanvas's incremental add/remove props — reassigned to a NEW
	// array reference on every genuine change so GraphCanvas's own
	// effects (which key off array IDENTITY, not content) fire exactly
	// once per actual add or remove.
	let addedElements = $state<FileGraphElement[]>([]);
	let removedElementIds = $state<string[]>([]);
	// pendingFileSymbols is the WHOLE of "which files currently have an
	// outstanding fileSymbols request" — a plain module-local Set, never
	// $state, since it exists only to gate DISPATCH inside toggleFile
	// (never read by a template or an effect). Without it, a re-expand
	// (collapse then expand again) arriving before the first request
	// settles falls through the fetch branch a second time: expandedFiles
	// alone cannot distinguish "no request outstanding" from "a request is
	// already in flight," since the collapse branch clears expandedFiles
	// regardless of whether anything is still pending. Consulted and
	// mutated ONLY inside toggleFile's fetch branch and its two response
	// handlers below — the guard that keeps a re-expand arriving before an
	// earlier request settles from ever dispatching a second, duplicate
	// fileSymbols request for the same path.
	let pendingFileSymbols = new Set<string>();
	// mounted guards the file-symbols response handler against applying a
	// response that outlived its consumer — the SAME
	// pending-request-outliving-its-consumer lifecycle the FileGraph fetch
	// in onMount below is naturally exempt from (unmounting before that
	// ONE request settles simply never transitions graphState past
	// 'loading', so nothing downstream ever runs), asserted explicitly
	// here because this is a REPEATED, per-file async path where the
	// route can genuinely still be mounted with a DIFFERENT, unrelated
	// state by the time a given file's request resolves.
	let mounted = true;

	// elements is the array bound to GraphCanvas's full-replace `elements`
	// prop, whose own effect calls replace() on every reference change
	// (T-05-47) — deliberately NOT a $derived of graphState any more
	// (06-05 Task 2). A live-triggered refresh below updates
	// graphState.response (so a SUBSEQUENT user gesture reads fresh data)
	// but feeds its own recomputed rollup through the SEPARATE
	// liveElements state further down instead of through this one:
	// reassigning this one from a live event would trigger GraphCanvas's
	// plain replace() path, which carries NO position write-back —
	// exactly the node movement LIV-04 exists to prevent. Recomputed
	// EXPLICITLY, at exactly the two points a genuine user action changes
	// what should be rendered: the initial fetch's own .then callback
	// below, and toggleDirectory's two branches.
	let elements = $state<FileGraphElement[]>([]);
	let nodeCount = $derived(elements.filter((el) => !('source' in el.data)).length);
	let edgeCount = $derived(elements.filter((el) => 'source' in el.data).length);

	// liveElements is D-06/Task 1's live-update seam prop — a NEW array
	// reference on every applied live refresh, entirely separate from
	// `elements`/`addedElements`/`removedElementIds`: a live re-index
	// event and a user gesture are different events and must not be
	// merged into one code path (06-05-PLAN.md). See
	// issueLiveGraphRefresh below.
	let liveElements = $state<FileGraphElement[]>([]);

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

	// selectedEdge is the WHOLE of edge-selection state — a single value,
	// never a list, so selecting a different edge REPLACES the previous
	// selection by construction rather than by any explicit "clear first"
	// step. A background tap (GraphCanvas's onBackgroundTapped) sets it
	// back to undefined, which is this route's deselect signal.
	let selectedEdge = $state<
		{ source: string; target: string; data: FileGraphEdgeData } | undefined
	>(undefined);

	// edgeKindRows iterates the SELECTED edge's own wire kindCounts map
	// entries directly — never a fixed list of known kinds — so a kind
	// the server did not report for this edge produces no row. The map
	// is sparse by convention (D-03/D-06's aggregation): an absent key
	// means "never observed," not "observed zero times."
	let edgeKindRows = $derived<EdgeKindRow[]>(
		selectedEdge
			? Object.entries(selectedEdge.data.kindCounts).map(([kind, count]) => ({ kind, count }))
			: []
	);

	function handleEdgeSelected(source: string, target: string, data: Record<string, unknown>) {
		selectedEdge = { source, target, data: data as unknown as FileGraphEdgeData };
	}

	function handleBackgroundTapped() {
		selectedEdge = undefined;
	}

	onMount(() => {
		// requestIssuedAt is captured HERE, before the rpc call, and
		// handed to GraphCanvas as a plain number prop — the canvas does
		// not guess it and does not reach for a navigation timing API of
		// its own. This is the start of the WIDE timeToInteractiveMs bar:
		// rpc round trip, protobuf decode, and the wire-to-elements
		// transform are all inside it. This is the ONLY FileGraph call a
		// user-driven expand or collapse ever triggers — the whole file
		// graph already arrived here; a LIVE re-index event is the one
		// other source of a fileGraph call this route ever issues, and it
		// is routed entirely separately (see issueLiveGraphRefresh below).
		const requestIssuedAt = performance.now();
		uiClient
			.fileGraph({ path: '' })
			.then((response) => {
				graphState = { kind: 'loaded', response, requestIssuedAt };
				elements = rollupToElements(response, expandedDirs);
			})
			.catch((err: unknown) => {
				graphState = { kind: 'failed', failure: classifyRpcError(err) };
			});

		return () => {
			mounted = false;
		};
	});

	// --- 06-05 Task 2 (LIV-02/LIV-04): the live re-index subscription ---
	//
	// Per D-05, the event carries NO graph payload — only "the index
	// moved" plus a monotonic generation — so this route re-fetches
	// through the SAME rpcs it already owns rather than any rpc's wire
	// shape being duplicated into the event. The coalescer below mirrors
	// health/+page.svelte's and browse/+page.svelte's own pending-
	// generation pattern (06-03) exactly: an event arriving while a
	// live-triggered refresh is already in flight is recorded (the
	// newest generation wins) and issues exactly ONE follow-up once that
	// refresh settles, rather than stacking a second concurrent call.
	const liveStore = getContext<LiveStore | undefined>('liveStore');
	let liveIssuedGeneration: bigint | null = null;
	let livePendingGeneration: bigint | null = null;
	let liveInFlight = false;

	// issueLiveGraphRefresh re-issues the fileGraph call, PLUS a
	// fileSymbols call for every file CURRENTLY showing its symbols
	// (through the same rpc that path already uses — never a new async
	// surface), and feeds the combined rollup through liveElements
	// (Task 1's seam) rather than through `elements` — see `elements`'s
	// own declaration comment above for why. graphState.response and
	// fileSymbolsCache ARE updated here, so the next user gesture reads
	// fresh data instead of reverting this refresh on its next toggle.
	function issueLiveGraphRefresh(generation: bigint): void {
		liveIssuedGeneration = generation;
		liveInFlight = true;
		const expandedFileIds = [...expandedFiles];
		Promise.all([
			uiClient.fileGraph({ path: '' }),
			...expandedFileIds.map((id) => uiClient.fileSymbols({ path: id }))
		])
			.then((results) => {
				if (!mounted) return;
				const response = results[0] as FileGraphResponse;
				const symbolResponses = results.slice(1) as FileSymbolsResponse[];
				const requestIssuedAt =
					graphState.kind === 'loaded' ? graphState.requestIssuedAt : performance.now();
				graphState = { kind: 'loaded', response, requestIssuedAt };
				if (expandedFileIds.length > 0) {
					const cache = new Map(fileSymbolsCache);
					expandedFileIds.forEach((id, i) => cache.set(id, symbolResponses[i]));
					fileSymbolsCache = cache;
				}
				const rollup = rollupToElements(response, expandedDirs);
				const symbolEls = expandedFileIds.flatMap((id, i) =>
					symbolElementsForFile(id, symbolResponses[i])
				);
				liveElements = [...rollup, ...symbolEls];
			})
			.catch(() => {
				// A live refresh failure is non-fatal — the view keeps
				// showing its last-known-good state, and the NEXT live
				// event gets another chance. Never surfaced as a blocking
				// page-level failure (the whole-graph load failure state
				// above is reserved for the initial mount fetch).
			})
			.finally(() => {
				liveInFlight = false;
				if (livePendingGeneration !== null) {
					const next = livePendingGeneration;
					livePendingGeneration = null;
					issueLiveGraphRefresh(next);
				}
			});
	}

	$effect(() => {
		if (!liveStore) return;
		let first = true;
		return liveStore.subscribe((live) => {
			if (first) {
				// The store's synchronous initial delivery is a BASELINE,
				// not a trigger — mounting this view while a live event is
				// already current must not issue an extra, redundant
				// refresh.
				first = false;
				if (live) liveIssuedGeneration = live.event.generation;
				return;
			}
			if (!live) return;
			if (graphState.kind !== 'loaded') return; // nothing to refresh yet
			const generation = live.event.generation;
			if (liveIssuedGeneration !== null && generation <= liveIssuedGeneration) return;
			if (liveInFlight) {
				if (livePendingGeneration === null || generation > livePendingGeneration) {
					livePendingGeneration = generation;
				}
				return;
			}
			issueLiveGraphRefresh(generation);
		});
	});

	// handleNodeSelected is GraphCanvas's onNodeSelected callback,
	// dispatching on the tapped element's kind. A symbol tap is a no-op —
	// a symbol has nothing further to expand.
	function handleNodeSelected(id: string, kind: 'directory' | 'file' | 'symbol') {
		if (kind === 'directory') {
			toggleDirectory(id);
		} else if (kind === 'file') {
			toggleFile(id);
		}
	}

	// toggleDirectory either collapses an already expanded directory
	// (always allowed — collapsing can only shrink the rendered count) or
	// expands a collapsed one, consulting plannedNodeCount against
	// EXPANSION_NODE_CEILING FIRST and leaving the graph untouched with a
	// readable refusal when the ceiling would be exceeded. Shared by the
	// canvas tap handler above AND the explicit collapse-affordance
	// buttons rendered below — one collapse rule, two ways to reach it.
	function toggleDirectory(id: string) {
		if (graphState.kind !== 'loaded') return;

		if (expandedDirs.has(id)) {
			const next = new Set(expandedDirs);
			next.delete(id);
			expandedDirs = next;
			elements = rollupToElements(graphState.response, next);
			refusalMessage = undefined;

			// Collapsing a directory removes every file node directly
			// inside it from the rendered element set — rollupToElements
			// replaces them with the single collapsed-directory node, and
			// GraphCanvas's replace() correctly drops any of that file's
			// own symbol children too, since a symbol's parent id no
			// longer exists in the same element batch. But expandedFiles/
			// fileSymbolsCache are THIS route's own, entirely separate
			// bookkeeping (see the module doc comment above) — nothing
			// else reconciles them when a file's own directory, rather
			// than an unrelated one, is what just collapsed. Left alone,
			// a file whose directory just collapsed keeps claiming
			// "expanded" (its collapse-affordance button stays visible)
			// while rendering zero symbols, and re-expanding the
			// directory later does not bring them back on its own — the
			// state and the rendered graph diverge, reached through the
			// file's own directory instead of an unrelated one. Dropping
			// it from expandedFiles here (fileSymbolsCache is left
			// untouched) means a later tap on the file, once its
			// directory is expanded again, correctly re-applies the
			// still-cached response rather than issuing a new request —
			// the once-per-file contract is preserved, it just requires
			// one further tap to reactivate after a same-directory
			// round trip, exactly as toggleFile's own re-expand-from-
			// cache branch already handles for every other case.
			const filesToCollapse = [...expandedFiles].filter((file) => dirOf(file) === id);
			if (filesToCollapse.length > 0) {
				const nextFiles = new Set(expandedFiles);
				for (const file of filesToCollapse) nextFiles.delete(file);
				expandedFiles = nextFiles;
			}
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
		elements = rollupToElements(graphState.response, next);
	}

	// toggleFile is the file-to-symbol expand/collapse/re-expand-from-
	// cache dispatcher. Shared by the canvas tap handler above AND the
	// explicit collapse-affordance buttons rendered below — one expansion
	// rule, two ways to reach it, exactly mirroring toggleDirectory's own
	// split.
	//
	// Three cases, in the order the three-click contract below tests
	// them:
	//   1. Already expanded -> COLLAPSE. Remove its symbol children (ids
	//      built from whatever is cached, so a still-in-flight file with
	//      nothing cached yet removes nothing — there is nothing to
	//      remove) and drop it from expandedFiles. Issues NO request.
	//   2. Not expanded, but its symbols are already cached -> RE-EXPAND
	//      FROM CACHE. Apply the cached response's elements again. Issues
	//      NO request.
	//   3. Not expanded and not cached -> FETCH. Add to expandedFiles
	//      OPTIMISTICALLY (before the request settles) so a second click
	//      arriving before it resolves is recognised as a collapse, not a
	//      second expand — this is what keeps the cumulative request
	//      count at exactly 1 even under the stale-response races below.
	//      Also consults pendingFileSymbols FIRST: expandedFiles alone
	//      cannot tell "no request outstanding" apart from "a request for
	//      this exact path is already in flight," since case 1's collapse
	//      branch clears expandedFiles regardless of whether anything is
	//      still pending — a THIRD click (re-expand, arriving before the
	//      first request settles) would otherwise fall through to a
	//      second, duplicate fetch for the same path.
	function toggleFile(id: string) {
		if (graphState.kind !== 'loaded') return;

		if (expandedFiles.has(id)) {
			const cached = fileSymbolsCache.get(id);
			removedElementIds = cached ? symbolElementIdsForFile(id, cached) : [];
			const next = new Set(expandedFiles);
			next.delete(id);
			expandedFiles = next;
			return;
		}

		const cached = fileSymbolsCache.get(id);
		if (cached) {
			addedElements = symbolElementsForFile(id, cached);
			const next = new Set(expandedFiles);
			next.add(id);
			expandedFiles = next;
			return;
		}

		const next = new Set(expandedFiles);
		next.add(id);
		expandedFiles = next;

		if (pendingFileSymbols.has(id)) {
			// A request for this exact path is already in flight — this is
			// a re-expand (collapse, then expand again) arriving before
			// that earlier request settled. expandedFiles now reflects the
			// wanted "expanded" state (set above); the in-flight request's
			// own .then, guarded by expandedFiles.has(id) below, applies it
			// when it resolves. Dispatching a SECOND request here would
			// double the once-per-file contract this function's own header
			// comment documents, and — since a symbol element's id is a
			// pure function of file path + wire symbol id
			// (symbolElementId, file-graph-transform.ts) — two responses
			// for the same file would each try to add the SAME element ids,
			// which the renderer cannot accept twice.
			return;
		}
		pendingFileSymbols.add(id);

		uiClient
			.fileSymbols({ path: id })
			.then((response) => {
				pendingFileSymbols.delete(id);
				// The guard is at the point of APPLICATION, not at the
				// point of dispatch — the request itself is never
				// cancelled. A response that is no longer wanted (its file
				// was collapsed by a second click while
				// this was in flight, or the whole route unmounted) is
				// still CACHED — never discarded — so a later re-expand of
				// the same path is served from it with zero further
				// requests, keeping the once-per-file contract and this
				// guard from trading off against each other.
				const cache = new Map(fileSymbolsCache);
				cache.set(id, response);
				fileSymbolsCache = cache;

				if (!mounted || !expandedFiles.has(id)) {
					return;
				}
				const failures = new Map(fileExpansionFailures);
				failures.delete(id);
				fileExpansionFailures = failures;
				addedElements = symbolElementsForFile(id, response);
			})
			.catch((err: unknown) => {
				pendingFileSymbols.delete(id);
				if (!mounted) return;
				const stillWanted = new Set(expandedFiles);
				stillWanted.delete(id);
				expandedFiles = stillWanted;
				const failures = new Map(fileExpansionFailures);
				failures.set(id, classifyRpcError(err));
				fileExpansionFailures = failures;
			});
	}

	// One vocabulary for a failed rpc (D-04): classifyRpcError's four
	// kinds, each with a plain sentence — never a second error taxonomy
	// invented for this route. Shared between the whole-graph load failure
	// above and a per-file expansion failure below.
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

	// expandedItems is the shared read of expandedDirs+expandedFiles the
	// collapse-affordance list below iterates — a real DOM control for
	// each currently-expanded directory or file: re-collapsing a compound
	// via a real mouse click on the canvas alone is not reliably
	// hit-testable at this renderer's typical density (a
	// measured sub-2px margin around a compound's own border once it has
	// children); this list is a genuinely hit-testable alternative that
	// serves both levels through the SAME toggleDirectory/toggleFile
	// functions the canvas tap handler above already calls, so there is
	// one collapse rule, reached two ways, never a second one invented for
	// this control.
	let expandedItems = $derived.by(() => {
		const dirs = [...expandedDirs].sort().map((id) => ({ id, kind: 'directory' as const }));
		const files = [...expandedFiles].sort().map((id) => ({ id, kind: 'file' as const }));
		return [...dirs, ...files];
	});
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
		{nodeCount} nodes, {edgeCount} edges. Select a directory to reveal the files inside it, or a file
		to reveal the symbols it declares; select it again to collapse it back.
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
			{addedElements}
			{removedElementIds}
			{liveElements}
			onNodeSelected={handleNodeSelected}
			onEdgeSelected={handleEdgeSelected}
			onBackgroundTapped={handleBackgroundTapped}
		/>
	</div>
	{#if expandedItems.length > 0}
		<!--
			The explicit collapse affordance: re-collapsing a compound by
			clicking it a second time on the canvas is not reliably
			hit-testable at this renderer's typical density, so
			every currently-expanded directory or file also gets a genuine
			DOM button here — clicking it calls the SAME toggleDirectory/
			toggleFile the canvas tap handler calls, so this is a second way
			to reach one collapse rule, not a second rule.
		-->
		<div class="mt-2 flex flex-wrap items-center gap-2" data-testid="graph-expanded-list">
			<span class="text-xs text-muted-foreground">Expanded — click to collapse:</span>
			{#each expandedItems as item (item.id)}
				<button
					type="button"
					class="inline-flex items-center gap-1 rounded border border-border px-2 py-1 text-xs text-foreground hover:bg-accent"
					data-testid={`graph-collapse-${item.kind}-${item.id}`}
					onclick={() => (item.kind === 'directory' ? toggleDirectory(item.id) : toggleFile(item.id))}
				>
					<span aria-hidden="true">▾</span>
					{item.id}
				</button>
			{/each}
		</div>
	{/if}
	{#each expandedFiles as file (file)}
		{#if fileSymbolsCache.get(file)?.truncated}
			<p class="mt-1 text-xs text-muted-foreground" data-testid={`graph-file-truncated-${file}`}>
				{file}: showing {fileSymbolsCache.get(file)?.symbols.length} of {fileSymbolsCache.get(
					file
				)?.totalCount} symbols — the list was truncated.
			</p>
		{/if}
	{/each}
	{#each [...fileExpansionFailures.entries()] as [file, failure] (file)}
		<p
			class="mt-1 text-xs text-destructive"
			data-testid={`graph-file-failure-${failure.kind}`}
		>
			{file}: {failureMessage(failure)}
		</p>
	{/each}
	{#if selectedEdge}
		<div class="mt-4 rounded-md border border-border p-4" data-testid="graph-edge-detail">
			<p class="text-sm">
				<span data-testid="graph-edge-detail-source">{selectedEdge.source}</span>
				<span aria-hidden="true">→</span>
				<span data-testid="graph-edge-detail-target">{selectedEdge.target}</span>
			</p>
			<p class="mt-1 text-xs text-muted-foreground" data-testid="graph-edge-detail-total">
				Total dependency count: {selectedEdge.data.totalCount}
			</p>
			<div class="mt-2">
				<DataTable
					rows={edgeKindRows}
					columns={edgeKindColumns}
					getRowId={edgeKindRowId}
					emptyMessage="No edge kinds recorded"
				/>
			</div>
		</div>
	{/if}
{/if}
