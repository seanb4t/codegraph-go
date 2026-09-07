// live-route-refetch.test.ts — 06-03 Task 3: the three non-graph
// consumers (health, browse, workbench's AnalysisPanel) re-fetch through
// the rpcs they already own when a new generation arrives on the live
// store, coalesced with a pending-generation flag rather than
// suppression. Mounted-component tests live here — never in
// live-client.test.ts or live-store.test.ts, which cover pure modules
// with no Svelte import.
//
// Mounting/mocking conventions match web/tests/health-page.test.ts,
// web/tests/browse-page.test.ts and web/tests/workbench-tracer.test.ts
// exactly (mockPage/resetMockPage singleton for $app/state +
// $app/navigation, a per-file $lib/client mock whose methods delegate to
// swappable dispatcher variables).
import { render, screen, waitFor } from '@testing-library/svelte';
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';

import type {
	GetHealthResponse,
	GetNodeDetailResponse,
	GetStatusResponse,
	ImpactResponse,
	CallersResponse,
	WatchGraphEvent
} from '$lib/gen/ui_pb';
import { NodeDetailMode } from '$lib/gen/ui_pb';
import type { IndexStatus, StatusGate } from '$lib/status';
import type { LiveEvent } from '$lib/live/live-client';
import type { LiveStore } from '$lib/live/live-store';

import { mockPage, resetMockPage } from './support/browse-page-state.svelte';

vi.mock('$app/state', async () => {
	const { mockPage } = await import('./support/browse-page-state.svelte');
	return { page: mockPage };
});

vi.mock('$app/navigation', async () => {
	const { mockPage } = await import('./support/browse-page-state.svelte');
	return {
		replaceState: (url: URL | string) => {
			mockPage.url = typeof url === 'string' ? new URL(url, mockPage.url) : new URL(url.href);
		},
		goto: (url: URL | string) => {
			mockPage.url = typeof url === 'string' ? new URL(url) : new URL(url.href);
			return Promise.resolve();
		}
	};
});

let currentGetHealthImpl: () => Promise<GetHealthResponse> = () =>
	Promise.reject(new Error('no getHealth stub configured'));
let currentGetNodeDetailImpl: (req: {
	symbol?: string;
	file?: string;
}) => Promise<GetNodeDetailResponse> = () =>
	Promise.reject(new Error('no getNodeDetail stub configured'));
let currentImpactImpl: () => Promise<ImpactResponse> = () =>
	Promise.resolve({ depth: 0, nodeCount: 0, edgeCount: 0, affected: [] } as unknown as ImpactResponse);
let currentCallersImpl: (req: { symbol: string; limit: number }) => Promise<CallersResponse> = () =>
	Promise.reject(new Error('no callers stub configured'));
let currentGetStatusImpl: () => Promise<GetStatusResponse> = () =>
	Promise.reject(new Error('no getStatus stub configured'));

vi.doMock('$lib/client', () => ({
	uiClient: {
		getHealth: (req: unknown, opts?: { signal?: AbortSignal }) => currentGetHealthImpl(),
		getStatus: (req: unknown, opts?: { signal?: AbortSignal }) => currentGetStatusImpl(),
		getNodeDetail: (req: { symbol?: string; file?: string }) => currentGetNodeDetailImpl(req),
		impact: () => currentImpactImpl(),
		callers: (req: { symbol: string; limit: number }) => currentCallersImpl(req),
		callees: () => Promise.reject(new Error('not used by this test file')),
		getPermalink: () =>
			Promise.resolve({ availability: 0, url: '', reason: '' } as unknown as never),
		search: () => Promise.resolve({ locations: [] } as unknown as never),
		files: () => Promise.resolve({ format: 'flat', files: [], tree: [] } as unknown as never),
		explore: () =>
			Promise.resolve({
				query: '',
				empty: true,
				stale: false,
				symbolCount: 0,
				groups: [],
				blasts: []
			} as unknown as never)
	}
}));

const { default: HealthPage } = await import('../src/routes/health/+page.svelte');
const { default: BrowsePage } = await import('../src/routes/browse/+page.svelte');
const { default: WorkbenchPage } = await import('../src/routes/workbench/+page.svelte');
const { default: StatusPage } = await import('../src/routes/+page.svelte');

function statusResponse(overrides: Partial<GetStatusResponse> = {}): GetStatusResponse {
	return {
		initialized: true,
		version: '1',
		nodeCount: 100n,
		edgeCount: 50n,
		fileCount: 10n,
		stale: false,
		commitSha: 'a'.repeat(40),
		storeExists: true,
		indexingInProgress: false,
		...overrides
	} as GetStatusResponse;
}

