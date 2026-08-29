// browse-nav.test.ts — 03-07 Task 1: the navigation module's own tests,
// written and run RED before web/src/lib/browse-nav.ts exists (recorded
// in the SUMMARY), then made GREEN. Every case here uses an injected
// spy in place of the real SvelteKit `goto` — createBrowseNavigator
// takes the navigation function as a parameter specifically so none of
// this needs a SvelteKit runtime.
import { describe, it, expect, vi } from 'vitest';
import { createBrowseNavigator, NAV_INTENT, type GotoFn } from '$lib/browse-nav';
import { serializeBrowseParams, parseBrowseParams, type BrowseParams } from '$lib/browse-url';

function spyGoto() {
	return vi.fn().mockResolvedValue(undefined) as unknown as GotoFn;
}

function lastCall(gotoFn: GotoFn) {
	const calls = (gotoFn as unknown as { mock: { calls: unknown[][] } }).mock.calls;
	const [url, opts] = calls[calls.length - 1] as [
		URL,
		{ replaceState: boolean; noScroll: boolean; keepFocus: boolean }
	];
	return { url, opts, params: parseBrowseParams(url.searchParams) };
}

describe('createBrowseNavigator: intent -> replaceState, and the two intents DIFFER', () => {
	it('a navigate intent calls the injected navigation function with replaceState false', () => {
		const gotoFn = spyGoto();
		const nav = createBrowseNavigator(gotoFn);
		nav.navigate(new URL('http://x/browse'), { symbol: 'Foo' }, NAV_INTENT.NAVIGATE);
		expect(lastCall(gotoFn).opts.replaceState).toBe(false);
	});

	it('a refine intent calls the injected navigation function with replaceState true', () => {
		const gotoFn = spyGoto();
		const nav = createBrowseNavigator(gotoFn);
		nav.navigate(new URL('http://x/browse'), { q: 'ab' }, NAV_INTENT.REFINE);
		expect(lastCall(gotoFn).opts.replaceState).toBe(true);
	});

	it('the two booleans DIFFER — not merely that each intent was called', () => {
		const gotoFn = spyGoto();
		const nav = createBrowseNavigator(gotoFn);
		nav.navigate(new URL('http://x/browse'), { symbol: 'Foo' }, NAV_INTENT.NAVIGATE);
		const navigateReplaceState = lastCall(gotoFn).opts.replaceState;
		nav.navigate(new URL('http://x/browse'), { q: 'ab' }, NAV_INTENT.REFINE);
		const refineReplaceState = lastCall(gotoFn).opts.replaceState;
		expect(navigateReplaceState).not.toBe(refineReplaceState);
	});
});

describe('createBrowseNavigator: both noScroll and keepFocus are always set', () => {
	it('every call carries noScroll:true and keepFocus:true, for both intents', () => {
		const gotoFn = spyGoto();
		const nav = createBrowseNavigator(gotoFn);
		nav.navigate(new URL('http://x/browse'), { symbol: 'Foo' }, NAV_INTENT.NAVIGATE);
		expect(lastCall(gotoFn).opts.noScroll).toBe(true);
		expect(lastCall(gotoFn).opts.keepFocus).toBe(true);
		nav.navigate(new URL('http://x/browse'), { q: 'ab' }, NAV_INTENT.REFINE);
		expect(lastCall(gotoFn).opts.noScroll).toBe(true);
		expect(lastCall(gotoFn).opts.keepFocus).toBe(true);
	});
});

