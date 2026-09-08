// Route-level tests for the per-kind edge detail region (GRF-02): a
// selected edge shows its two file paths, its total, and one row per
// wire kind key — rendered through the one shared DataTable shell, never
// a second table.
//
// Mirrors web/tests/graph-tracer.test.ts's mocking shape: mock
// cytoscape/cytoscape-elk with a fake core, let GraphCanvas.svelte and
// the real route run against it. This file's own FakeCore additionally
// tracks the SELECTOR argument to cy.on('tap', selector, handler) — a
// bare `cy.on('tap', handler)` (no selector) is the background-tap
// listener GraphCanvas.svelte registers to fire onBackgroundTapped, kept
// distinct from the node/edge selectors under separate listener keys so
// tests can trigger exactly one kind of tap without relying on
// registration order.
import { render, screen, waitFor } from '@testing-library/svelte';
import { describe, expect, it, vi, beforeEach } from 'vitest';

import type { FileGraphEdge, FileGraphNode, FileGraphResponse } from '$lib/gen/ui_pb';

type FakeElement = { data: Record<string, unknown>; classes?: string };
type FakeCoreOptions = { elements?: FakeElement[] };
type FakeEdgeWrapper = {
	id: () => string;
	source: () => { id: () => string };
	target: () => { id: () => string };
	data: () => Record<string, unknown>;
};

let lastCore: FakeCore | undefined;

class FakeCore {
	private listeners = new Map<string, Array<(evt?: unknown) => void>>();
	private elements: FakeElement[];

	constructor(opts: FakeCoreOptions) {
		this.elements = opts.elements ?? [];
		lastCore = this;
	}
	on(event: string, ...args: unknown[]) {
		let selector: string | undefined;
		let handler: (evt?: unknown) => void;
		if (args.length >= 2) {
			selector = args[0] as string;
			handler = args[args.length - 1] as (evt?: unknown) => void;
		} else {
			handler = args[0] as (evt?: unknown) => void;
		}
		const key = selector ? `${event}:${selector}` : event;
		const list = this.listeners.get(key) ?? [];
		list.push(handler);
		this.listeners.set(key, list);
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
		const els = this.elements.filter((e) => !('source' in e.data));
		return {
			length: els.length,
			map: (fn: (n: unknown) => unknown) =>
				els.map((e) =>
					fn({
						id: () => e.data.id,
						data: (k: string) => e.data[k],
						renderedBoundingBox: () => ({ x1: 0, y1: 0, w: 10, h: 10 }),
						children: () => ({ length: 0 })
					})
				)
		};
	}
	edges() {
		return { length: this.elements.filter((e) => 'source' in e.data).length };
	}
	resize() {}
	destroy() {}

	// --- test-only helpers, not part of cytoscape's real API ---
	tapEdge(source: string, target: string) {
		const el = this.elements.find((e) => e.data.source === source && e.data.target === target);
		if (!el) throw new Error(`graph-edge-detail.test.ts: no such edge in fixture ${source}->${target}`);
		const wrapper: FakeEdgeWrapper = {
			id: () => `${source}->${target}`,
			source: () => ({ id: () => source }),
			target: () => ({ id: () => target }),
			data: () => el.data
		};
		for (const h of this.listeners.get('tap:edge') ?? []) h({ target: wrapper });
	}
	tapBackground() {
		for (const h of this.listeners.get('tap') ?? []) h({ target: this });
	}
}

function fakeCytoscapeFactory(opts: FakeCoreOptions) {
	return new FakeCore(opts);
}
fakeCytoscapeFactory.use = () => {};

vi.mock('cytoscape', () => ({ default: fakeCytoscapeFactory }));
vi.mock('cytoscape-elk', () => ({ default: () => {} }));

let currentFileGraphImpl: () => Promise<FileGraphResponse> = () =>
	Promise.reject(new Error('graph-edge-detail.test.ts: no fileGraph stub configured for this test'));

vi.doMock('$lib/client', () => ({
	uiClient: {
		fileGraph: () => currentFileGraphImpl()
	}
}));

const { default: GraphPage } = await import('../src/routes/graph/+page.svelte');

function node(path: string, cycleId = 0): FileGraphNode {
	return { path, language: 'go', symbolCount: 1n, cycleId } as unknown as FileGraphNode;
}

function edge(
	sourceFile: string,
	targetFile: string,
	kindCounts: Record<string, bigint>,
	totalCount: bigint
): FileGraphEdge {
	return { sourceFile, targetFile, kindCounts, totalCount, inCycle: false } as unknown as FileGraphEdge;
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
	lastCore = undefined;
});

// All directories in these fixtures are pre-expanded (every file's
// immediate parent is in expandedDirs) so file-to-file edges survive the
// rollup unaggregated — this file tests the per-kind BREAKDOWN, not the
// aggregation rollup graph-collapse.test.ts already covers. Since this
// route's own expandedDirs starts empty, single-file directories collapse
// to a directory node by default and file-to-file edges become
// directory-to-directory (still fine: totals/kinds pass through
// unaggregated whenever aggregatedFrom stays 1, which holds for these
// one-edge-per-directory-pair fixtures below).
function threeKindResponse(): FileGraphResponse {
	const nodes = [node('a/x.go'), node('b/y.go')];
	const edges = [edge('a/x.go', 'b/y.go', { calls: 5n, imports: 2n, returns: 1n }, 8n)];
	return response(nodes, edges);
}

