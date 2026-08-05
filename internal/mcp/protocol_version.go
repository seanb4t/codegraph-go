package mcp

// ProtocolVersion is the MCP protocol revision this server declares.
//
// This is the revision today's server actually negotiates. It is owned by
// this repository rather than read from an SDK, so a dependency bump can
// never move it silently (VRFY-02).
//
// Mechanism honesty (Phase 1): this literal is an asserted compatibility
// pin, not the source of the negotiated value. mark3labs/mcp-go v0.56.0
// decides the negotiated revision inside the unexported
// (*MCPServer).protocolVersion method and exposes no
// WithProtocolVersion-style server option, so no caller can inject a
// repo-owned literal while that SDK is the backend. What this constant buys
// today is the alarm: it currently equals the SDK's LATEST_PROTOCOL_VERSION,
// so the moment a dependency bump moves the SDK's value, the wire oracle's
// spec anchor (test/wireoracle) turns red — which is exactly the "a
// dependency bump must never move wire behavior silently" property
// REQUIREMENTS.md VRFY-02 asks for ("asserted against a repo-owned
// literal"). The stricter ROADMAP phrasing ("reads from") lands in Phase 2,
// when the SDK swap supplies a backend whose protocol revision a caller can
// actually supply — do not mistake this Phase 1 pin for an injection point.
//
// This file deliberately references no SDK identifier.
const ProtocolVersion = "2025-11-25"
