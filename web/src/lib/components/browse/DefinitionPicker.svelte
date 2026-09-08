<script lang="ts">
	// DefinitionPicker is BRW-05's disambiguation surface: a bare symbol
	// name that resolves to several definitions offers every candidate the
	// server RETURNED, states the server's TRUE total alongside, and lets
	// the developer choose — it never guesses. Mounted from
	// web/src/routes/browse/+page.svelte for BrowseTargetState's
	// 'multi-def' variant, replacing the placeholder line SourcePane's
	// multi-def branch used to render (03-07's WINDOWS.md entry #23,
	// closed by this file).
	//
	// Two caps are in play and this component never conflates them
	// (D-21): `totalCandidates` is the TRUE total match count regardless
	// of how many were returned; `definitions.length` is what the server
	// actually SENT. Today's handler (internal/uiserver/handlers.go's
	// nodeDetailToProto) returns a `definitions` entry for every match and
	// caps only DETAIL GATHERING — so the two numbers are equal in
	// practice — but this component renders from BOTH numbers, never from
	// the assumption they agree, so it stays correct the day a future
	// server change caps what is returned too.
	import type { NodeDefinition } from '$lib/gen/ui_pb';
	import { NAV_INTENT, type BrowseNavDelta, type NavIntent } from '$lib/browse-nav';

	let {
		symbol,
		definitions,
		totalCandidates,
		onNavigate
	}: {
		symbol: string;
		definitions: NodeDefinition[];
		totalCandidates: number;
		onNavigate: (delta: BrowseNavDelta, intent: NavIntent) => void;
	} = $props();

	let listEl: HTMLUListElement | undefined = $state();

	function candidateKey(def: NodeDefinition, index: number): string {
		// The index is part of the key (not merely a fallback) because two
		// listed-only candidates in a large result set could in principle
		// share file+line+name; position is the one thing guaranteed
		// unique across a static, never-reordered list.
		const n = def.node;
		return n ? `${n.filePath}:${n.startLine}:${n.name}:${index}` : `unknown:${index}`;
	}

	// selectCandidate navigates with ALL THREE of symbol, file and line —
	// never file+line alone. Dropping symbol would silently defeat
	// BRW-05: buildNodeDetail (internal/query/detail.go:190-200) branches
	// on `symbol == ""` FIRST and returns NodeDetailModeFile, reopening
	// the whole FILE rather than the picked definition. With all three
	// set, enumerateSymbolDefs yields the candidates and file/line are
	// NODE-03 narrowing hints (detail.go:222-230) that narrow them back
	// to exactly this one, producing NodeDetailModeSingleDef — the only
	// URL shape that can express "this overload, not that one". This is
	// the deliberate, narrow exception to browse-nav.ts's own symbol/file
	// clearing rule (the disambiguation triple), which that module's own
	// applyTargetClearing already carves out.
	function selectCandidate(def: NodeDefinition): void {
		const n = def.node;
		if (!n) return;
		onNavigate({ symbol: n.name, file: n.filePath, line: n.startLine }, NAV_INTENT.NAVIGATE);
	}

	// Arrow-key traversal between candidates, mirroring the search
	// surface's own keyboard drivability (NAV-03) — this picker is a
	// second place a developer chooses among several results, and it
	// should feel the same. Enter is native: every candidate is a real
	// <button>, so Enter/Space already activate it with no extra wiring.
	// The listener lives on each BUTTON (an inherently interactive
	// element), never on the surrounding <ul>, which a11y linting
	// correctly flags for keyboard handlers on a non-interactive list.
	function handleCandidateKeydown(e: KeyboardEvent): void {
		if (e.key !== 'ArrowDown' && e.key !== 'ArrowUp') return;
		if (!listEl) return;
		const buttons = Array.from(
			listEl.querySelectorAll<HTMLButtonElement>('button[data-candidate]')
		);
		if (buttons.length === 0) return;
		const currentIndex = buttons.findIndex((b) => b === document.activeElement);
		e.preventDefault();
		if (currentIndex === -1) {
			buttons[0]?.focus();
			return;
		}
		const delta = e.key === 'ArrowDown' ? 1 : -1;
		const nextIndex = Math.min(Math.max(currentIndex + delta, 0), buttons.length - 1);
		buttons[nextIndex]?.focus();
	}
</script>

<div class="mt-4" data-testid="definition-picker">
	<p class="mb-2 text-sm text-muted-foreground" data-testid="definition-picker-summary">
		{definitions.length} candidate{definitions.length === 1 ? '' : 's'} for &quot;{symbol}&quot;
		{#if totalCandidates > definitions.length}
			<span data-testid="definition-picker-omitted-count">
				— showing {definitions.length} of {totalCandidates}; {totalCandidates -
					definitions.length} more not shown</span
			>
		{/if}
	</p>
	<ul bind:this={listEl} data-testid="definition-picker-list">
		{#each definitions as def, i (candidateKey(def, i))}
			{@const n = def.node}
			<li>
				<button
					type="button"
					data-candidate
					data-testid={`definition-candidate-${i}`}
					onclick={() => selectCandidate(def)}
					onkeydown={handleCandidateKeydown}
				>
					{n?.name}
					<span class="text-muted-foreground">{n?.kind} · {n?.filePath}:{n?.startLine}</span>
					{#if !def.detailGathered}
						<span
							class="text-xs text-muted-foreground"
							data-testid={`definition-candidate-ungathered-${i}`}
						>
							not fully loaded
						</span>
					{/if}
				</button>
			</li>
		{/each}
	</ul>
</div>
