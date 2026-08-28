---
phase: 3
slug: browse-inspect-navigation
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-28
---

# Phase 3 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Seeded from `03-RESEARCH.md` § Validation Architecture (lines 547-583).

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework (Go)** | stdlib `testing`, run via `task test:unit` (`go test` over every package except `internal/daemon`) — `Taskfile.yml:116-131` |
| **Framework (JS)** | **vitest + jsdom + @testing-library/svelte** — DECIDED 2026-08-28 (maintainer). None installed today: zero `*.test.*`/`*.spec.*` files under `web/`, no test deps in `web/package.json`. Wave 0 installs and configures. |
| **Config file (Go)** | none (stdlib) |
| **Config file (JS)** | none yet — Wave 0 adds a `test:` block to the EXISTING `web/vite.config.ts`. A separate `web/vitest.config.ts` is deliberately NOT used: `Taskfile.yml`'s `web_source_files()` enumerates `web/vite.config.ts` and would not hash a separate file, so the BLD-03 source digest would not see a change to the test configuration. |
| **Quick run command** | `go test ./internal/uiserver/... ./internal/query/... ./internal/gitmeta/... ./web/...` and `cd web && pnpm test` |
| **Full suite command** | `task test:unit` and `task web:test` |
| **Estimated runtime** | quick ~15s (scoped Go packages); JS and full suite ~TBD at first green run |

---

## Sampling Rate

- **After every task commit:** Run the touched package's `go test ./internal/<pkg>/...`
- **After every plan wave:** Run `task test:unit` (full Go suite) and `task web:test` (JS suite, which prints its observed executed-test count before judging it), plus manual browser UAT for client-only surfaces
- **Before `/gsd-verify-work`:** `task test:unit`, `task web:test`, `task proto:drift`, `task web:build:verify` and `task web:drift` all green
- **Max feedback latency:** 60 seconds for the quick run

---

## Per-Task Verification Map

