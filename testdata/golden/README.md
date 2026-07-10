# Golden Fixtures — TS CodeGraph v1.3.1 Ground Truth (D-06)

This directory captures ground truth from the live **TypeScript CodeGraph
v1.3.1** CLI while it was still installed and available, per RESEARCH.md
decision D-06/D-06a. It is the parity oracle that later phases diff against:

- **Phase 3** (MCP-04): output-shape parity for `codegraph_explore` and
  companion MCP tools.
- **Phase 4**: sync behavior parity.
- **Phase 7**: the one-way SQLite migration reader — `ts-schema.sql` /
  `ts-schema.dump.sql` are the DDL ground truth for reading a TS
  `.codegraph/` index.

This directory does **not** contain a converter. It is capture-only.

## Capture provenance

| Field | Value |
|---|---|
| TS `codegraph` CLI version | see `ts-version.txt` |
| Capture date (UTC) | see `ts-version.txt` |
| Capture host | darwin/arm64, `sqlite3` 3.54.0, `jq` 1.8.2 |
| Capture harness | `capture.sh` (re-runnable, `set -euo pipefail`) |

## Corpus (D-06a)

Two repos, chosen per D-06a ("a compact Go repo... plus the TS
`colbymchenry/codegraph` repo itself, multi-language, exercises the tool
surface broadly"):

| Corpus dir | Repo | Why | Pinned state at capture |
|---|---|---|---|
| `corpus/weft-go/` | [`seanb4t/weft`](https://github.com/seanb4t/weft) (public) | Compact (84 files), mostly-Go repo already available locally; aligns with Phase 2's first parser-target language | commit `f89ae3ea4e4c37509f7302fd4e37986212a72079` |
| `corpus/colbymchenry-codegraph/` | [`colbymchenry/codegraph`](https://github.com/colbymchenry/codegraph) | The original TS CodeGraph project itself — multi-language (TS/JS/Python/Astro/YAML), exercises the tool surface broadly | commit `edb9f2f14cd7394a4d31f94ebc871531ef498ab0` (shallow clone, default branch, at capture time) |

Per RESEARCH Open Question 2, both indexes are rebuilt with
`codegraph index --force` immediately before capture, so `builtWithVersion`
in `status.json` reflects the current CLI (`1.3.1`), not a stale earlier
extraction-version stamp.

Only the **captured JSON tool outputs** are committed here — not the corpus
source trees themselves. `weft` is cloned/available separately;
`colbymchenry/codegraph` is cloned into a throwaway temp directory by
`capture.sh` and discarded after capture.

Each corpus directory contains:

| File | Command | Notes |
|---|---|---|
| `status.json` | `codegraph status <path> --json` | Index stats (file/node/edge counts, `nodesByKind`, `builtWithVersion`) |
| `query.json` | `codegraph query <term> -p <path> --json --limit 5` | FTS5 search results |
| `callers.json` | `codegraph callers <symbol> -p <path> --json --limit 5` | |
| `callees.json` | `codegraph callees <symbol> -p <path> --json --limit 5` | |
| `impact.json` | `codegraph impact <symbol> -p <path> --json` | |
| `explore.json` | `codegraph explore <query> -p <path> --max-files 1` | No native `--json` flag — wrapped as `{"command": ..., "output": "<markdown text>"}` so every fixture is uniformly JSON |
| `node.json` | `codegraph node <symbol> -p <path> -f <file>` | Same wrapping as `explore.json` |

## Volatile fields (Pitfall 1) — stripped for byte-for-byte reproducibility

A captured fixture that isn't reproducible on a second capture run against
the same source tree is useless as a parity oracle. `capture.sh` strips (or
normalizes) the following before writing any fixture:

| Field / pattern | Where observed | Why it's volatile |
|---|---|---|
| `score` | `query.json` (FTS5 BM25 relevance) | Floating-point, e.g. `114.65185167578747` — reflects index-internal tokenization/ranking state, not stable across reindex |
| `*_at` / `*At` keys (e.g. `updatedAt`, `indexed_at`, `applied_at`, `modified_at`) | node/file records, everywhere | Epoch-millisecond "now" timestamps set at index time |
| `lastIndexed` | `status.json` | Same category — capture-time timestamp |
| `dbSizeBytes` | `status.json` | SQLite WAL/page-fragmentation dependent; not guaranteed byte-stable across reindexes even of identical source |
| `projectPath`, `indexPath` | `status.json` | Machine-local absolute paths — not portable across capture machines. Normalized to the literal string `<CORPUS_PATH>` (not deleted, since the key's presence/shape is still meaningful) |
| 13-digit epoch-ms integers (optionally with a decimal fraction, e.g. `1783108606938.7051`) | `ts-schema.dump.sql` sample rows (`updated_at`, `indexed_at`, `applied_at`, `modified_at` columns) | Same timestamp volatility as above, but embedded in raw SQL `INSERT` literals rather than JSON keys — normalized via regex to the literal token `<EPOCH_MS>` |

This was verified empirically: `capture.sh` was run twice back-to-back
against both corpus repos (with a full `index --force` rebuild in between)
and every fixture file — JSON and SQL — diffed **byte-for-byte identical**
across both runs.

`golden_test.go`'s `TestGoldenFixturesExist` encodes the JSON half of this
guarantee as an automated invariant: it fails if any corpus JSON fixture
re-introduces a raw `score` field or a `*_at`/`*At` timestamp key.

## Historical bug note (Pitfall 2) — edge-dedup, issue #1034

`ts-schema.sql` contains:

```sql
CREATE UNIQUE INDEX idx_edges_identity
  ON edges(source, target, kind, IFNULL(line, -1), IFNULL(col, -1));
```

This unique index is the TS project's fix for a real, historical bug
(upstream issue #1034): an `INSERT OR IGNORE` into `edges` with **no**
`UNIQUE` constraint to conflict on behaved like a plain `INSERT`, so two
indexing/resolve passes over the same source could silently double-insert
the same logical edge — inflating edge counts and polluting
callers/impact results.

**Why this matters for Phase 2:** the Pebble edge key design (`e/<src>/<kind>/<dst>`,
per D-03) does **not** include line/col in the key — meaning it dedupes at a
coarser grain than the TS schema's unique index does. That is almost
certainly the *desired* behavior for most edge kinds (repeated
sync/resolve passes should be idempotent, not append-only), but it also
means two structurally distinct call sites between the same (source, kind,
target) triple will collide and overwrite in the new store, where TS's
`(source, target, kind, line, col)` index would have kept them distinct.
Phase 2's edge-identity design should explicitly decide (and document)
whether any edge kinds need line/col folded into the key to avoid losing
multi-call-site information that TS's schema preserved.

## Re-running the capture

```sh
./testdata/golden/capture.sh
# or, to point at a different Go corpus repo:
WEFT_REPO=/path/to/a/small/committable/go/repo ./testdata/golden/capture.sh
```

Requires the live TS `codegraph` CLI (v1.3.1+) on `PATH`, plus `sqlite3`,
`jq`, and `git`. This is a time-sensitive, one-shot capture — the TS CLI may
drift in behavior or be uninstalled in the future. If `capture.sh` can no
longer run (TS CLI unavailable), the already-committed fixtures under
`corpus/`, `ts-schema.sql`, and `ts-schema.dump.sql` remain the frozen
ground truth; do not attempt to hand-edit them.
