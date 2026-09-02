---
phase: 6
reviewers: [codex]
reviewed_at: 2026-09-01T18:47:02Z
plans_reviewed:
  - 06-01-PLAN.md
  - 06-02-PLAN.md
  - 06-03-PLAN.md
  - 06-04-PLAN.md
  - 06-05-PLAN.md
  - 06-06-PLAN.md
models:
  codex: "gpt-5.6-sol (reasoning=low)"
model_sources:
  codex: "banner"
---

# Cross-AI Plan Review — Phase 6

## Codex Review

### Overall assessment

The phase is thoughtfully decomposed and unusually rigorous about non-vacuous verification. The wire fixture, message-by-message tracer, real multi-process gate, store open/close discipline, Flusher preservation, and threat-ID/DAG structure are all sound on paper and generally match the source.

However, several execution blockers remain. Most importantly:

- The watcher cannot detect first-time index creation when `.codegraph` does not exist.
- The multi-tab restart scenario cannot reconnect to a restarted `codegraph ui` because the command always chooses a new ephemeral port.
- Blocking a browser main thread does not reliably exercise server-side backpressure or coalescing.
- The final bundle staging assertion is mechanically wrong: staged changes still appear in `git status --porcelain`.
- The graph plan lacks serialization against overlapping asynchronous layout runs.

Overall risk: **HIGH until these issues are corrected.**

I verified the source directly. CodeGraph itself could not open `.codegraph/store/LOCK` under the workspace permissions, so I used read-only source inspection. I did not execute tests or mutate the repository.

---

### Plan 06-01 — Freeze the wire surface

#### Summary

This is a strong schema-freeze plan. It accurately targets the two fixture mechanisms already present in the repository and correctly delays code generation until after human approval.

#### Cross-plan strengths

- The method-count update is source-accurate. The current test hard-codes `13` in both the comparison and failure text at [readonly_test.go:90](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/readonly_test.go:90), while the fixture currently contains 13 names at [readonly_test.go:57](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/readonly_test.go:57). The plan explicitly updates all three facts.
- The seven-field descriptor extension matches the guard’s actual behavior. Direction 1 verifies each fixture entry resolves and retains its number at [readonly_test.go:520](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/readonly_test.go:520); Direction 2 rejects every unpinned descriptor field at [readonly_test.go:541](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/readonly_test.go:541).
- The new computed fixture length follows the existing compiler-checkable chain, whose current tail is `uiProtoFieldFixtureLenAtPlan0502 + 4` at [readonly_test.go:222](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/readonly_test.go:222).
- The name guard reads the live `mutatingVerbs` variable, whose current list includes both `Reindex` and `Index` at [readonly_test.go:118](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/readonly_test.go:118). The positive control prevents a zero-list false green.
- The `go test -run` checks count actual `--- PASS` lines instead of trusting exit status.
- The blocking human checkpoint is correctly placed before generated artifacts become committed API.

#### Concerns

- **LOW — Placeholder terminology is underspecified.** Task 3 says the placeholder returns a Connect “not-yet-implemented code,” but should explicitly name `connect.CodeUnimplemented` and require a test for that exact code. Otherwise a generic internal error could satisfy compilation.
- **LOW — The generated TypeScript grep is weaker than the descriptor guard.** Counting three `WatchGraph` strings proves the name exists in generated output but not necessarily that the generated method is server-streaming. The Go descriptor guards do not validate the TS method kind.

#### Suggestions

- Require a small TypeScript compile-time assertion that `uiClient.watchGraph(...)` returns an `AsyncIterable<WatchGraphEvent>`.
- Specify and test `connect.CodeUnimplemented` for the temporary handler.
- Preserve the current human checkpoint exactly; it is proportionate to the additive-only contract.

#### Risk Assessment

**LOW.** The source-backed fixture mechanisms are correctly understood, and the remaining concerns are verification refinements.

---

### Plan 06-02 — Publisher and store watcher

#### Summary

The open/read/close detector and bounded coalescing registry are well designed, but the absent-index watcher path contains a direct functional gap that prevents live push from working when the UI starts before the first index.

#### Strengths

- The plan correctly reuses `openEngine`, which is the package’s test seam at [handlers.go:26](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/handlers.go:26).
- The required close discipline matches `withEngine`: open at [handlers.go:56](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/handlers.go:56), immediate deferred close at line 60, and no retained engine field.
- `query.OpenAt` really does open the Pebble store and acquire a snapshot at [engine.go:189](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/engine.go:189), while its returned closer releases both snapshot and store. The reader contract explicitly requires closing snapshots at [store.go:36](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/graphstore/store.go:36).
- The plan’s open/close-count test has a non-zero floor, so equality cannot pass at zero.
- `ErrStoreLocked` is treated as retryable without mutating the last-known timestamp, and the negative assertion is paired with a subsequent successful publication.
- Capacity-one replacement is a good implementation of the locked coalescing decision. The tests verify the newest generation, not merely a message count.
- Registry counters are checked in both directions with non-zero controls.
- Importing the existing debouncer is reasonable. Its `Stop`/`Wait` lifecycle genuinely joins callbacks, as documented and implemented at [debounce.go:117](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/watch/debounce.go:117).

#### Concerns

- **HIGH — First-time index creation cannot be detected when `.codegraph` is absent.** The plan says that if `.codegraph` does not exist, the publisher “starts idle.” No filesystem object is then watched. Later creation of `.codegraph/store` cannot produce an event for a watcher that has no watched ancestor. This conflicts with the same task’s requirement that a UI opened before first index “must still go live.” `ResolveCodegraphDir` returns `ErrNotInitialized` once no `.codegraph` ancestor exists [resolve.go:25](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/resolve.go:25), so resolving alone cannot establish a watch.
- **MEDIUM — Store-directory replacement is not covered.** Watching the flat `store/` directory handles changes inside the existing directory, but an index rebuild or recovery path that replaces/renames the directory can invalidate the watch. The plan only discusses adding a watch when `store/` is created, not re-arming after removal or rename.
- **MEDIUM — The status payload may require more than the claimed single metadata read.** `IndexMeta()` is a direct reader lookup [engine.go:158](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/engine.go:158), but `Status()` derives staleness and several statistics from filesystem and graph scans. The plan should make explicit that this work happens only after a changed `last_sync_unix_ms`, not for every noisy fsnotify wake.
- **MEDIUM — `indexing_in_progress` is unlikely to be observed live.** While a writer owns the Pebble lock, the detector deliberately emits nothing. Once the store opens again, indexing has completed. Consequently the event’s `indexing_in_progress` field will normally remain false during the very interval it describes. This does not violate D-01, but the plan should stop implying that this field provides a real “indexing started” transition.

#### Suggestions

- When `.codegraph` is absent, watch the resolved repository root for creation/rename of `.codegraph`; then progressively re-arm onto `.codegraph` and `store/`.
- Handle `Remove`/`Rename` of the watched store path by falling back to the nearest extant parent and re-establishing the store watch when it reappears.
- Split detector work explicitly:

  1. open → `IndexMeta()` → close or continue;
  2. if timestamp changed, derive status inside that same bounded open;
  3. close before publication.

- Add a real test that starts with no `.codegraph`, creates the first index afterward, and receives an event. The current planned “start and Stop” absent-index test does not prove first-index activation.
- Document the `indexing_in_progress` limitation rather than promising a transition D-01 cannot produce.

#### Risk Assessment

**HIGH.** The primary indexed workflow works, but the first-index lifecycle is presently impossible under the specified watcher behavior.

---

### Plan 06-03 — Browser client and route subscriptions

#### Summary

The stream consumer and shared classifier are well shaped. The main weaknesses are incomplete test-file ownership and ambiguity around async supersession/coalescing across route components.

#### Strengths

- Reusing the existing single transport is source-consistent: `transport` and `uiClient` are exported at [client.ts:27](/Volumes/Code/github.com/seanb4t/codegraph-go/web/src/lib/client.ts:27).
- The incremental async-iterable test is structurally non-vacuous: delaying event 2 until callback 1 fires will deadlock a buffering implementation.
- Reconnect verification covers base growth, jitter, reset after success, resume generation, abort, and pending timers.
- Widening `classifyStatus` is appropriate. It currently reads exactly `commitSha`, `initialized`, `stale`, `storeExists`, and `indexingInProgress` at [status.ts:60](/Volumes/Code/github.com/seanb4t/codegraph-go/web/src/lib/status.ts:60).
- Applying a live event through the same classifier avoids a duplicate status representation.
- The existing status gate already has the right monotonic request pattern at [status.ts:178](/Volumes/Code/github.com/seanb4t/codegraph-go/web/src/lib/status.ts:178), so extending that counter to live observations is coherent.
- Route-level request coalescing prevents the client from undoing server-side coalescing.

