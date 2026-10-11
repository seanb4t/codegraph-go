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

## RED transcript (04-01 Task 2)

Command: `python3 -B -m unittest discover -s scripts -p 'test_fragment_policy.py' -v 2>&1`

Run against Task 1's GREEN module (`scripts/fragment_policy.py` at `1989a4c7`), with the 27 new methods appended to the suite and nothing else changed (`git diff --quiet -- scripts/fragment_policy.py` exited clean). 36 tests ran; 14 distinct methods fail on assertions, including all 13 the plan names. There is no `ERROR:` line.

```
test_api_error_fails_closed (test_fragment_policy.EvaluateTests.test_api_error_fails_closed) ... ok
test_empty_body_is_readable (test_fragment_policy.EvaluateTests.test_empty_body_is_readable) ... ok
test_empty_file_list_fails_closed (test_fragment_policy.EvaluateTests.test_empty_file_list_fails_closed) ... FAIL
test_evaluate_does_not_mutate_input (test_fragment_policy.EvaluateTests.test_evaluate_does_not_mutate_input) ... ok
test_exempt_marker_is_case_insensitive (test_fragment_policy.EvaluateTests.test_exempt_marker_is_case_insensitive) ... FAIL
test_exempt_marker_with_reason_passes (test_fragment_policy.EvaluateTests.test_exempt_marker_with_reason_passes) ... FAIL
test_exempt_marker_without_reason_does_not_exempt (test_fragment_policy.EvaluateTests.test_exempt_marker_without_reason_does_not_exempt) ... ok
test_fail_closed_messages_are_distinct (test_fragment_policy.EvaluateTests.test_fail_closed_messages_are_distinct) ... FAIL
test_fail_closed_precedence (test_fragment_policy.EvaluateTests.test_fail_closed_precedence) ... FAIL
test_fragment_must_be_added (test_fragment_policy.EvaluateTests.test_fragment_must_be_added) ... ok
test_fragment_outside_unreleased_does_not_count (test_fragment_policy.EvaluateTests.test_fragment_outside_unreleased_does_not_count) ... ok
test_gitkeep_is_not_a_fragment (test_fragment_policy.EvaluateTests.test_gitkeep_is_not_a_fragment) ... ok
test_identical_shape_with_fragment_passes (test_fragment_policy.EvaluateTests.test_identical_shape_with_fragment_passes) ... ok
test_marker_grammar_mirrors_pr_template_exempt (test_fragment_policy.EvaluateTests.test_marker_grammar_mirrors_pr_template_exempt) ... FAIL
test_no_shipped_code_passes_without_fragment (test_fragment_policy.EvaluateTests.test_no_shipped_code_passes_without_fragment) ... ok
test_other_marker_names_do_not_exempt (test_fragment_policy.EvaluateTests.test_other_marker_names_do_not_exempt) ... ok
test_path_rule_boundaries (test_fragment_policy.EvaluateTests.test_path_rule_boundaries) ... ok
test_removed_shipped_file_counts (test_fragment_policy.EvaluateTests.test_removed_shipped_file_counts) ... ok
test_rename_out_of_shipped_code_counts (test_fragment_policy.EvaluateTests.test_rename_out_of_shipped_code_counts) ... FAIL
test_shape_without_fragment_fails (test_fragment_policy.EvaluateTests.test_shape_without_fragment_fails) ... ok
test_status_comparison_ignores_case (test_fragment_policy.EvaluateTests.test_status_comparison_ignores_case) ... ok
test_test_files_and_testdata_are_not_shipped_code (test_fragment_policy.EvaluateTests.test_test_files_and_testdata_are_not_shipped_code) ... ok
test_unreadable_body_fails_closed (test_fragment_policy.EvaluateTests.test_unreadable_body_fails_closed) ... FAIL
test_verdict_ignores_file_order (test_fragment_policy.EvaluateTests.test_verdict_ignores_file_order) ... ok
test_escape_data_encodes_command_characters (test_fragment_policy.MainTests.test_escape_data_encodes_command_characters) ... ok
test_main_fails_and_names_paths_when_fragment_missing (test_fragment_policy.MainTests.test_main_fails_and_names_paths_when_fragment_missing) ... ok
test_main_fails_closed_on_malformed_json (test_fragment_policy.MainTests.test_main_fails_closed_on_malformed_json) ... FAIL
test_main_fails_closed_on_truncated_file_list (test_fragment_policy.MainTests.test_main_fails_closed_on_truncated_file_list) ... FAIL
test_main_fails_closed_when_gh_exits_nonzero (test_fragment_policy.MainTests.test_main_fails_closed_when_gh_exits_nonzero) ... ok
test_main_fails_closed_without_pr_body (test_fragment_policy.MainTests.test_main_fails_closed_without_pr_body) ... FAIL
test_main_flattens_paginated_pages (test_fragment_policy.MainTests.test_main_flattens_paginated_pages) ... ok
test_main_passes_when_fragment_added (test_fragment_policy.MainTests.test_main_passes_when_fragment_added) ... ok
test_main_rejects_malformed_pr_context (test_fragment_policy.MainTests.test_main_rejects_malformed_pr_context) ... 
  test_main_rejects_malformed_pr_context (test_fragment_policy.MainTests.test_main_rejects_malformed_pr_context) (override={'PR_NUMBER': ''}) ... FAIL
  test_main_rejects_malformed_pr_context (test_fragment_policy.MainTests.test_main_rejects_malformed_pr_context) (override={'PR_NUMBER': '0'}) ... FAIL
  test_main_rejects_malformed_pr_context (test_fragment_policy.MainTests.test_main_rejects_malformed_pr_context) (override={'PR_NUMBER': '07'}) ... FAIL
  test_main_rejects_malformed_pr_context (test_fragment_policy.MainTests.test_main_rejects_malformed_pr_context) (override={'PR_NUMBER': '7; rm -rf /'}) ... FAIL
  test_main_rejects_malformed_pr_context (test_fragment_policy.MainTests.test_main_rejects_malformed_pr_context) (override={'GITHUB_REPOSITORY': ''}) ... FAIL
  test_main_rejects_malformed_pr_context (test_fragment_policy.MainTests.test_main_rejects_malformed_pr_context) (override={'GITHUB_REPOSITORY': 'noslash'}) ... FAIL
test_main_reports_exemption_reason_as_notice (test_fragment_policy.MainTests.test_main_reports_exemption_reason_as_notice) ... FAIL
test_main_uses_paginated_rest_listing (test_fragment_policy.MainTests.test_main_uses_paginated_rest_listing) ... FAIL
test_untrusted_paths_cannot_start_workflow_commands (test_fragment_policy.MainTests.test_untrusted_paths_cannot_start_workflow_commands) ... ok

======================================================================
FAIL: test_empty_file_list_fails_closed (test_fragment_policy.EvaluateTests.test_empty_file_list_fails_closed)
----------------------------------------------------------------------
Traceback (most recent call last):
  File "/Volumes/Code/github.com/seanb4t/codegraph-go/scripts/test_fragment_policy.py", line 186, in test_empty_file_list_fails_closed
    self.assertFalse(verdict.ok)
    ~~~~~~~~~~~~~~~~^^^^^^^^^^^^
AssertionError: True is not false

======================================================================
FAIL: test_exempt_marker_is_case_insensitive (test_fragment_policy.EvaluateTests.test_exempt_marker_is_case_insensitive)
----------------------------------------------------------------------
Traceback (most recent call last):
  File "/Volumes/Code/github.com/seanb4t/codegraph-go/scripts/test_fragment_policy.py", line 145, in test_exempt_marker_is_case_insensitive
    self.assertTrue(verdict.ok)
    ~~~~~~~~~~~~~~~^^^^^^^^^^^^
AssertionError: False is not true

======================================================================
FAIL: test_exempt_marker_with_reason_passes (test_fragment_policy.EvaluateTests.test_exempt_marker_with_reason_passes)
----------------------------------------------------------------------
Traceback (most recent call last):
  File "/Volumes/Code/github.com/seanb4t/codegraph-go/scripts/test_fragment_policy.py", line 138, in test_exempt_marker_with_reason_passes
    self.assertTrue(verdict.ok)
    ~~~~~~~~~~~~~~~^^^^^^^^^^^^
AssertionError: False is not true

======================================================================
FAIL: test_fail_closed_messages_are_distinct (test_fragment_policy.EvaluateTests.test_fail_closed_messages_are_distinct)
----------------------------------------------------------------------
Traceback (most recent call last):
  File "/Volumes/Code/github.com/seanb4t/codegraph-go/scripts/test_fragment_policy.py", line 214, in test_fail_closed_messages_are_distinct
    self.assertFalse(verdict.ok)
    ~~~~~~~~~~~~~~~~^^^^^^^^^^^^
AssertionError: True is not false

======================================================================
FAIL: test_fail_closed_precedence (test_fragment_policy.EvaluateTests.test_fail_closed_precedence)
----------------------------------------------------------------------
Traceback (most recent call last):
  File "/Volumes/Code/github.com/seanb4t/codegraph-go/scripts/test_fragment_policy.py", line 222, in test_fail_closed_precedence
    self.assertEqual(fragment_policy.evaluate([], None).code, "empty-file-list")
    ~~~~~~~~~~~~~~~~^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
AssertionError: 'no-shipped-code' != 'empty-file-list'
- no-shipped-code
+ empty-file-list


======================================================================
FAIL: test_marker_grammar_mirrors_pr_template_exempt (test_fragment_policy.EvaluateTests.test_marker_grammar_mirrors_pr_template_exempt)
----------------------------------------------------------------------
Traceback (most recent call last):
  File "/Volumes/Code/github.com/seanb4t/codegraph-go/scripts/test_fragment_policy.py", line 173, in test_marker_grammar_mirrors_pr_template_exempt
    self.assertTrue(hasattr(fragment_policy, "CHANGELOG_EXEMPT_MARKER"))
    ~~~~~~~~~~~~~~~^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
AssertionError: False is not true

======================================================================
FAIL: test_rename_out_of_shipped_code_counts (test_fragment_policy.EvaluateTests.test_rename_out_of_shipped_code_counts)
----------------------------------------------------------------------
Traceback (most recent call last):
  File "/Volumes/Code/github.com/seanb4t/codegraph-go/scripts/test_fragment_policy.py", line 278, in test_rename_out_of_shipped_code_counts
    self.assertFalse(verdict.ok)
    ~~~~~~~~~~~~~~~~^^^^^^^^^^^^
AssertionError: True is not false

======================================================================
FAIL: test_unreadable_body_fails_closed (test_fragment_policy.EvaluateTests.test_unreadable_body_fails_closed)
----------------------------------------------------------------------
Traceback (most recent call last):
  File "/Volumes/Code/github.com/seanb4t/codegraph-go/scripts/test_fragment_policy.py", line 199, in test_unreadable_body_fails_closed
    self.assertFalse(verdict.ok)
    ~~~~~~~~~~~~~~~~^^^^^^^^^^^^
AssertionError: True is not false

======================================================================
FAIL: test_main_fails_closed_on_malformed_json (test_fragment_policy.MainTests.test_main_fails_closed_on_malformed_json)
----------------------------------------------------------------------
Traceback (most recent call last):
  File "/Volumes/Code/github.com/seanb4t/codegraph-go/scripts/test_fragment_policy.py", line 387, in test_main_fails_closed_on_malformed_json
    self.assertIsNotNone(rc, output)
    ~~~~~~~~~~~~~~~~~~~~^^^^^^^^^^^^
AssertionError: unexpectedly None : JSONDecodeError('Expecting value: line 1 column 1 (char 0)')

======================================================================
FAIL: test_main_fails_closed_on_truncated_file_list (test_fragment_policy.MainTests.test_main_fails_closed_on_truncated_file_list)
----------------------------------------------------------------------
Traceback (most recent call last):
  File "/Volumes/Code/github.com/seanb4t/codegraph-go/scripts/test_fragment_policy.py", line 380, in test_main_fails_closed_on_truncated_file_list
    self.assertEqual(rc, 1, output)
    ~~~~~~~~~~~~~~~~^^^^^^^^^^^^^^^
AssertionError: 0 != 1 : fragment-required: PASS (has-fragment): shipped code changed and a new .changes/unreleased/ fragment was added


======================================================================
FAIL: test_main_fails_closed_without_pr_body (test_fragment_policy.MainTests.test_main_fails_closed_without_pr_body)
----------------------------------------------------------------------
Traceback (most recent call last):
  File "/Volumes/Code/github.com/seanb4t/codegraph-go/scripts/test_fragment_policy.py", line 397, in test_main_fails_closed_without_pr_body
    self.assertEqual(rc, 1, output)
    ~~~~~~~~~~~~~~~~^^^^^^^^^^^^^^^
AssertionError: 0 != 1 : fragment-required: PASS (has-fragment): shipped code changed and a new .changes/unreleased/ fragment was added


======================================================================
FAIL: test_main_rejects_malformed_pr_context (test_fragment_policy.MainTests.test_main_rejects_malformed_pr_context) (override={'PR_NUMBER': ''})
----------------------------------------------------------------------
Traceback (most recent call last):
  File "/Volumes/Code/github.com/seanb4t/codegraph-go/scripts/test_fragment_policy.py", line 415, in test_main_rejects_malformed_pr_context
    self.assertIn("FAIL (api-error)", output)
    ~~~~~~~~~~~~~^^^^^^^^^^^^^^^^^^^^^^^^^^^^
AssertionError: 'FAIL (api-error)' not found in '::error::fragment-required: FAIL (missing-fragment): this PR changes shipped code (cmd/** or internal/**, excluding *_test.go and testdata/) but adds no changelog fragment under .changes/unreleased/\n::error::fragment-required: shipped-code paths: internal/a.go\n::error::fragment-required: add a fragment: CI=true task changie -- new -k <Kind> -b "<one sentence>" -m PR=<number>, then commit the new .changes/unreleased/*.yaml file (kinds are declared in .changie.yaml)\n::error::fragment-required: or, if this change needs no changelog entry, put <!-- changelog-exempt: <reason> --> in the PR body with a non-empty reason\nfragment-required: shipped-code path: internal/a.go\n'

======================================================================
FAIL: test_main_rejects_malformed_pr_context (test_fragment_policy.MainTests.test_main_rejects_malformed_pr_context) (override={'PR_NUMBER': '0'})
----------------------------------------------------------------------
Traceback (most recent call last):
  File "/Volumes/Code/github.com/seanb4t/codegraph-go/scripts/test_fragment_policy.py", line 415, in test_main_rejects_malformed_pr_context
    self.assertIn("FAIL (api-error)", output)
    ~~~~~~~~~~~~~^^^^^^^^^^^^^^^^^^^^^^^^^^^^
AssertionError: 'FAIL (api-error)' not found in '::error::fragment-required: FAIL (missing-fragment): this PR changes shipped code (cmd/** or internal/**, excluding *_test.go and testdata/) but adds no changelog fragment under .changes/unreleased/\n::error::fragment-required: shipped-code paths: internal/a.go\n::error::fragment-required: add a fragment: CI=true task changie -- new -k <Kind> -b "<one sentence>" -m PR=<number>, then commit the new .changes/unreleased/*.yaml file (kinds are declared in .changie.yaml)\n::error::fragment-required: or, if this change needs no changelog entry, put <!-- changelog-exempt: <reason> --> in the PR body with a non-empty reason\nfragment-required: shipped-code path: internal/a.go\n'

======================================================================
FAIL: test_main_rejects_malformed_pr_context (test_fragment_policy.MainTests.test_main_rejects_malformed_pr_context) (override={'PR_NUMBER': '07'})
----------------------------------------------------------------------
Traceback (most recent call last):
  File "/Volumes/Code/github.com/seanb4t/codegraph-go/scripts/test_fragment_policy.py", line 415, in test_main_rejects_malformed_pr_context
    self.assertIn("FAIL (api-error)", output)
    ~~~~~~~~~~~~~^^^^^^^^^^^^^^^^^^^^^^^^^^^^
AssertionError: 'FAIL (api-error)' not found in '::error::fragment-required: FAIL (missing-fragment): this PR changes shipped code (cmd/** or internal/**, excluding *_test.go and testdata/) but adds no changelog fragment under .changes/unreleased/\n::error::fragment-required: shipped-code paths: internal/a.go\n::error::fragment-required: add a fragment: CI=true task changie -- new -k <Kind> -b "<one sentence>" -m PR=<number>, then commit the new .changes/unreleased/*.yaml file (kinds are declared in .changie.yaml)\n::error::fragment-required: or, if this change needs no changelog entry, put <!-- changelog-exempt: <reason> --> in the PR body with a non-empty reason\nfragment-required: shipped-code path: internal/a.go\n'

======================================================================
FAIL: test_main_rejects_malformed_pr_context (test_fragment_policy.MainTests.test_main_rejects_malformed_pr_context) (override={'PR_NUMBER': '7; rm -rf /'})
----------------------------------------------------------------------
Traceback (most recent call last):
  File "/Volumes/Code/github.com/seanb4t/codegraph-go/scripts/test_fragment_policy.py", line 415, in test_main_rejects_malformed_pr_context
    self.assertIn("FAIL (api-error)", output)
    ~~~~~~~~~~~~~^^^^^^^^^^^^^^^^^^^^^^^^^^^^
AssertionError: 'FAIL (api-error)' not found in '::error::fragment-required: FAIL (missing-fragment): this PR changes shipped code (cmd/** or internal/**, excluding *_test.go and testdata/) but adds no changelog fragment under .changes/unreleased/\n::error::fragment-required: shipped-code paths: internal/a.go\n::error::fragment-required: add a fragment: CI=true task changie -- new -k <Kind> -b "<one sentence>" -m PR=<number>, then commit the new .changes/unreleased/*.yaml file (kinds are declared in .changie.yaml)\n::error::fragment-required: or, if this change needs no changelog entry, put <!-- changelog-exempt: <reason> --> in the PR body with a non-empty reason\nfragment-required: shipped-code path: internal/a.go\n'

======================================================================
FAIL: test_main_rejects_malformed_pr_context (test_fragment_policy.MainTests.test_main_rejects_malformed_pr_context) (override={'GITHUB_REPOSITORY': ''})
----------------------------------------------------------------------
Traceback (most recent call last):
  File "/Volumes/Code/github.com/seanb4t/codegraph-go/scripts/test_fragment_policy.py", line 415, in test_main_rejects_malformed_pr_context
    self.assertIn("FAIL (api-error)", output)
    ~~~~~~~~~~~~~^^^^^^^^^^^^^^^^^^^^^^^^^^^^
AssertionError: 'FAIL (api-error)' not found in '::error::fragment-required: FAIL (missing-fragment): this PR changes shipped code (cmd/** or internal/**, excluding *_test.go and testdata/) but adds no changelog fragment under .changes/unreleased/\n::error::fragment-required: shipped-code paths: internal/a.go\n::error::fragment-required: add a fragment: CI=true task changie -- new -k <Kind> -b "<one sentence>" -m PR=<number>, then commit the new .changes/unreleased/*.yaml file (kinds are declared in .changie.yaml)\n::error::fragment-required: or, if this change needs no changelog entry, put <!-- changelog-exempt: <reason> --> in the PR body with a non-empty reason\nfragment-required: shipped-code path: internal/a.go\n'

======================================================================
FAIL: test_main_rejects_malformed_pr_context (test_fragment_policy.MainTests.test_main_rejects_malformed_pr_context) (override={'GITHUB_REPOSITORY': 'noslash'})
----------------------------------------------------------------------
Traceback (most recent call last):
  File "/Volumes/Code/github.com/seanb4t/codegraph-go/scripts/test_fragment_policy.py", line 415, in test_main_rejects_malformed_pr_context
    self.assertIn("FAIL (api-error)", output)
    ~~~~~~~~~~~~~^^^^^^^^^^^^^^^^^^^^^^^^^^^^
AssertionError: 'FAIL (api-error)' not found in '::error::fragment-required: FAIL (missing-fragment): this PR changes shipped code (cmd/** or internal/**, excluding *_test.go and testdata/) but adds no changelog fragment under .changes/unreleased/\n::error::fragment-required: shipped-code paths: internal/a.go\n::error::fragment-required: add a fragment: CI=true task changie -- new -k <Kind> -b "<one sentence>" -m PR=<number>, then commit the new .changes/unreleased/*.yaml file (kinds are declared in .changie.yaml)\n::error::fragment-required: or, if this change needs no changelog entry, put <!-- changelog-exempt: <reason> --> in the PR body with a non-empty reason\nfragment-required: shipped-code path: internal/a.go\n'

======================================================================
FAIL: test_main_reports_exemption_reason_as_notice (test_fragment_policy.MainTests.test_main_reports_exemption_reason_as_notice)
----------------------------------------------------------------------
Traceback (most recent call last):
  File "/Volumes/Code/github.com/seanb4t/codegraph-go/scripts/test_fragment_policy.py", line 446, in test_main_reports_exemption_reason_as_notice
    self.assertEqual(rc, 0, output)
    ~~~~~~~~~~~~~~~~^^^^^^^^^^^^^^^
AssertionError: 1 != 0 : ::error::fragment-required: FAIL (missing-fragment): this PR changes shipped code (cmd/** or internal/**, excluding *_test.go and testdata/) but adds no changelog fragment under .changes/unreleased/
::error::fragment-required: shipped-code paths: internal/a.go
::error::fragment-required: add a fragment: CI=true task changie -- new -k <Kind> -b "<one sentence>" -m PR=<number>, then commit the new .changes/unreleased/*.yaml file (kinds are declared in .changie.yaml)
::error::fragment-required: or, if this change needs no changelog entry, put <!-- changelog-exempt: <reason> --> in the PR body with a non-empty reason
fragment-required: shipped-code path: internal/a.go


======================================================================
FAIL: test_main_uses_paginated_rest_listing (test_fragment_policy.MainTests.test_main_uses_paginated_rest_listing)
----------------------------------------------------------------------
Traceback (most recent call last):
  File "/Volumes/Code/github.com/seanb4t/codegraph-go/scripts/test_fragment_policy.py", line 430, in test_main_uses_paginated_rest_listing
    self.assertIn(["gh", "api", "repos/example/repo/pulls/7"], calls)
    ~~~~~~~~~~~~~^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
AssertionError: ['gh', 'api', 'repos/example/repo/pulls/7'] not found in [['gh', 'api', '--paginate', '--slurp', 'repos/example/repo/pulls/7/files?per_page=100']]

----------------------------------------------------------------------
Ran 36 tests in 0.003s

FAILED (failures=19)
```

