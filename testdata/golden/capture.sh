#!/usr/bin/env bash
# testdata/golden/capture.sh
#
# Reproducible capture harness for D-06 ground truth: the TS CodeGraph v1.3.1
# `.codegraph/` SQLite schema DDL, plus golden JSON snapshots of
# `codegraph_explore` and companion tool outputs on a small, pinned corpus
# (D-06a: a compact Go repo + the colbymchenry/codegraph TS repo itself).
#
# This is a *capture* tool, not a converter: it produces the fixtures under
# testdata/golden/ that Phase 3 (MCP-04), Phase 4 (sync), and Phase 7
# (migration reader) diff parity against. See README.md for which fields are
# volatile and why they are stripped (RESEARCH.md Pitfall 1).
#
# Requires: the live TS `codegraph` CLI (v1.3.1+), sqlite3, jq, git.
#
# Re-run: ./testdata/golden/capture.sh
#   WEFT_REPO=/path/to/small/go/repo ./testdata/golden/capture.sh   (override the Go corpus repo)

set -euo pipefail

GOLDEN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CORPUS_DIR="$GOLDEN_DIR/corpus"
CODEGRAPH_BIN="${CODEGRAPH_BIN:-codegraph}"
WEFT_REPO="${WEFT_REPO:-/Volumes/Code/github.com/seanb4t/weft}"
TS_REPO_URL="${TS_REPO_URL:-https://github.com/colbymchenry/codegraph.git}"

for bin in "$CODEGRAPH_BIN" sqlite3 jq git; do
  command -v "$bin" >/dev/null 2>&1 || {
    echo "capture.sh: required tool '$bin' not found on PATH" >&2
    exit 1
  }
done

if [ ! -d "$WEFT_REPO" ]; then
  echo "capture.sh: Go corpus repo not found at $WEFT_REPO" >&2
  echo "capture.sh: set WEFT_REPO=/path/to/a/small/committable/go/repo and re-run" >&2
  exit 1
fi

mkdir -p "$CORPUS_DIR"

# --- version + capture date stamp (D-06: record exact TS version + date) ---
CG_VERSION="$("$CODEGRAPH_BIN" version)"
CAPTURE_DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
cat >"$GOLDEN_DIR/ts-version.txt" <<EOF
codegraph_version=${CG_VERSION}
capture_date=${CAPTURE_DATE}
EOF

# --- determinism-stripping filters ------------------------------------------
# Pitfall 1 (RESEARCH.md): the FTS5 BM25 `score` float and every `*_at`/`*At`
# epoch-millisecond timestamp differ on every reindex, even of byte-identical
# source. `dbSizeBytes` (WAL/page-fragmentation dependent) and the
# machine-local `projectPath`/`indexPath` absolute paths are additional
# non-deterministic/non-portable fields discovered while building this
# harness -- stripped for the same reason: byte-for-byte reproducibility.
JSON_STRIP_FILTER='
  walk(if type == "object" then
    with_entries(select(.key as $k |
      (($k|test("_at$")) or ($k|test("At$")) or ($k=="score") or ($k=="lastIndexed") or ($k=="dbSizeBytes") | not)
    ))
  else . end)
  | walk(if type == "object" then
      (if has("projectPath") then .projectPath = "<CORPUS_PATH>" else . end)
      | (if has("indexPath") then .indexPath = "<CORPUS_PATH>" else . end)
    else . end)
'

strip_json() { jq "$JSON_STRIP_FILTER"; }

# The same volatile epoch-ms timestamps appear as raw INTEGER values in the
# SQL dump sample rows (updated_at/indexed_at/applied_at columns) -- 13-digit
# numbers in the 2020s epoch-ms range. Normalize them for the same reason.
strip_sql_timestamps() { sed -E 's/[0-9]{13}(\.[0-9]+)?/<EPOCH_MS>/g'; }

wrap_text() {
  # Wrap a raw CLI text/markdown command's output (explore, node -- neither
  # has a --json flag) in a JSON envelope so every corpus fixture is
  # uniformly JSON, per D-06's "golden JSON snapshots."
  local command="$1"
  jq -Rs --arg command "$command" '{command: $command, output: .}'
}

capture_repo() {
  local name="$1" path="$2" symbol="$3" symbol_file="$4" query_term="$5" explore_query="$6"
  local out="$CORPUS_DIR/$name"
  mkdir -p "$out"

  echo "capture.sh: [$name] force reindex (avoids stale builtWithVersion stamp)..." >&2
  "$CODEGRAPH_BIN" index --force -q "$path" >/dev/null

  "$CODEGRAPH_BIN" status "$path" --json | strip_json >"$out/status.json"
  "$CODEGRAPH_BIN" query "$query_term" -p "$path" --json --limit 5 | strip_json >"$out/query.json"
  "$CODEGRAPH_BIN" callers "$symbol" -p "$path" --json --limit 5 | strip_json >"$out/callers.json"
  "$CODEGRAPH_BIN" callees "$symbol" -p "$path" --json --limit 5 | strip_json >"$out/callees.json"
  "$CODEGRAPH_BIN" impact "$symbol" -p "$path" --json | strip_json >"$out/impact.json"
  "$CODEGRAPH_BIN" explore "$explore_query" -p "$path" --max-files 1 |
    wrap_text "explore \"$explore_query\" -p $name --max-files 1" >"$out/explore.json"
  "$CODEGRAPH_BIN" node "$symbol" -p "$path" -f "$symbol_file" |
    wrap_text "node \"$symbol\" -p $name -f $symbol_file" >"$out/node.json"

  echo "capture.sh: [$name] captured $(ls "$out" | wc -l | tr -d ' ') fixture files" >&2
}

# --- schema DDL capture (from the Go corpus index; D-06) --------------------
echo "capture.sh: capturing TS schema DDL from $WEFT_REPO ..." >&2
"$CODEGRAPH_BIN" index --force -q "$WEFT_REPO" >/dev/null
WEFT_DB="$(find "$WEFT_REPO/.codegraph" -maxdepth 1 -name '*.db' | head -1)"
if [ -z "$WEFT_DB" ]; then
  echo "capture.sh: no .codegraph/*.db found under $WEFT_REPO" >&2
  exit 1
fi

sqlite3 "$WEFT_DB" '.schema' >"$GOLDEN_DIR/ts-schema.sql"

sqlite3 "$WEFT_DB" <<'SQL' | strip_sql_timestamps >"$GOLDEN_DIR/ts-schema.dump.sql"
.mode insert nodes
SELECT * FROM nodes ORDER BY id LIMIT 5;
.mode insert edges
SELECT * FROM edges ORDER BY id LIMIT 5;
.mode insert files
SELECT * FROM files ORDER BY path LIMIT 5;
.mode insert schema_versions
SELECT * FROM schema_versions;
SQL

# --- golden corpus capture (D-06a) ------------------------------------------
capture_repo "weft-go" "$WEFT_REPO" "mergeStyle" "internal/cli/finish.go" "main" "main function"

TMP_ROOT="$(mktemp -d)"
trap 'rm -rf "$TMP_ROOT"' EXIT
TS_REPO="$TMP_ROOT/colbymchenry-codegraph"
echo "capture.sh: cloning $TS_REPO_URL ..." >&2
git clone --depth 1 --quiet "$TS_REPO_URL" "$TS_REPO"
"$CODEGRAPH_BIN" init "$TS_REPO" >/dev/null 2>&1
capture_repo "colbymchenry-codegraph" "$TS_REPO" "searchNodes" "src/db/queries.ts" "search" "search nodes"

echo "capture.sh: done. Fixtures written under $CORPUS_DIR" >&2
