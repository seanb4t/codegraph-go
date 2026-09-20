---
phase: 03-verb-fold
plan: 03
subsystem: cli
tags: [cobra, verb-fold, mutation-testing, coverage, mcp]

# Dependency graph
requires:
  - phase: 03-verb-fold/03-02
    provides: "the folded search --full / daemon unlock surface and the two hidden rename stubs (renamed.go) this plan's RED mutation targets"
provides:
  - "Family (b): a proven, non-vacuous RED demonstration that task docs:cli:drift and TestEveryRegisteredFlagIsAccountedFor both catch a re-visible query stub, with a byte-clean revert and a green re-run"
  - "Adjacency/empty/ordering proofs that the 03-02 re-freeze was a real, non-empty, deterministic diff"
  - "Completions/man proofs from the live HEAD binary (24 top-level, 3 under daemon, 29 man pages) with man's hidden precedent as the positive control, and confirmation that no completion/man artefact is committed"
  - "A zero-diff git fact across the whole phase for internal/mcp, testdata/wireoracle, test/wireoracle, plus the frozen 8-tool MCP name set and an uncached green wire-oracle run"
  - "COVERAGE.md's no-external-API declaration"
affects: [03-verb-fold/03-04-verb-fold-plan]

# Actuals (#2632)
actuals:
  tokens: 4371
  tasks: 2
  commits: 2
  plan_head_before: 32b75558772a78e3f5376e3f7d7b55b5ad574fad

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Family (b) mutation-log shape reproduced from 02-MUTATION-LOG.md: Test/guard, What are we testing and why, Pre-mutation gate, Mutation applied, Observed failure verbatim + exit code, Revert, Byte-clean proof, Green re-run, Verdict."
    - "Generated-surface proofs (completions/man/MCP/wire-oracle) recorded as an 'after' companion to 03-01's Baseline, never as a mutation-log family — no revert step because nothing was mutated for Task 2."

key-files:
  created:
    - .planning/phases/03-verb-fold/COVERAGE.md
  modified:
    - .planning/phases/03-verb-fold/03-MUTATION-LOG.md

key-decisions:
  - "No source, doc, or test file changed by this plan — internal/cli/renamed.go's Hidden flip was applied and reverted entirely in the working tree per task, never staged; both commits touch only .planning/ artifacts."

requirements-completed: [VERB-06, VERB-07]

coverage:
  - id: D1
    description: "Family (b): a re-visible query stub turns task docs:cli:drift and TestEveryRegisteredFlagIsAccountedFor RED with named output, and both return GREEN after a byte-clean revert"
    requirement: "VERB-07"
    verification:
      - kind: other
        ref: "task docs:cli:drift (RED: exit 201, '+## codegraph query'; GREEN: exit 0, byte-identical)"
        status: pass
      - kind: unit
        ref: "internal/cli/cli_reference_test.go#TestEveryRegisteredFlagIsAccountedFor (RED: --- FAIL, 'stale allowlist entry: codegraph query'; GREEN: --- PASS)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Adjacency/empty/ordering proofs: the feat commit's docs/CLI-REFERENCE.md numstat is non-empty (28+/49-); compared 1 generated file / walked 37 commands never read zero; two tools/clidoc regenerations are cmp-identical to each other and to the committed reference"
    requirement: "VERB-07"
    verification:
      - kind: other
        ref: "git show --numstat on the feat commit; two go run ./tools/clidoc runs compared via cmp"
        status: pass
    human_judgment: false
  - id: D3
    description: "Completions and man pages reflect the new surface from the live HEAD binary with no committed artefact: __complete lists 24 top-level / 3 under daemon (query/unlock/man absent, daemon has unlock); codegraph man writes 29 pages including daemon-unlock and excluding query/unlock/man"
    requirement: "VERB-06"
    verification:
      - kind: other
        ref: "codegraph __complete '' / __complete daemon '' / man <dir> against a HEAD build; git ls-files negative + docs/CLI-REFERENCE.md positive control"
        status: pass
    human_judgment: false
  - id: D4
    description: "The 8-tool MCP set and wire-oracle transcripts are unchanged across the whole phase; codegraph_search still calls eng.Search, never eng.Query; COVERAGE.md carries the no-external-API declaration"
    requirement: "VERB-07"
    verification:
      - kind: other
        ref: "git diff --quiet F^ HEAD -- internal/mcp testdata/wireoracle test/wireoracle; go test ./test/wireoracle/... -count=1 (ok, 51.787s uncached); task test:wireoracle (exit 0)"
        status: pass
    human_judgment: false

duration: ~30min active work
completed: 2026-09-16
status: complete
---

# Phase 3 Plan 03: Verb Fold — VERB-07 RED demonstration, and completions/man/MCP/wire-oracle proofs Summary

