// impact-columns.ts is the Impact analysis's ColumnDef set (D-06) — the
// second of four near-identical *-columns.ts files (04-01's
// callers-columns.ts is the first; callees-columns.ts, in this same
// plan, is the third) over Location's four wire fields (name, kind,
// file_path, start_line — generated camelCase of ui.proto:100-105).
//
// Impact's own node_count/edge_count are NOT columns here (D-06): they
// render as a header summary above the table via AnalysisPanel's
// `summary` snippet, keyed off ImpactResponse's scalar fields rather
// than Location's per-row shape.
import type { ColumnDef } from '@tanstack/svelte-table';
import type { Location } from '$lib/gen/ui_pb';
import { features } from './table-features';

export const impactColumns: ColumnDef<typeof features, Location>[] = [
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
