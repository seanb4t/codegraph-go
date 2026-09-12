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
//      fully visible; one pixel above it does not).
//
// RED STUB (test(09-03)): returns the wrong answer on purpose so the
// target tests fail on assertions, not on module resolution. Replaced by
// the real implementation in the immediately following feat(09-03) commit.

export interface SymbolRange {
	name: string;
	startLine: number;
	endLine: number;
}

export function innermostSymbolAt<T extends SymbolRange>(
	symbols: readonly T[],
	line: number
): T | null {
	void symbols;
	void line;
	return null;
}

export function firstFullyVisibleLine(
	codeTop: number,
	lineHeight: number,
	barBottom: number,
	lineCount: number
): number {
	void codeTop;
	void lineHeight;
	void barBottom;
	void lineCount;
	return 0;
}
