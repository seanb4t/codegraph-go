// Tests for graph-verdict.mjs's pure comparator (05-04 Task 1 behavior block)
// plus its output-path/note extension (05-08 Task 1). Pure-module vitest
// convention, mirroring browse-url.test.ts: no DOM, no browser, no
// SvelteKit runtime. compareObservation is exercised directly so a
// fail-closed regression here can never hide behind a mocked browser.
//
// The four new cases below use `writeVerdict`/`resolveOutputPath`/`parseArgs`
// with an EXPLICIT scratch output path, never the real fixed default —
// writeVerdict never touches corpora/graph-render-observations.json
// (the recorded FAIL, T-05-41) inside a test.
import * as fs from 'node:fs';
import * as os from 'node:os';
import * as path from 'node:path';

import { describe, expect, it } from 'vitest';
import { compareObservation, parseArgs, resolveOutputPath, writeVerdict } from '../scripts/graph-verdict.mjs';
import { mergeRawObservations, failedObservation } from '../scripts/graph-measure.mjs';

// Test fixtures below deliberately populate only the fields
// compareObservation/mergeRawObservations actually read (corpus, metrics,
// role) — the cast documents that the remaining RawObservation fields are
// irrelevant to the behavior under test, not that the runtime shape is
// wrong.
type RawObservation = import('../scripts/graph-measure.mjs').RawObservation;

const CORPUS = { repo: 'google/guava', sha: '94f39958baf7ad51ddf9c70e406ed6b188194daa' };

function threshold() {
	return {
		corpus: CORPUS,
		metrics: {
			timeToInteractiveMs: { max: 5000 },
			panZoomFrameTimeMs: { max: 33.3 },
			panZoomFrameTimeP95Ms: { max: 100 },
			fileGraphResponseBytes: { max: 16777216 }
		}
	};
}

function measured(value: number) {
	return { value, status: 'measured' as const, method: 'test fixture' };
}

function passingBinding(): RawObservation {
	return {
		corpus: CORPUS,
		metrics: {
			timeToInteractiveMs: measured(4000),
			panZoomFrameTimeMs: measured(20),
			panZoomFrameTimeP95Ms: measured(80),
			fileGraphResponseBytes: measured(3713528)
		}
	} as unknown as RawObservation;
}

describe('compareObservation: every metric within bar', () => {
	it('yields overall PASS and a PASS for every metric, one result per threshold bar', () => {
		const result = compareObservation(threshold(), { bindingObservation: passingBinding() });
		expect(result.verdict).toBe('PASS');
		expect(result.metricResults).toHaveLength(4);
		for (const r of result.metricResults) {
			expect(r.verdict).toBe('PASS');
		}
	});
});

describe('compareObservation: one metric over its bar', () => {
	it('marks only that metric FAIL, overall FAIL, and every other metric stays PASS', () => {
		const binding = passingBinding();
		binding.metrics.timeToInteractiveMs = measured(9000);
		const result = compareObservation(threshold(), { bindingObservation: binding });
		expect(result.verdict).toBe('FAIL');
		const byKey = Object.fromEntries(result.metricResults.map((r) => [r.metric, r.verdict]));
		expect(byKey.timeToInteractiveMs).toBe('FAIL');
		expect(byKey.panZoomFrameTimeMs).toBe('PASS');
		expect(byKey.panZoomFrameTimeP95Ms).toBe('PASS');
		expect(byKey.fileGraphResponseBytes).toBe('PASS');
	});
});

describe('compareObservation: absent metric', () => {
	it('yields FAIL for the metric and overall, with a reason naming the missing key', () => {
		const binding = passingBinding();
		delete (binding.metrics as any).panZoomFrameTimeP95Ms;
		const result = compareObservation(threshold(), { bindingObservation: binding });
		expect(result.verdict).toBe('FAIL');
		const entry = result.metricResults.find((r) => r.metric === 'panZoomFrameTimeP95Ms');
		expect(entry?.verdict).toBe('FAIL');
		expect(entry?.reason).toContain('panZoomFrameTimeP95Ms');
	});
});