#### Concerns

- **MEDIUM — Task 3 requires component tests but does not list their files.** The plan requires mounted Health and Workbench tests, yet `files_modified` lists only `live-client.test.ts` and `live-store.test.ts`. Neither is an obvious home for route/component tests. This creates either hidden scope expansion or acceptance criteria that cannot be implemented cleanly.
- **MEDIUM — Supersession semantics need one exact invariant.** The current gate drops a response when its request ID is no longer current [status.ts:195](/Volumes/Code/github.com/seanb4t/codegraph-go/web/src/lib/status.ts:195). A live event arriving after a unary request started must invalidate that request. Conversely, a unary request started after the live event may supersede it. “A later in-flight unary fetch and vice versa” is too ambiguous to guarantee this ordering.
- **MEDIUM — Coalescing follow-up behavior is underspecified.** “Let the newest generation win when the in-flight one settles” requires a dirty/pending-generation flag and a second fetch after settlement. Merely suppressing concurrent fetches would permanently lose the newest event.
- **LOW — Clean stream termination reconnects immediately.** The plan says a stream ending cleanly is restarted but only explicitly delays thrown failures. Repeated clean EOF could become a tight reconnect loop.

#### Suggestions

- Add the actual route/component test files to `files_modified`.
- Define the ordering rule precisely: every fetch start and every applied live event increments one shared observation counter; only the operation holding the current ID may emit.
- Require the route coalescer to store `pendingGeneration = max(...)` and start one follow-up request after the active request settles if a newer generation arrived.
- Apply backoff to unexpected clean EOF too. Only an explicit client stop should terminate without retry.
- Add a test for repeated clean EOF to ensure it does not reconnect in a tight loop.

#### Risk Assessment

**MEDIUM.** The architecture is sound, but race semantics and test ownership need tightening before execution.

---

### Plan 06-04 — Streaming transport and handler

#### Summary

The transport plan correctly addresses both the absolute write deadline and the Connect Flusher trap. Its end-to-end tracer is one of the strongest parts of the phase. Lifecycle ownership and disconnect classification need sharper specification.

#### Strengths

- The Flusher requirement is handled correctly. The existing origin guard passes the original writer unchanged at [originguard.go:65](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/originguard.go:65), and the plan explicitly requires the new middleware to do the same.
- Matching the generated procedure constant avoids path drift.
- The server’s existing 60-second bound is correctly identified at [server.go:46](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/server.go:46), and the plan retains it for unary and SPA routes.
- The small-deadline positive and sibling-path negative test prove path scoping rather than merely successful streaming.
- The tracer’s ordering is correct: receive message \(k\) before issuing write \(k+1\). A buffered stream cannot reach the next write, so it fails structurally rather than through a post-hoc timing heuristic.
- The handler is explicitly prohibited from opening the store. That cleanly separates subscriber writes from the short-lived detector snapshot.
- Subscriber count assertions include both an increase and return to baseline.
- Same-wave file ownership does not overlap with Plan 06-05.

#### Concerns

- **MEDIUM — Publisher lifetime cannot currently be “tied to the server context” inside `Listen`.** `Listen` has no context parameter and currently returns a `Server` containing only listener, HTTP server, and URL [server.go:74](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/server.go:74). The request context is only created later in CLI code before `Serve` [ui.go:74](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/cli/ui.go:74). Starting the publisher in `Listen` therefore needs an explicit owned cancel function and cleanup if `Serve` is never called.
- **MEDIUM — Disconnect error handling is oversimplified.** `mapContextError` correctly maps an already-cancelled request [handlers.go:68](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/handlers.go:68), but `stream.Send()` may return a broken-pipe or reset error before `ctx.Err()` becomes visible. Passing that raw error through `mapContextError` would not necessarily classify it as cancellation.
- **LOW — Middleware ordering should be specified as a concrete expression.** “Origin/host guard still runs before any handler” can be satisfied by either outer order, but it would be clearer to require `originHostGuard(port, clearWatchDeadline(mux))`, preserving security rejection before deadline mutation.

#### Suggestions

- Add `publisher` and `publisherCancel` to `Server`, or start the publisher in `Serve(ctx)` while keeping the already-constructed service wired to it. Require `Server.Close`/failed-Serve cleanup if such an API exists.
- Add a test for `Listen` followed by listener close without `Serve`, proving no watcher goroutine survives.
- For send failures:

  - if `req.Context().Err() != nil`, return `mapContextError(req.Context().Err())`;
  - otherwise return a Connect-classified transport error without claiming cancellation.

- Lock down the wrapper expression in the plan so the origin guard remains outermost.
- Keep the structural message-by-message tracer unchanged.

#### Risk Assessment

**MEDIUM.** The wire mechanism is correct, but server-owned background lifetime needs a concrete design.

---

### Plan 06-05 — Graph stability

#### Summary

The two-path graph strategy correctly implements D-06: no layout for data-only changes and authoritative leaf write-back for structural changes. The principal risk is overlapping asynchronous layouts, which the current renderer does not serialize.

#### Strengths

- The plan accurately identifies the existing shared layout configuration at [GraphCanvas.svelte:117](/Volumes/Code/github.com/seanb4t/codegraph-go/web/src/lib/components/graph/GraphCanvas.svelte:117).
- The no-layout path has a real precedent: `removeByIds` mutates the model and republishes geometry without layout at [GraphCanvas.svelte:378](/Volumes/Code/github.com/seanb4t/codegraph-go/web/src/lib/components/graph/GraphCanvas.svelte:378).
- Authoritative write-back correctly targets non-parent nodes. Current geometry already distinguishes compounds through child collections at [GraphCanvas.svelte:165](/Volumes/Code/github.com/seanb4t/codegraph-go/web/src/lib/components/graph/GraphCanvas.svelte:165).
- The survivor tests have non-empty floors.
- New-node placement has positive assertions, preventing an origin-stacked implementation from passing.
- The guava measurement binds leaf displacement, includes a survivor floor, requires at least one added node, and records non-binding compound displacement.
- The red demonstration with write-back disabled is a strong discrimination check.
- The measurement restores the corpus and asserts cleanliness.

#### Concerns

- **HIGH — Overlapping layout runs are not serialized or generation-guarded.** `runLayout` registers a one-shot `layoutstop` callback and starts an asynchronous layout at [GraphCanvas.svelte:251](/Volumes/Code/github.com/seanb4t/codegraph-go/web/src/lib/components/graph/GraphCanvas.svelte:251). `replace` can immediately start another layout at [GraphCanvas.svelte:315](/Volumes/Code/github.com/seanb4t/codegraph-go/web/src/lib/components/graph/GraphCanvas.svelte:315), as can symbol `add` at line 346. A live refresh racing a user expansion can therefore leave multiple callbacks restoring different captured positions and publishing stale geometry.
- **MEDIUM — “Byte-identical positions” is stronger than appropriate for JS numbers.** Positions are numeric values, not bytes. Exact `===` equality is appropriate after explicit write-back; byte terminology may encourage brittle serialization comparisons.
- **MEDIUM — The real-browser mutation may alter far more than one node.** Adding a source file and reindexing can introduce multiple symbol/file nodes and edge changes. The record should state the actual mutation and ensure the selected view truly exercises the intended file-level structural path.
- **LOW — The script’s cleanup must be trap/finally based.** The plan says the corpus is restored but does not explicitly require cleanup in a `finally` block before every failure exit.

#### Suggestions

- Introduce a layout generation token:

  - increment before every layout;
  - capture the token and survivor map;
  - ignore stale `layoutstop` callbacks;
  - optionally stop/cancel a prior layout if Cytoscape supports it.

- Add tests for:

  - live structural update followed immediately by another live update;
  - live update racing symbol expansion;
  - stale `layoutstop` unable to overwrite newer geometry.

- Say “exact numeric coordinate equality” rather than byte identity.
- Require cleanup through `try/finally` and write the failure record only after cleanup status is known.
- Record pre/post file-level node counts so the measurement proves it exercised the binding view.

#### Risk Assessment

