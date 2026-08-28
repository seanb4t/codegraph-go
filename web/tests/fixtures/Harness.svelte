<!--
	03-01 (BRW-06): minimal Svelte 5 runes fixture proving the vitest
	harness actually compiles and renders a REAL .svelte component —
	@testing-library/svelte's `render` takes a compiled component, and a
	.ts file has no expression form for Svelte markup, so this is the
	only shape in which the RED->GREEN proof is performable.
-->
<script lang="ts">
	let { label }: { label: string } = $props();
</script>

<span data-testid="harness-label">{label}</span>
