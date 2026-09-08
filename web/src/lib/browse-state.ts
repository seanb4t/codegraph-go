// browse-state.ts is the load seam between a parsed URL and a renderable
// view state: it maps BrowseParams onto a GetNodeDetailRequest field-for-
// field, awaits the call, and returns a BrowseTargetState — catching
// every rejection through rpc-errors.ts's classifyRpcError so it never
// throws. Kept as a plain module rather than logic inline in
// +page.svelte, which is what makes this whole slice testable without a
// SvelteKit runtime (no onMount, no $app/state needed here).
//
// 03-07 Task 2 extends this module (never a parallel one) with the
// symbol-target and multi-definition BrowseTargetState members, and
// loadBlastRadius — the Impact-rpc loader for the blast-radius region.
// The three-mode response is discriminated by reading the response's
// OWN `mode` field, never by inferring it from which fields happen to
// be non-empty (ui.proto's own doc comment: a client reading the wrong
// field for the current mode gets a zero value, not an error — an
// in-cap single-def candidate that genuinely calls nothing would
// otherwise be indistinguishable from a malformed response).
import type {
	GetNodeDetailRequestSchema,
	GetNodeDetailResponse,
	ImpactRequestSchema,
	ImpactResponse,
	Node,
	NodeDefinition,
	Location
} from '$lib/gen/ui_pb';
import { NodeDetailMode } from '$lib/gen/ui_pb';
import type { MessageInitShape } from '@bufbuild/protobuf';
import type { BrowseParams } from '$lib/browse-url';
import { classifyRpcError, type RpcFailure } from '$lib/rpc-errors';

// SourceRender is the single-def variant's own source shape — carried
// as a nested object (unlike the 'file' variant's flattened fields)
// because it is genuinely OPTIONAL here: a single-def candidate's
// source is only gathered for in-cap matches (NodeDefinition's own
// detail_gathered discipline), so "no source" is a real, renderable
// case for this mode in a way it never is for a bare file open.
export type SourceRender = {
	content: Uint8Array;
	truncated: boolean;
	totalLines: number;
	returnedLines: number;
};

export type BrowseTargetState =
	| { kind: 'idle' }
	| { kind: 'loading' }
	| {
			kind: 'file';
			path: string;
			source: Uint8Array;
			truncated: boolean;
			totalLines: number;
			returnedLines: number;
	  }
	| {
			kind: 'single-def';
			node: Node;
			calls: Node[];
			calledBy: Node[];
			source?: SourceRender;
	  }
	| {
			kind: 'multi-def';
			symbol: string;
			definitions: NodeDefinition[];
			totalCandidates: number;
	  }
	| { kind: 'failed'; failure: RpcFailure };

// NodeDetailClient is the minimal shape loadBrowseTarget needs from a
// UIService client — matches the real generated uiClient's getNodeDetail
// method (web/src/lib/client.ts) but is declared independently so a test
// stub can satisfy it without constructing a real Connect transport.
export interface NodeDetailClient {
	getNodeDetail(
		request: MessageInitShape<typeof GetNodeDetailRequestSchema>,
		options?: { signal?: AbortSignal }
	): Promise<GetNodeDetailResponse>;
}

export async function loadBrowseTarget(
	params: BrowseParams,
	client: NodeDetailClient,
	signal?: AbortSignal
): Promise<BrowseTargetState> {
	if (!params.symbol && !params.file) {
		return { kind: 'idle' };
	}

	try {
		const response = await client.getNodeDetail(
			{
				symbol: params.symbol ?? '',
				file: params.file ?? '',
				line: params.line
			},
			signal ? { signal } : undefined
		);

		if (response.mode === NodeDetailMode.FILE && response.source) {
			return {
				kind: 'file',
				path: response.path,
				source: response.source.content,
				truncated: response.source.truncated,
				totalLines: response.source.totalLines,
				returnedLines: response.source.returnedLines
			};
		}

		// SINGLE_DEF: node, calls and called_by are populated per
		// ui.proto's own mode table. The mode field alone selects this
		// branch — a single-def response with an EMPTY calls list is
		// still read as single-def, never inferred as "something else"
		// from the empty array.
		if (response.mode === NodeDetailMode.SINGLE_DEF && response.node) {
			return {
				kind: 'single-def',
				node: response.node,
				calls: response.calls,
				calledBy: response.calledBy,
				source: response.source
					? {
							content: response.source.content,
							truncated: response.source.truncated,
							totalLines: response.source.totalLines,
							returnedLines: response.source.returnedLines
						}
					: undefined
			};
		}

		// MULTI_DEF: symbol, definitions and total_candidates are
		// populated. Rendering this state lands in plan 03-08 — the
		// state itself is produced here, distinct from both other modes.
		if (response.mode === NodeDetailMode.MULTI_DEF) {
			return {
				kind: 'multi-def',
				symbol: response.symbol,
				definitions: response.definitions,
				totalCandidates: response.totalCandidates
			};
		}

		// An unexpected/unhandled mode (or a mode whose required fields
		// are absent, e.g. FILE with no source, SINGLE_DEF with no node)
		// is a named failure state, never a thrown exception or a blank
		// pane.
		return {
			kind: 'failed',
			failure: {
				kind: 'unknown',
				message: `browse-state: node detail mode ${response.mode} is not rendered by this view`
			}
		};
	} catch (err) {
		return { kind: 'failed', failure: classifyRpcError(err) };
	}
}

