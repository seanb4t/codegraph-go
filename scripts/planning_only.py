#!/usr/bin/env python3
"""Decide whether a push or pull request changes only planning files.

A change is planning-only when its file list is non-empty and every changed path,
and the previous path of every rename, is under openspec/. ci.yml's `changes` job
runs this script, and the heavy jobs skip through a job-level condition when it
prints planning_only=true.

The verdict fails open. Whenever the script cannot see the whole change (an API
error, a listing it cannot prove complete, an unknown event, a push with no
usable `before`) it prints planning_only=false with the reason on stderr and exits
0: the safe failure for a skip gate is to run the work.

Untrusted content reaches this script only through the environment, as in
fragment_policy.py, so it never reaches a shell:

    GITHUB_REPOSITORY    owner/name of the repository
    GITHUB_EVENT_NAME    pull_request or push
    PR_NUMBER            the pull request number (pull_request only)
    BEFORE_SHA           github.event.before (push only)
    AFTER_SHA            github.event.after (push only)
    GH_TOKEN             read by gh, which this script runs
    GITHUB_OUTPUT        the step output file; the verdict is one single-line write

A pull request's files come from fragment_policy.fetch_changed_files, which pages
through pulls/{n}/files and fails when the fetched count differs from the pull
request's changed_files. A push's files come from compare/{before}...{after},
which lists at most 300 files, so a listing that reaches that cap is undecided.

The rule lives here and nowhere else; scripts/test_planning_only.py imports it.
"""

from __future__ import annotations

import json
import os
import re
import sys
from collections.abc import Mapping, Sequence
from dataclasses import dataclass
from typing import TextIO

import fragment_policy
from fragment_policy import FileChange, Runner

PLANNING_ROOT = "openspec"
OUTPUT_KEY = "planning_only"
LOG_PREFIX = "planning-only"

EVENT_PULL_REQUEST = "pull_request"
EVENT_PUSH = "push"

# GitHub's compare endpoint lists at most this many files; a listing of exactly this
# length may be truncated, so it cannot prove the change is planning-only.
COMPARE_FILE_CAP = 300

SHA_PATTERN = re.compile(r"[0-9a-f]{40}")
ZERO_SHA = "0" * 40


@dataclass(frozen=True)
class Decision:
    planning_only: bool
    reason: str
    file_count: int = 0


def is_planning_path(path: str) -> bool:
    """True for a file inside openspec/, by exact first segment."""
    segments = path.split("/")
    return len(segments) >= 2 and segments[0] == PLANNING_ROOT


def classify(files: Sequence[FileChange]) -> bool:
    """True when the list is non-empty and every path and previous path is planning."""
    if not files:
        return False
    for change in files:
        for path in (change.path, change.previous_path):
            if path is not None and not is_planning_path(path):
                return False
    return True


def parse_compare_files(payload: str) -> list[FileChange]:
    """Read the `files` array of a compare response into FileChange tuples."""
    body = json.loads(payload)
    if not isinstance(body, dict):
        raise ValueError(f"expected an object, got {type(body).__name__}")
    items = body.get("files")
    if not isinstance(items, list):
        raise ValueError("compare response has no files array")
    changes: list[FileChange] = []
    for index, item in enumerate(items):
        if not isinstance(item, dict):
            raise ValueError(f"files[{index}] is {type(item).__name__}, expected an object")
        filename = item.get("filename")
        status = item.get("status")
        if not isinstance(filename, str) or not isinstance(status, str):
            raise ValueError(f"files[{index}] lacks a string filename and status")
        previous = item.get("previous_filename")
        if previous is not None and not isinstance(previous, str):
            raise ValueError(f"files[{index}] has a non-string previous_filename")
        changes.append(FileChange(filename, status, previous))
    return changes


def fetch_push_files(
    repo: str, before: str, after: str, run: Runner
) -> tuple[list[FileChange], str | None]:
    """Fetch a push's changed files, or explain why the change cannot be seen."""
    if before == ZERO_SHA:
        return [], "push has no before commit (new branch or force push)"
    endpoint = f"repos/{repo}/compare/{before}...{after}"
    rc, stdout, stderr = run(["gh", "api", endpoint])
    if rc != 0:
        return [], f"gh api {endpoint} exited {rc}: {fragment_policy._first_line(stderr)}"
    try:
        files = parse_compare_files(stdout)
    except ValueError as exc:
        return [], f"malformed compare response from GitHub: {exc}"
    if len(files) >= COMPARE_FILE_CAP:
        return [], (
            f"compare listing holds {len(files)} files, the API's cap, "
            "so it may be incomplete"
        )
    return files, None


def decide(environ: Mapping[str, str], run: Runner) -> Decision:
    """The verdict for the event described by the environment. Never raises."""
    repo = environ.get("GITHUB_REPOSITORY", "")
    if not fragment_policy.REPO_PATTERN.fullmatch(repo):
        return Decision(False, "GITHUB_REPOSITORY is missing or malformed")
    event = environ.get("GITHUB_EVENT_NAME", "")
    if event == EVENT_PULL_REQUEST:
        number = environ.get("PR_NUMBER", "")
        if not fragment_policy.PR_NUMBER_PATTERN.fullmatch(number):
            return Decision(False, "PR_NUMBER is missing or malformed")
        files, error = fragment_policy.fetch_changed_files(repo, number, run)
    elif event == EVENT_PUSH:
        before = environ.get("BEFORE_SHA", "")
        after = environ.get("AFTER_SHA", "")
        if not SHA_PATTERN.fullmatch(before) or not SHA_PATTERN.fullmatch(after):
            return Decision(False, "BEFORE_SHA or AFTER_SHA is missing or malformed")
        files, error = fetch_push_files(repo, before, after, run)
    else:
        return Decision(False, f"event {event!r} is not classified")
    if error is not None:
        return Decision(False, error)
    if not files:
        return Decision(False, "GitHub returned an empty changed-file list")
    if classify(files):
        return Decision(True, f"every path is under {PLANNING_ROOT}/", len(files))
    return Decision(False, f"a changed path is outside {PLANNING_ROOT}/", len(files))


def write_output(path: str | None, decision: Decision) -> None:
    """Append the single-line step output. The value is a fixed literal, never data."""
    if not path:
        return
    with open(path, "a", encoding="utf-8") as handle:
        handle.write(f"{OUTPUT_KEY}={'true' if decision.planning_only else 'false'}\n")


def report(decision: Decision, out: TextIO, err: TextIO) -> None:
    verdict = "true" if decision.planning_only else "false"
    reason = fragment_policy.escape_data(decision.reason)
    print(
        f"{LOG_PREFIX}: {decision.file_count} changed files, "
        f"{OUTPUT_KEY}={verdict} ({reason})",
        file=out,
    )
    if not decision.planning_only and decision.file_count == 0:
        print(f"{LOG_PREFIX}: fail open, running every job: {reason}", file=err)


def main(
    environ: Mapping[str, str] | None = None,
    run: Runner | None = None,
    out: TextIO | None = None,
    err: TextIO | None = None,
) -> int:
    environ = os.environ if environ is None else environ
    run = fragment_policy._run_gh if run is None else run
    out = sys.stdout if out is None else out
    err = sys.stderr if err is None else err
    decision = decide(environ, run)
    write_output(environ.get("GITHUB_OUTPUT"), decision)
    report(decision, out, err)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