**HIGH.** The core write-back idea is correct, but unguarded overlapping layouts can reintroduce movement or stale geometry in normal interaction.

---

### Plan 06-06 — Real-world gates and bundle

#### Summary

The criterion-5 pairing is strong and genuinely non-vacuous. However, the multi-tab reconnect/backpressure scenarios do not match browser and CLI behavior, and the bundle staging assertion will fail whenever the build actually changes.

#### Strengths

- Criterion 5 is correctly paired:

  - `flushesStarved == 0`;
  - `flushesCompleted == flushesAttempted`;
  - at least five attempted/completed flushes;
  - at least five delivered live events;
  - all three processes alive.

  This prevents the zero-work false green.
- The MCP liveness check requires an answered request rather than only a PID.
- The criterion-2 browser scenario waits for each tab to receive generation \(k\) before triggering \(k+1\), so delayed whole-stream buffering cannot pass.
- Reconnect attempts are bounded both above and below.
- The real-process criterion uses the actual daemon, MCP server, UI server, and shared store. The source confirms `daemon start` is the real foreground watcher/indexer at [daemon.go:113](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/cli/daemon.go:113), while `serve --mcp` performs startup reconciliation at [serve.go:220](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/cli/serve.go:220).
- The lifecycle verdict uses a positive control against the real `pendingWriter` implementation at [server.go:460](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/mcp/server.go:460).
- Threat IDs `T-06-01` through `T-06-38` are unique within the threat registers. The only accepted risks are low/medium severity, not high.
- The wave DAG has no same-wave `files_modified` overlap:

  - Wave 2: Go publisher vs browser client.
  - Wave 3: Go transport vs graph UI.

#### Concerns

- **HIGH — Restarting `codegraph ui` changes its URL.** The CLI always binds an ephemeral loopback port and exposes no port flag [ui.go:25](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/cli/ui.go:25). Stopping and restarting the process therefore gives the tabs a different origin. Their reconnect loop will continue calling the dead old URL and can never reach the new process. The planned restart scenario cannot pass as written.
- **HIGH — Blocking a tab’s main thread does not reliably create server-side backpressure.** Browser networking and socket buffering continue outside page JavaScript. A few tiny Connect messages can be read into browser/network buffers while the page callback is blocked. When JavaScript resumes, it may receive every intermediate event, so skipped generations do not prove—or reliably exercise—the publisher’s capacity-one channel.
- **HIGH — The bundle staging assertion is incorrect.** After `git add web/build`, `git status --porcelain web/build` reports staged changes until they are committed. The planned command:

  ```sh
  git add web/build &&
  test -z "$(git status --porcelain web/build)"
  ```

  fails precisely when the rebuilt bundle differs. To prove there are no unstaged build changes, use `git diff --quiet -- web/build`; separately prove staged content exists or matches expectations with `git diff --cached`.
- **MEDIUM — The raw curl stream is underspecified.** Connect streaming requires a correctly framed POST request and envelope decoding. Merely curling the procedure with Host/Origin headers is insufficient to count messages or generations. The plan needs an exact request body/content type and decoder.
- **MEDIUM — The MCP process harness is non-trivial and currently vague.** `serve --mcp` is stdio-only [serve.go:179](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/cli/serve.go:179). Holding it open and later issuing a real request requires persistent bidirectional pipe management, not a one-shot shell invocation.
- **MEDIUM — “Daemon alive” should refer to `daemon start`, not `daemon start` as if it daemonizes.** The source confirms it is a foreground process [daemon.go:124](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/cli/daemon.go:124). The script must background that foreground process itself and retain its PID.
- **LOW — The positive-control search in the final automated command is too broad.** `rg -c pendingWriter internal/mcp/server.go` counts comments and type references. It proves the file contains the word, but not that the search used for the no-analogue verdict discriminates on the actual counter shape.

#### Suggestions

- Test reconnect without changing the listener address. Viable shapes include:

  - add a test-only fault endpoint/seam that closes active streams while leaving the HTTP listener alive;
  - place a local proxy on a stable port and restart the upstream UI behind it;
  - run an in-process server with a fixed `Options.Addr` in a dedicated Playwright harness.

  Do not restart the released CLI and expect an ephemeral URL to remain stable.
- Separate browser event-loop delay from server backpressure. For real server-side pressure, use a client that opens the stream and deliberately stops reading the response body, then publish enough data/events to exceed transport buffers. Keep the browser three-tab check for fan-out and reconnect.
- Replace the staging assertions with:

  ```sh
  git add web/build
  git diff --quiet -- web/build
  test -n "$(git diff --cached --name-only -- web/build)"
  ```

  If the task commits before final verification, then verify both staged and unstaged diffs are empty after that commit.
- Use a small purpose-built Go Connect client for the concurrency script rather than raw curl. It can decode envelopes, record receipt times, and preserve a persistent stream accurately.
- Use a dedicated MCP harness that keeps stdin/stdout pipes open, performs initialization, calls one tool, and validates the response after the concurrency run.
- Make the lifecycle positive control target the declaration or mutation mechanism, such as `type pendingWriter struct` plus its pending field/write path, rather than a bare word count.

#### Risk Assessment

**HIGH.** Criterion 5 itself is well specified, but three concrete gate mechanisms—the restart test, slow-tab backpressure claim, and staging assertion—are currently non-functional or non-discriminating.

---

### Cross-plan findings

#### Strengths

- The six-plan dependency graph is coherent and has no same-wave file overlap.
- Criterion 2 is tested structurally at both Go and browser levels, not through eventual arrival counts.
- Criterion 5 pairs zero starvation with non-zero work and live-event floors.
- The Flusher trap is explicitly handled through original-writer pass-through.
- Store access is separated from handler writes and bounded by immediate close.
- Proto fixture coverage is complete in both directions.
- The plans consistently count actual test cases and use positive controls for most negative assertions.
- Threat IDs are sequential and unique in their registers; no high-severity threat is accepted.

#### Highest-priority corrections before execution

1. Fix absent-`.codegraph` watcher activation in 06-02.
2. Define publisher/server lifecycle ownership in 06-04.
3. Serialize or generation-guard graph layouts in 06-05.
4. Replace the ephemeral-port restart scenario in 06-06.
5. Replace the browser-main-thread backpressure test with a client that stops reading the transport.
6. Correct the `git status --porcelain` staging check.
7. Add the missing route/component test files to 06-03 ownership.

#### Final Risk Assessment

**HIGH.** The implementation architecture is fundamentally viable and the plans contain excellent verification discipline, but several high-severity test and lifecycle defects would either block execution outright or produce misleading evidence. Correcting the seven items above should reduce the phase to **MEDIUM**, with most remaining uncertainty concentrated in asynchronous graph layout behavior and the real multi-process harness.

---

## Consensus Summary

One prompt-fed, source-grounded lane ran this cycle (`codex`, `gpt-5.6-sol`, reasoning=low). Its
review carries `file:line` evidence throughout and does not carry the
`[reviewed-without-repo-access]` or `[reviewed-without-source-citations]` markers, so it is
weighted as a grounded plan review. With a single lane there is no cross-reviewer agreement to
measure, so every finding below was **independently re-verified against the source by the review
orchestrator** before being recorded — confirmed, refuted, or corrected. Findings marked
**[orchestrator-verified]** were reproduced with a command whose output is quoted; findings marked
**[orchestrator-only]** were not raised by the lane at all.

### Agreed Strengths

Confirmed against source, not merely restated from the plans:

- **The Flusher trap is genuinely handled.** `connect@v1.20.0/protocol.go:347-353`'s
  `checkServerStreamsCanFlush` is a bare type assertion with no `Unwrap()` traversal, so a wrapper
  struct really would break it; `06-04-PLAN.md:79-89` requires `next.ServeHTTP(w, r)` with the same
  `w`, and `:111` asserts the inner handler's writer satisfies the flusher interface — a positive
  assertion, not prose. `internal/uiserver/originguard.go:65` is the shape being copied, and the
  repo has no other `ResponseWriter` wrapper on the RPC path. **[orchestrator-verified]**
- **Criterion 5's pairing is real and mechanized.** `06-06-PLAN.md:183` asserts
  `flushesAttempted >= 5`, `flushesCompleted === flushesAttempted`, `flushesStarved === 0` and
  `liveEventsReceived >= 5` in one command, with a mandated RED run (`:177-180`) that holds the
  store open and confirms non-zero starvation. The zero is paired with non-zero floors on both work
  attempted and events delivered, so "nothing was starved" cannot be satisfied by nothing having
  happened.
