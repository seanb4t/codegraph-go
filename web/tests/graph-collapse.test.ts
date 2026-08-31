// DOM-free unit tests for the collapsed-default/progressive-expansion rollup
// (05-08 Task 1, GRF-01's remedy). Written and observed RED before
// rollupToElements existed — see 05-08-SUMMARY.md for the verbatim RED
// output. No DOM, no cytoscape import in the fixtures below — the transform
// itself is pure, so its tests stay pure too.
//
// Collapsing definition under test, reverse-engineered against the locked
// artifact's own pre-recorded 134/819 figures (05-04-SUMMARY.md): the
// endpoint of a file is the file itself when it has no directory or its
// IMMEDIATE parent is expanded, otherwise its immediate parent directory —
// no ancestor chain. The collapsed view is deliberately FLAT.
import { describe, expect, it } from 'vitest';

import type { FileGraphEdge, FileGraphNode, FileGraphResponse } from '$lib/gen/ui_pb';
import {
	rollupToElements,
	plannedNodeCount,
	EXPANSION_NODE_CEILING,
	type FileGraphElement,
	type FileGraphEdgeData,
	type FileGraphNodeData
} from '$lib/components/graph/file-graph-transform';
import { fileGraphStyle } from '$lib/components/graph/graph-style';
import * as transformModule from '$lib/components/graph/file-graph-transform';

function node(
	path: string,
	opts: Partial<{ language: string; symbolCount: bigint; cycleId: number }> = {}
): FileGraphNode {
	return {
		path,
		language: opts.language ?? 'go',
		symbolCount: opts.symbolCount ?? 1n,
		cycleId: opts.cycleId ?? 0
	} as unknown as FileGraphNode;
}

