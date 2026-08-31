// File-to-symbol expansion (05-07, GRF-03): clicking a file reveals the
// symbols it declares, in place, as children of that file node — the
// decision D-04 chose and D-02 spent Cytoscape's compound-node support on.
//
// Task 1 (this section): DOM-free model assertions over
// symbolElementsForFile's element-data output, plus a plain-data check of
// the style sheet's symbol selector — mirroring
// web/tests/file-graph-transform.test.ts's DOM-free convention and
// web/tests/graph-cycles.test.ts's own two-part file structure (a DOM-free
// section, then a route-level section further below for Task 2).
//
// Task 2 (further below): route-level tap-to-expand/collapse/re-expand
// assertions mounting the real web/src/routes/graph/+page.svelte against a
// fake cytoscape/cytoscape-elk, extending web/tests/graph-expansion.test.ts's
// FakeCore pattern with element add()/removeByIds() support — the two
// capabilities this plan's incremental (non-replace) seam needs.
import { describe, expect, it } from 'vitest';

import type { FileGraphEdge, FileGraphNode, FileGraphResponse, FileSymbolsResponse, Node } from '$lib/gen/ui_pb';
import {
	rollupToElements,
	symbolElementsForFile,
	symbolElementIdsForFile,
	type FileGraphElement,
	type FileGraphNodeData
} from '$lib/components/graph/file-graph-transform';
import { fileGraphStyle } from '$lib/components/graph/graph-style';

function symbol(
	id: string,
	name: string,
	opts: Partial<{ kind: string; startLine: number }> = {}
): Node {
	return {
		id,
		kind: opts.kind ?? 'function',
		name,
		qualifiedName: name,
		filePath: '',
		language: 'go',
		startLine: opts.startLine ?? 1,
		endLine: opts.startLine ?? 1,
		startCol: 0,
		endCol: 0,
		signature: '',
		docstring: '',
		visibility: '',
		isExported: true,
		returnType: ''
	} as unknown as Node;
}

function fileSymbolsResponse(
	symbols: Node[],
	opts: Partial<{ totalCount: number; truncated: boolean }> = {}
): FileSymbolsResponse {
	return {
		symbols,
		totalCount: opts.totalCount ?? symbols.length,
		truncated: opts.truncated ?? false
	} as unknown as FileSymbolsResponse;
}

