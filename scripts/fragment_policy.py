#!/usr/bin/env python3
"""Decide whether a pull request that changes shipped code adds a changelog fragment.

Shipped code is cmd/** or internal/**, excluding *_test.go files and anything under
a testdata/ directory. A pull request that changes shipped code passes only when it
adds a new file under .changes/unreleased/. The gate fails closed whenever it cannot
see the whole change.

Untrusted pull request content reaches this script only through the environment, as
in pr_template_policy.py, so it never reaches a shell:

    GITHUB_REPOSITORY    owner/name of the repository
    PR_NUMBER            the pull request number
    PR_BODY              the pull request body
    GH_TOKEN             read by gh, which this script runs

main() fetches the complete changed-file list itself, through gh's paginated REST
listing of pulls/{n}/files. It does not use `gh pr view --json files`, which stops at
100 files, and it writes no $GITHUB_OUTPUT heredoc, so scripts/
check-workflow-output-delimiter.sh has nothing new to cover.

Divergence from pr_template_policy.py: the exit status IS the verdict (0 pass, 1 fail),
because no later step reads an output and no comment is posted. The check log is the
only guidance a contributor gets.

The rule lives here and nowhere else; scripts/test_fragment_policy.py imports it.
"""

from __future__ import annotations

import json
import os
import subprocess
import sys
from collections.abc import Callable, Mapping, Sequence
from dataclasses import dataclass
from typing import TextIO

SHIPPED_ROOTS = ("cmd", "internal")
TEST_FILE_SUFFIX = "_test.go"
TESTDATA_DIR = "testdata"
FRAGMENT_DIR = ".changes/unreleased/"
FRAGMENT_KEEP_FILE = ".gitkeep"
FRAGMENT_STATUS = "added"

API_ERROR = "api-error"
NO_SHIPPED_CODE = "no-shipped-code"
HAS_FRAGMENT = "has-fragment"
MISSING_FRAGMENT = "missing-fragment"

_SHIPPED_DESCRIPTION = "cmd/** or internal/**, excluding *_test.go and testdata/"
MESSAGES = {
    MISSING_FRAGMENT: (
        f"this PR changes shipped code ({_SHIPPED_DESCRIPTION}) "
        f"but adds no changelog fragment under {FRAGMENT_DIR}"
    ),
    HAS_FRAGMENT: f"shipped code changed and a new {FRAGMENT_DIR} fragment was added",
    NO_SHIPPED_CODE: f"no changed file is shipped code ({_SHIPPED_DESCRIPTION})",
    API_ERROR: "could not read this PR's changed files, so the gate fails closed",
}

LISTED_PATH_LIMIT = 20
GH_TIMEOUT_SECONDS = 120

Runner = Callable[[Sequence[str]], tuple[int, str, str]]


@dataclass(frozen=True)
class FileChange:
    path: str
    status: str
    previous_path: str | None = None


@dataclass(frozen=True)
class Verdict:
    ok: bool
    code: str
    message: str
    shipped: tuple[str, ...] = ()


def is_shipped_code(path: str) -> bool:
    """True for cmd/** and internal/** files that are neither tests nor test data.

    Exact, case-sensitive segment comparison with no extension filter.
    """
    segments = path.split("/")
    if len(segments) < 2 or segments[0] not in SHIPPED_ROOTS:
        return False
    if path.endswith(TEST_FILE_SUFFIX):
        return False
    return TESTDATA_DIR not in segments[:-1]


def shipped_paths(files: Sequence[FileChange]) -> tuple[str, ...]:
    return tuple(sorted({f.path for f in files if is_shipped_code(f.path)}))


def is_fragment(change: FileChange) -> bool:
    """A newly added file under .changes/unreleased/ that is not the keep file."""
    return (
        change.status.lower() == FRAGMENT_STATUS
        and change.path.startswith(FRAGMENT_DIR)
        and len(change.path) > len(FRAGMENT_DIR)
        and change.path.rsplit("/", 1)[-1] != FRAGMENT_KEEP_FILE
    )


def evaluate(
    files: Sequence[FileChange], body: str | None, api_error: str | None = None
) -> Verdict:
    """Pure verdict over a complete changed-file list. Never mutates its input."""
    if api_error is not None:
        return Verdict(False, API_ERROR, f"{MESSAGES[API_ERROR]}: {api_error}")
    shipped = shipped_paths(files)
    if not shipped:
        return Verdict(True, NO_SHIPPED_CODE, MESSAGES[NO_SHIPPED_CODE])
    if any(is_fragment(f) for f in files):
        return Verdict(True, HAS_FRAGMENT, MESSAGES[HAS_FRAGMENT], shipped)
    return Verdict(False, MISSING_FRAGMENT, MESSAGES[MISSING_FRAGMENT], shipped)


