<script lang="ts">
	// SourcePane renders a BrowseTargetState: idle, loading, a failed named
	// state (never a blank pane — NAV-04), or a file's highlighted verbatim
	// source. This is the ONE unescaped-HTML render site (Svelte's @html
	// directive) in all of web/src — every other view in this repository
	// renders plain text or Svelte's own escaped interpolation.
	import { highlightSource } from '$lib/highlight';
	import type { BrowseTargetState } from '$lib/browse-state';

	let { state }: { state: BrowseTargetState } = $props();

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
{:else}
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
{/if}
