// file-search.ts is the workbench multi-file picker's (WRK-02) file
// search controller — 04-06's second configuration of debounced-rpc.ts's
// extracted mechanism (search.ts's live path is the first). It exists as
// a SEPARATE module from search.ts, not a second copy of its mechanism:
// createSearchController is not parameterisable as it stands (it
// hard-codes two symbol-specific RPCs, search.ts:129-231), so this
// module reuses the genuinely-common part (debounced-rpc.ts) and owns
// only what is specific to a Files-only, arbitrary-depth query.
//
// Arbitrary depth (the 04-02 dependency, made observable): search.ts's
// own live Files query still builds a single-star `*term*` pattern,
// which only matches root-level paths — a scope choice for that call
// site, unchanged by 04-02 (see search.ts:168-179). This module builds
// a recursive `**/*term*` pattern instead, so the picker finds a file at
// any depth — the nested-result assertion in file-search.test.ts is the
// observable proof that 04-02's doublestar.Match fix reaches this
// caller.
//
// escapeGlobLiteral (cycle-1 review finding, HIGH): the typed term is
// USER TEXT interpolated into a glob pattern sent to Engine.Files.
// Unescaped, a typed `[` reaches the pre-scan sanity check
// (internal/query/files.go:148-149) and fails the whole query with
// `query: invalid pattern`, and a typed `*` silently matches paths the
// user never asked for. This function backslash-escapes every glob
// metacharacter in the user's text before it is wrapped in this
// module's own `**/*…*` structure — the wildcards are ours, the term is
// theirs. The escaping convention (`\` is doublestar's own escape
// character, so `\x` matches the literal `x`) is verified against the
// real matcher by 04-02's escaped_metacharacter_is_literal subtest
// (internal/query/files_status_test.go:842-858), not assumed from
// library documentation.
import { writable, type Readable } from 'svelte/store';
import type { MessageInitShape } from '@bufbuild/protobuf';
import type { FilesRequestSchema, FilesResponse, FileEntry } from '$lib/gen/ui_pb';
import { classifyRpcError, type RpcFailure } from '$lib/rpc-errors';
import { createDebouncedRpc } from '$lib/debounced-rpc';
import { SEARCH_DEBOUNCE_MS, SEARCH_MIN_CHARS } from '$lib/search';

// FilesClient is the minimal shape this controller needs from a
// UIService client — declared independently of the real generated
// client (search.ts's own SearchClient / browse-state.ts's
// NodeDetailClient convention) so a test stub can satisfy it with no
// Connect transport.
export interface FilesClient {
	files(
		request: MessageInitShape<typeof FilesRequestSchema>,
		options?: { signal?: AbortSignal }
	): Promise<FilesResponse>;
}

export type FileSearchState = {
	query: string;
	results: FileEntry[];
	failure: RpcFailure | undefined;
};

export interface FileSearchController extends Readable<FileSearchState> {
	setQuery(value: string): void;
	dispose(): void;
}

// escapeGlobLiteral prefixes a backslash to each of `\ * ? [ ] { }` — a
// single-pass replace, so a backslash in the term is escaped exactly
// once (as `\\`) rather than being re-scanned and double-processed by a
// later pass over the other metacharacters. Everything else passes
// through unchanged, including an already-ordinary term, which must
// come back byte-identical.
const GLOB_METACHARACTERS = /[\\*?[\]{}]/g;

export function escapeGlobLiteral(term: string): string {
	return term.replace(GLOB_METACHARACTERS, (c) => `\\${c}`);
}

export function createFileSearchController(client: FilesClient): FileSearchController {
	const state = writable<FileSearchState>({ query: '', results: [], failure: undefined });

	const debounced = createDebouncedRpc<FilesResponse>({
		debounceMs: SEARCH_DEBOUNCE_MS,
		minChars: SEARCH_MIN_CHARS,
		dispatch: (term, signal) =>
			client.files(
				{
					pattern: `**/*${escapeGlobLiteral(term)}*`,
					filter: '',
					dir: '',
					// depth: 0 — unlimited (FilesOptions.Depth's own doc
					// comment); this module adds no second copy of that
					// bound. limit-like narrowing is likewise left to the
					// server's own MaxLimit cap (Task 1 step (e)).
					depth: 0,
					format: 'flat'
				},
				{ signal }
			),
		onResult: (resp) => {
			// FilesResponse is a UNION (ui.proto's own doc comment): read
			// format before deciding which field is populated, never
			// assume "flat".
			const results = resp.format === 'flat' ? resp.files : [];
			state.update((s) => ({ ...s, results, failure: undefined }));
		},
		onFailure: (err) => {
			state.update((s) => ({ ...s, failure: classifyRpcError(err) }));
		},
		onBelowMinimum: () => {
			state.update((s) => ({ ...s, results: [], failure: undefined }));
		}
	});

	function setQuery(value: string): void {
		state.update((s) => ({ ...s, query: value }));
		debounced.setQuery(value);
	}

	function dispose(): void {
		debounced.dispose();
	}

	return {
		subscribe: state.subscribe,
		setQuery,
		dispose
	};
}
