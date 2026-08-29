// browse-nav.ts is the ONE URL writer for the Browse view and every
// phase after it (D-11): every navigate/refine write goes through
// createBrowseNavigator's single navigate() method, which states its
// push/replace intent as a named value (NAV_INTENT) rather than a
// boolean literal repeated at each call site.
//
// D-11: a `navigate` intent (opening a node, clicking a neighbour,
// picking a search result) PUSHES history — a place a developer wants
// to come back to. A `refine` intent (typing in search, adjusting depth
// or limit) REPLACES the current entry — the URL stays correct and
// shareable at every instant, but back skips the intermediate states.
//
// This module imports goto's OWN TYPE from $app/navigation (a
// type-only import, erased at build time — never resolved or bundled at
// runtime) rather than importing goto itself: createBrowseNavigator
// takes the navigation function as a parameter, so every case here is
// testable with a plain spy and no SvelteKit runtime. The route module
// (+page.svelte) passes the real goto in.
//
// This module never imports SvelteKit's two shallow-routing history
// exports from $app/navigation — the standalone history-mutating pair,
// distinct from goto's own like-named OPTION below — here or anywhere
// else in web/src/. Those two assign only to page.state, never to
// page.url, so a component reading page.url.searchParams would never
// react to them (03-RESEARCH.md Pitfall 1; the address bar would
// visibly change while the view did not). goto's OWN replaceState
// OPTION is the real lever D-11 needs, and it is the only thing this
// module calls.
import type { goto } from '$app/navigation';
import { parseBrowseParams, serializeBrowseParams, type BrowseParams } from '$lib/browse-url';

export type GotoFn = typeof goto;

// NAV_INTENT names the push/replace choice as a value the caller
// states, rather than a bare boolean repeated at every call site.
// Navigation intents (push): opening a node, clicking a neighbour,
// selecting a search result. Refinement intents (replace): typing in
// search, adjusting depth or limit.
export const NAV_INTENT = {
	NAVIGATE: 'navigate',
	REFINE: 'refine'
} as const;
export type NavIntent = (typeof NAV_INTENT)[keyof typeof NAV_INTENT];

// BrowseNavDelta is the subset of BrowseParams a navigation call can
// set or clear. `unknown` is deliberately excluded: this module never
// invents a second way to touch passthrough params, and no call site in
// this phase needs to — unknown params always survive unchanged,
// carried over from the current URL.
export type BrowseNavDelta = Partial<
	Pick<BrowseParams, 'symbol' | 'file' | 'line' | 'depth' | 'limit' | 'q'>
>;

export interface BrowseNavigator {
	// navigate builds the next URL from `currentUrl` and `delta`, and
	// calls the injected navigation function with `intent`'s push/replace
	// choice. `currentUrl` is passed in explicitly (rather than read
	// internally from $app/state) so this stays free of any SvelteKit
	// runtime dependency.
	navigate(currentUrl: URL, delta: BrowseNavDelta, intent: NavIntent): void;
}

// hasOwn distinguishes "delta explicitly sets this key" (including to
// undefined, which means "remove it") from "delta does not mention this
// key at all" (which means "leave the current value alone") — the
// distinction the whole module's delta semantics depend on. A plain
// `delta[key] !== undefined` check cannot tell these apart: an object
// literal `{ q: undefined }` has an own `q` key, but the shorthand
// check would treat it the same as `{}` (no key at all).
function hasOwn(obj: BrowseNavDelta, key: keyof BrowseNavDelta): boolean {
	return Object.prototype.hasOwnProperty.call(obj, key);
}

// applyTargetClearing enforces the symbol/file/line target-kind rule:
// setting a symbol WITHOUT a file clears the previously set file AND
// line, and vice versa — setting a file WITHOUT a symbol clears the
// previously set symbol (and the previously set line too, unless the
// SAME delta also sets line explicitly, e.g. "open this file at this
// line" with no symbol). Setting BOTH symbol and file in the same delta
// is the disambiguation-triple case (or a caller-directed symbol+file
// pair): nothing here is inferred stale, because the caller named both
// target-kind fields itself in the one delta.
function applyTargetClearing(next: BrowseParams, delta: BrowseNavDelta): void {
	const symbolSet = hasOwn(delta, 'symbol') && delta.symbol !== undefined;
	const fileSet = hasOwn(delta, 'file') && delta.file !== undefined;
	const lineSet = hasOwn(delta, 'line') && delta.line !== undefined;

	if (symbolSet && !fileSet) {
		next.file = undefined;
		next.line = undefined;
	} else if (fileSet && !symbolSet) {
		next.symbol = undefined;
		if (!lineSet) next.line = undefined;
	}
}

const DELTA_KEYS = ['symbol', 'file', 'line', 'depth', 'limit', 'q'] as const;

function applyDelta(current: BrowseParams, delta: BrowseNavDelta): BrowseParams {
	const next: BrowseParams = { ...current };

	for (const key of DELTA_KEYS) {
		if (hasOwn(delta, key)) {
			// TypeScript can't narrow a homogeneous loop-indexed assignment
			// across a union of differently-typed properties; every field in
			// DELTA_KEYS is independently `string | number | undefined` on
			// both sides of this assignment, so the cast is a length
			// mismatch workaround, not a type-safety hole.
			(next as Record<string, unknown>)[key] = delta[key];
		}
	}

	applyTargetClearing(next, delta);

	return next;
}

export function createBrowseNavigator(gotoFn: GotoFn): BrowseNavigator {
	return {
		navigate(currentUrl, delta, intent) {
			const current = parseBrowseParams(currentUrl.searchParams);
			const next = applyDelta(current, delta);
			const nextSearch = serializeBrowseParams(next);

			const url = new URL(currentUrl.href);
			url.search = nextSearch.toString();

			gotoFn(url, {
				replaceState: intent === NAV_INTENT.REFINE,
				noScroll: true,
				keepFocus: true
			});
		}
	};
}
