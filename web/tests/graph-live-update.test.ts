// graph-live-update.test.ts — 06-05 Task 1: D-06's two-path live-update
// seam on GraphCanvas.svelte's renderer. Written and observed RED before
// createFileGraphRenderer()'s liveUpdate() existed — see 06-05-SUMMARY.md
// for the verbatim RED output.
//
// D-06 relocates the zero-displacement guarantee into an authoritative
// position write-back performed AFTER layout settles — never into ELK's
// own approximation, which its maintainer states on the record cannot
// precisely fix node positions (eclipse-elk#355). Two independent test
// strategies, mirroring graph-expansion.test.ts's own split:
//
//   1. Real headless cytoscape + real ELK (no DOM, no Svelte mount) for
//      the fast path, the structural path's exact write-back, symbol
//      survivor composition, and the geometry-republish call counts.
//      jsdom has no CSS layout engine, but ELK's own layout algorithm is
//      pure JS with no DOM dependency — the same real-renderer pattern
//      graph-expansion.test.ts's own Part 2 already established.
//
//   2. A minimal, FULLY CONTROLLABLE fake cytoscape core for the THREE
//      layout-serialization (generation-guard) tests. These need
//      precise, deterministic control over exactly WHEN a `layoutstop`
//      callback fires — including invoking one manually, well after a
//      newer run has already settled, to prove the guard rather than
//      hope real async ELK happens to race a particular way. Confirmed
//      this task, by reading the installed cytoscape's own
//      extension.mjs, that a layout's own `layoutstop` BUBBLES to `cy`
//      (`bubble: function(){ return true; }`) — so a `cy.one('layoutstop',
//      ...)` registered by an OLDER call can genuinely be invoked by a
//      NEWER layout's completion event; this fake reproduces that
//      multiple-pending-listener shape directly rather than assuming it.
//
//   3. 06-05 Task 2: a THIRD strategy for the graph ROUTE's own live
//      subscription — mocked cytoscape/cytoscape-elk (a fuller fake core
//      than Part 2's, this time implementing getElementById so
//      liveUpdate()'s real fast-path/structural-path logic actually
//      exercises against it) plus a mocked $lib/client and a fake
//      liveStore delivered via Svelte context, mirroring
//      live-route-refetch.test.ts's own fakeLiveStore() convention.
import { render, screen, waitFor } from '@testing-library/svelte';
import { describe, expect, it, vi, beforeEach } from 'vitest';
import type { FileGraphNode, FileGraphResponse, FileSymbolsResponse } from '$lib/gen/ui_pb';

async function realRendererModule() {
	const { default: realCytoscape } = await vi.importActual<{
		default: typeof import('cytoscape');
	}>('cytoscape');
	const { default: realElk } = await vi.importActual<typeof import('cytoscape-elk')>('cytoscape-elk');
	realCytoscape.use(realElk);
	const mod = await vi.importActual<typeof import('../src/lib/components/graph/GraphCanvas.svelte')>(
		'../src/lib/components/graph/GraphCanvas.svelte'
	);
	return { realCytoscape, createFileGraphRenderer: mod.createFileGraphRenderer };
}

// baseElements: 5 leaf nodes + 1 edge — comfortably clears the "at least
// 3 survivors compared" floor (rule 84d1gfpywd) for every structural-path
// test below, whichever single node is added or removed.
function baseElements() {
	return [
		{ data: { id: 'n1', isDirectory: false, cycleId: 0 } },
		{ data: { id: 'n2', isDirectory: false, cycleId: 0 } },
		{ data: { id: 'n3', isDirectory: false, cycleId: 0 } },
		{ data: { id: 'n4', isDirectory: false, cycleId: 0 } },
		{ data: { id: 'n5', isDirectory: false, cycleId: 0 } },
		{ data: { id: 'e12', source: 'n1', target: 'n2', totalCount: 1 } }
	];
}