describe('compareObservation: non-numeric present metric', () => {
	it.each([
		['a string', 'not-a-number'],
		[null, null],
		[NaN, NaN]
	])('%s yields FAIL, not a coerced comparison', (_label, value) => {
		const binding = passingBinding();
		(binding.metrics as any).timeToInteractiveMs = { value, status: 'measured', method: 'test fixture' };
		const result = compareObservation(threshold(), { bindingObservation: binding });
		const entry = result.metricResults.find((r) => r.metric === 'timeToInteractiveMs');
		expect(entry?.verdict).toBe('FAIL');
		expect(result.verdict).toBe('FAIL');
	});
});

describe('compareObservation: boundary', () => {
	it('a metric exactly equal to its bar yields PASS — at-most, never strictly-less', () => {
		const binding = passingBinding();
		binding.metrics.timeToInteractiveMs = measured(5000);
		const result = compareObservation(threshold(), { bindingObservation: binding });
		const entry = result.metricResults.find((r) => r.metric === 'timeToInteractiveMs');
		expect(entry?.verdict).toBe('PASS');
		expect(result.verdict).toBe('PASS');
	});
});

describe('compareObservation: unreadable threshold', () => {
	it('throws rather than defaulting — no permissive fallback', () => {
		expect(() => compareObservation(null as any, { bindingObservation: passingBinding() })).toThrow();
		expect(() => compareObservation({} as any, { bindingObservation: passingBinding() })).toThrow();
	});
});

describe('compareObservation: extra keys', () => {
	it('an observation carrying extra keys the threshold does not name never influences the verdict', () => {
		const binding = passingBinding();
		(binding.metrics as any).someExtraMetricNotInThreshold = measured(999999);
		const result = compareObservation(threshold(), { bindingObservation: binding });
		expect(result.verdict).toBe('PASS');
		expect(result.metricResults).toHaveLength(4);
		expect(result.metricResults.some((r) => r.metric === 'someExtraMetricNotInThreshold')).toBe(false);
	});
});

describe('compareObservation: declared measurement failure', () => {
	it('yields a per-metric FAIL carrying the reason, overall FAIL, and does NOT throw', () => {
		const binding = passingBinding();
		binding.metrics.timeToInteractiveMs = {
			value: null,
			status: 'measurement-failed',
			method: 'live browser session',
			failureReason: 'seamReady exceeded its 60000ms deadline'
		} as any;
		let result: ReturnType<typeof compareObservation> | undefined;
		expect(() => {
			result = compareObservation(threshold(), { bindingObservation: binding });
		}).not.toThrow();
		expect(result!.verdict).toBe('FAIL');
		const entry = result!.metricResults.find((r) => r.metric === 'timeToInteractiveMs');
		expect(entry?.verdict).toBe('FAIL');
		expect(entry?.reason).toContain('seamReady exceeded its 60000ms deadline');
	});
});

describe('compareObservation: self-contradictory record', () => {
	it('a value of null claiming status "measured" yields FAIL with a reason naming the inconsistency', () => {
		const binding = passingBinding();
		binding.metrics.timeToInteractiveMs = { value: null, status: 'measured', method: 'test fixture' } as any;
		const result = compareObservation(threshold(), { bindingObservation: binding });
		const entry = result.metricResults.find((r) => r.metric === 'timeToInteractiveMs');
		expect(entry?.verdict).toBe('FAIL');
		expect(entry?.reason.length).toBeGreaterThan(0);
		expect(result.verdict).toBe('FAIL');
	});
});

describe('compareObservation: no bindingObservation', () => {
	it('throws — there is nothing to judge and inventing one is exactly the forbidden fallback', () => {
		expect(() => compareObservation(threshold(), {} as any)).toThrow();
		expect(() => compareObservation(threshold(), { bindingObservation: null } as any)).toThrow();
	});
});

