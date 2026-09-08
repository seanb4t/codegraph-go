<script lang="ts">
	// data-table-location-host.svelte — a thin, concretely-typed wrapper
	// fixing DataTable's generic TRow to Location, so
	// data-table-render-cost.test.ts can render it from a plain .ts test
	// file with full type-checking intact.
	//
	// WHY THIS EXISTS: DataTable.svelte declares `generics="TRow extends
	// RowData"` (D-06 — ONE shared shell for every analysis row type).
	// Svelte's own tooling resolves that generic from JSX-like usage inside
	// another .svelte file (exactly how AnalysisPanel.svelte uses it,
	// `<DataTable rows={...} {columns} .../>`), but
	// @testing-library/svelte's `render()` infers its type parameter from
	// the IMPORTED COMPONENT VALUE alone — there is no concrete usage site
	// for it to read TRow off of, so it falls back to the generic's bound
	// (RowData), and any concretely-Location-typed props (columns,
	// getRowId) then fail to type-check against that fallback. This host
	// component IS that concrete usage site: Location flows in as a prop,
	// TRow is fixed to Location once, and the render-cost test never needs
	// its own type-check workaround.
	import type { ColumnDef } from '@tanstack/svelte-table';
	import type { Location } from '$lib/gen/ui_pb';
	import DataTable from '$lib/components/workbench/DataTable.svelte';
	import { features } from '$lib/components/workbench/table-features';

	let {
		rows,
		columns,
		getRowId,
		emptyMessage,
		onSelect
	}: {
		rows: Location[];
		columns: ColumnDef<typeof features, Location>[];
		getRowId: (row: Location) => string;
		emptyMessage: string;
		onSelect?: (row: Location) => void;
	} = $props();
</script>

<DataTable {rows} {columns} {getRowId} {emptyMessage} {onSelect} />
