---
phase: 4
cycle: 2
reviewers: [codex, opencode]
reviewed_at: 2026-08-11T10:45:00-04:00
plans_reviewed: [04-01-PLAN.md, 04-02-PLAN.md, 04-03-PLAN.md, 04-04-PLAN.md, 04-05-PLAN.md, 04-06-PLAN.md]
source_grounding: true
drift_guard_authority: intel
prior_cycle: dbdc68f^ (cycle-1 REVIEWS.md text retained in git history)
---

# Cross-AI Plan Review — Phase 4 (Cycle 2)

Both lanes ran with repo access and returned source-grounded reviews (no `[reviewed-without-repo-access]`
marker, neither stubbed). Reviewers were given the cycle-2 mandate (verify the 19 cycle-1 dispositions by
measurement, cover the three dimensions the internal checker skipped, hunt for fixes that merely re-arm the
same trap at a new threshold), the CONTEXT.md supersession ground rules (D-01 Caskroom-not-Cellar, D-08 seam
assertion, D-09 `--check` step-aside, D-04R linuxbrew) and repo rule `84d1gfpywd`. Neither raised a
stale-ROADMAP finding, so the ground rules held. `04-RESEARCH.md` and `04-VALIDATION.md` were both supplied
in the prompt this cycle.

## Codex Review

## Summary

Cycle 2 materially improves the plans: the numeric baselines now measure the right quantities, the Homebrew detector uses a root-preserving walk, `EvalSymlinks` has one consistent failure contract, and the previously vacuous test gates are now count-asserted. I measured 4 anchored cosign invocations versus the proposed floor of 5, 10 unanchored matching lines, 6 identity literals across all five files, and 0 identity literals in the current `verify:self-upgrade` region.

However, plan 04-06 still risks altering a maintainer’s pre-existing Homebrew state: it unconditionally untaps an existing tap and removes potentially pre-existing man pages. Plan 04-02’s revised freshness mechanism also cannot implement its promised size fallback because its baseline records only mtimes. These require revisions before execution. Verdict: **revisions required**.

Repository tests could not run because the read-only environment prevents Go from creating its temporary work directory. CodeGraph likewise could not acquire `.codegraph/store/LOCK`; source and shell measurements were used instead.

## Cycle-1 disposition verification

| # | Finding | Status | Verification |
|---|---|---|---|
| 1 | Cosign invocation floor was pre-satisfied | RESOLVED | Current source has 4 anchored invocations at [Taskfile.yml:1492](/Volumes/Code/github.com/seanb4t/codegraph-go/Taskfile.yml:1492), [1494](/Volumes/Code/github.com/seanb4t/codegraph-go/Taskfile.yml:1494), [2541](/Volumes/Code/github.com/seanb4t/codegraph-go/Taskfile.yml:2541), and [3087](/Volumes/Code/github.com/seanb4t/codegraph-go/Taskfile.yml:3087). Plan floor is 5 at [04-05-PLAN.md:178](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-05-PLAN.md:178), so it is RED today. Unanchored count measured 10. |
| 2 | Identity floor could not detect loss of Task 1 | RESOLVED | Baseline measured as 6: Taskfile ×2 plus one each in README, RELEASE, SECURITY, and RELEASE-PROCEDURES. Plan requires 7 and specifically requires a literal within `verify:self-upgrade` at [04-05-PLAN.md:267](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-05-PLAN.md:267). Current region has 0. |
| 3 | Absolute paths lost by split-and-rejoin | RESOLVED | Plan now mandates a `filepath.Dir` ancestry walk and explicitly forbids reconstruction at [04-01-PLAN.md:226](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-01-PLAN.md:226), with `filepath.IsAbs` assertions in every detected row at [04-01-PLAN.md:378](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-01-PLAN.md:378). |
| 4 | No mechanical rollback around Caskroom mutation | PARTIALLY RESOLVED | A single script and pre-mutation trap are now required at [04-06-PLAN.md:211](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-06-PLAN.md:211). New baseline-restoration defects remain; see concerns. |
| 5 | Arbitrary pre-existing cask cannot be restored | RESOLVED | The acceptance run now halts if a cask is already installed at [04-06-PLAN.md:112](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-06-PLAN.md:112). |
| 6 | `rg -q 'ok\|PASS'` passed with no tests | RESOLVED | Replaced by separate named-PASS counts at [04-02-PLAN.md:286](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-02-PLAN.md:286). Existing source defines exactly five `TestTaskfile*` tests at [taskfile_shape_test.go:936](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/taskfile_shape_test.go:936) through [1099](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/taskfile_shape_test.go:1099), plus one workflow-body test at [1342](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/taskfile_shape_test.go:1342). |
| 7 | Multi-pattern `rg -c` did not prove set membership | RESOLVED | Relevant assertions are now independent, including ROADMAP seam tokens and evidence disposition rows; see [04-03-PLAN.md:200](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-03-PLAN.md:200) and [04-06-PLAN.md:356](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-06-PLAN.md:356). |
| 8 | Finite corpus overclaimed language equivalence | RESOLVED | Plan consistently calls this “selected boundary-case parity” and records the residual limitation at [04-05-PLAN.md:474](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-05-PLAN.md:474). |
| 9 | Contradictory `EvalSymlinks` behavior | RESOLVED | One rule now applies: any error returns not-detected immediately, with dangling and loop rows at [04-01-PLAN.md:371](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-01-PLAN.md:371). |
| 10 | Eyeballed ordering assertions | RESOLVED | Replaced by scripted offsets in plan 04-01 and plan 04-05; for example [04-05-PLAN.md:178](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-05-PLAN.md:178). |
| 11 | `Time.now - 1` was a weak freshness oracle | PARTIALLY RESOLVED | Wall-clock comparison was replaced by per-path snapshots at [04-02-PLAN.md:150](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-02-PLAN.md:150), but the promised size fallback is not implementable as specified. |
| 12 | Identity extractor was format-fragile | RESOLVED | The plan requires provenance fields, unmatched-flag reporting, a documented parsing contract, and self-tests at [04-05-PLAN.md:292](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-05-PLAN.md:292). |
| 13 | Pointer no-drift claim ignored the CLI copy | RESOLVED | Plan 04-01 now limits the shared-builder guarantee to two surfaces; plan 04-04 explicitly treats CLI help as an independent copy at [04-04-PLAN.md:83](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-04-PLAN.md:83). |
| 14 | GoReleaser test pattern did not read `.goreleaser.yaml` | RESOLVED | Retargeted to the five `TestHomebrewCask*` tests; those tests exist at [goreleaser_shape_test.go:1072](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/goreleaser_shape_test.go:1072) through [1268](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/goreleaser_shape_test.go:1268). |
| 15 | Restoration gate had brew-absent and environment escape hatches | PARTIALLY RESOLVED | `command -v brew` and a positive receipt are now mandatory at [04-06-PLAN.md:268](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-06-PLAN.md:268), and `CODEGRAPH_CASK_PREEXISTING` is gone. Exact baseline restoration is still incomplete. |
| 16 | D-08 hash proxy was reintroduced | RESOLVED | Removed; plan explicitly records why the four seam assertions are stronger at [04-01-PLAN.md:207](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-01-PLAN.md:207). |
| 17 | Evidence gate did not associate IDs with dispositions | RESOLVED | The plan fixes the table shape and asserts one anchored row per ID at [04-06-PLAN.md:342](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-06-PLAN.md:342). |
| 18 | Conflicting man-page marker creation points | RESOLVED | One pre-install marker and two consumers are now specified at [04-02-PLAN.md:249](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-02-PLAN.md:249). |
| 19 | `--force` evidence demanded byte-identical messages | RESOLVED | Relaxed to the actual property: both contain the same resolved-path pointer sentence, at [04-06-PLAN.md:278](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-06-PLAN.md:278). |

