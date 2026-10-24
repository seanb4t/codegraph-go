#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.12"
# dependencies = []
# ///
"""Prove that Go tests catch the bugs they target, by running them against mutants.

`task mutation-check -- <mutants.json>` runs this script from the repository root.
The JSON file is a list of mutants, each an object with these string keys:

    name      a short label, for example "M1"; unique in the file
    file      the Go source file to mutate, relative to the repository root
    old       text that occurs exactly once in that file
    new       the text that replaces it (may be empty)
    package   the package to test, for example "./internal/daemon/"
    tests     the names of the top-level tests that must catch the mutant

First, each distinct (package, tests) pair runs once against the unmutated tree,
and every named test must report PASS. A pair that fails this baseline marks its
mutants BASELINE FAIL and none of them runs, because a test that already fails
proves nothing about a mutant.

Then each mutant's file is written to a scratch directory and the named tests run
with `go test <package> -overlay=<overlay.json> -run '^(<tests>)$'`, so the working
tree never changes. Each mutant ends in exactly one outcome:

    CAUGHT          a named test reported FAIL
    CRASHED         go test failed without a failing test: a timeout, a panic
                    outside a test, or os.Exit in the code under test
    SURVIVED        every named test passed: no test catches this mutant
    BUILD FAIL      the mutant does not compile or vet; it proves nothing
    BAD ANCHOR      `old` does not occur exactly once, or `file` is missing
    BASELINE FAIL   the named tests do not all pass on the unmutated tree

The exit status is 0 only when every mutant is CAUGHT or CRASHED, 1 otherwise,
and 2 when the mutants file is unreadable or malformed.
"""

from __future__ import annotations

import json
import re
import subprocess
import sys
import tempfile
from collections.abc import Callable, Sequence
from dataclasses import dataclass
from enum import StrEnum
from pathlib import Path
from typing import TextIO

Runner = Callable[[Sequence[str]], tuple[int, str]]

TIMEOUT = "5m"
FIELDS = ("name", "file", "old", "new", "package", "tests")
FAIL_LINE_RE = re.compile(r"^\s*--- FAIL: (\S+)", re.MULTILINE)
PASS_LINE_RE = re.compile(r"^--- PASS: (\S+)", re.MULTILINE)
BUILD_FAIL_RE = re.compile(r"\[(build|setup) failed\]")


class Outcome(StrEnum):
    CAUGHT = "CAUGHT"
    CRASHED = "CRASHED"
    SURVIVED = "SURVIVED"
    BUILD_FAIL = "BUILD FAIL"
    BAD_ANCHOR = "BAD ANCHOR"
    BASELINE_FAIL = "BASELINE FAIL"


PROVEN = (Outcome.CAUGHT, Outcome.CRASHED)


@dataclass(frozen=True)
class Mutant:
    name: str
    file: str
    old: str
    new: str
    package: str
    tests: tuple[str, ...]


@dataclass(frozen=True)
class Result:
    mutant: Mutant
    outcome: Outcome
    detail: str


def load_mutants(data: object) -> list[Mutant]:
    """Mutants from decoded JSON; raises ValueError on any malformed entry."""
    if not isinstance(data, list) or not data:
        raise ValueError("the mutants file must hold a non-empty JSON list")
    mutants: list[Mutant] = []
    for i, entry in enumerate(data):
        if not isinstance(entry, dict):
            raise ValueError(f"mutant {i} is not an object")
        missing = [k for k in FIELDS if k not in entry]
        if missing:
            raise ValueError(f"mutant {i} is missing {', '.join(missing)}")
        for key in ("name", "file", "old", "new", "package"):
            if not isinstance(entry[key], str):
                raise ValueError(f"mutant {i}: {key} must be a string")
        for key in ("name", "file", "old", "package"):
            if not entry[key]:
                raise ValueError(f"mutant {i}: {key} must not be empty")
        name = entry["name"]
        if entry["package"].startswith("-"):
            raise ValueError(f"mutant {i} ({name}): package must not start with '-'")
        tests = entry["tests"]
        if not isinstance(tests, list) or not tests or not all(isinstance(t, str) and t for t in tests):
            raise ValueError(f"mutant {i} ({name}) needs a non-empty list of test names")
        if any("/" in t for t in tests):
            raise ValueError(f"mutant {i} ({name}): name top-level tests only, without '/'")
        if entry["old"] == entry["new"]:
            raise ValueError(f"mutant {i} ({name}) has the same old and new text")
        if any(m.name == name for m in mutants):
            raise ValueError(f"mutant {i}: the name {name!r} is used twice")
        mutants.append(Mutant(name, entry["file"], entry["old"], entry["new"], entry["package"], tuple(tests)))
    return mutants


def run_pattern(tests: Sequence[str]) -> str:
    """An anchored -run pattern that matches exactly the named top-level tests."""
    return "^(" + "|".join(re.escape(t) for t in tests) + ")$"


