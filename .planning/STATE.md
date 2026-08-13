---
gsd_state_version: 1.0
milestone: v0.11.0
milestone_name: Standalone Project Identity
status: roadmap-complete
last_updated: "2026-08-13T22:10:00.000Z"
last_activity: 2026-08-13
progress:
  total_phases: 6
  completed_phases: 0
  total_plans: 0
  completed_plans: 0
  percent: 0
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-08-13)

**Core value:** An agent user can uninstall TS CodeGraph, install the Go binary, migrate their indexes, and everything works the same or better — faster, from a single verifiably-built binary. **As of v1.0 this is delivered, not aspirational.**
**Current focus:** v0.11.0 Standalone Project Identity — roadmap created, 6 phases, 25 requirements. Next: `/gsd-plan-phase 1`.

## Current Position

Phase: 1 of 6 — Corpus Selection by Measurement (not started)
Plan: —
Status: Roadmap complete, awaiting phase planning
Progress: 0/6 phases complete (0%)
Last activity: 2026-08-13 — v0.11.0 roadmap created

## Performance Metrics

**Velocity (v0.11.0):**

- Total plans completed: 0
- No execution data yet for this milestone.

**By Phase (v0.11.0):**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 1 | — | — | — |
| 2 | — | — | — |
| 3 | — | — | — |
| 4 | — | — | — |
| 5 | — | — | — |
| 6 | — | — | — |

**Velocity (v0.10.0 — archived, shipped 2026-08-13):** 4 phases, 15 plans, 34 tasks over 2 days.

**Velocity (v0.5.0 — archived, shipped 2026-08-11):** 4 phases, 24 plans, 59 tasks over 3 days.

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 1 | 6 | - | - |
| 02 | 7 | - | - |
| 03 | 5 | - | - |
| 04 | 6 | - | - |
| 05 | 4 | - | - |
| 06 | 4 | - | - |
| 07 | 4 | - | - |
| 08 | 3 | - | - |

**By Phase (v0.3.0 — archived, milestone shipped 2026-08-06):**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01 | 7 | - | - |
| 2 | 5 | - | - |
| 3 | 5 | - | - |
| 4 | 3 | - | - |
| 5 | 1 | - | - |

**By Phase (v1.0 — archived, milestone shipped 2026-08-03):**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 1 | 18 | - | - |
| 2 | 7 | - | - |
| 3 | 5 | - | - |
| 4 | 3 | - | - |
| 5 | 5 | - | - |
| 6 | 3 | - | - |
| 07 | 8 | - | - |
| 8 | 9 | - | - |
| 09 | 8 | - | - |
| 10 | 7 | - | - |

v1.0 totals: 73 plan summaries across 10 phases (v0.1 shipped 58+ plans across 8 phases — see `milestones/v0.1-*`). Note the standing reconciliation: "plans completed" counts SUMMARY files (66) while the old frontmatter `completed_plans` counted PLAN files (65) — Phase 01 carries 17 plans against 18 summaries. Reconcile deliberately rather than by editing one number to match the other.

**Per-plan metrics for v1.0, v0.3.0 and v0.5.0 are archived** with their milestones under `.planning/milestones/` and in each phase's own SUMMARY files; they were trimmed from this file at the v0.11.0 boundary to keep the live state readable. Nothing was deleted from the archives.

*Updated after each plan completion*

## Accumulated Context

### Decisions

Decisions already made for v0.11.0, before any phase executes (full rationale in
PROJECT.md → Current Milestone and ROADMAP.md → "Ordering is load-bearing"):

