// live-client.test.ts — 06-03 Task 1: the browser-side reconnecting
// stream consumer. Written and run RED before
// web/src/lib/live/live-client.ts existed (recorded in the SUMMARY),
// then made GREEN.
//
// Every test uses `random: () => 0.5` (which zeroes jitter: u = (0.5*2
// -1)*SPREAD = 0) wherever an EXACT delay value matters, so timing
// assertions are deterministic rather than relying on real elapsed time.
// vi.useFakeTimers() plus vi.advanceTimersByTimeAsync (the SAME
// convention web/tests/status.test.ts already established) drives the
// backoff waits — live-client.ts's default clock reads the global
// setTimeout/clearTimeout live at call time, so fake timers patch it
// transparently with no injected clock needed.
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

import {
	startLiveClient,
	LIVE_BACKOFF_BASE_MS,
	LIVE_BACKOFF_FACTOR,
	LIVE_BACKOFF_JITTER_SPREAD,
	LIVE_OBSERVATION_CAP,
	type LiveEvent
} from '$lib/live/live-client';
import type { WatchGraphEvent } from '$lib/gen/ui_pb';

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

function deferred<T = void>(): { promise: Promise<T>; resolve: (v: T) => void } {
	let resolve!: (v: T) => void;
	const promise = new Promise<T>((res) => {
		resolve = res;
	});
	return { promise, resolve };
}

type ScriptStep =
	| { kind: 'event'; event: WatchGraphEvent }
	| { kind: 'end' }
	| { kind: 'error'; error: unknown };

function scriptStream(steps: ScriptStep[]): AsyncIterable<WatchGraphEvent> {
	let i = 0;
	return {
		[Symbol.asyncIterator]() {
			return {
				async next(): Promise<IteratorResult<WatchGraphEvent>> {
					if (i >= steps.length) return { done: true, value: undefined };
					const step = steps[i++];
					if (step.kind === 'event') return { done: false, value: step.event };
					if (step.kind === 'end') return { done: true, value: undefined };
					throw step.error;
				}
			};
		}
	};
}

/** A stream whose second `next()` call blocks on an externally-resolved
 * promise before yielding — used to prove incremental (never buffered)
 * consumption. */
function twoEventStreamWithGate(
	first: WatchGraphEvent,
	second: WatchGraphEvent,
	gate: Promise<void>
): AsyncIterable<WatchGraphEvent> {
	let i = 0;
	return {
		[Symbol.asyncIterator]() {
			return {
				async next(): Promise<IteratorResult<WatchGraphEvent>> {
					if (i === 0) {
						i++;
						return { done: false, value: first };
					}
					if (i === 1) {
						i++;
						await gate;
						return { done: false, value: second };
					}
					return { done: true, value: undefined };
				}
			};
		}
	};
}

function clearObservations(): void {
	delete (window as unknown as { __codegraphLiveObservations?: unknown })
		.__codegraphLiveObservations;
}

beforeEach(() => {
	clearObservations();
	vi.useFakeTimers();
});

afterEach(() => {
	vi.useRealTimers();
});

describe('LIVE_BACKOFF_JITTER_SPREAD: the growth theorem holds only below 1/3', () => {
	it('is strictly less than 1/3', () => {
		expect(LIVE_BACKOFF_JITTER_SPREAD).toBeLessThan(1 / 3);
	});
});

describe('startLiveClient: incremental consumption — never buffers the stream', () => {
	it('the second event resolves only after the first callback already fired', async () => {
		const secondReady = deferred<void>();
		const received: LiveEvent[] = [];
		let firstCallbackFired = false;

		const stream = twoEventStreamWithGate(
			watchGraphEvent({ generation: 1n }),
			watchGraphEvent({ generation: 2n }),
			secondReady.promise
		);

		const handle = startLiveClient({
			watchGraph: () => stream,
			onEvent: (live) => {
				received.push(live);
				if (received.length === 1) firstCallbackFired = true;
			}
		});

		await vi.waitFor(() => expect(firstCallbackFired).toBe(true));
		// The second event has NOT been produced yet — a buffering
		// implementation (collect-then-iterate) would never have reached
		// this line with firstCallbackFired true, since collection cannot
		// finish while secondReady is still pending.
		expect(received).toHaveLength(1);

		secondReady.resolve();
		await vi.waitFor(() => expect(received).toHaveLength(2));
		expect(received[1].event.generation).toBe(2n);

		handle.stop();
	});
});

