#!/bin/sh
# .codex/hooks/codegraph-pretooluse-local.sh — CODEX-05 (Phase 7 D-19..D-22)
#
# The Codex CLI PreToolUse nudge guard (local/project scope). This is the
# TEMPLATE codegraph embeds and renders at install time; the file Codex
# itself trusts and runs is the rendered copy at
# .codex/hooks/codegraph-pretooluse.sh, a deliberately different name so an
# install run inside this very repository never overwrites this template.
#
# Codex has no CLAUDE_PROJECT_DIR-equivalent env var, so this guard derives
# its own repo root from its own invocation path ($0) with three successive
# "%/*" parameter-expansion strips: script -> hooks dir -> .codex -> repo
# root (D-22). Codex invokes hooks with an EMPTY PATH, so no PATH-dependent
# command (dirname, basename, command -v) is used anywhere in this file —
# only POSIX parameter expansion.
#
# 1. An un-indexed repo exits here, before any process starts or stdin is
#    read.
# 2. The codegraph binary path is rendered in at install time as the
#    absolute path of the binary that ran `codegraph install` (D-19): the
#    command registered in hooks.json never carries the binary path, only
#    this guard's own path.
# 3. A missing or non-executable binary exits silently.
# 4. The binary runs as a child with this script's stdin inherited,
#    through the Codex envelope (--harness codex, D-21); this hook never
#    blocks, denies, or reports a hook error — it exits 0 on every path.
hooks_dir=${0%/*}
codex_dir=${hooks_dir%/*}
root=${codex_dir%/*}
[ -d "$root/.codegraph" ] || exit 0
codegraph_bin='@codegraph-exec-path@'
if [ ! -f "$codegraph_bin" ] || [ ! -x "$codegraph_bin" ]; then
  exit 0
fi
"$codegraph_bin" hook pretooluse --harness codex
exit 0
