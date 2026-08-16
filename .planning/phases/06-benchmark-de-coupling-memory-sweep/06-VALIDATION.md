---
phase: 6
slug: benchmark-de-coupling-memory-sweep
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-16
---

# Phase 6 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Source: `06-RESEARCH.md` §Validation Architecture (line 506).

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go built-in `testing` (`go test`) |
| **Config file** | none — Go test discovery is convention-based (`*_test.go`) |
| **Quick run command** | `go test -count=1 ./internal/bench/... ./tools/bench/...` |
| **Full suite command** | `go build ./... && go vet ./... && go test -count=1 ./internal/bench/... ./tools/bench/... ./testdata/golden/...` |
| **Estimated runtime** | ~60–90 seconds (golden suite dominates) |

**Why the golden suite is in the full command:** `weft-go` is pinned in *both*
`tools/bench/realcorpus/manifest.go` and the Phase 1 corpus surface. Editing the
realcorpus manifest (D-09/D-10) must not regress the golden corpus's use of that
shared pin.

---

## Sampling Rate

- **After every task commit:** `go test -count=1 ./internal/bench/... ./tools/bench/...`
- **After every plan wave:** full suite command above, **plus** a fresh positive-controlled
  multiline census (D-04) over the Phase 6 file surface
- **Before `/gsd-verify-work`:** full suite green AND census TOTAL recorded verbatim as 0
  (the number, not merely a zero exit status — rule `84d1gfpywd`: a negative-only guard
  passes vacuously when its anchor stops matching)
- **Max feedback latency:** ~90 seconds

---

## Per-Task Verification Map

Seeded at requirement level — per-task rows are filled once PLAN.md files exist.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| TBD | TBD | TBD | BENCH-01 | — | N/A | inspection (census) | multiline census over `docs/BENCHMARKS.md`, asserting TOTAL=0 for comparison patterns | N/A — census is the test | ⬜ pending |
| TBD | TBD | TBD | BENCH-02 (gate fires) | T-06-01 | Mutation fully reverted; no mutation byte in the commit | unit + mutation rehearsal | `go test -count=1 ./internal/bench/... -run TestCheckRegression` green, then D-08 rehearsal RED → reverted-green | ✅ `internal/bench/regression_test.go` (21 subtests) | ⬜ pending |
| TBD | TBD | TBD | BENCH-02 (runner removed) | — | Input surface net-reduced (`-ts-binary` path input removed) | build + census | `go build ./... && go vet ./...` catches dangling refs to deleted `-ts-binary` / `resolveTSBinary`; census over `tools/bench/**` | ✅ existing build/vet tooling | ⬜ pending |
| TBD | TBD | TBD | BENCH-03 | T-06-03, T-06-04 | `rebless` keeps `contents: read` and no write token; all `uses:` stay SHA-pinned | CI dispatch (manual) + static census | Dispatch the `publish` job on a branch, inspect job summary + artifact; census `bench.yml` for `npm install`, `ts-binary`, comparison terms | Manual-only — Actions jobs are not locally `go test`-able | ⬜ pending |
| TBD | TBD | TBD | MEM-01 | — | Supersede-only; nothing overwritten or deleted | inspection | none — enumeration in `06-MEMORY-SWEEP.md` is the artifact (D-15) | Manual-only, by design | ⬜ pending |
| TBD | TBD | TBD | MEM-02 | — | No present-tense or forward-looking parity framing recalled | inspection | not automatable — D-15 explicitly declines post-sweep re-query | Manual-only, by design | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements. `internal/bench/regression_test.go`,
`tools/bench/runner/main_test.go`, and `tools/bench/realcorpus/manifest_test.go` all exist
and already cover the relevant behaviors.

**One non-obvious hazard the plan MUST handle (from RESEARCH.md Pattern 2):** the only
test-file edits needed are *subtractive*, and they are not optional —
`tools/bench/runner/main_test.go:482-533` and `tools/bench/realcorpus/manifest_test.go:72-77`
currently **assert the existence of** the exact surface D-07/D-09 delete (`-ts-binary`,
`resolveTSBinary`, the `colbymchenry-codegraph` entry). If they are not updated in the same
task as the production change, the suite goes red **for the wrong reason** — and an
"expected" red suite is precisely how a genuine regression gets waved through.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| `bench.yml` `publish` job emits absolute numbers with no second implementation invoked anywhere in the job | BENCH-03 | GitHub Actions jobs cannot be executed by `go test` | Dispatch the workflow on the phase branch; confirm the job summary shows absolute per-corpus figures, the artifact uploads (`if-no-files-found: error` did not trip), and no step installs or invokes another implementation |
| Every retired-framing spine record superseded, none overwritten or deleted | MEM-01 | The engram spine lives outside git; no CI gate can hold it | Review the committed `06-MEMORY-SWEEP.md`: every record enumerated by `short_id` with a supersede / leave-historical verdict and reason; confirm no verdict is "delete" or "overwrite" |
| A session started after the sweep recalls no port/parity framing in the present tense | MEM-02 | Requires a live session's recall, not a test | Start a fresh session in this repo and read the startup recall digest; confirm no record asserts the framing as a current standing fact or future obligation |

**Recorded gap (accepted, D-15):** post-sweep re-query evidence is *not* collected, so
"the supersede actually landed" remains asserted rather than demonstrated. The maintainer
accepted this trade-off; it is logged here so a later audit sees a decision, not an oversight.

**Execution precondition (D-16):** engram tool availability must be verified and fail loud
before the sweep begins. Note the research session found `mcp-gw.fzymgc.house` reachable
(DNS resolves; HTTP responds in ~36ms) — contradicting the `ENOTFOUND` observed at
context-gathering. Engram is configured at user level, not in this repo's `.mcp.json`, so
the real precondition is **`mcp__engram__*` tool availability in the executing session**,
not network reachability. Never record the sweep complete against a store that could not
be enumerated.

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or a recorded manual-only justification
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references (none — subtractive test edits only)
- [ ] Subtractive test edits land in the same task as their production change
- [ ] No watch-mode flags
- [ ] Census TOTAL recorded verbatim, not inferred from exit status
- [ ] Feedback latency < 90s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
