---
phase: 4
reviewers: [codex, opencode]
reviewed_at: 2026-08-11T09:41:00-04:00
plans_reviewed: [04-01-PLAN.md, 04-02-PLAN.md, 04-03-PLAN.md, 04-04-PLAN.md, 04-05-PLAN.md, 04-06-PLAN.md]
source_grounding: true
drift_guard_authority: intel
---

# Cross-AI Plan Review — Phase 4

Both lanes ran with repo access and returned source-grounded reviews (no `[reviewed-without-repo-access]`
marker on either, neither stubbed). Reviewers were given the CONTEXT.md supersession ground rules
(D-01 Caskroom-not-Cellar, D-08 seam assertion, D-09 `--check` step-aside, D-04R linuxbrew) and the
repo standing rule `84d1gfpywd`; neither raised a stale-ROADMAP finding, so the ground rules held.

## Codex Review

## Summary

The plans are unusually thorough and generally align with authoritative `04-CONTEXT.md`. The core architecture is sound: detection belongs at the head of `upgrade.Run`, existing injection seams support direct non-mutation proofs, and the cask/receipt model reflects the shipped layout. However, I found three blocking design issues: the prescribed absolute-path reconstruction can make detection fail on every real install; the real-tap acceptance plan lacks a mechanically guaranteed rollback and cannot reliably preserve a pre-existing installation; and the cosign identity test claims universal equivalence from a finite corpus. Several verification commands also misuse line counts or are insufficiently region-scoped.

Overall assessment: revisions required before execution.

## Strengths

- The early-branch insertion point is correct. `Run` currently resolves the release before its check/read-only branch at [internal/upgrade/upgrade.go:82](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/upgrade.go:82), so placing brew detection before the `resolveLatest` initialization and call at [internal/upgrade/upgrade.go:88](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/upgrade.go:88) genuinely makes refusal and `--check` offline.

- D-08’s seam-based proof is supported by real code. `Options` already exposes package-private `resolveLatest`, `download`, `verify`, and `swap` function fields at [internal/upgrade/upgrade.go:69](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/upgrade.go:69). Testing that these remain unfired directly proves the intended control flow.

- The `--force` decision fits existing semantics. `Force` currently bypasses only the same-version guard at [internal/upgrade/upgrade.go:57](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/upgrade.go:57) and [internal/upgrade/upgrade.go:112](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/upgrade.go:112). Keeping brew detection structurally above that guard prevents accidental expansion of its authority.

- The CLI propagation claim is accurate. `newUpgradeCmd` returns `upgradeRunFunc`’s error directly at [internal/cli/upgrade.go:50](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/cli/upgrade.go:50), so an error from `upgrade.Run` naturally becomes a non-zero Cobra invocation without another CLI branch.

- Removing the sentinel is correctly scoped. The current hook writes it only after both BREW-05 assertions at [.goreleaser.yaml:570](/Volumes/Code/github.com/seanb4t/codegraph-go/.goreleaser.yaml:570), and removes it at [.goreleaser.yaml:623](/Volumes/Code/github.com/seanb4t/codegraph-go/.goreleaser.yaml:623). The rehearsal also has several real sentinel dependencies at [Taskfile.yml:2009](/Volumes/Code/github.com/seanb4t/codegraph-go/Taskfile.yml:2009), [Taskfile.yml:2072](/Volumes/Code/github.com/seanb4t/codegraph-go/Taskfile.yml:2072), and [Taskfile.yml:2215](/Volumes/Code/github.com/seanb4t/codegraph-go/Taskfile.yml:2215), all of which Plan 04-02 identifies.

- Plan 04-05 targets a real security defect. `verify:self-upgrade` currently downloads a prior binary and immediately makes it executable at [Taskfile.yml:2708](/Volumes/Code/github.com/seanb4t/codegraph-go/Taskfile.yml:2708), then executes it at [Taskfile.yml:2724](/Volumes/Code/github.com/seanb4t/codegraph-go/Taskfile.yml:2724). There is no intervening signature verification.

- The compiled identity policy is properly anchored and narrow at [internal/upgrade/verify.go:31](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/verify.go:31). Using it as the comparison oracle is directionally correct.

## Concerns

- **HIGH — Plan 04-01’s prescribed path reconstruction loses the root of an absolute path.**  
  The action says to split the resolved path on `os.PathSeparator`, treat everything left of `Caskroom`/`Cellar` as the prefix, and rebuild paths with `filepath.Join`. For an absolute POSIX path, splitting `/opt/homebrew/Caskroom/...` produces an initial empty segment; `filepath.Join("", "opt", "homebrew", ...)` yields `opt/homebrew/...`, a relative path. The receipt `os.Stat` would therefore run relative to the process working directory and real detection would fail. This undermines every principal truth in Plan 04-01 even though the existing `Run` insertion point is sound at [internal/upgrade/upgrade.go:82](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/upgrade.go:82).

- **HIGH — Plan 04-06 does not mechanically guarantee restoration after modifying the real Caskroom payload.**  
  Task 2 says restoration is unconditional, but its action specifies no shell `trap`, cleanup helper, or subprocess harness that runs cleanup after failures. The plan overwrites a Homebrew-owned executable and then runs multiple commands expected to return non-zero. A premature executor failure, interruption, or assertion can leave the substituted payload installed. This is especially risky because the project’s existing rehearsal explicitly refuses to run over a pre-existing cask at [Taskfile.yml:1671](/Volumes/Code/github.com/seanb4t/codegraph-go/Taskfile.yml:1671), while Plan 04-06 proposes handling a pre-existing install informally.

- **HIGH — Plan 04-06 cannot reliably restore an arbitrary pre-existing version.**  
  “Reinstall to the recorded prior version” is not a complete mechanism. A tap normally exposes only its current cask, and `brew install`/`reinstall` does not guarantee recovery of an older recorded version. Task 1 also runs `brew install codegraph` even if the baseline reports an existing cask, which may no-op, fail, or upgrade rather than establish a clean acceptance state. The existing repository deliberately treats an installed cask as a blocking precondition, again visible at [Taskfile.yml:1671](/Volumes/Code/github.com/seanb4t/codegraph-go/Taskfile.yml:1671).

- **MEDIUM — `EvalSymlinks` error behavior is internally inconsistent.**  
  Plan 04-01 says an error should fall back to `targetPath` and continue scanning, while threat T-04-03 says an unresolvable path falls open to “not detected.” Continuing can still detect a brew shape if the unresolved path happens to contain `Caskroom`/`Cellar` and the receipt exists. Pick one contract and test it explicitly. The safest clear contract is usually `EvalSymlinks` error ⇒ not detected.

- **MEDIUM — the cosign policy test cannot prove “exactly the same certificate subjects” with a finite SAN corpus.**  
  The compiled pattern at [internal/upgrade/verify.go:44](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/verify.go:44) and the POSIX restatement at [Taskfile.yml:2544](/Volumes/Code/github.com/seanb4t/codegraph-go/Taskfile.yml:2544) are regular languages. Matching six hand-selected strings proves those cases, not language equivalence. A drift such as allowing an extra path segment or a different tag suffix could evade the corpus. The plan’s truth and test name overclaim what the mechanism establishes.

- **MEDIUM — Plan 04-05’s identity-literal extractor is unnecessarily format-fragile.**  
  It assumes a single-quoted literal appears on a continuation line following `--certificate-identity-regexp`. The current Taskfile has that shape at [Taskfile.yml:2544](/Volumes/Code/github.com/seanb4t/codegraph-go/Taskfile.yml:2544), but equivalent shell formatting could silently become unparseable. The minimum-literal guard catches wholesale loss but not necessarily attribution mistakes or duplicate extraction from one file.

