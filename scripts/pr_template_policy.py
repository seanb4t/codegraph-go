#!/usr/bin/env python3
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

TRUSTED = {"OWNER", "MEMBER", "COLLABORATOR"}

# Paths whose PRs never need a typed template. Matched as prefixes.
EXEMPT_PREFIXES = (
    ".github/",
    "docs/",
    ".planning/",
)
# Root-level machine-maintained manifests belong here by class, not by
# accident: this set is an allowlist of literal names, so every new one is
# gate-tripping until it is added by hand. Keep it in sync with the `case`
# list in .github/workflows/require-issue-link.yml, which reimplements the
# same policy in shell so that job stays dependency-free.
EXEMPT_EXACT = {
    "go.mod",
    "go.sum",
    # Isolated tool modfiles (Phase 10 / DEV-01). Recorded as a known
    # follow-up in issue #18: a PR touching only these is the same class of
    # housekeeping change as one touching go.mod.
    "go.tool.mod",
    "go.tool.sum",
    "go.tool-lint.mod",
    "go.tool-lint.sum",
    # release-please machinery. The manifest is rewritten by the bot on
    # every release, so without this the release PR — which touches exactly
    # {.release-please-manifest.json, CHANGELOG.md} — trips both hygiene
    # gates every single time, and the bot regenerates its body on each run
    # so a manual exemption marker cannot survive.
    ".release-please-manifest.json",
    "release-please-config.json",
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

EXEMPT_MARKER = re.compile(r"<!--\s*pr-template-exempt:\s*(?P<reason>[^->][^>]*?)\s*-->", re.I)
ROUTER_MARKER = re.compile(r"Wrong template\s*—\s*please pick", re.I)


def emit(action: str, reason: str) -> None:
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

    marker = EXEMPT_MARKER.search(body)
    if marker and marker.group("reason").strip():
        emit("pass", f"explicitly exempt: {marker.group('reason').strip()}")
        return 0

    if changed and all(
        p in EXEMPT_EXACT or p.startswith(EXEMPT_PREFIXES) for p in changed
    ):
        emit("pass", "touches only CI, docs, planning, or dependency manifests")
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
