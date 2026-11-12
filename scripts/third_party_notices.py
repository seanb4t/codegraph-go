#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.12"
# dependencies = []
# ///
"""Write THIRD_PARTY_NOTICES.txt from a go-licenses report and the web bundle's notices.

`task licenses:notices` and `task licenses:drift` run this script. The Go half
comes from `go-licenses report` with a template that prints one tab-separated
row per library: name, version, license, license file path. The Taskfile runs
it once per release platform and passes every row on stdin, so a module that
only one platform links is still listed. The web half is
web/build/THIRD_PARTY_NOTICES.txt, which the Vite build writes.

Libraries that share one license text are listed together above that text, so
the Apache-2.0 text appears a few times, not once per module.

go-licenses reports `Unknown` when it cannot classify a license file. Each such
library must match a prefix in SHORT_FORM, or the script fails and names it. A
library with no license is a question for a person, never a blank entry.

tests/test_third_party_notices.py proves the grouping and the failures.
"""

from __future__ import annotations

import argparse
import hashlib
import subprocess
import sys
from collections.abc import Callable, Iterable
from pathlib import Path

MAIN_MODULE = "github.com/seanb4t/codegraph-go"

# Modules whose LICENSE file holds only the Apache-2.0 short-form header, which
# go-licenses cannot classify. The value is the license that header names.
SHORT_FORM = {
    "github.com/cyberphone/json-canonicalization": "Apache-2.0",
    "github.com/in-toto/attestation": "Apache-2.0",
    "github.com/in-toto/in-toto-golang": "Apache-2.0",
}

RULE = "-" * 40


class NoticesError(Exception):
    pass


def module_dir(module: str) -> Path:
    out = subprocess.run(
        ["go", "list", "-m", "-f", "{{.Dir}}", module],
        check=True,
        capture_output=True,
        text=True,
    ).stdout.strip()
    if not out:
        raise NoticesError(f"go list printed no directory for module {module}")
    return Path(out)


def short_form_module(name: str) -> str | None:
    for module in SHORT_FORM:
        if name == module or name.startswith(module + "/"):
            return module
    return None


def parse_rows(lines: Iterable[str]) -> dict[tuple[str, str], tuple[str, str]]:
    """Return {(name, version): (license, path)} without the main module."""
    rows: dict[tuple[str, str], tuple[str, str]] = {}
    for line in lines:
        line = line.rstrip("\n")
        if not line:
            continue
        parts = line.split("\t")
        if len(parts) != 4:
            raise NoticesError(f"go-licenses row does not have 4 fields: {line!r}")
        name, version, license_name, path = parts
        if name == MAIN_MODULE or name.startswith(MAIN_MODULE + "/"):
            continue
        rows[(name, version)] = (license_name, path)
    return rows


def normalize(text: str) -> str:
    lines = [ln.rstrip() for ln in text.replace("\r\n", "\n").replace("\r", "\n").split("\n")]
    return "\n".join(lines).strip("\n")


def go_section(
    rows: dict[tuple[str, str], tuple[str, str]],
    find_module_dir: Callable[[str], Path] = module_dir,
) -> str:
    if not rows:
        raise NoticesError("the go-licenses report has no third-party rows")
    unknown: list[str] = []
    groups: dict[str, tuple[str, list[str]]] = {}
    for (name, version), (license_name, path) in sorted(rows.items()):
        if license_name == "Unknown":
            module = short_form_module(name)
            if module is None:
                unknown.append(name)
                continue
            license_name = SHORT_FORM[module]
            path = str(find_module_dir(module) / "LICENSE")
        text = normalize(Path(path).read_text(encoding="utf-8", errors="replace"))
        if not text:
            raise NoticesError(f"the license file of {name} is empty: {path}")
        key = hashlib.sha256(text.encode()).hexdigest()
        groups.setdefault(key, (text, []))[1].append(f"{name} {version} ({license_name})")
    if unknown:
        raise NoticesError(
            "go-licenses found no license for: "
            + ", ".join(unknown)
            + ". Read each module's license, then add it to SHORT_FORM or replace the module."
        )
    blocks = [
        "\n".join(members) + "\n\n" + text
        for text, members in sorted(groups.values(), key=lambda g: g[1][0])
    ]
    return f"\n\n{RULE}\n\n".join(blocks)


def render(go_rows: dict[tuple[str, str], tuple[str, str]], web_notices: str, **kw) -> str:
    web = normalize(web_notices)
    if "Name: " not in web:
        raise NoticesError("the web notices file lists no package")
    header = (
        "THIRD-PARTY NOTICES\n\n"
        "codegraph includes the software below. Each entry names the component,\n"
        "its version and its license, and the license text follows.\n"
        "`task licenses:notices` writes this file. Do not edit it by hand.\n"
    )
    return (
        f"{header}\n{'=' * 40}\nGo modules linked into the binary\n{'=' * 40}\n\n"
        f"{go_section(go_rows, **kw)}\n\n"
        f"{'=' * 40}\nPackages in the embedded graph UI\n{'=' * 40}\n\n{web}\n"
    )


def main(argv: list[str] | None = None) -> int:
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument("--web", type=Path, required=True, help="notices file of the web bundle")
    ap.add_argument("--out", type=Path, required=True, help="file to write or compare")
    ap.add_argument("--check", action="store_true", help="compare --out, write nothing")
    args = ap.parse_args(argv)
    try:
        rows = parse_rows(sys.stdin)
        text = render(rows, args.web.read_text(encoding="utf-8"))
    except (NoticesError, OSError, subprocess.CalledProcessError) as err:
        print(f"third_party_notices: {err}", file=sys.stderr)
        return 1
    if args.check:
        current = args.out.read_text(encoding="utf-8") if args.out.exists() else ""
        if current != text:
            print(
                f"third_party_notices: {args.out} is stale ({len(rows)} Go libraries compared). "
                "Run `task licenses:notices` and commit the result.",
                file=sys.stderr,
            )
            return 1
        print(f"third_party_notices: {args.out} is current ({len(rows)} Go libraries compared)")
        return 0
    args.out.write_text(text, encoding="utf-8")
    print(f"third_party_notices: wrote {args.out} ({len(rows)} Go libraries)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
