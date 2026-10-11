"""Unit suite for scripts/fragment_policy.py (GATE-01, GATE-04).

D-01: every case feeds literal inputs to our verdict function or to main():
FileChange tuples, body strings, environ dicts and canned
(returncode, stdout, stderr) runner results. Nothing here builds a git
repository, spawns gh, calls GitHub, runs a workflow or runs changie; how those
tools behave is not ours to test.

`task check:fragment-policy` runs this file and asserts that unittest's executed
count equals the number of `def test_` methods defined here, so a skipped or
undiscovered case cannot pass (rule 84d1gfpywd).

Names that a later task adds to fragment_policy are reached as module
attributes, so a missing name is an assertion failure and never an ImportError.
"""

from __future__ import annotations

import io
import json
import unittest

import fragment_policy
import pr_template_policy

FRAGMENT_PATH = ".changes/unreleased/Fixes-20260926-120000.000000000.yaml"

BASE_SHAPE = (
    fragment_policy.FileChange("internal/cli/root.go", "modified"),
    fragment_policy.FileChange("cmd/codegraph/main.go", "modified"),
    fragment_policy.FileChange("docs/CLI-REFERENCE.md", "modified"),
    fragment_policy.FileChange("internal/cli/root_test.go", "modified"),
)

REPO_ENV = {"GITHUB_REPOSITORY": "example/repo", "PR_NUMBER": "7", "PR_BODY": ""}

EXEMPT_REASON = "comment-only change, no behavior"
EXEMPT_BODY = f"Refactor only.\n<!-- changelog-exempt: {EXEMPT_REASON} -->"


def _change(path, status="modified", previous_path=None):
    return fragment_policy.FileChange(path, status, previous_path)


def _files_result(*pages):
    """Canned stdout of `gh api --paginate --slurp .../files`: one array of pages."""
    body = [
        [{"filename": path, "status": status} for path, status in page]
        for page in pages
    ]
    return (0, json.dumps(body), "")


def _meta_result(changed_files):
    return (0, json.dumps({"changed_files": changed_files}), "")


def _run_main(environ, files_result, meta_result=None):
    """Run main() against canned runner results; never raises.

    Returns (return code, output, recorded argv lists). A crash inside main
    comes back as a None return code carrying the exception's repr, so it
    reads as an assertion failure rather than an ERROR.
    """
    calls = []

    if meta_result is None:
        count = 0
        if files_result[0] == 0:
            try:
                count = sum(len(page) for page in json.loads(files_result[1]))
            except (ValueError, TypeError):
                count = 0
        meta_result = _meta_result(count)

    def runner(argv):
        calls.append(list(argv))
        return files_result if "/files" in argv[-1] else meta_result

    out = io.StringIO()
    try:
        rc = fragment_policy.main(environ=dict(environ), run=runner, out=out)
    except Exception as exc:  # noqa: BLE001 - a crash must be a FAIL, not an ERROR
        return None, out.getvalue() + repr(exc), calls
    return rc, out.getvalue(), calls