- **Corpus selection is decided by measurement, and it blocks everything downstream.** FIXT-01 is a Phase-1 spike that indexes candidate MIT/Apache-2.0 repositories and records real per-kind edge counts and per-language file counts before any set is locked. The trap it exists to catch is already in this repo's history: v1.0 Phase 1 established that codegraph-go's own idiomatic Go source produces **zero** `overrides` and `type_of` edges, so a corpus set can silently under-cover the 9-kind `RANK_EDGES` vocabulary while the whole suite stays green. Shortlist to measure (not locked): gohugoio/hugo, nestjs/nest, google/guava, apache/arrow.
- **Fetch at pinned SHA with CI cache; never vendor corpus source.** Avoids repo bloat and avoids adding redistribution obligations to the very `NOTICE` file this milestone is trimming.
- **The rename pass (CODE-02) and the re-freeze pass (FIXT-06) are separate reviewed diffs.** One diff containing both makes any regression un-attributable. The repo's discipline is one reviewed-diff pass with every changed transcript attributable to a single named cause (v0.3.0 Phase 3; v0.10.0 Phase 8). The identifier pass changes no golden byte; the re-freeze changes no identifier.
- **FIXT-07 runs after FIXT-06 and in a different phase, never in the same plan.** A gate is not trusted until demonstrated RED against a confirmed-applied, byte-cleanly-reverted mutation. A re-baseline that authors its own proof in the same change certifies its own oracle.
- **FIXT-03's "no self-skips" needs a positive assertion.** A negative-only guard passes vacuously — this repo already carries that class twice (rule `84d1gfpywd`; the `dry-run-signed` additions-only diff guard). The CI job must report and assert an executed-scenario count, on the `ExpectedScenarioCount` precedent from the wire oracle.
- **The sweep removes framing, never capability.** `codegraph migrate` keeps working end-to-end and is reframed as a legacy-index import. `internal/indexer/tsextract`, the language registry and the capability matrix are product surface — TypeScript-the-indexed-language, not TypeScript-the-origin-project. Resolved term-by-term with recorded reasons, never by regex.
- **No vocabulary drift guard (VOCAB-01), deliberately.** A term blocklist either goes vacuous or fights legitimate uses like `tsextract`. One-time sweep plus review discipline is the chosen posture, recorded in REQUIREMENTS.md → v2 so the decision stays visible.
- **`docs/CLI-REFERENCE.md` (DOCS-05) is deferred.** This milestone *deletes* `docs/FLAG-PARITY.md` and its drift guard; authoring a self-authored replacement with its own guard is separate work. This is a knowing, recorded reduction in flag-documentation coverage.
- **The engram memory sweep is last and is verified by inspection.** The spine lives outside git; no CI gate can hold it. MEM-01 supersedes rather than overwrites — nothing recording real history is deleted, only present-tense assertions of the retired framing are corrected.
- **`.planning/` archives and `CHANGELOG.md` are out of scope.** The first is an append-only record parsed by scope-sensitive tooling; the second is release-please-owned and hand-editing it breaks the tool that both writes and re-reads it.
- **`v0.11.0` is a prediction, not a tag.** release-please is the sole tag authority (D-06R); no phase schedules a `git tag` step.

Decisions from prior milestones are archived with them — per-phase decisions live in
`milestones/*-phases/*/`, and the durable product-level ones are summarized in
PROJECT.md → Key Decisions.

Standing decisions that outlive every milestone:

- Versions follow release-please + Conventional Commits; no version is ever forced (D-06R, maintainer directive 2026-07-29). There is deliberately no `v1.0.0` tag.
- **release-please is the sole tag authority** (D-06R). No hand-created tags of any kind — including milestone markers. `milestone-v0.1` exists only because it predates release-please. If a planning tag is ever reintroduced, it must not match `release.yml`'s `push: tags: "v[0-9]*"` trigger.
- The agent/MCP output path stays plain and parseable; all Charm styling is confined to the human path by a fail-closed ANSI-isolation archtest, not by convention.
- CGo tree-sitter is the single documented CGo exception (DIST-05 / PARSER-DECISION.md).
- `Taskfile.yml` is the single definition of every CI job body; `TestWorkflowRunBodiesInvokeTask` enforces it.
- **A gate is not trusted until it has been demonstrated RED against a confirmed-applied mutation.**
- **A guard must carry a positive assertion that it did its work.** Negative-only guards pass vacuously (rule `84d1gfpywd`).
- **Shared-array-entry ownership must be exact-identity, never shape/position** (hardened after the v0.10.0 hook-ownership vulnerability, commit `242ec0a`).
- **A transcript grep is a claim about the grep, not about the product.** Before recording an absence from a transcript, prove the same search can find the thing when it is present (established twice in the v0.10.0 Phase 6 rehearsal).

