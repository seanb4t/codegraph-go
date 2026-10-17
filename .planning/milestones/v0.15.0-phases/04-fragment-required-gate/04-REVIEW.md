---
phase: 04-fragment-required-gate
reviewed: 2026-10-13T00:00:00Z
depth: deep
files_reviewed: 7
files_reviewed_list:
  - .github/workflows/ci.yml
  - .github/workflows/components-drift.yml
  - .github/workflows/fragment-required.yml
  - Taskfile.yml
  - internal/upgrade/taskfile_shape_test.go
  - scripts/fragment_policy.py
  - scripts/test_fragment_policy.py
findings:
  critical: 0
  warning: 3
  info: 6
  total: 9
status: issues_found
---

# Phase 4: Code Review Report

**Reviewed:** 2026-10-13
**Depth:** deep
**Files Reviewed:** 7
**Status:** issues_found

## Summary

Reviewed the Phase 4 diff (`25859d16..HEAD`) for the seven files: the `pull_request_target` workflow, the `check:fragment-required` and `check:fragment-policy` Taskfile targets, `fragment_policy.py`, its 36-case suite, the shape-test edits that retired the ruleset-drift guard, and the `ci.yml` and `components-drift.yml` edits. I also ran `go vet` and `go test ./internal/upgrade/` (both clean) and `task check:fragment-policy` (`Ran 36 tests`, `OK`, `executed 36 of 36`).

The security-relevant structure holds up. The workflow checks out the base branch with no `ref:`. Untrusted input enters only through `env:`. The run body is a single `task` call. `gh` is invoked with an argv list and no shell. Token scopes are read-only and the Go cache is disabled. Every path and reason printed passes through `escape_data`; I confirmed a hostile multi-line exemption reason comes out as `ok%0A::stop-commands::x`. The removal of the drift guard is complete: there are no dangling references outside `.planning/`, and no unused helpers remain in the Go test (it vets clean). The executed-count guard carries a positive assertion (`ran == defined`, floor 36, an `OK` line).

One real defect is in the gate's own code: a cubic-time regex that can be used to hang the check. Two WARNINGs are test-coverage gaps in branches that the gate's central claim, "fails closed whenever it cannot see the whole change", depends on. Both concern our own code, not GitHub's behavior.

## Warnings

### WR-01: `CHANGELOG_EXEMPT_MARKER` backtracks cubically; a crafted PR body hangs the gate

**File:** `scripts/fragment_policy.py:49` (also `scripts/pr_template_policy.py:83`, which has the same pattern)
**Issue:** The pattern `<!--\s*changelog-exempt:\s*(?P<reason>[^->][^>]*?)\s*-->` has overlapping quantifiers. `\s*` and `[^->]` both match whitespace, and the lazy `[^>]*?` is followed by another `\s*`. When the body holds `<!-- changelog-exempt:` followed by a long run of whitespace and no `-->`, the engine tries every split, and each split rescans. I measured `find_exemption` on `"<!-- changelog-exempt:" + " " * n`:

| n | time |
|---|---|
| 1000 | 0.19s |
| 2000 | 1.55s |
| 4000 | 19.5s |

That is cubic growth. GitHub allows PR bodies up to about 65k characters, so a body of 20k to 60k spaces runs for hours. The job then hits `timeout-minutes: 10` and the check fails.

- It cannot make the gate pass. The result is a fail-closed timeout on the author's own PR.
- It lets any fork author burn up to 10 runner-minutes per `edited` or `synchronize` event.
- The test at `test_marker_grammar_mirrors_pr_template_exempt` pins the pattern text to `pr_template_policy`'s, so a one-sided fix breaks that test.

**Fix:** Remove the ambiguity and validate the reason after matching. Change both files together (Phase 5 / GRD-15 is already slated to edit `pr_template_policy.py`) and update the mirror test, or relax that test to compare marker semantics rather than pattern text:
```python
CHANGELOG_EXEMPT_MARKER = re.compile(r"<!--[ \t]*changelog-exempt:(?P<reason>[^>]*?)-->", re.I)

def find_exemption(body):
    m = CHANGELOG_EXEMPT_MARKER.search(body)
    if m is None:
        return None
    reason = m.group("reason").strip()
    if not reason or reason.startswith("-"):
        return None
    return reason
```
Add a test with `"<!-- changelog-exempt:" + " " * 50_000` that runs under a bounded time (or simply completes), and a test that an unterminated marker returns `None`.

