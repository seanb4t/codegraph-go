---
phase: 02-guards-ci-wiring-docs-burn-down
plan: 06
subsystem: docs
tags: [release-docs, security-docs, dependency-tree, provenance, census, go-sdk]

# Dependency graph
requires:
  - phase: 02-guards-ci-wiring-docs-burn-down
    provides: "prior plans in this phase (01/03/05) wired guards and CI checks this plan documents alongside"
provides:
  - "docs/RELEASE.md § 2 reshaped to never quote a raw dependency count again (D-13)"
  - "positive-controlled, whole-repo provenance census proving zero live 'provenance over checksums file' claims remain (D-14), GH #14 closed"
  - "SECURITY.md states the advisory tool-vuln job's scope in one sentence (D-15)"
affects: [docs, release-process, security-policy]

# Actuals (#2632)
actuals:
  tokens: 1125
  tasks: 3
  commits: 2
  plan_head_before: 40dec267d6264e98932b106bcf7e68065986e9e9

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Positive-controlled census: plant an untracked control file matching the exact incorrect-claim shape, prove both search patterns detect it and detect its absence after removal, only then trust a 'zero hits' result (rule 84d1gfpywd)."
    - "Wording-only doc fixes get no test/guard/Taskfile change (v0.13.0 Phase 12 ruling, carried into D-13/D-15)."

key-files:
  created: []
  modified:
    - docs/RELEASE.md
    - SECURITY.md

key-decisions:
  - "Task 1 (DOCS-09 census) produced no rewrite commit — RESEARCH's zero-live-hits finding was confirmed by the census, not contradicted, so there is nothing to commit for that task beyond the GH #14 close."
  - "Every BROAD-pattern hit classified correct-transport-wording or historical-correction; none reworded."

requirements-completed: [DOCS-09, DOCS-10]
# DOCS-08 is declared by both this plan and 02-07 (shared-ID gate, #2388): it
# stays unmarked here and completes when 02-07 (the last declaring plan)
# finishes, even though this plan's DOCS-08 acceptance criteria all pass.

coverage:
  - id: D1
    description: "Positive-controlled provenance census across the whole repo, zero live-incorrect hits, GH #14 closed on the evidence"
    requirement: "DOCS-09"
    verification:
      - kind: other
        ref: "census script (rg -n -U -i, NARROW + BROAD patterns) — planted control matched (1/1) while present, matched by neither pattern after removal, run against README.md/SECURITY.md/docs//.github/workflows//internal/upgrade/ excluding CHANGELOG.md"
        status: pass
      - kind: other
        ref: "gh issue view 14 --json state,comments — CLOSED with 1 comment citing the census"
        status: pass
    human_judgment: false
  - id: D2
    description: "docs/RELEASE.md § 2 reshaped: no raw dependency counts, modelcontextprotocol/go-sdk credited, deliberate direct requires named, historical callout intact"
    requirement: "DOCS-08"
    verification:
      - kind: other
        ref: "plan Task 2 <verify> block 1-4 (negative-grep counts/mark3labs, positive-grep required terms, historical-callout count, commit message/file-scope check) — all re-run and passing"
        status: pass
    human_judgment: false
  - id: D3
    description: "SECURITY.md gains exactly one sentence stating the advisory tool-vuln job's scope"
    requirement: "DOCS-10"
    verification:
      - kind: other
        ref: "plan Task 3 <verify> block 1-3 (diff size/content check, no-exposure-id check, commit message/file-scope check) — all re-run and passing"
        status: pass
    human_judgment: false

duration: 20min
completed: 2026-09-16
status: complete
---

# Phase 02 Plan 06: Docs burn-down — dependency counts, provenance census, tool-vuln scope Summary

