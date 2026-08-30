// 04-07 Task 3 (D-08): a MEASURED, not assumed, render-cost record for the
// Workbench's shared DataTable shell (D-06) at MaxLimit's real worst case —
// 1000 rows (internal/query/validate.go:26 MaxLimit). D-08's rule is
// measurement first, then virtualize only if a threshold trips — never a
// hand-rolled window and never a second client-side row cap over results
// the server already bounded.
//
// WHY THE THRESHOLD COMPARISON IS OPT-IN, NEVER UNCONDITIONAL. `task
// web:test` runs every file under web/tests/ indiscriminately
// (Taskfile.yml) and is the PR-triggered REQUIRED job at
// .github/workflows/ci.yml:149. A wall-clock duration compared against a
// fixed constant inside a required check varies with runner load,
// architecture, coverage instrumentation and accumulated DOM state — a
// flake generator, the same nondeterministic-gate shape 04-07's Task 2
// drift target rejects (for a different reason: a live network fetch
// rather than a wall-clock). So:
//   - UNCONDITIONALLY (every `task web:test` run, the first two tests
//     below): only deterministic assertions — exactly 1000 rendered rows,
//     and a sort toggle genuinely reorders the RENDERED rows (compared by
//     actual text content, never a class/attribute proxy).
//   - OPT-IN (`task web:render-cost`, which sets RENDER_COST_ASSERT=1 and
//     runs only this file, wired into NO workflow — the third test below):
//     both medians are PRINTED unconditionally on every run, but the
//     threshold comparison itself only fires under that env var.
//
// jsdom timing is NOT browser timing: the threshold in the third test is a
// coarse order-of-magnitude regression detector, not a performance budget.
import { fireEvent, render, within } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';

import type { Location } from '$lib/gen/ui_pb';
import { callersColumns, locationRowId } from '$lib/components/workbench/callers-columns';
// A thin, concretely Location-typed wrapper around DataTable.svelte — see
// its own header comment for why this test renders the host rather than
// DataTable directly: @testing-library/svelte's render() cannot infer
// DataTable's generic TRow from a plain .ts call site.
import DataTableHost from './support/data-table-location-host.svelte';

// internal/query/validate.go:26 — MaxLimit. This is the real worst case a
// workbench table can be asked to render, not an invented one.
const ROW_COUNT = 1000;
const REPETITIONS = 5;

function loc(name: string, kind: string, filePath: string, startLine: number): Location {
	return { name, kind, filePath, startLine } as unknown as Location;
}

// Deliberately shuffled, NOT ascending or descending: a monotonic
// insertion order (even a fully descending one) would make one of
// TanStack's two sorted states indistinguishable from the unsorted
// original, silently defeating the "genuinely reorders" assertion below.
// `(i * 457) % n` is a fixed, deterministic permutation of 0..n-1 (457 is
// prime and does not divide 1000, so it is coprime with n and the mapping
// is a bijection) — reproducible across runs, never a real Math.random.
function makeRows(n: number): Location[] {
	const rows: Location[] = new Array(n);
	for (let i = 0; i < n; i++) {
		const shuffledIndex = (i * 457) % n;
		const suffix = String(shuffledIndex).padStart(5, '0');
		rows[i] = loc(`symbol-${suffix}`, 'func', `pkg/file${i % 37}.go`, (i % 500) + 1);
	}
	return rows;
}

function median(values: number[]): number {
	const sorted = [...values].sort((a, b) => a - b);
	const mid = Math.floor(sorted.length / 2);
	return sorted.length % 2 === 0 ? (sorted[mid - 1] + sorted[mid]) / 2 : sorted[mid];
}

// rowsMatchNameOrder checks the rendered row text content is consistent
// with `name` values appearing in `expectedNames` order — each rendered
// row's text must CONTAIN its expected name at that position. Avoids a
// brittle exact-string match against every rendered column's formatting
// while still proving genuine reordering by name.
function rowsMatchNameOrder(renderedRows: string[], expectedNames: string[]): boolean {
	if (renderedRows.length !== expectedNames.length) return false;
	return renderedRows.every((rowText, i) => rowText.includes(expectedNames[i]));
}

const ROWS = makeRows(ROW_COUNT);

function mountTable() {
	return render(DataTableHost, {
		props: {
			rows: ROWS,
			columns: callersColumns,
			getRowId: locationRowId,
			emptyMessage: 'No rows'
		}
	});
}

