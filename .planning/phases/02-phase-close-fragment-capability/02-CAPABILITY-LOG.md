# 02-CAPABILITY-LOG — Phase-Close Fragment Capability

**Phase:** 02-phase-close-fragment-capability
**Date:** 2026-09-25
**Scope:** RED/GREEN evidence for `/Volumes/Code/github.com/seanb4t/gsd-capability-changie`
(`$CAP` below), a repository whose commit trail is outside this roadmap (D-09). Every test
run, RED transcript and GREEN transcript for 02-01 lives here, in codegraph-go, because
`$CAP` is a separate git history.

## Task 1 — end-to-end tracer (install, dispatch gate, write path)

### `$CAP` commit trail after Task 1

```
b1291df test(02-01): add failing install, dispatch-gate and write-path proofs
464510b feat(02-01): changie capability manifest, skill and fragment writer
```

(`git -C $CAP log --reverse --format='%h %s'`.)

### RED transcript

Command: `CHANGIE_BIN=<pinned changie build path> /bin/bash $CAP/test/run.sh`, run before
`capability.json`, `SKILL.md` or `write-fragments.sh` existed in `$CAP` (only the fixtures
and `test/run.sh` itself were committed).

```
run.sh: changie /Users/sean/Library/Caches/go-build/2f/2fe2f32e74fb59c69ef1f4f6661faa9bf90386be2fe91a042a6356f109956249-d/changie (changie version vdev)
run.sh: found 2 fixture SUMMARY files (floor 2)
run.sh: init.phase-op 1 --pick phase_dir = /private/var/folders/_b/3hyf5qvs62q0wh2vyh856z580000gn/T/tmp.O4SJotjAMf/work/.planning/phases/01-fixture
ok [ordering] config-set workflow.changie_command is rejected before install
::error::run.sh: [install] capability install exited 1: Error: capability install blocked: Cannot read capability.json from local path: /Volumes/Code/github.com/seanb4t/gsd-capability-changie/capability.json
```

`[ordering]` passes (a genuine, already-correct host behavior); `[install]` fails on an
assertion about planned behavior — the capability install call rejects a manifest that does
not exist yet — not a harness crash or a missing tool. This is intentional RED.

### GREEN transcript

Command: the same invocation, run after `capability.json`, `skills/changie-fragments/SKILL.md`
and `skills/changie-fragments/scripts/write-fragments.sh` were added and committed.

```
run.sh: changie /Users/sean/Library/Caches/go-build/2f/2fe2f32e74fb59c69ef1f4f6661faa9bf90386be2fe91a042a6356f109956249-d/changie (changie version vdev)
run.sh: found 2 fixture SUMMARY files (floor 2)
run.sh: init.phase-op 1 --pick phase_dir = /private/var/folders/_b/3hyf5qvs62q0wh2vyh856z580000gn/T/tmp.Row15czY8p/work/.planning/phases/01-fixture
ok [ordering] config-set workflow.changie_command is rejected before install
ok [install] capability install stages capability.json, SKILL.md, write-fragments.sh and the ledger
ok [manifest] installed capability.json has id/role/engines/runtimeCompat/steps/config exactly as required
run.sh: [dispatch-default] matching changie steps (key absent) = 1
ok [dispatch-default] render-hooks verify:post lists exactly one changie step with the key absent
run.sh: [dispatch-false] matching changie steps (key false) = 0
ok [dispatch-false] render-hooks verify:post omits the changie step when the key is false
run.sh: [dispatch-true] matching changie steps (key true) = 1
ok [dispatch-true] render-hooks verify:post lists exactly one changie step with the key true
ok [write] --list prints pr/pending/kinds and --write commits exactly 3 fragments with correct trailers
run.sh: 7 of 7 legs passed against a scratch project
```

### changie binary and version used

`CHANGIE_BIN="$(cd /Volumes/Code/github.com/seanb4t/codegraph-go && GOWORK=off go tool -modfile=go.tool-changie.mod -n changie)"`
resolved to `/Users/sean/Library/Caches/go-build/2f/2fe2f32e74fb59c69ef1f4f6661faa9bf90386be2fe91a042a6356f109956249-d/changie`,
reporting `changie version vdev` — the same checksum-pinned build `go.tool-changie.mod`
produces for this repository (Phase 1 D-04/D-05). No network fetch occurred; `go tool -n`
only resolves the already-built cache path.

`shellcheck` (0.11.0) is clean on `write-fragments.sh`, `test/run.sh`, `test/fixtures/bin/gh`
and `test/fixtures/bin/fake-task`.

## Task 2 — every skip case, entry validation, idempotency, rollback and lock

### `$CAP` commit trail after Task 2

