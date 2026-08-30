<script lang="ts">
	// D-18: fills the Phase 4 placeholder this route mounted (04-05).
	// Reads the trust verdict from the SAME shared status gate every
	// other route subscribes to (D-04) — this route creates no second
	// gate and issues no second GetStatus call (T-04-17). Every raw
	// number on this page comes from exactly ONE GetHealth call, issued
	// once when the view opens, with an AbortController tied to
	// component teardown. No timer, no polling, no refetch — Phase 3
	// D-05's no-polling contract.
	//
	// Document order (HLT-02/HLT-03, asserted by web/tests/health-page.
	// test.ts): the worktree warning (when present), then the trust
	// verdict, THEN the numeric blocks — never the reverse.
	import { getContext } from 'svelte';
	import { uiClient } from '$lib/client';
	import type { IndexStatus, StatusGate } from '$lib/status';
	import type { GetHealthResponse } from '$lib/gen/ui_pb';
	import { toCountRows, hasWorktreeMismatch, describeFreshness } from '$lib/health-view';
	import { describeWorkbenchFailure } from '$lib/workbench-failure';
	import TrustVerdict from '$lib/components/health/TrustVerdict.svelte';
	import WorktreeMismatchWarning from '$lib/components/health/WorktreeMismatchWarning.svelte';
	import CountTable from '$lib/components/health/CountTable.svelte';

	const statusGate = getContext<StatusGate>('statusGate');
	let indexStatus: IndexStatus = $state({ verdict: 'unknown', commit: 'unknown', commitSha: '' });
	$effect(() => {
		return statusGate.subscribe((s) => {
			indexStatus = s;
		});
	});

	type PageState =
		| { kind: 'loading' }
		| { kind: 'loaded'; response: GetHealthResponse }
		| { kind: 'failed'; failure: ReturnType<typeof describeWorkbenchFailure> };

	let pageState: PageState = $state({ kind: 'loading' });

	// This effect's synchronous body reads no reactive state — it fires
	// exactly once, on mount, and never re-runs (mirroring AnalysisPanel.
	// svelte's own untracked-read discipline for the same reason:
	// WRK-04's "moving a control issues no extra GetStatus" property,
	// here specialized to "opening this view issues exactly one
	// GetHealth"). indexStatus is read only inside the .catch() callback,
	// at call time, which establishes no tracked dependency.
	$effect(() => {
		const controller = new AbortController();
		uiClient
			.getHealth({}, { signal: controller.signal })
			.then((response) => {
				if (controller.signal.aborted) return;
				pageState = { kind: 'loaded', response };
			})
			.catch((err: unknown) => {
				if (controller.signal.aborted) return;
				pageState = { kind: 'failed', failure: describeWorkbenchFailure(err, indexStatus) };
			});
		return () => controller.abort();
	});
</script>

<div class="flex flex-col gap-4">
	{#if pageState.kind === 'loaded' && hasWorktreeMismatch(pageState.response)}
		<WorktreeMismatchWarning
			worktreeRoot={pageState.response.worktreeMismatch?.worktreeRoot ?? ''}
			indexRoot={pageState.response.worktreeMismatch?.indexRoot ?? ''}
		/>
	{/if}

	<TrustVerdict status={indexStatus} />

	{#if pageState.kind === 'failed'}
		<div
			data-testid={`workbench-failure-${pageState.failure.kind}`}
			class="rounded-md border border-destructive/40 bg-destructive/10 p-3 text-sm text-destructive"
		>
			<p class="font-semibold">{pageState.failure.title}</p>
			<p>{pageState.failure.detail}</p>
		</div>
	{:else if pageState.kind === 'loading'}
		<p class="text-sm text-muted-foreground" data-testid="health-loading">Loading index health…</p>
	{:else}
		{@const freshness = describeFreshness(pageState.response, indexStatus)}
		<div class="rounded-md border border-border p-3 text-sm" data-testid="health-freshness">
			<p>
				Commit: <code data-testid="health-commit-sha">{freshness.commitSha || 'unknown'}</code>
			</p>
			<p>
				Schema version: <span data-testid="health-schema-version">{freshness.schemaVersion}</span>
			</p>
			<p>Re-index recommended: {freshness.reindexRecommended ? 'yes' : 'no'}</p>
			{#if freshness.snapshotAgreement === 'differs'}
				<p
					class="mt-2 font-semibold text-amber-700"
					role="status"
					data-testid="health-snapshot-differs"
				>
					These numbers were read at a different indexed commit than the verdict above and may be
					superseded. The verdict above is never recomputed from them.
				</p>
			{/if}
		</div>

		<CountTable
			title="Files by language"
			rows={toCountRows(pageState.response.filesByLanguage)}
			keyHeader="Language"
		/>
		<CountTable
			title="Nodes by kind"
			rows={toCountRows(pageState.response.nodesByKind)}
			keyHeader="Kind"
		/>
		<CountTable
			title="Edges by kind"
			rows={toCountRows(pageState.response.edgesByKind)}
			keyHeader="Kind"
		/>
	{/if}
</div>
