"""Unit suite for scripts/mutation_check.py.

Every case drives the real baseline-then-mutate loop against files in a
temporary directory, with canned (returncode, output) runner results. Nothing
here runs go; how go test behaves is not ours to test, only what this loop does
with its exit status and output.

`task check:ci-scripts` runs this file and asserts that unittest's executed count
equals the number of `def test_` methods defined here, so a skipped or
undiscovered case cannot pass (rule 84d1gfpywd).
"""

from __future__ import annotations

import io
import json
import tempfile
import unittest
from pathlib import Path

import mutation_check
from mutation_check import Outcome

SOURCE = "package demo\n\nfunc Limit() int { return 10 }\n"
PASSED = (0, "=== RUN   TestLimit\n--- PASS: TestLimit (0.00s)\nPASS\n")
CAUGHT = (1, "=== RUN   TestLimit\n--- FAIL: TestLimit (0.00s)\nFAIL\n")


def _mutant(**overrides):
    entry = {
        "name": "M1",
        "file": "demo/demo.go",
        "old": "return 10",
        "new": "return 11",
        "package": "./demo/",
        "tests": ["TestLimit"],
    }
    entry.update(overrides)
    return entry


class _Tree:
    """A temporary repository root holding demo/demo.go."""

    def __enter__(self):
        self._dir = tempfile.TemporaryDirectory()
        self.root = Path(self._dir.name)
        (self.root / "demo").mkdir()
        (self.root / "demo" / "demo.go").write_text(SOURCE, encoding="utf-8")
        self.work = self.root / "scratch" / "000"
        return self

    def __exit__(self, *exc):
        self._dir.cleanup()


def _runner(baseline=PASSED, mutant=CAUGHT):
    """Canned results: baseline runs carry no -overlay, mutant runs do."""
    calls = []

    def run(argv):
        calls.append(list(argv))
        if any(a.startswith("-overlay=") for a in argv):
            return mutant
        return baseline

    return run, calls


def _overlaid(calls):
    return [c for c in calls if any(a.startswith("-overlay=") for a in c)]


class LoadTests(unittest.TestCase):
    def test_a_valid_list_loads(self):
        mutants = mutation_check.load_mutants([_mutant(), _mutant(name="M2", new="", tests=["TestA", "TestB"])])
        self.assertEqual([m.name for m in mutants], ["M1", "M2"])
        self.assertEqual(mutants[1].tests, ("TestA", "TestB"))
        self.assertEqual(mutants[1].new, "")

    def test_malformed_input_is_rejected(self):
        cases = {
            "not a list": {"name": "M1"},
            "empty list": [],
            "not an object": ["M1"],
            "missing key": [{k: v for k, v in _mutant().items() if k != "package"}],
            "no tests": [_mutant(tests=[])],
            "blank test name": [_mutant(tests=[""])],
            "subtest name": [_mutant(tests=["TestLimit/sub"])],
            "no change": [_mutant(new="return 10")],
            "non-string old": [_mutant(old=10)],
            "empty name": [_mutant(name="")],
            "empty file": [_mutant(file="")],
            "flag-shaped package": [_mutant(package="-exec=evil")],
            "duplicate name": [_mutant(), _mutant(old="Limit()", new="Limit2()")],
        }
        for label, data in cases.items():
            with self.subTest(label):
                with self.assertRaises(ValueError):
                    mutation_check.load_mutants(data)


class CommandTests(unittest.TestCase):
    def test_run_pattern_is_anchored_and_escaped(self):
        self.assertEqual(mutation_check.run_pattern(["TestA", "Test.B"]), r"^(TestA|Test\.B)$")

    def test_test_command_shape(self):
        self.assertEqual(
            mutation_check.test_command("./demo/", ["TestLimit"], "/tmp/o.json"),
            ["go", "test", "./demo/", "-overlay=/tmp/o.json", "-run", "^(TestLimit)$", "-count=1", "-v", "-timeout=5m"],
        )
        self.assertNotIn("-overlay", " ".join(mutation_check.test_command("./demo/", ["TestLimit"])))