describe('createBrowseNavigator: only the named parameters change, everything else survives', () => {
	it('changes symbol/file/line and leaves depth/limit/q and unknown params (incl. repeated) unchanged', () => {
		const gotoFn = spyGoto();
		const nav = createBrowseNavigator(gotoFn);
		const currentUrl = new URL(
			'http://x/browse?symbol=Old&depth=3&limit=50&q=abc&futureThing=1&futureThing=2'
		);
		nav.navigate(currentUrl, { symbol: 'New', file: 'a.go', line: 10 }, NAV_INTENT.NAVIGATE);
		const { params } = lastCall(gotoFn);
		expect(params.symbol).toBe('New');
		expect(params.file).toBe('a.go');
		expect(params.line).toBe(10);
		expect(params.depth).toBe(3);
		expect(params.limit).toBe(50);
		expect(params.q).toBe('abc');
		expect(params.unknown).toEqual([
			['futureThing', '1'],
			['futureThing', '2']
		]);
	});

	it('a parameter set to undefined in the delta is removed from the resulting URL', () => {
		const gotoFn = spyGoto();
		const nav = createBrowseNavigator(gotoFn);
		const currentUrl = new URL('http://x/browse?q=abc&depth=3');
		nav.navigate(currentUrl, { q: undefined }, NAV_INTENT.REFINE);
		const { params } = lastCall(gotoFn);
		expect(params.q).toBeUndefined();
		expect(params.depth).toBe(3);
	});

	it('a key absent from the delta leaves the current value untouched', () => {
		const gotoFn = spyGoto();
		const nav = createBrowseNavigator(gotoFn);
		const currentUrl = new URL('http://x/browse?symbol=Keep&depth=3');
		nav.navigate(currentUrl, { depth: 7 }, NAV_INTENT.REFINE);
		const { params } = lastCall(gotoFn);
		expect(params.symbol).toBe('Keep');
		expect(params.depth).toBe(7);
	});
});

describe('createBrowseNavigator: symbol/file/line target-kind clearing (D-10), with the disambiguation-triple exception', () => {
	it('setting a symbol without a file clears a previously set file and line', () => {
		const gotoFn = spyGoto();
		const nav = createBrowseNavigator(gotoFn);
		const currentUrl = new URL('http://x/browse?file=old.go&line=5');
		nav.navigate(currentUrl, { symbol: 'Foo' }, NAV_INTENT.NAVIGATE);
		const { params } = lastCall(gotoFn);
		expect(params.symbol).toBe('Foo');
		expect(params.file).toBeUndefined();
		expect(params.line).toBeUndefined();
	});

	it('setting a file without a symbol clears a previously set symbol (vice versa)', () => {
		const gotoFn = spyGoto();
		const nav = createBrowseNavigator(gotoFn);
		const currentUrl = new URL('http://x/browse?symbol=Old&line=5');
		nav.navigate(currentUrl, { file: 'new.go' }, NAV_INTENT.NAVIGATE);
		const { params } = lastCall(gotoFn);
		expect(params.file).toBe('new.go');
		expect(params.symbol).toBeUndefined();
		expect(params.line).toBeUndefined();
	});

	it('setting a file WITH an explicit line keeps that line — only symbol clears', () => {
		const gotoFn = spyGoto();
		const nav = createBrowseNavigator(gotoFn);
		const currentUrl = new URL('http://x/browse?symbol=Old&line=5');
		nav.navigate(currentUrl, { file: 'new.go', line: 20 }, NAV_INTENT.NAVIGATE);
		const { params } = lastCall(gotoFn);
		expect(params.file).toBe('new.go');
		expect(params.line).toBe(20);
		expect(params.symbol).toBeUndefined();
	});

	it('EXCEPTION: the symbol+file+line disambiguation triple survives a round trip intact, without clearing', () => {
		const gotoFn = spyGoto();
		const nav = createBrowseNavigator(gotoFn);
		const currentUrl = new URL('http://x/browse?symbol=Old&file=old.go&line=1');
		nav.navigate(
			currentUrl,
			{ symbol: 'Foo', file: 'foo.go', line: 42 },
			NAV_INTENT.NAVIGATE
		);
		const { params } = lastCall(gotoFn);
		expect(params.symbol).toBe('Foo');
		expect(params.file).toBe('foo.go');
		expect(params.line).toBe(42);
	});
});

describe('createBrowseNavigator: exactly one serialization', () => {
	it('the produced query string is byte-identical to serializeBrowseParams for the resulting state', () => {
		const gotoFn = spyGoto();
		const nav = createBrowseNavigator(gotoFn);
		const currentUrl = new URL('http://x/browse?symbol=Old&depth=3&futureThing=1');
		nav.navigate(currentUrl, { symbol: 'New', file: undefined, line: undefined }, NAV_INTENT.NAVIGATE);
		const { url } = lastCall(gotoFn);

		const expected: BrowseParams = {
			symbol: 'New',
			file: undefined,
			line: undefined,
			depth: 3,
			limit: undefined,
			q: undefined,
			unknown: [['futureThing', '1']]
		};
		expect(url.search).toBe('?' + serializeBrowseParams(expected).toString());
	});
});
