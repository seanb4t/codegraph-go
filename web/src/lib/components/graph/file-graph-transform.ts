// file-graph-transform.ts is the pure, DOM-free FileGraphResponse ->
// Cytoscape element-data rollup. Mirrors browse-url.ts's
// doc-comment-as-contract style: state up front what this module does
// NOT do.
//
// This module MUST NOT:
//   - touch the DOM. It reads and writes plain data only — no `document`,
//     no `window`, no browser global of any kind.
//   - import any cytoscape API. GraphCanvas.svelte is the ONLY file in
//     web/src permitted to import cytoscape or a cytoscape extension
//     (the renderer swap seam) — this module hands GraphCanvas plain
//     element-data objects shaped the way cytoscape's element list
//     expects, without ever calling into cytoscape itself.
//   - compute cycle membership. Cycle detection is server-side (D-06,
//     internal/query/filegraph_cycles.go) and arrives already computed on
//     the wire as FileGraphNode.cycleId / FileGraphEdge.inCycle. This
//     module copies or unions those fields onto element data unchanged;
//     deriving membership here would recreate exactly the renderer
//     coupling the swap seam exists to prevent. The `classes` string this
//     module also attaches (CYCLE_CLASS / cycleDiscriminatorClass below)
//     is presentation vocabulary copied straight FROM those same wire
//     fields — it is a second encoding of data already computed, never a
//     new computation.
//
// ONE function of (the decoded response, the set of expanded directory
// paths) produces BOTH the collapsed default and every expanded view —
// the remedy a maintainer selected after the expanded-view latency
// measurement missed its locked bar (decision `halt-collapse-default`,
// recorded in this repository's own planning history). The collapsed
// view is deliberately FLAT: the endpoint of a file is the file itself
// when it has no directory or its IMMEDIATE parent is in the expanded
// set, otherwise its immediate parent directory — NO ancestor chain is
// built. This matches the collapsed figures (134 nodes / 819 edges on
// the guava corpus) the locked threshold recorded before any
// measurement existed (corpora/graph-render-threshold.json's
// recordedNonBinding.collapsedView), reverse-engineered as: immediate
// parent directory of each file, directed distinct cross-directory
// pairs, self-pairs excluded.
//
// Directory-structural grouping is derived entirely from FilePath
// splitting: the server sends flat repository-relative paths, never a
// directory tree, so no new server concept is required — this is the
// client half of that split. Paths are guaranteed repository-relative
// and non-absolute by the server-side rollup, so no normalization or
// traversal handling is invented here.
//
// bigint -> number: the wire carries symbolCount/totalCount/kindCounts
// values as int64 (bigint in the generated TS types) because a Go int64
// has no safe round-trip through JSON as a JS number in general. In
// practice these are small counts (symbols in a file, edges between two
// files) — never within a magnitude where Number() loses precision — and
// cytoscape's own style functions (e.g. edge width by totalCount) expect
// plain numbers, not bigint. This module performs that narrowing once,
// here, so nothing downstream has to.
import type { FileGraphEdge, FileGraphNode, FileGraphResponse, FileSymbolsResponse } from '$lib/gen/ui_pb';

export type FileGraphNodeData = {
	id: string;
	label: string;
	isDirectory: boolean;
	parent?: string;
	language?: string;
	symbolCount?: number;
	cycleId?: number;
	// collapsed/fileCount/cycleIds are present ONLY on a collapsed
	// directory node — a union of server-provided per-file values (D-06),
	// never a client-side membership computation.
	collapsed?: boolean;
	fileCount?: number;
	cycleIds?: number[];
	// isSymbol/kind/startLine are present ONLY on a symbol element — a
	// leaf produced by symbolElementsForFile below, parented to the file
	// that declares it. isSymbol lets styling and collapse logic tell a
	// symbol apart from a file or a directory without parsing its id;
	// kind and startLine are copied verbatim from the wire symbol, never
	// derived.
	isSymbol?: boolean;
	kind?: string;
	startLine?: number;
};

export type FileGraphEdgeData = {
	source: string;
	target: string;
	kindCounts: Record<string, number>;
	totalCount: number;
	inCycle: boolean;
	// aggregatedFrom is the count of file-level wire edges folded into
	// this element — 1 for a pass-through file-to-file edge, more when
	// two or more file-level edges rolled up to the same directory pair.
	aggregatedFrom: number;
};

