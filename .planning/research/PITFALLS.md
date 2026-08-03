# Pitfalls Research

**Domain:** MCP protocol-revision migration (2026-07-28) and Go MCP SDK swap, for an already-shipped, installed-in-8-agents, stdio-only, tools-only server
**Researched:** 2026-08-03
**Confidence:** MEDIUM overall — official spec/SDK sources (modelcontextprotocol.io, blog.modelcontextprotocol.io, GitHub releases from `modelcontextprotocol/go-sdk` and `mark3labs/mcp-go`) are cross-checked across multiple independent hits (MEDIUM per this repo's classify-confidence seam, `--verified` tier — the seam has no HIGH tier for `exa`). Single-source vendor blog commentary and one-off migration write-ups are flagged LOW/inferred inline. Some conclusions about codegraph-go's own current test suite are direct code-reading (`internal/mcp/server_test.go`), not web research — those are marked INFERRED (repo-specific).

## Critical Pitfalls

### Pitfall 1: The spec's stateless rework is an HTTP-scaling story that stdio servers can wrongly over-apply

**What goes wrong:**
Every primary source on 2026-07-28 (the official blog, New Relic, Appwrite, Microsoft's App Service post, Solo.io) frames the entire release around removing session state so a *remote, load-balanced HTTP* server can scale. A team reading this guidance and mechanically "adopting the whole spec" risks two wasted-effort traps for a stdio server: (1) implementing the `Mcp-Method`/`Mcp-Name` HTTP header requirements (SEP-2243) and `ttlMs`/`cacheScope` cache hints (SEP-2549) — both are Streamable HTTP transport concepts with no stdio equivalent; (2) treating the removal of the `initialize`/`initialized` handshake as mandatory for a stdio server, when a real-world audit of four production stdio MCP servers found "None of this breaks stdio-transport servers immediately — stdio was never part of the stateful-session model in the first place. The exposure is almost entirely on the remote/HTTP+SSE side."

**Why it happens:**
The announcement material is written for the primary audience the spec revision was built for (remote/HTTP operators scaling behind load balancers). A stdio, one-process-per-client server reading that material top-to-bottom will naturally infer more required work than actually applies to it.

**How to avoid:**
Scope the impact assessment (backlog 999.6, the milestone's spine) explicitly to what reaches a stdio transport: version negotiation semantics (`server/discover`, `UnsupportedProtocolVersionError`), the Roots/Sampling/Logging deprecation (does codegraph-go use any of the three? — it does not, per the `internal/mcp` surface: tools-only, no `ClientCapabilities.roots/sampling`, no `ServerCapabilities.logging`), and the SDK's stdio-specific behavior changes (see Pitfall 8, `server/discover` probe cost on stdio). Explicitly write "N/A — HTTP-only" next to SEP-2243 (header routing) and SEP-2549 (cache hints) in the impact assessment so the scoping decision is visible and auditable, not implied by omission.

**Warning signs:**
The impact-assessment document spends more words on HTTP transport mechanics (headers, load balancing, `Mcp-Session-Id`) than on stdio version negotiation. That is the tell that HTTP-centric guidance is being over-applied.

**Phase to address:**
The MCP `2026-07-28` impact-assessment phase (999.6) — first phase of the milestone, before the SDK decision.

---

### Pitfall 2: `LATEST_PROTOCOL_VERSION` pin silently moves the declared wire version on a routine dependency bump

**What goes wrong:**
`internal/mcp/server_test.go` (line 81) sets `req.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION` — an SDK-owned constant, not a value this repo controls. If `mark3labs/mcp-go` ever ships 2026-07-28 support in a routine `go get -u` (no `go.mod` major-version bump, since Go modules don't gate this), the wire-declared protocol version changes with zero code review of the actual behavioral delta, and the test that's supposed to pin behavior instead tracks whatever the dependency now claims. This is exactly the failure the milestone's own "Adopt or dated-defer" requirement calls out: "on adopt, replace the `mcp.LATEST_PROTOCOL_VERSION` pin with an explicit asserted version so wire behavior cannot move silently on a dependency bump."

**Why it happens:**
`LATEST_PROTOCOL_VERSION` is the natural, low-friction choice when writing a test — it's always "correct" from the SDK's point of view and never needs updating by hand. That is precisely what makes it dangerous as a pin: it stops being a *test* of a specific behavior and becomes a *mirror* of the dependency.

**How to avoid:**
Replace every `mcp.LATEST_PROTOCOL_VERSION` reference in test and production code with the literal string codegraph-go has verified it correctly negotiates against real clients (e.g. `"2025-11-25"` until adoption is complete, or `"2026-07-28"` post-adoption), sourced from a single named constant this repo owns (not the SDK's). Add a CI check that fails if `mcp.LATEST_PROTOCOL_VERSION` appears anywhere outside that one constant's definition/tests-of-the-constant-itself — mirroring the pattern already used for the ANSI-isolation archtest in v1.0 Phase 4.

**Warning signs:**
`rg 'LATEST_PROTOCOL_VERSION'` returns hits outside a single, deliberately-named "asserted protocol version" seam.

**Phase to address:**
The "Adopt or dated-defer" phase — this is explicitly named as the deliverable in `PROJECT.md`'s milestone scope.

---

### Pitfall 3: mark3labs/mcp-go does not yet speak 2026-07-28 — "SDK decision" and "spec adoption" are two different clocks

**What goes wrong:**
As of this research (2026-08-03, six days after the spec's final release), `mark3labs/mcp-go`'s latest tagged release (`v0.57.0`) and its README/pkg.go.dev documentation still state it "implements the Model Context Protocol specification version 2025-11-25, with backward compatibility for versions 2025-06-18, 2025-03-26, and 2024-11-05." No evidence was found of a merged PR adding 2026-07-28 support. `modelcontextprotocol/go-sdk` v1.7.0, by contrast, shipped 2026-07-28 support the same day as the spec. This means "adopt the new spec" and "stay on mark3labs" are currently mutually exclusive — a team that decides "defer the SDK swap, just bump mark3labs" cannot land on 2026-07-28 today regardless of how much migration work is done, because the dependency itself hasn't caught up.

**Why it happens:**
Community SDKs lag official ones on day-one spec support by design (mark3labs is unofficial, single-maintainer-adjacent, and the official Go SDK is now the fast-follow reference implementation the spec team ships alongside the spec itself).

**How to avoid:**
Treat "does mark3labs support 2026-07-28 at all yet" as a gating fact-check performed *at execution time*, not at planning time — verify current mark3labs release notes and the spec-support table before writing the "adopt or dated-defer" decision. If mark3labs still doesn't support it, "dated-defer, revisit when mark3labs lands support or when the SDK-swap phase completes" is the only coherent adopt-path without doing the SDK swap first.

**Warning signs:**
Any adoption plan that says "bump mark3labs, get 2026-07-28" without first confirming mark3labs has a release that claims that support.

**Phase to address:**
The MCP `2026-07-28` impact-assessment phase (999.6), as an explicit fact-check gating the SDK-decision phase.

---

### Pitfall 4: The spec explicitly sanctions silent failure for a modern client against a legacy (or dual-era) server — this is not a bug, it's documented behavior

**What goes wrong:**
The official versioning spec (`modelcontextprotocol.io/specification/2026-07-28/basic/versioning`) publishes a client×server outcome matrix. The `Modern client × Legacy server` row reads (verbatim): *"Fails. The server may reject the request with an implementation-defined error, stay silent, or even process an era-ambiguous method under legacy semantics. On stdio, clients **SHOULD** send `server/discover` first to fail deterministically."* Note the word "SHOULD," not "MUST" — a client that skips the `server/discover` probe (permitted by spec) and simply sends a modern-shaped request to codegraph-go's stdio server has spec-sanctioned license to fail in an undefined, possibly-silent way. Since codegraph-go's dynamic tool catalog already returns **zero tools with no error** when there's no `.codegraph/` index (`TestNoIndexZeroTools`), the failure surface a user would see — "no tools" — is *indistinguishable* between "this repo isn't indexed" and "the client and server didn't understand each other." That ambiguity is the milestone's own named highest-severity risk.

**Why it happens:**
The spec designs for a world where a probe (`server/discover`) is available, but makes it optional for clients — a legacy server has no way to force a client to probe, and the outcome when a client doesn't is deliberately left implementation-defined ("may... stay silent").

**How to avoid:**
Do not rely on `server/discover` existing or being called — this milestone's server won't implement it as a modern feature yet (it's a modern-server-only requirement), and even if it did, clients aren't obligated to call it. Instead, instrument the *initialize handshake itself*: have `serve --mcp` log (to stderr, never stdout) the negotiated `protocolVersion` and the requesting `clientInfo.name`/`version` on every session, at a level visible without `--verbose` flags. This turns "which protocol era did this session actually negotiate" from an invisible internal detail into a one-line, always-on fact a user (or Sean, running this daily) can grep out of their agent's MCP debug log.

**Warning signs:**
A user reports "codegraph_explore isn't showing up" and the only diagnostic step available is re-running `codegraph status` to rule out "no index" — with no way to also rule out "the client's protocol era and codegraph-go's didn't match."

**Phase to address:**
"Real-client MCP verification" phase — the harness this phase builds should assert on the negotiated-version log line as a first-class observable output, not just on tool-list contents.

---

### Pitfall 5: Client-side stale tool-list caching is a second, independent silent-failure channel — already observed in the wild, unrelated to version negotiation

**What goes wrong:**
Multiple confirmed GitHub issues (`anthropics/claude-code#41123`, `#40025`, `#50515`; `anthropics/claude-ai-mcp#45`) document MCP clients — including Claude Code itself, one of codegraph-go's 8 installed agent targets — caching a server's `tools/list` response across reconnects, `/mcp` manual reconnects, and even full client restarts, with **no error surfaced** and no reliable client-side invalidation. One issue found the cache was keyed by server *name* in the client config, so identical binaries under a renamed entry showed the correct (fresh) tool list while the original name stayed stuck. This means even a byte-perfect protocol-version migration can still produce "codegraph_explore isn't showing up" reports that have nothing to do with 2026-07-28 at all — they're pre-existing client-side caching bugs that will be misdiagnosed as migration regressions if the team isn't already aware of them.

**Why it happens:**
Client implementations treat `tools/list` as cacheable more aggressively than the spec's `notifications/tools/list_changed` mechanism (or, in 2026-07-28, the `ttlMs`/`cacheScope` hints — HTTP-only, not applicable here) actually guarantees freshness for.

**How to avoid:**
Document this as a known confound *before* the migration ships: when a user reports missing tools post-upgrade, the triage checklist must include "does renaming the server entry in the client config make the tools reappear?" as a fast discriminator between a real codegraph-go regression and a pre-existing client cache bug. This belongs in a troubleshooting note shipped alongside the milestone, not just tribal knowledge.

**Warning signs:**
A "tools missing" bug report where `codegraph status` and a raw stdio protocol trace both look correct.

**Phase to address:**
"Real-client MCP verification" phase — the verification harness should explicitly test the "reconnect after tool-set change" path against at least one real agent client (not just the in-process client), since this is the one silent-failure mode a real client is required to exercise.

---

### Pitfall 6: A naive port to the official Go SDK silently changes error-to-JSON-RPC-error mapping on the wire

**What goes wrong:**
Per the `juburr/mad-skills` migration guide (verified against go-sdk v1.6.1) and confirmed by the SDK's own design doc language around typed handlers: in `mark3labs/mcp-go`, `return nil, err` from a tool handler produces a **protocol-level JSON-RPC error**. In `modelcontextprotocol/go-sdk`'s generic `AddTool` path, only a `*jsonrpc.Error` is treated as a protocol-level error — any other Go `error` returned from the typed handler is instead wrapped into a successful `CallToolResult` with `IsError: true`, and **the error message becomes visible to the LLM as tool output**, not to the transport layer. This is invisible in type signatures (`error` is `error` either way) and would silently change what an agent sees on every failure path — including codegraph-go's confinement-rejection path (`TestOpenEnginePathConfinedToRepoRoot`), which currently returns `result.IsError = true` deliberately (so this specific path is *already* using the "tool error" shape mark3labs also supports) — but any handler that currently does `return nil, err` to signal a protocol failure would change behavior on migration.

**Why it happens:**
The two SDKs made different design choices about what "an error means" at the handler boundary, and neither the Go type system nor a shallow code review surfaces the difference — only reading each SDK's dispatch code (or its docs) does.

**How to avoid:**
Before any SDK swap, audit every `mcp.CallToolResult` construction and every bare `return nil, err` in `internal/mcp/` and classify each as "intended to be a protocol-level failure" vs. "intended to be a tool-visible error the agent should see and potentially retry differently." Convert the former to the official SDK's `*jsonrpc.Error` construction explicitly; leave the latter as a normal Go error. Do this as a deliberate mapping table reviewed alongside the migration diff, not as an incidental side effect of "the code compiles."

**Warning signs:**
Post-migration, an agent that used to receive a hard connection-level error on a malformed request instead receives a normal-looking tool result with `isError: true` and has to parse free text to detect the failure — or vice versa.

**Phase to address:**
SDK-swap implementation (if adoption is chosen this milestone or a later one) — must be a named review step in that phase's plan, not assumed covered by "tests pass."

---

### Pitfall 7: Struct-tag JSON schema generation silently drops enum constraints and changes field-omission behavior

**What goes wrong:**
The official SDK infers tool input schemas from Go struct tags via reflection (`jsonschema.For[T]`), replacing mark3labs's explicit builder functions (`mcp.WithString(..., mcp.Enum(...))`). Per the migration guide's "Known Gotchas": *"Struct-tag-based schema generation does not support enums"* — a tool argument that mark3labs declared as a closed enum (rejecting invalid values at the schema-validation layer, before the handler even runs) silently becomes an unconstrained string unless the migration explicitly sets `Tool.InputSchema` via `jsonschema.For[T]` with `ForOptions{TypeSchemas: ...}` or falls back to the low-level `AddTool`. Separately, the go-sdk v1.7.0 release notes name an actual shipped regression of this exact shape: `ToolAnnotations.ReadOnlyHint`/`IdempotentHint` are bare `bool` (not `*bool`) in the Go type, so `omitempty` can no longer distinguish "false" from "unset" — the SDK's own fix required a `MCPGODEBUG=hintomitempty=1` escape hatch to restore the old wire shape, meaning this is a documented, not hypothetical, JSON-marshalling behavior change between SDK versions.

**Why it happens:**
Reflection-based schema generation from Go's type system cannot express every JSON Schema construct Go's type system doesn't have a native analogue for (enums chief among them); pointer-vs-value field types for optional booleans are a perennial Go/JSON-Schema impedance mismatch that bit the SDK's own authors.

**How to avoid:**
For every existing tool argument with a closed value set (are there any in codegraph-go's tool inputs? — audit `internal/mcp` tool registrations for this before migration), explicitly verify the migrated schema still rejects out-of-set values, by diffing the raw `tools/list` JSON schema output (not a parsed Go struct) before and after migration. Treat any `omitempty`-sensitive boolean field the same way.

**Warning signs:**
A tool that used to reject an invalid enum value with a schema-validation error now silently accepts it and fails later (or not at all) inside the handler.

**Phase to address:**
SDK-swap implementation, if pursued — schema-diff step should be part of the migration's verification loop, sourced from raw wire JSON (see Pitfall 9 below), not from Go struct equality.

---

### Pitfall 8: `server/discover` on stdio has real per-connection cost and a documented spawn-count trap for CLI tools

**What goes wrong:**
The TypeScript SDK's own docs (directly informative for understanding the official Go SDK's parallel design, since both implement the same SEP-2575 mechanism) document that the stdio transport's `server/discover` probe, when run in `'auto'` negotiation mode, spawns a **short-lived sibling process** separate from the real session process — because "some stdio servers exit on any pre-`initialize` request... the probe must not spend the caller's one child process." The same docs explicitly warn: *"Do not default a spawn-per-invocation CLI tool to `'auto'`. On stdio, a legacy server that never answers unknown pre-`initialize` requests stalls `connect()` for the full probe timeout before falling back, and the probe spawns an extra short-lived server process per connect."* This is directly relevant to codegraph-go: every one of the 8 installed agent clients spawns `codegraph serve --mcp` as a subprocess per session. If any of those 8 clients adopts `'auto'` era-negotiation as a default, every single codegraph-go MCP session incurs **two process spawns instead of one** (a throwaway discover-probe process plus the real session process) until codegraph-go itself speaks 2026-07-28 and can answer the probe inline. On a system indexing multiple monorepos with `codegraph install`'d agents, that is a measurable startup-latency and process-count regression codegraph-go did not cause but will be blamed for.

**Why it happens:**
The stdio transport has no way to ask "what era do you speak" without either sending a real (possibly session-mutating) request or spawning a disposable probe process — the SDK authors chose the latter to avoid corrupting a legacy server's single-process assumption.

**How to avoid:**
This is a client-side behavior codegraph-go cannot control directly, but the impact assessment should explicitly note it as a reason to land 2026-07-28 support (so the probe short-circuits to a real `server/discover` reply instead of a timeout-and-fallback) if any of the 8 target clients is confirmed to use `'auto'` mode. Treat "does this client's stdio transport probe-spawn" as one of the 8-agent audit's checklist items, not an afterthought.

**Warning signs:**
Doubled process counts or startup latency for `codegraph serve --mcp` sessions after an agent client's own SDK upgrade — with no codegraph-go-side change at all. This would look like a codegraph-go regression from a client-side dependency bump.

**Phase to address:**
The MCP `2026-07-28` impact-assessment phase's per-agent audit (999.6) — add "stdio probe behavior" as an audited dimension per client.

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|-----------------|-----------------|
| Keep `mcp.LATEST_PROTOCOL_VERSION` in tests "just for this milestone" | No test churn while the SDK decision is pending | Reintroduces Pitfall 2 the moment `go get -u` lands a new spec-supporting release, silently | Never past this milestone's "Adopt or dated-defer" deliverable — that item exists specifically to close this |
| Defer the whole spec assessment because "we're stdio-only, HTTP scaling doesn't apply to us" | Saves the impact-assessment effort | Misses the version-negotiation, deprecation-window, and probe-cost items that DO apply to stdio (Pitfalls 4, 6, 8) — "N/A" is a per-SEP judgment, not a per-revision one | Never as a blanket judgment; acceptable per-SEP once actually checked (see Pitfall 1's prevention) |
| Adopt mark3labs's next release blind, assuming semver-minor bumps are behavior-neutral | Fast, low-review-overhead dependency bump | mark3labs has shipped field-type breaking changes in minor-looking releases before (`Result.Meta` in v0.37.0); nothing in Go module versioning prevents this for a pre-1.0-spirited, actively-evolving library | Never for this dependency without reading the release's "Breaking Changes" section first — mark3labs publishes one when it applies |

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|-------------------|
| The 8-agent installed base (`codegraph install`) | Assuming an agent's MCP client behavior is static once installed — client apps auto-update independently of `codegraph install`/`upgrade` | Treat "which protocol era does each of the 8 agents' current released client actually negotiate" as a fact to re-verify at ship time, not something inferred once and assumed stable; agents update on their own release cadence |
| `mark3labs/mcp-go` in-process test client (`mcpclient.NewInProcessClient`) | Using it to validate the very SDK it's part of — see Pitfall/Testing-trap section below | Real-client verification harness must exercise a *different* implementation than the one under test (see Testing Traps) |
| `govulncheck` against `go.tool.mod`/`go.tool-lint.mod` | Running `govulncheck ./...` from the repo root, which only sees the main module's `go.mod` and silently never touches the ~400 tool-mod dependencies | Invoke govulncheck per-modfile, either via `go tool -modfile=go.tool.mod govulncheck ./...` executed with cwd/GOFLAGS pointed at that modfile, or an explicit `-C`/directory argument that makes `go list` resolve the tool modfile — verify by mutation (see govulncheck section below) |

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|-----------------|
| Client-side `'auto'` era-negotiation probe-spawn (Pitfall 8) | Doubled process count / startup latency per `serve --mcp` invocation, appearing after an *agent's* update, not codegraph-go's | Track per-agent stdio negotiation mode as part of the 8-agent audit; prioritize landing `server/discover` support if any agent is confirmed `'auto'` | The moment any of the 8 installed clients ships an SDK update defaulting to `'auto'` mode against a codegraph-go binary that doesn't yet answer `server/discover` |

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Treating a version-mismatch `UnsupportedProtocolVersionError` (or its absence) as authorization-adjacent | None currently applicable — codegraph-go's confinement checks (`TestOpenEnginePathConfinedToRepoRoot`) are unrelated to protocol version and must not be conflated with it during migration review | Keep the SDK-swap review's error-mapping audit (Pitfall 6) scoped to protocol-level vs. tool-level error semantics only; do not let it touch or "helpfully improve" the path-confinement trust boundary in the same change |

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-------------------|
| No visible log of the negotiated protocol version per session (Pitfall 4) | User sees "no tools" and has no way to distinguish "not indexed" from "protocol mismatch" without a raw wire capture | Always-on stderr line per session: negotiated version + clientInfo, cheap and non-intrusive, closes the diagnostic gap named as this milestone's top risk |
| Silently-stale client tool cache (Pitfall 5) misattributed to codegraph-go | User files a codegraph-go bug for what is actually a client-side caching defect they can't see | Ship a troubleshooting note pointing at the "rename the server entry" discriminator test before assuming a codegraph-go regression |

## "Looks Done But Isn't" Checklist

- [ ] **"Adopt or dated-defer" decision**: Often missing the *asserted-version constant* replacing `mcp.LATEST_PROTOCOL_VERSION` — verify with `rg 'LATEST_PROTOCOL_VERSION'` returning zero hits outside one definition site.
- [ ] **8-agent impact audit**: Often missing an actual check of each client's *current shipped* SDK/protocol support, substituting an assumption instead — verify each of the 8 has a dated, sourced note (release version + protocol era it negotiates), not a general "should be fine."
- [ ] **Real-client MCP verification harness**: Often "real" only in the sense of "not literally the SDK's Go client," while still round-tripping through parsed Go structs somewhere in the assertion path — verify by finding the exact line that reads raw bytes off the child process's stdout pipe, independent of any MCP SDK's own JSON-RPC decoder.
- [ ] **govulncheck over tool modfiles**: Often "wired into CI" while actually scanning zero of the ~400 tool-module dependencies — verify by deliberately introducing (in a throwaway branch) a known-vulnerable version pin into `go.tool.mod` and confirming the CI job goes red before merging that verification away.
- [ ] **Dated-defer decision (if chosen)**: Often missing a concrete trigger condition and calendar reminder mechanism — verify a dated defer names both an explicit removal-eligibility date (spec revision release + 12 months, i.e. no earlier than 2027-07-28) and a re-check owner/mechanism, not just prose acknowledging the window exists.

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|-----------------|
| `LATEST_PROTOCOL_VERSION` pin drifted silently (Pitfall 2) | LOW | Grep, replace with the asserted constant, add the CI guard; no data/state is at risk, this is a build-time-only fix |
| SDK-swap error-mapping regression shipped (Pitfall 6) | MEDIUM | Diff `tools/call` error responses (raw JSON) between the pre- and post-migration binaries against a fixed set of known-failing inputs (malformed args, confinement violation, missing index); patch the specific handlers whose shape changed |
| A client silently mis-negotiates in production (Pitfall 4) | LOW once instrumented, HIGH before | Ship the stderr negotiated-version log line; ask the affected user to paste it — turns an otherwise-unbounded investigation into a one-line diagnostic |

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|-------------------|---------------|
| HTTP-scaling guidance over-applied to stdio (P1) | MCP impact-assessment phase (999.6) | Impact-assessment doc explicitly marks each SEP N/A or applicable-with-reason for stdio |
| `LATEST_PROTOCOL_VERSION` silent drift (P2) | Adopt-or-defer decision phase | `rg` CI guard: zero hits outside the one owned constant |
| mark3labs lacks 2026-07-28 support today (P3) | MCP impact-assessment phase (999.6), gating the SDK-decision phase | Decision doc cites the mark3labs release/README checked, dated |
| Spec-sanctioned silent failure on version mismatch (P4) | Real-client MCP verification phase | Harness asserts the stderr negotiated-version+clientInfo log line is emitted every session |
| Client-side stale tool-list caching (P5) | Real-client MCP verification phase | Harness includes a "reconnect after tool-set change" scenario against a real (not in-process) client |
| Naive error-mapping port (P6) | SDK-swap implementation phase (if pursued) | Raw-JSON diff of `tools/call` error responses pre/post migration across a fixed fixture set |
| Struct-tag schema silently drops enums/omitempty (P7) | SDK-swap implementation phase (if pursued) | Raw `tools/list` JSON schema diff pre/post migration, not Go-struct comparison |
| `server/discover` stdio probe-spawn cost (P8) | MCP impact-assessment phase (999.6) 8-agent audit | Per-agent audit row records confirmed negotiation mode (`legacy`/`auto`/`pin`) where discoverable |
| Vacuous same-SDK conformance tests (see Testing Traps) | Real-client MCP verification phase | Mutation-proof: corrupt a wire byte / drop a required field and confirm the specific test goes red |
| `govulncheck` scanning nothing over tool modfiles (see govulncheck section) | Tool-modfile vulnerability scanning phase (999.3) | Deliberately-introduced known-vulnerable tool-mod dependency turns the job red before the fix lands |
| Deprecation-window discovered late (see Deprecation-window section) | Adopt-or-defer decision phase | Dated-defer entry includes explicit calendar trigger date + re-check owner, checked into `PROJECT.md`/`ROADMAP.md` backlog |

---

## Testing Traps — vacuous MCP conformance patterns (highest-value section per the research brief)

This project's own hard-won rule — a gate is not trusted until demonstrated RED against a confirmed-applied mutation — is the correct lens for every pattern below. Each entry gives the pattern, the mutation that *should* turn it red, and whether it actually would.

### Trap A: Using an SDK's own client to validate that SDK's own server

**The pattern (INFERRED — repo-specific, direct code reading):** `internal/mcp/server_test.go`'s `listToolNames` helper builds an `mcpclient.NewInProcessClient(s)` — a `mark3labs/mcp-go` client — against a `server.MCPServer` built by the same `mark3labs/mcp-go` library, then calls `c.ListTools`/`c.CallTool` through it. Every test in that file (`TestDefaultToolVisibility`, `TestAllowlist`, `TestNoIndexZeroTools`, `TestExploreHandlerDelegatesToEngine`, `TestOpenEnginePathConfinedToRepoRoot`, `TestConfinementAnchoredOnRepoRootNotStartPath`) goes through this same in-process, same-SDK client.

**Why this specific case is (mostly) not vacuous today, and where it would become vacuous:** These tests correctly validate codegraph-go's *own* logic (tool allowlisting, confinement, handler delegation) — the SDK is acting as inert plumbing between two pieces of codegraph-go's own code (`BuildServer` and the assertions), so a bug in codegraph-go's confinement logic or allowlist parsing genuinely turns them red. **The trap specifically applies to the migration**: if this same in-process-client pattern is reused to "verify" that the server *correctly speaks the 2026-07-28 wire protocol* or that a mark3labs→official-SDK swap didn't break wire compatibility, it becomes circular — the SDK's own client necessarily encodes/decodes using the SDK's own (possibly incorrect, possibly-just-changed) understanding of the wire format, so a bug in that shared understanding is invisible to a client built from the identical code.

**The mutation that should turn it red:** Deliberately have the server emit a wire-malformed frame (e.g. a stray non-JSON `fmt.Println` on stdout mid-session — exactly the "noisy" bug pattern documented as "the single most common way a working server fails in the wild") — an in-process client bypasses the wire (stdio bytes) entirely, so **this mutation would not turn the in-process tests red at all**, regardless of SDK.

**Prevention:** For anything claiming to verify *wire* behavior (as opposed to codegraph-go's own business logic, where the in-process pattern above remains legitimate), the "real-client MCP verification" harness must spawn the actual `codegraph serve --mcp` binary as a real subprocess and read/write real stdio bytes — reusing the project's own precedent from v1.0 Phase 4's `TestOutputHygieneStdoutIsJSONRPCOnly`-style raw-reader test (built specifically because "mcp-go's own client silently skips malformed lines and cannot fail a purity test"). Ideally use a *second, independent* MCP client implementation (the official Go SDK's client, or a hand-rolled minimal NDJSON reader, or the TypeScript reference client via `npx @modelcontextprotocol/inspector`) rather than mark3labs's own client, so a version-negotiation or schema-shape bug shared between "the server's SDK" and "the test's SDK" cannot hide.

**Phase to address:** Real-client MCP verification phase — explicitly required by `PROJECT.md` to land *before* any SDK swap.

---

### Trap B: Asserting on parsed Go objects instead of raw wire bytes

**The pattern:** A test calls `c.ListTools(ctx, ...)` and asserts on the resulting `[]mcp.Tool` slice, or calls `c.CallTool` and asserts on `result.Content[0].(mcp.TextContent).Text`. This is the *only* pattern currently used in `server_test.go` (see `TestExploreHandlerDelegatesToEngine` asserting via `mcp.AsTextContent`).

**Why it's dangerous specifically for a protocol-revision migration:** Every field-omission change (Pitfall 7's `omitempty`-on-bare-bool regression, real and shipped in go-sdk v1.7.0), every JSON-RPC error-code change (the go-sdk's own issue #976 shows a nonexistent-method error returning code `0` instead of the standard `-32601` over stdio — invisible if a test only checks `err != nil` or `result.IsError`), and every schema-shape drift (enum dropped, field renamed) is only visible in the *serialized bytes on the wire* — a Go struct comparison after both the encode and decode round-trip through the same library's (possibly buggy, possibly-just-changed) marshal/unmarshal code cannot see a bug in that exact round-trip.

**The mutation that should turn it red:** Change a JSON-RPC error's numeric code from the spec-correct `-32601` to the SDK's actual bug-shipped `0` (a real, cited go-sdk issue, not hypothetical). A parsed-object test that only asserts `result.IsError == true` or `err != nil` **would not go red** — both are still true regardless of the numeric code. A raw-bytes test asserting the literal `"code":-32601` substring (or a parsed-but-independently-decoded JSON map, not the SDK's typed error struct) **would** go red.

**Prevention:** For any assertion whose entire purpose is proving wire-format correctness (as opposed to codegraph-go's own business-logic correctness, where typed assertions remain appropriate and idiomatic), decode the raw response with `encoding/json` into a generic `map[string]any` or assert directly on the byte string, never through the SDK-under-test's own typed unmarshal path.

**Phase to address:** Real-client MCP verification phase.

---

### Trap C: Golden files regenerated from current behavior

**The pattern:** A snapshot/golden test captures "whatever the server currently outputs" as the expected value, then re-generates that golden file whenever the output changes (the `github-mcp-server` migration material found in this research literally documents this exact tool: `UPDATE_TOOLSNAPS=true go test ./...`, with the explicit warning "you should however, only update the toolsnaps after confirming that the schema changes are intentional and correct").

**Why it's dangerous:** A golden file regenerated *during* the SDK-swap migration itself captures whatever the new SDK happens to produce — including any of the unintended shape changes from Pitfall 7 — and then asserts that shape is "correct" forever after, because the person running `UPDATE_TOOLSNAPS=true` was focused on making CI pass, not on independently re-deriving what the schema *should* be.

**The mutation that should turn it red:** Regressing a real tool's schema (dropping a required field) *and simultaneously* regenerating the golden file in the same commit — by construction, this **never turns red**, because the golden file and the regression are updated together. This is the git-diff-of-generated-artifact class of vacuous gate this project's own `planning-artifacts.md` rule describes for a different domain (tool-owned files), applied here to test fixtures.

**Prevention:** Golden/snapshot files for MCP schema shape must be reviewed as a diff against a *hand-authored* expected schema (or the pre-migration golden, from TS CodeGraph or the pre-swap SDK, whichever is the actual source of truth for what the shape should be) at the moment they're regenerated — never regenerated and merged in the same review pass as the behavior change they're meant to catch. Practically: CI should refuse to auto-accept a golden-file diff in the same PR that also changes SDK version pins or tool-registration code, forcing a human to look at exactly the diff this trap would otherwise hide.

**Phase to address:** SDK-swap implementation phase (if pursued) — golden-file update discipline should be a stated rule in that phase's plan, not implicit.

---

### Trap D: Asserting a tool list is non-empty rather than set-equal

**The pattern:** `if len(got) == 0 { t.Fatal(...) }` instead of `equalStrings(got, want)`.

**Why it matters here specifically:** codegraph-go's actual tests already do this correctly — `TestDefaultToolVisibility` and `TestAllowlist` both assert exact set equality (`equalStrings(got, want)`), and `TestNoIndexZeroTools` asserts the *inverse* (exact zero), which is the strictest possible form. This is worth stating explicitly as a **positive existing pattern to preserve**, precisely because the failure mode this trap describes is so easy to regress toward during a migration: a v0.56.0→official-SDK port that accidentally starts registering an extra internal/debug tool, or drops one real tool while gaining an unrelated one (net-zero count, wrong set), would pass a non-empty check and pass a same-*count* check, but only a full set-equality assertion catches it.

**The mutation that should turn it red:** Swap the allowlist's expected tool `"node"` for an unregistered similarly-named tool while keeping the count the same. A `len(got) == 3` check stays green; `equalStrings(got, want)` goes red.

**Prevention:** Keep the existing set-equality discipline; explicitly re-verify it wasn't loosened to a count-only or non-empty check during the SDK migration's test-signature churn (the official SDK's 3-return-value handler signature and struct-based schema changes are exactly the kind of mechanical, high-diff-volume change where a "just make it compile and pass" pass can silently weaken an assertion).

**Phase to address:** SDK-swap implementation phase (if pursued) — a diff-review checklist item: "did any `equalStrings`/set-equality assertion get replaced with a weaker one during the mechanical port?"

---

## `govulncheck` over tool modfiles — non-vacuous gate checklist

The milestone's tool-modfile vulnerability scanning item (999.3, closing the ~400-module credentialed-CI-tooling gap) has several silent-pass failure shapes, catalogued here with what makes each demonstrably non-vacuous:

1. **Wrong-directory / wrong-modfile scan.** `govulncheck ./...` run from the main module's root only ever resolves the main `go.mod`'s dependency graph — Go's tool-modfile mechanism (`go.tool.mod`, introduced by the Go 1.24 `tool` directive design) is a *separate* module graph, invisible to a `govulncheck` invocation that doesn't explicitly target it. **Non-vacuous proof:** run `go list -modfile=go.tool.mod tool` (or the equivalent for `go.tool-lint.mod`) and confirm the CI job's actual invocation targets that same modfile — then deliberately downgrade one tool dependency in `go.tool.mod` to a version with a known, call-graph-reachable CVE (the tutorial-standard `golang.org/x/text@v0.3.5` / `GO-2021-0113` pair is a well-documented, reproducible example) and confirm the job goes red. Revert and confirm green.
2. **`-scan module` (or `-mode` equivalent) instead of the default `-scan symbol`.** Module-level scanning degrades govulncheck to "is the vulnerable module anywhere in the graph," discarding the call-graph reachability analysis that is govulncheck's entire value proposition over a naive advisory-matching scanner — this can *look* like the right tool while actually behaving like the noisy tool it was chosen to replace. **Non-vacuous proof:** confirm the CI invocation's flags don't include a coarser `-scan`/`-mode` override, and that a vulnerability in an *unreached* function of a real tool dependency reports as "informational," not a hard failure (proving the reachability analysis is actually running, not just presence-matching).
3. **Exit code swallowed by a pipe.** Any CI step piping `govulncheck`'s output through another command (`| tee`, `| jq`, a custom formatter) without checking `${PIPESTATUS[0]}` (bash) or equivalent loses the tool's own non-zero exit code — a real finding then produces "readable output" but a green CI job. **Non-vacuous proof:** confirm the CI step's exit-code check is against govulncheck's own exit status, not the exit status of a downstream pipe stage; verify with the same deliberately-downgraded-dependency mutation as (1) and confirm the *job* (not just the log output) is marked failed.
4. **No reachable entry point in a tool binary.** Some of the ~400 tool-mod dependencies are used only by tools invoked in narrowly-scoped ways (a linter plugin, a code-generator run once at build time) — govulncheck's call-graph analysis may correctly report zero reachable vulnerabilities for a tool whose vulnerable code path genuinely isn't exercised by how codegraph-go's CI invokes it. This is not a bug in the gate, but it *looks* identical to a mis-scoped scan from the outside. **Non-vacuous proof:** the same deliberately-introduced-CVE mutation (1) must be performed against a dependency that genuinely IS reachable from at least one tool's actual invocation (e.g. a CVE in a code path the linter/tool actually calls during its normal CI invocation, not merely present in its binary) — a scan that stays green against a genuinely-reachable known-vulnerable version, but only in that one case, points at (1) or (2), not at legitimate unreachability.

## Deprecation-Window Traps

**What goes wrong for projects that defer:** SEP-2596's window is measured **from the spec revision's release date (2026-07-28), not from when a given implementation notices or starts caring** — the clock is already running regardless of codegraph-go's own migration timeline. The SEP text itself flags the subtler trap: the 12-month floor is only observable in practice *if* the SDKs a project depends on support every revision released within that trailing window — "a consumer that updates the SDK between releases can move directly from one that predates the deprecation to one that postdates removal, never seeing the Deprecated marker" at all. For codegraph-go specifically: Roots/Sampling/Logging are irrelevant (the server implements none of the three — tools-only per `PROJECT.md`), so that half of SEP-2577 imposes zero migration burden regardless of timing. The HTTP+SSE transport deprecation is *also* irrelevant (stdio-only). The item that actually matters for a deferred decision is **version-negotiation behavior itself** (SEP-2575/2567) — that's a core-protocol mechanism, not a lifecycle-tagged feature, and has no analogous 12-month grace window; a legacy server simply stops being able to talk to a client that has fully removed legacy `initialize` support, whenever that client-side removal happens (which is NOT bound by the SEP-2596 clock at all, since that clock governs the *specification's own* removal of legacy-era support, not any individual client's choice to drop it early).

**What a deferring project should put in place now:**
1. A dated calendar entry, not just a prose note: earliest possible spec-mandated removal of the HTTP+SSE transport and Roots/Sampling/Logging is **2027-07-28 or later** (12 months from the 2026-07-28 revision release) — but since neither applies to codegraph-go, this date is informational only, not action-forcing.
2. The action-forcing date is **not** the SEP-2596 deprecation floor — it's whenever any of the 8 installed agent clients' *own* release cadence drops legacy `initialize` support entirely (unknowable in advance, not governed by the spec's grace period). The correct mitigation is the always-on negotiated-version stderr log (Pitfall 4) as an early-warning system, not a calendar date.
3. A named owner and re-check trigger recorded directly in the dated-defer decision itself (per `PROJECT.md`'s own requirement: "an explicit dated defer is an acceptable outcome... What is not acceptable is leaving the choice implicit") — e.g. "re-check mark3labs's 2026-07-28 support status and re-run the 8-agent negotiation audit at the next milestone boundary."

## Sources

- `modelcontextprotocol.io/specification/2026-07-28/changelog` (official spec changelog) — MEDIUM confidence (verified/cross-checked)
- `blog.modelcontextprotocol.io/posts/2026-07-28/` and `.../2026-07-28-release-candidate/` (official MCP blog, lead maintainers) — MEDIUM confidence (verified/cross-checked)
- `modelcontextprotocol.io/specification/2026-07-28/basic/versioning` (official versioning/compatibility spec page, client×server outcome matrix) — MEDIUM confidence (verified/cross-checked)
- `modelcontextprotocol.io/seps/2577-deprecate-roots-sampling-and-logging` and `.../seps/2596-spec-feature-lifecycle-and-deprecation` (official SEP text) — MEDIUM confidence (verified/cross-checked)
- `github.com/modelcontextprotocol/go-sdk` releases (v1.7.0 release notes, Version Compatibility table) — MEDIUM confidence (verified/cross-checked)
- `github.com/mark3labs/mcp-go` releases and `pkg.go.dev/github.com/mark3labs/mcp-go` (release history, current README/doc text) — MEDIUM confidence (verified/cross-checked)
- `github.com/juburr/mad-skills` — `go-mcp/references/migration-from-mark3labs.md` (community migration guide, verified against go-sdk v1.6.1) — LOW/MEDIUM confidence, single source but internally detailed and cross-consistent with `github.com/txn2/mcp-s3` PR #22 and `github/github-mcp-server` migration PRs
- `github.com/modelcontextprotocol/go-sdk/issues/976` (stdio malformed-JSON-RPC / error-code-0 bug report) — MEDIUM confidence, primary-source GitHub issue on the official SDK's own repo
- `github.com/anthropics/claude-code/issues/41123`, `#40025`, `#50515`; `github.com/anthropics/claude-ai-mcp/issues/45` (client-side stale tool-list caching reports) — MEDIUM confidence, primary-source GitHub issues, several with maintainer/bot responses confirming the behavior
- `ts.sdk.modelcontextprotocol.io/v2/protocol-versions.html` (TypeScript SDK docs on era negotiation, `server/discover` probe mechanics on stdio) — MEDIUM confidence (official SDK docs); used as informative parallel for the Go SDK's equivalent SEP-2575 mechanism, not a direct Go-SDK citation
- `likeone.ai/blog/mcp-server-spec-migration-audit-guide-2026` (real-world audit of 4 production stdio MCP servers against 2026-07-28) — LOW confidence, single vendor blog, but directly on-point and internally consistent with the official spec's own framing
- `www.alexedwards.net/blog/how-to-manage-tool-dependencies-in-go-1.24-plus` (canonical `-modfile`/tool-directive walkthrough) — LOW confidence, single blog source, but consistent with `go.dev/ref/mod` official documentation cited alongside it
- `go.dev/ref/mod` (official Go Modules Reference, `tool` directive) — MEDIUM confidence (official documentation)
- `dev.to/gabrielanhaia/govulncheck-scan-go-code-for-cves-you-can-actually-reach` and `go.dev/doc/tutorial/govulncheck` — LOW/MEDIUM confidence respectively; used for `-scan symbol` vs `-scan module` reachability-analysis distinction
- codegraph-go repo: `internal/mcp/server_test.go` (direct code reading, not web research) — used for Pitfalls 2, Testing Traps A/D, and the "N/A: Roots/Sampling/Logging unused" determination in the Deprecation-Window section
- codegraph-go repo: `.planning/PROJECT.md` (milestone scope, `PROJECT.md`'s own risk framing) — direct project-artifact reading, used throughout for phase-mapping and to confirm which pitfalls the milestone has already anticipated vs. which are new findings from this research

---
*Pitfalls research for: MCP 2026-07-28 protocol migration + Go MCP SDK swap on an installed, stdio-only, tools-only agent server*
*Researched: 2026-08-03*
