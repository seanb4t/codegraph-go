#!/usr/bin/env node
// graph-collapse-affordance-check.mjs — a real-browser proof that the
// explicit collapse-affordance buttons added alongside file-to-symbol
// expansion reliably re-collapse a compound, at BOTH the directory level
// (the case an earlier check already demonstrated broken by canvas click
// alone) and the file level (the identical hit-testing defect this task
// inherits at file scale).
//
// This does not retry the button click with jitter offsets the way an
// earlier canvas-click check had to — a plain HTML <button> element is a
// genuinely hit-testable DOM target, not a sub-two-pixel margin inside a
// canvas-rendered compound, so a single real click at its own bounding-box
// centre is the whole test. What this file proves is exactly that
// difference: canvas-click EXPAND still needs the geometry seam and a
// little jitter tolerance (reused from the earlier check's own approach);
// button-click COLLAPSE needs neither.
import { chromium } from '@playwright/test';
import * as fs from 'node:fs';
import * as path from 'node:path';
import * as url from 'node:url';

/**
 * @typedef {Object} GeometryEntry
 * @property {string} id
 * @property {number} x
 * @property {number} y
 * @property {boolean} expandable
 * @property {number} [fileCount]
 */

/** @returns {string} */
function repoRoot() {
	const here = path.dirname(url.fileURLToPath(import.meta.url));
	return path.resolve(here, '..', '..');
}

/**
 * @param {import('@playwright/test').Page} pg
 * @param {() => Promise<any>} check
 * @param {number} timeoutMs
 * @param {number} pollMs
 * @returns {Promise<any>}
 */
async function pollUntil(pg, check, timeoutMs, pollMs) {
	const deadlineAt = Date.now() + timeoutMs;
	for (;;) {
		const value = await check();
		if (value) return value;
		if (Date.now() >= deadlineAt) {
			throw new Error(`pollUntil: condition did not become true within ${timeoutMs}ms`);
		}
		await pg.waitForTimeout(pollMs);
	}
}

/**
 * getContainerBox re-queries the graph canvas's own on-screen position —
 * called fresh before EVERY canvas click, not cached once. A DOM button
 * click (Phase 2/5's collapse-affordance interaction) scrolls its target
 * into view before clicking, which can shift the page's scroll offset and
 * therefore the canvas element's screen position — a stale cached box
 * silently aims every subsequent canvas click at the wrong page location.
 * @param {import('@playwright/test').Page} page
 */
async function getContainerBox(page) {
	const box = await page.evaluate(() => {
		const el = document.querySelector('[data-testid="file-graph-canvas"]');
		if (!el) return null;
		const r = el.getBoundingClientRect();
		return { x: r.x, y: r.y };
	});
	if (!box) throw new Error('graph-collapse-affordance-check: no graph canvas container found');
	return box;
}

async function waitForGeometry(page, timeoutMs) {
	await pollUntil(
		page,
		() => page.evaluate(() => Array.isArray(/** @type {any} */ (window).__codegraphFileGraphGeometry)),
		timeoutMs,
		25
	);
}

const CLICK_JITTER_OFFSETS = [
	[0, 0],
	[3, 0],
	[-3, 0],
	[0, 3],
	[0, -3],
	[3, 3],
	[-3, -3],
	[3, -3],
	[-3, 3],
	[6, 0],
	[-6, 0],
	[0, 6],
	[0, -6],
	[6, 6],
	[-6, -6],
	[6, -6],
	[-6, 6],
	[12, 0],
	[-12, 0],
	[0, 12],
	[0, -12]
];

/**
 * clickCanvasUntil — real mouse down/up at the geometry point (plus a
 * small jitter budget), used ONLY for the canvas-rendered EXPAND clicks
 * this file still needs to reach a directory or a file in the first
 * place. Never used for collapse — collapse below goes through the real
 * DOM button instead.
 * @param {import('@playwright/test').Page} page
 * @param {{x:number,y:number}} containerBox
 * @param {{x:number,y:number}} basePoint
 * @param {() => Promise<any>} check
 */
async function clickCanvasUntil(page, containerBox, basePoint, check, opts = {}) {
	for (const [dx, dy] of CLICK_JITTER_OFFSETS) {
		await page.mouse.move(containerBox.x + basePoint.x + dx, containerBox.y + basePoint.y + dy);
		await page.mouse.down();
		await page.mouse.up();
		try {
			return await pollUntil(page, check, 3000, 25);
		} catch {
			if (opts.debug) {
				const g = await page.evaluate(() => /** @type {any} */ (window).__codegraphFileGraphGeometry);
				process.stdout.write(
					`  clickCanvasUntil debug: offset(${dx},${dy}) -> geometry length now ${Array.isArray(g) ? g.length : 'n/a'}\n`
				);
			}
			// try the next offset
		}
	}
	throw new Error(`clickCanvasUntil: condition never became true around (${basePoint.x}, ${basePoint.y})`);
}

