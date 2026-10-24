"""Unit suite for scripts/mutation_check.py.

Every case drives the real mutate-and-test loop against files in a temporary
directory, with canned (returncode, output) runner results. Nothing here runs go;
how go test behaves is not ours to test, only what this loop does with its exit
status and output.

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

SOURCE = "package demo\n\nfunc Limit() int { return 10 }\n"


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
        self.scratch = self.root / "scratch"
        self.scratch.mkdir()
        return self

    def __exit__(self, *exc):
        self._dir.cleanup()


def _runner(result=(1, "--- FAIL: TestLimit (0.00s)\nFAIL\n")):
    calls = []

    def run(argv):
        calls.append(list(argv))
        return result

    return run, calls


class LoadTests(unittest.TestCase):
    def test_a_valid_list_loads(self):
        mutants = mutation_check.load_mutants([_mutant(), _mutant(name="M2", tests=["TestA", "TestB"])])
        self.assertEqual([m.name for m in mutants], ["M1", "M2"])
        self.assertEqual(mutants[1].tests, ("TestA", "TestB"))

    def test_malformed_input_is_rejected(self):
        cases = {
            "not a list": {"name": "M1"},
            "empty list": [],
            "not an object": ["M1"],
            "missing key": [{k: v for k, v in _mutant().items() if k != "package"}],
            "no tests": [_mutant(tests=[])],
            "blank test name": [_mutant(tests=[""])],
            "no change": [_mutant(new="return 10")],
        }
        for label, data in cases.items():
            with self.subTest(label):
                with self.assertRaises(ValueError):
                    mutation_check.load_mutants(data)


class CommandTests(unittest.TestCase):
    def test_run_pattern_is_anchored_and_escaped(self):
        self.assertEqual(mutation_check.run_pattern(["TestA", "Test.B"]), r"^(TestA|Test\.B)$")

    def test_test_command_shape(self):
        mutant = mutation_check.load_mutants([_mutant()])[0]
        self.assertEqual(
            mutation_check.test_command(mutant, "/tmp/o.json"),
            ["go", "test", "./demo/", "-overlay=/tmp/o.json", "-run", "^(TestLimit)$", "-count=1"],
        )


class ClassifyTests(unittest.TestCase):
    def test_classify_table(self):
        cases = (
            (0, "ok  demo 0.1s\n", mutation_check.SURVIVED, ""),
            (1, "--- FAIL: TestLimit (0.00s)\nFAIL\n", mutation_check.CAUGHT, "TestLimit"),
            (1, "    --- FAIL: TestLimit/sub (0.00s)\n--- FAIL: TestLimit (0.00s)\n", mutation_check.CAUGHT, "TestLimit/sub, TestLimit"),
            (1, "--- FAIL: TestA\n--- FAIL: TestA\n", mutation_check.CAUGHT, "TestA"),
            (1, "demo.go:3:9: declared and not used: x\nFAIL demo [build failed]\n", mutation_check.BUILD_FAIL, "demo.go:3:9: declared and not used: x\nFAIL demo [build failed]"),
        )
        for rc, output, want, detail in cases:
            with self.subTest(rc=rc, output=output):
                self.assertEqual(mutation_check.classify(rc, output), (want, detail))


class CheckMutantTests(unittest.TestCase):
    def test_the_overlay_replaces_the_real_file_and_the_tree_is_untouched(self):
        with _Tree() as tree:
            mutant = mutation_check.load_mutants([_mutant()])[0]
            run, calls = _runner()
            result = mutation_check.check_mutant(mutant, tree.root, tree.scratch, run)
            self.assertEqual(result.outcome, mutation_check.CAUGHT)
            self.assertEqual(len(calls), 1)
            overlay_arg = next(a for a in calls[0] if a.startswith("-overlay="))
            overlay = json.loads(Path(overlay_arg.removeprefix("-overlay=")).read_text(encoding="utf-8"))
            real = str((tree.root / "demo" / "demo.go").resolve())
            self.assertEqual(list(overlay["Replace"]), [real])
            mutated = Path(overlay["Replace"][real]).read_text(encoding="utf-8")
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
                result = mutation_check.check_mutant(mutant, tree.root, tree.scratch, run)
                self.assertEqual(result.outcome, mutation_check.BAD_ANCHOR)
                self.assertIn(f"occurs {occurrences} times", result.detail)
                self.assertEqual(calls, [])

    def test_a_passing_run_is_a_survivor(self):
        with _Tree() as tree:
            mutant = mutation_check.load_mutants([_mutant()])[0]
            run, _ = _runner((0, "ok\n"))
            self.assertEqual(mutation_check.check_mutant(mutant, tree.root, tree.scratch, run).outcome, mutation_check.SURVIVED)


class ReportTests(unittest.TestCase):
    def _results(self, *outcomes):
        mutants = mutation_check.load_mutants([_mutant(name=f"M{i}") for i in range(len(outcomes))])
        details = {mutation_check.CAUGHT: "TestLimit", mutation_check.BUILD_FAIL: "undefined: x"}
        return [mutation_check.Result(m, o, details.get(o, "")) for m, o in zip(mutants, outcomes)]

    def test_every_mutant_caught_exits_zero(self):
        out = io.StringIO()
        rc = mutation_check.report(self._results(mutation_check.CAUGHT, mutation_check.CAUGHT), out)
        self.assertEqual(rc, 0)
        self.assertIn("M0: CAUGHT by TestLimit", out.getvalue())
        self.assertIn("mutation-check: 2/2 caught", out.getvalue())

    def test_anything_not_caught_exits_one(self):
        for outcome in (mutation_check.SURVIVED, mutation_check.BUILD_FAIL, mutation_check.BAD_ANCHOR):
            with self.subTest(outcome=outcome):
                out = io.StringIO()
                rc = mutation_check.report(self._results(mutation_check.CAUGHT, outcome), out)
                self.assertEqual(rc, 1)
                self.assertIn(f"M1: {outcome}", out.getvalue())
                self.assertIn("mutation-check: 1/2 caught", out.getvalue())

    def test_a_build_failure_prints_its_output(self):
        out = io.StringIO()
        mutation_check.report(self._results(mutation_check.BUILD_FAIL), out)
        self.assertIn("    undefined: x", out.getvalue())


class MainTests(unittest.TestCase):
    def test_usage_and_input_errors_exit_two(self):
        with tempfile.TemporaryDirectory() as d:
            bad = Path(d) / "bad.json"
            bad.write_text("{not json", encoding="utf-8")
            for argv in ([], ["a.json", "b.json"], [str(Path(d) / "missing.json")], [str(bad)]):
                with self.subTest(argv=argv):
                    self.assertEqual(mutation_check.main(argv=argv, run=_runner()[0], out=io.StringIO()), 2)

    def test_check_all_runs_every_mutant(self):
        with _Tree() as tree:
            mutants = mutation_check.load_mutants([_mutant(), _mutant(name="M2", old="Limit()", new="Limit2()")])
            run, calls = _runner()
            out = io.StringIO()
            self.assertEqual(mutation_check.check_all(mutants, tree.root, run, out), 0)
            self.assertEqual(len(calls), 2)
            self.assertIn("mutation-check: 2/2 caught", out.getvalue())


if __name__ == "__main__":
    unittest.main()
