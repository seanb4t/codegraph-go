#!/bin/sh
# .codex/hooks/codegraph-pretooluse-global.sh — CODEX-05 (Phase 7 D-19..D-22)
#
# The Codex CLI PreToolUse nudge guard (global/user scope). This is the
# TEMPLATE codegraph embeds and renders at install time; the file Codex
# itself trusts and runs is the rendered copy at
# ~/.codex/hooks/codegraph-pretooluse.sh.
#
# Unlike the local guard, a global hook's cwd is simply the session's own
# working directory (D-22) — there is no repo-relative path to derive, so
# this guard checks $PWD directly rather than walking up from its own
# invocation path.
#
# 1. An un-indexed repo (relative to $PWD) exits here, before any process
#    starts or stdin is read.
# 2. The codegraph binary path is rendered in at install time as the
#    absolute path of the binary that ran `codegraph install` (D-19): the
#    command registered in hooks.json never carries the binary path, only
#    a single-quoted absolute path to this guard (D-20).
# 3. A missing or non-executable binary exits silently.
# 4. The binary runs as a child with this script's stdin inherited,
#    through the Codex envelope (--harness codex, D-21); this hook never
#    blocks, denies, or reports a hook error — it exits 0 on every path.
[ -d "${PWD:-.}/.codegraph" ] || exit 0
codegraph_bin='@codegraph-exec-path@'
if [ ! -f "$codegraph_bin" ] || [ ! -x "$codegraph_bin" ]; then
  exit 0
fi
"$codegraph_bin" hook pretooluse --harness codex
exit 0