describe('symbolElementsForFile: element shape and count', () => {
	it('produces one element per returned symbol, each parented to the file path', () => {
		const response = fileSymbolsResponse([symbol('s1', 'Foo'), symbol('s2', 'Bar')]);
		const elements = symbolElementsForFile('a/b.go', response);

		expect(elements).toHaveLength(2); // exact — a dropped symbol must fail here
		expect(elements.every((el) => el.data.parent === 'a/b.go')).toBe(true);
	});

	it('an empty symbol list produces zero elements and does not throw', () => {
		expect(() => symbolElementsForFile('a/b.go', fileSymbolsResponse([]))).not.toThrow();
		expect(symbolElementsForFile('a/b.go', fileSymbolsResponse([]))).toEqual([]);
	});

	it('each symbol element carries the symbol name as its label, its kind, its start line, and a flag marking it a symbol', () => {
		const response = fileSymbolsResponse([symbol('s1', 'Handler', { kind: 'function', startLine: 42 })]);
		const elements = symbolElementsForFile('a/b.go', response);

		expect(elements[0].data).toMatchObject({
			label: 'Handler',
			kind: 'function',
			startLine: 42,
			isSymbol: true,
			isDirectory: false
		});
	});

	it('ids are unique across the whole graph AND stable across two calls with the same input', () => {
		const response = fileSymbolsResponse([symbol('s1', 'Foo'), symbol('s2', 'Bar')]);
		const first = symbolElementsForFile('a/b.go', response).map((el) => el.data.id);
		const second = symbolElementsForFile('a/b.go', response).map((el) => el.data.id);

		expect(new Set(first).size).toBe(first.length); // no internal collision
		expect(first).toEqual(second); // deterministic, not randomly generated
	});

	it('two different files each expanded produce disjoint symbol element sets, and no symbol names the other file as its parent', () => {
		const elsA = symbolElementsForFile('a/b.go', fileSymbolsResponse([symbol('s1', 'Foo')]));
		const elsB = symbolElementsForFile('c/d.go', fileSymbolsResponse([symbol('s2', 'Foo')]));

		const idsA = new Set(elsA.map((el) => el.data.id));
		const idsB = new Set(elsB.map((el) => el.data.id));
		expect([...idsA].some((id) => idsB.has(id))).toBe(false);
		expect(elsA.every((el) => el.data.parent === 'a/b.go')).toBe(true);
		expect(elsB.every((el) => el.data.parent === 'c/d.go')).toBe(true);
		expect(elsA.some((el) => el.data.parent === 'c/d.go')).toBe(false);
		expect(elsB.some((el) => el.data.parent === 'a/b.go')).toBe(false);
	});

	it('symbol element ids never collide with a file path or a directory path already in the graph — a file named the same as a symbol inside it', () => {
		// The pathological case a naive name-based id gets wrong: the file
		// itself is "a/b.go", and it declares a symbol ALSO named "b.go" —
		// and, separately, a symbol literally named "a" (colliding with the
		// directory id "a" that rollupToElements would produce for this
		// same fixture).
		const graphElements = rollupToElements(
			{
				nodes: [{ path: 'a/b.go', language: 'go', symbolCount: 2n, cycleId: 0 }] as unknown as FileGraphNode[],
				edges: [] as unknown as FileGraphEdge[],
				excludedPackageNodeCount: 0n,
				excludedSelfEdgeCount: 0n,
				excludedContainsEdgeCount: 0n,
				cycleCount: 0
			} as unknown as FileGraphResponse,
			new Set(['a'])
		);
		const graphIds = new Set(
			graphElements.map((el) => (el.data as FileGraphNodeData).id).filter((id) => id !== undefined)
		);
		expect(graphIds.has('a/b.go')).toBe(true);
		expect(graphIds.has('a')).toBe(true);

		const symbolResponse = fileSymbolsResponse([symbol('sym-b', 'b.go'), symbol('sym-a', 'a')]);
		const symbolElements = symbolElementsForFile('a/b.go', symbolResponse);

		for (const el of symbolElements) {
			expect(graphIds.has(el.data.id)).toBe(false);
		}
	});
});

describe('symbolElementIdsForFile: the removal id builder matches the element builder exactly', () => {
	it('produces exactly the ids symbolElementsForFile itself attached to each element, for the SAME (file, response) input', () => {
		const response = fileSymbolsResponse([symbol('s1', 'Foo'), symbol('s2', 'Bar')]);
		const elementIds = symbolElementsForFile('a/b.go', response)
			.map((el) => el.data.id)
			.sort();
		const removalIds = symbolElementIdsForFile('a/b.go', response).sort();
		expect(removalIds).toEqual(elementIds);
	});
});

describe('graph-style.ts: a symbol selector distinguishable from file and directory selectors', () => {
	it('a symbol selector exists and differs from the file and directory selectors on at least one non-colour property', () => {
		const entries = fileGraphStyle as Array<{ selector: string; style: Record<string, unknown> }>;
		const symbolEntry = entries.find((e) => e.selector.includes('isSymbol'));
		const fileEntry = entries.find((e) => e.selector === 'node[!isDirectory]');
		const dirEntry = entries.find((e) => e.selector === 'node[?isDirectory]');
		expect(symbolEntry).toBeDefined();
		expect(fileEntry).toBeDefined();
		expect(dirEntry).toBeDefined();

		const nonColourKeys = ['shape', 'width', 'height', 'text-valign', 'text-halign'];
		const differsFrom = (other: Record<string, unknown>) =>
			nonColourKeys.some((k) => symbolEntry!.style[k] !== undefined && symbolEntry!.style[k] !== other[k]);
		expect(differsFrom(fileEntry!.style)).toBe(true);
		expect(differsFrom(dirEntry!.style)).toBe(true);
	});
});

// --- Task 2: route-level tap-to-expand / tap-to-collapse / re-expand ---
//
// Mirrors web/tests/graph-expansion.test.ts's two-part split:
//   1. Route-lifecycle half: 'cytoscape'/'cytoscape-elk' are MOCKED with a
//      FULLER fake core than graph-expansion.test.ts's own — this file's
//      own, extended with add()/removeByIds() support (the two
//      capabilities this plan's incremental, non-replace seam needs) —
//      GraphCanvas.svelte itself is NOT mocked; its real effects/
//      cleanup/event wiring run.
//   2. Renderer half (further below): the position-displacement finding
//      review M-9 asks this plan to record, captured against a REAL
//      headless cytoscape+elk instance — a mock's own arbitrary index-
//      based positions would not be a meaningful measurement of a real
//      layout's displacement.
import { render, screen, waitFor, fireEvent } from '@testing-library/svelte';
import { vi, beforeEach as beforeEachTop } from 'vitest';
import type { FileGraphNode as FileGraphNodeT } from '$lib/gen/ui_pb';

