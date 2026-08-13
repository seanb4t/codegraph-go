# Phase 7: `codegraph install` Skill + Hooks Distribution (Claude Code) - Context

**Gathered:** 2026-08-13
**Status:** Ready for planning

<domain>
## Phase Boundary

Phase 7 extends `codegraph install`/`uninstall`/`upgrade` so the Claude Code target also writes and cleanly removes the skill+hooks package Phase 6 authored (`.claude/skills/codegraph/` and `.claude/hooks/hooks.json`), sourced from the running binary's own `go:embed`'d copy of Phase 6's canonical `.claude/` — never re-authored or copied by hand. Covers AGENT-01 (write, idempotent), AGENT-02 (clean removal preserving unrelated user content), AGENT-03 (versioned, refreshed by `upgrade`), and the read-error/malformed-file invariant across the {install, uninstall} × {read-error, malformed} matrix.

**Not in scope:** authoring SKILL.md/hooks.json content (done, Phase 6), the `instructions` wire-string rewrite (Phase 8), and porting this distribution mechanism to any agent besides Claude Code — Cursor (camelCase hooks schema), Codex CLI/Antigravity (different event sets), Kiro, etc. are deferred to v2 as AGENT-04…07. The PreToolUse guard hook (GUARD-HOOK-01/02) remains a v2 fallback, not built here.

</domain>

<decisions>
## Implementation Decisions

### Location scope
- **D-01:** Skill+hooks installation **follows the existing `--location`/`-l` flag** exactly like the MCP config entry and CLAUDE.md marker block do — no special-casing. Since the flag defaults to `global`, a plain `codegraph install` with no flags places SKILL.md and hooks.json under `~/.claude/...`, not the project-local `.claude/` Phase 6 dogfooded into. — **Reversibility:** costly — **rationale:** changing this default later means migrating or leaving orphaned files for every user who already installed under the old default; the manifest (D-03/D-04) makes that migration tractable but it is not free.
- **D-02:** The consequence of a global install — the skill becoming a candidate in every Claude Code session on the machine, including unindexed repos — is **accepted as-is**, relying entirely on Phase 6's SessionStart nudge (which checks `.codegraph/` existence per-repo) and SKILL.md's own decision-procedure-first structure to keep the agent from acting on codegraph guidance where no index exists. No additional guard added in this phase.

### Version observability (AGENT-03)
- **D-03:** Version is recorded in a **sidecar manifest file** written alongside SKILL.md/hooks.json, not as a field inside SKILL.md's YAML frontmatter. Rationale, informed by web research on Claude Code's own plugin convention: Anthropic's `plugin.json` carries a bundle's version in a sidecar manifest, never in the skill's own frontmatter — SKILL.md's frontmatter schema is the portable Agent Skills standard shared byte-identically across Claude/Cursor/Codex/opencode (Phase 6 canonical_refs), and adding a nonstandard `version` key risks other agents' skill loaders choking on or silently dropping it. A sidecar also has somewhere to record hooks.json's version, which as pure JSON has no comment/metadata convention at all. — **Reversibility:** costly — **rationale:** the manifest's file path and schema become a contract `upgrade`'s auto-refresh (D-06) and any future `codegraph install --status`-style reporting depend on; changing it means a migration step for already-installed manifests.
- **D-04:** The manifest records **the binary version plus a content hash of every file codegraph wrote** (SKILL.md, hooks.json, and any other files under the embedded skill tree) — not version alone. One mechanism serves both AGENT-03's version-observability requirement and the hand-edit drift signal D-05 needs, avoiding a second bespoke comparison mechanism. Mirrors `writeMcpEntry`'s existing normalized-content-comparison idempotency pattern (`internal/agents/shared.go`), extended from a single JSON key to whole files. — **Reversibility:** reversible — additive; per-file hashes could be dropped later without breaking the version field's own meaning.

