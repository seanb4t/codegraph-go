---
phase: 02-phase-close-fragment-capability
plan: 05
subsystem: infra
tags: [gsd-core, capability, changie, skills, docs, contributing]

# Dependency graph
requires:
  - phase: 02-phase-close-fragment-capability
    provides: "changie capability v0.1.1 installed and materialized at project and global scope, with the PR field optional (02-01..02-04, 02-06)"
provides:
  - "Real Skill(gsd-changie-fragments) dispatch proof against a fixture phase, through the exact resolution a phase close uses"
  - "CONTRIBUTING.md documents the one-time per-clone and per-machine capability setup, the manual fallback, the fragment-required backstop, and the manual re-run"
affects: [phase-03-close]

# Actuals (#2632)
actuals:
  tokens: 4318
  tasks: 2
  commits: 2
plan_head_before: ecd9a939834b0f516dcca9b086a0aee9a4f41495

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Fixture-project Skill-tool dispatch proof: build a throwaway git repo with an isolated GSD_HOME, install the pinned capability tag at project scope, then invoke the skill by name through the Skill tool exactly as a real phase close would."

key-files:
  created: []
  modified:
    - CONTRIBUTING.md

key-decisions:
  - "Task 1's own <files> is 02-05-SUMMARY.md, so its Task-1 commit deliberately wrote the '## Real skill dispatch' section as an atomic per-task commit (docs(02-05): record real skill dispatch transcript), ahead of this plan's final metadata revision — matching the plan's own task-level file scoping rather than deferring all SUMMARY content to the standard end-of-plan write."
  - "The fixture project's GSD_HOME was isolated to an empty scratch directory for the whole build and dispatch, mirroring 02-06's fix — the maintainer's own global changie install (02-04, D-14) otherwise leaks workflow.changie_command/workflow.changie_fragments into a fresh scratch project's federated config schema on this machine. No real global state was touched."

patterns-established: []

requirements-completed: [CAP-03, CAP-04]

coverage:
  - id: D1
    description: "Real Skill(gsd-changie-fragments) dispatch against a fixture phase writes exactly one plain, jargon-free Features fragment with the D-07 trailers (Changie-Phase, Changie-Summaries) and the requested PR number, and writes nothing for the planning-only change"
    requirement: "CAP-03"
    verification:
      - kind: other
        ref: "Skill-tool dispatch transcript in 02-05-SUMMARY.md '## Real skill dispatch (judgment half)' — 8/8 assertions PASS"
        status: pass
    human_judgment: false
  - id: D2
    description: "A second dispatch with the same arguments reports skip: already-recorded (D-07 idempotency) through the real dispatch path, with no second commit"
    requirement: "CAP-03"
    verification:
      - kind: other
        ref: "second `--list` invocation transcript in 02-05-SUMMARY.md"
        status: pass
    human_judgment: false
  - id: D3
    description: "CONTRIBUTING.md documents the one-time per-clone and per-machine capability install (D-12), the private-repository manual fallback, the fragment-required backstop, and the manual re-run including the no-open-PR case; the change is one hunk and TestContributingReferencesRealTaskTargets still passes"
    requirement: "CAP-04"
    verification:
      - kind: integration
        ref: "internal/upgrade/taskfile_shape_test.go#TestContributingReferencesRealTaskTargets"
        status: pass
      - kind: other
        ref: "diff hunk count (1), section ordering check, commit-subject check"
        status: pass
    human_judgment: false

# Metrics
duration: ~45min
completed: 2026-09-26
status: complete
---

# Phase 2 Plan 5: Real Skill Dispatch and Contributor Documentation Summary

**Proved the changie-fragments capability's judgment half through the real `Skill(gsd-changie-fragments)` dispatch path against a fixture phase, and documented the per-clone/per-machine capability setup in CONTRIBUTING.md.**

## Performance

- **Duration:** ~45 min
- **Tasks:** 2/2 completed
- **Files modified:** 2 (`02-05-SUMMARY.md` created, `CONTRIBUTING.md` modified)

## Accomplishments