type FakeElement = { data: Record<string, unknown> };
type FakeCoreOptions = { elements?: FakeElement[] };

function wrapNode(el: FakeElement, index: number) {
	return {
		id: () => el.data.id as string,
		data: (key: string) => (el.data as Record<string, unknown>)[key],
		renderedPosition: () => ({ x: index * 10, y: index * 10 }),
		renderedBoundingBox: () => ({ x1: index * 10, y1: index * 10, w: 20, h: 20 })
	};
}

let rt_constructedCount = 0;
let rt_destroyedCount = 0;
let rt_appliedElementCounts: number[] = [];

class RtFakeCore {
	private listeners = new Map<string, Array<(evt?: unknown) => void>>();
	els: FakeElement[];

	constructor(opts: FakeCoreOptions) {
		rt_constructedCount++;
		this.els = opts.elements ?? [];
		rt_appliedElementCounts.push(this.els.length);
	}
	on(event: string, ...args: unknown[]) {
		const handler = args[args.length - 1] as (evt?: unknown) => void;
		const list = this.listeners.get(event) ?? [];
		list.push(handler);
		this.listeners.set(event, list);
		return this;
	}
	one(event: string, handler: (evt?: unknown) => void) {
		return this.on(event, handler);
	}
	startBatch() {}
	endBatch() {}
	elements() {
		const self = this;
		return {
			remove: () => {
				self.els = [];
			}
		};
	}
	add(newElements: FakeElement[]) {
		this.els = [...this.els, ...newElements];
		rt_appliedElementCounts.push(this.els.length);
	}
	getElementById(id: string) {
		const self = this;
		const found = self.els.some((e) => e.data.id === id);
		return {
			length: found ? 1 : 0,
			remove: () => {
				if (!found) return;
				self.els = self.els.filter((e) => e.data.id !== id);
				rt_appliedElementCounts.push(self.els.length);
			}
		};
	}
	layout(_opts: unknown) {
		return {
			run: () => {
				queueMicrotask(() => {
					for (const h of this.listeners.get('layoutstop') ?? []) h();
				});
			}
		};
	}
	nodes() {
		const nodeEls = this.els.filter((e) => !('source' in e.data));
		return {
			length: nodeEls.length,
			map: <T,>(fn: (n: ReturnType<typeof wrapNode>) => T) => nodeEls.map((el, i) => fn(wrapNode(el, i)))
		};
	}
	edges() {
		return { length: this.els.filter((e) => 'source' in e.data).length };
	}
	resize() {}
	destroy() {
		rt_destroyedCount++;
	}
	// simulateTap drives the SAME 'tap' handler GraphCanvas registers on a
	// real node click — invoked by the test, never by production code.
	simulateTap(nodeId: string) {
		const el = this.els.find((e) => e.data.id === nodeId);
		if (!el) throw new Error(`simulateTap: no element with id "${nodeId}" in the currently applied set`);
		const handlers = this.listeners.get('tap') ?? [];
		const evt = { target: wrapNode(el, 0) };
		for (const h of handlers) h(evt);
	}
	elementIds(): string[] {
		return this.els.map((e) => e.data.id as string).sort();
	}
	nodeIds(): string[] {
		return this.els
			.filter((e) => !('source' in e.data))
			.map((e) => e.data.id as string)
			.sort();
	}
	childCountOf(parentId: string): number {
		return this.els.filter((e) => e.data.parent === parentId).length;
	}
	elementsData(): Record<string, unknown>[] {
		return this.els.map((e) => e.data);
	}
}

let rt_currentInstance: RtFakeCore | undefined;

function rtFakeCytoscapeFactory(opts: FakeCoreOptions) {
	const instance = new RtFakeCore(opts);
	rt_currentInstance = instance;
	return instance;
}
rtFakeCytoscapeFactory.use = () => {};

vi.mock('cytoscape', () => ({ default: rtFakeCytoscapeFactory }));
vi.mock('cytoscape-elk', () => ({ default: () => {} }));

