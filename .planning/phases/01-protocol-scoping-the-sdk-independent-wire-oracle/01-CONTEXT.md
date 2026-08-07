# Phase 1: Protocol Scoping & the SDK-Independent Wire Oracle - Context

**Gathered:** 2026-08-04
**Status:** Ready for planning

<domain>
## Phase Boundary

Build a verification oracle that reads the actual bytes on stdio and can genuinely fail — proven green against today's unmodified `mark3labs`-backed `serve --mcp` — plus the dated, evidence-backed scoping the rest of v0.3.0 is measured against.

**In scope:** VRFY-01 (raw-wire harness), VRFY-02 (repo-owned protocol version + CI guard), VRFY-03 (always-on negotiated-version stderr line), VRFY-04 (harness proven pre-migration), VRFY-05 (dated 8-agent negotiation audit), SDK-02 (`serve.go` imports no MCP SDK package). Plus the two non-requirement deliverables folded in from backlog 999.6: a SEP-by-SEP stdio applicability table, and the Team Scale strategic read-out recorded as a decision.

**Explicitly NOT in scope:** any change to which SDK `internal/mcp` is built on. `mark3labs/mcp-go v0.56.0` stays in `go.mod` through the end of this phase — that is Phase 2's work, and VRFY-04 makes the ordering a hard constraint, not a preference.

</domain>

<decisions>
## Implementation Decisions

### Oracle Form & Expected Values

- **D-01:** The oracle is a **standalone capture tool**, not a set of assertions embedded in tests. It spawns *any* binary path, drives a scripted request list over real stdio, and emits a normalized transcript. Tests invoke it and diff against frozen transcripts. The same tool runs against the pre-migration binary now, the post-migration binary in Phase 2, and a released asset later. — **Reversibility:** costly — Phases 2 and 3 both consume the frozen transcripts as their comparison baseline; changing the tool's output shape mid-milestone invalidates every transcript and the pre-migration baseline cannot be recaptured once `mark3labs` leaves `go.mod`.

- **D-02:** Expected values are **captured-and-frozen for the bulk, plus a small hand-authored set of spec anchors** that do *not* derive from the capture — the `-32601` code for an unknown method, the `protocolVersion` literal, JSON-RPC framing invariants. If the SDK swap changes an accident, the transcript diff shows it; if it changes something the spec pins, the hand-authored anchor goes red independently.

- **D-03:** Anti-regeneration is enforced by **two structural mechanisms**, not convention: (1) **no** `UPDATE_TRANSCRIPTS=1`-style path exists — the tool writes a transcript to stdout and a human redirects it deliberately; (2) a CI check fails when a single PR's diff touches **both** a frozen transcript **and** `go.mod`'s MCP dependency or `internal/mcp/*.go`. Both operate at PR granularity, which survives this repo's squash-merge model where per-commit discipline would not.
  - **Note for Phase 3:** transcripts *must* legitimately change there (`resultType`, `_meta.serverInfo`, `ttlMs` are deliberate wire additions). The rule is not "never regenerate" — it is "never regenerate in the same change that could have caused the diff." Phase 2 = frozen, no exceptions. Phase 3 = regenerate as its own separate, reviewable PR.

- **D-04:** Normalization is **named-field placeholder substitution only**. Bytes stay verbatim except an explicit, documented allowlist of unstable values replaced with placeholders (fixture temp path → `<REPO>`, ldflags version → `<VERSION>`, timestamps → `<TS>`). Everything not on the list is compared byte-for-byte. — **Reversibility:** costly — widening the allowlist later silently changes what every existing frozen transcript asserts; narrowing it forces a full regeneration. **Explicitly rejected:** canonicalizing by decoding and re-encoding through `encoding/json`, because that erases field presence/absence and ordering — the only place PITFALLS Pitfall 7's `omitempty`-on-bare-`bool` regression (real and shipped in go-sdk v1.7.0) is visible.

### Oracle Coverage Bar

