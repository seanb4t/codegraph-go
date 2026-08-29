// 03-04 Task 1 (the phase's tracer): the whole "open a file and read its
// highlighted source" slice, driven end to end through the real
// loadBrowseTarget + SourcePane modules with a stub client — no SvelteKit
// runtime, no real RPC transport. This file is written and observed RED
// BEFORE any of browse-url.ts / rpc-errors.ts / highlight.ts /
// browse-state.ts / SourcePane.svelte exist (recorded in the SUMMARY),
// then made GREEN by building those modules.
import { ConnectError, Code } from '@connectrpc/connect';
import { render, screen } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';

import { loadBrowseTarget, type NodeDetailClient } from '$lib/browse-state';
import { HIGHLIGHT_COVERAGE, HLJS_ALIAS_COVERAGE } from '$lib/highlight';
import SourcePane from '$lib/components/browse/SourcePane.svelte';
import type { GetNodeDetailResponse } from '$lib/gen/ui_pb';

const GO_SOURCE = 'package main\n\nfunc main() {}\n';

// NodeDetailMode.FILE = 1 (ui.proto NodeDetailMode enum, verified against
// web/src/lib/gen/ui_pb.ts). Using the literal rather than importing the
// enum keeps this stub decoupled from the generated module's import
// surface — the stub only needs to look like what the real RPC returns.
function fileModeResponse(path: string, source: string) {
	const bytes = new TextEncoder().encode(source);
	return {
		mode: 1,
		path,
		node: undefined,
		calls: [],
		calledBy: [],
		symbol: '',
		definitions: [],
		totalCandidates: 0,
		source: {
			content: bytes,
			truncated: false,
			totalLines: source.split('\n').length - (source.endsWith('\n') ? 1 : 0),
			totalBytes: bytes.length,
			returnedLines: source.split('\n').length - (source.endsWith('\n') ? 1 : 0),
			returnedBytes: bytes.length
		}
	};
}

// Stub clients satisfy NodeDetailClient exactly — this is the seam that
// makes loadBrowseTarget testable without a real Connect transport
// (browse-state.ts's own doc comment). Responses are cast to
// GetNodeDetailResponse because the plain-object shapes below carry only
// the fields this tracer path reads; a real generated message carries
// additional prototype/Symbol machinery this test does not need.
function stubClient(response: unknown): NodeDetailClient {
	return {
		getNodeDetail: () => Promise.resolve(response as GetNodeDetailResponse)
	};
}

function stubFailingClient(error: unknown): NodeDetailClient {
	return {
		getNodeDetail: () => Promise.reject(error)
	};
}

describe('browse tracer: open a file by URL and read its highlighted verbatim source', () => {
	it('parseBrowseParams -> loadBrowseTarget -> SourcePane renders the file text with highlight markup', async () => {
		const { parseBrowseParams } = await import('$lib/browse-url');
		const params = parseBrowseParams(new URLSearchParams('file=internal/query/node.go&line=42'));

		const state = await loadBrowseTarget(
			params,
			stubClient(fileModeResponse('internal/query/node.go', GO_SOURCE))
		);

		expect(state.kind).toBe('file');

		render(SourcePane, { props: { state } });

		const pane = screen.getByTestId('browse-source');
		expect(pane.textContent).toContain('func main()');
		expect(pane.querySelector('[class^="hljs-"], [class*=" hljs-"]')).not.toBeNull();
	});

	it('parseBrowseParams on an empty query string yields the idle state, not an error', async () => {
		const { parseBrowseParams } = await import('$lib/browse-url');
		const params = parseBrowseParams(new URLSearchParams(''));

		const state = await loadBrowseTarget(params, stubClient(fileModeResponse('unused', '')));

		expect(state.kind).toBe('idle');

		render(SourcePane, { props: { state } });
		expect(screen.getByTestId('browse-idle')).toBeInTheDocument();
	});

	it('classifyRpcError maps a not-found ConnectError, and loadBrowseTarget never throws on rejection', async () => {
		const { classifyRpcError } = await import('$lib/rpc-errors');
		const notFound = classifyRpcError(new ConnectError('no such symbol', Code.NotFound));
		expect(notFound.kind).toBe('not-found');

		const invalidInput = classifyRpcError(new ConnectError('bad path', Code.InvalidArgument));
		expect(invalidInput.kind).toBe('invalid-input');

		const bareUnavailable = classifyRpcError(new ConnectError('try later', Code.Unavailable));
		expect(bareUnavailable.kind).toBe('unknown');

		const notConnect = classifyRpcError(new Error('plain error'));
		expect(notConnect.kind).toBe('unknown');

		const state = await loadBrowseTarget(
			{ file: 'missing.go', unknown: [] },
			stubFailingClient(new ConnectError('no such file', Code.NotFound))
		);
		expect(state.kind).toBe('failed');
		if (state.kind === 'failed') {
			expect(state.failure.kind).toBe('not-found');
		}

		render(SourcePane, { props: { state } });
		expect(screen.getByTestId('browse-failed-not-found')).toBeInTheDocument();
	});

	it('highlightSource escapes unregistered languages as plaintext, never auto-detects', async () => {
		const { highlightSource } = await import('$lib/highlight');
		const highlighted = highlightSource('func main() {}', 'go');
		expect(highlighted).toContain('class="hljs-');
		expect(highlighted).not.toContain('<script>');

		const escaped = highlightSource('<script>alert(1)</script>', 'not-a-real-language');
		expect(escaped).not.toContain('<script>');
		expect(escaped).toContain('&lt;script&gt;');
	});

	it('highlightSource escapes hostile input on the REGISTERED-language path too (WR-04)', async () => {
		// The two assertions above never exercise escaping on the
		// registered-language branch: 'func main() {}' contains no '<', so
		// `not.toContain('<script>')` against it is vacuously true
		// regardless of what highlightSource does, and the genuine
		// escaping assertion above covers only the UNREGISTERED-language
		// fallback (escapeHtml) — the branch highlight.ts's own doc
		// comment says "should be unreachable in a correct tree".
		// SourcePane.svelte's {@html} sites render hljs.highlight()'s
		// output for a REGISTERED language, which had zero escaping
		// coverage before this test. Anchor the negative to an input that
		// CAN contain the forbidden string, on that path.
		const { highlightSource } = await import('$lib/highlight');
		const hostile = highlightSource('func main() { /* <script>alert(1)</script> */ }', 'go');
		expect(hostile).toContain('&lt;script&gt;'); // positive: escaping happened
		expect(hostile).not.toContain('<script>'); // negative: now non-vacuous
	});

	it('HIGHLIGHT_COVERAGE is exactly the registered module set plus the declared alias-map keys (no drift)', async () => {
		// Mirrors highlight.ts's own registerLanguage() argument list — kept
		// as an independent, hand-written list (not imported from
		// highlight.ts) so this assertion can actually catch highlight.ts
		// drifting from its own registration calls, per the plan's D-19
		// instruction that this derivation be proven by a test.
		const REGISTERED_MODULES = [
			'c',
			'cpp',
			'csharp',
			'go',
			'java',
			'javascript',
			'kotlin',
			'php',
			'python',
			'ruby',
			'rust',
			'swift',
			'typescript'
		];
		const expected = [...REGISTERED_MODULES, ...Object.keys(HLJS_ALIAS_COVERAGE)].sort();
		expect(HIGHLIGHT_COVERAGE).toEqual(expected);
	});
});
