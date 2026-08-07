# Roadmap: CodeGraph Go

## Overview

CodeGraph Go is a ground-up Go rewrite of TypeScript CodeGraph — a drop-in, TS-v1.3.x-parity replacement in a single static binary. **v0.1 (Initial Release) shipped 2026-07-14**: the core capabilities (indexing, query, MCP server, sync, migration) work from a signed/attested/SBOM'd release that beats TS 1.3.1 on every measured benchmark — but the CLI/agent surface still diverged *behaviorally* from TS. **v1.0 (Drop-in Parity & Human UX) shipped 2026-08-03**: those gaps are closed. An existing user can now swap binaries with zero change in experience — TS-identical `explore`/`node`/`status` behavior, watcher-on-MCP by default, git/worktree awareness, output hygiene, a human-facing Charm TUI behind a build-enforced rendering seam (the agent/MCP path never sees ANSI), systematic flag reconciliation, fully automated signed releases via release-please + GoReleaser, and contributor-facing local build tooling. Work was risk-front-loaded: the load-bearing shared-engine behavioral algorithms landed first, the human TUI last.

**v0.3.0 (MCP Protocol Currency) is in progress.** MCP published spec revision `2026-07-28` six days after v1.0 shipped, which makes codegraph-go's server a **Legacy** implementation in that spec's own terminology. This milestone brings the agent surface current on the official `modelcontextprotocol/go-sdk` — the maintainer's pre-decision, since `mark3labs/mcp-go` has no `2026-07-28` support and no announced timeline — without breaking any of the 8 agent clients `codegraph install` already configures. Work is ordered around one non-negotiable: the wire-level verification oracle is built and proven against the **current** server before any SDK code moves, because this project already established (v1.0 Phase 4) that an SDK's own client silently skips malformed stdout lines and therefore cannot fail a purity test.

**Versioning note:** "v1.0" is a *planning-milestone* name, never a release version. The shipped artifact line is `v0.2.0`, computed by release-please from Conventional Commits; there is deliberately no `v1.0.0` tag (maintainer directive D-06R, 2026-07-29). From v0.3.0 onward the milestone label tracks the actual release line, but it is still a planning label carrying **no git tag** — release-please remains the sole tag authority and computes the real version from Conventional Commits, so `v0.3.0` is a prediction that holds if this milestone lands `feat:` commits. Milestones carry no git tag; the milestone record lives in `MILESTONES.md` + `milestones/`. (`milestone-v0.1` exists only because it predates release-please.)

## Milestones

- ✅ **v0.1 — Initial Release** — Phases 1–8 (shipped 2026-07-14) — core capabilities + signed release; not yet a drop-in parity replacement
- ✅ **v1.0 — Drop-in Parity & Human UX** — Phases 1–10 (shipped 2026-08-03) — behavioral + surface parity with TS 1.3.1, human TUI, automated signed releases, local build tooling
- ✅ **v0.3.0 — MCP Protocol Currency** — Phases 1–5 (shipped 2026-08-06) — official Go SDK adoption, `2026-07-28` spec compliance without breaking Legacy clients, a wire-level verification oracle, tool-modfile vulnerability coverage
- 📋 **Later** — unscoped. Candidates: the Backlog items below (999.2 tmux TTY harness, 999.4 CheckRegression guard, 999.5 macOS Gatekeeper), Team Scale (central server, CI-distributed indexes), MRTR/elicitation (MRTR-01), annotations (embeddings/communities/export), local Svelte web UI (SEED-001), homebrew install path (SEED-002)

## Phases

<details>
<summary>✅ v0.1 — Initial Release (Phases 1–8) — SHIPPED 2026-07-14</summary>

Full phase details archived in [`milestones/v0.1-ROADMAP.md`](milestones/v0.1-ROADMAP.md); phase artifacts in [`milestones/v0.1-phases/`](milestones/v0.1-phases/).

