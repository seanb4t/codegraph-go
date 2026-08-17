---
phase: 01
plan: 05
title: Measurement pipeline — observation/selection records, -mode measure/select/kinds, prose renderer
agents: gsd-executor (worktree isolation)
status: complete
---

# Plan 01-05 — Measurement Pipeline

## What was built

The measurement pipeline the corpus spike depends on:

| Commit | What |
|---|---|
| `d7d280e` | Two typed documents (`internal/corpora/record.go`): the generated `observations.json` and the curated, hand-authored `selection.json`, with the layered volatile-key strip. `numericSubmap` reads `status[section]` and coerces values to `int64`. |
| `31d5482` | The three tool modes in `tools/corpora/main.go` (`measure`, `select`, `kinds`), the prose renderer (`renderMeasurementProse` → `docs/CORPUS-MEASUREMENT.md`), and the committed multi-language fixture `testdata/corpora/observations.fixture.json`. |
| `e2d9852` | The first committed measurement records: `corpora/observations.json` + `docs/CORPUS-MEASUREMENT.md` (generated from the fixture during smoke verification). |
| `097e673` | **Fix:** `fmt.Sprintf` verbs corrected to `%d` for `int64` values in the prose renderer; `main.go`'s selection read moved off the write-check grep path so a plan-wide grep for `selection.json` in Taskfile bodies stays green. |

## Contracts honored

- `observations.json` is **upsert keyed by `repo@sha`**, never "fully reconstructed"; `-prune` (default off) is the only removal path and `corpora:drift` never passes it.
- `selection.json` is **single-owner** — the tool has no write path to it; curated by hand.
- `-mode select` reads `-in <path>` (default `corpora/observations.json`), never writes, exits non-zero naming the path when input is missing, and emits exactly `{minEdgesPerKind, lockedSet, eligible}` with both arrays sorted.
- `-out` non-default skips the `docs/CORPUS-MEASUREMENT.md` write, so a smoke run with `-out /tmp/obs1.json` cannot overwrite the committed document from a scratch file.
- Never asserts an exact key count on `edgesByKind` — `contains` is a real unranked edge kind not in `RankEdges`; every `RankEdges` member is asserted present, extras permitted only with a positive count.
- `-mode kinds` prints `query.RankEdges` sorted; all Python verify blocks derive the ranked set from it (no hand-restated literals).
- Every `go test` carried `-count=1`. No `[ci skip]` / `[skip ci]` in any commit. Rule `84d1gfpywd` followed: every guard reports what it verified.

## Verification

- `go test -count=1 ./internal/corpora/... ./tools/corpora/...` — green.
- `go build ./...` — clean.
- New `corpora:measure` Taskfile target present.
- Fixture covers all five `PriorityLanguages` and every `RankEdges` kind.

## Deviations

One vet failure in `tools/corpora/prose.go` during execution (`%s` used with `int64`) — fixed in `097e673`. A second API interruption after the fix left SUMMARY.md unwritten; the orchestrator wrote this summary from the commits present. No other deviations.

## Handoff to 01-06

The measurement pipeline is complete and green. 01-06 (the spike: score, freeze N, lock the set) consumes `tools/corpora`'s modes, `corpora/observations.json`, and `internal/corpora.Coverage`. It may write `corpora/selection.json` and the locked-set flags in `corpora/manifest.json`, and must re-run `task corpora:measure` against the locked set to re-derive N.
