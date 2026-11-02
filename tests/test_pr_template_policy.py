"""Unit suite for scripts/pr_template_policy.py (GRD-15, WR-01).

Every case feeds literal inputs to the policy's functions. Nothing here calls
GitHub, gh or a workflow; how those behave is not ours to test.

`task check:ci-scripts` runs this file and asserts that unittest's executed
count equals the number of `def test_` methods defined here, so a skipped or
undiscovered case cannot pass (rule 84d1gfpywd).
"""

from __future__ import annotations

import contextlib
import io
import json
import os
import tempfile
import time
import unittest
from pathlib import Path
from unittest import mock

import pr_template_policy

REQUIRE_ISSUE_LINK_WORKFLOW = (
    Path(__file__).resolve().parent.parent / ".github" / "workflows" / "require-issue-link.yml"
)

LINEAR_BOUND_SECONDS = 1.0
MARKER = "pr-template-exempt"
# Each marker name with the pattern find_exemption takes for it.
MARKERS = (
    ("pr-template-exempt", pr_template_policy.EXEMPT_MARKER),
    ("changelog-exempt", pr_template_policy.CHANGELOG_EXEMPT_MARKER),
)


def _workflow_allowlist():
    """The (prefix set, exact set) in require-issue-link.yml's housekeeping `case`.

    Reads the workflow as text. Between `case "$f" in` and `esac`, an arm is a
    line `<pattern>[|<pattern>...]) ;;` with an empty body; the `*)` default arm
    has a body, so it is not an arm here. An item ending in `/*` is a prefix
    (the `*` dropped); any other item is an exact name.
    """
    lines = REQUIRE_ISSUE_LINK_WORKFLOW.read_text(encoding="utf-8").splitlines()
    start = next(i for i, line in enumerate(lines) if line.strip() == 'case "$f" in')
    end = next(i for i, line in enumerate(lines) if i > start and line.strip() == "esac")
    prefixes, exact = set(), set()
    for line in lines[start + 1 : end]:
        text = line.strip()
        if text.startswith("#") or not text.endswith(";;"):
            continue
        head = text[:-2].rstrip()
        if not head.endswith(")"):
            continue
        pattern = head[:-1]
        if " " in pattern:
            continue
        for item in pattern.split("|"):
            if item.endswith("/*"):
                prefixes.add(item[:-1])
            else:
                exact.add(item)
    return prefixes, exact


class HygieneAllowlistTests(unittest.TestCase):
    """GRD-15, D-05: the release PR's own paths are housekeeping in both hygiene gates."""

    def test_release_pr_paths_are_housekeeping(self):
        release_pr = [".changes/v0.15.0.md", ".changes/unreleased/Breaking-x.yaml", "CHANGELOG.md"]
        self.assertTrue(pr_template_policy.is_housekeeping(release_pr))
        for paths in ([".changesX/a.md"], [".Changes/a.md"]):
            with self.subTest(paths=paths):
                self.assertFalse(pr_template_policy.is_housekeeping(paths))

    def test_fragment_only_pr_is_housekeeping(self):
        self.assertTrue(
            pr_template_policy.is_housekeeping([".changes/unreleased/Fixes-x.yaml"])
        )

    def test_shipped_code_beside_changes_is_not_housekeeping(self):
        self.assertFalse(
            pr_template_policy.is_housekeeping(
                [".changes/unreleased/Fixes-x.yaml", "internal/cli/root.go"]
            )
        )

    def test_empty_path_list_is_not_housekeeping(self):
        self.assertFalse(pr_template_policy.is_housekeeping([]))

    def test_capped_path_list_is_not_housekeeping(self):
        limit = pr_template_policy.FILE_LIST_LIMIT
        self.assertTrue(pr_template_policy.is_housekeeping(["docs/a.md"] * (limit - 1)))
        self.assertFalse(pr_template_policy.is_housekeeping(["docs/a.md"] * limit))

    def test_exempt_sets_are_the_documented_sets(self):
        self.assertEqual(
            set(pr_template_policy.EXEMPT_PREFIXES),
            {".github/", "docs/", "openspec/", ".changes/"},
        )
        self.assertEqual(
            pr_template_policy.EXEMPT_EXACT,
            {
                "go.mod",
                "go.sum",
                "mise.lock",
                ".gitignore",
                "README.md",
                "CONTRIBUTING.md",
                "CODE_OF_CONDUCT.md",
                "SECURITY.md",
                "NOTICE",
                "LICENSE",
                "CHANGELOG.md",
            },
        )

    def test_hygiene_allowlists_mirror_require_issue_link(self):
        prefixes, exact = _workflow_allowlist()
        self.assertGreaterEqual(len(prefixes), 4, f"parsed prefixes: {sorted(prefixes)}")
        self.assertGreaterEqual(len(exact), 11, f"parsed exact names: {sorted(exact)}")
        mine_prefixes = set(pr_template_policy.EXEMPT_PREFIXES)
        mine_exact = set(pr_template_policy.EXEMPT_EXACT)
        self.assertEqual(
            prefixes,
            mine_prefixes,
            f"prefix side differs: only in workflow {sorted(prefixes - mine_prefixes)}, "
            f"only in pr_template_policy {sorted(mine_prefixes - prefixes)}",
        )
        self.assertEqual(
            exact,
            mine_exact,
            f"exact side differs: only in workflow {sorted(exact - mine_exact)}, "
            f"only in pr_template_policy {sorted(mine_exact - exact)}",
        )


