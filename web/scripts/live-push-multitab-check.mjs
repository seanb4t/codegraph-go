#!/usr/bin/env node
// live-push-multitab-check.mjs — 06-06 Task 2: the browser half of
// criterion 2 (per-message TIMING, paired with a per-tab received-
// generation superset check so the timing bound cannot hold vacuously
// over a single receipt) and criterion 3 (fan-out isolation and a
// jittered reconnect against a STABLE origin). Follows
// graph-collapse-affordance-check.mjs / graph-live-update-check.mjs's
// established shape: repoRoot(), real trusted input only where input is
// needed, a diagnostic JSON record written on EVERY run including
// failure, try/finally so cleanup runs on every exit path.
//
// Unlike those two scripts, this one never touches the graph canvas —
// every tab here is opened on /health, a route that (via +layout.svelte)
// unconditionally constructs the live client and store the instant the
// page mounts, with no user gesture required to "subscribe".
//
// This script points `codegraph ui` at THIS repository's own already-
// indexed .codegraph/ store (the plan's own precondition: "the corpus
// scale is 06-05's concern, not this one"). Real re-indexes are produced
// by creating tiny, disposable Go source files under a throwaway
// package directory (internal/livepushprobe/) and running a real
// `codegraph sync` — mirroring graph-live-update-check.mjs's own
// disposable-probe-file technique, adapted to Go source (this repo's
// own language) instead of the guava corpus's Java. Every probe file is
// removed, and one final `codegraph sync` is run, in the finally block
// — `corpusCleanAtExit` (checked against `git status --porcelain` for
// exactly the probe path) is asserted the same way
// graph-live-update-check.mjs asserts it for its own corpus checkout.
//
// The client's own backoff-shape constants (LIVE_BACKOFF_BASE_MS,
// LIVE_BACKOFF_JITTER_SPREAD) are read directly out of
// web/src/lib/live/live-client.ts by a small regex extraction rather
// than hard-coded here — the same "mirror the Go/TS source in JS rather
// than duplicate a number that can drift" discipline
// graph-live-update-check.mjs already established for corpusDir().
//
// [RED-path reproduction, per this task's own instruction to prove the
// record is written on failure too]: pass --red-skip-repoint to run the
// RECONNECT scenario with the proxy's upstream deliberately left
// pointed at the OLD (now-dead) codegraph ui process after the restart.
// This is a one-off diagnostic invocation, never used for the committed
// record — see 06-06-SUMMARY.md for the observed output.
import { chromium } from '@playwright/test';
import { spawn, execFileSync } from 'node:child_process';
import * as fs from 'node:fs';
import * as path from 'node:path';
import * as url from 'node:url';
import { startProxy } from './live-push-stable-proxy.mjs';

/** @returns {string} */
function repoRoot() {
	const here = path.dirname(url.fileURLToPath(import.meta.url));
	return path.resolve(here, '..', '..');
}

/**
 * extractLiveClientConstants mirrors live-client.ts's own pinned
 * backoff-shape constants by reading the TS source directly (this is a
 * plain Node script — it has no TS loader to `import` the module) rather
 * than hard-coding LIVE_BACKOFF_BASE_MS/LIVE_BACKOFF_JITTER_SPREAD as
 * bare numbers that could silently drift from the real implementation.
 * @param {string} root
 * @returns {{baseDelayMs: number, jitterSpread: number}}
 */
function extractLiveClientConstants(root) {
	const srcPath = path.join(root, 'web', 'src', 'lib', 'live', 'live-client.ts');
	const src = fs.readFileSync(srcPath, 'utf8');
	const baseMatch = src.match(/LIVE_BACKOFF_BASE_MS\s*=\s*([\d_]+)/);
	const jitterMatch = src.match(/LIVE_BACKOFF_JITTER_SPREAD\s*=\s*([\d.]+)/);
	if (!baseMatch || !jitterMatch) {
		throw new Error(
			`live-push-multitab-check: could not extract LIVE_BACKOFF_BASE_MS/LIVE_BACKOFF_JITTER_SPREAD from ${srcPath}`
		);
	}
	return {
		baseDelayMs: Number(baseMatch[1].replace(/_/g, '')),
		jitterSpread: Number(jitterMatch[1])
	};
}