describe('graph live update: real renderer (real headless cytoscape + real ELK)', () => {
	it('fast path: an unchanged node id set updates data in place, runs NO layout, republishes geometry once, and leaves every position untouched', async () => {
		const { realCytoscape, createFileGraphRenderer } = await realRendererModule();
		const elementsA = baseElements();
		const cy = realCytoscape({ headless: true, elements: elementsA });
		let geometryCalls = 0;
		const waiters: Array<() => void> = [];
		function nextGeometry() {
			return new Promise<void>((resolve) => waiters.push(resolve));
		}
		const renderer = createFileGraphRenderer({
			cy,
			requestIssuedAt: performance.now(),
			onMetrics: () => {},
			onGeometry: () => {
				geometryCalls++;
				waiters.shift()?.();
			}
		});
		renderer.start();
		await nextGeometry();

		const before: Record<string, { x: number; y: number }> = {};
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		cy.nodes().forEach((n: any) => {
			const pos = n.position();
			before[n.id()] = { x: pos.x, y: pos.y };
		});

		const layoutSpy = vi.spyOn(cy, 'layout');
		const callsBefore = geometryCalls;

		// The incoming set has the IDENTICAL node id set — only n2's
		// cycleId (and its class) changed, the "counts and cycle
		// membership" case D-06's fast path exists for.
		const newElements = baseElements().map((el) =>
			el.data.id === 'n2' ? { data: { ...el.data, cycleId: 7 }, classes: 'graph-cycle graph-cycle-7' } : el
		);

		renderer.liveUpdate(newElements);

		expect(layoutSpy).not.toHaveBeenCalled();
		expect(geometryCalls).toBe(callsBefore + 1);
		expect(cy.getElementById('n2').data('cycleId')).toBe(7);
		expect(cy.getElementById('n2').classes()).toContain('graph-cycle-7');

		let survivorsChecked = 0;
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		cy.nodes().forEach((n: any) => {
			const pos = n.position();
			expect(pos.x).toBe(before[n.id()].x);
			expect(pos.y).toBe(before[n.id()].y);
			survivorsChecked++;
		});
		expect(survivorsChecked).toBeGreaterThanOrEqual(3);

		cy.destroy();
	});

	it('structural path (nodes added): every surviving leaf node ends at EXACTLY its prior position, under exact numeric equality', async () => {
		const { realCytoscape, createFileGraphRenderer } = await realRendererModule();
		const elementsA = baseElements();
		const cy = realCytoscape({ headless: true, elements: elementsA });
		const waiters: Array<() => void> = [];
		function nextGeometry() {
			return new Promise<void>((resolve) => waiters.push(resolve));
		}
		const renderer = createFileGraphRenderer({
			cy,
			requestIssuedAt: performance.now(),
			onMetrics: () => {},
			onGeometry: () => waiters.shift()?.()
		});
		renderer.start();
		await nextGeometry();

		const distinctive: Record<string, { x: number; y: number }> = {
			n1: { x: 111, y: 211 },
			n2: { x: 122, y: 222 },
			n3: { x: 133, y: 233 },
			n4: { x: 144, y: 244 },
			n5: { x: 155, y: 255 }
		};
		for (const [id, pos] of Object.entries(distinctive)) cy.getElementById(id).position(pos);

		renderer.liveUpdate([...elementsA, { data: { id: 'n6', isDirectory: false } }]);
		await nextGeometry();

		let survivorsChecked = 0;
		for (const [id, pos] of Object.entries(distinctive)) {
			const actual = cy.getElementById(id).position();
			expect(actual.x).toBe(pos.x);
			expect(actual.y).toBe(pos.y);
			survivorsChecked++;
		}
		expect(survivorsChecked).toBeGreaterThanOrEqual(3);

		cy.destroy();
	});

	it('structural path (nodes removed): every surviving leaf node ends at EXACTLY its prior position', async () => {
		const { realCytoscape, createFileGraphRenderer } = await realRendererModule();
		const elementsA = baseElements();
		const cy = realCytoscape({ headless: true, elements: elementsA });
		const waiters: Array<() => void> = [];
		function nextGeometry() {
			return new Promise<void>((resolve) => waiters.push(resolve));
		}
		const renderer = createFileGraphRenderer({
			cy,
			requestIssuedAt: performance.now(),
			onMetrics: () => {},
			onGeometry: () => waiters.shift()?.()
		});
		renderer.start();
		await nextGeometry();

		const distinctive: Record<string, { x: number; y: number }> = {
			n1: { x: 311, y: 411 },
			n2: { x: 322, y: 422 },
			n3: { x: 333, y: 433 },
			n4: { x: 344, y: 444 }
		};
		for (const [id, pos] of Object.entries(distinctive)) cy.getElementById(id).position(pos);

		const withoutN5 = elementsA.filter((el) => el.data.id !== 'n5');
		renderer.liveUpdate(withoutN5);
		await nextGeometry();

		let survivorsChecked = 0;
		for (const [id, pos] of Object.entries(distinctive)) {
			const actual = cy.getElementById(id).position();
			expect(actual.x).toBe(pos.x);
			expect(actual.y).toBe(pos.y);
			survivorsChecked++;
		}
		expect(survivorsChecked).toBeGreaterThanOrEqual(3);
		expect(cy.getElementById('n5').length).toBe(0);

		cy.destroy();
	});

	it('structural path (nodes added AND removed): every surviving leaf node ends at EXACTLY its prior position', async () => {
		const { realCytoscape, createFileGraphRenderer } = await realRendererModule();
		const elementsA = baseElements();
		const cy = realCytoscape({ headless: true, elements: elementsA });
		const waiters: Array<() => void> = [];
		function nextGeometry() {
			return new Promise<void>((resolve) => waiters.push(resolve));
		}
		const renderer = createFileGraphRenderer({
			cy,
			requestIssuedAt: performance.now(),
			onMetrics: () => {},
			onGeometry: () => waiters.shift()?.()
		});
		renderer.start();
		await nextGeometry();

		const distinctive: Record<string, { x: number; y: number }> = {
			n1: { x: 511, y: 611 },
			n2: { x: 522, y: 622 },
			n3: { x: 533, y: 633 },
			n4: { x: 544, y: 644 }
		};
		for (const [id, pos] of Object.entries(distinctive)) cy.getElementById(id).position(pos);

		const combined = [
			...elementsA.filter((el) => el.data.id !== 'n5'),
			{ data: { id: 'n6', isDirectory: false } },
			{ data: { id: 'n7', isDirectory: false } }
		];
		renderer.liveUpdate(combined);
		await nextGeometry();

		let survivorsChecked = 0;
		for (const [id, pos] of Object.entries(distinctive)) {
			const actual = cy.getElementById(id).position();
			expect(actual.x).toBe(pos.x);
			expect(actual.y).toBe(pos.y);
			survivorsChecked++;
		}
		expect(survivorsChecked).toBeGreaterThanOrEqual(3);
		expect(cy.getElementById('n5').length).toBe(0);
		expect(cy.getElementById('n6').length).toBe(1);
		expect(cy.getElementById('n7').length).toBe(1);

		cy.destroy();
	});

	it('a newly added node is HANDED TO THE LAYOUT CALL and holds a defined position when layoutstop fires', async () => {
		const { realCytoscape, createFileGraphRenderer } = await realRendererModule();
		const elementsA = baseElements();
		const cy = realCytoscape({ headless: true, elements: elementsA });
		const waiters: Array<() => void> = [];
		function nextGeometry() {
			return new Promise<void>((resolve) => waiters.push(resolve));
		}
		const renderer = createFileGraphRenderer({
			cy,
			requestIssuedAt: performance.now(),
			onMetrics: () => {},
			onGeometry: () => waiters.shift()?.()
		});
		renderer.start();
		await nextGeometry();

		const layoutSpy = vi.spyOn(cy, 'layout');

		renderer.liveUpdate([...elementsA, { data: { id: 'n6', isDirectory: false } }]);
		// Synchronously, right after the swap and BEFORE layoutstop ever
		// fires: the new node is already part of cy, and cy.layout() (no
		// explicit element collection passed) operates over the whole
		// graph — this is what "handed to the layout call" means here.
		expect(cy.getElementById('n6').length).toBe(1);
		expect(layoutSpy).toHaveBeenCalledTimes(1);

		await nextGeometry();
		const pos = cy.getElementById('n6').position();
		expect(typeof pos.x).toBe('number');
		expect(typeof pos.y).toBe('number');
		expect(Number.isNaN(pos.x)).toBe(false);
		expect(Number.isNaN(pos.y)).toBe(false);

		cy.destroy();
	});

	it('geometry republishes on BOTH paths — a non-zero floor on each, counted separately', async () => {
		const { realCytoscape, createFileGraphRenderer } = await realRendererModule();
		const elementsA = baseElements();
		const cy = realCytoscape({ headless: true, elements: elementsA });
		let geometryCalls = 0;
		const waiters: Array<() => void> = [];
		function nextGeometry() {
			return new Promise<void>((resolve) => waiters.push(resolve));
		}
		const renderer = createFileGraphRenderer({
			cy,
			requestIssuedAt: performance.now(),
			onMetrics: () => {},
			onGeometry: () => {
				geometryCalls++;
				waiters.shift()?.();
			}
		});
		renderer.start();
		await nextGeometry();
		const afterStart = geometryCalls;

		// FAST PATH — identical node id set, data-only.
		renderer.liveUpdate(elementsA);
		const fastPathCalls = geometryCalls - afterStart;
		expect(fastPathCalls).toBeGreaterThan(0);

		// STRUCTURAL PATH — one node added.
		renderer.liveUpdate([...elementsA, { data: { id: 'n6', isDirectory: false } }]);
		await nextGeometry();
		const structuralCalls = geometryCalls - afterStart - fastPathCalls;
		expect(structuralCalls).toBeGreaterThan(0);

		cy.destroy();
	});

	it('symbol survivor composition: a symbol survives a live update whose parent file remains present', async () => {
		const { realCytoscape, createFileGraphRenderer } = await realRendererModule();
		const elementsA = [
			{ data: { id: 'dirX', isDirectory: true } },
			{ data: { id: 'dirX/a.go', isDirectory: false, parent: 'dirX' } },
			{ data: { id: 'y/c.go', isDirectory: false } }
		];
		const cy = realCytoscape({ headless: true, elements: elementsA });
		const waiters: Array<() => void> = [];
		function nextGeometry() {
			return new Promise<void>((resolve) => waiters.push(resolve));
		}
		const renderer = createFileGraphRenderer({
			cy,
			requestIssuedAt: performance.now(),
			onMetrics: () => {},
			onGeometry: () => waiters.shift()?.()
		});
		renderer.start();
		await nextGeometry();

		renderer.add([{ data: { id: 'sym1', isSymbol: true, parent: 'dirX/a.go' } }]);
		await nextGeometry();
		expect(cy.getElementById('sym1').length).toBe(1);

		// Structural live update: dirX/a.go is still present; an unrelated
		// new node is what makes the node id set change.
		renderer.liveUpdate([...elementsA, { data: { id: 'z/d.go', isDirectory: false } }]);
		await nextGeometry();
		expect(cy.getElementById('sym1').length).toBe(1);

		cy.destroy();
	});

	it('symbol survivor composition: a symbol is DROPPED when its parent file is gone after a live update', async () => {
		const { realCytoscape, createFileGraphRenderer } = await realRendererModule();
		const elementsA = [
			{ data: { id: 'dirX', isDirectory: true } },
			{ data: { id: 'dirX/a.go', isDirectory: false, parent: 'dirX' } },
			{ data: { id: 'y/c.go', isDirectory: false } }
		];
		const cy = realCytoscape({ headless: true, elements: elementsA });
		const waiters: Array<() => void> = [];
		function nextGeometry() {
			return new Promise<void>((resolve) => waiters.push(resolve));
		}
		const renderer = createFileGraphRenderer({
			cy,
			requestIssuedAt: performance.now(),
			onMetrics: () => {},
			onGeometry: () => waiters.shift()?.()
		});
		renderer.start();
		await nextGeometry();

		renderer.add([{ data: { id: 'sym1', isSymbol: true, parent: 'dirX/a.go' } }]);
		await nextGeometry();
		expect(cy.getElementById('sym1').length).toBe(1);

		// dirX/a.go (and its parent dirX) collapse away — a genuinely
		// structural change that drops the file the symbol was parented to.
		renderer.liveUpdate([{ data: { id: 'y/c.go', isDirectory: false } }, { data: { id: 'z/d.go', isDirectory: false } }]);
		await nextGeometry();
		expect(cy.getElementById('sym1').length).toBe(0);

		cy.destroy();
	});
});

