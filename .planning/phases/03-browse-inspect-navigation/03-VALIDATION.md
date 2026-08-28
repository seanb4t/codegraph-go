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
| **Framework (JS)** | **none installed** — zero `*.test.*`/`*.spec.*` files under `web/`; no `vitest`/`playwright`/`@testing-library` in `web/package.json` devDependencies |
| **Config file (Go)** | none (stdlib) |
| **Config file (JS)** | none — Wave 0 gap if the plan adopts JS unit tests |
| **Quick run command** | `go test ./internal/uiserver/... ./internal/query/... ./internal/gitmeta/...` |
| **Full suite command** | `task test:unit` |
| **Estimated runtime** | quick ~15s (scoped packages); full suite ~TBD at first green run |

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
| TBD | TBD | TBD | NAV-01, NAV-02 | — | URL round-trip is idempotent (parse→serialize); unknown params ignored not rejected (D-12); push vs replace produces the correct history-entry count | JS unit (pending framework decision) OR manual UAT | `pnpm --dir web vitest run browse-url` *(only if vitest adopted)* | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | BRW-01, BRW-08, NAV-03 | — | Search debounce / request abort / keyboard nav | Manual UAT (browser-driven) | — | N/A | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/uiserver/confinement_test.go` — SRV-05's negative cases **paired with** a passing in-repo control
- [ ] `internal/uiserver/permalink_test.go` and/or `internal/gitmeta/remote_test.go` — BRW-09's D-06/D-07/D-08/D-09 branches
- [ ] `internal/indexer/highlight_coverage_test.go` — mirrors `internal/indexer/capability/matrix_test.go:33-41`'s "no missing, no extra" set-equality pattern against `indexer.RegisteredLanguageIDs()`, guarding the 13-module hljs registration list against silent drift when a 15th language is added
- [ ] **Open decision — JS test framework:** adopt `vitest` for `web/`'s new pure-TS modules (`browse-url.ts`, `rpc-errors.ts`, the click-to-definition name matcher), or explicitly decline and accept Go-RPC-tests-plus-manual-UAT as this phase's JS coverage. No JS test framework exists today. This is a Wave-0-or-explicitly-declined decision, never a silent skip.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Search-as-you-type debounce and in-flight request abort | BRW-01 | No JS test framework installed; behavior is timing- and DOM-dependent | Type a partial symbol quickly; observe only the final query's results render and earlier requests are aborted |
| `Explore` NL results rendered alongside exact-name search | BRW-08 | Requires a live index and visual comparison of two result sections | Enter a natural-language question; confirm both sections populate and are distinguishable |
| Keyboard nav: focus shortcut, arrow selection, `Esc` dismiss | NAV-03 | Focus behavior is not observable without a real browser | Use only the keyboard from focus shortcut → result selection → `Esc` |

*Superseded in whole or part if the vitest decision above lands as "adopt".*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Per-Task Verification Map rewritten with real planner-assigned task IDs
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] JS test framework decision recorded (adopted or explicitly declined)
- [ ] Every guard carries a positive assertion that it did its work (rule `84d1gfpywd`)
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
