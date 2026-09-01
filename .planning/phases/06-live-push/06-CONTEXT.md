# Phase 6: Live Push - Context

**Gathered:** 2026-09-01
**Status:** Ready for planning

<domain>
## Phase Boundary

Open views stop going quietly stale — a watcher re-index reaches the browser over the
same schema and the same client as every other call, and updates what is on screen in
place.

**Requirements:** RPC-04, LIV-01, LIV-02, LIV-03, LIV-04

**Depends on:** Phase 5 (every view live push updates, including the graph, must exist and
be stable first) and Phase 1's `FIX-01`, whose root cause this phase's streaming lifecycle
is reviewed against.

**This is the last phase of milestone v0.12.0.**

## Grounding measured this session (not assumed)

| Fact | Evidence |
|---|---|
| `codegraph ui` has **no watcher** | zero `watch.`/`fsnotify` hits in `internal/cli/ui.go` and `internal/uiserver/` |
| The daemon has **no listener or socket** | nothing addressable from another process in `internal/daemon/daemon.go` |
| `ui` and `daemon` are **separate processes by requirement** | `SRV-01`; `ui.go:24` — *"no lifecycle shared with `codegraph daemon`"* |
| `UIService` has **0 streaming methods** today | `ui.proto`, 13 unary rpcs after Phase 5 |
| `schema.Meta` carries **`last_sync_unix_ms`** (field 4), plus `node_count`, `edge_count`, `healthy`, `health_message` | `internal/schema/*.proto:137-151` |
| `PutMeta` runs on **every index**, whoever ran it | `internal/indexer/sync.go:182,411`, `internal/indexer/resolve.go:790` |
| `writeTimeout = 60 * time.Second` is an **absolute deadline on a whole response** | `internal/uiserver/server.go:49` |
| `PendingChanges` is still an **inert all-zero placeholder** | Phase 4 D-06, explicit REQUIREMENTS out-of-scope row |
| `onWatchOpen` is **test-only**, not a production seam | `daemon.go:128-137` — *"Production callers leave it nil"* |

So today **nothing can tell the UI process that a re-index happened.** That gap is what
D-01 closes.

</domain>

<decisions>
## Implementation Decisions

- **D-01:** **The UI learns of a re-index by watching the STORE with fsnotify, and treats
  `Meta` as the authoritative signal.**

  `.codegraph/store/` is watched as the **wake-up**; `GetMeta()` is then read as the
  **truth**, and an event is emitted only when `last_sync_unix_ms` actually changed.

  Why this shape:
  - **Event-driven**, so criterion 2's *"as they happen"* holds — no poll interval latency.
  - **Pebble's directory is noisy** (compactions, WAL rotation, MANIFEST churn, OPTIONS
    rewrites — all observed with current timestamps in the live store). The `Meta`
    comparison is the filter that stops spurious wakeups reaching the browser. **The
    fsnotify event is never itself the signal.**
  - **Every writer is covered** — the daemon, a manual `codegraph index`, anything else —
    because they all go through `PutMeta`.
  - **No IPC, and `SRV-01` stays intact**: the two processes remain independent, and live
    push does not require a daemon to be running.
  - The `Meta` read is *also* exactly the payload D-05 needs, so the mechanism and the
    message agree by construction.

  Rejected: polling `GetMeta()` on an interval (weakens *"as they happen"*); a source-file
  watcher in `ui` (duplicates the daemon's fsnotify load, and a source change is not an
  index rebuild — it would signal staleness the UI cannot yet serve); a daemon subscribe
  socket (a new IPC protocol to design, version and secure; couples the processes `SRV-01`
  separated; and live push would then only work while a daemon happens to run).

