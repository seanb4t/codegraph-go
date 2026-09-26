---
phase: 02-phase-close-fragment-capability
plan: 05
status: in-progress
---

# Phase 2 Plan 5: Real Skill Dispatch and Contributor Documentation Summary

_(Task 1 of 2 committed. This file is completed after Task 2.)_

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
