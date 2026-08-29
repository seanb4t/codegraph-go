// source-pane.test.ts — 03-08 Task 2: SourcePane's truncation notice,
// permalink surface, and CopyAction's copy affordance. Written and run
// RED before the SourcePane extensions and CopyAction.svelte exist
// (RED observation recorded in the SUMMARY), then made GREEN.
import { render, screen, waitFor, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';

import SourcePane from '$lib/components/browse/SourcePane.svelte';
import CopyAction from '$lib/components/browse/CopyAction.svelte';
import { PermalinkAvailability } from '$lib/gen/ui_pb';
import type { BrowseTargetState, SourceRender } from '$lib/browse-state';
import type { GetPermalinkResponse, Node } from '$lib/gen/ui_pb';

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

function fileState(overrides: Partial<BrowseTargetState & { kind: 'file' }> = {}): BrowseTargetState {
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

// PermalinkRequest is a minimal duck type of GetPermalinkRequest — kept
// independent of the generated message's full prototype machinery,
// mirroring browse-tracer.test.ts's own stub convention. The mock
// implementation function (not `.mockResolvedValue()` alone) gives
// vitest a concrete signature to infer, so the resulting stub is
// structurally assignable to SourcePane's own (unexported) client prop
// type without a type-erasing cast.
interface PermalinkRequest {
	path: string;
	line?: number;
	endLine?: number;
}

function permalinkClient(response: Partial<GetPermalinkResponse>) {
	const full = {
		url: '',
		availability: PermalinkAvailability.NO_LINK,
		reason: '',
		...response
	} as GetPermalinkResponse;
	return {
		getPermalink: vi.fn(async (_request: PermalinkRequest) => full)
	};
}

describe('SourcePane: truncation notice — absence and presence are paired', () => {
	it('renders no notice when truncated is false, and a notice naming both counts when true', () => {
		const { unmount } = render(SourcePane, {
			props: { state: fileState({ truncated: false, totalLines: 5, returnedLines: 5 }) }
		});
		expect(screen.queryByTestId('browse-truncated')).toBeNull();
		unmount();

		render(SourcePane, {
			props: { state: fileState({ truncated: true, totalLines: 500, returnedLines: 100 }) }
		});
		const notice = screen.getByTestId('browse-truncated');
		expect(notice.textContent).toContain('100');
		expect(notice.textContent).toContain('500');
	});
});

describe('SourcePane: the permalink surface renders three PAIRWISE DISTINCT states', () => {
	it('LINKABLE, LINKABLE_UNVERIFIED and NO_LINK each render distinct output, and the unverified reason is visible', async () => {
		const linkable = permalinkClient({
			url: 'https://github.com/o/r/blob/sha/foo.go#L10-L20',
			availability: PermalinkAvailability.LINKABLE
		});
		const { unmount: unmount1 } = render(SourcePane, {
			props: { state: singleDefState(), client: linkable }
		});
		await waitFor(() => expect(screen.getByTestId('permalink-linkable')).toBeInTheDocument());
		const linkableText = screen.getByTestId('permalink-linkable').textContent;
		unmount1();

		const unverified = permalinkClient({
			url: 'https://github.com/o/r/blob/sha/foo.go#L10-L20',
			availability: PermalinkAvailability.LINKABLE_UNVERIFIED,
			reason: 'commit not found on any remote-tracking branch'
		});
		const { unmount: unmount2 } = render(SourcePane, {
			props: { state: singleDefState(), client: unverified }
		});
		await waitFor(() => expect(screen.getByTestId('permalink-unverified')).toBeInTheDocument());
		const unverifiedEl = screen.getByTestId('permalink-unverified');
		expect(unverifiedEl.textContent).toContain('commit not found on any remote-tracking branch');
		const unverifiedText = unverifiedEl.textContent;
		unmount2();

		const noLink = permalinkClient({
			availability: PermalinkAvailability.NO_LINK,
			reason: 'no GitHub remote configured'
		});
		render(SourcePane, { props: { state: singleDefState(), client: noLink } });
		await waitFor(() => expect(screen.getByTestId('permalink-no-link')).toBeInTheDocument());
		const noLinkEl = screen.getByTestId('permalink-no-link');
		expect(noLinkEl.textContent).toContain('no GitHub remote configured');
		const noLinkText = noLinkEl.textContent;

		expect(linkableText).not.toBe(unverifiedText);
		expect(linkableText).not.toBe(noLinkText);
		expect(unverifiedText).not.toBe(noLinkText);
	});
});

describe('SourcePane: permalink request shape — truncated vs. an opened node, paired', () => {
	it('a truncated source requests NO end line; a non-truncated opened node DOES carry one', async () => {
		const truncatedClient = permalinkClient({});
		render(SourcePane, {
			props: {
				state: singleDefState({ source: sourceRender({ truncated: true }) }),
				client: truncatedClient
			}
		});
		await waitFor(() => expect(truncatedClient.getPermalink).toHaveBeenCalled());
		const truncatedRequest = truncatedClient.getPermalink.mock.calls[0][0];
		expect(truncatedRequest.endLine).toBeUndefined();

		const openedClient = permalinkClient({});
		render(SourcePane, {
			props: {
				state: singleDefState({ source: sourceRender({ truncated: false }) }),
				client: openedClient
			}
		});
		await waitFor(() => expect(openedClient.getPermalink).toHaveBeenCalled());
		const openedRequest = openedClient.getPermalink.mock.calls[0][0];
		expect(openedRequest.endLine).toBe(20);
		expect(openedRequest.line).toBe(10);
	});
});

describe('CopyAction: an empty value renders nothing; a non-empty value renders the control', () => {
	it('is absent for an empty string, and present for a non-empty one', () => {
		const { unmount } = render(CopyAction, { props: { value: '', label: 'file path' } });
		expect(screen.queryByTestId('copy-action-file path')).toBeNull();
		unmount();

		render(CopyAction, { props: { value: 'internal/query/node.go', label: 'file path' } });
		expect(screen.getByTestId('copy-action-file path')).toBeInTheDocument();
	});
});

describe('CopyAction: writes the EXACT displayed string, untrimmed and unnormalized', () => {
	beforeEach(() => {
		Object.defineProperty(navigator, 'clipboard', {
			value: { writeText: vi.fn().mockResolvedValue(undefined) },
			configurable: true,
			writable: true
		});
	});

	it('copies the value byte-for-byte, including leading/trailing whitespace', async () => {
		const value = '  internal/query/node.go  ';
		render(CopyAction, { props: { value, label: 'file path' } });
		await fireEvent.click(screen.getByTestId('copy-action-file path'));
		expect(navigator.clipboard.writeText).toHaveBeenCalledWith(value);
	});

	it('two copy actions in quick succession leave the clipboard holding the SECOND value', async () => {
		render(CopyAction, { props: { value: 'path/one.go', label: 'file path' } });
		render(CopyAction, { props: { value: 'SymbolTwo', label: 'symbol name' } });
		await fireEvent.click(screen.getByTestId('copy-action-file path'));
		await fireEvent.click(screen.getByTestId('copy-action-symbol name'));
		expect(navigator.clipboard.writeText).toHaveBeenLastCalledWith('SymbolTwo');
	});
});

describe('SourcePane: copy affordances for a single-def target', () => {
	it('offers a distinctly-labelled copy control for the file path and the symbol name', () => {
		render(SourcePane, { props: { state: singleDefState({ node: node('Foo', { filePath: 'a/b.go' }) }) } });
		expect(screen.getByTestId('copy-action-file path')).toBeInTheDocument();
		expect(screen.getByTestId('copy-action-symbol name')).toBeInTheDocument();
	});
});
