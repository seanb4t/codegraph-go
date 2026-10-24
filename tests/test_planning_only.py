"""Unit suite for scripts/planning_only.py.

Every case feeds literal inputs to classify(), decide() or main(): FileChange
tuples, environ dicts and canned (returncode, stdout, stderr) runner results.
Nothing here spawns gh, calls GitHub or runs a workflow.

`task check:ci-scripts` runs this file and asserts that unittest's executed count
equals the number of `def test_` methods defined here, so a skipped or
undiscovered case cannot pass (rule 84d1gfpywd).
"""

from __future__ import annotations

import io
import json
import os
import tempfile
import unittest

import planning_only
from fragment_policy import FileChange

REPO = "example/repo"
BEFORE = "a" * 40
AFTER = "b" * 40
PR_ENV = {"GITHUB_REPOSITORY": REPO, "GITHUB_EVENT_NAME": "pull_request", "PR_NUMBER": "7"}
PUSH_ENV = {
    "GITHUB_REPOSITORY": REPO,
    "GITHUB_EVENT_NAME": "push",
    "BEFORE_SHA": BEFORE,
    "AFTER_SHA": AFTER,
}

PLANNING_FILES = (
    FileChange("openspec/changes/x/proposal.md", "added"),
    FileChange("openspec/specs/y/spec.md", "modified"),
)


def _pr_results(paths, changed_files=None):
    """Canned runner for a pull request: metadata, then one page of files."""
    page = [
        {"filename": path, "status": status, **({"previous_filename": prev} if prev else {})}
        for path, status, prev in paths
    ]
    count = len(page) if changed_files is None else changed_files
    return {
        "meta": (0, json.dumps({"changed_files": count}), ""),
        "files": (0, json.dumps([page]), ""),
    }


def _push_result(paths):
    files = [
        {"filename": path, "status": status, **({"previous_filename": prev} if prev else {})}
        for path, status, prev in paths
    ]
    return (0, json.dumps({"files": files}), "")


def _runner(results):
    """Dispatch on the endpoint: compare, pulls/{n}/files, or pulls/{n} metadata."""
    calls = []

    def run(argv):
        calls.append(list(argv))
        endpoint = argv[-1]
        if "/compare/" in endpoint:
            return results["compare"]
        if "/files" in endpoint:
            return results["files"]
        return results["meta"]

    return run, calls


def _run_main(environ, results):
    """Run main() with a scratch GITHUB_OUTPUT; returns (rc, stdout, stderr, output)."""
    run, _ = _runner(results)
    out, err = io.StringIO(), io.StringIO()
    with tempfile.TemporaryDirectory() as scratch:
        output_path = os.path.join(scratch, "output")
        env = dict(environ, GITHUB_OUTPUT=output_path)
        rc = planning_only.main(environ=env, run=run, out=out, err=err)
        with open(output_path, encoding="utf-8") as handle:
            written = handle.read()
    return rc, out.getvalue(), err.getvalue(), written


class ClassifyTests(unittest.TestCase):
    def test_all_planning_paths(self):
        self.assertTrue(planning_only.classify(PLANNING_FILES))

    def test_mixed_list_is_not_planning_only(self):
        files = PLANNING_FILES + (FileChange("internal/cli/root.go", "modified"),)
        self.assertFalse(planning_only.classify(files))

    def test_empty_list_is_not_planning_only(self):
        self.assertFalse(planning_only.classify(()))

    def test_paths_that_merely_contain_openspec(self):
        for path in (
            "docs/openspec-notes.md",
            "internal/openspec/loader.go",
            "openspec",
            "openspec.md",
            "tools/openspec/x",
        ):
            with self.subTest(path=path):
                self.assertFalse(planning_only.is_planning_path(path))
                self.assertFalse(planning_only.classify((FileChange(path, "modified"),)))

    def test_rename_out_of_code_counts_the_previous_path(self):
        moved = FileChange("openspec/changes/x/notes.md", "renamed", "internal/cli/notes.md")
        self.assertFalse(planning_only.classify((moved,)))

    def test_rename_inside_planning_stays_planning(self):
        moved = FileChange("openspec/changes/y/a.md", "renamed", "openspec/changes/x/a.md")
        self.assertTrue(planning_only.classify((moved,)))

    def test_classify_does_not_mutate_input(self):
        files = list(PLANNING_FILES)
        planning_only.classify(files)
        self.assertEqual(files, list(PLANNING_FILES))


