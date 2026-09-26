# 02-MUTATION-LOG — Phase-Close Fragment Capability (D-13 PR-optional)

**Phase:** 02-phase-close-fragment-capability
**Date:** 2026-09-26
**Scope:** RED-first evidence and mutation-family demonstrations for the re-pointed
codegraph-go guards that D-13 (the changie `PR` custom field becomes optional)
touches: `internal/upgrade/changie_shape_test.go#TestChangieConfigShape` (the
`custom[0]` key set) and `check:changie`'s legs 5, 7 and 12 in `Taskfile.yml`.

## Pre-mutation cleanliness gate — the convention this log follows

Before every tracked-file mutation, and before every revert, `git diff --quiet --
<file>` (or, for a directory-scoped check, `git status --porcelain -- <dir>`) is
asserted to exit empty/clean. This proves no pre-existing tracked edit was
overwritten by the mutation, and no revert was a destructive blind checkout of
someone else's in-flight work. Every family entry below records this gate's
result at the point it was checked. This log follows `01-MUTATION-LOG.md`'s
convention exactly (pre-mutation gate, applied confirmation, verbatim RED
transcript, byte-clean revert, GREEN re-run).

## RED-first transcripts (Task 1)

### Go RED — `TestChangieConfigShape`, committed before `.changie.yaml` changed

Command: `GOWORK=off go test ./internal/upgrade/ -run '^TestChangieConfigShape$' -count=1 -v`,
run immediately after the test-file-only commit `test(02-06): expect the changie
PR field to be optional (D-13)` and before `.changie.yaml` gained `optional: true`.

```
--- FAIL: TestChangieConfigShape (0.00s)
changie_shape_test.go:260: ../../.changie.yaml: custom[0] key set = [key minInt type], want exactly [key minInt optional type] (an absent optional key, or any other key, must fail — 02-CONTEXT D-13, which supersedes D-11)
--- FAIL: TestChangieConfigShape (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/upgrade	0.186s
FAIL
```

This is a genuine assertion failure naming the `custom[0]` key set (the real,
pre-change `.changie.yaml` has no `optional` key yet) — not a build failure or a
harness crash. Per the file's own header comment, this test is never gated on
`gsd-tools check tdd-red-evidence` (it parses TAP only and always reports
`INVALID_RED` for `go test`); the RED evidence is this commit's position on
`main..HEAD` plus this pasted `--- FAIL` transcript.

### `check:changie` leg 5 RED — re-pointed leg against the real, pre-change `.changie.yaml`

Command: `task check:changie`, run after `Taskfile.yml`'s leg 5 (and legs 6-9's
count expectations) were re-pointed to expect acceptance, but before
`.changie.yaml` gained `optional: true` — so the real host config still
refuses a no-PR fragment.

```
check:changie: found 14 seeded version files under .changes/ (floor 14)
check:changie: [1/11] changie latest = v0.14.0 (newest CHANGELOG.md heading v0.14.0; baseline v0.14.0) — ok
check:changie: [2/11] scratch unreleased/ holds 0 fragments before next auto
check:changie: [2/11] changie next auto (empty unreleased/) refused: no unreleased changes found for automatic bumping — ok
check:changie: [3/11] changie merge --dry-run byte-identical to CHANGELOG.md (dry run wrote nothing) — ok
check:changie: [4/11] changie new -k Fixes writes exactly one fragment with kind/body set — ok
::error::check:changie: [5/11] changie new with no -m PR= exited 1 (expected acceptance, D-13): Error: custom missing and prompt is disabled: custom key 'PR'
custom missing and prompt is disabled: custom key 'PR'
task: Failed to run task "check:changie": exit status 1
```

Exit code 201 (task's wrapping of the guard's `exit 1`). This is a genuine
assertion failure on the leg's newly-planned behavior (the fragment must now
be *accepted* without `-m PR=`), not a harness crash — the real, unedited
`.changie.yaml` still declares `PR` without `optional: true`, so changie's own
refusal fires exactly as it did before this plan.

### GREEN dry-run confirmation — real host config accepts a no-PR fragment

Command: `CI=true task changie -- new --dry-run -k Fixes -b "no-PR tracer probe"`,
run against the real, now-edited `.changie.yaml` (after `optional: true` was
added), from the plan's own `<verify>` block.

```
kind: Fixes
body: no-PR tracer probe
time: 2026-09-26T11:01:29.764608-04:00
```

No `PR:` line, no `custom:` block. `git status --porcelain -- .changes` was
empty afterward — the `--dry-run` flag wrote nothing.

<!-- gsd:write-continue -->
