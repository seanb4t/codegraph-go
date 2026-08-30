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
	// evaluation reads. Published two ways on the FIRST layout settle
	// only (never again) — as
	// `window.__codegraphFileGraphMetrics` (the single documented global
	// property) AND as the `detail` of a `codegraph:filegraph-metrics`
	// CustomEvent dispatched on `window`, so a test can read the same four
	// values without touching the global.
	//
	//   timeToInteractiveMs — elapsed milliseconds from the instant the
	//     ROUTE issued its FileGraph request (the `requestIssuedAt` prop,
	//     a timestamp the route captures BEFORE calling uiClient.fileGraph)
	//     to the FIRST layout's `layoutstop` event. Deliberately WIDE: it
	//     includes the rpc round trip, the protobuf decode, and the
	//     file-graph-transform.ts wire-to-elements transform — this is the
	//     bar locked ahead of this work (see the execution summary's
	//     Maintainer Decision record), not layout time alone. An
	//     EXPANSION never republishes this value — see the publish-once
	//     guard in createFileGraphRenderer below.
	//   layoutDurationMs — elapsed milliseconds from the FIRST layout's own
	//     `run()` call to the same `layoutstop` event. Recorded, never
	//     binding — the narrower window, kept separate so the locked
	//     `timeToInteractiveMs` name can never come to mean this smaller
	//     number.
	//   nodeCount / edgeCount — the counts of nodes and edges that were
	//     actually laid out at the FIRST settle, read from the cytoscape
	//     instance itself.
	export type FileGraphMetrics = {
		timeToInteractiveMs: number;
		layoutDurationMs: number;
		nodeCount: number;
		edgeCount: number;
	};

	// FileGraphNodeGeometry is the rendered-geometry seam: unlike the
	// metrics seam above, this publishes after EVERY layout settle,
	// carrying one entry per currently-rendered node. Canvas rendering
	// leaves no DOM element per graph node, so without this seam no
	// automated real-browser check can click a SPECIFIC node — this is
	// exactly the capability 05-VALIDATION.md's Manual-Only table needs
	// for both this plan's expansion check and a later file-to-symbol
	// expansion check.
	//
	//   x / y — the node's RENDERED position (cytoscape's
	//     `renderedPosition()`, container-relative pixel coordinates
	//     accounting for the current pan/zoom/fit), never the model
	//     position — a real mouse click targets pixels on screen, not
	//     model-space coordinates.
	//   expandable — true only for a collapsed directory node (isDirectory
	//     AND collapsed both true, read from element data cytoscape
	//     already carries — never a rendered class list or parsed id).
	export type FileGraphNodeGeometry = {
		id: string;
		x: number;
		y: number;
		expandable: boolean;
	};

	// The only elk algorithm requested anywhere in this file — a layered
	// (Sugiyama-style) DAG layout, never force-directed. hierarchyHandling
	// INCLUDE_CHILDREN lays out compound parents WITH their contents,
	// rather than as opaque boxes — required for the directory-structural
	// grouping this component renders. Reused, byte-identical, for every
	// layout run — an expansion re-runs the SAME layout, never a
	// different one.
	const LAYOUT_OPTIONS = {
		name: 'elk',
		fit: true,
		elk: {
			algorithm: 'layered',
			'elk.hierarchyHandling': 'INCLUDE_CHILDREN'
		}
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
	} as any;

	// computeGeometry returns undefined, rather than throwing, when the
	// live instance's node collection does not support the full
	// cytoscape collection API (.map over per-node .id()/.data()/
	// .renderedPosition()) — a MINIMAL cytoscape test double covering
	// only this component's core construction/layout/destroy path is a
	// legitimate mock for those properties, and the geometry seam is
	// additive telemetry a real renderer always provides, never something
	// the core paint path may depend on to avoid crashing.
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	function computeGeometry(cy: any): FileGraphNodeGeometry[] | undefined {
		if (typeof cy.nodes !== 'function') return undefined;
		const nodeCollection = cy.nodes();
		if (typeof nodeCollection.map !== 'function') return undefined;
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		return nodeCollection.map((n: any) => {
			const pos = n.renderedPosition();
			return {
				id: n.id(),
				x: pos.x,
				y: pos.y,
				expandable: Boolean(n.data('isDirectory')) && Boolean(n.data('collapsed'))
			};
		});
	}

	// createFileGraphRenderer wraps an already-constructed cytoscape
	// instance (real or headless) with the two operations this component
	// depends on: running the FIRST layout over whatever elements the
	// instance was constructed with, and REPLACING the element set on a
	// live instance for every subsequent view (an expansion or a
	// collapse) — never destroying and reconstructing the instance, and
	// never republishing the metrics seam past the first settle. A free
	// function, not a component method, specifically so a test can
	// construct a real headless cytoscape instance and exercise this
	// exact logic without mounting Svelte or touching jsdom's canvas-less
	// DOM (jsdom has no <canvas> 2D context — confirmed empirically).
	export function createFileGraphRenderer(opts: {
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		cy: any;
		requestIssuedAt: number;
		onMetrics: (metrics: FileGraphMetrics) => void;
		onGeometry: (geometry: FileGraphNodeGeometry[]) => void;
	}) {
		let metricsPublished = false;

		function runLayout(layoutRunStartedAt: number) {
			opts.cy.one('layoutstop', () => {
				const stoppedAt = performance.now();
				if (!metricsPublished) {
					metricsPublished = true;
					opts.onMetrics({
						timeToInteractiveMs: stoppedAt - opts.requestIssuedAt,
						layoutDurationMs: stoppedAt - layoutRunStartedAt,
						nodeCount: opts.cy.nodes().length,
						edgeCount: opts.cy.edges().length
					});
				}
				const geometry = computeGeometry(opts.cy);
				if (geometry !== undefined) {
					opts.onGeometry(geometry);
				}
			});
			opts.cy.layout(LAYOUT_OPTIONS).run();
		}

		return {
			// start runs the FIRST layout. The instance was already
			// constructed WITH its initial element array (never an empty
			// one followed by a synthetic first "replace") — so the very
			// first layout is not preceded by a remove/add cycle.
			start() {
				runLayout(performance.now());
			},
			// replace swaps the live instance's element set in ONE batch
			// and re-runs the SAME layered layout. This is what a
			// directory expansion or collapse calls — the instance itself
			// is never torn down.
			replace(newElements: unknown[]) {
				opts.cy.startBatch();
				opts.cy.elements().remove();
				// eslint-disable-next-line @typescript-eslint/no-explicit-any
				opts.cy.add(newElements as any);
				opts.cy.endBatch();
				runLayout(performance.now());
			}
		};
	}
