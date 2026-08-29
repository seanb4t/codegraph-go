<script lang="ts" module>
	// SearchSelection is what the panel hands back to its caller when a
	// result is opened (click or Enter-on-highlighted-item) — the panel
	// itself never navigates; the caller decides what "opening" means.
	// 03-07's +page.svelte routes every SearchSelection through
	// browse-nav.ts's navigator (a goto()-driven, NAVIGATE-intent URL
	// write), replacing the direct-state-assignment placeholder 03-06
	// left here for exactly this handover.
	import type { Location, FileEntry } from '$lib/gen/ui_pb';

	export type SearchSelection =
		| { kind: 'symbol'; location: Location }
		| { kind: 'file'; entry: FileEntry }
		// ExploreGroup carries only a path, not a full FileEntry (no
		// language/nodeCount/edgeCount on the wire for it) — a distinct
		// case rather than fabricating fake FileEntry fields.
		| { kind: 'explore-file'; path: string };
</script>

<script lang="ts">
	// SearchPanel composes the vendored Command primitive (Task 1) into
	// BRW-01's search-as-you-type + BRW-08's Explore-on-Enter surface,
	// wired to NAV-03's keyboard drivability (D-17).
	//
	// shouldFilter={false} is load-bearing, confirmed directly against
	// the vendored bits-ui source in Task 1 (command.svelte.d.ts:
	// `shouldFilter: boolean`; command.svelte.js's shouldRender checks
	// `=== false` to bypass its own fuzzy-filter/score computation
	// entirely) rather than trusted from 03-RESEARCH.md's
	// WebSearch-sourced Assumption A1. Results are already
	// server-filtered and server-ordered (D-15); letting the primitive
	// re-filter them against the typed text would hide any match whose
	// display string doesn't literally substring-contain what was typed.
	//
	// Arrow-key traversal, active-descendant wiring, and Enter-selects-
	// the-highlighted-item are the vendored primitive's own — confirmed
	// in Task 1's keyboard spike to already cross Command.Group
	// boundaries in visual order (bits-ui's updateSelectedByItem/
	// getValidItems walk one flat array of visible items regardless of
	// group membership). This component hand-writes nothing for that;
	// it only adds what the primitive does not provide: the focus
	// shortcuts, Escape-to-dismiss, and the "ask a question" trigger.
	//
	// The "ask" trigger. BRW-08's Explore has no keyboard affordance of
	// its own in the vendored primitive (Enter always means
	// "select-the-highlighted-item", which the primitive itself
	// implements as item.click()). Rather than hijacking Enter globally
	// (which would conflict with the ALSO-required "Enter on a
	// highlighted item invokes the open callback with that item"
	// behavior), Explore submission is wired through the SAME
	// mechanism: a single, unlabeled, always-first Command.Item whose
	// onSelect calls submit(). It renders whenever the query is
	// non-empty, and — because the primitive selects the first item by
	// default — a bare "type, then press Enter" reaches it naturally,
	// while arrowing further down still reaches every live result.
	import { untrack } from 'svelte';
	import { get } from 'svelte/store';
	import * as Command from '$lib/components/ui/command/index.js';
	import { createSearchController, type SearchClient, type SearchControllerState } from '$lib/search';

	let {
		client,
		initialQuery = '',
		onSelect,
		onQueryChange
	}: {
		client: SearchClient;
		initialQuery?: string;
		onSelect: (selection: SearchSelection) => void;
		// onQueryChange (03-07, D-11): typing is a REFINE intent — the
		// caller is expected to write it into the URL with `replaceState`
		// so the address bar stays correct and shareable at every
		// instant, without growing history one entry per keystroke.
		// Optional so a caller that only needs the live-search behavior
		// (any pre-03-07 test, or a future non-URL-backed use of this
		// panel) is unaffected.
		onQueryChange?: (query: string) => void;
	} = $props();

	// untrack: the controller is built ONCE from whatever client is
	// current at mount — this component does not support swapping the
	// client for a live instance, so intentionally reading it outside a
	// reactive dependency here is not a bug.
	const controller = untrack(() => createSearchController(client));
	let searchState = $state<SearchControllerState>(get(controller));

	// Bridges the controller's store-shaped state into a Svelte 5 rune —
	// $effect (not a bare subscribe-at-module-scope) so the subscription
	// is torn down on unmount, matching every other RPC-owning module in
	// this phase (+page.svelte's own $effect/AbortController pattern).
	$effect(() => {
		const unsubscribe = controller.subscribe((next) => {
			searchState = next;
		});
		return unsubscribe;
	});

	let open = $state(true);
	let inputRef: HTMLInputElement | null = $state(null);

	// Seed the query once on mount so a shared ?q=... link narrows the
	// box immediately, exactly as if the user had just typed it —
	// deliberately routed through setQuery (the same debounced path
	// typing uses), not a special-cased initial dispatch.
	$effect(() => {
		if (initialQuery) controller.setQuery(initialQuery);
	});

	function isTextEditable(el: Element | null): boolean {
		if (!(el instanceof HTMLElement)) return false;
		const tag = el.tagName;
		if (tag === 'INPUT' || tag === 'TEXTAREA') return true;
		return el.isContentEditable;
	}

	// Document-level, installed on mount / removed on destroy (the panel
	// must be reachable from anywhere in the view, not only while focus
	// already sits inside it) — NAV-03/D-17.
	function handleGlobalKeydown(e: KeyboardEvent) {
		const cmdOrCtrlK = (e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k';
		if (cmdOrCtrlK) {
			e.preventDefault();
			open = true;
			inputRef?.focus();
			return;
		}

		if (e.key === '/') {
			// File paths are full of slashes — never eat a `/` a user is
			// typing into some other field.
			if (isTextEditable(document.activeElement)) return;
			e.preventDefault();
			open = true;
			inputRef?.focus();
			return;
		}

		if (e.key === 'Escape') {
			open = false;
		}
	}

	$effect(() => {
		document.addEventListener('keydown', handleGlobalKeydown);
		return () => document.removeEventListener('keydown', handleGlobalKeydown);
	});

	function handleInputChange(value: string) {
		open = true;
		controller.setQuery(value);
		onQueryChange?.(value);
	}

	function symbolKey(loc: Location): string {
		return `symbol:${loc.name}:${loc.filePath}:${loc.startLine}`;
	}

	function fileKey(entry: FileEntry): string {
		return `file:${entry.path}`;
	}

	function exploreKey(path: string): string {
		return `explore:${path}`;
	}
</script>

<div data-testid="search-panel">
	<Command.Root shouldFilter={false}>
		<Command.Input
			bind:ref={inputRef}
			value={searchState.query}
			oninput={(e: Event) => handleInputChange((e.currentTarget as HTMLInputElement).value)}
			placeholder="Search symbols and files, or press Enter to ask..."
		/>
		{#if open}
			<Command.List>
				<Command.Empty>No results.</Command.Empty>

				{#if searchState.query.trim().length > 0}
					<Command.Item
						value="__ask__"
						data-testid="search-item-ask"
						onSelect={() => controller.submit()}
					>
						Ask &quot;{searchState.query}&quot;
					</Command.Item>
				{/if}

				{#if searchState.live.symbols.length > 0}
					<Command.Group heading="Symbols">
						{#each searchState.live.symbols as loc (symbolKey(loc))}
							<Command.Item
								value={symbolKey(loc)}
								data-testid={`search-item-${symbolKey(loc)}`}
								onSelect={() => onSelect({ kind: 'symbol', location: loc })}
							>
								{loc.name}
								<span class="ml-2 text-muted-foreground">{loc.filePath}:{loc.startLine}</span>
							</Command.Item>
						{/each}
					</Command.Group>
				{/if}

				{#if searchState.live.files.length > 0}
					<Command.Group heading="Files">
						{#each searchState.live.files as entry (fileKey(entry))}
							<Command.Item
								value={fileKey(entry)}
								data-testid={`search-item-${fileKey(entry)}`}
								onSelect={() => onSelect({ kind: 'file', entry })}
							>
								{entry.path}
							</Command.Item>
						{/each}
					</Command.Group>
				{/if}

				{#if searchState.explore}
					<Command.Group heading="Explore">
						{#if searchState.explore.kind === 'results'}
							{#each searchState.explore.groups as group (group.path)}
								<Command.Item
									value={exploreKey(group.path)}
									data-testid={`search-item-${exploreKey(group.path)}`}
									onSelect={() => onSelect({ kind: 'explore-file', path: group.path })}
								>
									{group.path}
									<span class="ml-2 text-muted-foreground">{group.symbols.length} match(es)</span>
								</Command.Item>
							{/each}
						{:else if searchState.explore.kind === 'empty'}
							<div data-testid="explore-empty" class="px-2 py-1.5 text-sm text-muted-foreground">
								No results for &quot;{searchState.explore.query}&quot;.
							</div>
						{:else}
							<div data-testid="explore-failed" class="px-2 py-1.5 text-sm text-destructive">
								Something went wrong: {searchState.explore.failure.message}
							</div>
						{/if}
					</Command.Group>
				{/if}
			</Command.List>
		{/if}
	</Command.Root>
</div>
