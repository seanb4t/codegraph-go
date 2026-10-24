#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.12"
# dependencies = []
# ///
"""Prove that Go tests catch the bugs they target, by running them against mutants.

`task mutation-check -- <mutants.json>` runs this script. The JSON file is a list
of mutants, each an object with these keys:

    name      a short label, for example "M1"
    file      the Go source file to mutate, relative to the repository root
    old       text that occurs exactly once in that file
    new       the text that replaces it
    package   the package to test, for example "./internal/daemon/"
    tests     the names of the tests that must catch the mutant

For each mutant the script writes the mutated file to a scratch directory and runs
`go test <package> -overlay=<overlay.json> -run '^(<tests>)$' -count=1`, so the
working tree never changes and there is nothing to restore. Each mutant ends in
exactly one outcome:

    CAUGHT       go test failed and printed at least one `--- FAIL` line
    SURVIVED     go test passed: the named tests do not catch this mutant
    BUILD FAIL   go test failed with no `--- FAIL` line; the mutant does not compile
    BAD ANCHOR   `old` does not occur exactly once in `file`; nothing was run

The exit status is 0 only when every mutant is CAUGHT.
"""

from __future__ import annotations

import json
import re
import subprocess
import sys
import tempfile
from collections.abc import Callable, Sequence
from dataclasses import dataclass
from pathlib import Path
from typing import TextIO

Runner = Callable[[Sequence[str]], tuple[int, str]]

CAUGHT = "CAUGHT"
SURVIVED = "SURVIVED"
BUILD_FAIL = "BUILD FAIL"
BAD_ANCHOR = "BAD ANCHOR"

FAIL_LINE_RE = re.compile(r"^\s*--- FAIL: (\S+)", re.MULTILINE)
REQUIRED_KEYS = ("name", "file", "old", "new", "package", "tests")


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
    outcome: str
    detail: str


def load_mutants(data: object) -> list[Mutant]:
    """Mutants from decoded JSON; raises ValueError on any malformed entry."""
    if not isinstance(data, list) or not data:
        raise ValueError("the mutants file must hold a non-empty JSON list")
    mutants = []
    for i, entry in enumerate(data):
        if not isinstance(entry, dict):
            raise ValueError(f"mutant {i} is not an object")
        missing = [k for k in REQUIRED_KEYS if k not in entry]
        if missing:
            raise ValueError(f"mutant {i} is missing {', '.join(missing)}")
        tests = entry["tests"]
        if not isinstance(tests, list) or not tests or not all(isinstance(t, str) and t for t in tests):
            raise ValueError(f"mutant {i} ({entry['name']}) needs a non-empty list of test names")
        if entry["old"] == entry["new"]:
            raise ValueError(f"mutant {i} ({entry['name']}) has the same old and new text")
        mutants.append(
            Mutant(entry["name"], entry["file"], entry["old"], entry["new"], entry["package"], tuple(tests))
        )
    return mutants


def run_pattern(tests: Sequence[str]) -> str:
    """An anchored -run pattern that matches exactly the named tests."""
    return "^(" + "|".join(re.escape(t) for t in tests) + ")$"


def test_command(mutant: Mutant, overlay: str) -> list[str]:
    return ["go", "test", mutant.package, f"-overlay={overlay}", "-run", run_pattern(mutant.tests), "-count=1"]


def classify(rc: int, output: str) -> tuple[str, str]:
    """The outcome of one go test run, and the failing tests or output behind it."""
    if rc == 0:
        return SURVIVED, ""
    failed = FAIL_LINE_RE.findall(output)
    if not failed:
        return BUILD_FAIL, output.strip()
    return CAUGHT, ", ".join(dict.fromkeys(failed))


def check_mutant(mutant: Mutant, root: Path, scratch: Path, run: Runner) -> Result:
    source = root / mutant.file
    text = source.read_text(encoding="utf-8")
    count = text.count(mutant.old)
    if count != 1:
        return Result(mutant, BAD_ANCHOR, f"`old` occurs {count} times in {mutant.file}, want exactly 1")
    work = scratch / mutant.name
    work.mkdir(parents=True)
    mutated = work / source.name
    mutated.write_text(text.replace(mutant.old, mutant.new, 1), encoding="utf-8")
    overlay = work / "overlay.json"
    overlay.write_text(json.dumps({"Replace": {str(source.resolve()): str(mutated)}}), encoding="utf-8")
    rc, output = run(test_command(mutant, str(overlay)))
    outcome, detail = classify(rc, output)
    return Result(mutant, outcome, detail)


def report(results: Sequence[Result], out: TextIO) -> int:
    for r in results:
        line = f"{r.mutant.name}: {r.outcome}"
        if r.outcome == CAUGHT:
            line += f" by {r.detail}"
        print(line, file=out)
        if r.outcome in (BUILD_FAIL, BAD_ANCHOR) and r.detail:
            print("    " + r.detail.replace("\n", "\n    "), file=out)
    caught = sum(1 for r in results if r.outcome == CAUGHT)
    print(f"mutation-check: {caught}/{len(results)} caught", file=out)
    return 0 if caught == len(results) else 1


def check_all(mutants: Sequence[Mutant], root: Path, run: Runner, out: TextIO) -> int:
    with tempfile.TemporaryDirectory(prefix="mutation-check-") as scratch:
        results = [check_mutant(m, root, Path(scratch), run) for m in mutants]
    return report(results, out)


def _run_command(argv: Sequence[str]) -> tuple[int, str]:
    done = subprocess.run(list(argv), capture_output=True, text=True, encoding="utf-8", errors="replace", check=False)
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
