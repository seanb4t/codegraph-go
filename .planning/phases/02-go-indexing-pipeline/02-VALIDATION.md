---
phase: 2
slug: go-indexing-pipeline
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-07-10
---

# Phase 2 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Per-task rows below are scaffolding — the executor / gsd-nyquist-auditor populate them from the final PLAN.md task IDs. See `02-RESEARCH.md` §"Validation Architecture" for the source strategy.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | `go test` (stdlib testing; table-driven) |
| **Config file** | none — `go.mod` present; CGo required (`CGO_ENABLED=1`) for tree-sitter |
| **Quick run command** | `go test ./internal/...` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~30–90 seconds (CGo parse tests dominate) |

---

## Sampling Rate

- **After every task commit:** Run `go test ./<package-under-change>/...`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd-verify-work`:** Full suite must be green; `go vet ./...` clean
- **Max feedback latency:** ~90 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 02-01-01 | 01 | 1 | LANG-01 | T-02-* / — | Extractor rejects >4 MiB source (MaxSourceBytes) before parse | unit | `go test ./internal/index/...` | ❌ W0 | ⬜ pending |
| 02-0X-XX | — | — | RES-01 | — | Cross-file call/import/embedding edges resolved across packages | integration | `go test ./internal/index/... -run Resolve` | ❌ W0 | ⬜ pending |
| 02-0X-XX | — | — | INDX-02 | — | Two from-scratch indexes of same fixture produce byte-identical Export() | integration | `go test ./internal/index/... -run Determinism` | ❌ W0 | ⬜ pending |
| 02-0X-XX | — | — | INDX-01 | — | `init` builds `.codegraph/`; `uninit --force` removes it cleanly | cli | `go test ./internal/cli/...` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky · Table IDs are provisional until PLAN.md task IDs are finalized.*

---

## Wave 0 Requirements

- [ ] Go test fixture: a small multi-package Go module under `testdata/` (or `internal/index/testdata/`) exercising functions, methods (pointer + value receivers), struct/interface embedding, intra- and cross-package calls, imports, constants — the ground truth for extraction + resolution assertions.
- [ ] Determinism harness: helper that indexes a fixture twice into temp dirs and diffs `GraphStore.Export()` output (NOT raw Pebble files — LSM internals aren't byte-stable).
- [ ] `go test` infrastructure already present (stdlib) — no framework install needed.

*Existing Phase-1 test infrastructure (`go test`, golden fixtures under `testdata/golden/`) covers tooling; Phase 2 adds the indexer fixture + determinism harness.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Indexing a real-world Go project produces expected structure (SC4) | LANG-01 | Depends on a large external repo; assert against golden `testdata/golden/corpus/weft-go/` shape rather than a hardcoded count | Index `weft-go` (or the pinned corpus repo); spot-check node kinds + call/contains edges against the golden outputs |

*Most phase behaviors have automated verification; SC4 real-repo parity is spot-checked against the Phase-1 golden corpus.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references (fixture module + determinism harness)
- [ ] No watch-mode flags
- [ ] Feedback latency < 90s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