- **D-02:** **The 60s write deadline is cleared per-request, for the streaming path only.**

  A thin middleware calls `http.NewResponseController(w).SetWriteDeadline(time.Time{})` for
  the streaming method and nothing else.

  `writeTimeout = 60s` is an **absolute** deadline on an entire response write, so a
  long-lived stream would be killed by it. The constant's own comment — *"writeTimeout must
  exceed the slowest legitimate response… 60s is roughly two orders of magnitude above
  that"* — is sound for unary calls and simply does not describe a stream.

  The 60s bound therefore **stays fully intact for all 13 unary rpcs and the SPA handler**,
  where the documented protection still applies. Go 1.26.5 supports `ResponseController`,
  and `server.go:141`'s `guarded := originHostGuard(port, mux)` already establishes
  middleware wrapping as this file's pattern. `TestListenSetsEveryServerTimeout` continues
  to guard the four server defaults unchanged.

  Rejected: setting `WriteTimeout: 0` server-wide (trades a documented safety property
  across the whole surface to serve one method); capping stream lifetime under 60s with
  client reconnects (forced churn at least once a minute per tab is close to the reconnect
  storm `LIV-03` exists to prevent, and each gap risks missed events).

- **D-03:** **LIV-04 is served by pinning existing node positions and laying out only new
  nodes — plus a no-layout fast path — and the result MUST be measured at corpus scale.**

  Two paths:
  - **Node set unchanged** (only counts, `cycleIds`, health changed): update data in place,
    **run no layout at all**.
  - **Nodes added or removed**: capture every surviving node's position, pin it as a layout
    constraint, and let ELK place only the genuinely new nodes.

  **The measurement is not optional.** `05-07-SUMMARY.md:262` recorded 0 of 6 unaffected
  nodes moved / 0.00px displacement — *at its own small fixture scale* — and explicitly
  flagged that as *"not necessarily representative of a large real corpus"*, naming this
  phase as the inheritor: *"Phase 6 will either need incremental layout or an explicit
  position-preservation pass if it wants stable in-place live updates at scale."*
  **Measure survivor displacement at guava scale (134 collapsed / 3,233 expanded), not at
  fixture scale.** A 0.00px result over 6 nodes does not generalise.

  Rejected: full re-layout then restoring survivor positions (new nodes get placed against a
  layout that is then overwritten, so they can land overlapping survivors, and the wasted
  full layout costs the time the collapsed default exists to avoid); fast path alone (fails
  LIV-04 in exactly the case it is about — a re-index that adds or removes a file).

- **D-04:** **Backpressure is a bounded per-subscriber channel that COALESCES.**

  Each subscriber holds its own small buffered channel. When it is full, a newer event
  **replaces** the unsent older one rather than being dropped or blocking the publisher.

  This is correct because these are **state-change notifications, not a log**: a client that
  never sees *"changed at T1"* loses nothing by receiving only *"changed at T2"*, since it
  re-reads current state either way. A slow tab therefore cannot block fast ones, nothing
  meaningful is lost, and memory stays O(1) per subscriber.

  Rejected: dropping the newest when full (leaves a client holding a **stale** queued event
  while a newer one was discarded — backwards for a state notification); disconnecting slow
  subscribers (a briefly-backgrounded tab gets dropped for a transient stall, and repeated
  cycles look like the reconnect storm `LIV-03` wants avoided).

- **D-05:** **The event carries the `Meta` delta plus a monotonic generation counter; views
  re-fetch through their existing rpcs.**

  Fields: `generation`, `node_count`, `edge_count`, `healthy`, `health_message`,
  `last_sync_unix_ms`.

  Views that display exactly those values — the **health chrome and the staleness banner** —
  render **directly from the event with no round trip**. That is what satisfies criterion 1's
  requirement that the chrome update *"by the same mechanism rather than staying stale itself
  while the data around it moves."* Views needing full data (graph, browse, workbench)
  re-fetch through the rpcs they already use, so **no rpc's wire shape is duplicated into the
  event** and the 13 existing rpcs remain the single source for their own data.

  Rejected: a bare generation-only ping (forces a round trip for health/staleness values the
  server already had in hand — the one thing criterion 1 calls out by name); streaming full
  per-view snapshots (duplicates several rpcs' wire shapes into a message that must then be
  kept in sync forever, and a full `FileGraph` payload measured **3,713,528 bytes** at guava
  scale — not something to push on every re-index).