### WR-02: The metadata-fetch and file-list-shape fail-closed branches have no test

**File:** `scripts/test_fragment_policy.py:344-416` (covering `scripts/fragment_policy.py:211-247`, `180-203`, `250-266`)
**Issue:** The gate's claim is that it fails closed whenever it cannot see the whole change. The tests exercise only some of the branches that back it.

- Every `main()` test feeds a successful metadata response (`_meta_result(int)`). `test_main_fails_closed_when_gh_exits_nonzero` fails only the second (`/files`) call. Nothing tests the first `gh api repos/.../pulls/N` call exiting non-zero or returning malformed JSON. Nothing tests `changed_files` being missing, a bool, a string, or negative. Those are lines 221-232.
- Mutation that survives: replace the `changed_files` validation with `expected = meta.get("changed_files", len(files))`, or drop the `isinstance(expected, bool)` check. A missing count would then silently disable the truncation proof, and all 36 tests still pass.
- `parse_pr_files` is only tested with `"not json"`. The five `ValueError` shape branches (top level not a list, page not a list, item not an object, non-string `filename` or `status`, non-string `previous_filename`) are never reached. Dropping `previous_filename` validation would not be caught.
- `_run_gh` is never executed by the suite. Its 127/124 mapping is tested only as canned runner tuples. The `FileNotFoundError` mapping is our code and can be tested without gh, for example `_run_gh(["codegraph-no-such-binary"])` returns `(127, "", "gh not found on PATH")`.

**Fix:** Add table-driven cases that assert `rc == 1` and `"FAIL (api-error)"`:
```python
def test_main_fails_closed_on_bad_pr_metadata(self):
    files = _files_result([("internal/a.go", "modified"), (FRAGMENT_PATH, "added")])
    cases = (
        (1, "", "HTTP 404"),
        (0, "not json", ""),
        (0, json.dumps({}), ""),
        (0, json.dumps({"changed_files": True}), ""),
        (0, json.dumps({"changed_files": "2"}), ""),
        (0, json.dumps({"changed_files": -1}), ""),
        (0, json.dumps([]), ""),
    )
    for meta in cases:
        with self.subTest(meta=meta):
            rc, output, _ = _run_main(REPO_ENV, files, meta)
            self.assertEqual(rc, 1, output)
            self.assertIn("FAIL (api-error)", output)
            self.assertNotIn("PASS", output)
```
Add a similar table for `parse_pr_files` payloads (`{}`, `[{}]`, `[[1]]`, `[[{"filename": 1, "status": "added"}]]`, a non-string `previous_filename`). Raise the `floor=36` in `Taskfile.yml` to the new count so the guard ratchets.

### WR-03: Workflow-command escaping of the exemption reason (attacker-controlled PR body) is untested

**File:** `scripts/test_fragment_policy.py:357-372, 442-450` (covering `scripts/fragment_policy.py:271-277`)
**Issue:** The reason regex `[^>]*?` matches newlines. A PR author can therefore put a multi-line payload in the reason, for example `<!-- changelog-exempt: x\n::stop-commands::tok\n -->`, and it reaches the log on two lines: the `PASS (exempt)` line and the `::notice::` line.

- The only exempt-path test, `test_main_reports_exemption_reason_as_notice`, uses a benign single-line reason.
- The injection test covers only fork-authored file paths.
- I checked that today's output is correctly escaped (`ok%0A::stop-commands::x`), but no test pins it.
- Mutation that survives: delete `escape_data` from the `::notice::` print, or from the `message` computed for the PASS line. The PR body is the most attacker-friendly input the script handles, and the task's security story (T-04-03) rests on this escaping.

