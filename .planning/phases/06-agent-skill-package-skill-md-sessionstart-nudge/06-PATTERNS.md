# Phase 6: Agent Skill Package — SKILL.md & SessionStart Nudge - Pattern Map

**Mapped:** 2026-08-12
**Files analyzed:** 6 new + 1 modified (test extension)
**Analogs found:** 6 / 6 (all role-match or better; content-type files have no exact prior analog since this is the first skill/hook authored in this repo — see No Analog Found)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `.claude/skills/codegraph/SKILL.md` | config/content (agent-facing markdown) | request-response (static, loaded on trigger) | `internal/mcp/resources/*.md` (Phase 5) + `internal/agents/instructions.go`'s `codegraphInstructionsBlock` constant | role-match (content type is new; authoring discipline is not) |
| `.claude/settings.json` | config | request-response (hook registration, read once at session start) | `.claude/settings.local.json` (this repo's own existing settings file, uncommitted sibling) | role-match (shape confirmed, but semantically new — first *committed* settings.json) |
| `.claude/hooks/session-nudge.sh` | utility (shell script, event-driven) | event-driven (SessionStart hook, stdin→stdout) | none in-repo (first shell hook script) — nearest analog is RESEARCH.md's own verified Pattern 2 snippet, itself cross-checked against official docs | no analog (net-new artifact type) |
| `.claude/hooks/hooks.json` | config/content (versioned fragment for Phase 7 `go:embed`) | file-I/O (static, read later by Phase 7's embed) | `.claude/settings.json` (same event/matcher JSON shape, sibling file this phase also creates) | role-match (shares JSON shape with settings.json's hooks block) |
| `.claude/skills/codegraph/verification/<rehearsal>.md` (transcript pair) | test (manual/rehearsal artifact) | request-response (captured, not executed) | `.planning/debug/resolved/mcp-server-one-tool-only.md` (the "before" half is this exact file, reused per D-02) | exact (before half is literally this file; after half mirrors its shape) |
| `internal/mcp/resources_schema_drift_test.go` (extended, not new) | test | structural/set-equality | itself — `TestResourceFileSetMatchesToolNames`/`numericClaimRe`/`envVarTokenRe`/`hostFactsPosixRe` (same file, existing GUARD-01/02 checkers) | exact |

## Pattern Assignments

### `.claude/skills/codegraph/SKILL.md` (content, request-response)

**Analog A — anti-duplication content discipline:** `internal/mcp/resources.go` doc comment + the `instructions` constant's own constraint comment, `internal/mcp/server.go:38-51` (read in Phase 5 pattern map; re-verified in RESEARCH.md this phase):
```go
// ... it MUST stay a compile-time literal with no interpolation
// of any kind — no repository path, no resolved index root, no hostname,
// no environment value ...
```
Apply the same "static, source-derived, no restated facts" discipline to SKILL.md's prose: point at `codegraph://tools/<name>` URIs by name (Pattern 3 in RESEARCH.md) rather than restating params/defaults Phase 5's resources already own.

**Analog B — existing decision-procedure-first shape already in this repo**, `.claude/CLAUDE.md`'s own `## CodeGraph` section (verbatim, currently live):
```markdown
## CodeGraph

In repositories indexed by CodeGraph (a `.codegraph/` directory exists at the repo root), reach for it BEFORE grep/find or reading files when you need to understand or locate code:

- **MCP tool** (when available): `codegraph_explore` answers most code questions in one call — the relevant symbols' verbatim source plus the call paths between them.
- **Shell** (always works): `codegraph explore "<symbol names or question>"` prints the same output.

If there is no `.codegraph/` directory, skip CodeGraph entirely — indexing is the user's decision.
```
SKILL.md's decision table (SKILL-01) is this same "reach for it BEFORE grep" framing expanded into a table with more question shapes, not a rewrite of its tone. Copy the imperative, second/third-person register and the explicit "skip if no `.codegraph/`" escape hatch verbatim.

**Analog C — the marker-fence pointer block this repo already ships**, `internal/agents/instructions.go:20-29` (`codegraphInstructionsBlock`, read in full above) — same "reach for it BEFORE grep/find or reading files" sentence appears here too. SKILL.md must not contradict or duplicate this block's claims; it supersedes it in depth (playbook-level) while this block stays the short pointer, per RESEARCH.md's "State of the Art" table row.

**Frontmatter shape** (RESEARCH.md Code Examples, agentskills.io-verified — copy verbatim structure):
```yaml
---
name: codegraph
description: Use when the user asks where a symbol is defined, how a function works, what calls or is called by a symbol, or what breaks if a signature changes — reach for codegraph's call/symbol graph instead of grep/find/Read in any .codegraph/-indexed repository.
---
```

**Worked example 1 (misdirection incident) — source of truth to quote from, not paraphrase drift:** `.planning/debug/resolved/mcp-server-one-tool-only.md:9-25,113-150` (root cause, timeline). Cross-reference the *current* (fixed) `instructions` constant text at `internal/mcp/server.go:56` so the worked example doesn't imply the bug is still live — RESEARCH.md's Code Examples section gives the exact current string to quote.

**Worked example 2 (impact analysis) — verified real CLI/MCP surface to copy:**
```
internal/mcp/tools.go:264-268   → ImpactArgs{Depth, Path, Symbol}, jsonschema:"BFS depth (default 2, max 50)"
internal/cli/impact.go:16-23,64-66 → Use: "impact <symbol>", flags --path/-p, --depth/-d (default 0 = engine default), --json/-j
```

---

### `.claude/settings.json` (config, request-response)

**Analog:** `.claude/settings.local.json` (this repo, read this session — full file, 23 lines) — confirms the top-level JSON shape (`permissions`, `enabledMcpjsonServers`, etc. as sibling top-level keys) this new file must slot a `hooks` key into without disturbing.

**Core hook-registration pattern to copy verbatim** (RESEARCH.md Pattern 2, verified against `code.claude.com/docs/en/hooks` fetched live):
```json
{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "startup",
        "hooks": [
          { "type": "command", "command": "${CLAUDE_PROJECT_DIR}/.claude/hooks/session-nudge.sh" }
        ]
      },
      {
        "matcher": "resume",
        "hooks": [
          { "type": "command", "command": "${CLAUDE_PROJECT_DIR}/.claude/hooks/session-nudge.sh" }
        ]
      }
    ]
  }
}
```
**Note:** this repo has no committed `.claude/settings.json` today — only `.claude/settings.local.json` (gitignored) exists. This file is net-new, not a merge into existing committed content, so no marker-fence/safe-merge concern applies here (that concern is deferred to Phase 7, per RESEARCH.md's Security Domain section).

---

### `.claude/hooks/session-nudge.sh` (utility, event-driven)

**Analog:** none in-repo — first shell hook script. Copy RESEARCH.md's verified, complete script body:
```sh
#!/bin/sh
# .claude/hooks/session-nudge.sh — NUDGE-01/02
# No stdin parsing needed: matcher already filters to startup|resume.
if [ -d "${CLAUDE_PROJECT_DIR:-.}/.codegraph" ]; then
  echo "This repo has a codegraph index — prefer codegraph_explore / \`codegraph explore\` over grep for where-is-X / how-does-Y questions."
fi
exit 0
```
Message text must match D-06 exactly: one line, no tool list, no examples, no fallback-syntax reminder.

**Error-handling/security pattern to copy:** RESEARCH.md's Security Domain table — no `eval`, no untrusted input parsing, reference the script by its full `${CLAUDE_PROJECT_DIR}`-relative path in settings.json (never bare filename) to avoid `$PATH`-shadowing spoofing.

---

### `.claude/hooks/hooks.json` (content, file-I/O)

**Analog:** same JSON shape as `.claude/settings.json`'s `hooks.SessionStart` block above — this file is the versioned fragment Phase 7 will `go:embed`; RESEARCH.md's Open Question 1 flags that the planner may choose to author only `.claude/settings.json` and skip this file for Phase 6, deferring extraction to Phase 7. If created, its content must be sourced from (not independently hand-typed against) the same block `.claude/settings.json` carries, per RESEARCH.md's A2 single-sourcing recommendation.

---

### `.claude/skills/codegraph/verification/<rehearsal>.md` (test/rehearsal, request-response)

**Analog ("before" half is literally this file):** `.planning/debug/resolved/mcp-server-one-tool-only.md` — reuse this file's documented timeline/root-cause/evidence sections as the "before" transcript per D-02; do not re-summarize, cite/quote directly.

**Analog (shape for the "after" half):** no in-repo transcript-capture precedent exists (RESEARCH.md's Validation Architecture confirms this is a Wave-0 gap) — follow the debug log's own section structure (timeline, evidence, root cause) for parallelism between before/after, making the diff legible.

---

### `internal/mcp/resources_schema_drift_test.go` (extended, structural/set-equality)

**Analog:** itself — the file's own existing GUARD-01/02 checkers (read this session, lines 1-120+):
```go
var toolNameTokenRe = regexp.MustCompile(`codegraph_[a-z]+`)

func resourceStemSetDiff(expected, actual []string) (missing, orphaned []string) {
    // ... symmetric missing/orphaned set-equality, sorted for stable failure messages
}
```
Extend the same regex/map-driven checkers (`numericClaimRe`, `envVarTokenRe`, `hostFactsPosixRe` — confirmed present per RESEARCH.md's Don't Hand-Roll table) to also scan `.claude/skills/codegraph/SKILL.md`'s embedded/read content, not a second parallel mechanism. Any numeric claim (tool count, depth default), `CODEGRAPH_MCP_TOOLS` mention, or absolute host path in SKILL.md must resolve through this file's existing derive-don't-hand-type checkers.

**Doc-comment tagging convention to copy** (file's own header comment, lines 17-25): tag the new SKILL.md-scanning logic with its own `GUARD-NN`/decision-ID reference, following the `D-NN`/`SURF-01` inline-tagging convention already pervasive in this file and `server.go`.

---

## Shared Patterns

### Decision-procedure-first framing (already proven in this repo)
**Source:** `.claude/CLAUDE.md`'s live `## CodeGraph` section; `internal/agents/instructions.go`'s `codegraphInstructionsBlock`
**Apply to:** SKILL.md's opening decision table — same "reach for it BEFORE grep/find" imperative, expanded not reinvented.

### Source-derived, never hand-duplicated facts (GUARD-01/02, the "SURF-01" lesson, now applying to a third surface)
**Source:** `internal/mcp/resources_schema_drift_test.go`'s header comment (lines 17-25), directly recounting the `defaultDepth` 5→2 drift incident.
**Apply to:** every factual/numeric claim in SKILL.md and the nudge text — must trace to `tools.go`/`server.go`/`companionNames`, checked by the extended drift-guard test, not eyeballed.

### `${CLAUDE_PROJECT_DIR}`-anchored command references
**Source:** RESEARCH.md Pattern 2 + Security Domain (spoofing mitigation), cross-checked against official docs.
**Apply to:** the `command` field in both `settings.json` and `hooks.json` — always the full anchored path, never a bare script name.

### No new Go dependency / no CGo consideration
**Source:** RESEARCH.md's "Standard Stack" section — this phase's primary deliverables are markdown/JSON/POSIX shell only; the one Go-touching change is a test file extension, using only already-imported `regexp`/`testing`/`slices`/`sort` per the existing drift-guard file's own imports (read this session, lines 1-15).

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `.claude/hooks/session-nudge.sh` | utility | event-driven | First shell hook script in this repo; RESEARCH.md's own verified snippet is the only available template — treat RESEARCH.md's Code Examples section as the analog source instead of in-repo code |
| `.claude/skills/codegraph/verification/<rehearsal>.md` ("after" half only) | test/rehearsal | request-response | No prior transcript-capture artifact exists in this repo (confirmed Wave-0 gap in RESEARCH.md); the "before" half has an exact analog (the debug log itself), but the "after" half must be captured fresh during execution, not copied from existing code |

## Metadata

**Analog search scope:** `.claude/` (confirmed empty of skills/hooks — `ls` run this session), `internal/mcp/` (Phase 5's resources + drift-guard test), `internal/agents/instructions.go`, `.planning/debug/resolved/`
**Files scanned:** `.claude/settings.local.json`, `.claude/CLAUDE.md`, `internal/agents/instructions.go`, `internal/mcp/resources_schema_drift_test.go` (partial, lines 1-120), `internal/mcp/resources.go` (grep only, `resourceURIFor`/`companionNames` symbols), `.planning/phases/05-.../05-PATTERNS.md`, `.planning/debug/resolved/mcp-server-one-tool-only.md` (referenced via RESEARCH.md, not re-read this session — already fully quoted there)
**Pattern extraction date:** 2026-08-12
