---
gsd_state_version: "1.0"
milestone: v0.14.0
milestone_name: Polish & Agent Reach
current_phase: 7
current_phase_name: Codex Parity
status: executing
stopped_at: Completed 07-10-PLAN.md
last_updated: "2026-09-19T21:56:16.782Z"
last_activity: 2026-09-19
last_activity_desc: Phase 7 execution started
state_head: fb1840374416397c5092e2748154a13460e51ab7
progress:
  total_phases: 7
  completed_phases: 6
  total_plans: 53
  completed_plans: 52
  percent: 86
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-09-19 after Phase 6)

**Core value:** CodeGraph Go gives coding agents a pre-indexed code knowledge graph — fast symbol/call-path/impact queries served from a single static, verifiably-built binary, with no bundled runtime to install or manage.
**Current focus:** Phase 7 — Codex Parity

## Current Position

Phase: 7 (Codex Parity) — EXECUTING
Plan: 11 of 11
Status: Ready to execute
Last activity: 2026-09-19 — Phase 7 execution started

## Performance Metrics

**Velocity (v0.12.0):**

- Phase 1: 11 plans across 7 waves, executed and verified 2026-08-23 (1 day). Per-task timings were not recorded in this phase's SUMMARY frontmatter, so no task count is reported rather than an inferred one.