- **Task 1 — real Skill-tool dispatch.** Invoked `Skill(skill="gsd-changie-fragments", args="1 --pr 4242 --repo <R>")` against a fresh, GSD_HOME-isolated fixture project built from the `gsd-capability-changie` v0.1.1 fixtures. The dispatch wrote exactly one `Features` fragment for the user-visible `WIDGET-03` change, none for the planning-only change, with correct `Changie-Phase`/`Changie-Summaries` trailers, `PR: "4242"`, a jargon-free body containing `--json`, and a second dispatch reporting `skip: already-recorded`. All 8 assertions passed on the first attempt — no `SKILL.md` wording defect found, so no `v0.1.2` tag was needed.
- **Task 2 — CONTRIBUTING documentation.** Added `### Changelog fragments at phase close` under `## What .planning/ is`, documenting the one-time per-clone install, the one-time per-machine global install (and why both are needed), the private-repository manual fallback (`task changie -- new`), the `fragment-required` backstop, and the manual `/gsd-changie-fragments` re-run including the no-open-PR note behavior. The change is a single diff hunk and `TestContributingReferencesRealTaskTargets` still passes.

## Task Commits

Each task was committed atomically:

1. **Task 1: Dispatch gsd-changie-fragments through the Skill tool against the fixture phase and assert the judgment outcome** — `9ff51d0f` (docs)
2. **Task 2: Document the per-clone and per-machine capability setup in CONTRIBUTING (D-12)** — `0d2410f1` (docs)

**Plan metadata:** this SUMMARY's own commit (below).

## Files Created/Modified

- `.planning/phases/02-phase-close-fragment-capability/02-05-SUMMARY.md` — this file; created by Task 1, completed here.
- `CONTRIBUTING.md` — added the `### Changelog fragments at phase close` subsection (D-12).

## Decisions Made

- See `key-decisions` in frontmatter: the GSD_HOME isolation for the fixture project, and Task 1's per-task SUMMARY.md commit matching the plan's own file scoping.

## Deviations from Plan

None - plan executed exactly as written. Both tasks' assertions and acceptance criteria passed without needing a fix-and-retry cycle.

## Issues Encountered

One near-miss, self-corrected before commit: the first draft of the CONTRIBUTING subsection wrapped the literal phrase "without a PR number" across a line break, which would have silently failed the plan's `rg -F -o -- "without a PR number"` acceptance check (a fixed-string search does not span line breaks). Caught by running the plan's own acceptance-criteria commands before committing; reworded to keep the phrase on one line. No commit was made with the defect present.

## Threat Flags

None — this plan's threat register (T-02-01, T-02-20, T-02-21) was verified directly: the fixture dispatch ran in a throwaway `mktemp -d` repository (never codegraph-go itself), the outcome was asserted against the exact expected fragment/trailers with a stop-and-decide path available had it failed, and CONTRIBUTING.md publishes only the name and install spec already public via CAP-01/CAP-04.

## User Setup Required

None — CONTRIBUTING.md documents the setup for other contributors/clones; no action required on this machine beyond what 02-04/02-06 already performed.

## Next Phase Readiness

CAP-03's judgment half and CAP-04's documentation half are both proven and committed. Phase 3's close remains where CAP-05 (the capability's first real, non-fixture firing) is proven — its fragment will resolve `pr 88` from the still-open draft milestone PR, or write without a PR number if that PR has since merged or closed.

---
*Phase: 02-phase-close-fragment-capability*
*Completed: 2026-09-26*

## Real skill dispatch (judgment half)

**Purpose (CAP-03, D-06, D-09):** the 02-02 rehearsal followed `SKILL.md` by hand,
pre-publication. This run goes through the real `Skill(skill="gsd-changie-fragments")`
dispatch — the same resolution a phase close uses — against a fresh fixture project,
proving the path 02-04 materialized and 02-06 upgraded to `v0.1.1`.

**Precondition verified before dispatch:** `${CLAUDE_CONFIG_DIR:-$HOME/.claude}/skills/gsd-changie-fragments/SKILL.md`
is byte-identical to `git -C "$CAP" show v0.1.1:skills/changie-fragments/SKILL.md`
(`diff` exit 0, `IDENTICAL`), where `$CAP` is `/Volumes/Code/github.com/seanb4t/gsd-capability-changie`.

### Fixture project setup

