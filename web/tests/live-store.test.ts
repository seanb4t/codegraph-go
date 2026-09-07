// live-store.test.ts — 06-03 Task 2: one classifier for both the unary
// GetStatus answer and a live WatchGraphEvent, StatusGate's new
// applyLiveEvent method (the one-counter invariant against fetchStatus),
// and live-store.ts's epoch-scoped generation gate plus its appliedAtMs
// stamping. Written and run RED before status.ts's widened classifyStatus
// signature / applyLiveEvent method and live-store.ts existed (recorded
// in the SUMMARY), then made GREEN.
import { describe, it, expect, vi } from 'vitest';

import { classifyStatus, createStatusGate, type StatusClient } from '$lib/status';
import type { GetStatusResponse, WatchGraphEvent } from '$lib/gen/ui_pb';
import { createLiveStore } from '$lib/live/live-store';
import type { LiveEvent } from '$lib/live/live-client';

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

function watchGraphEvent(overrides: Partial<WatchGraphEvent> = {}): WatchGraphEvent {
	return {
		generation: 1n,
		initialized: false,
		stale: false,
		storeExists: false,
		indexingInProgress: false,
		commitSha: '',
		...overrides
	} as WatchGraphEvent;
}

function liveEvent(generation: bigint, epoch: number, overrides: Partial<WatchGraphEvent> = {}): LiveEvent {
	return { event: watchGraphEvent({ generation, ...overrides }), epoch };
}

function statusClient(impl: () => Promise<GetStatusResponse>): {
	client: StatusClient;
	calls: ReturnType<typeof vi.fn>;
} {
	const calls = vi.fn(impl);
	return { client: { getStatus: calls }, calls };
}

function makeFakeClient(): {
	start: (onEvent: (live: LiveEvent) => void) => { stop(): void };
	deliver: (live: LiveEvent) => void;
	stopped: boolean;
} {
	let handler: ((live: LiveEvent) => void) | null = null;
	const state = { stopped: false };
	return {
		start(onEvent) {
			handler = onEvent;
			return {
				stop() {
					state.stopped = true;
				}
			};
		},
		deliver(live) {
			handler?.(live);
		},
		get stopped() {
			return state.stopped;
		}
	};
}

function clearObservations(): void {
	delete (window as unknown as { __codegraphLiveObservations?: unknown })
		.__codegraphLiveObservations;
}

describe('classifyStatus: one classifier for both a unary response and a live event', () => {
	const combos: Array<Pick<GetStatusResponse, 'initialized' | 'stale' | 'storeExists' | 'indexingInProgress' | 'commitSha'>> = [
		{ initialized: true, stale: false, storeExists: true, indexingInProgress: false, commitSha: '' },
		{ initialized: true, stale: true, storeExists: true, indexingInProgress: false, commitSha: 'a'.repeat(40) },
		{ initialized: false, stale: false, storeExists: false, indexingInProgress: false, commitSha: '' },
		{ initialized: false, stale: false, storeExists: true, indexingInProgress: true, commitSha: '' },
		{ initialized: false, stale: false, storeExists: true, indexingInProgress: false, commitSha: '' }, // unrecognized -> 'unknown'
		{ initialized: true, stale: false, storeExists: false, indexingInProgress: true, commitSha: 'b'.repeat(40) }
	];

	it('produces the SAME verdict for a GetStatusResponse-shaped value and a WatchGraphEvent-shaped value, across at least 6 field combinations', () => {
		expect(combos.length).toBeGreaterThanOrEqual(6);
		for (const combo of combos) {
			const fromResponse = classifyStatus(statusResponse(combo));
			const fromEvent = classifyStatus(watchGraphEvent({ generation: 42n, ...combo }));
			expect(fromEvent).toEqual(fromResponse);
		}
	});

	it('an unrecognized combination degrades to "unknown" for BOTH input shapes, never a sixth verdict', () => {
		const combo = {
			initialized: false,
			stale: false,
			storeExists: true,
			indexingInProgress: false,
			commitSha: ''
		} as const;
		expect(classifyStatus(statusResponse(combo)).verdict).toBe('unknown');
		expect(classifyStatus(watchGraphEvent({ generation: 1n, ...combo })).verdict).toBe('unknown');
	});
});