// --- Layout generation guard (serialization) — a minimal, fully
// controllable fake cytoscape core, per this file's own header comment. ---

type FakeEl = { data: Record<string, unknown>; pos: { x: number; y: number }; parentFlag: boolean };

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function wrapFakeEl(el: FakeEl): any {
	return {
		id: () => el.data.id as string,
		data: (arg?: string | Record<string, unknown>) => {
			if (arg === undefined) return el.data;
			if (typeof arg === 'string') return el.data[arg];
			el.data = { ...el.data, ...arg };
			return undefined;
		},
		position: (pos?: { x: number; y: number }) => {
			if (pos === undefined) return el.pos;
			el.pos = pos;
			return undefined;
		},
		isParent: () => el.parentFlag,
		renderedBoundingBox: () => ({ x1: el.pos.x, y1: el.pos.y, w: 10, h: 10 }),
		children: () => ({ length: 0 }),
		remove: () => {
			/* overridden by getElementById's own closure below */
		},
		classes: () => []
	};
}

// makeGuardFakeCy: a duck-typed cy double covering exactly the surface
// createFileGraphRenderer uses (per this file's own module comment on
// "a MINIMAL cytoscape test double... is a legitimate mock"), with two
// deliberate additions no production code needs: `__handlerAt`/
// `__handlerCount` expose every `cy.one('layoutstop', ...)` registration
// so a test can invoke ANY of them directly, in ANY order — reproducing
// cytoscape's own confirmed bubble-to-`cy` semantics (multiple pending
// 'one' listeners can each be invoked independently) with full
// determinism, which real asynchronous ELK cannot give.
function makeGuardFakeCy(initialIds: string[]) {
	let elements: FakeEl[] = initialIds.map((id) => ({ data: { id }, pos: { x: 0, y: 0 }, parentFlag: false }));
	const registeredHandlers: Array<() => void> = [];

	return {
		startBatch() {},
		endBatch() {},
		elements() {
			return {
				remove: () => {
					elements = [];
				},
				// eslint-disable-next-line @typescript-eslint/no-explicit-any
				forEach: (fn: (e: any) => void) => elements.forEach((e) => fn(wrapFakeEl(e)))
			};
		},
		add(newEls: Array<{ data: Record<string, unknown> }>) {
			for (const el of newEls) {
				elements.push({ data: { ...el.data }, pos: { x: 0, y: 0 }, parentFlag: false });
			}
		},
		getElementById(id: string) {
			const el = elements.find((e) => e.data.id === id);
			if (!el) return { length: 0 };
			return {
				length: 1,
				...wrapFakeEl(el),
				remove: () => {
					elements = elements.filter((e) => e !== el);
				}
			};
		},
		nodes() {
			const nodeEls = elements.filter((e) => !('source' in e.data));
			return {
				length: nodeEls.length,
				// eslint-disable-next-line @typescript-eslint/no-explicit-any
				map: (fn: (n: any) => unknown) => nodeEls.map((e) => fn(wrapFakeEl(e))),
				// eslint-disable-next-line @typescript-eslint/no-explicit-any
				forEach: (fn: (n: any) => void) => nodeEls.forEach((e) => fn(wrapFakeEl(e)))
			};
		},
		edges() {
			return { length: elements.filter((e) => 'source' in e.data).length };
		},
		resize() {},
		one(_event: string, handler: () => void) {
			registeredHandlers.push(handler);
		},
		layout(_opts: unknown) {
			return { run: () => {}, stop: () => {} };
		},
		__handlerAt(i: number) {
			return registeredHandlers[i];
		},
		__handlerCount() {
			return registeredHandlers.length;
		}
	};
}

