// health-view.test.ts — 04-05 Task 1 step (b): health-view.ts's pure
// projections, written RED before the module existed. RED: 0/10 passing
// (module did not exist — every import failed). GREEN after
// web/src/lib/health-view.ts was created.
//
// 10-04 Task 1 extends this file with the coverage projections and the
// bounded page walker (RED before health-view.ts carried any of the
// `toCoverageView`/`reasonLabel`/`reasonKeyOf`/`groupCoverageRows`/
// `fetchAllCoverageRows` exports — every import below failed until Task
// 1's GREEN commit).
import { describe, it, expect, vi } from 'vitest';
import { ConnectError, Code } from '@connectrpc/connect';

import {
	toCountRows,
	hasWorktreeMismatch,
	describeFreshness,
	toCoverageView,
	reasonLabel,
	reasonKeyOf,
	groupCoverageRows,
	fetchAllCoverageRows,
	COVERAGE_MAX_PAGES,
	type CoverageClient
} from '$lib/health-view';
import type { GetHealthResponse, CoverageRow, GetCoverageResponse } from '$lib/gen/ui_pb';
import { CoverageRowKind, ExclusionReason } from '$lib/gen/ui_pb';
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

function coverageRow(overrides: Partial<CoverageRow> = {}): CoverageRow {
	return {
		path: 'file.go',
		kind: CoverageRowKind.EXCLUDED,
		reason: ExclusionReason.UNSPECIFIED,
		detail: '',
		...overrides
	} as CoverageRow;
}

describe('toCoverageView: known/unknown discrimination, never a re-derived verdict', () => {
	it('is { known: false } when coverage is absent (a pre-Phase-10 graph)', () => {
		expect(toCoverageView(healthResponse({ coverage: undefined }))).toEqual({ known: false });
	});

	it('is { known: false } even when coverage.known is explicitly false with non-zero numbers present', () => {
		const view = toCoverageView(
			healthResponse({
				coverage: {
					known: false,
					discovered: 9n,
					indexed: 9n,
					excluded: 0n,
					extractionFailed: 0n,
					excludedByReason: {}
				} as never
			})
		);
		expect(view).toEqual({ known: false });
	});

	it('converts bigint counts to numbers and sums DIR_VENDOR + DIR_DOTPREFIX into directoryPrunes', () => {
		const view = toCoverageView(
			healthResponse({
				coverage: {
					known: true,
					discovered: 6n,
					indexed: 1n,
					excluded: 6n,
					extractionFailed: 1n,
					excludedByReason: {
						EXCLUSION_REASON_UNSUPPORTED_EXTENSION: 2n,
						EXCLUSION_REASON_DIR_DOTPREFIX: 1n,
						EXCLUSION_REASON_DIR_VENDOR: 1n,
						EXCLUSION_REASON_BUILD_TAG: 1n,
						EXCLUSION_REASON_SIZE_LIMIT: 1n
					}
				} as never
			})
		);
		expect(view.known).toBe(true);
		if (!view.known) throw new Error('unreachable');
		expect(view.discovered).toBe(6);
		expect(view.indexed).toBe(1);
		expect(view.excluded).toBe(6);
		expect(view.extractionFailed).toBe(1);
		expect(view.directoryPrunes).toBe(2);
		expect(view.byReason).toEqual([
			{ key: 'EXCLUSION_REASON_UNSUPPORTED_EXTENSION', count: 2 },
			{ key: 'EXCLUSION_REASON_BUILD_TAG', count: 1 },
			{ key: 'EXCLUSION_REASON_DIR_DOTPREFIX', count: 1 },
			{ key: 'EXCLUSION_REASON_DIR_VENDOR', count: 1 },
			{ key: 'EXCLUSION_REASON_SIZE_LIMIT', count: 1 }
		]);
	});

	it('directoryPrunes is 0 when neither directory reason is present', () => {
		const view = toCoverageView(
			healthResponse({
				coverage: {
					known: true,
					discovered: 1n,
					indexed: 1n,
					excluded: 0n,
					extractionFailed: 0n,
					excludedByReason: {}
				} as never
			})
		);
		expect(view.known).toBe(true);
		if (!view.known) throw new Error('unreachable');
		expect(view.directoryPrunes).toBe(0);
	});
});

