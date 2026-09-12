#!/usr/bin/env node
// breadcrumb-check.mjs — 09-03 Task 3 (BRW-10 success criterion 1): proves
// the sticky "which symbol am I inside" breadcrumb in a REAL Chromium
// against THIS REPOSITORY's own real index, with an oracle that is
// INDEPENDENT of the SPA's own derivation module (that module is never
// imported here — a wrong derivation in the SPA must not be able to
// agree with itself).
//
// This script owns the `codegraph ui` child process lifecycle itself
// (graph-live-update-check.mjs's startCodegraphUi shape) rather than
// taking an externally-provided --url, and follows
// graph-collapse-affordance-check.mjs's other conventions: real
// Chromium, real input (page.mouse.wheel — never a dispatched synthetic
// event or a fixed multi-second sleep), pollUntil (poll, don't sleep),
// and a diagnostic JSON record written on EVERY exit path including
// failure.
//
// The SAME script is also this plan's RED instrument: run against a
// pre-fix binary (built from the commit before this plan's first SPA
// commit), the breadcrumb element never appears, `breadcrumbPresent:
// false` is recorded, and the script exits 1 — pasted verbatim into
// 09-MUTATION-LOG.md family (a). There is no tracked-file mutation and
// therefore no revert step for this family (07-MUTATION-LOG.md family
// (a) precedent): the RED is the ABSENCE of the fix, not a mutation of
// otherwise-correct code.
import { chromium } from '@playwright/test';
import { spawn } from 'node:child_process';
import * as fs from 'node:fs';
import * as path from 'node:path';
import * as url from 'node:url';

/** @returns {string} */
function repoRoot() {
	const here = path.dirname(url.fileURLToPath(import.meta.url));
	return path.resolve(here, '..', '..');
}

/**
 * pollUntil — poll-don't-sleep, the same convention
 * graph-collapse-affordance-check.mjs's own pollUntil establishes.
 * @param {() => Promise<any>} check
 * @param {number} timeoutMs
 * @param {number} pollMs
 * @returns {Promise<any>}
 */
async function pollUntil(check, timeoutMs, pollMs) {
	const deadlineAt = Date.now() + timeoutMs;
	for (;;) {
		const value = await check();
		if (value) return value;
		if (Date.now() >= deadlineAt) {
			throw new Error(`pollUntil: condition did not become true within ${timeoutMs}ms`);
		}
		await new Promise((resolve) => setTimeout(resolve, pollMs));
	}
}

/**
 * startCodegraphUi — the same shape as graph-live-update-check.mjs's
 * function of the same name: spawn `<binary> ui --no-open --path <repo>`,
 * resolve once the printed URL appears in stdout, reject on early exit.
 * @param {string} binaryPath
 * @param {string} repoPath
 * @returns {Promise<{child: import('node:child_process').ChildProcess, url: string}>}
 */
