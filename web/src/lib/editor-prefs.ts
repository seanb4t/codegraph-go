// editor-prefs.ts — 09-04 Task 1: the per-browser editor-link override.
// This is the SPA's FIRST localStorage use (D-18, 09-PATTERNS.md § this
// file — no analog exists elsewhere in web/src). The server persists
// nothing about this preference (SRV-03) — the override rides on every
// GetEditorLinkRequest.template instead (D-06), so nothing here needs
// resetting or migrating server-side. Every Storage access is wrapped
// in try/catch: a missing, disabled or throwing Storage must degrade to
// "no override" and the pane must render correctly without it — never
// throw. templateForRequest is the ONE function that turns a stored
// preference into a request's template field; both the load-time probe
// and a gutter click call it so they always resolve identically for the
// same override (the assumption-delta invariant this plan locks).
import type { EditorPreset } from './gen/ui_pb';

export const EDITOR_OVERRIDE_STORAGE_KEY = 'codegraph.editorLink.override';

export type EditorOverride = { kind: 'preset'; id: string } | { kind: 'custom'; template: string };

// resolveStorage: when no storage argument is passed, fall back to
// globalThis.localStorage — but even READING that property can throw in
// some browsers when storage is disabled, so the property access itself
// is inside the try, not just the calls made against it. An explicit
// null/undefined argument is honored as-is (never silently upgraded to
// the global), matching the documented "storage is null/undefined"
// degrade case.
function resolveStorage(storage: Storage | null | undefined): Storage | null {
	if (storage !== undefined) return storage;
	try {
		return globalThis.localStorage ?? null;
	} catch {
		return null;
	}
}

function isValidOverride(value: unknown): value is EditorOverride {
	if (typeof value !== 'object' || value === null) return false;
	const v = value as Record<string, unknown>;
	if (v.kind === 'preset') return typeof v.id === 'string' && v.id.length > 0;
	if (v.kind === 'custom') return typeof v.template === 'string' && v.template.length > 0;
	return false;
}

export function readEditorOverride(storage?: Storage | null): EditorOverride | null {
	const s = resolveStorage(storage);
	if (!s) return null;
	try {
		const raw = s.getItem(EDITOR_OVERRIDE_STORAGE_KEY);
		if (raw === null) return null;
		const parsed: unknown = JSON.parse(raw);
		return isValidOverride(parsed) ? parsed : null;
	} catch {
		// Absent key, malformed JSON, a shape other than the two allowed
		// kinds, or a throwing getItem all degrade identically: no override.
		return null;
	}
}

export function writeEditorOverride(override: EditorOverride, storage?: Storage | null): void {
	const s = resolveStorage(storage);
	if (!s) return;
	try {
		s.setItem(EDITOR_OVERRIDE_STORAGE_KEY, JSON.stringify(override));
	} catch {
		// Quota exceeded or a disabled Storage — the preference simply does
		// not persist this time; never throw over an optional integration.
	}
}

export function clearEditorOverride(storage?: Storage | null): void {
	const s = resolveStorage(storage);
	if (!s) return;
	try {
		s.removeItem(EDITOR_OVERRIDE_STORAGE_KEY);
	} catch {
		// Degrade silently, same discipline as write.
	}
}

// templateForRequest: null -> undefined (send no template, the server's
// effective default applies); a preset override resolves to that
// preset's wire template from the CURRENT presets list, or undefined
// when the id is unknown (never fabricate a string); a custom override
// passes through verbatim — no client-side placeholder-substitution
// scan, the server decides via the same validator it applies to its own
// flag/env value.
export function templateForRequest(
	override: EditorOverride | null,
	presets: readonly EditorPreset[]
): string | undefined {
	if (!override) return undefined;
	if (override.kind === 'custom') return override.template;
	return presets.find((p) => p.id === override.id)?.template;
}