async function main() {
	const args = process.argv.slice(2);
	const urlIdx = args.indexOf('--url');
	const outIdx = args.indexOf('--out');
	if (urlIdx === -1 || outIdx === -1) {
		throw new Error('graph-collapse-affordance-check: --url <url> and --out <path> are both required');
	}
	const targetUrl = args[urlIdx + 1];
	const outPath = args[outIdx + 1];

	/** @type {string[]} */
	const pageErrors = [];

	const browser = await chromium.launch();
	const browserVersion = browser.version();
	const context = await browser.newContext({ viewport: { width: 1600, height: 1000 } });
	const page = await context.newPage();
	page.on('pageerror', (err) => pageErrors.push(err.message));

	await page.goto(targetUrl, { waitUntil: 'load', timeout: 30000 });
	await waitForGeometry(page, 15000);

	/** @type {GeometryEntry[]} */
	let geometry = await page.evaluate(() => /** @type {any} */ (window).__codegraphFileGraphGeometry);
	const collapsedCount = geometry.length;
	const expandable = geometry.filter((g) => g.expandable);
	if (expandable.length === 0) throw new Error('graph-collapse-affordance-check: no collapsed directory to expand');

	// Pick a small, isolated directory (few files) so the expanded file
	// list is short and this check runs quickly.
	const collapsedIds = new Set(geometry.map((g) => g.id));
	const dirTarget = [...expandable].sort((a, b) => (a.fileCount ?? 0) - (b.fileCount ?? 0))[0];

	// --- Phase 1: canvas-click EXPAND a directory ---
	// The click may land on a NEIGHBOUR directory rather than dirTarget
	// (the same risk an earlier canvas-click check already documented) —
	// determine which directory ACTUALLY expanded from the geometry diff
	// itself, rather than assuming the intended target was hit.
	geometry = await clickCanvasUntil(page, await getContainerBox(page), dirTarget, async () => {
		const g = await page.evaluate(() => /** @type {any} */ (window).__codegraphFileGraphGeometry);
		return Array.isArray(g) && g.length > collapsedCount ? g : null;
	});
	const expandedDirCount = geometry.length;
	const newChildEntry = geometry.find((g) => !collapsedIds.has(g.id) && g.id.includes('/'));
	if (!newChildEntry) {
		throw new Error('graph-collapse-affordance-check: node count grew but no new child-shaped id appeared');
	}
	const actualDirId = newChildEntry.id.slice(0, newChildEntry.id.lastIndexOf('/'));
	process.stdout.write(
		`graph-collapse-affordance-check: phase 1 OK — expanded "${actualDirId}" (${collapsedCount} -> ${expandedDirCount})\n`
	);

	// --- Phase 2: BUTTON-click COLLAPSE the directory (directory level) ---
	const dirButton = page.getByTestId(`graph-collapse-directory-${actualDirId}`);
	await dirButton.waitFor({ state: 'visible', timeout: 5000 });
	await dirButton.click();
	geometry = await pollUntil(
		page,
		async () => {
			const g = await page.evaluate(() => /** @type {any} */ (window).__codegraphFileGraphGeometry);
			return Array.isArray(g) && g.length === collapsedCount ? g : null;
		},
		3000,
		25
	);
	const dirCollapseRestoredExactly = geometry.length === collapsedCount;
	process.stdout.write(
		`graph-collapse-affordance-check: phase 2 OK — button-collapsed "${actualDirId}" (restoredExactly=${dirCollapseRestoredExactly}, count=${geometry.length})\n`
	);

	// --- Phase 3: re-expand the SAME directory (canvas click, using its CURRENT geometry point) ---
	const actualDirEntry = geometry.find((g) => g.id === actualDirId);
	if (!actualDirEntry) {
		throw new Error(`graph-collapse-affordance-check: "${actualDirId}" not found in the collapsed geometry`);
	}
	process.stdout.write(
		`graph-collapse-affordance-check: phase 3 — clicking "${actualDirId}" at (${actualDirEntry.x.toFixed(1)}, ${actualDirEntry.y.toFixed(1)})\n`
	);
	geometry = await clickCanvasUntil(
		page,
		await getContainerBox(page),
		actualDirEntry,
		async () => {
			const g = await page.evaluate(() => /** @type {any} */ (window).__codegraphFileGraphGeometry);
			const grew = Array.isArray(g) && g.length > collapsedCount;
			const stillHasTarget = grew && g.some((/** @type {GeometryEntry} */ e) => e.id === actualDirId);
			const hasChild = grew && g.some((/** @type {GeometryEntry} */ e) => e.id.startsWith(`${actualDirId}/`));
			return stillHasTarget && hasChild ? g : null;
		},
		{ debug: true }
	);
	const childPrefix = `${actualDirId}/`;
	const dirExpandedCountAfterReExpand = geometry.length;
	const fileCandidates = geometry.filter((g) => g.id.startsWith(childPrefix) && !g.expandable);
	if (fileCandidates.length === 0) {
		throw new Error(`graph-collapse-affordance-check: directory "${actualDirId}" has no file children to expand`);
	}

	// --- Phase 4: canvas-click EXPAND a file into its symbols ---
	// A file's symbol count can legitimately be zero, in which case its
	// geometry entry count would not change on click — try a bounded set
	// of file candidates until one grows, mirroring the directory check's
	// own candidate-retry discipline.
	let fileTarget;
	let expandedFileCount = dirExpandedCountAfterReExpand;
	let lastFileErr;
	for (const candidate of fileCandidates.slice(0, 5)) {
		try {
			const before = geometry.length;
			const after = await clickCanvasUntil(page, await getContainerBox(page), candidate, async () => {
				const g = await page.evaluate(() => /** @type {any} */ (window).__codegraphFileGraphGeometry);
				return Array.isArray(g) && g.length > before ? g : null;
			});
			fileTarget = candidate;
			geometry = after;
			expandedFileCount = after.length;
			break;
		} catch (err) {
			lastFileErr = err instanceof Error ? err.message : String(err);
		}
	}
	if (!fileTarget) {
		throw new Error(
			`graph-collapse-affordance-check: no file candidate in "${actualDirId}" expanded (tried ${fileCandidates
				.slice(0, 5)
				.map((c) => c.id)
				.join(', ')}); last error: ${lastFileErr}`
		);
	}

	// --- Phase 5: BUTTON-click COLLAPSE the file (file level) ---
	process.stdout.write(
		`graph-collapse-affordance-check: phase 4 OK — expanded file "${fileTarget.id}" (${dirExpandedCountAfterReExpand} -> ${expandedFileCount})\n`
	);
	const fileButton = page.getByTestId(`graph-collapse-file-${fileTarget.id}`);
	await fileButton.waitFor({ state: 'visible', timeout: 5000 });
	await fileButton.click();
	try {
		geometry = await pollUntil(
			page,
			async () => {
				const g = await page.evaluate(() => /** @type {any} */ (window).__codegraphFileGraphGeometry);
				return Array.isArray(g) && g.length === dirExpandedCountAfterReExpand ? g : null;
			},
			3000,
			25
		);
	} catch (err) {
		const g = await page.evaluate(() => /** @type {any} */ (window).__codegraphFileGraphGeometry);
		process.stdout.write(
			`graph-collapse-affordance-check: phase 5 DEBUG — expected length ${dirExpandedCountAfterReExpand}, actual ${Array.isArray(g) ? g.length : 'n/a'}, ids=${JSON.stringify((g || []).map((/** @type {GeometryEntry} */ e) => e.id))}\n`
		);
		throw err;
	}
	const fileCollapseRestoredExactly = geometry.length === dirExpandedCountAfterReExpand;

	const record = {
		success: dirCollapseRestoredExactly && fileCollapseRestoredExactly,
		collapsedNodes: collapsedCount,
		expandedDirCount,
		dirTarget: actualDirId,
		dirCollapseRestoredExactly,
		dirExpandedCountAfterReExpand,
		fileTarget: fileTarget.id,
		expandedFileCount,
		fileCollapseRestoredExactly,
		pageErrorCount: pageErrors.length,
		pageErrors,
		browserIdentity: { name: 'chromium', version: browserVersion }
	};

	await fs.promises.mkdir(path.dirname(outPath), { recursive: true });
	fs.writeFileSync(outPath, JSON.stringify(record, null, 2) + '\n');
	process.stdout.write(`graph-collapse-affordance-check: wrote ${outPath} — success=${record.success}\n`);

	await context.close();
	await browser.close();

	if (!record.success) {
		throw new Error(
			`graph-collapse-affordance-check: FAILED — dirCollapseRestoredExactly=${dirCollapseRestoredExactly} fileCollapseRestoredExactly=${fileCollapseRestoredExactly}`
		);
	}
}

const isMain = process.argv[1] && url.pathToFileURL(process.argv[1]).href === import.meta.url;
if (isMain) {
	main().catch((err) => {
		console.error(`graph-collapse-affordance-check: fatal — ${err instanceof Error ? err.message : String(err)}`);
		if (err instanceof Error && err.stack) console.error(err.stack);
		process.exit(1);
	});
}