function startCodegraphUi(binaryPath, repoPath) {
	return new Promise((resolve, reject) => {
		const child = spawn(binaryPath, ['ui', '--no-open', '--path', repoPath], {
			stdio: ['ignore', 'pipe', 'pipe']
		});
		let stdout = '';
		let settled = false;
		const timer = setTimeout(() => {
			if (!settled) {
				settled = true;
				reject(new Error('breadcrumb-check: codegraph ui printed no URL within 15s'));
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
				reject(new Error(`breadcrumb-check: codegraph ui exited early (code ${code})`));
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
 * @typedef {Object} OracleSymbol
 * @property {string} name
 * @property {number} startLine
 * @property {number} endLine
 */

/**
 * oracleInnermost — an INDEPENDENT re-implementation of the SPA's own
 * "innermost containing range" derivation (sort-based rather than
 * iterative min-tracking, deliberately a different shape), so this
 * script's expectation cannot silently share a bug with the code under
 * test. Same contract: normalise endLine<=startLine to a single-line
 * range, keep ranges containing `line` inclusively, tightest span wins,
 * ties break toward the later-starting range, a full tie toward the
 * later index — matching FileSymbols' own (startLine, then name) sort
 * order (D-02, D-03).
 * @param {OracleSymbol[]} symbols
 * @param {number} line
 * @returns {OracleSymbol | null}
 */
function oracleInnermost(symbols, line) {
	const containing = symbols
		.map((s, index) => ({
			s,
			index,
			start: s.startLine,
			end: Math.max(s.startLine, s.endLine)
		}))
		.filter(({ start, end }) => line >= start && line <= end);
	if (containing.length === 0) return null;
	containing.sort((a, b) => {
		const spanA = a.end - a.start;
		const spanB = b.end - b.start;
		if (spanA !== spanB) return spanA - spanB; // tighter span first
		if (a.start !== b.start) return b.start - a.start; // later-starting first
		return b.index - a.index; // later input index first
	});
	return containing[0].s;
}

async function main() {
	const args = process.argv.slice(2);
	/** @param {string} flag @param {string} fallback */
	const argOr = (flag, fallback) => {
		const i = args.indexOf(flag);
		return i === -1 ? fallback : args[i + 1];
	};

	const root = repoRoot();
	const outPath = path.resolve(argOr('--out', path.join(root, 'corpora', 'breadcrumb-check.json')));
	const binaryPath = path.resolve(argOr('--binary', path.join(root, 'codegraph')));
	const repoPath = path.resolve(argOr('--repo', root));
	const file = argOr('--file', 'internal/query/node.go');

	/** @type {Record<string, any>} */
	const record = {
		schemaVersion: 1,
		file,
		binaryPath,
		repoPath,
		symbolCount: null,
		breadcrumbPresent: false,
		observations: [],
		nonEmptyObservations: 0,
		emptyObservations: 0,
		pageErrorCount: null,
		pageErrors: [],
		browserIdentity: null,
		error: null,
		success: false
	};

	if (!fs.existsSync(binaryPath)) {
		throw new Error(`breadcrumb-check: binary not found at ${binaryPath} — run \`task build:release\` first`);
	}
	if (!fs.existsSync(path.join(repoPath, '.codegraph', 'store'))) {
		throw new Error(
			`breadcrumb-check: no index at ${path.join(repoPath, '.codegraph', 'store')} — see this task's <precondition>`
		);
	}

	/** @type {import('node:child_process').ChildProcess | undefined} */
	let child;
	/** @type {import('@playwright/test').Browser | undefined} */
	let browser;

	try {
		try {
			const started = await startCodegraphUi(binaryPath, repoPath);
			child = started.child;
			const baseUrl = started.url;

			// The oracle's data source: fetch FileSymbols over HTTP directly
			// — never via the SPA's own client code — so the expectation
			// this script asserts against is computed independently of
			// anything under test.
			const fsResponse = await fetch(`${baseUrl}/codegraph.ui.v1.UIService/FileSymbols`, {
				method: 'POST',
				headers: { 'content-type': 'application/json' },
				body: JSON.stringify({ path: file })
			});
			if (!fsResponse.ok) {
				throw new Error(`breadcrumb-check: FileSymbols rpc failed: HTTP ${fsResponse.status}`);
			}
			/** @type {{symbols?: OracleSymbol[]}} */
			const fsBody = await fsResponse.json();
			const symbols = fsBody.symbols ?? [];
			record.symbolCount = symbols.length;
			if (symbols.length < 2) {
				throw new Error(
					`breadcrumb-check: FileSymbols returned ${symbols.length} symbols for ${file} — need at least 2 (ranges AND a gap)`
				);
			}
			console.log(`breadcrumb-check: oracle has ${symbols.length} symbols for ${file}`);

			browser = await chromium.launch();
			record.browserIdentity = { name: 'chromium', version: browser.version() };
			const context = await browser.newContext({ viewport: { width: 1600, height: 1000 } });
			const page = await context.newPage();
			/** @type {string[]} */
			const pageErrors = [];
			page.on('pageerror', (err) => pageErrors.push(err.message));

			const targetUrl = `${baseUrl}/browse?file=${encodeURIComponent(file)}`;
			await page.goto(targetUrl, { waitUntil: 'load', timeout: 30000 });

			// This is the RED-path timeout against a pre-fix binary: the
			// breadcrumb element never appears, breadcrumbPresent stays
			// false, and the error thrown here propagates to the outer
			// catch — record is still written by the finally block below.
			await pollUntil(
				async () => (await page.locator('[data-testid="source-breadcrumb"]').count()) > 0,
				15000,
				250
			);
			record.breadcrumbPresent = true;
			console.log('breadcrumb-check: source-breadcrumb element present');

			const bar = page.locator('[data-testid="source-breadcrumb"]');
			const symbolEl = page.locator('[data-testid="source-breadcrumb-symbol"]');

			/** @returns {Promise<{lineAttr: string | null, line: number | null, empty: boolean, shown: string | null}>} */
			async function readOnce() {
				const lineAttr = await bar.getAttribute('data-current-line');
				const line = lineAttr === null ? null : Number(lineAttr);
				const empty = (await symbolEl.getAttribute('data-empty')) === 'true';
				const shown = empty ? null : ((await symbolEl.textContent()) ?? '').trim();
				return { lineAttr, line, empty, shown };
			}

			// readStableObservation: `data-current-line` and the crumb-
			// derived `data-empty`/button text are TWO SEPARATE template
			// bindings sharing the `currentLine` dependency, committed
			// within the same Svelte flush but that flush is scheduled
			// inside a requestAnimationFrame callback (the scroll-tracking
			// effect's own throttle) — a fixed-frame-count wait was
			// confirmed empirically to still sometimes catch this mid-
			// commit (data-current-line already updated, data-empty not
			// yet). Poll-until-STABLE instead of poll-until-changed: read
			// the full tuple repeatedly until TWO CONSECUTIVE reads agree,
			// which by construction cannot land on a torn intermediate
			// state (a torn read is definitionally unstable — it changes
			// again on the very next poll once the second binding lands).
			async function readStableObservation() {
				let last = JSON.stringify(await readOnce());
				const deadlineAt = Date.now() + 2000;
				for (;;) {
					await new Promise((resolve) => setTimeout(resolve, 60));
					const current = JSON.stringify(await readOnce());
					if (current === last) return JSON.parse(current);
					last = current;
					if (Date.now() >= deadlineAt) return JSON.parse(current);
				}
			}

			const observations = [];
			let nonEmptyObservations = 0;
			let emptyObservations = 0;
			const MAX_WHEEL_STEPS = 60;

			for (let step = 0; step < MAX_WHEEL_STEPS; step += 1) {
				const obs = await readStableObservation();
				if (obs.line === null) {
					throw new Error('breadcrumb-check: source-breadcrumb has no data-current-line attribute');
				}
				const expected = oracleInnermost(symbols, obs.line);
				const expectedName = expected ? expected.name : null;
				const agrees = obs.empty ? expectedName === null : obs.shown === expectedName;
				const observation = { line: obs.line, shown: obs.shown, expected: expectedName, empty: obs.empty, agrees };
				observations.push(observation);
				console.log(`breadcrumb-check: observation ${JSON.stringify(observation)}`);
				if (expectedName !== null) nonEmptyObservations += 1;
				else emptyObservations += 1;

				if (observations.length >= 3 && nonEmptyObservations >= 1 && emptyObservations >= 1) {
					break;
				}

				// Real wheel input — never a dispatched synthetic event.
				const beforeScrollY = await page.evaluate(() => window.scrollY);
				await page.mouse.wheel(0, 240);
				try {
					await pollUntil(
						async () => {
							const afterLineAttr = await bar.getAttribute('data-current-line');
							const afterScrollY = await page.evaluate(() => window.scrollY);
							return afterLineAttr !== obs.lineAttr || afterScrollY === beforeScrollY;
						},
						2000,
						50
					);
				} catch {
					// The page may already be at max scroll with an unchanged
					// current line — that is a legitimate end-of-file state,
					// not a failure; the loop's own line-diversity condition
					// (>=1 empty AND >=1 non-empty) or the step ceiling above
					// governs whether the run is deemed complete.
				}
			}

			record.observations = observations;
			record.nonEmptyObservations = nonEmptyObservations;
			record.emptyObservations = emptyObservations;
			record.pageErrorCount = pageErrors.length;
			record.pageErrors = pageErrors;

			record.success =
				record.breadcrumbPresent === true &&
				observations.length >= 3 &&
				observations.every((o) => o.agrees) &&
				nonEmptyObservations >= 1 &&
				emptyObservations >= 1 &&
				pageErrors.length === 0;

			await context.close();
		} catch (err) {
			record.error = err instanceof Error ? err.message : String(err);
			if (err instanceof Error && err.stack) record.errorStack = err.stack;
		}
	} finally {
		if (browser) {
			await browser.close().catch(() => {});
		}
		await stopChild(child);

		fs.mkdirSync(path.dirname(outPath), { recursive: true });
		fs.writeFileSync(outPath, JSON.stringify(record, null, 2) + '\n');
		console.log(`breadcrumb-check: wrote ${outPath} — success=${record.success}`);
	}

	if (!record.success) {
		throw new Error(
			`breadcrumb-check: FAILED — breadcrumbPresent=${record.breadcrumbPresent} error=${record.error}`
		);
	}
}

const isMain = process.argv[1] && url.pathToFileURL(process.argv[1]).href === import.meta.url;
if (isMain) {
	main().catch((err) => {
		console.error(`breadcrumb-check: fatal — ${err instanceof Error ? err.message : String(err)}`);
		if (err instanceof Error && err.stack) console.error(err.stack);
		process.exit(1);
	});
}