- **MEDIUM — Plan 04-02’s freshness gate has a weak time oracle.**  
  `hook_started_at = Time.now - 1` allows a stale man page written during the previous second to count as fresh. It is also susceptible to filesystem timestamp precision differences. The rehearsal-side marker-before-install check is stronger, but the shipped hook is the actual user-facing gate. A pre-command snapshot of each page’s existence/mtime, followed by a comparison proving at least one page was created or advanced, would avoid wall-clock slack.

- **LOW — Plan 04-01 adds a byte-hash/unchanged assertion after D-08 explicitly selected seam proof.**  
  This is not harmful by itself, but it reintroduces the same proxy reasoning that the authoritative context rejected. It adds test complexity without proving more than the four unfired seams already prove.

## Vacuous or misleading verification gates

- **MEDIUM — Plan 04-05 Task 1’s first automated count is file-wide, not scoped to `verify:self-upgrade`.**  
  It counts every `cosign verify-blob` line in `Taskfile.yml`, where two already exist at [Taskfile.yml:2541](/Volumes/Code/github.com/seanb4t/codegraph-go/Taskfile.yml:2541) and [Taskfile.yml:3087](/Volumes/Code/github.com/seanb4t/codegraph-go/Taskfile.yml:3087). The accompanying Node check correctly scopes ordering to the target, so the grep count is redundant and can appear green for unrelated reasons.

- **MEDIUM — several `rg -c` criteria describe occurrence counts, but `rg -c` counts matching lines.**  
  Concrete examples include:

  - Plan 04-05: “`SIG_DIR` increases by at least 3” and global `cosign verify-blob` counts.
  - Plan 04-06: `rg -c -e 'UPGR-01' -e 'UPGR-02' -e 'UPGR-03' ... -ge 3` does not establish that all three IDs occur; three lines containing only `UPGR-01` pass.
  - Plan 04-06: `rg -c -e 'Leg 1' -e 'Leg 2' -e 'Leg 3' ... -ge 3` similarly does not prove all three distinct headings.
  - Plan 04-03: combined counts for `Options.download`/`Options.swap` do not prove both strings are present.

  These violate the requested set-membership intent. Each required token should be asserted independently, or extracted and compared as a set.

- **LOW — Plan 04-02’s Go test gate uses `rg -q 'ok|PASS'` rather than a named test count.**  
  The command `go test ... -run '^TestTaskfile|^TestWorkflowRunBodiesInvokeTask$' | rg -q 'ok|PASS'` can succeed from the package-level `ok` line without proving either intended test ran. This is exactly the `go test -run` vacuity class called out in ground rule B. Assert named `--- PASS:` lines or run each exact test separately.

- **LOW — Plan 04-06’s evidence ID check conflates requirement citations with assumption dispositions.**  
  The automated check only establishes enough matching lines. The acceptance prose says each ID must appear in a disposition statement, but no executable gate enforces that association.

## Suggestions

- Replace segment splitting/rejoining with a root-preserving parent walk. Starting from `filepath.Dir(resolved)`, repeatedly inspect parent basenames for the exact version/token/tree relationship. This naturally preserves volume and root semantics on Unix and Windows-style paths.

- Make `EvalSymlinks` failure return `(brewInstall{}, false)` immediately, and add explicit dangling-symlink and symlink-loop tests.

- Require Plan 04-06 to run only when no codegraph cask is installed, matching the existing rehearsal safety policy. If pre-existing-install support is essential, first archive the entire relevant cask state and define a tested exact restoration mechanism.

- Implement acceptance mutation through a dedicated script with an EXIT trap installed before overwriting the payload. The trap should restore the original bytes first, verify their hash, and only then perform tap/cask cleanup.

- Reframe the identity test as “selected boundary-case behavioral parity,” or make the shell identity expression generated from one canonical source. If hand-written restatements remain, substantially expand property-based/generated testing around owner, repository, workflow extension, ref kind, tag prefix, whitespace, and suffix boundaries.

- Replace wall-clock freshness with before/after metadata comparison for the exact `codegraph*.1` set.

- Convert every distinct-token verification to independent assertions, for example one check each for `UPGR-01`, `UPGR-02`, and `UPGR-03`. Use `rg -o ... | wc -l` only when an actual occurrence count is intended.

- Replace Plan 04-02’s package-level `ok|PASS` matcher with exact named PASS counts for `TestWorkflowRunBodiesInvokeTask` and the intended `TestTaskfile*` tests.

## Risk Assessment

**Overall risk: HIGH.**

The implementation footprint is small and the repository already provides excellent seams, but the current path-construction prescription can make the headline detector fail against absolute real-world paths. The final acceptance plan also performs a destructive live mutation without a concrete guaranteed-cleanup mechanism. Those two issues affect correctness and workstation safety respectively and should be resolved before execution. The remaining concerns are mostly verification precision and security-claim calibration.


---

## OpenCode Review

# Cross-AI Plan Review — Phase 4: `codegraph upgrade` × Homebrew

Reviewed **with** repo access. All load-bearing source claims were traced against the actual tree.

## Summary

This is an unusually well-grounded plan set. The line citations I checked are overwhelmingly accurate: the seam block (`internal/upgrade/upgrade.go:38-43`), `Run()`'s head (`upgrade.go:82-105`), `checkWritable` (`swap.go:18-28`), the test-idiom anchors (`upgrade_test.go:15,54,174,220`), the CLI seam (`internal/cli/upgrade.go:17`), the stale doc claims (`README.md:81`, `docs/RELEASE.md:540-541`), the sentinel write (`.goreleaser.yaml:577-587`), the rehearsal assertions (`Taskfile.yml:2009-2029,2072,2091-2095,2215`), the falsified comment at `.goreleaser.yaml:451`, and the schema=1/schema=2 mismatch (`Taskfile.yml:1738` vs `2215`) all check out. The two real defects I found are both in **plan 04-05's acceptance gates**: one count-floor that is already vacuously satisfied before any edit, and one literal-count floor set one too low. Both are rule-`84d1gfpywd` violations of exactly the class this review was told to hunt.

## Strengths

- **The D-08 seam assertion is correctly targeted.** `Run()` (`upgrade.go:82-105`) calls `resolveLatest` at line 93, before the `opts.Check` branch at :98 — so plan 04-01's instruction to insert detection *above* `resolveLatest` (not merely above the Check branch) is precisely what makes both the refusal and the step-aside offline. The plan got this right where a careless reading would have placed it between :93 and :98.
- **The refusal precedent is real and correctly characterized.** `checkWritable` (`swap.go:18-28`, called at `upgrade.go:119`) is exactly the early error-returning shape D-05/D-11 cite, including the "BEFORE downloading" rationale.
- **Anti-vacuity discipline is mostly excellent.** Plan 04-01's Task 2 pre-loop guard (`t.Fatalf` if rows < 14 or not-detected rows < 4), the count-asserted `rg -c` PASS checks with the explicit "`go test -run` exits 0 on no match" rationale, and the paired negative/positive gates in 04-02 (`codegraph-brew-install` == 0 **and** `system_command binary, args: ["man", man_dir]` == 1) are textbook applications of rule B.
- **Plan 04-05 Task 1's region-scoped ordering assertion is the right tool.** Slicing `Taskfile.yml` between `verify:self-upgrade:` (2554) and `verify:gatekeeper:` (2751) — both targets verified to exist in that order — and asserting `cosign verify-blob` precedes `chmod +x "${PRIOR_BIN}"` (currently at 2715) is a genuinely structural check, not a grep.
- **Plan 04-02's `FileUtils.rm_f` count is accurate.** Current occurrences: `.goreleaser.yaml:621` (man pages) and `:624` (sentinel); post-removal "exactly 1" is correct and non-vacuous.
- **Plan 04-06's three-leg evidentiary split** (genuine layout / substituted payload named in the heading / natural path explicitly not claimed) correctly handles the hardest honesty problem in this phase — the released binary predates the detection code — without overclaiming.
- **Sentinel-removal blast radius is fully enumerated.** I confirmed zero `sentinel`/`codegraph-brew-install` matches in `goreleaser_shape_test.go` and `taskfile_shape_test.go` (rg exit 1), matching RESEARCH assumption A3 — no shape-test edits needed, and 04-02 Task 1 correctly re-runs that grep before editing.

