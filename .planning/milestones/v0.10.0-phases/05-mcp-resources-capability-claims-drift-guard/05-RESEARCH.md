# Phase 5: MCP Resources Capability & Claims Drift Guard - Research

**Researched:** 2026-08-12
**Domain:** `modelcontextprotocol/go-sdk@v1.7.0` MCP Resources API, embedded-content drift guards, wire-oracle re-capture
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Resource content depth**
- **D-01:** Each of the 8 per-tool resource docs is a fact-sheet only — the same params/defaults/return-shape facts already carried in the tool's jsonschema tags (`tools.go`), reformatted as markdown prose. No worked examples, no "use this instead of X" cross-references — those risk duplicating what Phase 6's SKILL.md decision table will state, and each additional claim (an example's expected output, a cross-tool recommendation) is a new fact GUARD-01 has to keep pinned.
- **D-02:** The "empty until indexed" caveat is NOT repeated inside each per-tool doc — it lives solely in the separate index-state-preconditions resource (D-04). Keeps each tool doc mechanical and avoids hand-typing the same caveat 8 times.
- **D-03:** No hard byte/line budget is enforced on resource docs (unlike the `instructions` wire-string's 600-byte cap). That cap exists because `instructions` is JSON-encoded into every wire-oracle transcript; these are separate `resources/read` responses with no per-transcript inflation cost. Fact-sheet-only content (D-01) stays short by discipline; no new guard is needed for this.
- **D-04:** The fact-sheets do NOT restate that all 8 tools are read-only/safe. `ToolAnnotations` (in `tools.go`) already communicates this via the protocol's own annotation mechanism, readable directly from `tools/list` — restating it in resource prose would be a second copy of a fact GUARD-01 would have to keep in sync with `tools.go` for no client-facing benefit.

**Resource split — filter semantics vs. index-state preconditions**
- **D-05:** `CODEGRAPH_MCP_TOOLS` narrowing-filter semantics and index-state preconditions are two SEPARATE resources, not one combined "server behavior" doc — matches the roadmap's own wording (success criterion 1 names them as distinct items) and keeps each resource's drift-guard surface tied to one source of truth (env-var resolution vs. `hasIndex` gating are different code paths in `server.go`).
- **D-06 — user-supplied fact, load-bearing for scope:** `CODEGRAPH_MCP_TOOLS` is planned for removal in a near-future release (not scheduled within this milestone). Phase 5 documents the filter's CURRENT, real behavior with no special "this may go away" caveat — if/when it's actually removed, GUARD-02 catches the resource going stale at that point; hedging about a future removal now would be documenting a decision that hasn't been made yet.
- **D-07:** The 2 behavior docs (filter semantics, index-state preconditions) live in the same directory/`go:embed` convention as the 8 tool docs — one embedded filesystem, one registration loop, no special-casing.

**URI naming scheme**
- **D-08:** Resources use a custom `codegraph://` scheme rather than plain descriptive strings (e.g. `docs/codegraph_explore`) — visually distinct from file/http(s) URIs at a glance, consistent with how other MCP servers expose non-file resources.
- **D-09:** Per-tool URIs are `codegraph://tools/<name>` where `<name>` is the short companion/CLI name (`explore`, `node`, `search`, `callers`, `callees`, `impact`, `files`, `status` — matching `companionNames` in `server.go`), not the full registered tool name (`codegraph_explore`). The `codegraph://` prefix already establishes context; repeating `codegraph_` in every segment is redundant.
- **D-10:** The 2 behavior-doc URIs are `codegraph://tools-filter` and `codegraph://index-state` — deliberately NOT `codegraph://env/CODEGRAPH_MCP_TOOLS`, since that would tie the URI to an env var name that D-06 establishes is going away, forcing a URI rename alongside its eventual removal. `codegraph://tools-filter` names what the doc is about, not the mechanism's current implementation detail.

### Claude's Discretion
- Exact markdown structure/headers within each fact-sheet (beyond "fact-sheet only, no examples, no cross-tool recommendations" from D-01).
- How GUARD-01's existing `engineConstantFor` regex+map pattern (`tools_schema_drift_test.go`) extends to cover claims made in the embedded resource markdown, versus a different pinning mechanism for prose content — this is implementation approach, not a vision decision.
- Wire-oracle scenario additions/re-capture mechanics for `resources/list`/`resources/read` (success criterion 5) — follows the existing `test/wireoracle` reviewed-diff discipline (`COVERAGE-BASELINE.md`, `MUTATION-PROOF.md`); planner/researcher own the specifics.

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope. `CODEGRAPH_MCP_TOOLS` removal itself is not deferred as a phase-backlog item here (it belongs to whatever future milestone actually removes it); it was surfaced only as context shaping this phase's decisions.

### Phase Boundary (from CONTEXT.md)
The server documents itself over the wire: `resources/list` + `resources/read` serve tool-by-tool reference content in any repository, indexed or not, with every stated fact derived from source or pinned by a test that can fire. This phase covers the MCP Resources capability itself (RSRC-01/02/03) and the drift guard that keeps its content honest (GUARD-01/02), plus the wire-oracle re-capture that `capabilities.resources` appearing in `initialize` forces. It does NOT cover SKILL.md authoring (Phase 6), the SessionStart nudge (Phase 6), install/uninstall distribution (Phase 7), or the `instructions` wire-string rewrite (Phase 8) — those name these resource URIs but resolve them in later phases.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-------------------|
| RSRC-01 | Agent client can call `resources/list` and see one resource for each of the 8 tools plus `CODEGRAPH_MCP_TOOLS` semantics and index-state preconditions | Pattern 1 (unconditional capability), Pattern 2 (registration loop) — 10 `AddResource` calls unconditionally in `BuildServer`; wire-level proof via new `test/wireoracle` scenarios (Open Question 3 scopes exact count) |
| RSRC-02 | Agent client can call `resources/read` on any listed resource URI and receive the full reference doc as `text/markdown` | Pattern 2 (`MIMEType: "text/markdown"` on `Resource`, auto-propagated to `ResourceContents`); Code Examples §2-3 |
| RSRC-03 | Resources register unconditionally at server startup — available even when zero index-gated tools are visible | Pattern 1 — registration must sit outside `if hasIndex`, mirroring D-11's existing `Capabilities.Tools` rationale; verified `recheckCatalog`'s method switch never touches resources methods, so no interaction with the dynamic tool-catalog mechanism |
| GUARD-01 | Every tool count, default value, and env var name stated in resources/skill/instructions is derived from source constants or checked by an automated test | Pattern 3 — extend `numericClaimRe`/`engineConstantFor` to scan embedded resource markdown; Common Pitfalls (source-derivation discipline already established) |
| GUARD-02 | Adding, removing, or renaming a tool fails a test if resource content wasn't updated to match | Pattern 4 — structural set-equality test (`companionNames` vs. `fs.ReadDir` file-stem set), directly modeled on `TestEveryRegisteredToolHasASuccessfulCallScenario`; demonstrated-red mutation proof required per `MUTATION-PROOF.md`'s established format |
</phase_requirements>

## Summary

The MCP Resources capability requires no new dependency — `Server.AddResource`/`AddResourceTemplate` are stable, general-availability APIs in the already-pinned `github.com/modelcontextprotocol/go-sdk@v1.7.0` module, confirmed by reading the module source directly at `$(go env GOMODCACHE)/github.com/modelcontextprotocol/go-sdk@v1.7.0/mcp/server.go:577-593`. The registration shape is a near-exact structural mirror of the existing `Capabilities.Tools` pattern this codebase already uses: an explicit, unconditional `ServerCapabilities.Resources` field at construction (this phase's D-11 analog), and 10 `s.AddResource(&mcp.Resource{...}, handler)` calls made unconditionally in `BuildServer`, outside the `if hasIndex` branch, so RSRC-03 is a structural property rather than an incidental one.

The drift-guard mechanism this phase extends is not new invention — `internal/mcp/tools_schema_drift_test.go`'s `engineConstantFor` map (regex-matched numeric claims pinned to `internal/query/validate.go` constants via `go/parser`) and `internal/mcp/instructions_contract_test.go`'s anchor-based prose checker (`docNamesCompanionsWithoutTheFilter`) are both directly reusable patterns. The concrete extension this research recommends: (1) generalize `numericClaimRe`/`engineConstantFor` to scan the embedded resource markdown text in addition to tool schema descriptions, keyed the same way; (2) add a structural, non-prose GUARD-02 test that derives the expected resource-URI set from `companionNames` (+ `"explore"` literal, + the 2 behavior-doc URI literals) and compares it against the actual `go:embed` filesystem's file-stem set — a rename/add/remove of a tool or resource file breaks this comparison mechanically, mirroring `TestEveryRegisteredToolHasASuccessfulCallScenario`'s exact shape in `test/wireoracle/oracle_test.go:822-853`.

One genuinely new decision this research surfaced, not covered by CONTEXT.md: go-sdk's `listResources`/`readResource` unconditionally call `setDefaultCacheableValues()`, which sets `CacheScope = "public"` (`protocol.go:1195-1197`) — and the existing `cacheScope`-correction middleware in `server.go:642-745` only patches `"tools/list"` and `"server/discover"`, never touching `"resources/list"`/`"resources/read"` at all. Since resource content is static and does not vary with `hasIndex`, `"public"` is very likely the semantically correct default and needs no override — but this is a decision the plan must make explicitly, not by omission, because it is the same class of correctness property ("ttlMs: 0 + cacheScope: private" per D-09/STATE.md) this codebase has already treated as load-bearing for a structurally similar wire field.

**Primary recommendation:** Register all 10 resources unconditionally in `BuildServer` via a single `go:embed`'d markdown directory + one registration loop that derives each resource's URI from its filename via a small, explicit map (not string manipulation), set `Capabilities.Resources = &mcp.ResourceCapabilities{}` (zero value — no `listChanged`/`subscribe`, matching the phase's explicit out-of-scope note), give every `Resource.MIMEType = "text/markdown"` so `resources/read` responses need not repeat it, and extend the two existing drift-guard test files' patterns rather than writing a third guard mechanism.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Resource content authoring (10 markdown fact-sheets) | API / Backend (`internal/mcp`) | — | Static, server-owned reference content; no client-side or storage-tier involvement |
| Resource registration (`AddResource` × 10) | API / Backend (`internal/mcp/server.go` `BuildServer`) | — | Same construction-time seam that registers tools; must run unconditionally, independent of `hasIndex` |
| `resources/list` / `resources/read` dispatch | API / Backend (go-sdk's `*mcp.Server` internals) | — | SDK-owned JSON-RPC method routing; codegraph supplies only handlers and the embedded bytes |
| Claims drift guard (GUARD-01/02) | API / Backend (`internal/mcp/*_test.go`, in-package) | — | Must read source-of-truth Go values (`companionNames`, tool `Description`/jsonschema tags, `go:embed` FS) directly; cannot live outside the `mcp` package without duplicating unexported constants |
| Wire-level proof (RSRC-01/02/03 success criterion 1, GUARD-02 success criterion 5) | Database / Storage tier is N/A — this is a **process-boundary** proof tier: `test/wireoracle` (subprocess + raw JSON-RPC) | — | Deliberately SDK-independent; must observe real stdio bytes from a spawned `codegraph serve --mcp` binary, never the server's own Go API as witness (explicit CONTEXT.md/ROADMAP constraint) |

## Standard Stack

No new dependency. `Server.AddResource`, `AddResourceTemplate`, `mcp.Resource`, `mcp.ResourceContents`, `mcp.ResourceHandler`, and `mcp.ResourceCapabilities` are all present and stable in the already-pinned `github.com/modelcontextprotocol/go-sdk@v1.7.0` `go.mod` require line `[VERIFIED: go.mod:1-3, module source at $(go env GOMODCACHE)/github.com/modelcontextprotocol/go-sdk@v1.7.0/mcp/server.go:577-613]`.

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/modelcontextprotocol/go-sdk` | v1.7.0 (already pinned) | `Server.AddResource`/`AddResourceTemplate`, `mcp.Resource`, `mcp.ResourceContents`, `mcp.ResourceCapabilities` | `[VERIFIED: go.mod:3, $(go env GOMODCACHE)/github.com/modelcontextprotocol/go-sdk@v1.7.0/mcp/server.go:577, 596, 615-653]` — the exact APIs RSRC-01/02/03 need, no upgrade required |
| Go standard library `embed` | Go 1.26.5 (project's `go.mod` directive) | `//go:embed` directive for the 10 markdown files | `[VERIFIED: go.mod:3]` — no existing `go:embed` use in this repo yet (`[VERIFIED: rg 'go:embed' --type go .` returned zero matches in the repo — this is the first use]`), but it is Go's own standard-library mechanism, not a new dependency |

### Package Legitimacy Audit

**Not applicable — this phase adds zero new Go module dependencies**, confirmed by CONTEXT.md's canonical-references note ("`Server.AddResource` is stable in the pinned `modelcontextprotocol/go-sdk@v1.7.0`; no new module dependency is needed anywhere in this milestone") and independently corroborated by reading the module source directly. No `npm view`/`pip index`/`cargo search` legitimacy gate applies.

## Architecture Patterns

### System Architecture Diagram

```
                     ┌──────────────────────────────────────────┐
                     │   Agent MCP client (any harness)          │
                     └───────────────┬────────────────────────┬─┘
                                     │ resources/list          │ resources/read <uri>
                                     ▼                          ▼
┌────────────────────────────────────────────────────────────────────────────┐
│  *mcp.Server (go-sdk-owned dispatch)                                        │
│    listResources()  ───────────────▶  serverResourceList (10 entries)       │
│    readResource(uri) ──────────────▶  lookupResourceHandler(uri) ──┐        │
└──────────────────────────────────────────────────────────────────┼─────────┘
                                                                     ▼
                     ┌───────────────────────────────────────────────────────┐
                     │  codegraph registration seam (BuildServer, server.go) │
                     │  — runs UNCONDITIONALLY, before/independent of the    │
                     │    `if hasIndex` tool-registration branch (RSRC-03)   │
                     │                                                        │
                     │  for each entry in embedded resourcesFS:               │
                     │    uri := uriFor(filename)   // explicit map, D-09/10 │
                     │    s.AddResource(&mcp.Resource{                       │
                     │        URI: uri, MIMEType: "text/markdown", ...},     │
                     │      handlerReturningEmbeddedBytes(filename))         │
                     └───────────────────────┬───────────────────────────────┘
                                              │
                                              ▼
                     ┌───────────────────────────────────────────────────────┐
                     │  //go:embed resources/*.md  → embed.FS                │
                     │  8 tool fact-sheets + 2 behavior docs (static bytes)  │
                     └───────────────────────────────────────────────────────┘

  Claims drift guard (build/test time, not runtime):
   companionNames (server.go) ──┐
   tools.go Description/jsonschema tags ──┤──▶ engineConstantFor-style      ──▶ go test ./internal/mcp/...
   internal/query/validate.go consts ──────┘    regex+anchor cross-check         (GUARD-01)
   resourcesFS file-stem set ───────────────▶ set-equality vs companionNames  ──▶ go test (GUARD-02)

  Wire-level proof (process boundary, SDK-independent):
   test/wireoracle: os/exec spawns real `codegraph serve --mcp`, drives raw
   JSON-RPC "resources/list"/"resources/read" over stdio, freezes bytes into
   testdata/wireoracle/transcripts/*.golden (RSRC-01/02/03 §1, GUARD-02 §5)
```

### Recommended Project Structure

```
internal/mcp/
├── resources/                   # NEW: 10 embedded markdown fact-sheets (D-07)
│   ├── explore.md                # codegraph://tools/explore
│   ├── node.md                   # codegraph://tools/node
│   ├── search.md                 # codegraph://tools/search
│   ├── callers.md                # codegraph://tools/callers
│   ├── callees.md                # codegraph://tools/callees
│   ├── impact.md                 # codegraph://tools/impact
│   ├── files.md                  # codegraph://tools/files
│   ├── status.md                 # codegraph://tools/status
│   ├── tools-filter.md           # codegraph://tools-filter
│   └── index-state.md            # codegraph://index-state
├── resources.go                 # NEW: go:embed directive, uriFor() map, registration loop, ResourceHandler
├── resources_test.go            # NEW: content presence + non-empty read tests (unit level)
├── resources_schema_drift_test.go  # NEW: GUARD-01/02 extension of engineConstantFor + set-equality check
├── server.go                    # MODIFIED: Capabilities.Resources added at BuildServer (~line 524-529); registerResources(s) call added unconditionally
├── tools.go                     # UNCHANGED — source of truth resource fact-sheets restate
├── tools_schema_drift_test.go   # UNCHANGED (or minimally extended if a shared helper is factored out)
└── instructions_contract_test.go # UNCHANGED — no instructions.go changes in this phase (WIRE-01 is Phase 8)

test/wireoracle/
├── scenarios.go                 # MODIFIED: +resourcesListRequest()/resourceReadRequest() helpers, new Scenario entries, ExpectedScenarioCount bump
└── (all 29 existing .golden files re-frozen — capabilities.resources changes every initialize/discover result)

testdata/wireoracle/transcripts/
└── <new resource scenario names>.golden   # NEW frozen transcripts
```

### Pattern 1: Unconditional capability + unconditional registration (RSRC-03)

**What:** `ServerCapabilities.Resources` must be set explicitly at `mcp.NewServer(...)` construction time, in the same static struct literal as `Capabilities.Tools`, and the `AddResource` calls must happen outside any `if hasIndex` conditional — exactly mirroring D-11's existing rationale for `Capabilities.Tools`.

**When to use:** Any capability that must be visible in `initialize`/`server/discover` regardless of runtime state (here: regardless of whether `.codegraph/` resolves).

**Verified current code (the exact insertion point):**
```go
// Source: internal/mcp/server.go:515-529 [VERIFIED: server.go:515-529]
// D-11: Capabilities must be set explicitly and unconditionally.
// Server.capabilities() only sets caps.Tools when HasTools ||
// tools.len() > 0 — without this, the "tools" key silently vanishes
// from the initialize response's capabilities object on the
// hasIndex=false path (MCP-03) ...
s := mcp.NewServer(&mcp.Implementation{Name: "codegraph", Version: version}, &mcp.ServerOptions{
	Capabilities: &mcp.ServerCapabilities{
		Tools: &mcp.ToolCapabilities{ListChanged: true},
		// ADD: Resources: &mcp.ResourceCapabilities{},   // zero value —
		// no listChanged/subscribe (out of scope per ROADMAP notes)
	},
	Instructions: instructions,
})
// ... immediately after, unconditionally (not inside `if hasIndex {`):
// registerResources(s)   // NEW — calls s.AddResource 10 times
```

**Why explicit `Capabilities.Resources` is technically redundant but still recommended:** go-sdk's own `capabilities()` method (`server.go:615-653`) *would* auto-populate `caps.Resources = &ResourceCapabilities{ListChanged: true}` at request time purely because `s.resources.len() > 0` once any resource is registered (`[VERIFIED: $(go env GOMODCACHE)/.../mcp/server.go:645-653]`, quoted: `"if s.opts.HasResources || s.resources.len() > 0 || s.resourceTemplates.len() > 0 { if caps.Resources == nil { caps.Resources = &ResourceCapabilities{ListChanged: true} } }"`). Relying on that implicit augmentation would (a) silently advertise `listChanged: true`, which is false — this phase explicitly puts `subscribe`/`listChanged` out of scope — and (b) break the codebase's own established convention (D-11) of never leaving a capability's presence to SDK-default inference. Setting `Resources: &mcp.ResourceCapabilities{}` explicitly avoids both problems and keeps `GUARD-01`'s "every claim is source-derived, not implicit" spirit.

### Pattern 2: `go:embed`'d markdown, one FS, one filename→URI map, one registration loop (D-07)

**What:** A single `embed.FS` over `internal/mcp/resources/*.md`, iterated once via `fs.ReadDir`, with an explicit `map[string]string` (filename stem → URI) as the single source of truth for the URI scheme (D-08/D-09/D-10) — never string concatenation that could silently produce a wrong URI for a renamed file.

**When to use:** Any static, server-owned, and MIME-typed content set that must register generically rather than via 10 hand-written `AddResource` calls (which would itself be a duplicated-fact risk GUARD-02 exists to prevent).

**Example (illustrative — exact code and file layout are Claude's discretion per CONTEXT.md):**
```go
// Source: pattern derived from go-sdk's own registration API
// ($(go env GOMODCACHE)/github.com/modelcontextprotocol/go-sdk@v1.7.0/mcp/server.go:577-586)
// and this codebase's existing registerTools loop (internal/mcp/server.go:427-449).

//go:embed resources/*.md
var resourcesFS embed.FS

// resourceURIFor is the ONE place the URI scheme lives — D-08 (custom
// codegraph:// scheme), D-09 (per-tool URIs use companion short names),
// D-10 (behavior-doc URIs are named for content, not mechanism).
var resourceURIFor = map[string]string{
	"explore.md":      "codegraph://tools/explore",
	"node.md":          "codegraph://tools/node",
	"search.md":        "codegraph://tools/search",
	"callers.md":       "codegraph://tools/callers",
	"callees.md":       "codegraph://tools/callees",
	"impact.md":        "codegraph://tools/impact",
	"files.md":         "codegraph://tools/files",
	"status.md":        "codegraph://tools/status",
	"tools-filter.md":  "codegraph://tools-filter",
	"index-state.md":   "codegraph://index-state",
}

func registerResources(s *mcp.Server) {
	entries, err := fs.ReadDir(resourcesFS, "resources")
	if err != nil {
		panic(fmt.Sprintf("registerResources: read embedded resources dir: %v", err))
	}
	for _, e := range entries {
		name := e.Name()
		uri, ok := resourceURIFor[name]
		if !ok {
			panic(fmt.Sprintf("registerResources: %s has no entry in resourceURIFor", name))
		}
		data, err := resourcesFS.ReadFile("resources/" + name)
		if err != nil {
			panic(fmt.Sprintf("registerResources: read %s: %v", name, err))
		}
		s.AddResource(&mcp.Resource{
			URI:      uri,
			Name:     name,
			MIMEType: "text/markdown",
		}, func(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			return &mcp.ReadResourceResult{
				Contents: []*mcp.ResourceContents{{URI: req.Params.URI, Text: string(data)}},
			}, nil
		})
	}
}
```
Note: the closure captures `data` per-iteration correctly in Go 1.22+ loop-variable semantics (project's `go 1.26.5` — no `data := data` shadow needed) `[VERIFIED: go.mod:3]`.

**Why `MIMEType` on the `Resource` (not repeated per-read):** go-sdk's `readResource` auto-populates `ResourceContents.MIMEType` from the registered `Resource.MIMEType` whenever the handler leaves it empty (`[VERIFIED: $(go env GOMODCACHE)/.../mcp/server.go:1040-1048]`, quoted: `"if c.MIMEType == \"\" { c.MIMEType = mimeType }"`), so the handler only needs to set `Text`, not `MIMEType`, on every read.

### Pattern 3: Extending the existing numeric-claim drift guard (GUARD-01)

**What:** `tools_schema_drift_test.go`'s `numericClaimRe`/`engineConstantFor` pattern already exists and is directly reusable — it just needs a second text source (the embedded resource markdown) fed through the same regex+map check.

**Verified current pattern:**
```go
// Source: internal/mcp/tools_schema_drift_test.go:16-32 [VERIFIED: tools_schema_drift_test.go:16-32]
var numericClaimRe = regexp.MustCompile(`(?i)\b(default|max)\s+(\d+)`)

var engineConstantFor = map[string]string{
	"codegraph_impact.depth.default":      "defaultDepth",
	"codegraph_impact.depth.max":          "MaxDepth",
	"codegraph_explore.max_files.default": "defaultMaxFiles",
}
```

**Recommended extension (implementation approach — Claude's discretion per CONTEXT.md):** iterate the same `resourcesFS` entries in a new test, run `numericClaimRe.FindAllStringSubmatch` against each file's text, and key claims as `"<resource-uri-or-filename>.<claim-kind>"` into either the *same* `engineConstantFor` map (if the fact-sheet restates a tool's numeric default) or a small sibling map — either is defensible; keeping ONE map is simpler and directly proves the fact-sheet's number equals the same engine constant the tool schema itself is pinned to, which is the strongest form of "cannot drift independently."

**Claim tag:** `[ASSUMED]` — the exact merge-vs-sibling-map choice and the exact regex reuse strategy are this research's recommendation, not a verified existing mechanism (the resource markdown files do not exist yet in this codebase).

### Pattern 4: Structural set-equality guard for resource *existence* (GUARD-02)

**What:** `test/wireoracle/oracle_test.go:822-853`'s `TestEveryRegisteredToolHasASuccessfulCallScenario` already solves the exact structural problem GUARD-02 describes for tools (derive the expected name set from source, derive the actual name set from a live capture, assert equality) — an in-package analog for resources is a direct, minimal-novelty extension.

**Verified precedent:**
```go
// Source: test/wireoracle/oracle_test.go:822-853 [VERIFIED: oracle_test.go:822-853]
func TestEveryRegisteredToolHasASuccessfulCallScenario(t *testing.T) {
	registered := toolNamesFromCapture(t, "toolslist-repeat", 2)
	if len(registered) != 8 {
		t.Fatalf("toolslist-repeat: registered %d tools, want 8: %v", len(registered), registered)
	}
	// ... derives `remaining` from source, deletes as scenarios prove
	// coverage, fails naming exactly what's missing.
}
```

**Recommended in-package analog (illustrative):** a test that (1) derives the *expected* resource-name set from `companionNames` + the literal `"explore"` + the two behavior-doc name literals (`"tools-filter"`, `"index-state"`) — the same sourcing `instructions_contract_test.go`'s tests already use for `companionNames` — and (2) derives the *actual* set from `fs.ReadDir(resourcesFS, "resources")`, then asserts the two sets are equal (both directions: missing file, and orphaned file). This is GUARD-02's exact contract: renaming a tool in `companionNames` without renaming its `.md` file turns this test red; the reverse (renaming the `.md` file without touching `companionNames`) also turns it red — genuinely bidirectional, unlike a one-way "file exists" check.

**Demonstrating this RED (required by success criterion 3):** apply a real mutation — e.g. rename `node.md` to `sym.md` without updating `companionNames` or `resourceURIFor` — run `go test ./internal/mcp/...`, capture the failure, then revert byte-clean. This mirrors `MUTATION-PROOF.md`'s "Mutation 2 — a dropped tool" demonstration exactly (`[VERIFIED: test/wireoracle/MUTATION-PROOF.md:93-168]`).

### Anti-Patterns to Avoid

- **Hand-typing the tool-count "8" or the URI list into a second Go literal.** Every fact used by both the registration loop and its guard test must come from ONE source (`companionNames`, the `resourceURIFor` map, or `fs.ReadDir`) — never re-typed in the test file, which is exactly the class of bug `TestMCPToolSchemaNumericClaimsMatchEngineConstants`'s doc comment describes as "SURF-01."
- **Relying on go-sdk's implicit `caps.Resources` augmentation instead of an explicit `ServerCapabilities.Resources` field.** Technically works (see Pattern 1) but silently advertises `listChanged: true`, which this phase's scope explicitly excludes, and breaks the codebase's established explicit-capability convention.
- **Batching multiple `resources/read` calls into one wire-oracle scenario.** The existing corpus enforces "at most one [async-dispatched] request per scenario, and it must be last" specifically because `tools/call` requests race each other and race trailing requests in go-sdk's worker pool (`[VERIFIED: test/wireoracle/scenarios.go:440-456]`). Whether `resources/read` shares this same async-dispatch risk is `[ASSUMED]` — not independently confirmed against go-sdk's jsonrpc2 dispatch internals in this research session — so the SAFE default is to treat every `resources/read` the same way: at most one per scenario, always last, until a specific reason to relax that is found.
- **Repeating the "empty until indexed" caveat inside each of the 8 tool fact-sheets.** Already decided against in CONTEXT.md D-02 — restating it 8 times is exactly the kind of duplicated fact GUARD-01 exists to prevent.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|--------------|-----|
| Resource content serving | A custom `resources/*` JSON-RPC method handler wired manually into the transport | `mcp.Server.AddResource`/`AddResourceTemplate` | The SDK already implements `resources/list` pagination, `resources/read` lookup/dispatch, and `ResourceNotFoundError` (`-32602`, SEP-2164-compliant) — reimplementing any of this duplicates SDK internals for no benefit |
| Claims drift detection | A new bespoke assertion mechanism for resource prose | Extend `engineConstantFor` (numeric claims) and the `docNamesCompanionsWithoutTheFilter`-style anchor pattern (prose claims) already in `internal/mcp` | CONTEXT.md's own established pattern: "GUARD-01/02 should follow this same shape... not introduce a parallel mechanism" |
| MIME-type propagation on read | Setting `MIMEType` on every `ResourceContents` returned from a handler | Set `MIMEType` once on the registered `Resource`; let `readResource`'s auto-populate fill it in | `[VERIFIED: $(go env GOMODCACHE)/.../mcp/server.go:1040-1048]` — the SDK already does this |

**Key insight:** every mechanism this phase needs (resource registration, capability advertisement, content-vs-source drift checking, wire-level proof) already has a proven, working, in-repo precedent from a structurally identical problem (tools). The phase's actual work is applying that precedent 10 more times, not inventing a new one.

## Common Pitfalls

### Pitfall 1: `cacheScope` silently defaults to `"public"` for resources
**What goes wrong:** go-sdk's `listResources`/`readResource` both call `res.setDefaultCacheableValues()`, which sets `CacheScope = "public"` (`[VERIFIED: $(go env GOMODCACHE)/.../mcp/protocol.go:1195-1197]`, quoted: `"func (c *Cacheable) setDefaultCacheableValues() { c.CacheScope = \"public\" }"`). The existing `cacheScope`-correction middleware in `server.go:642-745` only patches the `"tools/list"` and `"server/discover"` method cases — `"resources/list"`/`"resources/read"` are absent from its `switch method` (`[VERIFIED: server.go:653, 661-745]`).
**Why it happens:** the existing middleware was written for SPEC-04, before resources existed, and its `switch` is method-literal, not a catch-all.
**How to avoid:** decide explicitly (not by silent omission) whether `"public"` is correct. It is very likely CORRECT here — unlike `tools/list`, resource content never varies with `hasIndex`/`companions`/repo state, so caching it across repos is actually safe — but the plan must state this decision rather than let it happen by absence, matching the standing decision in STATE.md ("ttlMs: 0 + cacheScope: 'private' paired with a per-call hasIndex re-check — two halves of one correctness property, not independent options").
**Warning signs:** a wire-oracle transcript for a new resource scenario showing `"cacheScope":"public"` where a reviewer expected `"private"` (or vice versa) without that being a deliberate, reviewed line in the reviewed-diff pass.

### Pitfall 2: The `capabilities` field's key ORDER changes in every one of the 29 existing frozen transcripts
**What goes wrong:** `ServerCapabilities`'s Go struct field order is `Experimental, Extensions, Completions, Logging, Prompts, Resources, Tools` (`[VERIFIED: $(go env GOMODCACHE)/.../mcp/protocol.go:2264-2294]`). Go's `encoding/json` marshals struct fields in **declaration order**, so once `Resources` is set, every existing transcript's `"capabilities":{"tools":{"listChanged":true}}}` becomes `"capabilities":{"resources":{},"tools":{"listChanged":true}}}` — `resources` sorts BEFORE `tools` in the wire bytes, confirmed against the live golden file `[VERIFIED: testdata/wireoracle/transcripts/handshake-explore.golden:1]`, current bytes quoted: `{"jsonrpc":"2.0","id":1,"result":{"capabilities":{"tools":{"listChanged":true}}, ...`.
**Why it happens:** this is exactly the mechanical, structural cause success criterion 5 anticipates ("every changed transcript's diff attributed to a named cause with a count") — the cause here is precisely "capabilities.resources appears in every initialize/discover result," and it moves **all 29** existing transcripts' bytes, not a subset.
**How to avoid:** during the reviewed-diff re-capture pass, name this cause explicitly and expect its count to be 29 (every existing scenario) plus however many new resource-specific scenarios are added.
**Warning signs:** a re-capture diff where fewer than 29 transcripts change — that would mean `Capabilities.Resources` isn't actually reaching every code path (e.g., accidentally scoped inside `if hasIndex`).

### Pitfall 3: `resources/read` on an unregistered URI must not leak server configuration
**What goes wrong:** go-sdk's `readResource` deliberately returns the SAME `ResourceNotFoundError` for both "URI genuinely doesn't exist" and "resource exists but handler failed to find it internally" (`[VERIFIED: $(go env GOMODCACHE)/.../mcp/server.go:1020-1025]`, quoted comment: `"Don't expose the server configuration to the client. Treat an unregistered resource the same as a registered one that couldn't be found."`). Error code default is `-32602` (Invalid Params, SEP-2164) unless `MCPGODEBUG=customresnotfounderrcode=1` restores the pre-1.7.0 `-32002`.
**Why it happens:** SDK-level security posture change (SEP-2164), not something codegraph controls.
**How to avoid:** any wire-oracle error-shape scenario for an unknown resource URI should assert `-32602`, matching the codebase's existing `error-unknown-tool`/`error-malformed-args` scenarios' error code (both already `-32602`, per `COVERAGE-BASELINE.md`), for consistency — not `-32002`.
**Warning signs:** a frozen transcript asserting `-32002` for an unknown-resource scenario would indicate the deprecated `MCPGODEBUG` compatibility flag is accidentally set in the capture environment.

### Pitfall 4: `AddResource` panics on a malformed or relative URI
**What goes wrong:** `AddResource`'s doc comment states it "panics if the resource URI is invalid or not absolute (has an empty scheme)" (`[VERIFIED: $(go env GOMODCACHE)/.../mcp/server.go:576]`). A typo like `codegraph:tools/explore` (missing `//`) still has a non-empty scheme so it would NOT panic, but `tools/explore` (no scheme at all) WOULD panic at server-construction time, crashing the whole process on startup.
**Why it happens:** URL scheme validation is strict but silent about the `//` authority marker specifically — `url.Parse` accepts opaque URIs (`scheme:opaque`) as "absolute" too, so the panic guard only catches the fully-missing-scheme case.
**How to avoid:** all 10 URIs in this phase were verified in this research session to parse cleanly and absolutely via `net/url.Parse` (see Sources) — `codegraph://tools/<name>`, `codegraph://tools-filter`, `codegraph://index-state`. Keep the `resourceURIFor` map's values byte-identical to what was verified; do not introduce a new URI shape without re-checking `url.Parse` against it.
**Warning signs:** `go build`/`go test` failing with a panic trace pointing at `AddResource` during `BuildServer` — this is a startup-time crash, not a request-time error, so it would fail EVERY test that constructs a server, not just resource-specific ones.

## Code Examples

### Registering a resource with a static-bytes handler
```go
// Source: pattern synthesized from go-sdk's own example
// ($(go env GOMODCACHE)/github.com/modelcontextprotocol/go-sdk@v1.7.0/mcp/server_example_test.go:167-176)
// [VERIFIED: server_example_test.go:167-176]
handler := func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{URI: uri, Text: c}},
	}, nil
}
s := mcp.NewServer(&mcp.Implementation{Name: "server", Version: "v0.0.1"}, nil)
s.AddResource(&mcp.Resource{URI: "file:///a"}, handler)
```

### Testing resources/list and resources/read via an in-memory session (unit-level, in-package)
```go
// Source: existing codebase pattern, internal/mcp/server_test.go:74-127
// [VERIFIED: server_test.go:74-127]
// newTestSession already exists; ListTools is already exercised this way.
// ClientSession.ListResources and ClientSession.ReadResource exist with the
// identical shape [VERIFIED: $(go env GOMODCACHE)/.../mcp/client.go:1309-1345]:
//   func (cs *ClientSession) ListResources(ctx, *ListResourcesParams) (*ListResourcesResult, error)
//   func (cs *ClientSession) ReadResource(ctx, *ReadResourceParams) (*ReadResourceResult, error)
session, cleanup := newTestSession(t, s)
defer cleanup()
result, err := session.ListResources(context.Background(), nil)
// result.Resources: []*mcp.Resource, one per registered URI
```
**Scope note:** this in-package/in-memory pattern satisfies GUARD-01/02's unit-level drift checks. It does **not** satisfy RSRC-01/02's success criterion 1 ("observed on the wire... never by calling the server's own Go API as its own witness") — that half is `test/wireoracle`'s job specifically, via a spawned subprocess and raw JSON-RPC bytes.

### Wire-oracle scenario shape for a new resources/list + resources/read pair (pattern, not yet in codebase)
```go
// Source: pattern derived from test/wireoracle/scenarios.go:78-101's existing
// toolsListRequest/toolCallRequest helpers [VERIFIED: scenarios.go:78-101]
func resourcesListRequest(id int) map[string]any {
	return map[string]any{"jsonrpc": "2.0", "id": id, "method": "resources/list"}
}
func resourceReadRequest(id int, uri string) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0", "id": id, "method": "resources/read",
		"params": map[string]any{"uri": uri},
	}
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| `ResourceNotFoundError` returned `-32002` | Returns `-32602` (Invalid Params, SEP-2164) by default | go-sdk v1.7.0 (this codebase's already-pinned version) | Codebase's existing error-shape scenarios (`-32602` for `error-unknown-tool`/`error-malformed-args`) are already consistent with this; no migration needed, just alignment to confirm when adding a resource-not-found scenario |

**Deprecated/outdated:** none relevant — this is a net-new capability in this codebase, not a migration off a prior mechanism.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|----------------|
| A1 | The exact merge-vs-sibling-map choice for extending `engineConstantFor` to cover resource-markdown numeric claims (Pattern 3) | Architecture Patterns § Pattern 3 | Low — either choice satisfies GUARD-01's contract; only affects test-file organization, not correctness |
| A2 | `resources/read` requests are subject to the SAME async worker-pool ordering race that governs `tools/call` in wire-oracle scenarios (Anti-Patterns) | Architecture Patterns § Anti-Patterns to Avoid | Medium — if wrong (resources/read turns out to be synchronously dispatched), the plan could safely batch multiple reads per scenario and reduce subprocess-capture cost; if the assumption is dropped without verification and turns out RIGHT, the corpus could gain an intermittent CI flake identical to the one `scenarios.go:440-456` already documents having been fixed once |
| A3 | `cacheScope: "public"` (the SDK default, uncorrected) is the semantically CORRECT value for `resources/list`/`resources/read`, given resource content is invariant to `hasIndex` | Common Pitfalls § Pitfall 1 | Medium — if this project's maintainer wants resource caching treated with the same suspicion as tool-list caching (e.g., because resource CONTENT could change across a `codegraph upgrade` mid-session), the middleware needs an explicit `"resources/list"`/`"resources/read"` case added, mirroring the `tools/list` fix. This must be a stated decision, not a silent default. |
| A4 | Filename-to-URI mapping should be an explicit Go map (`resourceURIFor`) rather than a naming convention parsed from the filename string | Architecture Patterns § Pattern 2 | Low — an explicit map is strictly safer (a typo produces a build-time-obvious missing-key panic rather than a silently-wrong derived URI), but a convention-based approach is also viable and was not tested against |

## Open Questions

1. **Should `cacheScope` be forced to `"private"` for resources, matching `tools/list`'s treatment, or is the SDK's `"public"` default correct?**
   - What we know: the existing middleware's `switch method` deliberately enumerates only `"initialize"`, `"tools/list"`, `"tools/call"`, `"server/discover"` — resources methods are absent, so they get `"public"` by SDK default `[VERIFIED: protocol.go:1195-1197, server.go:653]`.
   - What's unclear: whether "public" is a deliberate correct choice for this phase or an oversight the plan should close.
   - Recommendation: treat resource content as legitimately public-cacheable (it doesn't depend on `hasIndex`/repo state, unlike the tool catalog) unless the planner identifies a reason resource content itself could change within a session (e.g., a future `codegraph upgrade` mid-process) that would make stale caching a real correctness bug — flag explicitly in the plan either way, do not let it default silently.

2. **Does `resources/read` share `tools/call`'s async worker-pool response-ordering race?**
   - What we know: `tools/call` demonstrably races other requests in the same session at the SDK dispatch layer (`[VERIFIED: scenarios.go:440-449]`, a directly observed, previously-reproduced non-determinism, not a theoretical concern).
   - What's unclear: whether `resources/read` shares the same dispatch path (this research did not trace go-sdk's `jsonrpc2` connection-handling internals deeply enough to confirm or rule this out with the same confidence).
   - Recommendation: default to the SAME safe scenario shape used for `tools/call` — at most one `resources/read` per wire-oracle scenario, always last — until/unless a specific investigation rules the race out for this method.

3. **How many new wire-oracle scenarios does RSRC-01/02/03's success criterion 1 actually require — one combined `resources/list` scenario plus 10 individual `resources/read` scenarios (mirroring the 7 existing `call-*` scenarios), or a smaller sufficient set?**
   - What we know: the existing `call-*` pattern uses one scenario per tool (7 scenarios) plus the tracer covering the 8th (`codegraph_explore`) — a directly analogous "one scenario per resource" approach would add roughly 10-11 new scenarios (1 `resources/list` + 10 `resources/read`, or fewer if some reads are combined into `toolslist-no-index`'s analog for success criterion 2).
   - What's unclear: whether the phase's own "no re-freeze deferred to a later phase" and "main is green at the phase boundary" constraints make a smaller, curated scenario set (e.g., one `resources/read` per DISTINCT content shape rather than per URI) sufficient, given `TestEveryRegisteredToolHasASuccessfulCallScenario`'s structural analog for resources could instead run at the unit level (Pattern 4) rather than requiring full wire-oracle coverage per URI.
   - Recommendation: this is squarely "Claude's Discretion" per CONTEXT.md ("Wire-oracle scenario additions/re-capture mechanics for resources/list/resources/read — follows the existing test/wireoracle reviewed-diff discipline; planner/researcher own the specifics") — the planner should size this deliberately against success criterion 1's literal wording ("gets non-empty text/markdown back from resources/read on every URI that list advertised — observed on the wire") which reads as requiring wire-level coverage of ALL 10 URIs, not a sample.

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go standard `testing` package |
| Config file | none — `go test ./...` via `Taskfile.yml`'s `test`/`test:wireoracle` tasks `[VERIFIED: test/wireoracle/MUTATION-PROOF.md:119-122 shows `task test:wireoracle` invoking `go test ./test/wireoracle/...`]` |
| Quick run command | `go test ./internal/mcp/...` |
| Full suite command | `go test ./... ` and `task test:wireoracle` (`go test ./test/wireoracle/...`) |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|---------------------|--------------|
| RSRC-01 | `resources/list` returns one entry per tool + 2 behavior docs | unit + wire | `go test ./internal/mcp/... -run TestResource` / `task test:wireoracle` | ❌ Wave 0 (new files) |
| RSRC-02 | `resources/read` returns non-empty `text/markdown` on every listed URI | unit + wire | same as above | ❌ Wave 0 |
| RSRC-03 | Resources register with zero index-gated tools | unit + wire | new `TestResourcesRegisterWithoutIndex`-style test + wire scenario mirroring `toolslist-no-index` | ❌ Wave 0 |
| GUARD-01 | Every numeric/env-var claim is source-derived or test-pinned | unit | extend `TestMCPToolSchemaNumericClaimsMatchEngineConstants`-style test to scan `resourcesFS` | ❌ Wave 0 |
| GUARD-02 | Tool add/remove/rename fails a resource-content test | unit | new set-equality test, `companionNames` vs `fs.ReadDir(resourcesFS, "resources")` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/mcp/...`
- **Per wave merge:** `go test ./... && task test:wireoracle`
- **Phase gate:** Full suite green before `/gsd-verify-work`, including a demonstrated-red-then-reverted mutation proof appended to `test/wireoracle/MUTATION-PROOF.md` for GUARD-02 (success criterion 3's explicit requirement: "demonstrated by applying that mutation, watching it fail, and reverting byte-clean, rather than asserted from reading the guard").

### Wave 0 Gaps
- [ ] `internal/mcp/resources/*.md` — the 10 fact-sheet/behavior-doc source files (content authoring, D-01 through D-10)
- [ ] `internal/mcp/resources.go` — `go:embed` directive, `resourceURIFor` map, `registerResources(s)` function
- [ ] `internal/mcp/resources_test.go` — unit-level list/read coverage (mirrors `server_test.go`'s `newTestSession`/`listToolNames` pattern)
- [ ] `internal/mcp/resources_schema_drift_test.go` (or extend `tools_schema_drift_test.go` directly) — GUARD-01/02 extension
- [ ] `test/wireoracle/scenarios.go` additions — new request helpers + new `Scenario` entries + `ExpectedScenarioCount` bump
- [ ] All 29 existing `testdata/wireoracle/transcripts/*.golden` files re-frozen (capabilities.resources changes every one) + N new `.golden` files for the added scenarios

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|----------------|---------|--------------------|
| V2 Authentication | No | MCP stdio transport has no auth layer in this codebase's model; unchanged by this phase |
| V3 Session Management | No | Resources are stateless, static, per-process content — no session-scoped resource state introduced |
| V4 Access Control | No | All 10 resources are uniformly readable by any connected client, same as all 8 tools — no differential access control exists or is needed |
| V5 Input Validation | Marginal — yes | `resources/read`'s only client-supplied input is the URI string itself; go-sdk's own `lookupResourceHandler` does exact-match lookup against the registered set (`[VERIFIED: server.go:1054]` area) — an unregistered URI cannot reach the filesystem or any handler logic, since these handlers close over pre-loaded `embed.FS` bytes, not a path-construction step. No new validation code is needed; the security property is structural (no path ever touches `os`/`filepath` at request time for THIS phase's static-content handlers — contrast with go-sdk's OWN `readFileResource` helper, which this phase does NOT use). |
| V6 Cryptography | No | Not applicable — no cryptographic material involved |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|------------------------|
| Path traversal via a crafted resource URI (e.g., `codegraph://tools/../../etc/passwd`) | Tampering / Information Disclosure | **Structurally avoided, not defended against:** this phase's handlers never resolve the client-supplied URI against a filesystem path at request time — they close over pre-loaded `embed.FS` bytes selected at REGISTRATION time via the exact-match `resourceURIFor` map, so a URI outside the 10 registered ones simply fails the SDK's own exact-URI lookup with `ResourceNotFoundError` before any codegraph handler code runs at all. This is stronger than sanitizing a path — there is no path to sanitize. (Contrast: go-sdk's own optional `readFileResource` helper DOES resolve URIs against a filesystem root with root-confinement checks `[VERIFIED: resource.go:68-132]` — codegraph does not use that helper for this phase's static content, so its confinement logic is not this phase's concern, but is worth knowing about if a FUTURE resource type ever serves live filesystem content.) |
| Resource content disclosing server configuration or filesystem layout | Information Disclosure | D-01 through D-04 already constrain content to mechanical fact-sheets (params/defaults/return-shapes) with no worked examples or cross-references — no repo path, hostname, or environment value is ever embedded, mirroring the EXACT same constraint the `instructions` wire-string already enforces (`[VERIFIED: server.go:47-49]`, quoted: "no repository path, no resolved index root, no hostname, no environment value"). This phase's markdown is static at build time (embedded), so it structurally cannot vary by deployment the way a runtime-interpolated string could. |

## Sources

### Primary (HIGH confidence)
- `$(go env GOMODCACHE)/github.com/modelcontextprotocol/go-sdk@v1.7.0/mcp/server.go` — `AddResource`/`AddResourceTemplate`/`RemoveResources`/`capabilities()`/`listResources`/`readResource`/`lookupResourceHandler` — read directly, lines 560-1120
- `$(go env GOMODCACHE)/github.com/modelcontextprotocol/go-sdk@v1.7.0/mcp/protocol.go` — `ServerCapabilities`, `ResourceCapabilities`, `ToolCapabilities`, `Resource`, `ReadResourceResult`, `ReadResourceParams`, `Cacheable`/`setDefaultCacheableValues` — read directly, lines 1168-2294
- `$(go env GOMODCACHE)/github.com/modelcontextprotocol/go-sdk@v1.7.0/mcp/resource.go` — `ResourceHandler`, `ResourceNotFoundError`, `readFileResource`/confinement helpers (not used by this phase, but confirms the SDK's own filesystem-resource pattern) — read directly, full file
- `$(go env GOMODCACHE)/github.com/modelcontextprotocol/go-sdk@v1.7.0/mcp/client.go` — `ClientSession.ListResources`/`ReadResource`/`ListResourceTemplates` — grepped and confirmed present, lines 1309-1345
- `$(go env GOMODCACHE)/github.com/modelcontextprotocol/go-sdk@v1.7.0/mcp/server_example_test.go` — canonical `AddResource`/`AddResourceTemplate` usage example — read directly, lines 160-190
- `internal/mcp/server.go` — `BuildServer`, `companionNames`, `instructions`, the `cacheScope`-correction middleware — read directly, lines 40-752 (multiple ranges)
- `internal/mcp/tools.go` — all 8 tool `Description`s and jsonschema-tagged arg structs — read directly, lines 161-544
- `internal/mcp/tools_schema_drift_test.go` — `engineConstantFor`/`numericClaimRe`/`parseQueryConstants` — read directly, full file
- `internal/mcp/instructions_contract_test.go` — `docNamesCompanionsWithoutTheFilter` anchor pattern — read directly, full file
- `internal/mcp/server_test.go` — `newTestSession`/`listToolNames` in-memory test pattern — read directly, lines 60-127
- `test/wireoracle/scenarios.go` — `Scenarios()`, `ExpectedScenarioCount`, request-builder helpers, the async-ordering invariant comment — read directly, lines 1-120, 440-505
- `test/wireoracle/oracle_test.go` — `TestFrozenTranscriptsMatch`, `TestScenarioCountIsExact`, `TestTranscriptSetMatchesScenarioSet`, `TestEveryRegisteredToolHasASuccessfulCallScenario` — read directly, lines 1-180, 813-862
- `test/wireoracle/COVERAGE-BASELINE.md` — full scenario index and the corpus-extension protocol — read directly, full file
- `test/wireoracle/MUTATION-PROOF.md` — Mutation 2 demonstrated-red example — read directly, lines 93-172
- `testdata/wireoracle/transcripts/handshake-explore.golden` — live confirmation of current `capabilities` wire shape — read directly
- Local `net/url.Parse` execution against all 10 proposed URIs — verified no panic risk, all absolute — executed in this session

### Secondary (MEDIUM confidence)
- None — every claim in this document that is not explicitly `[ASSUMED]` was verified directly against the pinned module source, this repository's existing code, or a locally executed check in this research session.

### Tertiary (LOW confidence)
- None.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — zero new dependencies; every API used was read directly from the pinned module source at the exact pinned version
- Architecture: HIGH — every registration/capability pattern recommended has a byte-verified precedent already live in this codebase (`Capabilities.Tools`, `registerTools`, the cacheScope middleware)
- Pitfalls: HIGH for Pitfalls 1-4 (all directly verified against source); the two Open Questions items (A2/Q2, async dispatch race for `resources/read`) are honestly flagged MEDIUM/LOW since go-sdk's `jsonrpc2` connection-dispatch internals were not traced deep enough in this session to confirm or rule out

**Research date:** 2026-08-12
**Valid until:** 30 days (go-sdk is pinned and stable; the risk is a future go-sdk upgrade elsewhere in the milestone changing resource-capability defaults, not organic staleness)
