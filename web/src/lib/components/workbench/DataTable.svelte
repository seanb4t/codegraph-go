<script lang="ts" generics="TRow extends RowData">
	// DataTable is the ONE shared table shell for every Workbench analysis
	// (D-06) — GENERIC over its row type via Svelte 5's generic-component
	// form, so the exact same shell type-checks for every analysis's row
	// shape, whatever fields that shape carries. Row identity is a PROP
	// (getRowId), never a body constant: hard-coding a composite identity
	// here would make this component unusable for any row type that
	// doesn't carry those exact fields, and `pnpm check` would reject a
	// prop typed to one specific analysis's row shape.
	//
	// Composition source (04-RESEARCH.md Pattern 3, verbatim from
	// github.com/TanStack/table main branch,
	// examples/svelte/lib-shadcn/src/App.svelte): shadcn-svelte's vendored
	// Table.* primitives wrap plain HTML table elements; FlexRender does
	// the actual header/cell content projection; a sortable header is the
	// vendored Button wrapping FlexRender, with
	// header.column.getToggleSortingHandler() as its onclick and
	// header.column.getIsSorted() driving the indicator. The {:else}
	// branch inside {#each table.getRowModel().rows} is TanStack's own
	// idiom for the empty state (D-06's empty/loading/error-states
	// discretion item).
	//
	// This component NAVIGATES NOTHING itself — it calls onSelect, the
	// same props-in/callback-out delegation shape the Browse view's own
	// results-list component already uses for its click-through.
	import type { ColumnDef, RowData } from '@tanstack/svelte-table';
	import { FlexRender, createTable } from '@tanstack/svelte-table';
	import * as Table from '$lib/components/ui/table';
	import { Button } from '$lib/components/ui/button';
	import { features } from './table-features';

	let {
		rows,
		columns,
		getRowId,
		emptyMessage,
		onSelect
	}: {
		rows: TRow[];
		columns: ColumnDef<typeof features, TRow>[];
		getRowId: (row: TRow) => string;
		emptyMessage: string;
		onSelect?: (row: TRow) => void;
	} = $props();

	const table = createTable({
		features,
		get columns() {
			return columns;
		},
		get data() {
			return rows;
		},
		get getRowId() {
			return getRowId;
		}
	});
</script>

<Table.Root>
	<Table.Header>
		{#each table.getHeaderGroups() as headerGroup (headerGroup.id)}
			<Table.Row>
				{#each headerGroup.headers as header (header.id)}
					<Table.Head colspan={header.colSpan}>
						{#if !header.isPlaceholder}
							{#if header.column.getCanSort()}
								<Button
									variant="ghost"
									size="sm"
									data-testid={`table-sort-${header.column.id}`}
									onclick={header.column.getToggleSortingHandler()}
								>
									<FlexRender {header} />
									{#if header.column.getIsSorted() === 'asc'}
										<span aria-hidden="true">▲</span>
									{:else if header.column.getIsSorted() === 'desc'}
										<span aria-hidden="true">▼</span>
									{/if}
								</Button>
							{:else}
								<FlexRender {header} />
							{/if}
						{/if}
					</Table.Head>
				{/each}
			</Table.Row>
		{/each}
	</Table.Header>
	<Table.Body>
		{#each table.getRowModel().rows as row (row.id)}
			<Table.Row
				data-testid={`table-row-${row.id}`}
				class={onSelect ? 'cursor-pointer' : undefined}
				onclick={onSelect ? () => onSelect(row.original) : undefined}
			>
				{#each row.getAllCells() as cell (cell.id)}
					<Table.Cell><FlexRender {cell} /></Table.Cell>
				{/each}
			</Table.Row>
		{:else}
			<Table.Row>
				<Table.Cell colspan={columns.length} class="h-24 text-center text-muted-foreground">
					{emptyMessage}
				</Table.Cell>
			</Table.Row>
		{/each}
	</Table.Body>
</Table.Root>
