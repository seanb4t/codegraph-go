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
import type { GetHealthRequestSchema, GetHealthResponse } from '$lib/gen/ui_pb';
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
