import adapter from '@sveltejs/adapter-static';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

// NOTE (deviation from 02-01-PLAN.md's file list, recorded in the plan's
// SUMMARY): the SvelteKit toolchain scaffolded by `sv@0.17.0` for
// @sveltejs/kit ^2.63.0 no longer reads svelte.config.js at all — project
// (kit) configuration, including the adapter, is a top-level option of
// the `sveltekit()` Vite plugin itself. There is no separate
// svelte.config.js to author; this file is both the Vite config and the
// SvelteKit config (confirmed against sveltejs/kit's own current docs and
// by this project's own `pnpm build`, verified below).
export default defineConfig({
	plugins: [
		sveltekit({
			compilerOptions: {
				// Force runes mode for the project, except for libraries. Can be removed in svelte 6.
				runes: ({ filename }) =>
					filename.split(/[/\\]/).includes('node_modules') ? undefined : true
			},

			// adapter-static (D-01/D-03/D-04): pages/assets default to
			// 'build' (adapter's own default, unchanged here); fallback
			// is 'index.html' — NOT SvelteKit's own recommended
			// '200.html' — because this app is served by our own Go mux,
			// which never special-cases a "200.html" filename the way a
			// static host does, and criterion 2's wording names
			// index.html literally (D-04). strict is false because
			// nothing in this app is prerendered (pure SPA mode, see
			// +layout.ts's ssr=false/prerender=false); adapter-static's
			// default strict:true would reject a build with no
			// prerendered pages. precompress is false (Claude's
			// Discretion): .br/.gz siblings would double the embedded
			// byte count and double every entry in criterion 1's
			// file-list diff for no benefit on a loopback socket.
			adapter: adapter({
				pages: 'build',
				assets: 'build',
				fallback: 'index.html',
				precompress: false,
				strict: false
			})
		})
	]
});
