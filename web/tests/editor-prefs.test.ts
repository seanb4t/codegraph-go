// editor-prefs.test.ts — 09-04 Task 1: the per-browser editor-link
// override in localStorage, and templateForRequest, the ONE function
// that turns a stored preference into a GetEditorLinkRequest.template
// value (the assumption-delta invariant: the probe and a gutter click
// must resolve identically for the same override). Written and run RED
// against stubs returning null/undefined before the real implementation
// exists.
import { describe, it, expect } from 'vitest';
import {
	EDITOR_OVERRIDE_STORAGE_KEY,
	readEditorOverride,
	writeEditorOverride,
	clearEditorOverride,
	templateForRequest,
	type EditorOverride
} from '$lib/editor-prefs';
import type { EditorPreset } from '$lib/gen/ui_pb';

// memoryStorage — a tiny in-memory Storage fake, the same shape jsdom's
// real localStorage exposes, so readEditorOverride/writeEditorOverride
// are exercised against a real Storage-shaped object without depending
// on jsdom's own global state leaking between tests.
function memoryStorage(initial: Record<string, string> = {}): Storage {
	const store = new Map(Object.entries(initial));
	return {
		getItem: (key: string) => (store.has(key) ? (store.get(key) as string) : null),
		setItem: (key: string, value: string) => {
			store.set(key, value);
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

// throwingStorage — a Storage fake whose named method throws, modeling
// a disabled/quota-exceeded/private-mode Storage. Every editor-prefs.ts
// call against one of these must degrade to "no override", never throw.
function throwingStorage(which: 'getItem' | 'setItem' | 'removeItem'): Storage {
	const boom = () => {
		throw new Error(`${which} boom`);
	};
	return {
		getItem: which === 'getItem' ? boom : () => null,
		setItem: which === 'setItem' ? boom : () => {},
		removeItem: which === 'removeItem' ? boom : () => {},
		clear: () => {},
		key: () => null,
		length: 0
	} as Storage;
}

function preset(id: string, template: string): EditorPreset {
	return { id, name: id, template } as EditorPreset;
}

const PRESETS: EditorPreset[] = [
	preset('vscode', 'vscode://file/{path}:{line}:{col}'),
	preset('cursor', 'cursor://file/{path}:{line}:{col}'),
	preset('jetbrains', 'jetbrains://gateway/navigate/reference?path={path}&line={line}')
];

describe('readEditorOverride', () => {
	it('returns null when the key is absent', () => {
		expect(readEditorOverride(memoryStorage())).toBeNull();
	});

	it('returns null when the stored value is not JSON', () => {
		const storage = memoryStorage({ [EDITOR_OVERRIDE_STORAGE_KEY]: 'not json {' });
		expect(readEditorOverride(storage)).toBeNull();
	});

	it('returns null for a shape other than the two allowed kinds', () => {
		const cases = [
			{ kind: 'preset' },
			{ kind: 'custom', template: 42 },
			{ kind: 'zzz' }
		];
		for (const value of cases) {
			const storage = memoryStorage({ [EDITOR_OVERRIDE_STORAGE_KEY]: JSON.stringify(value) });
			expect(readEditorOverride(storage)).toBeNull();
		}
	});

	it('returns null when storage is null or undefined', () => {
		expect(readEditorOverride(null)).toBeNull();
	});

	it('returns null when getItem throws', () => {
		expect(readEditorOverride(throwingStorage('getItem'))).toBeNull();
	});
});

describe('writeEditorOverride / clearEditorOverride', () => {
	it('round-trips a preset override through localStorage', () => {
		const storage = memoryStorage();
		const override: EditorOverride = { kind: 'preset', id: 'cursor' };
		writeEditorOverride(override, storage);
		expect(readEditorOverride(storage)).toEqual(override);
	});

	it('round-trips a custom override through localStorage', () => {
		const storage = memoryStorage();
		const override: EditorOverride = { kind: 'custom', template: 'x://{path}' };
		writeEditorOverride(override, storage);
		expect(readEditorOverride(storage)).toEqual(override);
	});

	it('does not throw when setItem throws (quota)', () => {
		expect(() =>
			writeEditorOverride({ kind: 'preset', id: 'cursor' }, throwingStorage('setItem'))
		).not.toThrow();
	});

	it('removes the key', () => {
		const storage = memoryStorage();
		writeEditorOverride({ kind: 'preset', id: 'cursor' }, storage);
		clearEditorOverride(storage);
		expect(readEditorOverride(storage)).toBeNull();
	});

	it('does not throw when removeItem throws', () => {
		expect(() => clearEditorOverride(throwingStorage('removeItem'))).not.toThrow();
	});
});

describe('templateForRequest', () => {
	it('is undefined for a null override (server default applies)', () => {
		expect(templateForRequest(null, PRESETS)).toBeUndefined();
	});

	it('resolves a preset override to that preset template from the given presets list', () => {
		expect(templateForRequest({ kind: 'preset', id: 'cursor' }, PRESETS)).toBe(
			'cursor://file/{path}:{line}:{col}'
		);
	});

	it('is undefined for a preset override whose id is not in the given presets list', () => {
		expect(templateForRequest({ kind: 'preset', id: 'zed' }, PRESETS)).toBeUndefined();
	});

	it('returns a custom template verbatim with no client-side validation', () => {
		expect(templateForRequest({ kind: 'custom', template: 'x://{path}' }, PRESETS)).toBe(
			'x://{path}'
		);
	});

	it('returns the identical value for the same override whether called for the probe or a gutter click (assumption-delta invariant)', () => {
		const override: EditorOverride = { kind: 'preset', id: 'vscode' };
		const forProbe = templateForRequest(override, PRESETS);
		const forGutterClick = templateForRequest(override, PRESETS);
		expect(forProbe).toBe(forGutterClick);
		expect(forProbe).toBe('vscode://file/{path}:{line}:{col}');
	});
});
