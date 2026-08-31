// The collapsed default is what paints first, a directory opens and
// closes in place, and the measurement seam publishes exactly once
// (05-08 Task 2, GRF-01's remedy). Written and observed RED before
// GraphCanvas.svelte's replace/geometry seam existed — see
// 05-08-SUMMARY.md for the verbatim RED output.
//
// Two-part strategy, mirroring graph-tracer.test.ts's Pitfall-4 split:
//   1. Route-lifecycle half: 'cytoscape'/'cytoscape-elk' are MOCKED with a
//      FULLER fake core (this file's own, not graph-tracer.test.ts's
//      minimal one) that supports element replacement and per-node
//      geometry — the two capabilities this task adds. GraphCanvas.svelte
//      itself is NOT mocked; its real effects/cleanup/event wiring run.
//   2. Renderer half: createFileGraphRenderer is exercised directly
//      against a REAL headless cytoscape instance (no DOM, no Svelte
//      mount) — proving the apply-without-reconstruct, publish-once and
//      publish-every-settle properties against the real renderer.
import { render, screen, waitFor } from '@testing-library/svelte';
import { describe, expect, it, vi, beforeEach } from 'vitest';

import type { FileGraphEdge, FileGraphNode, FileGraphResponse } from '$lib/gen/ui_pb';

// --- Part 1: mocked cytoscape/cytoscape-elk for route lifecycle ---

type FakeElement = { data: Record<string, unknown> };
type FakeCoreOptions = { elements?: FakeElement[] };

let constructedCount = 0;
let destroyedCount = 0;
let appliedElementCounts: number[] = [];

function wrapNode(el: FakeElement, index: number) {
	return {
		id: () => el.data.id as string,
		data: (key: string) => (el.data as Record<string, unknown>)[key],
		renderedPosition: () => ({ x: index * 10, y: index * 10 }),
		renderedBoundingBox: () => ({ x1: index * 10, y1: index * 10, w: 20, h: 20 })
	};
}

class FakeCore {
	private listeners = new Map<string, Array<(evt?: unknown) => void>>();
	private els: FakeElement[];

	constructor(opts: FakeCoreOptions) {
		constructedCount++;
		this.els = opts.elements ?? [];
		appliedElementCounts.push(this.els.length);
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
		appliedElementCounts.push(this.els.length);
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
		destroyedCount++;
	}
	// simulateTap drives the SAME 'tap' handler GraphCanvas registers on
	// a real node click — invoked by the test, never by production code.
	simulateTap(nodeId: string) {
		const el = this.els.find((e) => e.data.id === nodeId);
		if (!el) throw new Error(`simulateTap: no element with id "${nodeId}" in the currently applied set`);
		const handlers = this.listeners.get('tap') ?? [];
		const evt = { target: wrapNode(el, 0) };
		for (const h of handlers) h(evt);
	}
	currentElementIds(): string[] {
		return this.els.map((e) => e.data.id as string).sort();
	}
}

let currentInstance: FakeCore | undefined;

function fakeCytoscapeFactory(opts: FakeCoreOptions) {
	const instance = new FakeCore(opts);
	currentInstance = instance;
	return instance;
}
fakeCytoscapeFactory.use = () => {};

vi.mock('cytoscape', () => ({ default: fakeCytoscapeFactory }));
vi.mock('cytoscape-elk', () => ({ default: () => {} }));

let currentFileGraphImpl: () => Promise<FileGraphResponse> = () =>
	Promise.reject(new Error('graph-expansion.test.ts: no fileGraph stub configured for this test'));
let fileGraphCallCount = 0;

vi.doMock('$lib/client', () => ({
	uiClient: {
		fileGraph: () => {
			fileGraphCallCount++;
			return currentFileGraphImpl();
		}
	}
}));

const { default: GraphPage } = await import('../src/routes/graph/+page.svelte');

function node(path: string, opts: Partial<{ cycleId: number }> = {}): FileGraphNode {
	return { path, language: 'go', symbolCount: 1n, cycleId: opts.cycleId ?? 0 } as unknown as FileGraphNode;
}

