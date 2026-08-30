// Tests for graph-measure.mjs's pure exported helpers, with no browser
// (05-04 Task 1 behavior block). Pure-module vitest convention, mirroring
// browse-url.test.ts. Every browser operation the live path performs is
// INJECTED, which is what lets these tests drive a never-settling promise
// into the real session runner and read a written artifact back off disk,
// rather than testing a deadline helper in isolation and assuming the
// session uses it (review H-1 part 2, cycle 3).
import { describe, expect, it, vi } from 'vitest';
import * as fs from 'node:fs';
import * as os from 'node:os';
import * as path from 'node:path';
import {
	OPS_KEYS,
	failedObservation,
	readProtocol,
	sessionConfig,
	withDeadline,
	runSession
} from '../scripts/graph-measure.mjs';

const CORPUS = { repo: 'google/guava', sha: '94f39958baf7ad51ddf9c70e406ed6b188194daa' };

function baseProtocol(overrides: Record<string, number> = {}) {
	return {
		viewportWidth: 1600,
		viewportHeight: 1000,
		deviceScaleFactor: 1,
		coldReloads: 3,
		warmupMs: 500,
		sampleDurationMs: 3000,
		panStepPx: 40,
		panSteps: 60,
		zoomMin: 0.5,
		zoomMax: 2.0,
		zoomSteps: 20,
		launchTimeoutMs: 30000,
		navigationTimeoutMs: 30000,
		seamReadyTimeoutMs: 60000,
		samplerTimeoutMs: 30000,
		sessionTimeoutMs: 600000,
		...overrides
	};
}

function fullThreshold(protocolOverrides: Record<string, number> = {}) {
	return {
		corpus: CORPUS,
		metrics: {
			timeToInteractiveMs: { max: 5000 },
			panZoomFrameTimeMs: { max: 33.3 },
			panZoomFrameTimeP95Ms: { max: 100 },
			fileGraphResponseBytes: { max: 16777216 }
		},
		measurementProtocol: baseProtocol(protocolOverrides)
	};
}

function goodOps() {
	return {
		launch: async () => ({ identity: { name: 'Chromium', version: '999.0.0-test' } }),
		newPage: async () => ({}),
		navigate: async () => undefined,
		seamReady: async () => ({
			timeToInteractiveMs: 111,
			layoutDurationMs: 55,
			nodeCount: 10,
			edgeCount: 20
		}),
		sample: async () => ({ median: 16, p95: 30, count: 90 }),
		close: async () => undefined
	};
}

function neverSettle() {
	return new Promise(() => {
		/* never resolves, never rejects */
	});
}

function tmpOut(label: string) {
	return path.join(os.tmpdir(), `graph-measure-test-${label}-${Date.now()}-${Math.random().toString(36).slice(2)}.json`);
}

// ---------------------------------------------------------------------
// failedObservation
// ---------------------------------------------------------------------

describe('failedObservation', () => {
	it('returns one role-tagged raw observation with exactly one entry per key, each measurement-failed with no fabricated numbers (binding role)', () => {
		const raw = failedObservation('browser launch exceeded its 30000ms deadline', [
			'timeToInteractiveMs',
			'panZoomFrameTimeMs'
		], { role: 'binding', repo: CORPUS.repo, sha: CORPUS.sha });

		expect(raw.role).toBe('binding');
		expect(raw.corpus).toEqual(CORPUS);
		expect(raw.sessionError).toBe('browser launch exceeded its 30000ms deadline');
		expect(Object.keys(raw.metrics)).toEqual(['timeToInteractiveMs', 'panZoomFrameTimeMs']);
		for (const key of Object.keys(raw.metrics)) {
			expect(raw.metrics[key].value).toBeNull();
			expect(raw.metrics[key].status).toBe('measurement-failed');
			expect(raw.metrics[key].failureReason).toBe('browser launch exceeded its 30000ms deadline');
		}
		expect(raw.layoutDurationMs).toBeNull();
		expect(raw.frameSampleCount).toBeNull();
		expect(raw.nodeCount).toBeNull();
		expect(raw.edgeCount).toBeNull();
		expect(raw.browserIdentity).toBeNull();
	});

	it('a failed run of the second corpus yields a failed additional entry, never fabricating a binding one (additional role)', () => {
		const raw = failedObservation('sample exceeded its 30000ms deadline', ['panZoomFrameTimeMs'], {
			role: 'additional',
			repo: 'seanb4t/codegraph-go',
			sha: 'deadbeef'
		});
		expect(raw.role).toBe('additional');
		expect(raw.corpus).toEqual({ repo: 'seanb4t/codegraph-go', sha: 'deadbeef' });
		expect(raw.metrics.panZoomFrameTimeMs.status).toBe('measurement-failed');
	});
});

