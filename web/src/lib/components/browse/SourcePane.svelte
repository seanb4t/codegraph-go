<script lang="ts">
	// SourcePane renders a BrowseTargetState: idle, loading, a failed named
	// state (never a blank pane — NAV-04), or a file's highlighted verbatim
	// source. This is the ONE unescaped-HTML render site (Svelte's @html
	// directive) in all of web/src — every other view in this repository
	// renders plain text or Svelte's own escaped interpolation.
	import type { MessageInitShape } from '@bufbuild/protobuf';
	import { highlightSource } from '$lib/highlight';
	import { buildCallTargetIndex, callTargets } from '$lib/call-targets';
	import { NAV_INTENT, type BrowseNavDelta, type NavIntent } from '$lib/browse-nav';
	import type { BrowseTargetState } from '$lib/browse-state';
	import {
		PermalinkAvailability,
		type GetPermalinkRequestSchema,
		type GetPermalinkResponse,
		type Node as GraphNode
	} from '$lib/gen/ui_pb';
	import CopyAction from './CopyAction.svelte';

	// PermalinkClient is the minimal shape SourcePane needs from a
	// UIService client — the same declared-independently-of-the-real-
	// client convention browse-target.ts's NodeDetailClient/ImpactClient
	// already use, so a stub can satisfy it in a test with no Connect
	// transport. Passed as a PROP (never imported as the module-level
	// uiClient singleton) for exactly that reason.
	interface PermalinkClient {
		getPermalink(
			request: MessageInitShape<typeof GetPermalinkRequestSchema>,
			options?: { signal?: AbortSignal }
		): Promise<GetPermalinkResponse>;
	}

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
		state: target,
		client,
		onNavigate
	}: {
		state: BrowseTargetState;
		client?: PermalinkClient;
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

	// permalinkParamsFor computes GetPermalink's request shape from the
	// CURRENTLY RENDERED state (D-06, D-09, D-20). File mode carries no
	// line semantics at all — it is "open this whole file" — so its
	// permalink has no anchor. An opened symbol (single-def) carries both
	// a start and an end line and normally requests a RANGE permalink,
	// matching what the UI just showed (D-09).
	//
	// TRUNCATED-FILE DECISION (the open question 03-RESEARCH.md left for
	// this plan to answer, recorded here and in the SUMMARY): when the
	// rendered source was truncated, the permalink is requested with NO
	// anchor at all — not even the single start line — because the whole
	// point of offering the link is "the rest of the file is what the
	// local view could not show" (D-20), and either partial range would
	// still frame the remote view around the truncated portion rather
	// than the whole file. A plain file link is the strict reading of
	// that intent.
	interface PermalinkParams {
		path: string;
		line?: number;
		endLine?: number;
	}

	function permalinkParamsFor(s: BrowseTargetState): PermalinkParams | undefined {
		if (s.kind === 'file') {
			return { path: s.path };
		}
		if (s.kind === 'single-def') {
			if (s.source?.truncated) {
				return { path: s.node.filePath };
			}
			return { path: s.node.filePath, line: s.node.startLine, endLine: s.node.endLine };
		}
		return undefined;
	}

	type PermalinkState =
		| { kind: 'idle' }
		| { kind: 'loading' }
		| { kind: 'loaded'; response: GetPermalinkResponse };

	let permalinkState = $state<PermalinkState>({ kind: 'idle' });

	$effect(() => {
		const params = permalinkParamsFor(target);
		// No client (a caller that renders this pane with no permalink
		// surface, e.g. an existing test fixture predating this plan) is
		// treated exactly like "no params": idle, no permalink section
		// renders. Never a thrown error over an optional integration.
		if (!params || !client) {
			permalinkState = { kind: 'idle' };
			return;
		}
		permalinkState = { kind: 'loading' };
		const controller = new AbortController();
		client
			.getPermalink(
				{ path: params.path, line: params.line, endLine: params.endLine },
				{ signal: controller.signal }
			)
			.then((response) => {
				permalinkState = { kind: 'loaded', response };
			})
			.catch(() => {
				// A failed permalink lookup is not the whole pane's failure —
				// it only means no permalink surface renders for this target.
				permalinkState = { kind: 'idle' };
			});
		return () => controller.abort();
	});
</script>

{#snippet permalinkSurface()}
	{#if permalinkState.kind === 'loaded'}
		{@const p = permalinkState.response}
		{#if p.availability === PermalinkAvailability.LINKABLE}
			<a
				href={p.url}
				target="_blank"
				rel="noreferrer"
				class="text-xs underline decoration-dotted underline-offset-2"
				data-testid="permalink-linkable"
			>
				View on GitHub
			</a>
		{:else if p.availability === PermalinkAvailability.LINKABLE_UNVERIFIED}
			<span class="text-xs text-muted-foreground" data-testid="permalink-unverified">
				<a href={p.url} target="_blank" rel="noreferrer" class="underline decoration-dotted underline-offset-2"
					>View on GitHub</a
				>
				(unverified: {p.reason})
			</span>
		{:else if p.availability === PermalinkAvailability.NO_LINK}
			<span class="text-xs text-muted-foreground" data-testid="permalink-no-link">
				No permalink available: {p.reason}
			</span>
		{/if}
	{/if}
{/snippet}

{#if target.kind === 'idle'}
	<p class="mt-4 text-sm text-muted-foreground" data-testid="browse-idle">
		Search for a symbol or open a file to see its source.
	</p>
{:else if target.kind === 'loading'}
	<p class="mt-4 text-sm text-muted-foreground" data-testid="browse-loading">Loading…</p>
{:else if target.kind === 'failed'}
	{@const failure = target.failure}
	<p class="mt-4 text-sm text-destructive" data-testid={`browse-failed-${failure.kind}`}>
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
{:else if target.kind === 'file'}
	{@const text = new TextDecoder().decode(target.source)}
	{@const language = languageForPath(target.path)}
	<div class="mt-4" data-testid="browse-source">
		<div class="mb-2 flex flex-wrap items-center gap-3">
			<CopyAction value={target.path} label="file path" />
			{@render permalinkSurface()}
		</div>
		{#if target.truncated}
			<p class="mb-2 text-xs text-muted-foreground" data-testid="browse-truncated">
				Showing first {target.returnedLines} of {target.totalLines} lines.
			</p>
		{/if}
		<pre class="overflow-x-auto rounded border p-4 text-sm"><code>{@html highlightSource(
				text,
				language
			)}</code></pre>
	</div>
{:else if target.kind === 'single-def'}
	{@const source = target.source}
	{@const language = target.node.language}
	{@const callIndex = buildCallTargetIndex(target.calls)}
	<div class="mt-4" data-testid="browse-source">
		<div class="mb-2 flex flex-wrap items-center gap-3">
			<CopyAction value={target.node.filePath} label="file path" />
			<CopyAction value={target.node.name} label="symbol name" />
			{@render permalinkSurface()}
		</div>
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
{/if}
<!-- 'multi-def' has NO branch here (03-08, closes WINDOWS.md entry #23):
     web/src/routes/browse/+page.svelte routes that state to
     DefinitionPicker.svelte instead of mounting SourcePane at all, so
     this component is never asked to render it. -->

