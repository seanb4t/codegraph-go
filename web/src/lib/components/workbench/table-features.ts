// table-features.ts is the ONE place sorting is enabled for every
// Workbench analysis table (D-06). `@tanstack/svelte-table@9.2.4` bundles
// nothing by default — a table instance created without an explicit
// `tableFeatures({...})` registration has no sorting at all, no matter
// what `sortFn` a column definition names. This registration is written
// once here and imported by DataTable.svelte, so all four analyses (this
// plan's Callers table and 04-04's Impact/Affected/Callees tables) share
// the exact same sorting behavior rather than four independent copies
// that could silently drift apart.
//
// Verbatim source (04-RESEARCH.md Pattern 2, fetched from
// github.com/TanStack/table main branch,
// examples/svelte/sorting/src/tableHelper.svelte.ts, confirmed to match
// the installed 9.2.4 package) — NOT the store-based API shown on the
// public docs site, which is stale for this version (Pitfall 3).
import {
	createSortedRowModel,
	rowSortingFeature,
	sortFn_alphanumeric,
	sortFn_text,
	tableFeatures
} from '@tanstack/svelte-table';

export const features = tableFeatures({
	rowSortingFeature,
	sortedRowModel: createSortedRowModel(),
	sortFns: {
		alphanumeric: sortFn_alphanumeric,
		text: sortFn_text
	}
});
