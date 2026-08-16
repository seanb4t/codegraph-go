---
status: complete
phase: 06-benchmark-de-coupling-memory-sweep
source: [06-01-SUMMARY.md, 06-02-SUMMARY.md, 06-03-SUMMARY.md, 06-04-SUMMARY.md, 06-05-SUMMARY.md, 06-06-SUMMARY.md]
started: 2026-08-16T21:46:20Z
updated: 2026-08-16T22:02:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Fresh-session recall — file half (startup context)
expected: A genuinely fresh session's startup context (.claude/CLAUDE.md + harness-injected .planning/ content) describes codegraph-go on its own terms — no present-tense port/parity/drop-in framing, no stated obligation to another implementation's behaviour.
result: pass
observed: "Session began after /clear and assembled startup context live. .claude/CLAUDE.md states the Compatibility constraint is retired (v0.11.0, 2026-08-13) and that behavior is defined by codegraph-go's own requirements and frozen goldens; Core Value is a standalone claim. No present-tense port/parity/drop-in framing; other-implementation references are past-tense historical or explicitly retired."

### 2. Fresh-session recall — store half (engram spine)
expected: A genuinely fresh session's engram spine recall for `repo:github.com/seanb4t/codegraph-go` surfaces the four correcting records (xj1stbrsw6, gxwkk3necn, b9wjge7375, mw5z9s9bft) and none of the four superseded originals (3ekc84hbqt, gw79qy2a9z, agggksad53, 7f0pq2wepv). No recalled memory asserts a binding parity floor, functionality floor, or migration read contract.
result: skipped
reason: "User reported: I can't directly test this. The engram store is not queryable by the maintainer directly; the check requires MCP tool access. NOT converted to a pass — the human judgment this checkpoint asks for was not rendered."
agent_probe: |
  Performed in this session (which is itself a fresh post-/clear session), recorded as evidence
  rather than as a verdict:
  - search_memory phrased adversarially in the SUPERSEDED records' own vocabulary ("parity is a
    functionality baseline", "migration read contract still binds", "FilesByLanguage must be
    json:\"-\""), k=12 over repo:github.com/seanb4t/codegraph-go.
  - All four correcting records returned: gxwkk3necn (0.703), xj1stbrsw6 (0.687),
    b9wjge7375 (0.738), mw5z9s9bft (0.710).
  - None of the four superseded originals returned, despite the query being written in their own
    terms — the query most likely to resurrect them if suppression were broken.
  - get_memory("3ekc84hbqt") returns original content intact with
    superseded_by=fc4e8512-9fbc... (xj1stbrsw6) stamped: history preserved, nothing deleted.
  - Three non-superseded records still MENTION migrate/parity, all past-tense historical and
    consistent with the sweep's keep-historical verdict: swzxf5tr2x (Phase 7 migration tool
    SHIPPED), q339zmp200 (historical branch-name tag), gwzjpq9xvq (tagged
    NOT-superseded-here-on-purpose). None describes codegraph-go AS a port in the present tense.
  Caveat on an earlier weaker observation: absence from a limit=10 recency list is NOT evidence of
  suppression. Only the adversarial search above is.

### 3. BENCH-03 — bench.yml publish run inspection
expected: The dispatched bench.yml `publish` run on this branch shows an absolute per-corpus table (files/s, bytes/s, query latency, peak RSS, cold start) for weft-go and cockroachdb-pebble in its job summary; the publish-results artifact uploads without `if-no-files-found: error` tripping; no step installs or invokes any implementation other than the freshly-built Go binary.
result: pass
observed: |
  Re-derived from the live run rather than read from the ledger's transcription.
  - `gh run view 31973967130` -> conclusion=success, event=workflow_dispatch,
    headBranch=gsd/v0.11.0-standalone-project-identity. Job
    "benchmark publish (BENCH-01/BENCH-03, non-blocking)" success; all 6 other jobs skipped.
  - `gh run download 31973967130 -n publish-results` -> publish-results.json read directly:
    exactly 2 records, repos {weft-go, cockroachdb-pebble}, both "subject": "go", runner
    namespace-profile-linux-amd64-4x8. weft-go 3357.77 files/s, 29802491 bytes/s, 10.596 ms p50,
    57020416 peak RSS, 6.498 ms cold start. cockroachdb-pebble 1244.30 files/s, 19607524 bytes/s,
    25.467 ms p50, 186265600 peak RSS, 6.746 ms cold start. All positive. No comparison column,
    no second subject, no ratio.
  - Artifact downloadable => if-no-files-found: error did not trip.
  - Complete publish-job step list read in full: Set up job, Set up runner, Checkout, Set up Go,
    Cache Go modules and build, Build Go codegraph binary, Run benchmark, Publish results to job
    summary, Upload raw results artifact, + post-cleanup. No Node/npm/npx step exists.
  Cosmetic, not a defect: 06-LIVE-VERIFICATIONS.md records the artifact at 454 bytes; the
  downloaded JSON is 769 bytes — compressed zip size vs uncompressed content.

### 4. Bench runner has exactly two modes; comparison architecture gone
expected: tools/bench/runner has exactly two modes (publish, regression); the two-subject comparison architecture, its -ts-binary flag, resolveTSBinary/macOSHomebrewTSBinary, and runHeadToHead are gone
result: pass
source: automated
coverage_id: 06-01/D1

### 5. realcorpus.Corpora() carries exactly two entries
expected: realcorpus.Corpora() carries exactly two entries (weft-go, cockroachdb-pebble); both CommitSHA pins byte-unchanged
result: pass
source: automated
coverage_id: 06-01/D2

