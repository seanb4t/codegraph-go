// health-view.ts is /health's pure derivation module (04-05 Task 1,
// steps b-d): the map-to-rows projections for the three count tables,
// the worktree-mismatch presence check, and the freshness/snapshot-
// agreement view.
//
// This module computes NO verdict of its own (D-04). status.ts's
// verdict classifier is the ONE verdict authority in this tree; every
// function here is downstream of it, consuming an already-classified
// IndexStatus as a plain value. This file never imports that classifier
// and never reads any of GetHealthResponse's raw engine-availability
// flags to re-derive a health judgement — a second verdict function
// over one index is exactly the repudiation failure HLT-02 exists to
// prevent (T-04-17).
import type { MessageInitShape } from '@bufbuild/protobuf';
import { ConnectError, Code } from '@connectrpc/connect';
import { CoverageRowKind, ExclusionReasonSchema } from '$lib/gen/ui_pb';
import type {
	CoverageRow,
	GetCoverageRequestSchema,
	GetCoverageResponse,
	GetHealthRequestSchema,
	GetHealthResponse
} from '$lib/gen/ui_pb';
import type { IndexStatus, StatusVerdict } from './status';

// CountRow is the shared {key,count} row shape for all three of
// /health's count tables (per-language file counts, node counts by
// kind, edge counts by kind) — one shape, bound to the ONE generic
// DataTable shell (D-06) via three CountTable.svelte instantiations,
// never a second table implementation.
export interface CountRow {
	key: string;
	count: number;
}

// toCountRows projects a proto int64-valued map (bigint values) into
// CountRow[], sorted by descending count then ascending key — a
// deterministic default order so the rendered table has a stable shape
// even before the user sorts it. An undefined or empty map (a
// pre-upgrade or degraded response) renders an empty array, never a
// throw.
export function toCountRows(map: { [key: string]: bigint } | undefined): CountRow[] {
	if (!map) return [];
	return Object.entries(map)
		.map(([key, count]) => ({ key, count: Number(count) }))
		.sort((a, b) => b.count - a.count || a.key.localeCompare(b.key));
}

// hasWorktreeMismatch is true ONLY when worktreeMismatch is present AND
// both roots are non-empty. A present-but-blank mismatch reads as
// false: a false positive here would make HLT-03's warning — the
// loudest signal in the app — permanently on, training the user to
// ignore it (T-04-22's alarm-fatigue failure mode, one level up from
// the snapshot-disagreement case it mirrors).
export function hasWorktreeMismatch(response: GetHealthResponse): boolean {
	const mismatch = response.worktreeMismatch;
	return !!mismatch && mismatch.worktreeRoot !== '' && mismatch.indexRoot !== '';
}

// SnapshotAgreement names whether the two independently-fetched
// snapshots /health composes — the shared GetStatus gate's verdict and
// this route's own GetHealth response — were read at the same indexed
// commit. It is a comparison of two raw SHA strings, never a verdict:
// it adds no sixth StatusVerdict member and computes no staleness of
// its own (D-04 is untouched).
export type SnapshotAgreement = 'agree' | 'differs' | 'unknown';

// FreshnessView is describeFreshness's return shape: every fact the
// freshness block renders, as separate named fields, so the renderer
// never has to parse a formatted string.
export interface FreshnessView {
	// verdict is taken VERBATIM from the passed-in IndexStatus — never
	// recomputed here. The gate's verdict always wins (Task 2 block 3);
	// this field is how that rule is carried to the renderer.
	verdict: StatusVerdict;
	// commitSha is GetHealth's own commit SHA, '' when the server
	// recorded none — the wire's own representation of "unknown"
	// (ui.proto's commit_sha field, mirrored by status.ts's IndexStatus
	// widening). The renderer checks this for emptiness to show an
	// explicit unknown marker rather than a blank that reads as "no
	// problem" (T-04-21).
	commitSha: string;
	schemaVersion: string;
	reindexRecommended: boolean;
	// snapshotAgreement compares the GATE's commitSha (status.commitSha,
	// the field 04-05 Task 1 step (a) added to IndexStatus) against
	// GetHealth's own commitSha — NEVER the gate's known/unknown
	// presence flag, which is a two-member flag that is unequal to a
	// 40-character SHA on every healthy load and would light this
	// notice permanently (the exact cycle-2 defect).
	snapshotAgreement: SnapshotAgreement;
}

