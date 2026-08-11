---
phase: 04-codegraph-upgrade-homebrew
plan: 02
subsystem: release-packaging
tags: [goreleaser, homebrew-cask, taskfile, rehearsal, evidence-correction]

# Dependency graph
requires:
  - phase: 04-codegraph-upgrade-homebrew (plan 01)
    provides: structural brew-managed-install detection (D-02/D-03) that made the Phase-3 marker file obsolete
provides:
  - .goreleaser.yaml cask hooks with no marker file and a freshness-based man-page assertion (UF-5 closed)
  - Taskfile.yml release:rehearse-cask with no marker assertions and two independent freshness re-checks (pre-install, pre-reinstall)
  - a corrected 03-EVIDENCE.md that no longer claims a failed install can strand a marker
affects: [04-03, 04-04, 04-05, 04-06]

actuals:
  tokens: 6900
  tasks: 3
  commits: 4

tech-stack:
  added: []
  patterns:
    - "Per-path before/after freshness snapshot (mtime + size) as a UF-5-class fix: distinguishes 'this run produced fresh output' from 'output happens to be present', replacing both a removed marker-file check and a rejected wall-clock oracle."
    - "Per-marker one-creation/one-consumer discipline for shell rehearsal checks: two independently-created mktemp markers (pre-install, pre-reinstall) rather than one reused marker, so neither check's ability to fire depends on an unstated assumption about a hook under simultaneous edit."

key-files:
  created: []
  modified:
    - .goreleaser.yaml
    - Taskfile.yml
    - .planning/phases/03-homebrew-tap-cask/03-EVIDENCE.md
    - .planning/todos/completed/2026-08-10-03-evidence-falsely-claims-a-failed-install-can-strand-the-phase-4-sentinel.md
    - .planning/todos/completed/2026-08-10-post-install-man-page-assertion-can-be-satisfied-by-stale-pages.md

key-decisions:
  - "Marker file removed outright (D-02), not merely corrected — Phase 4's brew-managed-install detection is structural (04-01) and never needed it."
  - "Freshness, not growth, is the man-page assertion's property: a brew reinstall rewrites the same pages, so a count-grew check would break Step 5d's idempotency rehearsal; a per-path mtime+size comparison against a baseline passes on reinstall and still fails the UF-5 case (ran, exited 0, wrote nothing)."
  - "Two independent rehearsal markers (MAN_BASELINE_MARKER, MAN_REINSTALL_MARKER), not one reused marker — reusing the pre-install marker for the reinstall check would leave it pre-satisfied by the first install's own output regardless of whether the reinstall wrote anything."

patterns-established:
  - "Positive assertion of freshness (repo rule 84d1gfpywd extended): a guard proving 'this run produced the artifact' rather than 'the artifact exists' is the durable shape for any post-install/post-run check reading a directory that could carry residue from a prior run."

requirements-completed: [UPGR-02]

