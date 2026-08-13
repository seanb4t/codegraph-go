# Phase 8: Instructions & Marker-Block Rewrite - Research

**Researched:** 2026-08-13
**Domain:** In-repo wire-contract text rewrite (Go, MCP protocol, marker-fenced doc injection) — no new external dependencies
**Confidence:** HIGH (all load-bearing claims verified by reading the actual source this session; one external claim is LOW/directional only)

## Summary

This phase touches exactly two string constants and their two guard-test suites, plus a documented, mechanical wire-oracle re-freeze. Nothing about the *mechanism* is unknown — `internal/mcp/server.go`'s `instructions` const, `internal/agents/instructions.go`'s `codegraphInstructionsBlock` const, their existing test suites (`instructions_contract_test.go`, `registry_test.go` + 4 per-target `*_test.go` files), and the wire-oracle capture CLI (`test/wireoracle/cmd/wireoracle`) all already exist and were read this session. The work is: (1) rewrite `instructions` to add a generic skill/resources pointer inside the fixed 600-byte budget by trimming an existing clause, verified byte-exact below with two working candidate strings; (2) correct the stale "Phase 3" doc comment on `codegraphInstructionsBlock` and optionally extend its body with a resources pointer (skill-agnostic, per D-04); (3) add a WIRE-03 resolvability guard that derives from two already-exported source-of-truth values (`claudeassets.SkillMarkdownPath`/`SkillMarkdown()` for the skill claim, a live `ListResources`/`ReadResource` in-memory session for the resources claim) — both are exact analogs of patterns Phase 5 and Phase 7 already established and both compile cleanly with no import-cycle risk; (4) re-freeze exactly 38 of the 42 frozen wire-oracle transcripts (confirmed by grep this session, full list below) through the existing, documented, human-run capture CLI — never a hand-edit.

The one genuine open design tension — the wire `instructions` string is agent-agnostic (it ships to any MCP client, all 8 agent targets, on every session) but the skill it will name only exists for Claude Code (Phase 7, v1-scoped) — is flagged below as a Common Pitfall with two concrete resolutions, not a blocker: WIRE-03's own success-criterion text ("the named skill path is one `codegraph install` actually writes") only requires the claim to resolve against *one* real install path, which the Claude-Code path satisfies today.

**Primary recommendation:** Trim the existing `instructions` string's "optional path argument" clause (frees ~101 bytes) to make room for a new generic skill/resources clause; keep `codegraphInstructionsBlock`'s existing "MCP tool / Shell" structure intact and add one new "Resources" bullet (skill-agnostic, no path named); add two new derive-from-source tests to `internal/mcp/instructions_contract_test.go` (skill-resolvability via `claudeassets`, resources-resolvability via a live in-memory session) rather than a parallel guard file; re-freeze the 38 named transcripts via `go run ./test/wireoracle/cmd/wireoracle` (never hand-edited), one reviewed-diff pass, causes attributed in the commit message per `COVERAGE-BASELINE.md`'s own stated procedure.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| MCP `instructions` wire string (WIRE-01) | API / Backend (`internal/mcp`) | — | Compile-time literal served by the stdio MCP server process itself; no client-tier or storage-tier involvement |
| Marker-fenced `CLAUDE.md`/`AGENTS.md` block (WIRE-02) | CLI / Install tooling (`internal/agents`) | Filesystem (agent config files it upserts into) | `codegraph install` is a CLI-tier write operation targeting files the agent's own client (Browser/IDE-tier, out of this project's control) later reads |
| Skill-existence guard (WIRE-03, skill half) | API / Backend test (`internal/mcp` test importing root `claudeassets` embed) | Build/embed tier (`claudeassets.go`, `//go:embed`) | The guard reads the SAME embedded bytes `internal/agents/claude.go`'s `Install` writes — no filesystem I/O, no client tier involved |
| Resources-existence guard (WIRE-03, resources half) | API / Backend test (`internal/mcp` in-memory session) | — | Reuses Phase 5's `newTestSession`/`ListResources`/`ReadResource` in-process MCP session — no real stdio transport, no external service |
| Wire-oracle transcript re-freeze | Test / CI tooling (`test/wireoracle`) | Filesystem (`testdata/wireoracle/transcripts/*.golden`) | Golden-file regression harness; spawns the real built binary over real stdio, so it exercises the full API tier end-to-end but the artifact itself is a test fixture, not a shipped capability |

## User Constraints

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**D-01 (Wire budget vs. new content):** The `instructions` const's existing ~600-byte / single-paragraph / no-newline budget (`internal/mcp/instructions_contract_test.go`'s `instructionsMaxBytes`) is kept as-is, not raised. The current string already measures 580/600 bytes, so fitting a skill/resources pointer requires trimming or restructuring existing content, not just appending. Matches unanimous research consensus that instructions should stay under ~200-300 words and not restate what tool descriptions already convey. Reversibility: reversible — the byte cap is a repo-local test constant.