describe('StatusGate.applyLiveEvent: no round trip', () => {
	it("the stub status client's call count is unchanged across a live application, while the emitted verdict DOES change", async () => {
		const { client, calls } = statusClient(() => Promise.resolve(statusResponse()));
		const gate = createStatusGate(client, '/browse');
		await vi.waitFor(() => expect(calls).toHaveBeenCalledTimes(1));

		const observed: string[] = [];
		gate.subscribe((s) => observed.push(s.verdict));
		expect(observed.at(-1)).toBe('no-index'); // default statusResponse() classifies no-index

		gate.applyLiveEvent({
			initialized: true,
			stale: true,
			storeExists: true,
			indexingInProgress: false,
			commitSha: '',
			generation: 7n,
			epoch: 1
		});

		expect(calls).toHaveBeenCalledTimes(1); // unchanged — no getStatus call issued
		expect(observed.at(-1)).toBe('stale'); // the verdict DID change
	});
});

describe('StatusGate: the one-counter invariant orders live applications against unary fetches', () => {
	it('(a) a live event applied while a fetch is in flight invalidates that fetch — its response is dropped when it settles — paired with the same response emitting when no live event intervenes', async () => {
		// Positive control first: no live event intervenes, the fetch's own
		// response DOES emit.
		{
			let resolveFetch!: (r: GetStatusResponse) => void;
			const client: StatusClient = {
				getStatus: () => new Promise((resolve) => (resolveFetch = resolve))
			};
			const gate = createStatusGate(client, '/browse');
			const observed: string[] = [];
			gate.subscribe((s) => observed.push(s.verdict));
			resolveFetch(statusResponse({ initialized: true, stale: false }));
			await vi.waitFor(() => expect(observed.at(-1)).toBe('ok'));
		}

		// Now the case under test: a live event lands WHILE the constructor's
		// own fetch is still in flight.
		let resolveFetch!: (r: GetStatusResponse) => void;
		const client: StatusClient = {
			getStatus: () => new Promise((resolve) => (resolveFetch = resolve))
		};
		const gate = createStatusGate(client, '/browse');
		const observed: string[] = [];
		gate.subscribe((s) => observed.push(s.verdict));

		gate.applyLiveEvent({
			initialized: true,
			stale: false,
			storeExists: true,
			indexingInProgress: false,
			commitSha: '',
			generation: 1n,
			epoch: 1
		});
		expect(observed.at(-1)).toBe('ok');

		// The in-flight fetch settles with a DIFFERENT verdict — it must be
		// dropped, since it was superseded by the live application above.
		resolveFetch(statusResponse({ initialized: true, stale: true }));
		await new Promise((r) => setTimeout(r, 0));
		expect(observed.at(-1)).toBe('ok');
	});

	it('(b) a fetch started after a live application causes a LATE DUPLICATE of that live event to be dropped, paired with the fetch\'s own response emitting', async () => {
		let callCount = 0;
		let resolveSecond!: (r: GetStatusResponse) => void;
		const client: StatusClient = {
			getStatus: () => {
				callCount += 1;
				if (callCount === 1) return Promise.resolve(statusResponse({ initialized: true, stale: false }));
				return new Promise((resolve) => (resolveSecond = resolve));
			}
		};
		const gate = createStatusGate(client, '/browse');
		await vi.waitFor(() => expect(callCount).toBe(1));

		const observed: string[] = [];
		gate.subscribe((s) => observed.push(s.verdict));

		const liveFields = {
			initialized: true,
			stale: true,
			storeExists: true,
			indexingInProgress: false,
			commitSha: '',
			generation: 7n,
			epoch: 2
		} as const;

		gate.applyLiveEvent(liveFields); // the FIRST, genuine application
		expect(observed.at(-1)).toBe('stale');

		gate.notifyNavigated('/browse?x=1'); // a fetch starts AFTER the live application
		await vi.waitFor(() => expect(callCount).toBe(2));

		const lengthBeforeDuplicate = observed.length;
		gate.applyLiveEvent(liveFields); // a LATE DUPLICATE of the SAME event
		expect(observed.length).toBe(lengthBeforeDuplicate); // dropped — no new emission at all

		resolveSecond(statusResponse({ initialized: true, stale: false })); // the fetch's OWN response
		await vi.waitFor(() => expect(observed.at(-1)).toBe('ok'));
	});
});