describe('startLiveClient: generation resume', () => {
	it('the first ever connection sends sinceGeneration 0; a reconnect resumes from the last-seen generation', async () => {
		const calls: bigint[] = [];
		let call = 0;
		const watchGraph = vi.fn((req: { sinceGeneration: bigint }) => {
			calls.push(req.sinceGeneration);
			call += 1;
			if (call === 1) {
				return scriptStream([
					{ kind: 'event', event: watchGraphEvent({ generation: 5n }) },
					{ kind: 'error', error: new Error('dropped') }
				]);
			}
			return scriptStream([]); // hang forever (never resolves 'end' either) — irrelevant past this point
		});

		const handle = startLiveClient({ watchGraph, onEvent: () => {}, random: () => 0.5 });
		await vi.waitFor(() => expect(calls.length).toBeGreaterThanOrEqual(1));
		expect(calls[0]).toBe(0n);

		// Advance past the backoff so the reconnect fires.
		await vi.advanceTimersByTimeAsync(LIVE_BACKOFF_BASE_MS * LIVE_BACKOFF_FACTOR * 2);
		await vi.waitFor(() => expect(calls.length).toBeGreaterThanOrEqual(2));
		expect(calls[1]).toBe(5n);

		handle.stop();
	});
});

describe('startLiveClient: connection epoch', () => {
	it('epoch increments once per successful establishment, not per delivered event; a reconnect increments it again', async () => {
		const epochs: number[] = [];
		let call = 0;
		const watchGraph = vi.fn(() => {
			call += 1;
			if (call === 1) {
				return scriptStream([
					{ kind: 'event', event: watchGraphEvent({ generation: 1n }) },
					{ kind: 'event', event: watchGraphEvent({ generation: 2n }) },
					{ kind: 'event', event: watchGraphEvent({ generation: 3n }) },
					{ kind: 'end' }
				]);
			}
			return scriptStream([{ kind: 'event', event: watchGraphEvent({ generation: 4n }) }]);
		});

		const handle = startLiveClient({
			watchGraph,
			onEvent: (live) => epochs.push(live.epoch),
			random: () => 0.5
		});

		await vi.waitFor(() => expect(epochs).toHaveLength(3));
		expect(epochs).toEqual([1, 1, 1]);

		// The clean end after 3 delivered events still counts as a success
		// (receivedAny), so the reconnect's own backoff uses the RESET
		// exponent (0) — exactly LIVE_BACKOFF_BASE_MS with zero jitter.
		// Advance no further than that single step, so the second
		// connection's lone event is delivered without a THIRD reconnect
		// (which would also fire quickly, since it too resets on success)
		// sneaking into the same window.
		await vi.advanceTimersByTimeAsync(LIVE_BACKOFF_BASE_MS);
		expect(epochs).toHaveLength(4);
		expect(epochs[3]).toBe(2);

		handle.stop();
	});
});

