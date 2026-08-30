// The phase's tracer, end to end (05-03 Task 3): mounting /graph issues
// exactly one FileGraph call, transforms the response, and renders it
// through GraphCanvas.svelte's GRF-05 seam. Written per Task 3's
// <behavior> block.
//
// cytoscape cannot construct against a real container under jsdom — jsdom
// implements no <canvas> 2D context, and cytoscape's canvas renderer
// throws "Could not create canvas of type 2d" at construction time
// (confirmed empirically this task, not assumed). Two separate cytoscape
// usages appear below, deliberately, mirroring Pitfall 4's honest split:
//
//   1. 'cytoscape' and 'cytoscape-elk' are MOCKED for every test that
//      mounts the real /graph route through GraphCanvas.svelte's real
//      $effect/cleanup wiring. The mock tracks construction/destroy
//      counts precisely — exactly what the two unmount-lifecycle cases
//      require — without ever touching a real canvas. GraphCanvas.svelte
//      itself is NOT mocked: its actual effect, cleanup, and event
//      wiring run for real against the fake cytoscape.
//   2. The MODEL assertions (a parent node exists and reports itself as a
//      parent; a file's parent id is that directory; counts match and are
//      greater than zero) are proven with a REAL cytoscape instance in
//      `headless: true` mode, run directly against
//      file-graph-transform.ts's output — real cytoscape, real elk
//      layout, zero DOM. This does not go through GraphCanvas.svelte (the
//      mock above stands in for it elsewhere in this file); it proves the
//      data shape the real renderer actually receives is model-correct.
import { ConnectError, Code } from '@connectrpc/connect';
import { render, screen, waitFor } from '@testing-library/svelte';
import { describe, expect, it, vi, beforeEach } from 'vitest';

import type { FileGraphEdge, FileGraphNode, FileGraphResponse } from '$lib/gen/ui_pb';
import { fileGraphToElements } from '$lib/components/graph/file-graph-transform';

// --- Part 1: mocked cytoscape/cytoscape-elk for route-mount lifecycle ---

type FakeElement = { data: Record<string, unknown> };
type FakeCoreOptions = { elements?: FakeElement[] };

let constructedCount = 0;
let destroyedCount = 0;
let appliedElementCounts: number[] = [];

class FakeCore {
	private listeners = new Map<string, Array<(evt?: unknown) => void>>();
	private elements: FakeElement[];

	constructor(opts: FakeCoreOptions) {
		constructedCount++;
		this.elements = opts.elements ?? [];
		appliedElementCounts.push(this.elements.length);
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
		return { length: this.elements.filter((e) => !('source' in e.data)).length };
	}
	edges() {
		return { length: this.elements.filter((e) => 'source' in e.data).length };
	}
	resize() {}
	destroy() {
		destroyedCount++;
	}
}

function fakeCytoscapeFactory(opts: FakeCoreOptions) {
	return new FakeCore(opts);
}
fakeCytoscapeFactory.use = () => {};

vi.mock('cytoscape', () => ({ default: fakeCytoscapeFactory }));
vi.mock('cytoscape-elk', () => ({ default: () => {} }));

// $lib/client is mocked the same way workbench-tracer.test.ts mocks it: a
// STABLE object whose fileGraph method delegates to a mutable dispatcher,
// so +page.svelte is imported exactly ONCE (module-level Svelte runtime
// state — re-importing per test was tried elsewhere in this codebase and
// confirmed broken, per that file's own comment) and each test swaps the
// implementation before calling render() again.
let currentFileGraphImpl: () => Promise<FileGraphResponse> = () =>
	Promise.reject(new Error('graph-tracer.test.ts: no fileGraph stub configured for this test'));

vi.doMock('$lib/client', () => ({
	uiClient: {
		fileGraph: () => currentFileGraphImpl()
	}
}));

const { default: GraphPage } = await import('../src/routes/graph/+page.svelte');