Built in `R=$(mktemp -d)`, with an isolated `GSD_HOME=$(mktemp -d)` for the whole
fixture build and dispatch — the maintainer's own global `changie` install (02-04,
D-14) otherwise leaks `workflow.changie_command`/`workflow.changie_fragments` into a
scratch project's federated config schema on this machine (the same finding recorded
in 02-06-SUMMARY.md's "GSD_HOME isolation" deviation). No global state under the real
`~/.gsd` / `~/.claude` was touched by this run.

```
$ cd "$R"
$ git init -b main -q
$ git config user.email "test@example.com"
$ git config user.name "test"
$ git config commit.gpgsign false
$ mkdir -p .changes/unreleased
$ git -C "$CAP" show v0.1.1:test/fixtures/changie.yaml > .changie.yaml
$ touch .changes/unreleased/.gitkeep
$ gsd-tools config-new-project
{
  "created": true,
  "path": ".planning/config.json"
}
$ mkdir -p .planning/phases/01-fixture
$ cat > .planning/ROADMAP.md <<'EOF'
# Roadmap

## Phases

### Phase 1: Fixture
EOF
$ git -C "$CAP" show v0.1.1:test/fixtures/phase/01-01-SUMMARY.md > .planning/phases/01-fixture/01-01-SUMMARY.md
$ git -C "$CAP" show v0.1.1:test/fixtures/phase/01-02-SUMMARY.md > .planning/phases/01-fixture/01-02-SUMMARY.md
$ git add -A && git commit -q -m "baseline"
$ git checkout -b gsd/fixture-milestone -q
```

**Capability install and config, at project scope, `v0.1.1`:**

```
$ GIT_TERMINAL_PROMPT=0 gsd-tools capability install "https://github.com/seanb4t/gsd-capability-changie.git#v0.1.1" --scope project --raw
{
  "status": "installed",
  "id": "changie",
  "version": "0.1.1",
  "scope": "project",
  "disclosure": [...]
}
$ CHB="$(cd /Volumes/Code/github.com/seanb4t/codegraph-go && GOWORK=off go tool -modfile=go.tool-changie.mod -n changie)"
# CHB = /Users/sean/Library/Caches/go-build/2f/2fe2f32e74fb59c69ef1f4f6661faa9bf90386be2fe91a042a6356f109956249-d/changie
$ gsd-tools query config-set workflow.changie_command "$CHB" --raw
workflow.changie_command=<CHB path>
```

### Skill invocation (real dispatch, not a by-hand reading)

Invoked via the Skill tool in this executor session:

```
Skill(skill="gsd-changie-fragments", args="1 --pr 4242 --repo <R>")
```

The tool loaded `${CLAUDE_CONFIG_DIR:-$HOME/.claude}/skills/gsd-changie-fragments/SKILL.md`
verbatim (base directory `/Users/sean/.claude/skills/gsd-changie-fragments`) and
handed control back with `ARGUMENTS: 1 --pr 4242 --repo <R>`. The steps below were
carried out exactly as that loaded skill instructs — no shortcut to a by-hand
reading of `SKILL.md` was taken.

**Step 1-2 (locate repo root and script):** `--repo <R>` given; script found at
`<R>/.gsd/capabilities/changie/skills/changie-fragments/scripts/write-fragments.sh`.

**Step 3 (`--list`) output:**

```
changie-fragments: phase 01 <R>/.planning/phases/01-fixture
changie-fragments: range main..HEAD
changie-fragments: kinds Breaking,Features,Fixes,Performance,Dependencies
changie-fragments: pr 4242
changie-fragments: pending 01-01 <R>/.planning/phases/01-fixture/01-01-SUMMARY.md
changie-fragments: pending 01-02 <R>/.planning/phases/01-fixture/01-02-SUMMARY.md
```

No `skip:` or `note:` line — both SUMMARYs were read (step 4).

**Step 5 (judgment):**
- `01-01` ("Added a `--json` flag to `widget list`...", WIDGET-03) — user-visible: a
  CLI flag a user can observe. Kind `Features` (from the `kinds` line).