**D-02 (Anchor preservation):** The three existing pinned anchors (`TestInstructionsDescribesEveryVisibilityMechanism`'s "default"/`CODEGRAPH_MCP_TOOLS`/"codegraph init" tokens) must all still be present in the rewritten string — this phase adds a skill/resources pointer on top of, not instead of, that existing contract.

**D-03 (Resource URI naming granularity):** The rewritten `instructions` string points at the skill and resources generically — e.g. "see the codegraph skill" / "call resources/list for tool-by-tool reference docs" — and does NOT enumerate the 10 individual `codegraph://` URIs (8 per-tool + `tools-filter` + `index-state`) by name. The standard MCP pattern is client-side discovery via `resources/list`. Keeps WIRE-03's drift-guard surface narrower: the guard needs to prove the skill path and the resource capability are real, not keep 10 literal URI strings pinned inside the wire-budget-constrained instructions const itself. Reversibility: reversible — a later change is a wire-oracle re-capture, not a migration.

**D-04 (Marker-block content per agent target):** `codegraphInstructionsBlock` stays one shared constant across all 4 marker-block agent targets (Claude, Codex, opencode, Gemini) — it is rewritten to describe `codegraph_explore` / `codegraph explore` usage directly and does NOT name the installed skill by path or claim one exists. Only Claude Code received the actual SKILL.md+hooks install (Phase 7, v1-scoped); Codex/opencode/Gemini did not (AGENT-04…07, v2/deferred). A skill-agnostic shared block satisfies WIRE-03 uniformly for all 4 targets without introducing a second variant to keep in sync. Reversibility: costly — once Claude Code's own marker block starts referencing the installed skill, reverting to one shared block again means re-auditing all 4 targets' text.

**D-05 (Stale package comment):** The package comment in `internal/agents/instructions.go` describing "explicitly defers full tool guidance to the MCP initialize response (Phase 3)" is corrected as part of this rewrite — it's a stale internal-numbering reference and no longer describes what the block actually does post-rewrite. This is a code-comment accuracy fix Claude carries out directly, not a decision requiring user input.

### Claude's Discretion

- Exact wording of the generic skill/resources pointer phrase in both the `instructions` const and the marker block — within D-01/D-02/D-03/D-04's constraints (byte budget, required anchors, generic-not-enumerated, skill-agnostic-for-marker-block).
- What existing clause in the current 580-byte `instructions` string gets trimmed or restructured to make room, since D-01 keeps the budget fixed — a wording/prioritization call, not a vision decision.
- Exact mechanics of the wire-oracle re-capture and diff-attribution (success criterion 4) — follows the existing `test/wireoracle` reviewed-diff discipline already established in Phase 5.
- How WIRE-03's "every named capability is resolvable at test time" guard is implemented for the (now generic) skill/resources pointer — e.g. asserting the skill file path exists and `resources/list` returns a non-empty set, vs. some other mechanism.

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope. Six lower-relevance backlog todos were reviewed and explicitly not folded into this phase (release dry-run diff guard, post-release-verify conclusion guard, golangci-lint addition, brew tap trust docs, tap secret-distinctness test, and the wire-oracle response-ordering flake — the last of which touches the same `test/wireoracle` machinery but is a pre-existing, separately-tracked harness flake unrelated to `instructions` content).
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| WIRE-01 | The MCP `initialize` instructions string correctly defers to the skill + resources instead of the stale "Phase 3" promise in `internal/agents/instructions.go` | Byte-exact candidate rewrite provided below (Code Examples); exact current byte count (580/600) verified; doc-comment correction text drafted; import-cycle-free guard design given |
| WIRE-02 | The `codegraph install` marker block matches the rewritten instructions and promises nothing not yet shipped | Verified `codegraphInstructionsBlock`'s existing content already names no unshipped capability; verified no test in the repo pins its content byte-exactly (only structural `HasPrefix`/`HasSuffix`/`Contains("codegraph_explore")` checks across `registry_test.go` + 4 per-target test files), so content is free to extend |
| WIRE-03 | This rewrite ships only after RSRC and SKILL are verified working — never names something that doesn't exist yet | Two concrete, source-derived guard designs given (skill via `claudeassets.SkillMarkdown()`, resources via a live in-memory `ListResources`/`ReadResource` session), both grounded in patterns already proven in this repo (Phase 5's `resources_test.go`, Phase 7's `claude_skillpackage_test.go`) |
</phase_requirements>

## Standard Stack

No new external dependencies. This phase edits two existing Go string constants, two existing test files (plus possibly one new test function group), and re-freezes existing golden fixtures. All code uses only the Go standard library (`strings`, `testing`) and packages already imported elsewhere in this repository (`github.com/modelcontextprotocol/go-sdk/mcp` for the in-memory session, the root `claudeassets` package for the skill-embed source of truth). **Package Legitimacy Audit is not applicable** — no packages are installed or added.

## Architecture Patterns

### System Architecture Diagram

```
                    ┌─────────────────────────────┐
                    │   codegraph serve --mcp      │
                    │   (internal/mcp)              │
                    │                                │
  MCP client  ──initialize/discover──▶  Server        │
  (any of 8       (wire request)     │  Options{       │
   agents)                            │   Instructions:  │
                                       │   instructions } │◀── WIRE-01: rewritten
                    │                 └────────┬────────┘    const, source of the
                    │                          │              wire-oracle transcript
                    │                          ▼              byte comparison
                    │           ┌──────────────────────────┐
                    │           │ resources/list, read       │◀── live capability the
                    │           │ (registerResources, D-11)   │    "resources" clause
                    │           └──────────────────────────┘    in `instructions`
                    └─────────────────────────────┘             must resolve to (WIRE-03)


  codegraph install (internal/agents/claude.go, et al.)
        │
        ├─▶ upsertInstructionsEntry(CLAUDE.md/AGENTS.md, ...)
        │        writes codegraphInstructionsBlock          ◀── WIRE-02: rewritten
        │        (marker-fenced, byte-exact fences)              const, must "match"
        │                                                        the wire instructions'
        ├─▶ writeEmbeddedFile(SKILL.md path, claudeassets.SkillMarkdown())
        │        (Claude Code ONLY — Phase 7 v1 scope)        ◀── WIRE-03: the thing
        │                                                        "the codegraph skill"
        └─▶ (Codex/opencode/Gemini get the marker block          claim must resolve to
             but NO skill file — D-04 requires the marker
             block never claim one exists for them)

  Test-time guard (WIRE-03):
    internal/mcp/instructions_contract_test.go
        ├─▶ if instructions mentions "skill" → claudeassets.SkillMarkdown()
        │        must be non-empty (same bytes claude.go's Install writes)
        └─▶ if instructions mentions "resources" → live in-memory session,
                 ListResources() must be non-empty, ReadResource() on each
                 URI must succeed (mirrors resources_test.go, Phase 5)
```

### Recommended Project Structure

No new files/directories. All edits are in-place:
```
internal/mcp/
├── server.go                        # instructions const — rewrite (WIRE-01)
├── instructions_contract_test.go    # extend with WIRE-03 guard tests
internal/agents/
├── instructions.go                  # codegraphInstructionsBlock const + doc comment — rewrite (WIRE-02, D-05)
test/wireoracle/
├── cmd/wireoracle/main.go           # UNCHANGED — the capture CLI used to re-freeze
testdata/wireoracle/transcripts/
├── *.golden (38 of 42 files)        # re-frozen via the capture CLI, never hand-edited
```

### Pattern 1: Byte-budget-constrained wire-contract literal (existing, extend in place)

**What:** `internal/mcp/server.go`'s `instructions` const is a single Go string literal, no interpolation, single paragraph, capped by a companion test (`instructionsMaxBytes = 600` in `instructions_contract_test.go`).
**When to use:** Any time this constant is edited — it is the ONLY place the wire-contract text is authored; the test file only asserts properties of it, never redefines it.
**Example (current, verified this session, 580 bytes):**
```go
// Source: internal/mcp/server.go:56 (read this session)
const instructions = "codegraph indexes this repository's code into a call and symbol graph; try codegraph_explore first for a where-is-X or how-does-Y-work question, since it returns verbatim source plus call paths in one call. Every tool accepts an optional path argument; omitting it uses this server's own working directory. All eight tools register by default and appear automatically once an index exists, with no client restart required; an empty tool list means this repository has no index yet, so run codegraph init. Setting CODEGRAPH_MCP_TOOLS narrows the surface to the companions it names."
```

### Pattern 2: Derive-from-source-not-hand-typed guard (SURF-01 lesson, established GUARD-01/02 precedent)

**What:** Every claim of fact in a wire-facing string is checked against a real source-of-truth value, never a second hand-typed literal that can drift.
**When to use:** WIRE-03's new resolvability guard MUST follow this — read `claudeassets.SkillMarkdownPath`/`SkillMarkdown()` (the exact bytes `internal/agents/claude.go:433`'s `Install` already reads and writes) rather than re-typing a path string; read a live `resources/list`/`resources/read` round-trip rather than checking a hardcoded URI list.
**Example (Phase 5's proven resources test pattern, source of truth for the resources half of WIRE-03):**
```go
// Source: internal/mcp/resources_test.go (read this session, lines ~19-101)
result, err := session.ListResources(context.Background(), nil)
if err != nil {
    t.Fatalf("ListResources: %v", err)
}
if len(result.Resources) == 0 {
    t.Fatalf("ListResources returned no resources")
}
for _, r := range result.Resources {
    readRes, err := session.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: r.URI})
    if err != nil {
        t.Fatalf("ReadResource(%q): %v", r.URI, err)
    }
    // ... assert non-empty Text, MIMEType == "text/markdown"
}
```
**Example (skill-existence source of truth for the skill half of WIRE-03):**
```go
// Source: claudeassets.go (repo root, read this session, lines 38-52)
const SkillMarkdownPath = ".claude/skills/codegraph/SKILL.md"
func SkillMarkdown() ([]byte, error) { return FS.ReadFile(SkillMarkdownPath) }
```
`internal/mcp`'s test package does NOT currently import the root `claudeassets` package or `internal/agents` — verified this session via `rg` on both packages' import lists (no import cycle either direction: `internal/mcp` imports only `internal/gitmeta`/`internal/query`; `internal/agents` imports only the root `claudeassets` package, `internal/fsatomic`, `internal/version`). A new test file in `internal/mcp` importing `claudeassets "github.com/seanb4t/codegraph-go"` compiles cleanly.

### Pattern 3: Marker-fenced upsert with byte-exact fences, freely-editable body

**What:** `codegraphSectionStart`/`codegraphSectionEnd` (`<!-- CODEGRAPH_START/END -->`) are the ONLY byte-invariant part of the marker block — a cross-implementation contract with TS CodeGraph. The body between them has zero byte-pinning tests anywhere in the repo.
**When to use:** Confirms WIRE-02's content rewrite is unconstrained beyond "starts with fence, ends with fence, non-empty, contains `codegraph_explore`" (verified this session — see Code Examples' full test inventory below).