describe('reasonLabel: the five known reasons get human labels, everything else names itself', () => {
	it('maps every one of the five full proto names to its human label', () => {
		expect(reasonLabel('EXCLUSION_REASON_DIR_VENDOR')).toBe('Vendored directory (pruned)');
		expect(reasonLabel('EXCLUSION_REASON_DIR_DOTPREFIX')).toBe('Dot-prefixed directory (pruned)');
		expect(reasonLabel('EXCLUSION_REASON_UNSUPPORTED_EXTENSION')).toBe('Unsupported extension');
		expect(reasonLabel('EXCLUSION_REASON_BUILD_TAG')).toBe('Excluded by build constraints');
		expect(reasonLabel('EXCLUSION_REASON_SIZE_LIMIT')).toBe('Over the size limit');
	});

	it('falls back to "Unknown reason (<key>)" for EXCLUSION_REASON_UNSPECIFIED, empty, and any unrecognised key', () => {
		expect(reasonLabel('EXCLUSION_REASON_UNSPECIFIED')).toBe('Unknown reason (EXCLUSION_REASON_UNSPECIFIED)');
		expect(reasonLabel('')).toBe('Unknown reason ()');
		expect(reasonLabel('SOME_FUTURE_REASON')).toBe('Unknown reason (SOME_FUTURE_REASON)');
	});
});

describe('reasonKeyOf: resolves through the generated enum descriptor, never a hand-written switch', () => {
	it('resolves every known ExclusionReason number to its full proto name', () => {
		expect(reasonKeyOf(ExclusionReason.DIR_VENDOR)).toBe('EXCLUSION_REASON_DIR_VENDOR');
		expect(reasonKeyOf(ExclusionReason.DIR_DOTPREFIX)).toBe('EXCLUSION_REASON_DIR_DOTPREFIX');
		expect(reasonKeyOf(ExclusionReason.UNSUPPORTED_EXTENSION)).toBe('EXCLUSION_REASON_UNSUPPORTED_EXTENSION');
		expect(reasonKeyOf(ExclusionReason.BUILD_TAG)).toBe('EXCLUSION_REASON_BUILD_TAG');
		expect(reasonKeyOf(ExclusionReason.SIZE_LIMIT)).toBe('EXCLUSION_REASON_SIZE_LIMIT');
	});

	it('resolves an unrecognised number (99) to EXCLUSION_REASON_UNSPECIFIED — a lookup, never a throw', () => {
		expect(() => reasonKeyOf(99)).not.toThrow();
		expect(reasonKeyOf(99)).toBe('EXCLUSION_REASON_UNSPECIFIED');
	});
});

describe('groupCoverageRows: extraction-failed group first, one group per reason, unrecognised reasons kept', () => {
	it('returns [] for an empty input', () => {
		expect(groupCoverageRows([])).toEqual([]);
	});

	it('puts the EXTRACTION_FAILED group first, keeps row order within a group, and every row is accounted for including an unrecognised reason', () => {
		const rows: CoverageRow[] = [
			coverageRow({ path: 'broken.py', kind: CoverageRowKind.EXTRACTION_FAILED, detail: 'boom' }),
			coverageRow({ path: 'go.mod', kind: CoverageRowKind.EXCLUDED, reason: ExclusionReason.UNSUPPORTED_EXTENSION }),
			coverageRow({ path: 'notes.md', kind: CoverageRowKind.EXCLUDED, reason: ExclusionReason.UNSUPPORTED_EXTENSION }),
			coverageRow({ path: 'weird.bin', kind: CoverageRowKind.EXCLUDED, reason: 99 as ExclusionReason })
		];

		const groups = groupCoverageRows(rows);

		expect(groups[0].key).toBe('EXTRACTION_FAILED');
		expect(groups[0].label).toBe('Extraction failed');
		expect(groups[0].rows.map((r) => r.path)).toEqual(['broken.py']);

		const unsupported = groups.find((g) => g.key === 'EXCLUSION_REASON_UNSUPPORTED_EXTENSION');
		expect(unsupported?.count).toBe(2);
		expect(unsupported?.rows.map((r) => r.path)).toEqual(['go.mod', 'notes.md']);

		const unknown = groups.find((g) => g.key === 'EXCLUSION_REASON_UNSPECIFIED');
		expect(unknown?.rows.map((r) => r.path)).toEqual(['weird.bin']);
		expect(unknown?.label).toBe('Unknown reason (EXCLUSION_REASON_UNSPECIFIED)');

		for (const group of groups) {
			expect(group.count).toBe(group.rows.length);
		}
		const totalRows = groups.reduce((sum, g) => sum + g.rows.length, 0);
		expect(totalRows).toBe(rows.length);
	});

	it('omits the EXTRACTION_FAILED group entirely when no row has that kind', () => {
		const rows: CoverageRow[] = [coverageRow({ reason: ExclusionReason.BUILD_TAG })];
		const groups = groupCoverageRows(rows);
		expect(groups.some((g) => g.key === 'EXTRACTION_FAILED')).toBe(false);
	});
});

