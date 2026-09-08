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

	// copyState (WR-09): handleCopy previously had no try/catch and no
	// success/failure surface — every clipboard rejection became an
	// unhandled promise rejection and a silent no-op, exactly the pattern
	// this project's own CLAUDE.md forbids ("MUST surface problems
	// clearly, never hide them") and this component's own doc comment
	// argues against ("a present-but-inert affordance promises an action
	// it cannot perform"). Two ordinary failures reach here:
	// navigator.clipboard is undefined outside a secure context (D-08's
	// bind-address field is explicitly meant to be wired to a non-
	// loopback address later), and writeText rejects with
	// NotAllowedError when the document is not focused. Both now surface
	// a visible, transient state instead of a console-only failure.
	let copyState = $state<'idle' | 'copied' | 'failed'>('idle');
	let resetTimer: ReturnType<typeof setTimeout> | undefined;

	// IN-07: WR-09's fix schedules resetTimer after every copy attempt and
	// only ever clears it on the NEXT click (the line above, inside
	// handleCopy). Unmounting within the 1.5s window left the timer
	// pending, later assigning copyState on a destroyed component — a
	// cleanup-only effect (nothing inside is reactively read, so this
	// runs exactly once, on unmount) closes that gap.
	$effect(() => () => clearTimeout(resetTimer));

	async function handleCopy(): Promise<void> {
		clearTimeout(resetTimer);
		try {
			if (!navigator.clipboard) {
				throw new Error('clipboard API unavailable in this context');
			}
			await navigator.clipboard.writeText(value);
			copyState = 'copied';
		} catch {
			copyState = 'failed';
		}
		resetTimer = setTimeout(() => {
			copyState = 'idle';
		}, 1500);
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
		{#if copyState === 'copied'}
			Copied
		{:else if copyState === 'failed'}
			Copy failed
		{:else}
			Copy {label}
		{/if}
	</button>
{/if}
