<script lang="ts">
	// D-18: fills the four-tab surface this route mounted (04-01's
	// tracer, 04-04's Impact/Callers/Callees, 04-06's Affected). Reads
	// view state EXCLUSIVELY from page.url.searchParams via workbench-url.ts's
	// parseWorkbenchParams — the full input state lives in the URL (D-12)
	// — and every control write goes through serializeWorkbenchParams +
	// replaceState, never a direct state assignment.
	//
	// 04-01's tracer wired only `callers` inline, with its own
	// idle/loading/loaded/failed state machine duplicated per analysis.
	// 04-04 extracts that state machine into AnalysisPanel.svelte (D-06)
	// and instantiates it three more times — Impact (with its depth
	// control, WRK-01), Callers, Callees (each with a limit control,
	// WRK-03). 04-06 wires the fourth: `affected`, driven by FilePicker's
	// multi-file chip set (WRK-02) instead of a single symbol.
	//
	// D-13: the four analyses are TABS, with `mode` the URL parameter
	// selecting which one is active. Each mode's AnalysisPanel is gated by
	// an `{#if}` INSIDE its Tabs.Content, not by bits-ui's own
	// always-mounted-but-hidden Content wrapper — Tabs.Content renders
	// every value's content in the DOM simultaneously (toggling only the
	// `hidden` attribute for inactive panels), so relying on that alone
	// would mount all three wired AnalysisPanels at once and fan every
	// analysis's request out on every render, exactly the cross-tab
	// leakage Task 3's isolation test forbids. The `{#if}` is what
	// actually keeps only the active tab's panel — and its request
	// effect — alive.
	import { page } from '$app/state';
	import { replaceState } from '$app/navigation';
	import { uiClient } from '$lib/client';
	import * as Tabs from '$lib/components/ui/tabs';
	import {
		parseWorkbenchParams,
		serializeWorkbenchParams,
		isShapeInteger,
		WORKBENCH_MODES,
		type WorkbenchMode,
		type WorkbenchParams
	} from '$lib/workbench-url';
	import AnalysisPanel, {
		type AnalysisResult
	} from '$lib/components/workbench/AnalysisPanel.svelte';
	import { SEARCH_DEBOUNCE_MS } from '$lib/search';
	import { impactColumns } from '$lib/components/workbench/impact-columns';
	import { callersColumns } from '$lib/components/workbench/callers-columns';
	import { calleesColumns } from '$lib/components/workbench/callees-columns';
	import { affectedColumns } from '$lib/components/workbench/affected-columns';
	import FilePicker from '$lib/components/workbench/FilePicker.svelte';

	let params = $derived(parseWorkbenchParams(page.url.searchParams));
	let activeMode = $derived(params.mode ?? 'impact');

	const MODE_LABELS: Record<WorkbenchMode, string> = {
		impact: 'Impact',
		affected: 'Affected',
		callers: 'Callers',
		callees: 'Callees'
	};

	// Every Workbench control write goes through this one function (D-12):
	// merge the changed field(s) into the CURRENT parsed params, serialize
	// the whole result, and replace the URL in place — the address bar
	// stays correct and shareable at every instant, with no history entry
	// per keystroke, and no `goto`/navigation (WRK-01: a control change
	// does not leave the page).
	function writeParams(next: Partial<WorkbenchParams>): void {
		const merged: WorkbenchParams = { ...params, ...next };
		const url = new URL(page.url.href);
		url.search = serializeWorkbenchParams(merged).toString();
		replaceState(url, {});
	}

	function handleModeChange(next: string): void {
		writeParams({ mode: next as WorkbenchMode });
	}

	// Debounced (WR-03) — this is the free-text field only; depth/limit
	// stay immediate since those are committed values, not keystroke
	// streams. Undebounced, every keystroke re-derives requestKey and
	// re-triggers AnalysisPanel's dispatch effect, issuing a fresh
	// Callers/Callees/Impact RPC (the most expensive calls this UI can
	// make) that re-enters withEngine/openEngine and contends for the
	// Pebble store lock — reusing the same SEARCH_DEBOUNCE_MS the
	// codebase already applies to the cheaper search.ts/file-search.ts
	// surfaces rather than inventing a second constant. D-12's "full
	// input state lives in the URL" still holds — this only delays WHEN
	// the URL write happens, not whether it happens.
	let symbolTimer: ReturnType<typeof setTimeout> | undefined;
	function handleSymbolInput(e: Event): void {
		const value = (e.currentTarget as HTMLInputElement).value;
		clearTimeout(symbolTimer);
		symbolTimer = setTimeout(() => writeParams({ symbol: value || undefined }), SEARCH_DEBOUNCE_MS);
	}
	$effect(() => {
		return () => clearTimeout(symbolTimer);
	});

	// FilePicker owns no selection state of its own (its own doc comment)
	// — this is the one write path the chip set travels back through, the
	// same writeParams/replaceState mechanism every other control uses.
	function handleAffectedFilesChange(files: string[]): void {
		writeParams({ files });
	}

	// depthInvalid/limitInvalid (mirror NeighborsPanel.svelte's
	// depthInvalid, IN-08): a non-conforming committed value is simply not
	// written — the input keeps showing what the developer typed, but the
	// URL never claims a value this app's own parser would drop on the
	// very next read (D-07: no client-side range check is applied to
	// either field, only a shape check).
	let depthInvalid = $state(false);
	let limitInvalid = $state(false);

	function handleDepthInput(e: Event): void {
		const raw = (e.currentTarget as HTMLInputElement).value;
		if (raw === '') {
			depthInvalid = false;
			writeParams({ depth: undefined });
			return;
		}
		if (!isShapeInteger(raw)) {
			depthInvalid = true;
			return;
		}
		depthInvalid = false;
		writeParams({ depth: Number(raw) });
	}

	function handleLimitInput(e: Event): void {
		const raw = (e.currentTarget as HTMLInputElement).value;
		if (raw === '') {
			limitInvalid = false;
			writeParams({ limit: undefined });
			return;
		}
		if (!isShapeInteger(raw)) {
			limitInvalid = true;
			return;
		}
		limitInvalid = false;
		writeParams({ limit: Number(raw) });
	}

	type ImpactSummary = { nodeCount: number; edgeCount: number };

	// Each of these builders returns a FRESH closure on every call — that
	// is fine and deliberate: AnalysisPanel never treats `run`'s identity
	// as a reactive dependency (it tracks `requestKey` instead), so a new
	// closure reference on an unrelated re-render never triggers an extra
	// dispatch or an extra GetStatus call.
	function makeImpactRun(
		symbol: string,
		depth: number
	): (signal: AbortSignal) => Promise<AnalysisResult<ImpactSummary>> {
		return (signal) =>
			uiClient.impact({ symbol, depth }, { signal }).then((resp) => ({
				rows: resp.affected,
				summary: { nodeCount: resp.nodeCount, edgeCount: resp.edgeCount }
			}));
	}

	function makeCallersRun(
		symbol: string,
		limit: number
	): (signal: AbortSignal) => Promise<AnalysisResult<unknown>> {
		return (signal) =>
			uiClient.callers({ symbol, limit }, { signal }).then((resp) => ({ rows: resp.callers }));
	}

	function makeCalleesRun(
		symbol: string,
		limit: number
	): (signal: AbortSignal) => Promise<AnalysisResult<unknown>> {
		return (signal) =>
			uiClient.callees({ symbol, limit }, { signal }).then((resp) => ({ rows: resp.callees }));
	}

	type AffectedSummary = { files: string[] };

	// AffectedResponse's own `files` is the echo of the REQUEST's files —
	// carried through AnalysisResult's `summary` (04-04's widened
	// contract) rather than read from `params.files` directly, so a late
	// response from a superseded dispatch can never pair its rows with a
	// DIFFERENT (newer) file selection's echo: AnalysisPanel's own
	// request-identity guard discards the whole result, echo included, in
	// one piece.
	function makeAffectedRun(
		files: string[],
		depth: number
	): (signal: AbortSignal) => Promise<AnalysisResult<AffectedSummary>> {
		return (signal) =>
			uiClient.affected({ files, depth }, { signal }).then((resp) => ({
				rows: resp.affectedTests,
				summary: { files: resp.files }
			}));
	}

	// requestKey composes ONLY the fields that select THIS analysis's
	// query (mirrors the tracer's own callersKey discipline) — mode is
	// included so a freshly (re-)mounted panel always fires its effect at
	// least once, even though {#if}-gated mount/unmount already makes
	// that the common case.
	let impactKey = $derived(
		JSON.stringify([activeMode, params.symbol ?? null, params.depth ?? null])
	);
	let callersKey = $derived(
		JSON.stringify([activeMode, params.symbol ?? null, params.limit ?? null])
	);
	let calleesKey = $derived(
		JSON.stringify([activeMode, params.symbol ?? null, params.limit ?? null])
	);
	let affectedKey = $derived(JSON.stringify([activeMode, params.files, params.depth ?? null]));
