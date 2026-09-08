// browse-history.test.ts — 03-07 Task 3: the AUTOMATED half of NAV-02.
// Proves the URL<->state contract across a REAL jsdom history stack:
// building successive states through browse-nav.ts's navigator and a
// gotoFn stub that calls the REAL window.history.pushState/
// replaceState (jsdom implements history.pushState, back(), forward()
// and popstate — it does NOT run SvelteKit's router). The OTHER half of
// NAV-02 — that a real popstate re-derives page.url and re-renders the
// view — needs the router, which needs the app, which needs a real
// browser; it stays manual and is proven live against a real
// `codegraph ui` server, recorded verbatim in the SUMMARY.
import { describe, it, expect, beforeEach } from 'vitest';
import { createBrowseNavigator, NAV_INTENT, type GotoFn } from '$lib/browse-nav';
import { parseBrowseParams } from '$lib/browse-url';

// realGotoFn exercises goto()'s OWN replaceState-option contract
// against jsdom's REAL history object — the same push-vs-replace choice
// the real SvelteKit goto() makes — rather than a spy that only records
// what it was called with.
function realGotoFn(): GotoFn {
	const fn = (url: string | URL, opts?: { replaceState?: boolean }) => {
		const href = typeof url === 'string' ? url : url.toString();
		if (opts?.replaceState) {
			window.history.replaceState(null, '', href);
		} else {
			window.history.pushState(null, '', href);
		}
		return Promise.resolve();
	};
	return fn as unknown as GotoFn;
}

function currentUrl(): URL {
	return new URL(window.location.href);
}

function waitForPopstate(): Promise<void> {
	return new Promise((resolve) => {
		window.addEventListener('popstate', () => resolve(), { once: true });
	});
}

// jsdom's own default test document origin is http://localhost:3000/
// (vitest's jsdom environment default) — replaceState()/pushState()
// refuse to cross origins, so every URL built in this file must share
// it rather than a hand-picked 'http://localhost' with no port.
const ORIGIN = 'http://localhost:3000';

beforeEach(() => {
	window.history.replaceState(null, '', `${ORIGIN}/browse`);
});

describe('browse-history: URL<->state contract across a real jsdom history stack', () => {
	it('A -> B -> C (all navigate/push): one back() yields Bs parsed params exactly; forward() yields Cs', async () => {
		const navigator = createBrowseNavigator(realGotoFn());

		navigator.navigate(currentUrl(), { symbol: 'A' }, NAV_INTENT.NAVIGATE);
		navigator.navigate(currentUrl(), { symbol: 'B' }, NAV_INTENT.NAVIGATE);
		navigator.navigate(currentUrl(), { symbol: 'C' }, NAV_INTENT.NAVIGATE);

		expect(parseBrowseParams(currentUrl().searchParams).symbol).toBe('C');

		let popped = waitForPopstate();
		window.history.back();
		await popped;
		expect(parseBrowseParams(currentUrl().searchParams).symbol).toBe('B');

		popped = waitForPopstate();
		window.history.forward();
		await popped;
		expect(parseBrowseParams(currentUrl().searchParams).symbol).toBe('C');
	});

	it('push/replace stack DEPTH proof: a refine between B and C means one back() from C yields the REFINEMENT Bʹ, and a SECOND back() yields A', async () => {
		const navigator = createBrowseNavigator(realGotoFn());

		navigator.navigate(currentUrl(), { symbol: 'A' }, NAV_INTENT.NAVIGATE); // push -> A
		navigator.navigate(currentUrl(), { symbol: 'B' }, NAV_INTENT.NAVIGATE); // push -> B
		navigator.navigate(currentUrl(), { depth: 5 }, NAV_INTENT.REFINE); // replace -> Bʹ (B with depth=5)
		navigator.navigate(currentUrl(), { symbol: 'C' }, NAV_INTENT.NAVIGATE); // push -> C

		expect(parseBrowseParams(currentUrl().searchParams).symbol).toBe('C');

		let popped = waitForPopstate();
		window.history.back();
		await popped;
		const afterFirstBack = parseBrowseParams(currentUrl().searchParams);
		// This IS Bʹ (the refinement), never bare B — a refine REPLACES,
		// so the stack is [A, B', C], not [A, B, B', C].
		expect(afterFirstBack.symbol).toBe('B');
		expect(afterFirstBack.depth).toBe(5);

		popped = waitForPopstate();
		window.history.back();
		await popped;
		const afterSecondBack = parseBrowseParams(currentUrl().searchParams);
		// The SECOND back is the entire proof: correct wiring (refine
		// replaces) yields A here; wrong wiring (refine also pushes, stack
		// [A, B, B', C]) would yield bare B a second time instead.
		expect(afterSecondBack.symbol).toBe('A');
	});
});