- **D-05:** The oracle covers **all 8 tools + envelope + error/edge shapes**, roughly 18 scenarios: the handshake; all three `tools/list` variants (default explore-only, allowlisted subset, zero-tools-no-index); a `tools/call` for each of the 8 tools; and unknown-method, unknown-tool, malformed-args, and confinement-reject. This matches Phase 2's literal success criterion ("every existing MCP tool produces byte-identical output") with no interpretation gap. — **Reversibility:** **one-way** — any scenario not captured in this phase can never be captured against the pre-migration server afterward. Once Phase 2 removes `mark3labs` from `go.mod`, the `mark3labs` baseline is gone. Under-scoping here is unrecoverable; over-scoping costs only build time.

- **D-06:** The oracle **scripts older-protocol-version handshakes now** and freezes what today's server answers at each revision `mark3labs` supports (`2025-11-25`, `2025-06-18`, `2025-03-26`, `2024-11-05`) plus one unsupported version. Phase 3's SPEC-06 then asserts against a real pre-migration Legacy baseline instead of writing its first multi-era test against the SDK that already claims to handle five eras. — **Reversibility:** **one-way** — same unrecapturability as D-05, applied to what REQUIREMENTS.md names "the single highest-consequence mistake available in this milestone."

- **D-07:** Non-vacuity is proven **two ways**. One-time, demonstrated-and-reverted with the mutation confirmed applied before the result is trusted: a stray non-JSON stdout line, a dropped tool, a changed JSON-RPC error code, a changed `protocolVersion` literal. Permanent in-suite: a scenario-count assertion so the scenario list cannot silently shrink, an empty-transcript-is-never-a-match assertion, and a normalization self-defeat case proving each placeholder rule matches only its intended field. The permanent set deliberately attacks the *harness*, not the payload — a byte-exact comparator is near-impossible to make vacuous on content, so the real vacuity risks are structural.

- **D-08:** The oracle captures against a **dedicated frozen fixture** — a small purpose-built source tree checked in solely for the wire oracle, which does not change once transcripts are frozen. Any transcript diff is then unambiguously a server change, never fixture drift, and edits to the shared `test/integration` fixture cannot cascade into bulk regeneration pressure. — **Reversibility:** costly — the fixture and the frozen transcripts are a matched pair; changing it invalidates all of them, and the pre-migration set cannot be rebuilt.

### 8-Agent Negotiation Audit (VRFY-05)

- **D-09:** The measurement instrument is a **proxying capture shim** that each agent launches instead of `serve --mcp`. It records the raw `initialize` frame verbatim — `protocolVersion`, `clientInfo`, `capabilities` — then proxies stdio to the real server so the agent keeps working during the audit. Chosen over reading the VRFY-03 stderr line because it is independent of whether a given agent surfaces or discards codegraph's stderr, and it can observe a second process spawn if a client probes.

- **D-10:** Completeness bar is **measured-where-possible with structurally distinct `UNMEASURED` rows**. Every row is either MEASURED (captured frame + date) or UNMEASURED (blocking reason: no account, platform gap, etc.). A doc-sourced value may appear, but only in a separate column explicitly labelled unverified — **never** in the measured column. A doc-read must not be able to quietly occupy the slot where a measurement belongs.

- **D-11:** Each row records **client name and shipped version, the `protocolVersion` offered vs. negotiated, the declared capabilities block, and probe behavior** (pre-`initialize` request / second process spawn / neither). The probe column is PITFALLS Pitfall 8's early warning and is free to collect while the shim is already in place; it directly informs whether Phase 3's `server/discover` work is latency-urgent or merely spec compliance.

- **D-12:** Artifact locations: the SEP applicability table and the agent audit go in **`docs/`** (alongside the existing `docs/FLAG-PARITY.md` / `docs/RELEASE-PROCEDURES.md` precedent) — they have ongoing operational value, get re-run, and answer user-facing questions. The **Team Scale read-out goes in `.planning/`** — it is strategic input to a future unscoped milestone, not shipped user documentation.

### Negotiated-Version stderr Line (VRFY-03)

