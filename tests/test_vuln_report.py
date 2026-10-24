"""Unit suite for scripts/vuln_report.py.

Every case drives the real locate-and-scan loop with canned (returncode, stdout,
stderr) runner results keyed by tool name. Nothing here runs mise, govulncheck or
task; how those behave is not ours to test, only what this loop reports about
their exit statuses.

`task check:ci-scripts` runs this file and asserts that unittest's executed count
equals the number of `def test_` methods defined here, so a skipped or
undiscovered case cannot pass (rule 84d1gfpywd).
"""

from __future__ import annotations

import io
import tomllib
import unittest
from pathlib import Path

import vuln_report
from vuln_report import Outcome

MISE_TOML = Path(__file__).resolve().parents[1] / "mise.toml"

SCANNER = ["fake-scanner", "-mode=binary"]
NAMES = [tool.name for tool in vuln_report.ROSTER]
COUNT = len(NAMES)


def _path(name):
    return f"/mise/installs/{name}/bin/{name}"


def _tool_name(argv):
    """The tool a locate call (last argument) or scan call (binary path) names."""
    return argv[-1].rsplit("/", 1)[-1]


def _runner(locates=None, scans=None):
    """Canned results by tool name; everything not named is found and scans clean."""
    locates = locates or {}
    scans = scans or {}
    calls = []

    def run(argv):
        calls.append(list(argv))
        name = _tool_name(argv)
        if argv[0] == SCANNER[0]:
            return scans.get(name, (0, "", ""))
        return locates.get(name, (0, _path(name) + "\n", ""))

    return run, calls


def _report(locates=None, scans=None):
    run, calls = _runner(locates, scans)
    out = io.StringIO()
    rc = vuln_report.run_report(SCANNER, run, out)
    return rc, out.getvalue(), calls


def _summary(found, scanned, not_found=0, detected=0, scan_errors=0):
    return (
        f"vuln: found {found}/{COUNT}, scanned {scanned}, not found {not_found}, "
        f"detected {detected}, scan errors {scan_errors}"
    )


class ClassifyTests(unittest.TestCase):
    def test_classify_table(self):
        cases = (
            (True, 0, Outcome.CLEAN),
            (True, 3, Outcome.DETECTED),
            (True, 1, Outcome.SCAN_ERROR),
            (True, 2, Outcome.SCAN_ERROR),
            (True, 127, Outcome.SCAN_ERROR),
            (False, None, Outcome.NOT_FOUND),
            (False, 0, Outcome.NOT_FOUND),
        )
        for found, scan_rc, want in cases:
            with self.subTest(found=found, scan_rc=scan_rc):
                self.assertEqual(vuln_report.classify(found, scan_rc), want)

    def test_roster_names_eleven_distinct_tools(self):
        self.assertEqual(COUNT, 11)
        self.assertEqual(len(set(NAMES)), COUNT)
        for name in ("changie", "cosign", "goreleaser", "syft"):
            with self.subTest(tool=name):
                self.assertIn(name, NAMES)

    def test_every_roster_tool_is_pinned_in_mise(self):
        tools = tomllib.loads(MISE_TOML.read_text(encoding="utf-8"))["tools"]
        commands = {key.rsplit("/", 1)[-1] if key.startswith("go:") else key for key in tools}
        for name in NAMES:
            with self.subTest(tool=name):
                self.assertIn(name, commands)


