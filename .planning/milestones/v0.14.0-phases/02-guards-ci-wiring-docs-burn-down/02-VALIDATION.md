---
phase: "2"
slug: "guards-ci-wiring-docs-burn-down"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: validated
nyquist_compliant: true
wave_0_complete: true
created: "2026-09-16"
---

# Phase 2 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | `go test` (stdlib) for `internal/upgrade` shape tests + bash check scripts with `--self-test` positive controls (`scripts/check-ruleset-drift.sh`, `scripts/check-workflow-output-delimiter.sh`); Vitest + `svelte-check` for the web tree |
| **Config file** | `Taskfile.yml` (single source of truth for every CI job body — `TestWorkflowRunBodiesInvokeTask`); `.github/required-status-checks.txt` (shared fixture read by the Go test and the CI step) |
| **Quick run command** | `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/upgrade/... && bash scripts/check-ruleset-drift.sh` |
| **Full suite command** | `GOTOOLCHAIN=go1.26.6 task test:unit && cd web && pnpm check && pnpm vitest run && cd .. && task web:build && task web:drift && task web:components:drift` |
| **Estimated runtime** | ~2 seconds |

---

## Sampling Rate

- **After every task commit:** Run `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/upgrade/... && bash scripts/check-ruleset-drift.sh`
- **After every plan wave:** Run the full suite command above (the orchestrator ran `task test:unit` after each of the three waves: 52/52 packages ok each time)
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 2 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 02-01.1–2 | 02-01 | 1 | GRD-09 | T-02-01-01 | `web:drift` fails on a clean checkout of `98cd41dd` (exit 201, output half 32 vs 22); the Taskfile target is unchanged (D-01) | replay + CI gate | `task web:drift` (ci.yml BLD-03 step, every PR); RED replay `git worktree add --detach <scratch> 98cd41dd && (cd <scratch> && task web:drift)` | ✅ `02-MUTATION-LOG.md` Family (a); ✅ `Taskfile.yml` byte-identical since 68f74bb6 | ✅ green |
| 02-02.1–2 | 02-02 | 2 | GRD-11 | T-02-02-01 | `check:gonum` and `check:no-force-layout` block every PR via the required `test` job; syft installed at release.yml's SHA pin | CI wiring + positive controls | `task check:gonum` / `task check:no-force-layout` (each prints the count it inspected before asserting); `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/upgrade/ -run TestWorkflowRunBodiesInvokeTask` | ✅ ci.yml steps; ✅ Family (b) planted `name: 'cose'` RED | ✅ green |
| 02-03.1–2 | 02-03 | 1 | GRD-12 | T-02-03-01 | Fixture == live `protect-main` contexts (exact set equality); hard-fail on non-200/empty/unparseable/non-active/zero; never skips | shell (CI-only) + unit (tdd) | `bash scripts/check-ruleset-drift.sh` (self-check first; `--connect-timeout 10 --max-time 30` after WR-01); `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/upgrade/ -run 'TestRequiredCheck'` | ✅ `scripts/check-ruleset-drift.sh`, `.github/required-status-checks.txt`, loader tests (RED 53845d7d → GREEN af7ad8fb) | ✅ green |
| 02-04.1–2 | 02-04 | 3 | GRD-12 | T-02-04-01 | After the maintainer's D-08 PUT the fixture reads 8 and the step is GREEN 8-vs-8 | shell (CI-only) | `bash scripts/check-ruleset-drift.sh` → `ruleset-drift: PASS — 8 contexts identical` | ✅ Family (c) RED (6 vs 7) → GREEN (8 vs 8) | ✅ green |
| 02-05.1–2 | 02-05 | 1 | GRD-10 | T-02-05-01 | All eight vendored families equal one registry snapshot; the rebuilt bundle is committed alongside | shell/CI monitor + build gates | `task web:components:drift` (Corepack pnpm 11.23.0, 50/50 identical); `cd web && pnpm check && pnpm vitest run`; `task web:build && task web:drift` | ✅ (monitor scheduled Mon 08:00 UTC + dispatch; never a merge gate by D-16) | ✅ green |
| 02-06.1–3 | 02-06 | 1 | DOCS-08, DOCS-09, DOCS-10 | — | RELEASE.md carries no raw counts and credits go-sdk; zero live "provenance over the checksums file" claims; one advisory tool-vuln sentence | doc census (one-shot, positive-controlled) | `rg -c 'mark3labs' docs/RELEASE.md` = 0; census `rg -U` with planted control (recorded in 02-06-SUMMARY) | N/A by design — D-13/D-14/D-15: a wording change gets no test | ✅ closed |
| 02-07.1–3 | 02-07 | 2 | GRD-14, DOCS-11 | — | Ledger rows closed only through `gsd-tools windows fixed`; #20/#21/#34 stay `open`; Pending Todos match `gsd-tools init todos`; SEED-001 consumed | CLI verb + tool render | `gsd-tools windows status` (open 6 / fixed 28); `gsd-tools init todos` byte-match; `gsd-tools list-seeds implemented` | N/A — process gate through tool-owned files | ✅ closed |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] `scripts/check-ruleset-drift.sh` + `.github/required-status-checks.txt` — GRD-12 (new; self-check positive control built in)
- [x] `readRequiredCheckNames` loader tests in `internal/upgrade/taskfile_shape_test.go` — GRD-12 (RED 53845d7d → GREEN af7ad8fb)
- [x] `02-MUTATION-LOG.md` Families (a)/(b)/(c) — GRD-09/GRD-11/GRD-12 RED demonstrations
- [x] No framework install needed — every gap closed inside the existing Go / bash / Vitest conventions

