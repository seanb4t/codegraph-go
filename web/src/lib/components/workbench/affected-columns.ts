// affected-columns.ts is the Affected analysis's ColumnDef set (D-06) —
// the fourth of four near-identical *-columns.ts files (04-01's
// callers-columns.ts, 04-04's impact-columns.ts/callees-columns.ts) over
// Location's four wire fields (name, kind, file_path, start_line —
// generated camelCase of ui.proto:100-105). Rows come from
// AffectedResponse.affected_tests; the echoed `files` list is NOT a
// column here (D-06) — it renders as a header summary above the table
// via AnalysisPanel's `summary` snippet, matching impact-columns.ts's
// own node_count/edge_count precedent.
import type { ColumnDef } from '@tanstack/svelte-table';
import type { Location } from '$lib/gen/ui_pb';
import { features } from './table-features';

export const affectedColumns: ColumnDef<typeof features, Location>[] = [
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
