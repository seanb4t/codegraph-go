---
phase: 02-phase-close-fragment-capability
plan: 04
subsystem: infra
tags: [gsd-core, capability, changie, skills, gitignore, config]

# Dependency graph
requires:
  - phase: 02-phase-close-fragment-capability
    provides: "gsd-capability-changie tag v0.1.0, published private, and codegraph-go's draft milestone PR #88 (02-01, 02-02, 02-03)"
provides:
  - "changie capability installed at project scope in codegraph-go from the pinned HTTPS tag, with both federated config keys set through config-set"
  - "changie capability installed at global scope on the maintainer's machine and materialized so Claude Code resolves /gsd-changie-fragments"
affects: [02-05-contributing-docs, 02-06-pr-optional-upgrade, phase-03-close]

# Actuals (#2632)
actuals:
  tokens: 100
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "gsd-core capability install/materialize verbs only — never a hand copy into ~/.claude/skills or ~/.gsd"
    - "tool-owned .planning/config.json values written only through config-set, never hand-edited"

key-files:
  created: []
  modified:
    - .gitignore
    - .planning/config.json

key-decisions:
  - "Task 2 (blocking-human checkpoint): maintainer answered global-install — install v0.1.0 at global scope too, then materialize for Claude Code, because gsd-core 1.14.0 only reads a third-party skill's SKILL.md from the global capability root (install-profiles.cjs:733)."
  - "PR field optionality (D-13) and the global-install answer (D-14) were recorded in 02-CONTEXT.md by the maintainer-facing session before this continuation started (commit b79a7772); this plan executed the global-install branch exactly as instructed and made no changes related to D-13's v0.1.1 upgrade scope, which belongs to 02-06."

patterns-established:
  - "Materialization proof pattern: capture the global skills listing before and after with `comm -3`, require exactly one added entry and zero removed entries, byte-compare the materialized SKILL.md against `git show <tag>:...`, and check the `.gsd-capability-skill` marker names the capability id."

requirements-completed: [CAP-04]

coverage:
  - id: D1
    description: "changie capability v0.1.0 installed at project scope in codegraph-go from the pinned HTTPS tag; capability.json byte-identical to the tag"
    requirement: "CAP-04"
    verification:
      - kind: other
        ref: "cmp against `git -C $CAP show v0.1.0:capability.json`"
        status: pass
      - kind: other
        ref: "git check-ignore -v .gsd-capabilities.json / .gsd/capabilities/changie/capability.json (each attributed to its own .gitignore line)"
        status: pass
      - kind: other
        ref: "git ls-files -- .gsd .gsd-capabilities.json (empty)"
        status: pass
    human_judgment: false
  - id: D2
    description: "workflow.changie_fragments=true and workflow.changie_command=\"task changie --\" written via config-set, and the install commit touches only .gitignore + .planning/config.json"
    requirement: "CAP-04"
    verification:
      - kind: other
        ref: "gsd-tools query config-get workflow.changie_fragments --raw / workflow.changie_command --raw"
        status: pass
      - kind: other
        ref: "git show --name-only --format= 07705d56 (exactly .gitignore, .planning/config.json)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Live read-only preflight in codegraph-go resolves task changie --, the five host kinds, and the open draft PR #88, tree byte-unchanged"
    requirement: "CAP-04"
    verification:
      - kind: other
        ref: "bash .gsd/capabilities/changie/skills/changie-fragments/scripts/write-fragments.sh --phase 2 --list"
        status: pass
      - kind: other
        ref: "git status --porcelain --untracked-files=all before/after --list (identical, both empty)"
        status: pass
    human_judgment: false
  - id: D4
    description: "gsd-changie-fragments is dispatchable by Claude Code on the maintainer's machine via a global-scope install and materialization, with nothing else in the global skill set changed"
    requirement: "CAP-04"
    verification:
      - kind: other
        ref: "test -f ~/.claude/skills/gsd-changie-fragments/SKILL.md + cmp against git show v0.1.0:skills/changie-fragments/SKILL.md"
        status: pass
      - kind: other
        ref: "comm -3 skills.before skills.after (exactly one added: gsd-changie-fragments, zero removed)"
        status: pass
      - kind: other
        ref: "gsd-changie-fragments now appears in this session's own Skill tool listing"
        status: pass
    human_judgment: false

