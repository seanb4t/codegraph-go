<script lang="ts">
	// NeighborsPanel renders the three neighbour regions BRW-02 asks for
	// alongside the source pane: callers (GetNodeDetailResponse's
	// called_by — who calls this node), callees (its calls — what this
	// node calls), and blast radius (Impact's affected set, loaded
	// separately by browse-state.ts's loadBlastRadius and gated by the
	// SAME NavigationGeneration as the node-detail load — see
	// +page.svelte). This component never navigates itself: every click
	// and every depth-control change goes back through onNavigate, which
	// states its own push/replace intent (NAV_INTENT) rather than this
	// component deciding it.
	//
	// calls/calledBy are typed as Node — GetNodeDetailResponse.calls/
	// called_by are full Node messages, NOT Location (ui.proto:397-462).
	// Blast-radius entries are typed as Location — Impact/Callers/
	// Callees/Affected's shared, lighter projection. The two are named
	// apart in these props rather than unified: plan 03-08's permalink
	// logic depends on the distinction (a Location carries no end_line).
	import type { Node, Location } from '$lib/gen/ui_pb';
	import type { BlastRadiusState } from '$lib/browse-state';
	import { NAV_INTENT, type BrowseNavDelta, type NavIntent } from '$lib/browse-nav';
	import { isShapeInteger } from '$lib/browse-url';

	let {
		calls,
		calledBy,
		blastRadius,
		depth,
		onNavigate
	}: {
		calls: Node[];
		calledBy: Node[];
		blastRadius: BlastRadiusState;
		depth?: number;
		onNavigate: (delta: BrowseNavDelta, intent: NavIntent) => void;
	} = $props();

	function entryKey(entry: Node | Location): string {
		return `${entry.filePath}:${entry.startLine}:${entry.name}`;
	}

	// A caller/callee/blast-radius click is a NAVIGATE intent — the
	// developer wants to come back to wherever they clicked from
	// (D-11). The symbol+file+line triple this sends is the
	// disambiguation shape browse-nav.ts's own clearing rule exempts
	// from being cleared.
	function openEntry(entry: Node | Location): void {
		onNavigate(
			{ symbol: entry.name, file: entry.filePath, line: entry.startLine },
			NAV_INTENT.NAVIGATE
		);
	}

	// depthInvalid (IN-08): a non-conforming committed value was
	// previously discarded with no feedback at all — the control kept
	// showing the rejected text while the URL silently stayed at its
	// prior depth, with nothing telling the user why their edit had no
	// effect. Surfaced as aria-invalid below rather than a UI redesign,
	// which keeps this fix scoped to "give the ignored case visible
	// feedback" without also relitigating the empty-clears-depth
	// behavior WR-05's own fix already added and tested.
	let depthInvalid = $state(false);

	// A depth-control change is a REFINE intent — it rewrites the URL in
	// place so the address bar stays correct and shareable at every
	// instant, without filling history with one entry per adjustment.
	//
	// WR-05: this writer must speak the exact same grammar browse-url.ts's
	// reader (parseShapeInteger) accepts — a base-10 integer literal, no
	// decimal point, no exponent — via the shared isShapeInteger
	// predicate. An `<input type="number">` happily accepts "2.5" or
	// "1e3"; writing either into the URL previously produced a value the
	// app's own parser then silently dropped on the very next read,
	// leaving the address bar and the rendered blast radius disagreeing
	// with no error anywhere. A non-conforming value is simply not
	// written — the input keeps showing what the user typed, but the URL
	// (and thus any link shared from it) never claims a depth this app
	// cannot itself parse back.
	function handleDepthChange(e: Event): void {
		const raw = (e.currentTarget as HTMLInputElement).value;
		if (raw === '') {
			depthInvalid = false;
			onNavigate({ depth: undefined }, NAV_INTENT.REFINE);
			return;
		}
		if (!isShapeInteger(raw)) {
			depthInvalid = true;
			return;
		}
		depthInvalid = false;
		onNavigate({ depth: Number(raw) }, NAV_INTENT.REFINE);
	}
</script>

<div class="mt-4 grid gap-4 sm:grid-cols-3" data-testid="neighbors-panel">
	<section data-testid="neighbors-callers">
		<h2 class="text-sm font-semibold">Callers</h2>
		{#if calledBy.length === 0}
			<p class="text-sm text-muted-foreground" data-testid="neighbors-callers-empty">
				No callers.
			</p>
		{:else}
			<ul>
				{#each calledBy as entry (entryKey(entry))}
					<li>
						<button
							type="button"
							data-testid={`neighbor-caller-${entryKey(entry)}`}
							onclick={() => openEntry(entry)}
						>
							{entry.name}
							<span class="text-muted-foreground">{entry.kind} · {entry.filePath}</span>
						</button>
					</li>
				{/each}
			</ul>
		{/if}
	</section>

	<section data-testid="neighbors-callees">
		<h2 class="text-sm font-semibold">Callees</h2>
		{#if calls.length === 0}
			<p class="text-sm text-muted-foreground" data-testid="neighbors-callees-empty">
				No callees.
			</p>
		{:else}
			<ul>
				{#each calls as entry (entryKey(entry))}
					<li>
						<button
							type="button"
							data-testid={`neighbor-callee-${entryKey(entry)}`}
							onclick={() => openEntry(entry)}
						>
							{entry.name}
							<span class="text-muted-foreground">{entry.kind} · {entry.filePath}</span>
						</button>
					</li>
				{/each}
			</ul>
		{/if}
	</section>

	<section data-testid="neighbors-blast-radius">
		<h2 class="text-sm font-semibold">Blast Radius</h2>
		<label class="block text-xs">
			Depth
			<input
				type="number"
				min="0"
				value={depth ?? ''}
				onchange={handleDepthChange}
				data-testid="neighbors-depth-input"
				aria-invalid={depthInvalid}
			/>
		</label>
		{#if blastRadius.kind === 'loading'}
			<p class="text-sm text-muted-foreground" data-testid="neighbors-blast-loading">
				Loading…
			</p>
		{:else if blastRadius.kind === 'failed'}
			<p class="text-sm text-destructive" data-testid="neighbors-blast-failed">
				Something went wrong: {blastRadius.failure.message}
			</p>
		{:else if blastRadius.kind === 'loaded'}
			{#if blastRadius.affected.length === 0}
				<p class="text-sm text-muted-foreground" data-testid="neighbors-blast-empty">
					No affected nodes at depth {blastRadius.depth}.
				</p>
			{:else}
				<ul>
					{#each blastRadius.affected as entry (entryKey(entry))}
						<li>
							<button
								type="button"
								data-testid={`neighbor-blast-${entryKey(entry)}`}
								onclick={() => openEntry(entry)}
							>
								{entry.name}
								<span class="text-muted-foreground">{entry.kind} · {entry.filePath}</span>
							</button>
						</li>
					{/each}
				</ul>
			{/if}
		{/if}
	</section>
</div>
