# 03-MUTATION-LOG — FIXT-07 five-family mutation rehearsal

**Phase:** 03-non-vacuity-proof-unconditional-ci-execution (Plan 03-02)
**Date:** 2026-08-15
**Rehearsal suite:** the Phase-2 re-frozen golden suite in its FINAL form (Plan 03-01 output: `golden_test.go`, `behavioral_test.go`, per-language `behavioral_*_test.go`, `internal/corpora/coverage_test.go`) — unchanged by this rehearsal; nothing here re-authors the suite.

## Header — the D-01 call, recorded

**D-01 call (RE-MUTATE all five):** The FIXT-07 criterion is "the observed failure … recorded per family". Prior records do not carry the full observed-failure transcript:

- Family (b) has a **count-only** record (`02-04-SUMMARY.md:34`: "a deliberately-deleted golden fails it (25/26)") with no pasted failing output.
- Families (d) and (e) were **specified** to be RED-proven (`01-07-SUMMARY.md:24`: "Both `TestCorpusCoverageClaim` legs are specified to be demonstrated RED against a confirmed mutation … and reverted byte-clean") but no observed-failure transcript appears in the summaries.

Per the D-01 "re-mutate vs. cite" call recorded in 03-02-PLAN.md, **all five families are RE-MUTATED this phase** against the suite in its final form, and the observed failing output is pasted per family below. Prior records are cited as corroboration, not primary evidence:

| Prior record | What it corroborates |
|---|---|
| `01-04-SUMMARY.md:99` | The mutation-revert discipline precedent (append a line to a fetched corpus → `corpora:assert-one` fails part 3/4 → `git checkout --` restores byte-clean) |
| `01-07-SUMMARY.md:24` | Family (e) coverage-guard RED was *specified* (both legs "demonstrated RED against a confirmed mutation … reverted byte-clean"); re-mutated here with pasted output |
| `02-04-SUMMARY.md:34` | Family (b) golden-guard RED precedent (delete a golden → 25/26); re-mutated here with pasted output |

## Header — corpus precondition (review HIGH)

- `task corpora:fetch` ran before any rehearsal: **4/4 locked corpora confirmed present and verified** (`corpora:fetch: fetched or confirmed 4/4 locked corpora`).
- `task corpora:assert` ran before any rehearsal: **`corpora:assert: verified 4/4 locked corpora`** (four-part integrity check).
- **Resolved `CODEGRAPH_CORPUS_DIR`:** `/Users/sean/.cache/codegraph/corpora` — the default `CorpusRoot()` resolution (no `CODEGRAPH_CORPUS_DIR` override, no `XDG_CACHE_HOME` set → `$HOME/.cache/codegraph/corpora`). The locked trees resolve under it as `<root>/<slug>-<sha256[:8]>@<sha>` (e.g. `gohugoio-hugo-a30c7459@0805c734a41b75403e3970e0070227916b6845d2`).
- Rationale: a missing corpus must never be misattributed as the intended RED (T-03-P2-05). All rehearsals below therefore ran against a verified corpus root.

## Pre-mutation cleanliness gate (review finding)

Before every tracked-file mutation AND revert, `git diff --quiet -- <file>` was asserted. Result across all families: **the gate never fired (exit 0 every time)** — no pre-existing tracked edit was overwritten, and no revert was a destructive blind checkout (T-03-P2-06).

---

## Family (a) — CASES.json behavioral property assertions

**Test name:** `TestCorpusBehaviorSynthetic` — case `a-overloaded-same-named-symbols` (the `overloaded-defs-distinct` assertion mode), asserting exactly two distinct definitions for `Validate`.

**Mutation applied:** In `testdata/golden/behavioral_test.go`, inside `TestCorpusBehaviorSynthetic` under the `overloaded-defs-distinct` case, changed the defs-count boundary from `if len(locs) != 2 {` to `if len(locs) != 1 {` — a deliberately wrong expected value for the fixed line that asserts exactly two distinct `Validate` definitions.

**Pre-mutation gate:** `git diff --quiet -- testdata/golden/behavioral_test.go` → exit 0 (clean).

**Observed failure (pasted, `go test -count=1 ./testdata/golden/... -run 'TestCorpusBehaviorSynthetic'`):**

```
--- FAIL: TestCorpusBehaviorSynthetic (0.10s)
    --- FAIL: TestCorpusBehaviorSynthetic/a-overloaded-same-named-symbols (0.00s)
        behavioral_test.go:882: Node("Validate"): got 2 defs, want 2 (locations: map[accounts/validate.go:10:true orders/validate.go:10:true])
FAIL
FAIL	github.com/seanb4t/codegraph-go/testdata/golden	0.442s
```

The Validate output reports 2 defs; the mutated boundary expected 1, so the family went RED on the defs-count assertion naming case (a).

**Revert:** `git checkout -- testdata/golden/behavioral_test.go`.

**Byte-clean proof:** `git diff --stat testdata/golden/behavioral_test.go` empty after revert; `go test -count=1 ./testdata/golden/... -run 'TestCorpusBehaviorSynthetic'` → `ok` (green).

---

## Family (c) — CLI==MCP byte-identity trio

**Test name:** `TestExploreCLIMatchesMCP` (the shared comparison-loop test of the CLI==MCP trio; `TestNodeCLIMatchesMCP` and `TestNodeLineHintCLIMatchesMCP` share the same comparison shape — this is a **shared-comparison-loop demonstration**, not a per-sibling claim).

**Mutation applied:** In `testdata/golden/behavioral_test.go`, inside `TestExploreCLIMatchesMCP`, the MCP-side call (`mcpOut := callExploreViaMCP(t, dir, tc.query)`) was made to feed a one-word-suffixed query (`tc.query + " zzz"`) **when and only when the row's corpus is `behavioral`**; the CLI query and every other row were left untouched.

**Pre-mutation gate:** `git diff --quiet -- testdata/golden/behavioral_test.go` → exit 0 (clean).

**Observed failure (pasted, `go test -count=1 ./testdata/golden/... -run '^TestExploreCLIMatchesMCP$'`):**

```
--- FAIL: TestExploreCLIMatchesMCP (5.42s)
    --- FAIL: TestExploreCLIMatchesMCP/behavioral (0.16s)
        behavioral_test.go:1296: explore("user account") on behavioral: CLI and MCP output diverge (EXPL-05):
    --- PASS: TestExploreCLIMatchesMCP/hugo (1.28s)
    --- PASS: TestExploreCLIMatchesMCP/guava (3.46s)
    --- PASS: TestExploreCLIMatchesMCP/serilog (0.29s)
    --- PASS: TestExploreCLIMatchesMCP/requests (0.23s)
FAIL
```

With corpora fetched, ONLY the behavioral row diverged ("CLI and MCP output diverge (EXPL-05)"); the four locked rows (hugo/guava/serilog/requests) stayed green. This demonstrates the family's shared `cliOut != mcpOut` comparison loop going red on a behavioral-surface divergence.

**Revert:** `git checkout -- testdata/golden/behavioral_test.go`.

**Byte-clean proof:** `git diff --stat testdata/golden/behavioral_test.go` empty after revert; `go test -count=1 ./testdata/golden/... -run '^TestExploreCLIMatchesMCP$'` → `ok` (green).
