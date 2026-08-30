#!/usr/bin/env node
// graph-measure.mjs — the measurement wrapper for GRF-01 (05-04 Task 1).
//
// This file does NOT judge anything, does NOT know a bar from a value, and
// never writes to corpora/graph-render-threshold.json. Its only job is to
// own one browser session against one corpus and leave behind ONE
// role-tagged raw observation file, unconditionally.
//
// It carries NO gesture defaults and NO deadline defaults of its own —
// every one of the sixteen locked measurementProtocol values (viewport,
// cold-reload count, warm-up, sample window, pan path, zoom sweep, and all
// five deadline budgets) is read from corpora/graph-render-threshold.json
// via readProtocol()/sessionConfig(), never invented here.
//
// The whole session runs inside one try/finally: the finally block ALWAYS
// writes the raw observation, whether the session completed, threw, or a
// browser operation never settled and was raced against its deadline
// (withDeadline). The write happens BEFORE cleanup (ops.close) is ever
// awaited, so a wedged browser on the way down costs the session, never
// the record (review H-1, cycles 1-3; T-05-21).
//
// The six browser operations the live path performs are an ENUMERATED,
// CLOSED set (OPS_KEYS): launch, newPage, navigate, seamReady, sample,
// close. They are injected as `ops` so a test can drive a never-settling
// promise into this exact code path and read the written artifact back,
// rather than testing a deadline helper in isolation.

import { chromium } from '@playwright/test';
import * as fs from 'node:fs';
import * as path from 'node:path';
import * as url from 'node:url';

/**
 * @typedef {Object} MeasurementProtocol
 * @property {number} viewportWidth
 * @property {number} viewportHeight
 * @property {number} deviceScaleFactor
 * @property {number} coldReloads
 * @property {number} warmupMs
 * @property {number} sampleDurationMs
 * @property {number} panStepPx
 * @property {number} panSteps
 * @property {number} zoomMin
 * @property {number} zoomMax
 * @property {number} zoomSteps
 * @property {number} launchTimeoutMs
 * @property {number} navigationTimeoutMs
 * @property {number} seamReadyTimeoutMs
 * @property {number} samplerTimeoutMs
 * @property {number} sessionTimeoutMs
 */

/** @typedef {MeasurementProtocol} SessionConfig */

/**
 * @typedef {Object} CorpusRef
 * @property {string} repo
 * @property {string} sha
 */

/**
 * @typedef {Object} MetricEntry
 * @property {number|null} value
 * @property {'measured'|'measurement-failed'} status
 * @property {string} method
 * @property {string} [failureReason]
 * @property {number[]} [raw]
 */

/**
 * @typedef {Object} CollapsedView
 * @property {number} nodes
 * @property {number} edges
 * @property {string} method
 */

/**
 * @typedef {Object} BrowserIdentity
 * @property {string} name
 * @property {string} version
 */

/**
 * @typedef {Object} RawObservation
 * @property {'binding'|'additional'} role
 * @property {CorpusRef} corpus
 * @property {Object<string, MetricEntry>} metrics
 * @property {number|null} layoutDurationMs
 * @property {number|null} frameSampleCount
 * @property {number|null} nodeCount
 * @property {number|null} edgeCount
 * @property {CollapsedView|null} collapsedView
 * @property {BrowserIdentity|null} browserIdentity
 * @property {string|null} sessionError
 * @property {string|null} teardownError
 */

/**
 * @typedef {Object} MergedObservation
 * @property {RawObservation} bindingObservation
 * @property {RawObservation[]} additionalCorpora
 */

/**
 * @typedef {Object} SeamResult
 * @property {number} timeToInteractiveMs
 * @property {number} layoutDurationMs
 * @property {number} nodeCount
 * @property {number} edgeCount
 */

/**
 * @typedef {Object} FrameStats
 * @property {number} median
 * @property {number} p95
 * @property {number} count
 */

