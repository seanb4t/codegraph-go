#!/usr/bin/env node
// graph-console-check.mjs — 01-01 Task 1 (FIX-04/FIX-05, D-09): the permanent
// live-Chromium `/graph` console gate. Boots the real `codegraph ui` binary,
// loads `/graph`, and asserts D-08's bar — zero uncaught page errors AND zero
// `console.warn`/`console.error` originating from our code — on every pinned
// corpus. This is the FIRST script in web/scripts/ to register
// `page.on('console', ...)` (01-PATTERNS.md: no sibling script needs this
// half); `page.on('pageerror', ...)` follows the same convention every
// sibling already uses.
//
// This task wires the `self` corpus only (this repository's own index).
// Task 2 (01-01) extends the run loop with the pinned `guava` corpus and the
// invalid-endpoints overlap diagnostic that plan 01-09's FIX-05 fix consumes.
//
// The script is ALSO this gate's own RED instrument: run against the
// CURRENT, unmodified embedded build, it must exit non-zero and its
// committed verdict at corpora/graph-console-check.json must record the
// `notify` TypeError (pageerror) and the two `text-valign: right` style
// warnings (console.warn) — the pre-fix evidence plans 01-04/01-08 close
// against. `--self-test` is the positive control every real run depends on:
// a capture harness that cannot see a PLANTED console.warn, console.error
// and uncaught pageerror can never be trusted to see a REAL one (rule
// 84d1gfpywd).
import { chromium } from '@playwright/test';
import { spawn } from 'node:child_process';
import { execFileSync } from 'node:child_process';
import * as fs from 'node:fs';
import * as path from 'node:path';
import * as url from 'node:url';

/**
 * @typedef {Object} ConsoleEntry
 * @property {string} type - 'warning' | 'error'
 * @property {string} text
 * @property {{url?: string, lineNumber?: number, columnNumber?: number}} location
 */

/** @returns {string} */
function repoRoot() {
	const here = path.dirname(url.fileURLToPath(import.meta.url));
	return path.resolve(here, '..', '..');
}

/**
 * pollUntil — poll-don't-sleep, the same convention every sibling
 * web/scripts/*-check.mjs establishes.
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
 * startCodegraphUi — the same shape as breadcrumb-check.mjs's function of
 * the same name: spawn `<binary> ui --no-open --path <repo>`, resolve once
 * the printed URL appears in stdout, reject on early exit.
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
				reject(new Error('graph-console-check: codegraph ui printed no URL within 15s'));
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
				reject(new Error(`graph-console-check: codegraph ui exited early (code ${code})`));
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

// allowlist — EMPTY as shipped (D-08's bar is zero; pre-populating this to
// make the current build pass would convert the gate into a vacuous one —
// see this task's own threat register T-01-01-01). A future entry is
// `{ pattern: '<regex source>', reason: '<why this specific text is
// tolerated>' }`; the pass/fail verdict below is computed against the
// non-allowlisted remainder only.
/** @type {Array<{pattern: string, reason: string}>} */
const ALLOWLIST = [];

/** @param {string} text @returns {boolean} */
function isAllowlisted(text) {
	return ALLOWLIST.some(({ pattern }) => new RegExp(pattern).test(text));
}

/**
 * selfRepoIdentity — best-effort repo/sha identity for the `self` corpus
 * run. Unlike the guava pin (D-09's reproducibility requirement, enforced
 * by Task 2's verify gate), this repo's own identity is informational only
 * — a git failure here (e.g. a source tarball with no .git) degrades to a
 * directory-name/`"unknown"` fallback rather than failing the whole check.
 * @param {string} repoPath
 * @returns {{repo: string, sha: string}}
 */
function selfRepoIdentity(repoPath) {
	let sha = 'unknown';
	try {
		sha = execFileSync('git', ['rev-parse', 'HEAD'], {
			cwd: repoPath,
			stdio: ['ignore', 'pipe', 'ignore']
		})
			.toString()
			.trim();
	} catch {
		// No .git available — leave 'unknown'.
	}
	let repo = path.basename(repoPath);
	try {
		const remote = execFileSync('git', ['config', '--get', 'remote.origin.url'], {
			cwd: repoPath,
			stdio: ['ignore', 'pipe', 'ignore']
		})
			.toString()
			.trim();
		const m = remote.match(/[:/]([^/]+\/[^/]+?)(?:\.git)?$/);
		if (m) repo = m[1];
	} catch {
		// No remote configured — leave the directory-name fallback.
	}
	return { repo, sha };
}

/**
 * @param {string[]} argv
 */
