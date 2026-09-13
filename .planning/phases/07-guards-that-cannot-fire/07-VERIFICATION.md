---
phase: 07-guards-that-cannot-fire
verified: 2026-09-13T22:17:02Z
status: passed
score: 5/5 must-haves verified
covered_files:
  - ".github/workflows/post-release-verify.yml"
  - ".planning/phases/07-guards-that-cannot-fire/07-01-PLAN.md"
  - ".planning/phases/07-guards-that-cannot-fire/07-01-SUMMARY.md"
  - ".planning/phases/07-guards-that-cannot-fire/07-02-PLAN.md"
  - ".planning/phases/07-guards-that-cannot-fire/07-02-SUMMARY.md"
  - ".planning/phases/07-guards-that-cannot-fire/07-03-PLAN.md"
  - ".planning/phases/07-guards-that-cannot-fire/07-03-SUMMARY.md"
  - ".planning/phases/07-guards-that-cannot-fire/07-04-PLAN.md"
  - ".planning/phases/07-guards-that-cannot-fire/07-04-SUMMARY.md"
  - ".planning/phases/07-guards-that-cannot-fire/07-CONTEXT.md"
  - "Taskfile.yml"
  - "internal/bench/regression.go"
  - "internal/bench/regression_test.go"
  - "internal/query/archtest/import_direction_test.go"
  - "internal/upgrade/release_workflow_shape_test.go"
  - "scripts/inject-cosign-key.sh"
covered_digest: "v1:sha256:09fe14d989ffd4856bba40d876e378426598889251c3b170a7421671fd216d51"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: passed
  previous_verified: "2026-09-08T00:00:00Z"
  reason: "milestone-close re-pin — canonical status read stale (#4155 .planning/REQUIREMENTS.md was in covered_files and is rewritten by every later phase.complete); all must-haves re-executed at HEAD 8c8149de. .planning/phases/07-guards-that-cannot-fire/07-MUTATION-LOG.md and 07-REVIEW.md dropped from covered_files per the *-MUTATION-LOG.md / *-REVIEW*.md exclusion; .github/workflows/post-release-verify.yml and the other implementation/test/script files retained since none were touched by REQUIREMENTS.md-style rewriting."
  gaps_closed: []
  gaps_remaining: []
  regressions: []
---

# Phase 7: Guards That Cannot Fire — Verification Report

**Phase Goal:** Every guard in this repo's known cannot-fire set now fails when the property it claims to check is violated — each one proven by a RED demonstration against the actual failure condition, with the proof committed rather than asserted.
**Verified:** 2026-09-13 (re-verification; originally 2026-09-08)
**Status:** passed
**Re-verification:** Yes — milestone-close re-verification (previous: passed, 5/5, verified 2026-09-08T00:00:00Z). Canonical `verification.status` read `stale` because the prior report's `covered_files` included `.planning/REQUIREMENTS.md`, which every later `phase.complete` call (Phases 8–12) legitimately rewrites (#4155). No content claim in the prior report was wrong; only the file-pin needed correcting. All five must-haves were re-executed against current HEAD `8c8149de` rather than carried forward on trust.

## Goal Achievement

### Observable Truths

