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
//
// A THIRD global — the live-push observation seam — is populated by
// web/src/lib/live/live-client.ts (the reconnecting stream consumer) and
// web/src/lib/live/live-store.ts (the epoch-scoped generation gate, the
// only place that can know whether an event was ever ADMITTED). Its
// lifetime differs from both seams above: `events` grows by one entry
// per event actually delivered off the wire, for as long as the live
// client is connected, capped at a fixed size (oldest discarded first);
// `connections` grows by one entry per connection ATTEMPT — successful
// or not — for the client's whole lifetime, same cap. This seam is
// OBSERVATION ONLY: nothing under web/src/ ever reads it back — it
// exists solely for a real-browser harness to inspect reconnect timing,
// generation-gate admission, and resume cursors that are otherwise
// unobservable from outside the page (the resume cursor sits inside a
// length-prefixed wire envelope; "was this event applied" is a
// post-decode, post-gate fact with no other externally visible trace).
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
		__codegraphLiveObservations?: {
			events: Array<{
				generation: number;
				epoch: number;
				seeded: boolean;
				receivedAtMs: number;
				appliedAtMs: number | null;
			}>;
			connections: Array<{
				epoch: number;
				attempt: number;
				baseDelayMs: number | null;
				scheduledDelayMs: number | null;
				requestedSinceGeneration: number;
				openedAtMs: number | null;
			}>;
		};
	}
}

export {};