function parseArgs(argv) {
	const out = {
		selfTest: false,
		corpus: 'self,guava',
		binary: path.join(repoRoot(), 'codegraph'),
		out: path.join(repoRoot(), 'corpora', 'graph-console-check.json')
	};
	for (let i = 0; i < argv.length; i++) {
		const arg = argv[i];
		if (arg === '--self-test') {
			out.selfTest = true;
		} else if (arg === '--corpus') {
			out.corpus = argv[++i];
		} else if (arg === '--binary') {
			out.binary = path.resolve(argv[++i]);
		} else if (arg === '--out') {
			out.out = path.resolve(argv[++i]);
		} else {
			throw new Error(`graph-console-check: unrecognized argument "${arg}"`);
		}
	}
	return out;
}

/**
 * runSelfTest — the positive control. Navigates to about:blank (no binary
 * boot needed), registers BOTH capture halves, plants one console.warn, one
 * console.error and one uncaught pageerror (via a setTimeout throw, so it
 * fires asynchronously and independently of the evaluate() call's own
 * return), then asserts all three were recaptured. Prints the three
 * recaptured counts unconditionally — a self-test that prints nothing
 * cannot distinguish "captured" from "ran nothing" (rule 84d1gfpywd).
 * @returns {Promise<boolean>}
 */
async function runSelfTest() {
	const browser = await chromium.launch();
	try {
		const page = await browser.newPage();
		/** @type {string[]} */
		const warnings = [];
		/** @type {string[]} */
		const errors = [];
		/** @type {string[]} */
		const pageErrors = [];
		// Registered BEFORE any navigation/evaluate, matching every sibling
		// script's own pageerror discipline.
		page.on('pageerror', (err) => pageErrors.push(err.message));
		page.on('console', (msg) => {
			if (msg.type() === 'warning') warnings.push(msg.text());
			else if (msg.type() === 'error') errors.push(msg.text());
		});

		await page.goto('about:blank');
		await page.evaluate(() => {
			console.warn('graph-console-check --self-test: planted console.warn');
			console.error('graph-console-check --self-test: planted console.error');
			setTimeout(() => {
				throw new Error('graph-console-check --self-test: planted uncaught pageerror');
			}, 0);
		});

		try {
			await pollUntil(
				async () => warnings.length > 0 && errors.length > 0 && pageErrors.length > 0,
				5000,
				50
			);
		} catch {
			// Fall through — the counts below (likely still short) are what
			// gets reported and judged.
		}

		console.log(
			`graph-console-check --self-test: recaptured console.warn=${warnings.length} console.error=${errors.length} pageerror=${pageErrors.length}`
		);

		const ok = warnings.length > 0 && errors.length > 0 && pageErrors.length > 0;
		if (!ok) {
			console.error(
				'graph-console-check --self-test: FAIL — the capture harness did not recapture every planted entry'
			);
		} else {
			console.log('graph-console-check --self-test: PASS');
		}
		return ok;
	} finally {
		await browser.close().catch(() => {});
	}
}

/**
 * waitForGraphSettled — poll for the geometry/metrics seam GraphCanvas.svelte
 * publishes after the first layout settle (non-zero rendered node count),
 * then a short quiescence window so a late-resolving ELK promise's console
 * output (the exact FIX-04/FIX-05 mechanism, 01-RESEARCH.md Investigations
 * 1-2) is not missed.
 * @param {import('@playwright/test').Page} page
 * @param {number} settleTimeoutMs
 * @param {number} quiescenceMs
 */
async function waitForGraphSettled(page, settleTimeoutMs, quiescenceMs) {
	await pollUntil(
		() =>
			page.evaluate(() => {
				// eslint-disable-next-line @typescript-eslint/no-explicit-any
				const metrics = /** @type {any} */ (window).__codegraphFileGraphMetrics;
				// eslint-disable-next-line @typescript-eslint/no-explicit-any
				const geometry = /** @type {any} */ (window).__codegraphFileGraphGeometry;
				return Boolean(metrics) && metrics.nodeCount > 0 && Array.isArray(geometry) && geometry.length > 0;
			}),
		settleTimeoutMs,
		100
	);
	await page.waitForTimeout(quiescenceMs);
}

/**
 * runCorpusCheck — boots the real binary against `repoPath`, loads `/graph`
 * in a fresh page on the SHARED `browser`, and returns this run's verdict
 * record. Every exit path (success or thrown error) tears the child process
 * down via `stopChild` in a finally block.
 * @param {import('@playwright/test').Browser} browser
 * @param {{corpus: string, repoPath: string, repo: string, sha: string, binaryPath: string}} args
 */