**Reshaped the dependency-tree paragraph to stop quoting counts that go stale on every `go.mod` bump, proved by a positive-controlled census that no live text still claims provenance is attested over the checksums file (closing GH #14), and added one sentence to `SECURITY.md` naming what the advisory `tool-vuln` job actually covers.**

## Performance

- **Duration:** ~20 min
- **Started:** 2026-09-15 (session start)
- **Completed:** 2026-09-16T02:11:40Z
- **Tasks:** 3/3 completed
- **Files modified:** 2 (`docs/RELEASE.md`, `SECURITY.md`)

## Accomplishments

- **DOCS-09 census (Task 1):** Ran a positive-controlled, whole-repo census for "provenance attested over the checksums file" wording. Planted an untracked control sentence at `docs/zz-census-control.md` matching the incorrect-claim shape; both the NARROW and BROAD `rg -n -U -i` patterns detected it (1 hit each) while present and detected zero hits for it after `rm -f` removal inside an EXIT trap — proving the census can see before trusting its "clean" result. The final NARROW run found exactly one hit (`docs/RELEASE.md:145`), inside the existing `Corrected 2026-08-01` historical-correction callout — not a live claim. The BROAD run found 7 additional co-occurrence spans, all classified `correct-transport-wording` or `historical-correction` (table below) — zero live-incorrect claims, matching RESEARCH's prediction. No rewrite was made or needed. Closed GH #14 with a comment citing the census command, the positive-control result, the classified hit table, and what became of the issue's three originally-cited (now-stale) line references.
- **DOCS-08 (Task 2):** Reshaped `docs/RELEASE.md` § 2 "Dependency tree (DIST-05)" — removed every raw count (`134`, `107`, `27`, `14`, `13`), kept the wide-but-shallow framing (one grammar module per supported language, listed by name, plus the shared `tree-sitter/go-tree-sitter` binding), named the current deliberately-chosen direct requires from `go.mod` (`cockroachdb/pebble/v2`, `modelcontextprotocol/go-sdk`, `spf13/cobra`, `fsnotify/fsnotify`, `sigstore/sigstore-go`, `tailscale/hujson`, `google.golang.org/protobuf`, `gonum.org/v1/gonum`, the `charm.land/{bubbletea,bubbles,lipgloss}/v2` TUI trio, `go.uber.org/goleak`, and a handful of `golang.org/x/*` packages), and pointed readers at `go list -m all` / `go mod graph` for the live picture. `mark3labs` no longer appears anywhere in the file. The `Corrected 2026-08-01` callout and the CGo-exception/SBOM-authoritative paragraphs are byte-identical to before.
- **DOCS-10 (Task 3):** Appended exactly one sentence to `SECURITY.md`'s "Dependency vulnerabilities — two scanners, two disjoint trees" bullet: `govulncheck` blocks merges on the main module graph only, and the isolated tool modfiles (`go.tool*.mod`) are scanned separately by the advisory `tool-vuln` job in `ci.yml`, which reports and never fails the build. Names no vulnerability id or package; the diff is additions-only (4 lines added, 0 removed).

## Census: classified BROAD-pattern hits

| File:Line | Span (paraphrased) | Class | Action |
|---|---|---|---|
| `.github/workflows/release.yml:277-279` | comment: "...this Action parses the checksums file's own shasum-format lines as its subject list" | correct-transport-wording | kept |
| `docs/RELEASE-PROCEDURES.md:140-143` | "Attest — ...runs...over the SAME 8 payloads the checksums file covers..." | correct-transport-wording | kept |
| `docs/RELEASE-PROCEDURES.md:260` | "...the checksums file was not an attested subject..." | historical-correction (`Corrected 2026-08-01` block) | kept |
| `docs/RELEASE-PROCEDURES.md:681-682` | "...raw binaries, .zip archives, checksums file, cosign bundles, SBOMs, and build-provenance attestation — is already complete..." | correct-transport-wording (artifact list, no claim) | kept |
| `docs/RELEASE.md:101` | "Note the attestation's subjects are the binaries, not the checksums file." | correct-transport-wording (explicitly correct) | kept |
| `docs/RELEASE.md:145` | (NARROW hit — see below) | historical-correction | kept |
| `docs/RELEASE.md:147` | "...checksums file was not an attested subject. Following the old instructions..." | historical-correction (same block) | kept |

NARROW-pattern hit (`docs/RELEASE.md:145`) sits inside the `Corrected 2026-08-01` block that already documents this exact error as fixed — not live. **Zero live-incorrect hits found; zero rewrites made.**

## GH #14

Closed via `gh issue close 14 --comment "..."` citing the census command, the positive-control result, the classified table above, and confirmation that all three of the issue's originally-cited sites (`.github/workflows/release.yml:327`, `docs/RELEASE.md:26-28`, `docs/RELEASE-PROCEDURES.md:120-122`) were corrected by the `actions/attest-build-provenance` migration (their line numbers are stale but the underlying claim is fixed). `gh issue view 14 --json state` confirms `CLOSED`.

## Task Commits

Each task with a rewrite was committed atomically:

1. **Task 1: Provenance census, GH #14 closed** — no commit (no live-incorrect hit found; census evidence and issue-close link recorded in this SUMMARY per plan instructions).
2. **Task 2: Reshape docs/RELEASE.md § 2** — `99bcb512` (docs)
3. **Task 3: One sentence in SECURITY.md** — `62e8b299` (docs)

**Plan metadata:** committed alongside this SUMMARY.

## Files Created/Modified

- `docs/RELEASE.md` — § 2 "Dependency tree (DIST-05)" prose reshaped; no other section touched.
- `SECURITY.md` — one sentence appended to the two-scanner bullet; nothing else changed.

## Decisions Made

- Task 1's zero-rewrite outcome is documented here rather than forced into a commit — the plan explicitly anticipated this as the expected result of RESEARCH's finding.
- All census classifications are `correct-transport-wording` or `historical-correction`; none required rewording to the "binaries as subjects" standard.
- **Shared-ID gate (#2388):** DOCS-08 is declared by both this plan (02-06) and 02-07. This plan's DOCS-08 acceptance criteria all pass, but `requirements.ready-ids` correctly reports it `blocked` because 02-07 has not yet produced a SUMMARY — DOCS-08 was NOT marked complete in `REQUIREMENTS.md` by this plan. It will be marked when 02-07 finishes. DOCS-09 and DOCS-10 (declared only here) were marked complete immediately.

## Deviations from Plan

None - plan executed exactly as written. All `<verify>` blocks for all three tasks were re-run after implementation and passed; acceptance criteria for each task were satisfied on the first attempt.

## Issues Encountered

None.

## TDD Note

`workflow.tdd_mode` is enabled project-wide, but this plan's frontmatter is `type: execute` and every task is a documentation edit or a verification census — none add behavior, so no task is "behavior-adding" per the MVP+TDD gate and no RED/GREEN/REFACTOR commit sequence applies. Task 1's planted positive control (matched while present, absent after removal) is this plan's own RED/GREEN proof for the one claim that needed verification (DOCS-09); Tasks 2 and 3 are wording changes that get no test by explicit ruling (D-13/D-15, v0.13.0 Phase 12).

## Known Stubs

None.

## Threat Flags

None — this plan's threat register items (T-02-06-01 through T-02-06-07) were all addressed as `mitigate` and verified inline (control-detection proof, no-exposure-id grep, historical-callout-count grep, git-status-empty grep, GH-issue-closed-with-comment grep, no-CI-skip grep); no new surface was introduced beyond the two prose files.

## User Setup Required

None.

## Self-Check: PASSED

- `docs/RELEASE.md` exists on disk: FOUND
- `SECURITY.md` exists on disk: FOUND
- Commit `99bcb512` (Task 2) present in git history: FOUND
- Commit `62e8b299` (Task 3) present in git history: FOUND
- `gh issue view 14 --json state` reports: `CLOSED`