function node(path: string, cycleId = 0): FileGraphNode {
	return { path, language: 'go', symbolCount: 1n, cycleId } as unknown as FileGraphNode;
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

beforeEach(() => {
	constructedCount = 0;
	destroyedCount = 0;
	appliedElementCounts = [];
});

describe('graph tracer: mount issues exactly one FileGraph call and renders the seam', () => {
	it('mounts, calls FileGraph exactly once, and constructs exactly one renderer instance with the transformed elements applied', async () => {
		let calls = 0;
		currentFileGraphImpl = () => {
			calls++;
			return Promise.resolve(response([node('internal/query/traverse.go')], []));
		};

		render(GraphPage);
		await waitFor(() => expect(screen.getByTestId('file-graph-canvas')).toBeInTheDocument());

		expect(calls).toBe(1);
		expect(constructedCount).toBe(1);
		// 1 file + 2 directory compounds ('internal', 'internal/query') = 3
		expect(appliedElementCounts).toEqual([3]);
	});

	it('a rejected FileGraph call renders the named failure state and never constructs a renderer', async () => {
		currentFileGraphImpl = () => Promise.reject(new ConnectError('boom', Code.Unavailable));

		render(GraphPage);
		await waitFor(() => expect(screen.getByTestId('graph-failure-unknown')).toBeInTheDocument());

		expect(screen.queryByTestId('file-graph-canvas')).not.toBeInTheDocument();
		expect(constructedCount).toBe(0);
	});

	it('a NotFound rejection renders a DIFFERENT named failure state than an unclassified rejection (positive control)', async () => {
		currentFileGraphImpl = () => Promise.reject(new ConnectError('no graph', Code.NotFound));

		render(GraphPage);
		await waitFor(() => expect(screen.getByTestId('graph-failure-not-found')).toBeInTheDocument());
	});

	it('an empty-graph response renders a real empty state, not a blank canvas, and constructs no renderer', async () => {
		currentFileGraphImpl = () => Promise.resolve(response([], []));

		render(GraphPage);
		await waitFor(() => expect(screen.getByTestId('graph-empty')).toBeInTheDocument());

		expect(screen.queryByTestId('file-graph-canvas')).not.toBeInTheDocument();
		expect(constructedCount).toBe(0);
	});

	it('unmounting while the request is still PENDING creates ZERO renderer instances and applies no elements when the promise later resolves', async () => {
		let resolveFn: ((r: FileGraphResponse) => void) | undefined;
		currentFileGraphImpl = () =>
			new Promise((resolve) => {
				resolveFn = resolve;
			});

		const { unmount } = render(GraphPage);
		unmount();

		// The promise resolves AFTER unmount — a component that (incorrectly)
		// still applies the response would construct a renderer here.
		resolveFn!(response([node('a.go')], []));
		await new Promise((r) => setTimeout(r, 20));

		expect(constructedCount).toBe(0);
		expect(destroyedCount).toBe(0);
	});

	it('resolving first (so the canvas mounts) and THEN unmounting destroys exactly one instance', async () => {
		currentFileGraphImpl = () => Promise.resolve(response([node('a.go')], []));

		const { unmount } = render(GraphPage);
		await waitFor(() => expect(screen.getByTestId('file-graph-canvas')).toBeInTheDocument());
		expect(constructedCount).toBe(1);
		expect(destroyedCount).toBe(0);

		unmount();
		expect(destroyedCount).toBe(1);
	});

	it('publishes the measurement seam (both timers, both counts) on window after layoutstop, and dispatches the matching event', async () => {
		currentFileGraphImpl = () => Promise.resolve(response([node('a.go'), node('b.go')], [edge('a.go', 'b.go')]));

		let eventDetail: Record<string, number> | undefined;
		const handler = (evt: Event) => {
			eventDetail = (evt as CustomEvent<Record<string, number>>).detail;
		};
		window.addEventListener('codegraph:filegraph-metrics', handler);

		render(GraphPage);
		await waitFor(() => expect(eventDetail).toBeDefined());

		expect(eventDetail).toMatchObject({ nodeCount: 2, edgeCount: 1 });
		expect(typeof eventDetail!.timeToInteractiveMs).toBe('number');
		expect(typeof eventDetail!.layoutDurationMs).toBe('number');
		expect(window.__codegraphFileGraphMetrics).toEqual(eventDetail);

		window.removeEventListener('codegraph:filegraph-metrics', handler);
	});
});

describe('graph tracer: the graph MODEL (real cytoscape, headless, no DOM)', () => {
	it('a known directory reports itself as a parent, a known file resolves to that parent id, and node/edge counts match the transform output and are greater than zero', async () => {
		// This test deliberately re-imports the REAL packages, bypassing
		// the module-level vi.mock above via vi.importActual — the mock
		// exists for the route-lifecycle tests in the other describe
		// block above; this test proves the actual data shape against the
		// actual renderer, per Pitfall 4.
		const { default: realCytoscape } = await vi.importActual<{
			default: typeof import('cytoscape');
		}>('cytoscape');
		const { default: realElk } = await vi.importActual<typeof import('cytoscape-elk')>(
			'cytoscape-elk'
		);
		realCytoscape.use(realElk);

		const elements = fileGraphToElements(
			response(
				[node('internal/query/traverse.go'), node('internal/query/other.go')],
				[edge('internal/query/traverse.go', 'internal/query/other.go')]
			)
		);

		const cy = realCytoscape({
			headless: true,
			// eslint-disable-next-line @typescript-eslint/no-explicit-any
			elements: elements as any
		});

		await new Promise<void>((resolve, reject) => {
			cy.one('layoutstop', () => resolve());
			cy.layout({
				name: 'elk',
				elk: { algorithm: 'layered', 'elk.hierarchyHandling': 'INCLUDE_CHILDREN' }
				// eslint-disable-next-line @typescript-eslint/no-explicit-any
			} as any).run();
			setTimeout(() => reject(new Error('layoutstop never fired')), 5000);
		});

		const dir = cy.getElementById('internal/query');
		expect(dir.isParent()).toBe(true);

		const file = cy.getElementById('internal/query/traverse.go');
		expect(file.parent().first().id()).toBe('internal/query');

		expect(cy.nodes().length).toBe(elements.filter((el) => !('source' in el.data)).length);
		expect(cy.edges().length).toBe(elements.filter((el) => 'source' in el.data).length);
		expect(cy.nodes().length).toBeGreaterThan(0);
		expect(cy.edges().length).toBeGreaterThan(0);

		cy.destroy();
	});
});
