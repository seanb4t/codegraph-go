// health-page.test.ts — 04-05 Task 3: the three claims that look
// manual and are not (04-VALIDATION.md §Manual-Only Verifications
// explicitly removes all three from the manual list). Written and
// observed RED against Task 2's implementation before this file
// existed (module import failure — the file itself did not exist),
// then GREEN once every assertion below was satisfied by /health's
// existing markup with no component changes required.
import { render, screen, waitFor } from '@testing-library/svelte';
import { describe, expect, it, vi } from 'vitest';

import type { GetHealthResponse } from '$lib/gen/ui_pb';
import type { IndexStatus } from '$lib/status';

let currentGetHealthImpl: (
	req: unknown,
	opts?: { signal?: AbortSignal }
) => Promise<GetHealthResponse> = () =>
	Promise.reject(new Error('health-page.test.ts: no getHealth stub configured for this test'));

const getStatusSpy = vi.fn(() =>
	Promise.reject(new Error('health-page.test.ts: getStatus must never be called by this route'))
);

vi.doMock('$lib/client', () => ({
	uiClient: {
		getHealth: (req: unknown, opts?: { signal?: AbortSignal }) => currentGetHealthImpl(req, opts),
		getStatus: getStatusSpy
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
	status: Partial<IndexStatus> = {}
) {
	currentGetHealthImpl = getHealthImpl;
	return render(HealthPage, { context: new Map([['statusGate', fakeStatusGate(status)]]) });
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
	it('records exactly one getHealth invocation for one view open and never calls getStatus', async () => {
		let calls = 0;
		mountHealth(() => {
			calls += 1;
			return Promise.resolve(healthResponse());
		});

		await waitFor(() => expect(screen.getByTestId('health-freshness')).toBeInTheDocument());
		expect(calls).toBe(1);
		expect(getStatusSpy).not.toHaveBeenCalled();
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