## Strengths

- Plan 04-01 follows the existing orchestration seams. The current `Options` contains the four injectable functions at [upgrade.go:69](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/upgrade.go:69), while network resolution currently begins at [upgrade.go:88](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/upgrade.go:88). The prescribed detector insertion above that point genuinely proves zero-network refusal.

- The cosign floor is now calibrated correctly: 4 current shell invocations versus a post-task floor of 5. This is a meaningful RED-before-GREEN gate, not just a larger arbitrary count.

- The identity scan set is complete for current published/executed restatements: [README.md:59](/Volumes/Code/github.com/seanb4t/codegraph-go/README.md:59), [docs/RELEASE.md:62](/Volumes/Code/github.com/seanb4t/codegraph-go/docs/RELEASE.md:62), [SECURITY.md:35](/Volumes/Code/github.com/seanb4t/codegraph-go/SECURITY.md:35), [docs/RELEASE-PROCEDURES.md:238](/Volumes/Code/github.com/seanb4t/codegraph-go/docs/RELEASE-PROCEDURES.md:238), and two Taskfile occurrences.

- Research responsibilities are honored: detection stays in `internal/upgrade`, uses only the standard library, does not shell out to Homebrew, and the Linux-cask open question was resolved through D-04R rather than silently ignored.

- All 16 tasks carry automated verification, so sampling continuity holds. No three consecutive tasks lack an automated command.

## Concerns

- **HIGH — NEW:** Plan 04-06 does not restore a pre-existing tap. Task 1 explicitly allows the tap to pre-exist and records `brew tap | rg seanb4t` as baseline at [04-06-PLAN.md:132](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-06-PLAN.md:132), but the trap unconditionally runs `brew untap seanb4t/tap` at [04-06-PLAN.md:230](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-06-PLAN.md:230). A maintainer who already had the tap finishes in a different state.

- **HIGH — NEW:** Plan 04-06 can delete pre-existing man pages. Task 1 deliberately records the initial `codegraph` man-page count at [04-06-PLAN.md:133](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-06-PLAN.md:133), but the cleanup performs `brew uninstall --cask codegraph`, whose existing hook removes generated pages. The plan preserves neither their bytes nor metadata, despite promising the four final probes will match the baseline at [04-06-PLAN.md:289](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-06-PLAN.md:289). This is especially relevant because the phase’s own evidence records orphaned man pages as a real prior state.

- **MEDIUM — NEW:** The freshness algorithm promises a size fallback without recording baseline sizes. `man_pages_before` is specified as `path → File.mtime` at [04-02-PLAN.md:153](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-02-PLAN.md:153), but [04-02-PLAN.md:179](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-02-PLAN.md:179) says a changed size should also count as fresh. There is no prior size to compare against.

- **MEDIUM — NEW:** `trap restore_and_cleanup EXIT INT TERM` can invoke cleanup twice. A trapped `INT` or `TERM` calls the function, and a later shell exit invokes the same function again. That conflicts with the plan’s “do not call twice” intent at [04-06-PLAN.md:240](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-06-PLAN.md:240) and could make a second `brew uninstall`/`untap` obscure the successful first restoration.

- **LOW — NEW:** Plan 04-05’s temp-directory statement is backwards. It requires both `mktemp` calls before installing the trap and claims this prevents a leak, but failure of the second `mktemp` under `set -e` leaks the first directory. See [04-05-PLAN.md:143](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-05-PLAN.md:143).

- **MEDIUM — NEW:** `04-VALIDATION.md` does not consistently mirror the corresponding task commands despite the explicit cycle-2 requirement. Examples:

  - Row 04-02-02 omits the plan’s `schema=3` assertion ([VALIDATION:47](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-VALIDATION.md:47) versus [04-02-PLAN.md:286](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-02-PLAN.md:286)).
  - Row 04-03-01 omits the positive `Cellar` count from the task command ([VALIDATION:49](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-VALIDATION.md:49) versus [04-03-PLAN.md:192](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-03-PLAN.md:192)).
  - Row 04-04-02 omits the `brew trust` preservation check ([VALIDATION:52](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-VALIDATION.md:52) versus [04-04-PLAN.md:200](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-04-PLAN.md:200)).
  - Row 04-06-03 omits the plan command’s per-leg presence checks ([VALIDATION:58](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-VALIDATION.md:58) versus [04-06-PLAN.md:356](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-06-PLAN.md:356)).

