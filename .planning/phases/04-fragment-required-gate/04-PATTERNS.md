# Phase 4: Fragment-Required Gate - Pattern Map

**Mapped:** 2026-09-26
**Files analyzed:** 8 (3 new, 5 modified)
**Analogs found:** 8 / 8

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|--------------------|------|-----------|-----------------|----------------|
| `scripts/fragment_policy.py` (new) | service (pure verdict fn + CLI `__main__`) | request-response (env-in, verdict-out) | `scripts/pr_template_policy.py` | exact |
| `scripts/test_fragment_policy.py` (new) | test | transform (literal inputs → verdict) | none in-repo (first Python `unittest` file); design per D-01/GATE-04, structured like `internal/upgrade/taskfile_shape_test.go`'s table-driven Go tests | no analog — see below |
| `.github/workflows/fragment-required.yml` (new) | route/config (CI workflow) | event-driven (PR lifecycle → check status) | `.github/workflows/pr-template-format.yml` | exact |
| `Taskfile.yml` — add `check:fragment-required` + `check:fragment-policy` targets | config | batch (task invocation wrapper) | `Taskfile.yml:551` `check:changie` | role-match |
| `internal/upgrade/taskfile_shape_test.go` — add `fragment-required.yml` to `inScopeWorkflowFiles`/`inScopeJobs` | test (fixture population) | transform (disk-scan → assertion) | same file's existing `inScopeJobs`/`inScopeWorkflowFiles` entries (`:188-201`, `:1663`) | exact (self-extension) |
| `internal/upgrade/taskfile_shape_test.go` — delete `readRequiredCheckNames`/`requiredStatusChecksPath`/`TestRequiredCheckNamesPreserved*`/`TestReadRequiredCheckNames_*` (D-08) | test (guard removal) | transform | n/a — deletion, not a copy target | n/a |
| `.github/workflows/ci.yml` — delete "Ruleset drift check (GRD-12)" step + its `runBodyExceptions` entry (D-08) | config (CI workflow) | event-driven | n/a — deletion | n/a |
| `scripts/check-ruleset-drift.sh`, `.github/required-status-checks.txt` | n/a | n/a | n/a — whole-file deletion (D-08) | n/a |

## Pattern Assignments

### `scripts/fragment_policy.py` (service, request-response)

**Analog:** `scripts/pr_template_policy.py` (read verbatim in full — 141 lines)

**Module docstring / env-contract pattern** (lines 1-22):
```python
#!/usr/bin/env python3
"""Decide whether a pull request body follows one of the typed PR templates.
...
Reads from the environment so untrusted PR content never reaches a shell:

    PR_BODY              the pull request body
    AUTHOR_ASSOCIATION   OWNER | MEMBER | COLLABORATOR | CONTRIBUTOR | NONE ...
    CHANGED_FILES        newline-separated repo-relative paths

Writes `action` and `result` to $GITHUB_OUTPUT. Exit status is always 0 — the
workflow decides what to do with the verdict, so a policy opinion never looks
like a crashed step.
"""
```
Mirror this shape for `fragment_policy.py`'s docstring, but note D-03's divergence: `fragment-required.yml` has **no PR-comment step**, so the check's own exit code IS the verdict (fail closed → exit 1), unlike `pr_template_policy.py`'s "always exit 0, let the workflow's `if:` decide" convention. Do not carry that "always exit 0" contract over — GATE-03 requires the check itself to fail.

