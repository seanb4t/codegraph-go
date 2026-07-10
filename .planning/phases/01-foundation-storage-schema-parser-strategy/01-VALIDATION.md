---
phase: 1
slug: foundation-storage-schema-parser-strategy
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-07-10
---

# Phase 1 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Seeded from RESEARCH.md §"Validation Architecture". Task IDs are filled in by the planner; requirement→test mapping is locked here.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go standard `testing` package (`go test`) — no third-party test framework for a greenfield Go project |
| **Config file** | none yet — Wave 0 creates `go.mod` and the package skeleton |
| **Quick run command** | `go test ./<changed-package>/... -run <TestName> -v` |
| **Full suite command** | `go test ./... -race -count=1` |
| **Estimated runtime** | ~30–90s early; the `-race` concurrency test dominates |

---

## Sampling Rate

- **After every task commit:** Run targeted `go test ./<changed-package>/... -run <TestName>`
- **After every plan wave:** Run `go test ./... -race -count=1`
- **Before `/gsd-verify-work`:** Full suite green, **plus** the parser-spike benchmark numbers (CGo vs wazero: throughput, incremental reparse, static-build impact, crash isolation) recorded in a decision document before the phase gate
- **Max feedback latency:** ~90 seconds

---

## Per-Task Verification Map

> Requirement-anchored rows from RESEARCH.md. Planner assigns concrete `{plan}-{task}` IDs; every row must land on a task or a Wave 0 stub.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| TBD | TBD | 1 | INDX-05 | — | Concurrent lock-free readers alongside a single coordinated writer | integration (`-race`) | `go test ./internal/graphstore/... -race -run TestConcurrentReadersSingleWriter` | ❌ W0 | ⬜ pending |
| TBD | TBD | 1 | INDX-05 | — | No package bypasses the `GraphStore` interface | static/architecture | `go test ./internal/graphstore/archtest/... -run TestNoPackageBypassesGraphStore` | ❌ W0 | ⬜ pending |
| TBD | TBD | 1 | ARCH-01 | — | Schema-versioned records round-trip unknown/future fields | unit | `go test ./internal/schema/... -run TestSchemaRoundTripsUnknownFields` | ❌ W0 | ⬜ pending |
| TBD | TBD | 1 | ARCH-01 | — | Bulk graph export re-imports losslessly | integration | `go test ./internal/graphstore/... -run TestBulkExportReimportsLosslessly` | ❌ W0 | ⬜ pending |
| TBD | TBD | 2 | Success Criterion 3 (parser spike) | T-DoS-crash | CGo-vs-wazero parse throughput / incremental / crash-isolation comparison | benchmark (not pass/fail) | `go test ./tools/spike/... -bench=. -benchmem` | ❌ W0 | ⬜ pending |
| TBD | TBD | 1 | V5 input-validation | T-Tamper-key | Path/id key segments escaped/length-prefixed; file-size ceiling before parse | unit | `go test ./internal/graphstore/... -run TestKeyEncodingRejectsDelimiterInjection` | ❌ W0 | ⬜ pending |
| TBD | TBD | 2 | Success Criterion 4 (golden corpus) | — | TS DDL + golden JSON fixtures captured, non-empty, determinism-stripped | smoke | `go test ./testdata/golden/... -run TestGoldenFixturesExist` (or shell/Taskfile smoke check) | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `go.mod` — does not exist yet; `go mod init` is the first executable step of this phase
- [ ] `internal/graphstore/store_test.go` — concurrency test, covers INDX-05
- [ ] `internal/graphstore/archtest/import_graph_test.go` — interface-bypass test (via `golang.org/x/tools/go/packages`), covers INDX-05
- [ ] `internal/schema/roundtrip_test.go` — unknown-field round-trip, covers ARCH-01
- [ ] `internal/graphstore/export_test.go` (or folded into `store_test.go`) — bulk export/reimport, covers ARCH-01
- [ ] `internal/graphstore/keyenc_test.go` — key-delimiter injection guard, covers V5 input-validation
- [ ] `tools/spike/bench_test.go` — parser benchmark harness, covers Success Criterion 3
- [ ] `testdata/golden/` capture harness (shell script / `Taskfile` target invoking the installed `codegraph` CLI + `sqlite3`) — covers Success Criterion 4

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Parser-strategy decision (CGo vs wazero) | Success Criterion 3 | The spike produces *benchmark numbers + a documented decision*, not a green/red assertion — a human reads throughput / static-build / crash-isolation results and records the choice | Run `go test ./tools/spike/... -bench=. -benchmem`; capture throughput + incremental-reparse + `CGO_ENABLED=0` build result + crash-isolation outcome into a decision doc (e.g. `PARSER-DECISION.md`) |
| Golden-corpus capture from live TS v1.3.1 | Success Criterion 4 | Fixtures are produced by an external CLI (`codegraph` + `sqlite3`) against a live install, not by Go code; capture is a one-shot operator action | Run the capture harness against the on-disk TS index; strip non-deterministic FTS `score` floats and `updatedAt` timestamps; commit version-stamped fixtures under `testdata/golden/` |
| CGo crash-isolation dimension | Success Criterion 3 / T-DoS-crash | A C-level segfault in a grammar's external scanner cannot be caught by Go `recover()`; validating "valid input passes" does NOT validate this — needs deliberately malformed/adversarial input | Feed malformed/oversized/deeply-nested source to each backend in the spike; observe whether the host process survives (wazero trap → recoverable error) or dies (CGo segfault) |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 90s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
