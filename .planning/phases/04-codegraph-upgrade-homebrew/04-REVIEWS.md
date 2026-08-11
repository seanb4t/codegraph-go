---
phase: 4
cycle: 3
reviewers: [codex]
reviewers_unavailable: [opencode]
reviewed_at: 2026-08-11T11:06:24-04:00
plans_reviewed: [04-01-PLAN.md, 04-02-PLAN.md, 04-03-PLAN.md, 04-04-PLAN.md, 04-05-PLAN.md, 04-06-PLAN.md]
plans_revision: 63ac13e
source_grounding: true
drift_guard_authority: intel
prior_cycle: cycle-2 REVIEWS.md text retained in git history
---

# Cross-AI Plan Review — Phase 4 (Cycle 3, final convergence cycle)

Reviewers were given the cycle-3 mandate (verify each of the 11 claimed cycle-2 repairs **by
measurement**, confirm the do-not-regress list, and hunt for newly introduced defects — especially
a repair that lowers or raises a numeric floor), the CONTEXT.md supersession ground rules (D-01
Caskroom-not-Cellar, D-08 seam assertion, D-09 `--check` step-aside, D-04R linuxbrew both shapes)
and repo rule `84d1gfpywd`. Neither lane raised a stale-ROADMAP finding, so the ground rules held.

**One of two lanes completed.** Codex ran with repo access and returned a fully source-grounded
review. OpenCode was invoked twice and died mid-review both times — see its section for the exact
diagnostic. Because a single-lane cycle cannot produce cross-reviewer consensus, every Codex
finding below was **independently re-measured by the orchestrating agent** against the live repo;
those measurements are recorded in "Cycle-2 disposition verification (independently measured)".

## Codex Review

## Summary

Cycle 3 confirms that the stated cycle-2 repairs are largely present and correctly measured: all `<automated>` tags balance, Homebrew tap restoration is bidirectional, pre-existing man pages block acceptance, Mutation 3 preserves the identity-literal total, and the revised floors remain calibrated to measured baselines. However, two newly exposed execution gaps remain. Most importantly, plan 04-06 protects only Task 2 with a restoration trap even though Task 1 performs the initial `brew tap`, trust, and install mutations. The cask rehearsal also reuses a pre-install timestamp marker after reinstall, which cannot independently prove that reinstall refreshed any page. These keep the plans from final convergence.

## Strengths

- All six plans have balanced anchored `<automated>` tags: 01/02/05/06 have 3 open and 3 close tags; 03/04 have 2 and 2. This closes the structural defect without relying on the known-insufficient plan validator.

- The tap baseline is captured before mutation. Plan 04-06 explicitly probes `brew tap` before the first `brew tap seanb4t/tap`, persists `TAP_PREEXISTING=yes|no`, and requires conditional untapping ([04-06-PLAN.md:156](</Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-06-PLAN.md:156>), [04-06-PLAN.md:165](</Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-06-PLAN.md:165>), [04-06-PLAN.md:288](</Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-06-PLAN.md:288>)).

- Tap restoration is checked in both directions. The executable gate derives the required `TAP_ACTION` from the baseline and separately probes actual final tap state ([04-06-PLAN.md:352](</Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-06-PLAN.md:352>)).

- Pre-existing man pages now block the run before mutation ([04-06-PLAN.md:149](</Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-06-PLAN.md:149>)). This is necessary because the current cask hook unconditionally removes every matching shared-directory page ([.goreleaser.yaml:620](</Volumes/Code/github.com/seanb4t/codegraph-go/.goreleaser.yaml:620>)).

- Cleanup specifies an idempotency guard, `RESTORE_INVOCATIONS=1`, and disarm-then-exit handlers using 130/143 ([04-06-PLAN.md:299](</Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-06-PLAN.md:299>), [04-06-PLAN.md:352](</Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-06-PLAN.md:352>)).

- Mutation 3 is now genuinely total-preserving: it relocates the flag and literal as inert comments into `verify:gatekeeper`, requires the observed total to remain 7, and explicitly forbids lowering the floor to 6 ([04-05-PLAN.md:464](</Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-05-PLAN.md:464>), [04-05-PLAN.md:515](</Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-05-PLAN.md:515>)).