**Watched `task docs:cli:drift` and `TestEveryRegisteredFlagIsAccountedFor` both go RED against a one-token re-visible `query` stub and GREEN again after a byte-clean revert, then proved completions/man already reflect the folded surface from the live binary (with `man`'s own hidden precedent as the positive control) and that the 8-tool MCP set and wire-oracle transcripts carry zero diff across the whole phase — no source change, two `docs(03-03):` commits.**

## Performance

- **Duration:** ~30 min active work
- **Tasks:** 2
- **Commits:** 2 — measured via `git rev-list --count 32b75558772a78e3f5376e3f7d7b55b5ad574fad..HEAD`
- **Files modified:** 2 (1 created — `COVERAGE.md`; 1 modified — `03-MUTATION-LOG.md`)

## Accomplishments

- **Family (b) RED demonstration.** Flipped `internal/cli/renamed.go`'s `query` stub `Hidden: true` → `false` (one token; `unlock` stub untouched), and watched both committed gates fail on exactly this regression:
  - `task docs:cli:drift` → exit 201, `compared 1 generated file`, `differs from a fresh regeneration`, diff shows `+## codegraph query`.
  - `TestEveryRegisteredFlagIsAccountedFor` → `--- FAIL`, `stale allowlist entry: codegraph query` (the command-level allowlist entry stops being honored once the stub becomes `documentedByReference`).
  Reverted via `git checkout -- internal/cli/renamed.go`; `git diff --quiet` held; both gates returned GREEN (`byte-identical to a fresh regeneration`; `walked 37 commands … 3 accepted via testdata/cli-reference-allowlist.txt`).
- **Adjacency/empty/ordering proofs.** The feat commit `5d69ee2e`'s `docs/CLI-REFERENCE.md` numstat is `28  49` (real, non-empty re-freeze, not a no-op regeneration); `compared 1 generated file` / `walked 37 commands` never printed zero in any run; two consecutive `go run ./tools/clidoc` regenerations were `cmp`-identical to each other and to the committed reference.
- **Completions and man pages from the live binary (VERB-06).** Built the HEAD binary and probed it directly (RESEARCH Pattern 4 — nothing is committed to regenerate): `__complete ""` lists 24 entries (`search`/`daemon` present, `query`/`unlock`/`man` absent — `man`'s pre-existing hidden exclusion as the positive control); `__complete daemon ""` lists `start`, `stop`, `unlock` (3); `codegraph man <dir>` writes 29 pages including `codegraph-daemon-unlock.1` and excluding `codegraph-query.1`/`codegraph-unlock.1`/`codegraph-man.1`. Confirmed the generated completion *script* itself names no commands at all (`completion bash | rg -c -w 'query|unlock'` = 0, and the same for `'search|daemon'` = 0), which is why `__complete` — not a script grep — is the honest probe. `git ls-files` carries no `*.1`/`*.bash`/`*.zsh`/`*.fish`/`completions/` entry (paired against the positive control: `docs/CLI-REFERENCE.md` count 1).
- **MCP set and wire oracle zero-diff (VERB-07, D-14).** `git diff --quiet F^ HEAD -- internal/mcp testdata/wireoracle test/wireoracle` holds across the whole phase (`F` = the feat commit `5d69ee2e`); the eight tool names extracted from `internal/mcp/tools.go` are unchanged from 03-01's Baseline; `codegraph_search`'s handler calls `eng.Search` exactly once and `eng.Query` zero times; `go test ./test/wireoracle/... -count=1` → `ok` in 51.787s (uncached); `task test:wireoracle` → exit 0; `git status --porcelain testdata/wireoracle` empty.
- **`COVERAGE.md`** carries the exact declaration line the seal-time gate expects: `No external API integration: the phase folds two CLI verbs and freezes the existing 8-tool MCP set; no API/SDK/service is integrated`.
- **Cheap cross-check:** `git diff --quiet F^ HEAD -- .goreleaser.yaml` holds — the cask stanza that invokes `completion`/`man` at install time is unchanged by this phase.

## Task Commits

Each task was committed atomically, `docs(03-03):` throughout, no source change:

1. **Task 1** — `ef3e4c6b` `docs(03-03): record the VERB-07 RED demonstration against a re-visible query stub (Family b)` — `.planning/phases/03-verb-fold/03-MUTATION-LOG.md` only.
2. **Task 2** — `1fac6b74` `docs(03-03): declare no external API integration; record completions/man/MCP/wire-oracle proofs (VERB-06, VERB-07)` — `.planning/phases/03-verb-fold/COVERAGE.md` (new) and `.planning/phases/03-verb-fold/03-MUTATION-LOG.md`.

## Files Created/Modified

- `.planning/phases/03-verb-fold/COVERAGE.md` (new) — the one-line no-external-API declaration plus a pointer sentence.
- `.planning/phases/03-verb-fold/03-MUTATION-LOG.md` — gained `## Family (b) — VERB-07: a re-visible query command turns docs:cli:drift and TestEveryRegisteredFlagIsAccountedFor RED` (Task 1) and `## Generated-surface and MCP proofs (03-03 Task 2)` (Task 2); Family (a) and Baseline untouched.

## Verbatim Evidence

### Family (b) RED — `task docs:cli:drift` (mutation applied)

```
docs:cli:drift: compared 1 generated file
::error::docs:cli:drift: docs/CLI-REFERENCE.md differs from a fresh regeneration by the pinned toolchain — run `task docs:cli` and commit the result
--- docs/CLI-REFERENCE.md	2026-09-16 15:10:06
+++ /var/folders/.../CLI-REFERENCE.md	2026-09-16 17:51:39
@@ -34,6 +34,7 @@
+* [codegraph query](#codegraph-query)	 - Renamed to "search --full" — stub removed in v0.15.0
@@ -618,7 +619,25 @@
+## codegraph query
+
+Renamed to "search --full" — stub removed in v0.15.0
...
task: Failed to run task "docs:cli:drift": exit status 1
exit=201
```

### Family (b) RED — `TestEveryRegisteredFlagIsAccountedFor` (mutation applied)

```
cli_reference_test.go:254: walked 37 commands (hidden included), inspected 113 flags: 111 accepted via ../../docs/CLI-REFERENCE.md, 2 accepted via testdata/cli-reference-allowlist.txt
    cli_reference_test.go:274: 1 problem(s):
        stale allowlist entry: codegraph query (matches no registered command or flag)
--- FAIL: TestEveryRegisteredFlagIsAccountedFor (0.08s)
FAIL
exit=1
```

### Green control (after `git checkout -- internal/cli/renamed.go`)

```
docs:cli:drift: compared 1 generated file
docs:cli:drift: docs/CLI-REFERENCE.md byte-identical to a fresh regeneration (temporary file only — source tree untouched)
exit=0
```

```
cli_reference_test.go:254: walked 37 commands (hidden included), inspected 113 flags: 110 accepted via ../../docs/CLI-REFERENCE.md, 3 accepted via testdata/cli-reference-allowlist.txt
--- PASS: TestEveryRegisteredFlagIsAccountedFor (0.02s)
ok  	github.com/seanb4t/codegraph-go/internal/cli	0.451s
```

### Adjacency proof

```
$ F=$(git log --format=%H --grep='^feat(cli)!: fold query into search --full and unlock into daemon unlock$' -1)
$ echo "$F"
5d69ee2ea3c6276ed73c946b24be93612fae1698
$ git show --format= --numstat "$F" -- docs/CLI-REFERENCE.md
28	49	docs/CLI-REFERENCE.md
```

### `__complete` / man listings (Task 2)

```
$ /tmp/03-03-bin __complete ""
... 24 entries (search/daemon present; query/unlock/man absent) ...
$ /tmp/03-03-bin __complete daemon ""
start / stop / unlock  (3 entries)
$ /tmp/03-03-bin man <tmpdir> && ls <tmpdir> | wc -l
29
```

### MCP/wire-oracle zero-diff range

```
$ F=5d69ee2ea3c6276ed73c946b24be93612fae1698
$ git diff --quiet "$F^" HEAD -- internal/mcp testdata/wireoracle test/wireoracle
(exit 0 — holds)
$ rg -o 'Name:\s+"codegraph_[a-z]+"' internal/mcp/tools.go | rg -o 'codegraph_[a-z]+' | sort -u
codegraph_callees codegraph_callers codegraph_explore codegraph_files codegraph_impact codegraph_node codegraph_search codegraph_status
$ GOTOOLCHAIN=go1.26.6 go test ./test/wireoracle/... -count=1
ok  	github.com/seanb4t/codegraph-go/test/wireoracle	51.787s
```

## Decisions Made

None beyond the plan's own — executed exactly as written. No new architectural or scope decisions required.

## Deviations from Plan

None - plan executed exactly as written. Every task's `<verify>` blocks passed on the first run; no auto-fixes, no blocking issues, no Rule 4 escalations.

## Known Stubs

None new. The `query`/`unlock` rename stubs (`internal/cli/renamed.go`) are 03-02's intentional, documented deliverable — this plan only exercised them via a temporary, reverted working-tree mutation and never modified them.

## Threat Flags

None — every threat this plan's `<threat_model>` registered (T-03-03-01 … T-03-03-SC) was mitigated within scope: the RED mutation never reached a commit (T-03-03-02), no blanket regenerate occurred (T-03-03-03), the MCP/wire surface stayed zero-diff (T-03-03-04), the completion-script-vs-`__complete` distinction was recorded explicitly (T-03-03-05), and no CI-skip marker appears in either commit (T-03-03-06).

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- VERB-06 and VERB-07 are both fully evidenced: the generated-reference gates are proven non-vacuous, completions/man are proven from the live binary, and the MCP/wire-oracle surface is a zero-diff git fact across the phase.
- `COVERAGE.md` is in place for phase seal-time.
- 03-04 (the plan this SUMMARY's frontmatter lists as `affects`) can proceed — no blockers, no source change to reconcile.

---
*Phase: 03-verb-fold*
*Completed: 2026-09-16*

## Self-Check: PASSED
