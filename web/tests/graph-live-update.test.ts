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
import { describe, expect, it, vi } from 'vitest';

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
