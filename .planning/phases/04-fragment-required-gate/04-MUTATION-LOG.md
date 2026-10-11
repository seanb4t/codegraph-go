# 04-MUTATION-LOG — Fragment-Required Gate

**Phase:** 04-fragment-required-gate
**Date:** 2026-10-11
**Scope:** Evidence for the `fragment-required` gate (GATE-01..GATE-04), organized in the
sections below:
- 04-01: the RED-first and GREEN transcripts for the tracer (Task 1) and for GATE-02/GATE-03
  (Task 2), and the executed-count run (Task 3);
- 04-02: Families (a)-(e), the mutations of our own path rule, exemption grammar, fail-closed
  precedence and executed-count guard, each shown RED against a confirmed-applied mutation and
  reverted byte-cleanly.

Every transcript below is a stdlib `unittest` run. RED evidence is the `test(04-01):` commit that
landed before its implementation plus the pasted `FAIL:` output (rule x1cjy9vyhq);
`gsd-tools check tdd-red-evidence` is not used.

## Pre-mutation cleanliness gate — the convention this log follows

Before every tracked-file mutation, AND before every revert, `git diff --quiet -- <file>` (or, for
the directory-scoped checks, `git status --porcelain -- <dir>`) is asserted to exit empty/clean.
This proves no pre-existing tracked edit was overwritten by the mutation, and no revert was a
destructive blind checkout of someone else's in-flight work. Every family entry below records this
gate's result at the point it was checked.

## RED-first transcript (04-01 Task 1)

Command: `python3 -B -m unittest discover -s scripts -p 'test_fragment_policy.py' -v 2>&1`

Run against the RED skeleton of `scripts/fragment_policy.py` (every name defined, every verdict a
placeholder pass), before the `ci(04-01):` implementation commit. Every failure is an assertion
failure; there is no `ERROR:` line and no import failure. Pre-run gate:
`git diff --quiet -- scripts/pr_template_policy.py` exited clean.

