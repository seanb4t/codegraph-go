# Phase 8: Instructions & Marker-Block Rewrite - Pattern Map

**Mapped:** 2026-08-13
**Files analyzed:** 3 modified (no new files — pure in-place edits)
**Analogs found:** 3 / 3 (all edits are self-analogous — the closest pattern for each file is its own existing content, extended in place)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `internal/mcp/server.go` (`instructions` const, WIRE-01) | config (wire-contract string literal) | request-response (served in MCP `initialize`/`server/discover`) | itself (in-place literal rewrite) | exact — no other file in the repo carries a byte-budgeted wire-contract string |
| `internal/mcp/instructions_contract_test.go` (WIRE-03 new guard tests) | test (derive-from-source guard) | request-response (in-memory MCP session round-trip) | `internal/mcp/resources_test.go` (`newTestSession`/`ListResources`/`ReadResource` pattern) | exact — same package, same helper, proven pattern from Phase 5 |
| `internal/agents/instructions.go` (`codegraphInstructionsBlock` + doc comment, WIRE-02/D-05) | config (marker-fenced doc-injection string) | file-I/O (upserted into agent instruction files by `codegraph install`) | itself (in-place literal rewrite); fence contract unchanged | exact — no other file owns this const |

## Pattern Assignments

### `internal/mcp/server.go` (config, request-response)

**Analog:** itself — `internal/mcp/server.go:41-56` (current `instructions` doc comment + const)

**Current const, byte-verified 580/600 bytes** (`internal/mcp/server.go:56`):
```go
const instructions = "codegraph indexes this repository's code into a call and symbol graph; try codegraph_explore first for a where-is-X or how-does-Y-work question, since it returns verbatim source plus call paths in one call. Every tool accepts an optional path argument; omitting it uses this server's own working directory. All eight tools register by default and appear automatically once an index exists, with no client restart required; an empty tool list means this repository has no index yet, so run codegraph init. Setting CODEGRAPH_MCP_TOOLS narrows the surface to the companions it names."
```

**Hard constraints carried in the doc comment above it** (`server.go:41-55`) — MUST preserve verbatim in the rewrite:
- Compile-time literal, zero interpolation, ever (no repo path, no resolved value) — T-03-19.
- Single paragraph, no `\n`/`\r`.
- ≤ `instructionsMaxBytes` (600, defined in the test file, not raised — D-01).