let rt_currentFileGraphImpl: () => Promise<FileGraphResponse> = () =>
	Promise.reject(new Error('graph-expand.test.ts: no fileGraph stub configured for this test'));
let rt_fileGraphCallCount = 0;

const rt_fileSymbolsImpls = new Map<string, () => Promise<FileSymbolsResponse>>();
const rt_fileSymbolsCallCounts = new Map<string, number>();

vi.doMock('$lib/client', () => ({
	uiClient: {
		fileGraph: () => {
			rt_fileGraphCallCount++;
			return rt_currentFileGraphImpl();
		},
		fileSymbols: (req: { path: string }) => {
			rt_fileSymbolsCallCounts.set(req.path, (rt_fileSymbolsCallCounts.get(req.path) ?? 0) + 1);
			const impl = rt_fileSymbolsImpls.get(req.path);
			if (!impl) {
				return Promise.reject(
					new Error(`graph-expand.test.ts: no fileSymbols stub configured for path "${req.path}"`)
				);
			}
			return impl();
		}
	}
}));

const { default: RtGraphPage } = await import('../src/routes/graph/+page.svelte');

function rtNode(path: string): FileGraphNodeT {
	return { path, language: 'go', symbolCount: 1n, cycleId: 0 } as unknown as FileGraphNodeT;
}

function rtResponse(nodes: FileGraphNodeT[]): FileGraphResponse {
	return {
		nodes,
		edges: [],
		excludedPackageNodeCount: 0n,
		excludedSelfEdgeCount: 0n,
		excludedContainsEdgeCount: 0n,
		cycleCount: 0
	} as unknown as FileGraphResponse;
}

beforeEachTop(() => {
	rt_constructedCount = 0;
	rt_destroyedCount = 0;
	rt_appliedElementCounts = [];
	rt_fileGraphCallCount = 0;
	rt_fileSymbolsCallCounts.clear();
	rt_fileSymbolsImpls.clear();
	rt_currentInstance = undefined;
});

