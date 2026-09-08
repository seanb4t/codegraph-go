// Edge coverage for SearchPanel.svelte's rendering and keyboard bindings
// (BRW-01, BRW-08, NAV-03, D-15, D-17). This file is written and run RED
// (SearchPanel.svelte does not exist yet) before any implementation —
// recorded in 03-06-SUMMARY.md — then made GREEN.
//
// jsdom does not implement scrollIntoView; the vendored Command primitive
// calls it as a side effect of moving selection (confirmed in Task 1's
// keyboard spike). Stubbed once here, globally for this file, rather than
// per-test.
if (!Element.prototype.scrollIntoView) {
	Element.prototype.scrollIntoView = () => {};
}

import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';

import SearchPanel from '$lib/components/browse/SearchPanel.svelte';
import type { SearchClient } from '$lib/search';
import type { Location, FileEntry, SearchResponse, FilesResponse, ExploreResponse } from '$lib/gen/ui_pb';

function location(name: string, filePath = 'a.go', startLine = 1): Location {
	return { name, kind: 'func', filePath, startLine } as Location;
}

function fileEntry(path: string): FileEntry {
	return { path, language: 'go', nodeCount: 1n, edgeCount: 0n } as FileEntry;
}

function stubClient(overrides: Partial<SearchClient> = {}): SearchClient {
	return {
		search: overrides.search ?? (() => Promise.resolve({ locations: [] } as unknown as SearchResponse)),
		files:
			overrides.files ??
			(() => Promise.resolve({ format: 'flat', files: [], tree: [] } as unknown as FilesResponse)),
		explore:
			overrides.explore ??
			(() =>
				Promise.resolve({
					query: '',
					empty: true,
					stale: false,
					symbolCount: 0,
					groups: [],
					blasts: []
				} as unknown as ExploreResponse))
	};
}

const DEBOUNCE_SETTLE_MS = 250; // SEARCH_DEBOUNCE_MS (150) + margin

async function typeQuery(input: HTMLElement, value: string) {
	await fireEvent.input(input, { target: { value } });
	await vi.advanceTimersByTimeAsync(DEBOUNCE_SETTLE_MS);
	await Promise.resolve();
	await Promise.resolve();
}

beforeEach(() => {
	vi.useFakeTimers();
});

afterEach(() => {
	vi.useRealTimers();
});

describe('SearchPanel: two live sections, in server order', () => {
	it('shows Symbols then Files, each in the order supplied, given both result kinds', async () => {
		const client = stubClient({
			search: () =>
				Promise.resolve({
					locations: [location('Zeta'), location('Alpha')]
				} as unknown as SearchResponse),
			files: () =>
				Promise.resolve({
					format: 'flat',
					files: [fileEntry('z.go'), fileEntry('a.go')],
					tree: []
				} as unknown as FilesResponse)
		});

		render(SearchPanel, { props: { client, onSelect: vi.fn() } });
		const input = screen.getByPlaceholderText(/search/i);
		await typeQuery(input, 'ab');

		const headings = screen.getAllByText(/^(Symbols|Files)$/);
		expect(headings.map((h) => h.textContent)).toEqual(['Symbols', 'Files']);

		// Queried by testid rather than plain text: each item's sibling
		// <span> (file path / line) makes a bare-name getByText ambiguous
		// against the item's own full textContent.
		const symbolItems = screen.getAllByTestId(/^search-item-symbol:/);
		expect(symbolItems.map((el) => el.getAttribute('data-testid'))).toEqual([
			'search-item-symbol:Zeta:a.go:1',
			'search-item-symbol:Alpha:a.go:1'
		]);

		const fileItems = screen.getAllByTestId(/^search-item-file:/);
		expect(fileItems.map((el) => el.getAttribute('data-testid'))).toEqual([
			'search-item-file:z.go',
			'search-item-file:a.go'
		]);
	});
});

