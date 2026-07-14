# synthetic-parity corpus

A small, purpose-built Go source tree (per D-03/D-06) that deliberately
exercises the exact v0.1 blind spots `explore`/`node` behavioral parity
(Phase 1) is validated against, distinct from the pinned real-world corpora
(`corpus/weft-go/`, `corpus/colbymchenry-codegraph/`). This directory
contains only the source tree; captured TS 1.3.1 golden outputs (D-01) live
alongside it as `explore*.json`/`node*.json` — see `testdata/golden/README.md`
for the frozen-ground-truth policy. **No expected output is hand-authored
here** — outputs are captured from the live TS 1.3.1 oracle only.

Indexable at `src/` (contains its own `go.mod`, module `synthetic-parity`):

```sh
codegraph index --force -q testdata/golden/corpus/synthetic-parity/src
```

## D-03 case map

| Case | What it tests | Symbol(s) | File(s) |
|------|----------------|-----------|---------|
| (a) **overloaded name** | `node <name>` multi-definition enumeration (NODE-01) — two distinct top-level definitions sharing an identical symbol name across different files/packages | `Validate` (defined twice) | `src/accounts/validate.go`, `src/orders/validate.go` |
| (b) **multi-word query** | `explore`'s multi-word `<query...>` tokenization (EXPL-01) — a CamelCase type name a query like `explore user account` must tokenize (User/Account/Manager) and match | `UserAccountManager` | `src/accounts/manager.go` |
| (c) **Test\*-heavy, weakly-connected cluster** | The file-relevance gate (EXPL-03) — a `Test*`-named function that lexically matches a query (e.g. "account recovery") but has ZERO inbound graph edges, alongside a structurally-connected non-test symbol in the same cluster | `TestAccountRecovery` (zero inbound edges) vs. `recoverAccount`/`validateRecovery` (structurally connected) | `src/recovery/recovery_test.go`, `src/recovery/recovery.go` |
| (d) **structural-beats-lexical** | RWR ranking (EXPL-02) — a symbol whose NAME loosely matches the query but is graph-isolated, vs. a symbol that does NOT name-match but is heavily called-by/calls the query's true target | `AccountBalanceHelper` (isolated, matches "account balance") vs. `ReconcileLedger` (no name match, strictly more graph edges — called by `GetBalance`+`PostTransaction`, calls `applyAdjustment`+`AuditEntry`) | `src/ledger/ledger.go` |

Every behavioral fixture captured against this corpus runs on **both** the
TS CLI and the `codegraph_explore`/`codegraph_node` MCP tools (EXPL-05/
NODE-04), per D-03.

## Verifying the corpus mechanically (no algorithm required)

Case (a) — overloaded name, 2+ nodes:

```sh
codegraph query Validate -p testdata/golden/corpus/synthetic-parity/src --json
# -> 2 "function" nodes named Validate: accounts/validate.go, orders/validate.go
```

Case (d) — structural edge-count asymmetry (symbol B > symbol A):

```sh
codegraph callers ReconcileLedger -p testdata/golden/corpus/synthetic-parity/src   # 2 callers
codegraph callees ReconcileLedger -p testdata/golden/corpus/synthetic-parity/src   # 2 callees
codegraph callers AccountBalanceHelper -p testdata/golden/corpus/synthetic-parity/src  # 0 callers
codegraph callees AccountBalanceHelper -p testdata/golden/corpus/synthetic-parity/src  # 0 callees
```
