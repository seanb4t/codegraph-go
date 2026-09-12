// source-pane-breadcrumb.test.ts — 09-03 Task 2 (BRW-10): SourcePane's
// per-line row/gutter restructure and the sticky breadcrumb bar. Written
// and run RED against the pre-restructure component (missing test ids,
// no per-line rows) before the SourcePane changes exist (RED transcript
// pasted in the SUMMARY), then made GREEN. Fixture builders are small,
// deliberately-copied helpers matching source-pane.test.ts's own
// convention (not exported/shared across test files).
import { render, screen, waitFor, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';

import SourcePane from '$lib/components/browse/SourcePane.svelte';
import { PermalinkAvailability } from '$lib/gen/ui_pb';
import type { BrowseTargetState, SourceRender } from '$lib/browse-state';
import type { FileSymbolsResponse, GetPermalinkResponse, Node } from '$lib/gen/ui_pb';

function node(name: string, overrides: Partial<Node> = {}): Node {
	return {
		id: name,
		kind: 'func',
		name,
		qualifiedName: `pkg.${name}`,
		filePath: `${name}.go`,
		language: 'go',
		startLine: 10,
		endLine: 20,
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

function fileState(
	overrides: Partial<BrowseTargetState & { kind: 'file' }> = {}
): BrowseTargetState {
	return {
		kind: 'file',
		path: 'internal/query/node.go',
		source: new TextEncoder().encode('package query\n'),
		truncated: false,
		totalLines: 1,
		returnedLines: 1,
		...overrides
	} as BrowseTargetState;
}

function sourceRender(overrides: Partial<SourceRender> = {}): SourceRender {
	return {
		content: new TextEncoder().encode('func Foo() {}\n'),
		truncated: false,
		totalLines: 1,
		returnedLines: 1,
		...overrides
	};
}

function singleDefState(
	overrides: Partial<{ node: Node; source?: SourceRender }> = {}
): BrowseTargetState {
	return {
		kind: 'single-def',
		node: node('Foo'),
		calls: [],
		calledBy: [],
		source: sourceRender(),
		...overrides
	} as BrowseTargetState;
}

function fileSymbolsResponse(overrides: Partial<FileSymbolsResponse> = {}): FileSymbolsResponse {
	return {
		symbols: [],
		totalCount: 0,
		truncated: false,
		...overrides
	} as FileSymbolsResponse;
}

interface FileSymbolsRequestLike {
	path: string;
}

function stubClient(
	opts: {
		fileSymbols?: (
			request: FileSymbolsRequestLike,
			options?: { signal?: AbortSignal }
		) => Promise<FileSymbolsResponse>;
	} = {}
) {
	const base = {
		getPermalink: vi.fn(
			async () =>
				({
					url: '',
					availability: PermalinkAvailability.NO_LINK,
					reason: ''
				}) as GetPermalinkResponse
		)
	};
	return opts.fileSymbols ? { ...base, fileSymbols: opts.fileSymbols } : base;
}

describe('SourcePane: per-line rows and gutter (both views)', () => {
	it('renders one DOM row and gutter cell per source line in the file view', () => {
		const source = new TextEncoder().encode('package q\n\nfunc A() {}\n');
		const { unmount } = render(SourcePane, {
			props: { state: fileState({ source, totalLines: 3, returnedLines: 3 }) }
		});
		for (const n of [1, 2, 3]) {
			const row = document.querySelector(`[data-line="${n}"]`);
			expect(row).not.toBeNull();
			expect(row?.id).toBe(`L${n}`);
			expect(screen.getByTestId(`gutter-line-${n}`).textContent).toBe(String(n));
		}
		expect(document.querySelector('[data-line="4"]')).toBeNull();
		unmount();
	});

	it('renders per-line rows and gutter cells in the single-def view too', () => {
		const source = sourceRender({ content: new TextEncoder().encode('func Foo() {}\nreturn\n') });
		const { unmount } = render(SourcePane, { props: { state: singleDefState({ source }) } });
		expect(document.querySelectorAll('[data-line]')).toHaveLength(2);
		expect(screen.getByTestId('gutter-line-1')).toBeInTheDocument();
		expect(screen.getByTestId('gutter-line-2')).toBeInTheDocument();
		unmount();
	});
});

describe('SourcePane: sticky breadcrumb bar (file view only)', () => {
	it('names the innermost symbol containing line 1 when one exists', async () => {
		const client = stubClient({
			fileSymbols: vi.fn(async () =>
				fileSymbolsResponse({ symbols: [node('A', { startLine: 1, endLine: 3 })], totalCount: 1 })
			)
		});
		const target = fileState();
		const { unmount } = render(SourcePane, { props: { state: target, client } });
		await waitFor(() =>
			expect(screen.getByTestId('source-breadcrumb-symbol').getAttribute('data-empty')).toBe(
				'false'
			)
		);
		expect(screen.getByTestId('source-breadcrumb')).toBeInTheDocument();
		expect(screen.getByTestId('source-breadcrumb-path').textContent).toContain(
			(target as { path: string }).path
		);
		expect(screen.getByTestId('source-breadcrumb-symbol').textContent).toContain('A');
		expect(screen.getByTestId('source-breadcrumb').getAttribute('data-current-line')).toBe('1');
		unmount();
	});

	it('renders the honest empty state when line 1 is inside no symbol — never the nearest preceding symbol', async () => {
		const client = stubClient({
			fileSymbols: vi.fn(async () =>
				fileSymbolsResponse({ symbols: [node('B', { startLine: 2, endLine: 3 })], totalCount: 1 })
			)
		});
		const { unmount } = render(SourcePane, { props: { state: fileState(), client } });
		await waitFor(() =>
			expect(screen.getByTestId('source-breadcrumb-symbol').getAttribute('data-empty')).toBe(
				'true'
			)
		);
		expect(screen.getByTestId('source-breadcrumb-symbol').textContent?.trim()).toBe('');
		expect(screen.getByTestId('source-breadcrumb')).toBeInTheDocument();
		expect(screen.getByTestId('source-breadcrumb-path')).toBeInTheDocument();
		unmount();
	});

	it('surfaces the MaxFileSymbols cap with the returned and total counts', async () => {
		const client = stubClient({
			fileSymbols: vi.fn(async () =>
				fileSymbolsResponse({
					symbols: [node('A', { startLine: 1, endLine: 3 })],
					totalCount: 2500,
					truncated: true
				})
			)
		});
		const { unmount } = render(SourcePane, { props: { state: fileState(), client } });
		await waitFor(() =>
			expect(screen.getByTestId('source-breadcrumb').getAttribute('data-symbols-truncated')).toBe(
				'true'
			)
		);
		expect(screen.getByTestId('source-breadcrumb').textContent).toContain('2500');
		unmount();
	});

	it('renders the empty breadcrumb state with no throw when the client lacks fileSymbols', async () => {
		const client = stubClient();
		expect(() => render(SourcePane, { props: { state: fileState(), client } })).not.toThrow();
		await waitFor(() =>
			expect(screen.getByTestId('source-breadcrumb-symbol').getAttribute('data-empty')).toBe(
				'true'
			)
		);
	});

	it('renders no breadcrumb at all for a single-def target (D-04, file view only)', () => {
		const { unmount } = render(SourcePane, { props: { state: singleDefState() } });
		expect(screen.queryByTestId('source-breadcrumb')).toBeNull();
		unmount();
	});

	it('fetches FileSymbols exactly once per path across a re-render, and aborts when the path changes', async () => {
		const calls: Array<{ path: string; signal?: AbortSignal }> = [];
		const fileSymbolsSpy = vi.fn(
			async (request: FileSymbolsRequestLike, options?: { signal?: AbortSignal }) => {
				calls.push({ path: request.path, signal: options?.signal });
				return fileSymbolsResponse();
			}
		);
		const client = stubClient({ fileSymbols: fileSymbolsSpy });

		const { rerender, unmount } = render(SourcePane, {
			props: { state: fileState({ path: 'a.go' }), client }
		});
		await waitFor(() => expect(fileSymbolsSpy).toHaveBeenCalledTimes(1));

		// A different target OBJECT with the SAME path must not re-dispatch.
		await rerender({ state: fileState({ path: 'a.go' }), client });
		await Promise.resolve();
		expect(fileSymbolsSpy).toHaveBeenCalledTimes(1);

		const firstSignal = calls[0].signal;
		expect(firstSignal?.aborted).toBe(false);

		// A genuinely different path must dispatch again and abort the first.
		await rerender({ state: fileState({ path: 'b.go' }), client });
		await waitFor(() => expect(fileSymbolsSpy).toHaveBeenCalledTimes(2));
		expect(firstSignal?.aborted).toBe(true);

		unmount();
	});

	it('scrolls the row at the symbol start line into view when the non-empty crumb is clicked', async () => {
		const scrollSpy = vi.fn();
		const original = Element.prototype.scrollIntoView;
		Element.prototype.scrollIntoView = scrollSpy;
		try {
			const client = stubClient({
				fileSymbols: vi.fn(async () =>
					fileSymbolsResponse({ symbols: [node('A', { startLine: 1, endLine: 3 })], totalCount: 1 })
				)
			});
			const { unmount } = render(SourcePane, {
				props: { state: fileState({ source: new TextEncoder().encode('a\nb\nc\n') }), client }
			});
			await waitFor(() =>
				expect(screen.getByTestId('source-breadcrumb-symbol').getAttribute('data-empty')).toBe(
					'false'
				)
			);
			const button = screen.getByTestId('source-breadcrumb-symbol').querySelector('button');
			expect(button).not.toBeNull();
			await fireEvent.click(button as Element);
			expect(scrollSpy).toHaveBeenCalled();
			unmount();
		} finally {
			Element.prototype.scrollIntoView = original;
		}
	});
});
