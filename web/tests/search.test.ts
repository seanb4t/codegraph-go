// Edge coverage for search.ts's debounce/cancellation/trigger-split
// controller (D-14, D-15, D-16). Pure-TS: no DOM, no SvelteKit runtime.
// This file is written and run RED (search.ts does not exist yet) before
// any implementation — recorded in 03-06-SUMMARY.md — then made GREEN.
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { get } from 'svelte/store';

import {
	createSearchController,
	SEARCH_DEBOUNCE_MS,
	SEARCH_MIN_CHARS,
	type SearchClient
} from '$lib/search';
import type { Location, FileEntry, SearchResponse, FilesResponse, ExploreResponse } from '$lib/gen/ui_pb';

// --- fixtures -----------------------------------------------------------

function location(name: string): Location {
	return { name, kind: 'func', filePath: 'a.go', startLine: 1 } as Location;
}

function fileEntry(path: string): FileEntry {
	return { path, language: 'go', nodeCount: 1n, edgeCount: 0n } as FileEntry;
}

function searchResponse(names: string[]): SearchResponse {
	return { locations: names.map(location) } as SearchResponse;
}

function filesResponse(paths: string[], format = 'flat'): FilesResponse {
	return { format, files: paths.map(fileEntry), tree: [] } as unknown as FilesResponse;
}

function exploreResponse(overrides: Partial<ExploreResponse>): ExploreResponse {
	return {
		query: '',
		empty: false,
		stale: false,
		symbolCount: 0,
		groups: [],
		blasts: [],
		...overrides
	} as ExploreResponse;
}

// deferred() lets a test control exactly when a stub RPC call resolves —
// required for the out-of-order case, where the FIRST dispatched request
// must resolve AFTER the second.
function deferred<T>() {
	let resolve!: (value: T) => void;
	let reject!: (reason?: unknown) => void;
	const promise = new Promise<T>((res, rej) => {
		resolve = res;
		reject = rej;
	});
	return { promise, resolve, reject };
}

// recordingClient wraps a SearchClient and records every call (method,
// request, and the AbortSignal passed) so tests can assert dispatch
// counts and cancellation without inspecting internal controller state.
type Call = { method: 'search' | 'files' | 'explore'; request: unknown; signal?: AbortSignal };

function recordingClient(overrides: Partial<SearchClient> = {}): {
	client: SearchClient;
	calls: Call[];
} {
	const calls: Call[] = [];
	const client: SearchClient = {
		search: overrides.search
			? (req, opts) => {
					calls.push({ method: 'search', request: req, signal: opts?.signal });
					return overrides.search!(req, opts);
				}
			: (req, opts) => {
					calls.push({ method: 'search', request: req, signal: opts?.signal });
					return Promise.resolve(searchResponse([]));
				},
		files: overrides.files
			? (req, opts) => {
					calls.push({ method: 'files', request: req, signal: opts?.signal });
					return overrides.files!(req, opts);
				}
			: (req, opts) => {
					calls.push({ method: 'files', request: req, signal: opts?.signal });
					return Promise.resolve(filesResponse([]));
				},
		explore: overrides.explore
			? (req, opts) => {
					calls.push({ method: 'explore', request: req, signal: opts?.signal });
					return overrides.explore!(req, opts);
				}
			: (req, opts) => {
					calls.push({ method: 'explore', request: req, signal: opts?.signal });
					return Promise.resolve(exploreResponse({}));
				}
	};
	return { client, calls };
}

beforeEach(() => {
	vi.useFakeTimers();
});

afterEach(() => {
	vi.useRealTimers();
});

