---
phase: 03-verb-fold
verified: 2026-09-16T23:59:00Z
status: passed
score: 8/8 must-haves verified
covered_files: [".planning/REQUIREMENTS.md", ".planning/ROADMAP.md", ".planning/phases/03-verb-fold/03-01-PLAN.md", ".planning/phases/03-verb-fold/03-01-SUMMARY.md", ".planning/phases/03-verb-fold/03-02-PLAN.md", ".planning/phases/03-verb-fold/03-02-SUMMARY.md", ".planning/phases/03-verb-fold/03-03-PLAN.md", ".planning/phases/03-verb-fold/03-03-SUMMARY.md", ".planning/phases/03-verb-fold/03-04-PLAN.md", ".planning/phases/03-verb-fold/03-04-SUMMARY.md", ".planning/phases/03-verb-fold/03-MUTATION-LOG.md", ".planning/phases/03-verb-fold/03-REVIEW-FIX.md", ".planning/phases/03-verb-fold/03-REVIEW.md", ".planning/phases/03-verb-fold/COVERAGE.md", "docs/CLI-REFERENCE.md", "internal/cli/daemon.go", "internal/cli/daemon_test.go", "internal/cli/index.go", "internal/cli/index_lock_test.go", "internal/cli/notice_test.go", "internal/cli/query_cli_test.go", "internal/cli/renamed.go", "internal/cli/renamed_test.go", "internal/cli/root.go", "internal/cli/search.go", "internal/cli/testdata/cli-reference-allowlist.txt", "internal/daemon/lock.go", "test/integration/renamed_stubs_test.go"]
covered_digest: "v1:sha256:e182f9b79247cf7d0304198b93f0a6a4cfc036c1c6560d246cf5a4ea6addf1a4"
behavior_unverified: 0
overrides_applied: 0
---

# Phase 3: Verb Fold Verification Report

**Phase Goal:** The CLI verb surface is streamlined — `query` folds into `search --full` on the same ranking function and `unlock` becomes `daemon unlock` — with every consumer of the old names found before and after the edit, the removed verbs exiting non-zero with a rename message for one release, and the generated reference, completions, man pages and goldens re-frozen to the new surface in a reviewed diff.
**Verified:** 2026-09-16T23:59:00Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

All checks below were run independently in this session against the real HEAD binary and the real
gate commands — not taken from SUMMARY.md or 03-MUTATION-LOG.md narration. Where the mutation log
already contains a transcript, it is cited alongside my own independent re-run.

### Observable Truths

