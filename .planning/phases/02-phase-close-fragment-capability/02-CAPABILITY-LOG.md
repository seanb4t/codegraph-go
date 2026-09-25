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