// classes is cytoscape's own element-descriptor field for a space-
// separated class list — this module's SECOND piece of element-data
// vocabulary alongside `data`, used only to copy the wire cycle fields
// onto something the style sheet can select on. CYCLE_CLASS marks any
// element the server said is a cycle member (a file node's non-zero
// cycleId, a collapsed directory's non-empty cycleIds union, or an
// edge's inCycle flag); cycleDiscriminatorClass additionally marks WHICH
// cycle a node belongs to, so two adjacent single-node cycles are told
// apart from one larger one rather than blurred into a single blob.
// Both are copies of wire fields already on the element's own `data` —
// this is presentation vocabulary derived FROM data already computed
// server-side (D-06), never a second, class-string-only source of truth.
const CYCLE_CLASS = 'graph-cycle';

function cycleDiscriminatorClass(cycleId: number): string {
	return `graph-cycle-${cycleId}`;
}

export type FileGraphElement =
	| { data: FileGraphNodeData; classes?: string }
	| { data: FileGraphEdgeData; classes?: string };

// EXPANSION_NODE_CEILING bounds the rendered node count an expansion may
// reach. Strictly between the largest scale this renderer stack was
// measured INTERACTIVE at (714 nodes, this repository's own live index)
// and the scale at which it did NOT become interactive within the
// locked seam deadline (3,233 nodes, google/guava expanded, the
// recorded FAIL). This is an implementation ceiling derived from those
// two measured scales — it is NOT a value from
// corpora/graph-render-threshold.json and does not touch that locked
// artifact.
export const EXPANSION_NODE_CEILING = 1500;

