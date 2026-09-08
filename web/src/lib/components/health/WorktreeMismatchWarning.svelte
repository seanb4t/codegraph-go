<script lang="ts">
	// WorktreeMismatchWarning is HLT-03's "impossible to miss" signal —
	// the loudest warning in the app. role="alert" (not "status", which
	// TrustVerdict and StatusBanner use): an alert is announced to
	// assistive technology the moment it mounts, which is what
	// "impossible to miss" means for a screen-reader user. Its
	// border/background pair follows the same border-*/bg-*/text-*
	// idiom StatusBanner.svelte's degraded branches use, but in a color
	// distinct from every one of them (StatusBanner has no red branch)
	// so this reads as MORE urgent than the ordinary stale banner it sits
	// above on this page.
	//
	// Both worktreeRoot and indexRoot are host-absolute paths — the ONE
	// scoped exception to this app's path-privacy stance (T-04-18,
	// accepted): the warning is useless without naming both trees.
	let { worktreeRoot, indexRoot }: { worktreeRoot: string; indexRoot: string } = $props();
</script>

<div
	class="rounded-md border-2 border-red-600 bg-red-100 px-4 py-3 text-sm font-semibold text-red-900"
	role="alert"
	data-testid="health-worktree-mismatch"
>
	<p>This index was built from a different working tree than the one you are viewing.</p>
	<p class="mt-1 font-normal">
		Working tree: <code>{worktreeRoot}</code>
	</p>
	<p class="font-normal">
		Index root: <code>{indexRoot}</code>
	</p>
</div>