// describeFreshness takes the already-computed IndexStatus as a
// PARAMETER (D-04). It must not import or call the shared verdict
// classifier itself, and must not read any of the response's raw
// engine-availability flags to derive a verdict — there is one verdict
// authority and this module is downstream of it.
export function describeFreshness(response: GetHealthResponse, status: IndexStatus): FreshnessView {
	const gateSha = status.commitSha;
	const healthSha = response.commitSha;

	let snapshotAgreement: SnapshotAgreement;
	if (gateSha === '' || healthSha === '') {
		snapshotAgreement = 'unknown';
	} else if (gateSha === healthSha) {
		snapshotAgreement = 'agree';
	} else {
		snapshotAgreement = 'differs';
	}

	return {
		verdict: status.verdict,
		commitSha: healthSha,
		schemaVersion: response.version,
		reindexRecommended: response.indexHealth?.reindexRecommended ?? false,
		snapshotAgreement
	};
}

// HealthClient is the minimal shape the /health route needs from a
// UIService client — status.ts's StatusClient / browse-state.ts's
// NodeDetailClient convention, so a test stub can satisfy it with no
// Connect transport.
export interface HealthClient {
	getHealth(
		request: MessageInitShape<typeof GetHealthRequestSchema>,
		options?: { signal?: AbortSignal }
	): Promise<GetHealthResponse>;
}

// --- Coverage (Phase 10, plan 10-04, D-11) -------------------------------
//
// The functions below are pure projections of GetHealthResponse.coverage
// and GetCoverageResponse rows onto /health's Coverage section. Same
// discipline as everything above: no verdict is computed here (D-04),
// health-view.ts stays downstream of status.ts's ONE classifier, and
// these functions never re-derive a coverage number the server already
// counted — they only reshape what GetHealth/GetCoverage already sent.

// CoverageView discriminates on `known` (D-06's never-0/0 rule): a
// pre-Phase-10 graph, or a response whose `coverage` field is absent
// entirely, renders the SAME { known: false } shape — the renderer never
// sees an empty table it could mistake for "no gaps".
export interface CoverageViewUnknown {
	known: false;
}

export interface CoverageViewKnown {
	known: true;
	discovered: number;
	indexed: number;
	excluded: number;
	extractionFailed: number;
	// byReason is `excludedByReason` projected through the SAME
	// toCountRows sort every other count table on this page uses (count
	// desc, key asc) — one deterministic-order function, not a second one
	// for this table.
	byReason: CountRow[];
	// directoryPrunes is the sum of the two directory-level reasons
	// (DIR_VENDOR, DIR_DOTPREFIX) — entries whose contents were never
	// discovered at all, called out separately from per-file exclusions.
	directoryPrunes: number;
}

export type CoverageView = CoverageViewUnknown | CoverageViewKnown;

const DIRECTORY_REASON_KEYS = new Set(['EXCLUSION_REASON_DIR_VENDOR', 'EXCLUSION_REASON_DIR_DOTPREFIX']);

export function toCoverageView(response: GetHealthResponse): CoverageView {
	const coverage = response.coverage;
	if (!coverage || coverage.known === false) {
		return { known: false };
	}
	const byReason = toCountRows(coverage.excludedByReason);
	const directoryPrunes = byReason
		.filter((row) => DIRECTORY_REASON_KEYS.has(row.key))
		.reduce((sum, row) => sum + row.count, 0);
	return {
		known: true,
		discovered: Number(coverage.discovered),
		indexed: Number(coverage.indexed),
		excluded: Number(coverage.excluded),
		extractionFailed: Number(coverage.extractionFailed),
		byReason,
		directoryPrunes
	};
}