class EvaluateTests(unittest.TestCase):
    def test_shape_without_fragment_fails(self):
        verdict = fragment_policy.evaluate(BASE_SHAPE, "")
        self.assertFalse(verdict.ok)
        self.assertEqual(verdict.code, "missing-fragment")
        self.assertEqual(
            verdict.shipped, ("cmd/codegraph/main.go", "internal/cli/root.go")
        )

    def test_identical_shape_with_fragment_passes(self):
        files = BASE_SHAPE + (_change(FRAGMENT_PATH, "added"),)
        verdict = fragment_policy.evaluate(files, "")
        self.assertTrue(verdict.ok)
        self.assertEqual(verdict.code, "has-fragment")

    def test_no_shipped_code_passes_without_fragment(self):
        files = [
            _change(path)
            for path in (
                ".planning/x.md",
                "docs/a.md",
                "scripts/fragment_policy.py",
                ".github/workflows/ci.yml",
                "Taskfile.yml",
            )
        ]
        verdict = fragment_policy.evaluate(files, "")
        self.assertTrue(verdict.ok)
        self.assertEqual(verdict.code, "no-shipped-code")
        self.assertEqual(verdict.shipped, ())

    def test_test_files_and_testdata_are_not_shipped_code(self):
        self.assertTrue(fragment_policy.is_shipped_code("internal/cli/root.go"))
        paths = (
            "internal/cli/root_test.go",
            "internal/cli/testdata/golden/x.txt",
            "cmd/codegraph/testdata/fixture.go",
        )
        for path in paths:
            with self.subTest(path=path):
                self.assertFalse(fragment_policy.is_shipped_code(path))
        verdict = fragment_policy.evaluate([_change(p) for p in paths], "")
        self.assertTrue(verdict.ok)
        self.assertEqual(verdict.code, "no-shipped-code")


    # GATE-02: the exemption marker.

    def test_exempt_marker_with_reason_passes(self):
        verdict = fragment_policy.evaluate(BASE_SHAPE, EXEMPT_BODY)
        self.assertTrue(verdict.ok)
        self.assertEqual(verdict.code, "exempt")
        self.assertIn(EXEMPT_REASON, verdict.message)

    def test_exempt_marker_is_case_insensitive(self):
        body = "<!-- CHANGELOG-EXEMPT: internal rename -->"
        verdict = fragment_policy.evaluate(BASE_SHAPE, body)
        self.assertTrue(verdict.ok)
        self.assertEqual(verdict.code, "exempt")

    def test_exempt_marker_without_reason_does_not_exempt(self):
        bodies = (
            "<!-- changelog-exempt: -->",
            "<!-- changelog-exempt:    -->",
            "<!--changelog-exempt:-->",
        )
        for body in bodies:
            with self.subTest(body=body):
                verdict = fragment_policy.evaluate(BASE_SHAPE, body)
                self.assertFalse(verdict.ok)
                self.assertEqual(verdict.code, "missing-fragment")

    def test_other_marker_names_do_not_exempt(self):
        bodies = (
            "<!-- pr-template-exempt: x -->",
            "<!-- pr-issue-exempt: x -->",
            "changelog-exempt: x",
        )
        for body in bodies:
            with self.subTest(body=body):
                verdict = fragment_policy.evaluate(BASE_SHAPE, body)
                self.assertFalse(verdict.ok)
                self.assertEqual(verdict.code, "missing-fragment")

    def test_marker_grammar_mirrors_pr_template_exempt(self):
        self.assertTrue(hasattr(fragment_policy, "CHANGELOG_EXEMPT_MARKER"))
        mine = fragment_policy.CHANGELOG_EXEMPT_MARKER
        theirs = pr_template_policy.EXEMPT_MARKER
        self.assertEqual(
            mine.pattern, theirs.pattern.replace("pr-template-exempt", "changelog-exempt")
        )
        self.assertNotEqual(mine.pattern, theirs.pattern)
        self.assertEqual(mine.flags, theirs.flags)

    # GATE-03: the three states in which the change cannot be seen.

    def test_empty_file_list_fails_closed(self):
        verdict = fragment_policy.evaluate([], EXEMPT_BODY)
        self.assertFalse(verdict.ok)
        self.assertEqual(verdict.code, "empty-file-list")

    def test_api_error_fails_closed(self):
        files = BASE_SHAPE + (_change(FRAGMENT_PATH, "added"),)
        verdict = fragment_policy.evaluate(files, EXEMPT_BODY, api_error="HTTP 502")
        self.assertFalse(verdict.ok)
        self.assertEqual(verdict.code, "api-error")
        self.assertIn("HTTP 502", verdict.message)

    def test_unreadable_body_fails_closed(self):
        files = BASE_SHAPE + (_change(FRAGMENT_PATH, "added"),)
        verdict = fragment_policy.evaluate(files, None)
        self.assertFalse(verdict.ok)
        self.assertEqual(verdict.code, "body-unreadable")

    def test_empty_body_is_readable(self):
        with_fragment = BASE_SHAPE + (_change(FRAGMENT_PATH, "added"),)
        self.assertEqual(fragment_policy.evaluate(with_fragment, "").code, "has-fragment")
        self.assertEqual(fragment_policy.evaluate(BASE_SHAPE, "").code, "missing-fragment")

    def test_fail_closed_messages_are_distinct(self):
        verdicts = (
            fragment_policy.evaluate(BASE_SHAPE, "", api_error="HTTP 502"),
            fragment_policy.evaluate([], ""),
            fragment_policy.evaluate(BASE_SHAPE, None),
        )
        for verdict in verdicts:
            self.assertFalse(verdict.ok)
        self.assertEqual(len({v.code for v in verdicts}), 3)
        self.assertEqual(len({v.message for v in verdicts}), 3)

    def test_fail_closed_precedence(self):
        self.assertEqual(
            fragment_policy.evaluate([], None, api_error="x").code, "api-error"
        )
        self.assertEqual(fragment_policy.evaluate([], None).code, "empty-file-list")
        self.assertEqual(
            fragment_policy.evaluate(BASE_SHAPE, None).code, "body-unreadable"
        )

    # Fragment and path rules.

    def test_fragment_must_be_added(self):
        for status in ("modified", "renamed", "removed", "copied", "changed"):
            with self.subTest(status=status):
                files = BASE_SHAPE + (_change(".changes/unreleased/x.yaml", status),)
                verdict = fragment_policy.evaluate(files, "")
                self.assertFalse(verdict.ok)
                self.assertEqual(verdict.code, "missing-fragment")

    def test_gitkeep_is_not_a_fragment(self):
        files = BASE_SHAPE + (_change(".changes/unreleased/.gitkeep", "added"),)
        verdict = fragment_policy.evaluate(files, "")
        self.assertFalse(verdict.ok)
        self.assertEqual(verdict.code, "missing-fragment")

    def test_status_comparison_ignores_case(self):
        files = BASE_SHAPE + (_change(FRAGMENT_PATH, "ADDED"),)
        verdict = fragment_policy.evaluate(files, "")
        self.assertTrue(verdict.ok)
        self.assertEqual(verdict.code, "has-fragment")

    def test_path_rule_boundaries(self):
        shipped = (
            "cmd/codegraph/main.go",
            "internal/cli/root.go",
            "internal/testdata.go",
            "internal/x/testdata",
            "internal/web/static/app.js",
            "internal/x/y_test.go.orig",
        )
        not_shipped = (
            "cmdx/foo.go",
            "internalx/a.go",
            "Internal/a.go",
            "cmd",
            "pkg/internal/a.go",
            "internal/x/testdata/y.go",
            "internal/x/y_test.go",
            ".changes/unreleased/a.yaml",
        )
        for path in shipped:
            with self.subTest(shipped=path):
                self.assertTrue(fragment_policy.is_shipped_code(path))
        for path in not_shipped:
            with self.subTest(not_shipped=path):
                self.assertFalse(fragment_policy.is_shipped_code(path))

    def test_rename_out_of_shipped_code_counts(self):
        files = [_change("pkg/root.go", "renamed", previous_path="internal/cli/root.go")]
        verdict = fragment_policy.evaluate(files, "")
        self.assertFalse(verdict.ok)
        self.assertEqual(verdict.code, "missing-fragment")
        self.assertEqual(verdict.shipped, ("internal/cli/root.go",))

    def test_removed_shipped_file_counts(self):
        verdict = fragment_policy.evaluate(
            [_change("internal/cli/renamed.go", "removed")], ""
        )
        self.assertFalse(verdict.ok)
        self.assertEqual(verdict.code, "missing-fragment")

    def test_fragment_outside_unreleased_does_not_count(self):
        for path in (".changes/v0.15.0.md", ".changes/unreleasedX/a.yaml"):
            with self.subTest(path=path):
                files = BASE_SHAPE + (_change(path, "added"),)
                verdict = fragment_policy.evaluate(files, "")
                self.assertFalse(verdict.ok)
                self.assertEqual(verdict.code, "missing-fragment")

    # Ordering and purity.

    def test_verdict_ignores_file_order(self):
        files = list(BASE_SHAPE)
        self.assertEqual(
            fragment_policy.evaluate(files, ""),
            fragment_policy.evaluate(list(reversed(files)), ""),
        )
        doubled = [
            _change("internal/a.go"),
            _change("internal/a.go"),
            _change("internal/a.go", "renamed", previous_path="internal/a.go"),
        ]
        self.assertEqual(
            fragment_policy.evaluate(doubled, "").shipped, ("internal/a.go",)
        )

    def test_evaluate_does_not_mutate_input(self):
        files = list(BASE_SHAPE)
        before = list(files)
        first = fragment_policy.evaluate(files, "")
        second = fragment_policy.evaluate(files, "")
        self.assertEqual(files, before)
        self.assertEqual(first, second)


