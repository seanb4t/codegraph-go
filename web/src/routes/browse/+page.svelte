<script lang="ts">
	// D-18: fills the Phase 2 placeholder this route mounted. Reads view
	// state EXCLUSIVELY from page.url.searchParams (via $app/state) — never
	// a `load` function (this app is ssr=false/prerender=false, and the
	// RPC calls are already client-only) — so back/forward and a fresh load
	// of the same URL derive identical state (NAV-01/NAV-02). This route
	// never imports SvelteKit's shallow-routing history exports from
	// $app/navigation for view state: those only ever assign to
	// page.state, never page.url — a component reading page.url would not
	// react to them (03-RESEARCH.md Pitfall 1). URL WRITES go exclusively
	// through browse-nav.ts's navigator, built on goto() — never a direct
	// state assignment (03-06's own placeholder, replaced below).
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { uiClient } from '$lib/client';
	import { parseBrowseParams, type BrowseParams } from '$lib/browse-url';
	import {
		loadBrowseTarget,
		loadBlastRadius,
		createNavigationGate,
		type BrowseTargetState,
		type BlastRadiusState
	} from '$lib/browse-state';
	import {
		createBrowseNavigator,
		NAV_INTENT,
		type BrowseNavDelta,
		type NavIntent
	} from '$lib/browse-nav';
	import SourcePane from '$lib/components/browse/SourcePane.svelte';
	import NeighborsPanel from '$lib/components/browse/NeighborsPanel.svelte';
	import SearchPanel, { type SearchSelection } from '$lib/components/browse/SearchPanel.svelte';

	let params = $derived(parseBrowseParams(page.url.searchParams));
	let targetState = $state<BrowseTargetState>({ kind: 'idle' });
	let blastState = $state<BlastRadiusState>({ kind: 'idle' });

	// One navigator (built once, over the real goto), one gate (one
	// NavigationGeneration per URL change, minted here — the route —
	// never by a loader). The gate is what keeps a node-detail load and
	// a blast-radius load started for two DIFFERENT URL states from
	// ever being combined into one rendered view.
	const navigator = createBrowseNavigator(goto);
	const gate = createNavigationGate();

	$effect(() => {
		const currentParams = params;
		const generation = gate.advance();
		const controller = new AbortController();

		if (!currentParams.symbol && !currentParams.file) {
			targetState = { kind: 'idle' };
			blastState = { kind: 'idle' };
			return;
		}

		targetState = { kind: 'loading' };
		blastState = currentParams.symbol ? { kind: 'loading' } : { kind: 'idle' };

		loadBrowseTarget(currentParams, uiClient, controller.signal).then((result) => {
			if (!gate.isCurrent(generation) || controller.signal.aborted) return;
			targetState = result;
		});

		if (currentParams.symbol) {
			loadBlastRadius(currentParams, uiClient, controller.signal).then((result) => {
				if (!gate.isCurrent(generation) || controller.signal.aborted) return;
				blastState = result;
			});
		}

		return () => controller.abort();
	});

	// Every navigation (search selection, neighbour click) and every
	// refinement (blast-radius depth control) goes through the SAME
	// navigator, which is the one place in the client that writes a URL
	// (D-11). This page holds no view state of its own past this call —
	// everything renders from `params`/`state`/`blastState`, which the
	// effect above derives from the URL.
	function handleSearchSelect(selection: SearchSelection): void {
		const delta: BrowseNavDelta =
			selection.kind === 'symbol'
				? {
						symbol: selection.location.name,
						file: selection.location.filePath,
						line: selection.location.startLine
					}
				: selection.kind === 'file'
					? { file: selection.entry.path }
					: { file: selection.path };
		navigator.navigate(page.url, delta, NAV_INTENT.NAVIGATE);
	}

	function handleNeighborNavigate(delta: BrowseNavDelta, intent: NavIntent): void {
		navigator.navigate(page.url, delta, intent);
	}

	// Typing in search is a REFINE intent (D-11) — every keystroke
	// replaces the current history entry so the address bar stays
	// correct and shareable at every instant, without growing history
	// one entry per character. An empty query clears the `q` param
	// entirely rather than leaving `q=` in the URL.
	function handleQueryChange(query: string): void {
		navigator.navigate(page.url, { q: query || undefined }, NAV_INTENT.REFINE);
	}
</script>

<h1 class="text-lg font-semibold">Browse</h1>

<SearchPanel
	client={uiClient}
	initialQuery={params.q ?? ''}
	onSelect={handleSearchSelect}
	onQueryChange={handleQueryChange}
/>

<SourcePane state={targetState} />

{#if targetState.kind === 'single-def'}
	<NeighborsPanel
		calls={targetState.calls}
		calledBy={targetState.calledBy}
		blastRadius={blastState}
		depth={params.depth}
		onNavigate={handleNeighborNavigate}
	/>
{/if}
