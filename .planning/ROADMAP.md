# Roadmap: CodeGraph Go

## Overview

CodeGraph Go is a ground-up Go rewrite of TypeScript CodeGraph — a drop-in, TS-v1.3.x-parity replacement in a single static binary. **v0.1 (Initial Release) shipped 2026-07-14**: the core capabilities (indexing, query, MCP server, sync, migration) work from a signed/attested/SBOM'd release that beats TS 1.3.1 on every measured benchmark — but the CLI/agent surface still diverged *behaviorally* from TS. **v1.0 (Drop-in Parity & Human UX) shipped 2026-08-03**: those gaps are closed. An existing user can now swap binaries with zero change in experience — TS-identical `explore`/`node`/`status` behavior, watcher-on-MCP by default, git/worktree awareness, output hygiene, a human-facing Charm TUI behind a build-enforced rendering seam (the agent/MCP path never sees ANSI), systematic flag reconciliation, fully automated signed releases via release-please + GoReleaser, and contributor-facing local build tooling. Work was risk-front-loaded: the load-bearing shared-engine behavioral algorithms landed first, the human TUI last.

**Versioning note:** "v1.0" is a *planning-milestone* name, never a release version. The shipped artifact line is `v0.2.0`, computed by release-please from Conventional Commits; there is deliberately no `v1.0.0` tag (maintainer directive D-06R, 2026-07-29). Milestones carry **no git tag** — release-please is the sole tag authority since Phase 9, and the milestone record lives in `MILESTONES.md` + `milestones/`. (`milestone-v0.1` exists only because it predates release-please.)

## Milestones

- ✅ **v0.1 — Initial Release** — Phases 1–8 (shipped 2026-07-14) — core capabilities + signed release; not yet a drop-in parity replacement
- ✅ **v1.0 — Drop-in Parity & Human UX** — Phases 1–10 (shipped 2026-08-03) — behavioral + surface parity with TS 1.3.1, human TUI, automated signed releases, local build tooling
- 📋 **Next** — unscoped; run `/gsd-new-milestone`. Candidates: the five Backlog items below, Team Scale (central server, CI-distributed indexes), annotations (embeddings/communities/export), local Svelte web UI (SEED-001), homebrew install path (SEED-002)

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

## Progress

| Phase | Milestone | Plans Complete | Status | Completed |
| ----- | --------- | -------------- | ------ | --------- |
| 1. Behavioral Parity — explore & node | v1.0 | 17/17 | Complete | 2026-07-15 |
| 2. status Content & Git/Worktree Awareness | v1.0 | 7/7 | Complete | 2026-07-16 |
| 3. Watcher-on-MCP Default | v1.0 | 5/5 | Complete | 2026-07-16 |
| 4. Output Hygiene | v1.0 | 3/3 | Complete | 2026-07-16 |
| 5. Git Sync Hooks | v1.0 | 5/5 | Complete | 2026-07-17 |
| 6. Rendering Seam & Pretty status/files | v1.0 | 3/3 | Complete | 2026-07-17 |
| 7. Interactive TUI — Daemon Picker & Install Multi-Select | v1.0 | 8/8 | Complete | 2026-07-26 |
| 8. Surface Reconciliation & Signed v1.0.0 Release | v1.0 | 9/9 | Complete | 2026-07-28 |
| 9. release-please + GoReleaser | v1.0 | 8/8 | Complete | 2026-08-01 |
| 10. Local Build Tooling & CONTRIBUTING | v1.0 | 7/7 | Complete | 2026-08-03 |
| 999.2. tmux e2e/UAT test harness | Backlog | 0/0 | Not started | - |
| 999.3. Vulnerability scanning for tool modfiles | Backlog | 0/0 | Not started | - |
| 999.4. CheckRegression positivity guard | Backlog | 0/0 | Not started | - |
| 999.5. macOS Gatekeeper signing/notarization | Backlog | 0/0 | Not started | - |
| 999.6. MCP 2026-07-28 spec impact assessment | Backlog | 0/0 | Not started | - |

## Backlog

### Phase 999.2: tmux e2e/UAT test harness and suite (BACKLOG)