- [x] Phase 1: Foundation — Storage, Schema & Parser Strategy (7/7 plans) — completed 2026-07-10
- [x] Phase 2: Go Indexing Pipeline (6/6 plans) — completed 2026-07-11
- [x] Phase 3: Query Engine & MCP Server (9/9 plans) — completed 2026-07-11
- [x] Phase 4: Incremental Sync & File Watcher (9/9 plans) — completed 2026-07-11
- [x] Phase 5: Language Coverage & Resolution Breadth (14/13 plans) — completed 2026-07-12
- [x] Phase 6: Agent Integrations & CLI Lifecycle (6/6 plans) — completed 2026-07-12
- [x] Phase 7: Migration Tool (7/7 plans) — completed 2026-07-13
- [x] Phase 8: Release Hardening & Benchmarks (9/9 plans) — completed 2026-07-14

**Delivered:** the core capabilities of CodeGraph in a single static Go binary — index/query/MCP/sync/migrate — faster and lighter than TS 1.3.1, from a signed/attested/SBOM'd release verified end-to-end on `v0.0.0-rc.3`. **Known gaps carried into v1.0:** the CLI/agent surface diverges *behaviorally* from TS; DIST-02 (real `v*` tag) and PERF-01 (published numbers) remained pending.

</details>

<details>
<summary>✅ v1.0 — Drop-in Parity & Human UX (Phases 1–10) — SHIPPED 2026-08-03</summary>

Full phase details archived in [`milestones/v1.0-ROADMAP.md`](milestones/v1.0-ROADMAP.md); phase artifacts in [`milestones/v1.0-phases/`](milestones/v1.0-phases/); requirements in [`milestones/v1.0-REQUIREMENTS.md`](milestones/v1.0-REQUIREMENTS.md).

- [x] Phase 1: Behavioral Parity — explore & node (17/17 plans) — completed 2026-07-15
- [x] Phase 2: status Content & Git/Worktree Awareness (7/7 plans) — completed 2026-07-16
- [x] Phase 3: Watcher-on-MCP Default (5/5 plans) — completed 2026-07-16
- [x] Phase 4: Output Hygiene (3/3 plans) — completed 2026-07-16
- [x] Phase 5: Git Sync Hooks (5/5 plans) — completed 2026-07-17
- [x] Phase 6: Rendering Seam & Pretty status/files (3/3 plans) — completed 2026-07-17
- [x] Phase 7: Interactive TUI — Daemon Picker & Install Multi-Select (8/8 plans) — completed 2026-07-26
- [x] Phase 8: Surface Reconciliation & Signed v1.0.0 Release (9/9 plans) — completed 2026-07-28
- [x] Phase 9: release-please + GoReleaser (8/8 plans) — completed 2026-08-01 (promoted from backlog 999.3 on 2026-07-27)
- [x] Phase 10: Local Build Tooling & CONTRIBUTING (7/7 plans) — completed 2026-08-03 (promoted from backlog 999.1 on 2026-07-27)

**Delivered:** drop-in behavioral parity with TS CodeGraph v1.3.x, plus the human-facing surface v0.1 lacked. All 10 phases independently verified (`verification_status: passed`), 48/48 requirements satisfied, 72 plans, 594 commits over 20 days (755 files, +98,761/−1,942 lines). The "not yet drop-in" caveat is retired. Release cutting is fully automated — `v0.2.0` was tagged, changelogged, built, signed, SBOM'd and SLSA-attested with no human running `git tag`, and a genuinely shipped prior binary self-upgraded to it byte-identically to the attested subject.

**Carried forward:** four backlog items (999.2–999.5, below) and the residual darwin release-path check — `release.yml`'s goreleaser/cosign/SLSA steps first execute on the macOS runner class during a real tag push, though a permanent canary already machine-proves that runner's availability and native toolchain.

</details>

<details>
<summary>✅ v0.3.0 — MCP Protocol Currency (Phases 1–5) — SHIPPED 2026-08-06</summary>

Full phase details archived in [`milestones/v0.3.0-ROADMAP.md`](milestones/v0.3.0-ROADMAP.md); phase artifacts in [`milestones/v0.3.0-phases/`](milestones/v0.3.0-phases/); requirements in [`milestones/v0.3.0-REQUIREMENTS.md`](milestones/v0.3.0-REQUIREMENTS.md); audit in [`v0.3.0-MILESTONE-AUDIT.md`](v0.3.0-MILESTONE-AUDIT.md).

