// 04-01 Task 3: locks the Workbench URL grammar (workbench-url.ts) with
// executable tests, mirroring web/tests/browse-url.test.ts's round-trip
// structure — the multi-value `file=` case is this file's deliberate
// divergence (D-11), the one thing this phase is most likely to get
// silently wrong (a repeated `file=` collapsing to one entry, with every
// other test still green).
import { describe, expect, it } from 'vitest';

import {
	WORKBENCH_MODES,
	parseWorkbenchParams,
	serializeWorkbenchParams,
	type WorkbenchParams
} from '$lib/workbench-url';

describe('parseWorkbenchParams / serializeWorkbenchParams: round-trip', () => {
	it('round-trips every field: mode, symbol, two files, depth, limit', () => {
		const p: WorkbenchParams = {
			mode: 'callers',
			symbol: 'Engine.Status',
			files: ['internal/query/node.go', 'internal/query/status.go'],
			depth: 3,
			limit: 25,
			unknown: []
		};
		const roundTripped = parseWorkbenchParams(serializeWorkbenchParams(p));
		expect(roundTripped).toEqual(p);
	});

	it('a repeated file= URL parses to exactly three entries IN URL ORDER (parse direction, D-11)', () => {
		const params = parseWorkbenchParams(
			new URLSearchParams('file=a.go&file=b%2Fc.go&file=d.go')
		);
		expect(params.files).toEqual(['a.go', 'b/c.go', 'd.go']);
	});

	it('three files serialize back to three DISTINCT file= entries IN ARRAY ORDER (serialize direction, D-11)', () => {
		const serialized = serializeWorkbenchParams({
			files: ['a.go', 'b/c.go', 'd.go'],
			unknown: []
		});
		expect(serialized.getAll('file')).toEqual(['a.go', 'b/c.go', 'd.go']);
	});

	it('a repeated file= URL round-trips through BOTH directions unchanged — the single-value-reader regression this phase is most likely to introduce', () => {
		const original = new URLSearchParams('file=a.go&file=b%2Fc.go&file=d.go');
		const parsed = parseWorkbenchParams(original);
		const reserialized = serializeWorkbenchParams(parsed);
		expect(reserialized.getAll('file')).toEqual(['a.go', 'b/c.go', 'd.go']);
		// And the round-trip through parse again lands on the SAME parsed
		// shape — proving neither direction silently drops or reorders.
		expect(parseWorkbenchParams(reserialized)).toEqual(parsed);
	});

	it('an unrecognized parameter survives a full round-trip, including a REPEATED unrecognized parameter', () => {
		const params = parseWorkbenchParams(new URLSearchParams('z=1&z=2&a=x'));
		expect(params.unknown).toEqual([
			['z', '1'],
			['z', '2'],
			['a', 'x']
		]);
		const reserialized = serializeWorkbenchParams(params);
		// Unknown keys are sorted lexicographically at serialize time
		// (mirrors serializeBrowseParams), with same-key entries kept in
		// their original relative order — 'a' before 'z', and z=1 before
		// z=2, never collapsed to one entry.
		expect([...reserialized.entries()]).toEqual([
			['a', 'x'],
			['z', '1'],
			['z', '2']
		]);
	});

	it('an out-of-range but well-shaped integer parses to that exact number — shape, not range (D-07/D-12)', () => {
		const params = parseWorkbenchParams(new URLSearchParams('limit=999999&depth=9999'));
		expect(params.limit).toBe(999999);
		expect(params.depth).toBe(9999);
	});

	it('a NEGATIVE but well-shaped integer parses to that exact negative number and survives a full round-trip', () => {
		// INTEGER_SHAPE is ^-?\d+$ (browse-url.ts:48) — negatives are
		// accepted BY DESIGN and are the server's to refuse (D-07). This is
		// the one direction existing coverage (oversized positives) never
		// tests, and the one that would silently start returning
		// `undefined` if someone "tightened" the shape.
		const params = parseWorkbenchParams(new URLSearchParams('depth=-1&limit=-5'));
		expect(params.depth).toBe(-1);
		expect(params.limit).toBe(-5);

		const reserialized = serializeWorkbenchParams(params);
		expect(reserialized.get('depth')).toBe('-1');
		expect(reserialized.get('limit')).toBe('-5');
		expect(parseWorkbenchParams(reserialized)).toEqual(params);
	});

	it('a malformed integer parses to undefined, never 0', () => {
		expect(parseWorkbenchParams(new URLSearchParams('depth=1.5')).depth).toBeUndefined();
		expect(parseWorkbenchParams(new URLSearchParams('depth=abc')).depth).toBeUndefined();
		expect(parseWorkbenchParams(new URLSearchParams('depth=')).depth).toBeUndefined();
	});

	it('an unknown mode value parses to undefined, and every member of WORKBENCH_MODES parses to itself', () => {
		expect(
			parseWorkbenchParams(new URLSearchParams('mode=not-a-real-mode')).mode
		).toBeUndefined();

		for (const mode of WORKBENCH_MODES) {
			expect(parseWorkbenchParams(new URLSearchParams(`mode=${mode}`)).mode).toBe(mode);
		}
	});

	it('an absent file param parses to an empty array, never undefined', () => {
		expect(parseWorkbenchParams(new URLSearchParams('')).files).toEqual([]);
	});
});
