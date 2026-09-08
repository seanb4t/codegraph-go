// browse-state.test.ts — 03-07 Task 2: symbol targets and blast radius
// in the load seam. This file is created and run RED before
// loadBlastRadius / createNavigationGate / the single-def and multi-def
// BrowseTargetState variants exist in web/src/lib/browse-state.ts
// (recorded in the SUMMARY), then made GREEN.
import { describe, it, expect } from 'vitest';
import { ConnectError, Code } from '@connectrpc/connect';

import {
	loadBrowseTarget,
	loadBlastRadius,
	createNavigationGate,
	type NodeDetailClient,
	type ImpactClient
} from '$lib/browse-state';
import { NodeDetailMode } from '$lib/gen/ui_pb';
import type { GetNodeDetailResponse, ImpactResponse, Node } from '$lib/gen/ui_pb';

// node/singleDefResponse/multiDefResponse build plain-object stubs cast
// through `unknown` — mirroring browse-tracer.test.ts's own stub
// convention (03-04): these carry only the fields loadBrowseTarget
// actually reads, not the generated message's full prototype machinery.
function node(name: string, overrides: Partial<Node> = {}): Node {
	return {
		id: name,
		kind: 'func',
		name,
		qualifiedName: `pkg.${name}`,
		filePath: `${name}.go`,
		language: 'go',
		startLine: 1,
		endLine: 5,
		startCol: 0,
		endCol: 0,
		signature: '',
		docstring: '',
		visibility: '',
		isExported: true,
		returnType: '',
		...overrides
	} as unknown as Node;
}

function singleDefResponse(overrides: Partial<GetNodeDetailResponse> = {}): GetNodeDetailResponse {
	return {
		mode: NodeDetailMode.SINGLE_DEF,
		path: '',
		node: node('Foo'),
		calls: [],
		calledBy: [],
		symbol: '',
		definitions: [],
		totalCandidates: 0,
		source: undefined,
		...overrides
	} as unknown as GetNodeDetailResponse;
}

function multiDefResponse(overrides: Partial<GetNodeDetailResponse> = {}): GetNodeDetailResponse {
	return {
		mode: NodeDetailMode.MULTI_DEF,
		path: '',
		node: undefined,
		calls: [],
		calledBy: [],
		symbol: 'Foo',
		definitions: [],
		totalCandidates: 0,
		source: undefined,
		...overrides
	} as unknown as GetNodeDetailResponse;
}

function stubDetailClient(response: unknown): NodeDetailClient {
	return { getNodeDetail: () => Promise.resolve(response as GetNodeDetailResponse) };
}

describe('loadBrowseTarget: mode discrimination (never inferred from field emptiness)', () => {
	it('a single-def response with an EMPTY calls list is still classified as single-def', async () => {
		const response = singleDefResponse({ calls: [], calledBy: [] });
		const state = await loadBrowseTarget({ symbol: 'Foo', unknown: [] }, stubDetailClient(response));
		expect(state.kind).toBe('single-def');
		if (state.kind === 'single-def') {
			expect(state.calls).toEqual([]);
			expect(state.calledBy).toEqual([]);
			expect(state.node.name).toBe('Foo');
		}
	});

	it('a single-def response carries node, calls, called-by and source when populated', async () => {
		const response = singleDefResponse({
			node: node('Foo'),
			calls: [node('Bar')],
			calledBy: [node('Baz')],
			source: {
				content: new TextEncoder().encode('func Foo() {}'),
				truncated: false,
				totalLines: 1,
				totalBytes: 13,
				returnedLines: 1,
				returnedBytes: 13
			} as unknown as GetNodeDetailResponse['source']
		});
		const state = await loadBrowseTarget({ symbol: 'Foo', unknown: [] }, stubDetailClient(response));
		expect(state.kind).toBe('single-def');
		if (state.kind === 'single-def') {
			expect(state.calls.map((n) => n.name)).toEqual(['Bar']);
			expect(state.calledBy.map((n) => n.name)).toEqual(['Baz']);
			expect(state.source?.totalLines).toBe(1);
		}
	});

	it('a multi-def response yields a distinct multi-def state carrying the candidate list and true total', async () => {
		const definitions = [
			{ node: node('Foo'), calls: [], calledBy: [], detailGathered: true, source: undefined }
		] as unknown as GetNodeDetailResponse['definitions'];
		const response = multiDefResponse({ symbol: 'Foo', definitions, totalCandidates: 5 });
		const state = await loadBrowseTarget({ symbol: 'Foo', unknown: [] }, stubDetailClient(response));
		expect(state.kind).toBe('multi-def');
		if (state.kind === 'multi-def') {
			expect(state.symbol).toBe('Foo');
			expect(state.totalCandidates).toBe(5);
			expect(state.definitions).toHaveLength(1);
		}
	});
});

