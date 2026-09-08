// data-table-virtualization.test.ts — WR-04 and WR-05 (04-REVIEW.md):
// DataTable.svelte's virtualized row rendering, isolated from the
// render-cost budget concerns data-table-render-cost.test.ts owns.
import { fireEvent, render, within } from '@testing-library/svelte';
import { tick } from 'svelte';
import { describe, expect, it } from 'vitest';

import type { Location } from '$lib/gen/ui_pb';
import { callersColumns, locationRowId } from '$lib/components/workbench/callers-columns';
import DataTableHost from './support/data-table-location-host.svelte';

function loc(name: string, kind: string, filePath: string, startLine: number): Location {
	return { name, kind, filePath, startLine } as unknown as Location;
}

function makeRows(n: number): Location[] {
	return Array.from({ length: n }, (_, i) => loc(`symbol-${i}`, 'func', `pkg/file${i}.go`, i + 1));
}

function mountTable(rows: Location[]) {
	return render(DataTableHost, {
		props: {
			rows,
			columns: callersColumns,
			getRowId: locationRowId,
			emptyMessage: 'No rows'
		}
	});
}

describe('WR-05: aria-rowindex is carried onto each rendered row', () => {
	it('the header row is aria-rowindex=1, and the first visible data row is aria-rowindex=2 (1-based, offset by the header)', async () => {
		const result = mountTable(makeRows(50));
		await tick();
		const table = result.getByRole('table');

		const headerRow = within(table).getAllByRole('row')[0];
		expect(headerRow.getAttribute('aria-rowindex')).toBe('1');

		const firstDataRow = within(table).getAllByRole('row')[1];
		expect(firstDataRow.getAttribute('aria-rowindex')).toBe('2');
	});
});

describe('WR-04: DataTable does not crash when rows shrink in place past the virtualizer\'s scrolled offset', () => {
	it('scrolling near the end then rerendering with far fewer rows does not throw / blank the component', async () => {
		const result = mountTable(makeRows(60));
		await tick();

		const scrollContainer = result.getByTestId('data-table-scroll');
		// Push the virtualizer's scroll offset near the END of a 60-row
		// list (60 * 37px row height - 600px visible height).
		Object.defineProperty(scrollContainer, 'scrollTop', {
			configurable: true,
			value: 60 * 37 - 600
		});
		await fireEvent.scroll(scrollContainer);
		await tick();

		// rows shrink IN PLACE — the same DataTable instance, no remount —
		// to far fewer rows than the scrolled-to offset can index into.
		await expect(
			result.rerender({
				rows: makeRows(5),
				columns: callersColumns,
				getRowId: locationRowId,
				emptyMessage: 'No rows'
			})
		).resolves.not.toThrow();
		await tick();

		// The component must still be present and consistent — not blanked
		// by an uncaught mid-render TypeError.
		expect(result.getByRole('table')).toBeInTheDocument();
		const domRows = within(result.getByRole('table')).getAllByRole('row').slice(1);
		// RR-W-03: an upper bound ALONE is satisfied by 0 — i.e. by the blanked
		// component this guard exists to detect. The lower bound is what makes
		// it discriminate (rule 84d1gfpywd: pair every upper bound with a
		// non-zero lower bound).
		expect(domRows.length).toBeGreaterThan(0);
		expect(domRows.length).toBeLessThanOrEqual(5);
	});
});
