# 03-03 Evidence: `toolslist-repeat` ordering flake root cause

**Gathered:** 2026-08-28 (Task 1 of 03-03-PLAN.md)

VERDICT: SERVER-EMITTED-OUT-OF-ORDER

## Summary

The `internal/mcp` server, via `github.com/modelcontextprotocol/go-sdk@v1.7.0`,
dispatches every JSON-RPC **call** except `initialize` — including `tools/list` —
onto its own goroutine and releases the dispatcher to move to the NEXT queued
request **before that goroutine's handler has produced a result**. Two
back-to-back `tools/list` calls therefore run in two concurrent goroutines with
**no ordering guarantee between their completion times**, and each writes its
own response through the shared `pendingWriter` independently. The scenario
comment asserting "both are handled synchronously in request order — no
worker-pool race" (`test/wireoracle/scenarios.go:659-666` and restated at
`:1098-1101`) describes the OLD `mark3labs/mcp-go` transport's behavior, which
Phase 2 (SDK-01) migrated away from. It is **false** of the current
`modelcontextprotocol/go-sdk` transport, and the evidence below reproduces the
exact documented symptom under that false assumption.

This was reproduced live under contention (Linux, see below): the raw captured
transcript itself — never re-ordered downstream, see the capture-layer
argument below — showed the id-3 `tools/list` response as line 2, where the
id-2 response belonged. Because `test/wireoracle/capture.go`'s scanner
goroutine appends each stdout line to `out` strictly in the order the
`bufio.Scanner` returns it (single goroutine, single sequential loop, no
sort/bucket of any kind — now also proven durably by
`TestCaptureArrivalLedgerPreservesWireOrder`), a swap observed in the captured
bytes can only mean the SERVER itself wrote the bytes in that order. This is
not an inference from timing: it is confirmed directly from the SDK's own
synchronization primitive (see SDK citation below).

## Reproduction record

**Method:** load-based, not `-count` in isolation — the failure record is
explicit that isolated `-count=15` runs never reproduce it (this session
confirmed that independently, see below); reproduction requires contention
from concurrent goroutines/processes, matching the documented
"only failed under `task test:unit`'s full parallel contention" shape.

