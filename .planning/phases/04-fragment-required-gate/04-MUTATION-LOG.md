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

## Setup (Task 2)

```
$ S="$(mktemp -d)/p04-scratch-wt"
$ git worktree add --detach "$S" HEAD
Preparing worktree (detached HEAD 565e3e25)
HEAD is now at 565e3e25 ci(04-02): retire the ruleset-drift guard and its required-check fixture (D-08)

$ git worktree list
/Volumes/Code/github.com/seanb4t/codegraph-go                                       565e3e25 [gsd/v0.15.0-milestone]
/Volumes/Code/herdr-worktrees/codegraph-go/worktree-brave-river-a240                2df78be0 [chore/octopusignore]
<tmp>/p04-scratch-wt 565e3e25 (detached HEAD)
```

No commit was made in the worktree. The mutated tree contains this plan's D-08 commit, so the
Taskfile and workflow files under test are the ones that will reach `main`.

## Family (a) — GATE-01: the path rule

**Test/guard:** `scripts/test_fragment_policy.py` (`EvaluateTests`), over `is_shipped_code` and `SHIPPED_ROOTS` in `scripts/fragment_policy.py`.

**What and why:** The path rule is our own code (D-01). Each mutation narrows it by one clause and the suite must notice. Run in sequence in a disposable scratch worktree; the main checkout is never mutated.

### a1 — SHIPPED_ROOTS loses "internal"

**What and why:** If `internal/**` stopped counting as shipped code, a PR changing only `internal/` would pass without a fragment. The path-rule tests must fail.

**Pre-mutation gate:** `git -C "$S" diff --quiet -- scripts/fragment_policy.py` — exit=0 (clean).

**Mutation applied:** in `$S/scripts/fragment_policy.py`, `SHIPPED_ROOTS = ("cmd", "internal")` became `SHIPPED_ROOTS = ("cmd",)`.

**Confirmed applied:**

```
$ git -C "$S" diff --numstat -- scripts/fragment_policy.py
1	1	scripts/fragment_policy.py
$ git -C "$S" diff -U0 -- scripts/fragment_policy.py
@@ -40 +40 @@ from typing import TextIO
-SHIPPED_ROOTS = ("cmd", "internal")
+SHIPPED_ROOTS = ("cmd",)
```

**Observed result** (`python3 -B -m unittest discover -s scripts -p 'test_fragment_policy.py' -v` from `$S`, exit 1):

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
test_path_rule_boundaries (test_fragment_policy.EvaluateTests.test_path_rule_boundaries) ... 
  test_path_rule_boundaries (test_fragment_policy.EvaluateTests.test_path_rule_boundaries) (shipped='internal/cli/root.go') ... FAIL
  test_path_rule_boundaries (test_fragment_policy.EvaluateTests.test_path_rule_boundaries) (shipped='internal/testdata.go') ... FAIL
  test_path_rule_boundaries (test_fragment_policy.EvaluateTests.test_path_rule_boundaries) (shipped='internal/x/testdata') ... FAIL
  test_path_rule_boundaries (test_fragment_policy.EvaluateTests.test_path_rule_boundaries) (shipped='internal/web/static/app.js') ... FAIL
  test_path_rule_boundaries (test_fragment_policy.EvaluateTests.test_path_rule_boundaries) (shipped='internal/x/y_test.go.orig') ... FAIL
test_removed_shipped_file_counts (test_fragment_policy.EvaluateTests.test_removed_shipped_file_counts) ... FAIL
test_rename_out_of_shipped_code_counts (test_fragment_policy.EvaluateTests.test_rename_out_of_shipped_code_counts) ... FAIL
test_shape_without_fragment_fails (test_fragment_policy.EvaluateTests.test_shape_without_fragment_fails) ... FAIL
test_status_comparison_ignores_case (test_fragment_policy.EvaluateTests.test_status_comparison_ignores_case) ... ok
test_test_files_and_testdata_are_not_shipped_code (test_fragment_policy.EvaluateTests.test_test_files_and_testdata_are_not_shipped_code) ... FAIL
test_unreadable_body_fails_closed (test_fragment_policy.EvaluateTests.test_unreadable_body_fails_closed) ... ok
test_verdict_ignores_file_order (test_fragment_policy.EvaluateTests.test_verdict_ignores_file_order) ... FAIL
test_escape_data_encodes_command_characters (test_fragment_policy.MainTests.test_escape_data_encodes_command_characters) ... ok
test_main_fails_and_names_paths_when_fragment_missing (test_fragment_policy.MainTests.test_main_fails_and_names_paths_when_fragment_missing) ... FAIL
test_main_fails_closed_on_malformed_json (test_fragment_policy.MainTests.test_main_fails_closed_on_malformed_json) ... ok
test_main_fails_closed_on_truncated_file_list (test_fragment_policy.MainTests.test_main_fails_closed_on_truncated_file_list) ... ok
test_main_fails_closed_when_gh_exits_nonzero (test_fragment_policy.MainTests.test_main_fails_closed_when_gh_exits_nonzero) ... ok
test_main_fails_closed_without_pr_body (test_fragment_policy.MainTests.test_main_fails_closed_without_pr_body) ... ok
test_main_flattens_paginated_pages (test_fragment_policy.MainTests.test_main_flattens_paginated_pages) ... FAIL
test_main_passes_when_fragment_added (test_fragment_policy.MainTests.test_main_passes_when_fragment_added) ... FAIL
test_main_rejects_malformed_pr_context (test_fragment_policy.MainTests.test_main_rejects_malformed_pr_context) ... ok
test_main_reports_exemption_reason_as_notice (test_fragment_policy.MainTests.test_main_reports_exemption_reason_as_notice) ... ok
test_main_uses_paginated_rest_listing (test_fragment_policy.MainTests.test_main_uses_paginated_rest_listing) ... ok
test_untrusted_paths_cannot_start_workflow_commands (test_fragment_policy.MainTests.test_untrusted_paths_cannot_start_workflow_commands) ... FAIL

