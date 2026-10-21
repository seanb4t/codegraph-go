"""Unit suite for scripts/vuln_report.py.

Every case drives the real build-and-scan loop with canned (returncode, stdout,
stderr) runner results keyed by tool name. Nothing here runs go, govulncheck or
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
SCRATCH = "/scratch"
NAMES = [tool.name for tool in vuln_report.ROSTER]


def _binary_name(argv):
    """The tool a build (-o path) or scan (last argument) call names."""
    path = argv[argv.index("-o") + 1] if "-o" in argv else argv[-1]
    return path.rsplit("/", 1)[-1]


def _runner(builds=None, scans=None):
    """Canned results by tool name; everything not named builds and scans clean."""
    builds = builds or {}
    scans = scans or {}
    calls = []

    def run(argv):
        calls.append(list(argv))
        if argv[0] == SCANNER[0]:
            return scans.get(_binary_name(argv), (0, "", ""))
        return builds.get(_binary_name(argv), (0, "", ""))

    return run, calls


def _report(builds=None, scans=None):
    run, calls = _runner(builds, scans)
    out = io.StringIO()
    rc = vuln_report.run_report(SCANNER, SCRATCH, run, out)
    return rc, out.getvalue(), calls


class ClassifyTests(unittest.TestCase):
    def test_classify_table(self):
        cases = (
            (0, 0, vuln_report.CLEAN),
            (0, 3, vuln_report.DETECTED),
            (0, 1, vuln_report.SCAN_ERROR),
            (0, 2, vuln_report.SCAN_ERROR),
            (0, 127, vuln_report.SCAN_ERROR),
            (1, None, vuln_report.BUILD_ERROR),
            (2, 0, vuln_report.BUILD_ERROR),
        )
        for build_rc, scan_rc, want in cases:
            with self.subTest(build_rc=build_rc, scan_rc=scan_rc):
                self.assertEqual(vuln_report.classify(build_rc, scan_rc), want)

    def test_roster_has_nine_distinct_tools(self):
        self.assertEqual(len(vuln_report.ROSTER), 9)
        self.assertEqual(len(set(NAMES)), 9)
        for tool in vuln_report.ROSTER:
            with self.subTest(tool=tool.name):
                self.assertTrue(tool.modfile.startswith("go.tool"))
                self.assertIn("/", tool.package)


class LoopTests(unittest.TestCase):
    def test_all_clean(self):
        rc, out, _ = _report()
        self.assertEqual(rc, 0, out)
        self.assertIn(
            "vuln: built 9/9, scanned 9, build errors 0, detected 0, scan errors 0", out
        )
        self.assertNotIn("::warning::", out)
        self.assertNotIn("::error::", out)
        self.assertEqual(out.count(": CLEAN"), 9)

    def test_build_failure_is_reported_and_the_next_tool_is_still_scanned(self):
        rc, out, calls = _report(builds={"task": (1, "", "no such module")})
        self.assertEqual(rc, 0, out)
        self.assertIn("::warning::task: BUILD ERROR: go build exited 1", out)
        self.assertIn("no such module", out)
        self.assertIn(
            "vuln: built 8/9, scanned 8, build errors 1, detected 0, scan errors 0", out
        )
        scanned = [_binary_name(c) for c in calls if c[0] == SCANNER[0]]
        self.assertEqual(scanned, NAMES[1:])

    def test_scan_exit_three_is_detected(self):
        rc, out, _ = _report(scans={"goreleaser": (3, "Vulnerability #1: GO-2026-5932", "")})
        self.assertEqual(rc, 0, out)
        self.assertIn("::warning::goreleaser: DETECTED: govulncheck -mode=binary exited 3", out)
        self.assertIn("GO-2026-5932", out)
        self.assertIn(
            "vuln: built 9/9, scanned 9, build errors 0, detected 1, scan errors 0", out
        )

    def test_scan_exit_two_is_a_scan_error(self):
        rc, out, _ = _report(scans={"buf": (2, "", "vuln.go.dev unreachable")})
        self.assertEqual(rc, 0, out)
        self.assertIn("::warning::buf: SCAN ERROR: govulncheck -mode=binary exited 2", out)
        self.assertIn("vuln.go.dev unreachable", out)
        self.assertIn(
            "vuln: built 9/9, scanned 9, build errors 0, detected 0, scan errors 1", out
        )

    def test_every_tool_is_attempted_in_roster_order(self):
        _, _, calls = _report(
            builds={"actionlint": (1, "", "")}, scans={"changie": (3, "", "")}
        )
        built = [_binary_name(c) for c in calls if c[0] == "env"]
        self.assertEqual(built, NAMES)
        self.assertEqual(len(calls), 9 + 8)

    def test_all_builds_failing_still_exits_zero(self):
        rc, out, calls = _report(builds={name: (1, "", "proxy error") for name in NAMES})
        self.assertEqual(rc, 0, out)
        self.assertEqual(out.count("BUILD ERROR"), 9)
        self.assertIn(
            "vuln: built 0/9, scanned 0, build errors 9, detected 0, scan errors 0", out
        )
        self.assertEqual(len(calls), 9)

    def test_build_command_shape(self):
        tool = vuln_report.ROSTER[0]
        argv = vuln_report.build_command(tool, SCRATCH)
        self.assertEqual(argv[:4], ["env", "GOWORK=off", "go", "build"])
        self.assertIn(f"-modfile={tool.modfile}", argv)
        self.assertEqual(argv[-1], tool.package)
        self.assertEqual(argv[argv.index("-o") + 1], f"{SCRATCH}/{tool.name}")

    def test_scan_command_appends_the_binary(self):
        tool = vuln_report.ROSTER[1]
        self.assertEqual(
            vuln_report.scan_command(SCANNER, tool, SCRATCH),
            [*SCANNER, f"{SCRATCH}/{tool.name}"],
        )


class TallyTests(unittest.TestCase):
    def test_summary_line_format(self):
        totals = vuln_report.Tally(9, 7, 7, 2, 1, 1)
        self.assertEqual(
            totals.summary(),
            "vuln: built 7/9, scanned 7, build errors 2, detected 1, scan errors 1",
        )
        self.assertIsNone(totals.loop_defect())

    def test_missing_tool_is_a_loop_defect(self):
        self.assertEqual(vuln_report.Tally(9, 7, 7, 1, 0, 0).loop_defect(), "attempted 8 of 9 tools")

    def test_unscanned_binary_is_a_loop_defect(self):
        self.assertEqual(
            vuln_report.Tally(9, 9, 8, 0, 0, 0).loop_defect(), "scanned 8 of 9 built binaries"
        )

    def test_empty_loop_is_a_loop_defect(self):
        self.assertEqual(vuln_report.Tally(9, 0, 0, 0, 0, 0).loop_defect(), "attempted 0 of 9 tools")

    def test_tally_counts_each_outcome(self):
        tool = vuln_report.ROSTER[0]
        results = [
            vuln_report.Result(tool, vuln_report.CLEAN, 0, 0, ""),
            vuln_report.Result(tool, vuln_report.DETECTED, 0, 3, ""),
            vuln_report.Result(tool, vuln_report.SCAN_ERROR, 0, 2, ""),
            vuln_report.Result(tool, vuln_report.BUILD_ERROR, 1, None, ""),
        ]
        totals = vuln_report.tally(results, 4)
        self.assertEqual((totals.built, totals.scanned, totals.build_errors), (3, 3, 1))
        self.assertEqual((totals.detected, totals.scan_errors), (1, 1))


class MainTests(unittest.TestCase):
    def test_main_requires_a_scanner_after_double_dash(self):
        for argv in ([], ["--"], ["govulncheck"]):
            with self.subTest(argv=argv):
                self.assertEqual(vuln_report.main(argv=argv, run=_runner()[0], out=io.StringIO()), 2)

    def test_main_runs_the_loop_in_a_scratch_directory(self):
        run, calls = _runner()
        out = io.StringIO()
        rc = vuln_report.main(argv=["--", *SCANNER], run=run, out=out)
        self.assertEqual(rc, 0, out.getvalue())
        self.assertIn("vuln: built 9/9, scanned 9", out.getvalue())
        paths = [c[c.index("-o") + 1] if "-o" in c else c[-1] for c in calls]
        self.assertTrue(all("vuln-report-" in path for path in paths), paths)


if __name__ == "__main__":
    unittest.main()