- [x] Phase 1: Protocol Scoping & the SDK-Independent Wire Oracle (7/7 plans) — completed 2026-08-05
- [x] Phase 2: SDK Migration — official go-sdk on the existing surface (5/5 plans) — completed 2026-08-06
- [x] Phase 3: `2026-07-28` Spec Compliance (5/5 plans) — completed 2026-08-06
- [x] Phase 4: Supply-Chain Coverage & Daemon Substrate Fixes (3/3 plans) — completed 2026-08-06
- [x] Phase 5: Live Tool-Catalog Change Notification (1/1 plans) — completed 2026-08-06

**Delivered:** codegraph-go's stdio MCP server is current with spec revision `2026-07-28` on `modelcontextprotocol/go-sdk@v1.7.0`, with `mark3labs/mcp-go` gone. All 25 requirements satisfied, all 5 phases independently verified, 21 plans over 136 commits in 4 days (174 files, +32,372/−1,901). Legacy clients are unbroken — a `2024-11-05` client completes a session and calls a tool, asserted on the wire. The verification oracle grew 23 → 28 frozen transcripts and never imports the SDK it tests.

**Accepted limitations, carried not closed:** the daemon extreme-load timeout tail and its feedback-latency tradeoff (CI load ruled the governing standard for MAINT-02), and `GO-2026-5932` — a real, unmitigated vulnerability reachable in `goreleaser`'s binary through cosign/rekor's unmaintained openpgp, surfaced by the new advisory scan rather than resolved.

**No git tag.** release-please is the sole tag authority (D-06R); a `v0.3.0` tag would match `release.yml`'s `v[0-9]*` trigger and falsely fire the release pipeline.

</details>

## Backlog

### Phase 999.2: tmux e2e/UAT test harness and suite (BACKLOG)

**Goal:** [Captured for future planning] A real-PTY end-to-end test harness that drives the interactive TUI through **tmux** (send-keys + capture-pane) so the terminal actually replies to escape queries and actually scrolls — the exact conditions the current piped/non-TTY suite can never reproduce. Motivation: v1.0 Phase 7's human UAT caught two user-visible TUI bugs that BOTH the full piped automated suite AND a deep multi-agent code review missed, because they only manifest on a live TTY — G-07-1 (bare `daemon` on a TTY with an empty registry leaked the terminal's DECRQM capability-probe responses `^[[?2026;2$y^[[?2027;0$y`) and G-07-2 (both bubbletea pickers rendered inline without alt-screen → heavy flicker + blank list). bubbletea Models are unit-testable via synthetic `tea.Msg` (state transitions) but that path never renders. Scope a suite that spawns the release binary inside a tmux pane and asserts on `capture-pane` output: (a) bare `daemon` empty-registry prints ONLY `no running daemons` with no leaked escape sequences; (b) the daemon picker enters the alternate screen, renders `Running daemons` + a seeded record, and restores the main buffer on quit (no residual escapes in scrollback); (c) the install/uninstall checkbox picker renders `[x]`/`[ ]` glyphs, `space` toggles, `q`/`esc` cancels with zero config writes; (d) no flicker proxy (stable capture across N frames). Reuse the `tmux` skill's send-keys/capture-pane idioms; gate the suite behind a build tag / CI job that has tmux available (skip cleanly where it isn't). This is the missing rung between the piped never-hang/byte-identity integration tests (necessary, TTY-blind) and manual human UAT (thorough, unautomated).
**Requirements:** TBD
**Plans:** 0 plans

Plans:

- [ ] TBD (promote with /gsd-review-backlog when ready)

### Phase 999.4: CheckRegression current-metrics positivity guard (BACKLOG)

