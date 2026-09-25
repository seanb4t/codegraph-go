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
