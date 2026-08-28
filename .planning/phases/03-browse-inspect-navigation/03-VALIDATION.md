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
| **Config file (JS)** | none yet — Wave 0 creates (`vitest.config.ts` or `vite.config.ts` `test:` block) |
| **Quick run command** | `go test ./internal/uiserver/... ./internal/query/... ./internal/gitmeta/...` and `pnpm --dir web test` |
| **Full suite command** | `task test:unit` and `pnpm --dir web test` |
| **Estimated runtime** | quick ~15s (scoped Go packages); JS and full suite ~TBD at first green run |

---

## Sampling Rate

- **After every task commit:** Run the touched package's `go test ./internal/<pkg>/...`
- **After every plan wave:** Run `task test:unit` (full Go suite) plus manual browser UAT for client-only surfaces
- **Before `/gsd-verify-work`:** `task test:unit`, `task proto:drift`, and `task web:build:verify` all green
- **Max feedback latency:** 60 seconds for the quick run

---

## Per-Task Verification Map

Task IDs are assigned by the planner. This table is seeded at requirement level and
MUST be rewritten with real task IDs before `nyquist_compliant: true` can be set.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| TBD | TBD | 0 | SRV-05 | T-03-01 | `../` escape, absolute path and symlink escape each refused at the RPC boundary, **and** a legitimate in-repo path returns real source in the same test (positive control, rule `84d1gfpywd`) | Go integration (Connect handler, real client) | `go test ./internal/uiserver/ -run TestGetNodeDetailConfinement` | ❌ W0 | ⬜ pending |
| TBD | TBD | 0 | BRW-09 | — | `GetPermalink` returns correct URL/availability/reason across: pushed commit, unpushed commit, no remote, non-GitHub remote, missing `commit_sha` | Go unit | `go test ./internal/uiserver/... ./internal/gitmeta/... -run Permalink` | ❌ W0 | ⬜ pending |
| TBD | TBD | 0 | BRW-06 | — | Registered highlighter language set matches `indexer.RegisteredLanguageIDs()` exactly — no missing, no extra | Go unit, literal-fixture comparison | `go test ./internal/indexer/... -run TestHighlightRegistrationCoversRegisteredLanguages` | ❌ W0 | ⬜ pending |
| TBD | TBD | 0 | — | — | vitest + jsdom + @testing-library/svelte installed and configured; one deliberately-failing assertion proves the runner actually executes before any real test is trusted | JS harness bring-up | `pnpm --dir web test` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | NAV-01, NAV-02 | — | URL round-trip is idempotent (parse→serialize); unknown params ignored not rejected (D-12); `goto()` push vs replace produces the correct history-entry count (NOT `pushState` — shallow routing leaves `page.url` unchanged) | JS unit (pure TS, no DOM) | `pnpm --dir web test browse-url` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | BRW-01, NAV-03 | — | Search combobox: debounce, in-flight abort, arrow selection, focus shortcut, `Esc` dismiss | JS component (jsdom + @testing-library/svelte) | `pnpm --dir web test search` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | NAV-04 | — | Each of no-index / stale-index / symbol-not-found renders its own explicit named state, and the three are distinguishable from each other — not merely "not empty" | JS component (jsdom) | `pnpm --dir web test states` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | BRW-08 | — | `Explore` NL results and exact-name `Search` results both render and are distinguishable | Manual UAT (needs a live index) | — | N/A | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/uiserver/confinement_test.go` — SRV-05's negative cases **paired with** a passing in-repo control
- [ ] `internal/uiserver/permalink_test.go` and/or `internal/gitmeta/remote_test.go` — BRW-09's D-06/D-07/D-08/D-09 branches
- [ ] `internal/indexer/highlight_coverage_test.go` — mirrors `internal/indexer/capability/matrix_test.go:33-41`'s "no missing, no extra" set-equality pattern against `indexer.RegisteredLanguageIDs()`, guarding the 13-module hljs registration list against silent drift when a 15th language is added
- [ ] **JS test harness — DECIDED (maintainer, 2026-08-28): adopt `vitest` + `jsdom` + `@testing-library/svelte`.** Wave 0 installs and configures. Scope covers both pure-TS modules (`browse-url.ts`, `rpc-errors.ts`, the click-to-definition name matcher) and Svelte component behavior (search combobox, keyboard nav, error states).
  - Harness bring-up MUST prove the runner executes: land one deliberately-failing assertion, observe it RED, then fix it. A test suite that reports 0 failures because it ran 0 tests is the exact vacuous-pass shape rule `84d1gfpywd` names, and a fresh harness is where it is most likely.
  - New devDependencies enter `pnpm-lock.yaml`, which changes the package count BLD-06's audit gate asserts against. Wave 0 must update that expected count deliberately, not discover the drift as a CI failure.
  - `pnpm` ≥10 blocks lifecycle scripts by default and an install with newly-blocked scripts **exits 0 with only a warning**. If any new test dep needs a build script, BLD-05's `allowBuilds` approval must be recorded — silence here is not success.

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