/**
 * A generic poller with no Playwright page dependency (unlike the
 * established pollUntil in the other check scripts) — this file polls
 * plain async predicates (page.evaluate calls, filesystem state) rather
 * than page-bound conditions exclusively, and needs to poll several
 * pages independently in the same scenario.
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
			throw new Error(`live-push-multitab-check: pollUntil: condition did not become true within ${timeoutMs}ms`);
		}
		await new Promise((r) => setTimeout(r, pollMs));
	}
}

function sleep(ms) {
	return new Promise((r) => setTimeout(r, ms));
}

/**
 * @param {import('@playwright/test').Page} page
 * @returns {Promise<Window['__codegraphLiveObservations'] | null>}
 */
async function readObservations(page) {
	return page.evaluate(() => /** @type {any} */ (window).__codegraphLiveObservations ?? null);
}

/**
 * @param {string} binaryPath
 * @param {string} repoPath
 * @param {Record<string,string>} extraEnv
 * @returns {Promise<{child: import('node:child_process').ChildProcess, url: string}>}
 */
function startCodegraphUi(binaryPath, repoPath, extraEnv) {
	return new Promise((resolve, reject) => {
		const child = spawn(binaryPath, ['ui', '--no-open', '--path', repoPath], {
			stdio: ['ignore', 'pipe', 'pipe'],
			env: { ...process.env, ...extraEnv }
		});
		let stdout = '';
		let settled = false;
		const timer = setTimeout(() => {
			if (!settled) {
				settled = true;
				reject(new Error('live-push-multitab-check: codegraph ui printed no URL within 15s'));
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
				reject(new Error(`live-push-multitab-check: codegraph ui exited early (code ${code})`));
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
		child.kill('SIGKILL'); // a real crash/restart, not a graceful shutdown — the
		// RECONNECT scenario is about a process that is simply gone, not one
		// that closes its streams politely first.
		setTimeout(() => {
			if (child.exitCode === null && child.signalCode === null) child.kill('SIGKILL');
		}, 3000);
	});
}

/**
 * @typedef {{generation:number, epoch:number, seeded:boolean, receivedAtMs:number, appliedAtMs:number|null}} ObsEvent
 * @typedef {{epoch:number, attempt:number, baseDelayMs:number|null, scheduledDelayMs:number|null, requestedSinceGeneration:number, openedAtMs:number|null}} ObsConnection
 */

let probeCounter = 0;
/**
 * makeProbe creates one small, disposable, syntactically valid Go source
 * file under a throwaway package directory and returns its repo-relative
 * path. A brand new file (never a modification of an existing tracked
 * file) is the least invasive way to produce a REAL, Discover()-visible
 * change against this repository's own live .codegraph/ index —
 * mirroring graph-live-update-check.mjs's own disposable-Java-file
 * technique for the guava corpus, adapted to this repo's own language.
 * @param {string} root
 * @returns {string} the repo-relative path of the created file
 */
function makeProbe(root) {
	probeCounter += 1;
	const relPath = path.join('internal', 'livepushprobe', `probe${probeCounter}.go`);
	const absPath = path.join(root, relPath);
	fs.mkdirSync(path.dirname(absPath), { recursive: true });
	fs.writeFileSync(
		absPath,
		`package livepushprobe\n\n// Transient probe file for 06-06's live-push multitab check — removed\n// before this script exits. Exists solely to give \`codegraph sync\` a\n// real, Discover()-visible change to index.\nfunc Probe${probeCounter}() {}\n`
	);
	return relPath;
}

/**
 * runSync runs a real, synchronous `codegraph sync` against root — the
 * same real re-index every one of this plan's scenarios depends on.
 * @param {string} binaryPath
 * @param {string} root
 */
function runSync(binaryPath, root) {
	execFileSync(binaryPath, ['sync', root, '-q'], { stdio: 'pipe' });
}

async function main() {
	const args = process.argv.slice(2);
	const outIdx = args.indexOf('--out');
	const outPath =
		outIdx === -1
			? path.join(repoRoot(), 'corpora', 'live-push-multitab-check.json')
			: path.resolve(args[outIdx + 1]);
	// RED-path reproduction only — see this file's header comment. Never
	// used for the committed record.
	const redSkipRepoint = args.includes('--red-skip-repoint');

	const root = repoRoot();
	if (!fs.existsSync(path.join(root, '.codegraph', 'store'))) {
		throw new Error(`live-push-multitab-check: no .codegraph/store found at ${root} — see this task's <precondition>`);
	}
	const binaryPath = path.join(root, 'codegraph');
	if (!fs.existsSync(binaryPath)) {
		throw new Error(`live-push-multitab-check: codegraph binary not found at ${binaryPath} — run \`task build\` first`);
	}

	const { baseDelayMs: clientBaseDelayMs, jitterSpread } = extractLiveClientConstants(root);
	const minDowntimeMs = 4 * clientBaseDelayMs; // spans at least two scheduled
	// backoff intervals — see this plan's own derivation of why 1x is not
	// enough (a tab that reconnects on its first attempt is CORRECT
	// behaviour and must not be misread as a storm or a failure).
	const TAB_COUNT = 3;
	const TRIGGER_SPACING_MS = 600; // comfortably above the 100ms debounce
	// window this run configures below, so three real re-indexes never
	// coalesce into fewer than three generation events.
	const UI_DEBOUNCE_MS = '100';

	/** @type {Record<string, any>} */
	const record = {
		schemaVersion: 1,
		tabCount: TAB_COUNT,
		triggerSpacingMs: TRIGGER_SPACING_MS,
		clientBaseDelayMs,
		jitterSpread,
		minDowntimeMs,
		reconnectDelaysProvenance: 'client-scheduled',
		coalescingProvenBy:
			'internal/uiserver/livehandler_test.go#TestWatchGraphHandlerCoalescesForANonReadingClient',
		redSkipRepoint,
		seedGenerationPerTab: [],
		triggeredGenerations: [],
		receivedGenerationsPerTab: [],
		minInterArrivalMs: null,
		isolationTriggeredGenerations: [],
		blockedTabIndex: null,
		blockedTabBlockedMs: null,
		blockedTabReceiptsDuringBlock: null,
		healthyTabsIsolationSupersetOK: null,
		healthyTabsIsolationMinInterArrivalMs: null,
		slowTabFinalGeneration: null,
		newestGeneration: null,
		reconnectAttemptsPerTab: [],
		reconnectDelaysPerTab: [],
		reconnectBaseDelaysPerTab: [],
		resumeCursorSentPerTab: [],
		postReconnectAppliedPerTab: [],
		pageErrorCount: null,
		pageErrors: [],
		browserIdentity: null,
		corpusCleanAtExit: false,
		error: null,
		success: false
	};

	/** @type {import('node:child_process').ChildProcess | undefined} */
	let child;
	/** @type {import('node:child_process').ChildProcess | undefined} */
	let secondChild;
	/** @type {import('@playwright/test').Browser | undefined} */
	let browser;
	/** @type {Awaited<ReturnType<typeof startProxy>> | undefined} */
	let proxy;
	/** @type {string[]} */
	const createdProbePaths = [];

	try {
		try {
			proxy = await startProxy({});

			const started = await startCodegraphUi(binaryPath, root, { CODEGRAPH_DEBOUNCE_MS: UI_DEBOUNCE_MS });
			child = started.child;
			proxy.setUpstream(started.url);

			browser = await chromium.launch();
			record.browserIdentity = { name: 'chromium', version: browser.version() };
			const context = await browser.newContext({ viewport: { width: 1280, height: 800 } });

			/** @type {string[]} */
			const pageErrors = [];
			/** @type {import('@playwright/test').Page[]} */
			const pages = [];
			for (let i = 0; i < TAB_COUNT; i++) {
				const page = await context.newPage();
				page.on('pageerror', (err) => pageErrors.push(`tab${i}: ${err.message}`));
				await page.goto(`${proxy.url}/health`, { waitUntil: 'load', timeout: 30000 });
				pages.push(page);
			}

			// --- Seed: every tab's own first event, before any triggered
			// re-index ---
			const seedObs = await Promise.all(
				pages.map((p) =>
					pollUntil(
						async () => {
							const obs = await readObservations(p);
							return obs && obs.events.length >= 1 ? obs : null;
						},
						15000,
						100
					)
				)
			);
			record.seedGenerationPerTab = seedObs.map((obs) => {
				const seed = obs.events.find((/** @type {ObsEvent} */ e) => e.seeded);
				return seed ? seed.generation : null;
			});
			let lastKnownGen = Math.max(...record.seedGenerationPerTab);

			// --- FAN-OUT AND TIMING: 3 real re-indexes, each triggered only
			// after every tab has received the PREVIOUS one, spaced by
			// TRIGGER_SPACING_MS so a silently-buffered proxy or handler
			// cannot pass by delivering everything at once at the end. ---
			const TRIGGER_COUNT = 3;
			const triggeredGenerations = [];
			for (let k = 0; k < TRIGGER_COUNT; k++) {
				const relPath = makeProbe(root);
				createdProbePaths.push(relPath);
				runSync(binaryPath, root);

				const observedGen = await pollUntil(
					async () => {
						const obsAll = await Promise.all(pages.map((p) => readObservations(p)));
						const gens = obsAll.map((obs) => {
							const hit = obs.events.find(
								(/** @type {ObsEvent} */ e) => !e.seeded && e.generation > lastKnownGen
							);
							return hit ? hit.generation : null;
						});
						if (gens.every((g) => g !== null) && new Set(gens).size === 1) return gens[0];
						return null;
					},
					15000,
					100
				);
				triggeredGenerations.push(observedGen);
				lastKnownGen = observedGen;

				if (k < TRIGGER_COUNT - 1) await sleep(TRIGGER_SPACING_MS);
			}
			record.triggeredGenerations = triggeredGenerations;

			const postTriggerObs = await Promise.all(pages.map((p) => readObservations(p)));
			record.receivedGenerationsPerTab = postTriggerObs.map((obs) =>
				obs.events.filter((/** @type {ObsEvent} */ e) => !e.seeded).map((/** @type {ObsEvent} */ e) => e.generation)
			);

			let minInterArrival = Infinity;
			for (const obs of postTriggerObs) {
				const receivedAtByGen = new Map(
					obs.events
						.filter((/** @type {ObsEvent} */ e) => !e.seeded)
						.map((/** @type {ObsEvent} */ e) => [e.generation, e.receivedAtMs])
				);
				for (let i = 1; i < triggeredGenerations.length; i++) {
					const prev = receivedAtByGen.get(triggeredGenerations[i - 1]);
					const cur = receivedAtByGen.get(triggeredGenerations[i]);
					if (prev !== undefined && cur !== undefined) {
						minInterArrival = Math.min(minInterArrival, cur - prev);
					}
				}
			}
			record.minInterArrivalMs = Number.isFinite(minInterArrival) ? minInterArrival : null;

			// --- FAN-OUT ISOLATION: block one tab's main thread with a real
			// synchronous busy loop (never faked with a resolved promise —
			// the whole point is that this tab's own JS cannot run) while
			// the other two keep receiving real triggered re-indexes. ---
			const BLOCKED_IDX = TAB_COUNT - 1;
			const healthyIdx = pages.map((_, i) => i).filter((i) => i !== BLOCKED_IDX);
			record.blockedTabIndex = BLOCKED_IDX;

			const ISOLATION_TRIGGER_COUNT = 2;
			const blockedMs = TRIGGER_SPACING_MS * (ISOLATION_TRIGGER_COUNT + 1);
			const blockPromise = pages[BLOCKED_IDX].evaluate((ms) => {
				const start = performance.now();
				const end = start + ms;
				// eslint-disable-next-line no-empty
				while (performance.now() < end) {
					/* deliberately blocking the main thread */
				}
				return { start, end: performance.now() };
			}, blockedMs);

			const isolationTriggeredGenerations = [];
			for (let k = 0; k < ISOLATION_TRIGGER_COUNT; k++) {
				const relPath = makeProbe(root);
				createdProbePaths.push(relPath);
				runSync(binaryPath, root);

				const observedGen = await pollUntil(
					async () => {
						const obsAll = await Promise.all(healthyIdx.map((i) => readObservations(pages[i])));
						const gens = obsAll.map((obs) => {
							const hit = obs.events.find(
								(/** @type {ObsEvent} */ e) => !e.seeded && e.generation > lastKnownGen
							);
							return hit ? hit.generation : null;
						});
						if (gens.every((g) => g !== null) && new Set(gens).size === 1) return gens[0];
						return null;
					},
					15000,
					100
				);
				isolationTriggeredGenerations.push(observedGen);
				lastKnownGen = observedGen;

				if (k < ISOLATION_TRIGGER_COUNT - 1) await sleep(TRIGGER_SPACING_MS);
			}
			record.isolationTriggeredGenerations = isolationTriggeredGenerations;

			const blockResult = await blockPromise;
			record.blockedTabBlockedMs = blockResult.end - blockResult.start;

			const blockedObsDuring = await readObservations(pages[BLOCKED_IDX]);
			record.blockedTabReceiptsDuringBlock = blockedObsDuring.events.filter(
				(/** @type {ObsEvent} */ e) => e.receivedAtMs >= blockResult.start && e.receivedAtMs <= blockResult.end
			).length;

			// Healthy tabs: every isolation-phase triggered generation must
			// have arrived, with inter-arrival still bounded by the trigger
			// spacing — recorded for audit even though the plan's own
			// machine-verify command (fixed, see the plan's <verify>) reads
			// only the FAN-OUT/TIMING scenario's fields above; this is this
			// task's own acceptance-criteria obligation, checked directly.
			const healthyObsPostIsolation = await Promise.all(healthyIdx.map((i) => readObservations(pages[i])));
			record.healthyTabsIsolationSupersetOK = healthyObsPostIsolation.every((obs) => {
				const received = new Set(
					obs.events.filter((/** @type {ObsEvent} */ e) => !e.seeded).map((/** @type {ObsEvent} */ e) => e.generation)
				);
				return isolationTriggeredGenerations.every((g) => received.has(g));
			});
			let healthyMinInterArrival = Infinity;
			for (const obs of healthyObsPostIsolation) {
				const receivedAtByGen = new Map(
					obs.events
						.filter((/** @type {ObsEvent} */ e) => !e.seeded)
						.map((/** @type {ObsEvent} */ e) => [e.generation, e.receivedAtMs])
				);
				for (let i = 1; i < isolationTriggeredGenerations.length; i++) {
					const prev = receivedAtByGen.get(isolationTriggeredGenerations[i - 1]);
					const cur = receivedAtByGen.get(isolationTriggeredGenerations[i]);
					if (prev !== undefined && cur !== undefined) {
						healthyMinInterArrival = Math.min(healthyMinInterArrival, cur - prev);
					}
				}
			}
			record.healthyTabsIsolationMinInterArrivalMs = Number.isFinite(healthyMinInterArrival)
				? healthyMinInterArrival
				: null;

			// Let the blocked tab catch up (its stalled fetch reader keeps
			// draining once its main thread is free again — no additional
			// trigger needed) and record its recovered generation.
			const newestGeneration = lastKnownGen;
			record.newestGeneration = newestGeneration;
			await pollUntil(
				async () => {
					const obs = await readObservations(pages[BLOCKED_IDX]);
					return obs.events.some((/** @type {ObsEvent} */ e) => !e.seeded && e.generation === newestGeneration)
						? obs
						: null;
				},
				15000,
				100
			);
			const blockedObsAfter = await readObservations(pages[BLOCKED_IDX]);
			const blockedGens = blockedObsAfter.events
				.filter((/** @type {ObsEvent} */ e) => !e.seeded)
				.map((/** @type {ObsEvent} */ e) => e.generation);
			record.slowTabFinalGeneration = blockedGens.length > 0 ? Math.max(...blockedGens) : null;

			// --- RECONNECT: a real process kill, a real downtime spanning at
			// least two scheduled backoff intervals, a real second process,
			// and the SAME stable proxy origin throughout. ---
			const preRestartObs = await Promise.all(pages.map((p) => readObservations(p)));
			const preRestartConnCount = preRestartObs.map((obs) => obs.connections.length);
			const preRestartEpoch = preRestartObs.map((obs) => {
				const conns = obs.connections;
				return conns.length > 0 ? conns[conns.length - 1].epoch : 0;
			});

			await stopChild(child);
			child = undefined;
			const downtimeStart = Date.now();
			await sleep(minDowntimeMs + 250); // real margin over the floor

			const restarted = await startCodegraphUi(binaryPath, root, { CODEGRAPH_DEBOUNCE_MS: UI_DEBOUNCE_MS });
			secondChild = restarted.child;
			const actualDowntimeMs = Date.now() - downtimeStart;
			record.actualDowntimeMs = actualDowntimeMs;

			if (!redSkipRepoint) {
				proxy.setUpstream(restarted.url);
			}
			// redSkipRepoint: deliberately leave the proxy pointed at the
			// OLD, now-dead upstream — see this file's header comment. Every
			// tab's reconnect attempt will keep 502'ing forever, and the
			// wait below will time out, producing success: false with a
			// stated reason.

			await Promise.all(
				pages.map((p, i) =>
					pollUntil(
						async () => {
							const obs = await readObservations(p);
							return obs.connections.length > preRestartConnCount[i] &&
								obs.events.some((/** @type {ObsEvent} */ e) => e.epoch > preRestartEpoch[i])
								? obs
								: null;
						},
						30000,
						150
					)
				)
			);

			const postReconnectObs = await Promise.all(pages.map((p) => readObservations(p)));
			for (let i = 0; i < TAB_COUNT; i++) {
				const obs = postReconnectObs[i];
				const postEntries = obs.connections
					.slice(preRestartConnCount[i])
					.filter((/** @type {ObsConnection} */ c) => c.scheduledDelayMs !== null);
				record.reconnectAttemptsPerTab.push(postEntries.length);
				record.reconnectDelaysPerTab.push(postEntries.map((/** @type {ObsConnection} */ c) => c.scheduledDelayMs));
				record.reconnectBaseDelaysPerTab.push(postEntries.map((/** @type {ObsConnection} */ c) => c.baseDelayMs));
				record.resumeCursorSentPerTab.push(postEntries.length > 0 ? postEntries[0].requestedSinceGeneration : 0);
				record.postReconnectAppliedPerTab.push(
					obs.events.filter(
						(/** @type {ObsEvent} */ e) => e.epoch > preRestartEpoch[i] && e.appliedAtMs !== null
					).length
				);
			}

			record.pageErrorCount = pageErrors.length;
			record.pageErrors = pageErrors;
		} catch (err) {
			record.error = err instanceof Error ? err.message : String(err);
			if (err instanceof Error && err.stack) record.errorStack = err.stack;
		}
	} finally {
		let corpusCleanAtExit = false;
		try {
			for (const relPath of createdProbePaths) {
				const absPath = path.join(root, relPath);
				if (fs.existsSync(absPath)) fs.unlinkSync(absPath);
			}
			const probeDir = path.join(root, 'internal', 'livepushprobe');
			if (fs.existsSync(probeDir) && fs.readdirSync(probeDir).length === 0) {
				fs.rmdirSync(probeDir);
			}
			if (createdProbePaths.length > 0) {
				runSync(binaryPath, root);
			}
			const gitStatus = execFileSync('git', ['status', '--porcelain', '--', 'internal/livepushprobe'], {
				cwd: root
			}).toString();
			corpusCleanAtExit = gitStatus.trim().length === 0;
			if (!corpusCleanAtExit) record.probeStatusPorcelain = gitStatus;
		} catch (cleanupErr) {
			corpusCleanAtExit = false;
			record.cleanupError = cleanupErr instanceof Error ? cleanupErr.message : String(cleanupErr);
		}
		record.corpusCleanAtExit = corpusCleanAtExit;

		if (browser) await browser.close().catch(() => {});
		await stopChild(child);
		await stopChild(secondChild);
		if (proxy) await proxy.close().catch(() => {});

		const arraysNonEmptyAndConsistent =
			Array.isArray(record.reconnectDelaysPerTab) &&
			record.reconnectDelaysPerTab.length === TAB_COUNT &&
			record.reconnectDelaysPerTab.every((/** @type {number[]} */ d) => d.length >= 2);

		record.success =
			record.error === null &&
			record.pageErrorCount === 0 &&
			record.triggeredGenerations.length >= 3 &&
			record.receivedGenerationsPerTab.length === TAB_COUNT &&
			record.receivedGenerationsPerTab.every((/** @type {number[]} */ g) =>
				record.triggeredGenerations.every((/** @type {number} */ t) => g.includes(t))
			) &&
			typeof record.minInterArrivalMs === 'number' &&
			record.minInterArrivalMs >= record.triggerSpacingMs &&
			typeof record.blockedTabBlockedMs === 'number' &&
			record.blockedTabBlockedMs >= record.triggerSpacingMs &&
			record.blockedTabReceiptsDuringBlock === 0 &&
			record.healthyTabsIsolationSupersetOK === true &&
			record.slowTabFinalGeneration === record.newestGeneration &&
			arraysNonEmptyAndConsistent &&
			Array.isArray(record.reconnectAttemptsPerTab) &&
			Math.min(...record.reconnectAttemptsPerTab) >= 2 &&
			Math.max(...record.reconnectAttemptsPerTab) <= 6 &&
			Array.isArray(record.postReconnectAppliedPerTab) &&
			Math.min(...record.postReconnectAppliedPerTab) >= 1 &&
			Array.isArray(record.resumeCursorSentPerTab) &&
			Math.min(...record.resumeCursorSentPerTab) > 0 &&
			record.corpusCleanAtExit === true;

		fs.mkdirSync(path.dirname(outPath), { recursive: true });
		fs.writeFileSync(outPath, JSON.stringify(record, null, 2) + '\n');
		console.log(`live-push-multitab-check: wrote ${outPath} — success=${record.success}`);
	}

	if (!record.success) {
		throw new Error(`live-push-multitab-check: FAILED — error=${record.error} success=${record.success}`);
	}
}

const isMain = process.argv[1] && url.pathToFileURL(process.argv[1]).href === import.meta.url;
if (isMain) {
	main().catch((err) => {
		console.error(`live-push-multitab-check: fatal — ${err instanceof Error ? err.message : String(err)}`);
		if (err instanceof Error && err.stack) console.error(err.stack);
		process.exit(1);
	});
}