**Goal:** [Captured for future planning] A real-PTY end-to-end test harness that drives the interactive TUI through **tmux** (send-keys + capture-pane) so the terminal actually replies to escape queries and actually scrolls — the exact conditions the current piped/non-TTY suite can never reproduce. Motivation: v1.0 Phase 7's human UAT caught two user-visible TUI bugs that BOTH the full piped automated suite AND a deep multi-agent code review missed, because they only manifest on a live TTY — G-07-1 (bare `daemon` on a TTY with an empty registry leaked the terminal's DECRQM capability-probe responses `^[[?2026;2$y^[[?2027;0$y`) and G-07-2 (both bubbletea pickers rendered inline without alt-screen → heavy flicker + blank list). bubbletea Models are unit-testable via synthetic `tea.Msg` (state transitions) but that path never renders. Scope a suite that spawns the release binary inside a tmux pane and asserts on `capture-pane` output: (a) bare `daemon` empty-registry prints ONLY `no running daemons` with no leaked escape sequences; (b) the daemon picker enters the alternate screen, renders `Running daemons` + a seeded record, and restores the main buffer on quit (no residual escapes in scrollback); (c) the install/uninstall checkbox picker renders `[x]`/`[ ]` glyphs, `space` toggles, `q`/`esc` cancels with zero config writes; (d) no flicker proxy (stable capture across N frames). Reuse the `tmux` skill's send-keys/capture-pane idioms; gate the suite behind a build tag / CI job that has tmux available (skip cleanly where it isn't). This is the missing rung between the piped never-hang/byte-identity integration tests (necessary, TTY-blind) and manual human UAT (thorough, unautomated).
**Requirements:** TBD
**Plans:** 0 plans

Plans:

- [ ] TBD (promote with /gsd-review-backlog when ready)

### Phase 999.6: MCP 2026-07-28 spec — impact assessment and migration decision (BACKLOG)

