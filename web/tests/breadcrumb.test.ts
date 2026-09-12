// breadcrumb.test.ts — 09-03 Task 1 (BRW-10): innermostSymbolAt and
// firstFullyVisibleLine, the pure derivations behind the sticky
// breadcrumb bar. Written and run RED against stubs returning
// `null`/`0` before the real implementation exists (RED transcript
// pasted in the SUMMARY), then made GREEN.
import { describe, it, expect } from 'vitest';
import { innermostSymbolAt, firstFullyVisibleLine, type SymbolRange } from '$lib/breadcrumb';

// node() fixture builder copied from web/tests/source-pane.test.ts's
// convention (small helper, not exported/shared across test files).
function symbol(name: string, startLine: number, endLine: number): SymbolRange {
	return { name, startLine, endLine };
}

describe('innermostSymbolAt', () => {
	it('is null for an empty symbol list', () => {
		expect(innermostSymbolAt([], 5)).toBeNull();
	});

	it('is null when no range contains the line', () => {
		expect(innermostSymbolAt([symbol('A', 10, 20)], 5)).toBeNull();
	});

	it('picks the tighter of two containing ranges', () => {
		const symbols = [symbol('A', 1, 100), symbol('B', 10, 20)];
		expect(innermostSymbolAt(symbols, 15)?.name).toBe('B');
		expect(innermostSymbolAt(symbols, 5)?.name).toBe('A');
	});

	it('is inclusive of both start and end lines', () => {
		const symbols = [symbol('A', 1, 100), symbol('B', 10, 20)];
		expect(innermostSymbolAt(symbols, 10)?.name).toBe('B');
		expect(innermostSymbolAt(symbols, 20)?.name).toBe('B');
		expect(innermostSymbolAt(symbols, 21)?.name).toBe('A');
	});

	it('breaks an equal-span tie by later input order', () => {
		const symbols = [symbol('A', 10, 20), symbol('B', 10, 20)];
		expect(innermostSymbolAt(symbols, 12)?.name).toBe('B');
	});

	it('breaks a same-length tie by the later-starting range', () => {
		const symbols = [symbol('A', 10, 20), symbol('B', 15, 25)];
		expect(innermostSymbolAt(symbols, 18)?.name).toBe('B');
	});

	it('treats endLine <= 0 as a single-line range at startLine', () => {
		const symbols = [symbol('A', 7, 0)];
		expect(innermostSymbolAt(symbols, 7)?.name).toBe('A');
		expect(innermostSymbolAt(symbols, 8)).toBeNull();
	});

	it('treats endLine < startLine as a single-line range at startLine', () => {
		const symbols = [symbol('A', 7, 3)];
		expect(innermostSymbolAt(symbols, 7)?.name).toBe('A');
		expect(innermostSymbolAt(symbols, 8)).toBeNull();
	});
});

describe('firstFullyVisibleLine', () => {
	it('treats a top edge exactly at the bar bottom as fully visible', () => {
		expect(firstFullyVisibleLine(100, 20, 100, 50)).toBe(1);
	});

	it('moves to line 2 just past the boundary', () => {
		expect(firstFullyVisibleLine(100, 20, 101, 50)).toBe(2);
	});

	it('stays at line 2 through the rest of that line band', () => {
		expect(firstFullyVisibleLine(100, 20, 120, 50)).toBe(2);
		expect(firstFullyVisibleLine(100, 20, 119, 50)).toBe(2);
	});

	it('moves to line 3 exactly at the next boundary', () => {
		expect(firstFullyVisibleLine(100, 20, 121, 50)).toBe(3);
	});

	it('clamps to line 1 when the bar is above the code', () => {
		expect(firstFullyVisibleLine(100, 20, 0, 50)).toBe(1);
	});

	it('is 0 for a zero-line file', () => {
		expect(firstFullyVisibleLine(100, 20, 100, 0)).toBe(0);
	});

	it('clamps to lineCount past the last line', () => {
		expect(firstFullyVisibleLine(100, 20, 100000, 50)).toBe(50);
	});

	it('never returns NaN or Infinity for a zero line height', () => {
		const result = firstFullyVisibleLine(100, 0, 200, 50);
		expect(Number.isFinite(result)).toBe(true);
		expect(result).toBe(1);
	});
});