describe('graph live update: layout generation guard (serialization)', () => {
	it('an un-raced structural update restores its own captured survivor map on settle (positive control)', async () => {
		const cy = makeGuardFakeCy(['n1', 'n2']);
		const { createFileGraphRenderer } = await realRendererModule();
		let geometryCalls = 0;
		const renderer = createFileGraphRenderer({
			cy,
			requestIssuedAt: performance.now(),
			onMetrics: () => {},
			onGeometry: () => {
				geometryCalls++;
			}
		});
		cy.getElementById('n1').position({ x: 111, y: 222 });

		renderer.liveUpdate([{ data: { id: 'n1' } }, { data: { id: 'n3' } }]);
		expect(cy.getElementById('n1').position()).toEqual({ x: 0, y: 0 }); // reset by the swap
		expect(cy.__handlerCount()).toBe(1);

		cy.__handlerAt(0)();

		expect(cy.getElementById('n1').position()).toEqual({ x: 111, y: 222 });
		expect(geometryCalls).toBe(1);
	});

	it("two live structural updates back to back: the SECOND run's survivor map is the one restored, the first is a no-op", async () => {
		const cy = makeGuardFakeCy(['n1', 'n2']);
		const { createFileGraphRenderer } = await realRendererModule();
		let geometryCalls = 0;
		const renderer = createFileGraphRenderer({
			cy,
			requestIssuedAt: performance.now(),
			onMetrics: () => {},
			onGeometry: () => {
				geometryCalls++;
			}
		});

		cy.getElementById('n1').position({ x: 10, y: 10 });
		renderer.liveUpdate([{ data: { id: 'n1' } }, { data: { id: 'n3' } }]); // run 1 captures {n1:(10,10)}

		cy.getElementById('n1').position({ x: 20, y: 20 }); // a DISTINGUISHABLE position for run 2's capture
		renderer.liveUpdate([{ data: { id: 'n1' } }, { data: { id: 'n4' } }]); // run 2 captures {n1:(20,20)}

		expect(cy.__handlerCount()).toBe(2);

		cy.__handlerAt(0)(); // the OLDER run's (bubbled) completion fires
		expect(geometryCalls).toBe(0); // stale — no publish
		expect(cy.getElementById('n1').position()).toEqual({ x: 0, y: 0 }); // NOT restored to run 1's map

		cy.__handlerAt(1)(); // the CURRENT run's completion
		expect(geometryCalls).toBe(1);
		expect(cy.getElementById('n1').position()).toEqual({ x: 20, y: 20 }); // run 2's map, not run 1's
	});

	it('an un-raced add() publishes geometry exactly once on settle (positive control)', async () => {
		const cy = makeGuardFakeCy(['n1']);
		const { createFileGraphRenderer } = await realRendererModule();
		let geometryCalls = 0;
		const renderer = createFileGraphRenderer({
			cy,
			requestIssuedAt: performance.now(),
			onMetrics: () => {},
			onGeometry: () => {
				geometryCalls++;
			}
		});

		renderer.add([{ data: { id: 'sym1', parent: 'n1' } }]);
		expect(cy.__handlerCount()).toBe(1);

		cy.__handlerAt(0)();
		expect(geometryCalls).toBe(1);
	});

	it("a live update racing a user-driven add(): only the LATER run's geometry publishes, and it reflects add()'s own state", async () => {
		const cy = makeGuardFakeCy(['n1', 'n2']);
		const { createFileGraphRenderer } = await realRendererModule();
		let geometryCalls = 0;
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		let lastGeometry: any;
		const renderer = createFileGraphRenderer({
			cy,
			requestIssuedAt: performance.now(),
			onMetrics: () => {},
			onGeometry: (g) => {
				geometryCalls++;
				lastGeometry = g;
			}
		});

		// A live structural update starts (n2 removed, n3 added) — its
		// own layoutstop callback is still pending.
		renderer.liveUpdate([{ data: { id: 'n1' } }, { data: { id: 'n3' } }]);
		// Before it settles, a user expands a file's symbols — add() also
		// goes through runLayout and takes the next generation token.
		renderer.add([{ data: { id: 'sym1', parent: 'n1' } }]);

		expect(cy.__handlerCount()).toBe(2);

		cy.__handlerAt(0)(); // the live update's own callback — now stale
		expect(geometryCalls).toBe(0);

		cy.__handlerAt(1)(); // add()'s callback — current
		expect(geometryCalls).toBe(1); // only the LATER run published
		expect(Array.isArray(lastGeometry) && lastGeometry.some((e: { id: string }) => e.id === 'sym1')).toBe(true);
	});

	it('a stale layoutstop invoked manually after a newer run has settled publishes nothing and restores nothing, paired with the current run which does publish', async () => {
		const cy = makeGuardFakeCy(['n1', 'n2']);
		const { createFileGraphRenderer } = await realRendererModule();
		let geometryCalls = 0;
		const renderer = createFileGraphRenderer({
			cy,
			requestIssuedAt: performance.now(),
			onMetrics: () => {},
			onGeometry: () => {
				geometryCalls++;
			}
		});

		cy.getElementById('n1').position({ x: 5, y: 5 });
		renderer.liveUpdate([{ data: { id: 'n1' } }, { data: { id: 'n3' } }]); // run A
		const staleHandler = cy.__handlerAt(0);

		cy.getElementById('n1').position({ x: 50, y: 50 });
		renderer.liveUpdate([{ data: { id: 'n1' } }, { data: { id: 'n4' } }]); // run B

		cy.__handlerAt(1)(); // run B settles NORMALLY — the current run's callback, which DOES publish
		expect(geometryCalls).toBe(1);
		expect(cy.getElementById('n1').position()).toEqual({ x: 50, y: 50 });

		// Invoke run A's handler MANUALLY, well after B has already
		// settled — a genuinely stale callback arriving late.
		staleHandler();
		expect(geometryCalls).toBe(1); // unchanged — no publish
		expect(cy.getElementById('n1').position()).toEqual({ x: 50, y: 50 }); // unchanged — nothing restored
	});
});