```
b1291df test(02-01): add failing install, dispatch-gate and write-path proofs
464510b feat(02-01): changie capability manifest, skill and fragment writer
5c00c09 test(02-01): add failing skip-case, idempotency and guard proofs
8a6de32 feat(02-01): skip cases, idempotency, rollback and lock in the fragment writer
```

(`git -C $CAP log --reverse --format='%h %s'`.)

### RED transcript

Command: `CHANGIE_BIN=<pinned changie build path> /bin/bash $CAP/test/run.sh`, run after
extending `test/run.sh` with the 22 new legs but before `write-fragments.sh` implemented any
of the skip ids, the lock, entry validation or rollback.

```
run.sh: changie /Users/sean/Library/Caches/go-build/2f/2fe2f32e74fb59c69ef1f4f6661faa9bf90386be2fe91a042a6356f109956249-d/changie (changie version vdev)
run.sh: found 2 fixture SUMMARY files (floor 2)
run.sh: init.phase-op 1 --pick phase_dir = /private/var/folders/_b/3hyf5qvs62q0wh2vyh856z580000gn/T/tmp.FAqymL8g3d/work/.planning/phases/01-fixture
ok [ordering] config-set workflow.changie_command is rejected before install
ok [install] capability install stages capability.json, SKILL.md, write-fragments.sh and the ledger
ok [manifest] installed capability.json has id/role/engines/runtimeCompat/steps/config exactly as required
run.sh: [dispatch-default] matching changie steps (key absent) = 1
ok [dispatch-default] render-hooks verify:post lists exactly one changie step with the key absent
run.sh: [dispatch-false] matching changie steps (key false) = 0
ok [dispatch-false] render-hooks verify:post omits the changie step when the key is false
run.sh: [dispatch-true] matching changie steps (key true) = 1
ok [dispatch-true] render-hooks verify:post lists exactly one changie step with the key true
::error::run.sh: [skip:gsd-tools-missing] expected exit 0, got 127:
```

`[skip:gsd-tools-missing]` fails with exit 127 because the Task-1 script blindly execs an
unresolved `GSD_TOOLS` path instead of skipping — a genuine assertion failure on planned
behavior that did not exist yet, not a harness crash.

### GREEN transcript

Command: the same invocation, run after `write-fragments.sh` implemented the full preflight
chain, the lock, entry validation and rollback, and after fixing two pre-existing bugs the new
idempotency legs caught (unresolved `origin/HEAD` literal text producing a false `HEAD..HEAD`
range; two `%(trailers:...)` placeholders interleaving onto separate lines instead of one line
per commit).

```
run.sh: changie /Users/sean/Library/Caches/go-build/2f/2fe2f32e74fb59c69ef1f4f6661faa9bf90386be2fe91a042a6356f109956249-d/changie (changie version vdev)
run.sh: found 2 fixture SUMMARY files (floor 2)
run.sh: init.phase-op 1 --pick phase_dir = /private/var/folders/_b/3hyf5qvs62q0wh2vyh856z580000gn/T/tmp.6DHklVk1x1/work/.planning/phases/01-fixture
ok [ordering] config-set workflow.changie_command is rejected before install
ok [install] capability install stages capability.json, SKILL.md, write-fragments.sh and the ledger
ok [manifest] installed capability.json has id/role/engines/runtimeCompat/steps/config exactly as required
run.sh: [dispatch-default] matching changie steps (key absent) = 1
ok [dispatch-default] render-hooks verify:post lists exactly one changie step with the key absent
run.sh: [dispatch-false] matching changie steps (key false) = 0
ok [dispatch-false] render-hooks verify:post omits the changie step when the key is false
run.sh: [dispatch-true] matching changie steps (key true) = 1
ok [dispatch-true] render-hooks verify:post lists exactly one changie step with the key true
ok [skip:gsd-tools-missing] GSD_TOOLS pointing at a nonexistent path skips cleanly
ok [skip:disabled] workflow.changie_fragments=false skips cleanly
ok [skip:changie-unavailable] an unresolvable configured changie command skips cleanly, naming the command
ok [skip:changie-empty] an empty workflow.changie_command is rejected or skips as changie-unavailable
ok [skip:no-changie-config] a missing .changie.yaml skips cleanly
ok [skip:phase-not-found] an unknown phase number skips cleanly
ok [skip:no-summaries] a phase directory with no SUMMARY files skips cleanly
ok [skip:gh-missing] an unresolvable gh executable skips cleanly
ok [skip:pr-lookup-failed] a failing gh lookup skips cleanly, detail carries gh's stderr
ok [skip:no-open-pr] no open PR for the branch skips cleanly, naming the branch
ok [pr-override] an explicit --pr wins over the gh lookup, which is not consulted
ok [bad-pr] --pr 0, --pr abc and --pr -5 all exit 2 with error: bad-pr
ok [write] --list prints pr/pending/kinds and --write commits exactly 3 fragments with correct trailers
ok [idempotent] re-running --list/--write for already-covered summaries yields skip: already-recorded, HEAD and fragment count unchanged
ok [gap-closure] a SUMMARY added after the fragment commit is the only pending id
ok [skip:no-user-visible-change] --write with an empty entries file skips cleanly, HEAD unchanged
ok [undeclared-kind] an undeclared kind exits 2 with error: undeclared-kind, listing the declared kinds, no file written
ok [malformed-entry] a line with no TAB exits 2 with error: malformed-entry, no file written
ok [rollback] a changie failure on the second entry rolls back the first, releases the lock, exits 1 with error: changie-failed
ok [skip:locked] a pre-held lock directory skips cleanly
ok [argv-literal] a body with $(...), backticks, quotes, ; and * reaches the fragment literally and nothing runs
ok [commit-scope] a pre-existing untracked fragment and an unstaged tracked edit are left exactly as they were
ok [no-side-effects] $CAP_ROOT and the global Claude skills listing are unchanged
run.sh: 29 of 29 legs passed against a scratch project
```

