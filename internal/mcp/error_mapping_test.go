package mcp

// error_mapping_test.go pins SDK-04: the error-to-wire mapping for every
// failure class internal/mcp's tool handlers can produce.
//
// Under mark3labs/mcp-go, a handler's bare `return nil, err` produced a
// top-level JSON-RPC protocol error with code -32603 (measured,
// test/wireoracle/MUTATION-PROOF.md § Mutation 3, run against the real
// pre-migration binary). Under modelcontextprotocol/go-sdk, the exact same
// Go statement — `return nil, nil, err` — is converted by the SDK's own
// toolForErr wrapper into a tool-visible `{"content":[...],"isError":true}`
// result: a SUCCESSFUL JSON-RPC response, never a protocol error. The Go
// type signature is `error` either way, so nothing in a compile or a code
// review surfaces the difference (02-RESEARCH.md Q5). None of the 23 frozen
// wire-oracle scenarios (test/wireoracle/COVERAGE-BASELINE.md) exercises a
// handler's own argument-validation failure — that gap is why this
// assertion lives here, in internal/mcp, rather than as a 24th frozen
// scenario.
//
// Every test below checks THREE things, not one: that CallTool returned a
// nil Go error at the session level (proving the response was a successful
// JSON-RPC result rather than a protocol error — go-sdk surfaces a protocol
// error as a non-nil error from CallTool); that result.IsError is true; and
// that result.Content has exactly one element which type-asserts to
// *mcp.TextContent carrying the expected substring. Checking only IsError
// would stay green under exactly the mark3labs-versus-go-sdk difference
// this file exists to catch (PITFALLS.md Testing Trap B).

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// callToolRaw connects a fresh in-memory session to s and issues a
// tools/call for name with args, returning both the CallToolResult and any
// Go error CallTool itself returned. Unlike markdown_test.go's callTool
// (which t.Fatalf's on a non-nil err), every test in this file inspects the
// session-level error itself — SDK-04's whole point is that a
// handler-returned Go error must NOT surface as a non-nil err here.
func callToolRaw(t *testing.T, s *mcp.Server, name string, args map[string]any) (*mcp.CallToolResult, error) {
	t.Helper()

	session, cleanup := newTestSession(t, s)
	defer cleanup()

	return session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
}

// TestHandlerErrorIsToolResultNotProtocolError pins the handler-returned
// error class: codegraph_status's openEngine error path
// (confineToRepoRoot's rejection of a path outside the server's configured
// repo root) reaches the caller as a tool-visible result, not a protocol
// error.
func TestHandlerErrorIsToolResultNotProtocolError(t *testing.T) {
	dir := copyFixture(t)
	indexFixture(t, dir)
	outside := copyFixture(t)
	indexFixture(t, outside)

	s := BuildServer(true, map[string]bool{"status": true}, dir, dir)

	result, err := callToolRaw(t, s, "codegraph_status", map[string]any{"path": outside})
	if err != nil {
		t.Fatalf("CallTool codegraph_status returned a session-level (protocol) error: %v — want a successful JSON-RPC result with isError:true", err)
	}
	if !result.IsError {
		t.Fatalf("codegraph_status with a path outside the repo root: expected IsError true, got false: %+v", result)
	}
	if len(result.Content) != 1 {
		t.Fatalf("codegraph_status error result.Content has %d elements, want exactly 1: %+v", len(result.Content), result.Content)
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("codegraph_status error result.Content[0] is %T, want *mcp.TextContent", result.Content[0])
	}
	if !strings.Contains(text.Text, "outside") {
		t.Fatalf("codegraph_status error text = %q, want it to contain %q", text.Text, "outside")
	}
}