describe('DataTable render cost at MaxLimit (D-08, measurement-first)', () => {
	it('represents exactly 1000 rows in the row MODEL, with a genuinely virtualized (smaller) DOM window — positive control (rule 84d1gfpywd)', async () => {
		const result = mountTable();
		await Promise.resolve();
		const table = result.getByRole('table');

		// Virtualization (D-08, 04-07 Task 4 — @tanstack/svelte-virtual)
		// means the DOM no longer holds all 1000 <tr> elements at once, so
		// the "all rows represented" positive control is restated against
		// the table's row MODEL — aria-rowcount, which DataTable.svelte
		// forwards from table.getRowModel().rows.length onto the <table>
		// element for exactly this reason — never the DOM node count. This
		// still catches the failure mode the control exists for: a change
		// that silently drops rows from the MODEL, not merely from the
		// DOM window.
		// aria-rowcount counts EVERY row in the full table, header included
		// (WAI-ARIA 1.2), so the model row count is one less. RR-W-01 corrected
		// aria-rowcount from rows.length to rows.length + 1 precisely so it can
		// never be smaller than the largest aria-rowindex, which the header
		// occupies at 1 and data rows at index + 2.
		const modelRowCount = Number(table.getAttribute('aria-rowcount')) - 1;
		const domRowCount = within(table).getAllByRole('row').length - 1; // minus header row
		result.unmount();

		// A measurement over an empty or truncated render would read as fast
		// and prove nothing — this is UNCONDITIONAL, never gated behind
		// RENDER_COST_ASSERT.
		expect(modelRowCount).toBe(ROW_COUNT);

		// The complementary half of the same control: DOM row count must be
		// STRICTLY LESS than the model count (proving virtualization is
		// actually windowing the DOM, not merely claiming to via
		// aria-rowcount while still rendering everything) and greater than
		// zero (proving something actually rendered).
		expect(domRowCount).toBeGreaterThan(0);
		expect(domRowCount).toBeLessThan(ROW_COUNT);
	});

	it('clicking the Name header genuinely reorders the rendered rows', async () => {
		const result = mountTable();
		const table = result.getByRole('table');
		const nameHeaderButton = result.getByRole('button', { name: /Name/ });

		const renderedNameOrder = () =>
			within(table)
				.getAllByRole('row')
				.slice(1) // drop the header row
				.map((row) => row.textContent ?? '');

		const beforeOrder = renderedNameOrder();
		await fireEvent.click(nameHeaderButton);
		const afterOrder = renderedNameOrder();
		result.unmount();

		// Deterministic and UNCONDITIONAL: the sort toggle genuinely reorders
		// the RENDERED rows, compared by actual rendered text content — never
		// a toggled indicator class standing in for the real thing.
		//
		// Virtualization means only the scrolled-into-view WINDOW is in the
		// DOM (scroll position stays at the top across the click — sorting
		// re-derives the row model but does not move the scroll offset), so
		// this compares that window's rendered names against the SAME
		// leading slice of the full ascending-sorted name list, rather than
		// requiring all 1000 rows in the DOM at once.
		const ascNames = [...ROWS].map((r) => r.name).sort((a, b) => a.localeCompare(b));
		expect(afterOrder).not.toEqual(beforeOrder);
		expect(rowsMatchNameOrder(afterOrder, ascNames.slice(0, afterOrder.length))).toBe(true);
	});

	it(
		'records median initial-render and sort-toggle durations over 5 repetitions — threshold opt-in',
		async () => {
			// --- Initial render: REPETITIONS independent mounts, MEDIAN reported ---
			const renderDurations: number[] = [];
			for (let i = 0; i < REPETITIONS; i++) {
				const start = performance.now();
				const result = mountTable();
				// Settle: one microtask turn so any post-mount reactive work
				// (TanStack's row-model derivation) has actually run before the
				// clock stops.
				await Promise.resolve();
				renderDurations.push(performance.now() - start);
				result.unmount();
			}
			const renderMedian = median(renderDurations);

			// --- Sort toggle: REPETITIONS clicks on one freshly mounted table, MEDIAN reported ---
			const result = mountTable();
			const nameHeaderButton = result.getByRole('button', { name: /Name/ });
			const sortDurations: number[] = [];
			for (let i = 0; i < REPETITIONS; i++) {
				const start = performance.now();
				await fireEvent.click(nameHeaderButton);
				sortDurations.push(performance.now() - start);
			}
			result.unmount();
			const sortMedian = median(sortDurations);

			// Printed on EVERY run, unconditionally — a human reading CI output
			// sees the real numbers even when the opt-in threshold below never
			// runs.
			// eslint-disable-next-line no-console
			console.log(
				`[render-cost] rows=${ROW_COUNT} initial-render-median=${renderMedian.toFixed(2)}ms sort-toggle-median=${sortMedian.toFixed(2)}ms`
			);

			// D-08's threshold comparison — OPT-IN ONLY (see file header). Never
			// asserted inside the PR-required `task web:test`
			// (.github/workflows/ci.yml:149); only `task web:render-cost`
			// (Taskfile.yml) sets RENDER_COST_ASSERT=1.
			if (process.env.RENDER_COST_ASSERT === '1') {
				expect(
					renderMedian,
					'median initial render of 1000 rows in jsdom (coarse detector, not a budget)'
				).toBeLessThanOrEqual(400);
				expect(
					sortMedian,
					'median sort-toggle over 1000 rows in jsdom (coarse detector, not a budget)'
				).toBeLessThanOrEqual(200);
			}
		},
		30000
	);
});
