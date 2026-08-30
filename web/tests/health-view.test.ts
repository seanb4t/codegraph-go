// health-view.test.ts — 04-05 Task 1 step (b): health-view.ts's pure
// projections, written RED before the module existed. RED: 0/10 passing
// (module did not exist — every import failed). GREEN after
// web/src/lib/health-view.ts was created.
import { describe, it, expect } from 'vitest';

import { toCountRows, hasWorktreeMismatch, describeFreshness } from '$lib/health-view';
import type { GetHealthResponse } from '$lib/gen/ui_pb';
import type { IndexStatus } from '$lib/status';

function healthResponse(overrides: Partial<GetHealthResponse> = {}): GetHealthResponse {
	return {
		initialized: true,
		version: '7',
		fileCount: 10n,
		nodeCount: 100n,
		edgeCount: 50n,
		dbSizeBytes: 1024n,
		backend: 'pebble',
		filesByLanguage: {},
		languages: [],
		nodesByKind: {},
		edgesByKind: {},
		pendingChanges: undefined,
		indexHealth: undefined,
		worktreeMismatch: undefined,
		stale: false,
		commitSha: '',
		...overrides
	} as GetHealthResponse;
}

function status(overrides: Partial<IndexStatus> = {}): IndexStatus {
	return { verdict: 'ok', commit: 'known', commitSha: '', ...overrides };
}

describe('toCountRows: deterministic sort, no throw on missing data', () => {
	it('sorts by descending count then ascending key', () => {
		const rows = toCountRows({ go: 12n, ts: 3n, py: 3n });
		expect(rows).toEqual([
			{ key: 'go', count: 12 },
			{ key: 'py', count: 3 },
			{ key: 'ts', count: 3 }
		]);
	});

	it('returns an empty array for an empty map', () => {
		expect(toCountRows({})).toEqual([]);
	});

	it('returns an empty array for undefined without throwing', () => {
		expect(() => toCountRows(undefined)).not.toThrow();
		expect(toCountRows(undefined)).toEqual([]);
	});
});

describe('hasWorktreeMismatch: presence AND both roots non-empty', () => {
	it('is true when a mismatch is present with both roots populated', () => {
		const response = healthResponse({
			worktreeMismatch: { worktreeRoot: '/a/worktree', indexRoot: '/a/index' } as never
		});
		expect(hasWorktreeMismatch(response)).toBe(true);
	});

	it('is false when worktreeMismatch is absent (nil, the common/clean-tree path)', () => {
		expect(hasWorktreeMismatch(healthResponse({ worktreeMismatch: undefined }))).toBe(false);
	});

	it('is false for a present-but-blank mismatch — a false positive here would light the loudest warning in the app permanently', () => {
		const response = healthResponse({
			worktreeMismatch: { worktreeRoot: '', indexRoot: '' } as never
		});
		expect(hasWorktreeMismatch(response)).toBe(false);
	});
});

describe('describeFreshness: consumes the passed-in verdict, computes none of its own', () => {
	it('carries the verdict, commit SHA, schema version and re-index recommendation as separate named fields', () => {
		const response = healthResponse({
			version: '9',
			commitSha: 'c'.repeat(40),
			indexHealth: { reindexRecommended: true } as never
		});
		const view = describeFreshness(response, status({ verdict: 'stale' }));
		expect(view).toEqual({
			verdict: 'stale',
			commitSha: 'c'.repeat(40),
			schemaVersion: '9',
			reindexRecommended: true,
			snapshotAgreement: 'unknown'
		});
	});

	it('NEVER recomputes the verdict: an ok IndexStatus stays ok even when indexHealth.reindexRecommended is true', () => {
		const response = healthResponse({ indexHealth: { reindexRecommended: true } as never });
		const view = describeFreshness(response, status({ verdict: 'ok' }));
		expect(view.verdict).toBe('ok');
		expect(view.reindexRecommended).toBe(true);
	});

	it('snapshotAgreement is "agree" when the gate SHA and the response SHA are equal and both non-empty', () => {
		const sha = 'a'.repeat(40);
		const view = describeFreshness(
			healthResponse({ commitSha: sha }),
			status({ commitSha: sha })
		);
		expect(view.snapshotAgreement).toBe('agree');
	});

	it('snapshotAgreement is "differs" when the gate SHA and the response SHA are both non-empty and unequal', () => {
		const view = describeFreshness(
			healthResponse({ commitSha: 'b'.repeat(40) }),
			status({ commitSha: 'a'.repeat(40) })
		);
		expect(view.snapshotAgreement).toBe('differs');
	});

	it('snapshotAgreement is "unknown" when either SHA is empty', () => {
		const bothEmpty = describeFreshness(healthResponse({ commitSha: '' }), status({ commitSha: '' }));
		expect(bothEmpty.snapshotAgreement).toBe('unknown');

		const gateOnly = describeFreshness(
			healthResponse({ commitSha: 'a'.repeat(40) }),
			status({ commitSha: '' })
		);
		expect(gateOnly.snapshotAgreement).toBe('unknown');

		const healthOnly = describeFreshness(healthResponse({ commitSha: '' }), status({ commitSha: 'a'.repeat(40) }));
		expect(healthOnly.snapshotAgreement).toBe('unknown');
	});

	it('the comparison reads the passed-in IndexStatus.commitSha — never the two-member known/unknown presence flag, which is unequal to a real SHA on every healthy load', () => {
		// A comparison mistakenly reading `status.commit` ('known') against
		// a real 40-char SHA would report "differs" here even though both
		// snapshots are genuinely at the same commit — this is the exact
		// cycle-2 defect this test rejects.
		const sha = 'd'.repeat(40);
		const view = describeFreshness(healthResponse({ commitSha: sha }), status({ commit: 'known', commitSha: sha }));
		expect(view.snapshotAgreement).toBe('agree');
	});
});
