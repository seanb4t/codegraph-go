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
// documented global property 05-04 reads after every layoutstop. See
// GraphCanvas.svelte's own module-script comment for what each field
// means and exactly where its two timers start and stop; this shape is
// duplicated (not imported) here because an ambient .d.ts cannot import
// a type from a .svelte file's module context.
declare global {
	interface Window {
		__codegraphFileGraphMetrics?: {
			timeToInteractiveMs: number;
			layoutDurationMs: number;
			nodeCount: number;
			edgeCount: number;
		};
	}
}

export {};