### 6. Real publish-mode run emits two go-subject records, verified pure-Go
expected: A real -mode publish run over both corpus entries emits two subject:"go" records with positive metrics, verified end-to-end by publishcheck (no Node dependency in the verification path)
result: pass
source: automated
coverage_id: 06-01/D3

### 7. Six comparison-era capture files removed from tracking
expected: Six committed comparison-era headtohead-*.json capture files removed from tracking; git history preserves them
result: pass
source: automated
coverage_id: 06-01/D4

### 8. Surviving bench prose describes absolute single-subject measurement
expected: Surviving prose in tools/bench/realcorpus and tools/bench/runner describes absolute single-subject measurement, citing no second implementation
result: pass
source: automated
coverage_id: 06-01/D5

### 9. Publish job runs-on and runner env pinned
expected: TestBenchPublishJobShape asserts the publish job's runs-on and CODEGRAPH_BENCH_RUNNER env value are both exactly namespace-profile-linux-amd64-4x8
result: pass
source: automated
coverage_id: 06-02/D06.1

### 10. Publish job upload-artifact step and step-summary shape
expected: Publish job's upload-artifact step (its OWN with: map, not rebless's) carries name/path/if-no-files-found, and exactly one run body references GITHUB_STEP_SUMMARY
result: pass
source: automated
coverage_id: 06-02/D06.4

### 11. Publish job run bodies free of gate/regression/rebless/node invocations
expected: Concatenation of all publish-job run bodies is non-empty and contains no gate invocation, no regression-mode flag, no -rebless flag, no task bench call, no npm/npx/node
result: pass
source: automated
coverage_id: 06-02/D06.3

### 12. Taskfile bench-target set is exactly {bench:regression}
expected: Taskfile bench-target set equals exactly {bench:regression}; no task bench* call inside the publish job's run bodies
result: pass
source: automated
coverage_id: 06-02/D06.2

### 13. rebless job proven untouched
expected: rebless job proven untouched by a parsed-subtree fixture (runs-on, env, if, ordered 6-step names, upload with: map, no permissions:) plus a SHA-256 digest of its three run bodies
result: pass
source: automated
coverage_id: 06-02/RebRebless

### 14. Publish-job scoping demonstrated by discriminating perturbation
expected: Deleting only the publish job's if-no-files-found: error line reddens TestBenchPublishJobShape by name while rebless's identical line survives untouched
result: pass
source: automated
coverage_id: 06-02/PublishScoping

### 15. internal/bench prose describes absolute externally-observed measurement
expected: internal/bench package doc/comment prose describes absolute, externally-observed measurement on its own terms, citing no deleted capture file and no second runtime
result: pass
source: automated
coverage_id: 06-03/D1

### 16. Regression gate drives the committed baseline.json
expected: internal/bench/baseline_gate_test.go's TestCheckRegressionAgainstCommittedBaseline loads the COMMITTED tools/bench/baseline.json and drives CheckRegression with it
result: pass
source: automated
coverage_id: 06-03/D2

### 17. CheckRegression still fires — proven by two reverted mutations
expected: CheckRegression still fires against the committed baseline.json — demonstrated RED against two confirmed-applied, byte-cleanly-reverted mutations, each reddening both the pre-existing table AND the committed-baseline test
result: pass
source: automated
coverage_id: 06-03/D3

### 18. docs/BENCHMARKS.md publishes absolute provenanced figures
expected: docs/BENCHMARKS.md publishes measured, provenanced absolute figures (files/s, bytes/s, query latency, peak RSS, cold start) for both surviving corpora with methodology and the regression-gate description, and carries no comparison table, ratio, or second implementation
result: pass
source: automated
coverage_id: 06-04/D1

### 19. Phase-wide census instrument positive-controlled before its zero was trusted
expected: BENCH-02's in-tree half closed by a phase-wide census whose instrument was proven live against a planted wrapped-phrase fixture before its zero was trusted
result: pass
source: automated
coverage_id: 06-04/D2

### 20. BENCH-03 status recorded explicitly, not closed by implication
expected: 06-LIVE-VERIFICATIONS.md carries exactly one BENCH03_STATUS token, transcribed from the maintainer's checkpoint selection with independently re-verified evidence
result: pass
source: automated
coverage_id: 06-04/D3

### 21. Every framing occurrence in agent-facing files enumerated with a verdict
expected: Every framing occurrence across .claude/CLAUDE.md, .planning/PROJECT.md and .planning/STATE.md is enumerated with a sweep/keep-historical/keep-divergent verdict and reason in 06-MEMORY-SWEEP.md
result: pass
source: automated
coverage_id: 06-05/D1

### 22. Agent-facing files carry no present-tense parity framing
expected: The three agent-facing files carry no present-tense or forward-looking parity/port/drop-in framing (CLAUDE_MD_FRAMING_TOTAL=0, STATE_MD_RETIRED_CORE_VALUE=0, CORE_VALUE_EQUAL=true, COMPATIBILITY_BULLET_STALE_CAPABILITY_REFS=0)
result: pass
source: automated
coverage_id: 06-05/D2

### 23. Generated CLAUDE.md blocks re-synced with markers byte-identical
expected: Generated CLAUDE.md project block re-synced from PROJECT.md; stack block swept in place under a recorded keep-divergent verdict; all 14 marker lines and 25 heading lines byte-identical to the pre-plan file
result: pass
source: automated
coverage_id: 06-05/D3

## Summary

total: 23
passed: 22
issues: 0
pending: 0
skipped: 1
blocked: 0

## Gaps

[none yet]
