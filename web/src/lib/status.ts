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
	// commitSha is the ADDITIVE field 04-05 Task 1 adds (cycle-2 review):
	// the raw commit SHA `commit`'s presence flag was deliberately
	// derived from, carried alongside it rather than instead of it.
	// '' means "not recorded" — the wire's own representation
	// (ui.proto:151/GetHealthResponse's commit_sha), so there is one
	// representation of unknown, not two. Required (not optional) so
	// `pnpm check` enumerates every construction site rather than
	// letting one silently default to undefined. `commit === 'known'`
	// if and only if `commitSha !== ''` — the invariant every call site
	// below preserves.
	commitSha: string;
}

const UNKNOWN_STATUS: IndexStatus = { verdict: 'unknown', commit: 'unknown', commitSha: '' };

// StatusLikeFields (D-07): the exact five fields
// classifyStatus reads, expressed as a Pick over GetStatusResponse so
// the two can never drift apart — `pnpm check` fails if a field is
// renamed on either side. This is the whole reason D-07 gave
// WatchGraphEvent GetStatusResponse's own field NAMES: WatchGraphEvent
// structurally satisfies this type too (same five names, same types),
// so classifyStatus consumes a live event directly, with no second
// representation of the same state and no translation layer.
export type StatusLikeFields = Pick<
	GetStatusResponse,
	'initialized' | 'stale' | 'storeExists' | 'indexingInProgress' | 'commitSha'
>;

// classifyStatus is a pure function over EITHER a full GetStatus answer
// or a live WatchGraphEvent — both satisfy StatusLikeFields structurally.
// It never throws — a shape this function does not recognize
// (initialized false, store_exists true, indexing_in_progress false —
// not a combination degrade.go's degradedStatus produces today)
// degrades to the same 'unknown' verdict a rejected call reports, rather
// than inventing a sixth verdict for a case the server never actually
// sends.
export function classifyStatus(response: StatusLikeFields): IndexStatus {
	const commit: CommitKnowledge = response.commitSha ? 'known' : 'unknown';
	const commitSha = response.commitSha;

	if (response.initialized) {
		return { verdict: response.stale ? 'stale' : 'ok', commit, commitSha };
	}
	if (!response.storeExists) {
		return { verdict: 'no-index', commit, commitSha };
	}
	if (response.indexingInProgress) {
		return { verdict: 'indexing', commit, commitSha };
	}
	return { verdict: 'unknown', commit, commitSha };
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
//
// This list is GLOBAL — applied under every route — and stays that way:
// `q` selects no distinct view under ANY route it might appear on. It is
// never widened in place to carry a route-specific parameter; see
// ROUTE_LOCAL_PARAMS immediately below for that case.
const VIEW_LOCAL_PARAMS = ['q'];

// ROUTE_LOCAL_PARAMS is the route-SCOPED counterpart to VIEW_LOCAL_PARAMS
// (04-01 Task 2, T-04-32): a parameter that selects a different QUERY
// within one view is view-local for that route; a parameter that selects
// a different VIEW is not, and must keep minting a new identity.
//
// `/workbench`'s `mode`/`symbol`/`file`/`depth`/`limit` are exactly this
// first kind under `/workbench` — every Workbench control change is a
// refinement of the SAME view, never a navigation to a different one —
// but `symbol`/`file`/`depth`/`limit` are ALSO Browse parameters
// (browse-url.ts:24) where each one legitimately selects a DIFFERENT
// Browse view (a different symbol, a different file). Excluding them
// globally would silently stop refreshing the index verdict as a
// developer moves through Browse — an invisible behavior change to a
// shipped Phase-3 contract. The table below is keyed by pathname so the
// exclusion applies only where it is actually view-local.
//
// Without this table, `web/src/routes/+layout.svelte`'s `$effect` (which
// calls `statusGate.notifyNavigated(navigationIdentity(page.url))` on
// every `page.url` change) would fire one extra GetStatus RPC per
// Workbench control change — a symbol keystroke, a limit edit, later a
// depth slider move — each one re-entering withEngine/openEngine
// server-side, exactly the amplification the `q` exclusion above was
// added to fix, reintroduced through a different route.
const ROUTE_LOCAL_PARAMS: Record<string, readonly string[]> = {
	'/workbench': ['mode', 'symbol', 'file', 'depth', 'limit']
};

// navigationIdentity is the ONE normalizer both the gate's constructor
// call site and the layout's navigation effect derive their identity
// from (Task 2 wires both) — pathname plus the SORTED query string,
// minus VIEW_LOCAL_PARAMS and minus this route's own ROUTE_LOCAL_PARAMS
// entry (if any), so two renders of the same view never read as two
// distinct navigations, and the two call sites can never disagree
// because there is only one implementation.
export function navigationIdentity(url: URL): string {
	const params = new URLSearchParams(url.searchParams);
	for (const k of VIEW_LOCAL_PARAMS) params.delete(k);
	for (const k of ROUTE_LOCAL_PARAMS[url.pathname] ?? []) params.delete(k);
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
	// applyLiveEvent (criterion 1): classifies a live event
	// through the SAME classifyStatus fetchStatus uses and emits —
	// WITHOUT calling getStatus. This is what lets the health/staleness
	// chrome update from the event itself rather than a round trip the
	// event merely triggers. `generation` and `epoch` identify the event
	// for the dedup check below; they play no role in classification.
	//
	// One shared monotonic counter orders every fetchStatus START and
	// every applied live event: whichever mints the id LAST is current,
	// and an operation may emit only while its id is still current. A
	// live event applied while a fetch is in flight therefore invalidates
	// that fetch (its response is dropped when it settles); a fetch
	// started after a live application invalidates a LATER re-delivery of
	// that same (epoch, generation) event — recognized as a duplicate and
	// skipped without mint-and-emit, so it cannot un-invalidate a fetch
	// that started after the first, genuine application.
	applyLiveEvent(fields: StatusLikeFields & { generation: bigint; epoch: number }): void;
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

	// lastAppliedLive tracks the (epoch, generation) pair
	// of the most recently APPLIED live event, so a re-delivery of that
	// exact same event is recognized as a duplicate and skipped — it
	// mints no new id and therefore cannot supersede a fetch that
	// legitimately started after the first, genuine application.
	let lastAppliedLive: { epoch: number; generation: bigint } | null = null;

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
		},
		applyLiveEvent(fields) {
			if (
				lastAppliedLive !== null &&
				lastAppliedLive.epoch === fields.epoch &&
				lastAppliedLive.generation === fields.generation
			) {
				return; // a re-delivery of the same event — not new information
			}
			lastAppliedLive = { epoch: fields.epoch, generation: fields.generation };
			const id = ++requestId;
			// Synchronous end-to-end: nothing can supersede this id between
			// minting it and emitting, so it always wins UNLESS it was
			// itself recognized as a duplicate above.
			if (id !== requestId) return;
			emit(classifyStatus(fields));
		}
	};
}
