<script lang="ts">
	// EditorLinkPicker.svelte — 09-04 Task 3 (D-18): a pure view over a
	// GetEditorLinkResponse and the current EditorOverride. It holds no
	// rpc client and no storage access — SourcePane owns both, calling
	// this component's callbacks and writing through editor-prefs.ts
	// itself. Presets render from response.presets ONLY (the wire's own
	// three choices, D-06) — this file never hardcodes a preset table or
	// a scheme literal, and no rendered string here may name the editor
	// this phase deliberately excludes (D-18, BRW-12). There is
	// deliberately NO vendored popover
	// family in this codebase (04-07's web:components:drift gate covers
	// only what IS vendored), so this is a plain toggled panel over the
	// vendored Button/Input families, not a new dependency.
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { EditorLinkAvailability, EditorTemplateSource } from '$lib/gen/ui_pb';
	import type { EditorOverride } from '$lib/editor-prefs';
	import type { EditorPreset, GetEditorLinkResponse } from '$lib/gen/ui_pb';

	let {
		response,
		override,
		onChoosePreset,
		onApplyCustom,
		onUseServerDefault
	}: {
		response: GetEditorLinkResponse;
		override: EditorOverride | null;
		onChoosePreset: (id: string) => void;
		onApplyCustom: (template: string) => void;
		onUseServerDefault: () => void;
	} = $props();

	// customValue: the field's own typed-but-not-yet-applied text. Seeded
	// from the CURRENT override at mount so a custom override's template
	// (valid or TEMPLATE_INVALID) is visible and stays editable — never
	// cleared or replaced by an invalid answer (the plan's own
	// prohibition on silently discarding what the user typed).
	// svelte-ignore state_referenced_locally -- deliberate: this reads
	// `override` only ONCE, at mount, to seed the field's local editable
	// buffer. Making it a $derived would instead re-sync on every
	// override change, discarding whatever the user is mid-typing.
	let customValue = $state(override?.kind === 'custom' ? override.template : '');

	function isPresetActive(preset: EditorPreset): boolean {
		return override?.kind === 'preset' && override.id === preset.id;
	}

	// needsAssumedNote: Cursor's and JetBrains' URI syntax are
	// community-sourced, never officially documented, unlike VS Code's
	// (09-RESEARCH.md Assumptions A1/A2) — the note is removed only once
	// an end-of-phase human check confirms them against a real install.
	function needsAssumedNote(preset: EditorPreset): boolean {
		return preset.id === 'cursor' || preset.id === 'jetbrains';
	}

	// A plain string, not an inline template literal: Svelte's own
	// `{...}` attribute syntax would otherwise try to evaluate a bare
	// `{path}` in the markup below as a JS expression.
	const CUSTOM_TEMPLATE_PLACEHOLDER = 'scheme://…{path}:{line}:{col}';

	function provenanceText(): string {
		switch (response.defaultSource) {
			case EditorTemplateSource.FLAG:
				return 'flag';
			case EditorTemplateSource.ENV:
				return 'environment variable';
			case EditorTemplateSource.DISCOVERED:
				return `discovered: ${response.defaultEditor}`;
			case EditorTemplateSource.NONE:
				return 'none';
			case EditorTemplateSource.DISABLED:
				return 'disabled by operator';
			default:
				return 'unknown';
		}
	}
</script>

<div
	data-testid="editor-link-picker"
	role="dialog"
	aria-modal="true"
	aria-label="Editor link"
	class="mt-2 flex flex-col gap-3 rounded border bg-background p-3 text-xs"
>
	<p class="font-medium">Open in editor for this browser</p>
	<div class="flex flex-wrap items-start gap-3">
		{#each response.presets as preset (preset.id)}
			<div class="flex flex-col gap-1">
				<Button
					type="button"
					variant="outline"
					size="sm"
					data-testid={`editor-preset-${preset.id}`}
					aria-pressed={isPresetActive(preset)}
					onclick={() => onChoosePreset(preset.id)}
				>
					{preset.name}
				</Button>
				{#if needsAssumedNote(preset)}
					<span class="max-w-40 text-[10px] text-muted-foreground">
						URI scheme not yet verified against a real install
					</span>
				{/if}
			</div>
		{/each}
	</div>
	<div class="flex flex-col gap-1">
		<label class="text-muted-foreground" for="editor-link-picker-custom-template">
			Custom template
		</label>
		<div class="flex items-center gap-2">
			<Input
				id="editor-link-picker-custom-template"
				type="text"
				data-testid="editor-custom-template"
				placeholder={CUSTOM_TEMPLATE_PLACEHOLDER}
				bind:value={customValue}
			/>
			<Button
				type="button"
				variant="outline"
				size="sm"
				data-testid="editor-custom-apply"
				onclick={() => onApplyCustom(customValue)}
			>
				Apply
			</Button>
		</div>
		{#if response.availability === EditorLinkAvailability.TEMPLATE_INVALID}
			<p class="text-destructive" data-testid="editor-link-picker-invalid-reason">
				{response.reason}
			</p>
		{/if}
	</div>
	<Button
		type="button"
		variant="outline"
		size="sm"
		data-testid="editor-use-server-default"
		aria-pressed={override === null}
		onclick={onUseServerDefault}
	>
		Use server default
	</Button>
	<p class="text-muted-foreground" data-testid="editor-default-provenance">
		Server default: {provenanceText()}
	</p>
</div>
