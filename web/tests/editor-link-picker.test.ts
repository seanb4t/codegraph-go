// editor-link-picker.test.ts — 09-04 Task 3: EditorLinkPicker.svelte, a
// pure view over a GetEditorLinkResponse and the current EditorOverride
// (D-18). Written and run RED before the component exists.
import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';

import EditorLinkPicker from '$lib/components/browse/EditorLinkPicker.svelte';
import { EditorLinkAvailability, EditorTemplateSource } from '$lib/gen/ui_pb';
import type { EditorOverride } from '$lib/editor-prefs';
import type { EditorPreset, GetEditorLinkResponse } from '$lib/gen/ui_pb';

const PRESETS: EditorPreset[] = [
	{ id: 'vscode', name: 'VS Code', template: 'vscode://file/{path}:{line}:{col}' } as EditorPreset,
	{ id: 'cursor', name: 'Cursor', template: 'cursor://file/{path}:{line}:{col}' } as EditorPreset,
	{
		id: 'jetbrains',
		name: 'JetBrains',
		template: 'jetbrains://gateway/navigate/reference?path={path}&line={line}'
	} as EditorPreset
];

function response(overrides: Partial<GetEditorLinkResponse> = {}): GetEditorLinkResponse {
	return {
		url: '',
		availability: EditorLinkAvailability.NO_TEMPLATE,
		reason: '',
		defaultSource: EditorTemplateSource.NONE,
		defaultEditor: '',
		overrideApplied: false,
		presets: PRESETS,
		...overrides
	} as GetEditorLinkResponse;
}

function renderPicker(
	overrides: {
		response?: GetEditorLinkResponse;
		override?: EditorOverride | null;
		onChoosePreset?: (id: string) => void;
		onApplyCustom?: (template: string) => void;
		onUseServerDefault?: () => void;
	} = {}
) {
	return render(EditorLinkPicker, {
		props: {
			response: overrides.response ?? response(),
			override: overrides.override ?? null,
			onChoosePreset: overrides.onChoosePreset ?? vi.fn(),
			onApplyCustom: overrides.onApplyCustom ?? vi.fn(),
			onUseServerDefault: overrides.onUseServerDefault ?? vi.fn()
		}
	});
}

describe('EditorLinkPicker: renders exactly the wire\'s three presets, a custom row and provenance', () => {
	it('renders three preset buttons named from the wire, the custom row, and the default-use button', () => {
		renderPicker();
		expect(screen.getByTestId('editor-preset-vscode').textContent?.trim()).toBe('VS Code');
		expect(screen.getByTestId('editor-preset-cursor').textContent?.trim()).toBe('Cursor');
		expect(screen.getByTestId('editor-preset-jetbrains').textContent?.trim()).toBe('JetBrains');
		expect(screen.getByTestId('editor-custom-template')).toBeTruthy();
		expect(screen.getByTestId('editor-custom-apply')).toBeTruthy();
		expect(screen.getByTestId('editor-use-server-default')).toBeTruthy();
	});

	it('declares aria-modal="true" on its dialog root (WR-03)', () => {
		renderPicker();
		const dialog = screen.getByTestId('editor-link-picker');
		expect(dialog.getAttribute('role')).toBe('dialog');
		expect(dialog.getAttribute('aria-modal')).toBe('true');
	});

	it.each([
		[EditorTemplateSource.FLAG, '', 'flag'],
		[EditorTemplateSource.ENV, '', 'environment variable'],
		[EditorTemplateSource.DISCOVERED, 'goland', 'discovered: goland'],
		[EditorTemplateSource.NONE, '', 'none'],
		[EditorTemplateSource.DISABLED, '', 'disabled by operator']
	])('maps defaultSource %s to provenance text containing %s', (defaultSource, defaultEditor, expected) => {
		renderPicker({ response: response({ defaultSource, defaultEditor }) });
		expect(screen.getByTestId('editor-default-provenance').textContent).toContain(expected);
	});
});

