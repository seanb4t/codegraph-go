// health-page.test.ts — 04-05 Task 3: the three claims that look
// manual and are not (04-VALIDATION.md §Manual-Only Verifications
// explicitly removes all three from the manual list). Written and
// observed RED against Task 2's implementation before this file
// existed (module import failure — the file itself did not exist),
// then GREEN once every assertion below was satisfied by /health's
// existing markup with no component changes required.
//
// 10-04 Task 2 extends this file with the Coverage section: the
// getCoverage mock plumbing below, and the describe blocks at the
// bottom of the file, were written RED before CoverageSection.svelte
// and +page.svelte's loadCoverageRows existed (missing `health-
// coverage*` test ids and zero `getCoverage` wiring).
import { render, screen, waitFor } from '@testing-library/svelte';
import { describe, expect, it, vi, beforeEach } from 'vitest';

import type { GetHealthResponse, GetCoverageResponse, CoverageRow } from '$lib/gen/ui_pb';
import { CoverageRowKind, ExclusionReason } from '$lib/gen/ui_pb';
import type { IndexStatus } from '$lib/status';
import type { LiveEvent } from '$lib/live/live-client';
import type { LiveStore } from '$lib/live/live-store';

let currentGetHealthImpl: (
	req: unknown,
	opts?: { signal?: AbortSignal }
) => Promise<GetHealthResponse> = () =>
	Promise.reject(new Error('health-page.test.ts: no getHealth stub configured for this test'));

const getStatusSpy = vi.fn(() =>
	Promise.reject(new Error('health-page.test.ts: getStatus must never be called by this route'))
);

let currentGetCoverageImpl: (
	req: unknown,
	opts?: { signal?: AbortSignal }
) => Promise<GetCoverageResponse> = () =>
	Promise.resolve({ rows: [], nextPageToken: '', known: true } as unknown as GetCoverageResponse);
let getCoverageCalls: Array<{ req: unknown; opts?: { signal?: AbortSignal } }> = [];

beforeEach(() => {
	currentGetCoverageImpl = () =>
		Promise.resolve({ rows: [], nextPageToken: '', known: true } as unknown as GetCoverageResponse);
	getCoverageCalls = [];
});

vi.doMock('$lib/client', () => ({
	uiClient: {
		getHealth: (req: unknown, opts?: { signal?: AbortSignal }) => currentGetHealthImpl(req, opts),
		getStatus: getStatusSpy,
		getCoverage: (req: unknown, opts?: { signal?: AbortSignal }) => {
			getCoverageCalls.push({ req, opts });
			return currentGetCoverageImpl(req, opts);
		}
	}
}));

const { default: HealthPage } = await import('../src/routes/health/+page.svelte');

function healthResponse(overrides: Partial<GetHealthResponse> = {}): GetHealthResponse {
	return {
		initialized: true,
		version: '7',
		fileCount: 10n,
		nodeCount: 100n,
		edgeCount: 50n,
		dbSizeBytes: 1024n,
		backend: 'pebble',
		filesByLanguage: { go: 5n, ts: 2n },
		languages: ['go', 'ts'],
		nodesByKind: { func: 10n, struct: 3n },
		edgesByKind: { calls: 20n, imports: 4n },
		pendingChanges: undefined,
		indexHealth: undefined,
		worktreeMismatch: undefined,
		stale: false,
		commitSha: 'a'.repeat(40),
		...overrides
	} as GetHealthResponse;
}

function fakeStatusGate(status: Partial<IndexStatus> = {}) {
	const full: IndexStatus = { verdict: 'ok', commit: 'known', commitSha: '', ...status };
	return {
		subscribe(run: (s: IndexStatus) => void) {
			run(full);
			return () => {};
		},
		notifyNavigated: () => {}
	};
}

function mountHealth(
	getHealthImpl: typeof currentGetHealthImpl,
	status: Partial<IndexStatus> = {},
	liveStore?: LiveStore
) {
	currentGetHealthImpl = getHealthImpl;
	const context = new Map<string, unknown>([['statusGate', fakeStatusGate(status)]]);
	if (liveStore) context.set('liveStore', liveStore);
	return render(HealthPage, { context });
}

