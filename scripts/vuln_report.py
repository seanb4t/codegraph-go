#!/usr/bin/env python3
"""Build the tool binaries and scan each one with govulncheck, reporting every outcome.

`task vuln` runs this script. For each tool in ROSTER it builds the binary from its
tool modfile, then scans the built binary with the scanner command given after `--`
(the Taskfile passes `env GOWORK=off go tool -modfile=go.tool.mod govulncheck
-mode=binary`). Each tool ends in exactly one outcome:

    BUILD ERROR   go build exited non-zero; the binary was not scanned
    CLEAN         the scan exited 0
    DETECTED      the scan exited 3 (a symbol-matched vulnerability)
    SCAN ERROR    the scan exited with any other status

The scan is advisory: no outcome fails the task, because goreleaser's binary
carries a permanently unfixed, symbol-reachable match (GO-2026-5932) that would
keep a blocking gate red forever. The exit status is about the loop itself. It is
0 when every tool was attempted and every built binary was scanned, and 1
otherwise, because a tool the loop never tried or a binary it never scanned is a
defect in this script, not a vulnerability finding.

`vuln:selftest` in Taskfile.yml stays the proof that the real scanner can detect;
scripts/test_vuln_report.py proves this loop reports what the scanner says.
"""

from __future__ import annotations

import os
import subprocess
import sys
import tempfile
from collections.abc import Callable, Sequence
from dataclasses import dataclass
from typing import TextIO

Runner = Callable[[Sequence[str]], tuple[int, str, str]]


@dataclass(frozen=True)
class Tool:
    name: str
    modfile: str
    package: str


ROSTER: tuple[Tool, ...] = (
    Tool("task", "go.tool.mod", "github.com/go-task/task/v3/cmd/task"),
    Tool("goreleaser", "go.tool.mod", "github.com/goreleaser/goreleaser/v2"),
    Tool("govulncheck", "go.tool.mod", "golang.org/x/vuln/cmd/govulncheck"),
    Tool("actionlint", "go.tool-lint.mod", "github.com/rhysd/actionlint/cmd/actionlint"),
    Tool(
        "golangci-lint",
        "go.tool-golangci.mod",
        "github.com/golangci/golangci-lint/v2/cmd/golangci-lint",
    ),
    Tool("buf", "go.tool-proto.mod", "github.com/bufbuild/buf/cmd/buf"),
    Tool("protoc-gen-go", "go.tool-proto.mod", "google.golang.org/protobuf/cmd/protoc-gen-go"),
    Tool(
        "protoc-gen-connect-go",
        "go.tool-proto.mod",
        "connectrpc.com/connect/cmd/protoc-gen-connect-go",
    ),
    Tool("changie", "go.tool-changie.mod", "github.com/miniscruff/changie"),
)

BUILD_ERROR = "BUILD ERROR"
CLEAN = "CLEAN"
DETECTED = "DETECTED"
SCAN_ERROR = "SCAN ERROR"
DETECTED_EXIT = 3


@dataclass(frozen=True)
class Result:
    tool: Tool
    outcome: str
    build_rc: int
    scan_rc: int | None
    output: str


@dataclass(frozen=True)
class Tally:
    roster_size: int
    built: int
    scanned: int
    build_errors: int
    detected: int
    scan_errors: int

    def summary(self) -> str:
        return (
            f"vuln: built {self.built}/{self.roster_size}, scanned {self.scanned}, "
            f"build errors {self.build_errors}, detected {self.detected}, "
            f"scan errors {self.scan_errors}"
        )

    def loop_defect(self) -> str | None:
        """Why the loop itself failed, or None when every tool was handled."""
        attempted = self.built + self.build_errors
        if attempted != self.roster_size:
            return f"attempted {attempted} of {self.roster_size} tools"
        if self.scanned != self.built:
            return f"scanned {self.scanned} of {self.built} built binaries"
        return None


def classify(build_rc: int, scan_rc: int | None) -> str:
    """The outcome for one tool from its two exit statuses."""
    if build_rc != 0:
        return BUILD_ERROR
    if scan_rc == 0:
        return CLEAN
    if scan_rc == DETECTED_EXIT:
        return DETECTED
    return SCAN_ERROR


def build_command(tool: Tool, scratch: str) -> list[str]:
    return [
        "env",
        "GOWORK=off",
        "go",
        "build",
        f"-modfile={tool.modfile}",
        "-o",
        os.path.join(scratch, tool.name),
        tool.package,
    ]


def scan_command(scanner: Sequence[str], tool: Tool, scratch: str) -> list[str]:
    return [*scanner, os.path.join(scratch, tool.name)]


def run_tool(tool: Tool, scanner: Sequence[str], scratch: str, run: Runner) -> Result:
    build_rc, build_out, build_err = run(build_command(tool, scratch))
    if build_rc != 0:
        return Result(tool, BUILD_ERROR, build_rc, None, build_out + build_err)
    scan_rc, scan_out, scan_err = run(scan_command(scanner, tool, scratch))
    return Result(tool, classify(build_rc, scan_rc), build_rc, scan_rc, scan_out + scan_err)


def tally(results: Sequence[Result], roster_size: int) -> Tally:
    outcomes = [r.outcome for r in results]
    build_errors = outcomes.count(BUILD_ERROR)
    built = len(outcomes) - build_errors
    return Tally(
        roster_size=roster_size,
        built=built,
        scanned=sum(1 for r in results if r.scan_rc is not None),
        build_errors=build_errors,
        detected=outcomes.count(DETECTED),
        scan_errors=outcomes.count(SCAN_ERROR),
    )


def report_result(result: Result, out: TextIO) -> None:
    name = result.tool.name
    if result.outcome == CLEAN:
        print(
            f"    {name}: CLEAN (exit 0: no symbol-matched vulnerability, or a "
            "module-level-only match govulncheck does not consider a symbol hit)",
            file=out,
        )
        return
    if result.outcome == BUILD_ERROR:
        detail = f"go build exited {result.build_rc}; the binary was not scanned"
    elif result.outcome == DETECTED:
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


def run_report(scanner: Sequence[str], scratch: str, run: Runner, out: TextIO) -> int:
    """Build and scan every roster tool in order; the return value is the exit status."""
    results: list[Result] = []
    for tool in ROSTER:
        result = run_tool(tool, scanner, scratch, run)
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
    scanner = argv[1:]
    with tempfile.TemporaryDirectory(prefix="vuln-report-") as scratch:
        return run_report(scanner, scratch, run, out)


if __name__ == "__main__":
    raise SystemExit(main())
