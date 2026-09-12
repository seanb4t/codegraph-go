// editor-prefs.ts — 09-04 Task 1 RED stub. Exports declared with the
// exact shape the plan's <interfaces> requires, returning wrong answers
// (null/undefined, no-ops) so editor-prefs.test.ts fails on assertions,
// not module resolution. The real implementation lands in the GREEN
// commit.
import type { EditorPreset } from './gen/ui_pb';

export const EDITOR_OVERRIDE_STORAGE_KEY = 'codegraph.editorLink.override';

export type EditorOverride = { kind: 'preset'; id: string } | { kind: 'custom'; template: string };

export function readEditorOverride(_storage?: Storage | null): EditorOverride | null {
	return null;
}

export function writeEditorOverride(_override: EditorOverride, _storage?: Storage | null): void {
	// RED stub: no-op.
}

export function clearEditorOverride(_storage?: Storage | null): void {
	// RED stub: no-op.
}

export function templateForRequest(
	_override: EditorOverride | null,
	_presets: readonly EditorPreset[]
): string | undefined {
	return undefined;
}