/** fakeLiveStore mirrors web/tests/live-route-refetch.test.ts's own
 * harness: it delivers already-admitted events directly, since the
 * generation gate itself is live-store.ts's own tested concern. */
function fakeLiveStore(): LiveStore & { deliver: (live: LiveEvent) => void } {
	const listeners = new Set<(live: LiveEvent | null) => void>();
	let current: LiveEvent | null = null;
	return {
		subscribe(run) {
			listeners.add(run);
			run(current);
			return () => {
				listeners.delete(run);
			};
		},
		deliver(live) {
			current = live;
			for (const listener of listeners) listener(current);
		}
	};
}

function coverageRow(overrides: Partial<CoverageRow> = {}): CoverageRow {
	return {
		path: 'file.go',
		kind: CoverageRowKind.EXCLUDED,
		reason: ExclusionReason.UNSPECIFIED,
		detail: '',
		...overrides
	} as CoverageRow;
}

function knownCoverage() {
	return {
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
	} as never;
}

function coveragePages(): GetCoverageResponse[] {
	return [
		{
			rows: [
				coverageRow({
					path: 'broken.py',
					kind: CoverageRowKind.EXTRACTION_FAILED,
					detail: 'indexer: reading ./broken.py: no such file or directory'
				}),
				coverageRow({ path: 'go.mod', reason: ExclusionReason.UNSUPPORTED_EXTENSION }),
				coverageRow({ path: 'notes.md', reason: ExclusionReason.UNSUPPORTED_EXTENSION })
			],
			nextPageToken: 'p2',
			known: true
		} as unknown as GetCoverageResponse,
		{
			rows: [
				coverageRow({ path: 'tagged.go', reason: ExclusionReason.BUILD_TAG }),
				coverageRow({ path: 'vendor', reason: ExclusionReason.DIR_VENDOR }),
				coverageRow({ path: '.hidden', reason: ExclusionReason.DIR_DOTPREFIX }),
				coverageRow({ path: 'huge.go', reason: ExclusionReason.SIZE_LIMIT })
			],
			nextPageToken: '',
			known: true
		} as unknown as GetCoverageResponse
	];
}

function pagedGetCoverage(pages: GetCoverageResponse[]) {
	let i = 0;
	return () => {
		const page = pages[Math.min(i, pages.length - 1)];
		i += 1;
		return Promise.resolve(page);
	};
}

describe('HLT-01: completeness — every fact in one render, no navigation', () => {
	it('shows at least one per-language row, one node-kind row, one edge-kind row, the schema version and the commit SHA', async () => {
		mountHealth(() => Promise.resolve(healthResponse()));

		await waitFor(() => expect(screen.getByTestId('health-freshness')).toBeInTheDocument());

		expect(screen.getByText('go')).toBeInTheDocument();
		expect(screen.getByText('func')).toBeInTheDocument();
		expect(screen.getByText('calls')).toBeInTheDocument();
		expect(screen.getByTestId('health-schema-version').textContent).toBe('7');
		expect(screen.getByTestId('health-commit-sha').textContent).toBe('a'.repeat(40));
	});
});

describe('one call: exactly one getHealth, zero getStatus', () => {
	it('records exactly one getHealth invocation for one view open and never calls getStatus or getCoverage (fixture carries no coverage)', async () => {
		let calls = 0;
		mountHealth(() => {
			calls += 1;
			return Promise.resolve(healthResponse());
		});

		await waitFor(() => expect(screen.getByTestId('health-freshness')).toBeInTheDocument());
		expect(calls).toBe(1);
		expect(getStatusSpy).not.toHaveBeenCalled();
		expect(getCoverageCalls).toHaveLength(0);
	});
});

