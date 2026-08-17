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

---

## Family (b) — golden byte-identity guard

**Test name:** `TestReFrozenGoldensValid` (26-golden exact enumeration guard, `expectedGoCaptures`-derived).

**Mutation applied:** Temporarily deleted the tracked golden `testdata/golden/corpus/hugo/go-explore.json` (moved aside to `/tmp/go-explore.json.muttmp`), so the expected-set enumeration found 25 of 26.

**Pre-mutation gate:** `git diff --quiet -- testdata/golden/corpus/hugo/go-explore.json` → exit 0 (clean).

**Observed failure (pasted, `go test -count=1 ./testdata/golden/... -run 'TestReFrozenGoldensValid'`):**

```
--- FAIL: TestReFrozenGoldensValid (0.01s)
    --- FAIL: TestReFrozenGoldensValid/hugo/go-explore.json (0.00s)
        golden_test.go:287: expected golden corpus/hugo/go-explore.json not found (run `task golden:regen` to regenerate): open corpus/hugo/go-explore.json: no such file or directory
    golden_test.go:312: TestReFrozenGoldensValid: 25/26 goldens verified
    golden_test.go:317: verified 25 of 26 expected goldens — the executed-and-verified count must EXACTLY equal the enumerated total (a count short by even one means a golden was dropped or a subtest loop never ran; missing entries must be regenerated via `task golden:regen`)
FAIL
```

**Revert:** `git checkout -- testdata/golden/corpus/hugo/go-explore.json` (restored, 5002 bytes).

**Byte-clean proof:** file exists again at its tracked path; `go test -count=1 ./testdata/golden/... -run 'TestReFrozenGoldensValid'` → `ok` (26/26 green).

---

## Family (d) — hermetic fail-loud resolution

