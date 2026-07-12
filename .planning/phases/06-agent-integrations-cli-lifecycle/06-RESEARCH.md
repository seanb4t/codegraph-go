# Phase 6: Agent Integrations & CLI Lifecycle - Research

**Researched:** 2026-07-12
**Domain:** CLI installer/uninstaller subsystem for 8 coding-agent MCP integrations, CLI lifecycle ergonomics (version/help), and a cryptographically-verified self-update mechanism
**Confidence:** HIGH (agent config parity — read directly from the TS parity-oracle source, all 8 targets); MEDIUM (upgrade/sigstore mechanics — verified via Context7 + module proxy, no live fixture built yet); LOW (Hermes/Kiro doc conventions beyond what TS source + one doc cross-check confirms)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** Parity is behavioral, measured against the TS installer *source*, not a golden-output corpus. `install`/`uninstall` write to external agent config files (JSON/JSONC/TOML/Markdown/YAML), so there is no `codegraph`-emitted output to diff. The acceptance oracle is: for each agent, the config file ends in the same *shape* TS produces (same keys, same marker block, same quirk handling), and a round-trip `install`→`uninstall` returns the file to its pre-install bytes modulo the CodeGraph section.
- **D-01a:** Marker fences reproduce TS exactly — `<!-- CODEGRAPH_START -->` … `<!-- CODEGRAPH_END -->`. Hard parity contract — a Go `uninstall` must recognize a marker block a TS `install` wrote, and vice-versa.
- **D-02:** `codegraph install` / `codegraph uninstall` are two new top-level Cobra commands in `internal/cli/root.go`. All agent logic lives in a new `internal/agents` (or `internal/installer`) package exposing an `AgentTarget` interface — `Detect() bool`, `Install(cfg) → changed`, `Uninstall() → status`, plus metadata. One implementation per roster agent; the two commands iterate the registry.
- **D-03:** Flag surface mirrors TS: `--target auto|all|none|<csv-of-ids>` (default `auto`) and `--location global|local` (per-agent default). `auto` = configure only detected agents. TTY + no `--target` → interactive multi-select prefilled with detected agents (reuse/extend `confirm()` in `internal/cli/uninit.go`). No TTY or `--auto`/CI → default to `auto` without prompting.
- **D-04:** The MCP config `install` writes invokes this binary by **absolute path** — resolve `os.Executable()` at install time. Makes the drop-in swap real: the agent launches the Go binary the user just ran `install` from.
- **D-05:** The 5 TS-covered agents get their exact TS paths/formats (see the corrected/expanded parity table below — this research found TS source covers all 8, superseding the "5 covered" assumption).
- **D-05a:** AGNT-03 quirks to reproduce: Cursor `--path` (local=abs cwd, global=`${workspaceFolder}`); Codex is global-only; opencode JSONC must be edited comment-preservingly; Gemini local instruction file sits at project root (`./GEMINI.md`).
- **D-06:** The 3 non-TS roster agents — Hermes, Antigravity, Kiro — were assumed to have NO TS-parity reference. **This research found TS source DOES implement all three** (see below) — this is the single biggest correction to CONTEXT.md and closes the phase's stated "primary research gap" with source-verified answers instead of documented-partial guesses.
- **D-07:** All writes are surgical and format-preserving. JSON: parse → set/delete only the `codegraph` entry → write back, preserving unrelated keys (and, for opencode JSONC, comments). TOML: edit only the `[mcp_servers.codegraph]` table. Marker blocks: replace-in-place on re-run (idempotent), remove-only-the-marked-span on uninstall. Re-running `install` twice is a no-op at the byte level.
- **D-08:** `uninstall` reports per-agent status — `removed` / `not-configured` / `unsupported`. Never errors on an agent never installed; preserves everything outside the CodeGraph surfaces.
- **D-09:** `codegraph version` is a real subcommand; `--version` also wired on root. Prints semver + git commit + build date + Go version + os/arch, injected via `-ldflags -X`. Add `version --json`.
- **D-10:** `codegraph help [command]` uses Cobra's built-in help. No custom help engine.
- **D-11:** `codegraph upgrade [version]` downloads the target-platform binary from GitHub Releases, verifies signature/provenance BEFORE swapping, then atomically replaces the running binary. `--check` compares versions without downloading.
- **D-12:** Verification is mandatory and embedded — never download-and-swap unverified. Reproduces the Phase-8 cosign-keyless model, verified in-process (a sigstore Go verification library), NOT by shelling out to a `cosign` CLI. Dependency-weight tradeoff flagged for research (see Standard Stack / Package Legitimacy Audit below).
- **D-13:** Atomic self-replace: download to a temp file in the same directory as the target, `chmod +x`, then `os.Rename` over the current executable (POSIX). Windows: rename-self-aside-then-rename-new dance. Resolve the live path via `os.Executable()`; refuse to upgrade a binary the current user can't write.
- **D-14:** Phase 8 ships the actual signed releases (DIST-02). Phase 6 defines release-URL + signing-identity as named constants/config the verify path consumes, testable now against fixtures, so `upgrade` is fully implemented and unit-tested this phase; wiring the real production identity is a Phase-8 finalize step.
- **D-15:** `codegraph telemetry` prints a static statement that this build contains zero telemetry/phone-home code, and is honest about the one intentional network path: `codegraph upgrade` (explicit, user-initiated) is the only outbound connection the binary ever makes.

### Claude's Discretion

- Package name (`internal/agents` vs `internal/installer`) and the exact `AgentTarget` interface method set.
- JSONC-preservation mechanism for opencode (surgical text-span edit vs a JSONC-aware editor lib) — subject to the minimal-deps constraint.
- Interactive multi-select rendering (extend `confirm()` vs a small bubbletea-free prompt) — must degrade to `auto` with no TTY.
- Exact `version` string layout and whether `--json` is on `version` only or also `status`.
- The sigstore Go verification library (`sigstore-go` vs alternatives) — a researcher/Phase-8 alignment item, chosen for the smallest audited dep tree.
- Whether `install`/`uninstall` share one registry-iteration helper.

### Deferred Ideas (OUT OF SCOPE)

- Actual signed/attested/reproducible releases + SLSA provenance + SBOM publication — Phase 8 (DIST-01/02/03/04). Phase 6 builds the `upgrade` client that consumes them; Phase 8 produces them and finalizes the signing identity (D-14).
- Migration tool (`.codegraph/` SQLite → new format) — Phase 7 (MIGR-01/02).
- Benchmarks / 100k-file monorepo / peak-RSS gates — Phase 8 (PERF-01/02, INDX-06).
- New MCP tools, HTTP/SSE transport, remote/multi-user, auth — v2 (SERVER-01, BEYOND-01).
- Agents beyond the 8-agent roster — added on real user demand, not v1.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| AGNT-01 | `codegraph install` detects+configures the 8-agent roster, MCP config + marker-fenced instruction injection, idempotent on re-run | Full per-agent parity table below (all 8, sourced directly from TS `src/installer/targets/*.ts`); `AgentTarget` interface design; idempotency mechanics (`jsonDeepEqual`-style unchanged detection, marker upsert) |
| AGNT-02 | `codegraph uninstall` cleanly reverses everything `install` wrote, preserving user edits outside markers | `UninstallStatus` enum ported 1:1 from TS; per-target uninstall logic in the parity table; round-trip byte-invariant test strategy |
| AGNT-03 | Per-agent quirks handled (Cursor `--path`, etc.) | Full quirk catalogue below — Cursor `--path`, Codex global-only + hand-rolled TOML, opencode JSONC + XDG resolution, Gemini project-root `GEMINI.md`, Antigravity no-`type`-field + macOS PATH resolution + unified/legacy config migration, Hermes YAML line-range surgery + `platform_toolsets.cli`, Kiro steering-file self-heal-delete |
| CLI-01 | `codegraph help [command]` + `codegraph version` standard ergonomics | Cobra built-in help pattern (already used); `-ldflags -X` version package pattern with TS's simpler `packageJson.version` precedent for comparison |
| CLI-02 | `codegraph upgrade [version]` self-updates via signature-verified download-and-swap, `--check` | sigstore-go verification API (Context7-verified); GitHub Releases latest-version resolution (redirect-trick pattern, avoids API rate limits); atomic swap mechanics (temp-in-same-dir + os.Rename; Windows rename-aside dance); dependency-weight tradeoff documented in Package Legitimacy Audit |
| CLI-03 | `codegraph telemetry` reports zero telemetry; binary contains zero phone-home code | Static, no-op design — no TS parity needed (TS's telemetry IS real and collects data; this is an intentional Go-project divergence per PROJECT.md's stronger trust story) |
</phase_requirements>

## Summary

TS CodeGraph's installer subsystem (`colbymchenry/codegraph`, `src/installer/`) is a clean, well-documented `AgentTarget` interface with one implementation per agent, a central registry, and shared surgical-write helpers (`shared.ts`). Critically, **the TS source implements all 8 roster agents** — Claude Code, Cursor, Codex CLI, opencode, Gemini CLI, **and** Hermes Agent, Antigravity IDE, Kiro — not just the 5 CONTEXT.md assumed. This resolves the phase's stated "primary research gap" (D-06) with source-verified specifics rather than a documented-partial fallback: every agent's exact config path, format, and quirks are now known with HIGH confidence, cross-checked against each agent's own current docs where feasible (Hermes, Kiro).