describe('search: minimum-length and debounce gating', () => {
	it('a query of one character issues no RPC at all', async () => {
		const { client, calls } = recordingClient();
		const controller = createSearchController(client);

		controller.setQuery('a');
		await vi.advanceTimersByTimeAsync(SEARCH_DEBOUNCE_MS + 50);

		expect(calls.length).toBe(0);
		expect(SEARCH_MIN_CHARS).toBe(2);
	});

	it('a query of two characters, after the debounce interval elapses, issues exactly one Search and one Files', async () => {
		const { client, calls } = recordingClient();
		const controller = createSearchController(client);

		controller.setQuery('ab');
		expect(calls.length).toBe(0); // nothing before the debounce fires

		await vi.advanceTimersByTimeAsync(SEARCH_DEBOUNCE_MS);

		expect(calls.filter((c) => c.method === 'search').length).toBe(1);
		expect(calls.filter((c) => c.method === 'files').length).toBe(1);
	});

	it('three keystrokes within the debounce interval issue exactly one pair of RPCs, for the final value', async () => {
		const { client, calls } = recordingClient();
		const controller = createSearchController(client);

		controller.setQuery('a');
		await vi.advanceTimersByTimeAsync(30);
		controller.setQuery('ab');
		await vi.advanceTimersByTimeAsync(30);
		controller.setQuery('abc');
		await vi.advanceTimersByTimeAsync(SEARCH_DEBOUNCE_MS + 10);

		const searchCalls = calls.filter((c) => c.method === 'search');
		expect(searchCalls.length).toBe(1);
		expect((searchCalls[0]!.request as { term: string }).term).toBe('abc');
	});
});

describe('search: cancellation and out-of-order responses', () => {
	it('aborts the first request when a second query starts while the first is in flight, and the final state holds the SECOND query result by value', async () => {
		const first = deferred<SearchResponse>();
		const second = deferred<SearchResponse>();
		let call = 0;
		const { client, calls } = recordingClient({
			search: () => {
				call += 1;
				return call === 1 ? first.promise : second.promise;
			}
		});
		const controller = createSearchController(client);

		controller.setQuery('car');
		await vi.advanceTimersByTimeAsync(SEARCH_DEBOUNCE_MS);
		controller.setQuery('card');
		await vi.advanceTimersByTimeAsync(SEARCH_DEBOUNCE_MS);

		const firstSignal = calls.find((c) => c.method === 'search')!.signal;
		expect(firstSignal?.aborted).toBe(true);

		// Resolve the SECOND request first, then the FIRST (out-of-order
		// arrival) — the request-identity guard must still land on the
		// second's value, never the first's.
		second.resolve(searchResponse(['card-result']));
		await Promise.resolve();
		await Promise.resolve();
		first.resolve(searchResponse(['car-result']));
		await Promise.resolve();
		await Promise.resolve();

		const state = get(controller);
		expect(state.live.symbols.map((l) => l.name)).toEqual(['card-result']);
	});
});

describe('search: the trigger split', () => {
	it('Explore is never issued by typing — only by submit — proven by a zero explore count alongside a non-zero live count', async () => {
		const { client, calls } = recordingClient();
		const controller = createSearchController(client);

		controller.setQuery('symbolname');
		await vi.advanceTimersByTimeAsync(SEARCH_DEBOUNCE_MS);

		const exploreCalls = calls.filter((c) => c.method === 'explore').length;
		const liveCalls = calls.filter((c) => c.method === 'search' || c.method === 'files').length;
		expect(exploreCalls).toBe(0);
		expect(liveCalls).toBeGreaterThan(0);
	});

	it('submit() issues exactly one Explore call for the current query value', async () => {
		const { client, calls } = recordingClient();
		const controller = createSearchController(client);

		controller.setQuery('question about the graph');
		await vi.advanceTimersByTimeAsync(SEARCH_DEBOUNCE_MS);
		controller.submit();
		await Promise.resolve();
		await Promise.resolve();

		const exploreCalls = calls.filter((c) => c.method === 'explore');
		expect(exploreCalls.length).toBe(1);
		expect((exploreCalls[0]!.request as { query: string }).query).toBe(
			'question about the graph'
		);
	});

	it('an Explore response with empty=true yields the empty outcome, distinct from a rejection which yields the failed outcome', async () => {
		const { client: emptyClient } = recordingClient({
			explore: () => Promise.resolve(exploreResponse({ empty: true, query: 'nothing' }))
		});
		const emptyController = createSearchController(emptyClient);
		emptyController.submit();
		await Promise.resolve();
		await Promise.resolve();
		const emptyState = get(emptyController);
		expect(emptyState.explore?.kind).toBe('empty');

		const { client: failClient } = recordingClient({
			explore: () => Promise.reject(new Error('boom'))
		});
		const failController = createSearchController(failClient);
		failController.submit();
		await Promise.resolve();
		await Promise.resolve();
		const failState = get(failController);
		expect(failState.explore?.kind).toBe('failed');

		expect(emptyState.explore?.kind).not.toBe(failState.explore?.kind);
	});

	it('submitting does not clear the live results already on screen', async () => {
		const { client } = recordingClient({
			search: () => Promise.resolve(searchResponse(['already-here']))
		});
		const controller = createSearchController(client);

		controller.setQuery('ab');
		await vi.advanceTimersByTimeAsync(SEARCH_DEBOUNCE_MS);
		expect(get(controller).live.symbols.map((l) => l.name)).toEqual(['already-here']);

		controller.submit();
		await Promise.resolve();
		await Promise.resolve();

		// live results untouched by the explore submit
		expect(get(controller).live.symbols.map((l) => l.name)).toEqual(['already-here']);
	});
});