## Suggestions

1. In plan 04-06, record whether `seanb4t/tap` existed before the run. Untap only when this plan added it. Include that state in the restoration receipt.

2. Either halt when any `codegraph*.1` page exists before acceptance, or preserve and restore every matching page byte-for-byte with modes and mtimes. Do not rely only on the final count.

3. Change `man_pages_before` to store both mtime and size, for example `{mtime:, size:}`, and compare both fields after generation. Alternatively, remove the claimed size fallback.

4. Make `restore_and_cleanup` idempotent with a guard, and use signal handlers that exit after invoking cleanup while disabling traps before cleanup begins.

5. Install cleanup immediately after creating `DL_DIR`, then add `SIG_DIR` to state the cleanup function understands. This covers failure of the second `mktemp`.

6. Synchronize every `04-VALIDATION.md` automated command byte-for-byte with its corresponding task command, or explicitly document why a validation row intentionally uses a reduced sampling gate.

## Risk Assessment

**HIGH.** The production implementation plan is substantially improved and the cycle-1 numeric/vacuity defects are mostly closed. The remaining risk is concentrated in the real-workstation acceptance plan: as written, it can remove a pre-existing tap and pre-existing man pages while claiming exact restoration. Those are material, externally visible state changes, so execution should wait for the rollback protocol to be corrected.

---

## OpenCode Review


# Cross-AI Plan Review — Phase 4, Cycle 2

## Summary

The revision is substantively honest: every baseline I was asked to verify independently reproduced exactly as stated, and the floors that cycle 1 found pre-satisfied are now RED against the current tree. I re-measured all five load-bearing numbers rather than trusting the plans' assertions — `cosign verify-blob` real invocations (4 anchored / 10 unanchored lines, `Taskfile.yml:1492,1494,2541,3087`), identity literals (6 across 5 files, confirmed flag/literal line pairs at `Taskfile.yml:2544-2545,3090-3091`, `README.md:59-60`, `docs/RELEASE.md:62-63`, `SECURITY.md:35-36`, `docs/RELEASE-PROCEDURES.md:238-239`), the named-PASS baselines (`TestTaskfile` 5, `TestWorkflowRunBodiesInvokeTask` 1, `TestHomebrewCask` 5), the `verify:self-upgrade` region contents (0 `SIG_DIR`, 0 `cosign`, 1 `chmod +x` today), and the stale-doc citations (`README.md:81`, `docs/RELEASE.md:540-541`, `Taskfile.yml:1671`'s rehearsal refusal, the sentinel write at `.goreleaser.yaml:579` and removal at `:624`). One NEW internal inconsistency exists in `04-05` Task 3: Mutation 3 cannot produce the outcome its own acceptance criterion demands, because the recalibrated total floor fires before the region-scoped assertion. That is a one-paragraph fix. Everything else verifies. **Verdict: revisions required — one MEDIUM fix in 04-05 Task 3, plus two LOW clarifications. No HIGH findings remain.**

## Cycle-1 disposition verification

| # | Finding | Disposition | Evidence |
|---|---|---|---|
| 1 | `cosign verify-blob` floor pre-satisfied, baseline "2" wrong | **RESOLVED** | Measured: 4 anchored invocations (`Taskfile.yml:1492,1494,2541,3087`), 10 unanchored lines. New gate `rg -c -e '^\s*cosign verify-blob' ≥ 5` exceeds the true baseline — RED today. True baseline recorded in the criterion |
| 2 | Identity-literal floor 4 already met | **RESOLVED** | Extractor-yield derivation reproduced: exactly 6 literals across 5 files today (I confirmed all 6 flag→literal line pairs). Floor 7 > baseline 6, plus the region-scoped requirement (region measured: 0 cosign occurrences today) |
| 3 | Root-losing split-and-rejoin in `brew.go` algorithm | **RESOLVED** | `04-01` Task 1 action step 2 prescribes a parent walk via `filepath.Dir` terminating on `parent == dir`; acceptance gates `strings\.Split` == 0 paired with `filepath\.Dir` ≥ 1, plus an executing `filepath.IsAbs(InstallDir)` assertion in every detected row |
| 4 | No mechanical rollback in acceptance run | **RESOLVED** | `04-06` Task 2 mandates one script, `trap restore_and_cleanup EXIT INT TERM` installed at step 3 before the first mutating byte (step 4), bytes-first-then-cask ordering, `set -uo pipefail` with deliberate `-e` omission |
| 5 | Pre-existing cask restoration unimplementable | **RESOLVED** | Blocking precondition in `04-06` Task 1 halts if `brew list --cask --versions codegraph` reports anything; the "reinstall prior version" branch is deleted. Matches the repo's own rehearsal posture — confirmed at `Taskfile.yml:1671` (`'! brew list --cask codegraph …'`) |
| 6 | Vacuous `rg -q 'ok\|PASS'` gate | **RESOLVED** | `04-02` Task 2 now carries two independent count-asserted named-PASS gates; baselines measured today: `TestTaskfile` = 5, `TestWorkflowRunBodiesInvokeTask` = 1 — both match the plan's recorded numbers |
| 7 | `rg -c` multi-`-e` set-membership misuse | **RESOLVED** | All instances I re-checked are split per-token: `Options\.download`/`Options\.swap` independently asserted with escaped dots (04-03), `Leg 1/2/3` and `UPGR-01/02/03` as per-token loops with `-F` (04-06), `^\| UPGR-0N \|` row-shape anchors (04-06 Task 3) |
| 8 | Language-equivalence overclaim | **RESOLVED** | Test names, `must_haves`, and T-04-16 all say "boundary-case parity"; residual (drift outside probed boundaries) named. Prefix `TestCosignIdentityPolicy` retained so the `-eq 2` gate survives |
| 9 | `EvalSymlinks` contract self-contradictory | **RESOLVED** | Single rule (error ⇒ not-detected, immediate return) stated identically in Task 1 action, T-04-03, and `must_haves.truths`; two executing rows (dangling, loop) with per-token name assertions |
| 10 | Eyeball line-number comparisons | **RESOLVED** | `node -e` offset assertions with named throws in 04-01 Tasks 1/3; pattern traced — `s.indexOf("detectBrewManaged")` vs `s.indexOf("resolveLatest(releaseRepoSlug)")` targets real strings (`internal/upgrade/upgrade.go:95`) |
| 11 | `Time.now - 1` weak oracle | **RESOLVED** | Replaced by `man_pages_before` path→mtime snapshot; `Time\.now` absence gate is satisfiable — the only current occurrence (`.goreleaser.yaml:586`) lives inside the sentinel heredoc being deleted |
| 12 | Extractor format-fragility | **RESOLVED** | Parse contract documented, `File`/`Line`/`Value` struct, misattribution guard, phantom-literal self-test |
| 13 | Third unpinned pointer-message copy | **RESOLVED** (with honest residual) | 04-01 `key_links` recalibrated to "exactly two surfaces"; 04-04 cross-surface gate widened to four files including `brew.go`; surrounding-wording drift explicitly accepted rather than claimed away |
| 14 | `^TestGoreleaser` mis-scoped | **RESOLVED** | Confirmed the mis-scope independently: `^TestGoreleaser` matches only `TestGoreleaserPinParity`. Retargeted to `^TestHomebrewCask`, measured 5 PASS today — matches the plan's recorded baseline |
| 15 | Restoration gate escape hatches | **RESOLVED** | `CODEGRAPH_CASK_PREEXISTING` short-circuit deleted; `command -v brew` asserted first; positive receipt floor (`RESTORE_VERDICT=ok` + equal before/after sha256 via `sort -u | wc -l == 1`) |
| 16 | sha256 proxy reintroduced | **RESOLVED** | Removed from 04-01 Task 1 with a `review_note` recording why |
| 17 | ID↔disposition association unenforced | **RESOLVED** | Fixed table shape (`## Carried assumption dispositions`, ID as first cell) + three independent `^\| UPGR-0N \|` == 1 assertions |
| 18 | Two marker-creation points | **RESOLVED** | One `mktemp` (exactly-1 gate) paired with ≥4 total references; creation-before-install offset assertion |
| 19 | `--force` byte-identity too strict | **RESOLVED** | Relaxed to same-resolved-path + same-literal containment, matching the unit-test bar |