coverage:
  - id: D1
    description: "Cask hooks no longer write or remove a .codegraph-brew-install marker file; the falsified line-451 comment is corrected to describe structural detection."
    requirement: "UPGR-02"
    verification:
      - kind: unit
        ref: "go test ./internal/upgrade/ -run '^TestHomebrewCask' — 5/5 PASS"
        status: pass
      - kind: other
        ref: "task check:goreleaser — validates against pinned GoReleaser v2.17.1"
        status: pass
    human_judgment: false
  - id: D2
    description: "hooks.post.install's man-page assertion proves the current install produced fresh pages (before/after mtime+size snapshot), closing UF-5, and still passes on brew reinstall."
    requirement: "UPGR-02"
    verification:
      - kind: unit
        ref: "go test ./internal/upgrade/ -run '^TestHomebrewCask' — 5/5 PASS; go test ./internal/upgrade/... — ok"
        status: pass
      - kind: other
        ref: "rg -c 'fresh_man_pages'/'File\\.size' floors, node offset-ordering script (snapshot precedes generation) — all green"
        status: pass
    human_judgment: false
  - id: D3
    description: "release:rehearse-cask removes all sentinel/marker assertions and adds two independent find -newer freshness re-checks (Step 5 pre-install, Step 5d pre-reinstall), each with its own single-creation marker; evidence schema bumped to schema=3 on both pass and fail paths."
    requirement: "UPGR-02"
    verification:
      - kind: unit
        ref: "go test ./internal/upgrade/ -run '^TestTaskfile' — 5/5 PASS; go test ./internal/upgrade/ -run '^TestWorkflowRunBodiesInvokeTask$' — 1/1 PASS"
        status: pass
      - kind: other
        ref: "task --list-all exit 0; node offset-ordering script (each marker created before its own invocation, neither before the other's) — green; bash -n on extracted script — syntax OK"
        status: pass
    human_judgment: true
    rationale: "release:rehearse-cask itself requires CASK_REHEARSE=1 and installs/uninstalls a REAL cask into the maintainer's real Homebrew prefix — it is explicitly maintainer-only and was not run live in this session (no such opt-in was given). Static verification (syntax, gate counts, offset ordering, unit tests on the parsing/shape layer) is complete; a live run is the human-judgment gap."
  - id: D4
    description: "03-EVIDENCE.md's two false 'failed install can strand the sentinel' passages are corrected in place with checkable line citations and dated amendment notes; both folded todos are filed to completed/ with resolution notes."
    requirement: "UPGR-02"
    verification:
      - kind: other
        ref: "rg checks: 'leave the sentinel behind' = 0, '30 orphaned' >= 1, '577-587' >= 1, line-count delta within 40; per-name pending/completed membership checks for both todos — all green"
        status: pass
    human_judgment: false

duration: ~15min (git-commit-span; investigation/verification time not separately tracked)
completed: 2026-08-11
status: complete
---

# Phase 4 Plan 02: Remove Phase-3 marker file, close UF-5, correct 03-EVIDENCE.md Summary

**Removed the Phase-3 `.codegraph-brew-install` marker file from both cask hooks and all four rehearsal assertion sites now that Phase 4's brew-managed-install detection is structural (D-02), and closed the UF-5 man-page freshness defect in the same hook with a before/after mtime+size snapshot rather than a wall-clock or count-based check.**

## Performance

- **Duration:** ~15 min (git commit span 12:57:44Z-13:12:15Z; does not include file-reading/verification time)
- **Tasks:** 3/3 completed
- **Files modified:** 5 (`.goreleaser.yaml`, `Taskfile.yml`, `03-EVIDENCE.md`, 2 folded todos)

## Accomplishments

- Deleted the marker write from `hooks.post.install` and the marker removal from `hooks.post.uninstall`; BREW-05's two `raise` assertions (man-page count, version equality) survive byte-identical in behavior.
- Closed UF-5: `hooks.post.install` now snapshots `man_pages_before` (path -> {mtime, size}) immediately before `system_command binary, args: ["man", man_dir]`, and computes `fresh_man_pages` as the subset of the post-run glob that is newly present, mtime-advanced, or size-differing. Raises when empty, naming the man directory, before/total/fresh counts. Passes on `brew reinstall` (rewritten pages advance past their own baseline) — freshness, not growth, is the deliberate property.
- `release:rehearse-cask` (Taskfile.yml) drops Step 5b's sentinel assertions (existence, schema line, six-key loop) and the post-uninstall symmetry check's sentinel half, keeping the `OK=1` accumulator for Steps 5c/5d/6b/7. Adds two independently-created `mktemp` markers — `MAN_BASELINE_MARKER` (before the first install, consumed only by Step 5) and `MAN_REINSTALL_MARKER` (before the reinstall, consumed only by Step 5d) — each proven via `find -newer`. Evidence line bumped to `schema=3` on both the pass path and the fail-path trap, replacing the `sentinel=` field with `fresh_man_pages=` and resolving the pre-existing schema=1/schema=2 mismatch.
- Corrected both false passages in `.planning/phases/03-homebrew-tap-cask/03-EVIDENCE.md` claiming a failed install could strand the sentinel — the sentinel was always written strictly after both raises, so this was never possible; only man pages ever leaked. Dated amendment notes cite the exact line ordering (`.goreleaser.yaml:553-555`, `:566-568`, `:577-587` as of Phase 3) and note the correction is now moot (Phase 4 removed the sentinel outright).
- Folded both closed todos (`03-evidence-falsely-claims...`, `post-install-man-page-assertion-can-be-satisfied-by-stale-pages`) from `.planning/todos/pending/` to `.planning/todos/completed/` with resolution notes naming this plan.

