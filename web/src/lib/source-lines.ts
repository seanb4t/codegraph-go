// source-lines.ts — 09-03 Task 1 (BRW-10): splits highlight.js markup (or
// escapeHtml plaintext — see web/src/lib/highlight.ts) into balanced
// per-line strings, one per source line, so SourcePane.svelte can render
// one DOM row per line instead of a single `{@html}` blob (09-RESEARCH.md
// Pitfall 3).
//
// The input to splitHighlightedLines is ALWAYS either highlight.js's own
// `.value` output (only `<span class="hljs-…">` / `</span>` tags around
// already-escaped text) or highlight.ts's own `escapeHtml(code)` fallback.
// In both cases every `<` originating from repository bytes is already
// `&lt;` — a literal `<span`/`</span>` token scan can never mistake
// source text for a tag. This function is NOT a second markup-from-
// repository-bytes site (T-09-05): it only re-arranges tags it found in
// its own input across the newline boundaries highlight.js's single
// `.value` string does not otherwise respect; it never builds a tag from
// scratch and it never un-escapes anything.
//
// RED STUB (test(09-03)): returns the wrong answer on purpose so the
// target tests fail on assertions, not on module resolution. Replaced by
// the real implementation in the immediately following feat(09-03) commit.
export function splitHighlightedLines(html: string): string[] {
	void html;
	return [];
}
