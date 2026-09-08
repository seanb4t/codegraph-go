// 04-04 Task 2: the Impact tab, generalized off the tracer's shape
// (workbench-tracer.test.ts, 04-01 Task 2) — same mounting harness, same
// mockPage/replaceState convention, but driving Impact through the new
// AnalysisPanel.svelte shell instead of Callers' inlined state machine.
//
// This file is written and observed RED before Impact is wired through
// +page.svelte/AnalysisPanel.svelte (recorded in the SUMMARY), then made
// GREEN by the implementation.
//
// The "route component is not re-created" assertion (WRK-01) uses DOM
// node reference identity on the always-rendered `<h1>Workbench</h1>`
// heading rather than an injected onMount spy: that heading sits outside
// every {#if}/{:else if} branch in +page.svelte's template, so Svelte
// can only produce a NEW DOM node for it if the enclosing component
// itself were torn down and recreated (the property WRK-01 actually
// cares about) — a real remount is the only way this reference would
// change between two renders driven by the SAME `render()` call. This
// is the "module-scoped mount counter" alternative named in
// 04-04-PLAN.md's Task 2 action, chosen over an onMount spy specifically
// so no test-only instrumentation is added to production code.
import { render, screen, fireEvent, waitFor, within } from '@testing-library/svelte';
import { describe, expect, it, vi, beforeEach } from 'vitest';

import type { ImpactResponse, Location } from '$lib/gen/ui_pb';

import { mockPage, resetMockPage } from './support/browse-page-state.svelte';

// Both mock factories dynamically import the support module inside the
// factory body, deliberately: vi.mock calls are hoisted above this
// file's own imports (workbench-tracer.test.ts's own comment explains
// why a top-level closure would be closing over an import that has not
// run yet).
vi.mock('$app/state', async () => {
	const { mockPage } = await import('./support/browse-page-state.svelte');
	return { page: mockPage };
});

vi.mock('$app/navigation', async () => {
	const { mockPage } = await import('./support/browse-page-state.svelte');
	return {
		replaceState: (url: URL | string) => {
			mockPage.url = typeof url === 'string' ? new URL(url, mockPage.url) : new URL(url.href);
		},
		// goto is a spy, never wired to any real navigation: WRK-01
		// forbids the route from ever calling it for a depth/limit/mode
		// edit, and this lets that be asserted directly rather than only
		// inferred from the static grep in 04-04-PLAN.md's acceptance
		// criteria.
		goto: vi.fn()
	};
});

let currentImpactImpl: (req: { symbol: string; depth: number }) => Promise<ImpactResponse> = () =>
	Promise.reject(new Error('workbench-impact.test.ts: no impact stub configured for this test'));

// callers/callees are stubbed too (rejecting by default) so importing
// +page.svelte never throws even though this file only drives Impact —
// mirrors workbench-tracer.test.ts's single-method-under-test mock shape.
vi.doMock('$lib/client', () => ({
	uiClient: {
		impact: (req: { symbol: string; depth: number }) => currentImpactImpl(req),
		callers: () => Promise.reject(new Error('not used by this test file')),
		callees: () => Promise.reject(new Error('not used by this test file'))
	}
}));

const { default: WorkbenchPage } = await import('../src/routes/workbench/+page.svelte');

function loc(name: string, kind: string, filePath: string, startLine: number): Location {
	return { name, kind, filePath, startLine } as unknown as Location;
}

function impactResponse(
	symbol: string,
	depth: number,
	nodeCount: number,
	edgeCount: number,
	affected: Location[]
): ImpactResponse {
	return { symbol, depth, nodeCount, edgeCount, affected } as unknown as ImpactResponse;
}

function okStatusGate() {
	return {
		subscribe(run: (s: { verdict: string; commit: string }) => void) {
			run({ verdict: 'ok', commit: 'known' });
			return () => {};
		},
		notifyNavigated: () => {}
	};
}

function mountWorkbench(impactImpl: (req: { symbol: string; depth: number }) => Promise<ImpactResponse>) {
	currentImpactImpl = impactImpl;
	return render(WorkbenchPage, { context: new Map([['statusGate', okStatusGate()]]) });
}

beforeEach(async () => {
	const { goto } = await import('$app/navigation');
	vi.mocked(goto).mockClear();
});

