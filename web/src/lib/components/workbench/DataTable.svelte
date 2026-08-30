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
	// header.column.getIsSorted() driving the indicator.
	//
	// This component NAVIGATES NOTHING itself — it calls onSelect, the
	// same props-in/callback-out delegation shape the Browse view's own
	// results-list component already uses for its click-through.
	//
	// VIRTUALIZATION (D-08, 04-07 Task 4 — maintainer-approved
	// @tanstack/svelte-virtual@3.13.36 install after the render-cost
	// measurement at MaxLimit=1000 rows came back over the plan's fixed
	// jsdom threshold on both metrics). This is NOT a client-side row cap
	// (D-08 explicitly rejects that): the row MODEL
	// (table.getRowModel().rows) still carries every row returned by the
	// server — sorting, selection and aria-rowcount all operate over the
	// FULL set. Only the DOM footprint shrinks to the scrolled-into-view
	// window (+ overscan), via the two-padding-row technique standard for
	// virtualizing native <table> markup (a <tr> cannot be individually
	// absolute-positioned without breaking table layout, unlike a plain
	// list) — one spacer <tr> above and below the rendered slice, each
	// sized to stand in for the rows currently outside the DOM.
	import type { ColumnDef, RowData } from '@tanstack/svelte-table';
	import { FlexRender, createTable } from '@tanstack/svelte-table';
	import { createVirtualizer } from '@tanstack/svelte-virtual';
	import { get } from 'svelte/store';
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

	// A uniform estimate, not a per-row ResizeObserver measurement: every
	// rendered row shares the same Table.Cell padding/text-sm classes, so
	// a fixed height is accurate enough without measureElement (which
	// depends on ResizeObserver — unavailable/unreliable under jsdom, and
	// unnecessary for a uniform row height).
	const ROW_HEIGHT_PX = 37;
	const OVERSCAN = 10;

	let scrollContainer: HTMLDivElement | null = $state(null);

	// initialRect: the officially-supported fallback size for the SSR /
	// pre-hydration case, where getScrollElement() returns null and no
	// measurement can happen at all yet. It does NOT fix jsdom-based unit
	// tests on its own: @tanstack/virtual-core's observeElementRect calls
	// element.offsetWidth/offsetHeight SYNCHRONOUSLY on every setOptions,
	// which permanently overwrites this fallback the instant a real (but
	// jsdom-zeroed) scroll element is available — confirmed live against
	// this exact component (04-07 Task 4). The actual jsdom fix is
	// web/tests/setup.ts's scoped offsetWidth/offsetHeight stub for this
	// component's own data-testid="data-table-scroll" container; this
	// initialRect is still kept for the genuine SSR case it exists for.
	const CONTAINER_HEIGHT_PX = 600;

	const rowVirtualizer = createVirtualizer<HTMLDivElement, HTMLTableRowElement>({
		count: table.getRowModel().rows.length,
		getScrollElement: () => scrollContainer,
		estimateSize: () => ROW_HEIGHT_PX,
		overscan: OVERSCAN,
		initialRect: { width: 0, height: CONTAINER_HEIGHT_PX }
	});

	// createVirtualizer's options are captured ONCE at construction (its
	// own source only re-evaluates them on an explicit setOptions call,
	// never automatically on every render — it is a Svelte-store-based
	// API, not a runes-native one). rows/columns can change after mount in
	// real usage (a depth/limit control re-running an analysis), so this
	// effect keeps the virtualizer's row count and scroll element in sync.
	// Reads `rowVirtualizer` (the raw store) via get(), never `$rowVirtualizer`
	// (the auto-subscribed value) — subscribing here would make this
	// effect a dependent of its OWN setOptions call (every setOptions
	// unconditionally pushes a new store value, per the wrapper's own
	// comment), which would loop forever.
	$effect(() => {
		const rowCount = table.getRowModel().rows.length;
		const container = scrollContainer;
		get(rowVirtualizer).setOptions({
			count: rowCount,
			getScrollElement: () => container,
			estimateSize: () => ROW_HEIGHT_PX,
			overscan: OVERSCAN,
			initialRect: { width: 0, height: CONTAINER_HEIGHT_PX }
		});
	});
</script>

<div bind:this={scrollContainer} class="overflow-y-auto" style={`max-height: ${CONTAINER_HEIGHT_PX}px;`} data-testid="data-table-scroll">
	<Table.Root aria-rowcount={table.getRowModel().rows.length}>
		<Table.Header>
			{#each table.getHeaderGroups() as headerGroup (headerGroup.id)}
				<Table.Row aria-rowindex={1}>
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
			{#if table.getRowModel().rows.length === 0}
				<Table.Row>
					<Table.Cell colspan={columns.length} class="h-24 text-center text-muted-foreground">
						{emptyMessage}
					</Table.Cell>
				</Table.Row>
			{:else}
				{@const virtualRows = $rowVirtualizer.getVirtualItems()}
				{@const totalSize = $rowVirtualizer.getTotalSize()}
				{@const paddingTop = virtualRows.length > 0 ? virtualRows[0].start : 0}
				{@const paddingBottom =
					virtualRows.length > 0 ? totalSize - virtualRows[virtualRows.length - 1].end : 0}
				{#if paddingTop > 0}
					<tr aria-hidden="true" data-testid="table-virtual-padding-top">
						<td style={`height: ${paddingTop}px; padding: 0;`} colspan={columns.length}></td>
					</tr>
				{/if}
				{#each virtualRows as virtualRow (virtualRow.key)}
					{@const row = table.getRowModel().rows[virtualRow.index]}
					{#if row}
						<!-- WR-05: aria-rowindex is 1-based and row 1 is the
							 header row (set above), so a data row's WAI-ARIA
							 index is its position in the FULL (unvirtualized)
							 row model, offset by 2 — not its DOM position among
							 the ~27 rows actually rendered. Required once
							 aria-rowcount (below) reports the full model size
							 while the DOM holds only a windowed subset;
							 without it, assistive tech is told "N rows" and
							 handed a subset with no way to place them. -->
						<Table.Row
							data-testid={`table-row-${row.id}`}
							aria-rowindex={virtualRow.index + 2}
							class={onSelect ? 'cursor-pointer' : undefined}
							onclick={onSelect ? () => onSelect(row.original) : undefined}
						>
							{#each row.getAllCells() as cell (cell.id)}
								<Table.Cell><FlexRender {cell} /></Table.Cell>
							{/each}
						</Table.Row>
					{/if}
				{/each}
				{#if paddingBottom > 0}
					<tr aria-hidden="true" data-testid="table-virtual-padding-bottom">
						<td style={`height: ${paddingBottom}px; padding: 0;`} colspan={columns.length}></td>
					</tr>
				{/if}
			{/if}
		</Table.Body>
	</Table.Root>
</div>
