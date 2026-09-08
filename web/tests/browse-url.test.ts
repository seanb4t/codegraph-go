// Edge coverage for browse-url.ts's parse/serialize grammar (D-12, D-13).
// Pure-TS: no DOM, no SvelteKit runtime.
import { describe, expect, it } from 'vitest';
import { parseBrowseParams, serializeBrowseParams } from '$lib/browse-url';

function buildQuery(pairs: Array<[string, string]>): string {
	const params = new URLSearchParams();
	for (const [key, value] of pairs) params.append(key, value);
	return params.toString();
}

describe('browse-url: round-trip', () => {
	it('round-trips a jumbled full param set back into canonical BROWSE_PARAM_KEYS order', () => {
		const input = buildQuery([
			['file', 'internal/a.go'],
			['depth', '2'],
			['symbol', 'Foo'],
			['q', 'term'],
			['limit', '5'],
			['line', '10']
		]);
		const expected = buildQuery([
			['symbol', 'Foo'],
			['file', 'internal/a.go'],
			['line', '10'],
			['depth', '2'],
			['limit', '5'],
			['q', 'term']
		]);

		const params = parseBrowseParams(new URLSearchParams(input));
		expect(serializeBrowseParams(params).toString()).toBe(expected);
	});

	it('two different insertion orders of the same parameters serialize to the identical string', () => {
		const a = parseBrowseParams(new URLSearchParams('symbol=Foo&file=x.go&limit=5'));
		const b = parseBrowseParams(new URLSearchParams('limit=5&file=x.go&symbol=Foo'));
		const serializedA = serializeBrowseParams(a).toString();
		const serializedB = serializeBrowseParams(b).toString();
		expect(serializedA).toBe(serializedB);
		expect(serializedA.length).toBeGreaterThan(0);
	});
});

describe('browse-url: boundary values', () => {
	it('limit=0, limit=1, and one past MaxLimit (1000, internal/query/validate.go) all parse to their exact integer and pass through unbounded', () => {
		expect(parseBrowseParams(new URLSearchParams('limit=0')).limit).toBe(0);
		expect(parseBrowseParams(new URLSearchParams('limit=1')).limit).toBe(1);
		expect(parseBrowseParams(new URLSearchParams('limit=1001')).limit).toBe(1001);
	});
});

describe('browse-url: shape precision', () => {
	it('non-integer, empty, and out-of-safe-integer-range line values are absent, never 0 and never an error — paired with a present integer', () => {
		expect(parseBrowseParams(new URLSearchParams('line=1.5')).line).toBeUndefined();
		expect(parseBrowseParams(new URLSearchParams('line=abc')).line).toBeUndefined();
		expect(parseBrowseParams(new URLSearchParams('line=')).line).toBeUndefined();
		expect(
			parseBrowseParams(new URLSearchParams('line=99999999999999999999')).line
		).toBeUndefined();
		// Present case, same assertion block: proves the absence checks above
		// are inspecting a grammar that DOES populate `line` for a valid input,
		// not one that always returns undefined.
		expect(parseBrowseParams(new URLSearchParams('line=42')).line).toBe(42);
	});

	it('parseBrowseParams never throws on any input', () => {
		expect(() => parseBrowseParams(new URLSearchParams(''))).not.toThrow();
		expect(() => parseBrowseParams(new URLSearchParams('depth=-oops-'))).not.toThrow();
		expect(() => parseBrowseParams(new URLSearchParams('a=1&a=2&a=3'))).not.toThrow();
	});
});

describe('browse-url: empty and adjacency', () => {
	it('an empty query string yields the idle params object', () => {
		const params = parseBrowseParams(new URLSearchParams(''));
		expect(params.symbol).toBeUndefined();
		expect(params.file).toBeUndefined();
		expect(params.line).toBeUndefined();
		expect(params.depth).toBeUndefined();
		expect(params.limit).toBeUndefined();
		expect(params.q).toBeUndefined();
		expect(params.unknown).toEqual([]);
	});

	it('symbol and file are BOTH retained when both are present — neither is dropped nor merged', () => {
		const params = parseBrowseParams(new URLSearchParams('symbol=Foo&file=internal/x.go'));
		expect(params.symbol).toBe('Foo');
		expect(params.file).toBe('internal/x.go');
	});
});

describe('browse-url: unknown parameters', () => {
	it('a single unknown parameter round-trips through parse and serialize', () => {
		const params = parseBrowseParams(new URLSearchParams('symbol=Foo&futureThing=1'));
		expect(params.unknown).toEqual([['futureThing', '1']]);
		expect(serializeBrowseParams(params).toString()).toBe('symbol=Foo&futureThing=1');
	});

	it('a REPEATED unknown parameter round-trips with BOTH values, in original relative order', () => {
		const params = parseBrowseParams(
			new URLSearchParams('symbol=Foo&futureThing=1&futureThing=2')
		);
		expect(params.unknown).toEqual([
			['futureThing', '1'],
			['futureThing', '2']
		]);
		expect(serializeBrowseParams(params).toString()).toBe(
			'symbol=Foo&futureThing=1&futureThing=2'
		);
	});

	it('an EMPTY-VALUED unknown parameter is preserved, not dropped', () => {
		const params = parseBrowseParams(new URLSearchParams('symbol=Foo&futureThing='));
		expect(params.unknown).toEqual([['futureThing', '']]);
		expect(serializeBrowseParams(params).toString()).toBe('symbol=Foo&futureThing=');
	});
});

describe('browse-url: dotted paths', () => {
	it('a file path containing dots and slashes survives the round trip intact', () => {
		const params = parseBrowseParams(new URLSearchParams('file=internal/x.go&line=1'));
		expect(params.file).toBe('internal/x.go');
		expect(params.line).toBe(1);
		expect(serializeBrowseParams(params).toString()).toBe('file=internal%2Fx.go&line=1');
	});
});
