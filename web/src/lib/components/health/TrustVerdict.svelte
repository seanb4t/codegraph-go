<script lang="ts">
	// TrustVerdict is /health's own verdict banner (HLT-02). It renders
	// the SAME five-member StatusVerdict StatusBanner.svelte renders
	// application-wide (D-04) — no second verdict function exists, and
	// this component computes nothing: `status` arrives already
	// classified. Unlike StatusBanner, which renders nothing for the
	// 'ok' and 'unknown' verdicts (a banner that is always present is a
	// banner people stop reading), this component renders a branch for
	// ALL FIVE verdicts including 'ok': the health page's whole job is
	// to answer "should I trust this" affirmatively too, not only to
	// warn.
	import type { IndexStatus } from '$lib/status';

	let { status }: { status: IndexStatus } = $props();
</script>

{#if status.verdict === 'ok'}
	<div
		class="rounded-md border border-emerald-300 bg-emerald-50 px-4 py-2 text-sm text-emerald-900"
		role="status"
		data-testid="health-verdict-ok"
	>
		The index is current. The numbers below reflect the working tree.
	</div>
{:else if status.verdict === 'stale'}
	<div
		class="rounded-md border border-amber-300 bg-amber-50 px-4 py-2 text-sm text-amber-900"
		role="status"
		data-testid="health-verdict-stale"
	>
		The index is stale — it may not reflect recent changes. Run <code>codegraph index</code> to refresh
		it before trusting the numbers below.
	</div>
{:else if status.verdict === 'no-index'}
	<div
		class="rounded-md border border-destructive/30 bg-destructive/10 px-4 py-2 text-sm text-destructive"
		role="status"
		data-testid="health-verdict-no-index"
	>
		No index was found for this repository — there is nothing to trust yet. Run <code
			>codegraph init</code
		> to create one.
	</div>
{:else if status.verdict === 'indexing'}
	<div
		class="rounded-md border border-blue-300 bg-blue-50 px-4 py-2 text-sm text-blue-900"
		role="status"
		data-testid="health-verdict-indexing"
	>
		Indexing is in progress. The numbers below may be incomplete until it finishes.
	</div>
{:else}
	<div
		class="rounded-md border border-border bg-muted px-4 py-2 text-sm text-muted-foreground"
		role="status"
		data-testid="health-verdict-unknown"
	>
		The index's health could not be determined.
	</div>
{/if}
