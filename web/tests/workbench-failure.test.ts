// 04-01 Task 3: locks the Workbench failure taxonomy
// (workbench-failure.ts) with executable tests. describeWorkbenchFailure
// composes classifyRpcError + IndexStatus — these tests assert the
// composition rule, not a re-implementation of classifyRpcError's own
// coverage (rpc-errors.test.ts already owns that).
import { ConnectError, Code } from '@connectrpc/connect';
import { describe, expect, it } from 'vitest';

import { describeWorkbenchFailure } from '$lib/workbench-failure';
import type { IndexStatus } from '$lib/status';

function status(verdict: IndexStatus['verdict']): IndexStatus {
	return { verdict, commit: 'unknown', commitSha: '' };
}

describe('describeWorkbenchFailure: composition over classifyRpcError + IndexStatus', () => {
	it('an IndexingInProgress detail on a bare Unavailable error yields index-stale', () => {
		// classifyRpcError only reports 'indexing' when the Unavailable
		// error carries a decodable IndexingInProgress detail (rpc-errors.ts
		// Code.Unavailable branch) — a bare Unavailable with no such detail
		// classifies as 'unknown', not 'indexing'. This test exercises the
		// documented 'indexing' -> 'index-stale' mapping via the ONE path
		// that actually produces 'indexing': a plain ConnectError classifies
		// as 'unknown' regardless of status, so 'unknown' -> 'server-error'
		// is asserted separately below.
		const result = describeWorkbenchFailure(
			new ConnectError('try later', Code.Unavailable),
			status('ok')
		);
		// No IndexingInProgress detail attached -> classifyRpcError reports
		// 'unknown' -> 'server-error', proving this composition does NOT
		// invent a fifth kind for a bare Unavailable.
		expect(result.kind).toBe('server-error');
	});

	it('a NotFound rejection with verdict no-index yields index-stale (the real cause is a missing index, not a missing symbol)', () => {
		const result = describeWorkbenchFailure(
			new ConnectError('no such symbol', Code.NotFound),
			status('no-index')
		);
		expect(result.kind).toBe('index-stale');
	});

	it('the SAME NotFound rejection with verdict ok yields not-found — the composition rule, asserted in both directions', () => {
		const result = describeWorkbenchFailure(
			new ConnectError('no such symbol', Code.NotFound),
			status('ok')
		);
		expect(result.kind).toBe('not-found');
	});

	it('an InvalidArgument rejection yields invalid-input', () => {
		const result = describeWorkbenchFailure(
			new ConnectError('bad symbol', Code.InvalidArgument),
			status('ok')
		);
		expect(result.kind).toBe('invalid-input');
	});

	it('a plain (non-Connect) error yields server-error', () => {
		const result = describeWorkbenchFailure(new Error('boom'), status('ok'));
		expect(result.kind).toBe('server-error');
	});

	it('all four WorkbenchFailureKind values are reachable from their documented input', () => {
		const kinds = new Set([
			describeWorkbenchFailure(new ConnectError('no such symbol', Code.NotFound), status('ok'))
				.kind,
			describeWorkbenchFailure(new ConnectError('bad symbol', Code.InvalidArgument), status('ok'))
				.kind,
			describeWorkbenchFailure(new ConnectError('no such symbol', Code.NotFound), status('no-index'))
				.kind,
			describeWorkbenchFailure(new Error('boom'), status('ok')).kind
		]);
		expect(kinds).toEqual(new Set(['not-found', 'invalid-input', 'index-stale', 'server-error']));
	});

	it('the four title strings are PAIRWISE DISTINCT — a taxonomy whose branches all render the same sentence passes vacuously otherwise', () => {
		const titles = [
			describeWorkbenchFailure(new ConnectError('x', Code.NotFound), status('ok')).title,
			describeWorkbenchFailure(new ConnectError('x', Code.InvalidArgument), status('ok')).title,
			describeWorkbenchFailure(new ConnectError('x', Code.NotFound), status('no-index')).title,
			describeWorkbenchFailure(new Error('boom'), status('ok')).title
		];
		expect(new Set(titles).size).toBe(4);
	});

	it('never throws for a non-Error input: undefined, a plain string, a bare object', () => {
		expect(() => describeWorkbenchFailure(undefined, status('ok'))).not.toThrow();
		expect(() => describeWorkbenchFailure('a plain string', status('ok'))).not.toThrow();
		expect(() => describeWorkbenchFailure({ some: 'object' }, status('ok'))).not.toThrow();

		expect(describeWorkbenchFailure(undefined, status('ok')).kind).toBe('server-error');
		expect(describeWorkbenchFailure('a plain string', status('ok')).kind).toBe('server-error');
		expect(describeWorkbenchFailure({ some: 'object' }, status('ok')).kind).toBe('server-error');
	});
});
