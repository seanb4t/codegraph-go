#!/bin/sh
# .claude/hooks/pretooluse-nudge.sh — NUDGE-03 / NUDGE-04 (Phase 6 D-01, D-01b, D-04)
#
# The Claude Code PreToolUse nudge guard. This one file is both the template
# codegraph embeds and renders at install time and this repository's own
# dogfood guard.
#
# 1. An un-indexed repo exits here, before any process starts or stdin is
#    read (D-04, the same check as session-nudge.sh).
# 2. The codegraph binary path is rendered in at install time as the
#    absolute path of the binary that ran `codegraph install` (D-01b). Only
#    this unrendered repository copy falls back to the codegraph on PATH;
#    an installed guard never consults PATH.
# 3. A missing or non-executable binary exits silently.
# 4. The binary runs as a child with this script's stdin, and the script
#    exits 0 whatever it did: this hook never blocks, denies or reports a
#    hook error.
[ -d "${CLAUDE_PROJECT_DIR:-.}/.codegraph" ] || exit 0
codegraph_bin='@codegraph-exec-path@'
case $codegraph_bin in
@*@) codegraph_bin=$(command -v codegraph 2>/dev/null) || exit 0 ;;
esac
if [ ! -f "$codegraph_bin" ] || [ ! -x "$codegraph_bin" ]; then
  exit 0
fi
"$codegraph_bin" hook pretooluse
exit 0
