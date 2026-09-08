// workbench-failure.ts composes the Workbench's failure taxonomy
// (WRK-04 criterion 3, Claude's discretion per 04-CONTEXT.md). It CALLS
// classifyRpcError (rpc-errors.ts, Phase 3 D-04) and maps its four kinds
// onto a Workbench-specific vocabulary — it never re-inspects
// ConnectError itself and never introduces a second error classifier.
//
// The mapping folds in ONE piece of context classifyRpcError does not
// have: the current index status. A 'not-found' Callers/Callees/Impact/
// Affected rejection when the index verdict is 'no-index' means the
// symbol was never findable because there is no index at all — the real
// cause is a missing index, not a missing symbol — so it is reported as
// 'index-stale', matching the same failure kind an 'indexing' rejection
// gets. This is the one place that distinction is made; nowhere else in
// the Workbench re-derives it.
import { classifyRpcError } from './rpc-errors';
import type { IndexStatus } from './status';

export type WorkbenchFailureKind = 'not-found' | 'invalid-input' | 'index-stale' | 'server-error';

interface WorkbenchFailure {
	kind: WorkbenchFailureKind;
	title: string;
	detail: string;
}

// The four titles below are PAIRWISE DISTINCT — that distinctness is the
// whole point of WRK-04 criterion 3 (a taxonomy whose branches all
// render the same sentence would pass any "there IS a failure state"
// check vacuously) and is asserted by web/tests/workbench-failure.test.ts.
export function describeWorkbenchFailure(err: unknown, status: IndexStatus): WorkbenchFailure {
	const classified = classifyRpcError(err);

	switch (classified.kind) {
		case 'indexing':
			return {
				kind: 'index-stale',
				title: 'Index is being rebuilt',
				detail: classified.message
			};
		case 'not-found':
			if (status.verdict === 'no-index') {
				// Title text intentionally mirrors TrustVerdict.svelte's
				// 'no-index' copy ("No index was found for this
				// repository — there is nothing to trust yet. Run
				// `codegraph init` to create one.") — same verdict, same
				// vocabulary (D-04's spirit). 'Index is being rebuilt'
				// (the 'indexing' branch above) tells a developer to
				// WAIT; this state has no index at all and the correct
				// action is `codegraph init`, so it must not share that
				// title even though both currently share the
				// 'index-stale' kind for testid grouping.
				return {
					kind: 'index-stale',
					title: 'No index for this repository',
					detail:
						'No index exists yet for this repository, so the symbol could not be looked up. Run `codegraph init` to create one.'
				};
			}
			return {
				kind: 'not-found',
				title: 'Symbol not found',
				detail: classified.message
			};
		case 'invalid-input':
			return {
				kind: 'invalid-input',
				title: 'Invalid query',
				detail: classified.message
			};
		case 'unknown':
		default:
			return {
				kind: 'server-error',
				title: 'Something went wrong',
				detail: classified.message
			};
	}
}
