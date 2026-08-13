# Phase 8: Instructions & Marker-Block Rewrite - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-13
**Phase:** 8-Instructions & Marker-Block Rewrite
**Areas discussed:** Wire budget vs. new content, Resource URI naming granularity, Marker-block content per agent target

---

## Wire budget vs. new content

| Option | Description | Selected |
|--------|-------------|----------|
| Keep 600, go generic | Instructions stay a short pointer rather than restating content; may require trimming an existing clause to make room. | ✓ |
| Raise the budget | Lets the string be more explicit, at the cost of bigger wire-oracle diffs forever and going against research consensus. | |
| You decide | Let Claude pick during planning. | |

**User's choice:** Keep 600, go generic
**Notes:** The current `instructions` const measures 580/600 bytes with zero headroom. Research (MCP's own server-instructions blog, OpenAI's MCP guidance, niteagent's checklist) unanimously favors short, pointer-style instructions over exhaustive detail — instructions under ~300 words, "don't write a manual," state only what tool descriptions can't already convey.

---

## Resource URI naming granularity

| Option | Description | Selected |
|--------|-------------|----------|
| Generic pointer only | Instructions say "see the codegraph skill / call resources/list" rather than naming individual URIs — matches MCP idiom, fits the byte budget, lightens the drift-guard surface. | ✓ |
| Name specific URIs | Literally enumerate all 10 codegraph:// URIs — more discoverable without a round-trip, but expands the drift guard and doesn't fit the 600-byte budget. | |
| You decide | Let Claude pick the exact phrasing during planning. | |

**User's choice:** Generic pointer only
**Notes:** No research source found suggests re-listing discoverable resource URIs inside a top-level instructions string; the standard MCP pattern is client-side discovery via `resources/list`.

---

## Marker-block content per agent target

| Option | Description | Selected |
|--------|-------------|----------|
| One shared block, skill-agnostic | Single constant across all 4 targets (Claude, Codex, opencode, Gemini), describing `codegraph_explore` directly, never naming the skill by path. Simplest, zero drift risk, satisfies WIRE-03 uniformly. | ✓ |
| Diverge per target | Claude Code's block points at its installed skill; the other 3 keep the current self-contained paragraph. More "correct" per research, but turns one constant into a per-target template. | |
| You decide | Let Claude weigh this during planning. | |

**User's choice:** One shared block, skill-agnostic
**Notes:** Only Claude Code received the actual SKILL.md+hooks install (Phase 7, v1-scoped) — Codex/opencode/Gemini did not (v2, AGENT-04…07). Pointing those 3 targets at a skill file that was never installed for them would recreate the exact "promise nothing delivers" defect WIRE-01/02/03 exist to close.

---

## Claude's Discretion

- Exact wording of the generic skill/resources pointer phrase in both the `instructions` const and the marker block.
- Which existing clause in the current 580-byte `instructions` string gets trimmed/restructured to make room for the new pointer.
- Wire-oracle re-capture and diff-attribution mechanics — follows the existing `test/wireoracle` reviewed-diff discipline.
- Exact implementation of WIRE-03's "every named capability is resolvable at test time" guard.
- Correcting the stale "Phase 3" deferral reference in `internal/agents/instructions.go`'s package comment (accuracy fix, not a decision).

## Deferred Ideas

None — discussion stayed within phase scope.

**Todos reviewed, not folded:**
- `2026-08-07-wire-oracle-toolslist-repeat-response-ordering-flake.md` (score 0.9) — a wire-oracle response-*arrival-order* nondeterminism, unrelated to instructions content; already tracked separately in PROJECT.md's open threads.
- 5 lower-score (0.6) todos (release dry-run diff guard, post-release-verify conclusion guard, golangci-lint, brew tap trust docs, tap secret-distinctness test) — keyword-match only, no actual overlap.
