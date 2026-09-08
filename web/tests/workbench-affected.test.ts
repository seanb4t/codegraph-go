// workbench-affected.test.ts covers WRK-02 (04-06) in two halves:
//   - FilePicker.svelte in isolation (Task 2) — search, add, de-duplicate,
//     remove, and the URL round-trip in both directions.
//   - The Affected tab wired into the Workbench route (Task 3) — appended
//     below the picker half.
//
// jsdom does not implement scrollIntoView; the vendored Command primitive
// calls it as a side effect of moving selection (search-panel.test.ts's
// own precedent). Stubbed once here, globally for this file.
if (!Element.prototype.scrollIntoView) {
	Element.prototype.scrollIntoView = () => {};
}

import { render, screen, fireEvent, waitFor, within } from '@testing-library/svelte';
import { describe, expect, it, vi, beforeEach } from 'vitest';

import FilePicker from '$lib/components/workbench/FilePicker.svelte';
import type { FilesClient } from '$lib/file-search';
import type { AffectedResponse, FileEntry, FilesResponse, Location } from '$lib/gen/ui_pb';
import {
	parseWorkbenchParams,
	serializeWorkbenchParams,
	WORKBENCH_MODES
} from '$lib/workbench-url';

import { mockPage, resetMockPage } from './support/browse-page-state.svelte';

// The Affected-tab half (Task 3) mounts the full route, mirroring
// workbench-impact.test.ts / workbench-callers-callees.test.ts's mocking
// conventions exactly.
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

let currentAffectedImpl: (req: { files: string[]; depth: number }) => Promise<AffectedResponse> =
	() => Promise.reject(new Error('workbench-affected.test.ts: no affected stub configured'));
let currentFilesImpl: () => Promise<FilesResponse> = () =>
	Promise.resolve({ format: 'flat', files: [], tree: [] } as unknown as FilesResponse);

vi.doMock('$lib/client', () => ({
	uiClient: {
		impact: () => Promise.reject(new Error('not used by this test file')),
		callers: () => Promise.reject(new Error('not used by this test file')),
		callees: () => Promise.reject(new Error('not used by this test file')),
		affected: (req: { files: string[]; depth: number }) => currentAffectedImpl(req),
		files: () => currentFilesImpl()
	}
}));

const { default: WorkbenchPage } = await import('../src/routes/workbench/+page.svelte');

function loc(name: string, kind: string, filePath: string, startLine: number): Location {
	return { name, kind, filePath, startLine } as unknown as Location;
}