function edge(
	sourceFile: string,
	targetFile: string,
	opts: Partial<{ kindCounts: Record<string, bigint>; totalCount: bigint; inCycle: boolean }> = {}
): FileGraphEdge {
	return {
		sourceFile,
		targetFile,
		kindCounts: opts.kindCounts ?? { calls: 1n },
		totalCount: opts.totalCount ?? 1n,
		inCycle: opts.inCycle ?? false
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

function isEdgeData(data: FileGraphNodeData | FileGraphEdgeData): data is FileGraphEdgeData {
	return 'source' in data;
}

function nodeEls(elements: FileGraphElement[]): FileGraphNodeData[] {
	return elements.map((el) => el.data).filter((d): d is FileGraphNodeData => !isEdgeData(d));
}

function fileEls(elements: FileGraphElement[]): FileGraphNodeData[] {
	return nodeEls(elements).filter((d) => d.isDirectory === false);
}

function dirEls(elements: FileGraphElement[]): FileGraphNodeData[] {
	return nodeEls(elements).filter((d) => d.isDirectory === true);
}

function edgeEls(elements: FileGraphElement[]): FileGraphEdgeData[] {
	return elements.map((el) => el.data).filter(isEdgeData);
}

const EMPTY = new Set<string>();

describe('rollupToElements: collapsed default', () => {
	it('a fixture spanning three distinct directories produces exactly three node elements', () => {
		const nodes = [node('a/x.go'), node('b/y.go'), node('c/z.go')];
		const elements = rollupToElements(response(nodes, []), EMPTY);
		expect(nodeEls(elements)).toHaveLength(3);
	});

	it('the same fixture produces zero file-marked elements', () => {
		const nodes = [node('a/x.go'), node('b/y.go'), node('c/z.go')];
		const elements = rollupToElements(response(nodes, []), EMPTY);
		expect(fileEls(elements)).toHaveLength(0);
	});

	it('the sum of file counts over all collapsed elements equals the fixture file count exactly', () => {
		const nodes = [node('a/x.go'), node('a/y.go'), node('b/z.go')];
		const elements = rollupToElements(response(nodes, []), EMPTY);
		const dirs = dirEls(elements);
		const total = dirs.reduce((sum, d) => sum + (d.fileCount ?? 0), 0);
		expect(total).toBe(nodes.length);
	});

	it('a repository-root file is its own node element, not marked a directory', () => {
		const elements = rollupToElements(response([node('main.go')], []), EMPTY);
		const files = fileEls(elements);
		expect(files).toHaveLength(1);
		expect(files[0].id).toBe('main.go');
	});

	it('a repository-root file creates no empty-string directory element', () => {
		const elements = rollupToElements(response([node('main.go')], []), EMPTY);
		expect(nodeEls(elements).some((d) => d.id === '')).toBe(false);
	});

	it('two files in the SAME directory contribute one collapsed element, not two', () => {
		const nodes = [node('internal/query/a.go'), node('internal/query/b.go')];
		const elements = rollupToElements(response(nodes, []), EMPTY);
		const dirs = dirEls(elements).filter((d) => d.id === 'internal/query');
		expect(dirs).toHaveLength(1);
		expect(dirs[0].fileCount).toBe(2);
	});
});

describe('rollupToElements: edge aggregation', () => {
	it('two wire edges between files in the same directory pair aggregate to one edge with SUMMED per-kind counts', () => {
		const nodes = [node('a/x1.go'), node('a/x2.go'), node('b/y1.go')];
		const edges = [
			edge('a/x1.go', 'b/y1.go', { kindCounts: { calls: 2n }, totalCount: 2n }),
			edge('a/x2.go', 'b/y1.go', { kindCounts: { calls: 3n, imports: 1n }, totalCount: 4n })
		];
		const elements = rollupToElements(response(nodes, edges), EMPTY);
		const els = edgeEls(elements);
		expect(els).toHaveLength(1);
		expect(els[0].kindCounts).toEqual({ calls: 5, imports: 1 });
	});

	it('the same aggregated edge sums totalCount and carries an aggregated-file-edge count of two', () => {
		const nodes = [node('a/x1.go'), node('a/x2.go'), node('b/y1.go')];
		const edges = [
			edge('a/x1.go', 'b/y1.go', { totalCount: 2n }),
			edge('a/x2.go', 'b/y1.go', { totalCount: 4n })
		];
		const elements = rollupToElements(response(nodes, edges), EMPTY);
		const els = edgeEls(elements);
		expect(els[0].totalCount).toBe(6);
		expect(els[0].aggregatedFrom).toBe(2);
	});

	it('a wire edge whose two files share a directory produces NO edge element in the collapsed view', () => {
		const nodes = [node('a/x1.go'), node('a/x2.go')];
		const edges = [edge('a/x1.go', 'a/x2.go')];
		const elements = rollupToElements(response(nodes, edges), EMPTY);
		expect(edgeEls(elements)).toHaveLength(0);
	});

	it('a fixture edge crossing directories in the same call DOES produce one edge element (positive control)', () => {
		const nodes = [node('a/x1.go'), node('a/x2.go'), node('b/y1.go')];
		const edges = [edge('a/x1.go', 'a/x2.go'), edge('a/x1.go', 'b/y1.go')];
		const elements = rollupToElements(response(nodes, edges), EMPTY);
		expect(edgeEls(elements)).toHaveLength(1);
	});

	it('an edge the server marked in a cycle, aggregated with one it did not, produces an edge still marked in a cycle', () => {
		const nodes = [node('a/x1.go'), node('a/x2.go'), node('b/y1.go')];
		const edges = [
			edge('a/x1.go', 'b/y1.go', { inCycle: true }),
			edge('a/x2.go', 'b/y1.go', { inCycle: false })
		];
		const elements = rollupToElements(response(nodes, edges), EMPTY);
		expect(edgeEls(elements)[0].inCycle).toBe(true);
	});

	it('the reverse fixture — neither constituent marked — produces an element not marked (negative control)', () => {
		const nodes = [node('a/x1.go'), node('a/x2.go'), node('b/y1.go')];
		const edges = [
			edge('a/x1.go', 'b/y1.go', { inCycle: false }),
			edge('a/x2.go', 'b/y1.go', { inCycle: false })
		];
		const elements = rollupToElements(response(nodes, edges), EMPTY);
		expect(edgeEls(elements)[0].inCycle).toBe(false);
	});

	it('a collapsed directory containing two files with DIFFERENT non-zero cycle ids carries both, sorted and distinct', () => {
		const nodes = [node('a/x1.go', { cycleId: 5 }), node('a/x2.go', { cycleId: 3 })];
		const elements = rollupToElements(response(nodes, []), EMPTY);
		const dir = dirEls(elements)[0];
		expect(dir.cycleIds).toEqual([3, 5]);
	});

	it('a collapsed directory whose files all carry zero cycle id carries no cycle ids (negative control)', () => {
		const nodes = [node('a/x1.go', { cycleId: 0 }), node('a/x2.go', { cycleId: 0 })];
		const elements = rollupToElements(response(nodes, []), EMPTY);
		const dir = dirEls(elements)[0];
		expect(dir.cycleIds).toBeUndefined();
	});

	it('direction is preserved: an edge from A to B does not merge with an edge from B to A', () => {
		const nodes = [node('a/x1.go'), node('b/y1.go')];
		const edges = [edge('a/x1.go', 'b/y1.go'), edge('b/y1.go', 'a/x1.go')];
		const elements = rollupToElements(response(nodes, edges), EMPTY);
		const els = edgeEls(elements);
		expect(els).toHaveLength(2);
		expect(els.some((e) => e.source === 'a' && e.target === 'b')).toBe(true);
		expect(els.some((e) => e.source === 'b' && e.target === 'a')).toBe(true);
	});
});

describe('rollupToElements: expansion', () => {
	function threeDirFixture() {
		return [
			node('x/f1.go'),
			node('x/f2.go'),
			node('y/f1.go'),
			node('z/f1.go')
		];
	}

	it('with one directory expanded, its files appear as elements naming that directory as their compound parent', () => {
		const elements = rollupToElements(response(threeDirFixture(), []), new Set(['x']));
		const files = fileEls(elements).filter((f) => f.parent === 'x');
		expect(files.map((f) => f.id).sort()).toEqual(['x/f1.go', 'x/f2.go']);
	});

	it('the expanded directory element itself is no longer marked collapsed', () => {
		const elements = rollupToElements(response(threeDirFixture(), []), new Set(['x']));
		const xDir = dirEls(elements).find((d) => d.id === 'x');
		expect(xDir).toBeDefined();
		expect(xDir!.collapsed).toBeFalsy();
	});

	it('every OTHER directory is still a single collapsed element after one expansion', () => {
		const elements = rollupToElements(response(threeDirFixture(), []), new Set(['x']));
		const yDir = dirEls(elements).find((d) => d.id === 'y');
		const zDir = dirEls(elements).find((d) => d.id === 'z');
		expect(yDir?.collapsed).toBe(true);
		expect(zDir?.collapsed).toBe(true);
	});

	it('an edge from a file in the expanded directory to a file in a collapsed one has the file and the collapsed directory as endpoints', () => {
		const nodes = [node('x/f1.go'), node('x/f2.go'), node('y/f1.go')];
		const edges = [edge('x/f1.go', 'y/f1.go')];
		const elements = rollupToElements(response(nodes, edges), new Set(['x']));
		const els = edgeEls(elements);
		expect(els).toHaveLength(1);
		expect(els[0].source).toBe('x/f1.go');
		expect(els[0].target).toBe('y');
	});

	it('an edge between two files INSIDE the expanded directory now appears as a file-level edge (excluded when collapsed)', () => {
		const nodes = [node('x/f1.go'), node('x/f2.go')];
		const edges = [edge('x/f1.go', 'x/f2.go')];
		const collapsed = edgeEls(rollupToElements(response(nodes, edges), EMPTY));
		const expanded = edgeEls(rollupToElements(response(nodes, edges), new Set(['x'])));
		expect(collapsed).toHaveLength(0);
		expect(expanded).toHaveLength(1);
		expect(expanded[0]).toMatchObject({ source: 'x/f1.go', target: 'x/f2.go' });
	});

	it('expanding a directory that contains no files changes nothing and does not throw', () => {
		const nodes = threeDirFixture();
		const before = rollupToElements(response(nodes, []), EMPTY);
		expect(() => rollupToElements(response(nodes, []), new Set(['does-not-exist']))).not.toThrow();
		const after = rollupToElements(response(nodes, []), new Set(['does-not-exist']));
		expect(after).toEqual(before);
	});

	it('expanding a path that is not a directory in the response changes nothing and does not throw', () => {
		const nodes = threeDirFixture();
		const before = rollupToElements(response(nodes, []), EMPTY);
		const after = rollupToElements(response(nodes, []), new Set(['x/f1.go']));
		expect(after).toEqual(before);
	});
});

describe('rollupToElements / plannedNodeCount: the ceiling', () => {
	it('the planned node count for a given expanded set equals the number of node elements the rollup actually returns', () => {
		const nodes = [
			node('x/f1.go'),
			node('x/f2.go'),
			node('y/f1.go'),
			node('z/f1.go'),
			node('root.go')
		];
		for (const expanded of [EMPTY, new Set(['x']), new Set(['x', 'y'])]) {
			const r = response(nodes, []);
			const planned = plannedNodeCount(r, expanded);
			const actual = nodeEls(rollupToElements(r, expanded)).length;
			expect(planned).toBe(actual);
		}
	});

	it('the exported ceiling is strictly greater than 714, the largest scale measured interactive', () => {
		expect(EXPANSION_NODE_CEILING).toBeGreaterThan(714);
	});

	it('the exported ceiling is strictly less than 3233, the scale that did not become interactive', () => {
		expect(EXPANSION_NODE_CEILING).toBeLessThan(3233);
	});
});

describe('file-graph-transform: module surface', () => {
	it('exports exactly the rollup surface as a SET, with no retired whole-graph builder', () => {
		const names = new Set(Object.keys(transformModule));
		// 05-07 (GRF-03) adds symbolElementsForFile/symbolElementIdsForFile
		// alongside the existing directory-level rollup surface — a second,
		// separate element builder for file-to-symbol expansion, not a
		// replacement for rollupToElements. dirOf is exported so +page.svelte
		// can reconcile its own expandedFiles bookkeeping against the SAME
		// immediate-parent-directory rule this module's node rollup already
		// uses, rather than a second, independently-written rule that could
		// drift out of sync with it.
		expect(names).toEqual(
			new Set([
				'rollupToElements',
				'plannedNodeCount',
				'EXPANSION_NODE_CEILING',
				'symbolElementsForFile',
				'symbolElementIdsForFile',
				'dirOf'
			])
		);
	});
});

describe('graph-style: collapsed-directory selector', () => {
	it('is distinguishable from the general directory and file selectors on a non-colour property, and appears AFTER the general directory selector', () => {
		const selectors = fileGraphStyle as Array<{ selector: string; style: Record<string, unknown> }>;
		const dirIdx = selectors.findIndex((s) => s.selector === 'node[?isDirectory]');
		const collapsedIdx = selectors.findIndex(
			(s) => s.selector.includes('collapsed') && s !== selectors[dirIdx]
		);
		const fileIdx = selectors.findIndex((s) => s.selector === 'node[!isDirectory]');
		expect(dirIdx).toBeGreaterThanOrEqual(0);
		expect(collapsedIdx).toBeGreaterThan(dirIdx);
		const collapsedStyle = selectors[collapsedIdx].style;
		const dirStyle = selectors[dirIdx].style;
		const fileStyle = selectors[fileIdx].style;
		expect(collapsedStyle['text-wrap']).toBe('wrap');
		expect(dirStyle['text-wrap']).toBeUndefined();
		expect(fileStyle['text-wrap']).toBeUndefined();
		expect(collapsedStyle.width).not.toBe(fileStyle.width);
	});
});