class ClassifyTests(unittest.TestCase):
    def test_classify_table(self):
        cases = (
            (0, "ok  demo 0.1s\n", Outcome.SURVIVED, ""),
            (1, "--- FAIL: TestLimit (0.00s)\nFAIL\n", Outcome.CAUGHT, "TestLimit"),
            (1, "    --- FAIL: TestLimit/sub (0.00s)\n--- FAIL: TestLimit (0.00s)\n", Outcome.CAUGHT, "TestLimit/sub, TestLimit"),
            (1, "--- FAIL: TestA\n--- FAIL: TestA\n", Outcome.CAUGHT, "TestA"),
            (1, "demo.go:3:9: declared and not used: x\nFAIL\tdemo [build failed]\n", Outcome.BUILD_FAIL, "demo.go:3:9: declared and not used: x\nFAIL\tdemo [build failed]"),
            (1, "# demo\nvet: demo.go:3: bad\nFAIL\tdemo [setup failed]\n", Outcome.BUILD_FAIL, "# demo\nvet: demo.go:3: bad\nFAIL\tdemo [setup failed]"),
            (2, "panic: test timed out after 5m0s\nFAIL\tdemo\n", Outcome.CRASHED, "panic: test timed out after 5m0s\nFAIL\tdemo"),
            (1, "exit status 1\nFAIL\tdemo\n", Outcome.CRASHED, "exit status 1\nFAIL\tdemo"),
        )
        for rc, output, want, detail in cases:
            with self.subTest(rc=rc, output=output):
                self.assertEqual(mutation_check.classify(rc, output), (want, detail))

    def test_baseline_failure_table(self):
        tests = ["TestLimit", "TestOther"]
        both = "--- PASS: TestLimit (0.00s)\n--- PASS: TestOther (0.00s)\nPASS\n"
        self.assertIsNone(mutation_check.baseline_failure(0, both, tests))
        self.assertIn("exited 1", mutation_check.baseline_failure(1, "--- FAIL: TestLimit\n", tests))
        missing = mutation_check.baseline_failure(0, "--- PASS: TestLimit (0.00s)\nok\n", tests)
        self.assertIn("TestOther", missing)
        self.assertNotIn("TestLimit,", missing)
        self.assertIn("TestLimit", mutation_check.baseline_failure(0, "ok  demo [no tests to run]\n", ["TestLimit"]))


class CheckMutantTests(unittest.TestCase):
    def test_the_overlay_replaces_the_file_by_its_relative_path_and_the_tree_is_untouched(self):
        with _Tree() as tree:
            mutant = mutation_check.load_mutants([_mutant()])[0]
            run, calls = _runner()
            result = mutation_check.check_mutant(mutant, tree.root, tree.work, run)
            self.assertEqual(result.outcome, Outcome.CAUGHT)
            self.assertEqual(len(calls), 1)
            overlay_arg = next(a for a in calls[0] if a.startswith("-overlay="))
            overlay = json.loads(Path(overlay_arg.removeprefix("-overlay=")).read_text(encoding="utf-8"))
            self.assertEqual(list(overlay["Replace"]), ["demo/demo.go"])
            mutated = Path(overlay["Replace"]["demo/demo.go"]).read_text(encoding="utf-8")
            self.assertIn("return 11", mutated)
            self.assertNotIn("return 10", mutated)
            self.assertEqual((tree.root / "demo" / "demo.go").read_text(encoding="utf-8"), SOURCE)

    def test_an_anchor_that_does_not_occur_once_runs_nothing(self):
        for old in ("return 99", "i"):
            with self.subTest(old=old), _Tree() as tree:
                occurrences = SOURCE.count(old)
                self.assertNotEqual(occurrences, 1)
                mutant = mutation_check.load_mutants([_mutant(old=old)])[0]
                run, calls = _runner()
                result = mutation_check.check_mutant(mutant, tree.root, tree.work, run)
                self.assertEqual(result.outcome, Outcome.BAD_ANCHOR)
                self.assertIn(f"occurs {occurrences} times", result.detail)
                self.assertEqual(calls, [])

    def test_a_missing_file_is_a_bad_anchor_not_a_crash(self):
        with _Tree() as tree:
            mutant = mutation_check.load_mutants([_mutant(file="demo/absent.go")])[0]
            run, calls = _runner()
            result = mutation_check.check_mutant(mutant, tree.root, tree.work, run)
            self.assertEqual(result.outcome, Outcome.BAD_ANCHOR)
            self.assertIn("does not exist", result.detail)
            self.assertEqual(calls, [])

    def test_a_passing_run_is_a_survivor(self):
        with _Tree() as tree:
            mutant = mutation_check.load_mutants([_mutant()])[0]
            run, _ = _runner(mutant=PASSED)
            self.assertEqual(mutation_check.check_mutant(mutant, tree.root, tree.work, run).outcome, Outcome.SURVIVED)