// TestMissingRequiredArgumentIsToolVisibleError pins the schema-rejection
// class: a codegraph_explore call omitting the required "query" argument is
// rejected by applySchema BEFORE exploreHandler's body ever runs, and that
// rejection is also a tool-visible isError:true result rather than a
// protocol error. The rejection text is the SDK validator's own wording —
// asserted to differ from the deleted handler-authored wording
// (`required argument "query" not found`) so the divergence is pinned as
// observed, not assumed absent.
func TestMissingRequiredArgumentIsToolVisibleError(t *testing.T) {
	dir := copyFixture(t)
	indexFixture(t, dir)

	s := BuildServer(true, map[string]bool{}, dir, dir)

	result, err := callToolRaw(t, s, "codegraph_explore", map[string]any{})
	if err != nil {
		t.Fatalf("CallTool codegraph_explore (missing query) returned a session-level (protocol) error: %v — want a successful JSON-RPC result with isError:true", err)
	}
	if !result.IsError {
		t.Fatalf("codegraph_explore with query omitted: expected IsError true, got false: %+v", result)
	}
	if len(result.Content) != 1 {
		t.Fatalf("codegraph_explore missing-argument result.Content has %d elements, want exactly 1: %+v", len(result.Content), result.Content)
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("codegraph_explore missing-argument result.Content[0] is %T, want *mcp.TextContent", result.Content[0])
	}
	if !strings.Contains(text.Text, "query") {
		t.Fatalf("codegraph_explore missing-argument text = %q, want it to name %q", text.Text, "query")
	}
	const preMigrationText = `required argument "query" not found`
	if text.Text == preMigrationText {
		t.Fatalf("codegraph_explore missing-argument text is still the pre-migration handler-authored wording %q — expected the SDK validator's own wording", preMigrationText)
	}
}

// TestUnknownArgumentIsRejected pins D-10: an argument no tool schema
// declares is rejected (additionalProperties:false), not silently ignored.
func TestUnknownArgumentIsRejected(t *testing.T) {
	dir := copyFixture(t)
	indexFixture(t, dir)

	s := BuildServer(true, map[string]bool{}, dir, dir)

	result, err := callToolRaw(t, s, "codegraph_explore", map[string]any{
		"query":              "main",
		"bogus_unknown_argx": "x",
	})
	if err != nil {
		t.Fatalf("CallTool codegraph_explore (unknown argument) returned a session-level (protocol) error: %v — want a successful JSON-RPC result with isError:true", err)
	}
	if !result.IsError {
		t.Fatalf("codegraph_explore with an undeclared argument: expected IsError true (additionalProperties:false, D-10), got false: %+v", result)
	}
	if len(result.Content) != 1 {
		t.Fatalf("codegraph_explore unknown-argument result.Content has %d elements, want exactly 1: %+v", len(result.Content), result.Content)
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("codegraph_explore unknown-argument result.Content[0] is %T, want *mcp.TextContent", result.Content[0])
	}
	if text.Text == "" {
		t.Fatal("codegraph_explore unknown-argument error text is empty")
	}
}

// TestEngineErrorIsToolResult pins the engine-level failure class:
// Engine.Node's "symbol not found" error, returned as a bare Go error from
// companionHandler's "node" branch, arrives as a tool result, never a
// protocol error.
func TestEngineErrorIsToolResult(t *testing.T) {
	dir := copyFixture(t)
	indexFixture(t, dir)

	s := BuildServer(true, map[string]bool{"node": true}, dir, dir)

	result, err := callToolRaw(t, s, "codegraph_node", map[string]any{"symbol": "ThisSymbolDoesNotExistAnywhereInTheFixture"})
	if err != nil {
		t.Fatalf("CallTool codegraph_node (nonexistent symbol) returned a session-level (protocol) error: %v — want a successful JSON-RPC result with isError:true", err)
	}
	if !result.IsError {
		t.Fatalf("codegraph_node for a nonexistent symbol: expected IsError true, got false: %+v", result)
	}
	if len(result.Content) != 1 {
		t.Fatalf("codegraph_node error result.Content has %d elements, want exactly 1: %+v", len(result.Content), result.Content)
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("codegraph_node error result.Content[0] is %T, want *mcp.TextContent", result.Content[0])
	}
	if !strings.Contains(text.Text, "not found") {
		t.Fatalf("codegraph_node error text = %q, want it to contain %q", text.Text, "not found")
	}
}
