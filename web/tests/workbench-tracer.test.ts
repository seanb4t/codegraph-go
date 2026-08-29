// 04-01 Task 2 (the phase's tracer): the whole "open a Workbench URL, run
// Callers, see a sortable table, read a named failure kind" slice, driven
// end to end through the real workbench-url.ts / workbench-failure.ts /
// table-features.ts / DataTable.svelte / callers-columns.ts / +page.svelte
// modules with a stub Callers client — no real Connect transport. This
// file is written and observed RED BEFORE any of those modules exist
// (recorded in the SUMMARY), then made GREEN by building them, mirroring
// web/tests/browse-tracer.test.ts's own tracer discipline (03-04 Task 1).
//
// Tests 1-5 mount the real /workbench route component (+page.svelte)
// through the same $app/state + $app/navigation mocking convention
// web/tests/browse-page.test.ts established (tests/support/
// browse-page-state.svelte.ts's mockPage/resetMockPage singleton, reused
// here rather than duplicated — see that file's widened comment).
//
// Test 6 (status-gate quiescence, T-04-32) deliberately does NOT mount any
// component: it drives the real navigationIdentity + createStatusGate pair
// directly, exactly as web/tests/status.test.ts's own WR-04 test does for
// `q` — asserting the app's actual behavior through its real functions,
// never a hand-written identity string.
import { ConnectError, Code } from '@connectrpc/connect';
import { render, screen, fireEvent, waitFor, within } from '@testing-library/svelte';
import { describe, expect, it, vi } from 'vitest';

import type { CallersResponse, Location } from '$lib/gen/ui_pb';

import { mockPage, resetMockPage } from './support/browse-page-state.svelte';

// Both mock factories dynamically import the support module inside the
// factory body, deliberately: vi.mock calls are hoisted above this file's
// own imports, so a factory closing over the top-level `mockPage` binding
// would be closing over an import that (per hoisting) has not run yet —
// the same reasoning browse-page.test.ts's own comment records.
vi.mock('$app/state', async () => {
	const { mockPage } = await import('./support/browse-page-state.svelte');
	return { page: mockPage };
});

vi.mock('$app/navigation', async () => {
	const { mockPage } = await import('./support/browse-page-state.svelte');
	return {
		// replaceState here does exactly what SvelteKit's real client-side
		// replaceState does for this route's purposes: rewrite `page.url`
		// in place, which is what +page.svelte's own `$derived(params)`
		// reacts to. No real navigation/routing stack is mounted here.
		replaceState: (url: URL | string) => {
			mockPage.url = typeof url === 'string' ? new URL(url, mockPage.url) : new URL(url.href);
		}
	};
});

// $lib/client is mocked with a STABLE object whose `callers` method
// delegates to a mutable dispatcher, rather than re-mocking (and
// re-importing +page.svelte) per test: Svelte's runtime keeps
// module-level state, and `vi.resetModules()` between dynamic imports of
// the same component was tried and confirmed BROKEN — it loads a second
// copy of the `svelte` runtime module, so the freshly re-imported
// component's `$effect` calls run against a different runtime instance
// than the one `render()` (imported once, at the top of this file) is
// operating on, producing "`$effect` can only be used inside an effect"
// on every test after the first. +page.svelte is therefore imported
// exactly ONCE (below), and each test swaps `currentCallersImpl` before
// calling `render()` again — a fresh component instance per render call,
// same compiled module, same svelte runtime, no reset needed.
let currentCallersImpl: (req: { symbol: string; limit: number }) => Promise<CallersResponse> =
	() => Promise.reject(new Error('workbench-tracer.test.ts: no callers stub configured for this test'));

vi.doMock('$lib/client', () => ({
	uiClient: {
		callers: (req: { symbol: string; limit: number }) => currentCallersImpl(req)
	}
}));

const { default: WorkbenchPage } = await import('../src/routes/workbench/+page.svelte');

function loc(name: string, kind: string, filePath: string, startLine: number): Location {
	return { name, kind, filePath, startLine } as unknown as Location;
}