class BaselineTests(unittest.TestCase):
    def test_a_failing_baseline_runs_no_mutant_and_exits_one(self):
        with _Tree() as tree:
            mutants = mutation_check.load_mutants([_mutant(), _mutant(name="M2", old="Limit()", new="Limit2()")])
            run, calls = _runner(baseline=CAUGHT)
            out = io.StringIO()
            self.assertEqual(mutation_check.check_all(mutants, tree.root, run, out), 1)
            self.assertEqual(_overlaid(calls), [])
            self.assertEqual(out.getvalue().count("BASELINE FAIL"), 2)
            self.assertIn("mutation-check: 0/2 caught", out.getvalue())

    def test_a_named_test_missing_from_the_baseline_is_a_baseline_failure(self):
        with _Tree() as tree:
            mutants = mutation_check.load_mutants([_mutant(tests=["TestLimit", "TestTypo"])])
            run, calls = _runner()
            out = io.StringIO()
            self.assertEqual(mutation_check.check_all(mutants, tree.root, run, out), 1)
            self.assertEqual(_overlaid(calls), [])
            self.assertIn("TestTypo", out.getvalue())

    def test_each_package_and_tests_pair_runs_one_baseline(self):
        with _Tree() as tree:
            mutants = mutation_check.load_mutants([_mutant(), _mutant(name="M2", old="Limit()", new="Limit2()")])
            run, calls = _runner()
            out = io.StringIO()
            self.assertEqual(mutation_check.check_all(mutants, tree.root, run, out), 0)
            self.assertEqual(len(calls) - len(_overlaid(calls)), 1)
            self.assertEqual(len(_overlaid(calls)), 2)
            self.assertIn("mutation-check: 2/2 caught", out.getvalue())


class ReportTests(unittest.TestCase):
    def _results(self, *outcomes):
        mutants = mutation_check.load_mutants([_mutant(name=f"M{i}", old=f"x{i}") for i in range(len(outcomes))])
        details = {Outcome.CAUGHT: "TestLimit", Outcome.BUILD_FAIL: "undefined: x", Outcome.CRASHED: "panic: test timed out"}
        return [mutation_check.Result(m, o, details.get(o, "")) for m, o in zip(mutants, outcomes)]

    def test_caught_and_crashed_mutants_exit_zero(self):
        out = io.StringIO()
        rc = mutation_check.report(self._results(Outcome.CAUGHT, Outcome.CRASHED), out)
        self.assertEqual(rc, 0)
        self.assertIn("M0: CAUGHT by TestLimit", out.getvalue())
        self.assertIn("M1: CRASHED", out.getvalue())
        self.assertIn("    panic: test timed out", out.getvalue())
        self.assertIn("mutation-check: 2/2 caught", out.getvalue())

    def test_anything_unproven_exits_one(self):
        for outcome in (Outcome.SURVIVED, Outcome.BUILD_FAIL, Outcome.BAD_ANCHOR, Outcome.BASELINE_FAIL):
            with self.subTest(outcome=outcome):
                out = io.StringIO()
                rc = mutation_check.report(self._results(Outcome.CAUGHT, outcome), out)
                self.assertEqual(rc, 1)
                self.assertIn(f"M1: {outcome}", out.getvalue())
                self.assertIn("mutation-check: 1/2 caught", out.getvalue())

    def test_a_build_failure_prints_its_output(self):
        out = io.StringIO()
        mutation_check.report(self._results(Outcome.BUILD_FAIL), out)
        self.assertIn("    undefined: x", out.getvalue())


class MainTests(unittest.TestCase):
    def test_usage_and_input_errors_exit_two(self):
        with tempfile.TemporaryDirectory() as d:
            bad = Path(d) / "bad.json"
            bad.write_text("{not json", encoding="utf-8")
            dup = Path(d) / "dup.json"
            dup.write_text(json.dumps([_mutant(), _mutant()]), encoding="utf-8")
            for argv in ([], ["a.json", "b.json"], [str(Path(d) / "missing.json")], [str(bad)], [str(dup)]):
                with self.subTest(argv=argv):
                    self.assertEqual(mutation_check.main(argv=argv, run=_runner()[0], out=io.StringIO()), 2)

    def test_a_missing_go_binary_is_reported_not_raised(self):
        rc, output = mutation_check._run_command(["definitely-not-a-binary-mutation-check"])
        self.assertEqual(rc, 127)
        self.assertIn("definitely-not-a-binary-mutation-check", output)


if __name__ == "__main__":
    unittest.main()