### Anti-Patterns to Avoid
- **Hand-editing a `.golden` transcript file:** `test/wireoracle/COVERAGE-BASELINE.md`'s own "Instruction for whoever next extends this corpus" section states this explicitly: "run the oracle's capture CLI against a freshly rebuilt binary — never hand-write a `.golden` file." The capture CLI (`test/wireoracle/cmd/wireoracle`) exists specifically to make this a reviewable, deterministic redirect (`> file`), not a manual edit.
- **A second hand-typed skill path or resource-URI list inside the new WIRE-03 guard:** would recreate exactly the "SURF-01" drift class (`tools_schema_drift_test.go`'s own doc comment names the prior incident: a hand-typed "default 5" claim drifted from the engine's actual constant for a whole phase). Always read `claudeassets.SkillMarkdownPath`/`resourceURIFor` (Phase 5's map), never re-type either.
- **Per-client conditional instructions text:** the `instructions` const's doc comment (`server.go:41-55`, read this session) states a hard constraint — "MUST stay a compile-time literal with no interpolation of any kind... a per-transcript variation here would publish the capturing host's filesystem layout into every committed wire-oracle transcript." This forecloses any design that tries to detect which agent connected and vary the skill claim per-client — see Common Pitfall 1 below for the resulting tension and its resolution.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Regenerating a wire-oracle golden transcript | A script that greps/seds the old instructions substring inside `.golden` files | `go run ./test/wireoracle/cmd/wireoracle -bin <built-binary> -fixture testdata/wireoracle/fixture -scenario <name> > testdata/wireoracle/transcripts/<name>.golden` | The capture CLI normalizes non-deterministic fields (repo paths, timestamps) via `NormalizeWithLedger` before emitting — a substring edit would miss that normalization pass and risk silently diverging from what a real capture produces |
| Proving the skill claim is real | A new hand-typed path constant duplicating `.claude/skills/codegraph/SKILL.md` | `claudeassets.SkillMarkdownPath` / `claudeassets.SkillMarkdown()` (root package, already exported, already the exact source `internal/agents/claude.go`'s `Install` reads from) | Any second literal is a drift vector the moment the skill's install path ever changes |
| Proving the resources claim is real | A hardcoded list of the 10 `codegraph://` URIs | A live `session.ListResources()`/`session.ReadResource()` round-trip against an in-memory server (Phase 5's `newTestSession` pattern) | `resourceURIFor` (the real source of truth, `internal/mcp/resources.go:28`) can already change shape without this guard needing to track it — the guard only needs "is there at least one resource, and does reading it work," which is exactly what WIRE-03's prose asks for |

**Key insight:** every "don't hand-roll" item here is really the same rule stated three ways: this phase's own success condition (WIRE-03) is that nothing it names can go stale, so the guard code itself must never introduce a second hand-typed copy of anything it is checking — every assertion must trace to the one real source (`claudeassets`, `resourceURIFor`, or a live protocol call).

## Common Pitfalls

### Pitfall 1: The skill claim's honesty scope is narrower than the string that carries it

**What goes wrong:** The `instructions` const is a single compile-time literal shipped identically to every MCP client on every session, regardless of which of the 8 agent targets is actually connected. D-03 requires it to generically name "the codegraph skill." But Phase 7 (AGENT-01, v1-scoped) only ever installs that skill for Claude Code — Codex, opencode, Gemini, Cursor, Kiro, Hermes, and Antigravity clients connecting to the exact same `codegraph serve --mcp` process would receive a claim about a skill that was never installed for them.
**Why it happens:** The wire `instructions` string and the marker-fenced block operate at different scopes — the marker block is written per-agent-target (so D-04 can correctly make it skill-agnostic for the 3 targets without a skill), but the wire string is written once, globally, with no agent-detection mechanism (and per Pattern 1/Anti-Pattern 3, must never gain one).
**How to avoid:** Two resolutions, both compatible with D-03's locked wording and WIRE-03's actual success-criterion text ("the named skill path is one `codegraph install` actually writes" — singular, not "every agent's install"):
1. **Scope the phrase explicitly** — e.g. "Claude Code users: see the codegraph skill" instead of an unscoped "see the codegraph skill" — costs a few bytes but removes the ambiguity entirely.
2. **Treat it as acceptable as locked** — WIRE-03's own text only requires resolvability against one real path, and the wire string's job (per the MCP spec's own framing, cited below) is to describe the *server's* capabilities in general, not what any specific client happens to have installed locally; a client without the skill simply doesn't act on that clause, the same way it wouldn't act on a tool it chooses not to call.
Either is planning-time discretion (already flagged as such in CONTEXT.md's Claude's Discretion list) — this research surfaces the tension so the planner makes it a deliberate choice, not an accident.
**Warning signs:** A test asserting "every one of the 8 agent targets' installed files match what `instructions` claims" would be the wrong (over-strict) guard to write — WIRE-03 does not ask for that, and building it would either fail forever (since 7 of 8 never get a skill) or force D-04's shared-block decision to be revisited.

### Pitfall 2: Trimming the wrong clause breaks an existing anchor

**What goes wrong:** `instructions_contract_test.go`'s `TestInstructionsDescribesEveryVisibilityMechanism` pins three literal substrings: `"default"`, `CODEGRAPH_MCP_TOOLS`, `"codegraph init"`. The clause most tempting to cut for space — "All eight tools register by default and appear automatically once an index exists... an empty tool list means this repository has no index yet, so run codegraph init" — is exactly the clause carrying two of the three anchors.
**Why it happens:** The safest byte savings (verified this session, see Code Examples) come from the "Every tool accepts an optional path argument; omitting it uses this server's own working directory" clause, which carries zero anchors and is the least load-bearing sentence in the string — it's easy to instead reach for the visually larger "default"/"codegraph init" clause since it looks like the bigger win, but that clause cannot be shortened past the point where both anchors survive.
**How to avoid:** Cut the path-argument clause (verified this session — safe, saves ~101 bytes, zero anchor overlap); keep the default/init clause and the `CODEGRAPH_MCP_TOOLS` clause intact or lightly reworded with anchors preserved verbatim.
**Warning signs:** `go test ./internal/mcp/... -run TestInstructionsDescribesEveryVisibilityMechanism` failing after an edit — treat this as evidence the wrong clause was cut, not as a signal to weaken the test.

### Pitfall 3: Re-freezing fewer (or more) transcripts than actually carry the string

**What goes wrong:** Assuming "the instructions string changed, re-freeze everything" (wasteful, makes the reviewed-diff pass harder to read since unrelated transcripts show a zero-byte diff) or "only re-freeze the obvious ones" (misses transcripts, leaves a stale byte-mismatch that only fails much later when `TestFrozenTranscriptsMatch` runs).
**Why it happens:** `instructions` appears in the `initialize`/`discover` result of any scenario whose transcript includes that handshake — which is most, but not all, scenarios (4 of 42 verified this session to have none: `edge-call-before-initialize`, `modern-listen-catalog-change`, `modern-meta-invalid-params`, `modern-meta-unsupported-version` — these either skip the handshake deliberately or use a `_meta`-only Modern flow that doesn't echo `instructions`).
**How to avoid:** Grep before re-capturing, not after: `rg -l "codegraph indexes this repository" testdata/wireoracle/transcripts/` returns the exact 38-file set (listed in full under Code Examples below) — re-freeze exactly those, verify the other 4 are untouched by `git diff --stat` after the pass.
**Warning signs:** `git diff --stat` after the re-capture pass showing changes to `edge-call-before-initialize.golden` or the other 3 named above — investigate before committing, since that would mean the rewritten string leaked somewhere it structurally shouldn't be (e.g. a `_meta` failure response that isn't supposed to echo server-level instructions at all).

## Code Examples

### Candidate rewrite for `instructions` (WIRE-01) — byte-verified this session

Current string (verified via direct Python `len()` this session): **580 bytes**, budget **600**, only 20 bytes of headroom — insufficient on its own for any meaningful addition, confirming D-01's premise that a clause must be cut.

**Candidate A — minimal-diff (cuts the path-argument clause only, 566 bytes, 34 bytes headroom):**
```
codegraph indexes this repository's code into a call and symbol graph; try codegraph_explore first for a where-is-X or how-does-Y-work question, since it returns verbatim source plus call paths in one call. All eight tools register by default and appear automatically once an index exists, with no client restart required; an empty tool list means this repository has no index yet, so run codegraph init. Setting CODEGRAPH_MCP_TOOLS narrows the surface to the companions it names. See the codegraph skill and call resources/list for full tool-by-tool reference docs.
```
Verified this session: contains `"default"` ✓, `CODEGRAPH_MCP_TOOLS` ✓, `"codegraph init"` ✓, no `\n`/`\r` ✓, 566 ≤ 600 ✓.

**Candidate B — more aggressive trim (also compresses the default/init clause's wording while preserving both anchors verbatim, 469 bytes, 131 bytes headroom for future growth):**
```
codegraph indexes this repository's code into a call and symbol graph; try codegraph_explore first for a where-is-X or how-does-Y-work question, since it returns verbatim source plus call paths in one call. All eight tools register by default once an index exists; an empty list means no index yet, so run codegraph init. CODEGRAPH_MCP_TOOLS narrows the default surface to named companions. The codegraph skill and resources/list carry full tool-by-tool reference docs.
```
Verified this session: contains `"default"` ✓, `CODEGRAPH_MCP_TOOLS` ✓, `"codegraph init"` ✓, no `\n`/`\r` ✓, 469 ≤ 600 ✓.

Recommend **Candidate A** for a smaller wire-oracle diff (fewer changed words = an easier reviewed-diff pass across 38 transcripts); **Candidate B** if future phases are expected to need more headroom soon.

### Candidate addition to `codegraphInstructionsBlock` (WIRE-02) — skill-agnostic, D-04-compliant

Current body (verified this session, `internal/agents/instructions.go:20-29`) already satisfies D-04's "does not name the installed skill by path or claim one exists" — no change is strictly required to the body for D-04 compliance. To also satisfy the "matches the rewritten instructions" half of WIRE-02 (both artifacts telling a consistent skill/resources story), add one resources-only bullet — safe for all 4 targets since `resources/list`/`resources/read` register unconditionally for ANY MCP client (RSRC-03, Phase 5), unlike the skill file:
```go
const codegraphInstructionsBlock = codegraphSectionStart + `
## CodeGraph

In repositories indexed by CodeGraph (a ` + "`.codegraph/`" + ` directory exists at the repo root), reach for it BEFORE grep/find or reading files when you need to understand or locate code:

- **MCP tool** (when available): ` + "`codegraph_explore`" + ` answers most code questions in one call — the relevant symbols' verbatim source plus the call paths between them.
- **Shell** (always works): ` + "`codegraph explore \"<symbol names or question>\"`" + ` prints the same output.
- **Resources** (when available): call ` + "`resources/list`" + ` then ` + "`resources/read`" + ` for tool-by-tool reference docs beyond this summary.

If there is no ` + "`.codegraph/`" + ` directory, skip CodeGraph entirely — indexing is the user's decision.
` + codegraphSectionEnd
```
No test in the repository pins this constant's content byte-exactly — verified this session via `rg` across `internal/agents/*_test.go`: `registry_test.go` checks `HasPrefix`/`HasSuffix`/non-empty-between-fences/`Contains("codegraph_explore")`; `claude_test.go`, `codex_test.go`, `gemini_test.go` each check only `Contains(codegraphSectionStart)` and `Contains("codegraph_explore")`; `opencode_test.go` checks only `Contains(codegraphSectionStart)`. Any of these additions is free to make without breaking an existing test.

### Corrected doc comment (D-05)

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
...
```

### WIRE-03 guard test skeleton (extends `internal/mcp/instructions_contract_test.go`)

```go
// Source pattern verified this session: claudeassets.go (repo root,
// lines 38-52) and internal/mcp/resources_test.go (lines 19-101).
import claudeassets "github.com/seanb4t/codegraph-go"

// TestInstructionsSkillClaimIsResolvable pins WIRE-03's skill half: if
// instructions names the codegraph skill, that name must resolve to
// content codegraph install actually ships — derived from
// claudeassets.SkillMarkdown(), the exact same source
// internal/agents/claude.go's Install already reads, never a second
// hand-typed path.
func TestInstructionsSkillClaimIsResolvable(t *testing.T) {
	if !strings.Contains(instructions, "skill") {
		t.Skip("instructions does not name a skill; nothing to resolve")
	}
	content, err := claudeassets.SkillMarkdown()
	if err != nil || len(strings.TrimSpace(string(content))) == 0 {
		t.Fatalf("instructions names the codegraph skill, but claudeassets.SkillMarkdown() (%s, the exact bytes codegraph install writes) is missing or empty: %v",
			claudeassets.SkillMarkdownPath, err)
	}
}

// TestInstructionsResourcesClaimIsResolvable pins WIRE-03's resources
// half: if instructions names resources/list, a live in-memory session
// must actually be able to list and read at least one.
func TestInstructionsResourcesClaimIsResolvable(t *testing.T) {
	if !strings.Contains(instructions, "resources") {
		t.Skip("instructions does not name resources; nothing to resolve")
	}
	// build server, newTestSession(t, s) (existing helper, server_test.go),
	// session.ListResources — assert non-empty — then ReadResource on the
	// first URI, assert non-empty Text (mirrors resources_test.go exactly).
}
```

### Wire-oracle re-capture procedure (success criterion 4)

```bash
# 1. Build the real binary once
go build -o /tmp/codegraph-recapture ./cmd/codegraph

# 2. For each of the 38 affected scenarios (exact list below), capture and
#    redirect — NEVER hand-edit the .golden file (COVERAGE-BASELINE.md's
#    own stated rule):
go run ./test/wireoracle/cmd/wireoracle \
  -bin /tmp/codegraph-recapture \
  -fixture testdata/wireoracle/fixture \
  -scenario call-callees \
  > testdata/wireoracle/transcripts/call-callees.golden
# ... repeat per scenario name below ...

# 3. Review: git diff --stat testdata/wireoracle/transcripts/ — confirm
#    EXACTLY these 38 files changed, and the other 4 (below) did not.
# 4. Attribute every changed line's cause in the commit message per
#    COVERAGE-BASELINE.md's documented reviewed-diff mechanism (D-06) —
#    no separate ledger file, no sign-off step.
# 5. go test ./test/wireoracle/... — TestFrozenTranscriptsMatch must pass.
```

**Full list of the 38 scenarios carrying the `instructions` string** (verified this session via `rg -l "codegraph indexes this repository" testdata/wireoracle/transcripts/`):
`call-callees`, `call-callers`, `call-files`, `call-impact`, `call-node`, `call-search`, `call-status`, `error-confinement-reject`, `error-malformed-args`, `error-unknown-method`, `error-unknown-tool`, `handshake-explore`, `index-appears-mid-session`, `legacy-2024-11-05`, `legacy-2025-03-26`, `legacy-2025-06-18`, `legacy-2025-11-25`, `legacy-omitted-version`, `legacy-unsupported-2026-07-28`, `modern-discover-explore`, `resources-list`, `resources-list-no-index`, `resources-read-callees`, `resources-read-callers`, `resources-read-explore`, `resources-read-files`, `resources-read-impact`, `resources-read-index-state`, `resources-read-node`, `resources-read-search`, `resources-read-status`, `resources-read-tools-filter`, `resources-read-unknown`, `toolslist-default`, `toolslist-filter-empty`, `toolslist-narrowed`, `toolslist-no-index`, `toolslist-repeat`.

**The 4 scenarios verified NOT to carry it** (must show zero diff after the pass): `edge-call-before-initialize`, `modern-listen-catalog-change`, `modern-meta-invalid-params`, `modern-meta-unsupported-version`.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| `instructions` says nothing about the skill or resources at all | Points generically at both, within budget | This phase (WIRE-01) | Closes the origin incident (2026-08-08 misdirected debug session) that this milestone exists to fix |
| `codegraphInstructionsBlock`'s doc comment claims a deferral to "the MCP initialize response (Phase 3)" | Comment corrected to describe what the block actually does, referencing this phase's own resolution | This phase (D-05) | Removes the last stale forward-reference in the codebase to the old, superseded Phase 3 (v0.3.0's MCP-spec-currency phase, unrelated to this milestone's phase numbering) |

**Deprecated/outdated:** None — no library or protocol version changed in this phase; this is a pure content/prose correctness fix.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|----------------|
| A1 | The single external research query run this session (MCP server instructions best practices) returned results consistent with, but not independently cross-checked against, the sources CONTEXT.md already cited (MCP blog, OpenAI guidance, niteagent checklist) — tagged LOW confidence per this session's `classify-confidence --provider websearch` call, since it is a single non-authoritative aggregation, not an official-docs fetch. | Summary, Sources | Low — the guidance (concise, avoid duplicating tool descriptions, focus on cross-tool context) only reinforces an already-locked decision (D-01); it does not introduce a new claim the plan depends on. |

**All other claims in this research are `[VERIFIED: <path>:<lines>]`** — every source-of-truth value (byte counts, anchor strings, test assertions, embed paths, scenario names) was read directly from the repository this session, not recalled from training data or an external source.

## Open Questions

1. **Should the wire `instructions` string scope its skill claim to Claude Code explicitly, or leave it agent-agnostic per D-03's literal wording?**
   - What we know: D-03 is locked as "generic" phrasing; WIRE-03's actual success-criterion text only requires resolvability against one real install path, which Claude Code's path satisfies.
   - What's unclear: whether an unscoped claim reaching non-Claude clients is itself a (smaller-scale) instance of the exact defect this phase exists to close, or an acceptable simplification given the wire string's job is describing server capability, not per-client reality.
   - Recommendation: surfaced as Common Pitfall 1 with two concrete resolutions; leave the final call to the planner/discuss-phase, since CONTEXT.md already lists "exact wording" as Claude's Discretion.

## Environment Availability

Skipped — this phase has no external tool/service dependencies beyond what the repository's existing `go test`/`go build`/`go run` toolchain already provides, all of which are demonstrably available (this session ran `go build`-adjacent greps and file reads against a working checkout with no missing-tool errors).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go's built-in `testing` package (no external test framework) |
| Config file | none — `Taskfile.yml` defines the task wrappers |
| Quick run command | `go test ./internal/mcp/... ./internal/agents/...` |
| Full suite command | `task test:unit` (excludes daemon/wireoracle) then `task test:wireoracle` for the transcript comparison |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|---------------------|--------------|
| WIRE-01 | `instructions` names skill/resources, stays within budget, keeps all 3 anchors | unit | `go test ./internal/mcp/... -run TestInstructions -v` | ✅ (extend existing file) |
| WIRE-02 | `codegraphInstructionsBlock` starts/ends with fences, non-empty, mentions `codegraph_explore` | unit | `go test ./internal/agents/... -run TestCodegraphInstructionsBlock -v` (name illustrative — actual test is inside `registry_test.go`, no rename needed) | ✅ (existing) |
| WIRE-03 | Named skill/resources claims resolve at test time | unit | `go test ./internal/mcp/... -run TestInstructionsSkillClaimIsResolvable -run TestInstructionsResourcesClaimIsResolvable -v` | ❌ Wave 0 — new test functions to add |
| WIRE-01/02/04 (transcript byte-identity) | 38 re-frozen transcripts match rewritten `instructions` | integration | `go test ./test/wireoracle/...` | ✅ (existing harness; fixtures need re-freezing, not the test code) |

### Sampling Rate
- **Per task commit:** `go test ./internal/mcp/... ./internal/agents/...`
- **Per wave merge:** `task test:unit && task test:wireoracle`
- **Phase gate:** Full suite green (`task test`) before `/gsd-verify-work`, since a byte-budget or anchor regression here fails silently otherwise (JSON-encoded into every transcript, no runtime symptom).

### Wave 0 Gaps
- [ ] `internal/mcp/instructions_contract_test.go` — add `TestInstructionsSkillClaimIsResolvable` and `TestInstructionsResourcesClaimIsResolvable` (WIRE-03) — skeleton given above.
- [ ] No new fixtures or framework install needed — everything reuses existing helpers (`newTestSession`, `claudeassets`).

*(No conftest/fixture gap beyond the two new test functions — all supporting infrastructure already exists.)*

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|----------------|---------|-------------------|
| V5 Input Validation | Marginal | N/A — no new input parsing; `instructions`/`codegraphInstructionsBlock` remain compile-time literals with zero interpolation (T-03-19, `server.go:41-55`'s existing doc comment, unchanged constraint this phase must not violate) |
| V6 Cryptography | No | Not applicable — no crypto/secrets involved |
| V2/V3/V4 (Auth/Session/Access) | No | Not applicable — this phase touches only static prose content, no auth surface |

### Known Threat Patterns for this phase's stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|-----------------------|
| Wire-contract text leaking host-specific data (repo path, hostname) into a value re-shared with every client and committed into golden transcripts | Information Disclosure | Already enforced structurally: `instructions` MUST stay a compile-time literal with zero interpolation (existing `server.go` doc comment, T-03-19) — this phase's rewrite must preserve that property; do not introduce any `fmt.Sprintf`/variable substitution into either const |
| A promise (skill/resource claim) drifting from shipped reality after this phase, reintroducing the exact defect being fixed | Repudiation (of the fix) / correctness regression | WIRE-03's derive-from-source guard tests (skeleton above) — any future rename/removal of the skill file or a resource URI turns these tests red, per the SURF-01 precedent already established in this codebase |

## Sources

### Primary (HIGH confidence — read directly this session)
- `internal/mcp/server.go` (lines 1-80) — `instructions` const, its doc comment's hard constraints (compile-time literal, no interpolation, ~600 byte/no-newline budget), `BuildServer`'s `Instructions:` wiring
- `internal/mcp/instructions_contract_test.go` (full file, 224 lines) — all 5 existing test functions, `instructionsMaxBytes = 600`, the 3 pinned anchor literals, the README-gate checker
- `internal/agents/instructions.go` (full file, 29 lines) — marker fences, `codegraphInstructionsBlock` body, the stale "Phase 3" doc comment (D-05's target)
- `internal/agents/claude.go` (lines 1-165) — `claudeSkillDirPath`/`claudeSkillFilePath` (`.claude/skills/codegraph/SKILL.md` for local, `<home>/.claude/skills/codegraph/SKILL.md` for global), `instructionsBody()`
- `claudeassets.go` (repo root, full file) — `FS` embed.FS, `SkillMarkdownPath`, `SkillMarkdown()` — the WIRE-03 skill guard's source of truth
- `internal/mcp/resources.go` (lines 1-60) — `resourceURIFor` map (10 entries), `resourcesFS` embed — confirms the resources capability's real source of truth
- `internal/mcp/resources_test.go` (relevant line ranges) — `ListResources`/`ReadResource` in-memory session pattern, the WIRE-03 resources guard's template
- `internal/agents/registry_test.go`, `claude_test.go`, `codex_test.go`, `gemini_test.go`, `opencode_test.go` — confirmed no byte-exact pinning of `codegraphInstructionsBlock` content exists anywhere
- `test/wireoracle/COVERAGE-BASELINE.md` (lines 220-247) — the explicit "never hand-write a `.golden` file, run the capture CLI" procedure, the reviewed-diff/attribution discipline (D-06)
- `test/wireoracle/cmd/wireoracle/main.go` (full file, 59 lines) — the exact capture CLI invocation shape (`-bin`, `-fixture`, `-scenario` flags, stdout redirect convention)
- `testdata/wireoracle/transcripts/*.golden` — direct `rg` count confirming exactly 38 of 42 files carry the current `instructions` string, full name list captured
- `README.md` (lines 95-134) — confirmed no other doc file references `instructions`/marker-block content or the stale Phase 3 claim (grep across `README.md`, `docs/*.md` found zero additional hits beyond `internal/agents/instructions.go` itself)
- `.planning/phases/07-.../07-PATTERNS.md`, `.planning/phases/05-.../05-PATTERNS.md` — this session's uncommitted scratch pattern maps from the two immediately-preceding phases, confirming the `claudeassets`/`resources.go` conventions this phase's guard must reuse

### Secondary (MEDIUM confidence)
- CONTEXT.md's own cited sources (not independently re-fetched this session, already vetted during discuss-phase): MCP Blog "Server Instructions" (blog.modelcontextprotocol.io, 2025-11-03), OpenAI MCP server guidance, niteagent's authoring checklist (2026-06-10)

### Tertiary (LOW confidence)
- One WebSearch query run this session ("MCP server initialize instructions field best practices length and content") — aggregated, non-authoritative summary; consistent with but not independently verified against the Secondary sources above; classified LOW by this session's `classify-confidence --provider websearch` call and cached accordingly (research-store key `7a1607960c971325890f823334fda51b558f770386fdab4948c2f23e6d1b8b28`)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependencies, pure in-repo edit
- Architecture: HIGH — every file, function, and test read directly this session
- Pitfalls: HIGH for Pitfalls 2-3 (directly observed via `rg`/byte counts); MEDIUM for Pitfall 1 (a design-tension judgment call, not a verifiable fact)

**Research date:** 2026-08-13
**Valid until:** Effectively indefinite for the architectural facts (this is the terminal phase of the milestone, nothing downstream depends on this research staying current beyond phase execution); the byte-count candidates (Code Examples) are valid only until `instructions`'s current 580-byte baseline changes — re-verify byte counts if any earlier phase's work lands on `server.go` before this phase executes.
