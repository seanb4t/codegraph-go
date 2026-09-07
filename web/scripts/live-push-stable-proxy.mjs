#!/usr/bin/env node
// live-push-stable-proxy.mjs — 06-06 Task 1: a stable-origin loopback
// reverse proxy in front of a `codegraph ui` that may be stopped and
// restarted behind it.
//
// This exists because `codegraph ui` binds an ephemeral loopback port
// and registers no bind/port flag (internal/cli/ui.go:24-33, :44-84 —
// "no flag for a bind address, port, hostname, token or any other
// credential"). Stop and restart it and a browser tab pointed directly
// at its old URL is pointed at a dead origin forever: its reconnect loop
// can never reach the new process, so 06-06 Task 2's RECONNECT scenario
// ("every tab is receiving events again afterwards") is unsatisfiable
// without this proxy. Adding a port flag to `codegraph ui` is NOT the
// fix — that would change SRV-01/SRV-03's shipped command surface and
// reopen D-08 for a test harness's benefit. This proxy changes nothing
// that ships; it lives only under web/scripts/.
//
// Two properties are load-bearing, not incidental:
//
//   1. Host/Origin REWRITE. internal/uiserver/originguard.go's
//      originHostGuard requires Host to be a member of the allowlist for
//      the UPSTREAM's own bound port, AND (when present) Origin to equal
//      "http://" + that same Host — membership in an origin allowlist
//      alone is not enough; the pairing is the whole point (see that
//      file's own doc comment, lines 42-67). A browser tab is opened on
//      THIS proxy's origin, so every request it sends carries the
//      PROXY's Host/Origin, which the upstream would reject outright.
//      This proxy rewrites both headers to the upstream's own authority,
//      to the SAME spelling, before forwarding — exercising the guard
//      exactly as a real same-origin browser request would, never
//      bypassing it.
//
//   2. STREAMING pass-through, never buffering. `res.writeHead` copies
//      the upstream's status and headers verbatim, then the upstream
//      response is PIPED to the client — no accumulation of any kind.
//      This is not merely a latency nicety: Task 2's FAN-OUT scenario
//      waits for every open tab to receive generation k before
//      triggering generation k+1 over this same proxy. A buffering
//      proxy would never deliver an intermediate chunk at all, and that
//      wait would simply hang forever — a deadlock, not a diagnosis.
//      This file's own <verify> command below proves the streaming
//      property directly and by name, in seconds, so a future
//      regression here is caught here rather than as an unexplained
//      Task 2 timeout.
//
// No dependency beyond Node's own standard library — node:http and
// node:url only.
import * as http from 'node:http';
import { URL } from 'node:url';

/**
 * @typedef {Object} StableProxyHandle
 * @property {string} url - the proxy's own stable base URL (e.g.
 *   "http://127.0.0.1:54321"). This is the origin every browser tab in
 *   Task 2 is opened on, and it never changes even when the upstream
 *   `codegraph ui` process is stopped and a new one started in its
 *   place — that stability is this whole file's reason to exist.
 * @property {(upstreamUrl: string) => void} setUpstream - repoint the
 *   proxy at a newly started `codegraph ui` instance. Safe to call at
 *   any time, including while requests are in flight against the OLD
 *   upstream (those already-open sockets keep talking to whichever
 *   upstream they were dialed against; only requests dialed AFTER this
 *   call see the new one).
 * @property {() => Promise<void>} close - release the listener and
 *   forcibly end any sockets still open (a long-lived streaming
 *   response would otherwise hold the ordinary graceful http.Server
 *   close() pending indefinitely).
 */

/**
 * startProxy starts a loopback reverse proxy bound once, immediately,
 * and returns its handle. The upstream starts unset — the proxy answers
 * every request with 502 until setUpstream is called at least once,
 * exactly as it does when the upstream later goes down (both are "no
 * live upstream right now", handled identically).
 *
 * @param {{port?: number}} [opts] - `port` defaults to 0 (OS-assigned
 *   ephemeral); the caller only ever consumes the resulting `url`, never
 *   the raw port number, so an ephemeral proxy port is exactly as
 *   "stable" as a fixed one for this file's purpose — what matters is
 *   that it does not change out from under an already-open tab.
 * @returns {Promise<StableProxyHandle>}
 */
