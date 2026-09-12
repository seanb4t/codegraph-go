<script lang="ts">
	// SourcePane renders a BrowseTargetState: idle, loading, a failed named
	// state (never a blank pane — NAV-04), or a file's highlighted verbatim
	// source. This is the ONE unescaped-HTML render site (Svelte's @html
	// directive) in all of web/src — every other view in this repository
	// renders plain text or Svelte's own escaped interpolation. Source is
	// rendered as one DOM row per line (09-03, BRW-10/D-04's consequence)
	// via splitHighlightedLines ($lib/source-lines) rather than a single
	// blob — both the sticky breadcrumb below and plan 09-04's editor-link
	// gutter depend on that per-line addressability.
	import { untrack } from 'svelte';
	import type { MessageInitShape } from '@bufbuild/protobuf';
	import { highlightSource, languageForPath } from '$lib/highlight';
	import { splitHighlightedLines } from '$lib/source-lines';
	import { innermostSymbolAt, firstFullyVisibleLine } from '$lib/breadcrumb';
	import { buildCallTargetIndex, callTargets } from '$lib/call-targets';
	import { NAV_INTENT, type BrowseNavDelta, type NavIntent } from '$lib/browse-nav';
	import type { BrowseTargetState } from '$lib/browse-state';
	import {
		readEditorOverride,
		writeEditorOverride,
		clearEditorOverride,
		templateForRequest
	} from '$lib/editor-prefs';
	import type { EditorOverride } from '$lib/editor-prefs';
	import {
		PermalinkAvailability,
		EditorLinkAvailability,
		EditorTemplateSource,
		type GetPermalinkRequestSchema,
		type GetPermalinkResponse,
		type FileSymbolsRequestSchema,
		type FileSymbolsResponse,
		type GetEditorLinkRequestSchema,
		type GetEditorLinkResponse,
		type EditorPreset,
		type Node as GraphNode
	} from '$lib/gen/ui_pb';
	import CopyAction from './CopyAction.svelte';
	import EditorLinkPicker from './EditorLinkPicker.svelte';

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

	// FileSymbolsClient (09-03, BRW-10): the breadcrumb's ONLY data
	// source. Optional on the client prop — same "absent member behaves
	// as idle" discipline as PermalinkClient — so existing stub clients
	// in web/tests/source-pane.test.ts that predate this plan remain
	// valid with no changes.
	interface FileSymbolsClient {
		fileSymbols(
			request: MessageInitShape<typeof FileSymbolsRequestSchema>,
			options?: { signal?: AbortSignal }
		): Promise<FileSymbolsResponse>;
	}

	// EditorLinkClient (09-04, BRW-11/BRW-12): the header link's and the
	// gutter's ONLY data source. Optional on the client prop — same
	// "absent member behaves as idle" discipline as PermalinkClient and
	// FileSymbolsClient — so existing stub clients in tests that predate
	// this plan remain valid with no changes.
	interface EditorLinkClient {
		getEditorLink(
			request: MessageInitShape<typeof GetEditorLinkRequestSchema>,
			options?: { signal?: AbortSignal }
		): Promise<GetEditorLinkResponse>;
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
		indexStale = false,
		onNavigate
	}: {
		state: BrowseTargetState;
		client?: PermalinkClient & Partial<FileSymbolsClient> & Partial<EditorLinkClient>;
		// indexStale (03-09, D-03): the SAME `GetStatusResponse.stale` flag
		// the layout's status banner already reads, passed down so this
		// pane's own source-absent message can split by cause. Optional
		// so a caller that renders this pane with no index-status context
		// (e.g. 03-04's pre-existing tracer fixtures) is unaffected — the
		// default (false) preserves the plain no-source message.
		indexStale?: boolean;
		onNavigate?: (delta: BrowseNavDelta, intent: NavIntent) => void;
	} = $props();

	function handleCallTargetSelect(entry: readonly GraphNode[]): void {
		if (entry.length === 0) return;
		onNavigate?.({ symbol: entry[0].name }, NAV_INTENT.NAVIGATE);
	}

	// languageForPath (moved to $lib/highlight — WR-02): a file-mode
	// GetNodeDetailResponse carries no Node.language field (only
	// NODE_DETAIL_MODE_SINGLE_DEF/MULTI_DEF do, via Node.language), so for
	// a bare file open the language is derived from the path extension via
	// highlight.ts's EXTENSION_LANGUAGE map, which is bound to the
	// indexer's own registry by web/highlight_extension_coverage_test.go.

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

	// --- BRW-10: the sticky "which symbol am I inside" breadcrumb -----

	type FileSymbolsState =
		| { kind: 'idle' }
		| { kind: 'loading' }
		| { kind: 'loaded'; path: string; response: FileSymbolsResponse };

	let fileSymbolsState = $state<FileSymbolsState>({ kind: 'idle' });

	// fileSymbolsCache: the WHOLE of "have we already fetched this file's
	// symbols" (the graph page's fileSymbolsCache/pendingFileSymbols
	// pattern, web/src/routes/graph/+page.svelte) — reassigned to a NEW
	// Map on every insert, never mutated in place (Svelte 5 reactivity).
	// Navigating back to a previously-opened path is served from here
	// with zero further requests.
	let fileSymbolsCache = $state<Map<string, FileSymbolsResponse>>(new Map());

	// filePath is a $derived (not a plain read inside the effect below):
	// Svelte 5 only re-notifies dependents of a $derived when its
	// COMPUTED VALUE actually changes, not merely when the `target` prop
	// object's IDENTITY changes. That is exactly the once-per-path
	// contract this effect needs — a re-render passing a brand-new
	// BrowseTargetState object for the SAME path must not re-dispatch.
	let filePath = $derived(target.kind === 'file' ? target.path : undefined);
	// stableClient: $derived's own value-equality dedup (Object.is) means
	// this only propagates when the CLIENT REFERENCE genuinely changes —
	// reading `client` raw inside the effect below would instead follow
	// $props()'s own dirty-marking, which fires on every parent re-render
	// regardless of whether the referenced object actually differs.
	let stableClient = $derived(client);

	$effect(() => {
		const path = filePath;
		const activeClient = stableClient;
		if (path === undefined || !activeClient || !activeClient.fileSymbols) {
			fileSymbolsState = { kind: 'idle' };
			return;
		}

		// untrack: fileSymbolsCache is $state (reassigned to a NEW Map on
		// every insert below), but this effect must key ONLY on `filePath`
		// — reading the cache reactively here would make the effect's own
		// cache-write (below, once the fetch resolves) re-trigger itself,
		// spuriously re-running this effect and aborting the very request
		// that just completed.
		const cached = untrack(() => fileSymbolsCache.get(path));
		if (cached) {
			fileSymbolsState = { kind: 'loaded', path, response: cached };
			return;
		}

		fileSymbolsState = { kind: 'loading' };
		const controller = new AbortController();
		// Called as activeClient.fileSymbols(...), never detached into a
		// bare function reference — a real Connect client's generated
		// method may rely on `this` being the client instance.
		activeClient.fileSymbols({ path }, { signal: controller.signal })
			.then((response) => {
				const cache = new Map(fileSymbolsCache);
				cache.set(path, response);
				fileSymbolsCache = cache;
				// Apply only if the path this response was requested FOR still
				// matches the CURRENT target — the concurrency edge: a slow
				// response for an abandoned path must never overwrite the
				// crumb for whatever the user has since navigated to.
				if (filePath === path) {
					fileSymbolsState = { kind: 'loaded', path, response };
				}
			})
			.catch(() => {
				if (filePath === path) {
					fileSymbolsState = { kind: 'idle' };
				}
			});
		// Runs when `path` (the effect's only real dependency) changes —
		// aborting the PREVIOUS path's still-in-flight request, never the
		// one just dispatched above for the new path.
		return () => controller.abort();
	});

	let currentLine = $state(1);
	let codeEl = $state<HTMLElement | undefined>(undefined);
	let barEl = $state<HTMLElement | undefined>(undefined);

	function computeLines(text: string, language: string): string[] {
		return splitHighlightedLines(highlightSource(text, language));
	}

	// fileRenderedLines is a $derived (not a plain `{@const}`) specifically
	// so the scroll-tracking effect below can read `.length` as one of its
	// dependencies — recomputing `currentLine` whenever the rendered line
	// count changes, per <action>'s "recompute ... whenever lines.length
	// changes".
	let fileRenderedLines = $derived(
		target.kind === 'file'
			? computeLines(new TextDecoder().decode(target.source), languageForPath(target.path))
			: []
	);

	// currentLine tracking: zero rpcs on this path. `firstFullyVisibleLine`
	// is pure O(1) geometry (T-09-17) — one rAF-throttled recompute per
	// scroll/resize burst, never a per-row rect read. jsdom has no layout
	// engine (web/tests/setup.ts) — every rect is 0 there, so
	// firstFullyVisibleLine's own lineHeight<=0 guard keeps currentLine at
	// 1 in every component-level test; the live gate (Task 3) covers real
	// scrolling.
	$effect(() => {
		if (target.kind !== 'file' || !codeEl || !barEl) {
			currentLine = 1;
			return;
		}
		const code = codeEl;
		const bar = barEl;
		let rafScheduled = false;

		function recompute() {
			const firstRow = code.querySelector('[data-line]');
			const lineHeight = firstRow ? firstRow.getBoundingClientRect().height : 0;
			const barBottom = bar.getBoundingClientRect().bottom;
			const codeTop = code.getBoundingClientRect().top;
			currentLine = firstFullyVisibleLine(codeTop, lineHeight, barBottom, fileRenderedLines.length);
		}

		function onScrollOrResize() {
			if (rafScheduled) return;
			rafScheduled = true;
			requestAnimationFrame(() => {
				rafScheduled = false;
				recompute();
			});
		}

		window.addEventListener('scroll', onScrollOrResize, { passive: true });
		window.addEventListener('resize', onScrollOrResize);
		recompute();

		return () => {
			window.removeEventListener('scroll', onScrollOrResize);
			window.removeEventListener('resize', onScrollOrResize);
		};
	});

	let crumbSymbols = $derived(
		fileSymbolsState.kind === 'loaded' ? fileSymbolsState.response.symbols : []
	);
	let crumb = $derived(innermostSymbolAt(crumbSymbols, currentLine));
	let crumbTruncated = $derived(
		fileSymbolsState.kind === 'loaded' ? fileSymbolsState.response.truncated : false
	);

	function scrollToSymbolStart(startLine: number): void {
		document.getElementById(`L${startLine}`)?.scrollIntoView({ block: 'start' });
	}

	function gutterWidthCh(lineCount: number): string {
		return `${String(lineCount).length + 1}ch`;
	}

	// --- BRW-11/BRW-12: the "Open in editor" header link and gutter ----

	// editorLinkParamsFor mirrors permalinkParamsFor's shape: file mode
	// probes line 1 col 1 (D-09 — "open this whole file" reduces to its
	// first line for editor purposes); a single-def probes the node's
	// OWN start position. node.startCol is 0-based tree-sitter data
	// (internal/indexer/goextract/goextract.go:163), so the request adds
	// 1 to convert to the 1-based column editors expect.
	interface EditorLinkParams {
		path: string;
		line: number;
		col: number;
	}

	function editorLinkParamsFor(s: BrowseTargetState): EditorLinkParams | undefined {
		if (s.kind === 'file') {
			return { path: s.path, line: 1, col: 1 };
		}
		if (s.kind === 'single-def') {
			return { path: s.node.filePath, line: s.node.startLine, col: s.node.startCol + 1 };
		}
		return undefined;
	}

	// editorOverride: read ONCE at mount (D-18). The picker's callbacks
	// (Task 3) write through editor-prefs and reassign this $state, which
	// re-triggers the probe effect below with the new template.
	let editorOverride = $state<EditorOverride | null>(readEditorOverride());

	type EditorLinkState =
		| { kind: 'idle' }
		| { kind: 'loading' }
		| { kind: 'loaded'; response: GetEditorLinkResponse };

	let editorLinkState = $state<EditorLinkState>({ kind: 'idle' });

	// lastPresets: the most recent loaded response's presets, so a preset
	// override (id only) resolves to its template string on the NEXT
	// probe or click (D-06) — the browser never holds a second copy of a
	// template it constructs itself.
	let lastPresets = $state<EditorPreset[]>([]);

	// pickerOpenedOnce: D-14's "the SPA presents the picker on first use"
	// — fires at most once per mount, from the load-time probe only (a
	// gutter click's own answer never re-opens an already-dismissed
	// picker).
	let pickerOpenedOnce = false;
	let pickerOpen = $state(false);

	function needsPickerPrompt(response: GetEditorLinkResponse): boolean {
		if (response.availability === EditorLinkAvailability.TEMPLATE_INVALID) return true;
		return (
			response.availability === EditorLinkAvailability.NO_TEMPLATE &&
			response.defaultSource !== EditorTemplateSource.DISABLED
		);
	}

	// Individual primitive $deriveds (not one object $derived) — same
	// once-per-value-change discipline as filePath above: a re-render
	// passing a brand-new BrowseTargetState object for the SAME target
	// must not re-dispatch the probe.
	let editorTargetPath = $derived(editorLinkParamsFor(target)?.path);
	let editorTargetLine = $derived(editorLinkParamsFor(target)?.line);
	let editorTargetCol = $derived(editorLinkParamsFor(target)?.col);

	$effect(() => {
		const path = editorTargetPath;
		const line = editorTargetLine;
		const col = editorTargetCol;
		const override = editorOverride;
		const activeClient = stableClient;
		if (path === undefined || line === undefined || col === undefined || !activeClient?.getEditorLink) {
			editorLinkState = { kind: 'idle' };
			return;
		}
		editorLinkState = { kind: 'loading' };
		const controller = new AbortController();
		// untrack: lastPresets is $state and this effect's own .then below
		// writes it — reading it reactively here would make the write
		// re-trigger this same effect (the fileSymbolsCache pitfall 09-03
		// already documented and fixed the same way).
		const template = templateForRequest(override, untrack(() => lastPresets));
		activeClient
			.getEditorLink({ path, line, col, template }, { signal: controller.signal })
			.then(async (response) => {
				lastPresets = response.presets;
				// Corrective re-probe (CR-01): a pending PRESET override
				// cannot resolve to a template on the very first probe of a
				// page load, because templateForRequest needed `presets`
				// (untracked, still []) which only this response supplies.
				// Detect that exact case — template undefined, override is
				// a preset — and re-resolve against the presets THIS
				// response just delivered; if it now resolves to a real
				// template, re-issue the probe once so the settled state
				// reflects the user's saved preference instead of the
				// server's unrelated default. A `custom` override never
				// needs this: its template never depended on `presets`.
				if (template === undefined && override?.kind === 'preset') {
					const resolved = templateForRequest(override, response.presets);
					if (resolved !== undefined) {
						const corrected = await activeClient.getEditorLink(
							{ path, line, col, template: resolved },
							{ signal: controller.signal }
						);
						lastPresets = corrected.presets;
						editorLinkState = { kind: 'loaded', response: corrected };
						if (needsPickerPrompt(corrected) && !pickerOpenedOnce) {
							pickerOpenedOnce = true;
							pickerOpen = true;
						}
						return;
					}
				}
				editorLinkState = { kind: 'loaded', response };
				if (needsPickerPrompt(response) && !pickerOpenedOnce) {
					pickerOpenedOnce = true;
					pickerOpen = true;
				}
			})
			.catch(() => {
				// A failed probe is not the whole pane's failure — it only
				// means no editor-link surface renders for this target.
				editorLinkState = { kind: 'idle' };
			});
		return () => controller.abort();
	});

	// gutterClickInFlight: a SINGLE flag shared across every gutter cell
	// (T-09-20) — a click burst while a request is pending issues exactly
	// one rpc, never one per cell.
	let gutterClickInFlight = $state(false);

	let editorLinkBuildable = $derived(
		editorLinkState.kind === 'loaded' &&
			editorLinkState.response.availability === EditorLinkAvailability.BUILDABLE
	);

	async function handleGutterClick(lineNumber: number): Promise<void> {
		if (gutterClickInFlight) return;
		const activeClient = client;
		const path = editorLinkParamsFor(target)?.path;
		if (!activeClient?.getEditorLink || !path) return;
		gutterClickInFlight = true;
		try {
			const response = await activeClient.getEditorLink({
				path,
				line: lineNumber,
				col: 1,
				template: templateForRequest(editorOverride, lastPresets)
			});
			lastPresets = response.presets;
			if (response.availability === EditorLinkAvailability.BUILDABLE) {
				// A same-tab navigation to a custom external scheme — never
				// window.open, which leaves a blank tab behind for a
				// non-http(s) handoff (09-RESEARCH.md Open Question 2).
				window.location.assign(response.url);
			} else {
				// Surface the reason in the header exactly as the probe would.
				editorLinkState = { kind: 'loaded', response };
			}
		} catch {
			// Leave state as-is — a failed click is not the whole pane's
			// failure.
		} finally {
			gutterClickInFlight = false;
		}
	}

	// Picker callbacks (Task 3, D-18): write through editor-prefs.ts —
	// the picker itself holds no storage access — then reassign
	// editorOverride, which re-triggers the probe $effect above with the
	// new template. A preset/default choice closes the panel; a custom
	// apply keeps it open so a TEMPLATE_INVALID answer stays visible
	// (the plan's own instruction).
	function handleChoosePreset(id: string): void {
		const next: EditorOverride = { kind: 'preset', id };
		writeEditorOverride(next);
		editorOverride = next;
		pickerOpen = false;
	}

	function handleApplyCustom(template: string): void {
		const next: EditorOverride = { kind: 'custom', template };
		writeEditorOverride(next);
		editorOverride = next;
	}

	function handleUseServerDefault(): void {
		clearEditorOverride();
		editorOverride = null;
		pickerOpen = false;
	}

	function handlePickerKeydown(event: KeyboardEvent): void {
		if (event.key === 'Escape') {
			pickerOpen = false;
		}
	}

	// Focus management (WR-02 review, a11y): the picker can self-open
	// with no user gesture (a load-time probe answer, not a click), so
	// nothing else moves focus for it. panelWrapperEl wraps whichever
	// EditorLinkPicker instance is rendered; toggleButtonEl is bound to
	// whichever of the two toggle `<button>`s (BUILDABLE vs. reason)
	// renders — only one exists at a time. wasPickerOpen is a plain
	// (non-reactive) local, not $state: it exists only to let this
	// effect diff pickerOpen's PREVIOUS value against its current one,
	// the same edge-detection shape the fileSymbolsCache pitfall
	// (09-03) already established for effects that must react to a
	// transition, not merely a value.
	let panelWrapperEl: HTMLDivElement | undefined = $state();
	let toggleButtonEl: HTMLButtonElement | undefined = $state();
	let wasPickerOpen = false;

	$effect(() => {
		const isOpen = pickerOpen;
		if (isOpen && !wasPickerOpen) {
			// Opened: move focus into the panel's first focusable element
			// (never redesign the picker itself into a focus-trap — this
			// is the minimal "focus lands inside the dialog" contract).
			const target = panelWrapperEl?.querySelector<HTMLElement>(
				'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
			);
			target?.focus();
		} else if (!isOpen && wasPickerOpen) {
			// Closed (Escape or any callback that sets pickerOpen = false):
			// restore focus to the toggle that opened it, exactly as a
			// disclosure/dialog pattern is expected to on dismiss.
			toggleButtonEl?.focus();
		}
		wasPickerOpen = isOpen;
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

{#snippet editorLinkSurface()}
	{#if editorLinkState.kind === 'loaded'}
		{@const r = editorLinkState.response}
		{#if r.availability === EditorLinkAvailability.BUILDABLE}
			<a
				href={r.url}
				rel="noreferrer"
				class="text-xs underline decoration-dotted underline-offset-2"
				data-testid="editor-link"
			>
				Open in editor
			</a>
			<button
				bind:this={toggleButtonEl}
				type="button"
				class="text-xs text-muted-foreground"
				data-testid="editor-link-picker-toggle"
				aria-label="Editor link settings"
				aria-expanded={pickerOpen}
				onclick={() => (pickerOpen = !pickerOpen)}
			>
				⚙
			</button>
		{:else if r.availability === EditorLinkAvailability.NO_TEMPLATE && r.defaultSource === EditorTemplateSource.DISABLED}
			<!-- D-16: hidden, not explained — no editor surface at all. -->
		{:else}
			<span class="text-xs text-muted-foreground" data-testid="editor-link-reason">
				{r.reason}
			</span>
			<button
				bind:this={toggleButtonEl}
				type="button"
				class="text-xs text-muted-foreground"
				data-testid="editor-link-picker-toggle"
				aria-label="Editor link settings"
				aria-expanded={pickerOpen}
				onclick={() => (pickerOpen = !pickerOpen)}
			>
				⚙
			</button>
		{/if}
		{#if pickerOpen}
			<div bind:this={panelWrapperEl} onkeydown={handlePickerKeydown} role="presentation">
				<EditorLinkPicker
					response={r}
					override={editorOverride}
					onChoosePreset={handleChoosePreset}
					onApplyCustom={handleApplyCustom}
					onUseServerDefault={handleUseServerDefault}
				/>
			</div>
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
	{@const lines = fileRenderedLines}
	<div class="mt-4" data-testid="browse-source">
		<div class="mb-2 flex flex-wrap items-center gap-3">
			<CopyAction value={target.path} label="file path" />
			{@render permalinkSurface()}
			{@render editorLinkSurface()}
		</div>
		{#if target.truncated}
			<p class="mb-2 text-xs text-muted-foreground" data-testid="browse-truncated">
				Showing first {target.returnedLines} of {target.totalLines} lines.
			</p>
		{/if}
		<!-- D-01/D-02/D-03: a single-line sticky bar pinned above the <pre>,
		     naming the innermost FileSymbols range containing the first
		     fully visible line — or, when that line is in no symbol, the
		     honest empty state (file path only, dimmed). Never the graph
		     view (D-04: file view only). -->
		<div
			bind:this={barEl}
			data-testid="source-breadcrumb"
			data-current-line={currentLine}
			data-symbols-truncated={crumbTruncated}
			class="sticky top-0 z-10 mb-2 flex items-center gap-2 truncate border-b bg-background px-2 py-1 text-xs"
		>
			<span
				data-testid="source-breadcrumb-path"
				class={crumb ? '' : 'text-muted-foreground'}
			>
				{target.path}
			</span>
			<span data-testid="source-breadcrumb-symbol" data-empty={crumb ? 'false' : 'true'}>
				{#if crumb}
					<button
						type="button"
						class="underline decoration-dotted"
						onclick={() => scrollToSymbolStart(crumb.startLine)}
					>
						{crumb.name}
					</button>
				{/if}
			</span>
			{#if fileSymbolsState.kind === 'loaded' && fileSymbolsState.response.truncated}
				<span class="text-muted-foreground">
					symbols capped: {fileSymbolsState.response.symbols.length} of {fileSymbolsState.response
						.totalCount}
				</span>
			{/if}
		</div>
		<pre class="overflow-x-auto rounded border p-4 text-sm"><code
				bind:this={codeEl}
				>{#each lines as html, i}<span class="line" data-line={i + 1} id={`L${i + 1}`}
						>{#if editorLinkBuildable}<button
								type="button"
								class="gutter inline-block select-none text-right text-muted-foreground"
								data-testid={`gutter-line-${i + 1}`}
								data-line={i + 1}
								style={`width: ${gutterWidthCh(lines.length)}; margin-right: 1ch;`}
								onclick={() => handleGutterClick(i + 1)}
								>{i + 1}</button
							>{:else}<span
								class="gutter inline-block select-none text-right text-muted-foreground"
								data-testid={`gutter-line-${i + 1}`}
								aria-hidden="true"
								style={`width: ${gutterWidthCh(lines.length)}; margin-right: 1ch;`}
								>{i + 1}</span
							>{/if}{@html html}</span
					>
{/each}</code
			></pre>
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
			{@render editorLinkSurface()}
		</div>
		{#if source}
			{@const text = new TextDecoder().decode(source.content)}
			{@const lines = computeLines(text, language)}
			{#if source.truncated}
				<p class="mb-2 text-xs text-muted-foreground" data-testid="browse-truncated">
					Showing first {source.returnedLines} of {source.totalLines} lines.
				</p>
			{/if}
			<pre class="overflow-x-auto rounded border p-4 text-sm"><code
					use:callTargets={{ index: callIndex, onSelect: handleCallTargetSelect }}
					>{#each lines as html, i}<span class="line" data-line={i + 1} id={`L${i + 1}`}
							>{#if editorLinkBuildable}<button
									type="button"
									class="gutter inline-block select-none text-right text-muted-foreground"
									data-testid={`gutter-line-${i + 1}`}
									data-line={i + 1}
									style={`width: ${gutterWidthCh(lines.length)}; margin-right: 1ch;`}
									onclick={() => handleGutterClick(i + 1)}
									>{i + 1}</button
								>{:else}<span
									class="gutter inline-block select-none text-right text-muted-foreground"
									data-testid={`gutter-line-${i + 1}`}
									aria-hidden="true"
									style={`width: ${gutterWidthCh(lines.length)}; margin-right: 1ch;`}
									>{i + 1}</span
								>{/if}{@html html}</span
						>
{/each}</code
				></pre>
		{:else if indexStale}
			<!-- D-03: singleDefSourceBlob (internal/uiserver/handlers.go)
			     collapses three read failures into one nil source — an
			     empty file path, a confinement rejection, and a file
			     removed since indexing. The reasoning holds for the first
			     two, but not the third: it is the stale-index case NAV-04
			     names by hand, and the one the user CAN act on by
			     re-indexing. No proto change: GetStatusResponse.stale is
			     already on the wire, and the SAME flag drives the layout's
			     own status banner. -->
			<p class="text-sm text-muted-foreground" data-testid="browse-no-source-stale">
				Source unavailable — the index is stale, and this file may have moved or been removed
				since the last index. Run <code>codegraph index</code> to refresh it, then try again.
			</p>
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
