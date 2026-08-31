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

// 05-03 Task 3: GraphCanvas.svelte's measurement seam — the single
// documented global property the GRF-01 gate reads after the FIRST
// layout settle only, never again (05-08). See GraphCanvas.svelte's own
// module-script comment for what each field means and exactly where its
// two timers start and stop; this shape is duplicated (not imported)
// here because an ambient .d.ts cannot import a type from a .svelte
// file's module context.
//
// 05-08 Task 2: a SECOND, separately-documented global — the
// rendered-geometry seam — publishes after EVERY layout settle (a
// different lifetime from the metrics above). Canvas rendering leaves no
// DOM element per graph node; this is what lets a real browser click a
// SPECIFIC node by its own reported position.
declare global {
	interface Window {
		__codegraphFileGraphMetrics?: {
			timeToInteractiveMs: number;
			layoutDurationMs: number;
			nodeCount: number;
			edgeCount: number;
		};
		__codegraphFileGraphGeometry?: Array<{
			id: string;
			x: number;
			y: number;
			expandable: boolean;
			fileCount?: number;
		}>;
	}
}

export {};
