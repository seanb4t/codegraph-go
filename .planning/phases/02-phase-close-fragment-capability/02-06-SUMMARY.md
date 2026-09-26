---
phase: 02-phase-close-fragment-capability
plan: 06
subsystem: infra
tags: [changie, gsd-core, capability, gh, bash, tdd]

# Dependency graph
requires:
  - phase: 02-phase-close-fragment-capability
    provides: "changie capability v0.1.0 installed at project and global scope (02-01..02-04); draft milestone PR #88 open (02-03)"
provides:
  - "codegraph-go .changie.yaml PR field optional (type: int, minInt: 1, optional: true), with RED-first Go guard and re-pointed check:changie legs 5/7/12"
  - "gsd-capability-changie v0.1.1: PR resolution is explicit --pr, else the open PR, else no PR with a note (never a skip); gh failures redacted; published private, tagged and proven from a clean HTTPS clone"
  - "codegraph-go's project install and the maintainer's global install both upgraded to v0.1.1 through gsd-core verbs"
affects: [02-05-contributing-docs, phase-03-close]

# Actuals (#2632)
actuals:
  tokens: 28209
  tasks: 3
  commits: 10
plan_head_before: 9a78ff01fafa0639db5ed0659d68d2f44e5498d9

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "GSD_HOME isolation in a capability's own scratch test harness: a machine with the capability already installed at global scope pollutes federated config-key validation for every scratch project unless GSD_HOME points at an empty directory for the test run"
    - "note() alongside skip(): a note (stdout, exit 0, run continues) is a third outcome class distinct from skip (stdout, exit 0, run stops) and error (stderr, non-zero exit) — used for every gh/PR failure mode that must not block the write"

key-files:
  created:
    - .planning/phases/02-phase-close-fragment-capability/02-MUTATION-LOG.md
  modified:
    - .changie.yaml
    - internal/upgrade/changie_shape_test.go
    - Taskfile.yml
    - .planning/REQUIREMENTS.md
    - .planning/ROADMAP.md
    - .planning/STATE.md
    - .planning/phases/02-phase-close-fragment-capability/02-CAPABILITY-LOG.md
    - "gsd-capability-changie:skills/changie-fragments/scripts/write-fragments.sh"
    - "gsd-capability-changie:skills/changie-fragments/SKILL.md"
    - "gsd-capability-changie:test/run.sh"
    - "gsd-capability-changie:test/fixtures/changie.yaml"
    - "gsd-capability-changie:test/fixtures/bin/gh"
    - "gsd-capability-changie:README.md"
    - "gsd-capability-changie:capability.json"

key-decisions:
  - "GSD_HOME isolated in gsd-capability-changie's test/run.sh (new, not in the plan): the maintainer's own global install of this capability (02-04, D-14) was leaking workflow.changie_command/workflow.changie_fragments into every scratch project's federated config schema on this machine, silently defeating the [ordering] leg's 'unknown key before install' guarantee. Fixed by pointing GSD_HOME at an empty scratch directory for the whole test run, restoring the pre-D-14 isolation without touching the machine's real global state (Rule 3 deviation)."
  - "Annotated tag v0.1.1's message is treated as the capability's release note, since the repository has no CHANGELOG file (D-09/D-10 precedent from 02-01/02-03)."

patterns-established:
  - "A capability's scratch-project test harness must isolate GSD_HOME, not just build a scratch project — the federated-config overlay reads global capability state regardless of how isolated the project itself is."

requirements-completed: [CAP-03, CAP-04, CHG-01, CHG-04]