// dirOf returns the IMMEDIATE parent directory of a repository-relative
// path, or undefined for a repository-root file (no '/' in the path at
// all). This module never walks an ancestor chain — only this one level.
// Exported so +page.svelte can reconcile its own file-expansion
// bookkeeping against the SAME immediate-parent rule this module's own
// node rollup already uses, rather than a second, independently-written
// path-splitting rule that could drift out of sync with it.
export function dirOf(path: string): string | undefined {
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

// endpointOf resolves the rollup endpoint of a file path under a given
// expanded-directory set: the file itself when it has no directory or its
// immediate parent is expanded, otherwise its immediate parent directory.
function endpointOf(path: string, expandedDirs: ReadonlySet<string>): string {
	const dir = dirOf(path);
	if (dir === undefined) return path;
	if (expandedDirs.has(dir)) return path;
	return dir;
}

function fileNodeElement(
	n: FileGraphNode,
	parent: string | undefined
): { data: FileGraphNodeData; classes?: string } {
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
	if (n.cycleId !== 0) {
		return { data, classes: `${CYCLE_CLASS} ${cycleDiscriminatorClass(n.cycleId)}` };
	}
	return { data };
}

function expandedDirElement(dir: string): { data: FileGraphNodeData } {
	return {
		data: {
			id: dir,
			label: baseName(dir),
			isDirectory: true
		}
	};
}

function collapsedDirElement(
	dir: string,
	files: FileGraphNode[]
): { data: FileGraphNodeData; classes?: string } {
	const cycleIdSet = new Set<number>();
	for (const f of files) {
		if (f.cycleId !== 0) cycleIdSet.add(f.cycleId);
	}
	const cycleIds = [...cycleIdSet].sort((a, b) => a - b);
	const data: FileGraphNodeData = {
		id: dir,
		label: `${dir}\n${files.length} file${files.length === 1 ? '' : 's'}`,
		isDirectory: true,
		collapsed: true,
		fileCount: files.length
	};
	if (cycleIds.length > 0) {
		data.cycleIds = cycleIds;
		const classes = [CYCLE_CLASS, ...cycleIds.map(cycleDiscriminatorClass)].join(' ');
		return { data, classes };
	}
	return { data };
}

// buildNodeElements is the sole producer of node elements for a given
// (response, expandedDirs) pair. plannedNodeCount and rollupToElements
// BOTH call this — the ONLY way the predicate and the builder can never
// disagree is sharing this exact function, not two implementations kept
// in sync by hand.
function buildNodeElements(
	response: FileGraphResponse,
	expandedDirs: ReadonlySet<string>
): Array<{ data: FileGraphNodeData }> {
	const rootFiles: FileGraphNode[] = [];
	const filesByDir = new Map<string, FileGraphNode[]>();

	for (const n of response.nodes) {
		const dir = dirOf(n.path);
		if (dir === undefined) {
			rootFiles.push(n);
			continue;
		}
		let bucket = filesByDir.get(dir);
		if (!bucket) {
			bucket = [];
			filesByDir.set(dir, bucket);
		}
		bucket.push(n);
	}

	const elements: Array<{ data: FileGraphNodeData }> = [];

	for (const n of rootFiles) {
		elements.push(fileNodeElement(n, undefined));
	}

	for (const [dir, files] of filesByDir) {
		if (expandedDirs.has(dir)) {
			elements.push(expandedDirElement(dir));
			for (const n of files) {
				elements.push(fileNodeElement(n, dir));
			}
		} else {
			elements.push(collapsedDirElement(dir, files));
		}
	}

	return elements;
}

// buildEdgeElements is the ONE pass over the response's edges (map
// lookups only, never a nested scan — the response at the pinned corpus
// carries more than twenty thousand edges and this cost sits inside the
// interactive bar the live re-measure judges).
function buildEdgeElements(
	response: FileGraphResponse,
	expandedDirs: ReadonlySet<string>
): Array<{ data: FileGraphEdgeData; classes?: string }> {
	const byKey = new Map<string, FileGraphEdgeData>();

	for (const e of response.edges) {
		const source = endpointOf(e.sourceFile, expandedDirs);
		const target = endpointOf(e.targetFile, expandedDirs);
		if (source === target) continue;

		// A control character no repository-relative path can ever contain
		// separates source/target so two DISTINCT pairs can never collide
		// onto the same map key.
		const key = `${source}${target}`;
		let agg = byKey.get(key);
		if (!agg) {
			agg = { source, target, kindCounts: {}, totalCount: 0, inCycle: false, aggregatedFrom: 0 };
			byKey.set(key, agg);
		}
		const counts = kindCountsToNumbers(e.kindCounts);
		for (const [kind, count] of Object.entries(counts)) {
			agg.kindCounts[kind] = (agg.kindCounts[kind] ?? 0) + count;
		}
		agg.totalCount += Number(e.totalCount);
		agg.inCycle = agg.inCycle || e.inCycle;
		agg.aggregatedFrom += 1;
	}

	return [...byKey.values()].map((data) => (data.inCycle ? { data, classes: CYCLE_CLASS } : { data }));
}

// plannedNodeCount returns the number of node elements rollupToElements
// would produce for the same (response, expandedDirs) pair, without
// paying for edge aggregation — the route consults this BEFORE committing
// to an expansion, against EXPANSION_NODE_CEILING.
export function plannedNodeCount(
	response: FileGraphResponse,
	expandedDirs: ReadonlySet<string>
): number {
	return buildNodeElements(response, expandedDirs).length;
}

// rollupToElements maps a decoded FileGraphResponse and a set of expanded
// directory paths onto the complete Cytoscape element array for that
// view: the collapsed default (empty set) or any progressive expansion of
// it. This is the ONLY element builder in this module — GraphCanvas is
// handed whatever array this function returns and never computes which
// view it is showing.
export function rollupToElements(
	response: FileGraphResponse,
	expandedDirs: ReadonlySet<string>
): FileGraphElement[] {
	return [...buildNodeElements(response, expandedDirs), ...buildEdgeElements(response, expandedDirs)];
}

// symbolElementId builds a symbol element's id from the file path that
// declares it and the symbol's own wire id — deterministic (stable
// across two calls with the same input) and prefixed with a control
// character no repository-relative path can ever contain (the same
// discipline buildEdgeElements above already uses for its own composite
// key), so the result cannot collide with a file path or a directory path
// already in the graph, regardless of what the symbol's own name or wire
// id happen to be. This is exported so the route can rebuild the exact
// same ids for the removal prop when COLLAPSING a file, without a second,
// independent id-construction rule that could drift out of sync with the
// one used to CREATE the elements.
const SYMBOL_ID_SEPARATOR = '\u0000';
const SYMBOL_ID_PREFIX = `symbol${SYMBOL_ID_SEPARATOR}`;

function symbolElementId(filePath: string, symbolWireId: string): string {
	return `${SYMBOL_ID_PREFIX}${filePath}${SYMBOL_ID_SEPARATOR}${symbolWireId}`;
}

export function symbolElementIdsForFile(filePath: string, response: FileSymbolsResponse): string[] {
	return response.symbols.map((s) => symbolElementId(filePath, s.id));
}

// symbolElementsForFile maps a file path and its decoded FileSymbols
// response onto symbol element descriptors — the second element builder
// this module exports, alongside rollupToElements. Still no DOM, still no
// renderer import, still no derivation of any graph property: every
// field below is copied verbatim from the wire symbol. The file path is
// set as each element's compound PARENT at creation time — the same
// parent mechanism directory grouping already uses, never a second
// grouping concept.
export function symbolElementsForFile(
	filePath: string,
	response: FileSymbolsResponse
): Array<{ data: FileGraphNodeData }> {
	return response.symbols.map((s) => ({
		data: {
			id: symbolElementId(filePath, s.id),
			label: s.name,
			isDirectory: false,
			isSymbol: true,
			kind: s.kind,
			startLine: s.startLine,
			parent: filePath
		}
	}));
}