- The cosign floors match the live source: there are four current shell invocations in `Taskfile.yml`, so the planned floor of five is RED before implementation ([Taskfile.yml:1492](</Volumes/Code/github.com/seanb4t/codegraph-go/Taskfile.yml:1492>), [Taskfile.yml:2541](</Volumes/Code/github.com/seanb4t/codegraph-go/Taskfile.yml:2541>), [Taskfile.yml:3087](</Volumes/Code/github.com/seanb4t/codegraph-go/Taskfile.yml:3087>)). The identity baseline is also six literals across the five named files, with Task 1 intended to add the seventh.

- The size fallback is now executable rather than aspirational: the plan requires a `{mtime, size}` baseline and at least two `File.size` uses ([04-02-PLAN.md:157](</Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-02-PLAN.md:157>), [04-02-PLAN.md:226](</Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-02-PLAN.md:226>)). Edit 5 also explicitly owns the trailing `atomic_write` comment ([04-02-PLAN.md:202](</Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-02-PLAN.md:202>)).

- The five revised validation rows are synchronized with their task commands, including the named `TestCosignIdentityPolicy` PASS count and the expanded restoration checks ([04-VALIDATION.md:46](</Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-VALIDATION.md:46>), [04-VALIDATION.md:55](</Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-VALIDATION.md:55>), [04-VALIDATION.md:57](</Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-VALIDATION.md:57>)).

- The required plan-shape invariants remain present: artifacts sections in all six plans, the three flagged assumptions and no-silent-drop equality, the assumption-delta decision, descriptor-less prohibitions, and unrenumbered threat IDs.

## Concerns

- **HIGH — Task 1 mutates the workstation before any restoration trap exists.** Plan 04-06 Task 1 runs `brew tap`, `brew trust`, and `brew install`, then deliberately leaves the install for Task 2 ([04-06-PLAN.md:176](</Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-06-PLAN.md:176>), [04-06-PLAN.md:196](</Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-06-PLAN.md:196>)). The restoration trap is not installed until Task 2’s separate script ([04-06-PLAN.md:245](</Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-06-PLAN.md:245>)). An interruption, harness failure, or executor stop between tasks can therefore leave an added tap, trust state, installed cask, and man pages behind. This directly contradicts the validation claim that “all mutating steps run inside one script whose EXIT trap is installed before the first mutating byte” ([04-VALIDATION.md:98](</Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-VALIDATION.md:98>)).

- **MEDIUM — The reinstall freshness probe is pre-satisfied by the initial install.** Plan 04-02 creates one marker before the initial install and reuses it after `brew reinstall` ([04-02-PLAN.md:280](</Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-02-PLAN.md:280>), [04-02-PLAN.md:300](</Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-02-PLAN.md:300>)). Pages created by the first install remain newer than that marker even if reinstall writes nothing. Thus the Step 5d `find -newer` check cannot independently prove reinstall freshness, despite the claim that it does. The Ruby hook’s own before/after snapshot may still catch the defect, but the rehearsal’s purported independent confirmation does not.

- **LOW — Trust-state restoration is not represented in the acceptance baseline.** Task 1 invokes `brew trust --tap`, but the baseline records only tap presence, cask presence, man pages, and prefix ([04-06-PLAN.md:165](</Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-06-PLAN.md:165>), [04-06-PLAN.md:176](</Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-codegraph-upgrade-homebrew/04-06-PLAN.md:176>)). If the tap pre-existed but was not trusted, the run changes persistent Homebrew trust state and leaves it changed. This conflicts with the stated “machine left in the state it started in” guarantee, though the exact restoration command should be confirmed against the installed Homebrew version before prescribing it.

## Suggestions

- Merge 04-06 Tasks 1 and 2 into one guarded acceptance script, or install a cleanup trap in Task 1 before `brew tap`. The trap should preserve the real install long enough to run the harness and behavioral observations, then restore on every exit or signal.

- Capture Homebrew tap trust status in the baseline and restore it, or explicitly narrow the restoration claim and require the tap to be pre-trusted as another blocking precondition.

- Use two man-page markers:

  - one immediately before initial install for Step 5;
  - a second immediately before `brew reinstall` for Step 5d.

  This makes the reinstall freshness assertion independently RED-able. Keep the Ruby `{mtime, size}` before/after check as the in-hook gate.

- Correct the manual-validation row so it no longer claims all mutations are already contained in one trapped script until the Task 1 gap is fixed.

## Risk Assessment