- **D-13:** The always-on session line reports **the version requested, the version negotiated, client identity, and the advertised tool count**. The tool count is what collapses the "no tools" ambiguity in a single read: `tools=0` with a clean protocol pair means not indexed; `tools=0` with a version mismatch means negotiation. Reporting requested-vs-negotiated separately makes a silent downgrade visible rather than inferred. — **Reversibility:** costly — the audit shim and the oracle both parse it once it ships.

- **D-14:** Format is a **fixed prefix + `key=value` pairs**, e.g. `codegraph: mcp-session requested=2025-11-25 negotiated=2025-11-25 client=claude-code/1.2.3 tools=1`. Greppable by prefix, parseable without a JSON decoder, readable when pasted out of an agent log, and consistent with this repo's existing plain-stderr diagnostics — no structured-logging framework introduced.

- **D-15:** The oracle **freezes stdout bytes only**; stderr gets a targeted per-scenario assertion (the session line is present exactly once, parses, and carries the expected keys/values). Freezing stderr would drag watcher chatter, Pebble `Errorf` output, and timing-dependent lines into the frozen bytes — each becoming a normalization rule or a flake, and a flaky oracle gets relaxed.

- **D-16:** The session line is an **additive-only contract**: the `codegraph: mcp-session` prefix and the key names shipped in this phase never change or disappear; later phases may add keys (Phase 3 may want `discover=` or `cacheable=`). Documented in the troubleshooting doc and enforced by a format test, so a rename is a deliberate reviewed act. Shim and oracle parse defensively so additions do not break them. — **Reversibility:** **one-way** — this is a published diagnostic users will be asked to paste; breaking it breaks the audit shim, the oracle, and any user tooling built on it.

### Claude's Discretion

The user explicitly delegated these, with the leanings below recorded as starting positions, not locks:

- **VRFY-02 — version constant and guard mechanism.** Leaning: the repo-owned constant declares `"2025-11-25"` in this phase (what the current server actually negotiates), and the "no SDK-owned protocol constant anywhere in the tree" guard is a `go/packages` archtest with a self-defeat guard, following the v1.0 Phase 4 (stdout purity) and Phase 6 (ANSI isolation) precedent rather than a CI grep. Rationale: this repo has already been bitten by an inverted `rg -qv` gate. The guard must also survive Phase 2's SDK swap — it should forbid the *class* (an SDK-owned protocol-version constant), not one library's spelling.
- **SDK-02 — the `internal/mcp.Server` seam shape.** `BuildServer`'s `*server.MCPServer` return type *is* the leak; the seam must own the serve call so `serve.go` never names an SDK package. Whether that is a concrete type with a `Serve` method or an interface is the planner's call — but the shape determines how cheap Phase 2 is, and it should also be the natural hook for D-13's session line.
- **Tool location and CI run scope.** Where the capture tool lives in the tree, and whether all ~18 scenarios run on every PR or a narrower set gates while the full sweep runs on a schedule (18 subprocess spawns is not free).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

ROADMAP.md carries no `Canonical refs:` line for this phase; the list below was accumulated from REQUIREMENTS.md, the milestone research, and the codebase scout.

### Milestone scope and requirements
- `.planning/ROADMAP.md` § "Phase 1: Protocol Scoping & the SDK-Independent Wire Oracle" — the five success criteria this phase is verified against, and the "Ordering is load-bearing" preamble explaining why VRFY-04 is a hard sequencing constraint
- `.planning/REQUIREMENTS.md` § Verification (VRFY-01…05) and § SDK Migration (SDK-02) — the requirement text; also § Out of Scope, which locks "never drop Legacy `initialize` support" and excludes the dated-defer branch
- `.planning/PROJECT.md` § "Current Milestone: v0.3.0" — the milestone's own risk framing, including the quiet 8-agent failure mode