describe('loadBlastRadius: depth passthrough and the clamp/refuse asymmetry (D-12)', () => {
	it('sends depth verbatim (including above the server ceiling) and reports the RESPONSE-echoed depth, not the request depth', async () => {
		let captured: { symbol: string; depth: number } | undefined;
		const client: ImpactClient = {
			impact: (req) => {
				captured = req as { symbol: string; depth: number };
				return Promise.resolve({
					symbol: 'Foo',
					depth: 50,
					nodeCount: 3,
					edgeCount: 2,
					affected: [{ name: 'Bar', kind: 'func', filePath: 'b.go', startLine: 1 }]
				} as unknown as ImpactResponse);
			}
		};

		const state = await loadBlastRadius({ symbol: 'Foo', depth: 999, unknown: [] }, client);

		expect(captured?.depth).toBe(999); // sent verbatim, never clamped client-side
		expect(state.kind).toBe('loaded');
		if (state.kind === 'loaded') {
			expect(state.depth).toBe(50); // the response's OWN clamped value
			expect(state.nodeCount).toBe(3);
			expect(state.affected).toHaveLength(1);
		}
	});

	it('omits depth (sends the proto zero value) when absent from params — no client-side default', async () => {
		let captured: { symbol: string; depth: number } | undefined;
		const client: ImpactClient = {
			impact: (req) => {
				captured = req as { symbol: string; depth: number };
				return Promise.resolve({
					symbol: 'Foo',
					depth: 2,
					nodeCount: 0,
					edgeCount: 0,
					affected: []
				} as unknown as ImpactResponse);
			}
		};

		await loadBlastRadius({ symbol: 'Foo', unknown: [] }, client);
		expect(captured?.depth).toBe(0);
	});

	it('a rejection yields the failed state with the classified failure, never a thrown exception', async () => {
		const client: ImpactClient = {
			impact: () => Promise.reject(new ConnectError('bad depth', Code.InvalidArgument))
		};
		const state = await loadBlastRadius({ symbol: 'Foo', unknown: [] }, client);
		expect(state.kind).toBe('failed');
		if (state.kind === 'failed') expect(state.failure.kind).toBe('invalid-input');
	});

	it('no symbol target yields idle without calling the client', async () => {
		let called = false;
		const client: ImpactClient = {
			impact: () => {
				called = true;
				return Promise.resolve({} as ImpactResponse);
			}
		};
		const state = await loadBlastRadius({ unknown: [] }, client);
		expect(state.kind).toBe('idle');
		expect(called).toBe(false);
	});

	it('an in-flight load is abortable, and the caller discipline (skip commit if aborted) discards it regardless of outcome', async () => {
		const controller = new AbortController();
		let resolveImpact: ((v: ImpactResponse) => void) | undefined;
		const client: ImpactClient = {
			impact: () =>
				new Promise<ImpactResponse>((resolve) => {
					resolveImpact = resolve;
				})
		};

		const pending = loadBlastRadius({ symbol: 'Foo', unknown: [] }, client, controller.signal);
		controller.abort();
		resolveImpact?.({
			symbol: 'Foo',
			depth: 2,
			nodeCount: 1,
			edgeCount: 0,
			affected: []
		} as unknown as ImpactResponse);
		await pending;

		// The same caller-side guard +page.svelte's own effect uses:
		// discard whatever the loader resolved to if the signal fired.
		let committed: unknown = 'unset';
		if (!controller.signal.aborted) committed = await pending;
		expect(controller.signal.aborted).toBe(true);
		expect(committed).toBe('unset');
	});
});

describe('NavigationGeneration: one generation governs both loads', () => {
	function deferred<T>() {
		let resolve!: (v: T) => void;
		const promise = new Promise<T>((r) => (resolve = r));
		return { promise, resolve };
	}

	it('resolving A LAST after B still commits ONLY state Bs node detail and blast radius — never a mix', async () => {
		const gate = createNavigationGate();

		const aDetail = deferred<GetNodeDetailResponse>();
		const aImpact = deferred<ImpactResponse>();
		const bDetail = deferred<GetNodeDetailResponse>();
		const bImpact = deferred<ImpactResponse>();

		const detailClient: NodeDetailClient = {
			getNodeDetail: (req) =>
				(req as { symbol: string }).symbol === 'A' ? aDetail.promise : bDetail.promise
		};
		const impactClient: ImpactClient = {
			impact: (req) => ((req as { symbol: string }).symbol === 'A' ? aImpact.promise : bImpact.promise)
		};

		let committedNode: string | undefined;
		let committedBlastNodeCount: number | undefined;

		// Mirrors what +page.svelte's own $effect does: mint ONE
		// generation per URL change, stamp it on BOTH loads, only commit a
		// resolving handler's result if its stamp is still current.
		function startNavigation(symbol: string) {
			const generation = gate.advance();
			loadBrowseTarget({ symbol, unknown: [] }, detailClient).then((state) => {
				if (!gate.isCurrent(generation)) return;
				if (state.kind === 'single-def') committedNode = state.node.name;
			});
			loadBlastRadius({ symbol, unknown: [] }, impactClient).then((state) => {
				if (!gate.isCurrent(generation)) return;
				if (state.kind === 'loaded') committedBlastNodeCount = state.nodeCount;
			});
		}

		startNavigation('A'); // generation 1 — superseded before it resolves
		startNavigation('B'); // generation 2 — the current one

		// Resolve B's two loads first, then A's — INCLUDING resolving A's
		// node-detail LAST of all four, proving the gate discards A
		// regardless of resolution order, not merely because it was asked
		// first.
		bImpact.resolve({
			symbol: 'B',
			depth: 2,
			nodeCount: 9,
			edgeCount: 4,
			affected: []
		} as unknown as ImpactResponse);
		bDetail.resolve(singleDefResponse({ node: node('B') }));
		aImpact.resolve({
			symbol: 'A',
			depth: 2,
			nodeCount: 1,
			edgeCount: 0,
			affected: []
		} as unknown as ImpactResponse);
		aDetail.resolve(singleDefResponse({ node: node('A') }));

		await Promise.all([aDetail.promise, aImpact.promise, bDetail.promise, bImpact.promise]);
		// Flush the .then() handlers scheduled above.
		await Promise.resolve();
		await Promise.resolve();
		await Promise.resolve();

		expect(committedNode).toBe('B');
		expect(committedBlastNodeCount).toBe(9);
	});
});
