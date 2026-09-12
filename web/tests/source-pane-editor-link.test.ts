// source-pane-editor-link.test.ts — 09-04 Task 2: SourcePane's header
// "Open in editor" link from a load-time GetEditorLink probe, and the
// line-number gutter's one-rpc-per-click handoff. Written and run RED
// against the pre-Task-2 component (no editor-link surfaces, no gutter
// click handler) before the changes exist. Fixture builders are small,
// deliberately-copied helpers matching source-pane-breadcrumb.test.ts's
// own convention (not exported/shared across test files).
import { render, screen, waitFor, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';

import SourcePane from '$lib/components/browse/SourcePane.svelte';
import { writeEditorOverride, clearEditorOverride } from '$lib/editor-prefs';
import {
	PermalinkAvailability,
	EditorLinkAvailability,
	EditorTemplateSource
} from '$lib/gen/ui_pb';
import type { BrowseTargetState, SourceRender } from '$lib/browse-state';
import type { GetEditorLinkResponse, GetPermalinkResponse, Node } from '$lib/gen/ui_pb';

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
		source: new TextEncoder().encode('a\nb\nc\nd\n'),
		truncated: false,
		totalLines: 4,
		returnedLines: 4,
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

function editorLinkResponse(overrides: Partial<GetEditorLinkResponse> = {}): GetEditorLinkResponse {
	return {
		url: '',
		availability: EditorLinkAvailability.NO_TEMPLATE,
		reason: '',
		defaultSource: EditorTemplateSource.NONE,
		defaultEditor: '',
		overrideApplied: false,
		presets: [
			{ id: 'vscode', name: 'VS Code', template: 'vscode://file/{path}:{line}:{col}' },
			{ id: 'cursor', name: 'Cursor', template: 'cursor://file/{path}:{line}:{col}' },
			{
				id: 'jetbrains',
				name: 'JetBrains',
				template: 'jetbrains://gateway/navigate/reference?path={path}&line={line}'
			}
		],
		...overrides
	} as GetEditorLinkResponse;
}

type GetEditorLinkMock = ReturnType<typeof vi.fn>;

function stubClient(opts: { getEditorLink?: GetEditorLinkMock } = {}) {
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
	return opts.getEditorLink ? { ...base, getEditorLink: opts.getEditorLink } : base;
}

beforeEach(() => {
	clearEditorOverride();
});

describe('SourcePane: header "Open in editor" load-time probe', () => {
	it('resolves a real <a href> from the probe for a file target, requesting line 1 col 1', async () => {
		const getEditorLink = vi.fn(async () =>
			editorLinkResponse({
				url: 'vscode://file/x/a.go:1:1',
				availability: EditorLinkAvailability.BUILDABLE,
				defaultSource: EditorTemplateSource.FLAG
			})
		);
		render(SourcePane, {
			props: { state: fileState({ path: 'x/a.go' }), client: stubClient({ getEditorLink }) }
		});
		const link = await screen.findByTestId('editor-link');
		expect(link.getAttribute('href')).toBe('vscode://file/x/a.go:1:1');
		expect(link.getAttribute('rel')).toBe('noreferrer');
		expect(link.getAttribute('target')).toBeNull();
		expect(getEditorLink).toHaveBeenCalledTimes(1);
		expect(getEditorLink.mock.calls[0][0]).toEqual({
			path: 'x/a.go',
			line: 1,
			col: 1,
			template: undefined
		});
	});

	it('requests node.startCol + 1 for a single-def target', async () => {
		const getEditorLink = vi.fn(async () => editorLinkResponse());
		const n = node('Foo', { filePath: 'pkg/foo.go', startLine: 10, startCol: 4 });
		render(SourcePane, {
			props: { state: singleDefState({ node: n }), client: stubClient({ getEditorLink }) }
		});
		await waitFor(() => expect(getEditorLink).toHaveBeenCalled());
		expect(getEditorLink.mock.calls[0][0]).toEqual({
			path: 'pkg/foo.go',
			line: 10,
			col: 5,
			template: undefined
		});
	});

	it('sends a pre-seeded custom override as the probe template', async () => {
		writeEditorOverride({ kind: 'custom', template: 'cursor://file/{path}:{line}' });
		const getEditorLink = vi.fn(async () => editorLinkResponse());
		render(SourcePane, { props: { state: fileState(), client: stubClient({ getEditorLink }) } });
		await waitFor(() => expect(getEditorLink).toHaveBeenCalled());
		expect(getEditorLink.mock.calls[0][0].template).toBe('cursor://file/{path}:{line}');
	});

	it('renders no editor surfaces when the client lacks getEditorLink, and does not throw', () => {
		expect(() =>
			render(SourcePane, { props: { state: fileState(), client: stubClient() } })
		).not.toThrow();
		expect(screen.queryByTestId('editor-link')).toBeNull();
		expect(screen.queryByTestId('editor-link-reason')).toBeNull();
		expect(screen.queryByTestId('editor-link-picker-toggle')).toBeNull();
	});

	it('degrades to idle on a rejected getEditorLink promise, no unhandled rejection', async () => {
		const getEditorLink = vi.fn(async () => {
			throw new Error('boom');
		});
		render(SourcePane, { props: { state: fileState(), client: stubClient({ getEditorLink }) } });
		await waitFor(() => expect(getEditorLink).toHaveBeenCalled());
		await new Promise((resolve) => setTimeout(resolve, 0));
		expect(screen.queryByTestId('editor-link')).toBeNull();
		expect(screen.queryByTestId('editor-link-reason')).toBeNull();
		expect(screen.queryByTestId('editor-link-picker-toggle')).toBeNull();
	});
});

describe('SourcePane: NO_TEMPLATE / TEMPLATE_INVALID answers', () => {
	it('NO_TEMPLATE (defaultSource NONE) shows the reason, the toggle, and opens the picker on first use; gutter stays plain', async () => {
		const getEditorLink = vi.fn(async () =>
			editorLinkResponse({
				availability: EditorLinkAvailability.NO_TEMPLATE,
				reason: 'no editor configured',
				defaultSource: EditorTemplateSource.NONE
			})
		);
		render(SourcePane, { props: { state: fileState(), client: stubClient({ getEditorLink }) } });
		const reason = await screen.findByTestId('editor-link-reason');
		expect(reason.textContent?.trim()).toBe('no editor configured');
		expect(screen.getByTestId('editor-link-picker-toggle')).toBeTruthy();
		expect(screen.queryByTestId('editor-link')).toBeNull();
		await waitFor(() => expect(screen.getByTestId('editor-link-picker')).toBeTruthy());
		for (const g of screen.getAllByTestId(/gutter-line-/)) {
			expect(g.tagName).toBe('SPAN');
		}
	});

	it('NO_TEMPLATE (defaultSource DISABLED) renders no editor surface at all', async () => {
		const getEditorLink = vi.fn(async () =>
			editorLinkResponse({
				availability: EditorLinkAvailability.NO_TEMPLATE,
				reason: 'disabled by operator',
				defaultSource: EditorTemplateSource.DISABLED
			})
		);
		render(SourcePane, { props: { state: fileState(), client: stubClient({ getEditorLink }) } });
		await waitFor(() => expect(getEditorLink).toHaveBeenCalled());
		expect(screen.queryByTestId('editor-link')).toBeNull();
		expect(screen.queryByTestId('editor-link-reason')).toBeNull();
		expect(screen.queryByTestId('editor-link-picker-toggle')).toBeNull();
		expect(screen.queryByTestId('editor-link-picker')).toBeNull();
		for (const g of screen.getAllByTestId(/gutter-line-/)) {
			expect(g.tagName).toBe('SPAN');
		}
	});

	it('TEMPLATE_INVALID shows the reason and the toggle; gutter stays plain', async () => {
		const getEditorLink = vi.fn(async () =>
			editorLinkResponse({
				availability: EditorLinkAvailability.TEMPLATE_INVALID,
				reason: 'unknown scheme: zed',
				overrideApplied: true
			})
		);
		render(SourcePane, { props: { state: fileState(), client: stubClient({ getEditorLink }) } });
		const reason = await screen.findByTestId('editor-link-reason');
		expect(reason.textContent?.trim()).toBe('unknown scheme: zed');
		expect(screen.getByTestId('editor-link-picker-toggle')).toBeTruthy();
		for (const g of screen.getAllByTestId(/gutter-line-/)) {
			expect(g.tagName).toBe('SPAN');
		}
	});
});

describe('SourcePane: gutter one-rpc-per-click handoff', () => {
	it('BUILDABLE gutter cells are buttons; a click issues one rpc then navigates via location.assign, a burst click is ignored', async () => {
		let resolveClick: ((r: GetEditorLinkResponse) => void) | undefined;
		const getEditorLink = vi.fn((_request: unknown) => {
			if (getEditorLink.mock.calls.length === 1) {
				return Promise.resolve(
					editorLinkResponse({
						availability: EditorLinkAvailability.BUILDABLE,
						url: 'vscode://file/x/a.go:1:1'
					})
				);
			}
			return new Promise<GetEditorLinkResponse>((resolve) => {
				resolveClick = resolve;
			});
		});
		const assignSpy = vi.fn();
		Object.defineProperty(window, 'location', {
			value: { ...window.location, assign: assignSpy },
			writable: true,
			configurable: true
		});

		render(SourcePane, {
			props: { state: fileState({ path: 'x/a.go' }), client: stubClient({ getEditorLink }) }
		});
		await screen.findByTestId('editor-link');

		const gutterButtons = screen.getAllByTestId(/gutter-line-/);
		expect(gutterButtons.length).toBeGreaterThanOrEqual(4);
		for (const btn of gutterButtons) {
			expect(btn.tagName).toBe('BUTTON');
		}

		await fireEvent.click(screen.getByTestId('gutter-line-3'));
		await fireEvent.click(screen.getByTestId('gutter-line-4'));

		await waitFor(() => expect(getEditorLink).toHaveBeenCalledTimes(2));
		expect(getEditorLink.mock.calls[1][0]).toEqual({
			path: 'x/a.go',
			line: 3,
			col: 1,
			template: undefined
		});

		resolveClick?.(
			editorLinkResponse({
				availability: EditorLinkAvailability.BUILDABLE,
				url: 'vscode://file/x/a.go:3:1'
			})
		);
		await waitFor(() => expect(assignSpy).toHaveBeenCalledWith('vscode://file/x/a.go:3:1'));
		expect(getEditorLink).toHaveBeenCalledTimes(2);
	});
});