# Metrics
duration: ~35min (includes a blocking-human checkpoint wait for Task 2)
completed: 2026-09-26
status: complete
---

# Phase 2 Plan 4: Install and Materialize the Changie Capability Summary

**Installed gsd-capability-changie v0.1.0 at project scope in codegraph-go (config keys set via config-set, install output fully gitignored), then — per the maintainer's `global-install` answer at the Task 2 checkpoint — installed it at global scope too and materialized `gsd-changie-fragments` so Claude Code can dispatch it on this machine, with the global skill set changing by exactly one directory.**

## Performance

- **Duration:** ~35 min elapsed (2026-09-26T09:30 - 09:46 EDT for Task 1, then a blocking-human checkpoint wait for Task 2, then Task 3 and this SUMMARY at ~10:05 EDT)
- **Tasks:** 3/3 completed (Task 1 auto, Task 2 checkpoint:decision resolved by the maintainer, Task 3 auto)
- **Files modified (tracked):** 2 (`.gitignore`, `.planning/config.json`)

## Accomplishments

- **Task 1 — project-scope install and configuration.** `gsd-tools capability install https://github.com/seanb4t/gsd-capability-changie.git#v0.1.0 --scope project` exited 0. `.gsd/capabilities/changie/capability.json` is byte-identical to `git -C $CAP show v0.1.0:capability.json` (verified via `cmp`, re-confirmed in this continuation). `.gitignore` gained the `.gsd-capabilities.json` line directly after the existing `.gsd/` line (line 46), with a comment explaining it lands at the project root, not inside `.gsd/`. Both federated keys were set through `gsd-tools query config-set` — `workflow.changie_fragments=true`, `workflow.changie_command="task changie --"` — and confirmed with `config-get` (`true`, `task changie --`). The install commit `07705d56` touches exactly `.gitignore` and `.planning/config.json`; `git status --porcelain --untracked-files=all -- . ':(exclude).planning/phases'` is empty.
- **Task 2 — maintainer checkpoint resolved.** The blocking-human decision was answered `global-install`, recorded as D-14 in `02-CONTEXT.md` (commit `b79a7772`, made before this continuation started).
- **Task 3 — global install and materialization.** With `GIT_TERMINAL_PROMPT=0`, `gsd-tools capability install https://github.com/seanb4t/gsd-capability-changie.git#v0.1.0 --scope global` exited 0 (disclosure: one instruction surface, the `changie-fragments` skill, staged unverified — no pinned hash yet, consistent with a first install of this capability). `gsd-tools capability set changie --enable --runtime claude --scope global` then materialized `~/.claude/skills/gsd-changie-fragments/SKILL.md` and `.gsd-capability-skill`. Both are proven correct below and the skill now appears in this very session's own Skill listing (`gsd-changie-fragments: Writes changie changelog fragments...`), which is live proof of dispatchability, not just a file-existence check.

## Task Commits

Each task was committed atomically:

1. **Task 1: Install v0.1.0 at project scope, set both keys through config-set, ignore the ledger, and prove the live preflight finds the draft PR** — `07705d56` (chore) — committed by the prior executor instance before this continuation started.
2. **Task 2: Maintainer decision (checkpoint)** — no code commit; the answer (`global-install`) was recorded in `02-CONTEXT.md` as D-14 by the maintainer-facing session, commit `b79a7772`.
3. **Task 3: Global install and materialization** — mutates only the maintainer's global `~/.gsd` and `~/.claude/skills` state, which is outside codegraph-go's git history by design (threat model T-02-16/T-02-SC). No codegraph-go file changed: `git status --porcelain -- .gitignore .planning/config.json` is empty after materialization. Recorded here rather than as a separate codegraph-go commit.

**Plan metadata:** this SUMMARY commit (see below).

## Files Created/Modified

- `.gitignore` — added the `.gsd-capabilities.json` ignore line (with explanatory comment) directly after the existing `.gsd/` line.
- `.planning/config.json` — `workflow.changie_fragments: true`, `workflow.changie_command: "task changie --"` (via `config-set`, never hand-edited).

## Where files land (D-01)

