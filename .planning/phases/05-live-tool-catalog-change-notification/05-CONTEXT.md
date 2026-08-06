# Phase 5: Live Tool-Catalog Change Notification - Context

**Gathered:** 2026-08-06
**Status:** Ready for planning

<domain>
## Phase Boundary

A client that opts into `subscriptions/listen` learns that codegraph's tool
catalog changed when it changes, instead of on its next poll.

**In scope:** SPEC-09, and only SPEC-09.

**Explicitly NOT in scope:** anything else. SPEC-01…SPEC-08 closed in Phase 3;
VULN-* and MAINT-* closed in Phase 4. This is the milestone's last phase and it
was deliberately isolated so that slipping it could not block the core value.

**Calibration (binding, carried through all four prior phases).** The maintainer
judged this milestone over-engineered and directed the cheap mechanism over the
ceremonious one. It has now paid off four times, most sharply in Phase 3, where
probing the real binary showed most of an eight-requirement phase was already
delivered. **The same is true here** — see D-01. Do not build what already works.

</domain>

<decisions>
## Implementation Decisions

- **D-01: `subscriptions/listen` already works end-to-end. This phase asserts it;
  it does not build it.** Measured against a freshly built binary during
  discussion, not inferred:

  Request (Modern `_meta`, correct field name):
  ```
  {"jsonrpc":"2.0","id":1,"method":"subscriptions/listen",
   "params":{"_meta":{…},"notifications":{"toolsListChanged":true}}}
  ```
  Response — two frames, both correct:
  ```
  {"jsonrpc":"2.0","method":"notifications/subscriptions/acknowledged",
   "params":{"_meta":{"io.modelcontextprotocol/subscriptionId":1},
             "notifications":{"toolsListChanged":true}}}
  {"jsonrpc":"2.0","id":1,"result":{"resultType":"complete",
   "_meta":{"io.modelcontextprotocol/serverInfo":{…},
            "io.modelcontextprotocol/subscriptionId":1}}}
  ```

  Criterion 1 (`tools.listChanged: true` advertised) is **already satisfied** —
  it appears in every one of the 27 frozen transcripts' `initialize` capabilities.

  Phase 3's `AddTool`/`RemoveTools` work supplies the other half: `changeAndNotify`
  already fires `notifications/tools/list_changed`, and Phase 3's per-request
  `hasIndex` re-check is what makes a mid-session `codegraph init` mutate the
  catalog at all. **The plumbing is in place on both ends.**

  **What is genuinely unproven, and is this phase's actual work:** that an opted-in
  Modern listen stream *actually receives* the notification when the Phase-3
  re-check mutates the catalog mid-session. The stream works and the mutation
  works; nobody has observed the two connected. Prove it on the wire.

- **D-02: A silent-failure mode found by accident, and worth an assertion.**
  The first probe used `toolListChanged` (singular). The server **accepted it,
  returned a `subscriptionId`, and acknowledged `"notifications":{}`** — an empty
  subscription set. No error, anywhere.

  A client that misspells its opt-in therefore gets a successful-looking
  subscription that will never deliver a notification. This is SDK behavior, not
  codegraph's, and it is the same failure class this milestone was scoped
  around: *tools silently not advertised, no red check*.

  **The acknowledgment echo is the only discriminator between a live opt-in and a
  dead one.** That makes it the thing worth asserting — a scenario that captures
  the acknowledgment must show the echoed `notifications` object non-empty, or it
  proves nothing about whether the subscription is live.

- **D-03: Criterion 3 is a non-regression claim and needs a real discriminator.**
  "A client that does not opt in observes no change in session behavior from
  Phase 3's server" is satisfiable by the 27 existing frozen transcripts — none
  of them opens a listen stream, so if they still pass byte-identical, no
  non-opting client saw a change. **Prefer that over authoring a new
  no-op scenario**: the existing corpus already is the assertion, and Phase 3's
  reviewed-diff mechanism already reports any movement in it.

- **D-04: Transcript handling follows Phase 3's precedent exactly.** New scenarios
  are additive; `ExpectedScenarioCount` moves off 27 deliberately and is stated.
  If any pre-existing transcript moves, that is a **finding**, not a regeneration
  — criterion 3 says non-opting clients see no change, so movement in a
  non-listen transcript would falsify it. The anti-regeneration guard is advisory
  as of Phase 4 and will report the collision without blocking; do not silence it.

### Claude's Discretion

- Whether the mid-session notification proof reuses Phase 3's
  `Scenario.InitAfterRequest` harness hook (which already blocks on an observed
  response before running a real `codegraph init`) or needs a variant for
  long-lived streams. Leaning: reuse — it was built for exactly this shape and is
  proven byte-stable across three captures.
