// rpc-errors.ts is the ONE Connect-error translation for the Browse view
// and every view after it (D-04): Phases 4/5/6 hit the identical degraded
// server contract this module classifies, so it is built once here rather
// than as an ad-hoc `.catch((err: unknown) => ...)` per view (the shape
// web/src/routes/+page.svelte's existing GetStatus call uses, and the
// shape this module exists to replace everywhere else).
//
// Mirrors the server's own classification, internal/uiserver/handlers.go's
// mapEngineError: CodeNotFound, CodeInvalidArgument, and CodeUnavailable
// carrying a typed IndexingInProgress detail (distinguished from a BARE
// CodeUnavailable, which this module reports as 'unknown' rather than
// inventing a fifth kind for a case the server never actually produces
// today). classifyRpcError never throws — it is the last line between a
// server error and a blank pane.
import { ConnectError, Code } from '@connectrpc/connect';
import { IndexingInProgressSchema } from '$lib/gen/ui_pb';

export type RpcFailure =
	| { kind: 'not-found'; message: string }
	| { kind: 'invalid-input'; message: string }
	| { kind: 'indexing'; message: string }
	| { kind: 'unknown'; message: string };

function messageOf(err: ConnectError): string {
	return err.rawMessage || err.message;
}

export function classifyRpcError(err: unknown): RpcFailure {
	try {
		if (!(err instanceof ConnectError)) {
			return {
				kind: 'unknown',
				message: err instanceof Error ? err.message : String(err)
			};
		}

		switch (err.code) {
			case Code.NotFound:
				return { kind: 'not-found', message: messageOf(err) };
			case Code.InvalidArgument:
				return { kind: 'invalid-input', message: messageOf(err) };
			case Code.Unavailable: {
				// D-04: GetStatus is the only rpc that answers when degraded;
				// every OTHER handler returns CodeUnavailable carrying a typed
				// IndexingInProgress detail (degrade.go's errIndexingInProgress).
				// findDetails() decodes error details wrapped with
				// google.protobuf.Any on the wire; decoding failures are
				// ignored by the library and simply omit the detail — never
				// thrown here.
				const details = err.findDetails(IndexingInProgressSchema);
				if (details.length > 0) {
					return { kind: 'indexing', message: details[0].message || messageOf(err) };
				}
				return { kind: 'unknown', message: messageOf(err) };
			}
			default:
				return { kind: 'unknown', message: messageOf(err) };
		}
	} catch {
		// classifyRpcError must never throw (D-04): a malformed detail or
		// any other unexpected shape degrades to 'unknown' rather than
		// propagating and turning a classification bug into a blank pane.
		return { kind: 'unknown', message: 'an unknown error occurred' };
	}
}
