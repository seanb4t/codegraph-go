// search.ts is the search controller for BRW-01/BRW-08/NAV-03 (D-14, D-15,
// D-16). The trigger split is forced by the wire, not chosen for taste:
// `Search`/`Files` return bare Location/FileEntry projections, while
// `Explore` attaches a bounded SourceBlob PER MATCHED FILE (up to
// defaultMaxFiles=5, internal/query/validate.go) — roughly an order of
// magnitude more expensive, since it reads files off disk. So typing
// narrows through `Search`/`Files` live; Enter (submit()) asks the more
// expensive question via `Explore`, and the two never share a trigger.
//
// D-15: results are exposed in EXACTLY the order each RPC returned them.
// Location and FileEntry carry no comparable ranking signal, so this
// module never sorts, ranks, or merges them — a merged/re-ranked list
// would be a client-invented heuristic that silently disagrees with the server's own
// ordering and has nothing to be tested against.
//
// D-16: ~150ms debounce (the low end of the general 150-300ms guidance —
// this server is on loopback, not crossing a network) and a 2-character
// minimum (symbol search is a names case). Cancellation is independent of
// the delay and non-negotiable: debouncing alone does not fix out-of-order
// responses, since a slow response to a shorter prefix can still land
// after the response to a longer one. Every dispatch gets its own
// AbortController passed into the Connect call's own `signal` option, the
// previous one is aborted before a new one starts, AND a monotonically
// increasing request identity backs that up — the abort handles the
// common case, the identity guard handles the race the abort loses (e.g.
// an abort signal a stub/test client does not honor).
//
// Rejections are mapped through the ONE shared classifier (D-04,
// rpc-errors.ts) — no second error vocabulary here.
import { writable, type Readable } from 'svelte/store';
import type { MessageInitShape } from '@bufbuild/protobuf';
import type {
	SearchRequestSchema,
	SearchResponse,
	FilesRequestSchema,
	FilesResponse,
	ExploreRequestSchema,
	ExploreResponse,
	Location,
	FileEntry,
	ExploreGroup,
	BlastEntry
} from '$lib/gen/ui_pb';
import { classifyRpcError, type RpcFailure } from '$lib/rpc-errors';

export const SEARCH_DEBOUNCE_MS = 150;
export const SEARCH_MIN_CHARS = 2;

// LiveSearchResults holds Search's and Files' own result lists,
// UNMERGED — D-15's two labelled sections read straight off these two
// fields, each preserving its own RPC's order.
export type LiveSearchResults = {
	symbols: Location[];
	files: FileEntry[];
};

// ExploreOutcome is a discriminated union so "no results" (empty=true, a
// SUCCESSFUL response per ui.proto's own doc comment) and "the call
// failed" are two different values a caller cannot accidentally collapse
// — mirroring the wire's own empty-is-never-an-error contract.
export type ExploreOutcome =
	| {
			kind: 'results';
			query: string;
			symbolCount: number;
			groups: ExploreGroup[];
			blasts: BlastEntry[];
	  }
	| { kind: 'empty'; query: string }
	| { kind: 'failed'; failure: RpcFailure };

export type SearchControllerState = {
	query: string;
	live: LiveSearchResults;
	// Populated only if a live Search/Files dispatch that was NOT
	// superseded by a later one ends in rejection — classified through
	// the shared mapper (Rule 2: a swallowed live-search failure would
	// otherwise look identical to "no matches yet" to the user).
	liveFailure: RpcFailure | undefined;
	// undefined until the first submit() call.
	explore: ExploreOutcome | undefined;
};

// SearchClient is the minimal shape the controller needs from a
// UIService client — matches the real generated uiClient's search/files/
// explore methods (web/src/lib/client.ts) but is declared independently
// so a test stub can satisfy it without constructing a real Connect
// transport (browse-state.ts's NodeDetailClient precedent).
export interface SearchClient {
	search(
		request: MessageInitShape<typeof SearchRequestSchema>,
		options?: { signal?: AbortSignal }
	): Promise<SearchResponse>;
	files(
		request: MessageInitShape<typeof FilesRequestSchema>,
		options?: { signal?: AbortSignal }
	): Promise<FilesResponse>;
	explore(
		request: MessageInitShape<typeof ExploreRequestSchema>,
		options?: { signal?: AbortSignal }
	): Promise<ExploreResponse>;
}

export interface SearchController extends Readable<SearchControllerState> {
	// setQuery is the live path: updates the current query value and
	// (re)starts the debounce. A value shorter than SEARCH_MIN_CHARS
	// issues no RPC at all and clears any stale live results.
	setQuery(value: string): void;
	// submit is the explore path: issues Explore for whatever the
	// CURRENT query value is, on its own cancellation lineage,
	// independent of the live path — a submit never clears live results.
	submit(): void;
}

