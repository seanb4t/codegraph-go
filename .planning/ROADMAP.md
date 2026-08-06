# Roadmap: CodeGraph Go

## Overview

CodeGraph Go is a ground-up Go rewrite of TypeScript CodeGraph — a drop-in, TS-v1.3.x-parity replacement in a single static binary. **v0.1 (Initial Release) shipped 2026-07-14**: the core capabilities (indexing, query, MCP server, sync, migration) work from a signed/attested/SBOM'd release that beats TS 1.3.1 on every measured benchmark — but the CLI/agent surface still diverged *behaviorally* from TS. **v1.0 (Drop-in Parity & Human UX) shipped 2026-08-03**: those gaps are closed. An existing user can now swap binaries with zero change in experience — TS-identical `explore`/`node`/`status` behavior, watcher-on-MCP by default, git/worktree awareness, output hygiene, a human-facing Charm TUI behind a build-enforced rendering seam (the agent/MCP path never sees ANSI), systematic flag reconciliation, fully automated signed releases via release-please + GoReleaser, and contributor-facing local build tooling. Work was risk-front-loaded: the load-bearing shared-engine behavioral algorithms landed first, the human TUI last.

**v0.3.0 (MCP Protocol Currency) is in progress.** MCP published spec revision `2026-07-28` six days after v1.0 shipped, which makes codegraph-go's server a **Legacy** implementation in that spec's own terminology. This milestone brings the agent surface current on the official `modelcontextprotocol/go-sdk` — the maintainer's pre-decision, since `mark3labs/mcp-go` has no `2026-07-28` support and no announced timeline — without breaking any of the 8 agent clients `codegraph install` already configures. Work is ordered around one non-negotiable: the wire-level verification oracle is built and proven against the **current** server before any SDK code moves, because this project already established (v1.0 Phase 4) that an SDK's own client silently skips malformed stdout lines and therefore cannot fail a purity test.

**Versioning note:** "v1.0" is a *planning-milestone* name, never a release version. The shipped artifact line is `v0.2.0`, computed by release-please from Conventional Commits; there is deliberately no `v1.0.0` tag (maintainer directive D-06R, 2026-07-29). From v0.3.0 onward the milestone label tracks the actual release line, but it is still a planning label carrying **no git tag** — release-please remains the sole tag authority and computes the real version from Conventional Commits, so `v0.3.0` is a prediction that holds if this milestone lands `feat:` commits. Milestones carry no git tag; the milestone record lives in `MILESTONES.md` + `milestones/`. (`milestone-v0.1` exists only because it predates release-please.)

## Milestones

- ✅ **v0.1 — Initial Release** — Phases 1–8 (shipped 2026-07-14) — core capabilities + signed release; not yet a drop-in parity replacement
- ✅ **v1.0 — Drop-in Parity & Human UX** — Phases 1–10 (shipped 2026-08-03) — behavioral + surface parity with TS 1.3.1, human TUI, automated signed releases, local build tooling
- 🚧 **v0.3.0 — MCP Protocol Currency** — Phases 1–5 (in progress) — official Go SDK adoption, `2026-07-28` spec compliance without breaking Legacy clients, a wire-level verification oracle, tool-modfile vulnerability coverage
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

### 🚧 v0.3.0 — MCP Protocol Currency (In Progress)

**Milestone Goal:** Bring codegraph-go's stdio MCP server current with spec revision `2026-07-28` on `modelcontextprotocol/go-sdk@v1.7.0` — without breaking any of the 8 agent clients `codegraph install` already configures — and prove it with a harness that asserts on raw wire bytes and never uses the SDK under test as its own oracle. Phase numbering restarts at 1 for this milestone (v0.1 and v1.0 archived), matching this repo's established convention.

**Ordering is load-bearing, not stylistic:**

- The verification oracle (Phase 1) must pass against the **current, pre-migration** server before any SDK change lands (VRFY-04). A harness written after the swap, validated with the new SDK's own client, would describe the new behavior rather than test it — the exact failure mode v1.0 Phase 4 established.
- The SDK question is **already decided** (adopt `modelcontextprotocol/go-sdk@v1.7.0`, maintainer pre-decision 2026-08-03). There is no decision-gate phase and no defer branch; the impact-assessment work that remains genuinely useful — SEP-by-SEP stdio applicability scoping, the dated 8-agent roster audit, the Team Scale read-out — is folded into Phase 1.
- Vulnerability scanning (Phase 4) runs **after** the migration so it audits the final dependency closure, not one that is about to change.
- SPEC-09 (`tools.listChanged` + `subscriptions/listen`) is deliberately **last**. It is a long-lived-stream mechanism materially larger than every other SPEC item, and the research is explicit that it is not needed for correctness once `ttlMs: 0` ships. A correctness-complete server exists at the end of Phase 3, so SPEC-09 slipping cannot block the milestone's core value.

