<script lang="ts">
	// EditorLinkPicker.svelte — 09-04 Task 3 RED stub. Exports declared
	// with the exact prop shape <interfaces> requires; renders nothing so
	// editor-link-picker.test.ts fails on missing test ids, not module
	// resolution. The real implementation lands in the GREEN commit.
	import type { EditorOverride } from '$lib/editor-prefs';
	import type { GetEditorLinkResponse } from '$lib/gen/ui_pb';

	let {
		response: _response,
		override: _override,
		onChoosePreset: _onChoosePreset,
		onApplyCustom: _onApplyCustom,
		onUseServerDefault: _onUseServerDefault
	}: {
		response: GetEditorLinkResponse;
		override: EditorOverride | null;
		onChoosePreset: (id: string) => void;
		onApplyCustom: (template: string) => void;
		onUseServerDefault: () => void;
	} = $props();
</script>