// REASON_LABELS is keyed by ExclusionReason's FULL generated proto names
// (the same strings the server uses as excluded_by_reason map keys and
// reasonKeyOf resolves numbers to) — never the bare TS enum member name.
export const REASON_LABELS: Record<string, string> = {
	EXCLUSION_REASON_DIR_VENDOR: 'Vendored directory (pruned)',
	EXCLUSION_REASON_DIR_DOTPREFIX: 'Dot-prefixed directory (pruned)',
	EXCLUSION_REASON_UNSUPPORTED_EXTENSION: 'Unsupported extension',
	EXCLUSION_REASON_BUILD_TAG: 'Excluded by build constraints',
	EXCLUSION_REASON_SIZE_LIMIT: 'Over the size limit'
};

// reasonLabel falls back to "Unknown reason (<key>)" for
// EXCLUSION_REASON_UNSPECIFIED, an empty key, or any future reason this
// build does not yet know how to label (T-10-08) — never a crash and
// never a silently dropped row.
export function reasonLabel(key: string): string {
	return REASON_LABELS[key] ?? `Unknown reason (${key})`;
}

// reasonKeyOf resolves a wire ExclusionReason number to its full proto
// name through the GENERATED enum descriptor's `values` array — a
// lookup, never a hand-written switch that could silently skew from the
// wire (D-08/D-09). An unrecognised number (a future reason this build
// predates) resolves to EXCLUSION_REASON_UNSPECIFIED rather than
// throwing.
export function reasonKeyOf(reason: number): string {
	return ExclusionReasonSchema.values.find((v) => v.number === reason)?.name ?? 'EXCLUSION_REASON_UNSPECIFIED';
}

// CoverageGroup is one expandable group in the Coverage section: either
// the EXTRACTION_FAILED group (rows from a File with a non-empty errors
// list) or one group per exclusion reason present in the row list.
export interface CoverageGroup {
	key: string;
	label: string;
	count: number;
	rows: CoverageRow[];
}

// groupCoverageRows groups a page-walked CoverageRow[] for display. The
// EXTRACTION_FAILED group, when any row has that kind, is placed FIRST —
// extraction failures are a data-loss signal distinct from a pre-
// extraction exclusion and must not be buried alphabetically among
// exclusion reasons. Remaining rows are grouped by reasonKeyOf(row.
// reason) (an unrecognised reason lands in its own "Unknown reason"
// group rather than being dropped — T-10-08), then reason groups are
// ordered by count desc, key asc — the SAME deterministic tie-break
// toCountRows uses elsewhere on this page. Rows keep their incoming
// (server) order within a group.
export function groupCoverageRows(rows: CoverageRow[]): CoverageGroup[] {
	const failedRows = rows.filter((row) => row.kind === CoverageRowKind.EXTRACTION_FAILED);
	const exclusionRows = rows.filter((row) => row.kind !== CoverageRowKind.EXTRACTION_FAILED);

	const groups: CoverageGroup[] = [];
	if (failedRows.length > 0) {
		groups.push({
			key: 'EXTRACTION_FAILED',
			label: 'Extraction failed',
			count: failedRows.length,
			rows: failedRows
		});
	}

	const byKey = new Map<string, CoverageRow[]>();
	for (const row of exclusionRows) {
		const key = reasonKeyOf(row.reason);
		const existing = byKey.get(key);
		if (existing) {
			existing.push(row);
		} else {
			byKey.set(key, [row]);
		}
	}

	const reasonGroups: CoverageGroup[] = Array.from(byKey.entries())
		.map(([key, groupRows]) => ({
			key,
			label: reasonLabel(key),
			count: groupRows.length,
			rows: groupRows
		}))
		.sort((a, b) => b.count - a.count || a.key.localeCompare(b.key));

	return [...groups, ...reasonGroups];
}

// CoverageClient mirrors HealthClient's minimal-client shape (D-19's test-
// stub-without-a-transport convention) for the one method /health's
// coverage rows fetch needs.
export interface CoverageClient {
	getCoverage(
		request: MessageInitShape<typeof GetCoverageRequestSchema>,
		options?: { signal?: AbortSignal }
	): Promise<GetCoverageResponse>;
}

