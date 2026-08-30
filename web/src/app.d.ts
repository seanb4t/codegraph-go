// See https://svelte.dev/docs/kit/types#app.d.ts
// for information about these interfaces
declare global {
	namespace App {
		// interface Error {}
		// interface Locals {}
		// interface PageData {}
		// interface PageState {}
		// interface Platform {}
	}
}

// 05-03 Task 2: cytoscape-elk ships no bundled TypeScript definitions, and
// no @types/cytoscape-elk package exists on the npm registry (checked live
// against the registry this task, not assumed from 05-RESEARCH.md's
// unresolved A4 assumption). Its default export is a Cytoscape extension
// registration function — `cytoscape.use(elk)` — that installs the 'elk'
// layout name GraphCanvas.svelte (05-03 Task 3) passes to `layout: { name:
// 'elk', ... }`. A minimal ambient declaration here avoids adding a new
// devDependency for a type shim, matching ambient-node.d.ts's precedent of
// a narrowly-scoped ambient module over installing a package.
declare module 'cytoscape-elk' {
	import type cytoscape from 'cytoscape';
	const register: (cy: typeof cytoscape) => void;
	export default register;
}

export {};
