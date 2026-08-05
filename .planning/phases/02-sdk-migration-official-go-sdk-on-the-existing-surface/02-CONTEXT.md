# Phase 2: SDK Migration — official go-sdk on the existing surface - Context

**Gathered:** 2026-08-05
**Status:** Ready for planning

<domain>
## Phase Boundary

Swap `internal/mcp`'s backend from `mark3labs/mcp-go@v0.56.0` to
`modelcontextprotocol/go-sdk@v1.7.0` behind the `Server` seam SDK-02 already
built, prove the agent-facing surface did not change semantically, and remove
`mark3labs/mcp-go` from `go.mod`.

**In scope:** SDK-01 (the swap + semantic-equivalence proof), SDK-03 (mark3labs
out of `go.mod`, dependency closure re-audited via `govulncheck` and SBOM),
SDK-04 (error-to-wire mapping audited and asserted), SDK-05 (input schemas
audited for constraint loss).

**Explicitly NOT in scope:** implementing `2026-07-28`'s obligations —
`server/discover`, per-request `_meta` validation, `ttlMs`/`cacheScope`,
per-call index detection. That is Phase 3. This phase inherits five-era
*negotiation* from the dependency; it does not implement the revision's
semantics.

**Calibration note (user directive, 2026-08-05).** The maintainer's explicit
judgement is that this milestone was being over-engineered relative to the
risk. That call is recorded here as a scope input, not second-guessed: prefer
the cheap mechanism over the ceremonious one throughout this phase. The wire
oracle is kept because it already exists and costs ~17s to run — not because a
heavier process was chosen. Downstream agents should treat "add a review gate,
a ledger, a sign-off step" as a proposal that must justify itself, not a default.

</domain>

<decisions>
## Implementation Decisions

### The verification bar

- **D-01:** **The byte-identity bar is relaxed to semantic equivalence plus one
  human diff read.** ROADMAP criterion 1 and REQUIREMENTS `SDK-01` were edited
  in this session to match (see `<canonical_refs>` for the exact new wording).
  Rationale: byte-identity cannot distinguish `"readOnlyHint":false`
  disappearing under `omitempty` — semantically identical, since the MCP spec
  defines absent hints by default, and no client can observe it — from
  `protocolVersion` moving `2025-11-25` → `2026-07-28`, which is precisely the
  silent-tool-loss failure this milestone exists to prevent. A person reading
  the diff distinguishes them in seconds; a byte comparator never can. —
  **Reversibility:** costly — the criterion text is now the bar `gsd-verifier`
  scores Phase 2 against, and Phase 3 inherits the same corpus; tightening back
  to byte-identity later would require compensating code for divergences
  deliberately accepted here.

- **D-02:** **ROADMAP criterion 2 was restated in the same edit.** Its original
  wording ("never regenerated, relaxed, or re-baselined") directly contradicts
  D-01 — under D-01 some transcripts *will* legitimately move. The restated
  criterion separates the two properties that were conflated: the harness
  **code** stays unmodified, and transcript **bytes** may move only through the
  reviewed pass with a named cause. Wholesale regeneration and relaxing-to-pass
  remain forbidden. This is deliberate honesty about a contradiction, not a
  loosening — leaving it unedited would have left the phase unverifiable
  against its own roadmap.

- **D-03:** **No adjudication ceremony.** There is no pre-authored expected-diff
  file, no per-transcript sign-off record, no divergence ledger. The mechanism
  is: run the oracle against the pre-swap binary and the post-swap binary, read
  the diff, and if every changed line has an explanation, commit the new
  transcripts with those explanations in the commit message. Explicitly chosen
  over the heavier alternatives after the maintainer's over-engineering call.

### Protocol version

- **D-04:** **Take whatever the SDK negotiates — no pinning, no injection.**
  ROADMAP's own goal line already says five-era negotiation "arrives inherited
  from the dependency rather than as code this project writes"; this decision
  takes that literally. A `2026-07-28` client gets `2026-07-28`; older clients
  get their era.

- **D-05:** **`legacy-unsupported-2026-07-28.golden` is an expected, explained
  divergence.** It currently freezes `"protocolVersion":"2025-11-25"` because
  mark3labs did not recognize `2026-07-28` and coerced it down. go-sdk v1.7.0
  recognizes it. The transcript moves to `2026-07-28`, and the scenario is no
  longer "unsupported version" — it should be renamed to reflect what it now
  measures. This is the single most predictable diff in the phase.

