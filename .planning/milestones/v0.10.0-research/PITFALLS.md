# Pitfalls Research

**Domain:** Adding a self-teaching agent skill/plugin, an MCP Resources capability, and enforcement hooks to an existing mature CLI+MCP tool (codegraph-go v0.10.0)
**Researched:** 2026-08-12
**Confidence:** HIGH (grounded directly in this repo's own documented incident history and existing enforcement machinery — `internal/mcp/instructions_contract_test.go`, `test/wireoracle/`, `internal/agents/` `AgentTarget` registry, `internal/cli/archtest/`, and PROJECT.md's Key Decisions log of prior gate failures)

## Critical Pitfalls

### Pitfall 1: The skill is inert — an agent that has read it still reaches for grep first

**What goes wrong:**
The skill ships, is discoverable, reads well in isolation — and changes nothing. Agents keep defaulting to `grep`/`find`/`Read` for where-is-X and how-does-Y-work questions because the skill was written as a tool catalog ("codegraph has 8 tools: explore, search, node...") rather than a decision procedure, so it never actually out-competes the agent's own strong prior toward grep. This is the exact failure mode the scoping todo names verbatim: *"an agent that has read the skill and still reaches for grep first."*

**Why it happens:**
Tool descriptions and reference docs feel like the "complete" content to write, so authors front-load them. But an agent doesn't fail to use codegraph because it lacks facts about codegraph — it fails because grep is the path of least resistance and nothing has made the alternative path *cheaper to reach for* in the moment a where-is-X question appears. A catalog answers "what exists"; only a decision procedure answers "what do I do right now."

