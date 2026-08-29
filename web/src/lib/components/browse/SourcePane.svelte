<script lang="ts">
	// SourcePane renders a BrowseTargetState: idle, loading, a failed named
	// state (never a blank pane — NAV-04), or a file's highlighted verbatim
	// source. This is the ONE unescaped-HTML render site (Svelte's @html
	// directive) in all of web/src — every other view in this repository
	// renders plain text or Svelte's own escaped interpolation.
	import { highlightSource } from '$lib/highlight';
	import { buildCallTargetIndex, callTargets } from '$lib/call-targets';
	import { NAV_INTENT, type BrowseNavDelta, type NavIntent } from '$lib/browse-nav';
	import type { BrowseTargetState } from '$lib/browse-state';
	import type { Node as GraphNode } from '$lib/gen/ui_pb';

	// onNavigate (03-08 Task 1, D-18): a single-def view's `calls` list is
	// the click-to-definition index (BRW-04) — clicking a decorated
	// identifier re-issues navigation by SYMBOL NAME alone (never
	// symbol+file+line), because a click is text-driven and inherently
	// ambiguous; a name with several definitions lands in BRW-05's picker
	// (03-08 Task 3), exactly the same re-resolution path a search
	// selection already goes through. Optional so a caller that renders
	// this pane with no navigation surface (e.g. a future read-only
	// embed) is unaffected.
	let {
		state,
		onNavigate
	}: {
		state: BrowseTargetState;
		onNavigate?: (delta: BrowseNavDelta, intent: NavIntent) => void;
	} = $props();

	function handleCallTargetSelect(entry: readonly GraphNode[]): void {
		if (entry.length === 0) return;
		onNavigate?.({ symbol: entry[0].name }, NAV_INTENT.NAVIGATE);
	}

	// EXTENSION_LANGUAGE: a file-mode GetNodeDetailResponse carries no
	// Node.language field (only NODE_DETAIL_MODE_SINGLE_DEF/MULTI_DEF do,
	// via Node.language) — so for a bare file open, the language is
	// derived here from the path extension, mirroring the same extension
	// lists internal/indexer/languages_*.go registers each LanguageSpec
	// under. This is a rendering-layer best-effort hint, not a second
	// confinement or classification implementation: an unmatched extension
	// falls through to highlightSource's own honest plaintext-escape
	// degrade, never a guess.
	const EXTENSION_LANGUAGE: Record<string, string> = {
		'.c': 'c',
		'.h': 'c',
		'.cpp': 'cpp',
		'.cc': 'cpp',
		'.cxx': 'cpp',
		'.hpp': 'cpp',
		'.hh': 'cpp',
		'.cs': 'csharp',
		'.go': 'go',
		'.java': 'java',
		'.js': 'javascript',
		'.jsx': 'javascript',
		'.mjs': 'javascript',
		'.cjs': 'javascript',
		'.kt': 'kotlin',
		'.kts': 'kotlin',
		'.php': 'php',
		'.py': 'python',
		'.rb': 'ruby',
		'.rs': 'rust',
		'.swift': 'swift',
		'.ts': 'typescript',
		'.tsx': 'tsx'
	};

	function languageForPath(path: string): string {
		const dot = path.lastIndexOf('.');
		if (dot === -1) return '';
		const ext = path.slice(dot).toLowerCase();
		return EXTENSION_LANGUAGE[ext] ?? '';
	}
</script>

{#if state.kind === 'idle'}
	<p class="mt-4 text-sm text-muted-foreground" data-testid="browse-idle">
		Search for a symbol or open a file to see its source.
	</p>
{:else if state.kind === 'loading'}
	<p class="mt-4 text-sm text-muted-foreground" data-testid="browse-loading">Loading…</p>
{:else if state.kind === 'failed'}
	{@const failure = state.failure}
	<p
		class="mt-4 text-sm text-destructive"
		data-testid={`browse-failed-${failure.kind}`}
	>
		{#if failure.kind === 'not-found'}
			Not found: {failure.message}
		{:else if failure.kind === 'invalid-input'}
			Invalid request: {failure.message}
		{:else if failure.kind === 'indexing'}
			The index is being rebuilt: {failure.message}
		{:else}
			Something went wrong: {failure.message}
		{/if}
	</p>
{:else if state.kind === 'file'}
	{@const text = new TextDecoder().decode(state.source)}
	{@const language = languageForPath(state.path)}
	<div class="mt-4" data-testid="browse-source">
		{#if state.truncated}
			<p class="mb-2 text-xs text-muted-foreground" data-testid="browse-truncated">
				Showing first {state.returnedLines} of {state.totalLines} lines.
			</p>
		{/if}
		<pre class="overflow-x-auto rounded border p-4 text-sm"><code>{@html highlightSource(
				text,
				language
			)}</code></pre>
	</div>
{:else if state.kind === 'single-def'}
	{@const source = state.source}
	{@const language = state.node.language}
	{@const callIndex = buildCallTargetIndex(state.calls)}
	<div class="mt-4" data-testid="browse-source">
		{#if source}
			{@const text = new TextDecoder().decode(source.content)}
			{#if source.truncated}
				<p class="mb-2 text-xs text-muted-foreground" data-testid="browse-truncated">
					Showing first {source.returnedLines} of {source.totalLines} lines.
				</p>
			{/if}
			<pre class="overflow-x-auto rounded border p-4 text-sm"><code
					use:callTargets={{ index: callIndex, onSelect: handleCallTargetSelect }}
					>{@html highlightSource(text, language)}</code
				></pre>
		{:else}
			<p class="text-sm text-muted-foreground" data-testid="browse-no-source">
				No source available for this definition.
			</p>
		{/if}
	</div>
{:else if state.kind === 'multi-def'}
	<p class="mt-4 text-sm text-muted-foreground" data-testid="browse-multi-def">
		{state.totalCandidates} definitions found for &quot;{state.symbol}&quot; — a picker is
		not yet built (coming in a later plan).
	</p>
{/if}