function healthResponse(overrides: Partial<GetHealthResponse> = {}): GetHealthResponse {
	return {
		initialized: true,
		version: '7',
		fileCount: 10n,
		nodeCount: 100n,
		edgeCount: 50n,
		dbSizeBytes: 1024n,
		backend: 'pebble',
		filesByLanguage: { go: 5n },
		languages: ['go'],
		nodesByKind: { func: 10n },
		edgesByKind: { calls: 20n },
		pendingChanges: undefined,
		indexHealth: undefined,
		worktreeMismatch: undefined,
		stale: false,
		commitSha: 'a'.repeat(40),
		...overrides
	} as GetHealthResponse;
}

function nodeDetailResponse(symbol: string, file: string): GetNodeDetailResponse {
	return {
		mode: NodeDetailMode.SINGLE_DEF,
		path: '',
		node: {
			id: '1',
			kind: 'func',
			name: symbol,
			qualifiedName: symbol,
			filePath: file,
			startLine: 1,
			endLine: 4,
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

function watchGraphEvent(overrides: Partial<WatchGraphEvent> = {}): WatchGraphEvent {
	return {
		generation: 1n,
		initialized: true,
		stale: false,
		storeExists: true,
		indexingInProgress: false,
		commitSha: '',
		...overrides
	} as WatchGraphEvent;
}

function fakeStatusGate(status: Partial<IndexStatus> = {}): StatusGate {
	const full: IndexStatus = { verdict: 'ok', commit: 'known', commitSha: '', ...status };
	return {
		subscribe(run) {
			run(full);
			return () => {};
		},
		notifyNavigated: () => {},
		applyLiveEvent: () => {}
	};
}

/** fakeLiveStore delivers already-admitted events directly — the
 * generation gate itself is Task 2's own tested concern
 * (live-store.test.ts); this route-level harness only needs to prove
 * how a route REACTS to an admitted delivery. */
function fakeLiveStore(): LiveStore & { deliver: (live: LiveEvent) => void } {
	const listeners = new Set<(live: LiveEvent | null) => void>();
	let current: LiveEvent | null = null;
	return {
		subscribe(run) {
			listeners.add(run);
			run(current);
			return () => {
				listeners.delete(run);
			};
		},
		deliver(live) {
			current = live;
			for (const listener of listeners) listener(current);
		}
	};
}

beforeEach(() => {
	resetMockPage('http://localhost/browse');
});

afterEach(() => {
	vi.restoreAllMocks();
});

// 06-08: the root Status route was the ONLY index-data view this file did
// not cover, and it was the only one that did not subscribe. That symmetry
// is not a coincidence — the omission in the suite is why 464 green tests
// let a landing page render `Stale: no` over a generation-old count.
describe('+page.svelte (root Status route): live-triggered re-fetch (LIV-02)', () => {
	it('delivering one live event issues exactly ONE additional status call; a replayed lower generation issues none', async () => {
		let calls = 0;
		currentGetStatusImpl = () => {
			calls += 1;
			return Promise.resolve(statusResponse());
		};
		const live = fakeLiveStore();
		render(StatusPage, {
			context: new Map<string, unknown>([['liveStore', live]])
		});

		await waitFor(() => expect(screen.getByText('Index is healthy.')).toBeInTheDocument());
		expect(calls).toBe(1); // the mount-time fetch only

		live.deliver({ event: watchGraphEvent({ generation: 5n }), epoch: 1 });
		await waitFor(() => expect(calls).toBe(2));

		live.deliver({ event: watchGraphEvent({ generation: 3n }), epoch: 1 }); // replayed lower
		await new Promise((r) => setTimeout(r, 0));
		expect(calls).toBe(2); // unchanged
	});

	it('the pending-generation coalescer: an event during an in-flight re-fetch issues no second concurrent call, and exactly one follow-up carries the newer generation once it settles', async () => {
		let calls = 0;
		let resolveSecond!: (r: GetStatusResponse) => void;
		let resolveThird!: (r: GetStatusResponse) => void;
		currentGetStatusImpl = () => {
			calls += 1;
			if (calls === 1) return Promise.resolve(statusResponse());
			if (calls === 2) return new Promise((resolve) => (resolveSecond = resolve));
			return new Promise((resolve) => (resolveThird = resolve));
		};
		const live = fakeLiveStore();
		render(StatusPage, {
			context: new Map<string, unknown>([['liveStore', live]])
		});
		await waitFor(() => expect(calls).toBe(1));

		live.deliver({ event: watchGraphEvent({ generation: 5n }), epoch: 1 });
		await waitFor(() => expect(calls).toBe(2)); // the live-triggered fetch is now in flight

		live.deliver({ event: watchGraphEvent({ generation: 6n }), epoch: 1 }); // arrives DURING it
		await new Promise((r) => setTimeout(r, 0));
		expect(calls).toBe(2); // no second concurrent call

		resolveSecond(statusResponse());
		await waitFor(() => expect(calls).toBe(3)); // exactly one follow-up, for generation 6

		resolveThird(statusResponse());
	});

	it('CR-01 regression: a live event arriving while the mount fetch is still unresolved must not let the stale mount response overwrite the newer live-triggered one', async () => {
		let calls = 0;
		let resolveMount!: (r: GetStatusResponse) => void;
		let resolveLive!: (r: GetStatusResponse) => void;
		currentGetStatusImpl = () => {
			calls += 1;
			if (calls === 1) return new Promise((resolve) => (resolveMount = resolve));
			return new Promise((resolve) => (resolveLive = resolve));
		};
		const live = fakeLiveStore();
		render(StatusPage, {
			context: new Map<string, unknown>([['liveStore', live]])
		});
		await waitFor(() => expect(calls).toBe(1)); // mount fetch issued, still unresolved

		live.deliver({ event: watchGraphEvent({ generation: 5n }), epoch: 1 });
		await waitFor(() => expect(calls).toBe(2)); // live fetch issued while mount fetch in flight

		resolveLive(statusResponse({ commitSha: 'b'.repeat(40) }));
		await waitFor(() => expect(screen.getByText('b'.repeat(40))).toBeInTheDocument());

		// The stale mount fetch finally resolves — it must NOT overwrite the
		// newer live-triggered response with its now-stale data.
		resolveMount(statusResponse({ commitSha: 'a'.repeat(40) }));
		await new Promise((r) => setTimeout(r, 0));
		expect(screen.getByText('b'.repeat(40))).toBeInTheDocument();
		expect(screen.queryByText('a'.repeat(40))).toBeNull();
	});
});

describe('health/+page.svelte: live-triggered re-fetch (LIV-02)', () => {
	it('delivering one live event issues exactly ONE additional health call; a replayed lower generation issues none', async () => {
		let calls = 0;
		currentGetHealthImpl = () => {
			calls += 1;
			return Promise.resolve(healthResponse());
		};
		const live = fakeLiveStore();
		render(HealthPage, {
			context: new Map<string, unknown>([
				['statusGate', fakeStatusGate()],
				['liveStore', live]
			])
		});

		await waitFor(() => expect(screen.getByTestId('health-freshness')).toBeInTheDocument());
		expect(calls).toBe(1); // the mount-time fetch only

		// Baseline delivery (subscribe's synchronous initial call) already
		// happened during render/mount — this is the FIRST genuinely NEW
		// event.
		live.deliver({ event: watchGraphEvent({ generation: 5n }), epoch: 1 });
		await waitFor(() => expect(calls).toBe(2));

		live.deliver({ event: watchGraphEvent({ generation: 3n }), epoch: 1 }); // replayed lower
		await new Promise((r) => setTimeout(r, 0));
		expect(calls).toBe(2); // unchanged
	});

	it('the pending-generation coalescer: an event during an in-flight re-fetch issues no second concurrent call, and exactly one follow-up carries the newer generation once it settles', async () => {
		let calls = 0;
		let resolveSecond!: (r: GetHealthResponse) => void;
		let resolveThird!: (r: GetHealthResponse) => void;
		currentGetHealthImpl = () => {
			calls += 1;
			if (calls === 1) return Promise.resolve(healthResponse());
			if (calls === 2) return new Promise((resolve) => (resolveSecond = resolve));
			return new Promise((resolve) => (resolveThird = resolve));
		};
		const live = fakeLiveStore();
		render(HealthPage, {
			context: new Map<string, unknown>([
				['statusGate', fakeStatusGate()],
				['liveStore', live]
			])
		});
		await waitFor(() => expect(calls).toBe(1));

		live.deliver({ event: watchGraphEvent({ generation: 5n }), epoch: 1 });
		await waitFor(() => expect(calls).toBe(2)); // the live-triggered fetch is now in flight

		live.deliver({ event: watchGraphEvent({ generation: 6n }), epoch: 1 }); // arrives DURING the in-flight fetch
		await new Promise((r) => setTimeout(r, 0));
		expect(calls).toBe(2); // no second concurrent call

		resolveSecond(healthResponse());
		await waitFor(() => expect(calls).toBe(3)); // exactly one follow-up, for generation 6

		resolveThird(healthResponse());
	});

	it('CR-01 regression: a live event arriving while the mount fetch is still unresolved must not let the stale mount response overwrite the newer live-triggered one', async () => {
		let calls = 0;
		let resolveMount!: (r: GetHealthResponse) => void;
		let resolveLive!: (r: GetHealthResponse) => void;
		currentGetHealthImpl = () => {
			calls += 1;
			if (calls === 1) return new Promise((resolve) => (resolveMount = resolve));
			return new Promise((resolve) => (resolveLive = resolve));
		};
		const live = fakeLiveStore();
		render(HealthPage, {
			context: new Map<string, unknown>([
				['statusGate', fakeStatusGate()],
				['liveStore', live]
			])
		});
		await waitFor(() => expect(calls).toBe(1)); // mount fetch issued, still unresolved

		// A real re-index publishes a new generation BEFORE the mount fetch
		// settles — the exact race window CR-01 describes.
		live.deliver({ event: watchGraphEvent({ generation: 5n }), epoch: 1 });
		await waitFor(() => expect(calls).toBe(2)); // live-triggered fetch issued while mount fetch still in flight

		// The live-triggered fetch resolves FIRST (plausible: it started
		// later against an already-unlocked store while the mount fetch may
		// still be blocked behind a store lock retry).
		resolveLive(healthResponse({ commitSha: 'b'.repeat(40) }));
		await waitFor(() =>
			expect(screen.getByTestId('health-commit-sha')).toHaveTextContent('b'.repeat(40))
		);

		// The stale mount fetch finally resolves — it must NOT overwrite the
		// newer live-triggered response with its now-stale data.
		resolveMount(healthResponse({ commitSha: 'a'.repeat(40) }));
		await new Promise((r) => setTimeout(r, 0));
		expect(screen.getByTestId('health-commit-sha')).toHaveTextContent('b'.repeat(40));
	});
});

describe('workbench AnalysisPanel: live-triggered re-fetch (LIV-02)', () => {
	it('issues no call on a live event when it holds no result, but DOES re-issue when it holds one', async () => {
		mockPage.url = new URL('http://localhost/workbench?mode=callers');
		let callersCalls = 0;
		currentCallersImpl = () => {
			callersCalls += 1;
			return Promise.resolve({ symbol: 'Foo', callers: [] } as unknown as CallersResponse);
		};
		const live = fakeLiveStore();
		render(WorkbenchPage, {
			context: new Map<string, unknown>([
				['statusGate', fakeStatusGate()],
				['liveStore', live]
			])
		});

		// No symbol entered — the panel is idle, holding no result.
		live.deliver({ event: watchGraphEvent({ generation: 1n }), epoch: 1 });
		await new Promise((r) => setTimeout(r, 0));
		expect(callersCalls).toBe(0);

		// Now drive a symbol through the URL directly (mirrors how the
		// component reads page.url) so the panel dispatches and loads.
		mockPage.url = new URL('http://localhost/workbench?mode=callers&symbol=Foo');
		await waitFor(() => expect(callersCalls).toBe(1));
		await waitFor(() => expect(screen.queryByTestId('workbench-loading')).not.toBeInTheDocument());

		live.deliver({ event: watchGraphEvent({ generation: 2n }), epoch: 1 }); // now holds a result
		await waitFor(() => expect(callersCalls).toBe(2));
	});
});

describe('browse/+page.svelte: live-triggered re-fetch (LIV-02)', () => {
	it('a new generation re-issues the open node view through the SAME getNodeDetail call it already owns', async () => {
		resetMockPage('http://localhost/browse?symbol=Foo&file=a.go&line=1');
		let nodeDetailCalls = 0;
		currentGetNodeDetailImpl = (req) => {
			nodeDetailCalls += 1;
			return Promise.resolve(nodeDetailResponse(req.symbol ?? 'Foo', req.file ?? 'a.go'));
		};
		const live = fakeLiveStore();
		render(BrowsePage, {
			context: new Map<string, unknown>([
				['statusGate', fakeStatusGate()],
				['liveStore', live]
			])
		});
		await waitFor(() => expect(nodeDetailCalls).toBe(1));

		live.deliver({ event: watchGraphEvent({ generation: 5n }), epoch: 1 });
		await waitFor(() => expect(nodeDetailCalls).toBe(2));
	});
});
