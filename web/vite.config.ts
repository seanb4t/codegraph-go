/// <reference types="vitest/config" />
import tailwindcss from '@tailwindcss/vite';
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
		// Tailwind v4 (D-17) wires in as a Vite plugin, not PostCSS — no
		// postcss.config.* file exists anywhere under web/, and none should
		// be added. Must run before sveltekit() so its :root { @import
		// "tailwindcss"; ... } transform sees app.css before SvelteKit's own
		// asset pipeline processes it.
		tailwindcss(),
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
			//
			// CODEGRAPH_WEB_BUILD_DIR (02-06-PLAN.md Task 1, step (b)):
			// pages/assets read this env var, defaulting to the literal
			// 'build' when unset, so every existing invocation and every
			// existing path in this phase behaves identically. The ONLY
			// reason this override exists is so `task web:build:verify`
			// can prove the toolchain still produces a working build by
			// building into a scratch directory, without ever writing
			// into the committed tree. The default must never change:
			// D-03 fixes 'build' as the committed output directory, and
			// `//go:embed all:build` (web/embed.go) names it literally.
			//
			// DEVIATION from 02-06-PLAN.md's literal file list: the plan
			// names `web/svelte.config.js` as the file this override
			// belongs in. That file does not exist in this SvelteKit
			// toolchain version — adapter-static's configuration has
			// always lived here, in vite.config.ts's sveltekit() plugin
			// options (see the file-level NOTE above, and
			// 02-01-SUMMARY.md's Deviation 1, which first recorded this).
			adapter: adapter({
				pages: process.env.CODEGRAPH_WEB_BUILD_DIR || 'build',
				assets: process.env.CODEGRAPH_WEB_BUILD_DIR || 'build',
				fallback: 'index.html',
				precompress: false,
				strict: false
			})
		})
	],

	// DEVIATION (03-01-PLAN.md Task 2 step (c), recorded in the SUMMARY):
	// under jsdom, Vite's default package-export condition resolution
	// picked Svelte's SERVER build for .svelte imports (`svelte/src/
	// internal/server/errors.js`'s `lifecycle_function_unavailable`
	// thrown from `mount()`), rather than the client build the harness
	// fixture needs. Adding the 'browser' resolve condition — guarded so
	// it applies ONLY under vitest (process.env.VITEST, which Vitest
	// itself sets) — fixes the .svelte import to resolve client-side,
	// without changing resolution for `vite build`/`vite dev`.
	resolve: process.env.VITEST
		? {
				conditions: ['browser']
			}
		: undefined,

	// Vitest (03-01, BRW-06): configured HERE rather than a separate
	// vitest.config.ts, deliberately — Taskfile.yml's WEB_HASH_LIB
	// `web_source_files()` enumerates web/vite.config.ts as part of the
	// BLD-03 source digest but does NOT enumerate a hypothetical
	// vitest.config.ts, so a separate file would be invisible to the
	// build/drift guard. Tests live under web/tests/, which
	// .svelte-kit/tsconfig.json's `include` already covers
	// (../tests/**/*.ts, ../tests/**/*.svelte) and which
	// web_source_files() deliberately does NOT enumerate, so adding or
	// editing a test does not churn that digest.
	test: {
		environment: 'jsdom',
		include: ['tests/**/*.test.ts'],
		setupFiles: ['./tests/setup.ts'],
		globals: false
	}
});
