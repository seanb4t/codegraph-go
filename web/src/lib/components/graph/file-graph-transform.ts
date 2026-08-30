// file-graph-transform.ts is the pure, DOM-free FileGraphResponse ->
// Cytoscape element-data transform (05-03 Task 2, GRF-02). Mirrors
// browse-url.ts's doc-comment-as-contract style: state up front what this
// module does NOT do.
//
// This module MUST NOT:
//   - touch the DOM. It reads and writes plain data only — no `document`,
//     no `window`, no browser global of any kind.
//   - import any cytoscape API. GraphCanvas.svelte (05-03 Task 3) is the
//     ONLY file in web/src permitted to import cytoscape or a cytoscape
//     extension (GRF-05's seam) — this module hands GraphCanvas plain
//     element-data objects shaped the way cytoscape's element list
//     expects, without ever calling into cytoscape itself.
//   - compute cycle membership. Cycle detection is server-side (D-06,
//     internal/query/filegraph_cycles.go) and arrives already computed on
//     the wire as FileGraphNode.cycleId / FileGraphEdge.inCycle. This
//     module copies those fields onto element data unchanged; deriving
//     membership here would recreate exactly the renderer coupling
//     GRF-05's seam exists to prevent.
//
// Directory-structural grouping (GRF-02) is derived entirely from
// FilePath splitting: the server sends flat repository-relative paths
// (05-01's ENG-03 rollup), never a directory tree, so no new server
// concept is required — this is the client half of that split (see
// 05-RESEARCH.md's Architectural Responsibility Map). Paths are
// guaranteed repository-relative and non-absolute by 05-01's rollup, so
// no normalization or traversal handling is invented here.
//
// bigint -> number: the wire carries symbolCount/totalCount/kindCounts
// values as int64 (bigint in the generated TS types) because a Go int64
// has no safe round-trip through JSON as a JS number in general. In
// practice these are small counts (symbols in a file, edges between two
// files) — never within a magnitude where Number() loses precision — and
// cytoscape's own style functions (e.g. edge width by totalCount) expect
// plain numbers, not bigint. This module performs that narrowing once,
// here, so nothing downstream has to.
import type { FileGraphEdge, FileGraphNode, FileGraphResponse } from '$lib/gen/ui_pb';

export type FileGraphNodeData = {
	id: string;
	label: string;
	isDirectory: boolean;
	parent?: string;
	language?: string;
	symbolCount?: number;
	cycleId?: number;
};

export type FileGraphEdgeData = {
	source: string;
	target: string;
	kindCounts: Record<string, number>;
	totalCount: number;
	inCycle: boolean;
};

export type FileGraphElement = { data: FileGraphNodeData } | { data: FileGraphEdgeData };

// dirOf returns the directory portion of a repository-relative path, or
// undefined for a repository-root file (no '/' in the path at all).
function dirOf(path: string): string | undefined {
	const idx = path.lastIndexOf('/');
	return idx === -1 ? undefined : path.slice(0, idx);
}

function baseName(path: string): string {
	const idx = path.lastIndexOf('/');
	return idx === -1 ? path : path.slice(idx + 1);
}

function kindCountsToNumbers(counts: { [key: string]: bigint }): Record<string, number> {
	const out: Record<string, number> = {};
	for (const [kind, count] of Object.entries(counts)) {
		out[kind] = Number(count);
	}
	return out;
}

function fileNodeElement(n: FileGraphNode): { data: FileGraphNodeData } {
	const parent = dirOf(n.path);
	const data: FileGraphNodeData = {
		id: n.path,
		label: baseName(n.path),
		isDirectory: false,
		language: n.language,
		symbolCount: Number(n.symbolCount),
		cycleId: n.cycleId
	};
	if (parent !== undefined) {
		data.parent = parent;
	}
	return { data };
}

// directoryElements walks every prefix directory of path (excluding the
// file itself) and adds a compound-node element for each DISTINCT
// directory not already present in `seen`. seen/out are threaded through
// by the caller so many files sharing a directory prefix produce exactly
// one compound node for that directory (05-03 must_haves: dedup).
function directoryElements(
	path: string,
	seen: Set<string>,
	out: Array<{ data: FileGraphNodeData }>
): void {
	const segments = path.split('/');
	// Prefix directories only — the last segment is the file itself, never
	// a directory. A root-level file (segments.length === 1) has no
	// directory prefixes at all, so this loop body never runs for it.
	for (let i = 1; i < segments.length; i++) {
		const dir = segments.slice(0, i).join('/');
		if (seen.has(dir)) continue;
		seen.add(dir);
		const parent = dirOf(dir);
		const data: FileGraphNodeData = {
			id: dir,
			label: baseName(dir),
			isDirectory: true
		};
		if (parent !== undefined) {
			data.parent = parent;
		}
		out.push({ data });
	}
}

function edgeElement(e: FileGraphEdge): { data: FileGraphEdgeData } {
	return {
		data: {
			source: e.sourceFile,
			target: e.targetFile,
			kindCounts: kindCountsToNumbers(e.kindCounts),
			totalCount: Number(e.totalCount),
			inCycle: e.inCycle
		}
	};
}

// fileGraphToElements maps a decoded FileGraphResponse onto a flat array
// of Cytoscape element descriptors: one file node per response node, one
// compound directory node per distinct directory prefix, and one edge
// element per response edge. This is the whole of what it does — no
// filtering, no aggregation, no graph-theoretic computation. Every input
// node and every input edge produces exactly one output element.
export function fileGraphToElements(response: FileGraphResponse): FileGraphElement[] {
	const elements: FileGraphElement[] = [];
	const seenDirs = new Set<string>();
	const dirElements: Array<{ data: FileGraphNodeData }> = [];
	const fileElements: Array<{ data: FileGraphNodeData }> = [];

	for (const n of response.nodes) {
		directoryElements(n.path, seenDirs, dirElements);
		fileElements.push(fileNodeElement(n));
	}

	elements.push(...dirElements, ...fileElements);

	for (const e of response.edges) {
		elements.push(edgeElement(e));
	}

	return elements;
}
