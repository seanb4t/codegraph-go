// call-targets.ts is BRW-04's click-to-definition matcher and DOM
// decorator (D-18). There is exactly one viable shape this phase: the
// Pebble edge key omits line/col (internal/schema/graph.proto's Edge
// message, lines 70-95), so the store cannot enumerate call sites and
// server-side resolved reference spans are structurally impossible here.
// Resolution is therefore TEXT-DRIVEN — an identifier in rendered source
// is clickable only when its text exactly matches a name already on the
// wire in the opened node's `calls` list (GetNodeDetailResponse.calls,
// internal/uiproto/uiv1/ui.proto:397-462) — and BRW-05's picker exists in
// this same plan precisely because that resolution is inherently
// ambiguous: a click always re-issues GetNodeDetail by symbol name, and a
// name with several definitions lands there.
//
// Matching is exact string equality in both directions: no lowercasing,
// no Unicode normalization, no prefix or fuzzy comparison. The wire's
// name and the source text either agree byte-for-byte or the identifier
// is simply not clickable — anything looser would produce a click target
// that jumps somewhere the developer did not point at, which is worse
// than inert text.
//
// decorateCallTargets walks TEXT NODES ONLY (a DOM TreeWalker) and
// constructs replacement elements/text nodes through DOM APIs —
// createElement/createTextNode/insertBefore/replaceWith — never by
// assembling or assigning an HTML-string. The input here is verbatim,
// occasionally adversarial third-party repository content; highlight.ts
// (web/src/lib/highlight.ts) is the ONE other place in web/src that
// builds markup from repository bytes, and this module must never become
// a second one.
//
// SPLIT-IDENTIFIER BOUND (deliberate, not a bug): highlight.js can place
// parts of one qualified identifier in separate sibling elements — a
// package qualifier styled differently from the member it qualifies. A
// TreeWalker tokenizing each text node independently cannot see across
// that element boundary, so such a name produces NO click target. This
// module does not attempt to reassemble text across element boundaries
// to "fix" that: reassembly means inventing where a logical token begins
// across arbitrary markup, and getting that wrong produces a click
// target spanning DOM the user did not point at — worse than an
// identifier that is simply not clickable, the same failure mode exact
// matching already accepts everywhere else in this module.
//
// Node (the DOM interface) and Node (the wire message type from
// $lib/gen/ui_pb) share a name — the wire type is imported here as
// GraphNode specifically to avoid shadowing the global DOM Node
// interface this module's TreeWalker/element code depends on.
import type { Node as GraphNode } from '$lib/gen/ui_pb';

// IDENTIFIER_PATTERN is the Unicode-aware identifier tokenizer: an
// ID_Start code point (or `_`/`$`) followed by zero or more ID_Continue
// code points (or `_`/`$`, plus the two zero-width joiner characters
// ECMAScript's own IdentifierPart grammar permits). Using Unicode
// property escapes means a non-ASCII identifier is captured as ONE
// token, never split at its first non-ASCII code point and matched as a
// fragment.
export const IDENTIFIER_PATTERN = /[\p{ID_Start}$_][\p{ID_Continue}$\u200C\u200D]*/gu;

// CallTargetIndex maps a call target's exact NAME (as the wire sent it)
// to every GraphNode sharing that name — a list, not a single value, so
// an ambiguous name (several call targets sharing one identifier) is
// representable rather than silently collapsed. That ambiguity is real;
// resolving it is the disambiguation picker's job (BRW-05), not this
// module's.
export type CallTargetIndex = ReadonlyMap<string, readonly GraphNode[]>;

// buildCallTargetIndex builds the index keyed by the node name string
// exactly as the wire supplied it. Over an empty call list it returns an
// empty (but valid, never null/undefined) index.
export function buildCallTargetIndex(calls: readonly GraphNode[]): CallTargetIndex {
	const index = new Map<string, GraphNode[]>();
	for (const n of calls) {
		const existing = index.get(n.name);
		if (existing) {
			existing.push(n);
		} else {
			index.set(n.name, [n]);
		}
	}
	return index;
}

// lookupCallTarget returns the exact-match entry for identifier, or
// undefined when no call target carries that exact name. Matching is
// exact string equality only — no case folding, no normalization, no
// prefix/superstring matching. Safe to call on an empty index; never
// throws.
export function lookupCallTarget(
	index: CallTargetIndex,
	identifier: string
): readonly GraphNode[] | undefined {
	return index.get(identifier);
}

// DECORATED_ATTR marks a produced click-target element so a later
// decorateCallTargets pass over the SAME subtree can recognize it and
// skip re-decorating its own contents (idempotence, independent of
// whatever lifecycle wraps this function — the Svelte action below makes
// double-application unlikely, this makes it harmless regardless).
const DECORATED_ATTR = 'data-call-target';

interface HandlerRecord {
	el: Element;
	handler: EventListener;
}

interface DecorationRecord {
	parent: globalThis.Node;
	originalText: Text;
	insertedNodes: globalThis.Node[];
}

