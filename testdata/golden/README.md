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

## Running this suite (GOLDEN-01)

**`go test ./...` never runs anything in this directory.** The Go tool
ignores any directory named `testdata` when expanding a `./...` pattern —
`go list ./...` genuinely returns zero packages under `testdata/golden`, so
every "full suite green" run of `go test ./...` silently omits
`TestGoldenParity`, the CLI==MCP byte-identity harness, and the behavioral
corpora entirely. Run it explicitly:

```sh
go test ./testdata/golden/...
```

`.github/workflows/ci.yml`'s `test` job runs this as its own step (separate
from the `go list ./...`-driven step) specifically so this suite's coverage
is not silently lost in CI either.

## Capture provenance

| Field | Value |
|---|---|
| TS `codegraph` CLI version | see `ts-version.txt` |
| Capture date (UTC) | see `ts-version.txt` |
| Capture host | darwin/arm64, `sqlite3` 3.54.0, `jq` 1.8.2 |
| Capture harness | `capture.sh` (re-runnable, `set -euo pipefail`) |

## Corpus (D-06a, extended by D-03)

Two real-world repos, chosen per D-06a ("a compact Go repo... plus the TS
`colbymchenry/codegraph` repo itself, multi-language, exercises the tool
surface broadly"), plus a third purpose-built synthetic corpus added in
Phase 1 (D-03) for behavioral fixtures neither real repo reliably exercises:

| Corpus dir | Repo | Why | Pinned state at capture |
|---|---|---|---|
| `corpus/weft-go/` | [`seanb4t/weft`](https://github.com/seanb4t/weft) (public) | Compact (84 files), mostly-Go repo already available locally; aligns with Phase 2's first parser-target language | commit `f89ae3ea4e4c37509f7302fd4e37986212a72079` (baseline fixtures); behavioral fixtures re-captured at whatever `$WEFT_REPO` HEAD was checked out to at capture time — see `ts-version.txt` for the date |
| `corpus/colbymchenry-codegraph/` | [`colbymchenry/codegraph`](https://github.com/colbymchenry/codegraph) | The original TS CodeGraph project itself — multi-language (TS/JS/Python/Astro/YAML), exercises the tool surface broadly | commit `e871c49a3173a637172f501f21f6a2753ea5a39f` (shallow clone, default branch, at capture time — **not pinned**: every `capture.sh` re-run clones fresh HEAD, so ALL fixtures for this corpus, baseline included, drift with upstream; see `ts-version.txt` for the exact capture date) |
| `corpus/synthetic-parity/` | in-repo only — `testdata/golden/corpus/synthetic-parity/src/` | Purpose-built Go source tree (D-03) exercising the exact v0.1 blind spot: overloaded/same-named symbols, multi-word queries, a `Test*`-heavy weakly-connected cluster, and a structural-beats-lexical ranking case. See `corpus/synthetic-parity/README.md` for the per-file case map (a/b/c/d). **Behavioral fixtures only** — no `status`/`query`/`callers`/`callees`/`impact`/baseline `explore`/`node` fixtures, since this corpus exists solely to drive the new multi-def/multi-word/gate/RWR cases, not general tool-surface coverage. |

Per RESEARCH Open Question 2, `weft-go` and `colbymchenry-codegraph`'s
indexes are rebuilt with `codegraph index --force` immediately before
capture, so `builtWithVersion` in `status.json` reflects the current CLI
(`1.3.1`), not a stale earlier extraction-version stamp.

Only the **captured JSON tool outputs** are committed for `weft-go` and
`colbymchenry-codegraph` — not their corpus source trees. `weft` is
cloned/available separately; `colbymchenry/codegraph` is cloned into a
throwaway temp directory by `capture.sh` and discarded after capture.
`synthetic-parity`'s source tree IS committed (it's purpose-built and lives
nowhere else) — only its local `.codegraph/` index data is excluded (see
`.gitignore`).

Each of `weft-go`/`colbymchenry-codegraph`'s corpus directories contains:

| File | Command | Notes |
|---|---|---|
| `status.json` | `codegraph status <path> --json` | Index stats (file/node/edge counts, `nodesByKind`, `builtWithVersion`) |
| `query.json` | `codegraph query <term> -p <path> --json --limit 5` | FTS5 search results |
| `callers.json` | `codegraph callers <symbol> -p <path> --json --limit 5` | |
| `callees.json` | `codegraph callees <symbol> -p <path> --json --limit 5` | |
| `impact.json` | `codegraph impact <symbol> -p <path> --json` | |
| `explore.json` | `codegraph explore <query> -p <path> --max-files 1` | No native `--json` flag — wrapped as `{"command": ..., "output": "<markdown text>"}` so every fixture is uniformly JSON |
| `node.json` | `codegraph node <symbol> -p <path> -f <file>` | Same wrapping as `explore.json` |

## Behavioral fixtures (Phase 1, D-01/D-03/D-06 extension)

The fixtures above only ever exercised the **template-parity baseline**:
`explore` capped to one file (`--max-files 1`) and `node` pre-disambiguated
to a single definition (`-f <file>`) — never TS's actual multi-file ranking
or multi-definition enumeration (RESEARCH.md Pitfall 3). `capture.sh`'s
`capture_behavioral()` adds four NEW fixtures per corpus (all three:
`weft-go`, `colbymchenry-codegraph`, `synthetic-parity`), on **both** the
CLI and the `codegraph_explore`/`codegraph_node` MCP tools (EXPL-05/NODE-04):

| File | Surface | Command | Proves |
|---|---|---|---|
| `explore-multi.json` | CLI | `codegraph explore <multi-word query> -p <path>` (no `--max-files 1`) | Multi-word `<query...>` tokenization (EXPL-01) ranking across multiple files, not template shape |
| `node-multi.json` | CLI | `codegraph node <name> -p <path>` (no `-f`) | Multi-definition enumeration (NODE-01/02) on a name with 2+ real defs — the "N definitions named X" header, budget, and overflow list |
| `explore-mcp.json` | MCP | `codegraph_explore` tool call, same query as `explore-multi.json` | CLI/MCP output parity (EXPL-05) on the exact same behavioral query |
| `node-mcp.json` | MCP | `codegraph_node` tool call, same symbol as `node-multi.json` | CLI/MCP output parity (NODE-04) on the exact same overloaded symbol |

The MCP-surface fixtures are captured by `testdata/golden/mcp-capture.mjs`,
a small Node harness that spawns the live TS `codegraph serve --mcp` stdio
server (`CODEGRAPH_MCP_TOOLS=explore,node` — TS gates `codegraph_node` off
by default) and drives it via a minimal JSON-RPC 2.0
`initialize`/`notifications/initialized`/`tools/call` handshake. It is
gated on `codegraph --version` == `1.3.1` and hard-fails rather than
silently capturing against an unexpected TS version (D-01).

Per-corpus queries/symbols (see `capture.sh` for the exact invocations):

| Corpus | `node-multi`/`node-mcp` symbol | `explore-multi`/`explore-mcp` query |
|---|---|---|
| `weft-go` | `Run` (10 same-named function/method defs, verified via the TS SQLite index) | `epic worktree` (spans 4+ files) |
| `colbymchenry-codegraph` | `resolve` (27 same-named function defs) | `generated file detection` (spans 4+ files) |
| `synthetic-parity` | `Validate` (2 defs — `accounts/validate.go` + `orders/validate.go`, D-03 case a) | `user account` (tokenizes to match `UserAccountManager`, D-03 case b) |

**These are TS-side goldens ONLY.** The Go-side EXPECTED fixtures (what
`internal/query`'s ported RWR/multi-def pipeline should produce) are NOT
regenerated in this plan — they land in plan 17 (F5), after the D-09
edge-kind expansion and the RWR pipeline itself are implemented. Until then,
these fixtures exist purely as the frozen TS 1.3.1 oracle to diff against.

## Volatile fields (Pitfall 1) — stripped for byte-for-byte reproducibility

A captured fixture that isn't reproducible on a second capture run against
the same source tree is useless as a parity oracle. `capture.sh` strips (or
normalizes) the following before writing any fixture:

| Field / pattern | Where observed | Why it's volatile |
|---|---|---|
| `score` | `query.json` (FTS5 BM25 relevance) | Floating-point, e.g. `114.65185167578747` — reflects index-internal tokenization/ranking state, not stable across reindex |
| `*_at` / `*At` keys (e.g. `updatedAt`, `indexed_at`, `applied_at`, `modified_at`) | node/file records, everywhere | Epoch-millisecond "now" timestamps set at index time |
| `lastIndexed` | `status.json` | Same category — capture-time timestamp |
| `dbSizeBytes` | `status.json` | SQLite WAL/page-fragmentation dependent; not guaranteed byte-stable across reindexes even of identical source. **★ Go-side reversal (D-08, Phase 2):** this strip stays in force for the FROZEN TS oracle fixture above — a stale capture cannot supply a stable byte value, and `golden_test.go`'s shared `volatileKeys` map correctly keeps failing if a TS golden ever re-includes it. But codegraph-go's own `status --json` output DOES emit `dbSizeBytes` (a `filepath.WalkDir` byte sum over `.codegraph/store/` — Pebble has no single-file page-count analog to SQLite's `dbPath`), and `golden_parity_test.go`'s status subtest asserts it for presence, integer type, `> 0`, and well-formed MB rendering ONLY — never cross-run byte stability. The divergence is intentionally WIDER than the original SQLite rationale, not narrower: Pebble's LSM compaction makes the on-disk byte total genuinely nondeterministic across identical reindexes (background compaction can run between two `status` calls with zero source changes), a stronger form of SQLite's WAL/page-fragmentation volatility. This is a documented allowed divergence under Phase 1's D-02 normalized-parity oracle. |
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
`jq`, `git`, and (since Phase 1's behavioral fixtures) `node` — for
`mcp-capture.mjs`'s MCP-surface capture. This is a time-sensitive, one-shot
capture — the TS CLI may drift in behavior or be uninstalled in the future.
If `capture.sh` can no longer run (TS CLI unavailable), the
already-committed fixtures under `corpus/`, `ts-schema.sql`, and
`ts-schema.dump.sql` remain the frozen ground truth; do not attempt to
hand-edit them.

Note: `colbymchenry-codegraph`'s fixtures (baseline AND behavioral) are
**not pinned to a commit** — `capture.sh` clones the default branch's
current HEAD on every run, so re-running it will update those fixtures to
match wherever upstream is at re-capture time (this is expected, not a
regression; see the Corpus table above). `weft-go`'s baseline fixtures are
effectively pinned by the state of the local `$WEFT_REPO` checkout at
capture time. `synthetic-parity` is fully reproducible — it is a fixed,
in-repo source tree.
