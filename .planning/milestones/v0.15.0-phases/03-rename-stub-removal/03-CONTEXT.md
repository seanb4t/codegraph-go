# Phase 3: Rename-Stub Removal - Context

**Gathered:** 2026-09-26
**Status:** Ready for planning

<domain>
## Phase Boundary

This phase removes the hidden `query` and `unlock` rename stubs that v0.14.0 promised to remove in the next minor release (backlog 999.5, promoted as VERB-09). It then lets this phase's close be the first observed firing of the Phase 2 capability (CAP-05). The capability writes the milestone's `Breaking` fragment, and `changie next auto` answers `v0.15.0`.

In scope:
- Delete `internal/cli/renamed.go` and `internal/cli/renamed_test.go`.
- Drop `newQueryCmd()` and `newUnlockCmd()` from `root.go`'s `AddCommand` list.
- Remove the two `codegraph query` / `codegraph unlock` lines from `internal/cli/testdata/cli-reference-allowlist.txt`.
- Handle every other artifact that exists only because of the stubs.
- Re-run the census and write the evidence for the capability firing.

Out of scope:
- Changing the capability. It is at v0.1.2. A defect found at this close goes into a follow-up capability release, not into this phase.
- Removing release-please (Phase 5).
- The fragment gate (Phase 4).
- Any new CLI behaviour: no aliases, no suggestions, no new hints.

</domain>

<decisions>
## Implementation Decisions

### What "removed" means on the command line
- **D-01:** After removal, `codegraph query …` and `codegraph unlock …` fail with **cobra's own unknown-command error** and the process exits 1. The error comes from `root`'s `legacyArgs`: `unknown command "query" for "codegraph"`, printed once by `cmd/codegraph/main.go`'s single error path.
  - No `SuggestFor` entry on `search` or `daemon`, no alias, no custom hint text. ROADMAP SC1 says "unknown commands rather than printing a rename message", and a hint that names the replacement verb is a rename message in another form.
  - The researcher confirms the exact stderr text by running it, including for the flag-bearing forms the old tests used: `query --json main` and `unlock -p <dir>`. The root command has no `Args` and no `RunE`, so `legacyArgs` should apply. Also confirm that cobra's prefix/Levenshtein suggestion finds nothing for either name. A "Did you mean" line would change the expected text.
  - **Reversibility:** reversible. Adding a hint later is a one-line `SuggestFor`.

### Guard for the removal (tests)
- **D-02:** `internal/cli/renamed_test.go` is deleted, as the roadmap says.
  - **`test/integration/renamed_stubs_test.go` is not named in the roadmap, but it exists only for the stubs.** It spawns the real binary and asserts the old two-line rename text, so it goes RED the moment the stubs go.
  - It is **rewritten in place as the removal's guard**, renamed to `test/integration/removed_verbs_test.go` or similar (Claude's discretion). It keeps its four invocations: bare `query`, `query --json`, bare `unlock`, and `unlock` with a path. For each, it asserts:
    - exit status 1
    - empty stdout
    - stderr equals D-01's unknown-command line exactly
  - Rule `84d1gfpywd` applies: the guard asserts **positively** that each invocation reached cobra's unknown-command path. "Did not print the rename message" is not an acceptable assertion. The guard prints and asserts an executed-case count of 4, and fails on 0.
  - This is the RED-first test for the removal (rule `x1cjy9vyhq`). The `test(03-NN):` commit lands while the stubs still exist. Its `--- FAIL:` transcript is pasted into the SUMMARY. The `feat` commit that deletes the stubs follows.
  - Rejected: deleting both tests with no replacement. Nothing would then stop a future alias from reviving the verbs, and SC1's "exit non-zero" would rest on a one-off manual run.
  - Rejected: a unit-only guard through `execCmd`. It does not exercise `main.go`'s exit path, and SC1 is about the process exit status.
