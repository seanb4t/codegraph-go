// Edge coverage for rpc-errors.ts's classifyRpcError (D-04). Pure-TS: no
// DOM, no SvelteKit runtime. Fixtures use the REAL ConnectError type from
// @connectrpc/connect and a real generated IndexingInProgress detail
// message — a hand-shaped plain object would test the test, not the
// mapper.
import { describe, expect, it } from 'vitest';
import { ConnectError, Code } from '@connectrpc/connect';
import { IndexingInProgressSchema } from '$lib/gen/ui_pb';
import { classifyRpcError } from '$lib/rpc-errors';

describe('rpc-errors: each kind is produced for its own input, and the four are mutually distinct', () => {
	it('CodeNotFound classifies as not-found', () => {
		const failure = classifyRpcError(new ConnectError('no such symbol', Code.NotFound));
		expect(failure.kind).toBe('not-found');
		expect(failure.message).toContain('no such symbol');
	});

	it('CodeInvalidArgument classifies as invalid-input', () => {
		const failure = classifyRpcError(new ConnectError('bad path', Code.InvalidArgument));
		expect(failure.kind).toBe('invalid-input');
		expect(failure.message).toContain('bad path');
	});

	it('CodeUnavailable carrying a real IndexingInProgress detail classifies as indexing — distinct from a bare CodeUnavailable', () => {
		const withDetail = new ConnectError('unavailable', Code.Unavailable, undefined, [
			{
				desc: IndexingInProgressSchema,
				value: { message: 'The index is being rebuilt. Please retry shortly.' }
			}
		]);
		const indexing = classifyRpcError(withDetail);
		expect(indexing.kind).toBe('indexing');
		expect(indexing.message).toContain('index is being rebuilt');

		const bare = new ConnectError('try later', Code.Unavailable);
		const bareFailure = classifyRpcError(bare);
		expect(bareFailure.kind).toBe('unknown');
		expect(bareFailure.kind).not.toBe(indexing.kind);
	});

	it('all four kinds are mutually distinct for their own respective inputs', () => {
		const notFound = classifyRpcError(new ConnectError('a', Code.NotFound));
		const invalidInput = classifyRpcError(new ConnectError('b', Code.InvalidArgument));
		const indexing = classifyRpcError(
			new ConnectError('c', Code.Unavailable, undefined, [
				{ desc: IndexingInProgressSchema, value: { message: 'rebuilding' } }
			])
		);
		const unknown = classifyRpcError(new ConnectError('d', Code.Internal));

		const kinds = [notFound.kind, invalidInput.kind, indexing.kind, unknown.kind];
		expect(new Set(kinds).size).toBe(4);
	});
});

describe('rpc-errors: never throws, even on a non-Connect value', () => {
	it('a plain Error yields unknown', () => {
		expect(() => classifyRpcError(new Error('plain error'))).not.toThrow();
		const failure = classifyRpcError(new Error('plain error'));
		expect(failure.kind).toBe('unknown');
		expect(failure.message).toBe('plain error');
	});

	it('a bare string yields unknown', () => {
		expect(() => classifyRpcError('just a string')).not.toThrow();
		const failure = classifyRpcError('just a string');
		expect(failure.kind).toBe('unknown');
	});

	it('undefined yields unknown', () => {
		expect(() => classifyRpcError(undefined)).not.toThrow();
		const failure = classifyRpcError(undefined);
		expect(failure.kind).toBe('unknown');
	});
});
