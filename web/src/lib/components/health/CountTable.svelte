<script lang="ts">
	// CountTable is a thin wrapper over the ONE generic DataTable shell
	// (D-06, 04-01) instantiated at TRow = CountRow — the health page's
	// three count listings (per-language file counts, node counts by
	// kind, edge counts by kind) each get one instance of this component
	// rather than a second table implementation.
	//
	// getRowId is a PROP on the underlying DataTable for exactly this
	// reason: CountRow is {key, count} and has no filePath/startLine/
	// name, so a shell hard-coded to a Location identity would fail
	// `pnpm check` here.
	import type { ColumnDef } from '@tanstack/svelte-table';
	import DataTable from '$lib/components/workbench/DataTable.svelte';
	import { features } from '$lib/components/workbench/table-features';
	import type { CountRow } from '$lib/health-view';

	let {
		title,
		rows,
		keyHeader
	}: {
		title: string;
		rows: CountRow[];
		keyHeader: string;
	} = $props();

	let columns: ColumnDef<typeof features, CountRow>[] = $derived([
		{ id: 'key', accessorKey: 'key', header: keyHeader, sortFn: 'alphanumeric' },
		{
			// No explicit sortFn: numeric auto-detection is correct for a
			// number-valued column (04-RESEARCH.md Pattern 2, the same
			// convention every *-columns.ts file in workbench/ follows).
			id: 'count',
			accessorKey: 'count',
			header: 'Count'
		}
	]);

	function getRowId(row: CountRow): string {
		return row.key;
	}
</script>

<div>
	<h2 class="text-sm font-semibold">{title}</h2>
	<DataTable {rows} {columns} {getRowId} emptyMessage="No data" />
</div>