- **D-03:** No other test or golden should change. Hidden commands are already excluded from the generated reference, completions, the man tree, `documentedByReference`, and `commandGroups`.
  - The researcher confirms this by grepping for `query`/`unlock` **as command registrations**, not as the many unrelated MCP `"query"` argument keys in `testdata/wireoracle/**` and `test/wireoracle/**`.
  - If any golden does change, treat it as a finding and explain it. Do not regenerate it blindly.
  - `task docs:cli` must produce **no** diff to `docs/CLI-REFERENCE.md` (SC2).
  - `task docs:cli:drift` and `TestEveryRegisteredFlagIsAccountedFor` must be green after the two allowlist lines are removed. The allowlist header says "an entry matching nothing FAILS the guard", so leaving the lines in place would itself go RED. That is the removal's built-in second guard.

### Commits
- **D-04:** The stub removal is one commit whose type is `feat(cli)!:`, with a `BREAKING CHANGE:` footer. The footer names both removed commands and their replacements: `search --full` and `daemon unlock`.
  - The `!` is load-bearing. The capability's kind rule (Phase 2 D-06) classifies a phase commit carrying `!` / `BREAKING CHANGE` as `Breaking`.
  - Never `[ci skip]` (rule `f18zrdsgx5`).
  - Pipeline commits may skip signing only as rule `xmz3xknbj0` allows.
  - **Reversibility:** reversible.
- **D-05:** Stale comments that describe the stubs are updated in the same `feat` commit, not left to rot:
  - `root.go:14-15`, `:46`, and `:99-100` (the "hidden rename stubs (renamed.go) … for one release" prose)
  - any comment elsewhere found by the census or by a grep for `renamed.go`

  CHANGELOG.md and `.changes/v*.md` are tool-owned and are **never** edited, even where they say "removed in v0.15.0".

### Census (SC2)
- **D-06:** The v0.14.0 census instrument is re-run **verbatim**, byte-for-byte, from `.planning/milestones/v0.14.0-phases/03-verb-fold/03-MUTATION-LOG.md` Family (c):
  ```
  rg -nU -w --hidden 'codegraph\s+(query|unlock)' --glob '!.planning/**' --glob '!CHANGELOG.md' --glob '!web/build/**' --glob '!.git/**' .
  ```
  - The positive control runs first: the untracked `.github/__census_control__.md` with the same three lines, inside `trap 'rm -f …' EXIT`, never `git add`ed. It must be found on `:1:` and `:3:` before the clean-tree run is trusted.
  - The pre- and post-mutation `git status --porcelain .github` gates are kept.
  - The clean-tree run must report **zero hits**. Today it reports exactly four: `renamed.go:10-11` and the two allowlist lines. All four go away with D-01/D-05.
  - Evidence goes into a new `03-MUTATION-LOG.md` family entry in this phase's directory, mirroring the v0.14.0 layout.
  - **Consequence for new text:** the rewritten guard (D-02), its comments, the updated `root.go` prose (D-05) and the SUMMARY-facing commit message must not contain the two-token phrase `codegraph query` / `codegraph unlock` in any tracked file outside `.planning/`. Refer to the verbs as `query` / `unlock`, or write the invocation as argv (`"query", "main"`).
- **D-07:** The census is VERB-09's evidence **at execution time, before the phase close**. The capability then writes a fragment into `.changes/unreleased/`, which the census does not exclude. If that fragment's body spells `codegraph query`, a later re-run would hit it.
  - A hit **confined to the capability-written fragment file** is the changelog describing the removal, not a surviving reference. The verifier records it as expected and does not fail on it.
  - The instrument is **not** edited to exclude `.changes/`. The v0.14.0 "reused verbatim" contract wins over a cosmetically clean re-run.

### Capability firing and its evidence (SC3, CAP-05)
- **D-08:** The fragment is **not** pre-written in any plan. The capability writes it at the phase close, or CAP-05 is not met.
  - The phase is executed through `/gsd-execute-phase`. Its `aggregate_results` step dispatches `verify:post` (`execute-phase.md:1105`) **before** `verify_phase_goal` runs `gsd-verifier` (`:1186`). So the fragment commit already exists when VERIFICATION.md is written.
  - The verifier records the evidence **in VERIFICATION.md**, as ordinary verification of the phase's must-haves:
    - the fragment filename under `.changes/unreleased/`
    - its kind (`Breaking`)
    - its body
    - the commit SHA
    - the `Changie-Phase: 3` / `Changie-Summaries: …` trailers
    - proof that the commit is on `gsd/v0.15.0-milestone`
  - No invented section or file outside what the verifier already writes (`planning-artifacts` rule).
  - The planner puts these as must-have truths in the plan, so the verifier has something concrete to check.
  - **Reversibility:** reversible.