function edge(sourceFile: string, targetFile: string): FileGraphEdge {
	return {
		sourceFile,
		targetFile,
		kindCounts: { calls: 1n },
		totalCount: 1n,
		inCycle: false
	} as unknown as FileGraphEdge;
}

function response(nodes: FileGraphNode[], edges: FileGraphEdge[]): FileGraphResponse {
	return {
		nodes,
		edges,
		excludedPackageNodeCount: 0n,
		excludedSelfEdgeCount: 0n,
		excludedContainsEdgeCount: 0n,
		cycleCount: 0
	} as unknown as FileGraphResponse;
}

function manyFiles(dir: string, count: number): FileGraphNode[] {
	const out: FileGraphNode[] = [];
	for (let i = 0; i < count; i++) out.push(node(`${dir}/f${i}.go`));
	return out;
}

beforeEach(() => {
	constructedCount = 0;
	destroyedCount = 0;
	appliedElementCounts = [];
	fileGraphCallCount = 0;
	currentInstance = undefined;
});

describe('graph expansion: route lifecycle (mocked renderer)', () => {
	it('mounts with the collapsed default: one element per directory, zero file elements', async () => {
		currentFileGraphImpl = () =>
			Promise.resolve(response([node('x/a.go'), node('x/b.go'), node('y/c.go')], []));
		render(GraphPage);
		await waitFor(() => expect(screen.getByTestId('file-graph-canvas')).toBeInTheDocument());
		expect(constructedCount).toBe(1);
		// Two collapsed directories: x, y.
		expect(appliedElementCounts[0]).toBe(2);
	});

	it('the collapsed default at first paint has zero file-marked elements (positive control: node count non-zero)', async () => {
		currentFileGraphImpl = () =>
			Promise.resolve(response([node('x/a.go'), node('x/b.go'), node('y/c.go')], []));
		render(GraphPage);
		await waitFor(() => expect(screen.getByTestId('graph-summary')).toBeInTheDocument());
		expect(screen.getByTestId('graph-summary').textContent).toContain('2 nodes');
		expect(currentInstance!.currentElementIds()).toEqual(['x', 'y']);
	});

	it('selecting a collapsed directory applies a NEW element array with that directory expanded, WITHOUT constructing a second instance and WITHOUT destroying the first', async () => {
		currentFileGraphImpl = () => Promise.resolve(response([node('x/a.go'), node('x/b.go'), node('y/c.go')], []));
		render(GraphPage);
		await waitFor(() => expect(screen.getByTestId('file-graph-canvas')).toBeInTheDocument());

		currentInstance!.simulateTap('x');
		await waitFor(() => expect(currentInstance!.currentElementIds()).toContain('x/a.go'));

		expect(constructedCount).toBe(1);
		expect(destroyedCount).toBe(0);
		expect(currentInstance!.currentElementIds()).toEqual(['x', 'x/a.go', 'x/b.go', 'y']);
	});

	it('selecting the same directory again applies an array identical in id set to the one before the first selection', async () => {
		currentFileGraphImpl = () => Promise.resolve(response([node('x/a.go'), node('x/b.go'), node('y/c.go')], []));
		render(GraphPage);
		await waitFor(() => expect(screen.getByTestId('file-graph-canvas')).toBeInTheDocument());
		const beforeIds = currentInstance!.currentElementIds();

		currentInstance!.simulateTap('x');
		await waitFor(() => expect(currentInstance!.currentElementIds()).toContain('x/a.go'));

		currentInstance!.simulateTap('x');
		await waitFor(() => expect(currentInstance!.currentElementIds()).toEqual(beforeIds));
	});

	it('zero rpc calls are issued by either an expand or a collapse — the cumulative call count stays exactly 1', async () => {
		currentFileGraphImpl = () => Promise.resolve(response([node('x/a.go'), node('x/b.go'), node('y/c.go')], []));
		render(GraphPage);
		await waitFor(() => expect(screen.getByTestId('file-graph-canvas')).toBeInTheDocument());

		currentInstance!.simulateTap('x');
		await waitFor(() => expect(currentInstance!.currentElementIds()).toContain('x/a.go'));
		currentInstance!.simulateTap('x');
		await waitFor(() => expect(currentInstance!.currentElementIds()).not.toContain('x/a.go'));

		expect(fileGraphCallCount).toBe(1);
	});

	it('selecting a FILE node applies no new element array and issues no call', async () => {
		currentFileGraphImpl = () => Promise.resolve(response([node('x/a.go'), node('x/b.go'), node('y/c.go')], []));
		render(GraphPage);
		await waitFor(() => expect(screen.getByTestId('file-graph-canvas')).toBeInTheDocument());

		currentInstance!.simulateTap('x');
		await waitFor(() => expect(currentInstance!.currentElementIds()).toContain('x/a.go'));
		const afterExpand = currentInstance!.currentElementIds();
		const countBefore = appliedElementCounts.length;

		currentInstance!.simulateTap('x/a.go');
		await new Promise((r) => setTimeout(r, 20));

		expect(currentInstance!.currentElementIds()).toEqual(afterExpand);
		expect(appliedElementCounts.length).toBe(countBefore);
		expect(fileGraphCallCount).toBe(1);
	});

	it('a selection that would push the planned node count past the ceiling is refused: no new array is applied and a readable refusal appears', async () => {
		currentFileGraphImpl = () =>
			Promise.resolve(response([...manyFiles('big', 1600), node('y/c.go')], []));
		render(GraphPage);
		await waitFor(() => expect(screen.getByTestId('file-graph-canvas')).toBeInTheDocument());
		const before = currentInstance!.currentElementIds();

		currentInstance!.simulateTap('big');
		await waitFor(() => expect(screen.getByTestId('graph-refusal')).toBeInTheDocument());

		expect(currentInstance!.currentElementIds()).toEqual(before);
		expect(screen.queryByTestId('graph-refusal')?.textContent?.length ?? 0).toBeGreaterThan(0);
	});

	it('a selection comfortably inside the ceiling applies its expansion and renders no refusal (negative control)', async () => {
		currentFileGraphImpl = () => Promise.resolve(response([node('x/a.go'), node('x/b.go'), node('y/c.go')], []));
		render(GraphPage);
		await waitFor(() => expect(screen.getByTestId('file-graph-canvas')).toBeInTheDocument());

		currentInstance!.simulateTap('x');
		await waitFor(() => expect(currentInstance!.currentElementIds()).toContain('x/a.go'));

		expect(screen.queryByTestId('graph-refusal')).not.toBeInTheDocument();
	});

	it('the rendered summary states the current node and edge counts, and they change after an expansion', async () => {
		currentFileGraphImpl = () =>
			Promise.resolve(response([node('x/a.go'), node('x/b.go'), node('y/c.go')], [edge('x/a.go', 'x/b.go')]));
		render(GraphPage);
		await waitFor(() => expect(screen.getByTestId('graph-summary')).toBeInTheDocument());
		const before = screen.getByTestId('graph-summary').textContent ?? '';
		expect(before).toContain('2 nodes');

		currentInstance!.simulateTap('x');
		await waitFor(() => {
			const text = screen.getByTestId('graph-summary').textContent ?? '';
			// x (expanded, still a node) + x/a.go + x/b.go + y (still collapsed) = 4.
			expect(text).toContain('4 nodes');
		});
	});

	it('unmounting while the request is still pending constructs zero instances and applies nothing when the promise later resolves', async () => {
		let resolveFn: ((r: FileGraphResponse) => void) | undefined;
		currentFileGraphImpl = () =>
			new Promise((resolve) => {
				resolveFn = resolve;
			});

		const { unmount } = render(GraphPage);
		unmount();

		resolveFn!(response([node('x/a.go')], []));
		await new Promise((r) => setTimeout(r, 20));

		expect(constructedCount).toBe(0);
		expect(destroyedCount).toBe(0);
	});
});