## Task Commits

Each task was committed atomically:

1. **Task 1: Strip the marker file from the cask hooks and make the man-page assertion prove freshness** - `8ac1983` (refactor)
2. **Task 2: Remove the rehearsal's marker assertions and re-assert man-page freshness independently** - `83312ef` (refactor)
3. **Task 3: Correct 03-EVIDENCE.md's false stranding claim and retire the two folded todos** - `e8e988a` (docs), plus `ef14cf0` (docs) — a follow-up commit carrying content that an invalid `git add` pathspec had prevented from being staged in the first attempt (see Deviations)

**Plan metadata:** committed separately by the orchestrator (worktree mode — this plan does not update STATE.md/ROADMAP.md itself).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Task 1's freshness-oracle comment text accidentally re-introduced the literal strings `Time.now` and a second `FileUtils.rm_f` mention inside prose comments**
- **Found during:** Task 1 verification (`rg -c -e 'Time\.now'` returned 1, `rg -c -e 'FileUtils.rm_f'` returned 2, both violating the acceptance gate)
- **Issue:** the hook comment explaining *why* the wall-clock oracle was rejected quoted the literal old code (`Time.now - 1`), and the uninstall-hook comment explaining force-style removal used the literal method name `FileUtils.rm_f:` — both accidentally matched the gates meant to prove those mechanisms are gone
- **Fix:** reworded both comments to describe the mechanism in prose without repeating the literal removed identifiers
- **Files modified:** `.goreleaser.yaml`
- **Commit:** `8ac1983`

**2. [Rule 1 - Bug] Task 2's cleanup-trap placement of the two new markers broke the plan's own offset-ordering verification script**
- **Found during:** Task 2 verification (the node offset-ordering script threw "MAN_REINSTALL_MARKER created before the install")
- **Issue:** adding `"${MAN_BASELINE_MARKER:-}" "${MAN_REINSTALL_MARKER:-}"` to the `cleanup()` trap's `rm -rf` line put a literal textual mention of `MAN_REINSTALL_MARKER` before the install invocation, because the trap function is defined early in the script even though it only executes at exit — the verification script checks textual position, not runtime order
- **Fix:** removed both markers from the trap; each is instead cleaned up with an explicit `rm -f` immediately after its own consumer (Step 5's freshness check for `MAN_BASELINE_MARKER`, Step 5d's for `MAN_REINSTALL_MARKER`). Also removed a literal `MAN_REINSTALL_MARKER` mention from a comment attached to `MAN_BASELINE_MARKER`'s creation, which triggered the same false-early-match
- **Files modified:** `Taskfile.yml`
- **Commit:** `83312ef`

**3. [Rule 1 - Bug] The two literal-string acceptance gates (`rg -c -e 'leave the sentinel behind'` = 0) failed because the correction's own amendment notes quoted the false claim verbatim**
- **Found during:** Task 3 verification
- **Issue:** both amendment notes originally quoted the removed sentence in full (`"can leave the sentinel behind with no cask installed to explain it"`) to explain what was being corrected, which kept the exact banned phrase in the file
- **Fix:** reworded both amendments to describe the false claim in different words ("could strand the sentinel with no cask installed to explain its presence") rather than quoting it
- **Files modified:** `.planning/phases/03-homebrew-tap-cask/03-EVIDENCE.md`
- **Commit:** `e8e988a`