def escape_data(text: str) -> str:
    """Encode the characters a workflow command treats as line or escape syntax."""
    return text.replace("%", "%25").replace("\r", "%0D").replace("\n", "%0A")


def parse_pr_files(pages_json: str) -> list[FileChange]:
    """Flatten `gh api --paginate --slurp` output: an array of pages of file objects."""
    pages = json.loads(pages_json)
    if not isinstance(pages, list):
        raise ValueError(f"expected an array of pages, got {type(pages).__name__}")
    changes: list[FileChange] = []
    for page_index, page in enumerate(pages):
        if not isinstance(page, list):
            raise ValueError(
                f"page {page_index} is {type(page).__name__}, expected an array"
            )
        for item_index, item in enumerate(page):
            where = f"page {page_index} item {item_index}"
            if not isinstance(item, dict):
                raise ValueError(f"{where} is {type(item).__name__}, expected an object")
            filename = item.get("filename")
            status = item.get("status")
            if not isinstance(filename, str) or not isinstance(status, str):
                raise ValueError(f"{where} lacks a string filename and status")
            previous = item.get("previous_filename")
            if previous is not None and not isinstance(previous, str):
                raise ValueError(f"{where} has a non-string previous_filename")
            changes.append(FileChange(filename, status, previous))
    return changes


def _first_line(text: str) -> str:
    lines = text.strip().splitlines()
    return lines[0] if lines else ""


def fetch_changed_files(
    repo: str, number: str, run: Runner
) -> tuple[list[FileChange], str | None]:
    endpoint = f"repos/{repo}/pulls/{number}/files?per_page=100"
    rc, stdout, stderr = run(["gh", "api", "--paginate", "--slurp", endpoint])
    if rc != 0:
        return [], f"gh api {endpoint} exited {rc}: {_first_line(stderr)}"
    return parse_pr_files(stdout), None


def _run_gh(argv: Sequence[str]) -> tuple[int, str, str]:
    """Run gh directly from an argument list, with no shell in between."""
    try:
        done = subprocess.run(
            list(argv),
            capture_output=True,
            text=True,
            encoding="utf-8",
            errors="replace",
            check=False,
            timeout=GH_TIMEOUT_SECONDS,
        )
    except FileNotFoundError:
        return 127, "", "gh not found on PATH"
    except subprocess.TimeoutExpired:
        return 124, "", f"gh timed out after {GH_TIMEOUT_SECONDS}s"
    return done.returncode, done.stdout, done.stderr


def report(verdict: Verdict, out: TextIO) -> int:
    """Print the verdict. Every untrusted value passes through escape_data first."""
    message = escape_data(verdict.message)
    if verdict.ok:
        print(f"fragment-required: PASS ({verdict.code}): {message}", file=out)
        return 0
    print(f"::error::fragment-required: FAIL ({verdict.code}): {message}", file=out)
    if verdict.code == MISSING_FRAGMENT:
        shown = verdict.shipped[:LISTED_PATH_LIMIT]
        more = len(verdict.shipped) - len(shown)
        listed = ", ".join(escape_data(p) for p in shown)
        suffix = f" (+{more} more)" if more else ""
        print(f"::error::fragment-required: shipped-code paths: {listed}{suffix}", file=out)
        print(
            "::error::fragment-required: add a fragment: "
            'CI=true task changie -- new -k <Kind> -b "<one sentence>" -m PR=<number>, '
            "then commit the new .changes/unreleased/*.yaml file "
            "(kinds are declared in .changie.yaml)",
            file=out,
        )
        print(
            "::error::fragment-required: or, if this change needs no changelog entry, "
            "put <!-- changelog-exempt: <reason> --> in the PR body with a non-empty reason",
            file=out,
        )
        for path in verdict.shipped:
            print(f"fragment-required: shipped-code path: {escape_data(path)}", file=out)
    return 1


def main(
    environ: Mapping[str, str] | None = None,
    run: Runner | None = None,
    out: TextIO | None = None,
) -> int:
    environ = os.environ if environ is None else environ
    run = _run_gh if run is None else run
    out = sys.stdout if out is None else out
    files, api_error = fetch_changed_files(
        environ.get("GITHUB_REPOSITORY", ""), environ.get("PR_NUMBER", ""), run
    )
    return report(evaluate(files, environ.get("PR_BODY"), api_error), out)


if __name__ == "__main__":
    raise SystemExit(main())
