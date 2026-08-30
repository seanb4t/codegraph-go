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
 * @typedef {import('./graph-measure.mjs').CollapsedView} CollapsedView
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

/**
 * @typedef {Object} WrittenVerdict
 * @property {number} schemaVersion
 * @property {string} thresholdRef
 * @property {{repo: string, sha: string}} corpus
 * @property {string|undefined} bindingView
 * @property {RawObservation} bindingObservation
 * @property {RawObservation[]} additionalCorpora
 * @property {MetricResult[]} metricResults
 * @property {'PASS'|'FAIL'} verdict
 * @property {{layoutDurationMs: number|null, frameSampleCount: number|null, collapsedView: CollapsedView|null}} recordedNonBinding
 * @property {string} generatedBy
 * @property {string} generatedAt
 * @property {string} [note]
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
 * @property {string} [out]
 * @property {string} [note]
 */

/** @returns {string} */
function repoRoot() {
	const here = path.dirname(url.fileURLToPath(import.meta.url));
	return path.resolve(here, '..', '..');
}

/**
 * parseArgs — exported (05-08) so a test can exercise the flag surface
 * directly. An invocation using neither --out nor --note parses to
 * exactly the same keys it always has ({binding, additional}); the two
 * new flags are added to the returned object ONLY when supplied.
 *
 * @param {string[]} argv
 * @returns {VerdictArgs}
 */
export function parseArgs(argv) {
	/** @type {VerdictArgs} */
	const out = { additional: [] };
	for (let i = 0; i < argv.length; i++) {
		const arg = argv[i];
		if (arg === '--binding') {
			out.binding = argv[++i];
		} else if (arg === '--additional') {
			out.additional.push(argv[++i]);
		} else if (arg === '--out') {
			out.out = argv[++i];
		} else if (arg === '--note') {
			out.note = argv[++i];
		} else {
			throw new Error(`graph-verdict: unrecognized argument "${arg}"`);
		}
	}
	return out;
}

/**
 * resolveOutputPath — pure path resolution, no I/O. Exported (05-08) so
 * "no --out flag targets the fixed default" can be asserted against a
 * FAKE root in a test, without ever touching the real repository's
 * corpora/graph-render-observations.json (the recorded FAIL, T-05-41).
 *
 * @param {string} root
 * @param {string|undefined} outFlag
 * @returns {string}
 */
export function resolveOutputPath(root, outFlag) {
	return outFlag || path.join(root, 'corpora', 'graph-render-observations.json');
}

/**
 * writeVerdict — the write step (05-08), extracted so a re-measure at a
 * DIFFERENT rendered view (e.g. the collapsed default) can write to a NEW
 * artifact path without ever overwriting corpora/graph-render-observations.json
 * (T-05-41). Takes threshold/binding/additional/output/note explicitly —
 * it computes the verdict itself via compareObservation rather than
 * accepting a pre-computed one, so it stays a single self-contained unit
 * a test can call directly. `output` is REQUIRED here (no repo-root
 * default) — the CLI entry point below is the only caller allowed to fall
 * back to the fixed default path, via resolveOutputPath.
 *
 * @param {Object} args
 * @param {Threshold} args.threshold
 * @param {RawObservation} args.binding
 * @param {RawObservation[]} [args.additional]
 * @param {string} args.output
 * @param {string} [args.note]
 * @returns {WrittenVerdict} the written object (also the return value of JSON.parse(fs.readFileSync(args.output)))
 */
export function writeVerdict({ threshold, binding, additional = [], output, note }) {
	const result = compareObservation(threshold, { bindingObservation: binding, additionalCorpora: additional });
	/** @type {WrittenVerdict} */
	const written = {
		schemaVersion: 1,
		thresholdRef: 'corpora/graph-render-threshold.json',
		corpus: threshold.corpus,
		bindingView: threshold.bindingView,
		bindingObservation: result.bindingObservation,
		additionalCorpora: result.additionalCorpora,
		metricResults: result.metricResults,
		verdict: result.verdict,
		recordedNonBinding: {
			layoutDurationMs: result.bindingObservation.layoutDurationMs ?? null,
			frameSampleCount: result.bindingObservation.frameSampleCount ?? null,
			collapsedView: result.bindingObservation.collapsedView ?? null
		},
		generatedBy: 'web/scripts/graph-verdict.mjs',
		generatedAt: new Date().toISOString()
	};
	if (typeof note === 'string' && note.trim().length > 0) {
		written.note = note;
	}
	fs.writeFileSync(output, JSON.stringify(written, null, 2) + '\n');
	return written;
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
	const outputPath = resolveOutputPath(root, args.out);

	/** @type {Threshold} */
	const threshold = JSON.parse(fs.readFileSync(thresholdPath, 'utf8'));

	const raws = [readRoleTaggedFile(args.binding, 'binding')];
	for (const additionalPath of args.additional) {
		raws.push(readRoleTaggedFile(additionalPath, 'additional'));
	}

	const merged = mergeRawObservations(raws);
	const written = writeVerdict({
		threshold,
		binding: merged.bindingObservation,
		additional: merged.additionalCorpora,
		output: outputPath,
		note: args.note
	});

	process.stdout.write(`graph-verdict: wrote ${outputPath} — verdict: ${written.verdict}\n`);
}

const isMain = process.argv[1] && url.pathToFileURL(process.argv[1]).href === import.meta.url;
if (isMain) {
	main().catch((err) => {
		console.error(`graph-verdict: fatal — ${err instanceof Error ? err.message : String(err)}`);
		process.exit(1);
	});
}
