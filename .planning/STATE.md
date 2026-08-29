---
gsd_state_version: 1.0
milestone: v0.12.0
milestone_name: Local Graph UI
current_phase: 03
current_phase_name: Browse, Inspect & Navigation
status: executing
stopped_at: Completed 03-06-PLAN.md
last_updated: "2026-08-29T02:37:38.766Z"
last_activity: 2026-08-28
last_activity_desc: Phase 03 execution started
state_head: 3ced35013c6aced86e35d0207d01a64144b55320
progress:
  total_phases: 6
  completed_phases: 2
  total_plans: 28
  completed_plans: 24
  percent: 33
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-08-22)

**Core value:** CodeGraph Go gives coding agents a pre-indexed code knowledge graph — fast symbol/call-path/impact queries served from a single static, verifiably-built binary, with no bundled runtime to install or manage.
**Current focus:** Phase 03 — Browse, Inspect & Navigation

## Current Position

Phase: 03 (Browse, Inspect & Navigation) — EXECUTING
Plan: 7 of 10
Status: Ready to execute
Last activity: 2026-08-28 — Phase 03 execution started

Progress: [███░░░░░░░] 33% (1/6 phases)

## Performance Metrics

**Velocity (v0.12.0):**

- Phase 1: 11 plans across 7 waves, executed and verified 2026-08-23 (1 day). Per-task timings were not recorded in this phase's SUMMARY frontmatter, so no task count is reported rather than an inferred one.

**By Phase (v0.12.0):**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01 | 11 | - | - |
| 02 | 7 | - | - |
| 3 | TBD | - | - |
| 4 | TBD | - | - |
| 5 | TBD | - | - |
| 6 | TBD | - | - |

**Velocity (v0.11.0 — archived, shipped 2026-08-16):** 6 phases, 30 plans, 60 tasks over 4 days.

**Velocity (v0.10.0 — archived, shipped 2026-08-13):** 4 phases, 15 plans, 34 tasks over 2 days.

**Velocity (v0.5.0 — archived, shipped 2026-08-11):** 4 phases, 24 plans, 59 tasks over 3 days.

**Velocity (v0.3.0 — archived, shipped 2026-08-06):** 5 phases, 21 plans over 4 days.

**Velocity (v1.0 — archived, shipped 2026-08-03):** 10 phases, 72 plans, 162 tasks. v0.1 shipped 58+ plans across 8 phases.

Note the standing reconciliation carried from v1.0: "plans completed" counts SUMMARY files (66) while the old frontmatter `completed_plans` counted PLAN files (65) — Phase 01 carries 17 plans against 18 summaries. Reconcile deliberately rather than by editing one number to match the other.

**Per-plan metrics for v1.0, v0.3.0, v0.5.0, v0.10.0 and v0.11.0 are archived** with their milestones under `.planning/milestones/` and in each phase's own SUMMARY files. Nothing was deleted from the archives.

*Updated after each plan completion*
**Per-Plan Metrics:**

| Plan | Duration | Tasks | Files |
|------|----------|-------|-------|
| Phase 02 P01 | 35min | 2 tasks | 23 files |
| Phase 02 P02 | 40min | 2 tasks | 2 files |
| Phase 02 P03 | ~45min | 3 tasks | 6 files |
| Phase 02 P04 | 20min | 3 tasks | 1 files |
| Phase 02 P05 | ~55min | 3 tasks | 13 files |
| Phase 02 P06 | ~1h5min | 3 tasks | 4 files |
| Phase 02 P07 | 22min | 3 tasks | 15 files |
| Phase 03 P01 | ~5min active work | 3 tasks | 8 files |
| Phase 03-browse-inspect-navigation P02 | 25min | 2 tasks | 1 files |
| Phase 03 P03 | ~90min | 3 tasks | 7 files |
| Phase 03 P04 | ~35min | 3 tasks | 12 files |
| Phase 03 P05 | ~40min | 3 tasks | 10 files |
| Phase 03 P06 | ~50min active | 3 tasks | 44 files |