### Hand-edited content on reinstall
- **D-05:** When install/upgrade is about to overwrite a file and the manifest's stored hash shows the on-disk content no longer matches what codegraph last wrote (i.e., a hand edit happened since), the file is **silently overwritten** — no prompt, no warning, no new flag. Rationale: SKILL.md and hooks.json's codegraph-owned entries are wholly codegraph-authored package content, not user config — same posture as the CLAUDE.md marker-fenced span, which `replaceOrAppendMarkedSection` already re-applies unconditionally on every install. This keeps `install` non-interactive, consistent with the existing `--yes`/`-y` non-interactive-picker philosophy, and idempotent-to-latest by construction. — **Reversibility:** reversible — could add a warn-first or `--force`-gated variant later without touching the manifest schema.

### Upgrade auto-refresh (AGENT-03)
- **D-06:** `codegraph upgrade` — which today only swaps the binary — **reads the manifest(s) it finds and re-invokes each previously-configured target's `Install()`** with the new binary's embedded content, after the binary swap itself succeeds. This is what makes the skill package "refreshed BY upgrade" (roadmap's literal wording) true rather than requiring the user to separately remember to re-run `codegraph install`. The manifest from D-03/D-04 is what makes "which targets/locations were previously configured" answerable without new state. — **Reversibility:** costly — **rationale:** once shipped, users come to rely on upgrade keeping their skill current automatically; reverting to manual-only re-install would be a silent regression for anyone who never re-runs `install` themselves.
- **D-07:** If the auto-refresh step fails **after** the binary swap has already succeeded (e.g., a hooks.json read/parse error, which per the existing invariant must surface rather than overwrite or delete unreadable content), `codegraph upgrade` **reports the binary swap as successful** and surfaces the refresh failure as a **separate warning** naming `codegraph install` as the command to re-run — it does not make the whole `upgrade` invocation report failure. Rationale: the binary swap is atomic, verified, and independently load-bearing; conflating a config-file write hiccup with "your upgrade didn't work" would be actively misleading when the binary genuinely did update. — **Reversibility:** reversible — purely an exit-code/reporting decision, no on-disk contract.

### Claude's Discretion
- The exact hooks.json entry identity/merge strategy that lets uninstall remove precisely the entries codegraph added while preserving unrelated pre-existing entries (AGENT-02's byte-invariant round-trip against a fixture with unrelated entries) — a technical merge-algorithm question, not a vision question. Follow the existing `writeMcpEntry`/`removeMcpEntry` idempotent-JSON-merge pattern in `internal/agents/shared.go` as the closest precedent; do not invent a new file-safety primitive per the roadmap's explicit note.
- The manifest's exact filename and JSON schema (field names, nesting) beyond D-03/D-04's content requirements (version + per-file hashes) — planner's call, informed by the `go:embed` source layout.
- Whether to add a `codegraph install --status`-style reporting surface that reads the manifest — not required by AGENT-01/02/03's literal text; nice-to-have the planner may choose to include or defer.
- How `upgrade`'s auto-refresh (D-06) locates "the manifest(s) it finds" — scanning both known location roots (global `~/.claude/`, local `./.claude/`) vs. some other discovery mechanism — mechanism detail, not vision; the vision is "find what was configured and refresh it."

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Roadmap & requirements
- `.planning/ROADMAP.md` §"Phase 7: `codegraph install` Skill + Hooks Distribution (Claude Code)" — goal, 4 success criteria (AGENT-01/02/03 + the read-error/malformed-file matrix), and the explicit note: "Highest-risk phase of the milestone; a deep-review pass is warranted. Reuse `internal/fsatomic` and the existing `AgentTarget` / `recordFile` machinery — do not invent new file-safety primitives."
- `.planning/REQUIREMENTS.md` — AGENT-01, AGENT-02, AGENT-03 full text; coverage table confirms all three map only to Phase 7

### Phase 6 (dependency — artifacts being distributed)
- `.planning/phases/06-agent-skill-package-skill-md-sessionstart-nudge/06-CONTEXT.md` — **D-04**: `.claude/` is the canonical source, not a copy; Phase 7's `go:embed` directive points directly at `.claude/skills/codegraph/` and `.claude/hooks/hooks.json`. **D-03**: dogfood location (project-local `.claude/`). Both inform D-01/D-02 above.
- `.claude/skills/codegraph/SKILL.md`, `.claude/hooks/hooks.json` — the actual files this phase's `go:embed` source points at; read their current shape before writing the embed directive.

### Phase 5 (dependency — resource URIs the skill references)
- `.planning/phases/05-mcp-resources-capability-claims-drift-guard/05-CONTEXT.md` — confirms Phase 5 does not cover install/uninstall distribution (that's this phase) but SKILL.md's `codegraph://` URI references must already resolve.

### Milestone-level research
- `.planning/research/SUMMARY.md` — line 14: risk framing ("highest-implementation risk is the new skill-directory and hooks.json install/uninstall safety across 8 agent targets — this mirrors v1.0 Phase 5's githooks data-loss bugs"); line 27: Claude Code hooks.json schema and the "scope hooks delivery to Claude Code only for v1" call; line 39: SKILL.md portability vs. hooks.json non-portability; line 70: `internal/agents/skillfiles/<target>/...` embedded-tree precedent suggestion; line 166: per-agent hooks.json schema differences deferred to v2.

### Existing code — the machinery this phase extends
- `internal/agents/types.go` — `AgentTarget` interface (`Install`/`Uninstall`/`DescribePaths`/`Detect`), `WriteResult`/`FileResult`/`FileAction`/`InstallOptions`. This phase adds new `FileResult` entries to `claudeTarget`'s existing `Install`/`Uninstall`/`DescribePaths`, not a new interface.
- `internal/agents/claude.go` — `claudeTarget`'s current `Install`/`Uninstall` (MCP entry + CLAUDE.md marker block + optional AutoAllow permission) is the exact pattern this phase's new skill/hooks/manifest steps slot into, in the same function bodies.
- `internal/agents/shared.go` — `recordFile`/`recordAction` (the mandatory error-funnel, CR-01), `readJSONFile`/`writeJSONFile`/`jsonDeepEqual` (JSON read/write/compare primitives), `writeMcpEntry`/`removeMcpEntry` (JSON-key-scoped idempotent merge — closest precedent for hooks.json entry management), `replaceOrAppendMarkedSection`/`removeMarkedSection` (marker-fence precedent, not directly reusable for a whole-file artifact but the same `FileAction` reporting shape applies), `atomicWriteFile` (delegates to `internal/fsatomic`).
- `internal/fsatomic/fsatomic.go` — the atomic-write primitive every file this phase writes MUST go through (roadmap note: do not invent a new one).
- `internal/cli/install.go`, `internal/cli/uninstall.go` — CLI flag surface: `--target`/`-t` (auto|all|none|csv), `--location`/`-l` (defaults `global`), `--auto-allow`, `--yes`/`-y`. No new flags identified as necessary by this discussion.
- `internal/cli/upgrade.go`, `internal/upgrade/` — current binary-swap-only `upgrade` implementation; D-06/D-07 extend this to also drive manifest-based re-install after a successful swap.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/fsatomic.WriteFile` — atomic write primitive; every new file (SKILL.md copy, hooks.json, manifest) funnels through this.
- `internal/agents.AgentTarget` interface + `claudeTarget` — extend the existing `Install`/`Uninstall`/`DescribePaths` methods rather than introducing a parallel mechanism.
- `internal/agents.writeMcpEntry`/`removeMcpEntry` — direct precedent for "own exactly one JSON entry, preserve everything else, compare via `jsonDeepEqual` for idempotency" — the closest existing pattern for the hooks.json merge problem, even though hooks.json's shape (arrays of matcher-blocks per event) isn't a single named key like `mcpServers.codegraph`.
- `internal/agents.recordFile`/`recordAction` — the single funnel that guarantees a write/read error can never be silently swallowed (CR-01); every new code path in this phase must use it, matching the existing ~40 call sites.

### Established Patterns
- **D-07/D-08 invariants already documented in `types.go`**: re-running install twice is a byte-level no-op (`ActionUnchanged`); uninstall never errors on a target that was never installed (`ActionNotFound`). This phase's new artifact types must satisfy both, matching every existing `AgentTarget` method.
- **The read-error/malformed-file invariant** (roadmap success criterion 4, restated from v1.0 Phase 5's two reproduced data-loss Criticals): a read error or unparseable existing file makes install/uninstall surface the error rather than overwrite or delete content it couldn't read — `readJSONFile`'s existing malformed-JSON fallback-to-empty-map behavior is the shape to match for hooks.json specifically, while genuine I/O errors (permission failures, etc.) still propagate.

### Integration Points
- `go:embed` directive (new) pointing at `.claude/skills/codegraph/` and `.claude/hooks/hooks.json` — the embedded filesystem `claudeTarget.Install` reads from; per Phase 6 D-04 these paths are load-bearing and already fixed.
- `claudeTarget.Install`/`Uninstall`/`DescribePaths` in `internal/agents/claude.go` gain new steps (skill dir write/remove, hooks.json merge/unmerge, manifest write/read) alongside the three existing steps (MCP entry, CLAUDE.md marker, optional AutoAllow).
- `internal/cli/upgrade.go` / `internal/upgrade` package gains the manifest-driven re-install trigger (D-06) and the separate-warning-on-refresh-failure reporting path (D-07), invoked after a successful binary swap.

</code_context>

<specifics>
## Specific Ideas

- Phase 7 is explicitly called out in ROADMAP.md as the milestone's highest-risk phase, warranting a deep-review pass — carry that expectation into planning (extra review/verification steps, not just implementation steps).
- Web research on Claude Code's own plugin ecosystem (`plugins/README.md`, plugins-reference docs) confirmed the sidecar-manifest-for-version convention (`plugin.json`) that directly informed D-03/D-04 — this is Anthropic's own idiom, not an invented mechanism.
- Scope is deliberately narrow: Claude Code only. Every other agent target's hooks.json schema differs enough (Cursor camelCase, Codex/Antigravity different event sets, Kiro) that porting is its own future work, not something to generalize toward prematurely in this phase's design.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope. Multi-agent hooks porting (AGENT-04…07) and the PreToolUse guard hook (GUARD-HOOK-01/02) are already tracked as deliberate v2 deferrals in REQUIREMENTS.md, not new ideas surfaced here.

### Reviewed Todos (not folded)
- `2026-08-09-dry-run-signed-additions-only-diff-guard-passes-vacuously.md` (release area) — keyword match only (run/only/copy); no overlap with skill+hooks distribution.
- `2026-08-09-post-release-verify-event-aware-conclusion-guard-has-no-regression-assertion.md` (ci area) — keyword match only (event/internal/upgrade); about a CI workflow guard, not `codegraph upgrade`.
- `2026-08-10-tap-app-secret-distinctness-test-is-tautological-and-reads-no-workflow.md` (testing area) — keyword match only; unrelated to install/uninstall.
- `2026-08-10-add-golangci-lint-with-gofmt-and-idiomatic-go-linters.md` (ci area) — keyword match only (internal/files); a tooling todo, not phase-scoped.
- `2026-08-10-brew-trust-instructions-recommend-the-broader-tap-grant-with-no-security-framing.md` (docs area) — keyword match only (carry/install); about Homebrew tap trust docs, not the skill/hooks package.

</deferred>

---

*Phase: 7-`codegraph install` Skill + Hooks Distribution (Claude Code)*
*Context gathered: 2026-08-13*