**Exemption-marker regex pattern** (line 83, `EXEMPT_MARKER`):
```python
EXEMPT_MARKER = re.compile(r"<!--\s*pr-template-exempt:\s*(?P<reason>[^->][^>]*?)\s*-->", re.I)
```
D-04 mirrors this exactly, substituting the marker name:
```python
CHANGELOG_EXEMPT_MARKER = re.compile(r"<!--\s*changelog-exempt:\s*(?P<reason>[^->][^>]*?)\s*-->", re.I)
```
The `[^->][^>]*?` character class is deliberate (see research's "Don't Hand-Roll" note on why a naive `.+?` would treat a reason starting with `-` differently) — copy it verbatim, do not re-derive.

**`emit()` output-shape pattern** (lines 87-94):
```python
def emit(action: str, reason: str) -> None:
    payload = json.dumps({"action": action, "reason": reason})
    out = os.environ.get("GITHUB_OUTPUT")
    if out:
        with open(out, "a", encoding="utf-8") as fh:
            fh.write(f"action={action}\n")
            fh.write(f"result={payload}\n")
    print(f"{action}: {reason}", file=sys.stderr)
```
Model `fragment_policy.py`'s own emit/verdict-writing on this shape (action/reason pair to `$GITHUB_OUTPUT`, plus stderr for human-readable CI log output). D-03 additionally requires `::error::`-prefixed lines naming matched shipped-code paths — that is new surface this analog does not have; add it alongside, not instead of, the `emit()`-style output.

**Verdict/branching pattern** (`main()`, lines 97-131): sequential early-return checks (`marker` exemption → path-exemption → empty-body → router-marker → template-signal), each calling a named terminal (`emit`/`fail`) with a human-readable reason string. `fragment_policy.py`'s `evaluate()` should follow the identical sequential-early-return shape (api_error → body_unreadable → empty files → exemption → no-shipped-code → has-fragment → fail), per the skeleton in 04-RESEARCH.md's "Pattern 2".

**`__main__` entrypoint pattern** (lines 139-140):
```python
if __name__ == "__main__":
    raise SystemExit(main())
```
Copy verbatim.

**Divergence from this analog (do not copy):** `pr_template_policy.py` reads `CHANGED_FILES` from a prior shell step's `$GITHUB_OUTPUT` (heredoc-delimiter pattern in its companion workflow). Per D-10/research recommendation, `fragment_policy.py`'s own `__main__` must call `gh` itself via `subprocess` (REST, `--paginate`) — see the `require-issue-link.yml` analog below for the fail-closed empty-list shape, and 04-RESEARCH.md's "Anti-Patterns to Avoid" for why the heredoc pattern must NOT be reused here (it would silently escape `scripts/check-workflow-output-delimiter.sh`'s two-file-hardcoded coverage).

---

### `scripts/test_fragment_policy.py` (test, literal-input transform)

