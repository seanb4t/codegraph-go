// debounced-rpc.test.ts covers the extracted mechanism (04-06 Task 1,
// cycle-1 review finding) directly: min-length gate, one dispatch per
// debounce window, the un-aborted single signal, the aborted
// overlapping dispatch, out-of-order discard, rejection handling,
// dispose clearing a pending timer and aborting an in-flight request —
// plus the below-minimum-while-in-flight case (cycle-2 review finding)
// at the mechanism's own level, which neither this extraction's
// predecessor nor search.test.ts's existing suite covers (search.test.ts's
// min-length suite tests a one-character query that issues NO request;
// its cancellation suite tests long-to-longer overlap, never a
// below-minimum term arriving while a request is already in flight).
//
// This file is written and run RED (debounced-rpc.ts does not exist yet)
// before any implementation — recorded in 04-06-SUMMARY.md.
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';

import { createDebouncedRpc } from '$lib/debounced-rpc';

const DEBOUNCE_MS = 150;
const MIN_CHARS = 2;

function deferred<T>() {
	let resolve!: (value: T) => void;
	let reject!: (reason?: unknown) => void;
	const promise = new Promise<T>((res, rej) => {
		resolve = res;
		reject = rej;
	});
	return { promise, resolve, reject };
}

beforeEach(() => {
	vi.useFakeTimers();
});

afterEach(() => {
	vi.useRealTimers();
});

describe('createDebouncedRpc: minimum-length and debounce gating', () => {
	it('a term shorter than minChars issues no dispatch, and fires onBelowMinimum instead', async () => {
		const dispatch = vi.fn(() => Promise.resolve('x'));
		const onBelowMinimum = vi.fn();
		const rpc = createDebouncedRpc({
			debounceMs: DEBOUNCE_MS,
			minChars: MIN_CHARS,
			dispatch,
			onResult: vi.fn(),
			onFailure: vi.fn(),
			onBelowMinimum
		});

		rpc.setQuery('a');
		await vi.advanceTimersByTimeAsync(DEBOUNCE_MS + 50);

		expect(dispatch).not.toHaveBeenCalled();
		expect(onBelowMinimum).toHaveBeenCalledTimes(1);
	});

	it('a term at least minChars long dispatches exactly once after the debounce interval', async () => {
		const dispatch = vi.fn((_term: string, _signal: AbortSignal) => Promise.resolve('x'));
		const rpc = createDebouncedRpc({
			debounceMs: DEBOUNCE_MS,
			minChars: MIN_CHARS,
			dispatch,
			onResult: vi.fn(),
			onFailure: vi.fn(),
			onBelowMinimum: vi.fn()
		});

		rpc.setQuery('ab');
		expect(dispatch).not.toHaveBeenCalled(); // nothing before the debounce fires

		await vi.advanceTimersByTimeAsync(DEBOUNCE_MS);

		expect(dispatch).toHaveBeenCalledTimes(1);
		expect(dispatch.mock.calls[0]![0]).toBe('ab');
	});
});

describe('createDebouncedRpc: coalescing vs. genuine overlap', () => {
	it('three keystrokes within the debounce window issue exactly one dispatch, for the final value, with ONE un-aborted signal', async () => {
		const signals: AbortSignal[] = [];
		const dispatch = vi.fn((_term: string, signal: AbortSignal) => {
			signals.push(signal);
			return Promise.resolve('x');
		});
		const rpc = createDebouncedRpc({
			debounceMs: DEBOUNCE_MS,
			minChars: MIN_CHARS,
			dispatch,
			onResult: vi.fn(),
			onFailure: vi.fn(),
			onBelowMinimum: vi.fn()
		});

		rpc.setQuery('ab');
		await vi.advanceTimersByTimeAsync(30);
		rpc.setQuery('abc');
		await vi.advanceTimersByTimeAsync(30);
		rpc.setQuery('abcd');
		await vi.advanceTimersByTimeAsync(DEBOUNCE_MS + 10);

		expect(dispatch).toHaveBeenCalledTimes(1);
		expect(dispatch.mock.calls[0]![0]).toBe('abcd');
		expect(signals).toHaveLength(1);
		expect(signals[0]!.aborted).toBe(false);
	});

	it('a genuinely overlapping dispatch aborts the FIRST signal and leaves the second un-aborted', async () => {
		const first = deferred<string>();
		const second = deferred<string>();
		let call = 0;
		const signals: AbortSignal[] = [];
		const dispatch = vi.fn((_term: string, signal: AbortSignal) => {
			signals.push(signal);
			call += 1;
			return call === 1 ? first.promise : second.promise;
		});
		const rpc = createDebouncedRpc({
			debounceMs: DEBOUNCE_MS,
			minChars: MIN_CHARS,
			dispatch,
			onResult: vi.fn(),
			onFailure: vi.fn(),
			onBelowMinimum: vi.fn()
		});

		rpc.setQuery('ab');
		await vi.advanceTimersByTimeAsync(DEBOUNCE_MS);
		expect(dispatch).toHaveBeenCalledTimes(1);

		rpc.setQuery('abcd');
		await vi.advanceTimersByTimeAsync(DEBOUNCE_MS);
		expect(dispatch).toHaveBeenCalledTimes(2);

		expect(signals[0]!.aborted).toBe(true);
		expect(signals[1]!.aborted).toBe(false);
	});
});

