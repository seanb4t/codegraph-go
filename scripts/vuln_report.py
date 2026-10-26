#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.12"
# dependencies = []
# ///
"""Find the mise-installed tool binaries and scan each one with govulncheck, reporting every outcome.

`task vuln` runs this script. For each tool in ROSTER it finds the binary with
`mise which`, then scans it with the scanner command given after `--` (the
Taskfile passes `govulncheck -mode=binary`). Each tool ends in exactly one outcome:

    NOT FOUND     mise which exited non-zero or printed no path; nothing was scanned
    CLEAN         the scan exited 0
    DETECTED      the scan exited 3 (a symbol-matched vulnerability)
    SCAN ERROR    the scan exited with any other status

The scan is advisory: no scan outcome fails the task, because goreleaser's binary
carries a permanently unfixed, symbol-reachable match (GO-2026-5932) that would
keep a blocking gate red forever. The exit status is about the loop itself. It is
0 when every roster tool was found and scanned, and 1 otherwise. mise.toml pins
every roster tool, so NOT FOUND means a broken environment, and a run that scans
less than the whole roster must never read as clean.

`vuln:selftest` in Taskfile.yml stays the proof that the real scanner can detect;
tests/test_vuln_report.py proves this loop reports what the scanner says.
"""

from __future__ import annotations

import subprocess
import sys
from collections.abc import Callable, Sequence
from dataclasses import dataclass
from enum import StrEnum
from typing import TextIO

Runner = Callable[[Sequence[str]], tuple[int, str, str]]


@dataclass(frozen=True)
class Tool:
    """A Go binary that mise.toml pins, named as `mise which` names it."""

    name: str


ROSTER: tuple[Tool, ...] = (
    Tool("task"),
    Tool("goreleaser"),
    Tool("govulncheck"),
    Tool("actionlint"),
    Tool("golangci-lint"),
    Tool("buf"),
    Tool("protoc-gen-go"),
    Tool("protoc-gen-connect-go"),
    Tool("changie"),
    Tool("cosign"),
    Tool("syft"),
)

class Outcome(StrEnum):
    NOT_FOUND = "NOT FOUND"
    CLEAN = "CLEAN"
    DETECTED = "DETECTED"
    SCAN_ERROR = "SCAN ERROR"


DETECTED_EXIT = 3


@dataclass(frozen=True)
class Result:
    tool: Tool
    outcome: Outcome
    locate_rc: int
    scan_rc: int | None
    output: str


@dataclass(frozen=True, kw_only=True)
class Tally:
    roster_size: int
    found: int
    scanned: int
    not_found: int
    detected: int
    scan_errors: int

    def summary(self) -> str:
        return (
            f"vuln: found {self.found}/{self.roster_size}, scanned {self.scanned}, "
            f"not found {self.not_found}, detected {self.detected}, "
            f"scan errors {self.scan_errors}"
        )

    def loop_defect(self) -> str | None:
        """Why the loop itself failed, or None when every tool was handled."""
        attempted = self.found + self.not_found
        if attempted != self.roster_size:
            return f"attempted {attempted} of {self.roster_size} tools"
        if self.not_found:
            return f"{self.not_found} of {self.roster_size} roster tools not found"
        if self.scanned != self.found:
            return f"scanned {self.scanned} of {self.found} found binaries"
        return None


def classify(found: bool, scan_rc: int | None) -> Outcome:
    """The outcome for one tool from whether it was found and the scan's exit status."""
    if not found:
        return Outcome.NOT_FOUND
    if scan_rc == 0:
        return Outcome.CLEAN
    if scan_rc == DETECTED_EXIT:
        return Outcome.DETECTED
    return Outcome.SCAN_ERROR


def locate_command(tool: Tool) -> list[str]:
    return ["mise", "which", tool.name]


def scan_command(scanner: Sequence[str], path: str) -> list[str]:
    return [*scanner, path]


def run_tool(tool: Tool, scanner: Sequence[str], run: Runner) -> Result:
    locate_rc, locate_out, locate_err = run(locate_command(tool))
    path = locate_out.strip()
    if locate_rc != 0 or not path:
        return Result(tool, classify(False, None), locate_rc, None, locate_out + locate_err)
    scan_rc, scan_out, scan_err = run(scan_command(scanner, path))
    return Result(tool, classify(True, scan_rc), locate_rc, scan_rc, scan_out + scan_err)


def tally(results: Sequence[Result], roster_size: int) -> Tally:
    outcomes = [r.outcome for r in results]
    not_found = outcomes.count(Outcome.NOT_FOUND)
    found = len(outcomes) - not_found
    return Tally(
        roster_size=roster_size,
        found=found,
        scanned=sum(1 for r in results if r.scan_rc is not None),
        not_found=not_found,
        detected=outcomes.count(Outcome.DETECTED),
        scan_errors=outcomes.count(Outcome.SCAN_ERROR),
    )


def report_result(result: Result, out: TextIO) -> None:
    name = result.tool.name
    if result.outcome == Outcome.CLEAN:
        print(
            f"    {name}: CLEAN (exit 0: no symbol-matched vulnerability, or a "
            "module-level-only match govulncheck does not consider a symbol hit)",
            file=out,
        )
        return
    if result.outcome == Outcome.NOT_FOUND:
        located = f"mise which exited {result.locate_rc}" if result.locate_rc else "mise which printed no path"
        detail = f"{located}; nothing was scanned"
    elif result.outcome == Outcome.DETECTED:
        detail = (
            f"govulncheck -mode=binary exited {result.scan_rc} "
            "(symbol-matched vulnerability; advisory, does not fail the build)"
        )
    else:
        detail = (
            f"govulncheck -mode=binary exited {result.scan_rc} (usage error or "
            "vuln.go.dev unreachable; reported distinctly, never folded into CLEAN)"
        )
    print(f"::warning::{name}: {result.outcome}: {detail}", file=out)
    print(result.output.rstrip("\n"), file=out)


def run_report(scanner: Sequence[str], run: Runner, out: TextIO) -> int:
    """Find and scan every roster tool in order; the return value is the exit status."""
    results: list[Result] = []
    for tool in ROSTER:
        result = run_tool(tool, scanner, run)
        report_result(result, out)
        results.append(result)
    totals = tally(results, len(ROSTER))
    print(totals.summary(), file=out)
    defect = totals.loop_defect()
    if defect is not None:
        print(f"::error::vuln: loop defect: {defect}", file=out)
        return 1
    return 0


def _run_command(argv: Sequence[str]) -> tuple[int, str, str]:
    try:
        done = subprocess.run(
            list(argv),
            capture_output=True,
            text=True,
            encoding="utf-8",
            errors="replace",
            check=False,
        )
    except FileNotFoundError as exc:
        return 127, "", f"{argv[0]}: {exc}"
    return done.returncode, done.stdout, done.stderr


def main(argv: Sequence[str] | None = None, run: Runner | None = None, out: TextIO | None = None) -> int:
    argv = sys.argv[1:] if argv is None else list(argv)
    run = _run_command if run is None else run
    out = sys.stdout if out is None else out
    if not argv or argv[0] != "--" or len(argv) < 2:
        print("usage: vuln_report.py -- <scanner command...>", file=sys.stderr)
        return 2
    return run_report(argv[1:], run, out)


if __name__ == "__main__":
    raise SystemExit(main())