A second major finding: TS's `upgrade` has **zero cryptographic verification** — it's npm-shell-out or curl-pipe-sh, trusting HTTPS transport only. This confirms D-11/D-12's framing is correct: `upgrade`'s verify-then-swap is genuinely new work, not a port. `sigstore-go` (Context7-verified, `github.com/sigstore/sigstore-go`, current release v1.2.2) is the right verification-only library — it pulls a real dependency subtree (rekor, go-tuf, certificate-transparency-go, protobuf-specs, ~20 transitive modules) but this is the *smaller* of the two realistic options; shelling out to a `cosign` binary is disallowed by the single-static-binary constraint, and there is no lighter in-process alternative for Fulcio/Rekor/TUF-chain verification. The atomic-swap mechanics D-13 already specifies precisely enough (~40 lines, stdlib-only) that hand-rolling is recommended over adding `minio/selfupdate` (last tagged release Oct 2022, though the fork is still consumed in production by MinIO) — reference its `apply.go` for the Windows rename-aside pattern but do not add it as a dependency.

For opencode's JSONC preservation, `github.com/tailscale/hujson` is a near-zero-cost dependency (single indirect dep: `google/go-cmp`, already in `go.mod`) that provides exactly the `Parse` → `Patch` (RFC 6902) → `Format` → `Pack` surgical-edit workflow TS's `jsonc-parser` provides. For Codex's TOML, mirror TS's own choice: **hand-roll a narrow ~100-line table-block injector** (find `[mcp_servers.codegraph]`, splice in/out, preserve everything else verbatim) rather than adding a general TOML dependency — TS's own `toml.ts` file header explicitly documents this same reasoning ("~50KB dependency for ~6 lines of output").

**Primary recommendation:** Build `internal/agents` with one file per target mirroring the TS file-per-target layout (renamed to Go idiom), a `registry.go` matching TS's `registry.ts` almost mechanically, and `shared.go` porting `replaceOrAppendMarkedSection`/`removeMarkedSection`/`upsertInstructionsEntry`/`readJsonFile`/`writeJsonFile`/`jsonDeepEqual` as pure functions. This is a faithful, mechanical Go port of proven, tested logic (the TS test suite covers 1711 lines / ~90 cases across every quirk) — treat `__tests__/installer-targets.test.ts` as the RED-test source of truth to mirror, not just prose to summarize.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Agent config file detection/read/write (install/uninstall) | CLI / local filesystem | — | Pure local file I/O against each agent's own config dir; no network, no graph, no MCP server involvement |
| MCP server entry the config points at | CLI (constant string) | MCP server (already built, Phase 3) | `install` only writes the *pointer* (`codegraph serve --mcp [--path X]`); the actual server behavior is frozen from Phase 3 |
| Version metadata | Build tooling (ldflags) → CLI | — | Injected at compile time, read at runtime by a thin `version` package; no runtime computation |
| Self-update download + verify + swap | CLI / local filesystem + GitHub Releases (network) | Sigstore public-good infra (Rekor, Fulcio, TUF — network, read-only) | The only phase capability that legitimately touches the network; isolated to one command, one code path |
| Telemetry disclosure | CLI (static string) | — | No collection, no state, no network call of its own — literally a `fmt.Println` of a constant |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|---------------|
| `github.com/tailscale/hujson` | `v0.0.0-20260302212456-ecc657c15afd` [ASSUMED — package name from WebSearch/training, existence + version VERIFIED via Go module proxy] | Comment-preserving JSONC parse/patch/format for opencode's `opencode.jsonc` | Only pure-Go library offering an RFC-6902 JSON-Patch-based surgical edit over a comment-preserving syntax tree — the direct analog of TS's `jsonc-parser` `modify`/`applyEdits`. Near-zero dependency cost (`go.mod` requires only `google/go-cmp`, already a direct dep of this project). Maintained by Tailscale (used in their own `tailscale.conf`-style configs); no tagged release yet (pseudo-version only) — flag as a minor maintenance-signal risk, not a blocker. |
| `github.com/sigstore/sigstore-go` | `v1.2.2` [ASSUMED — package name from Context7/WebSearch, existence + version VERIFIED via Go module proxy] | In-process keyless (Fulcio+Rekor+TUF) bundle verification for `codegraph upgrade` | The one purpose-built verification-only Sigstore client for Go; explicitly designed (per its own docs, cross-checked via WebSearch) to have a smaller footprint than embedding the full `cosign` CLI's dependency tree, and is the library the ecosystem is consolidating on for exactly this use case (in-process verification without shelling to a CLI). See Package Legitimacy Audit for the dependency-weight detail this pulls in — it is real, but it is the smallest available for this exact job. |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/spf13/cobra` | (already in go.mod, v1.10.2) | `install`/`uninstall`/`upgrade`/`version`/`telemetry` subcommands | Already the project's CLI framework — no new dependency; follow the existing thin-command pattern in `internal/cli/*.go` |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `sigstore-go` (in-process verification) | Shell out to a bundled/PATH `cosign` binary | Rejected outright by the project's single-static-binary constraint (`.claude/CLAUDE.md` "What NOT to Use" pattern: no external-tool runtime dependency). Not a real option. |
| `sigstore-go` | Hand-roll Fulcio cert-chain + Rekor inclusion-proof + TUF trust-root verification from scratch | Rejected — this is exactly the "huge effort, high risk of a subtly-wrong crypto implementation" case the project's own CGo-parser precedent (`.claude/CLAUDE.md`, Option C) warns against for load-bearing infrastructure; sigstore-go is actively maintained by the org that defines the protocol. |
| Hand-rolled atomic-swap (`internal/upgrade/swap.go`, ~40 lines) | `github.com/minio/selfupdate` | `minio/selfupdate` (fork of the now-archived `inconshreveable/go-update`) does handle both the POSIX rename-in-place and the Windows rename-aside dance, plus optional bsdiff patching and its own SHA256 checksum gate — but its own crypto-verification hook would be *redundant* with sigstore-go's stronger verification, its last tagged release is Oct 2022 (~4 years stale as of this research), and D-13 already fully specifies the ~40 lines of mechanics needed. Recommend hand-rolling per the project's minimal-deps posture; reference `minio/selfupdate/apply.go` as a pattern source only (not a dependency) for the Windows rename-aside edge case. |
| Hand-rolled TOML table injector (`internal/agents/toml.go`, ~100 lines, mirroring TS's `toml.ts`) | A general pure-Go TOML library (e.g. `BurntSushi/toml`, `pelletier/go-toml`) | TS's own `toml.ts` file header states the reasoning directly: "~50KB dependency for ~6 lines of output" — a full TOML parser/serializer is unjustified for editing exactly one dotted-key table block while preserving everything else byte-for-byte. Mirror TS's own text-splice strategy (find `[mcp_servers.codegraph]` header line, splice to next top-level `[...]` header or EOF) — this is proven, tested logic (see `toml.ts` + its test coverage in `installer-targets.test.ts`), not a fresh design. |

**Installation:**
```bash
go get github.com/tailscale/hujson@v0.0.0-20260302212456-ecc657c15afd
go get github.com/sigstore/sigstore-go@v1.2.2
```
Manually promote both to the direct `require` block in `go.mod` per the project's established convention (never run a blanket `go mod tidy` — see STATE.md's repeated "Manually promoted X ... instead of running go mod tidy" decisions across every phase so far).