**HIGH.** The implementation plans are otherwise unusually rigorous, and every specifically requested cycle-2 repair measures what it claims. But the real-workstation acceptance plan still has an untrapped mutation window spanning an entire task boundary. Because that can leave persistent Homebrew state after interruption, it is a release-blocking execution-safety issue. The reinstall freshness flaw is narrower but should also be corrected before execution.

---

## OpenCode Review

**UNAVAILABLE — lane died mid-review, twice.**

| Attempt | Wall time | Terminal state | Captured stderr | Output produced |
|---|---|---|---|---|
| 1 | ~25 min | killed | `[spawn error: ETIMEDOUT]` | 284 bytes — 2 narration lines, no sections |
| 2 (`GSD_REVIEW_TIMEOUT_MS=3300000`) | ~12 min | killed | `[spawn error: ETIMEDOUT]` | 252 bytes — 3 narration lines, no sections |

The lane resolved and planned cleanly (`review-lane plan` → `ok: true`, transport `spawn`, no prompt
budget), and `review-lane invoke` reported `ok: true, stubbed: false` both times — the JSON result is
therefore **not** a reliable signal here: the adapter exited 0 while its child had already been killed.
The diagnostic lives only in `gsd-review-opencode.err`. Raising `GSD_REVIEW_TIMEOUT_MS` did not extend
the run, so the timeout that fired is the adapter's own spawn timeout, not the lane's declared
`timeoutFloorMs`; this lane cannot currently complete a review of this prompt size (355 KB) on this host.

Both truncated transcripts stop mid-investigation. They carry two verbatim intermediate confirmations
before dying — attempt 2's *"Tag balance confirmed on all six plans"* and *"Baselines confirmed"* — which
**agree with** the orchestrator's independent measurements below. They are recorded as corroborating
fragments only. They are **not** a review, carry no findings, and are **not** counted at consensus weight.

Attempt 1's final line names the one thread it was pulling when it died:

> "Now let me check whether 04-03's set-check gates are actually achievable — enumerating every `Cellar`
> line in the three target files and comparing against the plan's seven edits"

That thread was picked up and finished by the orchestrator rather than left dangling — see
"04-03 gate achievability" below. It resolves clean.

---

## Consensus Summary

**Single-lane caveat.** With OpenCode unavailable, there is no second independent model to agree or
disagree with Codex. Nothing below is a two-reviewer consensus. In its place, every Codex finding was
re-derived from the live repo by the orchestrating agent; a finding is marked **CONFIRMED** only where
that independent measurement reproduced it, and the measurement is shown.

### Agreed Strengths

Codex-reported, orchestrator-confirmed:

- **All eleven cycle-2 repairs land, and every one measures what it claims.** This is the substantive
  result of the cycle: the four HIGHs and seven actionable non-HIGHs are closed, not merely narrated.
- **No numeric floor was raised or lowered anywhere this cycle.** Counting comparison operators across
  the whole revision diff: added `-ge 5`×1, `-ge 4`×3, `-ge 3`×1, `-ge 2`×5, `-ge 1`×4, `-eq 7`×2,
  `-eq 2`×4, `-eq 1`×14, `-eq 0`×11; removed `-ge 5`×1, `-ge 3`×1, `-ge 2`×2, `-ge 1`×2, `-eq 7`×1,
  `-eq 2`×1, `-eq 1`×11, `-eq 0`×7. No threshold moved. The tempting-but-wrong repair was not taken.
- **HIGH #4 (Mutation 3) is fixed by changing the mutation, not the floor** — the single most important
  result of this cycle, since Mutation 3 is the designated proof that cycle-1 HIGH #2 closed.
- Every do-not-regress item survived: measured baselines, region requirement, named-PASS baselines,
  root-preserving walk, single `EvalSymlinks` contract, the pre-mutation trap with both escape hatches
  removed, the three flagged assumptions with no-silent-drop equality, the three descriptor-less
  prohibitions, the `T-04-01` breadcrumb, `<assumption_delta_decision>`, unrenumbered `T-04-01`…`T-04-22`,
  and the "Artifacts this phase produces" section in all six plans.

### Agreed Concerns

**C3-1 — HIGH — 04-06 Task 1 mutates the maintainer's workstation with no restoration trap in force,
and three artifacts assert the opposite.** CONFIRMED by independent measurement.

