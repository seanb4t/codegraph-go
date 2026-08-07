package mcp

// ProtocolVersion is the MCP protocol revision this server declares.
//
// This is the revision today's server actually negotiates. It is owned by
// this repository rather than read from an SDK, so a dependency bump can
// never move it silently (VRFY-02).
//
// Mechanism honesty (Phase 2, corrected from Phase 1's prediction): this
// literal is an asserted compatibility pin, not the source of the
// negotiated value, and that remains true under
// github.com/modelcontextprotocol/go-sdk v1.7.0 too. The negotiated
// revision is decided by the unexported package-level negotiatedVersion
// function in the SDK's own mcp package (mcp/shared.go), whose version-
// value constants (latestProtocolVersion, protocolVersion20260728, etc.)
// are all unexported and therefore structurally unreferenceable from
// internal/mcp — Go's own visibility rules make it impossible, not merely
// discouraged. ServerOptions exposes no ProtocolVersion field among its 17
// fields (it does expose Instructions and Capabilities); the sole reachable
// mechanism, AddReceivingMiddleware, only ever READS the SDK's own already-
// computed *InitializeResult after negotiation has happened (see
// server.go's session-line middleware) — it is not a setter. What this
// constant buys is the same alarm Phase 1 built: it currently equals the
// value go-sdk negotiates for this literal, so the moment a dependency bump
// moves that value, the wire oracle's spec anchor (test/wireoracle) turns
// red — the "a dependency bump must never move wire behavior silently"
// property REQUIREMENTS.md VRFY-02 asks for ("asserted against a repo-owned
// literal"). VRFY-02's stricter "reads from" phrasing is confirmed
// undeliverable against this SDK too (D-04, D-06) — not deferred again, but
// proven false by exhaustive source enumeration (02-RESEARCH.md Q1) rather
// than assumed. Do not mistake this pin for an injection point.
//
// This file deliberately references no SDK identifier.
const ProtocolVersion = "2025-11-25"