- **D-06:** **VRFY-02's stricter "reads from" property is expected to remain
  undeliverable, and that must be *proven*, not assumed.** Phase 1 shipped
  `ProtocolVersion` as an asserted pin because mark3labs exposed no injection
  point, and `internal/mcp/protocol_version.go:20` hands the stricter property
  to this phase. Scouting suggests go-sdk is the same: its initialize handler
  returns `ProtocolVersion: negotiatedVersion(params.ProtocolVersion)` — a
  package-level function — and no `ProtocolVersion` field appears in
  `ServerOptions` (which does expose `Instructions` and `Capabilities`). **This
  is Context7 doc evidence, not source enumeration.** The repo's own standard
  (Phase 1 enumerated every `func With…` in mark3labs before concluding
  absence) requires the researcher to confirm against real v1.7.0 source before
  anyone plans against it. If an injection point does exist, D-04 should be
  revisited before planning completes.

### Tool schemas (SDK-05)

- **D-07:** **Declare input schemas via Go struct tags and let `mcp.AddTool`
  infer them.** One input struct per tool with `json` + `jsonschema` tags,
  replacing the `mcp.WithString`/`mcp.WithNumber` builder chain in `tools.go`.
  Handlers get typed arguments instead of manual `RequireString`-style
  extraction. Chosen over hand-authored `*jsonschema.Schema` values (which the
  SDK does support — `Tool.InputSchema` is typed `any`) because D-01 removed
  the byte-control requirement that was the main argument for hand-authoring. —
  **Reversibility:** costly — the handler signatures change with the schema
  style, so switching later re-touches all 8 handlers.

- **D-08:** **SDK-05's headline risk is smaller than its requirement text
  implies, and the audit should say so.** The requirement names "enum
  constraints dropped by struct-tag reflection" as the notable loss.
  `internal/mcp/tools.go` declares **zero** enum constraints today — the
  closest is `codegraph_files`' `format` parameter, whose allowed values
  (`"flat"`, `"tree"`) live only in a `Description` string. The real audit
  surface is therefore description preservation and `type: "number"` vs
  `"integer"` fidelity, not enums. If the reflection path happens to make the
  `format` enum expressible, adding it is an improvement to note — not a
  requirement of this phase.

### Claude's Discretion

The user delegated these; leanings recorded as starting positions, not locks:

- **Migration shape.** Whether to build the go-sdk implementation as a sibling
  behind the existing `Server` interface (both SDKs briefly in `go.mod`, oracle
  run against both binaries, then delete mark3labs) or replace in place.
  Leaning: sibling, because the seam already exists for it
  (`internal/mcp/server.go:85-90` says so in its own doc comment) and it makes
  the before/after diff a one-command comparison. Either way SDK-03 requires
  mark3labs gone by phase end.
- **`BuildServer`'s return type.** It currently returns `*server.MCPServer` — an
  SDK type — with 17 positional call sites, all in tests. Whether the migrated
  version returns the new SDK's concrete type, or the leak gets closed properly
  now that `NewStdioServer` is the production entrypoint, is the planner's call.
- **Whether the `annotations` block is worth preserving as-is.** Today's frozen
  value is `readOnlyHint:false, destructiveHint:true, idempotentHint:false,
  openWorldHint:true` — mark3labs zero-values, not deliberate choices, and
  arguably *wrong* for read-only query tools. If the migration surfaces them,
  correcting them is reasonable; it is a semantic change and must be called out
  in the diff review rather than slipped in.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Milestone scope and requirements

- `.planning/ROADMAP.md` § "Phase 2: SDK Migration" — **criteria 1 and 2 were
  edited during this discussion (2026-08-05)**. Read the current text, not any
  cached memory of it. Criterion 1 now reads "semantically unchanged … the full
  transcript diff is read line by line and every changed line has a recorded
  cause"; criterion 2 now separates harness *code* (unmodified) from transcript
  *bytes* (may move through the reviewed pass with the cause named).
- `.planning/REQUIREMENTS.md` § SDK Migration — `SDK-01` was edited in the same
  session to match. `SDK-02` is already `[x]` complete. Also § Out of Scope,
  which locks "never drop Legacy `initialize` support".
