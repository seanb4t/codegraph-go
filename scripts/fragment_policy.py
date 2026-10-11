#!/usr/bin/env python3
"""Decide whether a pull request that changes shipped code adds a changelog fragment.

RED skeleton: signatures only. Every verdict is a placeholder pass.
"""

from __future__ import annotations

import os
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
    return False


def shipped_paths(files: Sequence[FileChange]) -> tuple[str, ...]:
    return ()


def is_fragment(change: FileChange) -> bool:
    return False


def evaluate(
    files: Sequence[FileChange], body: str | None, api_error: str | None = None
) -> Verdict:
    return Verdict(True, NO_SHIPPED_CODE, "red skeleton")


def escape_data(text: str) -> str:
    return text


def parse_pr_files(pages_json: str) -> list[FileChange]:
    return []


def fetch_changed_files(
    repo: str, number: str, run: Runner
) -> tuple[list[FileChange], str | None]:
    return [], None


def report(verdict: Verdict, out: TextIO) -> int:
    return 0


def main(
    environ: Mapping[str, str] | None = None,
    run: Runner | None = None,
    out: TextIO | None = None,
) -> int:
    return 0


if __name__ == "__main__":
    raise SystemExit(main(os.environ, None, sys.stdout))
