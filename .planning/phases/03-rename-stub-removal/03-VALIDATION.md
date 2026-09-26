---
phase: "3"
slug: "rename-stub-removal"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: "2026-09-26"
---

# Phase 3 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (stdlib `testing`) |
| **Config file** | none — `go.mod` |
| **Quick run command** | `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/... ./test/integration/... -v` (targeted to the touched packages) |
| **Full suite command** | `GOTOOLCHAIN=go1.26.6 task test:unit && GOTOOLCHAIN=go1.26.6 task test:integration && task docs:cli:drift` |
| **Estimated runtime** | not measured at plan time — record the first observed run in the SUMMARY |

`GOTOOLCHAIN=go1.26.6` is mandatory on every `go`/`task` invocation on this machine (03-RESEARCH.md: Go 1.27.1 breaks `cockroachdb/swiss`).

---

## Sampling Rate

- **After every task commit:** Run the quick run command
- **After every plan wave:** Run the full suite command
- **Before `/gsd-verify-work`:** Full suite must be green, plus the census (CONTEXT D-06/D-07) and `task changie -- next auto` (D-12) evidence recorded in VERIFICATION.md
- **Max feedback latency:** one full-suite run

---

## Per-Task Verification Map

Task IDs are filled in once plans exist; rows are seeded per requirement from 03-RESEARCH.md §Validation Architecture.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| TBD | TBD | TBD | VERB-09 (SC1) | — | N/A | integration (spawned binary) | `GOTOOLCHAIN=go1.26.6 go test ./test/integration/... -run <rewritten-guard-name> -v` | ❌ W0 (rewrite of `renamed_stubs_test.go`) | ⬜ pending |
| TBD | TBD | TBD | VERB-09 (SC2) | — | N/A | task + unit | `task docs:cli:drift`; `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/ -run 'TestEveryRegisteredFlagIsAccountedFor\|TestEveryCommandHasGroupID' -v` | ✅ | ⬜ pending |
| TBD | TBD | TBD | VERB-09 (SC2 census) | — | N/A | scripted census (positive control first) | `rg -nU -w --hidden 'codegraph\s+(query\|unlock)' --glob '!.planning/**' --glob '!CHANGELOG.md' --glob '!web/build/**' --glob '!.git/**' .` → zero hits on the clean tree | ✅ (one-off instrument) | ⬜ pending |
| TBD | TBD | TBD | CAP-05, VERB-09 (SC3/SC4) | — | N/A | process evidence at phase close | `git log --grep='Changie-Phase: 3' -1 --format='%H %(trailers:key=Changie-Phase,valueonly) %(trailers:key=Changie-Summaries,valueonly)'`; `task changie -- next auto` → `v0.15.0` | N/A (observed by the verifier) | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `test/integration/renamed_stubs_test.go` rewritten (or renamed) as the unknown-command guard, landing RED in a `test(03-NN):` commit before the `feat(cli)!:` removal (CONTEXT D-02, rule `x1cjy9vyhq`)
- [ ] `.planning/phases/03-rename-stub-removal/03-MUTATION-LOG.md` created during execution with the census family entry (D-06)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Capability `verify:post` dispatch writes the `Breaking` fragment | CAP-05 | Fires only at the real phase close inside `/gsd-execute-phase`; cannot be pre-written or simulated without invalidating CAP-05 | After close, the verifier reads `.changes/unreleased/`, the `Changie-Phase: 3` commit and its branch membership, and records them in VERIFICATION.md (CONTEXT D-08/D-10/D-11) |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency within one full-suite run
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