**Goal:** [Captured for future planning] Close the degenerate-input bypass in `internal/bench.CheckRegression`, surfaced and **reproduced** during the Phase 10 security audit (recorded in `10-SECURITY.md` → "Advisory — Unregistered Surface"; also code-review finding WR-06). Calling `CheckRegression(baseline, current, ceiling=1)` with `current.PeakRSSBytes = 0` and an otherwise-matching frame returns `nil` — **both** the relative RSS regression check and the absolute INDX-06 memory ceiling silently pass. The function already validates that the *baseline* metrics are positive; it never validates the *current* ones, so a zero or negative current reading reads as "no regression" instead of "unusable measurement". This is unreachable through today's only caller because `internal/bench.PeakRSSBytes` returns an error rather than a zero on failure, but `CheckRegression` is exported, its doc comment claims it "never misleads", and the phase-10 audit already showed how easily a frame-descriptor blind spot becomes a live gate failure. Scope: add a positivity/sanity check on `current` mirroring the existing baseline check, refusing rather than passing on a non-positive throughput or RSS reading, with an error naming which field was degenerate. This belongs to the repo's documented class of **gates that cannot fire** (the retracted 10.6% perf claim, the inverted `rg -qv` gate, the 51.5%-stale baseline) — so the fix must be demonstrated RED with a degenerate-input test, not merely added.
**Requirements:** TBD
**Plans:** 0 plans

Plans:

- [ ] TBD (promote with /gsd-review-backlog when ready)

### Phase 999.5: macOS Gatekeeper signing and notarization for darwin release binaries (BACKLOG)

**Goal:** [Captured for future planning] Decide whether to Apple-code-sign and notarize the darwin release binaries, and implement it if so. Surfaced during `/gsd-verify-work 10` while verifying the release pipeline's darwin legs. **The cosign signing this repo already does is a different mechanism entirely and does nothing for Gatekeeper:** cosign keyless (Sigstore) produces a *detached* `.sigstore.json` sidecar verified by `internal/upgrade` in-process, whereas Gatekeeper requires an *embedded* `LC_CODE_SIGNATURE` in the Mach-O, issued under an Apple-anchored identity. **Measured**, not assumed, on binaries built from `.goreleaser.yaml` at HEAD: `codesign -dvv` reports darwin/arm64 as `adhoc, linker-signed` with `TeamIdentifier=not set` (the Go linker emits this automatically because the Apple Silicon kernel refuses to exec a wholly unsigned binary — it satisfies the kernel, not Gatekeeper), darwin/amd64 as `code object is not signed at all`, and `spctl -a -vv -t exec` returns **`rejected`** for both. **Impact is genuinely scoped and should be confirmed before spending money:** Gatekeeper only engages on files carrying the `com.apple.quarantine` xattr, which browsers set and programmatic downloaders do not — a binary fetched by the real `codegraph upgrade` path was verified to carry only `com.apple.provenance`, no quarantine, so `codegraph upgrade` and `curl`-based installs are unaffected today. The affected population is users who download a release asset from the **GitHub Releases page in a browser**, who hit "Apple could not verify this app is free of malware" and must right-click→Open or `xattr -d com.apple.quarantine`. Scope if pursued: (a) paid Apple Developer Program membership plus a *Developer ID Application* certificate, and a decision on where that cert and the App Store Connect API key live as CI secrets; (b) `codesign` with hardened runtime — GoReleaser v2 has a native `notarize:` block for darwin that covers the cert + notary submission flow; (c) resolve the stapling wrinkle: a bare Mach-O **can be notarized but cannot be stapled** (stapling requires a container — `.zip`/`.dmg`/`.pkg`), so bare binaries fall back to an online Gatekeeper check that fails on an offline machine — which likely means shipping a `.zip` or `.dmg` for the browser-download path while keeping the raw per-platform binary the `codegraph upgrade` contract depends on (D-02/Finding 1: assets are raw binaries, never archives, because `internal/upgrade` downloads and swaps the binary directly). Note (c) is the real design decision here, not the signing itself. Per this repo's recurring lesson, whatever lands must be demonstrated against `spctl` actually returning `accepted` on a quarantined download, not merely wired up.
**Requirements:** TBD
**Plans:** 0 plans

Plans:

- [ ] TBD (promote with /gsd-review-backlog when ready)
