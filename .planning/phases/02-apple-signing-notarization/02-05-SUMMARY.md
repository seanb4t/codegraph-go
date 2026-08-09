---
phase: 02-apple-signing-notarization
plan: 05
subsystem: docs
tags: [docs, gatekeeper, notarization, release-verification, spctl]

# Dependency graph
requires:
  - phase: 02-apple-signing-notarization
    provides: "plan 02-01's task verify:gatekeeper (D-19 oracle) and 02-EVIDENCE.md (the RED baseline, Assumption A2 CONFIRMED, and the four/six insufficient-check enumeration) — this plan's every claim traces to that recorded measurement"
provides:
  - "docs/RELEASE.md §1d — the Gatekeeper verification section: a boundary-row applicability table, the exact three-part guarantee (notarized, online-verified, not stapled) scoped to that table, the offline-first-launch known limitation, the download/xattr-write/xattr-read-back/spctl reproduction commands, and a subordinated six-item not-verification list"
  - "The verbatim guarantee phrase 'notarized, online-verified, not stapled' as a quotable project claim"
  - "docs/RELEASE.md's stale pre-release status note replaced with the true current release state (v0.5.1, v0.6.0 both published and un-notarized)"
affects: [02-06, 02-07]

# Actuals (#2632)
actuals:
  tokens: 3006
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "Boundary-row applicability table (3 rows, not per-tag) for a guarantee that becomes true only from a future release onward — avoids both an unqualified present-tense overclaim and a hand-maintained release ledger"
    - "Verdict from exit status alone, documented and reinforced with a worked false-positive example (grep for 'accepted' matching inside a rejection string), mirroring the Taskfile oracle's own discipline in prose"
    - "Non-gating recorded observation documented in exactly one place (the not-verification list), never repeated, to avoid giving a non-verification a bigger footprint than the verification beside it"

key-files:
  created: []
  modified:
    - docs/RELEASE.md

key-decisions:
  - "Removed the literal string 'spctl -a -vv -t install' from the applicability table's prose (used a paraphrase instead) so the reproduction section's xattr-read-back-before-assessment ordering constraint holds across the WHOLE file, not just within the reproduction snippet — the table would otherwise have been the first occurrence of the assessment command, ahead of the required xattr read-back."
  - "v0.6.0 exists (published 2026-08-09, after v0.5.1, still un-notarized) and is named alongside v0.5.1 in the status note and applicability table rather than only the plan-authored v0.5.1, since the document must describe the actual current state, not the state at context-gathering time. The boundary-row table design already absorbs future un-notarized releases without further edits."
  - "The synthetic-quarantine xattr reproduction commands are documented unconditionally (no browser-download fallback branch), because 02-EVIDENCE.md's Assumption A2 is CONFIRMED, not REFUTED — the plan's conditional instruction only required a fallback in the REFUTED case."

requirements-completed: [SIGN-02, SIGN-03]

coverage:
  - id: D1
    description: "docs/RELEASE.md §1d Gatekeeper section: applicability table (3 boundary rows, pending marker on the first-notarized row), exact guarantee phrase scoped by reference to the table, operational meaning of each of the three guarantee parts, offline-first-launch limitation named with DIST-06, xattr-write→xattr-read-back→spctl reproduction commands (never .zip), pass/fail examples with source= line + exit status, exit-status-only verdict rationale, and a six-item not-verification list (green CI, codesign -dvv, notarytool history, never-quarantined assessment, -t exec, syspolicy_check) in a subordinated details block."
    requirement: SIGN-02
    verification:
      - kind: automated
        ref: "rg -c 'notarized, online-verified, not stapled' docs/RELEASE.md == 1"
        status: pass
      - kind: automated
        ref: "rg -c 'spctl -a -vv -t install' docs/RELEASE.md == 1; rg -c 'spctl -a -vv -t exec' docs/RELEASE.md == 1 (inside not-verification list only)"
        status: pass
      - kind: automated
        ref: "rg -n 'xattr -p com.apple.quarantine|spctl -a -vv -t install' docs/RELEASE.md — xattr readback line (231) precedes spctl line (235), and is the first occurrence of either string in the file"
        status: pass
      - kind: automated
        ref: "rg -c 'source=Notarized Developer ID' docs/RELEASE.md == 1; rg -c 'DIST-06' docs/RELEASE.md == 2"
        status: pass
      - kind: manual_procedural
        ref: "rg -n '\\.zip' within the §1d section's line range (163-300) — zero occurrences"
        status: pass
    human_judgment: true
    rationale: "Whether the prose overstates evidence is a judgment call against 02-EVIDENCE.md's recorded measurements, not a property an automated regex alone can certify — the automated checks above catch the mechanical failure modes (wrong command, wrong ordering, forbidden substring), and this pass records the additional judgment: no sentence claims a measurement that has not happened."
  - id: D2
    description: "docs/RELEASE.md's rest-of-document sweep (Task 2): stale pre-release status note replaced; reproducibility section scopes the darwin-signature claim as pending rather than asserted; codegraph-upgrade-as-consumer section distinguishes the detached Sigstore signature from the embedded Apple signature and states the former does nothing for Gatekeeper; §1 asset list cross-checked (no new asset type — notarization mutates the existing raw binary in place, per D-04)."
    requirement: SIGN-02
    verification:
      - kind: automated
        ref: "rg -n \"no .v\\*. tag has been pushed\" docs/RELEASE.md — zero matches (stale claim confirmed removed)"
        status: pass
      - kind: automated
        ref: "rg -c 'reproduc' docs/RELEASE.md == 9 (existing claim scoped, not deleted)"
        status: pass
      - kind: manual_procedural
        ref: "gh release view v0.5.1 / v0.6.0 --json assets — cross-checked both against §1's asset list; same 8-asset-per-4-platforms shape on both, no new asset type introduced by anything currently published"
        status: pass
    human_judgment: true
    rationale: "Confirming the asset-list cross-check required a live gh query against the real GitHub Releases API, not a property derivable from the repo alone."