- [x] **Phase 1: Protocol Scoping & the SDK-Independent Wire Oracle** - A raw-stdio regression oracle proven green against today's server, plus the dated scoping the rest of the milestone is measured against (completed 2026-08-05)
- [ ] **Phase 2: SDK Migration — official go-sdk on the existing surface** - `internal/mcp` moves to `modelcontextprotocol/go-sdk@v1.7.0` with semantically unchanged tool output and mark3labs gone from `go.mod`
- [ ] **Phase 3: `2026-07-28` Spec Compliance** - `server/discover`, per-request `_meta` validation, honest cache control and per-call index detection — with every Legacy client still working
- [ ] **Phase 4: Supply-Chain Coverage & Daemon Substrate Fixes** - The ~400-module credentialed CI tooling closure actually gets scanned, and the two known daemon test-seam races stop masking real regressions
- [ ] **Phase 5: Live Tool-Catalog Change Notification** - Opt-in `subscriptions/listen` clients learn about catalog changes as they happen

## Phase Details

### Phase 1: Protocol Scoping & the SDK-Independent Wire Oracle

**Goal**: Before any SDK code moves, the project owns a verification oracle that reads the actual bytes on stdio and can genuinely fail — plus a dated, evidence-backed scoping of what `2026-07-28` obliges a stdio, tools-only server to do. This phase also carries the non-requirement deliverables folded in from backlog 999.6: a SEP-by-SEP applicability table marking each SEP N/A-for-stdio or applicable-with-reason, and the Team Scale strategic read-out recorded as a decision (the stateless protocol core removes the sticky-routing/shared-session-store infrastructure a future central server would otherwise have needed).
**Depends on**: Nothing (first phase of v0.3.0)
**Requirements**: VRFY-01, VRFY-02, VRFY-03, VRFY-04, VRFY-05, SDK-02
**Success Criteria** (what must be TRUE):

  1. The harness runs against the current, unmodified `mark3labs`-backed `serve --mcp` and passes — asserting on raw stdio wire bytes rather than SDK-typed Go objects, and never using the SDK under test as its own oracle (VRFY-01, VRFY-04)
  2. `serve --mcp` writes the negotiated protocol version to stderr on every connection, with no flag or environment variable needed to turn it on — the only available mitigation for a spec-sanctioned silent version mismatch (VRFY-03)
  3. The server's declared protocol version is asserted against a repo-owned literal, and CI fails if any `LATEST_PROTOCOL_VERSION`-style SDK-owned constant reference remains anywhere in the tree, so a dependency bump can never move wire behavior silently (VRFY-02)
  4. `internal/cli/serve.go` bootstraps and serves entirely through the narrow `internal/mcp.Server` seam and imports no MCP SDK package — the one production-code SDK leak, closed while it is still cheap (SDK-02)
  5. A dated record states which protocol revision each of the 8 roster agent clients (Claude Code, Cursor, Codex CLI, opencode, Gemini CLI, Hermes, Antigravity, Kiro) negotiates, measured against the real clients rather than read from their docs (VRFY-05)

**Plans**: 7/7 plans executed

Plans:
**Wave 1**

- [x] 01-01-PLAN.md — Tracer: end-to-end wire oracle spine — repo-owned protocol literal, `mcp.Server` seam, always-on session line, one captured and frozen handshake path
- [x] 01-02-PLAN.md — VRFY-05 proxying capture shim, dated 8-agent negotiation audit, SEP applicability table and Team Scale read-out

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 01-03-PLAN.md — Enforcement guards: repo-owned protocol-version archtest (VRFY-02), no-MCP-SDK-in-`internal/cli` archtest (SDK-02), session-line contract tests
- [x] 01-04-PLAN.md — Oracle coverage bar: all 8 tools, three `tools/list` variants, four error/edge shapes, plus hand-authored spec anchors
- [x] 01-06-PLAN.md — Anti-regeneration CI guard: pull-request-granular cross-change check

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 01-05-PLAN.md — Multi-era Legacy handshake baseline frozen against the pre-migration server