- **Snapshot discipline is the plan's strongest single constraint.** `06-02-PLAN.md:92-98` mandates
  the existing `openEngine` seam, a deferred `Close`, read, return — matching
  `internal/uiserver/handlers.go:26-66`'s `withEngine` and `internal/graphstore/store.go:36-49`'s
  "must be Closed … to release its underlying snapshot". `06-04-PLAN.md:143-145` forbids the handler
  from opening the store at all, and `:172` enforces it with a zero-count grep paired with a
  `Subscribe >= 1` positive. The open/close balance gate (`06-02-PLAN.md:123`) asserts both
  `opens == closes` **and** `opens >= 2`. No snapshot is held across a stream or across a blocking
  client write.
- **Criterion 2's Go-level tracer is structurally non-vacuous.** `06-04-PLAN.md:199-207` blocks on
  receipt of message *k* before triggering *k+1*, and `:223-224` explicitly forbids a post-hoc
  timestamp comparison, pairing a floor of 3 messages with the inter-arrival bound. A buffering
  implementation times out rather than passing.
- **`go test -run` exit status is never trusted.** Every `-run` invocation in 06-01/06-02/06-04
  redirects to a log and counts `^--- PASS:` lines against a non-zero floor with `^--- FAIL` at
  zero. `06-02-PLAN.md:122` states the hazard verbatim.
- **`WatchGraph` is clean against the live fixture.** All 19 `mutatingVerbs`
  (`readonly_test.go:118-122`) checked by substring: no hit. The anti-emptying positive control
  (`Reindex` still present) appears twice — `06-01-PLAN.md:86` and `:131`. **[orchestrator-verified]**
