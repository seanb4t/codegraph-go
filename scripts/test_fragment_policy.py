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

FRAGMENT_PATH = ".changes/unreleased/Fixes-20260926-120000.000000000.yaml"

BASE_SHAPE = (
    fragment_policy.FileChange("internal/cli/root.go", "modified"),
    fragment_policy.FileChange("cmd/codegraph/main.go", "modified"),
    fragment_policy.FileChange("docs/CLI-REFERENCE.md", "modified"),
    fragment_policy.FileChange("internal/cli/root_test.go", "modified"),
)

REPO_ENV = {"GITHUB_REPOSITORY": "example/repo", "PR_NUMBER": "7", "PR_BODY": ""}


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


if __name__ == "__main__":
    unittest.main()