function callersResponse(symbol: string, callers: Location[]): CallersResponse {
	return { symbol, callers } as unknown as CallersResponse;
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

function mountWorkbench(callersImpl: (req: { symbol: string; limit: number }) => Promise<CallersResponse>) {
	currentCallersImpl = callersImpl;
	return render(WorkbenchPage, { context: new Map([['statusGate', okStatusGate()]]) });
}

describe('workbench tracer: a Workbench URL runs Callers and renders a sortable table', () => {
	it('parseWorkbenchParams -> Callers rpc -> DataTable renders all three rows, recording exactly one call with the URL symbol and limit', async () => {
		const calls: Array<{ symbol: string; limit: number }> = [];
		const response = callersResponse('Engine.Status', [
			loc('Alpha', 'func', 'a.go', 10),
			loc('Beta', 'func', 'b.go', 20),
			loc('Gamma', 'func', 'c.go', 30)
		]);

		resetMockPage('http://localhost/workbench?mode=callers&symbol=Engine.Status&limit=25');
		mountWorkbench((req) => {
			calls.push({ symbol: req.symbol, limit: req.limit });
			return Promise.resolve(response);
		});

		await waitFor(() => expect(screen.getByRole('table')).toBeInTheDocument());
		expect(screen.getByText('Alpha')).toBeInTheDocument();
		expect(screen.getByText('Beta')).toBeInTheDocument();
		expect(screen.getByText('Gamma')).toBeInTheDocument();

		expect(calls).toEqual([{ symbol: 'Engine.Status', limit: 25 }]);
	});

	it('WRK-04: clicking the name column header sorts ascending, clicking again reverses to descending', async () => {
		const response = callersResponse('Engine.Status', [
			loc('Beta', 'func', 'b.go', 20),
			loc('Gamma', 'func', 'c.go', 30),
			loc('Alpha', 'func', 'a.go', 10)
		]);

		resetMockPage('http://localhost/workbench?mode=callers&symbol=Engine.Status&limit=25');
		mountWorkbench(() => Promise.resolve(response));

		const table = await screen.findByRole('table');
		const nameHeader = screen.getByRole('button', { name: /Name/ });

		await fireEvent.click(nameHeader);
		await waitFor(() => {
			const rows = within(table).getAllByRole('row').slice(1);
			expect(rows.map((r) => r.textContent)).toEqual([
				expect.stringContaining('Alpha'),
				expect.stringContaining('Beta'),
				expect.stringContaining('Gamma')
			]);
		});

		await fireEvent.click(nameHeader);
		await waitFor(() => {
			const rows = within(table).getAllByRole('row').slice(1);
			expect(rows.map((r) => r.textContent)).toEqual([
				expect.stringContaining('Gamma'),
				expect.stringContaining('Beta'),
				expect.stringContaining('Alpha')
			]);
		});
	});

	it('D-07: a limit above MaxLimit (99999) reaches the stub client unchanged — the UI applies no client-side bound', async () => {
		const calls: Array<{ symbol: string; limit: number }> = [];
		resetMockPage('http://localhost/workbench?mode=callers&symbol=Engine.Status&limit=99999');
		mountWorkbench((req) => {
			calls.push({ symbol: req.symbol, limit: req.limit });
			return Promise.resolve(callersResponse('Engine.Status', []));
		});

		await waitFor(() => expect(calls).toEqual([{ symbol: 'Engine.Status', limit: 99999 }]));
	});

	it('an empty Callers result renders one visible empty-state row, not a bare table', async () => {
		resetMockPage('http://localhost/workbench?mode=callers&symbol=Engine.Status&limit=25');
		mountWorkbench(() => Promise.resolve(callersResponse('Engine.Status', [])));

		await waitFor(() => expect(screen.getByRole('table')).toBeInTheDocument());
		expect(screen.getByText(/no callers/i)).toBeInTheDocument();
	});

	it('WRK-04 criterion 3: a NotFound rejection renders the named not-found failure state', async () => {
		resetMockPage('http://localhost/workbench?mode=callers&symbol=Missing&limit=25');
		mountWorkbench(() => Promise.reject(new ConnectError('no such symbol', Code.NotFound)));

		await waitFor(() =>
			expect(screen.getByTestId('workbench-failure-not-found')).toBeInTheDocument()
		);
	});
});

describe('status-gate quiescence: Workbench control changes mint no new navigation identity (T-04-32)', () => {
	it('records exactly one getStatus call across a Workbench limit change, paired with a Browse symbol change that DOES refetch', async () => {
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

		const initialUrl = new URL('http://localhost/workbench?mode=callers&symbol=Engine.Status&limit=25');
		const gate = createStatusGate(client, navigationIdentity(initialUrl));
		await vi.waitFor(() => expect(getStatus).toHaveBeenCalledTimes(1));

		// Simulate "typing a new limit": the SAME Workbench params,
		// re-serialized through the real grammar with a different limit,
		// re-identified through the real route-scoped navigationIdentity.
		const nextParams = parseWorkbenchParams(initialUrl.searchParams);
		nextParams.limit = 50;
		const nextUrl = new URL(initialUrl.href);
		nextUrl.search = serializeWorkbenchParams(nextParams).toString();

		gate.notifyNavigated(navigationIdentity(nextUrl));
		expect(getStatus).toHaveBeenCalledTimes(1);

		// Positive control, paired in the same test (rule 84d1gfpywd): the
		// SAME kind of parameter change under /browse — a genuine distinct
		// Browse view — still mints a new identity and DOES refetch, so
		// the exclusion above is proven scoped to /workbench, not a gate
		// that has simply stopped fetching entirely.
		gate.notifyNavigated(navigationIdentity(new URL('http://localhost/browse?symbol=Foo')));
		await vi.waitFor(() => expect(getStatus).toHaveBeenCalledTimes(2));

		gate.notifyNavigated(navigationIdentity(new URL('http://localhost/browse?symbol=Bar')));
		await vi.waitFor(() => expect(getStatus).toHaveBeenCalledTimes(3));
	});
});
