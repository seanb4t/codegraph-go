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
import re
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

# Mirrors pr_template_policy.EXEMPT_MARKER with only the marker name changed (GATE-02):
# case-insensitive, and a reason that is empty after strip does not exempt.
CHANGELOG_EXEMPT_MARKER = re.compile(r"<!--\s*changelog-exempt:\s*(?P<reason>[^->][^>]*?)\s*-->", re.I)

API_ERROR = "api-error"
EMPTY_FILE_LIST = "empty-file-list"
BODY_UNREADABLE = "body-unreadable"
EXEMPT = "exempt"
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
    EMPTY_FILE_LIST: (
        "GitHub returned an empty changed-file list for this PR, "
        "so the gate fails closed rather than pass a change it cannot see"
    ),
    BODY_UNREADABLE: (
        "could not read this PR's body (PR_BODY is not set in the step environment), "
        "so an exemption marker cannot be checked and the gate fails closed"
    ),
}
EXEMPT_MESSAGE_PREFIX = "exempt by a changelog-exempt marker: "
CONTEXT_MESSAGE = (
    "GITHUB_REPOSITORY or PR_NUMBER is missing or malformed in the step environment"
)

REPO_PATTERN = re.compile(r"[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+")
PR_NUMBER_PATTERN = re.compile(r"[1-9][0-9]*")

LISTED_PATH_LIMIT = 20

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
    """Sorted, deduplicated shipped-code paths, counting the old path of a rename."""
    found: set[str] = set()
    for change in files:
        for path in (change.path, change.previous_path):
            if path is not None and is_shipped_code(path):
                found.add(path)
    return tuple(sorted(found))


def is_fragment(change: FileChange) -> bool:
    """A newly added file under .changes/unreleased/ that is not the keep file."""
    return (
        change.status.lower() == FRAGMENT_STATUS
        and change.path.startswith(FRAGMENT_DIR)
        and len(change.path) > len(FRAGMENT_DIR)
        and change.path.rsplit("/", 1)[-1] != FRAGMENT_KEEP_FILE
    )


def find_exemption(body: str) -> str | None:
    """The stripped reason of the first changelog-exempt marker, or None.

    Mirrors pr_template_policy.main: the first marker decides, and a reason that is
    empty after strip does not exempt.
    """
    match = CHANGELOG_EXEMPT_MARKER.search(body)
    if match is None:
        return None
    return match.group("reason").strip() or None


def evaluate(
    files: Sequence[FileChange], body: str | None, api_error: str | None = None
) -> Verdict:
    """Pure verdict over a complete changed-file list. Never mutates its input.

    Branch order is the precedence: api-error, empty-file-list, body-unreadable,
    exempt, no-shipped-code, has-fragment, missing-fragment.
    """
    if api_error is not None:
        return Verdict(False, API_ERROR, f"{MESSAGES[API_ERROR]}: {api_error}")
    if not files:
        return Verdict(False, EMPTY_FILE_LIST, MESSAGES[EMPTY_FILE_LIST])
    if body is None:
        return Verdict(False, BODY_UNREADABLE, MESSAGES[BODY_UNREADABLE])
    reason = find_exemption(body)
    if reason is not None:
        return Verdict(True, EXEMPT, EXEMPT_MESSAGE_PREFIX + reason, shipped_paths(files))
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
    """Fetch every changed file and prove the list is complete. Never raises.

    The count GitHub reports for the pull request must equal the number of files
    fetched, so a truncated listing (GitHub lists at most 3000 files) fails closed.
    """
    meta_endpoint = f"repos/{repo}/pulls/{number}"
    rc, stdout, stderr = run(["gh", "api", meta_endpoint])
    if rc != 0:
        return [], f"gh api {meta_endpoint} exited {rc}: {_first_line(stderr)}"
    try:
        meta = json.loads(stdout)
    except ValueError as exc:
        return [], f"malformed pull request metadata from GitHub: {exc}"
    expected = meta.get("changed_files") if isinstance(meta, dict) else None
    if isinstance(expected, bool) or not isinstance(expected, int) or expected < 0:
        return [], (
            "malformed pull request metadata from GitHub: "
            "changed_files is not a non-negative integer"
        )

    endpoint = f"{meta_endpoint}/files?per_page=100"
    rc, stdout, stderr = run(["gh", "api", "--paginate", "--slurp", endpoint])
    if rc != 0:
        return [], f"gh api {endpoint} exited {rc}: {_first_line(stderr)}"
    try:
        files = parse_pr_files(stdout)
    except ValueError as exc:
        return [], f"malformed file list from GitHub: {exc}"
    if len(files) != expected:
        return [], (
            f"file list incomplete: fetched {len(files)} of {expected} changed files "
            "(GitHub lists at most 3000 files per pull request)"
        )
    return files, None


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
            timeout=120,
        )
    except FileNotFoundError:
        return 127, "", "gh not found on PATH"
    except subprocess.TimeoutExpired:
        return 124, "", "gh timed out after 120s"
    return done.returncode, done.stdout, done.stderr


def report(verdict: Verdict, out: TextIO) -> int:
    """Print the verdict. Every untrusted value passes through escape_data first."""
    message = escape_data(verdict.message)
    if verdict.ok:
        print(f"fragment-required: PASS ({verdict.code}): {message}", file=out)
        if verdict.code == EXEMPT:
            reason = verdict.message.removeprefix(EXEMPT_MESSAGE_PREFIX)
            print(f"::notice::fragment-required: exempt: {escape_data(reason)}", file=out)
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
    repo = environ.get("GITHUB_REPOSITORY", "")
    number = environ.get("PR_NUMBER", "")
    files: list[FileChange] = []
    api_error: str | None = None
    if REPO_PATTERN.fullmatch(repo) and PR_NUMBER_PATTERN.fullmatch(number):
        files, api_error = fetch_changed_files(repo, number, run)
    else:
        api_error = CONTEXT_MESSAGE
    # An absent PR_BODY is unreadable; a present empty string is a readable empty body.
    return report(evaluate(files, environ.get("PR_BODY"), api_error), out)


if __name__ == "__main__":
    raise SystemExit(main())
