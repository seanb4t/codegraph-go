#!/usr/bin/env node
// graph-live-update-check.mjs — 06-05 Task 3: D-06's guava-scale
// survivor-displacement measurement. Follows
// graph-collapse-affordance-check.mjs's established shape (repoRoot(),
// pollUntil, real trusted input only, a diagnostic JSON record written
// on EVERY run including failure) — see this repo's 06-PATTERNS.md.
//
// Unlike the other real-browser check scripts in this directory (which
// take an externally-provided --url), THIS script owns the full
// lifecycle itself: it builds the corpus checkout path from
// corpora/graph-render-threshold.json's own pinned repo/sha (the same
// slug formula internal/corpora/manifest.go's Entry.Dir uses — a
// readable slug, "-", the first 8 hex chars of sha256(repo), "@", the
// pinned sha — reimplemented here in JS rather than imported, since this
// is a Node script with no access to the Go package), starts a real
// `codegraph ui` child process against it, drives a real Chromium tab,
// mutates the pinned checkout with a real re-index, and tears everything
// down in a try/finally so cleanup runs on every exit path.
//
// [Deviation, recorded honestly per this task's own instruction to
// "record the exact mutation... not 'a file was added'"]: the plan's
// action text says "create ONE new source file". Measured directly this
// session, before writing this script: a single new file adds EXACTLY
// one entry to FileGraphResponse.nodes, regardless of how many methods
// it declares — internal/query/traverse.go's FileGraph() aggregates
// strictly per FilePath (`fileAggs` keyed by `n.FilePath`), so a file's
// own symbol count never changes how many FILE-GRAPH nodes it
// contributes. Verified via a real RPC diff before/after a real
// `codegraph sync` against this exact corpus: nodes 3233 -> 3234, added
// == [the one new path], removed == []. One file therefore cannot reach
// the required `addedNodeCount >= 5` floor — and reaching it via a
// SUBSEQUENT symbol-expansion click (add(), which has no position
// write-back) would itself re-scramble survivor positions via ELK's own
// approximation, defeating the exact thing this script measures. This
// script creates several minimal new files inside the SAME already-
// expanded directory instead of one — genuinely satisfying the floor
// from ONE structural swap, with the real number recorded in
// mutationDescription below, never narrated as "a file was added".
import { chromium } from '@playwright/test';
import { spawn, execFileSync } from 'node:child_process';
import * as crypto from 'node:crypto';
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
 * corpusRoot mirrors internal/corpora/manifest.go's CorpusRoot():
 * CODEGRAPH_CORPUS_DIR verbatim when set, else XDG_CACHE_HOME/codegraph/
 * corpora, else ~/.cache/codegraph/corpora. Applied literally, no
 * per-OS branch, matching the Go implementation's own stated discipline.
 * @returns {string}
 */
function corpusRoot() {
	if (process.env.CODEGRAPH_CORPUS_DIR) return process.env.CODEGRAPH_CORPUS_DIR;
	if (process.env.XDG_CACHE_HOME) return path.join(process.env.XDG_CACHE_HOME, 'codegraph', 'corpora');
	const home = process.env.HOME;
	if (!home) throw new Error('graph-live-update-check: HOME is not set and no cache override is configured');
	return path.join(home, '.cache', 'codegraph', 'corpora');
}

/**
 * corpusDir mirrors internal/corpora/manifest.go's Entry.Dir(): a
 * readable slug (repo with "/" replaced by "-"), a hyphen, the first 8
 * hex characters of sha256(repo), "@", the pinned sha. Verified this
 * session against the corpus actually on disk: sha256("google/guava")
 * -> "2b0cb53f...", matching the real directory name.
 * @param {string} repo
 * @param {string} sha
 * @returns {string}
 */