describe('fetchAllCoverageRows: bounded page walker over GetCoverage', () => {
	function pagingClient(pages: GetCoverageResponse[]): { client: CoverageClient; calls: unknown[] } {
		const calls: unknown[] = [];
		let i = 0;
		const client: CoverageClient = {
			getCoverage: vi.fn((request: unknown, options?: unknown) => {
				calls.push({ request, options });
				const page = pages[i];
				i += 1;
				return Promise.resolve(page);
			})
		};
		return { client, calls };
	}

	it('pages until an empty next_page_token, concatenating rows in order and passing the signal through every call', async () => {
		const rowA = coverageRow({ path: 'a.go' });
		const rowB = coverageRow({ path: 'b.go' });
		const rowC = coverageRow({ path: 'c.go' });
		const { client, calls } = pagingClient([
			{ rows: [rowA], nextPageToken: 't1', known: true } as GetCoverageResponse,
			{ rows: [rowB], nextPageToken: 't2', known: true } as GetCoverageResponse,
			{ rows: [rowC], nextPageToken: '', known: true } as GetCoverageResponse
		]);
		const signal = new AbortController().signal;

		const result = await fetchAllCoverageRows(client, signal);

		expect(result.known).toBe(true);
		expect(result.rows.map((r) => r.path)).toEqual(['a.go', 'b.go', 'c.go']);
		expect(calls).toHaveLength(3);
		expect(calls[0]).toEqual({ request: { pageSize: 1000, pageToken: '' }, options: { signal } });
		expect(calls[1]).toEqual({ request: { pageSize: 1000, pageToken: 't1' }, options: { signal } });
		expect(calls[2]).toEqual({ request: { pageSize: 1000, pageToken: 't2' }, options: { signal } });
	});

	it('returns { known: false, rows: [] } after exactly one call when the first page reports known === false', async () => {
		const { client, calls } = pagingClient([
			{ rows: [], nextPageToken: '', known: false } as unknown as GetCoverageResponse
		]);

		const result = await fetchAllCoverageRows(client);

		expect(result).toEqual({ known: false, rows: [], incomplete: false });
		expect(calls).toHaveLength(1);
	});

	it('throws a "page limit" error after COVERAGE_MAX_PAGES calls when the token never empties', async () => {
		let calls = 0;
		const client: CoverageClient = {
			getCoverage: vi.fn(() => {
				calls += 1;
				return Promise.resolve({
					rows: [],
					nextPageToken: `t${calls}`,
					known: true
				} as unknown as GetCoverageResponse);
			})
		};

		await expect(fetchAllCoverageRows(client)).rejects.toThrow(/page limit/);
		expect(calls).toBe(COVERAGE_MAX_PAGES);
	});

	// WR-01: GetCoverage answers Code.Aborted when a page token's embedded
	// generation marker no longer matches the store (a Sync committed
	// between two page fetches). fetchAllCoverageRows retries the whole
	// walk from the first page exactly once before giving up.
	describe('WR-01: retries the whole walk once on Code.Aborted, then reports incomplete', () => {
		it('retries from the first page and succeeds when the retry does not abort', async () => {
			const rowA = coverageRow({ path: 'a.go' });
			let attempt = 0;
			let callsThisAttempt = 0;
			const client: CoverageClient = {
				getCoverage: vi.fn(async () => {
					callsThisAttempt += 1;
					if (attempt === 0 && callsThisAttempt === 2) {
						attempt += 1;
						callsThisAttempt = 0;
						throw new ConnectError('coverage: index changed', Code.Aborted);
					}
					if (callsThisAttempt === 1) {
						return { rows: [rowA], nextPageToken: 't1', known: true } as unknown as GetCoverageResponse;
					}
					return { rows: [], nextPageToken: '', known: true } as unknown as GetCoverageResponse;
				})
			};

			const result = await fetchAllCoverageRows(client);

			expect(result).toEqual({ known: true, rows: [rowA], incomplete: false });
		});

		it('reports { known: true, rows: [], incomplete: true } when the retry ALSO aborts', async () => {
			const client: CoverageClient = {
				getCoverage: vi.fn(async () => {
					throw new ConnectError('coverage: index changed', Code.Aborted);
				})
			};

			const result = await fetchAllCoverageRows(client);

			expect(result).toEqual({ known: true, rows: [], incomplete: true });
		});

		it('propagates a non-Aborted error unchanged, without retrying', async () => {
			const calls = vi.fn(async () => {
				throw new ConnectError('coverage: nope', Code.Internal);
			});
			const client: CoverageClient = { getCoverage: calls };

			await expect(fetchAllCoverageRows(client)).rejects.toMatchObject({ message: expect.stringContaining('nope') });
			expect(calls).toHaveBeenCalledTimes(1);
		});
	});
});