// ---------------------------------------------------------------------
// readProtocol
// ---------------------------------------------------------------------

describe('readProtocol', () => {
	it('returns the locked measurementProtocol block when all sixteen values are present and positive', () => {
		const t = fullThreshold();
		expect(readProtocol(t)).toEqual(t.measurementProtocol);
	});

	it('throws when measurementProtocol is absent — no defaults of its own', () => {
		const t = fullThreshold();
		delete (t as any).measurementProtocol;
		expect(() => readProtocol(t)).toThrow();
	});

	it('throws when a protocol value is missing', () => {
		const t = fullThreshold();
		delete (t.measurementProtocol as any).samplerTimeoutMs;
		expect(() => readProtocol(t)).toThrow(/samplerTimeoutMs/);
	});

	it('throws when a protocol value is non-positive', () => {
		const t = fullThreshold({ seamReadyTimeoutMs: 0 });
		expect(() => readProtocol(t)).toThrow(/seamReadyTimeoutMs/);
	});
});

// ---------------------------------------------------------------------
// sessionConfig — the sentinel test (review L-1, cycle 2)
// ---------------------------------------------------------------------

describe('sessionConfig', () => {
	it('derives every field from the protocol and nothing else — a sentinel protocol produces a config tracking it exactly', () => {
		const locked = baseProtocol();
		const sentinel = {
			viewportWidth: 7001,
			viewportHeight: 7003,
			deviceScaleFactor: 7,
			coldReloads: 7007,
			warmupMs: 7009,
			sampleDurationMs: 7013,
			panStepPx: 7019,
			panSteps: 7027,
			zoomMin: 7,
			zoomMax: 7033,
			zoomSteps: 7039,
			launchTimeoutMs: 7043,
			navigationTimeoutMs: 7057,
			seamReadyTimeoutMs: 7069,
			samplerTimeoutMs: 7079,
			sessionTimeoutMs: 7103
		};

		const lockedConfig = sessionConfig(locked);
		const sentinelConfig = sessionConfig(sentinel);

		for (const key of Object.keys(sentinel) as Array<keyof typeof sentinel>) {
			expect(sentinelConfig[key]).toBe(sentinel[key]);
			expect(sentinelConfig[key]).not.toBe(lockedConfig[key]);
		}
	});
});

// ---------------------------------------------------------------------
// withDeadline
// ---------------------------------------------------------------------

describe('withDeadline', () => {
	it('resolves with the underlying value when the promise settles inside the budget', async () => {
		await expect(withDeadline(Promise.resolve('ok'), 1000, 'test-op')).resolves.toBe('ok');
	});

	it('rejects with the underlying error when the promise rejects inside the budget', async () => {
		await expect(withDeadline(Promise.reject(new Error('boom')), 1000, 'test-op')).rejects.toThrow('boom');
	});

	it(
		'rejects with an Error naming the label and the budget when given a promise that never settles',
		async () => {
			await expect(withDeadline(neverSettle(), 25, 'launch')).rejects.toThrow(/launch/);
			await expect(withDeadline(neverSettle(), 25, 'launch')).rejects.toThrow(/25/);
		},
		2000
	);

	it('clears its timer on settlement — a successful run is not held open by its own guard', async () => {
		vi.useFakeTimers();
		try {
			await withDeadline(Promise.resolve('ok'), 100000, 'test-op');
			expect(vi.getTimerCount()).toBe(0);
		} finally {
			vi.useRealTimers();
		}
	});
});

// ---------------------------------------------------------------------
// OPS_KEYS — the closed enumeration (review H-1 part 2, cycle 3)
// ---------------------------------------------------------------------