Task IDs use `T-{phase}-{plan}-{task}`, matching the task order inside each PLAN.md.
Reconciled with the real planner-assigned tasks on 2026-08-28.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| T-03-01-01 | 03-01 | 1 | BRW-06 | T-03-SC | Blocking-human package-legitimacy gate for all five new npm packages before any install runs; never auto-approvable | Human checkpoint | — | N/A | ⬜ pending |
| T-03-01-02 | 03-01 | 1 | BRW-06 | T-03-10, T-03-11 | vitest + jsdom + @testing-library installed and configured; a deliberately-failing assertion observed RED before any green is trusted; `allowBuilds` outcome recorded, not assumed | JS harness bring-up | `cd web && pnpm test` | ❌ W0 | ⬜ pending |
| T-03-01-03 | 03-01 | 1 | BRW-06 | T-03-12 | `task web:test` prints its observed executed-test count BEFORE comparing to a floor; demonstrated RED against an empty include glob | Taskfile guard | `task web:test` | ❌ W0 | ⬜ pending |
| T-03-02-01 | 03-02 | 1 | SRV-05 | T-03-01 | `..` escape, absolute path, empty path and post-symlink escape each refused at the RPC boundary, **and** a legitimate in-repo path returns real source in the same test (positive control, rule `84d1gfpywd`) | Go integration (Connect handler, real client) | `go test ./internal/uiserver/ -run TestGetNodeDetailPathConfinementAtRPCBoundary -count=1 -v` | ❌ W0 | ⬜ pending |
| T-03-02-02 | 03-02 | 1 | SRV-05 | T-03-06 | No confinement refusal message discloses the host checkout path; paired with a positive containment assertion so the absence check is proven to inspect a real message | Go integration | `go test ./internal/uiserver/ -run TestGetNodeDetailConfinementRefusalDoesNotLeakHostPath -count=1 -v` | ❌ W0 | ⬜ pending |
| T-03-03-01 | 03-03 | 1 | TODO-MCP-01 | T-03-16 | Wire-oracle ordering flake reproduced under contention with a timestamped arrival sequence; verdict recorded before any change | Go integration + evidence capture | `go test ./test/wireoracle/ -run TestFrozenTranscriptsMatch/toolslist-repeat -count=1 -v` | ✅ | ⬜ pending |
| T-03-03-02 | 03-03 | 1 | TODO-MCP-01 | T-03-14 | Maintainer selects R1 / R2 / NR from the evidence; blocking-human so auto-advance cannot decide it | Human decision | — | N/A | ⬜ pending |
| T-03-03-03 | 03-03 | 1 | TODO-MCP-01 | T-03-15 | Chosen resolution proven RED first; planted content mutation still fails the oracle, proving canonicalization did not blind it | Go integration | `task test:wireoracle` | ⬜ | ⬜ pending |
| T-03-04-01 | 03-04 | 2 | BRW-02, BRW-06, NAV-01, NAV-04 | T-03-02 | TRACER: URL -> RPC -> error classification -> highlight -> DOM, one path end to end; exactly one unescaped-HTML site in `web/src` | JS component (jsdom + @testing-library/svelte) | `cd web && pnpm test` (names `browse-tracer`) | ❌ W0 | ⬜ pending |
| T-03-04-02 | 03-04 | 2 | NAV-01, NAV-04 | — | URL round-trip idempotent and deterministically ordered; unknown params ignored not rejected (D-12); non-integer numerics treated as absent, not coerced; Connect error kinds mutually distinct | JS unit (pure TS, no DOM) | `cd web && pnpm test` (names `browse-url`, `rpc-errors`) | ❌ W0 | ⬜ pending |
| T-03-04-03 | 03-04 | 2 | BRW-06 | — | Highlighter coverage (registered modules + declared aliases) set-equals `indexer.RegisteredLanguageIDs()` — no missing, no extra; a planted fixture proves the comparison discriminates in both directions | Go unit, set-equality over the real TS file | `go test ./web/ -run TestHighlight -count=1 -v` | ❌ W0 | ⬜ pending |
| T-03-05-01 | 03-05 | 2 | BRW-09 | T-03-24 | One-way wire shape frozen by a human before any generated code exists | Human decision (blocking-human) | — | N/A | ⬜ pending |
| T-03-05-02 | 03-05 | 2 | BRW-09 | T-03-03, T-03-20, T-03-22 | Remote URL parsed across all real transport shapes; exact `github.com` host equality; credentials stripped (asserted against a populated result); no network call | Go unit | `go test ./internal/gitmeta/ -run Permalink -count=1 -v` | ❌ W0 | ⬜ pending |
| T-03-05-03 | 03-05 | 2 | BRW-09, SRV-05 | T-03-01b, T-03-21 | `GetPermalink` correct across pushed / unpushed / no-remote / non-GitHub / absent-SHA; path percent-encoded; escaping paths refused by the reused confinement gate with a passing control | Go integration (Connect handler, real client) | `go test ./internal/uiserver/ -run Permalink -count=1 -v && task proto:drift` | ❌ W0 | ⬜ pending |
| T-03-06-01 | 03-06 | 3 | BRW-01 | T-03-07, T-03-SC-b | Every vendored component file and every npm package the registry CLI added is enumerated and human-approved before code is written against it | Human checkpoint (blocking-human) | — | N/A | ⬜ pending |
| T-03-06-02 | 03-06 | 3 | BRW-01, BRW-08 | T-03-25 | Debounce, two-character minimum, mandatory abort AND a request-identity guard; a late response cannot overwrite a newer one (asserted by value); explore never rides a keystroke | JS unit (fake timers, manual promises) | `cd web && pnpm test` (names `search.test`) | ❌ W0 | ⬜ pending |
| T-03-06-03 | 03-06 | 3 | BRW-01, BRW-08, NAV-03 | T-03-26 | Two live sections in server order plus a third explore section; empty and failed render differently; `/`, the command chord and Escape behave, with `/` suppressed inside inputs | JS component (jsdom) | `cd web && pnpm test` (names `search-panel`) | ❌ W0 | ⬜ pending |
| T-03-07-01 | 03-07 | 4 | NAV-01, NAV-02 | T-03-28 | One URL writer; navigate pushes and refine replaces (asserted as differing booleans); scroll and focus preserved; no shallow-routing history export imported anywhere | JS unit (spy for the injected navigator) | `cd web && pnpm test` (names `browse-nav`) | ❌ W0 | ⬜ pending |
| T-03-07-02 | 03-07 | 4 | BRW-02, NAV-01 | T-03-27 | Response mode read from its own discriminator, not inferred from emptiness; depth and limit reach the server exactly as the URL states them | JS unit | `cd web && pnpm test` (names `browse-state`) | ❌ W0 | ⬜ pending |
| T-03-07-03 | 03-07 | 4 | BRW-02, BRW-03, NAV-02 | T-03-17b | Callers, callees and blast radius visible together; click-through carries the navigate intent, depth change the refine intent; back/forward walk proven manually with recorded URLs | JS component + manual UAT | `cd web && pnpm test` (names `neighbors-panel`) | ❌ W0 | ⬜ pending |
| T-03-08-01 | 03-08 | 5 | BRW-04 | T-03-02b | Identifier matching is exact — no case folding, no normalization, no prefix; each negative case paired with a positive exact match; decoration builds no markup strings | JS unit + DOM | `cd web && pnpm test` (names `call-targets`) | ❌ W0 | ⬜ pending |
| T-03-08-02 | 03-08 | 5 | BRW-07, BRW-09 | T-03-29, T-03-31 | Truncation notice present/absent proven both ways; three permalink availability states pairwise different; empty-value copy affordance absent, paired with a present case | JS component (jsdom, stubbed clipboard) | `cd web && pnpm test` (names `source-pane`) | ❌ W0 | ⬜ pending |
| T-03-08-03 | 03-08 | 5 | BRW-05 | — | Picker lists every counted candidate, marks ungathered ones from the wire's own flag (proven against an empty-calls gathered candidate), auto-selects nothing, preserves server order | JS component (jsdom) | `cd web && pnpm test` (names `definition-picker`) | ❌ W0 | ⬜ pending |
| T-03-09-01 | 03-09 | 6 | NAV-04 | T-03-34 | No-index and indexing verdicts distinct despite both having `initialized` false; no polling, proven by an unchanged call count across advanced time paired with an increase after navigation | JS unit (fake timers) | `cd web && pnpm test` (names `status.test`) | ❌ W0 | ⬜ pending |
| T-03-09-02 | 03-09 | 6 | NAV-04 | T-03-33 | Each of no-index / stale / not-found renders its own named state, pairwise different; stale banner and in-view not-found coexist; the two source-absent messages differ by stale flag | JS component (jsdom) | `cd web && pnpm test` (names `degrade-states`) | ❌ W0 | ⬜ pending |
| T-03-09-03 | 03-09 | 6 | NAV-04 | T-03-32 | Committed bundle rebuilt; both drift digests recomputed and their observed counts recorded; no floor adjusted | Taskfile guards | `task web:build:verify && task web:drift && task web:test && task proto:drift && task test:unit` | ✅ | ⬜ pending |
| T-03-10-01 | 03-10 | 7 | TODO-CI-01 | T-03-SC-c | Linter pinned in the linters-only tool modfile without losing actionlint to an MVS bid; `.golangci.yml` states its own enabled/rejected set | Taskfile + tool modfile | `task lint:actions` | ⬜ | ⬜ pending |
| T-03-10-02 | 03-10 | 7 | TODO-CI-01 | T-03-38 | First-run backlog FIXED not suppressed; pre-fix inventory recorded; suppression count recorded before and after | Lint gate | `task lint:go` | ⬜ | ⬜ pending |
| T-03-10-03 | 03-10 | 7 | TODO-CI-01 | T-03-36, T-03-37 | Gate demonstrated RED on both a formatting and an idiomatic planted violation, then green; the new CI job is bound by the single-definition guard | Go unit + CI wiring | `go test ./internal/upgrade/ -run TestWorkflowRunBodiesInvokeTask -count=1 -v` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

