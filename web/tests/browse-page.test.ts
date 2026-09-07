// CR-01 regression (03-REVIEW.md): typing in the search box used to tear
// down and re-issue the whole open-node view on every keystroke, because
// +page.svelte's load effect read the ENTIRE `params` object — a fresh
// object literal on every URL write (browse-url.ts's parseBrowseParams) —
// rather than only the target-identifying fields. This is the ONE
// route-level test in the phase: every other browse-*.test.ts exercises
// the loaders/navigator in isolation and never reproduced this.
//
// jsdom does not implement scrollIntoView; the vendored Command
// primitive SearchPanel renders calls it as a side effect of moving
// selection (search-panel.test.ts's own note). Stubbed once here.
if (!Element.prototype.scrollIntoView) {
	Element.prototype.scrollIntoView = () => {};
}

import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { flushSync } from 'svelte';

import type {
	GetNodeDetailResponse,
	ImpactResponse,
	GetPermalinkResponse,
	SearchResponse,
	FilesResponse,
	ExploreResponse
} from '$lib/gen/ui_pb';
import { NodeDetailMode, PermalinkAvailability } from '$lib/gen/ui_pb';

// mockPage/resetMockPage: the ONE reactive `$app/state`-shaped object
// this file's $app/state mock and the test body both read/write — see
// support/browse-page-state.svelte.ts for why this has to be a
// singleton living in its own .svelte.ts module.
import { mockPage, resetMockPage } from './support/browse-page-state.svelte';

// Both mock factories dynamically import the support module INSIDE the
// factory, rather than closing over the top-level `mockPage` import,
// deliberately: `vi.mock` calls are hoisted above this file's own
// import statements, so a factory that referenced `mockPage` directly
// would be closing over a binding from an import that (per hoisting)
// has not run yet. A dynamic `import()` inside the factory body has no
// such ordering dependency, and ES module caching guarantees it
// resolves to the exact same singleton the top-level import above
// binds — so the mocked component and this test's assertions are always
// reading/writing the same object.
vi.mock('$app/state', async () => {
	const { mockPage } = await import('./support/browse-page-state.svelte');
	return { page: mockPage };
});

vi.mock('$app/navigation', async () => {
	const { mockPage } = await import('./support/browse-page-state.svelte');
	return {
		// goto() here does exactly what SvelteKit's real client-side
		// router does for this route's purposes: write the new URL into
		// `page.url`, which is what the component's own
		// `$derived(params)` actually reacts to. No real navigation/
		// routing stack is mounted in this test.
		//
		// IN-09: returns a resolved Promise, matching the real `goto`'s
		// Promise<void> return type ($app/navigation) that this mock
		// otherwise stands in for — browse-nav.ts's navigate() calls
		// `.catch()` on gotoFn's return value, which threw
		// "Cannot read properties of undefined (reading 'catch')" against
		// the old undefined-returning version of this mock.
		goto: (url: URL | string) => {
			mockPage.url = typeof url === 'string' ? new URL(url) : new URL(url.href);
			return Promise.resolve();
		}
	};
});

// Deferred lets the test control exactly when a stubbed getNodeDetail
// call resolves, so the POSITIVE case below can observe the transient
// `browse-loading` state before letting the call complete — proving the
// effect actually "re-enters loading", not just inferring it from a
// call count.
interface Deferred<T> {
	promise: Promise<T>;
	resolve: (value: T) => void;
}
function createDeferred<T>(): Deferred<T> {
	let resolve!: (value: T) => void;
	const promise = new Promise<T>((res) => {
		resolve = res;
	});
	return { promise, resolve };
}

function nodeDetailResponse(symbol: string, file: string, line: number): GetNodeDetailResponse {
	return {
		mode: NodeDetailMode.SINGLE_DEF,
		path: '',
		node: {
			id: '1',
			kind: 'func',
			name: symbol,
			qualifiedName: symbol,
			filePath: file,
			startLine: line,
			endLine: line + 3,
			language: 'go'
		},
		calls: [],
		calledBy: [],
		symbol: '',
		definitions: [],
		totalCandidates: 0,
		source: undefined
	} as unknown as GetNodeDetailResponse;
}

const impactResponseStub = {
	depth: 0,
	nodeCount: 0,
	edgeCount: 0,
	affected: []
} as unknown as ImpactResponse;

const permalinkResponseStub = {
	availability: PermalinkAvailability.UNSPECIFIED,
	url: '',
	reason: ''
} as unknown as GetPermalinkResponse;

const emptySearchResponse = { locations: [] } as unknown as SearchResponse;
const emptyFilesResponse = { format: 'flat', files: [], tree: [] } as unknown as FilesResponse;
const emptyExploreResponse = {
	query: '',
	empty: true,
	stale: false,
	symbolCount: 0,
	groups: [],
	blasts: []
} as unknown as ExploreResponse;

const BASE_URL =
	'http://localhost/browse?symbol=resolveSourcePath&file=internal%2Fquery%2Fnode.go&line=33';

beforeEach(() => {
	vi.useFakeTimers();
	resetMockPage(BASE_URL);
});