export function startProxy(opts = {}) {
	const { port = 0 } = opts;

	/** @type {string | null} */
	let upstreamUrl = null;

	/** @type {Set<import('node:net').Socket>} */
	const sockets = new Set();

	const server = http.createServer((req, res) => {
		if (!upstreamUrl) {
			res.writeHead(502, { 'content-type': 'text/plain' });
			res.end('live-push-stable-proxy: no upstream configured');
			return;
		}

		let upstream;
		try {
			upstream = new URL(upstreamUrl);
		} catch (err) {
			res.writeHead(502, { 'content-type': 'text/plain' });
			res.end(`live-push-stable-proxy: invalid upstream URL: ${err instanceof Error ? err.message : String(err)}`);
			return;
		}

		// Rewrite BOTH Host and Origin to the upstream's own authority,
		// to the identical spelling — originHostGuard's pairing check
		// (Origin === "http://" + Host) rejects a request where only one
		// of the two was rewritten, and its membership check rejects a
		// request that still carries the PROXY's own authority in
		// either header. See this file's header comment, point 1.
		const headers = { ...req.headers };
		headers['host'] = upstream.host;
		if (headers['origin'] !== undefined) {
			headers['origin'] = `http://${upstream.host}`;
		}

		// failUpstream is the ONE place that turns "the upstream is gone"
		// into "the client sees a genuine stream failure". It is reached
		// from three different signals below because a killed process
		// (SIGKILL, no graceful FIN — exactly the RECONNECT scenario's own
		// shape) surfaces differently depending on whether the failure
		// happens before or after the response headers were already
		// forwarded: before, `proxyReq` itself emits 'error'; after,
		// Node's http client reports it on the response object instead
		// (as 'aborted', as 'error', or as a 'close' that never saw
		// 'end') — a proxy that only listened on `proxyReq` would forward
		// an initial connection failure correctly but let a MID-STREAM
		// kill hang the client's fetch forever, which is exactly the
		// silent-hang bug this comment exists to prevent a future editor
		// from reintroducing.
		let failed = false;
		function failUpstream() {
			if (failed) return;
			failed = true;
			if (!res.headersSent) {
				res.writeHead(502, { 'content-type': 'text/plain' });
				res.end('live-push-stable-proxy: upstream unreachable');
			} else if (!res.writableEnded) {
				res.destroy();
			}
		}

		const proxyReq = http.request(
			{
				hostname: upstream.hostname,
				port: upstream.port,
				path: req.url,
				method: req.method,
				headers
			},
			(upstreamRes) => {
				// writeHead copies status + headers verbatim, then the
				// body is PIPED — no buffering middleware, no response
				// accumulation of any kind. See header comment, point 2.
				res.writeHead(upstreamRes.statusCode ?? 502, upstreamRes.headers);
				upstreamRes.on('error', failUpstream);
				upstreamRes.on('aborted', failUpstream);
				upstreamRes.on('close', () => {
					if (!upstreamRes.complete) failUpstream();
				});
				upstreamRes.pipe(res);
			}
		);

		proxyReq.on('error', failUpstream);

		req.on('error', () => {
			proxyReq.destroy();
		});

		req.pipe(proxyReq);
	});

	server.on('connection', (socket) => {
		sockets.add(socket);
		socket.on('close', () => sockets.delete(socket));
	});

	return new Promise((resolve, reject) => {
		server.once('error', reject);
		server.listen(port, '127.0.0.1', () => {
			const addr = server.address();
			if (addr === null || typeof addr === 'string') {
				reject(new Error('live-push-stable-proxy: server.address() returned no port'));
				return;
			}
			const base = `http://127.0.0.1:${addr.port}`;

			/** @type {StableProxyHandle} */
			const handle = {
				url: base,
				setUpstream(url) {
					upstreamUrl = url;
				},
				close() {
					return new Promise((resolveClose) => {
						for (const socket of sockets) socket.destroy();
						server.close(() => resolveClose());
					});
				}
			};
			resolve(handle);
		});
	});
}
