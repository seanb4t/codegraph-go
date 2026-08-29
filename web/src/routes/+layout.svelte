<script lang="ts">
	// D-17/T-02-05: import app.css here so the shadcn-svelte token set and
	// the Tailwind v4 @import actually reach the rendered page — an
	// app.css that is generated but never imported produces a build with
	// no CSS output at all.
	import '../app.css';
	import favicon from '$lib/assets/favicon.svg';
	import { page } from '$app/state';
	import { setContext } from 'svelte';
	import { uiClient } from '$lib/client';
	import { createStatusGate, navigationIdentity, type IndexStatus } from '$lib/status';
	import StatusBanner from '$lib/components/StatusBanner.svelte';

	let { children } = $props();

	// D-04/D-05: the ONE status gate for the whole app, created once
	// with this page's identity at construction time — the constructor
	// takes that value directly (03-09 Task 1) so its own fetch can
	// record the identity it fetched for, rather than being left to
	// infer it.
	const statusGate = createStatusGate(uiClient, navigationIdentity(page.url));

	// Descendant routes needing the same index-health facts (e.g. the
	// Browse view's stale-source-message split, D-03) subscribe to this
	// SAME gate via context rather than creating a second one — a
	// subscription only registers a listener, it never triggers a
	// fetch.
	setContext('statusGate', statusGate);

	let status = $state<IndexStatus>({ verdict: 'unknown', commit: 'unknown' });
	$effect(() => {
		return statusGate.subscribe((s) => {
			status = s;
		});
	});

	// The gate's ONE navigation trigger besides its own construction
	// above (D-04 Task 1/Task 2 split): reads the SAME reactive page URL
	// this component already derives its active-navigation highlighting
	// from, through the SAME exported normalizer used at construction —
	// two separate derivations that disagreed would make this effect's
	// first run look like a distinct navigation and bring the
	// double-fetch back by a different route. Nothing else in this app
	// calls the gate's navigation method; a second caller (e.g. a
	// beforeNavigate/afterNavigate hook) is exactly how that defect
	// would return.
	$effect(() => {
		statusGate.notifyNavigated(navigationIdentity(page.url));
	});

	// D-18: four navigation slots named for the four future views. Each
	// resolves to a REAL client-side route (Task 2's four placeholder
	// pages) — dead anchors would make ROADMAP criterion 2's SPA-fallback
	// claim untestable. Active-entry highlighting uses SvelteKit's own
	// page state ($app/state), never hand-tracked navigation state.
	const navEntries = [
		{ href: '/browse', label: 'Browse' },
		{ href: '/workbench', label: 'Workbench' },
		{ href: '/graph', label: 'Graph' },
		{ href: '/health', label: 'Health' }
	];

	function isActive(href: string): boolean {
		return page.url.pathname === href || page.url.pathname.startsWith(href + '/');
	}
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

<div class="flex min-h-screen flex-col bg-background text-foreground">
	<header class="border-b border-border">
		<nav class="flex items-center gap-1 px-4 py-3">
			<a href="/" class="mr-4 text-sm font-semibold">codegraph</a>
			{#each navEntries as entry (entry.href)}
				<a
					href={entry.href}
					aria-current={isActive(entry.href) ? 'page' : undefined}
					class="rounded-md px-3 py-1.5 text-sm transition-colors {isActive(entry.href)
						? 'bg-primary text-primary-foreground'
						: 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'}"
				>
					{entry.label}
				</a>
			{/each}
		</nav>
	</header>
	<StatusBanner {status} />
	<main class="flex-1 px-4 py-6">
		{@render children()}
	</main>
</div>
