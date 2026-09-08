// definition-picker.test.ts — 03-08 Task 3: DefinitionPicker.svelte's own
// tests, written and run RED before the component exists (RED observation
// recorded in the SUMMARY), then made GREEN.
import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';

import DefinitionPicker from '$lib/components/browse/DefinitionPicker.svelte';
import { NAV_INTENT } from '$lib/browse-nav';
import { loadBrowseTarget, type NodeDetailClient } from '$lib/browse-state';
import { NodeDetailMode } from '$lib/gen/ui_pb';
import type { Node, NodeDefinition, GetNodeDetailResponse } from '$lib/gen/ui_pb';

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

function definition(
	name: string,
	overrides: Partial<{ detailGathered: boolean; calls: Node[]; calledBy: Node[] }> = {}
): NodeDefinition {
	return {
		node: node(name),
		calls: [],
		calledBy: [],
		detailGathered: true,
		source: undefined,
		...overrides
	} as unknown as NodeDefinition;
}

describe('DefinitionPicker: lists every RETURNED candidate and states the true total', () => {
	it('renders one entry per candidate and shows the true total alongside', () => {
		render(DefinitionPicker, {
			props: {
				symbol: 'Foo',
				definitions: [definition('Foo', {}), definition('Foo', {})],
				totalCandidates: 2,
				onNavigate: vi.fn()
			}
		});
		const list = screen.getByTestId('definition-picker-list');
		expect(list.querySelectorAll('[data-candidate]')).toHaveLength(2);
		expect(screen.getByTestId('definition-picker-summary').textContent).toContain('2');
	});
});

describe('DefinitionPicker: the omitted-count line, paired equal-vs-unequal', () => {
	it('says nothing extra when totalCandidates equals definitions.length', () => {
		render(DefinitionPicker, {
			props: {
				symbol: 'Foo',
				definitions: [definition('Foo')],
				totalCandidates: 1,
				onNavigate: vi.fn()
			}
		});
		expect(screen.queryByTestId('definition-picker-omitted-count')).toBeNull();
	});

	it('states the difference in words when totalCandidates exceeds definitions.length (today\'s server never produces this — the test does)', () => {
		render(DefinitionPicker, {
			props: {
				symbol: 'Foo',
				definitions: [definition('Foo')],
				totalCandidates: 5,
				onNavigate: vi.fn()
			}
		});
		const omitted = screen.getByTestId('definition-picker-omitted-count');
		expect(omitted.textContent).toContain('1');
		expect(omitted.textContent).toContain('5');
	});
});

describe('DefinitionPicker: ungathered marker uses the WIRE FLAG, never inferred from emptiness', () => {
	it('a false-flag candidate with an EMPTY call list carries the marker; a true-flag candidate with an EMPTY call list does NOT', () => {
		render(DefinitionPicker, {
			props: {
				symbol: 'Foo',
				definitions: [
					definition('Foo', { detailGathered: false, calls: [], calledBy: [] }),
					definition('Foo', { detailGathered: true, calls: [], calledBy: [] })
				],
				totalCandidates: 2,
				onNavigate: vi.fn()
			}
		});
		expect(screen.getByTestId('definition-candidate-ungathered-0')).toBeInTheDocument();
		expect(screen.queryByTestId('definition-candidate-ungathered-1')).toBeNull();
	});
});

describe('DefinitionPicker: nothing is pre-selected', () => {
	it('no candidate carries a selected/active state on initial render, over a non-empty list', () => {
		render(DefinitionPicker, {
			props: {
				symbol: 'Foo',
				definitions: [definition('Foo'), definition('Foo')],
				totalCandidates: 2,
				onNavigate: vi.fn()
			}
		});
		const list = screen.getByTestId('definition-picker-list');
		const candidates = list.querySelectorAll('[data-candidate]');
		expect(candidates.length).toBeGreaterThan(0);
		for (const el of Array.from(candidates)) {
			expect(el.getAttribute('aria-selected')).toBeNull();
			expect(el.className).not.toMatch(/selected|active/);
		}
	});
});

describe('DefinitionPicker: rendered order equals the supplied order, element for element', () => {
	it('does not reorder candidates', () => {
		render(DefinitionPicker, {
			props: {
				symbol: 'Foo',
				definitions: [
					definition('Foo', {}),
					definition('Foo', {}) // both share the name; order is positional
				].map((d, i) => ({
					...d,
					node: { ...d.node, filePath: `${String.fromCharCode(122 - i)}.go` }
				})) as unknown as NodeDefinition[],
				totalCandidates: 2,
				onNavigate: vi.fn()
			}
		});
		const list = screen.getByTestId('definition-picker-list');
		const texts = Array.from(list.querySelectorAll('[data-candidate]')).map(
			(el) => el.textContent
		);
		// The first supplied entry has path "z.go", the second "y.go" —
		// rendered order must match that, not alphabetical.
		expect(texts[0]).toContain('z.go');
		expect(texts[1]).toContain('y.go');
	});
});

