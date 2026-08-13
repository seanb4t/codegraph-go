# Phase 5 — API Coverage Decision

**Decision:** opt out. No external API integration in this phase.

No external API integration: this phase adds MCP resource-serving capability using
`Server.AddResource` on the already-pinned `modelcontextprotocol/go-sdk@v1.7.0` — no new
external API, SDK, or service is integrated.

## Why the detector fired, and why it is a false positive

The deterministic detector returned `detected: true` on two signals, both re-verified here:

| Signal | Actual referent | New external API? |
|---|---|---|
| `wire` | The MCP **wire protocol** / the in-repo `test/wireoracle` harness terminology | No |
| `SDK` | `github.com/modelcontextprotocol/go-sdk@v1.7.0`, already in `go.mod` and already the live MCP backend since Phase 2 (`SDK-01`) | No |

`Server.AddResource` / `AddResourceTemplate` / `mcp.Resource` / `mcp.ResourceCapabilities` are
existing methods and types on that already-present, already-pinned dependency — verified by
reading the module source at `$(go env GOMODCACHE)/github.com/modelcontextprotocol/go-sdk@v1.7.0/mcp/server.go:575-607`
and `.../mcp/protocol.go:1262-1272, 1595-1607, 2248-2255`.

`go.mod` gains no `require` line in this phase. Everything else this phase adds is
`go:embed`'d markdown, Go test code, and frozen JSON-RPC transcripts.

No capability coverage matrix is fabricated for an API this phase does not integrate.