## GREEN after GATE-02/GATE-03 (04-01 Task 2)

Commit: `f1fa5347` (`ci(04-01): honor changelog-exempt and fail closed on unseen changes (GATE-02, GATE-03)`), after the RED commit `7d85f860`. Only `scripts/fragment_policy.py` changed. 36 of 36 pass, and the Task 1 stub-gh run still passes in both directions, now through the meta call as well.

Command: `python3 -B -m unittest discover -s scripts -p 'test_fragment_policy.py' -v 2>&1`

```
test_api_error_fails_closed (test_fragment_policy.EvaluateTests.test_api_error_fails_closed) ... ok
test_empty_body_is_readable (test_fragment_policy.EvaluateTests.test_empty_body_is_readable) ... ok
test_empty_file_list_fails_closed (test_fragment_policy.EvaluateTests.test_empty_file_list_fails_closed) ... ok
test_evaluate_does_not_mutate_input (test_fragment_policy.EvaluateTests.test_evaluate_does_not_mutate_input) ... ok
test_exempt_marker_is_case_insensitive (test_fragment_policy.EvaluateTests.test_exempt_marker_is_case_insensitive) ... ok
test_exempt_marker_with_reason_passes (test_fragment_policy.EvaluateTests.test_exempt_marker_with_reason_passes) ... ok
test_exempt_marker_without_reason_does_not_exempt (test_fragment_policy.EvaluateTests.test_exempt_marker_without_reason_does_not_exempt) ... ok
test_fail_closed_messages_are_distinct (test_fragment_policy.EvaluateTests.test_fail_closed_messages_are_distinct) ... ok
test_fail_closed_precedence (test_fragment_policy.EvaluateTests.test_fail_closed_precedence) ... ok
test_fragment_must_be_added (test_fragment_policy.EvaluateTests.test_fragment_must_be_added) ... ok
test_fragment_outside_unreleased_does_not_count (test_fragment_policy.EvaluateTests.test_fragment_outside_unreleased_does_not_count) ... ok
test_gitkeep_is_not_a_fragment (test_fragment_policy.EvaluateTests.test_gitkeep_is_not_a_fragment) ... ok
test_identical_shape_with_fragment_passes (test_fragment_policy.EvaluateTests.test_identical_shape_with_fragment_passes) ... ok
test_marker_grammar_mirrors_pr_template_exempt (test_fragment_policy.EvaluateTests.test_marker_grammar_mirrors_pr_template_exempt) ... ok
test_no_shipped_code_passes_without_fragment (test_fragment_policy.EvaluateTests.test_no_shipped_code_passes_without_fragment) ... ok
test_other_marker_names_do_not_exempt (test_fragment_policy.EvaluateTests.test_other_marker_names_do_not_exempt) ... ok
test_path_rule_boundaries (test_fragment_policy.EvaluateTests.test_path_rule_boundaries) ... ok
test_removed_shipped_file_counts (test_fragment_policy.EvaluateTests.test_removed_shipped_file_counts) ... ok
test_rename_out_of_shipped_code_counts (test_fragment_policy.EvaluateTests.test_rename_out_of_shipped_code_counts) ... ok
test_shape_without_fragment_fails (test_fragment_policy.EvaluateTests.test_shape_without_fragment_fails) ... ok
test_status_comparison_ignores_case (test_fragment_policy.EvaluateTests.test_status_comparison_ignores_case) ... ok
test_test_files_and_testdata_are_not_shipped_code (test_fragment_policy.EvaluateTests.test_test_files_and_testdata_are_not_shipped_code) ... ok
test_unreadable_body_fails_closed (test_fragment_policy.EvaluateTests.test_unreadable_body_fails_closed) ... ok
test_verdict_ignores_file_order (test_fragment_policy.EvaluateTests.test_verdict_ignores_file_order) ... ok
test_escape_data_encodes_command_characters (test_fragment_policy.MainTests.test_escape_data_encodes_command_characters) ... ok
test_main_fails_and_names_paths_when_fragment_missing (test_fragment_policy.MainTests.test_main_fails_and_names_paths_when_fragment_missing) ... ok
test_main_fails_closed_on_malformed_json (test_fragment_policy.MainTests.test_main_fails_closed_on_malformed_json) ... ok
test_main_fails_closed_on_truncated_file_list (test_fragment_policy.MainTests.test_main_fails_closed_on_truncated_file_list) ... ok
test_main_fails_closed_when_gh_exits_nonzero (test_fragment_policy.MainTests.test_main_fails_closed_when_gh_exits_nonzero) ... ok
test_main_fails_closed_without_pr_body (test_fragment_policy.MainTests.test_main_fails_closed_without_pr_body) ... ok
test_main_flattens_paginated_pages (test_fragment_policy.MainTests.test_main_flattens_paginated_pages) ... ok
test_main_passes_when_fragment_added (test_fragment_policy.MainTests.test_main_passes_when_fragment_added) ... ok
test_main_rejects_malformed_pr_context (test_fragment_policy.MainTests.test_main_rejects_malformed_pr_context) ... ok
test_main_reports_exemption_reason_as_notice (test_fragment_policy.MainTests.test_main_reports_exemption_reason_as_notice) ... ok
test_main_uses_paginated_rest_listing (test_fragment_policy.MainTests.test_main_uses_paginated_rest_listing) ... ok
test_untrusted_paths_cannot_start_workflow_commands (test_fragment_policy.MainTests.test_untrusted_paths_cannot_start_workflow_commands) ... ok

----------------------------------------------------------------------
Ran 36 tests in 0.001s

OK
```

