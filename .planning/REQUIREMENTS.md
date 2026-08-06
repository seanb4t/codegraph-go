# Requirements: CodeGraph Go — v0.3.0 (MCP Protocol Currency)

**Defined:** 2026-08-03
**Core Value:** An agent user can uninstall TypeScript CodeGraph, install the Go binary, migrate their indexes, and everything works the same or better — faster, from a single verifiably-built binary.

**Milestone framing.** MCP published spec revision `2026-07-28` on 2026-07-28, six days before this milestone was scoped. codegraph-go's MCP server is a **Legacy** implementation under that spec's own terminology. This milestone brings it current, and does so without breaking the 8 agent clients `codegraph install` already configures.

**Pre-decided by the maintainer (2026-08-03), superseding the "adopt or dated-defer" framing this milestone was captured under:** adopt `github.com/modelcontextprotocol/go-sdk@v1.7.0`. The evidence was one-sided — `mark3labs/mcp-go@v0.57.0` pins `LATEST_PROTOCOL_VERSION = "2025-11-25"`, has no `2026-07-28` support, no announced timeline, and one unclaimed tracking issue (#928). The official SDK shipped stable on 2026-07-27 with five-era negotiation built in. The dated-defer branch is therefore **not** in this milestone's scope; see Out of Scope.

## v0.3.0 Requirements

### MCP Spec Compliance (SPEC)

- [ ] **SPEC-01**: An MCP client can call `server/discover` and receive the server's capabilities without first calling any tool — the spec makes this a **MUST for servers**, not an optional discovery convenience
- [ ] **SPEC-02**: The server validates per-request `_meta` (protocol version, client identity, client capabilities), answering `-32602` on malformed or missing required fields and `UnsupportedProtocolVersionError` (`-32022`, SEP-2575) on an unsupported version, rather than failing silently or proceeding on assumptions
- [ ] **SPEC-03**: Every tool result carries `resultType: "complete"`
- [ ] **SPEC-04**: `tools/list` and `server/discover` responses carry `ttlMs: 0` and `cacheScope: "private"`, so no client caches a catalog this server cannot promise is still accurate
- [ ] **SPEC-05**: A user who runs `codegraph init` while an MCP server is already running sees the tools appear — `hasIndex` is re-checked per call rather than snapshotted once at server construction
- [ ] **SPEC-06**: An agent client speaking any prior protocol revision (`2025-11-25` and earlier) continues to work against the upgraded server, asserted by test rather than assumed from the SDK's documentation
- [ ] **SPEC-07**: `server/discover`'s `instructions` field carries codegraph usage guidance, so an agent gets orientation without spending a tool call
- [ ] **SPEC-08**: Tool results carry `io.modelcontextprotocol/serverInfo` in `_meta`, so a client debugging a negotiation problem can see which server version answered
- [ ] **SPEC-09**: A client that opts into `subscriptions/listen` is notified via `notifications/tools/list_changed` when the tool catalog changes, with `tools.listChanged: true` advertised

### SDK Migration (SDK)

- [x] **SDK-01**: `internal/mcp` runs on `modelcontextprotocol/go-sdk@v1.7.0`, with all existing tools semantically unchanged from the pre-migration server across the wire-oracle corpus — every transcript diff read line by line, every changed line explained, none regenerated wholesale
- [x] **SDK-02**: `internal/cli/serve.go` no longer imports an MCP SDK package directly — the process bootstrap goes through a narrow `internal/mcp.Server` interface, closing the one production-code SDK leak
- [x] **SDK-03**: `mark3labs/mcp-go` is removed from `go.mod`, and the resulting dependency closure is re-audited through the existing `govulncheck` and SBOM paths
- [x] **SDK-04**: Error-to-JSON-RPC mapping is explicitly audited and asserted — a handler returning a plain `error` becomes a *protocol* error under mark3labs but a tool-visible `IsError: true` *result* under the official SDK, a wire-behavior change invisible in the Go type signature
- [ ] **SDK-05**: Tool input schemas are explicitly audited for constraints lost in translation (notably enum constraints dropped by struct-tag reflection), with any loss either fixed or recorded as a deliberate divergence

### Verification (VRFY)

- [x] **VRFY-01**: A verification harness asserts on **raw stdio wire bytes**, not SDK-typed Go objects, and does not use the SDK under test as its own oracle
- [x] **VRFY-02**: The server's declared protocol version is asserted against a **repo-owned literal**, not an SDK-owned constant, with a CI guard proving no stray `LATEST_PROTOCOL_VERSION`-style references remain — a dependency bump must never move wire behavior silently
- [x] **VRFY-03**: The server logs the negotiated protocol version to stderr on every connection, always on — the only available mitigation for a spec-sanctioned silent version mismatch
- [x] **VRFY-04**: The harness passes against the **current, pre-migration** server before any SDK change lands, establishing it as a genuine regression oracle rather than a description of the new code's behavior
- [x] **VRFY-05**: A dated audit records which protocol revision each of the 8 roster agent clients negotiates, re-run immediately before the migration lands rather than relied on from earlier research

### Supply Chain (VULN)

- [ ] **VULN-01**: `govulncheck` covers `go.tool.mod` and `go.tool-lint.mod` — approximately 400 third-party modules executed as credentialed CI tooling — via `-mode=binary` over binaries built from those manifests, replacing the current `task vuln` target which was reproduced to be a no-op duplicate of the main-module CI scan
- [ ] **VULN-02**: The scanning job is demonstrated RED against a deliberately known-vulnerable pin before it is trusted
- [ ] **VULN-03**: The job's blocking-versus-advisory stance is stated explicitly in the workflow, so an advisory job cannot be mistaken for a gate

### Maintenance (MAINT)

- [ ] **MAINT-01**: Issue #13 — the daemon `-race` failure on the `getppid` test seam — is fixed, with the race demonstrated before the fix
- [ ] **MAINT-02**: Issue #17 — `TestRunWatchdogCancelsRunOnSimulatedReparent` failing under full-suite load while passing isolated — is fixed at the cause rather than by isolating the test
- [ ] **MAINT-03**: The GoReleaser version pin agrees between `ci.yml` and `release.yml` (currently v2.17.1 vs v2.17.0)

## Future Requirements

Acknowledged, deliberately not in v0.3.0.

### Team Scale (remote / multi-user)

- **Informed by this milestone.** The `2026-07-28` stateless core removes the sticky-routing / shared-session-store infrastructure a central codegraph MCP server would otherwise have needed. This milestone records that read-out as a decision; it builds none of it. codegraph-go's existing design does not foreclose it — `path` is already an explicit per-call tool argument and every handler already opens a fresh query-engine snapshot per call.

### MRTR / elicitation

- **MRTR-01**: The server can ask the user a question mid-call (e.g. "no index found here — initialize now?") via `resultType: "input_required"`, rather than returning an empty tool set. Genuinely enabled for the first time by this spec revision; deferred because it is new product behavior, not protocol currency.

### Tasks extension

- **TASK-01**: Long-running operations exposed via `io.modelcontextprotocol/tasks`. Not applicable today — codegraph's MCP tools are fast read-only queries, and indexing happens on the CLI/watcher side.

## Out of Scope

| Feature | Reason |
|---------|--------|
| Streamable HTTP transport | codegraph-go is stdio-only. The spec's headline changes are HTTP-scaling-driven, and its HTTP-only mechanics — `Mcp-Method`/`Mcp-Name` routing headers (SEP-2243), SSE resumability, load-balancer fan-out — have no surface here. Adding a transport is Team Scale work, not currency work |
| Dated-defer branch | Superseded by the maintainer's pre-decision to adopt. Retained here as an explicit exclusion so the milestone does not silently reacquire a decision it has already made |
| Dropping Legacy `initialize` support | Per the spec's own compatibility matrix, a Modern-only server **hard-fails** every still-Legacy client with no client-side fallback. Zero of the 8 roster clients have confirmed `2026-07-28` support. This is the single highest-consequence mistake available in this milestone |
| Roots / Sampling / Logging | Deprecated by this exact revision (SEP-2577). codegraph-go never declared them and already uses the spec-endorsed replacements — tool-argument paths, no sampling need, stderr logging |
| Authorization work (CIMD, DCR, RFC 9207 `iss`) | Applies to remote/HTTP MCP servers. A stdio subprocess has no OAuth surface |
| Bespoke dual-era negotiation logic | Would have been a P1 work item, but `go-sdk@v1.7.0` ships five-era negotiation (`2026-07-28` → `2024-11-05`) with `negotiatedVersion()` preferring the client's version. Inherited from the dependency choice — writing our own would be building a shadow SDK |
| Backlog 999.2 (tmux TTY harness), 999.4 (CheckRegression guard), 999.5 (Gatekeeper) | Orthogonal surfaces. 999.2's failure *shape* rhymes with VRFY-01, but it is the TUI surface and a different harness |

## Traceability

Populated during roadmap creation (2026-08-03). Phase numbering restarts at 1 for this milestone, matching the repo's convention (v0.1 was Phases 1–8, v1.0 was Phases 1–10).

| Requirement | Phase | Status |
|-------------|-------|--------|
| SPEC-01 | Phase 3 | Pending |
| SPEC-02 | Phase 3 | Pending |
| SPEC-03 | Phase 3 | Pending |
| SPEC-04 | Phase 3 | Pending |
| SPEC-05 | Phase 3 | Pending |
| SPEC-06 | Phase 3 | Pending |
| SPEC-07 | Phase 3 | Pending |
| SPEC-08 | Phase 3 | Pending |
| SPEC-09 | Phase 5 | Pending |
| SDK-01 | Phase 2 | Complete |
| SDK-02 | Phase 1 | Complete |
| SDK-03 | Phase 2 | Complete |
| SDK-04 | Phase 2 | Complete |
| SDK-05 | Phase 2 | Pending |
| VRFY-01 | Phase 1 | Complete |
| VRFY-02 | Phase 1 | Complete |
| VRFY-03 | Phase 1 | Complete |
| VRFY-04 | Phase 1 | Complete |
| VRFY-05 | Phase 1 | Complete |
| VULN-01 | Phase 4 | Pending |
| VULN-02 | Phase 4 | Pending |
| VULN-03 | Phase 4 | Pending |
| MAINT-01 | Phase 4 | Pending |
| MAINT-02 | Phase 4 | Pending |
| MAINT-03 | Phase 4 | Pending |

**Coverage:**

- v0.3.0 requirements: 25 total
- Mapped to phases: 25 ✓
- Unmapped: 0 — no orphans, no duplicates; every requirement maps to exactly one phase

**Per-phase distribution:**

| Phase | Requirements | Count |
|-------|--------------|-------|
| Phase 1 — Protocol Scoping & the SDK-Independent Wire Oracle | VRFY-01, VRFY-02, VRFY-03, VRFY-04, VRFY-05, SDK-02 | 6 |
| Phase 2 — SDK Migration (official go-sdk) | SDK-01, SDK-03, SDK-04, SDK-05 | 4 |
| Phase 3 — `2026-07-28` Spec Compliance | SPEC-01, SPEC-02, SPEC-03, SPEC-04, SPEC-05, SPEC-06, SPEC-07, SPEC-08 | 8 |
| Phase 4 — Supply-Chain Coverage & Daemon Substrate Fixes | VULN-01, VULN-02, VULN-03, MAINT-01, MAINT-02, MAINT-03 | 6 |
| Phase 5 — Live Tool-Catalog Change Notification | SPEC-09 | 1 |

**REQ-ID prefix note (2026-08-03):** `SPEC`, `SDK`, `VRFY`, `VULN`, and `MAINT` were chosen specifically to avoid collision with the prefixes already used by v0.1 (`AGNT ARCH CLI COMM DIST EMBED INDX LANG MCP MIGR PERF QRY RES SYNC VIZ`) and v1.0 (`DEV DMON EXPL HOOK HYG NODE REL STAT SURF TEST TUI WATCH WORK`). In particular **`MCP-01`–`MCP-04` are already taken by v0.1** — the v0.3.0 research artifacts reference "the existing MCP-03 gap", which is a v0.1 requirement ID, not a new one. This repo has already lost time to exactly this class of collision (a decision gate failed on a `D-08a` citation colliding with a v0.1 decision ID left in `serve.go` comments).

**Ordering note:** VRFY-04 is a hard sequencing constraint, not a preference. This project established in v1.0 Phase 4 that an SDK's own client silently skips malformed stdout lines and therefore cannot fail a purity test. A harness written after the swap, validated against the new SDK using that SDK's client, would be circular — it would describe the new behavior rather than test it.

---
*Requirements defined: 2026-08-03*
*Last updated: 2026-08-03 after roadmap creation — traceability populated, 25/25 mapped across 5 phases*
