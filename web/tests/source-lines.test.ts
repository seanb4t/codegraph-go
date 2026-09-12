// source-lines.test.ts — 09-03 Task 1 (BRW-10): splitHighlightedLines
// balanced per-line splitting of highlight.js markup (or escapeHtml
// plaintext). Written and run RED against a stub returning `[]` before
// the real implementation exists (RED transcript pasted in the SUMMARY),
// then made GREEN.
import { describe, it, expect } from 'vitest';
import { splitHighlightedLines } from '$lib/source-lines';

/** Counts non-overlapping occurrences of a literal substring. */
function countOccurrences(haystack: string, needle: string): number {
	let count = 0;
	let idx = haystack.indexOf(needle);
	while (idx !== -1) {
		count += 1;
		idx = haystack.indexOf(needle, idx + needle.length);
	}
	return count;
}

function isBalanced(line: string): boolean {
	return countOccurrences(line, '<span') === countOccurrences(line, '</span>');
}

describe('splitHighlightedLines', () => {
	it('returns one empty line for empty input', () => {
		expect(splitHighlightedLines('')).toEqual(['']);
	});

	it('does not add a trailing empty row for a trailing newline', () => {
		expect(splitHighlightedLines('a\nb\n')).toEqual(['a', 'b']);
	});

	it('produces the same result with or without a trailing newline', () => {
		expect(splitHighlightedLines('a\nb')).toEqual(['a', 'b']);
	});

	it('splits a multi-line span into balanced per-line entries', () => {
		const input = '<span class="hljs-comment">/* x\ny */</span>\nz';
		const result = splitHighlightedLines(input);
		expect(result).toHaveLength(3);
		expect(result[0]).toBe('<span class="hljs-comment">/* x</span>');
		expect(result[1]).toBe('<span class="hljs-comment">y */</span>');
		expect(result[2]).toBe('z');
		for (const line of result) {
			expect(isBalanced(line)).toBe(true);
		}
	});

	it('carries nested spans across a newline boundary', () => {
		const input = '<span class="a"><span class="b">p\nq</span></span>';
		const result = splitHighlightedLines(input);
		expect(result).toEqual([
			'<span class="a"><span class="b">p</span></span>',
			'<span class="a"><span class="b">q</span></span>'
		]);
	});

	it('never un-escapes text', () => {
		expect(splitHighlightedLines('&lt;script&gt;\n&amp;')).toEqual(['&lt;script&gt;', '&amp;']);
	});

	it('property: every generated input line is balanced and the count matches newlines', () => {
		const tags = ['<span class="hljs-keyword">', '<span class="hljs-string">', '<span class="hljs-comment">'];
		let seed = 42;
		function next(): number {
			// Deterministic LCG — no external randomness dependency.
			seed = (seed * 1103515245 + 12345) & 0x7fffffff;
			return seed;
		}
		for (let trial = 0; trial < 50; trial += 1) {
			const lineCount = 1 + (next() % 5);
			const parts: string[] = [];
			let openCount = 0;
			for (let i = 0; i < lineCount; i += 1) {
				let segment = `text${i}`;
				if (next() % 3 === 0) {
					segment += tags[next() % tags.length];
					openCount += 1;
				}
				if (openCount > 0 && next() % 2 === 0) {
					segment += '</span>';
					openCount -= 1;
				}
				parts.push(segment);
			}
			for (let i = 0; i < openCount; i += 1) {
				parts[parts.length - 1] += '</span>';
			}
			const input = parts.join('\n');
			const result = splitHighlightedLines(input);
			expect(result).toHaveLength(lineCount);
			for (const line of result) {
				expect(isBalanced(line)).toBe(true);
			}
		}
	});
});
