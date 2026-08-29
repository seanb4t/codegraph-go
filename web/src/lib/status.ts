// status.ts is the shared status gate (D-04/D-05): the ONE place
// GetStatus's field combination becomes a named health verdict plus an
// orthogonal commit-knowledge field, and the ONE fetch trigger this app
// has for "on load, on navigation, nothing else" — Phase 6 replaces
// this mechanism with live push, so it is deliberately a small seam,
// never a polling subsystem.
//
// GetStatus is the only rpc that ANSWERS when the index is degraded
// (internal/uiserver/degrade.go's degradedStatus); every other handler
// returns a Connect error instead (see rpc-errors.ts, the error half of
// the same problem). classifyStatus turns that answer into a named
// five-member health verdict, reading the FIELD COMBINATION rather than
// `initialized` alone — that flag is false in BOTH degrade cases, which
// is exactly why `store_exists` (ui.proto field 8) exists as a separate
// field distinguishing "no .codegraph/ directory anywhere" from "the
// directory exists but the store stayed locked past the retry budget".
//
// Commit knowledge is modeled as an ORTHOGONAL field, never a sixth
// verdict: an empty commit_sha means "unknown", independent of index
// health (ui.proto:151) — folding it into the verdict would render a
// healthy, current index as degraded purely because its commit was
// never recorded, which is this repository's own long-lived index
// today.
import type { MessageInitShape } from '@bufbuild/protobuf';
import type { GetStatusRequestSchema, GetStatusResponse } from '$lib/gen/ui_pb';

// StatusVerdict is INDEX HEALTH only — exactly five members, no sixth.
// `pnpm check` is the guard: a commit value assigned into this field
// does not type-check.
export type StatusVerdict = 'ok' | 'stale' | 'no-index' | 'indexing' | 'unknown';

// CommitKnowledge is the orthogonal dimension above — never folded into
// StatusVerdict.
export type CommitKnowledge = 'known' | 'unknown';

export interface IndexStatus {
	verdict: StatusVerdict;
	commit: CommitKnowledge;
}

const UNKNOWN_STATUS: IndexStatus = { verdict: 'unknown', commit: 'unknown' };

// classifyStatus is a pure function over one GetStatus answer. It never
// throws — a shape this function does not recognize (initialized false,
// store_exists true, indexing_in_progress false — not a combination
// degrade.go's degradedStatus produces today) degrades to the same
// 'unknown' verdict a rejected call reports, rather than inventing a
// sixth verdict for a case the server never actually sends.
export function classifyStatus(response: GetStatusResponse): IndexStatus {
	const commit: CommitKnowledge = response.commitSha ? 'known' : 'unknown';

	if (response.initialized) {
		return { verdict: response.stale ? 'stale' : 'ok', commit };
	}
	if (!response.storeExists) {
		return { verdict: 'no-index', commit };
	}
	if (response.indexingInProgress) {
		return { verdict: 'indexing', commit };
	}
	return { verdict: 'unknown', commit };
}

// VIEW_LOCAL_PARAMS lists query params that select no distinct view and
// therefore must not mint a new navigation identity by themselves
// (WR-04). `q` is Browse's search box (+page.svelte:64-66 documents it
// as view-local, undebounced, written on every keystroke by D-11's own
// shareable-URL contract) — before this exclusion, typing a five-letter
// query fired five extra GetStatus RPCs on top of the one the
// navigation contract allows, each one re-entering withEngine/openEngine
// server-side and contending with the store lock during an active
// re-index.
const VIEW_LOCAL_PARAMS = ['q'];

// navigationIdentity is the ONE normalizer both the gate's constructor
// call site and the layout's navigation effect derive their identity
// from (Task 2 wires both) — pathname plus the SORTED query string,
// minus VIEW_LOCAL_PARAMS, so two renders of the same view never read
// as two distinct navigations, and the two call sites can never
// disagree because there is only one implementation.
export function navigationIdentity(url: URL): string {
	const params = new URLSearchParams(url.searchParams);
	for (const k of VIEW_LOCAL_PARAMS) params.delete(k);
	params.sort();
	const qs = params.toString();
	return qs ? `${url.pathname}?${qs}` : url.pathname;
}

// StatusClient is the minimal shape createStatusGate needs from a
// UIService client — declared independently of the real generated
// client (browse-state.ts's NodeDetailClient/ImpactClient convention)
// so a test stub can satisfy it with no Connect transport.
export interface StatusClient {
	getStatus(
		request: MessageInitShape<typeof GetStatusRequestSchema>,
		options?: { signal?: AbortSignal }
	): Promise<GetStatusResponse>;
}

export interface StatusGate {
	// subscribe follows the store contract (subscribe(run) => unsubscribe,
	// invoked synchronously at least once with the current value) so
	// web/src/routes/+layout.svelte and any descendant needing the same
	// data can bind to it directly. A subscription only registers a
	// listener — it never triggers a fetch.
	subscribe(run: (status: IndexStatus) => void): () => void;
	// The gate's ONE trigger besides its own construction (D-05): a call
	// whose identity equals the one last acted on is a no-op. That
	// comparison is what makes the constructor's own fetch and an
	// effect firing immediately after construction with the same
	// identity compose into exactly one fetch, regardless of how the
	// mounting site is written.
	notifyNavigated(identity: string): void;
}

// createStatusGate fetches exactly ONCE on creation — unconditional,
// never contingent on any caller ever invoking the navigation method —
// and records initialNavigationIdentity as the identity that creation
// fetch was for. Making the fetch conditional on a caller's first call
// would let a mounting mistake produce a gate that never fetches at
// all and renders as permanently unknown, a silent failure; taking the
// identity as a constructor argument keeps the fetch unconditional
// while still making the identity guard implementable.
export function createStatusGate(
	client: StatusClient,
	initialNavigationIdentity: string
): StatusGate {
	let current: IndexStatus = UNKNOWN_STATUS;
	let lastIdentity = initialNavigationIdentity;
	const listeners = new Set<(status: IndexStatus) => void>();

	// requestId (WR-08): fetchStatus had no cancellation or
	// response-identity guard, so whichever GetStatus promise settled
	// LAST won via emit — an ordinary loopback interleaving (fetch A
	// starts while stale, fetch B starts 50ms later once indexing
	// finished and settles first) could revert the banner to a stale
	// verdict that then never updates until the next navigation. Every
	// other async surface in this phase already guards this
	// (search.ts's liveRequestId, browse-state.ts's NavigationGeneration)
	// — mint a monotonic id per fetch and drop any response whose id no
	// longer matches the most recent one, mirroring that convention.
	let requestId = 0;

	function emit(status: IndexStatus): void {
		current = status;
		for (const listener of listeners) listener(current);
	}

	function fetchStatus(): void {
		const id = ++requestId;
		client
			.getStatus({})
			.then((response) => {
				if (id !== requestId) return; // superseded by a later fetch
				emit(classifyStatus(response));
			})
			.catch(() => {
				if (id !== requestId) return; // superseded by a later fetch
				// A rejected GetStatus call classifies as 'unknown' and never
				// throws — this is the last line between a transport failure
				// and a gate that silently stops updating.
				emit(UNKNOWN_STATUS);
			});
	}

	fetchStatus();

	return {
		subscribe(run) {
			listeners.add(run);
			run(current);
			return () => {
				listeners.delete(run);
			};
		},
		notifyNavigated(identity) {
			if (identity === lastIdentity) return;
			lastIdentity = identity;
			fetchStatus();
		}
	};
}
