// DOM-free unit tests for the pure FileGraphResponse -> Cytoscape
// element-data rollup (05-03 Task 2, GRF-02; restated onto rollupToElements
// in 05-08 Task 1 — the whole-graph builder these tests exercised was
// retired when the collapsed default became GRF-01's remedy). This file
// imports the module in a plain module context: no jsdom stub, no
// cytoscape import, no DOM API anywhere below — proving the module itself
// needs none of those either.
//
// These assertions restate the ORIGINAL file/dir discrimination, lossless
// count, verbatim edge copy, cycle passthrough, dedup and empty-response
// properties against an EXPLICITLY EXPANDED set (05-08 Task 1's own
// instruction), which is where those properties still hold. The
// collapsed-default, edge-aggregation, ceiling and expansion behaviors
// this rollup ALSO owns are exercised in web/tests/graph-collapse.test.ts.
import { describe, expect, it } from 'vitest';

import type { FileGraphEdge, FileGraphNode, FileGraphResponse } from '$lib/gen/ui_pb';
import {
	rollupToElements,
	type FileGraphElement,
	type FileGraphEdgeData,
	type FileGraphNodeData
} from '$lib/components/graph/file-graph-transform';

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

function fileNodeData(elements: FileGraphElement[]): FileGraphNodeData[] {
	return elements
		.map((el) => el.data)
		.filter((d): d is FileGraphNodeData => !isEdgeData(d) && d.isDirectory === false);
}

function dirNodeData(elements: FileGraphElement[]): FileGraphNodeData[] {
	return elements
		.map((el) => el.data)
		.filter((d): d is FileGraphNodeData => !isEdgeData(d) && d.isDirectory === true);
}

function edgeData(elements: FileGraphElement[]): FileGraphEdgeData[] {
	return elements.map((el) => el.data).filter(isEdgeData);
}

describe('file-graph-transform: an expanded directory names itself as the immediate parent — no ancestor chain', () => {
	it('a nested file, with its immediate parent expanded, gets that directory as its ONLY parent — no grandparent compound node exists', () => {
		const elements = rollupToElements(
			response([node('internal/query/traverse.go')], []),
			new Set(['internal/query'])
		);

		const files = fileNodeData(elements);
		expect(files).toHaveLength(1);
		expect(files[0].id).toBe('internal/query/traverse.go');
		expect(files[0].parent).toBe('internal/query');

		const dirs = dirNodeData(elements);
		expect(dirs).toHaveLength(1);
		expect(dirs[0].id).toBe('internal/query');
		// The flat collapsed model never builds an 'internal' ancestor node
		// at all — only immediate parents of files exist as directory
		// elements.
		expect(dirs.some((d) => d.id === 'internal')).toBe(false);

		expect(elements).toHaveLength(2); // 1 file + 1 directory compound, no edges
	});
});

describe('file-graph-transform: repository-root files', () => {
	it('a root-level file has NO parent field and creates no empty-string compound node', () => {
		const elements = rollupToElements(response([node('main.go')], []), new Set());

		expect(elements).toHaveLength(1);
		const files = fileNodeData(elements);
		expect(files).toHaveLength(1);
		expect(files[0].id).toBe('main.go');
		expect('parent' in files[0]).toBe(false);

		const dirs = dirNodeData(elements);
		expect(dirs).toHaveLength(0);
		expect(elements.some((el) => (el.data as FileGraphNodeData).id === '')).toBe(false);
	});
});

describe('file-graph-transform: file vs. directory discrimination and lossless node count', () => {
	it('every compound parent carries isDirectory=true, every file carries isDirectory=false, and no file is silently dropped', () => {
		const nodes = [node('internal/a.go'), node('internal/b.go'), node('cmd/main.go')];
		const elements = rollupToElements(response(nodes, []), new Set(['internal', 'cmd']));

		const files = fileNodeData(elements);
		const dirs = dirNodeData(elements);
		expect(files).toHaveLength(nodes.length); // exact — a dropped file must fail here
		expect(files.every((f) => f.isDirectory === false)).toBe(true);
		expect(dirs.every((d) => d.isDirectory === true)).toBe(true);
	});
});

describe('file-graph-transform: edge elements', () => {
	it('a pass-through file-to-file edge (both endpoints expanded) carries source, target, kindCounts and totalCount verbatim, count matching exactly', () => {
		const nodes = [node('dir/a.go'), node('dir/b.go')];
		const edges = [edge('dir/a.go', 'dir/b.go', { kindCounts: { calls: 3n, imports: 1n }, totalCount: 4n })];
		const elements = rollupToElements(response(nodes, edges), new Set(['dir']));

		const edgeEls = edgeData(elements);
		expect(edgeEls).toHaveLength(edges.length); // exact
		expect(edgeEls[0]).toMatchObject({
			source: 'dir/a.go',
			target: 'dir/b.go',
			kindCounts: { calls: 3, imports: 1 },
			totalCount: 4,
			aggregatedFrom: 1
		});
	});
});

describe('file-graph-transform: cycle information is copied, never invented', () => {
	it('all-zero cycle ids on the wire produce all-zero cycle ids on elements', () => {
		const elements = rollupToElements(
			response([node('a.go', { cycleId: 0 }), node('b.go', { cycleId: 0 })], []),
			new Set()
		);
		const files = fileNodeData(elements);
		expect(files.every((f) => f.cycleId === 0)).toBe(true);
	});

	it('a non-zero cycle id and inCycle flag on the wire pass through unchanged (positive control)', () => {
		const elements = rollupToElements(
			response(
				[node('a.go', { cycleId: 7 }), node('b.go', { cycleId: 7 })],
				[edge('a.go', 'b.go', { inCycle: true })]
			),
			new Set()
		);
		const files = fileNodeData(elements);
		expect(files.every((f) => f.cycleId === 7)).toBe(true);
		const edgeEls = edgeData(elements);
		expect(edgeEls[0].inCycle).toBe(true);
	});
});

describe('file-graph-transform: directory deduplication', () => {
	it('many files sharing a directory prefix produce exactly ONE compound node for that directory when expanded', () => {
		const nodes = [
			node('internal/query/a.go'),
			node('internal/query/b.go'),
			node('internal/query/c.go'),
			node('internal/query/d.go')
		];
		const elements = rollupToElements(response(nodes, []), new Set(['internal/query']));

		const dirs = dirNodeData(elements);
		const queryDirs = dirs.filter((d) => d.id === 'internal/query');
		expect(queryDirs).toHaveLength(1);
		expect(fileNodeData(elements)).toHaveLength(4);
	});
});

describe('file-graph-transform: empty response', () => {
	it('zero nodes and zero edges produce zero elements and does not throw', () => {
		expect(() => rollupToElements(response([], []), new Set())).not.toThrow();
		expect(rollupToElements(response([], []), new Set())).toEqual([]);
	});
});