**4. [Rule 3 - Blocking issue] Task 3's first commit attempt silently staged only the two todo renames, not the file-content edits**
- **Found during:** post-commit self-check (`git status --short` showed the content edits still unstaged after the commit)
- **Issue:** a single `git add` invocation listed five paths, one of which (`.planning/todos/pending/...`) no longer existed after the preceding `git mv` calls. Git's pathspec validation is atomic across the whole invocation — the "did not match any files" fatal error aborted staging for ALL five paths, not just the invalid one — but `git mv` had already staged the two renames independently, so the commit "succeeded" with only those two changes and silently dropped the `03-EVIDENCE.md` correction and both resolution-note edits
- **Fix:** staged the three missed files explicitly and created a second commit (`ef14cf0`) carrying the content, rather than amending the first
- **Files modified:** none beyond what Task 3 already specified — same three files, split across two commits
- **Commit:** `ef14cf0`

### Acceptance-criteria deviation, documented rather than auto-fixed

**`OK=0` count in `Taskfile.yml` decreased from 29 (pre-task) to 26 (post-task), rather than staying unchanged-or-higher as the acceptance criteria requested.** Removing Step 5b's sentinel assertions (existence check, schema-line check, six-key loop — 3 textual `OK=0` sites) and the post-uninstall symmetry check's sentinel half (1 more) removed 4 sites; only 1 new `OK=0` site was added (Step 5d's `REINSTALL_FRESH_MAN_PAGE_COUNT` check). Step 5's own new freshness check (`FRESH_MAN_PAGE_COUNT`) intentionally uses `exit 1` rather than `OK=0`, matching the task's own action text ("Every new failure path uses `echo "::error::..."` followed by setting `OK=0` (inside the accumulating region) **or an explicit non-zero exit (outside it)**, matching the surrounding convention") — Step 5's checks are all pre-`OK=1`-declaration, exit-1-style, in the pre-existing file, and the new check simply follows that established pattern rather than the OK=0 pattern used from Step 5c onward. Padding the count back up with a contrived extra assertion was considered and rejected, per this same plan's own review-cycle-3 guidance against inventing references to preserve a number. This deviation is **not** part of the `<verify><automated>` gate (which does not check `OK=0` count) — only the prose `<acceptance_criteria>`. Net effect: fewer, more precise assertions replaced a larger number of assertions about a now-nonexistent file's internal structure; `TestTaskfile`/`TestWorkflowRunBodiesInvokeTask` (which pin this file's shape) both pass at their required counts, and `task --list-all` confirms the YAML still parses.

## Known Stubs

None.

## Threat Flags

None — this plan removes surface (the marker file mechanism) and closes an existing gap (UF-5); it introduces no new network endpoints, auth paths, or trust-boundary changes. `T-04-05`, `T-04-06`, `T-04-07`, `T-04-08` and `T-04-SC` (this plan's threat-register entries) are all addressed as designed; see `04-02-PLAN.md`'s `<threat_model>`.

## Self-Check: PASSED

- FOUND: `.goreleaser.yaml` (modified, present)
- FOUND: `Taskfile.yml` (modified, present)
- FOUND: `.planning/phases/03-homebrew-tap-cask/03-EVIDENCE.md` (modified, present)
- FOUND: `.planning/todos/completed/2026-08-10-03-evidence-falsely-claims-a-failed-install-can-strand-the-phase-4-sentinel.md`
- FOUND: `.planning/todos/completed/2026-08-10-post-install-man-page-assertion-can-be-satisfied-by-stale-pages.md`
- MISSING (expected — folded away): `.planning/todos/pending/2026-08-10-03-evidence-falsely-claims-a-failed-install-can-strand-the-phase-4-sentinel.md`, `.planning/todos/pending/2026-08-10-post-install-man-page-assertion-can-be-satisfied-by-stale-pages.md`
- FOUND commit `8ac1983` in `git log --oneline --all`
- FOUND commit `83312ef` in `git log --oneline --all`
- FOUND commit `e8e988a` in `git log --oneline --all`
- FOUND commit `ef14cf0` in `git log --oneline --all`