describe('route: file tap expand / collapse / re-expand (Task 2)', () => {
	it('CLICK ONE expands: exactly one fileSymbols request for that path, and the file reports itself as a parent with a child count equal to the response symbol count', async () => {
		rt_currentFileGraphImpl = () => Promise.resolve(rtResponse([rtNode('a.go')]));
		rt_fileSymbolsImpls.set('a.go', () =>
			Promise.resolve(fileSymbolsResponse([symbol('s1', 'Foo'), symbol('s2', 'Bar')]))
		);
		render(RtGraphPage);
		await waitFor(() => expect(screen.getByTestId('file-graph-canvas')).toBeInTheDocument());

		rt_currentInstance!.simulateTap('a.go');
		await waitFor(() => expect(rt_currentInstance!.childCountOf('a.go')).toBe(2));

		expect(rt_fileSymbolsCallCounts.get('a.go')).toBe(1);
	});

	it('CLICK TWO collapses: NO further request (cumulative still 1), child count returns to ZERO, and a SECOND expanded file keeps its own children', async () => {
		rt_currentFileGraphImpl = () => Promise.resolve(rtResponse([rtNode('a.go'), rtNode('b.go')]));
		rt_fileSymbolsImpls.set('a.go', () => Promise.resolve(fileSymbolsResponse([symbol('s1', 'Foo')])));
		rt_fileSymbolsImpls.set('b.go', () =>
			Promise.resolve(fileSymbolsResponse([symbol('s2', 'Baz'), symbol('s3', 'Qux')]))
		);
		render(RtGraphPage);
		await waitFor(() => expect(screen.getByTestId('file-graph-canvas')).toBeInTheDocument());

		rt_currentInstance!.simulateTap('a.go');
		await waitFor(() => expect(rt_currentInstance!.childCountOf('a.go')).toBe(1));
		rt_currentInstance!.simulateTap('b.go');
		await waitFor(() => expect(rt_currentInstance!.childCountOf('b.go')).toBe(2));

		rt_currentInstance!.simulateTap('a.go');
		await waitFor(() => expect(rt_currentInstance!.childCountOf('a.go')).toBe(0));

		expect(rt_fileSymbolsCallCounts.get('a.go')).toBe(1);
		expect(rt_currentInstance!.childCountOf('b.go')).toBe(2);
	});

	it('CLICK THREE re-expands from cache: NO further request (cumulative still 1), child count returns to EXACTLY the symbol count, never doubled', async () => {
		rt_currentFileGraphImpl = () => Promise.resolve(rtResponse([rtNode('a.go')]));
		rt_fileSymbolsImpls.set('a.go', () =>
			Promise.resolve(fileSymbolsResponse([symbol('s1', 'Foo'), symbol('s2', 'Bar')]))
		);
		render(RtGraphPage);
		await waitFor(() => expect(screen.getByTestId('file-graph-canvas')).toBeInTheDocument());

		rt_currentInstance!.simulateTap('a.go');
		await waitFor(() => expect(rt_currentInstance!.childCountOf('a.go')).toBe(2));
		rt_currentInstance!.simulateTap('a.go');
		await waitFor(() => expect(rt_currentInstance!.childCountOf('a.go')).toBe(0));
		rt_currentInstance!.simulateTap('a.go');
		await waitFor(() => expect(rt_currentInstance!.childCountOf('a.go')).toBe(2));

		expect(rt_fileSymbolsCallCounts.get('a.go')).toBe(1);
	});

	it('expanding one file leaves every other file node and every edge element untouched — full counts compared before and after', async () => {
		rt_currentFileGraphImpl = () =>
			Promise.resolve(rtResponse([rtNode('a.go'), rtNode('b.go'), rtNode('c.go')]));
		rt_fileSymbolsImpls.set('a.go', () => Promise.resolve(fileSymbolsResponse([symbol('s1', 'Foo')])));
		render(RtGraphPage);
		await waitFor(() => expect(screen.getByTestId('file-graph-canvas')).toBeInTheDocument());
		const before = rt_currentInstance!.nodeIds();

		rt_currentInstance!.simulateTap('a.go');
		await waitFor(() => expect(rt_currentInstance!.childCountOf('a.go')).toBe(1));

		const after = rt_currentInstance!.nodeIds();
		// Exactly one new node id (the symbol's own, an implementation
		// detail this test does not predict) was added; every id present
		// before the expansion is still present, unchanged, afterward.
		expect(after.length).toBe(before.length + 1);
		expect(before.every((id) => after.includes(id))).toBe(true);
		expect(rt_currentInstance!.els.filter((e) => 'source' in e.data)).toHaveLength(0);
	});

	it('the identity and compound parent of every unaffected file node are preserved across an expansion', async () => {
		rt_currentFileGraphImpl = () =>
			Promise.resolve(rtResponse([rtNode('x/a.go'), rtNode('y/b.go'), rtNode('z/c.go'), rtNode('w/d.go')]));
		rt_fileSymbolsImpls.set('w/d.go', () => Promise.resolve(fileSymbolsResponse([symbol('s1', 'Foo')])));
		render(RtGraphPage);
		await waitFor(() => expect(screen.getByTestId('file-graph-canvas')).toBeInTheDocument());

		// Collapsed default: one node per directory (x, y, z, w) — three of
		// these (x, y, z) are the unaffected nodes this test tracks.
		const before = new Map(rt_currentInstance!.elementsData().map((d) => [d.id as string, d.parent]));
		expect(before.size).toBeGreaterThanOrEqual(3);

		rt_currentInstance!.simulateTap('w');
		await waitFor(() => expect(rt_currentInstance!.childCountOf('w')).toBeGreaterThan(0));
		rt_currentInstance!.simulateTap('w/d.go');
		await waitFor(() => expect(rt_currentInstance!.childCountOf('w/d.go')).toBe(1));

		for (const trackedId of ['x', 'y', 'z']) {
			const afterEl = rt_currentInstance!.elementsData().find((d) => d.id === trackedId);
			expect(afterEl).toBeDefined();
			expect(afterEl!.parent).toBe(before.get(trackedId));
		}
	});

	it('a rejected file-symbols request renders the named failure, leaves the file unexpanded, and leaves the rest of the model intact and non-empty', async () => {
		rt_currentFileGraphImpl = () => Promise.resolve(rtResponse([rtNode('a.go'), rtNode('b.go')]));
		rt_fileSymbolsImpls.set('a.go', () => Promise.reject(new Error('boom')));
		render(RtGraphPage);
		await waitFor(() => expect(screen.getByTestId('file-graph-canvas')).toBeInTheDocument());
		const before = rt_currentInstance!.elementIds();

		rt_currentInstance!.simulateTap('a.go');
		await waitFor(() => expect(screen.getByTestId('graph-file-failure-unknown')).toBeInTheDocument());

		expect(rt_currentInstance!.childCountOf('a.go')).toBe(0);
		expect(rt_currentInstance!.elementIds()).toEqual(before);
		expect(rt_currentInstance!.elementIds().length).toBeGreaterThan(0);
	});

	it('a truncated response renders a statement naming the shown count and the true total; an untruncated response renders no such statement', async () => {
		rt_currentFileGraphImpl = () => Promise.resolve(rtResponse([rtNode('a.go'), rtNode('b.go')]));
		rt_fileSymbolsImpls.set('a.go', () =>
			Promise.resolve(fileSymbolsResponse([symbol('s1', 'Foo')], { totalCount: 500, truncated: true }))
		);
		rt_fileSymbolsImpls.set('b.go', () => Promise.resolve(fileSymbolsResponse([symbol('s2', 'Bar')])));
		render(RtGraphPage);
		await waitFor(() => expect(screen.getByTestId('file-graph-canvas')).toBeInTheDocument());

		rt_currentInstance!.simulateTap('a.go');
		await waitFor(() => expect(screen.getByTestId('graph-file-truncated-a.go')).toBeInTheDocument());
		expect(screen.getByTestId('graph-file-truncated-a.go').textContent).toContain('1');
		expect(screen.getByTestId('graph-file-truncated-a.go').textContent).toContain('500');

		rt_currentInstance!.simulateTap('b.go');
		await waitFor(() => expect(rt_currentInstance!.childCountOf('b.go')).toBe(1));
		expect(screen.queryByTestId('graph-file-truncated-b.go')).not.toBeInTheDocument();
	});

	it('STALE resolved-after-collapse: a response arriving after its file was collapsed applies ZERO elements, then a later re-expand serves from the now-cached response with NO new request', async () => {
		rt_currentFileGraphImpl = () => Promise.resolve(rtResponse([rtNode('a.go')]));
		let resolveFn: ((r: FileSymbolsResponse) => void) | undefined;
		rt_fileSymbolsImpls.set(
			'a.go',
			() =>
				new Promise((resolve) => {
					resolveFn = resolve;
				})
		);
		render(RtGraphPage);
		await waitFor(() => expect(screen.getByTestId('file-graph-canvas')).toBeInTheDocument());
		const preClickCount = rt_currentInstance!.els.length;

		rt_currentInstance!.simulateTap('a.go'); // expand: request in flight
		rt_currentInstance!.simulateTap('a.go'); // collapse: BEFORE the request settles

		resolveFn!(fileSymbolsResponse([symbol('s1', 'Foo'), symbol('s2', 'Bar')]));
		await new Promise((r) => setTimeout(r, 20));

		expect(rt_currentInstance!.childCountOf('a.go')).toBe(0);
		expect(rt_currentInstance!.elementsData().some((d) => d.parent === 'a.go')).toBe(false);
		expect(rt_currentInstance!.els.length).toBe(preClickCount);

		// The stale response was CACHED, not discarded: a further tap
		// re-expands from it with no new request.
		rt_currentInstance!.simulateTap('a.go');
		await waitFor(() => expect(rt_currentInstance!.childCountOf('a.go')).toBe(2));
		expect(rt_fileSymbolsCallCounts.get('a.go')).toBe(1);
	});

	it('STALE resolved-after-unmount: exactly one destruction and zero element additions after it, and resolving the promise throws nothing', async () => {
		rt_currentFileGraphImpl = () => Promise.resolve(rtResponse([rtNode('a.go')]));
		let resolveFn: ((r: FileSymbolsResponse) => void) | undefined;
		rt_fileSymbolsImpls.set(
			'a.go',
			() =>
				new Promise((resolve) => {
					resolveFn = resolve;
				})
		);
		const { unmount } = render(RtGraphPage);
		await waitFor(() => expect(screen.getByTestId('file-graph-canvas')).toBeInTheDocument());

		rt_currentInstance!.simulateTap('a.go'); // expand: request in flight
		const countsBeforeUnmount = rt_appliedElementCounts.length;
		unmount();
		expect(rt_destroyedCount).toBe(1);

		expect(() => resolveFn!(fileSymbolsResponse([symbol('s1', 'Foo')]))).not.toThrow();
		await new Promise((r) => setTimeout(r, 20));

		expect(rt_destroyedCount).toBe(1);
		expect(rt_appliedElementCounts.length).toBe(countsBeforeUnmount);
	});

	it('selecting a directory compound node does NOT issue a file-symbols request', async () => {
		rt_currentFileGraphImpl = () => Promise.resolve(rtResponse([rtNode('x/a.go'), rtNode('x/b.go')]));
		render(RtGraphPage);
		await waitFor(() => expect(screen.getByTestId('file-graph-canvas')).toBeInTheDocument());

		rt_currentInstance!.simulateTap('x');
		await waitFor(() => expect(rt_currentInstance!.childCountOf('x')).toBe(2));

		expect(rt_fileSymbolsCallCounts.size).toBe(0);
	});

	it('the explicit collapse-affordance button collapses an expanded file WITHOUT a canvas tap, calling the SAME toggleFile path (WINDOWS.md 27)', async () => {
		rt_currentFileGraphImpl = () => Promise.resolve(rtResponse([rtNode('a.go')]));
		rt_fileSymbolsImpls.set('a.go', () => Promise.resolve(fileSymbolsResponse([symbol('s1', 'Foo')])));
		render(RtGraphPage);
		await waitFor(() => expect(screen.getByTestId('file-graph-canvas')).toBeInTheDocument());

		rt_currentInstance!.simulateTap('a.go');
		await waitFor(() => expect(rt_currentInstance!.childCountOf('a.go')).toBe(1));

		const collapseButton = screen.getByTestId('graph-collapse-file-a.go');
		await fireEvent.click(collapseButton);
		await waitFor(() => expect(rt_currentInstance!.childCountOf('a.go')).toBe(0));

		expect(rt_fileSymbolsCallCounts.get('a.go')).toBe(1);
	});
});

