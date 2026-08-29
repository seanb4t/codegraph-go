// Ambient declaration for the one Node global web/vite.config.ts reads at
// config-authoring time (process.env.VITEST / process.env.CODEGRAPH_WEB_BUILD_DIR
// — both pre-existing, from 02-01/02-06). This project has no @types/node
// dependency (adding one is a new npm package and requires a
// package-legitimacy checkpoint per this repo's standing rule), so this
// stays a minimal, narrowly-scoped global declaration rather than pulling
// in a new devDependency for one config-time read.
//
// 03-04 deviation (Rule 3 — blocking issue): `pnpm check` (svelte-check)
// is not wired into any existing Taskfile target or CI job, so this gap
// pre-dated this plan and was never caught. It became a genuine blocker
// only because 03-04-PLAN.md is the first plan to assert `pnpm check`
// exits 0 as an acceptance criterion.
declare const process: {
	env: Record<string, string | undefined>;
};