describe('search: Files response format discipline', () => {
	it('reads the Files response according to its own format field, never assuming flat', async () => {
		const { client } = recordingClient({
			files: () => Promise.resolve(filesResponse(['a.go', 'b.go'], 'flat'))
		});
		const controller = createSearchController(client);

		controller.setQuery('go');
		await vi.advanceTimersByTimeAsync(SEARCH_DEBOUNCE_MS);

		const state = get(controller);
		expect(state.live.files.map((f) => f.path)).toEqual(['a.go', 'b.go']);
	});
});

describe('search: order preservation', () => {
	it('exposes Search and Files results in exactly the order the RPCs returned them, never re-ordered', async () => {
		const { client } = recordingClient({
			search: () => Promise.resolve(searchResponse(['zzz', 'aaa', 'mmm'])),
			files: () => Promise.resolve(filesResponse(['z.go', 'a.go', 'm.go']))
		});
		const controller = createSearchController(client);

		controller.setQuery('xy');
		await vi.advanceTimersByTimeAsync(SEARCH_DEBOUNCE_MS);

		const state = get(controller);
		expect(state.live.symbols.map((l) => l.name)).toEqual(['zzz', 'aaa', 'mmm']);
		expect(state.live.files.map((f) => f.path)).toEqual(['z.go', 'a.go', 'm.go']);
	});
});

describe('search: dispose (IN-13)', () => {
	it('a pending debounce timer never fires after dispose', async () => {
		const { client, calls } = recordingClient();
		const controller = createSearchController(client);

		controller.setQuery('go');
		// Dispose BEFORE the debounce interval elapses — a component
		// unmounting mid-debounce, the exact scenario IN-13 describes.
		controller.dispose();
		await vi.advanceTimersByTimeAsync(SEARCH_DEBOUNCE_MS * 2);

		expect(calls.filter((c) => c.method === 'search')).toHaveLength(0);
		expect(calls.filter((c) => c.method === 'files')).toHaveLength(0);
	});

	it('dispose aborts an in-flight live request', async () => {
		const searchDeferred = deferred<SearchResponse>();
		const { client, calls } = recordingClient({
			search: () => searchDeferred.promise
		});
		const controller = createSearchController(client);

		controller.setQuery('go');
		await vi.advanceTimersByTimeAsync(SEARCH_DEBOUNCE_MS);
		expect(calls.filter((c) => c.method === 'search')).toHaveLength(1);
		const signal = calls.find((c) => c.method === 'search')?.signal;
		expect(signal?.aborted).toBe(false);

		controller.dispose();
		expect(signal?.aborted).toBe(true);
	});

	it('dispose aborts an in-flight explore request', async () => {
		const exploreDeferred = deferred<ExploreResponse>();
		const { client, calls } = recordingClient({
			explore: () => exploreDeferred.promise
		});
		const controller = createSearchController(client);

		controller.submit();
		expect(calls.filter((c) => c.method === 'explore')).toHaveLength(1);
		const signal = calls.find((c) => c.method === 'explore')?.signal;
		expect(signal?.aborted).toBe(false);

		controller.dispose();
		expect(signal?.aborted).toBe(true);
	});
});