class PullRequestTests(unittest.TestCase):
    def test_all_planning_pr_is_true(self):
        results = _pr_results([(f.path, f.status, None) for f in PLANNING_FILES])
        rc, out, err, written = _run_main(PR_ENV, results)
        self.assertEqual(rc, 0)
        self.assertEqual(written, "planning_only=true\n")
        self.assertIn("planning-only: 2 changed files, planning_only=true", out)
        self.assertEqual(err, "")

    def test_mixed_pr_is_false(self):
        results = _pr_results(
            [("openspec/changes/x/proposal.md", "added", None), ("cmd/x/main.go", "modified", None)]
        )
        _, out, _, written = _run_main(PR_ENV, results)
        self.assertEqual(written, "planning_only=false\n")
        self.assertIn("2 changed files, planning_only=false", out)

    def test_rename_into_planning_is_false(self):
        results = _pr_results(
            [("openspec/changes/x/a.md", "renamed", "internal/cli/a.md")]
        )
        _, _, _, written = _run_main(PR_ENV, results)
        self.assertEqual(written, "planning_only=false\n")

    def test_short_listing_fails_open(self):
        results = _pr_results(
            [("openspec/changes/x/proposal.md", "added", None)], changed_files=2
        )
        rc, out, err, written = _run_main(PR_ENV, results)
        self.assertEqual(rc, 0)
        self.assertEqual(written, "planning_only=false\n")
        self.assertIn("fetched 1 of 2", err)
        self.assertIn("0 changed files, planning_only=false", out)

    def test_empty_listing_fails_open(self):
        results = _pr_results([])
        _, _, err, written = _run_main(PR_ENV, results)
        self.assertEqual(written, "planning_only=false\n")
        self.assertIn("empty changed-file list", err)

    def test_gh_error_fails_open(self):
        for result in ((1, "", "HTTP 502"), (127, "", "gh not found on PATH")):
            with self.subTest(returncode=result[0]):
                results = {"meta": result, "files": result}
                rc, _, err, written = _run_main(PR_ENV, results)
                self.assertEqual(rc, 0)
                self.assertEqual(written, "planning_only=false\n")
                self.assertIn("fail open", err)

    def test_malformed_pr_number_fails_open(self):
        env = dict(PR_ENV, PR_NUMBER="7; rm -rf /")
        _, _, err, written = _run_main(env, _pr_results([]))
        self.assertEqual(written, "planning_only=false\n")
        self.assertIn("PR_NUMBER", err)

    def test_pr_uses_the_count_checked_listing(self):
        run, calls = _runner(_pr_results([("openspec/a.md", "added", None)]))
        decision = planning_only.decide(PR_ENV, run)
        self.assertTrue(decision.planning_only)
        self.assertEqual(calls[0][-1], "repos/example/repo/pulls/7")
        self.assertIn("--paginate", calls[1])
        self.assertTrue(calls[1][-1].startswith("repos/example/repo/pulls/7/files"))


