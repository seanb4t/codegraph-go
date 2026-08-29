// workbench-url.ts is the complete Workbench URL grammar (D-10, D-11,
// D-12) — a SIBLING to web/src/lib/browse-url.ts, not a promotion of it.
//
// ASSUMPTION-DELTA (04-01-PLAN.md's <assumption_delta_decision>): the
// Workbench needs a MULTI-valued `file=` key (D-11 — one file becomes
// many), while browse-url.ts reads its known `file` key with a
// single-value `params.get('file')` that silently drops every occurrence
// after the first. The general guidance prefers PROMOTING an existing
// grammar to the wider shape over adding a sibling; this module is a
// deliberate, recorded exception (`add-alongside`, not `promote`).
//
// Why: Phase 3's Browse grammar is frozen, round-trip-asserted
// (web/tests/browse-url.test.ts) and shipped. Widening its known-key
// reader from `get` to `getAll` would change the canonical key space of
// every Browse link that already exists in the wild — `?file=a&file=b`
// currently round-trips through Browse as `file=a` plus a silently
// dropped value, and promoting would silently change what a
// previously-shared link resolves to.
//
// Accepted debt: two URL grammars now exist that share primitives
// (isShapeInteger/parseShapeInteger, below) but not their readers. The
// two named triggers that would force a later promotion into one shared
// multi-valued core, with a migration for existing Browse links:
//   1. a THIRD view needing the same multi-valued target (e.g. Phase 5's
//      graph view selecting several files), or
//   2. Browse itself gaining multi-file selection (the file-tree-with-
//      checkboxes idea deferred to Phase 5, 04-CONTEXT.md D-15).
// Either event means three copies of "how do I read a repeated key", and
// at that point the correct move is to promote browse-url.ts's reader
// into a shared core both views import. The next author of a
// multi-valued URL key should read this comment before adding a third
// grammar.
import { isShapeInteger, parseShapeInteger } from './browse-url';

export const WORKBENCH_PARAM_KEYS = ['mode', 'symbol', 'file', 'depth', 'limit'] as const;

const KNOWN_KEYS: ReadonlySet<string> = new Set(WORKBENCH_PARAM_KEYS);

export const WORKBENCH_MODES = ['impact', 'affected', 'callers', 'callees'] as const;

export type WorkbenchMode = (typeof WORKBENCH_MODES)[number];

const KNOWN_MODES: ReadonlySet<string> = new Set(WORKBENCH_MODES);

export type WorkbenchParams = {
	mode?: WorkbenchMode;
	symbol?: string;
	// D-11: multiple files are repeated `file=` keys, read with
	// `getAll()` — NOT comma-joined, NOT a single-value `get()` (which
	// would silently keep only the first occurrence). Always an array,
	// never undefined — an absent `file` param is simply an empty array.
	files: string[];
	depth?: number;
	limit?: number;
	// Unknown params, in wire order, sorted at serialize time (mirrors
	// browse-url.ts's own ordered-multimap discipline — never a plain
	// object, which would silently collapse a repeated unknown key).
	unknown: Array<[string, string]>;
};

export function parseWorkbenchParams(params: URLSearchParams): WorkbenchParams {
	const unknown: Array<[string, string]> = [];
	for (const [key, value] of params.entries()) {
		if (!KNOWN_KEYS.has(key)) {
			unknown.push([key, value]);
		}
	}

	const rawMode = params.get('mode');
	const mode = rawMode !== null && KNOWN_MODES.has(rawMode) ? (rawMode as WorkbenchMode) : undefined;

	return {
		mode,
		symbol: params.get('symbol') ?? undefined,
		// D-11: getAll(), the deliberate divergence from browse-url.ts's
		// single-value reader for the same key name.
		files: params.getAll('file'),
		depth: parseShapeInteger(params.get('depth')),
		limit: parseShapeInteger(params.get('limit')),
		unknown
	};
}

// serializeWorkbenchParams emits known keys in WORKBENCH_PARAM_KEYS
// order — one `file=` entry per element of `files`, IN ARRAY ORDER, at
// the `file` key's position — then unknown keys sorted lexicographically
// with same-key entries kept in their original relative order
// (Array.prototype.sort is stable), mirroring
// serializeBrowseParams's stable-sort contract exactly.
export function serializeWorkbenchParams(p: WorkbenchParams): URLSearchParams {
	const out = new URLSearchParams();

	for (const key of WORKBENCH_PARAM_KEYS) {
		switch (key) {
			case 'mode':
				if (p.mode !== undefined) out.append('mode', p.mode);
				break;
			case 'symbol':
				if (p.symbol !== undefined) out.append('symbol', p.symbol);
				break;
			case 'file':
				for (const file of p.files) out.append('file', file);
				break;
			case 'depth':
				if (p.depth !== undefined) out.append('depth', String(p.depth));
				break;
			case 'limit':
				if (p.limit !== undefined) out.append('limit', String(p.limit));
				break;
		}
	}

	const sortedUnknown = [...p.unknown].sort((a, b) => {
		if (a[0] < b[0]) return -1;
		if (a[0] > b[0]) return 1;
		return 0;
	});
	for (const [key, value] of sortedUnknown) {
		out.append(key, value);
	}

	return out;
}

// Re-exported so a writer of Workbench depth/limit controls (mirroring
// NeighborsPanel.svelte's use of isShapeInteger) can validate a raw
// input against the exact shape this module's reader accepts, without a
// second import from browse-url.ts.
export { isShapeInteger };
