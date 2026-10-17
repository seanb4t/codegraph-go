# Phase 4: Fragment-Required Gate - Research

**Researched:** 2026-09-26
**Domain:** GitHub Actions policy gate (`pull_request_target`), Python stdlib verdict function + `unittest`, Go shape-test fixture maintenance, git cherry-pick landing sequence
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** Tests exercise **this repository's code only** — the verdict function in `scripts/fragment_policy.py`, fed literal inputs: a changed-file list (path + `changeType`), a PR body, and error states (empty list, API error, unreadable body). No test builds a git repository, calls GitHub, runs a workflow, or runs changie to observe how those tools behave. GATE-04's "real fixtures" are these literal input cases; its "both directions" is the same path list with and without an ADDED fragment. The RED mutation (rule `84d1gfpywd`) mutates **our** path rule and shows **our** tests fail, then is reverted byte-cleanly; the executed-case count is asserted so the suite cannot pass over zero cases.
- **D-02:** `fragment-required.yml` runs on **`pull_request_target`** and checks out the **base branch** (no `ref:`), exactly as `pr-template-format.yml` does, so the policy that judges a PR always comes from `main` and a PR cannot weaken the gate by editing it. Untrusted input (PR body, changed paths) reaches Python only through environment variables, never interpolated into shell. — **Reversibility:** reversible — a trigger edit, but it drives D-05's landing sequence.
- **D-03:** The workflow posts **no PR comment**. Guidance is the check output only: `::error::` lines that name the matched shipped-code paths, how to add a fragment (`task changie -- new …`), and the `<!-- changelog-exempt: <reason> -->` marker. Permissions are therefore `contents: read` and `pull-requests: read` only — no `issues: write`, no `actions/github-script`.
- **D-04:** "Has a fragment" = at least one changed file under `.changes/unreleased/` whose `changeType` is **ADDED** and whose name is not `.gitkeep`. MODIFIED, RENAMED or DELETED files there do not satisfy it — a new code change needs its own entry. Shipped code = `cmd/**` or `internal/**`, excluding `*_test.go` and any path under a `testdata/` directory (GATE-01, verbatim). Exemption marker grammar mirrors `pr_template_policy.py`'s `EXEMPT_MARKER`: `<!-- changelog-exempt: <reason> -->`, case-insensitive, non-empty reason required (GATE-02). A PR touching no shipped code passes without a marker (planning-only PRs, the Phase 5 release PR).
- **D-05:** Because of D-02 the workflow must exist on `main` before it reports on any PR (including #88). The gate reaches `main` through **one PR off `main`** carrying only the gate (workflow, task target, `fragment_policy.py` + tests, shape-test population updates) **and** the D-08 guard removal. Commit types are `ci:` / `test:` / `chore:` so release-please (still live on `main` until Phase 5) opens no release. After it merges, the maintainer adds `fragment-required` to `protect-main` and the edit's timestamp is recorded. No admin-bypass merge is needed: the check is not required until after it exists on `main`.
  - Rejected: merging the milestone-so-far to `main` (the v0.13.0 PR #69 precedent) — it would put Phase 3's `feat(cli)!` on `main` while release-please is live, so release-please would open its own release PR and CHANGELOG edits in collision with changie.
  - Rejected: waiting for the milestone merge — Phase 4 could not close on a working gate.
- **D-06:** Commits are authored **on `gsd/v0.15.0-milestone`** as usual (the planning trail and RED-then-GREEN commit order stay together). A plan task cherry-picks the code-only commits (no `.planning/`) onto a branch off `main` for the D-05 PR; after it merges, `main` is **merged** back into the milestone branch (no rebase).
- **D-07:** The ruleset edit is a **blocking-human checkpoint** (maintainer repository-settings action, the v0.14.0 `GRD-12` precedent for the checkpoint shape). Before it, re-read the live ruleset read-only (it had 8 contexts on 2026-09-26, `strict: false`, admin bypass `RepositoryRole 5`); after it, re-read and record that `fragment-required` is present plus the timestamp.
- **D-08:** Delete both ruleset guards. Removal travels to `main` inside the D-05 PR (otherwise `check-ruleset-drift.sh` would go red on every PR to `main` the moment the ruleset gains a context the `main` copy of the list lacks):
  - `TestRequiredCheckNamesPreserved`, `TestRequiredCheckNamesPreserved_ZeroJobsIsError`, the `TestReadRequiredCheckNames_*` tests, `readRequiredCheckNames`, `requiredStatusChecksPath` (all in `internal/upgrade/taskfile_shape_test.go`), and the comment at `:1649-1654` that cites them.
  - `scripts/check-ruleset-drift.sh`, and the `Ruleset drift check (GRD-12)` step in `ci.yml` (`:236-252`), plus whatever run-body exception entry covers that step.
  - `.github/required-status-checks.txt`.
  - Why: T-10-01-05's premise ("a renamed required check silently un-gates `main`") is wrong — a required context that never reports blocks every PR at "Expected — waiting", which is loud; this milestone's own roadmap hazard relies on that. `check-ruleset-drift.sh` only catches the maintainer editing their own settings without editing a text file (declined once as GRD-07 at v0.13.0). Neither tests the gate's correctness. Maintainer decision, 2026-09-26.
- **D-09:** **GATE-05 is amended** to: "`fragment-required` is a required status check in ruleset `protect-main`, the edit's timestamp recorded." The clauses about the `.github/required-status-checks` fixture, `TestRequiredCheckNamesPreserved` and `scripts/check-ruleset-drift.sh` are dropped. ROADMAP Phase 4 success criterion 5 and its Notes (the "7 contexts" / "re-verify count" text) follow. The planner updates `REQUIREMENTS.md` and the criterion text (values in existing shapes — no new structure, per the planning-artifacts rule).
- **D-10:** New file **`scripts/fragment_policy.py`** — a pure verdict function plus a `__main__` that reads env and writes the result — and **`scripts/test_fragment_policy.py`** (stdlib `unittest`, no new dependency), run by a Taskfile target wired into `ci.yml`'s test job so `TestWorkflowRunBodiesInvokeTask` covers it. `pr_template_policy.py` is not touched (this supersedes the ROADMAP Phase 4 Notes' "one file, two edits" framing; GRD-15 in Phase 5 edits it alone). Python is already on the runner; `scripts/` is outside `cmd/` and `internal/`, so the gate cannot trip its own rule.

### Claude's Discretion

- Event types (mirror `pr-template-format.yml`: `opened, edited, reopened, synchronize, ready_for_review` — `edited` so adding a marker re-evaluates).
- How the changed-file list reaches the script: the script calls `gh` itself, or a prior step writes it using the per-run `$GITHUB_OUTPUT` delimiter (FIX-10 / `task check:workflow-output-delimiter`). Either way GATE-03's three failures (empty list, API error, unreadable body) must each be a distinct message and a fail.
- The Taskfile target names, the job/check `name:` (must read exactly `fragment-required` for the ruleset), and the exact `::error::` wording.
- Where `fragment-required.yml` sits in `inScopeWorkflowFiles` / `inScopeJobs` vs `workflowFileExceptions` — its body is `task`, so in scope is expected.

### Deferred Ideas (OUT OF SCOPE)

- A `unittest` suite for `scripts/pr_template_policy.py`, which has none today. It now has an obvious home beside `test_fragment_policy.py`. Backlog candidate.
- `CONTRIBUTING.md`'s prose on how to add a fragment or exempt — `DOCS-12`, Phase 5.
- Any edit to `scripts/pr_template_policy.py` — GRD-15 edits it alone in Phase 5 (D-10).
- Tests for `pr_template_policy.py` — deferred.
- Any change to the phase-close capability.
</user_constraints>

## Summary

This phase adds one new GitHub Actions workflow (`fragment-required.yml`), one new Python script pair (`scripts/fragment_policy.py` + `scripts/test_fragment_policy.py`), and removes two existing guards (`TestRequiredCheckNamesPreserved` and friends, `scripts/check-ruleset-drift.sh`) from `internal/upgrade/taskfile_shape_test.go` and `ci.yml`. All of this must land on `main` through one PR *before* the ruleset can require the new check (D-02: `pull_request_target` runs the workflow definition from the base ref, so a workflow that doesn't exist on `main` yet cannot report on any PR — including its own landing PR).

The two hardest technical facts this research settles: (1) `gh pr view --json files` truncates at 100 entries with no CLI pagination — the complete-list path is `gh api --paginate repos/{owner}/{repo}/pulls/{n}/files` (REST, `status` field lowercase: `added|removed|modified|renamed|copied|changed|unchanged`, 3000-file hard ceiling), with `pulls/{n}` itself carrying a `changed_files` count you can cross-check to detect truncation and fail closed; and (2) the existing "collect changed files into `$GITHUB_OUTPUT`" heredoc-delimiter pattern used by `pr-template-format.yml`/`require-issue-link.yml` is exercised by a THIRD script, `scripts/check-workflow-output-delimiter.sh`, which hardcodes exactly those two files by name (`specs=".github/workflows/require-issue-link.yml|..." ".github/workflows/pr-template-format.yml|..."`). If `fragment-required.yml` reuses that shell heredoc shape, this out-of-scope script silently stops covering a third instance of the same injection class. The clean design that sidesteps this entirely — and satisfies GATE-04's "job body is a `task` invocation" requirement in one step — is: `fragment_policy.py`'s own `__main__` calls `gh` itself via `subprocess`, so the workflow's only `run:` step is `task check:fragment-required`, with no shell-side `$GITHUB_OUTPUT` heredoc at all.

The D-08 guard removal is a clean, surgical edit: `readRequiredCheckNames`/`requiredStatusChecksPath` and six tests (`TestRequiredCheckNamesPreserved`, `TestRequiredCheckNamesPreserved_ZeroJobsIsError`, four `TestReadRequiredCheckNames_*`) all live in one contiguous region of `internal/upgrade/taskfile_shape_test.go` with no other consumer — confirmed by reading the file, not grepping. `parseWorkflowJobNames`/`mustWorkflowJobNames` are a separate concern used by `TestGateStancesStated` too and MUST be kept. The landing-sequence risk is low: a `git diff main..gsd/v0.15.0-milestone` shows the only code files that diverged since Phase 1 are `.changie.yaml`, `ci.yml` (+9 lines, one isolated step), `Taskfile.yml` (+418 lines, additive), `go.tool-changie.mod` (new), and `taskfile_shape_test.go` (+28 lines, in two regions that do not overlap the D-08 removal regions). `scripts/check-ruleset-drift.sh` and `.github/required-status-checks.txt` are byte-identical to `main`. A cherry-pick of Phase-4-only commits onto a branch off `main` should apply cleanly.

**Primary recommendation:** Write `fragment_policy.py` as a pure `evaluate(files, body, api_error, body_unreadable) -> Verdict` function with a thin `__main__` that calls `gh api --paginate .../pulls/{n}/files` itself (never a shell heredoc), test only the pure function with literal inputs per D-01, wire it into `ci.yml`'s existing `test` job as a new `task check:fragment-policy` step (no fixture change needed there — `test` is already in `inScopeJobs`), and add `fragment-required.yml` + its one job to `inScopeWorkflowFiles`/`inScopeJobs` (never `workflowFileExceptions`) so `TestWorkflowRunBodiesInvokeTask` covers its `task check:fragment-required` body.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| GATE-01 | `fragment-required` fails when changed files touch `cmd/**`/`internal/**` (excluding `*_test.go`/`testdata/`) and add nothing under `.changes/unreleased/`; path rule lives in one place | Path-rule design in "Code Examples"; verbatim rule text confirmed in REQUIREMENTS.md:38 and 04-CONTEXT.md D-04 |
| GATE-02 | `<!-- changelog-exempt: <reason> -->` with non-empty reason passes; grammar mirrors `pr-template-exempt` | `EXEMPT_MARKER` regex read verbatim from `scripts/pr_template_policy.py:83`; mirrored regex given below |
| GATE-03 | Empty changed-file list, API error, unreadable body each fail with a distinct message | `require-issue-link.yml` precedent read in full; function-signature design covers all three as literal inputs (D-01) |
| GATE-04 | Demonstrated both directions against real fixtures, executed-case count asserted, job body is `task <target>` | `inScopeJobs`/`TestWorkflowRunBodiesInvokeTask` mechanics read from source; mutation-log template from `03-MUTATION-LOG.md` |
| GATE-05 (amended, D-09) | `fragment-required` required in `protect-main`, edit timestamp recorded | Exact ROADMAP.md/REQUIREMENTS.md line ranges identified for the amendment edit |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Fetch changed-file list + PR body | CI / GitHub Actions (workflow step, via `gh`) | — | Only the Actions runtime has the PR context and a `GITHUB_TOKEN`; this is I/O, not policy |
| Verdict (shipped-code-without-fragment) | Repo-owned script (`scripts/fragment_policy.py`) | — | D-01: the ONLY thing this phase's tests exercise; must be a pure function isolable from `gh`/git/Actions |
| Path-rule definition (`cmd/**`/`internal/**` minus tests/testdata) | Same script, one shared constant/function | Shape-test fixtures (`inScopeJobs`, `inScopeWorkflowFiles`) reference the workflow, not the rule itself | GATE-01: "the path rule lives in one place shared with the gate's tests" — one Python module, imported by both `fragment_policy.py` and `test_fragment_policy.py` |
| CI job wiring / single-definition enforcement | Go shape tests (`internal/upgrade/taskfile_shape_test.go`) | Taskfile.yml (task body) | Existing project convention: every CI `run:` body must be `task <target>`, enforced by `TestWorkflowRunBodiesInvokeTask` |
| Branch-protection enforcement | GitHub ruleset (`protect-main`, ID 20157557) | — | Out-of-repo state; D-07's blocking-human checkpoint, not code |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Python 3 stdlib (`unittest`, `re`, `json`, `subprocess`, `os`, `sys`) | whatever `python3` ubuntu-latest/Namespace runners ship (repo already invokes bare `python3` with no version pin in `pr-template-format.yml:67`) | Verdict function + tests | D-10 explicitly: "no new dependency"; this repo already runs `python3 scripts/pr_template_policy.py` in CI unpinned — same convention, mirrored, not invented |
| `gh` CLI | Whatever GitHub-hosted/Namespace runner image ships (already used, unpinned, by `pr-template-format.yml`/`require-issue-link.yml`) | Fetch PR file list | Already the project's established mechanism (`GH_TOKEN: ${{ github.token }}`); GATE work must not introduce a second HTTP client |

No new external packages (npm/pip/cargo) are introduced by this phase.

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `actions/checkout@df4cb1c0...` (v6.0.3, already pinned in-repo) | pinned | Base-branch checkout for `pull_request_target` (no `ref:`) | Every step of `fragment-required.yml`; copy the exact pin `pr-template-format.yml:33` already uses |
| `./.github/actions/install-task` (in-repo composite action) | n/a | Build `task` from `go.tool.mod` | Needed only if the job's own run body is `task <target>`-shaped (D-10 discretion resolved below to: yes, always) |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Script calls `gh` itself (subprocess) | Prior shell step collects files into `$GITHUB_OUTPUT` via heredoc-delimiter pattern | The heredoc pattern is exercised by `scripts/check-workflow-output-delimiter.sh`, which **hardcodes exactly two workflow files by name** (`require-issue-link.yml`, `pr-template-format.yml` — verified by reading the script's `specs=` variable). A third file using the same pattern is invisible to that guard unless it too is added there — an out-of-scope edit CONTEXT.md's scope list does not mention. Calling `gh` from Python avoids the injection class (and the guard-scope question) entirely, and keeps the job's only `run:` step a single `task <target>` line, cleanly satisfying `TestWorkflowRunBodiesInvokeTask`'s single-definition property with zero exception entries. |
| REST `gh api --paginate .../pulls/{n}/files` | `gh pr view --json files` | The GraphQL-backed `--json files` connection is capped at 100 entries with **no CLI-level pagination support** ([cli/cli discussion #5359](https://github.com/cli/cli/discussions/5359), [#5373](https://github.com/cli/cli/discussions/5373)) — confirmed by two independent maintainer/community threads. A milestone-scale PR (this milestone's own PRs already exceed 100 files per multiple SUMMARY.md file counts) would silently see a truncated list under `--json files`, which is exactly the "miss a shipped-code path" failure GATE-01 must not have. |

## Package Legitimacy Audit

Not applicable — this phase installs no new npm/pip/cargo package. `python3` and `gh` are both already present and already invoked unpinned by two pre-existing workflows in this repository (`pr-template-format.yml`, `require-issue-link.yml`).

## Architecture Patterns

### System Architecture Diagram

```
PR opened/edited/reopened/synchronize/ready_for_review
        │
        ▼
GitHub Actions triggers fragment-required.yml (pull_request_target)
        │  workflow definition read from BASE ref (main) — never the PR head
        ▼
Step 1: actions/checkout (no ref: → checks out base branch, D-02)
        ▼
Step 2: ./.github/actions/install-task (builds `task` from go.tool.mod)
        ▼
Step 3: `task check:fragment-required`
        │  (single run: body — the ONLY step whose body is not a `uses:`)
        ▼
   scripts/fragment_policy.py  __main__
        │
        ├─► subprocess: gh api --paginate repos/{o}/{r}/pulls/{n}/files
        │       → [{filename, status}, ...]   (status: added|removed|modified|
        │                                       renamed|copied|changed|unchanged)
        │   cross-check len(files) against pulls/{n}'s own `changed_files` count
        │       → mismatch = truncation = FAIL CLOSED (api_error path)
        │
        ├─► os.environ["PR_BODY"]  (env-only, never interpolated into shell)
        │
        ▼
   evaluate(files, body, api_error, body_unreadable) -> Verdict
        │   ┌─ shipped_code_touched = any(f matches cmd/**|internal/** rule)
        │   ├─ has_fragment = any(f.path startswith .changes/unreleased/
        │   │                     and f.status == "added" and not .gitkeep)
        │   ├─ has_exemption = CHANGELOG_EXEMPT_MARKER.search(body)
        │   └─ verdict = pass unless (shipped_code_touched and not
        │                 has_fragment and not has_exemption), or one of the
        │                 three fail-closed states (GATE-03)
        ▼
   emit(action, reason) → $GITHUB_OUTPUT + stderr
        ▼
   exit 0 (pass) / exit 1 (fail) — sets the check's own status
        │
        ▼
GitHub ruleset `protect-main` (after D-07's checkpoint) blocks merge on failure
```

### Recommended Project Structure
```
scripts/
├── fragment_policy.py        # verdict function + __main__ (env in, gh subprocess, $GITHUB_OUTPUT out)
├── test_fragment_policy.py   # stdlib unittest, literal FileChange inputs only (D-01)
├── pr_template_policy.py     # UNTOUCHED this phase (D-10; GRD-15 edits it alone in Phase 5)
└── check-ruleset-drift.sh    # DELETED (D-08)
.github/
├── workflows/
│   └── fragment-required.yml # NEW — pull_request_target, base-branch checkout, task-only body
└── required-status-checks.txt # DELETED (D-08)
```

### Pattern 1: `pull_request_target` + base-branch checkout + env-only untrusted input
**What:** Trigger on `pull_request_target` (not `pull_request`) so the workflow always runs with the *base* ref's permissions and, critically, the base ref's own copy of the workflow file and any script it invokes — `actions/checkout` with no `ref:` input checks out `main`, not the PR head.
**When to use:** Any policy check whose own correctness must not be alterable by the PR under test — exactly D-02's requirement.
**Example:**
```yaml
# Source: .github/workflows/pr-template-format.yml:1-14 (read verbatim this session)
on:
  pull_request_target:
    types: [opened, edited, reopened, synchronize, ready_for_review]

concurrency:
  group: ${{ github.workflow }}-${{ github.event.pull_request.number || github.ref }}
  cancel-in-progress: true

permissions:
  contents: read
  pull-requests: read
```
`fragment-required.yml` needs no `issues: write` (D-03: no PR comment is posted — check output only), so its `permissions:` block is exactly `contents: read` + `pull-requests: read`.

### Pattern 2: Verdict function isolated from I/O (testability boundary, D-01)
**What:** All `gh`/env access lives in `__main__`; the decision logic is a pure function taking already-parsed data plus explicit failure flags.
**When to use:** Whenever a policy needs both a production I/O path and a test suite that "never builds a git repository, calls GitHub, runs a workflow, or runs changie" (D-01, verbatim).
**Example (skeleton, not verbatim source — to be written by the plan):**
```python
# Modeled on scripts/pr_template_policy.py's emit()/env-in shape (read verbatim
# this session: EXEMPT_MARKER at line 83, emit() at lines 87-94).
from dataclasses import dataclass

SHIPPED_PREFIXES = ("cmd/", "internal/")
FRAGMENT_DIR_PREFIX = ".changes/unreleased/"
CHANGELOG_EXEMPT_MARKER = re.compile(
    r"<!--\s*changelog-exempt:\s*(?P<reason>[^->][^>]*?)\s*-->", re.I
)

@dataclass(frozen=True)
class FileChange:
    path: str
    status: str  # REST value: added|removed|modified|renamed|copied|changed|unchanged

def is_shipped_code(path: str) -> bool:
    if not path.endswith(".go"):
        pass  # GATE-01's rule is path-shaped, not extension-shaped; do not add an
              # extension filter not present in the verbatim GATE-01/D-04 text.
    if path.endswith("_test.go"):
        return False
    segments = path.split("/")
    if "testdata" in segments:
        return False
    return segments[0] in ("cmd", "internal")

def has_fragment(files: list[FileChange]) -> bool:
    return any(
        f.path.startswith(FRAGMENT_DIR_PREFIX)
        and f.status == "added"
        and not f.path.endswith(".gitkeep")
        for f in files
    )

def evaluate(files, body, api_error=False, body_unreadable=False):
    if api_error:
        return Verdict(False, "could not determine the PR's changed files (API error)")
    if body_unreadable:
        return Verdict(False, "could not read the PR body")
    if not files:
        return Verdict(False, "the changed-file list is empty")
    if CHANGELOG_EXEMPT_MARKER.search(body or ""):
        reason = CHANGELOG_EXEMPT_MARKER.search(body).group("reason").strip()
        if reason:
            return Verdict(True, f"explicitly exempt: {reason}")
    if not any(is_shipped_code(f.path) for f in files):
        return Verdict(True, "touches no shipped code (cmd/**, internal/**)")
    if has_fragment(files):
        return Verdict(True, "shipped code change carries a fragment")
    return Verdict(False, "shipped code changed with no .changes/unreleased/ fragment and no exemption")
```
This is a skeleton for the planner to refine — every literal in it (`SHIPPED_PREFIXES`, `FRAGMENT_DIR_PREFIX`, the regex) is `[ASSUMED]` design, not verified output; only the `EXEMPT_MARKER` shape it mirrors (`pr_template_policy.py:83`) and the GATE-01/D-04 path rule text are `[VERIFIED]`.

### Anti-Patterns to Avoid
- **Reusing the `$GITHUB_OUTPUT` heredoc-delimiter pattern for a third file:** it works, but silently escapes `check-workflow-output-delimiter.sh`'s coverage (hardcoded to two files) unless that out-of-scope script is also extended. Calling `gh` from Python avoids the question.
- **Adding an extension filter (e.g. `.go`) to the shipped-code rule:** GATE-01's verbatim text is "`cmd/**` or `internal/**`, excluding `*_test.go` and `testdata/`" — no extension filter. `cmd/**`/`internal/**` may contain non-`.go` files (e.g. embedded assets) that still count as shipped code under the stated rule.
- **A `runBodyExceptions` entry to dodge the single-`task`-call rule:** unlike `ci.yml`'s "Ruleset drift check" (a live network fetch deliberately kept outside Taskfile so it can't be invoked offline and read as PASS), `fragment_policy.py`'s test suite is pure and offline-safe — it belongs in a Taskfile target, not an exception.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Exemption-marker grammar | A new regex from scratch | Mirror `pr_template_policy.py`'s `EXEMPT_MARKER` (`re.compile(r"<!--\s*pr-template-exempt:\s*(?P<reason>[^->][^>]*?)\s*-->", re.I)`, read verbatim at line 83) with `changelog-exempt` substituted | D-04 explicitly requires the grammar to mirror this; reusing the exact character-class avoids a subtly different edge-case (e.g. what "non-empty reason" means for a reason starting with `-`) |
| Complete changed-file list under `pull_request_target` | A hand-rolled GraphQL pagination loop | `gh api --paginate repos/{owner}/{repo}/pulls/{n}/files` | `gh`'s `--paginate` flag already follows the REST `Link` header across pages; no custom cursor logic needed |
| Fail-closed empty/error handling | A bespoke design | Mirror `require-issue-link.yml`'s shape: named, distinct `::error::`/failure messages for each failure class, never a silent skip | GATE-03 cites this exact precedent; the project has already solved this problem once |

**Key insight:** Every piece of this phase has a byte-for-byte precedent already in this repository (`pr_template_policy.py`'s marker grammar and `emit()` shape, `pr-template-format.yml`'s trigger/checkout shape, `require-issue-link.yml`'s fail-closed empty-list handling). The research risk is not "what pattern to use" — it is "which existing guard's *scope* silently does not extend to the new file" (the `check-workflow-output-delimiter.sh` and `inScopeJobs`/`inScopeWorkflowFiles` findings below).

## Runtime State Inventory

This is not a rename/refactor/migration phase in the traditional sense, but D-08 deletes two files and edits a live, off-repo GitHub ruleset (D-07), so this inventory is included for completeness.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | None — no database/datastore involved | none |
| Live service config | **GitHub ruleset `protect-main` (ID 20157557)** — its required-status-check list lives in GitHub's own settings, NOT in git. `.github/required-status-checks.txt` is this repo's own committed *mirror* of that live list (D-07), read by the guard being deleted. After D-08, no committed mirror exists at all — the ruleset becomes the sole source of truth, re-read live (read-only) before D-07's edit and again after, per D-07's checkpoint protocol. | Repository-settings action (D-07, blocking-human checkpoint); no code change beyond deleting the mirror file |
| OS-registered state | None | none |
| Secrets/env vars | None new. `GITHUB_TOKEN`/`GH_TOKEN` already used identically by `pr-template-format.yml`/`require-issue-link.yml`; `fragment-required.yml` reuses `${{ github.token }}` the same way, no new secret name | none |
| Build artifacts | None — no installed package, no compiled binary changes here | none |

**Canonical question answered:** after every file in the repo is updated, the only runtime state left holding the "old" shape is the GitHub ruleset itself — which D-07 already treats as a live, out-of-git system requiring a read-verify-edit-verify checkpoint, not a code change.

## Common Pitfalls

### Pitfall 1: `gh pr view --json files` silently truncates at 100 files
**What goes wrong:** A PR with more than 100 changed files (this milestone's own PRs already exceed that per SUMMARY.md file counts, e.g. 45 files, 44 files) gets a partial list. If a shipped-code file happens to sort past position 100, the gate never sees it and passes a PR that should have failed.
**Why it happens:** `--json files` is backed by GraphQL's `files` connection, whose default/max page size in this call shape is 100, and `gh`'s `--json` flag has no built-in pagination for this field.
**How to avoid:** Use `gh api --paginate repos/{owner}/{repo}/pulls/{n}/files` (REST), which `gh api` paginates automatically via the `Link` header. Cross-check the total count returned against `pulls/{n}`'s own `changed_files` field; a mismatch means truncation (approaching the REST endpoint's stated 3000-file ceiling) and must be treated as the GATE-03 "API error" fail-closed case, never silently evaluated on a partial list.
**Warning signs:** A gate that "worked in testing" (small PRs) but passes a real milestone-scale PR it should have failed.

### Pitfall 2: Reusing the heredoc-delimiter pattern extends an out-of-scope guard's blind spot
**What goes wrong:** `scripts/check-workflow-output-delimiter.sh` proves the per-run `$GITHUB_OUTPUT` delimiter defeats a fork-authored-filename injection (GH #15) — but only for the two files hardcoded in its `specs=` variable (`require-issue-link.yml`, `pr-template-format.yml`, confirmed by reading the script). A third workflow reusing the identical shell heredoc pattern is invisible to this guard until someone adds a third `specs=` line — an edit outside this phase's declared scope (04-CONTEXT.md's `<domain>` block does not list this file).
**Why it happens:** The guard extracts named steps from named files by literal path; a new file using the same pattern doesn't register itself.
**How to avoid:** Have `fragment_policy.py`'s `__main__` call `gh` directly via `subprocess`, so there is no shell-side `$GITHUB_OUTPUT` heredoc at all in the new workflow, and this guard's scope question never arises.
**Warning signs:** A plan step that copies the "Collect changed files" step verbatim from `require-issue-link.yml` into `fragment-required.yml`.

### Pitfall 3: Adding a workflow file to `inScopeWorkflowFiles` without also registering its job in `inScopeJobs`
**What goes wrong:** `TestInScopeJobsPopulationMatchesDisk` (`internal/upgrade/taskfile_shape_test.go:1864-1923`) iterates every job declared in every file named in `inScopeWorkflowFiles` and fails if that job isn't also in `inScopeJobs` (or `usesOnlyJobExceptions`). Registering the new file without its job entry fails this test on the first run.
**Why it happens:** `inScopeWorkflowFiles` (file-level) and `inScopeJobs` (job-level) are two independent, hand-maintained fixtures; nothing generates one from the other.
**How to avoid:** Add both in the same commit: `fragment-required.yml` to `inScopeWorkflowFiles` (`:1663`), and `{Workflow: "fragment-required.yml", JobID: "<job-id>"}` to `inScopeJobs` (`:188-201`).
**Warning signs:** `go test ./internal/upgrade/... -run TestInScopeJobsPopulationMatchesDisk` failing with "job(s) on disk... not present in inScopeJobs".

### Pitfall 4: `TestWorkflowRunBodiesInvokeTask`'s single-line regex forbids anything but `task <target>`
**What goes wrong:** `taskCallLineRe = regexp.MustCompile(`^task\s+[A-Za-z0-9:_-]+$`)` (read verbatim, `:254`) requires the run body, after stripping comments/blanks, to be *exactly* one `task <target>` line. Any additional shell (a `gh` call, an `if`, a second command) in the same step fails this test unless a `runBodyException` entry is added.
**Why it happens:** This is the deliberate single-definition property (D-01/D-02 from an earlier phase) — every CI command must have exactly one definition, in `Taskfile.yml`.
**How to avoid:** Push all logic (the `gh` call included) into `scripts/fragment_policy.py`, invoked by a Taskfile target whose `cmds:` do the multi-line work; the workflow step itself stays a bare `task check:fragment-required`.
**Warning signs:** A workflow step with a multi-line `run: |` block in a job you intend to add to `inScopeJobs`.

### Pitfall 5: Confusing REST `status` (lowercase) with GraphQL `changeType` (uppercase)
**What goes wrong:** `pr_template_policy.py`'s own doc-comment example and 04-CONTEXT.md's code_context both cite `changeType: ADDED` (GraphQL, uppercase) — but the REST endpoint this research recommends for completeness returns a lowercase `status: "added"` field with a different key name entirely. Code that checks `f["changeType"] == "ADDED"` against REST JSON will silently never match (KeyError or None), and "has a fragment" would always evaluate false.
**Why it happens:** GraphQL and REST model the same concept with different field names and casing; `gh pr view --json files` uses the former, `gh api .../pulls/{n}/files` the latter.
**How to avoid:** Since this research recommends the REST path for completeness (Pitfall 1), standardize on REST's `status` field and its lowercase enum (`added|removed|modified|renamed|copied|changed|unchanged`) throughout `fragment_policy.py` and its tests. Do not carry over the `changeType`/`ADDED` vocabulary from 04-CONTEXT.md's code_context note, which was observed against `--json files` (GraphQL), not the paginated REST call this phase should use.
**Warning signs:** `has_fragment()` always returning `False` even when a fragment file is genuinely present in the PR.

## Code Examples

### Workflow skeleton (`fragment-required.yml`)
```yaml
# Source pattern: .github/workflows/pr-template-format.yml (read verbatim this
# session) — same pull_request_target + base-branch-checkout + env-only-input
# shape. Diverges from that precedent by installing `task` (D-10: job body
# must be `task <target>`) and posting no PR comment (D-03).
name: Fragment Required

on:
  pull_request_target:
    types: [opened, edited, reopened, synchronize, ready_for_review]

concurrency:
  group: ${{ github.workflow }}-${{ github.event.pull_request.number || github.ref }}
  cancel-in-progress: true

permissions:
  contents: read
  pull-requests: read

jobs:
  fragment-required:              # name: MUST read exactly "fragment-required" (GATE-05 ruleset match)
    name: fragment-required
    runs-on: ubuntu-latest
    timeout-minutes: 3
    steps:
      - name: Check out policy from the base branch
        uses: actions/checkout@df4cb1c069e1874edd31b4311f1884172cec0e10 # v6.0.3

      - name: Install Task
        uses: ./.github/actions/install-task

      - name: Evaluate fragment policy
        env:
          GH_TOKEN: ${{ github.token }}
          GITHUB_REPOSITORY: ${{ github.repository }}
          PR_NUMBER: ${{ github.event.pull_request.number }}
          PR_BODY: ${{ github.event.pull_request.body }}
        run: task check:fragment-required
```

### Taskfile targets (skeleton)
```yaml
# Pattern source: Taskfile.yml:551-578 (check:changie, read verbatim) for the
# desc:/preconditions:/cmds: shape.
  check:fragment-required:
    desc: >-
      GATE-01..03's production entry point: runs scripts/fragment_policy.py's
      __main__ against the real PR context (GH_TOKEN/PR_NUMBER/PR_BODY env),
      which fetches the complete changed-file list via `gh api --paginate`
      and writes the verdict to $GITHUB_OUTPUT. Never invoked offline against
      a real PR — CI-only by nature of needing GH_TOKEN/PR_NUMBER.
    preconditions:
      - sh: command -v python3
        msg: "python3 not found — check:fragment-required runs scripts/fragment_policy.py."
      - sh: command -v gh
        msg: "gh not found — fragment_policy.py's __main__ calls gh api to fetch the PR's changed files."
    cmds:
      - python3 scripts/fragment_policy.py

  check:fragment-policy:
    desc: >-
      GATE-04's offline unit-test guard: runs scripts/test_fragment_policy.py's
      stdlib unittest suite against literal FileChange inputs only (D-01) —
      never gh, git, or a real PR. Asserts a positive executed-case count
      (rule 84d1gfpywd) from unittest's own "Ran N test(s)" summary line.
    preconditions:
      - sh: command -v python3
        msg: "python3 not found — check:fragment-policy runs scripts/test_fragment_policy.py."
    cmds:
      - |
        set -euo pipefail
        out=$(python3 -m unittest discover -s scripts -p 'test_fragment_policy.py' -v 2>&1) || { echo "$out"; exit 1; }
        echo "$out"
        ran=$(printf '%s\n' "$out" | grep -Eo 'Ran [0-9]+ test' | grep -Eo '[0-9]+' || echo 0)
        if [ "${ran}" -eq 0 ]; then
          echo "::error::check:fragment-policy: unittest reported 0 executed cases — a vacuous pass (rule 84d1gfpywd)"
          exit 1
        fi
        echo "check:fragment-policy: executed ${ran} case(s)"
```
Wire `check:fragment-policy` into `ci.yml`'s existing `test` job as a new step (that job is already in `inScopeJobs`, `:189` — no fixture change needed for this step alone). `check:fragment-required` becomes the new `fragment-required.yml` job's sole `run:` step and requires new `inScopeJobs`/`inScopeWorkflowFiles` entries (Pitfall 3).

### Complete-list fetch with truncation detection
```bash
# Design pattern — to be implemented inside fragment_policy.py's __main__ via
# subprocess, not as a separate shell step (Pitfall 2/4).
gh api "repos/${GITHUB_REPOSITORY}/pulls/${PR_NUMBER}" --jq '.changed_files'
gh api --paginate "repos/${GITHUB_REPOSITORY}/pulls/${PR_NUMBER}/files" \
  --jq '.[] | {path: .filename, status: .status}'
# If the paginated file count != the PR's own changed_files count, treat as
# an API error (GATE-03) rather than evaluating a partial list.
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| `.github/required-status-checks.txt` mirrors the live ruleset in git, checked both ways by `TestRequiredCheckNamesPreserved` + `check-ruleset-drift.sh` | No committed mirror; the ruleset is the sole source of truth, verified read-only at D-07's checkpoint | This phase (D-08, maintainer decision 2026-09-26) | Removes two guards that "tested how GitHub Actions/rulesets work" rather than this repo's own code (maintainer's explicit framing) — the hard constraint this whole phase operates under |

**Deprecated/outdated:** `scripts/check-ruleset-drift.sh` and `.github/required-status-checks.txt` — both deleted by D-08; do not reference them in any new code or docs written by this phase's plans.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The Python runtime on `ubuntu-latest`/Namespace `namespace-profile-linux-amd64-*` runners is recent enough (3.10+) for any stdlib feature the plan uses (`dataclasses`, `unittest.mock` if needed, f-strings) — inferred from the repo already invoking bare, unpinned `python3` in `pr-template-format.yml:67` with no version assertion anywhere in this repo | Standard Stack, Architecture Patterns | Low — stdlib `unittest`/`re`/`dataclasses` have been stable since 3.7; a plan targeting only these has no real exposure |
| A2 | "Unreadable body" (GATE-03's third fail-closed case) is best modeled as `os.environ.get("PR_BODY")` returning `None` (key genuinely absent from the environment), distinct from an empty string — this exact state may never occur in production under the `pull_request_target` trigger (GitHub Actions renders a `null` PR body as an empty string in `${{ }}` interpolation, not an absent env var), making this branch primarily a defensive/test-only path | Code Examples, Pattern 2 | Low-medium — if the planner locks a different semantics for "unreadable" (e.g. a decode error, or genuinely never distinguishing it from "empty"), GATE-03's three-messages requirement still needs three code paths; worth a locked decision at plan time rather than left implicit |
| A3 | The complete list of files that reference the D-08-removed guards outside `.planning/` is exactly: `internal/upgrade/taskfile_shape_test.go` (multiple regions, cited with line numbers below), `ci.yml:236-252` + its `runBodyExceptions` entry, `scripts/check-ruleset-drift.sh` (whole file), `.github/required-status-checks.txt` (whole file) — CONTRIBUTING.md/docs mentions were not found by this session's reading (not exhaustively grepped across every doc file) | Research Question 2 findings | Low — a missed prose mention in a doc file is a cosmetic follow-up, not a build break; the code/test/CI references are all confirmed by direct reads |

## Open Questions (RESOLVED)

1. **Exact semantics of "unreadable body" (GATE-03)** — RESOLVED: "unreadable" means `PR_BODY` is absent from the step environment, which `main()` passes to the verdict function as `None`. A present empty string is readable: it carries no exemption and is never reported as `body-unreadable`. The state fails closed with code `body-unreadable` and its own message; when several fail-closed states hold at once, precedence is `api-error`, then `empty-file-list`, then `body-unreadable`. Per CONTEXT D-01 the branch is tested as a literal input to the verdict function (`test_unreadable_body_fails_closed`, `test_empty_body_is_readable`, `test_main_fails_closed_without_pr_body`), not by observing GitHub Actions. Per D-10 it lives in `scripts/fragment_policy.py`. Locked in Plan 04-01 Task 2.
   - What we know: `require-issue-link.yml`/`pr_template_policy.py` never distinguish "body absent" from "body empty" — both collapse to `os.environ.get("PR_BODY") or ""`. GATE-03 asks for a *distinct* message for "unreadable body" as a third case alongside "empty changed-file list" and "API error".
   - What's unclear: what production condition actually produces an "unreadable" body distinct from "empty" — GitHub Actions' env interpolation of a `null` body already yields `""`, not an absent variable.
   - Recommendation: the plan should lock this as a literal input case for the verdict function's test suite (D-01 permits testing error states as literal inputs) even if production code can realistically only synthesize it defensively (e.g., a `try/except` around body decoding) — document the exact trigger condition in the plan rather than leaving it implicit.

2. **Exact job/step names inside `fragment-required.yml`** — RESOLVED: two Taskfile targets. `check:fragment-required` is the production path: it calls `gh` and is the single run body of `fragment-required.yml`. `check:fragment-policy` is the offline path: it runs the stdlib `unittest` suite with an executed-case-count guard and is wired into `ci.yml`'s `test` job per CONTEXT D-10, so `TestWorkflowRunBodiesInvokeTask` covers it. The workflow's job YAML key and its `name:` are both `fragment-required`, matching the ruleset context (GATE-05). Target names are Claude's Discretion in CONTEXT. Locked in Plan 04-01 Tasks 1 and 3.
   - What we know: the job's `name:` must read exactly `fragment-required` for the ruleset match (GATE-05); Claude's Discretion (04-CONTEXT.md) leaves the Taskfile target names and `::error::` wording open.
   - What's unclear: whether the plan wants `check:fragment-required` (production) and `check:fragment-policy` (test) as two distinct Taskfile targets (this research's recommendation) or a single target with a mode flag.
   - Recommendation: two targets, per the Code Examples section — keeps the CI-only (`gh`-calling) path and the offline-safe (pure-function) path unambiguously separate, matching this repo's own `check:changie`-vs-nothing-else precedent of one target per concern.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `python3` | `scripts/fragment_policy.py`, `scripts/test_fragment_policy.py` | ✓ (already invoked unpinned by `pr-template-format.yml:67`) | Runner-provided, unpinned in-repo | — |
| `gh` CLI | Fetching PR files/body via REST | ✓ (already invoked unpinned by `pr-template-format.yml`/`require-issue-link.yml`) | Runner-provided, unpinned in-repo | — |
| `task` | CI job body (`task check:fragment-required`) | ✓ (built by `./.github/actions/install-task` from `go.tool.mod`, already used throughout `ci.yml`) | Pinned via `go.tool.mod` | — |

No missing dependencies. This phase introduces no new environment requirement beyond what `ci.yml` and the two precedent workflows already assume.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Python stdlib `unittest` (new to this repo's Python surface — `pr_template_policy.py` has no test suite today; see 04-CONTEXT.md Deferred Ideas) |
| Config file | none — stdlib `unittest` needs none |
| Quick run command | `python3 -m unittest discover -s scripts -p 'test_fragment_policy.py' -v` |
| Full suite command | same (single test file this phase; `pr_template_policy.py` tests are deferred, per D-10/Deferred Ideas) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| GATE-01 | Shipped-code path touched + no fragment → fail; identical shape + fragment → pass | unit | `python3 -m unittest scripts.test_fragment_policy -v` | ❌ Wave 0 |
| GATE-02 | `<!-- changelog-exempt: reason -->` with non-empty reason passes; empty reason does not exempt | unit | same | ❌ Wave 0 |
| GATE-03 | Empty file list / api_error=True / body_unreadable=True each produce a distinct failing message | unit | same | ❌ Wave 0 |
| GATE-04 | Both directions demonstrated, executed-case count asserted, RED mutation of the path rule recorded | unit + mutation log | same, plus a scratch-worktree mutation of `is_shipped_code`'s prefixes/testdata check, RED then reverted (see Pitfall/Question 7 template below) | ❌ Wave 0 |
| GATE-05 | `fragment-required` name matches exactly; workflow shape passes existing Go guards | Go shape test (existing files, extended) | `go test ./internal/upgrade/... -run 'TestWorkflowRunBodiesInvokeTask|TestWorkflowFilePopulationMatchesDisk|TestInScopeJobsPopulationMatchesDisk'` | ✓ (tests exist; fixtures need new entries) |

### Sampling Rate
- **Per task commit:** `python3 -m unittest discover -s scripts -p 'test_fragment_policy.py' -v` and `go test ./internal/upgrade/...`
- **Per wave merge:** `task check:fragment-policy` (wired into `ci.yml`'s `test` job) plus full `go test ./...`
- **Phase gate:** Full suite green before `/gsd:verify-work`, plus the GATE-04 mutation-log evidence recorded per the `03-MUTATION-LOG.md` convention below.

### Wave 0 Gaps
- [ ] `scripts/fragment_policy.py` — the verdict function + `__main__` (does not exist yet)
- [ ] `scripts/test_fragment_policy.py` — the stdlib `unittest` suite (does not exist yet)
- [ ] `.github/workflows/fragment-required.yml` — the new workflow (does not exist yet)
- [ ] `Taskfile.yml` entries `check:fragment-required` / `check:fragment-policy` (do not exist yet)
- [ ] `internal/upgrade/taskfile_shape_test.go` fixture entries: `inScopeWorkflowFiles` (`:1663`), `inScopeJobs` (`:188-201`) — new entries for `fragment-required.yml`
- [ ] `internal/upgrade/taskfile_shape_test.go` deletions (D-08): `readRequiredCheckNames`/`requiredStatusChecksPath` (`:70-116`), `TestRequiredCheckNamesPreserved` + 5 sibling tests (`:792-925`), the `:1649-1654` comment reference

### GATE-04 mutation-log template (following `03-MUTATION-LOG.md`'s convention)
Read `.planning/phases/03-rename-stub-removal/03-MUTATION-LOG.md` in full this session — its shape (pre-mutation `git diff --quiet` gate, mutation applied, confirmed applied via `git diff --numstat`/`rg`, observed RED transcript, revert, byte-clean proof via `git diff --quiet`) is directly reusable for GATE-04, substituting `go test` for `python3 -m unittest`:

1. **Setup:** disposable scratch git worktree (`git worktree add --detach`), never the main tree.
2. **Pre-mutation gate:** `git -C "$S" diff --quiet -- scripts/fragment_policy.py` (clean).
3. **Mutation applied:** e.g. change `segments[0] in ("cmd", "internal")` to `segments[0] in ("cmd",)` (silently stop treating `internal/**` as shipped code) — a real, single, reasoned mutation of the path rule GATE-04 names.
4. **Confirmed applied:** `git -C "$S" diff --numstat -- scripts/fragment_policy.py` (exactly one line changed).
5. **Observed result:** paste the `unittest` FAIL transcript (rule x1cjy9vyhq: RED evidence = pasted FAIL output, not an assertion) — a literal `FileChange(path="internal/foo.go", status="modified")`-only case that should fail now (silently) passes.
6. **Revert:** `git -C "$S" checkout -- scripts/fragment_policy.py`.
7. **Byte-clean proof:** `git -C "$S" diff --quiet` (clean).
8. **Both directions (GATE-04's "both directions"):** the identical fixture (a shipped-code file, no fragment) with vs. without an `added` `.changes/unreleased/*.yaml` entry — both directions of the SAME path list.

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | n/a — no auth surface added |
| V3 Session Management | no | n/a |
| V4 Access Control | no | n/a — GitHub's own ruleset/permissions model governs merge access |
| V5 Input Validation | yes | Untrusted input (PR body, changed-file paths from a fork PR under `pull_request_target`) reaches Python only via environment variables, never shell-interpolated (D-02) — exact pattern already used by `pr_template_policy.py` |
| V6 Cryptography | no | n/a |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Fork PR injects a file path/body content that breaks out of a shell context under `pull_request_target` (GH #15 class) | Tampering | Env-var-only input (never interpolated into `run:` shell text); avoided entirely for the changed-file list by having Python call `gh` via `subprocess` with argument-list invocation (no shell string concatenation) rather than a shell heredoc |
| A fork PR edits the gate's own script to weaken it | Tampering / Elevation of Privilege | `pull_request_target` + base-branch checkout (D-02) — the script that runs is always `main`'s copy, never the PR's |
| A required check that never reports (e.g. workflow doesn't exist on `main` yet) silently un-gates `main` | Repudiation / availability | D-08's own rationale: GitHub blocks merge on a required context stuck at "Expected — waiting" — loud, not silent; this is why the two removed guards were judged to test GitHub's own behavior rather than a real gap in this repo's code |

## Sources

### Primary (HIGH confidence)
- `.github/workflows/pr-template-format.yml` (read verbatim, this repo) — trigger/checkout/env-input shape
- `.github/workflows/require-issue-link.yml` (read verbatim, this repo) — fail-closed empty-list handling, exemption marker precedent
- `scripts/pr_template_policy.py` (read verbatim, this repo) — `EXEMPT_MARKER` (line 83), `emit()` (lines 87-94)
- `scripts/check-workflow-output-delimiter.sh` (read verbatim, this repo) — hardcoded two-file `specs=` variable
- `internal/upgrade/taskfile_shape_test.go` (read verbatim, this repo, multiple regions cited by line) — `readRequiredCheckNames`/`requiredStatusChecksPath` (:70-116), `TestRequiredCheckNamesPreserved` family (:792-925), `inScopeJobs` (:188-201), `runBodyExceptions` (:236-249), `taskCallLineRe` (:254), `inScopeWorkflowFiles`/`workflowFileExceptions` (:1643-1731), `TestWorkflowFilePopulationMatchesDisk` (:1761-1812), `TestInScopeJobsPopulationMatchesDisk` (:1864-1923), `TestGateStancesStated` (:977-1026)
- `.github/workflows/ci.yml` (read verbatim, this repo) — `test` job structure, "Ruleset drift check (GRD-12)" step (:236-252), install-task usage
- `.github/actions/install-task/action.yml` (read verbatim) — composite action, no `ref:`-sensitivity issue under `pull_request_target`
- `.planning/phases/03-rename-stub-removal/03-MUTATION-LOG.md` (read verbatim) — mutation-log convention reused for GATE-04
- `git diff main..gsd/v0.15.0-milestone` (run this session) — confirmed exact divergence set for the landing-sequence risk assessment
- [GitHub REST API docs: List pull requests files](https://docs.github.com/en/rest/pulls/pulls#list-pull-requests-files) (fetched this session) — `status` enum values, 3000-file cap, `per_page`/`page` pagination

### Secondary (MEDIUM confidence)
- [cli/cli discussion #5359 "Is there any way to get more than 100 files in `gh pr view --json files`?"](https://github.com/cli/cli/discussions/5359) — confirms the 100-file GraphQL cap and the REST `--paginate` workaround
- [cli/cli discussion #5373 "support paging, server-side filtering, or increased limits"](https://github.com/cli/cli/discussions/5373) — same finding, second independent source

### Tertiary (LOW confidence)
- [community discussion #118311](https://github.com/orgs/community/discussions/118311) — cites a 300-file error message, but that message is from the *compare* API (`compare/{base}...{head}`), a different endpoint than `pulls/{n}/files`; not used to override the official docs' stated 3000-file cap on the endpoint this research recommends, flagged here only so the planner does not conflate the two endpoints

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — every dependency (`python3`, `gh`, `task`) is already in use, unpinned, in this exact repository for this exact class of policy check
- Architecture: HIGH — every pattern (trigger, checkout, env-input, verdict-function isolation) is copied from code read verbatim this session, not inferred
- Pitfalls: HIGH — all five pitfalls were found by reading the actual guard/script source, not by general GitHub Actions knowledge; two (Pitfall 2, Pitfall 3) are specific to this repository's own fixture files and would not appear in generic research

**Research date:** 2026-09-26
**Valid until:** 30 days (GitHub's REST/GraphQL pagination behavior and this repo's own shape-test fixtures are both stable; re-verify if `gh` CLI's `--json files` cap is ever raised, or if `taskfile_shape_test.go`'s fixture line numbers shift due to unrelated intervening commits)
