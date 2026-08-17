# Golden fixtures and the behavioral corpus

This directory holds the golden test suite that guards the committed
tool-output fixtures. The legacy capture path and the external,
network-fetched corpora it once captured from are retired as of Phase 2
(FIXT-04). The purpose-built
behavioral corpus now lives at the repo root as `corpus/behavioral/`
(D-03/D-04).

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

## Edge-identity design note — dedup grain

The Pebble edge key design (`e/<src>/<kind>/<dst>`, per D-03) does **not**
include line/col in the key, so two structurally distinct call sites
between the same (source, kind, target) triple collide and overwrite
rather than accumulate. This is deliberate: repeated sync/resolve passes
over the same source are idempotent rather than append-only, which avoids
a class of historical double-insert bugs where re-running indexing over
unchanged source silently inflated edge counts and polluted
callers/impact results. Any edge kind that needs multi-call-site
information preserved (rather than deduped) should fold line/col into the
key explicitly, as a documented design decision.