describe('startLiveClient: backoff growth and jitter bound', () => {
	it('the SCHEDULED delay strictly increases across at least three consecutive failures', async () => {
		const watchGraph = vi.fn(() => {
			throw new Error('synthetic connection failure');
		});
		const handle = startLiveClient({ watchGraph, onEvent: () => {}, random: () => 0.5 });

		// Advance in small steps, letting each backoff fire in turn.
		for (let i = 0; i < 5; i++) {
			await vi.advanceTimersByTimeAsync(LIVE_BACKOFF_BASE_MS * Math.pow(LIVE_BACKOFF_FACTOR, i + 2));
		}
		handle.stop();

		const conns = window.__codegraphLiveObservations!.connections;
		expect(conns.length).toBeGreaterThanOrEqual(4);
		expect(conns[0].scheduledDelayMs).toBeNull();
		const scheduled = conns.slice(1, 4).map((c) => c.scheduledDelayMs!);
		expect(scheduled[1]).toBeGreaterThan(scheduled[0]);
		expect(scheduled[2]).toBeGreaterThan(scheduled[1]);
	});

	it('two independently-seeded clients produce DIFFERENT delays for the same failure index', async () => {
		const watchGraphA = vi.fn(() => {
			throw new Error('fail A');
		});
		const handleA = startLiveClient({ watchGraph: watchGraphA, onEvent: () => {}, random: () => 0.1 });
		await vi.advanceTimersByTimeAsync(LIVE_BACKOFF_BASE_MS * LIVE_BACKOFF_FACTOR * 2);
		handleA.stop();
		expect(window.__codegraphLiveObservations!.connections.length).toBeGreaterThanOrEqual(2);
		const delayA = window.__codegraphLiveObservations!.connections[1].scheduledDelayMs;
		clearObservations();

		const watchGraphB = vi.fn(() => {
			throw new Error('fail B');
		});
		const handleB = startLiveClient({ watchGraph: watchGraphB, onEvent: () => {}, random: () => 0.9 });
		await vi.advanceTimersByTimeAsync(LIVE_BACKOFF_BASE_MS * LIVE_BACKOFF_FACTOR * 2);
		handleB.stop();
		expect(window.__codegraphLiveObservations!.connections.length).toBeGreaterThanOrEqual(2);
		const delayB = window.__codegraphLiveObservations!.connections[1].scheduledDelayMs;

		expect(delayA).not.toBeNull();
		expect(delayB).not.toBeNull();
		expect(delayA).not.toBe(delayB);
	});

	it('every recorded scheduledDelayMs lies within LIVE_BACKOFF_JITTER_SPREAD of its own baseDelayMs, and at least one differs from it', async () => {
		let call = 0;
		const seq = [0.1, 0.9, 0.3, 0.7];
		const watchGraph = vi.fn(() => {
			call += 1;
			throw new Error('fail');
		});
		const handle = startLiveClient({
			watchGraph,
			onEvent: () => {},
			random: () => seq[(call - 1) % seq.length] ?? 0.5
		});

		for (let i = 0; i < 4; i++) {
			await vi.advanceTimersByTimeAsync(LIVE_BACKOFF_BASE_MS * Math.pow(LIVE_BACKOFF_FACTOR, i + 2));
		}
		handle.stop();

		const conns = window.__codegraphLiveObservations!.connections.filter(
			(c) => c.baseDelayMs !== null
		);
		expect(conns.length).toBeGreaterThanOrEqual(3);
		let sawDifference = false;
		for (const c of conns) {
			const ratio = c.scheduledDelayMs! / c.baseDelayMs!;
			expect(Math.abs(ratio - 1)).toBeLessThanOrEqual(LIVE_BACKOFF_JITTER_SPREAD + 1e-9);
			if (c.scheduledDelayMs !== c.baseDelayMs) sawDifference = true;
		}
		expect(sawDifference).toBe(true);
	});

	it('the delay resets to its base after any successfully received event — not the pre-success value', async () => {
		let call = 0;
		const watchGraph = vi.fn(() => {
			call += 1;
			if (call === 1) throw new Error('fail 1'); // consecutiveEmptyAttempts: 0 -> 1
			if (call === 2) return scriptStream([{ kind: 'event', event: watchGraphEvent({ generation: 1n }) }]); // success -> resets to 0
			throw new Error('fail after success'); // uses the RESET base
		});

		const handle = startLiveClient({ watchGraph, onEvent: () => {}, random: () => 0.5 });

		await vi.advanceTimersByTimeAsync(LIVE_BACKOFF_BASE_MS * LIVE_BACKOFF_FACTOR * 4); // let attempt 2 (success) fire
		await vi.waitFor(() => expect(call).toBeGreaterThanOrEqual(2));
		await vi.advanceTimersByTimeAsync(LIVE_BACKOFF_BASE_MS * LIVE_BACKOFF_FACTOR * 4); // let attempt 3 (post-success failure) fire
		await vi.waitFor(() => expect(call).toBeGreaterThanOrEqual(3));
		handle.stop();

		const conns = window.__codegraphLiveObservations!.connections;
		// conns[2] is attempt 3 — its baseDelayMs was computed right after
		// attempt 2's success reset consecutiveEmptyAttempts to 0, so it
		// must equal the pure base term (exponent 0), not a continuation
		// of attempt 1's already-elevated streak.
		expect(conns[2].baseDelayMs).toBe(LIVE_BACKOFF_BASE_MS);
	});

	it('a stream that ends cleanly 5 times in a row backs off exactly like a thrown failure, growing and bounding the attempt count', async () => {
		const watchGraph = vi.fn(() => scriptStream([{ kind: 'end' }]));
		const handle = startLiveClient({ watchGraph, onEvent: () => {}, random: () => 0.5 });

		// Sum of the first 5 backoff delays (exponents 1..4, since attempt 1
		// is exempt): B*2 + B*4 + B*8 + B*16 = 30*B.
		await vi.advanceTimersByTimeAsync(LIVE_BACKOFF_BASE_MS * 30);
		handle.stop();

		const conns = window.__codegraphLiveObservations!.connections;
		// Floor: the run of clean ends actually produced multiple attempts.
		expect(conns.length).toBeGreaterThanOrEqual(5);
		// Ceiling: within this fixed virtual-time window, the exponential
		// schedule permits only a bounded number of attempts — a tight
		// reconnect loop would have produced far more than this.
		expect(conns.length).toBeLessThanOrEqual(6);

		const bases = conns.slice(1, 5).map((c) => c.baseDelayMs!);
		for (let i = 1; i < bases.length; i++) {
			expect(bases[i]).toBeGreaterThan(bases[i - 1]);
		}
	});
});

