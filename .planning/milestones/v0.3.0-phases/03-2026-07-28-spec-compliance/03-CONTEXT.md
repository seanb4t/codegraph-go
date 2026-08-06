# Phase 3: `2026-07-28` Spec Compliance - Context

**Gathered:** 2026-08-06
**Status:** Ready for planning

<domain>
## Phase Boundary

Make the stdio, tools-only server answer the `2026-07-28` wire contract correctly
— discovery, per-request `_meta` validation, result metadata, honest cache
control, per-call index detection — while every client still speaking an older
revision keeps working.

**In scope:** SPEC-01, SPEC-02, SPEC-03, SPEC-04, SPEC-05, SPEC-06, SPEC-07,
SPEC-08.

**Explicitly NOT in scope:** SPEC-09 (`subscriptions/listen` +
`notifications/tools/list_changed`). That is Phase 5, deliberately sequenced last
and isolated so slipping it cannot block the milestone. D-05 below pre-builds
part of its substrate as a side effect; it does not pull the requirement forward.

**Calibration (still binding, carried from Phase 2 `<domain>`).** The maintainer
judged this milestone over-engineered and directed that the cheap mechanism win
over the ceremonious one. Phase 2 honored that and the phase was *better* for it
— the relaxed reviewed-diff bar caught a semantic regression byte-identity would
have buried in cosmetic noise. Same rule here: a step that exists to add
assurance on top of assurance does not belong in this plan.

</domain>

<decisions>
## Implementation Decisions

### What this phase actually has to build (measured, not assumed)

- **D-01: Most of Phase 3 arrived with Phase 2's dependency swap.** This was
  measured against the real built binary during discussion, not inferred from
  the ROADMAP's framing. A single `server/discover` call against today's
  `bin/codegraph` returns:

  ```json
  {"resultType":"complete",
   "_meta":{"io.modelcontextprotocol/serverInfo":{"name":"codegraph","version":"0.1.0"}},
   "ttlMs":0,"cacheScope":"public",
   "supportedVersions":["2026-07-28","2025-11-25","2025-06-18","2025-03-26","2024-11-05"],
   "capabilities":{"tools":{"listChanged":true}}}
  ```

  That one response already delivers **SPEC-01** (discover answers with
  capabilities, no tool call first), **SPEC-03** (`resultType: "complete"`), and
  **SPEC-08** (`_meta` carries `io.modelcontextprotocol/serverInfo`), and
  demonstrates **SPEC-06**'s five-era support. The planner must **verify each
  before marking it satisfied** — inheriting is not the same as asserting — but
  should not plan to *build* them.

  **Probe warning, learned the hard way during this discussion:** SEP-2575
  signals the protocol version through **`_meta`**, not `params.protocolVersion`.
  A discover request carrying `params.protocolVersion` is rejected `-32601` and
  looks exactly like "the server does not implement discover." The first probe
  in this discussion made that mistake and produced a wrong conclusion that
  survived two messages. Use `io.modelcontextprotocol/protocolVersion` inside
  `_meta` (`mcp/protocol.go:2363`).

- **D-02: Modern-vs-Legacy gating is the SDK's job, not ours.** `mcp/mrtr.go:53`
  reads verbatim: *"For older clients the resultType is left unset."* The SDK
  ships `TestServerSessionHandle_SetsResultTypeOnNewProtocol`, and the
  method-availability switch at `mcp/server.go:1879-1893` already refuses
  `server/discover` below `2026-07-28` and refuses the removed classic methods
  above it. **Do not write a version branch in any handler.** If a Modern field
  needs suppressing for a Legacy client, that is a bug report against go-sdk, not
  code this project writes.

### The three genuine gaps

- **D-03: `cacheScope` on `server/discover` is `"public"` and must be
  `"private"`.** This is the same defect D-09 fixed for `tools/list` in Phase 2,
  in the one response path Phase 2 never touched. STATE.md's v0.3.0 decision log
  locks `ttlMs: 0` + `cacheScope: "private"` as "two halves of one correctness
  property, not independent options," and codegraph's catalog depends on a local
  `.codegraph/` index — it is never public. `ttlMs: 0` is already correct on
  discover and needs no change. Closes the remaining half of **SPEC-04**.