## Accumulated Context

### Decisions

Decisions already made for v0.12.0, before any phase executes (full rationale in
PROJECT.md → Current Milestone and ROADMAP.md → "Ordering is load-bearing"):

- **ConnectRPC over Protobuf is the wire protocol** (maintainer directive, 2026-08-22). Chosen over a hand-rolled JSON/REST surface: typed end-to-end, server-streaming works in browsers over plain HTTP/1.1 so it carries live push natively, and `connect-go` is pure Go layered over `net/http` rather than gRPC's transport stack — a much smaller supply-chain surface than `grpc-go`.
- **`pnpm` is the package manager, not `npm`** (maintainer directive, 2026-08-22). `pnpm-lock.yaml` is the lockfile, `pnpm install --frozen-lockfile` is the CI-correct install, and `pnpm audit` is the JS vulnerability gate. pnpm's strict non-hoisted `node_modules` and its default blocking of dependency lifecycle scripts are both live ways a developer's build can differ from CI's — which is exactly what a *committed* `dist/` cannot tolerate.
- **The SPA is committed and `go:embed`'d; the signed release path stays pure Go.** No JS toolchain inside the reproducible build. `//go:embed all:dist` — the `all:` prefix is mandatory, since Go's default dotfile/underscore exclusion silently drops Vite's `_app`-prefixed chunks while the build still succeeds.
- **The UI is a third consumer of `internal/query.Engine`, never a second implementation.** Eight of the Engine's ten read methods already return structured results; `ENG-01`/`ENG-02` extract the remaining two (`Node`, `Explore`) by exposing the fetch pipeline that already exists and stopping *before* the render call. `Node()`/`Explore()` stay byte-identical and the frozen goldens never see a diff.
- **No IPC-to-daemon architecture, and no long-lived store handle.** `pebble.Open` takes an exclusive directory lock regardless of `ReadOnly` (cockroachdb/pebble#1583), but `graphstore.Open` already wraps it in a bounded retry (5 × 100ms) classifying lock-held failures as the exported `ErrStoreLocked`. `codegraph ui` opens-snapshots-closes per RPC exactly as `internal/mcp`'s `openEngine` does (`SRV-04`). PITFALLS.md's "needs an explicit IPC-to-daemon decision" claim was investigated and is **wrong**; SUMMARY.md carries the correction.
- **Two distinct findings are both named CR-01 — never conflate them.** This milestone's `FIX-01` is `internal/mcp/server.go`'s `pendingWriter` counter corruption (**open**). The CR-01 cited throughout `internal/graphstore/pebble_store.go` is a v0.5.0 Phase-3 finding about the Pebble lock — **already fixed**; the retry loop *is* that fix.
- **`FIX-01` lands early (Phase 1), deliberately out of dependency order.** Its root cause — server-initiated writes decrementing a counter only client-initiated requests increment — is exactly the shape the new ConnectRPC streaming surface can reinvent, so fixing it first makes it a worked example Phase 6's design review checks against.
- **Origin/Host validation lands in the transport phase, before any RPC handler ships.** Retrofitting a rebinding defense after handlers exist is how CVE-2024-28224 (Ollama) and CVE-2025-66414/66416 (MCP SDKs) happened. Exact-match allowlist only — no `Contains`/`HasPrefix`/wildcard regex — with `localhost`, `127.0.0.1` and `[::1]` each admitted explicitly rather than treated as interchangeable.
- **Protobuf is already in-tree; this adds a SECOND proto surface, not the first.** `internal/schema/graph.proto` → `graph.pb.go` is the on-disk record format under an additive-only evolution discipline (D-02a: field numbers never renumbered or reused, retired fields `reserved`). The new UI schema (`RPC-01`) inherits that discipline rather than reinventing it, in a separate package with its own cadence.
- **`BLD-04` closes a pre-existing gap, not just a new one.** There is no proto codegen drift guard in this repo today and no regeneration task in `Taskfile.yml` — `roundtrip_test.go` tests serialization round-tripping, not codegen currency. The guard must cover both the new UI schema and the pre-existing `internal/schema/graph.proto`.
- **`ENG-04` is schema work and gates `BRW-09`.** `schema.Meta` records no commit SHA today. A permalink that matches what the UI just showed must point at the *indexed* commit, so `Meta` gains one additive field following the `HasFileIndex` precedent (absent ⇒ pre-upgrade graph, degrade gracefully). Sequenced with the proto work in Phase 1, not with UI work.
- **Both drift guards ship in the same phase as the artifact they guard** (`BLD-04` in Phase 1 with the new `.proto`; `BLD-03` in Phase 2 with the committed `dist/`), per the v0.10.0 pattern — never deferred to a cleanup phase at the end.
- **`GRF-01` is a blocking measurement spike with its pass condition locked before dispatch,** and its result — not a preference — selects between Cytoscape.js and Sigma.js + graphology. The roadmap deliberately pre-commits to neither renderer. Precedent: v0.5.0's zig-cross spike.
- **The graph view is never force-directed.** File/package aggregation with directory-structural grouping and a hierarchical layout is the v1 design, decided up front rather than discovered as a UX bug. Recovery from shipping the hairball is HIGH cost — layout choice reaches into drag, zoom and click-target code throughout.
- **The rollup is computed on demand, never precomputed.** `Engine.FileGraph()` follows `BuildReverseAdjacency`'s fresh-per-call full-scan discipline — no new record kind, no second staleness mechanism, no re-indexing.
- **Live push comes last, and its integration test runs against real processes.** "Apply this delta to a view" is not designable before the view exists, and the property live push must not violate (never holding the store open) is only observable with a real `codegraph daemon` / `serve --mcp` holding the same store. "Run `codegraph ui` alone" is the dev workflow that hides it.
- **`SEED-001`'s stated trigger is obsolete but the seed is consumed.** It waited on "CLI-surface parity" — a gate v0.11.0 retired outright. The warrant for building now is that the query surface is stable *on its own terms*, which is a stronger reason than the original one.
- **`v0.12.0` is a prediction, not a tag.** release-please is the sole tag authority (D-06R); no phase schedules a `git tag` step.

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
- **`go test -run PATTERN` exits 0 when the pattern matches nothing.** A green exit is never on its own evidence that a test ran; report a count.
- **Shared-array-entry ownership must be exact-identity, never shape/position** (hardened after the v0.10.0 hook-ownership vulnerability, commit `242ec0a`).
- **A transcript grep is a claim about the grep, not about the product.** Before recording an absence from a transcript, prove the same search can find the thing when it is present (established twice in the v0.10.0 Phase 6 rehearsal).
- **Never invent structure in a tool-owned generated file.** `.planning/ROADMAP.md` and `.planning/STATE.md` are parsed by scope-sensitive readers; an invented version-bearing or ✅-bearing `###` heading under `## Phases` silently truncates the active milestone's scope.
- [Phase 2]: No svelte.config.js in this SvelteKit toolchain version (kit ^2.63.0 scaffolded by sv@0.17.0) — adapter/kit config lives in vite.config.ts's sveltekit() plugin options instead
- [Phase 2]: typescript pinned to 6.0.3 (sv@0.17.0's own scaffold default) rather than 5.9.3 — the 6.x line postdates 02-RESEARCH.md and is what the current toolchain itself verified as compatible
- [Phase 02]: [Phase 2] buf v1.72.0's inputs: directory: override cannot scope a subdirectory already covered by a configured buf.yaml module — proto:gen/proto:drift use the --path CLI fallback + a relocate step instead (protoc-gen-es has no paths=source_relative equivalent)
- [Phase 02]: [Phase 02] BLD-07's release-path closure is derived from release.yml's own uses:/run: content (not hardcoded), with an errUnsupportedReachabilityEdge tripwire refusing loudly on edge kinds the model does not cover
- [Phase 02]: [Phase 02] shadcn-svelte@1.5.0 init has no non-interactive CLI path (piped stdin is treated as EOF, writes nothing) — drove it through a real pty (Python's pty module + pyte VT100 rendering) to answer each prompt and record every answer
- [Phase 02]: [Phase 02] shadcn-svelte init auto-added itself as a devDependency and imported its own package-relative shadcn-svelte/tailwind.css from app.css; removed the dependency (T-02-05-06 requires dlx-only) and inlined that CSS file's verbatim boilerplate content (not the token set) in its place
- [Phase 02]: WEB_HASH_LIB Taskfile var shares the source/output hashing pipeline between web:build and web:drift, never duplicated. — Task 2's own instruction: two copies of a hashing pipeline that must agree is a defect waiting to happen.
- [Phase 02]: CODEGRAPH_WEB_BUILD_DIR override lives in web/vite.config.ts, not web/svelte.config.js (which does not exist in this toolchain). — 02-01-SUMMARY.md already recorded that adapter-static config lives in vite.config.ts for this SvelteKit/sv@0.17.0 scaffold.
- [Phase 02]: [Phase 02] allowBuilds: {} written by hand in web/pnpm-workspace.yaml — pnpm approve-builds --all does not write the allowBuilds key when nothing is pending (confirmed empirically) — task web:deps:strict treats total absence as a named failure distinct from a committed empty map, so the empty map is committed by hand once.
- [Phase 02]: [Phase 02] Fixed live GHSA-pxg6-pf52-xh8x (cookie) via pnpm-workspace.yaml overrides — 02-07's empirical pnpm audit run surfaced a real, current advisory in cookie <0.7.0 pulled in by @sveltejs/kit@2.70.3's own ^0.6.0 range (not yet bumped upstream) — overrides cookie@<0.7.0: ^0.7.2 so task web:audit's own clean-tree verify is honestly green.
- [Phase 03]: 03-01: @testing-library/jest-dom newly tripped too-new legitimacy flag at install (published 2026-08-09, after 03-RESEARCH.md); approved under same fallback reasoning as the two tabled [SUS] packages — A recent release on a 2019-vintage, 63M-weekly-download project is not a new or hijacked package
- [Phase 03]: 03-01: added resolve.conditions:['browser'] to web/vite.config.ts, guarded on process.env.VITEST — Default jsdom resolution picked Svelte's server build for .svelte imports (lifecycle_function_unavailable from mount()); scoping to vitest only leaves vite build/dev resolution unaffected
- [Phase 03]: [Phase 03-03]: Wire-oracle toolslist-repeat ordering flake root-caused (SERVER-EMITTED-OUT-OF-ORDER, live Linux repro + modelcontextprotocol/go-sdk@v1.7.0 source citation) and resolved via R2 (CanonicalizeResponseOrder) — maintainer rejected R1 because the SDK is not silent about concurrent dispatch, and rejected NR because an intermittent red required gate teaches "re-run CI"
- [Phase 03]: [Phase 03-04]: highlight.js emits `hljs-*` class names only, no inline colour — added a `highlight.js/styles/github.css` theme import — Caught via mandated manual browser UAT, not any grep: markup was structurally correct but invisible without a loaded theme stylesheet.
- [Phase 03]: [Phase 03-04]: 03-04's Go coverage guard binds HIGHLIGHT_COVERAGE to the indexer registry with a REVERSE binding check beyond the plan's literal three assertions — Without it, a deleted registerLanguage(...) call with a stale array left behind failed only on a bare count, never naming the specific language — verified live before/after adding the check.
- [Phase 03]: [Phase 03]: 03-05: GetPermalink wire shape frozen via human checkpoint — PermalinkAvailability as a closed enum (not open string), and a tri-state RemotePresence{Unknown,Observed,NotObserved} + structured GitHubRemote{Owner,Repo,Host,Reason} so "could not check" never collapses into a false claim of no-link
- [Phase 03]: [Phase 03]: 03-05: since-deleted-file disposition CHOSEN as deleted-classify-in-handler (not the deleted-accept-internal default) — a since-deleted file is a NORMAL outcome for permalinks (they outlive the files they point at), so it is reclassified in internal/uiserver/permalink.go ONLY to CodeInvalidArgument naming the caller's own repo-relative path, never the underlying absolute host path
- [Phase 03]: [Phase 03]: 03-05: TestUIServiceMethodSetIsExactlyTheReadSet's hardcoded method-count literal (9->10) was updated alongside wantUIServiceMethods' new GetPermalink entry — the plan text said "change nothing else" but the literal makes the test permanently unpassable otherwise (Rule 3 deviation)
- [Phase 03]: [Phase 03]: 03-06: vendor gate approved — bits-ui@2.19.0, @internationalized/date@3.12.3 direct plus 10 transitive packages, 36-file shadcn-svelte command surface (registry v1.5.1); textarea/input-group-textarea kept unimported (atomic registry unit, stripping would defeat source-match discipline)
- [Phase 03]: [Phase 03]: 03-06: Command primitive confirmed (spike + live UAT) to traverse Command.Group boundaries in visual order — Task 3 used real grouped sections, no flat-list workaround
- [Phase 03]: [Phase 03]: 03-06: Explore submission wired via a single leading unlabeled Command.Item ("Ask") reusing the primitive's own item-select mechanism, rather than hijacking Enter globally — avoids racing against "Enter opens the highlighted item"
- [Phase 03]: [Phase 03]: 03-06: found live (manual UAT, not assumed) that FilesOptions.Pattern's glob (path/filepath.Match) never crosses '/' and has no recursive '**' — Files' live-search pattern only matches root-level files for nested repos; documented in search.ts, filed as a todo, not fixed (server-side/cross-cutting, out of scope). Search's own file-kind pseudo-node matches already cover arbitrary-depth file discovery
- [Phase 03]: [Phase 03]: 03-06: never name a Svelte 5 $state()-backed local variable literally 'state' — svelte-check reports spurious 'used before declaration'/implicit-any errors; renamed to searchState

### Pending Todos

8 pending — `/gsd-capture --list` to review. One (`CR-01`) is now in scope as this milestone's `FIX-01` (Phase 1); the `bench pinnedAt` item was expected to reconcile in v0.11.0 Phase 6 and should be re-checked. None block v0.12.0.

| Created | Area | Severity | Title |
|---------|------|----------|-------|
| 2026-08-07 | mcp | major | Wire oracle `toolslist-repeat` response ordering flake — id-2 response overtaken by id-3 under parallel load on Linux; latent on main, re-run of the identical commit passed |
| 2026-08-09 | release | — | `dry-run-signed` additions-only diff guard passes vacuously |
| 2026-08-09 | ci | — | post-release-verify event-aware conclusion guard has no regression assertion |
| 2026-08-10 | ci | — | Add golangci-lint with gofmt and idiomatic Go linters |
| 2026-08-10 | docs | — | `brew trust` instructions recommend broader tap grant with no security framing |
| 2026-08-10 | ci | — | Tap App secret distinctness test is tautological and reads no workflow |
| 2026-08-14 | bench | — | `tools/bench/runner/main.go:482` `pinnedAt()` validates a checkout by `git rev-parse HEAD` alone — the HEAD-only anti-pattern Phase 1's four-part integrity check replaces |
| — | mcp | major | **CR-01 — `internal/mcp/server.go` `pendingWriter` counter corrupted by server-initiated notifications. NOW IN SCOPE as v0.12.0 `FIX-01`, Phase 1.** |

Resolved and filed to `.planning/todos/completed/`:

| Resolved | Area | Title |
|----------|------|-------|
| 2026-07-28 | docs | Document release procedures (maintainer runbook) — closed by 09-04's `docs/RELEASE-PROCEDURES.md` rewrite |
| 2026-07-31 | perf | Bisect the indexer throughput regression — **REFUTED**; the regression did not exist (cross-platform baseline comparison) |
| 2026-07-31 | perf | Rebless perf baseline on ubuntu-latest — **DONE**; gate green on main |
| 2026-08-13 | agents | Author a codegraph usage skill for agents — closed by v0.10.0 Phases 6–8 |

### Blockers/Concerns

Nothing blocks v0.12.0. Carried forward from prior milestones:

- **CR-01 is no longer just carried — it is scoped.** `internal/mcp/server.go:225-349`'s `pendingWriter` "pending response" counter increments only on accepted client requests but decrements on every stdout `Write()`, including server-initiated notifications (`notifications/tools/list_changed`, `notifications/subscriptions/acknowledged`) that SPEC-09 routes through the identical writer. A notification landing between a request's acceptance and its response being written can zero the counter early, causing premature EOF propagation and silent loss of the still-in-flight response — confirmed reachable, not theoretical. Predates v0.10.0 (introduced in `13f2875`). Full trace and proposed fix in `.planning/phases/05-mcp-resources-capability-claims-drift-guard/05-REVIEW.md` (archived under `milestones/`). Now `FIX-01`, Phase 1.
- **Backlog bookkeeping inconsistency (needs a maintainer call).** `999.3` and `999.6` were both promoted into v0.3.0, but all `999.x` Backlog entries were preserved verbatim in `ROADMAP.md` by explicit instruction. Decide whether the promoted entries should be struck or annotated; nothing was removed pending that call. (`999.5` has since been consumed by v0.5.0; `999.2` and `999.4` remain.)
- **Client-side `tools/list` caching bugs are a known confound.** Real, primary-source GitHub issues exist against Claude Code itself (anthropics/claude-code #41123, #40025, #50515; claude-ai-mcp #45).
- **Open GitHub issues:** #14 provenance-over-checksums wording still uncorrected in `release.yml` and two docs · #15 `PRFILES_EOF` heredoc over fork-controlled paths in two `pull_request_target` workflows · #16 `CheckRegression` still never compares `Metrics.Repo` (corpus identity).
- **Advisory, unregistered surfaces** from the v1.0 Phase 10 security audit: the four `pull_request_target` workflows and the darwin canary have no threat-register entry, having landed after their registers were authored.
- **`GOOS=windows go vet`** on `internal/daemon` / `internal/graphstore` fails (`undefined: tree_sitter.Node` in `goextract/routes`) — CGo grammar bindings excluded under windows build constraints; pre-existing. Native Windows support was dropped in `v0.4.0` (WSL2 only).
- **GO-2026-5932 is a real, ACCEPTED, unmitigated exposure in release tooling.** goreleaser's binary reaches `golang.org/x/crypto/openpgp` (110 vulnerable symbols) via pipe/ko → google/ko → sigstore/cosign/oci → sigstore/rekor/pkg/pki/pgp. Upstream is unmaintained (Fixed in: N/A). The advisory `tool-vuln` job surfaces it — reported, not resolved. **Relevant to `BLD-06`:** `pnpm audit` adds a second, disjoint scanner covering the JS tree neither `govulncheck` nor Syft can see; `SECURITY.md` must state both scanners' actual scope rather than implying one covers everything.
- **Daemon extreme-load tail (ACCEPTED, not a gap).** 52/52 real `ci.yml` runs show no daemon failure on the actual runner class; CI load was ruled the governing standard for MAINT-02 (maintainer, 2026-08-06).
- **Wire-oracle `toolslist-repeat` ordering flake.** `TestFrozenTranscriptsMatch/toolslist-repeat` freezes JSON-RPC response *arrival* order, which the protocol does not guarantee and go-sdk's async dispatch does not provide.
- **Tooling gaps (not blocking work, and not hand-edited per the planning-artifacts rule):** `gsd-tools query state.advance-plan` failed with "Cannot parse Current Plan or Total Plans in Phase from STATE.md" when Current Position read "Plan: Not started". `gsd-tools query state.sync` counts a SUMMARY with `status: halted` as a completed plan, and MUTATES when invoked with no args — it has no dry-run probe mode.

### Quick Tasks Completed

| # | Description | Date | Commit | Directory |
|---|-------------|------|--------|-----------|
| 260807-gho | Drop native Windows support — WSL2 only | 2026-08-07 | 085b7a3 | [260807-gho-drop-native-windows-support-wsl2-only](./quick/260807-gho-drop-native-windows-support-wsl2-only/) |
| 260811-s5o | Install cosign in post-release-verify's self-upgrade job (v0.9.0 self-upgrade proof failed closed on a missing installer) | 2026-08-11 | 6135785 | [260811-s5o-add-sha-pinned-sigstore-cosign-installer](./quick/260811-s5o-add-sha-pinned-sigstore-cosign-installer/) |

## Deferred Items

Items acknowledged and deferred at milestone close, most recent first.

Deferred at **v0.12.0 scoping (2026-08-22)** — recorded so the decisions stay visible:

| Category | Item | Status |
|----------|------|--------|
| requirement | HLT-04 — coverage denominator ("files discovered but NOT indexed") | v2; needs new Engine surface, deliberately out of v1 scope |
| requirement | BRW-10 — "where am I" breadcrumb while scrolling a long file | v2 |
| requirement | BRW-11 — editor handoff (`vscode://file/...`) with configurable URI scheme | v2 |
| requirement | GRF-06 — community-detection clustering | v2; the schema reserves annotation space, directory-structural grouping is the correct v1 substitute |
| requirement | GRF-07 — opt-in whole-symbol graph within an already-drilled-into package | v2 |

Acknowledged at the **v0.11.0 close (2026-08-16)** — suppressed at the next `audit-open` scan, not fixed. All 10 predate v0.11.0. Acknowledgment is verdict-preserving and **self-invalidating**: it never rewrites an artifact's own `status:`, and the suppression lapses the moment the artifact's observed state changes again.

| Category | Item | Status | Deferred At | Milestone |
|----------|------|--------|-------------|-----------|
| debug_sessions | knowledge-base | unknown | 2026-08-17 | v0.11.0 |
| todos | 2026-08-07-wire-oracle-toolslist-repeat-response-ordering-flake.md | (presence-only) | 2026-08-17 | v0.11.0 |
| todos | 2026-08-09-dry-run-signed-additions-only-diff-guard-passes-vacuously.md | (presence-only) | 2026-08-17 | v0.11.0 |
| todos | 2026-08-09-post-release-verify-event-aware-conclusion-guard-has-no-regression-assertion.md | (presence-only) | 2026-08-17 | v0.11.0 |
| todos | 2026-08-10-add-golangci-lint-with-gofmt-and-idiomatic-go-linters.md | (presence-only) | 2026-08-17 | v0.11.0 |
| todos | 2026-08-10-brew-trust-instructions-recommend-the-broader-tap-grant-with-no-security-framing.md | (presence-only) | 2026-08-17 | v0.11.0 |
| todos | 2026-08-10-tap-app-secret-distinctness-test-is-tautological-and-reads-no-workflow.md | (presence-only) | 2026-08-17 | v0.11.0 |
| seeds | SEED-001-local-svelte-shadcn-graph-browsing-ui | **CONSUMED by v0.12.0** | 2026-08-17 | v0.11.0 |
| seeds | SEED-003-markdown-in-the-index | dormant | 2026-08-17 | v0.11.0 |
| deferred_items | 04/deferred-items.md: `internal/daemon` load-induced flake under full-suite parallel load | acknowledged | 2026-08-17 | v0.11.0 |

The `deferred_items` entry was acknowledged by hand rather than through `audit-open acknowledge`: that file uses the heading-delimited (#3457) entry shape, which the CLI writer deliberately refuses rather than risk writing into the wrong entry.

Deferred at v0.11.0 scoping (2026-08-13) — recorded so the decisions stay visible:

| Category | Item | Status |
|----------|------|--------|
| requirement | DOCS-05 — self-authored `docs/CLI-REFERENCE.md` with its own drift guard | v2; v0.11.0 deleted `docs/FLAG-PARITY.md`, a later milestone authors the replacement |
| requirement | VOCAB-01 — build-time vocabulary drift guard | **declined**, not deferred (blocklist goes vacuous or fights `tsextract`) |

Carried forward, acknowledged and deferred at v0.10.0 close on 2026-08-13:

| Category | Item | Status |
|----------|------|--------|
| requirement | GUARD-HOOK-01/02 — PreToolUse guard hook | v2; the fallback if skill+resources+nudge prove insufficient, and that evidence does not exist yet |
| requirement | AGENT-04…07 — multi-agent skill/hooks porting | v2; blocked on per-agent hook-schema differences across Cursor / Codex CLI / Antigravity |
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
| seed | SEED-001 — local Svelte + shadcn-svelte UI for browsing/querying the graph | **consumed by v0.12.0** (scoped across all 6 phases) |
| seed | SEED-002 — homebrew installation path | consumed by v0.5.0 Phases 1–4; tap published, cask installable |
| backlog | 999.5 — macOS Gatekeeper signing and notarization | promoted into v0.5.0 Phase 2 (SIGN-01…04) |
| backlog | 999.6 — MCP `2026-07-28` impact assessment | the spine of v0.3.0 |
| backlog | 999.3 — vulnerability scanning for the tool modfiles | VULN-01/02/03, v0.3.0 Phase 4 |
| todo | 2026-08-08 — author a codegraph usage skill for agents | closed by v0.10.0 Phases 6–8 |
| Release | DIST-02 — real signed `v*` tag | ✓ `v0.2.0` shipped via release-please (REL-02) |
| Perf | PERF-01 — published head-to-head numbers | ✓ retired by v0.11.0 BENCH-01; replaced by absolute single-subject numbers |

Not deferred but worth carrying: `.planning/debug/perf-gate-throughput-regress.md` is backed by
fresh data on issue #20 — the gate failed then passed on an inert diff at ~6.9% intra-run spread
against a 10% budget.

## Session Continuity

**Resume file:** None

Last session: 2026-08-29T02:37:38.682Z
Stopped at: Completed 03-06-PLAN.md
  NEXT: `/gsd-plan-phase 2` (SPA Toolchain, Embedded App Shell & JS Supply Chain)
  CARRY-OVER:

    - **Phase numbering restarts at 1** (this milestone was started with `--reset-phase-numbers`). `.planning/phases/` holds only `999.x` backlog directories, so Phases 1–6 collide with nothing.
    - **`branching_strategy: milestone`** — this milestone lives on one branch (`gsd/v0.12.0-local-graph-ui`) and is not incrementally merged.
    - **Phase 1 is complete and verified** (5/5 ROADMAP criteria, `01-VERIFICATION.md`). The Connect server, the nine read-only RPCs, the Origin/Host guard and the `internal/query` seams are live and golden-clean.
    - **Phase 2 inherits one open UAT item by design:** `01-UAT.md` test 1 records `GET /` → 404 because Phase 1 mounts only the Connect handler prefix and ships no SPA. It is recorded as `deferred` in `01-VERIFICATION.md`, and Phase 2's success criterion 1 (embedded SPA on this same mux) closes it by construction. `phase uat-passed 1` therefore reports `passed:false` on that one named blocker — this is a real, non-vacuous blocker, not the empty-blockers failure mode.
    - **The protobuf half of the drift guard already exists** (`task proto:drift`, 3 files compared, RED-proven). Phase 2's `dist/`↔SPA guard is the second consumer of that one pattern, not a new one.
    - **Within Phase 5, `GRF-01` blocks everything else in the phase.** Its pass condition must be written down before it is dispatched, and the roadmap deliberately names no renderer.
    - **No `v0.12.0` git tag.** release-please owns tagging (D-06R); a hand-created tag would match `release.yml`'s `v[0-9]*` trigger and falsely fire the release pipeline.
    - **`.planning/` and `CHANGELOG.md` stay tool-owned** — no invented headings, and no version-bearing or ✅-bearing `###` heading under `## Phases` other than the single active-milestone heading.

## Operator Next Steps

- Plan Phase 2 with `/gsd-plan-phase 2`