async function runCorpusCheck(browser, { corpus, repoPath, repo, sha, binaryPath }) {
	if (!fs.existsSync(binaryPath)) {
		throw new Error(`graph-console-check: binary not found at ${binaryPath} — run \`task build:release\` first`);
	}
	if (!fs.existsSync(path.join(repoPath, '.codegraph', 'store'))) {
		throw new Error(
			`graph-console-check: no index at ${path.join(repoPath, '.codegraph', 'store')} — see this task's <precondition>`
		);
	}

	/** @type {import('node:child_process').ChildProcess | undefined} */
	let child;
	try {
		const started = await startCodegraphUi(binaryPath, repoPath);
		child = started.child;
		const graphUrl = `${started.url}/graph`;

		const context = await browser.newContext();
		const page = await context.newPage();

		/** @type {string[]} */
		const pageErrors = [];
		/** @type {ConsoleEntry[]} */
		const consoleMessages = [];
		// Both registered BEFORE page.goto — the capture contract this
		// task's own must_haves require.
		page.on('pageerror', (err) => pageErrors.push(err.message));
		page.on('console', (msg) => {
			const type = msg.type();
			if (type === 'warning' || type === 'error') {
				consoleMessages.push({ type, text: msg.text(), location: msg.location() });
			}
		});

		await page.goto(graphUrl, { waitUntil: 'load', timeout: 30000 });
		await waitForGraphSettled(page, 20000, 3000);

		const consoleWarnCount = consoleMessages.filter((m) => m.type === 'warning').length;
		const consoleErrorCount = consoleMessages.filter((m) => m.type === 'error').length;
		const allowlistedCount =
			pageErrors.filter((e) => isAllowlisted(e)).length +
			consoleMessages.filter((m) => isAllowlisted(m.text)).length;

		await context.close();

		return {
			corpus,
			repo,
			sha,
			url: graphUrl,
			pageErrorCount: pageErrors.length,
			pageErrors,
			consoleWarnCount,
			consoleErrorCount,
			consoleMessages,
			allowlistedCount
		};
	} finally {
		await stopChild(child);
	}
}

/**
 * runIsClean — true iff every non-allowlisted page-error, warn and error
 * count is zero for this run.
 * @param {ReturnType<typeof runCorpusCheck> extends Promise<infer T> ? T : never} run
 */
function runIsClean(run) {
	const nonAllowlistedPageErrors = run.pageErrors.filter((e) => !isAllowlisted(e)).length;
	const nonAllowlistedWarn = run.consoleMessages.filter((m) => m.type === 'warning' && !isAllowlisted(m.text)).length;
	const nonAllowlistedError = run.consoleMessages.filter((m) => m.type === 'error' && !isAllowlisted(m.text)).length;
	return nonAllowlistedPageErrors === 0 && nonAllowlistedWarn === 0 && nonAllowlistedError === 0;
}

async function main() {
	const args = parseArgs(process.argv.slice(2));

	if (args.selfTest) {
		const ok = await runSelfTest();
		process.exit(ok ? 0 : 1);
		return;
	}

	const requestedCorpora = args.corpus
		.split(',')
		.map((s) => s.trim())
		.filter(Boolean);

	/** @type {Array<Awaited<ReturnType<typeof runCorpusCheck>>>} */
	const runs = [];
	/** @type {string | null} */
	let fatalError = null;
	/** @type {string | undefined} */
	let browserVersion;

	const browser = await chromium.launch();
	try {
		browserVersion = browser.version();
		for (const corpusName of requestedCorpora) {
			if (corpusName === 'self') {
				const repoPath = repoRoot();
				const { repo, sha } = selfRepoIdentity(repoPath);
				const run = await runCorpusCheck(browser, { corpus: 'self', repoPath, repo, sha, binaryPath: args.binary });
				runs.push(run);
				console.log(
					`graph-console-check: corpus=self pageErrors=${run.pageErrorCount} consoleWarn=${run.consoleWarnCount} consoleError=${run.consoleErrorCount}`
				);
			} else if (corpusName === 'guava') {
				// Task 2 (01-01) implements this corpus and its overlap
				// diagnostic. Not yet wired — fail loudly rather than
				// silently skipping (rule 84d1gfpywd).
				throw new Error(
					'graph-console-check: --corpus guava is not yet implemented (see plan 01-01 Task 2)'
				);
			} else {
				throw new Error(`graph-console-check: unknown --corpus value "${corpusName}"`);
			}
		}
	} catch (err) {
		fatalError = err instanceof Error ? err.message : String(err);
	} finally {
		await browser.close().catch(() => {});
	}

	const allRunsPresent = fatalError === null && runs.length === requestedCorpora.length;
	const success = allRunsPresent && runs.every((r) => runIsClean(r));

	const record = {
		schemaVersion: 1,
		success,
		generatedAt: new Date().toISOString(),
		browserIdentity: { name: 'chromium', version: browserVersion ?? null },
		runs,
		error: fatalError
	};

	await fs.promises.mkdir(path.dirname(args.out), { recursive: true });
	fs.writeFileSync(args.out, JSON.stringify(record, null, 2) + '\n');
	console.log(`graph-console-check: wrote ${args.out} — success=${success}`);

	if (!success) {
		process.exit(1);
	}
}

const isMain = process.argv[1] && url.pathToFileURL(process.argv[1]).href === import.meta.url;
if (isMain) {
	main().catch((err) => {
		console.error(`graph-console-check: fatal — ${err instanceof Error ? err.message : String(err)}`);
		if (err instanceof Error && err.stack) console.error(err.stack);
		process.exit(1);
	});
}