// --- 06-05 Task 2: the graph ROUTE's own live subscription ---
//
// A fuller fake cytoscape core than Part 2's guard fake above — this one
// implements getElementById so liveUpdate()'s REAL fast-path/structural-
// path logic actually runs against it (Part 2's fake didn't need this;
// it exercised liveUpdate() directly, never through the route's own
// getElementById-gated entry check). Mirrors graph-expansion.test.ts's
// own FakeCore shape, extended with getElementById/removeCallCount/
// dataMergeCallCount so a test can distinguish "the new seam updated
// data in place" from "the plain replace() path tore down and rebuilt
// every element" — the two paths this task's own behavior requires stay
// genuinely distinct, not merged into one.

type RouteFakeElement = { data: Record<string, unknown>; classes?: string };

class RouteFakeCore {
	private listeners = new Map<string, Array<(evt?: unknown) => void>>();
	private els: RouteFakeElement[];
	removeCallCount = 0;
	dataMergeCallCount = 0;

	// Elements are CLONED at construction and on add() — real cytoscape
	// snapshots element structure into its own internal model at add
	// time, never retaining a live reference into the caller's original
	// object. Without this, a route under test whose `elements`/
	// `liveElements` props are Svelte $state proxies would have this
	// fake mutate THOSE reactive objects directly (via getElementById's
	// data-merge setter below) — creating a genuine read-write cycle on
	// the same reactive graph the SIXTH effect (liveElements) reads,
	// which Svelte's own scheduler detects as an infinite update loop
	// (`effect_update_depth_exceeded`). Observed and fixed this session.
	constructor(opts: { elements?: RouteFakeElement[] }) {
		this.els = RouteFakeCore.cloneElements(opts.elements ?? []);
	}
	private static cloneElements(elements: RouteFakeElement[]): RouteFakeElement[] {
		return elements.map((el) => ({ data: { ...el.data }, classes: el.classes }));
	}
	on(event: string, ...args: unknown[]) {
		const handler = args[args.length - 1] as (evt?: unknown) => void;
		const list = this.listeners.get(event) ?? [];
		list.push(handler);
		this.listeners.set(event, list);
		return this;
	}
	one(event: string, handler: (evt?: unknown) => void) {
		return this.on(event, handler);
	}
	startBatch() {}
	endBatch() {}
	elements() {
		return {
			remove: () => {
				this.removeCallCount++;
				this.els = [];
			},
			// eslint-disable-next-line @typescript-eslint/no-explicit-any
			forEach: (fn: (e: any) => void) => this.els.forEach((e) => fn(this.wrap(e)))
		};
	}
	add(newElements: RouteFakeElement[]) {
		this.els = [...this.els, ...RouteFakeCore.cloneElements(newElements)];
	}
	getElementById(id: string) {
		const el = this.els.find((e) => e.data.id === id);
		if (!el) return { length: 0 };
		return {
			length: 1,
			id: () => el.data.id as string,
			data: (arg?: string | Record<string, unknown>) => {
				if (arg === undefined) return el.data;
				if (typeof arg === 'string') return el.data[arg];
				this.dataMergeCallCount++;
				el.data = { ...el.data, ...arg };
				return undefined;
			},
			position: () => ({ x: 0, y: 0 }),
			isParent: () => false,
			classes: (val?: string) => {
				if (val === undefined) return (el.classes ?? '').split(' ').filter(Boolean);
				el.classes = val;
				return undefined;
			},
			remove: () => {
				this.els = this.els.filter((e) => e !== el);
			}
		};
	}
	layout(_opts: unknown) {
		return {
			run: () => {
				queueMicrotask(() => {
					for (const h of this.listeners.get('layoutstop') ?? []) h();
				});
			},
			stop: () => {}
		};
	}
	nodes() {
		const nodeEls = this.els.filter((e) => !('source' in e.data));
		return {
			length: nodeEls.length,
			// eslint-disable-next-line @typescript-eslint/no-explicit-any
			map: <T,>(fn: (n: any) => T) => nodeEls.map((el) => fn(this.wrap(el))),
			// eslint-disable-next-line @typescript-eslint/no-explicit-any
			forEach: (fn: (n: any) => void) => nodeEls.forEach((el) => fn(this.wrap(el)))
		};
	}
	edges() {
		return { length: this.els.filter((e) => 'source' in e.data).length };
	}
	resize() {}
	destroy() {}
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	private wrap(el: RouteFakeElement): any {
		return {
			id: () => el.data.id as string,
			data: (key: string) => el.data[key],
			position: () => ({ x: 0, y: 0 }),
			isParent: () => false,
			renderedBoundingBox: () => ({ x1: 0, y1: 0, w: 10, h: 10 }),
			children: () => ({ length: 0 })
		};
	}
	simulateTap(nodeId: string) {
		const el = this.els.find((e) => e.data.id === nodeId);
		if (!el) throw new Error(`simulateTap: no element with id "${nodeId}" in the currently applied set`);
		const handlers = this.listeners.get('tap') ?? [];
		const evt = { target: this.wrap(el) };
		for (const h of handlers) h(evt);
	}
	currentElementIds(): string[] {
		return this.els.map((e) => e.data.id as string).sort();
	}
}

