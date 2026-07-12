# Phase 6: Agent Integrations & CLI Lifecycle - Context

**Gathered:** 2026-07-12
**Status:** Ready for planning

<domain>
## Phase Boundary

This phase delivers **the mechanics of the drop-in swap**: a user who runs TS CodeGraph today can point their 8 coding agents at the Go binary, keep it current, and rely on complete CLI ergonomics. Everything here is **agent-facing plumbing and CLI lifecycle** — it does NOT touch the graph, the indexer, the query engine, or the MCP tool surface (all frozen from Phases 2–5).

**In scope (AGNT-01/02/03, CLI-01/02/03):**
1. **`codegraph install`** — detect and configure the **8-agent roster** (Claude Code, Cursor, Codex CLI, opencode, Hermes, Gemini CLI, Antigravity, Kiro): write per-agent MCP server config **plus** marker-fenced instruction injection, **idempotent** on re-run. (AGNT-01)
2. **`codegraph uninstall`** — cleanly reverse everything `install` wrote (MCP entry, marker block, permissions), **preserving user edits outside the markers**. (AGNT-02)
3. **Per-agent quirks** handled via an `AgentTarget`-style registry — most notably Cursor's injected `--path` arg for the MCP subprocess cwd. (AGNT-03)
4. **`codegraph version`** + **`codegraph help [command]`** with standard CLI ergonomics. (CLI-01)
5. **`codegraph upgrade [version] [--check]`** — signature-verified binary download-and-swap. (CLI-02)
6. **`codegraph telemetry`** — reports that this build contains zero telemetry / phone-home code. (CLI-03)

**Out of scope (belongs to other phases / never):**
- **Any graph write / index / sync / query / MCP-tool behavior** — frozen from Phases 2–5. `install` only writes *external* agent config files that point at `codegraph serve --mcp`; it does not read or mutate `.codegraph/`. The MCP `initialize` response (Phase 3) remains the single source of truth for tool guidance — `install`'s injected block is a *short pointer to `codegraph explore`*, not the full playbook.
- **The actual signed/attested release artifacts + SLSA provenance + SBOM** — Phase 8 (DIST-02/03/04). Phase 6 builds the `upgrade` **verify-then-swap** client path; it consumes what Phase 8 will publish. The release URL + cosign signing identity are wired as constants that Phase 8 finalizes (see D-14).
- **The migration tool** (`.codegraph/` SQLite → new format) — Phase 7 (MIGR-01/02).
- **Benchmarks, 100k-file scale, peak-RSS** — Phase 8 (PERF-*, INDX-06).
- **Telemetry collection of any kind** — permanently out of scope (a stronger trust story than mere TS parity); `telemetry` exists precisely to *prove the negative*.
- **New MCP tools, HTTP/SSE transport, remote/multi-user** — v2 (SERVER-01, BEYOND-01).

</domain>

<decisions>
## Implementation Decisions

> Auto-resolved in `--auto` mode. Each decision is the recommended default grounded in: the **TS CodeGraph v1.3.x installer behavior** (the parity oracle — captured via DeepWiki of `colbymchenry/codegraph`, see canonical refs), the existing Cobra thin-CLI substrate (`internal/cli`, `serve --mcp --path`), the project's zero-telemetry + single-static-binary + minimal-audited-deps constraints, and the supply-chain stack in `.claude/CLAUDE.md` (cosign keyless, GitHub Releases, SBOM).

### Parity Model — no golden corpus (AGNT-*, mirrors Phase-3 D-07a)
- **D-01:** **Parity is behavioral, measured against the TS installer *source*, not a golden-output corpus.** `install`/`uninstall` write to external agent config files (JSON/JSONC/TOML/Markdown), so there is no `codegraph`-emitted output to diff. The acceptance oracle is: for each agent, the config file ends in the same *shape* TS produces (same keys, same marker block, same quirk handling), and a round-trip `install`→`uninstall` returns the file to its pre-install bytes modulo the CodeGraph section. Researcher/planner assert on **structural correctness of the written config**, not on a captured fixture.
- **D-01a:** **Marker fences reproduce TS exactly** — `<!-- CODEGRAPH_START -->` … `<!-- CODEGRAPH_END -->`. This is a hard parity contract, not a fresh choice: a Go `uninstall` must recognize a marker block a *TS* `install` wrote (the drop-in-swap user installed with TS, uninstalls with Go), and vice-versa. Do not invent new marker text.

