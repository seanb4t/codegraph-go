<script lang="ts">
	// D-18: fills the Phase 2 placeholder this route mounted. Reads view
	// state EXCLUSIVELY from page.url.searchParams (via $app/state) — never
	// a `load` function (this app is ssr=false/prerender=false, and the
	// RPC calls are already client-only) — so back/forward and a fresh load
	// of the same URL derive identical state (NAV-01/NAV-02). This route
	// never imports SvelteKit's shallow-routing history exports from
	// $app/navigation for view state: those only ever assign to
	// page.state, never page.url — a component reading page.url would not
	// react to them (03-RESEARCH.md Pitfall 1). URL WRITES (goto-driven
	// navigation, search-as-you-type) land in plan 03-07; this plan is
	// read-only against the URL.
	import { page } from '$app/state';
	import { uiClient } from '$lib/client';
	import { parseBrowseParams, type BrowseParams } from '$lib/browse-url';
	import { loadBrowseTarget, type BrowseTargetState } from '$lib/browse-state';
	import SourcePane from '$lib/components/browse/SourcePane.svelte';
	import SearchPanel, { type SearchSelection } from '$lib/components/browse/SearchPanel.svelte';

	let params = $derived(parseBrowseParams(page.url.searchParams));
	let state = $state<BrowseTargetState>({ kind: 'idle' });

	$effect(() => {
		const currentParams = params;
		const controller = new AbortController();

		if (!currentParams.symbol && !currentParams.file) {
			state = { kind: 'idle' };
			return;
		}

		state = { kind: 'loading' };
		loadBrowseTarget(currentParams, uiClient, controller.signal).then((result) => {
			if (!controller.signal.aborted) {
				state = result;
			}
		});

		return () => controller.abort();
	});

	// PLACEHOLDER-03-07
	//
	// Selecting a search result calls back into this page; the actual URL
	// write (a goto()-driven push, per D-11) lands in plan 03-07. For
	// this one wave, selecting sets the page's target state DIRECTLY —
	// bypassing `params`/the URL entirely — a deliberate, named, one-wave
	// deviation from the URL-as-source-of-truth contract (03-06's own
	// plan text records why: bringing 03-07's browse-nav.ts forward here
	// would put five tasks and two subsystems in one plan). 03-07 Task 3
	// replaces this function's body with a goto() call and removes the
	// marker above; its own acceptance criterion asserts that exact
	// marker's count in web/src/ has returned to zero afterward.
	async function handleSearchSelect(selection: SearchSelection): Promise<void> {
		const target: BrowseParams =
			selection.kind === 'symbol'
				? {
						symbol: selection.location.name,
						file: selection.location.filePath,
						line: selection.location.startLine,
						unknown: []
					}
				: selection.kind === 'file'
					? { file: selection.entry.path, unknown: [] }
					: { file: selection.path, unknown: [] };

		state = { kind: 'loading' };
		state = await loadBrowseTarget(target, uiClient);
	}
</script>

<h1 class="text-lg font-semibold">Browse</h1>

<SearchPanel client={uiClient} initialQuery={params.q ?? ''} onSelect={handleSearchSelect} />

<SourcePane {state} />
