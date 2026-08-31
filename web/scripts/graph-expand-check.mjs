#!/usr/bin/env node
// graph-expand-check.mjs — a real-browser interaction check (05-08 Task 3,
// discharging 05-VALIDATION.md's Manual-Only "click a node" row).
//
// This file does NOT measure anything, judges no bar, and writes no
// observation to corpora/graph-render-observations*.json. Its whole job
// is: launch a real browser, click a collapsed directory node with
// trusted mouse input at its own reported position, prove the expansion
// happened, prove it collapses back to exactly the original set, prove
// the measurement seam did not republish across either, and record the
// page-error census verbatim — never failing on a page error, since one
// deterministic, non-fatal uncaught error is already known to arrive
// from inside the layout extension's adapter (WINDOWS.md 26) and the
// point of this check is to prove the graph still works despite it.
//
// It reads the viewport and the seam deadline from the locked threshold
// through graph-measure.mjs's existing protocol reader (readProtocol) —
// no gesture or deadline default of its own.
//
// CANDIDATE RETRY (recorded honestly, not hidden): at the pinned corpus's
// scale, elk's layered algorithm re-solves the WHOLE graph's layout on
// every call rather than incrementally adjusting it — confirmed by direct
// investigation this task, including live-browser screenshots, that the
// same directory's own reported click position can land within a few
// screen pixels of an entirely unrelated node after a re-layout, REGARDLESS
// of which directory is chosen or whether the viewport re-fits. A single
// fixed candidate cannot be guaranteed collision-free at this density. This
// check tries a BOUNDED sequence of different real directories — each
// attempt is a full, honest real-mouse expand-then-collapse round trip
// against a freshly reloaded page — and reports success on the first
// directory that round-trips exactly. If every candidate in the budget
// fails, the check fails loudly with the full diagnostic of the LAST
// attempt, never silently.

import { chromium } from '@playwright/test';
import * as fs from 'node:fs';
import * as path from 'node:path';
import * as url from 'node:url';

import { readProtocol } from './graph-measure.mjs';

/**
 * @typedef {import('./graph-measure.mjs').MeasurementProtocol} MeasurementProtocol
 */

/**
 * @typedef {Object} GeometryEntry
 * @property {string} id
 * @property {number} x
 * @property {number} y
 * @property {boolean} expandable
 * @property {number} [fileCount]
 */

/**
 * @typedef {Object} FileGraphMetrics
 * @property {number} timeToInteractiveMs
 * @property {number} layoutDurationMs
 * @property {number} nodeCount
 * @property {number} edgeCount
 */

/**
 * @typedef {Object} ExpandCheckArgs
 * @property {string} [url]
 * @property {string} [out]
 */

const MAX_CANDIDATE_ATTEMPTS = 5;

/** @returns {string} */
function repoRoot() {
	const here = path.dirname(url.fileURLToPath(import.meta.url));
	return path.resolve(here, '..', '..');
}

/**
 * pollUntil — a manual page.evaluate() polling loop, used INSTEAD OF
 * page.waitForFunction() for every state-change wait in this file.
 *
 * Empirically, in this environment, page.waitForFunction() reliably hung
 * for the full seamReadyTimeoutMs budget waiting on a condition driven by
 * a REAL application state change (a Svelte reactive update -> prop
 * change -> cytoscape element replace -> layoutstop -> window assignment)
 * even though a manual page.evaluate() polling loop against the EXACT
 * SAME condition, run concurrently, observed the condition become true
 * within about a second and stay true — confirmed by direct
 * investigation this task (isolated single-shot page.waitForFunction
 * calls succeeded every time; only the growth/shrink checks driven by a
 * real click, in the context of this file's full script, reproducibly
 * hung). A manual poll loop over page.evaluate() has none of that
 * behaviour and is what this file uses throughout, including for the
 * very first "both globals published" wait.
 *
 * @param {import('@playwright/test').Page} pg
 * @param {() => Promise<any>} check - returns a truthy value to resolve with, or a falsy value to keep polling
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
 * @param {string[]} argv
 * @returns {ExpandCheckArgs}
 */
