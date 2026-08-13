# SKILL-03 Rehearsal — before/after

> Live-session verification record for SKILL-03: "An agent given a fresh session, the skill
> installed, and a 'where is X' prompt selects `codegraph_explore` over grep/find/Read — verified
> by transcript diff, not asserted." This is not re-run by CI — see **Verdict** below for why.

## Before — the 2026-08-08 misdirection incident

Source: `.planning/debug/resolved/mcp-server-one-tool-only.md` (cited verbatim, not paraphrased).

**Symptoms** (quoted):

> **Expected behavior**
> The `codegraph` MCP server exposes its full agent-facing tool surface. [...] The server's own
> MCP `instructions` string — the text shipped to clients and visible in this Claude Code session
> — tells agents: *"An empty tool list means this repository has no index yet, so run codegraph
> init to fix it, not that the server is broken. Tools appear automatically once an index exists,
> with no client restart required."* That sentence promises **index-gated** visibility.
>
> **Actual behavior**
> Claude Code's `/mcp` tool listing for the `codegraph` server shows:
>
> ```
> Tools for codegraph
> 1 tool
>
>   1. codegraph_explore   destructive, open-world
> ```
>
> Only `codegraph_explore`. This repo HAS an index (`.codegraph/store/` present, `daemon.lock`
> live as of 2026-08-08 21:02), so the "no index yet" explanation in the instructions does not
> apply.

**Timeline and root cause** (quoted from the resolution block):

> (AND-gate, three simultaneous contributing causes.) (1) SPEC: the accepted contract made
> codegraph_explore the only default-visible tool, gated behind an opt-in CODEGRAPH_MCP_TOOLS
> allowlist that no user-facing artifact named. (2) DOCUMENTATION: README.md asserted "Eight
> tools are exposed" directly beneath the install snippet — false for every default installation,
> and the origin of the reported expectation. (3) WIRE CONTRACT: the MCP instructions string
> attributed tool visibility SOLELY to index state, so the one explanation offered to the agent at
> the moment of confusion was the one cause that structurally cannot produce a 1-tool list (a
> missing index produces ZERO).

**Sequence of actions the session actually took:** the investigating session read the MCP
`instructions` string's index-state explanation, found it inapplicable (an index was present),
and had to fall back to reading `internal/cli/serve.go`, `internal/mcp/server.go`, and
`README.md` by hand — a multi-file, multi-hour debugging session — to discover the real cause was
an undocumented environment-variable allowlist. No `codegraph_explore` call could have surfaced
this on its own; the actual defect was in the wire contract describing the tools, not in the
tools' own behavior. This is the class of question SKILL-03 exists to shortcut for: an agent
facing a confusing "why does the server look broken" question, forced into ad hoc file reading
because nothing pointed it at the graph tools or explained what it was looking at.

## After — freshly captured session (2026-08-12)

**Session:** a genuinely new Claude Code session, no prior conversation turns, no authoring
context from this phase's planning work — distinct from both the phase's authoring session and
from the session used for the Part A nudge rehearsal (see `NUDGE-live-session.md`).

**Prompt used** (names no tool): *"How does the daemon decide when to trigger a re-sync of the
graph store?"*

**Whether the skill was listed:**

