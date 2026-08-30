// 04-04 Task 3: the Callers/Callees limit controls (WRK-03) and the
// four-way rendered failure distinctness WRK-04 criterion 3 asks for —
// generalizing the tracer's (04-01) Callers-only coverage to both tabs
// and adding the assertions 04-VALIDATION.md explicitly moves OUT of
// manual verification: a Set-of-size-4 over four rendered failure
// strings (not four independent "renders something" checks, which pass
// vacuously if all four collapse to one message), and the sorted row
// ORDER (not merely a toggled indicator class).
//
// Mounting/mocking conventions match workbench-impact.test.ts and
// workbench-tracer.test.ts exactly (mockPage/resetMockPage singleton,
// $app/navigation's replaceState+goto mock, a per-file $lib/client mock
// whose methods delegate to swappable dispatcher variables).
import { ConnectError, Code } from '@connectrpc/connect';
import { render, screen, fireEvent, waitFor, within } from '@testing-library/svelte';
import { describe, expect, it, vi, beforeEach } from 'vitest';

import type { CallersResponse, CalleesResponse, Location } from '$lib/gen/ui_pb';

import { mockPage, resetMockPage } from './support/browse-page-state.svelte';

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
		goto: vi.fn()
	};
});

let currentCallersImpl: (
	req: { symbol: string; limit: number },
	opts?: { signal?: AbortSignal }
) => Promise<CallersResponse> = () =>
	Promise.reject(new Error('workbench-callers-callees.test.ts: no callers stub configured'));
let currentCalleesImpl: (
	req: { symbol: string; limit: number },
	opts?: { signal?: AbortSignal }
) => Promise<CalleesResponse> = () =>
	Promise.reject(new Error('workbench-callers-callees.test.ts: no callees stub configured'));

vi.doMock('$lib/client', () => ({
	uiClient: {
		impact: () => Promise.reject(new Error('not used by this test file')),
		callers: (req: { symbol: string; limit: number }, opts?: { signal?: AbortSignal }) =>
			currentCallersImpl(req, opts),
		callees: (req: { symbol: string; limit: number }, opts?: { signal?: AbortSignal }) =>
			currentCalleesImpl(req, opts)
	}
}));

const { default: WorkbenchPage } = await import('../src/routes/workbench/+page.svelte');

function loc(name: string, kind: string, filePath: string, startLine: number): Location {
	return { name, kind, filePath, startLine } as unknown as Location;
}

function callersResponse(symbol: string, callers: Location[]): CallersResponse {
	return { symbol, callers } as unknown as CallersResponse;
}

function calleesResponse(symbol: string, callees: Location[]): CalleesResponse {
	return { symbol, callees } as unknown as CalleesResponse;
}

function statusGate(verdict: 'ok' | 'no-index') {
	return {
		subscribe(run: (s: { verdict: string; commit: string }) => void) {
			run({ verdict, commit: verdict === 'ok' ? 'known' : 'unknown' });
			return () => {};
		},
		notifyNavigated: () => {}
	};
}

function mountWorkbench(verdict: 'ok' | 'no-index' = 'ok') {
	return render(WorkbenchPage, { context: new Map([['statusGate', statusGate(verdict)]]) });
}

beforeEach(async () => {
	const { goto } = await import('$app/navigation');
	vi.mocked(goto).mockClear();
});