### Pending Todos

7 pending — `/gsd-capture --list` to review. All predate v0.11.0 and none block it.

| Created | Area | Severity | Title |
|---------|------|----------|-------|
| 2026-08-07 | mcp | major | Wire oracle `toolslist-repeat` response ordering flake — id-2 response overtaken by id-3 under parallel load on Linux; latent on main, re-run of the identical commit passed |
| 2026-08-09 | release | — | `dry-run-signed` additions-only diff guard passes vacuously |
| 2026-08-09 | ci | — | post-release-verify event-aware conclusion guard has no regression assertion |
| 2026-08-10 | ci | — | Add golangci-lint with gofmt and idiomatic Go linters |
| 2026-08-10 | docs | — | `brew trust` instructions recommend broader tap grant with no security framing |
| 2026-08-10 | ci | — | Tap App secret distinctness test is tautological and reads no workflow |
| — | mcp | major | CR-01 — `internal/mcp/server.go` `pendingWriter` counter corrupted by server-initiated notifications (see Blockers/Concerns) |

Resolved and filed to `.planning/todos/completed/`:

| Resolved | Area | Title |
|----------|------|-------|
| 2026-07-28 | docs | Document release procedures (maintainer runbook) — closed by 09-04's `docs/RELEASE-PROCEDURES.md` rewrite |
| 2026-07-31 | perf | Bisect the indexer throughput regression — **REFUTED**; the regression did not exist (cross-platform baseline comparison) |
| 2026-07-31 | perf | Rebless perf baseline on ubuntu-latest — **DONE**; gate green on main |
| 2026-08-13 | agents | Author a codegraph usage skill for agents — closed by v0.10.0 Phases 6–8 |

### Blockers/Concerns

Nothing blocks v0.11.0. Carried forward from prior milestones:

- **CR-01 (v0.10.0 Phase 5 code review, `internal/mcp/server.go:225-349`): `pendingWriter`'s "pending response" counter is corrupted by server-initiated notifications,** defeating the stdin-EOF-race fix it exists to provide. The counter increments only on accepted client requests but decrements on every stdout `Write()` call, including notifications (`notifications/tools/list_changed`, `notifications/subscriptions/acknowledged`) that SPEC-09 routes through the identical writer. A notification landing between a request's acceptance and its response being written can zero the counter early, causing premature EOF propagation and silent loss of the still-in-flight response — confirmed reachable, not merely theoretical. Predates v0.10.0 (introduced in `13f2875`). Full trace and proposed fix in `.planning/phases/05-mcp-resources-capability-claims-drift-guard/05-REVIEW.md`. Needs its own tracking issue/plan.
- **Backlog bookkeeping inconsistency (needs a maintainer call).** `999.3` and `999.6` were both promoted into v0.3.0, but all five `999.x` Backlog entries were preserved verbatim in `ROADMAP.md` by explicit instruction. Decide whether the promoted entries should be struck or annotated; nothing was removed pending that call. (`999.5` has since been consumed by v0.5.0 and is no longer in the Backlog; `999.2` and `999.4` remain.)
- **Client-side `tools/list` caching bugs are a known confound.** Real, primary-source GitHub issues exist against Claude Code itself (anthropics/claude-code #41123, #40025, #50515; claude-ai-mcp #45).
- **Open GitHub issues:** #14 provenance-over-checksums wording still uncorrected in `release.yml` and two docs · #15 `PRFILES_EOF` heredoc over fork-controlled paths in two `pull_request_target` workflows · #16 `CheckRegression` still never compares `Metrics.Repo` (corpus identity — note this touches BENCH-02's surface).
- **Advisory, unregistered surfaces** from the v1.0 Phase 10 security audit: the four `pull_request_target` workflows and the darwin canary have no threat-register entry, having landed after their registers were authored.
- **`GOOS=windows go vet`** on `internal/daemon` / `internal/graphstore` fails (`undefined: tree_sitter.Node` in `goextract/routes`) — CGo grammar bindings excluded under windows build constraints; pre-existing. Native Windows support was dropped in `v0.4.0` (WSL2 only).
- **GO-2026-5932 is a real, ACCEPTED, unmitigated exposure in release tooling.** goreleaser's binary reaches `golang.org/x/crypto/openpgp` (110 vulnerable symbols) via pipe/ko → google/ko → sigstore/cosign/oci → sigstore/rekor/pkg/pki/pgp. Upstream is unmaintained (Fixed in: N/A) and the ko pipe compiles into every goreleaser binary regardless of config. The advisory `tool-vuln` job surfaces it — reported, not resolved.
- **Daemon extreme-load tail (ACCEPTED, not a gap).** 52/52 real `ci.yml` runs show no daemon failure on the actual runner class; CI load was ruled the governing standard for MAINT-02 (maintainer, 2026-08-06). Not scheduled for further work. The associated feedback-latency tradeoff (daemon package ~65s clean at GOMAXPROCS=4, failures take ~250s to surface) is accepted at CI concurrency.
- **Wire-oracle `toolslist-repeat` ordering flake.** `TestFrozenTranscriptsMatch/toolslist-repeat` freezes JSON-RPC response *arrival* order, which the protocol does not guarantee and go-sdk's async dispatch does not provide.
- **Tooling gaps (not blocking work, and not hand-edited per the planning-artifacts rule):** `gsd-tools query state.advance-plan` failed with "Cannot parse Current Plan or Total Plans in Phase from STATE.md" when Current Position read "Plan: Not started". `gsd-tools query state.sync` counts a SUMMARY with `status: halted` as a completed plan, and MUTATES when invoked with no args — it has no dry-run probe mode.

### Quick Tasks Completed

| # | Description | Date | Commit | Directory |
|---|-------------|------|--------|-----------|
| 260807-gho | Drop native Windows support — WSL2 only | 2026-08-07 | 085b7a3 | [260807-gho-drop-native-windows-support-wsl2-only](./quick/260807-gho-drop-native-windows-support-wsl2-only/) |
| 260811-s5o | Install cosign in post-release-verify's self-upgrade job (v0.9.0 self-upgrade proof failed closed on a missing installer) | 2026-08-11 | 6135785 | [260811-s5o-add-sha-pinned-sigstore-cosign-installer](./quick/260811-s5o-add-sha-pinned-sigstore-cosign-installer/) |

## Deferred Items

Deferred at v0.11.0 scoping (2026-08-13) — recorded so the decisions stay visible:

| Category | Item | Status |
|----------|------|--------|
| requirement | DOCS-05 — self-authored `docs/CLI-REFERENCE.md` with its own drift guard | v2; this milestone deletes `docs/FLAG-PARITY.md`, a later one authors the replacement |
| requirement | VOCAB-01 — build-time vocabulary drift guard | v2, deliberately **declined** for this milestone (blocklist goes vacuous or fights `tsextract`) |

Carried forward, acknowledged and deferred at v0.10.0 close on 2026-08-13:

| Category | Item | Status |
|----------|------|--------|
| todo | 2026-08-07 — wire-oracle toolsList repeat-response ordering flake | pending [mcp] |
| todo | 2026-08-09 — dry-run-signed additions-only diff guard passes vacuously | pending [release] |
| todo | 2026-08-09 — post-release-verify event-aware conclusion guard has no regression assertion | pending [ci] |
| todo | 2026-08-10 — add golangci-lint with gofmt and idiomatic Go linters | pending [ci] |
| todo | 2026-08-10 — brew trust instructions recommend broader tap grant with no security framing | pending [docs] |
| todo | 2026-08-10 — tap App secret distinctness test is tautological and reads no workflow | pending [ci] |
| requirement | GUARD-HOOK-01/02 — PreToolUse guard hook | v2; the fallback if skill+resources+nudge prove insufficient, and that evidence does not exist yet |
| requirement | AGENT-04…07 — multi-agent skill/hooks porting | v2; blocked on per-agent hook-schema differences across Cursor / Codex CLI / Antigravity |
| seed | SEED-001 — local Svelte + shadcn-svelte UI for browsing/querying the graph | dormant |
| seed | SEED-003 — markdown in the index | dormant |
| backlog | 999.2 — tmux e2e/UAT test harness for the interactive TUI | in ROADMAP Backlog |
| backlog | 999.4 — CheckRegression current-metrics positivity guard | in ROADMAP Backlog |
| requirement | MRTR-01 — mid-call elicitation via `resultType: "input_required"` | deferred as new product behavior, not protocol currency |
| requirement | TASK-01 — long-running operations via `io.modelcontextprotocol/tasks` | not applicable today; codegraph's MCP tools are fast read-only queries |
| milestone | Team Scale (central server, CI-distributed indexes, concurrent access) | unscoped |
| deferral | DIST-06 — stapled offline-safe container | v0.5.x deferral; stapling is categorically impossible for bare Mach-O and `.zip` |
| deferral | BREW-07 — homebrew-core submission | v0.5.x deferral |

Closed and no longer deferred:

| Category | Item | Landed as |
|----------|------|-----------|
| seed | SEED-002 — homebrew installation path | consumed by v0.5.0 Phases 1–4; tap published, cask installable |
| backlog | 999.5 — macOS Gatekeeper signing and notarization | promoted into v0.5.0 Phase 2 (SIGN-01…04) |
| backlog | 999.6 — MCP `2026-07-28` impact assessment | the spine of v0.3.0 |
| backlog | 999.3 — vulnerability scanning for the tool modfiles | VULN-01/02/03, v0.3.0 Phase 4 |
| todo | 2026-08-08 — author a codegraph usage skill for agents | closed by v0.10.0 Phases 6–8 |
| Release | DIST-02 — real signed `v*` tag | ✓ `v0.2.0` shipped via release-please (REL-02) |
| Perf | PERF-01 — published head-to-head numbers | ✓ Closed in v1.0 Phase 8 (REL-03) — **note: BENCH-01 retires those numbers this milestone** |

Not deferred but worth carrying: `.planning/debug/perf-gate-throughput-regress.md` is backed by
fresh data on issue #20 — the gate failed then passed on an inert diff at ~6.9% intra-run spread
against a 10% budget. Relevant to BENCH-02.

## Session Continuity

Last session: 2026-08-13T22:10:00.000Z
Stopped at: v0.11.0 roadmap created — 6 phases, 25/25 requirements mapped
  NEXT: `/gsd-plan-phase 1` (Corpus Selection by Measurement — blocking spike)
  CARRY-OVER:

    - **Phase 1 is blocking.** No phase that re-freezes a golden may be planned until FIXT-01 locks the corpus set by recorded measurement. Phase 4 (Attribution & Documentation Sweep) has no dependency on Phases 1–3 and may be planned and run in parallel.
    - **PR #60 (v0.10.0) must be on `main` before this milestone's branch is cut** — verify it merged before starting. This repo is squash-merge-only with `squash_merge_commit_title: PR_TITLE`, so the PR title is the release decision.
    - **`branching_strategy: milestone`** — this milestone lives on one branch (`gsd/v0.11.0-standalone-project-identity`) and is not incrementally merged.
    - **No `v0.11.0` git tag.** release-please owns tagging (D-06R); a hand-created tag would match `release.yml`'s `v[0-9]*` trigger and falsely fire the release pipeline.
    - **`.planning/` and `CHANGELOG.md` are out of scope for the sweep** — the first would falsify project history and break GSD's scope-sensitive parsers, the second is release-please-owned.

Resume file: none

## Operator Next Steps

- Confirm PR #60 is merged to `main` (lands v0.10.0)
- `/gsd-plan-phase 1` — the blocking corpus-selection spike
