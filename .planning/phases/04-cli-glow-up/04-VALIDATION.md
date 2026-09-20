---
phase: "4"
slug: "cli-glow-up"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: validated
nyquist_compliant: true
wave_0_complete: true
created: "2026-09-17"
---

# Phase 4 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | `go test` (stdlib) — `internal/cli` and `internal/cli/present` package tests through the cobra tree (`execCmd`); `internal/cli/present/archtest` (TUI-01 import-graph guard); `test/integration` against the REAL built binary (`TestMain` builds `binPath`; `renamed_stubs_test.go` is the D-03 stub contract); `test/wireoracle` frozen MCP transcripts (38, spawns the real binary — the D-01 proof); `tools/clidoc` + `task docs:cli:drift` for the generated reference; RED demonstrations recorded in `04-MUTATION-LOG.md` |
| **Config file** | `Taskfile.yml` (`docs:cli`, `docs:cli:drift`, `test:wireoracle`); `internal/cli/testdata/cli-reference-allowlist.txt`; `go.mod` pins go 1.26.6 — every local Go gate runs under `GOTOOLCHAIN=go1.26.6` |
| **Quick run command** | `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/cli/... ./internal/cli/present/...` |
| **Full suite command** | `GOTOOLCHAIN=go1.26.6 go build ./... && GOTOOLCHAIN=go1.26.6 go test -count=1 ./... && GOTOOLCHAIN=go1.26.6 go test -count=1 ./test/integration/... && GOTOOLCHAIN=go1.26.6 task docs:cli:drift && GOTOOLCHAIN=go1.26.6 task test:wireoracle` |
| **Estimated runtime** | ~60 seconds quick; ~4–6 minutes full (wire oracle builds and spawns the binary) |

---

## Sampling Rate

- **After every task commit:** Run `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/cli/... ./internal/cli/present/...`
- **After every plan wave:** Run the full suite command above
- **Before `/gsd-verify-work`:** Full suite must be green, `04-MUTATION-LOG.md` carries a RED entry per guard widened this phase, `04-FANG-VERDICT.md` exists and precedes every renderer commit
- **Max feedback latency:** 90 seconds (quick command)

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| *(filled by the planner — one row per task, from the requirement → test map below)* | | | | | | | | | |

Requirement → test map (from `04-RESEARCH.md` Validation Architecture; every test asserts what the repo owns — D-00):

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CLI-01 / CLI-05 | Every styled verb's plain path is byte-identical to its pre-glow-up golden (captured BEFORE any renderer lands) and carries zero ESC bytes under `NO_COLOR=1` + non-TTY | unit (golden compare) | `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/cli/ -run 'TestPlainGolden\|TestNoColorNonTTYRegression'` | ✅ shipped — `internal/cli/testdata/plain/<verb>.golden` + the compare test (D-16) |
| CLI-02 / CLI-03 | `--color ∈ {auto,always,never,invalid} × TTY` resolver matrix: rewritten environ (incl. non-empty `NO_COLOR` → `NO_COLOR=1`), styled branch, usage error | unit | `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/cli/ -run TestResolveColor` | ✅ shipped — `internal/cli/colorflag.go` + `colorflag_test.go` (D-09/D-11) |
| CLI-04 | `HasDarkBackground` queried at most once, only when styled AND stdin AND stdout are TTYs | unit (injected query seam, call count) | `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/cli/ -run TestResolveColor` | ✅ shipped — same file |
| CLI-05 / CLI-08 | `serve --mcp` transcript byte-identical under any adopted wrapper | real-binary | `GOTOOLCHAIN=go1.26.6 task test:wireoracle` | ✅ exists — reused unmodified (D-01) |
| CLI-08 / D-03 | Rename stubs print exactly once to stderr and exit 1 under any adopted wrapper | real-binary | `GOTOOLCHAIN=go1.26.6 go test -count=1 ./test/integration/ -run TestRenamedStub` | ✅ exists — reused unmodified |
| CLI-06 | Every visible command carries one of the four `GroupID`s; none groupless (planted ungrouped command goes RED) | unit (walk the tree) | `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/cli/ -run TestEveryCommandHasGroupID` | ✅ shipped — extends `cli_reference_test.go`'s walk pattern (D-13) |
| CLI-07 | `-j`/`-l`/`-k`/`-p` present wherever the long form exists on a query verb (planted long-only flag goes RED) | unit (table over the tree) | `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/cli/ -run TestShortFlagsConsistent` | ✅ shipped — new pinning test (D-12) |
| CLI-06 / VERB-06 | Generated reference byte-identical after `--color` and groups land | drift gate | `GOTOOLCHAIN=go1.26.6 task docs:cli:drift && GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/cli/ -run TestEveryRegisteredFlagIsAccountedFor` | ✅ exists |
| GRD-13 | `present` archtest catches every charm-family import path (prefix match), self-defeat probe intact, RED per new module | unit (import-graph walk) + mutation log | `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/cli/present/archtest/...` | ✅ exists — `forbiddenImportPaths` widened (D-15); RED via planted import recorded in `04-MUTATION-LOG.md` |
| CLI-08 / D-02 | Zero new `govulncheck` findings; charm closure stays cgo-free | CI gate + existing test | `GOTOOLCHAIN=go1.26.6 govulncheck ./... && GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/cli/ -run TestCharmCgoClosure` | ✅ exists (baseline: 0 findings on 2026-09-17) |

