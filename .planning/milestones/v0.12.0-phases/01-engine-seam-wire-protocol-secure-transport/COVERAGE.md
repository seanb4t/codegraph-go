# Phase 1 — API Coverage Decision

**Decision:** opt out. No external API integration in this phase.

No external API integration: this phase serves codegraph's own first-party `uiv1.UIService` over a
loopback-only listener using `connectrpc.com/connect` as a server-side transport library — no
external API, SDK or service is consumed, and no capability surface belonging to a third party is
integrated.

## Why the detector fired, and why it is a false positive

The deterministic detector returned `detected: true` on one signal, re-verified here:

| Signal | Actual referent | New external API? |
|---|---|---|
| `wire` + `mcp` | One sentence in `01-09-PLAN.md:544` — "…answer over the **wire** through the same builders CLI and **MCP** use…" — describing the *first-party* `internal/query.Engine` read seam shared by the CLI, the MCP server and now the UI | No |

The clause names two in-repo consumers of an in-repo seam. There is no third-party service in it.

The signal is absent from the phase's ROADMAP section and appears only in a plan body, so the
`plan:pre` producer could not have seen it: run over the roadmap section alone the detector returns
`detected: false` (exit 1); run over the roadmap section plus the written `*-PLAN.md` bodies — the
scope the seal-time gate scans — it returns `detected: true` (exit 0) on this snippet. That
asymmetry, not an authoring omission, is why no `COVERAGE.md` was produced at plan time.

## The one new dependency, and why it is not an integration

Unlike the analogous v0.10.0 Phase 5 declaration, this phase **does** add a `require` line:
`connectrpc.com/connect v1.20.0`, introduced at `e09daba` (plan 01-01) with the
buf/protoc-gen-connect-go codegen pipeline.

It is a transport library, not a service client:

- **Server-side only.** The sole first-party use is `uiv1connect.NewUIServiceHandler` at
  `internal/uiserver/server.go:92`. No `connect.NewClient` / `NewUIServiceClient` construction
  exists outside generated code and tests.
- **No outbound calls.** `internal/uiserver` contains no `http.Client`, `http.Get`/`http.Post`, and
  no external host literal.
- **Loopback only.** `uiserver.DefaultAddr` is `127.0.0.1:0` (`internal/uiserver/server.go:22`),
  paired with the Origin/Host allowlist in `internal/uiserver/originguard.go`.
- **The served surface is ours.** The nine `UIService` methods are declared in the repo's own
  `internal/uiproto/uiv1/ui.proto`, not discovered from a vendor.

A capability matrix enumerating those nine methods would be enumerating codegraph's own API, which
is not what this artifact records, and is the fabricated-row case the checkpoint explicitly forbids.

## On MCP

`internal/mcp` source **was** modified this phase (`server.go`, `session_line.go`, and a new
`pending_writer_test.go` — the FIX-01 `pendingWriter` accounting fix). The phase's "every existing
CLI and MCP byte unchanged" claim is about *output* bytes, and it holds: the frozen golden suite
runs clean at 61 `--- PASS` / 0 `--- FAIL`, including `TestGoldensMatchLiveEngineOutputIsNonVacuous`
(proving the comparison actually executed) and the `TestGoldenScenarioCountIsExact` /
`TestReFrozenGoldensValid` pair that pins enumeration to both the constant and the filesystem.

No capability coverage matrix is fabricated for an API this phase does not integrate.