`shellcheck` (0.11.0) remains clean on all four executables after the Task 2 changes.
`git -C $CAP status --porcelain` is empty after the Task 2 commit.

## Judgment rehearsal (SKILL.md followed by the executor, pre-publication)

**Purpose (D-06, T-02-08):** the real Skill-tool dispatch comes in 02-05, after
materialization. This rehearsal follows `$CAP/skills/changie-fragments/SKILL.md` exactly as
written, pre-publication, so a wording defect is caught while the tag is still free to move.
The executor acted as the model half here, using no knowledge SKILL.md itself did not supply.

**SKILL.md commit it ran against:** `715cc9a docs(02-02): add README and MIT license` (SKILL.md
itself was last touched in `8a6de32`, Task 2 of 02-01; unchanged since).

**Fixture project setup** (mirrors `test/run.sh`'s own fixture build, in a fresh `mktemp -d`
scratch directory `$R`):

```
$ cd "$R"
$ git init -b main -q
$ git config user.email "test@example.com"
$ git config user.name "test"
$ git config commit.gpgsign false
$ mkdir -p .changes/unreleased
$ cp $CAP/test/fixtures/changie.yaml .changie.yaml
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
$ cp $CAP/test/fixtures/phase/01-01-SUMMARY.md .planning/phases/01-fixture/01-01-SUMMARY.md
$ cp $CAP/test/fixtures/phase/01-02-SUMMARY.md .planning/phases/01-fixture/01-02-SUMMARY.md
$ git add -A && git commit -q -m "baseline"
$ git checkout -b gsd/fixture-milestone -q
```

**Capability install and config, at project scope:**

```
$ gsd-tools capability install $CAP --scope project --raw
{
  "status": "installed",
  "id": "changie",
  "version": "0.1.0",
  "scope": "project",
  "disclosure": [...]
}
$ CHB="$(cd /Volumes/Code/github.com/seanb4t/codegraph-go && GOWORK=off go tool -modfile=go.tool-changie.mod -n changie)"
# CHB = /Users/sean/Library/Caches/go-build/2f/2fe2f32e74fb59c69ef1f4f6661faa9bf90386be2fe91a042a6356f109956249-d/changie
$ gsd-tools query config-set workflow.changie_command "$CHB" --raw
workflow.changie_command=<CHB path>
```

**Skill dispatch, followed exactly as SKILL.md steps 1-7, arguments `1 --pr 4242 --repo "$R"`:**

Step 2 (locate the script) resolved to
`$R/.gsd/capabilities/changie/skills/changie-fragments/scripts/write-fragments.sh`, which
existed and was executable.

Step 3 `--list` output:

```
changie-fragments: phase 01 <R>/.planning/phases/01-fixture
changie-fragments: range main..HEAD
changie-fragments: kinds Breaking,Features,Fixes,Performance,Dependencies
changie-fragments: pr 4242
changie-fragments: pending 01-01 <R>/.planning/phases/01-fixture/01-01-SUMMARY.md
changie-fragments: pending 01-02 <R>/.planning/phases/01-fixture/01-02-SUMMARY.md
```

No `skip:` line — both SUMMARYs were read (step 4).

