# Phase 5: MCP Resources Capability & Claims Drift Guard - Pattern Map

**Mapped:** 2026-08-12
**Files analyzed:** 6 new + 3 modified
**Analogs found:** 9 / 9

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `internal/mcp/resources/*.md` (10 files) | config/content | request-response (static) | none (net-new content type) — nearest analog is `instructions` string constant in `internal/mcp/server.go:47` | role-match (constant → file) |
| `internal/mcp/resources.go` | service (registration) | request-response | `internal/mcp/server.go`'s `registerTools`/`allToolNames` (lines ~406-440) | exact |
| `internal/mcp/resources_test.go` | test | request-response | `internal/mcp/server_test.go` (`newTestSession`, `listToolNames`) | exact |
| `internal/mcp/resources_schema_drift_test.go` | test | CRUD/structural | `internal/mcp/tools_schema_drift_test.go` + `internal/mcp/instructions_contract_test.go` | exact |
| `internal/mcp/server.go` (modified: `Capabilities.Resources`, `registerResources` call) | config/service | request-response | itself — extend existing `Capabilities.Tools`/`registerTools` call site (lines 515-529) | exact |
| `test/wireoracle/scenarios.go` (modified: new request helpers + scenarios) | test fixture | request-response | existing `toolsListRequest`/`toolCallRequest` helpers (lines 78-101) | exact |
| `test/wireoracle/oracle_test.go` (modified or sibling test added) | test | structural/set-equality | `TestEveryRegisteredToolHasASuccessfulCallScenario` (lines 822-853) | exact |
| `testdata/wireoracle/transcripts/*.golden` (29 re-frozen + N new) | fixture | file-I/O | existing golden transcript files (e.g. `handshake-explore.golden`) | exact |

## Pattern Assignments

### `internal/mcp/resources.go` (service, request-response)

**Analog:** `internal/mcp/server.go` — `registerTools`/`allToolNames` (lines ~395-441)

**Imports pattern** (mirror `server.go:17-32`):
```go
import (
	"context"
	"embed"
	"fmt"
	"io/fs"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)
```

**Core registration-loop pattern** (mirror `registerTools`, `server.go:~426-441` — single seam, one source of truth, iteration order from an explicit ordered structure, not a map range):
```go
func registerTools(s *mcp.Server, companions map[string]bool, repoPath, startPath string, detector *gitmeta.CachingDetector) int {
	count := 0
	mcp.AddTool(s, exploreTool(), exploreHandler(repoPath, startPath, detector))
	count++
	for _, name := range companionNames {
		if companions[name] {
			companionHandler(s, name, repoPath, startPath, detector)
			count++
		}
	}
	return count
}
```
Apply the same shape to `registerResources(s)`: one function, one seam, called unconditionally from `BuildServer`, deriving both the registered set and any drift-guard's expected set from the SAME map (`resourceURIFor`) — never a second hand-typed list (this is the anti-pattern the RESEARCH.md explicitly calls out, matching this file's own established "SURF-01" drift lesson in `tools_schema_drift_test.go`'s doc comment).

**Error-handling pattern** (`registerTools`'s sibling `unregisterTools`, `server.go:443-446`, uses a no-op-safe removal call — no error path needed at registration time in this codebase's convention). For `resources.go`, mirror the `panic`-on-programmer-error convention RESEARCH.md's Pattern 2 code example already demonstrates (embedded-FS read failure or missing map entry is a build-time programmer error, not a runtime condition to recover from) — consistent with this file's existing `BuildServer` doc comment treating a similar class of internal invariant violation as fatal-at-construction rather than propagated.

---

### `internal/mcp/server.go` (modified — `Capabilities.Resources` + unconditional call)

**Analog:** itself, `D-11` comment block, lines 515-529

