// live-client.ts is the browser's reconnecting consumer for WatchGraph
// (RPC-04/LIV-03): a for-await-shaped loop over the generated streaming
// method, exponential backoff with jitter on every disconnect — thrown
// or clean — and a connection EPOCH that lets a reconnecting tab keep
// applying events after the server's generation counter restarts at 1
// (T-06-41, guarded by Task 2's epoch-scoped gate downstream).
//
// This module never imports Svelte: its only inputs are the generated
// client method, a callback, and an injectable clock/random source, so
// it is unit-testable in plain jsdom with no component harness — the
// broadcast store in live-store.ts is the only Svelte-aware piece
// downstream of this file.
//
// Backoff shape (pinned — see this plan's own derivation, not left to
// runtime discretion): scheduled delay = baseTerm * (1 + u), baseTerm =
// LIVE_BACKOFF_BASE_MS * LIVE_BACKOFF_FACTOR^n (capped at
// LIVE_BACKOFF_CAP_MS), u drawn uniformly from
// [-LIVE_BACKOFF_JITTER_SPREAD, +LIVE_BACKOFF_JITTER_SPREAD].
// LIVE_BACKOFF_JITTER_SPREAD MUST stay strictly below 1/3 — that bound
// is what makes strict growth of the SCHEDULED delay (not merely its
// base term) a provable property rather than a hope: the largest value
// at level n is base*f^n*(1+s) and the smallest at level n+1 is
// base*f^(n+1)*(1-s), so growth is guaranteed exactly when
// 1+s < f*(1-s), i.e. s < 1/3 at f = 2. The same backoff applies to a
// clean stream end as to a thrown one — only an explicit stop() skips
// the next reconnect, never a graceful EOF.
import type { WatchGraphEvent } from '$lib/gen/ui_pb';

export const LIVE_BACKOFF_BASE_MS = 1000;
export const LIVE_BACKOFF_FACTOR = 2;
export const LIVE_BACKOFF_CAP_MS = 30_000;
export const LIVE_BACKOFF_JITTER_SPREAD = 0.25;
export const LIVE_OBSERVATION_CAP = 256;

/** One event delivered on one connection, paired with the connection's
 * own epoch — Task 2's store gate is scoped to this epoch, not to a
 * bare cross-connection generation comparison. */
export interface LiveEvent {
	event: WatchGraphEvent;
	epoch: number;
}

/** An injectable clock — real setTimeout/clearTimeout/performance.now by
 * default (transparently patchable by vi.useFakeTimers(), since these
 * are read live at call time, never cached at module load), swappable
 * in tests that want a synchronous, non-flaky clock instead. */
export interface LiveClientClock {
	now(): number;
	setTimeout(fn: () => void, ms: number): unknown;
	clearTimeout(handle: unknown): void;
}

export interface LiveClientDeps {
	/** The generated streaming method (uiClient.watchGraph), or a test
	 * double with the identical shape. */
	watchGraph(
		request: { sinceGeneration: bigint },
		options: { signal: AbortSignal }
	): AsyncIterable<WatchGraphEvent>;
	/** Invoked once per delivered event, in arrival order — never once
	 * per batch, and never after collecting the stream. */
	onEvent(live: LiveEvent): void;
	clock?: LiveClientClock;
	/** Returns a float in [0, 1); defaults to Math.random. Injectable so
	 * jitter is deterministic in tests and two independently-seeded
	 * clients can be proven to diverge. */
	random?: () => number;
}

export interface LiveClientHandle {
	/** Aborts any in-flight request, cancels a pending backoff timer, and
	 * schedules no further reconnect — a timer scheduled before stop()
	 * never fires a reconnect after it. */
	stop(): void;
}

function defaultClock(): LiveClientClock {
	return {
		now: () => performance.now(),
		setTimeout: (fn, ms) => setTimeout(fn, ms),
		clearTimeout: (handle) => clearTimeout(handle as ReturnType<typeof setTimeout>)
	};
}

function pushCapped<T>(arr: T[], item: T): void {
	arr.push(item);
	if (arr.length > LIVE_OBSERVATION_CAP) {
		arr.splice(0, arr.length - LIVE_OBSERVATION_CAP);
	}
}

function ensureObservations(): NonNullable<Window['__codegraphLiveObservations']> {
	if (!window.__codegraphLiveObservations) {
		window.__codegraphLiveObservations = { events: [], connections: [] };
	}
	return window.__codegraphLiveObservations;
}

interface ConnectionRecord {
	epoch: number;
	attempt: number;
	baseDelayMs: number | null;
	scheduledDelayMs: number | null;
	requestedSinceGeneration: number;
	openedAtMs: number | null;
}