`04-06-PLAN.md:176` has Task 1 run `brew tap seanb4t/tap`, then `brew trust --tap seanb4t/tap`, then
`brew install codegraph`. `04-06-PLAN.md:196` closes the action with *"Leave the install in place for
Task 2. Do not uninstall in this task."* Task 1 contains no `trap`, no cleanup function and no
enclosing script — the earliest trap in the plan is Task 2's, at `04-06-PLAN.md:262` ("Install the EXIT
trap now — before the first mutating byte"), inside a *different task's* script. Task 1's three
mutations therefore all precede every trap the plan installs, across a task boundary. An interruption,
a harness failure, an executor stop, or Task 2 simply never running leaves an added tap, changed trust
state, an installed cask and freshly written man pages on a real workstation with nothing to undo them.

This is not merely an uncovered window — three artifacts affirmatively claim it is covered:

- `04-06-PLAN.md:23` (`must_haves.truths`): "restored by an EXIT trap installed before the first
  mutating byte" — false for Task 1's tap/trust/install, which are the *actual* first mutating bytes.
- `04-06-PLAN.md:505` (T-04-19 mitigation 2): "the whole mutating sequence runs inside ONE script with
  `trap restore_and_cleanup EXIT INT TERM` installed BEFORE the first mutating byte" — the mutating
  sequence spans two tasks and only the second one is inside that script.
- `04-VALIDATION.md:98` (manual row): "All mutating steps run inside one script whose EXIT trap is
  installed before the first mutating byte" — same claim, same falsity.

