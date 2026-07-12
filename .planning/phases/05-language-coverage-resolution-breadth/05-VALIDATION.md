---
phase: 5
slug: language-coverage-resolution-breadth
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-07-11
---

# Phase 5 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | `go test` (standard library, table-driven; existing pattern in `internal/indexer/goextract/goextract_test.go`) |
| **Config file** | none — Go toolchain; grammar modules added via `go.mod` requires |
| **Quick run command** | `go test ./internal/indexer/... ./internal/parser/...` |
| **Full suite command** | `go test -race ./...` |
| **Estimated runtime** | ~60–180 seconds (grows as per-language extractor + golden-parity fixtures land; CGo grammars increase build time) |

---

## Sampling Rate

- **After every task commit:** Run the quick run command scoped to the touched package(s).
- **After every plan wave:** Run the full suite command (`go test -race ./...`).
- **Before `/gsd-verify-work`:** Full suite must be green, including per-language golden-parity fixtures.
- **Max feedback latency:** ~180 seconds.

---

## Per-Task Verification Map

> Populated by the planner/executor as tasks are defined. See `05-RESEARCH.md`
> §"Validation Architecture" for the per-requirement validation approach
> (per-language golden-parity fixtures, dispatch-edge assertions, route-node
> assertions, provenance-tag assertions, coverage matrix).

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 05-01-01 | 01 | A | (foundation) | T-05-01 / — | Parser-per-language selected correctly under multi-language worker pool | unit | `go test ./internal/indexer/...` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] Per-language golden-parity fixtures under `internal/indexer/testdata/` (Java, C#, Python, TS/JS) — expected node/edge shape captured from TS CodeGraph v1.3.x.
- [ ] Dispatch-synthesis fixtures asserting `implements` edges carry `provenance: heuristic` + Line/Col.
- [ ] Framework-route fixtures asserting `route` nodes link to handlers via heuristic `calls`/`handles` edges.
- [ ] Grammar module pins added to `go.mod` (versions per `05-RESEARCH.md` recommendations).

*Existing `go test` infrastructure covers the harness; Wave 0 adds fixtures + grammar deps.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Real-repo validation of each priority-4 language | LANG-02..05 | Requires cloning representative real-world repos too large to vendor | Run `codegraph index` on the selected corpus repo; diff shape against TS CodeGraph golden output per `05-RESEARCH.md` §Validation Architecture |
| Swift/Kotlin `[SUS]` grammar acceptance | LANG-06 | Community grammars, no semver tags — human verification gate before install | `checkpoint:human-verify` per research recommendation |

*Documented-partial mainstream-tier coverage is validated by the capability matrix (D-11), not full parity.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 180s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
