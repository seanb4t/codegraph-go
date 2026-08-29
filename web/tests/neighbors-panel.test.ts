// neighbors-panel.test.ts — 03-07 Task 3: NeighborsPanel.svelte's own
// tests, written and run RED before the component exists (recorded in
// the SUMMARY), then made GREEN.
import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';

import NeighborsPanel from '$lib/components/browse/NeighborsPanel.svelte';
import { NAV_INTENT } from '$lib/browse-nav';
import type { BlastRadiusState } from '$lib/browse-state';
import type { Node, Location } from '$lib/gen/ui_pb';

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

function location(name: string): Location {
	return { name, kind: 'func', filePath: `${name}.go`, startLine: 3 } as unknown as Location;
}

const IDLE_BLAST: BlastRadiusState = { kind: 'idle' };

describe('NeighborsPanel: three labelled regions, entries in supplied order', () => {
	it('shows Callers, Callees and Blast Radius, each with name/kind/filePath, in the order supplied', () => {
		const onNavigate = vi.fn();
		render(NeighborsPanel, {
			props: {
				calledBy: [node('CallerA'), node('CallerB')],
				calls: [node('CalleeA')],
				blastRadius: {
					kind: 'loaded',
					depth: 2,
					nodeCount: 1,
					edgeCount: 1,
					affected: [location('Affected1')]
				},
				depth: 2,
				onNavigate
			}
		});

		expect(screen.getByTestId('neighbors-callers')).toBeInTheDocument();
		expect(screen.getByTestId('neighbors-callees')).toBeInTheDocument();
		expect(screen.getByTestId('neighbors-blast-radius')).toBeInTheDocument();

		const callerButtons = screen
			.getByTestId('neighbors-callers')
			.querySelectorAll('[data-testid^="neighbor-entry-"]');
		expect(Array.from(callerButtons).map((el) => el.textContent)).toEqual([
			expect.stringContaining('CallerA'),
			expect.stringContaining('CallerB')
		]);
		expect(Array.from(callerButtons)[0]?.textContent).toContain('func');
		expect(Array.from(callerButtons)[0]?.textContent).toContain('CallerA.go');

		const blastEntries = screen
			.getByTestId('neighbors-blast-radius')
			.querySelectorAll('[data-testid^="neighbor-entry-"]');
		expect(blastEntries).toHaveLength(1);
		expect(blastEntries[0]?.textContent).toContain('Affected1');
	});
});

describe('NeighborsPanel: click-through and depth-change use DIFFERENT intents', () => {
	it('clicking an entry invokes onNavigate with that targets symbol/file/line and the NAVIGATE intent', async () => {
		const onNavigate = vi.fn();
		render(NeighborsPanel, {
			props: {
				calledBy: [node('CallerA')],
				calls: [],
				blastRadius: IDLE_BLAST,
				depth: undefined,
				onNavigate
			}
		});
		const button = screen
			.getByTestId('neighbors-callers')
			.querySelector('[data-testid^="neighbor-entry-"]') as HTMLElement;
		await fireEvent.click(button);

		expect(onNavigate).toHaveBeenCalledWith(
			{ symbol: 'CallerA', file: 'CallerA.go', line: 1 },
			NAV_INTENT.NAVIGATE
		);
	});

	it('changing the depth control invokes onNavigate with the REFINE intent', async () => {
		const onNavigate = vi.fn();
		render(NeighborsPanel, {
			props: { calledBy: [], calls: [], blastRadius: IDLE_BLAST, depth: 2, onNavigate }
		});
		const input = screen.getByTestId('neighbors-depth-input');
		await fireEvent.change(input, { target: { value: '4' } });

		expect(onNavigate).toHaveBeenCalledWith({ depth: 4 }, NAV_INTENT.REFINE);
	});

	it('the click-through intent and the depth-change intent are demonstrably different values', () => {
		expect(NAV_INTENT.NAVIGATE).not.toBe(NAV_INTENT.REFINE);
	});
});

describe('NeighborsPanel: empty and failed states are distinguishable, never an absent region', () => {
	it('an empty callers list renders an explicit "no callers" line rather than an absent region', () => {
		render(NeighborsPanel, {
			props: {
				calledBy: [],
				calls: [node('X')],
				blastRadius: IDLE_BLAST,
				depth: undefined,
				onNavigate: vi.fn()
			}
		});
		expect(screen.getByTestId('neighbors-callers')).toBeInTheDocument();
		expect(screen.getByTestId('neighbors-callers-empty')).toBeInTheDocument();
	});

	it('a failed blast-radius load renders the classified failure text, distinct from the empty text', () => {
		const { unmount } = render(NeighborsPanel, {
			props: {
				calledBy: [],
				calls: [],
				blastRadius: { kind: 'loaded', depth: 2, nodeCount: 0, edgeCount: 0, affected: [] },
				depth: 2,
				onNavigate: vi.fn()
			}
		});
		const emptyText = screen.getByTestId('neighbors-blast-empty').textContent;
		unmount();

		render(NeighborsPanel, {
			props: {
				calledBy: [],
				calls: [],
				blastRadius: { kind: 'failed', failure: { kind: 'unknown', message: 'boom' } },
				depth: 2,
				onNavigate: vi.fn()
			}
		});
		const failedText = screen.getByTestId('neighbors-blast-failed').textContent;

		expect(emptyText).toBeTruthy();
		expect(failedText).toBeTruthy();
		expect(emptyText).not.toBe(failedText);
	});
});