======================================================================
FAIL: test_path_rule_boundaries (test_fragment_policy.EvaluateTests.test_path_rule_boundaries) (shipped='internal/cli/root.go')
----------------------------------------------------------------------
Traceback (most recent call last):
  File "<tmp>/p04-scratch-wt/scripts/test_fragment_policy.py", line 270, in test_path_rule_boundaries
    self.assertTrue(fragment_policy.is_shipped_code(path))
    ~~~~~~~~~~~~~~~^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
AssertionError: False is not true

======================================================================
FAIL: test_path_rule_boundaries (test_fragment_policy.EvaluateTests.test_path_rule_boundaries) (shipped='internal/testdata.go')
----------------------------------------------------------------------
Traceback (most recent call last):
  File "<tmp>/p04-scratch-wt/scripts/test_fragment_policy.py", line 270, in test_path_rule_boundaries
    self.assertTrue(fragment_policy.is_shipped_code(path))
    ~~~~~~~~~~~~~~~^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
AssertionError: False is not true

======================================================================
FAIL: test_path_rule_boundaries (test_fragment_policy.EvaluateTests.test_path_rule_boundaries) (shipped='internal/x/testdata')
----------------------------------------------------------------------
Traceback (most recent call last):
  File "<tmp>/p04-scratch-wt/scripts/test_fragment_policy.py", line 270, in test_path_rule_boundaries
    self.assertTrue(fragment_policy.is_shipped_code(path))
    ~~~~~~~~~~~~~~~^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
AssertionError: False is not true

======================================================================
FAIL: test_path_rule_boundaries (test_fragment_policy.EvaluateTests.test_path_rule_boundaries) (shipped='internal/web/static/app.js')
----------------------------------------------------------------------
Traceback (most recent call last):
  File "<tmp>/p04-scratch-wt/scripts/test_fragment_policy.py", line 270, in test_path_rule_boundaries
    self.assertTrue(fragment_policy.is_shipped_code(path))
    ~~~~~~~~~~~~~~~^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
AssertionError: False is not true

======================================================================
FAIL: test_path_rule_boundaries (test_fragment_policy.EvaluateTests.test_path_rule_boundaries) (shipped='internal/x/y_test.go.orig')
----------------------------------------------------------------------
Traceback (most recent call last):
  File "<tmp>/p04-scratch-wt/scripts/test_fragment_policy.py", line 270, in test_path_rule_boundaries
    self.assertTrue(fragment_policy.is_shipped_code(path))
    ~~~~~~~~~~~~~~~^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
AssertionError: False is not true

======================================================================
FAIL: test_removed_shipped_file_counts (test_fragment_policy.EvaluateTests.test_removed_shipped_file_counts)
----------------------------------------------------------------------
Traceback (most recent call last):
  File "<tmp>/p04-scratch-wt/scripts/test_fragment_policy.py", line 286, in test_removed_shipped_file_counts
    self.assertFalse(verdict.ok)
    ~~~~~~~~~~~~~~~~^^^^^^^^^^^^
AssertionError: True is not false

======================================================================
FAIL: test_rename_out_of_shipped_code_counts (test_fragment_policy.EvaluateTests.test_rename_out_of_shipped_code_counts)
----------------------------------------------------------------------
Traceback (most recent call last):
  File "<tmp>/p04-scratch-wt/scripts/test_fragment_policy.py", line 278, in test_rename_out_of_shipped_code_counts
    self.assertFalse(verdict.ok)
    ~~~~~~~~~~~~~~~~^^^^^^^^^^^^
AssertionError: True is not false

======================================================================
FAIL: test_shape_without_fragment_fails (test_fragment_policy.EvaluateTests.test_shape_without_fragment_fails)
----------------------------------------------------------------------
Traceback (most recent call last):
  File "<tmp>/p04-scratch-wt/scripts/test_fragment_policy.py", line 93, in test_shape_without_fragment_fails
    self.assertEqual(
    ~~~~~~~~~~~~~~~~^
        verdict.shipped, ("cmd/codegraph/main.go", "internal/cli/root.go")
        ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
    )
    ^
AssertionError: Tuples differ: ('cmd/codegraph/main.go',) != ('cmd/codegraph/main.go', 'internal/cli/root.go')

Second tuple contains 1 additional elements.
First extra element 1:
'internal/cli/root.go'

- ('cmd/codegraph/main.go',)
+ ('cmd/codegraph/main.go', 'internal/cli/root.go')

======================================================================
FAIL: test_test_files_and_testdata_are_not_shipped_code (test_fragment_policy.EvaluateTests.test_test_files_and_testdata_are_not_shipped_code)
----------------------------------------------------------------------
Traceback (most recent call last):
  File "<tmp>/p04-scratch-wt/scripts/test_fragment_policy.py", line 120, in test_test_files_and_testdata_are_not_shipped_code
    self.assertTrue(fragment_policy.is_shipped_code("internal/cli/root.go"))
    ~~~~~~~~~~~~~~~^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
AssertionError: False is not true

======================================================================
FAIL: test_verdict_ignores_file_order (test_fragment_policy.EvaluateTests.test_verdict_ignores_file_order)
----------------------------------------------------------------------
Traceback (most recent call last):
  File "<tmp>/p04-scratch-wt/scripts/test_fragment_policy.py", line 310, in test_verdict_ignores_file_order
    self.assertEqual(
    ~~~~~~~~~~~~~~~~^
        fragment_policy.evaluate(doubled, "").shipped, ("internal/a.go",)
        ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
    )
    ^
AssertionError: Tuples differ: () != ('internal/a.go',)