**Wave 4** *(blocked on Wave 3 completion)*

- [x] 01-07-PLAN.md — Non-vacuity: permanent structural guards plus the one-time mutation matrix

### Phase 2: SDK Migration — official go-sdk on the existing surface

**Goal**: `internal/mcp` runs on `modelcontextprotocol/go-sdk@v1.7.0` with the agent-facing surface unchanged — same tools, same behavior, no semantic change on the wire — and `mark3labs/mcp-go` is gone. Five-era protocol negotiation (`2026-07-28` down to `2024-11-05`) arrives inherited from the dependency rather than as code this project writes.
**Depends on**: Phase 1 (the oracle must exist and be proven against the pre-migration server first — VRFY-04)
**Requirements**: SDK-01, SDK-03, SDK-04, SDK-05
**Success Criteria** (what must be TRUE):

  1. Every existing MCP tool is semantically unchanged from the pre-migration server across the wire-oracle corpus, with the server running on `modelcontextprotocol/go-sdk@v1.7.0`; the full transcript diff is read line by line and every changed line has a recorded cause (SDK-01)
  2. Phase 1's harness *code* is unmodified and still runs against the migrated server. Where a transcript legitimately moves, the new bytes land through that review pass with the cause named — never regenerated wholesale, and never relaxed to make the suite pass (SDK-01)
  3. `mark3labs/mcp-go` is absent from `go.mod`, and the resulting dependency closure has been re-audited through the existing `govulncheck` and SBOM paths (SDK-03)
  4. A handler that returns a plain Go `error` produces a known, asserted wire shape — the mark3labs protocol-error vs. official-SDK `IsError: true` tool-result difference is covered by an explicit test, not inferred from the unchanged Go type signature (SDK-04)
  5. Tool input schemas keep their constraint semantics (notably enum constraints that struct-tag reflection drops), or each loss is written down as a deliberate divergence rather than discovered later by a client (SDK-05)

**Plans**: 1/5 plans executed

Plans:
**Wave 1**

- [x] 02-01-PLAN.md — Tracer: the whole stdio path on go-sdk — backend swap behind the `Server` seam, all 8 tools on struct-tag schemas, explicit tools capability, `cacheScope: private`, session line via receiving middleware
- [ ] 02-02-PLAN.md — Unblock the blocking anti-regeneration gate: one self-expiring exemption for the SDK-01 swap diff shape, demonstrated red both ways

**Wave 2** *(blocked on Wave 1 completion)*

- [ ] 02-03-PLAN.md — SDK-04: the error-to-wire mapping asserted rather than inferred, demonstrated RED against a protocol-error mutation
- [ ] 02-04-PLAN.md — SDK-03: mark3labs out of `go.mod` — the last four client sites, both archtest self-tests re-pointed at go-sdk, closure re-audited

**Wave 3** *(blocked on Wave 2 completion)*

- [ ] 02-05-PLAN.md — Re-freeze the 23 transcripts through one reviewed diff pass; SDK-05 schema audit; oracle non-vacuity re-proved on the new backend

### Phase 3: `2026-07-28` Spec Compliance

**Goal**: The server answers the `2026-07-28` wire contract correctly for a stdio, tools-only implementation — discovery, per-request `_meta` validation, result metadata, honest cache control, and per-call index detection — while every client still speaking an older revision continues to work. At the end of this phase the server is correctness-complete for the milestone's core value.
**Depends on**: Phase 2
**Requirements**: SPEC-01, SPEC-02, SPEC-03, SPEC-04, SPEC-05, SPEC-06, SPEC-07, SPEC-08
**Success Criteria** (what must be TRUE):

  1. A client can call `server/discover` and get the server's capabilities without first calling any tool, and the response's `instructions` field carries codegraph usage guidance so an agent gets orientation without spending a tool call (SPEC-01, SPEC-07)
  2. Per-request `_meta` is validated rather than assumed: a malformed or missing required field answers `-32602`, and an unsupported protocol version answers `UnsupportedProtocolVersionError` (`-32022`) instead of failing silently or proceeding on assumptions (SPEC-02)
  3. Every tool result carries `resultType: "complete"` and `io.modelcontextprotocol/serverInfo` in `_meta`, so a client debugging a negotiation problem can see which server version answered (SPEC-03, SPEC-08)
  4. `tools/list` and `server/discover` carry `ttlMs: 0` and `cacheScope: "private"`, and a user who runs `codegraph init` while an MCP server is already running sees the tools appear — `hasIndex` is re-checked per call rather than snapshotted at server construction, so the cache promise is actually true (SPEC-04, SPEC-05)
  5. A client speaking `2025-11-25` or any earlier revision completes a session and calls tools against the upgraded server, asserted by test rather than assumed from the SDK's documentation — the single highest-consequence mistake available in this milestone is dropping Legacy support, which hard-fails every roster client with no client-side fallback (SPEC-06)