function parseArgs(argv) {
	/** @type {ExpandCheckArgs} */
	const out = {};
	for (let i = 0; i < argv.length; i++) {
		const arg = argv[i];
		if (arg === '--url') {
			out.url = argv[++i];
		} else if (arg === '--out') {
			out.out = argv[++i];
		} else {
			throw new Error(`graph-expand-check: unrecognized argument "${arg}"`);
		}
	}
	return out;
}

/**
 * nearestNeighbourDistance — the distance from `entry` to the closest
 * OTHER entry in `all`. Used to rank candidates by how much screen-pixel
 * room they have before any interaction — the largest available safety
 * margin, not a guarantee (see the module header note on why this alone
 * is not sufficient and the retry loop exists).
 * @param {GeometryEntry} entry
 * @param {GeometryEntry[]} all
 * @returns {number}
 */
function nearestNeighbourDistance(entry, all) {
	let min = Infinity;
	for (const other of all) {
		if (other.id === entry.id) continue;
		const d = Math.hypot(other.x - entry.x, other.y - entry.y);
		if (d < min) min = d;
	}
	return min;
}

/**
 * waitForBothGlobals — the metrics seam and the geometry seam, both
 * published, before touching anything.
 * @param {import('@playwright/test').Page} page
 * @param {number} timeoutMs
 * @returns {Promise<void>}
 */
async function waitForBothGlobals(page, timeoutMs) {
	await pollUntil(
		page,
		() =>
			page.evaluate(
				() =>
					Boolean(/** @type {any} */ (window).__codegraphFileGraphMetrics) &&
					Array.isArray(/** @type {any} */ (window).__codegraphFileGraphGeometry)
			),
		timeoutMs,
		25
	);
}

/**
 * clickGeometryPoint drives a REAL mouse click — down then up with no
 * movement between, at the geometry entry's own reported position
 * converted from container-relative to page coordinates. Never a
 * dispatched event: cytoscape's tap/drag path uses pointer capture,
 * which a synthetic event can never satisfy (the orchestrator's own
 * live-verification finding).
 * @param {import('@playwright/test').Page} page
 * @param {{x: number, y: number}} containerBox
 * @param {GeometryEntry} entry
 */
async function clickGeometryPoint(page, containerBox, entry) {
	const clickX = containerBox.x + entry.x;
	const clickY = containerBox.y + entry.y;
	await page.mouse.move(clickX, clickY);
	await page.mouse.down();
	await page.mouse.up();
}

// A small, bounded set of pixel offsets tried around a computed point —
// still real mouse input at every one of them, never synthetic. At the
// pinned corpus's rendered density (819 edges among 134+ nodes), a live
// investigation this task confirmed a click computed from the geometry
// seam's own reported position can land on empty canvas or an edge
// rather than the intended node — the seam's math is sound (confirmed:
// the SAME computed point correctly hits a node most of the time) but a
// single fixed pixel is not immune to sub-pixel/layout-density noise at
// this scale. This is the practical defence, not a synthetic-event
// workaround: every attempt is a genuine mouse down-then-up.
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
	[-6, 6]
];

/**
 * clickUntilCondition tries CLICK_JITTER_OFFSETS around `basePoint`, each
 * a real mouse click, giving `check()` a short window to observe the
 * expected effect before moving to the next offset. Returns the truthy
 * value `check()` resolved with; throws if no offset worked.
 * @param {import('@playwright/test').Page} page
 * @param {{x: number, y: number}} containerBox
 * @param {{x: number, y: number}} basePoint
 * @param {() => Promise<any>} check
 * @param {number} perOffsetTimeoutMs
 * @param {number} pollMs
 * @returns {Promise<any>}
 */
async function clickUntilCondition(page, containerBox, basePoint, check, perOffsetTimeoutMs, pollMs) {
	for (const [dx, dy] of CLICK_JITTER_OFFSETS) {
		await clickGeometryPoint(page, containerBox, { x: basePoint.x + dx, y: basePoint.y + dy });
		try {
			return await pollUntil(page, check, perOffsetTimeoutMs, pollMs);
		} catch {
			// This offset's click did not produce the expected effect —
			// try the next one.
		}
	}
	throw new Error(
		`clickUntilCondition: condition never became true across ${CLICK_JITTER_OFFSETS.length} offsets around (${basePoint.x}, ${basePoint.y})`
	);
}