</script>

<script lang="ts">
	import { untrack } from 'svelte';
	import type { FileGraphElement } from './file-graph-transform';

	let {
		elements,
		style,
		requestIssuedAt,
		onNodeSelected
	}: {
		// The rollup's plain element-data array — never a cytoscape type.
		// GraphCanvas is what turns this application vocabulary into
		// cytoscape's ElementDefinition shape, not the other way around.
		// Reactive: a CHANGED array (an expansion or a collapse) replaces
		// the live instance's elements without reconstructing it — see the
		// second $effect below.
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
		// or event object. The route decides what a directory tap means
		// (expand/collapse); a file tap is handed through unchanged and
		// this plan's route ignores it.
		onNodeSelected?: (id: string, isDirectory: boolean) => void;
	} = $props();

	let container: HTMLDivElement | undefined = $state();
	let renderer: ReturnType<typeof createFileGraphRenderer> | undefined;
	// trackedInitialized guards the SECOND effect below from re-applying
	// the elements the construction effect already handed to cytoscape at
	// construction time — without this guard, the second effect's own
	// first run (which happens in the same mount) would immediately
	// replace the freshly-constructed instance's elements with the exact
	// same array, running a redundant second layout on every mount.
	let trackedInitialized = false;

	function publish(metrics: FileGraphMetrics) {
		if (typeof window === 'undefined') return;
		window.__codegraphFileGraphMetrics = metrics;
		window.dispatchEvent(
			new CustomEvent<FileGraphMetrics>('codegraph:filegraph-metrics', { detail: metrics })
		);
	}

	function publishGeometry(geometry: FileGraphNodeGeometry[]) {
		if (typeof window === 'undefined') return;
		window.__codegraphFileGraphGeometry = geometry;
		window.dispatchEvent(
			new CustomEvent<FileGraphNodeGeometry[]>('codegraph:filegraph-geometry', { detail: geometry })
		);
	}

	// Construct-in-$effect / destroy-on-cleanup, the same shape
	// DataTable.svelte uses for wiring @tanstack/svelte-virtual's
	// createVirtualizer. This effect's ONLY reactive read is `container`
	// itself (via the `if (!container)` guard below) — `elements`,
	// `style` and `requestIssuedAt` are read UNTRACKED here, exactly
	// once, at construction. This is what makes a changed `elements`
	// array (an expansion) unable to ever tear down and rebuild the
	// instance: reconstruction would restart the metrics seam's timers
	// and record a LATER layout's numbers as the first-paint value the
	// gate reads.
	$effect(() => {
		if (!container) return;

		const initialElements = untrack(() => elements);
		const cy = cytoscapeLib({
			container,
			// eslint-disable-next-line @typescript-eslint/no-explicit-any
			elements: initialElements as any,
			// eslint-disable-next-line @typescript-eslint/no-explicit-any
			style: untrack(() => style) as any
		});

		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		cy.on('tap', 'node', (evt: any) => {
			const target = evt.target;
			onNodeSelected?.(target.id(), Boolean(target.data('isDirectory')));
		});

		renderer = createFileGraphRenderer({
			cy,
			requestIssuedAt: untrack(() => requestIssuedAt),
			onMetrics: publish,
			onGeometry: publishGeometry
		});
		trackedInitialized = false;
		renderer.start();

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
			renderer = undefined;
			cy.destroy();
		};
	});

	// A SECOND effect tracks the element array and, when it changes AFTER
	// construction, replaces the live instance's elements in one batch
	// and re-runs the same layered layout — never reconstructing
	// anything. Deliberately a separate effect from the construction one
	// above so this is the ONLY reactive read of `elements` — the
	// construction effect's read is untracked, so this effect firing can
	// never trigger the construction effect to re-run.
	$effect(() => {
		const current = elements;
		if (!trackedInitialized) {
			trackedInitialized = true;
			return;
		}
		renderer?.replace(current);
	});
</script>

<div bind:this={container} class="h-full w-full" data-testid="file-graph-canvas"></div>
