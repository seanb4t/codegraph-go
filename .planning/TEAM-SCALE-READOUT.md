# Team Scale strategic read-out (backlog 999.6, D-12)

**Dated:** 2026-08-05

**Status:** This is a strategic decision record for a future, unscoped
milestone. **v0.3.0 records this read-out and builds none of it.** No task
in this milestone creates a central server, a session store, or any
multi-tenant code path. This file exists so the analysis is not lost
between now and whenever Team Scale is actually planned, and it is kept in
`.planning/` rather than `docs/` deliberately (D-12): it is input to a
future planning cycle, not shipped user documentation.

## The finding

The `2026-07-28` stateless core (SEP-2567 removing protocol-level sessions
and the `Mcp-Session-Id` header; SEP-2575 removing the `initialize`
handshake in favor of per-request `_meta`) removes the sticky-routing and
shared-session-store infrastructure a future central codegraph MCP server
would otherwise have needed to build. This is not a coincidence the spec
handed us — it is because two decisions this codebase already made in
v1.0, for entirely local, single-process reasons, happen to already match
the shape `2026-07-28` recommends for stateless cross-call state:

1. **`path` is already an explicit per-call tool argument, never implicit
   session state.** Every one of the 8 MCP tools takes the repository path
   it operates on as a tool argument, not as something bound once at
   connection time and implicitly reused. A stateless server that must
   resolve identity fresh on every request loses nothing here — this
   codebase never relied on the session to carry that identity in the
   first place.
2. **Every handler already opens a fresh `query.Engine` snapshot per
   call.** There is no long-lived, connection-scoped engine instance whose
   staleness a central server would need to reason about across requests
   from different callers; each call already gets its own consistent view.

Both were ordinary v1.0 engineering choices made for a single local
process talking to a single local index. Neither was designed with
`2026-07-28`'s stateless-core requirement in mind — the alignment is
retrospective, not planned — but it means the sticky-routing and
shared-session-store layer a naive remote-server design would need is
largely unnecessary work a future Team Scale milestone gets to skip.

## The one real structural gap

`BuildServer`'s four parameters — `hasIndex bool`, `allowlist
map[string]bool`, `repoPath string`, `startPath string`
[VERIFIED: `internal/mcp/server.go:94`] — are **constructor-time-only**.
They are resolved once when the server process starts and baked into the
closures the tool handlers capture. That is correct and sufficient for a
single-repo, single-process stdio server: there is exactly one `repoPath`
for the whole process's lifetime, so binding it once at startup is not a
limitation, it is the natural shape of the problem this server currently
solves.

A multi-tenant central server serving multiple repos/callers over one
process would need these four values to become **per-request-resolved**
instead of constructor-time-resolved — each incoming call would need to
determine its own `hasIndex`/`allowlist`/`repoPath`/`startPath` from
request identity, rather than there being one fixed answer baked in at
process start. This is a **bounded, already-anticipated refactor** — it
does not require a different storage engine, a different protocol
approach, or a rewrite of the 8 tool handlers' logic, only a change to how
their captured configuration is threaded through. It is not, however,
free: every one of the 8 handlers closes over these values today, so the
refactor touches every handler's construction path, not just `BuildServer`
itself.

## What this milestone does and does not do about it

v0.3.0 does not touch `BuildServer`'s signature, does not add multi-tenant
resolution, and does not build any central-server infrastructure. This
read-out exists purely so that when Team Scale is eventually scoped as its
own milestone, the planning conversation starts from "here is the one
known gap and roughly how bounded it is" instead of re-deriving this
analysis from scratch.

## Sources

- `.planning/phases/01-protocol-scoping-the-sdk-independent-wire-oracle/01-RESEARCH.md` § "Team Scale Strategic Read-Out" — this file transcribes that section's already-derived conclusion per D-12's instruction, rather than re-deriving it
- `.planning/research/ARCHITECTURE.md` § Q1 — the original analysis this read-out is sourced from
- `internal/mcp/server.go:94` [VERIFIED] — `BuildServer`'s four constructor-time parameters