**19/19 verified against the repo. No REGRESSED and no NOT RESOLVED rows.**

## Strengths

- Every floor I re-derived independently matched the plan's stated baseline: 4 cosign invocations, 6 identity literals, 5/1/5 named-PASS counts, 0 `SIG_DIR` in the `verify:self-upgrade` region, `raise "codegraph cask post-install` = 2 today (04-02's ≥3 floor is RED today, `system_command … ["man", man_dir]` exactly 1 at `.goreleaser.yaml:551` ✓).
- The `04-05` region slice (`verify:self-upgrade:` at `Taskfile.yml:2554` → `verify:gatekeeper:` at `:2751`) is anchored on target keys whose first occurrences are the keys themselves — the probe and the production ordering assertion share the same anchors, closing the "second independent matcher" hole.
- 04-02's line citations trace: Step 5b sentinel block at `Taskfile.yml:2006-2030`, idempotency at `:2072`, symmetry at `:2091-2095`, evidence echo `schema=2` at `:2215` — all confirmed. The false EVIDENCE claim exists exactly where cited (`03-EVIDENCE.md:831,1039`) and the self-contradicting "30 orphaned" evidence at `:815`.
- VALIDATION.md rows match plan tasks one-for-one (spot-checked 04-01-02 `-ge 16`, 04-02-02 dual named-PASS gates, 04-05-01 `≥ 5`, 04-06-02 receipt gate — identical commands in both documents). Sampling continuity: 16/16 tasks carry an automated verify. Pending-todo arithmetic is consistent across plans (10 → 8 after 04-02 → 7 after 04-05).
- RESEARCH.md's two open questions are dispositioned: OQ1 resolved by D-04R + the 16-row table; OQ2 resolved as "no branch needed" with the single `brew upgrade codegraph` literal. Responsibility map honored (detection in `internal/upgrade`, D-12).

## Concerns

- **[MEDIUM — NEW] `04-05` Task 3, Mutation 3 is internally inconsistent with Task 2's check ordering.** Task 2 applies checks in order: (1) total < 7 → `t.Fatalf`, then (3) region-scoped → `t.Fatalf`. Mutation 3 deletes *only* the `verify:self-upgrade` literal, leaving 6 total — so **check 1 (total floor) fires before the region assertion is ever reached**. Yet Task 3's acceptance criterion demands: "Mutation 3 must show the REGION-SCOPED assertion firing, not the total floor — if the total floor is what fired, the guard has not been recalibrated correctly." As written, the criterion is unsatisfiable: any single-literal deletion trips the total floor first, and any mutation that preserves the total at ≥7 while emptying the region (e.g., moving the literal elsewhere in `Taskfile.yml`) would be a contrived non-regression. The fix is to reorder Task 2's checks so the region-scoped assertion runs *before* the total floor (region first, then total, then per-file) — then deleting Task 1's literal fires the region assertion exactly as the acceptance criterion requires, and the total floor remains RED-today as a separate guard.
- **[LOW — NEW] `04-01` Task 3 action text vs. its own "exactly 1 call site" criterion.** The action says "`fmt.Fprintln(out, ...) the string returned by `brewPointerMessage(inst)`" and "return the wrapped error Task 1 introduced" — read naively that is two call sites, while the acceptance criterion enforces `rg -c 'brewPointerMessage' internal/upgrade/upgrade.go == 1`. The criterion forces the right shape (compute `msg` once, branch on it), but the action should say so explicitly so the executor doesn't write two call sites and then churn on the failing gate.
- **[LOW — CARRIED in spirit] `04-02` Task 1 line-range citation drift.** The plan cites the sentinel write as lines 570-588; the actual `atomic_write` call is at `:579` and an `atomic_write`-mentioning comment sits at `:592`, *outside* the cited deletion range. The `rg -c 'atomic_write' == 0` acceptance gate makes this self-correcting, but Edit 5 should explicitly own removing the `:592` mention so the executor doesn't hunt for which edit was supposed to cover it.

