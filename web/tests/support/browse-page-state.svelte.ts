// Minimal reactive stand-in for `$app/state`'s `page`, used ONLY by
// browse-page.test.ts (CR-01's route-level regression test, 03-REVIEW.md)
// to drive +page.svelte's own `$derived(params)` exactly as SvelteKit's
// real client-side router does when a navigation changes `page.url`.
//
// This has to live in its own `.svelte.ts` module rather than inline in
// the test file: Svelte 5's `$state` rune is compiled by the Svelte
// compiler, and @sveltejs/vite-plugin-svelte only runs that compiler over
// `.svelte`, `.svelte.js` and `.svelte.ts` files ("universal reactivity")
// — a plain `*.test.ts` file is never rune-aware, so `$state` there would
// be an unresolved runtime call.
//
// mockPage is a SINGLETON (not a factory): the test file's `vi.mock('$app/
// state', ...)` factory and the test body both need the exact same
// reactive object reference — a factory-created instance would give the
// mocked module and the assertions two different objects, and mutating
// one would never be seen by the other.
export const mockPage = $state<{ url: URL }>({ url: new URL('http://localhost/browse') });

// resetMockPage re-points `mockPage.url` at a fresh URL between tests —
// mutating the existing `url` property (not replacing `mockPage` itself)
// keeps every earlier `import { mockPage } ...` binding live.
export function resetMockPage(url: string): void {
	mockPage.url = new URL(url);
}