## Concerns

- **HIGH — Plan 04-05 Task 1: the `cosign verify-blob` count gate is already satisfied.** Acceptance criterion says `rg -c -e 'cosign verify-blob' Taskfile.yml` must report "at least 3 … up from 2 before this task." Measured today: **9 matching lines, including 4 actual invocations** (`Taskfile.yml:1492, 1494, 2541, 3087`; the 1492/1494 pair in the notarize-rehearsal target was missed). The gate passes with zero edits — a textbook vacuous pass under rule B.1/B.2. The region-scoped node assertion in the same criterion is the real gate and saves this task, but the count claim and its recorded baseline are factually wrong and would mislead the SUMMARY's before/after record. Fix: drop the count criterion or raise the floor to ≥5 invocations with an accurate baseline.
- **HIGH — Plan 04-05 Task 2: the identity-literal floor is off by one.** The guard `t.Fatalf`s when fewer than **4** literals are found across `Taskfile.yml`, `README.md`, `docs/RELEASE.md`. Exactly 4 exist **today** (`README.md:59`, `docs/RELEASE.md:62`, `Taskfile.yml:2544, 3090`), before Task 1 adds the fifth in `verify:self-upgrade`. So the drift guard passes even if Task 1's cosign step — the plan's headline change — is silently dropped or its flag misspelled. The floor must be **5**, or better, the test should require a literal extracted from the `verify:self-upgrade` region specifically.
- **MEDIUM — Manual line-number comparisons presented as acceptance criteria.** Plan 04-01 Task 1 ("check with `rg -n …` and compare the two line numbers") and Task 3 ("`rg -n -e 'opts\.Check'` … shows the new brew-branch occurrence at a lower line number") are eyeball checks, not executable gates — inconsistent with the same plans' own insistence elsewhere on count-asserted automation, and with plan 04-05's scripted node-based ordering check, which proves the authors know how to automate exactly this class of assertion. Same applies to plan 04-01 Task 1's "the refusal error message … matches the shape …" criterion.
- **MEDIUM — Help text is a third, unpinned copy of the pointer message.** Plan 04-01 guarantees refusal and `--check` share one builder (`brewPointerMessage`, unexported), but plan 04-04's `Long` string in `internal/cli/upgrade.go` is a separate literal — the cli package cannot import the unexported builder. The two exit behaviors are pinned by `TestUpgradeCommand_HelpDocumentsBrewRefusalAndExitCodes`, but the *exact command string* is only substring-pinned, so `brew.go` and the help text can drift in wording while both tests stay green. Low practical risk (the command literal itself is asserted), but the "can never drift" claim in 04-01's `key_links` is true only of two of the three surfaces.
- **LOW — Plan 04-02 Task 2 internal inconsistency on the freshness baseline.** The action first defines `MAN_BASEMARKER` created "immediately BEFORE the `brew install --cask` invocation in Step 4," then describes the Step 5d idempotency extension as requiring pages "newer than a marker file created immediately before the reinstall." Both work (reinstall rewrites pages, so pages are newer than the pre-install marker too), but the two sentences name different marker-creation points; the executor could create a second marker or misplace the first.
- **LOW — Plan 04-06 Task 2 `--force` message-identity assertion.** "its captured message is identical to bare `upgrade`'s" is only true if the CLI layer adds nothing (cobra error prefixes etc.) between the two runs. `Run()` returns the same wrapped error in both cases so it should hold, but "identical" is stricter than the unit tests assert (substring). If cobra annotates differently per flag combination this fails spuriously; "contains the same pointer sentence" would be the calibrated bar.
- **LOW — `brew trust --tap` security framing.** Plan 04-06 T-04-21 correctly accepts this and names the deliberately-unfolded Phase-3 todo, but nothing in the phase closes the published-instruction defect itself (`docs/RELEASE.md:515-535` region). Fine as scoped; worth restating at ship time so it doesn't get lost between phases.

## Suggestions

1. **04-05 Task 1:** replace the count criterion with the region-scoped assertion alone, and record the true baseline (4 invocations / 9 lines) in the SUMMARY.
2. **04-05 Task 2:** raise the literal floor to 5 and/or add a region-scoped requirement that at least one literal comes from the `verify:self-upgrade` body — the same slicing technique Task 1 already uses.
3. **04-01 Tasks 1 & 3:** convert the two manual line-number comparisons into small scripted assertions (the `node -e` offset pattern from 04-05 works verbatim on `upgrade.go`).
4. **04-02 Task 2:** pick one marker-creation point (pre-install, reused for the reinstall check) and say so in both sentences.
5. **04-06 Task 2:** relax the `--force` criterion from "identical message" to "contains the same resolved-path pointer sentence as bare `upgrade`."

## Risk Assessment

**LOW-MEDIUM.** The core engineering plan (04-01) is verified against the real code: correct insertion point, correct seams, correct precedent, correct fixture realism (invoking through the symlink defeats the `EvalSymlinks` false-negative, and the `t.TempDir()`-is-itself-a-symlink flake is explicitly handled). Dependency ordering across waves is sound — 04-02/04-03 touch disjoint files from 04-01, 04-04 correctly waits on 04-01, 04-05 correctly waits on 04-02+04-04, and 04-06's acceptance depends on everything. The security-adjacent work (cosign ordering, identity-regexp behavioral equality, RED demonstrations with byte-clean reverts) is well designed. The two HIGH findings are both acceptance-gate calibration errors in 04-05 — real rule-B violations, but easily corrected before execution, and each is partially backstopped by a stronger sibling assertion. Nothing found threatens the phase's goals as amended by 04-CONTEXT.md.


---

## Consensus Summary

Two independent grounded reviewers converge on the same verdict: the phase's **design** is sound and
unusually well-evidenced against the real tree, and the residual risk is concentrated in **acceptance-gate
calibration** (plan 04-05) and **live-machine safety** (plan 04-06). Codex rates overall risk HIGH,
OpenCode LOW-MEDIUM; the gap is entirely about how much weight to give 04-06's missing rollback harness,
not about the engineering plan. Neither reviewer found a plan that fails to achieve the phase goal **as
amended by 04-CONTEXT.md**.

### Orchestrator-verified facts (measured, where the two reviewers disagreed)

The reviewers reported conflicting numbers for two load-bearing counts. Both were re-measured directly:

- `rg -c 'cosign verify-blob' Taskfile.yml` → **10 matching lines** (Codex implied 2; OpenCode said 9 —
  both wrong). Actual **invocation** sites are **4**: `Taskfile.yml:1492`, `:1494`, `:2541`, `:3087`.
  The remaining 6 lines are comments and `echo` statements. Either way, 04-05 Task 1's
  `… | rg -c -e 'cosign verify-blob'` ≥ 3 gate **passes today, before any edit**, and the plan's recorded
  baseline of "up from 2" (`04-05-PLAN.md:246`) is factually wrong. OpenCode's HIGH stands, sharpened.
- `--certificate-identity-regexp` literals across `Taskfile.yml`, `README.md`, `docs/RELEASE.md` →
  **exactly 4 today** (`README.md:59`, `docs/RELEASE.md:62`, `Taskfile.yml:2544`, `:3090`). 04-05 Task 2's
  guard `t.Fatalf`s only "if the total is below 4" (`04-05-PLAN.md:223`). The floor is therefore met
  before Task 1 adds the fifth literal, so the drift guard stays green even if Task 1's cosign step is
  dropped entirely. OpenCode's second HIGH is **confirmed exactly as stated**.

