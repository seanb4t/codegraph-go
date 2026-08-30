#!/usr/bin/env node
// graph-verdict.mjs — the fail-closed comparator for GRF-01 (05-04 Task 1).
//
// This file does NOT measure anything, does NOT know how a browser works,
// and never writes to corpora/graph-render-threshold.json. It turns a
// threshold and an observation into a per-metric verdict, with every
// ambiguity resolving to FAIL — never a default, never a skip, never a
// warning tier. The most likely way this gate returns a wrong answer is a
// measurement that did not happen being read as one that passed; this
// script exists to close that specific failure mode.
//
// Judges bindingObservation.metrics and nothing else. additionalCorpora
// entries are copied through untouched and never reach metricResults — a
// second corpus's numbers judged against the binding corpus's bars is
// precisely the wrong-answer mode the split exists to close (review M-5).

import * as fs from 'node:fs';
import * as path from 'node:path';
import * as url from 'node:url';

import { mergeRawObservations } from './graph-measure.mjs';

/**
 * @typedef {import('./graph-measure.mjs').RawObservation} RawObservation
 * @typedef {import('./graph-measure.mjs').MetricEntry} MetricEntry
 */

/**
 * @typedef {Object} ThresholdMetric
 * @property {number} max
 */

/**
 * @typedef {Object} Threshold
 * @property {{repo: string, sha: string}} corpus
 * @property {Object<string, ThresholdMetric>} metrics
 * @property {string} [bindingView]
 */

/**
 * @typedef {Object} MetricResult
 * @property {string} metric
 * @property {number|null} measured
 * @property {'measured'|'measurement-failed'} status
 * @property {number} bar
 * @property {string} comparison
 * @property {'PASS'|'FAIL'} verdict
 * @property {string} reason
 * @property {string} [failureReason]
 */

/**
 * @typedef {Object} MergedObservationInput
 * @property {RawObservation} bindingObservation
 * @property {RawObservation[]} [additionalCorpora]
 */

/**
 * @typedef {Object} VerdictResult
 * @property {'PASS'|'FAIL'} verdict
 * @property {MetricResult[]} metricResults
 * @property {RawObservation} bindingObservation
 * @property {RawObservation[]} additionalCorpora
 */

const COMPARISON = '<=';

/**
 * @param {string} key
 * @param {number} bar
 * @param {MetricEntry|undefined} metric
 * @returns {MetricResult}
 */
function judgeMetric(key, bar, metric) {
	if (metric === undefined || metric === null) {
		const reason = `metric "${key}" is absent from the observation`;
		return { metric: key, measured: null, status: 'measurement-failed', bar, comparison: COMPARISON, verdict: 'FAIL', reason, failureReason: reason };
	}

	if (metric.status === 'measurement-failed') {
		const failureReason =
			typeof metric.failureReason === 'string' && metric.failureReason.trim().length > 0
				? metric.failureReason
				: 'measurement failed with no recorded reason';
		return {
			metric: key,
			measured: null,
			status: 'measurement-failed',
			bar,
			comparison: COMPARISON,
			verdict: 'FAIL',
			reason: `measurement failed: ${failureReason}`,
			failureReason
		};
	}

	const value = metric.value;
	if (metric.status === 'measured' && !(typeof value === 'number' && Number.isFinite(value))) {
		const reason = `metric "${key}" declares status "measured" but its value is not a finite number (${JSON.stringify(value)}) — self-contradictory record`;
		return { metric: key, measured: value ?? null, status: 'measurement-failed', bar, comparison: COMPARISON, verdict: 'FAIL', reason, failureReason: reason };
	}

	if (!(typeof value === 'number' && Number.isFinite(value))) {
		const reason = `metric "${key}" carries a non-numeric value (${JSON.stringify(value)}) — not a coerced comparison`;
		return { metric: key, measured: null, status: 'measurement-failed', bar, comparison: COMPARISON, verdict: 'FAIL', reason, failureReason: reason };
	}

	const verdict = value <= bar ? 'PASS' : 'FAIL';
	const reason = verdict === 'FAIL' ? `measured ${value} exceeds the bar of ${bar} (comparison: ${COMPARISON})` : '';
	return { metric: key, measured: value, status: 'measured', bar, comparison: COMPARISON, verdict, reason };
}

/**
 * compareObservation — the pure comparison function. Takes a parsed
 * threshold object and a merged-observation-shaped object
 * ({ bindingObservation, additionalCorpora? }) and returns a verdict
 * object. Throws on a broken invocation (unreadable threshold, absent
 * bindingObservation, wrong-corpus bindingObservation) — those are
 * operator errors to fix, never outcomes to record. A declared
 * measurement failure inside bindingObservation.metrics is NOT a broken
 * invocation — it is a recordable FAIL and this function returns
 * normally for it.
 *
 * @param {Threshold} threshold
 * @param {MergedObservationInput} observation
 * @returns {VerdictResult}
 */