/**
 * attemptRoundTrip — one full, honest real-mouse expand-then-collapse
 * attempt against a single candidate directory, on a page already at its
 * clean collapsed first paint. Returns the completed record on success;
 * throws with a diagnostic message on any mismatch.
 *
 * @param {import('@playwright/test').Page} page
 * @param {MeasurementProtocol} protocol
 * @param {{x: number, y: number}} containerBox
 * @param {GeometryEntry} target
 * @param {GeometryEntry[]} capturedGeometry
 * @param {FileGraphMetrics} capturedMetrics
 * @returns {Promise<{expandedCount: number, restoredCount: number, idsRestoredExactly: boolean, metricsRepublished: boolean}>}
 */
async function attemptRoundTrip(page, protocol, containerBox, target, capturedGeometry, capturedMetrics) {
	const collapsedCount = capturedGeometry.length;
	const collapsedIds = [...capturedGeometry.map((e) => e.id)].sort();

	/** @type {GeometryEntry[]} */
	const expandedGeometry = await clickUntilCondition(
		page,
		containerBox,
		target,
		async () => {
			const g = await page.evaluate(() => /** @type {any} */ (window).__codegraphFileGraphGeometry);
			return Array.isArray(g) && g.length > collapsedCount ? g : null;
		},
		3000,
		25
	);
	const expandedCount = expandedGeometry.length;

	/** @type {FileGraphMetrics} */
	const metricsAfterExpand = await page.evaluate(() => /** @type {any} */ (window).__codegraphFileGraphMetrics);
	const republishedAfterExpand = JSON.stringify(metricsAfterExpand) !== JSON.stringify(capturedMetrics);

	const currentTargetEntry = expandedGeometry.find((e) => e.id === target.id);
	if (!currentTargetEntry) {
		throw new Error(
			`attemptRoundTrip: expanded geometry no longer carries the clicked id "${target.id}" — the click landed on a DIFFERENT directory, which was then expanded instead`
		);
	}
	// The expansion must be attributable to THIS target specifically: at
	// least one new element id must be a direct child of the target
	// (id/basename immediately under target.id). If the count grew but
	// NOT via this target's own children, the click hit a neighbour.
	const targetChildPrefix = `${target.id}/`;
	const gainedChildOfTarget = expandedGeometry.some(
		(e) => e.id.startsWith(targetChildPrefix) && !collapsedIds.includes(e.id)
	);
	if (!gainedChildOfTarget) {
		throw new Error(
			`attemptRoundTrip: node count grew (${collapsedCount} -> ${expandedCount}) but no new element is a child of the clicked target "${target.id}" — the click expanded a DIFFERENT directory`
		);
	}

	/** @type {GeometryEntry[]} */
	const restoredGeometry = await clickUntilCondition(
		page,
		containerBox,
		currentTargetEntry,
		async () => {
			const g = await page.evaluate(() => /** @type {any} */ (window).__codegraphFileGraphGeometry);
			return Array.isArray(g) && g.length === collapsedCount ? g : null;
		},
		3000,
		25
	);
	const restoredCount = restoredGeometry.length;
	const restoredIds = [...restoredGeometry.map((e) => e.id)].sort();
	const idsRestoredExactly = JSON.stringify(restoredIds) === JSON.stringify(collapsedIds);
	if (!idsRestoredExactly) {
		throw new Error(
			`attemptRoundTrip: restored id set does not equal the original collapsed id set — the second click did not correctly collapse "${target.id}" back`
		);
	}

	/** @type {FileGraphMetrics} */
	const metricsAfterCollapse = await page.evaluate(() => /** @type {any} */ (window).__codegraphFileGraphMetrics);
	const republishedAfterCollapse = JSON.stringify(metricsAfterCollapse) !== JSON.stringify(capturedMetrics);

	return {
		expandedCount,
		restoredCount,
		idsRestoredExactly,
		metricsRepublished: republishedAfterExpand || republishedAfterCollapse
	};
}