**Sampling continuity:** no three consecutive task IDs above lack an `<automated>` verify.
The four that carry none are the two blocking-human legitimacy/review checkpoints and the
two blocking-human decision checkpoints; none of them is adjacent to another such task
within the same plan.

**Manual-only residue** (each recorded in its plan's SUMMARY, none substituting for an
automated check): the browser-level history walk (T-03-07-03), the live permalink
follow-through before and after a re-index (T-03-08-02), the three live degrade
observations (T-03-09-02), and the rendered-highlighting spot check (T-03-04-01).

---

## Wave 0 Requirements

- [x] `internal/uiserver/confinement_test.go` — SRV-05's negative cases **paired with** a passing in-repo control (planned: 03-02, T-03-02-01/T-03-02-02)
- [x] `internal/gitmeta/permalink_test.go` and `internal/uiserver/permalink_test.go` — BRW-09's D-06/D-07/D-08/D-09 branches (planned: 03-05, T-03-05-02/T-03-05-03)
- [x] `web/highlight_coverage_test.go` (package `web_test`, so the production `web` package gains no dependency on `internal/indexer`; planned: 03-04, T-03-04-03) — mirrors `internal/indexer/capability/matrix_test.go:33-41`'s "no missing, no extra" set-equality pattern against `indexer.RegisteredLanguageIDs()`, guarding the 13-module hljs registration list (covering 14 registered `LanguageSpec.ID` values, since the typescript module's own alias list already claims `tsx`) against silent drift when a 15th language is added
- [x] **JS test harness — DECIDED (maintainer, 2026-08-28): adopt `vitest` + `jsdom` + `@testing-library/svelte`.** Planned: 03-01, T-03-01-01/T-03-01-02/T-03-01-03. Configuration lives in `web/vite.config.ts`'s `test:` block, NOT a separate `vitest.config.ts` — `web_source_files()` in `Taskfile.yml` enumerates `vite.config.ts` and would not see a separate file. Tests live in `web/tests/`, which SvelteKit's generated tsconfig already includes and which `web_source_files()` deliberately does NOT hash, so a test edit does not churn the BLD-03 source digest. Scope covers both pure-TS modules (`browse-url.ts`, `rpc-errors.ts`, the click-to-definition name matcher) and Svelte component behavior (search combobox, keyboard nav, error states).
  - Harness bring-up MUST prove the runner executes: land one deliberately-failing assertion, observe it RED, then fix it. A test suite that reports 0 failures because it ran 0 tests is the exact vacuous-pass shape rule `84d1gfpywd` names, and a fresh harness is where it is most likely.
  - New devDependencies enter `pnpm-lock.yaml`. Verified during planning: `task web:lockfile`'s assertion is a FLOOR of 50 against an observed 129, not an exact expected count, so adding packages cannot break it — but the target's `desc:` records the observed count as of 02-07 and 03-01 re-runs the gate so the new observed number is printed and recorded.
  - `pnpm` ≥10 blocks lifecycle scripts by default, and a bare pnpm install with newly-blocked scripts exits 0 with only a warning. Verified during planning: this repository already closes that hole — `web/pnpm-workspace.yaml` sets `strictDepBuilds: true`, which makes such an install exit NON-zero, and `task web:deps:strict` separately asserts the `allowBuilds` key is present (an absent key is a named failure distinct from a committed empty map). 03-01 still runs `pnpm approve-builds --all` and records WHICH of the two outcomes occurred, because a silent no-op and a real approval are not the same event.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| `Explore` NL results rendered alongside exact-name search | BRW-08 | Requires a live index with real relevance ranking; jsdom cannot supply meaningful ranked data | Run `codegraph ui` against this repo's index; enter a natural-language question; confirm both sections populate and are visually distinguishable |
| GitHub permalink opens the correct remote line at the indexed commit | BRW-09 | Requires a real remote and a browser; the Go unit tests cover URL *derivation*, not that the resulting page is right | Open a file at a known line, follow the permalink, confirm the remote view matches what the UI showed |
| Syntax highlighting renders correctly across the registered languages | BRW-06 | Visual correctness is not assertable; the automated guard covers *registration coverage*, not rendering quality | Open one file per registered language; confirm tokens are colored and nothing renders as plain text |

*Reduced from the pre-decision list: debounce/abort, keyboard nav and error states moved to automated jsdom coverage.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Per-Task Verification Map rewritten with real planner-assigned task IDs
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] JS harness proven RED-then-GREEN before any JS test result is trusted
- [ ] Every guard carries a positive assertion that it did its work (rule `84d1gfpywd`)
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
