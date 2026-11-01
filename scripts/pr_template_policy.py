#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.12"
# dependencies = []
# ///
"""Decide whether a pull request body follows one of the typed PR templates.

Adapted from open-gsd/gsd-core's scripts/pr-template-policy.cjs, rewritten in
Python: this repository has no Node toolchain of its own, and the .cjs original
would have been immediate migration debt.

Reads from the environment so untrusted PR content never reaches a shell:

    PR_BODY              the pull request body
    AUTHOR_ASSOCIATION   OWNER | MEMBER | COLLABORATOR | CONTRIBUTOR | NONE ...
    CHANGED_FILES        newline-separated repo-relative paths

Writes `action` and `result` to $GITHUB_OUTPUT. Exit status is always 0 — the
workflow decides what to do with the verdict, so a policy opinion never looks
like a crashed step.

Verdicts:
    pass   body follows a typed template, or is exempt
    warn   malformed, but the author is trusted — comment, do not fail
    close  malformed from an untrusted author — comment and fail the check
"""

from __future__ import annotations

import json
import os
import re
import sys
from collections.abc import Sequence

TRUSTED = {"OWNER", "MEMBER", "COLLABORATOR"}

# Paths whose PRs never need a typed template. Matched as prefixes.
# .changes/ is the changie release PR's own tree: that PR touches only .changes/**
# and CHANGELOG.md, and its body is the rendered notes, regenerated on every run,
# so a manual exemption marker cannot survive there (D-05, GRD-15).
# openspec/ holds spec proposals; the code that implements one arrives in its
# own PR, which does need a template.
EXEMPT_PREFIXES = (
    ".github/",
    "docs/",
    "openspec/",
    ".changes/",
)
# Root-level machine-maintained manifests belong here by class, not by
# accident: this set is an allowlist of literal names, so every new one is
# gate-tripping until it is added by hand. The `case` list in
# .github/workflows/require-issue-link.yml reimplements the same policy in shell
# so that job stays dependency-free; test_hygiene_allowlists_mirror_require_issue_link
# asserts that both lists stay equal.
EXEMPT_EXACT = {
    "go.mod",
    "go.sum",
    # mise.lock only: a re-lock is housekeeping, but mise.toml pins the release
    # toolchain and stays under the gates.
    "mise.lock",
    ".gitignore",
    "README.md",
    "CONTRIBUTING.md",
    "CODE_OF_CONDUCT.md",
    "SECURITY.md",
    "NOTICE",
    "LICENSE",
    "CHANGELOG.md",
}

# A body counts as templated if it carries the headings the typed templates use.
# Deliberately loose: contributors reflow and reword, and a check that demands
# byte-identical boilerplate fails honest PRs while catching nothing real.
TEMPLATE_SIGNALS = (
    re.compile(r"^##\s+What changed", re.I | re.M),
    re.compile(r"^##\s+What this adds", re.I | re.M),
    re.compile(r"^##\s+The bug", re.I | re.M),
    re.compile(r"^##\s+What was wrong with it", re.I | re.M),
    re.compile(r"^##\s+Required checklist", re.I | re.M),
    re.compile(r"^##\s+Checklist", re.I | re.M),
)

# The reason is captured whole and validated after the match (stripped, non-empty,
# not starting with `-`). The old pattern spent its backtracking budget on `\s*`
# beside `[^>]*?` and ran in cubic time on a body of spaces (WR-01). This one
# runs once per `<!--` opener.
EXEMPT_MARKER = re.compile(r"<!--\s*pr-template-exempt:(?P<reason>[^>]*?)-->", re.I)
ROUTER_MARKER = re.compile(r"Wrong template\s*—\s*please pick", re.I)


# The pull-request files REST listing returns at most this many files (#106).
FILE_LIST_LIMIT = 3000


def is_housekeeping(paths: Sequence[str]) -> bool:
    """True when the list is non-empty and every path is on the exempt allowlist.

    Comparison is exact and case-sensitive. A prefix matches only as written, so
    `.changesX/a.md` is not under `.changes/`. A list at FILE_LIST_LIMIT may be
    cut short, so it is never housekeeping.
    """
    return bool(paths) and len(paths) < FILE_LIST_LIMIT and all(
        p in EXEMPT_EXACT or p.startswith(EXEMPT_PREFIXES) for p in paths
    )


def find_exemption(body: str) -> str | None:
    """The stripped reason of the first pr-template-exempt marker, or None.

    The first marker decides. A reason that is empty after strip, or starts
    with `-`, does not exempt. task check:fragment-required reads the
    changelog-exempt marker with the same rules.
    """
    match = EXEMPT_MARKER.search(body)
    if match is None:
        return None
    reason = match.group("reason").strip()
    if not reason or reason.startswith("-"):
        return None
    return reason


def emit(action: str, reason: str) -> None:
    # A reason can carry PR-body text. A line that starts with `::` is a
    # workflow command, so the reason becomes one line with no leading `::` (#106).
    reason = " ".join(reason.splitlines())
    while reason.startswith("::"):
        reason = reason[2:]
    payload = json.dumps({"action": action, "reason": reason})
    out = os.environ.get("GITHUB_OUTPUT")
    if out:
        with open(out, "a", encoding="utf-8") as fh:
            fh.write(f"action={action}\n")
            fh.write(f"result={payload}\n")
    print(f"{action}: {reason}", file=sys.stderr)


def main() -> int:
    body = os.environ.get("PR_BODY") or ""
    assoc = (os.environ.get("AUTHOR_ASSOCIATION") or "NONE").upper()
    changed = [p.strip() for p in (os.environ.get("CHANGED_FILES") or "").splitlines() if p.strip()]

    reason = find_exemption(body)
    if reason is not None:
        emit("pass", f"explicitly exempt: {reason}")
        return 0

    if is_housekeeping(changed):
        emit("pass", "touches only CI, docs, specs, or dependency manifests")
        return 0

    if not body.strip():
        return fail(assoc, "the PR body is empty")

    if ROUTER_MARKER.search(body):
        return fail(
            assoc,
            "the default router template was submitted unchanged — pick fix, "
            "enhancement, or feature",
        )

    if not any(sig.search(body) for sig in TEMPLATE_SIGNALS):
        return fail(
            assoc,
            "the body does not match any typed template (no recognised section "
            "heading found)",
        )

    emit("pass", "body follows a typed template")
    return 0


def fail(assoc: str, reason: str) -> int:
    emit("warn" if assoc in TRUSTED else "close", reason)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
