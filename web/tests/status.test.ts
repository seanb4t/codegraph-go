// status.test.ts — 03-09 Task 1: the shared status gate. Written and run
// RED before web/src/lib/status.ts exists (recorded in the SUMMARY),
// then made GREEN.
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

import {
	classifyStatus,
	navigationIdentity,
	createStatusGate,
	type StatusClient
} from '$lib/status';
import type { GetStatusResponse } from '$lib/gen/ui_pb';

function statusResponse(overrides: Partial<GetStatusResponse> = {}): GetStatusResponse {
	return {
		initialized: false,
		version: '',
		nodeCount: 0,
		edgeCount: 0,
		fileCount: 0,
		stale: false,
		commitSha: '',
		storeExists: false,
		indexingInProgress: false,
		...overrides
	} as GetStatusResponse;
}

function statusClient(impl: () => Promise<GetStatusResponse>): {
	client: StatusClient;
	calls: ReturnType<typeof vi.fn>;
} {
	const calls = vi.fn(impl);
	return { client: { getStatus: calls }, calls };
}

describe('classifyStatus: health verdicts from the field combination', () => {
	it('initialized true, stale false classifies as ok', () => {
		const status = classifyStatus(statusResponse({ initialized: true, stale: false }));
		expect(status.verdict).toBe('ok');
	});

	it('initialized true, stale true classifies as stale', () => {
		const status = classifyStatus(statusResponse({ initialized: true, stale: true }));
		expect(status.verdict).toBe('stale');
	});

	it('initialized false, store_exists false classifies as no-index', () => {
		const status = classifyStatus(
			statusResponse({ initialized: false, storeExists: false, indexingInProgress: false })
		);
		expect(status.verdict).toBe('no-index');
	});

	it('initialized false, store_exists true, indexing_in_progress true classifies as indexing', () => {
		const status = classifyStatus(
			statusResponse({ initialized: false, storeExists: true, indexingInProgress: true })
		);
		expect(status.verdict).toBe('indexing');
	});

	it('no-index and indexing are DIFFERENT verdicts from the same initialized:false starting point — never collapsed by reading initialized alone', () => {
		const noIndex = classifyStatus(
			statusResponse({ initialized: false, storeExists: false, indexingInProgress: false })
		);
		const indexing = classifyStatus(
			statusResponse({ initialized: false, storeExists: true, indexingInProgress: true })
		);
		expect(noIndex.verdict).toBe('no-index');
		expect(indexing.verdict).toBe('indexing');
		expect(noIndex.verdict).not.toBe(indexing.verdict);
	});
});

describe('classifyStatus: commit knowledge is orthogonal to the health verdict', () => {
	it('an empty commit_sha yields commit: unknown paired with a populated commit_sha yielding commit: known — the two results differ in exactly one field', () => {
		const noSha = classifyStatus(
			statusResponse({ initialized: true, stale: false, commitSha: '' })
		);
		const withSha = classifyStatus(
			statusResponse({ initialized: true, stale: false, commitSha: 'deadbeef' })
		);
		expect(noSha).toEqual({ verdict: 'ok', commit: 'unknown' });
		expect(withSha).toEqual({ verdict: 'ok', commit: 'known' });
		// Exactly one field differs between the two.
		expect(noSha.verdict).toBe(withSha.verdict);
		expect(noSha.commit).not.toBe(withSha.commit);
	});

	it('a stale response with an empty commit_sha classifies as verdict stale, commit unknown — the two dimensions vary independently', () => {
		const status = classifyStatus(
			statusResponse({ initialized: true, stale: true, commitSha: '' })
		);
		expect(status).toEqual({ verdict: 'stale', commit: 'unknown' });
	});
});

describe('navigationIdentity: pathname plus sorted query string', () => {
	it('two orderings of the same query string produce the same identity', () => {
		const a = navigationIdentity(new URL('http://localhost/browse?b=2&a=1'));
		const b = navigationIdentity(new URL('http://localhost/browse?a=1&b=2'));
		expect(a).toBe(b);
	});

	it('a different path produces a different identity', () => {
		const a = navigationIdentity(new URL('http://localhost/browse?symbol=Foo'));
		const b = navigationIdentity(new URL('http://localhost/graph?symbol=Foo'));
		expect(a).not.toBe(b);
	});
});

describe('createStatusGate: a rejected GetStatus call never throws', () => {
	it('classifies as verdict unknown and does not propagate the rejection', async () => {
		const { client } = statusClient(() => Promise.reject(new Error('transport down')));
		let received: unknown;
		expect(() => {
			const gate = createStatusGate(client, '/browse');
			gate.subscribe((status) => {
				received = status;
			});
		}).not.toThrow();
		await vi.waitFor(() => {
			expect(received).toEqual({ verdict: 'unknown', commit: 'unknown' });
		});
	});
});

describe('createStatusGate: fetches on load and navigation only — no polling', () => {
	beforeEach(() => {
		vi.useFakeTimers();
	});
	afterEach(() => {
		vi.useRealTimers();
	});

	it('the call count is unchanged across a long span of simulated time with no navigation, and increases only after a navigation notification', async () => {
		const { client, calls } = statusClient(() => Promise.resolve(statusResponse()));
		const gate = createStatusGate(client, '/browse');
		await vi.advanceTimersByTimeAsync(0);
		expect(calls).toHaveBeenCalledTimes(1);

		// Advance well past any plausible polling interval.
		await vi.advanceTimersByTimeAsync(10 * 60 * 1000);
		expect(calls).toHaveBeenCalledTimes(1);

		gate.notifyNavigated('/graph');
		await vi.advanceTimersByTimeAsync(0);
		expect(calls).toHaveBeenCalledTimes(2);
	});
});

describe('createStatusGate: the identity guard', () => {
	it('1 after creation, still 1 after notifying with the SAME identity, 2 after notifying with a DIFFERENT identity', async () => {
		const { client, calls } = statusClient(() => Promise.resolve(statusResponse()));
		const identityA = '/browse?symbol=Foo';
		const identityB = '/browse?symbol=Bar';

		const gate = createStatusGate(client, identityA);
		await vi.waitFor(() => expect(calls).toHaveBeenCalledTimes(1));

		gate.notifyNavigated(identityA);
		expect(calls).toHaveBeenCalledTimes(1);

		gate.notifyNavigated(identityB);
		await vi.waitFor(() => expect(calls).toHaveBeenCalledTimes(2));
	});
});