**How to avoid:**
- Lead the skill body with a short decision table ("which tool for which question"), not a tool list. The todo itself specifies this: 2-3 worked examples plus a crisp table is "the most valuable part"; tool-by-tool reference is "the least valuable part."
- Validate against behavior, not against skill content review. A skill that reads well is not evidence it changes behavior — run real onboarding-style transcripts (a fresh agent session, a where-is-X prompt, no other context) before and after shipping the skill and diff tool-call choice. This is the same "green means nothing if the gate could not have failed" standing rule this repo already applies elsewhere (PROJECT.md's Key Decisions table) — apply it to the skill's own effectiveness, not just to code.
- Keep the worked examples adversarial: pick prompts phrased exactly the way a user would ask them ("where is the auth check", "how does X call Y"), not phrased to make codegraph_explore the obvious answer.

**Warning signs:**
- The skill file's word count on tool descriptions exceeds its word count on the decision procedure.
- No transcript-based verification step exists in the phase plan — only "skill reviewed" or "skill written per best practices."
- The skill was authored by pattern-matching an existing SKILL.md rather than fresh research (the todo explicitly warns against this: "do not write this from memory").

**Phase to address:**
Skill-authoring phase — build the transcript-diff verification into that phase's acceptance criteria, not as a follow-up.

---

### Pitfall 2: MCP Resources drift from tool behavior because nothing gates the two staying in sync

**What goes wrong:**
The new `resources/list`/`resources/read` capability serves detailed reference content — tool-by-tool docs, `CODEGRAPH_MCP_TOOLS` semantics, index-state preconditions. Those resources are hand-authored prose, separate from the actual tool registration logic in `internal/mcp/tools.go` and the `instructions` constant in `internal/mcp/server.go`. The moment either changes (a tool added/removed, a default flipped, a precondition altered) the resource content silently goes stale — exactly the class of bug this project has already shipped twice: SURF-01's "default 5" surviving after the constant became 2, and the `instructions` string blaming index state for a symptom it structurally cannot cause. A resource is a *third* independent surface making the same class of claim the `instructions` string and README already made and got wrong.

**Why it happens:**
Resources are new server-side content with no existing test harness pointed at them. The project's wire oracle (`test/wireoracle/`) freezes JSON-RPC transcripts including `tools/list` and `initialize`, but a fresh `resources/list`/`resources/read` capability starts with zero scenarios in that oracle and zero coverage in `instructions_contract_test.go`-style claim-derivation tests. Prose is cheap to write and easy to forget to update; a hand-typed tool count or flag default in a resource body is indistinguishable, at commit time, from one that's still correct.

**How to avoid:**
- Apply the exact discipline `instructions_contract_test.go` already established for the `instructions` string to every resource body: derive claims from source (tool registry length, `CODEGRAPH_MCP_TOOLS` behavior, actual default), never hand-type them into prose. If a resource says "8 tools," that number must come from `len(registeredTools)` or equivalent at test time, not be typed as a literal.
- Extend the wire oracle's frozen-transcript coverage to `resources/list` and `resources/read`, following the same pattern already used for `tools/list`: capture real transcripts, freeze them, diff on change. This makes a resource content drift show up as a red test, not as a silently-wrong document an agent reads mid-session.
- Add a single automated cross-check that resource content and `tools/list` output agree on tool names/count — a small "claims that must match reality" test, mirroring `instructions_contract_test.go`'s `strings.Contains(instructions, allowlistEnvName)` pattern but pointed at resource bodies.
- Do not let the skill *also* embed the same facts a resource states. Every fact should have exactly one source of truth (skill points to resource, resource derives from code) — duplicating a fact across skill + resource + instructions string is what created the AND-gate-of-three failure PROJECT.md documents for MCP-01 ("remove any one and it is caught in seconds" was true only because there were three independent copies to disagree).

**Warning signs:**
- A resource file contains a bare number, flag name, or default value with no `//go:generate`, no derived-constant reference, and no test asserting it.
- `resources/list`/`resources/read` has zero entries in `test/wireoracle/scenarios.go`.
- A PR changes tool registration (`internal/mcp/tools.go`) without a corresponding diff in resource content or a failing test forcing one.

**Phase to address:**
The MCP Resources phase itself — the derivation/gating mechanism must ship in the same phase as the resources, not as a follow-up, because an ungated resource is worse than no resource (it's a second `instructions`-string incident waiting to happen, and the project has already had two occurrences of this exact bug class).

---

### Pitfall 3: The guard hook false-positive-blocks legitimate grep/find/Read use

**What goes wrong:**
A PreToolUse/UserPromptSubmit guard meant to redirect grep/find toward `codegraph_explore` fires on cases where grep is actually the right tool — searching for a literal string/config key, grepping inside a single already-open file, searching non-code text (logs, CHANGELOGs, generated data), or operating in a directory with no `.codegraph/` index at all. A hook that blocks or nags on these cases trains the user/agent to route around it (disable it, ignore its output, or, worse, an agent burns a turn negotiating with its own hook instead of doing the task) — the opposite of the intended effect.

**Why it happens:**
"Nudge toward codegraph for where-is-X questions" is a semantic judgment call being implemented as a syntactic hook (tool-name + maybe argument pattern matching on `grep`/`find`/`Read`). Hooks don't have the judgment a skill's decision procedure provides; they only see the tool call, not the intent behind it. A hook author under time pressure will reach for "grep call in a `.codegraph/`-indexed repo == block/nudge," which conflates "grep is being used" with "grep is being used for a where-is-X code-navigation question," and those are not the same event.

**How to avoid:**
- Scope the guard's trigger surface narrowly and conservatively: fire only on clear where-is-X/how-does-Y-work signal (heuristics on the *prompt* text via UserPromptSubmit, not blanket PreToolUse interception of every grep call) rather than trying to classify every grep invocation's intent from the tool call alone.
- Make the default posture a *nudge* (inject context, suggest an alternative) not a *block* (deny the tool call), unless there is very high confidence — this project's own MCP tool visibility already defaults to permissive (all 8 tools visible by default, narrowing is opt-in per D-05/MCP-01's supersession) which is the established posture to mirror: default open, narrow only on explicit signal.
- Gracefully no-op, not warn/error, when `.codegraph/` doesn't exist or the MCP server isn't reachable — a hook that produces friction in an unindexed repo (where codegraph literally cannot help) actively contradicts the "reduce friction toward the better tool" goal. Detect absence cheaply (file-existence check for `.codegraph/`) and short-circuit before any nudge/guard logic runs.
- Test the guard against a corpus of real, legitimate grep/find/Read calls captured from this project's own session history (or a synthetic set covering: string literal search, single-file grep, non-code text search, no-index repo, `.codegraph/` present but stale) and assert zero false positives on that corpus before shipping — this is the practical equivalent of a wire-oracle transcript freeze, but for hook decisions instead of MCP frames.

**Warning signs:**
- The guard's matching logic is a bare tool-name check (`if tool == "Grep"`) with no context signal beyond "an index exists."
- No test corpus of legitimate grep/find/Read calls exists to run the guard against before shipping.
- The guard's failure mode on missing `.codegraph/` or unreachable MCP server is anything other than silent no-op (an error, a loud warning, a block).

**Phase to address:**
Hooks phase — the false-positive corpus test must be a phase acceptance gate, not a nice-to-have, given this repo's standing rule that green-without-a-real-fail-condition proves nothing.

---

### Pitfall 4: The guard hook is too passive — it exists but never actually fires

**What goes wrong:**
The opposite failure: the guard is written defensively enough (per Pitfall 3) that its trigger conditions are so narrow it essentially never activates. It ships, is documented as installed, and produces zero measurable change in tool selection — indistinguishable from Pitfall 1's inert skill, except now there are two inert mechanisms instead of one, and the SessionStart nudge on its own becomes the *entire* behavior-change budget, which the scoping todo already establishes is insufficient on its own (the skill's whole reason to exist is that passive documentation didn't work).

**Why it happens:**
Overcorrecting from Pitfall 3 (too aggressive) without a positive test proving the guard *does* fire on the cases it's meant to catch. Teams that add a false-positive test suite often stop there and never add the mirror-image true-positive suite, so "never blocks legitimate use" quietly becomes "never blocks anything."

**How to avoid:**
- Require both suites: a false-positive corpus (Pitfall 3) AND a true-positive corpus — a set of realistic where-is-X/how-does-Y prompts that MUST trigger the nudge/guard, run as an explicit test with an assertion that it fires, not just that it doesn't misfire.
- Instrument the guard (even minimally — a debug log line, a counter) during a manual rehearsal period so its actual fire rate on a full session can be checked against expectation before calling the phase done.
- Treat "guard exists in hooks.json and is syntactically valid" as zero evidence of anything — same standing rule this repo already applies to gates ("a gate is not trusted until demonstrated RED against a confirmed-applied mutation"). Demonstrate the guard actually intercepting a real grep call in a rehearsal session, not just unit-tested in isolation.

**Warning signs:**
- Only a false-positive/non-firing test suite exists; no true-positive/must-fire suite.
- The guard's trigger regex/heuristic was tightened repeatedly during development (each tightening in response to a false positive) with no corresponding check that true positives still pass.

**Phase to address:**
Hooks phase, same acceptance gate as Pitfall 3 — the two test suites (must-not-fire, must-fire) should ship together.

---

### Pitfall 5: Install/uninstall corrupts or silently drops the NEW file types (skill dirs, hooks.json) because the existing `AgentTarget` abstraction was built for a narrower shape

**What goes wrong:**
`internal/agents/`'s existing `AgentTarget` interface (`Install`/`Uninstall`/`Detect`/`DescribePaths`) was designed and hardened for exactly two artifact shapes: an MCP config JSON/JSONC/TOML/YAML entry, and a marker-fenced instructions-file injection (`<!-- CODEGRAPH_START/END -->`) into 4 of the 8 agents' instruction files. Skill directories (a directory tree: `SKILL.md` + supporting files) and `hooks.json` (a structured config file with its own schema per agent, some agents using `hooks.json`, others embedding hooks in settings, others not supporting hooks at all) are a *third and fourth* artifact shape this interface has never had to express. Bolting them on as ad-hoc special cases per agent — rather than extending the interface's contract — reproduces this project's own documented history of exactly this kind of gap: "swallowed I/O errors," "Antigravity migration data-loss," and "Hermes CRLF idempotency" were all found in v1.0 Phase 6's deep review of the *existing*, narrower two-shape install/uninstall subsystem. A new, less-tested third/fourth shape is higher risk, not lower.

**Why it happens:**
The temptation is to treat "write a skill directory" and "write hooks.json" as simple file-copy operations outside the `AgentTarget` abstraction, since they don't fit the existing `Install(loc, opts) WriteResult` / marker-fence model cleanly. But that's precisely how idempotency, byte-invariant round-trips, and safe co-existence with user-owned content got hardened for the existing two shapes — through iterated deep review, not through initial design. Skipping that same rigor for the new shapes because they're "just files" reproduces the bugs the existing subsystem already paid down.

**How to avoid:**
- Extend the `AgentTarget` interface's contract (or add a sibling interface with the same guarantees: idempotent install, byte-invariant uninstall restoring pre-install state, `DescribePaths` coverage, never destroying content it could not read+parse) to cover skill-directory and hooks.json writes, rather than writing bespoke one-off code per agent for these two new shapes.
- Directory writes (skills) introduce a failure mode the existing file-based shapes don't have: partial writes on interrupt (some files in the skill dir written, others not) and orphaned files on uninstall if the skill's file manifest changes between versions. Design for this explicitly — write to a temp location and atomic-rename the directory into place (this project already has `internal/fsatomic` for exactly this atomic-write pattern from the githooks work; reuse it, don't reinvent it), and have uninstall delete by a manifest the install step wrote, not by "delete everything under this directory name" (which would delete user-added files if a user ever drops something into the skill directory).
- `hooks.json` (or its per-agent equivalent) is very likely to be a file some agents already use for the *user's own* hooks, unrelated to codegraph. Apply the same "no Install/Remove sequence ever destroys content it could not read+parse" invariant the githooks work (v1.0 Phase 5) converged on after two rounds of reproduced data-loss Criticals — this is directly transferable prior art from this same codebase, not a hypothetical.
- Test each of the 8 agents' conventions individually and explicitly for the new shapes, the same way `claude_test.go`, `cursor_test.go`, etc. already do for MCP config + instructions. Do not assume a convention that works for one agent (e.g., Claude Code's `.claude/skills/<name>/SKILL.md` + `.claude/settings.json` hooks) transfers unmodified to another (Cursor, Codex, opencode, Gemini, Hermes, Antigravity, Kiro) — this project's own registry exists specifically because Cursor needs `--path`, Antigravity omits `type`, Gemini uses a root-level file, Codex is global-only, and opencode needs comment-preserving JSONC. Skills/hooks conventions across 8 clients are unlikely to be more uniform than MCP config conventions already proved to be.
- Decide explicitly, per agent, whether it supports skills/hooks at all (some of the 8 may not have an equivalent concept) and make `SupportsLocation`-style capability reporting cover skills/hooks, not just MCP config location — silently no-op'ing "install" for an agent with no skill concept is correct; silently writing something malformed is not.

**Warning signs:**
- Skill-dir/hooks.json writes happen through ad-hoc `os.WriteFile`/`os.MkdirAll` calls scattered outside the `internal/agents` package, rather than through the same interface and test pattern as the existing two artifact shapes.
- No per-agent test file exists for skill/hooks install-uninstall round-trip (mirroring `claude_test.go`, `cursor_test.go`, etc.).
- Uninstall for skills deletes by directory name rather than by an install-time-written manifest.
- No test asserts byte-invariance of *unrelated* content in a shared file (e.g., a user's own hooks alongside codegraph's) across an install→uninstall round trip — the exact property the existing marker-fence tests already assert for instructions files.

**Phase to address:**
Distribution phase (the "does `codegraph install` write the skill/hooks..." decision named in PROJECT.md's Current Milestone section) — this is the single highest-risk piece of this milestone given the repo's own incident history in this exact subsystem, and should get its own deep-review pass the way githooks (v1.0 Phase 5) and the original agent registry (v1.0 Phase... /main agent work) both did, not be folded silently into the skill-authoring phase as an afterthought.

---

### Pitfall 6: The rewritten `instructions` string and marker block repeat the exact "Phase 3" broken-promise pattern that motivated this milestone

**What goes wrong:**
`internal/agents/instructions.go` currently states the marker block "explicitly defers full tool guidance to the MCP initialize response (Phase 3)" — a promise Phase 3 (v0.3.0) never fulfilled. If the v0.10.0 rewrite defers to the skill/resources ("see the codegraph skill for usage guidance") without those artifacts actually existing and being installed together, atomically, in the same release, this project ships the identical bug a third time: a claim of guidance existing somewhere else, unverified.

**Why it happens:**
Sequencing risk: the instructions-string rewrite, the skill authoring, and the resources capability are three separate pieces of work that could land in different PRs/phases. If the instructions string is rewritten (or the marker block updated) before the skill/resources it points to are actually shipped and installed, there's a window — however short — where the defer-to claim is false again, and if that window isn't closed by a single atomic release, it risks becoming permanent the way Phase 3's promise did.

**How to avoid:**
- Gate the instructions-string/marker-block rewrite behind the existence of the thing it points to, at the same commit/PR granularity `instructions_contract_test.go` already enforces for other claims — e.g., a test asserting that if the instructions string references a resource URI or a skill name, that resource/skill actually exists and is served/installed.
- Do not merge the "point to skill+resources" rewrite in a phase before the skill+resources phase completes. Sequence explicitly: resources capability and skill both shippable and verified working end-to-end BEFORE the instructions string stops carrying the old (even if stale) self-contained content.
- Add the new claim to `instructions_contract_test.go`'s existing pattern (it already asserts specific mechanisms are named in the string) — assert the skill/resource pointer resolves to something real, not just that the string contains expected substrings.

**Warning signs:**
- The instructions string is edited to say "see the codegraph skill" in a PR that does not also ship a working, installable skill.
- No test connects the instructions string's pointer-language to the actual existence of the resource/skill it points to.

**Phase to address:**
Whichever phase rewrites `internal/agents/instructions.go` and `internal/mcp/server.go`'s `instructions` constant — must be sequenced last, after skill + resources are both verified working, not first.

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|-----------------|------------------|
| Ship the skill without a transcript-diff behavior test, relying on "reads well" review | Faster to ship | Repeats Pitfall 1 silently — no signal the skill does anything until a future debug session surfaces it, same as the original `instructions` incident | Never for the initial ship; acceptable only as a fast-follow if the transcript test is already scheduled within the same milestone |
| Hand-type tool counts/flag defaults into resource prose instead of deriving them | Faster initial authoring | Repeats the SURF-01 / `instructions`-blame-index-state bug class a third time | Never — this repo has already paid for this mistake twice and the todo explicitly names it as a guard-the-claims requirement |
| Write skill-dir/hooks.json installers as one-off code outside `internal/agents`'s tested interface pattern | Avoids extending a stable interface | Reproduces the exact bug class (swallowed I/O errors, migration data-loss, delayed-deletion) already found and fixed in the existing narrower install/uninstall subsystem | Never for the primary 8-agent roster; a very short-lived spike/prototype for one agent only, before the interface extension, is fine |
| Ship the guard hook with only a false-positive suite, no true-positive suite | Half the test-writing effort | Guard silently never fires (Pitfall 4), and nobody notices because "doesn't block anything" looks like success | Never — both suites are cheap relative to the cost of a silently inert hook |

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|--------------|-----------------|-------------------|
| MCP Resources capability (`modelcontextprotocol/go-sdk`) | Adding `resources/list`/`resources/read` handlers without adding corresponding wire-oracle transcript scenarios, leaving a whole capability outside the frozen-transcript regression net that already covers `tools/list`/`initialize` | Add `resources/*` scenarios to `test/wireoracle/scenarios.go` in the same phase the capability ships, following the existing capture-then-freeze pattern |
| Claude Code / Cursor / other client hook systems (SessionStart, PreToolUse, UserPromptSubmit) | Assuming one hook schema/config-file convention works across all 8 roster agents, the same false assumption the original agent-config work had to correct for (Cursor `--path`, Antigravity no-`type`, Gemini root-level file, Codex global-only) | Research each agent's actual hook/skill convention individually before implementing, the same way the existing `AgentTarget` registry required per-agent research; do not extrapolate from Claude Code's convention to the other 7 |
| Existing `internal/agents` marker-fence mechanism (`<!-- CODEGRAPH_START/END -->`) | Introducing a second, differently-shaped marker/manifest convention for skill-dir and hooks.json content instead of reusing the hardened marker-fence + `internal/fsatomic` machinery already proven safe against the "delayed user-content deletion" bug class | Reuse `internal/fsatomic` and the marker-fence pattern (or an explicit install-time manifest for directory content) rather than inventing new file-safety primitives for the new artifact shapes |

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|-----------------|
| SessionStart nudge does synchronous filesystem/MCP-server probing on every session start | Perceptible session-start latency, especially in large monorepos or when the MCP server is cold | Make the `.codegraph/`-exists check a cheap `os.Stat`, not an MCP round-trip; never block session start on server availability | Noticeable once probing involves a network/IPC round-trip rather than a local stat, or once it runs in every session regardless of repo size |
| Guard hook runs on every single tool call (PreToolUse on all tools) rather than scoping to grep/find/Read | Adds per-tool-call latency/overhead across the entire session, not just the tool calls it cares about | Scope the hook's event/matcher as narrowly as the mechanism allows (specific tool names, or UserPromptSubmit text heuristics) rather than a blanket PreToolUse handler that inspects every call | Becomes measurable in long sessions with many tool calls, especially if the hook shells out or does non-trivial work per invocation |

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Guard/hook scripts installed into agent config directories with world-writable permissions or without validating they're not clobbering a user's own script of the same name | A malicious or buggy write could corrupt or be overwritten by unrelated tooling; silent overwrite of user content (this project's documented recurring bug class) | Reuse `internal/fsatomic` atomic-write patterns and explicit collision detection (read-before-write, same invariant githooks Phase 5 converged on) for any new hook script files |
| Resources capability exposing internal implementation details (file paths, internal env var names beyond `CODEGRAPH_MCP_TOOLS`, config internals) beyond what's needed for agent guidance | Unintended information disclosure to any MCP client that connects, since resources are readable by any authenticated session the same as tools | Scope resource content to genuinely agent-facing guidance (mirroring what the `instructions` string already exposes), not raw internal config dumps |

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-------------------|
| SessionStart nudge fires every session, even for users who've long since internalized codegraph usage | Nag fatigue — users learn to ignore or disable codegraph's hooks entirely, defeating even the cases where the nudge would have helped | Make the nudge minimal/one-line and cheap to ignore, and consider making it conditional on a repo not yet having a codegraph-tool call in the session, rather than unconditional every time |
| Guard hook's redirect message is generic ("use codegraph instead") rather than actionable | User/agent doesn't know how to comply, tries once, gets frustrated, disables the hook | Redirect message should name the specific tool and give a one-line usage example inline, mirroring the skill's own worked-example approach — consistency between hook messaging and skill content matters |
| Distribution ships skill/hooks silently as part of `codegraph install`/`upgrade` with no visible confirmation of what was added | User has no way to discover what changed in their agent config, undermining trust in an already file-write-heavy install subsystem | Surface skill/hooks additions in `install`'s existing output/summary the same way MCP config and instructions-file writes are already reported per `DescribePaths` |

## "Looks Done But Isn't" Checklist

- [ ] **Skill authored and reads well:** Often missing a behavior-verification step — verify with a real fresh-session transcript diff showing tool-choice change on a where-is-X prompt, not just a content review
- [ ] **MCP Resources capability implemented:** Often missing wire-oracle coverage — verify `resources/list`/`resources/read` have frozen transcript scenarios in `test/wireoracle/scenarios.go`, same as every other capability
- [ ] **Resource content written:** Often missing claim derivation — verify every tool count/flag default/precondition stated in resource bodies traces to a test or generated constant, not hand-typed prose
- [ ] **Guard hook installed and passes its own tests:** Often missing the true-positive suite — verify a must-fire corpus exists alongside the must-not-fire corpus, and both are demonstrated (not just written)
- [ ] **Skill/hooks distribution via `codegraph install`:** Often missing per-agent-of-8 test coverage and atomic-write safety — verify each of the 8 `AgentTarget` implementations has explicit install/uninstall round-trip tests for the new artifact shapes, mirroring `claude_test.go`/`cursor_test.go` for the existing ones
- [ ] **`instructions` string / marker block rewritten to point at skill+resources:** Often missing sequencing — verify the pointed-to skill/resources actually exist and are installed in the SAME release, not a promise for a future phase (the exact bug this milestone exists to fix)
- [ ] **`CODEGRAPH_MCP_TOOLS` documented in the new skill/resources:** Often still missing from README/`serve --help` per the original incident — verify this milestone actually closes that gap in all three places (skill, resource, and the pre-existing surfaces named in the scoping todo), not just the new ones

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|----------------|------------------|
| Inert skill (Pitfall 1) shipped and discovered later via a debug session | LOW | Rewrite the skill body leading with the decision procedure; add the transcript-diff test retroactively; no data/format migration involved |
| Resource content drifted from tool reality (Pitfall 2) | MEDIUM | Add the derivation/gating test that should have shipped with the resource; audit all existing resource content for hand-typed claims and replace with derived values; no user-facing breakage, just a trust repair |
| Guard hook shipped too aggressive, users disabling it (Pitfall 3) | LOW-MEDIUM | Narrow the trigger heuristic, add the false-positive corpus test, re-release; users who disabled it need a changelog note explaining the fix to re-enable |
| Skill-dir/hooks.json install corrupted user content on one or more of the 8 agents (Pitfall 5) | HIGH | This is the costliest recovery in this milestone's scope — mirrors the "delayed user-content deletion" Critical found in githooks Phase 5: requires forensic reconstruction of what was destroyed, a fixed install/uninstall round-trip with regression tests per agent, and likely a `codegraph doctor`-style repair path for already-affected installs |
| Instructions string points at nonexistent skill/resources (Pitfall 6) | LOW | Revert the pointer-language change or fast-follow-ship the missing skill/resources; this is a documentation-accuracy fix, not a data-safety issue |

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|-------------------|----------------|
| 1. Inert skill | Skill-authoring phase | Fresh-session transcript diff on a where-is-X prompt shows codegraph_explore chosen over grep, captured as a repeatable test/rehearsal artifact |
| 2. Resources drift from tool reality | MCP Resources phase | New wire-oracle scenarios for `resources/*`; a claim-derivation test (mirroring `instructions_contract_test.go`) covering every stated tool count/default/precondition in resource bodies |
| 3. Guard too aggressive | Hooks phase | False-positive corpus test (legitimate grep/find/Read calls) passes with zero blocks/nudges |
| 4. Guard too passive | Hooks phase | True-positive corpus test (realistic where-is-X prompts) passes with the guard demonstrably firing |
| 5. Install/uninstall corrupts new artifact shapes across 8 agents | Distribution phase | Per-agent (all 8) install→uninstall round-trip tests for skill-dir and hooks.json, asserting byte-invariance of unrelated/pre-existing content, atomic writes via `internal/fsatomic`, and manifest-based (not directory-name-based) uninstall |
| 6. Instructions string points at nonexistent skill/resources | Final rewrite phase (sequenced last) | Test asserting the instructions string's pointer-language resolves to an actually-shipped, actually-installed skill/resource; phase ordering itself (resources+skill before the rewrite) enforced by planning, not just code |

## Sources

- `/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/PROJECT.md` (primary/HIGH confidence — this repo's own documented incident history: MCP-01's AND-gate-of-three failure, the retracted 10.6% perf claim, the 51.5%-stale benchmark baseline, the v1.0 Phase 5 githooks data-loss Criticals, the v1.0 Phase 6 install/uninstall swallowed-I/O-errors and Antigravity migration data-loss findings)
- `/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/todos/pending/2026-08-08-author-a-codegraph-usage-skill-for-agents.md` (primary/HIGH confidence — the scoping todo naming the exact incident, the "agent that reads the skill and still greps first" failure mode, and the guard-the-claims requirement)
- `/Volumes/Code/github.com/seanb4t/codegraph-go/internal/mcp/instructions_contract_test.go` (code inspection/HIGH confidence — existing pattern for claim-derivation testing to extend to resources)
- `/Volumes/Code/github.com/seanb4t/codegraph-go/internal/agents/types.go`, `claude.go` (code inspection/HIGH confidence — `AgentTarget` interface's current two-artifact-shape scope, showing skill-dir/hooks.json are genuinely new territory)
- `/Volumes/Code/github.com/seanb4t/codegraph-go/test/wireoracle/oracle_test.go` (code inspection/HIGH confidence — existing frozen-transcript oracle pattern to extend to `resources/*`)

---
*Pitfalls research for: codegraph-go v0.10.0 — Agent Onboarding Skill & MCP Resources*
*Researched: 2026-08-12*