duration: ~20min active work
completed: 2026-08-09
status: complete
---

# Phase 2 Plan 05: Publish the Gatekeeper Guarantee in docs/RELEASE.md Summary

**Added `docs/RELEASE.md` §1d — a Gatekeeper verification section stating the guarantee exactly as strong as the evidence supports (notarized, online-verified, not stapled, scoped to releases that don't exist yet) and handing the reader the exact commands `verify:gatekeeper` runs, with the xattr read-back mandatory before every assessment.**

## Performance

- **Duration:** ~20 min active work
- **Started / Completed:** 2026-08-09
- **Tasks:** 2/2 completed
- **Files modified:** 1 (`docs/RELEASE.md`)

## Accomplishments

- **§1d Gatekeeper section**, placed inside §1 alongside cosign/provenance/SBOM: a 3-row boundary applicability table (through the last un-notarized release; the first notarized release, tag pending 02-07; every release after), the exact guarantee phrase quotable verbatim and scoped to that table, what each of the three parts means operationally, the offline-first-launch limitation named with a footnote on why stapling is categorically impossible here (bare Mach-O / `.zip`, no staple command), the full reproduction sequence (download → xattr write → xattr read-back → `spctl -a -vv -t install`, verdict from exit status alone), worked pass/fail examples, and a subordinated six-item "what does not count as verification" `<details>` block.
- **Fixed a self-inflicted ordering bug before committing**: the applicability table's first draft named the literal `spctl -a -vv -t install` command in prose, which would have made it the first occurrence of that string in the whole file — ahead of the mandatory xattr read-back in the reproduction snippet, violating the plan's own line-order acceptance criterion. Reworded the table to a paraphrase instead.
- **Fixed a line-wrap bug**: the exact guarantee phrase `notarized, online-verified, not stapled` initially split across a markdown soft-wrap, which broke the `rg -c` substring match. Reflowed to keep the phrase on an unbroken line.
- **Reconciled the rest of the document** (Task 2): replaced the stale "no `v*` tag has been pushed yet" status note (two releases — `v0.5.1` and `v0.6.0` — are now real and published, both currently un-notarized); scoped the reproducibility claim to note the darwin binaries will carry an unreproducible embedded signature once notarized (marked pending, not asserted); added a one-sentence distinction in the `codegraph upgrade`-as-consumer section between the detached Sigstore signature it checks and the embedded Apple signature notarization adds, stating the former does nothing for Gatekeeper.
- Confirmed via `gh release view` that `v0.6.0`'s published asset shape is identical to `v0.5.1`'s (same 8-assets-per-4-platforms pattern) — no new asset type has appeared, consistent with D-04 (notarization mutates the existing raw binary, not a new file).
- Confirmed via `task --list-all` that no markdown/doc lint target exists in `Taskfile.yml`.

## Task Commits

1. **Task 1: The guarantee statement and the reproduction commands** — `8b7f131` (docs)
2. **Task 2: Reconcile the rest of the document with what now ships** — `b028290` (docs)

**Plan metadata:** this SUMMARY.md commit (worktree mode — STATE.md/ROADMAP.md excluded, orchestrator-owned)

## Files Created/Modified

- `docs/RELEASE.md` — new §1d Gatekeeper verification subsection (~140 lines); updated §1 opening asset list and "All three"→"All four" verification-steps count; rewritten status note; scoped reproducibility-posture bullet; added Sigstore-vs-Apple-signature distinction to the `codegraph upgrade` section.

## Decisions Made

- **Named `v0.6.0` alongside `v0.5.1`, not just `v0.5.1`.** The plan's context (`02-CONTEXT.md`) was authored before `v0.6.0` published; the document must describe the actual current release state. The boundary-row table design (not per-tag rows) absorbs this and any future un-notarized release without needing further edits — it already reads "at least `v0.5.1` and `v0.6.0`" rather than an exhaustive, staleness-prone list.
- **Documented the synthetic-xattr commands unconditionally**, per `02-EVIDENCE.md`'s Assumption A2 verdict: CONFIRMED (a synthetic quarantine attribute produces the identical `spctl` verdict as a genuine browser download on a byte-identical file). The plan's browser-download fallback branch only applies if A2 had been REFUTED.
- **Removed the literal `spctl -a -vv -t install` string from the applicability table** in favor of a paraphrase ("the Gatekeeper install-time assessment below") specifically to preserve the file-wide invariant that the xattr read-back line is the first occurrence of either the readback or the assessment command — a mechanical acceptance criterion the table's original wording would have silently violated.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Guarantee phrase split by markdown line-wrap, breaking its own acceptance criterion**
- **Found during:** Task 1 self-check, running `rg -c 'notarized, online-verified, not stapled' docs/RELEASE.md` before committing
- **Issue:** The guarantee sentence wrapped mid-phrase (`**notarized,` / `online-verified, not stapled**.`), so the exact-phrase `rg` match returned zero.
- **Fix:** Reflowed the sentence so the three-part phrase sits on one unbroken line.
- **Files modified:** `docs/RELEASE.md`
- **Verification:** `rg -c 'notarized, online-verified, not stapled' docs/RELEASE.md` returns 1.
- **Committed in:** `8b7f131` (caught before commit, not a separate follow-up)

**2. [Rule 1 - Bug] Applicability table's prose put the assessment command ahead of the mandatory xattr read-back, file-wide**
- **Found during:** Task 1 self-check, running the line-order check (`rg -n` on both strings, comparing line numbers)
- **Issue:** The table (placed first in the section per the plan's own explicit ordering requirement) named `spctl -a -vv -t install` in prose describing why §d doesn't apply to un-notarized releases — making that the first occurrence of the assessment command anywhere in the file, ahead of the xattr read-back that appears later in the reproduction snippet. This violated the acceptance criterion that the read-back precede the first assessment-command occurrence by line number.
- **Fix:** Reworded the table cell to a paraphrase ("the Gatekeeper install-time assessment below rejects these darwin binaries by design") that carries the same information without the literal command string.
- **Files modified:** `docs/RELEASE.md`
- **Verification:** `rg -n 'xattr -p com.apple.quarantine|spctl -a -vv -t install' docs/RELEASE.md` shows the readback at line 231, the assessment at line 235 — the only two occurrences in the file, in the correct order.
- **Committed in:** `8b7f131` (caught before commit, not a separate follow-up)

---

**Total deviations:** 2 auto-fixed (both Rule 1 — self-caught before their task commit landed, no scope creep).

## Issues Encountered

None.

## Known Stubs

None.

## Next Phase Readiness

- `docs/RELEASE.md` §1d's applicability table's "first notarized release" row carries an explicit `tag pending, filled in by plan 02-07 once it publishes` marker — plan 02-07 must replace that placeholder with the real tag once it ships the first notarized release, and should re-verify at that point that the guarantee's scope (previously bounded to "from the first notarized release onward") now correctly includes that real tag.
- The reproducibility-posture bullet on darwin signatures is marked `pending` (unmeasured) for the same reason — plan 02-07 is what turns it from a description of intent into a measured fact.
- No blockers for plan 02-06 (post-release GREEN Gatekeeper check) or 02-07 (first notarized release) — this plan only touched `docs/RELEASE.md`.

---
*Phase: 02-apple-signing-notarization*
*Completed: 2026-08-09*
