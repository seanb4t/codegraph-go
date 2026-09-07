// live-store.ts is the epoch-scoped broadcast store over Task 1's
// client (LIV-02/LIV-03): the ONE place that decides whether a
// delivered event is ADMITTED, and the only place that can stamp the
// observation seam's appliedAtMs — admission is the store's decision,
// not the client's, so the client cannot fill that field in itself.
//
// The generation gate is scoped to the CONNECTION EPOCH, not to a bare
// cross-connection generation comparison (T-06-41): an event is
// admitted when its epoch differs from the last admitted epoch, OR its
// generation is strictly greater than the last admitted generation
// within the SAME epoch. A strict `>` gate with no epoch would leave a
// tab permanently deaf after the server restarts and its generation
// counter begins again at 1 — the first event of every NEW connection is
// therefore always admitted, regardless of its generation number.
import { uiClient } from '$lib/client';
import { startLiveClient, type LiveEvent, type LiveClientHandle } from './live-client';

export interface LiveStore {
	// subscribe follows the same store contract createStatusGate already
	// exposes: it returns an unsubscribe and invokes `run` synchronously
	// once with the CURRENT value — `null` until the first event is ever
	// admitted, since (unlike the status gate) there is no initial fetch
	// to seed it with.
	subscribe(run: (live: LiveEvent | null) => void): () => void;
}

export interface LiveStoreHandle extends LiveStore {
	stop(): void;
}

function defaultStartClient(onEvent: (live: LiveEvent) => void): LiveClientHandle {
	return startLiveClient({
		watchGraph: (request, options) => uiClient.watchGraph(request, options),
		onEvent
	});
}

/** stampApplied looks the observation record up by IDENTITY — the
 * (epoch, generation) pair — never by generation alone: after a server
 * restart the same generation number recurs in a new epoch, and a
 * generation-only lookup would stamp the wrong record. Defensive no-op
 * if the seam or the record does not exist (e.g. a test feeding events
 * directly to the store with no live client installed). */
function stampApplied(epoch: number, generation: bigint): void {
	const observations = window.__codegraphLiveObservations;
	if (!observations) return;
	const generationNumber = Number(generation);
	for (let i = observations.events.length - 1; i >= 0; i--) {
		const record = observations.events[i];
		if (record.epoch === epoch && record.generation === generationNumber) {
			record.appliedAtMs = performance.now();
			return;
		}
	}
}

/** createLiveStore wires a live client (real by default; injectable for
 * tests) through the epoch-scoped admission gate described above. Every
 * ADMITTED event both updates `current` (delivered to subscribers) and
 * stamps the matching observation record — a DROPPED event does neither,
 * leaving its appliedAtMs null. */
export function createLiveStore(
	startClient: (onEvent: (live: LiveEvent) => void) => { stop(): void } = defaultStartClient
): LiveStoreHandle {
	let current: LiveEvent | null = null;
	let lastAdmittedEpoch: number | null = null;
	let lastAdmittedGeneration = 0n;
	const listeners = new Set<(live: LiveEvent | null) => void>();

	function admit(epoch: number, generation: bigint): boolean {
		if (lastAdmittedEpoch === null || epoch !== lastAdmittedEpoch) return true;
		return generation > lastAdmittedGeneration;
	}

	function handle(live: LiveEvent): void {
		if (!admit(live.epoch, live.event.generation)) return;
		lastAdmittedEpoch = live.epoch;
		lastAdmittedGeneration = live.event.generation;
		current = live;
		stampApplied(live.epoch, live.event.generation);
		for (const listener of listeners) listener(current);
	}

	const client = startClient(handle);

	return {
		subscribe(run) {
			listeners.add(run);
			run(current);
			return () => {
				listeners.delete(run);
			};
		},
		stop() {
			client.stop();
		}
	};
}