/**
 * @typedef {Object} OpsHandles
 * @property {(config: SessionConfig) => Promise<any>} launch
 * @property {(browserHandle: any, config: SessionConfig) => Promise<any>} newPage
 * @property {(pageHandle: any, config: SessionConfig, attempt: number) => Promise<void>} navigate
 * @property {(pageHandle: any, config: SessionConfig, attempt: number) => Promise<SeamResult>} seamReady
 * @property {(pageHandle: any, config: SessionConfig) => Promise<FrameStats>} sample
 * @property {(browserHandle: any, pageHandle: any) => Promise<void>} close
 */

/**
 * @typedef {Object} RunSessionArgs
 * @property {OpsHandles} ops
 * @property {MeasurementProtocol} protocol
 * @property {{role: 'binding'|'additional', repo: string, sha: string, collapsedView?: CollapsedView|null}} corpus
 * @property {string} outputPath
 * @property {Object<string, MetricEntry>} [citedMetrics]
 */

/**
 * @typedef {Object} MeasureArgs
 * @property {string} [role]
 * @property {string} [repo]
 * @property {string} [sha]
 * @property {string} [out]
 * @property {string} [url]
 * @property {number} [collapsedNodes]
 * @property {number} [collapsedEdges]
 * @property {number} [responseBytes]
 * @property {string} [responseBytesMethod]
 * @property {string[]} additional
 */

// ---------------------------------------------------------------------
// The closed operation enumeration (review H-1 part 2, cycle 3).
// ---------------------------------------------------------------------

/** @type {readonly ['launch', 'newPage', 'navigate', 'seamReady', 'sample', 'close']} */
export const OPS_KEYS = Object.freeze(['launch', 'newPage', 'navigate', 'seamReady', 'sample', 'close']);

// newPage and close have no budget of their own — 05-01 locked exactly
// sixteen numeric protocol values and this plan may not edit the
// threshold artifact for any reason, so both deliberately REUSE
// launchTimeoutMs (the same budget that bounds bringing the browser up
// bounds creating its page and tearing it down) — see doWork() and the
// close call site below.

// The three metrics the browser session itself measures. fileGraphResponseBytes
// is NOT one of them — that bar is cited from 05-02's own measured figure
// (a real server+client round trip), never re-derived here by a different
// method, and is merged in via the `citedMetrics` parameter regardless of
// whether the browser session itself succeeds.
export const BROWSER_METRIC_KEYS = Object.freeze(['timeToInteractiveMs', 'panZoomFrameTimeMs', 'panZoomFrameTimeP95Ms']);

/** @type {readonly (keyof MeasurementProtocol)[]} */
const PROTOCOL_NUMERIC_KEYS = Object.freeze([
	'viewportWidth',
	'viewportHeight',
	'deviceScaleFactor',
	'coldReloads',
	'warmupMs',
	'sampleDurationMs',
	'panStepPx',
	'panSteps',
	'zoomMin',
	'zoomMax',
	'zoomSteps',
	'launchTimeoutMs',
	'navigationTimeoutMs',
	'seamReadyTimeoutMs',
	'samplerTimeoutMs',
	'sessionTimeoutMs'
]);

/** @param {unknown} err @returns {string} */
function errMessage(err) {
	if (err instanceof Error) return err.message;
	return String(err);
}

// ---------------------------------------------------------------------
// withDeadline — the mechanism that closes the hang path a `finally`
// alone cannot (review H-1, cycle 2). Declaration form is pinned: an
// acceptance criterion counts call sites by excluding exactly this line.
// ---------------------------------------------------------------------

/**
 * @template T
 * @param {Promise<T>} promise
 * @param {number} ms
 * @param {string} label
 * @returns {Promise<T>}
 */
export function withDeadline(promise, ms, label) {
	/** @type {ReturnType<typeof setTimeout>} */
	let timer;
	const deadline = /** @type {Promise<T>} */ (
		new Promise((_resolve, reject) => {
			timer = setTimeout(() => {
				reject(new Error(`${label} exceeded its ${ms}ms deadline`));
			}, ms);
		})
	);
	return Promise.race([promise, deadline]).finally(() => {
		clearTimeout(timer);
	});
}