| # | Truth | Status | Evidence |
|---|---|---|---|
| 1 | `search --full` renders two-line human output (superset of default) and the `MarshalQueryJSON` envelope under `--json`; default `search`/`search --json` unchanged | ✓ VERIFIED | Built `/tmp/03-verify-bin` from HEAD. `search Alpha --full` → `Alpha (function) pkga/pkga.go:13` + `    Alpha  () int`; `search pkga --full` → `    example.com/gofixture/pkga` line for the package; `search zzz-no-such-symbol --full --json` → `[]`. `search matchNodes` / `search matchNodes --json` on the repo's own index reproduce 03-01's pre-fold `sha256` digests exactly (`322f8245…` and `8fda99aa…`) — independently confirms byte-identity, not just cited from the log. |
| 2 | `search` accepts `-k/-l/-j` shorthand equal to long forms, on both `search` and `search --full` | ✓ VERIFIED | `diff <(search main --full -k function -l 1 -j) <(search main --full --kind function --limit 1 --json)` → empty (byte-identical) |
| 3 | `codegraph query …` / `codegraph unlock …` are hidden stubs: stderr-only two-line rename message, empty stdout, exit 1, nothing executed, no `Deprecated` field, no `os.Exit` | ✓ VERIFIED | Ran the real binary: `query main` → stdout 0 bytes, stderr exactly 2 lines, exit=1; `unlock /tmp` → same shape, exit=1; `query --json foo bar` → identical two-line stderr (flags/args ignored). Read `internal/cli/renamed.go` in full: `Hidden: true` ×2, `DisableFlagParsing: true` ×2, single `fmt.Errorf` per stub (no `Fprintln`, no `Deprecated`, no `os.Exit`, no engine/filesystem calls). |
| 4 | `daemon unlock [path]` behaves identically to the old `unlock` verb (verbatim move); the two live messages name `codegraph daemon unlock` | ✓ VERIFIED | `daemon unlock <emptydir>` → `no lock present at … — nothing to do`, exit 0. Read `internal/cli/daemon.go:272-291` — `newDaemonUnlockCmd` is `Use: "unlock [path]"`, `MaximumNArgs(1)`, `daemon.Unlock(codegraphDir)`, registered via `AddCommand(newDaemonStartCmd(), newDaemonStopCmd(), newDaemonUnlockCmd())`. `internal/daemon/lock.go:208` and `internal/cli/index.go:126,128` both read `codegraph daemon unlock` (grepped directly). |
| 5 | VERB-05 census: positive-controlled, word-boundary, multiline; zero old-verb references remain outside the stub declaration | ✓ VERIFIED | Independently re-ran the exact instrument from 03-MUTATION-LOG.md (`rg -nU -w --hidden 'codegraph\s+(query\|unlock)' …`) on the live tree: only 4 hits, in `internal/cli/renamed.go` (doc comment, 2 lines) and `internal/cli/testdata/cli-reference-allowlist.txt` (the 2 D-12 lines) — matches Family (c)'s recorded "after" exactly. |
| 6 | `docs/CLI-REFERENCE.md` regenerated as a reviewed diff; `task docs:cli:drift` and `TestEveryRegisteredFlagIsAccountedFor` green; completions/man reflect new surface | ✓ VERIFIED | `task docs:cli:drift` → `compared 1 generated file`, byte-identical, exit 0. `TestEveryRegisteredFlagIsAccountedFor` → `--- PASS`, `walked 37 commands …, 3 accepted via testdata/cli-reference-allowlist.txt`. `__complete ""` → 24 entries, no `query`/`unlock`/`man`. `man <dir>` → 29 pages, `codegraph-daemon-unlock.1` present, stub pages absent. |
| 7 | Goldens re-frozen with a RED demonstration (never a blanket regenerate); 8-tool MCP set and wire-oracle transcripts unchanged | ✓ VERIFIED | **Independently reproduced** the RED demo myself (not just read from the log): flipped `query` stub's `Hidden` to `false` in the working tree → `task docs:cli:drift` exited 1 with `+## codegraph query` in the diff; `TestEveryRegisteredFlagIsAccountedFor` failed with `stale allowlist entry: codegraph query`. Reverted (`git checkout --`) → `git diff --quiet` holds, both gates green again. `git diff --quiet 21bb329e..HEAD -- internal/mcp testdata/wireoracle test/wireoracle` holds (zero diff); 8 MCP tool names match; `go test ./test/wireoracle/... -count=1` → `ok` (67.7s, uncached). |
| 8 | The fold lands as one `feat(cli)!:` commit with a `BREAKING CHANGE:` footer naming v0.15.0; all other phase commits are conventional and non-breaking | ✓ VERIFIED | `git log --oneline 21bb329e..HEAD` shows exactly one `feat(cli)!:` commit (`5d69ee2e`), with `BREAKING CHANGE:` footer naming both replacements and v0.15.0; every other subject in range checked against `pr-title.yml`'s regex — all pass; no `ci skip`/`skip ci` anywhere in the range. |

