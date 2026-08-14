# Golden fixtures and the behavioral corpus

This directory holds the legacy schema ground truth and the golden test
suite that guards it. The TS-era capture path and the external,
network-fetched corpora it once captured from are retired as of Phase 2
(FIXT-04). The purpose-built
behavioral corpus now lives at the repo root as `corpus/behavioral/`
(D-03/D-04).

- `ts-schema.sql` / `ts-schema.dump.sql` / `ts-version.txt` — the legacy
  SQLite schema DDL + version record that Phase 7's one-way migration reader
  treats as ground truth for reading a TS `.codegraph/` index. These are
  never deleted or reworded.
- `corpus/behavioral/**` (at the repo root, outside `testdata/`) — the
  committed, always-in-repo behavioral corpus: a small Go source tree
  (`src/`) whose four purpose-built cases exercise overloaded symbols,
  multi-word queries, a `Test*`-heavy weakly-connected cluster, and
  structural-beats-lexical ranking (the case map is committed as
  `CASES.json`), plus the two Go-captured goldens
  (`go-explore-multi.json` / `go-node-multi.json`).
- The language behavioral tests (`behavioral_{java,python,csharp,tsjs}_test.go`)
  self-skip unless a real corpus for that language is configured.

## Running this suite (GOLDEN-01)

**`go test ./...` never runs anything in this directory.** The Go tool
ignores any directory named `testdata` when expanding a `./...` pattern —
`go list ./...` genuinely returns zero packages under `testdata/golden`, so
every "full suite green" run of `go test ./...` silently omits the golden
harness. Run it explicitly:

```sh
go test ./testdata/golden/...
```

`.github/workflows/ci.yml`'s `test` job runs this as its own step (separate
from the `go list ./...`-driven step) so this suite's coverage is not
silently lost in CI either.

## Volatile fields (Pitfall 1) — stripped for byte-for-byte reproducibility

A captured fixture that isn't reproducible on a second capture run against
the same source tree is useless as a regression oracle. The legacy capture
path stripped (or normalized) the following before writing any fixture, and
the invariants live on in `golden_test.go`'s `TestGoldenFixturesExist`:

| Field / pattern | Where observed | Why it's volatile |
|---|---|---|
| `score` | `query.json` (FTS5 BM25 relevance) | Floating-point "now" ranking state, not stable across reindex |
| `*_at` / `*At` keys (e.g. `updatedAt`, `indexed_at`, `applied_at`, `modified_at`) | node/file records, everywhere | Epoch-millisecond "now" timestamps set at index time |
| `lastIndexed` | `status.json` | Capture-time timestamp |
| `dbSizeBytes` | `status.json` | Sizes vary across reindexes; codegraph-go's own `status --json` DOES emit it (D-08) and `behavioral_test.go` asserts it for presence/integer type/`> 0`/well-formed MB rendering only — never byte-stability |
| `projectPath`, `indexPath` | `status.json` | Machine-local absolute paths |

`golden_test.go`'s `TestGoldenFixturesExist` encodes the JSON half of this
guarantee as an automated invariant: it fails if a corpus JSON fixture
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

**Why this matters for the Go port:** the Pebble edge key design
(`e/<src>/<kind>/<dst>`, per D-03) does **not** include line/col in the key —
meaning it dedupes at a coarser grain than the TS schema's unique index
does. That is almost certainly the *desired* behavior for most edge kinds
(repeated sync/resolve passes should be idempotent, not append-only), but
it also means two structurally distinct call sites between the same
(source, kind, target) triple will collide and overwrite in the new store,
where TS's `(source, target, kind, line, col)` index would have kept them
distinct. The edge-identity design should explicitly decide (and document)
whether any edge kinds need line/col folded into the key to avoid losing
multi-call-site information that TS's schema preserved.