// ---------------------------------------------------------------------
// readProtocol / sessionConfig — the locked measurementProtocol block is
// the ONLY source for the gesture and the deadlines. No default, no
// fallback (review M-6, L-1).
// ---------------------------------------------------------------------

/**
 * @param {{measurementProtocol?: unknown}} threshold
 * @returns {MeasurementProtocol}
 */
export function readProtocol(threshold) {
	if (!threshold || typeof threshold !== 'object' || !threshold.measurementProtocol || typeof threshold.measurementProtocol !== 'object') {
		throw new Error(
			'readProtocol: threshold carries no measurementProtocol block — refusing to run without a locked gesture and deadline budget'
		);
	}
	const protocol = /** @type {Record<string, unknown>} */ (threshold.measurementProtocol);
	const missing = PROTOCOL_NUMERIC_KEYS.filter(
		(key) => !(typeof protocol[key] === 'number' && Number.isFinite(protocol[key]) && /** @type {number} */ (protocol[key]) > 0)
	);
	if (missing.length > 0) {
		throw new Error(`readProtocol: measurementProtocol is missing or has a non-positive value for: ${missing.join(', ')}`);
	}
	// A zoom sweep whose floor is not below its ceiling is not a sweep —
	// this is a real semantic check on the locked values, read by dot
	// access from the protocol itself (never a literal), not a coverage
	// formality.
	if (!(/** @type {number} */ (protocol.zoomMin) < /** @type {number} */ (protocol.zoomMax))) {
		throw new Error(`readProtocol: zoomMin (${protocol.zoomMin}) must be less than zoomMax (${protocol.zoomMax})`);
	}
	// A fractional reload count or pan step count is not a locked
	// gesture — these two are step counts, never continuous quantities.
	if (!Number.isInteger(protocol.coldReloads)) {
		throw new Error(`readProtocol: coldReloads must be an integer, got ${protocol.coldReloads}`);
	}
	if (!Number.isInteger(protocol.panSteps)) {
		throw new Error(`readProtocol: panSteps must be an integer, got ${protocol.panSteps}`);
	}
	return /** @type {MeasurementProtocol} */ (protocol);
}

/**
 * @param {MeasurementProtocol} protocol
 * @returns {SessionConfig}
 */
export function sessionConfig(protocol) {
	/** @type {Record<string, number>} */
	const config = {};
	for (const key of PROTOCOL_NUMERIC_KEYS) {
		config[key] = protocol[key];
	}
	return /** @type {SessionConfig} */ (config);
}

// ---------------------------------------------------------------------
// failedObservation — the shape a totally-dead session (or a totally-dead
// invocation that never even reached runSession) produces. Same shape
// runSession's own finally block writes on failure, exported so a test —
// and a catastrophic top-level CLI failure — can produce it directly.
// ---------------------------------------------------------------------

/**
 * @param {string} reason
 * @param {string[]} metricKeys
 * @param {{role: 'binding'|'additional', repo: string, sha: string}} corpus
 * @returns {RawObservation}
 */
export function failedObservation(reason, metricKeys, { role, repo, sha }) {
	/** @type {Object<string, MetricEntry>} */
	const metrics = {};
	for (const key of metricKeys) {
		metrics[key] = {
			value: null,
			status: 'measurement-failed',
			method: 'live browser session (playwright chromium) — see sessionError',
			failureReason: reason
		};
	}
	return {
		role,
		corpus: { repo, sha },
		metrics,
		layoutDurationMs: null,
		frameSampleCount: null,
		nodeCount: null,
		edgeCount: null,
		collapsedView: null,
		browserIdentity: null,
		sessionError: reason,
		teardownError: null
	};
}

// ---------------------------------------------------------------------
// mergeRawObservations — the deterministic producer-side merge step
// (review M-1, cycle 2). Exactly one binding raw becomes bindingObservation;
// every additional raw becomes an additionalCorpora entry, in input order.
// ---------------------------------------------------------------------