- `.planning/STATE.md` § Session Continuity → "PHASE 1 CARRY-INS FOR PHASE 2" —
  the three carry-ins, including the frozen-set gap and the VRFY-02 hand-off.

### Phase 1's output — the thing this phase is measured against

- `.planning/phases/01-protocol-scoping-the-sdk-independent-wire-oracle/01-CONTEXT.md` —
  D-01…D-16. **D-03 (anti-regeneration) is partially superseded by this phase's
  D-01/D-02/D-03**; its CI cross-change guard still stands, its "Phase 2 =
  frozen, no exceptions" clause does not.
- `test/wireoracle/COVERAGE-BASELINE.md` — what the 23 scenarios cover and why
  they cannot be recaptured after the swap.
- `test/wireoracle/MUTATION-PROOF.md` — the four mutations proven RED in Phase 1;
  §"mutation 3" documents the known coverage hole (see `<deferred>`).
- `testdata/wireoracle/transcripts/` — the 23 frozen `.golden` files.
- `test/wireoracle/scenarios.go:253` — `ExpectedScenarioCount = 23`, the
  assertion that stops the scenario list silently shrinking. If a scenario is
  renamed or added, this constant is the gate that notices.

### The migration surface

- `internal/mcp/server.go` — `Server` interface (:80), `mark3labsServer`
  adapter (:88), `NewStdioServer` (:130, panics on nil sessionLog by design),
  `BuildServer` (:164, returns `*server.MCPServer` — the remaining SDK leak),
  session-line hook via `AddAfterInitialize` (:182).
- `internal/mcp/tools.go` — all 8 tool declarations (:80-181) and handlers.
  **Every handler returns `mcp.NewToolResultError(err.Error()), nil`** — there
  are zero bare `return nil, err` paths. SDK-04 must *confirm* this rather than
  inherit it, but it should recalibrate the expected size of that requirement.
- `internal/mcp/protocol_version.go` — the asserted pin and the doc comment
  (:9-22) that hands "reads from" to this phase.
- `internal/mcp/archtest/` — the guard forbidding an SDK-owned protocol
  constant. It was written to forbid the *class*, not mark3labs' spelling —
  verify it still fires against go-sdk's equivalent.
- `internal/cli/serve.go` — depends only on `internal/mcp.Server` +
  `NewStdioServer` (SDK-02 closed this leak; keep it closed).

### Research

- `.planning/research/PITFALLS.md` — Pitfall 7 (`omitempty` on bare `bool`,
  shipped in go-sdk v1.7.0) is the predicted cause of the `annotations` diff;
  Pitfall 6 (error mapping) is the one the scout found smaller than feared;
  Testing Traps A–D still apply to any new assertion written here.

### External SDK documentation (fetched 2026-08-05 via Context7, `/modelcontextprotocol/go-sdk`)

- `docs/server.md` § "Customize Tool Schemas" — `Tool.InputSchema` is typed
  `any`; explicit `*jsonschema.Schema` values and post-inference mutation of
  `in.Properties[...]` are both supported paths.
- `mcp/protocol.go` § `type Tool` — field order is `Meta, Annotations,
  Description, InputSchema, Name, OutputSchema, Title, Icons`, which happens to
  match the frozen transcripts' key order.
- `mcp/server.go` § initialize handler — `ProtocolVersion:
  negotiatedVersion(params.ProtocolVersion)`; `ServerOptions` exposes
  `Instructions` and `Capabilities`. **Treat as a lead, not a finding** — D-06
  requires source confirmation.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets

- **The `Server` seam is already built and documented for exactly this.**
  `internal/mcp/server.go:85-90`'s own doc comment reads: "A future SDK swap
  (Phase 2) adds a sibling implementation behind this same interface;
  internal/cli never needs to change." Phase 2 should take that at its word.
- **The wire oracle imports no SDK.** Despite ~30 textual `mark3labs` mentions
  across `test/wireoracle/`, every one is a comment or a `[VERIFIED: …]`
  citation — `capture.go:4` states outright that it never imports any
  `mark3labs/mcp-go` package. Criterion 2's "harness code unmodified" is
  therefore architecturally achievable, not aspirational.
- **The functional-options pattern in `BuildServer`** (`Option`,
  `WithSessionLog`) already absorbs new configuration without touching the 17
  positional call sites. Reuse it rather than widening the signature.

