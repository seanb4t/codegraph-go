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
