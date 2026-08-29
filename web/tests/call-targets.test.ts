// call-targets.test.ts — 03-08 Task 1: the click-to-definition matcher
// and DOM decorator's own tests, written and run RED before
// web/src/lib/call-targets.ts exists (RED observation recorded in the
// SUMMARY), then made GREEN.
import { describe, it, expect, vi } from 'vitest';

import {
	buildCallTargetIndex,
	lookupCallTarget,
	decorateCallTargets,
	IDENTIFIER_PATTERN,
	type CallTargetIndex
} from '$lib/call-targets';
import type { Node } from '$lib/gen/ui_pb';

function node(name: string, overrides: Partial<Node> = {}): Node {
	return {
		id: name,
		kind: 'func',
		name,
		qualifiedName: `pkg.${name}`,
		filePath: `${name}.go`,
		language: 'go',
		startLine: 1,
		endLine: 2,
		startCol: 0,
		endCol: 0,
		signature: '',
		docstring: '',
		visibility: '',
		isExported: true,
		returnType: '',
		...overrides
	} as unknown as Node;
}

describe('buildCallTargetIndex: ambiguous names are known to be ambiguous', () => {
	it('two nodes sharing a name yield ONE entry holding both', () => {
		const a = node('Foo', { filePath: 'a.go' });
		const b = node('Foo', { filePath: 'b.go' });
		const index = buildCallTargetIndex([a, b]);
		expect(index.size).toBe(1);
		const entry = lookupCallTarget(index, 'Foo');
		expect(entry).toHaveLength(2);
		expect(entry?.map((n) => n.filePath)).toEqual(['a.go', 'b.go']);
	});
});

describe('lookupCallTarget: exact match only, in both directions', () => {
	it('hits for an identifier exactly equal to a call target name', () => {
		const index = buildCallTargetIndex([node('Foo')]);
		expect(lookupCallTarget(index, 'Foo')).toHaveLength(1);
	});

	it('a differently-cased spelling misses, paired with the exact spelling hitting', () => {
		const index = buildCallTargetIndex([node('Foo')]);
		expect(lookupCallTarget(index, 'foo')).toBeUndefined();
		expect(lookupCallTarget(index, 'FOO')).toBeUndefined();
		expect(lookupCallTarget(index, 'Foo')).toHaveLength(1);
	});

	it('a Unicode-normalization variant misses, paired with the exact NFC spelling hitting', () => {
		// "café" as a precomposed NFC string (single é code point, U+00E9)
		// vs. its NFD decomposition (plain e, U+0065, followed by a
		// combining acute accent, U+0301) — byte-different, visually
		// identical. No normalization is applied on either side.
		const nfc = 'caf\u00e9';
		const nfd = 'cafe\u0301';
		expect(nfc.normalize('NFC')).toBe(nfc);
		expect(nfc).not.toBe(nfd);
		const index = buildCallTargetIndex([node(nfc)]);
		expect(lookupCallTarget(index, nfd)).toBeUndefined();
		expect(lookupCallTarget(index, nfc)).toHaveLength(1);
	});

	it('a strict prefix and a strict superstring both miss, paired with the exact name hitting', () => {
		const index = buildCallTargetIndex([node('Foo')]);
		expect(lookupCallTarget(index, 'Fo')).toBeUndefined();
		expect(lookupCallTarget(index, 'FooBar')).toBeUndefined();
		expect(lookupCallTarget(index, 'Foo')).toHaveLength(1);
	});

	it('an EMPTY index returns no hit for any input, and does not throw', () => {
		const index = buildCallTargetIndex([]);
		expect(() => lookupCallTarget(index, 'Foo')).not.toThrow();
		expect(lookupCallTarget(index, 'Foo')).toBeUndefined();
		expect(lookupCallTarget(index, '')).toBeUndefined();
	});
});

describe('IDENTIFIER_PATTERN: the tokenizer', () => {
	it('splits short locals and keywords out as ordinary tokens', () => {
		const tokens = [...'err := ctx.Value(i)'.matchAll(IDENTIFIER_PATTERN)].map((m) => m[0]);
		expect(tokens).toEqual(['err', 'ctx', 'Value', 'i']);
	});

	it('treats a non-ASCII identifier as ONE token, not split at its first non-ASCII code point', () => {
		const tokens = [...'résumé + café'.matchAll(IDENTIFIER_PATTERN)].map((m) => m[0]);
		expect(tokens).toEqual(['résumé', 'café']);
	});
});

