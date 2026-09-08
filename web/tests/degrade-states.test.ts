// degrade-states.test.ts — 03-09 Task 2: the layout status banner and
// the source pane's completed degrade rendering. Written and run RED
// before StatusBanner.svelte existed and before SourcePane's
// indexStale-split source-absent message existed (recorded in the
// SUMMARY), then made GREEN.
import { render, screen } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import { ConnectError, Code } from '@connectrpc/connect';

import StatusBanner from '$lib/components/StatusBanner.svelte';
import SourcePane from '$lib/components/browse/SourcePane.svelte';
import { loadBrowseTarget, type NodeDetailClient } from '$lib/browse-state';
import type { IndexStatus } from '$lib/status';
import type { BrowseTargetState } from '$lib/browse-state';
import type { Node } from '$lib/gen/ui_pb';

function status(overrides: Partial<IndexStatus> = {}): IndexStatus {
	const commit = overrides.commit ?? 'known';
	return { verdict: 'ok', commit, commitSha: commit === 'known' ? 'deadbeef' : '', ...overrides };
}

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

function singleDefStateNoSource(): BrowseTargetState {
	return {
		kind: 'single-def',
		node: node('Foo'),
		calls: [],
		calledBy: [],
		source: undefined
	} as BrowseTargetState;
}

function failingClient(error: unknown): NodeDetailClient {
	return {
		getNodeDetail: () => Promise.reject(error)
	};
}

describe('StatusBanner: the four verdicts', () => {
	it('renders no banner for the ok verdict, paired with a degraded verdict that DOES render one', () => {
		const { unmount } = render(StatusBanner, { props: { status: status({ verdict: 'ok' }) } });
		expect(screen.queryByTestId('status-banner-stale')).toBeNull();
		expect(screen.queryByTestId('status-banner-no-index')).toBeNull();
		expect(screen.queryByTestId('status-banner-indexing')).toBeNull();
		unmount();

		render(StatusBanner, { props: { status: status({ verdict: 'stale' }) } });
		expect(screen.getByTestId('status-banner-stale')).toBeTruthy();
	});

	it('renders no banner for the unknown verdict (transient / rejected-call state)', () => {
		render(StatusBanner, { props: { status: status({ verdict: 'unknown', commit: 'unknown' }) } });
		expect(screen.queryByTestId('status-banner-stale')).toBeNull();
		expect(screen.queryByTestId('status-banner-no-index')).toBeNull();
		expect(screen.queryByTestId('status-banner-indexing')).toBeNull();
	});

	it('the stale, no-index and indexing banners have pairwise-different text', () => {
		const { unmount: u1 } = render(StatusBanner, {
			props: { status: status({ verdict: 'stale' }) }
		});
		const staleText = screen.getByTestId('status-banner-stale').textContent;
		u1();

		const { unmount: u2 } = render(StatusBanner, {
			props: { status: status({ verdict: 'no-index' }) }
		});
		const noIndexText = screen.getByTestId('status-banner-no-index').textContent;
		u2();

		render(StatusBanner, { props: { status: status({ verdict: 'indexing' }) } });
		const indexingText = screen.getByTestId('status-banner-indexing').textContent;

		expect(staleText).not.toBe(noIndexText);
		expect(staleText).not.toBe(indexingText);
		expect(noIndexText).not.toBe(indexingText);
	});

	it('the no-index banner names the exact command to run', () => {
		render(StatusBanner, { props: { status: status({ verdict: 'no-index' }) } });
		expect(screen.getByTestId('status-banner-no-index').textContent).toContain('codegraph init');
	});

	it('the stale banner names the remedy', () => {
		render(StatusBanner, { props: { status: status({ verdict: 'stale' }) } });
		expect(screen.getByTestId('status-banner-stale').textContent).toContain('codegraph index');
	});

	it('the indexing banner names that indexing is in progress', () => {
		render(StatusBanner, { props: { status: status({ verdict: 'indexing' }) } });
		expect(screen.getByTestId('status-banner-indexing').textContent?.toLowerCase()).toContain(
			'indexing is in progress'
		);
	});
});

describe('SourcePane: per-call failures render named states, never a blank pane', () => {
	it('a not-found failure renders a named state naming the target that was not found', async () => {
		const target = await loadBrowseTarget(
			{ symbol: 'DoesNotExist', unknown: [] },
			failingClient(new ConnectError('symbol "DoesNotExist" not found', Code.NotFound))
		);
		render(SourcePane, { props: { state: target } });
		const el = screen.getByTestId('browse-failed-not-found');
		expect(el.textContent).toContain('DoesNotExist');
	});

	it('an invalid-argument failure renders a named invalid-input state carrying the server message', async () => {
		const target = await loadBrowseTarget(
			{ symbol: 'Foo', depth: 999, unknown: [] },
			failingClient(new ConnectError('depth must be between 0 and 10', Code.InvalidArgument))
		);
		render(SourcePane, { props: { state: target } });
		const el = screen.getByTestId('browse-failed-invalid-input');
		expect(el.textContent).toContain('depth must be between 0 and 10');
	});
});

describe('A stale banner and an in-view not-found state render TOGETHER; neither suppresses the other', () => {
	it('both testids are present when both conditions hold', async () => {
		render(StatusBanner, { props: { status: status({ verdict: 'stale' }) } });

		const target = await loadBrowseTarget(
			{ symbol: 'Ghost', unknown: [] },
			failingClient(new ConnectError('symbol "Ghost" not found', Code.NotFound))
		);
		render(SourcePane, { props: { state: target } });

		expect(screen.getByTestId('status-banner-stale')).toBeTruthy();
		expect(screen.getByTestId('browse-failed-not-found')).toBeTruthy();
	});
});

describe('SourcePane: the two source-absent messages differ between the stale and non-stale cases (D-03)', () => {
	it('renders the plain no-source message when the index is not stale', () => {
		render(SourcePane, { props: { state: singleDefStateNoSource(), indexStale: false } });
		expect(screen.queryByTestId('browse-no-source')).toBeTruthy();
		expect(screen.queryByTestId('browse-no-source-stale')).toBeNull();
	});

	it('renders the stale-specific unavailable message and names re-indexing when the index is stale', () => {
		render(SourcePane, { props: { state: singleDefStateNoSource(), indexStale: true } });
		const el = screen.getByTestId('browse-no-source-stale');
		expect(el.textContent).toContain('stale');
		expect(el.textContent).toContain('codegraph index');
		expect(screen.queryByTestId('browse-no-source')).toBeNull();
	});

	it('the two texts are different strings', () => {
		const { unmount } = render(SourcePane, {
			props: { state: singleDefStateNoSource(), indexStale: false }
		});
		const plain = screen.getByTestId('browse-no-source').textContent;
		unmount();

		render(SourcePane, { props: { state: singleDefStateNoSource(), indexStale: true } });
		const stale = screen.getByTestId('browse-no-source-stale').textContent;

		expect(plain).not.toBe(stale);
	});
});