- **D-09:** To make the model-judgment half of the skill reach `Breaking` with an accurate sentence, the SUMMARY states the change in **user terms**:
  - which commands were removed
  - that invoking them is now an unknown-command error
  - what to run instead: `search --full <term>` and `daemon unlock [path]`

  Phase 2 D-06's illustrative body ("run `codegraph explore` instead") is **wrong** for this change and must not be copied. The replacements are `search --full` and `daemon unlock`.

  The capability chooses its own sentence. This context does not dictate the wording.
- **D-10:** If the capability-written fragment is **inaccurate**, for example it names the wrong replacement or uses the wrong kind:
  - Correct it in a **separate** commit (`docs(03): correct changelog fragment`).
  - Record both SHAs in VERIFICATION.md.
  - CAP-05 is still satisfied by the firing itself. The correction is logged as a capability-judgment finding for a later capability release.

  Do not amend or rewrite the capability's commit (no rebase or amend per user preference).
- **D-11:** If the step **does not fire** (skip, error, or not dispatched), the roadmap Notes already decide the fallback:
  - A hand-written `Breaking` fragment via `task changie -- new -k Breaking -b "…"` is committed, so VERB-09 is still met.
  - It is recorded as hand-written.
  - CAP-05 stays **open** and is re-attempted at Phase 4's close (then Phase 5's).
  - The skip reason the capability printed is quoted verbatim in VERIFICATION.md.

### Version check (SC4)
- **D-12:** With the fragment present, `task changie -- next auto` is run on the milestone branch. Its output must be `v0.15.0`, a minor bump over `.changes/v0.14.0.md`, because `Breaking` is `auto: minor` pre-1.0. The command and its verbatim output are recorded in VERIFICATION.md.
  - `changie` is not on `PATH` in this repo. Always call it through `task changie --` (Phase 2 D-02).

### Claude's Discretion
- How many plans to use. The code change is small. One plan with the RED test, the removal, and the census is expected, but the planner may split the census/evidence work.
- The new guard's filename and test function name.
- Exact wording of the updated `root.go` comments, within D-05/D-06's constraint.
- Whether `.planning/phases/999.5-remove-the-query-unlock-rename-stubs/` (only a `.gitkeep`) is left alone. Default: leave it. ROADMAP says backlog entries are preserved across milestone closes and 999.5 is annotated with its consuming phase. Removing the directory is the milestone close's call, not this phase's.

### Folded Todos
- **Adopt changie for changelog + version and replace release-please** (`.planning/todos/pending/2026-09-25-adopt-changie-replace-release-please.md`, score 0.9). This is folded as **context only**, as in Phase 2: this phase delivers the milestone's `Breaking` fragment (the 999.5 item the milestone scoping decision assigned to v0.15.0). The todo stays pending for Phase 5.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope and requirements
- `.planning/ROADMAP.md` §"Phase 3: Rename-Stub Removal": goal, four success criteria, and the Notes on verbatim census reuse, the hand-written fallback, and CAP-05 re-attempts
- `.planning/ROADMAP.md` §"Phase 999.5: remove the query/unlock rename stubs": the originating backlog entry
- `.planning/REQUIREMENTS.md`: VERB-09 (line 60) and CAP-05 (line 50)

### Census instrument (reused verbatim)
- `.planning/milestones/v0.14.0-phases/03-verb-fold/03-MUTATION-LOG.md` §"Family (a)" (line 23) and §"Family (c)" (line 703): the exact `rg` invocation, the `--hidden` correction, the positive-control file and trap, the pre/post gates, and the evidence layout to mirror

### Capability behaviour at the close
- `.planning/phases/02-phase-close-fragment-capability/02-CONTEXT.md` D-02 (`workflow.changie_command` = `task changie --`), D-06 (user-visible and `Breaking` rules; its example body is wrong for this change, see D-09), D-07 (idempotency trailers), D-08 (fragment commit convention), D-13 (PR optional)
- `.planning/phases/02-phase-close-fragment-capability/02-VERIFICATION.md` §"Requirement CAP-05 (Correctly Out of Scope)": the Phase 2 skip-path observation that this phase's firing follows
- `~/.claude/gsd-core/workflows/execute-phase.md` `aggregate_results` (≈line 1105, `verify:post` dispatch) and `verify_phase_goal` (≈line 1186): the ordering D-08 relies on
- `.changie.yaml`: kinds and auto bumps (`Breaking: auto: minor`), optional `PR`