**Exact insertion pattern** (verified current code, `server.go:515-529`):
```go
// D-11: Capabilities must be set explicitly and unconditionally.
// Server.capabilities() only sets caps.Tools when HasTools ||
// tools.len() > 0 — without this, the "tools" key silently vanishes
// from the initialize response's capabilities object on the
// hasIndex=false path (MCP-03) ...
s := mcp.NewServer(&mcp.Implementation{Name: "codegraph", Version: version}, &mcp.ServerOptions{
	Capabilities: &mcp.ServerCapabilities{
		Tools: &mcp.ToolCapabilities{ListChanged: true},
		// ADD: Resources: &mcp.ResourceCapabilities{},
	},
	Instructions: instructions,
})
// registerResources(s) — added here, unconditionally, NOT inside
// `if hasIndex {`, mirroring D-11's own rationale for Capabilities.Tools.
```

**Doc-comment convention to copy:** every non-obvious structural decision in this file is explained inline with a `D-NN`/`RSRC-NN` tag referencing the deciding doc (see `D-11`, `D-13` above it) — new code should tag its own rationale (`RSRC-03`) the same way, not leave it implicit.

---

### `internal/mcp/resources/*.md` (10 fact-sheet/behavior-doc files)

**Analog:** the `instructions` constant, `server.go:38-51` — closest existing precedent for "static, source-derived, wire-facing prose content with an explicit no-interpolation/no-repo-path constraint."

**Content constraint pattern to copy** (from `instructions`'s own doc comment, `server.go:38-51`):
```go
// ... it MUST stay a compile-time literal with no interpolation
// of any kind — no repository path, no resolved index root, no hostname,
// no environment value ...
```
Apply this same "no interpolation, static at build time" constraint to every resource markdown file's content — matches D-01 through D-04's fact-sheet-only scope and the Security Domain section's explicit reuse of this exact constraint for resource content.

**Source-of-truth to restate as markdown:** `internal/mcp/tools.go`'s `Description` fields and jsonschema-tagged `*Args` structs (lines 161-544) are the one and only source for the 8 per-tool fact-sheets — do not hand-author facts not traceable to `tools.go` or `server.go`'s `companionNames`/env-var resolution.

---

### `internal/mcp/resources_test.go` (test, request-response)

**Analog:** `internal/mcp/server_test.go` — `newTestSession` (lines 74-93) and `listToolNames` (lines 106-123)

**Core pattern to copy verbatim in shape** (`server_test.go:74-93`):
```go
func newTestSession(t *testing.T, s *mcp.Server) (*mcp.ClientSession, func()) {
	t.Helper()
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	ctx := context.Background()
	go func() {
		_ = s.Run(ctx, serverTransport)
	}()
	client := mcp.NewClient(&mcp.Implementation{Name: "codegraph-mcp-test", Version: "0.0.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client Connect: %v", err)
	}
	return session, func() { _ = session.Close() }
}
```
New `resources_test.go` reuses this exact `newTestSession` helper (same package, no duplication) and adds a `listResourceURIs`-shaped sibling to `listToolNames` (`server_test.go:106-123`):
```go
func listToolNames(t *testing.T, s *mcp.Server) []string {
	t.Helper()
	session, cleanup := newTestSession(t, s)
	defer cleanup()
	result, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	names := make([]string, len(result.Tools))
	for i, tool := range result.Tools {
		names[i] = tool.Name
	}
	sort.Strings(names)
	return names
}
```
Replace `session.ListTools`/`result.Tools` with `session.ListResources`/`result.Resources` (RESEARCH.md confirms identical shape at `client.go:1309-1345`); for read coverage use `session.ReadResource(ctx, &mcp.ReadResourceParams{URI: uri})` per URI and assert non-empty `Text` + `MIMEType == "text/markdown"`.

---

### `internal/mcp/resources_schema_drift_test.go` (test, structural/set-equality — GUARD-01/02)

**Analog A (GUARD-01, numeric claim pinning):** `internal/mcp/tools_schema_drift_test.go` lines 16-32

```go
var numericClaimRe = regexp.MustCompile(`(?i)\b(default|max)\s+(\d+)`)

var engineConstantFor = map[string]string{
	"codegraph_impact.depth.default":      "defaultDepth",
	"codegraph_impact.depth.max":          "MaxDepth",
	"codegraph_explore.max_files.default": "defaultMaxFiles",
}
```
Extend this exact map/regex pair to scan `resourcesFS` file contents in addition to tool schema descriptions — same regex, same map (RESEARCH.md Pattern 3 recommends ONE shared map over a sibling map, as the stronger drift-proof form).

**Analog B (GUARD-02, structural set-equality):** `test/wireoracle/oracle_test.go` — `TestEveryRegisteredToolHasASuccessfulCallScenario` (lines 822-853)

```go
func TestEveryRegisteredToolHasASuccessfulCallScenario(t *testing.T) {
	registered := toolNamesFromCapture(t, "toolslist-repeat", 2)
	if len(registered) != 8 {
		t.Fatalf("toolslist-repeat: registered %d tools, want 8: %v", len(registered), registered)
	}
	remaining := make(map[string]bool, len(registered))
	for _, name := range registered {
		remaining[name] = true
	}
	for _, sc := range Scenarios() {
		reqID, toolName, ok := findToolCallRequest(sc)
		if !ok || !remaining[toolName] {
			continue
		}
		tr, _ := mustCaptureScenario(t, sc.Name)
		if isSuccessfulToolCall(tr.Stdout, reqID) {
			delete(remaining, toolName)
		}
	}
	if len(remaining) > 0 {
		names := make([]string, 0, len(remaining))
		for name := range remaining {
			names = append(names, name)
		}
		sort.Strings(names)
		t.Fatalf("no scenario provides a successful tools/call for: %v", names)
	}
}
```
In-package analog for GUARD-02: derive expected set from `companionNames` + literals `"explore"`, `"tools-filter"`, `"index-state"`; derive actual set from `fs.ReadDir(resourcesFS, "resources")`; assert set equality both directions (missing file AND orphaned file) — bidirectional, unlike the tools test's one-directional "coverage" check.

**Mutation-proof discipline to copy:** `test/wireoracle/MUTATION-PROOF.md` lines 93-172 ("Mutation 2 — a dropped tool") — apply the identical demonstrated-red-then-revert protocol: rename a `.md` file without updating `companionNames`/`resourceURIFor`, run `go test ./internal/mcp/...`, capture the failure text, revert byte-clean, append to `MUTATION-PROOF.md`.

---

### `test/wireoracle/scenarios.go` (modified — new request helpers)

**Analog:** `toolsListRequest` and `toolCallRequest` (lines 78-101)

```go
func toolsListRequest(id int) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "tools/list",
	}
}

func toolCallRequest(id int, name string, arguments any) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      name,
			"arguments": arguments,
		},
	}
}
```
New helpers `resourcesListRequest(id int)` and `resourceReadRequest(id int, uri string)` follow this exact literal-map shape (RESEARCH.md Code Examples section already gives the concrete bodies, verbatim-derivable from this pattern). Register new `Scenario` entries the same way existing `call-*` scenarios are declared elsewhere in this file; bump `ExpectedScenarioCount` in lockstep (existing convention — see `TestScenarioCountIsExact` in `oracle_test.go`).

**Anti-pattern to avoid (from this file's own documented invariant, lines ~440-456):** at most one async-dispatched request (`resources/read`, treated conservatively the same as `tools/call`) per scenario, and it must be last — do not batch multiple resource reads into one scenario.

---

## Shared Patterns

### Unconditional capability + registration (RSRC-03)
**Source:** `internal/mcp/server.go` D-11 comment + `Capabilities.Tools` field (lines 515-529)
**Apply to:** `server.go`'s `Capabilities.Resources` field and the `registerResources(s)` call — both must sit outside the `if hasIndex` branch, exactly mirroring why `Capabilities.Tools` is unconditional today.

### Source-derived, never hand-duplicated facts (GUARD-01/GUARD-02, "SURF-01" lesson)
**Source:** `internal/mcp/tools_schema_drift_test.go`'s doc comment (lines 34-40) recounting the exact prior incident (`defaultDepth` 5→2 drift) this pattern exists to prevent.
**Apply to:** every new file — `resourceURIFor` map is the ONE place the URI-to-file mapping lives; the drift test reads from the SAME map, never retypes it; the tool-count/name set is read from `companionNames`, never a second hardcoded list.

### `D-NN`/`RSRC-NN`/`GUARD-NN` inline doc-comment tagging convention
**Source:** pervasive throughout `server.go` (D-05, D-08, D-11, D-13) and `tools_schema_drift_test.go` (SURF-01)
**Apply to:** all new files — every non-obvious structural choice (e.g., "why is `Capabilities.Resources` explicit not implicit," "why is the URI a map not a convention") gets an inline comment tagged with its originating decision ID from CONTEXT.md/RESEARCH.md.

### In-memory session test scaffolding
**Source:** `internal/mcp/server_test.go`'s `newTestSession`/`copyFixture`/`indexFixture` helpers
**Apply to:** `resources_test.go` — reuse `newTestSession` directly (same package); do not duplicate the in-memory transport setup.

### Wire-oracle reviewed-diff discipline
**Source:** `test/wireoracle/COVERAGE-BASELINE.md`, `MUTATION-PROOF.md`, `TestFrozenTranscriptsMatch`
**Apply to:** the mandatory re-capture of all 29 existing `.golden` transcripts (capabilities.resources changes every `initialize`/`discover` result — RESEARCH.md Pitfall 2) plus N new scenario transcripts; every changed line must be attributed to a named cause in the reviewed-diff pass.

## No Analog Found

None — every file in scope has at least a role-match analog already in the codebase. The one genuinely novel element (10 embedded markdown content files) has no direct prior analog as a *file type* in this repo (first `go:embed` use), but its *content-constraint* pattern (static, non-interpolated, source-derived prose) is directly modeled on the existing `instructions` string constant.

## Metadata

**Analog search scope:** `internal/mcp/`, `test/wireoracle/`, `testdata/wireoracle/transcripts/`
**Files scanned:** `server.go`, `tools.go`, `tools_schema_drift_test.go`, `instructions_contract_test.go`, `server_test.go`, `test/wireoracle/scenarios.go`, `test/wireoracle/oracle_test.go`, `test/wireoracle/MUTATION-PROOF.md`, `test/wireoracle/COVERAGE-BASELINE.md`
**Pattern extraction date:** 2026-08-12
