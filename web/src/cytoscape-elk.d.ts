// Ambient declaration for cytoscape-elk (05-03 Task 2). cytoscape-elk
// ships no bundled TypeScript definitions, and no @types/cytoscape-elk
// package exists on the npm registry (checked live against the registry
// this task, not assumed from 05-RESEARCH.md's unresolved A4 assumption).
// Its default export is a Cytoscape extension registration function —
// `cytoscape.use(elk)` — that installs the 'elk' layout name
// GraphCanvas.svelte (05-03 Task 3) passes to `layout: { name: 'elk',
// ... }`. A minimal ambient declaration here avoids adding a new
// devDependency for a type shim, matching ambient-node.d.ts's precedent
// of a narrowly-scoped standalone ambient module file over installing a
// package. This declaration is kept in its own file (rather than folded
// into app.d.ts) because svelte-check does not pick up a module
// augmentation of this shape when it lives inside app.d.ts alongside
// app.d.ts's own `declare global` block — confirmed empirically this
// task: the identical declaration in a standalone file is recognized,
// the same text pasted into app.d.ts is not.
//
// The registration function's parameter is typed `any` rather than the
// real cytoscape core type, deliberately: importing that type here would
// make this file itself an importer of the cytoscape PACKAGE (a plain
// `import ... "cytoscape"` specifier), breaking GraphCanvas.svelte's
// single-importer property — GRF-05's seam requires the cytoscape import
// to appear in exactly one file, and did NOT the first time this
// declaration imported that type.
declare module 'cytoscape-elk' {
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	const register: (cy: any) => void;
	export default register;
}