describe('compareObservation: wrong corpus', () => {
	it('throws BEFORE any comparison when bindingObservation.corpus differs from the threshold, naming both values', () => {
		const binding = passingBinding();
		binding.corpus = { repo: 'seanb4t/codegraph-go', sha: 'deadbeef' };
		let threw: Error | undefined;
		try {
			compareObservation(threshold(), { bindingObservation: binding });
		} catch (err) {
			threw = err as Error;
		}
		expect(threw).toBeDefined();
		expect(threw!.message).toContain('seanb4t/codegraph-go');
		expect(threw!.message).toContain('google/guava');
	});
});

describe('compareObservation: additionalCorpora pass-through', () => {
	it('copies additionalCorpora verbatim and never judges it — a failing additional corpus does not affect the verdict', () => {
		const failingAdditional = {
			corpus: { repo: 'seanb4t/codegraph-go', sha: 'abc123' },
			metrics: {
				timeToInteractiveMs: measured(999999),
				panZoomFrameTimeMs: measured(999999),
				panZoomFrameTimeP95Ms: measured(999999),
				fileGraphResponseBytes: measured(999999999)
			}
		} as unknown as RawObservation;
		const result = compareObservation(threshold(), {
			bindingObservation: passingBinding(),
			additionalCorpora: [failingAdditional]
		});
		expect(result.verdict).toBe('PASS');
		expect(result.additionalCorpora).toEqual([failingAdditional]);
		expect(
			result.metricResults.every((r) => !JSON.stringify(r).includes('seanb4t/codegraph-go'))
		).toBe(true);
	});
});

describe('compareObservation: end-to-end dead-session path', () => {
	it('failedObservation -> mergeRawObservations -> compareObservation yields overall FAIL with a per-metric FAIL for every locked bar, throws nothing', () => {
		const t = threshold();
		const metricKeys = Object.keys(t.metrics);
		const raw = failedObservation('browser launch exceeded its 30000ms deadline', metricKeys, {
			role: 'binding',
			repo: CORPUS.repo,
			sha: CORPUS.sha
		});
		expect(Object.values(raw.metrics).every((m) => m.status === 'measurement-failed')).toBe(true);
		let result: ReturnType<typeof compareObservation> | undefined;
		expect(() => {
			const merged = mergeRawObservations([raw]);
			result = compareObservation(t, merged);
		}).not.toThrow();
		expect(result!.verdict).toBe('FAIL');
		expect(result!.metricResults).toHaveLength(metricKeys.length);
		for (const r of result!.metricResults) {
			expect(r.verdict).toBe('FAIL');
			expect(r.status).toBe('measurement-failed');
			expect(r.reason.length).toBeGreaterThan(0);
		}
	});
});

describe('mergeRawObservations: role split', () => {
	it('exactly one binding-role raw becomes bindingObservation; every additional-role raw becomes an additionalCorpora entry, in input order', () => {
		const binding = { role: 'binding', corpus: CORPUS, metrics: {} } as unknown as RawObservation;
		const additional1 = { role: 'additional', corpus: { repo: 'a', sha: '1' }, metrics: {} } as unknown as RawObservation;
		const additional2 = { role: 'additional', corpus: { repo: 'b', sha: '2' }, metrics: {} } as unknown as RawObservation;
		const merged = mergeRawObservations([binding, additional1, additional2]);
		expect(merged.bindingObservation).toBe(binding);
		expect(merged.additionalCorpora).toHaveLength(2);
		expect(merged.additionalCorpora[0]).toBe(additional1);
		expect(merged.additionalCorpora[1]).toBe(additional2);
	});
});

describe('parseArgs: output-path and note flags (05-08)', () => {
	it('an invocation using neither flag parses to exactly the keys it does today', () => {
		const parsed = parseArgs(['--binding', '/tmp/x.json']);
		expect(new Set(Object.keys(parsed))).toEqual(new Set(['binding', 'additional']));
	});

	it('an invocation using both flags carries them through by name', () => {
		const parsed = parseArgs(['--binding', '/tmp/x.json', '--out', '/tmp/y.json', '--note', 'hello']);
		expect(parsed.out).toBe('/tmp/y.json');
		expect(parsed.note).toBe('hello');
	});
});

