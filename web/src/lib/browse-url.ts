// browse-url.ts is the ONE URL grammar for the Browse view (D-13): every
// later plan in this phase that adds a query parameter extends this
// module rather than inventing a second one. Param names are deliberately
// identical to GetNodeDetailRequest's own field names (D-10) so the
// URL-to-RPC mapping is identity and cannot drift.
//
// D-12: this module parses SHAPE only. It never validates a value's
// RANGE and never bounds one into range — validateLimit/MaxLimit/
// validateFilesDepth already refuse out-of-range values server-side with
// CodeInvalidArgument, which rpc-errors.ts turns into a named UI state.
// A client-side range restriction would both duplicate a bound that can
// drift and silently turn a shared link into a different query than its
// sender saw.
//
// Unknown query parameters are preserved, never rejected (D-12) — an
// older cached shell rejecting a newer phase's link is the exact
// stale-shell failure Phase 2's `no-store` rule exists to prevent.
// Unknown parameters are represented as an ORDERED LIST OF PAIRS, not a
// plain object: an object would silently collapse a repeated parameter
// (`?a=1&a=2`) to one entry, which would round-trip a link into a
// DIFFERENT link than the one its sender saw. URLSearchParams itself is a
// multimap; this module keeps that shape.

export const BROWSE_PARAM_KEYS = ['symbol', 'file', 'line', 'depth', 'limit', 'q'] as const;

const KNOWN_KEYS: ReadonlySet<string> = new Set(BROWSE_PARAM_KEYS);

export type BrowseParams = {
	symbol?: string;
	file?: string;
	line?: number;
	depth?: number;
	limit?: number;
	q?: string;
	// Unknown params, in wire order, sorted at serialize time (never
	// reordered at parse time — parse preserves exactly what was read).
	unknown: Array<[string, string]>;
};

// parseShapeInteger accepts only a base-10 integer literal (optionally
// signed) and rejects everything else — "1.5", "abc", "" and a value
// outside Number.isSafeInteger's range are all treated as ABSENT, never
// as 0 and never as a thrown error. This is a shape check, not a range
// check (D-12): an out-of-range integer that IS a valid integer literal
// (e.g. `limit=999999`) passes through unchanged for the server to judge.
function parseShapeInteger(raw: string | null): number | undefined {
	if (raw === null || raw === '') return undefined;
	if (!/^-?\d+$/.test(raw)) return undefined;
	const value = Number(raw);
	if (!Number.isSafeInteger(value)) return undefined;
	return value;
}

export function parseBrowseParams(params: URLSearchParams): BrowseParams {
	const unknown: Array<[string, string]> = [];
	for (const [key, value] of params.entries()) {
		if (!KNOWN_KEYS.has(key)) {
			unknown.push([key, value]);
		}
	}

	return {
		symbol: params.get('symbol') ?? undefined,
		file: params.get('file') ?? undefined,
		line: parseShapeInteger(params.get('line')),
		depth: parseShapeInteger(params.get('depth')),
		limit: parseShapeInteger(params.get('limit')),
		q: params.get('q') ?? undefined,
		unknown
	};
}

// serializeBrowseParams emits known keys in BROWSE_PARAM_KEYS order, then
// unknown keys sorted lexicographically, with same-key entries kept in
// their original relative order (Array.prototype.sort is stable) — so
// the same view state always serializes to the same string, and a
// repeated unknown parameter round-trips as itself rather than
// collapsing or reordering.
export function serializeBrowseParams(p: BrowseParams): URLSearchParams {
	const out = new URLSearchParams();

	if (p.symbol !== undefined) out.append('symbol', p.symbol);
	if (p.file !== undefined) out.append('file', p.file);
	if (p.line !== undefined) out.append('line', String(p.line));
	if (p.depth !== undefined) out.append('depth', String(p.depth));
	if (p.limit !== undefined) out.append('limit', String(p.limit));
	if (p.q !== undefined) out.append('q', p.q);

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
