# Requirements: CodeGraph Go — v0.14.0 Polish & Agent Reach

**Defined:** 2026-09-14
**Core Value:** CodeGraph Go gives coding agents a pre-indexed code knowledge graph — fast symbol/call-path/impact queries served from a single static, verifiably-built binary, with no bundled runtime to install or manage.

**Milestone goal:** Burn down every known defect, vacuous guard, flake and doc drift; give the CLI a colour-and-structure glow-up on a streamlined verb surface; and make the codegraph skill, instructions and nudge present and discoverable in every supported agent harness, with Codex brought to parity with Claude Code.

**ID policy:** prefixes that already exist continue their numbering (`FIX-` after v0.12.0's FIX-01; `GRD-` after v0.13.0's GRD-01…06 and the named-but-declined GRD-07/08; `DOCS-` after DOCS-07; `NUDGE-` after v0.10.0's NUDGE-01/02). The v0.10.0 v2 deferrals `AGENT-04`, `AGENT-06`, `AGENT-07` are promoted under their original IDs. `AGENT-05` (Codex porting) is superseded by the `CODEX-` set and `GUARD-HOOK-01/02` by `NUDGE-03…06` — the reframe from "redirect" to "add context, never deny" is a maintainer decision (2026-09-14), recorded here so the old IDs are not mistaken for open items. New prefixes: `CLI-`, `VERB-`, `CODEX-`.

**Standing rules that bind every requirement below:** a guard is not trusted until demonstrated RED against a confirmed-applied, byte-cleanly-reverted mutation (rule `84d1gfpywd`); the agent/MCP output path stays byte-identical and ANSI-free; MCP tool names/descriptions are frozen by the wire oracle; shared-array-entry ownership is exact-identity, never shape/position (commit `242ec0a`); `.planning/` and `CHANGELOG.md` are tool-owned; no milestone git tag (D-06R).

## v1 Requirements

Requirements for this milestone. Each maps to a roadmap phase.

### Bug & Window Burn-down

- [x] **FIX-02**: Every UI route serves a codegraph favicon that loads under the unchanged `default-src 'self'` CSP — shipped as a static file, not a `data:` URI and not a widened `img-src` (WINDOWS #30)
- [ ] **FIX-03**: The install/uninstall agent picker renders its help footer in a 100×30 pane with every registered target listed, with the height budget accounting for bubbles v2 list pagination — asserted by the tmux harness, re-run after the milestone's last target-count change (WINDOWS #32)
- [x] **FIX-04**: `/graph` loads with zero uncaught page errors on this repo's index and on guava — the cytoscape-elk `notify` null TypeError fixed, or isolated with its root cause recorded (WINDOWS #26)
- [x] **FIX-05**: The cytoscape "invalid endpoints" warnings at guava scale are root-caused (which collapsed pair overlaps under ELK) and either fixed or waived with the cause on record (WINDOWS #28)
- [x] **FIX-06**: `priorCoverageGeneration` distinguishes `ErrStoreLocked`/corrupt from `ErrNotFound` and refuses (or warns) before `RemoveAll`, with a hold-the-lock-across-`index --force` regression test (WINDOWS #36 / T-10-16)
- [x] **FIX-07**: `TestRunWatchdogCancelsRunOnSimulatedReparent` passes deterministically under full-suite parallel load, fixed at the load-sensitivity cause (time-source injection or isolation), not by a wider timeout (WINDOWS #12 / GH #17)
- [x] **FIX-08**: The getppid test seam is race-free — `go test -race ./internal/daemon/...` clean, the seam per-instance or explicitly synchronized (GH #13)
- [x] **FIX-09**: `CheckRegression` compares `Metrics.Repo` and refuses a baseline/current corpus-identity mismatch, the key chosen from what the write sites actually populate so a legitimate corpus rename does not go red (GH #16)
- [x] **FIX-10**: Neither `pull_request_target` workflow expands fork-controlled file paths through a fixed heredoc delimiter; the fix is exercised against a path containing the old delimiter (GH #15)
- [x] **FIX-11**: GH #20's two perf-gate follow-ups (Namespace cache volume on 8×16; the unexplained +44.8% baseline drift) each end in a recorded decision — fixed, or closed with the measurement that justifies closing

### Guards & CI

- [x] **GRD-09**: `web:drift` is demonstrated RED against commit `98cd41dd`'s exact incident shape (build output present on disk but unstaged) on a clean checkout, closing WINDOWS #29 with the recorded cause — the CI path was never vacuous — and the local `find` enumeration of `web/build` is retained by design (WINDOWS #29)
- [x] **GRD-10**: The vendored `button.svelte` drift is isolated by a `workflow_dispatch` run under Corepack-pinned pnpm 11.23.0; the component is re-vendored or the window waived with the cause recorded (WINDOWS #31)
- [x] **GRD-11**: `check:gonum` and `check:no-force-layout` run in `ci.yml` on every pull request
- [x] **GRD-12**: `requiredCheckNames` is compared against the live `protect-main` ruleset and fails when they diverge — the `tmux e2e` context is added once the maintainer's repository-settings action lands (promotes GRD-07)
- [x] **GRD-13**: The `present` archtest catches every charm-family import path added this milestone (`colorprofile`, `fang/v2`, `x/ansi`, …) — prefix match or exact paths added in the same commit as the `go.mod` change — demonstrated RED against a planted import
- [x] **GRD-14**: Stale-open windows #16 (bench TS framing, resolved v0.11.0 Phase 6) and #33 (tmux CI human-check, PR #71 merged) are closed by recorded verification via `gsd-tools windows fixed`; #20/#21/#34 stay open as record-only deviations with that status noted

### Docs & Bookkeeping

- [x] **DOCS-08**: `docs/RELEASE.md`'s dependency paragraph states counts derived from `go.mod` at the time of writing and credits `modelcontextprotocol/go-sdk`, not `mark3labs/mcp-go` (WINDOWS #13)
- [x] **DOCS-09**: SLSA provenance is described as attested over the binaries, not the checksums file, in `release.yml` and both docs that carry the claim (GH #14)
- [x] **DOCS-10**: Root `SECURITY.md` states govulncheck's and `pnpm audit`'s actual, disjoint scope with the advisory caveat (promotes GRD-08)
- [x] **DOCS-11**: Planning bookkeeping reconciled through tool-sanctioned writers where one exists: STATE.md's Pending Todos table matches `.planning/todos/`, and SEED-001's frontmatter records its consumption by v0.12.0 (the SEED-002 precedent applies where no writer verb exists)

### CLI Glow-up

- [x] **CLI-01**: Every human-output verb — `explore`, `search`, `node`, `callers`, `callees`, `impact`, `affected`, `files`, `status`, `init`, `index`, `sync`, `daemon`, `install`, `uninstall`, `upgrade`, `serve` (non-MCP output), `githooks`, `uninit`, `version`, `ui` — renders through a `present` renderer using one shared semantic palette (header, label, value, path, count, warning, error as distinct hues), consuming existing plain-struct seams (`NodeDetail`, `ExploreDetail`, `StatusResult`, …) with zero changes to `internal/query` or `internal/mcp`
- [x] **CLI-02**: Colour fidelity is downsampled via `colorprofile` at the RunE boundary, never inside `present`; `TERM=dumb`, 16-colour, 256-colour and truecolor terminals each render correctly
- [x] **CLI-03**: `--color=auto|always|never` exists on every styled verb with precedence `--color=always|never` > `NO_COLOR` (any non-empty value disables) > `CLICOLOR_FORCE` (forces) > `CLICOLOR=0` (disables) > TTY auto-detect, covered by a unit-test matrix over every combination
- [x] **CLI-04**: The palette is adaptive — `HasDarkBackground` read once at the call site — and every hue is readable on light and dark backgrounds, verified against a light theme (Solarized Light or macOS light Terminal) and a dark one
- [x] **CLI-05**: The agent/MCP path, `--json` output and non-TTY (piped) output are byte-identical before and after the glow-up — golden oracle and wire oracle unchanged, the TUI-01 archtest holding, and a regression test pins `NO_COLOR` + non-TTY plain output (the gh #13335 lesson)
- [x] **CLI-06**: `codegraph --help` groups commands into titled sections via `cobra.Group` (Query the graph / Build the index / Agents & serving / Maintenance), files `help` and `completion` into a group, and `<verb> --help` is styled consistently with root help
- [x] **CLI-07**: Short flags are consistent across the query verbs — `-j`, `-l`, `-k`, `-p` present wherever the long form exists
- [x] **CLI-08**: `charm.land/fang/v2` is spiked first with a recorded verdict; it is adopted when `WithoutManpage()`/`WithoutCompletions()` compose with the existing hidden `man` and cobra completions, `serve --mcp`'s transcript is byte-identical under the wrapper, and the SBOM/govulncheck delta is acceptable — otherwise the help template is hand-rolled; either way the decision precedes any renderer landing

### Verb Fold

- [x] **VERB-01**: `search --full` returns full node records — the `MarshalQueryJSON` envelope under `--json`, and a human branch that shows signature and qualified name — while default `search` output (locations) is unchanged
- [x] **VERB-02**: `search` accepts the flag superset of the removed `query` (`-j`, `-l`, `-k`, `-p` short forms) with a flag-parse test covering both forms
- [x] **VERB-03**: `codegraph query …` is a hidden dedicated stub that prints `"query" has been renamed to "search --full"` (or equivalent) to stderr and returns a non-nil error — exit non-zero, nothing executed; cobra's `Deprecated` field is not used
- [x] **VERB-04**: `unlock` becomes `daemon unlock` with identical flags and behaviour, and `codegraph unlock` is the same kind of stub naming `daemon unlock`
- [x] **VERB-05**: A positive-controlled, word-boundary, multiline census finds zero references to `codegraph query` / `codegraph unlock` outside the stubs across README, docs, SKILL.md, `internal/mcp/resources/*.md`, hook scripts, Taskfile, CI and tests — run before `internal/cli/` is edited and again after
- [x] **VERB-06**: `docs/CLI-REFERENCE.md` is regenerated with the stubs allowlisted by reason; `task docs:cli:drift` and `TestEveryRegisteredFlagIsAccountedFor` are green; shell completions and man pages reflect the new surface
- [x] **VERB-07**: Goldens affected by the fold are re-frozen in a reviewed diff with a RED demonstration; the 8-tool MCP set and the wire oracle's transcripts are unchanged
- [x] **VERB-08**: The rename lands as a `feat!:` conventional commit with the stubs' removal in the following minor recorded in the release notes and in `docs/CLI-REFERENCE.md`

### Agent Reach

- [x] **AGENT-08**: `AgentTarget` exposes per-target capabilities (skill directories, instructions path, hook mechanism, supported scopes) so `install`, `uninstall`, `Detect` and `--print-config-style` derive from one table rather than eight re-implementations
- [x] **AGENT-09**: The skill package is written once to `.agents/skills/codegraph/` (project) and `~/.agents/skills/codegraph/` (global) with its sidecar manifest, and each harness documented as reading that path is verified live to discover it; a harness-specific directory is written only where a live session shows the shared path is not read
- [x] **AGENT-04**: Cursor — skill package installed; repo-root `AGENTS.md` pickup probed in a live Cursor session before any Cursor-specific instructions target is written, the probe's outcome recorded either way
- [x] **AGENT-06**: opencode — skill package installed at a path opencode reads, SKILL.md frontmatter compatibility verified, the existing instructions block retained
- [x] **AGENT-07**: Antigravity — skill package via `.agents/skills/` and the instructions block in `AGENTS.md`
- [x] **AGENT-10**: Gemini CLI — skill package installed to `.gemini/skills/` (project) and its global equivalent, discoverable via `activate_skill`, the existing instructions block retained
- [x] **AGENT-11**: Kiro — skill package installed to `.kiro/skills/` (project) and `~/.kiro/skills/`, `AGENTS.md` retained as a steering source
- [x] **AGENT-13**: Every new per-harness write uses exact-identity ownership — a planted-foreign-entry test per harness proves an unrelated sibling entry is never overwritten — and `uninstall` reverses every write leaving unrelated content byte-identical (commit `242ec0a` cited in review)
- [ ] **AGENT-14**: A per-harness capability table (what each of the 8 targets receives: MCP config, instructions, skill, nudge, scopes) is published in the docs and kept honest — `[ASSUMED]` where live verification was not possible — with the `instructions.go` "4 of 8" comment and the MCP `instructions` skill sentence updated to match what ships

### Nudge Hook (Claude Code)

- [ ] **NUDGE-03**: A PreToolUse hook on `Bash` (grep/rg/find as the first binary in the pipe), `Grep`, `Glob` and `Read` in a `.codegraph/`-indexed repo returns only `hookSpecificOutput.additionalContext` pointing at `codegraph_explore`, exits 0 unconditionally, and never emits `permissionDecision` — verified against the current hooks reference, not memory
- [ ] **NUDGE-04**: The nudge fires on the first matched call, then at most once a minute, per session and separately per subagent (maintainer decision 2026-09-19, replacing "at most once per session"), via a session-scoped timestamp sentinel, and is silent with zero overhead in a repo without `.codegraph/` (NUDGE-02's property preserved)
- [ ] **NUDGE-05**: The hook is validated against a false-positive corpus (legitimate grep use) and a true-positive corpus (where-is-X patterns), and its fire rate is measured in a genuinely fresh live session (carries GUARD-HOOK-02)
- [ ] **NUDGE-06**: `codegraph install`/`uninstall` register and remove the hook through the existing exact-identity `writeHookEntry` as an opt-in alongside the default SessionStart nudge; a hand-edited own entry duplicates rather than overwrites

### Codex Parity

- [ ] **CODEX-01**: Before any `codex.go` change, a live verification in a scratch trusted project records — with dated citations — whether the current Codex CLI loads a project-scoped `.codex/config.toml`, which skill path(s) it reads (`.agents/skills/` vs `.codex/skills/`), and whether `hooks.json` works behind `features.hooks`; `codex.go`'s "no per-project config" comment is corrected in the same commit
- [ ] **CODEX-02**: `SupportsLocation(LocationLocal)` is true; project-local `install` writes `.codex/config.toml` through the TOML splice (existing tables, inline tables, comments and CRLF preserved) and tells the user the project must be trusted; `--target auto` detection and the agent picker reflect the new scope
- [ ] **CODEX-03**: The skill package is installed to the verified Codex skill path(s) at both scopes with its sidecar manifest, idempotently and byte-invariant against sibling content
- [ ] **CODEX-04**: Project-local install writes the marker-fenced instructions block into the repo-root `AGENTS.md` (global `~/.codex/AGENTS.md` behaviour unchanged) and `uninstall` removes it leaving the rest of the file byte-identical
- [ ] **CODEX-05**: A Codex PreToolUse nudge in `hooks.json` carries the same additionalContext-only contract as NUDGE-03/04 (first matched call, then at most once a minute per session and per subagent — the 2026-09-19 amendment of NUDGE-04); it is installed only when Codex's hooks feature is enabled (opt-in), skipped with a message otherwise, and its experimental status is stated in the docs
- [ ] **CODEX-06**: A genuinely fresh Codex session in an indexed repo reaches for codegraph unprompted — the skill is listed, and the MCP tool is called or `codegraph explore` is run — with the evidence recorded to the v0.10.0 live-session standard; `codex mcp list` shows the entry at both scopes

## v2 Requirements

Deferred to a later milestone. Tracked but not in this roadmap.

### Nudge hooks for other harnesses

- **NUDGE-07**: Cursor PreToolUse-equivalent nudge via `.cursor/hooks.json` — documented mechanism; non-blocking output shape unverified against primary docs
- **NUDGE-08**: Kiro nudge via `.kiro/hooks/<id>.json` `{"type":"agent"}` — the cleanest non-blocking primitive found; deferred by maintainer decision (2026-09-14) to keep this milestone's hook work to Claude Code + Codex
- **NUDGE-09**: Gemini CLI nudge via `.gemini/hooks/` shell scripts
- **NUDGE-10**: opencode / Antigravity nudges — mechanisms `[ASSUMED]`, no primary docs found this pass

### Agent reach, remaining

- **AGENT-12**: Hermes — no public documentation for config, skill or hook mechanisms was found; nothing beyond the existing MCP config is claimed until verified by non-web means

### Carried from earlier milestones (unchanged)

- GRF-07 opt-in whole-symbol graph (parked on the guava non-convergence evidence); Team Scale; SEED-003 markdown in the index; DIST-06 stapled offline-safe container; BREW-07 homebrew-core; MRTR-01 elicitation; GH #23 gsd-pi install target; GH #9 open-gsd workflow chore

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| A blocking guard hook (`permissionDecision: deny`/`ask`, exit 2) | Maintainer reframe 2026-09-14: the hook adds context and never denies; a heuristic false positive must not block a real tool call |
| A hidden alias that silently forwards `query` to `search --full` | Changes default output shape for anyone still typing `query` and gives scripts no signal; the loud non-zero stub is the decision |
| cobra's `Command.Deprecated` for the stubs | Prints a warning but still executes the command — the opposite of "exits non-zero, nothing executed" |
| Renaming or re-describing MCP tools | The wire oracle's transcripts pin the 8-tool set; agent-facing surface is frozen |
| `img-src data:` in the SPA CSP to make the favicon load | Widens the policy for a cosmetic asset; the asset choice is the defect, not the CSP |
| Widening the watchdog timeout to fix WINDOWS #12 | Masks load-sensitivity; the fix is the time source or isolation |
| Hand-rolling a help template before the fang verdict | Adopting fang afterwards throws that work away (FEATURES.md dependency) |
| Nudge hooks for Cursor, Kiro, Gemini, opencode, Antigravity, Hermes | Deferred to v2 by maintainer decision; skill + instructions still ship there |
| Fold `sync` into `index --sync` or `callers`/`callees`/`impact` into `node` flags | "Fold harder" option declined 2026-09-14; those verbs are distinct MCP tools too |
| Styling the MCP/agent output path or `--json` | Byte-identity is a milestone invariant |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| FIX-02 | Phase 1 | Complete |
| FIX-03 | Phase 7 | Pending |
| FIX-04 | Phase 1 | Complete |
| FIX-05 | Phase 1 | Complete |
| FIX-06 | Phase 1 | Complete |
| FIX-07 | Phase 1 | Complete |
| FIX-08 | Phase 1 | Complete |
| FIX-09 | Phase 1 | Complete |
| FIX-10 | Phase 1 | Complete |
| FIX-11 | Phase 1 | Complete |
| GRD-09 | Phase 2 | Complete |
| GRD-10 | Phase 2 | Complete |
| GRD-11 | Phase 2 | Complete |
| GRD-12 | Phase 2 | Complete |
| GRD-13 | Phase 4 | Complete |
| GRD-14 | Phase 2 | Complete |
| DOCS-08 | Phase 2 | Complete |
| DOCS-09 | Phase 2 | Complete |
| DOCS-10 | Phase 2 | Complete |
| DOCS-11 | Phase 2 | Complete |
| CLI-01 | Phase 4 | Complete |
| CLI-02 | Phase 4 | Complete |
| CLI-03 | Phase 4 | Complete |
| CLI-04 | Phase 4 | Complete |
| CLI-05 | Phase 4 | Complete |
| CLI-06 | Phase 4 | Complete |
| CLI-07 | Phase 4 | Complete |
| CLI-08 | Phase 4 | Complete |
| VERB-01 | Phase 3 | Complete |
| VERB-02 | Phase 3 | Complete |
| VERB-03 | Phase 3 | Complete |
| VERB-04 | Phase 3 | Complete |
| VERB-05 | Phase 3 | Complete |
| VERB-06 | Phase 3 | Complete |
| VERB-07 | Phase 3 | Complete |
| VERB-08 | Phase 3 | Complete |
| AGENT-08 | Phase 5 | Complete |
| AGENT-09 | Phase 5 | Complete |
| AGENT-04 | Phase 5 | Complete |
| AGENT-06 | Phase 5 | Complete |
| AGENT-07 | Phase 5 | Complete |
| AGENT-10 | Phase 5 | Complete |
| AGENT-11 | Phase 5 | Complete |
| AGENT-13 | Phase 5 | Complete |
| AGENT-14 | Phase 7 | Pending |
| NUDGE-03 | Phase 6 | Pending |
| NUDGE-04 | Phase 6 | Pending |
| NUDGE-05 | Phase 6 | Pending |
| NUDGE-06 | Phase 6 | Pending |
| CODEX-01 | Phase 7 | Pending |
| CODEX-02 | Phase 7 | Pending |
| CODEX-03 | Phase 7 | Pending |
| CODEX-04 | Phase 7 | Pending |
| CODEX-05 | Phase 7 | Pending |
| CODEX-06 | Phase 7 | Pending |

**Coverage:**

- v1 requirements: 55 total
- Mapped to phases: 55
- Unmapped: 0 ✓

---
*Requirements defined: 2026-09-14*
*Last updated: 2026-09-14 after roadmap creation (traceability filled: 7 phases)*