describe('workbench Impact tab: a Workbench URL runs Impact and renders a sortable table with a header summary', () => {
	it('parseWorkbenchParams -> Impact rpc -> DataTable renders all three rows, recording exactly one call with the URL symbol and depth', async () => {
		const calls: Array<{ symbol: string; depth: number }> = [];
		const response = impactResponse('Engine.Status', 2, 7, 9, [
			loc('Alpha', 'func', 'a.go', 10),
			loc('Beta', 'func', 'b.go', 20),
			loc('Gamma', 'func', 'c.go', 30)
		]);

		resetMockPage('http://localhost/workbench?mode=impact&symbol=Engine.Status&depth=2');
		mountWorkbench((req) => {
			calls.push({ symbol: req.symbol, depth: req.depth });
			return Promise.resolve(response);
		});

		await waitFor(() => expect(screen.getByRole('table')).toBeInTheDocument());
		expect(screen.getByText('Alpha')).toBeInTheDocument();
		expect(screen.getByText('Beta')).toBeInTheDocument();
		expect(screen.getByText('Gamma')).toBeInTheDocument();

		expect(calls).toEqual([{ symbol: 'Engine.Status', depth: 2 }]);
	});

	it('D-06: node_count/edge_count render in an element that PRECEDES the table in document order', async () => {
		const response = impactResponse('Engine.Status', 2, 7, 9, [loc('Alpha', 'func', 'a.go', 10)]);

		resetMockPage('http://localhost/workbench?mode=impact&symbol=Engine.Status&depth=2');
		mountWorkbench(() => Promise.resolve(response));

		const table = await screen.findByRole('table');
		const summaryEl = await screen.findByTestId('workbench-summary');

		// Node.compareDocumentPosition returns a bitmask; the
		// DOCUMENT_POSITION_FOLLOWING bit set on `table` (as observed from
		// `summaryEl`) means summaryEl precedes table in document order.
		const position = summaryEl.compareDocumentPosition(table);
		expect(position & Node.DOCUMENT_POSITION_FOLLOWING).toBe(Node.DOCUMENT_POSITION_FOLLOWING);
	});

	it('D-06: node_count/edge_count do NOT appear as column headers — the rendered <th> set is exactly the four Location columns', async () => {
		const response = impactResponse('Engine.Status', 2, 7, 9, [loc('Alpha', 'func', 'a.go', 10)]);

		resetMockPage('http://localhost/workbench?mode=impact&symbol=Engine.Status&depth=2');
		mountWorkbench(() => Promise.resolve(response));

		const table = await screen.findByRole('table');
		const headers = within(table)
			.getAllByRole('columnheader')
			.map((h) => h.textContent?.trim());

		expect(new Set(headers)).toEqual(new Set(['Name', 'Kind', 'File', 'Line']));
	});

	it('the summary values reach the header snippet THROUGH the panel result type, recording exactly one Impact call', async () => {
		const calls: Array<{ symbol: string; depth: number }> = [];
		const response = impactResponse('Engine.Status', 2, 7, 9, [loc('Alpha', 'func', 'a.go', 10)]);

		resetMockPage('http://localhost/workbench?mode=impact&symbol=Engine.Status&depth=2');
		mountWorkbench((req) => {
			calls.push({ symbol: req.symbol, depth: req.depth });
			return Promise.resolve(response);
		});

		await waitFor(() => expect(screen.getByTestId('workbench-summary')).toBeInTheDocument());
		expect(screen.getByTestId('workbench-impact-node-count')).toHaveTextContent('7');
		expect(screen.getByTestId('workbench-impact-edge-count')).toHaveTextContent('9');
		expect(calls).toHaveLength(1);
	});

	it('WRK-01: moving the depth control from 2 to 4 re-runs Impact and replaces the rows, without leaving the page', async () => {
		const calls: Array<{ symbol: string; depth: number }> = [];
		resetMockPage('http://localhost/workbench?mode=impact&symbol=Engine.Status&depth=2');
		mountWorkbench((req) => {
			calls.push({ symbol: req.symbol, depth: req.depth });
			return Promise.resolve(
				impactResponse('Engine.Status', req.depth, req.depth * 10, req.depth * 5, [
					loc(`Node-depth-${req.depth}`, 'func', 'x.go', req.depth)
				])
			);
		});

		await waitFor(() => expect(screen.getByText('Node-depth-2')).toBeInTheDocument());
		const headingBefore = screen.getByRole('heading', { name: /workbench/i });

		const depthInput = screen.getByTestId('workbench-depth-input');
		await fireEvent.input(depthInput, { target: { value: '4' } });

		await waitFor(() => expect(screen.getByText('Node-depth-4')).toBeInTheDocument());
		expect(screen.queryByText('Node-depth-2')).not.toBeInTheDocument();
		expect(calls).toEqual([
			{ symbol: 'Engine.Status', depth: 2 },
			{ symbol: 'Engine.Status', depth: 4 }
		]);

		// (a) the route component instance is not re-created.
		const headingAfter = screen.getByRole('heading', { name: /workbench/i });
		expect(headingAfter).toBe(headingBefore);

		// (b) no navigation was dispatched.
		const { goto } = await import('$app/navigation');
		expect(goto).not.toHaveBeenCalled();
	});

	it('T-04-32: moving the depth control issues NO additional GetStatus call — the shared status gate is quiescent', async () => {
		const { navigationIdentity, createStatusGate } = await import('$lib/status');
		const { parseWorkbenchParams, serializeWorkbenchParams } = await import('$lib/workbench-url');

		const getStatus = vi.fn(() =>
			Promise.resolve({
				initialized: true,
				version: '',
				nodeCount: 0,
				edgeCount: 0,
				fileCount: 0,
				stale: false,
				commitSha: '',
				storeExists: true,
				indexingInProgress: false
				// eslint-disable-next-line @typescript-eslint/no-explicit-any
			} as any)
		);
		const client = { getStatus };

		const initialUrl = new URL('http://localhost/workbench?mode=impact&symbol=Engine.Status&depth=2');
		const gate = createStatusGate(client, navigationIdentity(initialUrl));
		await vi.waitFor(() => expect(getStatus).toHaveBeenCalledTimes(1));

		const nextParams = parseWorkbenchParams(initialUrl.searchParams);
		nextParams.depth = 4;
		const nextUrl = new URL(initialUrl.href);
		nextUrl.search = serializeWorkbenchParams(nextParams).toString();

		gate.notifyNavigated(navigationIdentity(nextUrl));
		expect(getStatus).toHaveBeenCalledTimes(1);

		// Positive control (rule 84d1gfpywd): the SAME kind of parameter
		// change under /browse still mints a new identity and DOES
		// refetch, proving the exclusion above is scoped to /workbench.
		gate.notifyNavigated(navigationIdentity(new URL('http://localhost/browse?symbol=Foo')));
		await vi.waitFor(() => expect(getStatus).toHaveBeenCalledTimes(2));
	});

	it('D-07: a depth above MaxDepth (9999) reaches the stub client unchanged — the UI applies no client-side bound', async () => {
		const calls: Array<{ symbol: string; depth: number }> = [];
		resetMockPage('http://localhost/workbench?mode=impact&symbol=Engine.Status&depth=9999');
		mountWorkbench((req) => {
			calls.push({ symbol: req.symbol, depth: req.depth });
			return Promise.resolve(impactResponse('Engine.Status', req.depth, 0, 0, []));
		});

		await waitFor(() => expect(calls).toEqual([{ symbol: 'Engine.Status', depth: 9999 }]));
	});

	it('tab selection: ?mode=callees selects Callees and not Impact', async () => {
		resetMockPage('http://localhost/workbench?mode=callees&symbol=Engine.Status');
		mountWorkbench(() => Promise.resolve(impactResponse('Engine.Status', 0, 0, 0, [])));

		expect(screen.getByRole('tab', { name: /callees/i })).toHaveAttribute('aria-selected', 'true');
		expect(screen.getByRole('tab', { name: /^impact$/i })).toHaveAttribute('aria-selected', 'false');
	});

	it('tab selection: an unrecognized ?mode=zzz falls back to Impact', async () => {
		resetMockPage('http://localhost/workbench?mode=zzz&symbol=Engine.Status');
		mountWorkbench(() => Promise.resolve(impactResponse('Engine.Status', 0, 0, 0, [])));

		await waitFor(() =>
			expect(screen.getByRole('tab', { name: /^impact$/i })).toHaveAttribute('aria-selected', 'true')
		);
	});

	it('selecting a different tab writes mode into the URL through serializeWorkbenchParams and preserves the other parameters', async () => {
		resetMockPage('http://localhost/workbench?mode=impact&symbol=Engine.Status&depth=3');
		mountWorkbench(() => Promise.resolve(impactResponse('Engine.Status', 3, 0, 0, [])));

		const calleesTab = screen.getByRole('tab', { name: /callees/i });
		await fireEvent.click(calleesTab);

		await waitFor(() => expect(mockPage.url.searchParams.get('mode')).toBe('callees'));
		expect(mockPage.url.searchParams.get('symbol')).toBe('Engine.Status');
		expect(mockPage.url.searchParams.get('depth')).toBe('3');
	});
});