</script>

{#snippet symbolInput()}
	<label class="text-sm">
		Symbol
		<input
			type="text"
			value={params.symbol ?? ''}
			oninput={handleSymbolInput}
			data-testid="workbench-symbol-input"
			class="block rounded-md border border-input bg-background px-2 py-1 text-sm"
		/>
	</label>
{/snippet}

{#snippet impactInputs()}
	{@render symbolInput()}
	<label class="text-sm">
		Depth
		<input
			type="number"
			value={params.depth ?? ''}
			oninput={handleDepthInput}
			data-testid="workbench-depth-input"
			aria-invalid={depthInvalid}
			class="block rounded-md border border-input bg-background px-2 py-1 text-sm"
		/>
	</label>
{/snippet}

{#snippet limitInputs()}
	{@render symbolInput()}
	<label class="text-sm">
		Limit
		<input
			type="number"
			value={params.limit ?? ''}
			oninput={handleLimitInput}
			data-testid="workbench-limit-input"
			aria-invalid={limitInvalid}
			class="block rounded-md border border-input bg-background px-2 py-1 text-sm"
		/>
	</label>
{/snippet}

{#snippet impactSummary(s: ImpactSummary)}
	<span data-testid="workbench-impact-node-count">Nodes: {s.nodeCount}</span>
	<span class="ml-4" data-testid="workbench-impact-edge-count">Edges: {s.edgeCount}</span>
{/snippet}

{#snippet affectedDepthInput()}
	<label class="text-sm">
		Depth
		<input
			type="number"
			value={params.depth ?? ''}
			oninput={handleDepthInput}
			data-testid="workbench-affected-depth-input"
			aria-invalid={depthInvalid}
			class="block rounded-md border border-input bg-background px-2 py-1 text-sm"
		/>
	</label>
{/snippet}

{#snippet affectedSummary(s: AffectedSummary)}
	<div data-testid="workbench-affected-files-summary">
		<span class="font-medium">Files:</span>
		{#each s.files as f (f)}
			<span class="ml-2" data-testid={`workbench-affected-file-${f}`}>{f}</span>
		{/each}
	</div>
{/snippet}

<h1 class="text-lg font-semibold">Workbench</h1>
<p class="mt-1 text-sm text-muted-foreground">
	Run Impact, Affected, Callers and Callees over this repository's index.
</p>

<Tabs.Root value={activeMode} onValueChange={handleModeChange} class="mt-4">
	<Tabs.List>
		{#each WORKBENCH_MODES as mode (mode)}
			<Tabs.Trigger value={mode} data-testid={`workbench-tab-${mode}`}>
				{MODE_LABELS[mode]}
			</Tabs.Trigger>
		{/each}
	</Tabs.List>

	<Tabs.Content value="impact">
		{#if activeMode === 'impact'}
			<AnalysisPanel
				requestKey={impactKey}
				run={params.symbol ? makeImpactRun(params.symbol, params.depth ?? 0) : undefined}
				columns={impactColumns}
				summary={impactSummary}
				inputs={impactInputs}
				emptyMessage="No affected nodes found."
			/>
		{/if}
	</Tabs.Content>

	<Tabs.Content value="affected">
		{#if activeMode === 'affected'}
			<div class="mt-4">
				<FilePicker
					client={uiClient}
					files={params.files}
					onChange={handleAffectedFilesChange}
				/>
			</div>
			{#if params.files.length === 0}
				<p class="mt-4 text-sm text-muted-foreground" data-testid="workbench-affected-empty">
					Select at least one file to run Affected.
				</p>
			{:else}
				<AnalysisPanel
					requestKey={affectedKey}
					run={makeAffectedRun(params.files, params.depth ?? 0)}
					columns={affectedColumns}
					summary={affectedSummary}
					inputs={affectedDepthInput}
					emptyMessage="No affected tests found."
				/>
			{/if}
		{/if}
	</Tabs.Content>

	<Tabs.Content value="callers">
		{#if activeMode === 'callers'}
			<AnalysisPanel
				requestKey={callersKey}
				run={params.symbol ? makeCallersRun(params.symbol, params.limit ?? 0) : undefined}
				columns={callersColumns}
				inputs={limitInputs}
				emptyMessage="No callers found."
			/>
		{/if}
	</Tabs.Content>

	<Tabs.Content value="callees">
		{#if activeMode === 'callees'}
			<AnalysisPanel
				requestKey={calleesKey}
				run={params.symbol ? makeCalleesRun(params.symbol, params.limit ?? 0) : undefined}
				columns={calleesColumns}
				inputs={limitInputs}
				emptyMessage="No callees found."
			/>
		{/if}
	</Tabs.Content>
</Tabs.Root>