Second tuple contains 1 additional elements.
First extra element 0:
'internal/a.go'

- ()
+ ('internal/a.go',)

======================================================================
FAIL: test_main_fails_and_names_paths_when_fragment_missing (test_fragment_policy.MainTests.test_main_fails_and_names_paths_when_fragment_missing)
----------------------------------------------------------------------
Traceback (most recent call last):
  File "<tmp>/p04-scratch-wt/scripts/test_fragment_policy.py", line 328, in test_main_fails_and_names_paths_when_fragment_missing
    self.assertEqual(rc, 1, output)
    ~~~~~~~~~~~~~~~~^^^^^^^^^^^^^^^
AssertionError: 0 != 1 : fragment-required: PASS (no-shipped-code): no changed file is shipped code (cmd/** or internal/**, excluding *_test.go and testdata/)


======================================================================
FAIL: test_main_flattens_paginated_pages (test_fragment_policy.MainTests.test_main_flattens_paginated_pages)
----------------------------------------------------------------------
Traceback (most recent call last):
  File "<tmp>/p04-scratch-wt/scripts/test_fragment_policy.py", line 424, in test_main_flattens_paginated_pages
    self.assertIn("PASS (has-fragment)", output)
    ~~~~~~~~~~~~~^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
AssertionError: 'PASS (has-fragment)' not found in 'fragment-required: PASS (no-shipped-code): no changed file is shipped code (cmd/** or internal/**, excluding *_test.go and testdata/)\n'

======================================================================
FAIL: test_main_passes_when_fragment_added (test_fragment_policy.MainTests.test_main_passes_when_fragment_added)
----------------------------------------------------------------------
Traceback (most recent call last):
  File "<tmp>/p04-scratch-wt/scripts/test_fragment_policy.py", line 342, in test_main_passes_when_fragment_added
    self.assertIn("fragment-required: PASS (has-fragment)", output)
    ~~~~~~~~~~~~~^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
AssertionError: 'fragment-required: PASS (has-fragment)' not found in 'fragment-required: PASS (no-shipped-code): no changed file is shipped code (cmd/** or internal/**, excluding *_test.go and testdata/)\n'

======================================================================
FAIL: test_untrusted_paths_cannot_start_workflow_commands (test_fragment_policy.MainTests.test_untrusted_paths_cannot_start_workflow_commands)
----------------------------------------------------------------------
Traceback (most recent call last):
  File "<tmp>/p04-scratch-wt/scripts/test_fragment_policy.py", line 363, in test_untrusted_paths_cannot_start_workflow_commands
    self.assertEqual(rc, 1, output)
    ~~~~~~~~~~~~~~~~^^^^^^^^^^^^^^^
AssertionError: 0 != 1 : fragment-required: PASS (no-shipped-code): no changed file is shipped code (cmd/** or internal/**, excluding *_test.go and testdata/)


----------------------------------------------------------------------
Ran 36 tests in 0.002s

FAILED (failures=14)
```

**Revert:** `git -C "$S" checkout -- scripts/fragment_policy.py`.

**Byte-clean proof:** `git -C "$S" diff --quiet` — exit=0 (clean).

### a2 — the testdata-directory clause is removed from is_shipped_code

**What and why:** If files under a `testdata/` directory counted as shipped code, test fixtures would demand changelog fragments. Only the `TESTDATA_DIR` clause is removed; the `_test.go` clause stays.

**Pre-mutation gate:** `git -C "$S" diff --quiet -- scripts/fragment_policy.py` — exit=0 (clean).

**Mutation applied:** in `$S/scripts/fragment_policy.py`, `return TESTDATA_DIR not in segments[:-1]` became `return True`.

**Confirmed applied:**

```
$ git -C "$S" diff --numstat -- scripts/fragment_policy.py
1	1	scripts/fragment_policy.py
$ git -C "$S" diff -U0 -- scripts/fragment_policy.py
@@ -115 +115 @@ def is_shipped_code(path: str) -> bool:
-    return TESTDATA_DIR not in segments[:-1]
+    return True
```

**Observed result** (`python3 -B -m unittest discover -s scripts -p 'test_fragment_policy.py' -v` from `$S`, exit 1):

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
test_path_rule_boundaries (test_fragment_policy.EvaluateTests.test_path_rule_boundaries) ... 
  test_path_rule_boundaries (test_fragment_policy.EvaluateTests.test_path_rule_boundaries) (not_shipped='internal/x/testdata/y.go') ... FAIL
test_removed_shipped_file_counts (test_fragment_policy.EvaluateTests.test_removed_shipped_file_counts) ... ok
test_rename_out_of_shipped_code_counts (test_fragment_policy.EvaluateTests.test_rename_out_of_shipped_code_counts) ... ok
test_shape_without_fragment_fails (test_fragment_policy.EvaluateTests.test_shape_without_fragment_fails) ... ok
test_status_comparison_ignores_case (test_fragment_policy.EvaluateTests.test_status_comparison_ignores_case) ... ok
test_test_files_and_testdata_are_not_shipped_code (test_fragment_policy.EvaluateTests.test_test_files_and_testdata_are_not_shipped_code) ... 
  test_test_files_and_testdata_are_not_shipped_code (test_fragment_policy.EvaluateTests.test_test_files_and_testdata_are_not_shipped_code) (path='internal/cli/testdata/golden/x.txt') ... FAIL
  test_test_files_and_testdata_are_not_shipped_code (test_fragment_policy.EvaluateTests.test_test_files_and_testdata_are_not_shipped_code) (path='cmd/codegraph/testdata/fixture.go') ... FAIL
test_test_files_and_testdata_are_not_shipped_code (test_fragment_policy.EvaluateTests.test_test_files_and_testdata_are_not_shipped_code) ... FAIL
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

======================================================================
FAIL: test_path_rule_boundaries (test_fragment_policy.EvaluateTests.test_path_rule_boundaries) (not_shipped='internal/x/testdata/y.go')
----------------------------------------------------------------------
Traceback (most recent call last):
  File "<tmp>/p04-scratch-wt/scripts/test_fragment_policy.py", line 273, in test_path_rule_boundaries
    self.assertFalse(fragment_policy.is_shipped_code(path))
    ~~~~~~~~~~~~~~~~^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
AssertionError: True is not false

======================================================================
FAIL: test_test_files_and_testdata_are_not_shipped_code (test_fragment_policy.EvaluateTests.test_test_files_and_testdata_are_not_shipped_code) (path='internal/cli/testdata/golden/x.txt')
----------------------------------------------------------------------
Traceback (most recent call last):
  File "<tmp>/p04-scratch-wt/scripts/test_fragment_policy.py", line 128, in test_test_files_and_testdata_are_not_shipped_code
    self.assertFalse(fragment_policy.is_shipped_code(path))
    ~~~~~~~~~~~~~~~~^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
AssertionError: True is not false

======================================================================
FAIL: test_test_files_and_testdata_are_not_shipped_code (test_fragment_policy.EvaluateTests.test_test_files_and_testdata_are_not_shipped_code) (path='cmd/codegraph/testdata/fixture.go')
----------------------------------------------------------------------
Traceback (most recent call last):
  File "<tmp>/p04-scratch-wt/scripts/test_fragment_policy.py", line 128, in test_test_files_and_testdata_are_not_shipped_code
    self.assertFalse(fragment_policy.is_shipped_code(path))
    ~~~~~~~~~~~~~~~~^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
AssertionError: True is not false

======================================================================
FAIL: test_test_files_and_testdata_are_not_shipped_code (test_fragment_policy.EvaluateTests.test_test_files_and_testdata_are_not_shipped_code)
----------------------------------------------------------------------
Traceback (most recent call last):
  File "<tmp>/p04-scratch-wt/scripts/test_fragment_policy.py", line 130, in test_test_files_and_testdata_are_not_shipped_code
    self.assertTrue(verdict.ok)
    ~~~~~~~~~~~~~~~^^^^^^^^^^^^
AssertionError: False is not true

----------------------------------------------------------------------
Ran 36 tests in 0.001s

FAILED (failures=4)
```

**Revert:** `git -C "$S" checkout -- scripts/fragment_policy.py`.

**Byte-clean proof:** `git -C "$S" diff --quiet` — exit=0 (clean).

## Family (b) — D-04/GATE-02: the fragment and exemption rules

**Test/guard:** `scripts/test_fragment_policy.py` (`EvaluateTests`), over `is_fragment` and `find_exemption`.

**What and why:** A fragment must be a newly ADDED file under `.changes/unreleased/`, and an exemption needs a non-empty reason. Each mutation loosens one of those and the suite must notice.

### b1 — is_fragment ignores the status

**What and why:** If a modified or removed file under `.changes/unreleased/` counted as a fragment, a PR that merely edited an existing fragment would pass. Only the status condition is dropped.

**Pre-mutation gate:** `git -C "$S" diff --quiet -- scripts/fragment_policy.py` — exit=0 (clean).

**Mutation applied:** in `$S/scripts/fragment_policy.py`, `change.status.lower() == FRAGMENT_STATUS
        and change.path.startswith(FRAGMENT_DIR)` became `change.path.startswith(FRAGMENT_DIR)`.

**Confirmed applied:**

```
$ git -C "$S" diff --numstat -- scripts/fragment_policy.py
1	2	scripts/fragment_policy.py
$ git -C "$S" diff -U0 -- scripts/fragment_policy.py
@@ -131,2 +131 @@ def is_fragment(change: FileChange) -> bool:
-        change.status.lower() == FRAGMENT_STATUS
-        and change.path.startswith(FRAGMENT_DIR)
+        change.path.startswith(FRAGMENT_DIR)
```

**Observed result** (`python3 -B -m unittest discover -s scripts -p 'test_fragment_policy.py' -v` from `$S`, exit 1):

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
test_fragment_must_be_added (test_fragment_policy.EvaluateTests.test_fragment_must_be_added) ... 
  test_fragment_must_be_added (test_fragment_policy.EvaluateTests.test_fragment_must_be_added) (status='modified') ... FAIL
  test_fragment_must_be_added (test_fragment_policy.EvaluateTests.test_fragment_must_be_added) (status='renamed') ... FAIL
  test_fragment_must_be_added (test_fragment_policy.EvaluateTests.test_fragment_must_be_added) (status='removed') ... FAIL
  test_fragment_must_be_added (test_fragment_policy.EvaluateTests.test_fragment_must_be_added) (status='copied') ... FAIL
  test_fragment_must_be_added (test_fragment_policy.EvaluateTests.test_fragment_must_be_added) (status='changed') ... FAIL
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

======================================================================
FAIL: test_fragment_must_be_added (test_fragment_policy.EvaluateTests.test_fragment_must_be_added) (status='modified')
----------------------------------------------------------------------
Traceback (most recent call last):
  File "<tmp>/p04-scratch-wt/scripts/test_fragment_policy.py", line 234, in test_fragment_must_be_added
    self.assertFalse(verdict.ok)
    ~~~~~~~~~~~~~~~~^^^^^^^^^^^^
AssertionError: True is not false

======================================================================
FAIL: test_fragment_must_be_added (test_fragment_policy.EvaluateTests.test_fragment_must_be_added) (status='renamed')
----------------------------------------------------------------------
Traceback (most recent call last):
  File "<tmp>/p04-scratch-wt/scripts/test_fragment_policy.py", line 234, in test_fragment_must_be_added
    self.assertFalse(verdict.ok)
    ~~~~~~~~~~~~~~~~^^^^^^^^^^^^
AssertionError: True is not false

======================================================================
FAIL: test_fragment_must_be_added (test_fragment_policy.EvaluateTests.test_fragment_must_be_added) (status='removed')
----------------------------------------------------------------------
Traceback (most recent call last):
  File "<tmp>/p04-scratch-wt/scripts/test_fragment_policy.py", line 234, in test_fragment_must_be_added
    self.assertFalse(verdict.ok)
    ~~~~~~~~~~~~~~~~^^^^^^^^^^^^
AssertionError: True is not false

======================================================================
FAIL: test_fragment_must_be_added (test_fragment_policy.EvaluateTests.test_fragment_must_be_added) (status='copied')
----------------------------------------------------------------------
Traceback (most recent call last):
  File "<tmp>/p04-scratch-wt/scripts/test_fragment_policy.py", line 234, in test_fragment_must_be_added
    self.assertFalse(verdict.ok)
    ~~~~~~~~~~~~~~~~^^^^^^^^^^^^
AssertionError: True is not false

======================================================================
FAIL: test_fragment_must_be_added (test_fragment_policy.EvaluateTests.test_fragment_must_be_added) (status='changed')
----------------------------------------------------------------------
Traceback (most recent call last):
  File "<tmp>/p04-scratch-wt/scripts/test_fragment_policy.py", line 234, in test_fragment_must_be_added
    self.assertFalse(verdict.ok)
    ~~~~~~~~~~~~~~~~^^^^^^^^^^^^
AssertionError: True is not false

----------------------------------------------------------------------
Ran 36 tests in 0.002s

FAILED (failures=5)
```

**Revert:** `git -C "$S" checkout -- scripts/fragment_policy.py`.

**Byte-clean proof:** `git -C "$S" diff --quiet` — exit=0 (clean).

### b2 — find_exemption accepts an empty reason

**What and why:** If a marker with an empty or whitespace-only reason exempted the PR, `<!-- changelog-exempt: -->` would be a free pass. `find_exemption` now returns the stripped reason even when it is empty, and `evaluate` only tests `is not None`.

**Pre-mutation gate:** `git -C "$S" diff --quiet -- scripts/fragment_policy.py` — exit=0 (clean).

**Mutation applied:** in `$S/scripts/fragment_policy.py`, `return match.group("reason").strip() or None` became `return match.group("reason").strip()`.

**Confirmed applied:**

```
$ git -C "$S" diff --numstat -- scripts/fragment_policy.py
1	1	scripts/fragment_policy.py
$ git -C "$S" diff -U0 -- scripts/fragment_policy.py
@@ -147 +147 @@ def find_exemption(body: str) -> str | None:
-    return match.group("reason").strip() or None
+    return match.group("reason").strip()
```

**Observed result** (`python3 -B -m unittest discover -s scripts -p 'test_fragment_policy.py' -v` from `$S`, exit 1):

```
test_api_error_fails_closed (test_fragment_policy.EvaluateTests.test_api_error_fails_closed) ... ok
test_empty_body_is_readable (test_fragment_policy.EvaluateTests.test_empty_body_is_readable) ... ok
test_empty_file_list_fails_closed (test_fragment_policy.EvaluateTests.test_empty_file_list_fails_closed) ... ok
test_evaluate_does_not_mutate_input (test_fragment_policy.EvaluateTests.test_evaluate_does_not_mutate_input) ... ok
test_exempt_marker_is_case_insensitive (test_fragment_policy.EvaluateTests.test_exempt_marker_is_case_insensitive) ... ok
test_exempt_marker_with_reason_passes (test_fragment_policy.EvaluateTests.test_exempt_marker_with_reason_passes) ... ok
test_exempt_marker_without_reason_does_not_exempt (test_fragment_policy.EvaluateTests.test_exempt_marker_without_reason_does_not_exempt) ... 
  test_exempt_marker_without_reason_does_not_exempt (test_fragment_policy.EvaluateTests.test_exempt_marker_without_reason_does_not_exempt) (body='<!-- changelog-exempt: -->') ... FAIL
  test_exempt_marker_without_reason_does_not_exempt (test_fragment_policy.EvaluateTests.test_exempt_marker_without_reason_does_not_exempt) (body='<!-- changelog-exempt:    -->') ... FAIL
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

======================================================================
FAIL: test_exempt_marker_without_reason_does_not_exempt (test_fragment_policy.EvaluateTests.test_exempt_marker_without_reason_does_not_exempt) (body='<!-- changelog-exempt: -->')
----------------------------------------------------------------------
Traceback (most recent call last):
  File "<tmp>/p04-scratch-wt/scripts/test_fragment_policy.py", line 157, in test_exempt_marker_without_reason_does_not_exempt
    self.assertFalse(verdict.ok)
    ~~~~~~~~~~~~~~~~^^^^^^^^^^^^
AssertionError: True is not false

======================================================================
FAIL: test_exempt_marker_without_reason_does_not_exempt (test_fragment_policy.EvaluateTests.test_exempt_marker_without_reason_does_not_exempt) (body='<!-- changelog-exempt:    -->')
----------------------------------------------------------------------
Traceback (most recent call last):
  File "<tmp>/p04-scratch-wt/scripts/test_fragment_policy.py", line 157, in test_exempt_marker_without_reason_does_not_exempt
    self.assertFalse(verdict.ok)
    ~~~~~~~~~~~~~~~~^^^^^^^^^^^^
AssertionError: True is not false

----------------------------------------------------------------------
Ran 36 tests in 0.001s

FAILED (failures=2)
```

**Revert:** `git -C "$S" checkout -- scripts/fragment_policy.py`.

**Byte-clean proof:** `git -C "$S" diff --quiet` — exit=0 (clean).

## Family (c) — GATE-03: fail closed

**Test/guard:** `scripts/test_fragment_policy.py` (`EvaluateTests`, `MainTests`), over `evaluate` and `fetch_changed_files`.

**What and why:** The gate must fail, never pass, when it cannot see the whole change. Each mutation deletes one fail-closed branch.

### c1 — evaluate's empty-file-list branch is deleted

**What and why:** If an empty changed-file list fell through to "no shipped code", a listing failure that returned nothing would pass the gate.

**Pre-mutation gate:** `git -C "$S" diff --quiet -- scripts/fragment_policy.py` — exit=0 (clean).

**Mutation applied:** in `$S/scripts/fragment_policy.py`, `if not files:
        return Verdict(False, EMPTY_FILE_LIST, MESSAGES[EMPTY_FILE_LIST])` became `(deleted)`.

**Confirmed applied:**

```
$ git -C "$S" diff --numstat -- scripts/fragment_policy.py
0	2	scripts/fragment_policy.py
$ git -C "$S" diff -U0 -- scripts/fragment_policy.py
@@ -160,2 +159,0 @@ def evaluate(
-    if not files:
-        return Verdict(False, EMPTY_FILE_LIST, MESSAGES[EMPTY_FILE_LIST])
```

**Observed result** (`python3 -B -m unittest discover -s scripts -p 'test_fragment_policy.py' -v` from `$S`, exit 1):

```
test_api_error_fails_closed (test_fragment_policy.EvaluateTests.test_api_error_fails_closed) ... ok
test_empty_body_is_readable (test_fragment_policy.EvaluateTests.test_empty_body_is_readable) ... ok
test_empty_file_list_fails_closed (test_fragment_policy.EvaluateTests.test_empty_file_list_fails_closed) ... FAIL
test_evaluate_does_not_mutate_input (test_fragment_policy.EvaluateTests.test_evaluate_does_not_mutate_input) ... ok
test_exempt_marker_is_case_insensitive (test_fragment_policy.EvaluateTests.test_exempt_marker_is_case_insensitive) ... ok
test_exempt_marker_with_reason_passes (test_fragment_policy.EvaluateTests.test_exempt_marker_with_reason_passes) ... ok
test_exempt_marker_without_reason_does_not_exempt (test_fragment_policy.EvaluateTests.test_exempt_marker_without_reason_does_not_exempt) ... ok
test_fail_closed_messages_are_distinct (test_fragment_policy.EvaluateTests.test_fail_closed_messages_are_distinct) ... FAIL
test_fail_closed_precedence (test_fragment_policy.EvaluateTests.test_fail_closed_precedence) ... FAIL
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

======================================================================
FAIL: test_empty_file_list_fails_closed (test_fragment_policy.EvaluateTests.test_empty_file_list_fails_closed)
----------------------------------------------------------------------
Traceback (most recent call last):
  File "<tmp>/p04-scratch-wt/scripts/test_fragment_policy.py", line 186, in test_empty_file_list_fails_closed
    self.assertFalse(verdict.ok)
    ~~~~~~~~~~~~~~~~^^^^^^^^^^^^
AssertionError: True is not false

======================================================================
FAIL: test_fail_closed_messages_are_distinct (test_fragment_policy.EvaluateTests.test_fail_closed_messages_are_distinct)
----------------------------------------------------------------------
Traceback (most recent call last):
  File "<tmp>/p04-scratch-wt/scripts/test_fragment_policy.py", line 214, in test_fail_closed_messages_are_distinct
    self.assertFalse(verdict.ok)
    ~~~~~~~~~~~~~~~~^^^^^^^^^^^^
AssertionError: True is not false

======================================================================
FAIL: test_fail_closed_precedence (test_fragment_policy.EvaluateTests.test_fail_closed_precedence)
----------------------------------------------------------------------
Traceback (most recent call last):
  File "<tmp>/p04-scratch-wt/scripts/test_fragment_policy.py", line 222, in test_fail_closed_precedence
    self.assertEqual(fragment_policy.evaluate([], None).code, "empty-file-list")
    ~~~~~~~~~~~~~~~~^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
AssertionError: 'body-unreadable' != 'empty-file-list'
- body-unreadable
+ empty-file-list


----------------------------------------------------------------------
Ran 36 tests in 0.001s

FAILED (failures=3)
```

**Revert:** `git -C "$S" checkout -- scripts/fragment_policy.py`.

**Byte-clean proof:** `git -C "$S" diff --quiet` — exit=0 (clean).

### c2 — fetch_changed_files' changed_files cross-check is deleted

**What and why:** If the fetched count were not compared with the count GitHub reports for the PR, a truncated listing (GitHub lists at most 3000 files) would be judged as if it were complete.

**Pre-mutation gate:** `git -C "$S" diff --quiet -- scripts/fragment_policy.py` — exit=0 (clean).

**Mutation applied:** in `$S/scripts/fragment_policy.py`, `if len(files) != expected:
        return [], (
            f"file list incomplete: fetched {len(files)} of {expected} changed files "
            "(GitHub lists at most 3000 files per pull request)"
        )` became `(deleted)`.

**Confirmed applied:**

```
$ git -C "$S" diff --numstat -- scripts/fragment_policy.py
0	5	scripts/fragment_policy.py
$ git -C "$S" diff -U0 -- scripts/fragment_policy.py
@@ -242,5 +241,0 @@ def fetch_changed_files(
-    if len(files) != expected:
-        return [], (
-            f"file list incomplete: fetched {len(files)} of {expected} changed files "
-            "(GitHub lists at most 3000 files per pull request)"
-        )
```

**Observed result** (`python3 -B -m unittest discover -s scripts -p 'test_fragment_policy.py' -v` from `$S`, exit 1):

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
test_main_fails_closed_on_truncated_file_list (test_fragment_policy.MainTests.test_main_fails_closed_on_truncated_file_list) ... FAIL
test_main_fails_closed_when_gh_exits_nonzero (test_fragment_policy.MainTests.test_main_fails_closed_when_gh_exits_nonzero) ... ok
test_main_fails_closed_without_pr_body (test_fragment_policy.MainTests.test_main_fails_closed_without_pr_body) ... ok
test_main_flattens_paginated_pages (test_fragment_policy.MainTests.test_main_flattens_paginated_pages) ... ok
test_main_passes_when_fragment_added (test_fragment_policy.MainTests.test_main_passes_when_fragment_added) ... ok
test_main_rejects_malformed_pr_context (test_fragment_policy.MainTests.test_main_rejects_malformed_pr_context) ... ok
test_main_reports_exemption_reason_as_notice (test_fragment_policy.MainTests.test_main_reports_exemption_reason_as_notice) ... ok
test_main_uses_paginated_rest_listing (test_fragment_policy.MainTests.test_main_uses_paginated_rest_listing) ... ok
test_untrusted_paths_cannot_start_workflow_commands (test_fragment_policy.MainTests.test_untrusted_paths_cannot_start_workflow_commands) ... ok

======================================================================
FAIL: test_main_fails_closed_on_truncated_file_list (test_fragment_policy.MainTests.test_main_fails_closed_on_truncated_file_list)
----------------------------------------------------------------------
Traceback (most recent call last):
  File "<tmp>/p04-scratch-wt/scripts/test_fragment_policy.py", line 380, in test_main_fails_closed_on_truncated_file_list
    self.assertEqual(rc, 1, output)
    ~~~~~~~~~~~~~~~~^^^^^^^^^^^^^^^
AssertionError: 0 != 1 : fragment-required: PASS (has-fragment): shipped code changed and a new .changes/unreleased/ fragment was added


----------------------------------------------------------------------
Ran 36 tests in 0.001s

FAILED (failures=1)
```

**Revert:** `git -C "$S" checkout -- scripts/fragment_policy.py`.

**Byte-clean proof:** `git -C "$S" diff --quiet` — exit=0 (clean).

## Family (d) — rule 84d1gfpywd: the executed-count guard

**Test/guard:** `task check:fragment-policy` (Taskfile.yml), the guard that makes a skipped or undiscovered test a failure.

**What and why:** unittest alone exits 0 when a class is not discovered or a test is skipped. The Taskfile guard must catch both, so these runs call the task, not unittest.

### d1 — MainTests stops inheriting unittest.TestCase

**What and why:** With the base class gone, unittest discovers only `EvaluateTests` and still prints `OK` and exits 0. The pasted output shows both unittest's own `OK` and the guard's `::error::` line, proving the guard, not unittest, caught it.

**Pre-mutation gate:** `git -C "$S" diff --quiet -- scripts/test_fragment_policy.py` — exit=0 (clean).

**Mutation applied:** in `$S/scripts/test_fragment_policy.py`, `class MainTests(unittest.TestCase):` became `class MainTests:`.

**Confirmed applied:**

```
$ git -C "$S" diff --numstat -- scripts/test_fragment_policy.py
1	1	scripts/test_fragment_policy.py
$ git -C "$S" diff -U0 -- scripts/test_fragment_policy.py
@@ -323 +323 @@ class EvaluateTests(unittest.TestCase):
-class MainTests(unittest.TestCase):
+class MainTests:
```

**Observed result** (`task check:fragment-policy` from `$S`, exit 201):

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

----------------------------------------------------------------------
Ran 24 tests in 0.000s

OK
::error::check:fragment-policy: unittest ran 24 of 36 defined test methods
task: Failed to run task "check:fragment-policy": exit status 1
```

**Revert:** `git -C "$S" checkout -- scripts/test_fragment_policy.py`.

**Byte-clean proof:** `git -C "$S" diff --quiet` — exit=0 (clean).

### d2 — one test gains @unittest.skip

**What and why:** A skipped test counts toward `Ran N` and leaves unittest exiting 0, so the count comparison alone passes. The guard's exact-`OK` check must reject `OK (skipped=1)`.

**Pre-mutation gate:** `git -C "$S" diff --quiet -- scripts/test_fragment_policy.py` — exit=0 (clean).

**Mutation applied:** in `$S/scripts/test_fragment_policy.py`, `def test_gitkeep_is_not_a_fragment(self):` became `@unittest.skip("mutation d2")
    def test_gitkeep_is_not_a_fragment(self):`.

**Confirmed applied:**

```
$ git -C "$S" diff --numstat -- scripts/test_fragment_policy.py
1	0	scripts/test_fragment_policy.py
$ git -C "$S" diff -U0 -- scripts/test_fragment_policy.py
@@ -236,0 +237 @@ class EvaluateTests(unittest.TestCase):
+    @unittest.skip("mutation d2")
```

**Observed result** (`task check:fragment-policy` from `$S`, exit 201):

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
test_gitkeep_is_not_a_fragment (test_fragment_policy.EvaluateTests.test_gitkeep_is_not_a_fragment) ... skipped 'mutation d2'
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

OK (skipped=1)
::error::check:fragment-policy: result line is not exactly OK (skips are not executions)
task: Failed to run task "check:fragment-policy": exit status 1
```

**Revert:** `git -C "$S" checkout -- scripts/test_fragment_policy.py`.

**Byte-clean proof:** `git -C "$S" diff --quiet` — exit=0 (clean).

## Family (e) — GATE-04/D-10: the workflow is under the single-definition guard

**Test/guard:** `TestWorkflowRunBodiesInvokeTask` and `TestInScopeJobsPopulationMatchesDisk` (`internal/upgrade/taskfile_shape_test.go`).

**What and why:** The `fragment-required` workflow must be bound by the single-definition guard (its run body is exactly `task <target>`), and its registration in `inScopeJobs` must be bound to disk. Each mutation removes one half of that binding.

### e1 — the workflow run body becomes a direct python3 call

**What and why:** If the job body called the script directly instead of `task check:fragment-required`, the workflow would carry its own definition of the gate. The run-body guard must fail naming the workflow and job.

**Pre-mutation gate:** `git -C "$S" diff --quiet -- .github/workflows/fragment-required.yml` — exit=0 (clean).

**Mutation applied:** in `$S/.github/workflows/fragment-required.yml`, `run: task check:fragment-required` became `run: python3 scripts/fragment_policy.py`.

**Confirmed applied:**

```
$ git -C "$S" diff --numstat -- .github/workflows/fragment-required.yml
1	1	.github/workflows/fragment-required.yml
$ git -C "$S" diff -U0 -- .github/workflows/fragment-required.yml
@@ -65 +65 @@ jobs:
-        run: task check:fragment-required
+        run: python3 scripts/fragment_policy.py
```

**Observed result** (`GOTOOLCHAIN=go1.26.6 go test ./internal/upgrade/ -run '^TestWorkflowRunBodiesInvokeTask$' -count=1 -v` from `$S`, exit 1):

```
=== RUN   TestWorkflowRunBodiesInvokeTask
    taskfile_shape_test.go:1408: fragment-required.yml job "fragment-required" step "Evaluate fragment policy"'s run: body is not a single 'task <target>' call after stripping comments/blanks (D-01): "python3 scripts/fragment_policy.py"
--- FAIL: TestWorkflowRunBodiesInvokeTask (0.01s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/upgrade	0.200s
FAIL
```

**Revert:** `git -C "$S" checkout -- .github/workflows/fragment-required.yml`.

**Byte-clean proof:** `git -C "$S" diff --quiet` — exit=0 (clean).

### e2 — the inScopeJobs entry for fragment-required.yml is deleted

**What and why:** If the job were absent from `inScopeJobs`, the run-body guard would never look at it and every guard in that file would pass by omission. The population test binds the list to the jobs on disk.

**Pre-mutation gate:** `git -C "$S" diff --quiet -- internal/upgrade/taskfile_shape_test.go` — exit=0 (clean).

**Mutation applied:** in `$S/internal/upgrade/taskfile_shape_test.go`, `{Workflow: "fragment-required.yml", JobID: "fragment-required"},` became `(deleted)`.

**Confirmed applied:**

```
$ git -C "$S" diff --numstat -- internal/upgrade/taskfile_shape_test.go
0	1	internal/upgrade/taskfile_shape_test.go
$ git -C "$S" diff -U0 -- internal/upgrade/taskfile_shape_test.go
@@ -153 +152,0 @@ var inScopeJobs = []inScopeJob{
-	{Workflow: "fragment-required.yml", JobID: "fragment-required"},
```

**Observed result** (`GOTOOLCHAIN=go1.26.6 go test ./internal/upgrade/ -run '^TestInScopeJobsPopulationMatchesDisk$' -count=1 -v` from `$S`, exit 1):

```
=== RUN   TestInScopeJobsPopulationMatchesDisk
    taskfile_shape_test.go:1716: job(s) on disk in [ci.yml release-please.yml corpora.yml components-drift.yml fragment-required.yml] not present in inScopeJobs and not in usesOnlyJobExceptions: [fragment-required.yml/fragment-required] — a new CI job was added without registering it here
--- FAIL: TestInScopeJobsPopulationMatchesDisk (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/upgrade	0.208s
FAIL
```

**Revert:** `git -C "$S" checkout -- internal/upgrade/taskfile_shape_test.go`.

**Byte-clean proof:** `git -C "$S" diff --quiet` — exit=0 (clean).

## Teardown (Task 2)

```
$ git worktree remove --force "$S"
$ git worktree list
/Volumes/Code/github.com/seanb4t/codegraph-go                        565e3e25 [gsd/v0.15.0-milestone]
/Volumes/Code/herdr-worktrees/codegraph-go/worktree-brave-river-a240 2df78be0 [chore/octopusignore]
```

```
$ git status --porcelain -- scripts .github internal Taskfile.yml
(empty — the main checkout was never mutated.)
```

## Verdict (04-02 Task 2)

Ten mutations (a1, a2, b1, b2, c1, c2, d1, d2, e1, e2: two per family; the plan's prose says nine,
its family list holds ten, and all ten were run). Each was gated clean before it was applied,
confirmed applied by `git diff --numstat` and the `-U0` hunk, shown RED by the named test or guard,
and reverted byte-clean (`git diff --quiet` exit 0). Every one went RED:

- (a) the path rule: both mutations fail the path-rule tests (`test_path_rule_boundaries`,
  `test_shape_without_fragment_fails`, `test_rename_out_of_shipped_code_counts`,
  `test_test_files_and_testdata_are_not_shipped_code`);
- (b) the fragment and exemption rules: `test_fragment_must_be_added` and
  `test_exempt_marker_without_reason_does_not_exempt` fail;
- (c) fail-closed: `test_empty_file_list_fails_closed` and
  `test_main_fails_closed_on_truncated_file_list` fail;
- (d) the executed-count guard: with unittest itself printing `OK` (d1) and `OK (skipped=1)` (d2),
  `task check:fragment-policy` exits non-zero with `unittest ran 24 of 36 defined test methods` and
  `result line is not exactly OK`;
- (e) single-definition coverage: `TestWorkflowRunBodiesInvokeTask` fails naming
  `fragment-required.yml job "fragment-required"`, and `TestInScopeJobsPopulationMatchesDisk` fails
  naming `fragment-required.yml/fragment-required`.

No mutation reached a commit: only `(04-01)` commits ever touched the mutated files, and the
scratch worktree was removed. No test was adjusted to force a RED.