class MainTests(unittest.TestCase):
    def test_main_fails_and_names_paths_when_fragment_missing(self):
        rc, output, _ = _run_main(
            REPO_ENV, _files_result([("internal/cli/root.go", "modified")])
        )
        self.assertEqual(rc, 1, output)
        self.assertIn("::error::fragment-required: FAIL (missing-fragment)", output)
        self.assertIn("internal/cli/root.go", output)
        self.assertIn("task changie -- new", output)
        self.assertIn("changelog-exempt", output)

    def test_main_passes_when_fragment_added(self):
        rc, output, _ = _run_main(
            REPO_ENV,
            _files_result(
                [("internal/cli/root.go", "modified"), (FRAGMENT_PATH, "added")]
            ),
        )
        self.assertEqual(rc, 0, output)
        self.assertIn("fragment-required: PASS (has-fragment)", output)

    def test_main_fails_closed_when_gh_exits_nonzero(self):
        results = (
            (1, "", "HTTP 502"),
            (124, "", "gh timed out after 120s"),
            (127, "", "gh not found on PATH"),
        )
        for result in results:
            with self.subTest(returncode=result[0]):
                rc, output, _ = _run_main(REPO_ENV, result, _meta_result(1))
                self.assertEqual(rc, 1, output)
                self.assertIn("FAIL (api-error)", output)
                self.assertNotIn("PASS", output)

    def test_escape_data_encodes_command_characters(self):
        self.assertEqual(fragment_policy.escape_data("50%\r\nx"), "50%25%0D%0Ax")

    def test_untrusted_paths_cannot_start_workflow_commands(self):
        hostile = "internal/a\n::warning::x.go"
        rc, output, _ = _run_main(REPO_ENV, _files_result([(hostile, "modified")]))
        self.assertEqual(rc, 1, output)
        self.assertIn("internal/a%0A::warning::x.go", output)
        for line in output.splitlines():
            with self.subTest(line=line):
                self.assertFalse(line.startswith("::warning::"))
                if line.startswith("::"):
                    self.assertTrue(
                        line.startswith("::error::fragment-required:")
                        or line.startswith("::notice::fragment-required:")
                    )


    def test_main_fails_closed_on_truncated_file_list(self):
        files = _files_result(
            [("internal/a.go", "modified"), (FRAGMENT_PATH, "added")]
        )
        rc, output, _ = _run_main(REPO_ENV, files, _meta_result(3))
        self.assertEqual(rc, 1, output)
        self.assertIn("FAIL (api-error)", output)
        self.assertIn("fetched 2 of 3", output)
        self.assertNotIn("PASS", output)

    def test_main_fails_closed_on_malformed_json(self):
        rc, output, _ = _run_main(REPO_ENV, (0, "not json", ""), _meta_result(1))
        self.assertIsNotNone(rc, output)
        self.assertEqual(rc, 1, output)
        self.assertIn("FAIL (api-error)", output)

    def test_main_fails_closed_without_pr_body(self):
        environ = {k: v for k, v in REPO_ENV.items() if k != "PR_BODY"}
        files = _files_result(
            [("internal/a.go", "modified"), (FRAGMENT_PATH, "added")]
        )
        rc, output, _ = _run_main(environ, files)
        self.assertEqual(rc, 1, output)
        self.assertIn("FAIL (body-unreadable)", output)

    def test_main_rejects_malformed_pr_context(self):
        files = _files_result([("internal/a.go", "modified")])
        cases = (
            {"PR_NUMBER": ""},
            {"PR_NUMBER": "0"},
            {"PR_NUMBER": "07"},
            {"PR_NUMBER": "7; rm -rf /"},
            {"GITHUB_REPOSITORY": ""},
            {"GITHUB_REPOSITORY": "noslash"},
        )
        for override in cases:
            with self.subTest(override=override):
                environ = {**REPO_ENV, **override}
                rc, output, calls = _run_main(environ, files)
                self.assertEqual(rc, 1, output)
                self.assertIn("FAIL (api-error)", output)
                self.assertEqual(calls, [])

    def test_main_flattens_paginated_pages(self):
        files = _files_result(
            [("internal/a.go", "modified")], [(FRAGMENT_PATH, "added")]
        )
        rc, output, _ = _run_main(REPO_ENV, files, _meta_result(2))
        self.assertEqual(rc, 0, output)
        self.assertIn("PASS (has-fragment)", output)

    def test_main_uses_paginated_rest_listing(self):
        files = _files_result([("docs/a.md", "modified")])
        rc, output, calls = _run_main(REPO_ENV, files, _meta_result(1))
        self.assertEqual(rc, 0, output)
        self.assertIn(["gh", "api", "repos/example/repo/pulls/7"], calls)
        self.assertIn(
            [
                "gh",
                "api",
                "--paginate",
                "--slurp",
                "repos/example/repo/pulls/7/files?per_page=100",
            ],
            calls,
        )

    def test_main_reports_exemption_reason_as_notice(self):
        environ = {**REPO_ENV, "PR_BODY": EXEMPT_BODY}
        files = _files_result([("internal/a.go", "modified")])
        rc, output, _ = _run_main(environ, files)
        self.assertEqual(rc, 0, output)
        self.assertIn(
            f"::notice::fragment-required: exempt: {EXEMPT_REASON}",
            output.splitlines(),
        )


if __name__ == "__main__":
    unittest.main()