**Test name:** `TestPriorityLanguagesResolveToLockedCorpus` (5 priority languages; the `lockedCorpusDir` helper's `os.Stat` + `t.Fatalf` path).

**Mutation applied:** Renamed the fetched hugo tree `<root>/gohugoio-hugo-a30c7459@0805c734a41b75403e3970e0070227916b6845d2` (the tree `go` and `tsjs` resolve to) to `<same>.muttmp`. The rename → run → restore ran in **ONE shell invocation guarded by an EXIT trap** (`trap 'mv "$orig.muttmp" "$orig"' EXIT`) restoring the exact original path on any exit — a pure `mv` rename, never a copy/delete (T-03-P2-02, T-03-P2-07).

**Observed failure (pasted, `go test -count=1 ./testdata/golden/... -run 'TestPriorityLanguagesResolveToLockedCorpus'`):**

```
--- FAIL: TestPriorityLanguagesResolveToLockedCorpus (0.00s)
    --- FAIL: TestPriorityLanguagesResolveToLockedCorpus/go (0.00s)
        behavioral_test.go:1007: lockedCorpusDir("go"): locked tree directory /Users/sean/.cache/codegraph/corpora/gohugoio-hugo-a30c7459@0805c734a41b75403e3970e0070227916b6845d2 not found or not a directory: stat ...: no such file or directory; run 'task corpora:fetch'
    --- FAIL: TestPriorityLanguagesResolveToLockedCorpus/tsjs (0.00s)
        behavioral_test.go:1007: lockedCorpusDir("tsjs"): locked tree directory ... not found or not a directory: stat ...: no such file or directory; run 'task corpora:fetch'
FAIL
```

`lockedCorpusDir` **t.Fatalf'd** naming the missing tree for both surfaces that resolve to hugo (`go`, `tsjs`) — **fail-NOT-skip**, the suite never falls back to `t.Skip` (D-10).

**Revert:** the EXIT trap restored the exact original name (`mv "$orig.muttmp" "$orig"`) on shell exit; verified the original path exists again and the `.muttmp` path is gone.

**Byte-clean proof:** `ls -d <root>/gohugoio-hugo-a30c7459@0805c734a41b75403e3970e0070227916b6845d2` resolves; `go test -count=1 ./testdata/golden/... -run 'TestPriorityLanguagesResolveToLockedCorpus'` → `ok`, all 5 subtests (`go/java/csharp/python/tsjs`) PASS.

**D-01 call result (re-mutated):** family (d) was previously specified-but-without-transcript (`01-07-SUMMARY.md:24` names it among the legs "specified to be demonstrated RED"); re-mutated this phase with full pasted output above.

---

## Family (e) — coverage guard

**Test name:** `TestCorpusCoverageClaim` (reads the three real committed documents — `manifest.json`, `observations.json`, `selection.json` — and requires `CheckCoverage` to report zero failures).

**Mutation applied:** In `corpora/selection.json`'s `minEdgesPerKind`, temporarily raised the `calls` threshold from `29406` to `999999` — one scalar in the committed document, set above the observed best for that kind (measured calls best = 58812, guava) so it is unsatisfiable by construction.

**Pre-mutation gate:** `git diff --quiet -- corpora/selection.json` → exit 0 (clean).

**Observed failure (pasted, `go test -count=1 ./internal/corpora/... -run 'TestCorpusCoverageClaim'`):**

```
--- FAIL: TestCorpusCoverageClaim (0.00s)
    coverage_test.go:138: CheckCoverage(committed docs) reported failures:
        kind calls derived count 58812 below threshold 999999
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/corpora	0.176s
```

The guard reports the `calls` kind "below threshold", naming the derived count (58812) and the mutated threshold (999999).

**Revert:** `git checkout -- corpora/selection.json`.

**Byte-clean proof:** `git diff --stat corpora/selection.json` empty after revert; `go test -count=1 ./internal/corpora/... -run 'TestCorpusCoverageClaim'` → `ok` (green).

**D-01 call result (re-mutated):** family (e) was previously specified-but-without-transcript (`01-07-SUMMARY.md:24`: both `TestCorpusCoverageClaim` legs "specified to be demonstrated RED against a confirmed mutation … and reverted byte-clean"); re-mutated this phase with full pasted output above.

---

## Conclusion — FIXT-07 positive reading

All **five** assertion families were red-demonstrated against the suite in its FINAL form, with one targeted mutation per family, the observed failure pasted, and a byte-clean revert recorded per family:

| Family | Test red-demonstrated | Mutation | Observed RED (pasted above) | Revert | Green re-run |
|---|---|---|---|---|---|
| (a) behavioral properties | `TestCorpusBehaviorSynthetic/a-overloaded-same-named-symbols` | defs boundary `!= 2` → `!= 1` | `Node("Validate"): got 2 defs ...` | `git checkout -- testdata/golden/behavioral_test.go` | `-run TestCorpusBehaviorSynthetic` ok |
| (b) golden byte-identity | `TestReFrozenGoldensValid` | delete `corpus/hugo/go-explore.json` | `25/26 goldens verified` naming the missing golden | `git checkout -- testdata/golden/corpus/hugo/go-explore.json` | `-run TestReFrozenGoldensValid` ok (26/26) |
| (c) CLI==MCP trio | `TestExploreCLIMatchesMCP/behavioral` | MCP-side one-word query suffix, behavioral row only | `CLI and MCP output diverge (EXPL-05)` (locked rows green) | `git checkout -- testdata/golden/behavioral_test.go` | `-run '^TestExploreCLIMatchesMCP$'` ok |
| (d) hermetic resolution | `TestPriorityLanguagesResolveToLockedCorpus/go,+tsjs` | rename hugo tree (EXIT-trap restore) | `lockedCorpusDir("go"): ... not found ... run 'task corpora:fetch'` (fail-NOT-skip) | EXIT trap `mv` restore | `-run TestPriorityLanguagesResolveToLockedCorpus` ok (5/5) |
| (e) coverage guard | `TestCorpusCoverageClaim` | `calls` threshold 29406 → 999999 | `kind calls derived count 58812 below threshold 999999` | `git checkout -- corpora/selection.json` | `-run TestCorpusCoverageClaim` ok |

**Suite ran 26/26 goldens, red in all five families observed** — the re-baselined golden suite is non-vacuous. Post-rehearsal working tree: `git status --porcelain` clean except this log artifact; the final commit carries no mutation byte in `testdata/golden/`, `corpus/`, `corpora/`, or `internal/`.