Two further gate facts were measured by the orchestrator and are not in either reviewer's output:

- **`go test -run` empty-match vacuity is real here, and confirmed empirically.**
  `go test ./internal/upgrade/ -run '^TestZZZDefinitelyNoSuchTest$'` prints
  `ok  github.com/seanb4t/codegraph-go/internal/upgrade  0.173s [no tests to run]`. Therefore
  `04-02-PLAN.md:235`'s final leg — `go test ./internal/upgrade/ -run '^TestTaskfile|^TestWorkflowRunBodiesInvokeTask$' 2>&1 | rg -q 'ok|PASS'` —
  matches on the literal `ok` and **cannot distinguish "the tests ran and passed" from "no tests ran"**.
  This is a live instance of exactly the class rule `84d1gfpywd` bans, inside a phase whose sibling plans
  call the hazard out by name. Codex filed it LOW; it is raised here to **HIGH**.
- **`04-02-PLAN.md:176`'s `-run '^TestGoreleaser'` gate is mis-scoped, not vacuous.** `go test -run` is
  package-scoped, so the pattern *does* match — but it matches exactly one test,
  `TestGoreleaserPinParity` (`internal/upgrade/taskfile_shape_test.go:761`), which reads `tools/go.mod`
  and `.github/workflows/release.yml` for GoReleaser version-pin parity and **reads nothing in
  `.goreleaser.yaml`**. Zero tests in `internal/upgrade/goreleaser_shape_test.go` match `^TestGoreleaser`.
  The gate is already green and stays green whatever Task 1 does to the cask hooks, so the criterion's
  gloss "the shape tests that pin this file's invariants still pass" is false. The paired
  `go test ./internal/upgrade/...` leg does cover them. Retarget to `^TestHomebrewCask`. MEDIUM.

### Agreed Strengths

- **The `Run()` insertion point is correct and verified by both.** Detection must sit above the
  `resolveLatest` call (`internal/upgrade/upgrade.go:93`), not merely above the `opts.Check` branch
  (`:98`) — both reviewers independently traced this and confirmed the plan gets it right, which is what
  makes both the refusal and the `--check` step-aside genuinely offline.
- **D-08's seam proof is supported by real code.** `Options`' package-private `resolveLatest`/`download`/
  `verify`/`swap` func fields (`internal/upgrade/upgrade.go:69-72`) exist and are injectable, so asserting
  they never fire is a direct control-flow proof rather than a proxy.
- **Anti-vacuity discipline is, on the whole, excellent.** Both cite 04-01 Task 2's in-test pre-loop guard
  (`t.Fatalf` when rows < 14 or not-detected rows < 4), the count-asserted `--- PASS:` gates carrying the
  explicit "`go test -run` exits 0 on no match" rationale, and 04-02's paired negative/positive gates
  (`codegraph-brew-install` == 0 **and** `system_command binary, args: ["man", man_dir]` == 1 — measured
  today at 2 and 1 respectively, so that gate is genuinely RED before the task).
- **04-05 Task 1's region-scoped `node -e` ordering assertion is the right tool** and is the leg that
  actually earns the task. `verify:self-upgrade:` (`Taskfile.yml:2554`) and `verify:gatekeeper:` (`:2751`)
  both exist in that order, and the region today contains `chmod +x "${PRIOR_BIN}"` (`:2715`) with **no**
  preceding signature check — so the assertion is RED before the task and GREEN after.
- **04-05 targets a real security defect**, not a hypothetical one: the target downloads a prior release
  binary, `chmod +x`es it and executes it with nothing verifying it in between.
- **04-06's three-leg evidentiary split** handles the phase's hardest honesty problem — the shipped binary
  predates the detection code — by naming leg 3 as not executed rather than claiming it.

### Agreed Concerns

Ordered by severity. Items marked **[both]** were raised independently by both reviewers.

1. **HIGH — 04-05 Task 1's cosign count gate is pre-satisfied** (OpenCode HIGH, Codex MEDIUM as
   "file-wide, not scoped"): measured 10 lines / 4 invocations against a `-ge 3` floor and a recorded
   baseline of 2. Drop the count criterion in favour of the region-scoped node assertion, or raise the
   floor to ≥5 invocations with the corrected baseline.
2. **HIGH — 04-05 Task 2's identity-literal floor of 4 is already met** (OpenCode): the drift guard cannot
   detect that Task 1's cosign step was dropped. Raise to 5, or require at least one literal extracted
   from the `verify:self-upgrade` region specifically, reusing Task 1's own slicing technique.
3. **HIGH — 04-01's prescribed path reconstruction loses the root of an absolute path** (Codex):
   `04-01-PLAN.md:198-205` says to split the resolved path on `os.PathSeparator`, treat everything left of
   the tree segment as the prefix, and rebuild with `filepath.Join`. For `/opt/homebrew/Caskroom/…` the
   split yields a leading empty segment which `filepath.Join` discards, producing the **relative** path
   `opt/homebrew/…`; the receipt `os.Stat` would then resolve against the process CWD and real detection
   would fail. Mitigating: 04-01's own table test drives absolute fixture paths, so it would catch this at
   execution — but the plan text prescribes the wrong algorithm. Replace with a root-preserving parent
   walk from `filepath.Dir(resolved)`.
4. **HIGH — 04-06 has no mechanical rollback after overwriting a real Homebrew-owned payload** (Codex).
   `04-06-PLAN.md:185` says restoration is "unconditional, including on any failure path above", but the
   action specifies no `trap`, cleanup helper, or subprocess harness — and the task deliberately runs
   commands expected to exit non-zero. An interruption leaves a substituted payload installed in the user's
   real Caskroom. The repo's own rehearsal already refuses to run over a pre-existing cask
   (`Taskfile.yml:1671`), which is the safety posture this plan departs from. Install an EXIT trap before
   the first mutating byte; restore-and-hash-verify first, cleanup second.
5. **HIGH — 04-06 cannot reliably restore an arbitrary pre-existing cask version** (Codex).
   "Reinstall to the recorded prior version" is not a mechanism a tap generally supports, and Task 1 runs
   `brew install codegraph` regardless of what the baseline reported. Either make a pre-existing install a
   blocking precondition (matching `Taskfile.yml:1671`) or define and test an exact restoration path.
6. **HIGH — 04-02 Task 2's `rg -q 'ok|PASS'` leg is a confirmed vacuous pass** (orchestrator-measured;
   Codex flagged it LOW). See the measured evidence above. Replace with count-asserted named
   `--- PASS:` lines, as every sibling plan already does.
7. **MEDIUM [both] — `rg -c` is used where set-membership or occurrence counts are intended.** `rg -c`
   counts matching **lines**, so none of these prove what their prose claims: 04-06's
   `-e 'UPGR-01' -e 'UPGR-02' -e 'UPGR-03' … -ge 3` and `-e 'Leg 1' -e 'Leg 2' -e 'Leg 3' … -ge 3` (three
   lines all naming `UPGR-01` pass), 04-03's `-e 'Options.download' -e 'Options.swap' … -ge 1` (proves
   neither string individually, and the unescaped `.` matches any character), and 04-05's `SIG_DIR` and
   `cosign verify-blob` floors. Assert each required token independently, or extract with
   `rg -o … | wc -l` where an occurrence count is genuinely meant.
8. **MEDIUM — the cosign identity test overclaims** (Codex). Both the compiled
   `releaseWorkflowRefPattern` (`internal/upgrade/verify.go:44`) and its POSIX restatement
   (`Taskfile.yml:2544`) are regular languages; agreement on a finite hand-picked SAN corpus proves those
   cases, not language equivalence. 04-05's must-have wording ("accepts and rejects **exactly** the same
   certificate subjects") and the test name should be recalibrated to "selected boundary-case behavioural
   parity", or the shell expression should be generated from one canonical source.