describe('SearchPanel: the explore section', () => {
	it('shows a third section with explore results, without disturbing the live sections', async () => {
		const client = stubClient({
			search: () => Promise.resolve({ locations: [location('Foo')] } as unknown as SearchResponse),
			files: () =>
				Promise.resolve({
					format: 'flat',
					files: [fileEntry('foo.go')],
					tree: []
				} as unknown as FilesResponse),
			explore: () =>
				Promise.resolve({
					query: 'how does foo work',
					empty: false,
					stale: false,
					symbolCount: 1,
					groups: [{ path: 'foo.go', symbols: [], skeletonized: false, source: undefined }],
					blasts: []
				} as unknown as ExploreResponse)
		});

		render(SearchPanel, { props: { client, onSelect: vi.fn() } });
		const input = screen.getByPlaceholderText(/search/i);
		await typeQuery(input, 'how does foo work');

		// Live sections still present — queried by testid, not text, since
		// the symbol item's own sibling <span> makes plain getByText('Foo')
		// ambiguous against the item's full textContent.
		expect(screen.getByTestId('search-item-symbol:Foo:a.go:1')).toBeInTheDocument();

		// Home guarantees the leading "ask" item is highlighted regardless
		// of the primitive's own auto-select-first-item timing, so Enter
		// deterministically submits rather than opening a live result.
		await fireEvent.keyDown(input, { key: 'Home' });
		await fireEvent.keyDown(input, { key: 'Enter' });
		await Promise.resolve();
		await Promise.resolve();

		expect(screen.getByText('Explore')).toBeInTheDocument();
		// Distinct testid from the live Files item's search-item-file:foo.go
		expect(screen.getByTestId('search-item-explore:foo.go')).toBeInTheDocument();
		// live sections untouched by the submit
		expect(screen.getByTestId('search-item-symbol:Foo:a.go:1')).toBeInTheDocument();
	});

	it('renders a no-results line for an empty explore outcome, distinct from the failure line for a rejected explore', async () => {
		const emptyClient = stubClient({
			explore: () =>
				Promise.resolve({
					query: 'nothing here',
					empty: true,
					stale: false,
					symbolCount: 0,
					groups: [],
					blasts: []
				} as unknown as ExploreResponse)
		});
		const { unmount } = render(SearchPanel, { props: { client: emptyClient, onSelect: vi.fn() } });
		let input = screen.getByPlaceholderText(/search/i);
		await typeQuery(input, 'nothing here');
		await fireEvent.keyDown(input, { key: 'Home' });
		await fireEvent.keyDown(input, { key: 'Enter' });
		await Promise.resolve();
		await Promise.resolve();
		const emptyText = screen.getByTestId('explore-empty').textContent;
		unmount();

		const failClient = stubClient({ explore: () => Promise.reject(new Error('boom')) });
		render(SearchPanel, { props: { client: failClient, onSelect: vi.fn() } });
		input = screen.getByPlaceholderText(/search/i);
		await typeQuery(input, 'nothing here');
		await fireEvent.keyDown(input, { key: 'Home' });
		await fireEvent.keyDown(input, { key: 'Enter' });
		await Promise.resolve();
		await Promise.resolve();
		const failText = screen.getByTestId('explore-failed').textContent;

		expect(emptyText).not.toBe(failText);
		expect(emptyText).toBeTruthy();
		expect(failText).toBeTruthy();
	});
});

describe('SearchPanel: keyboard shortcuts', () => {
	it('/ on the document body moves focus to the search input', async () => {
		render(SearchPanel, { props: { client: stubClient(), onSelect: vi.fn() } });
		document.body.focus();
		expect(document.activeElement).toBe(document.body);

		await fireEvent.keyDown(document.body, { key: '/' });

		const input = screen.getByPlaceholderText(/search/i);
		expect(document.activeElement).toBe(input);
	});

	it('/ while focus is already inside a text input does not move focus and the character is inserted', async () => {
		render(SearchPanel, { props: { client: stubClient(), onSelect: vi.fn() } });
		// Appended via createElement/appendChild — NOT innerHTML +=, which
		// re-serializes and recreates the ENTIRE body subtree (including
		// the just-rendered component's container), silently breaking
		// @testing-library/svelte's cleanup-by-reference for every test
		// that runs after this one in the same file.
		const other = document.createElement('input');
		other.setAttribute('data-testid', 'other-text-input');
		other.value = 'path';
		document.body.appendChild(other);
		other.focus();
		expect(document.activeElement).toBe(other);

		const notPrevented = await fireEvent.keyDown(other, { key: '/' });
		expect(notPrevented).toBe(true); // fireEvent returns false only when preventDefault() was called

		// Simulate the browser's own default continuation of an
		// unprevented keydown: the character is appended to the input.
		await fireEvent.input(other, { target: { value: other.value + '/' } });

		expect(document.activeElement).toBe(other); // focus did not move
		expect(other.value).toBe('path/'); // character was inserted

		other.remove();
	});

	it('Cmd+K (metaKey) moves focus to the search input and calls preventDefault', async () => {
		render(SearchPanel, { props: { client: stubClient(), onSelect: vi.fn() } });
		document.body.focus();

		const notPrevented = await fireEvent.keyDown(document.body, { key: 'k', metaKey: true });
		expect(notPrevented).toBe(false); // preventDefault WAS called

		const input = screen.getByPlaceholderText(/search/i);
		expect(document.activeElement).toBe(input);
	});

	it('Ctrl+K moves focus to the search input and calls preventDefault', async () => {
		render(SearchPanel, { props: { client: stubClient(), onSelect: vi.fn() } });
		document.body.focus();

		const notPrevented = await fireEvent.keyDown(document.body, { key: 'k', ctrlKey: true });
		expect(notPrevented).toBe(false);

		const input = screen.getByPlaceholderText(/search/i);
		expect(document.activeElement).toBe(input);
	});

	it('Escape dismisses the results surface', async () => {
		const client = stubClient({
			search: () => Promise.resolve({ locations: [location('Foo')] } as unknown as SearchResponse)
		});
		render(SearchPanel, { props: { client, onSelect: vi.fn() } });
		const input = screen.getByPlaceholderText(/search/i);
		await typeQuery(input, 'ab');
		expect(screen.getByTestId('search-item-symbol:Foo:a.go:1')).toBeInTheDocument();

		await fireEvent.keyDown(input, { key: 'Escape' });

		expect(screen.queryByTestId('search-item-symbol:Foo:a.go:1')).not.toBeInTheDocument();
	});
});

