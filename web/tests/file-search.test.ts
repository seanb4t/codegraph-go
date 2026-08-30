// file-search.test.ts, written and run RED before file-search.ts exists,
// mirroring search.test.ts's structure (04-06 Task 1). The nested-result
// assertion is the observable proof that 04-02's recursive-glob fix
// (internal/query/files.go) reaches this second Files caller, and the
// metacharacter-escaping assertions are the observable proof that a
// typed `[`, `{`, `*`, `?` or `\` is matched literally rather than
// erroring through Engine.Files' pre-scan sanity check or silently
// matching something else (cycle-1 review finding, HIGH).
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { get } from 'svelte/store';

import { createFileSearchController, escapeGlobLiteral, type FilesClient } from '$lib/file-search';
import { SEARCH_DEBOUNCE_MS, SEARCH_MIN_CHARS } from '$lib/search';
import type { FileEntry, FilesResponse } from '$lib/gen/ui_pb';

// --- fixtures -----------------------------------------------------------

function fileEntry(path: string): FileEntry {
	return { path, language: 'go', nodeCount: 1n, edgeCount: 0n } as FileEntry;
}

function filesResponse(paths: string[], format = 'flat'): FilesResponse {
	return { format, files: paths.map(fileEntry), tree: [] } as unknown as FilesResponse;
}

function deferred<T>() {
	let resolve!: (value: T) => void;
	let reject!: (reason?: unknown) => void;
	const promise = new Promise<T>((res, rej) => {
		resolve = res;
		reject = rej;
	});
	return { promise, resolve, reject };
}

type FilesRequest = { pattern: string; filter: string; dir: string; depth: number; format: string };
type Call = { request: FilesRequest; signal?: AbortSignal };