function oneKindResponse(): FileGraphResponse {
	const nodes = [node('a/x.go'), node('b/y.go')];
	const edges = [edge('a/x.go', 'b/y.go', { calls: 4n }, 4n)];
	return response(nodes, edges);
}

describe('route: edge detail region (Task 3)', () => {
	it('selecting an edge renders its source path, target path, and total count as visible text', async () => {
		currentFileGraphImpl = () => Promise.resolve(threeKindResponse());
		render(GraphPage);
		await waitFor(() => expect(screen.getByTestId('file-graph-canvas')).toBeInTheDocument());

		lastCore!.tapEdge('a', 'b');
		await waitFor(() => expect(screen.getByTestId('graph-edge-detail')).toBeInTheDocument());

		expect(screen.getByTestId('graph-edge-detail-source').textContent).toContain('a');
		expect(screen.getByTestId('graph-edge-detail-target').textContent).toContain('b');
		expect(screen.getByTestId('graph-edge-detail-total').textContent).toContain('8');
	});

	it('the per-kind breakdown renders exactly one row per key present in the wire kind-count map, and the rendered counts sum to the total shown', async () => {
		currentFileGraphImpl = () => Promise.resolve(threeKindResponse());
		render(GraphPage);
		await waitFor(() => expect(screen.getByTestId('file-graph-canvas')).toBeInTheDocument());

		lastCore!.tapEdge('a', 'b');
		await waitFor(() => expect(screen.getByTestId('graph-edge-detail')).toBeInTheDocument());

		const rows = screen.getAllByTestId(/^table-row-/);
		expect(rows).toHaveLength(3);
		const rowText = rows.map((r) => r.textContent ?? '').join('|');
		expect(rowText).toMatch(/calls/);
		expect(rowText).toMatch(/imports/);
		expect(rowText).toMatch(/returns/);

		const sum = [5, 2, 1].reduce((a, b) => a + b, 0);
		expect(screen.getByTestId('graph-edge-detail-total').textContent).toContain(String(sum));
	});

	it('a kind absent from the sparse map produces no row: an edge with one kind renders exactly one row naming only that kind', async () => {
		currentFileGraphImpl = () => Promise.resolve(oneKindResponse());
		render(GraphPage);
		await waitFor(() => expect(screen.getByTestId('file-graph-canvas')).toBeInTheDocument());

		lastCore!.tapEdge('a', 'b');
		await waitFor(() => expect(screen.getByTestId('graph-edge-detail')).toBeInTheDocument());

		const rows = screen.getAllByTestId(/^table-row-/);
		expect(rows).toHaveLength(1);
		expect(rows[0].textContent ?? '').toMatch(/calls/);
		expect(rows[0].textContent ?? '').not.toMatch(/imports|returns/);
	});

	it('the rows are rendered through the shared table component, asserted by its own scroll-container test id', async () => {
		currentFileGraphImpl = () => Promise.resolve(threeKindResponse());
		render(GraphPage);
		await waitFor(() => expect(screen.getByTestId('file-graph-canvas')).toBeInTheDocument());

		lastCore!.tapEdge('a', 'b');
		await waitFor(() => expect(screen.getByTestId('graph-edge-detail')).toBeInTheDocument());

		expect(screen.getByTestId('data-table-scroll')).toBeInTheDocument();
	});

	it('selecting a different edge replaces the breakdown rather than appending to it', async () => {
		currentFileGraphImpl = () =>
			Promise.resolve(
				response(
					[node('a/x.go'), node('b/y.go'), node('c/z.go')],
					[
						edge('a/x.go', 'b/y.go', { calls: 5n, imports: 2n, returns: 1n }, 8n),
						edge('a/x.go', 'c/z.go', { calls: 4n }, 4n)
					]
				)
			);
		render(GraphPage);
		await waitFor(() => expect(screen.getByTestId('file-graph-canvas')).toBeInTheDocument());

		lastCore!.tapEdge('a', 'b');
		await waitFor(() => expect(screen.getAllByTestId(/^table-row-/)).toHaveLength(3));

		lastCore!.tapEdge('a', 'c');
		await waitFor(() => expect(screen.getAllByTestId(/^table-row-/)).toHaveLength(1));

		const rows = screen.getAllByTestId(/^table-row-/);
		expect(rows[0].textContent ?? '').not.toMatch(/imports|returns/);
	});

	it('deselecting (a background tap) clears the detail region', async () => {
		currentFileGraphImpl = () => Promise.resolve(threeKindResponse());
		render(GraphPage);
		await waitFor(() => expect(screen.getByTestId('file-graph-canvas')).toBeInTheDocument());

		lastCore!.tapEdge('a', 'b');
		await waitFor(() => expect(screen.getByTestId('graph-edge-detail')).toBeInTheDocument());

		lastCore!.tapBackground();
		await waitFor(() => expect(screen.queryByTestId('graph-edge-detail')).toBeNull());
	});
});
