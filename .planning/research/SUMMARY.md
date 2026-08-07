# Project Research Summary

**Project:** codegraph-go
**Domain:** MCP protocol-currency milestone (v0.3.0 "MCP Protocol Currency") — SDK decision + spec-compliance impact assessment for an already-shipped, stdio-only, tools-only, 8-agent-installed MCP server
**Researched:** 2026-08-03
**Confidence:** MEDIUM-HIGH overall — see per-area breakdown below. The single fastest-decaying finding in the whole research set is the 8-agent-roster empirical audit (Q6 in FEATURES.md); everything else is durable through at least the next milestone boundary.

## Executive Summary

codegraph-go's MCP server is currently a **Legacy** implementation under MCP's own new terminology: `mark3labs/mcp-go@v0.57.0` pins `LATEST_PROTOCOL_VERSION = "2025-11-25"`, has no announced timeline for `2026-07-28` support, and its one relevant open issue (#928, SEP-2575) is unclaimed. Against that, the coordinator directly verified — against the Go module proxy and module source, not blog posts — that `github.com/modelcontextprotocol/go-sdk@v1.7.0` is a **stable** release, published 2026-07-27 (the day before the spec's own public announcement), and it ships **five-era protocol negotiation out of the box** (`2026-07-28` down to `2024-11-05`, with `negotiatedVersion()` preferring whatever the calling client supports) plus the SEP-2575 error codes (`CodeUnsupportedProtocolVersion = -32022`). This single fact reshapes the milestone: adopting the official SDK is not "build dual-era support ourselves," it is "inherit dual-era support for free by choosing a dependency." The architecture research's original framing of "dual-era server support" as a bespoke P1 engineering item is superseded by this — it becomes a property of the SDK choice, not code this project writes.

The recommended approach is a strict four-phase sequence: (1) an impact assessment that correctly scopes what a stdio-only, tools-only server actually needs from a spec revision whose headline framing is HTTP-scaling-driven (most of the wire-level "table stakes" — `server/discover`, per-request `_meta`, `ttlMs`/`cacheScope` correctness, `UnsupportedProtocolVersionError` — do bind stdio, but HTTP-only mechanics like `Mcp-Method`/`Mcp-Name` headers and SSE resumability do not); (2) a real-client, raw-stdio-bytes verification harness built *before* any SDK swap, generalizing the existing `mcp_stdout_purity_test.go` precedent, specifically because this project already proved in v1.0 Phase 4 that an SDK's own client can silently swallow malformed wire output and cannot be trusted as its own oracle; (3) the SDK adopt/dated-defer decision itself, now made with the harness in hand and the "wait for tooling to mature" excuse for deferring no longer true; and (4) either the SDK swap (a bounded, mechanical port confined to `internal/mcp/`, not an architectural rewrite) or an explicit, dated, honestly-reasoned defer. Independently, tool-modfile vulnerability scanning (`task vuln`) is confirmed live to be a **no-op duplicate of the CI gate** — it re-scans the ~146-module main `go.mod`, not `go.tool.mod`'s ~368 third-party tool modules, leaving that closure with zero vulnerability coverage today despite running with full CI credentials.

The key risks are not "did we implement the spec correctly" but "can we verify we implemented it correctly without fooling ourselves," and "will breaking Legacy support hard-fail every currently-installed agent client." On the first: this research catalogs concrete, previously-seen vacuous-test patterns (same-SDK-validates-same-SDK, parsed-Go-object assertions instead of raw wire bytes, golden files regenerated in the same commit as the regression they should catch, non-empty checks masquerading as set-equality checks) that must be actively avoided, not merely intended-away. On the second: per the spec's own compatibility matrix, a Modern-only server hard-fails every still-Legacy client with no fallback — and as of this research, **zero of the project's 8 roster agent clients have confirmed `2026-07-28` support**, so dropping Legacy `initialize` support prematurely is the single highest-consequence mistake available in this milestone.

## Key Findings

### Recommended Stack

This is a decision question, not a technology list. `mark3labs/mcp-go` (current dependency, `v0.56.0` pinned, `v0.57.0` latest) does not and cannot speak `2026-07-28` today — verified directly against its source, not inferred. `modelcontextprotocol/go-sdk` (the official Go SDK) is the only path to `2026-07-28` currency and is a mature, stable, GitHub-scale-validated dependency (serving 500K+ users via GitHub's own MCP server per its own release notes) as of `v1.7.0` (2026-07-27, stable, not pre-release). Neither SDK exposes a constructor-time "pin this exact protocol version" option — whichever is chosen, the milestone must decouple its own protocol-version test assertions from any SDK-owned constant (a repo-owned literal, asserted against the actual wire-negotiated version) rather than continuing to reference `mcp.LATEST_PROTOCOL_VERSION`, which silently moves on a routine dependency bump.

**Core technologies / decision points:**
- `github.com/modelcontextprotocol/go-sdk@v1.7.0` (stable, verified via module proxy) — the only dependency offering `2026-07-28` support today, with five-era negotiation and SEP-2575 error codes built in. Migration cost: a bounded, mechanical port of `internal/mcp/tools.go` (8 handler bodies → 8 typed structs) and `server.go`, plus a parallel rewrite of every test currently driving the server via `mark3labs`'s in-process client.
- `github.com/mark3labs/mcp-go` (current, `v0.57.0` available, safe independent bump) — remains viable only under an explicit dated-defer; has no committed timeline for spec currency and its one relevant tracking issue explicitly scopes out most of the revision's actual content even if merged.
- `govulncheck -mode=binary` per tool binary, built fresh from `go.tool.mod`/`go.tool-lint.mod` — the verified-working replacement for the current no-op `task vuln` target. `-mode=source` with `-modfile` fails outright (`build flag -modfile only valid when using modules`); binary-mode, built and scanned per-tool, was proven live to surface real, distinct findings the main-module scan misses.

### Expected Features

**Must have (table stakes, if `2026-07-28` is adopted at all):**
- Per-request `_meta.protocolVersion`/`clientCapabilities` validation, with `-32602` on malformed/missing fields and `UnsupportedProtocolVersionError` (`-32022`) on version mismatch
- `server/discover` implementation — this is a spec-level **MUST for servers**, not a nice-to-have discovery convenience; it fully resolves what the milestone context treated as an open question
- `resultType: "complete"` on every tool result
- `ttlMs`/`cacheScope` correctness on `tools/list` and `server/discover` — see the ttlMs resolution below; this is the one field whose *value*, not mere presence, directly prevents the milestone's named "tools vanish after init" failure mode
- Dual-era serving (Legacy `initialize` fallback + Modern per-request `_meta`) during any transition — **now inherited for free from the official SDK's choice**, not bespoke work, per the coordinator-verified facts
- A per-call (not per-process) `hasIndex` re-check in the `tools/list`/`BuildServer` path, so `ttlMs`'s promise is actually true

**Should have (differentiators, second wave):**
- `tools.listChanged: true` + `notifications/tools/list_changed` via `subscriptions/listen` — a materially bigger lift (new long-lived-stream mechanism, absent from the current SDK entirely) than the `ttlMs=0` fix, and not needed for correctness once `ttlMs=0` ships
- `server/discover`'s `instructions` field, populated with codegraph-usage guidance
- `io.modelcontextprotocol/serverInfo` in result `_meta`

**Defer / anti-features (explicitly do NOT adopt this milestone):**
- Streamable HTTP transport, `Mcp-Method`/`Mcp-Name`/`x-mcp-header` — HTTP-only, N/A to a stdio-only product
- Roots/Sampling/Logging — deprecated by this exact spec revision (SEP-2577); codegraph-go already uses the spec-endorsed replacements natively (tool-arg paths, no sampling need, stderr logging)
- Dropping Legacy `initialize` support immediately "on spec-currency grounds alone" — per the compatibility matrix, converts every still-Legacy roster client from "works" to "fails, hard, with no client-side fallback"

### The `ttlMs` question — resolved

FEATURES.md and ARCHITECTURE.md initially framed this differently (a long `ttlMs` is mechanically defensible since the tool set genuinely cannot change within one process's life vs. a short/near-zero `ttlMs` is the honest product answer). **ARCHITECTURE.md's self-correction is the reading that survives**: a long `ttlMs` is a true statement about the *mechanism* but a dangerous one about the *promise* — it actively encourages clients to stop re-checking exactly the connection most likely to have gone stale (a server started before `.codegraph/` existed, permanently toolless for that connection's life under today's register-once code). The resolution: ship **`ttlMs: 0`, `cacheScope: "private"`** as the honest default, paired with making `hasIndex` genuinely per-call rather than per-process — the two are two halves of one correctness property, not independent options.

### The dual-era / "quiet failure" question — resolved

The milestone context frames the roster-compatibility risk as "quiet" (tools silently not advertised). The spec's actual compatibility matrix is worse than that framing in the direction that matters: **Legacy client × Modern server = fails, hard, with no client-side fallback mechanism at all** — not quiet degradation, an outright connection failure. The "quiet" framing does apply, but only to a different cell: a *Dual-era* client that incorrectly implements the spec's mandated any-error fallback rule against our current Legacy server. Per the coordinator-verified facts, adopting `modelcontextprotocol/go-sdk` inherits dual-era serving automatically (it negotiates across all five eras), which converts the highest-consequence risk in this research (hard roster breakage) into a non-issue *if and only if* the SDK swap is chosen and its version-negotiation surface is used as shipped, not narrowed to Modern-only.

### Architecture Approach

The existing seam (`internal/mcp` → `internal/query` → `internal/graphstore`, archtest-enforced) is untouched by this milestone and must stay untouched. The actual gaps are: (1) `internal/cli/serve.go` imports `mark3labs/mcp-go/server` directly just to call `ServeStdio` on `BuildServer`'s return value — a real, cheap-to-fix leak (`internal/mcp.Server` interface) worth doing now, independent of the SDK decision, since it's the same seam a future Team Scale milestone will need anyway; (2) the test surface is far more deeply coupled — nearly every existing test drives the server through `mark3labs`'s in-process client, which is the actual expensive part of any SDK swap, not the production code. Team Scale readiness is a genuine, favorable side-effect of the spec's statelessness rework (no sticky-session/shared-session-store infrastructure needed for a future remote server), and nothing in the current design forecloses it — `path` is already an explicit per-call tool argument, every handler already opens a fresh query engine snapshot per call, and the one process-lifetime cache (`gitmeta.CachingDetector`) is a pure performance optimization, not a correctness dependency.

**Major components:**
1. `internal/cli/serve.go` — process bootstrap; owns the one `BuildServer`/`ServeStdio` call pair; the one production-code SDK leak worth fixing now
2. `internal/mcp/server.go` + `tools.go` — tool registration and 8 handler bodies; the direct-port surface for an SDK swap; not abstractable behind a schema DSL without building a shadow SDK
3. `test/integration/mcp_stdout_purity_test.go` — the existing raw-stdio, SDK-independent pattern to generalize into the milestone's required verification harness
4. `internal/query.Engine` / `internal/graphstore` — untouched, SDK-agnostic query pipeline

### Critical Pitfalls

1. **Over-applying HTTP-scaling guidance to a stdio server** — most `2026-07-28` commentary is written for remote/load-balanced deployments; the impact assessment must explicitly mark each SEP N/A-for-stdio or applicable-with-reason, not infer scope from HTTP-centric material.
2. **`mcp.LATEST_PROTOCOL_VERSION` as a test pin** — an SDK-owned constant used to assert the SDK's own behavior is not a test, it's a mirror; replace with a repo-owned literal, CI-guarded (`rg` for stray references).
3. **Two independent silent-failure channels, not one** — (a) the spec-sanctioned "modern client × legacy server: may reject, may stay silent" outcome, mitigated only by an always-on stderr negotiated-version log line, and (b) real, already-observed client-side `tools/list` caching bugs (confirmed GitHub issues against Claude Code itself) that will be misdiagnosed as migration regressions unless documented as a known confound before the migration ships.
4. **Vacuous-test traps specific to this migration**: same-SDK-validates-same-SDK (the in-process client pattern is legitimate for codegraph-go's own business logic but circular for wire-protocol claims), asserting parsed Go objects instead of raw wire bytes (misses real, cited SDK bugs like a `0` error code instead of `-32601`), golden files regenerated in the same commit as the regression they exist to catch, and non-empty checks silently replacing set-equality checks during high-diff-volume mechanical porting.
5. **`task vuln` currently scans nothing new** — reproduced live: it re-scans the main module under a different modfile flag, not `go.tool.mod`'s ~368 modules; the fix (`-mode=binary` per tool binary) is verified working, not merely proposed.

## Implications for Roadmap

Based on combined research, the critical path is strictly sequential for the SDK question and independent for everything else:

### Phase 1: MCP `2026-07-28` Impact Assessment
**Rationale:** Every other phase depends on knowing precisely what a stdio-only, tools-only server must do — this phase produces that scoping and, critically, re-runs the 8-agent roster audit close to execution since it is this research's fastest-decaying finding.
**Delivers:** A scoped SEP-by-SEP applicability table (N/A vs. applicable-with-reason), a dated 8-agent negotiation-mode audit, and the Team Scale strategic read-out recorded as a decision entry (not a phase).
**Addresses:** Table-stakes feature list from FEATURES.md; Pitfalls 1, 3, 8 (HTTP over-application, mark3labs's lack of support, `server/discover` probe-spawn cost).
**Avoids:** Treating unscheduled community work (mark3labs issue #928) as a plan; assuming either adoption direction is settled without re-checking.

### Phase 2: Real-Client MCP Verification Harness
**Rationale:** Must exist and pass against the *current* server before any SDK code changes — this is a hard prerequisite, not a preference, per this project's own v1.0 Phase 4 precedent (an SDK's client can silently skip malformed wire output).
**Delivers:** Raw-stdio-bytes assertions (not SDK-typed objects) covering `server/discover`, statelessness-on-first-request, `ttlMs`/`cacheScope` presence, renumbered error codes, deterministic `tools/list` ordering, and the negotiated-version stderr log line.
**Uses:** Generalizes `test/integration/mcp_stdout_purity_test.go`'s existing pattern; optionally layers a JSON-Schema validator against the spec's own published schema as a non-circular second check.
**Implements:** The `internal/mcp.Server` narrow interface fix at the `ServeStdio` call site can land here at near-zero marginal cost, closing the one real production-code SDK leak.

### Phase 3: SDK Decision (Adopt vs. Dated Defer)
**Rationale:** Now has both the impact assessment and a working harness in hand — the decision can be verified, not just argued.
**Delivers:** A recorded decision. Per the coordinator-verified facts, `modelcontextprotocol/go-sdk@v1.7.0` is stable and already inherits five-era negotiation, meaning the "wait for tooling to mature" justification for deferring is no longer honest. **A dated defer remains a sanctioned outcome**, but the only argument that survives the correction is production-track-record caution — a ~1-week-old stable release versus mark3labs's longer field history on a now-superseded revision — and the decision record must name that tradeoff explicitly, not fall back to "no SDK supports this yet."
**Addresses:** Pitfall 2 (replace `LATEST_PROTOCOL_VERSION` with a repo-owned constant regardless of outcome).

### Phase 4a: SDK Swap (if ADOPT) or 4b: Dated Defer Record (if DEFER)
**Rationale:** Mutually exclusive branches gated by Phase 3.
**4a delivers:** A bounded, mechanical port of `internal/mcp/tools.go`/`server.go` (8 handler bodies, direct transcription of existing field descriptions into typed structs), `ttlMs: 0`/`cacheScope: "private"` wiring paired with the per-call `hasIndex` fix, and migration of every test off `mark3labs`'s client — verified against Phase 2's harness, unmodified, staying green. Must explicitly audit error-to-JSON-RPC-error mapping (Pitfall 6) and struct-tag schema generation for dropped enum constraints (Pitfall 7) as named review steps, not assumed-covered-by-tests-passing.
**4b delivers:** A decision record naming the SEP-2596 12-month floor (no earlier than 2027-07-28) as informational only (irrelevant clock for codegraph-go, since Roots/Sampling/Logging and HTTP+SSE don't apply), with the actual action-forcing trigger being the always-on negotiated-version stderr log plus a named owner/re-check mechanism at the next milestone boundary.
**Avoids:** Anti-Patterns 1-3 from ARCHITECTURE.md (shadow schema-builder DSL, same-SDK verification, coupling catalog cache-control to content-freshness reconciliation).

### Phase 5: Tool-Modfile Vulnerability Scanning (999.3)
**Rationale:** Independent of the SDK decision except that it should run *after* Phase 3/4 so it scans the final dependency closure, not a soon-to-change one.
**Delivers:** `govulncheck -mode=binary` per tool built fresh from `go.tool.mod`/`go.tool-lint.mod`, replacing the currently-proven-no-op `task vuln` target. Must be validated non-vacuous by deliberately introducing a known-vulnerable pin and confirming the job goes red before merging the fix.

### Phase 6 (float freely): Daemon test-seam fixes + GoReleaser pin reconciliation
**Rationale:** No dependency on the MCP work; bundle the daemon fixes with Phase 4a if adopting (same file neighborhood, reduces churn), otherwise schedule independently. GoReleaser pin reconciliation is pure housekeeping, schedule wherever convenient.

### Phase Ordering Rationale

- The verification-harness-before-SDK-swap ordering is not a preference, it is the same non-negotiable this project already proved necessary in v1.0 Phase 4 — sequencing it any other way reintroduces a known, previously-experienced failure mode (an SDK's client silently masking wire bugs).
- The 8-agent roster audit belongs at the start of Phase 1 *and* must be re-run close to Phase 3's execution — it is explicitly the fastest-decaying input in this entire research set, and the whole risk calculus (dual-era necessity) hinges on it.
- Bundling the vulnerability-scanning phase after the SDK decision avoids re-running a scan whose target dependency graph is about to change.

### Research Flags

Needs research at execution time (not resolvable further by desk research now):
- **Phase 1's 8-agent audit** — explicitly flagged by FEATURES.md as the fastest-decaying finding; re-run immediately before use, not trusted from this research pass.
- **Phase 3's SDK decision** — should re-verify mark3labs's release notes at execution time in case it has shipped `2026-07-28` support in the interim (unlikely but cheap to check).

Standard patterns (well-documented, low research need during planning):
- **Phase 2's harness** — direct generalization of an existing, working pattern in this codebase; no new technique required.
- **Phase 5's vulnerability scanning** — the exact working invocation shape was verified live in this research; implementation is mechanical.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Every version claim (mark3labs v0.57.0 lacking support, go-sdk v1.7.0 stable and five-era-capable) was fetched live from GitHub's API/raw source or reproduced by direct local execution — not recalled from training data or inferred from the MCP blog. |
| Features | MEDIUM-HIGH | Spec-text findings (`_meta`, `server/discover`, `ttlMs`/`cacheScope`, compatibility matrix) are HIGH — primary source, directly quoted, cross-referenced across the official spec's own pages. The 8-agent empirical adoption claims (Q6) are explicitly LOW/UNKNOWN by design — zero of 8 roster clients confirmed on `2026-07-28` either way, and all dated evidence predates the spec's finalization. Do not let this harden into a fact. |
| Architecture | MEDIUM | Code-derived findings (file/symbol enumeration, seam analysis) are HIGH — read directly from this repository. Protocol-spec findings are sourced from official primary material but scored MEDIUM per the research seam's generic webfetch-provider ceiling, not a signal about actual reliability. One self-correction already applied and carried forward (SDK maturity). |
| Pitfalls | MEDIUM | Official spec/SDK sources cross-checked across multiple independent hits. Several concrete findings (client-side caching bugs, error-code-0 regression) are primary-source GitHub issues, effectively HIGH-grade evidence despite the seam's MEDIUM ceiling. One or two supporting sources are single vendor blogs, explicitly flagged LOW inline and used only for corroboration. |

**Overall confidence:** MEDIUM-HIGH — the decision-relevant facts (SDK maturity, spec requirements, compatibility matrix, vulnerability-scan mechanics) are all HIGH-confidence and verified live. The one genuinely low-confidence area — real-world client adoption of `2026-07-28` — is honestly reported as UNKNOWN rather than guessed, which is itself the correct outcome for a five-day-old spec revision, not a research gap to close by more desk research.

### Gaps to Address

- **8-agent roster negotiation status**: UNKNOWN for all 8 clients as of this research; re-run this specific audit immediately before Phase 3's SDK decision, not trusted from this document at execution time.
- **mark3labs's own timeline**: no committed schedule exists for `2026-07-28` support; re-verify current release notes at Phase 3 execution in case this has changed.
- **`govulncheck -mode=source` + `-modfile`**: confirmed broken (reproduced failure, not guessed), with no upstream fix or workaround found in `golang/vuln`'s issue tracker — flagged as genuinely unresolved rather than worked around with an unverified guess; the binary-mode alternative sidesteps it without depending on this ever being fixed upstream.

## Sources

### Primary (HIGH confidence)
- `gh api repos/mark3labs/mcp-go/releases` and `/releases/tags/v0.57.0` — confirms current lack of `2026-07-28` support, full changelog
- `gh api repos/modelcontextprotocol/go-sdk/releases` and `/releases/tags/v1.7.0` — confirms stable, five-era-capable release, 2026-07-27
- Raw `mcp/types.go`, `server/server.go` (mark3labs @ v0.57.0) and `mcp/server.go`, `mcp/shared.go`, `examples/server/basic/main.go` (go-sdk @ v1.7.0) fetched at tag — primary source for every API-surface and constant claim
- `go list -m -json github.com/modelcontextprotocol/go-sdk@latest` and `go-sdk@v1.7.0/mcp/shared.go:45-65,394` — coordinator-verified, corrects an earlier "pre-release only" misreading
- `modelcontextprotocol.io/specification/2026-07-28/{changelog,basic,basic/transports,basic/transports/stdio,basic/versioning,server/discover,server/tools,server/utilities/caching}` — official spec, directly quoted throughout
- Live local execution in this repo: `govulncheck -mode=binary` proof-of-work, `-mode=source`+`-modfile` reproduced failure, `task vuln` no-op reproduction
- `internal/mcp/server.go`, `tools.go`, `server_test.go`, `internal/cli/serve.go`, `test/integration/mcp_stdout_purity_test.go` — direct repository code reading

### Secondary (MEDIUM confidence)
- `github.com/mark3labs/mcp-go/issues/928` — open, unclaimed SEP-2575 feature request, no maintainer timeline
- `github.com/modelcontextprotocol/go-sdk/issues/976` — real, cited error-code-0 regression, used as the concrete mutation example for the parsed-object testing trap
- `github.com/anthropics/claude-code/issues/{41123,40025,50515}`, `claude-ai-mcp#45` — confirmed, primary-source GitHub issues on client-side `tools/list` caching
- `canimcp.dev` per-client pages — dated, sourced compatibility matrix; every relevant "Core Protocol" row explicitly `Unknown`, predating the spec's finalization

### Tertiary (LOW confidence)
- `likeone.ai` real-world stdio-migration audit blog — single vendor source, used only as corroboration for "stdio was never part of the stateful-session model"
- `juburr/mad-skills` community migration guide — single source, internally detailed, cross-checked against related PRs but not independently verified against go-sdk v1.7.0 specifically
- Various WebSearch-only third-party commentary on statelessness/CIMD/auth — used only to corroborate the official changelog's own wording, never as sole support for a claim

---
*Research completed: 2026-08-03*
*Ready for roadmap: yes*