class LoopTests(unittest.TestCase):
    def test_all_clean(self):
        rc, out, _ = _report()
        self.assertEqual(rc, 0, out)
        self.assertIn(_summary(COUNT, COUNT), out)
        self.assertNotIn("::warning::", out)
        self.assertNotIn("::error::", out)
        self.assertEqual(out.count(": CLEAN"), COUNT)

    def test_missing_tool_is_reported_the_next_is_scanned_and_the_loop_fails(self):
        rc, out, calls = _report(locates={"task": (1, "", "task not installed")})
        self.assertEqual(rc, 1, out)
        self.assertIn("::warning::task: NOT FOUND: mise which exited 1", out)
        self.assertIn("task not installed", out)
        self.assertIn(_summary(COUNT - 1, COUNT - 1, not_found=1), out)
        self.assertIn(f"::error::vuln: loop defect: 1 of {COUNT} roster tools not found", out)
        scanned = [_tool_name(c) for c in calls if c[0] == SCANNER[0]]
        self.assertEqual(scanned, NAMES[1:])

    def test_an_empty_path_is_not_found(self):
        rc, out, calls = _report(locates={"syft": (0, "\n", "")})
        self.assertEqual(rc, 1, out)
        self.assertIn("::warning::syft: NOT FOUND: mise which printed no path; nothing was scanned", out)
        self.assertNotIn("syft", [_tool_name(c) for c in calls if c[0] == SCANNER[0]])

    def test_scan_exit_three_is_detected(self):
        rc, out, _ = _report(scans={"goreleaser": (3, "Vulnerability #1: GO-2026-5932", "")})
        self.assertEqual(rc, 0, out)
        self.assertIn("::warning::goreleaser: DETECTED: govulncheck -mode=binary exited 3", out)
        self.assertIn("GO-2026-5932", out)
        self.assertIn(_summary(COUNT, COUNT, detected=1), out)

    def test_scan_exit_two_is_a_scan_error(self):
        rc, out, _ = _report(scans={"buf": (2, "", "vuln.go.dev unreachable")})
        self.assertEqual(rc, 0, out)
        self.assertIn("::warning::buf: SCAN ERROR: govulncheck -mode=binary exited 2", out)
        self.assertIn("vuln.go.dev unreachable", out)
        self.assertIn(_summary(COUNT, COUNT, scan_errors=1), out)

    def test_every_tool_is_attempted_in_roster_order(self):
        _, _, calls = _report(
            locates={"actionlint": (1, "", "")}, scans={"changie": (3, "", "")}
        )
        located = [_tool_name(c) for c in calls if c[0] == "mise"]
        self.assertEqual(located, NAMES)
        self.assertEqual(len(calls), COUNT + COUNT - 1)

    def test_all_tools_missing_is_a_loop_defect(self):
        rc, out, calls = _report(locates={name: (1, "", "no mise") for name in NAMES})
        self.assertEqual(rc, 1, out)
        self.assertEqual(out.count("NOT FOUND"), COUNT)
        self.assertIn(_summary(0, 0, not_found=COUNT), out)
        self.assertEqual(len(calls), COUNT)

    def test_locate_command_shape(self):
        tool = vuln_report.ROSTER[0]
        self.assertEqual(vuln_report.locate_command(tool), ["mise", "which", tool.name])

    def test_the_located_path_is_scanned(self):
        _, _, calls = _report()
        scans = [c for c in calls if c[0] == SCANNER[0]]
        self.assertEqual(scans[1], [*SCANNER, _path(NAMES[1])])


class TallyTests(unittest.TestCase):
    def test_summary_line_format(self):
        totals = vuln_report.Tally(roster_size=9, found=9, scanned=9, not_found=0, detected=1, scan_errors=1)
        self.assertEqual(
            totals.summary(),
            "vuln: found 9/9, scanned 9, not found 0, detected 1, scan errors 1",
        )
        self.assertIsNone(totals.loop_defect())

    def test_missing_tool_is_a_loop_defect(self):
        self.assertEqual(
            vuln_report.Tally(roster_size=9, found=7, scanned=7, not_found=1, detected=0, scan_errors=0).loop_defect(),
            "attempted 8 of 9 tools",
        )

    def test_a_tool_not_found_is_a_loop_defect(self):
        self.assertEqual(
            vuln_report.Tally(roster_size=9, found=8, scanned=8, not_found=1, detected=0, scan_errors=0).loop_defect(),
            "1 of 9 roster tools not found",
        )

    def test_unscanned_binary_is_a_loop_defect(self):
        self.assertEqual(
            vuln_report.Tally(roster_size=9, found=9, scanned=8, not_found=0, detected=0, scan_errors=0).loop_defect(),
            "scanned 8 of 9 found binaries",
        )

    def test_empty_loop_is_a_loop_defect(self):
        self.assertEqual(
            vuln_report.Tally(roster_size=9, found=0, scanned=0, not_found=0, detected=0, scan_errors=0).loop_defect(),
            "attempted 0 of 9 tools",
        )

    def test_tally_counts_each_outcome(self):
        tool = vuln_report.ROSTER[0]
        results = [
            vuln_report.Result(tool, Outcome.CLEAN, 0, 0, ""),
            vuln_report.Result(tool, Outcome.DETECTED, 0, 3, ""),
            vuln_report.Result(tool, Outcome.SCAN_ERROR, 0, 2, ""),
            vuln_report.Result(tool, Outcome.NOT_FOUND, 1, None, ""),
        ]
        totals = vuln_report.tally(results, 4)
        self.assertEqual((totals.found, totals.scanned, totals.not_found), (3, 3, 1))
        self.assertEqual((totals.detected, totals.scan_errors), (1, 1))


class MainTests(unittest.TestCase):
    def test_main_requires_a_scanner_after_double_dash(self):
        for argv in ([], ["--"], ["govulncheck"]):
            with self.subTest(argv=argv):
                self.assertEqual(vuln_report.main(argv=argv, run=_runner()[0], out=io.StringIO()), 2)

    def test_main_runs_the_loop(self):
        run, _ = _runner()
        out = io.StringIO()
        rc = vuln_report.main(argv=["--", *SCANNER], run=run, out=out)
        self.assertEqual(rc, 0, out.getvalue())
        self.assertIn(f"vuln: found {COUNT}/{COUNT}, scanned {COUNT}", out.getvalue())


if __name__ == "__main__":
    unittest.main()