// BlastRadiusState is loadBlastRadius's own named result — kept distinct
// from BrowseTargetState because a symbol view's node detail and its
// blast radius are two independently loading halves of one screen (see
// NavigationGeneration below), never one combined state.
export type BlastRadiusState =
	| { kind: 'idle' }
	| { kind: 'loading' }
	| { kind: 'loaded'; depth: number; nodeCount: number; edgeCount: number; affected: Location[] }
	| { kind: 'failed'; failure: RpcFailure };

// ImpactClient is the minimal shape loadBlastRadius needs from a
// UIService client — mirrors NodeDetailClient's own declared-
// independently-of-the-real-client convention, so a test stub can
// satisfy it without a real Connect transport.
export interface ImpactClient {
	impact(
		request: MessageInitShape<typeof ImpactRequestSchema>,
		options?: { signal?: AbortSignal }
	): Promise<ImpactResponse>;
}

// loadBlastRadius issues the Impact rpc for a symbol target. depth is
// passed through EXACTLY as parsed: an absent depth sends the proto
// zero value (0) — the field's own "caller did not set a depth" wire
// convention (mirroring search.ts's limit/max_files/depth: 0
// precedent), never a client-side default. depth and limit are NOT
// symmetric on the wire, and this function does not pretend they are:
// the server rejects only a negative depth and otherwise silently
// bounds an above-ceiling value rather than refusing it (see
// browse-state.test.ts and the SUMMARY for the full validate.go/
// traverse.go citation this rationale rests on — kept out of this file
// so its own vocabulary doesn't trip this module's own "no second copy
// of the server's bound" guard below) — so an out-of-range depth here
// SUCCEEDS, and the committed state reports the RESPONSE's own echoed
// depth (the value Impact actually used), never re-displaying the
// request's raw value as though it were what ran. This function adds
// no client-side check for an over-ceiling depth: that would be a
// second copy of a bound the server already owns, and it would refuse
// a request the server is willing to answer.
export async function loadBlastRadius(
	params: BrowseParams,
	client: ImpactClient,
	signal?: AbortSignal
): Promise<BlastRadiusState> {
	if (!params.symbol) {
		return { kind: 'idle' };
	}

	try {
		const response = await client.impact(
			{ symbol: params.symbol, depth: params.depth ?? 0 },
			signal ? { signal } : undefined
		);
		return {
			kind: 'loaded',
			depth: response.depth,
			nodeCount: response.nodeCount,
			edgeCount: response.edgeCount,
			affected: response.affected
		};
	} catch (err) {
		return { kind: 'failed', failure: classifyRpcError(err) };
	}
}

// NavigationGeneration + NavigationGate close the cross-load race a
// per-call AbortController alone cannot: two loaders (loadBrowseTarget,
// loadBlastRadius) can each correctly supersede their OWN predecessor
// while the view still ends up combining a node-detail result from one
// URL state with a blast-radius result from a DIFFERENT one, because
// abort is best-effort and a response already in flight can still
// arrive. ONE generation, minted ONCE per URL change by the ROUTE
// (never by a loader — two loaders minting their own generations is
// the defect this closes, not a feature) and stamped on both loads,
// closes that race: a resolving handler compares its stamp against the
// CURRENT generation before committing anything, discarding a stale
// result whether or not its abort signal fired.
export type NavigationGeneration = number;

export interface NavigationGate {
	// advance mints a new generation and makes it the current one,
	// superseding whatever was current before. Call this ONCE per URL
	// change, before starting either load.
	advance(): NavigationGeneration;
	// isCurrent is true only for the MOST RECENTLY minted generation —
	// the guard every resolving handler checks before committing.
	isCurrent(generation: NavigationGeneration): boolean;
}

export function createNavigationGate(): NavigationGate {
	let current: NavigationGeneration = 0;
	return {
		advance() {
			current += 1;
			return current;
		},
		isCurrent(generation) {
			return generation === current;
		}
	};
}
