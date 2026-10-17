# Phase 4: Fragment-Required Gate - Context

**Gathered:** 2026-09-26
**Status:** Ready for planning

<domain>
## Phase Boundary

A pull request that changes shipped code cannot merge without a changelog fragment or an explicit, reasoned exemption. A `fragment-required` check runs on every pull request, fails closed, and is a required status check on `main` (GATE-01…GATE-05).

In scope:
- `scripts/fragment_policy.py` — the one home of the path rule and the verdict, with a stdlib `unittest` file beside it.
- A new workflow `fragment-required.yml` (`pull_request_target`, base-branch checkout) whose job body is a `task` invocation.
- Removing the two ruleset guards (`TestRequiredCheckNamesPreserved` and its helpers, `scripts/check-ruleset-drift.sh`, its `ci.yml` step, `.github/required-status-checks.txt`) — D-08.
- Landing the gate plus the guard removal on `main` through one PR off `main`, then the maintainer adding `fragment-required` to ruleset `protect-main` — D-05…D-07.

Out of scope:
- `CONTRIBUTING.md`'s prose on how to add a fragment or exempt — `DOCS-12`, Phase 5.
- Any edit to `scripts/pr_template_policy.py` — GRD-15 edits it alone in Phase 5 (D-10).
- Tests for `pr_template_policy.py` — deferred.
- Any change to the phase-close capability.

</domain>

<decisions>
## Implementation Decisions

### What is tested (applies to every plan)
- **D-01:** Tests exercise **this repository's code only** — the verdict function in `scripts/fragment_policy.py`, fed literal inputs: a changed-file list (path + `changeType`), a PR body, and error states (empty list, API error, unreadable body). No test builds a git repository, calls GitHub, runs a workflow, or runs changie to observe how those tools behave. GATE-04's "real fixtures" are these literal input cases; its "both directions" is the same path list with and without an ADDED fragment. The RED mutation (rule `84d1gfpywd`) mutates **our** path rule and shows **our** tests fail, then is reverted byte-cleanly; the executed-case count is asserted so the suite cannot pass over zero cases.
  - Why: maintainer, explicitly — "It had best not be how git, github, gh workflows, or changie _work_."

### Trigger and trust model
- **D-02:** `fragment-required.yml` runs on **`pull_request_target`** and checks out the **base branch** (no `ref:`), exactly as `pr-template-format.yml` does, so the policy that judges a PR always comes from `main` and a PR cannot weaken the gate by editing it. Untrusted input (PR body, changed paths) reaches Python only through environment variables, never interpolated into shell. — **Reversibility:** reversible — a trigger edit, but it drives D-05's landing sequence.
- **D-03:** The workflow posts **no PR comment**. Guidance is the check output only: `::error::` lines that name the matched shipped-code paths, how to add a fragment (`task changie -- new …`), and the `<!-- changelog-exempt: <reason> -->` marker. Permissions are therefore `contents: read` and `pull-requests: read` only — no `issues: write`, no `actions/github-script`.

### What counts as a fragment and an exemption
- **D-04:** "Has a fragment" = at least one changed file under `.changes/unreleased/` whose `changeType` is **ADDED** and whose name is not `.gitkeep`. MODIFIED, RENAMED or DELETED files there do not satisfy it — a new code change needs its own entry. Shipped code = `cmd/**` or `internal/**`, excluding `*_test.go` and any path under a `testdata/` directory (GATE-01, verbatim). Exemption marker grammar mirrors `pr_template_policy.py`'s `EXEMPT_MARKER`: `<!-- changelog-exempt: <reason> -->`, case-insensitive, non-empty reason required (GATE-02). A PR touching no shipped code passes without a marker (planning-only PRs, the Phase 5 release PR).

