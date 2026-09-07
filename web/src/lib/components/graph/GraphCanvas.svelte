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
	//   x / y — a point inside the node's own RENDERED bounding box
	//     (container-relative pixel coordinates accounting for the
	//     current pan/zoom/fit), inset a few pixels from its top-left
	//     corner — deliberately NOT the node's centroid
	//     (`renderedPosition()`). For a compound (expanded directory)
	//     node, the centroid typically falls on top of one of its own
	//     CHILD nodes, since children are laid out to fill the parent's
	//     interior — a real mouse click there hits the child, not the
	//     parent, silently no-opping an attempt to re-collapse it (found
	//     by this plan's own real-browser check). An inset corner point
	//     sits inside the parent's padding border, never inside a child,
	//     and is equally valid for a leaf node (file or collapsed
	//     directory), which has no children to collide with.
	//   expandable — true only for a collapsed directory node (isDirectory
	//     AND collapsed both true, read from element data cytoscape
	//     already carries — never a rendered class list or parsed id).
	//   fileCount — present only when expandable; the collapsed node's own
	//     file count (element data, never computed here). Lets a caller
	//     choose a SMALL directory to expand: this component's `fit:true`
	//     layout re-fits the WHOLE graph on every expansion, so expanding
	//     a large directory can zoom the rendered view out far enough
	//     that fixed-pixel style padding shrinks below one screen pixel —
	//     a real finding from this plan's own real-browser check, not a
	//     hypothetical.
	export type FileGraphNodeGeometry = {
		id: string;
		x: number;
		y: number;
		expandable: boolean;
		fileCount?: number;
	};

	// The only elk algorithm requested anywhere in this file — a layered
	// (Sugiyama-style) DAG layout, never force-directed. hierarchyHandling
	// INCLUDE_CHILDREN lays out compound parents WITH their contents,
	// rather than as opaque boxes — required for the directory-structural
	// grouping this component renders. Reused, byte-identical, for every
	// layout run — an expansion re-runs the SAME layout, never a
	// different one. Never forked into a second configuration for a live
	// update — the interactive strategies below apply to EVERY layout
	// this component ever runs, including the first paint.
	//
	// The three interactive-strategy options below are D-06's approximate
	// half of the live-update stability guarantee: cycle-breaking and
	// layering run in their interactive variant, and crossing
	// minimisation runs semi-interactively, so a structural live update's
	// new nodes are placed with knowledge of where the current
	// arrangement already sits (cytoscape-elk already sends every node's
	// current model position into ELK — src/layout.js). This is only ever
	// an approximation — ELK's own maintainer states on the record
	// (eclipse-elk#355) that the layered algorithm "cannot be used to
	// precisely fix the positions of the nodes," only "somewhat preserve
	// the topology." The EXACT half of the guarantee — a surviving node's
	// displacement is exactly zero — comes from the authoritative
	// position write-back in runLayout below, never from these options
	// alone. Option names verified present in the installed elkjs bundle
	// this task (elkjs@0.9.3, pnpm's un-hoisted store path — a grep
	// against the top-level web/node_modules/elkjs/ returns nothing for
	// every option, hoisted or not, so any verification must target the
	// pnpm store path directly and pair the search with a known-present
	// control term).
	const LAYOUT_OPTIONS = {
		name: 'elk',
		fit: true,
		elk: {
			algorithm: 'layered',
			'elk.hierarchyHandling': 'INCLUDE_CHILDREN',
			'org.eclipse.elk.interactive': true,
			'org.eclipse.elk.layered.cycleBreaking.strategy': 'INTERACTIVE',
			'org.eclipse.elk.layered.layering.strategy': 'INTERACTIVE',
			'org.eclipse.elk.layered.crossingMinimization.semiInteractive': true
		},
		// nodeLayoutOptions is cytoscape-elk's PER-NODE ELK option hook
		// (its own layout.js: `k.layoutOptions = options.nodeLayoutOptions(node)`)
		// — the graph-level `elk.padding` above does NOT reach a compound's
		// own child spacing; a compound's OWN padding is a property of
		// THAT node, not the whole graph. Without this, elk lays a
		// compound's children out against its own border with under 2px
        // of margin at this stack's typical density — confirmed by a
		// live-browser measurement this task (box.y1 vs its topmost
		// child's y1 differed by ~1px), leaving no real-mouse-clickable
		// area on the parent once it has children.
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		nodeLayoutOptions: (node: any) =>
			node.isParent() ? { 'elk.padding': '[top=40,left=20,bottom=20,right=20]' } : {}
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
			// includeLabels: false — cytoscape's bounding box otherwise
			// includes label overflow (the collapsed-directory label wraps
			// onto a second line, per graph-style.ts), which can push the
			// box's own top-left corner outside the node's actual
			// rendered shape. Confirmed by a live-browser screenshot this
			// task: with labels included, the computed inset point landed
			// visibly outside the node's rectangle.
			const box = n.renderedBoundingBox({ includeLabels: false });
			const expandable = Boolean(n.data('isDirectory')) && Boolean(n.data('collapsed'));
			// eslint-disable-next-line @typescript-eslint/no-explicit-any
			const children = typeof n.children === 'function' ? n.children() : null;
			let x: number;
			let y: number;
			if (children && children.length > 0 && typeof children.renderedBoundingBox === 'function') {
				// A compound (EXPANDED directory) node: elk lays children
				// out with under a couple of screen pixels of margin from
				// the compound's own border at this stack's typical
				// density — confirmed live this task; neither cytoscape's
				// own style `padding` (cosmetic only under an external
				// layout extension) nor elk's own `elk.padding` (tried
				// both graph-level and per-node via `nodeLayoutOptions`)
				// changed that measurably. The one region reliably WIDER
				// than a couple of pixels and reliably free of children is
				// the directory's own LABEL band, drawn above its content
				// via `text-valign: 'top'` — graph-style.ts sets
				// `text-events: 'yes'` specifically so that band is
				// hit-testable (cytoscape's default is 'no': a label is
				// not clickable on its own). The click point is the
				// vertical midpoint of that band — between the box's own
				// top edge WITH its label and the box's top edge WITHOUT
				// it (i.e. where the actual node shape / its children
				// begin).
				const boxWithLabel = n.renderedBoundingBox({ includeLabels: true });
				x = box.x1 + box.w / 2;
				y = (boxWithLabel.y1 + box.y1) / 2;
			} else {
				// A leaf node (a file, or a collapsed directory with no
				// rendered children) — no child to collide with, so an
				// inset corner point is safely inside its own shape.
				const inset = Math.min(5, box.w / 4, box.h / 4);
				x = box.x1 + inset;
				y = box.y1 + inset;
			}
			const entry: FileGraphNodeGeometry = { id: n.id(), x, y, expandable };
			if (expandable) {
				entry.fileCount = n.data('fileCount');
			}
			return entry;
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
		// liveAddedElements is the WHOLE of the incremental (non-replace)
		// seam's own state: every element add() has merged in, minus
		// whatever removeByIds() has since removed — tracked here, at the
		// renderer's own level, specifically so replace() (below) can
		// compose with it rather than silently destroying it whenever a
		// directory-level full-replace runs. A plain array, never
		// cytoscape state of its own: this component still treats the
		// live cy instance as the single source of truth for what is
		// actually rendered; this list exists only to know what to
		// RE-APPLY after a full swap.
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		let liveAddedElements: any[] = [];

		// layoutGeneration/activeLayoutRun are this renderer INSTANCE's own
		// overlapping-layout guard (D-06's serialization requirement).
		// runLayout takes the next token itself, on every call — start(),
		// replace(), add() and liveUpdate() below all inherit it simply by
		// going through runLayout, rather than each call site minting its
		// own. A `layoutstop` callback whose captured token is no longer
		// the current one performs NO survivor restoration and publishes
		// NO geometry: without this, a live refresh arriving while a user
		// expansion's layout is still running leaves two pending
		// callbacks, each holding a DIFFERENT captured survivor map, each
		// racing to restore positions and republish geometry — and
		// cytoscape's layout events bubble from the LAYOUT instance up to
		// `cy` (confirmed by reading the installed cytoscape's own
		// extension.mjs this task: `bubble: function(){ return true; }`),
		// so a `cy.one('layoutstop', ...)` registered by an OLDER call can
		// genuinely be invoked by a NEWER layout's completion event —
		// whichever lands last wins, and it may be the older one, which is
		// exactly the movement this guard exists to prevent.
		let layoutGeneration = 0;
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		let activeLayoutRun: any;

		// runLayout always uses the SAME layered/hierarchyHandling elk
		// algorithm configuration — never a different layout. Only `fit`
		// varies between the initial paint and a later replace: fitting
		// the WHOLE viewport to the WHOLE graph on every expansion (not
		// merely on first paint) is what a real-browser interaction check
		// this task found actually broken — each fit re-zooms the ENTIRE
		// graph out further, and at this corpus's edge density, unrelated
		// nodes end up within single-digit screen pixels of each other,
		// making ANY subsequent click (by a real user OR a real-mouse
		// test) land on the wrong element. A stable pan/zoom across
		// expansions is also the more usable behaviour on its own
		// merits: a developer expanding one directory should not have
		// the WHOLE graph jump to a new scale underneath them.
		//
		// survivorPositions, when provided, is D-06's authoritative
		// write-back: a map of node id -> the MODEL position that node
		// held immediately before the element swap that preceded this
		// layout run, captured by liveUpdate()'s structural path below.
		// On settle — before geometry republishes — every NON-PARENT node
		// still present that has a captured entry is set back to it
		// EXACTLY (`===` on x and y, since the value written is the exact
		// object read). Parents are excluded deliberately: a compound's
		// position is DERIVED from its children in cytoscape, so writing
		// it back would fight the very children this same pass just
		// restored. This is the EXACT half of the stability guarantee;
		// LAYOUT_OPTIONS' interactive strategies above are only the
		// approximate half that decides where genuinely NEW nodes land.
		function runLayout(
			layoutRunStartedAt: number,
			fit: boolean,
			resizeAfter = false,
			// eslint-disable-next-line @typescript-eslint/no-explicit-any
			survivorPositions?: Map<string, { x: number; y: number }>
		) {
			layoutGeneration += 1;
			const myGeneration = layoutGeneration;
			// Best-effort cleanup of the superseded run — NOT the
			// correctness mechanism. cytoscape's layout `stop()` is not
			// guaranteed to prevent an already-queued `layoutstop` from
			// firing (per the installed library's own semantics), so the
			// token check inside the callback below is what actually
			// matters and must stand on its own regardless of whether
			// stop() has any effect here.
			if (activeLayoutRun && typeof activeLayoutRun.stop === 'function') {
				activeLayoutRun.stop();
			}
			opts.cy.one('layoutstop', () => {
				// The token check is FIRST and unconditional: a superseded
				// run does no resize, no metrics, no survivor restoration,
				// and no geometry publish — it returns immediately.
				if (myGeneration !== layoutGeneration) return;
				if (survivorPositions) {
					const nodeCollection = opts.cy.nodes();
					if (typeof nodeCollection.forEach === 'function') {
						// eslint-disable-next-line @typescript-eslint/no-explicit-any
						nodeCollection.forEach((n: any) => {
							if (typeof n.isParent === 'function' && n.isParent()) return;
							const pos = survivorPositions.get(n.id());
							if (pos !== undefined) n.position(pos);
						});
					}
				}
				// resizeAfter is set by add() above and by liveUpdate()'s
				// structural path below when it merged symbol survivors
				// back in — an addition can grow a compound's rendered
				// extent, which cytoscape does not observe on its own;
				// replace() and the initial start() only pass it when the
				// SAME symbol-survivor merge happened, since removing-
				// then-re-adding the SAME element count never grows the
				// extent past what the prior layout already accounted for.
				if (resizeAfter && typeof opts.cy.resize === 'function') {
					opts.cy.resize();
				}
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
			activeLayoutRun = opts.cy.layout({ ...LAYOUT_OPTIONS, fit }).run();
		}

		// swapElements is the ONE element-swap implementation replace()
		// and liveUpdate()'s structural path below both call — the
		// symbol-survivor rule (an added element whose own PARENT id is
		// present among the new element set survives; one whose parent
		// just disappeared does not, because cytoscape's add() cannot
		// attach a child to a parent absent from the same batch) lives
		// here exactly once rather than being reimplemented a second time
		// for live updates. Returns whether any symbol survivors were
		// merged back in, which the caller uses to decide `resizeAfter`.
		function swapElements(newElements: unknown[]): boolean {
			opts.cy.startBatch();
			opts.cy.elements().remove();
			// eslint-disable-next-line @typescript-eslint/no-explicit-any
			opts.cy.add(newElements as any);
			// eslint-disable-next-line @typescript-eslint/no-explicit-any
			const newIds = new Set((newElements as any[]).map((el) => el.data.id));
			// eslint-disable-next-line @typescript-eslint/no-explicit-any
			const survivors = liveAddedElements.filter((el: any) => newIds.has(el.data.parent));
			if (survivors.length > 0) {
				opts.cy.add(survivors as any);
			}
			liveAddedElements = survivors;
			opts.cy.endBatch();
			return survivors.length > 0;
		}

		return {
			// start runs the FIRST layout, fitting the whole graph into
			// the viewport — one of only two fits this component ever
			// performs on its own initiative (the other is an explicit
			// focus() call below). The instance was already constructed
			// WITH its initial element array (never an empty one followed
			// by a synthetic first "replace") — so the very first layout
			// is not preceded by a remove/add cycle.
			start() {
				runLayout(performance.now(), true);
			},
			// replace swaps the live instance's element set in ONE batch
			// and re-runs the SAME layered layout WITHOUT re-fitting —
			// the current pan/zoom is preserved across an expansion or a
			// collapse. The instance itself is never torn down.
			//
			// This is the directory-level FULL-REPLACE path; add()/
			// removeByIds() below are the separate, INCREMENTAL path a
			// file's symbol expansion uses instead — but the two seams
			// are not independent: `opts.cy.elements().remove()` clears
			// EVERYTHING, including any symbol children a prior add()
			// merged in, so a bare remove-then-add-newElements would
			// silently destroy an expanded file's symbols on every
			// unrelated directory toggle. After the swap, re-apply
			// whichever of liveAddedElements survive
			// it: an added element whose own PARENT id is present among
			// newElements' own ids. A symbol's parent is the file id
			// that declares it (file-graph-transform.ts's
			// symbolElementsForFile) — that file id is present in
			// newElements only when its directory is still expanded, so
			// a symbol whose file just collapsed away is correctly
			// dropped (cytoscape's add() cannot attach a child to a
			// parent that does not exist in the same batch), while a
			// symbol whose file remains visible survives an unrelated
			// directory's own expand or collapse.
			replace(newElements: unknown[]) {
				const hadSymbolSurvivors = swapElements(newElements);
				runLayout(performance.now(), false, hadSymbolSurvivors);
			},
			// add MERGES a batch of new elements into the live instance
			// WITHOUT touching anything already present — the incremental
			// counterpart to replace() above, for file-to-symbol expansion:
			// a symbol element is layered on top of whatever the
			// directory-level `elements` prop currently holds, never
			// derived by recomputing that whole array. A no-op for an
			// empty batch, matching this seam's data-in contract (the
			// caller decides WHETHER to add; this method only decides HOW).
			// Re-runs the same layered layout WITHOUT re-fitting (`fit:
			// false`, mirroring replace()) so the new children are placed
			// rather than stacked at the origin, and calls resize() once
			// the layout settles: adding children changes the graph's own
			// rendered extent, which cytoscape does not observe on its own
			// (the ResizeObserver above watches the CONTAINER element's
			// pixel size, a different thing entirely).
			add(newElements: unknown[]) {
				if (newElements.length === 0) return;
				opts.cy.startBatch();
				// eslint-disable-next-line @typescript-eslint/no-explicit-any
				opts.cy.add(newElements as any);
				// eslint-disable-next-line @typescript-eslint/no-explicit-any
				liveAddedElements = [...liveAddedElements, ...(newElements as any[])];
				opts.cy.endBatch();
				runLayout(performance.now(), false, true);
			},
			// removeByIds removes EXACTLY the named elements from the live
			// instance — no layout re-run (removing a symbol's children
			// does not require re-placing anything else) and no re-fit.
			// Direct id lookup (getElementById), the same discipline
			// focus() below already follows, never a composed selector
			// string built from caller-supplied ids. An id with no current
			// match is silently skipped rather than throwing — the route's
			// own once-per-file bookkeeping is the source of truth for
			// WHICH ids to remove; this method does not second-guess it.
			//
			// Republishes the geometry seam directly (computeGeometry + the
			// caller's onGeometry callback) even though no layout ran: the
			// seam publishes after every SETTLE elsewhere because that is
			// the only point positions are known to have changed, but a
			// removal changes WHICH ids exist even when no position moves,
			// and every consumer of that seam (a real-browser interaction
			// driver, or any future one) reads it as the live, current
			// element list — not "current as of the last layout." Skipping
			// this republish leaves the seam reporting an id the model no
			// longer has, discovered via a live-browser check driving real
			// mouse input against the collapse-affordance button this task
			// added.
			removeByIds(ids: string[]) {
				if (ids.length === 0) return;
				if (typeof opts.cy.getElementById !== 'function') return;
				opts.cy.startBatch();
				for (const id of ids) {
					const el = opts.cy.getElementById(id);
					if (el && el.length > 0 && typeof el.remove === 'function') {
						el.remove();
					}
				}
				const removedIds = new Set(ids);
				// eslint-disable-next-line @typescript-eslint/no-explicit-any
				liveAddedElements = liveAddedElements.filter((el: any) => !removedIds.has(el.data.id));
				opts.cy.endBatch();
				const geometry = computeGeometry(opts.cy);
				if (geometry !== undefined) {
					opts.onGeometry(geometry);
				}
			},
			// liveUpdate is D-06's two-path live-update seam, applied
			// against an incoming element set produced by a re-index
			// event — never called for a user-driven expand/collapse,
			// which keep using replace()/add()/removeByIds() exactly as
			// they already do (a live update and a user gesture are
			// different events and must not be merged into one code
			// path).
			//
			// FAST PATH: the rendered NODE id set is unchanged (only
			// data — counts, cycle membership, health — moved, or an
			// edge appeared/disappeared between two already-rendered
			// nodes). Every element present in both the current and the
			// incoming set has its data (and classes) updated in place;
			// an edge id only the incoming set names is added; an edge
			// id only the current set names is removed. NO layout runs
			// at all — running one here would move nodes for no reason,
			// which is precisely the failure LIV-04 names — and geometry
			// republishes directly, the same "no layout, but still
			// republish" discipline removeByIds() above already
			// established.
			//
			// STRUCTURAL PATH: nodes were added or removed. Every
			// currently-rendered node's own model position is captured
			// BEFORE the swap (this is what survives the swap — the swap
			// itself, like replace()'s, removes and re-adds every
			// element, which resets position); the swap runs through the
			// SAME swapElements() replace() uses, so the symbol-survivor
			// rule is not duplicated; then the shared layout runs with
			// the interactive strategies LAYOUT_OPTIONS carries, and
			// runLayout's own write-back (above) restores every
			// surviving non-parent node to its captured position on
			// settle, before geometry republishes.
			liveUpdate(newElements: unknown[]) {
				if (typeof opts.cy.getElementById !== 'function') return;
				// eslint-disable-next-line @typescript-eslint/no-explicit-any
				const isNodeElement = (el: any) => !('source' in el.data);
				// eslint-disable-next-line @typescript-eslint/no-explicit-any
				const currentNodeIds = new Set(opts.cy.nodes().map((n: any) => n.id()));
				const newNodeIds = new Set(
					// eslint-disable-next-line @typescript-eslint/no-explicit-any
					(newElements as any[]).filter(isNodeElement).map((el) => el.data.id)
				);
				const nodeSetUnchanged =
					currentNodeIds.size === newNodeIds.size &&
					[...currentNodeIds].every((id) => newNodeIds.has(id));

				if (nodeSetUnchanged) {
					opts.cy.startBatch();
					// eslint-disable-next-line @typescript-eslint/no-explicit-any
					const newById = new Map((newElements as any[]).map((el) => [el.data.id, el]));
					const currentIds = new Set<string>();
					// eslint-disable-next-line @typescript-eslint/no-explicit-any
					opts.cy.elements().forEach((ele: any) => currentIds.add(ele.id()));
					for (const [id, el] of newById) {
						if (currentIds.has(id)) {
							const ele = opts.cy.getElementById(id);
							if (ele && ele.length > 0 && typeof ele.data === 'function') {
								ele.data(el.data);
								if (typeof ele.classes === 'function') {
									ele.classes(el.classes ?? '');
								}
							}
						} else {
							opts.cy.add([el]);
						}
					}
					for (const id of currentIds) {
						if (!newById.has(id)) {
							const ele = opts.cy.getElementById(id);
							if (ele && ele.length > 0 && typeof ele.remove === 'function') {
								ele.remove();
							}
						}
					}
					opts.cy.endBatch();
					const geometry = computeGeometry(opts.cy);
					if (geometry !== undefined) {
						opts.onGeometry(geometry);
					}
					return;
				}

				const survivorPositions = new Map<string, { x: number; y: number }>();
				const nodeCollection = opts.cy.nodes();
				if (typeof nodeCollection.forEach === 'function') {
					// eslint-disable-next-line @typescript-eslint/no-explicit-any
					nodeCollection.forEach((n: any) => {
						const pos = n.position();
						survivorPositions.set(n.id(), { x: pos.x, y: pos.y });
					});
				}
				const hadSymbolSurvivors = swapElements(newElements);
				runLayout(performance.now(), false, hadSymbolSurvivors, survivorPositions);
			},
			// focus fits the viewport to exactly the elements named by
			// `ids`, built by direct id lookup (getElementById) rather
			// than a selector string — a repository-relative file path
			// can contain characters (`/`, `.`) a CSS-like selector would
			// mis-parse, so this never composes a selector string out of
			// caller-supplied ids. A no-op for an empty list, matching
			// this seam's data-in contract: the caller decides WHETHER to
			// focus, this method only decides HOW. No layout re-run and
			// no element mutation — a pure viewport operation, requested
			// with plain id data rather than the route reaching for a
			// cytoscape method of its own.
			//
			// cy.fit() changes the current pan/zoom exactly as much as a
			// layout settle does, yet has no layoutstop event of its own
			// to hang a republish on — the same gap removeByIds() above
			// was already found and fixed to close for element removal.
			// Republishes the geometry seam directly (computeGeometry +
			// the caller's onGeometry callback), synchronously, so a
			// previously-published entry's x/y never goes stale the
			// instant a cycle-focus fit runs.
			focus(ids: string[]) {
				if (ids.length === 0) return;
				if (typeof opts.cy.collection !== 'function' || typeof opts.cy.getElementById !== 'function') {
					return;
				}
				// eslint-disable-next-line @typescript-eslint/no-explicit-any
				let matched: any = opts.cy.collection();
				for (const id of ids) {
					matched = matched.union(opts.cy.getElementById(id));
				}
				if (matched.length > 0) {
					opts.cy.fit(matched);
					const geometry = computeGeometry(opts.cy);
					if (geometry !== undefined) {
						opts.onGeometry(geometry);
					}
				}
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
		focusNodeIds,
		addedElements,
		removedElementIds,
		liveElements,
		onNodeSelected,
		onEdgeSelected,
		onBackgroundTapped
	}: {
		// The rollup's plain element-data array — never a cytoscape type.
		// GraphCanvas is what turns this application vocabulary into
		// cytoscape's ElementDefinition shape, not the other way around.
		// Reactive: a CHANGED array (an expansion or a collapse of a
		// DIRECTORY) replaces the live instance's elements without
		// reconstructing it — see the second $effect below. This is the
		// directory-level FULL-REPLACE path; addedElements/
		// removedElementIds below are the separate, INCREMENTAL path a
		// file's symbol expansion uses instead.
		elements: FileGraphElement[];
		// graph-style.ts's plain style-sheet data.
		style: unknown[];
		// performance.now() timestamp the ROUTE captured at the instant
		// it issued the FileGraph rpc call — see FileGraphMetrics above
		// for why this component does not guess it or reach for a
		// navigation timing API itself.
		requestIssuedAt: number;
		// A list of node ids to bring into view — plain string data, never
		// a cytoscape selector or a bound viewport method handed back to
		// the route. A non-empty array fits the viewport to exactly those
		// elements; an empty array leaves the current viewport alone. The
		// route decides WHICH ids (grouping by the typed cycleId already
		// on element data); this component decides only HOW to bring them
		// into view — see createFileGraphRenderer's focus() above.
		focusNodeIds?: string[];
		// addedElements: a plain element-data array to MERGE into the live
		// instance without touching anything already present — the
		// incremental counterpart to `elements`' full replace, used for
		// file-to-symbol expansion. A NEW array reference triggers a
		// fresh add via createFileGraphRenderer's add() above — see the
		// fourth $effect below. The route is responsible for handing this
		// a genuinely new batch each time; an unchanged reference is a
		// no-op by construction (Svelte's own reactivity, not a guard this
		// component adds).
		addedElements?: FileGraphElement[];
		// removedElementIds: a plain string-id array — the live instance
		// removes exactly these ids via removeByIds() above. A NEW array
		// reference triggers a fresh removal — see the fifth $effect
		// below.
		removedElementIds?: string[];
		// liveElements: a plain element-data array produced by a LIVE
		// re-index event (never a user gesture) — a NEW array reference
		// triggers createFileGraphRenderer's liveUpdate() above via the
		// sixth $effect below, which applies D-06's two-path decision
		// (data-only fast path when the node id set is unchanged; a
		// layout-and-write-back structural path otherwise). Deliberately
		// SEPARATE from `elements`/`addedElements`/`removedElementIds` —
		// a live update and a user-driven expand/collapse are different
		// events reaching this component through different props, never
		// merged into one code path. undefined/an unchanged reference is
		// a no-op, matching every other incremental prop's own contract.
		liveElements?: FileGraphElement[];
		// Fired on a node tap, carrying the tapped element's id and its
		// element kind — 'directory', 'file', or 'symbol' (never a
		// cytoscape Element or event object). The route decides what a
		// tap means for each kind (a directory or a file each
		// expand/collapse in place; a symbol tap is currently a
		// no-op on the route side, since a symbol has nothing further to
		// expand).
		onNodeSelected?: (id: string, kind: 'directory' | 'file' | 'symbol') => void;
		// Fired on an edge tap, carrying the edge's source id, target id
		// and its element data (kindCounts/totalCount/inCycle/
		// aggregatedFrom, the FileGraphEdgeData shape) — never a
		// cytoscape Element or event object, exactly the same
		// vocabulary-at-the-seam rule onNodeSelected already follows.
		onEdgeSelected?: (source: string, target: string, data: Record<string, unknown>) => void;
		// Fired on a tap that lands on neither a node nor an edge (the
		// graph background) — the route's deselect signal, kept separate
		// from onNodeSelected/onEdgeSelected so "nothing is selected" is
		// its own event rather than an inferred absence.
		onBackgroundTapped?: () => void;
	} = $props();

	let container: HTMLDivElement | undefined = $state();
	let renderer: ReturnType<typeof createFileGraphRenderer> | undefined;
	// lastAppliedElements tracks the EXACT `elements` array reference the
	// live cytoscape instance currently reflects — set at construction to
	// whatever `elements` held then, and updated every time the SECOND
	// effect below calls replace(). The second effect compares the
	// CURRENT `elements` reference against this by identity, rather than
	// a bare "have I run once yet" boolean: a boolean skip-first-run flag
	// silently eats whichever `elements` change happens to be the FIRST
	// one the second effect observes, WHETHER OR NOT it actually matches
	// what construction used — a real hazard if this component's own
	// construction effect ever runs more than once for a single logical
	// mount (observed this task: a genuine remount can occur before the
	// second effect gets its own first natural run, and a real
	// user-driven expansion arriving in that window was silently
	// swallowed by the old boolean guard, since it always treated its
	// OWN first invocation as "matches construction" without checking).
	// A reference comparison has no such blind spot: it only skips when
	// the value TRULY is what's already applied.
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	let lastAppliedElements: any;

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
			// Read the two boolean discriminators directly off element
			// data (never a rendered class list, never a parsed id) — the
			// same typed-data-over-class-string discipline this
			// application's cycle grouping already follows. A node is
			// never both a directory AND a symbol, so this is a genuine
			// three-way discriminator, not a priority order between
			// overlapping cases.
			const isDirectory = Boolean(target.data('isDirectory'));
			const isSymbol = Boolean(target.data('isSymbol'));
			const kind: 'directory' | 'file' | 'symbol' = isDirectory ? 'directory' : isSymbol ? 'symbol' : 'file';
			onNodeSelected?.(target.id(), kind);
		});

		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		cy.on('tap', 'edge', (evt: any) => {
			const target = evt.target;
			// A minimal cytoscape test double covering only an earlier
			// task's own node-tap path (a legitimate mock for THAT path)
			// may not implement source()/target() on the shape it hands
			// this handler — real cytoscape's own selector scoping
			// guarantees this handler only ever fires for a genuine edge,
			// so this guard exists for such doubles, not for production.
			if (typeof target.source !== 'function' || typeof target.target !== 'function') return;
			onEdgeSelected?.(target.source().id(), target.target().id(), target.data());
		});

		// A bare `cy.on('tap', handler)` (no selector) fires for every
		// tap, including ones cytoscape.js's own event-target rules
		// already route through the two selector-scoped handlers above —
		// the `evt.target === cy` check below is what narrows this to
		// ONLY the background case (cytoscape.js's own documented pattern
		// for distinguishing a background tap from an element tap).
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		cy.on('tap', (evt: any) => {
			if (evt.target === cy) {
				onBackgroundTapped?.();
			}
		});

		renderer = createFileGraphRenderer({
			cy,
			requestIssuedAt: untrack(() => requestIssuedAt),
			onMetrics: publish,
			onGeometry: publishGeometry
		});
		lastAppliedElements = initialElements;
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

	// A SECOND effect tracks the element array and, when its REFERENCE
	// differs from lastAppliedElements (the array construction actually
	// handed to cytoscape, or whatever replace() last applied), replaces
	// the live instance's elements in one batch and re-runs the same
	// layered layout — never reconstructing anything. Deliberately a
	// separate effect from the construction one above so this is the
	// ONLY reactive read of `elements` — the construction effect's read
	// is untracked, so this effect firing can never trigger the
	// construction effect to re-run.
	$effect(() => {
		const current = elements;
		if (current === lastAppliedElements) return;
		lastAppliedElements = current;
		renderer?.replace(current);
	});

	// A THIRD effect tracks focusNodeIds and, on every change, asks the
	// renderer to fit the viewport to exactly those ids. No
	// lastAppliedElements-style first-run guard is needed here (unlike the
	// elements effect above): the first run's default is an empty array,
	// and focus() itself no-ops on an empty list, so an unguarded first
	// run is already inert. `renderer` is read as a plain closure
	// variable, not $state — this effect only re-runs when
	// `focusNodeIds` itself changes, never when construction completes.
	$effect(() => {
		const ids = focusNodeIds ?? [];
		if (ids.length === 0) return;
		renderer?.focus(ids);
	});

	// A FOURTH effect tracks addedElements and, on every NEW array
	// reference, merges that batch into the live instance via add() above
	// — the incremental counterpart to the second effect's full replace.
	// Same no-guard-needed reasoning as the THIRD effect: the prop
	// defaults to an empty array and add() itself no-ops on an empty
	// batch, so an unguarded first run is already inert.
	$effect(() => {
		const batch = addedElements ?? [];
		if (batch.length === 0) return;
		renderer?.add(batch);
	});

	// A FIFTH effect tracks removedElementIds and, on every NEW array
	// reference, removes exactly those ids via removeByIds() above.
	// Same reasoning as the fourth effect above.
	$effect(() => {
		const ids = removedElementIds ?? [];
		if (ids.length === 0) return;
		renderer?.removeByIds(ids);
	});

	// A SIXTH effect tracks liveElements and, on every NEW array
	// reference, hands the batch to createFileGraphRenderer's liveUpdate()
	// above — D-06's two-path live-update seam, entirely separate from
	// the second effect's user-driven full replace. Same no-guard-needed
	// reasoning as the third/fourth/fifth effects: the prop defaults to
	// undefined and this effect no-ops on an empty/absent batch, so an
	// unguarded first run is already inert.
	$effect(() => {
		const batch = liveElements ?? [];
		if (batch.length === 0) return;
		renderer?.liveUpdate(batch);
	});
</script>

<div bind:this={container} class="h-full w-full" data-testid="file-graph-canvas"></div>