**Goal:** [Captured for future planning] Assess what the MCP `2026-07-28` specification release (published 2026-07-28, announced at <https://blog.modelcontextprotocol.io/posts/2026-07-28/>) changes for this project's agent-facing surface, and decide the migration stance. This landed **during v1.0 Phase 9/10** and was not visible to any v1.0 planning artifact — the whole parity story is built against the prior spec revision.

**What the release actually changes.** The headline is a **stateless protocol core**: the `initialize`/`initialized` handshake and the `Mcp-Session-Id` header are retired (SEP-2575, SEP-2567); every request now self-describes, carrying protocol version, client identity, and client capabilities in `_meta`, with an optional `server/discover` RPC for clients that want capabilities up front. Alongside that: **MRTR** (SEP-2322) replaces the server-initiated `elicitation/create` / `sampling/createMessage` / `roots/list` requests that needed a held-open stream, using a `resultType: "input_required"` + client-retry-with-`inputResponses` shape; **header-based routing** makes `Mcp-Method` / `Mcp-Name` mandatory on Streamable HTTP (SEP-2243); **list results become cacheable** with `ttlMs` / `cacheScope` on `tools/list`, `prompts/list`, `resources/list`, `resources/read` (SEP-2549); authorization hardens (RFC 9207 `iss` validation, DCR formally deprecated in favor of CIMD); Tasks moves out of the experimental core into an `io.modelcontextprotocol/tasks` extension under a newly formalized extensions framework. **Deprecated with a 12-month minimum window:** Roots, Sampling, Logging (SEP-2577) and the legacy HTTP+SSE transport. A formal deprecation policy now guarantees that window.

**Why the blast radius is probably smaller here than the headline suggests — but is NOT zero.** Grounded in the code at HEAD, not assumed:

- We are **stdio-only**. `serve --mcp` is the sole transport; there is no Streamable HTTP surface. So header-based routing, load-balancer fan-out, the stateless-scaling motivation, and the HTTP+SSE deprecation are all **inapplicable to us today** — they are the parts of the release the ecosystem quotes most, and the parts that matter to us least.
- We declare **tools capability only** — `server.NewMCPServer("codegraph", version, server.WithToolCapabilities(true))` at `internal/mcp/server.go:95`. No roots, no sampling, no logging, no prompts, no resources. **The entire deprecation list misses us**, which is a genuinely lucky position and should be confirmed rather than assumed at planning time.
- The parts that **do** reach a stdio tools-only server: (a) retirement of the `initialize`/`initialized` handshake, which is exactly what our stdio session establishment is; (b) `_meta`-carried client identity/capabilities replacing what the handshake used to negotiate; (c) `server/discover`; (d) `ttlMs`/`cacheScope` on `tools/list` — relevant because our tool catalog is *already* dynamic (zero tools without `.codegraph/`, `CODEGRAPH_MCP_TOOLS` allowlist gating), so a cacheable-list contract interacts directly with `tools_schema_drift_test.go` and with the reconnect reconcile path (`internal/mcp/reconnect_test.go`).

**The dependency question this forces — and it is the real decision.** We are on `github.com/mark3labs/mcp-go v0.56.0` (`go.mod:13`). The blog states all four Tier-1 SDKs — including **`modelcontextprotocol/go-sdk`** — speak `2026-07-28` as of release day. `.claude/CLAUDE.md`'s "Alternatives Considered" table already commits us to *"Re-evaluate the official SDK at each subsequent milestone boundary; if the official SDK's stdio transport and tool-registration ergonomics have matured and its adoption has grown, migrate. Don't block v1 parity on this decision."* **This is that boundary, and the official SDK now has a spec-currency argument that did not exist when the choice was made.** Scope must therefore cover: does `mark3labs/mcp-go` ship `2026-07-28` support, on what timeline, and does it stay the right call once the official SDK is the one tracking the spec? Note the swap was assessed as a *bounded refactor, not a rewrite* — re-verify that against the two handler signatures in `internal/mcp/tools.go` and `BuildServer` rather than trusting the prior estimate.

**Compatibility is the load-bearing risk, and it points at the 8-agent roster.** `codegraph install` configures **8** agent clients (Claude Code, Cursor, Codex CLI, opencode, Gemini CLI, Hermes, Antigravity, Kiro). Those clients will move to `2026-07-28` on their own schedules. The failure mode to design against is **an agent client that has moved forward refusing or mis-negotiating our older-spec server** — which, per this repo's documented history, would surface as a *quiet* failure: tools silently not advertised, not a red check anywhere. `internal/mcp/server_test.go` pins `mcp.LATEST_PROTOCOL_VERSION` from the SDK rather than a literal, so our declared version silently tracks whatever the SDK ships — convenient, but it means a dependency bump can move our wire behavior with no test asserting the change was intended.

**Scope if pursued:** (a) determine empirically which spec revision each of the 8 roster clients negotiates, and whether any already refuse the old handshake — measure against real clients, do not infer from their docs; (b) decide `mark3labs/mcp-go` vs `modelcontextprotocol/go-sdk` on spec currency + stdio ergonomics + adoption, per the standing CLAUDE.md re-evaluation commitment; (c) decide whether we adopt `2026-07-28` at all in this cycle or explicitly defer behind the 12-month deprecation window — **an explicit, dated defer is an acceptable outcome and should be recorded as a decision, not left implicit**; (d) if we adopt, replace the `LATEST_PROTOCOL_VERSION` pin with an explicit asserted version so a wire-behavior change can never ride in silently on a dependency bump; (e) re-run the live end-to-end MCP handshake verification (install → `codegraph_explore` advertised over stdio → uninstall restores config) that Phase 6 established, since that is the only check that has ever exercised the real negotiation path. Per this repo's recurring lesson, whatever lands must be demonstrated RED against a confirmed-applied mutation — a green handshake test that would also pass against a broken negotiation proves nothing.

**Requirements:** TBD
**Plans:** 0 plans

Plans:

- [ ] TBD (promote with /gsd-review-backlog when ready)

### Phase 999.3: Vulnerability scanning for the tool modfiles (BACKLOG)

**Goal:** [Captured for future planning] Close the supply-chain gap surfaced by the Phase 10 security audit (recorded in `10-SECURITY.md` → "Advisory — Unregistered Surface"; also code-review finding WR-07). Phase 10 introduced `go.tool.mod` and `go.tool-lint.mod` — roughly **400 modules** including goreleaser plus the AWS/GCP/Azure SDKs, k8s client libraries, cosign and sigstore, plus actionlint's dependency tree. These are built from source and **executed as credentialed CI tooling**: `goreleaser` signs and publishes releases, and `task` drives every CI job body. But `ci.yml:156-171`'s blocking `govulncheck` job scans the **root `go.mod` only**. `Taskfile.yml`'s `vuln` target is the only thing that ever points govulncheck at `go.tool.mod`, is documented as local-only (`go.tool.mod:10-15`), and is invoked by **no** CI job; `go.tool-lint.mod` has no vulnerability-scanning path at all. Threat T-10-01-01 covers *how* these modules are fetched (checksummed module proxy, no `curl | sh`), but nothing covers a **known-vulnerable dependency being executed** inside the credentialed release pipeline. Scope: (a) mint a registered threat for this trust boundary rather than leaving it advisory; (b) add a CI scanning path that covers both tool modfiles — note `govulncheck` is call-graph-aware, so scanning a modfile whose only entry points are third-party `main` packages needs its invocation shape thought through (`-mode=binary` over the built tool, or per-tool package targets, rather than a naive `./...`); (c) decide blocking vs advisory — the root-`go.mod` job is blocking, and a tool-modfile job that is merely advisory should say so out loud rather than look like a gate. Guard against this repo's recurring failure mode: whatever lands must be demonstrated RED against a known-vulnerable pin before it is trusted.
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
