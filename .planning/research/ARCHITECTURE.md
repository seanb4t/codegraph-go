# Architecture Research

**Domain:** Integration architecture for v0.10.0 (Agent Onboarding Skill & MCP Resources) inside the existing codegraph-go codebase
**Researched:** 2026-08-12
**Confidence:** HIGH (grounded directly in the current source tree, the frozen wire-oracle harness, the live archtest suite, and `modelcontextprotocol/go-sdk@v1.7.0`'s actual `Resource`/`AddResource` API — not general MCP-spec prose)

This is not an ecosystem survey — it is a design for three additions to an already-shipping, heavily guard-railed Go codebase. Every recommendation below is anchored to a file that exists today and a test that would need to pass or be extended.

## System Overview

```
┌───────────────────────────────────────────────────────────────────────┐
│ codegraph install / uninstall  (internal/cli/install.go)              │
│   printAgentResults() → for each AgentTarget: t.Install(loc, opts)    │
├───────────────────────────────────────────────────────────────────────┤
│ internal/agents  (AgentTarget registry, per-target files)             │
│                                                                        │
│  claude.go / cursor.go / codex.go / opencode.go / gemini.go /         │
│  hermes.go / antigravity.go / kiro.go                                 │
│    ├─ writeMcpEntry()            — MCP server config JSON/TOML/YAML   │
│    ├─ upsertInstructionsEntry()  — marker-fenced text block           │  ◄── existing, unchanged shape
│    │    (instructions.go: codegraphInstructionsBlock)                 │
│    └─ [NEW] writeSkillFiles()    — skill/hooks directory tree         │  ◄── new, this milestone
│         (internal/agents/skillfiles.go, embed.FS)                     │
│                                                                        │
│  shared.go: atomicWriteFile (→ internal/fsatomic), recordFile/        │
│  recordAction, replaceOrAppendMarkedSection — ALL THREE writers above │
│  reuse this same idempotent-compare-then-write discipline (D-07)      │
├───────────────────────────────────────────────────────────────────────┤
│ internal/mcp  (stdio MCP server; imports ONLY internal/query,         │
│                internal/gitmeta — never internal/graphstore)          │
│                                                                        │
│  server.go: BuildServer() — instructions const, tool registration,    │
│             catalogMu/hasCatalog live re-check, session-line middleware│
│  tools.go:  8 tools, openEngine() read seam, confineToRepoRoot()      │
│  [NEW] resources.go: registerResources() — static reference content   │  ◄── new, this milestone
│  [NEW] resourcedocs/*.md — go:embed'd source markdown                 │
├───────────────────────────────────────────────────────────────────────┤
│ Guardrails that gate every change above                               │
│                                                                        │
│  internal/graphstore/archtest  — pebble/v2 import confined            │
│  internal/cli/archtest         — MCP SDK import confined to internal/mcp│
│  internal/mcp/instructions_contract_test.go — instructions wire-string│
│  internal/mcp/tools_schema_drift_test.go    — numeric claims ↔ constants│
│  test/wireoracle (+ testdata/wireoracle/transcripts/*.golden)         │
│                    — byte-frozen initialize/tools/list/... responses  │
└───────────────────────────────────────────────────────────────────────┘
```

### Component Responsibilities

| Component | Responsibility | Owns / touches |
|-----------|-----------------|-----------------|
| `internal/mcp` (existing) | stdio MCP server: tools today, resources after this milestone | `server.go`, `tools.go`; imports `internal/query` + `internal/gitmeta` only |
| `internal/mcp/resources.go` (new) | Registers `resources/list` + `resources/read` against `*mcp.Server`, serves embedded markdown verbatim | Same package as `tools.go` — no new import boundary |
| `internal/mcp/resourcedocs/` (new) | Hand-authored reference markdown, source of truth for resource bodies | `go:embed`'d by `resources.go` |
| `internal/agents` (existing) | Per-target config + marker-block writers, `AgentTarget` registry | `types.go`, `registry.go`, `shared.go`, 8 per-target files |
| `internal/agents/skillfiles.go` (new) | Shared embed-and-write helper for the skill/hooks package | Called from whichever per-target `Install()`/`Uninstall()` opts in |
| `internal/agents/skillfiles/<target>/` (new) | Hand-authored SKILL.md + hooks.json + hook scripts, one embedded tree per supporting target | `go:embed`'d by `skillfiles.go` |
| `internal/cli/install.go` (existing) | Drives the per-target loop; **unchanged** by this milestone | `newInstallCmd`, `printAgentResults` |
| `test/wireoracle` + `testdata/wireoracle/transcripts/` | Frozen byte-level regression oracle for the stdio wire protocol | `scenarios.go`, `*.golden` files, standalone capture tool at `test/wireoracle/cmd/wireoracle` |

## Q1 — Where Resources lives, and what it risks

### Placement

`internal/mcp/resources.go`, a sibling of `tools.go` in the **same package**. This is not a new package boundary decision — it is deliberately the same shape `tools.go` already uses: `exploreTool()`/`companionTool()` → `codegraphResource(name)`; `registerTools(s, ...)` → `registerResources(s, ...)`; `mcp.AddTool(s, tool, handler)` → `s.AddResource(resource, handler)` (`(*mcp.Server).AddResource(r *mcp.Resource, h mcp.ResourceHandler)`, confirmed against the vendored `go-sdk@v1.7.0` source — `mcp/server.go:577`).

Registration call site: inside `BuildServer` in `server.go`, called **unconditionally**, not inside the `if hasIndex { ... }` block that gates `registerTools`. Reference content (what a tool is for, what `CODEGRAPH_MCP_TOOLS` does, the index-state precondition) is useful to an agent regardless of whether `.codegraph/` currently resolves — arguably *most* useful to an agent that just got zero tools and needs to understand why. This also sidesteps a whole class of new complexity: `registerTools`/`unregisterTools`/`recheckCatalog`/`catalogMu`/`hasCatalog`/the atomic `toolCount` exist because tool visibility is **repo-state-dependent** (SPEC-05). Static resource content is not, so it needs none of that re-check machinery, no new mutex, and no new atomic counter.

### Capability advertisement — a real asymmetry with Tools, in your favor

`BuildServer` had to set `Capabilities.Tools` **explicitly and unconditionally** (D-11) because `go-sdk`'s own `capabilities()` only sets `caps.Tools` when `HasTools || tools.len() > 0` — and on the `hasIndex=false` path zero tools are ever registered, so the key would silently vanish. Resources do not have this trap **for this milestone's shape**: `go-sdk`'s `capabilities()` sets `caps.Resources` whenever `s.opts.HasResources || s.resources.len() > 0` (`mcp/server.go:645-648`), and since `registerResources` is called unconditionally at construction time, `s.resources.len() > 0` is true from the first line of every session — `caps.Resources` appears by construction, no explicit `ServerOptions.Capabilities.Resources` needed (though setting it explicitly for symmetry with the Tools block is harmless and arguably better self-documentation).

### Wire-oracle risk — real, and it is the one piece of this milestone that MUST be sequenced, not incidental

Every scenario in `test/wireoracle/scenarios.go` (bar the deliberately one-way `mark3labs`-era legacy baselines) opens with an `initialize` request (`initializeRequest`/`initializeRequestWithVersion`), and the frozen `.golden` transcripts under `testdata/wireoracle/transcripts/` capture that response **byte-for-byte, including the `capabilities` object**. The moment `registerResources` starts running inside `BuildServer`, the `capabilities` object in **every current-server transcript** gains a `"resources"` key it did not have before. `TestFrozenTranscriptsMatch` (the byte-comparison oracle) will go red across essentially the entire transcript set — deliberately, not as a regression.

This is not a reason to avoid the change; it is a reason to sequence it as **one deliberate re-capture commit**, using the existing sanctioned mechanism:
- `scenarios.go` gains new scenarios for `resources/list` and `resources/read` (both a valid URI and an unknown-URI error shape — mirroring how `tools/list` already has "four variants" for the narrowing filter).
- The **standalone capture tool** (`test/wireoracle/cmd/wireoracle`) re-captures the full set against the real modified binary — never hand-edit a `.golden` file, and never let the SDK's own client validate itself as the oracle (VRFY-01/VRFY-04's standing rule, unaffected by this change since the capture tool remains SDK-independent).
- `MUTATION-PROOF.md`'s discipline (`git status --porcelain` checked before/after) applies unchanged — review the capabilities-object diff across the whole re-capture as one reviewable unit, not scattered across unrelated commits.
- The two **genuinely one-way, unrecapturable** transcripts (captured before `mark3labs/mcp-go` left `go.mod`) are historical baselines of a server that no longer exists in this tree; they are not reachable by, and not affected by, this change — do not touch them.