coverage:
  - id: D1
    description: "codegraph-go's .changie.yaml PR field is optional (type: int, minInt: 1, optional: true); TestChangieConfigShape pins the exact custom key set; check:changie legs 5 (no-PR accepted), 7 (PR=0 still refused) and 12 (no-PR render) all pass; Phase 1's byte reproduction stays green"
    requirement: "CHG-01"
    verification:
      - kind: unit
        ref: "internal/upgrade/changie_shape_test.go#TestChangieConfigShape"
        status: pass
      - kind: integration
        ref: "task check:changie (12 of 12)"
        status: pass
    human_judgment: false
  - id: D2
    description: "CHG-04 amended: a fragment without PR is accepted, PR=0 is still refused by minInt: 1, and an undeclared kind is still refused — proven both ways"
    requirement: "CHG-04"
    verification:
      - kind: integration
        ref: "task check:changie legs 5, 6, 7, 8, 12"
        status: pass
    human_judgment: false
  - id: D3
    description: "gsd-capability-changie v0.1.1: PR resolution is explicit --pr, else the open PR, else a note (no skip) with no PR number; gh-missing/pr-lookup-failed/no-open-pr/pr-lookup-invalid are all notes; gh stderr tokens are redacted; a required-PR host still refuses cleanly"
    requirement: "CAP-03"
    verification:
      - kind: integration
        ref: "test/run.sh (32 of 32), including [note:gh-missing], [note:pr-lookup-failed], [note:no-open-pr], [note:pr-lookup-invalid], [no-pr-required-host]"
        status: pass
    human_judgment: false
  - id: D4
    description: "Three mutation families (PR optional removed, changeFormat conditional removed, minInt removed) each go RED on the exact predicted leg in a disposable scratch worktree, then back to GREEN, with the main tree and worktree listing left clean"
    requirement: "CHG-01"
    verification:
      - kind: other
        ref: ".planning/phases/02-phase-close-fragment-capability/02-MUTATION-LOG.md Family (a)/(b)/(c)"
        status: pass
    human_judgment: false
  - id: D5
    description: "v0.1.1 published as main's annotated tip on the private repository, v0.1.0 unchanged, proven from a clean HTTPS clone (32 of 32) and from a fresh scratch install; both codegraph-go's project install and the maintainer's global install upgraded to v0.1.1 through gsd-core verbs with capability.json byte-identity, an unchanged skill set, and render-hooks still showing one changie step"
    requirement: "CAP-04"
    verification:
      - kind: other
        ref: "git ls-remote + clean clone test/run.sh + scratch capability install + project/global upgrade proofs (02-CAPABILITY-LOG.md Task 3)"
        status: pass
    human_judgment: false
  - id: D6
    description: "CHG-01/CHG-04 in REQUIREMENTS.md and Phase 1 SC2/SC4 plus the first Phase 2 Notes sentence in ROADMAP.md amended in place to describe the optional PR field, with no heading added"
    requirement: "CHG-01"
    verification:
      - kind: other
        ref: "rg checks against .planning/REQUIREMENTS.md and .planning/ROADMAP.md, plus a diff-heading check on the commit"
        status: pass
    human_judgment: false

# Metrics
duration: ~70min
completed: 2026-09-26
status: complete
---

# Phase 2 Plan 6: PR-Optional Upgrade (D-13) Summary

**Made the changie `PR` custom field optional end to end — codegraph-go's `.changie.yaml` and its RED-proven guards, the capability's gh-failure-as-note rewrite with stderr redaction, a published-and-proven `v0.1.1`, and both the project and global installs upgraded through gsd-core verbs.**

## Performance

- **Duration:** ~70 min (first commit `2026-09-26T10:38:40-04:00`, last commit `2026-09-26T11:48:17-04:00`)
- **Tasks:** 3/3 completed (Task 1 tracer, Task 2 expansion, Task 3 publish/upgrade)
- **Files modified:** 8 in codegraph-go, 7 in `gsd-capability-changie` (new file: `02-MUTATION-LOG.md`)
- **Commits:** 10 in codegraph-go (measured: `git rev-list --count 9a78ff01..HEAD`), 5 in `gsd-capability-changie` (`v0.1.0..v0.1.1`)

## Accomplishments