**Plans**: TBD

### Phase 4: Supply-Chain Coverage & Daemon Substrate Fixes

**Goal**: The ~400 third-party modules executed as credentialed CI tooling are actually scanned by a job proven able to fail, and the two known daemon test-seam defects stop producing flaky noise that masks real regressions on the substrate this milestone modifies.
**Depends on**: Phase 2 (so the scan audits the post-migration dependency closure, not one about to change). Independent of Phase 3.
**Requirements**: VULN-01, VULN-02, VULN-03, MAINT-01, MAINT-02, MAINT-03
**Success Criteria** (what must be TRUE):

  1. `govulncheck` covers `go.tool.mod` and `go.tool-lint.mod` via `-mode=binary` over binaries built from those manifests, replacing the `task vuln` target that was reproduced live to be a no-op duplicate of the main-module CI scan (VULN-01)
  2. The scanning job has been demonstrated RED against a deliberately known-vulnerable pin before being trusted, and the workflow states its blocking-versus-advisory stance out loud so an advisory job cannot be mistaken for a gate (VULN-02, VULN-03)
  3. The daemon `-race` failure on the `getppid` test seam (issue #13) is fixed, with the race demonstrated before the fix rather than assumed from a green run (MAINT-01)
  4. `TestRunWatchdogCancelsRunOnSimulatedReparent` (issue #17) passes under full-suite load, fixed at the cause rather than by isolating the test away from the load that exposes it (MAINT-02)
  5. `ci.yml` and `release.yml` name the same GoReleaser version (MAINT-03)

**Plans**: TBD

### Phase 5: Live Tool-Catalog Change Notification

**Goal**: A client that opts into `subscriptions/listen` is told when codegraph's tool catalog changes, instead of learning about it only on its next poll. Deliberately sequenced last: this is a long-lived-stream mechanism materially larger than the rest of the SPEC work and explicitly not required for correctness once `ttlMs: 0` shipped in Phase 3, so slipping it cannot block the milestone.
**Depends on**: Phase 3 (a correctness-complete server must already exist)
**Requirements**: SPEC-09
**Success Criteria** (what must be TRUE):

  1. The server advertises `tools.listChanged: true` in its capabilities (SPEC-09)
  2. A client that opts into `subscriptions/listen` receives `notifications/tools/list_changed` when the tool catalog actually changes — for example when `codegraph init` creates an index under an already-running server (SPEC-09)
  3. A client that does not opt in observes no change in session behavior from Phase 3's server (SPEC-09)

**Plans**: TBD

## Progress

**Execution Order:** Phases 1 → 2 → 3 → 4 → 5. Phase 4 depends only on Phase 2 and may run alongside Phase 3.

| Phase | Milestone | Plans Complete | Status | Completed |
| ----- | --------- | -------------- | ------ | --------- |
| 1. Protocol Scoping & the SDK-Independent Wire Oracle | v0.3.0 | 7/7 | Complete    | 2026-08-05 |
| 2. SDK Migration — official go-sdk on the existing surface | v0.3.0 | 1/5 | In Progress|  |
| 3. `2026-07-28` Spec Compliance | v0.3.0 | 0/0 | Not started | - |
| 4. Supply-Chain Coverage & Daemon Substrate Fixes | v0.3.0 | 0/0 | Not started | - |
| 5. Live Tool-Catalog Change Notification | v0.3.0 | 0/0 | Not started | - |
| 999.2. tmux e2e/UAT test harness | Backlog | 0/0 | Not started | - |
| 999.4. CheckRegression positivity guard | Backlog | 0/0 | Not started | - |
| 999.5. macOS Gatekeeper signing/notarization | Backlog | 0/0 | Not started | - |

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