Net: expect on the order of two dozen `.golden` files to shift in one commit. That is the correct, expected footprint of adding a server capability — not a sign something went wrong.

### `internal/query`-only import boundary — verified, not merely assumed

The actual enforcement mechanism for "no package outside `internal/graphstore` may import Pebble directly" is `internal/graphstore/archtest/import_graph_test.go`'s `TestNoPackageBypassesGraphStore`: a `go/packages`-based whole-module scan asserting every importer of `github.com/cockroachdb/pebble/v2` has an import path under `internal/graphstore`. `internal/mcp` satisfies this today not because of a dedicated `internal/mcp`-scoped archtest (none exists — the closest package-boundary archtests are `internal/graphstore/archtest` for Pebble and `internal/cli/archtest/mcp_sdk_confinement_test.go` for "internal/cli must not import the MCP SDK directly"), but simply because `server.go`/`tools.go` import `internal/query` and `internal/gitmeta`, never `internal/graphstore`.

`resources.go` preserves this **by construction, with zero new import risk**, provided resource content stays static/embedded rather than derived from a live graph query: `go:embed` + `internal/mcp`'s existing imports are sufficient; no new dependency is needed at all. If a future resource needs *dynamic* content (e.g., "current index stats" as a resource rather than the `codegraph_status` tool), it should route through the exact same `openEngine`/`query.Engine` seam `tools.go` already uses — never a direct `internal/graphstore` import — which keeps `TestNoPackageBypassesGraphStore` green with no changes to that test. This milestone's stated scope (tool-by-tool docs, `CODEGRAPH_MCP_TOOLS` semantics, index-state preconditions) is static prose, so this concern is real but not immediately load-bearing — flag it for whoever writes the first dynamic resource later.