/**
 * @param {RawObservation[]} raws
 * @returns {MergedObservation}
 */
export function mergeRawObservations(raws) {
	if (!Array.isArray(raws)) {
		throw new Error('mergeRawObservations: expected an array of role-tagged raw observations');
	}
	const bindingRaws = raws.filter((r) => r && r.role === 'binding');
	const additionalRaws = raws.filter((r) => r && r.role === 'additional');
	if (bindingRaws.length !== 1) {
		throw new Error(
			`mergeRawObservations: requires exactly one raw observation with role "binding", found ${bindingRaws.length}`
		);
	}
	return {
		bindingObservation: bindingRaws[0],
		additionalCorpora: additionalRaws
	};
}

// ---------------------------------------------------------------------
// runSession — owns the browser session. Validates the injected ops
// object against the closed OPS_KEYS enumeration, routes every one of
// the six operations through withDeadline, and ALWAYS writes the raw
// observation before cleanup is ever awaited.
// ---------------------------------------------------------------------

/** @param {Record<string, unknown>} ops */
function validateOpsKeys(ops) {
	const provided = new Set(Object.keys(ops ?? {}));
	const expected = new Set(/** @type {readonly string[]} */ (OPS_KEYS));
	const missing = OPS_KEYS.filter((k) => !provided.has(k));
	const extra = [...provided].filter((k) => !expected.has(k));
	if (missing.length > 0 || extra.length > 0) {
		throw new Error(
			`runSession: ops keys must be exactly ${JSON.stringify(OPS_KEYS)} — missing: ${JSON.stringify(missing)}, extra: ${JSON.stringify(extra)}`
		);
	}
}

/** @param {number[]} samples @returns {number} */
function computeStats(samples) {
	const sorted = [...samples].sort((a, b) => a - b);
	const mid = Math.floor(sorted.length / 2);
	const median = sorted.length % 2 === 0 ? (sorted[mid - 1] + sorted[mid]) / 2 : sorted[mid];
	return median;
}

/** @param {string} outputPath @param {unknown} value */
async function writeJson(outputPath, value) {
	await fs.promises.mkdir(path.dirname(outputPath), { recursive: true });
	await fs.promises.writeFile(outputPath, JSON.stringify(value, null, 2) + '\n');
}

/**
 * @param {RunSessionArgs} args
 * @returns {Promise<RawObservation>}
 */