describe('startLiveClient: stop()', () => {
	it('aborts scheduling — zero further calls after stop, with at least one call before it', async () => {
		const watchGraph = vi.fn(() => scriptStream([{ kind: 'end' }]));
		const handle = startLiveClient({ watchGraph, onEvent: () => {}, random: () => 0.5 });

		await vi.waitFor(() => expect(watchGraph).toHaveBeenCalledTimes(1));
		handle.stop();

		// Advance well past what would have been the next backoff — no
		// reconnect should ever fire, including the one a pending timer
		// scheduled before stop() would otherwise have fired.
		await vi.advanceTimersByTimeAsync(LIVE_BACKOFF_BASE_MS * 1000);
		expect(watchGraph).toHaveBeenCalledTimes(1);
	});
});

describe('startLiveClient: the observation seam is installed before the first attempt', () => {
	it('window.__codegraphLiveObservations exists with attempt 1 recorded synchronously, before any microtask runs', () => {
		const watchGraph = vi.fn(() => scriptStream([]));
		const handle = startLiveClient({ watchGraph, onEvent: () => {}, random: () => 0.5 });

		// No await at all — this assertion runs in the SAME synchronous
		// tick as startLiveClient's own call, proving the seam and its
		// first connection record exist before control ever yields.
		expect(window.__codegraphLiveObservations).toBeDefined();
		expect(window.__codegraphLiveObservations!.connections).toHaveLength(1);
		expect(window.__codegraphLiveObservations!.connections[0].attempt).toBe(1);

		handle.stop();
	});
});

describe('startLiveClient: the observation seam', () => {
	it('records N event entries and 2 connection entries after N events and one reconnect — first connection has null scheduledDelayMs, the reconnect a positive one', async () => {
		let call = 0;
		const watchGraph = vi.fn(() => {
			call += 1;
			if (call === 1) {
				return scriptStream([
					{ kind: 'event', event: watchGraphEvent({ generation: 1n }) },
					{ kind: 'event', event: watchGraphEvent({ generation: 2n }) },
					{ kind: 'error', error: new Error('dropped') }
				]);
			}
			return scriptStream([{ kind: 'event', event: watchGraphEvent({ generation: 3n }) }]);
		});

		const handle = startLiveClient({ watchGraph, onEvent: () => {}, random: () => 0.5 });
		await vi.waitFor(() => expect(call).toBeGreaterThanOrEqual(1));
		// call1 delivered 2 events before failing — receivedAny resets the
		// backoff exponent to 0, so the reconnect uses exactly
		// LIVE_BACKOFF_BASE_MS with zero jitter. Advance no further, so a
		// THIRD connection (which would also reset-and-fire quickly, since
		// call2's single event also counts as a success) cannot sneak in.
		await vi.advanceTimersByTimeAsync(LIVE_BACKOFF_BASE_MS);
		expect(window.__codegraphLiveObservations!.events.length).toBe(3);
		handle.stop();

		const observations = window.__codegraphLiveObservations!;
		expect(observations.events).toHaveLength(3);
		expect(observations.connections).toHaveLength(2);
		expect(observations.connections[0].scheduledDelayMs).toBeNull();
		expect(observations.connections[1].scheduledDelayMs).not.toBeNull();
		expect(observations.connections[1].scheduledDelayMs!).toBeGreaterThan(0);
	});

	it('caps the events array at LIVE_OBSERVATION_CAP, discarding the oldest first', async () => {
		const total = LIVE_OBSERVATION_CAP + 10;
		const steps: ScriptStep[] = [];
		for (let i = 1; i <= total; i++) {
			steps.push({ kind: 'event', event: watchGraphEvent({ generation: BigInt(i) }) });
		}
		const watchGraph = vi.fn(() => scriptStream(steps));
		const handle = startLiveClient({ watchGraph, onEvent: () => {}, random: () => 0.5 });

		await vi.waitFor(() =>
			expect(window.__codegraphLiveObservations!.events.length).toBe(LIVE_OBSERVATION_CAP)
		);
		handle.stop();

		const events = window.__codegraphLiveObservations!.events;
		expect(events.length).toBeLessThanOrEqual(LIVE_OBSERVATION_CAP);
		expect(events.length).toBeGreaterThan(0);
		expect(events[events.length - 1].generation).toBe(total);
	});
});