describe('OPS_KEYS', () => {
	it('names exactly six operations: launch, newPage, navigate, seamReady, sample, close', () => {
		expect(OPS_KEYS).toHaveLength(6);
		expect([...OPS_KEYS]).toEqual(['launch', 'newPage', 'navigate', 'seamReady', 'sample', 'close']);
	});

	it('runSession rejects an ops object missing a member, naming the difference it found', async () => {
		// Deliberately malformed for this one call: the whole point of the
		// test is to prove the RUNTIME guard rejects a shape TypeScript
		// would otherwise catch at compile time.
		const ops = goodOps();
		delete (ops as any).close;
		await expect(
			runSession({ ops: ops as any, protocol: baseProtocol(), corpus: { role: 'binding', ...CORPUS }, outputPath: tmpOut('missing') })
		).rejects.toThrow(/close/);
	});

	it('runSession rejects an ops object carrying an extra member, naming the difference it found', async () => {
		const ops: Record<string, unknown> = { ...goodOps(), extraOp: async () => undefined };
		await expect(
			runSession({ ops: ops as any, protocol: baseProtocol(), corpus: { role: 'binding', ...CORPUS }, outputPath: tmpOut('extra') })
		).rejects.toThrow(/extraOp/);
	});
});

// ---------------------------------------------------------------------
// The never-settling sweep — one case per OPS_KEYS member, driven from
// the enumeration itself so a new member cannot be added without a case
// appearing (review H-1 part 2, cycle 3).
// ---------------------------------------------------------------------

const OP_BUDGET_KEY: Record<string, string> = {
	launch: 'launchTimeoutMs',
	newPage: 'launchTimeoutMs',
	navigate: 'navigationTimeoutMs',
	seamReady: 'seamReadyTimeoutMs',
	sample: 'samplerTimeoutMs',
	close: 'launchTimeoutMs'
};

describe('never-settling sweep (one case per OPS_KEYS member)', () => {
	const collectedLabels: string[] = [];

	for (const opKey of OPS_KEYS) {
		it(
			`a never-settling "${opKey}" still returns and leaves a written artifact behind`,
			async () => {
				const protocol = baseProtocol({ [OP_BUDGET_KEY[opKey]]: 25 });
				const ops: Record<string, () => Promise<unknown>> = { ...goodOps(), [opKey]: neverSettle };
				const outputPath = tmpOut(opKey);

				await runSession({ ops: ops as any, protocol, corpus: { role: 'binding', ...CORPUS }, outputPath });

				expect(fs.existsSync(outputPath)).toBe(true);
				const written = JSON.parse(fs.readFileSync(outputPath, 'utf8'));

				if (opKey === 'close') {
					// TEARDOWN is the case with a DIFFERENT correct answer: a
					// wedged close must not retroactively fail a completed
					// measurement.
					expect(written.sessionError).toBeNull();
					expect(typeof written.teardownError).toBe('string');
					expect(written.teardownError.length).toBeGreaterThan(0);
					expect(written.teardownError).toContain('close');
					collectedLabels.push('close');
					for (const key of Object.keys(written.metrics)) {
						const m = written.metrics[key];
						expect(m.status).toBe('measured');
						expect(typeof m.value).toBe('number');
						expect(Number.isFinite(m.value)).toBe(true);
					}
				} else {
					expect(typeof written.sessionError).toBe('string');
					expect(written.sessionError.length).toBeGreaterThan(0);
					expect(written.sessionError).toContain(opKey);
					for (const key of Object.keys(written.metrics)) {
						const m = written.metrics[key];
						expect(m.status).toBe('measurement-failed');
						expect(m.value).toBeNull();
						expect(typeof m.failureReason).toBe('string');
						expect(m.failureReason).toContain(opKey);
					}
					collectedLabels.push(opKey);
				}
			},
			2000
		);
	}

	it('the collected labels set-equal OPS_KEYS — six cases, six distinct labels, no case reporting another’s label', () => {
		expect(collectedLabels).toHaveLength(6);
		expect(new Set(collectedLabels)).toEqual(new Set(OPS_KEYS));
	});
});

// ---------------------------------------------------------------------
// Ordering: the write happens BEFORE cleanup is awaited (review H-1
// part 2, cycle 3) — asserted independently of the deadline mechanism.
// ---------------------------------------------------------------------

describe('write-before-cleanup ordering', () => {
	it('the never-settling close stub observes the file already on disk at the moment it is entered', async () => {
		const outputPath = tmpOut('ordering');
		let existedWhenCloseWasEntered: boolean | undefined;

		const ops = {
			...goodOps(),
			close: async () => {
				existedWhenCloseWasEntered = fs.existsSync(outputPath);
				await neverSettle();
			}
		};

		const protocol = baseProtocol({ launchTimeoutMs: 25 });
		await runSession({ ops, protocol, corpus: { role: 'binding', ...CORPUS }, outputPath });

		expect(existedWhenCloseWasEntered).toBe(true);
	}, 2000);
});