9. **MEDIUM — the `EvalSymlinks` error contract is internally inconsistent** (Codex). `04-01-PLAN.md:194`
   says fall back to `targetPath` and keep scanning; threat T-04-03 says an unresolvable path falls open to
   "not detected". These differ observably when the unresolved path happens to contain `Caskroom`/`Cellar`
   and a receipt exists. Pick one — "error ⇒ not detected" is the safe reading — and add dangling-symlink
   and symlink-loop rows to the table.
10. **MEDIUM — 04-01 Tasks 1 and 3 use eyeball line-number comparisons as acceptance criteria**
    (OpenCode): "compare the two line numbers" and "shows the new brew-branch occurrence at a lower line
    number" are not executable gates, and are inconsistent with the same phase's insistence on count-
    asserted automation. 04-05's `node -e` offset pattern works verbatim on `upgrade.go`.
11. **MEDIUM — 04-02's `hook_started_at = Time.now - 1` is a weak freshness oracle** (Codex): the one-second
    slack lets a page written in the previous second count as fresh, and filesystem timestamp precision
    varies. A pre-command existence/mtime snapshot of the exact `codegraph*.1` set, compared afterwards, has
    no wall-clock slack.
12. **MEDIUM — 04-05's identity-literal extractor is format-fragile** (Codex): it assumes a single-quoted
    literal on the continuation line after `--certificate-identity-regexp`. That shape holds today
    (`Taskfile.yml:2544`), but equivalent shell formatting would silently become unparseable, and the
    minimum-count guard catches wholesale loss rather than misattribution.
13. **MEDIUM — the help text is a third, unpinned copy of the pointer message** (OpenCode):
    `brewPointerMessage` is unexported, so `internal/cli/upgrade.go`'s `Long` string cannot share it. The
    exit behaviours are pinned, and the command literal is substring-asserted, but 04-01's `key_links`
    claim that the surfaces "can never drift" is true of only two of the three.
14. **MEDIUM — 04-02 Task 1's `^TestGoreleaser` gate does not cover what it claims** (orchestrator). See
    the measured evidence above; retarget to `^TestHomebrewCask`.
15. **MEDIUM — 04-06 Task 2's restoration gate has two escape hatches** (orchestrator):
    `test "$(brew list --cask --versions codegraph 2>/dev/null | wc -l …)" -eq 0 -o -n "${CODEGRAPH_CASK_PREEXISTING:-}"`
    passes when `brew` is absent from `PATH` (the `2>/dev/null` swallows the failure and `wc -l` yields 0),
    and is short-circuited entirely by setting the env var. It is an all-negative gate with no positive
    floor proving restoration actually happened — the exact shape rule `84d1gfpywd` bans. Add a positive
    assertion: the restored payload's sha256 equals the Task-1 baseline.
16. **LOW — 04-01 reintroduces a byte-hash/unchanged assertion that D-08 deliberately replaced** (Codex).
    Harmless but it is the proxy reasoning the authoritative context rejected, and it adds test surface
    without proving more than the four unfired seams.
17. **LOW — 04-06's evidence-ID gate does not enforce the ID↔disposition association** its prose requires
    (Codex): matching enough lines is not the same as each ID appearing in a disposition statement.
18. **LOW — 04-02 Task 2 names two different marker-creation points** (OpenCode): "immediately BEFORE the
    `brew install --cask` invocation in Step 4" vs "created immediately before the reinstall". Both work,
    but an executor could create a second marker or misplace the first.
19. **LOW — 04-06's `--force` "identical message" criterion is stricter than the unit tests' bar**
    (OpenCode): substring identity is what the tests assert; "contains the same resolved-path pointer
    sentence" is the calibrated criterion.

### Divergent Views

- **Overall risk: Codex HIGH vs OpenCode LOW-MEDIUM.** The disagreement is not about the engineering
  plan — both verified 04-01's insertion point, seams, precedent and fixture realism as correct. Codex
  weights 04-06's missing rollback harness as a workstation-safety defect that blocks execution; OpenCode
  treats 04-06 as adequately scoped and rates only 04-05's gate calibration as HIGH. Both positions are
  compatible with executing waves 1–3 now and hardening 04-06 before wave 4.
- **04-01's path reconstruction.** Codex raises it as a blocking HIGH; OpenCode reviewed the same plan and
  did not flag it, instead praising the fixture realism. The orchestrator's read is that Codex is right
  about the prescription and OpenCode is right that the table test would catch it — so it is a real defect
  in the plan text with a working backstop, not a silent trap.
- **The pre-existing `cosign verify-blob` count.** Codex says 2, OpenCode says 9 lines / 4 invocations,
  measured truth is 10 lines / 4 invocations. Both reviewers reached the correct *conclusion* (the gate is
  not doing the work it claims) from a wrong count; the corrected numbers are recorded above so the
  SUMMARY's before/after record is right.

---

## Verification coverage

*(source-grounding pass; `plan_review.source_grounding = true`)*