**Version verification:** Both versions above were confirmed live against the Go module proxy (`https://proxy.golang.org/<module>/@latest`) during this research session — `sigstore-go` v1.2.2 (tagged 2026-07-06, one week before this research), `hujson` pseudo-version dated 2026-03-02 (no tagged release exists for hujson; this is normal for the project — it has never cut a tag — but confirm this hasn't changed before implementation).

## Package Legitimacy Audit

> The `gsd-tools query package-legitimacy check` seam only supports `--ecosystem npm|pypi|crates` — Go modules are not covered by the seam. The table below was produced by manual verification against the Go module proxy (`proxy.golang.org`) and GitHub org reputation, since this is a Go-only phase.

| Package | Registry | Age / Last Release | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|---------------------|-----------|--------------|---------|-------------|
| `github.com/sigstore/sigstore-go` | Go module proxy | v1.2.2 tagged 2026-07-06 (actively released) | N/A (Go modules have no download counter) | `github.com/sigstore/sigstore-go` — official Sigstore org, High reputation (Context7 "High" source reputation, 93.12 benchmark score) | OK | Approved — pulls a real transitive dependency subtree (~20 modules: `rekor`, `go-tuf/v2`, `certificate-transparency-go`, `protobuf-specs`, `in-toto-golang`, etc.) but every one is itself an official Sigstore-ecosystem or well-known crypto/protobuf module, not a slopsquat risk. This dependency weight is the real, honest tradeoff D-12 asked to be flagged. |
| `github.com/tailscale/hujson` | Go module proxy | Pseudo-version 2026-03-02, no tagged release | N/A | `github.com/tailscale/hujson` — Tailscale org (High reputation, production company, used in their own tooling) | OK (with a caveat) | Approved — near-zero dependency cost, reputable maintainer, but flag the never-tagged status: pin the exact pseudo-version commit hash and re-verify at each `go.mod` bump rather than trusting a floating `@latest`. |
| `github.com/minio/selfupdate` | Go module proxy | v0.6.0, tagged 2022-10-19 (~4 years stale) | N/A | `github.com/minio/selfupdate` — MinIO org (High reputation) | SUS (staleness, not illegitimacy) | **NOT recommended for adoption** — hand-roll instead (see Alternatives Considered). If a future researcher revisits this and decides to adopt it anyway, add a `checkpoint:human-verify` before installing given the multi-year gap since last tag. |

**Packages removed due to [SLOP] verdict:** none.
**Packages flagged as suspicious [SUS]:** `github.com/minio/selfupdate` — staleness flag only; not recommended for adoption in this phase regardless, so no install-time checkpoint is needed unless the planner overrides the hand-roll recommendation.

*`sigstore-go` and `hujson` were discovered via Context7/WebSearch (not an authoritative doc first-party to this project) and are tagged `[ASSUMED]` for package-name provenance per the tagging rule, even though their existence and version were separately VERIFIED against the Go module proxy in this session.*

## Architecture Patterns

### System Architecture Diagram

```
                    codegraph install / codegraph uninstall
                                    │
                                    ▼
                    ┌───────────────────────────────┐
                    │  internal/cli (Cobra command)  │
                    │  - parse --target/--location   │
                    │  - resolve os.Executable() path│
                    │  - TTY? → multiselect prompt   │
                    │  - else → resolveTargetFlag()  │
                    └───────────────┬────────────────┘
                                    │  []AgentTarget
                                    ▼
                    ┌───────────────────────────────┐
                    │  internal/agents (registry)    │
                    │  ALL_TARGETS = [claude, cursor,│
                    │   codex, opencode, hermes,     │
                    │   gemini, antigravity, kiro]   │
                    └───────────────┬────────────────┘
                                    │  for each target
                                    ▼
              ┌─────────────────────────────────────────┐
              │  target.Install(loc, opts) / Uninstall() │
              │  (one Go file per agent, mirrors TS)     │
              └──────┬───────────────┬───────────────┬───┘
                     │               │               │
         ┌───────────▼──┐  ┌─────────▼────────┐  ┌───▼──────────────┐
         │ JSON configs │  │ TOML (Codex)     │  │ YAML (Hermes)    │
         │ (shared.go:  │  │ (agents/toml.go: │  │ (line-range      │
         │  readJSON/   │  │  hand-rolled     │  │  surgery, no dep)│
         │  writeJSON/  │  │  table splice)   │  └───────────────────┘
         │  jsonEqual)  │  └──────────────────┘
         └──────┬───────┘
                │
         ┌──────▼────────────────────┐
         │ JSONC (opencode.jsonc):   │
         │ hujson.Parse → Patch →    │
         │ Format → Pack             │
         └───────────────────────────┘
                     │
                     ▼
         ┌───────────────────────────┐
         │ Marker-fenced instruction  │
         │ files (CLAUDE.md/AGENTS.md│
         │ /GEMINI.md — 4 of 8 agents)│
         │ shared.go: upsert/remove   │
         │ MarkedSection              │
         └────────────────────────────┘

                    codegraph upgrade [version] [--check]
                                    │
                                    ▼
                    ┌───────────────────────────────┐
                    │  internal/version (ldflags)    │
                    │  currentVersion string const   │
                    └───────────────┬────────────────┘
                                    │
                                    ▼
                    ┌───────────────────────────────┐
                    │  internal/upgrade              │
                    │  1. resolveLatest() — GitHub   │
                    │     Releases redirect trick    │
                    │  2. --check? report + exit     │
                    │  3. download binary + bundle   │
                    │  4. sigstore-go: verify bundle │
                    │     against release identity   │
                    │  5. verified? → atomic swap    │
                    │     (temp-in-same-dir + Rename;│
                    │      Windows rename-aside)     │
                    │  6. unverified? → abort, no    │
                    │     partial state, clear error │
                    └───────────────┬────────────────┘
                                    │
                                    ▼
                    GitHub Releases (network) + Sigstore
                    public-good infra (Rekor/Fulcio/TUF,
                    network, read-only, verification only)
```

### Recommended Project Structure

```
internal/
├── cli/
│   ├── install.go       # new: `codegraph install` — thin, delegates to internal/agents
│   ├── uninstall.go     # new: `codegraph uninstall` — thin, delegates to internal/agents
│   ├── upgrade.go       # new: `codegraph upgrade [version] [--check]` — thin, delegates to internal/upgrade
│   ├── version.go       # new: `codegraph version [--json]` — thin, reads internal/version
│   ├── telemetry.go     # new: `codegraph telemetry` — static print, no delegation needed
│   └── root.go          # extended: register the 5 new commands (D-02)
├── agents/               # NEW package (D-02's "internal/agents") — no import of graphstore/indexer/query
│   ├── types.go          # AgentTarget interface, Location, TargetId, DetectionResult, WriteResult, InstallOptions
│   ├── shared.go         # readJSONFile/writeJSONFile/jsonDeepEqual/replaceOrAppendMarkedSection/
│   │                      #   removeMarkedSection/upsertInstructionsEntry/atomicWriteFile — ported from shared.ts
│   ├── toml.go            # hand-rolled TOML table splice (mirrors toml.ts) — Codex only
│   ├── instructions.go    # the short CODEGRAPH_START/END block text — ported from instructions-template.ts
│   ├── registry.go        # ALL_TARGETS, GetTarget, DetectAll, ResolveTargetFlag — ported from registry.ts
│   ├── claude.go          # ~/.claude.json + ./.mcp.json + CLAUDE.md + settings.json permissions
│   ├── cursor.go          # ~/.cursor/mcp.json + ./.cursor/mcp.json + --path quirk; NO instructions file
│   ├── codex.go           # ~/.codex/config.toml (TOML) + ~/.codex/AGENTS.md; global-only
│   ├── opencode.go        # opencode.jsonc/.json (hujson) + AGENTS.md; XDG resolution
│   ├── gemini.go          # ~/.gemini/settings.json + GEMINI.md (project-root for local)
│   ├── antigravity.go     # ~/.gemini/config|antigravity/mcp_config.json; no `type` field; macOS PATH resolve; global-only
│   ├── hermes.go          # $HERMES_HOME/config.yaml (YAML line-range surgery, no lib); global-only
│   └── kiro.go            # ~/.kiro/settings/mcp.json; steering doc is create-then-self-heal-delete only
├── version/                # NEW package (D-09)
│   └── version.go          # var Version, Commit, Date, GoVersion, OS, Arch — set via -ldflags -X in production, "dev"/"unknown" defaults for `go run`
└── upgrade/                 # NEW package (D-11..D-14)
    ├── release.go           # resolveLatestVersion() — GitHub Releases redirect-trick + API fallback
    ├── verify.go            # sigstore-go wiring: NewVerifier, NewPolicy, WithArtifactDigest, WithCertificateIdentity
    ├── swap.go              # atomic self-replace: temp-in-same-dir + os.Rename (POSIX); rename-aside dance (Windows)
    └── upgrade.go            # orchestrator: resolve → check?  → download → verify → swap
```

### Pattern 1: AgentTarget interface (port of TS's `types.ts`)

**What:** One Go interface every agent implements; the registry iterates it, so `install`/`uninstall` never branch on agent identity outside the target's own file.
**When to use:** Every one of the 8 agent files.
**Example:**
```go
// Source: mechanical Go port of src/installer/targets/types.ts (colbymchenry/codegraph)
package agents

type Location string

const (
	LocationGlobal Location = "global"
	LocationLocal  Location = "local"
)

type TargetID string

const (
	Claude      TargetID = "claude"
	Cursor      TargetID = "cursor"
	Codex       TargetID = "codex"
	Opencode    TargetID = "opencode"
	Hermes      TargetID = "hermes"
	Gemini      TargetID = "gemini"
	Antigravity TargetID = "antigravity"
	Kiro        TargetID = "kiro"
)

type DetectionResult struct {
	Installed         bool
	AlreadyConfigured bool
	ConfigPath        string
}

type FileAction string

const (
	ActionCreated   FileAction = "created"
	ActionUpdated   FileAction = "updated"
	ActionUnchanged FileAction = "unchanged"
	ActionRemoved   FileAction = "removed"
	ActionNotFound  FileAction = "not-found"
	ActionKept      FileAction = "kept"
)

type FileResult struct {
	Path   string
	Action FileAction
}

type WriteResult struct {
	Files []FileResult
	Notes []string
}

type InstallOptions struct {
	AutoAllow bool // Claude-only permissions write; no-op for targets without the concept
}

type AgentTarget interface {
	ID() TargetID
	DisplayName() string
	SupportsLocation(loc Location) bool
	Detect(loc Location) DetectionResult
	Install(loc Location, opts InstallOptions) WriteResult
	Uninstall(loc Location) WriteResult
	PrintConfig(loc Location) string
	DescribePaths(loc Location) []string
}
```

### Pattern 2: idempotent JSON entry write (port of `writeMcpEntry` pattern used by 6 of 8 targets)

**What:** Read existing JSON (empty map if missing/unparseable), compare the existing `codegraph` entry to the desired one with an order-independent deep-equal, write only if different, report `created`/`updated`/`unchanged` correctly.
**When to use:** Claude, Cursor, Gemini, Kiro, Antigravity (all JSON `mcpServers.codegraph`-shaped agents).
**Example:**
```go
// Source: mechanical Go port of src/installer/targets/claude.ts writeMcpEntry() +
// src/installer/targets/shared.ts jsonDeepEqual/readJsonFile/writeJsonFile
func writeMcpEntry(file string, buildEntry func() any) (FileResult, error) {
	existing, err := readJSONFile(file) // returns map[string]any{}, nil on missing/unparseable
	if err != nil {
		return FileResult{}, err
	}
	mcpServers, _ := existing["mcpServers"].(map[string]any)
	if mcpServers == nil {
		mcpServers = map[string]any{}
	}
	before := mcpServers["codegraph"]
	after := buildEntry()

	if jsonDeepEqual(before, after) {
		return FileResult{Path: file, Action: ActionUnchanged}, nil
	}
	existedBefore := before != nil || fileExists(file)
	action := ActionCreated
	if existedBefore {
		action = ActionUpdated
	}
	mcpServers["codegraph"] = after
	existing["mcpServers"] = mcpServers
	if err := writeJSONFile(file, existing); err != nil {
		return FileResult{}, err
	}
	return FileResult{Path: file, Action: action}, nil
}
```

### Pattern 3: marker-fenced instruction block upsert (port of `shared.ts` `replaceOrAppendMarkedSection`)

**What:** Find `<!-- CODEGRAPH_START -->`...`<!-- CODEGRAPH_END -->` in a file; if present and identical, no-op; if present and different, replace in place; if absent, append (with a separating blank line if the file has content); if the file doesn't exist, create it.
**When to use:** Claude (`CLAUDE.md`), Codex (`AGENTS.md`), opencode (`AGENTS.md`), Gemini (`GEMINI.md`) — the 4 of 8 agents that get a marker-fenced instruction file. **Cursor, Antigravity, Hermes, Kiro do NOT get this treatment** (see Corrected Per-Agent Parity Table below) — do not write it for them.
**Example:**
```go
// Source: mechanical Go port of src/installer/targets/shared.ts replaceOrAppendMarkedSection
const (
	codegraphSectionStart = "<!-- CODEGRAPH_START -->"
	codegraphSectionEnd   = "<!-- CODEGRAPH_END -->"
)

// codegraphInstructionsBlock is the short pointer block (post-#529/#704 in TS
// terms): "reach for codegraph_explore / `codegraph explore` before grep,
// skip entirely if no .codegraph/ exists." This is a SHORT block — do not
// re-inject the old full playbook; the MCP `initialize` response (Phase 3)
// is the single source of truth for full tool guidance.
```

### Anti-Patterns to Avoid

- **Writing an instructions file for every agent uniformly:** Only 4 of 8 agents (Claude, Codex, opencode, Gemini) get a marker-fenced instructions file in the current TS source. Cursor and Kiro *used to* (pre-#529) and now actively self-heal by DELETING any legacy file a previous install left. Antigravity and Hermes never had one. Writing one for all 8 would be a parity regression, not an improvement.
- **Using a full TOML or general-purpose JSON-diff library:** Both TS's own `toml.ts` and this research's `hujson` recommendation exist specifically to avoid over-general dependencies for a narrow single-key-block edit. Don't reach for `pelletier/go-toml` when a ~100-line splice does the whole job.
- **Reading `~/.claude.json` for local-scope Claude Code config:** A common historical bug (TS issue #207) — Claude Code never reads a project-local `./.claude.json`; the correct local file is `./.mcp.json`. If a legacy `./.claude.json` codegraph entry exists, migrate/strip it (see Common Pitfalls).
- **Bare `codegraph` command in MCP config entries:** TS uses a bare `command: "codegraph"` (PATH-relative) for 7 of 8 agents and only resolves an absolute path for Antigravity (macOS GUI PATH-stripping). D-04 correctly generalizes absolute-path resolution to ALL agents for the Go port — do not regress to a bare command for "parity," since D-04 is a deliberate, already-decided improvement, not a fresh choice.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|--------------|-----|
| Comment-preserving JSON editing for opencode | A custom JSONC tokenizer/text-span editor | `github.com/tailscale/hujson` | RFC-6902 JSON Patch + comment preservation is exactly what's needed; hand-rolling a tokenizer to track comment positions through structural edits is high-risk, low-payoff work that hujson has already solved and tested |
| Fulcio certificate chain / Rekor transparency-log inclusion-proof / TUF trust-root verification | Custom crypto verification against Sigstore's public-good infra | `github.com/sigstore/sigstore-go` | This is exactly the class of "looks simple, is a minefield" cryptographic verification the project's own `.claude/CLAUDE.md` Option-C reasoning (re: pure-Go tree-sitter reimplementation) warns against — get it wrong once and `upgrade` silently accepts a forged binary |
| GitHub Releases "latest" resolution without hitting the rate-limited API | A hand-rolled API client with manual backoff/retry | The redirect-trick TS already validated: `GET https://github.com/<repo>/releases/latest` (unauthenticated, no rate limit) and parse the `Location` header's `/releases/tag/vX.Y.Z`, falling back to the API only if that fails | TS's own comment cites issue #325 — the unauthenticated GitHub API is rate-limited to 60 req/h/IP and 403s on shared/cloud hosts; the redirect trick has no such limit. Proven in production by the parity oracle. |

**Key insight:** Every "don't hand-roll" item in this phase is a case where the *TS parity oracle itself already chose not to hand-roll* — reuse its choices rather than re-deriving them from first principles.

## Corrected Per-Agent Parity Table (supersedes CONTEXT.md D-05/D-05a/D-06)

CONTEXT.md D-06 flagged Hermes/Antigravity/Kiro as having "NO TS-parity reference" and recommended a documented-partial fallback. **This is now resolved** — TS `src/installer/targets/{hermes,antigravity,kiro}.ts` implement all three fully, tested (1711-line test suite covers every one). The table below is sourced directly from the cloned TS repo (`colbymchenry/codegraph`, `src/installer/targets/*.ts`, read in full during this research session) — treat as `[CITED: colbymchenry/codegraph source, HEAD as of 2026-07-12]` throughout, cross-checked against live agent docs for Hermes and Kiro (`[VERIFIED: hermes-agent.nousresearch.com docs]`, `[VERIFIED: kiro.dev docs]`).

| Agent | Global config | Local config | Format / key shape | Instructions file | Location support | Notable quirks |
|---|---|---|---|---|---|---|
| **Claude Code** | `~/.claude.json` → `mcpServers.codegraph` | `./.mcp.json` → `mcpServers.codegraph` | JSON, `{type:"stdio", command, args:["serve","--mcp"]}` | `~/.claude/CLAUDE.md` / `./.claude/CLAUDE.md` — SHORT marker block (#704) | global + local | Also writes `~/.claude/settings.json` (or `./.claude/settings.json`) `permissions.allow += ["mcp__codegraph__*"]` when `autoAllow`. Migrates/strips a legacy pre-#207 `./.claude.json` local entry (Claude Code never reads that path). Self-heals stale pre-0.8 auto-sync hooks (`mark-dirty`/`sync-if-dirty` — **N/A for Go, no such legacy exists**). TS also has an opt-in `UserPromptSubmit` front-load hook (`codegraph prompt-hook`) — **out of scope for Go v1** (no `prompt-hook` subcommand exists or is planned; do not port this specific feature, it depends on a TS-only command). |
| **Cursor** | `~/.cursor/mcp.json` → `mcpServers.codegraph` | `./.cursor/mcp.json` → `mcpServers.codegraph` | JSON, `{type:"stdio", command, args:["serve","--mcp","--path",X]}` | **NONE** — a legacy `.cursor/rules/codegraph.mdc` file is actively self-heal-DELETED if a prior install left one; do not (re)write it | global + local | `--path` quirk: `local`→absolute cwd, `global`→literal `${workspaceFolder}` string (Cursor expands it). No permissions concept. |
| **Codex CLI** | `~/.codex/config.toml` → `[mcp_servers.codegraph]` (TOML) | *(unsupported — global only)* | TOML, `command = "..."`, `args = ["serve", "--mcp"]` | `~/.codex/AGENTS.md` — SHORT marker block | **global only** | Hand-rolled TOML table splice (see Pattern above / toml.go). `SupportsLocation("local")` returns false. |
| **opencode** | `~/.config/opencode/opencode.jsonc` (or `.json` if that's what exists) → `mcp.codegraph` | `./opencode.jsonc` (or `./opencode.json`) → `mcp.codegraph` | JSONC, `{$schema, mcp:{codegraph:{type:"local", command:["<bin>","serve","--mcp"], enabled:true}}}` | `~/.config/opencode/AGENTS.md` / `./AGENTS.md` — SHORT marker block | global + local | **XDG_CONFIG_HOME always wins, on every platform including Windows** — opencode never reads `%APPDATA%`; sweep/self-heal a legacy `%APPDATA%/opencode` entry if `APPDATA` is set and differs from the XDG path (issue #535 in TS). Prefers `.jsonc` if it exists, else `.json`, else defaults to `.jsonc` for new files. `command` is a combined `[binary, ...args]` array (opencode's own convention, differs from every other JSON-shaped target). |
| **Gemini CLI** | `~/.gemini/settings.json` → `mcpServers.codegraph` | `./.gemini/settings.json` → `mcpServers.codegraph` | JSON, `{type:"stdio", command, args:["serve","--mcp"]}` | `~/.gemini/GEMINI.md` (global) / **`./GEMINI.md` at project root, NOT under `.gemini/`** (local) — SHORT marker block | global + local | No permissions concept — Gemini gates per-server via a `trust` field this installer leaves unset. |
| **Antigravity IDE** | `~/.gemini/config/mcp_config.json` (unified, post-migration) OR `~/.gemini/antigravity/mcp_config.json` (legacy, pre-migration) → `mcpServers.codegraph` | *(unsupported — global only)* | JSON, `{command, args:["serve","--mcp"]}` — **NO `type` field** (Antigravity rejects entries carrying `type:"stdio"`) | **NONE of its own** — shares `~/.gemini/GEMINI.md`, which is written only by the Gemini target, not this one | **global only** | Detect the unified vs legacy path via a `~/.gemini/config/.migrated` marker file OR unified-file-already-exists; on install, sweep a stale legacy entry into the unified path. macOS-only: resolve `codegraph` to its **absolute path** via `command -v`/`which` (GUI apps launched from Dock/Finder get a stripped PATH that misses nvm-managed installs) — Go's D-04 already resolves absolute paths for every agent, so this quirk is automatically satisfied; no Antigravity-specific PATH logic needed in the Go port. |
| **Hermes Agent** | `$HERMES_HOME/config.yaml` (default `~/.hermes` if `HERMES_HOME` unset) → top-level `mcp_servers.codegraph` **+** `platform_toolsets.cli` list append | *(unsupported — global only)* | YAML — hand-rolled line-range surgery (Go: same approach, no YAML library), `mcp_servers: {codegraph: {command, args:[serve,--mcp], timeout:120, connect_timeout:60, enabled:true}}` plus `platform_toolsets: {cli: [...,"mcp-codegraph"]}` | **NONE** — Hermes has no AGENTS.md-equivalent instructions convention TS integrates with | **global only** | The `platform_toolsets.cli` append matters: without `mcp-codegraph` in that list, Hermes CLI profiles can filter out an otherwise-connected MCP server's tools from normal sessions. TS's own YAML surgery handles PyYAML's `default_flow_style=False` list-at-same-indent quirk (`- item` at same 2-space indent as `cli:` parent) — a real Hermes-emitted-YAML edge case (issue #456 in TS), reproduce this indent-detection logic exactly rather than assuming a fixed 4-space indent. `[VERIFIED: hermes-agent.nousresearch.com docs]` confirms `~/.hermes/config.yaml` + `mcp_servers` key as of 2026; note the docs also mention a 30s auto-reload timeout on config edits from within a running session — not installer-relevant but useful context for the instructions/notes text. |
| **Kiro** | `~/.kiro/settings/mcp.json` → `mcpServers.codegraph` | `./.kiro/settings/mcp.json` → `mcpServers.codegraph` | JSON, `{type:"stdio", command, args:["serve","--mcp"]}` | **NONE** — a legacy `~/.kiro/steering/codegraph.md` (or `./.kiro/steering/codegraph.md`) is actively self-heal-DELETED if a prior install left one | global + local | Kiro IDE ships with MCP support **disabled by default** even with a valid config file present — surface a install-time note telling IDE users to enable it in Settings (Kiro CLI users don't need this step). No permissions concept. `[VERIFIED: kiro.dev/docs/mcp/]` confirms JSON MCP config + `.kiro/steering/` directory conventions as of 2026, consistent with the TS source. |

**Summary — instructions files, corrected:** Only **Claude, Codex, opencode, Gemini** (4 of 8) get a marker-fenced instructions file written. **Cursor, Antigravity, Hermes, Kiro** (4 of 8) do not — for Cursor/Kiro this is an explicit *self-heal removal* of a legacy file if one exists from an older install; for Antigravity/Hermes no such file concept was ever implemented. This is the single most consequential correction from this research — CONTEXT.md D-05 correctly had Cursor as "none" but didn't anticipate that Kiro (unlisted) would also be "none," nor that Antigravity/Hermes have no instructions surface at all.

**`--target auto` fallback behavior (not covered by D-03, recommend matching TS):** When `auto` detects zero installed agents, TS falls back to `['claude']` rather than installing nothing — "least-surprise for existing users." Recommend the Go port match this exactly; otherwise a clean-environment `codegraph install` with no `--target` and no TTY does nothing, which is a confusing empty-success. This is a small, cheap parity item CONTEXT.md's discretion section doesn't explicitly rule on — flagging as a recommendation, not a locked decision.

## Common Pitfalls

### Pitfall 1: Treating "5 TS-covered agents" as the parity boundary
**What goes wrong:** Building only Claude/Cursor/Codex/opencode/Gemini against real TS parity and inventing Hermes/Antigravity/Kiro from scratch (as CONTEXT.md's D-06 anticipated might be necessary) produces avoidable divergence — three files' worth of decisions that didn't need to be made from scratch.
**Why it happens:** CONTEXT.md was written before this research confirmed TS's actual coverage; the assumption was reasonable given the info available at discuss-time.
**How to avoid:** Use the Corrected Per-Agent Parity Table above for all 8 agents; it is source-verified, not guessed.
**Warning signs:** A plan or PR description mentioning "documented-partial" for Hermes/Antigravity/Kiro — that language should not appear in this phase's output now that real source parity exists.

### Pitfall 2: Writing an instructions file for Cursor, Antigravity, Hermes, or Kiro
**What goes wrong:** A naive reading of "4 agents get instructions files, so probably all 8 should for consistency" adds unwanted, undocumented files these agents don't expect, and for Cursor/Kiro specifically breaks the TS-compatible round-trip (uninstall would need to remove a file TS's uninstall never wrote).
**Why it happens:** The pattern looks uniform at a glance (JSON MCP entry + instructions file, repeated 4 times) and it's tempting to generalize.
**How to avoid:** Follow the per-agent table exactly — `HasInstructions bool` (or simply: only 4 of 8 `AgentTarget.Install()` implementations call the shared `upsertInstructionsEntry` helper).
**Warning signs:** A `describePaths()` implementation for Cursor/Antigravity/Hermes/Kiro that lists more than the MCP config path (Cursor's describePaths in TS additionally lists the now-legacy rules path only for the self-heal check, not as an ongoing write target).

### Pitfall 3: Claude Code local scope writing to the wrong file
**What goes wrong:** Writing the local-scope MCP entry to `./.claude.json` instead of `./.mcp.json` — Claude Code silently never reads the former (TS issue #207), so the install "succeeds" but the agent never sees the server.
**Why it happens:** `~/.claude.json` (global) and `./.claude.json` (a plausible-looking local mirror) look like a natural pairing; the actual local file (`./.mcp.json`) has a different name.
**How to avoid:** `./.mcp.json` for local, `~/.claude.json` for global — different filenames, not just different directories. Additionally: on install, check for and migrate a legacy `./.claude.json` codegraph entry into `./.mcp.json`; on uninstall, strip it from both locations.
**Warning signs:** A test asserting the local Claude write path without an explicit assertion on the exact filename `.mcp.json` (not `.claude.json`).

### Pitfall 4: opencode config-dir resolution special-casing Windows
**What goes wrong:** Writing global opencode config to `%APPDATA%/opencode` on Windows "because that's the Windows convention" — opencode itself never reads that path (only `XDG_CONFIG_HOME` or `~/.config`, unconditionally, on every platform including Windows). A Windows-special-cased Go install would silently never be seen by opencode (mirrors TS issue #535, which TS itself only fixed later).
**Why it happens:** Reasonable Windows-idiom instinct; wrong for this specific tool because opencode deliberately doesn't follow Windows convention here.
**How to avoid:** Resolve exactly as TS does: `XDG_CONFIG_HOME` if set and non-empty, else `~/.config`, unconditionally on every OS. Additionally, self-heal by sweeping a stale `%APPDATA%/opencode` codegraph entry (only if `APPDATA` is set and differs from the resolved XDG path) so a user who installed the Go binary after a broken pre-fix TS install gets cleaned up.
**Warning signs:** Any `runtime.GOOS == "windows"` branch inside opencode's config-dir resolution.

### Pitfall 5: Hermes YAML surgery assuming a fixed list-item indent
**What goes wrong:** PyYAML's default serializer (`default_flow_style=False`) emits list items at the SAME indent level as their parent key, not indented further (`cli:\n- item` not `cli:\n  - item`). A naive "look for a line starting with 4+ spaces and `-`" heuristic mis-locates or truncates the `platform_toolsets.cli` block (TS's own issue #456).
**Why it happens:** Most hand-written YAML examples show deeper list indentation; PyYAML's actual default output looks unusual by comparison.
**How to avoid:** Detect the *actual* existing indent used by list items in the target block (as TS's `listChildBlock` does) rather than assuming a fixed indent, and match it when inserting new items.
**Warning signs:** A test fixture that only exercises hand-authored (4-space-indented) YAML and never a PyYAML-style (0-extra-indent) config.

### Pitfall 6: Antigravity's `type` field
**What goes wrong:** Reusing the shared `{type:"stdio", command, args}` JSON entry builder for Antigravity — Antigravity's own config UI rejects (or silently ignores registration for) entries carrying a `type` field; the working pattern observed in the wild omits it entirely.
**Why it happens:** 6 of the other 7 agents use the exact same `{type:"stdio",...}` shape, so it's tempting to always reuse one shared builder function.
**How to avoid:** Antigravity gets its own entry-builder (`{command, args}`, no `type` key) — do not route it through the shared JSON-entry helper used by Claude/Cursor/Gemini/Kiro.
**Warning signs:** A single `buildMcpEntry()` function called unconditionally by all JSON-shaped targets, with no Antigravity-specific override.

### Pitfall 7: `upgrade` swap without verifying first, or verifying-then-still-swapping-on-failure
**What goes wrong:** Any code path where the binary is written to the final path before signature verification completes, or where a verification error is logged-and-continued rather than aborting.
**Why it happens:** The natural download→write→verify ordering (verify what's on disk) is actually *wrong* for this use case — D-12 requires download→verify (in memory or a quarantined temp location)→swap, never download→swap→verify-after-the-fact.
**How to avoid:** Structure `internal/upgrade` so the temp file is verified via `sigstore-go` BEFORE the `os.Rename` swap step runs; treat any verification error as fatal (no partial state, clear error, exit non-zero) — never fall through to swap on a verification failure.
**Warning signs:** A test that only exercises the happy (verified) path — Wave 0 must include an explicit "tampered/unverifiable bundle → swap does NOT happen, original binary untouched" test.

## Code Examples

### Marker-fenced section upsert (the shared primitive 4 of 8 targets use)

```go
// Source: mechanical Go port of src/installer/targets/shared.ts
// replaceOrAppendMarkedSection / removeMarkedSection
package agents

import (
	"os"
	"strings"
)

func replaceOrAppendMarkedSection(filePath, body, startMarker, endMarker string) (FileAction, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		if err := atomicWriteFile(filePath, body+"\n"); err != nil {
			return "", err
		}
		return ActionCreated, nil
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	s := string(content)
	startIdx := strings.Index(s, startMarker)
	endIdx := strings.Index(s, endMarker)

	if startIdx != -1 && endIdx > startIdx {
		existingBlock := s[startIdx : endIdx+len(endMarker)]
		if existingBlock == body {
			return ActionUnchanged, nil
		}
		newContent := s[:startIdx] + body + s[endIdx+len(endMarker):]
		if err := atomicWriteFile(filePath, newContent); err != nil {
			return "", err
		}
		return ActionUpdated, nil
	}

	// No markers — append, preserving existing content.
	trimmed := strings.TrimRight(s, "\n \t")
	sep := ""
	if len(trimmed) > 0 {
		sep = "\n\n"
	}
	if err := atomicWriteFile(filePath, trimmed+sep+body+"\n"); err != nil {
		return "", err
	}
	return ActionUpdated, nil // TS reports "appended" mapped to "updated" by the caller
}
```

### sigstore-go verification wiring for `upgrade` (adapted for a GitHub-Actions-OIDC release identity)

```go
// Source: Context7 /sigstore/sigstore-go — verify.NewVerifier / verify.NewPolicy /
// verify.NewShortCertificateIdentity, adapted to this project's release identity shape.
package upgrade

import (
	"github.com/sigstore/sigstore-go/pkg/bundle"
	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/verify"
)

// verifyRelease checks a downloaded binary's bytes against its accompanying
// cosign-keyless bundle. releaseRepoSlug and workflowRef are named
// constants Phase 8 finalizes to the project's real GitHub Actions OIDC
// identity (D-14) — this phase wires the verification path against test
// fixtures, not production identity.
func verifyRelease(bundlePath string, artifactDigest []byte, releaseRepoSlug string) error {
	b, err := bundle.LoadJSONFromPath(bundlePath)
	if err != nil {
		return err
	}

	trustedRoot, err := root.FetchTrustedRoot() // Sigstore public-good TUF root
	if err != nil {
		return err
	}

	verifier, err := verify.NewVerifier(trustedRoot,
		verify.WithSignedCertificateTimestamps(1),
		verify.WithTransparencyLog(1),
		verify.WithObserverTimestamps(1),
	)
	if err != nil {
		return err
	}

	certID, err := verify.NewShortCertificateIdentity(
		"https://token.actions.githubusercontent.com", // GitHub Actions OIDC issuer
		"",
		"",
		"^https://github.com/"+releaseRepoSlug+"/", // this repo's own workflow identity
	)
	if err != nil {
		return err
	}

	policy := verify.NewPolicy(
		verify.WithArtifactDigest("sha256", artifactDigest),
		verify.WithCertificateIdentity(certID),
	)

	_, err = verifier.Verify(b, policy) // non-nil err = verification failed; caller MUST abort the swap
	return err
}
```

### GitHub Releases "latest" resolution without hitting the rate-limited API

```go
// Source: mechanical Go port of src/upgrade/index.ts resolveLatestVersion —
// the redirect trick (unauthenticated github.com/.../releases/latest has no
// rate limit; the API does).
package upgrade

import (
	"errors"
	"net/http"
	"regexp"
)

var tagFromLocation = regexp.MustCompile(`/releases/tag/([^/?#]+)`)

func resolveLatestVersion(repoSlug string) (string, error) {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // capture the redirect, don't follow it
		},
	}
	resp, err := client.Get("https://github.com/" + repoSlug + "/releases/latest")
	if err == nil {
		defer resp.Body.Close()
		if loc := resp.Header.Get("Location"); loc != "" {
			if m := tagFromLocation.FindStringSubmatch(loc); m != nil {
				return m[1], nil
			}
		}
	}
	// Fall back to the (rate-limited) API only if the redirect trick fails.
	return resolveLatestVersionViaAPI(repoSlug)
}

func resolveLatestVersionViaAPI(repoSlug string) (string, error) {
	// GET https://api.github.com/repos/<repoSlug>/releases/latest, parse
	// {"tag_name": "..."} — omitted here; see TS src/upgrade/index.ts for
	// the exact fallback shape (Accept: application/vnd.github+json).
	return "", errors.New("not implemented in this snippet — see TS fallback for full shape")
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|-------------------|---------------|--------|
| TS installer wrote a full usage playbook into every agent's instructions file | Short marker block pointing at `codegraph_explore`/`codegraph explore`; MCP `initialize` response is the single source of truth for full tool guidance | Issue #529 (removal), then #704 (short block reintroduced for subagents/non-MCP-harness coverage) | The Go port must write the SHORT block (already what CONTEXT.md's `.claude/CLAUDE.md` §"CodeGraph" section shows) — never the old full playbook |
| TS installer wrote `.cursor/rules/codegraph.mdc` and `.kiro/steering/codegraph.md` | Both removed entirely; both targets now self-heal-delete a legacy file left by an older install | Same #529 cleanup wave | Confirms these two agents get NO instructions file in the Go port either — and both need an active "delete if present" step on install (self-heal) as well as uninstall |
| `cosign` CLI shelled out to for verification | In-process `sigstore-go` library verification | Sigstore ecosystem guidance as of 2026 (per WebSearch: "sigstore-go is recommended for use in verification... cosign is not recommended for integration") | Directly informs D-12's library choice — this is the currently-recommended pattern, not a stopgap |
| `inconshreveable/go-update` | Archived April 2026; `minio/selfupdate` is the maintained fork | Confirmed via WebSearch during this research | Neither is recommended for adoption this phase regardless (see Alternatives Considered) — noted for completeness since a researcher revisiting self-update libraries later should know the original is dead |

**Deprecated/outdated:**
- `inconshreveable/go-update`: archived by its owner, April 2026 — do not depend on it directly even as a pattern reference; use `minio/selfupdate`'s fork (still not recommended as a dependency, but a live reference).

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|----------------|
| A1 | `github.com/sigstore/sigstore-go` and `github.com/tailscale/hujson` are the correct package import paths | Standard Stack | Low — both existence and current version were independently VERIFIED against the Go module proxy in this session; only the *choice* of these specific packages (vs. an unconsidered alternative) is unverified-by-authoritative-source |
| A2 | Hermes Agent's MCP config conventions haven't changed since the TS source (dated ~2026-05 per its own file comments) was written | Corrected Per-Agent Parity Table | Medium — cross-checked against live `hermes-agent.nousresearch.com` docs during this session and found consistent (`~/.hermes/config.yaml`, `mcp_servers` key), but the platform is actively developed (v0.7.0 released April 2026 per the same search) and could shift again before Phase 6 executes |
| A3 | Kiro's MCP/steering conventions haven't changed since the TS source was written | Corrected Per-Agent Parity Table | Medium — cross-checked against `kiro.dev/docs/mcp/` and found consistent; Kiro is an actively developed AWS product and the doc cross-check was a search-summary, not a full page read |
| A4 | Antigravity's unified-vs-legacy config migration marker (`~/.gemini/config/.migrated`) still reflects the current Antigravity release | Corrected Per-Agent Parity Table | Medium-High — this was NOT independently cross-checked against Antigravity's own docs (no live doc cross-check performed for Antigravity in this session, only TS source read); Antigravity is explicitly called out in the TS source's own comments as being "in the process of consolidating" — the fastest-moving of the 3 previously-unresearched agents |
| A5 | `--target auto` falling back to `['claude']` when nothing is detected is the right behavior to port (not explicitly locked by CONTEXT.md D-03) | Corrected Per-Agent Parity Table, closing note | Low — a UX recommendation, not a correctness risk; worst case is a plan that doesn't include this and produces a slightly confusing empty-success on a clean environment |

## Open Questions

1. **How should Phase 6 generate a test fixture for sigstore-go's Fulcio+Rekor+TUF keyless verification path, given Phase 8 hasn't produced real signed releases yet?**
   - What we know: D-14 says "testable now against fixtures (a signed test artifact)... Do not stub the verification." `sigstore-go`'s own repo ships example bundle fixtures (`examples/bundle-provenance.json`, referenced in its docs) used for its own tests, and the library also supports a `WithKey()` policy for simpler long-lived-key (non-Fulcio) bundles that don't require network calls to a live TUF root during tests.
   - What's unclear: Whether to (a) point Phase 6's tests at `sigstore-go`'s own packaged example bundle (proves the wiring/policy-construction/error-handling code path end-to-end against a real, valid bundle, but tests against someone else's identity, not codegraph-go's own), or (b) build a fully offline test harness using `verify.WithKey()` + a locally-generated test keypair (avoids any network dependency in `go test`, but doesn't exercise the actual Fulcio/Rekor/TUF chain D-12 requires in production), or (c) something else.
   - Recommendation: Use approach (a) for the "does our verification code correctly accept a valid bundle and reject a tampered one" tests (fast, no network needed beyond a one-time `root.FetchTrustedRoot()` call which itself may need a fixture/mock for CI hermeticity — check `sigstore-go`'s own test suite for how it mocks/fixtures `TrustedRoot` offline), and treat wiring the *production* identity constants as the explicit Phase-8 finalize step D-14 already anticipates. This needs a planner decision, not a research-time lock.

2. **Should the Go port replicate TS's `install --refresh` self-heal-on-upgrade behavior (re-running install for already-configured agents only, to pick up template changes from a newer binary)?**
   - What we know: TS's `runUpgrade` calls `codegraph install --refresh` after a successful upgrade specifically so the marker-fenced instruction blocks (baked into the binary, not version-agnostic) get updated to match the new version's wording. `refreshTargets()` in `index.ts` is a clean, separately-tested pure function for this.
   - What's unclear: CONTEXT.md's decisions (D-11 through D-15) don't mention a `--refresh` flag or self-heal-on-upgrade behavior at all — it may be considered out of scope for a v1 Go port's `upgrade` command, or it may be an oversight.
   - Recommendation: Flag for the planner/discuss-phase to explicitly decide in/out of scope. If in scope, it's a small addition (`install --refresh` iterating only `alreadyConfigured` targets with `autoAllow:false`) layered on top of the already-planned `install` command — low cost if wanted, no regression if omitted, since the marker-block content in a Go v1 phase 6 is unlikely to change version-to-version within this same milestone.

3. **Does the project want to reproduce TS's `codegraph install --print-config <id>` (print the MCP snippet for manual paste, no filesystem writes)?**
   - What we know: TS implements this via `target.printConfig(loc)` and every target file already has this method (D-02's proposed `AgentTarget` interface in CONTEXT.md doesn't list it, but adding it is a ~5-line addition per target given `Install()` already builds the same entry).
   - What's unclear: Not mentioned anywhere in CONTEXT.md's decisions or discretion list.
   - Recommendation: Low-cost nice-to-have; the planner can include or defer without research-time consequence. If included, `PrintConfig` should be added to the `AgentTarget` interface (Pattern 1 above) alongside `DescribePaths`.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|--------------|-----------|---------|----------|
| Go toolchain | Building `internal/agents`, `internal/upgrade`, `internal/version` | ✓ | go 1.26.5 (per go.mod) | — |
| Network access to `proxy.golang.org` | `go get` for `sigstore-go`/`hujson` | ✓ (confirmed live during this research session) | — | — |
| Network access to `github.com` (redirect endpoint) | `resolveLatestVersion` production behavior; NOT required for `go test` (should be mocked/injected per TS's own `UpgradeDeps.resolveLatest` injection pattern) | N/A at research time — this is a runtime dependency of the shipped binary, not a build dependency | — | If offline, `upgrade`/`upgrade --check` fail with a clear network error (matches TS's own error message pattern: "could not resolve the latest version from GitHub. Check your network, or pin a version") |
| Sigstore public-good infra (Rekor, Fulcio CT logs, TUF root) | `upgrade`'s verification step, production only | Not directly probed (external service, assumed available per Sigstore's own uptime SLOs) | — | None at the architecture level — D-12 mandates verification is non-optional; an outage here should fail the upgrade closed (no swap), not open |
| A TTY (for the interactive multi-select) | `install`/`uninstall` interactive mode (D-03) | Session-dependent — not a fixed environment fact | — | Falls back to `auto` (install) / `all` (uninstall) with no prompting, per D-03/D-08 — already the correct designed fallback, no gap |

**Missing dependencies with no fallback:** none identified — every external dependency this phase introduces (module proxy for `go get`, GitHub Releases for `upgrade`, Sigstore infra for verification) is either a build-time-only dependency (not shipped-binary-critical) or has an explicit fail-closed behavior already specified by CONTEXT.md's decisions.

**Missing dependencies with fallback:** No TTY → falls back to `auto`/`all` (already designed). Offline at `upgrade` runtime → clear error, no partial state (already designed by D-12/D-13).

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go's built-in `testing` package (stdlib) — no third-party test framework in this project (per `go.mod`, only `go.uber.org/goleak` for soak tests and `google/go-cmp` for diffing, both already direct deps) |
| Config file | none — `go test ./...` is the project's own convention (see `internal/cli/cli_test.go`'s `copyFixture(t)` pattern using `t.TempDir()`) |
| Quick run command | `go test ./internal/agents/... ./internal/upgrade/... ./internal/version/... -run <TestName>` |
| Full suite command | `go test ./... -race` (matches the project's established `-race` convention from Phase 2's determinism-gate testing) |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|---------------------|--------------|
| AGNT-01 | `install` writes correct config for each of 8 agents; idempotent re-run reports `unchanged` everywhere | unit (table-driven, one subtest per agent × location) | `go test ./internal/agents/... -run TestInstall` | ❌ Wave 0 |
| AGNT-01 | `install` preserves a pre-existing sibling MCP server / unrelated JSON keys | unit | `go test ./internal/agents/... -run TestInstall_PreservesSiblingConfig` | ❌ Wave 0 |
| AGNT-02 | `uninstall` reverses `install` exactly — round-trip byte-invariant (file returns to pre-install bytes modulo the CodeGraph section) | unit | `go test ./internal/agents/... -run TestInstallUninstall_RoundTrip` | ❌ Wave 0 |
| AGNT-02 | `uninstall` reports `removed`/`not-configured`/`unsupported` correctly per agent/location combination | unit | `go test ./internal/agents/... -run TestUninstallStatus` | ❌ Wave 0 |
| AGNT-03 | Cursor `--path` quirk: local=abs cwd, global=`${workspaceFolder}` literal | unit | `go test ./internal/agents/... -run TestCursor_PathQuirk` | ❌ Wave 0 |
| AGNT-03 | opencode preserves comments through install + idempotent re-run | unit | `go test ./internal/agents/... -run TestOpencode_PreservesComments` | ❌ Wave 0 |
| AGNT-03 | Hermes preserves PyYAML-default list-at-same-indent style (issue #456 equivalent) | unit | `go test ./internal/agents/... -run TestHermes_IndentPreservation` | ❌ Wave 0 |
| AGNT-03 | Antigravity entry has no `type` field; unified-vs-legacy path detection | unit | `go test ./internal/agents/... -run TestAntigravity` | ❌ Wave 0 |
| CLI-01 | `codegraph version` / `--version` prints ldflags-injected values; `--json` produces valid JSON | unit | `go test ./internal/cli/... -run TestVersion` | ❌ Wave 0 |
| CLI-01 | `codegraph help <command>` produces non-empty, command-specific output for every registered command | unit | `go test ./internal/cli/... -run TestHelp` | ❌ Wave 0 |
| CLI-02 | `upgrade` rejects a tampered/unverifiable bundle without swapping (original binary untouched) | unit | `go test ./internal/upgrade/... -run TestVerify_RejectsTampered` | ❌ Wave 0 |
| CLI-02 | `upgrade` accepts a valid fixture bundle and proceeds to swap; `--check` compares versions without downloading | unit | `go test ./internal/upgrade/... -run TestUpgrade` | ❌ Wave 0 |
| CLI-02 | Atomic swap: temp-in-same-dir + `os.Rename` on POSIX; rename-aside dance on Windows (build-tag-gated or `runtime.GOOS`-branched test) | unit | `go test ./internal/upgrade/... -run TestSwap` | ❌ Wave 0 |
| CLI-03 | `codegraph telemetry` output is static and does not perform any network call | unit (assert no network I/O attempted — e.g. via a `net.Dial`-intercepting test double, or simply asserting the function performs zero I/O by inspection/coverage) | `go test ./internal/cli/... -run TestTelemetry` | ❌ Wave 0 |

### Sampling Rate

- **Per task commit:** targeted `go test ./internal/agents/... -run <relevant subtest>` (fast, no `-race` needed for single-target file-write tests)
- **Per wave merge:** `go test ./internal/agents/... ./internal/upgrade/... ./internal/cli/... -race`
- **Phase gate:** `go test ./... -race` green before `/gsd-verify-work`

### Wave 0 Gaps

- [ ] `internal/agents/agents_test.go` — shared test harness: `t.Setenv("HOME", t.TempDir())` (and `t.Setenv("XDG_CONFIG_HOME", ...)`, `t.Setenv("HERMES_HOME", ...)` per-target as needed) so every agent's global-scope test runs against an isolated fake home directory, mirroring TS's per-test `os.homedir()` mocking
- [ ] `internal/agents/testdata/` — fixture files for "pre-existing sibling MCP server," "legacy pre-#207 `.claude.json`," "PyYAML-default-indent Hermes config," etc. — one fixture per Common Pitfall above
- [ ] `internal/upgrade/testdata/` — a signed test bundle + corresponding artifact (see Open Question 1) for the sigstore-go verification round-trip
- [ ] Framework install: none — `go test` is already fully available, no new test framework needed

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|----------------|---------|--------------------|
| V2 Authentication | No | This phase has no user-authentication surface |
| V3 Session Management | No | No sessions |
| V4 Access Control | Partial | `upgrade`'s "refuse to upgrade a binary the current user can't write" (D-13) is a local filesystem-permission check, not a traditional access-control surface — implement via `os.Rename`/`os.Chmod` error handling, not a custom permission model |
| V5 Input Validation | Yes | Every agent config write parses untrusted-in-the-sense-of-user-edited external files (JSON/JSONC/TOML/YAML) — must not panic or corrupt on malformed input; TS's own pattern (backup-then-treat-as-empty on unparseable JSON) is the right model, port it |
| V6 Cryptography | Yes | `upgrade`'s signature verification is THE cryptography surface of this phase — never hand-roll (see Don't Hand-Roll); use `sigstore-go` exclusively, never implement custom hash/signature comparison logic even for the "does this artifact match its declared digest" sub-step |
| V12 File and Resources | Yes | Every write in this phase (agent configs, the swapped binary itself) must use the atomic-write pattern (temp file + rename) already established by TS's `atomicWriteFileSync` and required by D-13 for the binary swap specifically — prevents partial-write corruption on crash/interrupt |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|------------------------|
| Malicious/malformed agent config file causes a panic or silent data-loss on `install` | Denial of Service / Tampering | Parse defensively (empty-map fallback + backup-before-overwrite on unparseable JSON, matching TS's `readJsonFile` pattern); never `panic` on malformed user input, always return an error the CLI surfaces cleanly |
| A compromised or MITM'd GitHub Releases response serves a malicious binary to `upgrade` | Tampering / Spoofing | This IS the entire reason D-11/D-12 mandate mandatory in-process verification before swap — sigstore-go's Fulcio-certificate-identity check (pinned to this repo's own GitHub Actions OIDC workflow identity, D-14) is the mitigation; HTTPS transport security alone (what TS relies on today) is explicitly NOT sufficient per this project's stronger trust bar |
| A partially-written binary swap (crash mid-`os.Rename` or mid-download) leaves the user with a broken `codegraph` install | Denial of Service | Atomic swap only: download+verify fully into a temp file first, `os.Rename` is the only step that touches the final path, and `os.Rename` within the same filesystem/directory is atomic on POSIX; Windows rename-aside dance achieves the same property with an extra step. Never write directly to the live binary path. |
| A stale/shadowing second `codegraph` install elsewhere on `PATH` causes `upgrade` to silently "succeed" while the user's terminal keeps resolving the old binary | Tampering (of user expectation, not data) — TS's own issue #1071 | TS's `verifyResolvedVersion` post-upgrade probe (spawn `codegraph --version` via PATH, compare to target) is a UX-quality mitigation, not a security control per se — worth porting as a nice-to-have but not a hard requirement of this phase's locked decisions; flagged here for awareness, not blocking |

## Sources

### Primary (HIGH confidence)
- `colbymchenry/codegraph` (shallow-cloned to scratchpad, read in full: `src/installer/targets/{types,shared,registry,claude,cursor,codex,opencode,gemini,antigravity,hermes,kiro,toml}.ts`, `src/installer/{index,config-writer,instructions-template}.ts`, `src/upgrade/{index,update-check}.ts`, `src/mcp/version.ts`, `src/bin/codegraph.ts` (install/uninstall/upgrade/version/telemetry command wiring), `__tests__/installer-targets.test.ts` (test-case inventory, 1711 lines / ~90 cases)) — the behavioral parity oracle CONTEXT.md's canonical_refs names explicitly
- `github.com/sigstore/sigstore-go` via Context7 (`/sigstore/sigstore-go`, High source reputation, 93.12 benchmark score) — `verify.NewVerifier`/`verify.NewPolicy`/`verify.NewShortCertificateIdentity` API surface
- Go module proxy (`proxy.golang.org`) direct queries — live-verified existence + current version of `sigstore/sigstore-go` (v1.2.2), `tailscale/hujson` (pseudo-version 2026-03-02), `minio/selfupdate` (v0.6.0)
- `pkg.go.dev/github.com/tailscale/hujson` via WebFetch — `Parse`/`Patch`/`Format`/`Pack`/`Find` API surface

### Secondary (MEDIUM confidence)
- WebSearch: "sigstore-go verification-only library... vs cosign CLI" — cross-checked claim that sigstore-go is the currently-recommended in-process verification path and cosign integration is explicitly discouraged for embedding
- WebSearch: "Go self-update binary atomic replace library" — `inconshreveable/go-update` archived-April-2026 status, `minio/selfupdate` fork lineage and last-tag date
- WebSearch: Hermes Agent docs (`hermes-agent.nousresearch.com`) — cross-checked `~/.hermes/config.yaml` + `mcp_servers` key against the TS source's own claim
- WebSearch: Kiro docs (`kiro.dev/docs/mcp/`, `kiro.dev/docs/mcp/configuration/`) — cross-checked JSON MCP config + `.kiro/steering/` conventions against the TS source's own claim

### Tertiary (LOW confidence)
- Antigravity's unified-vs-legacy migration marker path (`~/.gemini/config/.migrated`) — sourced only from the TS installer's own code comments, not independently cross-checked against Antigravity's own docs in this session (see Assumption A4)

## Metadata

**Confidence breakdown:**
- Standard stack (sigstore-go, hujson): MEDIUM — package existence/version VERIFIED via module proxy, API surface CITED via Context7, but package *choice* (vs. unresearched alternatives) rests on WebSearch cross-checks, not an exhaustive survey
- Architecture / per-agent parity table: HIGH for the 5 previously-known agents (direct TS source read) and HIGH for Hermes/Kiro (TS source + independent live-doc cross-check); MEDIUM-HIGH for Antigravity (TS source only, no independent cross-check — see A4)
- Pitfalls: HIGH — every pitfall listed traces to an explicit TS source-code comment documenting a real historical bug (issue numbers cited: #207, #456, #529, #535, #704, #1071), not speculation
- `upgrade` verification design: MEDIUM — the sigstore-go API is CITED/verified via Context7, but no live test fixture has been built yet (Open Question 1); the design itself (verify-before-swap, fail-closed) is a direct, low-risk application of D-12's already-locked requirements

**Research date:** 2026-07-12
**Valid until:** ~30 days for the TS parity table (stable protocol, low churn risk); ~14 days for Hermes/Antigravity/Kiro-specific paths (actively developed agent ecosystem, per this research's own findings about Hermes v0.7.0 and Antigravity's in-progress migration) — re-verify Antigravity's unified-config marker path specifically if phase execution slips more than 2 weeks past this research date; ~30 days for the sigstore-go/hujson dependency choices (stable, slow-moving libraries)