Step 5 judgment: `01-01` (WIDGET-03, "Added a `--json` flag to `widget list`...") is
user-visible — a CLI flag a user can observe. Kind `Features` (from the `kinds` line). `01-02`
("Added a shellcheck CI job, refactored an internal test helper, and updated planning docs —
no user-visible change") is CI/test/planning-only — not user-visible, no entry written for it.

Step 6 entries file (one line, `<Kind><TAB><Body>`, written with the file tool, no shell
string):

```
Features	Added a --json flag to widget list that prints one JSON object per widget, one per line.
```

Step 7 write, `--summaries 01-01,01-02` (both pending ids, per SKILL.md's instruction to pass
"the exact ids from the pending lines"):

```
$ bash <script> --phase 1 --pr 4242 --repo "$R" --write --summaries 01-01,01-02 --entries <file>
changie-fragments: wrote 1 fragment(s) in bf67b90: .changes/unreleased/Features-20260925-203052.882197000.yaml
```

Script's last line, reported verbatim: `changie-fragments: wrote 1 fragment(s) in bf67b90:
.changes/unreleased/Features-20260925-203052.882197000.yaml`

**Trailer read** (own-line transcript):

```
$ git -C "$R" log -1 --format='%(trailers:key=Changie-Summaries,valueonly)'
01-01,01-02
```

**Assertions and results:**

| Assertion | Result |
|---|---|
| `git -C "$R" log -1 --format=%s` is `docs(01): add changelog fragments` | PASS — `docs(01): add changelog fragments` |
| Changie-Summaries trailer is `01-01,01-02` | PASS — `01-01,01-02` |
| `git -C "$R" show --name-only --format= HEAD` lists exactly one file | PASS — `.changes/unreleased/Features-20260925-203052.882197000.yaml` |
| That file contains `kind: Features` and `PR: "4242"` | PASS |
| Its `body:` line contains `--json` | PASS |
| `rg '^body:' <file> \| rg -o 'WIDGET-\|\bD-[0-9]+\|\b0[0-9]-0[0-9]\b' \| wc -l` prints 0 | PASS — `0` |

Fragment file content:

```yaml
kind: Features
body: Added a --json flag to widget list that prints one JSON object per widget, one per line.
time: 2026-09-25T20:30:52.882197-04:00
custom:
    PR: "4242"
```

**Outcome, first pass:** all six rehearsal assertions above passed. But this task's own
acceptance criteria include a separate, broader check — `rg -F -o '.claude/' SKILL.md | wc -l`
prints 0 — and that one FAILED: SKILL.md's own Rules section carried the line "Keep any
`` `.claude/` `` path out of this file's body — the script resolves its own location without
one," which names the very substring it forbids. This is SKILL.md's wording defect, per the
plan's own framing (the rule *about* not referencing a `.claude/` path violated itself).

**Fix applied in `$CAP`:** reworded the line to describe the same restriction without the
literal substring:

```diff
-- Keep any `.claude/` path out of this file's body — the script resolves
-  its own location without one.
+- Keep any host-specific installed-skill directory path out of this file's
+  body — the script resolves its own location without one.
```

Committed as `fix(02-02): reword the no-.claude/-path rule to avoid the literal substring it
forbids` (`5700e60`). `rg -F -o '.claude/' SKILL.md | wc -l` now prints `0`.

**Re-run to 29 of 29** (`CHANGIE_BIN=<pinned changie build path> bash $CAP/test/run.sh`, HEAD
`5700e60`):

```
...
ok [no-side-effects] $CAP_ROOT and the global Claude skills listing are unchanged
run.sh: 29 of 29 legs passed against a scratch project
```

**Repeat rehearsal, fresh scratch project `$R2`** (identical setup to `$R`, install at HEAD
`5700e60`), same arguments `1 --pr 4242 --repo "$R2"`:

```
$ bash <script> --phase 1 --pr 4242 --repo "$R2" --write --summaries 01-01,01-02 --entries <file>
changie-fragments: wrote 1 fragment(s) in dc2b4ed: .changes/unreleased/Features-20260925-204931.456062000.yaml
```

| Assertion | Result |
|---|---|
| `git -C "$R2" log -1 --format=%s` is `docs(01): add changelog fragments` | PASS |
| Changie-Summaries trailer is `01-01,01-02` | PASS — `01-01,01-02` |
| `git -C "$R2" show --name-only --format= HEAD` lists exactly one file | PASS — `.changes/unreleased/Features-20260925-204931.456062000.yaml` |
| That file contains `kind: Features` and `PR: "4242"` | PASS |
| Its `body:` line contains `--json` | PASS |
| `rg '^body:' <file> \| rg -o 'WIDGET-\|\bD-[0-9]+\|\b0[0-9]-0[0-9]\b' \| wc -l` prints 0 | PASS — `0` |

Fragment file content (`$R2`):

```yaml
kind: Features
body: Added a --json flag to widget list that prints one JSON object per widget, one per line.
time: 2026-09-25T20:49:31.456062-04:00
custom:
    PR: "4242"
```

Both scratch projects (`$R`, `$R2`) and their capability installs were discarded (`mktemp -d`,
never committed, never touched `$CAP` or codegraph-go's own working trees) — confirmed by `git
status --porcelain` being clean in both real repositories immediately after each rehearsal run.
