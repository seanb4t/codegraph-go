<script lang="ts">
	// StatusBanner renders the ONE index-health signal every view
	// inherits (D-04), mounted once at layout level. The ok verdict
	// renders nothing at all — a banner that is always present is a
	// banner people stop reading. Each degraded verdict names both the
	// condition and what to do about it, in its own wording, so the
	// three degraded texts are never interchangeable. The 'unknown'
	// verdict (a rejected GetStatus call, or the transient state before
	// the first fetch resolves) also renders nothing: it is not one of
	// NAV-04's three named degrade states, and treating "we do not know
	// yet" as an alarming banner on every page load would be worse than
	// silence.
	import type { IndexStatus } from '$lib/status';

	let { status }: { status: IndexStatus } = $props();
</script>

{#if status.verdict === 'stale'}
	<div
		class="border-b border-amber-300 bg-amber-50 px-4 py-2 text-sm text-amber-900"
		role="status"
		data-testid="status-banner-stale"
	>
		The index is stale — it may not reflect recent changes. Run <code>codegraph index</code> to refresh
		it.
	</div>
{:else if status.verdict === 'no-index'}
	<div
		class="border-b border-destructive/30 bg-destructive/10 px-4 py-2 text-sm text-destructive"
		role="status"
		data-testid="status-banner-no-index"
	>
		No index was found for this repository. Run <code>codegraph init</code> to create one.
	</div>
{:else if status.verdict === 'indexing'}
	<div
		class="border-b border-blue-300 bg-blue-50 px-4 py-2 text-sm text-blue-900"
		role="status"
		data-testid="status-banner-indexing"
	>
		Indexing is in progress. Results may be incomplete until it finishes.
	</div>
{/if}