- **Threat register and wave DAG are coherent.** 38 numbered ids `T-06-01`…`T-06-38` plus `T-06-SC`,
  contiguous, zero duplicates across plans (the 40-vs-39 occurrence delta is `T-06-SC` cited once in
  06-06's verification block). Three `accept` rows, all at low/medium
  (`06-01:271`, `06-03:261`, `06-04:252`); every high and critical row is `mitigate`. No same-wave
  `files_modified` overlap: wave 2 and wave 3 each split cleanly Go/web. **[orchestrator-verified]**
- **No plan greps the un-hoisted `web/node_modules/elkjs/` path.** The only `node_modules` mention
  across all six plans is `06-05-PLAN.md:118-123`, which is the warning itself and mandates a paired
  control term. No `dispatchEvent` gesture faking anywhere; real gestures go through Playwright.
  **[orchestrator-verified]**
- **Leaf-only write-back is correctly specified.** `06-05-PLAN.md:113-116` targets non-parent nodes
  only for the stated cytoscape reason, and the binding gate is max **leaf** displacement with
  compounds recorded but explicitly non-binding (`:220-222`, `:229`).

### Agreed Concerns

Ordered by severity. Every HIGH below was reproduced against source.

**HIGH**

1. **`TestKnownUIProtoFieldNumbersAreStable` does not exist.** The real name is
   `TestUIProtoFieldNumbersAreStableAndUnique` (`internal/uiserver/readonly_test.go:482`); the
   plan's name appears to be borrowed from `internal/schema/meta_commit_test.go:120`. It is cited at
   `06-01-PLAN.md:226`, `:242` (the `<automated>` verify), `:245`, `:269` (threat T-06-02's
   mitigation) and `06-VALIDATION.md:68` — where it is also marked "File Exists ✅". Consequences:
   the `-ge 4` PASS floor is unreachable (max 3 real tests match the pattern), so Task 3 is
   unpassable as written; and the descriptor guard — the **sole** enforcer of the seven pinned
   fields and the computed length constant — is never executed by the task that adds them. Only the
   phase-level `task test:unit` runs it. **[orchestrator-verified]**
2. **The publisher cannot activate on a first index, and never re-arms.**
   `06-02-PLAN.md:211-217` says that when `.codegraph` is absent "the publisher starts idle and
   publishes nothing" — nothing is watched, so the store's later creation cannot produce an event,
   directly contradicting the same paragraph's "a UI opened before the first index must still go
   live when one appears". The acceptance criterion at `:240` bakes the deafness in: it asserts only
   that the publisher "starts and `Stop` returns … publishing nothing". Two further gaps: the
   `.codegraph`-parent fallback is non-recursive and the plan never says to `Add(storeDir)` once the
   store appears; and `06-02-PLAN.md:196`/`:206-208` explicitly forbid copying `internal/watch`'s
   re-`Add`-on-Create machinery without providing any handling for `Remove`/`Rename` of the watched
   directory, so a store rebuilt by rename leaves the publisher permanently deaf, untested.
3. **`06-05` has an undeclared dependency on `06-04` and they share a wave.** `06-05`'s frontmatter
   is `wave: 3, depends_on: ["06-03"]`; `06-04` is also `wave: 3`. But `06-05-PLAN.md:206-214`
   Task 3 starts a real `codegraph ui`, causes a real re-index and "waits for the live update to
   reach the page" — traffic that exists only after `06-04` replaces `livehandler.go`'s placeholder
   and wires the publisher in (`06-04-PLAN.md:139-147`). Run in parallel as the wave permits, 06-05
   Task 3 waits forever. Fix: `depends_on: ["06-03", "06-04"]`. **[orchestrator-verified]**
4. **The bundle staging assertion is mechanically inverted.**
   `git add web/build && test -z "$(git status --porcelain web/build)"` (`06-06-PLAN.md:242`,
   `:246`, `:284`, threat T-06-36 at `:274`). Reproduced in a scratch repo: after `git add`,
   `git status --porcelain web/build` prints `M  web/build/f.txt` and the `test -z` **FAILS**. So the
   gate fails precisely when the rebuilt bundle differs — the case it exists to cover — and passes
   only when nothing changed, where it proves nothing about staging. Fix:
   `git diff --quiet -- web/build` for "nothing unstaged", plus
   `test -n "$(git diff --cached --name-only -- web/build)"` for "the bundle really is staged".
   **[orchestrator-verified]**
5. **The multi-tab RECONNECT scenario cannot pass.** `06-06-PLAN.md:105` says "Stop and restart the
   server process while the tabs are open", but `codegraph ui` binds an ephemeral loopback port and
   exposes no bind/port flag — `internal/cli/ui.go:26` ("no flag for a bind address, port,
   hostname"), `:40` ("binds an ephemeral loopback port"), and `:81-82` (only `--path` and
   `--no-open`). The restarted process has a different origin; the tabs' reconnect loop keeps
   calling the dead URL, so `:108`'s "every tab is receiving events again afterwards" can never be
   satisfied. Needs a stable-address harness (test-only stream-fault seam, a fixed-port proxy, or an
   in-process server with an explicit `Options.Addr`). **[orchestrator-verified]**
6. **The browser-level criterion-2 pairing exists only in prose.** `06-06-PLAN.md:120` claims
   `minInterArrivalMs >= triggerSpacingMs` "is paired with a per-tab received-count equal to
   `generationsTriggered`". The verify command at `:114` reads nine fields and none of them is a
   per-tab received count, and `:117`'s required-record-field list does not name one either. With no
   per-tab receipt floor, an inter-arrival minimum computed over a single receipt is vacuously
   satisfied — the exact shape the plan's own prose (`:93`) calls vacuous. Fix: add `receivedPerTab`
   to the record and assert `Math.min(...r.receivedPerTab) === r.generationsTriggered`.
   **[orchestrator-verified]**
7. **Reconnect "resume" silently loses generations.** `06-04-PLAN.md:133`: a resume cursor greater
   than zero "does not replay history (there is none to replay) … the next real event is the next
   thing the client sees"; the handler sends nothing at subscribe (`:139-141`) and the publisher
   emits only on change (`06-02-PLAN.md:79-80`). A tab disconnected across a generation bump stays
   stale until the *next* change — indefinitely for a parked tab. This contradicts `06-03-PLAN.md:29`
   ("resumes from the last generation it saw") and defeats criterion 1's "open views stop going
   quietly stale". Cheap fix with no history store: send the publisher's current last-known state
   once at subscribe, or when `cursor < currentGeneration` — the publisher already holds that value
   (`06-02-PLAN.md:112-114`). **[orchestrator-verified]**
8. **Overlapping layout runs are not serialized or generation-guarded.** `runLayout` registers a
   one-shot `layoutstop` handler at
   `web/src/lib/components/graph/GraphCanvas.svelte:251-252`, and the initial paint (`:289`),
   `replace()` (`:329`) and `add()` (`:354`) each start a layout unconditionally with no in-flight
   guard or generation token. A
   live refresh racing a user expansion leaves multiple callbacks restoring different captured
   survivor maps and publishing stale geometry — reintroducing exactly the movement D-06's
   write-back exists to eliminate. 06-05 adds no serialization. **[orchestrator-verified]**
9. **The BACKPRESSURE scenario does not exercise server-side backpressure.** Occupying a tab's main
   thread (`06-06-PLAN.md:97-98`) does not stop the browser and kernel from draining a handful of
   small Connect envelopes, so `slowTabFinalGeneration === newestGeneration` passes whether or not
   the capacity-1 coalescing channel ever engaged. The "skipped count as positive evidence"
   (`:102`, T-06-34 at `:272`) is required in the record but never asserted non-zero by the verify
   command at `:114`. Criterion 3's backpressure clause is therefore unproven either way. A client
   that opens the stream and deliberately stops reading the response body is the shape that creates
   real transport pressure; keep the three-tab check for fan-out and reconnect.

**MEDIUM**

10. **`06-01-PLAN.md:133`'s floor is contaminated.** The command
    `rg -v '^\s*//' internal/uiproto/uiv1/ui.proto | rg -c 'since_generation = 1;|int64 generation = 1;|initialized = 2;|stale = 3;|store_exists = 4;|indexing_in_progress = 5;|commit_sha = 6;'`
    returns **1** on the current, unmodified tree — `ui.proto:563 bool stale = 3;`
    (`ExploreResponse.stale`) already matches. So `-ge 7` is satisfied with only six of the seven new
    declarations present. Compounding: `rg -c` counts matching *lines*, not matches. Anchor the
    alternatives to their message blocks or count each field separately. **[orchestrator-verified]**
11. **`06-01-PLAN.md:184`'s second conjunct is unconditionally true.**
    `test ! -f internal/uiproto/uiv1/ui.pb.go.orig` — nothing in this plan, in `task proto:gen`, or
    anywhere in the repo ever creates a `.orig` file (that is a merge/patch artifact). It stands in
    for "no generated artifact was written before approval", whose real command
    (`git status --porcelain` emptiness) appears only in the human `how-to-verify` at `:150`, never
    in `<automated>`. **[orchestrator-verified]**
12. **`06-01-PLAN.md:248` guards the constant declaration, not the fixture entries.**
    `rg -c 'uiProtoFieldFixtureLenAtPlan0506 \+ 7' … exactly 1` proves a constant was written and
    says nothing about the seven `uiProtoFieldNumbers` rows. The only thing that counts rows is
    `len(uiProtoFieldNumbers) != …` at `readonly_test.go:483` — inside the test concern 1 shows is
    never run. Second defect: `exactly 1` is brittle against the file's own convention of restating
    the arithmetic in a doc comment (`readonly_test.go:165`), which would make the count 2.
13. **`06-01-PLAN.md:249` is not checkable by the command it names.** `git diff --stat` prints a
    filename and insertion/deletion counts; it cannot show that changes landed "ONLY inside the
    fixture bodies and the length constants". The paired `"Reindex"` control is genuine; the
    location clause is prose.
14. **`06-04-PLAN.md:73` cites the wrong lines.** It points the executor at
    `internal/uiserver/originguard.go` lines 37-51 for the pass-through shape, but
    `next.ServeHTTP(w, r)` is at `originguard.go:65` — outside the cited range. Lines 37-51 are the
    doc-comment tail, the signature, the allowlist lookups and the host check.
15. **The anti-wrapper rule has no mechanical guard and the tracer has no RED.**
    `06-04-PLAN.md:89` ("do not introduce any writer type of your own in this file") is prose only —
    unlike the procedure-constant grep at `:112` and the no-store grep at `:172`, there is no
    negative grep for a writer type declaration in `watchtimeout.go`. And unlike Task 1's
    middleware-removal RED (`:109`) and 06-06 T2's store-held-open RED (`:177-180`), Task 3 mandates
    no RED demonstration against a deliberately buffering implementation.
16. **Publisher lifetime cannot be "tied to the server context" inside `Listen` as written.**
    `Listen` takes no context and `Server` holds only listener, HTTP server and URL
    (`internal/uiserver/server.go:74`); the context is created later in
    `internal/cli/ui.go:74` before `Serve`. Starting the publisher in `Listen` needs an owned cancel
    func plus cleanup when `Serve` is never called, and a test that `Listen` followed by listener
    close leaves no watcher goroutine.
17. **Disconnect classification is oversimplified.** `stream.Send()` can return a broken-pipe or
    reset error before `req.Context().Err()` becomes visible; passing that raw error through
    `mapContextError` (`internal/uiserver/handlers.go:68`) would not classify it as cancellation.
    Branch on `req.Context().Err() != nil` first.
18. **06-03 Task 3's component tests have no declared home.** The task requires mounted Health and
    Workbench tests, but `files_modified` lists only `web/tests/live-client.test.ts` and
    `live-store.test.ts`. Either hidden scope expansion or an unimplementable acceptance criterion.
19. **06-03's supersession and coalescing semantics need one exact invariant.** The current gate
    drops a response whose request id is no longer current (`web/src/lib/status.ts:200` and `:204`,
    `if (id !== requestId) return;`); "a later
    in-flight unary fetch and vice versa" does not pin the ordering. And "let the newest generation
    win when the in-flight one settles" requires a `pendingGeneration = max(...)` flag plus a
    follow-up fetch — mere suppression permanently loses the newest event.
20. **`06-05-PLAN.md:132` asserts a layout property in jsdom against a mocked engine.** The task is
    pinned to headless jsdom (`:88-90`), and the repo's precedent
    `web/tests/graph-expansion.test.ts` mocks `cytoscape-elk` (`:120`) with a `FakeCore` (`:40`)
    whose `layout()` (`:73`) computes nothing and only fires `layoutstop`. "Proving they were laid
    out, not stacked" is satisfiable by a stub returning two arbitrary distinct coordinates. Demote
    to "the new nodes were handed to the layout call" and let Task 3's real-browser
    `addedNodeCount` carry the claim.
21. **The full-suite floor proves nothing about the new tests.** `06-03-PLAN.md:231` and
    `06-05-PLAN.md:170` both run the whole vitest suite (already 38 files under `web/tests/`)
    against `numTotalTests < 1`. That cannot distinguish "my new file was collected" from "one
    pre-existing test ran". Assert the new file appears in `testResults`, or floor at the current
    total plus the new tests. **[orchestrator-verified]**
22. **`06-06-PLAN.md`'s real-process harness is underspecified in three ways.** `serve --mcp` is
    stdio-only (`internal/cli/serve.go:179`), so holding it open and later issuing a real request
    needs persistent bidirectional pipe management, not a one-shot invocation; `daemon start` is a
    foreground process (`internal/cli/daemon.go:124`), so the script must background it and retain
    the PID; and a raw `curl` against the streaming procedure cannot count messages without correct
    Connect framing and envelope decoding. A small purpose-built Go Connect client is the
    lower-risk shape.
23. **`maxFlushDurationMs` is recorded but bounded by nothing** (`06-06-PLAN.md:166-167`), and
    "starved" has no operational definition beyond "whether any store-lock contention was reported"
    (`:161-162`). A severely degraded but eventually-completing flush scores a clean pass.
24. **`reconnectDelaysGrowing` and `reconnectDelaysDistinctAcrossTabs` are self-reported booleans.**
    `06-06-PLAN.md:122` requires the raw delay arrays "so the claim is auditable", but the verify
    command at `:114` never recomputes growth or distinctness from them.
25. **`06-02`'s status derivation is not scoped.** `IndexMeta()` is a direct reader lookup
    (`internal/query/engine.go:158`), but `Status()` derives staleness and statistics from
    filesystem and graph scans. State explicitly that this work runs only after a changed
    `last_sync_unix_ms`, not on every noisy fsnotify wake.
26. **`indexing_in_progress` will normally read false in the live event.** While a writer owns the
    Pebble lock the detector deliberately emits nothing; by the time the store reopens, indexing has
    finished. This does not violate D-01, but the plan should document the limitation rather than
    implying an "indexing started" transition.

**LOW**

27. **`06-05-PLAN.md:134` is green before any work is done and cannot go red for what it claims.**
    `rg -v '^\s*//' GraphCanvas.svelte | rg -c 'LAYOUT_OPTIONS'` returns **2** on the current,
    unmodified file, so `-ge 2` already passes. Worse, a forked `LAYOUT_OPTIONS_LIVE` would *raise*
    the count, not lower it — the guard cannot detect the fork it names. It is also the only guard
    in these plans missing `|| echo 0`. Assert the absence of a second config
    (`rg -c 'LAYOUT_OPTIONS[_A-Z]' … -eq 0`) paired with the existing positive.
    **[orchestrator-verified]**
28. **`corpusCleanAtExit` is self-attested** (`06-05-PLAN.md:229`, `:238`, T-06-30 at `:260`) — the
    script that mutates the pinned guava checkout computes and reports its own cleanliness, and the
    verify only reads the field back. Add an independent
    `test -z "$(git -C <corpus> status --porcelain)"`.
29. **`06-05-PLAN.md:229` prints nothing on failure.** The `&&` chain means the reporting `node -e`
    never runs if the check script exits non-zero, so `:233`'s "prints the four load-bearing numbers
    BEFORE comparing any of them" holds only on the green path. Use `;`, or print inside the script.
30. **"Do not rebuild `web/build`" is contradicted by the same plans' own verify commands.**
    `06-03-PLAN.md:228` and `06-05-PLAN.md:167` say the bundle is 06-06's job, yet `:231` and `:170`
    both end with `pnpm run build` — `vite build` with `adapter-static`, which writes `web/build`,
    a file declared in 06-06's `files_modified`. Waves 2 and 3 will dirty it. Use
    `pnpm exec svelte-kit sync && pnpm run check`, or build to a scratch `--outDir`, if the intent
    is compile-only proof. **[orchestrator-verified]**
31. **Machine-specific absolute paths in committed verify commands.** `06-03-PLAN.md:181`, `:231`
    and `06-05-PLAN.md:170` each contain `cd /Volumes/Code/github.com/seanb4t/codegraph-go/web`,
    while the same lines already use a relative `cd web` earlier. **[orchestrator-verified]**
32. **The lifecycle positive control is too broad.** `rg -c 'pendingWriter' internal/mcp/server.go`
    (`06-06-PLAN.md:242`, `:247`) returns 9 today, counting comments and type references. It proves
    the file contains the word, not that the search discriminates on the counter shape the verdict
    is about. Target the declaration or the mutation path. **[orchestrator-verified]**
33. **`06-01-PLAN.md:132` does not strip comments though its sibling `:133` does** — a commented-out
    rpc signature would satisfy it with the real declaration missing.
34. **Task 3's placeholder error code is underspecified** (`06-01-PLAN.md`): name
    `connect.CodeUnimplemented` explicitly and test for that exact code, or a generic internal error
    satisfies compilation.
35. **The generated-TypeScript grep does not prove the method kind.** Counting three `WatchGraph`
    strings in `web/src/lib/gen/ui_pb.ts` (`06-01-PLAN.md:247`) could be satisfied by the two
    message schemas alone. A TS compile-time assertion that `uiClient.watchGraph(...)` returns an
    `AsyncIterable<WatchGraphEvent>` would close it.
36. **Two stale prose sites in `readonly_test.go` are not scheduled for update.** `:77` says "same
    length (13, as of plan 05-06's FileSymbols)" inside
    `TestUIServiceMethodSetIsExactlyTheReadSet`'s own doc comment — the plan's comment instruction
    (`06-01-PLAN.md:223-224`) targets the `wantUIServiceMethods` fixture comment, not this one — and
    `:104` still says "the ten-name read-only fixture", already three waves stale.
    **[orchestrator-verified]**
37. **`addedNodeCount >= 1` is the weakest possible structural perturbation** on a 3,233-node graph
    (`06-05-PLAN.md:229`, `:235`). The `leafSurvivorCount >= 100` floor carries the real weight;
    consider a floor of ~5 added nodes.
38. **Middleware ordering should be pinned as an expression.** `06-04-PLAN.md:154`'s "origin/host
    guard still runs before any handler" is satisfiable by either nesting; require
    `originHostGuard(port, clearWatchDeadline(mux))` so security rejection precedes deadline
    mutation.
39. **"Byte-identical" positions (`06-05-PLAN.md:77`) is the wrong vocabulary** — positions
    are JS numbers. Say exact numeric coordinate equality, to avoid encouraging brittle
    serialization comparisons.
40. **06-03 restarts a cleanly-ended stream immediately.** Only thrown failures are explicitly
    delayed, so repeated clean EOF could become a tight reconnect loop. Apply backoff to unexpected
    clean EOF and test it; only an explicit client stop should terminate without retry.

### Divergent Views

- **Read-deadline clearing.** No reviewer raised it, and the orchestrator investigated whether
  `readTimeout = 30 * time.Second` (`internal/uiserver/server.go:48`) would kill a long-lived stream
  the way the write deadline does. It does **not**: `net/http` clears the read deadline itself at
  `startBackgroundRead` (`$GOROOT/src/net/http/server.go:697`, `cr.rwc.SetReadDeadline(time.Time{})`)
  once the request body reaches EOF, which for a Connect server-streaming request happens
  immediately. D-02's write-deadline-only middleware is therefore sufficient, and no plan change is
  needed — but the verdict is worth recording in the SUMMARY under this phase's own
  "record it rather than assume it" discipline, because it is the obvious next question a reviewer
  asks. **[orchestrator-only]**
- **The lane's `06-01` risk rating (LOW) versus the orchestrator's (HIGH).** Codex rated 06-01 LOW,
  having correctly verified the two `13` literals, the seven-field pinning and the computed length
  constant against source. It did not check whether the test name those mechanisms are invoked
  under actually exists. It does not (concern 1), which makes the plan's central fixture gate
  unrunnable. This is the single largest divergence between the lane review and the verified state.
- **The `web:drift` / staging relationship.** The lane read the staging assertion as simply wrong;
  the plan reads it as a deliberate replacement for a gate that cannot see a partial stage
  (`06-06-PLAN.md:229-233`). Both are right about different things: the *intent* is sound and should
  be kept, the *command* is inverted and must change (concern 4). Fixing the command does not
  reopen the decision to assert staging directly rather than rely on `web:drift`.


---
---

# Cross-AI Plan Review — Phase 6 · CONVERGENCE CYCLE 2

- **reviewed_at:** 2026-09-01T21:40:00Z
- **reviewers:** codex
- **models:** codex: `gpt-5.6-sol (reasoning=low)`
- **model_sources:** codex: `banner`
- **plans_reviewed:** 06-01, 06-02, 06-03, 06-04, 06-05, 06-06, 06-07 (seven plans, six waves)
- **baseline:** cycle-1 revision `7d1b908c`, re-slice `9f3a75b8`
- **cycle-1 carried in:** 9 HIGH + 31 actionable = 40 findings

## Codex Review (cycle 2)

### Overall assessment

Cycle 2 resolves all nine prior HIGH findings and the spot-checked MEDIUM/LOW vacuity family.
The watcher now handles first-index creation and directory replacement; restart testing has a
stable-origin proxy; generation ordering is connection-epoch scoped; layout runs are
generation-guarded; shutdown stops the publisher before `Shutdown`; and the bundle staging
assertion now runs after the commit. The seven-plan re-slice preserved every cycle-1 resolution.

Two NEW execution risks remain, both on criterion 3, and both were confirmed by the
orchestrator with independent evidence rather than taken on the lane's word:

1. `06-06` requires per-event browser instrumentation that **no plan in the phase creates**.
2. `06-04`'s "2000 tiny messages comfortably exceed any socket write buffer" is **measurably
   false on this project's own development platform** — the real figure is ~7,800.

Apart from those, the revision is substantially stronger and close to executable.

### Cycle-1 fix verification

| Finding | Verdict | Evidence |
|---|---|---|
| **H1** citations / `-ge 4` floor / named PASS | **VERIFIED** | `06-01-PLAN.md:294` invokes `TestUIProtoFieldNumbersAreStableAndUnique` and asserts its `^--- PASS:` line `-eq 1` separately from the aggregate. Floor reachable: `internal/uiserver/readonly_test.go` declares exactly three matching tests (`:82`, `:130`, `:482`); Task 1 adds `TestRPCNameIsCleanAgainstMutatingVerbs` → 4. Subtest lines are indented, so `^--- PASS: Test` cannot inflate the count. |
| **H2** publisher arming | **VERIFIED** | `06-02-PLAN.md:278-291` — `armWatches` over `[repoRoot, .codegraph, store]`, deepest existing path **plus its parent**, shallower watches dropped; three reachable states. Re-run at construction, on every raw Create/Remove/Rename, and on every debounced flush. Two named tests asserted `-eq 1` each (`06-02-PLAN.md:335`), the rename test carrying a pre-rename positive control. |
| **H3** deadlock | **VERIFIED** | `06-05-PLAN.md:5-6` — `wave: 4`, `depends_on: ["06-03","06-04"]`. |
| **H4** inverted staging assertion | **VERIFIED** | `06-07-PLAN.md:316` — `git add -A web/build` → conditional commit → `test -z "$(git status --porcelain -- web/build)"`, after the commit. Positive control `git ls-files --error-unmatch web/build/.build-manifest` confirmed to resolve against the live tree. |
| **H5** reconnect / fixed-port proxy | **VERIFIED — and the new guard proven RED by the orchestrator** | See "The one new guard" below. |
| **H6** criterion-2 pairing mechanized | **VERIFIED as a check, BLOCKED as a measurement** | `supersetOK` is genuinely computed in the verify command from `receivedGenerationsPerTab` × `triggeredGenerations` (`06-06-PLAN.md:228`) — no longer prose. But nothing produces `receivedGenerationsPerTab`. See HIGH-1. |
| **H7** seeded `Subscribe` + restart deafness | **VERIFIED** | `06-03-PLAN.md:88`, `:158`, `:209` — epoch-scoped gate with three asserted cases including lower-generation-in-a-new-epoch. The server-side mirror of this bug does **not** exist: `since_generation` is explicitly inert server-side (`06-04-PLAN.md:154`, `T-06-26`), so a restarted server at generation 1 still delivers its seeded current state to a tab resuming from 4. |
| **H8** overlapping layouts | **VERIFIED** | `06-05-PLAN.md:136` — one shared generation incremented inside `runLayout`, stale callbacks rejected before restoration or publication, three race tests. Confirmed against unprotected source at `GraphCanvas.svelte:251` and `:315-354`. |
| **H9** coalescing / fan-out split | **PARTIAL** | The division of labour is honest and correctly recorded as `coalescingProvenBy`; the browser half explicitly disclaims the coalescing claim (`06-06-PLAN.md` Task 2 action). Its server half is blocked — see HIGH-2. |

**Planner-discovered defects (no review raised these):**

- **`Serve` shutdown ordering — VERIFIED.** Current source calls `Shutdown` directly
  (`internal/uiserver/server.go:196`); `06-04` stops the publisher first and bounds the return at 2s.
- **Restart deafness — VERIFIED** (H7 above).
- **`06-04` missing `server_test.go` — VERIFIED.** Present at `06-04-PLAN.md:12`.

**MEDIUM/LOW vacuity spot-checks — all VERIFIED:**

- **M10** contaminated floor — `06-01-PLAN.md:139-141` scopes counts *per message block* with
  exact equality (`event -eq 6`, `request -eq 1`), and records the synthetic RED that the
  unscoped `-ge 7` form passed a file with `commit_sha = 6;` commented out. `ui.proto:563`
  (`bool stale = 3;` on `ExploreResponse`) is the named contaminant.
- **M20** mocked cytoscape-elk — `06-05-PLAN.md:169` now limits jsdom to proving nodes reached
  layout and defers placement to the real-browser measurement. `graph-expansion.test.ts:73`/`:120`
  confirm `FakeCore.layout()` only queues `layoutstop`.
- **M21** `numTotalTests < 1` — replaced by named-file collection + pass checks in `06-03`/`06-05`.
- **M27** `LAYOUT_OPTIONS >= 2` — now paired with a negative fork-name check (`06-05-PLAN.md:172`).
- **M32** bare `pendingWriter` — now targets `type pendingWriter struct`, which occurs exactly
  once (`internal/mcp/server.go:466`).
- **M35** three string hits — replaced by a compile-time `AsyncIterable<WatchGraphEvent>`
  assignment, string counts demoted to a presence control (`06-01-PLAN.md:272`).

### The re-slice

**Verified byte-identical.** `git show 7d1b908c:...06-06-PLAN.md` diffed against the new pair:
the criterion-5 task body differs by exactly two hunks — the `<files>` line dropping
`scripts/live-push-probe.go`, and the probe-program prose moving verbatim into `06-07` Task 1
with a `read_first` pointer added. The browser task carried across unchanged apart from the
Task-1/Task-2 split. No cycle-1 resolution was lost, and `06-06` sat at the end of the DAG so no
other plan's `depends_on` moved.

### The one new guard — RED proof independently reproduced

The re-slice introduced exactly one new guard: the proxy check at `06-06-PLAN.md:129`. The
orchestrator extracted that verify command verbatim and ran it against three implementations:

| Implementation | Result | Failure names |
|---|---|---|
| correct proxy | **PASS** | — |
| rewrites `Host` but not `Origin` | **FAIL** | `originNotRewrittenOrNotPairedWithHost` (streaming checks still passed) |
| rewrites both, buffers the body | **FAIL** | `proxyBufferedTheResponse`, `chunksArrivedTogether` (header checks still passed) |

The two failure modes discriminate **independently**, exactly as claimed. The guard is real; the
recurring "vacuous guard created inside a vacuity fix" pattern did **not** recur here.
(`node -e … --input-type=module` was also confirmed to enable top-level `await` — the trailing
flag is honoured.)

### Concerns

#### HIGH — NEW: `06-06`'s browser instrumentation has no implementation seam

`06-06-PLAN.md:168` says, in six words, "Instrument each tab to record, per event, the generation
and a client-side receipt timestamp." Its verify command then reads **eight** application-level
fields: `receivedGenerationsPerTab`, `seedGenerationPerTab`, `triggeredGenerations`,
`blockedTabReceiptsDuringBlock`, `reconnectDelaysPerTab`, `reconnectAttemptsPerTab`,
`postReconnectAppliedPerTab`, `resumeCursorSentPerTab`.

Nothing in the phase produces them:

- `06-06`'s `files_modified` is `web/scripts/live-push-stable-proxy.mjs`,
  `web/scripts/live-push-multitab-check.mjs`, `corpora/live-push-multitab-check.json` — no
  browser source, so it cannot add a seam itself, and it sits in wave 5 behind a frozen client.
- `06-03` (the browser-client plan, wave 2) modifies `live-client.ts`, `live-store.ts`,
  `status.ts`, three routes and three test files. It does **not** touch `web/src/app.d.ts` and
  declares no global. `rg` for `app.d.ts|window\.|__codegraph|globalThis|instrument` across
  `06-03-PLAN.md` returns one unrelated hit (`:298`, a threat row).
- The existing ambient surface is exactly two globals — `__codegraphFileGraphMetrics` and
  `__codegraphFileGraphGeometry` (`web/src/app.d.ts:26-45`) — both graph-only.

The contrast with `06-05` is the proof this is an omission rather than a convention: `06-05`
reads the geometry seam and **cites its declaration explicitly** (`06-05-PLAN.md:79-80`, `:227`)
and lists `GraphCanvas.svelte` in `files_modified`. `06-06` does neither.

Playwright can observe *network* stream attempts, but the required facts are post-decode and
post-generation-gate: "the tab **applied** this event" is precisely what the epoch rule filters,
and `resumeCursorSentPerTab` lives inside a length-prefixed Connect protobuf envelope no plan
teaches the harness to decode.

**This is a recurrence of the project's failure shape (a): the H6 fix satisfies its own literal
check — `supersetOK` really is computed now — while the symptom (an unrunnable browser gate)
persists one level down.** Criterion 3 and the browser half of criterion 2 both rest on it.

*Fix:* give `06-03` a documented, observation-only live seam typed in `web/src/app.d.ts`
(generation, epoch, applied-at timestamp, reconnect attempt + scheduled delay, resume cursor),
following the two established `__codegraph*` precedents; add `web/src/app.d.ts` to `06-03`'s
`files_modified`; and have `06-06` `read_first` it the way `06-05` does.

#### HIGH — NEW: 2,000 messages do not create socket backpressure — measured, not argued

`06-04-PLAN.md:232` asserts: *"2000 tiny messages comfortably exceed any socket write buffer, so
the send loop blocks and the capacity-1 channel genuinely fills."*

The orchestrator measured this on the project's own development platform (macOS,
`net.inet.tcp.sendspace` = `net.inet.tcp.recvspace` = 131072) with a Go test reproducing the
exact shape — `httptest` HTTP/1.1, 65-byte payloads (5-byte Connect envelope header + a
generation, four bools and a short SHA), `Write` + `Flush` per message, client never reading:

```
RESULT: handler flushed 7827 messages (508755 bytes ≈ 496 KB) before the write blocked
RESULT: is 2000 enough to block? false
```

**2,000 is ~3.9× too few.** Linux is worse, not better: `tcp_rmem`/`tcp_wmem` autotune to
megabytes. With the socket never blocking, the capacity-1 registry channel never fills, the send
loop drains every publish, `received == published`, and `received < published` **fails** — or,
worse, passes nondeterministically on incidental goroutine-scheduling races, making a green run
no evidence at all.

This is the sole transport-level proof behind `coalescingProvenBy`, and the browser half honestly
disclaims the claim, so **nothing in the phase currently closes D-04 end to end.**

*Fix:* make the pressure deterministic rather than raising the constant — set a deliberately tiny
`SO_SNDBUF` on the accepted connection via a wrapped listener, or publish until an observable
send counter stops advancing (proving the handler is blocked) and only then publish the
generations that must be coalesced.

#### MEDIUM — NEW: `06-06`'s reconnect gate can fail on correct behaviour

`grew` requires `reconnectDelaysPerTab.every(d => d.length >= 2 && …)`, but the fail list only
enforces `Math.min(...reconnectAttemptsPerTab) >= 1`. A tab that reconnects on its first attempt
— entirely correct behaviour — yields one delay and fails `backoffNotGrowing`. The action block
never specifies how long the upstream stays down. State a minimum downtime spanning at least two
backoff intervals, and raise the attempt floor to 2 so the two assertions agree.

#### MEDIUM — NEW: reconnect-delay provenance is unspecified

The plan requires raw delay arrays but does not say whether they come from the client's own
scheduling or are inferred from network-request timestamps. Network-derived intervals fold in
process startup, proxy 502 handling, fetch scheduling and browser throttling, so *strict*
monotonic growth can fail even when the exponential base is correct. Pin the source (the seam
from HIGH-1 is the natural home); if network-derived, compare against ranges with tolerance
rather than requiring every observed interval to strictly increase.

#### MEDIUM — NEW: criterion 5's flush interval has no exact start signal, and the debounce dilutes the bound

`06-07-PLAN.md:177-179` says "modify a source file … wait for the daemon's own debounce, and
confirm the index actually moved". The completion signal is pinned (`last_sync_unix_ms` advanced)
but the start is not. Because the same script measures both runs, the comparison stays
apples-to-apples — but the daemon's debounce constant (2000 ms default) is then included in
*both* measurements, inflating `baselineMaxFlushDurationMs` and making the
`maxFlushDurationMs <= 3 * baseline + 1000` bound substantially less sensitive to real
degradation. Start the clock after the debounce window has elapsed, or subtract it from both,
and tie completion to a content-derived fact from the specific revision written so an unrelated
metadata advance cannot be mistaken for the intended flush.

#### LOW — NEW: `06-01` prose misplaces the nonexistent test name

`06-01-PLAN.md:297` says the wrong name `TestKnownUIProtoFieldNumbersAreStable` "lives in
`internal/schema/meta_commit_test.go:120`". It lives nowhere — `rg` returns zero repo-wide. What
is at `meta_commit_test.go:120` is `TestKnownMetaFieldNumbersAreStable`, a different guard over a
different message. The mechanism is correct; only the sentence is wrong.

#### LOW — NEW: `startProxy`'s handle contract omits `url`

`06-06-PLAN.md` Task 1 specifies "an exported `startProxy({ port })` returning a handle with a
`setUpstream(url)` method and a `close()`". Both the Task 1 verify (`px.url`) and Task 2 ("open
… tabs ON THE PROXY'S URL") require a `url` property that the contract never names.

#### LOW — NEW: "three processes" understates the harness topology

`06-07` runs a daemon, an MCP harness process, a UI process and a stream probe. Three are product
processes; four OS processes participate. Say which, so the PID/liveness records are unambiguous.

### Suggestions

1. Add the live observation seam to `06-03` (HIGH-1) — typed in `web/src/app.d.ts`,
   observation-only, installed before the live client starts, following the two existing
   `__codegraph*` precedents. Then have `06-06` read it.
2. Replace the fixed 2,000-message publish with a deterministic pressure mechanism (HIGH-2). Do
   not simply raise the constant — 7,827 is a *measured* figure for one platform, not a portable one.
3. Reconcile `06-06`'s reconnect attempt floor with `grew`'s `length >= 2` requirement and state
   a minimum upstream downtime.
4. Pin the provenance of `reconnectDelaysPerTab`.
5. Pin criterion 5's flush start signal and exclude the debounce from both measurements.
6. Correct the stale test-name sentence at `06-01-PLAN.md:297`.
7. Name `startProxy`'s `url` property in the Task 1 contract.
8. State the four-OS-process / three-product-process distinction in `06-07`.

### Risk assessment

**Overall risk: HIGH.**

No cycle-1 HIGH remains unfixed, and the structural work — waves, dependencies, threat coverage,
the re-slice, the new proxy guard — all holds under direct verification. The residual HIGH risk is
entirely *verification executability* on criterion 3: the multi-tab gate has no defined source for
the facts it must record, and the sole coalescing proof rests on a socket-buffer assumption
measured false by a factor of four. Both are bounded, well-localised plan edits. With them fixed
the phase should drop to MEDIUM/LOW, with the remaining uncertainty concentrated in the real
multi-process timing harness rather than in the product design.

---

## Consensus Summary (cycle 2)

Single-lane cycle (codex, source-grounded, `file:line` citations present throughout), so
"consensus" here means *lane finding corroborated by independent orchestrator verification*. Every
finding below was re-derived against the tree rather than accepted from the lane.

### Agreed strengths

- All nine cycle-1 HIGH findings are genuinely fixed, each with a mechanism that survives direct
  inspection — not restated claims.
- The re-slice is a true re-slice: diffed byte-for-byte against `7d1b908c`, the only substantive
  change is the probe-program prose relocating from `06-06` Task 2 into `06-07` Task 1.
- The one new guard introduced by the fix pass (the proxy check) was independently proven RED
  against two distinct broken implementations, discriminating on header rewriting and streaming
  **independently**. The project's recurring "vacuous guard inside a vacuity fix" pattern did not
  recur.
- The vacuity family (M10, M20, M21, M27, M32, M35) is comprehensively repaired: scoped exact
  counts, negative controls paired with positive ones, compile-time assertions replacing string
  greps.
- H7's fix is deeper than the finding asked for — the epoch rule is mirrored by making
  `since_generation` deliberately inert server-side, so the same restart bug cannot reappear from
  the server end.

### Agreed concerns

1. **HIGH — `06-06` records eight application-level fields no plan produces.** The H6 fix
   mechanized the *check* while leaving the *measurement* undefined. Failure shape (a).
2. **HIGH — the 2,000-message coalescing pressure is measurably insufficient** (7,827 needed on
   the dev platform; more on Linux). The sole proof of D-04 is environment-dependent at best and
   nondeterministic at worst.
3. **MEDIUM — three smaller gate-fragility issues** in `06-06`'s reconnect assertions and
   `06-07`'s flush interval, plus two LOW documentation defects.

### Divergent views

- **H9's rating.** The lane rated H9 PARTIAL and filed the socket-buffer problem as a separate
  HIGH. The orchestrator agrees the split of labour is *honest* — the browser scenario explicitly
  disclaims the coalescing claim rather than quietly implying it, which is the right call — so
  H9's design is resolved and its remaining exposure is entirely HIGH-2. Counted once, as HIGH-2.
- **Magnitude of HIGH-2.** The lane described the 2,000-message figure as "not a safe assumption"
  and "environment-dependent". Direct measurement is stronger than that: it is *wrong on the
  machine this phase will be developed on*, by a factor of 3.9. This upgrades the finding from a
  portability caveat to a plan defect.
- **Read-deadline clearing.** Settled in cycle 1 (`$GOROOT/src/net/http/server.go:697`) and
  correctly not re-raised by either party. Recorded here only so a future cycle does not reopen it.