describe('createDebouncedRpc: out-of-order safety', () => {
	it('a later dispatch settling AFTER an earlier one still wins — the earlier response is discarded', async () => {
		const first = deferred<string>();
		const second = deferred<string>();
		let call = 0;
		const dispatch = vi.fn(() => {
			call += 1;
			return call === 1 ? first.promise : second.promise;
		});
		const onResult = vi.fn();
		const rpc = createDebouncedRpc({
			debounceMs: DEBOUNCE_MS,
			minChars: MIN_CHARS,
			dispatch,
			onResult,
			onFailure: vi.fn(),
			onBelowMinimum: vi.fn()
		});

		rpc.setQuery('ab');
		await vi.advanceTimersByTimeAsync(DEBOUNCE_MS);
		rpc.setQuery('abcd');
		await vi.advanceTimersByTimeAsync(DEBOUNCE_MS);

		second.resolve('second-result');
		await Promise.resolve();
		await Promise.resolve();
		first.resolve('first-result');
		await Promise.resolve();
		await Promise.resolve();

		expect(onResult).toHaveBeenCalledTimes(1);
		expect(onResult).toHaveBeenCalledWith('second-result');
	});
});

describe('createDebouncedRpc: rejection handling', () => {
	it('a rejection calls onFailure with the raw error, and never throws', async () => {
		const err = new Error('boom');
		const dispatch = vi.fn(() => Promise.reject(err));
		const onFailure = vi.fn();
		const rpc = createDebouncedRpc({
			debounceMs: DEBOUNCE_MS,
			minChars: MIN_CHARS,
			dispatch,
			onResult: vi.fn(),
			onFailure,
			onBelowMinimum: vi.fn()
		});

		rpc.setQuery('ab');
		await vi.advanceTimersByTimeAsync(DEBOUNCE_MS);
		await Promise.resolve();
		await Promise.resolve();

		expect(onFailure).toHaveBeenCalledWith(err);
	});
});

describe('createDebouncedRpc: dispose', () => {
	it('dispose clears a pending debounce timer — it never fires afterward', async () => {
		const dispatch = vi.fn(() => Promise.resolve('x'));
		const rpc = createDebouncedRpc({
			debounceMs: DEBOUNCE_MS,
			minChars: MIN_CHARS,
			dispatch,
			onResult: vi.fn(),
			onFailure: vi.fn(),
			onBelowMinimum: vi.fn()
		});

		rpc.setQuery('ab');
		rpc.dispose();
		await vi.advanceTimersByTimeAsync(DEBOUNCE_MS * 2);

		expect(dispatch).not.toHaveBeenCalled();
	});

	it('dispose aborts an in-flight request', async () => {
		const inflight = deferred<string>();
		let signal: AbortSignal | undefined;
		const dispatch = vi.fn((_term: string, s: AbortSignal) => {
			signal = s;
			return inflight.promise;
		});
		const rpc = createDebouncedRpc({
			debounceMs: DEBOUNCE_MS,
			minChars: MIN_CHARS,
			dispatch,
			onResult: vi.fn(),
			onFailure: vi.fn(),
			onBelowMinimum: vi.fn()
		});

		rpc.setQuery('ab');
		await vi.advanceTimersByTimeAsync(DEBOUNCE_MS);
		expect(signal?.aborted).toBe(false);

		rpc.dispose();
		expect(signal?.aborted).toBe(true);
	});
});

describe('createDebouncedRpc: below the minimum while a request is in flight (cycle-2 finding)', () => {
	it('aborts the in-flight signal, fires onBelowMinimum exactly once, dispatches nothing further, and never fires onResult when the stale promise later settles', async () => {
		const inflight = deferred<string>();
		const dispatch = vi.fn((_term: string, _signal: AbortSignal) => inflight.promise);
		const onResult = vi.fn();
		const onBelowMinimum = vi.fn();
		const rpc = createDebouncedRpc({
			debounceMs: DEBOUNCE_MS,
			minChars: MIN_CHARS,
			dispatch,
			onResult,
			onFailure: vi.fn(),
			onBelowMinimum
		});

		rpc.setQuery('ab');
		await vi.advanceTimersByTimeAsync(DEBOUNCE_MS);
		expect(dispatch).toHaveBeenCalledTimes(1);
		const signal = dispatch.mock.calls[0]![1] as AbortSignal;

		rpc.setQuery('a'); // below minChars, WHILE the above request is still unresolved

		expect(signal.aborted).toBe(true);
		expect(onBelowMinimum).toHaveBeenCalledTimes(1);
		expect(dispatch).toHaveBeenCalledTimes(1); // no further dispatch

		// The load-bearing half: an abort alone does not stop an
		// already-resolved promise's .then from running — settling the
		// STALE promise must still never reach onResult, which is why the
		// request identity is bumped too, not merely the abort issued.
		inflight.resolve('stale-result');
		await Promise.resolve();
		await Promise.resolve();

		expect(onResult).not.toHaveBeenCalled();
	});

	it('a second below-minimum term fires onBelowMinimum again but dispatches nothing and aborts nothing further (nothing left in flight)', async () => {
		const dispatch = vi.fn(() => Promise.resolve('x'));
		const onBelowMinimum = vi.fn();
		const rpc = createDebouncedRpc({
			debounceMs: DEBOUNCE_MS,
			minChars: MIN_CHARS,
			dispatch,
			onResult: vi.fn(),
			onFailure: vi.fn(),
			onBelowMinimum
		});

		rpc.setQuery('a');
		expect(onBelowMinimum).toHaveBeenCalledTimes(1);

		rpc.setQuery('');
		expect(onBelowMinimum).toHaveBeenCalledTimes(2);
		expect(dispatch).not.toHaveBeenCalled();
	});
});