**No direct in-repo analog** (first `unittest` file in this repo's Python surface; `pr_template_policy.py` itself has no test suite — see 04-CONTEXT.md Deferred Ideas). Structure per D-01/GATE-04 and the stdlib `unittest` convention:

```python
import unittest
from fragment_policy import FileChange, evaluate

class EvaluateTests(unittest.TestCase):
    def test_shipped_code_without_fragment_fails(self):
        files = [FileChange(path="internal/foo.go", status="modified")]
        v = evaluate(files, body="", api_error=False, body_unreadable=False)
        self.assertFalse(v.ok)

    def test_shipped_code_with_added_fragment_passes(self):
        files = [
            FileChange(path="internal/foo.go", status="modified"),
            FileChange(path=".changes/unreleased/foo.yaml", status="added"),
        ]
        v = evaluate(files, body="", api_error=False, body_unreadable=False)
        self.assertTrue(v.ok)
    # ... GATE-02 exemption cases, GATE-03's three fail-closed cases,
    # each a distinct literal-input case, per D-01.

if __name__ == "__main__":
    unittest.main()
```
Executed-case-count assertion (rule `84d1gfpywd`) is enforced by the **Taskfile wrapper** (`check:fragment-policy`, see below), which greps `unittest -v`'s own `"Ran N test"` summary line — not inside the test file itself. This mirrors `internal/upgrade/taskfile_shape_test.go`'s repeated pattern of asserting a non-zero executed-case count at the harness level (e.g. `TestWorkflowRunBodiesInvokeTask`'s `len(inScopeJobs) == 0` guard, `TestInScopeJobsPopulationMatchesDisk`'s "vacuously pass over zero" checks) — the project's house style for this exact defect class.

---

### `.github/workflows/fragment-required.yml` (route/config, event-driven)

**Analog:** `.github/workflows/pr-template-format.yml` (read verbatim, 136 lines)

**Trigger + concurrency + permissions pattern** (lines 1-23):
```yaml
name: PR Template Format

on:
  pull_request_target:
    types: [opened, edited, reopened, synchronize, ready_for_review]

concurrency:
  group: ${{ github.workflow }}-${{ github.event.pull_request.number || github.ref }}
  cancel-in-progress: true

permissions:
  contents: read
  issues: write
  pull-requests: read
```
`fragment-required.yml` copies the `on:`/`concurrency:` block verbatim (same event types per Claude's Discretion). Its `permissions:` block per D-03 drops `issues: write` (no PR comment posted): `contents: read` + `pull-requests: read` only.

**Base-branch checkout pattern** (lines 30-33):
```yaml
- name: Check out policy from the base branch
  # No `ref:` — this deliberately checks out the base, not the PR head.
  uses: actions/checkout@df4cb1c069e1874edd31b4311f1884172cec0e10 # v6.0.3
```
Copy verbatim — same pin, same "no `ref:`" comment convention (D-02).

**Env-only untrusted-input pattern** (lines 60-67):
```yaml
- name: Evaluate template policy
  id: policy
  env:
    # Untrusted input reaches Python via the environment only.
    PR_BODY: ${{ github.event.pull_request.body }}
    AUTHOR_ASSOCIATION: ${{ github.event.pull_request.author_association }}
    CHANGED_FILES: ${{ steps.changed_files.outputs.files }}
  run: python3 scripts/pr_template_policy.py
```
`fragment-required.yml`'s single evaluate step follows this shape but (a) per Pitfall 4, its `run:` body must be exactly `task check:fragment-required` (not a bare `python3 ...` call — that convention belongs to this older, pre-D-01 workflow, which is itself excepted in `workflowFileExceptions`, see below); (b) drops `CHANGED_FILES` as an env var entirely since `fragment_policy.py`'s own `__main__` calls `gh` itself (no prior "Collect changed files" step at all — Pitfall 2/4). Needed env vars: `GH_TOKEN`, `GITHUB_REPOSITORY`, `PR_NUMBER`, `PR_BODY` (see 04-RESEARCH.md's workflow skeleton, "Code Examples" section, for the exact block).

**What NOT to copy from this analog:** the "Collect changed files" heredoc-delimiter step (lines 35-58) and the `actions/github-script` PR-comment step (lines 69-136) — both are explicitly out of scope per D-03 (no comment) and the anti-pattern noted above (no shell heredoc for changed files).

**Job `name:` exactness requirement:** GATE-05 requires the job's `name:` to read exactly `fragment-required` (for the ruleset context match) — unlike `pr-template-format.yml`'s job, whose `name:` (`Pull request template format`, line 27) is human-readable prose, not the raw job ID.

---

### `internal/upgrade/taskfile_shape_test.go` — `inScopeWorkflowFiles`/`inScopeJobs` additions

**Analog:** the file's own existing `inScopeJobs`/`inScopeWorkflowFiles` entries (read verbatim, lines 168-249 and 1643-1731).

**`inScopeJobs` entry pattern** (lines 188-201):
```go
var inScopeJobs = []inScopeJob{
	{Workflow: "ci.yml", JobID: "test"},
	{Workflow: "ci.yml", JobID: "actionlint"},
	...
	{Workflow: "components-drift.yml", JobID: "components-drift"},
}
```
Add `{Workflow: "fragment-required.yml", JobID: "fragment-required"}` (job's YAML map key, not its display `name:` string — both happen to be `fragment-required` per GATE-05, but the struct field binds the map key specifically).

**`inScopeWorkflowFiles` entry pattern** (line 1663):
```go
var inScopeWorkflowFiles = []string{"ci.yml", "release-please.yml", "corpora.yml", "components-drift.yml"}
```
Add `"fragment-required.yml"` to this slice — **not** to `workflowFileExceptions` (contrast with `pr-template-format.yml`/`require-issue-link.yml`, both of which sit in `workflowFileExceptions` at lines 1719-1730 because their run: bodies are `gh`/policy-script calls, not `task <target>`). `fragment-required.yml`'s sole run: body is `task check:fragment-required` (Pitfall 4), so it belongs in the in-scope population, matching Pitfall 3's warning: both fixtures must be updated in the same commit or `TestInScopeJobsPopulationMatchesDisk` fails.

**Existing `workflowFileException` entry shape to reference (not copy) for contrast** (lines 1719-1730):
```go
{
    Workflow: "pr-template-format.yml",
    Reason:   "PR-hygiene automation: its run: steps invoke a repo policy script (scripts/pr_template_policy.py) and git plumbing, not a Taskfile-target duplicate",
},
```
This is exactly the shape `fragment-required.yml` must NOT need, because its job body is a bare `task` call (D-10's explicit design choice differs from the two precedent workflows on this one point).

---

### `internal/upgrade/taskfile_shape_test.go` — D-08 deletions

**Exact regions to delete** (all read verbatim this session; line numbers as of 2026-09-26 — re-verify before editing, since earlier lines in this same file are also being added in this phase):

1. `requiredStatusChecksPath` constant + its doc comment — lines 70-76.
2. `readRequiredCheckNames` function + its doc comment — lines 78-116.
3. `TestRequiredCheckNamesPreserved` + its doc comment — lines 792-844.
4. `TestRequiredCheckNamesPreserved_ZeroJobsIsError` + its doc comment — lines 846-856.
5. `TestReadRequiredCheckNames_MissingFileIsError` — lines 858-868.
6. `TestReadRequiredCheckNames_EmptyOrBlankFileIsError` — lines 870-889.
7. `TestReadRequiredCheckNames_TrimsAndSkipsBlankLines` — lines 891-907.
8. `TestReadRequiredCheckNames_DuplicateIsError` — lines 909-923 (region continues past what was read; verify closing brace before deleting).
9. The `:1649-1654` comment block inside `inScopeWorkflowFiles`'s doc comment that cites `requiredStatusChecksPath`/`readRequiredCheckNames`/`check-ruleset-drift.sh` — this comment (part of the larger doc comment spanning ~1643-1662) needs editing, not wholesale deletion, since the surrounding comment about `inScopeWorkflowFiles` itself must stay; strike only the WR-03 sentence that name-drops the deleted machinery.

**MUST NOT touch (confirmed by direct read, not grep):** `parseWorkflowJobNames`/`mustWorkflowJobNames` — used independently by `TestGateStancesStated` (line 999, `jobNames := mustWorkflowJobNames(t, ciSrc)`) and by `TestRequiredCheckNamesPreserved` itself (the deleted test). These two helpers stay; only the required-check-specific constant/function/test family above is removed.

---

### `.github/workflows/ci.yml` — D-08 deletion

**Exact region to delete** (read verbatim, confirmed by grep this session): lines 236-252, the full "Ruleset drift check (GRD-12)" comment block + step:
```yaml
      # GRD-12 (D-05/D-06): proves the required-status-check fixture this
      # repo maintains (.github/required-status-checks.txt, D-07) still
      # matches what GitHub ruleset 20157557 (protect-main) ACTUALLY
      # requires, in both directions — ...
      - name: Ruleset drift check (GRD-12)
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: bash scripts/check-ruleset-drift.sh
```
Also remove the matching `runBodyExceptions` entry in `taskfile_shape_test.go` (the "third entry" at lines 243-248: `{Workflow: "ci.yml", Job: "test", Step: "Ruleset drift check (GRD-12)", Reason: "..."}`) — `TestWorkflowRunBodiesInvokeTask`'s stale-exception check (lines 1608-1613) fails loudly if this entry survives after the step it names is deleted, which is precisely the guard's intended behavior; delete the entry in the same commit as the step.

---

### Whole-file deletions (D-08)

- `scripts/check-ruleset-drift.sh` (182 lines) — delete entirely.
- `.github/required-status-checks.txt` (8 lines) — delete entirely.

No pattern extraction needed; these are pure removals with no successor file.

---

### `Taskfile.yml` — `check:fragment-required` / `check:fragment-policy` targets

**Analog:** `Taskfile.yml:551` `check:changie` (read verbatim, lines 551-580+)

**`desc:`/`preconditions:`/`cmds:` shape pattern** (lines 551-580):
```yaml
  check:changie:
    desc: >-
      CHG-03/CHG-04's live-tool guard (D-07), shaped like docs:cli:drift: ...
    silent: true
    preconditions:
      - sh: command -v go
        msg: "go not found — check:changie builds changie from go.tool-changie.mod (D-07)."
      - sh: command -v cmp
        msg: "cmp not found — check:changie byte-compares changie merge --dry-run against CHANGELOG.md."
    cmds:
      - |
        set -euo pipefail
        ...
```
Mirror this `desc:` (prose explaining the requirement IDs and mechanism) + `preconditions:` (tool-availability checks with a named `msg:`, per D-11's forbidden-gate-key rule: `status:`/`platforms:` are banned, only `preconditions:` is sanctioned) + `cmds:` shape for both new targets. See 04-RESEARCH.md's "Code Examples" → "Taskfile targets (skeleton)" section for the concrete skeleton (`check:fragment-required` gated on `python3`+`gh`; `check:fragment-policy` gated on `python3`, running `python3 -m unittest discover` and asserting the `Ran N test` count per rule `84d1gfpywd`, matching this project's house convention of asserting non-zero executed-case counts at the harness level).

Wire `check:fragment-policy` into `ci.yml`'s existing `test` job (already in `inScopeJobs`, no fixture change needed for that job) as a new step, immediately following the existing `check:changie` step at `ci.yml:212` per Taskfile.yml:571's own comment ("Wired into ci.yml's test job immediately after docs:cli:drift").

---

## Shared Patterns

### `pull_request_target` + base-branch checkout + env-only untrusted input
**Source:** `.github/workflows/pr-template-format.yml:1-33`
**Apply to:** `fragment-required.yml` (the only new workflow file this phase adds)
```yaml
on:
  pull_request_target:
    types: [opened, edited, reopened, synchronize, ready_for_review]
concurrency:
  group: ${{ github.workflow }}-${{ github.event.pull_request.number || github.ref }}
  cancel-in-progress: true
permissions:
  contents: read
  pull-requests: read
steps:
  - uses: actions/checkout@df4cb1c069e1874edd31b4311f1884172cec0e10 # v6.0.3
```

### Exemption-marker grammar
**Source:** `scripts/pr_template_policy.py:83` (`EXEMPT_MARKER`)
**Apply to:** `scripts/fragment_policy.py`'s `CHANGELOG_EXEMPT_MARKER` (D-04: mirrors the grammar, substitutes the marker name `changelog-exempt`)
```python
re.compile(r"<!--\s*changelog-exempt:\s*(?P<reason>[^->][^>]*?)\s*-->", re.I)
```

### Fail-closed, distinct-message error handling
**Source:** `.github/workflows/require-issue-link.yml:98-109` (three-branch `if/elif/elif/else` each writing a distinct `$GITHUB_OUTPUT` state) and `internal/upgrade/taskfile_shape_test.go`'s repeated "never a silently-empty/vacuous pass" doc-comment convention (e.g. `readRequiredCheckNames`'s zero-non-blank-lines error at lines 112-115, rule `84d1gfpywd`)
**Apply to:** `scripts/fragment_policy.py`'s `evaluate()` — GATE-03's three failure states (empty file list, API error, unreadable body) must each be a distinct, named message, never a shared generic "failed" string and never a silent pass-through.

### Single-definition CI invocation (`task <target>` only)
**Source:** `internal/upgrade/taskfile_shape_test.go:251-254` (`taskCallLineRe`) and `TestWorkflowRunBodiesInvokeTask` (lines 1573-1614)
**Apply to:** `fragment-required.yml`'s sole `run:` step, which must be exactly `task check:fragment-required` with no inline shell logic (Pitfall 4) — all `gh`-calling/verdict logic lives inside `scripts/fragment_policy.py`, invoked by the Taskfile target.

### Fixture-population dual-registration (file-level + job-level)
**Source:** `internal/upgrade/taskfile_shape_test.go` — `inScopeWorkflowFiles` (line 1663) and `inScopeJobs` (lines 188-201), cross-checked by `TestWorkflowFilePopulationMatchesDisk` (line 1761) and `TestInScopeJobsPopulationMatchesDisk` (line 1864)
**Apply to:** `fragment-required.yml` must be added to BOTH fixtures in the same commit (Pitfall 3) — file-level registration alone leaves the job-level guard failing on the next run.

### GATE-04 mutation-log evidence shape
**Source:** `.planning/phases/03-rename-stub-removal/03-MUTATION-LOG.md` (read verbatim, full file) — disposable `git worktree add --detach` scratch tree; pre-mutation `git diff --quiet` gate; single reasoned mutation; `git diff --numstat` confirms exactly the intended lines changed; pasted FAIL transcript (not an assertion) as RED evidence per rule `x1cjy9vyhq`; `git checkout --` revert; byte-clean `git diff --quiet` proof.
**Apply to:** the plan's GATE-04 mutation-log task, mutating `scripts/fragment_policy.py`'s `is_shipped_code`'s path rule (e.g. `segments[0] in ("cmd", "internal")` → `segments[0] in ("cmd",)`) and observing the `unittest` FAIL transcript, then reverting — see 04-RESEARCH.md's own "GATE-04 mutation-log template" section for the exact 8-step sequence to follow.

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `scripts/test_fragment_policy.py` | test | transform (literal-input → verdict) | First stdlib `unittest` file in this repo's Python surface — `pr_template_policy.py` has no test suite (04-CONTEXT.md Deferred Ideas confirms this gap explicitly). Structure follows D-01/GATE-04's requirements plus this repo's own Go-test house style (table-driven cases, executed-case-count assertion) rather than copying a Python analog that doesn't exist. See 04-RESEARCH.md's "Validation Architecture" section for the full Req→Test map. |

## Metadata

**Analog search scope:** `scripts/`, `.github/workflows/`, `.github/actions/`, `internal/upgrade/taskfile_shape_test.go`, `Taskfile.yml`, `.planning/phases/03-rename-stub-removal/`
**Files scanned:** 11 (all confirmed git-tracked via `git ls-files`, no gitignored mirrors involved — this repo has no `.gsd/capabilities/` install-mirror layout)
**Pattern extraction date:** 2026-09-26
