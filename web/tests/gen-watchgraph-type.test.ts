// gen-watchgraph-type.test.ts — plan 06-01 Task 3: a COMPILE-TIME
// assertion that the generated TypeScript client's watchGraph method is
// server-streaming, not merely present by name. Counting `WatchGraph`
// string occurrences in ui_pb.ts would prove the identifier exists and
// nothing about the method's KIND — both message schemas alone would
// satisfy that. This file asserts the shape through the type system
// instead, which `pnpm run check` enforces because the SvelteKit-
// generated tsconfig includes `../tests/**/*.ts`.
//
// If watchGraph were generated as a unary method, its return type would
// be `Promise<WatchGraphEvent>`, and assignAsyncIterable below would fail
// to type-check — that is the actual gate this file provides.
import { describe, it, expect } from 'vitest';
import { uiClient } from '$lib/client';
import type { WatchGraphEvent } from '$lib/gen/ui_pb';

type WatchGraphReturn = ReturnType<typeof uiClient.watchGraph>;

// assignAsyncIterable type-checks ONLY if WatchGraphReturn is (structurally
// compatible with) AsyncIterable<WatchGraphEvent> — the server-streaming
// shape Connect-ES generates for a `returns (stream ...)` rpc. A unary
// rpc's `Promise<WatchGraphEvent>` is not an AsyncIterable and would fail
// this assignment at compile time, not at runtime.
function assignAsyncIterable(v: WatchGraphReturn): AsyncIterable<WatchGraphEvent> {
	return v;
}

describe('watchGraph generated client method', () => {
	// The runtime formality: vitest fails a file with zero tests. The real
	// gate is the type-level assignAsyncIterable above, checked by
	// `pnpm run check`, not this assertion.
	it('is a function', () => {
		expect(typeof uiClient.watchGraph).toBe('function');
	});

	it('assignAsyncIterable is referenced (keeps the type check reachable)', () => {
		expect(typeof assignAsyncIterable).toBe('function');
	});
});