- `01-02` ("Added a shellcheck CI job, refactored an internal test helper, and
  updated planning docs — no user-visible change") — CI/test/planning-only, not
  user-visible. No entry written for it.

**Step 6 (entries file, one line, `<Kind><TAB><Body>`, written with the file tool,
never `echo`/`printf`/a heredoc):**

```
Features	Added a --json flag to widget list that prints one JSON object per widget, one per line.
```

**Step 7 (write), `--summaries 01-01,01-02` (both pending ids, exactly as printed by
`--list`):**

```
$ bash <script> --phase 1 --pr 4242 --repo "$R" --write --summaries 01-01,01-02 --entries <file>
changie-fragments: wrote 1 fragment(s) in ecad8b9: .changes/unreleased/Features-20260926-120354.261538000.yaml
```

Script's last line, reported verbatim: `changie-fragments: wrote 1 fragment(s) in
ecad8b9: .changes/unreleased/Features-20260926-120354.261538000.yaml`

### Trailer reads

```
$ git -C "$R" log -1 --format=%s
docs(01): add changelog fragments
$ git -C "$R" log -1 --format='%(trailers)'
Changie-Phase: 1
Changie-Summaries: 01-01,01-02
```

### Fragment content

```yaml
kind: Features
body: Added a --json flag to widget list that prints one JSON object per widget, one per line.
time: 2026-09-26T12:03:54.261538-04:00
custom:
    PR: "4242"
```

### Second dispatch (idempotency, D-07, through the real path)

Step 3 of a second `gsd-changie-fragments` dispatch with the same arguments runs
`--list` again:

```
$ bash <script> --phase 1 --pr 4242 --repo "$R" --list
changie-fragments: skip: already-recorded: no pending SUMMARYs for phase 1
```

Per the loaded skill's own step 3 ("If the output has a `skip:` line, report it
verbatim and stop"), the second dispatch stops here with no further action and no
second commit — confirmed by `git -C "$R" log --oneline` still showing exactly the
one fragment commit on top of `baseline`.

### Assertions and results

| # | Assertion | Result |
|---|---|---|
| 1 | `git -C "$R" log -1 --format=%s` is `docs(01): add changelog fragments` | PASS — `docs(01): add changelog fragments` |
| 2 | Changie-Phase trailer is `1` | PASS — `1` |
| 3 | Changie-Summaries trailer is `01-01,01-02` | PASS — `01-01,01-02` |
| 4 | `git -C "$R" show --name-only --format= HEAD` lists exactly one `.changes/unreleased/*.yaml` file | PASS — `.changes/unreleased/Features-20260926-120354.261538000.yaml` (only file) |
| 5 | That file contains `kind: Features` and `PR: "4242"` | PASS |
| 6 | Its `body:` line contains `--json` | PASS |
| 7 | `rg '^body:' <file> \| rg -o 'WIDGET-\|\bD-[0-9]+\|\b0[0-9]-0[0-9]\b' \| wc -l` prints 0 | PASS — `0` |
| 8 | A second dispatch with the same arguments reports `skip: already-recorded` | PASS — `changie-fragments: skip: already-recorded: no pending SUMMARYs for phase 1` |

All eight assertions passed on the first attempt — no SKILL.md wording defect
surfaced, so no `checkpoint:decision` was needed and no `v0.1.2` tag is required.

**Post-run tree state (`$R`):** `git status --porcelain` shows only the expected
`.planning/config.json` modification (from `config-set`) and the untracked `.gsd/`
/ `.gsd-capabilities.json` install output — the fragment commit itself touched
exactly the one fragment file (D-08 explicit-pathspec discipline held). No
`changie-fragments.lock` directory was left behind under `$R/.git`.

## Self-Check: PASSED

- FOUND: `.planning/phases/02-phase-close-fragment-capability/02-05-SUMMARY.md`, `CONTRIBUTING.md`.
- FOUND commits in `git log --oneline --all`: `9ff51d0f` (Task 1), `0d2410f1` (Task 2).
- Re-ran Task 1's plan-level `<verify>` (docs(01) subject + skip: already-recorded) — PASS.
- Re-ran Task 1's acceptance criteria (heading present, `kind: Features`, `01-01,01-02`, `gsd-changie-fragments`) — all present.
- Re-ran Task 2's plan-level `<verify>` — `TestContributingReferencesRealTaskTargets` PASS; all 8 required literal strings present in `CONTRIBUTING.md`.
- Re-ran Task 2's acceptance criteria — subsection sits inside `## What .planning/ is` (lines 196 < 206 < 246), the documenting commit is one hunk, and the commit subject matches exactly.
