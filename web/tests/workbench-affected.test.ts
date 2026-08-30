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
import { describe, expect, it, vi } from 'vitest';

import FilePicker from '$lib/components/workbench/FilePicker.svelte';
import type { FilesClient } from '$lib/file-search';
import type { FileEntry, FilesResponse } from '$lib/gen/ui_pb';
import { parseWorkbenchParams, serializeWorkbenchParams } from '$lib/workbench-url';

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
