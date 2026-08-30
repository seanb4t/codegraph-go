<script lang="ts">
	// FilePicker is WRK-02's search-and-add affordance: pick several files
	// at once, see them as removable chips, and drive Affected off the
	// resulting set (04-06). It composes the vendored Command primitive
	// exactly as SearchPanel.svelte does (shouldFilter={false} — results
	// are already server-filtered and server-ordered, D-15) over Task 1's
	// file-search.ts controller, the SAME debounce/abort/identity
	// mechanism the symbol search uses, configured a second time.
	//
	// Props-in/callback-out, no internal selection state (mirrors
	// NeighborsPanel.svelte's shape): `files` comes in, `onChange` goes
	// out, and the route holds the single source of truth in
	// workbench-url.ts's `files` array. This is what keeps the URL and
	// this component's rendering from ever developing two different
	// ideas of what is selected.
	import { untrack } from 'svelte';
	import { get } from 'svelte/store';
	import * as Command from '$lib/components/ui/command/index.js';
	import {
		createFileSearchController,
		type FilesClient,
		type FileSearchState
	} from '$lib/file-search';

	let {
		client,
		files,
		onChange
	}: {
		client: FilesClient;
		files: string[];
		onChange: (files: string[]) => void;
	} = $props();

	// untrack: the controller is built ONCE from whatever client is
	// current at mount (SearchPanel.svelte's own convention) — this
	// component does not support swapping the client for a live
	// instance.
	const controller = untrack(() => createFileSearchController(client));
	let searchState = $state<FileSearchState>(get(controller));

	$effect(() => {
		const unsubscribe = controller.subscribe((next) => {
			searchState = next;
		});
		return unsubscribe;
	});

	// IN-13: dispose the controller's own async surface on unmount — a
	// cleanup-only effect, matching search.ts's callers.
	$effect(() => {
		return () => controller.dispose();
	});

	function handleInputChange(value: string): void {
		controller.setQuery(value);
	}

	// De-duplicate on append by exact string equality (D-15's chosen
	// interaction model) — array order is what serializes to the URL, so
	// this never sorts or otherwise reorders the caller's array.
	function addFile(path: string): void {
		if (files.includes(path)) return;
		onChange([...files, path]);
	}

	function removeFile(path: string): void {
		onChange(files.filter((f) => f !== path));
	}
</script>

<div data-testid="file-picker">
	<Command.Root shouldFilter={false}>
		<Command.Input
			value={searchState.query}
			oninput={(e: Event) => handleInputChange((e.currentTarget as HTMLInputElement).value)}
			placeholder="Search files to add..."
			data-testid="file-picker-input"
		/>
		<Command.List>
			<Command.Empty>No results.</Command.Empty>
			{#if searchState.results.length > 0}
				<Command.Group heading="Files">
					{#each searchState.results as entry (entry.path)}
						<Command.Item
							value={entry.path}
							data-testid={`file-picker-result-${entry.path}`}
							onSelect={() => addFile(entry.path)}
						>
							{entry.path}
						</Command.Item>
					{/each}
				</Command.Group>
			{/if}
		</Command.List>
	</Command.Root>

	<ul class="mt-2 flex flex-wrap gap-2" data-testid="file-picker-chips">
		{#each files as path (path)}
			<li
				class="flex items-center gap-1 rounded-md border border-input bg-muted px-2 py-1 text-sm"
				data-testid={`file-picker-chip-${path}`}
			>
				<span>{path}</span>
				<button
					type="button"
					aria-label={`Remove ${path}`}
					data-testid={`file-picker-remove-${path}`}
					onclick={() => removeFile(path)}
				>
					&times;
				</button>
			</li>
		{/each}
	</ul>
</div>