- Whether to assert the `subscriptionId` correlation (`_meta.io.modelcontextprotocol/subscriptionId`)
  beyond its presence. It exists to demultiplex concurrent listens; codegraph
  opens one, so depth here is optional.
- Whether the D-02 empty-subscription behavior earns its own captured scenario or
  a unit-level assertion. It is SDK behavior we cannot change, so the question is
  only where the record lives.

</decisions>

<canonical_refs>
## Canonical References

- `.planning/ROADMAP.md` § "Phase 5" — the three criteria, and the explicit note that this phase was sequenced last so slipping it cannot block the milestone
- `.planning/REQUIREMENTS.md` § SPEC-09 — the only requirement in scope
- `.planning/phases/03-2026-07-28-spec-compliance/03-CONTEXT.md` — D-05's `AddTool`/`RemoveTools` decision, which explicitly deferred the Modern-side notification path to this phase
- `.planning/phases/03-2026-07-28-spec-compliance/03-RESEARCH.md` § Q2 — `changeAndNotify` fires automatically to Legacy sessions; Modern sessions require `subscriptions/listen`
- `.planning/phases/03-2026-07-28-spec-compliance/03-04-SUMMARY.md` — the per-request re-check and `Scenario.InitAfterRequest` harness hook this phase likely reuses
- `internal/mcp/server.go` — the per-request re-check, `registerTools`/`unregisterTools`, and the `AddReceivingMiddleware` seam
- `test/wireoracle/scenarios.go` — `ExpectedScenarioCount` (27), `Scenario.InitAfterRequest`
- `$(go env GOMODCACHE)/github.com/modelcontextprotocol/go-sdk@v1.7.0/mcp/protocol.go:2071-2092` — `NotificationSubscriptions` (note `toolsListChanged`, plural) and `SubscriptionsListenParams`
- `.../mcp/server.go:705-720` (`changeAndNotify`), `:766-789` (listen-stream demultiplexing, `injectMetaSubscriptionID`), `:1190` (listen requires a request ID)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable assets
- **`Scenario.InitAfterRequest`** (Phase 3, `03-04`) blocks on an *observed response* before running a real `codegraph init` mid-session — built precisely to capture a catalog change on the wire without sleeps, and proven byte-identical across three captures.
- **Phase 3's per-request `hasIndex` re-check** is what makes the catalog mutate at all; without it there would be nothing to notify about.
- **The advisory transcript-freeze guard** (Phase 4) reports rather than blocks, so a phase that adds scenarios no longer needs an exemption.

### Established patterns
- A gate is not trusted until demonstrated RED against a confirmed-applied mutation.
- Set-equality, never non-empty.
- Scenario count changes are stated deliberately, never drifted.
- **Probe with `_meta`, never `params.protocolVersion`** — three separate wrong conclusions this milestone traced to malformed probes.

### Integration points
- `changeAndNotify` is the SDK's, not ours — codegraph triggers it via `AddTool`/`RemoveTools`, it does not implement notification delivery.
- The session line is a one-way additive-only contract (Phase 1 D-16). If a listen stream changes what a session reports, that is a contract question.

</code_context>

<specifics>
## Specific Ideas

- Working probe recipe (use verbatim; note the plural in `toolsListChanged`):
  `{"jsonrpc":"2.0","id":1,"method":"subscriptions/listen","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientInfo":{"name":"probe","version":"1"},"io.modelcontextprotocol/clientCapabilities":{}},"notifications":{"toolsListChanged":true}}}`
- `bin/codegraph` is a gitignored build artifact — rebuild before any probe.
- The acknowledgment arrives **before** the result frame. A capture that reads only the id-matched response will miss it.

</specifics>

<deferred>
## Deferred Ideas

- **go-sdk issue #976** (`code: 0` on the pre-initialize rejection) — upstream, tracked since Phase 2.
- **The `legacy-unsupported-2026-07-28` rename** and the **`annotations` hint correction** — declined in Phases 2, 3 and 4. If this phase touches the corpus anyway the marginal cost is low, but both remain out of scope unless deliberately reopened.
- **The daemon extreme-load tail and its feedback-latency tradeoff** — accepted in Phase 4, recorded in STATE.md, not scheduled.
- **`GO-2026-5932`** — accepted unmitigated exposure in release tooling, surfaced by Phase 4's advisory job.

</deferred>

---

*Phase: 05-live-tool-catalog-change-notification*
*Context gathered: 2026-08-06*
