---
created: 2026-08-09T01:26:19.374Z
title: Author a codegraph usage skill for agents
area: agents
severity: major
files:
  - internal/agents/instructions.go:17-18
  - internal/mcp/server.go:56
  - README.md:91-93
  - internal/cli/install.go
  - internal/mcp/tools.go
---

## Problem

There is no Skill teaching an agent how to use codegraph well. Today an agent's
only guidance is (a) the marker block `codegraph install` writes into
CLAUDE.md/AGENTS.md and (b) the `instructions` string in the MCP initialize
response. Both are thin, and — as of the `mcp-server-one-tool-only` debug
session (2026-08-08) — at least one of them is actively wrong.

**This is not a fresh enhancement. It is the missing half of a hand-off the
codebase already documents as incomplete.** `internal/agents/instructions.go:17-18`
states that the marker block "explicitly defers full tool guidance to the MCP
initialize response (Phase 3)". Phase 3 (v0.3.0) shipped without ever writing
that guidance. So the marker block points at the initialize response, the
initialize response covers only tool *visibility* (and did so incorrectly), and
neither end carries usage guidance. An agent has nowhere to learn:

- When to reach for `codegraph explore` INSTEAD of grep/rg/Read — the single
  highest-value behavior, and the one the current guidance states as a bare
  assertion with no worked example.
- What each of the 8 tools is actually for, and which to pick for a given
  question (`explore` vs `search` vs `node`; `callers`/`callees` vs `impact`).
- That `codegraph explore` works as a shell command even when the MCP server is
  absent — the documented fallback, currently buried.
- How to read explore's output (verbatim source + call paths, including
  dynamic-dispatch hops grep cannot follow) and why that beats a text search.
- The `.codegraph/`-exists precondition, and what zero tools vs. a partial tool
  list actually mean.
- `CODEGRAPH_MCP_TOOLS` — which appears in ZERO user-facing surfaces today: not
  README, not `serve --help` (Short only, no Long/Example), not docs/.

Concrete evidence this gap costs real sessions: the 2026-08-08 debug session
was opened because the server showed one tool, and the shipped `instructions`
string blamed index state — an explanation that is not merely unhelpful but
INAPPLICABLE (no index yields zero tools, never one). It misdirected the first
two hypotheses of that investigation.

"Latest best practices" is a moving target and must be researched at
authoring time, not assumed: skill frontmatter/description conventions,
progressive disclosure, and the triggering-accuracy rules have all changed.
Do not write this from memory.

## Solution

TBD in detail, but the shape is constrained by work already in flight:

1. **Sequence after the `mcp-server-one-tool-only` fix lands.** That change
   flips the default to all 8 tools, inverts `CODEGRAPH_MCP_TOOLS` from an
   opt-in allowlist to an opt-out narrowing filter, and rewrites both the
   `instructions` string and README.md:91-93. Authoring the skill first would
   document a contract that is actively being replaced.
2. **Research current skill-authoring conventions** rather than pattern-matching
   an existing SKILL.md — use the `skill-creator` / `skill-development` guidance
   and verify against current docs.
3. **Decide the distribution question, which is the real design decision here:**
   does `codegraph install` WRITE the skill into the target agent's skills
   directory (making it versioned with the binary and updated by `codegraph
   upgrade`), or does it ship in-repo for users to install themselves? The
   former closes the deferred hand-off properly; the latter is lower risk but
   leaves install's output still deferring to something thin. Note that
   `internal/agents/instructions.go` is currently under a "must not change"
   constraint from an earlier phase decision (see y4er47xm4e:
   `agents-go-must-not-change`, bare `serve --mcp` = zero-config flip) — that
   constraint needs to be re-examined, not silently violated.
4. **Lead with the decision procedure, not a tool catalog.** The failure mode to
   design against is an agent that has read the skill and still reaches for grep
   first. A tool-by-tool reference is the least valuable part; a crisp
   "which tool for which question" table plus 2-3 worked examples is the most.
5. **Guard the claims.** The `mcp-server-one-tool-only` session established that
   this repo has now had TWO occurrences of wire-contract claims drifting from
   behavior with no gate comparing them (SURF-01's "default 5" after the
   constant became 2; the `instructions` visibility claim). A skill is a third
   such surface. If it names tool counts, defaults, or flag values, derive them
   or gate them the way `instructions_contract_test.go` does — do not hand-type
   numbers into prose that nothing checks.

Related: `wg23559t5t` (engram) records the full root-cause chain and the
maintainer decisions from the debug session that surfaced this gap.