- **Task 1 (tracer).** `.changie.yaml`'s `PR` custom field gained `optional: true` (RED-first: the Go test's `custom[0]` key-set assertion was committed and shown failing before the config changed). `check:changie` leg 5 flipped from "missing PR refused" to "no PR accepted", with legs 6-9's fragment-count expectations rebased. In the capability, `write-fragments.sh` gained a `note()` helper alongside `skip()`: a no-open-PR result is now a note (run continues, writes without a PR field) instead of a skip. The write loop's changie argv moved into an array so `-m PR=<n>` is appended only when a PR exists.
- **Task 2 (expansion).** Every remaining gh failure mode (`gh-missing`, `pr-lookup-failed`, `pr-lookup-invalid`) became a note with the same redaction contract (`gh[pousr]_.../github_pat_...` → `[REDACTED]`, gh stdout never echoed). A new `[no-pr-required-host]` leg proves a host whose `.changie.yaml` still requires `PR` refuses a no-PR write as `error: changie-failed`, leaving nothing written. `check:changie` gained leg 12 (`batch auto --dry-run` renders a no-PR line and a PR-linked line, writes nothing). Three mutation families — PR-optional removed, `changeFormat`'s conditional removed, `minInt` removed — each fired RED on the exact predicted leg (`[5/12]`, `[12/12]`, `[7/12]`) and returned to GREEN in a disposable, detached scratch worktree.
- **Task 3 (publish and upgrade).** `CHG-01`/`CHG-04` and the matching ROADMAP text were amended in place (value edits only, no new heading). `gsd-capability-changie`'s README/SKILL.md/`capability.json` were rewritten for v0.1.1 (HTTPS-only install spec, a new `## PR number` section, a `## Notes` table replacing the three retired skip rows, a `## Upgrading` section). The annotated tag `v0.1.1` was created on `main`'s tip and pushed fast-forward (`main` then the tag), proven from a clean HTTPS clone (32 of 32) and a fresh scratch install; `v0.1.0` is untouched and the repository is still `PRIVATE`. Both codegraph-go's project install and the maintainer's global install were upgraded to `v0.1.1` through `gsd-tools capability install`/`capability set` — never a hand copy — with `capability.json` byte-identity, an unchanged skill set (`comm -3` empty, content snapshot byte-equal), and `render-hooks verify:post` still showing exactly one `changie` step.

## Task Commits

### codegraph-go (10 commits, measured `9a78ff01..HEAD`)

1. `test(02-06): expect the changie PR field to be optional (D-13)` — `4093ccbf`
2. `feat(02-06): make the changie PR field optional and re-point the missing-PR leg (CHG-01, CHG-04, D-13)` — `67fbda9b`
3. `docs(02-06): record the no-PR tracer evidence` — `bb8ea638`
4. `feat(02-06): prove no-PR fragments render without a link in check:changie (CHG-04, D-13)` — `8c4d3f7a`
5. `docs(02-06): record PR-optional mutation evidence` — `6a142fa8`
6. `docs(02-06): record capability note-leg evidence` — `19ce1468`
7. `docs(02-06): fix GREEN re-run wording to carry the literal check:changie prefix` — `3ef9103c`
8. `docs(02-06): amend CHG-01 and CHG-04 for the optional PR field (D-13)` — `99b2923d`
9. `docs(02-06): record the capability tag-at-main-tip install constraint` — `754c0e79`
10. `docs(02-06): record v0.1.1 publish and upgrade evidence` — `0c27260d`

### `gsd-capability-changie` (5 commits, `v0.1.0..v0.1.1`)

1. `test(02-06): add failing no-open-PR note and no-PR write proofs` — `57dcbc4`
2. `feat(02-06): write fragments without a PR number when no open PR is found (D-13)` — `5e1b9d0`
3. `test(02-06): add failing gh-failure note, redaction, invalid-lookup and required-PR-host proofs` — `91c3754`
4. `feat(02-06): turn gh failures into notes with redacted stderr and validate the looked-up PR (D-13)` — `305c2c4`
5. `docs(02-06): document the optional PR, HTTPS install and notes for v0.1.1` — `35b674e`

**Plan metadata:** this SUMMARY's own commit (below).

## TDD Gate Compliance

Both repositories carry `test(02-06)` → `feat(02-06)` pairs, in order, for both TDD cycles in this plan (`rule x1cjy9vyhq`):

- codegraph-go: `test(02-06)` (`4093ccbf`) precedes `feat(02-06)` (`67fbda9b`).
- `gsd-capability-changie`: `test(02-06)` → `feat(02-06)` → `test(02-06)` → `feat(02-06)`, i.e. `57dcbc4` → `5e1b9d0` → `91c3754` → `305c2c4`.