## ⚠ Naming constraint discovered this session — read before proposing the rpc name

**`WatchIndex`, `IndexEvents` and `StreamIndex` are all BANNED.** Each contains `Index`,
which is one of the 19 substrings in `internal/uiserver/readonly_test.go`'s `mutatingVerbs`
fixture, matched by bare `strings.Contains`. This is the exact trap that forced Phase 4 to
abandon `GetIndexHealth`.

The orchestrator proposed `WatchIndex` during discussion and the live check rejected it —
**verified, not reasoned about.** The full banned list: `Create, Update, Delete, Remove,
Set, Put, Post, Write, Add, Insert, Mutate, Patch, Modify, Sync, Reindex, Index, Clear,
Reset, Save`. Note the non-obvious consequences: **`LiveUpdates` collides** (contains
`Update`), and anything containing `Reset` collides twice (`Reset` and `Set`).

Verified CLEAN this session: `WatchGraph`, `GraphEvents`, `WatchRepo`, `StreamEvents`,
`Subscribe`, `WatchStore`, `LiveEvents`, `WatchChanges`, `Events`, `GraphStream`.

**Re-run the check against the live fixture before the proto freeze regardless** — the
fixture is the authority, not this list, and a positive control must prove the checker
discriminates (asserting that `GetIndexHealth` still collides is the standing control).

</decisions>

### Post-Research Corrections (2026-09-01)

Research verified two of the decisions above against installed source and upstream, and
**both were wrong as originally written**. The maintainer ruled on each. These supersede the
corresponding text in D-03 and D-05; where they conflict, **these win**.

- **D-06 (CORRECTS D-03): the 0px guarantee comes from OUR write-back, not from ELK.**

  D-03 said "pin surviving node positions" and promised a *"survivors move 0px"* gate.
  **ELK cannot deliver that.** The ELK maintainer states it directly on `eclipse-elk#355`:
  the interactive options *"cannot be used to precisely fix the positions of the nodes
  during layered layout. You can use the various interactive strategies to somewhat
  preserve the topology."*

  What IS true, verified in the installed build (`elkjs@0.9.3`, with a positive control
  proving the extractor ran):
  - `interactive` ×12, `INTERACTIVE` ×4, `semiInteractive` ×1, `org.eclipse.elk.position` ×2
    — the interactive strategies genuinely exist and are usable.
  - `cytoscape-elk` **already sends every node's current position to ELK**
    (`src/layout.js:47-51` sets `k.x`/`k.y` from `node.position()`), so ELK lays out with
    full knowledge of the existing arrangement.

  **The decided shape:** run ELK with the interactive strategies enabled (so new nodes are
  placed sensibly, knowing where the survivors are), then **authoritatively write back each
  surviving node's prior x/y**. New nodes keep ELK's placement. The zero-displacement
  guarantee lives in the write-back, which is exact, rather than in ELK's best effort, which
  is explicitly approximate.

  The guava-scale measurement D-03 requires still stands — but it now measures a property
  the implementation *enforces*, rather than hoping ELK approximated it well enough.

  > ⚠ **Trap for anyone re-checking this:** `elkjs` is a *transitive* dep of `cytoscape-elk`
  > and pnpm does NOT hoist it. It exists **only** at
  > `web/node_modules/.pnpm/elkjs@0.9.3/node_modules/elkjs/`, never at
  > `web/node_modules/elkjs/`. A grep against the un-hoisted path returns zero for every
  > option — including ones that demonstrably work — which is indistinguishable from the
  > options being absent. The orchestrator hit exactly this and was saved only by a positive
  > control. **Pair any search of this bundle with a control term you know is present**
  > (`hierarchyHandling`, `INCLUDE_CHILDREN`, `layered`).