describe('SearchPanel: keyboard navigation across sections', () => {
	it('ArrowDown from the last Symbols item moves the highlight to the first Files item', async () => {
		const client = stubClient({
			search: () =>
				Promise.resolve({
					locations: [location('sym-a'), location('sym-b')]
				} as unknown as SearchResponse),
			files: () =>
				Promise.resolve({
					format: 'flat',
					files: [fileEntry('file-a'), fileEntry('file-b')],
					tree: []
				} as unknown as FilesResponse)
		});

		render(SearchPanel, { props: { client, onSelect: vi.fn() } });
		const input = screen.getByPlaceholderText(/search/i);
		await typeQuery(input, 'ab');

		// Navigate to sym-b (the last Symbols item) explicitly, then one
		// more ArrowDown must land on file-a (the first Files item).
		const observed: string[] = [];
		for (let i = 0; i < 6; i++) {
			await fireEvent.keyDown(input, { key: 'ArrowDown' });
			const selected = document.querySelector('[data-selected]');
			observed.push(selected?.getAttribute('data-value') ?? '(none)');
		}

		const lastSymbolKey = 'symbol:sym-b:a.go:1';
		const firstFileKey = 'file:file-a';
		const symBIndex = observed.indexOf(lastSymbolKey);
		expect(symBIndex).toBeGreaterThanOrEqual(0);
		expect(observed[symBIndex + 1]).toBe(firstFileKey);
	});

	it('Enter on a highlighted item invokes onSelect with that item', async () => {
		const client = stubClient({
			search: () =>
				Promise.resolve({ locations: [location('Foo', 'foo.go', 7)] } as unknown as SearchResponse)
		});
		const onSelect = vi.fn();
		render(SearchPanel, { props: { client, onSelect } });
		const input = screen.getByPlaceholderText(/search/i);
		await typeQuery(input, 'ab');

		// Navigate onto the Foo symbol item specifically.
		let observed = '';
		for (let i = 0; i < 6 && observed !== 'symbol:Foo:foo.go:7'; i++) {
			await fireEvent.keyDown(input, { key: 'ArrowDown' });
			observed = document.querySelector('[data-selected]')?.getAttribute('data-value') ?? '';
		}
		expect(observed).toBe('symbol:Foo:foo.go:7');

		await fireEvent.keyDown(input, { key: 'Enter' });

		expect(onSelect).toHaveBeenCalledWith({ kind: 'symbol', location: expect.objectContaining({ name: 'Foo' }) });
	});
});