**RED transcript (Task 1, Go), pasted verbatim from the commit that landed before `.changie.yaml` changed:**

```
--- FAIL: TestChangieConfigShape (0.00s)
changie_shape_test.go:260: ../../.changie.yaml: custom[0] key set = [key minInt type], want exactly [key minInt optional type] (an absent optional key, or any other key, must fail — 02-CONTEXT D-13, which supersedes D-11)
--- FAIL: TestChangieConfigShape (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/upgrade	0.186s
FAIL
```

This is a genuine assertion failure on planned behavior (the real, pre-change `.changie.yaml` has no `optional` key yet), not a build error. Per the test file's own header comment, this guard is never gated on `gsd-tools check tdd-red-evidence` (it parses TAP only and always reports `INVALID_RED` for `go test`); the RED evidence is this commit's position on `main..HEAD` plus this pasted `--- FAIL` transcript, exactly as the plan's `<output>` spec requires.

Full RED/GREEN transcripts for both Task 1 and Task 2's TDD cycles (Go and shell) are in `02-MUTATION-LOG.md`'s "RED-first transcripts" section and `02-CAPABILITY-LOG.md`'s "02-06 Task 1"/"02-06 Task 2" sections.

## Evidence Pointers

- `.planning/phases/02-phase-close-fragment-capability/02-MUTATION-LOG.md`:
  - "RED-first transcripts (Task 1)" — the Go RED, the `check:changie` leg-5 RED against the pre-change config, and the GREEN dry-run confirmation.
  - "Family (a)" / "Family (b)" / "Family (c)" — the three mutation families, each with pre-mutation gate, applied confirmation, both RED transcripts, byte-clean revert, and GREEN re-run. Ends "3 of 3 mutation families went RED on the named guard and back to GREEN."
- `.planning/phases/02-phase-close-fragment-capability/02-CAPABILITY-LOG.md`:
  - "02-06 Task 1 — no-PR tracer" — `$CAP` commit trail, RED (legs 1-15 pass, fails at `[note:no-open-pr]`), GREEN (30 of 30), fragment content and dry-run render, and the GSD_HOME environmental finding/fix (see Deviations).
  - "02-06 Task 2 — note legs and required-PR host" — RED (fails at `[note:gh-missing]`), GREEN (32 of 32), full commit trail.
  - "02-06 Task 3 — v0.1.1 publish and upgrade" — tag/push transcript, ls-remote proof, clean-clone 32-of-32, both scratch-install transcripts, and both project/global upgrade before/after records.

## Upgrade

The exact commands that upgraded both installs (02-05 copies these into `CONTRIBUTING.md`):

```bash
# Project scope (per clone)
gsd-tools capability install "https://github.com/seanb4t/gsd-capability-changie.git#v0.1.1" --scope project

# Global scope (per machine, for /gsd-changie-fragments to dispatch in Claude Code)
gsd-tools capability install "https://github.com/seanb4t/gsd-capability-changie.git#v0.1.1" --scope global
gsd-tools capability set changie --enable --runtime claude --scope global
```

`gsd-tools capability update <id>` is **not** the upgrade path — it re-fetches the spec already recorded in the ledger (`#v0.1.0`), so it cannot move to a new tag on its own (confirmed by reading `capability-lifecycle.cjs:366-381` and by the live `#v0.1.0` scratch-install failure below). Re-running `capability install <new-spec>` over an existing entry is gsd-core's upgrade swap (`isUpgradeLike`, backup, ledger rewrite).

Both upgrades were verified byte-identical to the tag (`capability.json` `cmp`, materialized `SKILL.md` `cmp`), left the rest of the global skills directory untouched (`comm -3` empty, content-snapshot `diff` empty), and left `render-hooks verify:post`'s changie-step count at 1 in codegraph-go.

## Upstream gap

gsd-core 1.14.0 fetches a git capability source with `git clone --depth 1` of the default branch, then `git checkout <ref>`. A depth-1 clone of `main` only carries the tags reachable from `main`'s current tip — so once `main` moved to publish `v0.1.1`, the `#v0.1.0` spec stopped resolving:

```
$ gsd-tools capability install "https://github.com/seanb4t/gsd-capability-changie.git#v0.1.0" --scope project --raw
Error: capability install blocked: git checkout "v0.1.0" failed (exit 128): fatal: invalid reference: v0.1.0
```

This exactly matched the `<interfaces>` prediction recorded before this plan ran. Consequence: **every future capability release must move `main` to its new tag before publishing**, and any doc naming an install spec (`CONTRIBUTING.md`, `CAP-04`, `SHIP-04`) must always name the newest tag. This is recorded as an upstream gap in `resolveGit` via `gsd-tools state add-blocker` (see STATE.md) rather than filed against `gsd-core` — filing it needs the maintainer's separate authorization.

## Interpretation: the annotated tag is the release note

`gsd-capability-changie` has no `CHANGELOG.md` or fragment-tracking file (D-09/D-10: its own commit trail lives outside this roadmap and its proofs are `test/run.sh`, not changie fragments). Per this plan's own instruction, the annotated tag `v0.1.1`'s message — created with the file-writing tool, listing the D-13 changes — **is** the capability's release note. There is no other document this repository could put a release note in.

## Files Created/Modified

**codegraph-go:**
- `.changie.yaml` — `PR` custom field gains `optional: true`; header comment rewritten for D-13.
- `internal/upgrade/changie_shape_test.go` — `TestChangieConfigShape`'s `custom[0]` key-set assertion expects `optional: true`; file header cites D-13.
- `Taskfile.yml` — `check:changie` leg 5 re-pointed, legs 6-9 rebased, leg 12 added; desc and pass-count updated to 12.
- `.planning/REQUIREMENTS.md`, `.planning/ROADMAP.md` — CHG-01/CHG-04 and Phase 1 SC2/SC4/Notes amended in place.
- `.planning/STATE.md` — the depth-1-clone install-constraint blocker recorded via `state add-blocker`.
- `.planning/phases/02-phase-close-fragment-capability/02-MUTATION-LOG.md` (new) — RED-first transcripts and the three mutation families.
- `.planning/phases/02-phase-close-fragment-capability/02-CAPABILITY-LOG.md` — three new `## 02-06 Task N` sections.

**`gsd-capability-changie`:**
- `skills/changie-fragments/scripts/write-fragments.sh` — `note()` helper; gh-missing/pr-lookup-failed/no-open-pr/pr-lookup-invalid are all notes with redaction and validation; write-loop argv built as an array.
- `skills/changie-fragments/SKILL.md` — step 3 distinguishes note from skip; PR rule restated; `metadata.version` → `0.1.1`.
- `test/run.sh` — `[note:no-open-pr]`, `[note:gh-missing]`, `[note:pr-lookup-failed]`, `[note:pr-lookup-invalid]`, `[write-no-pr]`, `[no-pr-required-host]` added/replaced; `[pr-override]` gained a no-note assertion; `EXPECTED_LEGS` 29 → 32; GSD_HOME isolated (see Deviations).
- `test/fixtures/changie.yaml` — refreshed byte copy of the host `.changie.yaml`.
- `test/fixtures/bin/gh` — new `noauth-token` mode.
- `README.md`, `capability.json` — v0.1.1 documentation and version bump.

## Decisions Made

