<script lang="ts">
	// D-17/T-02-05: import app.css here so the shadcn-svelte token set and
	// the Tailwind v4 @import actually reach the rendered page — an
	// app.css that is generated but never imported produces a build with
	// no CSS output at all.
	import '../app.css';
	import favicon from '$lib/assets/favicon.svg';

	let { children } = $props();
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

{@render children()}