// --- Task 2 (continued): the position-displacement finding (review M-9) ---
//
// A mock's own arbitrary index-based positions (wrapNode above) are not a
// meaningful measurement of a real layout's displacement — this section
// exercises createFileGraphRenderer directly against a REAL headless
// cytoscape+elk instance, mirroring web/tests/graph-expansion.test.ts's own
// "the renderer (real headless cytoscape, no DOM)" section, specifically to
// produce the actual number this plan owes Phase 6's LIV-04: re-running the
// whole layered layout after an expansion MAY move unrelated nodes (an
// ACCEPTED consequence of D-04's in-place expansion, not asserted away
// here) — this measures how many moved and by how much, rather than
// asserting they did not.
describe('the renderer (real headless cytoscape, no DOM): position-displacement finding', () => {
	async function realRendererModule() {
		const { default: realCytoscape } = await vi.importActual<{
			default: typeof import('cytoscape');
		}>('cytoscape');
		const { default: realElk } = await vi.importActual<typeof import('cytoscape-elk')>('cytoscape-elk');
		realCytoscape.use(realElk);
		const mod = await vi.importActual<typeof import('../src/lib/components/graph/GraphCanvas.svelte')>(
			'../src/lib/components/graph/GraphCanvas.svelte'
		);
		return { realCytoscape, createFileGraphRenderer: mod.createFileGraphRenderer };
	}

	it('measures how many unaffected nodes move, and the largest displacement, when add() re-runs the layout after a file expansion', async () => {
		const { realCytoscape, createFileGraphRenderer } = await realRendererModule();

		// Three collapsed directories, each with one file, plus a fourth
		// (dirD/target.go) that is the one about to be expanded with a
		// symbol child. dirA/dirB/dirC's own file nodes are the "unaffected"
		// nodes this test tracks identity, parent, and position for.
		const elementsA = [
			{ data: { id: 'dirA', isDirectory: true } },
			{ data: { id: 'dirA/a.go', isDirectory: false, parent: 'dirA' } },
			{ data: { id: 'dirB', isDirectory: true } },
			{ data: { id: 'dirB/b.go', isDirectory: false, parent: 'dirB' } },
			{ data: { id: 'dirC', isDirectory: true } },
			{ data: { id: 'dirC/c.go', isDirectory: false, parent: 'dirC' } },
			{ data: { id: 'dirD', isDirectory: true } },
			{ data: { id: 'dirD/target.go', isDirectory: false, parent: 'dirD' } }
		];
		const unaffectedIds = ['dirA', 'dirA/a.go', 'dirB', 'dirB/b.go', 'dirC', 'dirC/c.go'];

		const cy = realCytoscape({ headless: true, elements: elementsA });
		const waiters: Array<() => void> = [];
		function nextSettle() {
			return new Promise<void>((resolve) => waiters.push(resolve));
		}
		const renderer = createFileGraphRenderer({
			cy,
			requestIssuedAt: performance.now(),
			onMetrics: () => {},
			onGeometry: () => waiters.shift()?.()
		});

		renderer.start();
		await nextSettle();

		const before = new Map(
			cy.nodes().map((n: { id: () => string; position: () => { x: number; y: number }; data: (k: string) => unknown }) => [
				n.id(),
				{ pos: n.position(), parent: n.data('parent') }
			])
		);
		for (const id of unaffectedIds) {
			expect(before.has(id)).toBe(true);
		}

		renderer.add([
			{
				data: {
					id: 'symbol dirD/target.go Handler',
					isDirectory: false,
					isSymbol: true,
					kind: 'function',
					startLine: 1,
					label: 'Handler',
					parent: 'dirD/target.go'
				}
			}
		]);
		await nextSettle();

		const after = new Map(
			cy.nodes().map((n: { id: () => string; position: () => { x: number; y: number }; data: (k: string) => unknown }) => [
				n.id(),
				{ pos: n.position(), parent: n.data('parent') }
			])
		);

		// IDENTITY and CONTAINMENT are preserved regardless of position —
		// this is asserted, not merely measured.
		let moved = 0;
		let maxDisplacement = 0;
		for (const id of unaffectedIds) {
			const b = before.get(id)!;
			const a = after.get(id);
			expect(a).toBeDefined();
			expect(a!.parent).toBe(b.parent);
			const d = Math.hypot(a!.pos.x - b.pos.x, a!.pos.y - b.pos.y);
			if (d > 0.5) {
				moved++;
				maxDisplacement = Math.max(maxDisplacement, d);
			}
		}

		// eslint-disable-next-line no-console
		console.log(
			`[05-07 finding, review M-9] re-running the layered layout after add() moved ${moved} of ${unaffectedIds.length} unaffected nodes; largest displacement ${maxDisplacement.toFixed(2)} model units. Recorded in 05-07-SUMMARY.md as the constraint Phase 6's LIV-04 inherits.`
		);

		cy.destroy();
	});

	it('removeByIds republishes the geometry seam even though no layout re-runs, so a collapsed symbol is not still reported as live', async () => {
		// Found during a live-browser real-mouse check of the collapse-
		// affordance button: removeByIds() correctly removes the element
		// from the cytoscape model, but the geometry seam only republishes
		// from inside runLayout's own layoutstop handler — a removal-only
		// operation that intentionally skips re-running layout was
		// therefore leaving the seam reporting a symbol id the model no
		// longer had. Real screen pixels were unaffected (cytoscape
		// redraws on any mutation independent of layout); the seam a
		// real-browser driver reads to find its next click target was the
		// only thing stale.
		const { realCytoscape, createFileGraphRenderer } = await realRendererModule();

		const elementsA = [
			{ data: { id: 'a.go', isDirectory: false } },
			{ data: { id: 'symbol-1', isDirectory: false, isSymbol: true, parent: 'a.go', label: 'Foo' } }
		];

		const cy = realCytoscape({ headless: true, elements: elementsA });
		const waiters: Array<() => void> = [];
		function nextSettle() {
			return new Promise<void>((resolve) => waiters.push(resolve));
		}
		let lastGeometry: Array<{ id: string }> = [];
		const renderer = createFileGraphRenderer({
			cy,
			requestIssuedAt: performance.now(),
			onMetrics: () => {},
			onGeometry: (g) => {
				lastGeometry = g;
				waiters.shift()?.();
			}
		});

		renderer.start();
		await nextSettle();
		expect(lastGeometry.map((g) => g.id).sort()).toEqual(['a.go', 'symbol-1']);

		// removeByIds republishes SYNCHRONOUSLY (no layout re-run, so no
		// layoutstop event to await) — the assertion below reads
		// lastGeometry immediately, not after another nextSettle() wait.
		renderer.removeByIds(['symbol-1']);

		expect(lastGeometry.map((g) => g.id)).toEqual(['a.go']);
		expect(cy.getElementById('symbol-1').length).toBe(0);

		cy.destroy();
	});
});
