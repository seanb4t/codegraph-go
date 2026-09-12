// 03-01 (BRW-06): global vitest setup, loaded via vite.config.ts's
// test.setupFiles. Registers jsdom-aware DOM matchers
// (toBeInTheDocument, etc.) and @testing-library/svelte's own vitest
// integration (auto-cleanup after each test) once, for every test file
// under web/tests/.
import '@testing-library/jest-dom/vitest';
import '@testing-library/svelte/vitest';

// 09-04 (BRW-11/BRW-12, editor-prefs.ts): Node >=22 ships a global Web
// Storage API (localStorage/sessionStorage) ON by default, but every
// call silently no-ops with an ExperimentalWarning unless the process
// was started with --localstorage-file — and this global is installed
// on `globalThis` BEFORE jsdom constructs its own window, so jsdom's
// real Storage implementation never gets a chance to win the property.
// editor-prefs.ts is the SPA's first localStorage consumer (09-PATTERNS.md
// § editor-prefs.ts: no analog exists), so this is the first test run to
// notice a bare `window.localStorage.setItem` doing nothing at all.
// Replace it here, once, for the whole suite, with a minimal but fully
// functional in-memory Storage — a real browser's localStorage, and a
// pre-22 Node's jsdom-backed one, already behave this way; this shim
// only compensates for Node's own broken-by-default global.
function probeStorageWorks(storage: Storage | undefined): boolean {
	if (!storage) return false;
	try {
		const key = '__codegraph_storage_probe__';
		storage.setItem(key, '1');
		const ok = storage.getItem(key) === '1';
		storage.removeItem(key);
		return ok;
	} catch {
		return false;
	}
}

function inMemoryStorage(): Storage {
	const store = new Map<string, string>();
	return {
		getItem: (key: string) => (store.has(key) ? (store.get(key) as string) : null),
		setItem: (key: string, value: string) => {
			store.set(key, String(value));
		},
		removeItem: (key: string) => {
			store.delete(key);
		},
		clear: () => store.clear(),
		key: (index: number) => Array.from(store.keys())[index] ?? null,
		get length() {
			return store.size;
		}
	} as Storage;
}

if (!probeStorageWorks(globalThis.localStorage)) {
	Object.defineProperty(globalThis, 'localStorage', {
		value: inMemoryStorage(),
		configurable: true,
		writable: true
	});
}

// 04-07 Task 4 (D-08): @tanstack/svelte-virtual computes its visible row
// range from the scroll container's REAL offsetWidth/offsetHeight
// (@tanstack/virtual-core's `getRect`, not getBoundingClientRect) —
// jsdom has no layout engine at all (a long-standing, deliberate jsdom
// limitation: github.com/jsdom/jsdom, "jsdom does not implement layout"),
// so every element's offsetWidth/offsetHeight is hardcoded to 0. Without
// this stub, the virtualizer's outerSize is always 0, its calculated
// range is null, and DataTable.svelte renders ZERO rows under every
// jsdom-based test regardless of row count — confirmed live: this was
// the root cause of 15 failing tests across every file that renders a
// DataTable (workbench-*.test.ts AND health-page.test.ts via
// CountTable.svelte, D-06's shared shell) the first time virtualization
// landed.
//
// Scoped to ONLY the DataTable scroll container (its own
// data-testid="data-table-scroll", DataTable.svelte) rather than a
// blanket override of every element's offsetWidth/offsetHeight — every
// other element keeps jsdom's native (0) behavior, so this cannot mask
// an unrelated layout bug elsewhere. The height/width values are
// arbitrary but realistic — they only need to be non-zero and large
// enough that a handful of rows fit in the visible range so tests can
// assert on rendered content.
const DATA_TABLE_SCROLL_TESTID = 'data-table-scroll';
const STUBBED_HEIGHT = 600;
const STUBBED_WIDTH = 800;

Object.defineProperty(HTMLElement.prototype, 'offsetHeight', {
	configurable: true,
	get(this: HTMLElement) {
		return this.dataset.testid === DATA_TABLE_SCROLL_TESTID ? STUBBED_HEIGHT : 0;
	}
});
Object.defineProperty(HTMLElement.prototype, 'offsetWidth', {
	configurable: true,
	get(this: HTMLElement) {
		return this.dataset.testid === DATA_TABLE_SCROLL_TESTID ? STUBBED_WIDTH : 0;
	}
});