| Scope | Path | Contents |
|---|---|---|
| Project (codegraph-go) | `.gsd/capabilities/changie/` | full checkout: `.git/`, `LICENSE`, `README.md`, `capability.json`, `skills/changie-fragments/{SKILL.md,scripts/write-fragments.sh}`, `test/{run.sh,fixtures/...}` — gitignored via the `.gsd/` line (line 46) |
| Project (codegraph-go) | `.gsd-capabilities.json` | install ledger; one entry for `changie`, `files: [".gsd/capabilities/changie"]` — gitignored via its own new line (line 51) |
| Global (maintainer's machine) | `~/.gsd/capabilities/changie/` | identical shape to the project checkout (verified: same file list) |
| Global (maintainer's machine) | `~/.gsd-capabilities.json` | global install ledger; same shape as the project ledger |
| Global (maintainer's machine) | `${CLAUDE_CONFIG_DIR:-~/.claude}/skills/gsd-changie-fragments/SKILL.md` | byte-identical to `git -C $CAP show v0.1.0:skills/changie-fragments/SKILL.md` |
| Global (maintainer's machine) | `${CLAUDE_CONFIG_DIR:-~/.claude}/skills/gsd-changie-fragments/.gsd-capability-skill` | contains `changie` |

`CLAUDE_CONFIG_DIR` and `GSD_HOME` were both unset for this run, so the real defaults (`~/.claude`, `~/.gsd`) were used — no redirect, per the plan's prohibition.

## Verification Records

### Task 1 — install output
```
{
  "status": "installed",
  "id": "changie",
  "version": "0.1.0",
  "scope": "project"
}
```
(reconstructed field summary; full JSON matched the Task 3 global-install shape shown below)

### Task 1 — config-get outputs
```
$ gsd-tools query config-get workflow.changie_fragments --raw
true
$ gsd-tools query config-get workflow.changie_command --raw
task changie --
```

### Task 1 — render-hooks (project scope only, before Task 3)
One `activeHooks` entry with `capId: "changie"` at `verify:post` — confirmed both before and after the global install/materialization in Task 3 (see below); the count did **not** become 2, so D-07 idempotency was not exercised by this plan.

### Task 1 — full `--list` transcript
```
changie-fragments: phase 02 /Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/02-phase-close-fragment-capability
changie-fragments: range main..HEAD
changie-fragments: kinds Breaking,Features,Fixes,Performance,Dependencies
changie-fragments: pr 88
changie-fragments: pending 02-01 /Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/02-phase-close-fragment-capability/02-01-SUMMARY.md
changie-fragments: pending 02-02 /Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/02-phase-close-fragment-capability/02-02-SUMMARY.md
changie-fragments: pending 02-03 /Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/02-phase-close-fragment-capability/02-03-SUMMARY.md
```
No `skip:` line. `pr 88` matches the draft milestone PR opened in 02-03. `git status --porcelain --untracked-files=all` was identical (empty) before and after this run.

### Task 1 — before/after status comparison
Both empty (`git status --porcelain --untracked-files=all`), confirming `--list` is read-only.

## Skill materialization

**Answer:** `global-install` (D-14, recorded in `02-CONTEXT.md` commit `b79a7772`).

**Global install output:**
```json
{
  "status": "installed",
  "id": "changie",
  "version": "0.1.0",
  "scope": "global",
  "disclosure": [
    "This capability ships no executable surfaces, but contributes agent instructions:",
    "  instruction surfaces (1): installed into your agent's instruction context",
    "    - skill: changie-fragments",
    "        these bodies are installed verbatim and are NOT content-scanned",
    "  content: NO PINNED HASH — staged unverified (a computed sha512 is recorded in the ledger at install)"
  ]
}
```

**Materialization output:**
```json
{
  "id": "changie",
  "enabled": true,
  "surfaced": true,
  "installed": true
}
```

**Before/after global skills listing (`comm -3 skills.before skills.after`):**
```
	gsd-changie-fragments
```
Exactly one line, in the added column (single leading tab = column 2 = added-only), zero removed entries. `skills.before` had 140 entries (no `gsd-changie-fragments`); `skills.after` has 141.

**cmp result:** `git -C $CAP show v0.1.0:skills/changie-fragments/SKILL.md` piped to a temp file, then `cmp` against `~/.claude/skills/gsd-changie-fragments/SKILL.md` — exit 0, byte-identical.

**Marker content:** `~/.claude/skills/gsd-changie-fragments/.gsd-capability-skill` contains `changie`.

**Global paths that now exist:**
- `~/.gsd/capabilities/changie/` (full checkout, same file list as the project checkout)
- `~/.gsd-capabilities.json` (global ledger, one `changie` entry)
- `~/.claude/skills/gsd-changie-fragments/{SKILL.md,.gsd-capability-skill}`

**Both render-hooks counts:** 1 changie hook before the global install, 1 changie hook after (`gsd-tools loop render-hooks verify:post --raw` run from codegraph-go's root both times — the render-hooks view is per-project and reads only the project-scope registration; the global install does not add a second entry to codegraph-go's own hook list). `gsd-tools capability list --json` does show **two** `changie` entries system-wide — one `scope: "project"`, one `scope: "global"` — confirming both installs coexist without error.

**codegraph-go tracked-file check:** `git status --porcelain -- .gitignore .planning/config.json` is empty after materialization — no tracked codegraph-go file was touched by the global install/materialization.

**Live dispatchability proof:** this very executor session's Skill listing now includes `gsd-changie-fragments: Writes changie changelog fragments for a closed GSD phase. Dispatched automatically at verify:post when workflow.changie_fragments is true (the default). Can also be run by hand as /gsd-changie-fragments <phase> [--pr <n>] [--repo <dir>].` — direct evidence the skill is dispatchable by Claude Code on this machine, beyond the file-existence and byte-comparison checks.

## Decisions Made

- Maintainer chose `global-install` at the Task 2 blocking-human checkpoint (recorded as D-14). This was the plan's recommended option: it is the only path gsd-core 1.14.0 supports for materializing a third-party Claude skill (project-scope installs are invisible to the skill materializer, which reads only the global capability root — `install-profiles.cjs:733`).
- No changes were made related to D-13 (making the `PR` custom field optional) or the v0.1.1 capability upgrade it specifies — that scope belongs to plan 02-06, which runs before 02-05. This plan verified only that the currently-installed v0.1.0's live `--list` still resolves `pr 88` from the required-`PR` era; D-13's superseding of that requirement is noted, not acted on, here.

## Deviations from Plan

None — plan executed exactly as written. The `global-install` branch of Task 3 was followed literally, substituting the HTTPS install spec (D-11 amendment) as instructed by the continuation prompt.

## Issues Encountered

None.

## Threat Flags

None — this plan's threat register (T-02-16, T-02-SC, T-02-17, T-02-18, T-02-19) already anticipated the global materialization surface, and all four `mitigate`-disposition items were verified directly in the steps above (blocking-human checkpoint gated the global mutation; both installs pinned to the same tag and byte-compared; `.gsd/` and `.gsd-capabilities.json` each ignored by their own line with nothing else in the install commit; config values written only through `config-set`).

## User Setup Required

None — no external service configuration required beyond the one-time per-machine capability install this plan performed. (CONTRIBUTING.md documentation of this step is 02-05's scope, per D-12/D-14.)

## Next Phase Readiness

CAP-04's install half (project scope, config keys, gitignore) and the dispatchability half (CAP-05 precondition: the skill resolves on this machine) are both done. Ready for 02-05 (CONTRIBUTING.md documentation) — 02-06 (PR-optional v0.1.1 upgrade, D-13) runs first, per the maintainer's scope ordering. Phase 3's close is where CAP-05 (the capability's first real firing) gets proven.

---
*Phase: 02-phase-close-fragment-capability*
*Completed: 2026-09-26*

## Self-Check: PASSED

- FOUND: `.gitignore`, `.planning/config.json`, this SUMMARY, global `SKILL.md`, global `.gsd-capability-skill` marker, global `~/.gsd-capabilities.json` ledger.
- FOUND commits: `07705d56` (Task 1), `b79a7772` (D-13/D-14 decision record) in `git log --oneline --all`.
- Re-ran plan-level acceptance criteria: capability.json byte-identity (`cmp`) PASS; `git ls-files -- .gsd .gsd-capabilities.json` empty PASS; `--list` transcript kinds+pr line PASS; install commit subject PASS; Task 3 global SKILL.md byte-identity + marker PASS.
- Re-ran plan `<verification>`: install/ignore/config/commit proofs (Task 1 verify block) PASS; live `--list` resolution PASS; materialized skill byte-identical, sole global-skills addition, and no tracked codegraph-go file touched (Task 3 verify block) PASS.