**Fix:** Extend the hostile-input test to the exempt path.
```python
def test_hostile_exemption_reason_cannot_start_workflow_commands(self):
    body = "<!-- changelog-exempt: ok\n::stop-commands::tok\n::error::forged\n -->"
    environ = {**REPO_ENV, "PR_BODY": body}
    rc, output, _ = _run_main(environ, _files_result([("internal/a.go", "modified")]))
    self.assertEqual(rc, 0, output)
    self.assertIn("%0A::stop-commands::tok%0A::error::forged", output)
    for line in output.splitlines():
        self.assertFalse(line.startswith(("::stop-commands::", "::error::")), line)
```

## Info

### IN-01: Marker semantics can surprise: the first marker decides, and quoted examples exempt

**File:** `scripts/fragment_policy.py:138-147`
**Issue:** I verified three behaviors.
- `"<!-- changelog-exempt: -->\n<!-- changelog-exempt: real -->"` returns `None`. An empty first marker masks a valid later one. This was inherited from `pr_template_policy` and is documented ("first marker decides").
- A PR body that merely quotes the syntax, for example `` use `<!-- changelog-exempt: reason --> ` to opt out``, exempts with reason `reason`. A PR that discusses this gate in prose can accidentally disable it. The literal `<reason>` placeholder printed by the gate's own hint does not match, which is good.
- A reason containing `>` (for example `a -> b`) does not match at all, so it silently fails to exempt.

None of this is a bypass, since the marker is self-service by design. It is friction and false-pass surface. Consider iterating `finditer` and taking the first marker with a non-empty reason, or document the three behaviors in the module docstring.

### IN-02: Any added file under `.changes/unreleased/` counts as a fragment

**File:** `scripts/fragment_policy.py:128-135`
**Issue:** `is_fragment` checks status, prefix, and basename `!= .gitkeep` (this is D-04 as specified). An added `.changes/unreleased/notes.txt`, an empty file, or `sub/x.md` satisfies the gate, although the failure hint tells contributors to add a `*.yaml` file. The hint and the check disagree. Requiring `path.endswith((".yaml", ".yml"))` would align them at no cost. It is optional while the exemption marker is equally easy to satisfy.

### IN-03: `LISTED_PATH_LIMIT` is defeated by the trailing per-path loop

**File:** `scripts/fragment_policy.py:279-298`
**Issue:** The `::error::` line truncates the list to 20 paths and prints `(+N more)`. The `for path in verdict.shipped:` loop then prints every path, up to 3000 lines. The limit does nothing for log volume. Either cap that loop at `LISTED_PATH_LIMIT` too, or drop the truncation and `(+N more)` suffix.

### IN-04: `fetch_changed_files` claims "Never raises", but `_run_gh` can raise

**File:** `scripts/fragment_policy.py:214, 250-266`
**Issue:** `_run_gh` catches only `FileNotFoundError` and `TimeoutExpired`. `PermissionError` and other `OSError`s propagate out of `main()` as a traceback. The exit status is still non-zero, so the gate fails closed, but without the `::error::fragment-required:` line the module promises. Catch `OSError` and return `(126, "", f"gh could not be run: {exc.strerror}")`. The 120 and 100 literals could also be named constants next to `LISTED_PATH_LIMIT`.

### IN-05: Stale scaffolding and formatting leftovers

**File:** `scripts/test_fragment_policy.py:13-15, 132-133, 372-373`; `internal/upgrade/taskfile_shape_test.go:1454`
**Issue:**
- The test docstring says "Names that a later task adds to fragment_policy are reached as module attributes ...". That is leftover from the RED-first plan and is now misleading. The `assertTrue(hasattr(...))` on line 173 is similarly redundant.
- Two double-blank lines sit inside the `EvaluateTests` and `MainTests` classes.
- The reworked WR-03 comment in the Go test is collapsed into one 150-character line while the surrounding comment block is wrapped.

### IN-06: The `OK` check in `check:fragment-policy` is not anchored to the result line

**File:** `Taskfile.yml:2294`
**Issue:** `grep -x 'OK'` accepts a line equal to `OK` anywhere in the verbose output. The description says "the result line is exactly OK". In practice `OK (skipped=1)` still fails the check, so this is only hypothetical, but `printf '%s\n' "${out}" | tail -n 1 | grep -qx 'OK'` makes the check match its description.

---

_Reviewed: 2026-10-13_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