- **D-07 (CORRECTS D-05): the event carries what `classifyStatus` actually consumes.**

  D-05 claimed the health and staleness chrome could render directly from `Meta`'s fields.
  **Verified false.** `classifyStatus(response: GetStatusResponse)` (`web/src/lib/status.ts:60`)
  reads `initialized`, `stale`, `storeExists`, `indexingInProgress` and `commitSha` — **none
  of which were in D-05's field set** — and **nothing in the shipped UI renders `healthy` or
  `health_message` at all**. D-05's stated reason for rejecting a bare ping (that it would
  force a round trip for values the server already had in hand) was therefore based on a
  false premise: the server has those values in `StatusResult`, not in `Meta`.

  **The decided event shape** — mirroring `GetStatusResponse`'s own field names so
  `classifyStatus` can be fed directly, with no second representation of the same state:

  ```protobuf
  int64  generation           = 1;
  bool   initialized          = 2;
  bool   stale                = 3;
  bool   store_exists         = 4;
  bool   indexing_in_progress = 5;
  string commit_sha           = 6;
  ```

  **Dropped:** `node_count`, `edge_count`, `healthy`, `health_message` — no shipped component
  renders them. Graph, browse and workbench continue to re-fetch through their existing rpcs,
  exactly as D-05 intended.

  **D-01 is unaffected.** `Meta.last_sync_unix_ms` remains the authoritative *change detector*
  on the server side; it simply is not the *payload*. Detecting the change and describing the
  new state are two different jobs, and this is what the event carries, not how the event is
  triggered.

### Claude's Discretion

- The rpc's final name, chosen from the verified-clean set, and its request message shape.
  The proto freeze will earn a `blocking-human` checkpoint regardless, as every rpc in this
  milestone has.
- The store-watcher debounce window, sized so Pebble's compaction churn cannot produce an
  event burst. The `Meta` comparison in D-01 is the real filter; debounce is an efficiency
  measure, not the correctness mechanism.
- Client reconnect: exponential backoff with jitter, resuming from the last seen
  `generation`.
- Which routes subscribe, and how the graph route's subscription interacts with D-03's two
  layout paths.

### Deferred Ideas

- A daemon subscribe socket — revisit only if store-watching proves insufficient.
- Making `PendingChanges` live (still the Phase 4 D-06 placeholder; explicitly out of scope).
- Streaming per-view payloads to avoid re-fetches — revisit only if re-fetch latency is
  measured as a real problem.

### Carried-forward open items (from Phase 5, none blocking)

- **WINDOWS 26** — non-fatal `cytoscape-elk` adapter error.
- **WINDOWS 28** — guava-scale cytoscape layout warning, pre-existing.
- **WINDOWS 29** — `web:drift`'s output half enumerates the filesystem (`Taskfile.yml:60`)
  while its source half uses `git ls-files` (`:42`), so it cannot detect an incompletely-
  staged bundle. **This phase rebuilds and re-commits the bundle**, so stage with
  `git add web/build` and verify `git status --porcelain web/build` is empty — do not rely
  on the gate to catch a staging error.

Maintainer decision 2026-09-01: all three carry forward to the milestone audit.

### Criterion 3's recorded verdict — not optional

`LIV-03` requires the stream's lifecycle bookkeeping be *"checked against Phase 1's
`pendingWriter` root cause — server-initiated writes counted separately from client-initiated
pending state — with the verdict recorded rather than assumed."*

`FIX-01`'s bug was precisely that server-initiated notifications corrupted a counter meant
for client-initiated pending state (`internal/mcp/server.go:466`). The streaming rpc has the
same shape: server-initiated event writes alongside client-initiated request state. **Record
the verdict explicitly in the SUMMARY** — an unstated "we checked and it's fine" does not
satisfy the criterion.
