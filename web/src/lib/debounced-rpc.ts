// debounced-rpc.ts is the ONE debounce + per-dispatch AbortController +
// monotonic request-identity mechanism (D-15), extracted from
// search.ts's live path (04-06 Task 1, cycle-1 review finding: an
// earlier draft of this plan simultaneously forbade a second
// implementation of this mechanism AND directed file-search.ts to write
// exactly that, sharing only two exported constants — sharing two
// numbers is not sharing a mechanism). search.ts's live path and
// file-search.ts are this module's two configurations.
//
// This module owns exactly four things and nothing else: the debounce
// timer, ONE AbortController created per dispatch (never before — a
// coalesced keystroke that never reaches dispatch never creates one), a
// monotonically increasing request id whose stale responses are
// discarded in BOTH the resolve and the reject paths, and a dispose()
// that clears the timer and aborts the in-flight request. It holds no
// store and knows no RPC shape: state shaping stays with each caller,
// via the onResult/onFailure/onBelowMinimum callbacks below.
//
// onBelowMinimum (cycle-2 review finding): search.ts's original
// below-minimum branch did THREE things — abort the in-flight request,
// bump the request identity so an ALREADY-DISPATCHED response is
// discarded when it later settles, and clear the caller's own visible
// state. The first two are mechanism and belong here; the third is
// state shaping and belongs to the caller, which is why this callback
// exists. Re-deriving the minimum-length rule caller-side (each caller
// comparing term.length itself) would put minChars back in two places —
// the same sharing-numbers-instead-of-mechanism smell this extraction
// exists to remove — so the comparison lives here, once, and the
// caller only reacts to the outcome.
export interface DebouncedRpcOptions<TResult> {
	debounceMs: number;
	minChars: number;
	dispatch(term: string, signal: AbortSignal): Promise<TResult>;
	onResult(result: TResult): void;
	onFailure(err: unknown): void;
	onBelowMinimum(): void;
}

export interface DebouncedRpc {
	// setQuery mirrors search.ts's own setQuery shape: a below-minimum
	// value issues no dispatch, cancels whatever was in flight, and
	// invalidates any response still on the way; a value at or above the
	// minimum (re)starts the debounce.
	setQuery(value: string): void;
	// dispose (IN-13) clears a pending debounce timer and aborts an
	// in-flight request — the caller's own lifecycle exit for this
	// mechanism's async surface.
	dispose(): void;
}

export function createDebouncedRpc<TResult>(options: DebouncedRpcOptions<TResult>): DebouncedRpc {
	const { debounceMs, minChars, dispatch, onResult, onFailure, onBelowMinimum } = options;

	let debounceTimer: ReturnType<typeof setTimeout> | undefined;
	let abort: AbortController | undefined;
	let requestId = 0;

	function dispatchNow(term: string): void {
		if (abort) abort.abort();
		const controller = new AbortController();
		abort = controller;
		const id = ++requestId;

		dispatch(term, controller.signal)
			.then((result) => {
				if (id !== requestId) return; // superseded — discard
				onResult(result);
			})
			.catch((err: unknown) => {
				if (id !== requestId) return; // superseded — discard, not a real failure
				onFailure(err);
			});
	}

	function setQuery(value: string): void {
		if (debounceTimer !== undefined) {
			clearTimeout(debounceTimer);
			debounceTimer = undefined;
		}

		if (value.length < minChars) {
			// A below-minimum value issues no dispatch — and cancels
			// whatever was still in flight for a longer prefix, so a
			// backspace-to-short never lets a stale result land later.
			if (abort) {
				abort.abort();
				abort = undefined;
			}
			requestId += 1; // invalidate any in-flight response too
			onBelowMinimum();
			return;
		}

		debounceTimer = setTimeout(() => {
			debounceTimer = undefined;
			dispatchNow(value);
		}, debounceMs);
	}

	function dispose(): void {
		if (debounceTimer !== undefined) {
			clearTimeout(debounceTimer);
			debounceTimer = undefined;
		}
		if (abort) {
			abort.abort();
			abort = undefined;
		}
		// WR-08 (04-REVIEW.md): mirrors the below-minimum path's own
		// invalidation above — without this, a caller that disposes while
		// a request is in flight (e.g. FilePicker.svelte unmounting on tab
		// switch mid-search) gets that request's abort rejection delivered
		// to onFailure anyway, because nothing moved requestId and the
		// `id !== requestId` guard in dispatchNow's .catch still passes.
		// That writes into state belonging to a component already being
		// torn down — this module's own header comment claims stale
		// responses are discarded "in BOTH the resolve and the reject
		// paths", which was not true across dispose() until this line.
		requestId += 1; // invalidate any in-flight response, mirroring the below-minimum path
	}

	return { setQuery, dispose };
}