export function createSearchController(client: SearchClient): SearchController {
	const state = writable<SearchControllerState>({
		query: '',
		live: { symbols: [], files: [] },
		liveFailure: undefined,
		explore: undefined
	});

	let debounceTimer: ReturnType<typeof setTimeout> | undefined;
	let liveAbort: AbortController | undefined;
	let liveRequestId = 0;
	let exploreAbort: AbortController | undefined;
	let exploreRequestId = 0;

	function dispatchLive(term: string): void {
		if (liveAbort) liveAbort.abort();
		const abort = new AbortController();
		liveAbort = abort;
		const requestId = ++liveRequestId;

		// D-15: two independent RPCs, never merged. Files' pattern is a
		// substring-style glob (`*term*`) over the forward-slashed repo
		// path — internal/query.FilesOptions.Pattern is a raw
		// path/filepath.Match glob passed straight through with no
		// substring convenience of its own, so a search BOX (not a glob
		// box) needs this module to build that shape. limit/max_files/
		// depth are all passed as 0 — every one of those fields' own doc
		// comment in ui.proto states the server's own bound (MaxLimit,
		// defaultMaxFiles, "0 means unlimited" for Files' depth) is
		// authoritative; this module adds no second copy of any of them.
		Promise.all([
			client.search({ term, kind: '', limit: 0 }, { signal: abort.signal }),
			client.files(
				{ pattern: `*${term}*`, filter: '', dir: '', depth: 0, format: 'flat' },
				{ signal: abort.signal }
			)
		])
			.then(([searchResp, filesResp]) => {
				if (requestId !== liveRequestId) return; // superseded — discard
				// FilesResponse is a UNION (ui.proto's own doc comment): read
				// format before deciding which field is populated, never
				// assume "flat".
				const files = filesResp.format === 'flat' ? filesResp.files : [];
				state.update((s) => ({
					...s,
					live: { symbols: searchResp.locations, files },
					liveFailure: undefined
				}));
			})
			.catch((err: unknown) => {
				if (requestId !== liveRequestId) return; // superseded — discard, not a real failure
				state.update((s) => ({ ...s, liveFailure: classifyRpcError(err) }));
			});
	}

	function setQuery(value: string): void {
		state.update((s) => ({ ...s, query: value }));

		if (debounceTimer !== undefined) {
			clearTimeout(debounceTimer);
			debounceTimer = undefined;
		}

		if (value.length < SEARCH_MIN_CHARS) {
			// A query shorter than the minimum issues no RPC at all — and
			// cancels whatever was still in flight for a longer prefix, so a
			// backspace-to-short never lets a stale result land later.
			if (liveAbort) {
				liveAbort.abort();
				liveAbort = undefined;
			}
			liveRequestId += 1; // invalidate any in-flight response too
			state.update((s) => ({ ...s, live: { symbols: [], files: [] }, liveFailure: undefined }));
			return;
		}

		debounceTimer = setTimeout(() => {
			debounceTimer = undefined;
			dispatchLive(value);
		}, SEARCH_DEBOUNCE_MS);
	}

	function submit(): void {
		let currentQuery = '';
		const unsubscribe = state.subscribe((s) => {
			currentQuery = s.query;
		});
		unsubscribe();

		if (exploreAbort) exploreAbort.abort();
		const abort = new AbortController();
		exploreAbort = abort;
		const requestId = ++exploreRequestId;

		// max_files: 0 — Explore's own H21 adaptive budget and
		// defaultMaxFiles=5 already bound this server-side; this module
		// adds no second copy.
		client
			.explore({ query: currentQuery, maxFiles: 0 }, { signal: abort.signal })
			.then((resp) => {
				if (requestId !== exploreRequestId) return; // superseded — discard
				const outcome: ExploreOutcome = resp.empty
					? { kind: 'empty', query: resp.query }
					: {
							kind: 'results',
							query: resp.query,
							symbolCount: resp.symbolCount,
							groups: resp.groups,
							blasts: resp.blasts
						};
				// Never clears `live` — BRW-08's word is "alongside".
				state.update((s) => ({ ...s, explore: outcome }));
			})
			.catch((err: unknown) => {
				if (requestId !== exploreRequestId) return; // superseded — discard
				state.update((s) => ({ ...s, explore: { kind: 'failed', failure: classifyRpcError(err) } }));
			});
	}

	return {
		subscribe: state.subscribe,
		setQuery,
		submit
	};
}