**Score:** 8/8 truths verified (0 present, behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `internal/cli/search.go` | `--full`, `-k/-l/-j`, `renderFullLine`, relocated `resolveStartPath`/`writeJSONLine` | ✓ VERIFIED | Confirmed via `rg` for `StringVarP(&kind, "kind", "k"`, `IntVarP(&limit, "limit", "l"`, `BoolVarP(&jsonOut, "json", "j"`, `BoolVar(&full, "full"`; binary behavior matches D-01 exactly |
| `internal/cli/renamed.go` | `newQueryCmd`/`newUnlockCmd` hidden stubs | ✓ VERIFIED | Read in full — matches D-05/D-06 verbatim, including the WR-01 single-`Errorf` fix |
| `internal/cli/daemon.go` | `newDaemonUnlockCmd`, registered | ✓ VERIFIED | Read in full |
| `internal/cli/testdata/cli-reference-allowlist.txt`, `docs/CLI-REFERENCE.md` | Two new allowlist lines; regenerated reference | ✓ VERIFIED | Both allowlist lines present with D-12-shaped reasons; `docs:cli:drift` green |
| `test/integration/renamed_stubs_test.go` (WR-01 fix) | End-to-end binary-level stub assertion | ✓ VERIFIED | Exists, 65 lines; `go test ./test/integration/ -run TestRenamedStub -v` → 4/4 subtests PASS |
| `.planning/ROADMAP.md` `## Backlog` row `999.5` | v0.15.0 stub-removal record, written by the roadmap verb | ✓ VERIFIED | `### Phase 999.5: remove the query/unlock rename stubs` present after `999.4`, Goal value filled with SHA/file names/requirement IDs, no version token in the heading |
| `.planning/phases/03-verb-fold/03-MUTATION-LOG.md` | Families (a)/(b)/(c) + Baseline | ✓ VERIFIED | All three families present with verbatim transcripts; independently reproduced Family (a)'s census and Family (b)'s RED/revert cycle myself this session |
| `.planning/phases/03-verb-fold/COVERAGE.md` | No-external-API declaration | ✓ VERIFIED | Exact declaration line present |

### Key Link Verification

| From | To | Via | Status | Details |
|---|---|---|---|---|
| `root.go`'s `AddCommand(newQueryCmd(), newSearchCmd(), …, newUnlockCmd())` | `renamed.go` | Constructor names unchanged, zero registration edit | ✓ WIRED | Confirmed by grep; `root.go`'s `AddCommand` list byte-unchanged since the fold |
| `search --full` | `eng.Query` → `MarshalQueryJSON`/`renderFullLine` | one command, two tails | ✓ WIRED | `eng.Query(` and `eng.Search(` both present in `search.go`; default and `--full` branches produce distinct, correct output on the live binary |
| `Hidden: true` on stubs | `tools/clidoc`, `cli_reference_test.go`, cobra `__complete`/`doc.GenManTree` | exclusion mechanism | ✓ WIRED | Independently flipped `Hidden` to `false` and watched both consumers (`docs:cli:drift`, `TestEveryRegisteredFlagIsAccountedFor`) react — not vacuous |
| `index.go`/`lock.go` message text | `index_lock_test.go` assertion | RED-first message retarget | ✓ WIRED | `index_lock_test.go` asserts `codegraph daemon unlock`; live messages match; `go test ./internal/cli/...` green |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|---|---|---|---|
| Two-line `--full` render on gofixture | `search Alpha --full` | `Alpha (function) pkga/pkga.go:13` / `    Alpha  () int` | ✓ PASS |
| Shorthand flags equal long forms | `diff <(-k -l -j) <(--kind --limit --json)` | empty diff | ✓ PASS |
| Stub exits 1, stderr-only, stdout empty | `query main`, `unlock /tmp` | exit=1, stdout 0 bytes, 2-line stderr each | ✓ PASS |
| `daemon unlock` clean no-op | `daemon unlock <emptydir>` | `no lock present at … — nothing to do`, exit 0 | ✓ PASS |
| VERB-07 RED demonstration is not vacuous | flip `query` stub `Hidden` → `false`, run both gates, revert | both gates failed with named output, revert byte-clean, both gates green again | ✓ PASS |
| VERB-05 census after the fold | `rg -nU -w --hidden 'codegraph\s+(query\|unlock)' …` | 4 hits, only in `renamed.go` + allowlist | ✓ PASS |
| Byte-identity of default `search` | `sha256(search matchNodes[--json])` on HEAD vs. 03-01's pre-fold digest | identical hashes | ✓ PASS |
| Wire oracle uncached | `go test ./test/wireoracle/... -count=1` | `ok` 67.764s | ✓ PASS |
| `internal/cli/...`, `internal/daemon` lock tests, `go vet`, `gofmt` | full run | all green/clean | ✓ PASS |

### Probe Execution

No `scripts/*/tests/probe-*.sh` files exist in this repository and none are referenced by the phase's plans/summaries. SKIPPED (no runnable probes declared for this phase — verification instead ran the phase's own documented gate commands directly, per the Behavioral Spot-Checks above).

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|---|---|---|---|---|
| VERB-01 | 03-02 | `search --full` full-record envelope + byte-identical default `search` | ✓ SATISFIED | Binary output + digest match confirmed independently |
| VERB-02 | 03-02 | Flag superset `-j/-l/-k/-p` with parity test | ✓ SATISFIED | `diff` of long/short forms empty; `TestSearchFlagShortForms` green |
| VERB-03 | 03-02 | `query` hidden stub, stderr rename message, non-nil error, no `Deprecated` | ✓ SATISFIED | Binary run + source read |
| VERB-04 | 03-02 | `unlock` → `daemon unlock` verbatim move + stub | ✓ SATISFIED | Binary run + source read |
| VERB-05 | 03-01, 03-04 | Positive-controlled before/after census | ✓ SATISFIED | Independently re-ran the exact instrument; matches Family (a)/(c) |
| VERB-06 | 03-02, 03-03 | Reference regenerated, drift/flag gates green, completions/man reflect surface | ✓ SATISFIED | `docs:cli:drift`, `TestEveryRegisteredFlagIsAccountedFor`, `__complete`, `man` all confirmed |
| VERB-07 | 03-03 | Goldens re-frozen w/ RED demo; MCP/wire-oracle unchanged | ✓ SATISFIED | RED demo independently reproduced; zero-diff range + oracle green confirmed |
| VERB-08 | 03-02, 03-04 | Single `feat!:` commit, `BREAKING CHANGE:` footer, backlog row | ✓ SATISFIED | Commit history audited; ROADMAP row confirmed |