`internal/cli/archtest/mcp_sdk_confinement_test.go` (SDK-02) is untouched by any of this: it only asserts `internal/cli` itself never imports an MCP SDK package directly, which remains true — `internal/cli/serve.go` keeps bootstrapping through `mcp.NewStdioServer`/`mcp.Server`, and Resources living inside `internal/mcp` changes nothing about that seam.

### The `instructions` wire-string — the other thing this touches, and the order matters

`instructions` (`server.go:56`) is a compile-time literal, ≤600 bytes, no newlines, JSON-encoded into every transcript — already true today and unaffected mechanically by adding Resources, EXCEPT that the milestone's stated goal is for `instructions` to *point at* the new resource content instead of the stale "the marker block defers full tool guidance to the MCP initialize response (Phase 3)" promise `internal/agents/instructions.go:17-18` currently makes and Phase 3 never fulfilled. Build Resources **before** rewriting `instructions` to reference it — naming a resource that does not yet exist would recreate the exact "wire-contract claim drifts from behavior with no gate comparing them" failure this milestone exists to close (the `mcp-server-one-tool-only` incident). `internal/mcp/instructions_contract_test.go`'s existing three-anchor pattern (`TestInstructionsDescribesEveryVisibilityMechanism`: `"default"`, `CODEGRAPH_MCP_TOOLS`, `"codegraph init"`) is the template for a fourth anchor once a resource URI exists to name.

## Q2 — The AgentTarget seam for skill+hooks distribution

### Recommendation: no new interface method, no new registry — a shared helper called opt-in from each per-target file

`AgentTarget` (`types.go`) is a single interface all 8 targets implement uniformly (`ID`, `DisplayName`, `SupportsLocation`, `Detect`, `Install`, `Uninstall`, `DescribePaths`). Three shapes were in play; here is why the third wins:

1. **New interface method pair** (`InstallSkill`/`UninstallSkill` alongside `Install`/`Uninstall`) — mirrors the existing `SupportsLocation`/`Install`/`Uninstall` triplet shape and would work, but forces every one of the 8 per-target files to add a stub (`return WriteResult{}` for the 5+ agents with no skill concept today), and produces **two separate `WriteResult`s per target per install run** — `printAgentResults` (`install.go`) would need a second call site, doubling the reporting output and the idempotency/error-surfacing bookkeeping for something that is, from the user's perspective, one operation ("configure this agent for codegraph").
2. **A new parallel registry** (`internal/skills`, its own `SkillTarget` interface, its own `init()`-based self-registration) — rejected. It duplicates `Detect`/`Location`/idempotency machinery `internal/agents` already owns for what is fundamentally the same "make agent X aware of codegraph" operation, and splits one target's install status across two independently-iterated registries with no shared reporting loop. `registry.go`'s own doc comment states the design intent directly: "no agent-specific logic... every quirk lives in the target's own file" — a second registry reintroduces exactly the cross-cutting logic this package was built to avoid.
3. **Reuse the marker-fenced injection machinery unmodified** (grow `codegraphInstructionsBlock`'s body) — works for updating the *pointer text*, but categorically cannot carry the skill/hooks payload itself: `codegraphInstructionsBlock` is deliberately short prose injected into an existing agent-owned file (`CLAUDE.md`/`AGENTS.md`/etc.), and **hooks cannot be expressed as prose inside an instructions file at all** — `SessionStart`/`PreToolUse` hooks are registered via a dedicated `hooks.json`/settings mechanism, a structurally different artifact than a marker-fenced text block.

**What actually fits**: add `internal/agents/skillfiles.go`, a small shared helper (`writeSkillFiles(dir string, files embed.FS) ([]FileResult, []error)` or similar) that walks an embedded directory tree and, for each file, does exactly what `shared.go`'s existing writers already do — read current content if present, compare, and call `atomicWriteFile` (→ `internal/fsatomic.WriteFile`) only on a diff, classifying each file via `recordFile`/`recordAction` into the same `FileAction` vocabulary (`ActionCreated`/`ActionUpdated`/`ActionUnchanged`/`ActionRemoved`) `WriteResult.Files` already uses. Each supporting target's own `Install()`/`Uninstall()`/`DescribePaths()` calls this helper as one more step alongside its existing `writeMcpEntry`/`upsertInstructionsEntry` calls, folding the skill-file actions into the **same single `WriteResult`** the target already returns. `install.go`'s `printAgentResults` loop, its idempotency status line (`installStatus`), and its error-surfacing (`CR-01`) all keep working with **zero changes**, because a skill-file write is just more entries in `result.Files`.

Concretely, only the per-target files that choose to support skills change (a per-target file is where "quirks live" by this package's own convention) — non-supporting targets are **literally untouched**, which is a stronger form of "no interface change" than a stub method would give: there is nothing on `AgentTarget` a non-supporting target has to implement at all.

### Which targets can plausibly support this — a structural fact, not a scope decision

Skill directories with a `SKILL.md` index already exist as a convention across more of the roster than "Claude Code only": this environment's own project context lists `.claude/skills/`, `.agents/skills/`, `.cursor/skills/`, `.github/skills/`, and `.codex/skills/` as recognized skill locations — i.e. **Claude, Cursor, and Codex CLI** (three of codegraph-go's 8 roster targets) already have an established `<agent-dir>/skills/<name>/SKILL.md` shape to write into, not just Claude. Hooks are a narrower story: `SessionStart`/`PreToolUse`-style event hooks are a Claude Code concept as of today's research; the other agents' equivalents (if any) were not verified here and should not be assumed. This means the skill body (`SKILL.md`) and the hooks package (`hooks.json` + scripts) are reasonably **decoupled deliverables** inside the same embedded tree — a target can pick up the skill file without the hooks, but not vice versa. Scope which targets ship what in planning, not architecture; the seam above supports either choice without changing shape.

### Distribution mechanism: `go:embed` + write-at-install, matching an existing pattern exactly

The milestone's open question ("write into the target's skills directory, versioned with the binary and refreshed by `upgrade`" vs. "ship in-repo for manual install") has a precedent already shipping in this exact package: `codegraphInstructionsBlock` (`instructions.go`) **is already** a compile-time Go string literal embedded in the binary, written out by `Install()` via `upsertInstructionsEntry`. The skill/hooks package is the identical pattern scaled from one embedded string to an embedded directory tree (`go:embed skillfiles/claude` → `embed.FS`), written via the shared helper above instead of a single-file marker splice. This gets "versioned with the binary, refreshed by `codegraph upgrade`" for free — `upgrade` already swaps the binary atomically; a subsequent `install` (or a `postinstall`-style rerun the maintainer may choose to wire into `upgrade`) writes whatever skill content the *new* binary embeds, using the same idempotent-compare-then-write discipline that already makes re-running `install` a byte-level no-op (D-07). Shipping in-repo for manual install is the lower-engineering-cost alternative but reproduces the exact "install's output still defers to something thin/external" complaint the motivating todo raises about the current marker block — recommend against it for that reason, not for a technical one.

### Location scope

`AgentTarget`'s existing `Location` (`global`/`local`) maps naturally onto skill install scope the same way it already does for MCP config (`~/.claude.json` vs project `.mcp.json`): a global skill write goes to `~/.claude/skills/codegraph/`, a local one to `.claude/skills/codegraph/` (and analogously for Cursor/Codex if scoped in) — no new concept needed, reuse `Location` as-is. `DescribePaths(loc)` for each supporting target must be extended to list the new skill directory's file paths, since that method's contract ("every config/instructions file path this target reads or writes at loc") already exists precisely to keep this kind of addition from becoming an undocumented side effect.

## Q3 — Resource content data flow and drift prevention

### Recommendation: hand-authored Markdown, served verbatim, gated by a new drift test modeled on two that already exist in this codebase

Two existing tests are the direct precedent for how to prevent this milestone's stated risk ("gates that cannot fire" / "claims that drift from behavior with no gate comparing them"):

- **`internal/mcp/tools_schema_drift_test.go`**'s `TestMCPToolSchemaNumericClaimsMatchEngineConstants` — scans tool descriptions for a `(default|max) \d+` pattern (`numericClaimRe`) and requires every match to be pinned, by name, to an `internal/query` constant in a `map[string]string` (`engineConstantFor`); an unpinned numeric claim fails the build. This is literally SURF-01's fix, generalized into a reusable shape.
- **`internal/mcp/instructions_contract_test.go`**'s `docNamesCompanionsWithoutTheFilter` — a cross-document consistency checker (applied to `README.md` today) asserting that any document naming filterable companion tools also names `CODEGRAPH_MCP_TOOLS`, proven non-vacuous by its own dedicated boundary-case test (`TestREADMEGateCheckerIsNotVacuous`).

Both precedents point to the same answer for resource content: **hand-authored Markdown is fine for prose** (matching the codebase's existing preference for literal, reviewable strings over generated ones — `instructions` itself is hand-written, not templated), **but any factual claim inside it — a tool count, a flag default, an env var name, a resource URI — must be either (a) mechanically derived from the same source `tools.go`/`server.go` already treat as ground truth, or (b) covered by a drift test in the same shape as the two above.** Full codegen (rendering resource Markdown from a shared template that also drives `instructions`/README) is not necessary to get this property and would add real complexity (a build step, a codegen output to review) for content that changes rarely; a **generated-and-gated hybrid at the claim level**, not the document level, matches how this codebase already solved the identical problem twice.

### Concrete shape

- `internal/mcp/resourcedocs/*.md` — one file per resource (e.g. `tools.md`, `codegraph_mcp_tools.md`, `index-state.md`), hand-authored, `go:embed`'d by `resources.go` in the same package.
- `resources.go` derives the **resource catalog itself** (names, URIs, one-line descriptions surfaced in `resources/list`) from the exact same source `tools.go` already treats as authoritative — `companionNames`, `allToolNames()` — rather than a hand-typed parallel list, exactly how `registerTools`/`unregisterTools`/`allToolNames()` already share one source of truth so registration and de-registration can never drift apart (`server.go`'s own comment on `allToolNames()` makes this the explicit precedent to copy).
- `internal/mcp/resources_contract_test.go` (new) — the drift guard, structured as two checks mirroring the two precedents above:
  1. **Numeric/factual-claim pinning**: reuse (or lightly generalize) `numericClaimRe` against every embedded resource doc, requiring each match to resolve against the same constant map `tools_schema_drift_test.go` already builds from `internal/query`'s source via `go/parser` — one source of numeric truth for tools *and* resources, not two.
  2. **Tool-name/mechanism cross-check**: apply `docNamesCompanionsWithoutTheFilter`-shaped logic to the resource docs the same way `TestREADMEDocumentsToolVisibilityGate` already applies it to `README.md` — a resource doc naming filterable companion tools must also name `CODEGRAPH_MCP_TOOLS`, and (new) a resource doc naming a tool must name one that actually exists in `allToolNames()`, so a renamed or removed tool fails this test instead of silently going stale in a resource nobody re-reads.
- Because these checks operate on the **same anchor/constant vocabulary** `instructions_contract_test.go` and `tools_schema_drift_test.go` already use, extending `instructions_contract_test.go` with a fourth "names the resource mechanism" anchor (Q1, above) and adding `resources_contract_test.go` are naturally sequenced together — write them as one drift-guard pass, not two unrelated patches.

### Why not derive resources from README.md directly

`go:embed` patterns cannot reference a path outside the embedding file's own package directory tree (no `..`, no absolute paths) — `README.md` lives at the repo root, `internal/mcp` cannot `go:embed ../../README.md`. (`instructions_contract_test.go` already works around this for its *own* test-time read of `README.md` via a plain `os.ReadFile("../../README.md")`, which is legal for a `_test.go` file's `os.ReadFile` call but not for a production `go:embed` directive — a load-bearing distinction: test code can read arbitrary repo-relative paths at test time, but `go:embed` is a compile-time directive scoped to the package's own tree.) This forecloses "embed README.md verbatim as a resource" as an option; it does not foreclose keeping README.md and the resource docs *consistent* via the drift-test approach above, which needs no embed-time file-system reach-through at all — both `README.md` (via `os.ReadFile` in tests) and the embedded resource docs (via `go:embed` in production) can be checked against the same `companionNames`/constant-map ground truth independently, which is actually a cleaner property than one deriving from the other: neither surface can drift from the *code*, so they cannot drift from *each other* either.

## Recommended Build Order

Ordered to respect the two hard guardrails (wire-oracle re-capture must be one deliberate step; `instructions` must never name something that does not yet exist) and to keep the skill/hooks work — which touches a disjoint package — parallelizable with the Resources work:

1. **`internal/mcp/resources.go` + `internal/mcp/resourcedocs/*.md`** — build and unit-test the Resources capability in isolation (`registerResources`, resource catalog derived from `companionNames`/`allToolNames()`, static content served verbatim). No wire-oracle or `instructions` changes yet.
2. **Wire-oracle re-capture, as one commit**: add `resources/list`/`resources/read` scenarios to `test/wireoracle/scenarios.go` (valid URI, unknown-URI error shape), then re-run the standalone capture tool (`test/wireoracle/cmd/wireoracle`) against the now-Resources-capable server to refresh every affected `.golden` transcript's `capabilities` object. Review the diff as a single reviewable unit per `MUTATION-PROOF.md`'s discipline. `TestFrozenTranscriptsMatch` should go from red (expected, post-step-1) back to green here.
3. **`internal/mcp/resources_contract_test.go`** — the drift guard for resource content (Q3), built once real resource docs exist to check.
4. **Rewrite `instructions`** (`server.go`) to name the new resource mechanism instead of the stale Phase-3 promise; extend `instructions_contract_test.go` with the fourth anchor; re-verify the 600-byte/no-newline budget (this is the step most likely to need trimming elsewhere in the string to make room).
5. **In parallel with 1–4** (disjoint package, no dependency): `internal/agents/skillfiles.go` + `internal/agents/skillfiles/<target>/...` embedded trees, wired into whichever targets' `Install()`/`Uninstall()`/`DescribePaths()` opt in (Claude first, per the structural fact in Q2; Cursor/Codex CLI as a scope decision). Test via the existing per-target `_test.go` idempotency-round-trip convention (install → assert byte state → uninstall → assert restored) already used by every other writer in this package.
6. **Rewrite `internal/agents/instructions.go`'s `codegraphInstructionsBlock` body and its stale top-of-file comment** — last, because its correct new text depends on both the resource mechanism existing (step 1–4) and the skill file now being installed alongside it (step 5). `codegraphSectionStart`/`codegraphSectionEnd` themselves stay byte-unchanged (the file's own top comment: "This is a hard cross-implementation contract... Do not alter this text" — a TS-Go round-trip requirement, not a style choice); only the prose between the fences may change, and the TS↔Go marker round-trip test convention already covering this file should be re-run against the new body, not just the old one.
7. **`README.md`** — update last, once the real mechanism names/URIs are final, so `TestREADMEDocumentsToolVisibilityGate`'s existing checker and the new resource-drift checks both pass against real content rather than placeholders.

Steps 1–4 and step 5 have no ordering dependency on each other and can land as independently reviewable, independently revertable change sets; only step 6 depends on both having landed, and step 2 is the one step that must not be split across multiple commits.

## Sources

- `/Volumes/Code/github.com/seanb4t/codegraph-go/internal/mcp/server.go` — `BuildServer`, `instructions` const, `registerTools`/`unregisterTools`/`allToolNames`, `catalogMu`/`hasCatalog`/`recheckCatalog`, CacheScope correction for `tools/list`/`server/discover`
- `/Volumes/Code/github.com/seanb4t/codegraph-go/internal/mcp/tools.go` — `openEngine`, `confineToRepoRoot`, `toolAnnotations`, per-tool call-graph audit comment
- `/Volumes/Code/github.com/seanb4t/codegraph-go/internal/mcp/instructions_contract_test.go` — the three-anchor mechanism-coverage pattern, `docNamesCompanionsWithoutTheFilter`, the README cross-check
- `/Volumes/Code/github.com/seanb4t/codegraph-go/internal/mcp/tools_schema_drift_test.go` — `numericClaimRe`/`engineConstantFor`, the SURF-01 precedent for pinning numeric claims to source constants
- `/Volumes/Code/github.com/seanb4t/codegraph-go/internal/graphstore/archtest/import_graph_test.go` — `TestNoPackageBypassesGraphStore`, the actual mechanism enforcing the Pebble-import boundary
- `/Volumes/Code/github.com/seanb4t/codegraph-go/internal/cli/archtest/mcp_sdk_confinement_test.go` — SDK-02, `internal/cli` must not import an MCP SDK package directly
- `/Volumes/Code/github.com/seanb4t/codegraph-go/internal/agents/types.go`, `registry.go`, `shared.go`, `claude.go`, `instructions.go` — `AgentTarget` interface, self-registering registry, `atomicWriteFile`/`recordFile`/`replaceOrAppendMarkedSection`, the marker-fence "do not alter" contract
- `/Volumes/Code/github.com/seanb4t/codegraph-go/internal/cli/install.go` — `printAgentResults`, `installStatus`, per-target `Install(loc, opts)` call site
- `/Volumes/Code/github.com/seanb4t/codegraph-go/test/wireoracle/scenarios.go`, `oracle_test.go`, `MUTATION-PROOF.md`, `testdata/wireoracle/transcripts/*.golden` — the frozen wire-level regression oracle, its scenario/transcript pairing invariant, and its non-vacuity discipline
- `$GOMODCACHE/github.com/modelcontextprotocol/go-sdk@v1.7.0/mcp/server.go`, `resource.go`, `protocol.go`, `content.go` — verified directly against the vendored module source: `(*Server).AddResource`/`AddResourceTemplate`, `ResourceHandler` signature, `Resource`/`ResourceContents`/`ReadResourceResult` field shapes, and the `caps.Resources` auto-derivation (`s.opts.HasResources || s.resources.len() > 0 || s.resourceTemplates.len() > 0`) — HIGH confidence, read from the actual dependency version this project pins, not from general MCP-spec documentation
- `.planning/todos/pending/2026-08-08-author-a-codegraph-usage-skill-for-agents.md` — the motivating incident, the named "must not silently violate" constraint on `internal/agents/instructions.go`, and the "guard the claims" mandate this doc's Q3 section directly answers
- `.planning/PROJECT.md` — v0.10.0 milestone scope, prior Key Decisions (D-05/D-09/D-11/D-13 on `internal/mcp`'s registration and caching discipline) this design extends rather than re-derives
- This environment's own project-skill listing (system context) — corroborates that `.claude/skills/`, `.cursor/skills/`, and `.codex/skills/` are all recognized `SKILL.md`-index conventions today, informing Q2's "which roster targets can plausibly support this" note

---
*Architecture research for: codegraph-go v0.10.0 — Agent Onboarding Skill & MCP Resources*
*Researched: 2026-08-12*