// decorateTextNode tokenizes one text node's data and, for every token
// that exactly matches an index entry, replaces that token's span with an
// activatable element — constructed via createElement/createTextNode
// only, never an HTML string. A text node with no matching token is left
// completely untouched (not even re-inserted), so "leaves every other
// text node byte-identical" holds without qualification.
function decorateTextNode(
	textNode: Text,
	index: CallTargetIndex,
	onSelect: (entry: readonly GraphNode[]) => void,
	decorations: DecorationRecord[],
	handlers: HandlerRecord[]
): void {
	const text = textNode.data;
	const matches = [...text.matchAll(IDENTIFIER_PATTERN)];
	if (matches.length === 0) return;

	const parts: globalThis.Node[] = [];
	let cursor = 0;
	let hasMatch = false;

	for (const m of matches) {
		const token = m[0];
		const start = m.index ?? 0;
		const entry = lookupCallTarget(index, token);
		if (!entry || entry.length === 0) continue;

		hasMatch = true;
		if (start > cursor) {
			parts.push(document.createTextNode(text.slice(cursor, start)));
		}

		const el = document.createElement('button');
		el.type = 'button';
		el.setAttribute(DECORATED_ATTR, 'true');
		el.className = 'call-target cursor-pointer underline decoration-dotted underline-offset-2';
		el.textContent = token;
		// A native <button> is activatable by mouse AND by keyboard
		// (Enter/Space trigger a click event natively) with no extra
		// keydown wiring needed — the accessible activation path this
		// task requires, without a second event-handling code path to
		// keep in sync with the mouse one.
		const handler: EventListener = () => onSelect(entry);
		el.addEventListener('click', handler);
		handlers.push({ el, handler });

		parts.push(el);
		cursor = start + token.length;
	}

	if (!hasMatch) return;

	if (cursor < text.length) {
		parts.push(document.createTextNode(text.slice(cursor)));
	}

	const parent = textNode.parentNode;
	if (!parent) return;

	textNode.replaceWith(...parts);
	decorations.push({ parent, originalText: textNode, insertedNodes: parts });
}

// decorateCallTargets walks `root`'s text nodes with a TreeWalker,
// decorates every matching identifier token, and returns a TEARDOWN
// function that removes every listener it attached and restores the
// subtree's text content to its pre-decoration state exactly (each
// decorated span is replaced back with its ORIGINAL text node, not a
// freshly-normalized approximation).
//
// IDEMPOTENT independently of any caller lifecycle: the TreeWalker's
// acceptNode filter rejects any text node whose ancestor chain (up to
// root) already carries DECORATED_ATTR — i.e. text living inside an
// already-produced click-target element — so running this twice over the
// same subtree with no DOM change in between decorates nothing the
// second time and produces a DOM byte-identical to running it once.
export function decorateCallTargets(
	root: Element,
	index: CallTargetIndex,
	onSelect: (entry: readonly GraphNode[]) => void
): () => void {
	const decorations: DecorationRecord[] = [];
	const handlers: HandlerRecord[] = [];

	const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT, {
		acceptNode(node) {
			let el = node.parentElement;
			while (el && el !== root) {
				if (el.hasAttribute(DECORATED_ATTR)) return NodeFilter.FILTER_REJECT;
				el = el.parentElement;
			}
			return NodeFilter.FILTER_ACCEPT;
		}
	});

	const textNodes: Text[] = [];
	let current: globalThis.Node | null;
	while ((current = walker.nextNode())) {
		textNodes.push(current as Text);
	}

	for (const textNode of textNodes) {
		decorateTextNode(textNode, index, onSelect, decorations, handlers);
	}

	return () => {
		for (const { el, handler } of handlers) {
			el.removeEventListener('click', handler);
		}
		for (const { parent, originalText, insertedNodes } of decorations) {
			const anchor = insertedNodes[0];
			if (anchor && anchor.parentNode === parent) {
				parent.insertBefore(originalText, anchor);
			} else {
				// The anchor is no longer where we left it (something else
				// mutated this subtree between decoration and teardown) —
				// append rather than silently drop the original text.
				parent.appendChild(originalText);
			}
			for (const inserted of insertedNodes) {
				inserted.parentNode?.removeChild(inserted);
			}
		}
	};
}

// CallTargetsActionParams is what the Svelte action below re-decorates
// on every update — a fresh index (the currently opened node's calls
// list) and the callback the host component wants invoked on selection.
export interface CallTargetsActionParams {
	index: CallTargetIndex;
	onSelect: (entry: readonly GraphNode[]) => void;
}

// callTargets is the Svelte action wrapping decorateCallTargets with
// `update` (re-decorate when the index or the rendered source changes)
// and `destroy` (run the teardown) — the lifecycle decorateCallTargets
// itself does not own. A Svelte rerender destroys and recreates the
// element Svelte's html directive renders into; the host component
// applies this action to that SAME element rather than calling the
// decorator from an effect, so the action's own update/destroy hooks
// are the lifecycle authority.
export function callTargets(node: Element, params: CallTargetsActionParams) {
	let teardown = decorateCallTargets(node, params.index, params.onSelect);
	return {
		update(next: CallTargetsActionParams) {
			teardown();
			teardown = decorateCallTargets(node, next.index, next.onSelect);
		},
		destroy() {
			teardown();
		}
	};
}
