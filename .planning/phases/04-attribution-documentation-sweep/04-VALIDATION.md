---
phase: 4
slug: attribution-documentation-sweep
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-15
---

# Phase 4 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` (stdlib) |
| **Config file** | none — `Taskfile.yml` is the single CI definition |
| **Quick run command** | `go test -count=1 ./internal/cli/... ./internal/indexer/capability/...` |
| **Full suite command** | `go test -count=1 ./...` |
| **Estimated runtime** | quick ~15s · full ~1min |

> **`-count=1` is mandatory** — recorded repo gotcha.

---

## Sampling Rate

- **After every task commit:** `go test -count=1 ./...` (full — the sweep touches shared doc/test surfaces)
- **After every plan wave:** `go test -count=1 ./...` + the reference sweeps (`rg` for the deleted framing)
- **Before `/gsd-verify-work`:** full suite green, `rg "FLAG-PARITY"` (harness scope) empty, `gh api repos/seanb4t/codegraph-go/license` returns `MIT` live
- **Max feedback latency:** ~60s (full suite)

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| TBD | TBD | TBD | ATTR-01 | — | N/A | static | `NOTICE` contains MIT transcription + one origin sentence; `rg "ported-heuristics\|flag-parity\|drop-in" NOTICE` empty | ✅ | ⬜ pending |
| TBD | TBD | TBD | ATTR-02 | — | N/A | static | `rg "Relationship to the original" README.md` empty; `## License` contains one past-tense clause linking to NOTICE | ✅ | ⬜ pending |
| TBD | TBD | TBD | ATTR-03 | — | N/A | live | `gh api repos/seanb4t/codegraph-go/license --jq .license.spdx_id` returns `MIT` | ✅ | ⬜ pending |
| TBD | TBD | TBD | DOCS-01 | — | N/A | static | `rg "parity\|upstream\|drop-in\|the original" README.md CONTRIBUTING.md SECURITY.md CODE_OF_CONDUCT.md PARSER-DECISION.md` (comparison sense) empty | ✅ | ⬜ pending |
| TBD | TBD | TBD | DOCS-02 | — | N/A | static+test | `docs/FLAG-PARITY.md` + `internal/cli/flag_parity_test.go` absent; `rg "FLAG-PARITY\|flag_parity_test"` (tree) empty; `go test -count=1 ./...` green | deletion | ⬜ pending |
| TBD | TBD | TBD | DOCS-03 | — | N/A | static | `docs/LANGUAGE-CAPABILITY-MATRIX.md` carries no reference to another implementation's coverage | ✅ | ⬜ pending |
| TBD | TBD | TBD | DOCS-04 | — | N/A | static | `rg "parity\|upstream\|drop-in\|the original" docs/` (comparison sense) empty | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] None — no new test scaffolding; the sweep is static + a live licence check. Existing `go test ./...` (49 pkgs) and `TestWorkflowRunBodiesInvokeTask` must survive the FLAG-PARITY guard deletion (verified by research).

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Each borderline reference is resolved past-tense-origin (keep) vs comparison (remove), recorded per instance | DOCS-01/03/04 | Term-by-term judgment with recorded reasons, never regex | Read the per-instance decisions in the plan/summary; confirm each borderline reference is classified with a reason |
| The retained attribution reads as one sentence of origin, past tense, no comparison argument | ATTR-01/02 | Prose judgment | Read NOTICE and README `## License` |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] `go test -count=1 ./...` green after the deletion
- [ ] `gh api` licence check verified live, not assumed
- [ ] No watch-mode flags
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
