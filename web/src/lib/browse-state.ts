// browse-state.ts is the load seam between a parsed URL and a renderable
// view state: it maps BrowseParams onto a GetNodeDetailRequest field-for-
// field, awaits the call, and returns a BrowseTargetState — catching
// every rejection through rpc-errors.ts's classifyRpcError so it never
// throws. Kept as a plain module rather than logic inline in
// +page.svelte, which is what makes this whole slice testable without a
// SvelteKit runtime (no onMount, no $app/state needed here).
import type { GetNodeDetailRequestSchema, GetNodeDetailResponse } from '$lib/gen/ui_pb';
import { NodeDetailMode } from '$lib/gen/ui_pb';
import type { MessageInitShape } from '@bufbuild/protobuf';
import type { BrowseParams } from '$lib/browse-url';
import { classifyRpcError, type RpcFailure } from '$lib/rpc-errors';

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

		// This tracer plan wires the FILE mode only — SINGLE_DEF/MULTI_DEF
		// rendering (BRW-04/BRW-05) is built by later plans in this phase
		// (03-05/03-06), which extend this function rather than replacing
		// it. An unexpected/unhandled mode here is a named failure state,
		// never a thrown exception or a blank pane.
		return {
			kind: 'failed',
			failure: {
				kind: 'unknown',
				message: `browse-state: node detail mode ${response.mode} is not yet rendered by this view`
			}
		};
	} catch (err) {
		return { kind: 'failed', failure: classifyRpcError(err) };
	}
}
