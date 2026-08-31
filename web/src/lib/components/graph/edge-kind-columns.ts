// edge-kind-columns.ts is the per-kind edge breakdown's ColumnDef set,
// mirroring workbench/callers-columns.ts's convention: one *-columns.ts
// module per analysis row shape, rendered through the ONE shared
// DataTable shell (Phase 4's shared-table decision) rather than a
// bespoke table. The row shape here is the same {key,count} pattern
// health/CountTable.svelte already established for a sparse wire map —
// this module defines its own EdgeKindRow rather than importing that
// route's type, since the two are unrelated wire shapes that happen to
// share a column layout.
import type { ColumnDef } from '@tanstack/svelte-table';
import { features } from '$lib/components/workbench/table-features';

export interface EdgeKindRow {
	kind: string;
	count: number;
}

export const edgeKindRowId = (row: EdgeKindRow): string => row.kind;

export const edgeKindColumns: ColumnDef<typeof features, EdgeKindRow>[] = [
	{
		id: 'kind',
		accessorKey: 'kind',
		header: 'Kind',
		sortFn: 'alphanumeric'
	},
	{
		// No explicit sortFn: numeric auto-detection is correct for a
		// number-valued column (the same convention every *-columns.ts
		// file in workbench/ follows).
		id: 'count',
		accessorKey: 'count',
		header: 'Count'
	}
];
