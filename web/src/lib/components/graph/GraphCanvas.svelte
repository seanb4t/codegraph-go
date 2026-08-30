<script lang="ts" module>
	// GraphCanvas.svelte is the renderer swap seam: the ONLY file in
	// web/src permitted to import cytoscape or a cytoscape extension. A
	// renderer swap replaces this file and nothing else — every prop and
	// event crossing this boundary is expressed in this application's
	// vocabulary (file-graph-transform.ts's element-data shape, plain
	// numbers/booleans/strings) rather than a cytoscape type, so nothing
	// outside this file ever needs to know cytoscape exists.
	//
	// The elk layout extension is registered HERE, at module scope, not
	// inside the mount effect below: cytoscape.use() runs once when this
	// module is first evaluated, so repeated mounts of this component
	// (route re-entry, multiple instances) never re-register it — a
	// second registration is a documented cytoscape no-op, but running
	// registration once at module load makes that irrelevant rather than
	// merely harmless.
	//
	// cytoscape-elk's own src/layout.js imports `elkjs/lib/elk.bundled.js`
	// (confirmed by reading its source this task, and confirmed again by
	// reading its published dist/cytoscape-elk.js, which contains the
	// string "Worker" ZERO times) — the synchronous, main-thread ELK
	// build, never elk-api.js's Web-Worker-based variant. No worker
	// configuration is possible through this extension's public API, and
	// none is passed below. This matters concretely: the page is served
	// under a Content-Security-Policy of `default-src 'self'` with no
	// `worker-src` directive (internal/uiserver/spa.go), so a blob-URL
	// worker would be blocked and a remote one both blocked and a
	// supply-chain hole. Verified locally: no Content-Security-Policy
	// violation appears in the browser console when this component runs
	// (recorded in the execution summary for this change).
	import cytoscapeLib from 'cytoscape';
	import elk from 'cytoscape-elk';

	cytoscapeLib.use(elk);

	// FileGraphMetrics is the measurement seam a downstream latency
	// evaluation reads. Published two ways after every layoutstop: as
	// `window.__codegraphFileGraphMetrics` (the single documented global
	// property) AND as the `detail` of a `codegraph:filegraph-metrics`
	// CustomEvent dispatched on `window`, so a test can read the same four
	// values without touching the global.
	//
	//   timeToInteractiveMs — elapsed milliseconds from the instant the
	//     ROUTE issued its FileGraph request (the `requestIssuedAt` prop,
	//     a timestamp the route captures BEFORE calling uiClient.fileGraph)
	//     to this layout's `layoutstop` event. Deliberately WIDE: it
	//     includes the rpc round trip, the protobuf decode, and the
	//     file-graph-transform.ts wire-to-elements transform — this is the
	//     bar locked ahead of this work (see the execution summary's
	//     Maintainer Decision record), not layout time alone.
	//   layoutDurationMs — elapsed milliseconds from THIS layout's own
	//     `run()` call to the same `layoutstop` event. Recorded, never
	//     binding — the narrower window, kept separate so the locked
	//     `timeToInteractiveMs` name can never come to mean this smaller
	//     number.
	//   nodeCount / edgeCount — the counts of nodes and edges that were
	//     actually laid out, read from the cytoscape instance itself
	//     after layoutstop.
	export type FileGraphMetrics = {
		timeToInteractiveMs: number;
		layoutDurationMs: number;
		nodeCount: number;
		edgeCount: number;
	};
</script>

<script lang="ts">
	import type { FileGraphElement } from './file-graph-transform';

	let {
		elements,
		style,
		requestIssuedAt,
		onNodeSelected
	}: {
		// The transform's plain element-data array — never a cytoscape
		// type. GraphCanvas is what turns this application vocabulary
		// into cytoscape's ElementDefinition shape, not the other way
		// around.
		elements: FileGraphElement[];
		// graph-style.ts's plain style-sheet data.
		style: unknown[];
		// performance.now() timestamp the ROUTE captured at the instant
		// it issued the FileGraph rpc call — see FileGraphMetrics above
		// for why this component does not guess it or reach for a
		// navigation timing API itself.
		requestIssuedAt: number;
		// Fired on a node tap, carrying the tapped element's id and
		// whether it is a directory compound — never a cytoscape Element
		// or event object.
		onNodeSelected?: (id: string, isDirectory: boolean) => void;
	} = $props();

	let container: HTMLDivElement | undefined = $state();

	// Construct-in-$effect / destroy-on-cleanup, the same shape
	// DataTable.svelte uses for wiring @tanstack/svelte-virtual's
	// createVirtualizer — the closest structural analog in this codebase
	// for wrapping an imperative library in Svelte 5 runes. This effect's
	// only reactive read is `container` itself (via the `if (!container)`
	// guard below) — `elements`/`style`/`requestIssuedAt` are read once
	// per mount, not tracked, matching this tracer's single-fetch-per-mount
	// scope: it wires ONE FileGraph call and ONE render, not a
	// live-updating canvas that re-lays-out on every prop change.
	$effect(() => {
		if (!container) return;

		const cy = cytoscapeLib({
			container,
			// eslint-disable-next-line @typescript-eslint/no-explicit-any
			elements: elements as any,
			// eslint-disable-next-line @typescript-eslint/no-explicit-any
			style: style as any
		});

		cy.on('tap', 'node', (evt) => {
			const target = evt.target;
			onNodeSelected?.(target.id(), Boolean(target.data('isDirectory')));
		});

		const layoutRunStartedAt = performance.now();
		const layout = cy.layout({
			name: 'elk',
			fit: true,
			elk: {
				// The only elk algorithm requested anywhere in this file —
				// a layered (Sugiyama-style) DAG layout, never
				// force-directed. hierarchyHandling INCLUDE_CHILDREN lays
				// out compound parents WITH their contents, rather than as
				// opaque boxes — required for the directory-structural
				// grouping this component renders.
				algorithm: 'layered',
				'elk.hierarchyHandling': 'INCLUDE_CHILDREN'
				// eslint-disable-next-line @typescript-eslint/no-explicit-any
			} as any
			// eslint-disable-next-line @typescript-eslint/no-explicit-any
		} as any);

		cy.one('layoutstop', () => {
			const stoppedAt = performance.now();
			const metrics: FileGraphMetrics = {
				timeToInteractiveMs: stoppedAt - requestIssuedAt,
				layoutDurationMs: stoppedAt - layoutRunStartedAt,
				nodeCount: cy.nodes().length,
				edgeCount: cy.edges().length
			};
			if (typeof window !== 'undefined') {
				window.__codegraphFileGraphMetrics = metrics;
				window.dispatchEvent(
					new CustomEvent<FileGraphMetrics>('codegraph:filegraph-metrics', { detail: metrics })
				);
			}
		});

		layout.run();

		// Cytoscape does not observe arbitrary container resize, only the
		// `window` resize event (cytoscape.js core/resize.md) — an
		// explicit ResizeObserver on the container is required so the
		// canvas stays correctly sized when its flex/grid parent resizes
		// without a window-level resize. Guarded: ResizeObserver is not
		// implemented in jsdom (confirmed this task), and this component
		// must not throw when mounted under vitest.
		let resizeObserver: ResizeObserver | undefined;
		if (typeof ResizeObserver !== 'undefined') {
			resizeObserver = new ResizeObserver(() => cy.resize());
			resizeObserver.observe(container);
		}

		return () => {
			resizeObserver?.disconnect();
			cy.destroy();
		};
	});
</script>

<div bind:this={container} class="h-full w-full" data-testid="file-graph-canvas"></div>