The source-grounding authority for this review is **`intel`**. `.planning/intel/API-SURFACE.md` is empty —
it carries the generator's own banner `Incomplete: api-map.json has no entries (intel extraction is
regex/JS-only or not yet populated)`, `symbolCount: 0`, `stale: true`, and it is the only file in
`.planning/intel/`. CodeGraph-Go is a **Go** repository and the intel adapter's extractor is JS/TS-regex
only, so it can never populate a Go symbol. Per the workflow's own definitions — `MISSING` requires that
the adapter *can* check this language and kind — every Go symbol here, and equally every non-Go artifact
(Taskfile targets, GoReleaser cask hooks, GitHub Actions jobs, Markdown passages), is **UNCHECKABLE by the
adapter (INFO severity), never MISSING**. No adapter verdict below is a claim about whether a symbol
exists. So that a clean review does not silently mean "nothing was checked", a supplementary ripgrep/Read
sweep of the working tree was run against every symbol the six plans claim **already exists**; those
results are the `rg/Read confirmation` column and are the only existence evidence in this section.

| Symbol | Kind | Plan | Adapter verdict | rg/Read confirmation |
|---|---|---|---|---|
| `Run` | go-func | 04-01, 04-03, 04-04 | UNCHECKABLE (Go; empty API-SURFACE.md) | CONFIRMED `internal/upgrade/upgrade.go:82` |
| `Options` | go-type | 04-01 | UNCHECKABLE (Go) | CONFIRMED `internal/upgrade/upgrade.go:48` |
| `Options.resolveLatest` | go-var (func seam) | 04-01, 04-03 | UNCHECKABLE (Go) | CONFIRMED `internal/upgrade/upgrade.go:69` |
| `Options.download` | go-var (func seam) | 04-01, 04-03 | UNCHECKABLE (Go) | CONFIRMED `internal/upgrade/upgrade.go:70` |
| `Options.verify` | go-var (func seam) | 04-01 | UNCHECKABLE (Go) | CONFIRMED `internal/upgrade/upgrade.go:71` |
| `Options.swap` | go-var (func seam) | 04-01, 04-03 | UNCHECKABLE (Go) | CONFIRMED `internal/upgrade/upgrade.go:72` |
| `Options.Check` | go-var (field) | 04-01 | UNCHECKABLE (Go) | CONFIRMED `internal/upgrade/upgrade.go:51` |
| `Options.Force` | go-var (field) | 04-01 | UNCHECKABLE (Go) | CONFIRMED `internal/upgrade/upgrade.go:63` (doc block :58-63 as cited) |
| `Options.Out` | go-var (field) | 04-01 | UNCHECKABLE (Go) | CONFIRMED `internal/upgrade/upgrade.go:67` |
| `checkWritable` | go-func | 04-01 | UNCHECKABLE (Go) | CONFIRMED `internal/upgrade/swap.go:18` |
| `atomicSwap` | go-func | 04-01 | UNCHECKABLE (Go) | CONFIRMED `internal/upgrade/swap.go:40` |
| `releaseAssetName` | go-func | 04-01 | UNCHECKABLE (Go) | CONFIRMED `internal/upgrade/upgrade.go:209` |
| `releaseRepoSlug` | go-var (const) | 04-01, 04-05 | UNCHECKABLE (Go) | CONFIRMED `internal/upgrade/verify.go:43` |
| `releaseWorkflowRefPattern` | go-var (const) | 04-05 | UNCHECKABLE (Go) | CONFIRMED `internal/upgrade/verify.go:44` |
| `verifyRelease` | go-func | 04-05 | UNCHECKABLE (Go) | CONFIRMED `internal/upgrade/verify.go:88` |
| `newUpgradeCmd` | go-func | 04-04 | UNCHECKABLE (Go) | CONFIRMED `internal/cli/upgrade.go:26` |
| `upgradeRunFunc` | go-var | 04-04 | UNCHECKABLE (Go) | CONFIRMED `internal/cli/upgrade.go:17` |
| `newUpgradeCmd().Long` | go-var (cobra field) | 04-04 | UNCHECKABLE (Go) | CONFIRMED `internal/cli/upgrade.go:33` (cited :33-36) |
| `newUpgradeCmd().Example` | go-var (cobra field) | 04-04 | UNCHECKABLE (Go) | CONFIRMED `internal/cli/upgrade.go:37` |
| `os.Executable()` call site | go-func (call) | 04-04 | UNCHECKABLE (Go) | CONFIRMED `internal/cli/upgrade.go:45` |
| `TestUpgradeRun_CheckReportsAvailabilityWithoutDownloading` | go-test | 04-01 | UNCHECKABLE (Go) | CONFIRMED `internal/upgrade/upgrade_test.go:15` |
| `TestUpgradeRun_TamperedDownloadNeverSwaps` | go-test | 04-01 | UNCHECKABLE (Go) | CONFIRMED `internal/upgrade/upgrade_test.go:54` |
| `TestUpgradeRun_ForceReinstallsSameVersion` | go-test | 04-01 | UNCHECKABLE (Go) | CONFIRMED `internal/upgrade/upgrade_test.go:174` (cited :174-214) |
| `TestUpgradeRun_ForceStillVerifiesBeforeSwap` | go-test | 04-01 | UNCHECKABLE (Go) | CONFIRMED `internal/upgrade/upgrade_test.go:220` (cited :220-259) |
| `TestUpgradeCommand_PropagatesError` | go-test | 04-01, 04-04 | UNCHECKABLE (Go) | CONFIRMED `internal/cli/upgrade_test.go:50` |
| `TestUpgradeCommand_DelegatesWithCheckAndVersion` | go-test | 04-04 | UNCHECKABLE (Go) | CONFIRMED `internal/cli/upgrade_test.go:16` |
| `TestTaskfileGatesFailLoud` | go-test | 04-05 | UNCHECKABLE (Go) | CONFIRMED `internal/upgrade/taskfile_shape_test.go:1014` |
| `TestTaskfileGatesFailLoud_EmptyFileIsError` | go-test | 04-05 | UNCHECKABLE (Go) | CONFIRMED `internal/upgrade/taskfile_shape_test.go:1047` |
| `TestWorkflowRunBodiesInvokeTask` | go-test | 04-02, 04-05 | UNCHECKABLE (Go) | CONFIRMED `internal/upgrade/taskfile_shape_test.go:1342` |
| `verify_test.go:115-140` accept/reject pair | go-test | 04-05 | UNCHECKABLE (Go) | CONFIRMED `internal/upgrade/verify_test.go:121`, `:132` — inside the cited window |
| `release_workflow_shape_test.go:860-910` real-SAN block | go-test | 04-05 | UNCHECKABLE (Go) | CONFIRMED `:877`, `:893`, `:900`, `:907` — all three cited cases exist in the cited window |
| `taskfilePath`, `goreleaserPath` | go-var (const) | 04-05 | UNCHECKABLE (Go) | CONFIRMED `internal/upgrade/taskfile_shape_test.go:27-32` (cited :20-40) |
| `^TestGoreleaser…` (04-02's gate pattern) | go-test | 04-02 | UNCHECKABLE (Go) | CONFIRMED — matches **exactly one** test at package scope: `TestGoreleaserPinParity` (`internal/upgrade/taskfile_shape_test.go:761`). Zero matches in `goreleaser_shape_test.go`. See Findings |
| `internal/upgrade/{upgrade,swap,verify}.go` | file-path | 04-01, 04-05 | UNCHECKABLE (Go) | CONFIRMED (all exist) |
| `internal/upgrade/{upgrade,verify,taskfile_shape,goreleaser_shape,release_workflow_shape}_test.go` | file-path | 04-01, 04-02, 04-05 | UNCHECKABLE (Go) | CONFIRMED (all exist) |
| `internal/cli/upgrade.go`, `internal/cli/upgrade_test.go` | file-path | 04-01, 04-04 | UNCHECKABLE (Go) | CONFIRMED (both exist) |
| `cmd/codegraph` | file-path | 04-06 | UNCHECKABLE (Go) | CONFIRMED `cmd/codegraph/main.go` |
| `internal/upgrade/brew.go`, `brew_test.go` (asserted ABSENT) | file-path | 04-01 | UNCHECKABLE (Go) | CONFIRMED ABSENT (searched `ls internal/upgrade/brew*.go` → no matches) — matches the plan's own claim |
| `release:rehearse-cask` | task-target | 04-02 | UNCHECKABLE (adapter is JS-regex-only) | CONFIRMED `Taskfile.yml:1601` |
| `check:goreleaser` | task-target | 04-02 | UNCHECKABLE (JS-regex-only) | CONFIRMED `Taskfile.yml:2342` |
| `verify:release-assets` | task-target | 04-05 | UNCHECKABLE (JS-regex-only) | CONFIRMED `Taskfile.yml:2357` |
| `verify:self-upgrade` | task-target | 04-05 | UNCHECKABLE (JS-regex-only) | CONFIRMED `Taskfile.yml:2554` |
| `verify:gatekeeper` (region delimiter) | task-target | 04-05 | UNCHECKABLE (JS-regex-only) | CONFIRMED `Taskfile.yml:2751` — immediately follows `verify:self-upgrade`, so the delimiter is valid |
| `verify:notarized-suite` | task-target | 04-05 | UNCHECKABLE (JS-regex-only) | CONFIRMED `Taskfile.yml:3011` (cosign block :3087-3093, inside the cited :3040-3110) |
| `test` (the `task test` wrapper) | task-target | 04-01, 04-04, 04-05, 04-06 | UNCHECKABLE (JS-regex-only) | CONFIRMED `Taskfile.yml:3504` |
| `SENTINEL_PATH` / `SENTINEL_VERDICT` / `SENTINEL_FIRST_LINE` | shell-var | 04-02 | UNCHECKABLE (JS-regex-only) | CONFIRMED `Taskfile.yml:2009`, `:2011`, `:2017` |
| `REAL_BIN` / `REAL_BIN_DIR` | shell-var | 04-02 | UNCHECKABLE (JS-regex-only) | CONFIRMED `Taskfile.yml:2007`, `:2008` |
| `CASK-REHEARSE-EVIDENCE schema=1` / `schema=2` | evidence-line | 04-02 | UNCHECKABLE (JS-regex-only) | CONFIRMED `Taskfile.yml:1738` and `:2215` — the schema mismatch the plan describes is real |
| Step 5b marker assertions / six-key loop | shell-block | 04-02 | UNCHECKABLE (JS-regex-only) | CONFIRMED `Taskfile.yml:2007-2029` |
| idempotency check (man page AND marker) | shell-block | 04-02 | UNCHECKABLE (JS-regex-only) | CONFIRMED `Taskfile.yml:2072` (exact cited line) |
| post-uninstall symmetry check | shell-block | 04-02 | UNCHECKABLE (JS-regex-only) | CONFIRMED `Taskfile.yml:2091-2095` |
| `DL_DIR` + single-dir trap | shell-var | 04-05 | UNCHECKABLE (JS-regex-only) | CONFIRMED `Taskfile.yml:2708-2709` |
| `PRIOR_ASSET` / `PRIOR_BIN` / `PRIOR_SHA256` | shell-var | 04-05 | UNCHECKABLE (JS-regex-only) | CONFIRMED `Taskfile.yml:2711`, `:2714`, `:2716` |
| `chmod +x "${PRIOR_BIN}"` (the hazard 04-05 moves) | shell-stmt | 04-05 | UNCHECKABLE (JS-regex-only) | CONFIRMED `Taskfile.yml:2715` — with **no** signature check above it, exactly as the plan states |
| `SIG_DIR` two-temp-dirs precedent | shell-var | 04-05 | UNCHECKABLE (JS-regex-only) | CONFIRMED `Taskfile.yml:2529-2533` |
| `cosign verify-blob` (pre-existing sites) | shell-stmt | 04-05 | UNCHECKABLE (JS-regex-only) | CONFIRMED — **4 invocations** at `Taskfile.yml:1492`, `:1494`, `:2541`, `:3087`; **10 matching lines** total. Contradicts the plan's "up from 2" baseline — see Findings |
| `--certificate-identity-regexp` literals | shell-flag | 04-05 | UNCHECKABLE (JS-regex-only) | CONFIRMED — **exactly 4** today: `README.md:59`, `docs/RELEASE.md:62`, `Taskfile.yml:2544`, `:3090`. Equals the plan's floor — see Findings |
| `preconditions:` on `verify:self-upgrade` | task-field | 04-05 | UNCHECKABLE (JS-regex-only) | CONFIRMED `Taskfile.yml:2563` (block), `:2574` gh, `:2576` jq — no `cosign` entry yet, as the plan assumes |
| `homebrew_casks:` | cask-hook | 04-02 | UNCHECKABLE (JS-regex-only) | CONFIRMED `.goreleaser.yaml:458` |
| `hooks.post.install` / `hooks.post.uninstall` | cask-hook | 04-02 | UNCHECKABLE (JS-regex-only) | CONFIRMED `.goreleaser.yaml:524-526` and `:595` |
| BREW-05 raise #1 (man-page count) / #2 (version parity) | cask-hook | 04-02 | UNCHECKABLE (JS-regex-only) | CONFIRMED `.goreleaser.yaml:554` (cited 553-555), `:567` (cited 566-568) |
| `system_command binary, args: ["man", man_dir]` | cask-hook | 04-02 | UNCHECKABLE (JS-regex-only) | CONFIRMED `.goreleaser.yaml:551` — count is 1 today, matching the gate's `-eq 1` |
| `man_dir` / `real_bin_dir` / `sentinel` | cask-hook (Ruby local) | 04-02 | UNCHECKABLE (JS-regex-only) | CONFIRMED `.goreleaser.yaml:543`, `:620`; `:577`; `:578`, `:623` |
| `atomic_write` heredoc | cask-hook | 04-02 | UNCHECKABLE (JS-regex-only) | CONFIRMED `.goreleaser.yaml:579-588` (cited 577-587) |
| `FileUtils.rm_f` (man cleanup + marker removal) | cask-hook | 04-02 | UNCHECKABLE (JS-regex-only) | CONFIRMED `.goreleaser.yaml:621`, `:624` — post-removal "exactly 1" is correct |
| `auto_updates` comment + falsified D-08 line | cask-hook (comment) | 04-02 | UNCHECKABLE (JS-regex-only) | CONFIRMED `.goreleaser.yaml:447-448`; falsified claim at `:451` (exact cited line) |
| `.codegraph-brew-install` (marker filename) | string-literal | 04-02 | UNCHECKABLE (JS-regex-only) | CONFIRMED `.goreleaser.yaml:578`, `:623`; `Taskfile.yml:2009`. Count in `.goreleaser.yaml` is 2 today, so the gate's `-eq 0` is genuinely RED |
| `self-upgrade` job | workflow-job | 04-05 | UNCHECKABLE (JS-regex-only) | CONFIRMED `.github/workflows/post-release-verify.yml:245`; invokes `task verify:self-upgrade` at `:280` (exact cited line) |
| deliberate absence of `needs: verify-supply-chain` | workflow-job | 04-05 | UNCHECKABLE (JS-regex-only) | CONFIRMED — rationale comment at `post-release-verify.yml:241-244`; `verify-supply-chain` at `:209` |
| ROADMAP Phase-4 checkbox item | doc-passage | 04-03 | UNCHECKABLE (JS-regex-only) | CONFIRMED `.planning/ROADMAP.md:97` — exactly one `- [ ] **Phase 4:` line, and it says "Cellar", so the gate's `-eq 1` on `Caskroom` is genuinely RED today |
| ROADMAP milestone-ordering bullet / Phase-3 Notes / Phase-4 block | doc-passage | 04-03 | UNCHECKABLE (JS-regex-only) | CONFIRMED `.planning/ROADMAP.md:91`, `:187`, `:209`, `:211`, `:212`, `:216` |
| `UPGR-01` / `UPGR-02` / `UPGR-03` | requirement | 04-03, all | UNCHECKABLE (JS-regex-only) | CONFIRMED `.planning/REQUIREMENTS.md:40`, `:41`, `:42`; traceability rows `:86-88`, `:97` |
| PROJECT.md scope bullet + Key-Decisions row | doc-passage | 04-03 | UNCHECKABLE (JS-regex-only) | CONFIRMED `.planning/PROJECT.md:55`, `:184` (both carry "Cellar" and "path-prefix guess") |
| `.planning/STATE.md:191` mirrored decision line | doc-passage | 04-03 | UNCHECKABLE (JS-regex-only) | CONFIRMED — exact cited line |
| `getMilestonePhaseFilter` | js-func (external tool) | 04-03 | UNCHECKABLE (outside the indexed repo) | CONFIRMED `~/.claude/gsd-core/bin/lib/roadmap-parser.cjs:525` (exported :759) |
| `03-EVIDENCE.md` "leave the sentinel behind" (2 sites) | doc-passage | 04-02 | UNCHECKABLE (JS-regex-only) | CONFIRMED `.planning/phases/03-homebrew-tap-cask/03-EVIDENCE.md:831`, `:1039` — both exact cited lines |
| `03-EVIDENCE.md` "30 orphaned" / § BREW-01 | doc-passage | 04-02, 04-06 | UNCHECKABLE (JS-regex-only) | CONFIRMED `03-EVIDENCE.md:815`, `:754` |
| `README.md` future-tense brew claim / `bash-completion` prerequisite | doc-passage | 04-04 | UNCHECKABLE (JS-regex-only) | CONFIRMED `README.md:81` (block :68-81, inside cited :60-92); `README.md:77` |
| `docs/RELEASE.md` `**Upgrading.**` para / three false claims | doc-passage | 04-04 | UNCHECKABLE (JS-regex-only) | CONFIRMED `docs/RELEASE.md:535`; claims at `:540-542` |
| `docs/RELEASE.md` `brew trust` instructions | doc-passage | 04-04, 04-06 | UNCHECKABLE (JS-regex-only) | CONFIRMED `docs/RELEASE.md:525`, `:531`, `:570`, `:573` (≥2, satisfying the floor) |
| `docs/RELEASE.md` "Verified 2026-08-10 (plan 03-05)" block | doc-passage | 04-04 | UNCHECKABLE (JS-regex-only) | CONFIRMED `docs/RELEASE.md:544` |
| `.planning/todos/pending/` — 10 items | file-path | 04-02, 04-05 | UNCHECKABLE (JS-regex-only) | CONFIRMED — 10 today; 04-02 closes 2 → 8, 04-05 closes 1 → 7. The plans' `-eq 8` / `-eq 7` arithmetic is internally consistent |
| the three specific todo files closed by 04-02 / 04-05 | file-path | 04-02, 04-05 | UNCHECKABLE (JS-regex-only) | CONFIRMED (all three present in `.planning/todos/pending/`) |
| `.planning/todos/completed/` (git-mv destination) | file-path | 04-02, 04-05 | UNCHECKABLE (JS-regex-only) | CONFIRMED (directory exists) |
| `04-{CONTEXT,RESEARCH,PATTERNS,VALIDATION,DISCUSSION-LOG}.md` | file-path | all | UNCHECKABLE (JS-regex-only) | CONFIRMED — all five exist in the phase directory |
| `04-PATTERNS.md` §§ brew.go / .goreleaser.yaml / Taskfile.yml / folded-todo-3 | doc-section | 04-01, 04-02, 04-05 | UNCHECKABLE (JS-regex-only) | CONFIRMED `04-PATTERNS.md:21`, `:177`, `:219`, `:254` |
| `04-VALIDATION.md` §§ Wave 0 Requirements / Manual-Only Verifications | doc-section | 04-01, 04-06 | UNCHECKABLE (JS-regex-only) | CONFIRMED `04-VALIDATION.md:70`, `:85` |
| RESEARCH assumption A3 (no `sentinel`/`brew-install` in the two shape tests) | go-test (negative claim) | 04-02 | UNCHECKABLE (Go) | CONFIRMED STILL TRUE (searched `rg -e 'sentinel' -e 'brew-install' internal/upgrade/goreleaser_shape_test.go internal/upgrade/taskfile_shape_test.go` → no matches) |
| `seanb4t/homebrew-tap` published tap; Homebrew on the host | external | 04-06 | UNCHECKABLE (outside the repo and outside intel's reach) | NOT VERIFIABLE FROM THE WORKING TREE — network/host preconditions. 04-06 Task 1 already declares them as a `<precondition>` that halts rather than substitutes |

### Excluded — produced by this phase

Symbols and paths each plan declares under its own "Artifacts this phase produces" section were **not**
checked for existence; their absence is intended and is not a finding.

- **04-01** — `internal/upgrade/brew.go`, `internal/upgrade/brew_test.go`; `brewInstall` (+ fields
  `ResolvedBinary`, `InstallDir`, `Tree`, `Token`, `Version`, `Receipt`), `detectBrewManaged`,
  `brewPointerMessage`, `brewReceiptFilename`; `TestDetectBrewManaged`,
  `TestDetectBrewManaged_RealInstall`, `CODEGRAPH_BREW_ACCEPTANCE_PATH`,
  `TestUpgradeRun_RefusesBrewManagedCask`, `TestUpgradeRun_CheckBrewManagedStepsAside`,
  `TestUpgradeRun_ForceDoesNotOverrideBrewRefusal`.
- **04-02** — `hook_started_at`, `fresh_man_pages` (Ruby locals in `.goreleaser.yaml`);
  `MAN_BASELINE_MARKER`, `FRESH_MAN_PAGE_COUNT`, `CASK-REHEARSE-EVIDENCE schema=3` (`Taskfile.yml`). Its
  declared **removals** (`SENTINEL_PATH`, `SENTINEL_VERDICT`, `SENTINEL_FIRST_LINE`, `REAL_BIN`,
  `REAL_BIN_DIR`, `real_bin_dir`, `sentinel`) were checked as *currently existing* and appear in the table
  above — they must be present for the deletion to be meaningful.
- **04-03** — no new code symbols; amended passages only in `.planning/ROADMAP.md`,
  `.planning/REQUIREMENTS.md`, `.planning/PROJECT.md`.
- **04-04** — `TestUpgradeCommand_HelpDocumentsBrewRefusalAndExitCodes`; the amended `Long`/`Example`
  string values; amended `README.md` and `docs/RELEASE.md` passages. No new production symbols.
- **04-05** — `SIG_DIR`, `PRIOR_SIG`, `SELF-UPGRADE-VERIFY-EVIDENCE` (`Taskfile.yml`);
  `TestCosignIdentityPolicyMatchesCompiledPattern`, `extractCosignIdentityLiterals`,
  `TestCosignIdentityPolicyMatchesCompiledPattern_ZeroLiteralsIsError`, `cosignIdentitySANCorpus`
  (`internal/upgrade/taskfile_shape_test.go`).
- **04-06** — `.planning/phases/04-codegraph-upgrade-homebrew/04-EVIDENCE.md` and its
  `UPGR-ACCEPTANCE-EVIDENCE` line. No code symbols.

### Findings from the confirmation sweep

- **No symbol a plan claims already exists was missing.** Every Go function, method, struct field,
  package-level const, test function, Taskfile target, shell variable, GoReleaser cask hook, workflow job,
  Markdown passage and `.planning/` path asserted as pre-existing was located in the working tree, and
  every cited line number that was spot-checked matched or fell inside its cited range. The two
  claimed-absent items (`internal/upgrade/brew.go`, `brew_test.go`) were confirmed absent as the plan
  states, and RESEARCH assumption A3's negative result still holds.
- **Three claimed *quantities* were wrong, and all three weaken a gate** (carried into the Agreed Concerns
  above): the `cosign verify-blob` baseline (plan says 2, measured 10 lines / 4 invocations, so the `-ge 3`
  floor is pre-satisfied); the `--certificate-identity-regexp` literal floor (plan's `t.Fatalf` triggers
  below 4, measured exactly 4 today, so the drift guard is pre-satisfied); and 04-02's `^TestGoreleaser`
  gate.
- **Correction to an earlier reading of the `^TestGoreleaser` gate.** `go test -run` is **package-scoped,
  not file-scoped**, so `go test ./internal/upgrade/ -run '^TestGoreleaser'` is *not* an empty-match
  vacuous pass — it matches and runs exactly one test, `TestGoreleaserPinParity`
  (`internal/upgrade/taskfile_shape_test.go:761`), which passes today. The real defect is narrower:
  that test reads `tools/go.mod` and `.github/workflows/release.yml` for GoReleaser version-pin parity and
  reads **nothing** in `.goreleaser.yaml`, so the criterion's gloss "the shape tests that pin this file's
  invariants still pass" (`04-02-PLAN.md:176-178`) is false, and the gate is green before and after Task 1
  regardless of what happens to the cask hooks. Zero tests in `internal/upgrade/goreleaser_shape_test.go`
  match `^TestGoreleaser` (they are named `TestHomebrewCask*`, `TestParseGoreleaser*`, `TestNotarize*`,
  `TestSigns*`, `TestSboms*`, `TestRelease*`, …). Suggested correction: `-run '^TestHomebrewCask'`, which
  reaches `TestHomebrewCaskHooksHaveStructuralProperties` — the test that actually pins the hook block.
  Severity MEDIUM (mis-scoped coverage claim), not HIGH; the paired `go test ./internal/upgrade/...` leg
  does execute the real shape tests.
- **One naming adjacency, not a defect.** 04-05 introduces `SELF-UPGRADE-VERIFY-EVIDENCE` while
  `SELF-UPGRADE-EVIDENCE` already exists at `Taskfile.yml:2735` in the same target. Neither string
  contains the other, so counting gates on either name stay unambiguous.