describe('Callers/Callees limit controls (WRK-03)', () => {
	it('Callers: ?mode=callers&symbol=X&limit=5 calls the Callers stub exactly once with limit 5', async () => {
		const calls: Array<{ symbol: string; limit: number }> = [];
		currentCallersImpl = (req) => {
			calls.push(req);
			return Promise.resolve(callersResponse('X', [loc('Alpha', 'func', 'a.go', 10)]));
		};

		resetMockPage('http://localhost/workbench?mode=callers&symbol=X&limit=5');
		mountWorkbench();

		await waitFor(() => expect(screen.getByText('Alpha')).toBeInTheDocument());
		expect(calls).toEqual([{ symbol: 'X', limit: 5 }]);
	});

	it('Callers: moving the limit control from 5 to 50 re-runs Callers and replaces the rows, without leaving the page', async () => {
		const calls: Array<{ symbol: string; limit: number }> = [];
		currentCallersImpl = (req) => {
			calls.push(req);
			return Promise.resolve(callersResponse('X', [loc(`Row-limit-${req.limit}`, 'func', 'a.go', 1)]));
		};

		resetMockPage('http://localhost/workbench?mode=callers&symbol=X&limit=5');
		mountWorkbench();

		await waitFor(() => expect(screen.getByText('Row-limit-5')).toBeInTheDocument());
		const headingBefore = screen.getByRole('heading', { name: /workbench/i });

		await fireEvent.input(screen.getByTestId('workbench-limit-input'), { target: { value: '50' } });

		await waitFor(() => expect(screen.getByText('Row-limit-50')).toBeInTheDocument());
		expect(screen.queryByText('Row-limit-5')).not.toBeInTheDocument();
		expect(calls).toEqual([
			{ symbol: 'X', limit: 5 },
			{ symbol: 'X', limit: 50 }
		]);

		expect(screen.getByRole('heading', { name: /workbench/i })).toBe(headingBefore);
		const { goto } = await import('$app/navigation');
		expect(goto).not.toHaveBeenCalled();
	});

	it('T-04-32: moving the Callers limit control issues NO additional GetStatus call (a separate ROUTE_LOCAL_PARAMS entry from depth)', async () => {
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

		const initialUrl = new URL('http://localhost/workbench?mode=callers&symbol=X&limit=5');
		const gate = createStatusGate(client, navigationIdentity(initialUrl));
		await vi.waitFor(() => expect(getStatus).toHaveBeenCalledTimes(1));

		const nextParams = parseWorkbenchParams(initialUrl.searchParams);
		nextParams.limit = 50;
		const nextUrl = new URL(initialUrl.href);
		nextUrl.search = serializeWorkbenchParams(nextParams).toString();

		gate.notifyNavigated(navigationIdentity(nextUrl));
		expect(getStatus).toHaveBeenCalledTimes(1);

		// Positive control (rule 84d1gfpywd), same as Task 2's depth variant.
		gate.notifyNavigated(navigationIdentity(new URL('http://localhost/browse?symbol=Foo')));
		await vi.waitFor(() => expect(getStatus).toHaveBeenCalledTimes(2));
	});

	it('Callees: ?mode=callees&symbol=X&limit=5 calls the Callees stub exactly once — the Callers stub is never called (cross-tab isolation)', async () => {
		const calleesCalls: Array<{ symbol: string; limit: number }> = [];
		const callersCalls: Array<{ symbol: string; limit: number }> = [];
		currentCalleesImpl = (req) => {
			calleesCalls.push(req);
			return Promise.resolve(calleesResponse('X', [loc('Beta', 'func', 'b.go', 20)]));
		};
		currentCallersImpl = (req) => {
			callersCalls.push(req);
			return Promise.resolve(callersResponse('X', []));
		};

		resetMockPage('http://localhost/workbench?mode=callees&symbol=X&limit=5');
		mountWorkbench();

		await waitFor(() => expect(screen.getByText('Beta')).toBeInTheDocument());
		expect(calleesCalls).toEqual([{ symbol: 'X', limit: 5 }]);
		expect(callersCalls).toHaveLength(0);
	});

	it('Callers: driving the Callers tab records zero calls on the Callees stub (cross-tab isolation, the other direction)', async () => {
		const callersCalls: Array<{ symbol: string; limit: number }> = [];
		const calleesCalls: Array<{ symbol: string; limit: number }> = [];
		currentCallersImpl = (req) => {
			callersCalls.push(req);
			return Promise.resolve(callersResponse('X', [loc('Gamma', 'func', 'c.go', 30)]));
		};
		currentCalleesImpl = (req) => {
			calleesCalls.push(req);
			return Promise.resolve(calleesResponse('X', []));
		};

		resetMockPage('http://localhost/workbench?mode=callers&symbol=X&limit=5');
		mountWorkbench();

		await waitFor(() => expect(screen.getByText('Gamma')).toBeInTheDocument());
		expect(callersCalls).toEqual([{ symbol: 'X', limit: 5 }]);
		expect(calleesCalls).toHaveLength(0);
	});

	it('Callees: D-07 a limit above MaxLimit (99999) reaches the stub client unchanged', async () => {
		const calls: Array<{ symbol: string; limit: number }> = [];
		currentCalleesImpl = (req) => {
			calls.push(req);
			return Promise.resolve(calleesResponse('X', []));
		};

		resetMockPage('http://localhost/workbench?mode=callees&symbol=X&limit=99999');
		mountWorkbench();

		await waitFor(() => expect(calls).toEqual([{ symbol: 'X', limit: 99999 }]));
	});

	it('Callees: WRK-04 clicking the Name column header sorts ascending, clicking again reverses to descending', async () => {
		currentCalleesImpl = () =>
			Promise.resolve(
				calleesResponse('X', [
					loc('Beta', 'func', 'b.go', 20),
					loc('Gamma', 'func', 'c.go', 30),
					loc('Alpha', 'func', 'a.go', 10)
				])
			);

		resetMockPage('http://localhost/workbench?mode=callees&symbol=X&limit=25');
		mountWorkbench();

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

	it('WRK-04 criterion 3: four rejections through the SAME tab render four rendered strings whose SET has size four, with four distinct data-testids', async () => {
		const renderedTitles: string[] = [];
		const renderedTestIds: string[] = [];

		async function driveOneFailure(verdict: 'ok' | 'no-index', err: ConnectError): Promise<void> {
			currentCallersImpl = () => Promise.reject(err);
			resetMockPage('http://localhost/workbench?mode=callers&symbol=X&limit=25');
			const result = mountWorkbench(verdict);

			const failureEl = await screen.findByTestId(/^workbench-failure-/);
			renderedTestIds.push(failureEl.getAttribute('data-testid')!);
			// The failure title is the FIRST <p> inside the failure element
			// (AnalysisPanel.svelte's markup: title, then detail) — read
			// directly off the DOM rather than through an RTL text query,
			// since two <p> siblings both match a bare tag-name matcher.
			const titleEl = failureEl.querySelector('p');
			renderedTitles.push(titleEl?.textContent ?? failureEl.textContent ?? '');

			result.unmount();
		}

		// 1. NotFound with an 'ok' index verdict -> 'not-found'.
		await driveOneFailure('ok', new ConnectError('no such symbol', Code.NotFound));
		// 2. NotFound with a 'no-index' verdict -> 'index-stale' (the SAME kind
		//    an 'indexing' rejection gets — describeWorkbenchFailure's own
		//    documented behavior).
		await driveOneFailure('no-index', new ConnectError('no such symbol', Code.NotFound));
		// 3. InvalidArgument -> 'invalid-input'.
		await driveOneFailure('ok', new ConnectError('bad input', Code.InvalidArgument));
		// 4. Unavailable carrying NO typed detail -> 'server-error' (classifyRpcError's
		//    bare-Unavailable-is-'unknown' branch, rpc-errors.ts:54).
		await driveOneFailure('ok', new ConnectError('unavailable', Code.Unavailable));

		expect(new Set(renderedTestIds).size).toBe(4);
		const collapsed = new Set(renderedTitles);
		expect(collapsed.size, `collapsed pair(s) found in: ${JSON.stringify(renderedTitles)}`).toBe(4);
	});

	it('CR-01 regression: clearing the Symbol field while a request is in flight returns to idle, not a false "Something went wrong" failure', async () => {
		// The stub never settles on its own — it only rejects once its
		// AbortSignal fires, mirroring connect-web's real behavior when
		// AnalysisPanel's effect cleanup calls `controller.abort()`
		// (ConnectError with Code.Canceled). This reproduces the exact
		// `run → undefined` transition CR-01 describes: an in-flight
		// request aborted by clearing the input, not superseded by a new
		// request.
		currentCallersImpl = (_req, opts) =>
			new Promise<CallersResponse>((_resolve, reject) => {
				opts?.signal?.addEventListener('abort', () => {
					reject(new ConnectError('This operation was aborted', Code.Canceled));
				});
			});

		resetMockPage('http://localhost/workbench?mode=callers&symbol=X&limit=5');
		mountWorkbench();

		await waitFor(() => expect(screen.getByTestId('workbench-loading')).toBeInTheDocument());

		await fireEvent.input(screen.getByTestId('workbench-symbol-input'), {
			target: { value: '' }
		});

		await waitFor(() =>
			expect(screen.getByText('Enter a symbol to run this analysis.')).toBeInTheDocument()
		);
		expect(screen.queryByTestId(/^workbench-failure-/)).not.toBeInTheDocument();
	});
});
