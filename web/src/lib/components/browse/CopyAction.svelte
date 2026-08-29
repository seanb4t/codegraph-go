<script lang="ts">
	// CopyAction is BRW-07's one-action clipboard affordance (D-Claude's
	// Discretion: a click-to-copy button). It writes EXACTLY the string it
	// was handed — no trim, no case-fold, no normalization — because the
	// whole point is that the copied value matches the exact string shown
	// on screen, untouched. For an EMPTY value it renders NOTHING, not a
	// disabled control: a present-but-inert affordance promises an action
	// it cannot perform, which is a worse answer than its plain absence.
	let {
		value,
		label
	}: {
		value: string;
		// label distinguishes what is being copied ("file path" vs "symbol
		// name") in the accessible name, since a source pane can show both
		// in the same view.
		label: string;
	} = $props();

	async function handleCopy(): Promise<void> {
		await navigator.clipboard.writeText(value);
	}
</script>

{#if value}
	<button
		type="button"
		class="text-xs text-muted-foreground underline decoration-dotted underline-offset-2"
		aria-label={`Copy ${label}`}
		data-testid={`copy-action-${label}`}
		onclick={handleCopy}
	>
		Copy {label}
	</button>
{/if}