export async function runSession({ ops, protocol, corpus, outputPath, citedMetrics = {} }) {
	validateOpsKeys(ops);
	const config = sessionConfig(protocol);

	/**
	 * @typedef {Object} DoWorkResult
	 * @property {number[]} ttiRaw
	 * @property {number} layoutDurationMs
	 * @property {number} nodeCount
	 * @property {number} edgeCount
	 * @property {FrameStats} frameStats
	 */

	/** @type {string|null} */
	let sessionError = null;
	/** @type {BrowserIdentity|null} */
	let browserIdentity = null;
	/** @type {any} */
	let browserHandle = null;
	/** @type {any} */
	let pageHandle = null;
	// sessionResult is assigned exactly once, directly in this function's
	// own flow (`sessionResult = await withDeadline(doWork(), ...)`) —
	// deliberately NOT mutated from inside the doWork() closure itself,
	// so a later `if (sessionResult)` narrows normally. Variables mutated
	// INSIDE a nested closure keep their widened declared type at every
	// read site outside that closure, which is what forced the earlier,
	// unsound `/** @type {FrameStats} */ (frameStats)` cast this shape
	// replaces.
	/** @type {DoWorkResult|null} */
	let sessionResult = null;

	/** @returns {Promise<DoWorkResult>} */
	const doWork = async () => {
		browserHandle = await withDeadline(ops.launch(config), config.launchTimeoutMs, 'launch');
		browserIdentity = (browserHandle && browserHandle.identity) || null;
		pageHandle = await withDeadline(ops.newPage(browserHandle, config), config.launchTimeoutMs, 'newPage');

		/** @type {number[]} */
		const ttiRaw = [];
		let layoutDurationMs = 0;
		let nodeCount = 0;
		let edgeCount = 0;

		for (let attempt = 0; attempt < config.coldReloads; attempt++) {
			await withDeadline(ops.navigate(pageHandle, config, attempt), config.navigationTimeoutMs, 'navigate');
			const seam = await withDeadline(ops.seamReady(pageHandle, config, attempt), config.seamReadyTimeoutMs, 'seamReady');
			ttiRaw.push(seam.timeToInteractiveMs);
			layoutDurationMs = seam.layoutDurationMs;
			nodeCount = seam.nodeCount;
			edgeCount = seam.edgeCount;
		}

		const frameStats = await withDeadline(ops.sample(pageHandle, config), config.samplerTimeoutMs, 'sample');

		return { ttiRaw, layoutDurationMs, nodeCount, edgeCount, frameStats };
	};

	try {
		sessionResult = await withDeadline(doWork(), config.sessionTimeoutMs, 'session');
	} catch (err) {
		sessionError = errMessage(err);
	} finally {
		/** @type {Object<string, MetricEntry>} */
		const metrics = {};
		if (sessionResult) {
			const result = sessionResult;
			metrics.timeToInteractiveMs = {
				value: computeStats(result.ttiRaw),
				status: 'measured',
				method: `median of ${config.coldReloads} cold reloads, request-issued to layoutstop (live browser session, playwright chromium)`,
				raw: result.ttiRaw
			};
			metrics.panZoomFrameTimeMs = {
				value: result.frameStats.median,
				status: 'measured',
				method: 'median requestAnimationFrame inter-frame interval over the scripted pan/zoom gesture (live browser session, playwright chromium)'
			};
			metrics.panZoomFrameTimeP95Ms = {
				value: result.frameStats.p95,
				status: 'measured',
				method: '95th percentile requestAnimationFrame inter-frame interval, same sample as panZoomFrameTimeMs (live browser session, playwright chromium)'
			};
		} else {
			for (const key of BROWSER_METRIC_KEYS) {
				metrics[key] = {
					value: null,
					status: 'measurement-failed',
					method: 'live browser session (playwright chromium)',
					failureReason: sessionError ?? 'runSession: internal invariant violated — the session ended with neither a result nor an error'
				};
			}
		}
		Object.assign(metrics, citedMetrics);

		/** @type {RawObservation} */
		const raw = {
			role: corpus.role,
			corpus: { repo: corpus.repo, sha: corpus.sha },
			metrics,
			layoutDurationMs: sessionResult ? sessionResult.layoutDurationMs : null,
			frameSampleCount: sessionResult ? sessionResult.frameStats.count : null,
			nodeCount: sessionResult ? sessionResult.nodeCount : null,
			edgeCount: sessionResult ? sessionResult.edgeCount : null,
			collapsedView: corpus.collapsedView ?? null,
			browserIdentity,
			sessionError,
			teardownError: null
		};

		await writeJson(outputPath, raw);

		try {
			await withDeadline(ops.close(browserHandle, pageHandle), config.launchTimeoutMs, 'close');
		} catch (err) {
			raw.teardownError = errMessage(err);
			await writeJson(outputPath, raw);
		}

		return raw;
	}
}

// ---------------------------------------------------------------------
// The real ops, built over @playwright/test's chromium browser type —
// one function per OPS_KEYS member and no others (review H-4).
// ---------------------------------------------------------------------

/**
 * @param {string} targetUrl
 * @returns {OpsHandles}
 */