## Suggestions

1. **04-05 Task 2 `<action>`:** reorder the numbered checks — region-scoped assertion (current step 3) before the total floor (current step 1). **04-05 Task 3 `<action>`:** add one sentence: "With the region check ordered first, deleting only the `verify:self-upgrade` literal fires the region assertion before the total floor is evaluated."
2. **04-01 Task 3 `<action>`:** "Compute `msg := brewPointerMessage(inst)` once at the top of the branch; `fmt.Fprintln(out, msg)` under `opts.Check`, `fmt.Errorf("upgrade: %s", msg)` otherwise — one call site by construction."
3. **04-02 Task 1 Edit 5:** append "…including the `atomic_write` mention in the trailing comment at line 592, so the `atomic_write == 0` gate is satisfied."

## Risk Assessment

**LOW-to-MEDIUM.** Cycle 1's six HIGHs are all verifiably closed by measurement, not by plan-text assertion; the executable gates are RED today against the current tree and GREEN only after the intended work. The single remaining MEDIUM is a self-inconsistency *inside a self-test protocol* — the worst case is Task 3 of 04-05 reporting a confusing failure and the executor reordering two checks, not a silent vacuous gate. No new brittle-count-for-brittle-count substitutions were introduced: the two surviving numeric floors (`cosign verify-blob` ≥ 5, literals ≥ 7) are anchored on real quantities with recorded baselines, and everything else moved to region-scoped or per-token set-membership assertions. After the three suggestions land, this plan set is ready to execute.

---

## Consensus Summary

Both lanes independently re-derived every load-bearing number and both reached the same verdict on the
cycle-1 fixes: **the numeric-floor and vacuous-gate defect class is genuinely closed.** Cycle 1's dominant
failure mode — a floor already satisfied before the task meant to raise it — does not recur in any of the
surviving floors. Both lanes measured the same baselines, and the orchestrator reproduced all of them:

| Claim under verification | Plan's stated baseline | Measured | Floor | RED today? |
|---|---|---|---|---|
| `rg -c '^\s*cosign verify-blob' Taskfile.yml` | 4 real invocations / 10 unanchored lines | **4** (`:1492`, `:1494`, `:2541`, `:3087`); unanchored **10** | `-ge 5` | **yes** |
| identity literals across five files | 6 | **6** (`Taskfile.yml:2545`, `:3091`, `README.md:60`, `docs/RELEASE.md:63`, `SECURITY.md:36`, `docs/RELEASE-PROCEDURES.md:239`) | `-ge 7` | **yes** |
| identity literals in the `verify:self-upgrade` region | 0 | **0** | `>= 1` | **yes** |
| `--- PASS: TestTaskfile` | 5 | **5** | `-ge 5` | preservation gate |
| `--- PASS: TestWorkflowRunBodiesInvokeTask` | 1 | **1** | `-eq 1` | preservation gate |
| `--- PASS: TestHomebrewCask` | 5 | **5** | `-ge 5` | preservation gate |
| `.planning/todos/pending/` | 10 → 8 → 7 | **10** | consistent | n/a |

The five named cycle-2 verification targets all hold in executable PLAN.md content:

- **`04-05` T1** — floor `-ge 5` against a measured baseline of 4. Exceeds. The anchored matcher
  (`^\s*cosign verify-blob`) isolates real shell invocations rather than the 10 lines the unanchored
  pattern matches.
- **`04-05` T2** — scan set is genuinely five files, floor is genuinely 7 against a measured 6, and the
  region-scoped requirement is present. Both lanes confirmed the region probe shares Task 1's own
  delimiters (`verify:self-upgrade:` → `verify:gatekeeper:`), so the "second independent matcher" hole is
  closed. **But see the HIGH below on check ordering** — the region assertion is unreachable in the one
  scenario it exists to catch.
- **`04-02` T2** — the vacuous `rg -q 'ok|PASS'` is gone; replaced by two independent count-asserted
  named-PASS gates (`04-02-PLAN.md:286`).
- **`04-06` T2** — `trap restore_and_cleanup EXIT INT TERM` is mandated before the first mutating byte
  (`04-06-PLAN.md:217`, `:233`), both escape hatches are gone, and a positive receipt replaces the
  all-negative check. **But see the HIGH below** — the gate carrying that fix is structurally unparseable.
- **`04-01` T1** — root-preserving parent walk via `filepath.Dir` with `parent == dir` termination
  (`04-01-PLAN.md:241`), backstopped by a paired source gate (`strings.Split` absent AND `filepath.Dir`
  present, `:331`) and `filepath.IsAbs` assertions in every detected row. `EvalSymlinks` contract
  reconciled to one rule (error ⇒ not detected) across action, T-04-03 and `must_haves.truths`.

The three dimensions the internal checker skipped are covered and clean:

- **Research resolution** — `04-RESEARCH.md`'s two open questions are dispositioned. OQ1 (linuxbrew
  Caskroom coverage) is resolved via D-04R and honored by 8 detected rows in `04-01` and the criterion-3
  amendment in `04-03`. OQ2 (cask-vs-formula message branch) is resolved as "no branch needed"; no formula
  branch appears anywhere in the plans. The Architectural Responsibility Map is honored — detection lives
  in `internal/upgrade`, the refusal branch in `Run()`, the CLI stays a thin wrapper, sentinel removal
  stays in `.goreleaser.yaml`/`Taskfile.yml`, and no new dependency is introduced.
- **Nyquist compliance** — 16 map rows for 16 tasks (3+3+2+2+3+3), and **all 16 tasks carry an automated
  verify**. Sampling continuity holds trivially; there is no run of even one task without automated
  coverage, let alone three. *However*, the automated commands themselves diverge (see MEDIUM below), and
  two of them are structurally unreachable (see HIGH below).
