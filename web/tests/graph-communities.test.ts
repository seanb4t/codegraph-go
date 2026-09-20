// Community colouring tests (GRF-06/D-10/D-11/D-12): file nodes are
// coloured by the server-computed `communityId` from a fixed, deterministic
// 12-hue palette, on the SAME ELK layered layout — nothing in this file
// computes a community; every id below is a fixture input, exactly as the
// server would send it.
//
// Task 1 (this file's describes): DOM-free palette/stylesheet shape
// assertions, and a colour == community proof over a 14-file fixture —
// mirroring web/tests/graph-cycles.test.ts's DOM-free style-sheet check and
// web/tests/file-graph-transform.test.ts's DOM-free convention.
//
// Task 2 extends this file with a route-level "N communities" toolbar-line
// describe block, mirroring web/tests/graph-cycles.test.ts's route-level
// mocking shape.
import { render, screen, waitFor } from '@testing-library/svelte';
import { describe, expect, it, vi, beforeEach } from 'vitest';

import type { FileGraphEdge, FileGraphNode, FileGraphResponse } from '$lib/gen/ui_pb';
import {
	rollupToElements,
	type FileGraphElement,
	type FileGraphNodeData
} from '$lib/components/graph/file-graph-transform';
import {
	COMMUNITY_PALETTE,
	COMMUNITY_CLASS_PREFIX,
	communityPaletteIndex
} from '$lib/components/graph/community-palette';
import { fileGraphStyle } from '$lib/components/graph/graph-style';

function node(
	path: string,
	opts: Partial<{ language: string; symbolCount: bigint; cycleId: number; communityId: number }> = {}
): FileGraphNode {
	return {
		path,
		language: opts.language ?? 'go',
		symbolCount: opts.symbolCount ?? 1n,
		cycleId: opts.cycleId ?? 0,
		communityId: opts.communityId ?? 0
	} as unknown as FileGraphNode;
}

function response(
	nodes: FileGraphNode[],
	edges: FileGraphEdge[],
	opts: Partial<{ cycleCount: number; communityCount: number }> = {}
): FileGraphResponse {
	return {
		nodes,
		edges,
		excludedPackageNodeCount: 0n,
		excludedSelfEdgeCount: 0n,
		excludedContainsEdgeCount: 0n,
		cycleCount: opts.cycleCount ?? 0,
		communityCount: opts.communityCount ?? 0
	} as unknown as FileGraphResponse;
}

type StyleEntry = { selector: string; style: Record<string, unknown> };

function isNodeData(data: FileGraphElement['data']): data is FileGraphNodeData {
	return !('source' in data);
}