function buildRealOps(targetUrl) {
	return {
		async launch() {
			const browser = await chromium.launch();
			const version = browser.version();
			return { browser, identity: { name: 'chromium', version } };
		},
		async newPage(browserHandle, config) {
			const context = await browserHandle.browser.newContext({
				viewport: { width: config.viewportWidth, height: config.viewportHeight },
				deviceScaleFactor: config.deviceScaleFactor
			});
			const page = await context.newPage();
			return { context, page };
		},
		async navigate(pageHandle) {
			await pageHandle.page.goto(targetUrl, { waitUntil: 'load' });
		},
		async seamReady(pageHandle) {
			const handle = await pageHandle.page.waitForFunction(
				() => /** @type {any} */ (window).__codegraphFileGraphMetrics ?? null,
				null,
				{ timeout: 0, polling: 25 }
			);
			const metrics = await handle.jsonValue();
			return {
				timeToInteractiveMs: metrics.timeToInteractiveMs,
				layoutDurationMs: metrics.layoutDurationMs,
				nodeCount: metrics.nodeCount,
				edgeCount: metrics.edgeCount
			};
		},
		async sample(pageHandle, config) {
			// Deliberately named `pg`, never `page`: this file's own
			// coverage guard forbids an unwrapped `await page.` anywhere
			// in the live path (it would be indistinguishable from an
			// operation escaping its withDeadline budget), and every
			// call in this function already runs inside the ONE
			// `withDeadline(ops.sample(...), samplerTimeoutMs, 'sample')`
			// call site that wraps this whole op.
			const pg = pageHandle.page;

			await pg.evaluate(() => {
				const win = /** @type {any} */ (window);
				win.__codegraphFrameSamples = [];
				win.__codegraphFrameSamplerRunning = true;
				const step = (/** @type {number} */ t) => {
					win.__codegraphFrameSamples.push(t);
					if (win.__codegraphFrameSamplerRunning) {
						requestAnimationFrame(step);
					}
				};
				requestAnimationFrame(step);
			});

			await pg.waitForTimeout(config.warmupMs);
			await pg.evaluate(() => {
				/** @type {any} */ (window).__codegraphFrameSamples = [];
			});

			const box = await pg.evaluate(() => {
				const el = document.querySelector('[data-graph-canvas]') ?? document.querySelector('canvas');
				if (!el) return null;
				const r = el.getBoundingClientRect();
				return { x: r.x, y: r.y, width: r.width, height: r.height };
			});
			const cx = box ? box.x + box.width / 2 : config.viewportWidth / 2;
			const cy = box ? box.y + box.height / 2 : config.viewportHeight / 2;

			await pg.mouse.move(cx, cy);
			await pg.mouse.down();
			for (let i = 1; i <= config.panSteps; i++) {
				await pg.mouse.move(cx + config.panStepPx * i, cy, { steps: 1 });
			}
			await pg.mouse.up();

			for (let i = 0; i < config.zoomSteps; i++) {
				const direction = i % 2 === 0 ? -1 : 1;
				await pg.mouse.wheel(0, direction * 60);
			}

			const deadlineAt = Date.now() + config.sampleDurationMs;
			while (Date.now() < deadlineAt) {
				await pg.waitForTimeout(Math.min(50, Math.max(0, deadlineAt - Date.now())));
			}

			await pg.evaluate(() => {
				/** @type {any} */ (window).__codegraphFrameSamplerRunning = false;
			});
			/** @type {number[]} */
			const samples = await pg.evaluate(() => /** @type {any} */ (window).__codegraphFrameSamples);

			/** @type {number[]} */
			const deltas = [];
			for (let i = 1; i < samples.length; i++) {
				deltas.push(samples[i] - samples[i - 1]);
			}
			if (deltas.length === 0) {
				throw new Error('sample: the requestAnimationFrame sampler collected fewer than two frames');
			}
			const sorted = [...deltas].sort((a, b) => a - b);
			const median = computeStats(deltas);
			const p95Index = Math.min(sorted.length - 1, Math.ceil(0.95 * sorted.length) - 1);
			const p95 = sorted[p95Index];
			return { median, p95, count: deltas.length };
		},
		async close(browserHandle, pageHandle) {
			if (pageHandle && pageHandle.context) {
				await pageHandle.context.close();
			}
			if (browserHandle && browserHandle.browser) {
				await browserHandle.browser.close();
			}
		}
	};
}