/** sleep resolves after `ms`, or rejects immediately (and on any later
 * abort) if `signal` fires first — the mechanism that lets stop()
 * cancel a pending backoff wait with no further reconnect attempt. */
function sleep(clock: LiveClientClock, ms: number, signal: AbortSignal): Promise<void> {
	return new Promise<void>((resolve, reject) => {
		if (signal.aborted) {
			reject(new DOMException('aborted', 'AbortError'));
			return;
		}
		function onAbort(): void {
			clock.clearTimeout(handle);
			reject(new DOMException('aborted', 'AbortError'));
		}
		const handle = clock.setTimeout(() => {
			signal.removeEventListener('abort', onAbort);
			resolve();
		}, ms);
		signal.addEventListener('abort', onAbort);
	});
}

/** startLiveClient begins consuming the stream immediately (attempt 1
 * fires with no delay) and keeps reconnecting — with backoff, generation
 * resume, and epoch tracking — until stop() is called. */
export function startLiveClient(deps: LiveClientDeps): LiveClientHandle {
	const clock = deps.clock ?? defaultClock();
	const random = deps.random ?? Math.random;
	// Installed BEFORE the first connection attempt, synchronously, so
	// attempt 1 is always recorded — a seam created lazily on first event
	// would lose the initial connection record entirely.
	const observations = ensureObservations();
	const stopController = new AbortController();

	let epoch = 0;
	let lastGeneration = 0n;
	let everAttempted = false;
	// consecutiveEmptyAttempts resets to 0 on any attempt that delivers
	// at least one event, and increments otherwise (thrown failure OR a
	// clean end with zero events both count identically here — the same
	// backoff applies to both, per this module's own doc comment).
	let consecutiveEmptyAttempts = 0;

	function recordEvent(event: WatchGraphEvent, currentEpoch: number, seeded: boolean): void {
		pushCapped(observations.events, {
			generation: Number(event.generation),
			epoch: currentEpoch,
			seeded,
			receivedAtMs: clock.now(),
			appliedAtMs: null
		});
	}

	async function runLoop(): Promise<void> {
		while (!stopController.signal.aborted) {
			let baseDelayMs: number | null = null;
			let scheduledDelayMs: number | null = null;

			if (everAttempted) {
				const rawBase =
					LIVE_BACKOFF_BASE_MS * Math.pow(LIVE_BACKOFF_FACTOR, consecutiveEmptyAttempts);
				baseDelayMs = Math.min(rawBase, LIVE_BACKOFF_CAP_MS);
				const u = (random() * 2 - 1) * LIVE_BACKOFF_JITTER_SPREAD;
				scheduledDelayMs = baseDelayMs * (1 + u);
				try {
					await sleep(clock, scheduledDelayMs, stopController.signal);
				} catch {
					return; // stopped during the backoff wait — no further reconnect
				}
			}
			everAttempted = true;
			if (stopController.signal.aborted) return;

			const record: ConnectionRecord = {
				epoch: epoch + 1, // tentative — becomes exact the instant this attempt establishes
				attempt: consecutiveEmptyAttempts + 1,
				baseDelayMs,
				scheduledDelayMs,
				requestedSinceGeneration: Number(lastGeneration),
				openedAtMs: null
			};
			pushCapped(observations.connections, record);

			let established = false;
			let receivedAny = false;
			let firstEventOfConnection = true;
			try {
				const iterable = deps.watchGraph(
					{ sinceGeneration: lastGeneration },
					{ signal: stopController.signal }
				);
				const iterator = iterable[Symbol.asyncIterator]();
				for (;;) {
					const result = await iterator.next();
					if (stopController.signal.aborted) return;
					if (!established) {
						// Epoch increments once per successful establishment —
						// the first `.next()` step that does not throw — never
						// per delivered event, so three events on one
						// connection share exactly one epoch value.
						established = true;
						epoch += 1;
						record.openedAtMs = clock.now();
						record.epoch = epoch;
					}
					if (result.done) break;
					receivedAny = true;
					const event = result.value;
					lastGeneration = event.generation;
					recordEvent(event, epoch, firstEventOfConnection);
					firstEventOfConnection = false;
					deps.onEvent({ event, epoch });
				}
			} catch {
				if (stopController.signal.aborted) return;
				// Fall through to the same failure accounting a clean,
				// zero-event end receives — both back off identically.
			}

			consecutiveEmptyAttempts = receivedAny ? 0 : consecutiveEmptyAttempts + 1;
		}
	}

	void runLoop();


	return {
		stop(): void {
			stopController.abort();
		}
	};
}