describe('community palette and stylesheet (Task 1)', () => {
	it('COMMUNITY_PALETTE has exactly 12 unique, well-formed hex colours', () => {
		expect(COMMUNITY_PALETTE).toHaveLength(12);
		for (const hex of COMMUNITY_PALETTE) {
			expect(hex).toMatch(/^#[0-9a-f]{6}$/);
		}
		expect(new Set(COMMUNITY_PALETTE).size).toBe(12);
	});

	it('communityPaletteIndex is 1-based and cycles beyond 12', () => {
		expect(communityPaletteIndex(1)).toBe(0);
		expect(communityPaletteIndex(12)).toBe(11);
		expect(communityPaletteIndex(13)).toBe(0);
		expect(communityPaletteIndex(25)).toBe(0);
	});

	it('communityPaletteIndex throws on a non-integer or sub-1 id instead of silently mis-mapping (WR-01)', () => {
		// (0 - 1) % 12 === -1 in JavaScript (no floor-modulo), which would
		// otherwise produce the malformed, unmatched class "graph-community--1".
		expect(() => communityPaletteIndex(0)).toThrow(/integer >= 1/);
		expect(() => communityPaletteIndex(-1)).toThrow(/integer >= 1/);
		expect(() => communityPaletteIndex(1.5)).toThrow(/integer >= 1/);
	});

	it('the style sheet contains exactly 12 per-community rules, each setting ONLY background-color, in cascade order between the file base rule and the first cycle rule', () => {
		const entries = fileGraphStyle as StyleEntry[];
		const communityRe = /^node\[!isDirectory\]\.graph-community-(\d+)$/;

		const communityEntries = entries.filter((e) => communityRe.test(e.selector));
		expect(communityEntries).toHaveLength(12);

		const seenIndexes = new Set<number>();
		for (const entry of communityEntries) {
			const match = communityRe.exec(entry.selector);
			expect(match).not.toBeNull();
			const i = Number(match![1]);
			expect(i).toBeGreaterThanOrEqual(0);
			expect(i).toBeLessThanOrEqual(11);
			seenIndexes.add(i);
			expect(Object.keys(entry.style)).toEqual(['background-color']);
			expect(entry.style['background-color']).toBe(COMMUNITY_PALETTE[i]);
		}
		expect(seenIndexes.size).toBe(12);

		const baseFileIndex = entries.findIndex((e) => e.selector === 'node[!isDirectory]');
		const firstCommunityIndex = entries.findIndex((e) => communityRe.test(e.selector));
		const firstCycleIndex = entries.findIndex((e) => e.selector.includes('graph-cycle'));
		expect(baseFileIndex).toBeGreaterThanOrEqual(0);
		expect(firstCycleIndex).toBeGreaterThan(baseFileIndex);
		expect(firstCommunityIndex).toBeGreaterThan(baseFileIndex);
		expect(firstCommunityIndex).toBeLessThan(firstCycleIndex);

		// No compound-directory colouring rule exists (D-11): no selector
		// combines a directory existence check with the community class.
		const compoundCommunityRule = entries.some(
			(e) => e.selector.includes('?isDirectory') && e.selector.includes(COMMUNITY_CLASS_PREFIX)
		);
		expect(compoundCommunityRule).toBe(false);
	});
});

describe('colour == community (Task 1, D-12c)', () => {
	it('distinct colours equal distinct community ids over ids 1..12, shared ids share a colour, and ids beyond 12 cycle', () => {
		const files: FileGraphNode[] = [];
		for (let i = 1; i <= 14; i++) {
			files.push(node(`d/f${String(i).padStart(2, '0')}.go`, { communityId: i }));
		}
		files.push(node('d/g01.go', { communityId: 3 }));
		files.push(node('d/g02.go', { communityId: 3 }));

		const elements = rollupToElements(response(files, []), new Set(['d']));
		const entries = fileGraphStyle as StyleEntry[];

		function colourFor(id: string): string {
			const el = elements.find((e) => isNodeData(e.data) && e.data.id === id);
			if (!el) throw new Error(`no element for ${id}`);
			const classes = ('classes' in el ? el.classes : undefined) ?? '';
			const communityClass = classes
				.split(' ')
				.find((c) => c.startsWith(COMMUNITY_CLASS_PREFIX));
			if (!communityClass) throw new Error(`no community class on ${id} (classes: "${classes}")`);
			const rule = entries.find((e) => e.selector === `node[!isDirectory].${communityClass}`);
			if (!rule) throw new Error(`no style rule for ${communityClass}`);
			return rule.style['background-color'] as string;
		}

		const colours1to12 = new Set<string>();
		for (let i = 1; i <= 12; i++) {
			colours1to12.add(colourFor(`d/f${String(i).padStart(2, '0')}.go`));
		}
		expect(colours1to12.size).toBe(12);
		// eslint-disable-next-line no-console
		console.log(`distinct colours: ${colours1to12.size} over 12 distinct ids`);

		expect(colourFor('d/g01.go')).toBe(colourFor('d/f03.go'));
		expect(colourFor('d/g02.go')).toBe(colourFor('d/f03.go'));

		expect(colourFor('d/f13.go')).toBe(colourFor('d/f01.go'));
		expect(colourFor('d/f14.go')).toBe(colourFor('d/f02.go'));

		const compound = elements.find((e) => isNodeData(e.data) && e.data.id === 'd');
		expect(compound).toBeDefined();
		const compoundClasses = (compound && 'classes' in compound ? compound.classes : undefined) ?? '';
		expect(compoundClasses.includes(COMMUNITY_CLASS_PREFIX)).toBe(false);
	});
});

// --- Task 2: route-level "N communities" toolbar line ---
//
// Mirrors web/tests/graph-cycles.test.ts's route-level mocking shape (mock
// cytoscape/cytoscape-elk with a fake core; let the real +page.svelte run
// against it) so the real route and the real prop surface is what gets
// exercised.
type FakeElement = { data: Record<string, unknown>; classes?: string };
type FakeCoreOptions = { elements?: FakeElement[] };

class FakeCollection {
	constructor(public ids: string[]) {}
	get length() {
		return this.ids.length;
	}
	union(other: FakeCollection): FakeCollection {
		const merged = new Set([...this.ids, ...other.ids]);
		return new FakeCollection([...merged]);
	}
}

class FakeCore {
	private listeners = new Map<string, Array<(evt?: unknown) => void>>();
	private elements: FakeElement[];

	constructor(opts: FakeCoreOptions) {
		this.elements = opts.elements ?? [];
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
		// CR-01 fix: production code registers 'layoutstop' on THIS layout
		// instance (via `.one()` below), not on the fake's own `cy`-level
		// `this.listeners` map — track it locally per instance to match.
		const layoutstopListeners: Array<() => void> = [];
		return {
			one(event: string, handler: () => void) {
				if (event === 'layoutstop') layoutstopListeners.push(handler);
			},
			run: () => {
				queueMicrotask(() => {
					for (const h of layoutstopListeners.splice(0)) h();
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
		return {
			length: this.elements.filter((e) => 'source' in e.data).length,
			style: () => {},
			removeStyle: () => {}
		};
	}
	collection() {
		return new FakeCollection([]);
	}
	getElementById(id: string) {
		const found = this.elements.some((e) => e.data.id === id);
		return new FakeCollection(found ? [id] : []);
	}
	fit(_collection: FakeCollection) {}
	resize() {}
	destroy() {}
}

function fakeCytoscapeFactory(opts: FakeCoreOptions) {
	return new FakeCore(opts);
}
fakeCytoscapeFactory.use = () => {};

vi.mock('cytoscape', () => ({ default: fakeCytoscapeFactory }));
vi.mock('cytoscape-elk', () => ({ default: () => {} }));

let currentFileGraphImpl: () => Promise<FileGraphResponse> = () =>
	Promise.reject(new Error('graph-communities.test.ts: no fileGraph stub configured for this test'));

vi.doMock('$lib/client', () => ({
	uiClient: {
		fileGraph: () => currentFileGraphImpl()
	}
}));

const { default: GraphPage } = await import('../src/routes/graph/+page.svelte');

beforeEach(() => {
	currentFileGraphImpl = () =>
		Promise.reject(new Error('graph-communities.test.ts: no fileGraph stub configured for this test'));
});

describe('route: community count line (Task 2)', () => {
	it('a response with communityCount 3 renders the line with "3" and "communities", no interaction required', async () => {
		const nodes = [
			node('a/x.go', { communityId: 1 }),
			node('b/y.go', { communityId: 2 }),
			node('c/z.go', { communityId: 3 })
		];
		currentFileGraphImpl = () => Promise.resolve(response(nodes, [], { communityCount: 3 }));
		render(GraphPage);
		await waitFor(() => expect(screen.getByTestId('file-graph-canvas')).toBeInTheDocument());

		const summary = screen.getByTestId('graph-community-summary');
		expect(summary.textContent).toContain('3');
		expect(summary.textContent).toContain('communities');
	});

	it('a response with communityCount 1 renders the singular "1 community", not "communities"', async () => {
		currentFileGraphImpl = () =>
			Promise.resolve(response([node('a/x.go', { communityId: 1 })], [], { communityCount: 1 }));
		render(GraphPage);
		await waitFor(() => expect(screen.getByTestId('file-graph-canvas')).toBeInTheDocument());

		const summary = screen.getByTestId('graph-community-summary');
		expect(summary.textContent).toContain('1 community');
		expect(summary.textContent).not.toContain('communities');
	});

	it('a response with communityCount 0 renders "No communities computed"', async () => {
		currentFileGraphImpl = () => Promise.resolve(response([node('a/x.go')], [], { communityCount: 0 }));
		render(GraphPage);
		await waitFor(() => expect(screen.getByTestId('file-graph-canvas')).toBeInTheDocument());

		const summary = screen.getByTestId('graph-community-summary');
		expect(summary.textContent).toMatch(/no communities computed/i);
	});

	it('the count line reports the wire number, never a client recount — 5 nodes all in community 1 still shows "1 community"', async () => {
		const nodes = Array.from({ length: 5 }, (_, i) => node(`d/f${i}.go`, { communityId: 1 }));
		currentFileGraphImpl = () => Promise.resolve(response(nodes, [], { communityCount: 1 }));
		render(GraphPage);
		await waitFor(() => expect(screen.getByTestId('file-graph-canvas')).toBeInTheDocument());

		const summary = screen.getByTestId('graph-community-summary');
		expect(summary.textContent).toContain('1 community');
		expect(summary.textContent).not.toContain('communities');
	});
});