- **Architectural tier compliance** — no tier violation found by either lane.

**No new brittle-count-for-brittle-count substitution was introduced.** The two surviving numeric floors are
anchored on the real quantity with a recorded measured baseline; everything else moved to region-scoped or
per-token set-membership assertions. That specific anti-pattern the cycle-2 mandate warned about did not
materialize.

The remaining risk is concentrated almost entirely in **`04-06`, the real-workstation acceptance plan**, plus
one self-inconsistency in `04-05`'s mutation protocol.

### Agreed Strengths

- Every floor both lanes re-derived matched the plan's stated baseline exactly. The baselines are honest
  measurements, not inherited numbers — the plans even record *why* the cycle-1 count of "4" was a different
  quantity from the orchestrator's flag-occurrence count (`04-05-PLAN.md:239-269`).
- The `04-01` detector insertion point is correct against real code: `Options` exposes the four injectable
  seams at `internal/upgrade/upgrade.go:69`, and network resolution begins at `:88`, so a detector branch
  above that point genuinely proves zero-network refusal.
- The identity scan set is now complete for every published/executed restatement in the repository. Cycle 1's
  three-file list missed `SECURITY.md:36` and `docs/RELEASE-PROCEDURES.md:239`, which made the plan's own
  must-have ("**every** identity literal published or executed by this repository") false as written. Both
  are now in scope.
- `04-06`'s blocking precondition (halt if a codegraph cask is already installed) matches the repository's
  own rehearsal posture at `Taskfile.yml:1671`, and correctly abandons the unimplementable "reinstall the
  recorded prior version" promise rather than papering over it.
- Residual limitations are named rather than claimed away: the boundary-case-parity framing replacing the
  language-equivalence overclaim, the third independent pointer-message literal in `internal/cli/upgrade.go`
  that `brewPointerMessage` cannot reach because it is unexported, and the unproved end-to-end leg in
  `04-06`.

### Agreed Concerns

Both lanes independently reached "revisions required." No finding below is a re-litigation of a cycle-1
finding that was properly closed.

**HIGH — `04-06-PLAN.md` Tasks 2 and 3 have unclosed `<automated>` tags.** *(NEW; orchestrator-measured,
missed by both lanes.)* Precise scan across all six plans and all structural tags:

```
$ for tag in automated human-check verify acceptance_criteria action done read_first files name; do
    for f in .planning/phases/04-*/04-0*-PLAN.md; do
      o=$(rg -o -e "^\s*<$tag>" $f | wc -l); c=$(rg -o -e "</$tag>\s*$" $f | wc -l)
      [ "$o" != "$c" ] && echo "IMBALANCE <$tag> $(basename $f): open=$o close=$c"
    done; done
IMBALANCE <automated> 04-06-PLAN.md: open=3 close=1
```

Opens at `04-06-PLAN.md:164`, `:268`, `:356`; only `:164`'s is closed. `:268` runs straight into
`<human-check>` and `:356` runs straight into `<human-check>` with no `</automated>` between them. Cycle 1's
version of this file was balanced (3 open / 3 close at `dbdc68f^`), and every other plan in the phase — and
all five phase-3 plans — is balanced. **The revision introduced this.**

The consequence is precisely inverted from the intent: the two gates affected are *the two the revision added
to close cycle-1 findings #15 and #7*. An executor extracting `<automated>…</automated>` either finds no
command for those tasks (the gate silently does not run) or greedily swallows the `<human-check>` prose and
`</verify>` into the shell command. Either way the positive restore-receipt assertion and the per-ID
disposition assertion — the two most safety-critical gates in the phase — are unreachable. This downgrades
cycle-1 findings **#7 (the `04-06` instance) and #15 to PARTIALLY RESOLVED**: the corrected text exists, but
not in a parseable executable position.

*Fix:* add `</automated>` at the end of the command on `04-06-PLAN.md:268` and `:356`.

**HIGH — `04-06` does not restore a pre-existing tap.** *(NEW; Codex.)* Task 1 explicitly records
`brew tap | rg seanb4t` "(tolerating no match)" as a baseline **and restoration target**
(`04-06-PLAN.md:137-139`), but the trap's step (c) unconditionally runs `brew untap seanb4t/tap`
(`:230-231`). A maintainer who already had the tap ends in a different state than they started, and Task 2's
own acceptance criterion — "the four restoration probes … each match their Task 1 baseline values"
(`:295-297`) — fails by construction. *Measured on this host today: no `seanb4t` tap present, so the defect
does not fire here right now; it is a durable plan defect, not a live one.*

*Fix:* record whether the tap pre-existed and untap only if this run added it; carry that state in the
restore receipt.

**HIGH — `04-06`'s cleanup can delete pre-existing man pages.** *(NEW; Codex.)* Task 1 records
`ls "$(brew --prefix)/share/man/man1/" | rg -c '^codegraph'` "(tolerating zero)" as a baseline probe
(`04-06-PLAN.md:139`), but the trap's `brew uninstall --cask codegraph` fires the cask's uninstall hook,
which is `Dir.glob((man_dir/"codegraph*.1").to_s).each { |f| FileUtils.rm_f(f) }` at `.goreleaser.yaml:621`
— an unconditional glob removal over the *shared* man directory. It deletes pages this run never created.
This is not hypothetical: `03-EVIDENCE.md` records orphaned `codegraph*.1` pages as a real prior state on
this project (the "30 orphaned" claim `04-02` Task 3 is amending), and the cask precondition only excludes a
*cask*, not orphaned pages. *Measured on this host today: 0 matching pages, so again a durable defect rather
than a live one.*

*Fix:* either halt when any `codegraph*.1` page exists before acceptance, or preserve and restore every
matching page byte-for-byte with modes and mtimes. The final count probe alone cannot distinguish
"restored" from "deleted the maintainer's".