```
test_identical_shape_with_fragment_passes (test_fragment_policy.EvaluateTests.test_identical_shape_with_fragment_passes) ... FAIL
test_no_shipped_code_passes_without_fragment (test_fragment_policy.EvaluateTests.test_no_shipped_code_passes_without_fragment) ... ok
test_shape_without_fragment_fails (test_fragment_policy.EvaluateTests.test_shape_without_fragment_fails) ... FAIL
test_test_files_and_testdata_are_not_shipped_code (test_fragment_policy.EvaluateTests.test_test_files_and_testdata_are_not_shipped_code) ... FAIL
test_escape_data_encodes_command_characters (test_fragment_policy.MainTests.test_escape_data_encodes_command_characters) ... FAIL
test_main_fails_and_names_paths_when_fragment_missing (test_fragment_policy.MainTests.test_main_fails_and_names_paths_when_fragment_missing) ... FAIL
test_main_fails_closed_when_gh_exits_nonzero (test_fragment_policy.MainTests.test_main_fails_closed_when_gh_exits_nonzero) ... 
  test_main_fails_closed_when_gh_exits_nonzero (test_fragment_policy.MainTests.test_main_fails_closed_when_gh_exits_nonzero) (returncode=1) ... FAIL
  test_main_fails_closed_when_gh_exits_nonzero (test_fragment_policy.MainTests.test_main_fails_closed_when_gh_exits_nonzero) (returncode=124) ... FAIL
  test_main_fails_closed_when_gh_exits_nonzero (test_fragment_policy.MainTests.test_main_fails_closed_when_gh_exits_nonzero) (returncode=127) ... FAIL
test_main_passes_when_fragment_added (test_fragment_policy.MainTests.test_main_passes_when_fragment_added) ... FAIL
test_untrusted_paths_cannot_start_workflow_commands (test_fragment_policy.MainTests.test_untrusted_paths_cannot_start_workflow_commands) ... FAIL

======================================================================
FAIL: test_identical_shape_with_fragment_passes (test_fragment_policy.EvaluateTests.test_identical_shape_with_fragment_passes)
----------------------------------------------------------------------
Traceback (most recent call last):
  File "/Volumes/Code/github.com/seanb4t/codegraph-go/scripts/test_fragment_policy.py", line 97, in test_identical_shape_with_fragment_passes
    self.assertEqual(verdict.code, "has-fragment")
    ~~~~~~~~~~~~~~~~^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
AssertionError: 'no-shipped-code' != 'has-fragment'
- no-shipped-code
+ has-fragment


======================================================================
FAIL: test_shape_without_fragment_fails (test_fragment_policy.EvaluateTests.test_shape_without_fragment_fails)
----------------------------------------------------------------------
Traceback (most recent call last):
  File "/Volumes/Code/github.com/seanb4t/codegraph-go/scripts/test_fragment_policy.py", line 87, in test_shape_without_fragment_fails
    self.assertFalse(verdict.ok)
    ~~~~~~~~~~~~~~~~^^^^^^^^^^^^
AssertionError: True is not false

======================================================================
FAIL: test_test_files_and_testdata_are_not_shipped_code (test_fragment_policy.EvaluateTests.test_test_files_and_testdata_are_not_shipped_code)
----------------------------------------------------------------------
Traceback (most recent call last):
  File "/Volumes/Code/github.com/seanb4t/codegraph-go/scripts/test_fragment_policy.py", line 116, in test_test_files_and_testdata_are_not_shipped_code
    self.assertTrue(fragment_policy.is_shipped_code("internal/cli/root.go"))
    ~~~~~~~~~~~~~~~^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
AssertionError: False is not true

======================================================================
FAIL: test_escape_data_encodes_command_characters (test_fragment_policy.MainTests.test_escape_data_encodes_command_characters)
----------------------------------------------------------------------
Traceback (most recent call last):
  File "/Volumes/Code/github.com/seanb4t/codegraph-go/scripts/test_fragment_policy.py", line 165, in test_escape_data_encodes_command_characters
    self.assertEqual(fragment_policy.escape_data("50%\r\nx"), "50%25%0D%0Ax")
    ~~~~~~~~~~~~~~~~^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
AssertionError: '50%\r\nx' != '50%25%0D%0Ax'
+ 50%25%0D%0Ax
- 50%
- x


======================================================================
FAIL: test_main_fails_and_names_paths_when_fragment_missing (test_fragment_policy.MainTests.test_main_fails_and_names_paths_when_fragment_missing)
----------------------------------------------------------------------
Traceback (most recent call last):
  File "/Volumes/Code/github.com/seanb4t/codegraph-go/scripts/test_fragment_policy.py", line 135, in test_main_fails_and_names_paths_when_fragment_missing
    self.assertEqual(rc, 1, output)
    ~~~~~~~~~~~~~~~~^^^^^^^^^^^^^^^
AssertionError: 0 != 1 : 

======================================================================
FAIL: test_main_fails_closed_when_gh_exits_nonzero (test_fragment_policy.MainTests.test_main_fails_closed_when_gh_exits_nonzero) (returncode=1)
----------------------------------------------------------------------
Traceback (most recent call last):
  File "/Volumes/Code/github.com/seanb4t/codegraph-go/scripts/test_fragment_policy.py", line 160, in test_main_fails_closed_when_gh_exits_nonzero
    self.assertEqual(rc, 1, output)
    ~~~~~~~~~~~~~~~~^^^^^^^^^^^^^^^
AssertionError: 0 != 1 : 

======================================================================
FAIL: test_main_fails_closed_when_gh_exits_nonzero (test_fragment_policy.MainTests.test_main_fails_closed_when_gh_exits_nonzero) (returncode=124)
----------------------------------------------------------------------
Traceback (most recent call last):
  File "/Volumes/Code/github.com/seanb4t/codegraph-go/scripts/test_fragment_policy.py", line 160, in test_main_fails_closed_when_gh_exits_nonzero
    self.assertEqual(rc, 1, output)
    ~~~~~~~~~~~~~~~~^^^^^^^^^^^^^^^
AssertionError: 0 != 1 : 

======================================================================
FAIL: test_main_fails_closed_when_gh_exits_nonzero (test_fragment_policy.MainTests.test_main_fails_closed_when_gh_exits_nonzero) (returncode=127)
----------------------------------------------------------------------
Traceback (most recent call last):
  File "/Volumes/Code/github.com/seanb4t/codegraph-go/scripts/test_fragment_policy.py", line 160, in test_main_fails_closed_when_gh_exits_nonzero
    self.assertEqual(rc, 1, output)
    ~~~~~~~~~~~~~~~~^^^^^^^^^^^^^^^
AssertionError: 0 != 1 : 

======================================================================
FAIL: test_main_passes_when_fragment_added (test_fragment_policy.MainTests.test_main_passes_when_fragment_added)
----------------------------------------------------------------------
Traceback (most recent call last):
  File "/Volumes/Code/github.com/seanb4t/codegraph-go/scripts/test_fragment_policy.py", line 149, in test_main_passes_when_fragment_added
    self.assertIn("fragment-required: PASS (has-fragment)", output)
    ~~~~~~~~~~~~~^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
AssertionError: 'fragment-required: PASS (has-fragment)' not found in ''

======================================================================
FAIL: test_untrusted_paths_cannot_start_workflow_commands (test_fragment_policy.MainTests.test_untrusted_paths_cannot_start_workflow_commands)
----------------------------------------------------------------------
Traceback (most recent call last):
  File "/Volumes/Code/github.com/seanb4t/codegraph-go/scripts/test_fragment_policy.py", line 170, in test_untrusted_paths_cannot_start_workflow_commands
    self.assertEqual(rc, 1, output)
    ~~~~~~~~~~~~~~~~^^^^^^^^^^^^^^^
AssertionError: 0 != 1 : 

----------------------------------------------------------------------
Ran 9 tests in 0.001s

FAILED (failures=10)
```

