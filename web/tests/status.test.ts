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

	it('WR-04: varying ONLY q (Browse\'s view-local search box) produces the SAME identity', () => {
		// q is written on every keystroke, undebounced (+page.svelte:64-66,
		// D-11) — before this exclusion, each character typed minted a
		// fresh navigation identity and fired an extra GetStatus RPC.
		const a = navigationIdentity(new URL('http://localhost/browse?q=hel'));
		const b = navigationIdentity(new URL('http://localhost/browse?q=hello'));
		expect(a).toBe(b);
	});

	it('a target field (symbol) still produces a DIFFERENT identity, paired against the q case above', () => {
		// The positive control for the q-exclusion above: q must be the
		// ONLY thing dropped from the identity — an ordinary target-field
		// change must still be seen as a distinct navigation.
		const a = navigationIdentity(new URL('http://localhost/browse?symbol=Foo'));
		const b = navigationIdentity(new URL('http://localhost/browse?symbol=Bar'));
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

	it('WR-04: notifying through navigationIdentity as q varies fires no extra GetStatus, paired against a target-field change that does', async () => {
		// End-to-end version of the navigationIdentity unit tests above,
		// through the actual gate the layout drives: typing in Browse's
		// search box must never re-trigger GetStatus, while a real
		// navigation (a different symbol) still must.
		const { client, calls } = statusClient(() => Promise.resolve(statusResponse()));
		const gate = createStatusGate(client, navigationIdentity(new URL('http://localhost/browse?q=h')));
		await vi.waitFor(() => expect(calls).toHaveBeenCalledTimes(1));

		for (const q of ['he', 'hel', 'hell', 'hello']) {
			gate.notifyNavigated(navigationIdentity(new URL(`http://localhost/browse?q=${q}`)));
		}
		expect(calls).toHaveBeenCalledTimes(1);

		gate.notifyNavigated(navigationIdentity(new URL('http://localhost/browse?q=hello&symbol=Bar')));
		await vi.waitFor(() => expect(calls).toHaveBeenCalledTimes(2));
	});
});

describe('createStatusGate: response-identity guard (WR-08)', () => {
	it('a stale first response landing AFTER a fresher second response never overwrites the fresher verdict', async () => {
		// Reproduces the ordinary loopback interleaving WR-08 describes:
		// fetch A starts while the index is stale, fetch B starts shortly
		// after (indexing has since finished), and A's response lands
		// AFTER B's. Without a response-identity guard, "last settled
		// wins" reverts the gate to A's stale verdict and it stays there.
		let resolveFirst!: (r: GetStatusResponse) => void;
		let resolveSecond!: (r: GetStatusResponse) => void;
		let callCount = 0;

		const client: StatusClient = {
			getStatus: (): Promise<GetStatusResponse> => {
				callCount += 1;
				if (callCount === 1) {
					return new Promise<GetStatusResponse>((resolve) => {
						resolveFirst = resolve;
					});
				}
				return new Promise<GetStatusResponse>((resolve) => {
					resolveSecond = resolve;
				});
			}
		};

		const gate = createStatusGate(client, '/browse?symbol=Foo');
		const observed: string[] = [];
		gate.subscribe((status) => observed.push(status.verdict));

		// Trigger the second fetch (fetch B) before either resolves.
		gate.notifyNavigated('/browse?symbol=Bar');
		await vi.waitFor(() => expect(callCount).toBe(2));

		// Fetch B (the FRESHER request) resolves first: index is healthy.
		resolveSecond(statusResponse({ initialized: true, stale: false }));
		await vi.waitFor(() => expect(observed.at(-1)).toBe('ok'));

		// Fetch A (the STALE, superseded request) resolves last.
		resolveFirst(statusResponse({ initialized: true, stale: true }));
		// Give the (would-be) stale emit a turn to land if the guard were
		// absent, then assert the fresher verdict is still what's current.
		await new Promise((r) => setTimeout(r, 0));
		expect(observed.at(-1)).toBe('ok');
	});
});
