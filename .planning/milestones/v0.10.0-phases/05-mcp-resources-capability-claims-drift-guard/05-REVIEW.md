---
phase: 05-mcp-resources-capability-claims-drift-guard
reviewed: 2026-08-12T00:00:00Z
depth: standard
files_reviewed: 20
files_reviewed_list:
  - internal/mcp/resources_schema_drift_test.go
  - internal/mcp/resources_test.go
  - internal/mcp/resources.go
  - internal/mcp/resources/callees.md
  - internal/mcp/resources/callers.md
  - internal/mcp/resources/explore.md
  - internal/mcp/resources/files.md
  - internal/mcp/resources/impact.md
  - internal/mcp/resources/index-state.md
  - internal/mcp/resources/node.md
  - internal/mcp/resources/search.md
  - internal/mcp/resources/status.md
  - internal/mcp/resources/tools-filter.md
  - internal/mcp/server.go
  - test/wireoracle/anchors.go
  - test/wireoracle/COVERAGE-BASELINE.md
  - test/wireoracle/MUTATION-PROOF.md
  - test/wireoracle/oracle_test.go
  - test/wireoracle/scenarios.go
  - testdata/wireoracle/transcripts/*.golden (43 frozen wire transcripts, skimmed)
findings:
  critical: 1
  warning: 1
  info: 0
  total: 2
status: issues_found
---

# Phase 05: Code Review Report

**Reviewed:** 2026-08-12
**Depth:** standard
**Files Reviewed:** 20 (plus 43 golden transcript fixtures skimmed)
**Status:** issues_found

## Summary

This phase adds the MCP Resources capability (10 embedded fact-sheet/behavior-doc resources,
`internal/mcp/resources.go`), a set of drift guards comparing resource-doc prose against tool
schemas (`internal/mcp/resources_schema_drift_test.go`), and 13 new wire-oracle scenarios plus
supporting anchors proving the resources surface on the wire. The resource-registration code,
the drift-guard tests, and the 10 markdown fact sheets are all internally consistent and the
package's own test suite (`go test ./internal/mcp/...`) passes cleanly — I ran it as part of this
review and found no failures.

The one substantive defect found is not in the new resources code itself, but in
`internal/mcp/server.go`'s `ServeStdio`/`stdinLingerReader`/`pendingWriter` mechanism, a file in
this phase's review scope. That mechanism was built specifically to prevent a real race (a
response to the client's last request getting lost when the client closes stdin immediately after
sending it), but its bookkeeping does not distinguish "a write that resolves an owed response"
from "any write to stdout at all" — and this phase's sibling feature area (SPEC-09 live
notifications, `notifications/tools/list_changed`, `notifications/subscriptions/acknowledged`)
routes unsolicited server-initiated messages through the exact same writer, corrupting the
counter the whole mechanism depends on. Traced against the vendored SDK's own transport code to
confirm the write path is genuinely shared. See CR-01 below.

A second, minor finding: `resources/index-state.md`'s prose overstates when the tool catalog is
re-checked ("on every request"), when the implementation only re-checks on four specific methods.
This is exactly the class of claim the phase's own drift guards exist to catch, but the guards
check numeric claims, count claims, env-var names, and host-fact leakage — not this qualitative
behavioral claim — so it slipped through untested. See WR-01 below.

## Critical Issues

### CR-01: pendingWriter's "pending response" counter is corrupted by server-initiated notifications, defeating the EOF-race fix it exists to provide

**File:** `internal/mcp/server.go:225-349` (mechanism), specifically:
- `stdinLingerReader.Read` (lines 271-296): increments `pending` only when a line read from stdin
  looks like a *client* JSON-RPC call (`looksLikeJSONRPCCall`, lines 314-329) — i.e., only for
  requests the server now owes a response to.
- `pendingWriter.Write` (lines 339-343): decrements `pending` on **every** `Write()` call to
  stdout, unconditionally.

**Issue:** `ServeStdio`'s own extensive doc comment (lines 178-224) explains in detail why this
counter exists: go-sdk's stdio read loop marks the connection "shutting down" the instant it
observes stdin EOF, and any response still being computed for an already-accepted request is
silently dropped if EOF is reported too early. The fix is to hold EOF back until every accepted
request's response has actually reached the wire, tracked via `pending`.

That fix assumes a 1:1 relationship between "increment on an owed request" and "decrement on that
request's response being written." That assumption is false for this server: every outbound
message — including a server-initiated **notification**, not just a response to a specific
client request — goes through the identical `pendingWriter.Write` call and decrements the exact
same counter. I confirmed this by reading the vendored SDK's transport code directly
(`github.com/modelcontextprotocol/go-sdk@v1.7.0/mcp/transport.go`): `IOTransport.Connect` wraps
the given `Reader`/`Writer` into one `rwc` passed to `newIOConn`, and `ioConn.Write` (line 673) is
the single path every `jsonrpc.Message` — response, notification, or request — is written through,
one `rwc.Write(data)` call per message (line 720; outgoing batching is inert here since
`t.outgoingBatch` is never sized above its zero-capacity default, so no message is ever coalesced
with another).

This phase's own sibling work (SPEC-09, the same "Phase 5" area) added exactly the kind of traffic
that breaks this: `notifications/tools/list_changed` (server-initiated, fired by
`changeAndNotify` on a catalog change) and `notifications/subscriptions/acknowledged` (the
subscription ack, itself sent as a notification rather than a matched response — see
`modern-listen-catalog-change`'s own scenario doc comment in `test/wireoracle/scenarios.go:1131-1213`,
which states plainly that `SubscriptionsListenResult` is "sent only on graceful subscription
teardown, never on stdin EOF"). Each of these is a `Write()` call that decrements `pending`
without ever having incremented it for that specific write.

Concrete failure sequence this enables: a client sends one request (say `tools/call`, id=N) and
closes stdin immediately after (`pending` is now 1). Before the handler finishes and writes its
response, an unrelated notification fires (e.g., a debounced `tools/list_changed` from a
concurrent index change, or the subscription-ack notification for a `subscriptions/listen` the
same session opened earlier) and is written to stdout — `pendingWriter.Write` decrements `pending`
back to 0 even though the id=N response has not been written yet. `stdinLingerReader`'s very next
`Read()` call observes EOF; `waitForDrain()` sees `pending.Load() <= 0` and returns immediately
instead of waiting; EOF propagates to go-sdk, the connection is marked shutting down, and the
still-in-flight id=N response is dropped — exactly the data-loss bug this entire mechanism (and
its 45-line doc comment) was written to eliminate, reintroduced by an uncounted write class the
implementation never accounts for.

This is a real, reachable path (not merely theoretical): `modern-listen-catalog-change`'s own
frozen scenario proves this server emits exactly this kind of unsolicited-notification traffic in
production-shaped sessions today, and any Modern client opting into `subscriptions/listen` while
also making ordinary tool/resource calls is exposed to this race whenever a notification lands
between a request's acceptance and its response being written, near the end of a session.

**Fix:** Track pending responses per accepted request id (or at minimum, distinguish "a write that
is itself a response frame" from "a write that is a notification frame") rather than a single
undifferentiated counter incremented on reads and decremented on all writes. For example, have
`pendingWriter.Write` inspect the outgoing frame for an `"id"` field matching one of the
outstanding accepted ids (mirroring `looksLikeJSONRPCCall`'s own sniffing approach) and only
decrement when the write is actually resolving a pending id — a notification write (no `"id"`, or
an `"id"` that is a subscription id rather than a request id) should never decrement the counter:

```go
// pendingWriter should only decrement for a write that resolves a
// specific accepted request id, not for any write to stdout.
func (p *pendingWriter) Write(b []byte) (int, error) {
	n, err := p.w.Write(b)
	if isResponseFrame(b) { // has "id", has no "method" (i.e. is a Response, not a Notification/Request)
		p.pending.Add(-1)
	}
	return n, err
}
```

## Warnings

### WR-01: `resources/index-state.md` overstates when the tool catalog is re-checked

**File:** `internal/mcp/resources/index-state.md:7-10`
**Issue:** The doc states: "The catalog is re-checked on every request." In fact,
`BuildServer`'s middleware (`internal/mcp/server.go:669-672`) only calls `recheckCatalog()` for
four specific methods:

```go
switch method {
case "initialize", "tools/list", "tools/call", "server/discover":
	recheckCatalog()
}
```

`resources/list` and `resources/read` (and any future method) never trigger a re-check — which is
correct, since resources are index-independent and don't need one — but it means "on every
request" is not literally true. An agent reading this fact sheet could reasonably conclude every
method observes a live catalog state, which isn't the guarantee the code actually provides (only
the four catalog-dependent methods do). This is precisely the category of claim this phase's own
drift-guard suite (`resources_schema_drift_test.go`) was built to prevent from going stale, but
none of the existing guards (numeric claims, count claims, env-var names, host-fact leakage) check
qualitative behavioral prose like this, so it is currently unenforced.

**Fix:** Narrow the claim to match the actual trigger set, e.g.:

```markdown
The catalog is re-checked before every `initialize`, `tools/list`, `tools/call`, and
`server/discover` request. An index created part-way through a session appears without a client
restart or reconnect.
```

---

_Reviewed: 2026-08-12_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