- **GSD_HOME isolation in `test/run.sh` (Rule 3 deviation, not in the plan).** On this machine, 02-04's global-scope install of this very capability (D-14) leaks `workflow.changie_command`/`workflow.changie_fragments` into every scratch project's federated config schema, because `capability-loader.cjs` overlays both the global and project capability roots when resolving a project's config schema. This broke the `[ordering]` leg's "unknown key before install" guarantee, discovered on this plan's first re-run of `test/run.sh` since the global install happened. Fixed by exporting an isolated, empty `GSD_HOME` for the whole scratch run — no real global state touched, and `[ordering]` passes again exactly as it did before 02-04. Recorded in full in `02-CAPABILITY-LOG.md`'s Task 1 section.
- **The annotated tag message is the capability's release note** (see "Interpretation" above) — there is no CHANGELOG in `gsd-capability-changie` to write one into.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] GSD_HOME isolation added to `gsd-capability-changie:test/run.sh`**
- **Found during:** Task 1, step 6 (first run of `test/run.sh` since 02-04's global install)
- **Issue:** the `[ordering]` leg failed unconditionally — `workflow.changie_command` was accepted by `config-set` even in a scratch project that had never installed the capability, because the maintainer's own machine already has it installed at global scope (02-04, D-14) and gsd-core's federated-config overlay reads the global capability root regardless of project isolation.
- **Fix:** exported `GSD_HOME="${scratch}/isolated-home"` (empty, created fresh) at the top of `test/run.sh`, before the fixture project is built.
- **Files modified:** `gsd-capability-changie:test/run.sh` (7 lines).
- **Verification:** re-ran the suite; `[ordering]` passed, and the run proceeded past it to the genuine RED point for Task 1 (`[note:no-open-pr]`).
- **Committed in:** `57dcbc4` (part of the Task 1 RED commit).

---

**Total deviations:** 1 auto-fixed (1 blocking, Rule 3).
**Impact on plan:** Necessary to run the capability's own test suite at all on this machine going forward (any machine with the capability installed globally would hit the same leak). No scope creep — the fix touches only test infrastructure, not the D-13 behavior itself.

## Known Acceptance-Criterion Environmental Mismatch

Task 2's acceptance criterion `test "$(printf '%s\n' "$w" | wc -l | tr -d ' ')" = 1` (worktree list has exactly one entry after mutation teardown) **fails literally** on this machine: `git worktree list` shows **two** entries both before and after the mutation run — the main checkout, plus `worktree-brave-river-a240`, a pre-existing, unrelated worktree from a concurrent session (`/Volumes/Code/herdr-worktrees/codegraph-go/worktree-brave-river-a240`, branch `chore/octopusignore`). This worktree was **not created or touched by this plan** — it existed before Task 2's mutation setup and remains identical after teardown.

The substantively meaningful invariant this criterion protects — **no worktree was left behind by this plan's mutation testing** — was verified directly: the `git worktree list` output is byte-for-byte identical before and after (2 entries, same paths, same commits), and `git status --porcelain -- .changie.yaml .changes CHANGELOG.md` in the main tree is empty. Deleting `worktree-brave-river-a240` to force the literal count to 1 would destroy another session's active work and was not done. This is recorded here rather than papered over; see `02-MUTATION-LOG.md`'s "Teardown (Task 2)" section for the full before/after transcripts.

## Issues Encountered

None beyond the GSD_HOME finding above (handled as a Rule 3 deviation) and the acceptance-criterion mismatch above (documented, not a defect in the mutation evidence itself).

## Threat Flags

None — every threat register item this plan touches (T-02-01, T-02-22, T-02-23, T-02-24, T-02-25, T-02-SC, T-02-16, T-02-26, T-02-27) was already anticipated in the plan's own threat model and verified directly: argv-literal safety re-proven (`[argv-literal]`), gh token redaction proven with the new `noauth-token` stub mode, invalid-lookup validation proven with `PR=0`/`12abc`, the required-PR-host refusal proven with `[no-pr-required-host]`, the tag/push proven fast-forward-only with `v0.1.0` unchanged, and the global surface pass proven to touch nothing but the one materialized `SKILL.md`.

## User Setup Required

None — no external service configuration required beyond what 02-04 already performed. 02-05 (CONTRIBUTING.md) will copy this plan's exact upgrade commands.

## Next Phase Readiness

D-13's four parts are complete: codegraph-go's config and guards, capability `v0.1.1`, publish, and both installs upgraded. `02-05` (CONTRIBUTING docs) is next — it should reference this plan's `## Upgrade` section verbatim. Phase 3's close remains where `CAP-05` (the capability's first real firing) is proven; that fragment will resolve `pr 88` from the still-open draft milestone PR, or write without a PR number if that PR has since merged or closed.

---
*Phase: 02-phase-close-fragment-capability*
*Completed: 2026-09-26*