### Landing sequence
- **D-05:** Because of D-02 the workflow must exist on `main` before it reports on any PR (including #88). The gate reaches `main` through **one PR off `main`** carrying only the gate (workflow, task target, `fragment_policy.py` + tests, shape-test population updates) **and** the D-08 guard removal. Commit types are `ci:` / `test:` / `chore:` so release-please (still live on `main` until Phase 5) opens no release. After it merges, the maintainer adds `fragment-required` to `protect-main` and the edit's timestamp is recorded. No admin-bypass merge is needed: the check is not required until after it exists on `main`.
  - Rejected: merging the milestone-so-far to `main` (the v0.13.0 PR #69 precedent) — it would put Phase 3's `feat(cli)!` on `main` while release-please is live, so release-please would open its own release PR and CHANGELOG edits in collision with changie.
  - Rejected: waiting for the milestone merge — Phase 4 could not close on a working gate.
- **D-06:** Commits are authored **on `gsd/v0.15.0-milestone`** as usual (the planning trail and RED-then-GREEN commit order stay together). A plan task cherry-picks the code-only commits (no `.planning/`) onto a branch off `main` for the D-05 PR; after it merges, `main` is **merged** back into the milestone branch (no rebase).
- **D-07:** The ruleset edit is a **blocking-human checkpoint** (maintainer repository-settings action, the v0.14.0 `GRD-12` precedent for the checkpoint shape). Before it, re-read the live ruleset read-only (it had 8 contexts on 2026-09-26, `strict: false`, admin bypass `RepositoryRole 5`); after it, re-read and record that `fragment-required` is present plus the timestamp.

### Ruleset guards removed
- **D-08:** Delete both ruleset guards. Removal travels to `main` inside the D-05 PR (otherwise `check-ruleset-drift.sh` would go red on every PR to `main` the moment the ruleset gains a context the `main` copy of the list lacks):
  - `TestRequiredCheckNamesPreserved`, `TestRequiredCheckNamesPreserved_ZeroJobsIsError`, the `TestReadRequiredCheckNames_*` tests, `readRequiredCheckNames`, `requiredStatusChecksPath` (all in `internal/upgrade/taskfile_shape_test.go`), and the comment at `:1649-1654` that cites them.
  - `scripts/check-ruleset-drift.sh`, and the `Ruleset drift check (GRD-12)` step in `ci.yml` (`:236-252`), plus whatever run-body exception entry covers that step.
  - `.github/required-status-checks.txt`.
  - Why: T-10-01-05's premise ("a renamed required check silently un-gates `main`") is wrong — a required context that never reports blocks every PR at "Expected — waiting", which is loud; this milestone's own roadmap hazard relies on that. `check-ruleset-drift.sh` only catches the maintainer editing their own settings without editing a text file (declined once as GRD-07 at v0.13.0). Neither tests the gate's correctness. Maintainer decision, 2026-09-26.
- **D-09:** **GATE-05 is amended** to: "`fragment-required` is a required status check in ruleset `protect-main`, the edit's timestamp recorded." The clauses about the `.github/required-status-checks` fixture, `TestRequiredCheckNamesPreserved` and `scripts/check-ruleset-drift.sh` are dropped. ROADMAP Phase 4 success criterion 5 and its Notes (the "7 contexts" / "re-verify count" text) follow. The planner updates `REQUIREMENTS.md` and the criterion text (values in existing shapes — no new structure, per the planning-artifacts rule).

### Code placement
- **D-10:** New file **`scripts/fragment_policy.py`** — a pure verdict function plus a `__main__` that reads env and writes the result — and **`scripts/test_fragment_policy.py`** (stdlib `unittest`, no new dependency), run by a Taskfile target wired into `ci.yml`'s test job so `TestWorkflowRunBodiesInvokeTask` covers it. `pr_template_policy.py` is not touched (this supersedes the ROADMAP Phase 4 Notes' "one file, two edits" framing; GRD-15 in Phase 5 edits it alone). Python is already on the runner; `scripts/` is outside `cmd/` and `internal/`, so the gate cannot trip its own rule.

### Claude's Discretion
- Event types (mirror `pr-template-format.yml`: `opened, edited, reopened, synchronize, ready_for_review` — `edited` so adding a marker re-evaluates).
- How the changed-file list reaches the script: the script calls `gh` itself, or a prior step writes it using the per-run `$GITHUB_OUTPUT` delimiter (FIX-10 / `task check:workflow-output-delimiter`). Either way GATE-03's three failures (empty list, API error, unreadable body) must each be a distinct message and a fail.
- The Taskfile target names, the job/check `name:` (must read exactly `fragment-required` for the ruleset), and the exact `::error::` wording.
- Where `fragment-required.yml` sits in `inScopeWorkflowFiles` / `inScopeJobs` vs `workflowFileExceptions` — its body is `task`, so in scope is expected.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements and roadmap
- `.planning/REQUIREMENTS.md` — GATE-01…GATE-05 (GATE-05 amended by D-09)
- `.planning/ROADMAP.md` §"Phase 4: Fragment-Required Gate" and the milestone ordering bullets — hazards and precedents

### Precedents to reuse
- `.github/workflows/pr-template-format.yml` — `pull_request_target` + base-branch checkout + env-only untrusted input + Python policy; the shape D-02 copies
- `.github/workflows/require-issue-link.yml` — fail-closed empty-list handling, non-empty-reason exemption marker, per-run delimiter
- `scripts/pr_template_policy.py` — `EXEMPT_MARKER` regex (`:83`) the `changelog-exempt` grammar mirrors; `emit()` output shape
- `scripts/check-workflow-output-delimiter.sh` — FIX-10 guard over multi-line `$GITHUB_OUTPUT` writes
- `.github/actions/install-task` — local composite action ci.yml uses to install task

### Guards touched
- `internal/upgrade/taskfile_shape_test.go` — `TestWorkflowRunBodiesInvokeTask` (`:1573`), `inScopeWorkflowFiles` (`:1662`), `workflowFileExceptions`, `TestWorkflowFilePopulationMatchesDisk`; the D-08 removals at `:70-115`, `:792-925`, `:1649-1654`
- `.github/workflows/ci.yml` `:236-252` — drift step removed (D-08)
- `scripts/check-ruleset-drift.sh`, `.github/required-status-checks.txt` — deleted (D-08)

### Project rules
- engram rule `84d1gfpywd` — guards carry a positive assertion; RED against a confirmed-applied, byte-cleanly-reverted mutation
- engram rule `f18zrdsgx5` — never `[ci skip]`
- engram rule `x1cjy9vyhq` — RED evidence = the test commit before the implementation + pasted FAIL transcript (for Python: the `unittest` FAIL output)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `pr_template_policy.py`'s structure (env in, `emit(action, reason)` to `$GITHUB_OUTPUT`, exit code) — the model for `fragment_policy.py`'s `__main__`.
- `gh pr view --json files` returns `{path, additions, deletions, changeType}` (observed on #88: `.changes/header.tpl.md` → `ADDED`), which is what D-04 needs.

### Established Patterns
- Every CI job body is a `task <target>` invocation, and workflow files must be in exactly one of `inScopeWorkflowFiles` / `workflowFileExceptions`. A new workflow file that is in neither fails `TestWorkflowFilePopulationMatchesDisk`.
- Every action is SHA-pinned with a version comment; actionlint runs over all workflows.

### Integration Points
- Research flag: `gh pr view --json files` may cap at 100 files (GraphQL connection). Milestone PRs exceed that, so a truncated list could miss a shipped-code path or a fragment. The researcher confirms the cap and picks a fetch that is complete, or fails closed when it can't prove completeness. The REST `pulls/{n}/files` limit is 3000.
- The Phase 3 fragment is ADDED in #88's diff, so the milestone PR passes the gate once the gate is on `main`.

</code_context>

<specifics>
## Specific Ideas

- The maintainer rejects tests of third-party behaviour: git, GitHub, Actions or changie. See D-01.
- Ceremony gets questioned. A guard stays only if it tests something that can go wrong silently in our own code (the reason D-08 removes two).

</specifics>

<deferred>
## Deferred Ideas

- A `unittest` suite for `scripts/pr_template_policy.py`, which has none today. It now has an obvious home beside `test_fragment_policy.py`. Backlog candidate.

### Reviewed Todos (not folded)
- "Adopt changie for changelog + version and replace release-please…" (score 0.9): the milestone's origin todo. It closes at milestone end, not in this phase.
- "tools/bench/runner pinnedAt() validates a checkout by git rev-parse HEAD alone": bench, unrelated (keyword match only).
- "Reply on GH #85 / #82 / #80 with the reshaped contract": server-mode, unrelated.
- "Pin the Go toolchain so Go 1.27 can build codegraph-go": build, unrelated.

</deferred>

---

*Phase: 04-fragment-required-gate*
*Context gathered: 2026-09-26*