let currentRouteInstance: RouteFakeCore | undefined;

function routeFakeCytoscapeFactory(opts: { elements?: RouteFakeElement[] }) {
	const instance = new RouteFakeCore(opts);
	currentRouteInstance = instance;
	return instance;
}
routeFakeCytoscapeFactory.use = () => {};

vi.mock('cytoscape', () => ({ default: routeFakeCytoscapeFactory }));
vi.mock('cytoscape-elk', () => ({ default: () => {} }));

let currentFileGraphImpl: () => Promise<FileGraphResponse> = () =>
	Promise.reject(new Error('graph-live-update.test.ts: no fileGraph stub configured'));
let currentFileSymbolsImpl: (path: string) => Promise<FileSymbolsResponse> = () =>
	Promise.reject(new Error('graph-live-update.test.ts: no fileSymbols stub configured'));
let fileGraphCallCount = 0;

vi.doMock('$lib/client', () => ({
	uiClient: {
		fileGraph: () => {
			fileGraphCallCount++;
			return currentFileGraphImpl();
		},
		fileSymbols: (req: { path: string }) => currentFileSymbolsImpl(req.path)
	}
}));

const { default: GraphPage } = await import('../src/routes/graph/+page.svelte');

function routeNode(path: string): FileGraphNode {
	return { path, language: 'go', symbolCount: 1n, cycleId: 0 } as unknown as FileGraphNode;
}