### Agent Registry & Command Surface (AGNT-01, AGNT-03)
- **D-02:** **`codegraph install` / `codegraph uninstall` are two new top-level Cobra commands** added to `internal/cli/root.go` alongside the existing tree, following the Phase-2/3 thin-CLI pattern (command resolves flags/paths + delegates). All agent logic lives in a **new `internal/agents` (or `internal/installer`) package** exposing an **`AgentTarget` interface** — `Detect() bool`, `Install(cfg) → changed`, `Uninstall() → status`, plus metadata (id, display name, config path resolver, instruction-file path resolver). One implementation per roster agent; the two commands iterate the registry. This is the clean home for AGNT-03 per-agent quirks.
- **D-03:** **Flag surface mirrors TS:** `--target auto|all|none|<csv-of-ids>` (default `auto`) and `--location global|local` (per-agent default; see D-06). `auto` = configure only **detected** agents (detection = existence of the agent's config dir/primary file). When invoked with a TTY and **no `--target`**, present an **interactive multi-select prefilled with detected agents** (reuse/extend the `confirm()` helper idiom in `internal/cli/uninit.go`); with **no TTY or under `--auto`/CI**, default to `auto` **without prompting** (never block a non-interactive run on a prompt).
- **D-04:** **The MCP config `install` writes invokes this binary by absolute path** — resolve `os.Executable()` at install time and record it as the MCP server command (args: `serve --mcp`, plus `--path …` for Cursor). This is what makes the drop-in swap real: the agent launches *the Go binary the user just ran `install` from*, not a `PATH` guess.

### Per-Agent Config Matrix (AGNT-01, AGNT-03) — the parity table
- **D-05:** **The 5 TS-covered agents get their exact TS paths/formats** (parity, verified against the reference):
  | Agent | Global config | Local config | Format / key | Instruction file (marker block) |
  |---|---|---|---|---|
  | Claude Code | `~/.claude.json` | `./.mcp.json` | JSON, `mcpServers.codegraph` | `~/.claude/CLAUDE.md` / `./.claude/CLAUDE.md` |
  | Cursor | `~/.cursor/mcp.json` | `./.cursor/mcp.json` | JSON, `mcpServers.codegraph` | **none** (no rules file) |
  | Codex CLI | `~/.codex/config.toml` | *(global-only)* | TOML, `[mcp_servers.codegraph]` | `~/.codex/AGENTS.md` (global only) |
  | opencode | `~/.config/opencode/opencode.jsonc` | `./opencode.jsonc` | JSONC, `mcp.codegraph` (+`$schema`,`enabled`) | `~/.config/opencode/AGENTS.md` / `./AGENTS.md` |
  | Gemini CLI | `~/.gemini/settings.json` | `./.gemini/settings.json` | JSON, `mcpServers.codegraph` | `~/.gemini/GEMINI.md` / **`./GEMINI.md`** (project root, not under `.gemini/`) |
  Plus Claude Code's `settings.json` permission handling: add `mcp__codegraph__*` to the `allow` list (opt-in, parity with TS `autoAllow`).
- **D-05a:** **AGNT-03 quirks to reproduce:** (1) **Cursor `--path`** — inject a `--path` arg into Cursor's MCP command: `local` install → absolute cwd; `global` install → the literal `${workspaceFolder}` variable Cursor expands. (2) **Codex** is global-only. (3) **opencode** JSONC must be edited **comment-preservingly** (Go equivalent of `jsonc-parser` — surgical text edit or a JSONC-aware editor; do NOT round-trip through a plain JSON marshal that drops comments). (4) **Gemini** local instruction file sits at **project root** (`./GEMINI.md`), matching Gemini's hierarchical context loader.
- **D-06:** **The 3 non-TS roster agents — Hermes, Antigravity, Kiro — have NO TS-parity reference** (TS covers only the 5 above). ⚠ **This is the phase's primary research gap.** For each, the researcher MUST determine the real MCP-config path/format and instruction-file convention from that agent's own docs. Runtime-dir hints from the harness: Hermes `~/.hermes`, Antigravity `~/.gemini/antigravity`, Kiro (`~/.kiro`?). **Recommended default if a convention can't be reliably confirmed:** model it on the closest known runtime (Hermes → Claude/JSON-shaped; Antigravity → Gemini/JSON-shaped) and **document it as `partial` in the install report rather than guessing silently** — reuse Phase-5's "documented-partial, do not overstate" discipline. Do not block the 5 known agents on the 3 unknowns.

### Idempotency & User-Edit Preservation (AGNT-01, AGNT-02)
- **D-07:** **All writes are surgical and format-preserving.** JSON: parse → set/delete only the `codegraph` entry → write back, preserving unrelated keys (and, for opencode JSONC, comments). TOML: edit only the `[mcp_servers.codegraph]` table. Marker blocks: on re-run **replace** the content between markers (idempotent → identical bytes on repeat); on `uninstall` **remove** only the marked span. **Re-running `install` twice is a no-op** at the byte level. `uninstall` on JSON removes `mcpServers.codegraph` and, if that empties `mcpServers`, removes the now-empty `mcpServers` object too (parity — keep configs clean).
- **D-08:** **`uninstall` reports per-agent status** — `removed` / `not-configured` / `unsupported` (parity). It never errors on an agent that was never installed; it preserves everything outside the CodeGraph MCP entry / marker block / `mcp__codegraph__*` permissions.

### CLI Ergonomics (CLI-01)
- **D-09:** **`codegraph version`** is a real subcommand (CLI-01 names it explicitly) **and** `--version` is wired on root. It prints **semver + git commit + build date + Go version + os/arch**, injected at build time via `-ldflags -X` (a `version` package with `var Version/Commit/Date` set to `dev`/`unknown` defaults for `go run`). Add `version --json` for scriptability (used by `upgrade --check` comparisons). No version wiring exists today — this is greenfield.
- **D-10:** **`codegraph help [command]`** uses **Cobra's built-in help** — the root already sets `Short`/`Long` and `SilenceUsage`. Ensure every command carries a useful `Short`/`Long`/`Example`; no custom help engine.

### `upgrade` — verify-then-swap (CLI-02) + Phase-8 sequencing
- **D-11:** **`codegraph upgrade [version]`** downloads the target-platform binary for the requested (default: latest) release from **the project's GitHub Releases**, **verifies its signature/provenance BEFORE swapping**, then atomically replaces the running binary. **`--check`** queries the latest release, compares to the current `version`, and reports whether an upgrade is available **without downloading**.
- **D-12:** **Verification is mandatory and embedded — never download-and-swap unverified.** The check reproduces the Phase-8 signing model (cosign **keyless**, verified against the release's OIDC identity + Rekor). Verification is done **in-process** (a sigstore Go verification library), NOT by shelling out to a `cosign` CLI — the single-static-binary guarantee forbids an external-tool dependency. ⚠ **Dependency-weight tradeoff:** sigstore verification pulls a non-trivial dependency subtree; flag this to the researcher/planner against the minimal-audited-deps constraint (prefer `sigstore-go`'s verification-only surface over full `cosign` if it keeps the tree smaller). The exact library choice is a research item.
- **D-13:** **Atomic self-replace:** download to a temp file **in the same directory** as the target (so `os.Rename` stays on one filesystem), `chmod +x`, then `os.Rename` over the current executable (POSIX). On Windows, use the rename-self-aside-then-rename-new dance (can't overwrite a running `.exe`). Resolve the live path via `os.Executable()`; refuse to upgrade a binary the current user can't write (clear error, no partial state).
- **D-14:** **Phase-8 alignment:** Phase 8 ships the actual signed releases (DIST-02). Phase 6 defines the release-URL + signing-identity as **named constants/config** the verify path consumes, testable now against **fixtures** (a signed test artifact) so `upgrade` is fully implemented and unit-tested this phase; wiring the real production identity is a Phase-8 finalize step. Do not stub the verification — build it real against fixtures.

### `telemetry` — proving the negative (CLI-03)
- **D-15:** **`codegraph telemetry`** prints a static statement that this build contains **zero telemetry / phone-home code**, and is **honest about the one intentional network path**: `codegraph upgrade` (explicit, user-initiated, to GitHub Releases) is the *only* outbound connection the binary ever makes, and it happens only when the user runs it. The trust claim is "no passive/background phone-home," not "never touches the network." Point the reader at the SBOM + source as the verifiable proof. Keep the wording auditable and non-marketing.

### Claude's Discretion
- Package name (`internal/agents` vs `internal/installer`) and the exact `AgentTarget` interface method set.
- JSONC-preservation mechanism for opencode (surgical text-span edit vs a JSONC-aware editor lib) — subject to the minimal-deps constraint.
- Interactive multi-select rendering (extend `confirm()` vs a small bubbletea-free prompt) — must degrade to `auto` with no TTY.
- Exact `version` string layout and whether `--json` is on `version` only or also `status`.
- The sigstore Go verification library (`sigstore-go` vs alternatives) — a researcher/Phase-8 alignment item, chosen for the smallest audited dep tree.
- Whether `install`/`uninstall` share one registry-iteration helper.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents (researcher, planner, executor) MUST read these before planning or implementing.**

### Parity Oracle — the TS installer source (AGNT-*)
- `colbymchenry/codegraph` (GitHub, read via DeepWiki `ask_question` or a fresh clone) — **the behavioral parity oracle.** Read the installer subsystem specifically:
  - the `AgentTarget` interface + per-agent target implementations (config path resolution, MCP-entry read/merge/write, marker-block inject/strip, detection heuristic, uninstall) — the shape D-02/D-05/D-07 reproduce.
  - `__tests__/installer-targets.test.ts` — the behavioral test matrix (per-agent install/uninstall assertions) to mirror as Go tests.
  - Confirm the exact marker text (`<!-- CODEGRAPH_START -->`/`<!-- CODEGRAPH_END -->`), the `--target auto|all|none|csv` + `--location global|local` flag semantics, the Cursor `--path` quirk, and the post-#529 short-form injected block (points at `codegraph explore`; MCP `initialize` is the tool-guidance source of truth).
- **Research gap (D-06):** Hermes, Antigravity, Kiro have **no** entry in the TS reference — their MCP-config conventions must be researched from each agent's own docs.

### Existing Substrate (bind against — do not re-derive)
- `internal/cli/root.go` — the Cobra tree D-02 extends; `SilenceUsage`/`SilenceErrors` convention; `Short`/`Long` pattern for D-10.
- `internal/cli/serve.go` — `serve --mcp --path` is what `install` wires each agent to launch (D-04); the existing `-p/--path` flag IS the AGNT-03 Cursor-quirk substrate.
- `internal/cli/uninit.go` §`confirm()` — the interactive-prompt idiom D-03 reuses/extends for the multi-select; also the model for a destructive-op confirmation.
- `internal/cli/{init,index}.go` — thin-command → delegation pattern, path resolution, `ErrNotInitialized`/`ErrAlreadyInitialized` idioms.
- `cmd/codegraph/main.go` — the `-ldflags -X` injection target for D-09 (`version` package); no version wiring exists yet.

### Project Planning & Constraints
- `.planning/ROADMAP.md` §"Phase 6: Agent Integrations & CLI Lifecycle" — the 5 success criteria (incl. success-criterion 5: `telemetry` reports zero telemetry).
- `.planning/REQUIREMENTS.md` — **AGNT-01/02/03, CLI-01/02/03** are the locked contract; note the scoping line "daemon in v1, **zero telemetry**".
- `.planning/PROJECT.md` §Constraints + §"Out of Scope" — single static binary / no bundled runtime / minimal audited deps (bounds D-12's dep choice); "Telemetry collection — zero phone-home code" (D-15).
- `.claude/CLAUDE.md` §"Supporting Libraries" / §"What NOT to Use" — cosign **keyless** via GitHub Actions OIDC, SLSA3 Go builder, Syft SBOM, `govulncheck` (the Phase-8 model `upgrade` verifies against, D-11/D-12/D-14).
- `.claude/CLAUDE.md` §"CodeGraph" (the global-instructions block) — the *shape* of the short marker-fenced instruction content `install` injects (points agents at `codegraph_explore` / `codegraph explore`).

**Note:** No golden-output corpus exists for this phase (D-01) — install writes external config, not graph output. Parity is asserted structurally against the TS source above.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **Cobra CLI (`internal/cli`)** — `root.go` wires the tree; `install`/`uninstall`/`upgrade`/`version`/`telemetry` are five new thin commands delegating to a new `internal/agents` package (installers) and a `version`/`upgrade` package (lifecycle).
- **`confirm()` (`internal/cli/uninit.go:69`)** — the interactive y/n helper; the multi-select agent picker (D-03) extends this idiom and must degrade to non-interactive `auto` with no TTY.
- **`serve --mcp --path` (`internal/cli/serve.go`)** — the command each agent's MCP config is wired to launch; `--path` already exists, so the Cursor `--path` quirk (AGNT-03) is a config-writing concern, not a new server flag.
- **`os.Executable()`** — resolves the absolute binary path both for `install`'s MCP command (D-04) and `upgrade`'s self-replace target (D-13).

### Established Patterns
- **Thin-CLI delegation** (Phase 2/3) — commands resolve flags/paths and delegate all logic to an internal package; installers must not live in `internal/cli`.
- **RED→GREEN atomic commits** — write the failing per-agent install/uninstall test (mirroring `installer-targets.test.ts`) before the implementation.
- **Interface-boundary discipline** — the agent registry is a self-contained package; it does NOT import `internal/graphstore`/`indexer`/`query` (install writes config files, it does not touch the graph).
- **Documented-partial honesty** (Phase 5 D-11) — the model for D-06's 3 unreferenced agents: mark `partial`, never overstate.

### Integration Points
- **`install` → agent config files** — the single write seam; each `AgentTarget` owns one agent's file(s). Round-trip `install`→`uninstall` is the acceptance invariant (D-07).
- **`upgrade` → GitHub Releases + sigstore verification → self-replace** — the only outbound-network path in the binary (D-15); gated on signature verification (D-12) before the atomic swap (D-13).
- **`go.mod`** — `upgrade` adds a sigstore verification dependency (D-12); this is the phase's main new-dependency decision and must be justified against the minimal-audited-deps constraint.

</code_context>

<specifics>
## Specific Ideas

- **The drop-in-swap is the whole point.** The marker text (`<!-- CODEGRAPH_START/END -->`) and per-agent config paths are a *cross-implementation contract*, not a fresh design: a user who installed with TS must be able to `uninstall` with the Go binary. Reproduce TS byte-shapes; don't improve on them.
- **`install`'s injected block is short and points at `codegraph explore`** — the full tool guidance lives in the MCP `initialize` response (Phase 3), per TS's post-#529 model. Do not re-inject the old full playbook.
- **The 3 non-TS agents (Hermes, Antigravity, Kiro) are the real unknown** — surface them to the researcher first; the 5 known agents are a faithful port, but these three need their own doc research and may ship documented-partial.
- **`telemetry` must be honest about `upgrade`'s network call** — the claim is "no passive phone-home," and pretending the binary never touches the network would be the wrong kind of parity.
- **`upgrade` verification is security-critical** — never swap an unverified binary. Build the real cosign-keyless verify path against fixtures now (D-14); Phase 8 only finalizes the production identity.

</specifics>

<deferred>
## Deferred Ideas

- **Actual signed/attested/reproducible releases + SLSA provenance + SBOM publication** — Phase 8 (DIST-01/02/03/04). Phase 6 builds the `upgrade` client that consumes them; Phase 8 produces them and finalizes the signing identity (D-14).
- **Migration tool** (`.codegraph/` SQLite → new format) — Phase 7 (MIGR-01/02).
- **Benchmarks / 100k-file monorepo / peak-RSS gates** — Phase 8 (PERF-01/02, INDX-06).
- **New MCP tools, HTTP/SSE transport, remote/multi-user, auth** — v2 (SERVER-01, BEYOND-01).
- **Agents beyond the 8-agent roster** — added on real user demand, not v1.

None of the above are scope creep into Phase 6 — they are correctly-placed future work, recorded so nothing is lost.

</deferred>

---

*Phase: 6-Agent Integrations & CLI Lifecycle*
*Context gathered: 2026-07-12*
