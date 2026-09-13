<script lang="ts">
	// CoverageSection is /health's Coverage section (Phase 10, D-11): the
	// discovered/indexed/excluded/extraction-failed denominator, grouped
	// by reason with counts, each group expanding to per-file rows, and
	// a first-class "Coverage unknown" state for a pre-Phase-10 graph
	// (D-06's never-0/0 rule).
	//
	// This component renders exactly what its two props already say —
	// `view` (health-view.ts's toCoverageView projection) and `rows`
	// (the page-walked, grouped result) — and computes NEITHER a verdict
	// nor a count of its own (D-04's discipline, one level up).
	//
	// SRV-03 / T-10-06: the remedy is TEXT ONLY. This component contains
	// no raw-HTML directive, no interactive control element, and no
	// event handler of any kind — a "re-index this file" action on this
	// view would be an elevation-of-privilege surface this file must
	// never grow. Every path/detail string below is Svelte text
	// interpolation only (T-10-01): a value containing markup renders as
	// literal text and creates no element.
	import type { CoverageGroup, CoverageView } from '$lib/health-view';

	export type RowsState =
		| { kind: 'idle' }
		| { kind: 'loading' }
		| { kind: 'loaded'; groups: CoverageGroup[] }
		| { kind: 'failed'; message: string };

	let { view, rows }: { view: CoverageView; rows: RowsState } = $props();

	// The two directory-level reasons (D-01/D-02) get a distinct suffix
	// on their rows: their "path" names a pruned directory whose
	// contents were never discovered at all, not a single excluded file.
	const DIRECTORY_GROUP_KEYS = new Set(['EXCLUSION_REASON_DIR_VENDOR', 'EXCLUSION_REASON_DIR_DOTPREFIX']);

	// countsLine/prunesLine are computed here, as single strings, rather
	// than left as multi-expression markup: a template wrapped across
	// lines for readability would otherwise interpolate the source's own
	// literal newline/indentation into the rendered text (caught by
	// health-page.test.ts's exact-text assertion on the counts line).
	let countsLine = $derived(
		view.known
			? `${view.discovered} discovered · ${view.indexed} indexed · ${view.excluded} excluded · ${view.extractionFailed} extraction failures`
			: ''
	);
	let prunesLine = $derived(
		view.known && view.directoryPrunes > 0
			? `${view.directoryPrunes} ${view.directoryPrunes === 1 ? 'pruned directory' : 'pruned directories'} — their contents were never discovered.`
			: ''
	);
</script>

<section data-testid="health-coverage" class="rounded-md border border-border p-3 text-sm">
	<h2 class="text-sm font-semibold">Coverage</h2>

	{#if !view.known}
		<p data-testid="health-coverage-unknown" class="mt-1 text-muted-foreground">
			Coverage unknown — re-index to record it. Run <code>codegraph index</code> against this repository to
			record which files were discovered, indexed, or excluded and why.
		</p>
	{:else}
		<p data-testid="health-coverage-counts" class="mt-1">{countsLine}</p>

		{#if view.directoryPrunes > 0}
			<p data-testid="health-coverage-prunes" class="mt-1 text-muted-foreground">{prunesLine}</p>
		{/if}

		{#if rows.kind === 'loading'}
			<p data-testid="health-coverage-rows-loading" class="mt-2 text-muted-foreground">
				Loading coverage rows…
			</p>
		{:else if rows.kind === 'failed'}
			<p data-testid="health-coverage-rows-failed" class="mt-2 text-destructive">
				Could not load the per-file coverage rows: {rows.message}
			</p>
		{:else if rows.kind === 'loaded'}
			<div class="mt-2 flex flex-col gap-1">
				{#each rows.groups as group (group.key)}
					<details data-testid={'health-coverage-group-' + group.key}>
						<summary>{group.label} ({group.count})</summary>
						<ul class="ml-4 list-disc">
							{#each group.rows as row, i (row.path + '#' + i)}
								{#if group.key === 'EXTRACTION_FAILED'}
									<li data-testid="health-coverage-row-failed" class="text-destructive">
										extraction failed: {row.detail} (<code>{row.path}</code>)
									</li>
								{:else}
									<li data-testid="health-coverage-row">
										<code>{row.path}</code>{DIRECTORY_GROUP_KEYS.has(group.key)
											? ' (directory excluded)'
											: ''}
										<span>{row.detail}</span>
									</li>
								{/if}
							{/each}
						</ul>
					</details>
				{/each}
			</div>
		{/if}
	{/if}
</section>