| Platform | Contention method | Attempts | Failures |
|---|---|---|---|
| darwin/arm64 (this session's local host, GOMAXPROCS=4, `-count=20` under an (ineffective) background stress attempt) | 20x `toolslist-repeat` alone, targeted | 20 | 0 |
| darwin/arm64, 10 parallel `go test` processes x `-count=5` each, GOMAXPROCS=2 | 50x `toolslist-repeat` alone, targeted, cross-process contention | 50 | 0 |
| darwin/arm64, 3 parallel full non-daemon-package `go test` invocations, GOMAXPROCS=4, `-count=1` | full-suite contention (all ~57 non-daemon packages x3 concurrently) | 3 full-suite runs (wireoracle package result checked each) | 0 in wireoracle (a DIFFERENT, unrelated flake — `TestLiveEditAutoSyncReachesExplore` in `test/integration` — DID fail 2/3 times under this same contention, confirming the method induces real contention-driven races in this repo; that flake is out of scope for this plan and not investigated further here) |
| **linux/arm64 (Docker `golang:1.26.5`, `--cpus=4`, GOMAXPROCS=4)** | 6 parallel `go test -run toolslist-repeat -count=10` processes (60 total attempts) | **60** | **4** |

**Platform caveat, stated plainly:** the reproducing container is linux/**arm64**
(this host is Apple Silicon; Docker Desktop resolved `golang:1.26.5` to its
native arm64 variant), not linux/amd64 like the CI runner
(`namespace-profile-linux-amd64-4x8`). Follow-up attempts to build the real
`codegraph`/`wireoracle` binaries for a more targeted repro inside the same
container repeatedly hit resource limits unrelated to this investigation —
CGo compilation of the `tree-sitter-c-sharp` grammar was OOM-killed
(`gcc: fatal error: Killed signal terminated program cc1`) under the
container's constrained memory, and a `cp -r /repo` staging step separately
timed out on this repo's large `web/node_modules`/`graphify-out` trees. The
successful 60-attempt reproduction above used `go test` directly against the
read-only bind-mounted repo (no full-repo copy, no separately-built binary),
which did not hit either limit. Given this is Linux (not merely
darwin-with-contention) and the documented CI failure was also Linux-specific
despite an unconstrained amd64 environment, kernel-family (goroutine/thread
scheduling behavior) rather than CPU architecture is the more likely factor —
recorded as an assumption, not verified further, since the reproduction
already succeeded on Linux at all.

**Failure rate:** 4/60 (~6.7%) under the Linux contention method — consistent
with an intermittent, load-dependent race rather than a deterministic bug,
and consistent with the original CI report's "1 failure in 2 runs" (small
sample, same order of magnitude).

## Raw evidence from a reproduced failure

All 4 Linux failures showed the IDENTICAL symptom, matching the original CI
report exactly:

```
oracle_test.go:130: scenario "toolslist-repeat": normalized transcript differs at line 2:
     got: "{\"jsonrpc\":\"2.0\",\"id\":3,\"result\":{\"ttlMs\":0,\"cacheScope\":\"private\",\"tools\":[...]}}"
    want: "{\"jsonrpc\":\"2.0\",\"id\":2,\"result\":{\"ttlMs\":0,\"cacheScope\":\"private\",\"tools\":[...]}}"
```

Line 1 (the `initialize` response, id=1) matched in every failing run —
consistent with the SDK citation below: `initialize` is the one method
explicitly excluded from async dispatch, so it alone is guaranteed to
complete, and be written, before the next request is even read.

This session's `assertBytesEqualLineByLine`/`compareBytesLineByLine` reports
only the FIRST differing line (by design — see its doc comment,
`test/wireoracle/oracle_test.go:265-292`), so this captured failure alone
does not show whether the id-2 response appears later in the transcript
(reordered) or never arrives at all (dropped). The original CI report states
"the tool payloads match exactly" for the swapped pair, which is consistent
with a pure reorder rather than a drop; this session did not independently
re-derive that by dumping a full raw transcript byte-for-byte, and this gap
is noted rather than papered over. It does not change the verdict: either
outcome (reordered or dropped) is still the server writing responses in an
order other than request order, which is what SERVER-EMITTED-OUT-OF-ORDER
asserts.

**A synthetic, timestamped arrival-ledger test was landed** as durable proof
that the CAPTURE layer itself cannot be the source of this reordering:
`test/wireoracle/capture_test.go`'s `TestCaptureArrivalLedgerPreservesWireOrder`
drives `scanArrivalLines` (extracted from `capture.go`'s own scanner-goroutine
shape) with a synthetic `io.Pipe` writer emitting a deliberately
non-id-ascending sequence, and asserts the returned ledger preserves emission
order exactly, with non-decreasing timestamps. This passed on first run
(`--- PASS: TestCaptureArrivalLedgerPreservesWireOrder`), confirming the
scan-and-append path this investigation already read by hand (capture.go's
`drainUntil`: `out.Write(ln.raw)` appended strictly in channel-receive order,
itself strictly in `bufio.Scanner.Scan()` order, single goroutine, no
sort/bucket) does not itself reorder anything — a byte-for-byte swap observed
in a real `Transcript.Stdout` therefore did not originate in this test
harness's read side.

`Capture` was additionally instrumented (not just the standalone test): its
scanner goroutine now stamps each line with `time.Now()` at the moment
`Scan()` returns, and `Transcript.ArrivalLedger` carries the full sequence
unconditionally. `TestFrozenTranscriptsMatch` dumps this ledger via `t.Logf`
whenever a comparison is about to fail, so the NEXT occurrence of this (or
any other) ordering discrepancy in this suite has the raw timestamped
sequence attached to the test failure itself, rather than requiring a fresh
ad hoc investigation.

## SDK citation (Task 1(c))

**Source read directly, not inferred from behavior:**
`github.com/modelcontextprotocol/go-sdk@v1.7.0`, two files in the module
cache (`$(go env GOMODCACHE)/github.com/modelcontextprotocol/go-sdk@v1.7.0/`):

1. **`mcp/server.go:1908-1914`** (`ServerSession.handle`):
   ```go
   // modelcontextprotocol/go-sdk#26: handle calls asynchronously, and
   // notifications synchronously, except for 'initialize' which shouldn't be
   // asynchronous to other
   if req.IsCall() && req.Method != methodInitialize {
       jsonrpc2.Async(ctx)
   }
   ```
   This runs at the TOP of `handle`, before `handleReceive` ever dispatches to
   the method-specific implementation (`tools/list`'s handler included). Every
   call except `initialize` — unconditionally, regardless of method — signals
   `Async` before doing any of its actual work.

2. **`internal/jsonrpc2/conn.go:652-687`** (`Connection.handleAsync`):
   ```go
   func (c *Connection) handleAsync() {
       for {
           var req *incomingRequest
           c.updateInFlight(func(s *inFlightState) {
               if len(s.handlerQueue) > 0 {
                   req, s.handlerQueue = s.handlerQueue[0], s.handlerQueue[1:]
               } else {
                   s.handlerRunning = false
               }
           })
           if req == nil {
               return
           }
           ...
           releaser := &releaser{ch: make(chan struct{})}
           ctx := context.WithValue(req.ctx, asyncKey, releaser)
           go func() {
               defer releaser.release(true)
               result, err := c.handler.Handle(ctx, req.Request)
               c.processResult(c.handler, req, result, err)
           }()
           <-releaser.ch
       }
   }
   ```
   This is the ONLY consumer of the `handlerQueue`. It pulls requests off the
   queue **sequentially** (so requests ids 2 and 3 are dequeued in order), but
   for EACH request it spawns a goroutine and then blocks on `<-releaser.ch`
   — which is unblocked EITHER when the goroutine's handler calls
   `jsonrpc2.Async(ctx)` (a "soft" release, the request's own goroutine keeps
   running in the background) OR when the handler finishes (a "hard" release
   via the deferred call). Because `handle` (citation 1) calls `Async`
   unconditionally and immediately for `tools/list`, the loop is released
   almost instantly and moves on to dequeue and start the NEXT request's
   goroutine — while the FIRST request's goroutine is still doing its actual
   work (looking up and serializing the tool catalog) in the background.

**What this proves:** two consecutive `tools/list` requests run in two
independently-scheduled goroutines with **no synchronization or ordering
constraint between them** — nothing in this dispatch path guarantees the
id-2 goroutine finishes (and writes its response) before the id-3 goroutine
does. The SDK is **not silent** on this — it actively documents and
implements concurrent-by-default handling of all non-`initialize` calls
(`modelcontextprotocol/go-sdk#26`, cited in the comment itself). This directly
explains the observed symptom and is the basis for VERDICT:
SERVER-EMITTED-OUT-OF-ORDER.

**Confirms `internal/mcp/server.go`'s own package doc comment is accurate**
(and orthogonal to this bug): `internal/mcp/server.go:1-14` states this
package migrated from `mark3labs/mcp-go` to `modelcontextprotocol/go-sdk` in
Phase 2 (SDK-01). The scenario comment in `test/wireoracle/scenarios.go` was
never updated to reflect that the new SDK's dispatch model differs from the
old one's (which genuinely DID handle `tools/list` synchronously — its
worker-pool was reserved for `tools/call` only, per that file's own separate,
correctly-scoped "Concurrency ordering constraint" comment at
`scenarios.go:455-471`, which is about the OLD `mark3labs` transport and
remains historically accurate for that transport, just no longer describing
what actually runs today).

`pendingWriter` itself (`internal/mcp/server.go:465-533`) was checked and
ruled out as an independent contributor: its mutex serializes the underlying
`Write` call and the response-counting classification as ONE atomic unit
(preventing byte-level interleaving/corruption within a single line), but it
makes **no promise about which goroutine's `Write` call runs first** — two
goroutines racing to acquire `pendingWriter.mu` can acquire it in either
order, and `pendingWriter` was never designed to enforce request-id ordering
(confirmed by its own doc comment, which describes only the FIX-01 counter
invariant, never an ordering invariant).

## Task 1 acceptance-criteria housekeeping

- `git diff --stat -- test/wireoracle/testdata` is empty (no frozen transcript
  touched during this investigation — verified before writing this file and
  again after landing the capture-layer instrumentation).
- `task test:wireoracle` exits 0 after the capture instrumentation landed
  (verified this session, `ok github.com/seanb4t/codegraph-go/test/wireoracle 50.143s`).