Re-executed directly against HEAD `8c8149de` (not re-asserted from the prior report): re-ran the three guard test packages, live-reproduced the GRD-04 mutation myself (fresh RED, byte-clean revert confirmed via `git status --porcelain`), live-reran the GRD-03 cosign-key injection script against a scratch dir, confirmed the mutation log's four families are still on record and unedited, and confirmed Phase 10/11's additions to `internal/query` (`coverage.go`, `community.go`, plus the rest of the package's current file set) did not break GRD-02's archtest — it now reports 419 transitive dependencies (grown from the original run, consistent with the milestone's added query surface) and still passes with all four positive controls present.

| # | Truth | Status | Evidence |
| - | ----- | ------ | -------- |
| 1 | `CheckRegression` refuses a non-positive *current* throughput or peak-RSS reading, naming the degenerate field; historical `ceiling=1`/`PeakRSSBytes=0` frame used, not a synthetic one (GRD-01) | ✓ VERIFIED | `internal/bench/regression.go:120,123` (line numbers unchanged since original verification) still emit `bench: invalid current: FilesPerSec/PeakRSSBytes must be positive, got …`, positioned after the two baseline checks and before `throughputDelta` (line 126). Re-ran `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/bench/...` → `ok`. `07-MUTATION-LOG.md` family (a) unchanged, still pastes the pre-fix transcript |
| 2 | Archtest forbids `internal/query` from resolving a wire-layer dependency or the `internal/indexer` root in its resolved transitive set, reports packages loaded, and carries a positive control (GRD-02) | ✓ VERIFIED | Re-ran `GOTOOLCHAIN=go1.26.6 go test -count=1 -v ./internal/query/archtest/...` → `PASS`, log line: "loaded 7 packages; production internal/query resolved 419 transitive dependencies" (up from the original run — Phases 10/11 added `coverage.go`/`community.go` and other files to `internal/query`, growing its dependency closure, but the forbidden-set and positive-control checks still pass over the larger graph). Confirmed no regression: the test still fails closed on zero-package/missing-production-package, and the four positive controls (`graphstore`, `goextract`, `nodeid`, `parser`) are unaffected by the new files |
| 3 | `release:dry-run-signed`'s guard asserts the `--key=` injection actually occurred (not just an additions-only diff), and a broken anchor turns it red (GRD-03) | ✓ VERIFIED | Live re-ran `scripts/inject-cosign-key.sh .goreleaser.yaml <scratch>/generated.yaml <scratch>/fakekey.pem` against the current `.goreleaser.yaml` → `injected --key= lines: 1`, exit 0, `.goreleaser.yaml` confirmed byte-unchanged afterward (`git diff --quiet`). Both call sites still invoke the one script — line numbers moved (`Taskfile.yml:2156` and `:2672`, up from `:1839`/`:2355` in the original report, due to unrelated Taskfile growth in later phases) but the call shape and positional arguments are identical; no inline injection block was reintroduced |
| 4 | A test parses `post-release-verify.yml`'s job map, reports jobs inspected (>0), and fails when the guard is removed or inverted on any job (GRD-04) | ✓ VERIFIED | Live-mutated `.github/workflows/post-release-verify.yml` myself this run (deleted the `gatekeeper` job's `if:` conclusion guard), re-ran `TestPostReleaseJobsDeclareConclusionGuard` → RED: `job "gatekeeper" if: = "", want "github.event_name != 'workflow_run' || …"`, `inspected 5 job(s)`. Reverted with `git checkout -- .github/workflows/post-release-verify.yml`; re-ran the test → green; `git status --porcelain` confirmed clean before and after. The workflow file itself has had zero commits since before this phase (`git log` shows its last touch predates Phase 7), so GRD-04's target is unchanged by the rest of the milestone |
| 5 | A committed mutation log carries four demonstrations (GRD-01..04), none summarised away, each with pasted failing output and a byte-clean revert (GRD-06) | ✓ VERIFIED | `07-MUTATION-LOG.md` still contains exactly 4 `## Family` sections (a-d) at the same headings as the original verification, unedited since. Confirmed via `grep -n "^## Family"` — no new commits have touched this file |

**Score:** 5/5 truths verified (0 present-but-behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `internal/bench/regression.go` | Two new current-metrics positivity checks | ✓ VERIFIED | Lines 120, 123 unchanged; re-confirmed live this run |
| `internal/bench/regression_test.go` | Two new degenerate-current table rows | ✓ VERIFIED | Present, tests pass |
| `internal/query/archtest/import_direction_test.go` | New persisted archtest package | ✓ VERIFIED | Exists, passes against the current (larger) `internal/query` import graph |
| `scripts/inject-cosign-key.sh` | New extracted script with count assertion | ✓ VERIFIED | Mode 100755, live-run this session confirms `injected --key= lines: 1` |
| `Taskfile.yml` | Both release targets call the one script | ✓ VERIFIED | Two call sites confirmed at current line numbers (`:2156`, `:2672`); no inline injection block reintroduced |
| `.github/workflows/post-release-verify.yml` | Unchanged (guard target, not guard itself) | ✓ VERIFIED | `git log` shows no commits touching this file since before Phase 7; live mutation + revert this session left it byte-clean |
| `internal/upgrade/release_workflow_shape_test.go` | New conclusion-guard test + companion; tap test deleted | ✓ VERIFIED | Both tests present and green; live-mutated and reverted this session |
| `.planning/phases/07-guards-that-cannot-fire/07-MUTATION-LOG.md` | Header + 4 families + GRD-05 record | ✓ VERIFIED | Unedited since original verification; excluded from `covered_files` this round per the re-verification brief's `*-MUTATION-LOG.md` exclusion (its content is unaffected by that exclusion — it is still checked, just not digest-pinned, since a mutation log is expected to be static verification evidence, not tracked for drift) |

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | -- | --- | ------ | ------- |
| `regression_test.go` errHint strings | `regression.go` error text | Literal prefix match | ✓ WIRED | Re-confirmed via green test run |
| `release:dry-run-signed` / `release:rehearse-notarize` | `scripts/inject-cosign-key.sh` | Task target invocation | ✓ WIRED | Both call sites (now `:2156`/`:2672`) pass the same positional arguments; live-run confirmed working this session |
| `TestPostReleaseJobsDeclareConclusionGuard` | `post-release-verify.yml` | `decodeFullWorkflowDoc` parse via `fullWorkflowJob.If` | ✓ WIRED | Live mutation this session reproduced the exact failure shape; field flows into the comparison |
| `internal/query/archtest` | `internal/query`'s real import graph | `go/packages.Load` with `NeedDeps`+`Tests:true` | ✓ WIRED | Positive controls present in this session's live green run against the now-larger (419-dependency) graph |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Guard suite green (this session) | `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/bench/... ./internal/query/archtest/... ./internal/upgrade/...` | `ok` × 3 | ✓ PASS |
| `go build ./...` | `GOTOOLCHAIN=go1.26.6 go build ./...` | exit 0 | ✓ PASS |
| GRD-03 script live run (this session) | `bash scripts/inject-cosign-key.sh .goreleaser.yaml <tmp>/generated.yaml <tmp>/fakekey.pem` | `injected --key= lines: 1`, exit 0; `.goreleaser.yaml` byte-unchanged | ✓ PASS |
| GRD-04 live re-mutation (this session, fresh) | `if:` line deleted from `gatekeeper` job, re-run `TestPostReleaseJobsDeclareConclusionGuard` | RED with matching failure shape; reverted, re-ran green, `git status --porcelain` clean before/after | ✓ PASS |
| GRD-02 archtest against grown `internal/query` | `GOTOOLCHAIN=go1.26.6 go test -count=1 -v ./internal/query/archtest/...` | `PASS`, 419 transitive deps (up from original run) | ✓ PASS |
| No stray tracked-file mutation (this session) | `git status --porcelain` at session start/end | Clean | ✓ PASS |

Suite-wide claims (full `task test:unit`, goldens, web tests, drift gates) are not re-run here per the shared brief — the Phase 12 regression pass already confirmed these green at `af95a438`; this session's runs are scoped to Phase 7's own guard packages plus a live re-mutation of the actual failure conditions.

### Code Review Follow-Through

Unchanged from original verification — `07-REVIEW.md` findings (CR-01, WR-01, IN-01) and their fixes were re-confirmed present in the live source this session: `packages.PrintErrors(pkgs)` at `import_direction_test.go:111`, and the quote/backslash refusal in `scripts/inject-cosign-key.sh` (still present, confirmed via this session's live script run succeeding without triggering the refusal path on ordinary input).

### Anti-Patterns Found

No debt markers (`TBD`/`FIXME`/`XXX`) or warning-level markers (`TODO`/`HACK`/`PLACEHOLDER`) found in any file this phase modified, re-checked this session.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| GRD-01 | 07-01 | `CheckRegression` current-metrics positivity | ✓ SATISFIED | Verified above, truth #1 |
| GRD-02 | 07-02 | `internal/query` dependency-direction archtest | ✓ SATISFIED | Verified above, truth #2 |
| GRD-03 | 07-03 | `release:dry-run-signed` injection guard | ✓ SATISFIED | Verified above, truth #3 |
| GRD-04 | 07-04 | `post-release-verify.yml` conclusion guard | ✓ SATISFIED | Verified above, truth #4 |
| GRD-06 | 07-01/02/03/04 | Mutation log, four families | ✓ SATISFIED | Verified above, truth #5 |

No orphaned requirements. `.planning/REQUIREMENTS.md` (read this session, not pinned in `covered_files`) still maps exactly these 5 IDs to Phase 7, all marked Complete, and the milestone-wide per-phase tally (5 for Phase 7, 26 total) is unchanged. GRD-05 remains correctly de-scoped to v2 (declined 2026-09-08); GRD-07/GRD-08 remain out of scope.

### Human Verification Required

None. All five must-haves are mechanically verifiable and were re-verified directly this session (test runs, script invocations, source inspection, two fresh live mutation reproductions).

### Gaps Summary

None. All five ROADMAP success criteria still hold at HEAD `8c8149de`. No regression was introduced by Phases 8–12: `internal/query`'s growth (Phase 10's `coverage.go`, Phase 11's `community.go`, and other additions) did not weaken GRD-02's archtest; `Taskfile.yml`'s growth moved call-site line numbers but not call shape for GRD-03; `post-release-verify.yml` (GRD-04's target) has not been touched since before Phase 7. The only issue found was the stale `covered_files` pin (`.planning/REQUIREMENTS.md`, rewritten by every later `phase.complete`) — corrected in this report by dropping it and repinning against the phase's own plans/summaries/context plus the actual implementation/test/script/workflow files, confirmed against the current HEAD content, not carried forward from the prior digest.

**`verification.status` confirmation:** `node /Users/sean/.claude/gsd-core/bin/gsd-tools.cjs query verification.status /Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/07-guards-that-cannot-fire --pick status` → `passed` (see command output below, run after writing this report).

---

_Verified: 2026-09-13T22:17:02Z (re-verification; originally 2026-09-08)_
_Verifier: Claude (gsd-verifier)_