### Research (the load-bearing doc for this phase)
- `.planning/research/PITFALLS.md` — **read in full.** Specifically: Pitfall 1 (HTTP-scaling guidance over-applied to stdio → drives the SEP table), Pitfall 2 (`LATEST_PROTOCOL_VERSION` drift → VRFY-02), Pitfall 4 (spec-sanctioned silent failure → VRFY-03's justification), Pitfall 8 (`server/discover` stdio probe-spawn → D-11's probe column), and the entire **Testing Traps** section — Trap A (SDK-as-its-own-oracle), Trap B (parsed objects vs raw bytes), Trap C (goldens regenerated from current behavior → D-03), Trap D (set-equality vs non-empty)
- `.planning/research/SUMMARY.md`, `ARCHITECTURE.md`, `FEATURES.md`, `STACK.md` — not read during this discussion; the researcher should check them for anything bearing on the oracle or the audit

### Existing wire-level precedent
- `test/integration/mcp_stdout_purity_test.go` — the direct ancestor of the capture tool. Spawns the real binary, hand-frames JSON-RPC, reads `cmd.StdoutPipe()` with a `bufio.Scanner`, decodes into an anonymous struct, never through an SDK's typed unmarshal path. Its header comment documents *why* `mcp-go`'s own client cannot fail a purity test
- `test/integration/main_test.go` — `runBinary` / `copyFixture` harness (real binary, real fixture, real temp dir)

### Current MCP surface (the migration target)
- `internal/mcp/server.go` — `BuildServer` at :94, the `version` const at :26, `companionNames` at :32, `ParseAllowlist` at :42
- `internal/mcp/tools.go` — all 8 tool registrations and handlers
- `internal/cli/serve.go` — :13 (the sole production `mark3labs/mcp-go/server` import) and :252-253 (`BuildServer` → `server.ServeStdio`)
- `internal/mcp/server_test.go`, `internal/mcp/tools_schema_drift_test.go`, `internal/mcp/reconnect_test.go`
- `testdata/golden/golden_parity_test.go` — :1397 and :1438 build `mcpclient.NewInProcessClient`; :1477 uses `mcp.LATEST_PROTOCOL_VERSION`