describe('EditorLinkPicker: active-choice display', () => {
	it('marks the matching preset aria-pressed=true and the others false with a preset override', () => {
		renderPicker({ override: { kind: 'preset', id: 'cursor' } });
		expect(screen.getByTestId('editor-preset-cursor').getAttribute('aria-pressed')).toBe('true');
		expect(screen.getByTestId('editor-preset-vscode').getAttribute('aria-pressed')).toBe('false');
		expect(screen.getByTestId('editor-preset-jetbrains').getAttribute('aria-pressed')).toBe('false');
	});

	it('shows the custom template in the input with no preset pressed, for a custom override', () => {
		renderPicker({ override: { kind: 'custom', template: 'x://{path}' } });
		expect((screen.getByTestId('editor-custom-template') as HTMLInputElement).value).toBe(
			'x://{path}'
		);
		for (const id of ['vscode', 'cursor', 'jetbrains']) {
			expect(screen.getByTestId(`editor-preset-${id}`).getAttribute('aria-pressed')).toBe('false');
		}
	});

	it('marks "Use server default" pressed and no preset pressed for a null override', () => {
		renderPicker({ override: null });
		expect(screen.getByTestId('editor-use-server-default').getAttribute('aria-pressed')).toBe(
			'true'
		);
		for (const id of ['vscode', 'cursor', 'jetbrains']) {
			expect(screen.getByTestId(`editor-preset-${id}`).getAttribute('aria-pressed')).toBe('false');
		}
	});
});

describe('EditorLinkPicker: callbacks', () => {
	it('calls onChoosePreset(id) once on a preset click', async () => {
		const onChoosePreset = vi.fn();
		renderPicker({ onChoosePreset });
		await fireEvent.click(screen.getByTestId('editor-preset-jetbrains'));
		expect(onChoosePreset).toHaveBeenCalledTimes(1);
		expect(onChoosePreset).toHaveBeenCalledWith('jetbrains');
	});

	it('calls onApplyCustom(template) with the typed value on apply click', async () => {
		const onApplyCustom = vi.fn();
		renderPicker({ onApplyCustom });
		const input = screen.getByTestId('editor-custom-template');
		await fireEvent.input(input, { target: { value: 'x://{path}' } });
		await fireEvent.click(screen.getByTestId('editor-custom-apply'));
		expect(onApplyCustom).toHaveBeenCalledWith('x://{path}');
	});

	it('calls onUseServerDefault on that button\'s click', async () => {
		const onUseServerDefault = vi.fn();
		renderPicker({ onUseServerDefault });
		await fireEvent.click(screen.getByTestId('editor-use-server-default'));
		expect(onUseServerDefault).toHaveBeenCalledTimes(1);
	});
});

describe('EditorLinkPicker: TEMPLATE_INVALID keeps the value editable', () => {
	it('shows the reason inline and keeps the invalid custom value in the input, not cleared', () => {
		renderPicker({
			response: response({
				availability: EditorLinkAvailability.TEMPLATE_INVALID,
				reason: 'unknown scheme: zed',
				overrideApplied: true
			}),
			override: { kind: 'custom', template: 'zed://file/{path}' }
		});
		expect(screen.getByText(/unknown scheme: zed/)).toBeTruthy();
		expect((screen.getByTestId('editor-custom-template') as HTMLInputElement).value).toBe(
			'zed://file/{path}'
		);
	});
});

describe('EditorLinkPicker: Zed is nowhere', () => {
	it('renders no text containing "zed" (case-insensitive) for any response', () => {
		const { container } = renderPicker({
			response: response({
				availability: EditorLinkAvailability.TEMPLATE_INVALID,
				reason: 'unknown scheme: zed',
				overrideApplied: true
			})
		});
		// The reason string itself legitimately names the rejected scheme
		// ("zed") when a server answer reports it — that is DATA the
		// server sent, not UI copy. Excluding the reason paragraph, no
		// picker-authored string may say "zed".
		const reasonNode = screen.queryByText(/unknown scheme: zed/);
		const clone = container.cloneNode(true) as HTMLElement;
		if (reasonNode) {
			const match = clone.querySelector('[data-testid="editor-link-picker-invalid-reason"]');
			match?.remove();
		}
		expect(clone.textContent?.toLowerCase()).not.toContain('zed');
	});
});