describe('DefinitionPicker: selecting a candidate navigates with symbol+file+line, all three', () => {
	it('invokes onNavigate with the NAVIGATE intent carrying the picked candidate\'s symbol, file and line', async () => {
		const onNavigate = vi.fn();
		render(DefinitionPicker, {
			props: {
				symbol: 'Foo',
				definitions: [definition('Foo', {})],
				totalCandidates: 1,
				onNavigate
			}
		});
		await fireEvent.click(screen.getByTestId('definition-candidate-0'));
		expect(onNavigate).toHaveBeenCalledWith(
			{ symbol: 'Foo', file: 'Foo.go', line: 1 },
			NAV_INTENT.NAVIGATE
		);
	});

	it('the produced delta is NOT missing symbol — file+line alone would defeat BRW-05', async () => {
		const onNavigate = vi.fn();
		render(DefinitionPicker, {
			props: {
				symbol: 'Foo',
				definitions: [definition('Foo', {})],
				totalCandidates: 1,
				onNavigate
			}
		});
		await fireEvent.click(screen.getByTestId('definition-candidate-0'));
		const [delta] = onNavigate.mock.calls[0] as [{ symbol?: string; file?: string; line?: number }];
		expect(delta.symbol).toBe('Foo');
		expect(delta.file).toBeDefined();
		expect(delta.line).toBeDefined();
	});
});

describe('DefinitionPicker -> loadBrowseTarget: picking a candidate re-resolves to NodeDetailModeSingleDef', () => {
	it('the symbol+file+line triple, round-tripped through loadBrowseTarget, yields a single-def state — not a re-rendered "something"', async () => {
		const onNavigate = vi.fn();
		render(DefinitionPicker, {
			props: {
				symbol: 'Foo',
				definitions: [definition('Foo', {})],
				totalCandidates: 1,
				onNavigate
			}
		});
		await fireEvent.click(screen.getByTestId('definition-candidate-0'));
		const [delta] = onNavigate.mock.calls[0] as [{ symbol: string; file: string; line: number }];

		// Simulate the server's real buildNodeDetail behavior for a
		// symbol+file+line triple: enumerateSymbolDefs finds the matches,
		// NODE-03's file/line hints narrow them to exactly one, producing
		// NodeDetailModeSingleDef (internal/query/detail.go:190-234). A
		// stub client that asserts all three request fields were sent
		// proves the delta this component produced is what actually keeps
		// the server in symbol-narrowing mode rather than falling into the
		// symbol=="" FILE-mode branch.
		const stubClient: NodeDetailClient = {
			getNodeDetail: (request) => {
				expect(request.symbol).toBe('Foo');
				expect(request.file).toBe('Foo.go');
				expect(request.line).toBe(1);
				return Promise.resolve({
					mode: NodeDetailMode.SINGLE_DEF,
					path: '',
					node: node('Foo'),
					calls: [],
					calledBy: [],
					symbol: '',
					definitions: [],
					totalCandidates: 0,
					source: undefined
				} as unknown as GetNodeDetailResponse);
			}
		};

		const state = await loadBrowseTarget(
			{ symbol: delta.symbol, file: delta.file, line: delta.line, unknown: [] },
			stubClient
		);
		expect(state.kind).toBe('single-def');
	});
});

describe('DefinitionPicker: keyboard drivability', () => {
	it('ArrowDown moves focus to the next candidate; ArrowUp moves it back', async () => {
		render(DefinitionPicker, {
			props: {
				symbol: 'Foo',
				definitions: [definition('Foo', {}), definition('Foo', {})],
				totalCandidates: 2,
				onNavigate: vi.fn()
			}
		});
		const first = screen.getByTestId('definition-candidate-0');
		const second = screen.getByTestId('definition-candidate-1');
		first.focus();
		expect(document.activeElement).toBe(first);

		await fireEvent.keyDown(first, { key: 'ArrowDown' });
		expect(document.activeElement).toBe(second);

		await fireEvent.keyDown(second, { key: 'ArrowUp' });
		expect(document.activeElement).toBe(first);
	});
});