describe('StatusGate.applyLiveEvent: classifies through the SAME classifyStatus fetchStatus uses', () => {
	it('the emitted verdict for a live event equals classifyStatus applied directly to the same fields', async () => {
		const { client } = statusClient(() => Promise.resolve(statusResponse()));
		const gate = createStatusGate(client, '/browse');
		await vi.waitFor(() => {});

		const fields = {
			initialized: false,
			stale: false,
			storeExists: true,
			indexingInProgress: true,
			commitSha: '',
			generation: 3n,
			epoch: 1
		} as const;

		const observed: string[] = [];
		gate.subscribe((s) => observed.push(s.verdict));
		gate.applyLiveEvent(fields);

		expect(observed.at(-1)).toBe(classifyStatus(fields).verdict);
	});
});

describe('createLiveStore: subscribe invoked synchronously once, with no event yet', () => {
	it('delivers null on the FIRST synchronous call, before any event has ever been admitted', () => {
		const fake = makeFakeClient();
		const store = createLiveStore(fake.start);
		let firstCallValue: LiveEvent | null | 'not-called' = 'not-called';
		store.subscribe((live) => {
			if (firstCallValue === 'not-called') firstCallValue = live;
		});
		expect(firstCallValue).toBeNull();
		store.stop();
	});
});

describe('createLiveStore: the epoch-scoped generation gate', () => {
	it('a replayed lower generation in the SAME epoch is ignored; a higher generation in the same epoch is applied; a LOWER generation in a NEW epoch is applied (the server-restart case)', () => {
		const fake = makeFakeClient();
		const store = createLiveStore(fake.start);
		const seen: Array<LiveEvent | null> = [];
		store.subscribe((live) => seen.push(live));
		expect(seen).toEqual([null]); // synchronous initial call, no event yet

		fake.deliver(liveEvent(5n, 1)); // first ever -> admitted regardless
		expect(seen).toHaveLength(2);
		expect(seen[1]!.event.generation).toBe(5n);

		fake.deliver(liveEvent(3n, 1)); // replayed lower, SAME epoch -> ignored
		expect(seen).toHaveLength(2);

		fake.deliver(liveEvent(5n, 1)); // EQUAL generation, same epoch -> also ignored (strict >)
		expect(seen).toHaveLength(2);

		fake.deliver(liveEvent(9n, 1)); // higher, same epoch -> applied
		expect(seen).toHaveLength(3);
		expect(seen[2]!.event.generation).toBe(9n);

		fake.deliver(liveEvent(1n, 2)); // LOWER generation, NEW epoch -> applied
		expect(seen).toHaveLength(4);
		expect(seen[3]!.event.generation).toBe(1n);
		expect(seen[3]!.epoch).toBe(2);

		store.stop();
		expect(fake.stopped).toBe(true);
	});
});

describe('createLiveStore: appliedAtMs stamping', () => {
	it('an ADMITTED event stamps a non-null appliedAtMs on its observation record; a DROPPED one leaves it null', () => {
		clearObservations();
		window.__codegraphLiveObservations = {
			events: [
				{ generation: 5, epoch: 1, seeded: true, receivedAtMs: 0, appliedAtMs: null },
				{ generation: 3, epoch: 1, seeded: false, receivedAtMs: 0, appliedAtMs: null }
			],
			connections: []
		};
		const fake = makeFakeClient();
		const store = createLiveStore(fake.start);

		fake.deliver(liveEvent(5n, 1)); // admitted (first ever)
		expect(window.__codegraphLiveObservations!.events[0].appliedAtMs).not.toBeNull();

		fake.deliver(liveEvent(3n, 1)); // dropped (replayed lower, same epoch)
		expect(window.__codegraphLiveObservations!.events[1].appliedAtMs).toBeNull();

		store.stop();
	});
});

describe('createLiveStore: the subscribe contract', () => {
	it('notifies every subscriber, in registration order, exactly once per admitted event', () => {
		const fake = makeFakeClient();
		const store = createLiveStore(fake.start);
		const order: string[] = [];
		const unsubA = store.subscribe(() => order.push('A'));
		const unsubB = store.subscribe(() => order.push('B'));
		order.length = 0; // clear the two synchronous initial calls

		fake.deliver(liveEvent(1n, 1));
		expect(order).toEqual(['A', 'B']);

		unsubA();
		unsubB();
		store.stop();
	});

	it('unsubscribe stops further delivery to that listener only', () => {
		const fake = makeFakeClient();
		const store = createLiveStore(fake.start);
		let countA = 0;
		let countB = 0;
		const unsubA = store.subscribe(() => (countA += 1));
		store.subscribe(() => (countB += 1));
		countA = 0;
		countB = 0;

		unsubA();
		fake.deliver(liveEvent(1n, 1));
		expect(countA).toBe(0);
		expect(countB).toBe(1);

		store.stop();
	});
});