> **Retraction (2026-08-13).** This line previously read: *"No. The session's own skill catalog
> (inspected directly in its transcript, not inferred) does not list the `codegraph` skill at all,
> despite `.claude/skills/codegraph/SKILL.md` being correctly placed and committed."* **That was
> false.** The skill WAS listed. The captured session's transcript
> (`c4b8f662-07d5-4549-ba5d-be3f7ba795d2.jsonl`) contains an `attachment` record of
> `"type":"skill_listing"` whose content includes, verbatim, `\n- codegraph\n` — positioned at the
> end of the personal-skill block, exactly where a project-scoped `.claude/skills/` entry belongs.
> Project-skill discovery worked in the very session this finding was written from.
>
> **What was actually wrong, and why the original claim missed it:** the entry is *degraded*, not
> absent. It renders as a bare `- codegraph` with **no colon and no description**, unlike the 173
> entries that carry `- name: description`. The rehearsal's oracle was a grep of the transcript,
> and every natural spelling of that grep — `codegraph:`, or any fragment of the description text —
> matches nothing against a bare name. The oracle could not distinguish "absent from the catalog"
> from "present but stripped of its description," and reported the first.
>
> **Root cause of the degradation (environment, not this repository's authoring).** Claude Code
> renders the skill catalog into an attachment capped near 45,000 characters (measured
> 44,976–45,014 across four independent sessions, admitting **exactly 173** descriptions in every
> one) and emits every entry that does not fit as a bare name. This operator's installation carries
> ~238 skills, saturating the cap; project-scoped skills are appended after all personal and plugin
> skills, so this repository's entry competes for the leftover budget last and loses by
> construction. Proven not to be a property of the file: a probe skill whose description was
> byte-identical to a working skill's rendered **bare in the same listing** where the original
> rendered with its description. Full probe ladder in
> `.planning/debug/resolved/skill-discovery-not-listing.md`.
>
> **Consequence for this rehearsal, unchanged in substance:** an entry with no description carries
> no trigger signal, so the skill was still semantically undiscoverable to the captured session.
> The Verdict's caveat therefore stands on its merits — but for this reason, not for the
> now-retracted "discovery is broken" reason.

**Yes — listed, but degraded to a bare name with no description**, hence carrying no trigger
signal. See **Verdict** for what that means for SKILL-03's attribution.

**First code-search action:** the session's transcript tool-call sequence, in order, was
`ToolSearch` and `mcp__engram__list_memory` (both mandated by an unrelated, pre-existing global
memory-recall hook, not code search), then **`mcp__codegraph__codegraph_explore`** with query
`"daemon re-sync trigger graph store file watcher debounce"`. That is the first code-search
action, and it is `codegraph_explore` — not grep, find, or a file read.

**Answer path taken:** the session correctly traced the chain policy gate (`Daemon.Run`,
`internal/daemon/daemon.go:221`) → `watchLoop` (`internal/watch/watcher.go:88`) → `Debouncer`
(`internal/watch/debounce.go`) → `flush` (`daemon.go:435`) → `indexer.Sync`
(`internal/indexer/sync.go:34`), with accurate file:line citations consistent with genuine
`codegraph_explore` usage rather than fabrication.

## Verdict

**SKILL-03's literal criterion is met by this transcript:** the first code-search action for a
where-is-X-class prompt was `codegraph_explore`, not grep/find/Read.

**Open caveat, recorded honestly rather than glossed over** (corrected 2026-08-13; the original
text asserted the skill "was not surfaced to the captured session at all — its skill-listing
system reminder never named `codegraph`", which is false — see the retraction above): the
newly-authored `.claude/skills/codegraph/SKILL.md` *was* listed in the captured session, but as a
bare `- codegraph` with its description dropped by Claude Code's ~45,000-character catalog cap, so
it reached the session carrying no trigger signal. Two other, independent, pre-existing mechanisms
already in context are each independently sufficient to explain the correct tool choice: the
operator's global `~/.claude/CLAUDE.md` CodeGraph section, and the codegraph MCP server's own
`instructions` string (rewritten as part of the 2026-08-08 fix to say "try codegraph_explore first
for a where-is-X or how-does-Y-work question"). This rehearsal therefore cannot cleanly attribute
the correct routing to the phase's own artifact — it demonstrates the desired *outcome*, not that
this specific *skill* caused it. That limitation is unchanged by the retraction; only its cause is.

**Standing caution about this artifact's method.** The retracted claim above is the second false
negative this phase's live rehearsals produced by grepping a transcript (the first was the
resume-matcher finding in `NUDGE-live-session.md`). Both times the mechanism was working and the
grep was blind to how the runtime actually renders it. Before recording "X did not happen" from a
transcript search, demonstrate that the same search *can* find X when X does happen — otherwise
the finding is a claim about the grep, not about the product.

**Standing note — not re-run by CI:** this artifact is a one-off, human-run review, not a
continuously enforced gate. This repository has no harness for driving a real agent session from
an automated test, and D-01 (`06-CONTEXT.md`) explicitly declines to build one for this phase. A
future reader must not mistake this file for a standing guarantee that current code still
produces the same result — treat it as evidence from one point in time, not a live check.