afterEach(() => {
	// Never advanced (SearchPanel's own 150ms search debounce, IN-13,
	// deliberately never fires in this test) — fake timers just prevent
	// that pending real setTimeout from leaking into a later test file.
	vi.useRealTimers();
});

describe('browse route: typing in search does not tear down the open node view (CR-01)', () => {
	it('leaves the RPC call count and the rendered target unchanged across three keystrokes, but still updates the URL every time, and DOES reload for a target-field change', async () => {
		// 06-03 (deviation, Rule 3): the default 5000ms test timeout leaves
		// no margin once the suite grows past this plan's four new test
		// files — measured in isolation, this test's own body consistently
		// takes ~5.06s regardless of this plan's changes (reproduced against
		// this file's pre-06-03 content too), so the added suite-wide
		// contention was already right at the edge. Widening ONLY this
		// test's own timeout (no logic change) is what the framework's own
		// error message suggests; see 06-03-SUMMARY.md for the full
		// isolation trace that ruled out a regression in this plan's code.
		const nodeDetailReqs: Array<{ symbol: string; file: string }> = [];
		const nodeDetailDeferreds: Array<Deferred<GetNodeDetailResponse>> = [];
		let impactCalls = 0;

		const stubUiClient = {
			getNodeDetail: (req: { symbol: string; file: string }) => {
				nodeDetailReqs.push({ symbol: req.symbol, file: req.file });
				const deferred = createDeferred<GetNodeDetailResponse>();
				nodeDetailDeferreds.push(deferred);
				return deferred.promise;
			},
			impact: () => {
				impactCalls += 1;
				return Promise.resolve(impactResponseStub);
			},
			getPermalink: () => Promise.resolve(permalinkResponseStub),
			search: () => Promise.resolve(emptySearchResponse),
			files: () => Promise.resolve(emptyFilesResponse),
			explore: () => Promise.resolve(emptyExploreResponse)
		};

		vi.doMock('$lib/client', () => ({ uiClient: stubUiClient }));
		const { default: BrowsePage } = await import('../src/routes/browse/+page.svelte');

		const statusGate = {
			subscribe(run: (s: { verdict: string; commit: string; commitSha: string }) => void) {
				run({ verdict: 'ok', commit: 'known', commitSha: 'deadbeef' });
				return () => {};
			},
			notifyNavigated: () => {}
		};

		render(BrowsePage, { context: new Map([['statusGate', statusGate]]) });

		// Initial mount: exactly one GetNodeDetail + one Impact for the
		// symbol/file/line the URL named — the baseline this test's
		// "unchanged" assertions below are measured against.
		await waitFor(() => expect(nodeDetailReqs).toHaveLength(1));
		expect(nodeDetailReqs[0]).toEqual({
			symbol: 'resolveSourcePath',
			file: 'internal/query/node.go'
		});
		nodeDetailDeferreds[0].resolve(nodeDetailResponse('resolveSourcePath', 'internal/query/node.go', 33));
		await waitFor(() => expect(screen.getByTestId('browse-source')).toBeInTheDocument());
		expect(impactCalls).toBe(1);

		const input = screen.getByPlaceholderText('Search symbols and files, or press Enter to ask...');

		// Three keystrokes, three DIFFERENT `q` values — SearchPanel's own
		// undebounced onQueryChange (D-11) writes each one straight into
		// the URL.
		for (const value of ['C', 'Ca', 'Cal']) {
			await fireEvent.input(input, { target: { value } });
			// NEGATIVE (rule 84d1gfpywd requires this be paired with a
			// positive below — see after the loop): the load effect must
			// NOT have re-run for a `q`-only URL change.
			expect(nodeDetailReqs).toHaveLength(1);
			expect(impactCalls).toBe(1);
			expect(screen.queryByTestId('browse-loading')).not.toBeInTheDocument();
			expect(screen.getByTestId('browse-source')).toBeInTheDocument();
			// D-11's shareable-at-every-instant property must survive the
			// fix: the URL still updates on every keystroke even though
			// the load effect no longer reacts to it.
			expect(mockPage.url.searchParams.get('q')).toBe(value);
		}

		// POSITIVE (the pairing this negative test needs to not be
		// vacuous): changing a TARGET field — here, `symbol` — through
		// the exact same URL-write path MUST re-enter loading and MUST
		// re-issue both RPCs. Without this, the assertions above would
		// pass identically against an effect that never runs at all.
		resetMockPage(
			'http://localhost/browse?symbol=otherFunc&file=internal%2Fquery%2Fnode.go&line=33&q=Cal'
		);
		flushSync();

		expect(nodeDetailReqs).toHaveLength(2);
		expect(nodeDetailReqs[1]).toEqual({ symbol: 'otherFunc', file: 'internal/query/node.go' });
		expect(impactCalls).toBe(2);
		expect(screen.getByTestId('browse-loading')).toBeInTheDocument();

		nodeDetailDeferreds[1].resolve(nodeDetailResponse('otherFunc', 'internal/query/node.go', 33));
		await waitFor(() => expect(screen.getByTestId('browse-source')).toBeInTheDocument());
	}, 15000);
});