describe('decorateCallTargets: matching tokens become activatable elements; everything else is untouched', () => {
	it('converts exactly the matching identifier tokens, leaves other text byte-identical, and invokes the callback with the matched entry', () => {
		const root = document.createElement('code');
		root.textContent = 'call Foo and Bar and baz';
		const fooNode = node('Foo');
		const index = buildCallTargetIndex([fooNode]);
		const onSelect = vi.fn();

		decorateCallTargets(root, index, onSelect);

		const targets = root.querySelectorAll('[data-call-target]');
		expect(targets).toHaveLength(1);
		expect(targets[0]?.textContent).toBe('Foo');
		expect(root.textContent).toBe('call Foo and Bar and baz');

		(targets[0] as HTMLElement).click();
		expect(onSelect).toHaveBeenCalledTimes(1);
		expect(onSelect).toHaveBeenCalledWith(index.get('Foo'));
	});

	it('an identifier with no matching call target is inert — no element, no callback', () => {
		const root = document.createElement('code');
		root.textContent = 'local err is unrelated';
		const index = buildCallTargetIndex([node('Foo')]);
		const onSelect = vi.fn();

		decorateCallTargets(root, index, onSelect);

		expect(root.querySelectorAll('[data-call-target]')).toHaveLength(0);
		expect(root.textContent).toBe('local err is unrelated');
	});
});

describe('decorateCallTargets: IDEMPOTENCE', () => {
	it('running twice over the same subtree yields a DOM byte-identical to running once, and activation fires exactly once', () => {
		const root = document.createElement('code');
		root.textContent = 'Foo calls Foo again';
		const index = buildCallTargetIndex([node('Foo')]);
		const onSelect = vi.fn();

		decorateCallTargets(root, index, onSelect);
		const afterFirst = root.innerHTML;

		decorateCallTargets(root, index, onSelect);
		const afterSecond = root.innerHTML;

		expect(afterSecond).toBe(afterFirst);
		expect(root.querySelectorAll('[data-call-target]')).toHaveLength(2);
		// No nested wrapper: each decorated element's own text is exactly
		// the token, never containing a second decorated element.
		for (const el of Array.from(root.querySelectorAll('[data-call-target]'))) {
			expect(el.querySelectorAll('[data-call-target]')).toHaveLength(0);
		}

		const first = root.querySelectorAll('[data-call-target]')[0] as HTMLElement;
		first.click();
		expect(onSelect).toHaveBeenCalledTimes(1);
	});
});

describe('decorateCallTargets: TEARDOWN', () => {
	it('restores the subtree text content and releases listeners — activating the old location invokes nothing', () => {
		const root = document.createElement('code');
		root.textContent = 'Foo and Bar';
		const index = buildCallTargetIndex([node('Foo')]);
		const onSelect = vi.fn();

		const before = root.textContent;
		const teardown = decorateCallTargets(root, index, onSelect);
		expect(root.querySelectorAll('[data-call-target]')).toHaveLength(1);
		const decorated = root.querySelector('[data-call-target]') as HTMLElement;

		teardown();

		expect(root.textContent).toBe(before);
		expect(root.querySelectorAll('[data-call-target]')).toHaveLength(0);

		decorated.click();
		expect(onSelect).not.toHaveBeenCalled();
	});

	it('does not resurrect stale text when the owner ({@html}) already replaced the subtree', () => {
		// Reproduces WR-01: Svelte's {@html} can replace this element's
		// children with a fresh render before the previous decoration's
		// teardown runs (a direct single-def -> single-def rerender, with
		// no loading state interposed). Teardown must not append the old
		// render's text onto the new one.
		const root = document.createElement('code');
		root.textContent = 'Foo and Bar';
		const index = buildCallTargetIndex([node('Foo')]);
		const onSelect = vi.fn();

		const teardown = decorateCallTargets(root, index, onSelect);
		expect(root.querySelectorAll('[data-call-target]')).toHaveLength(1);

		// Simulate the owner replacing this element's children out from
		// under the decorator before teardown runs.
		root.innerHTML = 'Baz and Qux';

		teardown();

		// Positive assertion: the new owner's content is exactly what
		// survives — no stale text from the torn-down decoration appended.
		expect(root.textContent).toBe('Baz and Qux');
	});
});

describe('decorateCallTargets: the split-identifier bound (deliberate, documented)', () => {
	it('produces NO click target when a name is split across two sibling elements, and DOES produce one for the same name contiguous in a single text node', () => {
		// The index carries "FooBar" but NOT "Foo" or "Bar" alone — a text
		// node containing only "Foo" and a sibling text node containing
		// only "Bar" can never jointly match "FooBar", because each text
		// node is tokenized independently.
		const index: CallTargetIndex = buildCallTargetIndex([node('FooBar')]);

		const split = document.createElement('code');
		const span1 = document.createElement('span');
		span1.textContent = 'Foo';
		const span2 = document.createElement('span');
		span2.textContent = 'Bar';
		split.append(span1, span2);

		const onSelectSplit = vi.fn();
		decorateCallTargets(split, index, onSelectSplit);
		expect(split.querySelectorAll('[data-call-target]')).toHaveLength(0);

		const contiguous = document.createElement('code');
		contiguous.textContent = 'FooBar';
		const onSelectContiguous = vi.fn();
		decorateCallTargets(contiguous, index, onSelectContiguous);
		const targets = contiguous.querySelectorAll('[data-call-target]');
		expect(targets).toHaveLength(1);
		expect(targets[0]?.textContent).toBe('FooBar');
	});
});