describe('SearchPanel: initialQuery seeds on mount and resyncs only on a non-typing change (WR-01 fix for IN-12)', () => {
	it('a typing-originated initialQuery change (query already in sync) does not re-seed or reset the debounce timer', async () => {
		// IN-12's original bug: a bare tracked read of `initialQuery`
		// re-ran the seed effect on every prop change, including the
		// echo of the user's own typing, resetting the 150ms debounce
		// timer a second time per keystroke. The fix must still treat
		// this case as a no-op — but by comparing against the
		// controller's CURRENT query, not by ignoring `initialQuery`
		// changes altogether (that over-correction is WR-01).
		const searchSpy = vi.fn(
			() => Promise.resolve({ locations: [] }) as unknown as Promise<SearchResponse>
		);
		const client = stubClient({ search: searchSpy });
		const onSelect = vi.fn();

		const { rerender } = render(SearchPanel, { props: { client, initialQuery: 'Foo', onSelect } });

		// Advance to just under the debounce window the INITIAL seed
		// started.
		await vi.advanceTimersByTimeAsync(100);
		expect(searchSpy).not.toHaveBeenCalled();

		// Simulate typing, not a URL-only change: `handleInputChange`
		// calls `controller.setQuery` SYNCHRONOUSLY (search.ts), starting
		// a FRESH 150ms debounce window at t=100 (deadline t=250). By the
		// time the caller's `q` param — and thus this prop — reflects
		// 'FooBar', `searchState.query` already equals 'FooBar'. Fire the
		// input event first, exactly as the real component does, then
		// rerender with the matching `initialQuery` a little later (t=120)
		// so a second reset would be distinguishable in time from the
		// typing-triggered one.
		const input = screen.getByPlaceholderText(/search/i);
		await fireEvent.input(input, { target: { value: 'FooBar' } });
		await vi.advanceTimersByTimeAsync(20); // now t=120
		await rerender({ client, initialQuery: 'FooBar', onSelect });

		// Advance to just under the typing-triggered deadline (t=250).
		// If the seed effect had ALSO re-seeded at the t=120 rerender
		// (the old IN-12 bug, or a fix that ignores the current query),
		// the debounce clock would have reset AGAIN to a t=270 deadline,
		// and the search would not have fired yet here.
		await vi.advanceTimersByTimeAsync(125); // now t=245
		expect(searchSpy).not.toHaveBeenCalled();

		// Cross the ORIGINAL typing deadline (t=250) — proving the
		// rerender's seed effect did NOT reset the timer.
		await vi.advanceTimersByTimeAsync(10); // now t=255
		await Promise.resolve();
		await Promise.resolve();

		expect(searchSpy).toHaveBeenCalledTimes(1);
		expect(searchSpy).toHaveBeenCalledWith(
			expect.objectContaining({ term: 'FooBar' }),
			expect.anything()
		);
	});

	it('a URL-originated initialQuery change NOT preceded by typing (browser Back/Forward) resyncs the box and refires search (WR-01)', async () => {
		// Concrete repro from WR-01: type 'alpha', navigate away (q stays
		// 'alpha' in a pushed history entry), type 'beta' (replaces the
		// entry), then press Back — the URL reverts to q=alpha WITHOUT
		// the user typing it. `searchState.query` is still 'beta' at that
		// point; only `initialQuery` (the prop) changes.
		const searchSpy = vi.fn(
			() => Promise.resolve({ locations: [] }) as unknown as Promise<SearchResponse>
		);
		const client = stubClient({ search: searchSpy });
		const onSelect = vi.fn();

		const { rerender } = render(SearchPanel, { props: { client, initialQuery: 'beta', onSelect } });
		await vi.advanceTimersByTimeAsync(DEBOUNCE_SETTLE_MS);
		searchSpy.mockClear();

		await rerender({ client, initialQuery: 'alpha', onSelect });
		await vi.advanceTimersByTimeAsync(DEBOUNCE_SETTLE_MS);
		await Promise.resolve();

		const input = screen.getByPlaceholderText(/search/i) as HTMLInputElement;
		expect(input.value).toBe('alpha');
		expect(searchSpy).toHaveBeenCalledWith(
			expect.objectContaining({ term: 'alpha' }),
			expect.anything()
		);
	});
});

describe('SearchPanel: unmounting mid-debounce never dispatches a stale search (IN-13)', () => {
	it('a pending debounce timer started before unmount never fires the RPC afterward', async () => {
		const searchSpy = vi.fn(
			() => Promise.resolve({ locations: [] }) as unknown as Promise<SearchResponse>
		);
		const client = stubClient({ search: searchSpy });
		const onSelect = vi.fn();

		const { unmount } = render(SearchPanel, { props: { client, onSelect } });
		const input = screen.getByPlaceholderText(/search/i);

		// Start a debounce window, then unmount BEFORE it elapses — a
		// component torn down mid-debounce, the exact scenario IN-13
		// describes. Before this fix, search.ts's controller had no
		// dispose() and nothing called it on unmount, so this timer would
		// still fire and dispatch Search for a component that no longer
		// exists.
		await fireEvent.input(input, { target: { value: 'ab' } });
		unmount();

		await vi.advanceTimersByTimeAsync(DEBOUNCE_SETTLE_MS);
		expect(searchSpy).not.toHaveBeenCalled();
	});
});