class PushTests(unittest.TestCase):
    def test_all_planning_push_is_true(self):
        results = {"compare": _push_result([("openspec/changes/x/tasks.md", "modified", None)])}
        rc, out, _, written = _run_main(PUSH_ENV, results)
        self.assertEqual(rc, 0)
        self.assertEqual(written, "planning_only=true\n")
        self.assertIn("1 changed files, planning_only=true", out)

    def test_push_with_code_is_false(self):
        results = {"compare": _push_result([("Taskfile.yml", "modified", None)])}
        _, _, _, written = _run_main(PUSH_ENV, results)
        self.assertEqual(written, "planning_only=false\n")

    def test_push_reads_previous_filename(self):
        results = {
            "compare": _push_result([("openspec/changes/x/a.md", "renamed", "internal/a.md")])
        }
        _, _, _, written = _run_main(PUSH_ENV, results)
        self.assertEqual(written, "planning_only=false\n")

    def test_all_zero_before_fails_open(self):
        env = dict(PUSH_ENV, BEFORE_SHA="0" * 40)
        run, calls = _runner({})
        decision = planning_only.decide(env, run)
        self.assertFalse(decision.planning_only)
        self.assertIn("no before commit", decision.reason)
        self.assertEqual(calls, [])

    def test_compare_cap_is_undecided(self):
        paths = [(f"openspec/changes/x/{i}.md", "added", None) for i in range(300)]
        results = {"compare": _push_result(paths)}
        _, _, err, written = _run_main(PUSH_ENV, results)
        self.assertEqual(written, "planning_only=false\n")
        self.assertIn("300 files", err)

    def test_compare_just_under_cap_is_decided(self):
        paths = [(f"openspec/changes/x/{i}.md", "added", None) for i in range(299)]
        run, _ = _runner({"compare": _push_result(paths)})
        self.assertTrue(planning_only.decide(PUSH_ENV, run).planning_only)

    def test_malformed_compare_fails_open(self):
        for stdout in ("not json", "[]", json.dumps({"files": [{"status": "added"}]})):
            with self.subTest(stdout=stdout):
                run, _ = _runner({"compare": (0, stdout, "")})
                decision = planning_only.decide(PUSH_ENV, run)
                self.assertFalse(decision.planning_only)
                self.assertIn("malformed", decision.reason)

    def test_malformed_sha_fails_open(self):
        env = dict(PUSH_ENV, AFTER_SHA="HEAD")
        run, calls = _runner({})
        decision = planning_only.decide(env, run)
        self.assertFalse(decision.planning_only)
        self.assertEqual(calls, [])

    def test_push_uses_the_compare_endpoint(self):
        run, calls = _runner({"compare": _push_result([("openspec/a.md", "added", None)])})
        planning_only.decide(PUSH_ENV, run)
        self.assertEqual(calls, [["gh", "api", f"repos/example/repo/compare/{BEFORE}...{AFTER}"]])


class EnvironmentTests(unittest.TestCase):
    def test_unknown_event_fails_open(self):
        env = dict(PR_ENV, GITHUB_EVENT_NAME="workflow_dispatch")
        run, calls = _runner({})
        decision = planning_only.decide(env, run)
        self.assertFalse(decision.planning_only)
        self.assertIn("workflow_dispatch", decision.reason)
        self.assertEqual(calls, [])

    def test_malformed_repository_fails_open(self):
        env = dict(PR_ENV, GITHUB_REPOSITORY="")
        decision = planning_only.decide(env, _runner({})[0])
        self.assertFalse(decision.planning_only)
        self.assertIn("GITHUB_REPOSITORY", decision.reason)

    def test_output_is_one_line_without_heredoc(self):
        results = _pr_results([("openspec/a.md", "added", None)])
        _, _, _, written = _run_main(PR_ENV, results)
        self.assertEqual(written.count("\n"), 1)
        self.assertNotIn("<<", written)

    def test_no_output_file_still_reports(self):
        run, _ = _runner(_pr_results([("openspec/a.md", "added", None)]))
        out = io.StringIO()
        rc = planning_only.main(environ=dict(PR_ENV), run=run, out=out, err=io.StringIO())
        self.assertEqual(rc, 0)
        self.assertIn("planning_only=true", out.getvalue())

    def test_main_never_exits_nonzero(self):
        for env, results in (
            (PR_ENV, {"meta": (1, "", "boom"), "files": (1, "", "boom")}),
            (PUSH_ENV, {"compare": (0, "not json", "")}),
            ({}, {}),
        ):
            with self.subTest(env=env.get("GITHUB_EVENT_NAME", "none")):
                rc, _, _, written = _run_main(env, results)
                self.assertEqual(rc, 0)
                self.assertEqual(written, "planning_only=false\n")

    def test_reason_cannot_start_a_workflow_command(self):
        run, _ = _runner({"meta": (1, "", "::error::injected\nline"), "files": (1, "", "")})
        out = io.StringIO()
        planning_only.main(environ=dict(PR_ENV), run=run, out=out, err=io.StringIO())
        self.assertNotIn("\n::error::", out.getvalue())
        self.assertNotIn("line", out.getvalue())

    def test_report_escapes_a_multiline_reason(self):
        out = io.StringIO()
        planning_only.report(planning_only.Decision(False, "a\n::error::b"), out, io.StringIO())
        self.assertEqual(out.getvalue().count("\n"), 1)
        self.assertIn("a%0A::error::b", out.getvalue())


if __name__ == "__main__":
    unittest.main()