// COVERAGE_PAGE_SIZE matches GetCoverageRequest's server-side clamp
// (D-10 verbatim: > 1000 is clamped to 1000) — requesting the maximum
// every page minimizes round trips for the bounded walk below.
export const COVERAGE_PAGE_SIZE = 1000;

// COVERAGE_MAX_PAGES bounds the client-side walk (T-10-03): a server
// that never returns an empty next_page_token (bug or hostile response)
// must not spin this loop forever. 100 pages * 1000 rows/page is
// 100,000 coverage-gap rows — far beyond any repo this tool targets.
export const COVERAGE_MAX_PAGES = 100;

export interface CoverageRowsResult {
	known: boolean;
	rows: CoverageRow[];
	// incomplete is true when the walk could not be completed even after
	// one retry-from-start (WR-01): GetCoverage's page token carries a
	// generation marker (the index's Meta.coverage_generation counter at
	// the moment the token was produced — a monotonic counter, not a
	// wall-clock timestamp, since 10-REVIEW.md iteration 2), and the
	// server answers Code.Aborted rather
	// than a page of rows when a Sync committed between two page
	// fetches — the honest signal that the walk's cross-page consistency
	// can no longer be trusted, replacing the OLD failure mode of
	// silently returning a page that might disagree with rows already
	// collected (CR-01/WR-01, 10-REVIEW.md). Always false when `known`
	// is false, since there was nothing to page in that case.
	incomplete: boolean;
}

// isRetryableCoverageAbort reports whether err is the specific
// Code.Aborted GetCoverage answers for a stale page-token generation
// (WR-01) — never a general Connect-error classifier (that job belongs to
// rpc-errors.ts's ONE translation, D-04): this is a narrow, single-code
// check local to fetchAllCoverageRows' own retry loop, not a second
// error-kind taxonomy.
function isRetryableCoverageAbort(err: unknown): boolean {
	return err instanceof ConnectError && err.code === Code.Aborted;
}

// fetchAllCoverageRows pages GetCoverage to bounded completion, one row
// list for the whole Coverage section. Grouping happens CLIENT-SIDE
// (groupCoverageRows) because the per-file row list is bounded PER PAGE,
// never per call (D-10's transport-cap rationale) — the server has no
// single rpc that returns every row pre-grouped.
//
// A first page reporting `known === false` short-circuits after exactly
// one call: an old graph's coverage rows are exactly as unknown as its
// summary counts (D-06), so there is nothing to page.
//
// WR-01: if a page fetch answers Code.Aborted (a Sync committed between
// two page fetches, detected server-side via the page token's generation
// marker), the WHOLE walk is retried from the first page exactly once — a
// fresh walk starts a fresh generation and is not itself more likely to
// trip the same check. If that retry ALSO aborts, the Coverage section
// renders an honest incomplete result (`incomplete: true`) instead of
// throwing or silently asserting a complete list. Any other error still
// propagates to the caller unchanged.
export function fetchAllCoverageRows(
	client: CoverageClient,
	signal?: AbortSignal
): Promise<CoverageRowsResult> {
	async function attempt(): Promise<CoverageRowsResult> {
		const rows: CoverageRow[] = [];
		let pageToken = '';
		for (let page = 0; page < COVERAGE_MAX_PAGES; page++) {
			const response = await client.getCoverage(
				{ pageSize: COVERAGE_PAGE_SIZE, pageToken },
				{ signal }
			);
			if (!response.known) {
				return { known: false, rows: [], incomplete: false };
			}
			rows.push(...response.rows);
			if (!response.nextPageToken) {
				return { known: true, rows, incomplete: false };
			}
			pageToken = response.nextPageToken;
		}
		throw new Error('coverage: page limit exceeded');
	}

	return attempt().catch((err: unknown) => {
		if (!isRetryableCoverageAbort(err)) {
			throw err;
		}
		return attempt().catch((retryErr: unknown) => {
			if (!isRetryableCoverageAbort(retryErr)) {
				throw retryErr;
			}
			return { known: true, rows: [], incomplete: true };
		});
	});
}
