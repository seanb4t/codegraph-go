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
import unittest

import vuln_report

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
            (0, 0, vuln_report.CLEAN),
            (0, 3, vuln_report.DETECTED),
            (0, 1, vuln_report.SCAN_ERROR),
            (0, 2, vuln_report.SCAN_ERROR),
            (0, 127, vuln_report.SCAN_ERROR),
            (1, None, vuln_report.NOT_FOUND),
            (127, 0, vuln_report.NOT_FOUND),
        )
        for locate_rc, scan_rc, want in cases:
            with self.subTest(locate_rc=locate_rc, scan_rc=scan_rc):
                self.assertEqual(vuln_report.classify(locate_rc, scan_rc), want)

    def test_roster_names_eleven_distinct_tools(self):
        self.assertEqual(COUNT, 11)
        self.assertEqual(len(set(NAMES)), COUNT)
        for name in ("changie", "cosign", "goreleaser", "syft"):
            with self.subTest(tool=name):
                self.assertIn(name, NAMES)


class LoopTests(unittest.TestCase):
    def test_all_clean(self):
        rc, out, _ = _report()
        self.assertEqual(rc, 0, out)
        self.assertIn(_summary(COUNT, COUNT), out)
        self.assertNotIn("::warning::", out)
        self.assertNotIn("::error::", out)
        self.assertEqual(out.count(": CLEAN"), COUNT)

    def test_missing_tool_is_reported_and_the_next_tool_is_still_scanned(self):
        rc, out, calls = _report(locates={"task": (1, "", "task not installed")})
        self.assertEqual(rc, 0, out)
        self.assertIn("::warning::task: NOT FOUND: mise which exited 1", out)
        self.assertIn("task not installed", out)
        self.assertIn(_summary(COUNT - 1, COUNT - 1, not_found=1), out)
        scanned = [_tool_name(c) for c in calls if c[0] == SCANNER[0]]
        self.assertEqual(scanned, NAMES[1:])

    def test_an_empty_path_is_not_found(self):
        rc, out, calls = _report(locates={"syft": (0, "\n", "")})
        self.assertEqual(rc, 0, out)
        self.assertIn("::warning::syft: NOT FOUND", out)
        self.assertIn("mise which syft printed no path", out)
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

    def test_all_tools_missing_still_exits_zero(self):
        rc, out, calls = _report(locates={name: (1, "", "no mise") for name in NAMES})
        self.assertEqual(rc, 0, out)
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
        totals = vuln_report.Tally(9, 7, 7, 2, 1, 1)
        self.assertEqual(
            totals.summary(),
            "vuln: found 7/9, scanned 7, not found 2, detected 1, scan errors 1",
        )
        self.assertIsNone(totals.loop_defect())

    def test_missing_tool_is_a_loop_defect(self):
        self.assertEqual(vuln_report.Tally(9, 7, 7, 1, 0, 0).loop_defect(), "attempted 8 of 9 tools")

    def test_unscanned_binary_is_a_loop_defect(self):
        self.assertEqual(
            vuln_report.Tally(9, 9, 8, 0, 0, 0).loop_defect(), "scanned 8 of 9 found binaries"
        )

    def test_empty_loop_is_a_loop_defect(self):
        self.assertEqual(vuln_report.Tally(9, 0, 0, 0, 0, 0).loop_defect(), "attempted 0 of 9 tools")

    def test_tally_counts_each_outcome(self):
        tool = vuln_report.ROSTER[0]
        results = [
            vuln_report.Result(tool, vuln_report.CLEAN, 0, 0, ""),
            vuln_report.Result(tool, vuln_report.DETECTED, 0, 3, ""),
            vuln_report.Result(tool, vuln_report.SCAN_ERROR, 0, 2, ""),
            vuln_report.Result(tool, vuln_report.NOT_FOUND, 1, None, ""),
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
