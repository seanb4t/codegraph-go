// breadcrumb.ts — 09-03 Task 1 (BRW-10): the pure derivation the sticky
// breadcrumb bar renders from. Two independent concerns:
//
//   1. innermostSymbolAt — D-02's "innermost" and D-03's "never the
//      nearest preceding symbol" in one function. Given the FileSymbols
//      ranges for the open file and a 1-based line number, returns the
//      SINGLE symbol whose [startLine, endLine] most tightly contains
//      that line, or null when no range contains it — an honest empty
//      state, never a fallback to the last symbol before the gap. This
//      does not filter by symbol kind; it reports exactly what
//      FileSymbols returned, in the order it returned them.
//
//   2. firstFullyVisibleLine — the "current line" the crumb names, D-02's
//      "first fully visible source line under the sticky bar" made
//      geometric: given the code block's top edge, a uniform line
//      height, the sticky bar's bottom edge and the total line count,
//      returns the 1-based index of the first line whose top edge sits
//      AT OR BELOW the bar's bottom edge (the `>=` adjacency rule — a
//      line whose top edge exactly touches the bar's bottom counts as
//      fully visible; one pixel above it does not — the unit tests pin
//      the boundary and one step either side).

export interface SymbolRange {
	name: string;
	startLine: number;
	endLine: number;
}

/**
 * innermostSymbolAt normalises each range to `[startLine, max(startLine,
 * endLine)]` (a symbol with endLine <= startLine is treated as a
 * single-line range at startLine), keeps those containing `line`
 * inclusively, and returns the one with the smallest `(end - start)`
 * span. Ties break toward the LATER-starting range; a full tie (equal
 * span and equal start) breaks toward the LATER index in `symbols` —
 * both resolve deterministically to whatever FileSymbols' own
 * (startLine, then name) sort order placed last among equals.
 */
export function innermostSymbolAt<T extends SymbolRange>(
	symbols: readonly T[],
	line: number
): T | null {
	let best: T | null = null;
	let bestSpan = Infinity;
	let bestStart = -Infinity;

	for (const s of symbols) {
		const start = s.startLine;
		const end = Math.max(start, s.endLine);
		if (line < start || line > end) continue;

		const span = end - start;
		if (best === null || span < bestSpan || (span === bestSpan && start >= bestStart)) {
			best = s;
			bestSpan = span;
			bestStart = start;
		}
	}

	return best;
}

/**
 * firstFullyVisibleLine returns the 1-based index of the first source
 * line whose top edge is at or below `barBottom` — the "current line"
 * the sticky breadcrumb names. `lineCount === 0` returns 0 (nothing is
 * visible in an empty file); a non-positive `lineHeight` returns 1
 * rather than dividing by zero (never NaN or Infinity). The result is
 * always clamped to `[1, lineCount]`.
 */
export function firstFullyVisibleLine(
	codeTop: number,
	lineHeight: number,
	barBottom: number,
	lineCount: number
): number {
	if (lineCount <= 0) return 0;
	if (lineHeight <= 0) return 1;

	const raw = Math.ceil((barBottom - codeTop) / lineHeight) + 1;
	return Math.max(1, Math.min(lineCount, raw));
}
