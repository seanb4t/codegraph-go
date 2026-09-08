// SPA-mode requirement documented in SvelteKit's own adapter-static /
// single-page-apps docs: a fallback build (D-04) is only legal when the
// whole app opts out of both SSR and prerendering. Without this, `pnpm
// build` fails with an error naming the fallback/prerender conflict.
export const ssr = false;
export const prerender = false;