**Orphan check:** `.planning/REQUIREMENTS.md` maps VERB-01…VERB-08 only to Phase 3; no additional VERB-* IDs exist beyond the plan-declared set. No orphaned requirements.

**Note (non-blocking):** `.planning/REQUIREMENTS.md`'s checkboxes (`- [ ] **VERB-01**` … `**VERB-08**`) and its traceability table (lines 165-172, "Pending") have not yet been flipped to `[x]`/"Complete", unlike Phase 2's FIX-*/GRD-*/DOCS-* rows which were already checked off by the time that phase's verification ran. This is a documentation-bookkeeping gap, not a functional gap — every VERB requirement is independently confirmed SATISFIED against the live binary and gates above. Flagging for the orchestrator/maintainer to update the checkboxes and traceability table as part of phase close-out.

### Anti-Patterns Found

Scanned every phase-touched source file (`renamed.go`, `search.go`, `daemon.go`, `index.go`, `lock.go`, `root.go`, all touched test files) for `TBD|FIXME|XXX|TODO|HACK|PLACEHOLDER` and stub-shaped patterns (`return null`, empty handlers, hardcoded-empty state feeding rendered output). None found. `gofmt -l` and `go vet` both clean on `internal/cli/` and `internal/daemon/`.

The phase's own code-review cycle (`03-REVIEW.md`, `03-REVIEW-FIX.md`) found and fixed one real defect (WR-01: stubs printed a duplicated third stderr line via `main.go` because the stub itself also called `Fprintln`) — independently re-verified in this session: `query main` and `unlock /tmp` each produce exactly 2 stderr lines, 0 stdout bytes. One Info-level finding (IN-01: `search` has no `Long` help text documenting `--full`'s behavior) remains open by explicit maintainer/orchestrator scoping decision (Info severity, out of the `critical_warning` fix scope) — not a phase-goal blocker.

### Human Verification Required

None. Every observable truth in this phase is mechanically checkable (CLI stdout/stderr/exit-code behavior, git history shape, generated-file byte-comparison, `rg` census) and was independently re-run against the real binary and real gate commands in this session, not merely read from SUMMARY.md/03-MUTATION-LOG.md narration.

### Gaps Summary

No gaps. All 8 roadmap Success Criteria / VERB-01…VERB-08 requirements are independently confirmed against the live binary, the live gate commands (`task docs:cli:drift`, `TestEveryRegisteredFlagIsAccountedFor`, `go test ./internal/cli/...`, `go test ./test/wireoracle/...`), and a from-scratch re-run of the phase's own VERB-05/VERB-07 mutation demonstrations (planted-control census, and the `Hidden`-flip RED/revert cycle for the reference-drift gates). The one code-review defect found during the phase (WR-01) was fixed and independently re-verified here; the sole remaining finding (IN-01) is Info-severity and explicitly out of scope. The only non-blocking item is that `.planning/REQUIREMENTS.md`'s checkboxes/traceability table for VERB-01…VERB-08 have not yet been updated to reflect completion — a bookkeeping step, not a functional gap.

---

_Verified: 2026-09-16T23:59:00Z_
_Verifier: Claude (gsd-verifier)_
