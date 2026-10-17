# Research Summary — v0.14.0 Polish & Agent Reach

**Researched:** 2026-09-14  
**Scope:** CLI styling glow-up + multi-harness agent reach (skills/instructions/nudge hooks) + Codex parity + verb fold + bug burn-down, on an existing shipped Go CLI/MCP/UI codebase.

## Executive Summary

v0.14.0 is a brownfield polish + reach milestone on top of a shipped v0.13.0 codebase. The four research tracks converge on three key reconciliations: (1) **Codex CLI's config/skills/hooks surface changed during 2026** — `codex.go`'s existing "no per-project config" assumption is now stale, and Codex's hooks mechanism is shaped nearly identically to Claude Code's but is experimental/disabled-by-default and platform-limited (Windows unsupported); (2) **fang adoption is viable but has two hard requirements** — collision-awareness (`WithoutManpage()`, `WithoutCompletions()`) and archtest-guard maintenance (the current denylist is a fixed literal list that won't auto-catch new Charm dependencies); (3) **build order matters** — the verb fold and reach-capability-model work must come before the CLI glow-up and Codex parity, both because they're prerequisites and because they share the same code surfaces (`search.go`, `daemon.go`).

The milestone is achievable within the scoped four themes (bug burn-down, CLI glow-up, agent reach, Codex parity) but has **six critical assumptions marked [ASSUMED]** that must be verified live before committing implementations: Codex's project-local config stability, skill-directory path (`.agents/skills/` vs `.codex/skills/`), hook feature-flag status, Cursor's AGENTS.md auto-pickup, the exact non-blocking hook output shape per-harness, and Hermes' hook/skill mechanism entirely (no public docs found). Each is included below; none blocks the milestone start but several are critical for phase sequencing decisions.

## Key Findings

### From STACK.md

**Core technologies (no new Go dependencies for capabilities 1–3):**
- `charm.land/lipgloss/v2@v2.0.5` (already pinned) — styling engine; v2 delegates color downsampling to `colorprofile`.
- `github.com/charmbracelet/colorprofile@v0.4.3` (already indirect, promote to direct) — terminal capability detection + downsampling; required for real colour glow-up.
- `github.com/spf13/cobra@v1.10.2` (already pinned) — has built-in `Group`/`AddGroup`/`GroupID`/`SetHelpCommandGroupID` for grouped help; no version bump needed.
- `charm.land/fang/v2@v2.0.1` (NEW, evaluate only) — styled help/error/version chrome; brings lipgloss-based styling at the Execute boundary. **Collision risk:** ships its own `man` and `completion` subcommands matching this repo's existing commands; adoption requires `fang.WithoutManpage()` and `fang.WithoutCompletions()` + archtest denylist update.

**Critical reconciliation — Codex CLI surface (verified 2026-09-14):**
- **Project-scoped `.codex/config.toml`** — Now exists (trust-gated), contradicting `codex.go`'s current doc comment ("no per-project config concept"). `SupportsLocation(LocationLocal)` should flip to `true`; must be re-verified live against Codex CLI's own trust model before implementation.
- **Skills directory** — Official docs cite `.agents/skills/` / `~/.agents/skills/` (open-agent-skills standard). Community guides cite `.codex/skills/`. **[ASSUMED]**: `.agents/skills/` is primary target; verify which path(s) a live Codex session actually reads before finalizing the install path.
- **Hooks mechanism** — `~/.codex/hooks.json` or `<repo>/.codex/hooks.json` (project, trust-gated), gated behind `features.hooks = true`. Events: `SessionStart`, `PreToolUse`, `PostToolUse`, `UserPromptSubmit`, others. Output shape: `{"hookSpecificOutput":{"hookEventName":"...","additionalContext":"..."}}` (non-blocking context injection, no `permissionDecision` on the nudge path). **Load-bearing caveat:** experimental, disabled by default, no Windows support. **[ASSUMED]**: Project-local hook file is stable enough to ship; must be verified against Codex's own current status before committing to a PreToolUse implementation.

**Other harnesses — skill/instructions/hook surface (capability 5):**
- **Cursor** — `.cursor/skills/`, `.agents/skills/` (project); `~/.cursor/skills/`, `~/.agents/skills/` (global); reads `AGENTS.md` at repo root; `.cursor/hooks.json` (project-level hooks introduced v1.7+). Non-blocking-nudge output shape **[ASSUMED]** pending explicit docs fetch during planning.
- **Gemini CLI** — `.gemini/skills/` (project); native `SKILL.md` + discovery via `activate_skill` tool; `.gemini/hooks/` shell scripts (strict stdout-only JSON discipline); no AGENTS.md-native read detected.
- **opencode** — `.opencode/`, `.claude/skills/`, `.agents/skills/` (project); `~/.config/opencode/skills/`, `~/.claude/skills/`, `~/.agents/skills/` (global); AGENTS.md-native; hook mechanism via JS plugins **[ASSUMED]** pending direct opencode.ai/docs/plugins/ fetch.
- **Kiro** — `.kiro/skills/` (project); `~/.kiro/skills/` (personal); AGENTS.md as a steering source; `.kiro/hooks/<id>.json` with `{"type":"agent"}` action supporting non-blocking prompt-injection; hooks documentation current (Sept 2, 2026).
- **Antigravity** — `.agents/skills/` (explicitly open-agent-skills standard); AGENTS.md at project root; hooks mechanism location/schema **[ASSUMED]** (no official docs found); launched 2026-05-19, newest/thinnest documentation base.
- **Hermes** — No public documentation found for config/hook/skill mechanisms. **[ASSUMED unchanged]** from existing `hermes.go` comment; flagged as requiring non-web-search verification if the milestone needs Hermes hook parity.

### From FEATURES.md

**Verb fold dependencies:** `query` → `search --full`, `unlock` → `daemon unlock` must land BEFORE styled renderers (both themes touch `search.go`/`daemon.go`).

**Codex parity blocking point:** Re-verify project-local-config claim FIRST; gates whether Codex parity is "add new write path" or "confirm global-only still correct."

**Fang evaluation must decide BEFORE touching styles.go** — adopting after hand-rolled grouped-help means throwing work away.

### From ARCHITECTURE.md

**Build order (hard dependencies):**
```
1. Verb fold FIRST               2. Reach capability model FIRST
   (search --full, daemon unlock,   (AgentTarget.Capabilities(),
    hidden stubs, allowlist)        per-harness doc verification)
        │                                    │
        ▼                                    ▼
3. CLI glow-up SECOND            4. Codex parity SECOND
   (present.Render* once against    (first real consumer of
    FINAL verb surface;             capability model; project-
    --color flag; fang eval)        local scope verified live)
        │                                    │
        └──────────────┬─────────────────────┘
                       ▼
             5. Cross-cutting close-out
                (SKILL.md/resources docs naming search --full;
                 re-freeze CLI-REFERENCE + allowlist;
                 live-session verification per harness)
```

**Rationale:** Verb fold before glow-up: `search.go`/`daemon.go` are touched by both themes. Reach capability model before Codex parity: Codex is the first consumer of the new interface method. Bug burn-down orthogonal throughout except WINDOWS #32 (picker footer) before Codex-local-scope flip (affects target count).

**Fang integration risk:** Process-level wrapper at `Execute()` boundary; **requires wire-oracle-style byte-identical transcript proof for `serve --mcp` before adoption**. Must add fang's import path to `forbiddenImportPaths` denylist in same commit (see Pitfall 3).

### From PITFALLS.md

**Critical pitfalls (must be actively prevented):**

1. **Lipgloss v2 colour downsampling gap** (Pitfall 1) — Must route through `colorprofile.Writer`/`lipgloss.Println` at RunE boundaries, never inside `present` (D-03). Guard: Manual render check in `TERM=xterm`; test asserting `colorprofile`-derived fidelity.

2. **NO_COLOR/CLICOLOR_FORCE precedence** (Pitfall 2) — Correct order: NO_COLOR wins over all. Guard: Unit test matrix covering all 4 combinations set/unset.

3. **`present/archtest` denylist rot** (Pitfall 3) — Fixed literal list won't auto-catch fang/colorprofile/x/ansi. Must add exact import paths in same commit as `go.mod` changes, or convert to `charm.land/` prefix match. Guard: RED-then-fixed mutation log.

4. **`query`→`search --full` merge loses flags** (Pitfall 4) — Keep flag superset (`-j/-l/-k`), gate JSON shape behind `--full`. Guard: Flag-parse test covering both verbs and both flag forms.

5. **Verb-rename prose census gaps** (Pitfall 5) — Must search whole repo for `codegraph query`, `codegraph unlock` BEFORE editing `internal/cli/` (word-boundary, not package identifiers). Guard: Zero hits outside stub itself.

6. **Codex config-surface assumptions stale** (Pitfall 11) — Update `codex.go`'s doc comment in same commit as any code change, dated citation to current Codex docs. Re-verify: project-local file location, skill path precedence, PreToolUse hook existence.

7. **PreToolUse nudge accidentally blocking** (Pitfall 8) — Emit ONLY `additionalContext`, no `permissionDecision` key, exit 0 unconditionally. Guard: Test asserting exit 0 always, never blocks.

8. **Nudge firing on every unrelated call becomes noise** (Pitfall 9) — Gate on two things: (1) `.codegraph/` exists, AND (2) content heuristic (identifier-shaped grep, bash rg/grep/find only). Guard: Fresh-session transcript measuring fire-rate.

9. **Shared-array-entry ownership reintroduced per-harness** (Pitfall 10) — **Exact-identity matching**, never shape-based. Commit `242ec0a` reverted this; write ownership-identity test FIRST per harness. Guard: Planted-foreign-entry test, review cites `242ec0a`.

10. **`web:drift` gate still vacuous** (Pitfall 13) — Implement paired-assertion form (filesystem set == git-tracked set, both directions), not just delimiter rename. Guard: Mutation-log replay of commit 98cd41dd's exact incident.

11. **Daemon watchdog flake fixed by timeout** (Pitfall 14) — Load-sensitivity, not tightness. Fix by time-source injection or test isolation, not wider timeout. Guard: Comment cites WINDOWS #12.

12. **`getppid` global var data race** (Pitfall 15) — Make seam per-instance or explicitly synchronized. Guard: `go test -race ./internal/daemon/...` clean.

## Reconciliation of Research Disagreements

### 1. Codex CLI Hooks Mechanism

**Reconciled position:** Codex DOES have hooks (PreToolUse included) with output shape identical to Claude Code's. **Experimental, disabled by default, Windows-unsupported.** Must be feature-flagged in messaging. STACK found direct evidence; ARCHITECTURE wrote before that finding; FEATURES correctly flags caveats; PITFALLS marked unverified. **Live verification needed:** Confirm project-trust-gating model before implementing project-local hook writes.

### 2. Codex Skill Directory Path

**Reconciled position:** Use `.agents/skills/` as primary target (official docs). Verify in live Codex session which path(s) actually read; community guides describe `.codex/skills/` as also possibly read. Both sources defer to live verification — correct approach.

### 3. Fang Adoption Collision Findings

**Reconciled position:** All three findings are complementary, not contradictory. Adoption requires: (1) collision-awareness (`WithoutManpage()`, `WithoutCompletions()`); (2) MCP boundary proof (byte-identical `serve --mcp` transcript); (3) archtest denylist maintenance (add fang to `forbiddenImportPaths` in same commit). None are blockers, all are load-bearing.

### 4. Build Order Sequencing

**Merged order:**
- **Phase 0 (pre-Phase-1):** Re-verify Codex project-local claim, decide fang vs hand-rolled, probe Cursor AGENTS.md auto-pickup.
- **Phase 1:** Verb fold + Reach capability model (parallel independent tracks).
- **Phase 2:** CLI glow-up + Codex parity (parallel, each depends on own Phase 1 prerequisite).
- **Phase 3:** Cross-cutting close-out and verification.
- **Bug burn-down:** Orthogonal throughout; sequence WINDOWS #32 after Phase 2 Track B.

## Confidence Assessment

| Area | Confidence | Caveats |
|------|------------|---------|
| **Stack** | HIGH | Codex changes during 2026; skill directory `.codex/skills/` vs `.agents/skills/` marked [ASSUMED] |
| **Features** | HIGH | Codex hook experimental/disabled-by-default; opencode plugin mechanism from third-party only |
| **Architecture** | HIGH (repo source); MEDIUM (fang runtime, Codex precedence) | MCP boundary with fang requires explicit verification; Codex trust-gating [ASSUMED] |
| **Pitfalls** | HIGH (repo source); MEDIUM (stack/arch claims); LOW-MEDIUM (per-harness config) | Hermes found no public docs; Codex experimental status highest-uncertainty |

## Open Questions (marked [ASSUMED])

1. Codex project-local config stability — re-verify against `openai/codex`'s current docs.
2. Codex skill directory — verify `.agents/skills/` vs `.codex/skills/` in live session.
3. Codex trust-gating behavior — does install phase message warn user about trust requirement?
4. Cursor AGENTS.md auto-pickup — does it actually read repo-root block without Cursor-specific code?
5. Per-harness non-blocking hook output shape — do all harnesses have equivalent `additionalContext`-like fields?
6. Hermes hook/skill mechanism — no public docs found; treat existing comment as still accurate.
7. Claude Code PreToolUse matcher grammar — verify exact syntax before authoring hook script.
8. Codex PreToolUse non-blocking field name — verify exact field before Phase 2 Codex parity task.

---

*Research summary for: CodeGraph Go v0.14.0 "Polish & Agent Reach"*  
*Researched: 2026-09-14*