export function compareObservation(threshold, observation) {
	if (!threshold || typeof threshold !== 'object' || !threshold.metrics || typeof threshold.metrics !== 'object') {
		throw new Error('compareObservation: threshold is missing or unparseable — refusing to produce a verdict without one');
	}

	if (
		!observation ||
		typeof observation !== 'object' ||
		!observation.bindingObservation ||
		typeof observation.bindingObservation !== 'object' ||
		Array.isArray(observation.bindingObservation)
	) {
		throw new Error('compareObservation: observation carries no bindingObservation object — there is nothing to judge');
	}

	const binding = observation.bindingObservation;
	const thresholdCorpus = threshold.corpus || { repo: '', sha: '' };
	const bindingCorpus = binding.corpus || { repo: '', sha: '' };
	if (bindingCorpus.repo !== thresholdCorpus.repo || bindingCorpus.sha !== thresholdCorpus.sha) {
		throw new Error(
			`compareObservation: binding observation corpus ${JSON.stringify(bindingCorpus)} does not match the locked threshold corpus ${JSON.stringify(thresholdCorpus)}`
		);
	}

	const metricKeys = Object.keys(threshold.metrics);
	const metricResults = metricKeys.map((key) =>
		judgeMetric(key, threshold.metrics[key].max, binding.metrics ? binding.metrics[key] : undefined)
	);
	const verdict = metricResults.some((r) => r.verdict === 'FAIL') ? 'FAIL' : 'PASS';

	return {
		verdict,
		metricResults,
		bindingObservation: binding,
		additionalCorpora: Array.isArray(observation.additionalCorpora) ? observation.additionalCorpora : []
	};
}

// ---------------------------------------------------------------------
// Command-line entry point.
// ---------------------------------------------------------------------

/**
 * @typedef {Object} VerdictArgs
 * @property {string} [binding]
 * @property {string[]} additional
 */

/** @returns {string} */
function repoRoot() {
	const here = path.dirname(url.fileURLToPath(import.meta.url));
	return path.resolve(here, '..', '..');
}

/**
 * @param {string[]} argv
 * @returns {VerdictArgs}
 */
function parseArgs(argv) {
	/** @type {VerdictArgs} */
	const out = { additional: [] };
	for (let i = 0; i < argv.length; i++) {
		const arg = argv[i];
		if (arg === '--binding') {
			out.binding = argv[++i];
		} else if (arg === '--additional') {
			out.additional.push(argv[++i]);
		} else {
			throw new Error(`graph-verdict: unrecognized argument "${arg}"`);
		}
	}
	return out;
}

/**
 * @param {string} filePath
 * @param {'binding'|'additional'} expectedRole
 * @returns {RawObservation}
 */
function readRoleTaggedFile(filePath, expectedRole) {
	const raw = JSON.parse(fs.readFileSync(filePath, 'utf8'));
	if (raw.role !== expectedRole) {
		throw new Error(
			`graph-verdict: ${filePath} is stamped role "${raw.role}" but arrived under the --${expectedRole} flag — mislabelled argument, not a silent corpus swap`
		);
	}
	return raw;
}

async function main() {
	const args = parseArgs(process.argv.slice(2));
	if (!args.binding) {
		throw new Error('graph-verdict: --binding <path> is required');
	}

	const root = repoRoot();
	const thresholdPath = path.join(root, 'corpora', 'graph-render-threshold.json');
	const observationsPath = path.join(root, 'corpora', 'graph-render-observations.json');

	/** @type {Threshold} */
	const threshold = JSON.parse(fs.readFileSync(thresholdPath, 'utf8'));

	const raws = [readRoleTaggedFile(args.binding, 'binding')];
	for (const additionalPath of args.additional) {
		raws.push(readRoleTaggedFile(additionalPath, 'additional'));
	}

	const merged = mergeRawObservations(raws);
	const result = compareObservation(threshold, merged);

	const binding = result.bindingObservation;
	const output = {
		schemaVersion: 1,
		thresholdRef: 'corpora/graph-render-threshold.json',
		corpus: threshold.corpus,
		bindingView: threshold.bindingView,
		bindingObservation: binding,
		additionalCorpora: result.additionalCorpora,
		metricResults: result.metricResults,
		verdict: result.verdict,
		recordedNonBinding: {
			layoutDurationMs: binding.layoutDurationMs ?? null,
			frameSampleCount: binding.frameSampleCount ?? null,
			collapsedView: binding.collapsedView ?? null
		},
		generatedBy: 'web/scripts/graph-verdict.mjs',
		generatedAt: new Date().toISOString()
	};

	fs.writeFileSync(observationsPath, JSON.stringify(output, null, 2) + '\n');
	process.stdout.write(`graph-verdict: wrote ${observationsPath} — verdict: ${result.verdict}\n`);
}

const isMain = process.argv[1] && url.pathToFileURL(process.argv[1]).href === import.meta.url;
if (isMain) {
	main().catch((err) => {
		console.error(`graph-verdict: fatal — ${err instanceof Error ? err.message : String(err)}`);
		process.exit(1);
	});
}