**Recommended rewrite (RESEARCH.md's byte-verified Candidate A, 566 bytes)** — cut the path-argument clause (zero anchor overlap), append a generic skill/resources pointer:
```
codegraph indexes this repository's code into a call and symbol graph; try codegraph_explore first for a where-is-X or how-does-Y-work question, since it returns verbatim source plus call paths in one call. All eight tools register by default and appear automatically once an index exists, with no client restart required; an empty tool list means this repository has no index yet, so run codegraph init. Setting CODEGRAPH_MCP_TOOLS narrows the surface to the companions it names. See the codegraph skill and call resources/list for full tool-by-tool reference docs.
```
Verified: contains `"default"`, `CODEGRAPH_MCP_TOOLS`, `"codegraph init"` (all 3 required anchors, D-02); no newline; 566 ≤ 600.

**Anchors that MUST survive** (pinned by `TestInstructionsDescribesEveryVisibilityMechanism`, `internal/mcp/instructions_contract_test.go:88-104`):
- `"default"`
- `CODEGRAPH_MCP_TOOLS` (via `allowlistEnvName` const, same file line 22)
- `"codegraph init"`

**Clause safe to cut** — the path-argument sentence carries none of the three anchors (verified in RESEARCH.md Pitfall 2); do NOT cut the default/init clause, it carries two of three anchors.

---

### `internal/mcp/instructions_contract_test.go` (test, request-response)

**Analog:** `internal/mcp/resources_test.go:11-40` (`listResourceURIs`, `TestResourcesListAdvertisesRegisteredURIs`) for the resources half; `claudeassets.go:39-52` for the skill half.

**Existing test-file conventions to match** (this file, lines 1-33):
```go
package mcp

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

const allowlistEnvName = "CODEGRAPH_MCP_TOOLS"
const instructionsMaxBytes = 600
```
New WIRE-03 tests belong in this same file (extend, don't create a parallel guard file — RESEARCH.md's explicit recommendation), so add `claudeassets` and `context` to imports as needed:
```go
import claudeassets "github.com/seanb4t/codegraph-go"
```
Verified no import cycle: `internal/mcp` currently imports only `internal/gitmeta`/`internal/query`; `internal/agents` imports the root `claudeassets` package but `internal/mcp` does not import `internal/agents` at all.

**Skill-resolvability guard (WIRE-03) — source of truth, `claudeassets.go:39-52`:**
```go
const SkillMarkdownPath = ".claude/skills/codegraph/SKILL.md"
func SkillMarkdown() ([]byte, error) { return FS.ReadFile(SkillMarkdownPath) }
```
Pattern: `strings.Contains(instructions, "skill")` gate → `claudeassets.SkillMarkdown()` must return non-empty bytes. Never re-type the path — read it from the exported const, same bytes `internal/agents/claude.go`'s `Install` writes.

**Resources-resolvability guard (WIRE-03) — source of truth, `internal/mcp/resources_test.go:11-40`:**
```go
session, cleanup := newTestSession(t, s)
defer cleanup()

result, err := session.ListResources(context.Background(), nil)
if err != nil {
    t.Fatalf("ListResources: %v", err)
}
if len(result.Resources) == 0 {
    t.Fatalf("ListResources returned no resources")
}
```
Followed by a `session.ReadResource(ctx, &mcp.ReadResourceParams{URI: r.URI})` round-trip on at least one URI, asserting non-empty `Text`. `newTestSession` and `BuildServer` are existing helpers already in package `mcp` (server_test.go) — reuse, don't reimplement.

**Existing anchor-pinning shape to mirror for the new tests** (`TestInstructionsDescribesEveryVisibilityMechanism`, lines 88-104):
```go
func TestInstructionsDescribesEveryVisibilityMechanism(t *testing.T) {
	mechanisms := []struct {
		mechanism string
		anchor    string
	}{
		{"the default tool surface", "default"},
		{"the CODEGRAPH_MCP_TOOLS narrowing filter", allowlistEnvName},
		{"the missing-index remedy (MCP-03)", "codegraph init"},
	}
	for _, m := range mechanisms {
		if !strings.Contains(instructions, m.anchor) {
			t.Errorf("instructions never mentions %s ...", m.mechanism, ...)
		}
	}
}
```
The new WIRE-03 tests should follow this same "derive-from-source, fail loud with the string embedded in the message" convention, but check resolvability rather than a second literal anchor — see RESEARCH.md's skeleton (`TestInstructionsSkillClaimIsResolvable`, `TestInstructionsResourcesClaimIsResolvable`).

**Budget/format guard to keep passing unmodified** (`TestInstructionsStaysWithinWireBudget`, lines 112-122) — no changes needed here, just verify the rewritten const still satisfies it.

---

### `internal/agents/instructions.go` (config, file-I/O)

**Analog:** itself — current file, full 29 lines (already read in full above).

**Byte-exact, unalterable fences** (lines 8-11):
```go
const (
	codegraphSectionStart = "<!-- CODEGRAPH_START -->"
	codegraphSectionEnd   = "<!-- CODEGRAPH_END -->"
)
```
Do not touch — cross-implementation contract with TS CodeGraph.

**Current body + stale doc comment (target of D-05 correction), lines 13-29:**
```go
// codegraphInstructionsBlock is the short marker-fenced pointer block
// install injects into the 4 of 8 agent targets that get an instructions
// file (Claude, Codex, opencode, Gemini — see the 06-RESEARCH.md
// Corrected Per-Agent Parity Table). It points agents at codegraph_explore
// / `codegraph explore` and explicitly defers full tool guidance to the
// MCP initialize response (Phase 3) — this is deliberately SHORT, not
// the old full playbook TS removed in #529/#704.
const codegraphInstructionsBlock = codegraphSectionStart + `
## CodeGraph

In repositories indexed by CodeGraph (a ` + "`.codegraph/`" + ` directory exists at the repo root), reach for it BEFORE grep/find or reading files when you need to understand or locate code:

- **MCP tool** (when available): ` + "`codegraph_explore`" + ` answers most code questions in one call — the relevant symbols' verbatim source plus the call paths between them.
- **Shell** (always works): ` + "`codegraph explore \"<symbol names or question>\"`" + ` prints the same output.

If there is no ` + "`.codegraph/`" + ` directory, skip CodeGraph entirely — indexing is the user's decision.
` + codegraphSectionEnd
```

**Recommended rewrite** (RESEARCH.md Code Examples, D-04/D-05-compliant): correct the doc comment (drop the stale "Phase 3" reference, describe what the block does post-rewrite, cross-reference this phase's own resolution) and add one skill-agnostic "Resources" bullet (safe for all 4 marker-block targets since `resources/list`/`resources/read` register unconditionally for any MCP client — RSRC-03, Phase 5):
```go
// codegraphInstructionsBlock is the short marker-fenced pointer block
// install injects into the 4 of 8 agent targets that get an instructions
// file (Claude, Codex, opencode, Gemini — see the 06-RESEARCH.md
// Corrected Per-Agent Parity Table). It points agents at codegraph_explore
// / `codegraph explore` and, generically, at resources/list for deeper
// reference — this is deliberately SHORT, not the old full playbook TS
// removed in #529/#704. It never names the installed Claude Code skill by
// path or claims one exists for the 3 targets that never received it
// (Phase 7/AGENT-01 was Claude-Code-only in v1) — see 08-RESEARCH.md
// Pitfall 1 for why the wire-level instructions const (server.go) and
// this marker block intentionally diverge on that one point.
const codegraphInstructionsBlock = codegraphSectionStart + `
## CodeGraph

In repositories indexed by CodeGraph (a ` + "`.codegraph/`" + ` directory exists at the repo root), reach for it BEFORE grep/find or reading files when you need to understand or locate code:

- **MCP tool** (when available): ` + "`codegraph_explore`" + ` answers most code questions in one call — the relevant symbols' verbatim source plus the call paths between them.
- **Shell** (always works): ` + "`codegraph explore \"<symbol names or question>\"`" + ` prints the same output.
- **Resources** (when available): call ` + "`resources/list`" + ` then ` + "`resources/read`" + ` for tool-by-tool reference docs beyond this summary.

If there is no ` + "`.codegraph/`" + ` directory, skip CodeGraph entirely — indexing is the user's decision.
` + codegraphSectionEnd
```

**No byte-pinning test exists on this const's content** — verified via `rg` across `internal/agents/*_test.go`: `registry_test.go` checks `HasPrefix`/`HasSuffix`/non-empty-between-fences/`Contains("codegraph_explore")`; `claude_test.go`/`codex_test.go`/`gemini_test.go` check only `Contains(codegraphSectionStart)` + `Contains("codegraph_explore")`; `opencode_test.go` checks only `Contains(codegraphSectionStart)`. The body is free to extend without breaking any existing test.

---

## Shared Patterns

### Wire-contract literal discipline (T-03-19)
**Source:** `internal/mcp/server.go:41-55` doc comment
**Apply to:** `instructions` const only (NOT `codegraphInstructionsBlock`, which is file-I/O not wire-protocol)
Compile-time literal, zero interpolation, ever. No `fmt.Sprintf`, no variable substitution, no per-client branching.

### Derive-from-source-not-hand-typed guard (SURF-01 precedent)
**Source:** `internal/mcp/tools_schema_drift_test.go`'s own doc comment (named incident); pattern already used by `docNamesCompanionsWithoutTheFilter` in `instructions_contract_test.go:142-157` (reads `companionNames`, never re-types tool names) and `resources_test.go`'s `ListResources`/`ReadResource` round-trip.
**Apply to:** Both new WIRE-03 guard tests — every claim of fact checked against a real source (`claudeassets.SkillMarkdown()`, a live in-memory MCP session), never a second hand-typed literal.

### Marker-fence byte-exactness
**Source:** `internal/agents/instructions.go:3-11`
**Apply to:** `codegraphInstructionsBlock` edits — `codegraphSectionStart`/`codegraphSectionEnd` themselves must never change; only the body between them.

### Fail-loud test-message convention
**Source:** `internal/mcp/instructions_contract_test.go` throughout (e.g. lines 61-66, 98-103) — every failing assertion embeds the actual string/value in the `t.Fatalf`/`t.Errorf` message so a CI failure is self-explanatory without re-running locally.
**Apply to:** New WIRE-03 tests should follow the same convention.

## No Analog Found

None — all three files being modified are edited in place; the closest and only relevant analog for each is the file's own existing content plus one cross-file pattern (`resources_test.go`'s session helper) already used elsewhere in the same package.

## Metadata

**Analog search scope:** `internal/mcp/`, `internal/agents/`, root package (`claudeassets.go`)
**Files scanned:** `internal/mcp/server.go`, `internal/mcp/instructions_contract_test.go`, `internal/mcp/resources_test.go`, `internal/agents/instructions.go`, `internal/agents/registry_test.go`, `internal/agents/claude.go`, `claudeassets.go`
**Pattern extraction date:** 2026-08-13
</content>