## Executed-count guard (04-01 Task 3)

Commit: `484a8e27` (`ci(04-01): run the fragment-policy suite in the test job with an executed-count guard (GATE-04)`). Command: `task check:fragment-policy`. The target prints the suite, then its own summary line; the 04-02 Family (d) mutation shows the guard RED.

```
test_api_error_fails_closed (test_fragment_policy.EvaluateTests.test_api_error_fails_closed) ... ok
test_empty_body_is_readable (test_fragment_policy.EvaluateTests.test_empty_body_is_readable) ... ok
test_empty_file_list_fails_closed (test_fragment_policy.EvaluateTests.test_empty_file_list_fails_closed) ... ok
test_evaluate_does_not_mutate_input (test_fragment_policy.EvaluateTests.test_evaluate_does_not_mutate_input) ... ok
test_exempt_marker_is_case_insensitive (test_fragment_policy.EvaluateTests.test_exempt_marker_is_case_insensitive) ... ok
test_exempt_marker_with_reason_passes (test_fragment_policy.EvaluateTests.test_exempt_marker_with_reason_passes) ... ok
test_exempt_marker_without_reason_does_not_exempt (test_fragment_policy.EvaluateTests.test_exempt_marker_without_reason_does_not_exempt) ... ok
test_fail_closed_messages_are_distinct (test_fragment_policy.EvaluateTests.test_fail_closed_messages_are_distinct) ... ok
test_fail_closed_precedence (test_fragment_policy.EvaluateTests.test_fail_closed_precedence) ... ok
test_fragment_must_be_added (test_fragment_policy.EvaluateTests.test_fragment_must_be_added) ... ok
test_fragment_outside_unreleased_does_not_count (test_fragment_policy.EvaluateTests.test_fragment_outside_unreleased_does_not_count) ... ok
test_gitkeep_is_not_a_fragment (test_fragment_policy.EvaluateTests.test_gitkeep_is_not_a_fragment) ... ok
test_identical_shape_with_fragment_passes (test_fragment_policy.EvaluateTests.test_identical_shape_with_fragment_passes) ... ok
test_marker_grammar_mirrors_pr_template_exempt (test_fragment_policy.EvaluateTests.test_marker_grammar_mirrors_pr_template_exempt) ... ok
test_no_shipped_code_passes_without_fragment (test_fragment_policy.EvaluateTests.test_no_shipped_code_passes_without_fragment) ... ok
test_other_marker_names_do_not_exempt (test_fragment_policy.EvaluateTests.test_other_marker_names_do_not_exempt) ... ok
test_path_rule_boundaries (test_fragment_policy.EvaluateTests.test_path_rule_boundaries) ... ok
test_removed_shipped_file_counts (test_fragment_policy.EvaluateTests.test_removed_shipped_file_counts) ... ok
test_rename_out_of_shipped_code_counts (test_fragment_policy.EvaluateTests.test_rename_out_of_shipped_code_counts) ... ok
test_shape_without_fragment_fails (test_fragment_policy.EvaluateTests.test_shape_without_fragment_fails) ... ok
test_status_comparison_ignores_case (test_fragment_policy.EvaluateTests.test_status_comparison_ignores_case) ... ok
test_test_files_and_testdata_are_not_shipped_code (test_fragment_policy.EvaluateTests.test_test_files_and_testdata_are_not_shipped_code) ... ok
test_unreadable_body_fails_closed (test_fragment_policy.EvaluateTests.test_unreadable_body_fails_closed) ... ok
test_verdict_ignores_file_order (test_fragment_policy.EvaluateTests.test_verdict_ignores_file_order) ... ok
test_escape_data_encodes_command_characters (test_fragment_policy.MainTests.test_escape_data_encodes_command_characters) ... ok
test_main_fails_and_names_paths_when_fragment_missing (test_fragment_policy.MainTests.test_main_fails_and_names_paths_when_fragment_missing) ... ok
test_main_fails_closed_on_malformed_json (test_fragment_policy.MainTests.test_main_fails_closed_on_malformed_json) ... ok
test_main_fails_closed_on_truncated_file_list (test_fragment_policy.MainTests.test_main_fails_closed_on_truncated_file_list) ... ok
test_main_fails_closed_when_gh_exits_nonzero (test_fragment_policy.MainTests.test_main_fails_closed_when_gh_exits_nonzero) ... ok
test_main_fails_closed_without_pr_body (test_fragment_policy.MainTests.test_main_fails_closed_without_pr_body) ... ok
test_main_flattens_paginated_pages (test_fragment_policy.MainTests.test_main_flattens_paginated_pages) ... ok
test_main_passes_when_fragment_added (test_fragment_policy.MainTests.test_main_passes_when_fragment_added) ... ok
test_main_rejects_malformed_pr_context (test_fragment_policy.MainTests.test_main_rejects_malformed_pr_context) ... ok
test_main_reports_exemption_reason_as_notice (test_fragment_policy.MainTests.test_main_reports_exemption_reason_as_notice) ... ok
test_main_uses_paginated_rest_listing (test_fragment_policy.MainTests.test_main_uses_paginated_rest_listing) ... ok
test_untrusted_paths_cannot_start_workflow_commands (test_fragment_policy.MainTests.test_untrusted_paths_cannot_start_workflow_commands) ... ok

----------------------------------------------------------------------
Ran 36 tests in 0.001s

OK
check:fragment-policy: executed 36 of 36 defined test methods (floor 36)
```
