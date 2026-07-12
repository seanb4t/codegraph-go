# Phase 6: Agent Integrations & CLI Lifecycle - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-12
**Phase:** 6-Agent Integrations & CLI Lifecycle
**Mode:** `--auto` (fully autonomous — recommended default selected for every gray area, no interactive prompts)
**Areas discussed:** Agent registry & config matrix, Marker/idempotency model, Non-TS agents, `upgrade` verify-then-swap, CLI ergonomics, `telemetry`

---

## Agent detection & target selection

| Option | Description | Selected |
|--------|-------------|----------|
| Parity: `--target auto\|all\|none\|csv`, auto-detect default, interactive multi-select on TTY | Mirror TS installer exactly; degrade to `auto` with no TTY / under `--auto` | ✓ |
| Configure-all always | Simpler, but writes config for agents the user doesn't use | |
| Interactive-only (always prompt) | Blocks non-interactive/CI runs | |

**Selected (recommended default):** Parity `--target`/`--location` flags; `auto` detection; multi-select only on a TTY (D-03).
**Notes:** Non-interactive/`--auto`/CI never prompts — defaults to `auto`. Detection = agent config dir/file existence.

---

## Per-agent config matrix & quirks (AGNT-01/03)

| Option | Description | Selected |
|--------|-------------|----------|
| Reproduce TS paths/formats + quirks via an `AgentTarget` registry | Exact parity for the 5 TS-covered agents; registry hosts per-agent quirks | ✓ |
| One generic JSON writer for all agents | Breaks Codex (TOML), opencode (JSONC comments), Cursor (`--path`), Gemini root-file | |

**Selected (recommended default):** `AgentTarget` interface, one impl per agent, exact TS path/format matrix + Cursor `--path`, Codex global-only, opencode comment-preserving JSONC, Gemini root-level `GEMINI.md` (D-02/D-05/D-05a).
**Notes:** MCP config invokes this binary by absolute path via `os.Executable()` (D-04) — the drop-in-swap requirement.

---

## Non-TS roster agents: Hermes, Antigravity, Kiro (AGNT-01)

| Option | Description | Selected |
|--------|-------------|----------|
| Research each from its own docs; documented-partial if unconfirmable | Honest; models on closest known runtime; doesn't block the 5 known agents | ✓ |
| Guess a JSON `mcpServers` shape for all three | Risks writing configs the agents can't read | |
| Defer all three to a later phase | Violates AGNT-01's explicit 8-agent roster | |

**Selected (recommended default):** Flag as the phase's primary research gap; per-agent doc research; documented-partial discipline (D-06).
**Notes:** TS reference covers only 5 of 8. Runtime hints: `~/.hermes`, `~/.gemini/antigravity`, `~/.kiro`(?).

---

## Marker fences, idempotency, user-edit preservation (AGNT-01/02)

| Option | Description | Selected |
|--------|-------------|----------|
| Reproduce TS markers exactly; surgical format-preserving edits; marker-scoped strip | Cross-impl contract (TS-install → Go-uninstall); byte-idempotent re-run | ✓ |
| New marker text / full-file rewrite | Breaks cross-impl round-trip; destroys user edits | |

**Selected (recommended default):** `<!-- CODEGRAPH_START/END -->` verbatim; surgical JSON/JSONC/TOML edits; empty-`mcpServers` cleanup; per-agent uninstall status (D-01a/D-07/D-08).

---

## `upgrade` mechanism + Phase-8 sequencing (CLI-02)

| Option | Description | Selected |
|--------|-------------|----------|
| Verify-then-swap: GitHub Releases + embedded cosign-keyless verify + atomic self-replace; `--check` | Security-critical verify before swap; no external `cosign` CLI; fixtures now, Phase-8 finalizes identity | ✓ |
| Download-and-swap without signature verification | Unacceptable supply-chain risk | |
| Shell out to `cosign` CLI | Breaks single-static-binary guarantee | |

**Selected (recommended default):** Embedded sigstore verification, atomic `os.Rename` self-replace, `--check` compares versions without downloading (D-11/D-12/D-13/D-14).
**Notes:** sigstore Go dep weight flagged against minimal-deps constraint; library choice is a research item. Build real against fixtures, not stubbed.

---

## CLI ergonomics: `version` / `help` (CLI-01)

| Option | Description | Selected |
|--------|-------------|----------|
| `version` subcommand + `--version`, `-ldflags -X` semver/commit/date/go/arch, `version --json`; Cobra built-in help | Standard, scriptable, greenfield | ✓ |
| `--version` flag only | CLI-01 names `codegraph version` explicitly | |

**Selected (recommended default):** D-09/D-10.

---

## `telemetry` command (CLI-03)

| Option | Description | Selected |
|--------|-------------|----------|
| Honest zero-telemetry report that names `upgrade` as the sole user-initiated network path | Truthful "no passive phone-home"; points to SBOM/source as proof | ✓ |
| "Never touches the network" claim | False — `upgrade` makes an intentional outbound call | |

**Selected (recommended default):** D-15.

---

## Claude's Discretion

- Package name (`internal/agents` vs `internal/installer`) and exact `AgentTarget` method set.
- opencode JSONC comment-preservation mechanism (surgical text edit vs JSONC lib), bounded by minimal-deps.
- Multi-select prompt rendering; must degrade to `auto` with no TTY.
- `version` string layout; sigstore verification library selection (Phase-8-aligned).

## Deferred Ideas

- Signed/attested/reproducible releases + SLSA + SBOM publication → Phase 8 (finalizes `upgrade`'s signing identity).
- Migration tool → Phase 7.
- Benchmarks / 100k-file scale / peak-RSS → Phase 8.
- New MCP tools / HTTP-SSE transport / remote-multi-user → v2.
- Agents beyond the 8-agent roster → future, on demand.