function recordingClient(
	impl?: (req: FilesRequest, opts?: { signal?: AbortSignal }) => Promise<FilesResponse>
): { client: FilesClient; calls: Call[] } {
	const calls: Call[] = [];
	const client: FilesClient = {
		files: (req, opts) => {
			calls.push({ request: req as FilesRequest, signal: opts?.signal });
			return impl ? impl(req as FilesRequest, opts) : Promise.resolve(filesResponse([]));
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

describe('file-search: minimum-length and debounce gating', () => {
	it('a term shorter than SEARCH_MIN_CHARS issues no request', async () => {
		const { client, calls } = recordingClient();
		const controller = createFileSearchController(client);

		controller.setQuery('a');
		await vi.advanceTimersByTimeAsync(SEARCH_DEBOUNCE_MS + 50);

		expect(calls).toHaveLength(0);
		expect(SEARCH_MIN_CHARS).toBe(2);
	});

	it('a term at least SEARCH_MIN_CHARS long issues exactly one request after the debounce interval, in flat format', async () => {
		const { client, calls } = recordingClient();
		const controller = createFileSearchController(client);

		controller.setQuery('go');
		expect(calls).toHaveLength(0); // nothing before the debounce fires

		await vi.advanceTimersByTimeAsync(SEARCH_DEBOUNCE_MS);

		expect(calls).toHaveLength(1);
		expect(calls[0]!.request.format).toBe('flat');
	});
});

describe('file-search: nested results (the 04-02 dependency, made observable)', () => {
	it('a result nested several directories deep appears alongside a root-level result', async () => {
		const { client } = recordingClient(() =>
			Promise.resolve(filesResponse(['files.go', 'internal/query/files.go']))
		);
		const controller = createFileSearchController(client);

		controller.setQuery('files');
		await vi.advanceTimersByTimeAsync(SEARCH_DEBOUNCE_MS);

		const paths = get(controller).results.map((f) => f.path);
		expect(paths).toContain('files.go');
		expect(paths).toContain('internal/query/files.go');
	});
});

describe('file-search: debounce coalescing vs. genuine overlap', () => {
	it('three keystrokes within the debounce window issue exactly one dispatch, with a single un-aborted signal', async () => {
		const { client, calls } = recordingClient();
		const controller = createFileSearchController(client);

		controller.setQuery('fi');
		await vi.advanceTimersByTimeAsync(30);
		controller.setQuery('fil');
		await vi.advanceTimersByTimeAsync(30);
		controller.setQuery('file');
		await vi.advanceTimersByTimeAsync(SEARCH_DEBOUNCE_MS + 10);

		expect(calls).toHaveLength(1);
		expect(calls[0]!.signal?.aborted).toBe(false);
	});

	it('a genuinely overlapping dispatch aborts the first request and leaves the second un-aborted', async () => {
		const first = deferred<FilesResponse>();
		const second = deferred<FilesResponse>();
		let call = 0;
		const { client, calls } = recordingClient(() => {
			call += 1;
			return call === 1 ? first.promise : second.promise;
		});
		const controller = createFileSearchController(client);

		controller.setQuery('fi');
		await vi.advanceTimersByTimeAsync(SEARCH_DEBOUNCE_MS);
		controller.setQuery('file');
		await vi.advanceTimersByTimeAsync(SEARCH_DEBOUNCE_MS);

		expect(calls).toHaveLength(2);
		expect(calls[0]!.signal?.aborted).toBe(true);
		expect(calls[1]!.signal?.aborted).toBe(false);
	});
});

describe('file-search: out-of-order safety', () => {
	it('a stale response for a shorter prefix never overwrites a newer result', async () => {
		const first = deferred<FilesResponse>();
		const second = deferred<FilesResponse>();
		let call = 0;
		const { client } = recordingClient(() => {
			call += 1;
			return call === 1 ? first.promise : second.promise;
		});
		const controller = createFileSearchController(client);

		controller.setQuery('fi');
		await vi.advanceTimersByTimeAsync(SEARCH_DEBOUNCE_MS);
		controller.setQuery('file');
		await vi.advanceTimersByTimeAsync(SEARCH_DEBOUNCE_MS);

		second.resolve(filesResponse(['file.go']));
		await Promise.resolve();
		await Promise.resolve();
		first.resolve(filesResponse(['fi.go']));
		await Promise.resolve();
		await Promise.resolve();

		expect(get(controller).results.map((f) => f.path)).toEqual(['file.go']);
	});
});

describe('file-search: backspace below the minimum while a request is in flight', () => {
	it('backspacing below the minimum while in flight aborts the request, clears the results, and never lets the stale response land', async () => {
		const inflight = deferred<FilesResponse>();
		const { client, calls } = recordingClient(() => inflight.promise);
		const controller = createFileSearchController(client);

		controller.setQuery('file');
		await vi.advanceTimersByTimeAsync(SEARCH_DEBOUNCE_MS);
		expect(calls).toHaveLength(1);

		controller.setQuery('f'); // below the minimum, WHILE the request above is still unresolved

		expect(calls[0]!.signal?.aborted).toBe(true);
		expect(get(controller).results).toEqual([]);

		inflight.resolve(filesResponse(['file.go']));
		await Promise.resolve();
		await Promise.resolve();

		expect(get(controller).results).toEqual([]);
	});
});

describe('file-search: rejection handling', () => {
	it('a rejection sets a failure state classified by classifyRpcError, without throwing', async () => {
		const { client } = recordingClient(() => Promise.reject(new Error('boom')));
		const controller = createFileSearchController(client);

		controller.setQuery('file');
		await vi.advanceTimersByTimeAsync(SEARCH_DEBOUNCE_MS);
		await Promise.resolve();
		await Promise.resolve();

		const state = get(controller);
		expect(state.failure?.kind).toBe('unknown');
		expect(state.failure?.message).toBe('boom');
	});
});

describe('file-search: Files response format discipline', () => {
	it('a tree-format response yields zero flat results rather than crashing', async () => {
		const { client } = recordingClient(() =>
			Promise.resolve({ format: 'tree', files: [], tree: [] } as unknown as FilesResponse)
		);
		const controller = createFileSearchController(client);

		controller.setQuery('file');
		await vi.advanceTimersByTimeAsync(SEARCH_DEBOUNCE_MS);

		expect(get(controller).results).toEqual([]);
	});
});

describe('file-search: glob metacharacter escaping (cycle-1 review finding)', () => {
	const metacharacters = ['[', '{', '*', '?', '\\'];

	it.each(metacharacters)(
		'a term containing %s produces a request pattern with that character backslash-escaped',
		async (mc) => {
			const { client, calls } = recordingClient();
			const controller = createFileSearchController(client);

			const term = `foo${mc}bar`;
			controller.setQuery(term);
			await vi.advanceTimersByTimeAsync(SEARCH_DEBOUNCE_MS);

			const expectedEscaped = `foo\\${mc}bar`;
			expect(calls[0]!.request.pattern).toBe(`**/*${expectedEscaped}*`);
		}
	);

	it('escapeGlobLiteral leaves an ordinary term byte-identical', () => {
		expect(escapeGlobLiteral('plainname.go')).toBe('plainname.go');
	});
});