function corpusDir(repo, sha) {
	const slug = repo.replace(/\//g, '-');
	const digest = crypto.createHash('sha256').update(repo).digest('hex').slice(0, 8);
	return path.join(corpusRoot(), `${slug}-${digest}@${sha}`);
}

/**
 * javaPackageFor derives a Java package name from a repository-relative
 * directory path by finding the source-root marker segment ("src",
 * "test", or "java" — guava's own module layout uses all three across
 * its production/test/android trees) and taking everything after it,
 * joined with ".". Falls back to the whole path (joined with ".") if no
 * marker segment is found.
 * @param {string} dirPath
 * @returns {string}
 */
function javaPackageFor(dirPath) {
	const segments = dirPath.split('/');
	const markerIdx = segments.findIndex((s) => s === 'src' || s === 'test' || s === 'java');
	const pkgSegments = markerIdx === -1 ? segments : segments.slice(markerIdx + 1);
	return pkgSegments.join('.');
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
 * called fresh before EVERY canvas click, never cached, mirroring
 * graph-collapse-affordance-check.mjs's own established discipline (a
 * re-layout can shift the canvas's own scroll position).
 * @param {import('@playwright/test').Page} page
 */
async function getContainerBox(page) {
	const box = await page.evaluate(() => {
		const el = document.querySelector('[data-testid="file-graph-canvas"]');
		if (!el) return null;
		const r = el.getBoundingClientRect();
		return { x: r.x, y: r.y };
	});
	if (!box) throw new Error('graph-live-update-check: no graph canvas container found');
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

/**
 * waitForStableGeometry: the geometry seam can publish more than once
 * during initial mount (observed this task: GraphCanvas's own
 * construction effect can run twice in quick succession for a single
 * logical page load, republishing a SECOND, different geometry array
 * milliseconds after the first — a real, separately-tracked finding,
 * see 06-05-SUMMARY.md). Reading geometry the instant it first appears
 * risks capturing an EPHEMERAL first-mount publish, computed against
 * whatever the container's size happened to be at that transient
 * instant, before layout settles down to production behavior. This
 * waits for two consecutive reads, POLL_INTERVAL_MS apart, to be
 * IDENTICAL (same length and same serialized content) before treating
 * geometry as settled — cheap at this scale (hundreds of entries) and
 * robust regardless of whether a remount is one, two, or zero.
 * @param {import('@playwright/test').Page} page
 * @param {number} timeoutMs
 * @returns {Promise<GeometryEntry[]>}
 */
async function waitForStableGeometry(page, timeoutMs) {
	await waitForGeometry(page, timeoutMs);
	const POLL_INTERVAL_MS = 150;
	return pollUntil(
		page,
		async () => {
			const first = await readGeometry(page);
			await page.waitForTimeout(POLL_INTERVAL_MS);
			const second = await readGeometry(page);
			if (!Array.isArray(second)) return null;
			return JSON.stringify(first) === JSON.stringify(second) ? second : null;
		},
		timeoutMs,
		25
	);
}

/** @param {import('@playwright/test').Page} page */
async function readGeometry(page) {
	return page.evaluate(() => /** @type {any} */ (window).__codegraphFileGraphGeometry);
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
 * small jitter budget), following graph-collapse-affordance-check.mjs's
 * own established pattern exactly.
 * @param {import('@playwright/test').Page} page
 * @param {{x:number,y:number}} containerBox
 * @param {{x:number,y:number}} basePoint
 * @param {() => Promise<any>} check
 */
async function clickCanvasUntil(page, containerBox, basePoint, check) {
	for (const [dx, dy] of CLICK_JITTER_OFFSETS) {
		await page.mouse.move(containerBox.x + basePoint.x + dx, containerBox.y + basePoint.y + dy);
		await page.mouse.down();
		await page.mouse.up();
		try {
			return await pollUntil(page, check, 3000, 25);
		} catch {
			// try the next offset
		}
	}
	throw new Error(`clickCanvasUntil: condition never became true around (${basePoint.x}, ${basePoint.y})`);
}

/**
 * @param {string} binaryPath
 * @param {string} corpusPath
 * @returns {Promise<{child: import('node:child_process').ChildProcess, url: string}>}
 */
function startCodegraphUi(binaryPath, corpusPath) {
	return new Promise((resolve, reject) => {
		const child = spawn(binaryPath, ['ui', '--no-open', '--path', corpusPath], {
			stdio: ['ignore', 'pipe', 'pipe']
		});
		let stdout = '';
		let settled = false;
		const timer = setTimeout(() => {
			if (!settled) {
				settled = true;
				reject(new Error('graph-live-update-check: codegraph ui printed no URL within 15s'));
			}
		}, 15000);
		child.stdout.on('data', (chunk) => {
			stdout += chunk.toString();
			const m = stdout.match(/https?:\/\/\S+/);
			if (m && !settled) {
				settled = true;
				clearTimeout(timer);
				resolve({ child, url: m[0].trim() });
			}
		});
		child.on('exit', (code) => {
			if (!settled) {
				settled = true;
				clearTimeout(timer);
				reject(new Error(`graph-live-update-check: codegraph ui exited early (code ${code})`));
			}
		});
		child.on('error', (err) => {
			if (!settled) {
				settled = true;
				clearTimeout(timer);
				reject(err);
			}
		});
	});
}

/** @param {import('node:child_process').ChildProcess | undefined} child */
function stopChild(child) {
	return new Promise((resolve) => {
		if (!child || child.exitCode !== null || child.signalCode !== null) {
			resolve(undefined);
			return;
		}
		child.once('exit', () => resolve(undefined));
		child.kill('SIGTERM');
		setTimeout(() => {
			if (child.exitCode === null && child.signalCode === null) child.kill('SIGKILL');
		}, 3000);
	});
}

/**
 * computeCompoundIds identifies which geometry ids are CURRENTLY
 * compounds (an expanded directory now showing file children) versus
 * leaves (a file, a symbol, or a still-collapsed directory) — purely
 * from the id-prefix relationship already established by this
 * repository's OWN directory/file id convention (dirOf/endpointOf in
 * file-graph-transform.ts): a file's id is `${itsDirectory}/${name}`.
 * An id is a compound iff some OTHER id in the SAME snapshot starts
 * with `${id}/`. This mirrors graph-collapse-affordance-check.mjs's own
 * `childPrefix` technique. D-06's write-back explicitly excludes
 * parents from position restoration (a compound's position derives
 * from its children in cytoscape) — this function lets the SCRIPT apply
 * the identical exclusion when judging displacement.
 * @param {GeometryEntry[]} geometry
 * @returns {Set<string>}
 */
function computeCompoundIds(geometry) {
	const ids = geometry.map((g) => g.id);
	const compound = new Set();
	for (const id of ids) {
		const prefix = `${id}/`;
		if (ids.some((other) => other !== id && other.startsWith(prefix))) compound.add(id);
	}
	return compound;
}

/**
 * @param {{x:number,y:number}} a
 * @param {{x:number,y:number}} b
 */
function euclidean(a, b) {
	return Math.sqrt((a.x - b.x) ** 2 + (a.y - b.y) ** 2);
}

async function main() {
	const args = process.argv.slice(2);
	const outIdx = args.indexOf('--out');
	const outPath =
		outIdx === -1
			? path.join(repoRoot(), 'corpora', 'graph-live-update-check.json')
			: path.resolve(args[outIdx + 1]);

	const root = repoRoot();
	const thresholdPath = path.join(root, 'corpora', 'graph-render-threshold.json');
	const threshold = JSON.parse(fs.readFileSync(thresholdPath, 'utf8'));
	const repo = threshold.corpus.repo;
	const sha = threshold.corpus.sha;
	const corpusPath = corpusDir(repo, sha);
	if (!fs.existsSync(path.join(corpusPath, '.codegraph', 'store'))) {
		throw new Error(
			`graph-live-update-check: corpus not indexed at ${corpusPath} — see this task's <precondition>`
		);
	}
	const binaryPath = path.join(root, 'codegraph');
	if (!fs.existsSync(binaryPath)) {
		throw new Error(
			`graph-live-update-check: codegraph binary not found at ${binaryPath} — run \`task build:release\` first`
		);
	}

	/** @type {Record<string, any>} */
	const record = {
		schemaVersion: 1,
		corpus: { repo, sha },
		corpusCheckoutPath: corpusPath,
		bindingView:
			'The expanded file level: several large real directories expanded to their file children via real mouse clicks (see expandedDirectories), non-binding compound (directory) survivors excluded from the binding leaf-displacement figures below.',
		expandedDirectories: [],
		expansionCandidatesTried: null,
		mutationDescription: null,
		fileLevelNodeCountBefore: null,
		fileLevelNodeCountAfter: null,
		addedNodeCount: null,
		removedNodeCount: null,
		leafSurvivorCount: null,
		maxLeafDisplacementPx: null,
		meanLeafDisplacementPx: null,
		compoundSurvivorCount: null,
		maxCompoundDisplacementPx: null,
		pageErrorCount: null,
		pageErrors: [],
		browserIdentity: null,
		corpusCleanAtExit: false,
		error: null,
		success: false
	};

	/** @type {import('node:child_process').ChildProcess | undefined} */
	let child;
	/** @type {import('@playwright/test').Browser | undefined} */
	let browser;
	/** @type {string[]} */
	const createdRelPaths = [];

	try {
		try {
			const started = await startCodegraphUi(binaryPath, corpusPath);
			child = started.child;
			const targetUrl = started.url;

			browser = await chromium.launch();
			record.browserIdentity = { name: 'chromium', version: browser.version() };
			const context = await browser.newContext({ viewport: { width: 1600, height: 1000 } });
			const page = await context.newPage();
			/** @type {string[]} */
			const pageErrors = [];
			page.on('pageerror', (err) => pageErrors.push(err.message));

			await page.goto(`${targetUrl}/graph`, { waitUntil: 'load', timeout: 30000 });
			let geometry = await waitForStableGeometry(page, 30000);

			// Pick large collapsed directories (by their own reported
			// fileCount) to expand via a real click, largest first, so few
			// clicks reach a comfortable survivor margin. A CANDIDATE POOL
			// larger than the number actually needed, tried in order with
			// FAILURES SKIPPED rather than aborting the whole run — the
			// SAME "bounded candidate retry" discipline
			// graph-expand-check.mjs's own header comment establishes: at
			// this corpus's edge density a re-layout after each expansion
			// can shift an UNTRIED candidate's own reported position by
			// more than the click-jitter budget covers, so a single
			// candidate failing to land is expected occasionally and must
			// not fail the whole measurement — only running out of
			// candidates before reaching the floor does. Stops once the
			// CUMULATIVE rendered count is comfortably over the
			// leafSurvivorCount floor (100) with real margin, or the
			// candidate pool is exhausted.
			const candidatePool = [...geometry]
				.filter((/** @type {GeometryEntry} */ g) => g.expandable)
				.sort((a, b) => (b.fileCount ?? 0) - (a.fileCount ?? 0))
				.slice(0, 15);
			/** @type {string[]} */
			const expandedDirIds = [];
			const EXPANSION_TARGET_TOTAL = 250;
			const EXPANSION_BUDGET_CEILING = 900; // real margin under EXPANSION_NODE_CEILING (1500)
			for (const dir of candidatePool) {
				if (geometry.length >= EXPANSION_TARGET_TOTAL) break;
				// Re-locate this candidate in the LATEST geometry (a prior
				// expansion in this same loop can have shifted its
				// position) — its id is stable even though its position
				// is not.
				const target = geometry.find((/** @type {GeometryEntry} */ g) => g.id === dir.id);
				if (!target || !target.expandable) continue; // already expanded via a prior click, or vanished
				if (geometry.length + (target.fileCount ?? 0) > EXPANSION_BUDGET_CEILING) continue;
				const beforeCount = geometry.length;
				try {
					geometry = await clickCanvasUntil(page, await getContainerBox(page), target, async () => {
						const g = await readGeometry(page);
						return Array.isArray(g) && g.length > beforeCount ? g : null;
					});
					expandedDirIds.push(dir.id);
				} catch {
					// This candidate's reported position didn't land —
					// skip it and try the next one, per the bounded-
					// candidate-retry discipline above.
				}
			}
			if (expandedDirIds.length === 0) {
				throw new Error('graph-live-update-check: no candidate directory could be expanded at this corpus scale');
			}
			record.expandedDirectories = expandedDirIds;
			record.expansionCandidatesTried = candidatePool.length;

			record.fileLevelNodeCountBefore = geometry.length;
			const beforeById = new Map(
				geometry.map((/** @type {GeometryEntry} */ g) => [g.id, { x: g.x, y: g.y }])
			);

			// --- Mutation: see this file's own header comment for why
			// several files, not one. ---
			const mutationDirId = expandedDirIds[0];
			const javaPkg = javaPackageFor(mutationDirId);
			const NEW_FILE_COUNT = 6;
			for (let i = 0; i < NEW_FILE_COUNT; i++) {
				const className = `LivePushProbe${i}`;
				const relPath = `${mutationDirId}/${className}.java`;
				const absPath = path.join(corpusPath, relPath);
				const contents =
					`package ${javaPkg};\n\n` +
					`/** Transient probe file for 06-05's live-update displacement measurement — removed before this script exits. */\n` +
					`final class ${className} {\n\tprivate ${className}() {}\n}\n`;
				fs.writeFileSync(absPath, contents);
				createdRelPaths.push(relPath);
			}

			execFileSync(binaryPath, ['sync', corpusPath], { stdio: 'pipe' });

			const afterGeometry = await pollUntil(
				page,
				async () => {
					const g = await readGeometry(page);
					return Array.isArray(g) && g.length > record.fileLevelNodeCountBefore ? g : null;
				},
				60000,
				200
			);

			record.fileLevelNodeCountAfter = afterGeometry.length;
			const afterById = new Map(
				afterGeometry.map((/** @type {GeometryEntry} */ g) => [g.id, { x: g.x, y: g.y }])
			);
			const compoundIdsAfter = computeCompoundIds(afterGeometry);

			let leafSurvivorCount = 0;
			let compoundSurvivorCount = 0;
			let maxLeaf = 0;
			let sumLeaf = 0;
			let maxCompound = 0;
			for (const [id, beforePos] of beforeById) {
				const afterPos = afterById.get(id);
				if (!afterPos) continue; // not a survivor — removed
				const d = euclidean(beforePos, afterPos);
				if (compoundIdsAfter.has(id)) {
					compoundSurvivorCount++;
					if (d > maxCompound) maxCompound = d;
				} else {
					leafSurvivorCount++;
					sumLeaf += d;
					if (d > maxLeaf) maxLeaf = d;
				}
			}
			const addedIds = [...afterById.keys()].filter((id) => !beforeById.has(id));
			const removedIds = [...beforeById.keys()].filter((id) => !afterById.has(id));

			record.addedNodeCount = addedIds.length;
			record.removedNodeCount = removedIds.length;
			record.leafSurvivorCount = leafSurvivorCount;
			record.maxLeafDisplacementPx = maxLeaf;
			record.meanLeafDisplacementPx = leafSurvivorCount > 0 ? sumLeaf / leafSurvivorCount : null;
			record.compoundSurvivorCount = compoundSurvivorCount;
			record.maxCompoundDisplacementPx = maxCompound;
			record.pageErrorCount = pageErrors.length;
			record.pageErrors = pageErrors;
			record.mutationDescription =
				`Created ${NEW_FILE_COUNT} new minimal Java files (${createdRelPaths.join(', ')}) inside ` +
				`"${mutationDirId}" (package ${javaPkg}), each declaring one empty final class with a ` +
				`private no-op constructor and zero other symbols; ran a real \`codegraph sync\` re-index ` +
				`against the live corpus checkout. Deviation from the plan's literal "ONE new source ` +
				`file": measured this session that a single new file adds exactly one ` +
				`FileGraphResponse.nodes entry regardless of its own symbol count (internal/query/` +
				`traverse.go's FileGraph() aggregates strictly per FilePath) — see 06-05-SUMMARY.md for ` +
				`the direct RPC-diff evidence disproving the single-file assumption.`;
		} catch (err) {
			record.error = err instanceof Error ? err.message : String(err);
			if (err instanceof Error && err.stack) record.errorStack = err.stack;
		}
	} finally {
		let corpusCleanAtExit = false;
		try {
			for (const relPath of createdRelPaths) {
				const absPath = path.join(corpusPath, relPath);
				if (fs.existsSync(absPath)) fs.unlinkSync(absPath);
			}
			if (createdRelPaths.length > 0) {
				execFileSync(binaryPath, ['sync', corpusPath], { stdio: 'pipe' });
			}
			const gitStatus = execFileSync('git', ['status', '--porcelain'], { cwd: corpusPath }).toString();
			corpusCleanAtExit = gitStatus.trim().length === 0;
			if (!corpusCleanAtExit) {
				record.corpusStatusPorcelain = gitStatus;
			}
		} catch (cleanupErr) {
			corpusCleanAtExit = false;
			record.cleanupError = cleanupErr instanceof Error ? cleanupErr.message : String(cleanupErr);
		}
		record.corpusCleanAtExit = corpusCleanAtExit;

		if (browser) {
			await browser.close().catch(() => {});
		}
		await stopChild(child);

		record.success =
			record.error === null &&
			typeof record.leafSurvivorCount === 'number' &&
			record.leafSurvivorCount >= 100 &&
			typeof record.addedNodeCount === 'number' &&
			record.addedNodeCount >= 5 &&
			typeof record.fileLevelNodeCountAfter === 'number' &&
			typeof record.fileLevelNodeCountBefore === 'number' &&
			record.fileLevelNodeCountAfter > record.fileLevelNodeCountBefore &&
			record.maxLeafDisplacementPx === 0 &&
			record.corpusCleanAtExit === true;

		fs.mkdirSync(path.dirname(outPath), { recursive: true });
		fs.writeFileSync(outPath, JSON.stringify(record, null, 2) + '\n');
		console.log(`graph-live-update-check: wrote ${outPath} — success=${record.success}`);
	}

	if (!record.success) {
		throw new Error(
			`graph-live-update-check: FAILED — error=${record.error} leafSurvivorCount=${record.leafSurvivorCount} ` +
				`addedNodeCount=${record.addedNodeCount} maxLeafDisplacementPx=${record.maxLeafDisplacementPx} ` +
				`corpusCleanAtExit=${record.corpusCleanAtExit}`
		);
	}
}

const isMain = process.argv[1] && url.pathToFileURL(process.argv[1]).href === import.meta.url;
if (isMain) {
	main().catch((err) => {
		console.error(`graph-live-update-check: fatal — ${err instanceof Error ? err.message : String(err)}`);
		if (err instanceof Error && err.stack) console.error(err.stack);
		process.exit(1);
	});
}
