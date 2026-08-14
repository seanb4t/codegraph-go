---
phase: 2
slug: golden-harness-re-authoring-re-freeze
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-14
---

# Phase 2 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` (stdlib), plus `go-sdk` MCP in-process client for the CLI==MCP trio |
| **Config file** | none — `testdata/golden/*_test.go` are a normal `testing` package reached only via `go test ./testdata/golden/...` (GOLDEN-01) |
| **Quick run command** | `go test -count=1 ./testdata/golden/...` |
| **Full suite command** | `task test:golden` (CI runs this as its own step) |
| **Estimated runtime** | quick ~20s · full golden suite longer (locked-corpus indexing) |

> **`-count=1` is mandatory** — recorded repo gotcha: a cached `ok` can mask a real regression.

---

## Sampling Rate

- **After every task commit:** `go test -count=1 ./testdata/golden/...` (fast; behavioral corpus is in-repo)
- **After every plan wave:** `go test -count=1 ./testdata/golden/...` over the whole re-authored suite
- **Before `/gsd-verify-work`:** `go test -count=1 ./...` AND `go test -count=1 ./testdata/golden/...` both green (CODE-02 criterion 1)
- **Max feedback latency:** ~20s on the per-task loop

---

## Per-Task Verification Map

> Task IDs assigned by the planner; seeded from the requirement→test map.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| TBD | TBD | TBD | CODE-02 | — | N/A | static+test | `rg "parity" testdata/golden/` → empty (harness) + `go test -count=1 ./... && go test -count=1 ./testdata/golden/...` | rename diff | ⬜ pending |
| TBD | TBD | TBD | FIXT-04 | T-02-01 | Do NOT touch `NOTICE`/README origin attribution (licence, not framing) | static | `rg -i "weft\|colbymchenry\|mcp-capture\|capture.sh" testdata/golden/` → empty (harness scope) | n/a | ⬜ pending |
| TBD | TBD | TBD | FIXT-05 | — | N/A | unit+behavioral | `go test -count=1 ./testdata/golden/... -run TestCorpusBehavior` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | FIXT-06 | T-02-03 | capture-to-temp-then-move; non-empty + marker assertion before install | e2e re-baseline | byte-identity test over every re-frozen golden | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | FIXT-06 | T-02-04 | never index an unverified checkout — `task corpora:assert` (four-part) precedes gocapture reading locked trees | static | verify block runs `task corpora:assert` before capture | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `testdata/golden/gocapture/main.go` — locked-corpus + `corpus/behavioral/` capture specs (gocapture is a `main` program driven manually + guarded by existing `TestGoSideFixturesRegenerated`)
- [ ] `corpus/behavioral/CASES.json` — the D-04 case map (a test data file, consumed by the re-authored corpus test)
- [ ] `testdata/golden/behavioral_test.go` — re-authored successor to `golden_parity_test.go` reading `CASES.json` and the locked corpora
- [ ] A byte-identity or non-empty assertion over every re-frozen golden so a bare/missing golden cannot read as satisfied

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| One reviewed re-freeze diff, every changed line attributable to one named cause | CODE-02 criterion 2 | Review judgment — a diff must be read, not asserted | Read the re-freeze diff; confirm every golden byte traces to "re-captured from Go output against locked corpora" and no identifier moved |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Every test command forces `-count=1`
- [ ] Feedback latency < 30s on the per-task loop
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