describe('graph expansion: the renderer (real headless cytoscape, no DOM)', () => {
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

	it('applying a second element array to a live instance leaves exactly one instance, replaces its element set, and re-runs the layout', async () => {
		const { realCytoscape, createFileGraphRenderer } = await realRendererModule();

		const elementsA = [{ data: { id: 'a', isDirectory: false } }, { data: { id: 'b', isDirectory: false } }];
		const elementsB = [{ data: { id: 'c', isDirectory: false } }];

		const cy = realCytoscape({ headless: true, elements: elementsA });
		let settleCount = 0;
		const waiters: Array<() => void> = [];
		function nextSettle() {
			return new Promise<void>((resolve) => waiters.push(resolve));
		}
		const renderer = createFileGraphRenderer({
			cy,
			requestIssuedAt: performance.now(),
			onMetrics: () => {},
			onGeometry: () => {
				settleCount++;
				waiters.shift()?.();
			}
		});

		renderer.start();
		await nextSettle();
		expect(cy.nodes().map((n: { id: () => string }) => n.id()).sort()).toEqual(['a', 'b']);

		renderer.replace(elementsB);
		await nextSettle();
		expect(cy.nodes().map((n: { id: () => string }) => n.id()).sort()).toEqual(['c']);
		expect(settleCount).toBe(2);

		cy.destroy();
	});

	it('the measurement seam publishes on the FIRST layout settle and never again across a replace', async () => {
		const { realCytoscape, createFileGraphRenderer } = await realRendererModule();

		const elementsA = [{ data: { id: 'a', isDirectory: false } }];
		const elementsB = [{ data: { id: 'b', isDirectory: false } }];

		const cy = realCytoscape({ headless: true, elements: elementsA });
		let capturedMetrics: unknown;
		let metricsCallCount = 0;
		const waiters: Array<() => void> = [];
		function nextSettle() {
			return new Promise<void>((resolve) => waiters.push(resolve));
		}
		const renderer = createFileGraphRenderer({
			cy,
			requestIssuedAt: performance.now(),
			onMetrics: (m) => {
				metricsCallCount++;
				capturedMetrics = m;
			},
			onGeometry: () => waiters.shift()?.()
		});

		renderer.start();
		await nextSettle();
		const firstMetrics = capturedMetrics;
		expect(firstMetrics).toBeDefined();
		expect(metricsCallCount).toBe(1);

		renderer.replace(elementsB);
		await nextSettle();

		expect(metricsCallCount).toBe(1);
		expect(capturedMetrics).toEqual(firstMetrics);

		cy.destroy();
	});

	it('the rendered-geometry seam publishes after EVERY layout settle, one entry per node, with at least one entry marked expandable for a collapsed fixture', async () => {
		const { realCytoscape, createFileGraphRenderer } = await realRendererModule();

		const elementsA = [
			{ data: { id: 'x', isDirectory: true, collapsed: true } },
			{ data: { id: 'y/f.go', isDirectory: false } }
		];

		const cy = realCytoscape({ headless: true, elements: elementsA });
		let geometryCalls: Array<Array<{ id: string; expandable: boolean }>> = [];
		const waiters: Array<() => void> = [];
		function nextSettle() {
			return new Promise<void>((resolve) => waiters.push(resolve));
		}
		const renderer = createFileGraphRenderer({
			cy,
			requestIssuedAt: performance.now(),
			onMetrics: () => {},
			onGeometry: (g) => {
				geometryCalls.push(g);
				waiters.shift()?.();
			}
		});

		renderer.start();
		await nextSettle();
		expect(geometryCalls).toHaveLength(1);
		expect(geometryCalls[0]).toHaveLength(2);
		expect(new Set(geometryCalls[0].map((e) => e.id))).toEqual(new Set(['x', 'y/f.go']));
		expect(geometryCalls[0].some((e) => e.expandable)).toBe(true);

		renderer.replace(elementsA);
		await nextSettle();
		expect(geometryCalls).toHaveLength(2);

		cy.destroy();
	});
});