function routeResponse(nodes: FileGraphNode[]): FileGraphResponse {
	return {
		nodes,
		edges: [],
		excludedPackageNodeCount: 0n,
		excludedSelfEdgeCount: 0n,
		excludedContainsEdgeCount: 0n,
		cycleCount: 0
	} as unknown as FileGraphResponse;
}

type FakeLiveEvent = { event: { generation: bigint }; epoch: number };

function fakeLiveStore() {
	const listeners = new Set<(live: FakeLiveEvent | null) => void>();
	let current: FakeLiveEvent | null = null;
	return {
		subscribe(run: (live: FakeLiveEvent | null) => void) {
			listeners.add(run);
			run(current);
			return () => {
				listeners.delete(run);
			};
		},
		deliver(live: FakeLiveEvent) {
			current = live;
			for (const listener of listeners) listener(current);
		}
	};
}

function watchGraphEventLike(generation: bigint): FakeLiveEvent {
	return { event: { generation }, epoch: 1 };
}

beforeEach(() => {
	fileGraphCallCount = 0;
	currentRouteInstance = undefined;
});

describe('graph live update: the route subscription (LIV-02)', () => {
	it('a live event issues exactly ONE additional file-graph call; a replayed lower generation issues none', async () => {
		currentFileGraphImpl = () => Promise.resolve(routeResponse([routeNode('x/a.go'), routeNode('y/b.go')]));
		const live = fakeLiveStore();
		render(GraphPage, { context: new Map<string, unknown>([['liveStore', live]]) });
		await waitFor(() => expect(screen.getByTestId('file-graph-canvas')).toBeInTheDocument());
		expect(fileGraphCallCount).toBe(1); // the mount-time fetch only

		live.deliver(watchGraphEventLike(5n));
		await waitFor(() => expect(fileGraphCallCount).toBe(2));

		live.deliver(watchGraphEventLike(3n)); // replayed lower generation
		await new Promise((r) => setTimeout(r, 20));
		expect(fileGraphCallCount).toBe(2); // unchanged
	});

	it('the live path applies via the NEW seam (no full remove+add), paired with a user-driven expansion still using the plain replace path', async () => {
		currentFileGraphImpl = () =>
			Promise.resolve(routeResponse([routeNode('x/a.go'), routeNode('x/b.go'), routeNode('y/c.go')]));
		const live = fakeLiveStore();
		render(GraphPage, { context: new Map<string, unknown>([['liveStore', live]]) });
		await waitFor(() => expect(screen.getByTestId('file-graph-canvas')).toBeInTheDocument());
		const removeCallsAfterMount = currentRouteInstance!.removeCallCount;

		// The fresh live response has the IDENTICAL collapsed node id set
		// (x, y) — only data differs — so this takes the FAST PATH: no
		// elements().remove() call, but a data merge DID happen.
		currentFileGraphImpl = () =>
			Promise.resolve(routeResponse([routeNode('x/a.go'), routeNode('x/b.go'), routeNode('y/c.go')]));
		live.deliver(watchGraphEventLike(5n));
		await new Promise((r) => setTimeout(r, 20));
		expect(currentRouteInstance!.removeCallCount).toBe(removeCallsAfterMount); // NOT replace()
		expect(currentRouteInstance!.dataMergeCallCount).toBeGreaterThan(0); // the new seam DID run

		// Positive control: a user-driven expansion still goes through
		// replace(), which DOES call elements().remove().
		currentRouteInstance!.simulateTap('x');
		await waitFor(() => expect(currentRouteInstance!.removeCallCount).toBeGreaterThan(removeCallsAfterMount));
	});

	it('an event arriving during an in-flight re-fetch produces no second concurrent call; the newest generation is applied once it settles', async () => {
		let resolveSecond!: (r: FileGraphResponse) => void;
		let resolveThird!: (r: FileGraphResponse) => void;
		let calls = 0;
		currentFileGraphImpl = () => {
			calls++;
			if (calls === 1) return Promise.resolve(routeResponse([routeNode('x/a.go')]));
			if (calls === 2)
				return new Promise((resolve) => {
					resolveSecond = resolve;
				});
			return new Promise((resolve) => {
				resolveThird = resolve;
			});
		};
		const live = fakeLiveStore();
		render(GraphPage, { context: new Map<string, unknown>([['liveStore', live]]) });
		await waitFor(() => expect(calls).toBe(1));

		live.deliver(watchGraphEventLike(5n));
		await waitFor(() => expect(calls).toBe(2)); // the live-triggered fetch is now in flight

		live.deliver(watchGraphEventLike(6n)); // arrives DURING the in-flight fetch
		await new Promise((r) => setTimeout(r, 20));
		expect(calls).toBe(2); // no second concurrent call

		resolveSecond(routeResponse([routeNode('x/a.go'), routeNode('x/b.go')]));
		await waitFor(() => expect(calls).toBe(3)); // exactly one follow-up, for generation 6

		resolveThird(routeResponse([routeNode('x/a.go'), routeNode('x/b.go'), routeNode('z/c.go')]));
	});
});