function affectedResponse(files: string[], affectedTests: Location[]): AffectedResponse {
	return { files, affectedTests } as unknown as AffectedResponse;
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

function mountWorkbench(affectedImpl: (req: { files: string[]; depth: number }) => Promise<AffectedResponse>) {
	currentAffectedImpl = affectedImpl;
	return render(WorkbenchPage, { context: new Map([['statusGate', okStatusGate()]]) });
}

beforeEach(async () => {
	const { goto } = await import('$app/navigation');
	vi.mocked(goto).mockClear();
});

function fileEntry(path: string): FileEntry {
	return { path, language: 'go', nodeCount: 1n, edgeCount: 0n } as FileEntry;
}

function filesResponse(paths: string[]): FilesResponse {
	return { format: 'flat', files: paths.map(fileEntry), tree: [] } as unknown as FilesResponse;
}

function stubFilesClient(paths: string[]): FilesClient {
	return {
		files: () => Promise.resolve(filesResponse(paths))
	};
}

const DEBOUNCE_SETTLE_MS = 250; // SEARCH_DEBOUNCE_MS (150) + margin

async function typeQuery(input: HTMLElement, value: string) {
	await fireEvent.input(input, { target: { value } });
	await vi.advanceTimersByTimeAsync(DEBOUNCE_SETTLE_MS);
	await Promise.resolve();
	await Promise.resolve();
}

// Harness component driving FilePicker exactly the way +page.svelte
// will (props-in/callback-out, no separate selection state) — a plain
// mount helper that maintains `files` itself and re-renders FilePicker
// with the updated array on every onChange, letting a test simulate the
// full search-then-add loop across several selections.
function mountPicker(client: FilesClient, initialFiles: string[] = []) {
	let files = initialFiles;
	const onChange = vi.fn((next: string[]) => {
		files = next;
		rerender({ client, files, onChange });
	});
	const { rerender, unmount } = render(FilePicker, { props: { client, files, onChange } });
	return {
		unmount,
		onChange,
		currentFiles: () => files
	};
}

describe('FilePicker: search-and-add with removable chips (WRK-02)', () => {
	it('typing a term and selecting a result appends a chip whose text is the repo-relative path', async () => {
		vi.useFakeTimers();
		const client = stubFilesClient(['internal/query/files.go']);
		mountPicker(client);

		const input = screen.getByTestId('file-picker-input');
		await typeQuery(input, 'files');

		const result = screen.getByTestId('file-picker-result-internal/query/files.go');
		await fireEvent.click(result);

		const chip = await screen.findByTestId('file-picker-chip-internal/query/files.go');
		expect(within(chip).getByText('internal/query/files.go')).toBeInTheDocument();
		vi.useRealTimers();
	});

	it('selecting the same path twice yields one chip, not two', async () => {
		vi.useFakeTimers();
		const client = stubFilesClient(['a.go']);
		mountPicker(client);

		const input = screen.getByTestId('file-picker-input');
		await typeQuery(input, 'a.go');
		await fireEvent.click(screen.getByTestId('file-picker-result-a.go'));
		await typeQuery(input, 'a.go');
		await fireEvent.click(screen.getByTestId('file-picker-result-a.go'));

		expect(screen.getAllByTestId('file-picker-chip-a.go')).toHaveLength(1);
		vi.useRealTimers();
	});

	it('each chip has a remove control with an accessible name naming the path; activating it removes exactly that chip and preserves order', async () => {
		const client = stubFilesClient([]);
		const picker = mountPicker(client, ['a.go', 'b.go', 'c.go']);

		const removeButton = screen.getByRole('button', { name: 'Remove b.go' });
		await fireEvent.click(removeButton);

		expect(picker.currentFiles()).toEqual(['a.go', 'c.go']);
		expect(screen.queryByTestId('file-picker-chip-b.go')).not.toBeInTheDocument();
		expect(screen.getByTestId('file-picker-chip-a.go')).toBeInTheDocument();
		expect(screen.getByTestId('file-picker-chip-c.go')).toBeInTheDocument();
	});

	it('WR-07 regression: a rejected Files RPC renders a visible failure message, not an indistinguishable "No results."', async () => {
		vi.useFakeTimers();
		const client: FilesClient = {
			files: () => Promise.reject(new Error('server unavailable'))
		};
		mountPicker(client);

		const input = screen.getByTestId('file-picker-input');
		await typeQuery(input, 'foo');

		expect(screen.getByTestId('file-picker-failure')).toHaveTextContent('server unavailable');
		vi.useRealTimers();
	});
});

describe('FilePicker + workbench-url.ts: the URL round trip (D-11)', () => {
	it('write direction: three chips produce three file= entries, in chip order', () => {
		const params = parseWorkbenchParams(new URLSearchParams('mode=affected'));
		params.files = ['a.go', 'b/c.go', 'd.go'];
		const url = serializeWorkbenchParams(params);

		expect(new URLSearchParams(url).getAll('file')).toEqual(['a.go', 'b/c.go', 'd.go']);
	});

	it('read direction: ?mode=affected&file=a.go&file=b/c.go restores exactly two chips in that order', async () => {
		const parsed = parseWorkbenchParams(
			new URLSearchParams('mode=affected&file=a.go&file=b%2Fc.go')
		);
		expect(parsed.files).toEqual(['a.go', 'b/c.go']);

		const client = stubFilesClient([]);
		mountPicker(client, parsed.files);

		const chips = screen.getAllByTestId(/^file-picker-chip-/);
		expect(chips.map((c) => c.getAttribute('data-testid'))).toEqual([
			'file-picker-chip-a.go',
			'file-picker-chip-b/c.go'
		]);
	});

	it('a path containing a comma round-trips intact through the URL and back into a single chip', () => {
		const params = parseWorkbenchParams(new URLSearchParams('mode=affected'));
		params.files = ['a,b.go'];
		const url = serializeWorkbenchParams(params);

		const roundTripped = parseWorkbenchParams(new URLSearchParams(url));
		expect(roundTripped.files).toEqual(['a,b.go']);

		const client = stubFilesClient([]);
		mountPicker(client, roundTripped.files);
		expect(screen.getByTestId('file-picker-chip-a,b.go')).toBeInTheDocument();
	});
});

describe('workbench Affected tab: multi-file selection drives a sortable table (WRK-02, Task 3)', () => {
	it('parseWorkbenchParams -> Affected rpc -> DataTable renders all rows, recording exactly one call with the URL files and depth', async () => {
		const calls: Array<{ files: string[]; depth: number }> = [];
		const response = affectedResponse(
			['a.go', 'b.go'],
			[loc('TestAlpha', 'func', 'a_test.go', 5), loc('TestBeta', 'func', 'b_test.go', 9)]
		);

		resetMockPage('http://localhost/workbench?mode=affected&file=a.go&file=b.go&depth=3');
		mountWorkbench((req) => {
			calls.push({ files: req.files, depth: req.depth });
			return Promise.resolve(response);
		});

		await waitFor(() => expect(screen.getByRole('table')).toBeInTheDocument());
		expect(screen.getByText('TestAlpha')).toBeInTheDocument();
		expect(screen.getByText('TestBeta')).toBeInTheDocument();

		expect(calls).toEqual([{ files: ['a.go', 'b.go'], depth: 3 }]);
	});

	it('the echoed files render in an element PRECEDING the table and do NOT appear as a column — the Affected stub records exactly ONE call', async () => {
		const response = affectedResponse(['a.go', 'b.go'], [loc('TestAlpha', 'func', 'a_test.go', 5)]);
		const calls: Array<{ files: string[]; depth: number }> = [];

		resetMockPage('http://localhost/workbench?mode=affected&file=a.go&file=b.go');
		mountWorkbench((req) => {
			calls.push({ files: req.files, depth: req.depth });
			return Promise.resolve(response);
		});

		const table = await screen.findByRole('table');
		const summaryEl = await screen.findByTestId('workbench-affected-files-summary');

		// Node.compareDocumentPosition's DOCUMENT_POSITION_FOLLOWING bit,
		// set on `table` as observed from `summaryEl`, means summaryEl
		// precedes table in document order (workbench-impact.test.ts's
		// own D-06 precedent).
		const position = summaryEl.compareDocumentPosition(table);
		expect(position & Node.DOCUMENT_POSITION_FOLLOWING).toBe(Node.DOCUMENT_POSITION_FOLLOWING);

		const headers = within(table)
			.getAllByRole('columnheader')
			.map((h) => h.textContent?.trim());
		expect(new Set(headers)).toEqual(new Set(['Name', 'Kind', 'File', 'Line']));

		expect(calls).toHaveLength(1);
	});

	it("T-04-32-equivalent: adding a chip issues NO additional GetStatus call — 'file' is a per-key ROUTE_LOCAL_PARAMS entry under /workbench", async () => {
		const { navigationIdentity, createStatusGate } = await import('$lib/status');

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

		const initialUrl = new URL('http://localhost/workbench?mode=affected&file=a.go');
		const gate = createStatusGate(client, navigationIdentity(initialUrl));
		await vi.waitFor(() => expect(getStatus).toHaveBeenCalledTimes(1));

		const nextParams = parseWorkbenchParams(initialUrl.searchParams);
		nextParams.files = ['a.go', 'b.go'];
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

	it('with zero chips selected, no Affected request is issued and an explicit empty state renders — not a bare table', async () => {
		const calls: Array<{ files: string[]; depth: number }> = [];
		resetMockPage('http://localhost/workbench?mode=affected');
		mountWorkbench((req) => {
			calls.push({ files: req.files, depth: req.depth });
			return Promise.resolve(affectedResponse([], []));
		});

		expect(await screen.findByTestId('workbench-affected-empty')).toBeInTheDocument();
		expect(screen.queryByRole('table')).not.toBeInTheDocument();
		expect(calls).toHaveLength(0);
	});

	it('adding a chip issues a new Affected request and replaces the rows, without remounting the route or navigating', async () => {
		currentFilesImpl = () =>
			Promise.resolve({
				format: 'flat',
				files: [{ path: 'c.go', language: 'go', nodeCount: 1n, edgeCount: 0n }],
				tree: []
				// eslint-disable-next-line @typescript-eslint/no-explicit-any
			} as any);

		const calls: Array<{ files: string[]; depth: number }> = [];
		resetMockPage('http://localhost/workbench?mode=affected&file=a.go');
		mountWorkbench((req) => {
			calls.push({ files: req.files, depth: req.depth });
			return Promise.resolve(
				affectedResponse(req.files, [
					loc(`Test-for-${req.files.join(',')}`, 'func', 'x_test.go', 1)
				])
			);
		});

		await waitFor(() => expect(screen.getByText('Test-for-a.go')).toBeInTheDocument());
		const headingBefore = screen.getByRole('heading', { name: /workbench/i });

		const input = screen.getByTestId('file-picker-input');
		await fireEvent.input(input, { target: { value: 'c.go' } });
		// Real time — SEARCH_DEBOUNCE_MS (150) plus margin — rather than
		// fake timers, so this interaction does not need to coordinate
		// with the route's own (timer-free) request effects.
		await new Promise((resolve) => setTimeout(resolve, 250));

		const result = await screen.findByTestId('file-picker-result-c.go');
		await fireEvent.click(result);

		await waitFor(() => expect(screen.getByText('Test-for-a.go,c.go')).toBeInTheDocument());
		expect(screen.queryByText('Test-for-a.go')).not.toBeInTheDocument();
		expect(calls).toEqual([
			{ files: ['a.go'], depth: 0 },
			{ files: ['a.go', 'c.go'], depth: 0 }
		]);

		// (a) the route component instance is not re-created.
		const headingAfter = screen.getByRole('heading', { name: /workbench/i });
		expect(headingAfter).toBe(headingBefore);

		// (b) no navigation was dispatched.
		const { goto } = await import('$app/navigation');
		expect(goto).not.toHaveBeenCalled();
	});

	it('D-07: a depth above MaxDepth (9999) reaches the stub client unchanged — the UI applies no client-side bound', async () => {
		const calls: Array<{ files: string[]; depth: number }> = [];
		resetMockPage('http://localhost/workbench?mode=affected&file=a.go&depth=9999');
		mountWorkbench((req) => {
			calls.push({ files: req.files, depth: req.depth });
			return Promise.resolve(affectedResponse(req.files, []));
		});

		await waitFor(() => expect(calls).toEqual([{ files: ['a.go'], depth: 9999 }]));
	});

	it('WRK-04: clicking the Name column header sorts ascending, clicking again reverses to descending', async () => {
		resetMockPage('http://localhost/workbench?mode=affected&file=a.go');
		mountWorkbench(() =>
			Promise.resolve(
				affectedResponse(
					['a.go'],
					[
						loc('Beta', 'func', 'b_test.go', 20),
						loc('Gamma', 'func', 'c_test.go', 30),
						loc('Alpha', 'func', 'a_test.go', 10)
					]
				)
			)
		);

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

	it('every WORKBENCH_MODES tab renders a live panel — none renders a not-yet-wired placeholder', async () => {
		for (const mode of WORKBENCH_MODES) {
			resetMockPage(`http://localhost/workbench?mode=${mode}&symbol=Engine.Status&file=a.go`);
			const { unmount } = mountWorkbench(() => Promise.resolve(affectedResponse(['a.go'], [])));

			expect(screen.queryByTestId('workbench-mode-placeholder')).not.toBeInTheDocument();

			unmount();
		}
	});
});
