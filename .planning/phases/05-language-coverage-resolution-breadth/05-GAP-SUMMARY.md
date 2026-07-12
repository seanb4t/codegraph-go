---
phase: 05-language-coverage-resolution-breadth
plan: GAP-CLOSURE
subsystem: indexing
tags: [java, csharp, golden-parity, real-repo-validation, gap-closure, determinism]

# Dependency graph
requires:
  - phase: 05-language-coverage-resolution-breadth
    plan: 04
    provides: internal/indexer/javaextract, languages_java.go, testdata/golden/parity_java_test.go (self-skipping harness, previously only smoke-tested against a synthetic 3-file fixture)
  - phase: 05-language-coverage-resolution-breadth
    plan: 05
    provides: internal/indexer/csharpextract, languages_csharp.go, testdata/golden/parity_csharp_test.go (self-skipping harness, previously never exercised against any corpus)
provides:
  - Real-repo D-12 evidence for Java (temporal sdk-java, 1,271 files) and C# (serilog, 216 files), closing the ROADMAP Success Criterion 1 gap identified by 05-VERIFICATION.md
affects: [05-VERIFICATION.md (gap closed), docs/LANGUAGE-CAPABILITY-MATRIX.md's D-11 Java/C# rows (real-repo evidence now exists to cite)]

# Tech tracking
tech-stack:
  added: []
  patterns: []

key-files:
  created:
    - .planning/phases/05-language-coverage-resolution-breadth/05-GAP-SUMMARY.md
  modified:
    - .planning/phases/05-language-coverage-resolution-breadth/05-VERIFICATION.md

key-decisions:
  - "No source code changes were required. Both TestGoldenParity_Java and TestGoldenParity_CSharp (built in 05-04/05-05) ran cleanly against real, external, non-trivial repositories on the first attempt — no extraction or resolution bug was surfaced by real-world Java or C# source that unit/synthetic-fixture tests had missed."
  - "Java corpus: /Volumes/Code/github.com/temporalio/sdk-java (Temporal Java SDK) at commit f919926a2d67c10c34fee4b19eed1c605d4223a4 (local checkout provided by the user, not cloned by this session) — 1,223 real .java files across a genuine multi-module Gradle project (interfaces, classes, deep inheritance, heavy cross-package calls) — chosen because it was already available locally and is exactly the kind of large, structurally-rich, real-world Java codebase D-12 asks for."
  - "C# corpus: https://github.com/serilog/serilog (Apache-2.0) at commit 07d39cfb2928076ecd902a61d295f90d74fe1fa5, shallow-cloned to scratch — 216 real .cs files, a clean, canonical, moderate-size C# library with classes/interfaces/inheritance and cross-file calls. Chosen over Newtonsoft.Json for a faster, cleaner shallow clone; not committed to the repo (external, per the plan's constraints) and the scratch clone was ephemeral to the session."
  - "No wiring of the parity tests to a committed corpus path was done — mirroring Python/TS-JS's own precedent (05-06/05-07), the harnesses remain self-skipping via CODEGRAPH_JAVA_CORPUS/CODEGRAPH_CSHARP_CORPUS in the committed repo. The recorded counts below (plus this file) constitute the durable evidence, exactly as Python's litellm/types and TS-JS's ccstatusline evidence was recorded in their own SUMMARYs rather than as a checked-in corpus."

requirements-completed: [LANG-02, LANG-03]

coverage:
  - id: G1
    description: "Java (temporal sdk-java, 1,223 real .java files) indexed via the real indexer.Run pipeline through TestGoldenParity_Java: struct/interface/method node kinds all non-zero, calls/embeds/implements edges all non-zero, deterministic rebuild confirmed (test's own internal two-pass byte-identical-count assertion, plus re-run confirmed identical results across separate go test invocations)"
    requirement: "LANG-02"
    verification:
      - kind: integration
        ref: "testdata/golden/parity_java_test.go#TestGoldenParity_Java (CODEGRAPH_JAVA_CORPUS=/Volumes/Code/github.com/temporalio/sdk-java) — files=1271 nodes=14405 edges=19037 nodeKinds={file:1223 interface:500 method:9539 struct:2046} edgeKinds={calls:6141 contains:12219 embeds:267 implements:410}, deterministic rebuild confirmed"
        status: pass
    human_judgment: false
  - id: G2
    description: "C# (serilog, 216 real .cs files) indexed via the real indexer.Run pipeline through TestGoldenParity_CSharp: struct/interface/method node kinds all non-zero, calls/embeds/implements edges all non-zero, deterministic rebuild confirmed"
    requirement: "LANG-03"
    verification:
      - kind: integration
        ref: "testdata/golden/parity_csharp_test.go#TestGoldenParity_CSharp (CODEGRAPH_CSHARP_CORPUS=<scratch>/serilog@07d39cf) — files=216 nodes=1818 edges=1633 nodeKinds={file:216 interface:15 method:969 struct:242} edgeKinds={calls:387 contains:1227 embeds:17 implements:2}, deterministic rebuild confirmed"
        status: pass
    human_judgment: false
  - id: G3
    description: "No extraction or resolution regression introduced; full repo suite (all packages + testdata/golden, including pre-existing Go/Java/C#/Python/TS-JS golden-parity fixtures) remains green"
    requirement: "LANG-02"
    verification:
      - kind: other
        ref: "go build ./... && go vet ./... && go test ./... ./testdata/golden/... -count=1 (27 test targets, all ok)"
        status: pass
    human_judgment: false

duration: 15min
completed: 2026-07-12
status: complete
---

# Phase 5 Gap Closure: Java + C# Real-Repo D-12 Validation (LANG-02, LANG-03) Summary

**Closes the sole remaining Phase 5 verification gap (05-VERIFICATION.md's ROADMAP Success Criterion 1 finding): ran the pre-existing, previously-unexercised `TestGoldenParity_Java`/`TestGoldenParity_CSharp` self-skipping harnesses against two real, external, structurally-rich repositories — Temporal's Java SDK (1,223 files) and Serilog (216 files) — confirming genuine extraction, cross-file resolution, and deterministic rebuilds, matching the same evidence bar Python (litellm/types) and TS-JS (ccstatusline) already met in 05-06/05-07.**

## Performance

- **Duration:** ~15 min
- **Completed:** 2026-07-12
- **Tasks:** 2 (Java corpus run, C# corpus run) + evidence recording
- **Files modified:** 2 (1 created, 1 modified)

## Accomplishments

- Ran `TestGoldenParity_Java` against a **local checkout of the real Temporal Java SDK** (`temporalio/sdk-java`, 1,223 `.java` files, commit `f919926a2d67c10c34fee4b19eed1c605d4223a4`): **files=1271, nodes=14405, edges=19037**, node kinds `{file:1223, interface:500, method:9539, struct:2046}`, edge kinds `{calls:6141, contains:12219, embeds:267, implements:410}`. Every shape check (`struct`/`method` nodes non-zero, `calls` edges non-zero) passed on real code. The test's own internal second-pass rebuild confirmed byte-identical aggregate counts; re-invoking the whole test binary a second time reproduced identical output.
- Shallow-cloned **Serilog** (`github.com/serilog/serilog`, Apache-2.0, commit `07d39cfb2928076ecd902a61d295f90d74fe1fa5`) to scratch and ran `TestGoldenParity_CSharp` against it: **files=216, nodes=1818, edges=1633**, node kinds `{file:216, interface:15, method:969, struct:242}`, edge kinds `{calls:387, contains:1227, embeds:17, implements:2}`. All shape checks passed; deterministic rebuild confirmed by the test's own two-pass assertion.
- **No extraction or resolution bugs surfaced.** Both extractors (`javaextract`, `csharpextract`, built in 05-04/05-05) worked correctly against real, non-trivial, unmodified third-party source on the first run — no code changes were needed to close this gap, only running the pre-built harness against real corpora and recording the results.
- Ran the full repo test suite (`go build ./... && go vet ./... && go test ./... ./testdata/golden/... -count=1`): all 27 test targets pass, including the pre-existing Go/Java/C#/Python/TS-JS golden-parity fixtures — no regression.
- Updated `05-VERIFICATION.md`'s frontmatter and body to close the previously-recorded gap, citing this evidence.

## Task Commits

1. `docs(05-gap): close SC1 real-repo validation gap for Java + C#` — this SUMMARY + `05-VERIFICATION.md` update (evidence-only change; no source files modified since no bugs were found)

## Files Created/Modified
- `.planning/phases/05-language-coverage-resolution-breadth/05-GAP-SUMMARY.md` - this evidence record
- `.planning/phases/05-language-coverage-resolution-breadth/05-VERIFICATION.md` - gap marked closed, frontmatter/body updated with the Java/C# real-repo evidence

## Decisions Made
See `key-decisions` in frontmatter. Highlights:
- No source changes needed — both harnesses already correctly implemented D-12's fallback methodology (05-04/05-05); this session only supplied real corpora and recorded the resulting counts, exactly mirroring how 05-06/05-07 closed the same gap for Python/TS-JS.
- Chose `temporalio/sdk-java` (already available locally, no clone needed) for Java and `serilog/serilog` (small, clean, Apache-2.0, fast shallow clone) for C#, rather than the larger `Newtonsoft.Json` alternative — sufficient structural richness (classes, interfaces, inheritance, cross-file calls) at lower clone/index cost.
- The external corpora (`sdk-java`, the scratch `serilog` clone) were **not** committed to codegraph-go, per the plan's explicit constraint — only this evidence summary and the VERIFICATION.md update were committed.

## Deviations from Plan

### Auto-fixed Issues

None — plan executed exactly as written. No bugs found on real Java or C# code; no architectural changes needed.

## Issues Encountered

None. Both `TestGoldenParity_Java` and `TestGoldenParity_CSharp` ran cleanly on the first attempt against real, unmodified, external repositories, with all shape checks and the determinism check passing without any code changes.

## User Setup Required

None. The evidence in this file (plus the updated `05-VERIFICATION.md`) is the durable record. To reproduce:
```bash
CODEGRAPH_JAVA_CORPUS=/path/to/temporalio/sdk-java \
  go test ./testdata/golden/... -run TestGoldenParity_Java -v

CODEGRAPH_CSHARP_CORPUS=/path/to/serilog \
  go test ./testdata/golden/... -run TestGoldenParity_CSharp -v
```

## Next Phase Readiness

- ROADMAP Success Criterion 1 / D-12 is now closed for all four priority-4 languages (Go's own golden-parity fixture predates Phase 5; Python/TS-JS closed in 05-06/05-07; Java/C# closed here) — no further real-repo validation work is outstanding for Phase 5.
- `go build ./...`, `go vet ./...`, and `go test ./... ./testdata/golden/... -count=1` all pass across the full repo (27 targets) — no regression introduced by this gap-closure session.

---
*Phase: 05-language-coverage-resolution-breadth*
*Completed: 2026-07-12*

## Self-Check: PASSED

Verified `.planning/phases/05-language-coverage-resolution-breadth/05-GAP-SUMMARY.md` exists on disk. `05-VERIFICATION.md` update confirmed present. No commit hashes to verify yet (commit created in the state-update step below); git log check performed after final commit.
