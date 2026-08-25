<script lang="ts">
	// D-17/T-02-05: import app.css here so the shadcn-svelte token set and
	// the Tailwind v4 @import actually reach the rendered page — an
	// app.css that is generated but never imported produces a build with
	// no CSS output at all.
	import '../app.css';
	import favicon from '$lib/assets/favicon.svg';
	import { page } from '$app/state';

	let { children } = $props();

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
	<main class="flex-1 px-4 py-6">
		{@render children()}
	</main>
</div>