- **D-04: `instructions` is absent from the discover result** and is
  **SPEC-07**'s entire deliverable. `ServerOptions.Instructions` is the
  mechanism (confirmed in Phase 2's research, `mcp/server.go`), and the SDK
  copies it into the result with no extra wiring. The *content* — what guidance
  actually helps an agent orient without spending a tool call — is a writing
  task, not a plumbing task, and deserves more thought than a placeholder.

- **D-05: SPEC-05's live catalog is built with `Server.AddTool` /
  `Server.RemoveTools` mutation, not per-call filtering.** Today `hasIndex` is a
  `bool` parameter to `BuildServer`/`NewStdioServer` (`internal/mcp/server.go:310,344`),
  snapshotted at construction, so a server started before `codegraph init` never
  shows tools. go-sdk exposes both `AddTool` (`mcp/server.go:273`) and
  `RemoveTools` (`mcp/server.go:571`) post-`Connect`.
  - **Chosen over the middleware-filter alternative** (register all 8 always,
    re-check `hasIndex` in the receiving middleware Phase 2 already added) because
    mutation is faithful to "the catalog changed" and **pre-builds most of
    SPEC-09's substrate** for Phase 5.
  - **Two things the planner must resolve with evidence, not assumption:** what
    watches for the index appearing (the existing watcher? a per-call check that
    triggers mutation?), and go-sdk's concurrency guarantees around mutating
    registration while a request is in flight. The middleware approach sidesteps
    both; this one must answer them. If the concurrency story turns out to be
    unsafe, say so and re-open this decision rather than shipping a race.
  — **Reversibility:** costly — the mechanism choice shapes what Phase 5 inherits.

- **SPEC-02 is a real gap and the least understood item in the phase.** An
  unsupported version in `_meta` (`"1999-01-01"`) currently answers **`-32601`**,
  where the requirement demands **`-32022` `UnsupportedProtocolVersionError`**.
  The SDK *does* have `CodeUnsupportedProtocolVersion` (`mcp/server.go:1872-1877`)
  — it is simply not reached on this path. The `-32602`-on-malformed-`_meta` half
  was **not** measured at all. **This is the researcher's highest-value
  question**: which `_meta` failures the SDK already classifies correctly, which
  it does not, and whether the gap is ours to close or an upstream defect.

### Verification approach

- **D-06: Extend the frozen corpus and re-freeze through one reviewed-diff pass,
  in a single PR.** New scenarios are needed for surfaces the 23-transcript set
  has never covered: `server/discover` (Modern), `_meta` validation failures, and
  the index-appears-mid-session case. `ExpectedScenarioCount` moves off 23 — that
  constant is the gate that stops the list silently shrinking, so bump it
  deliberately and let it be reviewed.
  - This is exactly the regeneration Phase 1's D-03 sanctioned for Phase 3
    ("transcripts *must* legitimately change there").
  - **The two-PR split D-03's wording implies is structurally unavailable** —
    Phase 2 proved it: the wire-oracle job is a required PR leg, so a build-only
    PR is red on `TestFrozenTranscriptsMatch` while a regenerate-only PR is red
    for the opposite reason. That deadlock is why plan 02-02 exists.
  - **The freeze-gate exemption 02-02 built does NOT cover this phase.** It is
    keyed to a `go.mod` diff that both removes mark3labs and adds go-sdk — a
    shape this repo can produce exactly once, and Phase 3 will not reproduce it.
    Plan for that gate explicitly; do not discover it at PR time.
  - The reviewed-diff mechanism stays as Phase 2 defined it (D-01/D-03 there):
    run the oracle before and after, read the diff, every changed line gets a
    named cause in the commit message. No ledger file, no sign-off step. It
    earned its keep in Phase 2 by catching a ninth, unpredicted semantic
    divergence.

- **D-07: SPEC-06 is asserted, not built.** The six `legacy-*` transcripts
  already cover `2024-11-05` through `2025-11-25` plus omitted and unsupported
  versions, all passing against the migrated server, and `supportedVersions` in
  the discover result names all five eras. What SPEC-06 still wants is an
  explicit assertion that a Legacy client **completes a session and calls a
  tool** — the existing transcripts prove handshake negotiation, and the planner
  should check whether they also prove a tool call at an old revision. If they
  do, cite them; if not, that is the one scenario to add.

### Claude's Discretion

- The `instructions` text itself (D-04). Leaning: short and operational — what
  codegraph indexes, that `path` defaults to the server's cwd, and that an empty
  tool list means "not indexed here" rather than "broken." Keep it to a few
  sentences; it ships on every discover.
- Whether the index-appears scenario drives real `codegraph init` in a fixture or
  simulates the transition. The former is more honest, the latter more stable.
- Where `-32022` gets asserted if the SDK turns out not to emit it — handler,
  middleware, or a documented upstream deferral.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Milestone scope
- `.planning/ROADMAP.md` § "Phase 3" — the five success criteria
- `.planning/REQUIREMENTS.md` § SPEC-01…SPEC-08 (SPEC-09 is Phase 5, out of scope)
- `.planning/STATE.md` § Decisions — the locked `ttlMs: 0` + `cacheScope: "private"` pairing, and "never drop Legacy `initialize` support"

### Phase 2's output — the substrate this phase builds on
- `.planning/phases/02-.../02-CONTEXT.md` — D-01 (reviewed-diff bar), D-09 (the `cacheScope` correction this phase repeats for discover), D-11 (explicit `ServerOptions.Capabilities`)
- `.planning/phases/02-.../02-RESEARCH.md` — Q1–Q9 against go-sdk v1.7.0 source; **Q3 covers the initialize/discover result shape and is the closest prior art for this phase**
- `.planning/phases/02-.../02-VERIFICATION.md` — what was proven vs. disclosed
- `.planning/phases/02-.../02-05-SUMMARY.md` — the nine named causes and the reviewed-diff mechanism in practice

### The SDK (read the source, not the docs — this phase's own probe proved why)
- `$(go env GOMODCACHE)/github.com/modelcontextprotocol/go-sdk@v1.7.0/mcp/server.go` — `discover` handler at `:884`; method table at `:1783`; the Modern/Legacy availability switch at `:1879-1893`; `CodeUnsupportedProtocolVersion` at `:1872-1877`; `AddTool` at `:273`; `RemoveTools` at `:571`
- `.../mcp/mrtr.go:44-62` — `resultType` gating; "For older clients the resultType is left unset"
- `.../mcp/protocol.go:2328` (`methodDiscover`), `:2363-2369` (the three `_meta` keys)

### The codebase
- `internal/mcp/server.go` — `BuildServer` (`:344`) and `NewStdioServer` (`:310`) both take `hasIndex bool`; the Phase 2 receiving middleware is the existing seam for per-request work
- `internal/mcp/tools.go` — the 8 tools, now on struct-tag schemas
- `test/wireoracle/scenarios.go` — `ExpectedScenarioCount` at `:265`; the retracted `edge-call-before-initialize` comment records go-sdk's session-ordering enforcement
- `tools/transcriptfreeze/classify.go` — the freeze gate and 02-02's self-expiring exemption (which does **not** cover this phase)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable assets
- **The receiving middleware from Phase 2** (`AddReceivingMiddleware`) already
  carries the VRFY-03 session line and the `cacheScope` correction. It is the
  established seam for anything per-request, including D-03's discover fix.
- **The wire oracle** runs in ~17s and its capture CLI drives any binary — the
  new scenarios plug into the existing harness with no new infrastructure.
- **`ServerOptions.Capabilities`** is already set explicitly (D-11), so adding
  `Instructions` alongside it is a one-field change at a site that already exists.

### Established patterns to preserve
- Set-equality, never non-empty (`TestDefaultToolVisibility`, `TestAllowlist`,
  `TestNoIndexZeroTools`). SPEC-05 makes the tool set *dynamic* — these
  assertions must be re-expressed against the new model, not relaxed.
- A gate is not trusted until demonstrated RED against a confirmed-applied
  mutation. Standing project rule; every new assertion inherits it.
- Archtests over greps for "this must never appear in the tree."

### Integration points
- `internal/mcp/server.go:310,344` — the `hasIndex bool` parameter D-05 replaces.
  17 positional test call sites exist; Phase 2 kept them compiling via the
  variadic `Option` pattern, which is the precedent for changing this without a
  wide ripple.
- The session line is a **one-way additive-only contract** (Phase 1 D-16) —
  keys may be added, the `codegraph: mcp-session` prefix and existing key names
  may not change. If the live catalog changes what `tools=N` means mid-session,
  that is a contract question, not a formatting one.

</code_context>

<specifics>
## Specific Ideas

- The discover response's shape is already known and quoted verbatim in D-01 —
  the planner can diff against it rather than re-deriving it.
- Probe recipe that works (use this, not `params.protocolVersion`):
  `{"jsonrpc":"2.0","id":1,"method":"server/discover","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientInfo":{"name":"probe","version":"1"},"io.modelcontextprotocol/clientCapabilities":{}}}}`
- `bin/codegraph` is a gitignored build artifact — rebuild before any live probe.

</specifics>

<deferred>
## Deferred Ideas

- **SPEC-09** (`subscriptions/listen`, `notifications/tools/list_changed`) —
  Phase 5. D-05's `AddTool`/`RemoveTools` mutation builds much of its substrate;
  resist finishing it here.
- **The `legacy-unsupported-2026-07-28` scenario rename** — deferred from Phase 2
  deliberately (it changes harness code and a `.golden` filename for cosmetic
  gain). If this phase is already renaming/adding scenarios under D-06, folding
  it in becomes nearly free — worth reconsidering *then*, not now.
- **go-sdk issue #976** (`code: 0` on the pre-initialize rejection, observed live
  in Phase 2) — upstream. Track, do not work around.
- **Correcting the `annotations` hints** (`destructiveHint:true` on read-only
  query tools is almost certainly wrong) — carried from Phase 2's discretion
  list. Phase 3 already regenerates transcripts, so the marginal cost is low, but
  it is a semantic change that must be named in the diff review if taken.

</deferred>

---

*Phase: 03-2026-07-28-spec-compliance*
*Context gathered: 2026-08-06*
