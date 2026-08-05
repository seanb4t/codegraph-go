No external API integration: this phase builds an SDK-independent wire-verification harness, a repo-owned protocol-version literal, an internal package seam, and two scoping documents — `go.mod` is deliberately unchanged (VRFY-04 pins `github.com/mark3labs/mcp-go v0.56.0` through the end of the phase, and RESEARCH.md records the Package Legitimacy Audit as not applicable because no new external package is introduced).

The MCP capability surface this milestone integrates against is decided in Phase 2 (SDK adoption)
and Phase 3 (`2026-07-28` spec compliance); the SEP-by-SEP applicability table produced by this
phase at `docs/MCP-2026-07-28-SCOPING.md` is the input to that decision, not the decision itself.
