---
phase: "09"
slug: "source-view-follow-through-breadcrumb-editor-handoff"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: "2026-09-12"
---

# Phase 09 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Seeded from `09-RESEARCH.md` § Validation Architecture. Every framework version and
> file path below was read from `web/package.json`, `Taskfile.yml`, and `web/scripts/`
> during research, not recalled.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` stdlib (server, CLI, engine); `vitest` 4.1.11 + `@testing-library/svelte` 5.4.2 on jsdom 30.0.1 (frontend unit); `@playwright/test` 1.62.1 driven by committed `web/scripts/*.mjs` (live browser — no `playwright.config.*` exists; scripts launch `chromium` directly) |
| **Config file** | `web/vite.config.ts` for vitest (Phase 2's `resolve.conditions:['browser']` fix, guarded on `process.env.VITEST`); none for Playwright; `Taskfile.yml` targets `test:unit` / `web:test` are the Go/frontend config surface |
| **Quick run command** | `cd web && pnpm test` (frontend unit, seconds) and `GOTOOLCHAIN=go1.26.6 go test ./internal/uiserver/... ./internal/cli/... ./internal/query/...` (targeted Go) |
| **Full suite command** | `GOTOOLCHAIN=go1.26.6 task test:unit` + `task web:test` + `node web/scripts/breadcrumb-check.mjs` (new, live browser against a real index) |
| **Estimated runtime** | ~10–20 s (vitest) · ~30–60 s (targeted Go) · ~30 s (Playwright script, one chromium launch against a running `codegraph ui`) |

**Toolchain prefix (repo landmine).** Local `go build` fails on any machine whose Go is newer
than 1.26.6 because of a `cockroachdb/swiss` incompatibility. Prefix every local Go run with
`GOTOOLCHAIN=go1.26.6` — the Phase 8 verify commands all carry it and this phase inherits them.

**Live-browser gate mechanism (repo ruling).** Committed Playwright scripts under
`web/scripts/` are the sanctioned, re-runnable live-browser gate (`05-04-PLAN.md` rejected
`agent-browser` for anything that must be durable). BRW-10's "verified in a live browser
against a real index, not only in jsdom" criterion is discharged by a new
`web/scripts/breadcrumb-check.mjs` following `graph-collapse-affordance-check.mjs`'s exact
shape, never by an ad-hoc interactive session.

---

## Sampling Rate

- **After every task commit:** `cd web && pnpm test` plus the targeted Go command above
- **After every plan wave:** `GOTOOLCHAIN=go1.26.6 task test:unit` + `task web:test`; add `node web/scripts/breadcrumb-check.mjs` once the SourcePane per-line restructuring has landed
- **Before `/gsd-verify-work`:** full suite green **and** the Playwright breadcrumb script run once against a real index, with its RED-against-pre-fix-build demonstration recorded in `09-MUTATION-LOG.md`
- **Max feedback latency:** ~60 seconds

---

## Per-Task Verification Map

Task IDs are assigned by the planner; this map binds requirements to their automated commands so
the planner can attach them. Test names are illustrative until the planner fixes them.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| TBD | TBD | TBD | BRW-10 | — | Breadcrumb names the innermost containing symbol at the first visible line; empty state when outside every symbol | unit (jsdom, derivation) | `cd web && pnpm test -- source-pane` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | BRW-10 | — | Same, observed in a live chromium against a real index; RED against the pre-fix build | live-browser | `node web/scripts/breadcrumb-check.mjs` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | BRW-11 | T-09-xx (path confinement) | `GetEditorLink` refuses a traversal-shaped / absolute / symlink-escaping path with `CodeInvalidArgument` before any template substitution; the reason string names only the caller's own relative path | unit (Go, mirrors `TestGetPermalinkPathConfinementAtRPCBoundary`) | `GOTOOLCHAIN=go1.26.6 go test ./internal/uiserver/... -run TestGetEditorLink` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | BRW-11 | T-09-xx (scheme allowlist) | A template whose scheme is outside the positive allowlist is `TEMPLATE_INVALID`; `javascript:` / `data:` / `file:` never reach the browser | unit (Go) | `GOTOOLCHAIN=go1.26.6 go test ./internal/uiserver/... -run TestGetEditorLinkSchemeAllowlist` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | BRW-12 | — | Precedence flag → env → discovery → picker; malformed explicit template is a startup failure, never a silent fallback | unit (Go) | `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/... -run TestEditorURL` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | BRW-12 | — | Preset count is exactly 3 (VS Code, Cursor, JetBrains); the string `Zed` appears in no preset and no UI string | unit (TS) | `cd web && pnpm test -- editor-prefs` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | BRW-12 | — | A per-browser override replaces the server default for that browser only (localStorage-scoped) | unit (TS) + live-browser | `cd web && pnpm test -- editor-prefs` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | BRW-13 | all T-09-xx | Every named threat in `09-SECURITY.md` carries a test reference or a recorded verdict | manual (document) | N/A — audited by `/gsd-secure-phase 9` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/uiserver/editorlink_test.go` — BRW-11 confinement (via `ValidateRepoRelativePath`), scheme allowlist, reason-string distinctness, closed-enum coverage (`BUILDABLE` / `NO_TEMPLATE` / `TEMPLATE_INVALID`)
- [ ] `internal/cli/editordiscovery_test.go` (name illustrative) — BRW-12 precedence (flag → env → discovered → unconfigured) and D-17's fail-fast-on-malformed-explicit-value
- [ ] `web/tests/source-pane-breadcrumb.test.ts` (or extend `source-pane.test.ts`) — BRW-10 derivation logic at the unit level
- [ ] `web/tests/editor-prefs.test.ts` (name illustrative) — preset count, Zed absence, per-browser override precedence
- [ ] `web/scripts/breadcrumb-check.mjs` — live-browser Playwright script following `graph-collapse-affordance-check.mjs`; also the mechanism for the RED-against-pre-fix-build demonstration (run against the pre-fix checkout, capture the stale/empty breadcrumb, record in `09-MUTATION-LOG.md`)
- [ ] `09-SECURITY.md` — BRW-13; every threat row carries a test name or a recorded verdict
- [ ] Framework install: none — vitest, Playwright, and Go's stdlib testing are already present and already used for these exact test shapes in prior phases

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| JetBrains preset actually opens a file at a line on a real JetBrains install (path absolute vs project-relative — Open Question 1) | BRW-12 | Needs an installed IDE; no headless oracle for an external URI handler | With a JetBrains IDE installed and the repo open as a project, select the JetBrains preset, click a gutter line, accept the browser's external-protocol prompt, confirm the IDE lands on that line |
| Safari transient-activation survival for the async-rpc-then-navigate gutter path (Open Question 2) | BRW-11 | Cross-browser external-protocol behavior is not scriptable in the committed chromium-only Playwright gate | In Safari, click a gutter line and confirm the "open in app" prompt appears; record the verdict in `09-SECURITY.md` |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