### The stubs being removed
- `internal/cli/renamed.go`: the stubs and their doc comment, which records the D-05/D-06/D-09/D-12 history from v0.14.0
- `internal/cli/renamed_test.go`: unit tests being deleted
- `test/integration/renamed_stubs_test.go`: binary-level test rewritten per D-02
- `internal/cli/root.go:14-15, :46, :99-100, :127, :130`: registration and stale prose
- `internal/cli/testdata/cli-reference-allowlist.txt:1-9`: allowlist grammar ("an entry matching nothing FAILS the guard") and the two lines being removed
- `internal/cli/cli_reference_test.go`: `TestEveryRegisteredFlagIsAccountedFor` / `documentedByReference`

### Repo rules in force
- engram rules `84d1gfpywd` (positive-assertion guards), `x1cjy9vyhq` (Go RED evidence), `f18zrdsgx5` (no `[ci skip]`), `xmz3xknbj0` (commit signing)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `test/integration` helpers `copyFixture`, `runBinary`: already used by `renamed_stubs_test.go`, so the rewritten guard reuses them unchanged.
- `internal/cli` `execCmd`, `newRootCmd`: available if the planner wants an additional in-process assertion. Not required by D-02.
- `Taskfile.yml` `changie` task (`task changie -- …`): the only way to run changie in this repo.

### Established Patterns
- **Hidden-command allowlisting:** a bare `<command path>` allowlist line covers a hidden command's flags. Unmatched lines fail the guard, so the allowlist removal is mandatory, not cleanup.
- **Single error exit:** every command sets `SilenceUsage`/`SilenceErrors`, and `cmd/codegraph/main.go` prints the returned error once and exits 1. Unknown-command errors from cobra take the same path.
- **Mutation-log evidence:** each guard's RED is demonstrated against a confirmed-applied, byte-cleanly-reverted mutation, recorded in `NN-MUTATION-LOG.md`. Phases 1 and 2 of this milestone and v0.14.0 Phase 3 all use this format.

### Integration Points
- `root.go` `AddCommand` list (lines 126-131): two constructor calls removed. The group loop after it is unaffected, because the stubs never carried a `GroupID`.
- The `verify:post` capability dispatch at phase close: it reads this phase's `*-SUMMARY.md` and the phase commits (D-04's `!`).

</code_context>

<specifics>
## Specific Ideas

- The census's current four hits (`renamed.go:10-11`, allowlist `:8-9`) are the exact set expected to disappear. Any other hit after removal is a finding.
- Expected `changie next auto` answer: `v0.15.0`.
- The milestone PR is draft #88 (memory `y6y5vfkjs0`). The fragment may carry `PR=88` through the capability's open-PR lookup, or no PR. Both are valid per D-13.

</specifics>

<deferred>
## Deferred Ideas

- A `SuggestFor` / "did you mean `search --full`" hint for the removed verbs: rejected for this phase (D-01). A future CLI-ergonomics phase could revisit it.
- Capability judgment defects found at this close (D-10): these go into a later `gsd-capability-changie` release, not this phase.

### Reviewed Todos (not folded)
- **tools/bench/runner `pinnedAt()` validates a checkout by `git rev-parse HEAD` alone** (score 0.6): a bench-integrity item, unrelated to stub removal. It matched on generic keywords only.
- **Reply on GH #85 / #82 / #80 with the reshaped contract** (score 0.6): outward-facing, server-mode, and belongs to the milestone after v0.15.0.
- **Pin the Go toolchain so Go 1.27 can build codegraph-go** (score 0.6): build tooling, not in this phase's boundary.

These three scored at or above the 0.4 auto-fold threshold only on generic keywords ("phase", "minor", "test"). Folding them would be scope creep, so auto mode reviewed them and did not fold them.

</deferred>

---

*Phase: 03-rename-stub-removal*
*Context gathered: 2026-09-26*