Severity is HIGH for the same reason cycle-1 finding #4 and cycle-2 C2-2/C2-3 were: this is the plan's
own workstation-safety guarantee, it is stated as fact in a threat-model mitigation and a validation
row, and the gate that would notice (Task 2's receipt assertion at `04-06-PLAN.md:352`) only ever runs
if Task 2 runs — precisely the case where nothing went wrong. It is a guard that cannot fire in the
scenario it exists for.

Two admissible shapes: merge Tasks 1 and 2 into one trapped script (the shape T-04-19 already claims),
or install the trap in Task 1 before `brew tap` and hand the armed state to Task 2. Either way the three
assertions above must be brought back into agreement with the structure. Narrowing the claims without
closing the window is *not* sufficient here, because the residue is on a real maintainer machine.

**C3-2 — MEDIUM — 04-02's Step 5d reinstall-freshness assertion is pre-satisfied by the first install's
own output.** CONFIRMED by independent measurement, with an added mechanism note.

`04-02-PLAN.md:280-300` deletes the sentinel leg of the rehearsal's post-reinstall idempotency check
(`Taskfile.yml:2072`) and replaces it with "at least one `codegraph*.1` newer than
`MAN_BASELINE_MARKER`", where that marker is created at *exactly one* point — immediately before the
**first** `brew install --cask` in Step 4 — and explicitly **reused**: the plan says "Create no second
marker" and "never create a second marker", citing cycle-1 LOW.

For Step 5's own fresh-count assertion the single marker is correct and RED-able: a genuinely stale page
predating the rehearsal is older than the marker. For Step 5d it is not. Pages written by the *first*
install are already newer than the pre-install marker, so if the reinstall writes nothing they remain
and the assertion passes anyway. The plan's stated justification — "Reusing the pre-install marker is
strictly stronger for the reinstall case, because a reinstall rewrites the pages and so they are newer
than the pre-install marker too" — assumes the very property under test.

Mechanism note the orchestrator adds: the assertion is not *unconditionally* vacuous, because
`brew reinstall --cask` runs the uninstall hook first and `.goreleaser.yaml:621` is an unconditional
`Dir.glob((man_dir/"codegraph*.1")).each { FileUtils.rm_f }`, so on today's hook the pages *are* cleared
before the reinstall writes. That makes the check's ability to fire **contingent on an unstated
assumption about a hook this very phase is editing** — the exact `84d1gfpywd` shape. The fix is small and
mechanical: create a second marker immediately before `brew reinstall` and scope the Step 5d comparison
to it, keeping the single pre-install marker for Step 5. That requires amending 04-02's explicit
"never create a second marker" instruction, which is why it cannot be left to the executor.

**C3-3 — LOW — `brew trust` state is mutated but is not in the baseline and is never restored.**
CONFIRMED.

`04-06-PLAN.md:176` runs `brew trust --tap seanb4t/tap`. The baseline file specified at
`04-06-PLAN.md:165` carries exactly four keys — `TAP_PREEXISTING`, `CASK_PREEXISTING`, `MAN_PAGES_BEFORE`,
`BREW_PREFIX` — none about trust, and the trap at `04-06-PLAN.md:288` restores tap presence only. The
residue is narrow: when `TAP_PREEXISTING=no` the trap untaps and trust goes with the tap; the leak occurs
only on a machine where the maintainer already had `seanb4t/tap` but had not trusted it. In that case
the run persistently changes Homebrew trust state and `04-06-PLAN.md:23`'s "the machine is left in the
state it started in" is false. Either record and restore trust state, or add a third blocking
precondition (tap must be pre-trusted or absent) and narrow the truth accordingly. Confirm the exact
untrust command against the installed Homebrew before prescribing one.

### Divergent Views

None available — a second lane did not complete. Where a cycle-2 divergence would normally be resolved
by cross-lane comparison, this cycle substitutes orchestrator re-measurement, which is a weaker
instrument: it shares this agent's blind spots by construction. Two of the three findings above
originate from Codex alone and were confirmed, not independently discovered.

---

## Cycle-2 disposition verification (independently measured)

Every claim re-derived from the live repo at `63ac13e`. Structure re-confirmed unchanged: 6 plans,
4 waves, 16 tasks, 16 `<automated>` blocks, 16 VALIDATION rows.

| # | Cycle-2 finding | Verdict | Measurement |
|---|---|---|---|
| HIGH 1 | Unclosed `<automated>` tags in `04-06` | **CLOSED** | Anchored `rg -c '^\s*<automated>'` vs `rg -c '</automated>\s*$'` → 3/3, 3/3, 2/2, 2/2, 3/3, 3/3. `verify plan-structure` deliberately not used — it returns `valid:true` either way |
| HIGH 2 | Unconditional `brew untap` | **CLOSED** | Probe runs before the first tap (`:159`, required at `:221-222`); trap untaps only on `TAP_PREEXISTING=no` (`:288-290`); gate at `:352` is bidirectional (`yes⇒left-in-place`, `no⇒untapped`) **and** re-probes the machine: `brew tap \| rg -c '^seanb4t/tap$'` `-eq` `rg -c '^TAP_PREEXISTING=yes$' "$B"` |
| HIGH 3 | Cleanup deletes pre-existing man pages | **CLOSED** | `:128` is a real blocking `<precondition>` that HALTs on nonzero; `:224` requires `MAN_PAGES_BEFORE=0` specifically; `:393-394` states the final zero-probe is meaningful *because* the baseline is provably zero. `.goreleaser.yaml:621` confirmed as the unconditional glob |
| HIGH 4 | `04-05` Mutation 3 unsatisfiable | **CLOSED — the critical one** | (a) total-preserving: relocation into `verify:gatekeeper`, not deletion (`:464-478`); target ordering verified — `verify:self-upgrade:` at `Taskfile.yml:2554`, `verify:gatekeeper:` at `:2751`, so the relocated pair lands outside the sliced region while staying in the file; the extractor's stated parse contract (`:319`) does not exclude `#`-prefixed lines, so the total genuinely stays 7. (b) SUMMARY must record the observed count as **7**, and a recorded 6 explicitly fails the criterion (`:515-522`). (c) lowering the floor to 6 is forbidden in **both** Task 2's action (`:383`) and Task 3's criterion (`:521`) |
| C2-2 | `04-02` size fallback unimplementable | **CLOSED** | `{mtime:, size:}` two-key snapshot at `:162`; three-way freshness at `:171`; `File\.size` `-ge 2` leg present in the `<automated>` at `:213`. RED today — live `File.size` count in `.goreleaser.yaml` is **0** |
| C2-4 | Trap double-invokes on INT/TERM | **CLOSED** | Idempotency guard at `:267`; separate per-signal registration with disarm-then-exit at `:298-306`, exit codes 130/143 at `:301`; `RESTORE_INVOCATIONS=1` asserted `-eq 1` in the `<automated>` at `:352` |
| C2-x | Five VALIDATION rows resynced | **CLOSED** | 16 rows total. `04-02-01` carries the size leg; `04-02-02` `schema=3`; `04-03-01` the `-ge 4` vacuity guard; `04-04-02` `brew trust`; `04-05-03` the three-mutation replacement; `04-06-02`/`04-06-03` resynced |
| C2-5 | `04-05:453` `rg -q '^ok'` vacuity | **CLOSED** | Replaced by `--- PASS: TestCosignIdentityPolicy` `-eq 2`, promoted into the executable `<automated>` at `:507`. Vacuity re-confirmed live on this package: `go test ./internal/upgrade/ -run '^TestCosignIdentityPolicy'` exits **0** printing `[no tests to run]` |
| C2-6 | `mktemp` ordering inverted | **CLOSED** | `04-05:161-165` prescribes create-`DL_DIR` → install-trap → create-`SIG_DIR`, with `${SIG_DIR:-}` tolerance and the rationale required in a comment |
| C2-1 | `04-01` two `brewPointerMessage` call sites | **CLOSED** | Task 3's action at `04-01:498` binds `msg := brewPointerMessage(inst)` once at the top of the branch and rewrites Task 1's `fmt.Errorf` to consume the binding; the criterion at `:537` confirms the intended shape rather than discovering it through failure |
| C2-3 | `04-02` Edit 5 must own `.goreleaser.yaml:592` | **CLOSED** | Trailing `atomic_write` comment confirmed at `.goreleaser.yaml:592`; live file-wide count is **2**, and the `<automated>` at `04-02:213` requires `-eq 0`, so the gate is RED until both are removed |

### Do-not-regress list — all confirmed holding

| Item | Measurement |
|---|---|
| Cosign floor 5 vs measured baseline 4 | Live `rg -c '^\s*cosign verify-blob' Taskfile.yml` = **4**; gate `-ge 5` → RED today |
| Identity floor 7 vs measured 6 across five files | `<measured_baseline>` table at `04-05:257-278` reproduced: `Taskfile.yml` ×2 (`:2544`, `:3090`), `README.md:59`, `docs/RELEASE.md:62`, `SECURITY.md:35`, `docs/RELEASE-PROCEDURES.md:238` = **6**; Task 1 adds the 7th |
| Region requirement — 0 literals in `verify:self-upgrade` today | Confirmed: no `--certificate-identity-regexp` between `Taskfile.yml:2554` and `:2751` |
| Named-PASS baselines 5/1/5 | Live: `TestTaskfile*` = **5**, `TestWorkflowRunBodiesInvokeTask` = **1**, `TestHomebrewCask*` = **5** |
| Root-preserving `filepath.Dir` walk | `04-01:253-254`; paired source gate (`strings.Split` absent, `filepath.Dir` present) retained |
| Single `EvalSymlinks` contract (error ⇒ not detected) | `04-01:235` returns immediately on error; T-04-03 mitigation agrees; two executing rows (dangling symlink, symlink loop) |
| Pre-mutation trap with both escape hatches removed | `04-06:352` asserts `command -v brew` first and fails loudly; no `CODEGRAPH_CASK_PREEXISTING` short-circuit remains anywhere in the plan |
| `04-01` 3 flagged assumptions + no-silent-drop equality | `<flagged_assumptions>` table carries 3 rows with per-row "Closed by"; equality line reads `3 == 0 + 3` |
| 3 descriptor-less `must_haves.prohibitions` | `04-01:38-41` — exactly 3, no descriptors |
| `T-04-01` canon-referral breadcrumb | `04-01:148-152` and `:181` |
| `<assumption_delta_decision>` block | `04-01:111-129`, decision `promote`, no accepted debt |
| Threat IDs unrenumbered | `T-04-01`…`T-04-22` contiguous across the six plans, none reused or shifted |
| "Artifacts this phase produces" in all 6 plans | Present in `04-01`…`04-06` |

### `84d1gfpywd` hunt — guards that cannot fire

All 16 `<automated>` blocks were read and their targets measured live. Results:

- **No `-run PATTERN` gate is proved by exit status alone.** All six test gates count a named `--- PASS:`
  line. The vacuity is real on this package (measured above), so this matters.
- **No pre-satisfied "prove new work" floor.** Every floor meant to go GREEN on this phase's work is RED
  today: cosign `-ge 5` (live 4); identity `-ge 7` (live 6); `File.size` `-ge 2` (live 0);
  `fresh_man_pages` `-ge 2` (live 0); `FRESH_MAN_PAGE_COUNT` `-ge 3` (live 0); `schema=3` `-eq 2`
  (live 0); `SENTINEL` `-eq 0` (live 16); `codegraph-brew-install` `-eq 0` (live 2); `atomic_write`
  `-eq 0` (live 2); Cellar-without-Caskroom `-eq 0` (live 6 in ROADMAP, 3 in REQ+PROJ); Phase-4 checkbox
  with `Caskroom` `-eq 1` (live 0); `Caskroom` in REQ+PROJ `-ge 3` (live 0); `#19121` `-ge 1` (live 0);
  `brew upgrade codegraph` across README+RELEASE `-eq 2` (live 1); `TestCosignIdentityPolicy` `-eq 2`
  (live 0). The three *preservation* floors that are pre-satisfied by design — `system_command … man`
  `-eq 1`, `brew trust` `-ge 2`, and the named-PASS 5/1/5 — are anti-regression assertions, not
  proof-of-work, and are correctly used as such.
- **No `rg -c` is counting lines where occurrences are meant.** The two remaining count floors are
  anchored (`^\s*cosign verify-blob`) or set-scoped; everything else became region-scoped or per-token.
- **The multi-token loops prove N distinct tokens.** `04-06:461` replaced the multi-`-e … -ge 3` shape
  with three explicit loops, each iteration carrying `|| { echo …; exit 1; }`, plus `-F` fixed-string
  matching for the requirement IDs and a per-ID disposition-row `-eq 1`. A shell `for` whose body
  cannot fail was the trap here; it is not present.
- **No floor is probed through a second independent matcher.** The region-scoped literal requirement is
  probed through the *same* slice the production ordering assertion uses (`04-05:128-130`).
- **No brittle count was swapped for a higher one** — see the operator diff above.

**Todo-count arithmetic re-derived:** live `.planning/todos/pending/` = **10**. `04-02` Task 4 moves 2
(`stale-pages`, `strand-the-phase-4-sentinel`) → gate `-eq 8` ✓. `04-05` Task 3 moves 1
(`no-signature-check`) → gate `-eq 7` ✓. The `add-golangci-lint` todo captured at `ffad9ee`, *before*
this revision, is correctly absorbed.

**04-03 gate achievability** (the thread OpenCode died on, finished here): ROADMAP carries 7 `Cellar`
lines, 6 of them without `Caskroom` (`:91`, `:97`, `:211`, `:212`, `:216`, `:218`; `:226` already reads
"Caskroom/Cellar"). 04-03's Edits 1, 2, 4, 5 and 6 cover exactly those six, so leg 1 (`-eq 0`) is
reachable. Leg 2 (`-ge 4`) survives because Edits 1, 2, 5 and Criterion 3 are each instructed to name
**both** trees, and `:226` retains its pair — 5 lines keep the word, against a floor of 4. Reachable,
though with only one line of margin; the plan text pins the four "name both trees" edits, so the margin
is specified rather than accidental. No finding.

---

## Verification coverage

`plan_review.source_grounding` is `true` and `drift-guard authority` resolves to **`intel`**, but
`.planning/intel/API-SURFACE.md` carries `symbolCount: 0, stale: true` and self-declares *"api-map.json
has no entries (intel extraction is regex/JS-only or not yet populated). Treat absence here as
'unknown', not 'does not exist'."* This is a **Go** repository, so by the workflow's own definitions the
`intel` adapter **cannot** check these symbols: every row below is **UNCHECKABLE → INFO**, never MISSING.
Each is backed by a direct ripgrep or Read confirmation. Symbols declared under each plan's "Artifacts
this phase produces" section are excluded — they do not exist yet by design.

| Symbol / artifact | Kind | Cited by | Adapter verdict | Direct confirmation |
|---|---|---|---|---|
| `Options.resolveLatest` / `.download` / `.verify` / `.swap` | struct fields | 04-01 | UNCHECKABLE (intel empty; Go) | CONFIRMED `internal/upgrade/upgrade.go:68-71` — the plan's spelling matches the real field names (the `*Func` names are the *types*, `:39-42`) |
| `resolveLatest` | func field | 04-01 | UNCHECKABLE | CONFIRMED `internal/upgrade/upgrade.go` (9), `release.go` (6) |
| `atomicSwap` | func | 04-01 | UNCHECKABLE | CONFIRMED `internal/upgrade/swap.go` (5) |
| `checkWritable` | func | 04-01 | UNCHECKABLE | CONFIRMED `internal/upgrade/swap.go` (3) |
| `releaseAssetName` | func | 04-01 | UNCHECKABLE | CONFIRMED `internal/upgrade/upgrade.go` (3) |
| `releaseWorkflowRefPattern` | var | 04-05 | UNCHECKABLE | CONFIRMED `internal/upgrade/verify.go` (3) |
| `upgradeRunFunc` | var | 04-01, 04-04 | UNCHECKABLE | CONFIRMED `internal/cli/upgrade.go` (3), `install.go` (1) |
| `newUpgradeCmd` | func | 04-04 | UNCHECKABLE | CONFIRMED `internal/cli/upgrade.go` (2), `root.go` (1) |
| `TestTaskfileGatesFailLoud_EmptyFileIsError` | test | 04-05 | UNCHECKABLE | CONFIRMED `internal/upgrade/taskfile_shape_test.go`; 5 `TestTaskfile*` PASS measured |
| `TestWorkflowRunBodiesInvokeTask` | test | 04-02 | UNCHECKABLE | CONFIRMED — 1 PASS measured |
| `TestHomebrewCask*` | test | 04-02 | UNCHECKABLE | CONFIRMED — 5 PASS measured |
| `TestGoreleaserPinParity` | test | 04-02 | UNCHECKABLE | CONFIRMED — 1 PASS, sole `^TestGoreleaser` match |
| `.goreleaser.yaml` cask hooks (`:592` comment, `:620-621` man glob) | config | 04-02, 04-06 | UNCHECKABLE (non-JS) | CONFIRMED by Read |
| `Taskfile.yml:1671` rehearsal refusal | config | 04-06 | UNCHECKABLE | CONFIRMED |
| `Taskfile.yml` cosign invocations | config | 04-05 | UNCHECKABLE | CONFIRMED — 4 anchored |
| `verify:self-upgrade` `:2554` / `verify:gatekeeper` `:2751` / `verify:notarized-suite` / `verify:release-assets` / `release:rehearse-cask` | Taskfile targets | 04-02, 04-05 | UNCHECKABLE | CONFIRMED — each present exactly once, and `self-upgrade` precedes `gatekeeper` (load-bearing for Mutation 3) |
| identity literals ×6 across 5 files | config/docs | 04-05 | UNCHECKABLE | CONFIRMED — all six flag→literal pairs |
| `Taskfile.yml:2072` reinstall idempotency check | config | 04-02 | UNCHECKABLE | CONFIRMED by Read |
| `.planning/todos/pending/` (10 items) | file-path | 04-02, 04-05 | UNCHECKABLE | CONFIRMED — 10 today; 10→8→7 arithmetic consistent |
| `brewInstall`, `detectBrewManaged`, `brewPointerMessage`, `internal/upgrade/brew.go` | artifact | 04-01 | **EXCLUDED** — "Artifacts this phase produces" | n/a |
| `TestDetectBrewManaged`, `TestDetectBrewManaged_RealInstall`, `TestUpgradeRun_RefusesBrewManagedCask`, `TestUpgradeCommand_HelpDocumentsBrewRefusalAndExitCodes`, `TestCosignIdentityPolicy*`, `extractCosignIdentityLiterals`, `cosignIdentityLiteral`, `04-EVIDENCE.md` | artifact | 04-01, 04-04, 04-05, 04-06 | **EXCLUDED** — produced by this phase | n/a |

No symbol is reported MISSING. Coverage is complete for every non-excluded symbol the plans cite.

---

## Convergence status

| Cycle | HIGH | Actionable non-HIGH | Total |
|---|---|---|---|
| 1 | 6 | 13 | 19 |
| 2 | 4 | 7 | 11 |
| 3 | 1 | 2 | 3 |

All 11 cycle-2 findings are closed and verified by measurement; no cycle-2 finding survives. The three
open items are newly surfaced in cycle 3: one HIGH (C3-1) plus two actionable non-HIGH (C3-2, C3-3), all
three confined to plans `04-02` and `04-06` and none touching the detection algorithm, the drift guard,
or any numeric floor. Cycle 3 was the final scheduled convergence cycle, so this outcome escalates to
the maintainer rather than looping again.