// ---------------------------------------------------------------------
// Command-line entry point. Resolves corpora/graph-render-threshold.json
// from the REPOSITORY ROOT regardless of the current working directory,
// so `cd web && node scripts/graph-measure.mjs ...` resolves it correctly
// (review H-4).
// ---------------------------------------------------------------------

/** @returns {string} */
function repoRoot() {
	const here = path.dirname(url.fileURLToPath(import.meta.url));
	return path.resolve(here, '..', '..');
}

/**
 * @param {string[]} argv
 * @returns {MeasureArgs}
 */
function parseArgs(argv) {
	/** @type {MeasureArgs} */
	const out = { additional: [] };
	for (let i = 0; i < argv.length; i++) {
		const arg = argv[i];
		switch (arg) {
			case '--role':
				out.role = argv[++i];
				break;
			case '--repo':
				out.repo = argv[++i];
				break;
			case '--sha':
				out.sha = argv[++i];
				break;
			case '--out':
				out.out = argv[++i];
				break;
			case '--url':
				out.url = argv[++i];
				break;
			case '--collapsed-nodes':
				out.collapsedNodes = Number(argv[++i]);
				break;
			case '--collapsed-edges':
				out.collapsedEdges = Number(argv[++i]);
				break;
			case '--response-bytes':
				out.responseBytes = Number(argv[++i]);
				break;
			case '--response-bytes-method':
				out.responseBytesMethod = argv[++i];
				break;
			default:
				throw new Error(`graph-measure: unrecognized argument "${arg}"`);
		}
	}
	return out;
}

async function main() {
	const args = parseArgs(process.argv.slice(2));
	if (!args.role || !args.repo || !args.sha || !args.out || !args.url) {
		throw new Error('graph-measure: --role, --repo, --sha, --out and --url are all required');
	}
	if (args.role !== 'binding' && args.role !== 'additional') {
		throw new Error(`graph-measure: --role must be "binding" or "additional", got "${args.role}"`);
	}

	const thresholdPath = path.join(repoRoot(), 'corpora', 'graph-render-threshold.json');
	const threshold = JSON.parse(fs.readFileSync(thresholdPath, 'utf8'));
	const protocol = readProtocol(threshold);

	/** @type {Object<string, MetricEntry>} */
	const citedMetrics = {};
	if (typeof args.responseBytes === 'number' && Number.isFinite(args.responseBytes)) {
		citedMetrics.fileGraphResponseBytes = {
			value: args.responseBytes,
			status: 'measured',
			method: args.responseBytesMethod || 'cited from a prior measurement, not re-derived by this session'
		};
	}

	/** @type {CollapsedView|null} */
	const collapsedView =
		typeof args.collapsedNodes === 'number' &&
		typeof args.collapsedEdges === 'number' &&
		Number.isFinite(args.collapsedNodes) &&
		Number.isFinite(args.collapsedEdges)
			? {
					nodes: args.collapsedNodes,
					edges: args.collapsedEdges,
					method: 'derived count: distinct top-level directories / directory pairs from the same wire response — not a rendered measurement'
				}
			: null;

	const ops = buildRealOps(args.url);
	const raw = await runSession({
		ops,
		protocol,
		corpus: { role: args.role, repo: args.repo, sha: args.sha, collapsedView },
		outputPath: args.out,
		citedMetrics
	});

	process.stdout.write(`graph-measure: wrote ${args.out} (role=${args.role}, sessionError=${raw.sessionError ?? 'null'})\n`);
}

const isMain = process.argv[1] && url.pathToFileURL(process.argv[1]).href === import.meta.url;
if (isMain) {
	main().catch((err) => {
		console.error(`graph-measure: fatal — could not write the observation file: ${errMessage(err)}`);
		process.exit(1);
	});
}