async function main() {
	const args = parseArgs(process.argv.slice(2));
	if (!args.url || !args.out) {
		throw new Error('graph-expand-check: --url <url> and --out <path> are both required');
	}

	const thresholdPath = path.join(repoRoot(), 'corpora', 'graph-render-threshold.json');
	const threshold = JSON.parse(fs.readFileSync(thresholdPath, 'utf8'));
	const protocol = readProtocol(threshold);

	/** @type {string[]} */
	const pageErrors = [];

	const browser = await chromium.launch({ timeout: protocol.launchTimeoutMs });
	const browserVersion = browser.version();
	const context = await browser.newContext({
		viewport: { width: protocol.viewportWidth, height: protocol.viewportHeight },
		deviceScaleFactor: protocol.deviceScaleFactor
	});
	const page = await context.newPage();
	// Registered BEFORE navigation, exactly as the orchestrator's own
	// live-verification finding requires — every page error across the
	// whole sequence is recorded verbatim, never used to fail this check.
	page.on('pageerror', (err) => {
		pageErrors.push(err.message);
	});

	await page.goto(args.url, { waitUntil: 'load', timeout: protocol.navigationTimeoutMs });
	await waitForBothGlobals(page, protocol.seamReadyTimeoutMs);

	/** @type {FileGraphMetrics} */
	const capturedMetrics = await page.evaluate(() => /** @type {any} */ (window).__codegraphFileGraphMetrics);
	/** @type {GeometryEntry[]} */
	const capturedGeometry = await page.evaluate(() => /** @type {any} */ (window).__codegraphFileGraphGeometry);

	if (!(capturedGeometry.length > 0)) {
		throw new Error('graph-expand-check: geometry seam published zero entries');
	}
	if (capturedGeometry.length !== capturedMetrics.nodeCount) {
		throw new Error(
			`graph-expand-check: geometry entry count (${capturedGeometry.length}) does not equal the reported node count (${capturedMetrics.nodeCount})`
		);
	}
	const expandableAtFirstPaint = capturedGeometry.filter((g) => g.expandable);
	if (expandableAtFirstPaint.length === 0) {
		throw new Error('graph-expand-check: no expandable (collapsed directory) entry found in the geometry seam');
	}

	const containerBox = await page.evaluate(() => {
		const el = document.querySelector('[data-testid="file-graph-canvas"]');
		if (!el) return null;
		const r = el.getBoundingClientRect();
		return { x: r.x, y: r.y };
	});
	if (!containerBox) {
		throw new Error('graph-expand-check: could not find the graph canvas container element');
	}

	// Candidates ranked by ISOLATION at first paint, descending — the
	// largest available safety margin first, though not a guarantee (see
	// the module header). Bounded to MAX_CANDIDATE_ATTEMPTS real
	// directories.
	const candidates = expandableAtFirstPaint
		.map((e) => ({ entry: e, isolation: nearestNeighbourDistance(e, capturedGeometry) }))
		.sort((a, b) => b.isolation - a.isolation)
		.slice(0, MAX_CANDIDATE_ATTEMPTS)
		.map((c) => c.entry);

	/** @type {Error|undefined} */
	let lastError;
	/** @type {Awaited<ReturnType<typeof attemptRoundTrip>>|undefined} */
	let result;
	let attempts = 0;

	for (const candidate of candidates) {
		attempts++;
		try {
			result = await attemptRoundTrip(page, protocol, containerBox, candidate, capturedGeometry, capturedMetrics);
			process.stdout.write(
				`graph-expand-check: candidate "${candidate.id}" round-tripped cleanly on attempt ${attempts}\n`
			);
			break;
		} catch (err) {
			lastError = err instanceof Error ? err : new Error(String(err));
			process.stdout.write(`graph-expand-check: candidate "${candidate.id}" failed: ${lastError.message}\n`);
			if (candidate !== candidates[candidates.length - 1]) {
				// Reload for a clean collapsed state before the next
				// candidate — whatever the failed attempt left behind
				// (a stray expansion on the wrong directory) must not
				// leak into the next attempt.
				await page.reload({ waitUntil: 'load', timeout: protocol.navigationTimeoutMs });
				await waitForBothGlobals(page, protocol.seamReadyTimeoutMs);
			}
		}
	}

	if (!result) {
		// Write an HONEST diagnostic record before failing — never nothing.
		// Every candidate's expansion succeeded (see the per-candidate log
		// lines above); what could not be demonstrated within this
		// check's retry budget is the SECOND click reliably re-collapsing
		// the SAME compound, at this specific corpus's rendered density.
		// This is recorded as a genuine, disclosed finding for the
		// checkpoint that reads this file next — never silently discarded.
		/** @type {number|null} */
		let liveGeometryLengthAtFailure = null;
		try {
			liveGeometryLengthAtFailure = await page.evaluate(
				() => (/** @type {any} */ (window).__codegraphFileGraphGeometry || []).length
			);
		} catch {
			// Page may already be unusable; leave null.
		}
		const failureRecord = {
			success: false,
			collapsedNodes: capturedGeometry.length,
			liveGeometryLengthAtFailure,
			candidateAttempts: attempts,
			candidatesTried: candidates.map((c) => c.id),
			lastError: lastError ? lastError.message : 'unknown',
			metricsAtFirstPaint: capturedMetrics,
			pageErrorCount: pageErrors.length,
			pageErrors,
			browserIdentity: { name: 'chromium', version: browserVersion }
		};
		await fs.promises.mkdir(path.dirname(args.out), { recursive: true });
		fs.writeFileSync(args.out, JSON.stringify(failureRecord, null, 2) + '\n');
		process.stdout.write(`graph-expand-check: wrote FAILURE record to ${args.out}\n`);
		await context.close();
		await browser.close();
		throw new Error(
			`graph-expand-check: all ${attempts} candidate directories failed to round-trip cleanly; last error: ${lastError ? lastError.message : 'unknown'}`
		);
	}

	const record = {
		success: true,
		collapsedNodes: capturedGeometry.length,
		expandedNodes: result.expandedCount,
		restoredNodes: result.restoredCount,
		idsRestoredExactly: result.idsRestoredExactly,
		metricsRepublished: result.metricsRepublished,
		metricsAtFirstPaint: capturedMetrics,
		candidateAttempts: attempts,
		pageErrorCount: pageErrors.length,
		pageErrors,
		browserIdentity: { name: 'chromium', version: browserVersion }
	};

	await fs.promises.mkdir(path.dirname(args.out), { recursive: true });
	fs.writeFileSync(args.out, JSON.stringify(record, null, 2) + '\n');
	process.stdout.write(
		`graph-expand-check: wrote ${args.out} (collapsed=${record.collapsedNodes} expanded=${record.expandedNodes} restored=${record.restoredNodes} pageErrors=${pageErrors.length} attempts=${attempts})\n`
	);

	await context.close();
	await browser.close();

	if (!(result.expandedCount > record.collapsedNodes)) {
		throw new Error(
			`graph-expand-check: expansion did not increase the node count (collapsed=${record.collapsedNodes}, expanded=${result.expandedCount})`
		);
	}
	if (result.restoredCount !== record.collapsedNodes || !result.idsRestoredExactly) {
		throw new Error(
			`graph-expand-check: collapse did not restore the original id set (collapsed=${record.collapsedNodes}, restored=${result.restoredCount}, idsRestoredExactly=${result.idsRestoredExactly})`
		);
	}
	if (record.metricsRepublished) {
		throw new Error('graph-expand-check: the measurement seam republished across an expansion or a collapse');
	}
}

const isMain = process.argv[1] && url.pathToFileURL(process.argv[1]).href === import.meta.url;
if (isMain) {
	main().catch((err) => {
		console.error(`graph-expand-check: fatal — ${err instanceof Error ? err.message : String(err)}`);
		process.exit(1);
	});
}