### Repo standards this phase inherits
- `.claude/CLAUDE.md` § Alternatives Considered — the standing commitment to re-evaluate the MCP SDK at each milestone boundary (discharged by the maintainer's 2026-08-03 pre-decision)
- `.planning/milestones/v1.0-phases/04-output-hygiene/` — the archtest + raw-reader precedent D-07 and the VRFY-02 guard both build on
- `.planning/milestones/v1.0-phases/06-rendering-seam-pretty-status-files/` — the ANSI-isolation archtest's self-defeat guard, the pattern for "this test cannot pass vacuously"

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets

- **`test/integration/mcp_stdout_purity_test.go`** — the capture tool's core loop already exists here in miniature: `exec.Command(binPath, "serve", "--mcp")`, `StdinPipe`/`StdoutPipe`, a `syncBuffer` for stderr (guarding against the `os/exec` stderr-copy goroutine race, WR-01), hand-framed request writing, and a `bufio.Scanner` with a 10 MB buffer cap. Lift and generalize rather than reinvent.
- **`test/integration/main_test.go`** — `runBinary` and `copyFixture` give binary-building and fixture-staging for free.
- **`internal/mcp.ParseAllowlist` / `WarnUnknownToolsTo`** — the `CODEGRAPH_MCP_TOOLS` mechanism the oracle drives to produce the allowlisted-subset `tools/list` scenario.
- **The `CODEGRAPH_NO_WATCH=1` env** — the purity test already uses it to remove watcher timing nondeterminism without changing anything the wire assertions depend on. The oracle should do the same.

### Established Patterns

- **Archtests over greps.** v1.0 Phase 4 (`go/packages` closure walk forbidding `os.Stdout`/bare `fmt.Print*`) and Phase 6 (ANSI isolation with a self-defeat guard) are the house style for "this must never appear in the tree." A grep-shaped gate in this repo has already been found inverted.
- **Mutation-proof-or-it-does-not-count.** A gate is not trusted until demonstrated RED against a **confirmed-applied** mutation. Confirm the mutation landed before trusting the red — a mutation that no-ops proves nothing.
- **Set-equality, never non-empty.** `TestDefaultToolVisibility` and `TestAllowlist` assert exact set equality; `TestNoIndexZeroTools` asserts exact zero. PITFALLS Trap D names this a positive existing pattern to preserve through the migration's test churn.

### Integration Points

- **`internal/cli/serve.go:252-253`** is where SDK-02 lands: `mcp.BuildServer(...)` returns `*server.MCPServer` and `server.ServeStdio(s)` is called from `internal/cli`. Both must move behind the `internal/mcp` seam. This is also the natural place to hang D-13's session line, since it is the one point that sees both the negotiated handshake and the built tool catalog.
- **`internal/mcp/server.go:94`** — `BuildServer`'s return type is the leak itself, not just its callers.

### Findings from the scout that change downstream assumptions

1. **There is no wire coverage today besides the purity test.** `testdata/golden/golden_parity_test.go:1397,1438` drives the server through `mcpclient.NewInProcessClient` — PITFALLS Trap A exactly. It bypasses stdio entirely, so the existing golden parity corpus covers **zero** wire bytes. The oracle is not supplementing wire coverage; it *is* the wire coverage. This is why D-05 chose the full 8-tool bar.
2. **PITFALLS Pitfall 6's error-mapping risk looks materially lower than the research feared.** Every handler in `internal/mcp/tools.go` returns `mcp.NewToolResultError(err.Error()), nil` — the tool-visible error shape both SDKs support. There are no bare `return nil, err` protocol-error paths to reclassify. **Phase 2 must confirm this rather than inherit it**, but it should recalibrate the expected size of SDK-04.
3. **`LATEST_PROTOCOL_VERSION` has exactly 6 sites**, all in tests: `internal/mcp/server_test.go:81`, `testdata/golden/golden_parity_test.go:1477`, and `test/integration/{mcp_stdout_purity,watch_live_sync,watch_default,worktree_notice}_test.go`. VRFY-02's guard has a small, enumerable blast radius.

</code_context>

<specifics>
## Specific Ideas

- The session line's concrete shape, as discussed: `codegraph: mcp-session requested=2025-11-25 negotiated=2025-11-25 client=claude-code/1.2.3 tools=1`.
- The 8-agent roster is fixed and named in ROADMAP.md success criterion 5: Claude Code, Cursor, Codex CLI, opencode, Gemini CLI, Hermes, Antigravity, Kiro.
- The capture shim should **proxy**, not terminate — the agent must keep working while the audit runs, so the audit is something that can be done on a working machine rather than in a torn-down environment.
- The anti-regeneration CI guard's trigger set is deliberately narrow and named: a frozen transcript **plus** (`go.mod`'s MCP dependency **or** `internal/mcp/*.go`). It is not a blanket "no golden changes."

</specifics>

<deferred>
## Deferred Ideas

- **Per-agent tool-cache-across-reconnect observation** (PITFALLS Pitfall 5) — considered as a fifth audit column and deliberately left out. It is a client-side defect codegraph cannot fix, already documented against Claude Code itself (`anthropics/claude-code#41123`, `#40025`, `#50515`). It belongs in a troubleshooting note carrying the "rename the server entry in the client config" discriminator, shipped alongside the milestone — **not** as an audit dimension. Worth revisiting if a real "tools missing" report lands during v0.3.0.
- **Freezing stderr into the transcripts** — rejected for this phase (D-15) on flake risk. If a stderr-noise regression ever ships (the HYG-01 failure class), revisit as a separately-scoped stderr transcript with its own normalization rules.
- **Extending the oracle to the interactive TUI surface** — out of scope and already tracked as backlog 999.2 (tmux real-PTY harness). The failure *shape* rhymes; the surface and harness do not.

</deferred>

---

*Phase: 01-protocol-scoping-the-sdk-independent-wire-oracle*
*Context gathered: 2026-08-04*
