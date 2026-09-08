// callees-columns.ts is the Callees analysis's ColumnDef set (D-06) —
// the third of four near-identical *-columns.ts files (04-01's
// callers-columns.ts is the first; impact-columns.ts, in this same
// plan, is the second) over Location's four wire fields (name, kind,
// file_path, start_line — generated camelCase of ui.proto:100-105).
import type { ColumnDef } from '@tanstack/svelte-table';
import type { Location } from '$lib/gen/ui_pb';
import { features } from './table-features';

export const calleesColumns: ColumnDef<typeof features, Location>[] = [
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
