// callers-columns.ts is the Callers analysis's ColumnDef set (D-06) — the
// first of four near-identical *-columns.ts files (04-04 adds the
// remaining three) over Location's four wire fields (name, kind,
// file_path, start_line — generated camelCase of ui.proto:100-105).
//
// locationRowId reuses the SAME `${filePath}:${startLine}:${name}`
// composite identity NeighborsPanel.svelte's entryKey already uses, so
// every analysis passes one shared identity function into the generic
// DataTable shell instead of four independent copies of the template
// literal.
import type { ColumnDef } from '@tanstack/svelte-table';
import type { Location } from '$lib/gen/ui_pb';
import { features } from './table-features';

export const locationRowId = (row: Location): string =>
	`${row.filePath}:${row.startLine}:${row.name}`;

export const callersColumns: ColumnDef<typeof features, Location>[] = [
	{
		id: 'name',
		accessorKey: 'name',
		header: 'Name',
		sortFn: 'alphanumeric'
	},
	{
		id: 'kind',
		accessorKey: 'kind',
		header: 'Kind',
		sortFn: 'alphanumeric'
	},
	{
		id: 'filePath',
		accessorKey: 'filePath',
		header: 'File',
		sortFn: 'alphanumeric'
	},
	{
		// No explicit sortFn: numeric auto-detection is correct for a
		// number-valued column (04-RESEARCH.md Pattern 2).
		id: 'startLine',
		accessorKey: 'startLine',
		header: 'Line'
	}
];