**HIGH — `04-05` Task 3's Mutation 3 acceptance criterion is unsatisfiable as written.** *(NEW; OpenCode,
severity escalated from MEDIUM by the orchestrator — rationale below.)* Task 2's `<action>` applies the
extractor's checks in a fixed order (`04-05-PLAN.md:337-354`): step **1** is `t.Fatalf` if the TOTAL is below
7; step **3** is `t.Fatalf` if no literal falls inside the `verify:self-upgrade` region. Mutation 3 deletes
*only* the region literal, taking the total from 7 to 6 — so **step 1 fires first and step 3 is never
reached**. Yet Task 3's acceptance criterion demands the opposite (`:459`):

> Mutation 3 must show the REGION-SCOPED assertion firing, not the total floor — if the total floor is what
> fired, the guard has not been recalibrated correctly.

Escalated to HIGH because Mutation 3 *is* the designated proof that cycle-1 HIGH #2 was actually closed —
without it, the recalibration is asserted rather than demonstrated — and because the obvious repair an
executor would reach for under a failing criterion (lower the total floor to 6 so the region check runs) is
exactly the pre-satisfied-floor defect cycle 1 filed. The plan-text fix is one paragraph and does not have
that hazard.

*Fix:* reorder Task 2's checks so the region-scoped assertion precedes the total floor, and add a sentence to
Task 3 noting that with that ordering, deleting the region literal fires the region assertion first.

**MEDIUM — the promised man-page size fallback is unimplementable as specified.** *(NEW; Codex.)*
`04-02-PLAN.md:153` specifies `man_pages_before` as a Hash mapping each path to `File.mtime` — mtime only.
`:179-183` then says a page "whose SIZE changed" should count as fresh, as the stated mitigation for coarse
mtime granularity. There is no baseline size to compare against, so the residual the plan claims to handle
is not handled. *Fix:* store `{mtime:, size:}` and compare both, or delete the size-fallback claim.

**MEDIUM — `trap restore_and_cleanup EXIT INT TERM` can invoke cleanup twice.** *(NEW; Codex.)* A trapped
`INT` or `TERM` runs the handler and, because the handler does not exit, the later shell exit runs it again.
That contradicts the plan's own instruction at `04-06-PLAN.md:237-238` ("Do not also call the trap function
directly — a double restoration would report a confusing second verdict"), and a second
`brew uninstall`/`brew untap` would obscure a successful first restoration in the receipt. *Fix:* make
`restore_and_cleanup` idempotent with a guard flag, and have the signal legs disable the trap before running
cleanup and then exit.

**MEDIUM — `04-VALIDATION.md` rows are weaker subsets of the plan commands they map to.** *(NEW; Codex,
reproduced by the orchestrator; OpenCode disagreed — see Divergent Views.)* Four rows drop legs that exist in
the corresponding task:

| Row | Leg present in PLAN.md but absent from VALIDATION.md |
|---|---|
| `04-02-02` (`:47` vs `04-02-PLAN.md:286`) | `test "$(rg -c -e 'CASK-REHEARSE-EVIDENCE schema=3' Taskfile.yml)" -eq 2` |
| `04-03-01` (`:49` vs `04-03-PLAN.md:192`) | `test "$(rg … -e 'Cellar' .planning/ROADMAP.md \| wc -l)" -ge 4` — the vacuity guard that stops the first leg passing by deleting every `Cellar` mention |
| `04-04-02` (`:52` vs `04-04-PLAN.md:200`) | `test "$(rg -c -e 'brew trust' docs/RELEASE.md)" -ge 2` |
| `04-06-03` (`:58` vs `04-06-PLAN.md:356`) | the `for t in 'Leg 1' 'Leg 2' 'Leg 3'` presence loop and the `-F` per-ID presence loop |
| `04-05-03` (`:55` vs `04-05-PLAN.md:453`) | the `\| rg -q '^ok'` leg and the `todos/pending == 7` leg |

The `04-03-01` omission is the sharpest: it drops a deliberately-added vacuity guard, so the VALIDATION row
can pass on a ROADMAP with every `Cellar` mention deleted. *Fix:* synchronize each row with its task command,
or state explicitly why a row is a deliberately reduced sampling gate.

**MEDIUM — `04-05` Task 3's `<automated>` still uses the `go test … | rg -q '^ok'` vacuity shape.**
*(NEW; orchestrator.)* `04-05-PLAN.md:453` runs `go test ./internal/upgrade/... 2>&1 | rg -q -e '^ok'`. This
is the same class as cycle-1 HIGH #6: `^ok` proves the package compiled and nothing failed, not that the two
new `TestCosignIdentityPolicy*` tests ran. The strong assertion — `rg -c '^--- PASS: TestCosignIdentityPolicy'`
reports 2 — exists, but only in `<acceptance_criteria>` prose (`:463-464`), not in the executable command
`/gsd-execute-phase` runs. *Fix:* promote the named-PASS count into the `<automated>` block.

**LOW — `04-05`'s two-`mktemp` rationale is inverted.** *(NEW; Codex.)* `04-05-PLAN.md:143-145` says "Both
directories must exist before the trap is installed, so a failure between the two `mktemp` calls cannot leak
one of them." The opposite holds: if the second `mktemp` fails, `DL_DIR` exists with no trap installed and
leaks. *Fix:* install cleanup immediately after the first `mktemp`, then add the second directory to the
state it cleans.

**LOW — `04-01` Task 3's action implies two `brewPointerMessage` call sites; its criterion enforces one.**
*(NEW; OpenCode.)* Task 1 places `fmt.Errorf("upgrade: %s", brewPointerMessage(inst))` in `upgrade.go`
(`04-01-PLAN.md:280`) and Task 3 adds a `--check` fork printing "the string returned by
`brewPointerMessage(inst)`" (`:481`) — read literally, two call sites — while the acceptance criterion
requires `rg -c 'brewPointerMessage' internal/upgrade/upgrade.go` to be exactly 1 (`:512`). *Fix:* state the
shape in the action: compute `msg := brewPointerMessage(inst)` once, branch on it.