describe('HLT-02: the verdict precedes every element in the numeric region (DOM order, not textContent order)', () => {
	it('data-testid="health-verdict-stale" precedes the freshness block in document order', async () => {
		mountHealth(() => Promise.resolve(healthResponse()), { verdict: 'stale' });

		await waitFor(() => expect(screen.getByTestId('health-freshness')).toBeInTheDocument());

		const verdict = screen.getByTestId('health-verdict-stale');
		const freshness = screen.getByTestId('health-freshness');
		// DOCUMENT_POSITION_FOLLOWING (4) on `freshness` relative to
		// `verdict` means freshness comes AFTER verdict in the tree — the
		// DOM-order property HLT-02 requires, not a text-order inference.
		// eslint-disable-next-line no-bitwise
		expect(verdict.compareDocumentPosition(freshness) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
	});
});

describe('HLT-03: the worktree warning — presence, order, absence and loudness', () => {
	it('renders role="alert" with both roots named, and precedes the verdict element', async () => {
		mountHealth(() =>
			Promise.resolve(
				healthResponse({
					worktreeMismatch: { worktreeRoot: '/home/dev/worktree', indexRoot: '/home/dev/main' } as never
				})
			)
		);

		const warning = await screen.findByTestId('health-worktree-mismatch');
		expect(warning.getAttribute('role')).toBe('alert');
		expect(warning.textContent).toContain('/home/dev/worktree');
		expect(warning.textContent).toContain('/home/dev/main');

		const verdict = screen.getByTestId('health-verdict-ok');
		// eslint-disable-next-line no-bitwise
		expect(warning.compareDocumentPosition(verdict) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
	});

	it('renders NO warning element for a nil worktreeMismatch (the clean, common path) — both directions asserted, or a permanently-on warning would still pass a presence-only test', async () => {
		mountHealth(() => Promise.resolve(healthResponse({ worktreeMismatch: undefined })));

		await waitFor(() => expect(screen.getByTestId('health-freshness')).toBeInTheDocument());
		expect(screen.queryByTestId('health-worktree-mismatch')).toBeNull();
	});

	it('carries a class list distinct from TrustVerdict\'s, as the loudness signal', async () => {
		// getComputedStyle was tried first (per 04-VALIDATION.md's
		// technique) and found NOT to resolve a class-derived value in
		// this suite: this test file renders HealthPage directly, never
		// +layout.svelte, so app.css (and therefore the Tailwind utility
		// classes' actual color rules) is never loaded into jsdom —
		// getComputedStyle on either element resolves the browser's
		// unstyled default (transparent background, black border) for
		// BOTH the warning and an 'ok'-verdict TrustVerdict, which is
		// indistinguishable between them and would make the assertion
		// vacuous. Falling back to the class-list-differs check per the
		// plan's own documented fallback, recorded here rather than
		// silently dropped.
		mountHealth(() =>
			Promise.resolve(
				healthResponse({
					worktreeMismatch: { worktreeRoot: '/a', indexRoot: '/b' } as never
				})
			)
		);

		const warning = await screen.findByTestId('health-worktree-mismatch');
		const verdict = screen.getByTestId('health-verdict-ok');
		expect(warning.className).not.toBe(verdict.className);
		expect(warning.getAttribute('role')).toBe('alert');
		expect(verdict.getAttribute('role')).toBe('status');
	});
});

describe('failure path: a rejected GetHealth still shows the trust verdict', () => {
	it('renders a named failure state in place of the numbers, and the verdict still renders', async () => {
		const { ConnectError, Code } = await import('@connectrpc/connect');
		mountHealth(
			() => Promise.reject(new ConnectError('boom', Code.Internal)),
			{ verdict: 'ok' }
		);

		await waitFor(() => expect(screen.getByTestId('workbench-failure-server-error')).toBeInTheDocument());
		expect(screen.getByTestId('health-verdict-ok')).toBeInTheDocument();
		expect(screen.queryByTestId('health-freshness')).toBeNull();
	});
});

describe('snapshot disagreement: both directions asserted', () => {
	it('shows health-snapshot-differs AND still renders the verdict and the numbers when the gate SHA and the response SHA disagree', async () => {
		mountHealth(
			() => Promise.resolve(healthResponse({ commitSha: 'b'.repeat(40) })),
			{ verdict: 'ok', commitSha: 'a'.repeat(40) }
		);

		await waitFor(() => expect(screen.getByTestId('health-snapshot-differs')).toBeInTheDocument());
		expect(screen.getByTestId('health-verdict-ok')).toBeInTheDocument();
		expect(screen.getByTestId('health-freshness')).toBeInTheDocument();
		expect(screen.getByTestId('health-commit-sha').textContent).toBe('b'.repeat(40));
	});

	it('shows NO health-snapshot-differs element when the gate SHA and the response SHA agree — the absent direction, which catches a comparison reading the wrong field', async () => {
		const sha = 'c'.repeat(40);
		mountHealth(() => Promise.resolve(healthResponse({ commitSha: sha })), {
			verdict: 'ok',
			commitSha: sha
		});

		await waitFor(() => expect(screen.getByTestId('health-freshness')).toBeInTheDocument());
		expect(screen.queryByTestId('health-snapshot-differs')).toBeNull();
	});
});

describe('CR-02: the fabricated always-zero "Pending changes" tally is never rendered', () => {
	it('renders no pending-changes element or text even when the response carries a populated pendingChanges value — internal/query/status.go never assigns this field for real, so displaying it (even non-zero) would present engine-fabricated data as a live trust signal', async () => {
		mountHealth(() =>
			Promise.resolve(
				healthResponse({
					pendingChanges: { added: 3, modified: 2, removed: 1 } as never
				})
			)
		);

		await waitFor(() => expect(screen.getByTestId('health-freshness')).toBeInTheDocument());
		expect(screen.queryByTestId('health-pending-changes')).toBeNull();
		expect(screen.queryByText(/Pending changes/)).toBeNull();
	});
});

describe('Coverage section (D-11/HLT-04): unknown state is first-class, never an empty table', () => {
	it('renders health-coverage-unknown with no counts and zero getCoverage calls when coverage is absent', async () => {
		mountHealth(() => Promise.resolve(healthResponse({ coverage: undefined })));

		const unknown = await screen.findByTestId('health-coverage-unknown');
		expect(unknown.textContent).toContain('Coverage unknown — re-index to record it');
		expect(screen.queryByTestId('health-coverage-counts')).toBeNull();
		expect(screen.queryByTestId('health-coverage-group-EXTRACTION_FAILED')).toBeNull();
		expect(getCoverageCalls).toHaveLength(0);
	});

	it('renders health-coverage-unknown with no counts and zero getCoverage calls when coverage.known is false', async () => {
		mountHealth(() =>
			Promise.resolve(healthResponse({ coverage: { known: false } as never }))
		);

		const unknown = await screen.findByTestId('health-coverage-unknown');
		expect(unknown.textContent).toContain('Coverage unknown — re-index to record it');
		expect(screen.queryByTestId('health-coverage-counts')).toBeNull();
		expect(getCoverageCalls).toHaveLength(0);
	});
});

describe('Coverage section: known state — counts, prunes, groups, distinct failed rows', () => {
	it('renders the exact counts line and the pruned-directory note', async () => {
		currentGetCoverageImpl = pagedGetCoverage(coveragePages());
		mountHealth(() => Promise.resolve(healthResponse({ coverage: knownCoverage() })));

		await waitFor(() =>
			expect(screen.getByTestId('health-coverage-counts').textContent).toBe(
				'6 discovered · 1 indexed · 6 excluded · 1 extraction failures'
			)
		);
		expect(screen.getByTestId('health-coverage-prunes').textContent).toContain('2');
	});

	it('pages getCoverage exactly twice with the expected token sequence', async () => {
		currentGetCoverageImpl = pagedGetCoverage(coveragePages());
		mountHealth(() => Promise.resolve(healthResponse({ coverage: knownCoverage() })));

		await waitFor(() => expect(getCoverageCalls).toHaveLength(2));
		expect((getCoverageCalls[0].req as { pageToken: string }).pageToken).toBe('');
		expect((getCoverageCalls[1].req as { pageToken: string }).pageToken).toBe('p2');
	});

	it('groups rows with the EXTRACTION_FAILED group first in DOM order, 7 rows total, and the unsupported-extension group opened shows its two files', async () => {
		currentGetCoverageImpl = pagedGetCoverage(coveragePages());
		mountHealth(() => Promise.resolve(healthResponse({ coverage: knownCoverage() })));

		const failedGroup = await screen.findByTestId('health-coverage-group-EXTRACTION_FAILED');
		const unsupportedGroup = await screen.findByTestId(
			'health-coverage-group-EXCLUSION_REASON_UNSUPPORTED_EXTENSION'
		);
		// eslint-disable-next-line no-bitwise
		expect(
			failedGroup.compareDocumentPosition(unsupportedGroup) & Node.DOCUMENT_POSITION_FOLLOWING
		).toBeTruthy();

		expect(unsupportedGroup.textContent).toContain('2');
		expect(unsupportedGroup.textContent).toContain('go.mod');
		expect(unsupportedGroup.textContent).toContain('notes.md');

		await waitFor(() =>
			expect(screen.getAllByTestId(/^health-coverage-row/).length).toBe(7)
		);
	});

	it('renders the broken.py row as health-coverage-row-failed with a class list distinct from an exclusion row, text starting with "extraction failed:"', async () => {
		currentGetCoverageImpl = pagedGetCoverage(coveragePages());
		mountHealth(() => Promise.resolve(healthResponse({ coverage: knownCoverage() })));

		const failedRow = await screen.findByTestId('health-coverage-row-failed');
		expect(failedRow.textContent?.trim().startsWith('extraction failed:')).toBe(true);

		const exclusionRow = (await screen.findAllByTestId('health-coverage-row'))[0];
		expect(failedRow.className).not.toBe(exclusionRow.className);
	});

	it('renders a markup-bearing path as literal text — no img element is created (T-10-01)', async () => {
		const evilPath = '<img src=x onerror="alert(1)">';
		currentGetCoverageImpl = pagedGetCoverage([
			{
				rows: [coverageRow({ path: evilPath, reason: ExclusionReason.BUILD_TAG })],
				nextPageToken: '',
				known: true
			} as unknown as GetCoverageResponse
		]);
		const { container } = mountHealth(() =>
			Promise.resolve(healthResponse({ coverage: knownCoverage() }))
		);

		await waitFor(() => expect(screen.getByTestId('health-coverage-row')).toBeInTheDocument());
		expect(screen.getByText(evilPath)).toBeInTheDocument();
		expect(container.querySelector('img')).toBeNull();
	});

	it('offers no action control: zero buttons, zero links, zero [onclick] attributes within the section', async () => {
		currentGetCoverageImpl = pagedGetCoverage(coveragePages());
		mountHealth(() => Promise.resolve(healthResponse({ coverage: knownCoverage() })));

		const section = await screen.findByTestId('health-coverage');
		await waitFor(() =>
			expect(section.querySelectorAll('[data-testid^="health-coverage-group-"]').length).toBe(6)
		);
		expect(section.querySelectorAll('button').length).toBe(0);
		expect(section.querySelectorAll('a').length).toBe(0);
		expect(section.querySelectorAll('[onclick]').length).toBe(0);
	});

	it('renders health-coverage-rows-failed on a rejected getCoverage while the counts line (from GetHealth) still renders', async () => {
		currentGetCoverageImpl = () => Promise.reject(new Error('boom'));
		mountHealth(() => Promise.resolve(healthResponse({ coverage: knownCoverage() })));

		await waitFor(() => expect(screen.getByTestId('health-coverage-counts')).toBeInTheDocument());
		await waitFor(() => expect(screen.getByTestId('health-coverage-rows-failed')).toBeInTheDocument());
	});

	it('re-issues getCoverage on a live-triggered getHealth refetch', async () => {
		currentGetCoverageImpl = pagedGetCoverage(coveragePages());
		let healthCalls = 0;
		const live = fakeLiveStore();
		mountHealth(
			() => {
				healthCalls += 1;
				return Promise.resolve(healthResponse({ coverage: knownCoverage() }));
			},
			{},
			live
		);

		await waitFor(() => expect(getCoverageCalls).toHaveLength(2));

		live.deliver({
			event: {
				generation: 5n,
				initialized: true,
				stale: false,
				storeExists: true,
				indexingInProgress: false,
				commitSha: ''
			} as never,
			epoch: 1
		});

		await waitFor(() => expect(healthCalls).toBe(2));
		await waitFor(() => expect(getCoverageCalls.length).toBeGreaterThan(2));
	});
});