### Established Patterns

- **Set-equality, never non-empty.** `TestDefaultToolVisibility`,
  `TestAllowlist`, `TestNoIndexZeroTools` assert exact sets / exact zero.
  Preserve this through the migration's test churn (PITFALLS Trap D).
- **A gate is not trusted until demonstrated RED** against a confirmed-applied
  mutation. Standing project decision, listed in STATE.md as outliving v1.0.
  Any *new* assertion written for SDK-04 inherits it.
- **Archtests over greps** for "this must never appear in the tree" — this repo
  has already shipped an inverted `rg -qv` gate.

### Integration Points

- `internal/mcp/server.go:164` — `BuildServer` returns `*server.MCPServer`. This
  is the last SDK type in a non-`internal/mcp` -visible signature; 17 test call
  sites depend on it.
- `internal/mcp/server.go:182` — the `AddAfterInitialize` hook carrying VRFY-03's
  session line. go-sdk's hook/middleware model differs from mark3labs' `Hooks`
  struct; this is the single most likely place the migration needs real
  redesign rather than mechanical translation. The session line's format is a
  **one-way additive-only contract** (Phase 1 D-16) — the audit shim, the
  oracle, and any user tooling parse it. Keys may be added; the
  `codegraph: mcp-session` prefix and existing key names may not change.
- `go.mod:13` — `github.com/mark3labs/mcp-go v0.56.0`, the line SDK-03 deletes.

### Findings that change downstream assumptions

1. **Two transcript divergences are predicted, one near-certain.**
   `legacy-unsupported-2026-07-28.golden` will move `2025-11-25` → `2026-07-28`
   (near-certain — go-sdk recognizes the version mark3labs did not). The
   `annotations` block's two `false` hints may vanish under `omitempty`
   (predicted from PITFALLS Pitfall 7; unverified). `legacy-omitted-version.golden`
   is a third candidate — mark3labs coerced an omitted version to `2025-03-26`,
   and go-sdk's behavior on an empty `protocolVersion` is unknown.
2. **JSON key order should survive.** go-sdk's `Tool` struct declares fields in
   an order that marshals to `annotations, description, inputSchema, name` —
   matching the frozen transcripts. One less predicted diff.
3. **Zero enum constraints exist today** (see D-08). SDK-05 is smaller than its
   text suggests.

</code_context>

<specifics>
## Specific Ideas

- The maintainer's framing, verbatim in spirit: *"we're being way too careful
  here."* Where a plan step exists only to add assurance on top of assurance,
  cut it.
- The mechanism that survives the trim is deliberately one sentence: **run the
  oracle before and after, and look at what moved.**
- `legacy-unsupported-2026-07-28` should be renamed as part of the change — after
  the swap it no longer measures an unsupported version, it measures the newest
  supported one.

</specifics>

<deferred>
## Deferred Ideas

- **Extending the frozen set before the ONE-WAY window closes — deliberately
  declined.** Phase 1 left one known hole (`01-UAT.md` → Gaps,
  `test/wireoracle/MUTATION-PROOF.md` mutation 3): no scenario exercises a
  handler's own required-argument validation failure, so all four captured error
  shapes are protocol-level. `mark3labs` is still in `go.mod` *right now*, which
  makes this the last moment it is capturable against the pre-migration server —
  after the swap, never. It is being skipped because D-01 removed the
  byte-identity bar that gave a pre-swap baseline most of its value, and because
  SDK-04 will assert the error shape forward-only against go-sdk regardless.
  **This is an irreversible skip and the maintainer's to reverse** — say so
  before the swap PR lands if the call should go the other way.
- **Correcting the `annotations` hints** (`destructiveHint:true` on read-only
  query tools is almost certainly wrong) — noted under Claude's Discretion; if
  not taken in this phase it belongs with Phase 3's deliberate wire changes.
- **Client-side `tools/list` caching bugs as a known confound** — STATE.md
  Blockers records real upstream issues (`anthropics/claude-code#41123`,
  `#40025`, `#50515`). These will be misdiagnosed as migration regressions.
  Documenting them belongs alongside this milestone but is not a Phase 2 build
  task.

</deferred>

---

*Phase: 02-sdk-migration-official-go-sdk-on-the-existing-surface*
*Context gathered: 2026-08-05*