class MarkerGrammarTests(unittest.TestCase):
    """WR-01: the exemption marker is linear and has one documented grammar."""

    def test_exempt_marker_is_linear_on_adversarial_bodies(self):
        bodies = (
            ("cubic shape", "<!-- " + MARKER + ":" + " " * 2500),
            ("quadratic shape", "<!-- " + MARKER + ": a" + " " * 60000),
        )
        for shape, body in bodies:
            with self.subTest(shape=shape):
                start = time.perf_counter()
                match = pr_template_policy.EXEMPT_MARKER.search(body)
                elapsed = time.perf_counter() - start
                self.assertLess(
                    elapsed,
                    LINEAR_BOUND_SECONDS,
                    f"{shape}: search took {elapsed:.2f}s on a {len(body)}-char body",
                )
                self.assertIsNone(match)

    def test_unterminated_marker_does_not_exempt(self):
        for name, marker in MARKERS:
            with self.subTest(marker=name):
                self.assertIsNone(pr_template_policy.find_exemption(f"<!-- {name}: reason", marker))

    def test_marker_reason_starting_with_dash_does_not_exempt(self):
        for name, marker in MARKERS:
            for body in (f"<!-- {name}: - -->", f"<!-- {name}: --x -->"):
                with self.subTest(body=body):
                    self.assertIsNone(pr_template_policy.find_exemption(body, marker))

    def test_marker_semantics(self):
        table = (
            ("<!-- {n}: docs only -->", "docs only"),
            ("<!-- {n}: -->", None),
            ("<!-- {n}:     -->", None),
            ("<!-- {N}: Mixed Case Name -->", "Mixed Case Name"),
            ("<!-- {n}: - -->", None),
            ("<!-- {n}: --x -->", None),
            ("<!-- {n}: reason with no closer", None),
            ("<!-- {n}: --> text <!-- {n}: second marker is ignored -->", None),
            ("<!-- {n}: a - b -->", "a - b"),
            ("text\r\n<!--\n{n}: spans\nlines -->", "spans\nlines"),
            ("<!-- {other}: reason -->", None),
            ("no marker at all", None),
        )
        for name, marker in MARKERS:
            other = next(n for n, _ in MARKERS if n != name)
            for template, expected in table:
                body = template.format(n=name, N=name.upper(), other=other)
                with self.subTest(body=body):
                    self.assertEqual(pr_template_policy.find_exemption(body, marker), expected)

    def test_changelog_exempt_prints_the_reason_on_one_line(self):
        table = (
            ("<!-- changelog-exempt: CI only -->", 0, "CI only\n"),
            ("<!-- changelog-exempt:\nCI\nonly -->", 0, "CI only\n"),
            ("<!-- changelog-exempt: -->", 1, ""),
            ("<!-- pr-template-exempt: docs -->", 1, ""),
            ("", 1, ""),
        )
        for body, want_status, want_out in table:
            with self.subTest(body=body):
                stdout = io.StringIO()
                with mock.patch.dict(os.environ, {"PR_BODY": body}), contextlib.redirect_stdout(stdout):
                    status = pr_template_policy.main(["changelog-exempt"])
                self.assertEqual((status, stdout.getvalue()), (want_status, want_out))


class EmitTests(unittest.TestCase):
    """D11: PR-body text in a reason cannot become a workflow command."""

    def test_emit_escapes_workflow_command(self):
        reason = "explicitly exempt: ok\n::error::injected\r\n::add-mask::x"
        with tempfile.TemporaryDirectory() as tmp:
            out = os.path.join(tmp, "output")
            stderr = io.StringIO()
            with mock.patch.dict(os.environ, {"GITHUB_OUTPUT": out}), contextlib.redirect_stderr(
                stderr
            ):
                pr_template_policy.emit("pass", reason)
            printed = stderr.getvalue()
            written = Path(out).read_text(encoding="utf-8").splitlines()

        self.assertEqual(printed.count("\n"), 1, printed)
        self.assertNotIn("\n::", printed)
        self.assertFalse(printed.startswith("::"))
        self.assertEqual(len(written), 2, written)
        payload = json.loads(written[1].removeprefix("result="))
        self.assertNotIn("\n", payload["reason"])
        self.assertNotIn("\r", payload["reason"])
        self.assertIn("injected", payload["reason"])

    def test_emit_strips_a_leading_command_prefix(self):
        stderr = io.StringIO()
        with mock.patch.dict(os.environ, {"GITHUB_OUTPUT": ""}), contextlib.redirect_stderr(stderr):
            pr_template_policy.emit("close", "::warning::spoofed")
        self.assertEqual(stderr.getvalue(), "close: warning::spoofed\n")


if __name__ == "__main__":
    unittest.main()