*If none: "Existing infrastructure covers all phase requirements."*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Three visual/API deltas from the single-snapshot re-vendor (button secondary hover `color-mix`, `command-link-item` selected state, `table-row` `has-aria-expanded` utility) are acceptable | GRD-10 | D-12 reserves visual/API judgment for end-of-phase maintainer review; no automated check judges taste | Accepted by the maintainer 2026-09-16 (`02-UAT.md` test 1) |
| The D-08 ruleset change (every PR must pass `tmux e2e` + `goreleaser check`) was an informed decision | GRD-12 | A repository-settings consequence only the maintainer can weigh | Affirmed by the maintainer 2026-09-16 (`02-UAT.md` test 2) |
| Window rows #20/#21/#34 stay `open` as record-only deviations | GRD-14 | `gsd-tools windows` has no annotate verb/status; inventing one would violate the planning-artifacts rule | Recorded in 02-07-SUMMARY and STATE.md Blockers; upstream gap drafted |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies (75/75 verify commands resolved and carry a stated failing direction — plan-checker probes)
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags (`pnpm vitest run` one-shot; `go test -count=1`)
- [x] Feedback latency: quick command < 15 s (`go test ./internal/upgrade` ~0.4 s + one ruleset fetch); full suite ~10 min (`task test:unit`), run per wave
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** validated 2026-09-16 by /gsd-validate-phase (autonomous run — no gaps; manual-only rows are by explicit decision)

## Validation Audit 2026-09-16
| Metric | Count |
|--------|-------|
| Gaps found | 0 — every automatable requirement has a durable guard (GRD-09 via CI's clean-checkout `web:drift`, proven RED in Family (a); GRD-11/GRD-12 via steps + positive controls + loader tests; GRD-10 via the drift monitor and build gates); DOCS-08/09/10, GRD-14 and DOCS-11 are manual-only by decision (D-13/D-14/D-15, planning-artifacts rule) |
| Resolved | 0 (nothing to add) |
| Escalated | 0 |

Post-execution note: the code-review fix loop added `--connect-timeout 10 --max-time 30` to the ruleset fetch (WR-01, 97bb6a13) so a network stall fails loud like every other branch — RED proven against a black-hole address (exit 1 in ~10 s), GREEN 8-vs-8 unchanged.