def test_command(package: str, tests: Sequence[str], overlay: str | None = None) -> list[str]:
    argv = ["go", "test", package]
    if overlay is not None:
        argv.append(f"-overlay={overlay}")
    return [*argv, "-run", run_pattern(tests), "-count=1", "-v", f"-timeout={TIMEOUT}"]


def classify(rc: int, output: str) -> tuple[Outcome, str]:
    """The outcome of one go test run against a mutant, and the detail behind it."""
    if rc == 0:
        return Outcome.SURVIVED, ""
    if BUILD_FAIL_RE.search(output):
        return Outcome.BUILD_FAIL, output.strip()
    failed = FAIL_LINE_RE.findall(output)
    if failed:
        return Outcome.CAUGHT, ", ".join(dict.fromkeys(failed))
    return Outcome.CRASHED, output.strip()


def baseline_failure(rc: int, output: str, tests: Sequence[str]) -> str | None:
    """Why the unmutated run does not prove the tests pass, or None when it does."""
    if rc != 0:
        return f"go test exited {rc} on the unmutated tree:\n{output.strip()}"
    passed = set(PASS_LINE_RE.findall(output))
    missing = [t for t in tests if t not in passed]
    if missing:
        return f"no PASS line on the unmutated tree for {', '.join(missing)} (misspelled, skipped or build-tagged?)"
    return None


def check_mutant(mutant: Mutant, root: Path, work: Path, run: Runner) -> Result:
    source = root / mutant.file
    if not source.is_file():
        return Result(mutant, Outcome.BAD_ANCHOR, f"{mutant.file} does not exist")
    text = source.read_text(encoding="utf-8")
    count = text.count(mutant.old)
    if count != 1:
        return Result(mutant, Outcome.BAD_ANCHOR, f"`old` occurs {count} times in {mutant.file}, want exactly 1")
    work.mkdir(parents=True)
    mutated = work / source.name
    mutated.write_text(text.replace(mutant.old, mutant.new, 1), encoding="utf-8")
    overlay = work / "overlay.json"
    # A relative key resolves against go's own working directory, which a
    # symlinked checkout path would make differ from the resolved real path.
    overlay.write_text(json.dumps({"Replace": {mutant.file: str(mutated)}}), encoding="utf-8")
    rc, output = run(test_command(mutant.package, mutant.tests, str(overlay)))
    outcome, detail = classify(rc, output)
    return Result(mutant, outcome, detail)


def check_all(mutants: Sequence[Mutant], root: Path, run: Runner, out: TextIO) -> int:
    baselines: dict[tuple[str, tuple[str, ...]], str | None] = {}
    for m in mutants:
        key = (m.package, m.tests)
        if key not in baselines:
            baselines[key] = baseline_failure(*run(test_command(m.package, m.tests)), m.tests)
    results = []
    with tempfile.TemporaryDirectory(prefix="mutation-check-") as scratch:
        for i, m in enumerate(mutants):
            failure = baselines[(m.package, m.tests)]
            if failure is not None:
                results.append(Result(m, Outcome.BASELINE_FAIL, failure))
            else:
                results.append(check_mutant(m, root, Path(scratch) / f"{i:03d}", run))
    return report(results, out)


def report(results: Sequence[Result], out: TextIO) -> int:
    for r in results:
        line = f"{r.mutant.name}: {r.outcome}"
        if r.outcome == Outcome.CAUGHT:
            line += f" by {r.detail}"
        print(line, file=out)
        if r.outcome != Outcome.CAUGHT and r.detail:
            print("    " + r.detail.replace("\n", "\n    "), file=out)
    proven = sum(1 for r in results if r.outcome in PROVEN)
    print(f"mutation-check: {proven}/{len(results)} caught", file=out)
    return 0 if proven == len(results) else 1


def _run_command(argv: Sequence[str]) -> tuple[int, str]:
    try:
        done = subprocess.run(list(argv), capture_output=True, text=True, encoding="utf-8", errors="replace", check=False)
    except FileNotFoundError as exc:
        return 127, f"{argv[0]}: {exc}"
    return done.returncode, done.stdout + done.stderr


def main(argv: Sequence[str] | None = None, run: Runner | None = None, out: TextIO | None = None) -> int:
    argv = sys.argv[1:] if argv is None else list(argv)
    run = _run_command if run is None else run
    out = sys.stdout if out is None else out
    if len(argv) != 1:
        print("usage: mutation_check.py <mutants.json>", file=sys.stderr)
        return 2
    try:
        mutants = load_mutants(json.loads(Path(argv[0]).read_text(encoding="utf-8")))
    except (OSError, ValueError) as exc:
        print(f"mutation-check: {exc}", file=sys.stderr)
        return 2
    return check_all(mutants, Path.cwd(), run, out)


if __name__ == "__main__":
    raise SystemExit(main())