**By Phase (v0.12.0):**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01 | 9 | - | - |
| 02 | 7 | - | - |
| 03 | 4 | - | - |
| 4 | 8 | - | - |
| 5 | 7 | - | - |
| 6 | 7 | - | - |
| 7 | 4 | - | - |
| 08 | 5 | - | - |
| 09 | 6 | - | - |
| 10 | 6 | - | - |
| 11 | 5 | - | - |
| 12 | 3 | - | - |

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
| Phase 03-browse-inspect-navigation P07 | ~50min | 3 tasks | 10 files |
| Phase 03 P08 | ~13min | 3 tasks | 10 files |
| Phase 03 P09 | 9min | 3 tasks | 9 files |
| Phase 03 P10 | ~90 min | 3 tasks | 34 files |
| Phase 04 P02 | 30min | 3 tasks | 6 files |
| Phase 04 P01 | 85min | 3 tasks | 25 files |
| Phase 04 P03 | 70min | 3 tasks | 7 files |
| Phase 04 P04 | 40min | 3 tasks | 11 files |
| Phase 04 P05 | 55min | 3 tasks | 15 files |
| Phase 04-query-workbench-index-health P06 | 25min | 3 tasks | 9 files |
| Phase 04 P07 | 40min | 3 tasks | 40 files |
| Phase 04 P07 | 70min | 4 tasks | 44 files |
| Phase 05 P01 | 20min | 3 tasks | 5 files |
| Phase 05 P02 | 30min | 3 tasks | 7 files |
| Phase 05 P03 | 40 min | 3 tasks | 12 files |
| Phase 05 P05 | 3h40min | 3 tasks | 7 files |
| Phase 05-file-package-graph-view P06 | 45min | 3 tasks | 9 files |
| Phase 05 P07 | ~100 min | 3 tasks | 10 files |
| Phase 06 P01 | 191min | 3 tasks | 8 files |
| Phase 06 P02 | 210min | 3 tasks | 2 files |
| Phase 06 P03 | 54min | 3 tasks | 12 files |
| Phase 06-live-push P04 | 95min | 3 tasks | 9 files |
| Phase 06 P05 | 640min | 3 tasks | 5 files |
| Phase 06 P06 | 22min | 2 tasks | 3 files |
| Phase 06 P07 | 45min | 3 tasks | 4 files |
| Phase 07 P01 | 8min | 3 tasks | 3 files |
| Phase 07 P02 | 22 min | 3 tasks | 5 files |
| Phase 07 P03 | 18min | 3 tasks | 4 files |
| Phase 07 P04 | 15min | 3 tasks | 4 files |
| Phase 08 P01 | 35min | 3 tasks | 6 files |
| Phase 08 P02 | 27min | 3 tasks | 7 files |
| Phase 08 P03 | ~20min | 2 tasks | 2 files |
| Phase 08 P04 | ~25min | 3 tasks | 5 files |
| Phase 08 P05 | 25 min | 3 tasks | 8 files |
| Phase 09 P01 | 30min | 2 tasks | 15 files |
| Phase 09 P02 | 14min | 2 tasks | 4 files |
| Phase 09 P03 | 35min | 3 tasks | 45 files |
| Phase 09 P04 | 25min | 3 tasks | 34 files |
| Phase 09 P05 | 30min | 3 tasks | 2 files |
| Phase 09 P06 | 20min | 2 tasks | 14 files |
| Phase 10 P01 | 1h 5min | 2 tasks | 37 files |
| Phase 10 P02 | ~50min | 2 tasks | 13 files |
| Phase 10 P03 | 35min | 2 tasks | 4 files |
| Phase 10 P04 | 55min | 2 tasks | 5 files |
| Phase 10-index-health-the-coverage-denominator P05 | 55min | 2 tasks | 3 files |
| Phase 10 P06 | 50min | 2 tasks | 3 files |
| Phase 01 P01 | 45min | 2 tasks | 20 files |
| Phase 01 P02 | 35 min | 3 tasks | 6 files |
| Phase 01 P03 | 7min | 2 tasks | 2 files |
| Phase 01 P04 | 20min | 2 tasks | 2 files |
| Phase 01 P06 | ~30min | 2 tasks | 21 files |
| Phase 01 P07 | 20 min | 2 tasks | 4 files |
| Phase 01 P05 | 40min | 3 tasks | 1 files |
| Phase 01 P08 | 55min | 3 tasks | 2 files |
| Phase 01 P09 | ~40min | 2 tasks | 2 files |
| Phase 02 P01 | 7min | 2 tasks | 3 files |
| Phase 02 P03 | 25min | 2 tasks | 4 files |
| Phase 02 P05 | 15min | 2 tasks | 45 files |
| Phase 02 P06 | 20 min | 3 tasks | 2 files |
| Phase 02 P02 | 55min | 2 tasks | 4 files |
| Phase 02 P07 | ~25min | 3 tasks | 4 files |
| Phase 02 P04 | 12min | 2 tasks | 4 files |
| Phase 04 P01 | 22min | 3 tasks | 32 files |
| Phase 04 P02 | ~20min | 2 tasks | 4 files |
| Phase 04 P03 | 32min | 2 tasks | 15 files |
| Phase 4 P04 | 15min | 2 tasks | 7 files |
| Phase 04 P05 | 38min | 2 tasks | 6 files |
| Phase 04 P06 | 42min | 2 tasks | 7 files |
| Phase 04 P07 | 37min | 2 tasks | 7 files |
| Phase 04 P08 | 20min | 3 tasks | 7 files |
| Phase 05 P01 | 33min | 3 tasks | 21 files |
| Phase 05 P02 | 55min | 2 tasks | 5 files |
| Phase 5 P03 | 55min | 2 tasks | 5 files |
| Phase 5 P04 | 38min | 2 tasks | 13 files |
| Phase 05 P05 | ~40min | 2 tasks | 11 files |
| Phase 05 P06 | 1h32m | 3 tasks | 1 files |
| Phase 05 P07 | 13min | 3 tasks | 9 files |
| Phase 06 P01 | 10min | 2 tasks | 17 files |
| Phase 06 P02 | 5min | 2 tasks | 6 files |
| Phase 06 P03 | 10min | 3 tasks | 5 files |
| Phase 06 P04 | 46min | 3 tasks | 11 files |
| Phase 06 P05 | 11min | 3 tasks | 11 files |
| Phase 06 P06 | 4h | 3 tasks | 1 files |
| Phase 06 P07 | 5min | 2 tasks | 3 files |
| Phase 07 P01 | 51 min | 3 tasks | 7 files |
| Phase 07 P02 | 15min | 2 tasks | 6 files |
| Phase 07 P03 | ~15min | 3 tasks | 6 files |
| Phase 07 P04 | ~15min | 3 tasks | 1 files |
| Phase 7 P05 | ~55min | 3 tasks | 11 files |
| Phase 07 P06 | ~25min | 3 tasks | 7 files |
| Phase 07 P07 | ~40min | 3 tasks | 14 files |
| Phase 07 P08 | ~50min | 3 tasks | 11 files |
| Phase 7 P09 | ~15min | 3 tasks | 1 files |
| Phase 07 P10 | ~18min | 2 tasks | 4 files |

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
- [Phase 03-browse-inspect-navigation]: [Phase 03]: [Phase 03-07]: browse-nav.ts's symbol/file/line target-kind clearing — setting symbol without file clears file+line, setting file without symbol clears symbol (+line unless the same delta also sets line), and the symbol+file+line disambiguation triple (or any caller-directed symbol+file pair) clears nothing
- [Phase 03-browse-inspect-navigation]: [Phase 03]: [Phase 03-07]: NavigationGeneration is a plain module-scoped counter (createNavigationGate) rather than tied to $app/state/runes, kept testable with no SvelteKit runtime — closes the cross-load race a per-call AbortController alone cannot (a response in flight can still arrive after abort fires)
- [Phase 03-browse-inspect-navigation]: [Phase 03]: [Phase 03-07]: extended SourcePane.svelte for single-def/multi-def rendering though not in the plan's declared files_modified — Task 2's own pnpm check broke on the BrowseTargetState union growing, and the plan's must-haves require source+callers+callees+blast-radius visible together for an opened symbol
- [Phase 03-browse-inspect-navigation]: [Phase 03]: [Phase 03-07]: found live (manual UAT) that typing in search never wrote q into the URL — added SearchPanel's onQueryChange prop wired to a REFINE navigate() call in +page.svelte, required by the plan's own must_haves though not in Task 3's task-level behavior bullets
- [Phase 03]: [Phase 03-browse-inspect-navigation]: [Phase 03-08]: truncated-file permalink requests carry NO anchor at all (neither line nor end_line) — the strict reading of D-20's "the rest of the file is what the local view could not show"; any partial anchor still frames the remote view around the truncated portion
- [Phase 03]: [Phase 03-browse-inspect-navigation]: [Phase 03-08]: SourcePane.svelte's local `state` prop binding renamed to `target` internally (destructure rename, external prop name unchanged) — a bare local named `state` collides with the `$state` rune used elsewhere in the same component, the same pitfall 03-06/03-07 already documented, hit a third time
- [Phase 03]: Shared status gate (D-04/D-05) exposes reactive status via the Svelte store contract so both the layout banner and a descendant route's stale-source-message split can subscribe to the SAME gate instance with zero adapter code and no second fetch. — One shared reactive interface serves two consumers (layout banner, browse source-pane message) without duplicating the fetch trigger — keeps D-05's no-polling constraint intact even as more views need the same index-health data.
- [Phase 03]: Closed the intended-RED web:drift window opened by 03-01: rebuilt and re-committed web/build/ (source digest fc4ae27b...->f9a3632a..., 22->73 files; output digest c599a63e...->5162b279..., 25->27 files), verified GREEN. — The window was scheduled to close at 03-09 Task 3 from the start (03-01-SUMMARY.md), so every intervening plan's red web:drift leg was expected, not a regression.
- [Phase 03]: golangci-lint pinned in a fourth isolated tool modfile (go.tool-golangci.mod); gofmt (not gofumpt) chosen for the first formatting gate; std-error-handling exclusion preset and unlimited issue-reporting caps enabled for correctness.
- [Phase 03]: Task 2's real first-run backlog was 47 issues/26 files (not the plan's pre-measured 8-file formatting-only set); all fixed per the plan's own fix-dont-suppress mandate rather than narrowing the linter set.
- [Phase 03]: The pre-existing go.tool-proto.mod isolation/vuln-scan gap (since Phase 1) was closed alongside golangci-lint's own registration, with a new population-vs-disk guard preventing recurrence for any future tool modfile.
- [Phase 4]: D-14 (04-CONTEXT.md): Engine.Files glob bug fixed by swapping filepath.Match for doublestar.Match — one matcher mechanism, not a second Substring option
- [Phase 4]: go.mod's doublestar require added by hand (go get + manual indirect->direct promotion) rather than via full go mod tidy — pre-existing unrelated tree-sitter-swift module-resolution failure blocks tidy on this branch (confirmed on clean checkout)
- [Phase 4]: 04-01: Maintainer approved both [SUS] package-legitimacy verdicts (@tanstack/svelte-table@9.2.4, shadcn-svelte@1.5.1) — same too-new false-positive shape 03-01 already approved.
- [Phase 4]: 04-01: DataTable.svelte is generic over its row type (TRow extends RowData), never Location-typed, so 04-05's CountRow health tables reuse the same shell.
- [Phase 4]: 04-01: status.ts's navigationIdentity gained a route-scoped ROUTE_LOCAL_PARAMS table (T-04-32) — Workbench control changes mint no new navigation identity, while the same parameter names under /browse still correctly refetch.
- [Phase 4]: 04-03 Task 1 checkpoint: maintainer approved GetHealth's frozen wire shape (16-field GetHealthResponse + WorktreeMismatch/PendingChanges/IndexHealth), all 5 sub-decisions answered explicitly
- [Phase 4]: 04-03: GetHealth follows the ordinary withEngine handler shape (Callers/Callees/Files convention), not GetStatus's degrade-and-answer exception
- [Phase 4]: AnalysisPanel.svelte's dispatch effect tracks only requestKey (via untrack on run) — the mechanism behind no-extra-GetStatus on depth/limit edits
- [Phase 4]: Each tab's AnalysisPanel is {#if}-gated inside its Tabs.Content, since bits-ui mounts every tab's content simultaneously and toggles only 'hidden'
- [Phase 4]: WRK-01 no-remount property asserted via DOM node identity on the always-rendered heading, not an injected onMount spy
- [Phase 4]: IndexStatus widened additively with commitSha:string (cycle-2 fix); StatusVerdict/CommitKnowledge member counts unchanged
- [Phase 4]: TrustVerdict renders all five StatusVerdict branches (unlike StatusBanner) to answer HLT-02's affirmative-trust question with the same classifier
- [Phase 4]: 04-06: Extracted debounced-rpc.ts from search.ts's live path (D-15) rather than writing a second debounce/abort/identity implementation for file-search.ts — configured twice, with onBelowMinimum() as the seam between mechanism (abort+invalidate) and caller state-shaping.
- [Phase 4]: 04-06: escapeGlobLiteral neutralizes typed glob metacharacters (\ * ? [ ] { }) before they reach Engine.Files, verified against doublestar.Match's actual escaping contract rather than assumed from docs.
- [Phase 4]: All 8 vendored shadcn-svelte component families reproduce byte-identically at pinned 1.5.1 (04-07 Task 1)
- [Phase 4]: web:components:drift scopes its comparison strictly to components/ui/ — the CLI's own dependency-install step mutates package.json as a side effect
- [Phase 4]: Workbench DataTable's 1000-row render cost measured OVER threshold on both metrics (reproduced 4x); virtualization NOT installed — package-legitimacy checkpoint requested
- [Phase 4]: Maintainer approved @tanstack/svelte-virtual@3.13.36; wired into DataTable.svelte, render-cost medians now ~20-40x under threshold (task web:render-cost passes)
- [Phase 05]: GRF-01 pass condition locked and committed alone (05-01 Task 1, maintainer approve-as-proposed) before any measurement exists: google/guava corpus, expanded file-level binding view, 4 metric bars with stated timer boundaries, 16-value measurement protocol including 5 deadline budgets, two-remedy onFailure path.
- [Phase 05]: Engine.FileGraph()'s regression test against this repository's own live index asserts the D-08 structural invariant (no empty-path node/edge, ExcludedPackageNodes > 0) rather than the plan's literal 572/1057 counts, because this repository indexes itself via a live daemon and exact counts are not stable across this plan's own commits.
- [Phase 5]: FileGraph frozen at Task 1 blocking-human checkpoint, approve-as-proposed: FileGraphRequest(1)/FileGraphNode(4)/FileGraphEdge(5)/FileGraphResponse(6), all four sub-decisions confirmed (excluded_* counters kept, in_cycle kept redundant, name FileGraph re-verified against 19 mutatingVerbs substrings, unprefixed message names).
- [Phase 5]: Guava-scale FileGraph response measured (not estimated): 3,713,528 bytes serialized, 3233 nodes, 21554 edges, 162 cycles — 22.1% of the 16 MiB transportSendMaxBytes ceiling, replacing 05-RESEARCH.md's unverified 5-6 MB estimate.
- [Phase 05]: Maintainer approved all three renderer dependencies (cytoscape, cytoscape-elk, elkjs) by name; elkjs resolved transitively to 0.9.3, not the 0.12.0 research measured, and an override to force 0.12.0 was explicitly declined.
- [Phase 05]: Live browser verification against this repository's own index found and fixed a real bug: cytoscape boolean data selectors require [?field]/[!field] existence syntax, not [field = true] equality, which silently never matches.
- [Phase 05]: GRF-01 measured FAIL against the pinned google/guava corpus (expanded file-level scale, 3,233 nodes / 21,554 edges) — the layout never became interactive within the locked 60s seamReadyTimeoutMs, independently reproduced still not interactive after 10+ minutes. Maintainer decision, 2026-08-30: `halt-collapse-default` — make the collapsed directory view (134 nodes / 819 edges) the default first paint with progressive expansion, keep Cytoscape + cytoscape-elk, re-measure at collapsed scale before 05-05/05-06/05-07 resume. `halt-reconsider-stack` was rejected (the compound-node-narrowed field had nowhere better to go) and widening/lowering the threshold was never an available answer (corpora/graph-render-threshold.json retains exactly one commit, 2fb27746). See 05-04-SUMMARY.md.
- [Phase 05]: 05-08 Task 4 (checkpoint:decision, gate=blocking-human) is answered: maintainer selected `release-collapsed` (2026-08-30). GRF-01's re-measure at the collapsed scale cleared all four locked bars with wide margin against corpora/graph-render-threshold.json (timeToInteractiveMs 1179.2ms vs 5000ms, panZoomFrameTimeMs 8.3ms vs 33.3ms, panZoomFrameTimeP95Ms 9ms vs 100ms, fileGraphResponseBytes 3,713,528 vs 16,777,216). Rendered 134/819 exactly equal the collapsed-view figures the locked artifact pre-recorded. Controls independently re-verified: threshold has exactly 1 commit, clean diff over threshold/graph-measure.mjs/graph-render-observations.json, no value widened, lowered or re-scoped. 05-05, 05-06 and 05-07 resume onto the collapsed default — their preconditions and wave numbers (6/7/8) were already re-pointed at corpora/graph-render-observations-collapsed.json and this answer. The collapse-by-click hit-testing defect (WINDOWS.md 27, ~1.1px clickable margin around a compound directory once it has children, three mitigations tried and each measured ineffective) does NOT block this gate — GRF-01 asks whether the qualifying renderer stays interactive at the largest corpus, which it does decisively; the defect is assigned to 05-07, whose verified three-click contract requires an explicit collapse affordance rather than depending on the sub-2px margin. See 05-08-SUMMARY.md and corpora/graph-render-observations-collapsed.json.
- [Phase 05]: Cycle discriminator classes extend to collapsed directory nodes, not just file nodes. — 05-08's collapsed default means a file's cycleId is invisible until its directory is expanded, so collapsedDirElement's existing cycleIds union needed its own classes too.
- [Phase 05]: Cycle-focus grouping reads both data.cycleId (file nodes) and data.cycleIds (collapsed directories). — Every file cycle id is represented by exactly one element in any given view, so grouping always yields exactly cycleCount distinct groups regardless of expansion state.
- [Phase 05-file-package-graph-view]: 05-06 Task 1 checkpoint (blocking-human, one-way): maintainer approved FileSymbols (UIService's 13th rpc) exactly as proposed - reuse the shared Node message, MaxFileSymbols=2000 owned by internal/query, and an index-miss returns an empty result not an error.
- [Phase 5]: Symbol expansion added an explicit DOM collapse-affordance button (shared with directory-level collapse) rather than relying on canvas re-click, closing WINDOWS.md 27 with real-mouse Playwright proof at both this repo's and google/guava's scale
- [Phase 06]: WatchGraph frozen as the 14th UIService rpc, the services first Connect server-streaming method, with seven field numbers pinned (since_generation=1; generation=1, initialized=2, stale=3, store_exists=4, indexing_in_progress=5, commit_sha=6). — Verified clean against the live mutatingVerbs fixture with GetIndexHealth as the positive control; D-07s field set mirrors GetStatusResponse so classifyStatus consumes the event directly; server-streaming chosen because the client has nothing to say after subscribing.
- [Phase 06]: Second checkAndPublish mutex (livePublisher.checkMu) serializes compute-then-publish, closing a theoretical publish-reordering gap from concurrent Debouncer fire() calls — internal/watch.Debouncer's own doc comment documents a real possibility of two concurrent fire() invocations; without an extra lock beyond changeDetector's own, two such invocations could Publish out of generation order
- [Phase 06]: Per-test goleak.VerifyNone(t, goleak.IgnoreCurrent()) substituted for a package-wide TestMain in internal/uiserver — A package-wide goleak.VerifyTestMain probe failed on ~40 pre-existing, unrelated pebble vfs ticker goroutines from other tests in the package; scoping to IgnoreCurrent() proves the same 'no leak from this code' property without that collateral breakage
- [Phase 06]: Backoff shape pinned: base 1000ms, factor 2, cap 30000ms, jitter spread 0.25 (< 1/3). Exponent resets only on an event; connection epoch increments only on establishment - two independent counters for T-06-42 vs T-06-41.
- [Phase 06]: classifyStatus widened to StatusLikeFields so a WatchGraphEvent satisfies it directly; StatusGate.applyLiveEvent emits with no getStatus call, ordered against fetchStatus by one shared monotonic counter with (epoch,generation) dedup.
- [Phase 6]: clearWatchDeadline clears the absolute write deadline for exactly the WatchGraph procedure, passing the ResponseWriter through unwrapped so connect-go's flush-capability gate still admits the handler
- [Phase 6]: Server owns the live-push publisher's lifetime end to end: Listen constructs it, Serve and Close both stop it idempotently, and Serve stops it BEFORE calling Shutdown so an open stream cannot hold the 5s shutdown budget
- [Phase 6]: Two double-invocation Svelte 5 effect bugs fixed via identity-comparison guards (lastAppliedElements/lastAppliedLiveElements), found only by real-browser testing at guava scale
- [Phase 6]: Criterion 5 verified with a real 3-process gate (daemon+serve --mcp+ui), measured baseline, and a demonstrated RED (store held open -> flushesStarved 3/3)
- [Phase 6]: pendingWriter-analogue verdict recorded: no analogue in internal/uiserver, discriminating control on 'type pendingWriter struct' (=1) vs bare word (=9)
- [Phase 07]: Followed D-10 exactly: current-metrics positivity checks inserted immediately after the baseline positivity checks and before delta math
- [Phase 07]: 07-MUTATION-LOG.md family (a) has no revert step because RED was the absence of the fix, not a mutation of correct code
- [Phase 07]: GRD-02: internal/query archtest scoped the indexer-root rule to the production compilation unit only, allowing internal/query/engine_test.go's legitimate in-package import of the internal/indexer root for fixture construction, per D-01/D-04.
- [Phase 7]: [Phase 07]: 07-03: extracted the dry-run-signed cosign-key injection and additions-only diff guard into scripts/inject-cosign-key.sh, adding the missing positive assertion (exactly 1 injected --key= line) that closes T-02-08's vacuous-guard gap; no shape test added asserting the wiring per D-07
- [Phase 7]: [Phase 07]: 07-03: family (c)'s mutation-log entry has no tracked-file mutation or revert step (deliberate deviation) — the RED perturbation was applied to a copy of .goreleaser.yaml in a temp dir, since the committed release config must never be edited to prove a guard
- [Phase 7]: GRD-04 conclusion-guard test is a sibling of TestPostReleaseJobsDeclareCheckoutPolicy, comparing every job's parsed if: against one verbatim const with no fixed job-id list and no normaliser
- [Phase 7]: Deleted TestHomebrewTapAppSecretsDistinctFromReleasePleaseAppSecrets outright (D-09) rather than rewriting it -- a tautological test comparing two in-test constants is worse than none
- [Phase 08]: [Phase 08-01]: pollUntilStable's interval tuned to 1s/10s deadline (not the initial 100ms/5s guess) — measured a real, consistent 700-900ms pre-execution subprocess-startup plateau on this machine that a sub-second interval would false-converge on; comparison semantics unchanged
- [Phase 08]: [Phase 08-01]: pollUntilStable's interval pacing uses <-time.After(...), never time.Sleep(...) — the plan's own verify gate greps for zero time.Sleep occurrences in test/tmux; matches test/integration's existing goroutine+time.After bounded-wait convention
- [Phase 08]: TTY-05's positive-control assertion uses the picker title, not the help footer text — the footer never renders at the default 100x30 pane with all 8 agent targets due to a bubbles/v2/list pagination-padding overflow, verified via temporary reverted debug instrumentation
- [Phase 08]: confighash.go's doc comment avoids the literal substring 'sha256sum' after the plan's own verify gate tripped on it appearing in explanatory prose rather than a shell-out
- [Phase 08]: [Phase 08]: [Phase 08-03]: tmux-e2e CI job lands on ubuntu-latest with the TMUX_EXPECTED_VERSION sentinel deliberately unresolved — no real ci.yml run exists for this branch yet (gh run list returned empty), so D-11's bootstrap stays in its designed deferred state rather than guessing a version
- [Phase 08]: [Phase 08]: [Phase 08-03]: own comment prose in the new tmux-e2e job tripped the plan's own continue-on-error substring-count gate — reworded to describe the same no-soft-fail property without the literal token, third instance of this phase's recurring substring-proxy gate defect (after time.Sleep in 08-01, sha256sum in 08-02)
- [Phase 08]: [Phase 08-04]: Family (d)'s D-06-specified v.AltScreen=false mutation does NOT fail TestInstallPickerFrameStableWhileIdle — the test converges past the settling transient before its idle-stability loop begins, and the AltScreen-driven scroll this mutation targets is confined to that transient. Confirmed reproducibly (two runs); reported honestly in 08-MUTATION-LOG.md and WINDOWS.md (unmet-truth) rather than forced. — Following the plan's own explicit contingency instruction ("stop and report it... do not adjust the test") and Phase 7's D-07 precedent against adding a second guard/test for a property one assertion already covers.
- [Phase 08]: [Phase 08-05]: G-08-1 (TTY-03 cold-start poll race, found by verify-work re-executing the covering check at HEAD: FAILED 3/3 cold, PASSED ~13/13 warm) closed by giving pollUntilStable a REQUIRED readiness predicate — convergence is ready(capture) && capture == predecessor — rather than warming the binary in TestMain or raising stabilityPollInterval; a deterministic self-test commanding a fixed shell delay reproduces the race without depending on machine coldness (RED e994b0a5 → GREEN b9dfc849); TMUX_EXPECTED_TESTS 5→6 in the same commit as the sixth test.
- [Phase 08]: verify-work 2026-09-11: 21/21 UAT pass, 0 issues. Family (d) checkpoint asked for "a decision recorded"; the user's "defer" IS that decision, so it is recorded as pass with the deferred follow-up kept under 08-UAT.md → Deferred Follow-Ups (gsd-core's uat-predicate blocks on any `skipped`, contradicting #1921 — see Tooling gaps). 08-VERIFICATION.md status canonicalized to passed after the TTY-07 backstop fired at HEAD 5ffdc2a0 (CI run 34658987243: executed=6 skipped=0 expected=6, tmux 3.4).
- [Phase 08]: Fixed pollUntilStable's G-08-1 cold-start race by giving it a required readiness predicate (ready(capture) && capture == predecessor) rather than warming the binary or raising the poll interval.
- [Phase 08]: Cold-arm evidence runs all showed K<=1 (machine warm); did not re-run chasing K>=2 per the plan's own interpretation rule — relied on the deterministic self-test's RED/GREEN transcripts as proof instead.
- [Phase 09]: GetEditorLink shares GetPermalink's validator/answer-not-error discipline; field-number fixture extended additively (uiProtoFieldFixtureLenAtPlan0901)
- [Phase 09]: [Phase 09]: 09-02: gofmt's struct-literal column alignment padded discover: with extra spaces, defeating the plan's own single-space literal verify grep -- fixed by adding an explanatory comment above the field to break gofmt's alignment group, satisfying both gofmt and the check without touching either
- [Phase 09]: [Phase 09]: 09-02: templateForLauncher never restates the vscode/cursor template strings -- it looks them up from uiserver.EditorPresets() by ID, so both presets have exactly one source of truth across plans 09-01 and 09-02
- [Phase 09]: 09-03: innermostSymbolAt/firstFullyVisibleLine kept pure and DOM-free in breadcrumb.ts, unit-tested independently of SourcePane
- [Phase 09]: 09-03: gutter renders plain digits this plan (D-10) — 09-04 turns cells into links, no link markup added here
- [Phase 09]: 09-03: breadcrumb symbol is a button, never a hash-fragment <a>, to avoid colliding with the browse route's own URL-driven navigation identity
- [Phase 09]: 09-03: breadcrumb-check.mjs's oracle independently re-implements innermost-range derivation (sort-based) and fetches FileSymbols directly over HTTP, never importing the SPA's own module
- [Phase 09]: editor-prefs.ts is the SPA's first localStorage consumer: try/catch every access, degrade to no-override on any failure; templateForRequest is the one seam both the probe and a gutter click resolve through
- [Phase 09]: [Rule 3] Node >=26's built-in global Web Storage API shadows jsdom's real localStorage and no-ops silently without --localstorage-file; fixed with a probe-and-replace in-memory Storage shim in web/tests/setup.ts, test-infra only
- [Phase 09]: Family (b)'s live-gate demonstration needed a different --file target (internal/cli/editorurl.go) than the script's default, since the default's structure lets the scroll loop's own break condition fire before reaching the exposing gap
- [Phase 09]: 09-SECURITY.md records Cursor/JetBrains preset templates as still [ASSUMED] rather than confirmed — no real IDE was available to click through in this autonomous session; WINDOWS.md #35 stays open
- [Phase 09]: [Phase 09] 09-06: Closed the svelte-check gap (2 errors at SourcePane.svelte:446:31, introduced by CR-01 commit 28d5d795) by capturing the guard-narrowed getEditorLink once and calling it from both the initial probe and the corrective re-probe -- no behavioral change, gate restored to 0 errors, task web:build/web:drift both MATCH.
- [Phase 10]: prefixExcludedFile='c'; ExcludedByReason keyed by full enum name; Export Open Question 1 decided as implement (exportKindExcludedFile=5).
- [Phase 10]: [Phase 10] Assumption A1 (dangling .py symlink for extraction-failure fixture) VERIFIED on first probe run — no chmod fallback needed
- [Phase 10]: [Phase 10] Coverage-index test fixtures must never pre-create repoRoot/.codegraph/store before indexer.Run — DiscoverAll's walk runs before graphstore.Open creates it on a real first index, and pre-creating it produces a phantom DIR_DOTPREFIX exclusion
- [Phase 10]: fabricatePreCoverageStore forces HasCoverage=false as a direct literal (no bool param) to satisfy the plan's structural verify gate and keep the fabrication unambiguous
- [Phase 10]: Task 2's three backfill/preservation tests were written directly against Task 1's implementation and passed on first run — Task 1's coverageDirty gate and Plan 01's writeGraph range-delete were both already correct, no third write site needed
- [Phase 10]: Open Question 3 decided: /health Coverage verification stays at vitest level (health-page.test.ts / health-view.test.ts) — no Playwright gate added.
- [Phase 10]: Fixed CoverageRows pagination cursor to skip-until-seen-cursor-path per segment (was comparing paths lexically against the store's non-lexical length-prefixed key order). — The store's real iteration order for ExcludedFile/File records is length-prefixed (keys.go appendSegment), not lexical path order; a lexical <= cursor comparison silently broke once paging crossed segments, causing infinite duplicate rows.
- [Phase 10]: Merged colliding threat ids across Plans 01-05 (T-10-01/03/04/05/06/07/08/10/15) into single 10-SECURITY.md rows citing every contributing plan's mitigation — The read_first instructions required deduplicating by id and merging mitigation text rather than dropping or inventing ids
- [Phase 01]: 01-01: Added window.__codegraphFileGraphCy debug seam to GraphCanvas.svelte (Rule 2 deviation) — no existing seam exposed live cytoscape edge/node state needed for the FIX-05 overlap diagnosis.
- [Phase 01]: 01-01: Guava's invalid-endpoints warnings trace to one directory pair whose settled bounding boxes do not overlap, suggesting the warning fires during an earlier render pass before ELK's async write-back lands — recorded for plan 01-09 to investigate.
- [Phase 01]: 01-02: getppid seam made per-instance (unexported Daemon fields getppid/watchdogTicks) per RESEARCH.md's correction of D-13's literal wording, matching onSync/onSyncStart/syncFn/onWatchOpen rather than a new exported Option
- [Phase 01]: 01-02: GH #13's leaked-goroutine half REFUTED (both RunWithRetry spawn sites in soak_test.go already joined via joinDaemonRun; 5x -race iterations zero leak/race) — recorded as absence-of-symptom evidence (RESEARCH A5), not positive falsification, no test changed
- [Phase 01]: 01-02: unrelated pre-existing flake TestDaemonFlushLockRequeueGivesUpPerEpisode surfaced under this session's high local machine load during full-suite -race verification — logged to deferred-items.md, not fixed (out of scope; D-14 forbids widening any internal/daemon timeout constant)
- [Phase 01]: 01-03: CheckRegression's Repo guard closes GH #16 — strict equality per D-16, no normalisation; GREEN commit uses fix(01-03) not feat(01-03) per the plan's own instruction since this closes a missing-guard bug
- [Phase 01]: FIX-06: index --force now classifies prior-store errors three ways (ErrNotFound=silent floor 0, ErrStoreLocked=refuse before RemoveAll, other=warn+rebuild) via graphstore's exported sentinels, closing WINDOWS #36.
- [Phase 01]: GH #20 follow-up 1 (Namespace cache volume, 8x16) closed WON'T-DO on the record: ubuntu-latest is free and 28.6x more stable than Namespace 4x8, matching the issue's own adoption bar; a cache volume cannot reach host-placement variance, the leading unrefuted explanation.
- [Phase 01]: GH #20 follow-up 2's drift discriminator is fully specified in tools/bench/BASELINE.md (ref d4672cf5..., job rebless, -seed 42 -count 120000 -trials 7, ubuntu-latest) but NOT dispatched — dispatching requires pushing a temp branch and triggering CI, both outward-facing actions this session was not authorized to perform.
- [Phase 01]: 01-06: rsvg-convert used for PNG rendering (no new devDependency); 16px legibility human-check deferred to end-of-phase per human_verify_mode, no autonomous geometry change made
- [Phase 01]: FIX-10: per-run PRFILES_$(openssl rand -hex 16) delimiter inline in both pull_request_target workflows, not the researched shared-script extraction — require-issue-link.yml deliberately performs no checkout under pull_request_target; a shared script would need one, trading a narrow injection defect for a broader one
- [Phase 01]: check-workflow-output-delimiter.sh compares $GITHUB_OUTPUT via diff against a file, not bash array iteration — bash 3.2 (macOS default /bin/bash, also env bash's resolution) raises unbound-variable on ${arr[@]} expansion of an empty array under set -euo pipefail
- [Phase 01]: GH #20 baseline drift attributed to FLEET (+46.90% hardware vs +3.15% code, inside DefaultThroughputTolerance); Namespace cache volume follow-up closed won't-do; both recorded in tools/bench/BASELINE.md and GH #20 closed
- [Phase 01]: 01-08: cytoscape teardown fix required TWO closures, not one — deferring GraphCanvas's own cy.destroy() alone was insufficient because cytoscape's Core constructor auto-destroys a prior instance registered on a reused container element (container._cyreg), bypassing this component's own teardown timing entirely. Fixed by giving every mount a dedicated, never-reused cytoscape container element in addition to the deferred-destroy gate. — Live reproduction (instance/generation tracing) proved cy.destroyed() was already true before this component's own destroyNow() ever ran, tracing to cytoscape's own container-reuse auto-destroy path in its Core constructor.
- [Phase 01]: 01-09: guava invalid-endpoints race root-caused to cytoscape's own implicit default construction-time 'grid' layout (not an ELK option), fixed via layout:{name:'null'} on the Core constructor plus edge-hide/reveal defense-in-depth around ELK's real layout
- [Phase 02]: No change to Taskfile.yml (D-01): web:drift already RED against 98cd41dd's exact incident shape on a clean checkout — Live replay confirmed CI's clean-checkout find enumeration already fails on the incident; a find-vs-git-ls-files paired assertion would only test staging hygiene, not close a real gate blind spot
- [Phase 02]: [Phase 02]: GRD-12: requiredStatusChecksPath is a standalone top-level const, not folded into the existing multi-const block, so the plan's literal grep for the const declaration matches exactly.
- [Phase 02]: [Phase 02]: GRD-12: left ci.yml's pre-existing goreleaser-check job name field untouched despite matching the plan's D-07 negative-grep verify gate -- that occurrence predates this plan and is structurally required for GitHub's ruleset job-name matching and for TestRequiredCheckNamesPreserved; fixed only the one true duplication this plan introduced (the script's own header comment).
- [Phase 02]: GRD-10 re-vendor: regenerated directly in the real web/ tree (not a scratch copy), since pnpm's own packageManager self-management resolves the pinned 11.23.0 inside web/ without Corepack — D-09's fallback rule permits a local run whenever cd web && pnpm --version prints 11.23.0
- [Phase 02]: pnpm dlx shadcn-svelte@1.5.1 add itself runs under bare pnpm v12.4.1 (dlx does not inherit the project's packageManager pin the way pnpm install does) — pre-existing Taskfile target behavior, not changed by this plan — The differing-file set produced was byte-for-byte identical to the Corepack-pinned CI run from 2026-09-14, confirming the drift is registry-side, not a toolchain artifact
- [Phase 02]: DOCS-09 census proved zero live "provenance over checksums file" claims via a planted positive control (rule 84d1gfpywd); GH #14 closed on that evidence with no rewrite needed.
- [Phase 02]: docs/RELEASE.md § 2 dropped all raw dependency counts and now credits modelcontextprotocol/go-sdk (D-13); SECURITY.md gained one sentence on the advisory tool-vuln job's scope (D-15). DOCS-08 stays unmarked pending 02-07 (shared-ID gate, #2388).
- [Phase 02]: Fixed pre-existing grpc CVE (GO-2026-6348) via indirect dependency bump v1.82.1->v1.83.2, in a separate commit ahead of the ci.yml wiring commit, discovered because check:gonum had to pass locally before being wired into CI.
- [Phase 02]: WINDOWS #13/#16/#29/#31/#33 closed via `gsd-tools windows fixed <id>` with evidence recorded in 02-07-SUMMARY.md (the verb accepts no note); #20/#21/#34 stay open as record-only deviations since the ledger has no annotate/record-only status.
- [Phase 02]: STATE.md's Pending Todos table (17 stale rows) replaced wholesale with gsd-tools init todos' literal pending_todos_markdown render; the one genuinely open row (bench pinnedAt) filed as a real pending-todo file.
- [Phase 02]: GRD-12 closed: maintainer authorized the prepared gh api PUT to grow protect-main's required contexts to 8; the continuation agent independently re-verified the live state read-only before growing the fixture — Precondition halts must never be satisfied on the orchestrator's word alone; the fixture must follow the live ruleset, never lead it
- [Phase 03]: 03-02: the TDD runtime gate's RED commits were reconciled with D-15's one-feat!-commit decision at plan time — three test(03-02): RED commits (f6bd1ffb, 4215e42f, 4da74784; test files only, Task 1 plus a 4-line renderFullLine placeholder so the unit test compiles) precede the single GREEN feat(cli)!: commit 5d69ee2e; CONTEXT's Claude's-Discretion 'every commit go test-green' clause was clarified to exempt those RED commits (squash-merge collapses them on main) — workflow.tdd_mode=true halts any tdd="true" task without a prior test(NN-PP) commit; the planner's 'observe RED in the working tree, commit only GREEN' could not execute
- [Phase 03]: 03-02/review: rename stubs return ONE error whose text IS the two-line D-06 message (no direct Fprintln, no 'codegraph:' prefix) so cmd/codegraph/main.go's single exit path prints it exactly once (WR-01, fa81672c); pinned end-to-end by test/integration/renamed_stubs_test.go against the real binary — the plan's literal 'two Fprintln + error' printed a duplicated third line that only the compiled binary showed — SilenceErrors on every command makes main.go the one place a returned error is printed; a stub that prints AND returns duplicates
- [Phase 03]: 03-04: Backlog row 999.5 'remove the query/unlock rename stubs' (v0.15.0) written by gsd-tools phase add --id 999.5 (additions only, no version token in the heading, milestone phase filter unchanged) with the feat SHA and the exact files/lines to delete in its Goal value; check tdd-red-evidence returned INVALID_RED/zero_tests_discovered for all three Go RED records (TAP-only parser) — Go RED verified by --- FAIL transcript per the repo's documented precedent — planning-artifacts rule: tool-owned files get value edits in shapes the tool writes; the TDD evidence verb has no go test support (upstream gap)
- [Phase 04]: Phase 4 P01: 29 plain-output goldens frozen (D-16); >=28 floors used throughout since D-16's own verb enumeration counts to 29. install-local/uninstall-local use --target claude (the real agents.TargetID), not the plan's literal claude-code. serve-mcp-stderr runs the real serve --mcp exactly once per test binary via sync.Once, working around internal/mcp's process-global os.Stdin close on session end. — Both TestPlainGolden/TestNoColorNonTTYRegression and TestShortFlagsConsistent demonstrated RED against confirmed mutations (04-MUTATION-LOG.md Families a/b) and reverted byte-clean; no production file touched.
- [Phase 04]: [Phase 4]: 04-02: fang/v2 v2.0.1 declined - fang.Execute's DefaultErrorHandler wraps stderr in a *colorprofile.Writer with no Fd() method, so its own TTY-detection type assertion always fails and the plain non-TTY stderr branch is unreachable; every error renders styled, breaking D-03's exact-once stub contract (TestRenamedStubsPrintExactlyOnce failed under the real wrap). D-01/D-02/D-04 passed in isolation; the four criteria are conjunctive. Help stays hand-rolled per D-14.
- [Phase 04]: 04-03: present/tty.go and present/styles.go doc comments reworded to drop the literal substrings os.Getenv/term.IsTerminal (same meaning) -- the D-03 env-blind grep gate was matching pre-existing prose describing the constraint, not code violating it (third instance of this project's own recurring substring-proxy gate defect).
- [Phase 04]: 04-03: Task 2's literal whole-package verify cannot show zero FAIL lines because of the plan's own pre-announced --color-undocumented RED window (TestEveryRegisteredFlagIsAccountedFor, scheduled for plan 08) -- every other assertion in Task 2's chain was confirmed individually.
- [Phase 4]: RenderNotice styles a multi-line worktree notice line-by-line, preserving trailing-newline structure — Matches query.WorktreeNotice's exact byte shape (possibly multi-line) rather than assuming a single line, so the stripped-styled == plain contract holds for any notice content
- [Phase 4]: explore/node gain a styled branch consuming ExploreDetail/NodeDetail, pinned by a syntax-only markdown contract against the exported query renderers; NODE-02 multi-def budget constants (16/12000/20) duplicated in present, internal/query left at zero diff
- [Phase 4]: [Phase 4] 04-06: printSummary/printSummaryMode split so printSyncSummary resolves colour exactly once per RunE (D-11) instead of double-querying the dark background when reused for a second styled line
- [Phase 4]: [Phase 4] 04-06: uninit.go's codegraphDir sanitized via a package-local sanitizePathForDisplay before styling (CR-01), matching present/status.go's projectPath precedent though not spelled out in the plan text
- [Phase 4]: install.go's per-file action role predicate treats ActionUnchanged/ActionKept/ActionNotFound as no-op (Label), everything else mutating (Warning)
- [Phase 4]: Phase 4 complete: command tree grouped into D-13's four titled cobra.Groups, help hand-rolled via present.RenderHelp (fang declined per 04-02), docs/CLI-REFERENCE.md regenerated through the drift gate, fang verdict recorded in PROJECT.md Key Decisions
- [Phase 5]: [Phase 04] fang/v2 declined: DefaultErrorHandler wraps stderr in *colorprofile.Writer (no Fd()), so its plain non-TTY branch is dead and rename stubs box-render on a pipe; help hand-rolled via present.RenderHelp; verdict committed alone (d722804a) before any renderer
- [Phase 5]: [Phase 04] Colour is resolved ONCE per RunE in internal/cli/colorflag.go (single colorprofile.Detect over a rewritten environ; HasDarkBackground at most once, only when styled AND both fds are TTYs) and downsampled by colorprofile.Writer at the RunE boundary; present stays env-blind and tests assert only our environ rewrite/branch/GroupIDs/plain goldens (D-00: never charm/cobra behaviour or go.mod)
- [Phase 5]: [Phase 04] Phase-end UAT (colour legibility, TERM matrix, help, pager) validated by the orchestrating agent in a Herdr PTY at the maintainer's direction — light palette proven to engage via an OSC 11 background flip, WCAG ≥ AA proxy; residuals recorded in 04-UAT.md
- [Phase 5]: [Phase 05] 05-01: Narrowed antigravityConfigPath() to resolve unified path on a fresh machine (not just once migrated) — a real bug found by the D-03 test oracle
- [Phase 5]: [Phase 05] 05-01: HookFiles hardcodes claude's settings/hooks paths for HooksClaudeJSON and errors loudly (errHookFilesUndeclared) for HooksCodexJSON rather than silently naming nothing
- [Phase 5]: [Phase 05] 05-01: Split RED/GREEN across two commits — capabilities.go landed fully implemented in the RED commit while all eight targets' Capabilities() were zero-value placeholders, so the four named tests fail on assertion, never a build error
- [Phase 05]: manifestRequesters reads BOTH an unreadable and a present-but-nil-Targets manifest as owned by [claude] (D-07 planner amendment) — never as unknown, since Claude was the sole writer before this phase
- [Phase 05]: manifestSchemaVersion bumped 1->2 (costly, flagged for maintainer): a released binary older than 05-02 drops Targets on a schema-2 manifest, self-healed on the next new-binary install
- [Phase 5]: [Phase 05] 05-03: TestSymlinkedSkillDir_ClaudeAndSharedAreOnePackage passed even at RED — 05-02's D-07 legacy-manifest-reads-as-claude rule already produces the correct merged requester set on the read side; the write-side gaps (uninstall, foreign content, dangling link) are what GREEN actually closes.
- [Phase 5]: [Phase 05] 05-03: internal/daemon's TestConvergenceTwoSessions failed once during the full-module run (pre-existing WINDOWS.md #37 flake, unrelated to internal/agents changes) then passed on immediate retry.
- [Phase 5]: [Phase 05] 05-04: TestOwnershipExactIdentity (32-leaf D-13 ownership table, citing 242ec0a by SHA) wires Cursor and opencode onto the shared skill package through installDeclaredSkill/uninstallDeclaredSkill; Family (b1)/(b2) prove the guard fails against both historical shapes of the 242ec0a-class ownership vulnerability. Found the plan's own Task 1 verify script undercounts due to an rg substring-inclusion footgun (uninstallDeclaredSkill contains installDeclaredSkill); documented and worked around with a lookbehind-corrected check rather than altering the design.
- [Phase 5]: [Phase 05] Gemini writes its own .gemini/skills/codegraph dir (D-06 correction (a)); shared .agents/skills alias stays a documented read path only, never written
- [Phase 5]: [Phase 05] Kiro writes only .kiro/skills/codegraph (no shared alias, no AGENTS.md); Antigravity writes only the agy CLI global dir, the [ASSUMED] 2.0/IDE path stays documented-only
- [Phase 5]: 05-06: no Cursor account — Cursor and D-11 recorded not probed; Cursor stays [ASSUMED]; 05-07 takes the D-11 no-change branch
- [Phase 5]: 05-06 2A: opencode duplicate-skill-name WARN accepted as an advisory (copies byte-identical; no guard on the shared write)
- [Phase 5]: 05-06 1A: agy 1.2.6 reads user skills only from ~/.gemini/config/skills/ — Antigravity skill dir moves there in 05-07
- [Phase 5]: 05-06 3A: Antigravity cleanup judged on codegraph-owned paths and recorded hashes; agy runtime state classified, not reverted
- [Phase 5]: 05-07: D-11 recorded not probed, so Cursor keeps writing no instructions file (no-change branch)
- [Phase 5]: 05-07 (1A): Antigravity's only skill dir is ~/.gemini/config/skills/codegraph; the antigravity-cli path is not declared and gets no migration (unreleased)
- [Phase 6]: Phase 6 06-01: guard binary path delivered as one single-quoted token replaced with a POSIX-quoted absolute ExecPath (not text/template); the unrendered dogfood guard alone falls back to PATH
- [Phase 6]: Phase 6 06-01: two bare allowlist lines (codegraph hook, codegraph hook pretooluse), both hidden commands carry --help
- [Phase 6]: Phase 6 06-02: TestCorporaShape's logged fire rates are measured through Qualifies, not counted from the rows' want fields
- [Phase 6]: Phase 6 06-03: Gate.Due Lstat-checks the sentinel dir after every Mkdir, not only on EEXIST, so a dir it just created is held to the same symlink/is-dir/uid test
- [Phase 6]: Phase 6 06-03: a sentinel mtime in the future counts as due and is re-recorded, so a stepped-back clock cannot silence the nudge
- [Phase 6]: Phase 6 06-04: the PreToolUse opt-in is recorded as manifest Files keys hooks/pretooluse-nudge.sh and settings.json#hooks.PreToolUse (no schema bump); Keep refreshes only while either key is present and the manifest is readable
- [Phase 6]: Phase 6 06-04: an Off install reports only artifacts actually removed and drops both keys only when neither removal errored
- [Phase 6]: Phase 6 06-05: the PreToolUse fragment registers one handler per block (three Bash blocks for grep/rg/find), so a hand-edit of any single own handler duplicates rather than being overwritten via its siblings; ownership code unchanged
- [Phase 6]: Phase 6 06-05: --pretool-nudge is read through cobra Changed (not given = Keep); the D-09 note goes to stderr, plain, once, before the per-agent report
- [Phase 6]: Phase 6 06-06: D-18 verdict PASS (C1-C7) in Claude Code 2.1.278; fire rate 9/18, 7/9 true positives
- [Phase 6]: Phase 6 06-06: same-command PreToolUse handlers differing only in if are not deduplicated (validates 06-05 one-handler-per-block); subagents carry the parent session_id
- [Phase 6]: Phase 6 06-07: install/uninstall help describe only what shipped and passed live (D-18 PASS); CLI-REFERENCE.md regenerated only via task docs:cli
- [Phase 6]: Phase 6 06-07: mutation-family count is 19 (a1-a2, b1-b3, c1-c6, d1-d4, e1-e4) after the 06-05 amendment added (e4); each carries RED and revert proof
- [Phase 07]: findTOMLTableRange rewritten as a line scanner (splitTOMLLines/tomlLine/tomlLineState) tracking multi-line-string and bracket-depth state across lines; codegraph's range end backs off past the contiguous blank/comment run before the next header (or EOF), a deliberate change from the pre-existing implementation.
- [Phase 07]: tomlTableConflict is a separate scan from findTOMLTableRange with its own path normalization (tomlNormalizedHeaderPath/tomlKeyTablePath/tomlSplitDottedPath) that unquotes and trims dotted segments, refusing inline/dotted/quoted/spaced/array-of-tables/duplicate/detached-subtable forms of codegraph's own TOML table rather than duplicating a key.
- [Phase 7]: Family (b) mutation tests appended at end of install_test.go rather than interleaved, preserving existing test line-number references
- [Phase 7]: [Phase 07]: 07-03: Task 3 (Family c3 tmux RED/GREEN) not performed by maintainer decision (2026-09-19) — tmux replaced by herdr on the maintainer's machines; local tmux evidence skipped, CI tmux-e2e job is the only remaining real-PTY confirmation, follow-up filed at issue #75; 07-09's post-scope-flip tmux re-run is skipped under the same decision
- [Phase 7]: [Phase 07]: 07-03: bubbles v2 list.populatedView already inserts a row separator — checkboxDelegate/daemonDelegate must render exactly one line and never append their own trailing newline, or every row costs 2 lines against the Height() budget (D-24)
- [Phase 07]: CODEX-01 verdict PASS: all six pass-bar L-lines PASS in an isolated Codex scratch HOME before any codex.go change; real ~/.codex and ~/.agents files unchanged (L7).
- [Phase 07]: D-15/D-16/A2: Codex reads both .codex/skills and $CODEX_HOME/skills; trust-gates project MCP servers and hooks but NOT AGENTS.md or project skills; the -c trust_level override does not grant trust.
- [Phase 07]: A1: both the local D-20 hook command form and the single-quoted absolute global form (with a space in the path) are shell-expanded and executed; hook trust is keyed per hooks.json file+group+handler (D-23 append-last required).
- [Phase 7]: [Phase 07-05]: codexTrustNote's wording follows 07-LIVE-SESSIONS.md's live verdicts exactly -- names the TUI trust prompt and the literal [projects."<root>"] trust_level = "trusted" key (never the -c projects...trust_level override, which CODEX-01 confirmed grants no trust), and explicitly says the codegraph skill and AGENTS.md block are read regardless of trust (D-16=no, A2=no)
- [Phase 7]: [Phase 07-05]: codexSkillDirs declares BOTH D-15 read-only roots (.codex/skills locally, $CODEX_HOME/skills globally) since 07-LIVE-SESSIONS.md recorded both verdicts as yes -- codegraph never writes to either (D-14); DescribePaths never lists them
- [Phase 7]: [Phase 07-05]: Task 3's own plan-authored verify precondition grep (rg -c -F 'installDeclaredSkill(&result, t, loc)') has a substring-collision bug -- it matches inside uninstallDeclaredSkill too, returning 2 instead of 1. Verified the substance with a corrected negative-lookbehind pattern and confirmed the actual perl mutation touches only the intended call site; documented rather than hand-editing the plan or the code to force a false match
- [Phase 07]: [Phase 07-06]: instructionsRequestedElsewhere derives the shared-AGENTS.md requester set from AllTargets() on every Uninstall call — never a stored index — so codex/opencode's shared repo-root AGENTS.md is kept while a sibling still uses it and restored byte-for-byte only once the last sharer is gone
- [Phase 7]: [Phase 07]: 07-07: TestCodexPreToolUseGuard's PWD-based negative control for D-22 needed a redesign -- faking $PWD alone does nothing since bash re-derives it from getcwd() when mismatched; fixed by making the child process's actual OS-level cwd itself wrong for local guard subtests (commits 5c930bb5, 602e30a3)
- [Phase 7]: [Phase 07]: 07-07: capabilities.go's HookFiles switch generalized to wrap errHookFilesUndeclared for ANY unmapped HookMechanism (not just codex-json specifically), now that codex-json itself is a real declared case
- [Phase 7]: [Phase 7]: 07-08: Codex's PreToolUse stickiness evidence is its own exact-identity hooks.json group, read directly via hasOwnHookBlock (D-23) -- unlike Claude's manifest-backed preToolNudgeEvidenced, Codex has no manifest concept for this opt-in, so Keep probes hooks.json unconditionally rather than gating on a manifest record.
- [Phase 7]: [Phase 7]: 07-08: assertOwnEntriesGoneAfterUninstall's new Codex branch was moved out of that shared helper into runOwnershipLeaf after it broke TestOwnershipSharedInstructions -- that test calls the shared helper for Codex without ever planting the foreign ^Bash$ group this plan's guard checks for, so the assertion belongs only at the one call site that actually plants it.
- [Phase 7]: CODEX-06 verdict: PASS — a fresh Codex session in an indexed repo reaches for codegraph unprompted (skill listed, codegraph CLI run), entry shown at both project and global scopes
- [Phase 7]: CODEX-05 live verdict: PASS — nudge fires once then cools down (60s+ per session/subagent key), stays silent un-indexed, no hook error attributable to codegraph's hook
- [Phase 7]: A4: Codex subagent PreToolUse stdin carries agent_id (UUIDv7) + agent_type alongside the parent's session_id; the main thread carries neither — D-21 session_id+agent_id (else main) keying holds unchanged on Codex
- [Phase 7]: D-23: position-keyed hook trust re-flags a byte-identical foreign hook when its array index shifts; codegraph appends its group last so its own removal never shifts a foreign group
- [Phase 7]: FIX-03's local real-PTY tmux re-run explicitly NOT performed (maintainer decision 2026-09-19, carried from 07-03, issue #75) — the 07-03 model-level footprint guard is the local FIX-03 evidence at this HEAD
- [Phase 7]: [Phase 07] 07-10: docs/AGENT-CAPABILITIES.md published (16 rows, AllTargets() x [global,local]) with a Go drift test (TestCapabilityDoc_MirrorsCapabilities/VerificationColumn) that recomputes every code-derived cell from Capabilities() and never lets the doc overclaim [ASSUMED] rows as verified
- [Phase 7]: [Phase 07] 07-10: Claude's global row is [ASSUMED] (code.claude.com/docs/en/mcp, fetched 2026-09-19) after confirming no 0[5-7]-LIVE-SESSIONS.md file ran a dedicated Claude-global install+read session — only Claude local scope was live-verified (06-LIVE-SESSIONS.md)

### Pending Todos

- [2026-08-14] [bench] tools/bench/runner pinnedAt() validates a checkout by git rev-parse HEAD alone — the HEAD-only anti… — [todo file](.planning/todos/pending/2026-08-14-bench-pinnedat-validates-a-checkout-by-git-rev-parse-head-alone.md)

### Blockers/Concerns

Nothing blocks v0.12.0. Carried forward from prior milestones:

- **CR-01 is no longer just carried — it is scoped.** `internal/mcp/server.go:225-349`'s `pendingWriter` "pending response" counter increments only on accepted client requests but decrements on every stdout `Write()`, including server-initiated notifications (`notifications/tools/list_changed`, `notifications/subscriptions/acknowledged`) that SPEC-09 routes through the identical writer. A notification landing between a request's acceptance and its response being written can zero the counter early, causing premature EOF propagation and silent loss of the still-in-flight response — confirmed reachable, not theoretical. Predates v0.10.0 (introduced in `13f2875`). Full trace and proposed fix in `.planning/phases/05-mcp-resources-capability-claims-drift-guard/05-REVIEW.md` (archived under `milestones/`). Now `FIX-01`, Phase 1.
- **Backlog bookkeeping inconsistency (needs a maintainer call).** `999.3` and `999.6` were both promoted into v0.3.0, but all `999.x` Backlog entries were preserved verbatim in `ROADMAP.md` by explicit instruction. Decide whether the promoted entries should be struck or annotated; nothing was removed pending that call. (`999.5` has since been consumed by v0.5.0; `999.2` and `999.4` remain.)
- **Client-side `tools/list` caching bugs are a known confound.** Real, primary-source GitHub issues exist against Claude Code itself (anthropics/claude-code #41123, #40025, #50515; claude-ai-mcp #45).
- **Advisory, unregistered surfaces** from the v1.0 Phase 10 security audit: the four `pull_request_target` workflows and the darwin canary have no threat-register entry, having landed after their registers were authored.
- **`GOOS=windows go vet`** on `internal/daemon` / `internal/graphstore` fails (`undefined: tree_sitter.Node` in `goextract/routes`) — CGo grammar bindings excluded under windows build constraints; pre-existing. Native Windows support was dropped in `v0.4.0` (WSL2 only).
- **GO-2026-5932 is a real, ACCEPTED, unmitigated exposure in release tooling.** goreleaser's binary reaches `golang.org/x/crypto/openpgp` (110 vulnerable symbols) via pipe/ko → google/ko → sigstore/cosign/oci → sigstore/rekor/pkg/pki/pgp. Upstream is unmaintained (Fixed in: N/A). The advisory `tool-vuln` job surfaces it — reported, not resolved. **Relevant to `BLD-06`:** `pnpm audit` adds a second, disjoint scanner covering the JS tree neither `govulncheck` nor Syft can see; `SECURITY.md` must state both scanners' actual scope rather than implying one covers everything.
- **Daemon extreme-load tail (ACCEPTED, not a gap).** 52/52 real `ci.yml` runs show no daemon failure on the actual runner class; CI load was ruled the governing standard for MAINT-02 (maintainer, 2026-08-06).
- **Wire-oracle `toolslist-repeat` ordering flake.** `TestFrozenTranscriptsMatch/toolslist-repeat` freezes JSON-RPC response *arrival* order, which the protocol does not guarantee and go-sdk's async dispatch does not provide.
- **Tooling gaps (not blocking work, and not hand-edited per the planning-artifacts rule):** `gsd-tools query state.advance-plan` failed with "Cannot parse Current Plan or Total Plans in Phase from STATE.md" when Current Position read "Plan: Not started". `gsd-tools query state.sync` counts a SUMMARY with `status: halted` as a completed plan, and MUTATES when invoked with no args — it has no dry-run probe mode. `uat-predicate.cjs` accepts only `pass`/`passed` per test item, so a `skipped`-with-reason deferred follow-up blocks `phase uat-passed` even though the verify-work template calls that state `complete` and #1921 says a deferred follow-up must never block (P8 test 3). `gsd-verifier` declared every file in the phase dir — including `08-UAT.md` and `08-VALIDATION.md`, which verify-work and validate-phase WRITE — in `covered_files`, so the verification went `stale` by construction the moment its own downstream hooks ran; resolved by dropping those two outputs from the set and recomputing via `verification.fingerprint` (the verifier contract at `gsd-verifier.md:673` is PLAN/SUMMARY + requirements + impl files, not workflow outputs).
- ⚠️ [Phase 10] Phases 7 and 8 read `verification_status: stale` since Phase 9 completed: their `covered_files` include `.planning/REQUIREMENTS.md` (verifier contract #4155 — "mapped requirement"), which every later `phase.complete` rewrites. ROADMAP still shows them `[x]`; the tool-sanctioned repair is `/gsd-verify-work 07` / `08` re-verification. Will surface at the milestone audit.
- ⚠️ [Phase 11] `web/scripts/check-no-force-layout.mjs` proves only that no forbidden layout name appears as a string literal at the `name:` option position or as a `cytoscape-<x>` import/dependency specifier; a string-built or variable layout name is not detected. `GraphCanvas.svelte:374` spreads `LAYOUT_OPTIONS` (declared with literal `name: 'elk'`) and is reported as the one advisory `unresolvedLayoutNames` entry — non-fatal by design (WR-03).
- ⚠️ [Phase 11] `gonum.org/v1/gonum` package legitimacy is `[ASSUMED]` (long-lived, already transitive via sigstore in `go.sum`; `package-legitimacy check` has no Go ecosystem support) — recorded in `11-SECURITY.md`, not a gap.
- ⚠️ [Phase 11] Phases 7, 8, 9 AND 10 all read `verification_status: stale` after Phase 11's `phase.complete` by the #4155 mechanism (each `covered_files` list includes `.planning/REQUIREMENTS.md`, which every later `phase.complete` rewrites); only Phase 11's report was written without `REQUIREMENTS.md` in `covered_files` (its plans/summaries + implementation files only) and stays `passed`. Repair for 7–10 remains `/gsd-verify-work <phase>`; surfaces at the milestone audit.
- ⚠️ [Phase 12] The DOCS-06 guard matches `--name` as a whole-document substring of the generated reference (anchored on the flag's own `--` prefix and a trailing delimiter); a future flag whose name is a strict prefix of another's is the one shape that could read as documented when it is not — recorded in 12-SECURITY.md as accepted residual, re-checked by the guard's own counts on every run.
- ⚠️ [Phase 12] DOCS-07 has no ongoing gate by decision (D-12): nothing stops a future edit from re-introducing a copy-pasteable tap-wide `brew trust` instruction in `docs/RELEASE.md`; review of that file's diffs is the control.
- ⚠️ [Phase 12] `DisableAutoGenTag = true` is set on the tree `NewRootCmd()` returns; `codegraph man` builds its own tree in `man.go` and still emits cobra's auto-gen date line — unchanged behaviour, noted so nobody expects the man pages to be byte-stable across days.
- ⚠️ [Phase 9] Cursor and JetBrains editor-link URI templates are community-sourced, never officially documented (09-RESEARCH.md A1/A2) — shipped tagged `[ASSUMED]` in `editorpresets.go` with a visible note in the picker; WINDOWS.md #35 stays open until someone clicks through on a real Cursor/JetBrains install.
- ⚠️ [Phase 9] Safari/WebKit and Firefox are UNVERIFIED for the gutter's async-rpc-then-`location.assign` sequence (transient user-activation window); the committed live gate is chromium-only and the header link is a plain resolved `<a href>` by design, so the risk is confined to gutter clicks. Recorded in `09-SECURITY.md` T-09-09 notes.
- ⚠️ [Phase 1] `tools/bench/BASELINE.md` (lines ~439/477) links a specific CI run as the FIX-11 in-flight discriminator; GitHub prunes run logs/artifacts under its retention policy, so the link will rot — the conclusion is recorded inline, the link is convenience only (01-REVIEW.md IN-01, Info, left open by scope).
- ⚠️ [Phase 1] Plan-level `<verify>` gates for web plans ran `pnpm check` but never `pnpm vitest run`; the FIX-05 reveal broke seven test fakes and only the deep code review caught it (exit 1 with 11 unhandled errors behind a 585-passed count). Phase 2's guard work should add the vitest exit-code assertion to the plan template or the Taskfile gate.
- [Phase 2] WINDOWS #20, #21, #34 are open by decision, not pending work: #20 and #21 are Phase-2 (v0.12.0) deviations by design (no `svelte.config.js` in this SvelteKit toolchain; TypeScript pinned at 6.0.3 by the scaffold), #34 is a TTY-06 mutation-log finding whose assertion measures post-settle stability by design; the ledger has no annotate/record-only verb or status, so they stay `open` rather than being waived (a waiver reads as a deferred defect) or fixed (nothing was fixed). Plan 02-07, GRD-14.
- Tooling gaps (GRD-14/DOCS-11, 02-07): (a) `gsd-tools windows` has no annotate/record-only verb or status; (b) `windows fixed <id>` accepts no note (`broken-windows.cjs` `markFixed`/`cmdWindowsMarkFixed`, lines 286-293/1068-1088: one positional, zero flags), so verification evidence can only live in plan SUMMARYs; (c) no CLI verb creates a pending-todo file (the add-todo workflow is agent-authored, not a `gsd-tools` command); (d) STATE.md's Pending Todos renderer (`init.cjs` `renderPendingTodosMarkdown`, line 2024) emits bullets, so the previous hand-authored table was not a tool shape and was replaced by the rendered body in 02-07. Filing on open-gsd/gsd-core is the maintainer's call; the drafted issue body is in 02-07-SUMMARY.md.
- ⚠️ [Phase 3] WINDOWS #37: internal/daemon is load-flaky under cross-package go test (TestRunWatchdogCancelsRunOnSimulatedReparent 250s 'Run did not return after a simulated reparent', TestConvergenceTwoSessions 130s) — reproduced on the PRE-fold commit 5bc10ed8 (1/2 runs) and post-fold (1/3), passes alone every time; not a Phase 3 regression (4 message lines in lock.go) but it contradicts #12's 'fixed' (FIX-07 removed the ticker race, not the budget miss under load). Same session: tests/browse-page.test.ts hit its 15s vitest timeout in 2 of 3 full web runs at machine load ~35 and passed alone 2/2 with web/ untouched — load-induced, unrecorded.
- ⚠️ [Phase 3] Phase 1 reads verification_status: stale after Phase 3's phase.complete and the VERB-01..08 checkbox bookkeeping (#4155 mechanism — its covered_files list includes .planning/REQUIREMENTS.md; Phase 2 still reads passed); Phase 3's own report was re-stamped with 'gsd-tools verification fingerprint' over the verifier's unchanged 28-file list after confirming REQUIREMENTS.md was the only covered file that changed. IN-01 (search has no Long describing --full) left as an advisory docs: follow-up by maintainer choice.
- [Phase 4] Advisory (not a defect): after CR-02, lipgloss's own tab→4-space conversion (maybeConvertTabs) still applies on the styled explore/node path, and node_test.go models it with a styledTabWidth=4 constant mirroring a lipgloss internal (D-00 tension). One-line fix available at v2.0.5: Style.TabWidth(lipgloss.NoTabConversion) on the palette styles, then drop the constant so stripped-styled == plain byte-exact on tab-indented source.
- [Phase 4] Not verified: the D-11 ~2 s OSC-11 timeout on a non-answering terminal (tmux without allow-passthrough / SSH) — tmux is not installed locally and SSH was not attempted; every measured run in Herdr's terminal answered OSC 11 in ≤0.16 s. IN-01 (status 'Project:' value unstyled) left open as Info.
- [Phase 4] Phases 1–3 read verification_status: stale after Phase 4 — a GENUINE signal, not #4155 bookkeeping: their covered_files name internal/cli/{root,search,daemon,index,renamed}.go and docs/CLI-REFERENCE.md, all legitimately modified by the glow-up (plus REQUIREMENTS/ROADMAP via phase.complete). Phase 4's regression gate (52/52 pkgs, wire oracle, real-binary renamed_stubs test) covered the risk for this run; the digest was NOT re-stamped. Repair = /gsd-verify-work 02 / 03 (and 01) — surfaces at the milestone audit.
- [Phase 5] One unidentified internal/cli test failure after 05-05 (2026-09-18): the package run took 53.6 s vs a normal ~18 s — the machine was under other load (likely the 05-05 executor's own full-suite run still finishing) — and the failing test's name was not captured (output piped through tail). Six reruns passed (2 standalone, 4 with internal/agents in parallel). Treat as a possible load-sensitive flake in internal/cli; next occurrence: capture the full output, then decide whether it is a WINDOWS.md row.
- [Phase 5] The verify:post hooks (validate-phase, secure-phase) did NOT run at the Phase 4 transition: 04-VALIDATION.md is still status: draft / nyquist_compliant: false and there is no 04-SECURITY.md, although workflow.nyquist_validation and workflow.security_enforcement are both active. Run /gsd-validate-phase 4 and /gsd-secure-phase 4 before the milestone audit (audit-milestone §5.5 reports NOT-VALIDATED).
- [Phase 5] Live-verification breadth deferred by maintainer decision (05-UAT.md, 2 deferred follow-ups): Cursor (no account — D-11 verdict 'not probed', so Cursor keeps writing no instructions file) and Gemini CLI/Kiro (not installed — D-10). Their rows stay [ASSUMED] from primary docs; AGENT-14 (Phase 7) must publish them as [ASSUMED].
- [Phase 5] REQUIREMENTS.md AGENT-07 wording ('skill package via .agents/skills/ and the instructions block in AGENTS.md') is stale: live agy 1.2.6 reads user skills only from ~/.gemini/config/skills/ (maintainer decision 1A, shipped in 05-07) and Antigravity's instructions arrive via ~/.gemini/GEMINI.md (D-06(c)). Maintainer to reword at the milestone audit; the code is correct.
- [Phase 5] Advisories carried forward: (1) install --yes discards an explicit --target (todo 2026-09-18-install-yes-discards-explicit-target.md); (2) uninstall leaves an empty parent skills dir it created (removing the parent unconditionally is wrong — 05-07 Family (c-i)); (3) review Info IN-01 (dead caps.MCPConfig nil-check, printconfigstyle.go) and IN-02 (ActionKeptForeign renders with the default role, untested); (4) gsd-tools frontmatter set re-serializes covered_files with a blank line after the key (harmless to the parser, observed twice).
- [Phase 6] AR-06-08: the registered hook commands (SessionStart and PreToolUse) are unquoted shell-form paths — a project dir or $HOME containing whitespace splits them before the guard runs, giving a non-blocking 'hook error' notice (never a block, never injection). The exec-form/quoting fix changes the owned command identity (duplicates on upgrade), so it is a deferred decision (06-CONTEXT Deferred Ideas); CODEX-05 should decide its own form deliberately.
- [Phase 6] codegraph upgrade cannot reach a Claude location whose skill dir is fully foreign/unmanifested: its PreToolUse guard keeps the old ExecPath with NO user-visible signal (test-pinned: TestRefreshInstalledSkills_ForeignSkillDirLocationIsAcceptedLimitation; recovery = re-run codegraph install there). Open proposal (06-REVIEW-FIX WR-03): a one-line upgrade note naming such a location.
- [Phase 6] Tooling gaps (not blocking, not hand-edited): (a) plan-gate commands that pick a base via git log --grep='^test\\(NN-PP\\): ' | tail -1 resolve to EARLIER milestones' same-numbered plans (hit in 06-04, 06-05, 06-07 — verified against the phase base instead); (b) stale .git/gsd-plan-head-before-NN-PP markers from earlier milestones had to be removed; (c) state.update-progress warns on every plan that STATE.md has no 'Progress:' body line; (d) frontmatter set re-serializes covered_files with a blank line after the key (parser tolerates it).
- [Phase 6] Phase 5 now reads verification_status: stale — a GENUINE signal (like Phase 4 → Phases 1–3): Phase 6 legitimately modified files in Phase 5's covered_files (internal/agents/claude.go, capabilities.go, skillshared.go, manifest.go, types.go, shared.go, ownership/capabilities tests, internal/cli/install.go, uninstall.go, docs/CLI-REFERENCE.md). Re-verify Phases 1–5 at the milestone audit (/gsd-verify-work), together with Phase 4's missing validate-phase/secure-phase runs.
- [Phase 7] Released codegraph binaries carry a Codex TOML data-loss bug: findTOMLTableRange ends a table only at a column-0 '[', so an indented [mcp_servers.codegraph] swallows every sibling table up to the next column-0 header. Do NOT run 'codegraph install' or 'uninstall' with --target codex or --target all on a machine whose ~/.codex/config.toml indents headers until the Phase 7 fix (07-CONTEXT D-07/D-08) ships; no patch release (maintainer decision B2, 2026-09-19)
- [Phase 7] FIX-03 is marked complete on the model-level footprint test (07-03 Families c1/c2, RED on the pre-fix delegate, GREEN at HEAD), but its requirement text also asks for the tmux harness assertion AFTER the CODEX-02 scope flip. That run was skipped by maintainer decision 2026-09-19 (tmux retired, replaced by herdr; GH issue #75) and has NOT run in CI either (branch unpushed). The re-anchored TTY-05 assertion compiles (go vet -tags tmux) but is unexecuted post-flip; the CI tmux-e2e job on the eventual PR is the outstanding evidence

### Quick Tasks Completed

| # | Description | Date | Commit | Directory |
|---|-------------|------|--------|-----------|
| 260807-gho | Drop native Windows support — WSL2 only | 2026-08-07 | 085b7a3 | [260807-gho-drop-native-windows-support-wsl2-only](./quick/260807-gho-drop-native-windows-support-wsl2-only/) |
| 260811-s5o | Install cosign in post-release-verify's self-upgrade job (v0.9.0 self-upgrade proof failed closed on a missing installer) | 2026-08-11 | 6135785 | [260811-s5o-add-sha-pinned-sigstore-cosign-installer](./quick/260811-s5o-add-sha-pinned-sigstore-cosign-installer/) |
| 260913-pkp | Fix graphstore archtest to fail closed on per-package go/packages load errors (CR-01 sibling of a90b5457); todo 2026-09-08 resolved | 2026-09-13 | 2b553b62 | [260913-pkp-fix-graphstore-archtest-to-fail-closed-o](./quick/260913-pkp-fix-graphstore-archtest-to-fail-closed-o/) |

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

Acknowledged at the **v0.12.0 close (2026-09-07)** — 1 newly acknowledged, 8 carried forward from the v0.11.0 close.

| Category | Item | Status | Deferred At | Milestone |
|----------|------|--------|-------------|-----------|
| todos | 2026-09-07-internal-query-dependency-direction-has-no-persisted-archtest.md | (presence-only) | 2026-09-07 | v0.12.0 |

**Why this one is deferred rather than fixed:** T-01-18 protects the Engine seam's *direction* —
`internal/query` must never import the wire layer, or the "third consumer, never a second
implementation" premise inverts. The retroactive Phase 1 security audit found the invariant
**holds today** (re-ran `go list -deps ./internal/query` live: zero `connectrpc.com/connect`,
zero `internal/uiproto`, positive-controlled against `google.golang.org/protobuf`, which does
appear via `internal/schema`) — but its mitigation was a one-shot check written into two plans'
`<verification>` blocks and never committed as a test. It is an **unguarded invariant**, not an
open vulnerability: medium severity, below the `high` block_on threshold.

**Process note, recorded because it nearly hid this item.** When the todo was first filed during
the milestone audit, its frontmatter was copied from a sibling todo — including that sibling's
`audit_acknowledged` block. That silently suppressed the finding from its own milestone's close
before anyone decided to defer it. `complete-milestone.md` is explicit that the
`audit-open acknowledge` writer is the only path permitted to set that marker and that the
workflow never hand-authors it; the block was removed, the item resurfaced as `1 item requires
decisions before close`, and the deferral above was then made deliberately through the CLI
writer. **Copying a todo's frontmatter as a template copies its suppression.**

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

Last session: 2026-09-19T21:56:16.723Z
Stopped at: Completed 07-10-PLAN.md
  CARRY-OVER (v0.14.0):

    - **`branching_strategy: milestone`** — this milestone lives on `gsd/v0.14.0-milestone`; init computes `gsd/v0.14.0-polish-agent-reach` but the phase-1 work is on the former, so stay on it.
    - **Go toolchain:** ambient Homebrew `go` is 1.27.1; `go.mod` pins 1.26.6 and `cockroachdb/swiss` is `!go1.27` — every local `go build`/`go test` gate needs `GOTOOLCHAIN=go1.26.6` (CI is unaffected: `go-version-file: go.mod`).
    - **Web gates:** `pnpm vitest run` must assert exit 0 AND zero "unhandled errors" lines; a pass count alone is negative-only. Piped `svelte-check` is MACHINE format (uppercase ERRORS).
    - **WINDOWS.md** is tool-owned — close rows with `gsd-tools windows fixed <id>`, never by hand; the phase-1 rows (#12/#26/#28/#30/#36) were closed at 900d64cc and the verifier treats "in the ledger" as part of the phase goal.
    - **No `v0.14.0` git tag** — release-please owns tagging (D-06R).
    - **`.planning/` and `CHANGELOG.md` stay tool-owned** — no invented headings; only the single active-milestone heading under `## Phases`.

## Operator Next Steps

- v0.13.0 SHIPPED 2026-09-13 as a `verified_closeout` — 6/6 phases canonical `passed` after re-verification, 26/26 requirements, both pre-close open artefacts resolved (tty03 debug session; graphstore archtest vacuity fixed as quick task `260913-pkp`); archived under `milestones/v0.13.0-*`
- Push the branch (~180 commits ahead of origin) and open the phases 9–12 PR with a `feat:` title and `Resolves #N`; CI will run `docs:cli:drift` and the tmux `expected=6` gate at HEAD for the first time
- Repository-settings action still open (no agent can do it): add the required-status-check context `tmux e2e (real-pty harness, TTY-01..TTY-07)` to ruleset 20157557, then add the same string to `requiredCheckNames` in `internal/upgrade/taskfile_shape_test.go`
- Cheap follow-ups: wire `task check:gonum` and `task check:no-force-layout` into `ci.yml`; run `/gsd-validate-phase` against the archived 09–12 VALIDATION files if a `validated` record is wanted; reconcile the Pending Todos table; inspect and remove the stale `.claude/worktrees/agent-aebfa7de95041ec86` worktree
- Start the next milestone with /gsd-new-milestone (Phase numbering continues from 13)