describe('writeVerdict: default output path targeting (05-08)', () => {
	it('resolveOutputPath with no --out flag targets the fixed observations path under the given root', () => {
		const resolved = resolveOutputPath('/fake/repo/root', undefined);
		expect(resolved).toBe(path.join('/fake/repo/root', 'corpora', 'graph-render-observations.json'));
	});

	it('resolveOutputPath with an --out flag returns that path unchanged, never the default', () => {
		const resolved = resolveOutputPath('/fake/repo/root', '/scratch/custom.json');
		expect(resolved).toBe('/scratch/custom.json');
	});
});

describe('writeVerdict: written object key set (05-08)', () => {
	it('writing with no note produces EXACTLY the eleven top-level keys the comparator writes today', () => {
		const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'graph-verdict-test-'));
		const outputPath = path.join(dir, 'observation.json');
		try {
			const obj = writeVerdict({ threshold: threshold(), binding: passingBinding(), output: outputPath });
			expect(new Set(Object.keys(obj))).toEqual(
				new Set([
					'schemaVersion',
					'thresholdRef',
					'corpus',
					'bindingView',
					'bindingObservation',
					'additionalCorpora',
					'metricResults',
					'verdict',
					'recordedNonBinding',
					'generatedBy',
					'generatedAt'
				])
			);
		} finally {
			fs.rmSync(dir, { recursive: true, force: true });
		}
	});

	it('writing with a note carries it verbatim and gains exactly one key relative to the no-note case', () => {
		const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'graph-verdict-test-'));
		const outputPath = path.join(dir, 'observation.json');
		try {
			const withoutNote = writeVerdict({ threshold: threshold(), binding: passingBinding(), output: outputPath });
			const withNote = writeVerdict({
				threshold: threshold(),
				binding: passingBinding(),
				output: outputPath,
				note: 'a provenance note'
			});
			expect(withNote.note).toBe('a provenance note');
			expect(Object.keys(withNote)).toHaveLength(Object.keys(withoutNote).length + 1);
		} finally {
			fs.rmSync(dir, { recursive: true, force: true });
		}
	});
});

describe('writeVerdict: an explicit output path leaves an unrelated path untouched (05-08)', () => {
	it('writing to a custom path does not modify a separate pre-seeded file standing in for the default', () => {
		const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'graph-verdict-test-'));
		const standInDefaultPath = path.join(dir, 'default-observation.json');
		const customPath = path.join(dir, 'custom-observation.json');
		const seeded = JSON.stringify({ seeded: true });
		fs.writeFileSync(standInDefaultPath, seeded);
		try {
			writeVerdict({ threshold: threshold(), binding: passingBinding(), output: customPath });
			expect(fs.readFileSync(standInDefaultPath, 'utf8')).toBe(seeded);
			expect(fs.existsSync(customPath)).toBe(true);
		} finally {
			fs.rmSync(dir, { recursive: true, force: true });
		}
	});
});

describe('mergeRawObservations: binding count guard', () => {
	it('zero binding raws throws, naming the count found', () => {
		const onlyAdditional = [
			{ role: 'additional', corpus: { repo: 'a', sha: '1' }, metrics: {} } as unknown as RawObservation
		];
		expect(() => mergeRawObservations(onlyAdditional)).toThrow(/0/);
	});

	it('two binding raws throws, naming the count found', () => {
		const a = { role: 'binding', corpus: CORPUS, metrics: {} } as unknown as RawObservation;
		const b = { role: 'binding', corpus: CORPUS, metrics: {} } as unknown as RawObservation;
		expect(() => mergeRawObservations([a, b])).toThrow(/2/);
	});
});
