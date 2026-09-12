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
// scratch and it never un-escapes anything. The only real unescaped-
// render directive that consumes this output lives in the pane's single
// unescaped render site (SourcePane.svelte) — no HTML is parsed,
// constructed via DOM APIs, or re-interpreted here.
//
// No regex with backtracking over the whole input; no DOM construction
// or HTML parsing of any kind — a single linear left-to-right scan per
// segment, tracking a stack of currently-open tag strings that carries
// across segment (line) boundaries.

const OPEN_TOKEN = '<span';
const CLOSE_TOKEN = '</span>';

/**
 * splitHighlightedLines splits highlight.js markup (or escaped plaintext)
 * into one string per source line, each individually tag-balanced —
 * every `<span ...>` opened before or during a line is closed at that
 * line's end, and re-opened at the start of the next line if it was
 * still open in the original markup.
 *
 * A trailing newline does not produce a trailing empty line:
 * splitHighlightedLines('a\nb\n') === ['a', 'b'], same as
 * splitHighlightedLines('a\nb').
 */
export function splitHighlightedLines(html: string): string[] {
	const segments = html.split('\n');
	if (segments.length > 1 && segments[segments.length - 1] === '') {
		segments.pop();
	}

	const lines: string[] = [];
	/** Stack of open tag strings (e.g. '<span class="hljs-comment">'), in nesting order. */
	const openStack: string[] = [];

	for (const segment of segments) {
		let out = openStack.join('');
		let i = 0;
		while (i < segment.length) {
			if (segment.startsWith(OPEN_TOKEN, i)) {
				const end = segment.indexOf('>', i);
				if (end === -1) {
					// Malformed input (should not happen with highlight.js's own
					// output); emit the remainder verbatim rather than looping.
					out += segment.slice(i);
					i = segment.length;
					break;
				}
				const tag = segment.slice(i, end + 1);
				openStack.push(tag);
				out += tag;
				i = end + 1;
				continue;
			}
			if (segment.startsWith(CLOSE_TOKEN, i)) {
				openStack.pop();
				out += CLOSE_TOKEN;
				i += CLOSE_TOKEN.length;
				continue;
			}
			out += segment[i];
			i += 1;
		}
		// Close every still-open tag at this line's end so the line is
		// independently balanced when rendered as its own DOM row. The
		// stack itself is left untouched (not popped here) — the same
		// tags carry into the next segment's opening join above.
		for (let j = 0; j < openStack.length; j += 1) {
			out += CLOSE_TOKEN;
		}
		lines.push(out);
	}

	return lines;
}