## GREEN after the tracer (04-01 Task 1)

Commit: `1989a4c7` (`ci(04-01): add the fragment-required gate on pull_request_target (GATE-01)`), after the RED commit `484fc4b1`.

Command: `python3 -B -m unittest discover -s scripts -p 'test_fragment_policy.py' -v 2>&1`

```
test_identical_shape_with_fragment_passes (test_fragment_policy.EvaluateTests.test_identical_shape_with_fragment_passes) ... ok
test_no_shipped_code_passes_without_fragment (test_fragment_policy.EvaluateTests.test_no_shipped_code_passes_without_fragment) ... ok
test_shape_without_fragment_fails (test_fragment_policy.EvaluateTests.test_shape_without_fragment_fails) ... ok
test_test_files_and_testdata_are_not_shipped_code (test_fragment_policy.EvaluateTests.test_test_files_and_testdata_are_not_shipped_code) ... ok
test_escape_data_encodes_command_characters (test_fragment_policy.MainTests.test_escape_data_encodes_command_characters) ... ok
test_main_fails_and_names_paths_when_fragment_missing (test_fragment_policy.MainTests.test_main_fails_and_names_paths_when_fragment_missing) ... ok
test_main_fails_closed_when_gh_exits_nonzero (test_fragment_policy.MainTests.test_main_fails_closed_when_gh_exits_nonzero) ... ok
test_main_passes_when_fragment_added (test_fragment_policy.MainTests.test_main_passes_when_fragment_added) ... ok
test_untrusted_paths_cannot_start_workflow_commands (test_fragment_policy.MainTests.test_untrusted_paths_cannot_start_workflow_commands) ... ok

----------------------------------------------------------------------
Ran 9 tests in 0.000s

OK
```

Stub-gh end-to-end run of `task check:fragment-required`, in both directions. The stub `gh` on PATH is our test double; nothing calls GitHub (D-01). `task` prints its own failure line on the non-zero exit, so the first block ends with it.

```
=== no fragment: exit 201
::error::fragment-required: FAIL (missing-fragment): this PR changes shipped code (cmd/** or internal/**, excluding *_test.go and testdata/) but adds no changelog fragment under .changes/unreleased/
::error::fragment-required: shipped-code paths: internal/cli/root.go
::error::fragment-required: add a fragment: CI=true task changie -- new -k <Kind> -b "<one sentence>" -m PR=<number>, then commit the new .changes/unreleased/*.yaml file (kinds are declared in .changie.yaml)
::error::fragment-required: or, if this change needs no changelog entry, put <!-- changelog-exempt: <reason> --> in the PR body with a non-empty reason
fragment-required: shipped-code path: internal/cli/root.go
task: Failed to run task "check:fragment-required": exit status 1
=== with fragment: exit 0
fragment-required: PASS (has-fragment): shipped code changed and a new .changes/unreleased/ fragment was added
STUB_VERIFY_OK
```