**LOW — `04-02` Task 1's deletion range excludes an `atomic_write` mention its own gate requires removed.**
*(NEW; OpenCode.)* The action scopes deletion to `.goreleaser.yaml:570-588` and says "Delete nothing above
line 570" (`04-02-PLAN.md:140`), but `atomic_write` occurs twice in the file — `:579` (inside the range) and
`:592` (a trailing comment, outside it) — while the acceptance criterion requires
`rg -c 'atomic_write' .goreleaser.yaml` to report 0 (`:199`). Self-correcting via the failing gate, but the
edit should own the `:592` mention explicitly.

### Divergent Views

- **Do `04-VALIDATION.md` rows match their plan tasks?** OpenCode spot-checked four rows (`04-01-02`,
  `04-02-02`, `04-05-01`, `04-06-02`) and reported "identical commands in both documents." Codex checked
  four different rows and found each to be a strict subset. **Adjudicated in Codex's favor by direct
  measurement** — the orchestrator diffed all 16 rows and reproduced Codex's four divergences plus one more
  (`04-05-03`). OpenCode's sample happened to land on rows that do match; `04-02-02` is not one of them (it
  drops the `schema=3` leg). Recorded as an actionable MEDIUM.
- **Overall risk.** Codex rated **HIGH**, driven entirely by the `04-06` real-workstation state-restoration
  defects. OpenCode rated **LOW-to-MEDIUM**, having not examined the tap/man-page restoration paths against
  the cask's actual uninstall hook. The orchestrator's independent check of `.goreleaser.yaml:621` supports
  Codex's reading of the mechanism; the mitigating fact OpenCode did not have is that this host currently has
  no `seanb4t` tap, no codegraph cask and zero `codegraph*.1` pages, so neither defect fires *today*. Both
  are durable plan defects that must be fixed before `04-06` executes on any machine, but neither is an
  imminent hazard on this one.
- **Severity of the `04-05` Mutation-3 ordering defect.** OpenCode filed it MEDIUM ("a self-inconsistency
  inside a self-test protocol"). The orchestrator escalated it to HIGH because it disables the designated
  proof that cycle-1 HIGH #2 was closed and because its naive repair reinstates that exact HIGH. Codex did
  not find it.

## Verification coverage

`plan_review.source_grounding` is `true` and `drift-guard authority` resolves to **`intel`**, but
`.planning/intel/API-SURFACE.md` carries `symbolCount: 0, stale: true` and self-declares *"api-map.json has
no entries (intel extraction is regex/JS-only or not yet populated). Treat absence here as 'unknown', not
'does not exist'."* This is a **Go** repository, so per the workflow's own definitions the `intel` adapter
**cannot** check these symbols: every row below is **UNCHECKABLE → INFO**, never MISSING. Each is backed by a
direct ripgrep or Read confirmation instead. Symbols declared under each plan's "Artifacts this phase
produces" section are excluded (they do not exist yet by design).

| Symbol / artifact | Kind | Cited by | Adapter verdict | Direct confirmation |
|---|---|---|---|---|
| `resolveLatest` | func field | 04-01 | UNCHECKABLE (intel empty; Go) | CONFIRMED `internal/upgrade/upgrade.go` (5 refs) |
| `atomicSwap` | func | 04-01 | UNCHECKABLE | CONFIRMED `internal/upgrade/swap.go` (5 refs) |
| `checkWritable` | func | 04-01 | UNCHECKABLE | CONFIRMED `internal/upgrade/swap.go` (3 refs) |
| `releaseAssetName` | func | 04-01 | UNCHECKABLE | CONFIRMED `internal/upgrade/upgrade.go` (3 refs) |
| `releaseWorkflowRefPattern` | var | 04-05 | UNCHECKABLE | CONFIRMED `internal/upgrade/verify.go` (3 refs) |
| `upgradeRunFunc` | var | 04-01, 04-04 | UNCHECKABLE | CONFIRMED `internal/cli/install.go` |
| `newUpgradeCmd` | func | 04-04 | UNCHECKABLE | CONFIRMED `internal/cli/root.go` |
| `TestTaskfileGatesFailLoud` / `_EmptyFileIsError` | test | 04-05 | UNCHECKABLE | CONFIRMED — 5 `TestTaskfile*` PASS measured |
| `TestWorkflowRunBodiesInvokeTask` | test | 04-02 | UNCHECKABLE | CONFIRMED — 1 PASS measured |
| `TestHomebrewCask*` | test | 04-02 | UNCHECKABLE | CONFIRMED — 5 PASS measured |
| `TestGoreleaserPinParity` | test | 04-02 (as the mis-scope being corrected) | UNCHECKABLE | CONFIRMED — sole `^TestGoreleaser` match |
| `.goreleaser.yaml` cask hooks (`:551`, `:579`, `:621`, `:624`) | config | 04-02, 04-06 | UNCHECKABLE (non-JS) | CONFIRMED by Read |
| `Taskfile.yml:1671` rehearsal refusal | config | 04-06 | UNCHECKABLE | CONFIRMED |
| `Taskfile.yml` cosign invocations `:1492`, `:1494`, `:2541`, `:3087` | config | 04-05 | UNCHECKABLE | CONFIRMED (4 anchored) |
| identity literals ×6 across 5 files | config/docs | 04-05 | UNCHECKABLE | CONFIRMED (all six flag→literal pairs) |
| `.planning/todos/pending/` (10 items) | file-path | 04-02, 04-05 | UNCHECKABLE | CONFIRMED — 10 today; 10→8→7 arithmetic consistent |
| `brewInstall`, `detectBrewManaged`, `brewPointerMessage`, `internal/upgrade/brew.go` | artifact | 04-01 | **EXCLUDED** — declared under "Artifacts this phase produces" | n/a |
| `TestDetectBrewManaged`, `TestDetectBrewManaged_RealInstall`, `TestUpgradeRun_RefusesBrewManagedCask`, `TestCosignIdentityPolicy*`, `04-EVIDENCE.md` | artifact | 04-01, 04-05, 04-06 | **EXCLUDED** — produced by this phase | n/a |

No symbol is reported MISSING. Coverage is complete for every non-excluded symbol the plans cite.