---

## Wave 0 Requirements

- [x] `internal/cli/testdata/plain/*.golden` — plain-output goldens for every verb gaining a styled branch, frozen BEFORE any renderer lands (D-16)
- [x] `internal/cli/plain_golden_test.go` (or equivalent) — byte-equality + zero-ESC compare over the goldens (CLI-05)
- [x] `internal/cli/colorflag_test.go` — the D-09/D-11 resolver matrix (RED first, `test(04-NN):` commit before `colorflag.go`)
- [x] `TestEveryCommandHasGroupID` — CLI-06 guard, in or beside `cli_reference_test.go` (RED first against the ungrouped tree)
- [x] `TestShortFlagsConsistent` — CLI-07 pinning test (RED via a planted long-only flag, reverted byte-clean)
- [x] `04-MUTATION-LOG.md` — Phase 2/3 shape; one Family per guard proven RED (archtest prefix per new module, GroupID guard, short-flag guard)

*No framework install needed — `go test` is the toolchain-native framework throughout this repo.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Every hue readable on a light theme and a dark theme | CLI-04 (D-06) | Colour readability needs eyes; styled output is never golden-frozen (D-00) | Run `codegraph status`, `codegraph explore <term>`, `codegraph node <sym>`, `codegraph search <term> --full`, `codegraph --help` in (a) Solarized Light or macOS light Terminal and (b) a dark theme; confirm header/label/value/path/count/warning/error are distinct and legible in both |
| `TERM=dumb`, 16-colour, 256-colour, truecolor each render correctly | CLI-02 (D-06) | colorprofile downsampling is library behaviour — observed, not unit-tested (D-00) | Run a styled verb under `TERM=dumb`, `TERM=xterm` (16), `TERM=xterm-256color`, and `COLORTERM=truecolor`; ideally one run over SSH; confirm no garbled escapes and sensible degradation |
| OSC-11 background query latency | CLI-04 (D-11 correction) | `HasDarkBackground` blocks up to 2 s on a non-answering terminal — inherent to the live query | Run a styled verb inside tmux and over SSH; a ~2 s pause before styled output is the query timing out (expected symptom, record the observed wall time); plain/piped output must not pause |
| `--color=always` piped through `less -R` shows colour; `--color=never` on a TTY is plain | CLI-03 (D-09) | End-to-end terminal behaviour on a real pager | `codegraph status --color=always \| less -R`; `codegraph status --color=never` in a terminal |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 90s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** validated 2026-09-19 (retroactively, during the v0.14.0 milestone audit — the verify:post validate-phase hook was never run for this phase)

---

## Validation Audit 2026-09-19 (retroactive)

| Metric | Count |
|--------|-------|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |

Run during the v0.14.0 milestone audit to clear Phase 4's missed `verify:post` hooks (the same gap that left this phase without a SECURITY.md until 2026-09-19). Every Wave 0 item the strategy called for exists and is green at HEAD, so the `❌ W0` placeholders above are now `✅ shipped`:

- 31 plain goldens under `internal/cli/testdata/plain/` with `TestPlainGolden` and `TestNoColorNonTTYRegression` green.
- `internal/cli/colorflag_test.go` carries the D-09/D-11 resolver matrix (`TestResolveColorMatrix`).
- `TestEveryCommandHasGroupID` and `TestShortFlagsConsistent` green.
- `internal/cli/present/archtest` (import graph plus the cgo closure) green; `test/integration` `TestRenamedStub*` green; `task docs:cli:drift` exit 0.
- `04-MUTATION-LOG.md` holds 4 families.

Manual-only rows are unchanged and remain manual by nature (colour readability by eye, the `TERM` matrix, OSC-11 latency, and the `less -R` pager check). One caveat recorded rather than hidden: the security audit run at the same time found T-04-24 — notes reached the terminal unsanitized once Phases 5 and 7 added path-bearing notes. It was fixed under mutation Family (k1) in the Phase 7 log, and the new regression test lives in `internal/cli/install_test.go`.
